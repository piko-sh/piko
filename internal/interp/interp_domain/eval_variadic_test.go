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

func TestEvalVariadicCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "variadic int sum",
			code: `func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}
sum(1, 2, 3, 4, 5)`,
			expect: int64(15),
		},
		{
			name: "variadic with fixed params",
			code: `func first(sep string, parts ...string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result = result + sep
		}
		result = result + p
	}
	return result
}
first("-", "a", "b", "c")`,
			expect: "a-b-c",
		},
		{
			name: "variadic no arguments",
			code: `func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}
sum()`,
			expect: int64(0),
		},
		{
			name: "variadic single argument",
			code: `func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}
sum(42)`,
			expect: int64(42),
		},
		{
			name: "variadic spread",
			code: `func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}
s := []int{10, 20, 30}
sum(s...)`,
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
