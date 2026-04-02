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

//go:build safe || (js && wasm)

package vectormaths

import "math"

// normaliseF32 normalises a float32 vector in place using pure Go.
// This safe-mode implementation is used when the "safe" build tag is
// active or when compiling for js/wasm.
//
// Takes v ([]float32) which is modified in place to have unit length.
func normaliseF32(v []float32) {
	var sum float32
	for _, x := range v {
		sum += x * x
	}
	if sum == 0 {
		return
	}
	reciprocalSqrt := float32(1.0 / math.Sqrt(float64(sum)))
	for i := range v {
		v[i] *= reciprocalSqrt
	}
}
