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
	"context"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func BenchmarkExpression_Identifier(b *testing.B) {
	benchmarkExpressionParse(b, ExprIdentifier)
}

func BenchmarkExpression_MemberAccess(b *testing.B) {
	benchmarkExpressionParse(b, ExprMemberAccess)
}

func BenchmarkExpression_IndexAccess(b *testing.B) {
	benchmarkExpressionParse(b, ExprIndexAccess)
}

func BenchmarkExpression_OptionalChain(b *testing.B) {
	benchmarkExpressionParse(b, ExprOptionalChain)
}

func BenchmarkExpression_FunctionCall(b *testing.B) {
	benchmarkExpressionParse(b, ExprFunctionCall)
}

func BenchmarkExpression_FunctionCallMultiArg(b *testing.B) {
	benchmarkExpressionParse(b, ExprFunctionCallMultiArg)
}

func BenchmarkExpression_BinarySimple(b *testing.B) {
	benchmarkExpressionParse(b, ExprBinarySimple)
}

func BenchmarkExpression_BinaryComplex(b *testing.B) {
	benchmarkExpressionParse(b, ExprBinaryComplex)
}

func BenchmarkExpression_Ternary(b *testing.B) {
	benchmarkExpressionParse(b, ExprTernary)
}

func BenchmarkExpression_TernaryNested(b *testing.B) {
	benchmarkExpressionParse(b, ExprTernaryNested)
}

func BenchmarkExpression_ArrayLiteral(b *testing.B) {
	benchmarkExpressionParse(b, ExprArrayLiteral)
}

func BenchmarkExpression_ObjectLiteral(b *testing.B) {
	benchmarkExpressionParse(b, ExprObjectLiteral)
}

func BenchmarkExpression_ObjectLiteralNested(b *testing.B) {
	benchmarkExpressionParse(b, ExprObjectLiteralNested)
}

func BenchmarkExpression_TemplateLiteral(b *testing.B) {
	benchmarkExpressionParse(b, ExprTemplateLiteral)
}

func BenchmarkExpression_ForIn(b *testing.B) {
	benchmarkExpressionParse(b, ExprForIn)
}

func BenchmarkExpression_Complex(b *testing.B) {
	benchmarkExpressionParse(b, ExprComplex)
}

func BenchmarkExpression_DeepMemberChain(b *testing.B) {
	benchmarkExpressionParse(b, ExprDeepMemberChain)
}

func BenchmarkExpression_MathHeavy(b *testing.B) {
	benchmarkExpressionParse(b, ExprMathHeavy)
}

func BenchmarkExpression_ParserReuse_Simple(b *testing.B) {
	benchmarkExpressionParseWithReuse(b, ExprIdentifier)
}

func BenchmarkExpression_ParserReuse_Complex(b *testing.B) {
	benchmarkExpressionParseWithReuse(b, ExprComplex)
}

func BenchmarkExpression_ParserFresh_Simple(b *testing.B) {
	benchmarkExpressionParseWithFresh(b, ExprIdentifier)
}

func BenchmarkExpression_ParserFresh_Complex(b *testing.B) {
	benchmarkExpressionParseWithFresh(b, ExprComplex)
}

func BenchmarkExpression_Parallel_Simple(b *testing.B) {
	benchmarkExpressionParseParallel(b, ExprMemberAccess)
}

func BenchmarkExpression_Parallel_Complex(b *testing.B) {
	benchmarkExpressionParseParallel(b, ExprComplex)
}

func BenchmarkExpression_Scaling(b *testing.B) {
	expressions := []struct {
		name       string
		expression string
	}{
		{name: "Identifier", expression: ExprIdentifier},
		{name: "MemberAccess", expression: ExprMemberAccess},
		{name: "IndexAccess", expression: ExprIndexAccess},
		{name: "FunctionCall", expression: ExprFunctionCall},
		{name: "BinarySimple", expression: ExprBinarySimple},
		{name: "BinaryComplex", expression: ExprBinaryComplex},
		{name: "Ternary", expression: ExprTernary},
		{name: "ArrayLiteral", expression: ExprArrayLiteral},
		{name: "ObjectLiteral", expression: ExprObjectLiteral},
		{name: "Complex", expression: ExprComplex},
	}

	for _, tc := range expressions {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.expression)))
			benchmarkExpressionParse(b, tc.expression)
		})
	}
}

func benchmarkExpressionParse(b *testing.B, expression string) {
	b.Helper()
	b.ReportAllocs()

	parser := ast_domain.NewExpressionParser(context.Background(), "", "benchmark")
	b.ResetTimer()

	var parsed ast_domain.Expression

	for b.Loop() {
		parser.Reset(context.Background(), expression, "benchmark")
		parsed, _ = parser.ParseExpression(context.Background())
	}

	SinkExpr = parsed
}

func benchmarkExpressionParseWithReuse(b *testing.B, expression string) {
	b.Helper()
	b.ReportAllocs()

	parser := ast_domain.NewExpressionParser(context.Background(), "", "benchmark")
	b.ResetTimer()

	var parsed ast_domain.Expression

	for b.Loop() {
		parser.Reset(context.Background(), expression, "benchmark")
		parsed, _ = parser.ParseExpression(context.Background())
	}

	SinkExpr = parsed
}

func benchmarkExpressionParseWithFresh(b *testing.B, expression string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var parsed ast_domain.Expression

	for b.Loop() {
		parser := ast_domain.NewExpressionParser(context.Background(), expression, "benchmark")
		parsed, _ = parser.ParseExpression(context.Background())
		parser.Release()
	}

	SinkExpr = parsed
}

func benchmarkExpressionParseParallel(b *testing.B, expression string) {
	b.Helper()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		parser := ast_domain.NewExpressionParser(context.Background(), "", "benchmark")
		var parsed ast_domain.Expression

		for pb.Next() {
			parser.Reset(context.Background(), expression, "benchmark")
			parsed, _ = parser.ParseExpression(context.Background())
		}

		SinkExpr = parsed
	})
}

func BenchmarkExpressionClone_Identifier(b *testing.B) {
	parser := ast_domain.NewExpressionParser(context.Background(), ExprIdentifier, "benchmark")
	expression, _ := parser.ParseExpression(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	var cloned ast_domain.Expression

	for b.Loop() {
		cloned = expression.Clone()
	}

	SinkExpr = cloned
}

func BenchmarkExpressionClone_Complex(b *testing.B) {
	parser := ast_domain.NewExpressionParser(context.Background(), ExprComplex, "benchmark")
	expression, _ := parser.ParseExpression(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	var cloned ast_domain.Expression

	for b.Loop() {
		cloned = expression.Clone()
	}

	SinkExpr = cloned
}

func BenchmarkExpressionString_Identifier(b *testing.B) {
	parser := ast_domain.NewExpressionParser(context.Background(), ExprIdentifier, "benchmark")
	expression, _ := parser.ParseExpression(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	var str string

	for b.Loop() {
		str = expression.String()
	}

	_ = str
}

func BenchmarkExpressionString_Complex(b *testing.B) {
	parser := ast_domain.NewExpressionParser(context.Background(), ExprComplex, "benchmark")
	expression, _ := parser.ParseExpression(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	var str string

	for b.Loop() {
		str = expression.String()
	}

	_ = str
}

func BenchmarkExpressionCached_Simple(b *testing.B) {

	ast_domain.ClearExpressionCache()

	b.ReportAllocs()
	b.ResetTimer()

	var parsed ast_domain.Expression

	for b.Loop() {
		parsed, _ = ast_domain.ParseExpressionCached(context.Background(), ExprIdentifier, "benchmark")
	}

	SinkExpr = parsed
}

func BenchmarkExpressionCached_Complex(b *testing.B) {

	ast_domain.ClearExpressionCache()

	b.ReportAllocs()
	b.ResetTimer()

	var parsed ast_domain.Expression

	for b.Loop() {
		parsed, _ = ast_domain.ParseExpressionCached(context.Background(), ExprComplex, "benchmark")
	}

	SinkExpr = parsed
}
