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

package binder

import (
	"image/color"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/maths"
)

func makeFieldInfo(path string, t reflect.Type) *fieldInfo {
	fi := &fieldInfo{
		Path: path,
		Type: t,
	}

	effectiveType := t
	if effectiveType.Kind() == reflect.Pointer {
		effectiveType = effectiveType.Elem()
	}
	if unmarshaler, ok := implementsTextUnmarshaler(effectiveType); ok {
		fi.unmarshaler = unmarshaler
	}
	return fi
}

func TestConvertAndSet(t *testing.T) {
	binder := NewASTBinder()

	t.Run("sets non-pointer primitive value correctly", func(t *testing.T) {
		var target int
		v := reflect.ValueOf(&target).Elem()
		fi := makeFieldInfo("Age", reflect.TypeFor[int]())
		err := binder.convertAndSet(v, "123", "Age", fi)
		require.NoError(t, err)
		assert.Equal(t, 123, target)
	})

	t.Run("sets pointer to primitive value correctly", func(t *testing.T) {
		var target *int
		v := reflect.ValueOf(&target).Elem()
		fi := makeFieldInfo("Age", reflect.TypeFor[*int]())
		err := binder.convertAndSet(v, "456", "Age", fi)
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, 456, *target)
	})

	t.Run("sets non-pointer custom type value correctly", func(t *testing.T) {
		var target maths.Decimal
		v := reflect.ValueOf(&target).Elem()
		fi := makeFieldInfo("Price", reflect.TypeFor[maths.Decimal]())
		err := binder.convertAndSet(v, "19.99", "Price", fi)
		require.NoError(t, err)
		assert.Equal(t, "19.99", target.MustString())
	})

	t.Run("sets pointer to custom type value correctly", func(t *testing.T) {
		var target *maths.Decimal
		v := reflect.ValueOf(&target).Elem()
		fi := makeFieldInfo("Price", reflect.TypeFor[*maths.Decimal]())
		err := binder.convertAndSet(v, "29.99", "Price", fi)
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, "29.99", target.MustString())
	})

	t.Run("sets pointer to named primitive type without panicking", func(t *testing.T) {
		type Tag string
		var target *Tag
		v := reflect.ValueOf(&target).Elem()
		fi := makeFieldInfo("Tag", reflect.TypeFor[*Tag]())
		require.NotPanics(t, func() {
			err := binder.convertAndSet(v, "hello", "Tag", fi)
			require.NoError(t, err)
		})
		require.NotNil(t, target)
		assert.Equal(t, Tag("hello"), *target)
	})

	t.Run("returns error for invalid field", func(t *testing.T) {
		var v reflect.Value
		fi := makeFieldInfo("field", reflect.TypeFor[string]())
		err := binder.convertAndSet(v, "any", "path", fi)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field is not valid")
	})

	t.Run("returns error for un-settable field", func(t *testing.T) {
		type unexported struct {
			name string
		}
		var s unexported
		v := reflect.ValueOf(&s).Elem().FieldByName("name")
		fi := makeFieldInfo("field", reflect.TypeFor[string]())
		err := binder.convertAndSet(v, "any", "path", fi)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be set")
	})

	t.Run("returns conversion error when underlying conversion fails", func(t *testing.T) {
		var target int
		v := reflect.ValueOf(&target).Elem()
		fi := makeFieldInfo("Age", reflect.TypeFor[int]())
		err := binder.convertAndSet(v, "not-a-number", "Age", fi)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not set field 'Age' for path 'Age'")
	})
}

func TestConverterEmptyStringGuards(t *testing.T) {
	testCases := []struct {
		expectedVal any
		converter   func(string) (reflect.Value, error)
		name        string
	}{
		{name: "convertToBool empty", converter: convertToBool, expectedVal: false},
		{name: "convertToInt8 empty", converter: convertToInt8, expectedVal: int8(0)},
		{name: "convertToInt16 empty", converter: convertToInt16, expectedVal: int16(0)},
		{name: "convertToInt32 empty", converter: convertToInt32, expectedVal: int32(0)},
		{name: "convertToUint8 empty", converter: convertToUint8, expectedVal: uint8(0)},
		{name: "convertToUint16 empty", converter: convertToUint16, expectedVal: uint16(0)},
		{name: "convertToUint32 empty", converter: convertToUint32, expectedVal: uint32(0)},
		{name: "convertToUint64 empty", converter: convertToUint64, expectedVal: uint64(0)},
		{name: "convertToFloat32 empty", converter: convertToFloat32, expectedVal: float32(0)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := tc.converter("")
			require.NoError(t, err)
			assert.Equal(t, tc.expectedVal, value.Interface())
		})
	}
}

func TestConverterEdgeCases(t *testing.T) {
	binder := NewASTBinder()

	t.Run("parseTime with empty string returns zero time", func(t *testing.T) {
		fi := makeFieldInfo("Created", reflect.TypeFor[time.Time]())
		value, err := binder.convertToType("", fi)
		require.NoError(t, err)
		assert.True(t, value.Interface().(time.Time).IsZero())
	})

	t.Run("parseURL with empty string returns empty URL", func(t *testing.T) {
		fi := makeFieldInfo("Link", reflect.TypeFor[url.URL]())
		value, err := binder.convertToType("", fi)
		require.NoError(t, err)
		u, ok := value.Interface().(url.URL)
		require.True(t, ok, "expected value to be url.URL")
		assert.Empty(t, u.String())
	})

	t.Run("parseColour with 8-char RRGGBBAA hex", func(t *testing.T) {
		fi := makeFieldInfo("Colour", reflect.TypeFor[color.Color]())
		value, err := binder.convertToType("#FF000080", fi)
		require.NoError(t, err)
		c, ok := value.Interface().(color.RGBA)
		require.True(t, ok, "expected value to be color.RGBA")
		assert.Equal(t, uint8(0xFF), c.R)
		assert.Equal(t, uint8(0x00), c.G)
		assert.Equal(t, uint8(0x00), c.B)
		assert.Equal(t, uint8(0x80), c.A)
	})

	t.Run("parseColour with malformed hex returns error", func(t *testing.T) {
		fi := makeFieldInfo("Colour", reflect.TypeFor[color.Color]())
		_, err := binder.convertToType("XYZ", fi)
		require.Error(t, err)
	})
}

func TestConvertToType(t *testing.T) {
	binder := NewASTBinder()

	testCases := []struct {
		targetType   reflect.Type
		expectedVal  any
		customAssert func(t *testing.T, expected, actual any)
		name         string
		value        string
		errContains  string
		expectErr    bool
	}{
		{
			name:        "Custom type time.Time",
			value:       "2025-10-09T10:00:00Z",
			targetType:  reflect.TypeFor[time.Time](),
			expectedVal: time.Date(2025, 10, 9, 10, 0, 0, 0, time.UTC),
		},
		{
			name:        "Custom type maths.Decimal",
			value:       "123.456",
			targetType:  reflect.TypeFor[maths.Decimal](),
			expectedVal: maths.NewDecimalFromString("123.456"),
			customAssert: func(t *testing.T, expected, actual any) {
				eq, _ := expected.(maths.Decimal).Equals(actual.(maths.Decimal))
				assert.True(t, eq)
			},
		},
		{
			name:        "Custom type maths.Money",
			value:       "50.00",
			targetType:  reflect.TypeFor[maths.Money](),
			expectedVal: maths.NewMoneyFromString("50.00", "GBP"),
		},
		{
			name:        "Custom type time.Time with invalid format",
			value:       "not-a-time",
			targetType:  reflect.TypeFor[time.Time](),
			expectErr:   true,
			errContains: "could not parse",
		},
		{name: "Primitive type string", value: "hello", targetType: reflect.TypeFor[string](), expectedVal: "hello"},
		{name: "Primitive type bool (true)", value: "true", targetType: reflect.TypeFor[bool](), expectedVal: true},
		{name: "Primitive type bool (on)", value: "on", targetType: reflect.TypeFor[bool](), expectedVal: true},
		{name: "Primitive type int", value: "-42", targetType: reflect.TypeFor[int](), expectedVal: int(-42)},
		{name: "Primitive type uint64", value: "18446744073709551615", targetType: reflect.TypeFor[uint64](), expectedVal: uint64(18446744073709551615)},
		{name: "Primitive type float32", value: "3.14", targetType: reflect.TypeFor[float32](), expectedVal: float32(3.14)},
		{name: "Interface type any", value: "dynamic value", targetType: reflect.TypeFor[any](), expectedVal: "dynamic value"},
		{name: "Interface type any with empty string", value: "", targetType: reflect.TypeFor[any](), expectedVal: ""},
		{name: "Interface type any with special chars", value: `{"key": "value"}`, targetType: reflect.TypeFor[any](), expectedVal: `{"key": "value"}`},
		{
			name:        "Unsupported type",
			value:       "any",
			targetType:  reflect.TypeFor[chan int](),
			expectErr:   true,
			errContains: "unsupported type: chan",
		},
		{
			name:        "Invalid bool",
			value:       "maybe",
			targetType:  reflect.TypeFor[bool](),
			expectErr:   true,
			errContains: "invalid syntax",
		},
		{
			name:        "Invalid int",
			value:       "12a",
			targetType:  reflect.TypeFor[int](),
			expectErr:   true,
			errContains: "invalid syntax",
		},
		{
			name:        "Int overflow",
			value:       "128",
			targetType:  reflect.TypeFor[int8](),
			expectErr:   true,
			errContains: "value out of range",
		},
		{
			name:        "Scalar to []string coerces to single-element slice",
			value:       "consent-emails",
			targetType:  reflect.TypeFor[[]string](),
			expectedVal: []string{"consent-emails"},
		},
		{
			name:        "Scalar to []int coerces and converts element",
			value:       "42",
			targetType:  reflect.TypeFor[[]int](),
			expectedVal: []int{42},
		},
		{
			name:        "Empty string to []string yields single empty element",
			value:       "",
			targetType:  reflect.TypeFor[[]string](),
			expectedVal: []string{""},
		},
		{
			name:        "Scalar to []int with invalid element errors",
			value:       "not-a-number",
			targetType:  reflect.TypeFor[[]int](),
			expectErr:   true,
			errContains: "invalid syntax",
		},
		{
			name:        "[]byte is not wrapped and remains unsupported",
			value:       "abc",
			targetType:  reflect.TypeFor[[]byte](),
			expectErr:   true,
			errContains: "unsupported type",
		},
		{
			name:        "Slice of unsupported element still errors sensibly",
			value:       "x",
			targetType:  reflect.TypeFor[[]chan int](),
			expectErr:   true,
			errContains: "unsupported type: chan",
		},
		{
			name:        "Nested slice element is not coerced and errors",
			value:       "x",
			targetType:  reflect.TypeFor[[][]string](),
			expectErr:   true,
			errContains: "unsupported type: [][]string",
		},
		{
			name:        "Nested byte slice element errors sensibly",
			value:       "x",
			targetType:  reflect.TypeFor[[][]byte](),
			expectErr:   true,
			errContains: "unsupported type: []uint8",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fi := makeFieldInfo(tc.name, tc.targetType)
			actualVal, err := binder.convertToType(tc.value, fi)

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				require.True(t, actualVal.IsValid(), "Converter should return a valid reflect.Value on success")
				if tc.customAssert != nil {
					tc.customAssert(t, tc.expectedVal, actualVal.Interface())
				} else {
					assert.Equal(t, tc.expectedVal, actualVal.Interface())
				}
			}
		})
	}
}

func TestConvertToType_ScalarToSlice(t *testing.T) {
	binder := NewASTBinder()

	t.Run("named slice type preserves declared type", func(t *testing.T) {
		type Tags []string
		fi := makeFieldInfo("Tags", reflect.TypeFor[Tags]())
		got, err := binder.convertToType("go", fi)
		require.NoError(t, err)
		require.Equal(t, reflect.TypeFor[Tags](), got.Type())
		assert.Equal(t, Tags{"go"}, got.Interface())
	})

	t.Run("named element type converts before append", func(t *testing.T) {
		type Tag string
		fi := makeFieldInfo("Tags", reflect.TypeFor[[]Tag]())
		got, err := binder.convertToType("go", fi)
		require.NoError(t, err)
		assert.Equal(t, []Tag{"go"}, got.Interface())
	})

	t.Run("pointer to slice is dereferenced and coerced", func(t *testing.T) {
		fi := makeFieldInfo("Tags", reflect.TypeFor[*[]string]())
		got, err := binder.convertToType("go", fi)
		require.NoError(t, err)
		assert.Equal(t, []string{"go"}, got.Interface())
	})

	t.Run("registered element converter is used for slice element", func(t *testing.T) {
		type CustomID string
		b := NewASTBinder()
		b.RegisterConverter(reflect.TypeFor[CustomID](), func(value string) (reflect.Value, error) {
			return reflect.ValueOf(CustomID("id-" + value)), nil
		})
		fi := makeFieldInfo("IDs", reflect.TypeFor[[]CustomID]())
		got, err := b.convertToType("7", fi)
		require.NoError(t, err)
		assert.Equal(t, []CustomID{"id-7"}, got.Interface())
	})

	t.Run("single scalar yields exactly one element", func(t *testing.T) {
		fi := makeFieldInfo("Tags", reflect.TypeFor[[]string]())
		got, err := binder.convertToType("only", fi)
		require.NoError(t, err)
		require.Len(t, got.Interface(), 1)
	})

	t.Run("named byte-slice element coerces rather than being excluded", func(t *testing.T) {
		type Grade uint8
		fi := makeFieldInfo("Grades", reflect.TypeFor[[]Grade]())
		got, err := binder.convertToType("7", fi)
		require.NoError(t, err)
		assert.Equal(t, []Grade{7}, got.Interface())
	})

	t.Run("struct element without TextUnmarshaler errors without panicking", func(t *testing.T) {
		type Point struct{ X int }
		fi := makeFieldInfo("Points", reflect.TypeFor[[]Point]())
		require.NotPanics(t, func() {
			_, err := binder.convertToType("x", fi)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported type")
		})
	})
}

func TestConvertToType_ScalarToPointerSlice(t *testing.T) {
	binder := NewASTBinder()

	t.Run("primitive pointer element binds", func(t *testing.T) {
		require.NotPanics(t, func() {
			fi := makeFieldInfo("Scores", reflect.TypeFor[[]*int]())
			got, err := binder.convertToType("5", fi)
			require.NoError(t, err)
			scores, ok := got.Interface().([]*int)
			require.True(t, ok)
			require.Len(t, scores, 1)
			require.NotNil(t, scores[0])
			assert.Equal(t, 5, *scores[0])
		})
	})

	t.Run("string pointer element binds", func(t *testing.T) {
		require.NotPanics(t, func() {
			fi := makeFieldInfo("Tags", reflect.TypeFor[[]*string]())
			got, err := binder.convertToType("hello", fi)
			require.NoError(t, err)
			tags, ok := got.Interface().([]*string)
			require.True(t, ok)
			require.Len(t, tags, 1)
			require.NotNil(t, tags[0])
			assert.Equal(t, "hello", *tags[0])
		})
	})

	t.Run("named pointer element binds", func(t *testing.T) {
		type Tag string
		require.NotPanics(t, func() {
			fi := makeFieldInfo("Tags", reflect.TypeFor[[]*Tag]())
			got, err := binder.convertToType("go", fi)
			require.NoError(t, err)
			tags, ok := got.Interface().([]*Tag)
			require.True(t, ok)
			require.Len(t, tags, 1)
			require.NotNil(t, tags[0])
			assert.Equal(t, Tag("go"), *tags[0])
		})
	})

	t.Run("well-known pointer element binds", func(t *testing.T) {
		require.NotPanics(t, func() {
			fi := makeFieldInfo("Times", reflect.TypeFor[[]*time.Time]())
			got, err := binder.convertToType("2025-10-09T10:00:00Z", fi)
			require.NoError(t, err)
			times, ok := got.Interface().([]*time.Time)
			require.True(t, ok)
			require.Len(t, times, 1)
			require.NotNil(t, times[0])
			assert.Equal(t, time.Date(2025, 10, 9, 10, 0, 0, 0, time.UTC), *times[0])
		})
	})
}
