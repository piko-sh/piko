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
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync/atomic"
	"unsafe"
)

const (
	// maxCallDepth is the default maximum call stack depth before a stack overflow error is
	// raised.
	//
	// Conservative default - piko is a constrained, embeddable interpreter and unbounded
	// recursion pegs memory because each frame holds register-bank slices in the arena.
	// Programs needing deeper recursion (recursive-descent parsers on large documents, deep
	// tree walks) raise the bound via the WithMaxCallDepth(n) service option.
	maxCallDepth = 10000

	// cancellationCheckMask is the bitmask applied to the operation counter to decide when
	// to check for context cancellation.
	cancellationCheckMask uint32 = 0x3FF

	// errIdxOutOfRangeFmt is the format string for index-out-of-range error messages in VM
	// handlers.
	errIdxOutOfRangeFmt = "%w: %d with length %d"

	// opcodeTableSize is the number of entries in the opcode dispatch table (one per
	// possible uint8 opcode value).
	opcodeTableSize = 256

	// checkpointFlagGCPending signals a pending arena MinorGC at next safe point.
	//
	// Set when the arena crosses its nextGCAt threshold (via RegisterArena.noteAlloc) or by
	// the runtime.GC() intrinsic. Cleared by the dispatch loop after running GC.
	checkpointFlagGCPending uint8 = 1 << 0

	// arenaConcatStringAvgBytes is the per-call byte budget assumed for an opConcatString.
	// Real-world string concatenations span orders of magnitude; 32 bytes is a defensive
	// estimate that covers small-to-medium results without blowing out the slab for programs
	// that never concatenate large strings.
	arenaConcatStringAvgBytes = 32

	// arenaConcatRuneStringAvgBytes is the per-call byte budget for opConcatRuneString. The
	// base string contributes most of the length; the rune adds at most 4 UTF-8 bytes.
	arenaConcatRuneStringAvgBytes = 16

	// arenaRuneToStringMaxBytes is the worst-case UTF-8 length of a single rune.
	arenaRuneToStringMaxBytes = 4

	// arenaBytesToStringAvgBytes is the per-call byte budget for subOpBytesToString. The
	// actual byte slice length is dynamic; 64 bytes covers typical tokenisation /
	// small-string conversion patterns without bloating tiny programs.
	arenaBytesToStringAvgBytes = 64

	// arenaItoaMaxBytes is the worst-case byte length of an int64 formatted in base 10
	// (matches itoaMaxDigitsBase10 in register_arena_unsafe.go but expressed here so the
	// hint module can stand alone).
	arenaItoaMaxBytes = 20

	// arenaFormatIntMaxBytes is the worst-case byte length of an int64 formatted in any base
	// accepted by strconv.FormatInt (base 2 is the widest at 64 binary digits + 1 sign).
	arenaFormatIntMaxBytes = 65

	// arenaMakeSliceAvgCapacity is the per-occurrence element budget for makeSlice.
	//
	// Charged once per subOpMakeSlice* opcode. The capacity argument is dynamic so the
	// compiler can't see it; 16 is a reasonable guess covering loops that build small typed
	// slices. Programs that allocate larger slices fall back to the runtime grow path once
	// the pre-sized slab is exhausted - still better than growing from initial-default
	// capacity on every alloc.
	arenaMakeSliceAvgCapacity = 16

	// inlineUpvalueCells is the number of upvalue-cell pointers stored directly inside
	// runtimeClosure to avoid a separate make([]*upvalueCell, N) allocation per closure
	// creation. The vast majority of closures capture <= 4 upvalues; closures with more
	// allocate an external slice.
	//
	// Trade-off: extra struct size of 32 bytes per closure (4 pointers) against saving the
	// allocation and one GC scan barrier per closure creation when 1 <= N <= 4.
	inlineUpvalueCells = 4

	// snapshotChunkSize is the number of snapshotSliceHeader slots allocated per chunk of
	// vm.sliceSnapshotChunk.
	//
	// Larger values amortise mallocgc better but increase per-chunk minimum live memory
	// (since GC reclaims a whole chunk only after every slot in it is unreferenced). 4096
	// (96 KiB per chunk) suits multi-million snapshot workloads. The chunk grows lazily on
	// first acquire, so functions that never read a slice field pay zero extra memory.
	snapshotChunkSize = 4096

	// boundaryChunkSize is the slot count per per-type boundary-snapshot chunk.
	//
	// Matched to snapshotChunkSize for consistent alloc-count behaviour across
	// struct-snapshot and slice-snapshot paths: one reflect.MakeSlice allocation amortises
	// across boundaryChunkSize boundary copies. The chunk is per-type so a function that
	// allocates only a few struct kinds pays one chunk per kind on first acquire.
	boundaryChunkSize = 4096

	// maxArenaPreSizeBacking caps each per-bank backing total.
	//
	// Sized large enough to cover any realistic program (~64M entries per bank) while
	// remaining far below MaxInt so a pathological bytecode whose hint counts multiply past
	// 2^63 cannot wrap silently and trick EnsureCapacity into under-allocating. The arena's
	// lazy-grow path would absorb a too-small estimate; the cap exists strictly to keep the
	// multiplication well-defined.
	maxArenaPreSizeBacking = 1 << 26

	// maxFrameStackPreSize bounds the compile-time frame-stack pre-allocation. Programs that
	// genuinely recurse beyond this limit still grow lazily via growCallStack; the cap
	// exists to keep a hostile bytecode whose call graph fans out aggressively from
	// demanding gigabytes of frame storage up-front.
	maxFrameStackPreSize = 4096

	// maxBoxSlabPreSize bounds each of the five per-kind box slabs (int / uint / float /
	// string / complex). 65536 slots per kind covers realistic pack-heavy workloads while
	// leaving the cap well below the saturating product overflow point.
	maxBoxSlabPreSize = 1 << 16

	// maxUpvalueSlabPreSize bounds the upvalueCell and upvalueReference slabs.
	maxUpvalueSlabPreSize = 1 << 16

	// maxGenericBytesPreSize bounds the genericBytesSlab pre-size in bytes (4 MiB ceiling).
	maxGenericBytesPreSize = 4 << 20

	// maxSliceHeaderPreSize bounds the sliceHeaderSlab pre-size in slots (65536 slots ~= 1.5
	// MiB at 24 bytes per slot).
	maxSliceHeaderPreSize = 1 << 16

	// frameStackSafetyMargin is the bumper added on top of the static max call depth before
	// clamping to maxFrameStackPreSize.
	//
	// Absorbs analyses missing a small number of frames (e.g. host-side callbacks) without
	// forcing the dispatch loop to grow.
	frameStackSafetyMargin = 16

	// recursiveSCCFanout charges this many frames per member of a strongly-connected
	// component in the static call graph. Sized so direct/mutual recursion in typical
	// workloads (parseExpression, fib, JSON parser) lands inside the pre-sized frame stack
	// rather than tripping growCallStack.
	recursiveSCCFanout = 8

	// boxSlabPerOccurrenceFactor multiplies each opPackInterface or opPackTyped count to
	// derive the per-kind box slab budget. Conservative 4x absorbs amplification from loop
	// bodies.
	boxSlabPerOccurrenceFactor = 4

	// arenaMakeSliceGenericAvgBytes is the per-occurrence byte charge for opMakeSlice on the
	// general bank (slice-header + minimal element budget).
	arenaMakeSliceGenericAvgBytes = 32

	// arenaMakeMapAvgBytes is the per-occurrence byte charge for opMakeMap (covers the hmap
	// header allocation routed through the generic byte slab).
	arenaMakeMapAvgBytes = 48

	// arenaAllocIndirectAvgBytes is the per-occurrence byte charge for opAllocIndirect
	// (heap-promoted local storage; conservative estimate of a small struct).
	arenaAllocIndirectAvgBytes = 64
)

// VM is the piko bytecode interpreter.
//
// One VM corresponds to one invocation of a compiled program; it owns the call stack,
// register arena, defer state, panic state, and globals. A fresh VM is created per
// invocation via newVM; it is the register arena the VM borrows that is pooled and reset
// between runs, not the VM itself.
type VM struct {
	// ptrTypeCacheKey is the cached element type T for the one-slot LRU that bypasses
	// reflect.PointerTo's sync.Map lookup for the hot ADDR-chain pattern (`&Struct{}` in
	// tight constructors). A constructor that repeatedly allocates the same struct type hits
	// the cache on every allocation after the first.
	ptrTypeCacheKey reflect.Type

	// evalResult holds the return value from handleReturn when the base frame returns.
	// Extracted by the dispatch loop after opDone.
	evalResult any

	// panicValue holds the current panic value during unwinding.
	panicValue any

	// evalError holds the error from a handler that returns opPanicError. Extracted by the
	// dispatch loop after opPanicError.
	evalError error

	// ctx is the execution context for cancellation support.
	ctx context.Context

	// mapAccessCacheLastType is the most recently seen map reflect.Type passed through a
	// typed map handler.
	//
	// Paired with mapAccessCacheLastEntry to memoise the Key/Elem types so a hot map access
	// avoids repeated reflect.Type.Key/Elem calls. A function that repeatedly indexes the
	// same concrete map hits the cache after the first access.
	mapAccessCacheLastType reflect.Type

	// ptrTypeCacheValue holds *T paired with ptrTypeCacheKey.
	ptrTypeCacheValue reflect.Type

	// stderrWriter is the writer for print/println output, defaulting to os.Stderr but
	// overridable for testing.
	stderrWriter io.Writer

	// arena provides pooled register bank allocation. When set, pushFrame uses
	// arena.AllocRegisters instead of newRegisters, avoiding per-frame heap allocations.
	arena *RegisterArena

	// boundarySnapshotChunks caches per-type chunked slabs for boundary copies.
	//
	// Used by copyReflectValueArena for pointer-containing struct/array kinds (where the
	// byte-slab fast path cannot be used because GC needs typed scanning). Each chunk is a
	// reflect.MakeSlice([]T, N) allocation; per-snapshot cost is one slot bump-index plus a
	// Value.Index call (no heap alloc until the chunk fills). Saves a reflect.New per
	// snapshot. Keyed by *abi.Type so different types do not share chunks.
	boundarySnapshotChunks map[unsafe.Pointer]*boundaryChunk

	// pointerProvenance records the valid byte window of the allocation each interpreted
	// unsafe pointer was derived from, keyed by raw address.
	//
	// It is consulted only under the runtime safe mode (vm.limits.safeMode) to bounds-check
	// unsafe pointer arithmetic, and is nil and never allocated in fast mode. See
	// vm_safe_provenance.go.
	pointerProvenance map[uintptr]pointerBound

	// pointerProvenanceKeepAlive pins the pointees recorded in pointerProvenance so their
	// addresses remain valid for the whole execution (the map holds only uintptrs, invisible
	// to the GC). Safe mode only.
	pointerProvenanceKeepAlive []reflect.Value

	// symbols provides access to pre-registered native functions and values.
	symbols *SymbolRegistry

	// rootFunction is the top-level compiled function that owns the method table and
	// function table. Set during execute().
	rootFunction *CompiledFunction

	// asmCallInfoTables holds pre-computed asmCallInfo tables for each function, keyed by
	// *CompiledFunction. Built during execute() for ASM-inlined call/return.
	asmCallInfoTables map[*CompiledFunction][]asmCallInfo

	// debugHook is called at debug-relevant points when debugging is active. Nil when no
	// debugger is attached.
	debugHook DebugHook

	// methodCache maps (reflect.Type, method name) to the method index so that
	// handleGetMethod can use reflect.Value.Method(index) on repeat calls instead of the
	// allocating MethodByName lookup.
	methodCache map[methodCacheKey]int

	// typedHandleCache maps a chan/map pointer to its Go-typed handle.
	//
	// Keyed by reflect.Value.UnsafePointer; values are concrete handles (chan int64,
	// map[string]int64, etc.). The first typed send/recv/index against a given chan or map
	// extracts the handle once and stashes it here; subsequent ops dispatch via the cached
	// typed handle, bypassing reflect.Value.Send / Recv / MapIndex / SetMapIndex on the hot
	// path.
	typedHandleCache map[uintptr]any

	// mapKeyScratch caches an addressable reflect.Value per key type.
	//
	// Each entry is obtained from reflect.New(keyType).Elem(). The typed map fast paths
	// (opMapGetIntGeneral, opMapIndexOkIntGeneral and friends) reuse the scratch via .SetInt
	// / .SetString to avoid allocating a fresh int64/string wrapper per call. First call per
	// key type pays the reflect.New allocation; subsequent calls are free. Goroutine-safe
	// because VM is per-goroutine.
	mapKeyScratch map[reflect.Type]reflect.Value

	// debugActive is shared with the Debugger and set to 1 when debugging is active. Checked
	// via a cheap atomic load in the hot loop - same pattern as cancelled.
	debugActive *atomic.Uint32

	// nativeBackedErasurePointees holds valid erasure-boundary pointees.
	//
	// Lazily built from every registered NativeBackedGenericType sentinel's canonical erased
	// type plus its canonical erasure-argument types. reinterpretPointerArgument consults
	// the set so it reinterprets a pointer/pointer mismatch only across a genuine erasure
	// boundary, leaving an ordinary type mismatch unchanged. Nil until first use; an empty
	// (non-nil) map records "registry scanned, none found".
	nativeBackedErasurePointees map[reflect.Type]struct{}

	// internedMapKeys deduplicates strings used as map keys.
	//
	// Used by handleMapAddStringInt. Each unique key content is cloned once (via
	// strings.Clone) on first observation; subsequent inserts with the same content reuse
	// the cached copy. Eliminates the per-insert strings.Clone allocation. Lazily
	// initialised; goroutine-safe because piko's parallel goroutines each get their own VM
	// (runCompiledGoroutine). Grows monotonically with unique keys; memory cost is bounded
	// by the workload's distinct-key set.
	internedMapKeys map[string]string

	// stopWatcher deregisters the context.AfterFunc callback.
	//
	// Registered by newVM, the callback sets the cancelled flag when the execution context
	// is cancelled. finishWatcher calls it once execution returns to release the callback
	// promptly. Nil when no callback was registered because the parent context has no Done
	// channel. The AfterFunc stop func is safe to call more than once.
	stopWatcher func() bool

	// globals holds package-level variables shared across all functions in the program.
	globals *globalStore

	// debugState holds mutable debug state (breakpoints, stepping). Nil when no debugger is
	// attached.
	debugState *debugState

	// mapAccessCacheLastEntry holds the key/elem types and fast-path eligibility for
	// mapAccessCacheLastType. Read after a successful type-pointer equality check against
	// mapAccessCacheLastType; a miss recomputes and refreshes both fields.
	mapAccessCacheLastEntry mapAccessCacheEntry

	// modulePath identifies the loaded module owning the execution.
	//
	// Empty for main-program code; set to the module's canonical path when the VM is
	// executing inside a LoadModule'd bundle. Forwarded to every CapabilityHook consultation
	// so hosts can scope decisions per-module. Set at frame entry by dispatchers that cross
	// a module boundary; restored on return.
	modulePath string

	// closureCache caches reflect.Value wrappers for zero-upvalue closures.
	//
	// Holds immutable function values that never change. Indexed directly by funcIndex
	// (parallel to vm.functions) so lookup is a single slice load instead of a map hash.
	// Entries are zero-Value (IsValid()==false) until first cached. Grown lazily on first
	// makeClosure for a given funcIndex.
	closureCache []reflect.Value

	// deferStack holds deferred function calls. Each frame records its deferBase so that
	// only defers from the current frame are run when the frame returns.
	deferStack []deferredCall

	// selectCasesBuffer is a reusable buffer for handleSelect reflect cases.
	selectCasesBuffer []reflect.SelectCase

	// selectInfosBuffer is a reusable buffer for handleSelect case metadata.
	selectInfosBuffer []selectCaseInfo

	// functions is the program-level function table. All compiled functions are stored here,
	// referenced by index from CallSites.
	functions []*CompiledFunction

	// builtinArgumentsBuffer is a reusable buffer for handleCallBuiltin arguments. Safe
	// because VM is per-goroutine and builtins are synchronous.
	builtinArgumentsBuffer []any

	// tailCallArgBuffer is a reusable buffer for snapshotTailCallArgs.
	//
	// Every tail call captures argument values from caller registers into a
	// []tailCallArgument before reallocating the frame to the callee's layout. Reusing one
	// per-VM buffer is safe because tail-call processing is synchronous: the buffer is read
	// by placeTailCallArgs and then discarded before any subsequent dispatch can reach
	// snapshotTailCallArgs again. Grows lazily to the max argument count observed; never
	// shrinks (one-time cost per VM lifetime).
	tailCallArgBuffer []tailCallArgument

	// asmDispatchSaves is a parallel array to callStack, holding the dispatch register
	// values (codeBase, codeLength, intConstantsBase, floatConstantsBase) saved by the ASM
	// call handler for restoration by the return handler.
	asmDispatchSaves []asmDispatchSave

	// callStack holds the call frames. The current frame is at callStack[fp].
	callStack []callFrame

	// rootSnapshots runs parallel to callStack; entry i holds the dispatch state to restore
	// when frame i returns, or nil when no swap happened at push time.
	rootSnapshots []*frameRootSnapshot

	// sliceSnapshotChunk is a chunked slab of 24-byte slice-header buffers.
	//
	// Used by handleGetField to detach reference-kind field reads from their backing struct
	// storage. Each handleGetField slice-field read bump-allocates one slot. When the chunk
	// fills, a new chunk is allocated; old chunks stay alive as long as any slot in them is
	// referenced. Amortises the per-snapshot mallocgc cost across snapshotChunkSize
	// allocations per mallocgc call. Goroutine-safe because the VM is per-goroutine.
	sliceSnapshotChunk []snapshotSliceHeader

	// evalAllResults holds all return values from handleReturn when the base frame returns,
	// used by callClosureReflect to preserve multi-return values.
	evalAllResults []any

	// asmCallInfoBases is a parallel array to callStack, holding the asmCallInfo table base
	// pointer for the function at each frame. Used by ASM to switch tables on call/return.
	asmCallInfoBases []uintptr

	// limits holds resource constraints for DoS protection.
	limits vmLimits

	// inlineDispatchUintResult is the typed-fast-path return slot.
	//
	// Used by invokeInlineEvalFullFrame to receive the callee's uint result without boxing
	// through an `any` interface. handleReturn populates the slot (instead of vm.evalResult)
	// when inlineDispatchExpectUintResult is true. Eliminates the per-call allocation on the
	// inline dispatch path.
	inlineDispatchUintResult uint64

	// recoverEligibleFrame records the framePointer of the running deferred function.
	//
	// recover() only catches when the current framePointer == recoverEligibleFrame (i.e.
	// recover is called directly by the deferred function, not by a nested helper the
	// deferred function called). Set by executeDeferredCall on entry, restored on exit.
	// Sentinel -1 means no defer is active.
	recoverEligibleFrame int

	// sliceSnapshotNext is the bump index into the active sliceSnapshotChunk slab.
	sliceSnapshotNext int

	// costRemaining is the remaining computation cost budget.
	costRemaining int64

	// framePointer is the current frame pointer (index into callStack).
	framePointer int

	// baseFramePointer is the base frame pointer for the current run() invocation, used by
	// opReturn/opReturnVoid to detect when the base frame returns rather than hardcoding
	// zero.
	baseFramePointer int

	// yieldCounter counts instructions for Gosched() yielding.
	yieldCounter uint32

	// cancelled is set to 1 by a background goroutine when the execution context is done.
	// Checked via a cheap atomic load instead of the more expensive ctx.Err() in the hot
	// loop.
	cancelled atomic.Uint32

	// usesTypedSliceBanks reports whether typed-slice banks are reachable.
	//
	// True when any compiled function reachable by the VM uses a typed-slice or complex
	// register bank. Gates repairRegisterBasesFromCallers: when false, the cross-mask
	// stale-base repair (which only ever touches those banks) is a provable no-op, so its
	// O(depth) ancestor walk is skipped. Set monotonically (never cleared) whenever a
	// function table is adopted (prepareForExecution, swapToClosureRoot, child goroutine
	// VM), so it can never under-report a bank that is live on the stack, including across
	// the ASM-inline call path, which bypasses Go's frame push/pop.
	usesTypedSliceBanks bool

	// inlineDispatchExpectUintResult signals that the active vm.run invocation came from an
	// inline-dispatch path that expects a uint64 result. handleReturn checks this flag at
	// baseFramePointer to decide whether to populate inlineDispatchUintResult (fast path, no
	// alloc) or fall back to vm.evalResult (slow path, any boxing).
	inlineDispatchExpectUintResult bool

	// panicking is true when the VM is unwinding due to a panic.
	panicking bool

	// hasGoroutines is set when the first opGo is executed. When true, handleSetGlobal
	// clones arena-backed strings immediately instead of deferring to execute() cleanup,
	// because child goroutines share the global store and may outlive this arena.
	hasGoroutines bool

	// checkpointFlags is a bitset of pending checkpoint signals raised by the VM.
	//
	// Set by the VM's own goroutine (e.g. arena GC trigger, user-initiated runtime.GC()).
	// Checked at the existing cancellation safe-point in the dispatch loop every
	// cancellationCheckMask+1 instructions. Cross-goroutine signals (cancellation,
	// child-goroutine panic) remain on their atomic fields because they're set from other
	// goroutines.
	checkpointFlags uint8

	// holdsInterpreterLock is true while this VM holds its family's interpreter lock (the
	// per-family GIL) in safe mode.
	//
	// It makes acquire idempotent across the goroutine's nested dispatch calls and gates the
	// release-around and periodic-yield helpers. It is always false in fast mode.
	holdsInterpreterLock bool

	// reentrantInterpreterVM marks a fresh VM created to run an interpreted closure or
	// method invoked by native code.
	//
	// The native caller is a reflect.MakeFunc wrapper or a bound method value. Such a VM
	// must never acquire the interpreter lock: on the same goroutine it already runs under
	// the caller's held lock, and a native-goroutine invocation is the documented
	// native-concurrency residual. It is set at creation of those fresh VMs only.
	reentrantInterpreterVM bool
}

// updateASMCallInfoBase sets the asmCallInfoBases entry for the current frame after a
// Go-side frame change (push/pop). Grows the parallel arrays if the callStack has grown.
//
// Reads the pre-cached base directly off the CompiledFunction. The field is populated by
// buildASMCallInfoTables which only ever runs inside ensureASMCallInfoTables's sync.Once
// gate, so the write happens exactly once per root function and subsequent reads see the
// stable value.
func (vm *VM) updateASMCallInfoBase() {
	if vm.asmCallInfoBases == nil {
		return
	}
	if vm.framePointer >= len(vm.asmCallInfoBases) {
		vm.growCallStack()
	}
	if vm.framePointer >= 0 {
		vm.asmCallInfoBases[vm.framePointer] = vm.callStack[vm.framePointer].function.asmCallInfoBase
	}
}

// stderr returns the writer for print/println output.
//
// Returns the configured stderrWriter, or os.Stderr if none was set.
func (vm *VM) stderr() io.Writer {
	if vm.stderrWriter != nil {
		return vm.stderrWriter
	}
	return os.Stderr
}

// callDepthLimit returns the effective maximum call stack depth.
//
// Returns the configured maxCallDepth limit, or the default maxCallDepth constant if
// unset.
func (vm *VM) callDepthLimit() int {
	if vm.limits.maxCallDepth > 0 {
		return vm.limits.maxCallDepth
	}
	return maxCallDepth
}

// ensureGoroutineLimit returns the effective concurrent-goroutine cap and guarantees a
// resource tracker exists so the cap is enforceable.
//
// A directly-constructed VM, or one built with WithMaxGoroutines(0), has
// limits.maxGoroutines <= 0 and may have a nil tracker. Treating that as "unlimited" lets
// interpreted code spawn goroutines without bound, leaking host goroutines for every
// program that embeds piko. Instead a non-positive limit selects the defaultMaxGoroutines
// ceiling, and a missing tracker is lazily installed. The tracker is written back onto
// vm.limits before any goroutine launch copies the limits, so parent and child VMs share
// the same atomic counter.
//
// Returns the effective maximum number of concurrent goroutines.
func (vm *VM) ensureGoroutineLimit() int32 {
	if vm.limits.tracker == nil {
		vm.limits.tracker = &resourceTracker{}
	}
	if vm.limits.maxGoroutines > 0 {
		return vm.limits.maxGoroutines
	}
	return defaultMaxGoroutines
}

// stackOverflowError builds the error returned when call depth is exceeded.
//
// Includes the current depth and the effective limit so callers can tell whether they
// need to lift the bound (via WithMaxCallDepth) or whether the program is genuinely
// running away. The returned error wraps errStackOverflow so existing `errors.Is(err,
// errStackOverflow)` checks continue to work.
//
// Returns the wrapped stack-overflow error.
func (vm *VM) stackOverflowError() error {
	return fmt.Errorf("%w: call depth %d exceeded limit %d (raise via WithMaxCallDepth)",
		errStackOverflow, vm.framePointer, vm.callDepthLimit())
}

// guardCallDepth raises a stack-overflow panic when pushing another frame would exceed
// the effective call-depth limit.
//
// The compiled-call handlers (handleCall and friends) pre-check the depth and return
// opStackOverflow before pushing, but the three frame-push sites that bypass them
// (executeDeferredCall, callClosureReflect and pushFrame itself) would otherwise grow the
// call stack without bound and peg memory. This helper is the shared check for those
// sites. The panic is caught by handleRecoveredHandlerPanic at the dispatch boundary and
// surfaced as an interpreted stack-overflow panic, so interpreted defer/recover observes
// it the same way it would a runtime stack overflow.
//
// The panic carries an error wrapping errStackOverflow so that errors.Is(err,
// errStackOverflow) continues to match.
func (vm *VM) guardCallDepth() {
	if vm.framePointer >= vm.callDepthLimit() {
		panic(vm.stackOverflowError())
	}
}

// limitedStderr returns a writer that counts bytes written when an output size limit is
// configured.
//
// Returns a countingWriter wrapping stderr when limits are active, or the plain stderr
// writer otherwise.
func (vm *VM) limitedStderr() io.Writer {
	if vm.limits.maxOutputSize > 0 && vm.limits.tracker != nil {
		return &countingWriter{writer: vm.stderr(), tracker: vm.limits.tracker}
	}
	return vm.stderr()
}

// checkOutputLimit checks whether the output size limit has been exceeded and sets
// vm.evalError if so.
//
// Returns true when the limit has been exceeded, or false otherwise.
func (vm *VM) checkOutputLimit() bool {
	if vm.limits.maxOutputSize > 0 && vm.limits.tracker != nil {
		if vm.limits.tracker.outputBytes.Load() > int64(vm.limits.maxOutputSize) {
			vm.evalError = fmt.Errorf("%w: limit %d bytes", errOutputLimit, vm.limits.maxOutputSize)
			return true
		}
	}
	return false
}

// acquireArena returns a RegisterArena, using the custom factory if configured or the
// global pool otherwise.
//
// Returns a RegisterArena from the custom factory or the global pool.
func (vm *VM) acquireArena() *RegisterArena {
	var arena *RegisterArena
	if vm.limits.arenaFactory != nil {
		arena = vm.limits.arenaFactory()
	} else {
		arena = GetRegisterArena()
	}
	arena.maxArenaBytes = vm.limits.maxArenaBytes
	return arena
}

// ensureCallStack sets up an arena and call stack for cold-path VMs (varinit, init) that
// were not created via execute(). The caller must defer vm.releaseArena() to return the
// arena to the pool.
func (vm *VM) ensureCallStack() {
	if vm.arena == nil {
		vm.arena = vm.acquireArena()
	}
	if vm.callStack == nil {
		vm.callStack = vm.arena.frameStack()
	}
	if cap(vm.rootSnapshots) < len(vm.callStack) {
		vm.rootSnapshots = make([]*frameRootSnapshot, len(vm.callStack))
	} else {
		vm.rootSnapshots = vm.rootSnapshots[:len(vm.callStack)]
	}
}

// initialiseASMDispatch allocates the parallel ASM dispatch arrays (callInfoBases,
// dispatchSaves) from the arena. This must be called after ensureCallStack and after
// asmCallInfoTables has been set.
func (vm *VM) initialiseASMDispatch() {
	if vm.arena == nil {
		return
	}
	vm.asmCallInfoBases = vm.arena.CallInfoBases()
	vm.asmDispatchSaves = vm.arena.dispatchSaves()
}

// releaseArena returns the VM's arena to the global pool and clears the reference.
func (vm *VM) releaseArena() {
	if vm.arena != nil {
		vm.callStack = nil
		PutRegisterArena(vm.arena)
		vm.arena = nil
	}
}

// execute runs a compiled function and returns its result.
//
// Takes compiledFunction (*CompiledFunction) which is the function to execute.
//
// Returns any which is the result of the function.
// Returns error when execution fails.
//
// Panics when a non-budget recover surfaces a runtime panic that the deferred handler
// re-raises.
func (vm *VM) execute(compiledFunction *CompiledFunction) (result any, err error) {
	if vm.limits.safeMode {
		vm.resetPointerProvenance()
	}
	defer vm.finishWatcher()
	ownArena := false
	defer func() {
		if recovered := recover(); recovered != nil {
			if budgetErr, ok := recovered.(error); ok && errors.Is(budgetErr, errArenaBudgetExceeded) {
				if ownArena {
					vm.releaseOwnedArena()
				}
				result = nil
				err = budgetErr
				return
			}
			panic(recovered)
		}
	}()
	ownArena = vm.prepareForExecution(compiledFunction)

	if err := vm.runVariableInit(compiledFunction, ownArena); err != nil {
		return nil, err
	}

	vm.pushFrame(compiledFunction)
	result, err = vm.runDispatchedGuarded(0)

	if ownArena {
		result = vm.finaliseOwnedArena(result)
	} else if vm.arena != nil {
		vm.arena.ownerVM = nil
	}

	return result, err
}

// prepareForExecution wires the VM's execution-scoped state from compiledFunction and
// acquires an arena and call stack when none has been supplied externally.
//
// Takes compiledFunction (*CompiledFunction) which is the function about to execute.
//
// Returns true when execute owns the arena it just acquired and must release it on
// return.
func (vm *VM) prepareForExecution(compiledFunction *CompiledFunction) bool {
	vm.functions = compiledFunction.functions
	vm.rootFunction = compiledFunction
	vm.usesTypedSliceBanks = vm.usesTypedSliceBanks || bundleUsesTypedSliceBanks(compiledFunction)
	vm.costRemaining = vm.limits.costBudget

	ownArena := vm.arena == nil
	if ownArena {
		vm.arena = vm.acquireArena()
		vm.sizeArenaFromFunctions(compiledFunction)
	}
	vm.arena.ownerVM = vm

	if vm.callStack == nil {
		vm.callStack = vm.arena.frameStack()
	}

	vm.asmCallInfoTables = ensureASMCallInfoTables(compiledFunction)
	vm.asmCallInfoBases = vm.arena.CallInfoBases()
	vm.asmDispatchSaves = vm.arena.dispatchSaves()
	if table := vm.asmCallInfoTables[compiledFunction]; len(table) > 0 {
		vm.asmCallInfoBases[0] = uintptr(unsafe.Pointer(&table[0]))
	}
	return ownArena
}

// runVariableInit executes the function's variable-initialiser body when present,
// releasing the arena on failure.
//
// Takes compiledFunction (*CompiledFunction) whose variableInitFunction is consulted.
// Takes ownArena (bool) which indicates whether execute owns the arena and must release
// it on error.
//
// Returns the wrapped variable-init error or nil when no initialiser is configured.
func (vm *VM) runVariableInit(compiledFunction *CompiledFunction, ownArena bool) error {
	if compiledFunction.variableInitFunction == nil {
		return nil
	}
	vm.pushFrame(compiledFunction.variableInitFunction)
	if _, err := vm.runGuarded(0); err != nil {
		if ownArena {
			vm.releaseOwnedArena()
		}
		return fmt.Errorf("varinit: %w", err)
	}
	return nil
}

// finaliseOwnedArena materialises the dispatched result and any pending Eval-all results
// onto the heap before releasing the execute-owned arena.
//
// Takes result (any) which is the value returned by runDispatched.
//
// Returns the heap-materialised result.
func (vm *VM) finaliseOwnedArena(result any) any {
	if !vm.hasGoroutines {
		vm.globals.materialiseStrings(vm.arena)
	}
	result = materialiseAnyForArena(vm.arena, result)
	for index, allResult := range vm.evalAllResults {
		vm.evalAllResults[index] = materialiseAnyForArena(vm.arena, allResult)
	}
	vm.releaseOwnedArena()
	return result
}

// releaseOwnedArena tears down the dispatch-context pointers that reference the
// per-execution arena and returns the arena to the pool. Factored out so execute() stays
// under the function-length limit and so both the varinit error path and the successful
// return path share an identical sequence.
func (vm *VM) releaseOwnedArena() {
	vm.callStack = nil
	vm.asmCallInfoTables = nil
	vm.asmCallInfoBases = nil
	vm.asmDispatchSaves = nil
	vm.arena.ownerVM = nil
	PutRegisterArena(vm.arena)
	vm.arena = nil
}

// sizeArenaFromFunctions inspects the compiled function table to pre-size the arena so
// that AllocRegisters never triggers a grow during normal execution.
//
// With stack-based reclamation (popFrame restores the arena), only max-depth x
// max-registers matters, not total calls x registers. This makes the estimate tight even
// for deeply recursive functions like fib(20).
//
// Takes root (*CompiledFunction) which specifies the top-level compiled function whose
// function table is used to compute the arena capacity.
func (vm *VM) sizeArenaFromFunctions(root *CompiledFunction) {
	var totals typedSlabCounts
	var bytecodeHints arenaBytecodeHints

	allFuncs := append([]*CompiledFunction{root}, root.functions...)
	if root.variableInitFunction != nil {
		allFuncs = append(allFuncs, root.variableInitFunction)
	}
	for _, f := range allFuncs {
		totals.ints += int(f.numRegisters[registerInt])
		totals.floats += int(f.numRegisters[registerFloat])
		totals.strings += int(f.numRegisters[registerString])
		totals.generals += int(f.numRegisters[registerGeneral])
		totals.bools += int(f.numRegisters[registerBool])
		totals.uints += int(f.numRegisters[registerUint])
		totals.complexes += int(f.numRegisters[registerComplex])
		totals.slicesInts += int(f.numRegisters[registerSliceInt])
		totals.slicesFloats += int(f.numRegisters[registerSliceFloat])
		totals.slicesStrings += int(f.numRegisters[registerSliceString])
		totals.slicesBools += int(f.numRegisters[registerSliceBool])
		totals.slicesUints += int(f.numRegisters[registerSliceUint])

		accumulateArenaBytecodeHints(&bytecodeHints, f.body, allFuncs)
	}

	const depthEstimate = 64
	totals.ints *= depthEstimate
	totals.floats *= depthEstimate
	totals.strings *= depthEstimate
	totals.generals *= depthEstimate
	totals.bools *= depthEstimate
	totals.uints *= depthEstimate
	totals.complexes *= depthEstimate
	totals.slicesInts *= depthEstimate
	totals.slicesFloats *= depthEstimate
	totals.slicesStrings *= depthEstimate
	totals.slicesBools *= depthEstimate
	totals.slicesUints *= depthEstimate

	totals.bytes = saturatingProduct(bytecodeHints.bytes, depthEstimate)
	totals.intBacking = saturatingProduct3(bytecodeHints.makeSliceInt, arenaMakeSliceAvgCapacity, depthEstimate)
	totals.floatBacking = saturatingProduct3(bytecodeHints.makeSliceFloat, arenaMakeSliceAvgCapacity, depthEstimate)
	totals.stringBacking = saturatingProduct3(bytecodeHints.makeSliceString, arenaMakeSliceAvgCapacity, depthEstimate)
	totals.boolBacking = saturatingProduct3(bytecodeHints.makeSliceBool, arenaMakeSliceAvgCapacity, depthEstimate)
	totals.uintBacking = saturatingProduct3(bytecodeHints.makeSliceUint, arenaMakeSliceAvgCapacity, depthEstimate)
	totals.genericBytes = clampAtMost(saturatingProduct(bytecodeHints.genericBytes, depthEstimate), maxGenericBytesPreSize)
	totals.sliceHeaders = clampAtMost(saturatingProduct(bytecodeHints.sliceHeaders, depthEstimate), maxSliceHeaderPreSize)

	maxDepth := estimateMaxCallDepth(root)
	totals.frameStack = clampAtMost(maxDepth+frameStackSafetyMargin, maxFrameStackPreSize)

	totals.intBoxes = clampAtMost(saturatingProduct(bytecodeHints.boxInts, boxSlabPerOccurrenceFactor), maxBoxSlabPreSize)
	totals.uintBoxes = clampAtMost(saturatingProduct(bytecodeHints.boxUints, boxSlabPerOccurrenceFactor), maxBoxSlabPreSize)
	totals.floatBoxes = clampAtMost(saturatingProduct(bytecodeHints.boxFloats, boxSlabPerOccurrenceFactor), maxBoxSlabPreSize)
	totals.stringBoxes = clampAtMost(saturatingProduct(bytecodeHints.boxStrings, boxSlabPerOccurrenceFactor), maxBoxSlabPreSize)
	totals.complexBoxes = clampAtMost(saturatingProduct(bytecodeHints.boxComplexes, boxSlabPerOccurrenceFactor), maxBoxSlabPreSize)

	totals.upvalueCells = clampAtMost(saturatingProduct(bytecodeHints.upvalueCells, depthEstimate), maxUpvalueSlabPreSize)
	totals.upvalueReferences = clampAtMost(saturatingProduct(bytecodeHints.upvalueReferences, depthEstimate), maxUpvalueSlabPreSize)

	vm.arena.EnsureCapacity(totals)
}

// arenaBytecodeHints counts byteSlab-writing and MakeSlice opcodes.
//
// The counts drive arena pre-sizing at warmup: rather than letting the runtime grow the
// byte slab and the typed-backing slabs lazily (each grow performs mallocgc and opens a
// GC scan window during ASM dispatch), the arena is pre-sized to the program's known
// worst case.
type arenaBytecodeHints struct {
	// bytes is the cumulative byte budget for byteSlab writers.
	bytes int

	// makeSliceInt counts subOpMakeSliceInt occurrences.
	makeSliceInt int

	// makeSliceFloat counts subOpMakeSliceFloat occurrences.
	makeSliceFloat int

	// makeSliceString counts subOpMakeSliceString occurrences.
	makeSliceString int

	// makeSliceBool counts subOpMakeSliceBool occurrences.
	makeSliceBool int

	// makeSliceUint counts subOpMakeSliceUint occurrences.
	makeSliceUint int

	// genericBytes is the byte budget for the composite snapshot slab.
	//
	// Each opcode that snapshots a pointer-free struct/array (composite literals, struct
	// field reads, slice snapshots into reflect.Value) contributes an estimated byte cost.
	// Worst case is a small composite (~32-64 bytes) plus padding; we charge a conservative
	// average so the slab is pre-grown to minimum doubled capacity for the program's
	// expected working set.
	genericBytes int

	// sliceHeaders is the cumulative slot budget for arena-routed reflect.Value slice
	// headers. Each opcode that wraps a typed slice into the general bank
	// (opSubOpBoxSliceByte's arena form, pack-typed-slice paths, append fast-paths, slice
	// snapshots) consumes one slot.
	sliceHeaders int

	// boxInts counts opPackInterface/opPackTyped sites whose source register kind feeds the
	// int box slab. Multiplied by boxSlabPerOccurrenceFactor inside sizeArenaFromFunctions.
	boxInts int

	// boxUints counts pack sites feeding the uint box slab.
	boxUints int

	// boxFloats counts pack sites feeding the float box slab.
	boxFloats int

	// boxStrings counts pack sites feeding the string box slab.
	boxStrings int

	// boxComplexes counts pack sites feeding the complex box slab.
	boxComplexes int

	// upvalueCells is the cumulative descriptor count summed across every opMakeClosure
	// occurrence, looked up via the instruction's function-index operand and the callee's
	// upvalueDescriptors.
	upvalueCells int

	// upvalueReferences mirrors upvalueCells under the current ABI; kept separate so future
	// shape changes (e.g. cell vs reference fanout) do not require a hint-struct churn.
	upvalueReferences int

	// appendOccurrences counts opAppend / opAppendSpread / opAppendInPlace /
	// opAppendSpreadInPlace instructions.
	appendOccurrences int

	// makeSliceGeneric counts opMakeSlice instructions targeting the general bank (one
	// slice-header slot per occurrence).
	makeSliceGeneric int

	// makeMapOccurrences counts opMakeMap instructions.
	makeMapOccurrences int

	// allocIndirectOccurrences counts opAllocIndirect instructions (each charges one
	// slice-header slot plus an estimated bytes budget on the generic byte slab).
	allocIndirectOccurrences int
}

// growCallStack doubles the call stack capacity, growing the arena slabs or independent
// arrays to keep the parallel arrays (frames, ciBases, dispSaves) in sync.
//
// When an arena is available, the arena's slabs are grown so all three parallel arrays
// stay in sync. Without an arena, the callStack and parallel arrays are grown
// independently.
//
//go:noinline
func (vm *VM) growCallStack() {
	newCap := len(vm.callStack) * 2
	if vm.arena != nil {
		frames, ci, disp := vm.arena.growFrameStack(newCap)
		vm.callStack = frames
		vm.asmCallInfoBases = ci
		vm.asmDispatchSaves = disp
	} else {
		newStack := make([]callFrame, newCap)
		copy(newStack, vm.callStack)
		vm.callStack = newStack
		if vm.asmCallInfoBases != nil {
			newCI := make([]uintptr, newCap)
			copy(newCI, vm.asmCallInfoBases)
			vm.asmCallInfoBases = newCI
			newDisp := make([]asmDispatchSave, newCap)
			copy(newDisp, vm.asmDispatchSaves)
			vm.asmDispatchSaves = newDisp
		}
	}
	if cap(vm.rootSnapshots) < len(vm.callStack) {
		grown := make([]*frameRootSnapshot, len(vm.callStack))
		copy(grown, vm.rootSnapshots)
		vm.rootSnapshots = grown
	} else {
		vm.rootSnapshots = vm.rootSnapshots[:len(vm.callStack)]
	}
}

// pushFrame pushes a new call frame for the given function.
//
// Takes compiledFunction (*CompiledFunction) which specifies the compiled function to
// create a frame for.
func (vm *VM) pushFrame(compiledFunction *CompiledFunction) {
	vm.guardCallDepth()
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	f := &vm.callStack[vm.framePointer]
	if vm.arena != nil {
		if depth := vm.framePointer + 1; depth > vm.arena.framesUsed {
			vm.arena.framesUsed = depth
		}
		vm.arena.SaveInto(&f.arenaSave)
		compiledFunction.ensurePrecomputedAllocCounts()
		vm.arena.AllocRegistersIntoCached(&f.registers, compiledFunction.precomputedAllocCounts, compiledFunction.nonZeroBankMask)
	} else {
		f.arenaSave = ArenaSavePoint{}
		f.registers = newRegisters(compiledFunction.numRegisters)
	}
	f.function = compiledFunction
	f.programCounter = 0
	f.deferBase = len(vm.deferStack)
	if f.simpleDefer != nil {
		f.simpleDefer.active = false
	}
	f.upvalues = nil
	f.returnDestination = nil
	releaseSharedCellMap(f.sharedCells)
	f.sharedCells = nil
}

// popFrame pops the current call frame and restores the previous one. If the arena is in
// use, restores it to the save point recorded when the frame was pushed, reclaiming the
// frame's register slots.
func (vm *VM) popFrame() {
	frame := &vm.callStack[vm.framePointer]
	if vm.framePointer < len(vm.rootSnapshots) {
		if snapshot := vm.rootSnapshots[vm.framePointer]; snapshot != nil {
			vm.functions = snapshot.functions
			vm.rootFunction = snapshot.rootFunction
			vm.rootSnapshots[vm.framePointer] = nil
		}
	}
	if vm.arena != nil {
		vm.arena.Restore(frame.arenaSave)
	}
	vm.framePointer--
}

// currentFrame returns a pointer to the current call frame.
//
// Returns the callFrame at the current frame pointer position.
func (vm *VM) currentFrame() *callFrame {
	return &vm.callStack[vm.framePointer]
}

// extractResult extracts the return value from the final frame.
//
// Takes frame (*callFrame) which specifies the call frame to extract the result from.
//
// Returns the result value from the first result register, or nil if no results are
// declared.
func (vm *VM) extractResult(frame *callFrame) (any, error) {
	if len(frame.function.resultKinds) == 0 {
		return nil, nil
	}
	return vm.extractRegisterValue(&frame.registers, frame.function.resultKinds[0], 0), nil
}

// extractRegisterValue reads the boxed value held in the given register bank at the
// supplied index, returning nil when the index is past the bank's populated length or the
// general-bank slot holds an invalid reflect.Value. String banks are materialised against
// the VM arena so the returned value outlives frame teardown.
//
// Takes registers (*Registers) which is the frame's typed register file.
// Takes kind (registerKind) which selects the bank to read from.
// Takes index (int) which is the register slot within that bank.
//
// Returns the boxed register value, or nil when no value is present.
func (vm *VM) extractRegisterValue(registers *Registers, kind registerKind, index int) any {
	switch kind {
	case registerString:
		return extractStringRegisterValue(vm.arena, registers, index)
	case registerGeneral:
		return extractGeneralRegisterValue(registers, index)
	default:
		return extractScalarRegisterValue(registers, kind, index)
	}
}

// extractAllResults extracts all return values from the final frame. Used by
// callClosureReflect to preserve multi-return values.
//
// Takes frame (*callFrame) which specifies the call frame to extract results from.
//
// Returns a slice of all result values.
func (vm *VM) extractAllResults(frame *callFrame) []any {
	resultKinds := frame.function.resultKinds
	if len(resultKinds) == 0 {
		return nil
	}

	results := make([]any, len(resultKinds))
	var bankCounters [NumRegisterKinds]uint8
	registers := &frame.registers

	for i, kind := range resultKinds {
		sourceRegister := int(bankCounters[kind])
		bankCounters[kind]++
		results[i] = vm.extractRegisterValue(registers, kind, sourceRegister)
	}

	return results
}

// copyReturnValueAt copies a return value from the callee frame at the given source
// register to the caller frame at the destination.
//
// Takes calleeFrame (*callFrame) which specifies the frame containing the source value.
// Takes kind (registerKind) which specifies the register bank of the source value.
// Takes sourceRegister (uint8) which specifies the source register index within the
// callee.
// Takes dest (varLocation) which specifies the destination location in the caller frame.
func (vm *VM) copyReturnValueAt(calleeFrame *callFrame, kind registerKind, sourceRegister uint8, dest varLocation) {
	callerFrame := &vm.callStack[vm.framePointer-1]
	if kind == dest.kind {
		copySameKind(&callerFrame.registers, &calleeFrame.registers, kind, dest.register, sourceRegister)
	} else if kind == registerGeneral && dest.kind != registerGeneral {
		copyReturnFromGeneral(callerFrame, calleeFrame.registers.general[sourceRegister], dest)
	} else if dest.kind == registerGeneral && kind != registerGeneral {
		copyReturnToGeneral(callerFrame, &calleeFrame.registers, kind, sourceRegister, dest.register)
	}
}

// callClosureReflect executes a compiled closure via reflect, pushing a new frame,
// assigning arguments, and running the function to completion.
//
// Takes closure (*runtimeClosure) which specifies the closure to call.
// Takes arguments ([]reflect.Value) which provides the reflect.Value arguments to pass.
// Takes funcType (reflect.Type) which defines the expected function signature for result
// packaging.
//
// Returns the result values packaged as reflect.Values matching funcType's output
// signature.
func (vm *VM) callClosureReflect(closure *runtimeClosure, arguments []reflect.Value, funcType reflect.Type) []reflect.Value {
	callee := closure.function

	snapshot := vm.swapToClosureRoot(closure.rootFunction)

	vm.guardCallDepth()
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	closureFp := vm.framePointer
	f := &vm.callStack[vm.framePointer]
	if vm.arena != nil {
		vm.arena.SaveInto(&f.arenaSave)
		callee.ensurePrecomputedAllocCounts()
		vm.arena.AllocRegistersIntoCached(&f.registers, callee.precomputedAllocCounts, callee.nonZeroBankMask)
	} else {
		f.registers = newRegisters(callee.numRegisters)
	}
	f.function = callee
	f.programCounter = 0
	f.returnDestination = nil
	f.deferBase = len(vm.deferStack)
	if f.simpleDefer != nil {
		f.simpleDefer.active = false
	}
	f.upvalues = nil
	f.hasGeneralAlloc = callee.numRegisters[registerGeneral] > 0
	releaseSharedCellMap(f.sharedCells)
	f.sharedCells = nil
	vm.recordFrameSnapshot(closureFp, snapshot)
	if closure.upvalues != nil {
		f.initialiseUpvalues(closure.upvalues, vm.arena)
	}
	vm.updateASMCallInfoBase()

	assignReflectParams(&f.registers, callee.parameterKinds, arguments)

	_, err := vm.runDispatchedGuarded(closureFp)
	if err != nil {
		vm.evalError = err
	}

	allResults := vm.evalAllResults
	vm.evalAllResults = nil
	return buildReflectResults(allResults, funcType)
}

// runtimeClosure binds a compiled function to its captured upvalue cells.
//
// Up to inlineUpvalueCells captures live directly in inlineCells to avoid a separate
// slice allocation; the upvalues slice points at inlineCells for the inline case or at an
// external slice for closures with more captures. rootFunction is retained so
// callClosureReflect can swap the dispatch root when the closure runs.
type runtimeClosure struct {
	// inlineCells stores the first inlineUpvalueCells captures inline to avoid a separate
	// make([]*upvalueCell, N) per closure.
	inlineCells [inlineUpvalueCells]*upvalueCell

	// function is the compiled body this closure invokes.
	function *CompiledFunction

	// rootFunction is the top-level function whose dispatch context the closure belongs to;
	// restored when the closure is reflectively invoked from outside its defining program.
	rootFunction *CompiledFunction

	// upvalues references the captured cells; aliases inlineCells when the count fits,
	// otherwise points at an external slice.
	upvalues []*upvalueCell
}

// rangeIterator holds the state for a for-range loop over a slice, array, map, or
// channel.
type rangeIterator struct {
	// mapIterator drives map ranges; nil for non-map collections.
	mapIterator *reflect.MapIter

	// collection is the reflect-typed source for general-kind ranges.
	collection reflect.Value

	// valueScratch is the reusable value destination for map ranges.
	valueScratch reflect.Value

	// keyScratch is the reusable key destination for map ranges.
	keyScratch reflect.Value

	// stringSource is the source string when ranging over a string; indexed by index to
	// yield (rune, byte-offset) pairs.
	stringSource string

	// boolSlice is the source when ranging over a []bool fast path.
	boolSlice []bool

	// floatSlice is the source when ranging over a []float64 fast path.
	floatSlice []float64

	// stringSlice is the source when ranging over a []string fast path.
	stringSlice []string

	// intSlice is the source when ranging over a []int fast path.
	intSlice []int

	// index is the next slot to read; bumped on each Next.
	index int

	// isMap reports whether the iterator drives mapIterator.
	isMap bool

	// isChannel reports whether the iterator drives a channel recv.
	isChannel bool

	// isString reports whether the iterator drives a string range.
	isString bool
}

// writeRangeValue writes a reflect.Value to the appropriate register.
//
// Takes registers (*Registers) which specifies the register banks to write into.
// Takes value (reflect.Value) which specifies the reflect.Value to store.
// Takes register (uint8) which specifies the register index within the bank.
// Takes kind (registerKind) which specifies which register bank to target.
func (*VM) writeRangeValue(registers *Registers, value reflect.Value, register uint8, kind registerKind) {
	if kind != registerGeneral {
		value = unwrapInterfaceElement(value)
	}
	switch kind {
	case registerInt:
		switch value.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			registers.ints[register] = int64(value.Uint()) //nolint:gosec
		default:
			registers.ints[register] = value.Int()
		}
	case registerFloat:
		registers.floats[register] = value.Float()
	case registerString:
		registers.strings[register] = value.String()
	case registerGeneral:
		registers.general[register] = valueCopyForBoundary(value)
	case registerBool:
		registers.bools[register] = value.Bool()
	case registerUint:
		registers.uints[register] = value.Uint()
	case registerComplex:
		registers.complex[register] = value.Complex()
	default:
	}
}

// syncNamedResults syncs upvalue cells back to registers for captured named return
// variables, then re-copies from named result registers to return positions for Go spec
// compliance.
//
// Takes frame (*callFrame) which specifies the call frame whose named results are synced.
func (*VM) syncNamedResults(frame *callFrame) {
	namedLocations := frame.function.namedResultLocations
	if len(namedLocations) == 0 {
		return
	}

	if frame.sharedCells != nil {
		for _, location := range namedLocations {
			if location.isIndirect {
				continue
			}
			key := joinWide(uint8(location.kind), location.register)
			if cell, ok := frame.sharedCells.get(key); ok {
				syncCellToRegister(&frame.registers, cell, location)
			}
		}
	}

	var bankCounters [NumRegisterKinds]uint8
	for _, location := range namedLocations {
		if location.isIndirect {
			syncIndirectNamedResult(&frame.registers, location, &bankCounters)
			continue
		}
		destinationRegister := bankCounters[location.kind]
		bankCounters[location.kind]++
		if destinationRegister != location.register {
			copyRegisterSlot(&frame.registers, location.kind, destinationRegister, location.register)
		}
	}
}

// finishWatcher deregisters the context.AfterFunc callback.
//
// Registered by newVM once execution returns. AfterFunc otherwise retains the callback
// until the parent context is cancelled; deregistering frees it promptly. There is no
// watcher goroutine, so missing the call only delays the callback release rather than
// leaking a goroutine.
//
// Safe to call more than once and on a VM with no callback: a nil stopWatcher is ignored
// and the AfterFunc stop func is itself idempotent.
func (vm *VM) finishWatcher() {
	if vm.stopWatcher != nil {
		vm.stopWatcher()
	}
}

// shouldStopDebug returns true when the debug hook requests execution to stop.
//
// Takes frame (*callFrame) which is the current call frame to inspect for breakpoints.
//
// Returns bool which is true when the hook requests a stop, or false when debugging is
// inactive or the hook returns any action other than stop.
func (vm *VM) shouldStopDebug(frame *callFrame) bool {
	if vm.debugActive == nil || vm.debugActive.Load() == 0 {
		return false
	}
	return vm.checkDebug(frame) == DebugActionStop
}

// checkDebug evaluates breakpoints and stepping conditions at the current program
// counter.
//
// Takes frame (*callFrame) which specifies the current call frame.
//
// Returns DebugAction indicating whether to continue or stop.
func (vm *VM) checkDebug(frame *callFrame) DebugAction {
	if vm.debugHook == nil {
		return DebugActionContinue
	}

	pc := frame.programCounter
	if pc >= len(frame.function.body) {
		if vm.debugState == nil || vm.debugState.stepping != stepModeOut {
			return DebugActionContinue
		}
	}

	ctx := DebugContext{
		Function:       frame.function,
		ProgramCounter: pc,
		FramePointer:   vm.framePointer,
	}

	return vm.debugHook(ctx)
}

// boundaryChunk holds a pre-allocated slice of one Go type used as the backing storage
// for boundary-arena struct/array snapshots that can't go through the byte slab (because
// GC needs typed scanning). nextIdx is the next available slot; reaches boundaryChunkSize
// then triggers a fresh chunk allocation.
type boundaryChunk struct {
	// slab is the reflect.MakeSlice([]T, boundaryChunkSize) backing the chunk's slots;
	// element type T is determined at chunk creation.
	slab reflect.Value

	// nextIdx is the next free slot index inside slab; on reaching boundaryChunkSize the
	// next acquire allocates a fresh chunk.
	nextIdx int
}

// acquireSliceSnapshot bump-allocates the next slice-header buffer.
//
// When an arena is attached (the production hot path), the slot is drawn from
// arena.sliceHeaderSlab, the same per-VM bump allocator that backs reflect.Value
// snapshots for appendGenericFastPath etc. arena.Reset zeroes the slab between Execute
// calls, so escaped snapshots are detached through materialiseArenaValue at execute()
// exit.
//
// When no arena is attached (synthetic tests, debugger entry points) the fallback path
// uses the per-VM chunked slab so callers without an arena keep working unchanged.
//
// arenaSliceHeader and snapshotSliceHeader share the {Data unsafe.Pointer, Len int, Cap
// int} layout; the pointer cast at the arena path is safe because Go fixes struct layout
// per element type and the two types are byte-identical.
//
// Returns a pointer to the freshly bump-allocated slot.
//
//go:nosplit
func (vm *VM) acquireSliceSnapshot() *snapshotSliceHeader {
	if vm.arena != nil {
		slot := vm.arena.AllocSliceHeader()
		return (*snapshotSliceHeader)(unsafe.Pointer(slot))
	}
	if vm.sliceSnapshotNext >= len(vm.sliceSnapshotChunk) {
		vm.sliceSnapshotChunk = make([]snapshotSliceHeader, snapshotChunkSize)
		vm.sliceSnapshotNext = 0
	}
	slot := &vm.sliceSnapshotChunk[vm.sliceSnapshotNext]
	vm.sliceSnapshotNext++
	return slot
}

// acquireBoundarySnapshot returns a fresh addressable reflect.Value of type t.
//
// Suitable for use as the destination of a Set() call by the boundary copy machinery.
// Backed by a per-type chunked slab: 1 mallocgc per chunk (boundaryChunkSize values
// worth), bump-index per snapshot. The returned Value is GC-rooted via the chunk's slice
// backing array - pointer fields inside t are traced because the slab's element type is
// exactly t.
//
// Takes t (reflect.Type) which is the element type of the desired addressable Value;
// selects (and lazily allocates) the per-type chunk.
//
// Returns an addressable reflect.Value of type t backed by the chunk slab.
func (vm *VM) acquireBoundarySnapshot(t reflect.Type) reflect.Value {
	abiTyp := reflectValueABIType(t)
	chunk, ok := vm.boundarySnapshotChunks[abiTyp]
	if !ok || chunk.nextIdx >= chunk.slab.Len() {
		chunk = &boundaryChunk{
			slab:    reflect.MakeSlice(reflect.SliceOf(t), boundaryChunkSize, boundaryChunkSize),
			nextIdx: 0,
		}
		if vm.boundarySnapshotChunks == nil {
			vm.boundarySnapshotChunks = make(map[unsafe.Pointer]*boundaryChunk)
		}
		vm.boundarySnapshotChunks[abiTyp] = chunk
	}
	slot := chunk.slab.Index(chunk.nextIdx)
	chunk.nextIdx++
	return slot
}

// bundleUsesTypedSliceBanks reports whether a bundle uses typed banks.
//
// Reads only the immutable nonZeroBankMask fields (set at compile time), so it is safe to
// call concurrently from per-goroutine VMs that share a function table.
// O(len(functions)); called only when a function table is adopted, not on the hot
// dispatch path.
//
// Takes root (*CompiledFunction) which heads the bundle to inspect.
//
// Returns bool which is true when root or any sibling function uses a typed-slice or
// complex register bank.
func bundleUsesTypedSliceBanks(root *CompiledFunction) bool {
	if root == nil {
		return false
	}
	if root.nonZeroBankMask&typedSliceBankMask != 0 {
		return true
	}
	for _, fn := range root.functions {
		if fn != nil && fn.nonZeroBankMask&typedSliceBankMask != 0 {
			return true
		}
	}
	return false
}

// saturatingProduct returns a*b clamped at maxArenaPreSizeBacking. Both inputs are
// non-negative bytecode counts; negative values from pathological inputs are clamped to
// zero.
//
// Takes a, b (int) which are the multiplicands.
//
// Returns the saturated product.
func saturatingProduct(a, b int) int {
	if a <= 0 || b <= 0 {
		return 0
	}
	if a > maxArenaPreSizeBacking/b {
		return maxArenaPreSizeBacking
	}
	return a * b
}

// saturatingProduct3 returns a*b*c clamped at maxArenaPreSizeBacking. Computed in two
// saturating steps so neither intermediate can wrap.
//
// Takes a, b, c (int) which are the multiplicands.
//
// Returns the saturated product.
func saturatingProduct3(a, b, c int) int {
	return saturatingProduct(saturatingProduct(a, b), c)
}

// clampAtMost returns the smaller of value and ceiling. Used by the arena pre-sizing
// pipeline to enforce per-slab maxima on top of the program-wide maxArenaPreSizeBacking,
// so a hostile bytecode whose hint counts pass saturatingProduct can still be capped to a
// realistic per-slab budget.
//
// Takes value (int) which is the hint-derived budget.
// Takes ceiling (int) which is the per-slab safety cap.
//
// Returns value when value <= ceiling, otherwise ceiling. Negative values are normalised
// to zero.
func clampAtMost(value, ceiling int) int {
	if value < 0 {
		return 0
	}
	if value > ceiling {
		return ceiling
	}
	return value
}

// accumulateArenaBytecodeHints walks a function body, counting the opcodes that write to
// the byte slab or allocate from a typed backing slab. Each count contributes to its
// hint's per-opcode budget, summed across the program to give sizeArenaFromFunctions a
// tight pre-sizing target.
//
// Takes hints (*arenaBytecodeHints) which is the accumulator.
// Takes body ([]instruction) which is the function's bytecode.
func accumulateArenaBytecodeHints(hints *arenaBytecodeHints, body []instruction, allFuncs []*CompiledFunction) {
	for i := range body {
		switch body[i].op {
		case opConcatString:
			hints.bytes += arenaConcatStringAvgBytes
		case opConcatRuneString:
			hints.bytes += arenaConcatRuneStringAvgBytes
		case opDrillTier1:
			accumulateTier1Hints(hints, body[i].a)
		case opPackInterface, opPackTyped:
			accumulatePackHint(hints, registerKind(body[i].c))
		case opMakeClosure:
			accumulateClosureHint(hints, body[i], allFuncs)
		case opAppend, opAppendSpread, opAppendInPlace, opAppendSpreadInPlace:
			hints.sliceHeaders++
			hints.appendOccurrences++
		case opAppendByteFast, opAppendByteFastInPlace:
			hints.sliceHeaders++
		case opMakeSlice:
			hints.sliceHeaders++
			hints.makeSliceGeneric++
			hints.genericBytes += arenaMakeSliceGenericAvgBytes
		case opAllocIndirect:
			hints.allocIndirectOccurrences++
			hints.sliceHeaders++
			hints.genericBytes += arenaAllocIndirectAvgBytes
		default:
		}
	}
}

// accumulateClosureHint resolves the callee of an opMakeClosure instruction via its
// wide-index operand and charges the callee's upvalueDescriptor count against the upvalue
// hints. Each runtime invocation of the closure allocates one upvalueCell per descriptor.
//
// Falls through silently when the encoded index is out of range (the runtime emits a
// vmBoundsError on the same condition, so degrade-to-no-op is the right behaviour for the
// compile-time pass).
//
// Takes hints (*arenaBytecodeHints) which is the accumulator.
// Takes instr (instruction) which is the opMakeClosure instruction.
// Takes allFuncs ([]*CompiledFunction) which is the file-set function slice (root +
// functions + variable-init in that order).
func accumulateClosureHint(hints *arenaBytecodeHints, instr instruction, allFuncs []*CompiledFunction) {
	index := int(instr.wideIndex())
	if index < 0 || index >= len(allFuncs) {
		return
	}
	callee := allFuncs[index]
	if callee == nil {
		return
	}
	descriptors := len(callee.upvalueDescriptors)
	hints.upvalueCells += descriptors
	hints.upvalueReferences += descriptors
}

// accumulatePackHint bumps the appropriate per-kind box or slice-header hint based on the
// source register kind of an opPackInterface / opPackTyped instruction. Mirrors the
// runtime dispatch in handlePackInterface so the hint partition matches the slab
// partition.
//
// Takes hints (*arenaBytecodeHints) which is the accumulator.
// Takes sourceKind (registerKind) which selects the kind branch.
func accumulatePackHint(hints *arenaBytecodeHints, sourceKind registerKind) {
	switch sourceKind {
	case registerInt:
		hints.boxInts++
	case registerUint:
		hints.boxUints++
	case registerFloat:
		hints.boxFloats++
	case registerString:
		hints.boxStrings++
	case registerComplex:
		hints.boxComplexes++
	case registerSliceInt, registerSliceFloat, registerSliceString,
		registerSliceBool, registerSliceUint, registerSliceByte:
		hints.sliceHeaders++
	default:
	}
}

// accumulateTier1Hints handles a single opDrillTier1 instruction, charging the
// appropriate budget to the right hint field based on the umbrella sub-op encoded in
// operand A.
//
// Takes hints (*arenaBytecodeHints) which is the accumulator.
// Takes subOp (uint8) which is the sub-opcode discriminator from the umbrella
// instruction's A operand.
func accumulateTier1Hints(hints *arenaBytecodeHints, subOp uint8) {
	switch subOpcode(subOp) {
	case subOpBytesToString:
		hints.bytes += arenaBytesToStringAvgBytes
	case subOpStrconvItoa:
		hints.bytes += arenaItoaMaxBytes
	case subOpStrconvFormatInt:
		hints.bytes += arenaFormatIntMaxBytes
	case subOpRuneToString:
		hints.bytes += arenaRuneToStringMaxBytes
	case subOpMakeSliceInt:
		hints.makeSliceInt++
	case subOpMakeSliceFloat:
		hints.makeSliceFloat++
	case subOpMakeSliceString:
		hints.makeSliceString++
	case subOpMakeSliceBool:
		hints.makeSliceBool++
	case subOpMakeSliceUint:
		hints.makeSliceUint++
	case subOpMakeMap:
		hints.makeMapOccurrences++
		hints.genericBytes += arenaMakeMapAvgBytes
	default:
	}
}

// extractStringRegisterValue reads the arena string register at index, materialising it
// against the arena so the value outlives the frame.
//
// Takes arena (*RegisterArena) which backs the string storage.
// Takes registers (*Registers) which is the frame's register file.
// Takes index (int) which is the string-bank slot to read.
//
// Returns the materialised string, or nil when index is out of range.
func extractStringRegisterValue(arena *RegisterArena, registers *Registers, index int) any {
	if index < len(registers.strings) {
		return materialiseString(arena, registers.strings[index])
	}
	return nil
}

// extractGeneralRegisterValue reads the general-bank register at index, boxing the held
// reflect.Value.
//
// Takes registers (*Registers) which is the frame's register file.
// Takes index (int) which is the general-bank slot to read.
//
// Returns the boxed value, or nil when index is out of range or the slot holds an invalid
// reflect.Value.
func extractGeneralRegisterValue(registers *Registers, index int) any {
	if index >= len(registers.general) {
		return nil
	}
	if v := registers.general[index]; v.IsValid() {
		return v.Interface()
	}
	return nil
}

// extractScalarRegisterValue reads a register at index from the bank selected by kind,
// returning a boxed Go value.
//
// Takes registers (*Registers) which is the frame's register file.
// Takes kind (registerKind) which selects the bank.
// Takes index (int) which is the slot to read.
//
// Returns the boxed value, or nil when index is out of range or kind is not a supported
// bank.
//
// One clean switch over registerKind reads better than splitting per scalar/slice; Go
// compiles the dense enum switch to a jump table so the apparent complexity is constant
// runtime cost.
//
//nolint:revive // cyclomatic justification above.
func extractScalarRegisterValue(registers *Registers, kind registerKind, index int) any {
	switch kind {
	case registerInt:
		if index < len(registers.ints) {
			return registers.ints[index]
		}
	case registerFloat:
		if index < len(registers.floats) {
			return registers.floats[index]
		}
	case registerBool:
		if index < len(registers.bools) {
			return registers.bools[index]
		}
	case registerUint:
		if index < len(registers.uints) {
			return registers.uints[index]
		}
	case registerComplex:
		if index < len(registers.complex) {
			return registers.complex[index]
		}
	case registerSliceInt:
		if index < len(registers.slicesInt) {
			return registers.slicesInt[index]
		}
	case registerSliceFloat:
		if index < len(registers.slicesFloat) {
			return registers.slicesFloat[index]
		}
	case registerSliceString:
		if index < len(registers.slicesString) {
			return registers.slicesString[index]
		}
	case registerSliceBool:
		if index < len(registers.slicesBool) {
			return registers.slicesBool[index]
		}
	case registerSliceUint:
		if index < len(registers.slicesUint) {
			return registers.slicesUint[index]
		}
	case registerSliceByte:
		if index < len(registers.slicesByte) {
			return registers.slicesByte[index]
		}
	default:
	}
	return nil
}

// syncIndirectNamedResult dereferences a heap-promoted named result into its slot.
//
// The declaring frame heap-promoted the result so its register holds a *T pointer; this
// writes the pointed-to value into the return slot, recovering the post-defer state of
// named results that were mutated through the heap pointer, mirroring Go's spec: defers
// can modify named results, and the caller must observe those modifications.
//
// Takes registers (*Registers) which is the frame's register set.
// Takes location (varLocation) which holds the indirect register and originalKind
// metadata.
// Takes bankCounters (*[NumRegisterKinds]uint8) which tracks the next free return-slot
// register per bank; mutated to reserve the slot this named result claims.
func syncIndirectNamedResult(registers *Registers, location varLocation, bankCounters *[NumRegisterKinds]uint8) {
	kind := location.originalKind
	destinationRegister := bankCounters[kind]
	bankCounters[kind]++
	pointer := registers.general[location.register]
	if !pointer.IsValid() {
		return
	}
	target := pointer.Elem()
	if !target.IsValid() {
		return
	}
	switch kind {
	case registerInt:
		registers.ints[destinationRegister] = target.Int()
	case registerFloat:
		registers.floats[destinationRegister] = target.Float()
	case registerString:
		registers.strings[destinationRegister] = target.String()
	case registerGeneral:
		registers.general[destinationRegister] = target
	case registerBool:
		registers.bools[destinationRegister] = target.Bool()
	case registerUint:
		registers.uints[destinationRegister] = target.Uint()
	case registerComplex:
		registers.complex[destinationRegister] = target.Complex()
	default:
	}
}

// newVM creates a new virtual machine ready to execute bytecode. Concurrent use of the
// returned VM is not safe; each goroutine must use its own VM instance.
//
// Takes globals (*globalStore) which holds the package-level variable store shared across
// functions.
// Takes symbols (*SymbolRegistry) which provides access to pre-registered native
// functions and values.
//
// Returns the initialised VM.
func newVM(ctx context.Context, globals *globalStore, symbols *SymbolRegistry) *VM {
	vm := &VM{
		framePointer:         -1,
		recoverEligibleFrame: -1,
		globals:              globals,
		symbols:              symbols,
		ctx:                  ctx,
	}
	if ctx.Done() != nil {
		flag := &vm.cancelled
		vm.stopWatcher = context.AfterFunc(ctx, func() { flag.Store(1) })
	}
	return vm
}
