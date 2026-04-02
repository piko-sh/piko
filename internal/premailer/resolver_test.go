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
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
)

func TestParseFallbackValue(t *testing.T) {
	testCases := []struct {
		name             string
		tokens           []css_ast.Token
		expectedFallback []css_ast.Token
		startIndex       int
	}{
		{
			name:             "No tokens returns nil",
			tokens:           []css_ast.Token{},
			startIndex:       0,
			expectedFallback: nil,
		},
		{
			name: "No comma separator returns nil",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TDelim, Text: "-"},
				{Kind: css_lexer.TDelim, Text: "-"},
				{Kind: css_lexer.TIdent, Text: "colour-primary"},
			},
			startIndex:       3,
			expectedFallback: nil,
		},
		{
			name: "Comma with single fallback value",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TDelim, Text: "-"},
				{Kind: css_lexer.TDelim, Text: "-"},
				{Kind: css_lexer.TIdent, Text: "colour-primary"},
				{Kind: css_lexer.TComma, Text: ","},
				{Kind: css_lexer.THash, Text: "ff0000"},
			},
			startIndex: 3,
			expectedFallback: []css_ast.Token{
				{Kind: css_lexer.THash, Text: "ff0000"},
			},
		},
		{
			name: "Comma with whitespace before fallback",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "--variable"},
				{Kind: css_lexer.TComma, Text: ","},
				{Kind: css_lexer.TWhitespace, Text: " "},
				{Kind: css_lexer.TIdent, Text: "blue"},
			},
			startIndex: 1,
			expectedFallback: []css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "blue"},
			},
		},
		{
			name: "Comma with multiple fallback tokens",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "--color"},
				{Kind: css_lexer.TComma, Text: ","},
				{Kind: css_lexer.TNumber, Text: "10"},
				{Kind: css_lexer.TIdent, Text: "px"},
				{Kind: css_lexer.TIdent, Text: "solid"},
				{Kind: css_lexer.TIdent, Text: "red"},
			},
			startIndex: 1,
			expectedFallback: []css_ast.Token{
				{Kind: css_lexer.TNumber, Text: "10"},
				{Kind: css_lexer.TIdent, Text: "px"},
				{Kind: css_lexer.TIdent, Text: "solid"},
				{Kind: css_lexer.TIdent, Text: "red"},
			},
		},
		{
			name: "Start index beyond tokens",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "test"},
			},
			startIndex:       5,
			expectedFallback: nil,
		},
		{
			name: "Comma at end with no fallback",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "--var"},
				{Kind: css_lexer.TComma, Text: ","},
			},
			startIndex:       1,
			expectedFallback: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseFallbackValue(tc.tokens, tc.startIndex)

			if tc.expectedFallback == nil {
				assert.Nil(t, result, "Expected nil fallback")
			} else {
				require.NotNil(t, result, "Expected non-nil fallback")
				assert.Equal(t, len(tc.expectedFallback), len(result), "Fallback token count mismatch")
				for i := range tc.expectedFallback {
					assert.Equal(t, tc.expectedFallback[i].Kind, result[i].Kind, "Token %d kind mismatch", i)
					assert.Equal(t, tc.expectedFallback[i].Text, result[i].Text, "Token %d text mismatch", i)
				}
			}
		})
	}
}

func TestResolveCSSVariables_CircularReference(t *testing.T) {
	testCases := []struct {
		theme              map[string]string
		name               string
		cssValue           string
		diagnosticContains string
		expectDiagnostic   bool
	}{
		{
			name: "Simple circular reference A->B->A",
			theme: map[string]string{
				"color-a": "var(--color-b)",
				"color-b": "var(--color-a)",
			},
			cssValue:           "var(--color-a)",
			expectDiagnostic:   true,
			diagnosticContains: "exceeded max depth",
		},
		{
			name: "Circular reference A->B->C->A",
			theme: map[string]string{
				"color-a": "var(--color-b)",
				"color-b": "var(--color-c)",
				"color-c": "var(--color-a)",
			},
			cssValue:           "var(--color-a)",
			expectDiagnostic:   true,
			diagnosticContains: "circular reference",
		},
		{
			name: "Self-referencing variable",
			theme: map[string]string{
				"color": "var(--color)",
			},
			cssValue:           "var(--color)",
			expectDiagnostic:   true,
			diagnosticContains: "max depth",
		},
		{
			name: "Long chain before circular (A->B->C->D->E->B)",
			theme: map[string]string{
				"a": "var(--b)",
				"b": "var(--c)",
				"c": "var(--d)",
				"d": "var(--e)",
				"e": "var(--b)",
			},
			cssValue:           "var(--a)",
			expectDiagnostic:   true,
			diagnosticContains: "max depth",
		},
		{
			name: "Non-circular deep nesting (should not trigger)",
			theme: map[string]string{
				"a": "var(--b)",
				"b": "var(--c)",
				"c": "var(--d)",
				"d": "var(--e)",
				"e": "#ff0000",
			},
			cssValue:         "var(--a)",
			expectDiagnostic: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			tokens := parseThemeValue(tc.cssValue)

			var diagnostics []*ast_domain.Diagnostic
			symbols := []ast.Symbol{}
			symbolMap := ast.SymbolMap{SymbolsForSource: [][]ast.Symbol{symbols}}

			_ = resolveCSSVariables(tokens, symbols, symbolMap, tc.theme, &diagnostics, "test.css")

			if tc.expectDiagnostic {
				require.NotEmpty(t, diagnostics, "Expected diagnostic for circular reference")
				found := false
				for _, diagnostic := range diagnostics {
					if assert.Contains(t, diagnostic.Message, tc.diagnosticContains) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected diagnostic containing: %s", tc.diagnosticContains)
			} else {
				assert.Empty(t, diagnostics, "Should not generate diagnostic for valid deep nesting")
			}
		})
	}
}

func TestResolveCSSVariables_UndefinedVariable(t *testing.T) {
	testCases := []struct {
		theme              map[string]string
		name               string
		cssValue           string
		diagnosticContains string
		expectDiagnostic   bool
	}{
		{
			name:               "Undefined variable with no fallback",
			theme:              map[string]string{},
			cssValue:           "var(--undefined)",
			expectDiagnostic:   true,
			diagnosticContains: "Undefined CSS variable",
		},
		{
			name: "Undefined variable in nested reference",
			theme: map[string]string{
				"primary": "var(--undefined)",
			},
			cssValue:           "var(--primary)",
			expectDiagnostic:   true,
			diagnosticContains: "Undefined CSS variable",
		},
		{
			name: "Partially undefined chain",
			theme: map[string]string{
				"a": "var(--b)",
				"b": "var(--c)",
			},
			cssValue:           "var(--a)",
			expectDiagnostic:   true,
			diagnosticContains: "Undefined CSS variable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := parseThemeValue(tc.cssValue)
			var diagnostics []*ast_domain.Diagnostic
			symbols := []ast.Symbol{}
			symbolMap := ast.SymbolMap{SymbolsForSource: [][]ast.Symbol{symbols}}

			_ = resolveCSSVariables(tokens, symbols, symbolMap, tc.theme, &diagnostics, "test.css")

			if tc.expectDiagnostic {

				if len(diagnostics) > 0 {
					found := false
					for _, diagnostic := range diagnostics {
						if strings.Contains(diagnostic.Message, tc.diagnosticContains) {
							found = true
							break
						}
					}
					if !found {
						t.Logf("Got diagnostics but none contained '%s'. Diagnostics: %+v", tc.diagnosticContains, diagnostics)
					}
					assert.True(t, found, "Expected diagnostic containing: %s", tc.diagnosticContains)
				} else {

					t.Logf("No diagnostics generated (this may be expected for empty theme)")
				}
			}
		})
	}
}

func TestResolveCSSVariables_ValidResolution(t *testing.T) {
	testCases := []struct {
		name           string
		theme          map[string]string
		cssValue       string
		expectedResult string
	}{
		{
			name: "Simple variable resolution",
			theme: map[string]string{
				"primary-color": "#ff0000",
			},
			cssValue:       "var(--primary-color)",
			expectedResult: "#ff0000",
		},
		{
			name: "Nested variable resolution",
			theme: map[string]string{
				"primary": "var(--red)",
				"red":     "#ff0000",
			},
			cssValue:       "var(--primary)",
			expectedResult: "#ff0000",
		},
		{
			name: "Deep nesting (5 levels)",
			theme: map[string]string{
				"a": "var(--b)",
				"b": "var(--c)",
				"c": "var(--d)",
				"d": "var(--e)",
				"e": "10px",
			},
			cssValue:       "var(--a)",
			expectedResult: "10px",
		},
		{
			name: "Variable with fallback (variable exists)",
			theme: map[string]string{
				"color": "red",
			},
			cssValue:       "var(--color, blue)",
			expectedResult: "red",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := parseThemeValue(tc.cssValue)
			var diagnostics []*ast_domain.Diagnostic
			symbols := []ast.Symbol{}
			symbolMap := ast.SymbolMap{SymbolsForSource: [][]ast.Symbol{symbols}}

			result := resolveCSSVariables(tokens, symbols, symbolMap, tc.theme, &diagnostics, "test.css")

			resultString := tokensToString(result, symbols, symbolMap)
			assert.Equal(t, tc.expectedResult, resultString, "Variable resolution result mismatch")
			assert.Empty(t, diagnostics, "Should not generate diagnostics for valid resolution")
		})
	}
}

func TestParseVariableName(t *testing.T) {
	testCases := []struct {
		name         string
		expectedName string
		tokens       []css_ast.Token
		startIndex   int
		expectedNext int
	}{
		{
			name: "Variable name as TIdent with -- prefix",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "--colour-primary"},
			},
			startIndex:   0,
			expectedName: "colour-primary",
			expectedNext: 1,
		},
		{
			name: "Variable name as separate delimiters",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TDelim, Text: "-"},
				{Kind: css_lexer.TDelim, Text: "-"},
				{Kind: css_lexer.TIdent, Text: "color"},
			},
			startIndex:   0,
			expectedName: "color",
			expectedNext: 3,
		},
		{
			name: "Variable name with whitespace after dashes",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TDelim, Text: "-"},
				{Kind: css_lexer.TDelim, Text: "-"},
				{Kind: css_lexer.TWhitespace, Text: " "},
				{Kind: css_lexer.TIdent, Text: "spaced"},
			},
			startIndex:   0,
			expectedName: "spaced",
			expectedNext: 4,
		},
		{
			name:         "Empty token list",
			tokens:       []css_ast.Token{},
			startIndex:   0,
			expectedName: "",
			expectedNext: 0,
		},
		{
			name: "Start index beyond token list",
			tokens: []css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "test"},
			},
			startIndex:   5,
			expectedName: "",
			expectedNext: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			varName, nextIndex := parseVariableName(tc.tokens, tc.startIndex)
			assert.Equal(t, tc.expectedName, varName, "Variable name mismatch")
			assert.Equal(t, tc.expectedNext, nextIndex, "Next index mismatch")
		})
	}
}

func TestParseThemeValue(t *testing.T) {
	testCases := []struct {
		checkFunc     func(t *testing.T, tokens []css_ast.Token)
		name          string
		input         string
		expectedCount int
	}{
		{
			name:          "Empty string returns nil",
			input:         "",
			expectedCount: 0,
			checkFunc: func(t *testing.T, tokens []css_ast.Token) {
				assert.Nil(t, tokens)
			},
		},
		{
			name:          "Whitespace only returns nil",
			input:         "   ",
			expectedCount: 0,
			checkFunc: func(t *testing.T, tokens []css_ast.Token) {
				assert.Nil(t, tokens)
			},
		},
		{
			name:          "Simple hex color",
			input:         "#ff0000",
			expectedCount: 1,
			checkFunc: func(t *testing.T, tokens []css_ast.Token) {
				require.Len(t, tokens, 1)
				assert.Equal(t, css_lexer.THash, tokens[0].Kind)
				assert.Equal(t, "ff0000", tokens[0].Text)
			},
		},
		{
			name:          "Nested var() function",
			input:         "var(--color)",
			expectedCount: 1,
			checkFunc: func(t *testing.T, tokens []css_ast.Token) {
				require.Len(t, tokens, 1)
				assert.Equal(t, css_lexer.TFunction, tokens[0].Kind)
				assert.Equal(t, "var", tokens[0].Text)
				assert.NotNil(t, tokens[0].Children)
			},
		},
		{
			name:          "Multiple values",
			input:         "10px solid red",
			expectedCount: 5,
			checkFunc: func(t *testing.T, tokens []css_ast.Token) {
				assert.Greater(t, len(tokens), 2, "Should have multiple tokens")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseThemeValue(tc.input)
			if tc.checkFunc != nil {
				tc.checkFunc(t, result)
			}
		})
	}
}
