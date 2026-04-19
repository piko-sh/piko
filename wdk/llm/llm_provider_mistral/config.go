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

package llm_provider_mistral

import "errors"

// DefaultBaseURL is the default Mistral API endpoint.
const DefaultBaseURL = "https://api.mistral.ai"

// Config holds configuration for the Mistral provider.
type Config struct {
	// APIKey is the Mistral API key. Required.
	APIKey string

	// BaseURL specifies a custom API endpoint for self-hosted deployments or proxy
	// services. Leave empty to use the default Mistral endpoint.
	BaseURL string

	// DefaultModel is the model to use when not specified in requests.
	// Defaults to "mistral-large-latest" if empty.
	DefaultModel string

	// DefaultEmbeddingModel is the embedding model to use when not specified
	// in a request. Defaults to "mistral-embed" if empty.
	DefaultEmbeddingModel string

	// EmbeddingDimensions overrides the default vector dimension for the
	// configured embedding model. When zero, the dimension is resolved from
	// a built-in lookup table.
	EmbeddingDimensions int
}

// Validate reports whether the configuration is valid.
//
// Returns error when the API key is missing.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("mistral: API key is required")
	}
	return nil
}

// mistralEmbeddingDimensions maps known Mistral embedding models to their
// default vector dimensions.
var mistralEmbeddingDimensions = map[string]int{
	"mistral-embed": 1024,
}

// WithDefaults returns a copy of the config with default values applied.
//
// Returns Config with any empty fields set to their default values.
func (c Config) WithDefaults() Config {
	if c.DefaultModel == "" {
		c.DefaultModel = "mistral-large-latest"
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.DefaultEmbeddingModel == "" {
		c.DefaultEmbeddingModel = "mistral-embed"
	}
	if c.EmbeddingDimensions == 0 {
		c.EmbeddingDimensions = mistralEmbeddingDimensions[c.DefaultEmbeddingModel]
	}
	return c
}
