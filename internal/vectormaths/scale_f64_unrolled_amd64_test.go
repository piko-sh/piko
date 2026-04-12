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
	"fmt"
	"math/rand/v2"
	"testing"
)

func TestScaleF64KernelBoundariesAMD64(t *testing.T) {
	t.Parallel()

	kernels := []struct {
		name string
		fn   func(a []float64, k float64)
	}{
		{"SSE", scaleF64SSE},
		{"AVX2", scaleF64AVX2},
	}

	coefficients := []float64{1.7, -0.31, 1.0 / 3.0}

	for _, k := range kernels {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			for _, length := range scaleF64UnrolledBoundaryDims {
				t.Run(fmt.Sprintf("dim=%d", length), func(t *testing.T) {
					t.Parallel()
					rng := rand.New(rand.NewPCG(uint64(length)*0x9E3779B97F4A7C15+7, 0xFEEDFACEDEADC0DE))
					original := make([]float64, length)
					for i := range original {
						original[i] = rng.Float64()*2 - 1
					}
					for _, coefficient := range coefficients {
						work := make([]float64, length)
						copy(work, original)
						k.fn(work, coefficient)
						for i := range work {
							want := original[i] * coefficient
							if work[i] != want {
								t.Fatalf("%s dim=%d k=%v index=%d: got %v want %v",
									k.name, length, coefficient, i, work[i], want)
							}
						}
					}
				})
			}
		})
	}
}
