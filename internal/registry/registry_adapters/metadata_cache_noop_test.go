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

package registry_adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestNewNoOpMetadataCache(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil instance", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()

		require.NotNil(t, cache)
	})

	t.Run("implements MetadataCache interface", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()

		require.Implements(t, (*registry_domain.MetadataCache)(nil), cache)
	})
}

func TestNoOpMetadataCache_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns nil artefact and nil error", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()

		artefact, err := cache.Get(context.Background(), "any-key")

		assert.Nil(t, artefact)
		assert.NoError(t, err)
	})

	t.Run("returns nil for every key", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()
		ctx := context.Background()

		for _, key := range []string{"alpha", "bravo", "charlie", ""} {
			artefact, err := cache.Get(ctx, key)
			assert.Nil(t, artefact)
			assert.NoError(t, err)
		}
	})
}

func TestNoOpMetadataCache_GetMultiple(t *testing.T) {
	t.Parallel()

	t.Run("returns nil hits and all IDs as misses", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()
		ids := []string{"id-1", "id-2", "id-3"}

		hits, misses := cache.GetMultiple(context.Background(), ids)

		assert.Nil(t, hits)
		assert.Equal(t, ids, misses)
	})

	t.Run("handles empty input", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()

		hits, misses := cache.GetMultiple(context.Background(), []string{})

		assert.Nil(t, hits)
		assert.Empty(t, misses)
	})

	t.Run("handles nil input", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()

		hits, misses := cache.GetMultiple(context.Background(), nil)

		assert.Nil(t, hits)
		assert.Nil(t, misses)
	})
}

func TestNoOpMetadataCache_Set(t *testing.T) {
	t.Parallel()

	t.Run("does not panic on set", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()
		artefact := &registry_dto.ArtefactMeta{ID: "test-id"}

		assert.NotPanics(t, func() {
			cache.Set(context.Background(), artefact)
		})
	})

	t.Run("set does not affect subsequent get", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()
		ctx := context.Background()
		artefact := &registry_dto.ArtefactMeta{ID: "cached-id"}

		cache.Set(ctx, artefact)

		result, err := cache.Get(ctx, "cached-id")

		assert.Nil(t, result)
		assert.NoError(t, err)
	})
}

func TestNoOpMetadataCache_SetMultiple(t *testing.T) {
	t.Parallel()

	t.Run("does not panic on set multiple", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()
		artefacts := []*registry_dto.ArtefactMeta{
			{ID: "a"},
			{ID: "b"},
		}

		assert.NotPanics(t, func() {
			cache.SetMultiple(context.Background(), artefacts)
		})
	})
}

func TestNoOpMetadataCache_Delete(t *testing.T) {
	t.Parallel()

	t.Run("does not panic on delete", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()

		assert.NotPanics(t, func() {
			cache.Delete(context.Background(), "any-key")
		})
	})
}

func TestNoOpMetadataCache_Close(t *testing.T) {
	t.Parallel()

	t.Run("returns nil error", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()

		err := cache.Close(context.Background())

		assert.NoError(t, err)
	})

	t.Run("multiple close calls return nil", func(t *testing.T) {
		t.Parallel()

		cache := NewNoOpMetadataCache()
		ctx := context.Background()

		assert.NoError(t, cache.Close(ctx))
		assert.NoError(t, cache.Close(ctx))
	})
}
