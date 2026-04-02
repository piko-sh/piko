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
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalUintArithmetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect uint64
	}{
		{name: "add", code: "var a uint = 5; var b uint = 3; a + b", expect: 8},
		{name: "sub", code: "var a uint = 10; var b uint = 3; a - b", expect: 7},
		{name: "mul", code: "var a uint = 4; var b uint = 5; a * b", expect: 20},
		{name: "div", code: "var a uint = 15; var b uint = 3; a / b", expect: 5},
		{name: "rem", code: "var a uint = 17; var b uint = 5; a % b", expect: 2},
		{name: "inc", code: "var a uint = 5; a++; a", expect: 6},
		{name: "dec", code: "var a uint = 5; a--; a", expect: 4},
		{name: "shift left", code: "var a uint = 1; a << 4", expect: 16},
		{name: "shift right", code: "var a uint = 256; a >> 4", expect: 16},
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

func TestEvalUintBitwise(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect uint64
	}{
		{name: "and", code: "var a uint = 0xFF; var b uint = 0x0F; a & b", expect: 0x0F},
		{name: "or", code: "var a uint = 0xF0; var b uint = 0x0F; a | b", expect: 0xFF},
		{name: "xor", code: "var a uint = 0xFF; var b uint = 0x0F; a ^ b", expect: 0xF0},
		{name: "and not", code: "var a uint = 0xFF; var b uint = 0x0F; a &^ b", expect: 0xF0},
		{name: "not", code: "var a uint = 0; ^a", expect: math.MaxUint64},
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

func TestEvalUintComparison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect bool
	}{
		{name: "eq true", code: "var a uint = 5; a == 5", expect: true},
		{name: "eq false", code: "var a uint = 5; a == 3", expect: false},
		{name: "ne true", code: "var a uint = 5; a != 3", expect: true},
		{name: "ne false", code: "var a uint = 5; a != 5", expect: false},
		{name: "lt true", code: "var a uint = 3; a < 5", expect: true},
		{name: "lt false", code: "var a uint = 5; a < 3", expect: false},
		{name: "le true", code: "var a uint = 5; a <= 5", expect: true},
		{name: "le false", code: "var a uint = 6; a <= 5", expect: false},
		{name: "gt true", code: "var a uint = 5; a > 3", expect: true},
		{name: "gt false", code: "var a uint = 3; a > 5", expect: false},
		{name: "ge true", code: "var a uint = 5; a >= 5", expect: true},
		{name: "ge false", code: "var a uint = 4; a >= 5", expect: false},
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

func TestEvalUintConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "uint literal", code: "uint(42)", expect: uint64(42)},
		{name: "uint to int", code: "int(uint(42))", expect: int64(42)},
		{name: "uint to float", code: "float64(uint(42))", expect: float64(42)},
		{name: "int to uint", code: "var i int = 100; uint(i)", expect: uint64(100)},
		{name: "float to uint", code: "var f float64 = 42.9; uint(f)", expect: uint64(42)},
		{name: "uint move", code: "var a uint = 99; b := a; b", expect: uint64(99)},
		{name: "uint var to float", code: "var a uint = 42; float64(a)", expect: float64(42)},
		{name: "uint var to int", code: "var a uint = 42; int(a)", expect: int64(42)},
		{name: "int var to uint", code: "var a int = 100; uint(a)", expect: uint64(100)},
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

func TestEvalUintCompound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect uint64
	}{
		{name: "add assign", code: "var a uint = 5; a += 3; a", expect: 8},
		{name: "sub assign", code: "var a uint = 10; a -= 3; a", expect: 7},
		{name: "mul assign", code: "var a uint = 4; a *= 5; a", expect: 20},
		{name: "div assign", code: "var a uint = 15; a /= 3; a", expect: 5},
		{name: "rem assign", code: "var a uint = 17; a %= 5; a", expect: 2},
		{name: "and assign", code: "var a uint = 0xFF; a &= 0x0F; a", expect: 0x0F},
		{name: "or assign", code: "var a uint = 0xF0; a |= 0x0F; a", expect: 0xFF},
		{name: "xor assign", code: "var a uint = 0xFF; a ^= 0x0F; a", expect: 0xF0},
		{name: "shift left assign", code: "var a uint = 1; a <<= 4; a", expect: 16},
		{name: "shift right assign", code: "var a uint = 256; a >>= 4; a", expect: 16},
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

func TestEvalUintGlobal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "global uint set and get",
			code:   "var g uint = 0\nfunc set() { g = 99 }\nset()\ng",
			expect: uint64(99),
		},
		{
			name:   "global uint default",
			code:   "var g uint\ng",
			expect: uint64(0),
		},
		{
			name:   "global uint increment",
			code:   "var g uint = 5\nfunc inc() { g++ }\ninc()\ninc()\ng",
			expect: uint64(7),
		},
		{
			name:   "global uint compound assign",
			code:   "var g uint = 10\nfunc add(n uint) { g += n }\nadd(5)\ng",
			expect: uint64(15),
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
