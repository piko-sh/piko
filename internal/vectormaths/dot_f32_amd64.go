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

import "golang.org/x/sys/cpu"

// hasAVX2 holds whether the CPU supports AVX2 instructions
// for SIMD dot product acceleration.
var hasAVX2 = cpu.X86.HasAVX2

// dotF32SSE computes the dot product using SSE (4 float32s per iteration).
// Guaranteed on all amd64 CPUs.
//
// Takes a ([]float32) which is the first input vector.
// Takes b ([]float32) which is the second input vector.
//
// Returns float32 which is the dot product of the two vectors.
//
//go:noescape
func dotF32SSE(a, b []float32) float32

// dotF32AVX2 computes the dot product using AVX2 (8 float32s per iteration).
// Requires AVX2 support (Haswell+ / Excavator+).
//
// Takes a ([]float32) which is the first input vector.
// Takes b ([]float32) which is the second input vector.
//
// Returns float32 which is the dot product of the two vectors.
//
//go:noescape
func dotF32AVX2(a, b []float32) float32

// dotF32 computes the dot product of two float32 slices using SIMD.
// Dispatches to AVX2 (8-wide) or SSE (4-wide) based on runtime CPU detection.
//
// Takes a ([]float32) which is the first input vector.
// Takes b ([]float32) which is the second input vector.
//
// Returns float32 which is the sum of element-wise products.
func dotF32(a, b []float32) float32 {
	if hasAVX2 {
		return dotF32AVX2(a, b)
	}
	return dotF32SSE(a, b)
}
