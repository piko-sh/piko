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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "valid config with API key",
			config: Config{
				APIKey: "test-api-key",
			},
			wantError: false,
		},
		{
			name:      "missing API key",
			config:    Config{},
			wantError: true,
		},
		{
			name: "empty API key",
			config: Config{
				APIKey: "",
			},
			wantError: true,
		},
		{
			name: "valid config with all fields",
			config: Config{
				APIKey:              "test-api-key",
				BaseURL:             "https://custom.voyageai.com",
				DefaultModel:        "voyage-code-3",
				EmbeddingDimensions: 512,
			},
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "API key is required")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	t.Run("applies default model", func(t *testing.T) {
		config := Config{
			APIKey: "test-key",
		}

		result := config.WithDefaults()

		assert.Equal(t, "voyage-3.5", result.DefaultModel)
	})

	t.Run("applies default base URL", func(t *testing.T) {
		config := Config{
			APIKey: "test-key",
		}

		result := config.WithDefaults()

		assert.Equal(t, DefaultBaseURL, result.BaseURL)
	})

	t.Run("applies default embedding dimensions from lookup table", func(t *testing.T) {
		config := Config{
			APIKey: "test-key",
		}

		result := config.WithDefaults()

		assert.Equal(t, 1024, result.EmbeddingDimensions)
	})

	t.Run("applies correct dimensions for different models", func(t *testing.T) {
		config := Config{
			APIKey:       "test-key",
			DefaultModel: "voyage-4-large",
		}

		result := config.WithDefaults()

		assert.Equal(t, 2048, result.EmbeddingDimensions)
	})

	t.Run("preserves custom model", func(t *testing.T) {
		config := Config{
			APIKey:       "test-key",
			DefaultModel: "voyage-code-3",
		}

		result := config.WithDefaults()

		assert.Equal(t, "voyage-code-3", result.DefaultModel)
	})

	t.Run("preserves custom base URL", func(t *testing.T) {
		customURL := "https://custom.example.com"
		config := Config{
			APIKey:  "test-key",
			BaseURL: customURL,
		}

		result := config.WithDefaults()

		assert.Equal(t, customURL, result.BaseURL)
	})

	t.Run("preserves custom dimensions", func(t *testing.T) {
		config := Config{
			APIKey:              "test-key",
			EmbeddingDimensions: 256,
		}

		result := config.WithDefaults()

		assert.Equal(t, 256, result.EmbeddingDimensions)
	})
}

func TestNewVoyageProvider(t *testing.T) {
	t.Run("creates provider with valid config", func(t *testing.T) {
		config := Config{
			APIKey: "test-api-key",
		}

		provider, err := NewVoyageProvider(config)

		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("fails with invalid config", func(t *testing.T) {
		config := Config{}

		provider, err := NewVoyageProvider(config)

		require.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("applies defaults", func(t *testing.T) {
		config := Config{
			APIKey: "test-api-key",
		}

		provider, err := NewVoyageProvider(config)

		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, "voyage-3.5", provider.defaultModel)
		assert.Equal(t, 1024, provider.embeddingDimensions)
		assert.Equal(t, DefaultBaseURL, provider.config.BaseURL)
	})
}
