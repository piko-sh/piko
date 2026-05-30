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
	"math"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestDescriptorToReflectTypeBasicKinds(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	tests := []struct {
		expected  reflect.Type
		name      string
		basicKind reflect.Kind
	}{
		{name: "Bool", basicKind: reflect.Bool, expected: reflect.TypeFor[bool]()},
		{name: "Int", basicKind: reflect.Int, expected: reflect.TypeFor[int]()},
		{name: "Int8", basicKind: reflect.Int8, expected: reflect.TypeFor[int8]()},
		{name: "Int16", basicKind: reflect.Int16, expected: reflect.TypeFor[int16]()},
		{name: "Int32", basicKind: reflect.Int32, expected: reflect.TypeFor[int32]()},
		{name: "Int64", basicKind: reflect.Int64, expected: reflect.TypeFor[int64]()},
		{name: "Uint", basicKind: reflect.Uint, expected: reflect.TypeFor[uint]()},
		{name: "Uint8", basicKind: reflect.Uint8, expected: reflect.TypeFor[uint8]()},
		{name: "Uint16", basicKind: reflect.Uint16, expected: reflect.TypeFor[uint16]()},
		{name: "Uint32", basicKind: reflect.Uint32, expected: reflect.TypeFor[uint32]()},
		{name: "Uint64", basicKind: reflect.Uint64, expected: reflect.TypeFor[uint64]()},
		{name: "Uintptr", basicKind: reflect.Uintptr, expected: reflect.TypeFor[uintptr]()},
		{name: "Float32", basicKind: reflect.Float32, expected: reflect.TypeFor[float32]()},
		{name: "Float64", basicKind: reflect.Float64, expected: reflect.TypeFor[float64]()},
		{name: "Complex64", basicKind: reflect.Complex64, expected: reflect.TypeFor[complex64]()},
		{name: "Complex128", basicKind: reflect.Complex128, expected: reflect.TypeFor[complex128]()},
		{name: "String", basicKind: reflect.String, expected: reflect.TypeFor[string]()},
		{name: "UnsafePointer", basicKind: reflect.UnsafePointer, expected: reflect.TypeFor[unsafe.Pointer]()},
		{name: "UnknownKind", basicKind: reflect.Kind(255), expected: reflect.TypeFor[any]()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			descriptor := typeDescriptor{
				kind:      kindBasic,
				basicKind: uint8(tt.basicKind),
			}

			got, err := descriptorToReflectType(descriptor, registry)
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestDescriptorToReflectTypeComposite(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	basicIntDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.Int)}
	basicStringDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.String)}
	basicBoolDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.Bool)}

	t.Run("Pointer", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    kindPtr,
			element: &basicIntDesc,
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[*int](), got)
	})

	t.Run("Slice", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    kindSlice,
			element: &basicIntDesc,
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[[]int](), got)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    kindArray,
			element: &basicStringDesc,
			length:  3,
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[[3]string](), got)
	})

	t.Run("Map", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:  kindMap,
			key:   &basicStringDesc,
			value: &basicIntDesc,
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[map[string]int](), got)
	})

	t.Run("ChanBidirectional", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    kindChan,
			element: &basicIntDesc,
			dir:     int(reflect.BothDir),
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[chan int](), got)
	})

	t.Run("ChanSendOnly", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    kindChan,
			element: &basicIntDesc,
			dir:     int(reflect.SendDir),
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		expected := reflect.ChanOf(reflect.SendDir, reflect.TypeFor[int]())
		require.Equal(t, expected, got)
	})

	t.Run("ChanRecvOnly", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    kindChan,
			element: &basicIntDesc,
			dir:     int(reflect.RecvDir),
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		expected := reflect.ChanOf(reflect.RecvDir, reflect.TypeFor[int]())
		require.Equal(t, expected, got)
	})

	t.Run("FuncNonVariadic", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:       kindFunc,
			params:     []typeDescriptor{basicIntDesc, basicStringDesc},
			results:    []typeDescriptor{basicBoolDesc},
			isVariadic: false,
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		expected := reflect.FuncOf(
			[]reflect.Type{reflect.TypeFor[int](), reflect.TypeFor[string]()},
			[]reflect.Type{reflect.TypeFor[bool]()},
			false,
		)
		require.Equal(t, expected, got)
	})

	t.Run("FuncVariadic", func(t *testing.T) {
		t.Parallel()
		sliceOfStringDesc := typeDescriptor{
			kind:    kindSlice,
			element: &basicStringDesc,
		}
		descriptor := typeDescriptor{
			kind:       kindFunc,
			params:     []typeDescriptor{basicIntDesc, sliceOfStringDesc},
			results:    []typeDescriptor{basicBoolDesc},
			isVariadic: true,
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		expected := reflect.FuncOf(
			[]reflect.Type{reflect.TypeFor[int](), reflect.TypeFor[[]string]()},
			[]reflect.Type{reflect.TypeFor[bool]()},
			true,
		)
		require.Equal(t, expected, got)
	})

	t.Run("Struct", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind: kindStruct,
			fields: []typeDescField{
				{name: "X", typ: basicIntDesc},
				{name: "Y", tag: `json:"y"`, typ: basicStringDesc},
			},
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		expected := reflect.StructOf([]reflect.StructField{
			{Name: "X", Type: reflect.TypeFor[int]()},
			{Name: "Y", Type: reflect.TypeFor[string](), Tag: `json:"y"`},
		})
		require.Equal(t, expected, got)
	})

	t.Run("InterfaceEmpty", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: kindInterface}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[any](), got)
	})

	t.Run("UnknownKindFallback", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: typeDescKind(255)}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[any](), got)
	})
}

func TestDescriptorToReflectTypeNamed(t *testing.T) {
	t.Parallel()

	t.Run("RegisteredNamedType", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(SymbolExports{
			"time": {
				"Time": reflect.ValueOf((*time.Time)(nil)),
			},
		})

		descriptor := typeDescriptor{
			kind:        kindNamed,
			packagePath: "time",
			name:        "Time",
		}

		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[time.Time](), got)
	})

	t.Run("UnregisteredNamedType", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(nil)

		descriptor := typeDescriptor{
			kind:        kindNamed,
			packagePath: "nonexistent",
			name:        "Missing",
		}

		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nonexistent.Missing")
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("NamedTypeNonPointerValue", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(SymbolExports{
			"math": {
				"Pi": reflect.ValueOf(math.Pi),
			},
		})

		descriptor := typeDescriptor{
			kind:        kindNamed,
			packagePath: "math",
			name:        "Pi",
		}

		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[float64](), got)
	})
}

func TestDescriptorToReflectTypeNestedComposites(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	t.Run("SliceOfPointers", func(t *testing.T) {
		t.Parallel()

		basicIntDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.Int)}
		ptrDesc := typeDescriptor{kind: kindPtr, element: &basicIntDesc}
		descriptor := typeDescriptor{kind: kindSlice, element: &ptrDesc}

		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[[]*int](), got)
	})

	t.Run("MapOfSlices", func(t *testing.T) {
		t.Parallel()

		basicStringDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.String)}
		basicIntDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.Int)}
		sliceDesc := typeDescriptor{kind: kindSlice, element: &basicIntDesc}
		descriptor := typeDescriptor{kind: kindMap, key: &basicStringDesc, value: &sliceDesc}

		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[map[string][]int](), got)
	})
}

func TestDescriptorToReflectTypeErrorPropagation(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	badNamed := typeDescriptor{
		kind:        kindNamed,
		packagePath: "nonexistent",
		name:        "Bad",
	}

	t.Run("PointerWithBadElement", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: kindPtr, element: &badNamed}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nonexistent.Bad")
	})

	t.Run("SliceWithBadElement", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: kindSlice, element: &badNamed}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("ArrayWithBadElement", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: kindArray, element: &badNamed, length: 5}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("MapWithBadKey", func(t *testing.T) {
		t.Parallel()
		basicIntDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.Int)}
		descriptor := typeDescriptor{kind: kindMap, key: &badNamed, value: &basicIntDesc}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("MapWithBadValue", func(t *testing.T) {
		t.Parallel()
		basicIntDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.Int)}
		descriptor := typeDescriptor{kind: kindMap, key: &basicIntDesc, value: &badNamed}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("ChanWithBadElement", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: kindChan, element: &badNamed, dir: int(reflect.BothDir)}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("FuncWithBadParam", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:   kindFunc,
			params: []typeDescriptor{badNamed},
		}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("FuncWithBadResult", func(t *testing.T) {
		t.Parallel()
		basicIntDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.Int)}
		descriptor := typeDescriptor{
			kind:    kindFunc,
			params:  []typeDescriptor{basicIntDesc},
			results: []typeDescriptor{badNamed},
		}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("StructWithBadFieldType", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind: kindStruct,
			fields: []typeDescField{
				{name: "Bad", typ: badNamed},
			},
		}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})
}

func TestReconstructGeneralConstant(t *testing.T) {
	t.Parallel()

	t.Run("PackageSymbolFound", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(SymbolExports{
			"math": {
				"Pi": reflect.ValueOf(math.Pi),
			},
		})

		descriptor := generalConstantDescriptor{
			kind:        generalConstantPackageSymbol,
			packagePath: "math",
			symbolName:  "Pi",
		}

		got, err := reconstructGeneralConstant(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, math.Pi, got.Float())
	})

	t.Run("PackageSymbolNotFound", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(nil)

		descriptor := generalConstantDescriptor{
			kind:        generalConstantPackageSymbol,
			packagePath: "nonexistent",
			symbolName:  "Foo",
		}

		_, err := reconstructGeneralConstant(descriptor, registry)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nonexistent.Foo")
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("NamedTypeZeroFound", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(SymbolExports{
			"time": {
				"Duration": reflect.ValueOf((*time.Duration)(nil)),
			},
		})

		descriptor := generalConstantDescriptor{
			kind:        generalConstantNamedTypeZero,
			packagePath: "time",
			symbolName:  "Duration",
		}

		got, err := reconstructGeneralConstant(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[time.Duration](), got.Type())
		require.True(t, got.IsZero())
	})

	t.Run("NamedTypeZeroNotFound", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(nil)

		descriptor := generalConstantDescriptor{
			kind:        generalConstantNamedTypeZero,
			packagePath: "nonexistent",
			symbolName:  "Missing",
		}

		_, err := reconstructGeneralConstant(descriptor, registry)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nonexistent.Missing")
	})

	t.Run("CompositeZero", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(nil)

		basicIntDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.Int)}
		basicStringDesc := typeDescriptor{kind: kindBasic, basicKind: uint8(reflect.String)}

		structDesc := typeDescriptor{
			kind: kindStruct,
			fields: []typeDescField{
				{name: "X", typ: basicIntDesc},
				{name: "Y", typ: basicStringDesc},
			},
		}

		descriptor := generalConstantDescriptor{
			kind:     generalConstantCompositeZero,
			typeDesc: structDesc,
		}

		got, err := reconstructGeneralConstant(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.Struct, got.Kind())
		require.Equal(t, 2, got.NumField())
		require.True(t, got.IsZero())
	})

	t.Run("CompositeZeroWithBadType", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(nil)

		descriptor := generalConstantDescriptor{
			kind: generalConstantCompositeZero,
			typeDesc: typeDescriptor{
				kind:        kindNamed,
				packagePath: "nonexistent",
				name:        "Bad",
			},
		}

		_, err := reconstructGeneralConstant(descriptor, registry)
		require.Error(t, err)
		require.Contains(t, err.Error(), "reconstructing composite type")
	})

	t.Run("UnknownKind", func(t *testing.T) {
		t.Parallel()

		registry := NewSymbolRegistry(nil)

		descriptor := generalConstantDescriptor{
			kind: generalConstantKind(255),
		}

		_, err := reconstructGeneralConstant(descriptor, registry)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown general constant kind")
	})
}

func TestReconstructGeneralConstantExportedAPI(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(SymbolExports{
		"math": {
			"Pi": reflect.ValueOf(math.Pi),
		},
	})

	data := GeneralConstantDescriptorData{
		PackagePath: "math",
		SymbolName:  "Pi",
		Kind:        uint8(generalConstantPackageSymbol),
	}

	got, err := ReconstructGeneralConstant(data, registry)
	require.NoError(t, err)
	require.Equal(t, math.Pi, got.Float())
}

func TestDescriptorToReflectTypeExportedAPI(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	data := TypeDescriptorData{
		Kind:      uint8(kindBasic),
		BasicKind: uint8(reflect.Int),
	}

	got, err := DescriptorToReflectType(data, registry)
	require.NoError(t, err)
	require.Equal(t, reflect.TypeFor[int](), got)
}

func TestDescriptorRoundTrip(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	tests := []struct {
		goType reflect.Type
		name   string
	}{
		{name: "Int", goType: reflect.TypeFor[int]()},
		{name: "String", goType: reflect.TypeFor[string]()},
		{name: "Bool", goType: reflect.TypeFor[bool]()},
		{name: "PointerToInt", goType: reflect.TypeFor[*int]()},
		{name: "SliceOfString", goType: reflect.TypeFor[[]string]()},
		{name: "ArrayOfInt", goType: reflect.TypeFor[[3]int]()},
		{name: "MapStringInt", goType: reflect.TypeFor[map[string]int]()},
		{name: "ChanInt", goType: reflect.TypeFor[chan int]()},
		{name: "FuncIntToString", goType: reflect.FuncOf(
			[]reflect.Type{reflect.TypeFor[int]()},
			[]reflect.Type{reflect.TypeFor[string]()},
			false,
		)},
		{name: "EmptyStruct", goType: reflect.TypeFor[struct{}]()},
		{name: "Float64", goType: reflect.TypeFor[float64]()},
		{name: "Complex128", goType: reflect.TypeFor[complex128]()},
		{name: "SliceOfSlice", goType: reflect.TypeFor[[][]int]()},
		{name: "PointerToSlice", goType: reflect.TypeFor[*[]string]()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			descriptor := reflectTypeToDescriptor(tt.goType)

			reconstructed, err := descriptorToReflectType(descriptor, registry)
			require.NoError(t, err)
			require.Equal(t, tt.goType, reconstructed,
				"round-trip failed for %s: expected %v, got %v", tt.name, tt.goType, reconstructed)
		})
	}
}

func TestDescriptorRoundTripStruct(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	original := reflect.StructOf([]reflect.StructField{
		{Name: "Name", Type: reflect.TypeFor[string](), Tag: `json:"name"`},
		{Name: "Age", Type: reflect.TypeFor[int]()},
		{Name: "Active", Type: reflect.TypeFor[bool](), Tag: `db:"active"`},
	})

	descriptor := reflectTypeToDescriptor(original)
	reconstructed, err := descriptorToReflectType(descriptor, registry)
	require.NoError(t, err)
	require.Equal(t, original.NumField(), reconstructed.NumField())

	for i := range original.NumField() {
		originalField := original.Field(i)
		reconstructedField := reconstructed.Field(i)
		require.Equal(t, originalField.Name, reconstructedField.Name)
		require.Equal(t, originalField.Type, reconstructedField.Type)
		require.Equal(t, originalField.Tag, reconstructedField.Tag)
	}
}

func TestDescriptorRoundTripFunc(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	t.Run("NonVariadic", func(t *testing.T) {
		t.Parallel()
		original := reflect.FuncOf(
			[]reflect.Type{reflect.TypeFor[int](), reflect.TypeFor[string]()},
			[]reflect.Type{reflect.TypeFor[bool]()},
			false,
		)
		descriptor := reflectTypeToDescriptor(original)
		reconstructed, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, original.Kind(), reconstructed.Kind())
		require.Equal(t, original.NumIn(), reconstructed.NumIn())
		require.Equal(t, original.NumOut(), reconstructed.NumOut())
		require.False(t, reconstructed.IsVariadic())
		for i := range original.NumIn() {
			require.Equal(t, original.In(i), reconstructed.In(i))
		}
		for i := range original.NumOut() {
			require.Equal(t, original.Out(i), reconstructed.Out(i))
		}
	})

	t.Run("Variadic", func(t *testing.T) {
		t.Parallel()
		original := reflect.FuncOf(
			[]reflect.Type{reflect.TypeFor[string](), reflect.TypeFor[[]any]()},
			[]reflect.Type{reflect.TypeFor[int]()},
			true,
		)
		descriptor := reflectTypeToDescriptor(original)
		reconstructed, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, original.Kind(), reconstructed.Kind())
		require.Equal(t, original.NumIn(), reconstructed.NumIn())
		require.Equal(t, original.NumOut(), reconstructed.NumOut())
		require.True(t, reconstructed.IsVariadic())
	})

	t.Run("NoParamsNoResults", func(t *testing.T) {
		t.Parallel()
		original := reflect.FuncOf(nil, nil, false)
		descriptor := reflectTypeToDescriptor(original)
		reconstructed, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.Func, reconstructed.Kind())
		require.Equal(t, 0, reconstructed.NumIn())
		require.Equal(t, 0, reconstructed.NumOut())
	})
}

func TestBasicKindToReflectDirect(t *testing.T) {
	t.Parallel()

	require.Equal(t, reflect.TypeFor[bool](), basicKindToReflect(reflect.Bool))
	require.Equal(t, reflect.TypeFor[int](), basicKindToReflect(reflect.Int))
	require.Equal(t, reflect.TypeFor[string](), basicKindToReflect(reflect.String))
	require.Equal(t, reflect.TypeFor[unsafe.Pointer](), basicKindToReflect(reflect.UnsafePointer))

	require.Equal(t, reflect.TypeFor[any](), basicKindToReflect(reflect.Kind(200)))
}

func TestImportExportTypeDescriptorRoundTrip(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	original := reflect.FuncOf(
		[]reflect.Type{
			reflect.TypeFor[map[string][]int](),
			reflect.TypeFor[*bool](),
		},
		[]reflect.Type{
			reflect.ChanOf(reflect.BothDir, reflect.TypeFor[float64]()),
		},
		false,
	)

	internalDesc := reflectTypeToDescriptor(original)

	exported := exportTypeDescriptor(internalDesc)

	imported := ImportTypeDescriptor(exported)

	reconstructed, err := descriptorToReflectType(imported, registry)
	require.NoError(t, err)

	require.Equal(t, original.Kind(), reconstructed.Kind())
	require.Equal(t, original.NumIn(), reconstructed.NumIn())
	require.Equal(t, original.NumOut(), reconstructed.NumOut())
	for i := range original.NumIn() {
		require.Equal(t, original.In(i), reconstructed.In(i))
	}
	for i := range original.NumOut() {
		require.Equal(t, original.Out(i), reconstructed.Out(i))
	}
}
