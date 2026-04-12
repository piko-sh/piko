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
	dotF64UnrolledBoundaryDims = []int{
		0, 1, 2, 3, 4, 7, 8, 9, 15, 16, 17, 31,
		127, 128, 129,
		1535, 1536, 1537,
		4095, 4096, 4097,
	}
	dotF64ProductionSink float64
)

func TestDotF64ProductionBoundaries(t *testing.T) {
	t.Parallel()
	for _, length := range dotF64UnrolledBoundaryDims {
		t.Run(fmt.Sprintf("dim=%d", length), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewPCG(uint64(length)*0x9E3779B97F4A7C15+1, 0xDEADBEEFCAFEBABE))
			a := make([]float64, length)
			b := make([]float64, length)
			for i := range a {
				a[i] = rng.Float64()*2 - 1
				b[i] = rng.Float64()*2 - 1
			}
			got := DotF64(a, b)
			want := kahanDotF64(a, b)
			tolerance := math.Max(1e-9, math.Abs(want)*1e-12)
			if math.Abs(got-want) > tolerance {
				t.Fatalf("dim=%d: DotF64 got %v want %v (diff=%g, tol=%g)",
					length, got, want, math.Abs(got-want), tolerance)
			}
		})
	}
}

func kahanDotF64(a, b []float64) float64 {
	var sum, comp float64
	for i := range a {
		y := a[i]*b[i] - comp
		t := sum + y
		comp = (t - sum) - y
		sum = t
	}
	return sum
}

func BenchmarkDotF64_Production(b *testing.B) {
	for _, dim := range []int{128, 768, 1536, 4096} {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			x := makeF64Bench(dim, 1, 2)
			y := makeF64Bench(dim, 3, 4)
			b.ResetTimer()
			b.SetBytes(int64(dim) * 8 * 2)
			var sink float64
			for range b.N {
				sink = DotF64(x, y)
			}
			dotF64ProductionSink = sink
		})
	}
}
