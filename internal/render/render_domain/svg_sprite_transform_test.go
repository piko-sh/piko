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
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestTransformSVG_HoistsAndNamespaces(t *testing.T) {
	testCases := []struct {
		name       string
		inner      string
		expectBody string
		expectDefs string
	}{
		{
			name:       "figma user space gradient in defs",
			inner:      `<defs><linearGradient id="paint0" x1="0" gradientUnits="userSpaceOnUse" gradientTransform="rotate(45)"><stop stop-color="#fff"/><stop offset="1" stop-color="#000"/></linearGradient></defs><path fill="url(#paint0)" d="M0 0h10v10H0z"/>`,
			expectBody: `<path fill="url(#p-paint0)" d="M0 0h10v10H0z"/>`,
			expectDefs: `<linearGradient id="p-paint0" x1="0" gradientUnits="userSpaceOnUse" gradientTransform="rotate(45)"><stop stop-color="#fff"/><stop offset="1" stop-color="#000"/></linearGradient>`,
		},
		{
			name:       "radial gradient",
			inner:      `<defs><radialGradient id="g"><stop/></radialGradient></defs><circle fill="url(#g)"/>`,
			expectBody: `<circle fill="url(#p-g)"/>`,
			expectDefs: `<radialGradient id="p-g"><stop/></radialGradient>`,
		},
		{
			name:       "pattern",
			inner:      `<pattern id="pt"><rect/></pattern><rect fill="url(#pt)"/>`,
			expectBody: `<rect fill="url(#p-pt)"/>`,
			expectDefs: `<pattern id="p-pt"><rect/></pattern>`,
		},
		{
			name:       "filter",
			inner:      `<filter id="f"><feGaussianBlur stdDeviation="2"/></filter><rect filter="url(#f)"/>`,
			expectBody: `<rect filter="url(#p-f)"/>`,
			expectDefs: `<filter id="p-f"><feGaussianBlur stdDeviation="2"/></filter>`,
		},
		{
			name:       "clip path",
			inner:      `<defs><clipPath id="c" clipPathUnits="objectBoundingBox"><circle r="5"/></clipPath></defs><path clip-path="url(#c)"/>`,
			expectBody: `<path clip-path="url(#p-c)"/>`,
			expectDefs: `<clipPath id="p-c" clipPathUnits="objectBoundingBox"><circle r="5"/></clipPath>`,
		},
		{
			name:       "mask wrapping group",
			inner:      `<mask id="m"><rect/></mask><g mask="url(#m)"></g>`,
			expectBody: `<g mask="url(#p-m)"></g>`,
			expectDefs: `<mask id="p-m"><rect/></mask>`,
		},
		{
			name:       "marker end reference",
			inner:      `<marker id="mk"><path/></marker><line marker-end="url(#mk)"/>`,
			expectBody: `<line marker-end="url(#p-mk)"/>`,
			expectDefs: `<marker id="p-mk"><path/></marker>`,
		},
		{
			name:       "gradient href inheritance",
			inner:      `<defs><linearGradient id="b" xlink:href="#a"/><linearGradient id="a"><stop/></linearGradient></defs>`,
			expectBody: ``,
			expectDefs: `<linearGradient id="p-b" xlink:href="#p-a"/><linearGradient id="p-a"><stop/></linearGradient>`,
		},
		{
			name:       "standalone gradient not wrapped in defs",
			inner:      `<linearGradient id="g"><stop/></linearGradient><rect fill="url(#g)"/>`,
			expectBody: `<rect fill="url(#p-g)"/>`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "def nested in transformed group loses ancestor transform",
			inner:      `<g transform="translate(5)"><linearGradient id="g"><stop/></linearGradient></g><rect fill="url(#g)"/>`,
			expectBody: `<g transform="translate(5)"></g><rect fill="url(#p-g)"/>`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "inline style with multiple urls and fallback paint",
			inner:      `<linearGradient id="a"><stop/></linearGradient><linearGradient id="b"><stop/></linearGradient><path style="fill:url(#a);stroke:url(#b)"/><rect fill="url(#a) #ff0000"/>`,
			expectBody: `<path style="fill:url(#p-a);stroke:url(#p-b)"/><rect fill="url(#p-a) #ff0000"/>`,
			expectDefs: `<linearGradient id="p-a"><stop/></linearGradient><linearGradient id="p-b"><stop/></linearGradient>`,
		},
		{
			name:       "self referencing use is rewritten",
			inner:      `<defs><path id="s" d="M0 0"/></defs><use href="#s"/>`,
			expectBody: `<use href="#p-s"/>`,
			expectDefs: `<path id="p-s" d="M0 0"/>`,
		},
		{
			name:       "style url rewritten but selector left untouched",
			inner:      `<defs><linearGradient id="g"><stop/></linearGradient></defs><style>.a{fill:url(#g)} #g{x:1}</style><path class="a"/>`,
			expectBody: `<style>.a{fill:url(#p-g)} #g{x:1}</style><path class="a"/>`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "comments preserved in place",
			inner:      `<!-- hi --><defs><linearGradient id="g"><stop/></linearGradient></defs><path fill="url(#g)"/><!-- bye -->`,
			expectBody: `<!-- hi --><path fill="url(#p-g)"/><!-- bye -->`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "preserves attribute case on image",
			inner:      `<radialGradient id="g"><stop/></radialGradient><image preserveAspectRatio="xMidYMid meet" xlink:href="#g"/>`,
			expectBody: `<image preserveAspectRatio="xMidYMid meet" xlink:href="#p-g"/>`,
			expectDefs: `<radialGradient id="p-g"><stop/></radialGradient>`,
		},
		{
			name:       "external file reference is not rewritten",
			inner:      `<linearGradient id="g"><stop/></linearGradient><rect fill="url(sprite.svg#g)"/><use href="other.svg#x"/>`,
			expectBody: `<rect fill="url(sprite.svg#g)"/><use href="other.svg#x"/>`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "single quoted url is rewritten preserving quote",
			inner:      `<linearGradient id="g"><stop/></linearGradient><rect fill="url('#g')"/>`,
			expectBody: `<rect fill="url('#p-g')"/>`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "boolean and unquoted attributes",
			inner:      `<linearGradient id="g"><stop/></linearGradient><rect hidden width=10 fill="url(#g)"/>`,
			expectBody: `<rect hidden width="10" fill="url(#p-g)"/>`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "script body is preserved verbatim",
			inner:      `<script>var marker = "#keepme";</script><linearGradient id="g"><stop/></linearGradient><rect fill="url(#g)"/>`,
			expectBody: `<script>var marker = "#keepme";</script><rect fill="url(#p-g)"/>`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "cdata and processing instruction preserved",
			inner:      `<?xml-stylesheet type="text/css"?><linearGradient id="g"><stop/></linearGradient><![CDATA[raw#g]]><rect fill="url(#g)"/>`,
			expectBody: `<?xml-stylesheet type="text/css"?><![CDATA[raw#g]]><rect fill="url(#p-g)"/>`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
		{
			name:       "doctype and trailing text preserved",
			inner:      `<!DOCTYPE svg><linearGradient id="g"><stop/></linearGradient><rect fill="url(#g)"/>tail`,
			expectBody: `<!DOCTYPE svg><rect fill="url(#p-g)"/>tail`,
			expectDefs: `<linearGradient id="p-g"><stop/></linearGradient>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, defs, ok := transformSVG("p", tc.inner)
			require.True(t, ok, "transform should succeed")
			assert.Equal(t, tc.expectBody, body, "symbol body")
			assert.Equal(t, tc.expectDefs, defs, "hoisted defs")
		})
	}
}

func TestTransformSVG_FallsBackOnMalformed(t *testing.T) {
	_, _, ok := transformSVG("p", `<defs><linearGradient id="g"><stop/></defs>`)
	assert.False(t, ok, "unbalanced tags should report failure")
}

func TestTransformSVG_FallsBackOnUnterminatedComment(t *testing.T) {
	_, _, ok := transformSVG("p", `<linearGradient id="g"><!-- never closed`)
	assert.False(t, ok, "unterminated comment should report failure")
}

func TestTransformSVG_FallsBackOnOversizedInput(t *testing.T) {
	huge := strings.Repeat("a", maxSVGTransformInputBytes+1)
	_, _, ok := transformSVG("p", huge)
	assert.False(t, ok, "oversized input should report failure")
}

func TestSvgIDPrefix_StableAndDistinct(t *testing.T) {
	first := svgIDPrefix("testmodule/lib/icon.svg")
	second := svgIDPrefix("testmodule/lib/icon.svg")
	assert.Equal(t, first, second, "prefix must be stable for the same artefact id")

	other := svgIDPrefix("testmodule/lib/other.svg")
	assert.NotEqual(t, first, other, "different artefact ids must produce different prefixes")

	valid := regexp.MustCompile(`^s[0-9a-z]+$`)
	assert.Regexp(t, valid, first, "prefix must be a valid identifier fragment")
}

func TestSvgNeedsTransform(t *testing.T) {
	testCases := []struct {
		name  string
		inner string
		want  bool
	}{
		{name: "plain path", inner: `<path d="M0 0"/>`, want: false},
		{name: "plain icon with stray id", inner: `<g id="layer1"><path/></g>`, want: false},
		{name: "url reference", inner: `<path fill="url(#g)"/>`, want: true},
		{name: "fragment href double quote", inner: `<use href="#x"/>`, want: true},
		{name: "fragment href single quote", inner: `<use href='#x'/>`, want: true},
		{name: "defs present", inner: `<defs></defs><path/>`, want: true},
		{name: "gradient present", inner: `<linearGradient/>`, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, svgNeedsTransform(tc.inner))
		})
	}
}

func TestComputeSymbolAndDefs_FastPathPlainIcon(t *testing.T) {
	data := &ParsedSvgData{
		InnerHTML: `<path d="M0 0"/>`,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "viewBox", Value: "0 0 24 24"},
		},
	}

	symbol, defs, _ := ComputeSymbolAndDefs("icon", data)

	assert.Equal(t, `<symbol id="icon" viewBox="0 0 24 24"><path d="M0 0"/></symbol>`, symbol)
	assert.Empty(t, defs, "plain icon has no hoisted defs")
}

func TestComputeSymbolAndDefs_HoistsGradient(t *testing.T) {
	data := &ParsedSvgData{
		InnerHTML: `<defs><linearGradient id="paint0"><stop/></linearGradient></defs><path fill="url(#paint0)"/>`,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "viewBox", Value: "0 0 24 24"},
		},
	}

	symbol, defs, _ := ComputeSymbolAndDefs("icon", data)

	prefix := svgIDPrefix("icon")
	assert.Equal(t, `<symbol id="icon" viewBox="0 0 24 24"><path fill="url(#`+prefix+`-paint0)"/></symbol>`, symbol)
	assert.Equal(t, `<linearGradient id="`+prefix+`-paint0"><stop/></linearGradient>`, defs)
}

func TestComputeSymbolAndDefs_HandlesNilData(t *testing.T) {
	symbol, defs, _ := ComputeSymbolAndDefs("icon", nil)
	assert.Empty(t, symbol)
	assert.Empty(t, defs)
}

func TestComputeSymbolAndDefs_EscapesSymbolIdentifier(t *testing.T) {
	data := &ParsedSvgData{
		InnerHTML: `<path fill="url(#g)"/><linearGradient id="g"><stop/></linearGradient>`,
	}

	symbol, _, _ := ComputeSymbolAndDefs(`icon<script>`, data)
	assert.Contains(t, symbol, `<symbol id="icon&lt;script&gt;">`)
}

func TestComputeSymbolAndDefs_DistinctPrefixesAvoidCollision(t *testing.T) {
	inner := `<defs><linearGradient id="paint0_linear"><stop/></linearGradient></defs><path fill="url(#paint0_linear)"/>`

	_, defsA, _ := ComputeSymbolAndDefs("icon-a", &ParsedSvgData{InnerHTML: inner})
	_, defsB, _ := ComputeSymbolAndDefs("icon-b", &ParsedSvgData{InnerHTML: inner})

	prefixA := svgIDPrefix("icon-a")
	prefixB := svgIDPrefix("icon-b")
	require.NotEqual(t, prefixA, prefixB)

	assert.Contains(t, defsA, `id="`+prefixA+`-paint0_linear"`)
	assert.Contains(t, defsB, `id="`+prefixB+`-paint0_linear"`)
	assert.NotContains(t, defsA, `id="paint0_linear"`, "raw colliding id must not survive")
	assert.NotContains(t, defsB, `id="paint0_linear"`, "raw colliding id must not survive")
}

func TestTransformSVG_FallsBackWhenIDLimitExceeded(t *testing.T) {
	var builder strings.Builder
	for index := 0; index <= maxSVGTransformIDs; index++ {
		builder.WriteString(`<a id="`)
		builder.WriteString(strconv.Itoa(index))
		builder.WriteString(`"/>`)
	}

	_, _, ok := transformSVG("p", builder.String())
	assert.False(t, ok, "exceeding the identifier limit should fall back to verbatim")
}

func TestTransformSVG_FallsBackWhenTokenLimitExceeded(t *testing.T) {
	inner := strings.Repeat("<a/>", maxSVGTransformTokens+5)

	_, _, ok := transformSVG("p", inner)
	assert.False(t, ok, "exceeding the token limit should fall back to verbatim")
}

func TestComputeSymbolAndDefs_FormatsSymbol(t *testing.T) {
	testCases := []struct {
		name       string
		id         string
		parsedData *ParsedSvgData
		expected   []string
	}{
		{
			name: "basic symbol with viewBox",
			id:   "test-icon",
			parsedData: &ParsedSvgData{
				InnerHTML: `<path d="M0 0"/>`,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "viewBox", Value: "0 0 24 24"},
				},
			},
			expected: []string{
				`<symbol id="test-icon"`,
				`viewBox="0 0 24 24"`,
				`<path d="M0 0"/>`,
				`</symbol>`,
			},
		},
		{
			name: "symbol without viewBox",
			id:   "no-viewbox",
			parsedData: &ParsedSvgData{
				InnerHTML:  `<rect width="10" height="10"/>`,
				Attributes: []ast_domain.HTMLAttribute{},
			},
			expected: []string{
				`<symbol id="no-viewbox">`,
				`<rect width="10" height="10"/>`,
				`</symbol>`,
			},
		},
		{
			name: "escapes special characters in ID",
			id:   `icon<script>`,
			parsedData: &ParsedSvgData{
				InnerHTML: `<path/>`,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "viewBox", Value: "0 0 24 24"},
				},
			},
			expected: []string{
				`&lt;script&gt;`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _, _ := ComputeSymbolAndDefs(tc.id, tc.parsedData)
			for _, exp := range tc.expected {
				assert.Contains(t, result, exp)
			}
		})
	}
}

func TestComputeSymbolAndDefs_HandlesEmptyInnerHTML(t *testing.T) {
	parsedData := &ParsedSvgData{
		InnerHTML: "",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "viewBox", Value: "0 0 24 24"},
		},
	}

	result, _, _ := ComputeSymbolAndDefs("empty-svg", parsedData)

	assert.Contains(t, result, `<symbol id="empty-svg"`)
	assert.Contains(t, result, `</symbol>`)
}

func TestComputeSymbolAndDefs_HandlesNoViewBox(t *testing.T) {
	parsedData := &ParsedSvgData{
		InnerHTML: `<path d="M0 0"/>`,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "width", Value: "24"},
			{Name: "height", Value: "24"},
		},
	}

	result, _, _ := ComputeSymbolAndDefs("no-viewbox", parsedData)

	assert.Contains(t, result, `<symbol id="no-viewbox">`)
	assert.NotContains(t, result, `viewBox=`)
	assert.Contains(t, result, `<path d="M0 0"/>`)
}

func TestComputeSymbolAndDefs_EscapesViewBoxValue(t *testing.T) {
	parsedData := &ParsedSvgData{
		InnerHTML: `<path/>`,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "viewBox", Value: `0 0 24" onload="alert(1)`},
		},
	}

	result, _, _ := ComputeSymbolAndDefs("xss-test", parsedData)

	assert.Contains(t, result, "&#34;")
	assert.Contains(t, result, `viewBox="0 0 24&#34; onload=&#34;alert(1)"`)
}
