//go:build bench

package interp_test_bench

import (
	"context"
	"testing"

	"piko.sh/piko/internal/interp/interp_domain"
)

const (
	dotProductSize = 1024
)

func BenchmarkDotProductNative(b *testing.B) {
	a := make([]float64, dotProductSize)
	c := make([]float64, dotProductSize)
	for i := range a {
		a[i] = float64(i) * 0.5
		c[i] = float64(i) * 0.25
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		sum := 0.0
		for i := range a {
			sum += a[i] * c[i]
		}
		_ = sum
	}
}

func BenchmarkDotProductPiko(b *testing.B) {
	const source = `package main
const dotProductSize = 1024
func EntrypointRun() float64 {
	a := make([]float64, dotProductSize)
	c := make([]float64, dotProductSize)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i) * 0.5
		c[i] = float64(i) * 0.25
	}
	sum := 0.0
	for i := 0; i < len(a); i++ {
		sum += a[i] * c[i]
	}
	return sum
}`
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

func BenchmarkDotProductPikoLoopOnly(b *testing.B) {
	const source = `package main
const dotProductSize = 1024
var a []float64
var c []float64
func init() {
	a = make([]float64, dotProductSize)
	c = make([]float64, dotProductSize)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i) * 0.5
		c[i] = float64(i) * 0.25
	}
}
func EntrypointRun() float64 {
	sum := 0.0
	for i := 0; i < len(a); i++ {
		sum += a[i] * c[i]
	}
	return sum
}`
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
