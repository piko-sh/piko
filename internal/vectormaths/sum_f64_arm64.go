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

// sumF64Kern computes sum(a[i]) using NEON with eight parallel accumulators.
//
// Uses V0 and V4-V10 as 128-bit accumulators (2 doubles each, 16 doubles per main
// iteration). A 2-wide sub-loop and a scalar tail consume residuals when len(a) % 16 !=
// 0. Multiple parallel accumulators break the FADD latency chain so the load + add ports
// saturate rather than the single-accumulator critical path.
//
// Takes a ([]float64) which is the input vector.
//
// Returns the scalar sum(a[i]).
//
//go:noescape
func sumF64Kern(a []float64) float64

// SumF64 reduces a []float64 to its sum on arm64 via the 8-way unrolled NEON kernel. The
// base ARMv8-A NEON unit is required on every arm64 target Go supports, so no runtime
// feature dispatch is needed.
//
// Takes a ([]float64) which is the operand slice.
//
// Returns sum(a[i]).
func SumF64(a []float64) float64 {
	return sumF64Kern(a)
}
