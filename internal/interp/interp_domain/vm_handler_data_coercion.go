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

	"piko.sh/piko/wdk/safeconv"
)

// writeIntegerFieldFast attempts the integer-widening fast path for handleSetField.
//
// When both source and field have integer kinds (signed or unsigned) and a
// width-compatible reinterpret is possible, the source value is written directly into the
// field's backing storage via unsafe, saving the reflect.Value.Convert allocation that
// coerceValue would otherwise pay (~1 alloc per call, dominant in polyast's
// struct-literal initialisation profile). When either kind is non-integer, the field is
// non-addressable, or the kind pair would require a non-integer conversion, the caller
// must fall through to the slow coerceValue + field.Set path.
//
// Sign extension: reflect.Value.Int() returns the source value sign-extended to int64; we
// then mask to the target width via the destination store (e.g. *(*int8)(addr) =
// int8(value) takes the low 8 bits, preserving the sign because int64 to int8 conversion
// in Go is defined as truncation + reinterpretation). Bit width: reflect.Value.Uint()
// returns the source value zero-extended to uint64; destination stores via
// *(*uintN)(addr) take the low N bits.
//
// Takes field (reflect.Value) which is the addressable destination field.
// Takes source (reflect.Value) which is the integer source value.
//
// Returns true when the field was written via the fast path.
// Returns false when the caller must fall back to the slow coerceValue path.
func writeIntegerFieldFast(field reflect.Value, source reflect.Value) bool {
	sourceKind := source.Kind()
	if !isIntegerLikeKind(sourceKind) {
		return false
	}
	addr := unsafe.Pointer(field.UnsafeAddr())
	switch field.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Uint:
		return writeUnsignedIntegerField(addr, field.Kind(), source, sourceKind)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return writeSignedIntegerField(addr, field.Kind(), source, sourceKind)
	default:
	}
	return false
}

// isIntegerLikeKind reports whether the kind is any signed or unsigned integer width
// recognised by writeIntegerFieldFast.
//
// Takes kind (reflect.Kind) which is the kind to test.
//
// Returns true when the kind is one of the integer widths.
func isIntegerLikeKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
	}
	return false
}

// writeUnsignedIntegerField narrows source to destinationKind and writes it at addr using
// Go's modular conversion semantics for typed VM writes.
//
// Takes addr (unsafe.Pointer) which is the field storage address.
// Takes destinationKind (reflect.Kind) which is the destination integer width.
// Takes source (reflect.Value) which is the source integer value.
// Takes sourceKind (reflect.Kind) which is the source integer width.
//
// Returns true once the value has been written.
func writeUnsignedIntegerField(addr unsafe.Pointer, destinationKind reflect.Kind, source reflect.Value, sourceKind reflect.Kind) bool {
	value := integerSourceAsUint64(source, sourceKind)
	switch destinationKind {
	case reflect.Uint8:
		*(*uint8)(addr) = uint8(value) //nolint:gosec // narrow + reinterpret matches reflect.Convert
	case reflect.Uint16:
		*(*uint16)(addr) = uint16(value) //nolint:gosec // narrow + reinterpret matches reflect.Convert
	case reflect.Uint32:
		*(*uint32)(addr) = uint32(value) //nolint:gosec // narrow + reinterpret matches reflect.Convert
	case reflect.Uint64, reflect.Uintptr, reflect.Uint:
		*(*uint64)(addr) = value
	default:
		return false
	}
	return true
}

// writeSignedIntegerField narrows source to destinationKind and writes it at addr using
// Go's modular conversion semantics for typed VM writes.
//
// Takes addr (unsafe.Pointer) which is the field storage address.
// Takes destinationKind (reflect.Kind) which is the destination integer width.
// Takes source (reflect.Value) which is the source integer value.
// Takes sourceKind (reflect.Kind) which is the source integer width.
//
// Returns true once the value has been written.
func writeSignedIntegerField(addr unsafe.Pointer, destinationKind reflect.Kind, source reflect.Value, sourceKind reflect.Kind) bool {
	value := integerSourceAsInt64(source, sourceKind)
	switch destinationKind {
	case reflect.Int8:
		*(*int8)(addr) = int8(value) //nolint:gosec // narrow + reinterpret matches reflect.Convert
	case reflect.Int16:
		*(*int16)(addr) = int16(value) //nolint:gosec // narrow + reinterpret matches reflect.Convert
	case reflect.Int32:
		*(*int32)(addr) = int32(value) //nolint:gosec // narrow + reinterpret matches reflect.Convert
	case reflect.Int64, reflect.Int:
		*(*int64)(addr) = value
	default:
		return false
	}
	return true
}

// integerSourceAsUint64 widens an integer-typed reflect.Value to uint64, reinterpreting
// signed bits where necessary.
//
// Takes source (reflect.Value) which is the source value.
// Takes sourceKind (reflect.Kind) which is the source integer width.
//
// Returns the widened uint64.
func integerSourceAsUint64(source reflect.Value, sourceKind reflect.Kind) uint64 {
	switch sourceKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return safeconv.Int64ToUint64Reinterpret(source.Int())
	default:
	}
	return source.Uint()
}

// integerSourceAsInt64 widens an integer-typed reflect.Value to int64, reinterpreting
// unsigned bits where necessary.
//
// Takes source (reflect.Value) which is the source value.
// Takes sourceKind (reflect.Kind) which is the source integer width.
//
// Returns the widened int64.
func integerSourceAsInt64(source reflect.Value, sourceKind reflect.Kind) int64 {
	switch sourceKind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return safeconv.Uint64ToInt64Reinterpret(source.Uint())
	default:
	}
	return source.Int()
}

// coerceValue converts value to match targetType, handling closure-to-func coercion and
// standard reflect conversions as needed.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes value (reflect.Value) which is the value to convert.
// Takes targetType (reflect.Type) which is the desired target type.
//
// Returns reflect.Value matching targetType after coercion.
//
// Skips reflect.Value.Convert when the value is already assignable to the target. The
// common case is a concrete T being stored into an interface{} field or element;
// reflect.Value.Set performs the T to interface boxing internally without the separate
// Convert allocation. In LRU this drops the Convert path on hot field stores into the
// Node{value any} field.
func coerceValue(vm *VM, value reflect.Value, targetType reflect.Type) reflect.Value {
	if !value.IsValid() {
		return reflect.Zero(targetType)
	}
	if value.Type() == targetType {
		return value
	}
	if value.Type().AssignableTo(targetType) {
		return value
	}
	if targetType.Kind() == reflect.Func {
		value = coerceClosureToFunc(vm, value, targetType)
	}
	if value.Type() == targetType {
		return value
	}
	return convertOrCoerceContainer(value, targetType)
}

// convertOrCoerceContainer applies the standard reflect.Convert when the conversion is
// permitted, falling back to the slice/array/map element-coercion paths used by
// cycle-broken `any`-element containers.
//
// Takes value (reflect.Value) which is the source value (already known to differ from
// targetType).
// Takes targetType (reflect.Type) which is the desired target type.
//
// Returns the converted reflect.Value (or the original value when no coercion applies).
func convertOrCoerceContainer(value reflect.Value, targetType reflect.Type) reflect.Value {
	if targetType.Kind() == reflect.Bool && value.CanInt() {
		return reflect.ValueOf(value.Int() != 0)
	}
	if value.Type().ConvertibleTo(targetType) {
		return value.Convert(targetType)
	}
	if isSliceElementCoercion(value.Type(), targetType) {
		return coerceSliceElements(value, targetType)
	}
	if isArrayElementCoercion(value.Type(), targetType) {
		return coerceArrayElements(value, targetType)
	}
	if isMapElementCoercion(value.Type(), targetType) {
		return coerceMapElements(value, targetType)
	}
	return value
}

// isArrayElementCoercion / coerceArrayElements / isMapElementCoercion / coerceMapElements
// parallel the slice variants. They cover the case where a struct's field type was
// cycle-broken from [N]*RecursiveType / map[K]*RecursiveType to [N]any / map[K]any (see
// convertFieldBreakingCycles): a write of the original typed container literal can't
// assign to the cycle-broken `any`-element container without per-element materialisation.

// isArrayElementCoercion reports whether from and to are both arrays of the same length,
// with from's element type assignable to to's element type.
//
// Takes from (reflect.Type), to (reflect.Type).
//
// Returns true on a valid element-coercion shape.
func isArrayElementCoercion(from, to reflect.Type) bool {
	if from.Kind() != reflect.Array || to.Kind() != reflect.Array {
		return false
	}
	if from.Len() != to.Len() {
		return false
	}
	return from.Elem().AssignableTo(to.Elem())
}

// coerceArrayElements materialises a fresh array of targetType with element-by-element
// assignment from value.
//
// Takes value (reflect.Value) which is the source array.
// Takes targetType (reflect.Type) which is the desired array type.
//
// Returns reflect.Value of targetType with elements copied.
func coerceArrayElements(value reflect.Value, targetType reflect.Type) reflect.Value {
	out := reflect.New(targetType).Elem()
	length := value.Len()
	for i := range length {
		out.Index(i).Set(value.Index(i))
	}
	return out
}

// isMapElementCoercion reports whether from and to are both maps with identical keys and
// from's element type assignable to to's element type.
//
// Takes from (reflect.Type), to (reflect.Type).
//
// Returns true on a valid element-coercion shape.
func isMapElementCoercion(from, to reflect.Type) bool {
	if from.Kind() != reflect.Map || to.Kind() != reflect.Map {
		return false
	}
	if from.Key() != to.Key() {
		return false
	}
	return from.Elem().AssignableTo(to.Elem())
}

// coerceMapElements materialises a fresh map of targetType with every key/value pair from
// value re-inserted under the target type.
//
// Takes value (reflect.Value) which is the source map.
// Takes targetType (reflect.Type) which is the desired map type.
//
// Returns reflect.Value of targetType with entries copied.
func coerceMapElements(value reflect.Value, targetType reflect.Type) reflect.Value {
	out := reflect.MakeMapWithSize(targetType, value.Len())
	iter := value.MapRange()
	for iter.Next() {
		out.SetMapIndex(iter.Key(), iter.Value())
	}
	return out
}

// isSliceElementCoercion reports an element-assignable slice conversion.
//
// True when both types are slices and the element type of `from` is assignable to the
// element type of `to`. The runtime needs this when a generic body with a `[]T` field is
// instantiated from a concrete `[]int` (or similar) since reflect erases T to interface{}
// in the synthesised struct.
//
// Takes from (reflect.Type) which is the source value's type.
// Takes to (reflect.Type) which is the target type.
//
// Returns true when both types are slices and `from`'s element type is assignable to
// `to`'s element type.
func isSliceElementCoercion(from, to reflect.Type) bool {
	if from.Kind() != reflect.Slice || to.Kind() != reflect.Slice {
		return false
	}
	return from.Elem().AssignableTo(to.Elem())
}

// coerceSliceElements builds a fresh targetType slice copied from value.
//
// Relies on AssignableTo at the element level. Used by coerceValue when the source slice
// element type is assignable to the target slice element type but the slice types
// themselves are not interconvertible (e.g. []int into []any).
//
// Takes value (reflect.Value) which is the source slice.
// Takes targetType (reflect.Type) which is the desired slice type.
//
// Returns reflect.Value of targetType with element-by-element values copied from `value`.
func coerceSliceElements(value reflect.Value, targetType reflect.Type) reflect.Value {
	length := value.Len()
	out := reflect.MakeSlice(targetType, length, length)
	for i := range length {
		out.Index(i).Set(value.Index(i))
	}
	return out
}
