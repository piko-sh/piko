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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalTypeAssertCommaOk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "successful string assertion returns value",
			code: `var i interface{} = "hello"
s, ok := i.(string)
_ = ok
s`,
			expect: "hello",
		},
		{
			name: "successful int assertion returns value",
			code: `var i interface{} = 42
v, ok := i.(int)
_ = ok
v`,
			expect: int64(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}

	t.Run("failed assertion comma-ok returns false", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), "var i interface{} = \"hello\"\n_, ok := i.(int)\nok")
		require.NoError(t, err)
		require.Equal(t, false, result)
	})
}

func TestEvalTypeSwitchFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "type switch dispatches to int case",
			code: `func check(v interface{}) string {
	switch v.(type) {
	case int:
		return "int"
	case string:
		return "string"
	default:
		return "other"
	}
}
check(42)`,
			expect: "int",
		},
		{
			name: "type switch dispatches to string case",
			code: `func check(v interface{}) string {
	switch v.(type) {
	case int:
		return "int"
	case string:
		return "string"
	default:
		return "other"
	}
}
check("hello")`,
			expect: "string",
		},
		{
			name: "type switch dispatches to default case",
			code: `func check(v interface{}) string {
	switch v.(type) {
	case int:
		return "int"
	case string:
		return "string"
	default:
		return "other"
	}
}
check(3.14)`,
			expect: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalInterfacePacking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "pack int into interface",
			code:   "var i interface{} = 42\ni",
			expect: int64(42),
		},
		{
			name:   "pack string into interface",
			code:   "var i interface{} = \"hello\"\ni",
			expect: "hello",
		},
		{
			name:   "pack float into interface",
			code:   "var i interface{} = 3.14\ni",
			expect: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalConvertRuntime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "float to int via variable",
			code:   "x := 3.14\nint(x)",
			expect: int64(3),
		},
		{
			name:   "int to float via variable",
			code:   "x := 42\nfloat64(x)",
			expect: float64(42),
		},
		{
			name:   "bytes to string via variable",
			code:   "b := []byte{72, 105}\nstring(b)",
			expect: "Hi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}

	t.Run("string to bytes does not error", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), "s := \"hello\"\n[]byte(s)")
		require.NoError(t, err)
	})
}
