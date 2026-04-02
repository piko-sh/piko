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

package llm_provider_anthropic

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func newTestProvider(t *testing.T) *anthropicProvider {
	t.Helper()
	config := Config{
		APIKey: "test-api-key",
	}
	provider, err := New(config)
	require.NoError(t, err)
	ap, ok := provider.(*anthropicProvider)
	require.True(t, ok, "expected provider to be *anthropicProvider")
	return ap
}

func unmarshalMessage(t *testing.T, jsonString string) *anthropic.Message {
	t.Helper()
	var message anthropic.Message
	require.NoError(t, json.Unmarshal([]byte(jsonString), &message))
	return &message
}

func TestAnthropicProvider_ExtractSystemMessage(t *testing.T) {
	p := newTestProvider(t)

	t.Run("extracts system message", func(t *testing.T) {
		messages := []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "You are a helpful assistant."},
			{Role: llm_dto.RoleUser, Content: "Hello"},
		}

		result := p.extractSystemMessage(messages)

		assert.Equal(t, "You are a helpful assistant.", result)
	})

	t.Run("returns empty when no system message", func(t *testing.T) {
		messages := []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		}

		result := p.extractSystemMessage(messages)

		assert.Empty(t, result)
	})

	t.Run("returns first system message", func(t *testing.T) {
		messages := []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "First system"},
			{Role: llm_dto.RoleSystem, Content: "Second system"},
		}

		result := p.extractSystemMessage(messages)

		assert.Equal(t, "First system", result)
	})

	t.Run("returns empty for empty messages", func(t *testing.T) {
		result := p.extractSystemMessage(nil)

		assert.Empty(t, result)
	})
}

func TestAnthropicProvider_ConvertMessages(t *testing.T) {
	p := newTestProvider(t)

	t.Run("filters out system messages", func(t *testing.T) {
		messages := []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "System prompt"},
			{Role: llm_dto.RoleUser, Content: "Hello"},
			{Role: llm_dto.RoleAssistant, Content: "Hi there"},
		}

		result := p.convertMessages(messages)

		require.Len(t, result, 2)
		assert.Equal(t, anthropic.MessageParamRoleUser, result[0].Role)
		assert.Equal(t, anthropic.MessageParamRoleAssistant, result[1].Role)
	})

	t.Run("converts empty messages", func(t *testing.T) {
		result := p.convertMessages(nil)

		assert.Empty(t, result)
	})
}

func TestAnthropicProvider_ConvertMessage(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts user message", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleUser,
			Content: "Hello, how are you?",
		}

		result := p.convertMessage(message)

		assert.Equal(t, anthropic.MessageParamRoleUser, result.Role)
		require.Len(t, result.Content, 1)
		require.NotNil(t, result.Content[0].OfText)
		assert.Equal(t, "Hello, how are you?", result.Content[0].OfText.Text)
	})

	t.Run("converts assistant message", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: "I'm doing well!",
		}

		result := p.convertMessage(message)

		assert.Equal(t, anthropic.MessageParamRoleAssistant, result.Role)
		require.Len(t, result.Content, 1)
		require.NotNil(t, result.Content[0].OfText)
		assert.Equal(t, "I'm doing well!", result.Content[0].OfText.Text)
	})

	t.Run("converts assistant message with tool calls", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: "Let me check that.",
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
		}

		result := p.convertMessage(message)

		assert.Equal(t, anthropic.MessageParamRoleAssistant, result.Role)
		require.Len(t, result.Content, 2)
		require.NotNil(t, result.Content[0].OfText)
		assert.Equal(t, "Let me check that.", result.Content[0].OfText.Text)
		require.NotNil(t, result.Content[1].OfToolUse)
		assert.Equal(t, "call_123", result.Content[1].OfToolUse.ID)
		assert.Equal(t, "get_weather", result.Content[1].OfToolUse.Name)
	})

	t.Run("converts assistant message with tool calls and no text", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleAssistant,
			ToolCalls: []llm_dto.ToolCall{
				{
					ID:   "call_456",
					Type: "function",
					Function: llm_dto.FunctionCall{
						Name:      "search",
						Arguments: `{"query": "test"}`,
					},
				},
			},
		}

		result := p.convertMessage(message)

		assert.Equal(t, anthropic.MessageParamRoleAssistant, result.Role)
		require.Len(t, result.Content, 1)
		require.NotNil(t, result.Content[0].OfToolUse)
	})

	t.Run("converts tool result message", func(t *testing.T) {
		message := llm_dto.Message{
			Role:       llm_dto.RoleTool,
			Content:    `{"temperature": 15}`,
			ToolCallID: new("call_123"),
		}

		result := p.convertMessage(message)

		assert.Equal(t, anthropic.MessageParamRoleUser, result.Role)
		require.Len(t, result.Content, 1)
		require.NotNil(t, result.Content[0].OfToolResult)
		assert.Equal(t, "call_123", result.Content[0].OfToolResult.ToolUseID)
	})

	t.Run("converts tool result with nil tool call ID", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleTool,
			Content: `{"result": "ok"}`,
		}

		result := p.convertMessage(message)

		assert.Equal(t, anthropic.MessageParamRoleUser, result.Role)
		require.Len(t, result.Content, 1)
		require.NotNil(t, result.Content[0].OfToolResult)
		assert.Equal(t, "", result.Content[0].OfToolResult.ToolUseID)
	})

	t.Run("converts user message with text-only ContentParts", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("hello world"),
			},
		}

		result := p.convertMessage(message)

		assert.Equal(t, anthropic.MessageParamRoleUser, result.Role)
		require.Len(t, result.Content, 1)
		require.NotNil(t, result.Content[0].OfText)
		assert.Equal(t, "hello world", result.Content[0].OfText.Text)
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

		assert.Equal(t, anthropic.MessageParamRoleUser, result.Role)
		require.Len(t, result.Content, 2)
		require.NotNil(t, result.Content[0].OfText)
		assert.Equal(t, "describe this", result.Content[0].OfText.Text)
		require.NotNil(t, result.Content[1].OfImage)
		require.NotNil(t, result.Content[1].OfImage.Source.OfURL)
		assert.Equal(t, "https://example.com/photo.jpg", result.Content[1].OfImage.Source.OfURL.URL)
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

		assert.Equal(t, anthropic.MessageParamRoleUser, result.Role)
		require.Len(t, result.Content, 2)
		require.NotNil(t, result.Content[0].OfText)
		assert.Equal(t, "what is this?", result.Content[0].OfText.Text)
		require.NotNil(t, result.Content[1].OfImage)
		require.NotNil(t, result.Content[1].OfImage.Source.OfBase64)
		assert.Equal(t, "iVBORw0KGgo=", result.Content[1].OfImage.Source.OfBase64.Data)
		assert.Equal(t, anthropic.Base64ImageSourceMediaType("image/png"), result.Content[1].OfImage.Source.OfBase64.MediaType)
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

		assert.Equal(t, anthropic.MessageParamRoleUser, result.Role)
		require.Len(t, result.Content, 3)
		require.NotNil(t, result.Content[0].OfText)
		require.NotNil(t, result.Content[1].OfImage)
		require.NotNil(t, result.Content[1].OfImage.Source.OfURL)
		require.NotNil(t, result.Content[2].OfImage)
		require.NotNil(t, result.Content[2].OfImage.Source.OfBase64)
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

		require.Len(t, result.Content, 1)
		require.NotNil(t, result.Content[0].OfText)
		assert.Equal(t, "this should be used", result.Content[0].OfText.Text)
	})
}

func TestAnthropicProvider_ConvertStopReason(t *testing.T) {
	p := newTestProvider(t)

	testCases := []struct {
		input    anthropic.StopReason
		expected llm_dto.FinishReason
	}{
		{input: anthropic.StopReasonMaxTokens, expected: llm_dto.FinishReasonLength},
		{input: anthropic.StopReasonToolUse, expected: llm_dto.FinishReasonToolCalls},
		{input: anthropic.StopReasonEndTurn, expected: llm_dto.FinishReasonStop},
		{input: anthropic.StopReasonStopSequence, expected: llm_dto.FinishReasonStop},
		{input: "", expected: llm_dto.FinishReasonStop},
		{input: "unknown_reason", expected: llm_dto.FinishReasonStop},
	}

	for _, tc := range testCases {
		name := string(tc.input)
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			result := p.convertStopReason(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAnthropicProvider_ConvertToolChoice(t *testing.T) {
	p := newTestProvider(t)

	t.Run("auto", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeAuto}

		result := p.convertToolChoice(choice)

		assert.NotNil(t, result.OfAuto)
	})

	t.Run("none", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeNone}

		result := p.convertToolChoice(choice)

		assert.NotNil(t, result.OfNone)
	})

	t.Run("required converts to any", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeRequired}

		result := p.convertToolChoice(choice)

		assert.NotNil(t, result.OfAny)
	})

	t.Run("specific function", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{
			Type: llm_dto.ToolChoiceTypeFunction,
			Function: &llm_dto.ToolChoiceFunction{
				Name: "get_weather",
			},
		}

		result := p.convertToolChoice(choice)

		require.NotNil(t, result.OfTool)
		assert.Equal(t, "get_weather", result.OfTool.Name)
	})

	t.Run("function type without function defaults to auto", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: llm_dto.ToolChoiceTypeFunction}

		result := p.convertToolChoice(choice)

		assert.NotNil(t, result.OfAuto)
	})

	t.Run("unknown type defaults to auto", func(t *testing.T) {
		choice := &llm_dto.ToolChoice{Type: "unknown"}

		result := p.convertToolChoice(choice)

		assert.NotNil(t, result.OfAuto)
	})
}

func TestAnthropicProvider_SchemaToProperties(t *testing.T) {
	p := newTestProvider(t)

	t.Run("nil schema returns empty map", func(t *testing.T) {
		result := p.schemaToProperties(nil)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Empty(t, resultMap)
	})

	t.Run("schema with nil properties returns empty map", func(t *testing.T) {
		schema := &llm_dto.JSONSchema{
			Type: "object",
		}

		result := p.schemaToProperties(schema)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Empty(t, resultMap)
	})

	t.Run("converts schema properties", func(t *testing.T) {
		schema := &llm_dto.JSONSchema{
			Type: "object",
			Properties: map[string]*llm_dto.JSONSchema{
				"location": {
					Type:        "string",
					Description: new("The city name"),
				},
			},
		}

		result := p.schemaToProperties(schema)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)
		require.Contains(t, resultMap, "location")
		locMap, ok := resultMap["location"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", locMap["type"])
		assert.Equal(t, "The city name", locMap["description"])
	})
}

func TestAnthropicProvider_ConvertTools(t *testing.T) {
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
		require.NotNil(t, result[0].OfTool)
		assert.Equal(t, "get_weather", result[0].OfTool.Name)
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
		require.NotNil(t, result[0].OfTool)
		assert.Equal(t, "simple_function", result[0].OfTool.Name)
	})

	t.Run("converts multiple tools", func(t *testing.T) {
		tools := []llm_dto.ToolDefinition{
			{Type: "function", Function: llm_dto.FunctionDefinition{Name: "tool_a"}},
			{Type: "function", Function: llm_dto.FunctionDefinition{Name: "tool_b"}},
		}

		result := p.convertTools(tools)

		require.Len(t, result, 2)
		assert.Equal(t, "tool_a", result[0].OfTool.Name)
		assert.Equal(t, "tool_b", result[1].OfTool.Name)
	})
}

func TestAnthropicProvider_ConvertResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts basic text response", func(t *testing.T) {
		message := unmarshalMessage(t, `{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Hello! How can I help you?"}],
			"stop_reason": "end_turn",
			"stop_sequence": null,
			"usage": {"input_tokens": 10, "output_tokens": 8, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`)

		result := p.convertResponse(message, "claude-sonnet-4-5-20250929")

		assert.Equal(t, "msg_123", result.ID)
		assert.Equal(t, "claude-sonnet-4-5-20250929", result.Model)
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
		message := unmarshalMessage(t, `{
			"id": "msg_456",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "tool_use", "id": "call_abc", "name": "get_weather", "input": {"location": "Paris"}}],
			"stop_reason": "tool_use",
			"stop_sequence": null,
			"usage": {"input_tokens": 15, "output_tokens": 20, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`)

		result := p.convertResponse(message, "claude-sonnet-4-5-20250929")

		require.Len(t, result.Choices, 1)
		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		tc := result.Choices[0].Message.ToolCalls[0]
		assert.Equal(t, "call_abc", tc.ID)
		assert.Equal(t, "function", tc.Type)
		assert.Equal(t, "get_weather", tc.Function.Name)
		assert.Contains(t, tc.Function.Arguments, `"location"`)
		assert.Contains(t, tc.Function.Arguments, `"Paris"`)
		assert.Equal(t, llm_dto.FinishReasonToolCalls, result.Choices[0].FinishReason)
	})

	t.Run("converts response with text and tool use", func(t *testing.T) {
		message := unmarshalMessage(t, `{
			"id": "msg_789",
			"type": "message",
			"role": "assistant",
			"content": [
				{"type": "text", "text": "Let me look that up."},
				{"type": "tool_use", "id": "call_xyz", "name": "search", "input": {"q": "test"}}
			],
			"stop_reason": "tool_use",
			"stop_sequence": null,
			"usage": {"input_tokens": 0, "output_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`)

		result := p.convertResponse(message, "claude-sonnet-4-5-20250929")

		require.Len(t, result.Choices, 1)
		assert.Equal(t, "Let me look that up.", result.Choices[0].Message.Content)
		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		assert.Equal(t, "search", result.Choices[0].Message.ToolCalls[0].Function.Name)
	})

	t.Run("converts response without usage", func(t *testing.T) {
		message := unmarshalMessage(t, `{
			"id": "msg_no_usage",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Response without usage"}],
			"stop_reason": "end_turn",
			"stop_sequence": null,
			"usage": {"input_tokens": 0, "output_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`)

		result := p.convertResponse(message, "claude-sonnet-4-5-20250929")

		assert.Nil(t, result.Usage)
	})
}

func TestAnthropicProvider_BuildMessageParams(t *testing.T) {
	p := newTestProvider(t)

	t.Run("builds basic request", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		assert.Equal(t, anthropic.Model("claude-sonnet-4-5-20250929"), result.Model)
		assert.Equal(t, int64(DefaultMaxTokens), result.MaxTokens)
		require.Len(t, result.Messages, 1)
	})

	t.Run("extracts system message", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleSystem, Content: "You are helpful."},
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		require.Len(t, result.System, 1)
		assert.Equal(t, "You are helpful.", result.System[0].Text)
		require.Len(t, result.Messages, 1)
	})

	t.Run("uses custom max tokens", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages:  []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
			MaxTokens: new(100),
		}

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		assert.Equal(t, int64(100), result.MaxTokens)
	})

	t.Run("sets temperature", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages:    []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
			Temperature: new(0.7),
		}

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		require.NotNil(t, result.Temperature)
	})

	t.Run("sets top_p", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
			TopP:     new(0.9),
		}

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		require.NotNil(t, result.TopP)
	})

	t.Run("sets stop sequences", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
			Stop:     []string{"END", "STOP"},
		}

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		assert.Equal(t, []string{"END", "STOP"}, result.StopSequences)
	})

	t.Run("sets tools", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
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

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		require.Len(t, result.Tools, 1)
		assert.NotNil(t, result.ToolChoice.OfAuto)
	})

	t.Run("sets metadata user_id when present", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
			Metadata: map[string]string{
				"user_id": "usr-abc-123",
			},
		}

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		assert.True(t, result.Metadata.UserID.Valid())
		assert.Equal(t, "usr-abc-123", result.Metadata.UserID.Value)
	})

	t.Run("does not set metadata when user_id is absent", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
			Metadata: map[string]string{
				"some_other_key": "value",
			},
		}

		result := p.buildMessageParams(request, "claude-sonnet-4-5-20250929")

		assert.False(t, result.Metadata.UserID.Valid())
	})
}

func TestAnthropicProvider_CapabilityMethods(t *testing.T) {
	p := newTestProvider(t)

	assert.Equal(t, false, p.SupportsPenalties())
	assert.Equal(t, false, p.SupportsSeed())
	assert.Equal(t, false, p.SupportsParallelToolCalls())
	assert.Equal(t, false, p.SupportsMessageName())
}
