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

import "math"

// Metric specifies how to measure similarity between vectors.
type Metric string

const (
	// Cosine uses cosine similarity. Values range from -1 to 1, where 1 means
	// the vectors point in the same direction.
	Cosine Metric = "cosine"

	// Euclidean uses Euclidean distance converted to a similarity score in
	// [0, 1] where 1 means identical vectors.
	Euclidean Metric = "euclidean"

	// DotProduct uses the raw dot product. Higher values indicate greater
	// similarity.
	DotProduct Metric = "dot_product"
)

// ComputeSimilarity calculates the similarity between two vectors using the
// specified metric.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
// Takes metric (Metric) which selects the similarity method.
//
// Returns float32 which is the similarity score using the chosen metric.
func ComputeSimilarity(a, b []float32, metric Metric) float32 {
	switch metric {
	case Euclidean:
		return EuclideanSimilarity(a, b)
	case DotProduct:
		return DotProductSimilarity(a, b)
	default:
		return CosineSimilarity(a, b)
	}
}

// CosineSimilarity calculates the cosine similarity between two vectors.
//
// Takes a ([]float32) which is the first vector to compare.
// Takes b ([]float32) which is the second vector to compare.
//
// Returns float32 which is a value in [-1, 1] where 1 indicates identical
// direction, 0 indicates orthogonal vectors, and -1 indicates opposite
// direction. Returns 0 when vectors have different lengths or are empty.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	dot := dotF32(a, b)
	normA := dotF32(a, a)
	normB := dotF32(b, b)

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / float32(math.Sqrt(float64(normA)*float64(normB)))
}

// EuclideanSimilarity converts Euclidean distance to a similarity score.
//
// Takes a ([]float32) which is the first vector to compare.
// Takes b ([]float32) which is the second vector to compare.
//
// Returns float32 which is a value in [0, 1] where 1 indicates identical
// vectors. Returns 0 when vectors have different lengths or are empty.
func EuclideanSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	sumSq := euclidSqF32Kern(a, b)
	distance := math.Sqrt(float64(sumSq))
	return float32(1.0 / (1.0 + distance))
}

// EuclidSqF32 computes the squared Euclidean distance between two float32
// vectors. Using squared distance preserves ordering while avoiding the
// sqrt call, making it suitable for nearest-neighbour comparisons.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the sum of squared differences, or 0 if the
// vectors have different lengths or are empty.
func EuclidSqF32(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	return euclidSqF32Kern(a, b)
}

// DotProductSimilarity calculates the dot product between two vectors.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the dot product result, or zero if the vectors
// have different lengths or are empty.
func DotProductSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	return dotF32(a, b)
}

// NormaliseF32 normalises a float32 vector in place, dividing each
// element by the vector's Euclidean magnitude so that the result has
// unit length (magnitude of 1.0). If the vector is zero (all elements
// are zero), it is left unchanged to avoid division by zero.
//
// This is useful for pre-normalising vectors before cosine similarity
// comparisons, which then reduce to a single dot product.
//
// Takes v ([]float32) which is modified in place.
func NormaliseF32(v []float32) {
	if len(v) == 0 {
		return
	}
	normaliseF32(v)
}

// NormF32 returns the L2 (Euclidean) magnitude of a float32 vector.
// This is the square root of the sum of squared elements.
//
// Takes v ([]float32) which is the input vector.
//
// Returns float32 which is the L2 norm, or 0 if the vector is empty.
func NormF32(v []float32) float32 {
	if len(v) == 0 {
		return 0
	}
	return float32(math.Sqrt(float64(dotF32(v, v))))
}
