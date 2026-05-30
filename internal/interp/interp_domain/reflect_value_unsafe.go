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

//go:build go1.26 && !safe && !(js && wasm)

package interp_domain

import (
	"reflect"
	"unsafe"
)

const (
	// flagKindMask masks the low 5 bits where reflect.Kind is stored.
	flagKindMask uintptr = 1<<5 - 1

	// flagIndir indicates that v.ptr is a *T rather than the value itself.
	// reflect.NewAt(elem, p).Elem() sets this because p points at the element's storage.
	flagIndir uintptr = 1 << 7

	// flagAddr indicates that the Value can have its address taken (CanAddr() returns true).
	// reflect.NewAt(elem, p).Elem() sets this because the storage at p is settable via Elem.
	flagAddr uintptr = 1 << 8
)

// unsafeReflectValue mirrors the runtime layout of reflect.Value on Go 1.26.
//
// The three machine words are typ_, ptr, flag. The layout is verified by
// TestReflectValueLayout. Do not change the field order or types without updating that
// test.
type unsafeReflectValue struct {
	// typ points at the *abi.Type for the element type.
	typ unsafe.Pointer

	// ptr is the settable storage of the element value.
	ptr unsafe.Pointer

	// flag carries flagAddr | flagIndir | Kind(elem).
	flag uintptr
}

// emptyInterface mirrors the runtime layout of an interface header. reflect.Type is an
// interface whose dynamic type is *abi.Type; the data word IS the *abi.Type pointer.
type emptyInterface struct {
	// typ is the dynamic type pointer for the interface header.
	typ unsafe.Pointer

	// data is the dynamic value word; for reflect.Type it IS the *abi.Type pointer.
	data unsafe.Pointer
}

// reflectValueABIType extracts the *abi.Type pointer from a reflect.Type. Compile-time
// callers pre-derive this once per type and pass the cached pointer on the hot path.
//
// Takes t (reflect.Type) which is the element type.
//
// Returns the underlying *abi.Type pointer (an unsafe.Pointer so no abi import is
// required).
func reflectValueABIType(t reflect.Type) unsafe.Pointer {
	if t == nil {
		return nil
	}
	return (*emptyInterface)(unsafe.Pointer(&t)).data
}

// reflectValuePtr extracts the internal .ptr field of a reflect.Value.
//
// For an addressable Value (flagAddr set, flagIndir set) this is the address of the
// value's storage - the same pointer reflect.Value.Addr().UnsafePointer() returns, but
// without reflect.Value.Addr's internal ptrTo() lookup. The caller is responsible for
// ensuring the source Value is addressable and indirect.
//
// Takes v (reflect.Value) which must be addressable+indirect (the shape returned by
// reflect.Value.Field on an addressable struct).
//
// Returns the storage address as an unsafe.Pointer.
func reflectValuePtr(v reflect.Value) unsafe.Pointer {
	return (*unsafeReflectValue)(unsafe.Pointer(&v)).ptr
}

// wrapTypedSliceSrcType extracts the *abi.Type pointer from a slice- typed reflect.Value
// via a single layout-cast field load. Used by the typed-append fast paths to avoid the
// cost of reflect.Value.Type() + reflectValueABIType in the unsafe build's hot path.
//
// Takes v (reflect.Value) which is the source slice value.
//
// Returns the abi.Type pointer suitable for unsafeNewAt.
func wrapTypedSliceSrcType(v reflect.Value) unsafe.Pointer {
	return (*unsafeReflectValue)(unsafe.Pointer(&v)).typ
}

// unsafeNewAt is the unsafe analogue of reflect.NewAt(elem, p).Elem(). It constructs a
// reflect.Value of the element type referenced by cachedABIType, pointing at storage p,
// with the same flags (flagAddr | flagIndir | Kind(elem)) that NewAt(elem, p).Elem()
// produces.
//
// The caller is responsible for pre-caching cachedABIType via reflectValueABIType for the
// same element type T. Passing the wrong *abi.Type silently produces a Value whose Kind
// disagrees with its storage, which can corrupt memory.
//
// Takes cachedABIType (unsafe.Pointer) which is the *abi.Type of the element type T. NOT
// the *T type.
// Takes p (unsafe.Pointer) which is the address of the element storage. May not be nil;
// .Elem() on a nil pointer is invalid.
// Takes kind (reflect.Kind) which is the reflect.Kind of T.
//
// Returns a reflect.Value behaviourally equivalent to reflect.NewAt(T, p).Elem().
func unsafeNewAt(cachedABIType unsafe.Pointer, p unsafe.Pointer, kind reflect.Kind) reflect.Value {
	v := unsafeReflectValue{
		typ:  cachedABIType,
		ptr:  p,
		flag: uintptr(kind) | flagAddr | flagIndir,
	}
	return *(*reflect.Value)(unsafe.Pointer(&v))
}

// unsafeReadOnlyValue is the same as unsafeNewAt but omits flagAddr, producing a
// non-addressable Value. Used by snapshotPointerLeaf to construct a detached copy of a
// pointer field's value without the reflect.New allocation that the safe alternative
// requires.
//
// Takes cachedABIType (unsafe.Pointer) which is the *abi.Type of T.
// Takes p (unsafe.Pointer) which is storage holding the T value.
// Takes kind (reflect.Kind) which is the reflect.Kind of T.
//
// Returns a non-addressable Value of type T at p.
func unsafeReadOnlyValue(cachedABIType unsafe.Pointer, p unsafe.Pointer, kind reflect.Kind) reflect.Value {
	v := unsafeReflectValue{
		typ:  cachedABIType,
		ptr:  p,
		flag: uintptr(kind) | flagIndir,
	}
	return *(*reflect.Value)(unsafe.Pointer(&v))
}

// unsafePointerKindValue constructs a Pointer-kind reflect.Value whose internal ptr field
// IS the pointer value (no indirection, no addressability). This matches the layout
// reflect uses for values produced by reflect.ValueOf(somePointerVar): for Pointer kinds
// the Value's ptr field is the pointer VALUE itself, not a pointer to the pointer's
// storage.
//
// Used by snapshotPointerLeaf to detach a pointer-field Value from its struct backing
// storage without allocating a new heap location to hold the pointer value.
//
// Takes cachedABIType (unsafe.Pointer) which is the *abi.Type of *T (i.e. the pointer
// type, not the element type). pass the type pointer from field.Type() where field has
// Kind == Pointer.
// Takes ptrValue (unsafe.Pointer) which is the pointer value itself (what the user code
// would observe as a `*T` value).
//
// Returns a non-addressable Pointer-kind Value carrying ptrValue.
func unsafePointerKindValue(cachedABIType unsafe.Pointer, ptrValue unsafe.Pointer) reflect.Value {
	v := unsafeReflectValue{
		typ:  cachedABIType,
		ptr:  ptrValue,
		flag: uintptr(reflect.Pointer),
	}
	return *(*reflect.Value)(unsafe.Pointer(&v))
}

// snapshotSliceHeader mirrors the runtime layout of a Go slice header.
//
// The three machine words are Data, Len, Cap. Used as the heap-allocated storage for a
// slice-field snapshot in handleGetField. The Data field is `unsafe.Pointer` so the
// garbage collector tracks the backing array as reachable through this header.
type snapshotSliceHeader struct {
	// Data points at the slice's element backing array.
	Data unsafe.Pointer

	// Len is the slice's logical length.
	Len int

	// Cap is the slice's underlying capacity.
	Cap int
}

// unsafeDirectIfaceKindValue constructs a non-addressable reflect.Value for a
// direct-iface kind (Pointer, Map, Chan, Func, UnsafePointer) where the 1-word value is
// held inline in the Value's ptr field - the same layout
// reflect.ValueOf(somePointerOrMap) produces. No flagIndir, no flagAddr.
//
// This is the alloc-free analogue of `reflect.New(t).Elem().Set(field)` for direct-iface
// kinds: it copies the 1-word value out of the originating storage and packs it into a
// fresh Value without touching the heap.
//
// Takes cachedABIType (unsafe.Pointer) which is the *abi.Type of T (the field's own type
// - e.g. *Token for a Pointer field, map[K]V for a Map field).
// Takes value (unsafe.Pointer) which is the 1-word value (the pointer / *hmap / *hchan /
// *funcVal as it appears in user code).
// Takes kind (reflect.Kind) which must be one of reflect.Pointer, reflect.Map,
// reflect.Chan, reflect.Func, reflect.UnsafePointer.
//
// Returns a non-addressable Value of type T carrying value.
func unsafeDirectIfaceKindValue(cachedABIType unsafe.Pointer, value unsafe.Pointer, kind reflect.Kind) reflect.Value {
	v := unsafeReflectValue{
		typ:  cachedABIType,
		ptr:  value,
		flag: uintptr(kind),
	}
	return *(*reflect.Value)(unsafe.Pointer(&v))
}

// resetSafeTypeRegistryForTest is the unsafe-build no-op sibling of the function in
// reflect_value_safe.go. The unsafe build derives reflect.Value internals directly from
// runtime headers and keeps no package-level dedup maps, so there is nothing to clear
// here.
func resetSafeTypeRegistryForTest() {}
