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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestAnthropicProvider_ConvertStructuredOutputResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("extracts structured output from tool use", func(t *testing.T) {
		message := unmarshalMessage(t, fmt.Sprintf(`{
			"id": "msg_structured",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "tool_use", "id": "call_so", "name": %q, "input": {"name": "John", "age": 30}}],
			"stop_reason": "tool_use",
			"stop_sequence": null,
			"usage": {"input_tokens": 20, "output_tokens": 15, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`, structuredOutputToolName))

		result, err := p.convertStructuredOutputResponse(message, "claude-sonnet-4-5-20250929")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "msg_structured", result.ID)
		assert.Equal(t, "claude-sonnet-4-5-20250929", result.Model)
		require.Len(t, result.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, result.Choices[0].Message.Role)
		assert.Equal(t, llm_dto.FinishReasonStop, result.Choices[0].FinishReason)

		var parsed map[string]any
		err = json.Unmarshal([]byte(result.Choices[0].Message.Content), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "John", parsed["name"])

		require.NotNil(t, result.Usage)
		assert.Equal(t, 20, result.Usage.PromptTokens)
		assert.Equal(t, 15, result.Usage.CompletionTokens)
		assert.Equal(t, 35, result.Usage.TotalTokens)
	})

	t.Run("falls back to regular conversion when no structured output tool", func(t *testing.T) {
		message := unmarshalMessage(t, `{
			"id": "msg_fallback",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Plain text response"}],
			"stop_reason": "end_turn",
			"stop_sequence": null,
			"usage": {"input_tokens": 0, "output_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`)

		result, err := p.convertStructuredOutputResponse(message, "claude-sonnet-4-5-20250929")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "msg_fallback", result.ID)
		assert.Equal(t, "Plain text response", result.Choices[0].Message.Content)
	})

	t.Run("ignores tool use blocks with different names", func(t *testing.T) {
		message := unmarshalMessage(t, `{
			"id": "msg_other_tool",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "tool_use", "id": "call_other", "name": "regular_tool", "input": {"key": "value"}}],
			"stop_reason": "tool_use",
			"stop_sequence": null,
			"usage": {"input_tokens": 0, "output_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`)

		result, err := p.convertStructuredOutputResponse(message, "claude-sonnet-4-5-20250929")

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Choices[0].Message.ToolCalls, 1)
		assert.Equal(t, "regular_tool", result.Choices[0].Message.ToolCalls[0].Function.Name)
	})

	t.Run("returns response without usage when zero tokens", func(t *testing.T) {
		message := unmarshalMessage(t, fmt.Sprintf(`{
			"id": "msg_no_usage",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "tool_use", "id": "call_so", "name": %q, "input": {"result": true}}],
			"stop_reason": "tool_use",
			"stop_sequence": null,
			"usage": {"input_tokens": 0, "output_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`, structuredOutputToolName))

		result, err := p.convertStructuredOutputResponse(message, "claude-sonnet-4-5-20250929")

		require.NoError(t, err)
		assert.Nil(t, result.Usage)
	})
}
