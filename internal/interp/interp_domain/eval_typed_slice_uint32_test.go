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

func TestEvalSubIntSliceCallBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{

			name: "uint32_index_read",
			code: `func first(xs []uint32) uint32 {
	return xs[0]
}
xs := []uint32{42, 7}
first(xs)`,
			expect: uint64(42),
		},
		{

			name: "int32_sum_pair",
			code: `func sumPair(xs []int32) int32 {
	return xs[0] + xs[1]
}
xs := []int32{40, 2}
sumPair(xs)`,
			expect: int64(42),
		},
		{

			name: "uint16_pass_through_two_hops",
			code: `func b(xs []uint16) []uint16 {
	return xs
}
func a(xs []uint16) []uint16 {
	return b(xs)
}
xs := []uint16{1, 2, 3, 4, 5}
a(xs)[3]`,
			expect: uint64(4),
		},
		{

			name: "int16_sum_loop",
			code: `func sumAll(xs []int16) int16 {
	var total int16
	for i := 0; i < len(xs); i++ {
		total = total + xs[i]
	}
	return total
}
xs := []int16{10, 20, 30}
sumAll(xs)`,
			expect: int64(60),
		},
		{

			name: "uint32_generic_sum",
			code: `func gsum[T ~uint32](xs []T) T {
	var total T
	for i := 0; i < len(xs); i++ {
		total = total + xs[i]
	}
	return total
}
xs := []uint32{100, 200, 300}
gsum(xs)`,
			expect: uint64(600),
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

func TestEvalSubIntSliceMutationPropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "int32_mutate_first_propagates",
			code: `func setFirst(xs []int32, v int32) {
	xs[0] = v
}
xs := []int32{1, 2, 3}
setFirst(xs, 99)
xs[0]`,
			expect: int64(99),
		},
		{
			name: "uint32_mutate_index_propagates",
			code: `func setAt(xs []uint32, i int, v uint32) {
	xs[i] = v
}
xs := []uint32{10, 20, 30, 40}
setAt(xs, 2, 777)
xs[2]`,
			expect: uint64(777),
		},
		{
			name: "int16_mutate_propagates",
			code: `func write(xs []int16, v int16) {
	xs[0] = v
}
xs := []int16{5, 6, 7}
write(xs, 42)
xs[0]`,
			expect: int64(42),
		},
		{
			name: "uint16_mutate_propagates",
			code: `func write(xs []uint16, v uint16) {
	xs[0] = v
}
xs := []uint16{1, 2, 3}
write(xs, 65000)
xs[0]`,
			expect: uint64(65000),
		},
		{
			name: "int32_two_callee_mutations_both_visible",
			code: `func setLoHi(xs []int32, lo, hi int32) {
	xs[0] = lo
	xs[len(xs)-1] = hi
}
xs := []int32{0, 0, 0, 0}
setLoHi(xs, -7, 7)
xs[0]*100 + xs[3]`,
			expect: int64(-700 + 7),
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
