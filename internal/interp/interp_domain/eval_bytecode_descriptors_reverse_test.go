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
		name      string
		basicKind reflect.Kind
		expected  reflect.Type
	}{
		{"Bool", reflect.Bool, reflect.TypeFor[bool]()},
		{"Int", reflect.Int, reflect.TypeFor[int]()},
		{"Int8", reflect.Int8, reflect.TypeFor[int8]()},
		{"Int16", reflect.Int16, reflect.TypeFor[int16]()},
		{"Int32", reflect.Int32, reflect.TypeFor[int32]()},
		{"Int64", reflect.Int64, reflect.TypeFor[int64]()},
		{"Uint", reflect.Uint, reflect.TypeFor[uint]()},
		{"Uint8", reflect.Uint8, reflect.TypeFor[uint8]()},
		{"Uint16", reflect.Uint16, reflect.TypeFor[uint16]()},
		{"Uint32", reflect.Uint32, reflect.TypeFor[uint32]()},
		{"Uint64", reflect.Uint64, reflect.TypeFor[uint64]()},
		{"Uintptr", reflect.Uintptr, reflect.TypeFor[uintptr]()},
		{"Float32", reflect.Float32, reflect.TypeFor[float32]()},
		{"Float64", reflect.Float64, reflect.TypeFor[float64]()},
		{"Complex64", reflect.Complex64, reflect.TypeFor[complex64]()},
		{"Complex128", reflect.Complex128, reflect.TypeFor[complex128]()},
		{"String", reflect.String, reflect.TypeFor[string]()},
		{"UnsafePointer", reflect.UnsafePointer, reflect.TypeFor[unsafe.Pointer]()},
		{"UnknownKind", reflect.Kind(255), reflect.TypeFor[any]()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			descriptor := typeDescriptor{
				kind:      typeDescBasic,
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

	basicIntDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.Int)}
	basicStringDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.String)}
	basicBoolDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.Bool)}

	t.Run("Pointer", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    typeDescPtr,
			element: &basicIntDesc,
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[*int](), got)
	})

	t.Run("Slice", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    typeDescSlice,
			element: &basicIntDesc,
		}
		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[[]int](), got)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:    typeDescArray,
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
			kind:  typeDescMap,
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
			kind:    typeDescChan,
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
			kind:    typeDescChan,
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
			kind:    typeDescChan,
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
			kind:       typeDescFunc,
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
			kind:    typeDescSlice,
			element: &basicStringDesc,
		}
		descriptor := typeDescriptor{
			kind:       typeDescFunc,
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
			kind: typeDescStruct,
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
		descriptor := typeDescriptor{kind: typeDescInterface}
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
			kind:        typeDescNamed,
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
			kind:        typeDescNamed,
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
			kind:        typeDescNamed,
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

		basicIntDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.Int)}
		ptrDesc := typeDescriptor{kind: typeDescPtr, element: &basicIntDesc}
		descriptor := typeDescriptor{kind: typeDescSlice, element: &ptrDesc}

		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[[]*int](), got)
	})

	t.Run("MapOfSlices", func(t *testing.T) {
		t.Parallel()

		basicStringDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.String)}
		basicIntDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.Int)}
		sliceDesc := typeDescriptor{kind: typeDescSlice, element: &basicIntDesc}
		descriptor := typeDescriptor{kind: typeDescMap, key: &basicStringDesc, value: &sliceDesc}

		got, err := descriptorToReflectType(descriptor, registry)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[map[string][]int](), got)
	})
}

func TestDescriptorToReflectTypeErrorPropagation(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	badNamed := typeDescriptor{
		kind:        typeDescNamed,
		packagePath: "nonexistent",
		name:        "Bad",
	}

	t.Run("PointerWithBadElement", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: typeDescPtr, element: &badNamed}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nonexistent.Bad")
	})

	t.Run("SliceWithBadElement", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: typeDescSlice, element: &badNamed}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("ArrayWithBadElement", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: typeDescArray, element: &badNamed, length: 5}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("MapWithBadKey", func(t *testing.T) {
		t.Parallel()
		basicIntDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.Int)}
		descriptor := typeDescriptor{kind: typeDescMap, key: &badNamed, value: &basicIntDesc}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("MapWithBadValue", func(t *testing.T) {
		t.Parallel()
		basicIntDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.Int)}
		descriptor := typeDescriptor{kind: typeDescMap, key: &basicIntDesc, value: &badNamed}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("ChanWithBadElement", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{kind: typeDescChan, element: &badNamed, dir: int(reflect.BothDir)}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("FuncWithBadParam", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind:   typeDescFunc,
			params: []typeDescriptor{badNamed},
		}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("FuncWithBadResult", func(t *testing.T) {
		t.Parallel()
		basicIntDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.Int)}
		descriptor := typeDescriptor{
			kind:    typeDescFunc,
			params:  []typeDescriptor{basicIntDesc},
			results: []typeDescriptor{badNamed},
		}
		_, err := descriptorToReflectType(descriptor, registry)
		require.Error(t, err)
	})

	t.Run("StructWithBadFieldType", func(t *testing.T) {
		t.Parallel()
		descriptor := typeDescriptor{
			kind: typeDescStruct,
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

		basicIntDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.Int)}
		basicStringDesc := typeDescriptor{kind: typeDescBasic, basicKind: uint8(reflect.String)}

		structDesc := typeDescriptor{
			kind: typeDescStruct,
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
				kind:        typeDescNamed,
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
		Kind:      uint8(typeDescBasic),
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
		name   string
		goType reflect.Type
	}{
		{"Int", reflect.TypeFor[int]()},
		{"String", reflect.TypeFor[string]()},
		{"Bool", reflect.TypeFor[bool]()},
		{"PointerToInt", reflect.TypeFor[*int]()},
		{"SliceOfString", reflect.TypeFor[[]string]()},
		{"ArrayOfInt", reflect.TypeFor[[3]int]()},
		{"MapStringInt", reflect.TypeFor[map[string]int]()},
		{"ChanInt", reflect.TypeFor[chan int]()},
		{"FuncIntToString", reflect.FuncOf(
			[]reflect.Type{reflect.TypeFor[int]()},
			[]reflect.Type{reflect.TypeFor[string]()},
			false,
		)},
		{"EmptyStruct", reflect.TypeFor[struct{}]()},
		{"Float64", reflect.TypeFor[float64]()},
		{"Complex128", reflect.TypeFor[complex128]()},
		{"SliceOfSlice", reflect.TypeFor[[][]int]()},
		{"PointerToSlice", reflect.TypeFor[*[]string]()},
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
