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
	"io"
	"os"
	"reflect"
	"runtime"
	"sync/atomic"
	"unsafe"
)

const (
	// maxCallDepth is the maximum call stack depth before a stack overflow
	// error is raised.
	maxCallDepth = 10000

	// cancellationCheckMask is the bitmask applied to the operation
	// counter to decide when to check for context cancellation.
	cancellationCheckMask uint32 = 0x3FF

	// errIdxOutOfRangeFmt is the format string for index-out-of-range
	// error messages in VM handlers.
	errIdxOutOfRangeFmt = "%w: %d with length %d"

	// opcodeTableSize is the number of entries in the opcode dispatch
	// table (one per possible uint8 opcode value).
	opcodeTableSize = 256
)

// VM is the virtual machine that executes compiled bytecode.
type VM struct {
	// evalResult holds the return value from handleReturn when the base
	// frame returns. Extracted by the dispatch loop after opDone.
	evalResult any

	// evalAllResults holds all return values from handleReturn when
	// the base frame returns, used by callClosureReflect to preserve
	// multi-return values.
	evalAllResults []any

	// panicValue holds the current panic value during unwinding.
	panicValue any

	// evalError holds the error from a handler that returns opPanicError.
	// Extracted by the dispatch loop after opPanicError.
	evalError error

	// stderrWriter is the writer for print/println output, defaulting
	// to os.Stderr but overridable for testing.
	stderrWriter io.Writer

	// ctx is the execution context for cancellation support.
	ctx context.Context

	// rootFunction is the top-level compiled function that owns the method
	// table and function table. Set during execute().
	rootFunction *CompiledFunction

	// globals holds package-level variables shared across all
	// functions in the program.
	globals *globalStore

	// symbols provides access to pre-registered native functions
	// and values.
	symbols *SymbolRegistry

	// arena provides pooled register bank allocation. When set,
	// pushFrame uses arena.AllocRegisters instead of newRegisters,
	// avoiding per-frame heap allocations.
	arena *RegisterArena

	// asmCallInfoTables holds pre-computed asmCallInfo tables for
	// each function, keyed by *CompiledFunction. Built during
	// execute() for ASM-inlined call/return.
	asmCallInfoTables map[*CompiledFunction][]asmCallInfo

	// closureCache caches reflect.Value wrappers for zero-upvalue
	// closures (immutable function values that never change).
	closureCache map[uint16]reflect.Value

	// methodCache maps (reflect.Type, method name) to the method
	// index so that handleGetMethod can use reflect.Value.Method(index)
	// on repeat calls instead of the allocating MethodByName lookup.
	methodCache map[methodCacheKey]int

	// deferStack holds deferred function calls. Each frame records
	// its deferBase so that only defers from the current frame are
	// run when the frame returns.
	deferStack []deferredCall

	// asmCallInfoBases is a parallel array to callStack, holding the
	// asmCallInfo table base pointer for the function at each
	// frame. Used by ASM to switch tables on call/return.
	asmCallInfoBases []uintptr

	// builtinArgsBuf is a reusable buffer for handleCallBuiltin arguments.
	// Safe because VM is per-goroutine and builtins are synchronous.
	builtinArgsBuf []any

	// selectCasesBuf is a reusable buffer for handleSelect reflect cases.
	selectCasesBuf []reflect.SelectCase

	// selectInfosBuf is a reusable buffer for handleSelect case metadata.
	selectInfosBuf []selectCaseInfo

	// asmDispatchSaves is a parallel array to callStack, holding the
	// dispatch register values (codeBase, codeLength, intConstantsBase,
	// floatConstantsBase) saved by the ASM call handler for restoration
	// by the return handler.
	asmDispatchSaves []asmDispatchSave

	// callStack holds the call frames. The current frame is at
	// callStack[fp].
	callStack []callFrame

	// rootSnapshots runs parallel to callStack; entry i holds the
	// dispatch state to restore when frame i returns, or nil when no
	// swap happened at push time.
	rootSnapshots []*frameRootSnapshot

	// functions is the program-level function table. All compiled
	// functions are stored here, referenced by index from CallSites.
	functions []*CompiledFunction

	// debugHook is called at debug-relevant points when debugging is
	// active. Nil when no debugger is attached.
	debugHook DebugHook

	// debugState holds mutable debug state (breakpoints, stepping).
	// Nil when no debugger is attached.
	debugState *debugState

	// debugActive is shared with the Debugger and set to 1 when
	// debugging is active. Checked via atomic load (~1.5ns) in the
	// hot loop - same pattern as cancelled.
	debugActive *atomic.Uint32

	// limits holds resource constraints for DoS protection.
	limits vmLimits

	// costRemaining is the remaining computation cost budget.
	costRemaining int64

	// yieldCounter counts instructions for Gosched() yielding.
	yieldCounter uint32

	// framePointer is the current frame pointer (index into callStack).
	framePointer int

	// baseFramePointer is the base frame pointer for the current run()
	// invocation, used by opReturn/opReturnVoid to detect when the base
	// frame returns rather than hardcoding zero.
	baseFramePointer int

	// cancelled is set to 1 by a background goroutine when the
	// execution context is done. Checked via atomic load (~1.5ns)
	// instead of ctx.Err() (~7-10ns) in the hot loop.
	cancelled atomic.Uint32

	// panicking is true when the VM is unwinding due to a panic.
	panicking bool

	// hasGoroutines is set when the first opGo is executed. When
	// true, handleSetGlobal clones arena-backed strings immediately
	// instead of deferring to execute() cleanup, because child
	// goroutines share the global store and may outlive this arena.
	hasGoroutines bool
}

// updateASMCallInfoBase sets the asmCallInfoBases entry for the current frame
// after a Go-side frame change (push/pop). Grows the parallel arrays
// if the callStack has grown.
func (vm *VM) updateASMCallInfoBase() {
	if vm.asmCallInfoBases == nil {
		return
	}
	if vm.framePointer >= len(vm.asmCallInfoBases) {
		vm.growCallStack()
	}
	if vm.framePointer >= 0 {
		frame := &vm.callStack[vm.framePointer]
		if table, ok := vm.asmCallInfoTables[frame.function]; ok && len(table) > 0 {
			vm.asmCallInfoBases[vm.framePointer] = uintptr(unsafe.Pointer(&table[0]))
		} else {
			vm.asmCallInfoBases[vm.framePointer] = 0
		}
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
// Returns the configured maxCallDepth limit, or the default
// maxCallDepth constant if unset.
func (vm *VM) callDepthLimit() int {
	if vm.limits.maxCallDepth > 0 {
		return vm.limits.maxCallDepth
	}
	return maxCallDepth
}

// limitedStderr returns a writer that counts bytes written when an
// output size limit is configured.
//
// Returns a countingWriter wrapping stderr when limits are active, or
// the plain stderr writer otherwise.
func (vm *VM) limitedStderr() io.Writer {
	if vm.limits.maxOutputSize > 0 && vm.limits.tracker != nil {
		return &countingWriter{writer: vm.stderr(), tracker: vm.limits.tracker}
	}
	return vm.stderr()
}

// checkOutputLimit checks whether the output size limit has been
// exceeded and sets vm.evalError if so.
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

// deferredCall records a single deferred function call.
type deferredCall struct {
	// function is the closure or function to call.
	function *runtimeClosure

	// arguments holds the eagerly evaluated argument values.
	arguments []reflect.Value

	// frameIndex is the call frame that registered this defer.
	frameIndex int
}

// callFrame represents a single function invocation on the call stack.
type callFrame struct {
	// registers holds the typed register banks for this frame.
	registers Registers

	// function is the compiled function being executed in this frame.
	function *CompiledFunction

	// sharedCells maps (kind<<8|regIndex) to an upvalue cell so that
	// multiple closures capturing the same variable share one cell.
	sharedCells map[uint16]*upvalueCell

	// upvalues holds the captured variables for closures.
	upvalues []upvalue

	// returnDestination records where to place return values in the CALLER's
	// frame when this function returns. Set by opCall.
	returnDestination []varLocation

	// programCounter is the program counter (next instruction index to execute).
	programCounter int

	// deferBase is the index into vm.deferStack at which this frame's
	// deferred calls start. When the frame returns, all defers from
	// deferBase onwards are executed in LIFO order.
	deferBase int

	// arenaSave records the arena state before this frame's registers
	// were allocated. Restored on popFrame to reclaim arena space,
	// turning the bump allocator into a stack allocator.
	arenaSave ArenaSavePoint
}

// initialiseUpvalues sets up the frame's upvalue slice from closure cells.
//
// Takes cells ([]*upvalueCell) which provides the upvalue cells
// captured by the closure.
func (f *callFrame) initialiseUpvalues(cells []*upvalueCell) {
	f.upvalues = make([]upvalue, len(cells))
	for i, cell := range cells {
		f.upvalues[i] = upvalue{value: cell}
	}
}

// upvalue holds a reference to a captured variable in a closure.
type upvalue struct {
	// value holds the captured value. For heap-escaped variables, this
	// is shared across all closures that captured the same variable.
	value *upvalueCell
}

// upvalueCell is a heap-allocated box for a captured variable. All
// closures that capture the same variable share the same cell.
type upvalueCell struct {
	// generalValue holds the captured value when the kind is registerGeneral.
	generalValue reflect.Value

	// stringValue holds the captured value when the kind is registerString.
	stringValue string

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
}

// tailCallArg snapshots a single argument value before a tail call
// reclaims the current frame's registers via arena.Restore(). This
// type must NOT live in the register arena; the snapshot must survive
// the arena restore that invalidates the caller's register memory.
type tailCallArg struct {
	// generalValue holds the snapshotted value when the kind is registerGeneral.
	generalValue reflect.Value

	// stringValue holds the snapshotted value when the kind is registerString.
	stringValue string

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
	// tracker holds shared atomic counters for resource usage across VMs.
	tracker *resourceTracker

	// arenaFactory provides a custom RegisterArena constructor for testing.
	arenaFactory func() *RegisterArena

	// costTable is the per-opcode cost table for cost metering. Nil
	// when cost metering is disabled.
	costTable *CostTable

	// maxAllocSize is the maximum allocation size in bytes for a single object.
	maxAllocSize int

	// maxStringSize is the maximum string length in bytes that a
	// concatenation may produce. Zero means unlimited.
	maxStringSize int

	// maxCallDepth is the maximum call stack depth before stack overflow.
	maxCallDepth int

	// maxOutputSize is the maximum total output bytes allowed for
	// print statements.
	maxOutputSize int

	// costBudget is the total computation cost budget. Zero means
	// cost metering is disabled.
	costBudget int64

	// maxGoroutines is the maximum number of concurrent goroutines allowed.
	maxGoroutines int32

	// yieldInterval is the number of instructions between
	// runtime.Gosched() calls.
	yieldInterval uint32

	// forceGoDispatch forces the pure Go dispatch loop even on
	// architectures with ASM threaded dispatch (amd64, arm64).
	// Used for testing dispatch parity.
	forceGoDispatch bool
}

// resourceTracker holds shared atomic counters for resource tracking
// across parent and child VMs.
type resourceTracker struct {
	// goroutineCount tracks the number of active goroutines spawned by the VM.
	goroutineCount atomic.Int32

	// outputBytes tracks the total bytes written to stderr by print statements.
	outputBytes atomic.Int64
}

// countingWriter wraps an io.Writer to count bytes written via the
// shared resourceTracker.
type countingWriter struct {
	// writer is the underlying writer that receives the output bytes.
	writer io.Writer

	// tracker is the shared resource tracker that accumulates byte counts.
	tracker *resourceTracker
}

// Write writes p to the underlying writer and adds the byte count to
// the tracker.
//
// Takes p ([]byte) which specifies the bytes to write.
//
// Returns the number of bytes written and any write error.
func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.writer.Write(p)
	c.tracker.outputBytes.Add(int64(n))
	return n, err
}

// acquireArena returns a RegisterArena, using the custom factory if
// configured or the global pool otherwise.
//
// Returns a RegisterArena from the custom factory or the global pool.
func (vm *VM) acquireArena() *RegisterArena {
	if vm.limits.arenaFactory != nil {
		return vm.limits.arenaFactory()
	}
	return GetRegisterArena()
}

// ensureCallStack sets up an arena and call stack for cold-path VMs
// (varinit, init) that were not created via execute(). The caller
// must defer vm.releaseArena() to return the arena to the pool.
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

// initialiseASMDispatch allocates the parallel ASM dispatch arrays
// (callInfoBases, dispatchSaves) from the arena. This must be called
// after ensureCallStack and after asmCallInfoTables has been set.
func (vm *VM) initialiseASMDispatch() {
	if vm.arena == nil {
		return
	}
	vm.asmCallInfoBases = vm.arena.CallInfoBases()
	vm.asmDispatchSaves = vm.arena.dispatchSaves()
}

// releaseArena returns the VM's arena to the global pool and clears
// the reference.
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
func (vm *VM) execute(compiledFunction *CompiledFunction) (any, error) {
	vm.functions = compiledFunction.functions
	vm.rootFunction = compiledFunction
	vm.costRemaining = vm.limits.costBudget

	ownArena := vm.arena == nil
	if ownArena {
		vm.arena = vm.acquireArena()
		vm.sizeArenaFromFunctions(compiledFunction)
	}

	if vm.callStack == nil {
		vm.callStack = vm.arena.frameStack()
	}

	compiledFunction.asmCallInfoTablesOnce.Do(func() {
		compiledFunction.asmCallInfoTables, _ = buildASMCallInfoTables(compiledFunction, compiledFunction.functions)
	})
	vm.asmCallInfoTables = compiledFunction.asmCallInfoTables
	vm.asmCallInfoBases = vm.arena.CallInfoBases()
	vm.asmDispatchSaves = vm.arena.dispatchSaves()
	if table := vm.asmCallInfoTables[compiledFunction]; len(table) > 0 {
		vm.asmCallInfoBases[0] = uintptr(unsafe.Pointer(&table[0]))
	}

	if compiledFunction.variableInitFunction != nil {
		vm.pushFrame(compiledFunction.variableInitFunction)
		if _, err := vm.run(0); err != nil {
			if ownArena {
				vm.callStack = nil
				vm.asmCallInfoTables = nil
				vm.asmCallInfoBases = nil
				vm.asmDispatchSaves = nil
				PutRegisterArena(vm.arena)
				vm.arena = nil
			}
			return nil, fmt.Errorf("varinit: %w", err)
		}
	}

	vm.pushFrame(compiledFunction)
	result, err := vm.runDispatched(0)

	if ownArena {
		if !vm.hasGoroutines {
			vm.globals.materialiseStrings(vm.arena)
		}
		vm.callStack = nil
		vm.asmCallInfoTables = nil
		vm.asmCallInfoBases = nil
		vm.asmDispatchSaves = nil
		PutRegisterArena(vm.arena)
		vm.arena = nil
	}

	return result, err
}

// sizeArenaFromFunctions inspects the compiled function table to
// pre-size the arena so that AllocRegisters never triggers a grow
// during normal execution.
//
// With stack-based reclamation (popFrame restores the arena), only
// max-depth x max-registers matters, not total calls x registers.
// This makes the estimate tight even for deeply recursive functions
// like fib(20).
//
// Takes root (*CompiledFunction) which specifies the top-level
// compiled function whose function table is used to compute the arena
// capacity.
func (vm *VM) sizeArenaFromFunctions(root *CompiledFunction) {
	var totalInt, totalFloat, totalString, totalGeneral int
	var totalBool, totalUint, totalComplex int

	allFuncs := append([]*CompiledFunction{root}, root.functions...)
	if root.variableInitFunction != nil {
		allFuncs = append(allFuncs, root.variableInitFunction)
	}
	for _, f := range allFuncs {
		totalInt += int(f.numRegisters[registerInt])
		totalFloat += int(f.numRegisters[registerFloat])
		totalString += int(f.numRegisters[registerString])
		totalGeneral += int(f.numRegisters[registerGeneral])
		totalBool += int(f.numRegisters[registerBool])
		totalUint += int(f.numRegisters[registerUint])
		totalComplex += int(f.numRegisters[registerComplex])
	}

	const depthEstimate = 64
	vm.arena.EnsureCapacity(
		totalInt*depthEstimate,
		totalFloat*depthEstimate,
		totalString*depthEstimate,
		totalGeneral*depthEstimate,
		totalBool*depthEstimate,
		totalUint*depthEstimate,
		totalComplex*depthEstimate,
	)
}

// growCallStack doubles the call stack capacity, growing the arena
// slabs or independent arrays to keep the parallel arrays (frames,
// ciBases, dispSaves) in sync.
//
// When an arena is available, the arena's slabs are grown so all three
// parallel arrays stay in sync. Without an arena, the callStack and
// parallel arrays are grown independently.
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
// Takes compiledFunction (*CompiledFunction) which specifies the
// compiled function to create a frame for.
func (vm *VM) pushFrame(compiledFunction *CompiledFunction) {
	var save ArenaSavePoint
	if vm.arena != nil {
		save = vm.arena.Save()
	}
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	f := &vm.callStack[vm.framePointer]
	if vm.arena != nil {
		vm.arena.AllocRegistersInto(&f.registers, compiledFunction.numRegisters)
	} else {
		f.registers = newRegisters(compiledFunction.numRegisters)
	}
	f.function = compiledFunction
	f.programCounter = 0
	f.deferBase = len(vm.deferStack)
	f.arenaSave = save
	f.upvalues = nil
	f.returnDestination = nil
	f.sharedCells = nil
}

// popFrame pops the current call frame and restores the previous one.
// If the arena is in use, restores it to the save point recorded when
// the frame was pushed, reclaiming the frame's register slots.
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

// run is the main execution loop, dispatching all opcodes via the
// shared handlerTable LUT defined in vm_handlers.go.
//
// Takes baseFramePointer (int) which specifies the frame index at
// which this invocation should stop and return results.
//
// Returns the execution result and any error encountered during
// dispatch.
//
//revive:disable:cognitive-complexity // VM dispatch loops are inherently complex.
func (vm *VM) run(baseFramePointer int) (any, error) {
	savedBaseFp := vm.baseFramePointer
	vm.baseFramePointer = baseFramePointer
	defer func() { vm.baseFramePointer = savedBaseFp }()

	frame := &vm.callStack[vm.framePointer]
	registers := &frame.registers

	var ops uint32
	for {
		ops++
		if ops&cancellationCheckMask == 0 && vm.cancelled.Load() != 0 {
			return nil, vm.ctx.Err()
		}
		if vm.shouldStopDebug(frame) {
			return nil, ErrDebuggerStop
		}
		if frame.programCounter >= len(frame.function.body) {
			done, result, err := vm.handleEndOfBody(frame, baseFramePointer)
			if done {
				return result, err
			}
			frame = &vm.callStack[vm.framePointer]
			registers = &frame.registers
			continue
		}

		instruction := frame.function.body[frame.programCounter]
		frame.programCounter++

		if vm.costRemaining > 0 {
			vm.costRemaining -= vm.limits.costTable[instruction.op]
			if vm.costRemaining <= 0 {
				return nil, errCostBudgetExceeded
			}
		}

		if vm.limits.yieldInterval > 0 {
			vm.yieldCounter++
			if vm.yieldCounter&(vm.limits.yieldInterval-1) == 0 {
				runtime.Gosched()
			}
		}

		rc := handlerTable[instruction.op](vm, frame, registers, instruction)
		if rc == opContinue {
			continue
		}
		result, terminal, err := vm.handleOpResult(rc)
		if terminal {
			return result, err
		}
		frame = &vm.callStack[vm.framePointer]
		registers = &frame.registers
	}
}

// handleEndOfBody processes the end-of-body condition for the current
// frame by running defers, then either returning the result for the
// base frame or popping the frame and continuing.
//
// Takes frame (*callFrame) which specifies the current call frame
// being completed.
// Takes baseFramePointer (int) which specifies the frame index that
// marks the bottom of this run invocation.
//
// Returns (true, result, err) when the caller should return, or
// (false, _, _) to continue the loop.
func (vm *VM) handleEndOfBody(frame *callFrame, baseFramePointer int) (bool, any, error) {
	if len(vm.deferStack) > frame.deferBase {
		vm.runDefers()
	}
	if vm.framePointer == baseFramePointer {
		result, err := vm.extractResult(frame)
		vm.popFrame()
		return true, result, err
	}
	vm.popFrame()
	if vm.framePointer < baseFramePointer {
		return true, nil, nil
	}
	return false, nil, nil
}

// handleOpResult translates an opcode handler return code into either
// a terminal result or a signal that the frame pointer changed and
// local variables must be refreshed.
//
// Takes rc (opResult) which specifies the opcode handler return code
// to translate.
//
// Returns the result value, a terminal flag indicating whether
// execution should stop, and any error.
func (vm *VM) handleOpResult(rc opResult) (result any, terminal bool, err error) {
	switch rc {
	case opDone:
		result = vm.evalResult
		vm.evalResult = nil
		return result, true, nil
	case opDivByZero:
		return nil, true, errDivisionByZero
	case opStackOverflow:
		return nil, true, errStackOverflow
	case opPanicError:
		err = vm.evalError
		vm.evalError = nil
		return nil, true, err
	default:
		return nil, false, nil
	}
}

// extractResult extracts the return value from the final frame.
//
// Takes frame (*callFrame) which specifies the call frame to extract
// the result from.
//
// Returns the result value from the first result register, or nil if
// no results are declared.
func (vm *VM) extractResult(frame *callFrame) (any, error) {
	if len(frame.function.resultKinds) == 0 {
		return nil, nil
	}

	kind := frame.function.resultKinds[0]
	switch kind {
	case registerInt:
		return frame.registers.ints[0], nil
	case registerFloat:
		return frame.registers.floats[0], nil
	case registerString:
		return materialiseString(vm.arena, frame.registers.strings[0]), nil
	case registerGeneral:
		v := frame.registers.general[0]
		if !v.IsValid() {
			return nil, nil
		}
		return v.Interface(), nil
	case registerBool:
		return frame.registers.bools[0], nil
	case registerUint:
		return frame.registers.uints[0], nil
	case registerComplex:
		return frame.registers.complex[0], nil
	default:
		return nil, nil
	}
}

// extractAllResults extracts all return values from the final frame.
// Used by callClosureReflect to preserve multi-return values.
//
// Takes frame (*callFrame) which specifies the call frame to extract
// results from.
//
// Returns a slice of all result values.
func (vm *VM) extractAllResults(frame *callFrame) []any {
	resultKinds := frame.function.resultKinds
	if len(resultKinds) == 0 {
		return nil
	}

	results := make([]any, len(resultKinds))
	var bankCounters [NumRegisterKinds]uint8

	for i, kind := range resultKinds {
		srcReg := bankCounters[kind]
		bankCounters[kind]++

		switch kind {
		case registerInt:
			results[i] = frame.registers.ints[srcReg]
		case registerFloat:
			results[i] = frame.registers.floats[srcReg]
		case registerString:
			results[i] = materialiseString(vm.arena, frame.registers.strings[srcReg])
		case registerGeneral:
			v := frame.registers.general[srcReg]
			if v.IsValid() {
				results[i] = v.Interface()
			}
		case registerBool:
			results[i] = frame.registers.bools[srcReg]
		case registerUint:
			results[i] = frame.registers.uints[srcReg]
		case registerComplex:
			results[i] = frame.registers.complex[srcReg]
		}
	}

	return results
}

// copyReturnValueAt copies a return value from the callee frame at
// the given source register to the caller frame at the destination.
//
// Takes calleeFrame (*callFrame) which specifies the frame containing
// the source value.
// Takes kind (registerKind) which specifies the register bank of the
// source value.
// Takes srcReg (uint8) which specifies the source register index
// within the callee.
// Takes dest (varLocation) which specifies the destination location in
// the caller frame.
func (vm *VM) copyReturnValueAt(calleeFrame *callFrame, kind registerKind, srcReg uint8, dest varLocation) {
	callerFrame := &vm.callStack[vm.framePointer-1]
	if kind == dest.kind {
		copySameKind(&callerFrame.registers, &calleeFrame.registers, kind, dest.register, srcReg)
	} else if kind == registerGeneral && dest.kind != registerGeneral {
		copyReturnFromGeneral(callerFrame, calleeFrame.registers.general[srcReg], dest)
	} else if dest.kind == registerGeneral && kind != registerGeneral {
		copyReturnToGeneral(callerFrame, &calleeFrame.registers, kind, srcReg, dest.register)
	}
}

// callClosureReflect executes a compiled closure via reflect, pushing
// a new frame, assigning arguments, and running the function to
// completion.
//
// Takes closure (*runtimeClosure) which specifies the closure to call.
// Takes arguments ([]reflect.Value) which provides the reflect.Value
// arguments to pass.
// Takes funcType (reflect.Type) which defines the expected function
// signature for result packaging.
//
// Returns the result values packaged as reflect.Values matching
// funcType's output signature.
func (vm *VM) callClosureReflect(closure *runtimeClosure, arguments []reflect.Value, funcType reflect.Type) []reflect.Value {
	callee := closure.function

	snapshot := vm.swapToClosureRoot(closure.rootFunction)

	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	closureFp := vm.framePointer
	f := &vm.callStack[vm.framePointer]
	if vm.arena != nil {
		f.arenaSave = vm.arena.Save()
		vm.arena.AllocRegistersInto(&f.registers, callee.numRegisters)
	} else {
		f.registers = newRegisters(callee.numRegisters)
	}
	f.function = callee
	f.programCounter = 0
	f.returnDestination = nil
	f.deferBase = len(vm.deferStack)
	f.upvalues = nil
	f.sharedCells = nil
	vm.recordFrameSnapshot(closureFp, snapshot)
	if closure.upvalues != nil {
		f.initialiseUpvalues(closure.upvalues)
	}
	vm.updateASMCallInfoBase()

	assignReflectParams(&f.registers, callee.paramKinds, arguments)

	_, err := vm.runDispatched(closureFp)
	if err != nil {
		vm.evalError = err
	}

	allResults := vm.evalAllResults
	vm.evalAllResults = nil
	return buildReflectResults(allResults, funcType)
}

// runtimeClosure wraps a CompiledFunction with its captured upvalue
// cells. Created by opMakeClosure.
type runtimeClosure struct {
	// function is the compiled function that this closure executes.
	function *CompiledFunction

	// rootFunction is the compile root whose .functions slice resolves
	// any funcIndex references in function's bytecode; nil falls back
	// to the current VM's rootFunction.
	rootFunction *CompiledFunction

	// upvalues holds the captured variable cells shared with the enclosing scope.
	upvalues []*upvalueCell
}

// rangeIterator holds the state for a for-range loop over a slice,
// array, map, or channel.
type rangeIterator struct {
	// mapIterator holds the reflect map iterator for map range loops.
	mapIterator *reflect.MapIter

	// collection holds the reflect.Value of the slice, array, map, or
	// channel being iterated.
	collection reflect.Value

	// intSlice is a type-asserted fast path for []int collections,
	// set once at init.
	intSlice []int

	// stringSlice is a type-asserted fast path for []string
	// collections, set once at init.
	stringSlice []string

	// floatSlice is a type-asserted fast path for []float64
	// collections, set once at init.
	floatSlice []float64

	// boolSlice is a type-asserted fast path for []bool collections,
	// set once at init.
	boolSlice []bool

	// index is the current iteration position within the collection.
	index int

	// isMap indicates whether the collection is a map type.
	isMap bool

	// isChan indicates whether the collection is a channel type.
	isChan bool
}

// writeRangeValue writes a reflect.Value to the appropriate register.
//
// Takes registers (*Registers) which specifies the register banks to
// write into.
// Takes value (reflect.Value) which specifies the reflect.Value to
// store.
// Takes register (uint8) which specifies the register index within the
// bank.
// Takes kind (registerKind) which specifies which register bank to
// target.
func (*VM) writeRangeValue(registers *Registers, value reflect.Value, register uint8, kind registerKind) {
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
		registers.general[register] = value
	case registerBool:
		registers.bools[register] = value.Bool()
	case registerUint:
		registers.uints[register] = value.Uint()
	case registerComplex:
		registers.complex[register] = value.Complex()
	}
}

// syncNamedResults syncs upvalue cells back to registers for captured
// named return variables, then re-copies from named result registers
// to return positions for Go spec compliance.
//
// Takes frame (*callFrame) which specifies the call frame whose named
// results are synced.
func (*VM) syncNamedResults(frame *callFrame) {
	namedLocs := frame.function.namedResultLocs
	if len(namedLocs) == 0 || frame.sharedCells == nil {
		return
	}

	for _, location := range namedLocs {
		key := joinWide(uint8(location.kind), location.register)
		if cell, ok := frame.sharedCells[key]; ok {
			syncCellToRegister(&frame.registers, cell, location)
		}
	}

	var bankCounters [NumRegisterKinds]uint8
	for _, location := range namedLocs {
		destReg := bankCounters[location.kind]
		bankCounters[location.kind]++
		if destReg != location.register {
			copyRegisterSlot(&frame.registers, location.kind, destReg, location.register)
		}
	}
}

// runDefers executes all deferred calls registered by the current
// frame, in LIFO order. Called on normal function return.
func (vm *VM) runDefers() {
	frame := vm.currentFrame()
	for i := len(vm.deferStack) - 1; i >= frame.deferBase; i-- {
		d := vm.deferStack[i]
		vm.executeDeferredCall(d)
	}
	vm.deferStack = vm.deferStack[:frame.deferBase]
}

// unwindPanic handles panic unwinding by running deferred calls for
// each frame in LIFO order, catching panics when a deferred call
// contains a recover().
//
// Returns nil if the panic was recovered, or an error wrapping the
// panic value otherwise.
func (vm *VM) unwindPanic() error {
	for vm.framePointer >= 0 {
		if vm.unwindFrame() {
			return nil
		}
	}
	return fmt.Errorf("panic: %v", vm.panicValue)
}

// unwindFrame runs deferred calls for the current frame during panic
// unwinding, popping the frame in either case.
//
// Returns true if a recover() was found and the panic was caught, or
// false otherwise.
func (vm *VM) unwindFrame() bool {
	frame := vm.currentFrame()
	for i := len(vm.deferStack) - 1; i >= frame.deferBase; i-- {
		vm.executeDeferredCall(vm.deferStack[i])
		if !vm.panicking {
			vm.runRemainingDefers(frame.deferBase, i-1)
			vm.deferStack = vm.deferStack[:frame.deferBase]
			vm.popFrame()
			return true
		}
	}
	vm.deferStack = vm.deferStack[:frame.deferBase]
	vm.popFrame()
	return false
}

// runRemainingDefers executes deferred calls from index down to base
// (inclusive) after a recover() has caught the panic.
//
// Takes base (int) which specifies the lowest defer stack index to
// execute.
// Takes from (int) which specifies the highest defer stack index to
// start from.
func (vm *VM) runRemainingDefers(base, from int) {
	for j := from; j >= base; j-- {
		vm.executeDeferredCall(vm.deferStack[j])
	}
}

// executeDeferredCall runs a single deferred closure call by pushing
// a new call frame and executing it.
//
// Takes d (deferredCall) which specifies the deferred call record
// containing the closure and arguments.
func (vm *VM) executeDeferredCall(d deferredCall) {
	callee := d.function.function
	var deferSave ArenaSavePoint
	var deferRegs Registers
	if vm.arena != nil {
		deferSave = vm.arena.Save()
		vm.arena.AllocRegistersInto(&deferRegs, callee.numRegisters)
	} else {
		deferRegs = newRegisters(callee.numRegisters)
	}
	newFrame := callFrame{
		registers:      deferRegs,
		function:       callee,
		programCounter: 0,
		deferBase:      len(vm.deferStack),
		arenaSave:      deferSave,
	}

	if d.function.upvalues != nil {
		newFrame.initialiseUpvalues(d.function.upvalues)
	}

	var kindIndex [NumRegisterKinds]int
	for i, argument := range d.arguments {
		if i >= len(callee.paramKinds) {
			break
		}
		kind := callee.paramKinds[i]
		dest := kindIndex[kind]
		kindIndex[kind]++
		switch kind {
		case registerInt:
			newFrame.registers.ints[dest] = argument.Int()
		case registerFloat:
			newFrame.registers.floats[dest] = argument.Float()
		case registerString:
			newFrame.registers.strings[dest] = argument.String()
		case registerGeneral:
			newFrame.registers.general[dest] = argument
		case registerBool:
			newFrame.registers.bools[dest] = argument.Bool()
		case registerUint:
			newFrame.registers.uints[dest] = argument.Uint()
		case registerComplex:
			newFrame.registers.complex[dest] = argument.Complex()
		}
	}

	snapshot := vm.swapToClosureRoot(d.function.rootFunction)

	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	vm.callStack[vm.framePointer] = newFrame
	vm.recordFrameSnapshot(vm.framePointer, snapshot)

	_, _ = vm.runDispatched(vm.framePointer)
}

// newVM creates a new virtual machine ready to execute bytecode.
// Concurrent use of the returned VM is not safe; each goroutine must
// use its own VM instance.
//
// Takes ctx (context.Context) which provides the execution context for
// cancellation support.
// Takes globals (*globalStore) which holds the package-level variable
// store shared across functions.
// Takes symbols (*SymbolRegistry) which provides access to
// pre-registered native functions and values.
//
// Returns the initialised VM.
func newVM(ctx context.Context, globals *globalStore, symbols *SymbolRegistry) *VM {
	vm := &VM{
		framePointer: -1,
		globals:      globals,
		symbols:      symbols,
		ctx:          ctx,
	}
	if ctx.Done() != nil {
		flag := &vm.cancelled
		go func() {
			<-ctx.Done()
			flag.Store(1)
		}()
	}
	return vm
}

// shouldStopDebug returns true when the debug hook requests
// execution to stop.
//
// Takes frame (*callFrame) which is the current call frame to
// inspect for breakpoints.
//
// Returns bool which is true when the hook requests a stop, or
// false when debugging is inactive or the hook returns any action
// other than stop.
func (vm *VM) shouldStopDebug(frame *callFrame) bool {
	if vm.debugActive == nil || vm.debugActive.Load() == 0 {
		return false
	}
	return vm.checkDebug(frame) == DebugActionStop
}

// checkDebug evaluates breakpoints and stepping conditions at the
// current program counter.
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

// copySameKind copies a register value between frames when the source
// and destination kinds are identical.
//
// Takes dst (*Registers) which specifies the destination register banks.
// Takes source (*Registers) which specifies the source register banks.
// Takes kind (registerKind) which specifies which register bank to
// copy from.
// Takes dstReg (uint8) which specifies the destination register index.
// Takes srcReg (uint8) which specifies the source register index.
func copySameKind(dst, source *Registers, kind registerKind, dstReg, srcReg uint8) {
	switch kind {
	case registerInt:
		dst.ints[dstReg] = source.ints[srcReg]
	case registerFloat:
		dst.floats[dstReg] = source.floats[srcReg]
	case registerString:
		dst.strings[dstReg] = source.strings[srcReg]
	case registerGeneral:
		dst.general[dstReg] = source.general[srcReg]
	case registerBool:
		dst.bools[dstReg] = source.bools[srcReg]
	case registerUint:
		dst.uints[dstReg] = source.uints[srcReg]
	case registerComplex:
		dst.complex[dstReg] = source.complex[srcReg]
	}
}

// copyReturnFromGeneral unpacks a general-register value into the
// caller's typed register bank, handling interface unwrapping and
// zero-value defaults for generics compiled as any.
//
// Takes callerFrame (*callFrame) which specifies the frame to write
// the value into.
// Takes reflectValue (reflect.Value) which specifies the
// general-register value to unpack.
// Takes dest (varLocation) which specifies the destination typed
// register location.
func copyReturnFromGeneral(callerFrame *callFrame, reflectValue reflect.Value, dest varLocation) {
	if reflectValue.IsValid() && reflectValue.Kind() == reflect.Interface {
		reflectValue = reflectValue.Elem()
	}
	if !reflectValue.IsValid() {
		zeroTypedRegister(&callerFrame.registers, dest)
		return
	}
	unpackGeneralToTyped(&callerFrame.registers, reflectValue, dest)
}

// zeroTypedRegister writes a zero value into the destination register.
//
// Takes regs (*Registers) which specifies the register banks to write
// into.
// Takes dest (varLocation) which specifies the register location to
// zero.
func zeroTypedRegister(regs *Registers, dest varLocation) {
	switch dest.kind {
	case registerInt:
		regs.ints[dest.register] = 0
	case registerFloat:
		regs.floats[dest.register] = 0
	case registerString:
		regs.strings[dest.register] = ""
	case registerBool:
		regs.bools[dest.register] = false
	case registerUint:
		regs.uints[dest.register] = 0
	case registerComplex:
		regs.complex[dest.register] = 0
	}
}

// unpackGeneralToTyped extracts a concrete value from a reflect.Value
// into the appropriate typed register bank.
//
// Takes regs (*Registers) which specifies the register banks to write
// into.
// Takes v (reflect.Value) which specifies the reflect.Value to extract
// the concrete value from.
// Takes dest (varLocation) which specifies the destination typed
// register location.
func unpackGeneralToTyped(regs *Registers, v reflect.Value, dest varLocation) {
	switch dest.kind {
	case registerInt:
		if v.Kind() == reflect.Bool {
			regs.ints[dest.register] = boolToInt64(v.Bool())
		} else {
			regs.ints[dest.register] = v.Int()
		}
	case registerFloat:
		regs.floats[dest.register] = v.Float()
	case registerString:
		regs.strings[dest.register] = v.String()
	case registerBool:
		regs.bools[dest.register] = v.Bool()
	case registerUint:
		regs.uints[dest.register] = v.Uint()
	case registerComplex:
		regs.complex[dest.register] = v.Complex()
	}
}

// copyReturnToGeneral boxes a typed register value into a
// reflect.Value and stores it in the caller's general register.
//
// Takes callerFrame (*callFrame) which specifies the frame whose
// general register receives the value.
// Takes srcRegs (*Registers) which specifies the source register banks
// containing the typed value.
// Takes kind (registerKind) which specifies which typed register bank
// to read from.
// Takes srcReg (uint8) which specifies the source register index.
// Takes dstReg (uint8) which specifies the destination general
// register index.
func copyReturnToGeneral(callerFrame *callFrame, srcRegs *Registers, kind registerKind, srcReg, dstReg uint8) {
	switch kind {
	case registerInt:
		callerFrame.registers.general[dstReg] = reflect.ValueOf(srcRegs.ints[srcReg])
	case registerFloat:
		callerFrame.registers.general[dstReg] = reflect.ValueOf(srcRegs.floats[srcReg])
	case registerString:
		callerFrame.registers.general[dstReg] = reflect.ValueOf(srcRegs.strings[srcReg])
	case registerBool:
		callerFrame.registers.general[dstReg] = reflect.ValueOf(srcRegs.bools[srcReg])
	case registerUint:
		callerFrame.registers.general[dstReg] = reflect.ValueOf(srcRegs.uints[srcReg])
	case registerComplex:
		callerFrame.registers.general[dstReg] = reflect.ValueOf(srcRegs.complex[srcReg])
	}
}

// assignReflectParams writes reflect.Value arguments into the typed
// register banks according to the compiled parameter kinds.
//
// Takes regs (*Registers) which specifies the register banks to write
// into.
// Takes paramKinds ([]registerKind) which specifies the register kind
// for each parameter position.
// Takes arguments ([]reflect.Value) which provides the reflect.Value
// arguments to assign.
func assignReflectParams(regs *Registers, paramKinds []registerKind, arguments []reflect.Value) {
	var paramIndex [NumRegisterKinds]uint8
	for i, argument := range arguments {
		if i >= len(paramKinds) {
			break
		}
		if argument.Kind() == reflect.Interface && !argument.IsNil() {
			argument = argument.Elem()
		}
		kind := paramKinds[i]
		register := paramIndex[kind]
		paramIndex[kind]++
		assignReflectArg(regs, kind, register, argument)
	}
}

// assignReflectArg writes a single reflect.Value argument into the
// register bank for the given kind.
//
// Takes regs (*Registers) which specifies the register banks to write
// into.
// Takes kind (registerKind) which specifies which register bank to
// target.
// Takes register (uint8) which specifies the register index within the
// bank.
// Takes argument (reflect.Value) which provides the reflect.Value to
// assign.
func assignReflectArg(regs *Registers, kind registerKind, register uint8, argument reflect.Value) {
	switch kind {
	case registerInt:
		regs.ints[register] = argument.Int()
	case registerFloat:
		regs.floats[register] = argument.Float()
	case registerString:
		regs.strings[register] = argument.String()
	case registerBool:
		regs.bools[register] = argument.Bool()
	case registerUint:
		regs.uints[register] = argument.Uint()
	case registerComplex:
		regs.complex[register] = argument.Complex()
	default:
		regs.general[register] = argument
	}
}

// buildReflectResults packages the VM's results into a slice of
// reflect.Value matching the function's output signature.
//
// Takes allResults ([]any) which specifies all return values from the
// VM execution.
// Takes funcType (reflect.Type) which defines the function signature
// whose output types determine the result packaging.
//
// Returns a slice of reflect.Values with results converted to match
// funcType's output types.
func buildReflectResults(allResults []any, funcType reflect.Type) []reflect.Value {
	numOut := funcType.NumOut()
	results := make([]reflect.Value, numOut)
	for i := range numOut {
		outType := funcType.Out(i)
		if i < len(allResults) && allResults[i] != nil {
			rv := reflect.ValueOf(allResults[i])
			if rv.Type().ConvertibleTo(outType) {
				results[i] = rv.Convert(outType)
			} else {
				results[i] = rv
			}
		} else {
			results[i] = reflect.Zero(outType)
		}
	}
	return results
}

// syncCellToRegister copies the value from an upvalue cell back into
// the register at the given location.
//
// Takes regs (*Registers) which specifies the register banks to write
// into.
// Takes cell (*upvalueCell) which specifies the upvalue cell to read
// from.
// Takes location (varLocation) which specifies the register location to
// write to.
func syncCellToRegister(regs *Registers, cell *upvalueCell, location varLocation) {
	switch location.kind {
	case registerInt:
		regs.ints[location.register] = cell.intValue
	case registerFloat:
		regs.floats[location.register] = cell.floatValue
	case registerString:
		regs.strings[location.register] = cell.stringValue
	case registerGeneral:
		regs.general[location.register] = cell.generalValue
	case registerBool:
		regs.bools[location.register] = cell.boolValue
	case registerUint:
		regs.uints[location.register] = cell.uintValue
	case registerComplex:
		regs.complex[location.register] = cell.complexValue
	}
}

// copyRegisterSlot copies a value within the same register bank from
// srcReg to dstReg.
//
// Takes regs (*Registers) which specifies the register banks to
// operate on.
// Takes kind (registerKind) which specifies which register bank to
// copy within.
// Takes dstReg (uint8) which specifies the destination register index.
// Takes srcReg (uint8) which specifies the source register index.
func copyRegisterSlot(regs *Registers, kind registerKind, dstReg, srcReg uint8) {
	switch kind {
	case registerInt:
		regs.ints[dstReg] = regs.ints[srcReg]
	case registerFloat:
		regs.floats[dstReg] = regs.floats[srcReg]
	case registerString:
		regs.strings[dstReg] = regs.strings[srcReg]
	case registerGeneral:
		regs.general[dstReg] = regs.general[srcReg]
	case registerBool:
		regs.bools[dstReg] = regs.bools[srcReg]
	case registerUint:
		regs.uints[dstReg] = regs.uints[srcReg]
	case registerComplex:
		regs.complex[dstReg] = regs.complex[srcReg]
	}
}

// reflectBinaryOp performs a binary operation on two reflect.Values,
// dispatching on the underlying kind to the appropriate operation
// function.
//
// Takes a (reflect.Value) which specifies the left operand value.
// Takes b (reflect.Value) which specifies the right operand value.
// Takes intOp (func(int64, int64) int64) which provides the operation
// for integer types, or nil to skip.
// Takes floatOp (func(float64, float64) float64) which provides the
// operation for float types, or nil to skip.
// Takes stringOp (func(string, string) string) which provides the
// operation for string types, or nil to skip.
//
// Returns a reflect.Value of the same type as the operands, or an
// invalid reflect.Value if the kind is unsupported.
func reflectBinaryOp(
	a, b reflect.Value,
	intOp func(int64, int64) int64,
	floatOp func(float64, float64) float64,
	stringOp func(string, string) string,
) reflect.Value {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intOp != nil {
			result := intOp(a.Int(), b.Int())
			out := reflect.New(a.Type()).Elem()
			out.SetInt(result)
			return out
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if intOp != nil {
			result := intOp(int64(a.Uint()), int64(b.Uint())) //nolint:gosec // complex bit manipulation
			out := reflect.New(a.Type()).Elem()
			out.SetUint(uint64(result)) //nolint:gosec // complex bit manipulation
			return out
		}
	case reflect.Float32, reflect.Float64:
		if floatOp != nil {
			result := floatOp(a.Float(), b.Float())
			out := reflect.New(a.Type()).Elem()
			out.SetFloat(result)
			return out
		}
	case reflect.String:
		if stringOp != nil {
			result := stringOp(a.String(), b.String())
			out := reflect.New(a.Type()).Elem()
			out.SetString(result)
			return out
		}
	}
	return reflect.Value{}
}

// reflectCompare performs an ordered comparison of two reflect.Values.
//
// Takes a (reflect.Value) which specifies the left value to compare.
// Takes b (reflect.Value) which specifies the right value to compare.
//
// Returns -1 if a < b, 0 if a == b, or 1 if a > b.
func reflectCompare(a, b reflect.Value) int {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		av, bv := a.Int(), b.Int()
		if av < bv {
			return -1
		} else if av > bv {
			return 1
		}
		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		av, bv := a.Uint(), b.Uint()
		if av < bv {
			return -1
		} else if av > bv {
			return 1
		}
		return 0
	case reflect.Float32, reflect.Float64:
		av, bv := a.Float(), b.Float()
		if av < bv {
			return -1
		} else if av > bv {
			return 1
		}
		return 0
	case reflect.String:
		av, bv := a.String(), b.String()
		if av < bv {
			return -1
		} else if av > bv {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// typeAssertKindCompatible checks if two types are compatible for type
// assertions despite being different named types, handling the register
// bank boxing issue where int is stored as int64 internally.
//
// Takes source (reflect.Type) which specifies the source type from the
// register bank.
// Takes dst (reflect.Type) which specifies the destination type from
// the type assertion.
//
// Returns true if the types are kind-compatible for assertion, or
// false otherwise.
func typeAssertKindCompatible(source, dst reflect.Type) bool {
	sk, dk := source.Kind(), dst.Kind()

	if isSignedInt(sk) && isSignedInt(dk) {
		return true
	}

	if isUnsignedInt(sk) && isUnsignedInt(dk) {
		return true
	}

	if isFloat(sk) && isFloat(dk) {
		return true
	}
	return false
}

// isSignedInt reports whether the reflect kind is a signed integer
// type.
//
// Takes k (reflect.Kind) which specifies the kind to check.
//
// Returns true if k is a signed integer kind, or false otherwise.
func isSignedInt(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Int64
}

// isUnsignedInt reports whether the reflect kind is an unsigned
// integer type.
//
// Takes k (reflect.Kind) which specifies the kind to check.
//
// Returns true if k is an unsigned integer kind, or false otherwise.
func isUnsignedInt(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uintptr
}

// isFloat reports whether the reflect kind is a floating-point type.
//
// Takes k (reflect.Kind) which specifies the kind to check.
//
// Returns true if k is a floating-point kind, or false otherwise.
func isFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

// boolToInt64 converts a boolean to an int64, returning 1 for true
// and 0 for false.
//
// Takes b (bool) which specifies the boolean value to convert.
//
// Returns 1 if b is true, or 0 if b is false.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
