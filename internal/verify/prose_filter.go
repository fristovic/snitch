package verify

import (
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func filterFileClaimsWithGitCommitOnly(claims []verifiers.Claim, calls []transcript.ToolCall, ctx verifiers.VerifyContext) []verifiers.Claim {
	if !gitCommitWithoutFileTools(calls) {
		return claims
	}
	// Keep file claims when prior turns have file tool evidence.
	for _, c := range claims {
		if isFileClaim(c.Type) && c.Target != "" && verifiers.HasFileToolForPath(ctx, c.Target) {
			return claims
		}
	}
	var out []verifiers.Claim
	for _, c := range claims {
		switch c.Type {
		case verifiers.ClaimFileCreated, verifiers.ClaimFileModified, verifiers.ClaimFileDeleted:
			continue
		default:
			out = append(out, c)
		}
	}
	return out
}

func gitCommitWithoutFileTools(calls []transcript.ToolCall) bool {
	hasCommit := false
	for _, tc := range calls {
		switch tc.Name {
		case "Write", "StrReplace", "Delete":
			return false
		case "Shell":
			cmd := strings.ToLower(shellCommandFromCall(tc))
			if strings.Contains(cmd, "git commit") {
				hasCommit = true
			}
		}
	}
	return hasCommit
}

func shellCommandFromCall(tc transcript.ToolCall) string {
	if tc.Target != "" {
		return tc.Target
	}
	return ""
}
