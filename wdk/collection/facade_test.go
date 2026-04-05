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

package collection

import (
	"context"
	"errors"
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_dto"
)

type mockSimpleProvider struct {
	fetchErr    error
	name        string
	providerTyp ProviderType
	etag        string
	collections []CollectionInfo
	items       []ContentItem
}

func (m *mockSimpleProvider) Name() string       { return m.name }
func (m *mockSimpleProvider) Type() ProviderType { return m.providerTyp }

func (m *mockSimpleProvider) DiscoverCollections(_ context.Context, _ ProviderConfig) ([]CollectionInfo, error) {
	return m.collections, nil
}

func (m *mockSimpleProvider) FetchContent(_ context.Context, _ string, _ *FetchOptions) ([]ContentItem, error) {
	return m.items, m.fetchErr
}

func (m *mockSimpleProvider) ComputeETag(_ context.Context, _ string) (string, error) {
	return m.etag, nil
}

func (m *mockSimpleProvider) ValidateETag(_ context.Context, _ string, expectedETag string) (string, bool, error) {
	changed := m.etag != expectedETag
	return m.etag, changed, nil
}

func TestNewSimpleProviderAdapter(t *testing.T) {
	t.Parallel()

	provider := &mockSimpleProvider{name: "test"}
	adapter := NewSimpleProviderAdapter(provider)

	require.NotNil(t, adapter)
}

func TestSimpleProviderAdapter_Name(t *testing.T) {
	t.Parallel()

	provider := &mockSimpleProvider{name: "my-provider"}
	adapter := NewSimpleProviderAdapter(provider)

	assert.Equal(t, "my-provider", adapter.Name())
}

func TestSimpleProviderAdapter_Type(t *testing.T) {
	t.Parallel()

	provider := &mockSimpleProvider{providerTyp: ProviderTypeStatic}
	adapter := NewSimpleProviderAdapter(provider)

	assert.Equal(t, ProviderTypeStatic, adapter.Type())
}

func TestSimpleProviderAdapter_DiscoverCollections(t *testing.T) {
	t.Parallel()

	collections := []CollectionInfo{{Name: "posts"}}
	provider := &mockSimpleProvider{collections: collections}
	adapter := NewSimpleProviderAdapter(provider)

	result, err := adapter.DiscoverCollections(context.Background(), ProviderConfig{})

	require.NoError(t, err)
	assert.Equal(t, collections, result)
}

func TestSimpleProviderAdapter_ValidateTargetType(t *testing.T) {
	t.Parallel()

	adapter := NewSimpleProviderAdapter(&mockSimpleProvider{})

	err := adapter.ValidateTargetType(&ast.Ident{Name: "MyStruct"})

	assert.NoError(t, err)
}

func TestSimpleProviderAdapter_FetchStaticContent(t *testing.T) {
	t.Parallel()

	items := []ContentItem{{Slug: "hello"}}
	provider := &mockSimpleProvider{items: items}
	adapter := NewSimpleProviderAdapter(provider)

	result, err := adapter.FetchStaticContent(context.Background(), "posts", collection_dto.ContentSource{})

	require.NoError(t, err)
	assert.Equal(t, items, result)
}

func TestSimpleProviderAdapter_GenerateRuntimeFetcher(t *testing.T) {
	t.Parallel()

	adapter := NewSimpleProviderAdapter(&mockSimpleProvider{})

	code, err := adapter.GenerateRuntimeFetcher(context.Background(), "posts", nil, FetchOptions{})

	assert.Nil(t, code)
	assert.ErrorIs(t, err, ErrCodeGenerationNotSupported)
}

func TestSimpleProviderAdapter_ComputeETag(t *testing.T) {
	t.Parallel()

	provider := &mockSimpleProvider{etag: "abc123"}
	adapter := NewSimpleProviderAdapter(provider)

	etag, err := adapter.ComputeETag(context.Background(), "posts", collection_dto.ContentSource{})

	require.NoError(t, err)
	assert.Equal(t, "abc123", etag)
}

func TestSimpleProviderAdapter_ValidateETag(t *testing.T) {
	t.Parallel()

	provider := &mockSimpleProvider{etag: "v2"}
	adapter := NewSimpleProviderAdapter(provider)

	t.Run("changed", func(t *testing.T) {
		t.Parallel()

		currentETag, changed, err := adapter.ValidateETag(context.Background(), "posts", "v1", collection_dto.ContentSource{})

		require.NoError(t, err)
		assert.Equal(t, "v2", currentETag)
		assert.True(t, changed)
	})

	t.Run("not changed", func(t *testing.T) {
		t.Parallel()

		currentETag, changed, err := adapter.ValidateETag(context.Background(), "posts", "v2", collection_dto.ContentSource{})

		require.NoError(t, err)
		assert.Equal(t, "v2", currentETag)
		assert.False(t, changed)
	})
}

func TestSimpleProviderAdapter_GenerateRevalidator(t *testing.T) {
	t.Parallel()

	adapter := NewSimpleProviderAdapter(&mockSimpleProvider{})

	code, err := adapter.GenerateRevalidator(context.Background(), "posts", nil, HybridConfig{})

	assert.Nil(t, code)
	assert.ErrorIs(t, err, ErrCodeGenerationNotSupported)
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("ErrCodeGenerationNotSupported", func(t *testing.T) {
		t.Parallel()
		assert.True(t, errors.Is(ErrCodeGenerationNotSupported, ErrCodeGenerationNotSupported))
		assert.Contains(t, ErrCodeGenerationNotSupported.Error(), "code generation")
	})

	t.Run("ErrProviderNotFound", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, ErrProviderNotFound.Error(), "provider not found")
	})

	t.Run("ErrCollectionNotFound", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, ErrCollectionNotFound.Error(), "collection not found")
	})

	t.Run("ErrETagNotSupported", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, ErrETagNotSupported.Error(), "ETag")
	})
}

var _ SimpleProvider = (*mockSimpleProvider)(nil)
