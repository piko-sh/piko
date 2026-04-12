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
	"go/types"
)

const (
	// maxForStmtFingerprintKeyExprs caps pre-extracted AST expressions.
	//
	// Six covers the widest current pattern (axpy: loop var, accumulator, scalar
	// coefficient, slice operand a, slice operand b, upper bound) without bloating the
	// fingerprint struct. Increase only with a corresponding audit of every recogniser to
	// confirm the new slots stay zero-valued for shapes that do not need them.
	maxForStmtFingerprintKeyExprs = 6
)

// forStmtInitShape classifies the init clause of a for-statement. The classifier maps the
// AST shape to one of these values once; recognisers then read the value, never re-walk
// the AST.
type forStmtInitShape uint8

const (
	// forInitOther is the fallback for any init clause that does not match a recognised
	// pattern; recognisers should refuse on this.
	forInitOther forStmtInitShape = iota

	// forInitNone marks `for ; cond; post {}` (no init clause).
	forInitNone

	// forInitConstZeroDecl marks `i := 0` (short variable declaration of a loop counter to
	// the integer literal 0).
	forInitConstZeroDecl

	// forInitConstZeroAssign marks `i = 0` (plain assignment of an already-declared loop
	// counter to 0).
	forInitConstZeroAssign
)

// forStmtCondShape classifies the loop condition. Recognisers declare which condition
// shapes they accept via fingerprint signatures, so the classifier must produce stable
// enum values per distinct loop family.
type forStmtCondShape uint8

const (
	// forCondOther is the fallback for any condition that does not match a recognised
	// pattern.
	forCondOther forStmtCondShape = iota

	// forCondLtLen marks `i < len(x)` where x is an identifier.
	forCondLtLen

	// forCondLtConst marks `i < N` where N resolves to an integer constant via go/types.
	forCondLtConst

	// forCondLeLen marks `i <= len(x)`.
	forCondLeLen

	// forCondLeConst marks `i <= N`.
	forCondLeConst
)

// forStmtPostShape classifies the post clause of a for-statement.
type forStmtPostShape uint8

const (
	// forPostOther is the fallback for any post clause that does not match a recognised
	// pattern.
	forPostOther forStmtPostShape = iota

	// forPostPlusPlus marks `i++`.
	forPostPlusPlus

	// forPostMinusMinus marks `i--`.
	forPostMinusMinus
)

// forStmtBodyShape classifies the body of a for-statement.
//
// The fingerprint captures shape, not semantics: recognisers do the precise per-pattern
// validation (types, aliasing, operand matching) in Match. Multiple recognisers may
// accept the same body shape; signature collisions resolve at dispatch time via the
// bucketed candidate list.
type forStmtBodyShape uint8

const (
	// forBodyOther is the fallback for any body that does not match a recognised pattern.
	forBodyOther forStmtBodyShape = iota

	// forBodyEmpty marks an empty body (`for ... {}`).
	forBodyEmpty

	// forBodySingleAssign marks one assignment statement (`s = expr` or `s += expr`) whose
	// RHS does not contain an IndexExpr, used by simple-accumulator patterns where the loop
	// adds a constant or non-indexed value.
	forBodySingleAssign

	// forBodySingleAssignBinaryIndex marks one assignment whose RHS is a binary expression
	// with at least one IndexExpr operand; covers `sum += a[i]`, `dst[i] = a[i] * 2`, etc.
	forBodySingleAssignBinaryIndex

	// forBodySingleAssignBinaryIndexIndex marks one assignment whose RHS is a binary
	// expression with TWO IndexExpr operands: the canonical dot-product / element-wise
	// pattern (`sum += a[i] * b[i]`, `dst[i] = a[i] + b[i]`).
	forBodySingleAssignBinaryIndexIndex

	// forBodySingleAssignIndex marks one assignment whose RHS is itself an IndexExpr: the
	// copy pattern (`dst[i] = src[i]`).
	forBodySingleAssignIndex

	// forBodySingleIfMaxMin marks one if-statement with a max/min-update shape (`if a[i] > m
	// { m = a[i] }`).
	forBodySingleIfMaxMin
)

// forStmtFingerprint summarises the shape of a for-statement in a fixed-size hashable
// structure. Produced by a single classification pass over the AST; consumed by the
// recogniser registry to dispatch in O(1) to a small candidate set rather than walking
// every registered recogniser per node.
//
// The fingerprint is intentionally coarse: it captures structural shape but not semantic
// constraints (types, aliasing, constant proofs). Those constraints are validated in each
// recogniser's Match method, which fires only for candidates the fingerprint
// pre-selected.
//
// Extending the fingerprint: add new enum values (forInit*, forCond*, forPost*, forBody*)
// without renumbering existing ones; add new key-expression slots only with a
// corresponding bump of maxForStmtFingerprintKeyExprs. Recognisers tolerate unknown enum
// values by refusing them in AcceptedForStmtSignatures.
type forStmtFingerprint struct {
	// keyExprs are AST expression pointers the classifier pre-extracted so recognisers do
	// not re-walk the loop.
	keyExprs [maxForStmtFingerprintKeyExprs]ast.Expr

	// constUpperBound holds the integer constant N when condShape is forCondLtConst or
	// forCondLeConst; zero otherwise.
	constUpperBound int64

	// initShape is the classified init clause.
	initShape forStmtInitShape

	// condShape is the classified loop condition.
	condShape forStmtCondShape

	// postShape is the classified post clause.
	postShape forStmtPostShape

	// bodyShape is the coarse body classification.
	bodyShape forStmtBodyShape

	// bodyStmtCount is the number of top-level statements in the loop body (capped at 255
	// for fingerprint compactness; bodies larger than 255 statements are unlikely to match
	// any recogniser and are floored at the cap).
	bodyStmtCount uint8
}

// forStmtFingerprintSignature is the recogniser-registry hash key.
//
// Two fingerprints with the same signature land in the same dispatch bucket; signature
// collisions then resolve via the bucket's small candidate list and each recogniser's
// Match. Intentionally narrower than the full fingerprint: numeric fields
// (constUpperBound, bodyStmtCount) and key expressions do not participate, so recognisers
// gating on those still share a bucket with sibling recognisers gating on the same shape
// but different constants.
type forStmtFingerprintSignature struct {
	// initShape is the classified init clause.
	initShape forStmtInitShape

	// condShape is the classified loop condition.
	condShape forStmtCondShape

	// postShape is the classified post clause.
	postShape forStmtPostShape

	// bodyShape is the coarse body classification.
	bodyShape forStmtBodyShape
}

// signature returns the dispatch key for this fingerprint.
//
// Returns the four-shape tuple that the recogniser index uses to look up candidate
// recognisers.
func (fingerprint forStmtFingerprint) signature() forStmtFingerprintSignature {
	return forStmtFingerprintSignature{
		initShape: fingerprint.initShape,
		condShape: fingerprint.condShape,
		postShape: fingerprint.postShape,
		bodyShape: fingerprint.bodyShape,
	}
}

// forStmtRecogniser is the interface every for-statement AST-pattern consumer implements.
// Recognisers register at init time; the subsystem dispatches via fingerprint-keyed
// lookup so adding new recognisers does not slow the dispatch path for existing ones.
//
// Match runs ONLY when the fingerprint signature matches one of the signatures returned
// by AcceptedSignatures. It performs the targeted per-pattern validation (type checks,
// aliasing checks, constant proofs) that fingerprints cannot capture and returns an
// opaque token the Emit step consumes.
//
// Emit replaces the standard compile of the matched for-statement with the recogniser's
// specialised emission. It must produce bytecode equivalent in observable semantics to
// what compileFor would have produced, or the program miscompiles. Returning an error
// aborts compilation.
type forStmtRecogniser interface {
	// Name returns a short stable identifier for diagnostics and disassembler annotations.
	// Use a dotted namespace prefix (`simd.dot_product_f64`, `loop.unroll`) so multiple
	// recognisers from the same consumer share an obvious tag.
	Name() string

	// Priority returns the bucket tiebreaker rank.
	//
	// Higher priority wins when multiple recognisers accept the same signature, used to
	// resolve overlap between, e.g., the SIMD kernel recogniser (narrow shape) and the loop
	// unroller (broader const-bound shape). Recommended ranges: 0 (default) for fallback
	// recognisers, 10 for general-purpose, 100 for narrow-specialised. Equal priorities are
	// ordered by registration time.
	//
	// Returns int which is the tiebreaker rank.
	Priority() int

	// AcceptedSignatures returns every fingerprint signature this recogniser is willing to
	// consider. The registry indexes recognisers by these signatures at registration time so
	// dispatch is O(1) per node regardless of how many recognisers are registered.
	AcceptedSignatures() []forStmtFingerprintSignature

	// Match performs targeted per-pattern validation.
	//
	// Match must be conservative: false positives miscompile, false negatives just miss the
	// optimisation. Returns the opaque token the Emit step consumes when the pattern
	// applies.
	//
	// Takes ctx (recogniseContext) which provides read-only state.
	// Takes statement (*ast.ForStmt) which is the matched node.
	// Takes fingerprint (forStmtFingerprint) which carries the shape.
	//
	// Returns token (any) which the Emit step consumes.
	// Returns ok (bool) which is true when the pattern applies.
	Match(ctx recogniseContext, statement *ast.ForStmt, fingerprint forStmtFingerprint) (token any, ok bool)

	// Emit produces bytecode for the matched statement.
	//
	// Replaces the standard compile path. Receives the token Match returned and the caller's
	// compile-time context for threading into fallback paths (compileForFallback,
	// compileStmt).
	//
	// Takes ctx (context.Context) which carries cancellation.
	// Takes emit (emitContext) which exposes the compiler state.
	// Takes statement (*ast.ForStmt) which is the node to emit.
	// Takes matchToken (any) which Match returned.
	//
	// Returns varLocation which is the destination of the loop's result (typically the zero
	// varLocation).
	// Returns error when compilation fails.
	Emit(ctx context.Context, emit emitContext, statement *ast.ForStmt, matchToken any) (varLocation, error)
}

// recogniseContext bundles the read-only state Match needs. Restricted to read-only ops
// so a Match cannot mutate compile state by accident; mutation happens only inside Emit
// via the emitContext.
type recogniseContext struct {
	// info is the go/types info table for the package being compiled. Recognisers use it to
	// resolve identifier types and extract constant values via info.Types[expr].Value.
	info *types.Info

	// function is the current CompiledFunction being built. Read-only access for inspection
	// only; Match must not write.
	function *CompiledFunction
}

// emitContext is the mutable compile state Emit uses.
//
// Wraps the compiler so Emit shares the register allocator, bytecode emitter, scope
// stack, and helper methods that the standard compileStmt/compileExpression paths use.
// The caller's compile-time context is threaded as a separate Emit parameter rather than
// carried on the struct so cancellation flows the same way the rest of the compile path
// uses it.
type emitContext struct {
	// compiler is the active compiler instance; Emit mutates state (registers, scopes,
	// emitted bytecode) through this handle.
	compiler *compiler
}

var (
	// forStmtRecogniserIndex maps fingerprint signatures to recogniser buckets.
	//
	// Built at init/registration time and treated as immutable so dispatch is lock-free.
	// nil-valued buckets are equivalent to absent buckets.
	//
	//nolint:gochecknoglobals // process-wide registry, initialised once at package init
	forStmtRecogniserIndex map[forStmtFingerprintSignature][]forStmtRecogniser
)

// astPatternRecogniseContext builds the read-only context the recogniser subsystem passes
// to Match. Called once per for-statement compile from compileFor; cheap struct
// construction with no allocation.
//
// Returns the recogniseContext snapshot.
func (c *compiler) astPatternRecogniseContext() recogniseContext {
	return recogniseContext{
		info:     c.info,
		function: c.function,
	}
}

// astPatternEmitContext builds the mutable context for Emit.
//
// Wraps the compiler so Emit shares the register allocator, scope stack, and bytecode
// emitter the standard compile path uses. The compile-time context is threaded to Emit as
// a separate parameter rather than stored on the struct, matching the calling convention
// used elsewhere in the compile path.
//
// Returns emitContext which carries the compiler handle.
func (c *compiler) astPatternEmitContext() emitContext {
	return emitContext{compiler: c}
}

// registerForStmtRecogniser inserts a recogniser into the index.
//
// Indexes r under every signature it reports via AcceptedSignatures. Called once per
// recogniser from a package-level init function. Registration order determines candidate
// order within a bucket; the first recogniser whose Match returns true wins.
//
// Takes r (forStmtRecogniser) which is the recogniser to register.
func registerForStmtRecogniser(r forStmtRecogniser) {
	if forStmtRecogniserIndex == nil {
		forStmtRecogniserIndex = make(map[forStmtFingerprintSignature][]forStmtRecogniser)
	}
	for _, sig := range r.AcceptedSignatures() {
		bucket := forStmtRecogniserIndex[sig]
		insertAt := len(bucket)
		for i, existing := range bucket {
			if existing.Priority() < r.Priority() {
				insertAt = i
				break
			}
		}
		bucket = append(bucket, nil)
		copy(bucket[insertAt+1:], bucket[insertAt:])
		bucket[insertAt] = r
		forStmtRecogniserIndex[sig] = bucket
	}
}

// tryRecogniseForStmt is the dispatch entry called from compileFor. Runs a single
// classification pass over the statement, looks up the bucket of candidate recognisers by
// fingerprint signature, and returns the first whose Match accepts.
//
// The per-call cost is one classification pass over the for-statement (O(body-size)), one
// map lookup (O(1) amortised), and zero-to-two targeted Match calls on the small
// candidate set. The total is independent of how many recognisers are registered; the
// 50th recogniser does not slow the dispatch path for the other 49.
//
// Takes ctx (recogniseContext) which provides go/types info and the current function.
// Read-only.
// Takes statement (*ast.ForStmt) which is the for-statement being compiled.
//
// Returns the recogniser that accepted (or nil), the token its Match produced (for the
// Emit step), and ok=true on success.
// Returns ok=false when no recogniser matches; the caller then runs the standard scalar
// compile path.
func tryRecogniseForStmt(ctx recogniseContext, statement *ast.ForStmt) (forStmtRecogniser, any, bool) {
	if len(forStmtRecogniserIndex) == 0 {
		return nil, nil, false
	}
	fingerprint := classifyForStmt(statement, ctx.info)
	bucket := forStmtRecogniserIndex[fingerprint.signature()]
	if len(bucket) == 0 {
		return nil, nil, false
	}
	for _, recogniser := range bucket {
		if token, ok := recogniser.Match(ctx, statement, fingerprint); ok {
			return recogniser, token, true
		}
	}
	return nil, nil, false
}
