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

package vectormaths

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

var (
	addF64UnrolledBoundaryDims = []int{
		0, 1, 2, 3, 4, 7, 8, 9, 15, 16, 17, 31,
		127, 128, 129, 255, 256,
		1535, 1536, 1537,
		4095, 4096, 4097,
	}
)

func TestAddF64ProductionBoundaries(t *testing.T) {
	t.Parallel()
	for _, length := range addF64UnrolledBoundaryDims {
		t.Run(fmt.Sprintf("dim=%d", length), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewPCG(uint64(length)*0x9E3779B97F4A7C15, 0xCAFEF00DBEEFFACE))
			a := make([]float64, length)
			b := make([]float64, length)
			want := make([]float64, length)
			for i := range a {
				a[i] = rng.Float64()*2 - 1
				b[i] = rng.Float64()*2 - 1
				want[i] = a[i] + b[i]
			}
			dst := make([]float64, length)
			AddF64(dst, a, b)
			for i := range dst {
				if dst[i] != want[i] {
					t.Fatalf("dim=%d index=%d: AddF64 got %v want %v", length, i, dst[i], want[i])
				}
			}
		})
	}
}

func BenchmarkAddF64_Production(b *testing.B) {
	for _, dim := range []int{128, 768, 1536, 4096} {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			a := makeF64Bench(dim, 1, 2)
			c := makeF64Bench(dim, 3, 4)
			dst := make([]float64, dim)
			b.ResetTimer()
			b.SetBytes(int64(dim) * 8 * 3)
			for range b.N {
				AddF64(dst, a, c)
			}
		})
	}
}

func makeF64Bench(n int, seed1, seed2 uint64) []float64 {
	rng := rand.New(rand.NewPCG(seed1, seed2))
	out := make([]float64, n)
	for i := range out {
		out[i] = rng.Float64()*2 - 1
	}
	return out
}
