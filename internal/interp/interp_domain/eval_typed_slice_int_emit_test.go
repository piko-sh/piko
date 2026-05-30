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

func TestTypedSliceIntCompilerEmission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		source string
	}{
		{
			name: "self_contained_make_get_set_len",
			source: `func run() int {
	s := make([]int, 3)
	s[0] = 7
	s[1] = 11
	s[2] = 13
	return s[0] + s[1] + s[2] + len(s)
}
run()`,
			expect: int64(34),
		},
		{
			name: "range_over_typed_slice",
			source: `func run() int {
	s := make([]int, 4)
	s[0] = 1
	s[1] = 2
	s[2] = 3
	s[3] = 4
	total := 0
	for _, value := range s {
		total += value
	}
	return total
}
run()`,
			expect: int64(10),
		},
		{
			name: "address_of_index_disqualifies",
			source: `func run() int {
	s := make([]int, 1)
	s[0] = 99
	pointer := &s[0]
	*pointer = 42
	return s[0]
}
run()`,
			expect: int64(42),
		},
		{
			name: "closure_capture_disqualifies",
			source: `func run() int {
	s := make([]int, 1)
	defer func() {
		s[0] = 5
	}()
	return s[0]
}
run()`,
			expect: int64(0),
		},
		{
			name: "function_argument_disqualifies",
			source: `func sum(items []int) int {
	total := 0
	for _, value := range items {
		total += value
	}
	return total
}
func run() int {
	s := make([]int, 3)
	s[0] = 1
	s[1] = 2
	s[2] = 3
	return sum(s)
}
run()`,
			expect: int64(6),
		},
		{
			name: "return_disqualifies",
			source: `func makeIt() []int {
	s := make([]int, 2)
	s[0] = 4
	s[1] = 5
	return s
}
func run() int {
	x := makeIt()
	return x[0] + x[1]
}
run()`,
			expect: int64(9),
		},
		{
			name: "empty_typed_slice",
			source: `func run() int {
	s := make([]int, 0)
	return len(s)
}
run()`,
			expect: int64(0),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), testCase.source)
			require.NoError(t, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}
