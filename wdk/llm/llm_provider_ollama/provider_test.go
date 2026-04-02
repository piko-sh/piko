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

package llm_provider_ollama

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func newTestProvider(t *testing.T) *ollamaProvider {
	t.Helper()
	return &ollamaProvider{
		defaultModel:          Model("llama3.2"),
		defaultEmbeddingModel: Model("all-minilm"),
		config:                Config{}.WithDefaults(),
	}
}

func TestBuildChatRequest_BasicMessage(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello, world!"},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	assert.Equal(t, "llama3.2", chatRequest.Model)
	require.Len(t, chatRequest.Messages, 1)
	assert.Equal(t, "user", chatRequest.Messages[0].Role)
	assert.Equal(t, "Hello, world!", chatRequest.Messages[0].Content)
	assert.Nil(t, chatRequest.Options)
}

func TestBuildChatRequest_SystemMessage(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "You are a helpful assistant."},
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 2)
	assert.Equal(t, "system", chatRequest.Messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", chatRequest.Messages[0].Content)
	assert.Equal(t, "user", chatRequest.Messages[1].Role)
	assert.Equal(t, "Hi", chatRequest.Messages[1].Content)
}

func TestBuildChatRequest_MultipleMessages(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "Be concise."},
			{Role: llm_dto.RoleUser, Content: "What is Go?"},
			{Role: llm_dto.RoleAssistant, Content: "A programming language."},
			{Role: llm_dto.RoleUser, Content: "Tell me more."},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 4)
	assert.Equal(t, "system", chatRequest.Messages[0].Role)
	assert.Equal(t, "user", chatRequest.Messages[1].Role)
	assert.Equal(t, "assistant", chatRequest.Messages[2].Role)
	assert.Equal(t, "user", chatRequest.Messages[3].Role)
	assert.Equal(t, "Tell me more.", chatRequest.Messages[3].Content)
}

func TestBuildChatRequest_WithTemperature(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		Temperature: new(0.7),
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, 0.7, chatRequest.Options["temperature"])
}

func TestBuildChatRequest_WithMaxTokens(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		MaxTokens: new(256),
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, 256, chatRequest.Options["num_predict"])
}

func TestBuildChatRequest_WithStopSequences(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		Stop: []string{"END", "STOP"},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, []string{"END", "STOP"}, chatRequest.Options["stop"])
}

func TestBuildChatRequest_WithTopP(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		TopP: new(0.9),
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, 0.9, chatRequest.Options["top_p"])
}

func TestBuildChatRequest_WithSeed(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		Seed: new(int64(42)),
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, int64(42), chatRequest.Options["seed"])
}

func TestBuildChatRequest_WithFrequencyPenalty(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		FrequencyPenalty: new(0.5),
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, 0.5, chatRequest.Options["frequency_penalty"])
}

func TestBuildChatRequest_WithPresencePenalty(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		PresencePenalty: new(0.3),
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, 0.3, chatRequest.Options["presence_penalty"])
}

func TestBuildChatRequest_WithProviderOptions(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		ProviderOptions: map[string]any{
			"mirostat":      1,
			"num_ctx":       4096,
			"repeat_last_n": 64,
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, 1, chatRequest.Options["mirostat"])
	assert.Equal(t, 4096, chatRequest.Options["num_ctx"])
	assert.Equal(t, 64, chatRequest.Options["repeat_last_n"])
}

func TestBuildChatRequest_ProviderOptionsOverrideKnownOptions(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		Temperature: new(0.7),
		ProviderOptions: map[string]any{
			"temperature": 0.9,
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, 0.9, chatRequest.Options["temperature"])
}

func TestBuildChatRequest_WithAllOptions(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		Temperature:      new(0.5),
		TopP:             new(0.8),
		MaxTokens:        new(100),
		Stop:             []string{"<end>"},
		Seed:             new(int64(7)),
		FrequencyPenalty: new(0.4),
		PresencePenalty:  new(0.6),
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "custom-model")

	assert.Equal(t, "custom-model", chatRequest.Model)
	require.NotNil(t, chatRequest.Options)
	assert.Equal(t, 0.5, chatRequest.Options["temperature"])
	assert.Equal(t, 0.8, chatRequest.Options["top_p"])
	assert.Equal(t, 100, chatRequest.Options["num_predict"])
	assert.Equal(t, []string{"<end>"}, chatRequest.Options["stop"])
	assert.Equal(t, int64(7), chatRequest.Options["seed"])
	assert.Equal(t, 0.4, chatRequest.Options["frequency_penalty"])
	assert.Equal(t, 0.6, chatRequest.Options["presence_penalty"])
}

func TestBuildChatRequest_NoOptions(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	assert.Nil(t, chatRequest.Options)
}

func TestBuildChatRequest_EmptyMessages(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	assert.Empty(t, chatRequest.Messages)
}

func TestBuildChatRequest_JSONResponseFormat(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages:       []llm_dto.Message{{Role: "user", Content: "test"}},
		ResponseFormat: llm_dto.ResponseFormatJSON(),
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.NotNil(t, chatRequest.Format)
	assert.Equal(t, `"json"`, string(chatRequest.Format))
}

func TestBuildChatRequest_NoResponseFormat(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: "user", Content: "test"}},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	assert.Nil(t, chatRequest.Format)
}

func TestConvertChatResponse_TextResponse(t *testing.T) {
	p := newTestProvider(t)

	response := &api.ChatResponse{
		Message: api.Message{
			Role:    "assistant",
			Content: "Hello! How can I help you?",
		},
	}

	result := p.convertChatResponse(response, "llama3.2")

	assert.Equal(t, "llama3.2", result.Model)
	require.Len(t, result.Choices, 1)
	assert.Equal(t, 0, result.Choices[0].Index)
	assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
	assert.Equal(t, "Hello! How can I help you?", result.Choices[0].Message.Content)
	assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)
}

func TestConvertChatResponse_WithUsage(t *testing.T) {
	p := newTestProvider(t)

	response := &api.ChatResponse{
		Message: api.Message{
			Role:    "assistant",
			Content: "The answer is 42.",
		},
	}
	response.PromptEvalCount = 15
	response.EvalCount = 8

	result := p.convertChatResponse(response, "llama3.2")

	require.NotNil(t, result.Usage)
	assert.Equal(t, 15, result.Usage.PromptTokens)
	assert.Equal(t, 8, result.Usage.CompletionTokens)
	assert.Equal(t, 23, result.Usage.TotalTokens)
}

func TestConvertChatResponse_EmptyContent(t *testing.T) {
	p := newTestProvider(t)

	response := &api.ChatResponse{
		Message: api.Message{
			Role:    "assistant",
			Content: "",
		},
	}

	result := p.convertChatResponse(response, "llama3.2")

	require.Len(t, result.Choices, 1)
	assert.Empty(t, result.Choices[0].Message.Content)
	assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
	assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)
}

func TestConvertChatResponse_ZeroUsage(t *testing.T) {
	p := newTestProvider(t)

	response := &api.ChatResponse{
		Message: api.Message{
			Role:    "assistant",
			Content: "Hello",
		},
	}

	result := p.convertChatResponse(response, "llama3.2")

	require.NotNil(t, result.Usage)
	assert.Equal(t, 0, result.Usage.PromptTokens)
	assert.Equal(t, 0, result.Usage.CompletionTokens)
	assert.Equal(t, 0, result.Usage.TotalTokens)
}

func TestConvertChatResponse_PreservesModel(t *testing.T) {
	p := newTestProvider(t)

	response := &api.ChatResponse{
		Message: api.Message{
			Role:    "assistant",
			Content: "Hi",
		},
	}

	result := p.convertChatResponse(response, "mistral:7b")

	assert.Equal(t, "mistral:7b", result.Model)
}

func TestSupportsStreaming(t *testing.T) {
	p := newTestProvider(t)

	assert.True(t, p.SupportsStreaming())
}

func TestSupportsTools(t *testing.T) {
	p := newTestProvider(t)

	assert.True(t, p.SupportsTools())
}

func TestSupportsStructuredOutput(t *testing.T) {
	p := newTestProvider(t)

	assert.False(t, p.SupportsStructuredOutput())
}

func TestBuildChatRequest_WithTools(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "What is the weather?"},
		},
		Tools: []llm_dto.ToolDefinition{
			{
				Type: "function",
				Function: llm_dto.FunctionDefinition{
					Name:        "get_weather",
					Description: new("Get the weather for a location"),
					Parameters: &llm_dto.JSONSchema{
						Type: "object",
						Properties: map[string]*llm_dto.JSONSchema{
							"location": {Type: "string"},
						},
						Required: []string{"location"},
					},
				},
			},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Tools, 1)
	assert.Equal(t, "function", chatRequest.Tools[0].Type)
	assert.Equal(t, "get_weather", chatRequest.Tools[0].Function.Name)
	assert.Equal(t, "Get the weather for a location", chatRequest.Tools[0].Function.Description)
	assert.Equal(t, []string{"location"}, chatRequest.Tools[0].Function.Parameters.Required)

	prop, ok := chatRequest.Tools[0].Function.Parameters.Properties.Get("location")
	require.True(t, ok)
	assert.Equal(t, api.PropertyType{"string"}, prop.Type)
}

func TestBuildChatRequest_WithToolCallMessages(t *testing.T) {
	p := newTestProvider(t)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "What is the weather?"},
			{
				Role:    llm_dto.RoleAssistant,
				Content: "",
				ToolCalls: []llm_dto.ToolCall{
					{
						ID:   "call_123",
						Type: "function",
						Function: llm_dto.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"London"}`,
						},
					},
				},
			},
			{
				Role:       llm_dto.RoleTool,
				Content:    "Sunny, 22C",
				ToolCallID: new("call_123"),
			},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 3)

	assistantMessage := chatRequest.Messages[1]
	require.Len(t, assistantMessage.ToolCalls, 1)
	assert.Equal(t, "call_123", assistantMessage.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", assistantMessage.ToolCalls[0].Function.Name)

	toolMessage := chatRequest.Messages[2]
	assert.Equal(t, "tool", toolMessage.Role)
	assert.Equal(t, "Sunny, 22C", toolMessage.Content)
	assert.Equal(t, "call_123", toolMessage.ToolCallID)
}

func TestConvertChatResponse_WithToolCalls(t *testing.T) {
	p := newTestProvider(t)

	arguments := api.NewToolCallFunctionArguments()
	arguments.Set("location", "London")

	response := &api.ChatResponse{
		Message: api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{
					ID: "call_456",
					Function: api.ToolCallFunction{
						Name:      "get_weather",
						Arguments: arguments,
					},
				},
			},
		},
	}

	result := p.convertChatResponse(response, "llama3.2")

	require.Len(t, result.Choices, 1)
	assert.Equal(t, llm_dto.FinishReasonToolCalls, result.Choices[0].FinishReason)
	require.Len(t, result.Choices[0].Message.ToolCalls, 1)

	tc := result.Choices[0].Message.ToolCalls[0]
	assert.Equal(t, "call_456", tc.ID)
	assert.Equal(t, "function", tc.Type)
	assert.Equal(t, "get_weather", tc.Function.Name)
	assert.Equal(t, `{"location":"London"}`, tc.Function.Arguments)
}

func TestBuildChatRequest_ContentParts_TextOnly(t *testing.T) {
	p := newTestProvider(t)
	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("hello world"),
				},
			},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 1)
	assert.Equal(t, "hello world", chatRequest.Messages[0].Content)
	assert.Empty(t, chatRequest.Messages[0].Images)
}

func TestBuildChatRequest_ContentParts_ImageData(t *testing.T) {
	p := newTestProvider(t)
	b64Data := base64.StdEncoding.EncodeToString([]byte("fake-image-bytes"))
	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("describe this"),
					llm_dto.ImageDataPart("image/png", b64Data),
				},
			},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 1)
	assert.Equal(t, "describe this", chatRequest.Messages[0].Content)
	require.Len(t, chatRequest.Messages[0].Images, 1)
	assert.Equal(t, api.ImageData([]byte("fake-image-bytes")), chatRequest.Messages[0].Images[0])
}

func TestBuildChatRequest_ContentParts_ImageURL_NoFetcher(t *testing.T) {
	p := newTestProvider(t)
	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("describe this"),
					llm_dto.ImageURLPart("https://example.com/photo.jpg"),
				},
			},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 1)
	assert.Equal(t, "describe this", chatRequest.Messages[0].Content)
	assert.Empty(t, chatRequest.Messages[0].Images)
}

func TestBuildChatRequest_ContentParts_ImageURL_WithFetcher(t *testing.T) {
	imageBytes := []byte("server-image-data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBytes)
	}))
	defer server.Close()

	p := &ollamaProvider{
		defaultModel:          Model("llama3.2"),
		defaultEmbeddingModel: Model("all-minilm"),
		config: Config{
			ImageFetch: &ImageFetchConfig{
				MaxBytes: defaultImageFetchMaxBytes,
				Timeout:  5 * time.Second,
			},
		}.WithDefaults(),
		imageFetcher: &http.Client{Timeout: 5 * time.Second},
	}

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("describe this"),
					llm_dto.ImageURLPart(server.URL + "/photo.png"),
				},
			},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 1)
	assert.Equal(t, "describe this", chatRequest.Messages[0].Content)
	require.Len(t, chatRequest.Messages[0].Images, 1)
	assert.Equal(t, api.ImageData(imageBytes), chatRequest.Messages[0].Images[0])
}

func TestBuildChatRequest_ContentParts_Mixed(t *testing.T) {
	p := newTestProvider(t)
	b64Data := base64.StdEncoding.EncodeToString([]byte("img-data"))
	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("first"),
					llm_dto.TextPart("second"),
					llm_dto.ImageDataPart("image/jpeg", b64Data),
					llm_dto.ImageURLPart("https://example.com/ignored.jpg"),
				},
			},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 1)
	assert.Equal(t, "firstsecond", chatRequest.Messages[0].Content)
	require.Len(t, chatRequest.Messages[0].Images, 1)
	assert.Equal(t, api.ImageData([]byte("img-data")), chatRequest.Messages[0].Images[0])
}

func TestBuildChatRequest_ContentParts_Priority(t *testing.T) {
	p := newTestProvider(t)
	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "this should be ignored",
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("this should be used"),
				},
			},
		},
	}

	chatRequest := p.buildChatRequest(context.Background(), request, "llama3.2")

	require.Len(t, chatRequest.Messages, 1)
	assert.Equal(t, "this should be used", chatRequest.Messages[0].Content)
}

func TestFetchImage_SizeLimit(t *testing.T) {
	largeData := make([]byte, 1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(largeData)
	}))
	defer server.Close()

	p := &ollamaProvider{
		config: Config{
			ImageFetch: &ImageFetchConfig{
				MaxBytes: 512,
				Timeout:  5 * time.Second,
			},
		},
		imageFetcher: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := p.fetchImage(context.Background(), server.URL+"/large.png")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestOllamaProvider_CapabilityMethods(t *testing.T) {
	p := newTestProvider(t)

	assert.Equal(t, true, p.SupportsPenalties())
	assert.Equal(t, true, p.SupportsSeed())
	assert.Equal(t, false, p.SupportsParallelToolCalls())
	assert.Equal(t, false, p.SupportsMessageName())
}
