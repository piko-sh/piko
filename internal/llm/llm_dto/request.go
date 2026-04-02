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

// CompletionRequest contains all parameters for requesting a completion from
// an LLM provider. All generation parameters are optional (nil means use
// provider defaults).
type CompletionRequest struct {
	// Seed sets a value for deterministic sampling when the provider supports it.
	Seed *int64

	// StreamOptions provides extra settings for streaming output.
	StreamOptions *StreamOptions

	// Temperature controls randomness in generation. Higher values (e.g., 0.8)
	// make output more random, lower values (e.g., 0.2) make it more deterministic.
	Temperature *float64

	// MaxTokens limits how many tokens the model can generate;
	// nil uses the default.
	MaxTokens *int

	// TopP controls nucleus sampling, where the model considers tokens
	// with top_p probability mass as an alternative to temperature.
	TopP *float64

	// FrequencyPenalty reduces repetition by penalising tokens based on how often
	// they appear in the text so far. Range: -2.0 to 2.0.
	FrequencyPenalty *float64

	// PresencePenalty reduces repetition by penalising tokens that have appeared
	// at all in the text so far. Range: -2.0 to 2.0.
	PresencePenalty *float64

	// Metadata holds key-value pairs for tracking and logging.
	Metadata map[string]string

	// ProviderOptions holds settings that are specific to each provider and not
	// covered by the standard fields. The keys and values depend on the provider.
	ProviderOptions map[string]any

	// ToolChoice controls how the model uses tools (auto, none,
	// required, or specific).
	ToolChoice *ToolChoice

	// ParallelToolCalls allows the model to make several tool calls at once.
	ParallelToolCalls *bool

	// ResponseFormat specifies the output format: text, JSON
	// object, or JSON schema.
	ResponseFormat *ResponseFormat

	// Model is the identifier of the model to use
	// (e.g. "gpt-4o", "claude-3-5-sonnet").
	Model string

	// Tools holds the function definitions the model may call during generation.
	Tools []ToolDefinition

	// Messages holds the conversation history to send to the model.
	Messages []Message

	// Stop contains sequences that tell the model to stop generating more tokens.
	Stop []string

	// Stream enables streaming responses when set to true.
	Stream bool
}

// StreamOptions configures streaming behaviour.
type StreamOptions struct {
	// IncludeUsage requests token usage data in the final stream event.
	IncludeUsage bool
}

// NewCompletionRequest creates a new CompletionRequest with
// the specified model.
//
// Takes model (string) which identifies the model to use.
//
// Returns *CompletionRequest ready for further configuration.
func NewCompletionRequest(model string) *CompletionRequest {
	return &CompletionRequest{
		Model: model,
	}
}
