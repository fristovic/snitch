package verifiers_test

import (
	"testing"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func TestParseTestOutputFrameworks(t *testing.T) {
	cases := []struct {
		name   string
		output string
		pass   bool
		found  bool
	}{
		{"go pass", "ok  \tgithub.com/foo\t0.1s\n", true, true},
		{"go fail", "--- FAIL: TestFoo", false, true},
		{"pytest fail", "===== 2 failed, 1 passed in 0.1s =====", false, true},
		{"cargo ok", "test result: ok. 1 passed", true, true},
		{"cargo fail", "error: test failed", false, true},
		{"npm fail", "Tests: 1 failed, 2 passed", false, true},
		{"vitest fail", "Test Files 1 failed | 2 passed", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pass, found := verifiers.ParseTestOutput(tc.output)
			if found != tc.found || pass != tc.pass {
				t.Fatalf("got pass=%v found=%v", pass, found)
			}
		})
	}
}
