package benchmark_test

import (
	"testing"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func BenchmarkFileVerifier(b *testing.B) {
	v := &verifiers.FileVerifier{}
	claim := verifiers.Claim{Type: "Read", Source: "tool", Target: "/etc/hosts"}
	ctx := verifiers.VerifyContext{Cwd: "/"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.Verify(claim, ctx)
	}
}
