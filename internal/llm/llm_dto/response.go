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

// CompletionResponse represents the response from an LLM completion request.
type CompletionResponse struct {
	// Usage holds token counts for the request; nil when not available.
	Usage *Usage

	// FallbackInfo contains details about fallback execution if fallback was
	// used. This is nil if the request did not use fallback or succeeded with
	// the primary provider.
	FallbackInfo *FallbackResult

	// ID is the unique identifier for this completion.
	ID string

	// Model is the name of the model that produced this completion.
	Model string

	// Choices holds the generated completions. Most requests return one choice,
	// but the n parameter can request more.
	Choices []Choice

	// Sources holds vector search results used for RAG context injection.
	// This is nil when the request did not use RAG, and empty when RAG was
	// configured but no matching documents were found.
	Sources []VectorSearchResult

	// Created is the Unix timestamp in seconds when the completion was made.
	Created int64
}

// Choice represents a single completion option from the model.
type Choice struct {
	// FinishReason indicates why the model stopped generating tokens.
	FinishReason FinishReason

	// Message holds the generated response content. It contains the text output
	// and any tool calls the model has requested.
	Message Message

	// Index is the position of this choice in the response, starting from 0.
	Index int
}

// FinishReason indicates why the model stopped generating tokens.
type FinishReason string

const (
	// FinishReasonStop indicates the model reached a natural stop point or
	// a provided stop sequence.
	FinishReasonStop FinishReason = "stop"

	// FinishReasonLength means the model stopped because it reached the maximum
	// number of tokens allowed.
	FinishReasonLength FinishReason = "length"

	// FinishReasonToolCalls means the model wants to call one or more tools.
	FinishReasonToolCalls FinishReason = "tool_calls"

	// FinishReasonContentFilter indicates the content was filtered due to policy.
	FinishReasonContentFilter FinishReason = "content_filter"
)

// Usage holds token usage figures for a completion request.
type Usage struct {
	// EstimatedCost holds the cost worked out for this request.
	// The service layer fills this in using the pricing table.
	EstimatedCost *CostEstimate

	// PromptTokens is the number of tokens in the prompt input.
	PromptTokens int

	// CompletionTokens is the number of tokens in the generated output.
	CompletionTokens int

	// TotalTokens is the total number of tokens used (prompt plus completion).
	TotalTokens int

	// CachedTokens is the number of prompt tokens served from the provider's
	// prompt cache. These are billed at a lower rate when the model supports it.
	CachedTokens int
}

// FirstChoice returns the first choice from the response, or an empty Choice
// if no choices are present. This is a convenience method since most requests
// return exactly one choice.
//
// Returns Choice which is the first choice, or an empty Choice if none exist.
func (r *CompletionResponse) FirstChoice() Choice {
	if len(r.Choices) == 0 {
		return Choice{}
	}
	return r.Choices[0]
}

// Content returns the text content from the first choice. This is a convenience
// method for the common case of extracting the assistant's response.
//
// Returns string which is the content of the first choice's message, or empty
// string if no choices exist.
func (r *CompletionResponse) Content() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}

// HasToolCalls reports whether the first choice contains tool calls.
//
// Returns bool which is true if tool calls are present.
func (r *CompletionResponse) HasToolCalls() bool {
	if len(r.Choices) == 0 {
		return false
	}
	return len(r.Choices[0].Message.ToolCalls) > 0
}

// ToolCalls returns the tool calls from the first choice. This is a convenience
// method for extracting tool calls from the response.
//
// Returns []ToolCall which contains the tool calls, or nil if none exist.
func (r *CompletionResponse) ToolCalls() []ToolCall {
	if len(r.Choices) == 0 {
		return nil
	}
	return r.Choices[0].Message.ToolCalls
}
