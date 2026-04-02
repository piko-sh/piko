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

//go:build fuzz

package ast_domain

import (
	"context"
	"strings"
	"testing"
)

func FuzzParseAndTransform(f *testing.F) {
	f.Add("<div>Hello</div>")
	f.Add("<div><div><div>x</div></div></div>")
	f.Add("<div><span>")
	f.Add("<div></span></div>")
	f.Add("")
	f.Add("   \n\t  ")
	f.Add("<")
	f.Add(">")
	f.Add("{")
	f.Add("}")
	f.Add("{{ expression")
	f.Add("{{ {{ x }} }}")
	f.Add("<div p-if=\"\">x</div>")
	f.Add("<div p-for=\"\">x</div>")
	f.Add("<div p-if=\"   \">x</div>")
	f.Add("<div p-on>x</div>")
	f.Add("<div p-on:>x</div>")
	f.Add("<div p-bind>x</div>")
	f.Add("<div :>x</div>")
	f.Add("<div p-unknown=\"x\">y</div>")
	f.Add("<div p-text=\"a\" p-html=\"b\">x</div>")
	f.Add("<div p-if=\"a\" p-else>x</div>")
	f.Add("<!--[if mso | IE]><table><![endif]-->")
	f.Add("<!-- comment -->")
	f.Add("<script>const x = '<div>';</script>")
	f.Add("<style>.class { content: '</div>'; }</style>")
	f.Add("<div attr=\"value with 'quotes'\">x</div>")
	f.Add("<div attr='value with \"quotes\"'>x</div>")
	f.Add("<div>中文日本語한국어</div>")
	f.Add("<div>🚀🎉🔥💯</div>")
	f.Add("<div>café résumé naïve</div>")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseAndTransform panicked with input=%q: %v", truncate(input, 100), r)
			}
		}()

		ast, _ := ParseAndTransform(context.Background(), input, "fuzz_test.piko")

		if ast != nil {
			_ = ast.DeepClone()
		}
	})
}

func FuzzParseExpression(f *testing.F) {
	f.Add("x")
	f.Add("1 + 2")
	f.Add("obj.prop")
	f.Add("arr[0]")
	f.Add("fn(x)")
	f.Add("a ? b : c")
	f.Add("!x")
	f.Add("'string'")
	f.Add("123")
	f.Add("true")
	f.Add("false")
	f.Add("nil")
	f.Add("[1, 2, 3]")
	f.Add("{a: 1}")
	f.Add("a?.b")
	f.Add("a ?? b")
	f.Add("item in items")
	f.Add("(i, item) in items")
	f.Add("a + b - c * d / e % f")
	f.Add("a < b < c")
	f.Add("a || b ?? c")
	f.Add("a ? b ? c : d : e")
	f.Add("a +")
	f.Add("+ b")
	f.Add("a + + b")
	f.Add("()")
	f.Add("(a")
	f.Add("[a")
	f.Add("{a:")
	f.Add("a)")
	f.Add("a]")
	f.Add("a}")
	f.Add("[a,,b]")
	f.Add("f(a,,b)")
	f.Add("9223372036854775807")
	f.Add("9223372036854775808")
	f.Add("1.7976931348623157e+308")
	f.Add("1e10000")
	f.Add("-0")
	f.Add("99999999999999999999999999999n")
	f.Add("0.1d + 0.2d")
	f.Add("''")
	f.Add("\"\"")
	f.Add("'it\\'s'")
	f.Add("\"say \\\"hi\\\"\"")
	f.Add("'test\\\\'")
	f.Add("`hello`")
	f.Add("`${x}`")
	f.Add("`outer ${`inner`}`")
	f.Add("`${x")
	f.Add("`${}`")
	f.Add("((((x))))")
	f.Add("[[[[x]]]]")
	f.Add("a.b.c.d.e.f.g")
	f.Add("a?.b?.c?.d?.e")
	f.Add("f(g(h(i(j(x)))))")
	f.Add("_private")
	f.Add("_")
	f.Add("café")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseExpression panicked with input=%q: %v", truncate(input, 100), r)
			}
		}()

		parser := NewExpressionParser(context.Background(), input, "fuzz_test.piko")
		expression, _ := parser.ParseExpression(context.Background())

		if expression != nil {
			_ = expression.Clone()
		}
	})
}

func FuzzEvaluateExpression(f *testing.F) {
	f.Add("x", "hello")
	f.Add("x + 1", "123")
	f.Add("x.foo", "bar")
	f.Add("x[0]", "test")
	f.Add("f(x)", "value")
	f.Add("x == y", "true")
	f.Add("!x", "false")
	f.Add("x ?? y", "")
	f.Add("x ? a : b", "nil")

	f.Fuzz(func(t *testing.T, expressionString, scopeVal string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("EvaluateExpression panicked with expression=%q scope=%q: %v", truncate(expressionString, 50), truncate(scopeVal, 50), r)
			}
		}()

		parser := NewExpressionParser(context.Background(), expressionString, "fuzz_test.piko")
		expression, diagnostics := parser.ParseExpression(context.Background())
		if len(diagnostics) > 0 || expression == nil {
			return
		}

		scopes := []map[string]any{
			{"x": scopeVal},
			{"x": nil},
			{"x": 123},
			{"x": 1.5},
			{"x": true},
			{"x": []any{scopeVal}},
			{"x": map[string]any{"foo": scopeVal}},
			{"x": scopeVal, "y": "other", "a": "alt_a", "b": "alt_b"},
			{"f": func(v any) any { return v }},
		}

		for _, scope := range scopes {
			_ = EvaluateExpression(expression, scope)
		}
	})
}

func FuzzCloneTemplateAST(f *testing.F) {
	f.Add("<div>Hello</div>")
	f.Add("<div p-if=\"x\"><span p-for=\"i in items\">{{ i }}</span></div>")
	f.Add("<div :class=\"classes\" :style=\"styles\">x</div>")
	f.Add("<div p-on:click=\"handle\">Click me</div>")
	f.Add("<template><div>fragment</div></template>")
	f.Add("<!-- comment --><div>text</div>")
	f.Add("<div><div><div><div>deep</div></div></div></div>")
	f.Add("<div a=\"1\" b=\"2\" c=\"3\" d=\"4\" e=\"5\">many attrs</div>")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Clone panicked with input=%q: %v", truncate(input, 100), r)
			}
		}()

		ast, _ := ParseAndTransform(context.Background(), input, "fuzz_test.piko")
		if ast == nil {
			return
		}

		_ = ast.Clone()

		deepClone := ast.DeepClone()
		if deepClone == nil {
			return
		}

		for _, node := range ast.RootNodes {
			if node != nil {
				_ = node.Clone()
				_ = node.DeepClone()
			}
		}
	})
}

func FuzzQueryAll(f *testing.F) {
	f.Add("<div class=\"test\"></div>", "div")
	f.Add("<div class=\"test\"></div>", ".test")
	f.Add("<div id=\"main\"></div>", "#main")
	f.Add("<div data-x=\"y\"></div>", "[data-x]")
	f.Add("<div data-x=\"y\"></div>", "[data-x=y]")
	f.Add("<ul><li></li></ul>", "ul li")
	f.Add("<div><span></span></div>", "div > span")
	f.Add("<div></div><span></span>", "div + span")
	f.Add("<div></div><span></span>", "div ~ span")
	f.Add("<ul><li></li><li></li><li></li></ul>", "li:first-child")
	f.Add("<ul><li></li><li></li><li></li></ul>", "li:last-child")
	f.Add("<ul><li></li><li></li><li></li></ul>", "li:nth-child(2)")
	f.Add("<div></div>", "")
	f.Add("<div></div>", "*")
	f.Add("<div></div>", "unknown-element")
	f.Add("<div></div>", ".")
	f.Add("<div></div>", "#")
	f.Add("<div></div>", "[")
	f.Add("<div></div>", "]")
	f.Add("<div></div>", "::")
	f.Add("<div></div>", "div[")
	f.Add("<div></div>", "div.")
	f.Add("<div></div>", "div#")
	f.Add("<div></div>", "div:unknown-pseudo")

	f.Fuzz(func(t *testing.T, html, selector string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("QueryAll panicked with html=%q selector=%q: %v", truncate(html, 50), truncate(selector, 50), r)
			}
		}()

		ast, _ := ParseAndTransform(context.Background(), html, "fuzz_test.piko")
		if ast == nil {
			return
		}

		_, _ = QueryAll(ast, selector, "fuzz_test.piko")
	})
}

func FuzzWalkTree(f *testing.F) {
	f.Add("<div>text</div>")
	f.Add("<div><span><a>deep</a></span></div>")
	f.Add("<div>{{ expression }}</div>")
	f.Add("<div p-for=\"x in items\">{{ x }}</div>")
	f.Add("<template><div>fragment children</div></template>")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Walk panicked with input=%q: %v", truncate(input, 100), r)
			}
		}()

		ast, _ := ParseAndTransform(context.Background(), input, "fuzz_test.piko")
		if ast == nil {
			return
		}

		count := 0
		ast.Walk(func(node *TemplateNode) bool {
			count++
			return count < 10000
		})

		count = 0
		for node := range ast.Nodes() {
			_ = node.TagName
			count++
			if count > 10000 {
				break
			}
		}

		count = 0
		for node, parent := range ast.NodesWithParent() {
			_, _ = node.TagName, parent
			count++
			if count > 10000 {
				break
			}
		}

		it := ast.NewIterator()
		count = 0
		for it.Next() {
			_ = it.Node.TagName
			count++
			if count > 10000 {
				break
			}
		}

		pit := ast.NewPostOrderIterator()
		count = 0
		for pit.Next() {
			_ = pit.Node.TagName
			count++
			if count > 10000 {
				break
			}
		}
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func FuzzDeeplyNestedParens(f *testing.F) {
	for depth := 1; depth <= 300; depth += 50 {
		expression := strings.Repeat("(", depth) + "x" + strings.Repeat(")", depth)
		f.Add(expression)
	}

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseExpression panicked on deeply nested input (len=%d): %v", len(input), r)
			}
		}()

		parser := NewExpressionParser(context.Background(), input, "fuzz_test.piko")
		_, _ = parser.ParseExpression(context.Background())
	})
}

func FuzzDeeplyNestedHTML(f *testing.F) {
	for depth := 1; depth <= 1100; depth += 200 {
		html := strings.Repeat("<div>", depth) + "x" + strings.Repeat("</div>", depth)
		f.Add(html)
	}

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseAndTransform panicked on deeply nested HTML (len=%d): %v", len(input), r)
			}
		}()

		ast, _ := ParseAndTransform(context.Background(), input, "fuzz_test.piko")
		if ast == nil {
			return
		}

		ast.Walk(func(node *TemplateNode) bool {
			return true
		})
		_ = ast.DeepClone()
	})
}
