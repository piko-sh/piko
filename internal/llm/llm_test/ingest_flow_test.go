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

package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

type mockLoader struct {
	docs []llm_domain.Document
}

func (m *mockLoader) Load(_ context.Context) ([]llm_domain.Document, error) {
	return m.docs, nil
}

func TestLLM_IngestFlow(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	mockEmb := llm_domain.NewMockEmbeddingProvider()
	h.RegisterEmbeddingProvider("mock-emb", mockEmb)
	require.NoError(t, h.service.SetDefaultEmbeddingProvider("mock-emb"))

	loader := &mockLoader{
		docs: []llm_domain.Document{
			{ID: "doc1", Content: "This is a long document that needs splitting. It has multiple sentences."},
		},
	}

	splitter, err := llm_domain.NewRecursiveCharacterSplitter(20, 5)
	require.NoError(t, err)

	err = h.service.NewIngest("kb").
		Loader(loader).
		Splitter(splitter).
		Do(ctx)

	require.NoError(t, err)

	vStore := h.service.GetVectorStore()
	require.NotNil(t, vStore)

	searchResp, err := vStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "kb",
		Vector:    []float32{0.1, 0.2, 0.3},
		TopK:      10,
	})

	require.NoError(t, err)
	assert.True(t, searchResp.TotalCount > 1, "Should have multiple chunks")

	foundChunk := false
	for _, searchResult := range searchResp.Results {
		if assert.Contains(t, searchResult.ID, "doc1-chunk-") {
			foundChunk = true
			break
		}
	}
	assert.True(t, foundChunk, "Should have found IDs with chunk suffix")
}

func TestLLM_AddText_Convenience(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	mockEmb := llm_domain.NewMockEmbeddingProvider()
	h.RegisterEmbeddingProvider("mock-emb", mockEmb)
	require.NoError(t, h.service.SetDefaultEmbeddingProvider("mock-emb"))

	err := h.service.AddText(ctx, "quick-ns", "text1", "Hello Piko")
	require.NoError(t, err)

	vStore := h.service.GetVectorStore()
	document, err := vStore.Get(ctx, "quick-ns", "text1")
	require.NoError(t, err)
	require.NotNil(t, document)
	assert.Equal(t, "Hello Piko", document.Content)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, document.Vector)
}
