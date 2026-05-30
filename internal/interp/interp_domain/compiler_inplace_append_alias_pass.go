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
	"go/ast"
	"go/types"
)

const (
	// blankIdentifier is Go's reserved name for "ignore this binding".
	blankIdentifier = "_"
)

// collectInPlaceAppendAliases finds slice locals that may alias.
//
// Produces the set of local variable names whose slice reflect.Value may be aliased to
// another register at some point during execution of the function body.
// lookupInPlaceAppendTarget consults the set to refuse in-place append emission for
// slices that may have an outstanding alias, because the in-place opcode mutates the
// arenaSliceHeader slot and any aliased register would observe that mutation, breaking
// Go's slice-header value semantics.
//
// Alias-creation is recognised when a variable appears as the RHS of an *ast.AssignStmt
// (both `saved := output` and `saved = output`, since slice moves emit opMoveGeneral in
// moveGeneralModeAlias and both registers end up pointing at the same arenaSliceHeader
// slot), or when it is the operand of an `&x` address-of expression. Compound expressions
// like `x.field` or `x[i]` do not move the header and are ignored, as are read-only
// `len`/`cap`/`x[i]`/`x[a:b]`, returns, and call-argument passing (handled receiver-side
// by unconditionally marking slice-typed parameters).
//
// The pass is scope-naive: a name marked aliased anywhere in the function is marked
// aliased throughout, which is conservative but safe. Cost is one ast.Inspect pass over
// the body plus the parameter list, bounded by AST node count.
//
// Takes typeContext (*compiler) which provides go/types information for parameter
// classification.
// Takes parameters (*ast.FieldList) which is the function signature's parameter list, or
// nil if the function has none.
// Takes body (*ast.BlockStmt) which is the function body. May be nil for
// declared-but-not-defined functions.
//
// Returns the set of aliased names, or nil when neither the body nor the parameters
// contributed any names. nil signals "no restrictions"; the predicate treats nil and
// empty-map identically.
func collectInPlaceAppendAliases(typeContext *compiler, parameters *ast.FieldList, body *ast.BlockStmt) map[string]bool {
	aliased := make(map[string]bool)

	collectInPlaceAppendAliasesFromParameters(typeContext, parameters, aliased)
	if body != nil {
		ast.Inspect(body, func(node ast.Node) bool {
			collectInPlaceAppendAliasesFromNode(node, aliased)
			return true
		})
	}

	if len(aliased) == 0 {
		return nil
	}
	return aliased
}

// collectInPlaceAppendAliasesFromParameters marks slice parameters.
//
// Adds every slice-typed parameter to the alias set. Parameters receive their slice
// header via the call ABI's value-copy-for-boundary path which, for slices, is the
// alias-fast path: the parameter register shares the arenaSliceHeader slot with the
// caller's argument. Mutating that slot in place inside the callee would corrupt the
// caller's view.
//
// Takes typeContext (*compiler) for types.Info access. nil-safe.
// Takes parameters (*ast.FieldList) which may be nil.
// Takes aliased (map[string]bool) into which slice-typed parameter names are inserted.
func collectInPlaceAppendAliasesFromParameters(typeContext *compiler, parameters *ast.FieldList, aliased map[string]bool) {
	if parameters == nil {
		return
	}
	for _, field := range parameters.List {
		if !inPlaceAppendFieldIsSlice(typeContext, field) {
			continue
		}
		for _, name := range field.Names {
			if name == nil || name.Name == blankIdentifier {
				continue
			}
			aliased[name.Name] = true
		}
	}
}

// inPlaceAppendFieldIsSlice reports whether the function-parameter field declares a slice
// type. Uses go/types when available; otherwise inspects the AST type expression for
// *ast.ArrayType with no length (which is Go's source-level form for slice types).
//
// Takes typeContext (*compiler) which provides types.Info; nil-safe.
// Takes field (*ast.Field) which is one parameter declaration.
//
// Returns true when the field's declared type is a slice.
func inPlaceAppendFieldIsSlice(typeContext *compiler, field *ast.Field) bool {
	if typeContext != nil && typeContext.info != nil && field.Type != nil {
		if typeInfo, ok := typeContext.info.Types[field.Type]; ok && typeInfo.Type != nil {
			_, isSlice := typeInfo.Type.Underlying().(*types.Slice)
			return isSlice
		}
	}
	arrayType, ok := field.Type.(*ast.ArrayType)
	if !ok {
		return false
	}
	return arrayType.Len == nil
}

// collectInPlaceAppendAliasesFromNode records alias-creation observations for a single
// AST node. Called from ast.Inspect on every node in the function body.
//
// Takes node (ast.Node) which is the candidate AST node.
// Takes aliased (map[string]bool) which accumulates alias names.
func collectInPlaceAppendAliasesFromNode(node ast.Node, aliased map[string]bool) {
	switch typed := node.(type) {
	case *ast.AssignStmt:
		recordInPlaceAppendAssignRHS(typed, aliased)
	case *ast.UnaryExpr:
		recordInPlaceAppendAddressOf(typed, aliased)
	case *ast.ValueSpec:
		recordInPlaceAppendVarDeclValues(typed, aliased)
	}
}

// recordInPlaceAppendAssignRHS marks bare-identifier RHS values aliased.
//
// For multi-assign (`a, b := x, y`) each Rhs element is examined independently. Compound
// expressions like `x.field` or `x[i]` are not Idents so they are not recorded; they do
// not move the slice header.
//
// Takes statement (*ast.AssignStmt) which is the assignment node.
// Takes aliased (map[string]bool) which accumulates alias names.
func recordInPlaceAppendAssignRHS(statement *ast.AssignStmt, aliased map[string]bool) {
	for _, expression := range statement.Rhs {
		ident, ok := expression.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "" || ident.Name == blankIdentifier {
			continue
		}
		aliased[ident.Name] = true
	}
}

// recordInPlaceAppendAddressOf marks the target of an `&x` expression. The address-of
// itself heap-promotes the variable via piko's escape analysis (varLocation.isIndirect is
// set), which the in-place predicate already inspects; this is a defensive
// belt-and-braces capture in case the predicate is called before the indirect flag has
// been set on the location.
//
// Takes expression (*ast.UnaryExpr) which is the candidate node.
// Takes aliased (map[string]bool) which accumulates alias names.
func recordInPlaceAppendAddressOf(expression *ast.UnaryExpr, aliased map[string]bool) {
	if expression.Op.String() != "&" {
		return
	}
	ident, ok := expression.X.(*ast.Ident)
	if !ok {
		return
	}
	if ident.Name == "" || ident.Name == blankIdentifier {
		return
	}
	aliased[ident.Name] = true
}

// recordInPlaceAppendVarDeclValues handles `var y = x` and similar nested var
// declarations whose initialiser is a bare Ident. Same alias-creation semantics as `y :=
// x` but a different AST shape.
//
// Takes spec (*ast.ValueSpec) which is the var-decl spec node.
// Takes aliased (map[string]bool) which accumulates alias names.
func recordInPlaceAppendVarDeclValues(spec *ast.ValueSpec, aliased map[string]bool) {
	for _, value := range spec.Values {
		ident, ok := value.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "" || ident.Name == blankIdentifier {
			continue
		}
		aliased[ident.Name] = true
	}
}
