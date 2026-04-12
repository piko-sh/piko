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

func TestEvalTypedSliceHeaderPropagatesThroughReturn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{

			name: "int64_append_return",
			code: `func appendOne(xs []int64, v int64) []int64 {
	return append(xs, v)
}
xs := []int64{1, 2, 3}
appendOne(xs, 4)[3]`,
			expect: int64(4),
		},
		{

			name: "float64_append_return",
			code: `func appendOne(xs []float64, v float64) []float64 {
	return append(xs, v)
}
xs := []float64{1.0, 2.0, 3.0}
appendOne(xs, 4.0)[3]`,
			expect: float64(4),
		},
		{

			name: "uint64_append_return",
			code: `func appendOne(xs []uint64, v uint64) []uint64 {
	return append(xs, v)
}
xs := []uint64{1, 2, 3}
appendOne(xs, 4)[3]`,
			expect: uint64(4),
		},
		{

			name: "string_append_return",
			code: `func appendOne(xs []string, v string) []string {
	return append(xs, v)
}
xs := []string{"a", "b", "c"}
appendOne(xs, "d")[3]`,
			expect: "d",
		},
		{

			name: "bool_append_return",
			code: `func appendOne(xs []bool, v bool) []bool {
	return append(xs, v)
}
xs := []bool{true, false, true}
appendOne(xs, true)[3]`,
			expect: true,
		},
		{

			name: "byte_append_return",
			code: `func appendOne(xs []byte, v byte) []byte {
	return append(xs, v)
}
xs := []byte{1, 2, 3}
appendOne(xs, 4)[3]`,
			expect: uint64(4),
		},
		{

			name: "int64_generic_append_return",
			code: `func appendOne[T ~int64](xs []T, v T) []T {
	return append(xs, v)
}
xs := []int64{1, 2, 3}
appendOne(xs, 4)[3]`,
			expect: int64(4),
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
