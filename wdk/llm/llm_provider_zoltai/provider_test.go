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

package llm_provider_zoltai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestComplete(t *testing.T) {
	p, err := newProvider(Config{Seed: 42})
	require.NoError(t, err)

	response, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "zoltai-1", response.Model)
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, llm_dto.RoleAssistant, response.Choices[0].Message.Role)
	assert.NotEmpty(t, response.Choices[0].Message.Content)
	assert.Contains(t, response.Choices[0].Message.Content, "Zoltai")
	assert.Contains(t, response.Choices[0].Message.Content, "\n\n")
	assert.Equal(t, llm_dto.FinishReasonStop, response.Choices[0].FinishReason)
	assert.NotNil(t, response.Usage)
}

func TestCompleteDeterministic(t *testing.T) {
	p1, _ := newProvider(Config{Seed: 123})
	p2, _ := newProvider(Config{Seed: 123})

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Test"},
		},
	}

	r1, _ := p1.Complete(context.Background(), request)
	r2, _ := p2.Complete(context.Background(), request)

	assert.Equal(t, r1.Choices[0].Message.Content, r2.Choices[0].Message.Content)
}

func TestStream(t *testing.T) {
	p, err := newProvider(Config{Seed: 42})
	require.NoError(t, err)

	events, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
	})
	require.NoError(t, err)

	var content strings.Builder
	var doneEvent *llm_dto.StreamEvent

	for ev := range events {
		switch ev.Type {
		case llm_dto.StreamEventChunk:
			if ev.Chunk != nil && ev.Chunk.Delta != nil && ev.Chunk.Delta.Content != nil {
				content.WriteString(*ev.Chunk.Delta.Content)
			}
		case llm_dto.StreamEventDone:
			doneEvent = &ev
		case llm_dto.StreamEventError:
			t.Fatalf("unexpected error: %v", ev.Error)
		}
	}

	streamed := content.String()
	assert.NotEmpty(t, streamed)
	assert.Contains(t, streamed, "Zoltai")
	assert.Contains(t, streamed, "\n\n")
	require.NotNil(t, doneEvent)
	assert.True(t, doneEvent.Done)
}

func TestEmbed(t *testing.T) {
	p, err := newProvider(Config{})
	require.NoError(t, err)

	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"hello world", "foo bar"},
	})
	require.NoError(t, err)

	assert.Equal(t, "zoltai-embed-1", response.Model)
	assert.Len(t, response.Embeddings, 2)
	assert.Len(t, response.Embeddings[0].Vector, 384)
	assert.Len(t, response.Embeddings[1].Vector, 384)
}

func TestEmbedDeterministic(t *testing.T) {
	p1, _ := newProvider(Config{})
	p2, _ := newProvider(Config{})

	r1, _ := p1.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"same text"},
	})
	r2, _ := p2.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"same text"},
	})

	assert.Equal(t, r1.Embeddings[0].Vector, r2.Embeddings[0].Vector)
}

func TestEmbedDifferentInputs(t *testing.T) {
	p, _ := newProvider(Config{})

	response, _ := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"hello", "world"},
	})

	assert.NotEqual(t, response.Embeddings[0].Vector, response.Embeddings[1].Vector)
}

func TestListModels(t *testing.T) {
	p, _ := newProvider(Config{})

	models, err := p.ListModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 1)
	assert.Equal(t, "zoltai-1", models[0].ID)
}

func TestListEmbeddingModels(t *testing.T) {
	p, _ := newProvider(Config{})

	models, err := p.ListEmbeddingModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 1)
	assert.Equal(t, "zoltai-embed-1", models[0].ID)
}

func TestCapabilities(t *testing.T) {
	p, _ := newProvider(Config{})

	assert.True(t, p.SupportsStreaming())
	assert.True(t, p.SupportsStructuredOutput())
	assert.True(t, p.SupportsTools())
}

func TestZoltaiProvider_CapabilityMethods(t *testing.T) {
	p, _ := newProvider(Config{})

	assert.Equal(t, false, p.SupportsPenalties())
	assert.Equal(t, false, p.SupportsSeed())
	assert.Equal(t, false, p.SupportsParallelToolCalls())
	assert.Equal(t, false, p.SupportsMessageName())
}

func TestClose(t *testing.T) {
	p, _ := newProvider(Config{})
	assert.NoError(t, p.Close(context.Background()))
}

func TestConfigWithDefaults(t *testing.T) {
	config := Config{}.WithDefaults()
	assert.Equal(t, "zoltai-1", config.DefaultModel)
	assert.Equal(t, "zoltai-embed-1", config.DefaultEmbeddingModel)
	assert.Equal(t, 384, config.EmbeddingDimensions)
	assert.NotNil(t, config.FormatFortune)
	assert.NotEmpty(t, config.Fortunes)
}

func TestConfigWithDefaultsPreservesValues(t *testing.T) {
	customFormat := func(f string) string { return "custom: " + f }
	customFortunes := []string{"fortune one", "fortune two"}

	config := Config{
		DefaultModel:          "my-model",
		DefaultEmbeddingModel: "my-embed",
		EmbeddingDimensions:   512,
		Fortunes:              customFortunes,
		FormatFortune:         customFormat,
	}.WithDefaults()

	assert.Equal(t, "my-model", config.DefaultModel)
	assert.Equal(t, "my-embed", config.DefaultEmbeddingModel)
	assert.Equal(t, 512, config.EmbeddingDimensions)
	assert.Equal(t, customFortunes, config.Fortunes)
	assert.Equal(t, "custom: test", config.FormatFortune("test"))
}

func TestConfigWithDefaultsNegativeDimensions(t *testing.T) {
	config := Config{EmbeddingDimensions: -1}.WithDefaults()
	assert.Equal(t, 384, config.EmbeddingDimensions)
}

func TestConfigValidate(t *testing.T) {
	assert.NoError(t, (&Config{}).Validate())
	assert.NoError(t, (&Config{DefaultModel: "custom"}).Validate())
}

func TestFormatFortune(t *testing.T) {
	result := formatFortune("test wisdom")
	assert.Contains(t, result, "Zoltai")
	assert.Contains(t, result, "test wisdom")
	assert.Contains(t, result, "\n\n")
}

func TestHashToVector(t *testing.T) {
	vec := hashToVector("hello", 128)
	assert.Len(t, vec, 128)

	vec2 := hashToVector("hello", 128)
	assert.Equal(t, vec, vec2, "same input produces same vector")

	vec3 := hashToVector("world", 128)
	assert.NotEqual(t, vec, vec3, "different input produces different vector")
}

func TestHashToVectorUnitLength(t *testing.T) {
	vec := hashToVector("normalisation test", 256)
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	assert.InDelta(t, 1.0, norm, 0.01, "vector should be unit length")
}

func TestEmbedCustomDimensions(t *testing.T) {
	p, err := newProvider(Config{})
	require.NoError(t, err)

	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input:      []string{"hello"},
		Dimensions: new(64),
	})
	require.NoError(t, err)
	assert.Len(t, response.Embeddings[0].Vector, 64)
}

func TestEmbeddingDimensions(t *testing.T) {
	p, _ := newProvider(Config{EmbeddingDimensions: 512})
	assert.Equal(t, 512, p.EmbeddingDimensions())
}

func TestDefaultModelAccessor(t *testing.T) {
	p, _ := newProvider(Config{DefaultModel: "test-model"})
	assert.Equal(t, "test-model", p.DefaultModel())
}

func TestCompleteWithRequestModel(t *testing.T) {
	p, _ := newProvider(Config{Seed: 42})
	response, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Model: "override-model",
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "override-model", response.Model)
}

func TestEmbedWithRequestModel(t *testing.T) {
	p, _ := newProvider(Config{})
	response, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Model: "custom-embed-model",
		Input: []string{"hello"},
	})
	require.NoError(t, err)
	assert.Equal(t, "custom-embed-model", response.Model)
}

func TestStreamDoneEvent(t *testing.T) {
	p, _ := newProvider(Config{Seed: 42})
	events, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	})
	require.NoError(t, err)

	var chunks int
	var hasDone bool
	for ev := range events {
		switch ev.Type {
		case llm_dto.StreamEventChunk:
			chunks++
		case llm_dto.StreamEventDone:
			hasDone = true
		}
	}
	assert.True(t, hasDone, "should receive a done event")
	assert.Greater(t, chunks, 0, "should receive at least one chunk")
}

func TestCustomFortunes(t *testing.T) {
	p, _ := newProvider(Config{
		Seed:     42,
		Fortunes: []string{"only fortune"},
	})
	response, _ := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	})
	assert.Contains(t, response.Choices[0].Message.Content, "only fortune")
}

func TestCustomFormatFortune(t *testing.T) {
	p, _ := newProvider(Config{
		Seed:          42,
		FormatFortune: func(f string) string { return "PREFIX:" + f },
	})
	response, _ := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	})
	assert.True(t, strings.HasPrefix(response.Choices[0].Message.Content, "PREFIX:"))
}

func TestCustomModel(t *testing.T) {
	p, _ := newProvider(Config{DefaultModel: "zoltai-pro"})

	response, _ := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	})

	assert.Equal(t, "zoltai-pro", response.Model)
}

func TestCompleteWithTools(t *testing.T) {
	p, _ := newProvider(Config{Seed: 42})

	tools := []llm_dto.ToolDefinition{
		llm_dto.NewFunctionTool("get_weather", "Get the weather", &llm_dto.JSONSchema{
			Type: "object",
			Properties: map[string]*llm_dto.JSONSchema{
				"city": {Type: "string"},
			},
			Required: []string{"city"},
		}),
	}

	response, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "What is the weather?"},
		},
		Tools: tools,
	})
	require.NoError(t, err)

	assert.Equal(t, llm_dto.FinishReasonToolCalls, response.Choices[0].FinishReason)
	require.Len(t, response.Choices[0].Message.ToolCalls, 1)

	tc := response.Choices[0].Message.ToolCalls[0]
	assert.Equal(t, "get_weather", tc.Function.Name)
	assert.Equal(t, "{}", tc.Function.Arguments)
	assert.Equal(t, "function", tc.Type)
	assert.NotEmpty(t, tc.ID)
}

func TestCompleteWithToolsPicksFirstTool(t *testing.T) {
	p, _ := newProvider(Config{Seed: 42})

	tools := []llm_dto.ToolDefinition{
		llm_dto.NewFunctionTool("alpha", "First tool", nil),
		llm_dto.NewFunctionTool("beta", "Second tool", nil),
	}

	response, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "test"},
		},
		Tools: tools,
	})
	require.NoError(t, err)
	assert.Equal(t, "alpha", response.Choices[0].Message.ToolCalls[0].Function.Name)
}

func TestCompleteWithStructuredOutputJSONObject(t *testing.T) {
	p, _ := newProvider(Config{Seed: 42, Fortunes: []string{"wisdom"}})

	response, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "test"},
		},
		ResponseFormat: llm_dto.ResponseFormatJSON(),
	})
	require.NoError(t, err)

	var result map[string]string
	require.NoError(t, json.Unmarshal([]byte(response.Content()), &result))
	assert.Contains(t, result["response"], "wisdom")
}

func TestCompleteWithStructuredOutputJSONSchema(t *testing.T) {
	p, _ := newProvider(Config{Seed: 42, Fortunes: []string{"oracle says"}})

	schema := llm_dto.ObjectSchema(map[string]*llm_dto.JSONSchema{
		"message": {Type: "string"},
		"count":   {Type: "integer"},
	}, []string{"message", "count"})

	response, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "test"},
		},
		ResponseFormat: llm_dto.ResponseFormatStructured("test_schema", schema),
	})
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(response.Content()), &result))
	assert.Contains(t, result["message"], "oracle says")
	assert.Equal(t, float64(0), result["count"])
}

func TestStreamWithTools(t *testing.T) {
	p, _ := newProvider(Config{Seed: 42})

	tools := []llm_dto.ToolDefinition{
		llm_dto.NewFunctionTool("lookup", "Look something up", nil),
	}

	events, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "test"},
		},
		Tools: tools,
	})
	require.NoError(t, err)

	var toolCallChunks int
	var doneEvent *llm_dto.StreamEvent

	for ev := range events {
		switch ev.Type {
		case llm_dto.StreamEventChunk:
			if ev.Chunk != nil && ev.Chunk.Delta != nil && len(ev.Chunk.Delta.ToolCalls) > 0 {
				toolCallChunks++
			}
		case llm_dto.StreamEventDone:
			doneEvent = &ev
		case llm_dto.StreamEventError:
			t.Fatalf("unexpected error: %v", ev.Error)
		}
	}

	assert.Greater(t, toolCallChunks, 0)
	require.NotNil(t, doneEvent)
	assert.True(t, doneEvent.Done)
	require.NotNil(t, doneEvent.FinalResponse)
	assert.Equal(t, llm_dto.FinishReasonToolCalls, doneEvent.FinalResponse.Choices[0].FinishReason)
	require.Len(t, doneEvent.FinalResponse.Choices[0].Message.ToolCalls, 1)
	assert.Equal(t, "lookup", doneEvent.FinalResponse.Choices[0].Message.ToolCalls[0].Function.Name)
}
