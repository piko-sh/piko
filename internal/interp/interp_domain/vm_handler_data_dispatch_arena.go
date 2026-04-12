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
	"maps"
	"reflect"
	"strings"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// copyReflectValue returns an independent copy of v with freshly allocated backing.
//
// Used by snapshot paths (return-value placement and the deref-and-snapshot variant of
// opDeref) to break aliasing between caller and heap memory: assigning the result to a
// general register gives Go-style value-copy semantics even though reflect.Value normally
// shares its addressable backing storage with the source. For pointer and interface kinds
// this preserves the shared pointee (only the header is copied), which matches Go's
// pointer-copy semantics. For slices, maps, channels, and funcs the header is copied; the
// underlying buffer or runtime object remains shared, again matching Go.
//
// Takes v (reflect.Value) which must be Valid; the caller checks.
//
// Returns a reflect.Value of the same Type as v whose memory is owned by a fresh
// reflect.New allocation.
func copyReflectValue(v reflect.Value) reflect.Value {
	snapshot := reflect.New(v.Type()).Elem()
	snapshot.Set(v)
	return snapshot
}

// copyReflectValueArena is the arena-aware sibling of copyReflectValue.
//
// For types whose memory layout is provably pointer-free (typeIsPointerFree) it
// bump-allocates a fresh slot from arena.genericBytesSlab instead of invoking reflect.New
// (which goes through Go's mallocgc). For types containing pointers it falls back to
// copyReflectValue because writing pointers into a []byte slab makes them invisible to
// the GC. The expr_eval token{kind int, value int} hot path is the motivating case:
// ~28.9M reflect.New allocations per iteration become arena bumps. Arena is required (not
// nil-safe) so callers without arena context must use copyReflectValue.
//
// Takes arena (*RegisterArena) which is the bump arena to allocate from.
// Takes v (reflect.Value) which is the source value to copy.
//
// Returns a reflect.Value independent of v whose backing lives in the arena or on the
// heap, depending on pointer-freedom of v's type.
func copyReflectValueArena(arena *RegisterArena, v reflect.Value) reflect.Value {
	return copyReflectValueArenaWithVM(arena, nil, v)
}

// copyReflectValueArenaWithVM is the vm-aware sibling of copyReflectValueArena.
//
// When vm is non-nil, pointer-containing Struct/Array kinds bump-allocate from the vm's
// boundarySnapshotChunks slab instead of reflect.New.
//
// Takes arena (*RegisterArena) which is the bump arena; nil routes to the plain heap-copy
// path.
// Takes vm (*VM) which provides the per-type boundary snapshot slabs for
// pointer-containing composites; may be nil.
// Takes v (reflect.Value) which is the source value to copy.
//
// Returns a reflect.Value independent of v whose backing storage is chosen based on the
// source's kind and pointer-freedom.
//
// Kind-first dispatch - for the dominant kinds in expr_eval and lru_cache (Slice,
// Pointer, Interface, Map, Chan, Func) the decision is made from the kind alone, avoiding
// the per-call typeIsPointerFree lookup (~6 % cum CPU on expr_eval). Only Struct/Array
// need the recursive typeIsPointerFree walk.
func copyReflectValueArenaWithVM(arena *RegisterArena, vm *VM, v reflect.Value) reflect.Value {
	if arena == nil {
		return copyReflectValue(v)
	}
	switch v.Kind() {
	case reflect.Slice:
		return snapshotSliceHeaderArena(arena, v)
	case reflect.Pointer:
		return unsafePointerKindValue(reflectValueABIType(v.Type()), v.UnsafePointer())
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		return snapshotPointerFreeValue(arena, v)
	case reflect.Struct, reflect.Array:
		return snapshotComposite(arena, vm, v)
	default:
	}
	return copyReflectValue(v)
}

// snapshotSliceHeaderArena copies the slice header into an arena slot.
//
// The 24-byte header is copied while the backing array is shared. derefSnapshot semantics
// only need a fresh header because slices are reference types and Go's "copy on
// assignment" only copies the header, not the elements. The arena's sliceHeaderSlab is
// GC-aware (its Data field is unsafe.Pointer), so the backing array stays reachable. This
// avoids the reflect.New plus ptrTo lookup that the slow path would otherwise perform for
// slice copies.
//
// Takes arena (*RegisterArena) which provides the header slab.
// Takes v (reflect.Value) which is the source slice.
//
// Returns a reflect.Value with the arena-resident header.
func snapshotSliceHeaderArena(arena *RegisterArena, v reflect.Value) reflect.Value {
	v = ensureSliceAddressableForHeader(v)
	t := v.Type()
	ptr := reflectValuePtr(v)
	srcHeader := (*arenaSliceHeader)(ptr)
	slot := arena.AllocSliceHeader()
	slot.Data = srcHeader.Data
	slot.Len = srcHeader.Len
	slot.Cap = srcHeader.Cap
	return unsafeNewAt(reflectValueABIType(t), unsafe.Pointer(slot), reflect.Slice)
}

// snapshotPointerFreeValue allocates a fresh arena byte-slab copy of a pointer-free
// value, falling back to copyReflectValue when the type's alignment requirement exceeds
// the slab guarantee.
//
// Takes arena (*RegisterArena) which provides the byte slab.
// Takes v (reflect.Value) which is the source pointer-free value.
//
// Returns a reflect.Value with arena-backed storage holding a value copy of v.
func snapshotPointerFreeValue(arena *RegisterArena, v reflect.Value) reflect.Value {
	t := v.Type()
	align := safeconv.IntToUintptr(t.Align())
	if align == 0 {
		align = 1
	}
	if align > arenaMaxAlignment {
		return copyReflectValue(v)
	}
	pointer := arena.AllocBytes(t.Size(), align)
	snapshot := unsafeNewAt(reflectValueABIType(t), pointer, t.Kind())
	snapshot.Set(v)
	return snapshot
}

// snapshotComposite handles Struct and Array kinds, routing pointer- free composites to
// the arena byte slab and pointer-containing composites to the per-VM boundary-snapshot
// chunk slab (one mallocgc per boundaryChunkSize snapshots instead of one per snapshot).
//
// Takes arena (*RegisterArena) which provides the byte slab when the composite is
// pointer-free.
// Takes vm (*VM) which provides the boundary-snapshot slab when the composite contains
// pointers; may be nil, in which case the snapshot falls back to reflect.New.
// Takes v (reflect.Value) which is the source composite.
//
// Returns a reflect.Value independent of v.
func snapshotComposite(arena *RegisterArena, vm *VM, v reflect.Value) reflect.Value {
	t := v.Type()
	if typeIsPointerFree(t) {
		return snapshotPointerFreeValue(arena, v)
	}
	if vm != nil {
		snapshot := vm.acquireBoundarySnapshot(t)
		snapshot.Set(v)
		return snapshot
	}
	return copyReflectValue(v)
}

func init() {
	typeIsPointerFreeCache.Store(new(make(map[unsafe.Pointer]bool, typeIsPointerFreeCacheInitialHint)))
}

// resetTypeIsPointerFreeCacheForTest installs an empty cache for tests.
//
// Replaces the live cache so successive test runs cannot observe pointer-free decisions
// minted by a previous test. Used by TestMain via the umbrella ResetGlobals helper in
// export_test.go.
//
// Not exported as TestX because tests within the package call it directly through the
// export_test.go re-export. The function name's ForTest suffix follows the Go-stdlib
// convention for test-only hooks.
//
// Concurrency: takes typeIsPointerFreeMu in write mode while swapping the cache pointer.
func resetTypeIsPointerFreeCacheForTest() {
	typeIsPointerFreeMu.Lock()
	defer typeIsPointerFreeMu.Unlock()
	typeIsPointerFreeCache.Store(new(make(map[unsafe.Pointer]bool, typeIsPointerFreeCacheInitialHint)))
}

// typeIsPointerFree reports whether t's memory layout contains no GC pointers.
//
// When true it is safe to store an instance in a []byte arena without the GC missing
// pointer references. True for primitive numerics (Int*/Uint*/Float*/Bool/Complex*),
// pure-numeric structs and arrays, and recursively pure-numeric composites. False for
// Pointer, String, Slice, Map, Chan, Func, Interface, or any composite containing them.
//
// Takes t (reflect.Type) which is the type to inspect; nil returns false.
//
// Returns true when t is pointer-free, false otherwise.
func typeIsPointerFree(t reflect.Type) bool {
	if t == nil {
		return false
	}
	key := reflectValueABIType(t)
	current := typeIsPointerFreeCache.Load()
	if cached, ok := (*current)[key]; ok {
		return cached
	}
	return typeIsPointerFreeSlow(t, key)
}

// typeIsPointerFreeSlow handles the cold path: compute the answer, publish a new map via
// copy-on-write under the writer lock, then return. The reader fast path never touches
// the lock.
//
// Takes t (reflect.Type) which is the type to inspect.
// Takes key (unsafe.Pointer) which is the cache key (t's *abi.Type pointer).
//
// Returns true when t is pointer-free, false otherwise.
//
// Concurrency: takes typeIsPointerFreeMu to publish the updated cache; readers consult
// the atomic snapshot without locking.
func typeIsPointerFreeSlow(t reflect.Type, key unsafe.Pointer) bool {
	result := computeTypeIsPointerFree(t)
	typeIsPointerFreeMu.Lock()
	current := typeIsPointerFreeCache.Load()
	if cached, ok := (*current)[key]; ok {
		typeIsPointerFreeMu.Unlock()
		return cached
	}
	next := make(map[unsafe.Pointer]bool, len(*current)+1)
	maps.Copy(next, *current)
	next[key] = result
	typeIsPointerFreeCache.Store(&next)
	typeIsPointerFreeMu.Unlock()
	return result
}

// computeTypeIsPointerFree walks the type tree to determine pointer- freedom. Called once
// per unique type and cached.
//
// Takes t (reflect.Type) which is the type to inspect; nil returns false.
//
// Returns true when t and all its component types are pointer-free, false otherwise.
func computeTypeIsPointerFree(t reflect.Type) bool {
	if t == nil {
		return false
	}
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		return true
	case reflect.Array:
		return computeTypeIsPointerFree(t.Elem())
	case reflect.Struct:
		for field := range t.Fields() {
			if !computeTypeIsPointerFree(field.Type) {
				return false
			}
		}
		return true
	default:

		return false
	}
}

// valueCopyForBoundary enforces Go's value-copy semantics at general-bank boundaries.
//
// Snapshots Struct and Array kinds so callers and callees, or assignment source and
// destination, get independent storage; without it, piko's reflect.Value-as-register
// representation aliases the source's addressable backing memory, and a write through the
// destination would silently mutate the source. Reference kinds (Pointer, Slice, Map,
// Chan, Func, Interface) pass through unchanged because the reflect.Value-struct copy
// that the caller already performs gives exactly the header copy Go's semantics require:
// the pointee, slice array, map header, channel runtime object, etc., are shared.
//
// Used by copySameKindArg (function args), handleMoveGeneral (assignment moves),
// handleIndex / handleMapIndex (slice/map element reads), rangeSliceValue (range-loop
// value bind), handleChannelReceive (channel receive), handleTypeAssert (matched
// assertion), and handleGetField (struct field reads). The single rule "every store into
// general[] from a possibly-aliased reflect.Value must call this when Kind in {Struct,
// Array}" is the architectural contract that keeps the VM's value semantics aligned with
// Go's spec.
//
// Takes v (reflect.Value) which is the source value that may alias caller storage.
//
// Returns either a fresh independent reflect.Value (Struct/Array) or the input unchanged
// (everything else, including invalid values).
func valueCopyForBoundary(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Struct, reflect.Array:
		return copyReflectValue(v)
	default:
		return v
	}
}

// valueCopyForBoundaryArena is the arena-aware sibling of valueCopyForBoundary.
//
// Used at hot boundary sites (handleMoveGeneral, copyOneCallArgument,
// asmCallSetupGeneralBank trampoline). For pointer-free Struct/Array kinds, the snapshot
// is bump-allocated from arena.genericBytesSlab instead of the GC heap; for
// pointer-containing types and non-composite kinds the semantics are identical to
// valueCopyForBoundary.
//
// Takes arena (*RegisterArena) which is the bump arena to allocate from.
// Takes v (reflect.Value) which is the value to snapshot.
//
// Returns either a fresh independent reflect.Value (Struct/Array) or v unchanged.
func valueCopyForBoundaryArena(arena *RegisterArena, v reflect.Value) reflect.Value {
	return valueCopyForBoundaryArenaWithVM(arena, nil, v)
}

// valueCopyForBoundaryArenaWithVM is the vm-aware variant of valueCopyForBoundaryArena.
//
// Passes vm through so pointer-containing struct/array kinds can bump from the per-type
// boundary slab instead of reflect.New.
//
// Takes arena (*RegisterArena) which is the bump arena to allocate from.
// Takes vm (*VM) which provides the per-type boundary snapshot slabs; may be nil.
// Takes v (reflect.Value) which is the value to snapshot.
//
// Returns either a fresh independent reflect.Value (Struct/Array) or v unchanged.
func valueCopyForBoundaryArenaWithVM(arena *RegisterArena, vm *VM, v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Struct, reflect.Array:
		return copyReflectValueArenaWithVM(arena, vm, v)
	default:
		return v
	}
}

// materialiseArenaValue forces an arena-backed value onto the heap.
//
// This is the escape-copy guard that lets handleMakeMap (and other arena allocators)
// bump-allocate from the pointer-free byte slab while still respecting the contract that
// values stored in longer-lived containers (interfaces, maps, slices, heap-backed struct
// fields, channel buffers, defer captures, closure upvalues, return slots) must outlive
// the bump pointer's window.
//
// Selection rule: invalid or nil arena passes through. Struct or Array kind checks
// arena.ownsBytePointer; if owned, heap-copy via copyReflectValue (reflect.New + Set),
// the heap allocation being the price for letting the value escape. Slice kind checks
// arena.ownsSliceHeaderPointer on the header; if owned, fully unshare by allocating a
// fresh heap header AND a fresh heap backing via reflect.MakeSlice, then reflect.Copy the
// elements across. Sharing the backing (Go's natural slice semantics) is not enough
// because the backing may itself live in the arena's byte slab
// (arenaMakeStructSliceBacking) and thus escape into the same trap. Other kinds (Pointer,
// Interface, Map, Chan, Func, String, scalar) are never produced directly by the arena
// allocators, so they pass through; their underlying pointer is either inline (scalars)
// or already heap-backed (reference kinds with Go-managed pointees).
//
// Takes arena (*RegisterArena) which owns the candidate slabs.
// Takes v (reflect.Value) which is the value about to escape.
//
// Returns the heap-materialised equivalent of v when it was arena- backed; v unchanged
// otherwise.
func materialiseArenaValue(arena *RegisterArena, v reflect.Value) reflect.Value {
	if !v.IsValid() || arena == nil || !arena.isLeaf {
		return v
	}
	return materialiseArenaValueUnconditional(arena, v)
}

// materialiseArenaValueUnconditional applies the heap-escape copy regardless of
// arena.isLeaf. Used by boundary paths (return path, panic capture) where the arena is
// about to Reset whether it is the main arena or a leaf child.
//
// Takes arena (*RegisterArena) which owns the candidate slabs.
// Takes v (reflect.Value) which is about to escape the arena.
//
// Returns a heap-materialised equivalent when arena-backed; v unchanged otherwise.
func materialiseArenaValueUnconditional(arena *RegisterArena, v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Struct, reflect.Array:
		if !arena.ownsBytePointer(reflectValuePtr(v)) {
			return v
		}
		return copyReflectValue(v)
	case reflect.Slice:
		return materialiseArenaSliceUnconditional(arena, v)
	case reflect.String:
		s := v.String()
		if !arena.ownsString(s) {
			return v
		}
		return reflect.ValueOf(strings.Clone(s))
	default:
		return v
	}
}

// materialiseArenaSliceUnconditional severs slice arena aliasing.
//
// Handles the slice branch of materialiseArenaValueUnconditional. When the slice header
// itself lives outside the arena, only element contents are recursively materialised so
// heap-promoted []string with arena-backed bytes is preserved. When the header is
// arena-resident, a fresh copy is allocated and elements are materialised through the new
// backing.
//
// Use this when the captured slice must outlive the arena (goroutine launches that copy
// args across arenas, eval-result returns crossing the runEntrypointFunction boundary,
// panic-info capture). The fresh backing severs the arena lifetime but breaks
// Data-pointer aliasing with the source slice. For same-arena lifetime captures (defer
// args, simpleDefer args) prefer materialiseArenaSliceAliasing, which rewraps the header
// on the heap while keeping Data pointed at the original backing so mutations through the
// captured slice propagate to the caller's view.
//
// Takes arena (*RegisterArena) which owns the candidate slabs.
// Takes v (reflect.Value) which is the slice to materialise.
//
// Returns reflect.Value which is the heap-detached slice.
func materialiseArenaSliceUnconditional(arena *RegisterArena, v reflect.Value) reflect.Value {
	if !arena.ownsSliceHeaderPointer(reflectValuePtr(v)) {
		n := v.Len()
		for i := range n {
			elem := v.Index(i)
			materialised := materialiseArenaValueUnconditional(arena, elem)
			if materialised.IsValid() && materialised != elem && elem.CanSet() {
				elem.Set(materialised)
			}
		}
		return v
	}
	if asBytes, ok := reflect.TypeAssert[[]byte](v); ok {
		destination := make([]byte, len(asBytes))
		copy(destination, asBytes)
		return reflect.ValueOf(destination)
	}
	t := v.Type()
	n := v.Len()
	fresh := reflect.MakeSlice(t, n, n)
	reflect.Copy(fresh, v)
	for i := range n {
		elem := fresh.Index(i)
		materialised := materialiseArenaValueUnconditional(arena, elem)
		if materialised.IsValid() && materialised != elem {
			elem.Set(materialised)
		}
	}
	return fresh
}

// materialiseArenaValueAliasing detaches headers but preserves aliasing.
//
// Same-arena-lifetime sibling of materialiseArenaValueUnconditional. The semantic
// difference is the slice branch: instead of fresh-copying the backing array (which
// severs Data-pointer aliasing with the caller's view), the captured slice keeps its
// original Data pointer and only its slice header escapes the arena's sliceHeaderSlab.
// This preserves Go's shared-slice semantics so mutations through the captured slice
// propagate back to the caller, a hard requirement for defer's `f(s)` style where `s` is
// a slice and the defer body mutates `s[i]` (see SYNTHESIS_V2 section 1 ARCH5 plus
// TestEvalDefer/defer_named_function).
//
// Callers whose captured value's lifetime stays bounded by the arena's lifetime can use
// this helper, including defer-arg capture (defers run before arena.Reset per the
// runEntrypointFunction lifecycle) and any other same-Eval-call snapshot. Cross-arena
// escapes must continue using materialiseArenaValueUnconditional which severs aliasing
// for lifetime safety.
//
// Takes arena (*RegisterArena) which owns the candidate slabs.
// Takes v (reflect.Value) which is the value to capture.
//
// Returns reflect.Value whose .ptr is heap-resident (survives arena.Reset of the
// sliceHeaderSlab) but whose Data still aliases the caller's original backing.
func materialiseArenaValueAliasing(arena *RegisterArena, v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Struct, reflect.Array:
		if !arena.ownsBytePointer(reflectValuePtr(v)) {
			return v
		}
		return copyReflectValue(v)
	case reflect.Slice:
		return materialiseArenaSliceAliasing(arena, v)
	case reflect.String:
		s := v.String()
		if !arena.ownsString(s) {
			return v
		}
		return reflect.ValueOf(strings.Clone(s))
	default:
		return v
	}
}

// materialiseArenaSliceAliasing rewraps an arena-resident slice header on the heap while
// keeping the original Data pointer. Mutations through the resulting reflect.Value
// propagate to the caller's slice (Go's defer-and-mutate semantics).
//
// When the header is NOT arena-resident, recursively materialises element contents
// in-place and returns v unchanged (mirrors materialiseArenaSliceUnconditional's
// same-branch behaviour for heap-resident headers).
//
// Takes arena (*RegisterArena) which owns the slice-header slab.
// Takes v (reflect.Value) which is the slice value to materialise.
//
// Returns a reflect.Value with heap-resident header and original Data pointer.
func materialiseArenaSliceAliasing(arena *RegisterArena, v reflect.Value) reflect.Value {
	if !arena.ownsSliceHeaderPointer(reflectValuePtr(v)) {
		n := v.Len()
		for i := range n {
			elem := v.Index(i)
			materialised := materialiseArenaValueAliasing(arena, elem)
			if materialised.IsValid() && materialised != elem && elem.CanSet() {
				elem.Set(materialised)
			}
		}
		return v
	}

	header := &arenaSliceHeader{
		Data: unsafe.Pointer(v.Pointer()),
		Len:  v.Len(),
		Cap:  v.Cap(),
	}
	return unsafeNewAt(reflectValueABIType(v.Type()), unsafe.Pointer(header), reflect.Slice)
}

// Pointer materialisation must preserve identity, not duplicate: deep-copying an
// arena-backed pointee to the heap would break pointer aliasing because the original
// arena value and the copied heap value would diverge on subsequent writes.
// handleAllocIndirect's arena routing is therefore gated on the program staying within
// one top- level execution (the arena Reset happens between executions).

// materialiseAnyForArena returns a heap-materialised copy of v when v is arena-backed, so
// the value can safely outlive the arena. Used by the panic-handling paths and the
// return-value boundary to detach captured values from arena slabs before the arena
// Reset.
//
// Recursively walks Map (rebuilding with materialised keys/values), Interface
// (re-wrapping the materialised concrete value), Struct, Array, Slice, and String.
// Pointer/Chan/Func/UnsafePointer are preserved by identity (deep-copying would break
// aliasing semantics that Go programs may rely on); the caller is responsible for not
// returning Pointer values that reference arena memory across the arena Reset boundary.
//
// Takes arena (*RegisterArena) which owns the candidate slabs.
// Takes v (any) which is the candidate value.
//
// Returns the heap-materialised equivalent when arena-backed; v unchanged otherwise.
func materialiseAnyForArena(arena *RegisterArena, v any) any {
	if v == nil || arena == nil {
		return v
	}
	if s, ok := v.(string); ok {
		return materialiseStringUnconditional(arena, s)
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return v
	}
	return walkArenaEscape(arena, rv).Interface()
}

// walkArenaEscape is the reflect-driven recursive helper used by materialiseAnyForArena.
// For each kind it detaches arena-backed storage by allocating fresh heap memory and
// copying the contents through (cleansing nested arena references as it goes).
//
// Takes arena (*RegisterArena) which owns the candidate slabs.
// Takes v (reflect.Value) which is the value to walk.
//
// Returns the heap-detached equivalent of v.
func walkArenaEscape(arena *RegisterArena, v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if !arena.ownsString(s) {
			return v
		}
		return reflect.ValueOf(strings.Clone(s)).Convert(v.Type())
	case reflect.Struct, reflect.Array, reflect.Slice:
		return materialiseArenaValueUnconditional(arena, v)
	case reflect.Map:
		if v.IsNil() {
			return v
		}
		fresh := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			fresh.SetMapIndex(walkArenaEscape(arena, iter.Key()), walkArenaEscape(arena, iter.Value()))
		}
		return fresh
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		inner := walkArenaEscape(arena, v.Elem())
		if !inner.IsValid() {
			return v
		}
		wrapped := reflect.New(v.Type()).Elem()
		wrapped.Set(inner)
		return wrapped
	default:
		return v
	}
}

// handleConvert performs a type conversion on a general register value using the target
// type from the function's type table, or allocates a new pointer.
//
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleConvert(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	if instruction.c == 1 {
		registers.general[instruction.a] = reflect.New(reflectType)
	} else {
		source := registers.general[instruction.b]
		if !source.IsValid() {
			vmPanicInvalidRegister("handleConvert", "source", instruction.b, instruction, frame, registers)
		}
		if unsafePointerConvertNeeded(source.Type(), reflectType) {
			registers.general[instruction.a] = convertUnsafePointer(source, reflectType)
		} else if converted, ok := saturatingFloatToIntConvert(vm.arena, source, reflectType); ok {
			registers.general[instruction.a] = converted
		} else {
			registers.general[instruction.a] = source.Convert(reflectType)
		}
	}
	return opContinue
}

// saturatingFloatToIntConvert performs a saturating float to int convert.
//
// Uses Go's native cast (which saturates to the destination width's min and max when the
// float is out-of-range) instead of reflect.Value.Convert (which returns 0 for
// out-of-range floats). Required for Go-spec parity on snippets like `int32(1e20)` which
// Go saturates to MinInt32 but reflect.Convert produces 0.
//
// Takes arena (*RegisterArena) which provides arena-backed storage for the resulting
// reflect.Value when applicable.
// Takes source (reflect.Value) which is the value being converted.
// Takes destination (reflect.Type) which is the requested target type.
//
// Returns reflect.Value which is the saturated conversion result on the float-to-int fast
// path.
// Returns bool which reports whether the fast path applied; false for non-float sources,
// non-integer destinations, or unsupported widths.
func saturatingFloatToIntConvert(arena *RegisterArena, source reflect.Value, destination reflect.Type) (reflect.Value, bool) {
	if !source.IsValid() {
		return reflect.Value{}, false
	}
	sourceKind := source.Kind()
	if sourceKind != reflect.Float32 && sourceKind != reflect.Float64 {
		return reflect.Value{}, false
	}
	f := source.Float()
	destinationKind := destination.Kind()
	if arena == nil {
		return saturatingFloatToIntConvertFallback(f, destination, destinationKind)
	}
	destinationABI := reflectValueABIType(destination)
	switch destinationKind {
	case reflect.Int:
		slot := arena.AllocIntBox(int64(int(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Int), true
	case reflect.Int8:
		slot := arena.AllocIntBox(int64(int8(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Int8), true
	case reflect.Int16:
		slot := arena.AllocIntBox(int64(int16(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Int16), true
	case reflect.Int32:
		slot := arena.AllocIntBox(int64(int32(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Int32), true
	case reflect.Int64:
		slot := arena.AllocIntBox(int64(f))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Int64), true
	case reflect.Uint:
		slot := arena.AllocUintBox(uint64(uint(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Uint), true
	case reflect.Uint8:
		slot := arena.AllocUintBox(uint64(uint8(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Uint8), true
	case reflect.Uint16:
		slot := arena.AllocUintBox(uint64(uint16(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Uint16), true
	case reflect.Uint32:
		slot := arena.AllocUintBox(uint64(uint32(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Uint32), true
	case reflect.Uint64:
		slot := arena.AllocUintBox(uint64(f))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Uint64), true
	case reflect.Uintptr:
		slot := arena.AllocUintBox(uint64(uintptr(f)))
		return unsafeNewAt(destinationABI, unsafe.Pointer(slot), reflect.Uintptr), true
	default:
	}
	return reflect.Value{}, false
}

// saturatingFloatToIntConvertFallback handles the non-arena code path (test scaffolds
// where vm.arena is nil). Pays the reflect.ValueOf + Convert allocations that the arena
// path eliminates; preserved so test build paths still produce correct results.
//
// Takes f (float64) the source float value.
// Takes destination (reflect.Type) the target integer type.
// Takes destinationKind (reflect.Kind) cached destination kind to avoid the second
// .Kind() call.
//
// Returns the converted reflect.Value and true when destinationKind is an integer kind;
// zero value and false otherwise.
func saturatingFloatToIntConvertFallback(f float64, destination reflect.Type, destinationKind reflect.Kind) (reflect.Value, bool) {
	switch destinationKind {
	case reflect.Int:
		return reflect.ValueOf(int(f)).Convert(destination), true
	case reflect.Int8:
		return reflect.ValueOf(int8(f)).Convert(destination), true
	case reflect.Int16:
		return reflect.ValueOf(int16(f)).Convert(destination), true
	case reflect.Int32:
		return reflect.ValueOf(int32(f)).Convert(destination), true
	case reflect.Int64:
		return reflect.ValueOf(int64(f)).Convert(destination), true
	case reflect.Uint:
		return reflect.ValueOf(uint(f)).Convert(destination), true
	case reflect.Uint8:
		return reflect.ValueOf(uint8(f)).Convert(destination), true
	case reflect.Uint16:
		return reflect.ValueOf(uint16(f)).Convert(destination), true
	case reflect.Uint32:
		return reflect.ValueOf(uint32(f)).Convert(destination), true
	case reflect.Uint64:
		return reflect.ValueOf(uint64(f)).Convert(destination), true
	case reflect.Uintptr:
		return reflect.ValueOf(uintptr(f)).Convert(destination), true
	default:
	}
	return reflect.Value{}, false
}

// unsafePointerConvertNeeded reports whether a conversion between source and destination
// types requires unsafe.Pointer intermediation.
//
// Takes source (reflect.Type) which is the source type to check.
// Takes destination (reflect.Type) which is the destination type to check.
//
// Returns bool indicating whether unsafe.Pointer intermediation is needed.
func unsafePointerConvertNeeded(source, destination reflect.Type) bool {
	sourceIsUnsafePointer := source == unsafePointerType
	destinationIsUnsafePointer := destination == unsafePointerType
	if !sourceIsUnsafePointer && !destinationIsUnsafePointer {
		return false
	}
	return sourceIsUnsafePointer != destinationIsUnsafePointer
}

// convertUnsafePointer performs an unsafe.Pointer conversion between a pointer type and
// unsafe.Pointer, or vice versa.
//
// Takes source (reflect.Value) which is the source value to convert.
// Takes destinationType (reflect.Type) which is the target type to convert to.
//
// Returns reflect.Value holding the converted pointer value.
func convertUnsafePointer(source reflect.Value, destinationType reflect.Type) reflect.Value {
	if destinationType == unsafePointerType {
		return reflect.ValueOf(unsafe.Pointer(source.Pointer())) //nolint:gosec // *T -> unsafe.Pointer
	}
	if destinationType.Kind() == reflect.Uintptr {
		converted := reflect.New(destinationType).Elem()
		converted.SetUint(uint64(uintptr(unsafe.Pointer(source.Pointer())))) //nolint:gosec // unsafe.Pointer -> uintptr
		return converted
	}
	pointer := unsafe.Pointer(source.Pointer()) //nolint:gosec // unsafe.Pointer -> *T
	return reflect.NewAt(destinationType.Elem(), pointer)
}
