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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scalarDotF32(a, b []float32) float32 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return float32(sum)
}

func scalarEuclidSqF32(a, b []float32) float32 {
	var sum float64
	for i := range a {
		diff := float64(a[i]) - float64(b[i])
		sum += diff * diff
	}
	return float32(sum)
}

func TestDotF32(t *testing.T) {
	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float32
		eps  float64
	}{
		{
			name: "empty",
			a:    []float32{},
			b:    []float32{},
			want: 0,
			eps:  0,
		},
		{
			name: "single element",
			a:    []float32{3},
			b:    []float32{4},
			want: 12,
			eps:  1e-6,
		},
		{
			name: "three elements (tail only)",
			a:    []float32{1, 2, 3},
			b:    []float32{4, 5, 6},
			want: 32,
			eps:  1e-5,
		},
		{
			name: "four elements (exact SIMD width)",
			a:    []float32{1, 2, 3, 4},
			b:    []float32{5, 6, 7, 8},
			want: 70,
			eps:  1e-5,
		},
		{
			name: "five elements (one SIMD + one tail)",
			a:    []float32{1, 2, 3, 4, 5},
			b:    []float32{6, 7, 8, 9, 10},
			want: 130,
			eps:  1e-4,
		},
		{
			name: "eight elements (two SIMD iterations)",
			a:    []float32{1, 1, 1, 1, 1, 1, 1, 1},
			b:    []float32{2, 2, 2, 2, 2, 2, 2, 2},
			want: 16,
			eps:  1e-6,
		},
		{
			name: "negative values",
			a:    []float32{-1, 2, -3, 4},
			b:    []float32{4, -3, 2, -1},
			want: -20,
			eps:  1e-5,
		},
		{
			name: "self-dot (norm squared)",
			a:    []float32{3, 4},
			b:    []float32{3, 4},
			want: 25,
			eps:  1e-6,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dotF32(tc.a, tc.b)
			assert.InDelta(t, tc.want, got, tc.eps)
		})
	}
}

func TestEuclidSqF32(t *testing.T) {
	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float32
		eps  float64
	}{
		{
			name: "empty",
			a:    []float32{},
			b:    []float32{},
			want: 0,
			eps:  0,
		},
		{
			name: "identical",
			a:    []float32{1, 2, 3},
			b:    []float32{1, 2, 3},
			want: 0,
			eps:  1e-6,
		},
		{
			name: "unit distance",
			a:    []float32{0, 0},
			b:    []float32{1, 0},
			want: 1,
			eps:  1e-6,
		},
		{
			name: "four elements (exact SIMD width)",
			a:    []float32{1, 2, 3, 4},
			b:    []float32{5, 6, 7, 8},
			want: 64,
			eps:  1e-5,
		},
		{
			name: "five elements (SIMD + tail)",
			a:    []float32{0, 0, 0, 0, 0},
			b:    []float32{3, 4, 0, 0, 5},
			want: 50,
			eps:  1e-5,
		},
		{
			name: "negative differences",
			a:    []float32{10, 20, 30},
			b:    []float32{7, 16, 25},
			want: 34,
			eps:  1e-5,
		},
	}

	tests[5].want = 50

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := euclidSqF32Kern(tc.a, tc.b)
			assert.InDelta(t, tc.want, got, tc.eps)
		})
	}
}

func TestDotF32_CrossVerify(t *testing.T) {
	randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
	sizes := []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 128, 255, 256, 768, 1536}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			a := randomVector(randomSource, n)
			b := randomVector(randomSource, n)

			got := dotF32(a, b)
			want := scalarDotF32(a, b)

			if want == 0 {
				assert.Equal(t, want, got)
				return
			}

			relErr := math.Abs(float64(got-want)) / math.Abs(float64(want))
			require.Less(t, relErr, 1e-4,
				"n=%d: got=%v want=%v relErr=%v", n, got, want, relErr)
		})
	}
}

func TestEuclidSqF32_CrossVerify(t *testing.T) {
	randomSource := rand.New(rand.NewPCG(99, 99>>1|1))
	sizes := []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 128, 255, 256, 768, 1536}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			a := randomVector(randomSource, n)
			b := randomVector(randomSource, n)

			got := euclidSqF32Kern(a, b)
			want := scalarEuclidSqF32(a, b)

			if want == 0 {
				assert.Equal(t, want, got)
				return
			}

			relErr := math.Abs(float64(got-want)) / math.Abs(float64(want))
			require.Less(t, relErr, 1e-4,
				"n=%d: got=%v want=%v relErr=%v", n, got, want, relErr)
		})
	}
}

func TestEuclidSqF32_Exported(t *testing.T) {
	t.Run("matching vectors", func(t *testing.T) {
		got := EuclidSqF32([]float32{1, 2, 3}, []float32{4, 5, 6})
		assert.InDelta(t, 27.0, got, 1e-5)
	})

	t.Run("different lengths returns zero", func(t *testing.T) {
		got := EuclidSqF32([]float32{1, 2}, []float32{1})
		assert.Equal(t, float32(0), got)
	})

	t.Run("empty returns zero", func(t *testing.T) {
		got := EuclidSqF32([]float32{}, []float32{})
		assert.Equal(t, float32(0), got)
	})
}

func BenchmarkDotF32(b *testing.B) {
	dims := []int{3, 128, 768, 1536}
	for _, dim := range dims {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
			a := randomVector(randomSource, dim)
			v := randomVector(randomSource, dim)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = dotF32(a, v)
			}
		})
	}
}

func BenchmarkEuclidSqF32(b *testing.B) {
	dims := []int{3, 128, 768, 1536}
	for _, dim := range dims {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
			a := randomVector(randomSource, dim)
			v := randomVector(randomSource, dim)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = euclidSqF32Kern(a, v)
			}
		})
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	dims := []int{3, 128, 768, 1536}
	for _, dim := range dims {
		b.Run(fmt.Sprintf("dim=%d", dim), func(b *testing.B) {
			randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
			a := randomVector(randomSource, dim)
			v := randomVector(randomSource, dim)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = CosineSimilarity(a, v)
			}
		})
	}
}

func TestNormaliseF32(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		v := []float32{}
		NormaliseF32(v)
		assert.Empty(t, v)
	})

	t.Run("single_element", func(t *testing.T) {
		v := []float32{5.0}
		NormaliseF32(v)
		assert.InDelta(t, 1.0, v[0], 1e-6)
	})

	t.Run("zero_vector", func(t *testing.T) {
		v := []float32{0, 0, 0, 0}
		NormaliseF32(v)
		for _, x := range v {
			assert.Equal(t, float32(0), x)
		}
	})

	t.Run("unit_vector", func(t *testing.T) {
		v := []float32{1, 0, 0, 0}
		NormaliseF32(v)
		assert.InDelta(t, 1.0, v[0], 1e-6)
		assert.InDelta(t, 0.0, v[1], 1e-6)
		assert.InDelta(t, 0.0, v[2], 1e-6)
		assert.InDelta(t, 0.0, v[3], 1e-6)
	})

	t.Run("known_result_3_4", func(t *testing.T) {
		v := []float32{3, 4}
		NormaliseF32(v)
		assert.InDelta(t, 0.6, v[0], 1e-6)
		assert.InDelta(t, 0.8, v[1], 1e-6)
	})

	t.Run("negative_values", func(t *testing.T) {
		v := []float32{-3, -4}
		NormaliseF32(v)
		assert.InDelta(t, -0.6, v[0], 1e-6)
		assert.InDelta(t, -0.8, v[1], 1e-6)
	})

	sizes := []int{4, 5, 7, 8, 9, 15, 16, 17, 31, 32, 33}
	for _, n := range sizes {
		t.Run(fmt.Sprintf("size_%d_magnitude_is_one", n), func(t *testing.T) {
			randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
			v := randomVector(randomSource, n)
			NormaliseF32(v)
			magnitude := NormF32(v)
			assert.InDelta(t, 1.0, magnitude, 1e-4,
				"n=%d: magnitude after normalisation should be ~1.0, got %v", n, magnitude)
		})
	}
}

func TestNormaliseF32_CrossVerify(t *testing.T) {
	randomSource := rand.New(rand.NewPCG(99, 99>>1|1))
	sizes := []int{1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 128, 255, 256, 768, 1536}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			simdVector := randomVector(randomSource, n)
			referenceVector := make([]float32, n)
			copy(referenceVector, simdVector)

			NormaliseF32(simdVector)
			scalarNormaliseF32(referenceVector)

			for i := range simdVector {
				if referenceVector[i] == 0 {
					assert.InDelta(t, 0.0, simdVector[i], 1e-6,
						"n=%d i=%d", n, i)
					continue
				}
				relErr := math.Abs(float64(simdVector[i]-referenceVector[i])) / math.Abs(float64(referenceVector[i]))
				require.Less(t, relErr, 1e-4,
					"n=%d i=%d: got=%v want=%v relErr=%v", n, i, simdVector[i], referenceVector[i], relErr)
			}
		})
	}
}

func TestNormF32(t *testing.T) {
	t.Run("3_4_triangle", func(t *testing.T) {
		assert.InDelta(t, 5.0, NormF32([]float32{3, 4}), 1e-6)
	})
	t.Run("zero_vector", func(t *testing.T) {
		assert.Equal(t, float32(0), NormF32([]float32{0, 0, 0}))
	})
	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, float32(0), NormF32(nil))
	})
	t.Run("unit", func(t *testing.T) {
		assert.InDelta(t, 1.0, NormF32([]float32{1, 0, 0}), 1e-6)
	})
}

func BenchmarkNormaliseF32(b *testing.B) {
	dims := []int{3, 128, 768, 1536}
	for _, dim := range dims {
		b.Run(fmt.Sprintf("simd/dim=%d", dim), func(b *testing.B) {
			randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
			v := randomVector(randomSource, dim)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				NormaliseF32(v)
			}
		})

		b.Run(fmt.Sprintf("scalar/dim=%d", dim), func(b *testing.B) {
			randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
			v := randomVector(randomSource, dim)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				scalarNormaliseF32(v)
			}
		})
	}
}

func scalarNormaliseF32(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return
	}
	reciprocalSqrt := 1.0 / math.Sqrt(sum)
	for i := range v {
		v[i] = float32(float64(v[i]) * reciprocalSqrt)
	}
}

func randomVector(randomSource *rand.Rand, n int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = randomSource.Float32()*2 - 1
	}
	return v
}
