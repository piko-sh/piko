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
	"go/types"
	"math"
	"reflect"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// blankIdentName is the Go blank identifier "_", used to discard values in assignments
	// and declarations.
	blankIdentName = "_"

	// sentinelFieldDeref is the field-index sentinel used with opSetField to indicate "set
	// via pointer dereference" rather than an actual struct field.
	sentinelFieldDeref uint8 = 255

	// maxSmallConstant is the largest value that fits in opLoadIntConstSmall (single-byte
	// immediate 0-255).
	maxSmallConstant int64 = 255

	// sliceMaxBitFlag is the bit flag ORed into the flags byte of opSliceOp when a
	// three-index slice expression has a max bound.
	sliceMaxBitFlag uint8 = 4

	// rangeOverFuncReturnPendingFlag is the state-flag value that signals a return statement
	// was encountered inside the yield body.
	rangeOverFuncReturnPendingFlag int64 = 2

	// rangeOverFuncFirstLabelFlag is the first state-flag value assigned to labelled
	// outer-loop targets in range-over-func bodies. Values 0-2 are reserved
	// (normal/break/return-pending).
	rangeOverFuncFirstLabelFlag int64 = 3

	// commaOkResultCount is the number of LHS variables in a comma-ok assignment (v, ok :=
	// ...).
	commaOkResultCount = 2

	// sliceLowBoundFlag is the bit flag indicating the low bound is present in a slice or
	// string-slice operation.
	sliceLowBoundFlag uint8 = 1

	// sliceHighBoundFlag is the bit flag indicating the high bound is present in a slice or
	// string-slice operation.
	sliceHighBoundFlag uint8 = 2

	// rangeKeyFlag is the bit flag indicating a key variable is present in a range
	// iteration.
	rangeKeyFlag uint8 = 1

	// rangeValueFlag is the bit flag indicating a value variable is present in a range
	// iteration.
	rangeValueFlag uint8 = 2

	// identTrue is the Go predeclared identifier for the boolean true.
	identTrue = "true"

	// identFalse is the Go predeclared identifier for the boolean false.
	identFalse = "false"

	// identNil is the Go predeclared identifier for the untyped nil.
	identNil = "nil"

	// initFuncName is the name of Go init functions.
	initFuncName = "init"

	// evalFuncName is the synthetic function name used by the interpreter for eval snippet
	// execution.
	evalFuncName = "_eval_"

	// replaceAllArgCount is the expected argument count for strings.ReplaceAll intrinsic
	// compilation.
	replaceAllArgCount = 3

	// makeSliceMinCapArgs is the minimum argument count for a make() call to include an
	// explicit capacity argument.
	makeSliceMinCapArgs = 3

	// wideBitShift is the number of bits to shift when encoding or decoding wide (16-bit)
	// instruction operands from B|(C<<8).
	wideBitShift = 8

	// initialFileTableCapacity is the initial capacity for the source map's shared file
	// table during debug compilation.
	initialFileTableCapacity = 4

	// defaultMaxExpressionDepth caps recursion when compiling expressions.
	//
	// The bound defends the host process against stack exhaustion from pathologically deep
	// input expressions. Overridable per Service via WithMaxExpressionDepth; the compiler
	// reads the limit through compiler.expressionDepthLimit.
	defaultMaxExpressionDepth = 1024

	// maxStatementDepth caps recursion when compiling statements.
	//
	// compileStmt recurses through block, if/for/switch/select bodies and closure literals;
	// without a bound pathologically nested input could exhaust the host goroutine stack.
	// The cap mirrors defaultMaxExpressionDepth's intent for the statement walk. Each
	// closure body compiles on a fresh compiler instance whose statementDepth starts at
	// zero, so the cap is per-function rather than cumulative across closure nesting.
	maxStatementDepth = 1024

	// compileLoopCheckMask is the iteration-count mask used by the main statement-list walk
	// to amortise the cost of polling ctx.Err() across every (mask+1) statements, mirroring
	// optimisationLoopCheckMask.
	compileLoopCheckMask = 1023

	// optimisationLoopCheckMask is the iteration-count mask used by long-running
	// optimisation passes to amortise the cost of polling ctx.Err() across every (mask+1)
	// iterations.
	optimisationLoopCheckMask = 1023
)

// globalVariableInfo describes a package-level variable's location in the globalStore.
type globalVariableInfo struct {
	// index is the slot index within the globalStore for the variable.
	index int

	// kind is the register kind that determines which typed store holds the variable.
	kind registerKind
}

// compiler translates type-checked Go AST into bytecode. Field order is
// alignment-optimised (see fieldalignment lint); the logical grouping (compilation state,
// generic substitution, escape and heap-promotion state, debug, defer/loop
// classification) is captured in the per-field comments below.
type compiler struct {
	// stickyError records the first non-recoverable emit error.
	//
	// Captures errors observed by void-returning emit helpers (e.g. constant-pool overflow
	// inside emitLocalZeroValue, emitGlobalGeneralConst, registerMethodReceiver). Compile
	// entry points consult resourceError after the body has been emitted so the host
	// surfaces the first such failure instead of silently dropping it.
	stickyError error

	// typeSubstitutionsCache memoises substituteType results within the lifetime of a single
	// specialisation compile. Reset alongside typeSubstitutions when descending into a
	// non-specialised closure.
	typeSubstitutionsCache map[types.Type]types.Type

	// inPlaceAppendAliases names locals with potentially aliased slices.
	//
	// Built once at function entry by collectInPlaceAppendAliases. Consulted by
	// lookupInPlaceAppendTarget to refuse opAppendByteFastInPlace emission for slices that
	// have outstanding aliases (the in-place opcode mutates the arenaSliceHeader slot, and
	// any aliased register would observe the mutation, breaking Go's slice-header value
	// semantics for patterns like `saved := output; output = append(output, b)`). Nil when
	// no aliasing patterns were detected in the function.
	inPlaceAppendAliases map[string]bool

	// scopes tracks nested lexical scopes for register allocation.
	scopes *scopeStack

	// rootFunction is the top-level function where all compiled functions are registered.
	// For sub-compilers, this points to the parent's root function.
	rootFunction *CompiledFunction

	// globals is the runtime global store used for allocating package-level variable slots
	// at compile time.
	globals *globalStore

	// symbols provides access to pre-registered native symbols for resolving imported
	// package references at compile time.
	symbols *SymbolRegistry

	// rangeOverFunc is non-nil when compiling a range-over-func yield callback body.
	// Controls break/continue/return transformation.
	rangeOverFunc *rangeOverFuncContext

	// info is the type information from go/types.Check().
	info *types.Info

	// globalVariables maps package-level variable names to their location in the
	// globalStore. Shared across all compiler instances for the same compilation unit.
	globalVariables map[string]globalVariableInfo

	// fileSet is the file set from parsing.
	fileSet *token.FileSet

	// reflectTypeCache caches reflect.Type synthesis for named types (including anonymous
	// struct types encountered along the recursion). Caching is essential for mutually
	// recursive named types, where each cycle-detection entry point would otherwise produce
	// a structurally different reflect.Type for the same nominal type, breaking value
	// assignments across entry points.
	reflectTypeCache map[types.Type]reflect.Type

	// upvalueMap maps captured variable names to their upvalue index and register kind. Only
	// set when compiling a closure body.
	upvalueMap map[string]upvalueReference

	// heapPromotedNames lists local names that must be heap-promoted at declaration.
	// Populated by a pre-pass at function entry so each declareVar site emits
	// opAllocIndirect immediately, giving the variable a stable addressable cell that
	// closures and goroutines can read and write by pointer.
	heapPromotedNames map[string]bool

	// typedSliceLocals maps qualifying typed-slice local names to their registerKind.
	//
	// Populated by classifyTypedSliceLocals at function entry. A local qualifies only when
	// every reference is indexed element read/write, len(), cap(), range iteration, or a
	// make([]T, ...) initialiser; address-of, append, copy, argument passing, container
	// insertion, type assertion, or return all disqualify the local and leave it on the
	// reflect path. Supported kinds: registerSliceInt, registerSliceFloat,
	// registerSliceString, registerSliceBool, registerSliceUint.
	typedSliceLocals map[string]registerKind

	// closureCapturedNames is the full set of names captured by any inner closure.
	//
	// Includes scalar captures that do not require heap-promotion. Gates opResetSharedCell
	// so each loop iteration produces a fresh closure snapshot rather than aliasing the cell
	// from the preceding iteration.
	closureCapturedNames map[string]bool

	// debugFileIDs deduplicates file names to fileID indices in the source map's files
	// slice.
	debugFileIDs map[string]uint16

	// writtenLocalNames lists locals written after their declaration.
	//
	// Detected writes include direct assignment, compound assignment, inc/dec, indirect or
	// member write rooted at the name, address-of, or non-`:=` range bind. Names absent from
	// this map are read-only, letting emitValueCopyForLocalAssignment skip the snapshot copy
	// for struct/array `:=` initialisers. Conservatively name-scoped: a write to `x` in any
	// nested scope marks all same-named locals as written; false positives only cost an
	// unnecessary snapshot.
	writtenLocalNames map[string]bool

	// funcTable maps function names to indices in rootFunction.functions.
	funcTable map[string]uint16

	// structLayoutIndex deduplicates struct-field layout entries across access sites.
	//
	// Keyed by (struct reflect.Type, joined field path); values are indexes into
	// c.function.structLayoutTable. Lazily allocated and cleared per function body.
	structLayoutIndex map[structFieldLayoutKey]uint16

	// labelTable maps label names to their instruction PCs. Set when an *ast.LabeledStmt is
	// encountered during compilation.
	labelTable map[string]int

	// forwardGotos holds goto jumps targeting labels that have not yet been declared,
	// indexed by label name and patched once the label is visited.
	forwardGotos map[string][]int

	// debugSourceMap is the source map being built during compilation. Nil when debug info
	// is disabled.
	debugSourceMap *sourceMap

	// function is the current function being compiled.
	function *CompiledFunction

	// typeSubstitutions maps generic TypeParams to concrete instantiation types. Nil for
	// non-specialised compilers; when non-nil, kindFor / reflectFor / staticTypeOf consult
	// this map so emit decisions resolve TypeParams to the concrete type and pick typed-bank
	// opcodes.
	typeSubstitutions map[*types.TypeParam]types.Type

	// escapeAllocSitePCs maps each heap-promoted local name to the bytecode PC of its
	// opAllocIndirect emit site.
	//
	// Populated by promoteToIndirect; consumed by classifyLocalEscapes to build
	// CompiledFunction.arenaSafeAllocPCs. Per-function scratch only.
	escapeAllocSitePCs map[string]int

	// loopPostInitDeclRegisters tracks init-declared loop register pairs.
	//
	// Contains (register, kind) entries for variables declared in the for-stmt init clause
	// whose post statement is currently being compiled. emitSyncCaptured suppresses
	// opWriteSharedCell only when a destination matches one of these entries: post-statement
	// mutations of *init-declared* loop variables target a separate per-iteration cell (Go
	// 1.22+ per-iter scoping) and must not bleed into the closures captured in the current
	// iteration. Variables declared OUTSIDE the for-stmt (`for ; i < 3; i++`, where i is in
	// the enclosing scope) share one cell across all iterations and MUST receive the sync
	// write so captured closures see the post-loop final value. Populated by compileFor
	// before entering the post and cleared after.
	loopPostInitDeclRegisters map[loopPostDeclKey]struct{}

	// pendingLabel is set when compiling a labelled statement, so that the inner loop or
	// switch can attach it to its breakable context for labelled break/continue.
	pendingLabel string

	// breakables is a stack of contexts for break/continue targets. Loops and switches push
	// onto this stack.
	breakables []breakableContext

	// currentResultTypes records declared result types in source order.
	//
	// Populated by compileFuncBody before the body is emitted. compileReturnExprs consults
	// the list to detect when a bare `nil` identifier needs to be promoted to a typed-zero
	// load (e.g. `return nil` from `func() *Holder`) so the returned reflect.Value retains
	// its concrete type tag and downstream method dispatch and interface comparison stays
	// Go-spec-compliant. Nil for functions with no declared results or before the body
	// compile begins.
	currentResultTypes []types.Type

	// initFunctionIndices holds indices (into rootFunction.functions) of init() functions in
	// source order, for auto-execution before _eval_.
	initFunctionIndices []uint16

	// expressionDepth tracks the current depth of compileExpression recursion, capped at
	// expressionDepthLimit so that pathologically deep input cannot exhaust the host stack.
	expressionDepth int

	// maxLiteralElements is the maximum number of elements in a single composite literal.
	// Zero means unlimited.
	maxLiteralElements int

	// loopDepth tracks nested loop bodies so compileDefer can check whether the defer it is
	// compiling lives inside a loop. compileFor / compileForRange / compileSelect with loop
	// semantics increment on entry and decrement on exit.
	loopDepth int

	// deferCount counts defer statements compiled in the current function body. Combined
	// with hasRecover, deferInLoop, and the per-defer thisDeferTrivial classification it
	// determines whether the function qualifies for the trivial-defer fast path.
	deferCount int

	// maxExpressionDepth caps the recursion depth when compiling nested expressions. Zero
	// means use defaultMaxExpressionDepth (1024).
	maxExpressionDepth int

	// currentPosition is the current source position set before compiling each statement or
	// expression. Used by the emit hook to record source positions.
	currentPosition token.Pos

	// statementDepth tracks the current compileStmt recursion depth.
	//
	// Walks through nested blocks, control-flow bodies, and closure literals, capped at
	// maxStatementDepth so that pathologically nested input cannot exhaust the host
	// goroutine stack. Each closure body compiles on a fresh compiler instance, so the
	// counter resets to zero per function rather than accumulating across closure nesting.
	statementDepth int

	// features controls which Go language constructs are allowed during compilation.
	features InterpFeature

	// deferInLoop is true when a defer statement was compiled inside a loop body (for,
	// range, select with persistent select-case loop). Per-iteration registration
	// disqualifies the trivial-defer fast path because there is more than one defer per
	// call.
	deferInLoop bool

	// inLoopPost is true while compiling a for-loop post statement (e.g. i++), suppressing
	// opWriteSharedCell because Go 1.22+ per-iteration scoping means the post statement
	// mutates the *next* iteration's variable, not the current iteration's captured cell.
	inLoopPost bool

	// hasDefers is set to true when a defer statement is compiled. Used to suppress tail
	// call optimisation when defers are present.
	hasDefers bool

	// debugEnabled is true when debug info generation is active.
	debugEnabled bool

	// thisDeferTrivial is true when the most recently compiled defer matched the trivial
	// shape (direct method-value or top-level func target, no closure capture, args
	// resolvable from existing registers). The downgrade pass combines this with deferCount
	// and hasRecover to decide fast-path eligibility.
	thisDeferTrivial bool

	// simpleDeferArgCount records the deferred call's evaluated arg count when
	// classification succeeds. Used by the runtime arena's EnsureCapacity to pre-size the
	// simpleDeferArgs buffer in callee frames for any function that registers a trivial
	// defer.
	simpleDeferArgCount uint8

	// hasRecover is true when an AST walk over the current function body found a call to the
	// predeclared recover() builtin. Set once at the start of compileFuncBody so
	// compileDefer can read it before classifying any defer it encounters.
	hasRecover bool
}

// recordStickyError stores the first non-recoverable emit error.
//
// Subsequent errors are dropped to avoid thrashing the field on cascaded failures.
//
// Takes err (error) which is the candidate error to record.
func (c *compiler) recordStickyError(err error) {
	if err == nil || c.stickyError != nil {
		return
	}
	c.stickyError = err
}

// resourceError returns the first sticky error recorded during emit.
//
// Returns the overflowError result when no sticky error has been recorded. Used by
// compile entry points to fail fast when a void helper observed a
// pool/method/specialisation overflow.
//
// Returns error which is the first sticky or overflow error, or nil.
func (c *compiler) resourceError() error {
	if c.stickyError != nil {
		return c.stickyError
	}
	return c.scopes.overflowError()
}

// expressionDepthLimit returns the configured expression-depth ceiling.
//
// Substitutes the package default when none was set.
//
// Returns int which is the active expression-depth ceiling.
func (c *compiler) expressionDepthLimit() int {
	if c.maxExpressionDepth > 0 {
		return c.maxExpressionDepth
	}
	return defaultMaxExpressionDepth
}

// substitutedType returns t with each TypeParam replaced by its concrete instantiation.
//
// When the substitution map is nil (the common non-generic case), t is returned
// unchanged.
//
// Takes t: the type to substitute through the active generic map.
//
// Returns the substituted type, or t unchanged when no substitutions apply.
func (c *compiler) substitutedType(t types.Type) types.Type {
	if c == nil || c.typeSubstitutions == nil || t == nil {
		return t
	}
	return substituteType(t, c.typeSubstitutions, c.typeSubstitutionsCache)
}

// kindFor maps t through the active substitution and then to a registerKind.
//
// Specialised bodies pick typed-bank kinds for the concrete instantiation rather than
// registerGeneral.
//
// Takes t: the type whose register kind is required.
//
// Returns the registerKind selected for the substituted type.
func (c *compiler) kindFor(t types.Type) registerKind {
	return kindForType(c.substitutedType(t))
}

// promotionContext builds a kindPromotionContext.
//
// The context is seeded with the compiler's active generic substitution map. Used by the
// boundary sites that consult kindForPromotedSlot. Sites that need additional gates
// (binding-name disqualifier, callee-vector consultation) can take the returned context
// and set the remaining fields before passing it to the predicate.
//
// Returns nil when the compiler has no substitution map (the non-generic /
// non-specialised case); the predicate is nil-safe so callers can pass the result
// directly.
//
// Returns *kindPromotionContext which holds c.typeSubstitutions and
// c.typeSubstitutionsCache, or nil when no substitutions are active.
func (c *compiler) promotionContext() *kindPromotionContext {
	if c == nil || c.typeSubstitutions == nil {
		return nil
	}
	return &kindPromotionContext{
		substitutions:     c.typeSubstitutions,
		substitutionCache: c.typeSubstitutionsCache,
	}
}

// kindForCallSlot picks the call-slot register kind for t.
//
// Routes t through the unified ARCH5 promotion predicate so parameter and return slots
// take the same typed-bank decision as every other ARCH5 boundary site (struct-field
// reads, generic specialisation, escape gate). Slice types route to their typed bank when
// both the type and the active substitution admit it; everything else falls back to the
// type-only verdict.
//
// Takes t (types.Type) which is the call-slot type.
//
// Returns the register kind for the slot.
func (c *compiler) kindForCallSlot(t types.Type) registerKind {
	kind, _ := kindForPromotedSlot(t, c.promotionContext())
	return kind
}

// parameterSlotKind picks the register-kind for a parameter slot.
//
// The variadic last parameter is held on the general bank: the caller side packs varargs
// into a heterogeneous reflect-backed slice that the callee unpacks via reflect, so a
// typed-bank slot would mismatch the packing ABI. Non-variadic slots route through
// kindForCallSlot so primitive-typed slices land on their typed bank.
//
// Takes signature (*types.Signature) which carries the variadic flag.
// Takes parameterType (types.Type) which is the parameter's type.
// Takes parameterIndex (int) which is the parameter's position within Params().
// Takes parameterCount (int) which is the total number of parameters.
//
// Returns the register kind for the slot.
func (c *compiler) parameterSlotKind(signature *types.Signature, parameterType types.Type, parameterIndex, parameterCount int) registerKind {
	if signature.Variadic() && parameterIndex == parameterCount-1 {
		return c.kindFor(parameterType)
	}
	return c.kindForCallSlot(parameterType)
}

// underlyingTypeOf returns the underlying type recorded by go/types for expr.
//
// The compiler walks user-supplied, go/types-checked AST. On malformed or only partially
// type-checked input the c.info.Types map may have no entry for expr, or carry a
// TypeAndValue whose Type is nil. Either case would nil-dereference if the caller chained
// .Type.Underlying() directly, crashing the host. This helper performs the ok-check on
// the map read and the nil-check on the TypeAndValue.Type so callers can surface a clean
// compile error instead.
//
// Takes expr (ast.Expr) which is the expression whose underlying type is required.
//
// Returns the underlying type and true when go/types recorded a non-nil type for expr,
// and (nil, false) otherwise.
func (c *compiler) underlyingTypeOf(expr ast.Expr) (types.Type, bool) {
	tv, ok := c.info.Types[expr]
	if !ok || tv.Type == nil {
		return nil, false
	}
	return tv.Type.Underlying(), true
}

// typeToReflect converts a go/types.Type to a reflect.Type.
//
// Threads the compiler's shared cache so mutually recursive named types yield identical
// reflect.Types across all call sites.
//
// Takes ctx: cancellation context forwarded to the reflect synthesis.
// Takes t: the source type to convert.
//
// Returns the reflect.Type matching t after generic substitution.
func (c *compiler) typeToReflect(ctx context.Context, t types.Type) reflect.Type {
	if c.reflectTypeCache == nil {
		c.reflectTypeCache = make(map[types.Type]reflect.Type)
	}
	t = c.substitutedType(t)
	return typeToReflectCached(ctx, t, c.symbols, c.reflectTypeCache, c.globals)
}

// checkFeature returns an error when feature is not enabled in the compiler's feature
// set.
//
// Takes feature: the interpreter feature being checked.
// Takes pos: the source position formatted into the error message.
//
// Returns nil when the feature is enabled.
//
// Returns errFeatureNotAllowed when the feature is disabled.
func (c *compiler) checkFeature(feature InterpFeature, pos token.Pos) error {
	if c.features.Has(feature) {
		return nil
	}
	return fmt.Errorf("%w: %s at %s", errFeatureNotAllowed, feature, c.fileSet.Position(pos))
}

// rangeOverFuncContext holds state for compiling the yield callback body of a
// range-over-func loop. It transforms break/continue/return into yield return values and
// state flag mutations.
type rangeOverFuncContext struct {
	// returnStashUpvalueIndices are the upvalue indices for stashing return values when a
	// return statement is encountered inside the range-over-func body.
	returnStashUpvalueIndices []int

	// returnKinds are the register kinds of the enclosing function's return values, used to
	// emit the correct opGetUpvalue after the iterator call.
	returnKinds []registerKind

	// outerLabels are labelled break/continue targets from enclosing loops. Enables
	// cross-closure labelled break/continue by encoding the target as a state flag value.
	outerLabels []outerLabelTarget

	// stateFlagUpvalueIndex is the upvalue index for the state flag register in the yield
	// closure. 0=normal, 1=break, 2=return-pending, 3+=labelled break/continue to outer
	// loops.
	stateFlagUpvalueIndex int
}

// outerLabelTarget describes a labelled loop in an enclosing scope that can be targeted
// by break/continue from within a range-over-func yield body.
type outerLabelTarget struct {
	// label is the name of the labelled loop target in the enclosing scope.
	label string

	// breakFlag is the state flag value for labelled break.
	breakFlag int64

	// continueFlag is the state flag value for labelled continue (0 if not a loop).
	continueFlag int64

	// breakableIndex is the index into the outer compiler's breakables slice.
	breakableIndex int
}

// upvalueReference tracks a captured variable's upvalue index and kind within a closure
// being compiled.
type upvalueReference struct {
	// index is the upvalue slot index within the closure's upvalue table.
	index int

	// kind is the register kind the closure body sees the upvalue as.
	//
	// For non-promoted captures this matches the parent's register bank. For heap-promoted
	// captures (isIndirect is true) this is originalKind so the closure body can keep
	// typed-bank semantics while the cell stores a *T pointer.
	kind registerKind

	// isIndirect is true when the captured variable is heap-promoted.
	//
	// compileIdent emits opGetUpvalue with kind registerGeneral to fetch the *T pointer,
	// then opDeref + opUnpackInterface to recover the typed value into the closure's typed
	// register. compileAssignTarget mirrors the pattern with opPackInterface + indirect
	// write.
	isIndirect bool

	// originalKind preserves the typed bank the variable had pre-promotion.
	//
	// The closure body's source-level type matches originalKind even though the cell stores
	// a pointer. Meaningful only when isIndirect is true.
	originalKind registerKind
}

// breakableContext tracks pending break and continue jumps for a loop or switch
// statement.
type breakableContext struct {
	// label is the optional label name for labelled break/continue.
	label string

	// breakJumps holds instruction offsets of break jumps to patch.
	breakJumps []int

	// continueJumps holds instruction offsets of continue jumps to patch. Only meaningful
	// for loops.
	continueJumps []int

	// fallthroughJumps holds instruction offsets of fallthrough jumps to patch. Only
	// meaningful for switch statements.
	fallthroughJumps []int

	// isLoop is true for for-loops, false for switch statements.
	isLoop bool
}

// coerceEvalBoolResult converts a registerInt result to registerBool when the
// expression's static type is bool.
//
// Eval returns a bool regardless of whether the value was constant-folded or computed at
// runtime.
//
// Takes info: type information used to read the expression's static type.
// Takes expression: the AST node whose result is being coerced.
// Takes location: the source register holding the comparison or logical result.
//
// Returns the coerced varLocation in registerBool, or the original location when no
// coercion is required.
func (c *compiler) coerceEvalBoolResult(_ context.Context, info *types.Info, expression ast.Expr, location varLocation) varLocation {
	if location.kind != registerInt {
		return location
	}
	tv, ok := info.Types[expression]
	if !ok {
		return location
	}
	basic, bOk := tv.Type.Underlying().(*types.Basic)
	if !bOk || basic.Kind() != types.Bool {
		return location
	}
	booleanRegister := c.scopes.alloc.alloc(registerBool)
	c.function.emit(opDrillTier1, uint8(subOpIntToBool), booleanRegister, location.register)
	return varLocation{register: booleanRegister, kind: registerBool}
}

// compileStmtList compiles a list of statements in order.
//
// Intermediate registers from non-declaring statements are reclaimed via a watermark
// restore between statements. Declaring statements (:= and var) are exempt because their
// registers must survive to subsequent statements.
//
// Takes statements: the statements to compile in source order.
//
// Returns the result location of the last statement, or the zero varLocation when the
// list is empty.
//
// Returns the first compilation error encountered.
func (c *compiler) compileStmtList(ctx context.Context, statements []ast.Stmt) (varLocation, error) {
	lastUseIndices := computeLastUseIndices(statements)

	var activeDeclarations []activeDeclaration
	var lastLocation varLocation
	for i, statement := range statements {
		if i&compileLoopCheckMask == 0 {
			if err := ctx.Err(); err != nil {
				return varLocation{}, err
			}
		}
		watermark := c.scopes.alloc.snapshot()
		location, err := c.compileStmt(ctx, statement)
		if err != nil {
			return varLocation{}, err
		}
		lastLocation = location

		activeDeclarations = c.trackOrRestoreDeclarations(statement, watermark, activeDeclarations)
		activeDeclarations = c.recycleDeadDeclarations(activeDeclarations, lastUseIndices, i)
	}
	return lastLocation, nil
}

// compileStmt compiles a single statement, dispatching by AST node type.
//
// Takes statement: the statement node to compile.
//
// Returns the result location produced by the compiled statement.
//
// Returns an error when the statement type is unsupported.
func (c *compiler) compileStmt(ctx context.Context, statement ast.Stmt) (varLocation, error) {
	if c.statementDepth >= maxStatementDepth {
		return varLocation{}, fmt.Errorf("%w: statement nesting exceeded %d at %s", errCompileDepthLimit, maxStatementDepth, c.positionString(statement.Pos()))
	}
	c.statementDepth++
	defer func() { c.statementDepth-- }()

	c.setDebugPosition(ctx, statement.Pos())
	if location, handled, err := c.compileConcurrencyStmt(ctx, statement); handled {
		return location, err
	}
	switch s := statement.(type) {
	case *ast.ExprStmt:
		return c.compileExpression(ctx, s.X)
	case *ast.AssignStmt:
		return c.compileAssign(ctx, s)
	case *ast.DeclStmt:
		return c.compileDecl(ctx, s.Decl)
	case *ast.ReturnStmt:
		return c.compileReturn(ctx, s)
	case *ast.BlockStmt:
		c.scopes.pushScope()
		location, err := c.compileStmtList(ctx, s.List)
		c.scopes.popScope()
		return location, err
	case *ast.IfStmt:
		return c.compileIf(ctx, s)
	case *ast.ForStmt:
		return c.compileFor(ctx, s)
	case *ast.IncDecStmt:
		return c.compileIncDec(ctx, s)
	case *ast.BranchStmt:
		return c.compileBranch(ctx, s)
	case *ast.SwitchStmt:
		return c.compileSwitch(ctx, s)
	case *ast.RangeStmt:
		return c.compileForRange(ctx, s)
	case *ast.EmptyStmt:
		return varLocation{}, nil
	case *ast.LabeledStmt:
		return c.compileLabeledStmt(ctx, s)
	default:
		return varLocation{}, fmt.Errorf("unsupported statement type: %T at %s", statement, c.positionString(statement.Pos()))
	}
}

// compileConcurrencyStmt compiles the goroutine, deferral, and channel statement kinds,
// keeping the main compileStmt dispatch small enough to stay within the
// cyclomatic-complexity budget.
//
// Takes statement (ast.Stmt) which is the statement node to compile.
//
// Returns the compiled statement's result location, a handled flag that is true only when
// statement was one of the concurrency-related kinds this helper owns, and any
// compilation error.
func (c *compiler) compileConcurrencyStmt(ctx context.Context, statement ast.Stmt) (location varLocation, handled bool, err error) {
	switch s := statement.(type) {
	case *ast.DeferStmt:
		location, err = c.compileDefer(ctx, s)
		return location, true, err
	case *ast.GoStmt:
		location, err = c.compileGo(ctx, s)
		return location, true, err
	case *ast.SendStmt:
		location, err = c.compileSend(ctx, s)
		return location, true, err
	case *ast.TypeSwitchStmt:
		location, err = c.compileTypeSwitch(ctx, s)
		return location, true, err
	case *ast.SelectStmt:
		location, err = c.compileSelect(ctx, s)
		return location, true, err
	default:
		return varLocation{}, false, nil
	}
}

// compileDecl compiles a declaration node, dispatching by concrete AST type.
//
// Takes declaration: the declaration node to compile.
//
// Returns the location of the last compiled spec.
//
// Returns an error when the declaration type is unsupported.
func (c *compiler) compileDecl(ctx context.Context, declaration ast.Decl) (varLocation, error) {
	switch d := declaration.(type) {
	case *ast.GenDecl:
		return c.compileGenDecl(ctx, d)
	default:
		return varLocation{}, fmt.Errorf("unsupported declaration type: %T at %s", declaration, c.positionString(declaration.Pos()))
	}
}

// registerPackageLevelVar allocates a globalStore slot for each name declared in spec.
//
// Dispatches on the variable's register kind to the matching typed allocator. Invoked
// during the first compilation pass before any function body is emitted.
//
// Takes spec: the value specification declaring one or more package-level variables.
func (c *compiler) registerPackageLevelVar(_ context.Context, spec *ast.ValueSpec) {
	for _, name := range spec.Names {
		if name.Name == blankIdentName {
			continue
		}
		typeObject := c.info.Defs[name]
		if typeObject == nil {
			continue
		}
		kind := kindForType(typeObject.Type())
		var index int
		switch kind {
		case registerInt:
			index = c.globals.allocInt(0)
		case registerFloat:
			index = c.globals.allocFloat(0)
		case registerString:
			index = c.globals.allocString("")
		case registerBool:
			index = c.globals.allocBool(false)
		case registerUint:
			index = c.globals.allocUint(0)
		case registerComplex:
			index = c.globals.allocComplex(0)
		case registerGeneral:
			index = c.globals.allocGeneral(reflect.Value{})
		default:
		}
		c.globalVariables[name.Name] = globalVariableInfo{index: index, kind: kind}
	}
}

// compilePackageLevelVarInit emits bytecode to initialise each package-level variable in
// spec.
//
// Vars with explicit initialisers compile the expression; zero-value vars emit opLoadZero
// + opSetGlobal so the varinit function can be re-run to reset globals between Execute
// calls.
//
// Takes spec: the value specification holding one or more package-level variables.
//
// Returns nil on success.
//
// Returns the first compilation error encountered while emitting an initialiser.
func (c *compiler) compilePackageLevelVarInit(ctx context.Context, spec *ast.ValueSpec) error {
	for i, name := range spec.Names {
		if name.Name == blankIdentName {
			continue
		}
		gv, ok := c.globalVariables[name.Name]
		if !ok {
			continue
		}
		if err := c.compilePackageLevelVar(ctx, spec, i, name, gv); err != nil {
			return err
		}
	}
	return nil
}

// compilePackageLevelVar emits the initialiser for a single package-level variable within
// spec.
//
// Takes spec: the value specification holding the variable's source declaration.
// Takes i: the index of the variable within spec.Names.
// Takes name: the identifier being initialised.
// Takes gv: the global slot information for the variable.
//
// Returns nil on success.
//
// Returns the compilation error from the initialiser expression.
func (c *compiler) compilePackageLevelVar(ctx context.Context, spec *ast.ValueSpec, i int, name *ast.Ident, gv globalVariableInfo) error {
	if i < len(spec.Values) {
		valueLocation, err := c.compileExpression(ctx, spec.Values[i])
		if err != nil {
			return err
		}
		c.emitSetGlobal(ctx, gv, valueLocation)
		return nil
	}

	if gv.kind == registerGeneral {
		c.emitGlobalZeroGeneral(ctx, name, gv)
		return nil
	}

	register := c.scopes.alloc.allocTemp(gv.kind)
	c.function.emit(opDrillTier1, uint8(subOpLoadZero), register, uint8(gv.kind))
	c.emitSetGlobalOp(ctx, register, gv)
	c.scopes.alloc.freeTemp(gv.kind, register)
	return nil
}

// emitGlobalZeroGeneral emits a zero-value initialiser for a registerGeneral
// package-level variable.
//
// Prefers a named-type zero from the symbol registry before falling back to a composite
// (array/struct) zero reflect.Value.
//
// Takes name: the identifier being initialised.
// Takes gv: the global slot information for the variable.
func (c *compiler) emitGlobalZeroGeneral(ctx context.Context, name *ast.Ident, gv globalVariableInfo) {
	typeObject := c.info.Defs[name]
	if typeObject == nil {
		return
	}
	if c.symbols != nil {
		if zeroValue, ok := c.zeroValueForNamedType(ctx, typeObject.Type()); ok {
			if named, isNamed := typeObject.Type().(*types.Named); isNamed {
				c.emitGlobalGeneralConst(ctx, gv, zeroValue, generalConstantDescriptor{
					kind:        generalConstantNamedTypeZero,
					packagePath: named.Obj().Pkg().Path(),
					symbolName:  named.Obj().Name(),
				})
				return
			}
		}
	}
	if zeroValue, ok := c.zeroValueForCompositeType(ctx, typeObject.Type()); ok {
		c.emitGlobalGeneralConst(ctx, gv, zeroValue, generalConstantDescriptor{
			kind:     generalConstantCompositeZero,
			typeDesc: reflectTypeToDescriptor(c.typeToReflect(ctx, typeObject.Type())),
		})
	}
}

// emitGlobalGeneralConst adds value to the function's general constant pool and stores it
// into the supplied global slot.
//
// Takes gv: the global slot information for the destination variable.
// Takes value: the reflect.Value to register as a constant.
// Takes descriptor: the descriptor classifying the constant for the runtime.
func (c *compiler) emitGlobalGeneralConst(ctx context.Context, gv globalVariableInfo, value reflect.Value, descriptor generalConstantDescriptor) {
	register := c.scopes.alloc.allocTemp(registerGeneral)
	constIndex, err := c.function.addGeneralConstant(value, descriptor)
	if err != nil {
		c.recordStickyError(err)
		c.scopes.alloc.freeTemp(registerGeneral, register)
		return
	}
	c.function.emitWide(opLoadGeneralConst, register, constIndex)
	c.emitSetGlobalOp(ctx, register, gv)
	c.scopes.alloc.freeTemp(registerGeneral, register)
}

// emitGetGlobal loads the global identified by gv into a freshly allocated register.
//
// Chooses the wide opcode form when the global index exceeds a uint8.
//
// Takes gv: the global slot information identifying the source global.
//
// Returns the varLocation holding the loaded value.
func (c *compiler) emitGetGlobal(_ context.Context, gv globalVariableInfo) varLocation {
	dest := c.scopes.alloc.alloc(gv.kind)
	if gv.index <= math.MaxUint8 {
		c.function.emit(opGetGlobal, dest, safeconv.MustIntToUint8(gv.index), uint8(gv.kind))
	} else {
		c.function.emit(opGetGlobalWide, dest, 0, uint8(gv.kind))
		c.function.emitExtension(safeconv.MustIntToUint16(gv.index), 0)
	}
	return varLocation{register: dest, kind: gv.kind}
}

// emitSetGlobal stores source into the global identified by gv.
//
// Inserts a bank coercion through a temporary register when source.kind differs from
// gv.kind.
//
// Takes gv: the global slot information identifying the destination global.
// Takes source: the source location holding the value to store.
func (c *compiler) emitSetGlobal(ctx context.Context, gv globalVariableInfo, source varLocation) {
	if source.kind != gv.kind {
		temp := c.scopes.alloc.allocTemp(gv.kind)
		c.emitMove(ctx, varLocation{register: temp, kind: gv.kind}, source)
		c.emitSetGlobalOp(ctx, temp, gv)
		c.scopes.alloc.freeTemp(gv.kind, temp)
		return
	}
	c.emitSetGlobalOp(ctx, source.register, gv)
}

// emitSetGlobalOp emits the narrow or wide opSetGlobal instruction.
//
// Selects the wide form when the global index does not fit in a uint8.
//
// Takes sourceRegister: the register supplying the value to store.
// Takes gv: the global slot information identifying the destination global.
func (c *compiler) emitSetGlobalOp(_ context.Context, sourceRegister uint8, gv globalVariableInfo) {
	if gv.index <= math.MaxUint8 {
		c.function.emit(opSetGlobal, sourceRegister, safeconv.MustIntToUint8(gv.index), uint8(gv.kind))
	} else {
		c.function.emit(opSetGlobalWide, sourceRegister, 0, uint8(gv.kind))
		c.function.emitExtension(safeconv.MustIntToUint16(gv.index), 0)
	}
}

// compileGenDecl compiles a general declaration (var, const, type).
//
// Type specs do not emit bytecode.
//
// Takes declaration: the general declaration node to compile.
//
// Returns the location of the last value spec emitted, or the zero varLocation when no
// value specs were emitted.
//
// Returns the first compilation error encountered while emitting a value spec.
func (c *compiler) compileGenDecl(ctx context.Context, declaration *ast.GenDecl) (varLocation, error) {
	var lastLocation varLocation
	for _, spec := range declaration.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			location, err := c.compileValueSpec(ctx, s)
			if err != nil {
				return varLocation{}, err
			}
			lastLocation = location
		case *ast.TypeSpec:

			_ = s
		}
	}
	return lastLocation, nil
}

// compileValueSpec compiles a var or const declaration.
//
// Declares each name in the current scope and emits either its initialiser or the
// zero-value for its type.
//
// Takes spec: the value specification to compile.
//
// Returns the location of the first declared variable, or the zero varLocation when no
// variables are declared.
//
// Returns the first compilation error encountered while emitting an initialiser.
func (c *compiler) compileValueSpec(ctx context.Context, spec *ast.ValueSpec) (varLocation, error) {
	for i, name := range spec.Names {
		if err := c.compileValueSpecName(ctx, spec, i, name); err != nil {
			return varLocation{}, err
		}
	}

	if len(spec.Names) > 0 {
		location, ok := c.scopes.lookupVar(spec.Names[0].Name)
		if ok {
			return location, nil
		}
	}

	return varLocation{}, nil
}

// compileValueSpecName declares and initialises a single name from a var or const
// declaration, emitting either the initialiser at index or the zero value for the name's
// type.
//
// Takes spec (*ast.ValueSpec) which is the value specification.
// Takes index (int) which is the position of name within spec.Names.
// Takes name (*ast.Ident) which is the identifier being declared.
//
// Returns the first compilation error encountered while emitting the initialiser, or nil
// when the name is blank or has no type info.
func (c *compiler) compileValueSpecName(ctx context.Context, spec *ast.ValueSpec, index int, name *ast.Ident) error {
	if name.Name == blankIdentName {
		return nil
	}

	typeObject := c.info.Defs[name]
	if typeObject == nil {
		return nil
	}

	kind := c.kindFor(typeObject.Type())
	location := c.scopes.declareVar(name.Name, kind)

	if index < len(spec.Values) {
		if err := c.emitValueSpecInitialiser(ctx, spec.Values[index], typeObject.Type(), location); err != nil {
			return err
		}
	} else {
		c.emitLocalZeroValue(ctx, typeObject, location)
	}
	c.tryHeapPromoteCapturedLocal(ctx, name.Name, name)
	return nil
}

// emitValueSpecInitialiser compiles an initialiser expression and moves its value into
// the declared variable's location.
//
// Takes value (ast.Expr) which is the initialiser expression.
// Takes declaredType (types.Type) which is the declared variable type.
// Takes location (varLocation) which is the destination location.
//
// Returns the first compilation error encountered.
func (c *compiler) emitValueSpecInitialiser(ctx context.Context, value ast.Expr, declaredType types.Type, location varLocation) error {
	valueLocation, handled, herr := c.compileTypedNilOrExpression(ctx, value, declaredType)
	if herr != nil {
		return herr
	}
	if !handled {
		var err error
		valueLocation, err = c.compileExpression(ctx, value)
		if err != nil {
			return err
		}
	}
	valueLocation = c.coerceEvalBoolResult(ctx, c.info, value, valueLocation)
	c.emitMoveTyped(ctx, location, valueLocation, c.staticTypeOf(value))
	return nil
}

// emitLocalZeroValue initialises a local variable to its zero value.
//
// Typed-bank locals use opLoadZero directly; general-bank locals attempt a named-type
// zero from the symbol registry, then a composite (array/struct) zero, then a generic
// reflect.Zero before falling back to opLoadZero.
//
// Takes typeObject: the type object describing the local being initialised.
// Takes location: the destination register location.
func (c *compiler) emitLocalZeroValue(ctx context.Context, typeObject types.Object, location varLocation) {
	if location.kind != registerGeneral {
		c.function.emit(opDrillTier1, uint8(subOpLoadZero), location.register, uint8(location.kind))
		return
	}
	if c.tryEmitNamedTypeZero(ctx, typeObject, location) {
		return
	}
	if c.tryEmitCompositeTypeZero(ctx, typeObject, location) {
		return
	}
	if c.tryEmitReflectZero(ctx, typeObject, location) {
		return
	}
	c.function.emit(opDrillTier1, uint8(subOpLoadZero), location.register, uint8(location.kind))
}

// tryEmitNamedTypeZero loads a zero value for a named type.
//
// Consults the symbol registry for the registered zero value.
//
// Takes ctx (context.Context) which threads cancellation.
// Takes typeObject (types.Object) which names the type.
// Takes location (varLocation) which is the destination register.
//
// Returns bool which is true when the load was emitted.
func (c *compiler) tryEmitNamedTypeZero(ctx context.Context, typeObject types.Object, location varLocation) bool {
	if c.symbols == nil {
		return false
	}
	zeroValue, ok := c.zeroValueForNamedType(ctx, typeObject.Type())
	if !ok {
		return false
	}
	named, isNamed := typeObject.Type().(*types.Named)
	if !isNamed {
		return false
	}
	constIndex, err := c.function.addGeneralConstant(zeroValue, generalConstantDescriptor{
		kind:        generalConstantNamedTypeZero,
		packagePath: named.Obj().Pkg().Path(),
		symbolName:  named.Obj().Name(),
	})
	if err != nil {
		c.recordStickyError(err)
		return true
	}
	c.function.emitWide(opLoadGeneralConst, location.register, constIndex)
	return true
}

// tryEmitCompositeTypeZero emits a composite zero constant load.
//
// Acts only when typeObject's underlying type is array or struct.
//
// Takes ctx (context.Context) which threads cancellation.
// Takes typeObject (types.Object) which names the type.
// Takes location (varLocation) which is the destination register.
//
// Returns bool which is true when the load was emitted.
func (c *compiler) tryEmitCompositeTypeZero(ctx context.Context, typeObject types.Object, location varLocation) bool {
	zeroValue, ok := c.zeroValueForCompositeType(ctx, typeObject.Type())
	if !ok {
		return false
	}
	constIndex, err := c.function.addGeneralConstant(zeroValue, generalConstantDescriptor{
		kind:     generalConstantCompositeZero,
		typeDesc: reflectTypeToDescriptor(c.typeToReflect(ctx, typeObject.Type())),
	})
	if err != nil {
		c.recordStickyError(err)
		return true
	}
	c.function.emitWide(opLoadGeneralConst, location.register, constIndex)
	return true
}

// tryEmitReflectZero emits a reflect.Zero general constant load.
//
// Acts only for non-interface types whose reflect.Type is resolvable.
//
// Takes ctx (context.Context) which threads cancellation.
// Takes typeObject (types.Object) which names the type.
// Takes location (varLocation) which is the destination register.
//
// Returns bool which is true when the load was emitted.
func (c *compiler) tryEmitReflectZero(ctx context.Context, typeObject types.Object, location varLocation) bool {
	if _, isInterface := typeObject.Type().Underlying().(*types.Interface); isInterface {
		return false
	}
	reflectType := c.typeToReflect(ctx, typeObject.Type())
	if reflectType == nil {
		return false
	}
	constIndex, err := c.function.addGeneralConstant(reflect.Zero(reflectType), generalConstantDescriptor{
		kind:     generalConstantCompositeZero,
		typeDesc: reflectTypeToDescriptor(reflectType),
	})
	if err != nil {
		c.recordStickyError(err)
		return true
	}
	c.function.emitWide(opLoadGeneralConst, location.register, constIndex)
	return true
}

// zeroValueForCompositeType returns an addressable zero reflect.Value for an array or
// struct type.
//
// Takes t: the type whose zero value is required.
//
// Returns the addressable zero reflect.Value and true when t's underlying type is an
// array or struct, and (reflect.Value{}, false) otherwise.
func (c *compiler) zeroValueForCompositeType(ctx context.Context, t types.Type) (reflect.Value, bool) {
	reflectType := c.typeToReflect(ctx, t)
	if reflectType == nil {
		return reflect.Value{}, false
	}
	switch t.Underlying().(type) {
	case *types.Array, *types.Struct:
		return reflect.New(reflectType).Elem(), true
	}
	return reflect.Value{}, false
}

// zeroValueForNamedType returns an addressable zero reflect.Value for a registered named
// type.
//
// Takes t: the type whose zero value is required.
//
// Returns the addressable zero reflect.Value and true when t is a named type registered
// in the symbol registry, and (reflect.Value{}, false) otherwise.
func (c *compiler) zeroValueForNamedType(_ context.Context, t types.Type) (reflect.Value, bool) {
	named, ok := t.(*types.Named)
	if !ok {
		return reflect.Value{}, false
	}
	typeObject := named.Obj()
	if typeObject.Pkg() == nil {
		return reflect.Value{}, false
	}
	return c.symbols.ZeroValueForType(typeObject.Pkg().Path(), typeObject.Name())
}

// compileAssign compiles an assignment statement, dispatching to the appropriate
// specialised path.
//
// Recognised paths: short-variable, compound, multi-assignment, index-RMW rewrite, or
// generic single-pair.
//
// Takes statement: the assignment statement to compile.
//
// Returns the location of the final assignment target.
//
// Returns the first compilation error encountered.
func (c *compiler) compileAssign(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	if statement.Tok == token.DEFINE {
		return c.compileShortVarDecl(ctx, statement)
	}
	if isCompoundAssign(statement.Tok) {
		return c.compileCompoundAssign(ctx, statement)
	}
	if len(statement.Lhs) > 1 {
		return c.compileMultiAssign(ctx, statement)
	}
	if rewritten, ok := c.tryRewriteIndexRMWAssign(statement); ok {
		return c.compileCompoundAssign(ctx, rewritten)
	}
	return c.compileSingleAssignPairs(ctx, statement)
}

// compileSingleAssignPairs walks the LHS/RHS pairs of a non-compound, non-define
// assignment and emits each store.
//
// Each pair is first offered to the structural fast paths (struct-into-collection,
// star-append-byte) before falling back to the generic expression-then-store sequence.
//
// Takes statement: the assignment statement whose pairs are being emitted.
//
// Returns the location of the final emitted assignment, or the zero varLocation when no
// pairs were processed.
//
// Returns the first compilation error encountered.
func (c *compiler) compileSingleAssignPairs(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	var lastLocation varLocation
	for i, leftHandSide := range statement.Lhs {
		location, applied, err := c.compileAssignFastPath(ctx, leftHandSide, statement.Rhs[i])
		if err != nil {
			return varLocation{}, err
		}
		if applied {
			lastLocation = location
			continue
		}
		lastLocation, err = c.compileAssignPair(ctx, leftHandSide, statement.Rhs[i])
		if err != nil {
			return varLocation{}, err
		}
	}
	return lastLocation, nil
}

// compileAssignFastPath offers a single assignment pair to the structural fast paths in
// priority order.
//
// Takes leftHandSide: the LHS expression of the pair.
// Takes rightHandSide: the RHS expression of the pair.
//
// Returns the result location produced by a matching fast path, and applied=true when any
// fast path emitted code; false otherwise.
//
// Returns any emission error from the selected fast path.
func (c *compiler) compileAssignFastPath(ctx context.Context, leftHandSide, rightHandSide ast.Expr) (varLocation, bool, error) {
	if location, applied, err := c.tryCompileStructIntoCollection(ctx, leftHandSide, rightHandSide); applied {
		return location, true, err
	}
	if location, applied, err := c.tryCompileStarAppendByteFast(ctx, leftHandSide, rightHandSide); applied {
		return location, true, err
	}
	if location, applied, err := c.tryCompileInPlaceAppend(ctx, leftHandSide, rightHandSide); applied {
		return location, true, err
	}
	return varLocation{}, false, nil
}

// compileAssignPair compiles the RHS expression and emits the store into the supplied LHS
// target.
//
// Takes leftHandSide: the LHS expression receiving the value.
// Takes rightHandSide: the RHS expression producing the value.
//
// Returns the location holding the stored value.
//
// Returns the compilation error from the RHS expression or the store.
func (c *compiler) compileAssignPair(ctx context.Context, leftHandSide, rightHandSide ast.Expr) (varLocation, error) {
	var expectedType types.Type
	if c.info != nil {
		if tv, ok := c.info.Types[leftHandSide]; ok {
			expectedType = tv.Type
		}
	}
	valueLocation, handled, herr := c.compileTypedNilOrExpression(ctx, rightHandSide, expectedType)
	if herr != nil {
		return varLocation{}, herr
	}
	if !handled {
		var err error
		valueLocation, err = c.compileExpression(ctx, rightHandSide)
		if err != nil {
			return varLocation{}, err
		}
	}
	return c.emitAssignTarget(ctx, leftHandSide, valueLocation)
}

// emitAssignTarget stores valueLocation into the given LHS target.
//
// Dispatches on the AST node type (identifier, index, selector, or star expression).
//
// Takes leftHandSide: the LHS expression receiving the value.
// Takes valueLocation: the source location holding the value to store.
//
// Returns the resulting location of the store.
//
// Returns an error when the LHS target type is unsupported.
func (c *compiler) emitAssignTarget(ctx context.Context, leftHandSide ast.Expr, valueLocation varLocation) (varLocation, error) {
	switch target := leftHandSide.(type) {
	case *ast.Ident:
		return c.emitIdentAssign(ctx, target, valueLocation)
	case *ast.IndexExpr:
		if err := c.compileIndexAssign(ctx, target, valueLocation); err != nil {
			return varLocation{}, err
		}
		return valueLocation, nil
	case *ast.SelectorExpr:
		if err := c.compileSelectorAssign(ctx, target, valueLocation); err != nil {
			return varLocation{}, err
		}
		return valueLocation, nil
	case *ast.StarExpr:
		if err := c.compileStarAssign(ctx, target, valueLocation); err != nil {
			return varLocation{}, err
		}
		return valueLocation, nil
	default:
		return varLocation{}, fmt.Errorf("unsupported assignment target: %T at %s", leftHandSide, c.positionString(leftHandSide.Pos()))
	}
}

// emitIdentAssign stores valueLocation into an identifier target.
//
// Resolves the identifier against the upvalue map, the global variable table, and the
// lexical scope chain in that order. Blank identifiers are silently dropped.
//
// Takes target: the identifier receiving the value.
// Takes valueLocation: the source location holding the value to store.
//
// Returns the location of the resolved destination, or valueLocation when the target is
// blank.
//
// Returns an error when the identifier resolves to no known binding.
func (c *compiler) emitIdentAssign(ctx context.Context, target *ast.Ident, valueLocation varLocation) (varLocation, error) {
	if target.Name == blankIdentName {
		return valueLocation, nil
	}
	if ref, ok := c.upvalueMap[target.Name]; ok {
		coerced := c.coerceToKind(ctx, valueLocation, ref.kind)
		c.function.emit(opSetUpvalue, coerced.register, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
		if coerced.register != valueLocation.register || coerced.kind != valueLocation.kind {
			c.scopes.alloc.freeTemp(coerced.kind, coerced.register)
		}
		return valueLocation, nil
	}
	if gv, ok := c.globalVariables[target.Name]; ok {
		c.emitSetGlobal(ctx, gv, valueLocation)
		return valueLocation, nil
	}
	destLocation, found := c.scopes.lookupVar(target.Name)
	if !found {
		return varLocation{}, fmt.Errorf("undefined variable: %s at %s", target.Name, c.positionString(target.Pos()))
	}
	var destType types.Type
	if c.info != nil {
		if obj := c.info.ObjectOf(target); obj != nil {
			destType = obj.Type()
		}
	}
	c.emitMoveTyped(ctx, destLocation, valueLocation, destType)
	c.emitSyncCaptured(ctx, destLocation)
	return destLocation, nil
}

// positionString formats pos as a "file:line:col" string for error messages.
//
// Takes pos: the source position to format.
//
// Returns the formatted position, or "<unknown>" when pos is invalid or the file set is
// nil.
func (c *compiler) positionString(pos token.Pos) string {
	if !pos.IsValid() || c.fileSet == nil {
		return "<unknown>"
	}
	position := c.fileSet.Position(pos)
	return position.String()
}

// setDebugPosition records pos as the current source position for debug source mapping.
//
// No-op when debug info is disabled.
//
// Takes pos: the source position to record.
func (c *compiler) setDebugPosition(_ context.Context, pos token.Pos) {
	if c.debugEnabled {
		c.currentPosition = pos
	}
}

// initDebugInfo wires up the debug source map, file ID table, and emit hook on the
// current function.
//
// sharedFiles is reused across sub-compilers when non-nil; passing nil creates a new file
// table. No-op when debug info is disabled.
//
// Takes sharedFiles: pointer to a shared files slice reused across sub-compilers, or nil
// to allocate a fresh table.
func (c *compiler) initDebugInfo(ctx context.Context, sharedFiles *[]string) {
	if !c.debugEnabled {
		return
	}

	files := sharedFiles
	if files == nil {
		files = new(make([]string, 0, initialFileTableCapacity))
	}

	sm := &sourceMap{files: files}
	c.debugSourceMap = sm
	c.debugFileIDs = make(map[string]uint16)
	c.function.debugSourceMap = sm
	c.function.debugVarTable = &debugVarTable{}

	c.scopes.debugVarTable = c.function.debugVarTable
	c.scopes.debugBodyLenFunc = func() int { return len(c.function.body) }

	c.function.debugEmitHook = func(pc int) {
		if !c.currentPosition.IsValid() {
			for len(sm.positions) <= pc {
				sm.positions = append(sm.positions, sourcePosition{})
			}
			return
		}
		pos := c.fileSet.Position(c.currentPosition)
		fileID := c.resolveFileID(ctx, pos.Filename)

		for len(sm.positions) <= pc {
			sm.positions = append(sm.positions, sourcePosition{})
		}
		sm.positions[pc] = sourcePosition{
			line:   safeconv.IntToInt32(pos.Line),
			column: safeconv.IntToInt16(pos.Column),
			fileID: fileID,
		}
	}
}

// resolveFileID returns the uint16 file ID for filename.
//
// Appends the filename to the source map's shared files slice when not already present.
//
// Takes filename: the source file name to resolve.
//
// Returns the uint16 file ID assigned to the filename.
func (c *compiler) resolveFileID(_ context.Context, filename string) uint16 {
	if id, ok := c.debugFileIDs[filename]; ok {
		return id
	}
	id := safeconv.IntToUint16(len(*c.debugSourceMap.files))
	*c.debugSourceMap.files = append(*c.debugSourceMap.files, filename)
	c.debugFileIDs[filename] = id
	return id
}

// propagateDebugToSubCompiler enables debug info on sub and seeds it with the parent's
// shared files slice.
//
// No-op when debug info is disabled.
//
// Takes sub: the sub-compiler whose debug state is being initialised.
func (c *compiler) propagateDebugToSubCompiler(ctx context.Context, sub *compiler) {
	if !c.debugEnabled {
		return
	}
	sub.debugEnabled = true
	sub.initDebugInfo(ctx, c.debugSourceMap.files)
}

// containsTypeParameter reports whether t mentions a type parameter anywhere in its
// structure (slice element, map element, pointer pointee, struct field, etc.). Used by
// parameter classification to keep generic-instantiation-dependent slots on the general
// bank; the typed-bank classification only applies once a specialised body is created
// with concrete types.
//
// Takes t (types.Type) which is the type to inspect.
//
// Returns true when t directly is or transitively contains a TypeParam.
func containsTypeParameter(t types.Type) bool {
	if t == nil {
		return false
	}
	if isTypeParameter(t) {
		return true
	}
	switch u := t.Underlying().(type) {
	case *types.Slice:
		return containsTypeParameter(u.Elem())
	case *types.Array:
		return containsTypeParameter(u.Elem())
	case *types.Map:
		return containsTypeParameter(u.Key()) || containsTypeParameter(u.Elem())
	case *types.Chan:
		return containsTypeParameter(u.Elem())
	case *types.Pointer:
		return containsTypeParameter(u.Elem())
	}
	return false
}

// collectClosureHeapPromotedParamNames mirrors collectHeapPromotedParamNames for a
// function literal's parameters. Returns the set of parameter names the literal's body
// captures or address-takes, so the closure parameter slot can be demoted to general
// bank.
//
// Takes typeContext (*compiler).
// Takes lit (*ast.FuncLit).
//
// Returns the heap-promoted parameter name set.
//
//nolint:dupl // per-element-kind specialisation
func collectClosureHeapPromotedParamNames(typeContext *compiler, lit *ast.FuncLit) map[string]bool {
	if lit == nil || lit.Body == nil {
		return map[string]bool{}
	}
	promoted := collectHeapPromotedNames(typeContext, lit.Body)
	if promoted == nil {
		return map[string]bool{}
	}
	parameterNames := map[string]bool{}
	if lit.Type != nil && lit.Type.Params != nil {
		for _, field := range lit.Type.Params.List {
			for _, name := range field.Names {
				parameterNames[name.Name] = true
			}
		}
	}
	result := map[string]bool{}
	for name := range promoted {
		if parameterNames[name] {
			result[name] = true
		}
	}
	return result
}

// classifyTypedSliceClosureParameters runs the typed-slice-survivor body-walk against a
// function literal's parameter list, mirroring classifyTypedSliceParameters for declared
// functions.
//
// Takes typeContext (*compiler) which provides go/types info.
// Takes lit (*ast.FuncLit) which is the function literal.
// Takes signature (*types.Signature) which carries parameter types.
//
// Returns map[string]registerKind which is the surviving parameter name to typed-slice
// kind map.
//
//nolint:dupl // per-element-kind specialisation
func classifyTypedSliceClosureParameters(typeContext *compiler, lit *ast.FuncLit, signature *types.Signature) map[string]registerKind {
	if lit == nil || lit.Body == nil || signature == nil {
		return nil
	}
	parameterCount := signature.Params().Len()
	candidates := map[string]registerKind{}
	parameterIndex := 0
	for p := range signature.Params().Variables() {
		isVariadicLast := signature.Variadic() && parameterIndex == parameterCount-1
		parameterIndex++
		if isVariadicLast {
			continue
		}
		if isTypeParameter(p.Type()) || containsTypeParameter(p.Type()) {
			continue
		}
		kind := kindForCallSlot(typeContext.substitutedType(p.Type()))
		if !isTypedSliceKind(kind) {
			continue
		}
		if p.Name() == "" || p.Name() == "_" {
			continue
		}
		candidates[p.Name()] = kind
	}
	if len(candidates) == 0 {
		return nil
	}
	return classifyTypedSliceParamNames(typeContext, lit.Body, candidates)
}

// classifyTypedSliceParameters picks typed-slice-bank parameters.
//
// Mirrors the local-side classifyTypedSliceLocals analysis but seeds the candidate set
// from the parameter list (instead of make-allocation sites). Parameters whose static
// type maps to a typed-slice bank via kindForCallSlot AND whose body usage doesn't
// trigger any typed-bank disqualifier
// (append/copy/address-of/closure-capture/type-assertion/etc.) survive. Survivors stay on
// the typed bank; the rest fall back to general bank for ABI safety.
//
// Takes typeContext (*compiler) which provides go/types info and access to funcTable for
// the call-site permissive rule.
// Takes declaration (*ast.FuncDecl) which is the declaring AST.
// Takes signature (*types.Signature) which carries parameter types.
//
// Returns map[string]registerKind which is the surviving parameter name to typed-slice
// kind map, or nil when declaration has no body, no parameters, or no
// typed-slice-eligible parameters.
//
//nolint:dupl // per-element-kind specialisation
func classifyTypedSliceParameters(typeContext *compiler, declaration *ast.FuncDecl, signature *types.Signature) map[string]registerKind {
	if declaration == nil || declaration.Body == nil || signature == nil {
		return nil
	}
	parameterCount := signature.Params().Len()
	candidates := map[string]registerKind{}
	parameterIndex := 0
	for p := range signature.Params().Variables() {
		isVariadicLast := signature.Variadic() && parameterIndex == parameterCount-1
		parameterIndex++
		if isVariadicLast {
			continue
		}
		if isTypeParameter(p.Type()) || containsTypeParameter(p.Type()) {
			continue
		}
		kind := kindForCallSlot(typeContext.substitutedType(p.Type()))
		if !isTypedSliceKind(kind) {
			continue
		}
		if p.Name() == "" || p.Name() == "_" {
			continue
		}
		candidates[p.Name()] = kind
	}
	if len(candidates) == 0 {
		return nil
	}
	return classifyTypedSliceParamNames(typeContext, declaration.Body, candidates)
}

// collectHeapPromotedParamNames returns heap-promoted parameter names.
//
// Surfaces parameters in declaration that would be heap-promoted because the body
// captures them in a closure or takes their address. Used by registerFuncDecl to demote
// such parameters to the general bank before parameterKinds is published - the indirect
// storage cell type uses the user-declared slice type and cannot accept the typed-bank
// storage equivalent, so heap-promoted parameters stay on general for ABI consistency.
// Returns an empty (non-nil) map when declaration has no body.
//
// Takes typeContext (*compiler) which provides the kind-filter helpers for
// collectClosureCapturedNamesFiltered.
// Takes declaration (*ast.FuncDecl).
//
// Returns the heap-promoted parameter name set.
//
//nolint:dupl // per-element-kind specialisation
func collectHeapPromotedParamNames(typeContext *compiler, declaration *ast.FuncDecl) map[string]bool {
	if declaration == nil || declaration.Body == nil {
		return map[string]bool{}
	}
	promoted := collectHeapPromotedNames(typeContext, declaration.Body)
	if promoted == nil {
		return map[string]bool{}
	}
	parameterNames := map[string]bool{}
	if declaration.Type != nil && declaration.Type.Params != nil {
		for _, field := range declaration.Type.Params.List {
			for _, name := range field.Names {
				parameterNames[name.Name] = true
			}
		}
	}
	result := map[string]bool{}
	for name := range promoted {
		if parameterNames[name] {
			result[name] = true
		}
	}
	return result
}

// compileEvalExpression compiles a single expression as the body of a synthetic "<eval>"
// function.
//
// The expression result is moved into register 0 of its bank and the function's
// resultKinds records that bank for the runtime to extract.
//
// Takes fileSet: the file set used for position reporting.
// Takes info: the type information for the expression.
// Takes expression: the expression to compile.
// Takes symbols: the registry of pre-registered native symbols.
// Takes features: the compiler features permitted during emission.
// Takes maxLiteralElements: the upper bound on elements in any composite literal, or 0
// for unlimited.
//
// Returns the compiled synthetic function on success.
//
// Returns the first compilation error encountered.
func compileEvalExpression(
	ctx context.Context,
	fileSet *token.FileSet,
	info *types.Info,
	expression ast.Expr,
	symbols *SymbolRegistry,
	features InterpFeature,
	maxLiteralElements int,
) (*CompiledFunction, error) {
	evalFunction := &CompiledFunction{name: "<eval>"}
	c := &compiler{
		fileSet:            fileSet,
		info:               info,
		function:           evalFunction,
		scopes:             newScopeStack("<eval>"),
		funcTable:          make(map[string]uint16),
		rootFunction:       evalFunction,
		symbols:            symbols,
		features:           features,
		maxLiteralElements: maxLiteralElements,
	}
	c.scopes.pushScope()

	location, err := c.compileExpression(ctx, expression)
	if err != nil {
		return nil, fmt.Errorf("compiling eval expression: %w", err)
	}

	location = c.coerceEvalBoolResult(ctx, info, expression, location)

	c.emitMoveToRegisterZero(ctx, location)

	c.function.resultKinds = []registerKind{location.kind}
	if err := c.resourceError(); err != nil {
		return nil, fmt.Errorf("compiling eval expression: %w", err)
	}
	c.function.numRegisters = c.scopes.peakRegisters()
	if err := c.function.optimise(ctx); err != nil {
		return nil, fmt.Errorf("compiling eval expression: %w", err)
	}
	c.scopes.popScope()

	return c.function, nil
}

// isCompoundAssign reports whether operatorToken is one of Go's compound assignment
// operators.
//
// The recognised set is +=, -=, *=, /=, %=, &=, |=, ^=, &^=, <<=, >>=.
//
// Takes operatorToken: the token to classify.
//
// Returns true when operatorToken is a compound assignment operator; false otherwise.
func isCompoundAssign(operatorToken token.Token) bool {
	switch operatorToken {
	case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN,
		token.QUO_ASSIGN, token.REM_ASSIGN,
		token.AND_ASSIGN, token.OR_ASSIGN, token.XOR_ASSIGN,
		token.AND_NOT_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN:
		return true
	default:
	}
	return false
}

// compoundToOp maps a compound assignment token to its plain binary operator.
//
// For example, ADD_ASSIGN maps to ADD.
//
// Takes operatorToken: the token to map.
//
// Returns the matching binary operator token, or operatorToken unchanged when it is not a
// compound assignment.
func compoundToOp(operatorToken token.Token) token.Token {
	switch operatorToken {
	case token.ADD_ASSIGN:
		return token.ADD
	case token.SUB_ASSIGN:
		return token.SUB
	case token.MUL_ASSIGN:
		return token.MUL
	case token.QUO_ASSIGN:
		return token.QUO
	case token.REM_ASSIGN:
		return token.REM
	case token.AND_ASSIGN:
		return token.AND
	case token.OR_ASSIGN:
		return token.OR
	case token.XOR_ASSIGN:
		return token.XOR
	case token.AND_NOT_ASSIGN:
		return token.AND_NOT
	case token.SHL_ASSIGN:
		return token.SHL
	case token.SHR_ASSIGN:
		return token.SHR
	default:
		return operatorToken
	}
}
