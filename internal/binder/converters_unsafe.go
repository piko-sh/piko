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

package binder

import (
	"reflect"
	"strconv"
	"unsafe"
)

const (
	// float32BitSize is the bit size for parsing 32-bit floats.
	float32BitSize = 32

	// float64BitSize is the bit size used when parsing float64 values.
	float64BitSize = 64
)

var (
	// intBitSizes maps reflect.Kind to the bit size parameter for
	// strconv.ParseInt and strconv.ParseUint.
	intBitSizes = map[reflect.Kind]int{
		reflect.Int:    64,
		reflect.Int8:   8,
		reflect.Int16:  16,
		reflect.Int32:  32,
		reflect.Int64:  64,
		reflect.Uint:   64,
		reflect.Uint8:  8,
		reflect.Uint16: 16,
		reflect.Uint32: 32,
		reflect.Uint64: 64,
	}

	// signedIntSetters maps reflect.Kind to type-specific setter operations.
	// This eliminates duplicate switch statements in setDirectSignedInt.
	//
	//nolint:dupl // parallel setter maps
	signedIntSetters = map[reflect.Kind]intSetter[int64]{
		reflect.Int:   {setZero: func(p unsafe.Pointer) { *(*int)(p) = 0 }, setValue: func(p unsafe.Pointer, v int64) { *(*int)(p) = int(v) }},
		reflect.Int8:  {setZero: func(p unsafe.Pointer) { *(*int8)(p) = 0 }, setValue: func(p unsafe.Pointer, v int64) { *(*int8)(p) = int8(v) }},    // #nosec G115 -- bitSize=8 ensures no overflow
		reflect.Int16: {setZero: func(p unsafe.Pointer) { *(*int16)(p) = 0 }, setValue: func(p unsafe.Pointer, v int64) { *(*int16)(p) = int16(v) }}, // #nosec G115 -- bitSize=16 ensures no overflow
		reflect.Int32: {setZero: func(p unsafe.Pointer) { *(*int32)(p) = 0 }, setValue: func(p unsafe.Pointer, v int64) { *(*int32)(p) = int32(v) }}, // #nosec G115 -- bitSize=32 ensures no overflow
		reflect.Int64: {setZero: func(p unsafe.Pointer) { *(*int64)(p) = 0 }, setValue: func(p unsafe.Pointer, v int64) { *(*int64)(p) = v }},
	}

	// unsignedIntSetters maps reflect.Kind to type-specific setter operations.
	// This eliminates duplicate switch statements in setDirectUnsignedInt.
	//
	//nolint:dupl // parallel setter maps
	unsignedIntSetters = map[reflect.Kind]intSetter[uint64]{
		reflect.Uint: {
			setZero:  func(p unsafe.Pointer) { *(*uint)(p) = 0 },
			setValue: func(p unsafe.Pointer, v uint64) { *(*uint)(p) = uint(v) },
		},
		reflect.Uint8: {
			setZero:  func(p unsafe.Pointer) { *(*uint8)(p) = 0 },
			setValue: func(p unsafe.Pointer, v uint64) { *(*uint8)(p) = uint8(v) }, // #nosec G115 -- bitSize=8 ensures no overflow
		},
		reflect.Uint16: {
			setZero:  func(p unsafe.Pointer) { *(*uint16)(p) = 0 },
			setValue: func(p unsafe.Pointer, v uint64) { *(*uint16)(p) = uint16(v) }, // #nosec G115 -- bitSize=16 ensures no overflow
		},
		reflect.Uint32: {
			setZero:  func(p unsafe.Pointer) { *(*uint32)(p) = 0 },
			setValue: func(p unsafe.Pointer, v uint64) { *(*uint32)(p) = uint32(v) }, // #nosec G115 -- bitSize=32 ensures no overflow
		},
		reflect.Uint64: {
			setZero:  func(p unsafe.Pointer) { *(*uint64)(p) = 0 },
			setValue: func(p unsafe.Pointer, v uint64) { *(*uint64)(p) = v },
		},
	}
)

// intSetter holds type-specific operations for integer types. It uses generics
// to handle both signed and unsigned variants with a single definition.
type intSetter[T int64 | uint64] struct {
	// setZero sets the field at the given pointer to its zero value.
	setZero func(unsafe.Pointer)

	// setValue is the function that writes the parsed value to the target field.
	setValue func(unsafe.Pointer, T)
}

// convertAndSetDirect uses unsafe pointer arithmetic to set primitive fields
// directly, bypassing reflect.Value.Field() and reflect.Value.Set() for
// maximum performance.
//
// Takes structVal (reflect.Value) which is the addressable struct to modify.
// Takes value (string) which is the string value to parse and set.
// Takes fullPath (string) which is the field path for error messages.
// Takes fi (*fieldInfo) which provides cached field metadata and offset.
//
// Returns error when the value cannot be parsed for the target type.
//
// # Safety Analysis
//
// This function uses unsafe.Add and pointer casting to write directly to
// struct fields. This is SAFE because:
//
//  1. COMPILER-VERIFIED OFFSETS: fi.Offset comes from reflect.StructField.Offset,
//     which is computed by the Go compiler. The compiler guarantees this offset
//     is correct for the field's position within the struct's memory layout.
//
//  2. CONTROLLED ENTRY POINTS: This function is only called when fi.CanDirect
//     is true. CanDirect requires ALL of these conditions (see cache.go
//     processField):
//     - len(Index) == 1: Single-level access, no nested struct traversal
//     - Kind != Ptr: Not a pointer field (no nil checks or allocation needed)
//     - isPrimitiveKind(): Only basic types with known sizes
//     - !hasWellKnownConverter(): No special parsing needed (e.g., time.Duration)
//     - unmarshaler == nil: No TextUnmarshaler to call
//
//  3. TYPE-SAFE CASTING: The switch on fi.Kind ensures we cast to the correct
//     type. fi.Kind comes from reflect.Type.Kind(), which is set at compile
//     time. Mismatched types would be caught at cache-build time.
//
//  4. VALID POINTER SOURCE: structVal comes from a valid `reflect.Value` obtained
//     by traversing from the user-provided destination struct. The reflect
//     package guarantees this points to valid, addressable memory.
//
//  5. USER CONVERTER CHECK: We check for user-registered converters at runtime
//     before using the unsafe path, maintaining the converter precedence
//     contract.
//
//  6. VALIDATED BY TESTS: See TestConvertAndSetDirect_* in unsafe_safety_test.go
//     which verifies correctness for all primitive types, edge cases, and
//     concurrent access.
//
// # Why This Is Faster
//
// The reflection path creates multiple allocations per field:
//   - reflect.Value for the field (via Field())
//   - reflect.Value for the converted value (via ValueOf())
//   - Type checking in Set()
//
// This function eliminates all of that: parse the string, write directly to
// memory. Benchmarks show 57-73% reduction in allocations.
func (b *ASTBinder) convertAndSetDirect(structVal reflect.Value, value string, fullPath string, fi *fieldInfo) error {
	if b.hasConverters.Load() {
		if converter := b.getUserConverter(fi.Type); converter != nil {
			field := structVal.Field(fi.Index[0])
			return b.convertAndSet(field, value, fullPath, fi)
		}
	}

	structPtr := structVal.Addr().UnsafePointer()

	fieldPtr := unsafe.Add(structPtr, fi.Offset)

	switch fi.Kind {
	case reflect.String:
		*(*string)(fieldPtr) = value
		return nil
	case reflect.Bool:
		return setDirectBool(fieldPtr, value, fullPath, fi)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setDirectSignedInt(fieldPtr, value, fi.Kind, fullPath, fi)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setDirectUnsignedInt(fieldPtr, value, fi.Kind, fullPath, fi)
	case reflect.Float32, reflect.Float64:
		return setDirectFloat(fieldPtr, value, fi.Kind, fullPath, fi)
	default:
		return b.convertAndSet(structVal.Field(fi.Index[0]), value, fullPath, fi)
	}
}

// setDirectBool parses a string and sets a boolean field using an unsafe
// pointer.
//
// When the value is empty, the field is set to false. When the value is "on",
// the field is set to true. Other values are parsed using strconv.ParseBool.
//
// Takes fieldPtr (unsafe.Pointer) which points to the boolean field to set.
// Takes value (string) which is the string to parse as a boolean.
// Takes fullPath (string) which is the full path for error messages.
// Takes fi (*fieldInfo) which provides field details for error messages.
//
// Returns error when the value cannot be parsed as a boolean.
func setDirectBool(fieldPtr unsafe.Pointer, value string, fullPath string, fi *fieldInfo) error {
	if value == "" {
		*(*bool)(fieldPtr) = false
		return nil
	}
	if value == "on" {
		*(*bool)(fieldPtr) = true
		return nil
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return errSetField{path: fullPath, field: fi.Path, fieldType: fi.Type.String(), err: err}
	}
	*(*bool)(fieldPtr) = b
	return nil
}

// setDirectSignedInt parses and sets a signed integer field using unsafe
// pointer access. Uses a dispatch table to select the correct setter for the
// target type.
//
// When the value is empty, sets the field to zero and returns nil.
//
// Takes fieldPtr (unsafe.Pointer) which points to the target integer field.
// Takes value (string) which is the string to parse.
// Takes kind (reflect.Kind) which specifies the target integer type.
// Takes fullPath (string) which is the full path for error messages.
// Takes fi (*fieldInfo) which provides field details for error messages.
//
// Returns error when the value cannot be parsed as a signed integer or does
// not fit in the target type.
func setDirectSignedInt(fieldPtr unsafe.Pointer, value string, kind reflect.Kind, fullPath string, fi *fieldInfo) error {
	setter := signedIntSetters[kind]
	if value == "" {
		setter.setZero(fieldPtr)
		return nil
	}
	i, err := strconv.ParseInt(value, 10, intBitSizes[kind])
	if err != nil {
		return errSetField{path: fullPath, field: fi.Path, fieldType: fi.Type.String(), err: err}
	}
	setter.setValue(fieldPtr, i)
	return nil
}

// setDirectUnsignedInt parses and sets an unsigned integer field using an
// unsafe pointer. Uses a dispatch table to set the correct integer size.
//
// When value is empty, sets the field to zero and returns nil.
//
// Takes fieldPtr (unsafe.Pointer) which points to the field to set.
// Takes value (string) which contains the value to parse.
// Takes kind (reflect.Kind) which specifies the unsigned integer type.
// Takes fullPath (string) which identifies the field location for errors.
// Takes fi (*fieldInfo) which provides field metadata.
//
// Returns error when parsing fails or the value is too large for the type.
func setDirectUnsignedInt(fieldPtr unsafe.Pointer, value string, kind reflect.Kind, fullPath string, fi *fieldInfo) error {
	setter := unsignedIntSetters[kind]
	if value == "" {
		setter.setZero(fieldPtr)
		return nil
	}
	u, err := strconv.ParseUint(value, 10, intBitSizes[kind])
	if err != nil {
		return errSetField{path: fullPath, field: fi.Path, fieldType: fi.Type.String(), err: err}
	}
	setter.setValue(fieldPtr, u)
	return nil
}

// setDirectFloat parses a string and sets a float field using an unsafe
// pointer.
//
// Takes fieldPtr (unsafe.Pointer) which points to the float field to set.
// Takes value (string) which holds the float value to parse.
// Takes kind (reflect.Kind) which shows whether the field is float32 or
// float64.
// Takes fullPath (string) which gives the field path for error messages.
// Takes fi (*fieldInfo) which provides field details for error reporting.
//
// Returns error when the value cannot be parsed as a valid float.
func setDirectFloat(fieldPtr unsafe.Pointer, value string, kind reflect.Kind, fullPath string, fi *fieldInfo) error {
	if value == "" {
		if kind == reflect.Float32 {
			*(*float32)(fieldPtr) = 0
		} else {
			*(*float64)(fieldPtr) = 0
		}
		return nil
	}
	bitSize := float32BitSize
	if kind == reflect.Float64 {
		bitSize = float64BitSize
	}
	f, err := strconv.ParseFloat(value, bitSize)
	if err != nil {
		return errSetField{path: fullPath, field: fi.Path, fieldType: fi.Type.String(), err: err}
	}
	if kind == reflect.Float32 {
		*(*float32)(fieldPtr) = float32(f)
	} else {
		*(*float64)(fieldPtr) = f
	}
	return nil
}
