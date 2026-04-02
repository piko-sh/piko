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

package ast_test_bench

import (
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func BenchmarkQueryLex_Tag(b *testing.B) {
	benchmarkQueryLex(b, QueryTag)
}

func BenchmarkQueryLex_Class(b *testing.B) {
	benchmarkQueryLex(b, QueryClass)
}

func BenchmarkQueryLex_ID(b *testing.B) {
	benchmarkQueryLex(b, QueryID)
}

func BenchmarkQueryLex_Complex(b *testing.B) {
	benchmarkQueryLex(b, QueryComplex)
}

func BenchmarkQueryLex_VeryComplex(b *testing.B) {
	benchmarkQueryLex(b, QueryVeryComplex)
}

func benchmarkQueryLex(b *testing.B, query string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		lexer := ast_domain.NewQueryLexer(query)
		for {
			token := lexer.NextToken()
			if token.Type == ast_domain.TokenEOF || token.Type == ast_domain.TokenIllegal {
				break
			}
		}
		lexer.Release()
	}
}

func BenchmarkQueryParse_Tag(b *testing.B) {
	benchmarkQueryParse(b, QueryTag)
}

func BenchmarkQueryParse_Class(b *testing.B) {
	benchmarkQueryParse(b, QueryClass)
}

func BenchmarkQueryParse_TagWithClass(b *testing.B) {
	benchmarkQueryParse(b, QueryTagWithClass)
}

func BenchmarkQueryParse_MultipleClasses(b *testing.B) {
	benchmarkQueryParse(b, QueryMultipleClasses)
}

func BenchmarkQueryParse_Attribute(b *testing.B) {
	benchmarkQueryParse(b, QueryAttribute)
}

func BenchmarkQueryParse_AttributeValue(b *testing.B) {
	benchmarkQueryParse(b, QueryAttributeValue)
}

func BenchmarkQueryParse_Descendant(b *testing.B) {
	benchmarkQueryParse(b, QueryDescendant)
}

func BenchmarkQueryParse_Child(b *testing.B) {
	benchmarkQueryParse(b, QueryChild)
}

func BenchmarkQueryParse_PseudoClass(b *testing.B) {
	benchmarkQueryParse(b, QueryPseudoClass)
}

func BenchmarkQueryParse_PseudoClassNot(b *testing.B) {
	benchmarkQueryParse(b, QueryPseudoClassNot)
}

func BenchmarkQueryParse_Complex(b *testing.B) {
	benchmarkQueryParse(b, QueryComplex)
}

func BenchmarkQueryParse_MultipleSelectors(b *testing.B) {
	benchmarkQueryParse(b, QueryMultipleSelectors)
}

func BenchmarkQueryParse_VeryComplex(b *testing.B) {
	benchmarkQueryParse(b, QueryVeryComplex)
}

func benchmarkQueryParse(b *testing.B, query string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var selector ast_domain.SelectorSet

	for b.Loop() {
		lexer := ast_domain.NewQueryLexer(query)
		parser := ast_domain.NewQueryParser(lexer, "benchmark")
		selector = parser.Parse()
		parser.Release()
		lexer.Release()
	}

	SinkSelector = selector
}

func BenchmarkQueryAll_Simple_Tag(b *testing.B) {
	benchmarkQueryAll(b, GetParsedSimple(), QueryTag)
}

func BenchmarkQueryAll_Simple_Class(b *testing.B) {
	benchmarkQueryAll(b, GetParsedSimple(), QueryClass)
}

func BenchmarkQueryAll_Complex_Tag(b *testing.B) {
	benchmarkQueryAll(b, GetParsedComplex(), QueryTag)
}

func BenchmarkQueryAll_Complex_Class(b *testing.B) {
	benchmarkQueryAll(b, GetParsedComplex(), QueryClass)
}

func BenchmarkQueryAll_Complex_Descendant(b *testing.B) {
	benchmarkQueryAll(b, GetParsedComplex(), QueryDescendant)
}

func BenchmarkQueryAll_Complex_Child(b *testing.B) {
	benchmarkQueryAll(b, GetParsedComplex(), QueryChild)
}

func BenchmarkQueryAll_Complex_Attribute(b *testing.B) {
	benchmarkQueryAll(b, GetParsedComplex(), `[p-if]`)
}

func BenchmarkQueryAll_Mega_Tag(b *testing.B) {
	benchmarkQueryAll(b, GetParsedMega(), "div")
}

func BenchmarkQueryAll_Mega_Class(b *testing.B) {
	benchmarkQueryAll(b, GetParsedMega(), ".stat-card")
}

func BenchmarkQueryAll_Mega_Descendant(b *testing.B) {
	benchmarkQueryAll(b, GetParsedMega(), "section div")
}

func BenchmarkQueryAll_Mega_Complex(b *testing.B) {
	benchmarkQueryAll(b, GetParsedMega(), "main.dashboard-content > section table tbody tr")
}

func BenchmarkQueryAll_Giga_Tag(b *testing.B) {
	benchmarkQueryAll(b, GetParsedGiga(), "div")
}

func BenchmarkQueryAll_Giga_Class(b *testing.B) {
	benchmarkQueryAll(b, GetParsedGiga(), ".feed-item")
}

func BenchmarkQueryAll_Giga_Descendant(b *testing.B) {
	benchmarkQueryAll(b, GetParsedGiga(), "main section div")
}

func BenchmarkQueryAll_Giga_AttributeSelector(b *testing.B) {
	benchmarkQueryAll(b, GetParsedGiga(), "[p-for]")
}

func BenchmarkQueryAll_Giga_MultipleSelectors(b *testing.B) {
	benchmarkQueryAll(b, GetParsedGiga(), "h1, h2, h3, h4, h5, h6")
}

func benchmarkQueryAll(b *testing.B, ast *ast_domain.TemplateAST, query string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var results []*ast_domain.TemplateNode

	for b.Loop() {
		results, _ = ast_domain.QueryAll(ast, query, "benchmark")
	}

	SinkNodes = results
}

func BenchmarkQueryParse_Scaling(b *testing.B) {
	queries := []struct {
		name  string
		query string
	}{
		{name: "Tag", query: QueryTag},
		{name: "Class", query: QueryClass},
		{name: "ID", query: QueryID},
		{name: "TagWithClass", query: QueryTagWithClass},
		{name: "Attribute", query: QueryAttribute},
		{name: "Descendant", query: QueryDescendant},
		{name: "Child", query: QueryChild},
		{name: "PseudoClass", query: QueryPseudoClass},
		{name: "Complex", query: QueryComplex},
		{name: "VeryComplex", query: QueryVeryComplex},
	}

	for _, tc := range queries {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.query)))
			benchmarkQueryParse(b, tc.query)
		})
	}
}

func BenchmarkQueryAll_ASTSizeScaling(b *testing.B) {
	asts := []struct {
		ast  *ast_domain.TemplateAST
		name string
	}{
		{name: "Simple", ast: GetParsedSimple()},
		{name: "Complex", ast: GetParsedComplex()},
		{name: "Mega", ast: GetParsedMega()},
		{name: "Giga", ast: GetParsedGiga()},
	}

	query := "div"

	for _, tc := range asts {
		nodeCount := CountNodes(tc.ast)
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(nodeCount), "nodes")
			benchmarkQueryAll(b, tc.ast, query)
		})
	}
}

func BenchmarkQueryAll_Parallel_Mega(b *testing.B) {
	ast := GetParsedMega()
	query := "div.stat-card"
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		var results []*ast_domain.TemplateNode

		for pb.Next() {
			results, _ = ast_domain.QueryAll(ast, query, "benchmark")
		}

		SinkNodes = results
	})
}

func BenchmarkQueryAll_Parallel_Giga(b *testing.B) {
	ast := GetParsedGiga()
	query := "div.feed-item"
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		var results []*ast_domain.TemplateNode

		for pb.Next() {
			results, _ = ast_domain.QueryAll(ast, query, "benchmark")
		}

		SinkNodes = results
	})
}

func BenchmarkQueryAll_Giga_Cached(b *testing.B) {
	ast := GetParsedGiga()
	query := "div"
	b.ReportAllocs()

	ast_domain.QueryAll(ast, query, "warmup")

	b.ResetTimer()

	var results []*ast_domain.TemplateNode
	for b.Loop() {
		results, _ = ast_domain.QueryAll(ast, query, "benchmark")
	}

	SinkNodes = results
}

func BenchmarkQueryAll_Giga_Uncached(b *testing.B) {
	ast := GetParsedGiga()
	query := "div"
	b.ReportAllocs()
	b.ResetTimer()

	var results []*ast_domain.TemplateNode
	for b.Loop() {

		ast.InvalidateQueryContext()
		results, _ = ast_domain.QueryAll(ast, query, "benchmark")
	}

	SinkNodes = results
}

func BenchmarkQueryAll_Giga_FirstQueryCost(b *testing.B) {
	ast := GetParsedGiga()
	query := "div"
	b.ReportAllocs()
	b.ResetTimer()

	var results []*ast_domain.TemplateNode
	for b.Loop() {

		ast.InvalidateQueryContext()
		results, _ = ast_domain.QueryAll(ast, query, "benchmark")
	}

	SinkNodes = results
}

func BenchmarkQueryAll_Mega_Cached(b *testing.B) {
	ast := GetParsedMega()
	query := "div"
	b.ReportAllocs()

	ast_domain.QueryAll(ast, query, "warmup")

	b.ResetTimer()

	var results []*ast_domain.TemplateNode
	for b.Loop() {
		results, _ = ast_domain.QueryAll(ast, query, "benchmark")
	}

	SinkNodes = results
}

func BenchmarkQueryAll_Mega_Uncached(b *testing.B) {
	ast := GetParsedMega()
	query := "div"
	b.ReportAllocs()
	b.ResetTimer()

	var results []*ast_domain.TemplateNode
	for b.Loop() {
		ast.InvalidateQueryContext()
		results, _ = ast_domain.QueryAll(ast, query, "benchmark")
	}

	SinkNodes = results
}

func BenchmarkQueryAll_PremailerSimulation_50Rules(b *testing.B) {
	ast := GetParsedGiga()

	selectors := []string{
		"div", "p", "span", "a", "img",
		"h1", "h2", "h3", "h4", "h5",
		"table", "tr", "td", "th", "tbody",
		".feed-item", ".header", ".footer", ".content", ".sidebar",
		"#main", "#container", "#wrapper", "#header", "#footer",
		"div.content", "p.intro", "span.highlight", "a.button", "img.logo",
		"table.data", "tr.odd", "td.value", "th.header", "tbody.items",
		"[p-for]", "[p-if]", "[p-show]", "[class]", "[id]",
		"div > p", "main section", "header nav", "footer ul", "aside div",
		"h1, h2, h3", "p, span", ".a, .b", "div, span, p", "table, div",
	}

	b.ReportAllocs()
	b.ResetTimer()

	var totalMatches int
	for b.Loop() {

		ast.InvalidateQueryContext()
		totalMatches = 0

		for _, selector := range selectors {
			results, _ := ast_domain.QueryAll(ast, selector, "premailer")
			totalMatches += len(results)
		}
	}

	SinkInt = totalMatches
}

func BenchmarkQueryAll_PremailerSimulation_10Rules(b *testing.B) {
	ast := GetParsedMega()

	selectors := []string{
		"div", "p", "span", "a", "table",
		".stat-card", "#main", "div > p", "h1, h2", "[p-if]",
	}

	b.ReportAllocs()
	b.ResetTimer()

	var totalMatches int
	for b.Loop() {
		ast.InvalidateQueryContext()
		totalMatches = 0

		for _, selector := range selectors {
			results, _ := ast_domain.QueryAll(ast, selector, "premailer")
			totalMatches += len(results)
		}
	}

	SinkInt = totalMatches
}
