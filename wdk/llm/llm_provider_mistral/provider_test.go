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

package llm_provider_mistral

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

func contentString(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var s string
	require.NoError(t, json.Unmarshal(raw, &s))
	return s
}

func newTestProvider(t *testing.T) *mistralProvider {
	t.Helper()
	config := Config{
		APIKey: "test-api-key",
	}
	provider, err := New(config)
	require.NoError(t, err)
	mp, ok := provider.(*mistralProvider)
	require.True(t, ok, "expected provider to be *mistralProvider")
	return mp
}

func TestMistralProvider_ConvertMessages(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts system message", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:    llm_dto.RoleSystem,
				Content: "You are a helpful assistant.",
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		assert.Equal(t, "system", result[0].Role)
		assert.Equal(t, "You are a helpful assistant.", contentString(t, result[0].Content))
	})

	t.Run("converts user message", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "Hello, how are you?",
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		assert.Equal(t, "user", result[0].Role)
		assert.Equal(t, "Hello, how are you?", contentString(t, result[0].Content))
	})

	t.Run("converts assistant message", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:    llm_dto.RoleAssistant,
				Content: "I'm doing well, thank you!",
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		assert.Equal(t, "assistant", result[0].Role)
		assert.Equal(t, "I'm doing well, thank you!", contentString(t, result[0].Content))
	})

	t.Run("converts assistant message with tool calls", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:    llm_dto.RoleAssistant,
				Content: "",
				ToolCalls: []llm_dto.ToolCall{
					{
						ID:   "call_123",
						Type: "function",
						Function: llm_dto.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location": "London"}`,
						},
					},
				},
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		assert.Equal(t, "assistant", result[0].Role)
		require.Len(t, result[0].ToolCalls, 1)
		assert.Equal(t, "call_123", result[0].ToolCalls[0].ID)
		assert.Equal(t, "function", result[0].ToolCalls[0].Type)
		assert.Equal(t, "get_weather", result[0].ToolCalls[0].Function.Name)
		assert.Equal(t, `{"location": "London"}`, result[0].ToolCalls[0].Function.Arguments)
	})

	t.Run("converts tool message", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:       llm_dto.RoleTool,
				Content:    `{"temperature": 15, "unit": "celsius"}`,
				ToolCallID: new("call_123"),
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		assert.Equal(t, "tool", result[0].Role)
		assert.Equal(t, `{"temperature": 15, "unit": "celsius"}`, contentString(t, result[0].Content))
		assert.Equal(t, "call_123", result[0].ToolCallID)
	})

	t.Run("converts multiple messages", func(t *testing.T) {
		messages := []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "System prompt"},
			{Role: llm_dto.RoleUser, Content: "User message"},
			{Role: llm_dto.RoleAssistant, Content: "Assistant response"},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 3)
		assert.Equal(t, "system", result[0].Role)
		assert.Equal(t, "user", result[1].Role)
		assert.Equal(t, "assistant", result[2].Role)
	})

	t.Run("converts user message with text-only ContentParts", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("hello world"),
				},
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		var parts []mistralContentPart
		require.NoError(t, json.Unmarshal(result[0].Content, &parts))
		require.Len(t, parts, 1)
		assert.Equal(t, "text", parts[0].Type)
		assert.Equal(t, "hello world", parts[0].Text)
	})

	t.Run("converts user message with image URL", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("describe this"),
					llm_dto.ImageURLPart("https://example.com/photo.jpg"),
				},
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		var parts []mistralContentPart
		require.NoError(t, json.Unmarshal(result[0].Content, &parts))
		require.Len(t, parts, 2)
		assert.Equal(t, "text", parts[0].Type)
		assert.Equal(t, "describe this", parts[0].Text)
		assert.Equal(t, "image_url", parts[1].Type)
		require.NotNil(t, parts[1].ImageURL)
		assert.Equal(t, "https://example.com/photo.jpg", parts[1].ImageURL.URL)
	})

	t.Run("converts user message with base64 image data", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("what is this?"),
					llm_dto.ImageDataPart("image/png", "iVBORw0KGgo="),
				},
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		var parts []mistralContentPart
		require.NoError(t, json.Unmarshal(result[0].Content, &parts))
		require.Len(t, parts, 2)
		assert.Equal(t, "text", parts[0].Type)
		assert.Equal(t, "image_url", parts[1].Type)
		require.NotNil(t, parts[1].ImageURL)
		assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", parts[1].ImageURL.URL)
	})

	t.Run("converts user message with mixed content", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("compare these"),
					llm_dto.ImageURLPart("https://example.com/a.jpg"),
					llm_dto.ImageDataPart("image/jpeg", "dGVzdA=="),
				},
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		var parts []mistralContentPart
		require.NoError(t, json.Unmarshal(result[0].Content, &parts))
		require.Len(t, parts, 3)
		assert.Equal(t, "text", parts[0].Type)
		assert.Equal(t, "image_url", parts[1].Type)
		assert.Equal(t, "image_url", parts[2].Type)
	})

	t.Run("ContentParts takes priority over Content", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "this should be ignored",
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("this should be used"),
				},
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		var parts []mistralContentPart
		require.NoError(t, json.Unmarshal(result[0].Content, &parts))
		require.Len(t, parts, 1)
		assert.Equal(t, "this should be used", parts[0].Text)
	})

	t.Run("converts message with name", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "Hello from Alice",
				Name:    new("Alice"),
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		assert.Equal(t, "user", result[0].Role)
		assert.Equal(t, "Alice", result[0].Name)
		assert.Equal(t, "Hello from Alice", contentString(t, result[0].Content))
	})

	t.Run("omits name when nil", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "Hello",
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		assert.Empty(t, result[0].Name)
	})

	t.Run("assistant message ignores ContentParts", func(t *testing.T) {
		messages := []llm_dto.Message{
			{
				Role:    llm_dto.RoleAssistant,
				Content: "response text",
				ContentParts: []llm_dto.ContentPart{
					llm_dto.TextPart("should be ignored"),
				},
			},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 1)
		assert.Equal(t, "response text", contentString(t, result[0].Content))
	})
}

func TestMistralProvider_ConvertTools(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts tool with all fields", func(t *testing.T) {
		tools := []llm_dto.ToolDefinition{
			{
				Type: "function",
				Function: llm_dto.FunctionDefinition{
					Name:        "get_weather",
					Description: new("Get the current weather"),
					Parameters: &llm_dto.JSONSchema{
						Type: "object",
						Properties: map[string]*llm_dto.JSONSchema{
							"location": {
								Type:        "string",
								Description: new("The city name"),
							},
						},
						Required: []string{"location"},
					},
				},
			},
		}

		result := p.convertTools(tools)

		require.Len(t, result, 1)
		assert.Equal(t, "function", result[0].Type)
		assert.Equal(t, "get_weather", result[0].Function.Name)
		assert.Equal(t, "Get the current weather", result[0].Function.Description)
		assert.NotNil(t, result[0].Function.Parameters)
	})

	t.Run("converts tool without description", func(t *testing.T) {
		tools := []llm_dto.ToolDefinition{
			{
				Type: "function",
				Function: llm_dto.FunctionDefinition{
					Name: "simple_function",
				},
			},
		}

		result := p.convertTools(tools)

		require.Len(t, result, 1)
		assert.Equal(t, "simple_function", result[0].Function.Name)
		assert.Empty(t, result[0].Function.Description)
	})
}

func TestMistralProvider_ConvertToolChoice(t *testing.T) {
	p := newTestProvider(t)

	testCases := []struct {
		expected any
		choice   *llm_dto.ToolChoice
		name     string
	}{
		{
			name:     "auto",
			choice:   &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeAuto},
			expected: "auto",
		},
		{
			name:     "none",
			choice:   &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeNone},
			expected: "none",
		},
		{
			name:     "required converts to any",
			choice:   &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeRequired},
			expected: "any",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := p.convertToolChoice(tc.choice)
			assert.Equal(t, tc.expected, result)
		})
	}

	t.Run("specific function", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{
			Type: llm_dto.ToolChoiceTypeFunction,
			Function: &llm_dto.ToolChoiceFunction{
				Name: "get_weather",
			},
		}

		result := p.convertToolChoice(choice)

		tc, ok := result.(mistralToolChoiceFunction)
		require.True(t, ok)
		assert.Equal(t, "function", tc.Type)
		assert.Equal(t, "get_weather", tc.Function.Name)
	})
}

func TestMistralProvider_ConvertResponseFormat(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts JSON object format", func(t *testing.T) {
		format := &llm_dto.ResponseFormat{
			Type: llm_dto.ResponseFormatJSONObject,
		}

		result := p.convertResponseFormat(format)

		require.NotNil(t, result)
		assert.Equal(t, "json_object", result.Type)
	})

	t.Run("converts text format", func(t *testing.T) {
		format := &llm_dto.ResponseFormat{
			Type: llm_dto.ResponseFormatText,
		}

		result := p.convertResponseFormat(format)

		require.NotNil(t, result)
		assert.Equal(t, "text", result.Type)
	})
}

func TestMistralProvider_ConvertResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts basic response", func(t *testing.T) {
		response := &mistralResponse{
			ID:      "chat-123",
			Model:   "mistral-large-latest",
			Created: 1234567890,
			Choices: []mistralChoice{
				{
					Index: 0,
					Message: mistralChoiceMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you?",
					},
					FinishReason: "stop",
				},
			},
			Usage: &mistralUsage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		result := p.convertResponse(response)

		assert.Equal(t, "chat-123", result.ID)
		assert.Equal(t, "mistral-large-latest", result.Model)
		assert.Equal(t, int64(1234567890), result.Created)
		require.Len(t, result.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
		assert.Equal(t, "Hello! How can I help you?", result.Choices[0].Message.Content)
		assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)
		require.NotNil(t, result.Usage)
		assert.Equal(t, 10, result.Usage.PromptTokens)
		assert.Equal(t, 8, result.Usage.CompletionTokens)
		assert.Equal(t, 18, result.Usage.TotalTokens)
	})

	t.Run("converts response with tool calls", func(t *testing.T) {
		response := &mistralResponse{
			ID:      "chat-456",
			Model:   "mistral-large-latest",
			Created: 1234567890,
			Choices: []mistralChoice{
				{
					Index: 0,
					Message: mistralChoiceMessage{
						Role:    "assistant",
						Content: "",
						ToolCalls: []mistralToolCall{
							{
								ID:   "call_abc",
								Type: "function",
								Function: mistralFunctionCall{
									Name:      "get_weather",
									Arguments: `{"location": "Paris"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}

		result := p.convertResponse(response)

		require.Len(t, result.Choices, 1)
		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		tc := result.Choices[0].Message.ToolCalls[0]
		assert.Equal(t, "call_abc", tc.ID)
		assert.Equal(t, "function", tc.Type)
		assert.Equal(t, "get_weather", tc.Function.Name)
		assert.Equal(t, `{"location": "Paris"}`, tc.Function.Arguments)
		assert.Equal(t, llm_dto.FinishReasonToolCalls, result.Choices[0].FinishReason)
	})

	t.Run("converts response without usage", func(t *testing.T) {
		response := &mistralResponse{
			ID:      "chat-789",
			Model:   "mistral-large-latest",
			Created: 1234567890,
			Choices: []mistralChoice{
				{
					Index: 0,
					Message: mistralChoiceMessage{
						Role:    "assistant",
						Content: "Response without usage",
					},
					FinishReason: "stop",
				},
			},
		}

		result := p.convertResponse(response)

		assert.Nil(t, result.Usage)
	})
}

func TestMistralProvider_ConvertFinishReason(t *testing.T) {
	p := newTestProvider(t)

	testCases := []struct {
		input    string
		expected llm_dto.FinishReason
	}{
		{input: "stop", expected: llm_dto.FinishReasonStop},
		{input: "length", expected: llm_dto.FinishReasonLength},
		{input: "tool_calls", expected: llm_dto.FinishReasonToolCalls},
		{input: "model_length", expected: llm_dto.FinishReasonLength},
		{input: "unknown", expected: llm_dto.FinishReasonStop},
		{input: "", expected: llm_dto.FinishReasonStop},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := p.convertFinishReason(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMistralProvider_BuildRequest(t *testing.T) {
	p := newTestProvider(t)

	t.Run("builds basic request", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		result := p.buildRequest(request, "mistral-large-latest", false)

		assert.Equal(t, "mistral-large-latest", result.Model)
		assert.False(t, result.Stream)
		require.Len(t, result.Messages, 1)
	})

	t.Run("builds request with all parameters", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			Temperature: new(0.7),
			MaxTokens:   new(100),
			TopP:        new(0.9),
			Stop:        []string{"END"},
			Seed:        new(int64(42)),
		}

		result := p.buildRequest(request, "mistral-large-latest", true)

		assert.True(t, result.Stream)
		require.NotNil(t, result.Temperature)
		assert.Equal(t, 0.7, *result.Temperature)
		require.NotNil(t, result.MaxTokens)
		assert.Equal(t, 100, *result.MaxTokens)
		require.NotNil(t, result.TopP)
		assert.Equal(t, 0.9, *result.TopP)
		assert.Equal(t, []string{"END"}, result.Stop)
		require.NotNil(t, result.RandomSeed)
		assert.Equal(t, int64(42), *result.RandomSeed)
	})

	t.Run("builds request with tools", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			Tools: []llm_dto.ToolDefinition{
				{
					Type: "function",
					Function: llm_dto.FunctionDefinition{
						Name:        "test_func",
						Description: new("A test function"),
					},
				},
			},
			ToolChoice: &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeAuto},
		}

		result := p.buildRequest(request, "mistral-large-latest", false)

		require.Len(t, result.Tools, 1)
		assert.Equal(t, "test_func", result.Tools[0].Function.Name)
		assert.Equal(t, "auto", result.ToolChoice)
	})

	t.Run("builds request with frequency penalty", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			FrequencyPenalty: new(0.5),
		}

		result := p.buildRequest(request, "mistral-large-latest", false)

		require.NotNil(t, result.FrequencyPenalty)
		assert.Equal(t, 0.5, *result.FrequencyPenalty)
	})

	t.Run("builds request with presence penalty", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			PresencePenalty: new(0.8),
		}

		result := p.buildRequest(request, "mistral-large-latest", false)

		require.NotNil(t, result.PresencePenalty)
		assert.Equal(t, 0.8, *result.PresencePenalty)
	})

	t.Run("omits penalties when nil", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		result := p.buildRequest(request, "mistral-large-latest", false)

		assert.Nil(t, result.FrequencyPenalty)
		assert.Nil(t, result.PresencePenalty)
	})

	t.Run("builds request with response format", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			ResponseFormat: &llm_dto.ResponseFormat{
				Type: llm_dto.ResponseFormatJSONObject,
			},
		}

		result := p.buildRequest(request, "mistral-large-latest", false)

		require.NotNil(t, result.ResponseFormat)
		assert.Equal(t, "json_object", result.ResponseFormat.Type)
	})
}

func TestMistralProvider_CapabilityMethods(t *testing.T) {
	p := newTestProvider(t)

	assert.Equal(t, true, p.SupportsPenalties())
	assert.Equal(t, true, p.SupportsSeed())
	assert.Equal(t, false, p.SupportsParallelToolCalls())
	assert.Equal(t, true, p.SupportsMessageName())
}
