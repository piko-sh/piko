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

package llm_provider_ollama

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("returns nil for empty config", func(t *testing.T) {
		config := Config{}

		err := config.Validate()

		require.NoError(t, err)
	})

	t.Run("returns nil for fully populated config", func(t *testing.T) {
		config := Config{
			Host:                  "http://custom:1234",
			DefaultModel:          Model("mistral"),
			DefaultEmbeddingModel: Model("nomic-embed-text"),
			AutoStart:             new(true),
			AutoPull:              new(false),
			BinaryPath:            "/usr/local/bin/ollama",
		}

		err := config.Validate()

		require.NoError(t, err)
	})
}

func TestConfig_WithDefaults(t *testing.T) {
	t.Run("applies default host", func(t *testing.T) {
		config := Config{}

		result := config.WithDefaults()

		assert.Equal(t, "http://localhost:11434", result.Host)
	})

	t.Run("applies default model", func(t *testing.T) {
		config := Config{}

		result := config.WithDefaults()

		assert.Equal(t, Model("llama3.2"), result.DefaultModel)
	})

	t.Run("applies default embedding model", func(t *testing.T) {
		config := Config{}

		result := config.WithDefaults()

		assert.Equal(t, Model("all-minilm"), result.DefaultEmbeddingModel)
	})

	t.Run("applies default auto start", func(t *testing.T) {
		config := Config{}

		result := config.WithDefaults()

		require.NotNil(t, result.AutoStart)
		assert.True(t, *result.AutoStart)
	})

	t.Run("applies default auto pull", func(t *testing.T) {
		config := Config{}

		result := config.WithDefaults()

		require.NotNil(t, result.AutoPull)
		assert.True(t, *result.AutoPull)
	})

	t.Run("does not set default binary path", func(t *testing.T) {
		config := Config{}

		result := config.WithDefaults()

		assert.Empty(t, result.BinaryPath)
	})

	t.Run("preserves custom host", func(t *testing.T) {
		config := Config{
			Host: "http://custom:9999",
		}

		result := config.WithDefaults()

		assert.Equal(t, "http://custom:9999", result.Host)
	})

	t.Run("preserves custom model", func(t *testing.T) {
		config := Config{
			DefaultModel: Model("mistral"),
		}

		result := config.WithDefaults()

		assert.Equal(t, Model("mistral"), result.DefaultModel)
	})

	t.Run("preserves custom embedding model", func(t *testing.T) {
		config := Config{
			DefaultEmbeddingModel: Model("nomic-embed-text"),
		}

		result := config.WithDefaults()

		assert.Equal(t, Model("nomic-embed-text"), result.DefaultEmbeddingModel)
	})

	t.Run("preserves custom auto start false", func(t *testing.T) {
		config := Config{
			AutoStart: new(false),
		}

		result := config.WithDefaults()

		require.NotNil(t, result.AutoStart)
		assert.False(t, *result.AutoStart)
	})

	t.Run("preserves custom auto pull false", func(t *testing.T) {
		config := Config{
			AutoPull: new(false),
		}

		result := config.WithDefaults()

		require.NotNil(t, result.AutoPull)
		assert.False(t, *result.AutoPull)
	})

	t.Run("preserves custom binary path", func(t *testing.T) {
		config := Config{
			BinaryPath: "/opt/ollama/bin/ollama",
		}

		result := config.WithDefaults()

		assert.Equal(t, "/opt/ollama/bin/ollama", result.BinaryPath)
	})

	t.Run("does not mutate original config", func(t *testing.T) {
		config := Config{}

		_ = config.WithDefaults()

		assert.Empty(t, config.Host)
		assert.True(t, config.DefaultModel.IsZero())
		assert.True(t, config.DefaultEmbeddingModel.IsZero())
		assert.Nil(t, config.AutoStart)
		assert.Nil(t, config.AutoPull)
	})
}

func TestModelRef(t *testing.T) {
	t.Run("Model creates ref without digest", func(t *testing.T) {
		ref := Model("llama3.2")

		assert.Equal(t, "llama3.2", ref.Name)
		assert.Empty(t, ref.Digest)
		assert.Equal(t, "llama3.2", ref.String())
		assert.False(t, ref.IsZero())
	})

	t.Run("ModelWithDigest creates ref with digest", func(t *testing.T) {
		ref := ModelWithDigest("llama3.2", "a8b0c5157701")

		assert.Equal(t, "llama3.2", ref.Name)
		assert.Equal(t, "a8b0c5157701", ref.Digest)
	})

	t.Run("zero value is zero", func(t *testing.T) {
		var ref ModelRef

		assert.True(t, ref.IsZero())
	})

	t.Run("verifyDigest passes when no digest set", func(t *testing.T) {
		ref := Model("llama3.2")

		err := ref.verifyDigest("sha256:abc123")

		require.NoError(t, err)
	})

	t.Run("verifyDigest passes on exact match", func(t *testing.T) {
		ref := ModelWithDigest("llama3.2", "a8b0c5157701")

		err := ref.verifyDigest("a8b0c5157701")

		require.NoError(t, err)
	})

	t.Run("verifyDigest passes on prefix match", func(t *testing.T) {
		ref := ModelWithDigest("llama3.2", "a8b0c5")

		err := ref.verifyDigest("a8b0c5157701deadbeef")

		require.NoError(t, err)
	})

	t.Run("verifyDigest passes when got is prefix of want", func(t *testing.T) {
		ref := ModelWithDigest("llama3.2", "a8b0c5157701deadbeef")

		err := ref.verifyDigest("a8b0c5157701")

		require.NoError(t, err)
	})

	t.Run("verifyDigest strips sha256 prefix", func(t *testing.T) {
		ref := ModelWithDigest("llama3.2", "sha256:a8b0c5157701")

		err := ref.verifyDigest("sha256:a8b0c5157701")

		require.NoError(t, err)
	})

	t.Run("verifyDigest fails on mismatch", func(t *testing.T) {
		ref := ModelWithDigest("llama3.2", "a8b0c5157701")

		err := ref.verifyDigest("deadbeef1234")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "digest mismatch")
		assert.Contains(t, err.Error(), "supply chain")
	})
}
