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
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
)

func TestMockRegistryService_UpsertArtefact(t *testing.T) {
	t.Parallel()

	t.Run("nil UpsertArtefactFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.UpsertArtefact(context.Background(), "art-1", "/src", strings.NewReader("data"), "backend-1", nil)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpsertArtefactCallCount))
	})

	t.Run("delegates to UpsertArtefactFunc", func(t *testing.T) {
		t.Parallel()
		want := &registry_dto.ArtefactMeta{ID: "art-1"}
		m := &MockRegistryService{
			UpsertArtefactFunc: func(_ context.Context, artefactID, sourcePath string, sourceData io.Reader, storageBackendID string, desiredProfiles []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, "art-1", artefactID)
				assert.Equal(t, "/src", sourcePath)
				assert.Equal(t, "backend-1", storageBackendID)
				return want, nil
			},
		}

		got, err := m.UpsertArtefact(context.Background(), "art-1", "/src", strings.NewReader("data"), "backend-1", nil)

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpsertArtefactCallCount))
	})

	t.Run("propagates error from UpsertArtefactFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("upsert failed")
		m := &MockRegistryService{
			UpsertArtefactFunc: func(context.Context, string, string, io.Reader, string, []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.UpsertArtefact(context.Background(), "art-1", "/src", strings.NewReader("data"), "backend-1", nil)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_AddVariant(t *testing.T) {
	t.Parallel()

	t.Run("nil AddVariantFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.AddVariant(context.Background(), "art-1", &registry_dto.Variant{})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.AddVariantCallCount))
	})

	t.Run("delegates to AddVariantFunc", func(t *testing.T) {
		t.Parallel()
		want := &registry_dto.ArtefactMeta{ID: "art-1"}
		variant := &registry_dto.Variant{}
		m := &MockRegistryService{
			AddVariantFunc: func(_ context.Context, artefactID string, newVariant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, "art-1", artefactID)
				assert.Same(t, variant, newVariant)
				return want, nil
			},
		}

		got, err := m.AddVariant(context.Background(), "art-1", variant)

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.AddVariantCallCount))
	})

	t.Run("propagates error from AddVariantFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("add variant failed")
		m := &MockRegistryService{
			AddVariantFunc: func(context.Context, string, *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.AddVariant(context.Background(), "art-1", &registry_dto.Variant{})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_DeleteArtefact(t *testing.T) {
	t.Parallel()

	t.Run("nil DeleteArtefactFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		err := m.DeleteArtefact(context.Background(), "art-1")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DeleteArtefactCallCount))
	})

	t.Run("delegates to DeleteArtefactFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{
			DeleteArtefactFunc: func(_ context.Context, artefactID string) error {
				assert.Equal(t, "art-1", artefactID)
				return nil
			},
		}

		err := m.DeleteArtefact(context.Background(), "art-1")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DeleteArtefactCallCount))
	})

	t.Run("propagates error from DeleteArtefactFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("delete failed")
		m := &MockRegistryService{
			DeleteArtefactFunc: func(context.Context, string) error {
				return expectedErr
			},
		}

		err := m.DeleteArtefact(context.Background(), "art-1")

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_GetArtefact(t *testing.T) {
	t.Parallel()

	t.Run("nil GetArtefactFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.GetArtefact(context.Background(), "art-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetArtefactCallCount))
	})

	t.Run("delegates to GetArtefactFunc", func(t *testing.T) {
		t.Parallel()
		want := &registry_dto.ArtefactMeta{ID: "art-1"}
		m := &MockRegistryService{
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
		m := &MockRegistryService{
			GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.GetArtefact(context.Background(), "art-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_GetMultipleArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("nil GetMultipleArtefactsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.GetMultipleArtefacts(context.Background(), []string{"a", "b"})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMultipleArtefactsCallCount))
	})

	t.Run("delegates to GetMultipleArtefactsFunc", func(t *testing.T) {
		t.Parallel()
		want := []*registry_dto.ArtefactMeta{{ID: "a"}, {ID: "b"}}
		m := &MockRegistryService{
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
		m := &MockRegistryService{
			GetMultipleArtefactsFunc: func(context.Context, []string) ([]*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.GetMultipleArtefacts(context.Background(), []string{"a"})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_ListAllArtefactIDs(t *testing.T) {
	t.Parallel()

	t.Run("nil ListAllArtefactIDsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.ListAllArtefactIDs(context.Background())

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ListAllArtefactIDsCallCount))
	})

	t.Run("delegates to ListAllArtefactIDsFunc", func(t *testing.T) {
		t.Parallel()
		want := []string{"id-1", "id-2"}
		m := &MockRegistryService{
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
		m := &MockRegistryService{
			ListAllArtefactIDsFunc: func(context.Context) ([]string, error) {
				return nil, expectedErr
			},
		}

		got, err := m.ListAllArtefactIDs(context.Background())

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_SearchArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("nil SearchArtefactsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.SearchArtefacts(context.Background(), SearchQuery{})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SearchArtefactsCallCount))
	})

	t.Run("delegates to SearchArtefactsFunc", func(t *testing.T) {
		t.Parallel()
		want := []*registry_dto.ArtefactMeta{{ID: "found"}}
		query := SearchQuery{SimpleTagQuery: map[string]string{"env": "prod"}}
		m := &MockRegistryService{
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
		m := &MockRegistryService{
			SearchArtefactsFunc: func(context.Context, SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.SearchArtefacts(context.Background(), SearchQuery{})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_SearchArtefactsByTagValues(t *testing.T) {
	t.Parallel()

	t.Run("nil SearchArtefactsByTagValuesFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.SearchArtefactsByTagValues(context.Background(), "env", []string{"prod"})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SearchArtefactsByTagValuesCallCount))
	})

	t.Run("delegates to SearchArtefactsByTagValuesFunc", func(t *testing.T) {
		t.Parallel()
		want := []*registry_dto.ArtefactMeta{{ID: "tagged"}}
		m := &MockRegistryService{
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
		m := &MockRegistryService{
			SearchArtefactsByTagValuesFunc: func(context.Context, string, []string) ([]*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.SearchArtefactsByTagValues(context.Background(), "env", []string{"prod"})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_FindArtefactByVariantStorageKey(t *testing.T) {
	t.Parallel()

	t.Run("nil FindArtefactByVariantStorageKeyFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.FindArtefactByVariantStorageKey(context.Background(), "key-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FindArtefactByVariantStorageKeyCallCount))
	})

	t.Run("delegates to FindArtefactByVariantStorageKeyFunc", func(t *testing.T) {
		t.Parallel()
		want := &registry_dto.ArtefactMeta{ID: "found-by-key"}
		m := &MockRegistryService{
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
		m := &MockRegistryService{
			FindArtefactByVariantStorageKeyFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
				return nil, expectedErr
			},
		}

		got, err := m.FindArtefactByVariantStorageKey(context.Background(), "key-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_GetVariantData(t *testing.T) {
	t.Parallel()

	t.Run("nil GetVariantDataFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.GetVariantData(context.Background(), &registry_dto.Variant{})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetVariantDataCallCount))
	})

	t.Run("delegates to GetVariantDataFunc", func(t *testing.T) {
		t.Parallel()
		want := io.NopCloser(strings.NewReader("variant-data"))
		variant := &registry_dto.Variant{}
		m := &MockRegistryService{
			GetVariantDataFunc: func(_ context.Context, v *registry_dto.Variant) (io.ReadCloser, error) {
				assert.Same(t, variant, v)
				return want, nil
			},
		}

		got, err := m.GetVariantData(context.Background(), variant)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetVariantDataCallCount))
	})

	t.Run("propagates error from GetVariantDataFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("variant data failed")
		m := &MockRegistryService{
			GetVariantDataFunc: func(context.Context, *registry_dto.Variant) (io.ReadCloser, error) {
				return nil, expectedErr
			},
		}

		got, err := m.GetVariantData(context.Background(), &registry_dto.Variant{})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_GetVariantChunk(t *testing.T) {
	t.Parallel()

	t.Run("nil GetVariantChunkFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.GetVariantChunk(context.Background(), &registry_dto.Variant{}, "chunk-0")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetVariantChunkCallCount))
	})

	t.Run("delegates to GetVariantChunkFunc", func(t *testing.T) {
		t.Parallel()
		want := io.NopCloser(strings.NewReader("chunk-data"))
		variant := &registry_dto.Variant{}
		m := &MockRegistryService{
			GetVariantChunkFunc: func(_ context.Context, v *registry_dto.Variant, chunkID string) (io.ReadCloser, error) {
				assert.Same(t, variant, v)
				assert.Equal(t, "chunk-0", chunkID)
				return want, nil
			},
		}

		got, err := m.GetVariantChunk(context.Background(), variant, "chunk-0")

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetVariantChunkCallCount))
	})

	t.Run("propagates error from GetVariantChunkFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("chunk failed")
		m := &MockRegistryService{
			GetVariantChunkFunc: func(context.Context, *registry_dto.Variant, string) (io.ReadCloser, error) {
				return nil, expectedErr
			},
		}

		got, err := m.GetVariantChunk(context.Background(), &registry_dto.Variant{}, "chunk-0")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_GetVariantDataRange(t *testing.T) {
	t.Parallel()

	t.Run("nil GetVariantDataRangeFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.GetVariantDataRange(context.Background(), &registry_dto.Variant{}, 0, 100)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetVariantDataRangeCallCount))
	})

	t.Run("delegates to GetVariantDataRangeFunc", func(t *testing.T) {
		t.Parallel()
		want := io.NopCloser(strings.NewReader("range-data"))
		variant := &registry_dto.Variant{}
		m := &MockRegistryService{
			GetVariantDataRangeFunc: func(_ context.Context, v *registry_dto.Variant, offset, length int64) (io.ReadCloser, error) {
				assert.Same(t, variant, v)
				assert.Equal(t, int64(10), offset)
				assert.Equal(t, int64(50), length)
				return want, nil
			},
		}

		got, err := m.GetVariantDataRange(context.Background(), variant, 10, 50)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetVariantDataRangeCallCount))
	})

	t.Run("propagates error from GetVariantDataRangeFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("range failed")
		m := &MockRegistryService{
			GetVariantDataRangeFunc: func(context.Context, *registry_dto.Variant, int64, int64) (io.ReadCloser, error) {
				return nil, expectedErr
			},
		}

		got, err := m.GetVariantDataRange(context.Background(), &registry_dto.Variant{}, 0, 100)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_GetBlobStore(t *testing.T) {
	t.Parallel()

	t.Run("nil GetBlobStoreFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.GetBlobStore("backend-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetBlobStoreCallCount))
	})

	t.Run("delegates to GetBlobStoreFunc", func(t *testing.T) {
		t.Parallel()
		want := &MockBlobStore{}
		m := &MockRegistryService{
			GetBlobStoreFunc: func(backendID string) (BlobStore, error) {
				assert.Equal(t, "backend-1", backendID)
				return want, nil
			},
		}

		got, err := m.GetBlobStore("backend-1")

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetBlobStoreCallCount))
	})

	t.Run("propagates error from GetBlobStoreFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("blob store not found")
		m := &MockRegistryService{
			GetBlobStoreFunc: func(string) (BlobStore, error) {
				return nil, expectedErr
			},
		}

		got, err := m.GetBlobStore("backend-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_PopGCHints(t *testing.T) {
	t.Parallel()

	t.Run("nil PopGCHintsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got, err := m.PopGCHints(context.Background(), 10)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PopGCHintsCallCount))
	})

	t.Run("delegates to PopGCHintsFunc", func(t *testing.T) {
		t.Parallel()
		want := []registry_dto.GCHint{{BackendID: "b1", StorageKey: "k1"}}
		m := &MockRegistryService{
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
		m := &MockRegistryService{
			PopGCHintsFunc: func(context.Context, int) ([]registry_dto.GCHint, error) {
				return nil, expectedErr
			},
		}

		got, err := m.PopGCHints(context.Background(), 10)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockRegistryService_ArtefactEventsPublished(t *testing.T) {
	t.Parallel()

	t.Run("nil ArtefactEventsPublishedFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{}

		got := m.ArtefactEventsPublished()

		assert.Equal(t, int64(0), got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ArtefactEventsPublishedCallCount))
	})

	t.Run("delegates to ArtefactEventsPublishedFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryService{
			ArtefactEventsPublishedFunc: func() int64 {
				return 42
			},
		}

		got := m.ArtefactEventsPublished()

		assert.Equal(t, int64(42), got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ArtefactEventsPublishedCallCount))
	})
}

func TestMockRegistryService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockRegistryService
	ctx := context.Background()

	got1, err := m.UpsertArtefact(ctx, "", "", &bytes.Buffer{}, "", nil)
	assert.Nil(t, got1)
	assert.NoError(t, err)

	got2, err := m.AddVariant(ctx, "", nil)
	assert.Nil(t, got2)
	assert.NoError(t, err)

	assert.NoError(t, m.DeleteArtefact(ctx, ""))

	got3, err := m.GetArtefact(ctx, "")
	assert.Nil(t, got3)
	assert.NoError(t, err)

	got4, err := m.GetMultipleArtefacts(ctx, nil)
	assert.Nil(t, got4)
	assert.NoError(t, err)

	got5, err := m.ListAllArtefactIDs(ctx)
	assert.Nil(t, got5)
	assert.NoError(t, err)

	got6, err := m.SearchArtefacts(ctx, SearchQuery{})
	assert.Nil(t, got6)
	assert.NoError(t, err)

	got7, err := m.SearchArtefactsByTagValues(ctx, "", nil)
	assert.Nil(t, got7)
	assert.NoError(t, err)

	got8, err := m.FindArtefactByVariantStorageKey(ctx, "")
	assert.Nil(t, got8)
	assert.NoError(t, err)

	got9, err := m.GetVariantData(ctx, nil)
	assert.Nil(t, got9)
	assert.NoError(t, err)

	got10, err := m.GetVariantChunk(ctx, nil, "")
	assert.Nil(t, got10)
	assert.NoError(t, err)

	got11, err := m.GetVariantDataRange(ctx, nil, 0, 0)
	assert.Nil(t, got11)
	assert.NoError(t, err)

	got12, err := m.GetBlobStore("")
	assert.Nil(t, got12)
	assert.NoError(t, err)

	got13, err := m.PopGCHints(ctx, 0)
	assert.Nil(t, got13)
	assert.NoError(t, err)

	got14 := m.ArtefactEventsPublished()
	assert.Equal(t, int64(0), got14)
}

func TestMockRegistryService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockRegistryService{
		UpsertArtefactFunc: func(context.Context, string, string, io.Reader, string, []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		AddVariantFunc: func(context.Context, string, *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		DeleteArtefactFunc: func(context.Context, string) error { return nil },
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
		GetVariantDataFunc: func(context.Context, *registry_dto.Variant) (io.ReadCloser, error) {
			return nil, nil
		},
		GetVariantChunkFunc: func(context.Context, *registry_dto.Variant, string) (io.ReadCloser, error) {
			return nil, nil
		},
		GetVariantDataRangeFunc: func(context.Context, *registry_dto.Variant, int64, int64) (io.ReadCloser, error) {
			return nil, nil
		},
		GetBlobStoreFunc:            func(string) (BlobStore, error) { return nil, nil },
		PopGCHintsFunc:              func(context.Context, int) ([]registry_dto.GCHint, error) { return nil, nil },
		ArtefactEventsPublishedFunc: func() int64 { return 1 },
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			_, _ = m.UpsertArtefact(ctx, "", "", &bytes.Buffer{}, "", nil)
			_, _ = m.AddVariant(ctx, "", nil)
			_ = m.DeleteArtefact(ctx, "")
			_, _ = m.GetArtefact(ctx, "")
			_, _ = m.GetMultipleArtefacts(ctx, nil)
			_, _ = m.ListAllArtefactIDs(ctx)
			_, _ = m.SearchArtefacts(ctx, SearchQuery{})
			_, _ = m.SearchArtefactsByTagValues(ctx, "", nil)
			_, _ = m.FindArtefactByVariantStorageKey(ctx, "")
			_, _ = m.GetVariantData(ctx, nil)
			_, _ = m.GetVariantChunk(ctx, nil, "")
			_, _ = m.GetVariantDataRange(ctx, nil, 0, 0)
			_, _ = m.GetBlobStore("")
			_, _ = m.PopGCHints(ctx, 0)
			_ = m.ArtefactEventsPublished()
		})
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.UpsertArtefactCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.AddVariantCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DeleteArtefactCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetArtefactCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetMultipleArtefactsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ListAllArtefactIDsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SearchArtefactsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SearchArtefactsByTagValuesCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.FindArtefactByVariantStorageKeyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetVariantDataCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetVariantChunkCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetVariantDataRangeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetBlobStoreCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PopGCHintsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ArtefactEventsPublishedCallCount))
}
