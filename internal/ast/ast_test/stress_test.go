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

package ast_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestParse_UTF8EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{

		{name: "UTF-8 BOM at start", input: "\xef\xbb\xbf<div>Hello</div>"},
		{name: "UTF-8 BOM mid-content", input: "<div>\xef\xbb\xbfHello</div>"},
		{name: "UTF-16 LE BOM (invalid)", input: "\xff\xfe<div>Hello</div>"},
		{name: "UTF-16 BE BOM (invalid)", input: "\xfe\xff<div>Hello</div>"},

		{name: "Zero-width space U+200B", input: "<div>\u200b</div>"},
		{name: "Zero-width non-joiner U+200C", input: "<div>\u200c</div>"},
		{name: "Zero-width joiner U+200D", input: "<div>\u200d</div>"},
		{name: "Word joiner U+2060", input: "<div>\u2060</div>"},
		{name: "ZWNBSP/BOM U+FEFF", input: "<div>\ufeff</div>"},
		{name: "Zero-width in identifier", input: `<div p-if="a` + "\u200b" + `b">x</div>`},

		{name: "RTL override U+202E", input: "<div>\u202e reversed</div>"},
		{name: "LTR override U+202D", input: "<div>\u202d normal</div>"},
		{name: "Pop directional U+202C", input: "<div>\u202c</div>"},
		{name: "RLI U+2067", input: "<div>\u2067text\u2069</div>"},
		{name: "LRI U+2066", input: "<div>\u2066text\u2069</div>"},

		{name: "2-byte: Latin Extended", input: "<div>café résumé naïve</div>"},
		{name: "3-byte: CJK", input: "<div>中文日本語한국어</div>"},
		{name: "3-byte: Currency symbols", input: "<div>€£¥₹</div>"},
		{name: "4-byte: Emoji", input: "<div>🚀🎉🔥💯</div>"},
		{name: "4-byte: Musical", input: "<div>𝄞𝄢</div>"},
		{name: "4-byte: Math", input: "<div>𝕏𝔸𝔹ℂ</div>"},
		{name: "Max codepoint U+10FFFF", input: "<div>\U0010FFFF</div>"},

		{name: "Precomposed é U+00E9", input: "<div>café</div>"},
		{name: "Decomposed é (e + U+0301)", input: "<div>cafe\u0301</div>"},
		{name: "Multiple combiners ộ", input: "<div>o\u0302\u0323</div>"},
		{name: "Zalgo text (many combiners)", input: "<div>H\u0335\u0310\u0305e\u0335\u0310\u0305l\u0335\u0310\u0305l\u0335\u0310\u0305o\u0335\u0310\u0305</div>"},
		{name: "Combiner in attribute", input: `<div title="cafe` + "\u0301" + `">x</div>`},

		{name: "Invalid: lone continuation 0x80", input: "<div>\x80</div>"},
		{name: "Invalid: lone continuation 0xBF", input: "<div>\xbf</div>"},
		{name: "Invalid: multiple continuations", input: "<div>\x80\x80\x80</div>"},

		{name: "Invalid: 0xC0 (overlong)", input: "<div>\xc0</div>"},
		{name: "Invalid: 0xC1 (overlong)", input: "<div>\xc1</div>"},
		{name: "Invalid: 0xF5+ (beyond Unicode)", input: "<div>\xf5</div>"},
		{name: "Invalid: 0xFE", input: "<div>\xfe</div>"},
		{name: "Invalid: 0xFF", input: "<div>\xff</div>"},

		{name: "Truncated: 2-byte missing cont", input: "<div>\xc2</div>"},
		{name: "Truncated: 3-byte missing 1", input: "<div>\xe0\xa0</div>"},
		{name: "Truncated: 3-byte missing 2", input: "<div>\xe0</div>"},
		{name: "Truncated: 4-byte missing 1", input: "<div>\xf0\x90\x80</div>"},
		{name: "Truncated: 4-byte missing 2", input: "<div>\xf0\x90</div>"},
		{name: "Truncated: 4-byte missing 3", input: "<div>\xf0</div>"},

		{name: "Overlong: NUL as 2-byte", input: "<div>\xc0\x80</div>"},
		{name: "Overlong: slash as 2-byte", input: "<div>\xc0\xaf</div>"},
		{name: "Overlong: NUL as 3-byte", input: "<div>\xe0\x80\x80</div>"},
		{name: "Overlong: NUL as 4-byte", input: "<div>\xf0\x80\x80\x80</div>"},

		{name: "Invalid: high surrogate D800", input: "<div>\xed\xa0\x80</div>"},
		{name: "Invalid: low surrogate DC00", input: "<div>\xed\xb0\x80</div>"},
		{name: "Invalid: surrogate pair encoded", input: "<div>\xed\xa0\x80\xed\xb0\x80</div>"},

		{name: "Invalid: beyond U+10FFFF", input: "<div>\xf4\x90\x80\x80</div>"},

		{name: "Invalid at tag boundary", input: "<div\x80>x</div>"},
		{name: "Invalid in attribute name", input: "<div da\x80ta=\"x\">y</div>"},
		{name: "Invalid in attribute value", input: `<div title="` + "\x80" + `">x</div>`},
		{name: "Invalid in directive expression", input: `<div p-if="x` + "\x80" + `y">z</div>`},
		{name: "Invalid in interpolation", input: "<div>{{ x\x80y }}</div>"},
		{name: "UTF-8 at buffer boundary", input: "<div>" + strings.Repeat("x", 4095) + "é</div>"},

		{name: "Valid emoji + invalid", input: "<div>🚀\x80🎉</div>"},
		{name: "CJK + invalid", input: "<div>中\x80文</div>"},

		{name: "Mixed LTR/RTL", input: "<div>Hello مرحبا שלום World</div>"},
		{name: "CJK mixed", input: "<div>Hello 你好 こんにちは 안녕</div>"},

		{name: "Unicode identifier", input: `<div p-if="naïve">x</div>`},
		{name: "Emoji in string literal", input: `<div p-if="x == '🚀'">y</div>`},
		{name: "CJK in string", input: `<div p-text="'你好世界'">x</div>`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Parser panicked: %v", r)
				}
			}()

			ast, _ := ast_domain.ParseAndTransform(context.Background(), tc.input, "test.piko")
			require.NotNil(t, ast, "ParseAndTransform should return a result")
		})
	}
}

func TestParse_ResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource limit tests in short mode")
	}

	tests := []struct {
		generator func() string
		name      string
		timeout   time.Duration
	}{

		{
			name: "100 levels deep",
			generator: func() string {
				return strings.Repeat("<div>", 100) + "x" + strings.Repeat("</div>", 100)
			},
			timeout: 10 * time.Second,
		},
		{
			name: "500 levels deep",
			generator: func() string {
				return strings.Repeat("<div>", 500) + "x" + strings.Repeat("</div>", 500)
			},
			timeout: 30 * time.Second,
		},

		{
			name: "1000 siblings",
			generator: func() string {
				return "<div>" + strings.Repeat("<span>x</span>", 1000) + "</div>"
			},
			timeout: 10 * time.Second,
		},
		{
			name: "5000 siblings",
			generator: func() string {
				return "<div>" + strings.Repeat("<span>x</span>", 5000) + "</div>"
			},
			timeout: 30 * time.Second,
		},

		{
			name: "100 attributes",
			generator: func() string {
				var b strings.Builder
				b.WriteString("<div")
				for i := range 100 {
					_, _ = fmt.Fprintf(&b, ` data-%d="%d"`, i, i)
				}
				b.WriteString(">x</div>")
				return b.String()
			},
			timeout: 10 * time.Second,
		},
		{
			name: "500 attributes",
			generator: func() string {
				var b strings.Builder
				b.WriteString("<div")
				for i := range 500 {
					_, _ = fmt.Fprintf(&b, ` data-%d="%d"`, i, i)
				}
				b.WriteString(">x</div>")
				return b.String()
			},
			timeout: 30 * time.Second,
		},

		{
			name: "100KB text content",
			generator: func() string {
				return "<div>" + strings.Repeat("x", 100*1024) + "</div>"
			},
			timeout: 30 * time.Second,
		},

		{
			name: "50 chained operators",
			generator: func() string {
				var builder strings.Builder
				builder.WriteString("a")
				for range 50 {
					builder.WriteString(" + b")
				}
				return fmt.Sprintf(`<div p-if="%s">x</div>`, builder.String())
			},
			timeout: 10 * time.Second,
		},

		{
			name: "100 interpolations",
			generator: func() string {
				var b strings.Builder
				b.WriteString("<div>")
				for i := range 100 {
					_, _ = fmt.Fprintf(&b, "{{ var%d }}", i)
				}
				b.WriteString("</div>")
				return b.String()
			},
			timeout: 2 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.generator()

			done := make(chan struct{})
			go func() {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Parser panicked: %v", r)
					}
				}()

				ast, _ := ast_domain.ParseAndTransform(context.Background(), input, "test.piko")
				require.NotNil(t, ast, "Should return a result")

				_ = ast.DeepClone()
				ast.Walk(func(node *ast_domain.TemplateNode) bool {
					return true
				})
			}()

			select {
			case <-done:

			case <-time.After(tc.timeout):
				t.Fatalf("Test timed out after %v", tc.timeout)
			}
		})
	}
}

func TestParse_LocationTracking(t *testing.T) {
	tests := []struct {
		validate func(t *testing.T, ast *ast_domain.TemplateAST)
		name     string
		input    string
	}{
		{
			name:  "1000 lines",
			input: strings.Repeat("\n", 1000) + "<div>x</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.Len(t, ast.RootNodes, 1)

				loc := ast.RootNodes[0].Location
				assert.Greater(t, loc.Line, 0, "Line should be positive")
			},
		},
		{
			name:  "1000 columns",
			input: strings.Repeat("x", 1000) + "<div>y</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.Len(t, ast.RootNodes, 2)
			},
		},
		{
			name:  "CRLF line endings",
			input: "<div>\r\nx\r\ny\r\n</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast)
			},
		},
		{
			name:  "CR only line endings",
			input: "<div>\rx\ry\r</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast)
			},
		},
		{
			name:  "LF only line endings",
			input: "<div>\nx\ny\n</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast)
			},
		},
		{
			name:  "Mixed line endings",
			input: "<div>\r\n\r\nx\n\ry</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast)
			},
		},
		{
			name:  "Tabs in content",
			input: "<div>\t\t\tx</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast)
			},
		},
		{
			name:  "Tabs in attributes",
			input: "<div\ttitle=\"x\">y</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast)
			},
		},
		{
			name:  "Expression location",
			input: "<div>{{ x }}</div>",
			validate: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.Len(t, ast.RootNodes, 1)
				require.Len(t, ast.RootNodes[0].Children, 1)
				textNode := ast.RootNodes[0].Children[0]
				require.Len(t, textNode.RichText, 1)

				expression := textNode.RichText[0].Expression
				if expression != nil {
					loc := expression.GetRelativeLocation()
					assert.GreaterOrEqual(t, loc.Offset, 0, "Offset should be non-negative")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ast, _ := ast_domain.ParseAndTransform(context.Background(), tc.input, "test.piko")
			require.NotNil(t, ast)
			tc.validate(t, ast)
		})
	}
}

func TestParse_MalformedDirectives(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{

		{name: "empty p-if", input: `<div p-if="">x</div>`},
		{name: "empty p-for", input: `<div p-for="">x</div>`},
		{name: "whitespace only", input: `<div p-if="   ">x</div>`},

		{name: "p-on alone", input: `<div p-on>x</div>`},
		{name: "p-on:", input: `<div p-on:>x</div>`},
		{name: "p-on:.", input: `<div p-on:.prevent>x</div>`},

		{name: "p-bind alone", input: `<div p-bind>x</div>`},
		{name: "p-bind:", input: `<div p-bind:>x</div>`},
		{name: ": alone", input: `<div :>x</div>`},

		{name: "unknown directive", input: `<div p-unknown="x">y</div>`},
		{name: "typo directive", input: `<div p-iff="x">y</div>`},

		{name: "p-text and p-html", input: `<div p-text="a" p-html="b">x</div>`},
		{name: "p-if and p-else", input: `<div p-if="a" p-else>x</div>`},
		{name: "duplicate p-if", input: `<div p-if="a" p-if="b">x</div>`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Parser panicked: %v", r)
				}
			}()

			ast, _ := ast_domain.ParseAndTransform(context.Background(), tc.input, "test.piko")
			require.NotNil(t, ast, "Parser should return AST even for malformed input")
		})
	}
}

func TestExpr_NumericEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "Max int64", input: "9223372036854775807"},
		{name: "Max int64 + 1", input: "9223372036854775808"},
		{name: "Max float64", input: "1.7976931348623157e+308"},
		{name: "Overflow float", input: "1e10000"},
		{name: "Negative zero", input: "-0"},
		{name: "Very small float", input: "0.0000000000000001"},
		{name: "Scientific notation positive", input: "1.5e10"},
		{name: "Scientific notation negative", input: "1.5e-10"},
		{name: "Leading zeros", input: "007"},
		{name: "Multiple decimal points", input: "1.2.3"},
		{name: "Hex notation", input: "0xFF"},
		{name: "Octal notation", input: "0o77"},
		{name: "Binary notation", input: "0b1010"},
		{name: "BigInt", input: "99999999999999999999999999999n"},
		{name: "Decimal", input: "0.1d"},
		{name: "Decimal arithmetic", input: "0.1d + 0.2d"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Parser panicked: %v", r)
				}
			}()

			parser := ast_domain.NewExpressionParser(context.Background(), tc.input, "test.piko")
			_, _ = parser.ParseExpression(context.Background())
		})
	}
}

func TestExpr_StringEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "Empty single quote", input: "''"},
		{name: "Empty double quote", input: `""`},
		{name: "Escaped single quote", input: `'it\'s'`},
		{name: "Escaped double quote", input: `"say \"hi\""`},
		{name: "Double escape", input: `'test\\'`},
		{name: "Unicode escape", input: `'\u0041'`},
		{name: "Hex escape", input: `'\x41'`},
		{name: "Newline escape", input: `'line1\nline2'`},
		{name: "Tab escape", input: `'col1\tcol2'`},
		{name: "Carriage return escape", input: `'line1\rline2'`},
		{name: "Backspace escape", input: `'text\bmore'`},
		{name: "Form feed escape", input: `'page1\fpage2'`},
		{name: "Mixed escapes", input: `'\t\n\r\\\'\"\x00'`},
		{name: "Unterminated single", input: `'unclosed`},
		{name: "Unterminated double", input: `"unclosed`},
		{name: "Very long string", input: `'` + strings.Repeat("x", 10000) + `'`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Parser panicked: %v", r)
				}
			}()

			parser := ast_domain.NewExpressionParser(context.Background(), tc.input, "test.piko")
			_, _ = parser.ParseExpression(context.Background())
		})
	}
}

func TestExpr_DepthLimits(t *testing.T) {
	tests := []struct {
		name        string
		depth       int
		shouldError bool
	}{
		{name: "50 parens - within limit", depth: 50, shouldError: false},
		{name: "100 parens - within limit", depth: 100, shouldError: false},
		{name: "200 parens - within limit", depth: 200, shouldError: false},
		{name: "300 parens - exceeds limit", depth: 300, shouldError: true},
		{name: "500 parens - exceeds limit", depth: 500, shouldError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Parser panicked at depth %d: %v", tc.depth, r)
				}
			}()

			input := strings.Repeat("(", tc.depth) + "x" + strings.Repeat(")", tc.depth)
			parser := ast_domain.NewExpressionParser(context.Background(), input, "test.piko")
			_, diagnostics := parser.ParseExpression(context.Background())

			if tc.shouldError {
				require.NotEmpty(t, diagnostics, "Expected depth limit error at depth %d", tc.depth)
			}
		})
	}
}

func TestClone_Fidelity(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{

		{
			name: "all directives",
			input: `<div p-if="a" p-show="visible" p-class="classes" p-style="styles"
			             p-text="text" p-ref="ref" p-context="ctx">content</div>`,
		},

		{
			name: "all expressions",
			input: `<div :a="ident" :b="1 + 2" :c="obj.prop" :d="arr[0]" :e="fn(x)"
			             :f="a ? b : c" :g="!x" :h="'string'" :i="123" :j="true"
			             :k="nil" :l="[1, 2, 3]" :m="{a: 1}" :n="a?.b" :o="a ?? b">x</div>`,
		},

		{
			name:  "deep clone",
			input: strings.Repeat("<div>", 30) + "x" + strings.Repeat("</div>", 30),
		},

		{
			name:  "rich text",
			input: "<div>Hello {{ name }}, you have {{ count }} messages</div>",
		},

		{
			name:  "for loop",
			input: `<div p-for="(i, item) in items" p-key="item.id">{{ item.name }}</div>`,
		},

		{
			name:  "events",
			input: `<button p-on:click="handleClick" p-on:mouseover="handleHover">Click</button>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Clone panicked: %v", r)
				}
			}()

			ast, _ := ast_domain.ParseAndTransform(context.Background(), tc.input, "test.piko")
			require.NotNil(t, ast)

			shallowClone := ast.Clone()
			require.NotNil(t, shallowClone)
			assert.Equal(t, len(ast.RootNodes), len(shallowClone.RootNodes))

			deepClone := ast.DeepClone()
			require.NotNil(t, deepClone)
			assert.Equal(t, len(ast.RootNodes), len(deepClone.RootNodes))

			if len(deepClone.RootNodes) > 0 && deepClone.RootNodes[0] != nil {
				deepClone.RootNodes[0].TagName = "modified"
				assert.NotEqual(t, ast.RootNodes[0].TagName, deepClone.RootNodes[0].TagName,
					"Deep clone should be independent")
			}
		})
	}
}

func TestEval_TypeCoercion(t *testing.T) {
	tests := []struct {
		scope      map[string]any
		name       string
		expression string
		panics     bool
	}{
		{name: "nil access", expression: "x.foo", scope: map[string]any{"x": nil}, panics: false},
		{name: "missing property", expression: "obj.nonexistent", scope: map[string]any{"obj": map[string]any{}}, panics: false},
		{name: "non-map member access", expression: "x.foo", scope: map[string]any{"x": 123}, panics: false},
		{name: "negative index", expression: "arr[-1]", scope: map[string]any{"arr": []int{1, 2, 3}}, panics: false},
		{name: "out of bounds index", expression: "arr[999]", scope: map[string]any{"arr": []int{1, 2, 3}}, panics: false},
		{name: "non-integer index", expression: "arr[\"foo\"]", scope: map[string]any{"arr": []int{1, 2, 3}}, panics: false},
		{name: "nil array index", expression: "x[0]", scope: map[string]any{"x": nil}, panics: false},
		{name: "string to number add", expression: "x + 1", scope: map[string]any{"x": "not a number"}, panics: false},
		{name: "bool arithmetic", expression: "x + 1", scope: map[string]any{"x": true}, panics: false},
		{name: "deep nil chain", expression: "a.b.c.d.e", scope: map[string]any{"a": nil}, panics: false},
		{name: "optional chaining nil", expression: "a?.b?.c", scope: map[string]any{"a": nil}, panics: false},
		{name: "nullish coalesce", expression: "x ?? 'default'", scope: map[string]any{"x": nil}, panics: false},
		{name: "empty array access", expression: "arr[0]", scope: map[string]any{"arr": []int{}}, panics: false},
		{name: "empty map access", expression: "obj.key", scope: map[string]any{"obj": map[string]any{}}, panics: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					if !tc.panics {
						t.Fatalf("EvaluateExpression panicked unexpectedly: %v", r)
					}
				}
			}()

			parser := ast_domain.NewExpressionParser(context.Background(), tc.expression, "test.piko")
			expression, diagnostics := parser.ParseExpression(context.Background())
			if len(diagnostics) > 0 || expression == nil {
				return
			}

			_ = ast_domain.EvaluateExpression(expression, tc.scope)
		})
	}
}

func TestQuery_SelectorEdgeCases(t *testing.T) {
	html := `<div id="main" class="container">
		<span class="item first"></span>
		<span class="item"></span>
		<span class="item last"></span>
	</div>`

	tests := []struct {
		name     string
		selector string
	}{

		{name: "element", selector: "div"},
		{name: "class", selector: ".container"},
		{name: "id", selector: "#main"},
		{name: "attribute presence", selector: "[id]"},
		{name: "attribute value", selector: "[id=main]"},
		{name: "descendant", selector: "div span"},
		{name: "child", selector: "div > span"},
		{name: "adjacent sibling", selector: "span + span"},
		{name: "general sibling", selector: "span ~ span"},
		{name: "first-child", selector: "span:first-child"},
		{name: "last-child", selector: "span:last-child"},
		{name: "nth-child", selector: "span:nth-child(2)"},
		{name: "nth-child odd", selector: "span:nth-child(odd)"},
		{name: "nth-child even", selector: "span:nth-child(even)"},
		{name: "nth-child formula", selector: "span:nth-child(2n+1)"},
		{name: "universal", selector: "*"},
		{name: "compound", selector: "span.item.first"},
		{name: "multiple", selector: ".first, .last"},

		{name: "empty", selector: ""},
		{name: "single dot", selector: "."},
		{name: "single hash", selector: "#"},
		{name: "single bracket", selector: "["},
		{name: "unmatched bracket", selector: "]"},
		{name: "double colon", selector: "::"},
		{name: "incomplete class", selector: "div."},
		{name: "incomplete id", selector: "div#"},
		{name: "unknown pseudo", selector: "div:unknown-pseudo"},
		{name: "invalid nth formula", selector: "span:nth-child(abc)"},
		{name: "deeply nested combinators", selector: "div > span + span ~ span"},
		{name: "whitespace only", selector: "   "},
	}

	ast, _ := ast_domain.ParseAndTransform(context.Background(), html, "test.piko")
	require.NotNil(t, ast)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("QueryAll panicked with selector=%q: %v", tc.selector, r)
				}
			}()

			_, _ = ast_domain.QueryAll(ast, tc.selector, "test.piko")
		})
	}
}

func TestWalk_DepthLimits(t *testing.T) {
	tests := []struct {
		name  string
		depth int
	}{
		{name: "100 deep", depth: 100},
		{name: "500 deep", depth: 500},
		{name: "1000 deep", depth: 1000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Walk panicked at depth %d: %v", tc.depth, r)
				}
			}()

			input := strings.Repeat("<div>", tc.depth) + "x" + strings.Repeat("</div>", tc.depth)
			ast, _ := ast_domain.ParseAndTransform(context.Background(), input, "test.piko")
			require.NotNil(t, ast)

			count := 0
			ast.Walk(func(node *ast_domain.TemplateNode) bool {
				count++
				return true
			})
			assert.Greater(t, count, 0, "Walk should visit nodes")

			it := ast.NewIterator()
			count = 0
			for it.Next() {
				count++
			}
			assert.Greater(t, count, 0, "Iterator should visit nodes")

			clone := ast.DeepClone()
			assert.NotNil(t, clone, "DeepClone should succeed")
		})
	}
}
