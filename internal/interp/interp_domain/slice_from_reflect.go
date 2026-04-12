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

// sliceFromReflectField extracts a typed slice from a reflect.Value.
//
// Unlike `reflect.TypeAssert[[]T](v)` it accepts any element type whose memory layout
// matches T's (so `[]int` reads as `[]int64` on a 64-bit host, named `[]MyInt` types
// work, etc.). Avoids `Interface()`, so it tolerates flagRO reflect.Values that come from
// reading unexported struct fields.
//
// Used by vm_handler_struct_field_safe.go for typed-slice struct-field reads when the
// field is unexported, and by vm_value_transfer.go for typed-slice argument assignment
// when a native callback passes back a slice whose element type doesn't exactly match the
// ARCH5-promoted typed-bank kind (the closure parameter expects `[]int64`/`[]float64`
// etc. but the native library yielded `[]int`/`[]float32` etc.).
//
// Takes field (reflect.Value) which is the slice value to extract.
//
// Returns the typed slice and true on success; (nil, false) when the element layout does
// not match T or the value is not a slice.
func sliceFromReflectField[T any](field reflect.Value) ([]T, bool) {
	if field.Kind() != reflect.Slice {
		return nil, false
	}
	elemType := field.Type().Elem()
	var zero T
	zeroType := reflect.TypeOf(zero)
	if elemType != zeroType && elemType.Size() != zeroType.Size() {
		return nil, false
	}
	length := field.Len()
	capacity := field.Cap()
	if capacity == 0 {
		return nil, true
	}

	data := (*T)(field.UnsafePointer())
	return unsafe.Slice(data, capacity)[:length:capacity], true
}
