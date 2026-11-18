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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/maths"
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

func TestCompletionResponse_Clone(t *testing.T) {
	t.Parallel()

	t.Run("nil response", func(t *testing.T) {
		t.Parallel()

		var response *CompletionResponse
		assert.Nil(t, response.Clone())
	})

	t.Run("copy equals the original", func(t *testing.T) {
		t.Parallel()

		original := newFullResponse()
		assert.Equal(t, original, original.Clone())
	})

	t.Run("empty response stays empty", func(t *testing.T) {
		t.Parallel()

		original := &CompletionResponse{ID: "empty"}
		assert.Equal(t, original, original.Clone())
	})

	t.Run("changing the copy leaves the original alone", func(t *testing.T) {
		t.Parallel()

		original := newFullResponse()
		want := newFullResponse()

		clone := original.Clone()
		mutateResponse(clone)

		assert.Equal(t, want, original)
	})
}

func newFullResponse() *CompletionResponse {
	name := "assistant"
	toolCallID := "call-1"
	text := "part text"
	detail := "high"

	return &CompletionResponse{
		Usage: &Usage{
			EstimatedCost: &CostEstimate{
				Timestamp: time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC),
				TotalCost: maths.NewMoneyFromString("1.25", CostCurrency),
				Model:     "gpt-4o",
			},
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		FallbackInfo: &FallbackResult{
			Errors:             map[string]error{"openai": errors.New("timeout")},
			UsedProvider:       "anthropic",
			AttemptedProviders: []string{"openai", "anthropic"},
		},
		ID:    "response-1",
		Model: "gpt-4o",
		Choices: []Choice{{
			FinishReason: FinishReasonToolCalls,
			Message: Message{
				Name:       &name,
				ToolCallID: &toolCallID,
				Role:       RoleAssistant,
				Content:    "hello",
				ContentParts: []ContentPart{{
					Text:      &text,
					ImageURL:  &ImageURL{Detail: &detail, URL: "https://example.com/a.png"},
					ImageData: &ImageData{MIMEType: "image/png", Data: "AAAA"},
					Type:      ContentPartTypeText,
				}},
				ToolCalls: []ToolCall{{ID: "call-1", Type: "function", Function: FunctionCall{Name: "lookup", Arguments: "{}"}}},
			},
		}},
		Sources: []VectorSearchResult{{
			Metadata: map[string]any{"title": "doc"},
			ID:       "doc-1",
			Content:  "content",
			Vector:   []float32{0.5, 0.25},
			Score:    0.9,
		}},
		Created: 1700000000,
	}
}

func mutateResponse(response *CompletionResponse) {
	response.Usage.PromptTokens = 99
	response.Usage.EstimatedCost.Model = "changed"
	response.FallbackInfo.Errors["openai"] = errors.New("changed")
	response.FallbackInfo.AttemptedProviders[0] = "changed"

	choice := &response.Choices[0]
	choice.FinishReason = FinishReasonStop
	*choice.Message.Name = "changed"
	*choice.Message.ToolCallID = "changed"
	choice.Message.Content = "changed"
	part := &choice.Message.ContentParts[0]
	*part.Text = "changed"
	*part.ImageURL.Detail = "changed"
	part.ImageURL.URL = "changed"
	part.ImageData.Data = "changed"
	choice.Message.ToolCalls[0].Function.Arguments = "changed"

	response.Sources[0].Metadata["title"] = "changed"
	response.Sources[0].Vector[0] = 1
	response.Sources[0].Content = "changed"
}
