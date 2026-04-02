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

package render_domain

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

func newTestRenderContext(_ RegistryPort) *renderContext {

	logger_domain.GetLogger("test")

	return &renderContext{
		originalCtx:        context.Background(),
		requiredSvgSymbols: make([]svgSymbolEntry, 0),
	}
}

func testRenderOrchestrator() *RenderOrchestrator {
	return &RenderOrchestrator{}
}

func renderToString(
	t *testing.T,
	node *ast_domain.TemplateNode,
	rctx *renderContext,
	renderFunction func(*RenderOrchestrator, *ast_domain.TemplateNode, *qt.Writer, *renderContext) error,
) string {
	t.Helper()
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	ro := testRenderOrchestrator()
	err := renderFunction(ro, node, qw, rctx)
	require.NoError(t, err)

	return buffer.String()
}

func TestRenderPikoA(t *testing.T) {
	testCases := []struct {
		name           string
		inputNode      *ast_domain.TemplateNode
		expectedOutput string
	}{
		{
			name: "Basic piko:a with href",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/home"},
				},
			},
			expectedOutput: `<a href="/home" piko:a=""></a>`,
		},
		{
			name: "piko:a with additional attributes",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/about"},
					{Name: "class", Value: "nav-item"},
					{Name: "id", Value: "about-link"},
				},
			},
			expectedOutput: `<a class="nav-item" id="about-link" href="/about" piko:a=""></a>`,
		},
		{
			name: "piko:a with href attribute",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/contact"},
				},
			},
			expectedOutput: `<a href="/contact" piko:a=""></a>`,
		},
		{
			name: "piko:a with no href attribute",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "disabled"},
				},
			},
			expectedOutput: `<a class="disabled" href="" piko:a=""></a>`,
		},
		{
			name: "piko:a with children renders children",
			inputNode: &ast_domain.TemplateNode{
				TagName:    "piko:a",
				Attributes: []ast_domain.HTMLAttribute{{Name: "href", Value: "/"}},
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Go Home"},
				},
			},
			expectedOutput: `<a href="/" piko:a="">Go Home</a>`,
		},
		{
			name: "Empty piko:a tag",
			inputNode: &ast_domain.TemplateNode{
				TagName:    "piko:a",
				Attributes: []ast_domain.HTMLAttribute{},
			},
			expectedOutput: `<a href="" piko:a=""></a>`,
		},
		{
			name: "piko:a removes lang attribute from output",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/page"},
					{Name: "lang", Value: "fr"},
					{Name: "class", Value: "link"},
				},
			},

			expectedOutput: `<a class="link" href="/page" piko:a=""></a>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := newTestRenderContext(nil)
			output := renderToString(t, tc.inputNode, rctx, renderPikoA)
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}

func TestRenderPikoA_WithI18n(t *testing.T) {
	testCases := []struct {
		name           string
		inputNode      *ast_domain.TemplateNode
		i18nStrategy   string
		currentLocale  string
		defaultLocale  string
		expectedOutput string
	}{
		{
			name: "prefix strategy adds locale prefix",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/about"},
				},
			},
			i18nStrategy:   "prefix",
			currentLocale:  "fr",
			expectedOutput: `<a href="/fr/about" piko:a=""></a>`,
		},
		{
			name: "prefix_except_default skips default locale",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/about"},
				},
			},
			i18nStrategy:   "prefix_except_default",
			currentLocale:  "en",
			defaultLocale:  "en",
			expectedOutput: `<a href="/about" piko:a=""></a>`,
		},
		{
			name: "prefix_except_default adds prefix for non-default",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/about"},
				},
			},
			i18nStrategy:   "prefix_except_default",
			currentLocale:  "fr",
			defaultLocale:  "en",
			expectedOutput: `<a href="/fr/about" piko:a=""></a>`,
		},
		{
			name: "query-only strategy adds query parameter",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/about"},
				},
			},
			i18nStrategy:   "query-only",
			currentLocale:  "de",
			expectedOutput: `<a href="/about?locale=de" piko:a=""></a>`,
		},
		{
			name: "lang attribute overrides current locale",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/about"},
					{Name: "lang", Value: "es"},
				},
			},
			i18nStrategy:   "prefix",
			currentLocale:  "fr",
			expectedOutput: `<a href="/es/about" piko:a=""></a>`,
		},
		{
			name: "empty lang attribute disables transformation",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "/about"},
					{Name: "lang", Value: ""},
				},
			},
			i18nStrategy:   "prefix",
			currentLocale:  "fr",
			expectedOutput: `<a href="/about" piko:a=""></a>`,
		},
		{
			name: "absolute URLs are not transformed",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "https://example.com"},
				},
			},
			i18nStrategy:   "prefix",
			currentLocale:  "fr",
			expectedOutput: `<a href="https://example.com" piko:a=""></a>`,
		},
		{
			name: "anchor links are not transformed",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:a",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "href", Value: "#section"},
				},
			},
			i18nStrategy:   "prefix",
			currentLocale:  "fr",
			expectedOutput: `<a href="#section" piko:a=""></a>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := newTestRenderContext(nil)
			rctx.i18nStrategy = tc.i18nStrategy
			rctx.currentLocale = tc.currentLocale
			rctx.defaultLocale = tc.defaultLocale

			output := renderToString(t, tc.inputNode, rctx, renderPikoA)
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}

func TestRenderPikoImg(t *testing.T) {
	testCases := []struct {
		name           string
		inputNode      *ast_domain.TemplateNode
		expectedOutput string
	}{
		{
			name: "Basic piko:img with src",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "github.com/example/assets/image.png"},
				},
			},
			expectedOutput: `<img src="/_piko/assets/github.com/example/assets/image.png" />`,
		},
		{
			name: "piko:img with alt and class",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "github.com/example/assets/hero.jpg"},
					{Name: "alt", Value: "Hero image"},
					{Name: "class", Value: "hero-img"},
				},
			},
			expectedOutput: `<img src="/_piko/assets/github.com/example/assets/hero.jpg" alt="Hero image" class="hero-img" />`,
		},
		{
			name: "piko:img removes profile attribute",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "github.com/example/assets/image.png"},
					{Name: "profile", Value: "thumbnail"},
				},
			},
			expectedOutput: `<img src="/_piko/assets/github.com/example/assets/image.png" />`,
		},
		{
			name: "piko:img removes densities attribute",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "github.com/example/assets/image.png"},
					{Name: "densities", Value: "1x,2x"},
				},
			},
			expectedOutput: `<img src="/_piko/assets/github.com/example/assets/image.png" />`,
		},
		{
			name: "piko:img removes sizes attribute",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "github.com/example/assets/image.png"},
					{Name: "sizes", Value: "(max-width: 600px) 100vw"},
				},
			},
			expectedOutput: `<img src="/_piko/assets/github.com/example/assets/image.png" />`,
		},
		{
			name: "piko:img preserves width and height",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "github.com/example/assets/image.png"},
					{Name: "width", Value: "200"},
					{Name: "height", Value: "150"},
				},
			},
			expectedOutput: `<img src="/_piko/assets/github.com/example/assets/image.png" width="200" height="150" />`,
		},
		{
			name: "piko:img with no src",
			inputNode: &ast_domain.TemplateNode{
				TagName: "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "alt", Value: "Missing image"},
				},
			},
			expectedOutput: `<img alt="Missing image" />`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := newTestRenderContext(nil)
			output := renderToString(t, tc.inputNode, rctx, renderPikoImg)
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}

func TestWriteErrorDiv_NoAttributes(t *testing.T) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeErrorDiv(qw, nil, "Error message")

	output := buffer.String()
	assert.Equal(t, "<div>Error message</div>", output)
}

func TestWriteErrorDiv_WithAttributes(t *testing.T) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	attrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "error-container"},
		{Name: "id", Value: "svg-error"},
		{Name: "data-error", Value: "missing-src"},
	}

	writeErrorDiv(qw, attrs, "<!-- piko:svg error: missing src -->")

	output := buffer.String()
	assert.Contains(t, output, "<div")
	assert.Contains(t, output, `class="error-container"`)
	assert.Contains(t, output, `id="svg-error"`)
	assert.Contains(t, output, `data-error="missing-src"`)
	assert.Contains(t, output, "<!-- piko:svg error: missing src -->")
	assert.Contains(t, output, "</div>")
}

func TestWriteErrorDiv_EmptyMessage(t *testing.T) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeErrorDiv(qw, nil, "")

	output := buffer.String()
	assert.Equal(t, "<div></div>", output)
}

func TestWriteErrorDiv_SingleAttribute(t *testing.T) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	attrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "icon-fallback"},
	}

	writeErrorDiv(qw, attrs, "Icon not found")

	output := buffer.String()
	assert.Equal(t, `<div class="icon-fallback">Icon not found</div>`, output)
}

func TestToLowerIfNeeded_AlreadyLowercase(t *testing.T) {
	input := "lowercase"
	result := toLowerIfNeeded(input)
	assert.Equal(t, input, result)
}

func TestToLowerIfNeeded_WithUppercase(t *testing.T) {
	result := toLowerIfNeeded("MixedCase")
	assert.Equal(t, "mixedcase", result)
}

func TestToLowerIfNeeded_AllUppercase(t *testing.T) {
	result := toLowerIfNeeded("UPPERCASE")
	assert.Equal(t, "uppercase", result)
}

func TestToLowerIfNeeded_EmptyString(t *testing.T) {
	result := toLowerIfNeeded("")
	assert.Equal(t, "", result)
}

func TestToLowerIfNeeded_NumbersAndSymbols(t *testing.T) {
	input := "data-value123"
	result := toLowerIfNeeded(input)
	assert.Equal(t, input, result)
}
