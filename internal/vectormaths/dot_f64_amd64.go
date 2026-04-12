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

import (
	"golang.org/x/sys/cpu"
)

var (
	// hasAVX2F64 records whether the CPU supports AVX2.
	//
	// Captured at package init so the runtime dispatcher avoids the cpuid query on every
	// kernel call. Shared with the float32 kernels via the equivalent flag in
	// dot_f32_amd64.go; declared separately here only because each kernel pair owns its own
	// runtime-detect flag for build-tag granularity.
	hasAVX2F64 = cpu.X86.HasAVX2
)

// dotF64SSE computes the dot product of two equal-length []float64 slices using SSE2 (2
// doubles per loop iteration). Available on every amd64 CPU; the project's amd64 baseline
// assumes SSE2.
//
// Takes a ([]float64) which is the first input vector.
// Takes b ([]float64) which is the second input vector (must have the same length as a;
// caller verifies before calling).
//
// Returns the scalar dot product sum(a[i]*b[i]).
//
//go:noescape
func dotF64SSE(a, b []float64) float64

// dotF64AVX2 computes the dot product of two equal-length []float64 slices using AVX2 (4
// doubles per loop iteration). Requires Haswell+ on Intel / Excavator+ on AMD; the
// dispatcher in DotF64 falls back to SSE2 on older CPUs.
//
// Takes a ([]float64) which is the first input vector.
// Takes b ([]float64) which is the second input vector.
//
// Returns the scalar dot product sum(a[i]*b[i]).
//
//go:noescape
func dotF64AVX2(a, b []float64) float64

// DotF64 computes the scalar dot product of two []float64 slices.
//
// Dispatches at runtime to the widest SIMD variant the CPU supports. Empirically AVX2 is
// not always the winner on memory-bandwidth-bound workloads (it triggers AVX frequency
// throttling on many Intel parts and adds the SSE/AVX-mode transition penalty), so the
// dispatcher prefers AVX2 only when present.
//
// Takes a, b ([]float64) which are the equal-length operand vectors. Caller is
// responsible for length equality; iteration covers min(len(a), len(b)) only when the
// kernel checks via a_len.
//
// Returns the dot product sum(a[i]*b[i]).
func DotF64(a, b []float64) float64 {
	if hasAVX2F64 {
		return dotF64AVX2(a, b)
	}
	return dotF64SSE(a, b)
}
