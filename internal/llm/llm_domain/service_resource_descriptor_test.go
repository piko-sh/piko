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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/provider/provider_domain"
)

type metadataMockProvider struct {
	MockLLMProvider
	providerType string
	metadata     map[string]any
}

func (m *metadataMockProvider) GetProviderType() string             { return m.providerType }
func (m *metadataMockProvider) GetProviderMetadata() map[string]any { return m.metadata }

var _ provider_domain.ProviderMetadata = (*metadataMockProvider)(nil)

func newLLMService() *service {
	svc := NewService("")
	s, ok := svc.(*service)
	if !ok {
		panic("NewService did not return *service")
	}
	return s
}

func newLLMServiceWithDefault(name string) *service {
	svc := NewService(name)
	s, ok := svc.(*service)
	if !ok {
		panic("NewService did not return *service")
	}
	return s
}

func TestLLMService_ResourceType(t *testing.T) {
	t.Parallel()

	svc := newLLMService()
	assert.Equal(t, "llm", svc.ResourceType())
}

func TestLLMService_ResourceListColumns(t *testing.T) {
	t.Parallel()

	svc := newLLMService()
	cols := svc.ResourceListColumns()
	require.Len(t, cols, 6)
	assert.Equal(t, "NAME", cols[0].Header)
	assert.Equal(t, "name", cols[0].Key)
	assert.Equal(t, "TYPE", cols[1].Header)
	assert.Equal(t, "type", cols[1].Key)
	assert.Equal(t, "DEFAULT MODEL", cols[2].Header)
	assert.Equal(t, "default_model", cols[2].Key)
	assert.False(t, cols[2].WideOnly)
	assert.Equal(t, "STREAMING", cols[3].Header)
	assert.True(t, cols[3].WideOnly)
	assert.Equal(t, "TOOLS", cols[4].Header)
	assert.True(t, cols[4].WideOnly)
	assert.Equal(t, "STRUCTURED", cols[5].Header)
	assert.True(t, cols[5].WideOnly)
}

func TestLLMService_ResourceListProviders(t *testing.T) {
	t.Parallel()

	t.Run("empty returns empty", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()

		entries := svc.ResourceListProviders(context.Background())
		assert.Empty(t, entries)
	})

	t.Run("single provider", func(t *testing.T) {
		t.Parallel()
		svc := newLLMServiceWithDefault("openai")
		mp := &MockLLMProvider{
			DefaultModelValue:       "gpt-5",
			SupportsStreamingValue:  true,
			SupportsToolsValue:      true,
			SupportsStructuredValue: true,
		}
		require.NoError(t, svc.RegisterProvider(context.Background(), "openai", mp))

		entries := svc.ResourceListProviders(context.Background())
		require.Len(t, entries, 1)
		assert.Equal(t, "openai", entries[0].Name)
		assert.True(t, entries[0].IsDefault)
		assert.Equal(t, "gpt-5", entries[0].Values["default_model"])
		assert.Equal(t, "unknown", entries[0].Values["type"])
		assert.Equal(t, "true", entries[0].Values["streaming"])
		assert.Equal(t, "true", entries[0].Values["tools"])
		assert.Equal(t, "true", entries[0].Values["structured"])
	})

	t.Run("default is marked", func(t *testing.T) {
		t.Parallel()
		svc := newLLMServiceWithDefault("anthropic")
		require.NoError(t, svc.RegisterProvider(context.Background(), "openai", NewMockLLMProvider()))
		require.NoError(t, svc.RegisterProvider(context.Background(), "anthropic", NewMockLLMProvider()))

		entries := svc.ResourceListProviders(context.Background())
		require.Len(t, entries, 2)

		assert.Equal(t, "anthropic", entries[0].Name)
		assert.True(t, entries[0].IsDefault)
		assert.Equal(t, "openai", entries[1].Name)
		assert.False(t, entries[1].IsDefault)
	})

	t.Run("multiple providers are sorted", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()
		require.NoError(t, svc.RegisterProvider(context.Background(), "zebra", NewMockLLMProvider()))
		require.NoError(t, svc.RegisterProvider(context.Background(), "alpha", NewMockLLMProvider()))

		entries := svc.ResourceListProviders(context.Background())
		require.Len(t, entries, 2)
		assert.Equal(t, "alpha", entries[0].Name)
		assert.Equal(t, "zebra", entries[1].Name)
	})

	t.Run("with ProviderMetadata shows type", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()
		mp := &metadataMockProvider{
			MockLLMProvider: *NewMockLLMProvider(),
			providerType:    "anthropic",
		}
		require.NoError(t, svc.RegisterProvider(context.Background(), "claude", mp))

		entries := svc.ResourceListProviders(context.Background())
		require.Len(t, entries, 1)
		assert.Equal(t, "anthropic", entries[0].Values["type"])
	})
}

func TestLLMService_ResourceDescribeProvider(t *testing.T) {
	t.Parallel()

	t.Run("not found returns error", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()

		_, err := svc.ResourceDescribeProvider(context.Background(), "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns overview and capabilities sections", func(t *testing.T) {
		t.Parallel()
		svc := newLLMServiceWithDefault("test")
		mp := &MockLLMProvider{
			DefaultModelValue:              "test-model",
			SupportsStreamingValue:         true,
			SupportsStructuredValue:        true,
			SupportsToolsValue:             true,
			SupportsPenaltiesValue:         true,
			SupportsSeedValue:              false,
			SupportsParallelToolCallsValue: true,
			SupportsMessageNameValue:       false,
		}
		require.NoError(t, svc.RegisterProvider(context.Background(), "test", mp))

		detail, err := svc.ResourceDescribeProvider(context.Background(), "test")
		require.NoError(t, err)
		assert.Equal(t, "test", detail.Name)
		require.Len(t, detail.Sections, 2)

		overview := detail.Sections[0]
		assert.Equal(t, "Overview", overview.Title)
		assertLLMInfoEntry(t, overview.Entries, "Name", "test")
		assertLLMInfoEntry(t, overview.Entries, "Type", "unknown")
		assertLLMInfoEntry(t, overview.Entries, "Default", "true")
		assertLLMInfoEntry(t, overview.Entries, "Default Model", "test-model")

		capabilities := detail.Sections[1]
		assert.Equal(t, "Capabilities", capabilities.Title)
		assertLLMInfoEntry(t, capabilities.Entries, "Streaming", "true")
		assertLLMInfoEntry(t, capabilities.Entries, "Structured Output", "true")
		assertLLMInfoEntry(t, capabilities.Entries, "Tools", "true")
		assertLLMInfoEntry(t, capabilities.Entries, "Penalties", "true")
		assertLLMInfoEntry(t, capabilities.Entries, "Seed", "false")
		assertLLMInfoEntry(t, capabilities.Entries, "Parallel Tool Calls", "true")
		assertLLMInfoEntry(t, capabilities.Entries, "Message Name", "false")
	})

	t.Run("with ProviderMetadata adds Configuration section", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()
		mp := &metadataMockProvider{
			MockLLMProvider: *NewMockLLMProvider(),
			providerType:    "openai",
			metadata: map[string]any{
				"region":  "us-east-1",
				"api_key": "sk-***",
			},
		}
		require.NoError(t, svc.RegisterProvider(context.Background(), "prod", mp))

		detail, err := svc.ResourceDescribeProvider(context.Background(), "prod")
		require.NoError(t, err)
		require.Len(t, detail.Sections, 3)

		config := detail.Sections[2]
		assert.Equal(t, "Configuration", config.Title)

		assert.Equal(t, "api_key", config.Entries[0].Key)
		assert.Equal(t, "region", config.Entries[1].Key)
	})
}

func TestLLMService_SubResourceDescriptor(t *testing.T) {
	t.Parallel()

	t.Run("sub-resource name is models", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()
		assert.Equal(t, "models", svc.ResourceSubResourceName())
	})

	t.Run("sub-resource columns", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()
		cols := svc.ResourceSubResourceColumns()
		require.Len(t, cols, 3)
		assert.Equal(t, "MODEL", cols[0].Header)
		assert.Equal(t, "CONTEXT", cols[1].Header)
		assert.Equal(t, "MAX OUTPUT", cols[2].Header)
	})

	t.Run("lists models for provider", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()
		mp := &MockLLMProvider{
			ListModelsFunc: func(_ context.Context) ([]llm_dto.ModelInfo, error) {
				return []llm_dto.ModelInfo{
					{ID: "gpt-5", ContextWindow: 128000, MaxOutputTokens: 16384},
					{ID: "gpt-5-mini", ContextWindow: 128000, MaxOutputTokens: 4096},
				}, nil
			},
		}
		require.NoError(t, svc.RegisterProvider(context.Background(), "openai", mp))

		entries, err := svc.ResourceListSubResources(context.Background(), "openai")
		require.NoError(t, err)
		require.Len(t, entries, 2)
		assert.Equal(t, "gpt-5", entries[0].Name)
		assert.Equal(t, "128000", entries[0].Values["context"])
		assert.Equal(t, "16384", entries[0].Values["max_output"])
		assert.Equal(t, "gpt-5-mini", entries[1].Name)
		assert.Equal(t, "4096", entries[1].Values["max_output"])
	})

	t.Run("provider not found returns error", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()

		_, err := svc.ResourceListSubResources(context.Background(), "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ListModels error returns ErrNoSubResources", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()
		mp := &MockLLMProvider{
			ListModelsFunc: func(_ context.Context) ([]llm_dto.ModelInfo, error) {
				return nil, errors.New("api unreachable")
			},
		}
		require.NoError(t, svc.RegisterProvider(context.Background(), "broken", mp))

		_, err := svc.ResourceListSubResources(context.Background(), "broken")
		require.ErrorIs(t, err, provider_domain.ErrNoSubResources)
	})
}

func TestLLMService_ResourceDescribeType(t *testing.T) {
	t.Parallel()

	t.Run("no providers", func(t *testing.T) {
		t.Parallel()
		svc := newLLMService()

		detail := svc.ResourceDescribeType(context.Background())
		assert.Equal(t, "llm", detail.Name)
		require.Len(t, detail.Sections, 1)
		assertLLMInfoEntry(t, detail.Sections[0].Entries, "Resource Type", "llm")
		assertLLMInfoEntry(t, detail.Sections[0].Entries, "Completion Provider Count", "0")
		assertLLMInfoEntry(t, detail.Sections[0].Entries, "Default Completion Provider", "")
	})

	t.Run("with providers", func(t *testing.T) {
		t.Parallel()
		svc := newLLMServiceWithDefault("openai")
		require.NoError(t, svc.RegisterProvider(context.Background(), "openai", NewMockLLMProvider()))
		require.NoError(t, svc.RegisterProvider(context.Background(), "anthropic", NewMockLLMProvider()))

		detail := svc.ResourceDescribeType(context.Background())
		assertLLMInfoEntry(t, detail.Sections[0].Entries, "Completion Provider Count", "2")
		assertLLMInfoEntry(t, detail.Sections[0].Entries, "Default Completion Provider", "openai")
	})

	t.Run("with embedding providers", func(t *testing.T) {
		t.Parallel()
		svc := newLLMServiceWithDefault("openai")
		require.NoError(t, svc.RegisterProvider(context.Background(), "openai", NewMockLLMProvider()))

		embeddingProvider := &MockEmbeddingProvider{}
		require.NoError(t, svc.RegisterEmbeddingProvider(context.Background(), "voyage", embeddingProvider))
		require.NoError(t, svc.SetDefaultEmbeddingProvider("voyage"))

		detail := svc.ResourceDescribeType(context.Background())
		assertLLMInfoEntry(t, detail.Sections[0].Entries, "Embedding Provider Count", "1")
		assertLLMInfoEntry(t, detail.Sections[0].Entries, "Default Embedding Provider", "voyage")
	})
}

func TestLLMService_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ provider_domain.ResourceDescriptor = (*service)(nil)
	var _ provider_domain.SubResourceDescriptor = (*service)(nil)
	var _ provider_domain.ResourceTypeDescriptor = (*service)(nil)
}

func assertLLMInfoEntry(t *testing.T, entries []provider_domain.InfoEntry, key, expectedValue string) {
	t.Helper()
	for _, e := range entries {
		if e.Key == key {
			assert.Equal(t, expectedValue, e.Value, "entry %q", key)
			return
		}
	}
	t.Errorf("expected entry with key %q not found", key)
}
