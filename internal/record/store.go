package record

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fristovic/snitch/internal/scrub"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store provides SQLite persistence for runs and claims.
type Store struct {
	db      *sql.DB
	dataDir string
}

// Open opens or creates the SQLite database at dataDir/snitch.db.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "snitch.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, dataDir: dataDir}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		if isCorruption(err) {
			backup := dbPath + ".corrupted." + time.Now().Format("20060102-150405")
			_ = os.Rename(dbPath, backup)
			return Open(dataDir)
		}
		return nil, err
	}
	return s, nil
}

func isCorruption(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "malformed") || strings.Contains(msg, "corrupt")
}

func (s *Store) migrate() error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(string(data)); err != nil {
			if strings.Contains(err.Error(), "duplicate column") ||
				strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying database for analytics aggregation.
func (s *Store) DB() *sql.DB {
	return s.db
}

// InsertRun inserts a new run record.
func (s *Store) InsertRun(run Run) error {
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	if run.Harness == "" {
		run.Harness = "cursor"
	}
	cmd := scrub.Scrub(run.Command)
	_, err := s.db.Exec(`
		INSERT INTO runs (id, created_at, session_id, transcript_path, project_path,
			harness, command, duration_ms, output_hash, tool_call_count, verdict, max_severity,
			claim_count, verified_claims, false_claims, device_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.CreatedAt.Format(time.RFC3339), run.SessionID, run.TranscriptPath, run.ProjectPath,
		run.Harness, cmd, run.DurationMS, run.OutputHash, run.ToolCallCount, string(run.Verdict),
		run.MaxSeverity, run.ClaimCount, run.VerifiedClaims, run.FalseClaims, run.DeviceID)
	return err
}

// UpdateRunVerdict updates verdict fields on a run.
func (s *Store) UpdateRunVerdict(runID string, verdict Verdict, maxSeverity int, claimCount, verified, falseClaims int) error {
	_, err := s.db.Exec(`
		UPDATE runs SET verdict=?, max_severity=?, claim_count=?, verified_claims=?, false_claims=?
		WHERE id=?`, string(verdict), maxSeverity, claimCount, verified, falseClaims, runID)
	return err
}

// InsertClaims inserts claim rows for a run.
func (s *Store) InsertClaims(claims []Claim) error {
	for _, c := range claims {
		ev, _ := json.Marshal(c.Evidence)
		_, err := s.db.Exec(`
			INSERT INTO claims (run_id, claim_type, source, target, claimed, actual, verified, severity, verifier, evidence)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.RunID, c.ClaimType, c.Source, scrub.Scrub(c.Target), scrub.Scrub(c.Claimed),
			scrub.Scrub(c.Actual), c.Verified, c.Severity, c.Verifier, string(ev))
		if err != nil {
			return err
		}
	}
	return nil
}

// SaveTrace persists verification trace lines for a run.
func (s *Store) SaveTrace(runID string, trace []string) error {
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE runs SET trace=? WHERE id=?`, string(data), runID)
	return err
}

// AnalyticsStats returns aggregate stats for the reporting window.
func (s *Store) AnalyticsStats(since time.Time) (total int, sevDist map[string]int, err error) {
	sevDist = map[string]int{"level0": 0, "level1": 0, "level2": 0, "level3": 0}
	rows, err := s.db.Query(`SELECT max_severity, COUNT(*) FROM runs WHERE created_at >= ? GROUP BY max_severity`,
		since.Format(time.RFC3339))
	if err != nil {
		return 0, sevDist, err
	}
	defer rows.Close()
	for rows.Next() {
		var sev, count int
		if err := rows.Scan(&sev, &count); err != nil {
			return 0, sevDist, err
		}
		total += count
		if sev >= 0 && sev <= 3 {
			sevDist[fmt.Sprintf("level%d", sev)] += count
		}
	}
	return total, sevDist, rows.Err()
}

const runSelectCols = `id, created_at, session_id, transcript_path, project_path,
		harness, command, duration_ms, output_hash, tool_call_count, verdict, max_severity,
		claim_count, verified_claims, false_claims, device_id, trace`

// GetRuns returns runs matching filter.
func (s *Store) GetRuns(filter RunFilter) ([]Run, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	q := `SELECT ` + runSelectCols + ` FROM runs WHERE 1=1`
	args := []any{}
	if filter.Verdict != "" {
		q += " AND verdict=?"
		args = append(args, filter.Verdict)
	}
	if filter.FailuresOnly {
		q += " AND verdict IN ('fail', 'warn')"
	}
	if filter.Harness != "" {
		q += " AND harness=?"
		args = append(args, filter.Harness)
	}
	if filter.ProjectPath != "" {
		q += " AND project_path=?"
		args = append(args, filter.ProjectPath)
	}
	if filter.SessionID != "" {
		q += " AND session_id=?"
		args = append(args, filter.SessionID)
	}
	if !filter.Since.IsZero() {
		q += " AND created_at >= ?"
		args = append(args, filter.Since.UTC().Format(time.RFC3339))
	}
	if filter.Search != "" {
		q += " AND (command LIKE ? OR id LIKE ?)"
		pat := "%" + filter.Search + "%"
		args = append(args, pat, pat)
	}
	q += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, filter.Offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// GetRunsByProject returns runs for a project path.
func (s *Store) GetRunsByProject(projectPath string, limit int) ([]Run, error) {
	return s.GetRuns(RunFilter{ProjectPath: projectPath, Limit: limit})
}

// CountDistinctSessions returns distinct session_id count.
func (s *Store) CountDistinctSessions() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(DISTINCT session_id) FROM runs WHERE session_id != ''`).Scan(&n)
	return n, err
}

// CountDistinctProjects returns distinct project_path count.
func (s *Store) CountDistinctProjects() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(DISTINCT project_path) FROM runs WHERE project_path != ''`).Scan(&n)
	return n, err
}

func (s *Store) GetRunByID(id string) (*Run, error) {
	if id == "" {
		return nil, nil
	}
	row := s.db.QueryRow(`SELECT `+runSelectCols+` FROM runs WHERE id=?`, id)
	r, err := scanRunRow(row)
	if err == nil {
		return &r, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	if len(id) >= 36 {
		return nil, nil
	}
	row = s.db.QueryRow(`SELECT `+runSelectCols+` FROM runs WHERE id LIKE ? ORDER BY created_at DESC LIMIT 1`, id+"%")
	r, err = scanRunRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetClaimsByRun returns claims for a run.
func (s *Store) GetClaimsByRun(runID string) ([]Claim, error) {
	rows, err := s.db.Query(`
		SELECT id, run_id, claim_type, source, target, claimed, actual, verified, severity, verifier, evidence, created_at
		FROM claims WHERE run_id=? ORDER BY id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanClaims(rows)
}

// GetClaims returns claims matching filter, joined with run metadata.
func (s *Store) GetClaims(filter ClaimFilter) ([]LieClaim, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	minSev := filter.MinSeverity
	if filter.LiesOnly && minSev < 2 {
		minSev = 2
	}
	q := `
		SELECT c.id, c.run_id, c.claim_type, c.source, c.target, c.claimed, c.actual,
			c.verified, c.severity, c.verifier, c.evidence, c.created_at,
			r.project_path, r.session_id, r.command, r.created_at, r.verdict
		FROM claims c
		JOIN runs r ON c.run_id = r.id
		WHERE 1=1`
	args := []any{}
	if minSev > 0 {
		q += " AND c.severity >= ?"
		args = append(args, minSev)
	}
	if filter.LiesOnly {
		q += " AND c.verified = -1"
	}
	if filter.ClaimType != "" {
		q += " AND c.claim_type = ?"
		args = append(args, filter.ClaimType)
	}
	if filter.ProjectPath != "" {
		q += " AND r.project_path = ?"
		args = append(args, filter.ProjectPath)
	}
	if filter.SessionID != "" {
		q += " AND r.session_id = ?"
		args = append(args, filter.SessionID)
	}
	if !filter.Since.IsZero() {
		q += " AND r.created_at >= ?"
		args = append(args, filter.Since.UTC().Format(time.RFC3339))
	}
	if filter.Search != "" {
		q += " AND (c.claimed LIKE ? OR c.actual LIKE ? OR c.target LIKE ?)"
		pat := "%" + filter.Search + "%"
		args = append(args, pat, pat, pat)
	}
	q += " ORDER BY r.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, filter.Offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LieClaim
	for rows.Next() {
		var lc LieClaim
		var ev, created, runCreated, verdict string
		if err := rows.Scan(&lc.ID, &lc.RunID, &lc.ClaimType, &lc.Source, &lc.Target, &lc.Claimed,
			&lc.Actual, &lc.Verified, &lc.Severity, &lc.Verifier, &ev, &created,
			&lc.ProjectPath, &lc.SessionID, &lc.RunCommand, &runCreated, &verdict); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(ev), &lc.Evidence)
		lc.CreatedAt, _ = time.Parse(time.RFC3339, created)
		lc.RunCreated, _ = time.Parse(time.RFC3339, runCreated)
		lc.RunVerdict = Verdict(verdict)
		out = append(out, lc)
	}
	return out, rows.Err()
}

// LieStats returns aggregate lie statistics.
func (s *Store) LieStats() (LieStats, error) {
	stats := LieStats{ByClaimType: make(map[string]int)}
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM runs`).Scan(&stats.TotalRuns)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM runs WHERE verdict IN ('fail','warn')`).Scan(&stats.SnitchedRuns)

	rows, err := s.db.Query(`
		SELECT claim_type, COUNT(*) FROM claims
		WHERE verified = -1 AND severity >= 2
		GROUP BY claim_type ORDER BY COUNT(*) DESC`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var n int
		if err := rows.Scan(&t, &n); err != nil {
			return stats, err
		}
		stats.ByClaimType[t] = n
		if stats.TopClaimType == "" {
			stats.TopClaimType = t
		}
	}
	return stats, rows.Err()
}

// CountRuns returns total run count.
func (s *Store) CountRuns() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM runs`).Scan(&n)
	return n, err
}

// RunExistsByOutputHash checks dedup.
func (s *Store) RunExistsByOutputHash(hash string) (bool, error) {
	if hash == "" {
		return false, nil
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM runs WHERE output_hash=?`, hash).Scan(&n)
	return n > 0, err
}

// EnqueueAnalytics adds payload to queue.
func (s *Store) EnqueueAnalytics(start, end string, payload []byte) error {
	_, err := s.db.Exec(`INSERT INTO analytics_queue (period_start, period_end, payload) VALUES (?,?,?)`,
		start, end, string(payload))
	return err
}

// Vacuum runs SQLite vacuum.
func (s *Store) Vacuum() error {
	_, err := s.db.Exec(`VACUUM`)
	return err
}

// ApplyRetention deletes old runs per policy.
func (s *Store) ApplyRetention(maxDays int, keepFailures bool) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -maxDays).Format(time.RFC3339)
	if keepFailures {
		_, err := s.db.Exec(`DELETE FROM runs WHERE created_at < ? AND verdict NOT IN ('fail')`, cutoff)
		return err
	}
	_, err := s.db.Exec(`DELETE FROM runs WHERE created_at < ?`, cutoff)
	return err
}

func scanRun(rows *sql.Rows) (Run, error) {
	return scanRunRow(rows)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRunRow(row rowScanner) (Run, error) {
	var r Run
	var created, verdict string
	var traceJSON sql.NullString
	err := row.Scan(&r.ID, &created, &r.SessionID, &r.TranscriptPath, &r.ProjectPath,
		&r.Harness, &r.Command, &r.DurationMS, &r.OutputHash, &r.ToolCallCount,
		&verdict, &r.MaxSeverity, &r.ClaimCount, &r.VerifiedClaims, &r.FalseClaims,
		&r.DeviceID, &traceJSON)
	if err != nil {
		return r, err
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, created)
	r.Verdict = Verdict(verdict)
	if traceJSON.Valid && traceJSON.String != "" {
		_ = json.Unmarshal([]byte(traceJSON.String), &r.Trace)
	}
	return r, nil
}

func scanClaims(rows *sql.Rows) ([]Claim, error) {
	var claims []Claim
	for rows.Next() {
		var c Claim
		var ev string
		var created string
		if err := rows.Scan(&c.ID, &c.RunID, &c.ClaimType, &c.Source, &c.Target, &c.Claimed,
			&c.Actual, &c.Verified, &c.Severity, &c.Verifier, &ev, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(ev), &c.Evidence)
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		claims = append(claims, c)
	}
	return claims, rows.Err()
}

// EnsureDeviceID returns or creates device ID in config table.
func (s *Store) EnsureDeviceID() (string, error) {
	var id string
	err := s.db.QueryRow(`SELECT value FROM config WHERE key='device_id'`).Scan(&id)
	if err == nil && id != "" {
		return id, nil
	}
	id = fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	_, err = s.db.Exec(`INSERT OR REPLACE INTO config (key, value) VALUES ('device_id', ?)`, id)
	return id, err
}
