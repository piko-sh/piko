//go:build bench

package interp_test_bench

import (
	"context"
	"testing"

	"piko.sh/piko/internal/interp/interp_domain"
)

func BenchmarkRecursiveFibExecOnly(b *testing.B) {
	const source = `package main
func fib(n int) int {
	if n < 2 { return n }
	return fib(n-1) + fib(n-2)
}
func EntrypointRun() int { return fib(20) }`
	service := interp_domain.NewService(interp_domain.WithMaxCallDepth(200000))
	compiled, err := service.CompileFileSet(context.Background(),
		map[string]string{"main.go": source})
	if err != nil {
		b.Fatalf("compile: %v", err)
	}
	if _, err := service.ExecuteEntrypoint(context.Background(), compiled, "EntrypointRun"); err != nil {
		b.Fatalf("warmup: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := service.ExecuteEntrypoint(context.Background(), compiled, "EntrypointRun")
		if err != nil {
			b.Fatalf("exec: %v", err)
		}
	}
}
