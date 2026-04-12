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
	"math"
	"math/rand/v2"
	"testing"
)

func TestDotF64KernelBoundariesAMD64(t *testing.T) {
	t.Parallel()

	kernels := []struct {
		name string
		fn   func(a, b []float64) float64
	}{
		{"SSE", dotF64SSE},
		{"AVX2", dotF64AVX2},
	}

	for _, k := range kernels {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			for _, length := range dotF64UnrolledBoundaryDims {
				t.Run(fmt.Sprintf("dim=%d", length), func(t *testing.T) {
					t.Parallel()
					rng := rand.New(rand.NewPCG(uint64(length)*0x9E3779B97F4A7C15+3, 0xDEADBEEFCAFEBABE))
					a := make([]float64, length)
					b := make([]float64, length)
					for i := range a {
						a[i] = rng.Float64()*2 - 1
						b[i] = rng.Float64()*2 - 1
					}
					want := kahanDotF64(a, b)
					got := k.fn(a, b)
					tolerance := math.Max(1e-9, math.Abs(want)*1e-12)
					if math.Abs(got-want) > tolerance {
						t.Fatalf("%s dim=%d: got %v want %v (diff=%g, tol=%g)",
							k.name, length, got, want, math.Abs(got-want), tolerance)
					}
				})
			}
		})
	}
}
