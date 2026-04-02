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

//go:build integration

package llm_integration_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestEmbed_SingleInput(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	response, err := handle.embedding.Embed(ctx, &llm_dto.EmbeddingRequest{
		Model: globalEnv.embeddingModel,
		Input: []string{"Hello world"},
	})
	require.NoError(t, err, "embedding request")

	require.NotNil(t, response)
	require.Len(t, response.Embeddings, 1, "expected one embedding")
	assert.Equal(t, 0, response.Embeddings[0].Index)
	assert.NotEmpty(t, response.Embeddings[0].Vector, "expected non-empty vector")
	assert.Equal(t, globalEnv.embeddingModel, response.Model)
}

func TestEmbed_MultipleInputs(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	inputs := []string{
		"The cat sat on the mat",
		"Dogs are friendly animals",
		"Machine learning is fascinating",
	}

	response, err := handle.embedding.Embed(ctx, &llm_dto.EmbeddingRequest{
		Model: globalEnv.embeddingModel,
		Input: inputs,
	})
	require.NoError(t, err, "batch embedding request")

	require.NotNil(t, response)
	require.Len(t, response.Embeddings, len(inputs), "expected one embedding per input")

	dim := len(response.Embeddings[0].Vector)
	for i, emb := range response.Embeddings {
		assert.Equal(t, i, emb.Index, "embedding index mismatch")
		assert.Len(t, emb.Vector, dim, "all embeddings should have same dimension")
	}
}

func TestEmbed_SimilarTextsSimilarVectors(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	response, err := handle.embedding.Embed(ctx, &llm_dto.EmbeddingRequest{
		Model: globalEnv.embeddingModel,
		Input: []string{
			"The weather is sunny today",
			"It is a beautiful sunny day",
			"Quantum computing uses qubits for calculation",
		},
	})
	require.NoError(t, err)
	require.Len(t, response.Embeddings, 3)

	simSimilar := cosineSimilarity(response.Embeddings[0].Vector, response.Embeddings[1].Vector)
	simDifferent := cosineSimilarity(response.Embeddings[0].Vector, response.Embeddings[2].Vector)

	assert.Greater(t, simSimilar, simDifferent,
		"similar texts should have higher cosine similarity (%.4f) than dissimilar texts (%.4f)",
		simSimilar, simDifferent,
	)
}

func TestEmbed_UsageReported(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	response, err := handle.embedding.Embed(ctx, &llm_dto.EmbeddingRequest{
		Model: globalEnv.embeddingModel,
		Input: []string{"Test input for usage tracking"},
	})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Usage, "expected usage stats")
	assert.Greater(t, response.Usage.PromptTokens, 0)
}

func TestEmbed_VectorDimensionConsistency(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	resp1, err := handle.embedding.Embed(ctx, &llm_dto.EmbeddingRequest{
		Model: globalEnv.embeddingModel,
		Input: []string{"First request"},
	})
	require.NoError(t, err)

	resp2, err := handle.embedding.Embed(ctx, &llm_dto.EmbeddingRequest{
		Model: globalEnv.embeddingModel,
		Input: []string{"Completely different second request"},
	})
	require.NoError(t, err)

	assert.Equal(t,
		len(resp1.Embeddings[0].Vector),
		len(resp2.Embeddings[0].Vector),
		"embedding dimensions should be consistent across requests",
	)
}

func cosineSimilarity(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
