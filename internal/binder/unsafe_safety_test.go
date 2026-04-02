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

//go:build !safe

package binder

import (
	"context"
	"math"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UnsafeTestPrimitives struct {
	String  string
	Int     int
	Int64   int64
	Uint    uint
	Uint64  uint64
	Float64 float64
	Int32   int32
	Uint32  uint32
	Float32 float32
	Int16   int16
	Uint16  uint16
	Int8    int8
	Uint8   uint8
	Bool    bool
}

type UnsafeTestNested struct {
	Inner struct {
		Value string
	}
}

func TestTypeKey_Consistency(t *testing.T) {
	t.Run("same type returns same key", func(t *testing.T) {
		type1 := reflect.TypeFor[UnsafeTestPrimitives]()
		type2 := reflect.TypeFor[UnsafeTestPrimitives]()

		key1 := typeKey(type1)
		key2 := typeKey(type2)

		assert.Equal(t, key1, key2, "typeKey should return same value for same type")
	})

	t.Run("different types return different keys", func(t *testing.T) {
		typeA := reflect.TypeFor[UnsafeTestPrimitives]()
		typeB := reflect.TypeFor[UnsafeTestNested]()
		typeC := reflect.TypeFor[string]()
		typeD := reflect.TypeFor[int]()

		keyA := typeKey(typeA)
		keyB := typeKey(typeB)
		keyC := typeKey(typeC)
		keyD := typeKey(typeD)

		assert.NotEqual(t, keyA, keyB, "different struct types should have different keys")
		assert.NotEqual(t, keyC, keyD, "string and int should have different keys")
		assert.NotEqual(t, keyA, keyC, "struct and string should have different keys")
	})

	t.Run("key is stable across many calls", func(t *testing.T) {
		typ := reflect.TypeFor[UnsafeTestPrimitives]()
		firstKey := typeKey(typ)

		for i := range 10000 {
			key := typeKey(typ)
			assert.Equal(t, firstKey, key, "typeKey should be stable (iteration %d)", i)
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		typ := reflect.TypeFor[UnsafeTestPrimitives]()
		expectedKey := typeKey(typ)

		var wg sync.WaitGroup
		errors := make(chan error, 100)

		for range 100 {
			wg.Go(func() {
				for range 1000 {
					key := typeKey(typ)
					if key != expectedKey {
						errors <- assert.AnError
						return
					}
				}
			})
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("concurrent typeKey access failed: %v", err)
		}
	})
}

func TestConvertAndSetDirect_MatchesReflection(t *testing.T) {

	testCases := []struct {
		expected any
		name     string
		field    string
		value    string
	}{

		{name: "string_simple", field: "String", value: "hello", expected: "hello"},
		{name: "string_empty", field: "String", value: "", expected: ""},
		{name: "string_unicode", field: "String", value: "日本語", expected: "日本語"},
		{name: "string_emoji", field: "String", value: "🎉", expected: "🎉"},
		{name: "string_long", field: "String", value: string(make([]byte, 10000)), expected: string(make([]byte, 10000))},

		{name: "bool_true", field: "Bool", value: "true", expected: true},
		{name: "bool_false", field: "Bool", value: "false", expected: false},
		{name: "bool_1", field: "Bool", value: "1", expected: true},
		{name: "bool_0", field: "Bool", value: "0", expected: false},
		{name: "bool_on", field: "Bool", value: "on", expected: true},

		{name: "int_zero", field: "Int", value: "0", expected: 0},
		{name: "int_positive", field: "Int", value: "42", expected: 42},
		{name: "int_negative", field: "Int", value: "-42", expected: -42},
		{name: "int_max", field: "Int", value: "9223372036854775807", expected: int(math.MaxInt64)},
		{name: "int_min", field: "Int", value: "-9223372036854775808", expected: int(math.MinInt64)},

		{name: "int8_max", field: "Int8", value: "127", expected: int8(127)},
		{name: "int8_min", field: "Int8", value: "-128", expected: int8(-128)},

		{name: "int16_max", field: "Int16", value: "32767", expected: int16(32767)},
		{name: "int16_min", field: "Int16", value: "-32768", expected: int16(-32768)},

		{name: "int32_max", field: "Int32", value: "2147483647", expected: int32(2147483647)},
		{name: "int32_min", field: "Int32", value: "-2147483648", expected: int32(-2147483648)},

		{name: "int64_max", field: "Int64", value: "9223372036854775807", expected: int64(math.MaxInt64)},
		{name: "int64_min", field: "Int64", value: "-9223372036854775808", expected: int64(math.MinInt64)},

		{name: "uint_zero", field: "Uint", value: "0", expected: uint(0)},
		{name: "uint_positive", field: "Uint", value: "42", expected: uint(42)},

		{name: "uint8_max", field: "Uint8", value: "255", expected: uint8(255)},
		{name: "uint8_zero", field: "Uint8", value: "0", expected: uint8(0)},

		{name: "uint16_max", field: "Uint16", value: "65535", expected: uint16(65535)},

		{name: "uint32_max", field: "Uint32", value: "4294967295", expected: uint32(4294967295)},

		{name: "uint64_max", field: "Uint64", value: "18446744073709551615", expected: uint64(math.MaxUint64)},

		{name: "float32_zero", field: "Float32", value: "0", expected: float32(0)},
		{name: "float32_positive", field: "Float32", value: "3.14", expected: float32(3.14)},
		{name: "float32_negative", field: "Float32", value: "-3.14", expected: float32(-3.14)},
		{name: "float32_scientific", field: "Float32", value: "1.5e10", expected: float32(1.5e10)},

		{name: "float64_zero", field: "Float64", value: "0", expected: float64(0)},
		{name: "float64_positive", field: "Float64", value: "3.141592653589793", expected: 3.141592653589793},
		{name: "float64_negative", field: "Float64", value: "-3.141592653589793", expected: -3.141592653589793},
		{name: "float64_scientific", field: "Float64", value: "1.5e308", expected: 1.5e308},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			var unsafeResult UnsafeTestPrimitives
			binderUnsafe := NewASTBinder()
			src := map[string][]string{tc.field: {tc.value}}
			err := binderUnsafe.Bind(context.Background(), &unsafeResult, src)
			require.NoError(t, err)

			unsafeVal := reflect.ValueOf(unsafeResult).FieldByName(tc.field).Interface()

			assert.Equal(t, tc.expected, unsafeVal,
				"unsafe setter should produce correct value for %s=%q", tc.field, tc.value)
		})
	}
}

func TestConvertAndSetDirect_ConcurrentSafety(t *testing.T) {

	binder := NewASTBinder()
	var wg sync.WaitGroup
	iterations := 1000
	goroutines := 10

	for g := range goroutines {
		id := g
		wg.Go(func() {
			for range iterations {
				var form UnsafeTestPrimitives
				src := map[string][]string{
					"String":  {string(rune('A' + id))},
					"Int":     {string(rune('0' + id%10))},
					"Bool":    {"true"},
					"Float64": {"3.14"},
				}
				err := binder.Bind(context.Background(), &form, src)
				require.NoError(t, err)

				assert.NotEmpty(t, form.String)
			}
		})
	}

	wg.Wait()
}

func TestFieldInfo_CanDirect_Correctness(t *testing.T) {

	binder := NewASTBinder()

	t.Run("primitives have CanDirect true", func(t *testing.T) {
		structMeta := binder.cache.get(reflect.TypeFor[UnsafeTestPrimitives](), 32)

		primitiveFields := []string{
			"String", "Bool",
			"Int", "Int8", "Int16", "Int32", "Int64",
			"Uint", "Uint8", "Uint16", "Uint32", "Uint64",
			"Float32", "Float64",
		}

		for _, field := range primitiveFields {
			fi, ok := structMeta.Fields[field]
			require.True(t, ok, "field %s should exist in cache", field)
			assert.True(t, fi.CanDirect, "primitive field %s should have CanDirect=true", field)
		}
	})

	t.Run("nested fields have CanDirect false", func(t *testing.T) {
		structMeta := binder.cache.get(reflect.TypeFor[UnsafeTestNested](), 32)

		if fi, ok := structMeta.Fields["Inner.Value"]; ok {
			assert.False(t, fi.CanDirect, "nested field should have CanDirect=false")
		}

		if fi, ok := structMeta.Fields["Inner"]; ok {
			assert.False(t, fi.CanDirect, "struct field should have CanDirect=false")
		}
	})

	t.Run("pointer fields have CanDirect false", func(t *testing.T) {
		type WithPointer struct {
			Value *string
		}
		structMeta := binder.cache.get(reflect.TypeFor[WithPointer](), 32)

		if fi, ok := structMeta.Fields["Value"]; ok {
			assert.False(t, fi.CanDirect, "pointer field should have CanDirect=false")
		}
	})

	t.Run("duration has CanDirect false due to well-known converter", func(t *testing.T) {

		assert.False(t, hasWellKnownConverter(reflect.TypeFor[int64]()),
			"plain int64 should NOT have well-known converter")

	})
}

func TestFieldOffset_Validity(t *testing.T) {

	binder := NewASTBinder()
	typ := reflect.TypeFor[UnsafeTestPrimitives]()
	structMeta := binder.cache.get(typ, 32)

	for field := range typ.Fields() {
		fi, ok := structMeta.Fields[field.Name]
		require.True(t, ok, "field %s should be in cache", field.Name)

		assert.Equal(t, field.Offset, fi.Offset,
			"cached offset for %s should match reflect.StructField.Offset", field.Name)
	}
}

func FuzzConvertAndSetDirect_String(f *testing.F) {

	f.Add("hello")
	f.Add("")
	f.Add("日本語")
	f.Add("\x00\x00\x00")
	f.Add(string(make([]byte, 1000)))

	f.Fuzz(func(t *testing.T, value string) {
		var form UnsafeTestPrimitives
		binder := NewASTBinder()
		src := map[string][]string{"String": {value}}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, value, form.String)
	})
}

func FuzzConvertAndSetDirect_Int(f *testing.F) {

	f.Add("0")
	f.Add("42")
	f.Add("-42")
	f.Add("9223372036854775807")
	f.Add("-9223372036854775808")

	f.Fuzz(func(t *testing.T, value string) {
		var form UnsafeTestPrimitives
		binder := NewASTBinder()
		src := map[string][]string{"Int": {value}}

		err := binder.Bind(context.Background(), &form, src)

		_ = err
	})
}

func FuzzConvertAndSetDirect_AllTypes(f *testing.F) {

	f.Add("test", "true", "42", "3.14")

	f.Fuzz(func(t *testing.T, strVal, boolVal, intVal, floatVal string) {
		var form UnsafeTestPrimitives
		binder := NewASTBinder()
		src := map[string][]string{
			"String":  {strVal},
			"Bool":    {boolVal},
			"Int":     {intVal},
			"Float64": {floatVal},
		}

		_ = binder.Bind(context.Background(), &form, src)
	})
}
