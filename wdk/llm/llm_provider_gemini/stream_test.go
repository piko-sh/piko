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

func TestNewStreamState(t *testing.T) {
	state := newStreamState("gemini-2.5-flash")

	assert.Equal(t, "gemini-2.5-flash", state.model)
	assert.Equal(t, llm_dto.FinishReasonStop, state.lastFinishReason)
	assert.NotEmpty(t, state.messageID)
	assert.Contains(t, state.messageID, "gemini-")
	assert.Empty(t, state.accumulatedToolCalls)
	assert.Nil(t, state.finalUsage)
}

func TestGeminiProvider_ProcessCandidateParts(t *testing.T) {
	p := newTestProvider(t)

	t.Run("handles text part", func(t *testing.T) {
		state := newStreamState("test-model")
		candidate := &genai.Candidate{
			Content: &genai.Content{
				Parts: []*genai.Part{genai.NewPartFromText("Hello world")},
			},
		}

		delta := p.processCandidateParts(candidate, state)

		require.NotNil(t, delta)
		require.NotNil(t, delta.Content)
		assert.Equal(t, "Hello world", *delta.Content)
	})

	t.Run("handles nil content", func(t *testing.T) {
		state := newStreamState("test-model")
		candidate := &genai.Candidate{
			Content: nil,
		}

		delta := p.processCandidateParts(candidate, state)

		require.NotNil(t, delta)
		assert.Nil(t, delta.Content)
		assert.Empty(t, delta.ToolCalls)
	})

	t.Run("handles function call part", func(t *testing.T) {
		state := newStreamState("test-model")
		candidate := &genai.Candidate{
			Content: &genai.Content{
				Parts: []*genai.Part{
					genai.NewPartFromFunctionCall("get_weather", map[string]any{"location": "London"}),
				},
			},
		}

		delta := p.processCandidateParts(candidate, state)

		require.NotNil(t, delta)
		require.Len(t, delta.ToolCalls, 1)
		assert.Equal(t, 0, delta.ToolCalls[0].Index)
		require.NotNil(t, delta.ToolCalls[0].Function)
		assert.Equal(t, "get_weather", *delta.ToolCalls[0].Function.Name)
	})
}

func TestGeminiProvider_ProcessPart(t *testing.T) {
	p := newTestProvider(t)

	t.Run("processes text part", func(t *testing.T) {
		state := newStreamState("test-model")
		delta := &llm_dto.MessageDelta{}

		p.processPart(genai.NewPartFromText("Hello"), delta, state)

		require.NotNil(t, delta.Content)
		assert.Equal(t, "Hello", *delta.Content)
	})

	t.Run("processes function call part", func(t *testing.T) {
		state := newStreamState("test-model")
		delta := &llm_dto.MessageDelta{}

		p.processPart(genai.NewPartFromFunctionCall("test_func", map[string]any{"key": "value"}), delta, state)

		require.Len(t, delta.ToolCalls, 1)
		require.Len(t, state.accumulatedToolCalls, 1)
	})
}

func TestGeminiProvider_HandleFunctionCall(t *testing.T) {
	p := newTestProvider(t)

	t.Run("creates tool call and delta", func(t *testing.T) {
		state := newStreamState("test-model")
		delta := &llm_dto.MessageDelta{}

		fc := &genai.FunctionCall{
			Name: "get_weather",
			Args: map[string]any{"location": "Paris"},
		}

		p.handleFunctionCall(fc, delta, state)

		require.Len(t, state.accumulatedToolCalls, 1)
		assert.Equal(t, "call_0", state.accumulatedToolCalls[0].ID)
		assert.Equal(t, "function", state.accumulatedToolCalls[0].Type)
		assert.Equal(t, "get_weather", state.accumulatedToolCalls[0].Function.Name)
		assert.Contains(t, state.accumulatedToolCalls[0].Function.Arguments, "Paris")

		require.Len(t, delta.ToolCalls, 1)
		assert.Equal(t, 0, delta.ToolCalls[0].Index)
		require.NotNil(t, delta.ToolCalls[0].ID)
		assert.Equal(t, "call_0", *delta.ToolCalls[0].ID)
	})

	t.Run("increments tool call index", func(t *testing.T) {
		state := newStreamState("test-model")
		delta1 := &llm_dto.MessageDelta{}
		delta2 := &llm_dto.MessageDelta{}

		p.handleFunctionCall(&genai.FunctionCall{Name: "tool_a", Args: map[string]any{}}, delta1, state)
		p.handleFunctionCall(&genai.FunctionCall{Name: "tool_b", Args: map[string]any{}}, delta2, state)

		require.Len(t, state.accumulatedToolCalls, 2)
		assert.Equal(t, "call_0", state.accumulatedToolCalls[0].ID)
		assert.Equal(t, "call_1", state.accumulatedToolCalls[1].ID)
		assert.Equal(t, 1, delta2.ToolCalls[0].Index)
	})
}

func TestGeminiProvider_UpdateUsage(t *testing.T) {
	p := newTestProvider(t)

	t.Run("updates usage when metadata present", func(t *testing.T) {
		state := newStreamState("test-model")
		response := &genai.GenerateContentResponse{
			UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 20,
				TotalTokenCount:      30,
			},
		}

		p.updateUsage(response, state)

		require.NotNil(t, state.finalUsage)
		assert.Equal(t, 10, state.finalUsage.PromptTokens)
		assert.Equal(t, 20, state.finalUsage.CompletionTokens)
		assert.Equal(t, 30, state.finalUsage.TotalTokens)
	})

	t.Run("skips when no metadata", func(t *testing.T) {
		state := newStreamState("test-model")
		response := &genai.GenerateContentResponse{}

		p.updateUsage(response, state)

		assert.Nil(t, state.finalUsage)
	})
}

func TestGeminiProvider_BuildFinalResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("builds basic final response", func(t *testing.T) {
		state := &streamState{
			messageID:        "gemini-123",
			model:            "gemini-2.5-flash",
			lastFinishReason: llm_dto.FinishReasonStop,
		}

		result := p.buildFinalResponse(state)

		assert.Equal(t, "gemini-123", result.ID)
		assert.Equal(t, "gemini-2.5-flash", result.Model)
		require.Len(t, result.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
		assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)
	})

	t.Run("includes usage in final response", func(t *testing.T) {
		state := &streamState{
			messageID:        "gemini-123",
			model:            "gemini-2.5-flash",
			lastFinishReason: llm_dto.FinishReasonStop,
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
			messageID:        "gemini-456",
			model:            "gemini-2.5-flash",
			lastFinishReason: llm_dto.FinishReasonStop,
			accumulatedToolCalls: []llm_dto.ToolCall{
				{
					ID:   "call_0",
					Type: "function",
					Function: llm_dto.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location":"Paris"}`,
					},
				},
			},
		}

		result := p.buildFinalResponse(state)

		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		assert.Equal(t, "call_0", result.Choices[0].Message.ToolCalls[0].ID)
		assert.Equal(t, "get_weather", result.Choices[0].Message.ToolCalls[0].Function.Name)
	})
}
