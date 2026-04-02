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

package config_domain

import (
	"encoding"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetFieldString(t *testing.T) {
	var s string
	field := reflect.ValueOf(&s).Elem()

	err := setField(field, "hello", "")
	require.NoError(t, err)
	assert.Equal(t, "hello", s)
}

func TestSetFieldInt(t *testing.T) {
	testCases := []struct {
		expected any
		setup    func() reflect.Value
		name     string
		input    string
		wantErr  bool
	}{
		{
			name: "int",
			setup: func() reflect.Value {
				var i int
				return reflect.ValueOf(&i).Elem()
			},
			input:    "42",
			expected: 42,
		},
		{
			name: "int8",
			setup: func() reflect.Value {
				var i int8
				return reflect.ValueOf(&i).Elem()
			},
			input:    "127",
			expected: int8(127),
		},
		{
			name: "int16",
			setup: func() reflect.Value {
				var i int16
				return reflect.ValueOf(&i).Elem()
			},
			input:    "32767",
			expected: int16(32767),
		},
		{
			name: "int32",
			setup: func() reflect.Value {
				var i int32
				return reflect.ValueOf(&i).Elem()
			},
			input:    "2147483647",
			expected: int32(2147483647),
		},
		{
			name: "int64",
			setup: func() reflect.Value {
				var i int64
				return reflect.ValueOf(&i).Elem()
			},
			input:    "9223372036854775807",
			expected: int64(9223372036854775807),
		},
		{
			name: "negative int",
			setup: func() reflect.Value {
				var i int
				return reflect.ValueOf(&i).Elem()
			},
			input:    "-42",
			expected: -42,
		},
		{
			name: "invalid int",
			setup: func() reflect.Value {
				var i int
				return reflect.ValueOf(&i).Elem()
			},
			input:   "not-a-number",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field := tc.setup()
			err := setField(field, tc.input, "")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, field.Interface())
			}
		})
	}
}

func TestSetFieldUint(t *testing.T) {
	testCases := []struct {
		expected any
		setup    func() reflect.Value
		name     string
		input    string
		wantErr  bool
	}{
		{
			name: "uint",
			setup: func() reflect.Value {
				var u uint
				return reflect.ValueOf(&u).Elem()
			},
			input:    "42",
			expected: uint(42),
		},
		{
			name: "uint8",
			setup: func() reflect.Value {
				var u uint8
				return reflect.ValueOf(&u).Elem()
			},
			input:    "255",
			expected: uint8(255),
		},
		{
			name: "uint16",
			setup: func() reflect.Value {
				var u uint16
				return reflect.ValueOf(&u).Elem()
			},
			input:    "65535",
			expected: uint16(65535),
		},
		{
			name: "uint32",
			setup: func() reflect.Value {
				var u uint32
				return reflect.ValueOf(&u).Elem()
			},
			input:    "4294967295",
			expected: uint32(4294967295),
		},
		{
			name: "uint64",
			setup: func() reflect.Value {
				var u uint64
				return reflect.ValueOf(&u).Elem()
			},
			input:    "18446744073709551615",
			expected: uint64(18446744073709551615),
		},
		{
			name: "invalid uint (negative)",
			setup: func() reflect.Value {
				var u uint
				return reflect.ValueOf(&u).Elem()
			},
			input:   "-1",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field := tc.setup()
			err := setField(field, tc.input, "")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, field.Interface())
			}
		})
	}
}

func TestSetFieldFloat(t *testing.T) {
	testCases := []struct {
		expected any
		setup    func() reflect.Value
		name     string
		input    string
		wantErr  bool
	}{
		{
			name: "float32",
			setup: func() reflect.Value {
				var f float32
				return reflect.ValueOf(&f).Elem()
			},
			input:    "3.14",
			expected: float32(3.14),
		},
		{
			name: "float64",
			setup: func() reflect.Value {
				var f float64
				return reflect.ValueOf(&f).Elem()
			},
			input:    "3.141592653589793",
			expected: 3.141592653589793,
		},
		{
			name: "negative float",
			setup: func() reflect.Value {
				var f float64
				return reflect.ValueOf(&f).Elem()
			},
			input:    "-123.456",
			expected: -123.456,
		},
		{
			name: "invalid float",
			setup: func() reflect.Value {
				var f float64
				return reflect.ValueOf(&f).Elem()
			},
			input:   "not-a-float",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field := tc.setup()
			err := setField(field, tc.input, "")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, field.Interface())
			}
		})
	}
}

func TestSetFieldBool(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
		wantErr  bool
	}{
		{name: "true", input: "true", expected: true},
		{name: "false", input: "false", expected: false},
		{name: "1", input: "1", expected: true},
		{name: "0", input: "0", expected: false},
		{name: "True", input: "True", expected: true},
		{name: "FALSE", input: "FALSE", expected: false},
		{name: "invalid", input: "maybe", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var b bool
			field := reflect.ValueOf(&b).Elem()
			err := setField(field, tc.input, "")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, b)
			}
		})
	}
}

func TestSetFieldDuration(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{name: "seconds", input: "5s", expected: 5 * time.Second},
		{name: "minutes", input: "10m", expected: 10 * time.Minute},
		{name: "hours", input: "2h", expected: 2 * time.Hour},
		{name: "complex", input: "1h30m45s", expected: 1*time.Hour + 30*time.Minute + 45*time.Second},
		{name: "milliseconds", input: "500ms", expected: 500 * time.Millisecond},
		{name: "invalid", input: "not-a-duration", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var d time.Duration
			field := reflect.ValueOf(&d).Elem()
			err := setField(field, tc.input, "")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, d)
			}
		})
	}
}

func TestSetFieldSlice(t *testing.T) {
	testCases := []struct {
		expected any
		setup    func() reflect.Value
		name     string
		input    string
		tags     reflect.StructTag
		wantErr  bool
	}{
		{
			name: "string slice with default delimiter",
			setup: func() reflect.Value {
				var s []string
				return reflect.ValueOf(&s).Elem()
			},
			input:    "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name: "string slice with custom delimiter",
			setup: func() reflect.Value {
				var s []string
				return reflect.ValueOf(&s).Elem()
			},
			input:    "a|b|c",
			tags:     `delimiter:"|"`,
			expected: []string{"a", "b", "c"},
		},
		{
			name: "int slice",
			setup: func() reflect.Value {
				var s []int
				return reflect.ValueOf(&s).Elem()
			},
			input:    "1,2,3",
			expected: []int{1, 2, 3},
		},
		{
			name: "empty slice",
			setup: func() reflect.Value {
				var s []string
				return reflect.ValueOf(&s).Elem()
			},
			input:    "",
			expected: []string{},
		},
		{
			name: "byte slice",
			setup: func() reflect.Value {
				var s []byte
				return reflect.ValueOf(&s).Elem()
			},
			input:    "hello",
			expected: []byte("hello"),
		},
		{
			name: "slice with whitespace",
			setup: func() reflect.Value {
				var s []string
				return reflect.ValueOf(&s).Elem()
			},
			input:    " a , b , c ",
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field := tc.setup()
			err := setField(field, tc.input, tc.tags)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, field.Interface())
			}
		})
	}
}

func TestSetFieldMap(t *testing.T) {
	testCases := []struct {
		expected any
		setup    func() reflect.Value
		name     string
		input    string
		tags     reflect.StructTag
		wantErr  bool
	}{
		{
			name: "string map with default delimiters",
			setup: func() reflect.Value {
				var m map[string]string
				return reflect.ValueOf(&m).Elem()
			},
			input:    "key1:val1,key2:val2",
			expected: map[string]string{"key1": "val1", "key2": "val2"},
		},
		{
			name: "string map with custom delimiters",
			setup: func() reflect.Value {
				var m map[string]string
				return reflect.ValueOf(&m).Elem()
			},
			input:    "key1=val1|key2=val2",
			tags:     `delimiter:"|" separator:"="`,
			expected: map[string]string{"key1": "val1", "key2": "val2"},
		},
		{
			name: "empty map",
			setup: func() reflect.Value {
				var m map[string]string
				return reflect.ValueOf(&m).Elem()
			},
			input:    "",
			expected: map[string]string{},
		},
		{
			name: "map with int values",
			setup: func() reflect.Value {
				var m map[string]int
				return reflect.ValueOf(&m).Elem()
			},
			input:    "a:1,b:2",
			expected: map[string]int{"a": 1, "b": 2},
		},
		{
			name: "invalid map item (missing separator)",
			setup: func() reflect.Value {
				var m map[string]string
				return reflect.ValueOf(&m).Elem()
			},
			input:   "key1val1",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field := tc.setup()
			err := setField(field, tc.input, tc.tags)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, field.Interface())
			}
		})
	}
}

func TestSetFieldOverwrite(t *testing.T) {
	testCases := []struct {
		name     string
		initial  string
		input    string
		tags     reflect.StructTag
		expected string
	}{
		{
			name:     "overwrites by default",
			initial:  "original",
			input:    "new",
			expected: "new",
		},
		{
			name:     "skips when overwrite=false and not zero",
			initial:  "original",
			input:    "new",
			tags:     `overwrite:"false"`,
			expected: "original",
		},
		{
			name:     "sets when overwrite=false and zero",
			initial:  "",
			input:    "new",
			tags:     `overwrite:"false"`,
			expected: "new",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.initial
			field := reflect.ValueOf(&s).Elem()

			err := setField(field, tc.input, tc.tags)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, s)
		})
	}
}

func TestSetFieldPointer(t *testing.T) {
	var s *string
	field := reflect.ValueOf(&s).Elem()

	err := setField(field, "hello", "")
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, "hello", *s)
}

func TestSetFieldUnsupportedType(t *testing.T) {
	var c complex64
	field := reflect.ValueOf(&c).Elem()

	err := setField(field, "1+2i", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

type customTextType struct {
	value string
}

func (c *customTextType) UnmarshalText(text []byte) error {
	c.value = "custom-" + string(text)
	return nil
}

var _ encoding.TextUnmarshaler = (*customTextType)(nil)

func TestSetFieldTextUnmarshaler(t *testing.T) {
	var c customTextType
	field := reflect.ValueOf(&c).Elem()

	err := setField(field, "value", "")
	require.NoError(t, err)
	assert.Equal(t, "custom-value", c.value)
}

func TestIsUnmarshaler(t *testing.T) {
	testCases := []struct {
		value    any
		name     string
		expected bool
	}{
		{
			name:     "TextUnmarshaler",
			value:    &customTextType{},
			expected: true,
		},
		{
			name:     "non-unmarshaler",
			value:    new(string),
			expected: false,
		},
		{
			name:     "int",
			value:    new(int),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field := reflect.ValueOf(tc.value).Elem()
			assert.Equal(t, tc.expected, isUnmarshaler(field))
		})
	}
}
