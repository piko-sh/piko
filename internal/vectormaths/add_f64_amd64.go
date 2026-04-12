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

//go:build !safe && !(js && wasm) && amd64

package vectormaths

// addF64SSE computes dst[i] = a[i] + b[i] using SSE2 (2 doubles per iteration). All three
// slices must have the same length.
//
// Takes dst ([]float64) which is the destination, mutated in place.
// Takes a, b ([]float64) which are the operand slices.
//
//go:noescape
func addF64SSE(dst, a, b []float64)

// addF64AVX2 computes dst[i] = a[i] + b[i] using AVX2 (4 doubles per iteration).
//
// Takes dst ([]float64) which is the destination, mutated in place.
// Takes a, b ([]float64) which are the operand slices.
//
//go:noescape
func addF64AVX2(dst, a, b []float64)

// AddF64 writes dst[i] = a[i] + b[i] for every i, dispatching to the widest SIMD variant
// the CPU supports. Caller must ensure len(dst) == len(a) == len(b); the kernel iterates
// len(dst) elements.
//
// Takes dst ([]float64) which is the destination slice.
// Takes a, b ([]float64) which are the operand slices.
func AddF64(dst, a, b []float64) {
	if hasAVX2F64 {
		addF64AVX2(dst, a, b)
		return
	}
	addF64SSE(dst, a, b)
}
