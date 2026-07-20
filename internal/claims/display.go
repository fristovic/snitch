package claims

import (
	"fmt"
	"strings"

	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/shellpreview"
	"github.com/fristovic/snitch/internal/textutil"
)

// DisplayFields is the claim text needed for formatting.
type DisplayFields struct {
	ClaimType     string
	Source        string
	Claimed       string
	Actual        string
	ClaimSentence string
	ClaimContext  string
	Evidence      []string
	Target        string
	Verifier      string
	Severity      int
	Epistemic     string
}

// FromRecord builds DisplayFields from a persisted claim.
func FromRecord(c record.Claim) DisplayFields {
	return DisplayFields{
		ClaimType:     c.ClaimType,
		Source:        c.Source,
		Claimed:       c.Claimed,
		Actual:        c.Actual,
		ClaimSentence: c.ClaimSentence,
		ClaimContext:  c.ClaimContext,
		Evidence:      c.Evidence,
		Target:        c.Target,
		Verifier:      c.Verifier,
		Severity:      c.Severity,
		Epistemic:     record.ClaimEpistemic(c),
	}
}

// EpistemicLabel returns a short label for the claim's epistemic status.
func EpistemicLabel(epistemic string) string {
	switch epistemic {
	case "supported":
		return "Supported"
	case "contradicted":
		return "Contradicted"
	case "missing":
		return "Unverified"
	case "stale":
		return "Stale"
	default:
		if epistemic == "" {
			return ""
		}
		return epistemic
	}
}

var claimTypeLabels = map[string]string{
	TypeTestPass:          "Tests passed",
	TypeCommitted:         "Git commit",
	TypePushed:            "Git push",
	TypeFileCreated:       "File created",
	TypeFileModified:      "File modified",
	TypeFileDeleted:       "File deleted",
	TypeCommandRan:        "Command ran",
	TypeCommandSucceeded:  "Command succeeded",
	TypeStub:              "False completion",
	TypeNoAction:          "No action taken",
	TypeSelfContradiction: "Self-contradiction",
	TypeCountMismatch:     "File count mismatch",
	TypeNegationViolation: "Negation violation",
	TypeMissed:            "Missed claim",
}

// ClaimTypeLabel returns a short human label for a claim_type ID.
func ClaimTypeLabel(claimType string) string {
	claimType = strings.TrimSpace(claimType)
	if claimType == "" {
		return "Claim"
	}
	if label, ok := claimTypeLabels[claimType]; ok {
		return label
	}
	normalized := NormalizeClaimType(claimType)
	if label, ok := toolClaimLabels[normalized]; ok {
		return label
	}
	if label, ok := toolClaimLabels[claimType]; ok {
		return label
	}
	if label, ok := claimTypeLabels[normalized]; ok {
		return label
	}
	return claimType
}

const (
	maxToolFlaggedRunes  = 100
	maxProseFlaggedRunes = 240
	maxContextRunes      = 200
)

func isToolClaim(d DisplayFields) bool {
	t := NormalizeClaimType(d.ClaimType)
	return strings.HasPrefix(t, "tool_") || d.Source == "tool"
}

// FlaggedText returns the best user-facing "what was flagged" string.
func FlaggedText(d DisplayFields) string {
	if isToolClaim(d) {
		return compactToolFlagged(d)
	}
	if s := strings.TrimSpace(d.ClaimSentence); s != "" {
		return s
	}
	if s := strings.TrimSpace(d.Claimed); s != "" {
		return s
	}
	if s := strings.TrimSpace(d.Target); s != "" {
		return ClaimTypeLabel(d.ClaimType) + " " + s
	}
	return ClaimTypeLabel(d.ClaimType)
}

func compactToolFlagged(d DisplayFields) string {
	if t := strings.TrimSpace(d.Target); t != "" {
		return textutil.OneLine(shellpreview.OneLiner(t), maxToolFlaggedRunes)
	}
	raw := strings.TrimSpace(d.Claimed)
	if raw == "" {
		raw = strings.TrimSpace(d.ClaimSentence)
	}
	raw = stripToolClaimPrefix(raw)
	raw = shellpreview.OneLiner(raw)
	if raw == "" {
		return ClaimTypeLabel(d.ClaimType)
	}
	return textutil.OneLine(raw, maxToolFlaggedRunes)
}

func stripToolClaimPrefix(s string) string {
	for _, p := range toolClaimPrefixes {
		if strings.HasPrefix(s, p) {
			return strings.TrimSpace(s[len(p):])
		}
	}
	return s
}

// ShortSummary is for menu previews and list rows.
func ShortSummary(d DisplayFields, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 42
	}
	label := ClaimTypeLabel(d.ClaimType)
	flagged := textutil.OneLine(FlaggedText(d), maxRunes)
	if flagged == "" || flagged == label {
		return label
	}
	return fmt.Sprintf("%s — %q", label, flagged)
}

// NotificationBody is the short notification body: flagged → checked.
func NotificationBody(d DisplayFields, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 120
	}
	flagged := strings.TrimSpace(FlaggedText(d))
	actual := strings.TrimSpace(d.Actual)
	if actual == "" {
		return textutil.TruncateRunes(flagged, maxRunes)
	}
	return fmt.Sprintf("%q → %s",
		textutil.TruncateRunes(flagged, maxRunes),
		textutil.TruncateRunes(actual, maxRunes))
}

// ArrowSummary is a compact flagged → checked line (log summaries).
func ArrowSummary(d DisplayFields, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 72
	}
	return NotificationBody(d, maxRunes)
}

// RichDetail formats a multi-line detail block for log/dashboard.
func RichDetail(d DisplayFields) string {
	var b strings.Builder
	label := ClaimTypeLabel(d.ClaimType)
	b.WriteString("Claim: ")
	b.WriteString(label)
	if d.ClaimType != "" && d.ClaimType != label {
		b.WriteString(" (")
		b.WriteString(d.ClaimType)
		b.WriteString(")")
	}
	if d.Source != "" {
		b.WriteString(" [")
		b.WriteString(d.Source)
		b.WriteString("]")
	}
	b.WriteByte('\n')

	flagged := FlaggedText(d)
	if flagged != "" {
		cap := maxProseFlaggedRunes
		if isToolClaim(d) {
			cap = maxToolFlaggedRunes
		}
		b.WriteString("Flagged: ")
		b.WriteString(textutil.OneLine(flagged, cap))
		b.WriteByte('\n')
	}
	if !isToolClaim(d) {
		if ctx := strings.TrimSpace(d.ClaimContext); ctx != "" && ctx != flagged && ctx != strings.TrimSpace(d.ClaimSentence) {
			b.WriteString("Context: ")
			b.WriteString(textutil.OneLine(ctx, maxContextRunes))
			b.WriteByte('\n')
		}
	}
	if actual := strings.TrimSpace(d.Actual); actual != "" {
		b.WriteString("Checked: ")
		b.WriteString(textutil.OneLine(actual, 160))
		b.WriteByte('\n')
	}
	if label := EpistemicLabel(d.Epistemic); label != "" {
		b.WriteString("Status: ")
		b.WriteString(label)
		b.WriteByte('\n')
	}
	if d.Verifier != "" || d.Severity > 0 {
		b.WriteString("Verifier: ")
		if d.Verifier != "" {
			b.WriteString(d.Verifier)
		} else {
			b.WriteString("unknown")
		}
		if d.Severity > 0 {
			fmt.Fprintf(&b, " (sev %d)", d.Severity)
		}
		b.WriteByte('\n')
	}
	for _, e := range d.Evidence {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		b.WriteString("Evidence: ")
		b.WriteString(textutil.OneLine(e, 160))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
