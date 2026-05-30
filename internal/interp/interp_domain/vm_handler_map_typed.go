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
	"reflect"
	"unsafe"
)

const (
	// mapKeyFast64SizeBytes is the byte size of an 8-byte key type recognised by
	// runtimeMapaccess2_fast64. Keys that are not exactly this wide cannot use the fast
	// specialisation.
	mapKeyFast64SizeBytes = 8
)

// mapKindError reports whether the supplied value is a map, raising an interpreted
// runtime panic on the VM when it is not. Typed map handlers call this after the IsValid
// check so a non-map register surfaces as an interpreted panic instead of a host crash
// inside reflect.
//
// Takes vm (*VM) which receives the interpreted panic error.
// Takes m (reflect.Value) which is the value expected to be a map.
//
// Returns true when m is a map kind, false otherwise.
func mapKindError(vm *VM, m reflect.Value) bool {
	if m.Kind() == reflect.Map {
		return true
	}
	return mapKindErrorSlow(vm, m)
}

// mapKindErrorSlow records the non-map diagnostic. It is split out and marked noinline so
// the newRuntimePanicError formatting stays off mapKindError's hot path, letting that
// guard collapse to a single Kind comparison that inlines into every typed-map handler.
//
// Takes vm (*VM) which receives the evaluation error.
// Takes m (reflect.Value) which is the non-map value.
//
// Returns false, so callers can return mapKindErrorSlow(...) directly.
//
//go:noinline
func mapKindErrorSlow(vm *VM, m reflect.Value) bool {
	vm.evalError = newRuntimePanicError("interp: map operation on non-map value (%s)", m.Kind())
	return false
}

// mapAccessCacheEntry memoises the per-map-type info that typed map handlers
// (opMapIndexOkIntGeneral and friends) would otherwise recompute on every invocation.
//
// Cached on the VM via mapAccessCacheLastType and refreshed lazily when the type pointer
// changes. The single-slot design exploits the observation that hot call sites usually
// index the same concrete map type repeatedly.
type mapAccessCacheEntry struct {
	// keyType is the cached m.Type().Key() result.
	keyType reflect.Type

	// elemType is the cached m.Type().Elem() result.
	elemType reflect.Type

	// fast64Eligible records whether keyType passes mapKeyIsFast64Eligible so the handler
	// can branch without recomputing the size+kind check.
	fast64Eligible bool
}

// resolveMapAccessCache returns the cached (keyType, elemType, fast64Eligible) tuple for
// the given map's reflect.Type. The cache is a per-VM single-slot LRU: a fresh map type
// misses, recomputes, and overwrites the slot; the next call against the same type hits
// with two loads and a pointer-equality compare.
//
// Callers MUST pass mapType = m.Type() exactly (not the underlying element type).
// Returning the entry by value avoids any aliasing concern with the cached slot.
//
// Takes vm (*VM) which owns the single-slot cache.
// Takes mapType (reflect.Type) which is the map's static type.
//
// Returns the cached entry, recomputed on miss.
func resolveMapAccessCache(vm *VM, mapType reflect.Type) mapAccessCacheEntry {
	if vm.mapAccessCacheLastType == mapType {
		return vm.mapAccessCacheLastEntry
	}
	keyType := mapType.Key()
	elemType := mapType.Elem()
	entry := mapAccessCacheEntry{
		keyType:        keyType,
		elemType:       elemType,
		fast64Eligible: mapKeyIsFast64Eligible(keyType),
	}
	vm.mapAccessCacheLastType = mapType
	vm.mapAccessCacheLastEntry = entry
	return entry
}

// handleMapIndexOkIntInt handles opMapIndexOkIntInt: reads ints[A] = map[int]int (or
// map[int]int64) in general[B] with key ints[C], plus the extension word's A field as the
// ok register.
//
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapIndexOkIntInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkIntInt", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.ints[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[int]int](m); ok {
		value, present := concrete[int(key)]
		registers.ints[instruction.a] = int64(value)
		registers.ints[extensionWord.a] = boolToInt64(present)
		return opContinue
	}
	if concrete, ok := reflect.TypeAssert[map[int64]int64](m); ok {
		value, present := concrete[key]
		registers.ints[instruction.a] = value
		registers.ints[extensionWord.a] = boolToInt64(present)
		return opContinue
	}
	keyReflectValue := intMapKeyScratch(vm, m.Type().Key())
	keyReflectValue.SetInt(key)
	result := m.MapIndex(keyReflectValue)
	if result.IsValid() {
		registers.ints[instruction.a] = result.Int()
		registers.ints[extensionWord.a] = 1
	} else {
		registers.ints[instruction.a] = 0
		registers.ints[extensionWord.a] = 0
	}
	return opContinue
}

// handleMapIndexOkStringInt handles opMapIndexOkStringInt: reads ints[A] = map[string]int
// (or map[string]int64) in general[B] with key strings[C], plus the extension word's A
// field as the ok register.
//
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapIndexOkStringInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkStringInt", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.strings[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[string]int](m); ok {
		value, present := concrete[key]
		registers.ints[instruction.a] = int64(value)
		registers.ints[extensionWord.a] = boolToInt64(present)
		return opContinue
	}
	if concrete, ok := reflect.TypeAssert[map[string]int64](m); ok {
		value, present := concrete[key]
		registers.ints[instruction.a] = value
		registers.ints[extensionWord.a] = boolToInt64(present)
		return opContinue
	}
	keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
	result := m.MapIndex(keyReflectValue)
	if result.IsValid() {
		registers.ints[instruction.a] = result.Int()
		registers.ints[extensionWord.a] = 1
	} else {
		registers.ints[instruction.a] = 0
		registers.ints[extensionWord.a] = 0
	}
	return opContinue
}

// handleMapIndexOkStringString handles opMapIndexOkStringString: reads strings[A] =
// map[string]string in general[B] with key strings[C], plus the extension word's A field
// as the ok register.
//
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapIndexOkStringString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkStringString", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.strings[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[string]string](m); ok {
		value, present := concrete[key]
		registers.strings[instruction.a] = value
		registers.ints[extensionWord.a] = boolToInt64(present)
		return opContinue
	}
	keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
	result := m.MapIndex(keyReflectValue)
	if result.IsValid() {
		registers.strings[instruction.a] = result.String()
		registers.ints[extensionWord.a] = 1
	} else {
		registers.strings[instruction.a] = ""
		registers.ints[extensionWord.a] = 0
	}
	return opContinue
}

// handleMapIndexOkIntString handles opMapIndexOkIntString: reads strings[A] =
// map[int]string (or map[int64]string) in general[B] with key ints[C], plus the extension
// word's A field as the ok register.
//
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapIndexOkIntString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkIntString", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.ints[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[int]string](m); ok {
		value, present := concrete[int(key)]
		registers.strings[instruction.a] = value
		registers.ints[extensionWord.a] = boolToInt64(present)
		return opContinue
	}
	if concrete, ok := reflect.TypeAssert[map[int64]string](m); ok {
		value, present := concrete[key]
		registers.strings[instruction.a] = value
		registers.ints[extensionWord.a] = boolToInt64(present)
		return opContinue
	}
	keyReflectValue := intMapKeyScratch(vm, m.Type().Key())
	keyReflectValue.SetInt(key)
	result := m.MapIndex(keyReflectValue)
	if result.IsValid() {
		registers.strings[instruction.a] = result.String()
		registers.ints[extensionWord.a] = 1
	} else {
		registers.strings[instruction.a] = ""
		registers.ints[extensionWord.a] = 0
	}
	return opContinue
}

// intMapKeyScratch returns a reusable, addressable reflect.Value for the given
// integer-kind map key type.
//
// The first call per key type pays a reflect.New allocation; subsequent calls return the
// cached scratch. Callers must SetInt before passing to MapIndex - the scratch is shared
// so previous contents are arbitrary.
//
// Takes vm (*VM) which owns the per-VM scratch cache.
// Takes keyType (reflect.Type) which is the map's declared key type.
//
// Returns the reusable scratch reflect.Value.
func intMapKeyScratch(vm *VM, keyType reflect.Type) reflect.Value {
	if cached, ok := vm.mapKeyScratch[keyType]; ok {
		return cached
	}
	scratch := reflect.New(keyType).Elem()
	if vm.mapKeyScratch == nil {
		vm.mapKeyScratch = make(map[reflect.Type]reflect.Value, initialMapKeyScratchCapacity)
	}
	vm.mapKeyScratch[keyType] = scratch
	return scratch
}

// handleMapGetIntGeneral handles opMapGetIntGeneral.
//
// Reads general[A] = map[int]V (or map[int64]V) in general[B] with key ints[C], where V
// is any value type whose register kind is registerGeneral (pointers, interfaces, slices,
// maps of nested types, user structs). Avoids boxing the int key into a fresh
// reflect.Value per call by reusing a per-keyType scratch (vm.mapKeyScratch). The result
// allocation for the value is unavoidable without unsafe map access.
//
// Takes vm (*VM) which owns the per-VM key scratch cache.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapGetIntGeneral(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapGetIntGeneral", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.ints[instruction.c]
	keyScratch := intMapKeyScratch(vm, m.Type().Key())
	keyScratch.SetInt(key)
	result := m.MapIndex(keyScratch)
	if result.IsValid() {
		registers.general[instruction.a] = result
	} else {
		registers.general[instruction.a] = zeroValueForType(m.Type().Elem())
	}
	return opContinue
}

// handleMapIndexOkIntGeneral handles opMapIndexOkIntGeneral.
//
// Reads general[A] = map[int]V (or map[int64]V) in general[B] with key ints[C], plus the
// extension word's A field as the int register holding the ok flag. Mirrors
// handleMapGetIntGeneral but sets instruction.a to the zero value of the map's element
// type when the key is absent (matching Go's `v, ok := m[k]` semantics).
//
// Takes vm (*VM) which owns the per-VM key scratch cache.
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapIndexOkIntGeneral(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkIntGeneral", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.ints[instruction.c]
	cacheEntry := resolveMapAccessCache(vm, m.Type())
	keyType := cacheEntry.keyType
	elemType := cacheEntry.elemType
	if useMapFastLinkname() && cacheEntry.fast64Eligible {
		resultPtr, present := mapAccessFast64ToGeneral(m, key)
		if !present {
			registers.general[instruction.a] = zeroValueForType(elemType)
			registers.ints[extensionWord.a] = 0
			return opContinue
		}
		registers.general[instruction.a] = wrapMapElemFast(elemType, resultPtr)
		registers.ints[extensionWord.a] = 1
		return opContinue
	}
	keyScratch := intMapKeyScratch(vm, keyType)
	keyScratch.SetInt(key)
	result := m.MapIndex(keyScratch)
	if result.IsValid() {
		registers.general[instruction.a] = result
		registers.ints[extensionWord.a] = 1
	} else {
		registers.general[instruction.a] = zeroValueForType(elemType)
		registers.ints[extensionWord.a] = 0
	}
	return opContinue
}

// mapKeyIsFast64Eligible reports whether keyType is eligible for the 8-byte
// mapaccess2_fast64 specialisation.
//
// Restricted to plain int/int64/uint/uint64/uintptr - the strictest subset that always
// passes runtimeMapaccess2Fast64's internal type checks regardless of Go runtime version.
//
// Takes keyType (reflect.Type) which is the candidate map key type.
//
// Returns true when keyType qualifies, false otherwise.
func mapKeyIsFast64Eligible(keyType reflect.Type) bool {
	if keyType.Size() != mapKeyFast64SizeBytes {
		return false
	}
	switch keyType.Kind() {
	case reflect.Int, reflect.Int64,
		reflect.Uint, reflect.Uint64,
		reflect.Uintptr:
		return true
	default:
	}
	return false
}

// wrapMapElemFast wraps the slot pointer from a fast-path map access into a reflect.Value
// of type elemType.
//
// For pointer-kind elements the slot bytes ARE the pointer value, so a non-addressable
// Pointer Value built via unsafePointerKindValue avoids the reflect.Value.Elem allocation
// that the generic reflect path would produce. For other kinds we fall back to
// unsafeNewAt which returns an addressable Value pointing at the map's internal slot -
// safe as long as the map isn't mutated before the caller copies the Value out.
//
// Takes elemType (reflect.Type) which is the map's element type.
// Takes slotPtr (unsafe.Pointer) which is the pointer into the map's internal slot
// returned by mapaccess2_fast64 / faststr.
//
// Returns a reflect.Value backed by the slot (alloc-free for pointer kinds; addressable
// Value otherwise).
func wrapMapElemFast(elemType reflect.Type, slotPtr unsafe.Pointer) reflect.Value {
	abiType := reflectValueABIType(elemType)
	switch elemType.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Chan, reflect.Func, reflect.Map:
		return unsafePointerKindValue(abiType, *(*unsafe.Pointer)(slotPtr))
	default:
	}
	return unsafeNewAt(abiType, slotPtr, elemType.Kind())
}

// handleMapGetIntInt handles the opMapGetIntInt instruction by reading an integer value
// from a map with an integer key.
//
// Takes registers (*Registers) which holds the map and key values.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapGetIntInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapGetIntInt", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.ints[instruction.c]

	if concreteMap, ok := reflect.TypeAssert[map[int]int](m); ok {
		registers.ints[instruction.a] = int64(concreteMap[int(key)])
		return opContinue
	}

	keyReflectValue := intMapKeyScratch(vm, m.Type().Key())
	keyReflectValue.SetInt(key)
	result := m.MapIndex(keyReflectValue)
	if result.IsValid() {
		registers.ints[instruction.a] = result.Int()
	} else {
		registers.ints[instruction.a] = 0
	}
	return opContinue
}

// handleMapSetIntInt handles the opMapSetIntInt instruction by writing an integer value
// to a map with an integer key.
//
// Takes registers (*Registers) which holds the map, key, and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapSetIntInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapSetIntInt", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.ints[instruction.b]
	value := registers.ints[instruction.c]

	if recovered, panicked := guardChannelOp(func() {
		if concreteMap, ok := reflect.TypeAssert[map[int]int](m); ok {
			concreteMap[int(key)] = int(value)
			return
		}
		keyReflectValue := intMapKeyScratch(vm, m.Type().Key())
		keyReflectValue.SetInt(key)
		valueReflectValue := reflect.New(m.Type().Elem()).Elem()
		valueReflectValue.SetInt(value)
		m.SetMapIndex(keyReflectValue, valueReflectValue)
	}); panicked {
		return raiseNativePanicAsInterpreted(vm, recovered)
	}
	return opContinue
}

// handleMapGetStringInt handles opMapGetStringInt: reads ints[A] = map[string]int (or
// map[string]int64) in general[B] with string key strings[C]. Type-asserts the typed map
// handle to dispatch directly, bypassing reflect.MapIndex on the hot path.
//
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapGetStringInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapGetStringInt", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.strings[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[string]int](m); ok {
		registers.ints[instruction.a] = int64(concrete[key])
		return opContinue
	}
	if concrete, ok := reflect.TypeAssert[map[string]int64](m); ok {
		registers.ints[instruction.a] = concrete[key]
		return opContinue
	}
	keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
	result := m.MapIndex(keyReflectValue)
	if result.IsValid() {
		registers.ints[instruction.a] = result.Int()
	} else {
		registers.ints[instruction.a] = 0
	}
	return opContinue
}

// handleMapSetStringInt handles opMapSetStringInt: writes general[A][strings[B]] =
// ints[C] for map[string]int (or map[string]int64).
//
// Takes registers (*Registers) which holds the map, key, and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapSetStringInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapSetStringInt", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := materialiseString(vm.arena, registers.strings[instruction.b])
	value := registers.ints[instruction.c]
	if recovered, panicked := guardChannelOp(func() {
		if concrete, ok := reflect.TypeAssert[map[string]int](m); ok {
			concrete[key] = int(value)
			return
		}
		if concrete, ok := reflect.TypeAssert[map[string]int64](m); ok {
			concrete[key] = value
			return
		}
		keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
		valueReflectValue := reflect.New(m.Type().Elem()).Elem()
		valueReflectValue.SetInt(value)
		m.SetMapIndex(keyReflectValue, valueReflectValue)
	}); panicked {
		return raiseNativePanicAsInterpreted(vm, recovered)
	}
	return opContinue
}

// handleMapGetStringString handles opMapGetStringString: reads strings[A] =
// map[string]string in general[B] with key strings[C].
//
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapGetStringString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapGetStringString", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.strings[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[string]string](m); ok {
		registers.strings[instruction.a] = concrete[key]
		return opContinue
	}
	keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
	result := m.MapIndex(keyReflectValue)
	if result.IsValid() {
		registers.strings[instruction.a] = result.String()
	} else {
		registers.strings[instruction.a] = ""
	}
	return opContinue
}

// handleMapSetStringString handles opMapSetStringString: writes general[A][strings[B]] =
// strings[C] for map[string]string.
//
// Takes registers (*Registers) which holds the map, key, and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapSetStringString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapSetStringString", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := materialiseString(vm.arena, registers.strings[instruction.b])
	value := materialiseString(vm.arena, registers.strings[instruction.c])
	if recovered, panicked := guardChannelOp(func() {
		if concrete, ok := reflect.TypeAssert[map[string]string](m); ok {
			concrete[key] = value
			return
		}
		keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
		valueReflectValue := reflect.ValueOf(value).Convert(m.Type().Elem())
		m.SetMapIndex(keyReflectValue, valueReflectValue)
	}); panicked {
		return raiseNativePanicAsInterpreted(vm, recovered)
	}
	return opContinue
}

// handleMapGetIntString handles opMapGetIntString: reads strings[A] = map[int]string (or
// map[int64]string) in general[B] with key ints[C].
//
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapGetIntString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapGetIntString", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.ints[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[int]string](m); ok {
		registers.strings[instruction.a] = concrete[int(key)]
		return opContinue
	}
	if concrete, ok := reflect.TypeAssert[map[int64]string](m); ok {
		registers.strings[instruction.a] = concrete[key]
		return opContinue
	}
	keyReflectValue := intMapKeyScratch(vm, m.Type().Key())
	keyReflectValue.SetInt(key)
	result := m.MapIndex(keyReflectValue)
	if result.IsValid() {
		registers.strings[instruction.a] = result.String()
	} else {
		registers.strings[instruction.a] = ""
	}
	return opContinue
}

// handleMapSetIntString handles opMapSetIntString: writes general[A][ints[B]] =
// strings[C] for map[int]string (or map[int64]string).
//
// Takes registers (*Registers) which holds the map, key, and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapSetIntString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapSetIntString", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.ints[instruction.b]
	value := materialiseString(vm.arena, registers.strings[instruction.c])
	if recovered, panicked := guardChannelOp(func() {
		if concrete, ok := reflect.TypeAssert[map[int]string](m); ok {
			concrete[int(key)] = value
			return
		}
		if concrete, ok := reflect.TypeAssert[map[int64]string](m); ok {
			concrete[key] = value
			return
		}
		keyReflectValue := intMapKeyScratch(vm, m.Type().Key())
		keyReflectValue.SetInt(key)
		valueReflectValue := reflect.ValueOf(value).Convert(m.Type().Elem())
		m.SetMapIndex(keyReflectValue, valueReflectValue)
	}); panicked {
		return raiseNativePanicAsInterpreted(vm, recovered)
	}
	return opContinue
}

// handleMapAddIntInt handles opMapAddIntInt.
//
// Performs general[A][ints[B]] += ints[C] for map[int]int (or map[int64]int64). Absent
// keys are treated as 0. Fuses get+add+set into one dispatch with a single map probe,
// eliminating the redundant hash and bucket walk that the unfused get/add/set sequence
// pays.
//
// Takes registers (*Registers) which holds the map, key, and delta.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapAddIntInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapAddIntInt", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.ints[instruction.b]
	delta := registers.ints[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[int]int](m); ok {
		concrete[int(key)] += int(delta)
		return opContinue
	}
	if concrete, ok := reflect.TypeAssert[map[int64]int64](m); ok {
		concrete[key] += delta
		return opContinue
	}
	keyReflectValue := intMapKeyScratch(vm, m.Type().Key())
	keyReflectValue.SetInt(key)
	current := int64(0)
	if existing := m.MapIndex(keyReflectValue); existing.IsValid() {
		current = existing.Int()
	}
	valueReflectValue := reflect.New(m.Type().Elem()).Elem()
	valueReflectValue.SetInt(current + delta)
	m.SetMapIndex(keyReflectValue, valueReflectValue)
	return opContinue
}

// handleMapAddStringInt handles opMapAddStringInt.
//
// Performs general[A][strings[B]] += ints[C] for map[string]int (or map[string]int64).
// Absent keys are treated as 0. Fuses get+add+set into one dispatch with one hash of the
// key, vs the two probes the unfused sequence performs.
//
// Takes registers (*Registers) which holds the map, key, and delta.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapAddStringInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapAddStringInt", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	rawKey := registers.strings[instruction.b]
	key := internMapKey(vm, rawKey)
	delta := registers.ints[instruction.c]
	if concrete, ok := reflect.TypeAssert[map[string]int](m); ok {
		concrete[key] += int(delta)
		return opContinue
	}
	if concrete, ok := reflect.TypeAssert[map[string]int64](m); ok {
		concrete[key] += delta
		return opContinue
	}
	keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
	current := int64(0)
	if existing := m.MapIndex(keyReflectValue); existing.IsValid() {
		current = existing.Int()
	}
	valueReflectValue := reflect.New(m.Type().Elem()).Elem()
	valueReflectValue.SetInt(current + delta)
	m.SetMapIndex(keyReflectValue, valueReflectValue)
	return opContinue
}

// handleMapGetStringGeneral handles opMapGetStringGeneral.
//
// Reads general[A] = map[string]V in general[B] with key strings[C]. Mirrors
// handleMapGetIntGeneral for string keys: routes through runtime.mapaccess2_faststr (via
// mapAccessFastStrToGeneral) to skip the per-call key-boxing allocation that
// reflect.Value.MapIndex performs.
//
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapGetStringGeneral(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapGetStringGeneral", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := registers.strings[instruction.c]
	elemType := m.Type().Elem()
	if !useMapFastLinkname() {
		result := m.MapIndex(reflect.ValueOf(key))
		if !result.IsValid() {
			registers.general[instruction.a] = zeroValueForType(elemType)
		} else {
			registers.general[instruction.a] = result
		}
		return opContinue
	}
	resultPtr, present := mapAccessFastStrToGeneral(m, key)
	if !present {
		registers.general[instruction.a] = zeroValueForType(elemType)
		return opContinue
	}
	registers.general[instruction.a] = wrapMapElemFast(elemType, resultPtr)
	return opContinue
}

// handleMapIndexOkStringGeneral handles opMapIndexOkStringGeneral.
//
// Reads general[A] = map[string]V in general[B] with key strings[C], plus an extension
// word whose A field is the int register holding the ok flag. Mirrors
// handleMapIndexOkIntGeneral for string keys.
//
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which holds the map and key.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapIndexOkStringGeneral(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkStringGeneral", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.strings[instruction.c]
	elemType := m.Type().Elem()
	if !useMapFastLinkname() {
		result := m.MapIndex(reflect.ValueOf(key))
		if !result.IsValid() {
			registers.general[instruction.a] = zeroValueForType(elemType)
			registers.ints[extensionWord.a] = 0
		} else {
			registers.general[instruction.a] = result
			registers.ints[extensionWord.a] = 1
		}
		return opContinue
	}
	resultPtr, present := mapAccessFastStrToGeneral(m, key)
	if !present {
		registers.general[instruction.a] = zeroValueForType(elemType)
		registers.ints[extensionWord.a] = 0
		return opContinue
	}
	registers.general[instruction.a] = wrapMapElemFast(elemType, resultPtr)
	registers.ints[extensionWord.a] = 1
	return opContinue
}

// handleMapSetStringGeneral handles opMapSetStringGeneral.
//
// Writes general[A][strings[B]] = general[C] for map[string]V where V is in the general
// bank. Routes through runtime.mapassign_faststr and runtime.typedmemmove to skip the
// per-call allocation in reflect.Value.SetMapIndex. The source pointer for typedmemmove
// depends on the reflect.Value's storage shape: when flagIndir is set the underlying ptr
// addresses the value bytes directly (slices, strings, structs, addressable inline
// values); when flagIndir is clear and the kind is a direct-interface kind (Pointer, Map,
// Chan, Func, UnsafePointer) ptr IS the value word and the copy reads from the address of
// valueRaw.ptr itself. A nil slotPtr indicates a nil map; the reflect.Value.SetMapIndex
// fallback raises the matching panic which guardChannelOp converts. The 32-byte zero
// buffer covers every general-bank element shape (slice header, eface, string).
//
// Takes vm (*VM) which surfaces evalError on nil-map writes.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the map, key, and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapSetStringGeneral(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapSetStringGeneral", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	key := materialiseString(vm.arena, registers.strings[instruction.b])
	value := materialiseArenaValue(vm.arena, coerceValue(vm, registers.general[instruction.c], m.Type().Elem()))
	if recovered, panicked := guardChannelOp(func() {
		slotPtr := mapAssignFastStr(m, key)
		if slotPtr == nil {
			keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
			m.SetMapIndex(keyReflectValue, value)
			return
		}
		valueRaw := (*unsafeReflectValue)(unsafe.Pointer(&value))
		elemType := m.Type().Elem()
		elemABIType := reflectValueABIType(elemType)
		if !value.IsValid() {
			var zero [32]byte
			runtimeTypedmemmove(elemABIType, slotPtr, unsafe.Pointer(&zero[0]))
			return
		}
		var srcPtr unsafe.Pointer
		if valueRaw.flag&flagIndir != 0 {
			srcPtr = valueRaw.ptr
		} else {
			srcPtr = unsafe.Pointer(&valueRaw.ptr)
		}
		runtimeTypedmemmove(elemABIType, slotPtr, srcPtr)
	}); panicked {
		return raiseNativePanicAsInterpreted(vm, recovered)
	}
	return opContinue
}
