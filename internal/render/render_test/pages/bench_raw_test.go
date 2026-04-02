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

package pages_test

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/render/render_test/pages/fixtures"
	"piko.sh/piko/internal/render/render_test/pages/mocks"
	"piko.sh/piko/internal/templater/templater_dto"
)

func BenchmarkRenderOrchestrator_Raw(b *testing.B) {
	mockRegistry := mocks.NewMockRegistry(b)
	mockCSRF := mocks.NewMockCSRF()
	orchestrator := render_domain.NewRenderOrchestrator(nil, nil, mockRegistry, mockCSRF)
	request := httptest.NewRequest("GET", "/", nil)

	mockRegistry.OnGetComponent("my-card", &render_dto.ComponentMetadata{TagName: "my-card", BaseJSPath: "/dist/my-card.js"})
	mockRegistry.OnGetComponent("another-component", &render_dto.ComponentMetadata{TagName: "another-component", BaseJSPath: "/dist/another.js"})
	mockRegistry.OnGetSVG("testmodule/lib/icon.svg", &render_domain.ParsedSvgData{
		InnerHTML: "<path d='...'></path>",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "viewBox", Value: "0 0 24 24"},
		},
	})
	mockRegistry.OnGetSVG("testmodule/lib/logo.svg", &render_domain.ParsedSvgData{
		InnerHTML: `<path d="logo-path"></path>`,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "viewBox", Value: "0 0 100 40"},
		},
	})

	simpleAST := fixtures.SimplePageAST()
	metadataSimple := templater_dto.InternalMetadata{}

	complexAST := fixtures.ComplexPageAST()
	metadataComplex := templater_dto.InternalMetadata{
		CustomTags: []string{"my-card", "another-component"},
	}

	megaComplexAST := fixtures.MegaComplexPageAST()
	metadataMega := templater_dto.InternalMetadata{
		CustomTags: []string{"my-card", "another-component"},
	}

	b.Run("SimpleFullPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			err := orchestrator.RenderAST(context.Background(), io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench-simple", Template: simpleAST, Metadata: &metadataSimple, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ComplexFullPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			err := orchestrator.RenderAST(context.Background(), io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench-complex", Template: complexAST, Metadata: &metadataComplex, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ComplexFragmentPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			err := orchestrator.RenderAST(context.Background(), io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench-complex", Template: complexAST, Metadata: &metadataComplex, IsFragment: true, Styling: "", SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MegaComplexFullPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			err := orchestrator.RenderAST(context.Background(), io.Discard, nil, request, render_domain.RenderASTOptions{PageID: "bench-mega", Template: megaComplexAST, Metadata: &metadataMega, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
