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

//go:build safe

package interp_domain

import "reflect"

// ensureSliceAddressableForHeader promotes non-addressable slices.
//
// Wraps a non-addressable slice in a heap-backed addressable holder so the safe-build
// reflectValuePtr can return a valid 24-byte header pointer.
//
// Required because the safe-build reflectValuePtr only returns Addr().UnsafePointer() for
// addressable Values; without promotion, snapshotSliceHeaderArena would receive a nil
// pointer when callers (for example coerceClosureToFunc returning a slice from a
// piko-internal closure) hand it a temporary slice constructed via reflect.MakeSlice or
// similar.
//
// Returning a reflect.Value (rather than a raw unsafe.Pointer from a helper) keeps the
// heap-promoted storage GC-rooted across the subsequent header read in
// snapshotSliceHeaderArena.
//
// The unsafe-build sibling is a no-op; its reflectValuePtr reads the internal .ptr field
// of the reflect.Value layout directly and is never nil for a valid slice.
//
// Takes v (reflect.Value) which is the slice value to promote.
//
// Returns reflect.Value which is the addressable equivalent, or v unchanged when already
// addressable.
func ensureSliceAddressableForHeader(v reflect.Value) reflect.Value {
	if v.CanAddr() {
		return v
	}
	holder := reflect.New(v.Type()).Elem()
	holder.Set(v)
	return holder
}
