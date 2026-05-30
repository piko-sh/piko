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

// ScaleF64 multiplies every element of a by k in place. Pure Go fallback selected by the
// safe build tag or the js+wasm build.
//
// Takes a ([]float64) which is mutated in place.
// Takes k (float64) which is the scalar coefficient.
func ScaleF64(a []float64, k float64) {
	for i := range a {
		a[i] *= k
	}
}
