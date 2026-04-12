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

//go:build !safe && !(js && wasm)

package interp_domain

import (
	"reflect"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// runtimeMapaccess2Fast64 is a direct linkname to runtime.mapaccess2_fast64.
//
// reflect.Value.MapIndex always dispatches via the generic mapaccess path even for
// int64-keyed maps where the runtime has a specialised mapaccess2_fast64 that skips the
// key-hasher dispatch. Going direct here bypasses reflect's key.assignTo coercion and the
// generic hasher. Safety: signatures must match runtime.mapaccess2_fast64 exactly, which
// expects *abi.MapType and *internal/runtime/maps.Map. Both are internal; we treat them
// as unsafe.Pointer because the runtime only reads from them. The *abi.MapType is the
// same starting layout as *abi.Type (reflect.Value.typ() gives this for a Map kind
// value), and the *Map pointer is what reflect.Value.UnsafePointer returns for a Map kind
// value.
//
// Takes t (unsafe.Pointer) which is the *abi.MapType describing the map.
// Takes m (unsafe.Pointer) which is the *runtime.Map.
// Takes key (uint64) which is the int64 / uint64 key.
//
// Returns the value slot pointer and the present flag.
//
//go:linkname runtimeMapaccess2Fast64 runtime.mapaccess2_fast64
//go:noescape
func runtimeMapaccess2Fast64(t unsafe.Pointer, m unsafe.Pointer, key uint64) (unsafe.Pointer, bool)

// runtimeMapassignFast64 looks up or allocates a slot for a uint64 key.
//
// Caller must initialise the returned slot via runtimeTypedmemmove with the map's value
// type.
//
// Takes t (unsafe.Pointer) which is the *abi.MapType describing the map.
// Takes m (unsafe.Pointer) which is the *runtime.Map.
// Takes key (uint64) which is the int64 / uint64 key.
//
// Returns the pointer to the value slot.
//
//go:linkname runtimeMapassignFast64 runtime.mapassign_fast64
//go:noescape
func runtimeMapassignFast64(t unsafe.Pointer, m unsafe.Pointer, key uint64) unsafe.Pointer

// runtimeTypedmemmove copies a value of type t to destination from source.
//
// Emits the GC write barriers that reflect.Value.Set would emit. Used by
// handleSetStructFieldGeneralT0 to write an interface field directly from a Pointer-kind
// reflect.Value without paying reflect's full Set dispatch path. Safety: t must
// accurately describe the bytes at destination and source. For these callers, t is the
// *abi.Type of the field's declared type, sourced from
// frame.function.typeTable[layout.FieldTypeIndex].
//
// Takes t (unsafe.Pointer) which is the *abi.Type describing the value.
// Takes destination (unsafe.Pointer) which is the destination slot.
// Takes source (unsafe.Pointer) which is the source slot.
//
//go:linkname runtimeTypedmemmove runtime.typedmemmove
//go:noescape
func runtimeTypedmemmove(t unsafe.Pointer, destination unsafe.Pointer, source unsafe.Pointer)

// runtimeMapaccess2Faststr looks up a string-keyed map entry.
//
// Same signature shape as runtimeMapaccess2Fast64 but specialised for the
// runtime.aeshashstr / strhash hash path that string keys use, avoiding
// reflect.Value.MapIndex's key-boxing allocation entirely.
//
// Takes t (unsafe.Pointer) which is the *abi.MapType describing the map.
// Takes m (unsafe.Pointer) which is the *runtime.Map.
// Takes key (string) which is the string lookup key.
//
// Returns the value slot pointer and the present flag.
//
//go:linkname runtimeMapaccess2Faststr runtime.mapaccess2_faststr
//go:noescape
func runtimeMapaccess2Faststr(t unsafe.Pointer, m unsafe.Pointer, key string) (unsafe.Pointer, bool)

// runtimeMapassignFaststr looks up or allocates a slot for a string key.
//
// Caller must initialise the returned slot via runtimeTypedmemmove with the map's value
// type.
//
// Takes t (unsafe.Pointer) which is the *abi.MapType describing the map.
// Takes m (unsafe.Pointer) which is the *runtime.Map.
// Takes key (string) which is the string key.
//
// Returns the pointer to the value slot.
//
//go:linkname runtimeMapassignFaststr runtime.mapassign_faststr
//go:noescape
func runtimeMapassignFaststr(t unsafe.Pointer, m unsafe.Pointer, key string) unsafe.Pointer

// mapAccessFast64ToGeneral does a direct runtime.mapaccess2_fast64 against an int64-keyed
// map and stores the result + ok flag into the destination register banks. Used by
// handleMapIndexOkIntGeneral for int-keyed map lookups in the general bank.
//
// Resolves the underlying *Map pointer from the reflect.Value's storage shape: when
// flagIndir is set (addressable map slot, e.g. read from a struct field via unsafeNewAt)
// rv.ptr is the storage location of the *Map and is dereferenced once; otherwise rv.ptr
// IS the *Map pointer directly.
//
// Takes mapValue (reflect.Value) which must be a Map kind value whose key type is exactly
// int64 / uint64 / int (8-byte int).
// Takes key (int64) which is the lookup key.
//
// Returns the value slot pointer (cast to unsafe.Pointer of the map's element type) and
// the present flag.
//
// Returns (nil, false) when caller should fall back to the reflect path (map is nil, key
// type isn't 8-byte int, etc.).
func mapAccessFast64ToGeneral(mapValue reflect.Value, key int64) (unsafe.Pointer, bool) {
	if !mapValue.IsValid() {
		return nil, false
	}
	rv := (*unsafeReflectValue)(unsafe.Pointer(&mapValue))
	if rv.typ == nil {
		return nil, false
	}
	var mapPtr unsafe.Pointer
	if rv.flag&flagIndir != 0 {
		mapPtr = *(*unsafe.Pointer)(rv.ptr)
	} else {
		mapPtr = rv.ptr
	}
	if mapPtr == nil {
		return nil, false
	}
	resultPtr, ok := runtimeMapaccess2Fast64(rv.typ, mapPtr, safeconv.Int64ToUint64Reinterpret(key))
	return resultPtr, ok
}

// mapAccessFastStrToGeneral reads a string-keyed map via the runtime fast path.
//
// Mirrors mapAccessFast64ToGeneral exactly; only the linkname symbol and the key type
// differ. Used by handleMapGetStringGeneral and handleMapIndexOkStringGeneral. Returns
// (nil, false) when caller should fall back to the reflect path (map is nil, etc.).
//
// Takes mapValue (reflect.Value) which must be a Map kind value with string keys.
// Takes key (string) which is the lookup key.
//
// Returns the value slot pointer and the present flag.
func mapAccessFastStrToGeneral(mapValue reflect.Value, key string) (unsafe.Pointer, bool) {
	if !mapValue.IsValid() {
		return nil, false
	}
	rv := (*unsafeReflectValue)(unsafe.Pointer(&mapValue))
	if rv.typ == nil {
		return nil, false
	}
	var mapPtr unsafe.Pointer
	if rv.flag&flagIndir != 0 {
		mapPtr = *(*unsafe.Pointer)(rv.ptr)
	} else {
		mapPtr = rv.ptr
	}
	if mapPtr == nil {
		return nil, false
	}
	resultPtr, ok := runtimeMapaccess2Faststr(rv.typ, mapPtr, key)
	return resultPtr, ok
}

// mapAssignFastStr returns the value slot pointer for a string key.
//
// Allocates the slot if absent. Caller must initialise the slot via runtimeTypedmemmove
// with the map's value-type abi pointer. Mirrors the read-side helpers; used by
// handleMapSetStringGeneral.
//
// Takes mapValue (reflect.Value) which must be a Map kind value with string keys.
// Takes key (string) which is the key to assign.
//
// Returns the value slot pointer, or nil when the map is invalid or nil.
func mapAssignFastStr(mapValue reflect.Value, key string) unsafe.Pointer {
	if !mapValue.IsValid() {
		return nil
	}
	rv := (*unsafeReflectValue)(unsafe.Pointer(&mapValue))
	if rv.typ == nil {
		return nil
	}
	var mapPtr unsafe.Pointer
	if rv.flag&flagIndir != 0 {
		mapPtr = *(*unsafe.Pointer)(rv.ptr)
	} else {
		mapPtr = rv.ptr
	}
	if mapPtr == nil {
		return nil
	}
	return runtimeMapassignFaststr(rv.typ, mapPtr, key)
}

// useMapFastLinkname reports whether the linkname-backed map fast paths are available.
//
// Returns true on the unsafe build.
func useMapFastLinkname() bool {
	return true
}
