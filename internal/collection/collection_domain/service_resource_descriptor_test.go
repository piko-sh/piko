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

package collection_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/provider/provider_domain"
)

type metadataProvider struct {
	metadata map[string]any
	MockCollectionProvider
}

func (m *metadataProvider) GetProviderType() string             { return "test" }
func (m *metadataProvider) GetProviderMetadata() map[string]any { return m.metadata }

var _ provider_domain.ProviderMetadata = (*metadataProvider)(nil)

func TestCollectionService_ResourceType(t *testing.T) {
	t.Parallel()

	registry := newTestProviderRegistry()
	service := NewCollectionService(context.Background(), registry)
	cs := mustCastToCollectionService(t, service)

	assert.Equal(t, "collection", cs.ResourceType())
}

func TestCollectionService_ResourceListColumns(t *testing.T) {
	t.Parallel()

	registry := newTestProviderRegistry()
	service := NewCollectionService(context.Background(), registry)
	cs := mustCastToCollectionService(t, service)

	cols := cs.ResourceListColumns()
	require.Len(t, cols, 2)
	assert.Equal(t, "NAME", cols[0].Header)
	assert.Equal(t, "name", cols[0].Key)
	assert.Equal(t, "TYPE", cols[1].Header)
	assert.Equal(t, "type", cols[1].Key)
}

func TestCollectionService_ResourceListProviders(t *testing.T) {
	t.Parallel()

	t.Run("empty registry returns empty", func(t *testing.T) {
		t.Parallel()
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)
		cs := mustCastToCollectionService(t, service)

		entries := cs.ResourceListProviders(context.Background())
		assert.Empty(t, entries)
	})

	t.Run("single provider", func(t *testing.T) {
		t.Parallel()
		registry := newTestProviderRegistry()
		_ = registry.Register(&MockCollectionProvider{
			NameFunc: func() string { return "blog" },
			TypeFunc: func() ProviderType { return "filesystem" },
		})
		service := NewCollectionService(context.Background(), registry)
		cs := mustCastToCollectionService(t, service)

		entries := cs.ResourceListProviders(context.Background())
		require.Len(t, entries, 1)
		assert.Equal(t, "blog", entries[0].Name)
		assert.Equal(t, "blog", entries[0].Values["name"])
		assert.Equal(t, "filesystem", entries[0].Values["type"])
		assert.False(t, entries[0].IsDefault)
	})

	t.Run("multiple providers are sorted", func(t *testing.T) {
		t.Parallel()
		registry := newTestProviderRegistry()
		_ = registry.Register(&MockCollectionProvider{
			NameFunc: func() string { return "zebra" },
			TypeFunc: func() ProviderType { return "api" },
		})
		_ = registry.Register(&MockCollectionProvider{
			NameFunc: func() string { return "alpha" },
			TypeFunc: func() ProviderType { return "filesystem" },
		})
		service := NewCollectionService(context.Background(), registry)
		cs := mustCastToCollectionService(t, service)

		entries := cs.ResourceListProviders(context.Background())
		require.Len(t, entries, 2)
		assert.Equal(t, "alpha", entries[0].Name)
		assert.Equal(t, "zebra", entries[1].Name)
	})

	t.Run("Get returns false skips provider", func(t *testing.T) {
		t.Parallel()
		registry := &MockProviderRegistry{
			ListFunc: func() []string { return []string{"ghost"} },
			GetFunc: func(_ string) (CollectionProvider, bool) {
				return nil, false
			},
		}
		service := NewCollectionService(context.Background(), registry)
		cs := mustCastToCollectionService(t, service)

		entries := cs.ResourceListProviders(context.Background())
		assert.Empty(t, entries)
	})
}

func TestCollectionService_ResourceDescribeProvider(t *testing.T) {
	t.Parallel()

	t.Run("not found returns error", func(t *testing.T) {
		t.Parallel()
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)
		cs := mustCastToCollectionService(t, service)

		_, err := cs.ResourceDescribeProvider(context.Background(), "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("basic detail has Overview section", func(t *testing.T) {
		t.Parallel()
		registry := newTestProviderRegistry()
		_ = registry.Register(&MockCollectionProvider{
			NameFunc: func() string { return "posts" },
			TypeFunc: func() ProviderType { return "filesystem" },
		})
		service := NewCollectionService(context.Background(), registry)
		cs := mustCastToCollectionService(t, service)

		detail, err := cs.ResourceDescribeProvider(context.Background(), "posts")
		require.NoError(t, err)
		assert.Equal(t, "posts", detail.Name)
		require.Len(t, detail.Sections, 1)
		assert.Equal(t, "Overview", detail.Sections[0].Title)

		found := false
		for _, e := range detail.Sections[0].Entries {
			if e.Key == "Name" && e.Value == "posts" {
				found = true
			}
		}
		assert.True(t, found, "expected Name=posts entry in Overview")
	})

	t.Run("with ProviderMetadata adds Configuration section", func(t *testing.T) {
		t.Parallel()
		registry := newTestProviderRegistry()
		mp := &metadataProvider{
			MockCollectionProvider: MockCollectionProvider{
				NameFunc: func() string { return "docs" },
				TypeFunc: func() ProviderType { return "api" },
			},
			metadata: map[string]any{
				"region": "eu-west-1",
			},
		}
		_ = registry.Register(mp)
		service := NewCollectionService(context.Background(), registry)
		cs := mustCastToCollectionService(t, service)

		detail, err := cs.ResourceDescribeProvider(context.Background(), "docs")
		require.NoError(t, err)
		require.Len(t, detail.Sections, 2)
		assert.Equal(t, "Configuration", detail.Sections[1].Title)
		assert.Equal(t, "region", detail.Sections[1].Entries[0].Key)
		assert.Equal(t, "eu-west-1", detail.Sections[1].Entries[0].Value)
	})
}
