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

func TestNewStreamState(t *testing.T) {
	state := newStreamState("claude-sonnet-4-5-20250929")

	assert.Equal(t, "claude-sonnet-4-5-20250929", state.model)
	assert.Equal(t, -1, state.currentToolIndex)
	assert.Empty(t, state.messageID)
	assert.Empty(t, state.accumulatedToolCalls)
	assert.Nil(t, state.finalUsage)
	assert.Nil(t, state.lastFinishReason)
}

func unmarshalContentBlockStartEvent(t *testing.T, jsonString string) anthropic.ContentBlockStartEvent {
	t.Helper()
	var event anthropic.ContentBlockStartEvent
	require.NoError(t, json.Unmarshal([]byte(jsonString), &event))
	return event
}

func unmarshalContentBlockDeltaEvent(t *testing.T, jsonString string) anthropic.ContentBlockDeltaEvent {
	t.Helper()
	var event anthropic.ContentBlockDeltaEvent
	require.NoError(t, json.Unmarshal([]byte(jsonString), &event))
	return event
}

func TestAnthropicProvider_HandleContentBlockStart(t *testing.T) {
	p := newTestProvider(t)

	t.Run("handles tool use block", func(t *testing.T) {
		state := newStreamState("test-model")
		p.handleContentBlockStart(new(unmarshalContentBlockStartEvent(t, `{
			"type": "content_block_start",
			"index": 0,
			"content_block": {"type": "tool_use", "id": "call_123", "name": "get_weather", "input": {}}
		}`)), state)

		assert.Equal(t, 0, state.currentToolIndex)
		require.Len(t, state.accumulatedToolCalls, 1)
		assert.Equal(t, "call_123", state.accumulatedToolCalls[0].ID)
		assert.Equal(t, "get_weather", state.accumulatedToolCalls[0].Function.Name)
		assert.Equal(t, "function", state.accumulatedToolCalls[0].Type)
	})

	t.Run("handles multiple tool use blocks", func(t *testing.T) {
		state := newStreamState("test-model")

		p.handleContentBlockStart(new(unmarshalContentBlockStartEvent(t, `{
			"type": "content_block_start",
			"index": 0,
			"content_block": {"type": "tool_use", "id": "call_1", "name": "tool_a", "input": {}}
		}`)), state)

		p.handleContentBlockStart(new(unmarshalContentBlockStartEvent(t, `{
			"type": "content_block_start",
			"index": 1,
			"content_block": {"type": "tool_use", "id": "call_2", "name": "tool_b", "input": {}}
		}`)), state)

		assert.Equal(t, 1, state.currentToolIndex)
		require.Len(t, state.accumulatedToolCalls, 2)
		assert.Equal(t, "call_1", state.accumulatedToolCalls[0].ID)
		assert.Equal(t, "call_2", state.accumulatedToolCalls[1].ID)
	})

	t.Run("ignores non-tool-use block", func(t *testing.T) {
		state := newStreamState("test-model")
		p.handleContentBlockStart(new(unmarshalContentBlockStartEvent(t, `{
			"type": "content_block_start",
			"index": 0,
			"content_block": {"type": "text", "text": ""}
		}`)), state)

		assert.Equal(t, -1, state.currentToolIndex)
		assert.Empty(t, state.accumulatedToolCalls)
	})
}

func TestAnthropicProvider_HandleContentBlockDelta(t *testing.T) {
	p := newTestProvider(t)

	t.Run("handles text delta", func(t *testing.T) {
		state := newStreamState("test-model")
		delta := p.handleContentBlockDelta(new(unmarshalContentBlockDeltaEvent(t, `{
			"type": "content_block_delta",
			"index": 0,
			"delta": {"type": "text_delta", "text": "Hello world"}
		}`)), state)

		require.NotNil(t, delta)
		require.NotNil(t, delta.Content)
		assert.Equal(t, "Hello world", *delta.Content)
	})

	t.Run("handles input JSON delta", func(t *testing.T) {
		state := newStreamState("test-model")
		state.currentToolIndex = 0
		state.accumulatedToolCalls = []llm_dto.ToolCall{
			{Type: "function", ID: "call_1"},
		}

		delta := p.handleContentBlockDelta(new(unmarshalContentBlockDeltaEvent(t, `{
			"type": "content_block_delta",
			"index": 0,
			"delta": {"type": "input_json_delta", "partial_json": "{\"location\":"}
		}`)), state)

		require.NotNil(t, delta)
		require.Len(t, delta.ToolCalls, 1)
		assert.Equal(t, 0, delta.ToolCalls[0].Index)
		require.NotNil(t, delta.ToolCalls[0].Function)
		require.NotNil(t, delta.ToolCalls[0].Function.Arguments)
		assert.Equal(t, `{"location":`, *delta.ToolCalls[0].Function.Arguments)
		assert.Equal(t, `{"location":`, state.accumulatedToolCalls[0].Function.Arguments)
	})

	t.Run("accumulates input JSON delta", func(t *testing.T) {
		state := newStreamState("test-model")
		state.currentToolIndex = 0
		state.accumulatedToolCalls = []llm_dto.ToolCall{
			{Type: "function", ID: "call_1", Function: llm_dto.FunctionCall{Arguments: `{"loc`}},
		}

		p.handleContentBlockDelta(new(unmarshalContentBlockDeltaEvent(t, `{
			"type": "content_block_delta",
			"index": 0,
			"delta": {"type": "input_json_delta", "partial_json": "ation\":\"London\"}"}
		}`)), state)

		assert.Equal(t, `{"location":"London"}`, state.accumulatedToolCalls[0].Function.Arguments)
	})
}

func TestAnthropicProvider_HandleMessageDelta(t *testing.T) {
	p := newTestProvider(t)

	t.Run("updates usage", func(t *testing.T) {
		state := newStreamState("test-model")
		event := anthropic.MessageDeltaEvent{
			Usage: anthropic.MessageDeltaUsage{
				OutputTokens: 42,
			},
		}

		p.handleMessageDelta(&event, state)

		require.NotNil(t, state.finalUsage)
		assert.Equal(t, 42, state.finalUsage.CompletionTokens)
	})

	t.Run("updates finish reason", func(t *testing.T) {
		state := newStreamState("test-model")
		event := anthropic.MessageDeltaEvent{
			Delta: anthropic.MessageDeltaEventDelta{
				StopReason: "end_turn",
			},
		}

		p.handleMessageDelta(&event, state)

		require.NotNil(t, state.lastFinishReason)
		assert.Equal(t, llm_dto.FinishReasonStop, *state.lastFinishReason)
	})

	t.Run("updates finish reason for tool use", func(t *testing.T) {
		state := newStreamState("test-model")
		event := anthropic.MessageDeltaEvent{
			Delta: anthropic.MessageDeltaEventDelta{
				StopReason: "tool_use",
			},
		}

		p.handleMessageDelta(&event, state)

		require.NotNil(t, state.lastFinishReason)
		assert.Equal(t, llm_dto.FinishReasonToolCalls, *state.lastFinishReason)
	})

	t.Run("no update when no usage and no stop reason", func(t *testing.T) {
		state := newStreamState("test-model")
		event := anthropic.MessageDeltaEvent{}

		p.handleMessageDelta(&event, state)

		assert.Nil(t, state.finalUsage)
		assert.Nil(t, state.lastFinishReason)
	})
}

func TestAnthropicProvider_BuildFinalResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("builds basic final response", func(t *testing.T) {
		state := &streamState{
			messageID: "msg_123",
			model:     "claude-sonnet-4-5-20250929",
		}

		result := p.buildFinalResponse(state)

		assert.Equal(t, "msg_123", result.ID)
		assert.Equal(t, "claude-sonnet-4-5-20250929", result.Model)
		require.Len(t, result.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
		assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)
	})

	t.Run("includes usage in final response", func(t *testing.T) {
		state := &streamState{
			messageID: "msg_123",
			model:     "test-model",
			finalUsage: &llm_dto.Usage{
				CompletionTokens: 50,
			},
		}

		result := p.buildFinalResponse(state)

		require.NotNil(t, result.Usage)
		assert.Equal(t, 50, result.Usage.CompletionTokens)
	})

	t.Run("includes finish reason from state", func(t *testing.T) {
		state := &streamState{
			messageID:        "msg_123",
			model:            "test-model",
			lastFinishReason: new(llm_dto.FinishReasonToolCalls),
		}

		result := p.buildFinalResponse(state)

		assert.Equal(t, llm_dto.FinishReasonToolCalls, result.Choices[0].FinishReason)
	})

	t.Run("includes accumulated tool calls", func(t *testing.T) {
		state := &streamState{
			messageID: "msg_123",
			model:     "test-model",
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

		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		assert.Equal(t, "call_abc", result.Choices[0].Message.ToolCalls[0].ID)
		assert.Equal(t, "get_weather", result.Choices[0].Message.ToolCalls[0].Function.Name)
	})
}

func TestAnthropicProvider_ValidateToolCallArguments(t *testing.T) {
	p := newTestProvider(t)

	t.Run("valid JSON arguments pass validation", func(t *testing.T) {
		toolCalls := []llm_dto.ToolCall{
			{Function: llm_dto.FunctionCall{Arguments: `{"key": "value"}`}},
		}

		p.validateToolCallArguments(toolCalls)
	})

	t.Run("invalid JSON arguments are handled gracefully", func(t *testing.T) {
		toolCalls := []llm_dto.ToolCall{
			{Function: llm_dto.FunctionCall{Arguments: `{invalid json`}},
		}

		p.validateToolCallArguments(toolCalls)
	})

	t.Run("empty arguments are handled gracefully", func(t *testing.T) {
		toolCalls := []llm_dto.ToolCall{
			{Function: llm_dto.FunctionCall{Arguments: ""}},
		}

		p.validateToolCallArguments(toolCalls)
	})

	t.Run("empty slice passes validation", func(t *testing.T) {
		p.validateToolCallArguments(nil)
	})
}
