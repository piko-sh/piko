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
	"go/ast"
	"go/token"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// unrollMaxTripCount caps the constant N the unroller accepts.
	//
	// Tight defaults bias towards safety: the most aggressive unroll expands an 8-iteration
	// 8-statement body into 64 emitted body statements (a small but not micro loop).
	// Anything bigger keeps the standard scalar loop.
	unrollMaxTripCount = 8

	// unrollMaxBodyStmts caps the number of statements per iteration the unroller accepts.
	unrollMaxBodyStmts = 8

	// unrollMaxTotalStmts caps the post-unroll bytecode footprint (trips * stmts). Cheaper
	// to reject conservatively than to risk bytecode bloat.
	unrollMaxTotalStmts = 64

	// loopUnrollerPriority is the recogniser priority for the loop unroller. Sits below the
	// SIMD kernel recogniser (priority simdKernelPriority) so explicit SIMD shapes win when
	// both match a fingerprint bucket.
	loopUnrollerPriority = 10
)

// loopUnrollRecogniser unrolls const-bound canonical for loops.
//
// Matches the `for i := 0; i < N; i++ { body }` shape with a small constant N and a body
// free of break/continue/goto/&loopVar/closures over loopVar, and emits N copies of the
// body with the loop var pre-loaded to k for the k-th copy. No back-edge jump, no cond,
// no post.
//
// Registered AFTER the SIMD recogniser so SIMD-matching loops (which are a strict subset
// of const-bound loop shapes) take precedence. The unroller picks up the remaining
// const-bound shapes the SIMD recogniser refused (non-canonical bodies, non-float64
// operands, etc.).
type loopUnrollRecogniser struct{}

// Name returns the recogniser identifier for diagnostics + disassembler annotations.
//
// Returns "loop.unroll".
func (*loopUnrollRecogniser) Name() string { return "loop.unroll" }

// Priority returns the broad-fallback tiebreaker rank.
//
// The unroller intentionally sits below narrow specialisations like simdKernelRecogniser
// so const-bound canonical loops still hit SIMD when both match the same fingerprint
// bucket.
//
// Returns int which is the loopUnrollerPriority rank.
func (*loopUnrollRecogniser) Priority() int { return loopUnrollerPriority }

// AcceptedSignatures lists every fingerprint signature the unroller is willing to
// consider. Accepts every body shape - including forBodyOther for multi-statement bodies
// - gated on the canonical `i := 0` / `i < N` (const) / `i++` framing.
//
// Returns the full signature set.
func (*loopUnrollRecogniser) AcceptedSignatures() []forStmtFingerprintSignature {
	bodyShapes := []forStmtBodyShape{
		forBodyEmpty,
		forBodySingleAssign,
		forBodySingleAssignBinaryIndex,
		forBodySingleAssignBinaryIndexIndex,
		forBodySingleAssignIndex,
		forBodySingleIfMaxMin,
		forBodyOther,
	}
	initShapes := []forStmtInitShape{forInitConstZeroDecl, forInitConstZeroAssign}
	signatures := make([]forStmtFingerprintSignature, 0, len(initShapes)*len(bodyShapes))
	for _, initShape := range initShapes {
		for _, bodyShape := range bodyShapes {
			signatures = append(signatures, forStmtFingerprintSignature{
				initShape: initShape,
				condShape: forCondLtConst,
				postShape: forPostPlusPlus,
				bodyShape: bodyShape,
			})
		}
	}
	return signatures
}

// Match validates the per-pattern constraints that the fingerprint cannot capture: trip
// count and body size within the unroll budget, body free of control flow /
// address-of-loop-var / closures capturing loop var, body does not write to the loop var.
//
// Takes ctx, statement, fingerprint as for the Recogniser interface.
//
// Returns the loopUnrollMatch token and ok=true on success.
func (*loopUnrollRecogniser) Match(_ recogniseContext, statement *ast.ForStmt, fingerprint forStmtFingerprint) (any, bool) {
	if fingerprint.constUpperBound <= 0 || fingerprint.constUpperBound > unrollMaxTripCount {
		return nil, false
	}
	if fingerprint.bodyStmtCount == 0 || fingerprint.bodyStmtCount > unrollMaxBodyStmts {
		return nil, false
	}
	if fingerprint.constUpperBound*int64(fingerprint.bodyStmtCount) > unrollMaxTotalStmts {
		return nil, false
	}
	loopVar, ok := extractCanonicalLoopVarIdent(statement.Init)
	if !ok {
		return nil, false
	}
	if !isIdentNameEqual(extractCondLeftIdent(statement.Cond), loopVar.Name) {
		return nil, false
	}
	if !isIdentNameEqual(extractPostIdent(statement.Post), loopVar.Name) {
		return nil, false
	}
	if !unrollBodyIsSafe(statement.Body, loopVar) {
		return nil, false
	}
	return loopUnrollMatch{
		loopVarIdent: loopVar,
		tripCount:    int(fingerprint.constUpperBound),
	}, true
}

// Emit produces the unrolled bytecode for the loop.
//
// Emits an `i := 0` declaration in a fresh scope, then for each k in 0..tripCount-1 a
// LOAD_INT_CONST of k into i followed by a compileStmt of the body. No back-edge jump, no
// cond evaluation, no post clause; the loop's semantics reduce to a flat sequence at
// compile time.
//
// Takes ctx (context.Context) which carries cancellation and deadlines.
// Takes emit (emitContext) which carries the compiler handle.
// Takes statement (*ast.ForStmt) which is the matched loop.
// Takes matchToken (any) which is the loopUnrollMatch from Match.
//
// Returns varLocation which is always the zero value for a for-stmt.
// Returns error when body compilation fails.
func (*loopUnrollRecogniser) Emit(ctx context.Context, emit emitContext, statement *ast.ForStmt, matchToken any) (varLocation, error) {
	match, ok := matchToken.(loopUnrollMatch)
	if !ok {
		return emit.compiler.compileForFallback(ctx, statement)
	}
	c := emit.compiler
	c.scopes.pushScope()
	loopVarLocation := c.scopes.declareVar(match.loopVarIdent.Name, registerInt)
	if loopVarLocation.isSpilled || loopVarLocation.isIndirect {
		c.scopes.popScope()
		return c.compileForFallback(ctx, statement)
	}
	defer c.scopes.popScope()
	for k := range match.tripCount {
		c.setDebugPosition(ctx, statement.For)
		c.function.emit(opDrillTier1, uint8(subOpLoadIntConstSmall), loopVarLocation.register, safeconv.MustIntToUint8(k))
		if _, err := c.compileStmt(ctx, statement.Body); err != nil {
			return varLocation{}, err
		}
	}
	return varLocation{}, nil
}

// loopUnrollMatch is the opaque token Match returns to Emit.
//
// Holds the loop variable identifier (used by Emit to declare i in the fresh scope and
// load k into it per iteration) and the validated trip count.
type loopUnrollMatch struct {
	// loopVarIdent names the canonical loop variable to declare during emission.
	loopVarIdent *ast.Ident

	// tripCount is the validated constant iteration count.
	tripCount int
}

// unrollBodyIsSafe reports whether the body is safe to unroll.
//
// Walks the loop body once and refuses any pattern that would miscompile if unrolled:
// break, continue, or goto (loop-scoped control flow has no equivalent in a flattened
// body), taking the address of the loop variable (an unrolled body would silently produce
// different pointers across iterations), any assignment to the loop variable (unrolling
// assumes it is read-only inside the body), and any function literal (Go 1.22+
// per-iteration scope semantics make unrolled closures over the loop variable observably
// different from the loop semantics). A return statement is fine because it leaves the
// surrounding function.
//
// Takes body (*ast.BlockStmt) which is the loop body.
// Takes loopVar (*ast.Ident) which is the canonical loop variable.
//
// Returns bool which is true when the body is safe to unroll.
func unrollBodyIsSafe(body *ast.BlockStmt, loopVar *ast.Ident) bool {
	if body == nil {
		return true
	}
	safe := true
	ast.Inspect(body, func(node ast.Node) bool {
		if !safe {
			return false
		}
		if !nodeIsSafeForUnroll(node, loopVar) {
			safe = false
			return false
		}
		return true
	})
	return safe
}

// nodeIsSafeForUnroll reports whether one node is unroll-compatible.
//
// Control-flow keywords (break, continue, goto, fallthrough), addressing or mutating the
// loop variable, and closures all disqualify a loop body.
//
// Takes node (ast.Node) which is the AST node under inspection.
// Takes loopVar (*ast.Ident) which is the canonical loop variable.
//
// Returns bool which is true when node poses no unroll hazard.
func nodeIsSafeForUnroll(node ast.Node, loopVar *ast.Ident) bool {
	switch n := node.(type) {
	case *ast.BranchStmt:
		return !branchStmtBreaksControlFlow(n)
	case *ast.UnaryExpr:
		return !unaryTakesAddressOfLoopVar(n, loopVar)
	case *ast.AssignStmt:
		return !assignWritesLoopVar(n, loopVar)
	case *ast.IncDecStmt:
		return !incDecMutatesLoopVar(n, loopVar)
	case *ast.FuncLit:
		return false
	}
	return true
}

// branchStmtBreaksControlFlow reports whether the branch breaks unrolling.
//
// Takes n (*ast.BranchStmt) which is the branch statement under inspection.
//
// Returns bool which is true for break/continue/goto/fallthrough.
func branchStmtBreaksControlFlow(n *ast.BranchStmt) bool {
	switch n.Tok {
	case token.BREAK, token.CONTINUE, token.GOTO, token.FALLTHROUGH:
		return true
	default:
	}
	return false
}

// unaryTakesAddressOfLoopVar reports whether n is `&loopVar`.
//
// Address-of would alias the loop variable to the heap and disqualify unrolling.
//
// Takes n (*ast.UnaryExpr) which is the unary expression under inspection.
// Takes loopVar (*ast.Ident) which is the canonical loop variable.
//
// Returns bool which is true when n addresses the loop variable.
func unaryTakesAddressOfLoopVar(n *ast.UnaryExpr, loopVar *ast.Ident) bool {
	if n.Op != token.AND {
		return false
	}
	ident, ok := n.X.(*ast.Ident)
	return ok && ident.Name == loopVar.Name
}

// assignWritesLoopVar reports whether the assignment writes the loop var.
//
// Takes n (*ast.AssignStmt) which is the assignment statement.
// Takes loopVar (*ast.Ident) which is the canonical loop variable.
//
// Returns bool which is true when any LHS binds the loop variable name.
func assignWritesLoopVar(n *ast.AssignStmt, loopVar *ast.Ident) bool {
	for _, lhs := range n.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok && ident.Name == loopVar.Name {
			return true
		}
	}
	return false
}

// incDecMutatesLoopVar reports whether the inc/dec targets the loop var.
//
// Takes n (*ast.IncDecStmt) which is the inc/dec statement.
// Takes loopVar (*ast.Ident) which is the canonical loop variable.
//
// Returns bool which is true when n targets the loop variable.
func incDecMutatesLoopVar(n *ast.IncDecStmt, loopVar *ast.Ident) bool {
	ident, ok := n.X.(*ast.Ident)
	return ok && ident.Name == loopVar.Name
}

func init() { //nolint:gochecknoinits // process-wide recogniser registration.
	if !loopUnrollRecogniserEnabled {
		return
	}
	registerForStmtRecogniser(&loopUnrollRecogniser{})
}
