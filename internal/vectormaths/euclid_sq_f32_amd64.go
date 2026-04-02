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

// euclidSqF32SSE computes the squared Euclidean distance using SSE
// (4 float32s per iteration). Guaranteed on all amd64 CPUs.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the squared Euclidean distance between a and b.
//
//go:noescape
func euclidSqF32SSE(a, b []float32) float32

// euclidSqF32AVX2 computes the squared Euclidean distance using AVX2
// (8 float32s per iteration). Requires AVX2 support.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the squared Euclidean distance between the vectors.
//
//go:noescape
func euclidSqF32AVX2(a, b []float32) float32

// euclidSqF32Kern computes the squared Euclidean distance between two float32
// slices using SIMD. Dispatches to AVX2 or SSE based on runtime CPU detection.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the squared Euclidean distance between the vectors.
func euclidSqF32Kern(a, b []float32) float32 {
	if hasAVX2 {
		return euclidSqF32AVX2(a, b)
	}
	return euclidSqF32SSE(a, b)
}
