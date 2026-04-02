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

func TestEvalExplicitReturnVoid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "explicit return in void function mutates slice",
			code: `func f(s []int) { s[0] = 99; return }
s := make([]int, 1)
f(s)
s[0]`,
			expect: int64(99),
		},
		{
			name: "early return in void function via conditional",
			code: `func f(s []int) {
	if len(s) > 0 {
		s[0] = 42
		return
	}
	s[0] = 0
}
s := []int{0}
f(s)
s[0]`,
			expect: int64(42),
		},
		{
			name: "explicit return skips subsequent statements",
			code: `func f(s []int) {
	s[0] = 10
	return
	s[0] = 20
}
s := make([]int, 1)
f(s)
s[0]`,
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

func TestEvalNilLoad(t *testing.T) {
	t.Parallel()

	t.Run("nil passed to function returns sentinel", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), "func f(s []int) int { if s == nil { return -1 }; return len(s) }\nf(nil)")
		require.NoError(t, err)
		require.Equal(t, int64(-1), result)
	})

	t.Run("nil slice append creates new slice", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), "s := append([]int(nil), 1, 2, 3)\nlen(s)")
		require.NoError(t, err)
		require.Equal(t, int64(3), result)
	})

	t.Run("var slice append without initialiser", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), "var s []int\ns = append(s, 10)\ns = append(s, 20)\ns = append(s, 30)\ns[0] + s[1] + s[2]")
		require.NoError(t, err)
		require.Equal(t, int64(60), result)
	})

	t.Run("nil map passed to function", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), "func f(m map[string]int) int { if m == nil { return -1 }; return len(m) }\nf(nil)")
		require.NoError(t, err)
		require.Equal(t, int64(-1), result)
	})

	t.Run("make empty slice then var nil check", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), "s := make([]int, 0)\nlen(s)")
		require.NoError(t, err)
		require.Equal(t, int64(0), result)
	})
}

func TestEvalGeneralComparisonsExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "struct equality same values",
			code: `type Point struct { X int; Y int }
a := Point{X: 1, Y: 2}
b := Point{X: 1, Y: 2}
a == b`,
			expect: true,
		},
		{
			name: "struct inequality different values",
			code: `type Point struct { X int; Y int }
a := Point{X: 1, Y: 2}
b := Point{X: 3, Y: 4}
a == b`,
			expect: false,
		},
		{
			name: "struct not-equal operator",
			code: `type Point struct { X int; Y int }
a := Point{X: 1, Y: 2}
b := Point{X: 3, Y: 4}
a != b`,
			expect: true,
		},
		{
			name: "interface nil equality via function",
			code: `func isNil(v any) bool { return v == nil }
isNil(nil)`,
			expect: true,
		},
		{
			name: "interface non-nil equality via function",
			code: `func isNil(v any) bool { return v == nil }
isNil(42)`,
			expect: false,
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

func TestEvalDeferExplicit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "defer runs after early return",
			code: `func run(s []int) {
	defer func() { s[0] = s[0] + 100 }()
	s[0] = 5
	return
}
s := make([]int, 1)
run(s)
s[0]`,
			expect: int64(105),
		},
		{
			name: "multiple defers LIFO order with early return",
			code: `func run(s []int) {
	defer func() { s[0] = s[0] * 10 }()
	defer func() { s[0] = s[0] + 5 }()
	s[0] = 1
	return
}
s := make([]int, 1)
run(s)
s[0]`,
			expect: int64(60),
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

func TestEvalDeferExplicitConditional(t *testing.T) {
	t.Parallel()

	t.Run("defer in conditional branch fires when taken", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `func run(s []int, flag bool) {
	if flag {
		defer func() { s[0] = s[0] + 50 }()
	}
	s[0] = 10
}
s := make([]int, 1)
run(s, true)
s[0]`)
		require.NoError(t, err)
		require.Equal(t, int64(60), result)
	})

	t.Run("defer does not fire when branch not taken", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `func run(s []int, flag bool) {
	if flag {
		defer func() { s[0] = s[0] + 50 }()
	}
	s[0] = 10
}
s := make([]int, 1)
run(s, false)
s[0]`)
		require.NoError(t, err)
		require.Equal(t, int64(10), result)
	})
}
