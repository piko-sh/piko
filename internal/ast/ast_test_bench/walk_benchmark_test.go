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

func BenchmarkWalk_Callback_Simple(b *testing.B) {
	benchmarkWalkCallback(b, GetParsedSimple())
}

func BenchmarkWalk_Callback_Complex(b *testing.B) {
	benchmarkWalkCallback(b, GetParsedComplex())
}

func BenchmarkWalk_Callback_Mega(b *testing.B) {
	benchmarkWalkCallback(b, GetParsedMega())
}

func BenchmarkWalk_Callback_Giga(b *testing.B) {
	benchmarkWalkCallback(b, GetParsedGiga())
}

func benchmarkWalkCallback(b *testing.B, ast *ast_domain.TemplateAST) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		ast.Walk(func(node *ast_domain.TemplateNode) bool {
			count++
			return true
		})
	}

	SinkInt = count
}

func BenchmarkWalk_Nodes_Simple(b *testing.B) {
	benchmarkWalkNodes(b, GetParsedSimple())
}

func BenchmarkWalk_Nodes_Complex(b *testing.B) {
	benchmarkWalkNodes(b, GetParsedComplex())
}

func BenchmarkWalk_Nodes_Mega(b *testing.B) {
	benchmarkWalkNodes(b, GetParsedMega())
}

func BenchmarkWalk_Nodes_Giga(b *testing.B) {
	benchmarkWalkNodes(b, GetParsedGiga())
}

func benchmarkWalkNodes(b *testing.B, ast *ast_domain.TemplateAST) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		for range ast.Nodes() {
			count++
		}
	}

	SinkInt = count
}

func BenchmarkWalk_NodesWithParent_Simple(b *testing.B) {
	benchmarkWalkNodesWithParent(b, GetParsedSimple())
}

func BenchmarkWalk_NodesWithParent_Complex(b *testing.B) {
	benchmarkWalkNodesWithParent(b, GetParsedComplex())
}

func BenchmarkWalk_NodesWithParent_Mega(b *testing.B) {
	benchmarkWalkNodesWithParent(b, GetParsedMega())
}

func BenchmarkWalk_NodesWithParent_Giga(b *testing.B) {
	benchmarkWalkNodesWithParent(b, GetParsedGiga())
}

func benchmarkWalkNodesWithParent(b *testing.B, ast *ast_domain.TemplateAST) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		for range ast.NodesWithParent() {
			count++
		}
	}

	SinkInt = count
}

func BenchmarkWalk_Iterator_Simple(b *testing.B) {
	benchmarkWalkIterator(b, GetParsedSimple())
}

func BenchmarkWalk_Iterator_Complex(b *testing.B) {
	benchmarkWalkIterator(b, GetParsedComplex())
}

func BenchmarkWalk_Iterator_Mega(b *testing.B) {
	benchmarkWalkIterator(b, GetParsedMega())
}

func BenchmarkWalk_Iterator_Giga(b *testing.B) {
	benchmarkWalkIterator(b, GetParsedGiga())
}

func benchmarkWalkIterator(b *testing.B, ast *ast_domain.TemplateAST) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		it := ast.NewIterator()
		for it.Next() {
			count++
		}
	}

	SinkInt = count
}

func BenchmarkWalk_PostOrderIterator_Simple(b *testing.B) {
	benchmarkWalkPostOrderIterator(b, GetParsedSimple())
}

func BenchmarkWalk_PostOrderIterator_Complex(b *testing.B) {
	benchmarkWalkPostOrderIterator(b, GetParsedComplex())
}

func BenchmarkWalk_PostOrderIterator_Mega(b *testing.B) {
	benchmarkWalkPostOrderIterator(b, GetParsedMega())
}

func BenchmarkWalk_PostOrderIterator_Giga(b *testing.B) {
	benchmarkWalkPostOrderIterator(b, GetParsedGiga())
}

func benchmarkWalkPostOrderIterator(b *testing.B, ast *ast_domain.TemplateAST) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		it := ast.NewPostOrderIterator()
		for it.Next() {
			count++
		}
	}

	SinkInt = count
}

type benchmarkVisitor struct {
	count int
}

func (v *benchmarkVisitor) Enter(_ context.Context, node *ast_domain.TemplateNode) (ast_domain.Visitor, error) {
	v.count++
	return v, nil
}

func (v *benchmarkVisitor) Exit(_ context.Context, node *ast_domain.TemplateNode) error {
	return nil
}

func BenchmarkWalk_Visitor_Simple(b *testing.B) {
	benchmarkWalkVisitor(b, GetParsedSimple())
}

func BenchmarkWalk_Visitor_Complex(b *testing.B) {
	benchmarkWalkVisitor(b, GetParsedComplex())
}

func BenchmarkWalk_Visitor_Mega(b *testing.B) {
	benchmarkWalkVisitor(b, GetParsedMega())
}

func BenchmarkWalk_Visitor_Giga(b *testing.B) {
	benchmarkWalkVisitor(b, GetParsedGiga())
}

func benchmarkWalkVisitor(b *testing.B, ast *ast_domain.TemplateAST) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	visitor := &benchmarkVisitor{}

	for b.Loop() {
		visitor.count = 0
		_ = ast.Accept(context.Background(), visitor)
	}

	SinkInt = visitor.count
}

func BenchmarkWalk_Parallel_Complex_1Worker(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedComplex(), 1)
}

func BenchmarkWalk_Parallel_Complex_2Workers(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedComplex(), 2)
}

func BenchmarkWalk_Parallel_Complex_4Workers(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedComplex(), 4)
}

func BenchmarkWalk_Parallel_Mega_1Worker(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedMega(), 1)
}

func BenchmarkWalk_Parallel_Mega_2Workers(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedMega(), 2)
}

func BenchmarkWalk_Parallel_Mega_4Workers(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedMega(), 4)
}

func BenchmarkWalk_Parallel_Giga_1Worker(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedGiga(), 1)
}

func BenchmarkWalk_Parallel_Giga_2Workers(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedGiga(), 2)
}

func BenchmarkWalk_Parallel_Giga_4Workers(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedGiga(), 4)
}

func BenchmarkWalk_Parallel_Giga_8Workers(b *testing.B) {
	benchmarkWalkParallel(b, GetParsedGiga(), 8)
}

func benchmarkWalkParallel(b *testing.B, ast *ast_domain.TemplateAST, workers int) {
	b.Helper()
	b.ReportAllocs()

	ctx := context.Background()
	b.ResetTimer()

	for b.Loop() {
		_ = ast.ParallelWalk(ctx, workers, func(ctx context.Context, node *ast_domain.TemplateNode) error {

			_ = node.TagName
			return nil
		})
	}
}

func BenchmarkWalk_Stream_Simple(b *testing.B) {
	benchmarkWalkStream(b, GetParsedSimple())
}

func BenchmarkWalk_Stream_Complex(b *testing.B) {
	benchmarkWalkStream(b, GetParsedComplex())
}

func BenchmarkWalk_Stream_Mega(b *testing.B) {
	benchmarkWalkStream(b, GetParsedMega())
}

func benchmarkWalkStream(b *testing.B, ast *ast_domain.TemplateAST) {
	b.Helper()
	b.ReportAllocs()

	ctx := context.Background()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		for range ast.StreamNodes(ctx) {
			count++
		}
	}

	SinkInt = count
}

func BenchmarkWalk_Find_Complex(b *testing.B) {
	ast := GetParsedComplex()
	b.ReportAllocs()
	b.ResetTimer()

	var found *ast_domain.TemplateNode

	for b.Loop() {
		found = ast.Find(func(node *ast_domain.TemplateNode) bool {
			return node.TagName == "div"
		})
	}

	SinkNode = found
}

func BenchmarkWalk_Find_Mega(b *testing.B) {
	ast := GetParsedMega()
	b.ReportAllocs()
	b.ResetTimer()

	var found *ast_domain.TemplateNode

	for b.Loop() {
		found = ast.Find(func(node *ast_domain.TemplateNode) bool {
			return node.TagName == "table"
		})
	}

	SinkNode = found
}

func BenchmarkWalk_FindAll_Complex(b *testing.B) {
	ast := GetParsedComplex()
	b.ReportAllocs()
	b.ResetTimer()

	var found []*ast_domain.TemplateNode

	for b.Loop() {
		found = ast.FindAll(func(node *ast_domain.TemplateNode) bool {
			return node.TagName == "div"
		})
	}

	SinkNodes = found
}

func BenchmarkWalk_FindAll_Mega(b *testing.B) {
	ast := GetParsedMega()
	b.ReportAllocs()
	b.ResetTimer()

	var found []*ast_domain.TemplateNode

	for b.Loop() {
		found = ast.FindAll(func(node *ast_domain.TemplateNode) bool {
			return node.TagName == "div"
		})
	}

	SinkNodes = found
}

func BenchmarkWalk_NodeExpressions_Complex(b *testing.B) {
	ast := GetParsedComplex()
	b.ReportAllocs()
	b.ResetTimer()

	var expressionCount int

	for b.Loop() {
		expressionCount = 0
		ast.Walk(func(node *ast_domain.TemplateNode) bool {
			ast_domain.WalkNodeExpressions(node, func(expression ast_domain.Expression) {
				expressionCount++
			})
			return true
		})
	}

	SinkInt = expressionCount
}

func BenchmarkWalk_NodeExpressions_Mega(b *testing.B) {
	ast := GetParsedMega()
	b.ReportAllocs()
	b.ResetTimer()

	var expressionCount int

	for b.Loop() {
		expressionCount = 0
		ast.Walk(func(node *ast_domain.TemplateNode) bool {
			ast_domain.WalkNodeExpressions(node, func(expression ast_domain.Expression) {
				expressionCount++
			})
			return true
		})
	}

	SinkInt = expressionCount
}

func BenchmarkWalk_VisitExpression_Simple(b *testing.B) {
	parser := ast_domain.NewExpressionParser(context.Background(), ExprMemberAccess, "benchmark")
	expression, _ := parser.ParseExpression(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		ast_domain.VisitExpression(expression, func(e ast_domain.Expression) bool {
			count++
			return true
		})
	}

	SinkInt = count
}

func BenchmarkWalk_VisitExpression_Complex(b *testing.B) {
	parser := ast_domain.NewExpressionParser(context.Background(), ExprComplex, "benchmark")
	expression, _ := parser.ParseExpression(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		ast_domain.VisitExpression(expression, func(e ast_domain.Expression) bool {
			count++
			return true
		})
	}

	SinkInt = count
}

func BenchmarkWalk_Comparison_Mega(b *testing.B) {
	ast := GetParsedMega()
	nodeCount := CountNodes(ast)

	b.Run("Callback", func(b *testing.B) {
		b.ReportAllocs()
		b.ReportMetric(float64(nodeCount), "nodes")
		benchmarkWalkCallback(b, ast)
	})

	b.Run("Nodes", func(b *testing.B) {
		b.ReportAllocs()
		b.ReportMetric(float64(nodeCount), "nodes")
		benchmarkWalkNodes(b, ast)
	})

	b.Run("NodesWithParent", func(b *testing.B) {
		b.ReportAllocs()
		b.ReportMetric(float64(nodeCount), "nodes")
		benchmarkWalkNodesWithParent(b, ast)
	})

	b.Run("Iterator", func(b *testing.B) {
		b.ReportAllocs()
		b.ReportMetric(float64(nodeCount), "nodes")
		benchmarkWalkIterator(b, ast)
	})

	b.Run("PostOrderIterator", func(b *testing.B) {
		b.ReportAllocs()
		b.ReportMetric(float64(nodeCount), "nodes")
		benchmarkWalkPostOrderIterator(b, ast)
	})

	b.Run("Visitor", func(b *testing.B) {
		b.ReportAllocs()
		b.ReportMetric(float64(nodeCount), "nodes")
		benchmarkWalkVisitor(b, ast)
	})
}

func BenchmarkWalk_Scaling_Callback(b *testing.B) {
	asts := []struct {
		ast  *ast_domain.TemplateAST
		name string
	}{
		{name: "Simple", ast: GetParsedSimple()},
		{name: "Complex", ast: GetParsedComplex()},
		{name: "Mega", ast: GetParsedMega()},
		{name: "Giga", ast: GetParsedGiga()},
	}

	for _, tc := range asts {
		nodeCount := CountNodes(tc.ast)
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(nodeCount), "nodes")
			benchmarkWalkCallback(b, tc.ast)
		})
	}
}

func BenchmarkWalk_Scaling_Iterator(b *testing.B) {
	asts := []struct {
		ast  *ast_domain.TemplateAST
		name string
	}{
		{name: "Simple", ast: GetParsedSimple()},
		{name: "Complex", ast: GetParsedComplex()},
		{name: "Mega", ast: GetParsedMega()},
		{name: "Giga", ast: GetParsedGiga()},
	}

	for _, tc := range asts {
		nodeCount := CountNodes(tc.ast)
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(nodeCount), "nodes")
			benchmarkWalkIterator(b, tc.ast)
		})
	}
}

func BenchmarkWalk_ParallelScaling_Giga(b *testing.B) {
	ast := GetParsedGiga()

	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(b.Name()+"_"+string(rune('0'+workers))+"Workers", func(b *testing.B) {
			b.ReportAllocs()
			benchmarkWalkParallel(b, ast, workers)
		})
	}
}

func BenchmarkWalk_IteratorSkipChildren_Mega(b *testing.B) {
	ast := GetParsedMega()
	b.ReportAllocs()
	b.ResetTimer()

	var count int

	for b.Loop() {
		count = 0
		it := ast.NewIterator()
		for it.Next() {
			count++

			if len(it.Node.Children) > 5 {
				it.SkipChildren()
			}
		}
	}

	SinkInt = count
}
