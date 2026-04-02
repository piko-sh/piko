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

func TestEvalIntegerArithmetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect int64
	}{
		{name: "addition", code: `1 + 2`, expect: 3},
		{name: "subtraction", code: `10 - 3`, expect: 7},
		{name: "multiplication", code: `4 * 5`, expect: 20},
		{name: "division", code: `15 / 3`, expect: 5},
		{name: "remainder", code: `17 % 5`, expect: 2},
		{name: "negation", code: `-42`, expect: -42},
		{name: "complex expression", code: `(2 + 3) * 4`, expect: 20},
		{name: "nested parens", code: `((1 + 2) * (3 + 4))`, expect: 21},
		{name: "zero", code: `0`, expect: 0},
		{name: "large number", code: `1000000 * 1000000`, expect: 1000000000000},
		{name: "subtraction negative result", code: `3 - 10`, expect: -7},
		{name: "unary plus", code: `+5`, expect: 5},
		{name: "double negation", code: `-(-5)`, expect: 5},
		{name: "mixed operations", code: `2 + 3 * 4`, expect: 14},
		{name: "left associative", code: `10 - 3 - 2`, expect: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEvalFloatArithmetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect float64
	}{
		{name: "addition", code: `1.5 + 2.5`, expect: 4.0},
		{name: "subtraction", code: `10.0 - 3.5`, expect: 6.5},
		{name: "multiplication", code: `2.5 * 4.0`, expect: 10.0},
		{name: "division", code: `15.0 / 4.0`, expect: 3.75},
		{name: "negation", code: `-3.14`, expect: -3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.InDelta(t, tt.expect, result, 0.0001)
		})
	}
}

func TestEvalStringOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect string
	}{
		{name: "literal", code: `"hello"`, expect: "hello"},
		{name: "concatenation", code: `"hello" + " " + "world"`, expect: "hello world"},
		{name: "empty string", code: `""`, expect: ""},
		{name: "concat empty", code: `"hello" + ""`, expect: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEvalBooleanOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "true", code: `true`, expect: true},
		{name: "false", code: `false`, expect: false},
		{name: "not true", code: `!true`, expect: false},
		{name: "not false", code: `!false`, expect: true},
		{name: "double not", code: `!!true`, expect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEvalComparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "int equal true", code: `5 == 5`, expect: true},
		{name: "int equal false", code: `5 == 6`, expect: false},
		{name: "int not equal true", code: `5 != 6`, expect: true},
		{name: "int not equal false", code: `5 != 5`, expect: false},
		{name: "int less than true", code: `3 < 5`, expect: true},
		{name: "int less than false", code: `5 < 3`, expect: false},
		{name: "int less equal true", code: `5 <= 5`, expect: true},
		{name: "int less equal false", code: `6 <= 5`, expect: false},
		{name: "int greater true", code: `5 > 3`, expect: true},
		{name: "int greater false", code: `3 > 5`, expect: false},
		{name: "int greater equal true", code: `5 >= 5`, expect: true},
		{name: "int greater equal false", code: `4 >= 5`, expect: false},
		{name: "string equal true", code: `"abc" == "abc"`, expect: true},
		{name: "string equal false", code: `"abc" == "definition"`, expect: false},
		{name: "string less true", code: `"abc" < "definition"`, expect: true},
		{name: "string less false", code: `"definition" < "abc"`, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEvalBitwiseOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect int64
	}{
		{name: "and", code: `0xFF & 0x0F`, expect: 0x0F},
		{name: "or", code: `0xF0 | 0x0F`, expect: 0xFF},
		{name: "xor", code: `0xFF ^ 0x0F`, expect: 0xF0},
		{name: "and not", code: `0xFF &^ 0x0F`, expect: 0xF0},
		{name: "bit not", code: `^0`, expect: -1},
		{name: "shift left", code: `1 << 4`, expect: 16},
		{name: "shift right", code: `16 >> 2`, expect: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEvalLogicalOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "and both true", code: `true && true`, expect: true},
		{name: "and left false", code: `false && true`, expect: false},
		{name: "and right false", code: `true && false`, expect: false},
		{name: "and both false", code: `false && false`, expect: false},
		{name: "or both true", code: `true || true`, expect: true},
		{name: "or left true", code: `true || false`, expect: true},
		{name: "or right true", code: `false || true`, expect: true},
		{name: "or both false", code: `false || false`, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
