package verify

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxClaimContextRunes = 500

// expandClaimWindow returns the full sentence containing the match [start,end)
// and a capped ±1–2 sentence context window from the same text.
// If expansion fails, sentence falls back to the match span and context is empty.
func expandClaimWindow(text string, start, end int) (sentence, context string) {
	if start < 0 {
		start = 0
	}
	if end > len(text) {
		end = len(text)
	}
	if start >= end {
		return "", ""
	}
	quote := text[start:end]
	sentStart, sentEnd := sentenceBounds(text, start, end)
	if sentStart >= sentEnd {
		return quote, ""
	}
	sentence = strings.TrimSpace(text[sentStart:sentEnd])
	if sentence == "" {
		sentence = quote
	}

	prev1Start, _ := prevSentenceBounds(text, sentStart)
	prev2Start, _ := prevSentenceBounds(text, prev1Start)
	_, next1End := nextSentenceBounds(text, sentEnd)
	_, next2End := nextSentenceBounds(text, next1End)

	ctxStart := prev2Start
	if prev1Start > 0 && prev2Start == prev1Start {
		ctxStart = prev1Start
	}
	ctxEnd := next2End
	if next2End == next1End {
		ctxEnd = next1End
	}
	if ctxStart < 0 {
		ctxStart = 0
	}
	if ctxEnd > len(text) {
		ctxEnd = len(text)
	}
	if ctxStart >= ctxEnd {
		return sentence, ""
	}
	context = truncateRunes(strings.TrimSpace(text[ctxStart:ctxEnd]), maxClaimContextRunes)
	return sentence, context
}

func sentenceBounds(text string, start, end int) (int, int) {
	s := start
	for s > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:s])
		if r == utf8.RuneError && size == 1 {
			s--
			continue
		}
		if isSentenceBoundary(r) || r == '\n' {
			break
		}
		s -= size
	}
	// Skip leading whitespace after previous boundary.
	for s < len(text) {
		r, size := utf8.DecodeRuneInString(text[s:])
		if !unicode.IsSpace(r) {
			break
		}
		s += size
	}

	e := end
	for e < len(text) {
		r, size := utf8.DecodeRuneInString(text[e:])
		if r == utf8.RuneError && size == 1 {
			e++
			continue
		}
		e += size
		if isSentenceBoundary(r) || r == '\n' {
			break
		}
	}
	return s, e
}

func prevSentenceBounds(text string, before int) (int, int) {
	if before <= 0 {
		return 0, 0
	}
	// Walk back over trailing whitespace/boundary of the current sentence.
	i := before
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:i])
		if !unicode.IsSpace(r) && !isSentenceBoundary(r) {
			break
		}
		i -= size
	}
	if i <= 0 {
		return 0, 0
	}
	return sentenceBounds(text, i-1, i)
}

func nextSentenceBounds(text string, after int) (int, int) {
	if after >= len(text) {
		return len(text), len(text)
	}
	i := after
	for i < len(text) {
		r, size := utf8.DecodeRuneInString(text[i:])
		if !unicode.IsSpace(r) && !isSentenceBoundary(r) {
			break
		}
		i += size
	}
	if i >= len(text) {
		return len(text), len(text)
	}
	return sentenceBounds(text, i, i+1)
}

func isSentenceBoundary(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？'
}

func truncateRunes(s string, n int) string {
	if n <= 0 || utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	if n <= 1 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}
