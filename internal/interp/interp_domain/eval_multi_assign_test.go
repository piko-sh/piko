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

func TestEvalMultiAssign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "swap_a",
			code:   "a := 1\nb := 2\na, b = b, a\na",
			expect: int64(2),
		},
		{
			name:   "swap_b",
			code:   "a := 1\nb := 2\na, b = b, a\nb",
			expect: int64(1),
		},
		{
			name:   "triple_swap",
			code:   "a := 1\nb := 2\nc := 3\na, b, c = c, a, b\na",
			expect: int64(3),
		},
		{
			name: "multi_return_assign",
			code: `func divmod(a, b int) (int, int) {
	return a / b, a % b
}
q := 0
r := 0
q, r = divmod(17, 5)
q + r`,
			expect: int64(5),
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
