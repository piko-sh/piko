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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		p, err := New(Config{APIKey: "test-key"})
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, "grok-3", p.DefaultModel())
	})

	t.Run("missing API key", func(t *testing.T) {
		p, err := New(Config{})
		require.Error(t, err)
		assert.Nil(t, p)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("custom default model", func(t *testing.T) {
		p, err := New(Config{APIKey: "key", DefaultModel: "grok-4-fast-reasoning"})
		require.NoError(t, err)
		assert.Equal(t, "grok-4-fast-reasoning", p.DefaultModel())
	})
}

func TestGrokModels(t *testing.T) {
	models := grokModels()

	t.Run("returns expected number of models", func(t *testing.T) {
		assert.Len(t, models, 8)
	})

	t.Run("all models have provider set to grok", func(t *testing.T) {
		for _, m := range models {
			assert.Equal(t, "grok", m.Provider, "model %s has wrong provider", m.ID)
		}
	})

	t.Run("all models support streaming", func(t *testing.T) {
		for _, m := range models {
			assert.True(t, m.SupportsStreaming, "model %s should support streaming", m.ID)
		}
	})

	t.Run("all models support tools", func(t *testing.T) {
		for _, m := range models {
			assert.True(t, m.SupportsTools, "model %s should support tools", m.ID)
		}
	})

	t.Run("all models support structured output", func(t *testing.T) {
		for _, m := range models {
			assert.True(t, m.SupportsStructuredOutput, "model %s should support structured output", m.ID)
		}
	})

	t.Run("contains expected model IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for _, m := range models {
			ids[m.ID] = true
		}

		expected := []string{
			"grok-3",
			"grok-3-mini",
			"grok-4-0709",
			"grok-4-fast-reasoning",
			"grok-4-fast-non-reasoning",
			"grok-4-1-fast-reasoning",
			"grok-4-1-fast-non-reasoning",
			"grok-code-fast-1",
		}
		for _, id := range expected {
			assert.True(t, ids[id], "missing expected model: %s", id)
		}
	})
}

func TestIsGrokChatModel(t *testing.T) {
	testCases := []struct {
		id   string
		want bool
	}{
		{"grok-3", true},
		{"grok-3-mini", true},
		{"grok-4-fast-reasoning", true},
		{"grok-code-fast-1", true},
		{"gpt-4o", false},
		{"claude-3-opus", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.id, func(t *testing.T) {
			assert.Equal(t, tc.want, isGrokChatModel(tc.id))
		})
	}
}

func TestCapabilities(t *testing.T) {
	p, err := New(Config{APIKey: "test-key"})
	require.NoError(t, err)

	t.Run("supports streaming", func(t *testing.T) {
		assert.True(t, p.SupportsStreaming())
	})

	t.Run("supports tools", func(t *testing.T) {
		assert.True(t, p.SupportsTools())
	})

	t.Run("supports structured output", func(t *testing.T) {
		assert.True(t, p.SupportsStructuredOutput())
	})
}

func TestGrokProvider_CapabilityMethods(t *testing.T) {
	p, err := New(Config{APIKey: "test-key"})
	require.NoError(t, err)

	assert.Equal(t, true, p.SupportsPenalties())
	assert.Equal(t, true, p.SupportsSeed())
	assert.Equal(t, true, p.SupportsParallelToolCalls())
	assert.Equal(t, true, p.SupportsMessageName())
}
