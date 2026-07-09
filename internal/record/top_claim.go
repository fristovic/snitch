package record

// SelectTopFalseClaim returns the highest-severity false claim (verified=-1).
// Tie-break: later index wins (callers should pass claims in insertion order).
// Returns nil when there is no false claim.
func SelectTopFalseClaim(claims []Claim) *Claim {
	var best *Claim
	for i := range claims {
		c := &claims[i]
		if c.Verified != -1 {
			continue
		}
		if best == nil || c.Severity > best.Severity {
			best = c
		}
	}
	return best
}
