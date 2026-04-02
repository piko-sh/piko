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

package llm_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletionResponse_FirstChoice(t *testing.T) {
	t.Parallel()

	t.Run("with choices", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{
			Choices: []Choice{
				{Index: 0, Message: Message{Content: "hello"}},
				{Index: 1, Message: Message{Content: "world"}},
			},
		}
		choice := response.FirstChoice()
		assert.Equal(t, "hello", choice.Message.Content)
		assert.Equal(t, 0, choice.Index)
	})

	t.Run("empty choices", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{}
		choice := response.FirstChoice()
		assert.Equal(t, Choice{}, choice)
	})
}

func TestCompletionResponse_Content(t *testing.T) {
	t.Parallel()

	t.Run("with content", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{
			Choices: []Choice{
				{Message: Message{Content: "test response"}},
			},
		}
		assert.Equal(t, "test response", response.Content())
	})

	t.Run("empty choices", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{}
		assert.Equal(t, "", response.Content())
	})
}

func TestCompletionResponse_HasToolCalls(t *testing.T) {
	t.Parallel()

	t.Run("with tool calls", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{
			Choices: []Choice{
				{Message: Message{
					ToolCalls: []ToolCall{
						{ID: "call_1", Type: "function", Function: FunctionCall{Name: "search"}},
					},
				}},
			},
		}
		assert.True(t, response.HasToolCalls())
	})

	t.Run("without tool calls", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{
			Choices: []Choice{
				{Message: Message{Content: "no tools"}},
			},
		}
		assert.False(t, response.HasToolCalls())
	})

	t.Run("empty choices", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{}
		assert.False(t, response.HasToolCalls())
	})
}

func TestCompletionResponse_ToolCalls(t *testing.T) {
	t.Parallel()

	t.Run("with tool calls", func(t *testing.T) {
		t.Parallel()

		calls := []ToolCall{
			{ID: "call_1", Type: "function", Function: FunctionCall{Name: "search"}},
			{ID: "call_2", Type: "function", Function: FunctionCall{Name: "read"}},
		}
		response := &CompletionResponse{
			Choices: []Choice{
				{Message: Message{ToolCalls: calls}},
			},
		}
		result := response.ToolCalls()
		assert.Len(t, result, 2)
		assert.Equal(t, "search", result[0].Function.Name)
		assert.Equal(t, "read", result[1].Function.Name)
	})

	t.Run("empty choices", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{}
		assert.Nil(t, response.ToolCalls())
	})
}
