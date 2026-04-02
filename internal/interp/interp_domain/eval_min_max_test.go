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

func TestEvalMinMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "min_1_2", code: `min(1, 2)`, expect: int64(1)},
		{name: "max_1_2", code: `max(1, 2)`, expect: int64(2)},
		{name: "min_5_3", code: `min(5, 3)`, expect: int64(3)},
		{name: "max_5_3", code: `max(5, 3)`, expect: int64(5)},
		{name: "min_equal", code: `min(1, 1)`, expect: int64(1)},
		{name: "max_equal", code: `max(1, 1)`, expect: int64(1)},
		{name: "min_negative", code: `min(-1, 1)`, expect: int64(-1)},
		{name: "max_negative", code: `max(-1, 1)`, expect: int64(1)},
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

func TestEvalMinMaxVars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "min int vars", code: "a := 5; b := 3; min(a, b)", expect: int64(3)},
		{name: "max int vars", code: "a := 5; b := 3; max(a, b)", expect: int64(5)},
		{name: "min float vars", code: "var a float64 = 1.5; var b float64 = 2.5; min(a, b)", expect: float64(1.5)},
		{name: "max float vars", code: "var a float64 = 1.5; var b float64 = 2.5; max(a, b)", expect: float64(2.5)},
		{name: "min string vars", code: `a := "abc"; b := "abd"; min(a, b)`, expect: "abc"},
		{name: "max string vars", code: `a := "abc"; b := "abd"; max(a, b)`, expect: "abd"},
		{name: "min multi vars", code: "a := 5; b := 3; c := 8; d := 1; min(a, b, c, d)", expect: int64(1)},
		{name: "max multi vars", code: "a := 5; b := 3; c := 8; d := 1; max(a, b, c, d)", expect: int64(8)},
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

func TestEvalMinMaxFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect float64
	}{
		{name: "min_floats", code: `min(3.14, 2.72)`, expect: 2.72},
		{name: "max_floats", code: `max(3.14, 2.72)`, expect: 3.14},
		{name: "min_equal_floats", code: `min(1.0, 1.0)`, expect: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.InDelta(t, tt.expect, result, 0.0001, "code: %s", tt.code)
		})
	}
}

func TestEvalMinMaxString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect string
	}{
		{name: "min_strings", code: `min("abc", "abd")`, expect: "abc"},
		{name: "max_strings", code: `max("abc", "abd")`, expect: "abd"},
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

func TestEvalMinMaxMultipleArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "min_four_args", code: `min(5, 3, 8, 1)`, expect: int64(1)},
		{name: "max_four_args", code: `max(5, 3, 8, 1)`, expect: int64(8)},
		{name: "min_five_args", code: `min(1, 2, 3, 4, 5)`, expect: int64(1)},
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
