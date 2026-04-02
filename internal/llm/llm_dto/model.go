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
	// ID is the unique identifier for the model
	// (e.g. "gpt-5", "claude-sonnet-4-5-20250929").
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

	// SupportsStructuredOutput indicates whether the model
	// supports JSON schema output.
	SupportsStructuredOutput bool

	// SupportsVision indicates whether the model can process images.
	SupportsVision bool
}
