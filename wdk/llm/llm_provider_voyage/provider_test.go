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

package llm_provider_voyage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

func newTestProvider(t *testing.T) *voyageProvider {
	t.Helper()
	config := Config{
		APIKey: "test-api-key",
	}
	provider, err := NewVoyageProvider(config)
	require.NoError(t, err)
	return provider.voyageProvider
}

func newTestProviderWithServer(t *testing.T, server *httptest.Server) *voyageProvider {
	t.Helper()
	config := Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}
	provider, err := NewVoyageProvider(config)
	require.NoError(t, err)
	return provider.voyageProvider
}

func TestEmbed_BasicRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var reqBody voyageEmbedRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "voyage-3.5", reqBody.Model)
		assert.Equal(t, []string{"hello world"}, reqBody.Input)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(voyageEmbedResponse{
			Model: "voyage-3.5",
			Data: []voyageEmbedData{
				{Embedding: []float64{0.1, 0.2, 0.3}, Index: 0},
			},
			Usage: &voyageUsage{TotalTokens: 5},
		})
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"hello world"},
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "voyage-3.5", response.Model)
	require.Len(t, response.Embeddings, 1)
	assert.Equal(t, 0, response.Embeddings[0].Index)
	assert.InDelta(t, float32(0.1), response.Embeddings[0].Vector[0], 1e-6)
	assert.InDelta(t, float32(0.2), response.Embeddings[0].Vector[1], 1e-6)
	assert.InDelta(t, float32(0.3), response.Embeddings[0].Vector[2], 1e-6)
	require.NotNil(t, response.Usage)
	assert.Equal(t, 5, response.Usage.TotalTokens)
}

func TestEmbed_WithDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody voyageEmbedRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		require.NotNil(t, reqBody.OutputDimension)
		assert.Equal(t, 512, *reqBody.OutputDimension)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(voyageEmbedResponse{
			Model: "voyage-3.5",
			Data: []voyageEmbedData{
				{Embedding: []float64{0.1, 0.2}, Index: 0},
			},
		})
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input:      []string{"test"},
		Dimensions: new(512),
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Embeddings, 1)
}

func TestEmbed_WithInputType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody voyageEmbedRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "query", reqBody.InputType)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(voyageEmbedResponse{
			Model: "voyage-3.5",
			Data: []voyageEmbedData{
				{Embedding: []float64{0.5}, Index: 0},
			},
		})
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"search query"},
		ProviderOptions: map[string]any{
			"input_type": "query",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestEmbed_DefaultModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody voyageEmbedRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "voyage-3.5", reqBody.Model)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(voyageEmbedResponse{
			Model: "voyage-3.5",
			Data: []voyageEmbedData{
				{Embedding: []float64{0.1}, Index: 0},
			},
		})
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"test"},
	})

	require.NoError(t, err)
	assert.Equal(t, "voyage-3.5", response.Model)
}

func TestEmbed_CustomModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody voyageEmbedRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "voyage-code-3", reqBody.Model)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(voyageEmbedResponse{
			Model: "voyage-code-3",
			Data: []voyageEmbedData{
				{Embedding: []float64{0.1}, Index: 0},
			},
		})
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Model: "voyage-code-3",
		Input: []string{"func main()"},
	})

	require.NoError(t, err)
	assert.Equal(t, "voyage-code-3", response.Model)
}

func TestEmbed_MultipleInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody voyageEmbedRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		require.Len(t, reqBody.Input, 3)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(voyageEmbedResponse{
			Model: "voyage-3.5",
			Data: []voyageEmbedData{
				{Embedding: []float64{0.1, 0.2}, Index: 0},
				{Embedding: []float64{0.3, 0.4}, Index: 1},
				{Embedding: []float64{0.5, 0.6}, Index: 2},
			},
			Usage: &voyageUsage{TotalTokens: 15},
		})
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"first", "second", "third"},
	})

	require.NoError(t, err)
	require.Len(t, response.Embeddings, 3)
	assert.Equal(t, 0, response.Embeddings[0].Index)
	assert.Equal(t, 1, response.Embeddings[1].Index)
	assert.Equal(t, 2, response.Embeddings[2].Index)
	assert.InDelta(t, float32(0.1), response.Embeddings[0].Vector[0], 1e-6)
	assert.InDelta(t, float32(0.3), response.Embeddings[1].Vector[0], 1e-6)
	assert.InDelta(t, float32(0.5), response.Embeddings[2].Vector[0], 1e-6)
	assert.Equal(t, 15, response.Usage.TotalTokens)
}

func TestEmbed_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"detail":"Invalid API key"}`)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"test"},
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "voyage embedding API error")
	assert.Contains(t, err.Error(), "401")
}

func TestEmbed_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"detail":"Internal server error"}`)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"test"},
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "500")
}

func TestEmbed_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{invalid json`)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"test"},
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to decode embedding response")
}

func TestEmbed_NoUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(voyageEmbedResponse{
			Model: "voyage-3.5",
			Data: []voyageEmbedData{
				{Embedding: []float64{0.1}, Index: 0},
			},
		})
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"test"},
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Nil(t, response.Usage)
}

func TestEmbed_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(voyageEmbedResponse{
			Model: "voyage-3.5",
			Data: []voyageEmbedData{
				{Embedding: []float64{0.1}, Index: 0},
			},
		})
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	response, err := p.Embed(ctx, &llm_dto.EmbeddingRequest{
		Input: []string{"test"},
	})

	require.Error(t, err)
	assert.Nil(t, response)
}

func TestListEmbeddingModels(t *testing.T) {
	p := newTestProvider(t)
	models, err := p.ListEmbeddingModels(context.Background())

	require.NoError(t, err)
	require.Len(t, models, 9)

	for _, model := range models {
		assert.Equal(t, "voyage", model.Provider)
		assert.Equal(t, model.ID, model.Name)
		assert.NotEmpty(t, model.ID)
	}

	ids := make(map[string]bool, len(models))
	for _, model := range models {
		ids[model.ID] = true
	}
	assert.True(t, ids["voyage-3.5"])
	assert.True(t, ids["voyage-4"])
	assert.True(t, ids["voyage-4-large"])
	assert.True(t, ids["voyage-code-3"])
}

func TestEmbeddingDimensions(t *testing.T) {
	p := newTestProvider(t)
	assert.Equal(t, 1024, p.EmbeddingDimensions())
}

func TestClose(t *testing.T) {
	p := newTestProvider(t)
	err := p.Close(context.Background())
	assert.NoError(t, err)
}
