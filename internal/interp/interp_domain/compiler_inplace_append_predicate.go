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
	// inPlaceAppendArgCount is the argument count for matched append shapes.
	//
	// Counts the `x = append(x, e)` and `x = append(x, src...)` forms the predicate matches.
	// Mirrors starAppendArgCount in compiler_assign_through.go.
	inPlaceAppendArgCount = 2
)

// matchInPlaceAppendIdentity verifies that the first append argument resolves to the same
// types.Object as the LHS identifier so the rewrite is SSA-equivalent without alias
// analysis.
//
// Conservative: only matches when both sides are bare *ast.Ident references. Compound
// expressions (e.g. selectors, index expressions) are refused and route to the general
// append path.
//
// Takes lhsIdent (*ast.Ident) which is the LHS identifier.
// Takes callRHS (*ast.CallExpr) which is the append call expression.
//
// Returns true when both identifiers reference the same declared object.
func (c *compiler) matchInPlaceAppendIdentity(lhsIdent *ast.Ident, callRHS *ast.CallExpr) bool {
	rhsIdent, ok := callRHS.Args[0].(*ast.Ident)
	if !ok {
		return false
	}
	lhsObject := c.info.ObjectOf(lhsIdent)
	if lhsObject == nil {
		return false
	}
	return lhsObject == c.info.ObjectOf(rhsIdent)
}

// lookupInPlaceAppendTarget resolves a safe in-place append target.
//
// Resolves an LHS identifier to its varLocation and verifies the safety preconditions for
// in-place arena-slot mutation: the location must sit in the general register bank
// (typed-slice banks already mutate the header by value), must not be captured by a
// closure, addressed via `&x`, stored as an upvalue, or spilled.
//
// Takes lhsIdent (*ast.Ident) which is the LHS identifier from the assignment.
//
// Returns varLocation which is the resolved location.
// Returns bool which is true when all safety checks pass.
func (c *compiler) lookupInPlaceAppendTarget(lhsIdent *ast.Ident) (varLocation, bool) {
	location, found := c.scopes.lookupVar(lhsIdent.Name)
	if !found {
		return varLocation{}, false
	}
	if location.kind != registerGeneral {
		return varLocation{}, false
	}
	if location.isCaptured || location.isIndirect || location.isUpvalue || location.isSpilled {
		return varLocation{}, false
	}
	if c.inPlaceAppendAliases != nil && c.inPlaceAppendAliases[lhsIdent.Name] {
		return varLocation{}, false
	}
	return location, true
}

// inPlaceAppendElementKind classifies the append element kind.
//
// Reports byte for []byte appends (the hot path), or the underlying basic-type string for
// other typed elements. Returns the empty string for shapes the predicate cannot classify
// (interface elements, struct elements, etc.); those fall through to the generic
// catch-all in-place opcode (opAppendInPlace). Also reports whether the append uses
// spread (`...`).
//
// Takes callRHS (*ast.CallExpr) which is the append call.
//
// Returns the element kind name, the spread flag, and a bool that is true when the call's
// slice-element type was resolvable.
func (c *compiler) inPlaceAppendElementKind(callRHS *ast.CallExpr) (kindName string, spread bool, ok bool) {
	sliceTypeInfo, found := c.info.Types[callRHS.Args[0]]
	if !found || sliceTypeInfo.Type == nil {
		return "", false, false
	}
	sliceType, sliceOk := sliceTypeInfo.Type.Underlying().(*types.Slice)
	if !sliceOk {
		return "", false, false
	}
	elementName := sliceType.Elem().Underlying().String()
	spread = callRHS.Ellipsis.IsValid()
	return elementName, spread, true
}

// inPlaceAppendCandidate composes the in-place append predicates.
//
// Combines the AST shape match, identity check, target-safety lookup, and element-kind
// classification into a single helper. tryCompileInPlaceAppend calls this before deciding
// which in-place opcode to emit.
//
// Takes leftHandSide (ast.Expr) which is the assignment LHS.
// Takes rightHandSide (ast.Expr) which is the assignment RHS.
//
// Returns inPlaceAppendMatch with ok set when the candidate is safe and classifiable; the
// destination location and element kind describe where and how to compile the in-place
// form.
func (c *compiler) inPlaceAppendCandidate(leftHandSide, rightHandSide ast.Expr) inPlaceAppendMatch {
	lhsIdent, callRHS, shapeOK := matchInPlaceAppendShape(leftHandSide, rightHandSide)
	if !shapeOK {
		return inPlaceAppendMatch{}
	}
	if !c.matchInPlaceAppendIdentity(lhsIdent, callRHS) {
		return inPlaceAppendMatch{}
	}
	location, locationOK := c.lookupInPlaceAppendTarget(lhsIdent)
	if !locationOK {
		return inPlaceAppendMatch{}
	}
	elementKind, spread, kindOK := c.inPlaceAppendElementKind(callRHS)
	if !kindOK {
		return inPlaceAppendMatch{}
	}
	return inPlaceAppendMatch{
		location:    location,
		callRHS:     callRHS,
		elementKind: elementKind,
		spread:      spread,
		ok:          true,
	}
}

// inPlaceAppendMatch is the destructured return of inPlaceAppendCandidate. ok==false
// signals the candidate is unsafe or unclassifiable and the other fields are meaningless.
type inPlaceAppendMatch struct {
	// callRHS is the matched append call expression.
	callRHS *ast.CallExpr

	// elementKind is the slice element's underlying basic-type name.
	elementKind string

	// location is the resolved LHS varLocation.
	location varLocation

	// spread is true when the append uses `...`.
	spread bool

	// ok is true when all preconditions hold.
	ok bool
}

// matchInPlaceAppendShape pattern-matches `x = append(x, ...)` at the AST level. Returns
// the LHS identifier, the RHS append call, and a bool that is true when the shape
// matches.
//
// Counterpart of matchStarAppendByteShape (compiler_assign_through.go) but for the plain
// identifier form rather than the dereferenced pointer form.
//
// Takes leftHandSide (ast.Expr) which is the LHS expression of the assignment.
// Takes rightHandSide (ast.Expr) which is the RHS expression.
//
// Returns the LHS Ident, the RHS CallExpr, and true when matched.
//
//nolint:dupl // per-element-kind specialisation
func matchInPlaceAppendShape(leftHandSide, rightHandSide ast.Expr) (*ast.Ident, *ast.CallExpr, bool) {
	lhsIdent, ok := leftHandSide.(*ast.Ident)
	if !ok {
		return nil, nil, false
	}
	callRHS, ok := rightHandSide.(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}
	if len(callRHS.Args) != inPlaceAppendArgCount {
		return nil, nil, false
	}
	funIdent, ok := callRHS.Fun.(*ast.Ident)
	if !ok || funIdent.Name != "append" {
		return nil, nil, false
	}
	return lhsIdent, callRHS, true
}
