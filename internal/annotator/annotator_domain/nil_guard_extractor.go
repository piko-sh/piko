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

package annotator_domain

// Extracts nil guards from p-if condition expressions to enable compile-time tracking
// of known non-nil values within guarded blocks, suppressing false positive warnings.

import (
	goast "go/ast"

	"piko.sh/piko/internal/ast/ast_domain"
)

// ExtractNilGuardsFromCondition analyses a condition expression and returns
// the string forms of expressions that are guaranteed non-nil when the
// condition is true.
//
// Supports patterns like:
//   - expr != nil
//   - expr !== nil
//   - !(expr == nil)
//   - a != nil && b != nil
//   - bare truthiness check on pointer types (if annotations are present)
//
// Takes condExpr (ast_domain.Expression) which is the condition to analyse.
//
// Returns []string which contains the string forms of guarded expressions.
func ExtractNilGuardsFromCondition(condExpr ast_domain.Expression) []string {
	if condExpr == nil {
		return nil
	}
	var guards []string
	extractGuards(condExpr, &guards)
	return guards
}

// extractGuards walks an expression and appends nil guards to the list.
//
// Takes expression (ast_domain.Expression) which is the expression to examine.
// Takes guards (*[]string) which collects the guards found.
func extractGuards(expression ast_domain.Expression, guards *[]string) {
	switch e := expression.(type) {
	case *ast_domain.BinaryExpression:
		handleBinaryExpr(e, guards)

	case *ast_domain.UnaryExpression:
		handleUnaryExpr(e, guards)

	case *ast_domain.MemberExpression, *ast_domain.Identifier:
		handleTruthinessCheck(expression, guards)
	}
}

// handleBinaryExpr checks a binary expression for nil guard patterns.
//
// Takes e (*ast_domain.BinaryExpression) which is the binary expression to check.
// Takes guards (*[]string) which collects the names of guarded variables.
func handleBinaryExpr(e *ast_domain.BinaryExpression, guards *[]string) {
	if e.Operator == ast_domain.OpNe || e.Operator == ast_domain.OpLooseNe {
		if isNilLiteral(e.Right) {
			*guards = append(*guards, e.Left.String())
		} else if isNilLiteral(e.Left) {
			*guards = append(*guards, e.Right.String())
		}
	}
	if e.Operator == ast_domain.OpAnd {
		extractGuards(e.Left, guards)
		extractGuards(e.Right, guards)
	}
}

// handleUnaryExpr processes unary expressions for nil guard patterns.
//
// Takes e (*ast_domain.UnaryExpression) which is the unary expression to check.
// Takes guards (*[]string) which collects the names of guarded variables.
func handleUnaryExpr(e *ast_domain.UnaryExpression, guards *[]string) {
	if e.Operator != ast_domain.OpNot {
		return
	}
	binary, ok := e.Right.(*ast_domain.BinaryExpression)
	if !ok {
		return
	}
	if binary.Operator != ast_domain.OpEq && binary.Operator != ast_domain.OpLooseEq {
		return
	}
	if isNilLiteral(binary.Right) {
		*guards = append(*guards, binary.Left.String())
	} else if isNilLiteral(binary.Left) {
		*guards = append(*guards, binary.Right.String())
	}
}

// handleTruthinessCheck handles bare expressions used as boolean conditions.
// If the expression has been annotated and is a pointer type, it implies a
// nil check.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
// Takes guards (*[]string) which collects any discovered guards.
func handleTruthinessCheck(expression ast_domain.Expression, guards *[]string) {
	ann := expression.GetGoAnnotation()
	if ann != nil && ann.ResolvedType != nil {
		if isPointerTypeExpr(ann.ResolvedType.TypeExpression) {
			*guards = append(*guards, expression.String())
		}
	}
}

// isNilLiteral checks whether an expression is the nil literal.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if the expression is a nil literal.
func isNilLiteral(expression ast_domain.Expression) bool {
	_, ok := expression.(*ast_domain.NilLiteral)
	return ok
}

// isPointerTypeExpr checks whether a Go AST type expression is a pointer type.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is a pointer (StarExpr).
func isPointerTypeExpr(typeExpr goast.Expr) bool {
	_, ok := typeExpr.(*goast.StarExpr)
	return ok
}
