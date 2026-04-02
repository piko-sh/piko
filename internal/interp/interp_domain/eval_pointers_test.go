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

func TestEvalPointers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "address and deref int", code: `x := 42; p := &x; *p`, expect: int64(42)},
		{name: "address and deref string", code: `s := "hello"; p := &s; *p`, expect: "hello"},
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

func TestEvalPointerMutation(t *testing.T) {
	t.Parallel()

	t.Run("mutate through pointer", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `x := 1; p := &x; *p = 99; x`)
		require.NoError(t, err)
		require.Equal(t, int64(99), result)
	})

	t.Run("mutate string through pointer", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `s := "old"; p := &s; *p = "new"; s`)
		require.NoError(t, err)
		require.Equal(t, "new", result)
	})
}

func TestEvalPointerNew(t *testing.T) {
	t.Parallel()

	t.Run("new int", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `p := new(int); *p = 5; *p`)
		require.NoError(t, err)
		require.Equal(t, int64(5), result)
	})

	t.Run("new struct", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `type S struct{ X int }; p := new(S); p.X = 7; p.X`)
		require.NoError(t, err)
		require.Equal(t, int64(7), result)
	})
}

func TestEvalNewExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "new int literal",
			code:   `p := new(42); *p`,
			expect: int64(42),
		},
		{
			name:   "new string literal",
			code:   `p := new("hello"); *p`,
			expect: "hello",
		},
		{
			name:   "new expression",
			code:   `x := 10; p := new(x + 32); *p`,
			expect: int64(42),
		},
		{
			name:   "new func call",
			code:   "func d(n int) int { return n * 2 }\np := new(d(21))\n*p",
			expect: int64(42),
		},
		{
			name:   "new expr independence",
			code:   `x := 42; p := new(x); *p = 99; x`,
			expect: int64(42),
		},
		{
			name:   "new type backward compat",
			code:   `p := new(int); *p`,
			expect: int64(0),
		},
		{
			name:   "new bool literal",
			code:   `p := new(true); *p`,
			expect: true,
		},
		{
			name:   "new float literal",
			code:   `p := new(3.14); *p`,
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

func TestEvalAddressOfIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "address of slice element",
			code:   "s := []int{10, 20, 30}\np := &s[1]\n*p",
			expect: int64(20),
		},
		{
			name:   "mutate via slice element pointer",
			code:   "s := []int{10, 20, 30}\np := &s[1]\n*p = 99\ns[1]",
			expect: int64(99),
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

func TestEvalIndirectWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "write through pointer func",
			code:   "func set(p *int, v int) { *p = v }\nx := 0\nset(&x, 42)\nx",
			expect: int64(42),
		},
		{
			name:   "write through pointer string func",
			code:   "func set(p *string, v string) { *p = v }\ns := \"old\"\nset(&s, \"new\")\ns",
			expect: "new",
		},
		{
			name:   "assign heap-escaped variable",
			code:   "x := 5\np := &x\nx = 10\n*p",
			expect: int64(10),
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

func TestEvalPointerToField(t *testing.T) {
	t.Parallel()

	t.Run("pointer to struct field", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `type S struct{ V int }; s := S{V: 10}; p := &s.V; *p`)
		require.NoError(t, err)
		require.Equal(t, int64(10), result)
	})
}
