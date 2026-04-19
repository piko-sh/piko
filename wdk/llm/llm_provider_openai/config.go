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

package llm_provider_openai

import "errors"

// Config holds settings for the OpenAI provider.
type Config struct {
	// APIKey is the OpenAI API key. Required.
	APIKey string

	// BaseURL is a custom API endpoint for Azure OpenAI or proxy services.
	// Leave empty to use the default OpenAI endpoint.
	BaseURL string

	// Organisation is the OpenAI organisation ID; optional.
	Organisation string

	// DefaultModel is the model to use when not given in requests.
	// Defaults to "gpt-4.1" if empty.
	DefaultModel string

	// DefaultEmbeddingModel is the embedding model to use when not specified
	// in a request. Defaults to "text-embedding-3-small" if empty.
	DefaultEmbeddingModel string

	// EmbeddingDimensions overrides the default vector dimension for the
	// configured embedding model. When zero, the dimension is resolved from
	// a built-in lookup table.
	EmbeddingDimensions int
}

// Validate reports whether the configuration is valid.
//
// Returns error when required fields are missing.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("openai: API key is required")
	}
	return nil
}

// openaiEmbeddingDimensions maps known OpenAI embedding models to their
// default vector dimensions.
var openaiEmbeddingDimensions = map[string]int{
	"text-embedding-3-small": 1536,
	"text-embedding-3-large": 3072,
	"text-embedding-ada-002": 1536,
}

// WithDefaults returns a copy of the config with default values applied.
//
// Returns Config with any missing fields set to their default values.
func (c Config) WithDefaults() Config {
	if c.DefaultModel == "" {
		c.DefaultModel = "gpt-4.1"
	}
	if c.DefaultEmbeddingModel == "" {
		c.DefaultEmbeddingModel = "text-embedding-3-small"
	}
	if c.EmbeddingDimensions == 0 {
		c.EmbeddingDimensions = openaiEmbeddingDimensions[c.DefaultEmbeddingModel]
	}
	return c
}
