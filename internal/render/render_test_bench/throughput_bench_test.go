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

func BenchmarkRenderThroughput_BySize(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	request := NewTestRequest()
	ctx := context.Background()

	sizes := []FixtureSize{SizeTiny, SizeSmall, SizeMedium, SizeLarge}

	for _, size := range sizes {
		b.Run(size.String(), func(b *testing.B) {
			ast := BuildGenericAST(size.NodeCount())
			nodeCount := CountNodes(ast)
			metadata := &templater_dto.InternalMetadata{
				CustomTags: []string{"my-card", "another-component"},
			}

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			var totalBytes int64
			for b.Loop() {
				cw := &CountingWriter{}
				err := orchestrator.RenderAST(ctx, cw, nil, request, render_domain.RenderASTOptions{PageID: "bench", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
				totalBytes += cw.BytesWritten
			}

			b.StopTimer()
			b.ReportMetric(float64(nodeCount), "nodes")
			if b.N > 0 {
				b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op-output")
			}
		})
	}
}

func BenchmarkRenderThroughput_FullPageVsFragment(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	request := NewTestRequest()
	ctx := context.Background()

	ast := BuildMixedAST(5)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title:       "Benchmark Page",
			Description: "A benchmark test page",
		},
		CustomTags: []string{"my-card", "another-component"},
	}

	WarmUpOrchestrator(orchestrator, ast)

	b.Run("FullPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		var totalBytes int64
		for b.Loop() {
			cw := &CountingWriter{}
			err := orchestrator.RenderAST(ctx, cw, nil, request, render_domain.RenderASTOptions{PageID: "bench-full", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
			totalBytes += cw.BytesWritten
		}

		b.StopTimer()
		if b.N > 0 {
			b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op-output")
		}
	})

	b.Run("Fragment", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		var totalBytes int64
		for b.Loop() {
			cw := &CountingWriter{}
			err := orchestrator.RenderAST(ctx, cw, nil, request, render_domain.RenderASTOptions{PageID: "bench-fragment", Template: ast, Metadata: metadata, IsFragment: true, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
			totalBytes += cw.BytesWritten
		}

		b.StopTimer()
		if b.N > 0 {
			b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op-output")
		}
	})
}

func BenchmarkRenderThroughput_FlatVsNested(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	request := NewTestRequest()
	ctx := context.Background()

	const nodeCount = 100

	b.Run("Flat", func(b *testing.B) {
		ast := BuildFlatAST(nodeCount)
		metadata := &templater_dto.InternalMetadata{}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench-flat", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Nested", func(b *testing.B) {
		ast := BuildGenericAST(nodeCount)
		metadata := &templater_dto.InternalMetadata{}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench-nested", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Deep", func(b *testing.B) {
		ast := BuildDeepAST(nodeCount)
		metadata := &templater_dto.InternalMetadata{}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench-deep", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRenderThroughput_WithStyling(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	request := NewTestRequest()
	ctx := context.Background()

	ast := BuildMixedAST(5)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "Styled Page",
		},
		CustomTags: []string{"my-card", "another-component"},
	}

	styling := `
		.container { display: flex; flex-direction: column; }
		.header { background: #333; color: white; padding: 1rem; }
		.main-content { flex: 1; padding: 2rem; }
		.footer { background: #f0f0f0; padding: 1rem; }
		.nav-link { color: #0066cc; text-decoration: none; }
		.nav-link:hover { text-decoration: underline; }
		.content-section { margin-bottom: 2rem; }
		.card-icon { width: 24px; height: 24px; }
	`

	WarmUpOrchestrator(orchestrator, ast)

	b.Run("WithoutStyling", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithStyling", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench", Template: ast, Metadata: metadata, Styling: styling, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRenderThroughput_RealWorld(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	request := NewTestRequest()
	ctx := context.Background()

	b.Run("SimpleLandingPage", func(b *testing.B) {
		ast := BuildMixedAST(3)
		metadata := &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{
				Title:       "Landing Page",
				Description: "Welcome to our site",
			},
			CustomTags: []string{"my-card"},
		}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "landing", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ProductListingPage", func(b *testing.B) {
		ast := BuildMixedAST(10)
		metadata := &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{
				Title:       "Products",
				Description: "Browse our products",
			},
			CustomTags: []string{"my-card", "another-component"},
		}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "products", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("DashboardPage", func(b *testing.B) {
		ast := BuildMixedAST(20)
		metadata := &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{
				Title:       "Dashboard",
				Description: "User dashboard",
			},
			CustomTags: []string{"my-card", "another-component", "custom-button"},
		}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "dashboard", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRenderThroughput_Concurrent(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildMixedAST(5)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "Concurrent Test",
		},
		CustomTags: []string{"my-card", "another-component"},
	}

	WarmUpOrchestrator(orchestrator, ast)

	b.Run("Sequential", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "seq", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "par", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}
