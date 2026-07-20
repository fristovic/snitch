package verifiers

// EpistemicRank orders outcomes for picking the worse verifier result on ties.
func EpistemicRank(e Epistemic) int {
	switch e {
	case EpistemicContradicted:
		return 3
	case EpistemicMissing:
		return 2
	case EpistemicStale:
		return 1
	default:
		return 0
	}
}

// EpistemicToVerified maps epistemic state to the legacy verified column.
func EpistemicToVerified(e Epistemic) int {
	switch e {
	case EpistemicSupported:
		return 1
	case EpistemicContradicted:
		return -1
	default:
		return 0
	}
}

// PreferWorseResult returns the more severe verification outcome.
func PreferWorseResult(a, b Result) Result {
	if b.Verifier == "" {
		return a
	}
	if a.Verifier == "" {
		return b
	}
	if b.Severity > a.Severity {
		return b
	}
	if b.Severity < a.Severity {
		return a
	}
	if EpistemicRank(b.Epistemic) > EpistemicRank(a.Epistemic) {
		return b
	}
	return a
}
