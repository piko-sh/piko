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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float32
		eps  float32
	}{
		{
			name: "identical vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
			eps:  1e-6,
		},
		{
			name: "opposite vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{-1, 0, 0},
			want: -1.0,
			eps:  1e-6,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{0, 1, 0},
			want: 0.0,
			eps:  1e-6,
		},
		{
			name: "normalised similar",
			a:    []float32{0.6, 0.8, 0},
			b:    []float32{0.8, 0.6, 0},
			want: 0.96,
			eps:  1e-5,
		},
		{
			name: "different lengths returns zero",
			a:    []float32{1, 0},
			b:    []float32{1, 0, 0},
			want: 0,
			eps:  0,
		},
		{
			name: "empty vectors return zero",
			a:    []float32{},
			b:    []float32{},
			want: 0,
			eps:  0,
		},
		{
			name: "zero vector returns zero",
			a:    []float32{0, 0, 0},
			b:    []float32{1, 2, 3},
			want: 0,
			eps:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CosineSimilarity(tc.a, tc.b)
			assert.InDelta(t, tc.want, got, float64(tc.eps))
		})
	}
}

func TestEuclideanSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float32
		eps  float32
	}{
		{
			name: "identical vectors",
			a:    []float32{1, 2, 3},
			b:    []float32{1, 2, 3},
			want: 1.0,
			eps:  1e-6,
		},
		{
			name: "distant vectors",
			a:    []float32{0, 0, 0},
			b:    []float32{10, 10, 10},
			want: float32(1.0 / (1.0 + math.Sqrt(300))),
			eps:  1e-5,
		},
		{
			name: "unit distance",
			a:    []float32{0, 0},
			b:    []float32{1, 0},
			want: 0.5,
			eps:  1e-6,
		},
		{
			name: "different lengths returns zero",
			a:    []float32{1},
			b:    []float32{1, 2},
			want: 0,
			eps:  0,
		},
		{
			name: "empty vectors return zero",
			a:    []float32{},
			b:    []float32{},
			want: 0,
			eps:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EuclideanSimilarity(tc.a, tc.b)
			assert.InDelta(t, tc.want, got, float64(tc.eps))
		})
	}
}

func TestDotProductSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float32
		eps  float32
	}{
		{
			name: "unit vectors same direction",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
			eps:  1e-6,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0},
			b:    []float32{0, 1},
			want: 0.0,
			eps:  1e-6,
		},
		{
			name: "scaled vectors",
			a:    []float32{2, 3},
			b:    []float32{4, 5},
			want: 23.0,
			eps:  1e-5,
		},
		{
			name: "negative dot product",
			a:    []float32{1, 0},
			b:    []float32{-1, 0},
			want: -1.0,
			eps:  1e-6,
		},
		{
			name: "different lengths returns zero",
			a:    []float32{1, 2, 3},
			b:    []float32{1, 2},
			want: 0,
			eps:  0,
		},
		{
			name: "empty vectors return zero",
			a:    []float32{},
			b:    []float32{},
			want: 0,
			eps:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DotProductSimilarity(tc.a, tc.b)
			assert.InDelta(t, tc.want, got, float64(tc.eps))
		})
	}
}

func TestComputeSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}

	t.Run("cosine metric", func(t *testing.T) {
		got := ComputeSimilarity(a, b, Cosine)
		assert.InDelta(t, 0.0, got, 1e-6)
	})

	t.Run("euclidean metric", func(t *testing.T) {
		got := ComputeSimilarity(a, b, Euclidean)
		expected := float32(1.0 / (1.0 + math.Sqrt(2)))
		assert.InDelta(t, expected, got, 1e-5)
	})

	t.Run("dot_product metric", func(t *testing.T) {
		got := ComputeSimilarity(a, b, DotProduct)
		assert.InDelta(t, 0.0, got, 1e-6)
	})

	t.Run("unknown metric defaults to cosine", func(t *testing.T) {
		got := ComputeSimilarity(a, b, "unknown")
		cosine := CosineSimilarity(a, b)
		assert.Equal(t, cosine, got)
	})
}
