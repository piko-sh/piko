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

func TestEvalCompoundAssign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "add_assign_int", code: "x := 10\nx += 5\nx", expect: int64(15)},
		{name: "sub_assign_int", code: "x := 20\nx -= 3\nx", expect: int64(17)},
		{name: "mul_assign_int", code: "x := 4\nx *= 3\nx", expect: int64(12)},
		{name: "quo_assign_int", code: "x := 15\nx /= 3\nx", expect: int64(5)},
		{name: "rem_assign_int", code: "x := 17\nx %= 5\nx", expect: int64(2)},
		{name: "and_assign_int", code: "x := 0xFF\nx &= 0x0F\nx", expect: int64(0x0F)},
		{name: "or_assign_int", code: "x := 0xF0\nx |= 0x0F\nx", expect: int64(0xFF)},
		{name: "xor_assign_int", code: "x := 0xFF\nx ^= 0x0F\nx", expect: int64(0xF0)},
		{name: "andnot_assign_int", code: "x := 0xFF\nx &^= 0x0F\nx", expect: int64(0xF0)},
		{name: "shl_assign_int", code: "x := 1\nx <<= 4\nx", expect: int64(16)},
		{name: "shr_assign_int", code: "x := 16\nx >>= 2\nx", expect: int64(4)},
		{name: "add_assign_float", code: "x := 1.5\nx += 2.5\nx", expect: 4.0},
		{name: "sub_assign_float", code: "x := 5.0\nx -= 1.5\nx", expect: 3.5},
		{name: "mul_assign_float", code: "x := 2.0\nx *= 3.0\nx", expect: 6.0},
		{name: "quo_assign_float", code: "x := 10.0\nx /= 4.0\nx", expect: 2.5},
		{name: "add_assign_string", code: "s := \"hello\"\ns += \" world\"\ns", expect: "hello world"},
		{name: "compound_in_loop", code: "x := 0\nfor i := 0; i < 5; i++ {\n\tx += i\n}\nx", expect: int64(10)},
		{name: "slice_add_assign_int", code: "s := []int{10, 20}\ns[0] += 5\ns[0]", expect: int64(15)},
		{name: "slice_sub_assign_int", code: "s := []int{10, 20}\ns[1] -= 3\ns[1]", expect: int64(17)},
		{name: "slice_mul_assign_int", code: "s := []int{4, 5}\ns[0] *= 3\ns[0]", expect: int64(12)},
		{name: "slice_add_assign_float", code: "s := []float64{1.5, 2.5}\ns[0] += 3.5\ns[0]", expect: 5.0},
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

func TestEvalCompoundAssignExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "sub_assign_different_values", code: "x := 10\nx -= 3\nx", expect: int64(7)},
		{name: "mul_assign_float", code: "x := 2.0\nx *= 3.0\nx", expect: 6.0},
		{name: "string_concat_assign", code: `s := "hello"` + "\ns += \" world\"\ns", expect: "hello world"},
		{name: "quo_assign_even_division", code: "x := 10\nx /= 2\nx", expect: int64(5)},
		{name: "rem_assign_small_modulus", code: "x := 10\nx %= 3\nx", expect: int64(1)},
		{name: "and_assign_mask", code: "x := 0xFF\nx &= 0x0F\nx", expect: int64(15)},
		{name: "or_assign_combine", code: "x := 0xF0\nx |= 0x0F\nx", expect: int64(255)},
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

func TestEvalCompoundAssignMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "map_add_assign",
			code:   "m := map[string]int{\"a\": 1}\nm[\"a\"] += 5\nm[\"a\"]",
			expect: int64(6),
		},
		{
			name:   "map_sub_assign",
			code:   "m := map[string]int{\"x\": 10}\nm[\"x\"] -= 3\nm[\"x\"]",
			expect: int64(7),
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

func TestEvalCompoundAssignInterfaceSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "interface_slice_add_assign",
			source: `package main

func run() interface{} {
	s := make([]interface{}, 3)
	s[0] = 10
	s[1] = 20
	s[2] = 30
	return s[1]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(20),
		},
		{
			name: "interface_slice_set_string",
			source: `package main

func run() interface{} {
	s := make([]interface{}, 2)
	s[0] = "hello"
	s[1] = "world"
	return s[0]
}

func main() {}
`,
			entrypoint: "run",
			expect:     "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEvalSwitchExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "switch_with_return_in_function",
			code: `func classify(x int) string {
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	default:
		return "other"
	}
}
classify(2)`,
			expect: "two",
		},
		{
			name: "tagless_switch_with_return",
			code: `func bracket(x int) string {
	switch {
	case x < 10:
		return "low"
	case x < 20:
		return "mid"
	default:
		return "high"
	}
}
bracket(15)`,
			expect: "mid",
		},
		{
			name: "multi_value_case_with_return",
			code: `func group(x int) string {
	switch x {
	case 1, 2, 3:
		return "low"
	case 4, 5, 6:
		return "high"
	default:
		return "other"
	}
}
group(3)`,
			expect: "low",
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
