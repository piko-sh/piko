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

func TestEvalRuntimeFloatComparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "eq true", code: `x := 1.0; y := 1.0; x == y`, expect: true},
		{name: "eq false", code: `x := 1.0; y := 2.0; x == y`, expect: false},
		{name: "lt true", code: `x := 1.0; y := 2.0; x < y`, expect: true},
		{name: "lt false", code: `x := 2.0; y := 1.0; x < y`, expect: false},
		{name: "le true equal", code: `x := 1.0; y := 1.0; x <= y`, expect: true},
		{name: "le false", code: `x := 2.0; y := 1.0; x <= y`, expect: false},
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

func TestEvalRuntimeStringComparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "lt true", code: `a := "abc"; b := "abd"; a < b`, expect: true},
		{name: "le true equal", code: `a := "abc"; b := "abc"; a <= b`, expect: true},
		{name: "eq true", code: `a := "abc"; b := "abc"; a == b`, expect: true},
		{name: "eq false", code: `a := "abc"; b := "xyz"; a == b`, expect: false},
		{name: "lt false", code: `a := "z"; b := "a"; a < b`, expect: false},
		{name: "le false", code: `a := "z"; b := "a"; a <= b`, expect: false},
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

func TestEvalRuntimeNotOperator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "int ne true", code: `x := 1; y := 2; x != y`, expect: true},
		{name: "int ne false", code: `x := 1; y := 1; x != y`, expect: false},
		{name: "not true var", code: `x := true; !x`, expect: false},
		{name: "not false var", code: `x := false; !x`, expect: true},
		{name: "float ne true", code: `x := 1.0; y := 2.0; x != y`, expect: true},
		{name: "float ne false", code: `x := 1.0; y := 1.0; x != y`, expect: false},
		{name: "string ne true", code: `a := "x"; b := "y"; a != b`, expect: true},
		{name: "string ne false", code: `a := "x"; b := "x"; a != b`, expect: false},
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

func TestEvalRuntimeLogicalOps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "and both true", code: `x := true; y := true; x && y`, expect: true},
		{name: "and true false", code: `x := true; y := false; x && y`, expect: false},
		{name: "and false short-circuit", code: `x := false; y := true; x && y`, expect: false},
		{name: "or both false", code: `x := false; y := false; x || y`, expect: false},
		{name: "or false true", code: `x := false; y := true; x || y`, expect: true},
		{name: "or true short-circuit", code: `x := true; y := false; x || y`, expect: true},
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

func TestEvalRuntimeGeneralComparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "eq via interface func",
			code: `func eq(a, b any) bool { return a == b }
eq(1, 1)`,
			expect: true,
		},
		{
			name: "ne via interface func",
			code: `func eq(a, b any) bool { return a == b }
eq(1, 2)`,
			expect: false,
		},
		{
			name: "lt via interface func",
			code: `func lt(a, b any) bool { return a.(int) < b.(int) }
lt(1, 2)`,
			expect: true,
		},
		{
			name: "lt false via interface",
			code: `func lt(a, b any) bool { return a.(int) < b.(int) }
lt(2, 1)`,
			expect: false,
		},
		{
			name: "le via interface",
			code: `func le(a, b any) bool { return a.(int) <= b.(int) }
le(1, 1)`,
			expect: true,
		},
		{
			name: "gt via interface",
			code: `func gt(a, b any) bool { return a.(int) > b.(int) }
gt(3, 1)`,
			expect: true,
		},
		{
			name: "ge via interface",
			code: `func ge(a, b any) bool { return a.(int) >= b.(int) }
ge(1, 2)`,
			expect: false,
		},
		{
			name: "string eq via interface",
			code: `func eq(a, b any) bool { return a.(string) == b.(string) }
eq("hello", "hello")`,
			expect: true,
		},
		{
			name: "string lt via interface",
			code: `func lt(a, b any) bool { return a.(string) < b.(string) }
lt("a", "b")`,
			expect: true,
		},
		{
			name: "direct any eq string",
			code: `func eq(a, b any) bool { return a == b }
eq("hello", "hello")`,
			expect: true,
		},
		{
			name: "direct any eq nil",
			code: `func eq(a, b any) bool { return a == b }
eq(nil, nil)`,
			expect: true,
		},
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
