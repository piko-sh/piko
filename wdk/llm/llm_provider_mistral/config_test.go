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
				APIKey:       "test-api-key",
				BaseURL:      "https://custom.mistral.ai",
				DefaultModel: "mistral-small-latest",
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

		assert.Equal(t, "mistral-large-latest", result.DefaultModel)
	})

	t.Run("applies default base URL", func(t *testing.T) {
		config := Config{
			APIKey: "test-key",
		}

		result := config.WithDefaults()

		assert.Equal(t, DefaultBaseURL, result.BaseURL)
	})

	t.Run("preserves custom model", func(t *testing.T) {
		config := Config{
			APIKey:       "test-key",
			DefaultModel: "mistral-small-latest",
		}

		result := config.WithDefaults()

		assert.Equal(t, "mistral-small-latest", result.DefaultModel)
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
}

func TestNew(t *testing.T) {
	t.Run("creates provider with valid config", func(t *testing.T) {
		config := Config{
			APIKey: "test-api-key",
		}

		provider, err := New(config)

		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("fails with invalid config", func(t *testing.T) {
		config := Config{}

		provider, err := New(config)

		require.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("applies defaults", func(t *testing.T) {
		config := Config{
			APIKey: "test-api-key",
		}

		provider, err := New(config)

		require.NoError(t, err)
		require.NotNil(t, provider)

		mp, ok := provider.(*mistralProvider)
		require.True(t, ok, "provider should be *mistralProvider")
		assert.Equal(t, "mistral-large-latest", mp.defaultModel)
		assert.Equal(t, DefaultBaseURL, mp.config.BaseURL)
	})
}

func TestMistralProvider_Capabilities(t *testing.T) {
	config := Config{
		APIKey: "test-api-key",
	}
	provider, err := New(config)
	require.NoError(t, err)

	t.Run("supports streaming", func(t *testing.T) {
		assert.True(t, provider.SupportsStreaming())
	})

	t.Run("supports structured output", func(t *testing.T) {
		assert.True(t, provider.SupportsStructuredOutput())
	})

	t.Run("supports tools", func(t *testing.T) {
		assert.True(t, provider.SupportsTools())
	})
}
