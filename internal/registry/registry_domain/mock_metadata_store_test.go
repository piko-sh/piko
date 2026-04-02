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

func TestMockMetadataStore_GetArtefact(t *testing.T) {
	t.Parallel()

	t.Run("nil GetArtefactFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.GetArtefact(context.Background(), "art-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetArtefactCallCount))
	})

	t.Run("delegates to GetArtefactFunc", func(t *testing.T) {
		t.Parallel()
		want := &registry_dto.ArtefactMeta{ID: "art-1"}
		m := &MockMetadataStore{
			GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, "art-1", artefactID)
				return want, nil
			},
		}

		got, err := m.GetArtefact(context.Background(), "art-1")

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetArtefactCallCount))
	})

	t.Run("propagates error from GetArtefactFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("get failed")
		m := &MockMetadataStore{
			GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.GetArtefact(context.Background(), "art-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_GetMultipleArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("nil GetMultipleArtefactsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.GetMultipleArtefacts(context.Background(), []string{"a", "b"})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMultipleArtefactsCallCount))
	})

	t.Run("delegates to GetMultipleArtefactsFunc", func(t *testing.T) {
		t.Parallel()
		want := []*registry_dto.ArtefactMeta{{ID: "a"}, {ID: "b"}}
		m := &MockMetadataStore{
			GetMultipleArtefactsFunc: func(_ context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, []string{"a", "b"}, artefactIDs)
				return want, nil
			},
		}

		got, err := m.GetMultipleArtefacts(context.Background(), []string{"a", "b"})

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMultipleArtefactsCallCount))
	})

	t.Run("propagates error from GetMultipleArtefactsFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("multi-get failed")
		m := &MockMetadataStore{
			GetMultipleArtefactsFunc: func(context.Context, []string) ([]*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.GetMultipleArtefacts(context.Background(), []string{"a"})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_ListAllArtefactIDs(t *testing.T) {
	t.Parallel()

	t.Run("nil ListAllArtefactIDsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.ListAllArtefactIDs(context.Background())

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ListAllArtefactIDsCallCount))
	})

	t.Run("delegates to ListAllArtefactIDsFunc", func(t *testing.T) {
		t.Parallel()
		want := []string{"id-1", "id-2"}
		m := &MockMetadataStore{
			ListAllArtefactIDsFunc: func(context.Context) ([]string, error) {
				return want, nil
			},
		}

		got, err := m.ListAllArtefactIDs(context.Background())

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ListAllArtefactIDsCallCount))
	})

	t.Run("propagates error from ListAllArtefactIDsFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("list failed")
		m := &MockMetadataStore{
			ListAllArtefactIDsFunc: func(context.Context) ([]string, error) {
				return nil, expectedErr
			},
		}

		got, err := m.ListAllArtefactIDs(context.Background())

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_SearchArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("nil SearchArtefactsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.SearchArtefacts(context.Background(), SearchQuery{})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SearchArtefactsCallCount))
	})

	t.Run("delegates to SearchArtefactsFunc", func(t *testing.T) {
		t.Parallel()
		want := []*registry_dto.ArtefactMeta{{ID: "found"}}
		query := SearchQuery{SimpleTagQuery: map[string]string{"env": "prod"}}
		m := &MockMetadataStore{
			SearchArtefactsFunc: func(_ context.Context, q SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, query, q)
				return want, nil
			},
		}

		got, err := m.SearchArtefacts(context.Background(), query)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SearchArtefactsCallCount))
	})

	t.Run("propagates error from SearchArtefactsFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("search failed")
		m := &MockMetadataStore{
			SearchArtefactsFunc: func(context.Context, SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.SearchArtefacts(context.Background(), SearchQuery{})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_SearchArtefactsByTagValues(t *testing.T) {
	t.Parallel()

	t.Run("nil SearchArtefactsByTagValuesFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.SearchArtefactsByTagValues(context.Background(), "env", []string{"prod"})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SearchArtefactsByTagValuesCallCount))
	})

	t.Run("delegates to SearchArtefactsByTagValuesFunc", func(t *testing.T) {
		t.Parallel()
		want := []*registry_dto.ArtefactMeta{{ID: "tagged"}}
		m := &MockMetadataStore{
			SearchArtefactsByTagValuesFunc: func(_ context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, "env", tagKey)
				assert.Equal(t, []string{"prod", "staging"}, tagValues)
				return want, nil
			},
		}

		got, err := m.SearchArtefactsByTagValues(context.Background(), "env", []string{"prod", "staging"})

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SearchArtefactsByTagValuesCallCount))
	})

	t.Run("propagates error from SearchArtefactsByTagValuesFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("tag search failed")
		m := &MockMetadataStore{
			SearchArtefactsByTagValuesFunc: func(context.Context, string, []string) ([]*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.SearchArtefactsByTagValues(context.Background(), "env", []string{"prod"})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_FindArtefactByVariantStorageKey(t *testing.T) {
	t.Parallel()

	t.Run("nil FindArtefactByVariantStorageKeyFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.FindArtefactByVariantStorageKey(context.Background(), "key-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FindArtefactByVariantStorageKeyCallCount))
	})

	t.Run("delegates to FindArtefactByVariantStorageKeyFunc", func(t *testing.T) {
		t.Parallel()
		want := &registry_dto.ArtefactMeta{ID: "found-by-key"}
		m := &MockMetadataStore{
			FindArtefactByVariantStorageKeyFunc: func(_ context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, "key-1", storageKey)
				return want, nil
			},
		}

		got, err := m.FindArtefactByVariantStorageKey(context.Background(), "key-1")

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FindArtefactByVariantStorageKeyCallCount))
	})

	t.Run("propagates error from FindArtefactByVariantStorageKeyFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("lookup failed")
		m := &MockMetadataStore{
			FindArtefactByVariantStorageKeyFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.FindArtefactByVariantStorageKey(context.Background(), "key-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_PopGCHints(t *testing.T) {
	t.Parallel()

	t.Run("nil PopGCHintsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.PopGCHints(context.Background(), 10)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PopGCHintsCallCount))
	})

	t.Run("delegates to PopGCHintsFunc", func(t *testing.T) {
		t.Parallel()
		want := []registry_dto.GCHint{{BackendID: "b1", StorageKey: "k1"}}
		m := &MockMetadataStore{
			PopGCHintsFunc: func(_ context.Context, limit int) ([]registry_dto.GCHint, error) {
				assert.Equal(t, 5, limit)
				return want, nil
			},
		}

		got, err := m.PopGCHints(context.Background(), 5)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PopGCHintsCallCount))
	})

	t.Run("propagates error from PopGCHintsFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("gc hints failed")
		m := &MockMetadataStore{
			PopGCHintsFunc: func(context.Context, int) ([]registry_dto.GCHint, error) {
				return nil, expectedErr
			},
		}

		got, err := m.PopGCHints(context.Background(), 10)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_AtomicUpdate(t *testing.T) {
	t.Parallel()

	t.Run("nil AtomicUpdateFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		err := m.AtomicUpdate(context.Background(), nil)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.AtomicUpdateCallCount))
	})

	t.Run("delegates to AtomicUpdateFunc", func(t *testing.T) {
		t.Parallel()
		actions := []registry_dto.AtomicAction{{ArtefactID: "art-1"}}
		m := &MockMetadataStore{
			AtomicUpdateFunc: func(_ context.Context, a []registry_dto.AtomicAction) error {
				assert.Equal(t, actions, a)
				return nil
			},
		}

		err := m.AtomicUpdate(context.Background(), actions)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.AtomicUpdateCallCount))
	})

	t.Run("propagates error from AtomicUpdateFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("atomic update failed")
		m := &MockMetadataStore{
			AtomicUpdateFunc: func(context.Context, []registry_dto.AtomicAction) error {
				return expectedErr
			},
		}

		err := m.AtomicUpdate(context.Background(), nil)

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_IncrementBlobRefCount(t *testing.T) {
	t.Parallel()

	t.Run("nil IncrementBlobRefCountFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.IncrementBlobRefCount(context.Background(), BlobReference{})

		assert.Equal(t, 0, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.IncrementBlobRefCountCallCount))
	})

	t.Run("delegates to IncrementBlobRefCountFunc", func(t *testing.T) {
		t.Parallel()
		blob := BlobReference{StorageKey: "key-1"}
		m := &MockMetadataStore{
			IncrementBlobRefCountFunc: func(_ context.Context, b BlobReference) (int, error) {
				assert.Equal(t, blob, b)
				return 3, nil
			},
		}

		got, err := m.IncrementBlobRefCount(context.Background(), blob)

		require.NoError(t, err)
		assert.Equal(t, 3, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.IncrementBlobRefCountCallCount))
	})

	t.Run("propagates error from IncrementBlobRefCountFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("increment failed")
		m := &MockMetadataStore{
			IncrementBlobRefCountFunc: func(context.Context, BlobReference) (int, error) {
				return 0, expectedErr
			},
		}

		got, err := m.IncrementBlobRefCount(context.Background(), BlobReference{})

		assert.Equal(t, 0, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_DecrementBlobRefCount(t *testing.T) {
	t.Parallel()

	t.Run("nil DecrementBlobRefCountFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		count, deleted, err := m.DecrementBlobRefCount(context.Background(), "key-1")

		assert.Equal(t, 0, count)
		assert.False(t, deleted)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DecrementBlobRefCountCallCount))
	})

	t.Run("delegates to DecrementBlobRefCountFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{
			DecrementBlobRefCountFunc: func(_ context.Context, storageKey string) (int, bool, error) {
				assert.Equal(t, "key-1", storageKey)
				return 0, true, nil
			},
		}

		count, deleted, err := m.DecrementBlobRefCount(context.Background(), "key-1")

		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.True(t, deleted)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DecrementBlobRefCountCallCount))
	})

	t.Run("propagates error from DecrementBlobRefCountFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("decrement failed")
		m := &MockMetadataStore{
			DecrementBlobRefCountFunc: func(context.Context, string) (int, bool, error) {
				return 0, false, expectedErr
			},
		}

		count, deleted, err := m.DecrementBlobRefCount(context.Background(), "key-1")

		assert.Equal(t, 0, count)
		assert.False(t, deleted)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_GetBlobRefCount(t *testing.T) {
	t.Parallel()

	t.Run("nil GetBlobRefCountFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		got, err := m.GetBlobRefCount(context.Background(), "key-1")

		assert.Equal(t, 0, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetBlobRefCountCallCount))
	})

	t.Run("delegates to GetBlobRefCountFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{
			GetBlobRefCountFunc: func(_ context.Context, storageKey string) (int, error) {
				assert.Equal(t, "key-1", storageKey)
				return 7, nil
			},
		}

		got, err := m.GetBlobRefCount(context.Background(), "key-1")

		require.NoError(t, err)
		assert.Equal(t, 7, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetBlobRefCountCallCount))
	})

	t.Run("propagates error from GetBlobRefCountFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("ref count failed")
		m := &MockMetadataStore{
			GetBlobRefCountFunc: func(context.Context, string) (int, error) {
				return 0, expectedErr
			},
		}

		got, err := m.GetBlobRefCount(context.Background(), "key-1")

		assert.Equal(t, 0, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil CloseFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockMetadataStore{}

		err := m.Close()

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})

	t.Run("delegates to CloseFunc", func(t *testing.T) {
		t.Parallel()
		called := false
		m := &MockMetadataStore{
			CloseFunc: func() error {
				called = true
				return nil
			},
		}

		err := m.Close()

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})

	t.Run("propagates error from CloseFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("close failed")
		m := &MockMetadataStore{
			CloseFunc: func() error {
				return expectedErr
			},
		}

		err := m.Close()

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockMetadataStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockMetadataStore
	ctx := context.Background()

	got1, err := m.GetArtefact(ctx, "")
	assert.Nil(t, got1)
	assert.NoError(t, err)

	got2, err := m.GetMultipleArtefacts(ctx, nil)
	assert.Nil(t, got2)
	assert.NoError(t, err)

	got3, err := m.ListAllArtefactIDs(ctx)
	assert.Nil(t, got3)
	assert.NoError(t, err)

	got4, err := m.SearchArtefacts(ctx, SearchQuery{})
	assert.Nil(t, got4)
	assert.NoError(t, err)

	got5, err := m.SearchArtefactsByTagValues(ctx, "", nil)
	assert.Nil(t, got5)
	assert.NoError(t, err)

	got6, err := m.FindArtefactByVariantStorageKey(ctx, "")
	assert.Nil(t, got6)
	assert.NoError(t, err)

	got7, err := m.PopGCHints(ctx, 0)
	assert.Nil(t, got7)
	assert.NoError(t, err)

	assert.NoError(t, m.AtomicUpdate(ctx, nil))

	got8, err := m.IncrementBlobRefCount(ctx, BlobReference{})
	assert.Equal(t, 0, got8)
	assert.NoError(t, err)

	got9, got9b, err := m.DecrementBlobRefCount(ctx, "")
	assert.Equal(t, 0, got9)
	assert.False(t, got9b)
	assert.NoError(t, err)

	got10, err := m.GetBlobRefCount(ctx, "")
	assert.Equal(t, 0, got10)
	assert.NoError(t, err)

	assert.NoError(t, m.Close())
}

func TestMockMetadataStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockMetadataStore{
		GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		GetMultipleArtefactsFunc: func(context.Context, []string) ([]*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		ListAllArtefactIDsFunc: func(context.Context) ([]string, error) { return nil, nil },
		SearchArtefactsFunc: func(context.Context, SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		SearchArtefactsByTagValuesFunc: func(context.Context, string, []string) ([]*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		FindArtefactByVariantStorageKeyFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		PopGCHintsFunc:            func(context.Context, int) ([]registry_dto.GCHint, error) { return nil, nil },
		AtomicUpdateFunc:          func(context.Context, []registry_dto.AtomicAction) error { return nil },
		IncrementBlobRefCountFunc: func(context.Context, BlobReference) (int, error) { return 0, nil },
		DecrementBlobRefCountFunc: func(context.Context, string) (int, bool, error) { return 0, false, nil },
		GetBlobRefCountFunc:       func(context.Context, string) (int, error) { return 0, nil },
		CloseFunc:                 func() error { return nil },
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			_, _ = m.GetArtefact(ctx, "")
			_, _ = m.GetMultipleArtefacts(ctx, nil)
			_, _ = m.ListAllArtefactIDs(ctx)
			_, _ = m.SearchArtefacts(ctx, SearchQuery{})
			_, _ = m.SearchArtefactsByTagValues(ctx, "", nil)
			_, _ = m.FindArtefactByVariantStorageKey(ctx, "")
			_, _ = m.PopGCHints(ctx, 0)
			_ = m.AtomicUpdate(ctx, nil)
			_, _ = m.IncrementBlobRefCount(ctx, BlobReference{})
			_, _, _ = m.DecrementBlobRefCount(ctx, "")
			_, _ = m.GetBlobRefCount(ctx, "")
			_ = m.Close()
		})
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetArtefactCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetMultipleArtefactsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ListAllArtefactIDsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SearchArtefactsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SearchArtefactsByTagValuesCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.FindArtefactByVariantStorageKeyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PopGCHintsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.AtomicUpdateCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.IncrementBlobRefCountCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DecrementBlobRefCountCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetBlobRefCountCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CloseCallCount))
}
