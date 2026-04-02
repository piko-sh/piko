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

func TestOpenAIProvider_BuildDelta(t *testing.T) {
	p := newTestProvider(t)

	t.Run("handles text delta", func(t *testing.T) {
		state := &streamState{}
		choice := &openai.ChatCompletionChunkChoice{
			Delta: openai.ChatCompletionChunkChoiceDelta{
				Content: "Hello world",
			},
		}

		delta := p.buildDelta(choice, state)

		require.NotNil(t, delta)
		require.NotNil(t, delta.Content)
		assert.Equal(t, "Hello world", *delta.Content)
	})

	t.Run("handles role delta", func(t *testing.T) {
		state := &streamState{}
		choice := &openai.ChatCompletionChunkChoice{
			Delta: openai.ChatCompletionChunkChoiceDelta{
				Role: "assistant",
			},
		}

		delta := p.buildDelta(choice, state)

		require.NotNil(t, delta)
		require.NotNil(t, delta.Role)
		assert.Equal(t, llm_dto.Role("assistant"), *delta.Role)
	})

	t.Run("handles empty delta", func(t *testing.T) {
		state := &streamState{}
		choice := &openai.ChatCompletionChunkChoice{
			Delta: openai.ChatCompletionChunkChoiceDelta{},
		}

		delta := p.buildDelta(choice, state)

		require.NotNil(t, delta)
		assert.Nil(t, delta.Content)
		assert.Nil(t, delta.Role)
		assert.Empty(t, delta.ToolCalls)
	})

	t.Run("handles tool call delta", func(t *testing.T) {
		state := &streamState{}
		choice := &openai.ChatCompletionChunkChoice{
			Delta: openai.ChatCompletionChunkChoiceDelta{
				ToolCalls: []openai.ChatCompletionChunkChoiceDeltaToolCall{
					{
						Index: 0,
						ID:    "call_1",
						Type:  "function",
						Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
							Name:      "get_weather",
							Arguments: `{"loc`,
						},
					},
				},
			},
		}

		delta := p.buildDelta(choice, state)

		require.NotNil(t, delta)
		require.Len(t, delta.ToolCalls, 1)
		assert.Equal(t, 0, delta.ToolCalls[0].Index)
		require.NotNil(t, delta.ToolCalls[0].ID)
		assert.Equal(t, "call_1", *delta.ToolCalls[0].ID)
		require.NotNil(t, delta.ToolCalls[0].Function)
		require.NotNil(t, delta.ToolCalls[0].Function.Name)
		assert.Equal(t, "get_weather", *delta.ToolCalls[0].Function.Name)
	})
}

func TestOpenAIProvider_BuildToolCallDeltas(t *testing.T) {
	p := newTestProvider(t)

	t.Run("converts multiple tool calls", func(t *testing.T) {
		state := &streamState{}
		toolCalls := []openai.ChatCompletionChunkChoiceDeltaToolCall{
			{Index: 0, ID: "call_1", Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{Name: "tool_a"}},
			{Index: 1, ID: "call_2", Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{Name: "tool_b"}},
		}

		deltas := p.buildToolCallDeltas(toolCalls, state)

		require.Len(t, deltas, 2)
		assert.Equal(t, 0, deltas[0].Index)
		assert.Equal(t, 1, deltas[1].Index)
	})
}

func TestOpenAIProvider_BuildFunctionCallDelta(t *testing.T) {
	p := newTestProvider(t)

	t.Run("builds delta with name and arguments", func(t *testing.T) {
		functionCall := &openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
			Name:      "get_weather",
			Arguments: `{"location": "Paris"}`,
		}

		result := p.buildFunctionCallDelta(functionCall)

		require.NotNil(t, result.Name)
		assert.Equal(t, "get_weather", *result.Name)
		require.NotNil(t, result.Arguments)
		assert.Equal(t, `{"location": "Paris"}`, *result.Arguments)
	})

	t.Run("builds delta with only name", func(t *testing.T) {
		functionCall := &openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
			Name: "my_func",
		}

		result := p.buildFunctionCallDelta(functionCall)

		require.NotNil(t, result.Name)
		assert.Equal(t, "my_func", *result.Name)
		assert.Nil(t, result.Arguments)
	})

	t.Run("builds delta with only arguments", func(t *testing.T) {
		functionCall := &openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
			Arguments: `ation": "London"}`,
		}

		result := p.buildFunctionCallDelta(functionCall)

		assert.Nil(t, result.Name)
		require.NotNil(t, result.Arguments)
		assert.Equal(t, `ation": "London"}`, *result.Arguments)
	})

	t.Run("builds empty delta", func(t *testing.T) {
		functionCall := &openai.ChatCompletionChunkChoiceDeltaToolCallFunction{}

		result := p.buildFunctionCallDelta(functionCall)

		assert.Nil(t, result.Name)
		assert.Nil(t, result.Arguments)
	})
}

func TestOpenAIProvider_ExtractFinishReason(t *testing.T) {
	p := newTestProvider(t)

	t.Run("returns nil when no finish reason", func(t *testing.T) {
		state := &streamState{}
		choice := &openai.ChatCompletionChunkChoice{}

		result := p.extractFinishReason(choice, state)

		assert.Nil(t, result)
		assert.Nil(t, state.lastFinishReason)
	})

	t.Run("extracts stop finish reason", func(t *testing.T) {
		state := &streamState{}
		choice := &openai.ChatCompletionChunkChoice{
			FinishReason: "stop",
		}

		result := p.extractFinishReason(choice, state)

		require.NotNil(t, result)
		assert.Equal(t, llm_dto.FinishReasonStop, *result)
		require.NotNil(t, state.lastFinishReason)
		assert.Equal(t, llm_dto.FinishReasonStop, *state.lastFinishReason)
	})

	t.Run("extracts tool_calls finish reason", func(t *testing.T) {
		state := &streamState{}
		choice := &openai.ChatCompletionChunkChoice{
			FinishReason: "tool_calls",
		}

		result := p.extractFinishReason(choice, state)

		require.NotNil(t, result)
		assert.Equal(t, llm_dto.FinishReasonToolCalls, *result)
	})
}

func TestOpenAIProvider_ExtractUsage(t *testing.T) {
	p := newTestProvider(t)

	t.Run("extracts usage when present", func(t *testing.T) {
		state := &streamState{}
		chunk := &openai.ChatCompletionChunk{
			Usage: openai.CompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
		streamChunk := &llm_dto.StreamChunk{}

		p.extractUsage(chunk, streamChunk, state)

		require.NotNil(t, streamChunk.Usage)
		assert.Equal(t, 10, streamChunk.Usage.PromptTokens)
		assert.Equal(t, 20, streamChunk.Usage.CompletionTokens)
		assert.Equal(t, 30, streamChunk.Usage.TotalTokens)
		require.NotNil(t, state.finalUsage)
		assert.Equal(t, 30, state.finalUsage.TotalTokens)
	})

	t.Run("skips usage when zero total tokens", func(t *testing.T) {
		state := &streamState{}
		chunk := &openai.ChatCompletionChunk{}
		streamChunk := &llm_dto.StreamChunk{}

		p.extractUsage(chunk, streamChunk, state)

		assert.Nil(t, streamChunk.Usage)
		assert.Nil(t, state.finalUsage)
	})
}

func TestOpenAIProvider_BuildFinalResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("builds basic final response", func(t *testing.T) {
		state := &streamState{
			lastID:           "chatcmpl-123",
			lastModel:        "gpt-4.1",
			lastFinishReason: new(llm_dto.FinishReasonStop),
		}

		result := p.buildFinalResponse(state)

		assert.Equal(t, "chatcmpl-123", result.ID)
		assert.Equal(t, "gpt-4.1", result.Model)
		require.Len(t, result.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
		assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)
	})

	t.Run("includes usage in final response", func(t *testing.T) {
		state := &streamState{
			lastID:           "chatcmpl-123",
			lastModel:        "gpt-4.1",
			lastFinishReason: new(llm_dto.FinishReasonStop),
			finalUsage: &llm_dto.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		result := p.buildFinalResponse(state)

		require.NotNil(t, result.Usage)
		assert.Equal(t, 30, result.Usage.TotalTokens)
	})

	t.Run("includes accumulated tool calls", func(t *testing.T) {
		state := &streamState{
			lastID:           "chatcmpl-456",
			lastModel:        "gpt-4.1",
			lastFinishReason: new(llm_dto.FinishReasonToolCalls),
			accumulatedToolCalls: []llm_dto.ToolCall{
				{
					ID:   "call_abc",
					Type: "function",
					Function: llm_dto.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location": "Paris"}`,
					},
				},
			},
		}

		result := p.buildFinalResponse(state)

		require.Len(t, result.Choices, 1)
		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		assert.Equal(t, "call_abc", result.Choices[0].Message.ToolCalls[0].ID)
		assert.Equal(t, "get_weather", result.Choices[0].Message.ToolCalls[0].Function.Name)
	})

	t.Run("no choices when no finish reason", func(t *testing.T) {
		state := &streamState{
			lastID:    "chatcmpl-789",
			lastModel: "gpt-4.1",
		}

		result := p.buildFinalResponse(state)

		assert.Empty(t, result.Choices)
	})
}

func TestOpenAIProvider_AccumulateToolCall(t *testing.T) {
	p := newTestProvider(t)

	t.Run("creates new tool call entry", func(t *testing.T) {
		toolCalls := []llm_dto.ToolCall{}
		delta := llm_dto.ToolCallDelta{
			Index: 0,
			ID:    new("call_1"),
			Type:  new("function"),
			Function: &llm_dto.FunctionCallDelta{
				Name:      new("get_weather"),
				Arguments: new(`{"loc`),
			},
		}

		p.accumulateToolCall(&toolCalls, delta)

		require.Len(t, toolCalls, 1)
		assert.Equal(t, "call_1", toolCalls[0].ID)
		assert.Equal(t, "function", toolCalls[0].Type)
		assert.Equal(t, "get_weather", toolCalls[0].Function.Name)
		assert.Equal(t, `{"loc`, toolCalls[0].Function.Arguments)
	})

	t.Run("appends arguments to existing tool call", func(t *testing.T) {
		toolCalls := []llm_dto.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: llm_dto.FunctionCall{
					Name:      "get_weather",
					Arguments: `{"loc`,
				},
			},
		}
		delta := llm_dto.ToolCallDelta{
			Index: 0,
			Function: &llm_dto.FunctionCallDelta{
				Arguments: new(`ation": "Paris"}`),
			},
		}

		p.accumulateToolCall(&toolCalls, delta)

		require.Len(t, toolCalls, 1)
		assert.Equal(t, `{"location": "Paris"}`, toolCalls[0].Function.Arguments)
	})

	t.Run("grows slice for higher index", func(t *testing.T) {
		toolCalls := []llm_dto.ToolCall{}
		delta := llm_dto.ToolCallDelta{
			Index: 2,
			ID:    new("call_2"),
		}

		p.accumulateToolCall(&toolCalls, delta)

		require.Len(t, toolCalls, 3)
		assert.Equal(t, "call_2", toolCalls[2].ID)
		assert.Equal(t, "", toolCalls[0].ID)
		assert.Equal(t, "", toolCalls[1].ID)
	})

	t.Run("handles delta with nil fields", func(t *testing.T) {
		toolCalls := []llm_dto.ToolCall{
			{ID: "call_1", Type: "function"},
		}

		delta := llm_dto.ToolCallDelta{Index: 0}

		p.accumulateToolCall(&toolCalls, delta)

		assert.Equal(t, "call_1", toolCalls[0].ID)
		assert.Equal(t, "function", toolCalls[0].Type)
	})
}
