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
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func BenchmarkStreaming_TTFB(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	sizes := []FixtureSize{SizeSmall, SizeMedium, SizeLarge}

	for _, size := range sizes {
		b.Run(size.String(), func(b *testing.B) {
			ast := BuildMixedAST(size.NodeCount() / 10)
			metadata := &templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{
					Title: "TTFB Test",
				},
				CustomTags: []string{"my-card", "another-component"},
			}

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			var totalTTFB time.Duration
			var totalHeadTime time.Duration
			var totalBytes int64

			for b.Loop() {
				b.StopTimer()
				mw := NewMetricsWriter(io.Discard)
				request := NewTestRequest()
				b.StartTimer()

				err := orchestrator.RenderAST(ctx, mw, nil, request, render_domain.RenderASTOptions{PageID: "ttfb-test", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}

				b.StopTimer()
				mw.Finish()
				metrics := mw.Metrics()
				totalTTFB += metrics.TTFB
				totalHeadTime += metrics.HeadTime
				totalBytes += metrics.BytesWritten
				b.StartTimer()
			}

			b.StopTimer()
			if b.N > 0 {
				b.ReportMetric(float64(totalTTFB.Nanoseconds())/float64(b.N), "ttfb-ns/op")
				b.ReportMetric(float64(totalHeadTime.Nanoseconds())/float64(b.N), "head-ns/op")
				b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
			}
		})
	}
}

func BenchmarkStreaming_FullVsFragment(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildMixedAST(10)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title:       "Streaming Test",
			Description: "Testing streaming performance",
		},
		CustomTags: []string{"my-card", "another-component"},
	}

	WarmUpOrchestrator(orchestrator, ast)

	b.Run("FullPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		var totalTTFB time.Duration
		var totalBytes int64
		var totalWriteCount int

		for b.Loop() {
			b.StopTimer()
			mw := NewMetricsWriter(io.Discard)
			request := NewTestRequest()
			b.StartTimer()

			err := orchestrator.RenderAST(ctx, mw, nil, request, render_domain.RenderASTOptions{PageID: "full", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}

			b.StopTimer()
			mw.Finish()
			metrics := mw.Metrics()
			totalTTFB += metrics.TTFB
			totalBytes += metrics.BytesWritten
			totalWriteCount += metrics.WriteCount
			b.StartTimer()
		}

		b.StopTimer()
		if b.N > 0 {
			b.ReportMetric(float64(totalTTFB.Nanoseconds())/float64(b.N), "ttfb-ns/op")
			b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
			b.ReportMetric(float64(totalWriteCount)/float64(b.N), "writes/op")
		}
	})

	b.Run("Fragment", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		var totalTTFB time.Duration
		var totalBytes int64
		var totalWriteCount int

		for b.Loop() {
			b.StopTimer()
			mw := NewMetricsWriter(io.Discard)
			request := NewTestRequest()
			b.StartTimer()

			err := orchestrator.RenderAST(ctx, mw, nil, request, render_domain.RenderASTOptions{PageID: "fragment", Template: ast, Metadata: metadata, IsFragment: true, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}

			b.StopTimer()
			mw.Finish()
			metrics := mw.Metrics()
			totalTTFB += metrics.TTFB
			totalBytes += metrics.BytesWritten
			totalWriteCount += metrics.WriteCount
			b.StartTimer()
		}

		b.StopTimer()
		if b.N > 0 {
			b.ReportMetric(float64(totalTTFB.Nanoseconds())/float64(b.N), "ttfb-ns/op")
			b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
			b.ReportMetric(float64(totalWriteCount)/float64(b.N), "writes/op")
		}
	})
}

func BenchmarkStreaming_OutputSize(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	scales := []int{1, 5, 10, 20, 50}

	for _, scale := range scales {
		b.Run(scaleLabel(scale), func(b *testing.B) {
			ast := BuildMixedAST(scale)
			nodeCount := CountNodes(ast)
			metadata := &templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{
					Title: "Output Size Test",
				},
				CustomTags: []string{"my-card", "another-component"},
			}

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			var totalBytes int64

			for b.Loop() {
				cw := &CountingWriter{}
				request := NewTestRequest()

				err := orchestrator.RenderAST(ctx, cw, nil, request, render_domain.RenderASTOptions{PageID: "output-test", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
				totalBytes += cw.BytesWritten
			}

			b.StopTimer()
			if b.N > 0 {
				avgBytes := float64(totalBytes) / float64(b.N)
				b.ReportMetric(float64(nodeCount), "nodes")
				b.ReportMetric(avgBytes, "bytes/op")
				b.ReportMetric(avgBytes/float64(nodeCount), "bytes/node")
			}
		})
	}
}

func BenchmarkStreaming_WritePatterns(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	testCases := []struct {
		ast  func() *ast_domain.TemplateAST
		name string
	}{
		{name: "TextHeavy", ast: func() *ast_domain.TemplateAST { return BuildGenericAST(100) }},
		{name: "SVGHeavy", ast: func() *ast_domain.TemplateAST { return BuildSVGHeavyAST(50) }},
		{name: "AttributeHeavy", ast: func() *ast_domain.TemplateAST { return BuildAttributeHeavyAST(50, 10) }},
		{name: "CommentHeavy", ast: func() *ast_domain.TemplateAST { return BuildCommentHeavyAST(50) }},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			ast := tc.ast()
			metadata := &templater_dto.InternalMetadata{}

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			var totalBytes int64
			var totalWriteCount int

			for b.Loop() {
				cw := &CountingWriter{}
				request := NewTestRequest()

				err := orchestrator.RenderAST(ctx, cw, nil, request, render_domain.RenderASTOptions{PageID: "pattern-test", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
				totalBytes += cw.BytesWritten
				totalWriteCount += cw.WriteCount
			}

			b.StopTimer()
			if b.N > 0 {
				b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
				b.ReportMetric(float64(totalWriteCount)/float64(b.N), "writes/op")
				b.ReportMetric(float64(totalBytes)/float64(totalWriteCount), "bytes/write")
			}
		})
	}
}

func BenchmarkStreaming_BufferedVsDirect(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildMixedAST(10)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "Buffer Test",
		},
		CustomTags: []string{"my-card", "another-component"},
	}

	WarmUpOrchestrator(orchestrator, ast)

	b.Run("Discard", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "discard", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Buffer", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var buffer bytes.Buffer
			buffer.Grow(64 * 1024)
			request := NewTestRequest()

			err := orchestrator.RenderAST(ctx, &buffer, nil, request, render_domain.RenderASTOptions{PageID: "buffer", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CountingWriter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			cw := &CountingWriter{}
			request := NewTestRequest()

			err := orchestrator.RenderAST(ctx, cw, nil, request, render_domain.RenderASTOptions{PageID: "counting", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func scaleLabel(scale int) string {
	switch {
	case scale <= 1:
		return "Scale1"
	case scale <= 5:
		return "Scale5"
	case scale <= 10:
		return "Scale10"
	case scale <= 20:
		return "Scale20"
	default:
		return "Scale50"
	}
}
