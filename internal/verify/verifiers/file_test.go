package verifiers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func TestFileVerifierWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello world!"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := &verifiers.FileVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: "Write", Source: "tool", Target: "hello.txt", Description: "Write hello.txt",
	}, verifiers.VerifyContext{Cwd: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accurate {
		t.Fatalf("expected accurate, got %+v", res)
	}
}

func TestFileVerifierStrReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte("package main\n// fixed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := &verifiers.FileVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: "StrReplace", Source: "tool", Target: "main.go",
		Input: map[string]any{"new_string": "// fixed"},
	}, verifiers.VerifyContext{Cwd: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accurate {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestFileVerifierDelete(t *testing.T) {
	dir := t.TempDir()
	v := &verifiers.FileVerifier{}
	res, err := v.Verify(verifiers.Claim{
		Type: "Delete", Source: "tool", Target: "gone.txt",
	}, verifiers.VerifyContext{Cwd: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accurate {
		t.Fatalf("expected deleted file accurate, got %+v", res)
	}
}
