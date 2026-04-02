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

package premailer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/logger"
)

func parseTestCSS(t *testing.T, css string) css_ast.AST {
	t.Helper()
	log := logger.NewDeferLog(logger.DeferLogAll, nil)
	source := logger.Source{Contents: css}
	cssAST := css_parser.Parse(log, source, css_parser.Options{})
	require.False(t, log.HasErrors(), "CSS parsing should not have errors for test case: %s", css)
	return cssAST
}

func TestProcessCSS(t *testing.T) {
	testCases := []struct {
		name                    string
		css                     string
		expectedTopSelector     string
		expectedInlineableCount int
		expectedLeftoverCount   int
	}{
		{
			name: "Simple case with one of each type",
			css: `
				p { color: blue; }
				#main { border: 1px solid black; }
				.card { background: white; }
			`,
			expectedInlineableCount: 3,
			expectedLeftoverCount:   0,
			expectedTopSelector:     "#main",
		},
		{
			name: "Un-inlineable rules are correctly categorised",
			css: `
				p { color: blue; }
				a:hover { text-decoration: underline; }
				.card::before { content: 'Hi'; }
				@media (min-width: 600px) {
					body { font-size: 16px; }
				}
			`,
			expectedInlineableCount: 1,
			expectedLeftoverCount:   3,
			expectedTopSelector:     "p",
		},
		{
			name: "Specificity sort order is correct",
			css: `
				p { color: red; }                  /* spec: 1 */
				#content { margin: 0; }           /* spec: 100 */
				div.card { padding: 10px; }       /* spec: 11 */
				a { color: blue; }                 /* spec: 1 */
			`,
			expectedInlineableCount: 4,
			expectedLeftoverCount:   0,
			expectedTopSelector:     "#content",
		},
		{
			name: "Comma-separated selectors are split into multiple rules",
			css: `
				h1, .title { font-weight: bold; }
				p { color: gray; }
			`,
			expectedInlineableCount: 3,
			expectedLeftoverCount:   0,
			expectedTopSelector:     ".title",
		},
		{
			name: "Empty and comment-only CSS produces an empty ruleset",
			css: `
				/* This is a comment */
			`,
			expectedInlineableCount: 0,
			expectedLeftoverCount:   0,
			expectedTopSelector:     "",
		},
		{
			name:                    "Completely empty css",
			css:                     ``,
			expectedInlineableCount: 0,
			expectedLeftoverCount:   0,
			expectedTopSelector:     "",
		},
		{
			name: "Stable sort is maintained for rules with same specificity",
			css: `
				a { color: red; } /* spec: 1, first */
				p { color: blue; } /* spec: 1, second */
			`,
			expectedInlineableCount: 2,
			expectedLeftoverCount:   0,
			expectedTopSelector:     "p",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cssAST := parseTestCSS(t, tc.css)

			var diagnostics []*ast_domain.Diagnostic
			ruleSet := ProcessCSS(cssAST, &Options{ExpandShorthands: false}, &diagnostics, "test.css")

			assert.Len(t, ruleSet.InlineableRules, tc.expectedInlineableCount)
			assert.Len(t, ruleSet.LeftoverRules, tc.expectedLeftoverCount)

			if tc.expectedTopSelector != "" {
				require.NotEmpty(t, ruleSet.InlineableRules)
				highestSpecRule := ruleSet.InlineableRules[len(ruleSet.InlineableRules)-1]
				assert.Equal(t, tc.expectedTopSelector, highestSpecRule.selector)
			} else {
				assert.Empty(t, ruleSet.InlineableRules)
			}
		})
	}

	t.Run("Properties are correctly parsed", func(t *testing.T) {
		css := `p { color: red !important; font-size: 12px; }`
		cssAST := parseTestCSS(t, css)
		var diagnostics []*ast_domain.Diagnostic
		ruleSet := ProcessCSS(cssAST, &Options{ExpandShorthands: false}, &diagnostics, "test.css")

		require.Len(t, ruleSet.InlineableRules, 1)
		rule := ruleSet.InlineableRules[0]

		assert.Equal(t, "p", rule.selector)
		assert.Len(t, rule.properties, 2)

		colorProp, ok := rule.properties["color"]
		assert.True(t, ok)
		assert.Equal(t, "#ff0000", colorProp.value)
		assert.True(t, colorProp.important)

		fontProp, ok := rule.properties["font-size"]
		assert.True(t, ok)
		assert.Equal(t, "12px", fontProp.value)
		assert.False(t, fontProp.important)
	})
}

func TestIsInlineable(t *testing.T) {
	testCases := []struct {
		name     string
		selector string
		expected bool
	}{
		{name: "simple element", selector: "p {}", expected: true},
		{name: "simple class", selector: ".foo {}", expected: true},
		{name: "simple id", selector: "#bar {}", expected: true},
		{name: "attribute selector", selector: "[href] {}", expected: true},
		{name: "complex inlineable", selector: "div.foo#bar[data-test] {}", expected: true},
		{name: "descendant selector", selector: "div p {}", expected: true},
		{name: "child selector", selector: "ul > li {}", expected: true},
		{name: "universal selector", selector: "* {}", expected: false},
		{name: "un-inlineable pseudo-class", selector: "a:hover {}", expected: false},
		{name: "un-inlineable pseudo-element", selector: "p::before {}", expected: false},
		{name: "un-inlineable functional pseudo-class", selector: "div:not(.foo) {}", expected: false},
		{name: "un-inlineable functional pseudo-class :is", selector: "p:is(a, b) {}", expected: false},
		{name: "comma-separated list with one un-inlineable part", selector: "p, a:hover {}", expected: false},
		{name: "at-rule (@media)", selector: "@media screen {}", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			if strings.HasPrefix(tc.selector, "@") {
				cssAST := parseTestCSS(t, tc.selector)
				require.NotEmpty(t, cssAST.Rules)
				assert.False(t, isInlineable(cssAST.Rules[0]))
				return
			}

			cssAST := parseTestCSS(t, tc.selector)
			require.NotEmpty(t, cssAST.Rules)
			rule := cssAST.Rules[0]

			actual := isInlineable(rule)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestCalculateSpecificityAST(t *testing.T) {
	testCases := []struct {
		name     string
		selector string
		expected int
	}{
		{name: "universal selector", selector: "*", expected: 0},
		{name: "element selector", selector: "p", expected: 1},
		{name: "class selector", selector: ".foo", expected: 10},
		{name: "attribute selector", selector: "[href]", expected: 10},
		{name: "ID selector", selector: "#bar", expected: 100},
		{name: "pseudo-class", selector: ":hover", expected: 10},
		{name: "pseudo-element", selector: "::before", expected: 1},
		{name: "element and class", selector: "div.foo", expected: 11},
		{name: "element and id", selector: "div#foo", expected: 101},
		{name: "multiple classes", selector: ".a.b.c", expected: 30},
		{name: "complex chain", selector: "#nav .item > a", expected: 111},
		{name: "very specific", selector: `ul#nav li.item a[href^="https"]:active`, expected: 133},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cssAST := parseTestCSS(t, tc.selector+" {}")
			require.NotEmpty(t, cssAST.Rules)
			rule, ok := cssAST.Rules[0].Data.(*css_ast.RSelector)
			require.True(t, ok)
			require.NotEmpty(t, rule.Selectors)
			complexSelector := rule.Selectors[0]

			actual := calculateSpecificityAST(complexSelector)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
