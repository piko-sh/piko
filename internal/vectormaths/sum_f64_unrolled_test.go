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
	sumF64ProductionBenchDims = []int{128, 768, 1536, 4096}
	sumF64ProductionSink float64
)

func kahanSumF64(a []float64) float64 {
	var sum, comp float64
	for _, v := range a {
		y := v - comp
		t := sum + y
		comp = (t - sum) - y
		sum = t
	}
	return sum
}

func makeSumF64Input(n int, seed1, seed2 uint64) []float64 {
	rng := rand.New(rand.NewPCG(seed1, seed2))
	out := make([]float64, n)
	for i := range out {
		out[i] = rng.Float64()*2 - 1
	}
	return out
}

func TestSumF64ProductionBoundaries(t *testing.T) {
	t.Parallel()

	dims := []int{
		0, 1, 2, 3, 4, 7, 8, 9,
		15, 16, 17, 31, 32, 33,
		127, 128, 129, 255, 256,
		1535, 1536, 1537,
		4095, 4096, 4097,
	}

	for _, n := range dims {
		t.Run(fmt.Sprintf("dim=%d", n), func(t *testing.T) {
			t.Parallel()

			a := makeSumF64Input(n, uint64(n)*0x9E3779B97F4A7C15+1, 0xBADF00DCAFEBABE)
			want := kahanSumF64(a)
			got := SumF64(a)
			tol := math.Max(1e-9, math.Abs(want)*1e-12)
			if math.Abs(got-want) > tol {
				t.Errorf("SumF64 dim=%d: got %v want %v (diff=%g, tol=%g)",
					n, got, want, math.Abs(got-want), tol)
			}
		})
	}
}

func BenchmarkSumF64_Production(b *testing.B) {
	for _, dim := range sumF64ProductionBenchDims {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			a := makeSumF64Input(dim, 1, 2)
			b.ResetTimer()
			b.SetBytes(int64(dim) * 8)
			var sink float64
			for range b.N {
				sink = SumF64(a)
			}
			sumF64ProductionSink = sink
		})
	}
}
