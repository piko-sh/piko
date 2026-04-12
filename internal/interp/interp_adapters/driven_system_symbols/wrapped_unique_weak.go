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

import (
	"fmt"
	"reflect"
	"unique"
	"weak"
)

// wrappedUniqueMake bridges unique.Make[T comparable] for interpreted code.
//
// Because reflect.ValueOf cannot capture an uninstantiated generic, the implementation
// routes through fast-path type switches for common comparable kinds and panics on rare
// ones (string, integer, float, bool, complex, channel, pointer, interface, struct of
// comparable fields). Interpreted code accesses the underlying value via Handle.Value()
// (also a generic method whose receiver type is preserved through reflect).
//
// Takes value (any) the value to canonicalise.
//
// Returns any which is the typed unique.Handle[T] wrapped as any.
//
// Panics with "unique.Make: unsupported type %T" when the runtime type is not in the
// fast-path set.
func wrappedUniqueMake(value any) any {
	switch typed := value.(type) {
	case string:
		return unique.Make(typed)
	case int:
		return unique.Make(typed)
	case int8:
		return unique.Make(typed)
	case int16:
		return unique.Make(typed)
	case int32:
		return unique.Make(typed)
	case int64:
		return unique.Make(typed)
	case uint:
		return unique.Make(typed)
	case uint8:
		return unique.Make(typed)
	case uint16:
		return unique.Make(typed)
	case uint32:
		return unique.Make(typed)
	case uint64:
		return unique.Make(typed)
	case uintptr:
		return unique.Make(typed)
	case float32:
		return unique.Make(typed)
	case float64:
		return unique.Make(typed)
	case complex64:
		return unique.Make(typed)
	case complex128:
		return unique.Make(typed)
	case bool:
		return unique.Make(typed)
	}
	panic(fmt.Sprintf("unique.Make: unsupported type %T (must be a built-in comparable)", value))
}

// wrappedWeakMake bridges weak.Make[T any] for interpreted code.
//
// Go's reflect package cannot instantiate a generic at runtime, so the only mechanism for
// runtime dispatch is a static type switch over every concrete pointer kind. We cover
// every built-in primitive and the most commonly-pointed-to standard composites (slices,
// strings, maps, any) here. Arbitrary user struct pointers - the cases the stdlib's
// caches and canonicalisation maps actually target - would require either code generation
// per user type or an unsafe.Pointer bridge that bypasses generic typing (and therefore
// loses Handle's per-T identity guarantees). The current contract is: works for any
// pointer-to-built-in; panics with an actionable message otherwise.
//
// Takes pointer (any) which must be a non-nil pointer to a built-in.
//
// Returns the weak.Pointer[T] wrapped as any.
//
// Panics with "weak.Make: argument must be a pointer" when pointer is not a pointer kind,
// or "weak.Make: unsupported pointer element type" when the element kind isn't in the
// supported set.
func wrappedWeakMake(pointer any) any {
	rv := reflect.ValueOf(pointer)
	if rv.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("weak.Make: argument must be a pointer, got %T", pointer))
	}
	if handle, ok := weakMakeScalarPointer(pointer); ok {
		return handle
	}
	if handle, ok := weakMakeCompositePointer(pointer); ok {
		return handle
	}
	panic(fmt.Sprintf("weak.Make: unsupported pointer element type %T - add to wrapped_unique_weak.go fast-path list, or wrap your value in *any before calling weak.Make", pointer))
}

// weakMakeScalarPointer handles pointers to the built-in scalar kinds (booleans, strings,
// every sized integer, floats, and complex). Each case instantiates weak.Make with the
// concrete element type so the returned handle preserves its per-T identity.
//
// Takes pointer (any) a non-nil pointer to a built-in scalar.
//
// Returns the weak.Pointer[T] wrapped as any and true when matched; nil and false
// otherwise.
func weakMakeScalarPointer(pointer any) (any, bool) {
	switch typed := pointer.(type) {
	case *string:
		return weak.Make(typed), true
	case *bool:
		return weak.Make(typed), true

	case *int:
		return weak.Make(typed), true
	case *int8:
		return weak.Make(typed), true
	case *int16:
		return weak.Make(typed), true
	case *int32:
		return weak.Make(typed), true
	case *int64:
		return weak.Make(typed), true

	case *uint:
		return weak.Make(typed), true
	case *uint8:
		return weak.Make(typed), true
	case *uint16:
		return weak.Make(typed), true
	case *uint32:
		return weak.Make(typed), true
	case *uint64:
		return weak.Make(typed), true
	case *uintptr:
		return weak.Make(typed), true

	case *float32:
		return weak.Make(typed), true
	case *float64:
		return weak.Make(typed), true
	case *complex64:
		return weak.Make(typed), true
	case *complex128:
		return weak.Make(typed), true
	default:
		return nil, false
	}
}

// weakMakeCompositePointer handles pointers to the commonly-pointed-to composite types (a
// few slice and map shapes plus *any).
//
// Takes pointer (any) a non-nil pointer to a supported composite.
//
// Returns the weak.Pointer[T] wrapped as any and true when matched; nil and false
// otherwise.
func weakMakeCompositePointer(pointer any) (any, bool) {
	switch typed := pointer.(type) {
	case *[]byte:
		return weak.Make(typed), true
	case *[]string:
		return weak.Make(typed), true
	case *[]int:
		return weak.Make(typed), true
	case *map[string]any:
		return weak.Make(typed), true
	case *map[string]string:
		return weak.Make(typed), true
	case *any:
		return weak.Make(typed), true
	default:
		return nil, false
	}
}

func init() {
	if _, ok := Symbols["unique"]; ok {
		Symbols["unique"]["Make"] = reflect.ValueOf(wrappedUniqueMake)
	}
	if _, ok := Symbols["weak"]; ok {
		Symbols["weak"]["Make"] = reflect.ValueOf(wrappedWeakMake)
	}
}
