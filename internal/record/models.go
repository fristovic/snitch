package record

import (
	"time"
)

// Verdict is the overall run verdict.
type Verdict string

const (
	VerdictPass       Verdict = "pass"
	VerdictWarn       Verdict = "warn"
	VerdictFail       Verdict = "fail"
	VerdictUnverified Verdict = "unverified"
)

// Run represents a single agent turn record.
type Run struct {
	ID             string            `json:"id"`
	CreatedAt      time.Time         `json:"created_at"`
	SessionID      string            `json:"session_id,omitempty"`
	TranscriptPath string            `json:"transcript_path,omitempty"`
	ProjectPath    string            `json:"project_path,omitempty"`
	Harness        string            `json:"harness,omitempty"`
	Model          string            `json:"model,omitempty"`
	Command        string            `json:"command,omitempty"`
	DurationMS     int64             `json:"duration_ms,omitempty"`
	OutputHash     string            `json:"output_hash,omitempty"`
	ToolCallCount  int               `json:"tool_call_count"`
	Verdict        Verdict           `json:"verdict"`
	MaxSeverity    int               `json:"max_severity,omitempty"`
	ClaimCount     int               `json:"claim_count"`
	VerifiedClaims int               `json:"verified_claims"`
	FalseClaims    int               `json:"false_claims"`
	DeviceID       string            `json:"device_id"`
	Trace          []string          `json:"trace,omitempty"`
	StartHEAD      string            `json:"start_head,omitempty"`
	EndHEAD        string            `json:"end_head,omitempty"`
	FileManifest   map[string]string `json:"file_manifest,omitempty"`
	PayloadJSON    string            `json:"-"` // raw stored payload; use GetRunPayload

	// User feedback labels (v1 data flywheel). Populated by SetRunLabel; read
	// on demand by the feedback UI and telemetry sync.
	LabelVerdict   string    `json:"label_verdict,omitempty"`   // "correct" | "incorrect"
	LabelShared    bool      `json:"label_shared,omitempty"`    // user opted into sharing
	LabelTimestamp time.Time `json:"label_timestamp,omitempty"` // when labeled
	LabelSession   string    `json:"label_session,omitempty"`   // dedup key
	LabelSynced    bool      `json:"label_synced,omitempty"`    // telemetry forwarded
}

// RunLabel is a labeled run (or missed-claim report) ready for telemetry
// sync. When the user opts into sharing, training fields (sentence, context,
// claimed, actual) may leave the machine — never prompts, code, or paths.
type RunLabel struct {
	RunID           string    `json:"run_id,omitempty"`
	MissedID        int64     `json:"-"` // local id for missed-claim rows
	Harness         string    `json:"harness,omitempty"`
	Model           string    `json:"model,omitempty"`             // underlying LLM, if extractable
	ClaimType       string    `json:"claim_type,omitempty"`        // top claim type, if any
	Verdict         Verdict   `json:"verdict,omitempty"`           // Snitch's original verdict
	LabelVerdict    string    `json:"label_verdict"`               // "correct" | "incorrect" | "added"
	ClaimedTextHash string    `json:"claimed_text_hash,omitempty"` // sha256 of claim sentence (dedup)
	ClaimSentence   string    `json:"claim_sentence,omitempty"`    // full sentence containing the match
	ClaimContext    string    `json:"claim_context,omitempty"`     // capped surrounding sentences
	Claimed         string    `json:"claimed,omitempty"`           // Snitch claimed text / missed claimed
	Actual          string    `json:"actual,omitempty"`            // Snitch actual / missed actual
	LabeledAt       time.Time `json:"labeled_at"`
}

// Claim is a verified prose or tool claim.
type Claim struct {
	ID            int64     `json:"id"`
	RunID         string    `json:"run_id"`
	ClaimType     string    `json:"claim_type"`
	Source        string    `json:"source"`
	Target        string    `json:"target"`
	Claimed       string    `json:"claimed"`
	Actual        string    `json:"actual,omitempty"`
	ClaimSentence string    `json:"claim_sentence,omitempty"` // full sentence containing match
	ClaimContext  string    `json:"claim_context,omitempty"`  // capped surrounding sentences
	Verified      int       `json:"verified"`
	Severity      int       `json:"severity"`
	Verifier      string    `json:"verifier,omitempty"`
	Evidence      []string  `json:"evidence,omitempty"`
	Confidence    int       `json:"confidence,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// LieClaim is a claim joined with run context for querying.
type LieClaim struct {
	Claim
	ProjectPath string    `json:"project_path"`
	SessionID   string    `json:"session_id"`
	RunCommand  string    `json:"run_command,omitempty"`
	RunCreated  time.Time `json:"run_created"`
	RunVerdict  Verdict   `json:"run_verdict"`
}

// RunFilter filters run queries.
type RunFilter struct {
	Verdict      string
	Harness      string
	ProjectPath  string
	SessionID    string
	Search       string
	Since        time.Time
	FailuresOnly bool
	Limit        int
	Offset       int
}

// ClaimFilter filters lie/claim queries.
type ClaimFilter struct {
	ClaimType   string
	ProjectPath string
	SessionID   string
	MinSeverity int
	LiesOnly    bool
	Since       time.Time
	Search      string
	Limit       int
	Offset      int
}

// LieStats summarizes caught lies.
type LieStats struct {
	TotalRuns    int            `json:"total_runs"`
	SnitchedRuns int            `json:"snitched_runs"`
	ByClaimType  map[string]int `json:"by_claim_type"`
	TopClaimType string         `json:"top_claim_type,omitempty"`
}

// DaemonStatus is returned by IPC status method.
type DaemonStatus struct {
	Running         bool     `json:"running"`
	UptimeSeconds   int64    `json:"uptime_seconds"`
	Version         string   `json:"version"`
	TotalRuns       int      `json:"total_runs"`
	SnitchedRuns    int      `json:"snitched_runs"`
	TopLieType      string   `json:"top_lie_type,omitempty"`
	ProjectsWatched int      `json:"projects_watched"`
	SessionsSeen    int      `json:"sessions_seen"`
	LieStats        LieStats `json:"lie_stats"`
	// RunsByHarness maps harness name → run count, for per-platform status.
	RunsByHarness map[string]int `json:"runs_by_harness,omitempty"`
}
