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

package llm_provider_gemini

import (
	"testing"

	"google.golang.org/genai"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func newTestProvider(t *testing.T) *geminiProvider {
	t.Helper()
	return &geminiProvider{
		defaultModel: "gemini-2.5-flash",
		config: Config{
			APIKey:       "test-key",
			DefaultModel: "gemini-2.5-flash",
		},
	}
}

func TestGeminiProvider_ConvertMessage(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts user message", func(t *testing.T) {
		message := llm_dto.Message{Role: llm_dto.RoleUser, Content: "Hello"}

		result := p.convertMessage(message)

		assert.Equal(t, "user", string(result.Role))
		require.Len(t, result.Parts, 1)
		assert.Equal(t, "Hello", result.Parts[0].Text)
	})

	t.Run("converts assistant message", func(t *testing.T) {
		message := llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Hi there"}

		result := p.convertMessage(message)

		assert.Equal(t, "model", string(result.Role))
		require.Len(t, result.Parts, 1)
		assert.Equal(t, "Hi there", result.Parts[0].Text)
	})

	t.Run("converts assistant message with tool calls", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleAssistant,
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

		assert.Equal(t, "model", string(result.Role))
		require.Len(t, result.Parts, 1)
		require.NotNil(t, result.Parts[0].FunctionCall)
		assert.Equal(t, "get_weather", result.Parts[0].FunctionCall.Name)
		assert.Equal(t, "Paris", result.Parts[0].FunctionCall.Args["location"])
	})

	t.Run("converts assistant message with text and tool calls", func(t *testing.T) {
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: "Let me check",
			ToolCalls: []llm_dto.ToolCall{
				{
					ID:       "call_1",
					Type:     "function",
					Function: llm_dto.FunctionCall{Name: "my_tool", Arguments: `{}`},
				},
			},
		}

		result := p.convertMessage(message)

		assert.Equal(t, "model", string(result.Role))
		require.Len(t, result.Parts, 2)
		assert.NotEmpty(t, result.Parts[0].Text)
		assert.NotNil(t, result.Parts[1].FunctionCall)
	})

	t.Run("converts tool result message", func(t *testing.T) {
		message := llm_dto.Message{
			Role:       llm_dto.RoleTool,
			Content:    `{"temp": 22}`,
			ToolCallID: new("get_weather"),
		}

		result := p.convertMessage(message)

		assert.Equal(t, "user", string(result.Role))
		require.Len(t, result.Parts, 1)
		require.NotNil(t, result.Parts[0].FunctionResponse)
		assert.Equal(t, "get_weather", result.Parts[0].FunctionResponse.Name)
		assert.Equal(t, float64(22), result.Parts[0].FunctionResponse.Response["temp"])
	})

	t.Run("converts tool result with non-JSON content", func(t *testing.T) {
		message := llm_dto.Message{
			Role:       llm_dto.RoleTool,
			Content:    "plain text result",
			ToolCallID: new("my_tool"),
		}

		result := p.convertMessage(message)

		require.NotNil(t, result.Parts[0].FunctionResponse)
		assert.Equal(t, "plain text result", result.Parts[0].FunctionResponse.Response["result"])
	})

	t.Run("converts empty content message", func(t *testing.T) {
		message := llm_dto.Message{Role: llm_dto.RoleUser, Content: ""}

		result := p.convertMessage(message)

		assert.Equal(t, "user", string(result.Role))
		assert.Empty(t, result.Parts)
	})

	t.Run("converts user message with text-only ContentParts", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("hello world"),
			},
		}

		result := p.convertMessage(message)

		assert.Equal(t, "user", string(result.Role))
		require.Len(t, result.Parts, 1)
		assert.Equal(t, "hello world", result.Parts[0].Text)
	})

	t.Run("converts user message with image URL", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("describe this"),
				llm_dto.ImageURLPart("https://example.com/photo.png"),
			},
		}

		result := p.convertMessage(message)

		assert.Equal(t, "user", string(result.Role))
		require.Len(t, result.Parts, 2)
		assert.Equal(t, "describe this", result.Parts[0].Text)
		require.NotNil(t, result.Parts[1].FileData)
		assert.Equal(t, "https://example.com/photo.png", result.Parts[1].FileData.FileURI)
		assert.Equal(t, "image/png", result.Parts[1].FileData.MIMEType)
	})

	t.Run("converts user message with base64 image data", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("what is this?"),
				llm_dto.ImageDataPart("image/png", "dGVzdA=="),
			},
		}

		result := p.convertMessage(message)

		assert.Equal(t, "user", string(result.Role))
		require.Len(t, result.Parts, 2)
		assert.Equal(t, "what is this?", result.Parts[0].Text)
		require.NotNil(t, result.Parts[1].InlineData)
		assert.Equal(t, "image/png", result.Parts[1].InlineData.MIMEType)
		assert.Equal(t, []byte("test"), result.Parts[1].InlineData.Data)
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

		require.Len(t, result.Parts, 3)
		assert.Equal(t, "compare these", result.Parts[0].Text)
		require.NotNil(t, result.Parts[1].FileData)
		require.NotNil(t, result.Parts[2].InlineData)
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

		require.Len(t, result.Parts, 1)
		assert.Equal(t, "this should be used", result.Parts[0].Text)
	})

	t.Run("skips invalid base64 image data", func(t *testing.T) {
		message := llm_dto.Message{
			Role: llm_dto.RoleUser,
			ContentParts: []llm_dto.ContentPart{
				llm_dto.TextPart("hello"),
				llm_dto.ImageDataPart("image/png", "!!!invalid-base64!!!"),
			},
		}

		result := p.convertMessage(message)

		require.Len(t, result.Parts, 1)
		assert.Equal(t, "hello", result.Parts[0].Text)
	})
}

func TestInferImageMIMEType(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/photo.png", "image/png"},
		{"https://example.com/photo.jpg", "image/jpeg"},
		{"https://example.com/photo.jpeg", "image/jpeg"},
		{"https://example.com/photo.gif", "image/gif"},
		{"https://example.com/photo.webp", "image/webp"},
		{"https://example.com/photo.PNG", "image/png"},
		{"https://example.com/photo", "image/jpeg"},
		{"https://example.com/photo.bmp", "image/bmp"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, inferImageMIMEType(tt.url))
		})
	}
}

func TestGeminiProvider_ConvertHistory(t *testing.T) {
	p := newTestProvider(t)

	t.Run("skips system messages", func(t *testing.T) {
		messages := []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "You are helpful"},
			{Role: llm_dto.RoleUser, Content: "Hello"},
			{Role: llm_dto.RoleAssistant, Content: "Hi"},
		}

		result := p.convertHistory(messages)

		assert.Len(t, result, 2)
		assert.Equal(t, "user", string(result[0].Role))
		assert.Equal(t, "model", string(result[1].Role))
	})

	t.Run("converts empty messages", func(t *testing.T) {
		result := p.convertHistory([]llm_dto.Message{})
		assert.Empty(t, result)
	})
}

func TestGeminiProvider_ConvertTools(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts tool with all fields", func(t *testing.T) {
		tools := []llm_dto.ToolDefinition{
			{
				Type: "function",
				Function: llm_dto.FunctionDefinition{
					Name:        "get_weather",
					Description: new("Get weather for a location"),
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
		assert.Equal(t, "get_weather", result[0].Name)
		assert.Equal(t, "Get weather for a location", result[0].Description)
		require.NotNil(t, result[0].Parameters)
		assert.Equal(t, genai.TypeObject, result[0].Parameters.Type)
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
		assert.Equal(t, "simple_tool", result[0].Name)
		assert.Equal(t, "", result[0].Description)
	})

	t.Run("converts multiple tools", func(t *testing.T) {
		tools := []llm_dto.ToolDefinition{
			{Type: "function", Function: llm_dto.FunctionDefinition{Name: "tool_a"}},
			{Type: "function", Function: llm_dto.FunctionDefinition{Name: "tool_b"}},
		}

		result := p.convertTools(tools)

		require.Len(t, result, 2)
		assert.Equal(t, "tool_a", result[0].Name)
		assert.Equal(t, "tool_b", result[1].Name)
	})
}

func TestGeminiProvider_ConvertSchema(t *testing.T) {
	p := newTestProvider(t)

	t.Run("nil schema returns nil", func(t *testing.T) {
		result := p.convertSchema(nil)
		assert.Nil(t, result)
	})

	t.Run("converts simple schema", func(t *testing.T) {
		schema := &llm_dto.JSONSchema{
			Type:        "object",
			Description: new("A person's name"),
			Properties: map[string]*llm_dto.JSONSchema{
				"name": {Type: "string"},
				"age":  {Type: "integer"},
			},
			Required: []string{"name"},
		}

		result := p.convertSchema(schema)

		require.NotNil(t, result)
		assert.Equal(t, genai.TypeObject, result.Type)
		assert.Equal(t, "A person's name", result.Description)
		require.Len(t, result.Properties, 2)
		assert.Equal(t, genai.TypeString, result.Properties["name"].Type)
		assert.Equal(t, genai.TypeInteger, result.Properties["age"].Type)
		assert.Equal(t, []string{"name"}, result.Required)
	})

	t.Run("converts schema with items", func(t *testing.T) {
		schema := &llm_dto.JSONSchema{
			Type: "array",
			Items: &llm_dto.JSONSchema{
				Type: "string",
			},
		}

		result := p.convertSchema(schema)

		require.NotNil(t, result)
		assert.Equal(t, genai.TypeArray, result.Type)
		require.NotNil(t, result.Items)
		assert.Equal(t, genai.TypeString, result.Items.Type)
	})

	t.Run("converts schema with enum", func(t *testing.T) {
		schema := &llm_dto.JSONSchema{
			Type: "string",
			Enum: []any{"red", "green", "blue"},
		}

		result := p.convertSchema(schema)

		require.NotNil(t, result)
		assert.Equal(t, []string{"red", "green", "blue"}, result.Enum)
	})

	t.Run("ignores non-string enum values", func(t *testing.T) {
		schema := &llm_dto.JSONSchema{
			Type: "string",
			Enum: []any{"valid", 42, true},
		}

		result := p.convertSchema(schema)

		require.NotNil(t, result)
		assert.Equal(t, []string{"valid"}, result.Enum)
	})
}

func TestGeminiProvider_ConvertSchemaType(t *testing.T) {
	p := newTestProvider(t)

	testCases := []struct {
		name   string
		input  string
		expect genai.Type
	}{
		{name: "string", input: "string", expect: genai.TypeString},
		{name: "number", input: "number", expect: genai.TypeNumber},
		{name: "integer", input: "integer", expect: genai.TypeInteger},
		{name: "boolean", input: "boolean", expect: genai.TypeBoolean},
		{name: "array", input: "array", expect: genai.TypeArray},
		{name: "object", input: "object", expect: genai.TypeObject},
		{name: "unknown", input: "unknown", expect: genai.TypeUnspecified},
		{name: "empty", input: "", expect: genai.TypeUnspecified},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, p.convertSchemaType(tc.input))
		})
	}
}

func TestGeminiProvider_ConvertResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts basic text response", func(t *testing.T) {
		response := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{genai.NewPartFromText("Hello world")},
					},
					FinishReason: genai.FinishReasonStop,
				},
			},
			UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 5,
				TotalTokenCount:      15,
			},
		}

		result := p.convertResponse(response, "gemini-2.5-flash")

		assert.Equal(t, "gemini-2.5-flash", result.Model)
		assert.NotEmpty(t, result.ID)
		require.Len(t, result.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
		assert.Equal(t, "Hello world", result.Choices[0].Message.Content)
		assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)
		require.NotNil(t, result.Usage)
		assert.Equal(t, 10, result.Usage.PromptTokens)
		assert.Equal(t, 5, result.Usage.CompletionTokens)
		assert.Equal(t, 15, result.Usage.TotalTokens)
	})

	t.Run("converts response with function calls", func(t *testing.T) {
		response := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							genai.NewPartFromFunctionCall("get_weather", map[string]any{"location": "Paris"}),
						},
					},
					FinishReason: genai.FinishReasonStop,
				},
			},
		}

		result := p.convertResponse(response, "gemini-2.5-flash")

		require.Len(t, result.Choices, 1)
		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		tc := result.Choices[0].Message.ToolCalls[0]
		assert.Equal(t, "call_0", tc.ID)
		assert.Equal(t, "function", tc.Type)
		assert.Equal(t, "get_weather", tc.Function.Name)
		assert.Contains(t, tc.Function.Arguments, "Paris")
	})

	t.Run("converts response with nil content", func(t *testing.T) {
		response := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content:      nil,
					FinishReason: genai.FinishReasonSafety,
				},
			},
		}

		result := p.convertResponse(response, "gemini-2.5-flash")

		require.Len(t, result.Choices, 1)
		assert.Equal(t, "", result.Choices[0].Message.Content)
		assert.Equal(t, llm_dto.FinishReasonContentFilter, result.Choices[0].FinishReason)
	})

	t.Run("converts response without usage", func(t *testing.T) {
		response := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{genai.NewPartFromText("Hello")},
					},
				},
			},
		}

		result := p.convertResponse(response, "gemini-2.5-flash")

		assert.Nil(t, result.Usage)
	})

	t.Run("converts response with multiple text parts", func(t *testing.T) {
		response := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							genai.NewPartFromText("Hello "),
							genai.NewPartFromText("world"),
						},
					},
				},
			},
		}

		result := p.convertResponse(response, "gemini-2.5-flash")

		assert.Equal(t, "Hello world", result.Choices[0].Message.Content)
	})
}

func TestGeminiProvider_BuildGenerateContentConfig(t *testing.T) {
	p := newTestProvider(t)

	t.Run("sets FrequencyPenalty when provided", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			FrequencyPenalty: new(float64(0.5)),
		}

		config := p.buildGenerateContentConfig(request)

		require.NotNil(t, config.FrequencyPenalty)
		assert.InDelta(t, float32(0.5), *config.FrequencyPenalty, 0.0001)
	})

	t.Run("does not set FrequencyPenalty when nil", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{}

		config := p.buildGenerateContentConfig(request)

		assert.Nil(t, config.FrequencyPenalty)
	})

	t.Run("sets PresencePenalty when provided", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			PresencePenalty: new(float64(1.2)),
		}

		config := p.buildGenerateContentConfig(request)

		require.NotNil(t, config.PresencePenalty)
		assert.InDelta(t, float32(1.2), *config.PresencePenalty, 0.0001)
	})

	t.Run("does not set PresencePenalty when nil", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{}

		config := p.buildGenerateContentConfig(request)

		assert.Nil(t, config.PresencePenalty)
	})

	t.Run("sets Seed when provided", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Seed: new(int64(42)),
		}

		config := p.buildGenerateContentConfig(request)

		require.NotNil(t, config.Seed)
		assert.Equal(t, int32(42), *config.Seed)
	})

	t.Run("does not set Seed when nil", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{}

		config := p.buildGenerateContentConfig(request)

		assert.Nil(t, config.Seed)
	})

	t.Run("sets all penalty and seed fields together", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			FrequencyPenalty: new(float64(0.3)),
			PresencePenalty:  new(float64(0.7)),
			Seed:             new(int64(99)),
		}

		config := p.buildGenerateContentConfig(request)

		require.NotNil(t, config.FrequencyPenalty)
		assert.InDelta(t, float32(0.3), *config.FrequencyPenalty, 0.0001)
		require.NotNil(t, config.PresencePenalty)
		assert.InDelta(t, float32(0.7), *config.PresencePenalty, 0.0001)
		require.NotNil(t, config.Seed)
		assert.Equal(t, int32(99), *config.Seed)
	})
}

func TestGeminiProvider_ConvertFinishReason(t *testing.T) {
	p := newTestProvider(t)

	testCases := []struct {
		name   string
		input  genai.FinishReason
		expect llm_dto.FinishReason
	}{
		{name: "stop", input: genai.FinishReasonStop, expect: llm_dto.FinishReasonStop},
		{name: "max tokens", input: genai.FinishReasonMaxTokens, expect: llm_dto.FinishReasonLength},
		{name: "safety", input: genai.FinishReasonSafety, expect: llm_dto.FinishReasonContentFilter},
		{name: "recitation", input: genai.FinishReasonRecitation, expect: llm_dto.FinishReasonContentFilter},
		{name: "unspecified", input: genai.FinishReasonUnspecified, expect: llm_dto.FinishReasonStop},
		{name: "other", input: genai.FinishReasonOther, expect: llm_dto.FinishReasonStop},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, p.convertFinishReason(tc.input))
		})
	}
}

func TestGeminiProvider_CapabilityMethods(t *testing.T) {
	p := newTestProvider(t)

	assert.Equal(t, true, p.SupportsPenalties())
	assert.Equal(t, true, p.SupportsSeed())
	assert.Equal(t, false, p.SupportsParallelToolCalls())
	assert.Equal(t, false, p.SupportsMessageName())
}
