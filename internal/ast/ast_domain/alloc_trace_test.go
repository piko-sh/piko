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

package ast_domain

import (
	"context"
	"fmt"
	"testing"
)

func TestTraceAllocations(t *testing.T) {
	t.Parallel()

	expression := `(user.isAdmin || (user.permissions?.canEdit && resource.ownerId == user.id)) && !resource.isLocked ? performAction(resource.id, { notify: true }) : showError('Access denied')`

	ctx := context.Background()
	p := NewExpressionParser(ctx, expression, "test")
	ast, diagnostics := p.ParseExpression(ctx)
	p.Release()

	if len(diagnostics) > 0 {
		t.Fatalf("Parse failed: %v", diagnostics)
	}

	counts := make(map[string]int)
	countNodes(ast, counts)

	fmt.Println("\n=== AST NODE ALLOCATIONS ===")
	total := 0
	for nodeType, count := range counts {
		fmt.Printf("  %s: %d\n", nodeType, count)
		total += count
	}
	fmt.Printf("\nTotal AST nodes: %d\n", total)
}

func countNodes(expression Expression, counts map[string]int) {
	if expression == nil {
		return
	}

	switch e := expression.(type) {
	case *Identifier:
		counts["Identifier"]++
	case *MemberExpression:
		counts["MemberExpr"]++
		countNodes(e.Base, counts)
		countNodes(e.Property, counts)
	case *IndexExpression:
		counts["IndexExpr"]++
		countNodes(e.Base, counts)
		countNodes(e.Index, counts)
	case *CallExpression:
		counts["CallExpr"]++
		countNodes(e.Callee, counts)
		for _, argument := range e.Args {
			countNodes(argument, counts)
		}
	case *BinaryExpression:
		counts["BinaryExpr"]++
		countNodes(e.Left, counts)
		countNodes(e.Right, counts)
	case *UnaryExpression:
		counts["UnaryExpr"]++
		countNodes(e.Right, counts)
	case *TernaryExpression:
		counts["TernaryExpr"]++
		countNodes(e.Condition, counts)
		countNodes(e.Consequent, counts)
		countNodes(e.Alternate, counts)
	case *ObjectLiteral:
		counts["ObjectLiteral"]++
		counts["ObjectLiteral.map"]++
		for _, v := range e.Pairs {
			countNodes(v, counts)
		}
	case *ArrayLiteral:
		counts["ArrayLiteral"]++
		for _, element := range e.Elements {
			countNodes(element, counts)
		}
	case *StringLiteral:
		counts["StringLiteral"]++
	case *IntegerLiteral:
		counts["IntegerLiteral"]++
	case *FloatLiteral:
		counts["FloatLiteral"]++
	case *BooleanLiteral:
		counts["BooleanLiteral"]++
	case *NilLiteral:
		counts["NilLiteral"]++
	default:
		counts[fmt.Sprintf("Unknown(%T)", e)]++
	}
}
