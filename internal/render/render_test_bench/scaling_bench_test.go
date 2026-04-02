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

func BenchmarkScaling_NodeCount(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	nodeCounts := []int{10, 25, 50, 100, 200, 500, 1000}

	for _, count := range nodeCounts {
		b.Run(nodeCountLabel(count), func(b *testing.B) {
			ast := BuildGenericAST(count)
			actualCount := CountNodes(ast)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-nodes", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(actualCount), "nodes")
		})
	}
}

func BenchmarkScaling_NestingDepth(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	depths := []int{5, 10, 20, 50, 100, 200}

	for _, depth := range depths {
		b.Run(depthLabel(depth), func(b *testing.B) {
			ast := BuildDeepAST(depth)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-depth", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(depth), "depth")
		})
	}
}

func BenchmarkScaling_Width(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	widths := []int{10, 25, 50, 100, 200, 500}

	for _, width := range widths {
		b.Run(widthLabel(width), func(b *testing.B) {
			ast := BuildFlatAST(width)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-width", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(width), "siblings")
		})
	}
}

func BenchmarkScaling_SVGCount(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	svgCounts := []int{1, 5, 10, 25, 50, 100}

	for _, count := range svgCounts {
		b.Run(svgCountLabel(count), func(b *testing.B) {
			ast := BuildSVGHeavyAST(count)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-svg", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(count), "svg-count")
		})
	}
}

func BenchmarkScaling_AttributeCount(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	const nodeCount = 50
	attributeCounts := []int{1, 3, 5, 10, 15, 20}

	for _, attributeCount := range attributeCounts {
		b.Run(attributeCountLabel(attributeCount), func(b *testing.B) {
			ast := BuildAttributeHeavyAST(nodeCount, attributeCount)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-attrs", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(attributeCount), "attrs-per-node")
		})
	}
}

func BenchmarkScaling_LinkCount(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	linkCounts := []int{5, 10, 25, 50, 100}

	for _, count := range linkCounts {
		b.Run(linkCountLabel(count), func(b *testing.B) {
			ast := BuildLinkHeavyAST(count)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-links", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(count), "link-count")
		})
	}
}

func BenchmarkScaling_FormCount(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	formCounts := []int{1, 5, 10, 25, 50}

	for _, count := range formCounts {
		b.Run(formCountLabel(count), func(b *testing.B) {
			ast := BuildCSRFHeavyAST(count)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-forms", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(count), "form-count")
		})
	}
}

func BenchmarkScaling_EventHandlerCount(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	eventCounts := []int{5, 10, 25, 50, 100}

	for _, count := range eventCounts {
		b.Run(eventCountLabel(count), func(b *testing.B) {
			ast := BuildEventHeavyAST(count)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-events", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(count), "event-elements")
		})
	}
}

func BenchmarkScaling_CustomComponentCount(b *testing.B) {
	ctx := context.Background()

	componentCounts := []int{1, 5, 10, 20}

	for _, count := range componentCounts {
		b.Run(componentCountLabel(count), func(b *testing.B) {
			ast := BuildMixedAST(count)
			tags := make([]string, count)
			for i := range count {
				tags[i] = "my-card"
			}
			metadata := &templater_dto.InternalMetadata{
				CustomTags: tags,
			}

			orchestrator := NewTestOrchestrator()
			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-components", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(count), "component-types")
		})
	}
}

func BenchmarkScaling_FragmentCount(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	fragmentCounts := []int{5, 10, 25, 50}
	const childrenPerFragment = 3

	for _, count := range fragmentCounts {
		b.Run(fragmentCountLabel(count), func(b *testing.B) {
			ast := BuildFragmentAST(count, childrenPerFragment)

			WarmUpOrchestrator(orchestrator, ast)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				request := NewTestRequest()
				err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "scale-fragments", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(count), "fragments")
		})
	}
}

func nodeCountLabel(count int) string {
	switch {
	case count <= 10:
		return "N10"
	case count <= 25:
		return "N25"
	case count <= 50:
		return "N50"
	case count <= 100:
		return "N100"
	case count <= 200:
		return "N200"
	case count <= 500:
		return "N500"
	default:
		return "N1000"
	}
}

func depthLabel(depth int) string {
	switch {
	case depth <= 5:
		return "D5"
	case depth <= 10:
		return "D10"
	case depth <= 20:
		return "D20"
	case depth <= 50:
		return "D50"
	case depth <= 100:
		return "D100"
	default:
		return "D200"
	}
}

func widthLabel(width int) string {
	switch {
	case width <= 10:
		return "W10"
	case width <= 25:
		return "W25"
	case width <= 50:
		return "W50"
	case width <= 100:
		return "W100"
	case width <= 200:
		return "W200"
	default:
		return "W500"
	}
}

func svgCountLabel(count int) string {
	switch {
	case count <= 1:
		return "SVG1"
	case count <= 5:
		return "SVG5"
	case count <= 10:
		return "SVG10"
	case count <= 25:
		return "SVG25"
	case count <= 50:
		return "SVG50"
	default:
		return "SVG100"
	}
}

func attributeCountLabel(count int) string {
	switch {
	case count <= 1:
		return "Attr1"
	case count <= 3:
		return "Attr3"
	case count <= 5:
		return "Attr5"
	case count <= 10:
		return "Attr10"
	case count <= 15:
		return "Attr15"
	default:
		return "Attr20"
	}
}

func linkCountLabel(count int) string {
	switch {
	case count <= 5:
		return "Links5"
	case count <= 10:
		return "Links10"
	case count <= 25:
		return "Links25"
	case count <= 50:
		return "Links50"
	default:
		return "Links100"
	}
}

func formCountLabel(count int) string {
	switch {
	case count <= 1:
		return "Forms1"
	case count <= 5:
		return "Forms5"
	case count <= 10:
		return "Forms10"
	case count <= 25:
		return "Forms25"
	default:
		return "Forms50"
	}
}

func eventCountLabel(count int) string {
	switch {
	case count <= 5:
		return "Events5"
	case count <= 10:
		return "Events10"
	case count <= 25:
		return "Events25"
	case count <= 50:
		return "Events50"
	default:
		return "Events100"
	}
}

func componentCountLabel(count int) string {
	switch {
	case count <= 1:
		return "Comp1"
	case count <= 5:
		return "Comp5"
	case count <= 10:
		return "Comp10"
	default:
		return "Comp20"
	}
}

func fragmentCountLabel(count int) string {
	switch {
	case count <= 5:
		return "Frag5"
	case count <= 10:
		return "Frag10"
	case count <= 25:
		return "Frag25"
	default:
		return "Frag50"
	}
}
