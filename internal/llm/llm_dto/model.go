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

// ModelInfo contains metadata about an available LLM model.
type ModelInfo struct {
	// ID is the unique identifier for the model (e.g. "gpt-5",
	// "claude-sonnet-4-5-20250929").
	ID string

	// Name is the display name for the model that people can read.
	Name string

	// Provider is the name of the provider (e.g. "openai", "anthropic", "gemini").
	Provider string

	// Created is the Unix timestamp when the model was created.
	Created int64

	// ContextWindow is the maximum context length in tokens.
	ContextWindow int

	// MaxOutputTokens is the largest number of tokens the model can produce.
	MaxOutputTokens int

	// SupportsStreaming indicates whether the model supports streaming responses.
	SupportsStreaming bool

	// SupportsTools indicates whether the model supports tool/function calling.
	SupportsTools bool

	// SupportsStructuredOutput indicates whether the model supports JSON schema output.
	SupportsStructuredOutput bool

	// SupportsVision indicates whether the model can process images.
	SupportsVision bool
}

// NewFullCapabilityModelInfo creates a ModelInfo for a generation model that supports
// streaming, tool calls, structured output and vision.
//
// Takes id (string) which is the provider's model identifier.
// Takes name (string) which is the human-readable model name.
// Takes provider (string) which is the name of the provider serving the model.
// Takes contextWindow (int) which is the maximum context size in tokens.
// Takes maxOutputTokens (int) which is the maximum number of tokens per response.
//
// Returns ModelInfo which describes the model.
func NewFullCapabilityModelInfo(id, name, provider string, contextWindow, maxOutputTokens int) ModelInfo {
	return ModelInfo{
		ID:                       id,
		Name:                     name,
		Provider:                 provider,
		Created:                  0,
		ContextWindow:            contextWindow,
		MaxOutputTokens:          maxOutputTokens,
		SupportsStreaming:        true,
		SupportsTools:            true,
		SupportsStructuredOutput: true,
		SupportsVision:           true,
	}
}

// NewEmbeddingModelInfo creates a ModelInfo for an embedding model, named after its
// identifier and reporting no context window, output limit or generation capabilities.
//
// Takes id (string) which is the provider's model identifier, also used as its name.
// Takes provider (string) which is the name of the provider serving the model.
// Takes created (int64) which is the model's creation time as a Unix timestamp, or 0 when
// unknown.
//
// Returns ModelInfo which describes the embedding model.
func NewEmbeddingModelInfo(id, provider string, created int64) ModelInfo {
	return ModelInfo{
		ID:                       id,
		Name:                     id,
		Provider:                 provider,
		Created:                  created,
		ContextWindow:            0,
		MaxOutputTokens:          0,
		SupportsStreaming:        false,
		SupportsTools:            false,
		SupportsStructuredOutput: false,
		SupportsVision:           false,
	}
}
