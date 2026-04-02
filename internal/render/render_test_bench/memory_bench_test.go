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
	"runtime"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func BenchmarkMemory_Allocations(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	testCases := []struct {
		ast  func() *ast_domain.TemplateAST
		meta *templater_dto.InternalMetadata
		name string
	}{
		{
			name: "MinimalAST",
			ast:  func() *ast_domain.TemplateAST { return BuildGenericAST(5) },
			meta: &templater_dto.InternalMetadata{},
		},
		{
			name: "StandardPage",
			ast:  func() *ast_domain.TemplateAST { return BuildMixedAST(5) },
			meta: &templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{
					Title: "Standard Page",
				},
				CustomTags: []string{"my-card", "another-component"},
			},
		},
		{
			name: "SVGHeavy",
			ast:  func() *ast_domain.TemplateAST { return BuildSVGHeavyAST(20) },
			meta: &templater_dto.InternalMetadata{},
		},
		{
			name: "CSRFHeavy",
			ast:  func() *ast_domain.TemplateAST { return BuildCSRFHeavyAST(10) },
			meta: &templater_dto.InternalMetadata{},
		},
		{
			name: "AttributeHeavy",
			ast:  func() *ast_domain.TemplateAST { return BuildAttributeHeavyAST(20, 15) },
			meta: &templater_dto.InternalMetadata{},
		},
		{
			name: "EventHeavy",
			ast:  func() *ast_domain.TemplateAST { return BuildEventHeavyAST(20) },
			meta: &templater_dto.InternalMetadata{},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			ast := tc.ast()
			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "mem-test", Template: ast, Metadata: tc.meta, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMemory_PerNode(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	sizes := []int{10, 50, 100, 200, 500}

	for _, size := range sizes {
		b.Run(sizeLabel(size), func(b *testing.B) {
			ast := BuildGenericAST(size)
			nodeCount := CountNodes(ast)
			metadata := &templater_dto.InternalMetadata{}

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "per-node", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(nodeCount), "nodes")
		})
	}
}

func BenchmarkMemory_PoolEfficiency(b *testing.B) {
	ctx := context.Background()

	ast := BuildMixedAST(10)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "Pool Test",
		},
		CustomTags: []string{"my-card", "another-component"},
	}

	b.Run("Fresh", func(b *testing.B) {

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			orchestrator := NewTestOrchestrator()
			request := NewTestRequest()

			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "fresh", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Warmed", func(b *testing.B) {
		orchestrator := NewTestOrchestrator()
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "warmed", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkMemory_GCPressure(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	ast := BuildMixedAST(20)
	metadata := &templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "GC Pressure Test",
		},
		CustomTags: []string{"my-card", "another-component"},
	}

	WarmUpOrchestrator(orchestrator, ast)

	b.Run("WithGCStats", func(b *testing.B) {
		var memStatsBefore, memStatsAfter runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&memStatsBefore)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "gc-test", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}

		b.StopTimer()
		runtime.ReadMemStats(&memStatsAfter)

		if b.N > 0 {
			gcCycles := memStatsAfter.NumGC - memStatsBefore.NumGC
			b.ReportMetric(float64(gcCycles)/float64(b.N), "gc-cycles/op")

			totalAlloc := memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc
			b.ReportMetric(float64(totalAlloc)/float64(b.N), "total-alloc-bytes/op")
		}
	})
}

func BenchmarkMemory_SVGCaching(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	b.Run("UniqueSVGs", func(b *testing.B) {

		ast := BuildSVGHeavyAST(20)
		metadata := &templater_dto.InternalMetadata{}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "unique-svg", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RepeatedSVGs", func(b *testing.B) {

		ast := BuildRepeatedSVGAST(20, 5)
		metadata := &templater_dto.InternalMetadata{}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "repeated-svg", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkMemory_FragmentAttrs(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	b.Run("NoFragments", func(b *testing.B) {
		ast := BuildGenericAST(50)
		metadata := &templater_dto.InternalMetadata{}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "no-frag", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithFragments", func(b *testing.B) {
		ast := BuildFragmentAST(10, 5)
		metadata := &templater_dto.InternalMetadata{}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "with-frag", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkMemory_DeepCloneOverhead(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(sizeLabel(size), func(b *testing.B) {
			ast := BuildMixedAST(size / 10)
			nodeCount := CountNodes(ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_ = ast
			}

			b.StopTimer()
			b.ReportMetric(float64(nodeCount), "nodes")
		})
	}
}

func BuildRepeatedSVGAST(totalCount, uniqueCount int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, totalCount)

	icons := make([]string, uniqueCount)
	for i := range uniqueCount {
		icons[i] = "testmodule/lib/icon.svg"
	}

	for i := range totalCount {
		icon := icons[i%uniqueCount]
		children = append(children, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:svg",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: icon},
				{Name: "class", Value: "repeated-icon"},
			},
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "repeated-svg-container"}},
				Children:   children,
			},
		},
	}
}

func sizeLabel(size int) string {
	switch {
	case size <= 10:
		return "Nodes10"
	case size <= 50:
		return "Nodes50"
	case size <= 100:
		return "Nodes100"
	case size <= 200:
		return "Nodes200"
	default:
		return "Nodes500"
	}
}
