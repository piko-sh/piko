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

package driven_system_symbols

import "reflect"

// coerce converts a value to the target type T, bridging the impedance
// mismatch between VM register types and concrete dispatch wrapper types.
//
// Takes v (any) which is the value to convert.
//
// Returns the converted value of type T, or the zero value if conversion
// fails.
func coerce[T any](v any) T {
	if typed, ok := v.(T); ok {
		return typed
	}
	converted, ok := reflect.TypeAssert[T](reflect.ValueOf(v).Convert(reflect.TypeFor[T]()))
	if !ok {
		var zero T
		return zero
	}
	return converted
}
