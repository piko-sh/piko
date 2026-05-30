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
	"go/types"
)

// simdKernelKind enumerates the canonical numerical loop kernels.
//
// Each value corresponds to a tier-1 sub-opcode in vm_handler_simd.go and covers one of
// the supported shapes (dot product, sum reduction, element-wise add, slice scale) that
// exercise the distinct AST body shapes the fingerprint classifier produces. Each kernel
// is extended by adding to the enum, the matchKernelByBody dispatch, and the per-kernel
// emit functions, with no subsystem changes required.
type simdKernelKind uint8

const (
	// simdKernelNone indicates no SIMD kernel was matched.
	simdKernelNone simdKernelKind = iota //nolint:unused // documented enum slot retained for ABI stability

	// simdKernelDotProductFloat64 names the float64 dot-product kernel.
	simdKernelDotProductFloat64

	// simdKernelSumSliceFloat64 names the float64 sum-reduction kernel.
	simdKernelSumSliceFloat64

	// simdKernelAddSliceFloat64 names the float64 element-wise add kernel.
	simdKernelAddSliceFloat64

	// simdKernelScaleSliceFloat64 names the float64 in-place scale kernel.
	simdKernelScaleSliceFloat64
)

// simdKernelMatch is the opaque token a successful Match returns.
//
// Pre-extracted operand expressions skip the need for Emit to re-walk the AST.
type simdKernelMatch struct {
	// sliceA is the first slice operand expression (always an *ast.Ident; types are checked
	// in Match).
	sliceA ast.Expr

	// sliceB is the second slice operand, or nil for kernels with only one slice operand
	// (sum, scale, clear, fill).
	sliceB ast.Expr

	// destinationScalar is the scalar accumulator destination for reduction kernels (dot
	// product, sum). nil for non-reductions.
	destinationScalar ast.Expr

	// destinationSlice is the destination slice for element-wise kernels (add, sub, mul).
	// nil for reductions and in-place kernels (scale, clear, fill).
	destinationSlice ast.Expr

	// scalarOperand is the scalar coefficient for axpy/scale/fill kernels. nil otherwise.
	scalarOperand ast.Expr

	// boundSlice is the *ast.Ident the count comes from (the slice inside `len(slice)`) when
	// boundShape is simdBoundLenSlice; otherwise nil.
	boundSlice ast.Expr

	// boundConstValue is the constant iteration count when boundShape is simdBoundConst;
	// otherwise 0.
	boundConstValue int64

	// kind names which SIMD opcode Emit should produce.
	kind simdKernelKind

	// boundShape names how the loop's iteration count is computed at runtime.
	// simdBoundLenSlice means "len(boundSlice)"; simdBoundConst means "boundConstValue
	// elements (subject to runtime length check)".
	boundShape simdKernelBoundShape
}

// simdKernelBoundShape names the runtime origin of the iteration count.
//
// simdBoundLenSlice means the count is read from a matching operand slice's length (the
// canonical `for i := 0; i < len(slice); i++` shape). simdBoundConst means the count is a
// compile-time constant from the loop condition (the `for i := 0; i < N; i++` shape).
type simdKernelBoundShape uint8

const (
	// simdBoundUnspecified indicates the bound shape has not been set.
	simdBoundUnspecified simdKernelBoundShape = iota //nolint:unused // documented enum slot retained for ABI stability

	// simdBoundLenSlice indicates the count comes from len(slice).
	simdBoundLenSlice

	// simdBoundConst indicates the count is a compile-time constant.
	simdBoundConst
)

const (
	// simdKernelPriority is the recogniser priority for the SIMD kernel matcher; sits above
	// the loop unroller (priority loopUnrollerPriority) so explicit SIMD shapes win when
	// both match a fingerprint bucket.
	simdKernelPriority = 100

	// simdAcceptedSignatureCapacity is the up-front capacity for the AcceptedSignatures
	// bucket. Sized to hold the cartesian product of init shape (2) x cond shape (2) x body
	// shape (4).
	simdAcceptedSignatureCapacity = 16

	// maxUint8Value is the largest value that fits in a uint8 operand byte. Used to gate the
	// small-const fast-path that packs the bound into a single instruction byte.
	maxUint8Value = 0xFF
)

// simdKernelRecogniser is a consumer of the AST pattern-recognition subsystem. It matches
// canonical numerical loop shapes against the SIMD kernel patterns and emits the
// corresponding tier-1 SIMD sub-opcode in place of the scalar loop.
//
// Conservative by design: every constraint that fingerprint classification cannot capture
// (operand types, aliasing, identity of the loop variable across body uses) is validated
// explicitly in Match. A false positive miscompiles; a false negative just misses the
// optimisation and the scalar loop emits unchanged.
type simdKernelRecogniser struct{}

// Name returns the recogniser identifier for diagnostics + disassembler annotations.
//
// Returns "simd.kernels".
func (*simdKernelRecogniser) Name() string { return "simd.kernels" }

// Priority returns simdKernelPriority.
//
// SIMD kernels target a precise shape (typed float64 slice operands plus canonical body),
// so they outrank broader optimisations like the loop unroller when both match a
// fingerprint bucket.
//
// Returns int which is the recogniser priority.
func (*simdKernelRecogniser) Priority() int { return simdKernelPriority }

// AcceptedSignatures lists every fingerprint signature accepted.
//
// All entries share the same canonical loop framing (`i := 0` or `i = 0` / `i++`) and
// differ in cond shape (len-of-slice vs constant) and body shape. The registry indexes by
// signature so dispatch finds this recogniser via one map lookup when a matching loop is
// compiled.
//
// Returns []forStmtFingerprintSignature which is the accepted set.
func (*simdKernelRecogniser) AcceptedSignatures() []forStmtFingerprintSignature {
	signatures := make([]forStmtFingerprintSignature, 0, simdAcceptedSignatureCapacity)
	for _, initShape := range []forStmtInitShape{forInitConstZeroDecl, forInitConstZeroAssign} {
		for _, condShape := range []forStmtCondShape{forCondLtLen, forCondLtConst} {
			for _, bodyShape := range []forStmtBodyShape{
				forBodySingleAssignBinaryIndexIndex,
				forBodySingleAssignIndex,
				forBodySingleAssign,
			} {
				signatures = append(signatures, forStmtFingerprintSignature{
					initShape: initShape,
					condShape: condShape,
					postShape: forPostPlusPlus,
					bodyShape: bodyShape,
				})
			}
		}
	}
	return signatures
}

// Match validates a for-statement whose fingerprint matched one of AcceptedSignatures.
// Performs the precise per-kernel checks (loop-variable consistency, slice-element types,
// aliasing) and dispatches to the appropriate per-kernel matcher.
//
// Takes ctx (recogniseContext) which carries go/types info.
// Takes statement (*ast.ForStmt) which is the candidate loop.
// Takes fingerprint (forStmtFingerprint) which the classifier pre-extracted.
//
// Returns the simdKernelMatch token and ok=true on success; nil and ok=false on any
// constraint failure.
func (*simdKernelRecogniser) Match(ctx recogniseContext, statement *ast.ForStmt, fingerprint forStmtFingerprint) (any, bool) {
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
	var (
		upperBoundSlice *ast.Ident
		bound           simdKernelBoundInfo
	)
	switch fingerprint.condShape {
	case forCondLtLen:
		ident, ok := extractLenSliceIdent(fingerprint.keyExprs[forStmtKeyExprUpperBound])
		if !ok {
			return nil, false
		}
		upperBoundSlice = ident
		bound = simdKernelBoundInfo{shape: simdBoundLenSlice, slice: ident}
	case forCondLtConst:
		if fingerprint.constUpperBound < 0 {
			return nil, false
		}
		bound = simdKernelBoundInfo{shape: simdBoundConst, constValue: fingerprint.constUpperBound}
	default:
		return nil, false
	}
	switch fingerprint.bodyShape {
	case forBodySingleAssignBinaryIndexIndex:
		return matchBinaryIndexIndexKernel(ctx, statement, fingerprint, loopVar, upperBoundSlice, bound)
	case forBodySingleAssignIndex:
		return matchAssignIndexKernel(ctx, statement, fingerprint, loopVar, upperBoundSlice, bound)
	case forBodySingleAssign:
		return matchSingleAssignKernel(ctx, statement, fingerprint, loopVar, upperBoundSlice, bound)
	default:
	}
	return nil, false
}

// simdKernelBoundInfo bundles the count-bound information.
//
// The recogniser pre-extracts the bound in Match and forwards it to the per-kernel
// matchers and through them to the simdKernelMatch token.
type simdKernelBoundInfo struct {
	// slice is the slice identifier referenced by the cond's len() call when shape is
	// simdBoundLenSlice; otherwise nil.
	slice *ast.Ident

	// constValue is the compile-time iteration count when shape is simdBoundConst; otherwise
	// zero.
	constValue int64

	// shape names whether the count comes from len(slice) or a const.
	shape simdKernelBoundShape
}

// Emit produces the bytecode for a matched SIMD kernel.
//
// Replaces the standard scalar loop emission entirely when the per-kernel emit succeeds.
// When the emit declines (e.g. an operand is not on the typed slicesFloat bank), Emit
// falls through to the standard scalar compileFor path so the loop still compiles
// correctly.
//
// Takes ctx (context.Context) which threads cancellation into emits.
// Takes emit (emitContext) which provides the compiler handle.
// Takes statement (*ast.ForStmt) which is the matched loop.
// Takes matchToken (any) which is the simdKernelMatch from Match.
//
// Returns varLocation which is always the zero value since for-statements do not produce
// values.
// Returns error when compilation fails.
func (*simdKernelRecogniser) Emit(ctx context.Context, emit emitContext, statement *ast.ForStmt, matchToken any) (varLocation, error) {
	match, ok := matchToken.(simdKernelMatch)
	if !ok {
		return emit.compiler.compileForFallback(ctx, statement)
	}
	location, emitted, err := emitSimdKernelByKind(ctx, emit.compiler, match)
	if err != nil {
		return varLocation{}, err
	}
	if emitted {
		return location, nil
	}
	return emit.compiler.compileForFallback(ctx, statement)
}

// emitSimdKernelByKind dispatches to the per-kernel emit routine.
//
// emitted=false means the caller (Emit) should fall back to the standard scalar emission
// path.
//
// Takes ctx (context.Context) which is threaded into per-kernel emits for any
// sub-expression compilation they perform (e.g. the scalar operand on the scale kernel).
// Takes c (*compiler) which carries the active emit state.
// Takes match (simdKernelMatch) which names the matched kernel.
//
// Returns varLocation which is the kernel's location.
// Returns bool which is true when the SIMD opcode was written.
// Returns error when compilation fails.
func emitSimdKernelByKind(ctx context.Context, c *compiler, match simdKernelMatch) (varLocation, bool, error) {
	switch match.kind {
	case simdKernelDotProductFloat64:
		return emitSimdDotProductFloat64(c, match)
	case simdKernelSumSliceFloat64:
		return emitSimdSumSliceFloat64(c, match)
	case simdKernelAddSliceFloat64:
		return emitSimdAddSliceFloat64(c, match)
	case simdKernelScaleSliceFloat64:
		return emitSimdScaleSliceFloat64(ctx, c, match)
	default:
	}
	return varLocation{}, false, nil
}

// matchBinaryIndexIndexKernel dispatches loops with binary-indexed RHS.
//
// The body is one assignment with a binary expression of two IndexExprs on the RHS. Two
// SIMD kernels share this body shape and disambiguate via the LHS: a scalar Ident selects
// the dot product kernel, an IndexExpr selects the element-wise op.
//
// Takes ctx (recogniseContext) which carries go/types info.
// Takes statement (*ast.ForStmt) which is the candidate loop.
// Takes fingerprint (forStmtFingerprint) which the classifier pre-extracted.
// Takes loopVar (*ast.Ident) which is the canonical loop variable.
// Takes upperBoundSlice (*ast.Ident) which is the slice the cond's len() reads.
// Takes bound (simdKernelBoundInfo) which carries count-bound info.
//
// Returns any which is the matched simdKernelMatch token.
// Returns bool which is true when the match succeeds.
func matchBinaryIndexIndexKernel(ctx recogniseContext, statement *ast.ForStmt, fingerprint forStmtFingerprint, loopVar, upperBoundSlice *ast.Ident, bound simdKernelBoundInfo) (any, bool) {
	assign, ok := statement.Body.List[0].(*ast.AssignStmt)
	if !ok {
		return nil, false
	}
	binary, ok := fingerprint.keyExprs[forStmtKeyExprAssignRHS].(*ast.BinaryExpr)
	if !ok {
		return nil, false
	}
	sliceA, sliceB, ok := matchBinaryFloat64IndexOperands(ctx, binary, loopVar)
	if !ok {
		return nil, false
	}
	if bound.shape == simdBoundLenSlice && !isOneOfSlices(upperBoundSlice, sliceA, sliceB) {
		return nil, false
	}
	sources := simdKernelBinarySources{sliceA: sliceA, sliceB: sliceB}
	switch leftHand := assign.Lhs[0].(type) {
	case *ast.Ident:
		return matchDotProductFloat64Kernel(ctx, assign, binary, leftHand, sources, bound)
	case *ast.IndexExpr:
		return matchElementwiseAddFloat64Kernel(ctx, assign, binary, leftHand, loopVar, sources, bound)
	}
	return nil, false
}

// simdKernelBinarySources bundles the two source slices.
//
// Binary SIMD kernels operate on two source slices; bundling them keeps per-kernel
// matchers under the per-function argument cap.
type simdKernelBinarySources struct {
	// sliceA is the first source slice identifier.
	sliceA *ast.Ident

	// sliceB is the second source slice identifier.
	sliceB *ast.Ident
}

// matchBinaryFloat64IndexOperands resolves both binary operands.
//
// Treats each operand as a float64 slice index against the loop variable and returns the
// two source slices when the shape matches.
//
// Takes ctx (recogniseContext) which carries go/types info.
// Takes binary (*ast.BinaryExpr) which is the candidate RHS.
// Takes loopVar (*ast.Ident) which is the canonical loop variable.
//
// Returns sliceA (*ast.Ident) which is the first source slice.
// Returns sliceB (*ast.Ident) which is the second source slice.
// Returns ok (bool) which is true when both operands match.
func matchBinaryFloat64IndexOperands(ctx recogniseContext, binary *ast.BinaryExpr, loopVar *ast.Ident) (sliceA, sliceB *ast.Ident, ok bool) {
	sliceA, ok = indexedSliceWithLoopVar(binary.X, loopVar)
	if !ok {
		return nil, nil, false
	}
	sliceB, ok = indexedSliceWithLoopVar(binary.Y, loopVar)
	if !ok {
		return nil, nil, false
	}
	if !isFloat64Slice(ctx.info, sliceA) || !isFloat64Slice(ctx.info, sliceB) {
		return nil, nil, false
	}
	return sliceA, sliceB, true
}

// matchDotProductFloat64Kernel recognises the dot-product shape.
//
// The accepted shape is `scalar += a[i] * b[i]`.
//
// Takes ctx (recogniseContext) which carries go/types info.
// Takes assign (*ast.AssignStmt) which is the body's assignment.
// Takes binary (*ast.BinaryExpr) which is the RHS multiplication.
// Takes leftHand (*ast.Ident) which is the scalar destination.
// Takes sources (simdKernelBinarySources) which carries the two source slices.
// Takes bound (simdKernelBoundInfo) which carries count-bound info.
//
// Returns any which is the matched simdKernelMatch token.
// Returns bool which is true when the match succeeds.
func matchDotProductFloat64Kernel(ctx recogniseContext, assign *ast.AssignStmt, binary *ast.BinaryExpr, leftHand *ast.Ident, sources simdKernelBinarySources, bound simdKernelBoundInfo) (any, bool) {
	if assign.Tok != token.ADD_ASSIGN || binary.Op != token.MUL {
		return nil, false
	}
	if !isFloat64Scalar(ctx.info, leftHand) {
		return nil, false
	}
	if isSameIdent(sources.sliceA, leftHand) || isSameIdent(sources.sliceB, leftHand) {
		return nil, false
	}
	return makeSimdKernelMatch(simdKernelDotProductFloat64, simdKernelOperands{
		sliceA:            sources.sliceA,
		sliceB:            sources.sliceB,
		destinationScalar: leftHand,
	}, bound), true
}

// matchElementwiseAddFloat64Kernel recognises the elementwise-add shape.
//
// The accepted shape is `dest[i] = a[i] + b[i]`.
//
// Takes ctx (recogniseContext) which carries go/types info.
// Takes assign (*ast.AssignStmt) which is the body's assignment.
// Takes binary (*ast.BinaryExpr) which is the RHS addition.
// Takes leftHand (*ast.IndexExpr) which is the destination index.
// Takes loopVar (*ast.Ident) which is the canonical loop variable.
// Takes sources (simdKernelBinarySources) which carries the two source slices.
// Takes bound (simdKernelBoundInfo) which carries count-bound info.
//
// Returns any which is the matched simdKernelMatch token.
// Returns bool which is true when the match succeeds.
func matchElementwiseAddFloat64Kernel(
	ctx recogniseContext,
	assign *ast.AssignStmt,
	binary *ast.BinaryExpr,
	leftHand *ast.IndexExpr,
	loopVar *ast.Ident,
	sources simdKernelBinarySources,
	bound simdKernelBoundInfo,
) (any, bool) {
	if assign.Tok != token.ASSIGN || binary.Op != token.ADD {
		return nil, false
	}
	destSlice, ok := indexedSliceWithLoopVar(leftHand, loopVar)
	if !ok {
		return nil, false
	}
	if !isFloat64Slice(ctx.info, destSlice) {
		return nil, false
	}
	if isSameIdent(destSlice, sources.sliceA) || isSameIdent(destSlice, sources.sliceB) {
		return nil, false
	}
	return makeSimdKernelMatch(simdKernelAddSliceFloat64, simdKernelOperands{
		sliceA:           sources.sliceA,
		sliceB:           sources.sliceB,
		destinationSlice: destSlice,
	}, bound), true
}

// simdKernelOperands bundles the resolved operand expressions.
//
// makeSimdKernelMatch stitches the operands together with the bound info into the
// simdKernelMatch token.
type simdKernelOperands struct {
	// sliceA is the first slice operand expression, or nil.
	sliceA ast.Expr

	// sliceB is the second slice operand expression, or nil.
	sliceB ast.Expr

	// destinationScalar is the reduction scalar destination, or nil.
	destinationScalar ast.Expr

	// destinationSlice is the element-wise slice destination, or nil.
	destinationSlice ast.Expr

	// scalarOperand is the scalar coefficient operand, or nil.
	scalarOperand ast.Expr
}

// makeSimdKernelMatch composes a simdKernelMatch token from a resolved kernel kind,
// operand expressions, and the loop's bound info. Centralising token construction keeps
// the per-kernel matchers free of bookkeeping for the bound shape.
//
// Takes kind, operands, bound.
//
// Returns the populated simdKernelMatch.
func makeSimdKernelMatch(kind simdKernelKind, operands simdKernelOperands, bound simdKernelBoundInfo) simdKernelMatch {
	var boundSlice ast.Expr
	if bound.slice != nil {
		boundSlice = bound.slice
	}
	return simdKernelMatch{
		kind:              kind,
		sliceA:            operands.sliceA,
		sliceB:            operands.sliceB,
		destinationScalar: operands.destinationScalar,
		destinationSlice:  operands.destinationSlice,
		scalarOperand:     operands.scalarOperand,
		boundShape:        bound.shape,
		boundSlice:        boundSlice,
		boundConstValue:   bound.constValue,
	}
}

// matchAssignIndexKernel handles bodies whose RHS is a single IndexExpr (the
// sum-reduction shape `sum += a[i]`).
//
// Takes ctx, statement, fingerprint, loopVar, upperBoundSlice as the dispatching wrapper
// above.
//
// Returns the matched token or false.
func matchAssignIndexKernel(ctx recogniseContext, statement *ast.ForStmt, fingerprint forStmtFingerprint, loopVar, upperBoundSlice *ast.Ident, bound simdKernelBoundInfo) (any, bool) {
	assign, ok := statement.Body.List[0].(*ast.AssignStmt)
	if !ok {
		return nil, false
	}
	if assign.Tok != token.ADD_ASSIGN {
		return nil, false
	}
	destinationScalar, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, false
	}
	if !isFloat64Scalar(ctx.info, destinationScalar) {
		return nil, false
	}
	indexExpr, ok := fingerprint.keyExprs[forStmtKeyExprAssignRHS].(*ast.IndexExpr)
	if !ok {
		return nil, false
	}
	sliceA, ok := indexedSliceWithLoopVar(indexExpr, loopVar)
	if !ok {
		return nil, false
	}
	if !isFloat64Slice(ctx.info, sliceA) {
		return nil, false
	}
	if bound.shape == simdBoundLenSlice && !isOneOfSlices(upperBoundSlice, sliceA) {
		return nil, false
	}
	if isSameIdent(sliceA, destinationScalar) {
		return nil, false
	}
	return makeSimdKernelMatch(simdKernelSumSliceFloat64, simdKernelOperands{
		sliceA:            sliceA,
		destinationScalar: destinationScalar,
	}, bound), true
}

// matchSingleAssignKernel handles bodies whose RHS does not directly contain an IndexExpr
// top-level. It matches the scale shape `s[i] *= k`.
//
// Takes ctx, statement, fingerprint, loopVar, upperBoundSlice as the dispatching wrapper
// above.
//
// Returns the matched token or false.
func matchSingleAssignKernel(ctx recogniseContext, statement *ast.ForStmt, fingerprint forStmtFingerprint, loopVar, upperBoundSlice *ast.Ident, bound simdKernelBoundInfo) (any, bool) {
	assign, ok := statement.Body.List[0].(*ast.AssignStmt)
	if !ok {
		return nil, false
	}
	if assign.Tok != token.MUL_ASSIGN {
		return nil, false
	}
	destinationIndex, ok := assign.Lhs[0].(*ast.IndexExpr)
	if !ok {
		return nil, false
	}
	destinationSlice, ok := indexedSliceWithLoopVar(destinationIndex, loopVar)
	if !ok {
		return nil, false
	}
	if !isFloat64Slice(ctx.info, destinationSlice) {
		return nil, false
	}
	if bound.shape == simdBoundLenSlice && !isOneOfSlices(upperBoundSlice, destinationSlice) {
		return nil, false
	}
	scalarIdent, ok := assign.Rhs[0].(*ast.Ident)
	if !ok {
		return nil, false
	}
	if !isFloat64Scalar(ctx.info, scalarIdent) {
		return nil, false
	}
	if isSameIdent(destinationSlice, scalarIdent) {
		return nil, false
	}
	_ = fingerprint
	return makeSimdKernelMatch(simdKernelScaleSliceFloat64, simdKernelOperands{
		sliceA:           destinationSlice,
		destinationSlice: destinationSlice,
		scalarOperand:    scalarIdent,
	}, bound), true
}

// emitSimdDotProductFloat64 emits the dot-product SIMD opcode.
//
// Computes `destinationScalar += dot(sliceA, sliceB)` via one tier-1 SIMD opcode plus one
// extension word. Both slice operands must resolve to the typed slicesFloat bank; the
// recogniser falls through to the standard scalar loop emission when either operand is on
// the reflect general-bank path (slice literals, function returns of unknown type, etc.).
// The destination is the scalar float register already holding `sum`. The count operand
// is materialised as a fresh int register populated either from len(slice) (LtLen shape)
// or a constant load (LtConst shape).
//
// Takes c (*compiler) which carries the active emit state.
// Takes match (simdKernelMatch) which holds the matched operand expressions and bound
// info.
//
// Returns varLocation which is always zero since for-statements do not produce values.
// Returns bool which is true when the SIMD opcode was emitted.
// Returns error when compilation fails.
func emitSimdDotProductFloat64(c *compiler, match simdKernelMatch) (varLocation, bool, error) {
	destLocation, ok := resolveScalarDestination(c, match.destinationScalar)
	if !ok {
		return varLocation{}, false, nil
	}
	sliceALocation, ok, err := resolveFloat64SliceOperand(c, match.sliceA)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	sliceBLocation, ok, err := resolveFloat64SliceOperand(c, match.sliceB)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	countRegister, ok, err := emitSimdCountRegister(c, match, sliceALocation.register)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	c.function.emit(opDrillTier1, uint8(subOpSimdDotProductFloat64), destLocation.register, sliceALocation.register)
	c.function.emit(opExt, sliceBLocation.register, countRegister, 0)
	return varLocation{}, true, nil
}

// emitSimdSumSliceFloat64 emits one tier-1 SIMD opcode (with one extension word for
// count) that computes `destinationScalar += sum(sliceA[:count])`. Refuses when sliceA is
// not on the typed slicesFloat bank.
//
// Takes c (*compiler).
// Takes match (simdKernelMatch).
//
// Returns the zero varLocation, ok=true when the SIMD opcode was emitted, and any
// compilation error.
func emitSimdSumSliceFloat64(c *compiler, match simdKernelMatch) (varLocation, bool, error) {
	destLocation, ok := resolveScalarDestination(c, match.destinationScalar)
	if !ok {
		return varLocation{}, false, nil
	}
	sliceALocation, ok, err := resolveFloat64SliceOperand(c, match.sliceA)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	countRegister, ok, err := emitSimdCountRegister(c, match, sliceALocation.register)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	c.function.emit(opDrillTier1, uint8(subOpSimdSumSliceFloat64), destLocation.register, sliceALocation.register)
	c.function.emit(opExt, countRegister, 0, 0)
	return varLocation{}, true, nil
}

// emitSimdAddSliceFloat64 emits one tier-1 SIMD opcode (with one extension word carrying
// sliceB and count) that computes `destinationSlice[i] = sliceA[i] + sliceB[i]` for the
// first `count` indices. All three slices must be on the typed slicesFloat bank.
//
// Takes c (*compiler).
// Takes match (simdKernelMatch).
//
// Returns the zero varLocation, ok=true when the SIMD opcode was emitted, and any
// compilation error.
func emitSimdAddSliceFloat64(c *compiler, match simdKernelMatch) (varLocation, bool, error) {
	destLocation, ok, err := resolveFloat64SliceOperand(c, match.destinationSlice)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	sliceALocation, ok, err := resolveFloat64SliceOperand(c, match.sliceA)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	sliceBLocation, ok, err := resolveFloat64SliceOperand(c, match.sliceB)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	countRegister, ok, err := emitSimdCountRegister(c, match, sliceALocation.register)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	c.function.emit(opDrillTier1, uint8(subOpSimdAddSliceFloat64), destLocation.register, sliceALocation.register)
	c.function.emit(opExt, sliceBLocation.register, countRegister, 0)
	return varLocation{}, true, nil
}

// emitSimdScaleSliceFloat64 emits one tier-1 SIMD opcode (with one extension word for
// count) that multiplies the first `count` elements of sliceA in place by scalarOperand.
// Refuses when sliceA is not on the typed slicesFloat bank.
//
// Takes c (*compiler).
// Takes match (simdKernelMatch).
//
// Returns the zero varLocation, ok=true when the SIMD opcode was emitted, and any
// compilation error.
func emitSimdScaleSliceFloat64(ctx context.Context, c *compiler, match simdKernelMatch) (varLocation, bool, error) {
	sliceLocation, ok, err := resolveFloat64SliceOperand(c, match.sliceA)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	scalarLocation, err := c.compileExpression(ctx, match.scalarOperand)
	if err != nil {
		return varLocation{}, false, err
	}
	if scalarLocation.kind != registerFloat {
		return varLocation{}, false, nil
	}
	countRegister, ok, err := emitSimdCountRegister(c, match, sliceLocation.register)
	if !ok || err != nil {
		return varLocation{}, false, err
	}
	c.function.emit(opDrillTier1, uint8(subOpSimdScaleSliceFloat64), sliceLocation.register, scalarLocation.register)
	c.function.emit(opExt, countRegister, 0, 0)
	return varLocation{}, true, nil
}

// emitSimdCountRegister materialises the runtime iteration count.
//
// Allocates a fresh int register. For simdBoundLenSlice the helper emits
// LEN_SLICE_FLOAT_DIRECT against the slice the matcher already resolved as the dominant
// operand (passed as fallbackSliceRegister); for simdBoundConst it loads the constant via
// LOAD_INT_CONST_SMALL when the value fits in 0..255 and falls back to the int-constant
// pool otherwise. A count that cannot be represented (e.g. negative const) triggers
// refusal so the recogniser falls back to the scalar loop emission.
//
// Takes c (*compiler) which carries the active emit state.
// Takes match (simdKernelMatch) which holds the bound info.
// Takes fallbackSliceRegister (uint8) which is the slicesFloat register used by
// LEN_SLICE_FLOAT_DIRECT in the simdBoundLenSlice case. The matcher requires the bound's
// slice to be one of the operand slices and the operand kinds were already validated, so
// this register is correct by construction.
//
// Returns uint8 which is the count int register.
// Returns bool which is true when emission succeeded.
// Returns error when constant pool allocation fails.
func emitSimdCountRegister(c *compiler, match simdKernelMatch, fallbackSliceRegister uint8) (uint8, bool, error) {
	countRegister := c.scopes.alloc.alloc(registerInt)
	switch match.boundShape {
	case simdBoundLenSlice:
		c.function.emit(opDrillTier1, uint8(subOpLenSliceFloatDirect), countRegister, fallbackSliceRegister)
		return countRegister, true, nil
	case simdBoundConst:
		if match.boundConstValue < 0 {
			return 0, false, nil
		}
		if match.boundConstValue <= maxUint8Value {
			//nolint:gosec // bound is range-checked above
			c.function.emit(opDrillTier1, uint8(subOpLoadIntConstSmall), countRegister, uint8(match.boundConstValue))
			return countRegister, true, nil
		}
		constantIndex, err := c.function.addIntConstant(match.boundConstValue)
		if err != nil {
			return 0, false, err
		}
		c.function.emitWide(opLoadIntConst, countRegister, constantIndex)
		return countRegister, true, nil
	default:
	}
	return 0, false, nil
}

// resolveScalarDestination looks up the register holding the already-declared scalar
// destination of a reduction kernel. The matcher already verified the AST node is an
// *ast.Ident referring to a float64 variable; this helper resolves it to a varLocation.
//
// Takes c (*compiler).
// Takes destinationExpression (ast.Expr) which must be an *ast.Ident referencing an
// in-scope float64.
//
// Returns the location and ok=true on success; the zero varLocation and ok=false when the
// lookup fails (which the recogniser treats as a refusal, falling emission through to the
// standard scalar compile path).
func resolveScalarDestination(c *compiler, destinationExpression ast.Expr) (varLocation, bool) {
	ident, ok := destinationExpression.(*ast.Ident)
	if !ok {
		return varLocation{}, false
	}
	location, found := c.scopes.lookupVar(ident.Name)
	if !found {
		return varLocation{}, false
	}
	if location.kind != registerFloat {
		return varLocation{}, false
	}
	return location, true
}

// resolveFloat64SliceOperand resolves a typed slicesFloat operand.
//
// When the operand already lives on the slicesFloat bank (the fast path, e.g. typed
// locals from `make([]float64, ...)`) the existing location is returned. When the operand
// is on the general bank (a reflect.Value holding []float64 from function parameters,
// returns of [][]float64 element accesses, etc.) the helper allocates a fresh slicesFloat
// register and emits a one-instruction adoption opcode that snapshots the slice header
// into the typed bank for the SIMD opcode's consumption. Spilled and indirect operands
// fall back to the scalar emission since the SIMD opcode encoding has no room for
// spill/indirection bookkeeping.
//
// Takes c (*compiler) which carries the active emit state.
// Takes operandExpression (ast.Expr) which must be an *ast.Ident referencing an in-scope
// []float64 variable.
//
// Returns varLocation which is the typed-bank location.
// Returns bool which is true when the operand was resolved.
// Returns error when adoption emission fails.
func resolveFloat64SliceOperand(c *compiler, operandExpression ast.Expr) (varLocation, bool, error) {
	ident, ok := operandExpression.(*ast.Ident)
	if !ok {
		return varLocation{}, false, nil
	}
	location, found := c.scopes.lookupVar(ident.Name)
	if !found {
		return varLocation{}, false, nil
	}
	if location.isSpilled || location.isIndirect {
		return varLocation{}, false, nil
	}
	switch location.kind {
	case registerSliceFloat:
		return location, true, nil
	case registerGeneral:
		adoptedRegister := c.scopes.alloc.alloc(registerSliceFloat)
		c.function.emit(opDrillTier1, uint8(subOpAdoptGeneralToSlicesFloat), adoptedRegister, location.register)
		return varLocation{register: adoptedRegister, kind: registerSliceFloat}, true, nil
	default:
	}
	return varLocation{}, false, nil
}

// extractCanonicalLoopVarIdent returns the *ast.Ident of the loop variable when init has
// the canonical shape `i := 0` or `i = 0`.
//
// Takes init (ast.Stmt) which is the for-statement's init clause.
//
// Returns the loop var identifier and ok=true on success.
func extractCanonicalLoopVarIdent(init ast.Stmt) (*ast.Ident, bool) {
	assign, ok := init.(*ast.AssignStmt)
	if !ok {
		return nil, false
	}
	if len(assign.Lhs) != 1 {
		return nil, false
	}
	ident, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, false
	}
	return ident, true
}

// extractCondLeftIdent returns the *ast.Ident on the left side of the loop condition when
// it has shape `i < expr` or `i <= expr`.
//
// Takes condition (ast.Expr) which is the cond clause.
//
// Returns the identifier or nil.
func extractCondLeftIdent(condition ast.Expr) *ast.Ident {
	binary, ok := condition.(*ast.BinaryExpr)
	if !ok {
		return nil
	}
	ident, ok := binary.X.(*ast.Ident)
	if !ok {
		return nil
	}
	return ident
}

// extractPostIdent returns the *ast.Ident the post clause increments or decrements (`i++`
// / `i--`).
//
// Takes post (ast.Stmt) which is the post clause.
//
// Returns the identifier or nil.
func extractPostIdent(post ast.Stmt) *ast.Ident {
	incDec, ok := post.(*ast.IncDecStmt)
	if !ok {
		return nil
	}
	ident, ok := incDec.X.(*ast.Ident)
	if !ok {
		return nil
	}
	return ident
}

// extractLenSliceIdent returns the *ast.Ident of the slice that appears inside the cond's
// `len(slice)` call when condShape is forCondLtLen.
//
// Takes upperBound (ast.Expr) which is the cond's RHS expression the classifier
// extracted.
//
// Returns the slice identifier and ok=true on success.
func extractLenSliceIdent(upperBound ast.Expr) (*ast.Ident, bool) {
	return matchLenCall(upperBound)
}

// isIdentNameEqual reports whether ident is non-nil and has the given name.
//
// Takes ident (*ast.Ident) which may be nil.
// Takes name (string) which is the expected name.
//
// Returns true when ident is non-nil and ident.Name == name.
func isIdentNameEqual(ident *ast.Ident, name string) bool {
	return ident != nil && ident.Name == name
}

// isSameIdent reports whether two AST expressions are both *ast.Ident with the same Name.
// Used for the no-aliasing rule: operand slices must be distinct from destinations.
//
// Takes a (ast.Expr).
// Takes b (ast.Expr).
//
// Returns true when both are identifiers and share a Name.
func isSameIdent(a, b ast.Expr) bool {
	aIdent, aOk := a.(*ast.Ident)
	bIdent, bOk := b.(*ast.Ident)
	if !aOk || !bOk {
		return false
	}
	return aIdent.Name == bIdent.Name
}

// indexedSliceWithLoopVar returns the slice *ast.Ident when expr is `slice[loopVar]` for
// the given loop variable. Refuses any other index expression (constant index, computed
// index, nested index).
//
// Takes expr (ast.Expr) which is the candidate IndexExpr.
// Takes loopVar (*ast.Ident) which is the loop variable.
//
// Returns the slice identifier and ok=true on success.
func indexedSliceWithLoopVar(expr ast.Expr, loopVar *ast.Ident) (*ast.Ident, bool) {
	indexExpr, ok := expr.(*ast.IndexExpr)
	if !ok {
		return nil, false
	}
	sliceIdent, ok := indexExpr.X.(*ast.Ident)
	if !ok {
		return nil, false
	}
	indexIdent, ok := indexExpr.Index.(*ast.Ident)
	if !ok {
		return nil, false
	}
	if indexIdent.Name != loopVar.Name {
		return nil, false
	}
	return sliceIdent, true
}

// isFloat64Slice reports whether expr has type []float64 via go/types info.
//
// Takes info (*types.Info) which carries the type-checker output.
// Takes expr (ast.Expr) which is the candidate expression.
//
// Returns true when expr is a []float64.
func isFloat64Slice(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	typeAndValue, ok := info.Types[expr]
	if !ok || typeAndValue.Type == nil {
		return false
	}
	sliceType, ok := typeAndValue.Type.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	basic, ok := sliceType.Elem().Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return basic.Kind() == types.Float64
}

// isFloat64Scalar reports whether expr has type float64.
//
// Takes info (*types.Info) which carries the type-checker output.
// Takes expr (ast.Expr) which is the candidate expression.
//
// Returns true when expr is a float64.
func isFloat64Scalar(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	typeAndValue, ok := info.Types[expr]
	if !ok || typeAndValue.Type == nil {
		return false
	}
	basic, ok := typeAndValue.Type.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return basic.Kind() == types.Float64
}

// isOneOfSlices reports whether candidate matches any comparison.
//
// Compares identifiers by Name. Used to verify that the loop's `len(slice)` upper bound
// names one of the operand slices, otherwise the loop's iteration count is not the
// operand slice length and the SIMD kernel would over- or under-iterate.
//
// Takes candidate (*ast.Ident) which is the slice named in len(slice).
// Takes comparisons (variadic *ast.Ident) which are the operand slices the kernel will
// work over.
//
// Returns true when candidate.Name matches any comparison's Name.
func isOneOfSlices(candidate *ast.Ident, comparisons ...*ast.Ident) bool {
	for _, comparison := range comparisons {
		if comparison != nil && candidate.Name == comparison.Name {
			return true
		}
	}
	return false
}

func init() { //nolint:gochecknoinits // process-wide recogniser registration.
	if !simdKernelRecogniserEnabled {
		return
	}
	registerForStmtRecogniser(&simdKernelRecogniser{})
}
