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

package llm_provider_gemini

import "errors"

// Config holds configuration for the Gemini provider.
type Config struct {
	// APIKey is the Google AI API key. Required.
	APIKey string

	// DefaultModel is the model to use when not specified in requests.
	// Defaults to "gemini-2.5-flash" if empty.
	DefaultModel string

	// DefaultEmbeddingModel is the embedding model to use when not specified
	// in a request. Defaults to "text-embedding-004" if empty.
	DefaultEmbeddingModel string

	// EmbeddingDimensions overrides the default vector dimension for the
	// configured embedding model. When zero, the dimension is resolved from
	// a built-in lookup table.
	EmbeddingDimensions int
}

// Validate checks that the configuration is valid.
//
// Returns error when required fields are missing.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("gemini: API key is required")
	}
	return nil
}

// geminiEmbeddingDimensions maps known Gemini embedding models to their
// default vector dimensions.
var geminiEmbeddingDimensions = map[string]int{
	"text-embedding-004": 768,
}

// WithDefaults returns a copy of the configuration with default values set.
//
// Returns Config with any empty fields set to their default values.
func (c Config) WithDefaults() Config {
	if c.DefaultModel == "" {
		c.DefaultModel = "gemini-2.5-flash"
	}
	if c.DefaultEmbeddingModel == "" {
		c.DefaultEmbeddingModel = "text-embedding-004"
	}
	if c.EmbeddingDimensions == 0 {
		c.EmbeddingDimensions = geminiEmbeddingDimensions[c.DefaultEmbeddingModel]
	}
	return c
}
