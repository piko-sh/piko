// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

// Package render_test_bench provides benchmarks for the render package.
//
// This benchmark suite is designed to:
//   - Establish performance baselines before optimisations
//   - Enable regression detection via benchstat comparisons
//   - Profile memory allocation patterns
//   - Understand scaling characteristics
//
// # Running Benchmarks
//
// Run all benchmarks:
// go test -bench=. -benchmem ./internal/render/render_test_bench/...
// Run specific benchmark groups:
// go test -bench=BenchmarkBaseline -benchmem ./internal/render/render_test_bench/...
// go test -bench=BenchmarkScaling -benchmem ./internal/render/render_test_bench/...
// go test -bench=BenchmarkMemory -benchmem ./internal/render/render_test_bench/...
// Generate baseline for comparison:
// go test -bench=. -benchmem -count=10 ./internal/render/render_test_bench/... | tee baseline.txt
// Compare after changes:
// go test -bench=. -benchmem -count=10 ./internal/render/render_test_bench/... | tee new.txt
// benchstat baseline.txt new.txt
// # CPU Profiling
// go test -bench=BenchmarkBaseline_Reference -benchmem -cpuprofile=cpu.prof ./internal/render/render_test_bench/...
// go tool pprof cpu.prof
// # Memory Profiling
// go test -bench=BenchmarkBaseline_Reference -benchmem -memprofile=mem.prof ./internal/render/render_test_bench/...
// go tool pprof mem.prof
// # Trace Analysis
// go test -bench=BenchmarkBaseline_Reference -trace=trace.out ./internal/render/render_test_bench/...
// go tool trace trace.out

//go:build bench

package render_test_bench

import (
	"context"
	"io"
	"testing"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func BenchmarkBaseline_Reference(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	b.Run("Minimal", func(b *testing.B) {
		ast := BuildGenericAST(5)
		metadata := &templater_dto.InternalMetadata{}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "baseline-minimal", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
		}
	})

	b.Run("Standard", func(b *testing.B) {
		ast := BuildMixedAST(5)
		metadata := &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{
				Title:       "Standard Page",
				Description: "A standard test page",
			},
			CustomTags: []string{"my-card", "another-component"},
		}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "baseline-standard", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
		}
	})

	b.Run("Complex", func(b *testing.B) {
		ast := BuildMixedAST(20)
		metadata := &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{
				Title:       "Complex Page",
				Description: "A complex test page with many sections",
			},
			CustomTags: []string{"my-card", "another-component", "custom-button"},
		}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "baseline-complex", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
		}
	})
}

func BenchmarkBaseline_FullPage(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildMixedAST(10)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title:        "Full Page Baseline",
			Description:  "Complete page with all rendering features",
			Keywords:     "benchmark, test, render",
			CanonicalURL: "https://example.com/benchmark",
			Language:     "en",
		},
		CustomTags: []string{"my-card", "another-component"},
	}
	websiteConfig := &config.WebsiteConfig{
		Theme: map[string]string{
			"primary-color":   "#0066cc",
			"secondary-color": "#333333",
			"font-family":     "system-ui, sans-serif",
		},
	}
	styling := `.container { max-width: 1200px; margin: 0 auto; }
.header { padding: 1rem 0; }
.main-content { padding: 2rem 0; }
.footer { padding: 1rem 0; background: #f5f5f5; }`

	WarmUpOrchestrator(orchestrator, ast)

	b.ReportAllocs()
	b.ResetTimer()

	var totalBytes int64
	for b.Loop() {
		cw := &CountingWriter{}
		request := NewTestRequest()
		_ = orchestrator.RenderAST(ctx, cw, nil, request, render_domain.RenderASTOptions{PageID: "baseline-full", Template: ast, Metadata: metadata, Styling: styling, SiteConfig: websiteConfig})
		totalBytes += cw.BytesWritten
	}

	b.StopTimer()
	if b.N > 0 {
		b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
	}
}

func BenchmarkBaseline_Fragment(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildMixedAST(5)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "Fragment Baseline",
		},
		CustomTags: []string{"my-card"},
	}

	WarmUpOrchestrator(orchestrator, ast)

	b.ReportAllocs()
	b.ResetTimer()

	var totalBytes int64
	for b.Loop() {
		cw := &CountingWriter{}
		request := NewTestRequest()
		_ = orchestrator.RenderAST(ctx, cw, nil, request, render_domain.RenderASTOptions{PageID: "baseline-fragment", Template: ast, Metadata: metadata, IsFragment: true, SiteConfig: &config.WebsiteConfig{}})
		totalBytes += cw.BytesWritten
	}

	b.StopTimer()
	if b.N > 0 {
		b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
	}
}

func BenchmarkBaseline_SVGHeavy(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildSVGHeavyAST(30)
	metadata := &templater_dto.InternalMetadata{}

	WarmUpOrchestrator(orchestrator, ast)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		request := NewTestRequest()
		_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "baseline-svg", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
	}
}

func BenchmarkBaseline_FormHeavy(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildCSRFHeavyAST(10)
	metadata := &templater_dto.InternalMetadata{}

	WarmUpOrchestrator(orchestrator, ast)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		request := NewTestRequest()
		_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "baseline-forms", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
	}
}

func BenchmarkBaseline_DeepNesting(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildDeepAST(50)
	metadata := &templater_dto.InternalMetadata{}

	WarmUpOrchestrator(orchestrator, ast)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		request := NewTestRequest()
		_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "baseline-deep", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
	}
}

func BenchmarkBaseline_Wide(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildFlatAST(100)
	metadata := &templater_dto.InternalMetadata{}

	WarmUpOrchestrator(orchestrator, ast)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		request := NewTestRequest()
		_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "baseline-wide", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
	}
}

func BenchmarkBaseline_Parallel(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildMixedAST(5)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "Parallel Test",
		},
		CustomTags: []string{"my-card", "another-component"},
	}

	WarmUpOrchestrator(orchestrator, ast)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			request := NewTestRequest()
			_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "baseline-parallel", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
		}
	})
}

func BenchmarkBaseline_QuickSmoke(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildMixedAST(3)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "Smoke Test",
		},
		CustomTags: []string{"my-card"},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		request := NewTestRequest()
		err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "smoke", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
		if err != nil {
			b.Fatal(err)
		}
	}
}
