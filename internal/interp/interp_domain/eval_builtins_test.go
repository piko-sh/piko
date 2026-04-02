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

func TestEvalBuiltinMinMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "min_two_ints", code: "min(3, 1)", expect: int64(1)},
		{name: "min_three_ints", code: "min(5, 2, 8)", expect: int64(2)},
		{name: "max_two_ints", code: "max(3, 7)", expect: int64(7)},
		{name: "max_three_ints", code: "max(1, 9, 4)", expect: int64(9)},
		{name: "min_floats", code: "min(2.5, 1.5)", expect: 1.5},
		{name: "max_floats", code: "max(2.5, 1.5)", expect: 2.5},
		{name: "min_equal", code: "min(5, 5)", expect: int64(5)},
		{name: "max_equal", code: "max(5, 5)", expect: int64(5)},
		{name: "min_negative", code: "min(-3, -1, -5)", expect: int64(-5)},
		{name: "max_negative", code: "max(-3, -1, -5)", expect: int64(-1)},
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

func TestEvalBuiltinClear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "clear_map",
			code:   "m := map[string]int{\"a\": 1, \"b\": 2}\nclear(m)\nlen(m)",
			expect: int64(0),
		},
		{
			name:   "clear_slice",
			code:   "s := []int{1, 2, 3}\nclear(s)\ns[0]",
			expect: int64(0),
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
