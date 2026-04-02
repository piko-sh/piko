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

// normaliseF32SSE normalises a float32 vector in place using SSE
// (4 float32s per iteration). Guaranteed on all amd64 CPUs.
//
// Takes v ([]float32) which is modified in place to have unit length.
//
//go:noescape
func normaliseF32SSE(v []float32)

// normaliseF32AVX2 normalises a float32 vector in place using AVX2
// (8 float32s per iteration). Requires AVX2 support.
//
// Takes v ([]float32) which is modified in place to have unit length.
//
//go:noescape
func normaliseF32AVX2(v []float32)

// normaliseF32 normalises a float32 vector in place using SIMD.
// Dispatches to AVX2 or SSE based on runtime CPU detection.
//
// Takes v ([]float32) which is modified in place to have unit length.
func normaliseF32(v []float32) {
	if hasAVX2 {
		normaliseF32AVX2(v)
		return
	}
	normaliseF32SSE(v)
}
