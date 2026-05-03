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

package llm_provider_grok

import "errors"

// DefaultBaseURL is the default xAI Grok API endpoint.
const DefaultBaseURL = "https://api.x.ai/v1"

// Config holds settings for the Grok provider.
type Config struct {
	// APIKey is the xAI API key. Required.
	APIKey string

	// BaseURL is a custom API endpoint. Leave empty to use the default
	// xAI endpoint (https://api.x.ai/v1).
	BaseURL string

	// DefaultModel is the model to use when not given in requests.
	// Defaults to "grok-3" if empty.
	DefaultModel string
}

// Validate reports whether the configuration is valid.
//
// Returns error when required fields are missing.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("grok: API key is required")
	}
	return nil
}

// WithDefaults returns a copy of the config with default values applied.
//
// Returns Config with any missing fields set to their default values.
func (c Config) WithDefaults() Config {
	if c.DefaultModel == "" {
		c.DefaultModel = "grok-3"
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	return c
}
