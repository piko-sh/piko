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

// safeBuildDetachSliceHeader detaches a slice header from the arena.
//
// Rewraps v's slice header on the heap so the reflect.Value survives an arena release.
// Used by safe-build closure-cell writes: when a native iter.Seq (for example
// `maps.Keys(m)`) invokes a piko yield closure, the wrapper at makeClosureWrapperFunc
// spawns a freshVM whose arena is released the moment the closure returns. If the closure
// stored a slice in a shared upvalue cell whose reflect.Value points into the freshVM's
// sliceHeaderSlab, the slot is wiped by arena.Reset and the cell observes (Data, Len,
// Cap) = (0, 0, 0) thereafter.
//
// The unsafe build avoids this naturally: materialiseArenaSliceUnconditional's
// fresh-backing branch fires (because ownsSliceHeaderPointer can precisely detect
// arena-resident headers) and produces a heap-managed reflect.Value before storage. Safe
// build's ownsSliceHeaderPointer always returns false, so the materialise path returns v
// unchanged; this helper supplies the missing detach step.
//
// Takes v (reflect.Value) which is about to be stored into a long-lived cell.
//
// Returns reflect.Value which is the heap-anchored equivalent for slice kinds, or v
// itself otherwise. No-op in the unsafe build.
func safeBuildDetachSliceHeader(v reflect.Value) reflect.Value {
	if arenaUsesUnsafeSlabs {
		return v
	}
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return v
	}
	header := &arenaSliceHeader{
		Data: unsafe.Pointer(v.Pointer()),
		Len:  v.Len(),
		Cap:  v.Cap(),
	}
	return unsafeNewAt(reflectValueABIType(v.Type()), unsafe.Pointer(header), reflect.Slice)
}
