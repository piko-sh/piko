// Copyright 2026 PolitePixels Limited
// Benchmark to verify allocation behaviour for AttributeWriters slice growth

//go:build bench

package generator_helpers

import (
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func BenchmarkAttrWriters_NoPrealloc(b *testing.B) {

	for range 100 {
		dw := ast_domain.GetDirectWriter()
		ast_domain.PutDirectWriter(dw)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var writers []*ast_domain.DirectWriter
		for range 6 {
			dw := ast_domain.GetDirectWriter()
			writers = append(writers, dw)
		}
		for _, dw := range writers {
			ast_domain.PutDirectWriter(dw)
		}
	}
}

func BenchmarkAttrWriters_WithPrealloc(b *testing.B) {

	for range 100 {
		dw := ast_domain.GetDirectWriter()
		ast_domain.PutDirectWriter(dw)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		writers := make([]*ast_domain.DirectWriter, 0, 6)
		for range 6 {
			dw := ast_domain.GetDirectWriter()
			writers = append(writers, dw)
		}
		for _, dw := range writers {
			ast_domain.PutDirectWriter(dw)
		}
	}
}

func BenchmarkAttrWriters_NoPrealloc_10(b *testing.B) {
	for range 100 {
		dw := ast_domain.GetDirectWriter()
		ast_domain.PutDirectWriter(dw)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var writers []*ast_domain.DirectWriter
		for range 10 {
			dw := ast_domain.GetDirectWriter()
			writers = append(writers, dw)
		}
		for _, dw := range writers {
			ast_domain.PutDirectWriter(dw)
		}
	}
}

func BenchmarkAttrWriters_WithPrealloc_10(b *testing.B) {
	for range 100 {
		dw := ast_domain.GetDirectWriter()
		ast_domain.PutDirectWriter(dw)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		writers := make([]*ast_domain.DirectWriter, 0, 10)
		for range 10 {
			dw := ast_domain.GetDirectWriter()
			writers = append(writers, dw)
		}
		for _, dw := range writers {
			ast_domain.PutDirectWriter(dw)
		}
	}
}
