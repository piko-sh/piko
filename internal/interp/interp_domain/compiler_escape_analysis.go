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

package interp_domain

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"slices"
)

// escapeAnalysisVerdict captures a single local's escape status from the per-function AST
// walk. CompiledFunction stores only the per-PC outcome for runtime consumption.
type escapeAnalysisVerdict struct {
	// reasonCode names the first rule that classified the local as escaping. Unset when
	// escapes is false.
	reasonCode string

	// escapes is true when any use of the local violates a conservative rule (returned,
	// stored in a heap-anchored slot, passed to a callee, captured by an escaping closure,
	// taken more than once, or aliased into another variable). The default classifies
	// "anything not provably safe" as escaping.
	escapes bool
}

// escapeWalkState carries the cumulative state of a single classifyLocalEscape AST walk.
// The visit method serves as the ast.Inspect callback, delegating to focused per-node
// helpers.
type escapeWalkState struct {
	// name is the local being classified.
	name string

	// reasonCode names the first rule that produced an escape verdict. Empty when escapes is
	// false.
	reasonCode string

	// addressOfCount counts &name occurrences seen so far; a second occurrence triggers the
	// multiple-address-of escape rule.
	addressOfCount int

	// escapes is true once any rule has fired; further visits short-circuit.
	escapes bool
}

// visit dispatches on the node kind, delegating each case to a focused helper.
//
// Takes node (ast.Node) which is the AST node visited by the walk.
//
// Returns false once an escape verdict is reached to short-circuit the walk, true to
// continue.
func (state *escapeWalkState) visit(node ast.Node) bool {
	if state.escapes {
		return false
	}
	switch typed := node.(type) {
	case *ast.UnaryExpr:
		return state.visitUnaryExpr(typed)
	case *ast.ReturnStmt:
		return state.visitReturnStmt(typed)
	case *ast.FuncLit:
		return state.visitFuncLit(typed)
	}
	return true
}

// visitUnaryExpr applies the address-of rules.
//
// &name appearing more than once, or in any non-immediate-deref context, escapes.
//
// Takes expression (*ast.UnaryExpr) which is the unary expression under inspection.
//
// Returns false once an escape verdict is reached, true to continue the walk.
func (state *escapeWalkState) visitUnaryExpr(expression *ast.UnaryExpr) bool {
	if expression.Op != token.AND {
		return true
	}
	rootIdent := rootIdentForMutation(expression.X)
	if rootIdent == nil || rootIdent.Name != state.name {
		return true
	}
	state.addressOfCount++
	if state.addressOfCount > 1 {
		state.escapes = true
		state.reasonCode = "multiple-address-of"
		return false
	}
	state.escapes = true
	state.reasonCode = "address-of-non-deref"
	return false
}

// visitReturnStmt flags the local as escaping when it appears in any result expression of
// a return statement.
//
// Takes statement (*ast.ReturnStmt) which is the return statement under inspection.
//
// Returns false once an escape verdict is reached, true to continue the walk.
func (state *escapeWalkState) visitReturnStmt(statement *ast.ReturnStmt) bool {
	for _, result := range statement.Results {
		if usesIdent(result, state.name) {
			state.escapes = true
			state.reasonCode = "returned"
			return false
		}
	}
	return true
}

// visitFuncLit flags the local as escaping when any closure literal references it, since
// closure literals can outlive the parent frame.
//
// Takes literal (*ast.FuncLit) which is the function literal under inspection.
//
// Returns false once an escape verdict is reached, true to continue the walk.
func (state *escapeWalkState) visitFuncLit(literal *ast.FuncLit) bool {
	if usesIdent(literal.Body, state.name) {
		state.escapes = true
		state.reasonCode = "closure-capture"
		return false
	}
	return true
}

// classifyLocalEscapes populates cf.arenaSafeAllocPCs with the PCs of opAllocIndirect
// sites whose target local is statically proven not to escape cf's frame.
//
// Operates on cf's source AST plus the compiler's per-name emit-PC map (recorded by
// promoteToIndirect at emit time). heapPromotedNames names the candidate locals.
//
// Takes ctx (context.Context) which cancels the scan.
// Takes cf (*CompiledFunction) which receives the arenaSafeAllocPCs update; cf.body must
// be finalised so PCs are stable.
// Takes body (*ast.BlockStmt) which is the function body to scan.
// Takes allocSitePCs (map[string]int) which maps each heap-promoted local name to the PC
// of its opAllocIndirect emit site.
// Takes heapPromotedNames (map[string]bool) which lists candidate locals from the
// heap-promotion pre-pass.
//
// Returns error when the context is cancelled mid-scan.
func classifyLocalEscapes(
	ctx context.Context,
	cf *CompiledFunction,
	body *ast.BlockStmt,
	allocSitePCs map[string]int,
	heapPromotedNames map[string]bool,
) error {
	if cf == nil || body == nil || len(allocSitePCs) == 0 || len(heapPromotedNames) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("classifyLocalEscapes cancelled: %w", err)
	}
	for name, pc := range allocSitePCs {
		if !heapPromotedNames[name] {
			continue
		}
		verdict := classifyLocalEscape(name, body)
		if verdict.escapes {
			continue
		}
		if cf.arenaSafeAllocPCs == nil {
			cf.arenaSafeAllocPCs = make(map[int]bool)
		}
		cf.arenaSafeAllocPCs[pc] = true
	}
	return nil
}

// classifyLocalEscape returns the escape verdict for a single named local by walking the
// function body's AST. The verdict's reasonCode names the first rule that triggered the
// escape conclusion.
//
// Conservative rules treat name as escaping if it is declared at more than one site in
// the body (shadowing: the heap-promotion path keys allocSitePCs and the AST walk by bare
// name, so colliding names cannot be distinguished without full go/types object-identity
// resolution), if &name appears more than once (aliasing risk), if &name appears in any
// context other than a direct `*(&name)` deref, or if name appears as a top-level
// identifier in a return statement, a heap-anchored store, or any other escape-relevant
// context. The default verdict is "escapes"; the function only returns {escapes: false}
// when none of those triggers.
//
// Takes name (string) which is the local being classified, and body (*ast.BlockStmt)
// which is the function body AST.
//
// Returns the escape verdict for name in body.
func classifyLocalEscape(name string, body *ast.BlockStmt) escapeAnalysisVerdict {
	if nameDeclaredMoreThanOnce(body, name) {
		return escapeAnalysisVerdict{escapes: true, reasonCode: "shadowed-name"}
	}
	state := escapeWalkState{name: name}
	ast.Inspect(body, state.visit)
	if state.escapes {
		return escapeAnalysisVerdict{escapes: true, reasonCode: state.reasonCode}
	}
	return escapeAnalysisVerdict{escapes: false}
}

// nameDeclaredMoreThanOnce reports whether the identifier name is introduced by more than
// one declaration site within body.
//
// A declaration introduces a fresh variable; two such sites for the same name mean the
// name is shadowed and the bare-name escape walk cannot tell the variables apart.
// Recognised declaration sites are short variable declarations (`:=`) on the LHS,
// var/const value specifications, for-range key and value identifiers using `:=`, and
// type-switch guard bindings. Function-literal parameter and result names nested inside
// body are also counted, because a closure parameter named the same as the promoted local
// is a genuine shadow within the walked subtree. The count saturates at two; the caller
// only needs the "more than one" predicate.
//
// Takes body (*ast.BlockStmt) which is the function body AST, and name (string) which is
// the local name being checked.
//
// Returns true when at least two declaration sites introduce name.
func nameDeclaredMoreThanOnce(body *ast.BlockStmt, name string) bool {
	count := 0
	tally := func(ident *ast.Ident) bool {
		if ident == nil || ident.Name != name {
			return false
		}
		count++
		return count >= 2
	}
	stop := false
	ast.Inspect(body, func(node ast.Node) bool {
		if stop {
			return false
		}
		if countDeclarationSite(node, tally) {
			stop = true
			return false
		}
		return true
	})
	return count >= 2
}

// countDeclarationSite feeds every declaring identifier of node to tally, returning true
// as soon as tally reports the saturation threshold was reached.
//
// Takes node (ast.Node) which is the AST node under inspection, and tally
// (func(*ast.Ident) bool) which records a declaring identifier and returns true once the
// count threshold is met.
//
// Returns true when tally signalled the threshold; false otherwise.
func countDeclarationSite(node ast.Node, tally func(*ast.Ident) bool) bool {
	switch typed := node.(type) {
	case *ast.AssignStmt:
		return countDefineAssignDeclarations(typed, tally)
	case *ast.ValueSpec:
		return slices.ContainsFunc(typed.Names, tally)
	case *ast.RangeStmt:
		return countRangeDeclarations(typed, tally)
	case *ast.TypeSwitchStmt:
		return countTypeSwitchDeclarations(typed, tally)
	case *ast.FuncLit:
		return countFuncLitParamDeclarations(typed, tally)
	}
	return false
}

// countDefineAssignDeclarations feeds every left-hand-side identifier of a short variable
// declaration (`:=`) to tally. Plain assignments declare nothing and are skipped.
//
// Takes assign (*ast.AssignStmt) which is the candidate assignment, and tally
// (func(*ast.Ident) bool) which records a declaring identifier.
//
// Returns true when tally signalled the saturation threshold.
func countDefineAssignDeclarations(assign *ast.AssignStmt, tally func(*ast.Ident) bool) bool {
	if assign.Tok != token.DEFINE {
		return false
	}
	return countIdentExprs(assign.Lhs, tally)
}

// countRangeDeclarations feeds the key and value identifiers of a range statement to
// tally when the statement declares them with `:=`.
//
// Takes statement (*ast.RangeStmt) which is the range statement, and tally
// (func(*ast.Ident) bool) which records a declaring identifier.
//
// Returns true when tally signalled the saturation threshold.
func countRangeDeclarations(statement *ast.RangeStmt, tally func(*ast.Ident) bool) bool {
	if statement.Tok != token.DEFINE {
		return false
	}
	return countIdentExprs([]ast.Expr{statement.Key, statement.Value}, tally)
}

// countTypeSwitchDeclarations feeds the bound identifier of a type switch guard (`switch
// v := x.(type)`) to tally.
//
// Takes statement (*ast.TypeSwitchStmt) which is the type switch, and tally
// (func(*ast.Ident) bool) which records a declaring identifier.
//
// Returns true when tally signalled the saturation threshold.
func countTypeSwitchDeclarations(statement *ast.TypeSwitchStmt, tally func(*ast.Ident) bool) bool {
	assign, ok := statement.Assign.(*ast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE {
		return false
	}
	return countIdentExprs(assign.Lhs, tally)
}

// countIdentExprs feeds every expression in exprs that is a plain identifier to tally,
// ignoring non-identifier expressions.
//
// Takes exprs ([]ast.Expr) which is the expression list to scan, and tally
// (func(*ast.Ident) bool) which records a declaring identifier.
//
// Returns true when tally signalled the saturation threshold.
func countIdentExprs(exprs []ast.Expr, tally func(*ast.Ident) bool) bool {
	for _, expr := range exprs {
		if ident, ok := expr.(*ast.Ident); ok && tally(ident) {
			return true
		}
	}
	return false
}

// countFuncLitParamDeclarations feeds every parameter and result name of a function
// literal to tally.
//
// A closure parameter or named result sharing the promoted local's name shadows it within
// the literal's body, which the bare-name escape walk would otherwise misattribute.
//
// Takes literal (*ast.FuncLit) which is the function literal, and tally (func(*ast.Ident)
// bool) which records a declaring identifier.
//
// Returns true when tally signalled the saturation threshold.
func countFuncLitParamDeclarations(literal *ast.FuncLit, tally func(*ast.Ident) bool) bool {
	if literal.Type == nil {
		return false
	}
	for _, fieldList := range []*ast.FieldList{literal.Type.Params, literal.Type.Results} {
		if fieldList == nil {
			continue
		}
		for _, field := range fieldList.List {
			if slices.ContainsFunc(field.Names, tally) {
				return true
			}
		}
	}
	return false
}

// usesIdent reports whether node's subtree contains any identifier reference to name.
// Bare identifiers, selector receivers, index targets, and call function expressions all
// count.
//
// Takes node (ast.Node) which is the AST subtree to scan, and name (string) which is the
// identifier to look for.
//
// Returns true when name appears at least once in node.
func usesIdent(node ast.Node, name string) bool {
	if node == nil {
		return false
	}
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}
