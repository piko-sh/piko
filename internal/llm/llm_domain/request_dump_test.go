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

package llm_domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestRequestDump_String(t *testing.T) {
	dump := &RequestDump{
		Timestamp: time.Date(2026, 2, 18, 4, 30, 0, 0, time.UTC),
		Model:     "tinyllama",
		Provider:  "ollama",
		MaxTokens: new(500),
		Messages: []llm_dto.Message{
			llm_dto.NewSystemMessage("You are a helpful assistant."),
			llm_dto.NewUserMessage("How does caching work?"),
		},
		Sources: []llm_dto.VectorSearchResult{
			{
				ID:      "doc-chunk-3",
				Score:   0.8723,
				Content: "Piko's cache system supports multiple providers.",
				Metadata: map[string]any{
					"heading": "Cache System",
					"source":  "guide/caching.md",
					"section": "guide",
				},
			},
		},
	}

	out := dump.String()

	assert.Contains(t, out, "Model: tinyllama")
	assert.Contains(t, out, "Provider: ollama")
	assert.Contains(t, out, "MaxTokens: 500")
	assert.Contains(t, out, "2026-02-18T04:30:00Z")
	assert.Contains(t, out, "=== Messages (2) ===")
	assert.Contains(t, out, "--- system ---")
	assert.Contains(t, out, "You are a helpful assistant.")
	assert.Contains(t, out, "--- user ---")
	assert.Contains(t, out, "How does caching work?")
	assert.Contains(t, out, "=== Sources (1) ===")
	assert.Contains(t, out, "score=0.8723")
	assert.Contains(t, out, "id=doc-chunk-3")
	assert.Contains(t, out, "Heading: Cache System")
	assert.Contains(t, out, "File: guide/caching.md")
	assert.Contains(t, out, "Piko's cache system supports multiple providers.")
}

func TestRequestDump_NoSources(t *testing.T) {
	dump := &RequestDump{
		Timestamp: time.Now(),
		Model:     "gpt-4o",
		Provider:  "openai",
		Messages: []llm_dto.Message{
			llm_dto.NewUserMessage("Hello"),
		},
	}

	out := dump.String()

	assert.Contains(t, out, "Model: gpt-4o")
	assert.Contains(t, out, "=== Messages (1) ===")
	assert.NotContains(t, out, "=== Sources")
}

func TestRequestDump_WriteTo(t *testing.T) {
	dump := &RequestDump{
		Timestamp: time.Now(),
		Model:     "test",
		Provider:  "test",
		MaxTokens: new(100),
		Messages: []llm_dto.Message{
			llm_dto.NewUserMessage("test"),
		},
	}

	var buffer strings.Builder
	n, err := dump.WriteTo(&buffer)

	require.NoError(t, err)
	assert.Greater(t, n, int64(0))
	assert.Equal(t, int64(len(buffer.String())), n)
}

func TestRequestDump_Temperature(t *testing.T) {
	dump := &RequestDump{
		Timestamp:   time.Now(),
		Model:       "test",
		Provider:    "test",
		Temperature: new(0.7),
		Messages:    []llm_dto.Message{llm_dto.NewUserMessage("test")},
	}

	out := dump.String()

	assert.Contains(t, out, "Temperature: 0.70")
}

func TestRequestDump_OmitsUnsetFields(t *testing.T) {
	dump := &RequestDump{
		Timestamp: time.Now(),
		Model:     "test",
		Provider:  "test",
		Messages:  []llm_dto.Message{llm_dto.NewUserMessage("test")},
	}

	out := dump.String()

	assert.NotContains(t, out, "MaxTokens:")
	assert.NotContains(t, out, "Temperature:")
	assert.NotContains(t, out, "=== Sources")
	assert.NotContains(t, out, "=== Tools")
	assert.NotContains(t, out, "=== Query Rewriting")
}

func TestRequestDump_WithQueryRewriting(t *testing.T) {
	dump := &RequestDump{
		Timestamp:     time.Now(),
		Model:         "test",
		Provider:      "test",
		Messages:      []llm_dto.Message{llm_dto.NewUserMessage("test")},
		OriginalQuery: "give me an example pk file",
		RewrittenQueries: []string{
			"example .pk single file component template",
			"piko page Render function code example",
		},
	}

	out := dump.String()

	assert.Contains(t, out, "=== Query Rewriting ===")
	assert.Contains(t, out, "Original: give me an example pk file")
	assert.Contains(t, out, "Rewritten (2):")
	assert.Contains(t, out, "1. example .pk single file component template")
	assert.Contains(t, out, "2. piko page Render function code example")
}

func TestRequestDump_NoRewriting_OmitsSection(t *testing.T) {
	dump := &RequestDump{
		Timestamp:     time.Now(),
		Model:         "test",
		Provider:      "test",
		Messages:      []llm_dto.Message{llm_dto.NewUserMessage("test")},
		OriginalQuery: "some query",
	}

	out := dump.String()

	assert.NotContains(t, out, "=== Query Rewriting")
}

func TestMetaString_NilMap(t *testing.T) {
	result := metaString(nil, "key")

	assert.Equal(t, "", result)
}

func TestMetaString_MissingKey(t *testing.T) {
	result := metaString(map[string]any{"a": "b"}, "missing")

	assert.Equal(t, "", result)
}

func TestMetaString_NonStringValue(t *testing.T) {
	result := metaString(map[string]any{"key": 42}, "key")

	assert.Equal(t, "", result)
}

func TestMetaString_ValidKey(t *testing.T) {
	result := metaString(map[string]any{"key": "value"}, "key")

	assert.Equal(t, "value", result)
}
