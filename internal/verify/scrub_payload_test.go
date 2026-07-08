package verify

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/transcript"
)

func TestScrubPayloadRedactsAllTextFields(t *testing.T) {
	secret := "OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwx"
	p := capture.RunPayload{
		UserText:      "set " + secret + " please",
		AssistantText: "exported " + secret,
		Command:       secret,
		ToolCalls: []transcript.ToolCall{{
			Name:   "Shell",
			Target: "export " + secret,
			Result: "env now has " + secret,
			Input: map[string]json.RawMessage{
				"command": json.RawMessage(`"export ` + secret + `"`),
			},
		}},
	}
	out := scrubPayload(p)
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sk-abcdefghijklmnop") {
		t.Fatalf("secret survived scrub: %s", data)
	}
	if !json.Valid(data) {
		t.Fatal("scrubbed payload is not valid JSON")
	}
	// Original payload must be untouched (scrub works on a copy).
	if !strings.Contains(p.ToolCalls[0].Result, "sk-") {
		t.Fatal("scrubPayload mutated the input payload")
	}
}
