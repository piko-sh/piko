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
	"maps"
	"slices"
)

// CompletionResponse represents the response from an LLM completion request.
type CompletionResponse struct {
	// Usage holds token counts for the request; nil when not available.
	Usage *Usage

	// FallbackInfo contains details about fallback execution if fallback was used. This is
	// nil if the request did not use fallback or succeeded with the primary provider.
	FallbackInfo *FallbackResult

	// ID is the unique identifier for this completion.
	ID string

	// Model is the name of the model that produced this completion.
	Model string

	// Choices holds the generated completions. Most requests return one choice, but the n
	// parameter can request more.
	Choices []Choice

	// Sources holds vector search results used for RAG context injection. This is nil when
	// the request did not use RAG, and empty when RAG was configured but no matching
	// documents were found.
	Sources []VectorSearchResult

	// Created is the Unix timestamp in seconds when the completion was made.
	Created int64
}

// Choice represents a single completion option from the model.
type Choice struct {
	// FinishReason indicates why the model stopped generating tokens.
	FinishReason FinishReason

	// Message holds the generated response content. It contains the text output and any tool
	// calls the model has requested.
	Message Message

	// Index is the position of this choice in the response, starting from 0.
	Index int
}

// FinishReason indicates why the model stopped generating tokens.
type FinishReason string

const (
	// FinishReasonStop indicates the model reached a natural stop point or a provided stop
	// sequence.
	FinishReasonStop FinishReason = "stop"

	// FinishReasonLength means the model stopped because it reached the maximum number of
	// tokens allowed.
	FinishReasonLength FinishReason = "length"

	// FinishReasonToolCalls means the model wants to call one or more tools.
	FinishReasonToolCalls FinishReason = "tool_calls"

	// FinishReasonContentFilter indicates the content was filtered due to policy.
	FinishReasonContentFilter FinishReason = "content_filter"
)

// Usage holds token usage figures for a completion request.
type Usage struct {
	// EstimatedCost holds the cost worked out for this request. The service layer fills this
	// in using the pricing table.
	EstimatedCost *CostEstimate

	// PromptTokens is the number of tokens in the prompt input.
	PromptTokens int

	// CompletionTokens is the number of tokens in the generated output.
	CompletionTokens int

	// TotalTokens is the total number of tokens used (prompt plus completion).
	TotalTokens int

	// CachedTokens is the number of prompt tokens served from the provider's prompt cache.
	// These are billed at a lower rate when the model supports it.
	CachedTokens int
}

// FirstChoice returns the first choice from the response, or an empty Choice if no
// choices are present. This is a convenience method since most requests return exactly
// one choice.
//
// Returns Choice which is the first choice, or an empty Choice if none exist.
func (r *CompletionResponse) FirstChoice() Choice {
	if len(r.Choices) == 0 {
		return Choice{}
	}
	return r.Choices[0]
}

// Content returns the text content from the first choice. This is a convenience method
// for the common case of extracting the assistant's response.
//
// Returns string which is the content of the first choice's message, or empty string if
// no choices exist.
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

// ToolCalls returns the tool calls from the first choice. This is a convenience method
// for extracting tool calls from the response.
//
// Returns []ToolCall which contains the tool calls, or nil if none exist.
func (r *CompletionResponse) ToolCalls() []ToolCall {
	if len(r.Choices) == 0 {
		return nil
	}
	return r.Choices[0].Message.ToolCalls
}

// Clone copies the response.
//
// Metadata maps on sources are copied one level deep. Nested values inside those maps
// remain shared.
//
// Returns *CompletionResponse which is the copy, or nil when r is nil.
func (r *CompletionResponse) Clone() *CompletionResponse {
	if r == nil {
		return nil
	}

	clone := *r
	clone.Usage = r.Usage.clone()
	clone.FallbackInfo = r.FallbackInfo.clone()
	clone.Choices = cloneChoices(r.Choices)
	clone.Sources = cloneSources(r.Sources)
	return &clone
}

// clone returns a copy of the usage, including its cost estimate.
//
// Returns *Usage which is the copy, or nil when u is nil.
func (u *Usage) clone() *Usage {
	if u == nil {
		return nil
	}

	clone := *u
	if u.EstimatedCost != nil {
		clone.EstimatedCost = new(*u.EstimatedCost)
	}
	return &clone
}

// clone returns a copy of the fallback result with its own map and slice.
//
// Returns *FallbackResult which is the copy, or nil when r is nil.
func (r *FallbackResult) clone() *FallbackResult {
	if r == nil {
		return nil
	}

	clone := *r
	clone.Errors = maps.Clone(r.Errors)
	clone.AttemptedProviders = slices.Clone(r.AttemptedProviders)
	return &clone
}

// clone returns a deep copy of the message.
//
// Returns Message which is the copy.
func (m Message) clone() Message {
	clone := m
	clone.Name = clonePointer(m.Name)
	clone.ToolCallID = clonePointer(m.ToolCallID)
	clone.ToolCalls = slices.Clone(m.ToolCalls)
	if m.ContentParts != nil {
		clone.ContentParts = make([]ContentPart, len(m.ContentParts))
		for index, part := range m.ContentParts {
			clone.ContentParts[index] = part.clone()
		}
	}
	return clone
}

// clone returns a deep copy of the content part.
//
// Returns ContentPart which is the copy.
func (p ContentPart) clone() ContentPart {
	clone := p
	clone.Text = clonePointer(p.Text)
	clone.ImageData = clonePointer(p.ImageData)
	if p.ImageURL != nil {
		imageURL := *p.ImageURL
		imageURL.Detail = clonePointer(p.ImageURL.Detail)
		clone.ImageURL = &imageURL
	}
	return clone
}

// cloneChoices returns a deep copy of the choices.
//
// Takes choices ([]Choice) which are the choices to copy.
//
// Returns []Choice which is the copy, or nil when choices is nil.
func cloneChoices(choices []Choice) []Choice {
	if choices == nil {
		return nil
	}

	clone := make([]Choice, len(choices))
	for index, choice := range choices {
		clone[index] = choice
		clone[index].Message = choice.Message.clone()
	}
	return clone
}

// cloneSources returns a copy of the sources with their own vectors and metadata maps.
//
// Takes sources ([]VectorSearchResult) which are the sources to copy.
//
// Returns []VectorSearchResult which is the copy, or nil when sources is nil.
func cloneSources(sources []VectorSearchResult) []VectorSearchResult {
	if sources == nil {
		return nil
	}

	clone := make([]VectorSearchResult, len(sources))
	for index, source := range sources {
		clone[index] = source
		clone[index].Metadata = maps.Clone(source.Metadata)
		clone[index].Vector = slices.Clone(source.Vector)
	}
	return clone
}

// clonePointer copies the value referenced by value.
//
// Takes value (*T) which is the pointer to copy.
//
// Returns *T which points to the copy, or nil when value is nil.
func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	return new(*value)
}
