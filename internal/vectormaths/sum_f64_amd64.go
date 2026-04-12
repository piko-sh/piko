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

// sumF64SSE computes the sum of a []float64 using SSE2 (2 doubles per loop iteration).
// Available on every amd64 CPU.
//
// Takes a ([]float64) which is the input vector.
//
// Returns the scalar sum(a[i]).
//
//go:noescape
func sumF64SSE(a []float64) float64

// sumF64AVX2 computes the sum of a []float64 using AVX2 (4 doubles per loop iteration).
// Requires Haswell+ on Intel / Excavator+ on AMD; the dispatcher in SumF64 falls back to
// SSE2 on older CPUs.
//
// Takes a ([]float64) which is the input vector.
//
// Returns the scalar sum(a[i]).
//
//go:noescape
func sumF64AVX2(a []float64) float64

// SumF64 reduces a []float64 to its sum, dispatching to the widest SIMD variant the CPU
// supports. Same SSE-vs-AVX2 caveat as DotF64: AVX2 frequency throttling and SSE/AVX
// transition penalties can make SSE2 the better choice on some workloads, so the
// dispatcher keeps SSE2 as the guaranteed fast path.
//
// Takes a ([]float64) which is the operand slice.
//
// Returns sum(a[i]).
func SumF64(a []float64) float64 {
	if hasAVX2F64 {
		return sumF64AVX2(a)
	}
	return sumF64SSE(a)
}
