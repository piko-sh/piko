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
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	// derefSnapshot is the opDeref operand-c sentinel requesting a freestanding value copy.
	//
	// The result has the same dynamic type as the source but its backing storage is a fresh
	// reflect.New allocation, so mutations to the source's heap memory (typically by
	// deferred closures) cannot leak into the copy.
	//
	// snapshotReturnValueIfNeeded uses this when placing heap-promoted general-bank locals
	// into return slots: the source has already been dereffed by emitIndirectRead, and what
	// we need at the return slot is a value-copy of that dereffed view (slice header,
	// struct, pointer, etc.) so the caller observes Go's "copy on assignment" semantics.
	//
	// The default operand value 0 keeps opDeref's existing live-view semantics, which
	// receiver-side pointer dispatch and field-address takers depend on to write through to
	// the original memory.
	derefSnapshot uint8 = 1

	// maxTypedHandleCacheEntries caps the per-VM typed channel handle cache.
	//
	// Stops a pathological program creating many short-lived channels from growing the cache
	// without bound. When the cap is reached, new channels still dispatch correctly via the
	// typed fast path on first use, but the typed handle is not memoised; subsequent ops on
	// those channels pay the reflect.Value extraction each time. The cap is chosen to
	// comfortably accommodate normal programs (which use tens of channels at most) while
	// bounding worst-case memory.
	maxTypedHandleCacheEntries = 1024

	// maxMethodCacheEntries caps the per-VM method-index cache.
	//
	// Programs that observe many distinct types or method names will see normal correct
	// dispatch beyond the cap, just at the cost of repeating reflect.Type.MethodByName each
	// call. Sized to comfortably exceed typical programs while bounding worst-case memory.
	maxMethodCacheEntries = 4096

	// initialTypedHandleCacheCapacity sizes the typedHandleCache map on first use. Eight
	// matches the typical channel/map count in early program startup and avoids two or three
	// growth rehashes for realistic workloads while keeping the cold-start memory cost low.
	initialTypedHandleCacheCapacity = 8

	// initialMapKeyScratchCapacity sizes the mapKeyScratch reuse cache on first insert. Most
	// programs use a handful of map key types (often just int and string) so a small initial
	// map fits the working set.
	initialMapKeyScratchCapacity = 4

	// typeAssertModePanic is the `x.(T)` mode where non-match panics with an interpreted
	// "interface conversion" message.
	typeAssertModePanic uint8 = 1

	// typeAssertModeTypeSwitch is a type-switch case probe.
	//
	// On non-match the destination register is left untouched. A multi-type case (`case int,
	// int64:`) emits one probe per listed type into a shared destination register;
	// clobbering it with a zero on the non-matching probes would destroy the value written
	// by the matching probe. Only ok is cleared.
	typeAssertModeTypeSwitch uint8 = 2

	// addrSourceStable is the operand-C marker for opAddr on heap-stable sources.
	//
	// Emitted by the compiler at opAddr sites whose source value lives inside a heap-stable
	// container (slice/array element loaded via opIndex, etc.). At those sites the
	// escape-promote copy in handleAddr breaks aliasing: a copy would produce a pointer to a
	// separate location, and mutations through that pointer would not be visible to
	// subsequent reads of the same slice/array element. The default value 0 preserves the
	// original escape-promote behaviour for locals whose address is taken.
	addrSourceStable uint8 = 1

	// typeIsPointerFreeCacheInitialHint is the starting capacity for the typeIsPointerFree
	// cache map. Sized for the typical interpreter session's distinct-type count; the map
	// grows automatically.
	typeIsPointerFreeCacheInitialHint = 64
)

var (
	// unsafePointerType holds the reflect.Type for unsafe.Pointer, used to detect pointer
	// conversions.
	//
	//nolint:gochecknoglobals // cached reflect type; immutable after package init
	unsafePointerType = reflect.TypeFor[unsafe.Pointer]()

	// typeIsPointerFreeCache memoises typeIsPointerFree decisions per type.
	//
	// Decisions are stable so the cache is write-once-per-type and read-many. The canonical
	// map sits behind an atomic.Pointer and is rewritten copy-on-write on insert so the hot
	// read path is a single atomic load + map lookup, with no lock and no traversal. The
	// map's key is the type's *abi.Type pointer so equality is one word.
	typeIsPointerFreeCache atomic.Pointer[map[unsafe.Pointer]bool]

	// typeIsPointerFreeMu serialises copy-on-write updates to typeIsPointerFreeCache; the
	// read path is lock-free.
	typeIsPointerFreeMu sync.Mutex
)

// selectCaseInfo tracks the destination register for a select receiver case.
type selectCaseInfo struct {
	// destinationRegister is the register index where the received value is stored.
	destinationRegister uint8

	// destinationKind identifies which typed register bank receives the value.
	destinationKind registerKind

	// hasOk reports whether the recv case captures the comma-ok boolean.
	hasOk bool

	// okRegister is the int register that receives the comma-ok boolean (1 if value was
	// received from a channel, 0 if the channel was closed).
	okRegister uint8
}

// methodCacheKey identifies a (type, method name) pair for the per-VM method index cache
// used by handleGetMethod.
type methodCacheKey struct {
	// typ is the reflect type of the receiver.
	typ reflect.Type

	// name is the method name being looked up.
	name string
}

// boundMethodVM holds the captured state for invoking a bound method or method expression
// in a fresh child VM.
type boundMethodVM struct {
	// vm is the parent VM providing shared context and function table.
	vm *VM

	// callee is the compiled function to invoke as the method body.
	callee *CompiledFunction

	// rootFunctionOverride pins the rootFunction the child VM will execute under, overriding
	// the parent VM's. Set for the cross-package adapter path; nil for the in-package case.
	rootFunctionOverride *CompiledFunction

	// limits carries the resource limits inherited from the parent VM.
	limits vmLimits
}

// invoke sets up a child VM, copies the receiver and arguments into registers, runs the
// callee, and returns the reflect results.
//
// The extract function converts each arg before storing (identity for bound methods, Elem
// for method expressions).
//
// Takes receiver (reflect.Value) which is the method receiver value.
// Takes arguments ([]reflect.Value) which holds the method arguments.
// Takes extract (func(reflect.Value) reflect.Value) which converts each arg.
//
// Returns []reflect.Value which represents the method's return values, or zero-valued
// result slots when the child VM errors. Errors are recorded on the parent VM's
// evalError.
//
// Panics when the child VM raises an interpreted panic and an upstream native dispatch
// frame is still active, so the panic reaches an interpreted defer/recover.
func (b *boundMethodVM) invoke(receiver reflect.Value, arguments []reflect.Value, extract func(reflect.Value) reflect.Value) []reflect.Value {
	childVM := newVM(b.vm.ctx, b.vm.globals, b.vm.symbols)
	childVM.reentrantInterpreterVM = true
	childVM.limits = b.limits
	if b.rootFunctionOverride != nil {
		childVM.functions = b.rootFunctionOverride.functions
		childVM.rootFunction = b.rootFunctionOverride
	} else {
		childVM.functions = b.vm.functions
		childVM.rootFunction = b.vm.rootFunction
	}
	arena := GetRegisterArena()
	arena.isLeaf = true
	childVM.arena = arena
	childVM.callStack = arena.frameStack()
	defer childVM.finishWatcher()
	defer func() {
		childVM.callStack = nil
		PutRegisterArena(arena)
	}()
	childVM.pushFrame(b.callee)
	f := childVM.currentFrame()
	placeBoundMethodReceiver(&f.registers, b.callee, receiver)
	setMethodArgs(&f.registers, b.callee, arguments, extract)
	result, err := childVM.run(0)
	allResults := childVM.evalAllResults
	childVM.evalAllResults = nil
	if err != nil {
		if childVM.panicValue != nil && b.vm.globals != nil && b.vm.globals.dispatchDepth.Load() > 0 {
			panic(childVM.panicValue)
		}
		if b.vm.evalError == nil {
			b.vm.evalError = fmt.Errorf("bound method invocation failed: %w", err)
		}
		return reflectResults(nil, b.callee.resultKinds)
	}
	return reflectResultsMulti(result, allResults, b.callee.resultKinds)
}

// rangeNextContext bundles the decoded extension-word parameters needed by the
// per-collection-type range-next helpers.
type rangeNextContext struct {
	// doneDestination is the int register index that receives 1 when iterating or 0 when
	// exhausted.
	doneDestination uint8

	// hasKey indicates whether the range loop binds a key variable.
	hasKey bool

	// hasValue indicates whether the range loop binds a value variable.
	hasValue bool

	// keyInstruction encodes the destination register and kind for the key.
	keyInstruction instruction

	// valInstruction encodes the destination register and kind for the value.
	valInstruction instruction
}

// cachedAddr returns v.Addr() but bypasses reflect.PointerTo's sync.Map lookup via a
// one-slot per-VM cache. Hot for the `&Struct{...}` ADDR opcode chain in tight
// constructors.
//
// Takes v (reflect.Value) which the caller has already proved CanAddr.
//
// Returns a Pointer-kind Value equivalent to v.Addr().
func (vm *VM) cachedAddr(v reflect.Value) reflect.Value {
	elemType := v.Type()
	ptrType := vm.ptrTypeCacheValue
	if vm.ptrTypeCacheKey != elemType {
		ptrType = reflect.PointerTo(elemType)
		vm.ptrTypeCacheKey = elemType
		vm.ptrTypeCacheValue = ptrType
	}
	return unsafePointerKindValue(reflectValueABIType(ptrType), reflectValuePtr(v))
}

// reflectResultsMulti packages a bound method's return values.
//
// Prefers the multi-return slice (allResults) when the callee declared more than one
// result; falls back to the single result value otherwise. Without it, calls like a piko
// Read body returning (int, error) only surface the int slot, so the adapter drops EOF
// and ErrCustom-style errors and io.ReadAll loops forever returning (0, nil).
//
// Takes result (any) which is the single return path's first value.
// Takes allResults ([]any) which holds every declared result (nil when the callee
// declared zero or one result).
// Takes resultKinds ([]registerKind) which describes the slot kinds.
//
// Returns []reflect.Value matching resultKinds slot-by-slot.
func reflectResultsMulti(result any, allResults []any, resultKinds []registerKind) []reflect.Value {
	if len(resultKinds) <= 1 {
		return reflectResults(result, resultKinds)
	}
	results := make([]reflect.Value, len(resultKinds))
	for i, kind := range resultKinds {
		var raw any
		if i < len(allResults) {
			raw = allResults[i]
		}
		if raw == nil {
			results[i] = zeroValueForKind(kind)
			continue
		}
		results[i] = reflect.ValueOf(raw)
	}
	return results
}

// placeBoundMethodReceiver writes the receiver into general register 0.
//
// Method receivers on piko-synth structs land in the general bank (the default), but
// receivers on named primitives (`type Colour int`, `time.Duration`, ...) live in a typed
// bank. Without this, value receivers see the zero value of their type because general[0]
// is populated but the method body reads from ints[0] / floats[0].
//
// Receiver shape: when callee is a VALUE-receiver method but the incoming reflect.Value
// is a pointer (a common occurrence in the cross-package bound-method dispatch where
// addressableMethodReceiver promotes method-less synthetic structs to `*T` to make Go's
// reflect method-set discovery succeed), deref the pointer so the body sees the struct
// value. Without this, opAllocIndirect inside the body (emitted whenever the body
// implicitly takes the address of the receiver to call a pointer-receiver helper, e.g.
// shopspring/decimal's `Truncate` -> `d.ensureInitialized()`) cannot assign a `*T` source
// into the fresh `T` heap cell and silently leaves it zero, so every arithmetic Mul/Add
// on the result then returns the zero Decimal.
//
// Takes registers (*Registers) which receives the receiver value.
// Takes callee (*CompiledFunction) which is the method body being prepared;
// callee.isPointerReceiver gates the deref normalisation.
// Takes receiver (reflect.Value) which is the receiver value to install.
func placeBoundMethodReceiver(registers *Registers, callee *CompiledFunction, receiver reflect.Value) {
	if callee != nil && !callee.isPointerReceiver &&
		receiver.IsValid() && receiver.Kind() == reflect.Pointer && !receiver.IsNil() {
		receiver = receiver.Elem()
	}
	registers.general[0] = receiver
}

// reflectResults converts a VM execution result into a slice of reflect.Value matching
// the expected result kinds for return to native callers.
//
// Takes result (any) which is the raw VM execution result.
// Takes resultKinds ([]registerKind) which describes the expected return types.
//
// Returns []reflect.Value matching the result kinds, or nil if none.
func reflectResults(result any, resultKinds []registerKind) []reflect.Value {
	if len(resultKinds) == 0 {
		return nil
	}
	if result == nil {
		results := make([]reflect.Value, len(resultKinds))
		for i, k := range resultKinds {
			results[i] = zeroValueForKind(k)
		}
		return results
	}
	return []reflect.Value{reflect.ValueOf(result)}
}

// kindDefaultReflectType returns the default reflect.Type for a given register kind, used
// to construct zero values when no result is available.
//
// Takes k (registerKind) which is the register kind to map to a type.
//
// Returns reflect.Type corresponding to the default Go type for that kind.
func kindDefaultReflectType(k registerKind) reflect.Type {
	switch k {
	case registerInt:
		return reflect.TypeFor[int64]()
	case registerFloat:
		return reflect.TypeFor[float64]()
	case registerString:
		return reflect.TypeFor[string]()
	case registerBool:
		return reflect.TypeFor[bool]()
	case registerUint:
		return reflect.TypeFor[uint64]()
	case registerComplex:
		return reflect.TypeFor[complex128]()
	default:
		return reflect.TypeFor[any]()
	}
}

// handleAddr stores the address of the source value into the destination register.
//
// When the source lives inside arena-bumped storage and operand-C is not
// addrSourceStable, the value is heap-promoted via copyReflectValueArena first so the
// resulting pointer cannot alias subsequent arena bytes. The promoted value is also
// written back to the source register so reads from the same slot continue to see the
// heap-stable value.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the source value and destination.
// Takes instruction (instruction) which encodes the source/destination register indices
// and the addrSourceStable marker.
//
// Returns opResult indicating the next execution step. The body raises a piko-internal
// panic via vmPanicInvalidRegister when the source register holds an invalid
// reflect.Value, but the keyword does not appear at this call depth.
func handleAddr(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.b]
	if !v.IsValid() {
		vmPanicInvalidRegister("handleAddr", "source", instruction.b, instruction, frame, registers)
	}
	if instruction.c != addrSourceStable &&
		vm.arena != nil && (v.Kind() == reflect.Struct || v.Kind() == reflect.Array) &&
		v.CanAddr() && vm.arena.ownsBytePointer(reflectValuePtr(v)) {
		heap := copyReflectValueArena(vm.arena, v)
		registers.general[instruction.b] = heap
		v = heap
	}
	if v.CanAddr() {
		registers.general[instruction.a] = vm.cachedAddr(v)
	} else {
		pointer := reflect.New(v.Type())
		pointer.Elem().Set(v)
		registers.general[instruction.a] = pointer

		registers.general[instruction.b] = pointer.Elem()
	}
	return opContinue
}

// handleDeref dereferences a pointer in the general register bank and stores the
// pointed-to value in the destination register.
//
// Takes registers (*Registers) which holds the pointer and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleDeref(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.b]
	if instruction.c == derefSnapshot {
		if !v.IsValid() {
			registers.general[instruction.a] = v
			return opContinue
		}
		registers.general[instruction.a] = copyReflectValueArena(vm.arena, v)
		return opContinue
	}
	if !v.IsValid() {
		vmPanicInvalidRegister("handleDeref", "source", instruction.b, instruction, frame, registers)
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: invalid memory address or nil pointer dereference"))
		}
		registers.general[instruction.a] = v.Elem()
	case reflect.Interface:
		registers.general[instruction.a] = v.Elem()
	default:
		registers.general[instruction.a] = v
	}
	return opContinue
}
