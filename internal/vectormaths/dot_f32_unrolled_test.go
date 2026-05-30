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
	"math"
	"math/rand/v2"
	"testing"
)

var (
	productionDotBoundaryDims = []int{
		0, 1, 2, 3, 4, 7, 8, 9, 15, 16, 17, 31, 32, 33,
		63, 64, 65, 127, 128, 129, 256, 1535, 1536, 1537,
		4095, 4096, 4097,
	}
	runtimeKeepAliveDotF32 float32
)

func productionDotTolerance(a, b []float32) float64 {
	var sumAbs float64
	for i := range a {
		sumAbs += math.Abs(float64(a[i]) * float64(b[i]))
	}
	dim := len(a)
	noise := math.Sqrt(float64(dim)) * sumAbs * 6e-8
	return math.Max(1e-4, noise)
}

func TestDotF32_ProductionBoundary(t *testing.T) {
	t.Parallel()
	for _, dim := range productionDotBoundaryDims {
		t.Run(fmt.Sprintf("dim=%d", dim), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewPCG(uint64(dim)*1234+1, 5678))
			a := make([]float32, dim)
			b := make([]float32, dim)
			for i := range a {
				a[i] = rng.Float32()*2 - 1
				b[i] = rng.Float32()*2 - 1
			}
			want := scalarDotF32(a, b)
			got := dotF32(a, b)
			tol := productionDotTolerance(a, b)
			if math.Abs(float64(got-want)) > tol {
				t.Fatalf("dim=%d: got %v want %v (tol %v)", dim, got, want, tol)
			}
		})
	}
}

func TestDotF32_ProductionRandomised(t *testing.T) {
	t.Parallel()
	for trial := range 32 {
		dim := productionDotBoundaryDims[trial%len(productionDotBoundaryDims)]
		t.Run(fmt.Sprintf("trial=%d/dim=%d", trial, dim), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewPCG(uint64(trial)*99+1, uint64(dim)+1))
			a := make([]float32, dim)
			b := make([]float32, dim)
			for i := range a {
				a[i] = (rng.Float32()*2 - 1) * 10
				b[i] = (rng.Float32()*2 - 1) * 10
			}
			want := scalarDotF32(a, b)
			got := dotF32(a, b)
			tol := productionDotTolerance(a, b)
			if math.Abs(float64(got-want)) > tol {
				t.Fatalf("trial=%d dim=%d: got %v want %v (tol %v)", trial, dim, got, want, tol)
			}
		})
	}
}

func BenchmarkDotF32_Production(b *testing.B) {
	for _, dim := range []int{128, 768, 1536, 4096} {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			rng := rand.New(rand.NewPCG(42, 7))
			a := make([]float32, dim)
			v := make([]float32, dim)
			for i := range a {
				a[i] = rng.Float32()
				v[i] = rng.Float32()
			}
			b.ResetTimer()
			b.ReportAllocs()
			b.SetBytes(int64(dim) * 4 * 2)
			var sink float32
			for range b.N {
				sink = dotF32(a, v)
			}
			runtimeKeepAliveDotF32 = sink
		})
	}
}
