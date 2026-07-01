package verifiers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

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
