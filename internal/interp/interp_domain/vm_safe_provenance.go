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
)

// pointerBound records the valid byte window of the allocation an interpreted unsafe
// pointer was derived from.
//
// It backs the runtime safe mode's bounds checks so a script cannot read or write outside
// the origin object via unsafe pointer arithmetic. It is distinct from the "safe" build
// tag.
type pointerBound struct {
	// base is the first valid byte address of the origin allocation.
	base uintptr

	// limit is the one-past-end byte address of the origin allocation.
	limit uintptr

	// elemSize is the size in bytes of the origin pointer's element type.
	elemSize uintptr
}

// recordPointerProvenance records the origin window for the raw address held by pointer
// and pins the pointee for the rest of the execution so the recorded address stays valid
// (the side-table holds only uintptrs, which the GC ignores). It is a no-op outside safe
// mode or for invalid/nil pointers.
//
// Takes pointer (reflect.Value) which is the pointer whose address is the key.
// Takes bound (pointerBound) which is the origin window to record.
func (vm *VM) recordPointerProvenance(pointer reflect.Value, bound pointerBound) {
	if !vm.limits.safeMode || !pointer.IsValid() {
		return
	}
	address := pointer.Pointer()
	if address == 0 {
		return
	}
	if vm.pointerProvenance == nil {
		vm.pointerProvenance = make(map[uintptr]pointerBound)
	}
	vm.pointerProvenance[address] = bound
	vm.pointerProvenanceKeepAlive = append(vm.pointerProvenanceKeepAlive, pointer)
}

// lookupPointerProvenance returns the recorded origin window for a raw address.
//
// Takes address (uintptr) which is the raw pointer address to look up.
//
// Returns pointerBound which is the recorded window.
// Returns bool which reports whether an entry was found. A missing entry marks an
// external pointer (from a uintptr round-trip or a native return) that the unsafe
// handlers refuse to operate on.
func (vm *VM) lookupPointerProvenance(address uintptr) (pointerBound, bool) {
	if vm.pointerProvenance == nil {
		return pointerBound{}, false
	}
	bound, found := vm.pointerProvenance[address]
	return bound, found
}

// resetPointerProvenance clears the side-table between executions on a reused VM so
// recorded windows never describe a later allocation that reused an address.
func (vm *VM) resetPointerProvenance() {
	if len(vm.pointerProvenance) > 0 {
		clear(vm.pointerProvenance)
	}
	vm.pointerProvenanceKeepAlive = vm.pointerProvenanceKeepAlive[:0]
}

// checkUnsafeSpan reports a runtime panic error when a safe-mode access of span bytes
// starting at address would fall outside the origin window recorded for that address.
//
// A pointer with no recorded origin (external: from a uintptr round-trip or a native
// return) is refused. It returns nil when safe mode is off or the access is in bounds.
//
// Takes operation (string) which names the unsafe operation for the error message.
// Takes address (uintptr) which is the start address of the access.
// Takes span (uintptr) which is the number of bytes the access touches.
//
// Returns error which is non-nil when the access is out of bounds or external.
func (vm *VM) checkUnsafeSpan(operation string, address, span uintptr) error {
	if !vm.limits.safeMode {
		return nil
	}
	bound, found := vm.lookupPointerProvenance(address)
	if !found {
		return newRuntimePanicError("interp: %s: %v (pointer has no tracked origin)", operation, errUnsafeBounds)
	}
	if address < bound.base || address > bound.limit || span > bound.limit-address {
		return newRuntimePanicError("interp: %s: %v", operation, errUnsafeBounds)
	}
	return nil
}

// convertUnsafePointerChecked performs an unsafe.Pointer conversion, applying the
// safe-mode provenance tracking and uintptr rejection when safe mode is on and delegating
// to the plain conversion otherwise.
//
// Takes source (reflect.Value) which is the value being converted.
// Takes destinationType (reflect.Type) which is the conversion target.
//
// Returns reflect.Value which is the converted pointer.
// Returns bool which reports whether the conversion succeeded.
func (vm *VM) convertUnsafePointerChecked(source reflect.Value, destinationType reflect.Type) (reflect.Value, bool) {
	if !vm.limits.safeMode {
		return convertUnsafePointer(source, destinationType), true
	}
	return vm.safeModeConvertUnsafePointer(source, destinationType)
}

// safeModeConvertUnsafePointer performs an unsafe.Pointer conversion under safe mode,
// tracking provenance and rejecting the unbounded uintptr-pointer round-trip.
//
// For *T -> unsafe.Pointer it records a fresh origin window derived from the source
// pointer's element type, unless the address already carries a propagated origin (from an
// earlier unsafe.Add). The unsafe.Pointer -> *T direction needs no action: the side-table
// is keyed by address, which a conversion preserves, so an existing origin is retained
// and an external pointer (no entry) stays external and so is refused by a later
// unsafe.Slice/Add.
//
// Takes source (reflect.Value) which is the value being converted.
// Takes destinationType (reflect.Type) which is the conversion target.
//
// Returns reflect.Value which is the converted pointer.
// Returns bool which reports whether the conversion succeeded; on rejection it sets
// vm.evalError and returns false.
func (vm *VM) safeModeConvertUnsafePointer(source reflect.Value, destinationType reflect.Type) (reflect.Value, bool) {
	if destinationType.Kind() == reflect.Uintptr || source.Kind() == reflect.Uintptr {
		vm.evalError = newRuntimePanicError("interp: %v (uintptr-pointer conversion is not permitted in safe mode)", errUnsafeBounds)
		return reflect.Value{}, false
	}

	result := convertUnsafePointer(source, destinationType)

	if destinationType == unsafePointerType && source.Kind() == reflect.Pointer {
		address := result.Pointer()
		if _, found := vm.lookupPointerProvenance(address); !found {
			elementSize := source.Type().Elem().Size()
			vm.recordPointerProvenance(result, pointerBound{base: address, limit: address + elementSize, elemSize: elementSize})
		}
	}

	return result, true
}
