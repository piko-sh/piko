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
	"io"
	"reflect"
	"sync/atomic"
)

// deferredCall records a single deferred function call.
type deferredCall struct {
	// function is the piko closure to call, or nil when the target is a native callable.
	function *runtimeClosure

	// nativeFunction is the reflect.Value of a callable Go func to invoke via reflect.Call
	// when the target is not a piko closure. Holds reflect.Value{} when function is non-nil.
	nativeFunction reflect.Value

	// arguments holds the eagerly evaluated argument values.
	arguments []reflect.Value

	// frameIndex is the call frame that registered this defer.
	frameIndex int
}

// callFrame holds the runtime state for a single function invocation.
type callFrame struct {
	// function is the compiled function being executed in this frame.
	function *CompiledFunction

	// sharedCells deduplicates upvalueCell pointers across closures.
	//
	// The (kind<<8|regIndex) key maps to the cell that holds the captured variable so
	// multiple closures see one cell. Lazily allocated (via the sharedCellMapPool) on first
	// closure creation in a frame; released back to the pool on popFrame. Same 8-byte field
	// size as the map[uint16]*upvalueCell it replaced, so asm offsets after CF_SHARED_CELLS
	// are unchanged.
	sharedCells *sharedCellMap

	// simpleDefer holds the trivial-defer fast-path slot for this frame.
	//
	// Nil when the frame has no trivial defer armed. The pointer is heap-allocated only at
	// the first registerTrivialDefer call on each frame; subsequent calls reuse the slot.
	// Storing as a pointer rather than inline keeps callFrame at the size baked into the ASM
	// dispatch's hardcoded CALLFRAME_SIZE.
	simpleDefer *simpleDeferRecord

	// registers holds the typed register banks for this frame.
	registers Registers

	// upvalues holds the captured variables for closures.
	upvalues []upvalue

	// returnDestination records where to place return values in the CALLER's frame on
	// return. Set by opCall.
	returnDestination []varLocation

	// arenaSave records the arena state before this frame's registers were allocated.
	// Restored on popFrame to reclaim arena space, turning the bump allocator into a stack
	// allocator.
	arenaSave ArenaSavePoint

	// programCounter is the program counter (next instruction index to execute).
	programCounter int

	// deferBase is the index into vm.deferStack at which this frame's deferred calls start.
	// When the frame returns, all defers from deferBase onwards are executed in LIFO order.
	deferBase int

	// hasGeneralAlloc records whether this frame allocated general-bank slots from the
	// arena.
	//
	// Set when callee.numRegisters[general]>0 at frame setup. The inline-return ASM reads it
	// as a single-byte CMP to decide whether the general-slab clear+restore trampoline must
	// run. Set at frame setup from both the ASM call-side (isFastPath==3) and Go-side
	// pushCompiledFrame using the same register-count check, so the return-side check costs
	// one cache-hot byte load.
	hasGeneralAlloc bool
}

// simpleDeferRecord captures the registration state for a trivial defer.
//
// Held off-frame via a pointer to avoid bloating callFrame (whose size is pinned by the
// ASM dispatch's CALLFRAME_SIZE constant). Allocated lazily on the first trivial-defer
// registration per frame and reused across nested invocations of the same frame slot via
// the arena's frame reuse.
type simpleDeferRecord struct {
	// target holds the deferred piko closure when non-nil. The classifier guarantees the
	// closure has no captures beyond already-live registers, so the target survives frame
	// teardown when handed to executeDeferredCall.
	target *runtimeClosure

	// nativeFunction holds the deferred native callable when target is nil. Zero
	// reflect.Value otherwise.
	nativeFunction reflect.Value

	// arguments holds the eagerly-evaluated arguments. Sized to the classifier-accepted
	// maximum (typically zero for the defer-method-value pattern); reused across re-arming.
	arguments []reflect.Value

	// active reports whether this record holds a registration that runFrameSimpleDefer
	// should dispatch at return time. The frame checks this before invoking the deferred
	// call.
	active bool
}

// initialiseUpvalues sets up the frame's upvalue slice from closure cells. When arena is
// non-nil the upvalue slice comes from the arena's upvalueReferenceSlab, bump-allocated
// and freed alongside the frame, eliminating the per-call heap allocation that the make()
// path otherwise pays.
//
// Takes cells ([]*upvalueCell) which provides the upvalue cells captured by the closure.
// Takes arena (*RegisterArena) which provides slab allocation; nil falls back to make()
// for the no-arena dispatch path.
func (f *callFrame) initialiseUpvalues(cells []*upvalueCell, arena *RegisterArena) {
	var upvals []upvalue
	if arena != nil {
		upvals = arena.allocUpvalueRefs(len(cells))
	} else {
		upvals = make([]upvalue, len(cells))
	}
	for i, cell := range cells {
		upvals[i] = upvalue{value: cell}
	}
	f.upvalues = upvals
}

// upvalue holds a reference to a captured variable in a closure.
type upvalue struct {
	// value holds the captured value. For heap-escaped variables, this is shared across all
	// closures that captured the same variable.
	value *upvalueCell
}

// upvalueCell is a heap-allocated box for a captured variable. All closures that capture
// the same variable share the same cell.
type upvalueCell struct {
	// generalValue holds the captured value when the kind is registerGeneral.
	generalValue reflect.Value

	// stringValue holds the captured value when the kind is registerString.
	stringValue string

	// sliceIntValue holds the captured slice header when the kind is registerSliceInt. Slice
	// headers are value types; mutating elements affects the array shared with the declaring
	// frame, and re-slicing produces a fresh header local to the closure.
	sliceIntValue []int64

	// sliceFloatValue mirrors sliceIntValue for registerSliceFloat.
	sliceFloatValue []float64

	// sliceStringValue mirrors sliceIntValue for registerSliceString.
	sliceStringValue []string

	// sliceBoolValue mirrors sliceIntValue for registerSliceBool.
	sliceBoolValue []bool

	// sliceUintValue mirrors sliceIntValue for registerSliceUint.
	sliceUintValue []uint64

	// sliceByteValue mirrors sliceIntValue for registerSliceByte.
	sliceByteValue []byte

	// intValue holds the captured value when the kind is registerInt.
	intValue int64

	// floatValue holds the captured value when the kind is registerFloat.
	floatValue float64

	// uintValue holds the captured value when the kind is registerUint.
	uintValue uint64

	// complexValue holds the captured value when the kind is registerComplex.
	complexValue complex128

	// boolValue holds the captured value when the kind is registerBool.
	boolValue bool

	// kind identifies which register bank this cell corresponds to.
	kind registerKind

	// isIndirect signals that generalValue holds a heap-box pointer.
	//
	// True when generalValue is a *T pointer to a heap-allocated cell shared between the
	// declaring frame and every closure that captures the variable. The compile-time
	// pre-pass promotes a captured local via opAllocIndirect when an inner closure (or
	// goroutine launched from it) writes the variable or takes its address; both parent and
	// child then read/write the same memory through the pointer rather than maintaining
	// independent snapshots. Reads route through cell.generalValue.Elem() and writes through
	// cell.generalValue.Elem().Set...().
	isIndirect bool

	// originalKind names the register bank the variable had before heap promotion.
	//
	// handleGetUpvalue / handleSetUpvalue use this to unbox the dereferenced reflect.Value
	// back into the typed bank the closure body expects. Meaningful only when isIndirect is
	// true; ignored on snapshot cells.
	originalKind registerKind
}

// tailCallArgument snapshots a single argument value before a tail call reclaims the
// current frame's registers via arena.Restore(). Instances must NOT live in the register
// arena; the snapshot must survive the arena restore that invalidates the caller's
// register memory.
type tailCallArgument struct {
	// generalValue holds the snapshotted value when the kind is registerGeneral.
	generalValue reflect.Value

	// stringValue holds the snapshotted value when the kind is registerString.
	stringValue string

	// sliceIntValue holds the snapshotted slice header when the kind is registerSliceInt;
	// the header survives arena restore because Go slice headers are value types.
	sliceIntValue []int64

	// sliceFloatValue holds the snapshotted slice header for registerSliceFloat.
	sliceFloatValue []float64

	// sliceStringValue holds the snapshotted slice header for registerSliceString.
	sliceStringValue []string

	// sliceBoolValue holds the snapshotted slice header for registerSliceBool.
	sliceBoolValue []bool

	// sliceUintValue holds the snapshotted slice header for registerSliceUint.
	sliceUintValue []uint64

	// sliceByteValue holds the snapshotted slice header for registerSliceByte.
	sliceByteValue []byte

	// intValue holds the snapshotted value when the kind is registerInt.
	intValue int64

	// floatValue holds the snapshotted value when the kind is registerFloat.
	floatValue float64

	// uintValue holds the snapshotted value when the kind is registerUint.
	uintValue uint64

	// complexValue holds the snapshotted value when the kind is registerComplex.
	complexValue complex128

	// boolValue holds the snapshotted value when the kind is registerBool.
	boolValue bool

	// kind identifies which register bank this argument belongs to.
	kind registerKind
}

// vmLimits holds resource constraints enforced during execution.
type vmLimits struct {
	// capabilityHook is the optional capability hook consulted before every gated native
	// operation; nil is treated as permissive.
	capabilityHook CapabilityHook

	// tracker holds shared atomic counters for resource usage across VMs.
	tracker *resourceTracker

	// arenaFactory provides a custom RegisterArena constructor for testing.
	arenaFactory func() *RegisterArena

	// costTable is the per-opcode cost table for cost metering. Nil when cost metering is
	// disabled.
	costTable *CostTable

	// diagnostics is the per-VM counters block for fast-path anomalies. Lazily allocated on
	// first read so VMs that never trigger an anomaly pay no memory cost.
	diagnostics *fastPathDiagnostics

	// maxCallDepth is the maximum call stack depth before stack overflow.
	maxCallDepth int

	// maxStringSize is the maximum string length in bytes that a concatenation may produce.
	// Zero means unlimited.
	maxStringSize int

	// maxOutputSize is the maximum total output bytes allowed for print statements.
	maxOutputSize int

	// costBudget is the total computation cost budget. Zero means cost metering is disabled.
	costBudget int64

	// maxArenaBytes caps the cumulative bytes the register arena may grow to within a single
	// Execute; zero selects the default.
	maxArenaBytes uint64

	// maxAllocSize is the maximum allocation size in bytes for a single object.
	maxAllocSize int

	// maxGoroutines is the maximum number of concurrent goroutines allowed.
	maxGoroutines int32

	// yieldInterval is the number of instructions between runtime.Gosched() calls.
	yieldInterval uint32

	// forceGoDispatch forces the pure Go dispatch loop even on architectures with ASM
	// threaded dispatch (amd64, arm64). Used for testing dispatch parity.
	forceGoDispatch bool
}

// resourceTracker holds shared atomic counters for resource tracking across parent and
// child VMs.
type resourceTracker struct {
	// goroutineCount tracks the number of active goroutines spawned by the VM.
	goroutineCount atomic.Int32

	// outputBytes tracks the total bytes written to stderr by print statements.
	outputBytes atomic.Int64
}

// countingWriter wraps an io.Writer to count bytes written via the shared
// resourceTracker.
type countingWriter struct {
	// writer is the underlying writer that receives the output bytes.
	writer io.Writer

	// tracker is the shared resource tracker that accumulates byte counts.
	tracker *resourceTracker
}

// Write writes p to the underlying writer and adds the byte count to the tracker.
//
// Takes p ([]byte) which specifies the bytes to write.
//
// Returns the number of bytes written and any write error.
func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.writer.Write(p)
	c.tracker.outputBytes.Add(int64(n))
	return n, err
}
