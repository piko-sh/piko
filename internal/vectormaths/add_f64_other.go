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

//go:build !safe && !(js && wasm) && !amd64 && !arm64

package vectormaths

// AddF64 writes dst[i] = a[i] + b[i] on architectures piko does not ship hand-written
// SIMD for. Pure Go fallback.
//
// Takes dst ([]float64) which is the destination slice.
// Takes a, b ([]float64) which are the operand slices.
func AddF64(dst, a, b []float64) {
	for i := range dst {
		dst[i] = a[i] + b[i]
	}
}
