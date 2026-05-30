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

//go:build !safe && !(js && wasm)

package vectormaths

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

var (
	euclidSqUnrolledBoundaryDims = []int{
		0, 1, 2, 3, 4, 7, 8, 9, 15, 16, 17,
		31, 32, 33, 63, 64, 65, 127, 128, 129,
		256, 1535, 1536, 1537, 4095, 4096, 4097,
	}
	productionEuclidSqSink float32
)

func TestEuclidSqF32_UnrolledBoundaryDims(t *testing.T) {
	rng := rand.New(rand.NewPCG(0xEC11D, 0x5C1A0))
	for _, n := range euclidSqUnrolledBoundaryDims {
		t.Run(fmt.Sprintf("dim=%d", n), func(t *testing.T) {
			a := make([]float32, n)
			b := make([]float32, n)
			for i := range a {
				a[i] = rng.Float32()*2 - 1
				b[i] = rng.Float32()*2 - 1
			}
			want := scalarEuclidSqF32(a, b)
			got := euclidSqF32Kern(a, b)
			tol := float32(math.Max(1e-4, math.Abs(float64(want))*1e-5))
			if math.Abs(float64(got-want)) > float64(tol) {
				t.Errorf("dim=%d: got %v want %v (tol %v)", n, got, want, tol)
			}
		})
	}
}

func BenchmarkEuclidSqF32_Production(b *testing.B) {
	productionDims := []int{128, 768, 1536, 4096}
	for _, dim := range productionDims {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			rng := rand.New(rand.NewPCG(uint64(dim), uint64(dim)>>1|1))
			a := make([]float32, dim)
			v := make([]float32, dim)
			for i := range a {
				a[i] = rng.Float32()*2 - 1
				v[i] = rng.Float32()*2 - 1
			}
			b.ResetTimer()
			b.SetBytes(int64(dim) * 4 * 2)
			var sink float32
			for range b.N {
				sink = euclidSqF32Kern(a, v)
			}
			productionEuclidSqSink = sink
		})
	}
}
