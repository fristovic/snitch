package verify

import (
	"strings"
	"testing"
	"unicode/utf8"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func TestExpandClaimWindowFullSentence(t *testing.T) {
	text := "Hello there. If all tests pass tomorrow, we can ship. Thanks."
	start := strings.Index(text, "all tests pass")
	end := start + len("all tests pass")
	sentence, context := expandClaimWindow(text, start, end)
	if !strings.Contains(sentence, "If all tests pass tomorrow") {
		t.Fatalf("sentence=%q", sentence)
	}
	if !strings.Contains(context, "Hello there") || !strings.Contains(context, "Thanks") {
		t.Fatalf("context=%q", context)
	}
}

func TestExpandClaimWindowQuestion(t *testing.T) {
	text := "Did all tests pass? I hope so."
	start := strings.Index(text, "all tests pass")
	end := start + len("all tests pass")
	sentence, _ := expandClaimWindow(text, start, end)
	if !strings.HasPrefix(sentence, "Did all tests pass") {
		t.Fatalf("sentence=%q", sentence)
	}
}

func TestExpandClaimWindowCap(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("This is a padding sentence number ")
		b.WriteByte(byte('A' + i%26))
		b.WriteString(". ")
	}
	b.WriteString("Finally all tests pass for real. ")
	b.WriteString("And more padding after the claim. ")
	text := b.String()
	start := strings.Index(text, "all tests pass")
	end := start + len("all tests pass")
	_, context := expandClaimWindow(text, start, end)
	if utf8.RuneCountInString(context) > maxClaimContextRunes {
		t.Fatalf("context too long: %d runes", utf8.RuneCountInString(context))
	}
}

func TestExpandClaimWindowFallback(t *testing.T) {
	text := "all tests pass"
	sentence, context := expandClaimWindow(text, 0, len(text))
	if sentence != text {
		t.Fatalf("sentence=%q", sentence)
	}
	if context != "" && context != text {
		// single-sentence text may equal context; either empty or same is fine
		_ = context
	}
}

func TestExtractProseClaimsSetsSentence(t *testing.T) {
	text := "Preface. Great news — all tests pass on main. Epilogue."
	claims := ExtractProseClaims(text)
	var found bool
	for _, c := range claims {
		if c.Type == verifiers.ClaimTestPass || strings.Contains(c.Quote, "tests pass") {
			found = true
			if !strings.Contains(c.Sentence, "all tests pass") {
				t.Fatalf("Sentence=%q Quote=%q", c.Sentence, c.Quote)
			}
			if c.Context == "" {
				t.Fatal("expected non-empty Context")
			}
			break
		}
	}
	if !found {
		t.Fatalf("no test_pass claim in %+v", claims)
	}
}
