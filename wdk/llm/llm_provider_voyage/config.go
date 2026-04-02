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

package llm_provider_voyage

import "errors"

// DefaultBaseURL is the default Voyage AI API endpoint.
const DefaultBaseURL = "https://api.voyageai.com"

// Config holds configuration for the Voyage AI embedding provider.
type Config struct {
	// APIKey is the Voyage AI API key. Required.
	APIKey string

	// BaseURL specifies a custom API endpoint. Leave empty to use the
	// default Voyage AI endpoint.
	BaseURL string

	// DefaultModel is the embedding model to use when not specified in a
	// request. Defaults to "voyage-3.5" if empty.
	DefaultModel string

	// EmbeddingDimensions overrides the default vector dimension for the
	// configured model. When zero, the dimension is resolved from a
	// built-in lookup table.
	EmbeddingDimensions int
}

// voyageEmbeddingDimensions maps known Voyage models to their default vector
// dimensions.
var voyageEmbeddingDimensions = map[string]int{
	"voyage-3.5":       1024,
	"voyage-3.5-lite":  512,
	"voyage-4":         1024,
	"voyage-4-lite":    512,
	"voyage-4-large":   2048,
	"voyage-3-large":   1024,
	"voyage-code-3":    1024,
	"voyage-finance-2": 1024,
	"voyage-law-2":     1024,
}

// Validate checks that the configuration is valid.
//
// Returns error when the API key is missing.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("voyage: API key is required")
	}
	return nil
}

// WithDefaults returns a copy of the config with default values applied.
//
// Returns Config with any empty fields set to their default values.
func (c Config) WithDefaults() Config {
	if c.DefaultModel == "" {
		c.DefaultModel = "voyage-3.5"
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.EmbeddingDimensions == 0 {
		c.EmbeddingDimensions = voyageEmbeddingDimensions[c.DefaultModel]
	}
	return c
}
