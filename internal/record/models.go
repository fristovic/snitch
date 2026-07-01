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
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	SessionID      string    `json:"session_id,omitempty"`
	TranscriptPath string    `json:"transcript_path,omitempty"`
	ProjectPath    string    `json:"project_path,omitempty"`
	Harness        string    `json:"harness,omitempty"`
	Command        string    `json:"command,omitempty"`
	DurationMS     int64     `json:"duration_ms,omitempty"`
	OutputHash     string    `json:"output_hash,omitempty"`
	ToolCallCount  int       `json:"tool_call_count"`
	Verdict        Verdict   `json:"verdict"`
	MaxSeverity    int       `json:"max_severity,omitempty"`
	ClaimCount     int       `json:"claim_count"`
	VerifiedClaims int       `json:"verified_claims"`
	FalseClaims    int       `json:"false_claims"`
	DeviceID       string    `json:"device_id"`
	Trace          []string  `json:"trace,omitempty"`
}

// Claim is a verified prose or tool claim.
type Claim struct {
	ID        int64     `json:"id"`
	RunID     string    `json:"run_id"`
	ClaimType string    `json:"claim_type"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Claimed   string    `json:"claimed"`
	Actual    string    `json:"actual,omitempty"`
	Verified  int       `json:"verified"`
	Severity  int       `json:"severity"`
	Verifier  string    `json:"verifier,omitempty"`
	Evidence  []string  `json:"evidence,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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
	Model        string
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
	ClaimType    string
	ProjectPath  string
	SessionID    string
	MinSeverity  int
	LiesOnly     bool
	Since        time.Time
	Search       string
	Limit        int
	Offset       int
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
}
