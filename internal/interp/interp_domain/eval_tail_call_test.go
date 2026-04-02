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

func TestEvalTailCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
		skip   string
	}{
		{
			name: "self-recursive accumulator",
			code: `func sum(n, acc int) int {
	if n == 0 { return acc }
	return sum(n-1, acc+n)
}
sum(100, 0)`,
			expect: int64(5050),
		},
		{
			name: "countdown to zero",
			code: `func countdown(n int) int {
	if n <= 0 { return 0 }
	return countdown(n-1)
}
countdown(50)`,
			expect: int64(0),
		},
		{
			name: "factorial with accumulator",
			code: `func fact(n, acc int) int {
	if n <= 1 { return acc }
	return fact(n-1, n*acc)
}
fact(10, 1)`,
			expect: int64(3628800),
		},
		{
			name: "deep recursion requiring tail call optimisation",
			code: `func deep(n int) int {
	if n == 0 { return 42 }
	return deep(n-1)
}
deep(10000)`,
			expect: int64(42),
		},
		{
			name: "tail call with string param",
			code: `func repeat(s string, n int) string {
	if n <= 0 { return s }
	return repeat(s + "x", n-1)
}
repeat("", 5)`,
			expect: "xxxxx",
		},
		{
			name: "tail call with float accumulator",
			code: `func addUp(n int, acc float64) float64 {
	if n <= 0 { return acc }
	return addUp(n-1, acc + 1.5)
}
addUp(4, 0.0)`,
			expect: 6.0,
		},
		{
			name: "tail call with bool param",
			code: `func toggle(n int, b bool) bool {
	if n <= 0 { return b }
	return toggle(n-1, !b)
}
toggle(5, false)`,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
