package verify

import (
	"strings"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func shouldSuppressClaim(t verifiers.ClaimType, text string, start, end int) bool {
	switch t {
	case verifiers.ClaimTestPass:
		return isFigurativeTestPhrase(text, start, end)
	case verifiers.ClaimCommitted:
		return isHistoricalOrFigurativeCommit(text, start, end)
	case verifiers.ClaimPushed:
		return isFigurativePush(text, start, end)
	case verifiers.ClaimCommandRan:
		return isFigurativeCommandRan(text, start, end)
	case verifiers.ClaimCommandSucceeded:
		return isConditionalPhrase(text, start)
	case verifiers.ClaimStub:
		return isColloquialDone(text, start, end)
	default:
		return false
	}
}

func isFigurativeTestPhrase(text string, start, end int) bool {
	snippet := strings.ToLower(text[start:end])
	if strings.Contains(snippet, "your patience") {
		return true
	}
	after := strings.ToLower(strings.TrimSpace(text[end:]))
	if strings.HasPrefix(after, "in the spec") || strings.HasPrefix(after, "in the dashboard") {
		return true
	}
	window := strings.ToLower(text[max(0, start-20):min(len(text), end+30)])
	return strings.Contains(window, "last week") || strings.Contains(window, "in ci")
}

func isHistoricalOrFigurativeCommit(text string, start, end int) bool {
	window := strings.ToLower(text[max(0, start-20):min(len(text), end+40)])
	if strings.Contains(window, "commit to ") {
		return true
	}
	for _, phrase := range []string{
		"earlier", "yesterday", "you committed", "pre-commit", "amend committed",
		"changelog says", "changelog notes", "do not amend", "hook committed",
	} {
		if strings.Contains(window, phrase) {
			return true
		}
	}
	return false
}

func isFigurativePush(text string, start, end int) bool {
	window := strings.ToLower(text[max(0, start-10):min(len(text), end+30)])
	for _, phrase := range []string{
		"pushed for", "pushed forward", "pushed to the docs", "pushed to clarify",
	} {
		if strings.Contains(window, phrase) {
			return true
		}
	}
	return false
}

func isFigurativeCommandRan(text string, start, end int) bool {
	after := strings.ToLower(strings.TrimSpace(text[end:]))
	return strings.HasPrefix(after, "in my head") || strings.HasPrefix(after, "into ") ||
		strings.HasPrefix(after, "output ")
}

func isConditionalPhrase(text string, start int) bool {
	before := strings.ToLower(text[max(0, start-20):start])
	return strings.Contains(before, "if ") || strings.Contains(before, "if the ")
}

func isColloquialDone(text string, start, end int) bool {
	snippet := strings.ToLower(strings.TrimSpace(text[start:end]))
	if snippet == "done" || snippet == "done." {
		return true
	}
	if strings.HasPrefix(snippet, "we're done") || strings.HasPrefix(snippet, "we are done") {
		return true
	}
	if strings.Contains(snippet, "done building") || strings.Contains(snippet, "done with") {
		return true
	}
	if strings.Contains(snippet, "nothing left to do on your end") {
		return true
	}
	return false
}
