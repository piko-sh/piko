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

func TestEvalNilValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "zero value int var", code: `var x int; x`, expect: int64(0)},
		{name: "zero value float var", code: `var x float64; x`, expect: 0.0},
		{name: "zero value string var", code: `var x string; x`, expect: ""},
		{name: "zero value bool var", code: `var x bool; x`, expect: false},
		{name: "empty slice len", code: `s := make([]int, 0); len(s)`, expect: int64(0)},
		{name: "empty slice cap", code: `s := make([]int, 0, 10); cap(s)`, expect: int64(10)},
		{name: "multiple zero vars", code: `var a int; var b int; a + b`, expect: int64(0)},
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

func TestEvalNilSliceOperations(t *testing.T) {
	t.Parallel()

	t.Run("nil slice len", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `var s []int; len(s)`)
		require.NoError(t, err)
		require.Equal(t, int64(0), result)
	})

	t.Run("nil slice cap", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `var s []int; cap(s)`)
		require.NoError(t, err)
		require.Equal(t, int64(0), result)
	})
}

func TestEvalNilComparisons(t *testing.T) {
	t.Parallel()

	t.Run("nil slice comparison", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `var s []int; s == nil`)
		require.NoError(t, err)
		require.Equal(t, true, result)
	})

	t.Run("nil map comparison", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `var m map[string]int; m == nil`)
		require.NoError(t, err)
		require.Equal(t, true, result)
	})

	t.Run("nil pointer comparison", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `var p *int; p == nil`)
		require.NoError(t, err)
		require.Equal(t, true, result)
	})

	t.Run("nil map len", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `var m map[string]int; len(m)`)
		require.NoError(t, err)
		require.Equal(t, int64(0), result)
	})
}
