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
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// appendLimitExceededFormat is the shared fmt format string used by every alloc-limit
	// overflow path in this file. Centralising it keeps the wording consistent and satisfies
	// revive's add-constant.
	appendLimitExceededFormat = "%w: append result length %d exceeds limit %d"
)

var (
	// intSliceReflectType is the cached reflect.Type of []int64. Used by
	// handlePackInterface's typed-slice-to-general path to avoid the per-call
	// reflect.TypeFor lookup.
	intSliceReflectType = reflect.TypeFor[[]int64]()

	// floatSliceReflectType is the cached reflect.Type of []float64.
	floatSliceReflectType = reflect.TypeFor[[]float64]()

	// stringSliceReflectType is the cached reflect.Type of []string.
	stringSliceReflectType = reflect.TypeFor[[]string]()

	// boolSliceReflectType is the cached reflect.Type of []bool.
	boolSliceReflectType = reflect.TypeFor[[]bool]()

	// uintSliceReflectType is the cached reflect.Type of []uint64.
	uintSliceReflectType = reflect.TypeFor[[]uint64]()
)

// arenaWrapTypedSlice wraps a typed []byte (or any slice whose reflect.Type matches
// sliceType) into a reflect.Value backed by an arena slice header slot. Saves the
// ~24-byte heap slice header allocation that reflect.ValueOf does for slice values via
// runtime.convTslice (~2.5 % cum CPU on expr_eval, ~330 MB allocs).
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes data (unsafe.Pointer) which is the slice's backing array pointer (or nil for an
// empty slice).
// Takes length (int) which is the slice's len.
// Takes capacity (int) which is the slice's cap.
// Takes sliceType (reflect.Type) which is the typed slice's reflect.Type (e.g.
// reflect.TypeFor[[]byte]()).
//
// Returns a reflect.Value of kind Slice referring to the arena slice-header slot.
func arenaWrapTypedSlice(arena *RegisterArena, data unsafe.Pointer, length, capacity int, sliceType reflect.Type) reflect.Value {
	slot := arena.AllocSliceHeader()
	slot.Data = data
	slot.Len = length
	slot.Cap = capacity
	return unsafeNewAt(reflectValueABIType(sliceType), unsafe.Pointer(slot), reflect.Slice)
}

// arenaWrapByteSlice is the []byte specialisation of arenaWrapTypedSlice. Inlined into
// the byte-builder hot path of expr_eval (~3M calls per profile run).
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes s ([]byte) which is the source byte slice whose header is wrapped.
//
// Returns a reflect.Value of kind Slice referring to the arena slice-header slot.
func arenaWrapByteSlice(arena *RegisterArena, s []byte) reflect.Value {
	var data unsafe.Pointer
	if cap(s) > 0 {
		data = unsafe.Pointer(unsafe.SliceData(s))
	}
	return arenaWrapTypedSlice(arena, data, len(s), cap(s), reflect.TypeFor[[]byte]())
}

// arenaWrapMakeBacking is the generic helper used by arenaMakeSliceBacking to wrap an
// arena-allocated typed backing (any scalar element type) into an arena slice-header
// slot. Avoids the per-call reflect.ValueOf heap slice-header allocation that
// reflect.ValueOf(backing[:length:capacity]) would otherwise incur via
// runtime.convTslice.
//
// The capacity == 0 guard mirrors arenaWrapByteSlice: empty slices have no backing
// pointer to wire into the slot's Data field.
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes backing ([]T) which is the arena-allocated backing array.
// Takes length (int), capacity (int) which are the make() arguments.
// Takes reflectType (reflect.Type) which is the slice type to attach to the resulting
// reflect.Value. Passed through verbatim so named types (e.g. type MyBytes []byte) keep
// their original identity.
//
// Returns a reflect.Value of kind Slice whose header lives in the arena slab.
func arenaWrapMakeBacking[T any](arena *RegisterArena, backing []T, length, capacity int, reflectType reflect.Type) reflect.Value {
	var data unsafe.Pointer
	if capacity > 0 {
		data = unsafe.Pointer(unsafe.SliceData(backing))
	}
	return arenaWrapTypedSlice(arena, data, length, capacity, reflectType)
}

// arenaWrapTypedSliceFromSource builds a result reflect.Value for a grown typed slice by
// reusing the source value's abi type pointer directly, avoiding both reflect.ValueOf's
// per-call convTslice mallocgc AND the cached reflect.TypeFor[[]T]() lookup. Used by the
// typed-append handlers (handleAppendString/Int/Float/Bool/Uint) to box the post-grow
// slice header through the arena slab.
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes sourceValue (reflect.Value) which is the input slice value whose abi type is
// reused for the result.
// Takes data (unsafe.Pointer) which is the grown slice's backing pointer.
// Takes length (int) which is the grown slice's len.
// Takes capacity (int) which is the grown slice's cap.
//
// Returns a reflect.Value of kind Slice with the same dynamic type as sourceValue, no
// heap allocation.
func arenaWrapTypedSliceFromSource(arena *RegisterArena, sourceValue reflect.Value, data unsafe.Pointer, length, capacity int) reflect.Value {
	srcType := wrapTypedSliceSrcType(sourceValue)
	slot := arena.AllocSliceHeader()
	slot.Data = data
	slot.Len = length
	slot.Cap = capacity
	return unsafeNewAt(srcType, unsafe.Pointer(slot), reflect.Slice)
}

// sliceDataPtrString returns the backing-array pointer for the given []string, or nil
// when empty. Wrapped so callers don't need to pull in unsafe.SliceData at every site.
//
// Takes s ([]string) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrString(s []string) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrInt64 returns the backing-array pointer for the given []int64, or nil when
// empty.
//
// Takes s ([]int64) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrInt64(s []int64) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrFloat64 returns the backing-array pointer for the given []float64, or nil
// when empty.
//
// Takes s ([]float64) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrFloat64(s []float64) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrBool returns the backing-array pointer for the given []bool, or nil when
// empty.
//
// Takes s ([]bool) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrBool(s []bool) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrUint64 returns the backing-array pointer for the given []uint64, or nil
// when empty.
//
// Takes s ([]uint64) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrUint64(s []uint64) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrInt returns the backing-array pointer for the given []int, or nil when
// empty.
//
// Takes s ([]int) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrInt(s []int) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrUint returns the backing-array pointer for the given []uint, or nil when
// empty.
//
// Takes s ([]uint) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrUint(s []uint) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrUint32 returns the backing-array pointer for the given []uint32, or nil
// when empty.
//
// Takes s ([]uint32) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrUint32(s []uint32) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrUint16 returns the backing-array pointer for the given []uint16, or nil
// when empty.
//
// Takes s ([]uint16) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrUint16(s []uint16) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// sliceDataPtrUintptr returns the backing-array pointer for the given []uintptr, or nil
// when empty.
//
// Takes s ([]uintptr) which is the slice whose data pointer is read.
//
// Returns the unsafe.Pointer to the backing array, or nil when the slice has zero
// capacity.
func sliceDataPtrUintptr(s []uintptr) unsafe.Pointer {
	if cap(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s))
}

// appendFastPath attempts a type-assertion fast path for common concrete slice types to
// avoid reflect.Append overhead.
//
// When arena is non-nil, the grow path for element kinds matching an arena backing slab
// routes through the arenaAppend* helpers so the new backing lands in the arena instead
// of triggering mallocgc.
//
// Takes arena (*RegisterArena) which provides the arena-aware grow helpers (or nil to opt
// out).
// Takes sliceValue (reflect.Value) which is the slice to append to.
// Takes element (reflect.Value) which is the element to append.
//
// Returns reflect.Value and bool; true if a fast path was taken.
//
// []byte is checked first because expr_eval's `*output = append (*output, '(')`
// byte-builder pattern is the single hottest general-bank append; ordering the byte
// branch first avoids the TypeAssert misses that would otherwise precede it.
func appendFastPath(arena *RegisterArena, sliceValue, element reflect.Value) (reflect.Value, bool) {
	if s, ok := reflect.TypeAssert[[]byte](sliceValue); ok {
		return appendByteSliceFastPath(arena, s, element)
	}
	if result, matched := appendIntSliceFastPath(arena, sliceValue, element); matched {
		return result, result.IsValid()
	}
	if result, matched := appendFloatSliceFastPath(arena, sliceValue, element); matched {
		return result, result.IsValid()
	}
	if result, matched := appendStringSliceFastPath(arena, sliceValue, element); matched {
		return result, result.IsValid()
	}
	if result, matched := appendBoolSliceFastPath(arena, sliceValue, element); matched {
		return result, result.IsValid()
	}
	return reflect.Value{}, false
}

// appendIntSliceFastPath handles the []int64 and []int element-type branches of
// appendFastPath.
//
// Takes arena (*RegisterArena), sliceValue (reflect.Value), and element (reflect.Value).
//
// Returns the wrapped result and true when sliceValue had a matching element-typed slice
// (the result is invalid when the element didn't match the slice's element type).
// Returns (zero Value, false) when sliceValue did not match either int-typed slice.
func appendIntSliceFastPath(arena *RegisterArena, sliceValue, element reflect.Value) (reflect.Value, bool) {
	if s, ok := reflect.TypeAssert[[]int64](sliceValue); ok {
		if v, ok := reflect.TypeAssert[int64](element); ok {
			grown := arenaAppendInt(arena, s, v)
			return arenaWrapTypedSliceFromSource(arena, sliceValue, sliceDataPtrInt64(grown), len(grown), cap(grown)), true
		}
		return reflect.Value{}, true
	}
	if s, ok := reflect.TypeAssert[[]int](sliceValue); ok {
		if v, ok := reflect.TypeAssert[int](element); ok {
			s = append(s, v)
			return arenaWrapTypedSliceFromSource(arena, sliceValue, sliceDataPtrInt(s), len(s), cap(s)), true
		}
		return reflect.Value{}, true
	}
	return reflect.Value{}, false
}

// appendFloatSliceFastPath handles the []float64 branch of appendFastPath.
//
// Takes arena (*RegisterArena), sliceValue (reflect.Value), and element (reflect.Value).
//
// Returns the wrapped result and true when sliceValue had a matching element-typed slice;
// otherwise (zero Value, false).
func appendFloatSliceFastPath(arena *RegisterArena, sliceValue, element reflect.Value) (reflect.Value, bool) {
	if s, ok := reflect.TypeAssert[[]float64](sliceValue); ok {
		if v, ok := reflect.TypeAssert[float64](element); ok {
			grown := arenaAppendFloat(arena, s, v)
			return arenaWrapTypedSliceFromSource(arena, sliceValue, sliceDataPtrFloat64(grown), len(grown), cap(grown)), true
		}
		return reflect.Value{}, true
	}
	return reflect.Value{}, false
}

// appendStringSliceFastPath handles the []string branch of appendFastPath.
//
// Takes arena (*RegisterArena), sliceValue (reflect.Value), and element (reflect.Value).
//
// Returns the wrapped result and true when sliceValue had a matching element-typed slice;
// otherwise (zero Value, false).
func appendStringSliceFastPath(arena *RegisterArena, sliceValue, element reflect.Value) (reflect.Value, bool) {
	if s, ok := reflect.TypeAssert[[]string](sliceValue); ok {
		if v, ok := reflect.TypeAssert[string](element); ok {
			grown := arenaAppendString(arena, s, v)
			return arenaWrapTypedSliceFromSource(arena, sliceValue, sliceDataPtrString(grown), len(grown), cap(grown)), true
		}
		return reflect.Value{}, true
	}
	return reflect.Value{}, false
}

// appendBoolSliceFastPath handles the []bool branch of appendFastPath.
//
// Takes arena (*RegisterArena), sliceValue (reflect.Value), and element (reflect.Value).
//
// Returns the wrapped result and true when sliceValue had a matching element-typed slice;
// otherwise (zero Value, false).
func appendBoolSliceFastPath(arena *RegisterArena, sliceValue, element reflect.Value) (reflect.Value, bool) {
	if s, ok := reflect.TypeAssert[[]bool](sliceValue); ok {
		if v, ok := reflect.TypeAssert[bool](element); ok {
			grown := arenaAppendBool(arena, s, v)
			return arenaWrapTypedSliceFromSource(arena, sliceValue, sliceDataPtrBool(grown), len(grown), cap(grown)), true
		}
		return reflect.Value{}, true
	}
	return reflect.Value{}, false
}

// appendByteSliceFastPath handles the []byte branch of appendFastPath, covering both the
// direct byte-typed element and the widened-integer boxing path that expr_eval emits for
// rune-literal appends.
//
// expr_eval's `append(*output, '(')` boxes the untyped rune constant as int/int64;
// without this widening the call would fall through to coerceValue + reflect.Append.
// Masking to a byte matches Go's `byte(int(x))` truncation semantics; the upstream
// compiler has already type-checked that the conversion is valid.
//
// Takes arena (*RegisterArena) which provides the byte slab when a grow is needed.
// Takes destination ([]byte) which is the slice being appended to.
// Takes element (reflect.Value) which is the element to coerce and append.
//
// Returns the wrapped result reflect.Value plus true on success, or (zero Value, false)
// when the element kind is not boxable as a byte.
func appendByteSliceFastPath(arena *RegisterArena, destination []byte, element reflect.Value) (reflect.Value, bool) {
	if v, ok := reflect.TypeAssert[byte](element); ok {
		grown := arenaAppendByte(arena, destination, v)
		return arenaWrapByteSlice(arena, grown), true
	}
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		grown := arenaAppendByte(arena, destination, safeconv.Int64ToUint8(element.Int()))
		return arenaWrapByteSlice(arena, grown), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		grown := arenaAppendByte(arena, destination, safeconv.Uint64ToUint8(element.Uint()))
		return arenaWrapByteSlice(arena, grown), true
	default:
	}
	return reflect.Value{}, false
}

// handleAppend handles the opAppend instruction by appending a general register element
// to a slice with type coercion as needed.
//
// Takes registers (*Registers) which holds the slice and element values.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppend(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	sliceValue := registers.general[instruction.b]
	element := registers.general[instruction.c]

	if !sliceValue.IsValid() {
		if !element.IsValid() {
			vm.evalError = newRuntimePanicError("interp: append to nil slice with invalid element")
			return opPanicError
		}
		sliceValue = reflect.MakeSlice(reflect.SliceOf(element.Type()), 0, 0)
	}
	if vm.limits.maxAllocSize > 0 && sliceValue.Len()+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, sliceValue.Len()+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	if result, ok := appendFastPath(vm.arena, sliceValue, element); ok {
		registers.general[instruction.a] = result
		return opContinue
	}

	if result, ok := appendGenericFastPath(vm.arena, sliceValue, element); ok {
		registers.general[instruction.a] = result
		return opContinue
	}
	elementType := sliceValue.Type().Elem()
	element = coerceValue(vm, element, elementType)
	registers.general[instruction.a] = reflect.Append(sliceValue, element)
	return opContinue
}

// appendGenericFastPath handles the spare-capacity case for arbitrary slice element
// types.
//
// When sliceCap > sliceLen and the element is type-compatible with the slice's element
// kind (exact-match or assignable), it widens the slice header to len+1, writes the
// element via reflect.Value.Set (which emits the correct GC write barriers for
// pointer-containing element types), and returns the new slice. Falls back (ok=false)
// when sliceValue is invalid or not a Slice kind, when sliceLen == sliceCap (growth left
// to reflect.Append's tuned policy), or when the element type is not assignable to the
// slice's element type (the slow path runs coerceValue).
//
// Takes arena (*RegisterArena) which provides the arena-resident header slot when
// non-nil.
// Takes sliceValue (reflect.Value) which is the slice being appended to.
// Takes element (reflect.Value) which is the element to append.
//
// Returns the new slice and true on success, or zero Value and false when the slow path
// must run.
func appendGenericFastPath(arena *RegisterArena, sliceValue, element reflect.Value) (reflect.Value, bool) {
	if !sliceValue.IsValid() || sliceValue.Kind() != reflect.Slice {
		return reflect.Value{}, false
	}
	sliceLen := sliceValue.Len()
	sliceCap := sliceValue.Cap()
	if sliceLen >= sliceCap {
		return reflect.Value{}, false
	}
	if !element.IsValid() {
		return reflect.Value{}, false
	}
	elementType := sliceValue.Type().Elem()
	elementValueType := element.Type()
	if elementValueType != elementType {
		return reflect.Value{}, false
	}
	if arena != nil {
		if sourceHeaderPtr := reflectValuePtr(sliceValue); sourceHeaderPtr != nil {
			sourceHeader := (*arenaSliceHeader)(sourceHeaderPtr)
			slot := arena.AllocSliceHeader()
			slot.Data = sourceHeader.Data
			slot.Len = sliceLen + 1
			slot.Cap = sliceCap
			extended := unsafeNewAt(reflectValueABIType(sliceValue.Type()), unsafe.Pointer(slot), reflect.Slice)

			eKind := elementType.Kind()
			if (eKind == reflect.Struct || eKind == reflect.Array) &&
				typeIsPointerFree(elementType) &&
				element.CanAddr() {
				elemSize := elementType.Size()
				destinationPointer := unsafe.Add(sourceHeader.Data, elemSize*safeconv.IntToUintptr(sliceLen))
				sourcePointer := reflectValuePtr(element)
				destination := unsafe.Slice((*byte)(destinationPointer), elemSize)
				source := unsafe.Slice((*byte)(sourcePointer), elemSize)
				copy(destination, source)
				return extended, true
			}
			extended.Index(sliceLen).Set(element)
			return extended, true
		}
	}
	extended := sliceValue.Slice(0, sliceLen+1)
	extended.Index(sliceLen).Set(element)
	return extended, true
}

// handleAppendSpread handles the opAppendSpread instruction by appending every element of
// a source slice into a destination slice via reflect.AppendSlice. This is the
// variadic-spread form `append(destination, source...)` where the spread argument is
// itself a slice.
//
// Takes registers (*Registers) which holds both slices in general.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendSpread(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	destinationSlice := registers.general[instruction.b]
	sourceSlice := registers.general[instruction.c]

	if !sourceSlice.IsValid() {
		registers.general[instruction.a] = destinationSlice
		return opContinue
	}

	if sourceSlice.Kind() == reflect.String {
		return handleAppendStringSpread(vm, registers, instruction, destinationSlice, sourceSlice)
	}
	if !destinationSlice.IsValid() {
		destinationSlice = reflect.MakeSlice(sourceSlice.Type(), 0, sourceSlice.Len())
	}
	if vm.limits.maxAllocSize > 0 && destinationSlice.Len()+sourceSlice.Len() > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, destinationSlice.Len()+sourceSlice.Len(), vm.limits.maxAllocSize)
		return opPanicError
	}

	if result, ok := appendSpreadFastPath(vm.arena, destinationSlice, sourceSlice); ok {
		registers.general[instruction.a] = result
		return opContinue
	}
	if destination, destinationOk := reflect.TypeAssert[[]byte](destinationSlice); destinationOk {
		if source, sourceOk := reflect.TypeAssert[[]byte](sourceSlice); sourceOk {
			result := arenaAppendByteSpread(vm.arena, destination, source)
			registers.general[instruction.a] = arenaWrapByteSpreadResult(vm.arena, destinationSlice, result)
			return opContinue
		}
	}
	registers.general[instruction.a] = reflect.AppendSlice(destinationSlice, sourceSlice)
	return opContinue
}

// arenaWrapByteSpreadResult constructs an arena-resident reflect.Value of slice kind that
// shares the destination's dynamic type and points at the supplied result data, length,
// and capacity.
//
// Routes the slice header through an arena slot to avoid the heap composite-literal
// allocation that the equivalent reflect.ValueOf(result) path would incur. The data may
// itself be arena-resident (e.g. when arenaAppendByteSpread grew via AllocByteBacking);
// the resulting reflect.Value remains compatible with materialise, which deep-copies on
// escape when the header points inside arena memory.
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes sourceValue (reflect.Value) whose dynamic type is reused for the result.
// Takes result ([]byte) which is the new slice's backing data and header content.
//
// Returns a reflect.Value of kind Slice referring to an arena slice- header slot.
func arenaWrapByteSpreadResult(arena *RegisterArena, sourceValue reflect.Value, result []byte) reflect.Value {
	sourceShape := (*unsafeReflectValue)(unsafe.Pointer(&sourceValue))
	slot := arena.AllocSliceHeader()
	slot.Data = unsafe.Pointer(unsafe.SliceData(result))
	slot.Len = len(result)
	slot.Cap = cap(result)
	out := unsafeReflectValue{
		typ:  sourceShape.typ,
		ptr:  unsafe.Pointer(slot),
		flag: uintptr(reflect.Slice) | flagAddr | flagIndir,
	}
	return *(*reflect.Value)(unsafe.Pointer(&out))
}

// arenaAppendByteSpread appends every byte of source to destination.
//
// When destination has spare capacity for the full spread, this is an in-place memcpy +
// slice extension. When a grow is required, the new backing comes from the arena's byte
// slab (same path as arenaAppendByte's grow case) so the result avoids mallocgc.
//
// Takes arena (*RegisterArena) which supplies the byte slab when a grow is needed.
// Takes destination ([]byte) which receives the appended bytes.
// Takes source ([]byte) which is the source whose bytes are appended.
//
// Returns the extended []byte slice header (sharing destination's backing when capacity
// allowed in-place, or pointing at a fresh arena-allocated backing after a grow).
func arenaAppendByteSpread(arena *RegisterArena, destination, source []byte) []byte {
	newLen := len(destination) + len(source)
	if newLen <= cap(destination) {
		extended := destination[:newLen]
		copy(extended[len(destination):], source)
		return extended
	}
	newCap := max(2*cap(destination), newLen)
	backing := arena.AllocByteBacking(newCap)
	copy(backing, destination)
	copy(backing[len(destination):], source)
	return backing[:newLen]
}

// arenaAppendByteFromString is the string-source twin of arenaAppendByteSpread.
//
// Supports the Go spec's special case `append(byteSlice, stringValue...)`. Go's built-in
// `copy([]byte, string)` form lets us copy the string's UTF-8 bytes without materialising
// an intermediate []byte. Grow path uses the arena byte slab, matching
// arenaAppendByteSpread's policy.
//
// Takes arena (*RegisterArena) which supplies the byte slab on grow.
// Takes destination ([]byte) which receives the appended bytes.
// Takes source (string) which is the source whose bytes are appended.
//
// Returns the extended []byte slice header (sharing destination's backing when capacity
// allowed in-place, or pointing at a fresh arena-allocated backing after a grow).
func arenaAppendByteFromString(arena *RegisterArena, destination []byte, source string) []byte {
	newLen := len(destination) + len(source)
	if newLen <= cap(destination) {
		extended := destination[:newLen]
		copy(extended[len(destination):], source)
		return extended
	}
	newCap := max(2*cap(destination), newLen)
	backing := arena.AllocByteBacking(newCap)
	copy(backing, destination)
	copy(backing[len(destination):], source)
	return backing[:newLen]
}

// handleAppendStringSpread spreads a string into a byte slice destination.
//
// This is the Go-spec special case `append(byteSlice, str...)` where the spread argument
// is a string and the destination has core type []byte. Mirrors the []byte-to-[]byte fast
// path in handleAppendSpread: routes the grow through the arena byte slab and constructs
// the result reflect.Value via the heap-slot bypass to avoid runtime.convTslice on every
// call.
//
// Takes destinationSlice (reflect.Value) which holds the destination byte slice. May be
// invalid; a fresh []byte is seeded in that case.
// Takes sourceValue (reflect.Value) which holds the string source (Kind() ==
// reflect.String).
//
// Returns opResult indicating the next execution step.
func handleAppendStringSpread(vm *VM, registers *Registers, instruction instruction, destinationSlice, sourceValue reflect.Value) opResult {
	sourceString := sourceValue.String()
	sourceLen := len(sourceString)

	if !destinationSlice.IsValid() {
		destinationSlice = reflect.MakeSlice(reflect.TypeFor[[]byte](), 0, sourceLen)
	}
	if vm.limits.maxAllocSize > 0 && destinationSlice.Len()+sourceLen > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, destinationSlice.Len()+sourceLen, vm.limits.maxAllocSize)
		return opPanicError
	}
	if destination, destinationOk := reflect.TypeAssert[[]byte](destinationSlice); destinationOk {
		result := arenaAppendByteFromString(vm.arena, destination, sourceString)
		registers.general[instruction.a] = arenaWrapByteSpreadResult(vm.arena, destinationSlice, result)
		return opContinue
	}

	destinationType := destinationSlice.Type()
	if destinationType.Elem().Kind() == reflect.Uint8 {
		typedSource := reflect.MakeSlice(destinationType, sourceLen, sourceLen)
		reflect.Copy(typedSource, sourceValue)
		registers.general[instruction.a] = reflect.AppendSlice(destinationSlice, typedSource)
		return opContinue
	}

	vm.evalError = fmt.Errorf("interp: cannot spread string into %v", destinationType)
	return opPanicError
}

// appendSpreadFastPath bypasses reflect.AppendSlice's per-call header allocation when the
// destination has spare capacity.
//
// The element copy is done by reflect.Copy on the destination[len:len+sourceLength] view,
// which uses runtime.typedslicecopy internally for correct GC barriers. The result header
// is bump-allocated from arena.sliceHeaderSlab and shares the destination's backing
// array. Falls through (ok=false) when arena is nil (callers without arena context), when
// destination.Cap() is below destination.Len() + source.Len() (growth handled by
// reflect.AppendSlice), or when destination and source have incompatible types.
//
// Takes arena (*RegisterArena) which provides the slice-header slab; nil disables the
// fast path.
// Takes destination (reflect.Value) which is the destination slice.
// Takes source (reflect.Value) which is the source slice whose elements are spread into
// destination.
//
// Returns the new slice and true on success, or zero Value and false when the slow path
// must run.
func appendSpreadFastPath(arena *RegisterArena, destination, source reflect.Value) (reflect.Value, bool) {
	if arena == nil {
		return reflect.Value{}, false
	}
	if !destination.IsValid() || destination.Kind() != reflect.Slice {
		return reflect.Value{}, false
	}
	if source.Kind() != reflect.Slice {
		return reflect.Value{}, false
	}
	destinationLength := destination.Len()
	sourceLength := source.Len()
	newLen := destinationLength + sourceLength
	if newLen > destination.Cap() {
		return reflect.Value{}, false
	}
	if destination.Type() != source.Type() {
		return reflect.Value{}, false
	}

	destinationHeaderPtr := reflectValuePtr(destination)
	if destinationHeaderPtr == nil {
		return reflect.Value{}, false
	}
	sourceHeader := (*arenaSliceHeader)(destinationHeaderPtr)
	slot := arena.AllocSliceHeader()
	slot.Data = sourceHeader.Data
	slot.Len = newLen
	slot.Cap = destination.Cap()
	extended := unsafeNewAt(reflectValueABIType(destination.Type()), unsafe.Pointer(slot), reflect.Slice)

	if destinationBytes, destinationOk := reflect.TypeAssert[[]byte](destination); destinationOk {
		if sourceBytes, sourceOk := reflect.TypeAssert[[]byte](source); sourceOk {
			tail := unsafe.Slice((*byte)(unsafe.Add(slot.Data, destinationLength)), sourceLength)
			_ = destinationBytes
			copy(tail, sourceBytes)
			_ = arena
			return extended, true
		}
	}
	reflect.Copy(extended.Slice(destinationLength, newLen), source)
	_ = arena
	return extended, true
}

// handleSubOpAppendUint implements the subOpAppendUint tier-1 sub-op: general[B] :=
// append(general[C], byte(uints[ext.A])). Eliminates the cascading reflect.TypeAssert
// chain in appendFastPath for the statically-known uint-element-slice case, which is
// expr_eval's hot `*output = append(*output, '(')` byte-builder pattern.
//
// Encoding:
//
//	op = opDrillTier1
//	a  = subOpAppendUint
//	b  = destination register (general bank)
//	c  = source slice register (general bank)
//	ext.a = element register (uint bank)
//
// Takes vm (*VM) which provides arena, evalError, and allocation limits.
// Takes frame (*callFrame) whose programCounter is advanced past the extension word.
// Takes registers (*Registers) which holds the general slice and uint element banks.
// Takes instr (instruction) which encodes the destination, slice and element register
// indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpAppendUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	sliceValue := registers.general[instr.c]
	element := registers.uints[extensionWord.a]

	if !sliceValue.IsValid() {
		registers.general[instr.b] = reflect.ValueOf([]byte{safeconv.Uint64ToUint8(element)})
		return opContinue
	}
	if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
		return rc
	}

	if s, ok := reflect.TypeAssert[[]byte](sliceValue); ok {
		grown := arenaAppendByte(vm.arena, s, safeconv.Uint64ToUint8(element))
		registers.general[instr.b] = arenaWrapByteSlice(vm.arena, grown)
		return opContinue
	}
	if s, ok := reflect.TypeAssert[[]uint64](sliceValue); ok {
		grown := arenaAppendUint(vm.arena, s, element)
		registers.general[instr.b] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrUint64(grown), len(grown), cap(grown))
		return opContinue
	}
	if s, ok := reflect.TypeAssert[[]uint](sliceValue); ok {
		s = append(s, uint(element))
		registers.general[instr.b] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrUint(s), len(s), cap(s))
		return opContinue
	}
	if s, ok := reflect.TypeAssert[[]uint32](sliceValue); ok {
		s = append(s, safeconv.Uint64ToUint32(element))
		registers.general[instr.b] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrUint32(s), len(s), cap(s))
		return opContinue
	}
	if s, ok := reflect.TypeAssert[[]uint16](sliceValue); ok {
		s = append(s, safeconv.Uint64ToUint16(element))
		registers.general[instr.b] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrUint16(s), len(s), cap(s))
		return opContinue
	}
	if s, ok := reflect.TypeAssert[[]uintptr](sliceValue); ok {
		s = append(s, uintptr(element))
		registers.general[instr.b] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrUintptr(s), len(s), cap(s))
		return opContinue
	}

	registers.general[instr.b] = reflect.Append(sliceValue, reflect.ValueOf(element).Convert(sliceValue.Type().Elem()))
	return opContinue
}

// handleSubOpAppendInt is the tier-1 adapter for the typed-int append.
//
// Reads the element register from the extension word and delegates to the underlying
// handleAppendInt with the tier-0 synthetic instruction shape (a=dest, b=slice,
// c=intReg).
//
// Takes vm (*VM) which is forwarded to handleAppendInt.
// Takes frame (*callFrame) whose programCounter is advanced past the extension word.
// Takes registers (*Registers) which holds the slice and int banks.
// Takes instr (instruction) which encodes the destination and slice register indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpAppendInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	return handleAppendInt(vm, frame, registers, instruction{a: instr.b, b: instr.c, c: extensionWord.a})
}

// handleSubOpAppendString is the tier-1 adapter for the typed-string append.
//
// Same shape as handleSubOpAppendInt: reads the element register from the extension word
// and delegates to handleAppendString.
//
// Takes vm (*VM) which is forwarded to handleAppendString.
// Takes frame (*callFrame) whose programCounter is advanced past the extension word.
// Takes registers (*Registers) which holds the slice and string banks.
// Takes instr (instruction) which encodes the destination and slice register indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpAppendString(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	return handleAppendString(vm, frame, registers, instruction{a: instr.b, b: instr.c, c: extensionWord.a})
}

// handleSubOpAppendFloat is the tier-1 adapter for the typed-float append.
//
// Same shape as handleSubOpAppendInt: reads the element register from the extension word
// and delegates to handleAppendFloat.
//
// Takes vm (*VM) which is forwarded to handleAppendFloat.
// Takes frame (*callFrame) whose programCounter is advanced past the extension word.
// Takes registers (*Registers) which holds the slice and float banks.
// Takes instr (instruction) which encodes the destination and slice register indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpAppendFloat(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	return handleAppendFloat(vm, frame, registers, instruction{a: instr.b, b: instr.c, c: extensionWord.a})
}

// handleSubOpAppendBool is the tier-1 adapter for the typed-bool append.
//
// Same shape as handleSubOpAppendInt: reads the element register from the extension word
// and delegates to handleAppendBool.
//
// Takes vm (*VM) which is forwarded to handleAppendBool.
// Takes frame (*callFrame) whose programCounter is advanced past the extension word.
// Takes registers (*Registers) which holds the slice and bool banks.
// Takes instr (instruction) which encodes the destination and slice register indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpAppendBool(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	return handleAppendBool(vm, frame, registers, instruction{a: instr.b, b: instr.c, c: extensionWord.a})
}

// handleAppendInt handles the opAppendInt instruction by appending an integer value from
// an int register to a slice in a general register.
//
// Takes registers (*Registers) which holds the slice and integer element.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	sliceValue := registers.general[instruction.b]
	element := registers.ints[instruction.c]

	if instruction.a == instruction.b {
		return appendIntInPlace(vm, registers, instruction.a, sliceValue, element)
	}

	if !sliceValue.IsValid() {
		registers.general[instruction.a] = reflect.ValueOf([]int{int(element)})
		return opContinue
	}
	if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
		return rc
	}

	if s, ok := reflect.TypeAssert[[]int64](sliceValue); ok {
		grown := arenaAppendInt(vm.arena, s, element)
		registers.general[instruction.a] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrInt64(grown), len(grown), cap(grown))
		return opContinue
	}
	if s, ok := reflect.TypeAssert[[]int](sliceValue); ok {
		s = append(s, int(element))
		registers.general[instruction.a] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrInt(s), len(s), cap(s))
		return opContinue
	}

	registers.general[instruction.a] = reflect.Append(sliceValue, reflect.ValueOf(int(element)))
	return opContinue
}

// appendIntInPlace appends an int element to a slice in-place using
// Grow/SetLen/Index.Set, avoiding reflect.ValueOf allocations.
//
// The slice value is promoted to addressable on first use.
//
// Takes registers (*Registers) which holds the register banks.
// Takes destination (uint8) which is the register to store the result slice in.
// Takes sliceValue (reflect.Value) which is the slice to append to.
// Takes element (int64) which is the value to append.
//
// Returns opResult indicating the next execution step.
func appendIntInPlace(vm *VM, registers *Registers, destination uint8, sliceValue reflect.Value, element int64) opResult {
	return appendScalarInPlace(vm, registers, destination, sliceValue, reflect.TypeFor[[]int](), func(target reflect.Value) {
		target.SetInt(element)
	})
}

// appendScalar is a generic helper for typed append handlers. It attempts a
// concrete-slice fast path before falling back to reflect.Append.
//
// Callers that own a *RegisterArena route through the arena-aware fast paths in the typed
// append handlers (handleAppendInt etc.) so the grow path keeps the new backing inside
// the arena.
//
// Takes registers (*Registers) which provides the register file.
// Takes instruction (instruction) which specifies the current instruction.
// Takes element (T) which is the element to append.
//
// Returns opResult indicating continuation.
func appendScalar[T comparable](registers *Registers, instruction instruction, element T) opResult {
	sliceValue := registers.general[instruction.b]
	if !sliceValue.IsValid() {
		registers.general[instruction.a] = reflect.ValueOf([]T{element})
		return opContinue
	}
	if s, ok := reflect.TypeAssert[[]T](sliceValue); ok {
		registers.general[instruction.a] = reflect.ValueOf(append(s, element))
		return opContinue
	}
	registers.general[instruction.a] = reflect.Append(sliceValue, reflect.ValueOf(element))
	return opContinue
}

// handleAppendString handles the opAppendString instruction by appending a string value
// from a string register to a slice in a general register.
//
// Takes registers (*Registers) which holds the slice and string element.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
//
//nolint:dupl // typed-slice fast paths; per-element-type specialisation is intentional
func handleAppendString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	element := registers.strings[instruction.c]
	if instruction.a == instruction.b {
		return appendStringInPlace(vm, registers, instruction.a, registers.general[instruction.b], element)
	}
	sliceValue := registers.general[instruction.b]
	if sliceValue.IsValid() {
		if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
			return rc
		}
		if s, ok := reflect.TypeAssert[[]string](sliceValue); ok {
			grown := arenaAppendString(vm.arena, s, element)
			registers.general[instruction.a] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrString(grown), len(grown), cap(grown))
			return opContinue
		}
	}
	return appendScalarChecked(vm, registers, instruction, element)
}

// appendStringInPlace appends a string element to an addressable slice using
// Grow/SetLen/SetString, avoiding reflect.ValueOf boxing.
//
// Takes registers (*Registers) which provides the register file.
// Takes destination (uint8) which is the destination general register.
// Takes sliceValue (reflect.Value) which is the current slice.
// Takes element (string) which is the element to append.
//
// Returns opResult indicating continuation.
func appendStringInPlace(vm *VM, registers *Registers, destination uint8, sliceValue reflect.Value, element string) opResult {
	return appendScalarInPlace(vm, registers, destination, sliceValue, reflect.TypeFor[[]string](), func(target reflect.Value) {
		target.SetString(element)
	})
}

// handleAppendFloat handles the opAppendFloat instruction by appending a float value from
// a float register to a slice in a general register.
//
// Takes registers (*Registers) which holds the slice and float element.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
//
//nolint:dupl // typed-slice fast paths; per-element-type specialisation is intentional
func handleAppendFloat(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	element := registers.floats[instruction.c]
	if instruction.a == instruction.b {
		return appendFloatInPlace(vm, registers, instruction.a, registers.general[instruction.b], element)
	}
	sliceValue := registers.general[instruction.b]
	if sliceValue.IsValid() {
		if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
			return rc
		}
		if s, ok := reflect.TypeAssert[[]float64](sliceValue); ok {
			grown := arenaAppendFloat(vm.arena, s, element)
			registers.general[instruction.a] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrFloat64(grown), len(grown), cap(grown))
			return opContinue
		}
	}
	return appendScalarChecked(vm, registers, instruction, element)
}

// appendFloatInPlace appends a float element to an addressable slice using
// Grow/SetLen/SetFloat, avoiding reflect.ValueOf boxing.
//
// Takes registers (*Registers) which provides the register file.
// Takes destination (uint8) which is the destination general register.
// Takes sliceValue (reflect.Value) which is the current slice.
// Takes element (float64) which is the element to append.
//
// Returns opResult indicating continuation.
func appendFloatInPlace(vm *VM, registers *Registers, destination uint8, sliceValue reflect.Value, element float64) opResult {
	return appendScalarInPlace(vm, registers, destination, sliceValue, reflect.TypeFor[[]float64](), func(target reflect.Value) {
		target.SetFloat(element)
	})
}

// handleAppendBool handles the opAppendBool instruction by appending a bool value from a
// bool register to a slice in a general register.
//
// Takes registers (*Registers) which holds the slice and bool element.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
//
//nolint:dupl // typed-slice fast paths; per-element-type specialisation is intentional
func handleAppendBool(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	element := registers.bools[instruction.c]
	if instruction.a == instruction.b {
		return appendBoolInPlace(vm, registers, instruction.a, registers.general[instruction.b], element)
	}
	sliceValue := registers.general[instruction.b]
	if sliceValue.IsValid() {
		if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
			return rc
		}
		if s, ok := reflect.TypeAssert[[]bool](sliceValue); ok {
			grown := arenaAppendBool(vm.arena, s, element)
			registers.general[instruction.a] = arenaWrapTypedSliceFromSource(vm.arena, sliceValue, sliceDataPtrBool(grown), len(grown), cap(grown))
			return opContinue
		}
	}
	return appendScalarChecked(vm, registers, instruction, element)
}

// appendBoolInPlace appends a bool element to an addressable slice using
// Grow/SetLen/SetBool, avoiding reflect.ValueOf boxing.
//
// Takes registers (*Registers) which provides the register file.
// Takes destination (uint8) which is the destination general register.
// Takes sliceValue (reflect.Value) which is the current slice.
// Takes element (bool) which is the element to append.
//
// Returns opResult indicating continuation.
func appendBoolInPlace(vm *VM, registers *Registers, destination uint8, sliceValue reflect.Value, element bool) opResult {
	return appendScalarInPlace(vm, registers, destination, sliceValue, reflect.TypeFor[[]bool](), func(target reflect.Value) {
		target.SetBool(element)
	})
}

// checkAppendLimit returns opPanicError if appending one element to sliceValue would
// exceed maxAllocSize.
//
// Takes vm (*VM) which provides access to allocation limits.
// Takes sliceValue (reflect.Value) which is the slice being appended to.
//
// Returns opResult which is opPanicError when the limit is exceeded, or opContinue
// otherwise.
func checkAppendLimit(vm *VM, sliceValue reflect.Value) opResult {
	if vm.limits.maxAllocSize > 0 && sliceValue.IsValid() && sliceValue.Len()+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, sliceValue.Len()+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	return opContinue
}

// appendScalarChecked is appendScalar with an allocation limit check.
//
// Takes vm (*VM) which provides access to allocation limits.
// Takes registers (*Registers) which holds the slice and destination.
// Takes instruction (instruction) which encodes the register indices.
// Takes element (T) which is the value to append.
//
// Returns opResult which is opPanicError when the limit is exceeded, or the result of
// appendScalar otherwise.
func appendScalarChecked[T comparable](vm *VM, registers *Registers, instruction instruction, element T) opResult {
	sliceValue := registers.general[instruction.b]
	if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
		return rc
	}
	return appendScalar(registers, instruction, element)
}

// appendScalarInPlace is the shared implementation for all typed in-place append
// handlers. It uses Grow/SetLen/Index to extend the slice without allocating a new
// reflect.Value via reflect.ValueOf.
//
// Takes vm (*VM) which provides access to allocation limits.
// Takes registers (*Registers) which provides the register file.
// Takes destination (uint8) which is the destination general register.
// Takes sliceValue (reflect.Value) which is the current slice.
// Takes zeroSliceType (reflect.Type) which is the slice type to create when the current
// value is invalid (nil slice).
// Takes setter (func(reflect.Value)) which writes the element into the target
// reflect.Value at the appended index.
//
// Returns opResult indicating continuation.
func appendScalarInPlace(
	vm *VM,
	registers *Registers,
	destination uint8,
	sliceValue reflect.Value,
	zeroSliceType reflect.Type,
	setter func(reflect.Value),
) opResult {
	if !sliceValue.IsValid() {
		slicePointer := reflect.New(zeroSliceType)
		addressable := slicePointer.Elem()
		addressable.Grow(1)
		addressable.SetLen(1)
		setter(addressable.Index(0))
		registers.general[destination] = addressable
		return opContinue
	}
	if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
		return rc
	}
	if !sliceValue.CanSet() {
		slicePointer := reflect.New(sliceValue.Type())
		slicePointer.Elem().Set(sliceValue)
		sliceValue = slicePointer.Elem()
	}
	length := sliceValue.Len()
	sliceValue.Grow(1)
	sliceValue.SetLen(length + 1)
	setter(sliceValue.Index(length))
	registers.general[destination] = sliceValue
	return opContinue
}

// handleAppendByteFast is the tier-0 specialised byte-builder hot path.
//
// Emitted by compileBuiltinAppend when the slice element type is statically `byte` and
// the call is single-element (no spread). Skips the reflect.TypeAssert cascade and
// checkAppendLimit walk that handleSubOpAppendUint and handleAppend pay: the compiler has
// already proven the slice is []byte and the element is a uint8 value held in a uint
// register.
//
// Encoding:
//
//	A = destination general register (resulting []byte reflect.Value)
//	B = source slice general register (existing []byte reflect.Value)
//	C = byte value uint register (the single byte to append)
//
// expr_eval's generateExpression / generateTerm functions emit `*output = append(*output,
// '(')`-style patterns thousands of times per iteration; this opcode is their dominant
// cost.
//
// Takes vm (*VM) which provides arena, evalError, and allocation limits.
// Takes registers (*Registers) which holds the general slice and uint element banks.
// Takes instr (instruction) which encodes A, B, C register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendByteFast(vm *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	sliceValue := registers.general[instr.b]
	if !sliceValue.IsValid() {
		registers.general[instr.a] = reflect.ValueOf([]byte{safeconv.Uint64ToUint8(registers.uints[instr.c])})
		return opContinue
	}

	if vm.limits.maxAllocSize > 0 && sliceValue.Len()+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, sliceValue.Len()+1, vm.limits.maxAllocSize)
		return opPanicError
	}

	s, ok := reflect.TypeAssert[[]byte](sliceValue)
	if !ok {
		element := registers.uints[instr.c]
		boxedElement := reflect.ValueOf(safeconv.Uint64ToUint8(element))
		if result, ok := appendFastPath(vm.arena, sliceValue, boxedElement); ok {
			registers.general[instr.a] = result
			return opContinue
		}
		registers.general[instr.a] = reflect.Append(sliceValue, boxedElement)
		return opContinue
	}
	//nolint:gocritic // fresh capture so the result can wrap via arenaWrapByteSpreadResult
	result := append(s, safeconv.Uint64ToUint8(registers.uints[instr.c]))
	registers.general[instr.a] = arenaWrapByteSpreadResult(vm.arena, sliceValue, result)
	return opContinue
}

// handleAppendByteFastInPlace is the unified in-place byte-builder append.
//
// Handles both `x = append(x, b)` (when registers.general[instr.b] holds a []byte slice
// reflect.Value backed by an arena slice-header slot, mutating Data/Len/Cap directly) and
// `*p = append(*p, b)` (when the register holds a *[]byte pointer reflect.Value,
// dereferencing through srcShape's flagIndir and writing the new header via
// runtimeTypedmemmove so heap-resident headers honour the GC write barrier on the Data
// pointer field).
//
// At runtime instr.a == instr.b by construction; in-place semantics work because the
// underlying slice header is mutated, not the register cell. For the x form the
// destination register's reflect.Value still points at the (now-mutated) slot; for the *p
// form the destination register is the unchanged pointer and the side effect is
// observable through subsequent *p reads. The x-form defensively falls back to
// handleAppendByteFast (which allocates a fresh slot) when the source slice header is not
// arena-owned. The *p form uses runtimeTypedmemmove which is type-safe regardless of
// where the pointed-to header lives.
//
// Takes vm (*VM) which provides arena, evalError, and allocation limits.
// Takes frame (*callFrame) which the x-form fallback path forwards.
// Takes registers (*Registers) which holds the slice/pointer and uint banks.
// Takes instr (instruction) which encodes the A, B, C register indices (A == B; C is the
// byte value).
//
// Returns opResult indicating the next execution step.
func handleAppendByteFastInPlace(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	sliceValue := registers.general[instr.b]
	if !sliceValue.IsValid() {
		registers.general[instr.a] = reflect.ValueOf([]byte{safeconv.Uint64ToUint8(registers.uints[instr.c])})
		return opContinue
	}

	switch sliceValue.Kind() {
	case reflect.Slice:
		return appendByteFastInPlaceSlice(vm, frame, registers, instr, sliceValue)
	case reflect.Pointer:
		if sliceValue.IsNil() {
			return handleAppendByteFast(vm, frame, registers, instr)
		}
		return appendByteFastInPlacePointer(vm, registers, instr, sliceValue)
	default:
		return handleAppendByteFast(vm, frame, registers, instr)
	}
}

// appendByteFastInPlaceSlice handles the x = append(x, b) shape.
//
// Inspects the source reflect.Value's internal ptr field; if it points at an arena-owned
// arenaSliceHeader slot, mutates Data/Len/Cap on that slot in place. Otherwise routes to
// handleAppendByteFast which allocates a fresh slot, the defensive path for cases the
// safety predicate didn't anticipate.
//
// Takes vm (*VM) which provides arena and allocation limits.
// Takes frame (*callFrame) which is forwarded to the fallback path.
// Takes registers (*Registers) which holds the slice and uint banks.
// Takes instr (instruction) which encodes the register indices.
// Takes sliceValue (reflect.Value) which is the source slice.
//
// Returns opResult indicating the next execution step.
func appendByteFastInPlaceSlice(vm *VM, frame *callFrame, registers *Registers, instr instruction, sliceValue reflect.Value) opResult {
	if vm.limits.maxAllocSize > 0 && sliceValue.Len()+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, sliceValue.Len()+1, vm.limits.maxAllocSize)
		return opPanicError
	}

	sourceShape := (*unsafeReflectValue)(unsafe.Pointer(&sliceValue))
	headerPtr := (*arenaSliceHeader)(sourceShape.ptr)
	if headerPtr == nil || !vm.arena.ownsSliceHeaderPointer(unsafe.Pointer(headerPtr)) {
		return handleAppendByteFast(vm, frame, registers, instr)
	}

	current := unsafe.Slice((*byte)(headerPtr.Data), headerPtr.Cap)[:headerPtr.Len]
	result := arenaAppendByte(vm.arena, current, safeconv.Uint64ToUint8(registers.uints[instr.c]))

	headerPtr.Data = unsafe.Pointer(unsafe.SliceData(result))
	headerPtr.Len = len(result)
	headerPtr.Cap = cap(result)
	return opContinue
}

// handleAppendInPlace is the generic in-place append.
//
// Handles arbitrary element types via the same kind switch as
// handleAppendByteFastInPlace; the only difference is the element register lives on the
// general bank (passed by reflect.Value) so the per-type fast paths route through
// appendGenericFastPath / appendFastPath helpers. At runtime A == B by construction.
// Falls back to handleAppend on non-arena slices or non-slice/pointer source kinds.
//
// Encoding (mirrors opAppend): A=dest general, B=src general, C=element general, followed
// by opExt.
//
// Takes vm (*VM) which provides arena and allocation limits.
// Takes frame (*callFrame) which is forwarded to fallback paths.
// Takes registers (*Registers) which holds the general bank.
// Takes instr (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendInPlace(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	sliceValue := registers.general[instr.b]
	if !sliceValue.IsValid() {
		return handleAppend(vm, frame, registers, instr)
	}
	switch sliceValue.Kind() {
	case reflect.Slice:
		return appendInPlaceSlice(vm, frame, registers, instr, sliceValue)
	case reflect.Pointer:
		if sliceValue.IsNil() {
			return handleAppend(vm, frame, registers, instr)
		}
		return appendInPlacePointer(vm, frame, registers, instr, sliceValue)
	default:
		return handleAppend(vm, frame, registers, instr)
	}
}

// appendInPlaceSlice mutates the slice header on the arena-owned slot.
//
// Routes through appendGenericFastPath when the element fits in spare capacity (no grow
// needed). On grow or non-arena slot, falls back to handleAppend.
//
// Takes vm (*VM) which provides arena and allocation limits.
// Takes frame (*callFrame) which is forwarded to fallback paths.
// Takes registers (*Registers) which holds the general bank.
// Takes instr (instruction) which encodes the register indices.
// Takes sliceValue (reflect.Value) which is the source slice.
//
// Returns opResult indicating the next execution step.
func appendInPlaceSlice(vm *VM, frame *callFrame, registers *Registers, instr instruction, sliceValue reflect.Value) opResult {
	sourceShape := (*unsafeReflectValue)(unsafe.Pointer(&sliceValue))
	headerPtr := (*arenaSliceHeader)(sourceShape.ptr)
	if headerPtr == nil || !vm.arena.ownsSliceHeaderPointer(unsafe.Pointer(headerPtr)) {
		return handleAppend(vm, frame, registers, instr)
	}
	if vm.limits.maxAllocSize > 0 && sliceValue.Len()+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, sliceValue.Len()+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	element := registers.general[instr.c]
	if !element.IsValid() {
		return handleAppend(vm, frame, registers, instr)
	}

	if result, ok := appendFastPath(vm.arena, sliceValue, element); ok {
		resultShape := (*unsafeReflectValue)(unsafe.Pointer(&result))
		resultHeader := (*arenaSliceHeader)(resultShape.ptr)
		if resultHeader != nil && vm.arena.ownsSliceHeaderPointer(unsafe.Pointer(resultHeader)) {
			headerPtr.Data = resultHeader.Data
			headerPtr.Len = resultHeader.Len
			headerPtr.Cap = resultHeader.Cap
			return opContinue
		}
		registers.general[instr.a] = result
		return opContinue
	}
	if result, ok := appendGenericFastPath(vm.arena, sliceValue, element); ok {
		resultShape := (*unsafeReflectValue)(unsafe.Pointer(&result))
		resultHeader := (*arenaSliceHeader)(resultShape.ptr)
		if resultHeader != nil && vm.arena.ownsSliceHeaderPointer(unsafe.Pointer(resultHeader)) {
			headerPtr.Data = resultHeader.Data
			headerPtr.Len = resultHeader.Len
			headerPtr.Cap = resultHeader.Cap
			return opContinue
		}
		registers.general[instr.a] = result
		return opContinue
	}
	return handleAppend(vm, frame, registers, instr)
}

// appendInPlacePointer handles *p = append(*p, e) for arbitrary types.
//
// Uses the same runtimeTypedmemmove machinery as the byte variant. Falls back to
// handleAppend for slot-extraction failures.
//
// Takes vm (*VM) which provides arena and allocation limits.
// Takes frame (*callFrame) which is forwarded to fallback paths.
// Takes registers (*Registers) which holds the general bank.
// Takes instr (instruction) which encodes the register indices.
// Takes pointerValue (reflect.Value) which is the pointer to the destination slice.
//
// Returns opResult indicating the next execution step.
func appendInPlacePointer(vm *VM, frame *callFrame, registers *Registers, instr instruction, pointerValue reflect.Value) opResult {
	recvShape := (*unsafeReflectValue)(unsafe.Pointer(&pointerValue))
	var headerPtr unsafe.Pointer
	if recvShape.flag&flagIndir != 0 {
		headerPtr = *(*unsafe.Pointer)(recvShape.ptr)
	} else {
		headerPtr = recvShape.ptr
	}
	if headerPtr == nil {
		return handleAppend(vm, frame, registers, instr)
	}

	elemType := pointerValue.Type().Elem()
	current := reflect.NewAt(elemType, headerPtr).Elem()
	if vm.limits.maxAllocSize > 0 && current.Len()+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, current.Len()+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	element := registers.general[instr.c]
	if !element.IsValid() {
		return handleAppend(vm, frame, registers, instr)
	}
	coerced := coerceValue(vm, element, elemType.Elem())
	extended := reflect.Append(current, coerced)
	runtimeTypedmemmove(reflectValueABIType(elemType), headerPtr, unsafe.Pointer(extended.UnsafePointer()))
	current.Set(extended)
	return opContinue
}

// handleAppendSpreadInPlace is the spread sibling of handleAppendInPlace.
//
// Handles x = append(x, src...) by routing to handleAppendSpread for the generic
// catch-all path (which is correct: the tier-0 opcode has the same operand-encoding shape
// as opAppendSpread, including the trailing extension word). The tier-0 fallback to
// allocate a fresh slot is acceptable here because in-place spread for arbitrary element
// types is rare in real workloads (byte spread is the hot case and routes through
// subOpAppendByteSpreadInPlace).
//
// Encoding: A=dest general, B=src general, C=source slice general, followed by opExt. At
// runtime A == B.
//
// Takes vm (*VM) which provides arena and allocation limits.
// Takes frame (*callFrame) which is forwarded to handleAppendSpread.
// Takes registers (*Registers) which holds the general bank.
// Takes instr (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendSpreadInPlace(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleAppendSpread(vm, frame, registers, instr)
}
