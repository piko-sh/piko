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

package driven_code_emitter_go_literal

import (
	goast "go/ast"
	"go/token"
	"maps"
	"slices"

	"piko.sh/piko/internal/ast/ast_domain"
)

// tryEmitCompositeExpression handles composite expressions such as identifiers,
// object literals, template literals, and array literals.
//
// Takes expression (ast_domain.Expression) which is the expression to emit.
//
// Returns goast.Expr which is the emitted Go expression, or nil if not handled.
// Returns []goast.Stmt which contains any additional statements needed.
// Returns []*ast_domain.Diagnostic which contains any diagnostics generated.
// Returns bool which indicates whether the expression was handled.
func (ee *expressionEmitter) tryEmitCompositeExpression(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic, bool) {
	switch n := expression.(type) {
	case *ast_domain.Identifier:
		goExpr, statements, diagnostics := ee.emitIdentifier(n)
		return goExpr, statements, diagnostics, true
	case *ast_domain.ObjectLiteral:
		goExpr, statements, diagnostics := ee.emitObjectLiteral(n)
		return goExpr, statements, diagnostics, true
	case *ast_domain.TemplateLiteral:
		goExpr, statements, diagnostics := ee.emitTemplateLiteral(n)
		return goExpr, statements, diagnostics, true
	case *ast_domain.ArrayLiteral:
		goExpr, statements, diagnostics := ee.emitArrayLiteral(n)
		return goExpr, statements, diagnostics, true
	}
	return nil, nil, nil, false
}

// emitTemplateLiteral builds code for template literals (e.g. `Hello ${name}!`).
//
// Takes n (*ast_domain.TemplateLiteral) which is the template literal node to
// process.
//
// Returns goast.Expr which is the joined string expression.
// Returns []goast.Stmt which holds any statements needed before the main
// expression from embedded parts.
// Returns []*ast_domain.Diagnostic which holds any problems found while
// processing the template parts.
func (ee *expressionEmitter) emitTemplateLiteral(n *ast_domain.TemplateLiteral) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if len(n.Parts) == 0 {
		return strLit(""), nil, nil
	}

	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic
	var finalGoExpr goast.Expr

	for i, part := range n.Parts {
		var currentPartGoExpr goast.Expr
		if part.IsLiteral {
			currentPartGoExpr = strLit(part.Literal)
		} else {
			var partPrereqs []goast.Stmt
			var partDiags []*ast_domain.Diagnostic
			var nestedGoExpr goast.Expr
			nestedGoExpr, partPrereqs, partDiags = ee.emit(part.Expression)
			allStmts = append(allStmts, partPrereqs...)
			allDiags = append(allDiags, partDiags...)
			currentPartGoExpr = ee.valueToString(nestedGoExpr, getAnnotationFromExpression(part.Expression))
		}

		if i == 0 {
			finalGoExpr = currentPartGoExpr
		} else {
			finalGoExpr = &goast.BinaryExpr{X: finalGoExpr, Op: token.ADD, Y: currentPartGoExpr}
		}
	}

	return finalGoExpr, allStmts, allDiags
}

// emitTemplateLiteralParts extracts each part of a template literal as a
// separate Go expression, without concatenating them. This is used for
// generating variadic calls like BuildClassBytesV(part1, part2, ...) that
// avoid intermediate string allocation from the + operator.
//
// Takes n (*ast_domain.TemplateLiteral) which is the template literal to
// process.
//
// Returns []goast.Expr which contains one Go expression per template part.
// Returns []goast.Stmt which contains any prerequisite statements.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (ee *expressionEmitter) emitTemplateLiteralParts(n *ast_domain.TemplateLiteral) ([]goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if len(n.Parts) == 0 {
		return nil, nil, nil
	}

	parts := make([]goast.Expr, 0, len(n.Parts))
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	for _, part := range n.Parts {
		if part.IsLiteral {
			if part.Literal != "" {
				parts = append(parts, strLit(part.Literal))
			}
		} else {
			nestedGoExpr, partPrereqs, partDiags := ee.emit(part.Expression)
			allStmts = append(allStmts, partPrereqs...)
			allDiags = append(allDiags, partDiags...)
			parts = append(parts, ee.valueToString(nestedGoExpr, getAnnotationFromExpression(part.Expression)))
		}
	}

	return parts, allStmts, allDiags
}

// emitObjectLiteral converts an object literal into a Go map literal.
//
// Takes n (*ast_domain.ObjectLiteral) which is the object literal to convert.
//
// Returns goast.Expr which is the Go map literal.
// Returns []goast.Stmt which contains statements needed for nested values.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (ee *expressionEmitter) emitObjectLiteral(n *ast_domain.ObjectLiteral) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	ann := getAnnotationFromExpression(n)
	var mapType goast.Expr = &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("any")}
	if ann != nil && ann.ResolvedType != nil && ann.ResolvedType.TypeExpression != nil {
		if mt, ok := ann.ResolvedType.TypeExpression.(*goast.MapType); ok {
			mapType = mt
		}
	}

	mapLit := &goast.CompositeLit{Type: mapType, Elts: make([]goast.Expr, 0, len(n.Pairs))}
	keys := slices.Sorted(maps.Keys(n.Pairs))

	for _, key := range keys {
		valueExpr := n.Pairs[key]
		goValueExpr, valueStmts, valueDiags := ee.emit(valueExpr)
		allStmts = append(allStmts, valueStmts...)
		allDiags = append(allDiags, valueDiags...)
		mapLit.Elts = append(mapLit.Elts, &goast.KeyValueExpr{Key: strLit(key), Value: goValueExpr})
	}

	return mapLit, allStmts, allDiags
}

// emitArrayLiteral converts an array literal node (e.g., `[1, 2, 3]`) into Go
// code.
//
// Takes n (*ast_domain.ArrayLiteral) which is the array literal node to
// convert.
//
// Returns goast.Expr which is the Go composite literal expression.
// Returns []goast.Stmt which contains any statements needed by the element
// expressions.
// Returns []*ast_domain.Diagnostic which contains any issues found while
// processing the elements.
func (ee *expressionEmitter) emitArrayLiteral(n *ast_domain.ArrayLiteral) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	ann := getAnnotationFromExpression(n)
	var arrayType goast.Expr = &goast.ArrayType{Elt: cachedIdent("any")}
	if ann != nil && ann.ResolvedType != nil && ann.ResolvedType.TypeExpression != nil {
		if at, ok := ann.ResolvedType.TypeExpression.(*goast.ArrayType); ok {
			arrayType = at
		}
	}

	arrayLit := &goast.CompositeLit{Type: arrayType, Elts: make([]goast.Expr, 0, len(n.Elements))}

	for _, el := range n.Elements {
		goElExpr, elStmts, elDiags := ee.emit(el)
		allStmts = append(allStmts, elStmts...)
		allDiags = append(allDiags, elDiags...)
		arrayLit.Elts = append(arrayLit.Elts, goElExpr)
	}

	return arrayLit, allStmts, allDiags
}
