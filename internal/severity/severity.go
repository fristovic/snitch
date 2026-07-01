package severity

// Level identifies claim inaccuracy severity (0 = verified, 3 = false claim).
type Level int

const (
	Level0 Level = iota
	Level1
	Level2
	Level3
)

// String returns a human-readable label.
func (l Level) String() string {
	switch l {
	case Level0:
		return "verified"
	case Level1:
		return "minor inaccuracy"
	case Level2:
		return "partial failure"
	case Level3:
		return "false claim"
	default:
		return "unknown"
	}
}

// Verdict maps max severity to a run verdict string.
func Verdict(max Level) string {
	switch {
	case max >= Level3:
		return "fail"
	case max >= Level2:
		return "warn"
	default:
		return "pass"
	}
}
