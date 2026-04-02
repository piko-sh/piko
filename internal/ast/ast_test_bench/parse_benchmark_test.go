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

func BenchmarkParse_Minimal(b *testing.B) {
	benchmarkParse(b, TemplateMinimal)
}

func BenchmarkParse_Simple(b *testing.B) {
	benchmarkParse(b, TemplateSimple)
}

func BenchmarkParse_Nested(b *testing.B) {
	benchmarkParse(b, TemplateNested)
}

func BenchmarkParse_Attributes(b *testing.B) {
	benchmarkParse(b, TemplateWithAttributes)
}

func BenchmarkParse_Complex(b *testing.B) {
	benchmarkParse(b, TemplateComplex)
}

func BenchmarkParse_ExpressionHeavy(b *testing.B) {
	benchmarkParse(b, TemplateExpressionHeavy)
}

func BenchmarkParse_Mega(b *testing.B) {
	benchmarkParse(b, TemplateMega)
}

func BenchmarkParse_Giga(b *testing.B) {
	benchmarkParse(b, TemplateGiga)
}

func benchmarkParse(b *testing.B, template string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var ast *ast_domain.TemplateAST
	var err error

	for b.Loop() {
		ast, err = ast_domain.Parse(context.Background(), template, "benchmark", nil)
		if err != nil {
			b.Fatal(err)
		}
	}

	SinkAST = ast
}

func BenchmarkParseAndTransform_Minimal(b *testing.B) {
	benchmarkParseAndTransform(b, TemplateMinimal)
}

func BenchmarkParseAndTransform_Simple(b *testing.B) {
	benchmarkParseAndTransform(b, TemplateSimple)
}

func BenchmarkParseAndTransform_Nested(b *testing.B) {
	benchmarkParseAndTransform(b, TemplateNested)
}

func BenchmarkParseAndTransform_Attributes(b *testing.B) {
	benchmarkParseAndTransform(b, TemplateWithAttributes)
}

func BenchmarkParseAndTransform_Complex(b *testing.B) {
	benchmarkParseAndTransform(b, TemplateComplex)
}

func BenchmarkParseAndTransform_ExpressionHeavy(b *testing.B) {
	benchmarkParseAndTransform(b, TemplateExpressionHeavy)
}

func BenchmarkParseAndTransform_Mega(b *testing.B) {
	benchmarkParseAndTransform(b, TemplateMega)
}

func BenchmarkParseAndTransform_Giga(b *testing.B) {
	benchmarkParseAndTransform(b, TemplateGiga)
}

func benchmarkParseAndTransform(b *testing.B, template string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var ast *ast_domain.TemplateAST
	var err error

	for b.Loop() {
		ast, err = ast_domain.ParseAndTransform(context.Background(), template, "benchmark")
		if err != nil {
			b.Fatal(err)
		}
	}

	SinkAST = ast
}

func BenchmarkParseAndTransform_Complex_Parallel(b *testing.B) {
	benchmarkParseAndTransformParallel(b, TemplateComplex)
}

func BenchmarkParseAndTransform_Mega_Parallel(b *testing.B) {
	benchmarkParseAndTransformParallel(b, TemplateMega)
}

func BenchmarkParseAndTransform_Giga_Parallel(b *testing.B) {
	benchmarkParseAndTransformParallel(b, TemplateGiga)
}

func benchmarkParseAndTransformParallel(b *testing.B, template string) {
	b.Helper()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		var ast *ast_domain.TemplateAST
		var err error

		for pb.Next() {
			ast, err = ast_domain.ParseAndTransform(context.Background(), template, "benchmark")
			if err != nil {
				b.Fatal(err)
			}
		}

		SinkAST = ast
	})
}

func BenchmarkValidateAST_Complex(b *testing.B) {
	ast := GetParsedComplex()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ast_domain.ValidateAST(ast)
	}
}

func BenchmarkValidateAST_Mega(b *testing.B) {
	ast := GetParsedMega()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ast_domain.ValidateAST(ast)
	}
}

func BenchmarkValidateAST_Giga(b *testing.B) {
	ast := GetParsedGiga()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ast_domain.ValidateAST(ast)
	}
}

func BenchmarkParseAndTransform_Scaling(b *testing.B) {
	templates := []struct {
		name     string
		template string
	}{
		{name: "Minimal", template: TemplateMinimal},
		{name: "Simple", template: TemplateSimple},
		{name: "Nested", template: TemplateNested},
		{name: "Complex", template: TemplateComplex},
		{name: "Mega", template: TemplateMega},
		{name: "Giga", template: TemplateGiga},
	}

	for _, tc := range templates {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.template)))

			var ast *ast_domain.TemplateAST
			var err error

			for b.Loop() {
				ast, err = ast_domain.ParseAndTransform(context.Background(), tc.template, "benchmark")
				if err != nil {
					b.Fatal(err)
				}
			}

			SinkAST = ast
		})
	}
}

func BenchmarkDeepClone_Complex(b *testing.B) {
	ast := GetParsedComplex()
	b.ReportAllocs()
	b.ResetTimer()

	var cloned *ast_domain.TemplateAST

	for b.Loop() {
		cloned = ast.DeepClone()
	}

	SinkAST = cloned
}

func BenchmarkDeepClone_Mega(b *testing.B) {
	ast := GetParsedMega()
	b.ReportAllocs()
	b.ResetTimer()

	var cloned *ast_domain.TemplateAST

	for b.Loop() {
		cloned = ast.DeepClone()
	}

	SinkAST = cloned
}

func BenchmarkDeepClone_Giga(b *testing.B) {
	ast := GetParsedGiga()
	b.ReportAllocs()
	b.ResetTimer()

	var cloned *ast_domain.TemplateAST

	for b.Loop() {
		cloned = ast.DeepClone()
	}

	SinkAST = cloned
}
