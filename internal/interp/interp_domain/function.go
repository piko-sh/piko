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
	"go/types"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// CompiledFunction holds the bytecode and metadata for a single function.
//
// Field order is grouped by responsibility (method/type tables, code constant pools,
// debug info, optimisation flags) rather than strict alignment-optimal order; reordering
// for the modest pointer-byte saving would scatter the per-section documentation across
// the struct without observable runtime benefit.
type CompiledFunction struct {
	// variableInitFunction holds bytecode for package-level variable initialisers. When
	// non-nil, Execute runs it before the main body to reset globals to their declared
	// values on each invocation.
	variableInitFunction *CompiledFunction

	// globalBases shifts every opGetGlobal{,Wide} / opSetGlobal{,Wide} operand by a per-kind
	// offset at dispatch time so a bytecode-loaded bundle's globals land in the slots
	// reserved for it in the target Service's globalStore.
	//
	// nil for source-compiled functions (operands are already absolute against this
	// Service's globalStore). Non-nil only when set by Service.LoadModule after reserving
	// slots: every CompiledFunction reachable from a loaded CompiledFileSet shares the same
	// pointer.
	//
	// Same pointer is shared across all functions in a bundle so the VM hot path is a single
	// indirection.
	globalBases *SlotAllocation

	// genericTypeParams is the *types.TypeParamList from the generic signature, retained so
	// call-site type-arg resolution can build a substitution map by zipping these against
	// c.info.Instances[ident].TypeArgs. Nil for non-generic functions.
	genericTypeParams *types.TypeParamList

	// asmCallInfoTables caches the pre-computed asmCallInfo tables keyed by compiled
	// function, populated when the owning function is the root of an Execute() call. Built
	// once on first execution, then reused - the tables are deterministic for a given
	// compiled function and its function table.
	asmCallInfoTables map[*CompiledFunction][]asmCallInfo

	// complexConstIndex accelerates constant deduplication during compilation. Nil at
	// runtime after optimise() releases it.
	complexConstIndex map[complex128]uint16

	// floatConstIndex accelerates constant deduplication during compilation. Nil at runtime
	// after optimise() releases it.
	floatConstIndex map[float64]uint16

	// stringConstIndex accelerates constant deduplication during compilation. Nil at runtime
	// after optimise() releases it.
	stringConstIndex map[string]uint16

	// uintConstIndex accelerates constant deduplication during compilation. Nil at runtime
	// after optimise() releases it.
	uintConstIndex map[uint64]uint16

	// typeRefIndex accelerates type table deduplication during compilation. Nil at runtime
	// after optimise() releases it.
	typeRefIndex map[reflect.Type]uint16

	// typeNames maps reflect.Type to the source-level type name for types created via
	// reflect.StructOf (which have an empty Name()). Used by handleCallMethod to resolve
	// method table keys at runtime.
	typeNames map[reflect.Type]string

	// debugSourceMap maps program counter offsets to source file positions. Nil when debug
	// info is disabled.
	debugSourceMap *sourceMap

	// peepholeProvenance records, per rewritten PC, the kind of rewrite the optimisation
	// pipeline applied and where relevant the origin PC from which the new instruction was
	// derived; the disassembler reads this side table to annotate CSE-inserted moves and
	// LICM-hoisted reads, nil until the first rewrite is recorded and populated by
	// recordPeepholeRewrite.
	peepholeProvenance map[int]peepholeAnnotation

	// arenaSafeAllocPCs lists the bytecode PCs of opAllocIndirect instructions whose target
	// local is statically proven to NOT escape the call frame.
	//
	// nil means "no site classified" (default before analysis runs). A non-nil entry {pc ->
	// true} authorises handleAllocIndirect at that PC to bump-allocate the pointee from the
	// arena instead of reflect.New, on the premise that no escape boundary will ever be
	// crossed for that pointer (so no materialiseArenaValue / heap promotion is needed).
	//
	// Soundness gate: this map MUST be populated only by the escape analysis; speculative
	// entries cause use-after-free on arena Reset. Keyed by PC of the opAllocIndirect
	// instruction word itself (not its extension word).
	arenaSafeAllocPCs map[int]bool

	// aliasInfo holds the per-PC pointer-alias environments.
	//
	// Computed by runPointerAliasAnalysis. CSE, LICM, and GVN consult it to decide whether a
	// write through one general-bank register can invalidate cached reads through another.
	// Nil when the analysis has not been run (e.g. debug builds that skip the inliner) or
	// when the function is empty.
	aliasInfo *pointerAliasInfo

	// intConstIndex accelerates constant deduplication during compilation. Nil at runtime
	// after optimise() releases it.
	intConstIndex map[int64]uint16

	// debugVarTable holds variable debug information (names, liveness ranges). Nil when
	// debug info is disabled.
	debugVarTable *debugVarTable

	// genericDeclaration is the AST FuncDecl retained for body re-compilation when a call
	// site triggers specialisation. Nil for non-generic functions and for specialisations
	// themselves (which don't re-specialise further).
	genericDeclaration *ast.FuncDecl

	// methodTable maps "TypeName.MethodName" to a function index in functions. Used for
	// runtime dispatch of interface method calls where the concrete type is unknown at
	// compile time.
	methodTable map[string]uint16

	// getMethodReceiverTypeNames carries the source-level type name the compiler resolved at
	// each opGetMethod site, keyed by the PC of the GET_METHOD instruction.
	//
	// At runtime handleGetMethod can recover the receiver's named type via pikoTypeName for
	// struct receivers (the `_pikoID_<Name>` sentinel) but not for named slice / map / array
	// receivers - those reflect.Types carry no fields and the rootFunction.typeNames map
	// only contains struct entries. For the cross-package case (lib compiled in one
	// CompileProgram batch, main in another), the runtime needs the source-level name to
	// look the method up in globalStore.externalMethods.
	//
	// Populated only when compileSelectorMethodValue takes the subOpGetMethod fallback
	// (cross-package method values where funcTable doesn't have the entry) and the receiver
	// has a resolvable static type name. Nil for functions without any such sites - the
	// common case.
	getMethodReceiverTypeNames map[uint32]string

	// specialisationOrigin points back at the generic CompiledFunction that produced this
	// specialised body.
	//
	// Nil for non-specialised functions. Used for disassembler naming and to walk back to
	// the generic table for cap-tracking.
	specialisationOrigin *CompiledFunction

	// specialisations maps a tuple of concrete type-args to the funcIndex of a specialised
	// body in rootFunction.functions.
	//
	// Lives on the GENERIC callee; populated lazily when a call site first observes a
	// particular instantiation. Nil for non-generic functions.
	specialisations map[specialisationKey]uint16

	// debugEmitHook is called after each instruction is emitted during compilation, used by
	// the compiler to record source positions.
	debugEmitHook func(pc int)

	// sourceFile is the source file path for error reporting.
	sourceFile string

	// name is the function's qualified name (e.g., "main.BuildAST").
	name string

	// intConstants holds int64 constants referenced by opLoadIntConst.
	intConstants []int64

	// upvalueDescriptors describes captured variables for closures. Each entry tells the VM
	// how to initialise an upvalue when creating a closure from the function.
	upvalueDescriptors []UpvalueDescriptor

	// resultKinds maps each return value position to its register kind.
	resultKinds []registerKind

	// namedResultLocations holds the register locations of named return values, if any. Used
	// by bare return statements to copy named result variables to return positions.
	namedResultLocations []varLocation

	// namedResultNames holds the source-level identifier of each named return in the same
	// order as namedResultLocations. Used at emit-return time to re-lookup the current
	// variable state in the scope, so that promotions to indirect storage (via &n) or
	// capture by closures are reflected in the value moved into the return slot.
	namedResultNames []string

	// stringConstants holds string constants referenced by opLoadStringConst.
	stringConstants []string

	// functions holds nested function literals (closures) defined within the enclosing
	// function. Referenced by opMakeClosure via index.
	functions []*CompiledFunction

	// generalConstants holds reflect.Value constants referenced by opLoadGeneralConst. These
	// include type values, function values, and complex constants.
	generalConstants []reflect.Value

	// typeTable holds reflect.Type values referenced by type operation instructions
	// (opTypeAssert, opConvert, opMakeSlice, etc.).
	typeTable []reflect.Type

	// generalConstantDescriptors records how each generalConstants entry was created so it
	// can be reconstructed from a serialised representation.
	generalConstantDescriptors []generalConstantDescriptor

	// typeTableDescriptors records a serialisable description of each typeTable entry for
	// bytecode serialisation.
	typeTableDescriptors []typeDescriptor

	// typeTableInterfaceMethods records required method-name sets per typeTable entry.
	//
	// Nil or empty entries mean the entry is the genuine empty interface (any) and matches
	// every value. Non-empty entries are consulted by handleTypeAssert before calling
	// matchTypeAssertion, because piko collapses every interface type down to
	// reflect.TypeFor[any]() at conversion time (compile-time reflect cannot synthesise
	// interface types with custom method sets, since reflect has no InterfaceOf). Without
	// this side-table a `switch v.(type) { case error: ...; case fmt.Stringer: ...; }` would
	// always match the FIRST arm because srcType.Implements(any) is always true - go-spew's
	// printer relies on this distinction.
	typeTableInterfaceMethods [][]string

	// boolConstants holds bool constants referenced by opLoadBoolConst.
	boolConstants []bool

	// callSites describes each function call in the bytecode. opCall references call sites
	// by index.
	callSites []callSite

	// parameterEscapes records, per pointer-typed parameter, whether the parameter (or
	// *parameter) escapes the owning function's frame.
	//
	// Populated by the per-function escape analysis (compiler_escape_ analysis.go) in
	// bottom-up call-graph order. nil / empty means "everything escapes" (the conservative
	// default before analysis runs, and the fallback for SCC members / native callees whose
	// body cannot be analysed). A non-nil slice has one entry per declared parameter
	// (regardless of kind); non-pointer parameters carry false but are irrelevant since
	// their escape state isn't consulted.
	//
	// Consumed by the caller-side escape analysis to decide whether passing &local to a
	// callee escapes the pointee of &local.
	parameterEscapes []bool

	// complexConstants holds complex128 constants referenced by opLoadComplexConst.
	complexConstants []complex128

	// uintConstants holds uint64 constants referenced by opLoadUintConst.
	uintConstants []uint64

	// body is the bytecode instruction sequence.
	body []instruction

	// floatConstants holds float64 constants referenced by opLoadFloatConst.
	floatConstants []float64

	// parameterTypeRefs records each parameter's declared types.Type as observed at
	// registerFuncDecl time.
	//
	// Used by compileSpecialisedBody to recompute parameterKinds under a substitution map.
	// Nil for non-generic functions where recomputation is unnecessary.
	parameterTypeRefs []types.Type

	// parameterKinds maps each parameter position to its register kind.
	parameterKinds []registerKind

	// parameterRegisters maps each parameter position to the register index the caller must
	// write the argument into.
	//
	// The naive "first N registers in their respective banks" rule is wrong when a parameter
	// is address-taken (`&p` inside the body): the prologue's opAllocIndirect runs against
	// the live register allocator and consumes general-bank slots, pushing the indices of
	// subsequent general-bank parameters past 0. Without an explicit table the caller-side
	// buildCallArgCopyProgram resorts to a per-bank counter starting at 0 and lands the
	// second general parameter on top of the prologue's freshly-allocated heap-promote
	// pointer (general[0]); the body then reads the parameter from the register the compiler
	// actually assigned (general[1]) and observes a zero value.
	//
	// Populated alongside parameterKinds during compileFuncParams. Each entry is the
	// scope.declareVar return-register index captured when the parameter was declared (i.e.
	// before any subsequent heap-promote bumps the allocator). Length equals
	// len(parameterKinds).
	parameterRegisters []uint8

	// parameterIsGeneric reports whether each parameter is the instantiation of a TypeParam.
	//
	// Used by call sites to decide whether to install a generic-monomorphisation cache.
	// Length equals len(parameterKinds) when populated; nil for non-generic callees so the
	// call-site probe is a single nil-check away.
	parameterIsGeneric []bool

	// parameterTypedSlicePromoted records typed-slice survivor verdicts per parameter
	// position.
	//
	// True means the parameter's body-usage analysis allowed it to keep the typed-slice
	// register bank assigned by kindForCallSlot at signature-derivation time
	// (registerSliceInt / Float / Uint / String / Bool / Byte). False means it was demoted
	// to registerGeneral by the survivor walk in populateFuncDeclParameterKinds /
	// applySpecialisationTypedSliceSurvivors.
	//
	// Distinct from parameterEscapes above - that vector tracks pointer-escape state for
	// heap promotion. parameterTypedSlicePromoted is the cross-function-visible view of the
	// typed-slice survivor verdict, consumed by kindForPromotedSlot at call sites with the
	// callee in hand.
	//
	// Length matches parameterKinds when populated; nil when no parameter was a typed-slice
	// candidate (the survivor walk did not run).
	parameterTypedSlicePromoted []bool

	// resultTypeRefs is the result-side analogue of parameterTypeRefs.
	resultTypeRefs []types.Type

	// structLayoutTable holds resolved struct field layouts for fast-path field accessors.
	//
	// Entries are referenced by the subOpGet/SetStructFieldXxx sub-ops and encode the byte
	// offset of the leaf field within its parent struct plus the field kind, so runtime
	// handlers can do direct unsafe-pointer typed loads / stores without entering reflect.
	// Built lazily by the compiler at each field-access emit site; indexed by uint16 from
	// the extension word following the primary instruction. Empty when no eligible field
	// accesses appear in the function body.
	structLayoutTable []structFieldLayout

	// precomputedAllocCounts is the typedSlabCounts derived from numRegisters, cached on the
	// function so AllocRegistersInto doesn't have to convert per call. Populated lazily on
	// first use via ensurePrecomputedAllocCounts.
	precomputedAllocCounts typedSlabCounts

	// asmCallInfoBase holds the function's asmCallInfo table base pointer.
	//
	// Populated by buildASMCallInfoTables and read by updateASMCallInfoBase on every Go-side
	// frame change so the dispatcher bypasses a map probe. The write is race-free because
	// buildASMCallInfoTables only runs inside the root function's asmCallInfoTablesOnce
	// sync.Once; subsequent readers observe a stable value.
	asmCallInfoBase uintptr

	// maxMethods caps the number of entries that may be registered in the methodTable for
	// the function. Zero means use the package default (defaultMaxMethods).
	maxMethods int

	// maxSpecialisations caps the number of generic instantiations the function may
	// accumulate. Zero means use the package default (defaultMaxSpecialisations).
	maxSpecialisations int

	// maxConstantPoolSize caps the entry count of each constant pool (int, float, string,
	// bool, uint, complex, general) and the type table for the function. Zero means use the
	// package default (defaultMaxConstantPoolSize).
	maxConstantPoolSize int

	// numRegisters tracks peak register usage per bank [int, float, string, general, bool,
	// uint, complex]. Used to allocate the register file for each call frame.
	numRegisters [NumRegisterKinds]uint32

	// tinyLeafLayout is the structFieldLayout for the leaf's primary field read.
	//
	// Used by runTinyLeafInline together with the receiver from the caller's general bank to
	// do an unsafe direct read of the field, skipping handler dispatch + frame setup. Valid
	// only when tinyLeafShape != tinyLeafNone.
	tinyLeafLayout structFieldLayout

	// asmCallInfoTablesOnce guards one-time computation of asmCallInfoTables.
	asmCallInfoTablesOnce sync.Once

	// nonZeroBankMask is a bitmask of which register banks have non-zero counts.
	//
	// Bit i is set when numRegisters[i] > 0. Used by AllocRegistersInto to skip empty-bank
	// slice-header writes on closures/methods that only touch a subset of banks. Lazily
	// populated alongside precomputedAllocCounts.
	nonZeroBankMask uint16

	// nonEmptyConstantMask is a bitmask of non-empty per-function pools.
	//
	// Bit positions map to the constMask* constants (intConstants=0, floatConstants=1,
	// stringConstants=2, boolConstants=3, structLayoutTable=4, typeTable=5,
	// complexConstants=6, uintConstants=7). Used by rebuildDispatchPointers to skip the
	// slice-header load + compare for empty pools, turning 8 conditionals into 8
	// single-bit-test branches that the predictor handles uniformly. Populated alongside
	// nonZeroBankMask in ensurePrecomputedAllocCounts.
	nonEmptyConstantMask uint16

	// inRecursionCycle is true when the function participates in a non-trivial SCC of the
	// call graph.
	//
	// Covers direct or indirect recursion. Set once per pass by the bytecode inliner's SCC
	// analysis. Recursive callees cannot be inlined safely (would expand infinitely);
	// canInline reads this to refuse those specific sites without aborting the entire pass
	// for unrelated functions.
	inRecursionCycle bool

	// isGenericFunc is true when the function declared TypeParams.
	//
	// Set by registerFuncDecl when sig.TypeParams() is non-empty. Used by
	// maybeSpecialiseCallee to decide whether a call site should look up or compile a
	// specialisation.
	isGenericFunc bool

	// isPointerReceiver is true when the method's declared receiver type is *T rather than
	// T.
	//
	// The compiler reads this at every method-call site to decide whether the receiver
	// expression needs a compile-time address-take: when the callee expects *T and the
	// caller hands a value-typed T, the call site emits compileAddressOf so the callee gets
	// a pointer, matching Go's automatic `(&c).Inc()` rewrite. False for non-method
	// functions and for value-receiver methods.
	isPointerReceiver bool

	// isVariadic is true when the last parameter is variadic (...T).
	isVariadic bool

	// simpleDeferArgCount records how many arguments the trivial defer takes.
	//
	// Used by EnsureCapacity to pre-size each callee frame's simpleDeferArgs buffer. Zero
	// when simpleDeferOnly is false.
	simpleDeferArgCount uint8

	// emittedInlineBlocker records the first inline-blocking opcode observed by emit()
	// during compilation.
	//
	// Set to inlineRefusalUnknown when no blocker was emitted. Lets scanCalleeForRefusal
	// answer eligibility in O(1) without re-walking the bytecode body. Kept in sync by the
	// emit() chokepoint via blockerForOpcode; optimise()'s peephole fusions never introduce
	// blocker opcodes so this stays valid past optimise().
	emittedInlineBlocker inlineRefusal

	// jumpRangeExceeded is set by encodeJumpOffset when a relative jump distance overflows
	// the signed 16-bit encoding (the body grew past the addressable range). optimise
	// rejects such a function with errCompileJumpRange so the compiler reports a clean error
	// instead of panicking.
	jumpRangeExceeded bool

	// hasReceiver is true when the compiled function is a method.
	//
	// Its declared signature carries a receiver, prepended as parameter 0 of parameterKinds.
	// Set during compileFunctionLike when sig.Recv() != nil. Used by the bytecode inliner to
	// refuse splicing receiver methods until the splice logic for receiver argument
	// materialisation (especially pointer-receiver address-take) is verified end-to-end.
	hasReceiver bool

	// cachedMaxCallDepth memoises estimateMaxCallDepth(root).
	//
	// Saves each goroutine launch and child-VM build from re-walking the call graph,
	// allocating Tarjan SCC state, rebuilding the adjacency list, or populating the
	// depth-memo array. Encoded as sentinel + 1: 0 means not computed, n > 0 means depth = n
	// - 1. Only meaningful on the *root* CompiledFunction passed to estimateMaxCallDepth;
	// child / non-root functions share the parent root's cache via root.functions.
	//
	// atomic.Int32 because concurrent goroutine launches from a shared root race-safely
	// populate the same slot; the value is deterministic so any winning store is correct.
	cachedMaxCallDepth atomic.Int32

	// cachedInlineRefusal records the inliner's verdict on this callee.
	//
	// Most refusals are body-properties (closure, defer, recursion) that don't change after
	// compilation, so the bytecode inliner caches the scan result here to avoid re-walking
	// the body for every call site. Zero value is inlineRefusalUnknown which forces a fresh
	// scan on first probe.
	cachedInlineRefusal inlineRefusal

	// tinyLeafShape classifies the function as a recognised tiny-leaf shape for inline
	// execution.
	//
	// When set, handleCallMethod runs the body in the caller's frame instead of pushing a
	// fresh one. Stays tinyLeafNone when the body does not match any known shape (the
	// default for all functions except polyast-style accessor leaves). Populated by
	// classifyTinyLeaf during optimise().
	tinyLeafShape tinyLeafShape

	// heapMutationClass records the heap-mutation classification for the function and its
	// transitive callees.
	//
	// Captures whether observable mutations to struct fields, slice elements, map entries,
	// channels, or upvalues may occur outside the call frame. The CSE and LICM passes
	// consult this at opCall sites whose callee is statically resolvable: a heapPureCallee
	// call does not invalidate cached struct-field reads. Set by runHeapPurityAnalysis
	// during post-compilation; zero (heapMutationUnknown) before then.
	heapMutationClass heapMutationClassification

	// simpleDeferOnly is true when the function qualifies for the frame-local defer fast
	// path.
	//
	// Set when the body contains exactly one defer, that defer has the trivial shape (no
	// closure capture, args resolvable from existing registers), and no recover() call is
	// reachable in the body. The runtime then uses the frame-local simpleDefer slot instead
	// of appending to vm.deferStack, dropping the per-defer ~155ns of bookkeeping to
	// ~25-30ns. False for functions that have no defer, multiple defers, recover, or a defer
	// in a loop.
	simpleDeferOnly bool

	// tinyLeafEnvArgIdx is the parameter index of the env/slice argument for slice-indexing
	// tiny-leaf shapes.
	//
	// References site.arguments. Used by shapes like varNode.Eval that read env[recv.slot].
	// Zero for shapes that do not need it. Valid only when tinyLeafShape ==
	// tinyLeafReturnEnvUintAtIntFieldSlot.
	tinyLeafEnvArgIdx uint8

	// precomputedAllocCountsValid tracks whether precomputedAllocCounts has been
	// initialised. Lazily set on first AllocRegistersInto call to avoid an extra
	// compile-time pass through every function in the program.
	precomputedAllocCountsValid bool
}

const (
	// defaultMaxConstantPoolSize is the default upper bound on entries in each per-function
	// constant pool (int, float, string, bool, uint, complex, general, type, callSites).
	// uint16 indices saturate at 65535, so this also bounds the encoding.
	defaultMaxConstantPoolSize = 65535

	// defaultMaxSpecialisations is the default upper bound on the number of generic-function
	// specialisations registered per generic callee.
	defaultMaxSpecialisations = 1000

	// defaultMaxMethods is the default upper bound on the size of methodTable on the root
	// CompiledFunction.
	defaultMaxMethods = 10000

	// maxSpecialisationsPerFunction caps how many instantiations a generic function may
	// accumulate.
	//
	// Beyond this, new call sites fall back to the type-erased dispatch path. Bounds memory
	// growth for generics called with many types.
	maxSpecialisationsPerFunction = 8

	// maxSpecialisationTypeArgs caps the number of type-args supported per generic
	// specialisation.
	//
	// Bigger arities fall back to the type-erased path. Sized to match the specialisationKey
	// array length below.
	maxSpecialisationTypeArgs = 4
)

// monomorphicCacheEntry is an immutable atomic snapshot of a call site's resolved
// monomorphic dispatch.
//
// Holds the (receiverType, funcIndex) pair, stored behind callSite.monomorphic as an
// atomic.Pointer so concurrent goroutines reading and updating the cache cannot tear the
// two fields. Once published, an entry is never mutated; cache transitions install a
// fresh entry pointer.
type monomorphicCacheEntry struct {
	// receiverType is the dereferenced concrete receiver type the cache observed. nil when
	// the entry is the disabled sentinel.
	receiverType reflect.Type

	// callee caches the resolved *CompiledFunction so handleCallMethod can skip the
	// vm.functions[funcIndex] indirection on the IC-hit hot path. The (receiverType,
	// funcIndex, callee) triple is atomically published as one entry, so a reader either
	// observes the full triple or a different (older) entry - never a tear between fields.
	callee *CompiledFunction

	// funcIndex is the methodTable-resolved funcIndex for receiverType. Only valid when
	// receiverType is non-nil.
	funcIndex uint16
}

// innerCalleeCacheEntry is an immutable atomic snapshot of an inner-Eval callee
// resolution within an inlineDescriptor.
//
// Holds the (receiverType, callee) pair, published behind a slot pointer in
// inlineDescriptor.innerCalleeSlots via atomic.StorePointer so concurrent readers either
// observe the full pair or a different (older) entry. Once published, an entry is never
// mutated; cache transitions install a fresh entry pointer.
type innerCalleeCacheEntry struct {
	// receiverType is the dereferenced concrete receiver type this entry was resolved for.
	// nil entries are never published.
	receiverType reflect.Type

	// callee is the resolved *CompiledFunction for the inner Eval dispatch on receiverType.
	callee *CompiledFunction
}

// inlineShape selects the runtime inlining path for a call site.
//
// Resolves to one of the shape constants below for a given (callSite, recvType) pair.
// inlineShapeNone is the default "no fused inlining - use standard dispatch" marker.
// Additional shapes (binop, getter chain, etc.) extend this enum.
type inlineShape uint8

const (
	// inlineShapeNone marks a descriptor that did not match any known fused shape. The
	// runtime falls back to pushCompiledFrame or runTinyLeafInline for the cached callee.
	inlineShapeNone inlineShape = iota

	// inlineShapeBinopUint marks a fused uint64 binop-Eval shape.
	//
	// Bodies of the form
	//
	// 	return (recv.left.Eval(env) OP recv.right.Eval(env)) & mask
	//
	// where OP is one of ADD/SUB/MUL/MOD on uint64 and the optional mask is captured in
	// maskValue + maskApplies. The runtime path runs the two inner Evals via
	// dispatchMethodCallSite (reusing the caller's site, since all inline-eval Eval methods
	// share signature) and applies the binop inline.
	inlineShapeBinopUint
)

// inlineDescriptor caches the classification of a call-site receiver pair.
//
// Allows the runtime to skip handleCallMethod's generic dispatch when the resolved callee
// body is a known fused shape (binop today, extensible to other shapes later). Atomically
// published like methodICSlots; tearing only costs a re-classification, never
// correctness.
type inlineDescriptor struct {
	// recvType is the concrete receiver type this descriptor was classified against. nil
	// entries are never published.
	recvType reflect.Type

	// callee is the resolved compiled function for this (site, type). Always populated - the
	// fallback path needs it even when the inline shape is none.
	callee *CompiledFunction

	// innerCalleeSlots memoises inner Eval callee resolutions.
	//
	// Stores resolved (concrete-recvType -> *CompiledFunction) lookups for the inner Eval
	// dispatches. Populated lazily by invokeInlineEvalInner on each new type. Each slot
	// stores an unsafe.Pointer to an innerCalleeCacheEntry; concurrent access uses
	// atomic.LoadPointer / atomic.StorePointer (mirrors methodICSlots and
	// inlineDescriptorSlots) so two goroutines walking the same descriptor cannot tear an
	// entry. Round-robin eviction via innerCalleeVictim - wrong-eviction only costs a
	// re-resolve, never correctness. The inline-eval AST has 8 concrete node types so
	// innerCalleeSlotCount = 8 covers the typical case without forcing eviction.
	innerCalleeSlots [innerCalleeSlotCount]unsafe.Pointer

	// maskValue captures the trailing `& const` mask in the binop body.
	maskValue uint64

	// leftLayout is the field-read layout for the binop method's left operand access. Valid
	// only when shape == inlineShapeBinopUint.
	leftLayout structFieldLayout

	// rightLayout is the field-read layout for the binop method's right operand access.
	// Valid only when shape == inlineShapeBinopUint.
	rightLayout structFieldLayout

	// innerCalleeVictim is the round-robin eviction counter for innerCalleeSlots. Not
	// atomically updated; tearing only costs a sub-optimal eviction, never correctness.
	innerCalleeVictim uint32

	// binopOpcode encodes which arithmetic op the binop body applies (opAddUint, opSubUint,
	// opMulUint, opModUint). Valid only when shape == inlineShapeBinopUint.
	binopOpcode opcode

	// shape selects the inlined runtime path. inlineShapeNone means "fall back to standard
	// pushCompiledFrame or runTinyLeafInline".
	shape inlineShape

	// maskApplies is true when the binop body has a trailing `& maskValue` masking step.
	maskApplies bool
}

const (
	// inlineDescriptorSlotCount sets the IC slot count per call site.
	//
	// Each slot holds a different receiver-type classification. Smaller than
	// methodICSlotCount because inline-eligible call sites tend to be less polymorphic than
	// method dispatch in general.
	inlineDescriptorSlotCount = 8

	// inlineDescriptorVictimMask masks the per-site round-robin victim counter down to a
	// slot index. Must equal inlineDescriptorSlotCount - 1.
	inlineDescriptorVictimMask = inlineDescriptorSlotCount - 1

	// innerCalleeSlotCount sets the inner-Eval callee cache slot count.
	//
	// Sized per inlineDescriptor for the cache populated by invokeInlineEvalInner. Each slot
	// caches a (concrete-recvType, callee) pair. The inline-eval AST has 8 concrete node
	// types so 8 slots cover the typical inner-dispatch working set without forcing
	// eviction. Slots are atomic.LoadPointer / atomic.StorePointer for tear-free concurrent
	// access.
	innerCalleeSlotCount = 8

	// innerCalleeVictimMask masks innerCalleeVictim down to a slot index. Must equal
	// innerCalleeSlotCount - 1.
	innerCalleeVictimMask = innerCalleeSlotCount - 1

	// methodICSlotCount sets the inline-cache slot count per opCallMethod call site.
	//
	// Sized to absorb common polymorphic-but-bounded receiver-type sets without unduly
	// inflating callSite memory. A linear scan over the atomic pointers remains cheap on the
	// hot path and matches a generous V8/JSC PIC size. Sixteen slots comfortably hold the
	// working set of typical AST-walker dispatches with headroom before any eviction kicks
	// in.
	methodICSlotCount = 16

	// methodICVictimMask masks methodICVictim down to a slot index. Must match
	// methodICSlotCount - 1.
	methodICVictimMask = methodICSlotCount - 1

	// rangeCheckWindowSize is the instruction count of the fuseRangeCheckUintJumpFalse
	// window.
	rangeCheckWindowSize = 8

	// rangeCheckFirstJumpOffset is the position of the first JumpIfFalse relative to the
	// window start.
	rangeCheckFirstJumpOffset = 3

	// rangeCheckSecondLoadOffset is the position of the high-bound LoadUintConstSmall
	// relative to the window start.
	rangeCheckSecondLoadOffset = 4

	// rangeCheckSecondMoveOffset is the position of the second MoveInt sub-op relative to
	// the window start.
	rangeCheckSecondMoveOffset = 6

	// rangeCheckSecondJumpOffset is the position of the second JumpIfFalse relative to the
	// window start.
	rangeCheckSecondJumpOffset = 7

	// rangeCheckFirstNopOffset is the starting slot from which nop padding replaces consumed
	// instructions inside the window.
	rangeCheckFirstNopOffset = 3

	// rangeCheckFirstJumpDelta is the expected signed offset on the first JumpIfFalse
	// chaining into the second jump.
	rangeCheckFirstJumpDelta = 3

	// threeInstructionWindow is the minimum body remaining required to match any of the
	// load+compare+jump triples (e.g. fuseEqUintConstJumpFalse, fuseMapIndexOkJumpFalse).
	threeInstructionWindow = 3
)

// specialisationKey is a fixed-size tuple of reflect.Type used as a generic instantiation
// key.
//
// Holds up to maxSpecialisationTypeArgs entries. reflect.Type is comparable in Go, so
// this fixed-size array is directly usable as a map key without a hash function. Unused
// slots hold the zero reflect.Type which compares equal across keys, but the
// maxSpecialisationTypeArgs cap ensures arities match before lookup so collision is
// impossible.
type specialisationKey [maxSpecialisationTypeArgs]reflect.Type

// FuncName returns the function's qualified name.
//
// Returns string which is the fully qualified function name.
func (cf *CompiledFunction) FuncName() string { return cf.name }

// BodyLen returns the number of bytecode instructions.
//
// Returns int which is the instruction count in the body.
func (cf *CompiledFunction) BodyLen() int { return len(cf.body) }

// SubFunctions returns the nested function literals defined within the function.
//
// Returns []*CompiledFunction which holds the closure definitions nested inside the
// function.
func (cf *CompiledFunction) SubFunctions() []*CompiledFunction { return cf.functions }

// RegisterCounts returns the peak register usage per bank.
//
// Returns [NumRegisterKinds]uint32 which holds the maximum register index used in each
// register bank.
func (cf *CompiledFunction) RegisterCounts() [NumRegisterKinds]uint32 { return cf.numRegisters }

// constantPoolCap returns the configured constant-pool ceiling.
//
// Substitutes the package default when none was set at compile time.
//
// Returns int which is the maximum number of pool entries permitted.
func (cf *CompiledFunction) constantPoolCap() int {
	if cf.maxConstantPoolSize > 0 {
		return cf.maxConstantPoolSize
	}
	return defaultMaxConstantPoolSize
}

// specialisationsCap returns the per-generic specialisation ceiling.
//
// Substitutes the package default when none was set.
//
// Returns int which is the maximum number of specialisations permitted.
func (cf *CompiledFunction) specialisationsCap() int {
	if cf.maxSpecialisations > 0 {
		return cf.maxSpecialisations
	}
	return defaultMaxSpecialisations
}

// methodsCap returns the configured methodTable ceiling.
//
// Substitutes the package default when none was set.
//
// Returns int which is the maximum number of method entries permitted.
func (cf *CompiledFunction) methodsCap() int {
	if cf.maxMethods > 0 {
		return cf.maxMethods
	}
	return defaultMaxMethods
}

// registerMethod records funcIndex under tableName in the receiver's methodTable,
// defending against pathological method declarations by capping the table size.
//
// Takes tableName (string) which is the "TypeName.MethodName" key.
// Takes funcIndex (uint16) which is the position of the method's CompiledFunction in
// rootFunction.functions.
//
// Returns nil on success, or ErrMethodTableExhausted when adding the entry would exceed
// the configured ceiling.
func (cf *CompiledFunction) registerMethod(tableName string, funcIndex uint16) error {
	if _, ok := cf.methodTable[tableName]; ok {
		cf.methodTable[tableName] = funcIndex
		return nil
	}
	if len(cf.methodTable) >= cf.methodsCap() {
		return fmt.Errorf("%w: %s", ErrMethodTableExhausted, tableName)
	}
	if cf.methodTable == nil {
		cf.methodTable = make(map[string]uint16)
	}
	cf.methodTable[tableName] = funcIndex
	return nil
}

// lookupSpecialisation returns the funcIndex of an existing specialisation matching key,
// or false when no specialisation has been registered for that type-args tuple.
//
// Takes key (specialisationKey) which is the type-args tuple to look up.
//
// Returns the funcIndex of the specialised CompiledFunction in rootFunction.functions,
// and a bool indicating whether a match was found.
func (cf *CompiledFunction) lookupSpecialisation(key specialisationKey) (uint16, bool) {
	if cf.specialisations == nil {
		return 0, false
	}
	index, ok := cf.specialisations[key]
	return index, ok
}

// registerSpecialisation records the funcIndex of a fresh specialised body for the given
// type-args tuple. Must be called BEFORE the body is emitted so recursive generic calls
// within the body find the reserved funcIndex and emit a normal opCall rather than
// triggering an infinite re-specialisation cascade.
//
// Takes key (specialisationKey) which is the type-args tuple.
// Takes funcIndex (uint16) which is the index of the specialised CompiledFunction in
// rootFunction.functions.
//
// Returns nil on success, or ErrSpecialisationLimitReached when the configured ceiling
// for this generic callee is exhausted; callers should fall back to the generic reflect
// path on this error.
func (cf *CompiledFunction) registerSpecialisation(key specialisationKey, funcIndex uint16) error {
	if _, ok := cf.specialisations[key]; ok {
		cf.specialisations[key] = funcIndex
		return nil
	}
	if len(cf.specialisations) >= cf.specialisationsCap() {
		return fmt.Errorf("%w: %s", ErrSpecialisationLimitReached, cf.name)
	}
	if cf.specialisations == nil {
		cf.specialisations = make(map[specialisationKey]uint16, 1)
	}
	cf.specialisations[key] = funcIndex
	return nil
}

// reflectFuncType returns the reflect.Type for the method's signature excluding the
// receiver parameter. Used for creating bound method values via reflect.MakeFunc.
//
// Returns the function type and true if the function has parameters, or nil and false
// otherwise.
func (cf *CompiledFunction) reflectFuncType() (reflect.Type, bool) {
	if len(cf.parameterKinds) == 0 {
		return nil, false
	}
	parameterKinds := cf.parameterKinds[1:]
	inTypes := make([]reflect.Type, len(parameterKinds))
	for i, k := range parameterKinds {
		inTypes[i] = kindDefaultReflectType(k)
	}
	outTypes := make([]reflect.Type, len(cf.resultKinds))
	for i, k := range cf.resultKinds {
		outTypes[i] = kindDefaultReflectType(k)
	}
	return reflect.FuncOf(inTypes, outTypes, cf.isVariadic), true
}

// reflectMethodExprType returns the reflect.Type for the method's signature including the
// receiver as the first parameter. General-register params use interface{} so concrete
// struct types can pass through reflect.Call without type mismatches.
//
// Returns the function type and true if the function has parameters, or nil and false
// otherwise.
func (cf *CompiledFunction) reflectMethodExprType() (reflect.Type, bool) {
	if len(cf.parameterKinds) == 0 {
		return nil, false
	}
	inTypes := make([]reflect.Type, len(cf.parameterKinds))
	for i, k := range cf.parameterKinds {
		if k == registerGeneral {
			inTypes[i] = reflect.TypeFor[any]()
		} else {
			inTypes[i] = kindDefaultReflectType(k)
		}
	}
	outTypes := make([]reflect.Type, len(cf.resultKinds))
	for i, k := range cf.resultKinds {
		outTypes[i] = kindDefaultReflectType(k)
	}
	return reflect.FuncOf(inTypes, outTypes, cf.isVariadic), true
}

// UpvalueDescriptor describes a single captured variable in a closure.
type UpvalueDescriptor struct {
	// index is the register index in the enclosing scope, or the upvalue index if isLocal is
	// false (transitive capture).
	index uint8

	// kind is the register bank of the captured variable. When the variable is heap-promoted
	// (isIndirect is true) this is registerGeneral because the parent's register holds the
	// *T pointer to the heap cell.
	kind registerKind

	// isLocal is true when the upvalue captures a register directly from the immediately
	// enclosing function. When false, the upvalue is captured transitively from the
	// enclosing function's own upvalue table.
	isLocal bool

	// isIndirect is true when the captured variable is heap-promoted in the enclosing scope.
	// handleMakeClosure stores the *T pointer from the parent's general register into
	// cell.generalValue and marks cell.isIndirect / cell.originalKind so handleGetUpvalue
	// and handleSetUpvalue dereference through the pointer rather than reading or writing
	// the per-kind snapshot fields.
	isIndirect bool

	// originalKind names the typed register bank the variable had before heap promotion.
	//
	// The closure body expects values in this bank, so handleGetUpvalue copies the
	// dereferenced value into originalKind's slot rather than into the descriptor's kind.
	// Meaningful only when isIndirect is true.
	originalKind registerKind
}

// callSite describes a function call in the compiled bytecode. It stores all the
// information the VM needs to set up arguments and retrieve return values without
// additional opcodes.
type callSite struct {
	// methodICSlots is the polymorphic inline cache for opCallMethod dispatch.
	//
	// Each slot caches a (receiverType, funcIndex) entry observed at this call site. Slots
	// are read and published independently via atomic.LoadPointer / atomic.StorePointer so
	// multiple goroutines executing the same bytecode (callSites are shared on
	// *CompiledFunction) cannot tear an entry. Cache hit: any slot whose entry.receiverType
	// matches the current receiver type. Cache miss: the slow path resolves via methodTable
	// / resolvePromotedMethod, then publishes a fresh entry into a round-robin victim slot.
	//
	// Slots are nil when unprobed. There is no "disable" sentinel; wrong-eviction only costs
	// a re-resolve, never correctness, so the cache stays warm indefinitely for code that
	// uses 2-4 receiver types per call site.
	//
	// Runtime-only cache. Not persisted to flatbuffers; warms on first call after Compile.
	// Reads at runtime always go through a *callSite from cf.callSites, never a copied
	// value, so concurrent updates land on the same storage cell.
	methodICSlots [methodICSlotCount]unsafe.Pointer

	// inlineDescriptorSlots caches per-receiver-type inlineDescriptor classifications.
	//
	// Consulted by handleCallMethodInlineable's fast path. Each slot stores an
	// unsafe.Pointer to an inlineDescriptor. Concurrent access uses atomic.LoadPointer /
	// atomic.StorePointer (mirrors methodICSlots). Round-robin eviction via
	// inlineDescriptorVictim - wrong-eviction only costs re-classify. Only populated when
	// the function's rewriteInlineableMethodCalls pass tagged the call site as inlineable;
	// opCallMethod sites leave these slots untouched and pay no overhead.
	inlineDescriptorSlots [inlineDescriptorSlotCount]unsafe.Pointer

	// lastReceiverType is a non-atomic single-slot cache of the most recently observed
	// concrete receiver type at this call site. dispatchMethodCallSite checks this BEFORE
	// the methodICSlots walk; on hit the entire IC scan is skipped.
	//
	// Concurrent writes can tear; tearing only causes a fast-path miss - the methodICSlots
	// walk re-resolves. No correctness risk. Pattern mirrors cachedClosurePtr below.
	lastReceiverType reflect.Type

	// runtimeVariadicSliceType is the reflect.Type of the variadic parameter's slice.
	//
	// Example: []int for ...int. Set for callees whose signature is variadic and where the
	// source did not use the ellipsis spread. The runtime closure dispatcher uses this to
	// pack the trailing register arguments into a slice before transferring control to a
	// compiled variadic function. Nil for non-variadic sites and for sites that already pass
	// a spread slice. Not serialised through the bytecode; recomputed on recompilation.
	runtimeVariadicSliceType reflect.Type

	// nativeFastPath publishes the fast-path classification atomically.
	//
	// Used for native call sites dispatched via handleCallNative. Nil means unprobed,
	// otherwise it holds a *nativeFastPathEntry whose fn is nativeFastPathNone when no fast
	// path matched. Read with atomic.LoadPointer and written with atomic.StorePointer
	// because the enclosing CompiledFunction (and so the call site) is shared across the
	// per-goroutine VMs a `go` statement spawns; publishing the two-word fn interface
	// non-atomically risks a torn read.
	nativeFastPath unsafe.Pointer

	// cachedClosureRoot is the root *CompiledFunction extracted from the last-seen closure,
	// paired with cachedClosurePtr.
	cachedClosureRoot *CompiledFunction

	// lastReceiverCallee partners with lastReceiverType; the pair is written together on
	// every IC resolve. Tearing safety: see lastReceiverType above.
	lastReceiverCallee *CompiledFunction

	// cachedCallee is the pre-bound *CompiledFunction for the funcIndex.
	//
	// Direct (non-closure) call handlers prefer this over the per-call
	// vm.functions[funcIndex] lookup so the bounds check + slice load disappear from the hot
	// path. Nil when the site is a closure call or when the callee can't be resolved at emit
	// time (e.g. forward references resolved later); the runtime falls back to the slice
	// lookup in that case.
	cachedCallee *CompiledFunction

	// cachedClosurePtr caches the last-seen *runtimeClosure raw pointer at a closure call
	// site.
	//
	// Enables the hot-loop fast path where the same closure is invoked repeatedly
	// (closures_pipeline's applyMap/applyFilter/applyReduce each call the same closure 100K
	// times). When the reflect.Value's raw pointer matches the cache, handleCall skips the
	// TypeAssert + struct-field reads and jumps straight to the cached callee/upvalues/root.
	//
	// Tearing is benign here: a torn read (closurePtr matching but callee/upvalues/root
	// stale from a different concurrent update) only happens when two goroutines invoke the
	// same callSite with different closures - in practice the same source-level call site is
	// single-closure across iterations. The cache is initialised to nil; the first call
	// seeds it.
	cachedClosurePtr unsafe.Pointer

	// cachedClosureCallee is the *CompiledFunction extracted from the last-seen closure
	// value, paired with cachedClosurePtr.
	cachedClosureCallee *CompiledFunction

	// nativeFunctionPath caches the dotted symbol identifier.
	//
	// Examples: "net/http.Get", "(*bytes.Buffer).Write". Used when consulting the
	// CapabilityHook. Resolved lazily on first hook consultation via runtime.FuncForPC and
	// reused across calls. Empty when no hook is configured or when resolution failed (the
	// hook still runs with the empty path; embedders that need names install a hook that
	// tolerates ""). Runtime-only; not serialised.
	nativeFunctionPath string

	// variadicArgumentsBuffer is a pre-allocated buffer for variadic fast-path calls. Avoids
	// make([]any, n) per call for functions like fmt.Sprintf that take ...interface{}
	// parameters.
	variadicArgumentsBuffer []any

	// cachedClosureUpvalues holds the upvalue cells extracted from the last-seen closure,
	// paired with cachedClosurePtr.
	cachedClosureUpvalues []*upvalueCell

	// argCopyProgram is the pre-computed per-argument copy plan for this call site.
	//
	// Built at compile time by buildCallArgCopyProgram once the callee's parameterKinds are
	// known. The runtime copyCallArgs walks the slice with one tight switch per arg. Nil for
	// sites built before the callee was resolved (e.g. closure call sites with kinds-unknown
	// args); those fall back to the generic walk.
	argCopyProgram []callArgCopy

	// argumentStaticTypeNames holds each argument's bare source-level named type.
	//
	// Set for arguments whose type is a *types.Named (e.g. "Colour", "Bomb"), unwrapping one
	// pointer layer. Empty string when the argument's static type is not named (basic,
	// slice, map, etc.). Used by tryBuildInterfaceAdapter to select the right adapter for
	// named primitives where the typeNames-by-reflect-type lookup collides (every int-backed
	// named type maps to the same int64 reflect.Type). Populated only for native call sites;
	// nil for compiled-function targets where the callee's parameter banks are already
	// typed.
	argumentStaticTypeNames []string

	// argumentStaticTypeStrings holds each argument's Go-syntax type representation.
	//
	// Produced by go/types.TypeString with a package-name qualifier (e.g. "int", "[]int",
	// "*main.Bomb", "string"). Empty string when the static type is unavailable. Used by the
	// %T fmt interceptor to substitute the source-level type rather than piko's internal
	// reflect.Type representation.
	argumentStaticTypeStrings []string

	// parameterInterfaceFlags marks each fixed parameter of the callee that has interface
	// kind.
	//
	// Computed at compile time from the resolved callee signature.
	// `siteArgsRequireInterfaceAdapter` reads this to decide whether the slow path's adapter
	// wrapping needs to fire - a piko-defined type whose receiver type is fine for a
	// concrete parameter slot but needs adapter wrapping when the slot is an interface.
	//
	// The bit-of-bool representation rather than a packed bitmask is deliberate: callsites
	// with <= 4 fixed args are the dominant case and the slice header itself is the same
	// eight words either way; indexing is a single load. Nil for sites without any interface
	// parameters (zero allocation in the common no-interface case).
	//
	// Index 0 corresponds to the first fixed parameter (after the receiver for methods).
	// Variadic tails are handled separately because their target type is `any` / interface{}
	// which is the universal interface-kind case and is checked dynamically.
	parameterInterfaceFlags []bool

	// parameterTypes caches the expected parameter types for native calls. Populated lazily
	// on first call to avoid repeated fn.Type().In(i).
	parameterTypes []reflect.Type

	// arguments records where each argument lives in the CALLER's frame.
	arguments []varLocation

	// linkedTypeArgs holds the instantiated type arguments for a //piko:link-routed generic
	// call, resolved at compile time from types.Info.Instances. When non-empty the native
	// call handler prepends one reflect.Type value per element before the regular arguments
	// and skips the nativeFastPath / parameterTypes caches.
	linkedTypeArgs []reflect.Type

	// returns records where to put each return value in the CALLER's frame after the call
	// completes.
	returns []varLocation

	// inlineDescriptorVictim is the round-robin eviction counter for inlineDescriptorSlots.
	// Not atomically updated; tearing only costs a sub-optimal eviction, never correctness.
	inlineDescriptorVictim uint32

	// methodICVictim is the round-robin eviction counter for methodICSlots.
	//
	// Incremented atomically on every cache miss; the low bits select the slot to overwrite.
	// Wrong-eviction is always recoverable (re-resolve on next miss) so no ordering
	// guarantee is needed between the counter and the slot it selects.
	methodICVictim uint32

	// funcIndex is the index into the enclosing function's functions slice for the callee.
	// Ignored when isClosure is true.
	funcIndex uint16

	// closureRegister is the general register holding the closure value. Only used when
	// isClosure is true.
	closureRegister uint8

	// nativeRegister is the general register holding the native function value. Only used
	// when isNative is true.
	nativeRegister uint8

	// isClosure is true when the callee is a closure stored in a general register rather
	// than a static function reference.
	isClosure bool

	// isNative is true when the callee is a native Go function stored in a general register
	// (not a compiled function).
	isNative bool

	// isMethod is true when the callee is a bound method obtained via opGetMethod, where
	// handleCallNative validates the cached fast path against the current receiver address
	// before reuse.
	isMethod bool

	// methodReceiverRegister is the general register holding the receiver for method calls
	// (only valid when isMethod is true). Used to validate the cached fast path by comparing
	// the receiver address across invocations.
	methodReceiverRegister uint8

	// runtimeVariadicNumFixed counts the fixed (non-variadic) parameters that precede the
	// variadic slice parameter in the callee signature. Only meaningful when
	// runtimeVariadicSliceType is non-nil.
	runtimeVariadicNumFixed uint8

	// isEllipsisSpread is true when the source call used the `...` ellipsis spread on the
	// final argument, requiring dispatch via reflect.Value.CallSlice instead of Call (so the
	// trailing slice is passed as the variadic parameter rather than spread into individual
	// values). Only meaningful for native calls.
	isEllipsisSpread bool

	// tailReuseFrameInPlace marks a tail-call site that can reuse the caller's register file
	// in place.
	//
	// True when the callee's register counts match the caller in every bank. The runtime
	// tail-call handler reads this flag to skip the arena.Restore, arena.SaveInto, and
	// AllocRegistersIntoCached dance entirely; the caller's register windows ARE the
	// callee's when the layouts match, so the existing slice headers in frame.registers and
	// frame.arenaSave indices remain correct. For self-tail-calls this is always true; for
	// cross-function tail-calls of identical-shape callees it is often true. Only meaningful
	// for tail-call sites (opTailCall); ignored for regular call sites.
	tailReuseFrameInPlace bool

	// tailArgsAlias marks a tail-call site whose argument copy needs a snapshot to avoid
	// aliasing.
	//
	// When true the runtime must snapshot args to a side buffer before placing them, because
	// copying directly against the shared register file (the tailReuseFrameInPlace case)
	// would suffer source-destination aliasing. When false the runtime can call copyCallArgs
	// directly for a no-snapshot fast path. Only meaningful when tailReuseFrameInPlace is
	// true.
	tailArgsAlias bool

	// recursionUnrolled marks a self-recursive call site already spliced once by the
	// recursive-unrolling path.
	//
	// Set in trySpliceCallAt after a single-level splice succeeds in canInline so that
	// re-entry through the same site refuses further unrolling, preventing infinite
	// expansion. Only meaningful for sites whose cachedCallee equals the containing caller;
	// ignored for cross-function calls.
	recursionUnrolled bool

	// nativeIsVariadic records the resolved reflect.Type.IsVariadic() flag.
	//
	// variadicElementTypeForSite must consult this rather than guessing from a trailing
	// slice parameter, because non-variadic native functions like reflect.Value.Call
	// legitimately take a []T final parameter (bug 750).
	nativeIsVariadic bool

	// nativeIsVariadicSeen is true once nativeIsVariadic has been populated by
	// cacheParamTypes.
	nativeIsVariadicSeen bool
}

// addCallSite adds a call site and returns its index.
//
// Takes site (*callSite) which specifies the call site descriptor to append. Pointer
// rather than value because callSite is large (~456 bytes); the receiver still gets a
// fresh copy in the slice.
//
// Returns the index of the newly added call site, or ErrConstantPoolExhausted when the
// call-site table has reached its configured ceiling.
func (cf *CompiledFunction) addCallSite(site *callSite) (uint16, error) {
	if len(cf.callSites) >= cf.constantPoolCap() {
		return 0, fmt.Errorf("%w: callSites", ErrConstantPoolExhausted)
	}
	index := len(cf.callSites)
	cf.callSites = append(cf.callSites, *site)
	return safeconv.IntToUint16(index), nil
}

// addIntConstant adds an int64 constant and returns its index.
//
// Takes v (int64) which specifies the constant value to add or look up.
//
// Returns the index of the constant in the IntConstants pool, or ErrConstantPoolExhausted
// when the pool has reached its ceiling.
func (cf *CompiledFunction) addIntConstant(v int64) (uint16, error) {
	return addDedupedConstant(&cf.intConstants, &cf.intConstIndex, v, cf.constantPoolCap())
}

// addFloatConstant adds a float64 constant and returns its index.
//
// Takes v (float64) which specifies the constant value to add or look up.
//
// Returns the index of the constant in the FloatConstants pool, or
// ErrConstantPoolExhausted when the pool has reached its ceiling.
func (cf *CompiledFunction) addFloatConstant(v float64) (uint16, error) {
	return addDedupedConstant(&cf.floatConstants, &cf.floatConstIndex, v, cf.constantPoolCap())
}

// addStringConstant adds a string constant and returns its index.
//
// Takes v (string) which specifies the constant value to add or look up.
//
// Returns the index of the constant in the StringConstants pool, or
// ErrConstantPoolExhausted when the pool has reached its ceiling.
func (cf *CompiledFunction) addStringConstant(v string) (uint16, error) {
	return addDedupedConstant(&cf.stringConstants, &cf.stringConstIndex, v, cf.constantPoolCap())
}

// addGeneralConstant adds a reflect.Value constant with its reconstruction descriptor and
// returns its index.
//
// Takes v (reflect.Value) which specifies the constant value to append.
// Takes descriptor (generalConstantDescriptor) which records how to reconstruct the value
// from a serialised form.
//
// Returns the index of the constant in the GeneralConstants pool, or
// ErrConstantPoolExhausted when the pool has reached its ceiling.
func (cf *CompiledFunction) addGeneralConstant(v reflect.Value, descriptor generalConstantDescriptor) (uint16, error) {
	if len(cf.generalConstants) >= cf.constantPoolCap() {
		return 0, fmt.Errorf("%w: generalConstants", ErrConstantPoolExhausted)
	}
	index := len(cf.generalConstants)
	cf.generalConstants = append(cf.generalConstants, v)
	cf.generalConstantDescriptors = append(cf.generalConstantDescriptors, descriptor)
	return safeconv.IntToUint16(index), nil
}

// addBoolConstant adds a bool constant and returns its index.
//
// Takes v (bool) which specifies the constant value to add or look up.
//
// Returns the index of the constant in the BoolConstants pool, or
// ErrConstantPoolExhausted when the pool has reached its ceiling.
func (cf *CompiledFunction) addBoolConstant(v bool) (uint16, error) {
	for i, c := range cf.boolConstants {
		if c == v {
			return safeconv.IntToUint16(i), nil
		}
	}
	if len(cf.boolConstants) >= cf.constantPoolCap() {
		return 0, fmt.Errorf("%w: boolConstants", ErrConstantPoolExhausted)
	}
	index := len(cf.boolConstants)
	cf.boolConstants = append(cf.boolConstants, v)
	return safeconv.IntToUint16(index), nil
}

// addUintConstant adds a uint64 constant and returns its index.
//
// Takes v (uint64) which specifies the constant value to add or look up.
//
// Returns the index of the constant in the UintConstants pool, or
// ErrConstantPoolExhausted when the pool has reached its ceiling.
func (cf *CompiledFunction) addUintConstant(v uint64) (uint16, error) {
	return addDedupedConstant(&cf.uintConstants, &cf.uintConstIndex, v, cf.constantPoolCap())
}

// addComplexConstant adds a complex128 constant and returns its index.
//
// Takes v (complex128) which specifies the constant value to add or look up.
//
// Returns the index of the constant in the ComplexConstants pool, or
// ErrConstantPoolExhausted when the pool has reached its ceiling.
func (cf *CompiledFunction) addComplexConstant(v complex128) (uint16, error) {
	return addDedupedConstant(&cf.complexConstants, &cf.complexConstIndex, v, cf.constantPoolCap())
}

// addTypeRef adds a reflect.Type to the type table and returns its index.
//
// Takes t (reflect.Type) which specifies the type to add or look up.
//
// Returns the index of the type in the TypeTable, or ErrConstantPoolExhausted when the
// table has reached its ceiling.
func (cf *CompiledFunction) addTypeRef(t reflect.Type) (uint16, error) {
	return cf.addTypeRefWithMethods(t, nil)
}

// addTypeRefWithMethods registers a reflect.Type entry with method-set constraints.
//
// The methods slice may be nil (the no-constraint case - empty interface / any). When
// non-nil, handleTypeAssert consults the parallel typeTableInterfaceMethods slice to
// enforce method-set membership.
//
// Takes t (reflect.Type) which is the type to register.
// Takes methods ([]string) which lists required method names, or nil.
//
// Returns uint16 which is the typeTable index of t (existing or fresh).
// Returns error when the table is full (ErrConstantPoolExhausted).
func (cf *CompiledFunction) addTypeRefWithMethods(t reflect.Type, methods []string) (uint16, error) {
	if cf.typeRefIndex == nil && len(cf.typeTable) > 0 {
		cf.typeRefIndex = make(map[reflect.Type]uint16, len(cf.typeTable))
		for i, c := range cf.typeTable {
			if _, ok := cf.typeRefIndex[c]; !ok {
				cf.typeRefIndex[c] = uint16(i)
			}
		}
	}
	if index, ok := cf.typeRefIndex[t]; ok {
		if int(index) < len(cf.typeTableInterfaceMethods) &&
			len(cf.typeTableInterfaceMethods[index]) == 0 && len(methods) > 0 {
			cf.typeTableInterfaceMethods[index] = methods
		}
		return index, nil
	}
	if len(cf.typeTable) >= cf.constantPoolCap() {
		return 0, fmt.Errorf("%w: typeTable", ErrConstantPoolExhausted)
	}
	index := safeconv.IntToUint16(len(cf.typeTable))
	cf.typeTable = append(cf.typeTable, t)
	cf.typeTableDescriptors = append(cf.typeTableDescriptors, reflectTypeToDescriptor(t))
	for len(cf.typeTableInterfaceMethods) < len(cf.typeTable)-1 {
		cf.typeTableInterfaceMethods = append(cf.typeTableInterfaceMethods, nil)
	}
	cf.typeTableInterfaceMethods = append(cf.typeTableInterfaceMethods, methods)
	if cf.typeRefIndex == nil {
		cf.typeRefIndex = make(map[reflect.Type]uint16)
	}
	cf.typeRefIndex[t] = index
	return index, nil
}

// emit appends an instruction to the function body and returns its offset for later
// patching.
//
// Takes op (opcode) which specifies the opcode.
// Takes a (uint8) which specifies the first instruction operand.
// Takes b (uint8) which specifies the second instruction operand.
// Takes c (uint8) which specifies the third instruction operand.
//
// Returns the instruction offset in the body.
func (cf *CompiledFunction) emit(op opcode, a, b, c uint8) int {
	pc := len(cf.body)
	cf.body = append(cf.body, makeInstruction(op, a, b, c))
	if cf.emittedInlineBlocker == inlineRefusalUnknown {
		if r := blockerForOpcode(op); r != inlineRefusalUnknown {
			cf.emittedInlineBlocker = r
		}
	}
	if cf.debugEmitHook != nil {
		cf.debugEmitHook(pc)
	}
	return pc
}

// emitWide emits an instruction with a 16-bit wide index split across B and C operands.
//
// Takes op (opcode) which specifies the opcode.
// Takes a (uint8) which specifies the first operand.
// Takes wide (uint16) which specifies the 16-bit index to encode in B (low byte) and C
// (high byte).
//
// Returns the instruction offset in the body.
func (cf *CompiledFunction) emitWide(op opcode, a uint8, wide uint16) int {
	lo, hi := splitWide(wide)
	return cf.emit(op, a, lo, hi)
}

// emitExtension emits an opExt instruction with a 16-bit payload in A (low byte) and B
// (high byte), plus an extra byte in C.
//
// Takes wide (uint16) which specifies the 16-bit payload.
// Takes c (uint8) which specifies the extra operand in C.
//
// Returns the instruction offset in the body.
func (cf *CompiledFunction) emitExtension(wide uint16, c uint8) int {
	lo, hi := splitWide(wide)
	return cf.emit(opExt, lo, hi, c)
}

// emitJump emits a jump instruction whose offset is patched later by patchJump.
//
// Takes op (opcode) which specifies the jump opcode.
// Takes conditionRegister (uint8) which specifies the condition register index.
//
// Returns the instruction offset for later patching with patchJump.
func (cf *CompiledFunction) emitJump(op opcode, conditionRegister uint8) int {
	return cf.emit(op, conditionRegister, 0, 0)
}

// emitTier1Jump emits an unconditional tier-1 jump whose offset is patched later by
// patchJump.
//
// Emits (opDrillTier1, subOpJump, 0, 0). The 16-bit offset lives in operands B|(C<<8) -
// the same byte positions patchJump writes to for a tier-0 jump, so patchJump works
// unchanged.
//
// Returns the instruction offset for later patching with patchJump.
func (cf *CompiledFunction) emitTier1Jump() int {
	return cf.emit(opDrillTier1, uint8(subOpJump), 0, 0)
}

// patchJump patches an earlier emitted jump instruction at patchPC to jump to the current
// instruction offset.
//
// Takes patchPC (int) which specifies the instruction offset to patch.
func (cf *CompiledFunction) patchJump(patchPC int) {
	lo, hi := cf.encodeJumpOffset(len(cf.body) - patchPC - 1)
	cf.body[patchPC].b = lo
	cf.body[patchPC].c = hi
}

// encodeJumpOffset splits a PC-relative jump distance into operand bytes.
//
// Flags the function when the distance overflows the signed 16-bit jump encoding. A
// flagged function is rejected by optimise with errCompileJumpRange, so the compiler
// surfaces a clean error instead of safeconv.MustIntToInt16 panicking and unwinding into
// the host. The placeholder bytes returned on overflow are never executed because
// compilation aborts.
//
// Takes delta (int) which is the PC-relative jump distance.
//
// Returns lowByte (uint8) which is the low operand byte.
// Returns highByte (uint8) which is the high operand byte.
func (cf *CompiledFunction) encodeJumpOffset(delta int) (lowByte, highByte uint8) {
	if !fitsJumpOffset(delta) {
		cf.jumpRangeExceeded = true
		return 0, 0
	}
	return splitOffset(safeconv.MustIntToInt16(delta))
}

// currentPC returns the offset for the next instruction to be emitted.
//
// Returns the current program counter offset.
func (cf *CompiledFunction) currentPC() int {
	return len(cf.body)
}

// optimise applies peephole optimisations to the instruction body.
//
// Fuses common instruction sequences into superinstructions. Dead slots are replaced with
// opNop to preserve instruction indices (jump offsets reference indices, not byte
// offsets).
//
// Takes ctx (context.Context) which cancels the optimisation pass.
//
// Returns error when cancellation fires or a sub-pass reports failure.
func (cf *CompiledFunction) optimise(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("optimise cancelled: %w", err)
	}
	if cf.jumpRangeExceeded {
		return fmt.Errorf("%w: in function %q", errCompileJumpRange, cf.name)
	}
	if err := cf.hoistLoopInvariantStructFieldReads(ctx); err != nil {
		return err
	}
	body := cf.body
	if err := cf.runPeepholeFusions(ctx, body); err != nil {
		return err
	}
	cf.releaseConstantIndices()
	cf.elideUnusedSyncClosureUpvalues(body)
	cf.elideRedundantTruncateAfterBitAnd(body)
	cf.elideRedundantBitAndAfterTruncate(body)
	if err := cf.elideRedundantStructFieldRead(ctx, body); err != nil {
		return err
	}
	cf.syncSourceMapAfterOptimise(body)
	cf.ensurePrecomputedAllocCounts()
	cf.classifyTinyLeaf()
	for _, child := range cf.functions {
		if err := child.optimise(ctx); err != nil {
			return err
		}
	}
	return nil
}

// runPeepholeFusions walks the body once and applies every peephole fusion / load-const
// optimisation the dispatcher recognises. Each fuse* helper returns true when it consumed
// the slot and the loop must advance to the next instruction.
//
// Takes ctx (context.Context) which cancels the walk.
// Takes body ([]instruction) which is cf.body captured locally so the loop avoids
// repeated slice header loads.
//
// Returns error when cancellation fires.
func (cf *CompiledFunction) runPeepholeFusions(ctx context.Context, body []instruction) error {
	n := len(body)
	jumpTargets := cf.buildJumpTargets(body)
	for i := range n {
		if i&optimisationLoopCheckMask == 0 {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("runPeepholeFusions cancelled: %w", err)
			}
		}
		if cf.fuseCopyStructFieldGeneralT0(body, i, n, jumpTargets) {
			continue
		}
		if cf.fuseLongPatterns(body, i, n, jumpTargets) ||
			cf.fuseThreeInstrPatterns(body, i, n, jumpTargets) ||
			cf.fuseArithConst(body, i, n, jumpTargets) ||
			cf.fuseAddIntJump(body, i, n, jumpTargets) ||
			cf.fuseConcatRune(body, i, n, jumpTargets) ||
			cf.fuseAppendMove(body, i, n, jumpTargets) ||
			cf.fuseMoveElimination(body, i, n, jumpTargets) ||
			cf.fuseStringIndexToInt(body, i, n, jumpTargets) {
			continue
		}
		cf.optimiseLoadIntConst(body, i)
		cf.optimiseLoadUintConst(body, i)
	}
	return nil
}

// releaseConstantIndices clears the per-pool dedup maps and the emit-time debug hook once
// optimise has finished walking the body. The runtime never reads these, so dropping them
// releases memory proportional to the program size.
func (cf *CompiledFunction) releaseConstantIndices() {
	cf.intConstIndex = nil
	cf.floatConstIndex = nil
	cf.stringConstIndex = nil
	cf.uintConstIndex = nil
	cf.complexConstIndex = nil
	cf.typeRefIndex = nil
	cf.debugEmitHook = nil
}

// elideUnusedSyncClosureUpvalues replaces opSyncClosureUpvalues with opNop when the
// caller never emits opMakeClosure.
//
// Compile-time elision: opSyncClosureUpvalues after a closure call is only meaningful
// when the caller might have a sharedCells map populated, which happens only when the
// caller itself emits opMakeClosure. For closure-calling functions that never CREATE
// closures (the dominant case for pipeline helpers that take a closure as a parameter and
// call it in a tight loop), the runtime sync handler always early-returns at the
// sharedCells==nil check. Skip emission entirely.
//
// Takes body ([]instruction) which is the function's instruction stream.
func (cf *CompiledFunction) elideUnusedSyncClosureUpvalues(body []instruction) {
	if cf.mayCreateSharedCells() {
		return
	}
	for i := range body {
		if body[i].op == opSyncClosureUpvalues && body[i].b == 0 {
			body[i] = makeInstruction(opNop, 0, 0, 0)
		}
	}
}

// mayCreateSharedCells reports whether the body emits any opcode that can populate the
// caller frame's sharedCells map.
//
// True when an opMakeClosure is present anywhere in the body (closure creation is the
// sole producer of shared upvalue cells). When false, every opSyncClosureUpvalues with
// b==0 in the function is provably a no-op at runtime - handleSyncClosureUpvalues' fast
// path early-returns on sharedCells==nil. Eliding such ops at compile time skips both the
// tier-2 dispatch trampoline and the handler-side check, ~30-50 ns per closure call site.
//
// Returns true when an opMakeClosure exists in the body.
func (cf *CompiledFunction) mayCreateSharedCells() bool {
	for i := range cf.body {
		if cf.body[i].op == opMakeClosure {
			return true
		}
	}
	return false
}

// addDedupedConstant appends v to the pool when no equal entry is already present,
// returning the existing or freshly-added index.
//
// Lazily rebuilds the dedup map from the pool when the map is nil but the pool is
// non-empty. optimise() nils the index to release memory once compile is done;
// post-optimise adders (e.g. the bytecode inliner merging constant pools across
// functions) must reseed the dedup map from existing entries, otherwise additions would
// duplicate pre-existing constants.
//
// Takes pool (*[]K) which points to the parallel pool slice.
// Takes index (*map[K]uint16) which points to the dedup index map.
// Takes v (K) which is the value to add or look up.
// Takes maxPoolSize (int) which caps total entries; on overflow returns (0,
// ErrConstantPoolExhausted) rather than panicking.
//
// Returns the existing index when an equal entry is present, otherwise the index of the
// freshly-appended entry, or the sentinel error when the pool would exceed maxPoolSize.
func addDedupedConstant[K comparable](pool *[]K, index *map[K]uint16, v K, maxPoolSize int) (uint16, error) {
	if *index == nil && len(*pool) > 0 {
		seeded := make(map[K]uint16, len(*pool))
		for i, c := range *pool {
			if _, ok := seeded[c]; !ok {
				seeded[c] = uint16(i)
			}
		}
		*index = seeded
	}
	if existing, ok := (*index)[v]; ok {
		return existing, nil
	}
	if len(*pool) >= maxPoolSize {
		return 0, fmt.Errorf("%w: %T pool", ErrConstantPoolExhausted, v)
	}
	newIndex := safeconv.IntToUint16(len(*pool))
	*pool = append(*pool, v)
	if *index == nil {
		*index = make(map[K]uint16)
	}
	(*index)[v] = newIndex
	return newIndex, nil
}
