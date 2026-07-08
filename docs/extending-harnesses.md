# Adding a New Harness

Snitch supports any AI coding agent whose sessions can be read from disk. This
guide walks through the full checklist with a worked example: a fictional
JSONL-based agent called **Acme** that stores sessions under
`~/.acme/sessions/*.jsonl`.

A harness touches exactly five places. Nothing else in the pipeline (capture,
verification, storage, CLI, telemetry) needs changes — it is harness-agnostic
by design.

```
internal/transcript/parser_acme.go      1. parser (wire format → ParsedLine)
internal/transcript/path_acme.go        2. path resolver (cwd + session id)
internal/harness/acme.go                3. descriptor (registers everything)
internal/config/config.go + defaults.go 4. config entry
test/fixtures + tests                   5. required tests
```

## 1. The parser

Implement `transcript.TranscriptParser`: decode one JSONL line into a
normalized `ParsedLine`. Look at the existing parsers as templates —
[parser_claude.go](../internal/transcript/parser_claude.go) (content blocks,
tool results on user lines), [parser_codex.go](../internal/transcript/parser_codex.go)
(wrapped payloads, explicit turn markers), [parser_pi.go](../internal/transcript/parser_pi.go)
(nested message envelope, user-message turn boundaries).

```go
// internal/transcript/parser_acme.go
package transcript

import "encoding/json"

// AcmeParser decodes Acme's session JSONL format.
//
// Location: ~/.acme/sessions/<uuid>.jsonl
// Each line: {"role":"user"|"assistant","content":"...","tools":[...],"done":bool}
type AcmeParser struct{}

func (AcmeParser) Harness() string { return "acme" }

func (AcmeParser) ParseLine(line string) (ParsedLine, bool) {
	var row struct {
		Role    string `json:"role"`
		Content string `json:"content"`
		Tools   []struct {
			Name string          `json:"name"`
			ID   string          `json:"id"`
			Args json.RawMessage `json:"args"`
		} `json:"tools"`
		Done bool `json:"done"`
	}
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return ParsedLine{}, false // malformed lines are skipped, never fatal
	}
	pl := ParsedLine{Role: row.Role, Text: row.Content, TurnEnded: row.Done}
	for _, t := range row.Tools {
		tc := ToolCall{
			// canonicalToolName maps raw names (bash, edit, ...) to Snitch's
			// vocabulary (Shell, StrReplace, ...); pass overrides for names
			// the shared lowercase map doesn't cover.
			Name:      canonicalToolName(t.Name, nil),
			ToolUseID: t.ID,
		}
		if len(t.Args) > 0 {
			_ = json.Unmarshal(t.Args, &tc.Input)
		}
		tc.Target = deriveTarget(tc)
		pl.ToolCalls = append(pl.ToolCalls, tc)
	}
	if pl.Text == "" && len(pl.ToolCalls) == 0 && !pl.TurnEnded {
		return ParsedLine{}, false
	}
	return pl, true
}
```

Rules the parser must follow:

- **Skip thinking/reasoning blocks** — they are not prose claims.
- **Normalize tool names** via `canonicalToolName` so verifiers never see raw
  harness names. If your harness exposes the model name, set `ParsedLine.Model`.
- **Turn boundaries**: set `TurnEnded` on whatever your format uses (explicit
  marker, assistant stop reason, or the next user message like Pi — the turn
  assembler in [assembler.go](../internal/transcript/assembler.go) handles all
  three shapes, and the idle-flush timer covers sessions that end without a marker).
- **Tool results**: attach them as `ParsedLine.ToolResults` with the matching
  `ToolUseID`; correlation to calls happens automatically.

Non-JSONL sources (databases, etc.) implement a poll reader instead — see
[reader_opencode.go](../internal/transcript/reader_opencode.go) as the template.

## 2. The path resolver

Implement `transcript.PathResolver` (three methods). Reuse the shared helpers:

```go
// internal/transcript/path_acme.go
package transcript

type AcmePathResolver struct{}

// Acme doesn't encode the cwd in the path (like Codex) — return "" and set
// ParsedLine.Cwd from session metadata in the parser instead. If your harness
// encodes it like Cursor/Claude, use slugProjectCwd.
func (AcmePathResolver) ProjectCwd(path string) string  { return "" }
func (AcmePathResolver) ProjectDir(path string) string  { return "" }
func (AcmePathResolver) SessionID(path string) string   { return sessionIDFromFilename(path, "") }
```

## 3. The descriptor

Register everything in one place:

```go
// internal/harness/acme.go
package harness

import (
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

// acmeDescriptor describes Acme's transcript ingestion. Shell output is
// inline in tool results, so the shell resolver is noop.
func acmeDescriptor() Descriptor {
	return Descriptor{
		Name:  "acme",
		Shell: transcript.NoopShellOutputResolver(),
		Ingest: jsonlIngest("acme", transcript.AcmeParser{}, transcript.AcmePathResolver{},
			func(path string) bool { return strings.HasSuffix(path, ".jsonl") },
			func(path string) bool { return true }),
	}
}
```

Add `acmeDescriptor()` to the list in `NewRegistry` in
[registry.go](../internal/harness/registry.go). If your harness has an on-disk
shell-output artifact (like Cursor's terminal files), implement a
`ShellOutputResolver` in `internal/transcript` and set it as `Shell`.

## 4. Config

Two edits in `internal/config`:

- Add an `Acme PlatformConfig` field to `PlatformsConfig` and an entry in its
  `byHarness()` map ([config.go](../internal/config/config.go)) — that single
  map drives `ForHarness`, `config set/get`, and path expansion.
- Add a default in [defaults.go](../internal/config/defaults.go), **disabled**
  by default: `Acme: PlatformConfig{Enabled: false, TranscriptWatchPath: "~/.acme/sessions"}`.

Also add `"acme"` to the harness name list in `cmd/snitchd/main.go`
(`enabledPlatforms`) and `cmd/snitch/cmd/doctor.go`.

## 5. Required tests (PRs without these are not reviewed)

1. **Fixture**: a realistic sample session at
   `test/fixtures/sample_transcripts/acme_session.jsonl` — captured from a real
   session with secrets removed, covering at least: user text, assistant text,
   one tool call with result, one turn boundary.
2. **Parser test** in `internal/transcript/harness_parsers_test.go`: parse the
   fixture, assert tool-name normalization, turn boundaries, and tool-result
   correlation.
3. **Pipeline entry** in `test/integration/multiharness_test.go`: add your
   harness to the fixtures table so the full parse→verify→store path runs.
4. **Live watcher case** in `test/integration/watcher_live_test.go`: add a case
   so incremental fsnotify ingestion is covered.

## 6. Validate against a real session

Before opening the PR, run your parser against real on-device sessions:

```bash
go build ./... && go run ./cmd/snitch replay --harness acme ~/.acme/sessions/
```

Review the output: turns must have sensible user/assistant text, tool calls
must carry canonical names and targets, and there must be no obviously bogus
claims. Paste a (redacted) sample of the replay output in the PR description —
the PR template asks for it.

## Format verification is mandatory

Do not guess the wire format. Verify every field name against a real session
file (or official documentation) before writing the parser. Snitch's history
includes parsers written against imagined formats — they parse nothing and
waste everyone's review time.
