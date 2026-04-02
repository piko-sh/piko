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

package registry_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
)

func TestMockMetadataCache_Get(t *testing.T) {
	t.Parallel()

	t.Run("nil GetFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataCache{}

		got, err := m.Get(context.Background(), "art-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCallCount))
	})

	t.Run("delegates to GetFunc", func(t *testing.T) {
		t.Parallel()
		want := &registry_dto.ArtefactMeta{ID: "art-1"}
		m := &MockMetadataCache{
			GetFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, "art-1", artefactID)
				return want, nil
			},
		}

		got, err := m.Get(context.Background(), "art-1")

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCallCount))
	})

	t.Run("propagates error from GetFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("cache miss")
		m := &MockMetadataCache{
			GetFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.Get(context.Background(), "art-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataCache_GetMultiple(t *testing.T) {
	t.Parallel()

	t.Run("nil GetMultipleFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataCache{}

		got, misses := m.GetMultiple(context.Background(), []string{"a", "b"})

		assert.Nil(t, got)
		assert.Nil(t, misses)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMultipleCallCount))
	})

	t.Run("delegates to GetMultipleFunc", func(t *testing.T) {
		t.Parallel()
		wantHits := []*registry_dto.ArtefactMeta{{ID: "a"}}
		wantMisses := []string{"b"}
		m := &MockMetadataCache{
			GetMultipleFunc: func(_ context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, []string) {
				assert.Equal(t, []string{"a", "b"}, artefactIDs)
				return wantHits, wantMisses
			},
		}

		got, misses := m.GetMultiple(context.Background(), []string{"a", "b"})

		assert.Equal(t, wantHits, got)
		assert.Equal(t, wantMisses, misses)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMultipleCallCount))
	})
}

func TestMockMetadataCache_Set(t *testing.T) {
	t.Parallel()

	t.Run("nil SetFunc is safe to call", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataCache{}

		m.Set(context.Background(), &registry_dto.ArtefactMeta{ID: "art-1"})

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SetCallCount))
	})

	t.Run("delegates to SetFunc", func(t *testing.T) {
		t.Parallel()
		art := &registry_dto.ArtefactMeta{ID: "art-1"}
		m := &MockMetadataCache{
			SetFunc: func(_ context.Context, artefact *registry_dto.ArtefactMeta) {
				assert.Same(t, art, artefact)
			},
		}

		m.Set(context.Background(), art)

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SetCallCount))
	})
}

func TestMockMetadataCache_SetMultiple(t *testing.T) {
	t.Parallel()

	t.Run("nil SetMultipleFunc is safe to call", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataCache{}

		m.SetMultiple(context.Background(), []*registry_dto.ArtefactMeta{{ID: "a"}})

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SetMultipleCallCount))
	})

	t.Run("delegates to SetMultipleFunc", func(t *testing.T) {
		t.Parallel()
		arts := []*registry_dto.ArtefactMeta{{ID: "a"}, {ID: "b"}}
		m := &MockMetadataCache{
			SetMultipleFunc: func(_ context.Context, artefacts []*registry_dto.ArtefactMeta) {
				assert.Equal(t, arts, artefacts)
			},
		}

		m.SetMultiple(context.Background(), arts)

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SetMultipleCallCount))
	})
}

func TestMockMetadataCache_Delete(t *testing.T) {
	t.Parallel()

	t.Run("nil DeleteFunc is safe to call", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataCache{}

		m.Delete(context.Background(), "art-1")

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DeleteCallCount))
	})

	t.Run("delegates to DeleteFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataCache{
			DeleteFunc: func(_ context.Context, artefactID string) {
				assert.Equal(t, "art-1", artefactID)
			},
		}

		m.Delete(context.Background(), "art-1")

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DeleteCallCount))
	})
}

func TestMockMetadataCache_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil CloseFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataCache{}

		err := m.Close(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})

	t.Run("delegates to CloseFunc", func(t *testing.T) {
		t.Parallel()
		called := false
		m := &MockMetadataCache{
			CloseFunc: func(_ context.Context) error {
				called = true
				return nil
			},
		}

		err := m.Close(context.Background())

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})

	t.Run("propagates error from CloseFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("close failed")
		m := &MockMetadataCache{
			CloseFunc: func(_ context.Context) error {
				return expectedErr
			},
		}

		err := m.Close(context.Background())

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataCache_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockMetadataCache
	ctx := context.Background()

	got1, err := m.Get(ctx, "")
	assert.Nil(t, got1)
	assert.NoError(t, err)

	got2, got2b := m.GetMultiple(ctx, nil)
	assert.Nil(t, got2)
	assert.Nil(t, got2b)

	m.Set(ctx, nil)
	m.SetMultiple(ctx, nil)
	m.Delete(ctx, "")

	assert.NoError(t, m.Close(context.Background()))
}

func TestMockMetadataCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockMetadataCache{
		GetFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		GetMultipleFunc: func(context.Context, []string) ([]*registry_dto.ArtefactMeta, []string) {
			return nil, nil
		},
		SetFunc:         func(context.Context, *registry_dto.ArtefactMeta) {},
		SetMultipleFunc: func(context.Context, []*registry_dto.ArtefactMeta) {},
		DeleteFunc:      func(context.Context, string) {},
		CloseFunc:       func(context.Context) error { return nil },
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			_, _ = m.Get(ctx, "")
			_, _ = m.GetMultiple(ctx, nil)
			m.Set(ctx, nil)
			m.SetMultiple(ctx, nil)
			m.Delete(ctx, "")
			_ = m.Close(context.Background())
		})
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetMultipleCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SetCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SetMultipleCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DeleteCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CloseCallCount))
}
