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

func TestEvalComplexArithmetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect complex128
	}{
		{name: "literal", code: "(1+2i)", expect: 1 + 2i},
		{name: "add", code: "(1+2i) + (3+4i)", expect: 4 + 6i},
		{name: "sub", code: "(5+7i) - (1+2i)", expect: 4 + 5i},
		{name: "mul", code: "(2+3i) * (4+5i)", expect: -7 + 22i},
		{name: "div", code: "(10+0i) / (2+0i)", expect: 5 + 0i},
		{name: "neg", code: "-(3+4i)", expect: -3 - 4i},
		{name: "sub var", code: "a := 5+7i; b := 1+2i; a - b", expect: 4 + 5i},
		{name: "mul var", code: "a := 2+3i; b := 4+5i; a * b", expect: -7 + 22i},
		{name: "div var", code: "a := 10+0i; b := 2+0i; a / b", expect: 5 + 0i},
		{name: "neg var", code: "a := 3+4i; -a", expect: -3 - 4i},
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

func TestEvalComplexComparison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect bool
	}{
		{name: "eq true", code: "(1+2i) == (1+2i)", expect: true},
		{name: "eq false", code: "(1+2i) == (3+4i)", expect: false},
		{name: "ne true", code: "(1+2i) != (3+4i)", expect: true},
		{name: "ne false", code: "(1+2i) != (1+2i)", expect: false},
		{name: "eq var true", code: "a := 1+2i; b := 1+2i; a == b", expect: true},
		{name: "eq var false", code: "a := 1+2i; b := 3+4i; a == b", expect: false},
		{name: "ne var true", code: "a := 1+2i; b := 3+4i; a != b", expect: true},
		{name: "ne var false", code: "a := 1+2i; b := 1+2i; a != b", expect: false},
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

func TestEvalComplexBuiltins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "real", code: "real(3+4i)", expect: float64(3)},
		{name: "imag", code: "imag(3+4i)", expect: float64(4)},
		{name: "complex", code: "complex(3.0, 4.0)", expect: complex128(3 + 4i)},
		{name: "roundtrip", code: "complex(real(1+2i), imag(1+2i))", expect: complex128(1 + 2i)},
		{name: "real var", code: "c := 3+4i; real(c)", expect: float64(3)},
		{name: "imag var", code: "c := 3+4i; imag(c)", expect: float64(4)},
		{name: "complex var", code: "var r float64 = 3.0; var i float64 = 4.0; complex(r, i)", expect: complex128(3 + 4i)},
		{name: "roundtrip var", code: "c := 1+2i; complex(real(c), imag(c))", expect: complex128(1 + 2i)},
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

func TestEvalComplexVariable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "var decl", code: "var c complex128 = 1+2i; c", expect: complex128(1 + 2i)},
		{name: "assign and add", code: "c := 1+2i; c = c + (3+4i); c", expect: complex128(4 + 6i)},
		{
			name:   "complex zero var",
			code:   "var c complex128; c",
			expect: complex128(0),
		},
		{
			name:   "global complex",
			code:   "var g complex128\nfunc s() { g = 1+2i }\ns()\ng",
			expect: complex128(1 + 2i),
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
