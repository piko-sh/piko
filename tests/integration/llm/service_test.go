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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestService_CompleteViaService(t *testing.T) {

	service, ctx := createLLMService(t)

	response, err := service.Complete(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Say hello"},
		},
		MaxTokens: new(500),
	})
	require.NoError(t, err, "service completion")
	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)
	assert.NotEmpty(t, response.Choices[0].Message.Content)
}

func TestService_StreamViaService(t *testing.T) {

	service, ctx := createLLMService(t)

	events, err := service.Stream(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Say goodbye"},
		},
		MaxTokens: new(500),
	})
	require.NoError(t, err, "service stream")

	var chunks []string
	var gotDone bool

	for event := range events {
		switch event.Type {
		case llm_dto.StreamEventChunk:
			if event.Chunk != nil && event.Chunk.Delta != nil && event.Chunk.Delta.Content != nil {
				chunks = append(chunks, *event.Chunk.Delta.Content)
			}
		case llm_dto.StreamEventDone:
			gotDone = true
		case llm_dto.StreamEventError:
			t.Fatalf("unexpected stream error: %v", event.Error)
		}
	}

	assert.True(t, gotDone, "expected done event")
	assert.NotEmpty(t, strings.Join(chunks, ""), "expected stream content")
}

func TestService_EmbedViaService(t *testing.T) {

	service, ctx := createLLMService(t)

	response, err := service.Embed(ctx, &llm_dto.EmbeddingRequest{
		Model: globalEnv.embeddingModel,
		Input: []string{"Integration test embedding via service layer"},
	})
	require.NoError(t, err, "service embedding")
	require.NotNil(t, response)
	require.Len(t, response.Embeddings, 1)
	assert.NotEmpty(t, response.Embeddings[0].Vector)
}

func TestService_EmbedViaBuilder(t *testing.T) {

	service, ctx := createLLMService(t)

	response, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("First text", "Second text").
		Embed(ctx)
	require.NoError(t, err, "builder embedding")
	require.NotNil(t, response)
	require.Len(t, response.Embeddings, 2)
}

func TestService_CompletionBuilder(t *testing.T) {

	service, ctx := createLLMService(t)

	response, err := service.NewCompletion().
		Model(globalEnv.completionModel).
		System("You only reply with 'OK'.").
		User("How are you?").
		MaxTokens(50).
		Do(ctx)
	require.NoError(t, err, "builder completion")
	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)
}

func TestService_ListModels(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	models, err := handle.llm.ListModels(ctx)
	require.NoError(t, err, "listing models")
	require.NotEmpty(t, models, "expected at least one model")

	for _, m := range models {
		assert.NotEmpty(t, m.Name, "model should have a name")
		assert.Equal(t, "ollama", m.Provider)
	}
}

func TestService_ProviderCapabilities(t *testing.T) {

	handle, _ := createOllamaProvider(t)

	assert.True(t, handle.llm.SupportsStreaming(), "ollama supports streaming")
	assert.False(t, handle.llm.SupportsStructuredOutput(), "ollama does not support structured output")
	assert.True(t, handle.llm.SupportsTools(), "ollama supports tools")
}
