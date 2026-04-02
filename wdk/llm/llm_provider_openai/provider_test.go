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

package llm_provider_openai

import (
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func newTestProvider(t *testing.T) *openaiProvider {
	t.Helper()
	config := Config{
		APIKey: "test-api-key",
	}
	provider, err := New(config)
	require.NoError(t, err)
	op, ok := provider.(*openaiProvider)
	require.True(t, ok, "expected provider to be *openaiProvider")
	return op
}

func TestOpenAIProvider_ConvertMessages(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts all message roles", func(t *testing.T) {
		messages := []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "You are a helper."},
			{Role: llm_dto.RoleUser, Content: "Hello"},
			{Role: llm_dto.RoleAssistant, Content: "Hi there"},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 3)
		assert.NotNil(t, result[0].OfSystem)
		assert.NotNil(t, result[1].OfUser)
		assert.NotNil(t, result[2].OfAssistant)
	})

	t.Run("converts empty messages", func(t *testing.T) {
		result := p.convertMessages([]llm_dto.Message{})
		assert.Empty(t, result)
	})
}

func TestOpenAIProvider_ConvertMessage(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts system message", func(t *testing.T) {
		message := llm_dto.Message{Role: llm_dto.RoleSystem, Content: "Be helpful"}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfSystem)
	})

	t.Run("converts user message", func(t *testing.T) {
		message := llm_dto.Message{Role: llm_dto.RoleUser, Content: "Hello"}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
	})

	t.Run("converts assistant message", func(t *testing.T) {
		message := llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Hi there"}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfAssistant)
	})

	t.Run("converts assistant message with tool calls", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: "Let me check",
			ToolCalls: []llm_dto.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: llm_dto.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location": "Paris"}`,
					},
				},
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfAssistant)
		require.Len(t, result.OfAssistant.ToolCalls, 1)
		require.NotNil(t, result.OfAssistant.ToolCalls[0].OfFunction)
		assert.Equal(t, "call_1", result.OfAssistant.ToolCalls[0].OfFunction.ID)
		assert.Equal(t, "get_weather", result.OfAssistant.ToolCalls[0].OfFunction.Function.Name)
		assert.Equal(t, `{"location": "Paris"}`, result.OfAssistant.ToolCalls[0].OfFunction.Function.Arguments)
	})

	t.Run("converts tool result message", func(t *testing.T) {
		message := llm_dto.Message{
			Role:       llm_dto.RoleTool,
			Content:    `{"temp": 22}`,
			ToolCallID: new("call_1"),
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfTool)
		assert.Equal(t, "call_1", result.OfTool.ToolCallID)
	})

	t.Run("converts tool result with nil tool call ID", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleTool,
			Content: `{"temp": 22}`,
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfTool)
		assert.Equal(t, "", result.OfTool.ToolCallID)
	})

	t.Run("unknown role defaults to user", func(t *testing.T) {
		message := llm_dto.Message{Role: "unknown", Content: "Hello"}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
	})

	t.Run("converts user message with text-only ContentParts", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("hello world"),
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		require.Len(t, result.OfUser.Content.OfArrayOfContentParts, 1)
		require.NotNil(t, result.OfUser.Content.OfArrayOfContentParts[0].OfText)
		assert.Equal(t, "hello world", result.OfUser.Content.OfArrayOfContentParts[0].OfText.Text)
	})

	t.Run("converts user message with image URL", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("describe this"),
				llm_dto.ImageURLPart("https://example.com/photo.jpg"),
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		parts := result.OfUser.Content.OfArrayOfContentParts
		require.Len(t, parts, 2)
		require.NotNil(t, parts[0].OfText)
		assert.Equal(t, "describe this", parts[0].OfText.Text)
		require.NotNil(t, parts[1].OfImageURL)
		assert.Equal(t, "https://example.com/photo.jpg", parts[1].OfImageURL.ImageURL.URL)
	})

	t.Run("converts user message with image URL and detail", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("analyse this"),
				llm_dto.ImageURLPart("https://example.com/photo.jpg", "high"),
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		parts := result.OfUser.Content.OfArrayOfContentParts
		require.Len(t, parts, 2)
		require.NotNil(t, parts[1].OfImageURL)
		assert.Equal(t, "high", parts[1].OfImageURL.ImageURL.Detail)
	})

	t.Run("converts user message with base64 image data", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("what is this?"),
				llm_dto.ImageDataPart("image/png", "iVBORw0KGgo="),
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		parts := result.OfUser.Content.OfArrayOfContentParts
		require.Len(t, parts, 2)
		require.NotNil(t, parts[1].OfImageURL)
		assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", parts[1].OfImageURL.ImageURL.URL)
	})

	t.Run("converts user message with mixed content", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("compare these"),
				llm_dto.ImageURLPart("https://example.com/a.jpg"),
				llm_dto.ImageDataPart("image/jpeg", "dGVzdA=="),
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		parts := result.OfUser.Content.OfArrayOfContentParts
		require.Len(t, parts, 3)
		require.NotNil(t, parts[0].OfText)
		require.NotNil(t, parts[1].OfImageURL)
		require.NotNil(t, parts[2].OfImageURL)
	})

	t.Run("ContentParts takes priority over Content", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleUser,
			Content: "this should be ignored",
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("this should be used"),
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		parts := result.OfUser.Content.OfArrayOfContentParts
		require.Len(t, parts, 1)
		require.NotNil(t, parts[0].OfText)
		assert.Equal(t, "this should be used", parts[0].OfText.Text)
	})

	t.Run("sets Name on system message", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleSystem,
			Content: "Be helpful",
			Name:    new("system-agent"),
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfSystem)
		assert.Equal(t, "system-agent", result.OfSystem.Name.Value)
	})

	t.Run("sets Name on user message", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleUser,
			Content: "Hello",
			Name:    new("alice"),
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		assert.Equal(t, "alice", result.OfUser.Name.Value)
	})

	t.Run("sets Name on user message with ContentParts", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			Name: new("bob"),
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("hello world"),
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		assert.Equal(t, "bob", result.OfUser.Name.Value)
	})

	t.Run("sets Name on assistant message", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: "Hi there",
			Name:    new("helper-bot"),
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfAssistant)
		assert.Equal(t, "helper-bot", result.OfAssistant.Name.Value)
	})

	t.Run("sets Name on assistant message with tool calls", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: "Let me check",
			Name:    new("tool-agent"),
			ToolCalls: []llm_dto.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: llm_dto.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location": "London"}`,
					},
				},
			},
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfAssistant)
		assert.Equal(t, "tool-agent", result.OfAssistant.Name.Value)
	})

	t.Run("does not set Name when nil", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleUser,
			Content: "Hello",
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfUser)
		assert.Equal(t, "", result.OfUser.Name.Value)
	})

	t.Run("does not set Name on system message when nil", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleSystem,
			Content: "Be helpful",
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfSystem)
		assert.Equal(t, "", result.OfSystem.Name.Value)
	})

	t.Run("does not set Name on assistant message when nil", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: "Hi there",
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.OfAssistant)
		assert.Equal(t, "", result.OfAssistant.Name.Value)
	})
}

func TestOpenAIProvider_ConvertTools(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts tool with all fields", func(t *testing.T) {
		tools := []llm_dto.ToolDefinition{
			{
				Type: "function",
				Function: llm_dto.FunctionDefinition{
					Name:        "get_weather",
					Description: new("Get weather"),
					Strict:      new(true),
					Parameters: &llm_dto.JSONSchema{
						Type: "object",
						Properties: map[string]*llm_dto.JSONSchema{
							"location": {Type: "string"},
						},
						Required: []string{"location"},
					},
				},
			},
		}

		result := p.convertTools(tools)

		require.Len(t, result, 1)
		require.NotNil(t, result[0].OfFunction)
		assert.Equal(t, "get_weather", result[0].OfFunction.Function.Name)
		assert.Equal(t, "Get weather", result[0].OfFunction.Function.Description.Value)
		assert.True(t, result[0].OfFunction.Function.Strict.Value)
	})

	t.Run("converts tool without description", func(t *testing.T) {
		tools := []llm_dto.ToolDefinition{
			{
				Type: "function",
				Function: llm_dto.FunctionDefinition{
					Name: "simple_tool",
				},
			},
		}

		result := p.convertTools(tools)

		require.Len(t, result, 1)
		require.NotNil(t, result[0].OfFunction)
		assert.Equal(t, "simple_tool", result[0].OfFunction.Function.Name)
		assert.Equal(t, "", result[0].OfFunction.Function.Description.Value)
	})

	t.Run("converts multiple tools", func(t *testing.T) {
		tools := []llm_dto.ToolDefinition{
			{Type: "function", Function: llm_dto.FunctionDefinition{Name: "tool_a"}},
			{Type: "function", Function: llm_dto.FunctionDefinition{Name: "tool_b"}},
		}

		result := p.convertTools(tools)

		require.Len(t, result, 2)
		require.NotNil(t, result[0].OfFunction)
		require.NotNil(t, result[1].OfFunction)
		assert.Equal(t, "tool_a", result[0].OfFunction.Function.Name)
		assert.Equal(t, "tool_b", result[1].OfFunction.Function.Name)
	})
}

func TestOpenAIProvider_ConvertToolChoice(t *testing.T) {
	p := newTestProvider(t)

	t.Run("auto", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeAuto}

		result := p.convertToolChoice(choice)

		assert.Equal(t, "auto", result.OfAuto.Value)
	})

	t.Run("none", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeNone}

		result := p.convertToolChoice(choice)

		assert.Equal(t, "none", result.OfAuto.Value)
	})

	t.Run("required", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeRequired}

		result := p.convertToolChoice(choice)

		assert.Equal(t, "required", result.OfAuto.Value)
	})

	t.Run("specific function", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{
			Type:     llm_dto.ToolChoiceTypeFunction,
			Function: &llm_dto.ToolChoiceFunction{Name: "my_func"},
		}

		result := p.convertToolChoice(choice)

		require.NotNil(t, result.OfFunctionToolChoice)
		assert.Equal(t, "my_func", result.OfFunctionToolChoice.Function.Name)
	})

	t.Run("function type without function defaults to auto", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeFunction}

		result := p.convertToolChoice(choice)

		assert.Equal(t, "auto", result.OfAuto.Value)
	})

	t.Run("unknown type defaults to auto", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: "unknown"}

		result := p.convertToolChoice(choice)

		assert.Equal(t, "auto", result.OfAuto.Value)
	})
}

func TestOpenAIProvider_ConvertResponseFormat(t *testing.T) {
	p := newTestProvider(t)

	t.Run("text format", func(t *testing.T) {
		format := &llm_dto.ResponseFormat{Type: llm_dto.ResponseFormatText}

		result := p.convertResponseFormat(format)

		assert.NotNil(t, result.OfText)
	})

	t.Run("JSON object format", func(t *testing.T) {
		format := &llm_dto.ResponseFormat{Type: llm_dto.ResponseFormatJSONObject}

		result := p.convertResponseFormat(format)

		assert.NotNil(t, result.OfJSONObject)
	})

	t.Run("JSON schema format", func(t *testing.T) {
		format := &llm_dto.ResponseFormat{
			Type: llm_dto.ResponseFormatJSONSchema,
			JSONSchema: &llm_dto.JSONSchemaDefinition{
				Name:        "person",
				Description: new("A person"),
				Strict:      new(true),
				Schema: llm_dto.JSONSchema{
					Type: "object",
					Properties: map[string]*llm_dto.JSONSchema{
						"name": {Type: "string"},
					},
				},
			},
		}

		result := p.convertResponseFormat(format)

		require.NotNil(t, result.OfJSONSchema)
		assert.Equal(t, "person", result.OfJSONSchema.JSONSchema.Name)
		assert.Equal(t, "A person", result.OfJSONSchema.JSONSchema.Description.Value)
		assert.True(t, result.OfJSONSchema.JSONSchema.Strict.Value)
	})

	t.Run("JSON schema without schema falls back to text", func(t *testing.T) {
		format := &llm_dto.ResponseFormat{
			Type: llm_dto.ResponseFormatJSONSchema,
		}

		result := p.convertResponseFormat(format)

		assert.NotNil(t, result.OfText)
	})

	t.Run("unknown format defaults to text", func(t *testing.T) {
		format := &llm_dto.ResponseFormat{Type: "unknown"}

		result := p.convertResponseFormat(format)

		assert.NotNil(t, result.OfText)
	})
}

func TestOpenAIProvider_SchemaToMap(t *testing.T) {
	p := newTestProvider(t)

	t.Run("nil schema returns nil", func(t *testing.T) {
		result := p.schemaToMap(nil)
		assert.Nil(t, result)
	})

	t.Run("converts simple schema", func(t *testing.T) {
		schema := &llm_dto.JSONSchema{
			Type: "object",
			Properties: map[string]*llm_dto.JSONSchema{
				"name": {Type: "string"},
				"age":  {Type: "integer"},
			},
			Required: []string{"name"},
		}

		result := p.schemaToMap(schema)

		require.NotNil(t, result)
		assert.Equal(t, "object", result["type"])
		props, ok := result["properties"].(map[string]any)
		require.True(t, ok)
		nameProps, ok := props["name"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", nameProps["type"])
	})
}

func TestOpenAIProvider_ConvertResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts basic text response", func(t *testing.T) {
		completion := &openai.ChatCompletion{
			ID:      "chatcmpl-123",
			Model:   "gpt-4.1",
			Created: 1700000000,
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Content: "Hello there!",
					},
					FinishReason: "stop",
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		result := p.convertResponse(completion)

		assert.Equal(t, "chatcmpl-123", result.ID)
		assert.Equal(t, "gpt-4.1", result.Model)
		assert.Equal(t, int64(1700000000), result.Created)
		require.Len(t, result.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
		assert.Equal(t, "Hello there!", result.Choices[0].Message.Content)
		assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)
		require.NotNil(t, result.Usage)
		assert.Equal(t, 10, result.Usage.PromptTokens)
		assert.Equal(t, 5, result.Usage.CompletionTokens)
		assert.Equal(t, 15, result.Usage.TotalTokens)
	})

	t.Run("converts response with tool calls", func(t *testing.T) {
		completion := &openai.ChatCompletion{
			ID:    "chatcmpl-456",
			Model: "gpt-4.1",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
							{
								ID:   "call_abc",
								Type: "function",
								Function: openai.ChatCompletionMessageFunctionToolCallFunction{
									Name:      "get_weather",
									Arguments: `{"location": "Paris"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     15,
				CompletionTokens: 20,
				TotalTokens:      35,
			},
		}

		result := p.convertResponse(completion)

		require.Len(t, result.Choices, 1)
		assert.Equal(t, llm_dto.FinishReasonToolCalls, result.Choices[0].FinishReason)
		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		tc := result.Choices[0].Message.ToolCalls[0]
		assert.Equal(t, "call_abc", tc.ID)
		assert.Equal(t, "function", tc.Type)
		assert.Equal(t, "get_weather", tc.Function.Name)
		assert.Equal(t, `{"location": "Paris"}`, tc.Function.Arguments)
	})

	t.Run("converts response without usage", func(t *testing.T) {
		completion := &openai.ChatCompletion{
			ID:    "chatcmpl-789",
			Model: "gpt-4.1",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Content: "Hello",
					},
					FinishReason: "stop",
				},
			},
		}

		result := p.convertResponse(completion)

		assert.Nil(t, result.Usage)
	})

	t.Run("converts response with multiple choices", func(t *testing.T) {
		completion := &openai.ChatCompletion{
			ID:    "chatcmpl-multi",
			Model: "gpt-4.1",
			Choices: []openai.ChatCompletionChoice{
				{Index: 0, Message: openai.ChatCompletionMessage{Content: "A"}, FinishReason: "stop"},
				{Index: 1, Message: openai.ChatCompletionMessage{Content: "B"}, FinishReason: "stop"},
			},
			Usage: openai.CompletionUsage{TotalTokens: 20, PromptTokens: 10, CompletionTokens: 10},
		}

		result := p.convertResponse(completion)

		require.Len(t, result.Choices, 2)
		assert.Equal(t, 0, result.Choices[0].Index)
		assert.Equal(t, "A", result.Choices[0].Message.Content)
		assert.Equal(t, 1, result.Choices[1].Index)
		assert.Equal(t, "B", result.Choices[1].Message.Content)
	})
}

func TestOpenAIProvider_ConvertFinishReason(t *testing.T) {
	p := newTestProvider(t)

	testCases := []struct {
		name   string
		input  string
		expect llm_dto.FinishReason
	}{
		{name: "stop", input: "stop", expect: llm_dto.FinishReasonStop},
		{name: "length", input: "length", expect: llm_dto.FinishReasonLength},
		{name: "tool_calls", input: "tool_calls", expect: llm_dto.FinishReasonToolCalls},
		{name: "content_filter", input: "content_filter", expect: llm_dto.FinishReasonContentFilter},
		{name: "empty defaults to stop", input: "", expect: llm_dto.FinishReasonStop},
		{name: "unknown defaults to stop", input: "unknown", expect: llm_dto.FinishReasonStop},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, p.convertFinishReason(tc.input))
		})
	}
}

func TestIsOAChatModel(t *testing.T) {
	testCases := []struct {
		name   string
		id     string
		expect bool
	}{
		{name: "gpt-4.1", id: "gpt-4.1", expect: true},
		{name: "gpt-4.1-mini", id: "gpt-4.1-mini", expect: true},
		{name: "gpt-4o", id: "gpt-4o", expect: true},
		{name: "gpt-3.5-turbo", id: "gpt-3.5-turbo", expect: true},
		{name: "gpt-5", id: "gpt-5", expect: true},
		{name: "o1-preview", id: "o1-preview", expect: true},
		{name: "o3-mini", id: "o3-mini", expect: true},
		{name: "o4-mini", id: "o4-mini", expect: true},
		{name: "chatgpt-4o-latest", id: "chatgpt-4o-latest", expect: true},
		{name: "o1-mini", id: "o1-mini", expect: true},
		{name: "o1.5", id: "o1.5", expect: true},
		{name: "o10", id: "o10", expect: false},
		{name: "dall-e-3", id: "dall-e-3", expect: false},
		{name: "whisper-1", id: "whisper-1", expect: false},
		{name: "text-embedding-3-small", id: "text-embedding-3-small", expect: false},
		{name: "empty string", id: "", expect: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, isOAChatModel(tc.id))
		})
	}
}

func TestOpenAIProvider_BuildChatParams(t *testing.T) {
	p := newTestProvider(t)

	t.Run("builds basic request", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.Equal(t, "gpt-4.1", params.Model)
		require.Len(t, params.Messages, 1)
	})

	t.Run("sets temperature", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages:    []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			Temperature: new(0.7),
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.InDelta(t, 0.7, params.Temperature.Value, 0.001)
	})

	t.Run("sets max tokens", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages:  []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			MaxTokens: new(100),
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.Equal(t, int64(100), params.MaxCompletionTokens.Value)
	})

	t.Run("sets top p", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			TopP:     new(0.9),
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.InDelta(t, 0.9, params.TopP.Value, 0.001)
	})

	t.Run("sets frequency penalty", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages:         []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			FrequencyPenalty: new(0.5),
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.InDelta(t, 0.5, params.FrequencyPenalty.Value, 0.001)
	})

	t.Run("sets presence penalty", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages:        []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			PresencePenalty: new(0.3),
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.InDelta(t, 0.3, params.PresencePenalty.Value, 0.001)
	})

	t.Run("sets stop sequences", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			Stop:     []string{"END", "STOP"},
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.Equal(t, []string{"END", "STOP"}, params.Stop.OfStringArray)
	})

	t.Run("sets seed", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			Seed:     new(int64(42)),
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.Equal(t, int64(42), params.Seed.Value)
	})

	t.Run("sets tools", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			Tools: []llm_dto.ToolDefinition{
				{Type: "function", Function: llm_dto.FunctionDefinition{Name: "my_tool"}},
			},
		}

		params := p.buildChatParams(request, "gpt-4.1")

		require.Len(t, params.Tools, 1)
		require.NotNil(t, params.Tools[0].OfFunction)
		assert.Equal(t, "my_tool", params.Tools[0].OfFunction.Function.Name)
	})

	t.Run("sets parallel tool calls", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
			Tools: []llm_dto.ToolDefinition{
				{Type: "function", Function: llm_dto.FunctionDefinition{Name: "my_tool"}},
			},
			ParallelToolCalls: new(true),
		}

		params := p.buildChatParams(request, "gpt-4.1")

		assert.True(t, params.ParallelToolCalls.Value)
	})
}

func TestOpenAIProvider_CapabilityMethods(t *testing.T) {
	p := newTestProvider(t)

	assert.Equal(t, true, p.SupportsPenalties())
	assert.Equal(t, true, p.SupportsSeed())
	assert.Equal(t, true, p.SupportsParallelToolCalls())
	assert.Equal(t, true, p.SupportsMessageName())
}
