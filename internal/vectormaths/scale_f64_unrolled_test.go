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
	scaleF64UnrolledBoundaryDims = []int{
		0, 1, 2, 3, 4, 7, 8, 9, 15, 16, 17, 31,
		127, 128, 129,
		1535, 1536, 1537,
		4095, 4096, 4097,
	}
)

func TestScaleF64ProductionBoundaries(t *testing.T) {
	t.Parallel()
	for _, length := range scaleF64UnrolledBoundaryDims {
		t.Run(fmt.Sprintf("dim=%d", length), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewPCG(uint64(length)*0x9E3779B97F4A7C15+1, 0xFEEDFACEDEADC0DE))
			original := make([]float64, length)
			for i := range original {
				original[i] = rng.Float64()*2 - 1
			}
			coefficient := rng.Float64()*2 - 1
			work := make([]float64, length)
			copy(work, original)
			ScaleF64(work, coefficient)
			for i := range work {
				want := original[i] * coefficient
				if math.Abs(work[i]-want) > 1e-12 {
					t.Fatalf("dim=%d index=%d: ScaleF64 got %v want %v",
						length, i, work[i], want)
				}
			}
		})
	}
}

func BenchmarkScaleF64_Production(b *testing.B) {
	for _, dim := range []int{128, 768, 1536, 4096} {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			work := makeF64Bench(dim, 5, 6)
			coefficient := 1.0000001
			b.ResetTimer()
			b.SetBytes(int64(dim) * 8 * 2)
			for range b.N {
				ScaleF64(work, coefficient)
			}
		})
	}
}
