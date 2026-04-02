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
	"errors"
	"go/ast"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestMockCollectionProvider_Name(t *testing.T) {
	t.Parallel()

	t.Run("nil NameFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		got := m.Name()

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.NameCallCount))
	})

	t.Run("delegates to NameFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
		}

		got := m.Name()

		assert.Equal(t, "markdown", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.NameCallCount))
	})
}

func TestMockCollectionProvider_Type(t *testing.T) {
	t.Parallel()

	t.Run("nil TypeFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		got := m.Type()

		assert.Equal(t, ProviderType(""), got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.TypeCallCount))
	})

	t.Run("delegates to TypeFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{
			TypeFunc: func() ProviderType { return "static" },
		}

		got := m.Type()

		assert.Equal(t, ProviderType("static"), got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.TypeCallCount))
	})
}

func TestMockCollectionProvider_DiscoverCollections(t *testing.T) {
	t.Parallel()

	t.Run("nil DiscoverCollectionsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		got, err := m.DiscoverCollections(context.Background(), collection_dto.ProviderConfig{})

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DiscoverCollectionsCallCount))
	})

	t.Run("delegates to DiscoverCollectionsFunc", func(t *testing.T) {
		t.Parallel()
		expected := []collection_dto.CollectionInfo{{Name: "blog"}}
		m := &MockCollectionProvider{
			DiscoverCollectionsFunc: func(_ context.Context, _ collection_dto.ProviderConfig) ([]collection_dto.CollectionInfo, error) {
				return expected, nil
			},
		}

		got, err := m.DiscoverCollections(context.Background(), collection_dto.ProviderConfig{})

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from DiscoverCollectionsFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("discovery failed")
		m := &MockCollectionProvider{
			DiscoverCollectionsFunc: func(_ context.Context, _ collection_dto.ProviderConfig) ([]collection_dto.CollectionInfo, error) {
				return nil, wantErr
			},
		}

		_, err := m.DiscoverCollections(context.Background(), collection_dto.ProviderConfig{})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCollectionProvider_ValidateTargetType(t *testing.T) {
	t.Parallel()

	t.Run("nil ValidateTargetTypeFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		err := m.ValidateTargetType(&ast.Ident{Name: "Post"})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ValidateTargetTypeCallCount))
	})

	t.Run("delegates to ValidateTargetTypeFunc", func(t *testing.T) {
		t.Parallel()
		var captured ast.Expr
		m := &MockCollectionProvider{
			ValidateTargetTypeFunc: func(targetType ast.Expr) error {
				captured = targetType
				return nil
			},
		}
		expression := &ast.Ident{Name: "Post"}

		err := m.ValidateTargetType(expression)

		assert.NoError(t, err)
		assert.Equal(t, expression, captured)
	})

	t.Run("propagates error from ValidateTargetTypeFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("invalid target type")
		m := &MockCollectionProvider{
			ValidateTargetTypeFunc: func(_ ast.Expr) error { return wantErr },
		}

		err := m.ValidateTargetType(&ast.Ident{Name: "X"})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCollectionProvider_FetchStaticContent(t *testing.T) {
	t.Parallel()

	t.Run("nil FetchStaticContentFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		got, err := m.FetchStaticContent(context.Background(), "blog")

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FetchStaticContentCallCount))
	})

	t.Run("delegates to FetchStaticContentFunc", func(t *testing.T) {
		t.Parallel()
		expected := []collection_dto.ContentItem{{Metadata: map[string]any{"title": "hello"}}}
		m := &MockCollectionProvider{
			FetchStaticContentFunc: func(_ context.Context, name string) ([]collection_dto.ContentItem, error) {
				assert.Equal(t, "blog", name)
				return expected, nil
			},
		}

		got, err := m.FetchStaticContent(context.Background(), "blog")

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from FetchStaticContentFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("fetch failed")
		m := &MockCollectionProvider{
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				return nil, wantErr
			},
		}

		_, err := m.FetchStaticContent(context.Background(), "blog")

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCollectionProvider_GenerateRuntimeFetcher(t *testing.T) {
	t.Parallel()

	t.Run("nil GenerateRuntimeFetcherFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		got, err := m.GenerateRuntimeFetcher(context.Background(), "blog", &ast.Ident{Name: "Post"}, collection_dto.FetchOptions{})

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GenerateRuntimeFetcherCallCount))
	})

	t.Run("delegates to GenerateRuntimeFetcherFunc", func(t *testing.T) {
		t.Parallel()
		expected := &collection_dto.RuntimeFetcherCode{}
		m := &MockCollectionProvider{
			GenerateRuntimeFetcherFunc: func(_ context.Context, name string, _ ast.Expr, _ collection_dto.FetchOptions) (*collection_dto.RuntimeFetcherCode, error) {
				assert.Equal(t, "blog", name)
				return expected, nil
			},
		}

		got, err := m.GenerateRuntimeFetcher(context.Background(), "blog", &ast.Ident{Name: "Post"}, collection_dto.FetchOptions{})

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from GenerateRuntimeFetcherFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("codegen failed")
		m := &MockCollectionProvider{
			GenerateRuntimeFetcherFunc: func(_ context.Context, _ string, _ ast.Expr, _ collection_dto.FetchOptions) (*collection_dto.RuntimeFetcherCode, error) {
				return nil, wantErr
			},
		}

		_, err := m.GenerateRuntimeFetcher(context.Background(), "blog", &ast.Ident{Name: "Post"}, collection_dto.FetchOptions{})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCollectionProvider_ComputeETag(t *testing.T) {
	t.Parallel()

	t.Run("nil ComputeETagFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		got, err := m.ComputeETag(context.Background(), "blog")

		assert.NoError(t, err)
		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ComputeETagCallCount))
	})

	t.Run("delegates to ComputeETagFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{
			ComputeETagFunc: func(_ context.Context, name string) (string, error) {
				return "etag-" + name, nil
			},
		}

		got, err := m.ComputeETag(context.Background(), "blog")

		require.NoError(t, err)
		assert.Equal(t, "etag-blog", got)
	})

	t.Run("propagates error from ComputeETagFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("etag computation failed")
		m := &MockCollectionProvider{
			ComputeETagFunc: func(_ context.Context, _ string) (string, error) {
				return "", wantErr
			},
		}

		_, err := m.ComputeETag(context.Background(), "blog")

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCollectionProvider_ValidateETag(t *testing.T) {
	t.Parallel()

	t.Run("nil ValidateETagFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		etag, changed, err := m.ValidateETag(context.Background(), "blog", "old-etag")

		assert.NoError(t, err)
		assert.Equal(t, "", etag)
		assert.False(t, changed)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ValidateETagCallCount))
	})

	t.Run("delegates to ValidateETagFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{
			ValidateETagFunc: func(_ context.Context, _ string, expected string) (string, bool, error) {
				return "new-etag", expected != "new-etag", nil
			},
		}

		etag, changed, err := m.ValidateETag(context.Background(), "blog", "old-etag")

		require.NoError(t, err)
		assert.Equal(t, "new-etag", etag)
		assert.True(t, changed)
	})

	t.Run("propagates error from ValidateETagFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("validation failed")
		m := &MockCollectionProvider{
			ValidateETagFunc: func(_ context.Context, _ string, _ string) (string, bool, error) {
				return "", false, wantErr
			},
		}

		_, _, err := m.ValidateETag(context.Background(), "blog", "old")

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCollectionProvider_GenerateRevalidator(t *testing.T) {
	t.Parallel()

	t.Run("nil GenerateRevalidatorFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionProvider{}

		got, err := m.GenerateRevalidator(context.Background(), "blog", &ast.Ident{Name: "Post"}, collection_dto.HybridConfig{})

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GenerateRevalidatorCallCount))
	})

	t.Run("delegates to GenerateRevalidatorFunc", func(t *testing.T) {
		t.Parallel()
		expected := &collection_dto.RuntimeFetcherCode{}
		m := &MockCollectionProvider{
			GenerateRevalidatorFunc: func(_ context.Context, _ string, _ ast.Expr, _ collection_dto.HybridConfig) (*collection_dto.RuntimeFetcherCode, error) {
				return expected, nil
			},
		}

		got, err := m.GenerateRevalidator(context.Background(), "blog", &ast.Ident{Name: "Post"}, collection_dto.HybridConfig{})

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from GenerateRevalidatorFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("revalidator generation failed")
		m := &MockCollectionProvider{
			GenerateRevalidatorFunc: func(_ context.Context, _ string, _ ast.Expr, _ collection_dto.HybridConfig) (*collection_dto.RuntimeFetcherCode, error) {
				return nil, wantErr
			},
		}

		_, err := m.GenerateRevalidator(context.Background(), "blog", &ast.Ident{Name: "Post"}, collection_dto.HybridConfig{})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCollectionProvider_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockCollectionProvider{}

	assert.Equal(t, "", m.Name())
	assert.Equal(t, ProviderType(""), m.Type())

	items, err := m.DiscoverCollections(context.Background(), collection_dto.ProviderConfig{})
	assert.NoError(t, err)
	assert.Nil(t, items)

	assert.NoError(t, m.ValidateTargetType(&ast.Ident{Name: "T"}))

	content, err := m.FetchStaticContent(context.Background(), "c")
	assert.NoError(t, err)
	assert.Nil(t, content)

	fetcher, err := m.GenerateRuntimeFetcher(context.Background(), "c", &ast.Ident{Name: "T"}, collection_dto.FetchOptions{})
	assert.NoError(t, err)
	assert.Nil(t, fetcher)

	etag, err := m.ComputeETag(context.Background(), "c")
	assert.NoError(t, err)
	assert.Equal(t, "", etag)

	currentETag, changed, err := m.ValidateETag(context.Background(), "c", "e")
	assert.NoError(t, err)
	assert.Equal(t, "", currentETag)
	assert.False(t, changed)

	revalidator, err := m.GenerateRevalidator(context.Background(), "c", &ast.Ident{Name: "T"}, collection_dto.HybridConfig{})
	assert.NoError(t, err)
	assert.Nil(t, revalidator)
}

func TestMockCollectionProvider_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockCollectionProvider{
		NameFunc: func() string { return "md" },
		TypeFunc: func() ProviderType { return "static" },
		DiscoverCollectionsFunc: func(_ context.Context, _ collection_dto.ProviderConfig) ([]collection_dto.CollectionInfo, error) {
			return nil, nil
		},
		ValidateTargetTypeFunc: func(_ ast.Expr) error { return nil },
		FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) { return nil, nil },
		GenerateRuntimeFetcherFunc: func(_ context.Context, _ string, _ ast.Expr, _ collection_dto.FetchOptions) (*collection_dto.RuntimeFetcherCode, error) {
			return nil, nil
		},
		ComputeETagFunc:  func(_ context.Context, _ string) (string, error) { return "", nil },
		ValidateETagFunc: func(_ context.Context, _ string, _ string) (string, bool, error) { return "", false, nil },
		GenerateRevalidatorFunc: func(_ context.Context, _ string, _ ast.Expr, _ collection_dto.HybridConfig) (*collection_dto.RuntimeFetcherCode, error) {
			return nil, nil
		},
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			m.Name()
			m.Type()
			_, _ = m.DiscoverCollections(context.Background(), collection_dto.ProviderConfig{})
			_ = m.ValidateTargetType(&ast.Ident{Name: "T"})
			_, _ = m.FetchStaticContent(context.Background(), "c")
			_, _ = m.GenerateRuntimeFetcher(context.Background(), "c", &ast.Ident{Name: "T"}, collection_dto.FetchOptions{})
			_, _ = m.ComputeETag(context.Background(), "c")
			_, _, _ = m.ValidateETag(context.Background(), "c", "e")
			_, _ = m.GenerateRevalidator(context.Background(), "c", &ast.Ident{Name: "T"}, collection_dto.HybridConfig{})
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.NameCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.TypeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DiscoverCollectionsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ValidateTargetTypeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.FetchStaticContentCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GenerateRuntimeFetcherCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ComputeETagCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ValidateETagCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GenerateRevalidatorCallCount))
}
