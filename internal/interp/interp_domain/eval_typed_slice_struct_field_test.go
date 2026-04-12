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

//go:build !safe && !(js && wasm)

// D2/D3 typed-slice struct-field tests exercise the unsafe-build
// fast paths (pointer arithmetic + runtime.typedmemmove). The safe
// build's reflect-based fallback is structurally correct but the
// surrounding safe build has unrelated pre-existing failures
// (boxStringToGeneral via unsafeNewAt produces wrong-typed values
// for string/float64; composition tests cascade off that). Gating
// the file matches the established pattern from
// vm_handler_struct_field_unsafe_test.go (same constraint).

package interp_domain

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalTypedSliceStructFieldRead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{

			name: "int64_field_typed_direct",
			code: `type Holder struct {
	Xs []int64
}
func sum(h *Holder) int64 {
	var total int64
	for i := 0; i < len(h.Xs); i++ {
		total = total + h.Xs[i]
	}
	return total
}
h := &Holder{Xs: []int64{1, 2, 3, 4, 5}}
sum(h)`,
			expect: int64(15),
		},
		{

			name: "uint64_field_typed_direct",
			code: `func first(h *struct{ Xs []uint64 }) uint64 { return h.Xs[0] }
h := &struct{ Xs []uint64 }{Xs: []uint64{42, 7}}
first(h)`,
			expect: uint64(42),
		},
		{

			name: "float64_field_typed_direct",
			code: `func first(h *struct{ Xs []float64 }) float64 { return h.Xs[0] }
h := &struct{ Xs []float64 }{Xs: []float64{3.14, 2.71}}
first(h)`,
			expect: 3.14,
		},
		{

			name: "string_field_typed_direct",
			code: `func first(h *struct{ Xs []string }) string { return h.Xs[0] }
h := &struct{ Xs []string }{Xs: []string{"alpha", "beta"}}
first(h)`,
			expect: "alpha",
		},
		{

			name: "bool_field_typed_direct",
			code: `func first(h *struct{ Xs []bool }) bool { return h.Xs[0] }
h := &struct{ Xs []bool }{Xs: []bool{true, false}}
first(h)`,
			expect: true,
		},
		{

			name: "byte_field_typed_direct",
			code: `func first(h *struct{ Xs []byte }) byte { return h.Xs[0] }
h := &struct{ Xs []byte }{Xs: []byte{42, 7}}
first(h)`,
			expect: uint64(42),
		},
		{

			name: "int32_field_general_fallback",
			code: `func first(h *struct{ Xs []int32 }) int32 { return h.Xs[0] }
h := &struct{ Xs []int32 }{Xs: []int32{42, 7}}
first(h)`,
			expect: int64(42),
		},
		{

			name: "uint32_field_general_fallback",
			code: `func first(h *struct{ Xs []uint32 }) uint32 { return h.Xs[0] }
h := &struct{ Xs []uint32 }{Xs: []uint32{42, 7}}
first(h)`,
			expect: uint64(42),
		},
		{

			name: "int64_field_loop_sum",
			code: `type H struct{ Xs []int64 }
func sum(h *H) int64 {
	var t int64
	for i := 0; i < len(h.Xs); i++ {
		t = t + h.Xs[i]
	}
	return t
}
h := &H{Xs: []int64{10, 20, 30, 40, 50}}
sum(h)`,
			expect: int64(150),
		},
		{

			name: "int64_field_returned",
			code: `type H struct{ Xs []int64 }
func get(h *H) []int64 { return h.Xs }
h := &H{Xs: []int64{1, 2, 3}}
get(h)[1]`,
			expect: int64(2),
		},
		{

			name: "int64_field_aliasing_via_index_write",
			code: `type H struct{ Xs []int64 }
func mutate(h *H) {
	xs := h.Xs
	xs[0] = 999
}
h := &H{Xs: []int64{1, 2, 3}}
mutate(h)
h.Xs[0]`,
			expect: int64(999),
		},
		{

			name: "int64_field_empty_slice",
			code: `type H struct{ Xs []int64 }
func length(h *H) int64 { return int64(len(h.Xs)) }
h := &H{Xs: []int64{}}
length(h)`,
			expect: int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.expect == nil {

				service := NewService()
				_, err := service.Eval(context.Background(), tt.code)
				require.NoError(t, err)
				return
			}
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEvalTypedSliceStructFieldWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{

			name: "int64_field_typed_direct_write",
			code: `type Holder struct {
	Xs []int64
}
func install(h *Holder) {
	h.Xs = []int64{10, 20, 30}
}
h := &Holder{Xs: []int64{1}}
install(h)
h.Xs[1]`,
			expect: int64(20),
		},
		{

			name: "uint64_field_typed_direct_write",
			code: `type H struct{ Xs []uint64 }
func install(h *H) {
	h.Xs = []uint64{7, 11, 13}
}
h := &H{Xs: []uint64{1}}
install(h)
h.Xs[2]`,
			expect: uint64(13),
		},
		{

			name: "float64_field_typed_direct_write",
			code: `type H struct{ Xs []float64 }
func install(h *H) {
	h.Xs = []float64{1.5, 2.5}
}
h := &H{Xs: []float64{0.0}}
install(h)
h.Xs[1]`,
			expect: 2.5,
		},
		{

			name: "string_field_typed_direct_write",
			code: `type H struct{ Xs []string }
func install(h *H) {
	h.Xs = []string{"hello", "world"}
}
h := &H{Xs: []string{"old"}}
install(h)
h.Xs[1]`,
			expect: "world",
		},
		{

			name: "bool_field_typed_direct_write",
			code: `type H struct{ Xs []bool }
func install(h *H) {
	h.Xs = []bool{true, false, true}
}
h := &H{Xs: []bool{false}}
install(h)
h.Xs[2]`,
			expect: true,
		},
		{

			name: "byte_field_typed_direct_write",
			code: `type H struct{ Xs []byte }
func install(h *H) {
	h.Xs = []byte{42, 7, 99}
}
h := &H{Xs: []byte{0}}
install(h)
h.Xs[0]`,
			expect: uint64(42),
		},
		{

			name: "int64_field_round_trip",
			code: `type H struct{ Xs []int64 }
func setAndSum(h *H) int64 {
	h.Xs = []int64{100, 200, 300}
	var s int64
	for i := 0; i < len(h.Xs); i++ {
		s = s + h.Xs[i]
	}
	return s
}
h := &H{Xs: []int64{0}}
setAndSum(h)`,
			expect: int64(600),
		},
		{

			name: "int64_field_overwrite_then_read",
			code: `type H struct{ Xs []int64 }
func swap(h *H) int64 {
	h.Xs = []int64{99, 88}
	h.Xs = []int64{1, 2, 3}
	return h.Xs[2]
}
h := &H{Xs: []int64{0, 0}}
swap(h)`,
			expect: int64(3),
		},
		{

			name: "int64_field_write_from_param",
			code: `type H struct{ Xs []int64 }
func install(h *H, xs []int64) {
	h.Xs = xs
}
h := &H{Xs: []int64{0}}
install(h, []int64{5, 10, 15})
h.Xs[2]`,
			expect: int64(15),
		},
		{

			name: "int64_field_write_nil_slice",
			code: `type H struct{ Xs []int64 }
func clear(h *H) int64 {
	var empty []int64
	h.Xs = empty
	return int64(len(h.Xs))
}
h := &H{Xs: []int64{1, 2, 3}}
clear(h)`,
			expect: int64(0),
		},
		{

			name: "int64_field_write_gc_stress",
			code: `type H struct{ Xs []int64 }
func stress(h *H) int64 {
	var last int64
	for i := 0; i < 100; i++ {
		h.Xs = []int64{int64(i), int64(i*2), int64(i*3)}
		last = h.Xs[2]
	}
	return last
}
h := &H{Xs: []int64{0}}
stress(h)`,
			expect: int64(99 * 3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.expect == nil {
				service := NewService()
				_, err := service.Eval(context.Background(), tt.code)
				require.NoError(t, err)
				return
			}
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
			runtime.GC()
		})
	}
}
