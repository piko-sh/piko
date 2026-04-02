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

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func BenchmarkTransformation_SVG(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	b.Run("Single", func(b *testing.B) {
		ast := BuildSVGHeavyAST(1)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "svg-single", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithClasses", func(b *testing.B) {
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "piko:svg",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "src", Value: "testmodule/lib/icon.svg"},
								{Name: "class", Value: "icon icon-primary icon-lg icon-animated"},
								{Name: "aria-hidden", Value: "true"},
								{Name: "data-testid", Value: "test-icon"},
							},
						},
					},
				},
			},
		}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "svg-classes", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithManyAttributes", func(b *testing.B) {
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "piko:svg",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "src", Value: "testmodule/lib/icon.svg"},
								{Name: "class", Value: "icon"},
								{Name: "width", Value: "24"},
								{Name: "height", Value: "24"},
								{Name: "aria-label", Value: "Icon"},
								{Name: "role", Value: "img"},
								{Name: "focusable", Value: "false"},
								{Name: "data-icon", Value: "test"},
								{Name: "data-size", Value: "medium"},
								{Name: "data-variant", Value: "primary"},
							},
						},
					},
				},
			},
		}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "svg-attrs", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CacheHitPath", func(b *testing.B) {

		ast := BuildRepeatedSVGAST(50, 1)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "svg-cached", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CacheMissPath", func(b *testing.B) {

		ast := BuildSVGHeavyAST(50)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "svg-uncached", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkTransformation_Link(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	b.Run("Single", func(b *testing.B) {
		ast := BuildLinkHeavyAST(1)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "link-single", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithAttributes", func(b *testing.B) {
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "nav",
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "piko:a",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "href", Value: "/about"},
								{Name: "class", Value: "nav-link primary active"},
								{Name: "aria-current", Value: "page"},
								{Name: "data-nav", Value: "main"},
							},
							Children: []*ast_domain.TemplateNode{
								{NodeType: ast_domain.NodeText, TextContent: "About Us"},
							},
						},
					},
				},
			},
		}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "link-attrs", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NavigationMenu", func(b *testing.B) {
		ast := BuildLinkHeavyAST(20)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "link-nav", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkTransformation_CSRF(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	b.Run("SingleForm", func(b *testing.B) {
		ast := BuildCSRFHeavyAST(1)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "csrf-single", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MultipleForms_SameToken", func(b *testing.B) {

		ast := BuildCSRFHeavyAST(10)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "csrf-multi", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NoCSRF", func(b *testing.B) {

		ast := BuildFormsWithoutCSRF(10)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "no-csrf", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkTransformation_CSRF_WithResponseWriter(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	b.Run("SingleForm_WithCSRF", func(b *testing.B) {
		ast := BuildCSRFHeavyAST(1)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			response := NewTestResponseWriter()
			err := orchestrator.RenderAST(ctx, io.Discard, response, request, render_domain.RenderASTOptions{PageID: "csrf-real-single", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MultipleForms_WithCSRF", func(b *testing.B) {

		ast := BuildCSRFHeavyAST(10)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			response := NewTestResponseWriter()
			err := orchestrator.RenderAST(ctx, io.Discard, response, request, render_domain.RenderASTOptions{PageID: "csrf-real-multi", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ManyForms_WithCSRF", func(b *testing.B) {

		ast := BuildCSRFHeavyAST(50)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			response := NewTestResponseWriter()
			err := orchestrator.RenderAST(ctx, io.Discard, response, request, render_domain.RenderASTOptions{PageID: "csrf-real-many", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NoCSRF_WithResponseWriter", func(b *testing.B) {

		ast := BuildFormsWithoutCSRF(10)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			response := NewTestResponseWriter()
			err := orchestrator.RenderAST(ctx, io.Discard, response, request, render_domain.RenderASTOptions{PageID: "no-csrf-rw", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCSRF_LazyVsEager(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}
	websiteConfig := &config.WebsiteConfig{}

	withCSRF := BuildCSRFHeavyAST(5)
	withoutCSRF := BuildFormsWithoutCSRF(5)

	WarmUpOrchestrator(orchestrator, withCSRF)
	WarmUpOrchestrator(orchestrator, withoutCSRF)

	b.Run("WithCSRF_WithResponseWriter", func(b *testing.B) {

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			response := NewTestResponseWriter()
			_ = orchestrator.RenderAST(ctx, io.Discard, response, request, render_domain.RenderASTOptions{
				PageID:     "lazy-csrf",
				Template:   withCSRF,
				Metadata:   metadata,
				SiteConfig: websiteConfig,
			})
		}
	})

	b.Run("WithCSRF_NilResponseWriter", func(b *testing.B) {

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			_ = orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{
				PageID:     "skip-csrf",
				Template:   withCSRF,
				Metadata:   metadata,
				SiteConfig: websiteConfig,
			})
		}
	})

	b.Run("NoCSRF_WithResponseWriter", func(b *testing.B) {

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			response := NewTestResponseWriter()
			_ = orchestrator.RenderAST(ctx, io.Discard, response, request, render_domain.RenderASTOptions{
				PageID:     "no-csrf-base",
				Template:   withoutCSRF,
				Metadata:   metadata,
				SiteConfig: websiteConfig,
			})
		}
	})

	b.Run("Parallel_WithCSRF", func(b *testing.B) {

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				request := NewTestRequest()
				response := NewTestResponseWriter()
				_ = orchestrator.RenderAST(ctx, io.Discard, response, request, render_domain.RenderASTOptions{
					PageID:     "parallel-csrf",
					Template:   withCSRF,
					Metadata:   metadata,
					SiteConfig: websiteConfig,
				})
			}
		})
	})
}

func BenchmarkTransformation_Events(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	b.Run("SingleEvent", func(b *testing.B) {
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "button",
					OnEvents: map[string][]ast_domain.Directive{
						"click": {{Type: ast_domain.DirectiveOn, RawExpression: "handleClick"}},
					},
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "Click"},
					},
				},
			},
		}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "event-single", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MultipleEventTypes", func(b *testing.B) {
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					OnEvents: map[string][]ast_domain.Directive{
						"click":      {{Type: ast_domain.DirectiveOn, RawExpression: "handleClick"}},
						"mouseover":  {{Type: ast_domain.DirectiveOn, RawExpression: "handleHover"}},
						"mouseout":   {{Type: ast_domain.DirectiveOn, RawExpression: "handleOut"}},
						"touchstart": {{Type: ast_domain.DirectiveOn, RawExpression: "handleTouch"}},
					},
					CustomEvents: map[string][]ast_domain.Directive{
						"custom-event": {{Type: ast_domain.DirectiveEvent, RawExpression: "handleCustom"}},
					},
					DirRef: &ast_domain.Directive{
						Type:          ast_domain.DirectiveRef,
						RawExpression: "myElement",
						Expression:    &ast_domain.StringLiteral{Value: "myElement"},
					},
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "Interactive Element"},
					},
				},
			},
		}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "event-multi", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("EventHeavyPage", func(b *testing.B) {
		ast := BuildEventHeavyAST(20)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "event-heavy", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkTransformation_Fragment(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()
	metadata := &templater_dto.InternalMetadata{}

	b.Run("SingleFragment", func(b *testing.B) {
		ast := BuildFragmentAST(1, 5)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "frag-single", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NestedFragments", func(b *testing.B) {
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeFragment,
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "outer-fragment"},
					},
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeFragment,
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "class", Value: "inner-fragment"},
							},
							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "span",
									Attributes: []ast_domain.HTMLAttribute{
										{Name: "class", Value: "element"},
									},
									Children: []*ast_domain.TemplateNode{
										{NodeType: ast_domain.NodeText, TextContent: "Nested content"},
									},
								},
							},
						},
					},
				},
			},
		}
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "frag-nested", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("FragmentWithManyChildren", func(b *testing.B) {
		ast := BuildFragmentAST(5, 20)
		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "frag-children", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkTransformation_MixedContent(b *testing.B) {
	orchestrator := NewTestOrchestrator()
	ctx := context.Background()

	b.Run("AllTransformations", func(b *testing.B) {

		ast := BuildMixedAST(10)
		metadata := &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{
				Title: "Mixed Transform Test",
			},
			CustomTags: []string{"my-card", "another-component"},
		}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "mixed-all", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NoTransformations", func(b *testing.B) {

		ast := BuildGenericAST(100)
		metadata := &templater_dto.InternalMetadata{}

		WarmUpOrchestrator(orchestrator, ast)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			request := NewTestRequest()
			err := orchestrator.RenderAST(ctx, io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "no-transform", Template: ast, Metadata: metadata, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BuildFormsWithoutCSRF(count int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, count)

	for range count {
		children = append(children, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "form",

			Attributes: []ast_domain.HTMLAttribute{
				{Name: "action", Value: "/search"},
				{Name: "method", Value: "GET"},
			},
			Children: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "input",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "type", Value: "text"},
						{Name: "name", Value: "q"},
					},
				},
			},
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "forms-no-csrf"}},
				Children:   children,
			},
		},
	}
}
