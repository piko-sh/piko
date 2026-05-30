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

var (
	normaliseF32KernelBoundaryDimsAMD64 = []int{
		0, 1, 2, 3, 4, 7, 8, 9, 15, 16, 17, 31,
		127, 128, 129,
		1535, 1536, 1537,
	}
)

func TestNormaliseF32KernelBoundariesAMD64(t *testing.T) {
	t.Parallel()

	kernels := []struct {
		name string
		fn   func(v []float32)
	}{
		{"SSE", normaliseF32SSE},
		{"AVX2", normaliseF32AVX2},
	}

	for _, k := range kernels {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			for _, length := range normaliseF32KernelBoundaryDimsAMD64 {
				t.Run(fmt.Sprintf("dim=%d", length), func(t *testing.T) {
					t.Parallel()
					rng := rand.New(rand.NewPCG(uint64(length)*0x9E3779B97F4A7C15+5, 0xCAFEF00DBEEFFACE))
					original := make([]float32, length)
					for i := range original {
						original[i] = rng.Float32()*2 - 1
					}
					want := make([]float32, length)
					copy(want, original)
					scalarNormaliseF32(want)
					got := make([]float32, length)
					copy(got, original)
					k.fn(got)
					for i := range got {
						if want[i] == 0 {
							if math.Abs(float64(got[i])) > 1e-6 {
								t.Fatalf("%s dim=%d index=%d: got %v want ~0",
									k.name, length, i, got[i])
							}
							continue
						}
						relErr := math.Abs(float64(got[i]-want[i])) / math.Abs(float64(want[i]))
						if relErr > 1e-4 {
							t.Fatalf("%s dim=%d index=%d: got %v want %v relErr=%v",
								k.name, length, i, got[i], want[i], relErr)
						}
					}
				})
			}
		})
	}
}
