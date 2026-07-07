package verifiers_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func TestContradictionTestPassWithFailedResult(t *testing.T) {
	v := &verifiers.ContradictionVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimTestPass, Source: "prose", Quote: "all tests pass",
	}, verifiers.VerifyContext{
		ToolCalls: []transcript.ToolCall{{
			Name:      "Shell",
			ToolUseID: "toolu_1",
			Target:    "go test ./...",
			Result:    "FAIL\tpkg\tbuild failed\n",
			IsError:   true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Accurate || res.Severity != severity.Level3 {
		t.Fatalf("expected L3 lie, got %+v", res)
	}
}

func TestContradictionTestPassNoRun(t *testing.T) {
	v := &verifiers.ContradictionVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimTestPass, Source: "prose", Quote: "all tests pass",
	}, verifiers.VerifyContext{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Accurate || res.Severity != severity.Level3 {
		t.Fatalf("expected L3 lie, got %+v", res)
	}
}

func TestContradictionStubDetection(t *testing.T) {
	v := &verifiers.ContradictionVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimStub, Source: "prose", Quote: "fully implemented",
	}, verifiers.VerifyContext{
		ToolCalls: []transcript.ToolCall{{
			Name:   "Write",
			Target: "main.go",
			Input: map[string]json.RawMessage{
				"contents": json.RawMessage(`"package main\n\nfunc main() { panic(\"TODO\") }"`),
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Accurate || res.Severity != severity.Level3 {
		t.Fatalf("expected stub lie, got %+v", res)
	}
}

func TestConsistencySelfContradiction(t *testing.T) {
	claims := verifiers.ExtractConsistencyClaims(
		"I won't modify schema.sql in this turn.",
		[]transcript.ToolCall{{Name: "Write", Target: "schema.sql"}},
		"",
	)
	if len(claims) != 1 || claims[0].Type != verifiers.ClaimSelfContradiction {
		t.Fatalf("claims: %+v", claims)
	}
	v := &verifiers.ConsistencyVerifier{}
	res, err := v.Verify(claims[0], verifiers.VerifyContext{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Accurate {
		t.Fatalf("expected inconsistency, got %+v", res)
	}
}

func TestParseTestOutput(t *testing.T) {
	passed, found := verifiers.ParseTestOutput("ok\tpkg\t0.5s\n")
	if !found || !passed {
		t.Fatal("expected pass")
	}
	passed, found = verifiers.ParseTestOutput("FAIL\tpkg\n")
	if !found || passed {
		t.Fatal("expected fail")
	}
}

func TestContradictionCommitNoEvidence(t *testing.T) {
	v := &verifiers.ContradictionVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimCommitted, Source: "prose",
	}, verifiers.VerifyContext{ProjectPath: t.TempDir(), StartHEAD: "abc123"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Accurate || res.Severity != severity.Level3 {
		t.Fatalf("expected L3, got %+v", res)
	}
}

func TestContradictionFileCreatedWithWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := &verifiers.ContradictionVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimFileCreated, Source: "prose", Target: "foo.go",
	}, verifiers.VerifyContext{
		ProjectPath: dir,
		ToolCalls: []transcript.ToolCall{
			{Name: "Write", Target: "foo.go"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accurate {
		t.Fatalf("expected pass, got %+v", res)
	}
}

func TestContradictionNoAction(t *testing.T) {
	v := &verifiers.ContradictionVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimNoAction, Source: "prose",
	}, verifiers.VerifyContext{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Accurate || res.Severity != severity.Level3 {
		t.Fatalf("expected L3 no_action lie, got %+v", res)
	}
}

func TestTerminalFileParsing(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/terminals/1.txt"
	// exercised indirectly via time window helper defaults
	_ = time.Now()
	_, _, found := verifiers.ShellOutputForCommand(transcript.ToolCall{
		Name:   "Shell",
		Target: "echo hi",
	}, verifiers.VerifyContext{})
	if found {
		t.Fatal("expected no terminal match in empty context")
	}
	_ = path
}
