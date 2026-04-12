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
	"go/constant"
	"go/token"
	"go/types"
)

const (
	// maxFingerprintStmtCount caps the body statement count stored in the
	// forStmtFingerprint. Bodies with more statements saturate at this value and are
	// unlikely to match any recogniser anyway.
	maxFingerprintStmtCount = 255
)

const (
	// forStmtKeyExprUpperBound indexes the loop's upper-bound key expression slot.
	forStmtKeyExprUpperBound = iota

	// forStmtKeyExprAssignLHS indexes the body assignment LHS slot.
	forStmtKeyExprAssignLHS

	// forStmtKeyExprAssignRHS indexes the body assignment RHS slot.
	forStmtKeyExprAssignRHS

	// forStmtKeyExprBinaryLHS indexes the body binary-expression LHS slot.
	forStmtKeyExprBinaryLHS

	// forStmtKeyExprBinaryRHS indexes the body binary-expression RHS slot.
	forStmtKeyExprBinaryRHS

	// forStmtKeyExprScalarOperand indexes the body scalar operand slot.
	forStmtKeyExprScalarOperand //nolint:unused // documented enum slot retained for ABI stability
)

// classifyForStmt produces a coarse shape-fingerprint for the loop.
//
// Walks statement once. The recogniser registry uses the fingerprint to dispatch
// candidates in O(1). The classifier is intentionally permissive: it produces
// fingerprints for shapes that no recogniser may ultimately accept. Recognisers do the
// precise per-pattern validation (types, aliasing, constant proofs) in Match. The
// classifier never raises errors: every for-statement produces some fingerprint, even if
// every shape field is its "Other" fallback. That keeps the dispatcher branch-free.
//
// Takes statement (*ast.ForStmt) which is the loop to classify.
// Takes info (*types.Info) which provides constant-folded values for condition bounds.
// May be nil; constUpperBound stays zero when info cannot resolve a constant.
//
// Returns the populated forStmtFingerprint.
func classifyForStmt(statement *ast.ForStmt, info *types.Info) forStmtFingerprint {
	var fingerprint forStmtFingerprint
	fingerprint.initShape = classifyForStmtInit(statement.Init)
	fingerprint.condShape = classifyForStmtCond(statement.Cond, info, &fingerprint)
	fingerprint.postShape = classifyForStmtPost(statement.Post)
	classifyForStmtBody(statement.Body, &fingerprint)
	return fingerprint
}

// classifyForStmtInit maps the init clause to its shape enum.
//
// Takes init (ast.Stmt) which is the loop init clause (may be nil).
//
// Returns the matched forStmtInitShape.
func classifyForStmtInit(init ast.Stmt) forStmtInitShape {
	if init == nil {
		return forInitNone
	}
	assign, ok := init.(*ast.AssignStmt)
	if !ok {
		return forInitOther
	}
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return forInitOther
	}
	if _, ok := assign.Lhs[0].(*ast.Ident); !ok {
		return forInitOther
	}
	if !isIntegerLiteralZero(assign.Rhs[0]) {
		return forInitOther
	}
	switch assign.Tok {
	case token.DEFINE:
		return forInitConstZeroDecl
	case token.ASSIGN:
		return forInitConstZeroAssign
	default:
		return forInitOther
	}
}

// classifyForStmtCond maps the cond clause to its shape enum and populates the
// upper-bound key expression (and constant value when applicable) on fingerprint.
//
// Takes cond (ast.Expr) which is the loop condition (may be nil).
// Takes info (*types.Info) which resolves constant bounds; may be nil, in which case
// constant bounds degrade to forCondLtConst / forCondLeConst only when the bound is a
// bare *ast.BasicLit.
// Takes fingerprint (*forStmtFingerprint) which receives the upper-bound key expression
// and (for constant bounds) the constant value.
//
// Returns the matched forStmtCondShape.
func classifyForStmtCond(cond ast.Expr, info *types.Info, fingerprint *forStmtFingerprint) forStmtCondShape {
	if cond == nil {
		return forCondOther
	}
	binary, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return forCondOther
	}
	if _, ok := binary.X.(*ast.Ident); !ok {
		return forCondOther
	}
	if _, lenOk := matchLenCall(binary.Y); lenOk {
		fingerprint.keyExprs[forStmtKeyExprUpperBound] = binary.Y
		switch binary.Op {
		case token.LSS:
			return forCondLtLen
		case token.LEQ:
			return forCondLeLen
		default:
		}
		return forCondOther
	}
	if value, ok := resolveIntegerConstant(binary.Y, info); ok {
		fingerprint.keyExprs[forStmtKeyExprUpperBound] = binary.Y
		fingerprint.constUpperBound = value
		switch binary.Op {
		case token.LSS:
			return forCondLtConst
		case token.LEQ:
			return forCondLeConst
		default:
		}
		return forCondOther
	}
	return forCondOther
}

// classifyForStmtPost maps the post clause to its shape enum.
//
// Takes post (ast.Stmt) which is the loop post clause (may be nil).
//
// Returns the matched forStmtPostShape.
func classifyForStmtPost(post ast.Stmt) forStmtPostShape {
	incDec, ok := post.(*ast.IncDecStmt)
	if !ok {
		return forPostOther
	}
	if _, ok := incDec.X.(*ast.Ident); !ok {
		return forPostOther
	}
	switch incDec.Tok {
	case token.INC:
		return forPostPlusPlus
	case token.DEC:
		return forPostMinusMinus
	default:
	}
	return forPostOther
}

// classifyForStmtBody walks the loop body once, decides its coarse shape, and populates
// fingerprint with the body shape, statement count, and any key expressions the chosen
// shape exposes.
//
// Takes body (*ast.BlockStmt) which is the loop body (may be nil).
// Takes fingerprint (*forStmtFingerprint) which is populated in place.
func classifyForStmtBody(body *ast.BlockStmt, fingerprint *forStmtFingerprint) {
	if body == nil {
		fingerprint.bodyShape = forBodyEmpty
		return
	}
	stmtCount := len(body.List)
	if stmtCount > maxFingerprintStmtCount {
		fingerprint.bodyStmtCount = maxFingerprintStmtCount
	} else {
		fingerprint.bodyStmtCount = uint8(stmtCount) //nolint:gosec // bounded by maxFingerprintStmtCount above
	}
	if stmtCount == 0 {
		fingerprint.bodyShape = forBodyEmpty
		return
	}
	if stmtCount == 1 {
		switch only := body.List[0].(type) {
		case *ast.AssignStmt:
			fingerprint.bodyShape = classifyForBodyAssign(only, fingerprint)
			return
		case *ast.IfStmt:
			fingerprint.bodyShape = classifyForBodyIf(only, fingerprint)
			return
		}
	}
	fingerprint.bodyShape = forBodyOther
}

// classifyForBodyAssign classifies a single-assignment loop body and extracts its operand
// key expressions.
//
// Takes assign (*ast.AssignStmt) which is the body's sole statement.
// Takes fingerprint (*forStmtFingerprint) which receives the extracted
// destination/source/binary-operand key expressions.
//
// Returns the matched forStmtBodyShape.
func classifyForBodyAssign(assign *ast.AssignStmt, fingerprint *forStmtFingerprint) forStmtBodyShape {
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return forBodyOther
	}
	fingerprint.keyExprs[forStmtKeyExprAssignLHS] = assign.Lhs[0]
	fingerprint.keyExprs[forStmtKeyExprAssignRHS] = assign.Rhs[0]
	switch rhs := assign.Rhs[0].(type) {
	case *ast.BinaryExpr:
		_, leftIsIndex := rhs.X.(*ast.IndexExpr)
		_, rightIsIndex := rhs.Y.(*ast.IndexExpr)
		if leftIsIndex && rightIsIndex {
			fingerprint.keyExprs[forStmtKeyExprBinaryLHS] = rhs.X
			fingerprint.keyExprs[forStmtKeyExprBinaryRHS] = rhs.Y
			return forBodySingleAssignBinaryIndexIndex
		}
		if leftIsIndex || rightIsIndex {
			fingerprint.keyExprs[forStmtKeyExprBinaryLHS] = rhs.X
			fingerprint.keyExprs[forStmtKeyExprBinaryRHS] = rhs.Y
			return forBodySingleAssignBinaryIndex
		}
		return forBodySingleAssign
	case *ast.IndexExpr:
		return forBodySingleAssignIndex
	}
	return forBodySingleAssign
}

// classifyForBodyIf classifies a single-if loop body.
//
// Recognises the canonical max/min shape `if a[i] > m { m = a[i] }` (or its < flip).
// Bodies that match get key expressions populated; non-matching shapes fall through to
// forBodyOther.
//
// Takes ifStmt (*ast.IfStmt) which is the body's sole statement.
// Takes fingerprint (*forStmtFingerprint) which receives the extracted
// operand/destination key expressions.
//
// Returns the matched forStmtBodyShape.
func classifyForBodyIf(ifStmt *ast.IfStmt, fingerprint *forStmtFingerprint) forStmtBodyShape {
	if ifStmt.Init != nil || ifStmt.Else != nil {
		return forBodyOther
	}
	cond, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return forBodyOther
	}
	switch cond.Op {
	case token.GTR, token.LSS, token.GEQ, token.LEQ:
	default:
		return forBodyOther
	}
	if _, ok := cond.X.(*ast.IndexExpr); !ok {
		return forBodyOther
	}
	if _, ok := cond.Y.(*ast.Ident); !ok {
		return forBodyOther
	}
	if ifStmt.Body == nil || len(ifStmt.Body.List) != 1 {
		return forBodyOther
	}
	bodyAssign, ok := ifStmt.Body.List[0].(*ast.AssignStmt)
	if !ok {
		return forBodyOther
	}
	if len(bodyAssign.Lhs) != 1 || len(bodyAssign.Rhs) != 1 || bodyAssign.Tok != token.ASSIGN {
		return forBodyOther
	}
	fingerprint.keyExprs[forStmtKeyExprAssignLHS] = bodyAssign.Lhs[0]
	fingerprint.keyExprs[forStmtKeyExprAssignRHS] = bodyAssign.Rhs[0]
	fingerprint.keyExprs[forStmtKeyExprBinaryLHS] = cond.X
	fingerprint.keyExprs[forStmtKeyExprBinaryRHS] = cond.Y
	return forBodySingleIfMaxMin
}

// isIntegerLiteralZero reports whether expr is the integer literal 0 (the only init RHS
// classifyForStmtInit accepts). Constant-folded expressions that evaluate to zero are
// intentionally NOT accepted: the classifier stays cheap and shape-driven; recognisers
// needing stronger predicates use info.Types directly in Match.
//
// Takes expr (ast.Expr) which is the candidate expression.
//
// Returns true when expr is literally "0".
func isIntegerLiteralZero(expr ast.Expr) bool {
	literal, ok := expr.(*ast.BasicLit)
	if !ok {
		return false
	}
	return literal.Kind == token.INT && literal.Value == "0"
}

// matchLenCall reports whether expr is a call of the form `len(slice)` where slice is a
// bare identifier. Returns the slice identifier when matched.
//
// Takes expr (ast.Expr) which is the candidate expression.
//
// Returns the slice identifier and true when expr matches the shape; nil and false
// otherwise.
func matchLenCall(expr ast.Expr) (*ast.Ident, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	fn, ok := call.Fun.(*ast.Ident)
	if !ok || fn.Name != "len" {
		return nil, false
	}
	arg, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return nil, false
	}
	return arg, true
}

// resolveIntegerConstant resolves expr to an integer constant when the type-checker has
// folded it. Used to populate constUpperBound for the loop unroller's gating decision.
//
// Takes expr (ast.Expr) which is the candidate expression.
// Takes info (*types.Info) which provides constant-folded values; may be nil, in which
// case only bare *ast.BasicLit integers are resolved.
//
// Returns the integer value and true when resolved; 0 and false otherwise.
func resolveIntegerConstant(expr ast.Expr, info *types.Info) (int64, bool) {
	if info != nil {
		if typeAndValue, ok := info.Types[expr]; ok && typeAndValue.Value != nil {
			if value, ok := constant.Int64Val(typeAndValue.Value); ok {
				return value, true
			}
		}
	}
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.INT {
		return 0, false
	}
	value, ok := constant.Int64Val(constant.MakeFromLiteral(literal.Value, token.INT, 0))
	if !ok {
		return 0, false
	}
	return value, true
}
