package record

// SelectTopFalseClaim returns the highest-severity contradicted claim.
// Tie-break: later index wins (callers should pass claims in insertion order).
func SelectTopFalseClaim(claims []Claim) *Claim {
	var best *Claim
	for i := range claims {
		c := &claims[i]
		if !isContradictedClaim(*c) {
			continue
		}
		if best == nil || c.Severity > best.Severity {
			best = c
		}
	}
	return best
}

func isContradictedClaim(c Claim) bool {
	if c.Epistemic == "contradicted" {
		return true
	}
	return c.Epistemic == "" && c.Verified == -1
}

// IsContradictedClaim reports whether a stored claim is epistemically contradicted.
func IsContradictedClaim(c Claim) bool {
	return isContradictedClaim(c)
}

// ClaimEpistemic returns the epistemic status for display and filtering.
// Legacy rows with empty epistemic are mapped from verified until backfill runs.
func ClaimEpistemic(c Claim) string {
	if c.Epistemic != "" {
		return c.Epistemic
	}
	switch {
	case c.Verified > 0:
		return "supported"
	case c.Verified < 0:
		return "contradicted"
	default:
		return "missing"
	}
}

// ShowClaimInDetail reports whether a claim should appear in log/dashboard problem views.
func ShowClaimInDetail(c Claim, minSeverity int) bool {
	if c.Severity >= minSeverity {
		return true
	}
	return ClaimEpistemic(c) != "supported"
}
