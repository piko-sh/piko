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

import "errors"

const (
	// DefaultMaxTokens is the default maximum output tokens when not specified
	// in requests. This balances response length with cost.
	DefaultMaxTokens = 8192

	// DefaultModel is the Claude model used when no model is set in a request.
	DefaultModel = "claude-sonnet-4-5-20250929"
)

// Config holds settings for the Anthropic provider.
type Config struct {
	// APIKey is the Anthropic API key. Required.
	APIKey string

	// BaseURL is a custom API endpoint. Leave empty to use the default endpoint.
	BaseURL string

	// DefaultModel is the model to use when not set in requests.
	// Defaults to "claude-sonnet-4-5-20250929" if empty.
	DefaultModel string

	// DefaultMaxTokens specifies the token limit when max_tokens is not provided.
	// Required by the Anthropic API; defaults to 4096.
	DefaultMaxTokens int
}

// Validate checks that the configuration is valid.
//
// Returns error when required fields are missing.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("anthropic: API key is required")
	}
	return nil
}

// WithDefaults returns a copy of the config with default values applied.
//
// Returns Config which has any empty fields set to their default values.
func (c Config) WithDefaults() Config {
	if c.DefaultModel == "" {
		c.DefaultModel = DefaultModel
	}
	if c.DefaultMaxTokens == 0 {
		c.DefaultMaxTokens = DefaultMaxTokens
	}
	return c
}
