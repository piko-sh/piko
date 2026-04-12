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
	"math"
	"math/rand"
	"testing"
)

func scalarDotF64(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func scalarSumF64(a []float64) float64 {
	sum := 0.0
	for _, v := range a {
		sum += v
	}
	return sum
}

func TestDotF64Boundaries(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(0x12345678))
	for length := 0; length <= 17; length++ {
		a := make([]float64, length)
		b := make([]float64, length)
		for i := range a {
			a[i] = rng.NormFloat64()
			b[i] = rng.NormFloat64()
		}
		got := DotF64(a, b)
		want := scalarDotF64(a, b)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("len=%d: DotF64 got %v want %v", length, got, want)
		}
	}
}

func TestSumF64Boundaries(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(0xCAFEBABE))
	for length := 0; length <= 17; length++ {
		a := make([]float64, length)
		for i := range a {
			a[i] = rng.NormFloat64()
		}
		got := SumF64(a)
		want := scalarSumF64(a)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("len=%d: SumF64 got %v want %v", length, got, want)
		}
	}
}

func TestAddF64Boundaries(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(0xDEADBEEF))
	for length := 0; length <= 17; length++ {
		a := make([]float64, length)
		b := make([]float64, length)
		dst := make([]float64, length)
		want := make([]float64, length)
		for i := range a {
			a[i] = rng.NormFloat64()
			b[i] = rng.NormFloat64()
			want[i] = a[i] + b[i]
		}
		AddF64(dst, a, b)
		for i := range dst {
			if math.Abs(dst[i]-want[i]) > 1e-12 {
				t.Errorf("len=%d index=%d: AddF64 got %v want %v", length, i, dst[i], want[i])
			}
		}
	}
}

func TestScaleF64Boundaries(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(0xBADF00D))
	for length := 0; length <= 17; length++ {
		original := make([]float64, length)
		for i := range original {
			original[i] = rng.NormFloat64()
		}
		k := rng.NormFloat64() + 1.0
		work := make([]float64, length)
		copy(work, original)
		ScaleF64(work, k)
		for i := range work {
			want := original[i] * k
			if math.Abs(work[i]-want) > 1e-12 {
				t.Errorf("len=%d index=%d: ScaleF64 got %v want %v", length, i, work[i], want)
			}
		}
	}
}
