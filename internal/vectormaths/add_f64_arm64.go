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

//go:build !safe && !(js && wasm) && arm64

package vectormaths

// addF64NEON computes dst[i] = a[i] + b[i] using a 4-way unrolled NEON kernel (8 doubles
// per main-loop iteration). All three slices must have the same length.
//
// Takes dst ([]float64) which is the destination, mutated in place.
// Takes a, b ([]float64) which are the operand slices.
//
//go:noescape
func addF64NEON(dst, a, b []float64)

// AddF64 writes dst[i] = a[i] + b[i] for every i, using the 4-way unrolled NEON kernel.
// Caller must ensure len(dst) == len(a) == len(b); the kernel iterates len(dst) elements.
//
// Takes dst ([]float64) which is the destination slice.
// Takes a, b ([]float64) which are the operand slices.
func AddF64(dst, a, b []float64) {
	addF64NEON(dst, a, b)
}
