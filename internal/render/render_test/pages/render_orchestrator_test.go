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

package pages_test

import (
	"bytes"
	"context"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/render/render_test/pages/fixtures"
	"piko.sh/piko/internal/render/render_test/pages/mocks"
	"piko.sh/piko/internal/templater/templater_dto"
)

var update = flag.Bool("update", false, "update golden files")

func setupTest(t testing.TB) (*render_domain.RenderOrchestrator, *mocks.MockRegistry, *http.Request) {
	render_domain.ClearSpriteSheetCacheForTesting()
	mockRegistry := mocks.NewMockRegistry(t)
	mockCSRF := mocks.NewMockCSRF()
	orchestrator := render_domain.NewRenderOrchestrator(nil, nil, mockRegistry, mockCSRF)
	request := httptest.NewRequest("GET", "/", nil)
	return orchestrator, mockRegistry, request
}

func assertGolden(t *testing.T, goldenFile string, actual []byte) {
	t.Helper()
	if *update {
		t.Logf("Updating golden file: %s", goldenFile)
		require.NoError(t, os.WriteFile(goldenFile, actual, 0644), "failed to update golden file")
	}

	expected, err := os.ReadFile(goldenFile)
	require.NoError(t, err, "failed to read golden file")

	assert.Equal(t, string(expected), string(actual), "output does not match golden file: %s", goldenFile)
}

func TestRenderOrchestrator_GoldenFiles(t *testing.T) {
	orchestrator, mockRegistry, request := setupTest(t)

	t.Run("should render simple page correctly against golden file", func(t *testing.T) {
		ast := fixtures.SimplePageAST()
		var buffer bytes.Buffer
		err := orchestrator.RenderAST(context.Background(), &buffer, httptest.NewRecorder(), request, render_domain.RenderASTOptions{PageID: "simple-page", Template: ast, Metadata: &templater_dto.InternalMetadata{}, SiteConfig: &config.WebsiteConfig{}})
		require.NoError(t, err)

		assertGolden(t, filepath.Join("golden", "simple_page.golden.html"), buffer.Bytes())
	})

	t.Run("should render complex page correctly against golden file", func(t *testing.T) {
		mockRegistry.OnGetComponent("my-card", &render_dto.ComponentMetadata{TagName: "my-card", BaseJSPath: "/dist/my-card.js"})
		mockRegistry.OnGetComponent("another-component", &render_dto.ComponentMetadata{TagName: "another-component", BaseJSPath: "/dist/another.js"})
		mockRegistry.OnGetSVG("testmodule/lib/icon.svg", &render_domain.ParsedSvgData{
			InnerHTML:  `<path d="..."></path>`,
			Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
		})

		ast := fixtures.ComplexPageAST()
		metadata := templater_dto.InternalMetadata{
			Metadata:   templater_dto.Metadata{Title: "Golden Test"},
			CustomTags: []string{"my-card", "another-component"},
		}

		var buffer bytes.Buffer
		err := orchestrator.RenderAST(context.Background(), &buffer, httptest.NewRecorder(), request, render_domain.RenderASTOptions{PageID: "golden-page", Template: ast, Metadata: &metadata, IsFragment: false, Styling: "body { color: red; }", SiteConfig: &config.WebsiteConfig{}})
		require.NoError(t, err)

		assertGolden(t, filepath.Join("golden", "complex_page.golden.html"), buffer.Bytes())
	})

	t.Run("should render mega complex page correctly against golden file", func(t *testing.T) {
		mockRegistry.OnGetComponent("my-card", &render_dto.ComponentMetadata{TagName: "my-card", BaseJSPath: "/dist/my-card.js"})
		mockRegistry.OnGetComponent("another-component", &render_dto.ComponentMetadata{TagName: "another-component", BaseJSPath: "/dist/another.js"})
		mockRegistry.OnGetSVG("logo.svg", &render_domain.ParsedSvgData{
			InnerHTML:  `<path d="logo-path"></path>`,
			Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 100 40"}},
		})

		ast := fixtures.MegaComplexPageAST()
		metadata := templater_dto.InternalMetadata{
			Metadata:   templater_dto.Metadata{Title: "Mega Complex Test"},
			CustomTags: []string{"my-card", "another-component"},
		}

		var buffer bytes.Buffer
		err := orchestrator.RenderAST(context.Background(), &buffer, httptest.NewRecorder(), request, render_domain.RenderASTOptions{PageID: "mega-golden-page", Template: ast, Metadata: &metadata, SiteConfig: &config.WebsiteConfig{}})
		require.NoError(t, err)

		assertGolden(t, filepath.Join("golden", "mega_complex_page.golden.html"), buffer.Bytes())
	})
}

func TestRenderOrchestrator_Unit(t *testing.T) {
	testCases := []struct {
		node                 *ast_domain.TemplateNode
		setupMock            func(m *mocks.MockRegistry)
		name                 string
		expectedToContain    []string
		expectedNotToContain []string
		metadata             templater_dto.InternalMetadata
		isFragment           bool
	}{
		{
			name:              "simple text node",
			node:              &ast_domain.TemplateNode{NodeType: ast_domain.NodeText, TextContent: "Hello World"},
			isFragment:        true,
			expectedToContain: []string{"Hello World"},
		},
		{
			name: "simple element with attributes",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "id", Value: "test-id"},
					{Name: "class", Value: "test-class"},
				},
			},
			isFragment:        true,
			expectedToContain: []string{`<div id="test-id" class="test-class"></div>`},
		},
		{
			name:                 "void element (img) is self-closing",
			node:                 &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "img", Attributes: []ast_domain.HTMLAttribute{{Name: "src", Value: "image.png"}}},
			isFragment:           true,
			expectedToContain:    []string{`<img src="image.png" />`},
			expectedNotToContain: []string{`</img>`},
		},
		{
			name:              "innerHTML is rendered raw",
			node:              &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div", InnerHTML: "<strong>Raw HTML</strong>"},
			isFragment:        true,
			expectedToContain: []string{`<div><strong>Raw HTML</strong></div>`},
		},
		{
			name:              "TextContent is rendered (already escaped by generator)",
			node:              &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div", TextContent: "&lt;strong&gt;Escaped&lt;/strong&gt;"},
			isFragment:        true,
			expectedToContain: []string{`<div>&lt;strong&gt;Escaped&lt;/strong&gt;</div>`},
		},
		{
			name: "NeedsCSRF flag injects CSRF tokens into form",
			node: &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "form", RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
				NeedsCSRF: true,
			}},
			isFragment: true,
			expectedToContain: []string{
				`data-csrf-ephemeral-token="mock-ephemeral-token"`,
				`data-csrf-action-token="mock-action-token-payload^mock-signature"`,
			},
		},
		{
			name: "piko:a is transformed to a tag",
			node: &ast_domain.TemplateNode{
				NodeType:   ast_domain.NodeElement,
				TagName:    "piko:a",
				Attributes: []ast_domain.HTMLAttribute{{Name: "href", Value: "/test"}},
			},
			isFragment:        true,
			expectedToContain: []string{`<a href="/test" piko:a=""></a>`},
		},
		{
			name: "piko:svg is transformed and sprite sheet is generated",
			node: fixtures.SvgComponentNode(),
			setupMock: func(m *mocks.MockRegistry) {
				m.OnGetSVG("testmodule/lib/icon.svg", &render_domain.ParsedSvgData{
					InnerHTML:  `<path d="..."></path>`,
					Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
				})
			},
			isFragment: true,
			expectedToContain: []string{
				`<svg class="icon" viewBox="0 0 24 24"><use href="#testmodule/lib/icon.svg"></use></svg>`,
				`<symbol id="testmodule/lib/icon.svg" viewBox="0 0 24 24"><path d="..."></path></symbol>`,
			},
		},
		{
			name: "piko:svg merges attributes, user attributes take precedence",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:svg",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "testmodule/lib/icon.svg"},
					{Name: "class", Value: "user-class"},
					{Name: "fill", Value: "currentColor"},
				},
			},
			setupMock: func(m *mocks.MockRegistry) {
				m.OnGetSVG("testmodule/lib/icon.svg", &render_domain.ParsedSvgData{
					InnerHTML: `<path d="..."></path>`,
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "viewBox", Value: "0 0 24 24"},
						{Name: "class", Value: "file-class"},
					},
				})
			},
			isFragment: true,

			expectedToContain: []string{
				`<svg class="file-class user-class" viewBox="0 0 24 24" fill="currentColor"><use href="#testmodule/lib/icon.svg"></use></svg>`,
			},
		},
		{
			name:                 "Fragment render includes only essential tags",
			node:                 &ast_domain.TemplateNode{NodeType: ast_domain.NodeText, TextContent: "Fragment"},
			isFragment:           true,
			expectedToContain:    []string{"<head>", "<body>", "Fragment", "</body>"},
			expectedNotToContain: []string{"<!DOCTYPE html>", "<html>"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			orchestrator, mockRegistry, request := setupTest(t)

			if tc.setupMock != nil {
				tc.setupMock(mockRegistry)
			}

			var vast *ast_domain.TemplateAST
			if tc.node != nil {
				vast = &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{tc.node}}
			} else {
				vast = &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{}}
			}

			var buffer bytes.Buffer
			err := orchestrator.RenderAST(context.Background(), &buffer, httptest.NewRecorder(), request, render_domain.RenderASTOptions{PageID: "unit-test", Template: vast, Metadata: &tc.metadata, IsFragment: tc.isFragment, Styling: "", SiteConfig: &config.WebsiteConfig{}})
			require.NoError(t, err)

			output := buffer.String()
			for _, expected := range tc.expectedToContain {
				assert.Contains(t, output, expected)
			}
			for _, notExpected := range tc.expectedNotToContain {
				assert.NotContains(t, output, notExpected)
			}
		})
	}
}

func TestRenderOrchestrator_Metadata(t *testing.T) {
	orchestrator, mockRegistry, request := setupTest(t)

	t.Run("should collect link headers for components and fonts", func(t *testing.T) {
		mockRegistry.OnGetComponent("comp-a", &render_dto.ComponentMetadata{BaseJSPath: "/dist/comp-a.js"})
		mockRegistry.OnGetComponent("comp-b", &render_dto.ComponentMetadata{BaseJSPath: "/dist/comp-b.js"})

		metadata := templater_dto.InternalMetadata{
			CustomTags: []string{"comp-a", "comp-b"},
		}
		siteConfig := &config.WebsiteConfig{
			Fonts: []config.FontDefinition{
				{URL: "https://fonts.googleapis.com/css?family=Roboto", Instant: true},
				{URL: "/fonts/custom.woff2", Instant: false},
			},
		}

		headers, _, err := orchestrator.CollectMetadata(context.Background(), request, &metadata, siteConfig)
		require.NoError(t, err)

		headerMap := make(map[string]render_dto.LinkHeader)
		for _, h := range headers {
			headerMap[h.URL] = h
		}

		assert.Len(t, headers, 9)

		assert.Equal(t, "modulepreload", headerMap["/dist/comp-a.js"].Rel)
		assert.Equal(t, "modulepreload", headerMap["/dist/comp-b.js"].Rel)
		assert.Equal(t, "preload", headerMap["/_piko/dist/ppframework.core.es.js"].Rel)
		assert.Equal(t, "preload", headerMap["/theme.css"].Rel)
		assert.Equal(t, "style", headerMap["/theme.css"].As)
		assert.Equal(t, "modulepreload", headerMap["/_piko/dist/ppframework.components.es.js"].Rel)
		assert.Equal(t, "preconnect", headerMap["https://fonts.googleapis.com"].Rel)
		assert.Equal(t, "preconnect", headerMap["https://fonts.gstatic.com"].Rel)
		assert.Equal(t, "preload", headerMap["https://fonts.googleapis.com/css?family=Roboto"].Rel)
		assert.Equal(t, "preload", headerMap["/fonts/custom.woff2"].Rel)
		assert.Equal(t, "font", headerMap["/fonts/custom.woff2"].As)
	})
}

func TestRenderOrchestrator_Errors(t *testing.T) {
	orchestrator, _, request := setupTest(t)

	t.Run("should not fail render if component metadata is missing", func(t *testing.T) {
		ast := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "missing-comp"},
		}}
		metadata := templater_dto.InternalMetadata{CustomTags: []string{"missing-comp"}}
		var buffer bytes.Buffer

		err := orchestrator.RenderAST(context.Background(), &buffer, httptest.NewRecorder(), request, render_domain.RenderASTOptions{PageID: "test-error", Template: ast, Metadata: &metadata, IsFragment: false, Styling: "", SiteConfig: nil})
		require.NoError(t, err)

		assert.Contains(t, buffer.String(), "<missing-comp></missing-comp>")
	})
}
