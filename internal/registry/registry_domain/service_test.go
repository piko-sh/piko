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

package registry_domain_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

type testFixture struct {
	service     registry_domain.RegistryService
	metaStore   *registry_domain.MockMetadataStore
	blobStore   *registry_domain.MockBlobStore
	eventBus    *registry_domain.MockEventBus
	cache       *registry_domain.MockMetadataCache
	blobStores  map[string]registry_domain.BlobStore
	testContext context.Context
}

func setupTest() testFixture {
	metaStore := &registry_domain.MockMetadataStore{}
	blobStore := &registry_domain.MockBlobStore{}
	eventBus := &registry_domain.MockEventBus{}
	blobStores := map[string]registry_domain.BlobStore{
		"local_disk_cache": blobStore,
	}
	service := registry_domain.NewRegistryService(metaStore, blobStores, eventBus, nil)
	return testFixture{
		service:     service,
		metaStore:   metaStore,
		blobStore:   blobStore,
		eventBus:    eventBus,
		cache:       nil,
		blobStores:  blobStores,
		testContext: context.Background(),
	}
}

func setupTestWithCache() testFixture {
	metaStore := &registry_domain.MockMetadataStore{}
	blobStore := &registry_domain.MockBlobStore{}
	eventBus := &registry_domain.MockEventBus{}
	cache := &registry_domain.MockMetadataCache{}
	blobStores := map[string]registry_domain.BlobStore{
		"local_disk_cache": blobStore,
	}
	service := registry_domain.NewRegistryService(metaStore, blobStores, eventBus, cache)
	return testFixture{
		service:     service,
		metaStore:   metaStore,
		blobStore:   blobStore,
		eventBus:    eventBus,
		cache:       cache,
		blobStores:  blobStores,
		testContext: context.Background(),
	}
}

func setupTestWithHealthyBlobStore() testFixture {
	metaStore := &registry_domain.MockMetadataStore{}
	blobStore := &registry_domain.MockHealthyBlobStore{}
	eventBus := &registry_domain.MockEventBus{}
	blobStores := map[string]registry_domain.BlobStore{
		"local_disk_cache": blobStore,
	}
	service := registry_domain.NewRegistryService(metaStore, blobStores, eventBus, nil)
	return testFixture{
		service:     service,
		metaStore:   metaStore,
		blobStore:   &blobStore.MockBlobStore,
		eventBus:    eventBus,
		cache:       nil,
		blobStores:  blobStores,
		testContext: context.Background(),
	}
}

type ArtefactBuilder struct {
	artefact *registry_dto.ArtefactMeta
}

func NewArtefactBuilder(id string) *ArtefactBuilder {
	return &ArtefactBuilder{
		artefact: &registry_dto.ArtefactMeta{
			ID:              id,
			CreatedAt:       time.Now().Add(-1 * time.Hour),
			UpdatedAt:       time.Now(),
			SourcePath:      "path/to/" + id + ".js",
			ActualVariants:  []registry_dto.Variant{},
			DesiredProfiles: []registry_dto.NamedProfile{},
		},
	}
}

func (b *ArtefactBuilder) WithSourcePath(path string) *ArtefactBuilder {
	b.artefact.SourcePath = path
	return b
}

func (b *ArtefactBuilder) WithCreatedAt(t time.Time) *ArtefactBuilder {
	b.artefact.CreatedAt = t
	return b
}

func (b *ArtefactBuilder) WithUpdatedAt(t time.Time) *ArtefactBuilder {
	b.artefact.UpdatedAt = t
	return b
}

func (b *ArtefactBuilder) WithSourceVariant(hash string) *ArtefactBuilder {
	b.artefact.ActualVariants = append(b.artefact.ActualVariants, registry_dto.Variant{
		CreatedAt:        time.Now().Add(-30 * time.Minute),
		MetadataTags:     registry_dto.TagsFromMap(map[string]string{"type": "source", "hash": hash}),
		VariantID:        "source",
		StorageBackendID: "local_disk_cache",
		StorageKey:       "source/" + hash + ".js",
		MimeType:         "text/javascript",
		Status:           registry_dto.VariantStatusReady,
		ContentHash:      hash,
		SizeBytes:        1024,
		Chunks:           nil,
	})
	return b
}

func (b *ArtefactBuilder) WithVariant(id, storageKey string) *ArtefactBuilder {
	b.artefact.ActualVariants = append(b.artefact.ActualVariants, registry_dto.Variant{
		CreatedAt:        time.Now().Add(-15 * time.Minute),
		MetadataTags:     registry_dto.TagsFromMap(map[string]string{"type": id}),
		VariantID:        id,
		StorageBackendID: "local_disk_cache",
		StorageKey:       storageKey,
		MimeType:         "text/javascript",
		Status:           registry_dto.VariantStatusReady,
		SizeBytes:        512,
		Chunks:           nil,
	})
	return b
}

func (b *ArtefactBuilder) WithVariantFull(variant registry_dto.Variant) *ArtefactBuilder {
	b.artefact.ActualVariants = append(b.artefact.ActualVariants, variant)
	return b
}

func (b *ArtefactBuilder) WithProfile(name string, dependsOn ...string) *ArtefactBuilder {
	var deps registry_dto.Dependencies
	for _, dependency := range dependsOn {
		deps.Add(dependency)
	}
	b.artefact.DesiredProfiles = append(b.artefact.DesiredProfiles, registry_dto.NamedProfile{
		Name: name,
		Profile: registry_dto.DesiredProfile{
			CapabilityName: name,
			DependsOn:      deps,
		},
	})
	return b
}

func (b *ArtefactBuilder) WithProfileFull(name string, profile registry_dto.DesiredProfile) *ArtefactBuilder {
	b.artefact.DesiredProfiles = append(b.artefact.DesiredProfiles, registry_dto.NamedProfile{
		Name:    name,
		Profile: profile,
	})
	return b
}

func (b *ArtefactBuilder) Build() *registry_dto.ArtefactMeta {
	return b.artefact
}

type VariantBuilder struct {
	variant registry_dto.Variant
}

func NewVariantBuilder(id string) *VariantBuilder {
	return &VariantBuilder{
		variant: registry_dto.Variant{
			CreatedAt:        time.Now().Add(-15 * time.Minute),
			VariantID:        id,
			StorageBackendID: "local_disk_cache",
			StorageKey:       "generated/" + id + ".js",
			MimeType:         "text/javascript",
			Status:           registry_dto.VariantStatusReady,
			SizeBytes:        512,
			Chunks:           nil,
		},
	}
}

func (b *VariantBuilder) WithStorageKey(key string) *VariantBuilder {
	b.variant.StorageKey = key
	return b
}

func (b *VariantBuilder) WithStorageBackend(backendID string) *VariantBuilder {
	b.variant.StorageBackendID = backendID
	return b
}

func (b *VariantBuilder) WithMimeType(mimeType string) *VariantBuilder {
	b.variant.MimeType = mimeType
	return b
}

func (b *VariantBuilder) WithContentHash(hash string) *VariantBuilder {
	b.variant.ContentHash = hash
	return b
}

func (b *VariantBuilder) WithSize(size int64) *VariantBuilder {
	b.variant.SizeBytes = size
	return b
}

func (b *VariantBuilder) WithStatus(status registry_dto.VariantStatus) *VariantBuilder {
	b.variant.Status = status
	return b
}

func (b *VariantBuilder) WithChunk(chunkID, storageKey string, sequenceNum int) *VariantBuilder {
	b.variant.Chunks = append(b.variant.Chunks, registry_dto.VariantChunk{
		ChunkID:          chunkID,
		StorageKey:       storageKey,
		StorageBackendID: b.variant.StorageBackendID,
		SequenceNumber:   sequenceNum,
		SizeBytes:        256,
		MimeType:         b.variant.MimeType,
	})
	return b
}

func (b *VariantBuilder) WithTag(key, value string) *VariantBuilder {
	b.variant.MetadataTags.SetByName(key, value)
	return b
}

func (b *VariantBuilder) Build() registry_dto.Variant {
	return b.variant
}

func wireNewArtefactUpsertMocks(f *testFixture) {
	f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrArtefactNotFound
	}
	f.blobStore.PutFunc = func(_ context.Context, _ string, r io.Reader) error {
		_, _ = io.Copy(io.Discard, r)
		return nil
	}
	f.blobStore.ExistsFunc = func(_ context.Context, _ string) (bool, error) { return false, nil }
	f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
		return 1, nil
	}
	f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }
}

func TestRegistryService_UpsertArtefact(t *testing.T) {
	artefactID := "test-artefact"
	sourcePath := "path/to/source.js"
	sourceData := "console.log('hello');"
	storageBackendID := "local_disk_cache"
	desiredProfiles := []registry_dto.NamedProfile{{Name: "minified", Profile: registry_dto.DesiredProfile{}}}

	expectedHash := "b98785ede1f35602a98818397e292fd8d4dcb66267c427d7d5486196b8b3bcd1"
	expectedFinalKey := fmt.Sprintf("source/%s.js", expectedHash)

	t.Run("New Artefact Success", func(t *testing.T) {
		f := setupTest()
		wireNewArtefactUpsertMocks(&f)
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, artefactID, artefact.ID)
		assert.WithinDuration(t, time.Now(), artefact.CreatedAt, time.Second)
		assert.WithinDuration(t, time.Now(), artefact.UpdatedAt, time.Second)
		require.Len(t, capturedActions, 1)
		action := capturedActions[0]
		art := action.Artefact
		assert.Equal(t, registry_dto.ActionTypeUpsertArtefact, action.Type)
		assert.Equal(t, artefactID, art.ID)
		assert.Equal(t, sourcePath, art.SourcePath)
		require.Len(t, art.ActualVariants, 1)
		assert.Equal(t, "source", art.ActualVariants[0].VariantID)
		assert.Equal(t, expectedFinalKey, art.ActualVariants[0].StorageKey)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})

	t.Run("Update Artefact with Changed Content", func(t *testing.T) {
		f := setupTest()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID:        artefactID,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			ActualVariants: []registry_dto.Variant{
				{VariantID: "source", StorageKey: "source/oldhash.js"},
			},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.blobStore.PutFunc = func(_ context.Context, _ string, r io.Reader) error {
			_, _ = io.Copy(io.Discard, r)
			return nil
		}
		f.blobStore.ExistsFunc = func(_ context.Context, _ string) (bool, error) { return false, nil }
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 1, nil
		}
		f.metaStore.DecrementBlobRefCountFunc = func(_ context.Context, key string) (int, bool, error) {
			assert.Equal(t, "source/oldhash.js", key)
			return 0, true, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.NoError(t, err)
		assert.Equal(t, existingArtefact.CreatedAt, artefact.CreatedAt, "CreatedAt should not change on update")
		assert.True(t, artefact.UpdatedAt.After(existingArtefact.CreatedAt), "UpdatedAt should be newer")
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})

	t.Run("Update Artefact with Unchanged Content", func(t *testing.T) {
		f := setupTest()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID:         artefactID,
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			SourcePath: sourcePath,
			ActualVariants: []registry_dto.Variant{
				{VariantID: "source", StorageKey: expectedFinalKey},
			},
			DesiredProfiles: []registry_dto.NamedProfile{{Name: "old", Profile: registry_dto.DesiredProfile{}}},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.blobStore.PutFunc = func(_ context.Context, _ string, r io.Reader) error {
			_, _ = io.Copy(io.Discard, r)
			return nil
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, artefactID, artefact.ID)
		assert.Equal(t, existingArtefact.CreatedAt, artefact.CreatedAt, "CreatedAt should be preserved")
		assert.Equal(t, desiredProfiles, artefact.DesiredProfiles, "DesiredProfiles should be updated")
		require.Len(t, capturedActions, 1)
		art := capturedActions[0].Artefact
		assert.Equal(t, artefactID, art.ID)
		require.Len(t, art.DesiredProfiles, 1)
		assert.Equal(t, "minified", art.DesiredProfiles[0].Name)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.RenameCallCount))
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})

	t.Run("Error during Blob Put", func(t *testing.T) {
		f := setupTest()
		putErr := errors.New("disk full")
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		f.blobStore.PutFunc = func(_ context.Context, _ string, _ io.Reader) error { return putErr }

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.Error(t, err)
		assert.ErrorIs(t, err, putErr)
		assert.Nil(t, artefact)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount))
	})
}

func TestRegistryService_AddVariant(t *testing.T) {
	artefactID := "test-artefact"
	existingArtefact := &registry_dto.ArtefactMeta{
		ID: artefactID,
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source/key1"},
		},
	}
	newVariant := registry_dto.Variant{
		VariantID:        "minified",
		StorageKey:       "generated/key2",
		StorageBackendID: "local",
		SizeBytes:        512,
		MimeType:         "application/javascript",
		ContentHash:      "minified_hash",
	}

	t.Run("Add New Variant Success", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 1, nil
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &newVariant)

		require.NoError(t, err)
		require.NotEmpty(t, capturedActions)
		art := capturedActions[0].Artefact
		assert.Len(t, art.ActualVariants, 2)
		assert.Equal(t, "minified", art.ActualVariants[1].VariantID)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})

	t.Run("Replace Existing Variant", func(t *testing.T) {
		f := setupTest()
		variantToReplace := registry_dto.Variant{VariantID: "minified", StorageKey: "generated/old_key"}
		artefactWithVariant := &registry_dto.ArtefactMeta{
			ID: artefactID,
			ActualVariants: []registry_dto.Variant{
				{VariantID: "source", StorageKey: "source/key1"},
				variantToReplace,
			},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return artefactWithVariant, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 1, nil
		}
		f.metaStore.DecrementBlobRefCountFunc = func(_ context.Context, key string) (int, bool, error) {
			assert.Equal(t, "generated/old_key", key)
			return 0, true, nil
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &newVariant)

		require.NoError(t, err)
		require.Len(t, capturedActions, 2)
		assert.Equal(t, registry_dto.ActionTypeUpsertArtefact, capturedActions[0].Type)
		assert.Len(t, capturedActions[0].Artefact.ActualVariants, 2)
		assert.Equal(t, registry_dto.ActionTypeAddGCHints, capturedActions[1].Type)
		require.Len(t, capturedActions[1].GCHints, 1)
		assert.Equal(t, "generated/old_key", capturedActions[1].GCHints[0].StorageKey)
	})
}

func TestRegistryService_DeleteArtefact(t *testing.T) {
	artefactID := "test-artefact"
	existingArtefact := &registry_dto.ArtefactMeta{
		ID: artefactID,
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageBackendID: "local", StorageKey: "key1"},
			{VariantID: "minified", StorageBackendID: "local", StorageKey: "key2"},
		},
	}

	t.Run("Success", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.DecrementBlobRefCountFunc = func(_ context.Context, _ string) (int, bool, error) {
			return 0, true, nil
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		err := f.service.DeleteArtefact(f.testContext, artefactID)

		require.NoError(t, err)
		require.Len(t, capturedActions, 2)
		assert.Equal(t, registry_dto.ActionTypeDeleteArtefact, capturedActions[0].Type)
		assert.Equal(t, artefactID, capturedActions[0].ArtefactID)
		assert.Equal(t, registry_dto.ActionTypeAddGCHints, capturedActions[1].Type)
		require.Len(t, capturedActions[1].GCHints, 2)
		keys := map[string]bool{capturedActions[1].GCHints[0].StorageKey: true, capturedActions[1].GCHints[1].StorageKey: true}
		assert.True(t, keys["key1"] && keys["key2"])
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})

	t.Run("Artefact Not Found", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}

		err := f.service.DeleteArtefact(f.testContext, artefactID)

		require.NoError(t, err, "Deleting a non-existent artefact should not be an error")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount))
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})
}
func TestRegistryService_UpsertArtefact_MetadataOnly(t *testing.T) {
	artefactID := "test-artefact"
	sourcePath := "path/to/source.js"
	storageBackendID := "local_disk_cache"
	var minifiedParams registry_dto.ProfileParams
	minifiedParams.SetByName("level", "2")
	desiredProfiles := []registry_dto.NamedProfile{
		{
			Name: "minified",
			Profile: registry_dto.DesiredProfile{
				CapabilityName: "minify",
				Params:         minifiedParams,
			},
		},
	}

	t.Run("Metadata-Only Update on Existing Artefact", func(t *testing.T) {
		f := setupTest()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID:         artefactID,
			SourcePath: sourcePath,
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			UpdatedAt:  time.Now().Add(-30 * time.Minute),
			ActualVariants: []registry_dto.Variant{
				{VariantID: "source", StorageKey: "source/hash123.js", Status: registry_dto.VariantStatusReady},
			},
			DesiredProfiles: []registry_dto.NamedProfile{
				{Name: "old_profile", Profile: registry_dto.DesiredProfile{CapabilityName: "old"}},
			},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, nil, storageBackendID, desiredProfiles)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, artefactID, artefact.ID)
		assert.Equal(t, existingArtefact.CreatedAt, artefact.CreatedAt, "CreatedAt should be preserved")
		assert.True(t, artefact.UpdatedAt.After(existingArtefact.UpdatedAt), "UpdatedAt should be newer")
		assert.Len(t, artefact.ActualVariants, 1, "Should preserve existing variants")
		assert.Equal(t, "source", artefact.ActualVariants[0].VariantID)
		assert.Equal(t, desiredProfiles, artefact.DesiredProfiles, "Should update desired profiles")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.PutCallCount))
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.RenameCallCount))
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.DeleteCallCount))
		require.Len(t, capturedActions, 1)
		action := capturedActions[0]
		art := action.Artefact
		assert.Equal(t, registry_dto.ActionTypeUpsertArtefact, action.Type)
		assert.Equal(t, artefactID, art.ID)
		assert.Len(t, art.ActualVariants, 1)
		assert.Equal(t, "source", art.ActualVariants[0].VariantID)
		assert.Equal(t, "source/hash123.js", art.ActualVariants[0].StorageKey)
		assert.Len(t, art.DesiredProfiles, 1)
		assert.Equal(t, "minified", art.DesiredProfiles[0].Name)
		assert.Equal(t, "minify", art.DesiredProfiles[0].Profile.CapabilityName)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})

	t.Run("Metadata-Only Update on Non-Existent Artefact Creates Placeholder", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, nil, storageBackendID, desiredProfiles)

		require.NoError(t, err, "Should succeed and create placeholder artefact")
		require.NotNil(t, artefact)
		assert.Equal(t, artefactID, artefact.ID)
		assert.Len(t, artefact.ActualVariants, 0, "Placeholder should have no variants yet")
		assert.Equal(t, desiredProfiles, artefact.DesiredProfiles, "Should set desired profiles")
		assert.WithinDuration(t, time.Now(), artefact.CreatedAt, time.Second)
		assert.WithinDuration(t, time.Now(), artefact.UpdatedAt, time.Second)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.PutCallCount))
		require.Len(t, capturedActions, 1)
		assert.Equal(t, registry_dto.ActionTypeUpsertArtefact, capturedActions[0].Type)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})
}

func TestRegistryService_UpsertArtefact_InputValidation(t *testing.T) {
	storageBackendID := "local_disk_cache"
	desiredProfiles := []registry_dto.NamedProfile{{Name: "minified", Profile: registry_dto.DesiredProfile{}}}

	t.Run("Empty ArtefactID Should Error", func(t *testing.T) {
		f := setupTest()
		artefact, err := f.service.UpsertArtefact(f.testContext, "", "path/to/file.js", strings.NewReader("data"), storageBackendID, desiredProfiles)
		require.Error(t, err)
		assert.Nil(t, artefact)
		assert.Contains(t, err.Error(), "artefactID cannot be empty")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.GetArtefactCallCount))
	})

	t.Run("Empty SourcePath Should Error", func(t *testing.T) {
		f := setupTest()
		artefact, err := f.service.UpsertArtefact(f.testContext, "artefact-id", "", strings.NewReader("data"), storageBackendID, desiredProfiles)
		require.Error(t, err)
		assert.Nil(t, artefact)
		assert.Contains(t, err.Error(), "sourcePath cannot be empty")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.GetArtefactCallCount))
	})
}

func TestRegistryService_UpsertArtefact_ErrorHandling(t *testing.T) {
	artefactID := "test-artefact"
	sourcePath := "path/to/source.js"
	sourceData := "console.log('hello');"
	storageBackendID := "local_disk_cache"
	desiredProfiles := []registry_dto.NamedProfile{{Name: "minified", Profile: registry_dto.DesiredProfile{}}}

	t.Run("Storage Backend Not Found", func(t *testing.T) {
		f := setupTest()
		invalidBackendID := "non_existent_backend"
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), invalidBackendID, desiredProfiles)

		require.Error(t, err)
		assert.Nil(t, artefact)
		assert.Contains(t, err.Error(), "not configured or found")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.PutCallCount))
	})

	t.Run("Blob Rename Failure", func(t *testing.T) {
		f := setupTest()
		renameErr := errors.New("filesystem error")
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		f.blobStore.PutFunc = func(_ context.Context, _ string, r io.Reader) error {
			_, _ = io.Copy(io.Discard, r)
			return nil
		}
		f.blobStore.ExistsFunc = func(_ context.Context, _ string) (bool, error) { return false, nil }
		f.blobStore.RenameFunc = func(_ context.Context, _, _ string) error { return renameErr }

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.Error(t, err)
		assert.ErrorIs(t, err, renameErr)
		assert.Nil(t, artefact)
		assert.True(t, atomic.LoadInt64(&f.blobStore.DeleteCallCount) > 0)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount))
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.IncrementBlobRefCountCallCount))
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})

	t.Run("MetaStore GetArtefact Failure", func(t *testing.T) {
		f := setupTest()
		metaStoreErr := errors.New("database connection lost")
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, metaStoreErr
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.Error(t, err)
		assert.ErrorIs(t, err, metaStoreErr)
		assert.Nil(t, artefact)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.PutCallCount))
	})

	t.Run("AtomicUpdate Failure", func(t *testing.T) {
		f := setupTest()
		atomicUpdateErr := errors.New("transaction failed")
		wireNewArtefactUpsertMocks(&f)
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error {
			return atomicUpdateErr
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.Error(t, err)
		assert.ErrorIs(t, err, atomicUpdateErr)
		assert.Nil(t, artefact)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})
}

func TestRegistryService_GetArtefact(t *testing.T) {
	t.Run("Cache Hit Returns Cached Artefact", func(t *testing.T) {
		f := setupTestWithCache()
		expected := NewArtefactBuilder("cached-artefact").WithSourceVariant("hash123").Build()
		f.cache.GetFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return expected, nil
		}

		result, err := f.service.GetArtefact(f.testContext, "cached-artefact")

		require.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
		assert.Equal(t, expected.ActualVariants[0].ContentHash, result.ActualVariants[0].ContentHash)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.GetArtefactCallCount))
	})

	t.Run("Cache Miss Falls Through to Store", func(t *testing.T) {
		f := setupTestWithCache()
		expected := NewArtefactBuilder("store-artefact").WithSourceVariant("hash456").Build()
		f.cache.GetFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrCacheMiss
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return expected, nil
		}
		var cachedArtefact *registry_dto.ArtefactMeta
		f.cache.SetFunc = func(_ context.Context, a *registry_dto.ArtefactMeta) { cachedArtefact = a }

		result, err := f.service.GetArtefact(f.testContext, "store-artefact")

		require.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
		assert.Equal(t, expected, cachedArtefact)
	})

	t.Run("Artefact Not Found in Store", func(t *testing.T) {
		f := setupTestWithCache()
		f.cache.GetFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrCacheMiss
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}

		result, err := f.service.GetArtefact(f.testContext, "missing")

		require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
		assert.Nil(t, result)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.cache.SetCallCount))
	})

	t.Run("No Cache Configured Falls Through to Store", func(t *testing.T) {
		f := setupTest()
		expected := NewArtefactBuilder("no-cache-artefact").WithSourceVariant("hash789").Build()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return expected, nil
		}

		result, err := f.service.GetArtefact(f.testContext, "no-cache-artefact")

		require.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
	})

	t.Run("Cache Error Treated as Miss", func(t *testing.T) {
		f := setupTestWithCache()
		expected := NewArtefactBuilder("cache-error-artefact").WithSourceVariant("hashabc").Build()
		f.cache.GetFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("redis connection error")
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return expected, nil
		}
		f.cache.SetFunc = func(_ context.Context, _ *registry_dto.ArtefactMeta) {}

		result, err := f.service.GetArtefact(f.testContext, "cache-error-artefact")

		require.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
	})
}

func TestRegistryService_GetMultipleArtefacts(t *testing.T) {
	t.Run("All Cache Hits", func(t *testing.T) {
		f := setupTestWithCache()
		ids := []string{"art1", "art2", "art3"}
		cached := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("art1").WithSourceVariant("h1").Build(),
			NewArtefactBuilder("art2").WithSourceVariant("h2").Build(),
			NewArtefactBuilder("art3").WithSourceVariant("h3").Build(),
		}
		f.cache.GetMultipleFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, []string) {
			return cached, []string{}
		}

		results, err := f.service.GetMultipleArtefacts(f.testContext, ids)

		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.GetMultipleArtefactsCallCount))
	})

	t.Run("Partial Cache Hit Fetches Misses from Store", func(t *testing.T) {
		f := setupTestWithCache()
		ids := []string{"art1", "art2", "art3"}
		cached := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("art1").WithSourceVariant("h1").Build(),
		}
		fromStore := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("art2").WithSourceVariant("h2").Build(),
			NewArtefactBuilder("art3").WithSourceVariant("h3").Build(),
		}
		f.cache.GetMultipleFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, []string) {
			return cached, []string{"art2", "art3"}
		}
		f.metaStore.GetMultipleArtefactsFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return fromStore, nil
		}
		f.cache.SetMultipleFunc = func(_ context.Context, _ []*registry_dto.ArtefactMeta) {}

		results, err := f.service.GetMultipleArtefacts(f.testContext, ids)

		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("All Cache Misses", func(t *testing.T) {
		f := setupTestWithCache()
		ids := []string{"art1", "art2"}
		fromStore := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("art1").WithSourceVariant("h1").Build(),
			NewArtefactBuilder("art2").WithSourceVariant("h2").Build(),
		}
		f.cache.GetMultipleFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, []string) {
			return []*registry_dto.ArtefactMeta{}, ids
		}
		f.metaStore.GetMultipleArtefactsFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return fromStore, nil
		}
		f.cache.SetMultipleFunc = func(_ context.Context, _ []*registry_dto.ArtefactMeta) {}

		results, err := f.service.GetMultipleArtefacts(f.testContext, ids)

		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("Empty ID List Returns Empty", func(t *testing.T) {
		f := setupTestWithCache()

		results, err := f.service.GetMultipleArtefacts(f.testContext, []string{})

		require.NoError(t, err)
		assert.Len(t, results, 0)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.cache.GetMultipleCallCount))
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.GetMultipleArtefactsCallCount))
	})

	t.Run("No Cache Configured Fetches All from Store", func(t *testing.T) {
		f := setupTest()
		ids := []string{"art1", "art2"}
		fromStore := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("art1").WithSourceVariant("h1").Build(),
			NewArtefactBuilder("art2").WithSourceVariant("h2").Build(),
		}
		f.metaStore.GetMultipleArtefactsFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return fromStore, nil
		}

		results, err := f.service.GetMultipleArtefacts(f.testContext, ids)

		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestRegistryService_ListAllArtefactIDs(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := setupTest()
		expectedIDs := []string{"art1", "art2", "art3"}
		f.metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) { return expectedIDs, nil }

		ids, err := f.service.ListAllArtefactIDs(f.testContext)

		require.NoError(t, err)
		assert.Equal(t, expectedIDs, ids)
	})

	t.Run("Empty List", func(t *testing.T) {
		f := setupTest()
		f.metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) { return []string{}, nil }

		ids, err := f.service.ListAllArtefactIDs(f.testContext)

		require.NoError(t, err)
		assert.Len(t, ids, 0)
	})

	t.Run("Store Error", func(t *testing.T) {
		f := setupTest()
		storeErr := errors.New("database error")
		f.metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) { return nil, storeErr }

		ids, err := f.service.ListAllArtefactIDs(f.testContext)

		require.ErrorIs(t, err, storeErr)
		assert.Nil(t, ids)
	})
}

func TestRegistryService_SearchArtefacts(t *testing.T) {
	t.Run("Simple Tag Query", func(t *testing.T) {
		f := setupTestWithCache()
		query := registry_domain.SearchQuery{SimpleTagQuery: map[string]string{"category": "images"}}
		results := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("img1").WithSourceVariant("h1").Build(),
			NewArtefactBuilder("img2").WithSourceVariant("h2").Build(),
		}
		f.metaStore.SearchArtefactsFunc = func(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
			return results, nil
		}
		f.cache.SetMultipleFunc = func(_ context.Context, _ []*registry_dto.ArtefactMeta) {}

		artefacts, err := f.service.SearchArtefacts(f.testContext, query)

		require.NoError(t, err)
		assert.Len(t, artefacts, 2)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.cache.SetMultipleCallCount))
	})

	t.Run("Empty Results Not Cached", func(t *testing.T) {
		f := setupTestWithCache()
		query := registry_domain.SearchQuery{SimpleTagQuery: map[string]string{"category": "nonexistent"}}
		f.metaStore.SearchArtefactsFunc = func(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
			return []*registry_dto.ArtefactMeta{}, nil
		}

		artefacts, err := f.service.SearchArtefacts(f.testContext, query)

		require.NoError(t, err)
		assert.Len(t, artefacts, 0)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.cache.SetMultipleCallCount))
	})
}

func TestRegistryService_SearchArtefactsByTagValues(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := setupTestWithCache()
		results := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("media1").Build(),
			NewArtefactBuilder("media2").Build(),
		}
		f.metaStore.SearchArtefactsByTagValuesFunc = func(_ context.Context, _ string, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return results, nil
		}
		f.cache.SetMultipleFunc = func(_ context.Context, _ []*registry_dto.ArtefactMeta) {}

		artefacts, err := f.service.SearchArtefactsByTagValues(f.testContext, "category", []string{"images", "videos"})

		require.NoError(t, err)
		assert.Len(t, artefacts, 2)
	})

	t.Run("Empty Tag Values Returns Empty", func(t *testing.T) {
		f := setupTestWithCache()

		artefacts, err := f.service.SearchArtefactsByTagValues(f.testContext, "category", []string{})

		require.NoError(t, err)
		assert.Len(t, artefacts, 0)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.SearchArtefactsByTagValuesCallCount))
	})
}

func TestRegistryService_FindArtefactByVariantStorageKey(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := setupTestWithCache()
		expected := NewArtefactBuilder("found-artefact").WithSourceVariant("abc123").Build()
		f.metaStore.FindArtefactByVariantStorageKeyFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return expected, nil
		}
		f.cache.SetFunc = func(_ context.Context, _ *registry_dto.ArtefactMeta) {}

		result, err := f.service.FindArtefactByVariantStorageKey(f.testContext, "source/abc123.js")

		require.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.cache.SetCallCount))
	})

	t.Run("Not Found", func(t *testing.T) {
		f := setupTestWithCache()
		f.metaStore.FindArtefactByVariantStorageKeyFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}

		result, err := f.service.FindArtefactByVariantStorageKey(f.testContext, "source/nonexistent.js")

		require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
		assert.Nil(t, result)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.cache.SetCallCount))
	})
}

func TestRegistryService_GetVariantData(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := setupTest()
		expectedData := io.NopCloser(strings.NewReader("file content"))
		f.blobStore.GetFunc = func(_ context.Context, key string) (io.ReadCloser, error) {
			if key == "source/hash123.js" {
				return expectedData, nil
			}
			return nil, registry_domain.ErrBlobNotFound
		}

		data, err := f.service.GetVariantData(f.testContext, new(NewVariantBuilder("source").WithStorageKey("source/hash123.js").Build()))

		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("Storage Backend Not Found", func(t *testing.T) {
		f := setupTest()
		data, err := f.service.GetVariantData(f.testContext, new(NewVariantBuilder("source").WithStorageBackend("unknown_backend").Build()))

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "not found")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.GetCallCount))
	})

	t.Run("Blob Not Found", func(t *testing.T) {
		f := setupTest()
		f.blobStore.GetFunc = func(_ context.Context, _ string) (io.ReadCloser, error) {
			return nil, registry_domain.ErrBlobNotFound
		}

		data, err := f.service.GetVariantData(f.testContext, new(NewVariantBuilder("source").WithStorageKey("missing/key.js").Build()))

		require.ErrorIs(t, err, registry_domain.ErrBlobNotFound)
		assert.Nil(t, data)
	})
}

func TestRegistryService_GetVariantChunk(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := setupTest()
		expectedData := io.NopCloser(strings.NewReader("chunk data"))
		f.blobStore.GetFunc = func(_ context.Context, key string) (io.ReadCloser, error) {
			if key == "chunks/1.bin" {
				return expectedData, nil
			}
			return nil, registry_domain.ErrBlobNotFound
		}

		data, err := f.service.GetVariantChunk(f.testContext, new(NewVariantBuilder("video").
			WithChunk("chunk-0", "chunks/0.bin", 0).
			WithChunk("chunk-1", "chunks/1.bin", 1).
			Build()), "chunk-1")

		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("Chunk Not Found", func(t *testing.T) {
		f := setupTest()
		data, err := f.service.GetVariantChunk(f.testContext, new(NewVariantBuilder("video").
			WithChunk("chunk-0", "chunks/0.bin", 0).
			Build()), "missing-chunk")

		require.ErrorIs(t, err, registry_domain.ErrChunkNotFound)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.GetCallCount))
	})

	t.Run("Chunk Backend Not Found", func(t *testing.T) {
		f := setupTest()
		variant := registry_dto.Variant{
			VariantID: "video",
			Chunks: []registry_dto.VariantChunk{
				{ChunkID: "chunk-0", StorageKey: "chunks/0.bin", StorageBackendID: "unknown_backend"},
			},
		}

		data, err := f.service.GetVariantChunk(f.testContext, &variant, "chunk-0")

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "not found")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.GetCallCount))
	})
}

func TestRegistryService_GetVariantDataRange(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := setupTest()
		expectedData := io.NopCloser(strings.NewReader("partial content"))
		f.blobStore.RangeGetFunc = func(_ context.Context, _ string, _ int64, _ int64) (io.ReadCloser, error) {
			return expectedData, nil
		}

		data, err := f.service.GetVariantDataRange(f.testContext, new(NewVariantBuilder("video").WithStorageKey("video/file.mp4").Build()), 100, 50)

		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("Invalid Offset (Negative)", func(t *testing.T) {
		f := setupTest()
		data, err := f.service.GetVariantDataRange(f.testContext, new(NewVariantBuilder("video").WithStorageKey("video/file.mp4").Build()), -1, 50)

		require.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.RangeGetCallCount))
	})

	t.Run("Invalid Length (Zero)", func(t *testing.T) {
		f := setupTest()
		data, err := f.service.GetVariantDataRange(f.testContext, new(NewVariantBuilder("video").WithStorageKey("video/file.mp4").Build()), 100, 0)

		require.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
		assert.Nil(t, data)
	})

	t.Run("Invalid Length (Negative)", func(t *testing.T) {
		f := setupTest()
		data, err := f.service.GetVariantDataRange(f.testContext, new(NewVariantBuilder("video").WithStorageKey("video/file.mp4").Build()), 100, -10)

		require.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
		assert.Nil(t, data)
	})

	t.Run("Backend Not Found", func(t *testing.T) {
		f := setupTest()
		data, err := f.service.GetVariantDataRange(f.testContext, new(NewVariantBuilder("video").WithStorageBackend("unknown_backend").Build()), 100, 50)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "not found")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.RangeGetCallCount))
	})

	t.Run("Blob Not Found", func(t *testing.T) {
		f := setupTest()
		f.blobStore.RangeGetFunc = func(_ context.Context, _ string, _ int64, _ int64) (io.ReadCloser, error) {
			return nil, registry_domain.ErrBlobNotFound
		}

		data, err := f.service.GetVariantDataRange(f.testContext, new(NewVariantBuilder("video").WithStorageKey("video/missing.mp4").Build()), 0, 100)

		require.ErrorIs(t, err, registry_domain.ErrBlobNotFound)
		assert.Nil(t, data)
	})
}

func TestRegistryService_GetBlobStore(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := setupTest()
		store, err := f.service.GetBlobStore("local_disk_cache")
		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("Not Found", func(t *testing.T) {
		f := setupTest()
		store, err := f.service.GetBlobStore("nonexistent_backend")
		require.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "not configured or found")
	})
}

func TestRegistryService_PopGCHints(t *testing.T) {
	t.Run("Success With Hints", func(t *testing.T) {
		f := setupTest()
		expectedHints := []registry_dto.GCHint{
			{BackendID: "local", StorageKey: "old/key1.js"},
			{BackendID: "local", StorageKey: "old/key2.js"},
		}
		f.metaStore.PopGCHintsFunc = func(_ context.Context, limit int) ([]registry_dto.GCHint, error) {
			assert.Equal(t, 10, limit)
			return expectedHints, nil
		}

		hints, err := f.service.PopGCHints(f.testContext, 10)

		require.NoError(t, err)
		assert.Len(t, hints, 2)
		assert.Equal(t, expectedHints, hints)
	})

	t.Run("No Hints Available", func(t *testing.T) {
		f := setupTest()
		f.metaStore.PopGCHintsFunc = func(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
			return []registry_dto.GCHint{}, nil
		}

		hints, err := f.service.PopGCHints(f.testContext, 10)

		require.NoError(t, err)
		assert.Len(t, hints, 0)
	})

	t.Run("Store Error", func(t *testing.T) {
		f := setupTest()
		storeErr := errors.New("database error")
		f.metaStore.PopGCHintsFunc = func(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
			return nil, storeErr
		}

		hints, err := f.service.PopGCHints(f.testContext, 10)

		require.ErrorIs(t, err, storeErr)
		assert.Nil(t, hints)
	})
}
func TestValidateUpsertInput(t *testing.T) {
	t.Run("Valid Input", func(t *testing.T) {
		err := registry_domain.ValidateUpsertInput("artefact-id", "path/to/file.js")
		assert.NoError(t, err)
	})

	t.Run("Empty ArtefactID", func(t *testing.T) {
		err := registry_domain.ValidateUpsertInput("", "path/to/file.js")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "artefactID cannot be empty")
	})

	t.Run("Empty SourcePath", func(t *testing.T) {
		err := registry_domain.ValidateUpsertInput("artefact-id", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourcePath cannot be empty")
	})

	t.Run("Both Empty", func(t *testing.T) {
		err := registry_domain.ValidateUpsertInput("", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "artefactID cannot be empty")
	})
}

func TestDetectMimeType(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "JavaScript", path: "file.js", expected: "javascript"},
		{name: "CSS", path: "styles.css", expected: "css"},
		{name: "PNG", path: "image.png", expected: "png"},
		{name: "JPEG", path: "photo.jpg", expected: "jpeg"},
		{name: "WebP", path: "image.webp", expected: "webp"},
		{name: "HTML", path: "page.html", expected: "html"},
		{name: "JSON", path: "data.json", expected: "json"},
		{name: "Unknown Extension", path: "file.xyz", expected: "application/octet-stream"},
		{name: "No Extension", path: "README", expected: "application/octet-stream"},
		{name: "Nested Path", path: "path/to/deep/file.js", expected: "javascript"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := registry_domain.DetectMimeType(tc.path)
			assert.Contains(t, result, tc.expected)
		})
	}
}

func TestBuildDependencyMap(t *testing.T) {
	t.Run("Builds Correct Map", func(t *testing.T) {
		var minifiedDeps, gzippedDeps, webpDeps registry_dto.Dependencies
		minifiedDeps.Add("source")
		gzippedDeps.Add("minified")
		webpDeps.Add("source")

		profiles := []registry_dto.NamedProfile{
			{Name: "source", Profile: registry_dto.DesiredProfile{}},
			{Name: "minified", Profile: registry_dto.DesiredProfile{DependsOn: minifiedDeps}},
			{Name: "gzipped", Profile: registry_dto.DesiredProfile{DependsOn: gzippedDeps}},
			{Name: "webp", Profile: registry_dto.DesiredProfile{DependsOn: webpDeps}},
		}

		depMap := registry_domain.BuildDependencyMap(profiles)

		assert.Equal(t, "source", depMap["minified"])
		assert.Equal(t, "minified", depMap["gzipped"])
		assert.Equal(t, "source", depMap["webp"])
		assert.NotContains(t, depMap, "source")
	})

	t.Run("Empty Profiles", func(t *testing.T) {
		depMap := registry_domain.BuildDependencyMap([]registry_dto.NamedProfile{})
		assert.Len(t, depMap, 0)
	})

	t.Run("No Dependencies", func(t *testing.T) {
		profiles := []registry_dto.NamedProfile{
			{Name: "standalone1", Profile: registry_dto.DesiredProfile{}},
			{Name: "standalone2", Profile: registry_dto.DesiredProfile{}},
		}

		depMap := registry_domain.BuildDependencyMap(profiles)

		assert.Len(t, depMap, 0)
	})
}

func TestGetInvalidatedVariants(t *testing.T) {
	t.Run("Cascading Invalidation", func(t *testing.T) {
		depMap := map[string]string{
			"minified": "source",
			"gzipped":  "minified",
			"brotli":   "minified",
		}
		initial := map[string]struct{}{"source": {}}

		invalidated := registry_domain.GetInvalidatedVariants(depMap, initial)

		assert.Contains(t, invalidated, "source")
		assert.Contains(t, invalidated, "minified")
		assert.Contains(t, invalidated, "gzipped")
		assert.Contains(t, invalidated, "brotli")
	})

	t.Run("Partial Invalidation", func(t *testing.T) {
		depMap := map[string]string{
			"minified": "source",
			"webp":     "source",
			"gzipped":  "minified",
		}
		initial := map[string]struct{}{"minified": {}}

		invalidated := registry_domain.GetInvalidatedVariants(depMap, initial)

		assert.Contains(t, invalidated, "minified")
		assert.Contains(t, invalidated, "gzipped")
		assert.NotContains(t, invalidated, "source")
		assert.NotContains(t, invalidated, "webp")
	})

	t.Run("No Dependencies", func(t *testing.T) {
		depMap := map[string]string{}
		initial := map[string]struct{}{"source": {}}

		invalidated := registry_domain.GetInvalidatedVariants(depMap, initial)

		assert.Contains(t, invalidated, "source")
		assert.Len(t, invalidated, 1)
	})

	t.Run("Diamond Dependency", func(t *testing.T) {
		depMap := map[string]string{
			"minified": "source",
			"webp":     "source",
			"final":    "minified",
		}
		initial := map[string]struct{}{"source": {}}

		invalidated := registry_domain.GetInvalidatedVariants(depMap, initial)

		assert.Contains(t, invalidated, "source")
		assert.Contains(t, invalidated, "minified")
		assert.Contains(t, invalidated, "webp")
		assert.Contains(t, invalidated, "final")
	})
}

func TestFindVariantByID(t *testing.T) {
	variants := []registry_dto.Variant{
		{VariantID: "source", StorageKey: "source/key1"},
		{VariantID: "minified", StorageKey: "gen/key2"},
		{VariantID: "gzipped", StorageKey: "gen/key3"},
	}

	t.Run("Found", func(t *testing.T) {
		result := registry_domain.FindVariantByID(variants, "minified")

		require.NotNil(t, result)
		assert.Equal(t, "minified", result.VariantID)
		assert.Equal(t, "gen/key2", result.StorageKey)
	})

	t.Run("Not Found", func(t *testing.T) {
		result := registry_domain.FindVariantByID(variants, "nonexistent")

		assert.Nil(t, result)
	})

	t.Run("Empty Slice", func(t *testing.T) {
		result := registry_domain.FindVariantByID([]registry_dto.Variant{}, "source")

		assert.Nil(t, result)
	})
}

func TestDeduplicateAndSort(t *testing.T) {
	t.Run("Deduplicates and Sorts", func(t *testing.T) {
		input := []string{"c", "a", "b", "a", "c", "d"}

		result := registry_domain.DeduplicateAndSort(input)

		assert.Equal(t, []string{"a", "b", "c", "d"}, result)
	})

	t.Run("Already Unique and Sorted", func(t *testing.T) {
		input := []string{"a", "b", "c"}

		result := registry_domain.DeduplicateAndSort(input)

		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("Empty Slice", func(t *testing.T) {
		result := registry_domain.DeduplicateAndSort([]string{})

		assert.Len(t, result, 0)
	})

	t.Run("Single Element", func(t *testing.T) {
		result := registry_domain.DeduplicateAndSort([]string{"only"})

		assert.Equal(t, []string{"only"}, result)
	})

	t.Run("All Duplicates", func(t *testing.T) {
		input := []string{"same", "same", "same"}

		result := registry_domain.DeduplicateAndSort(input)

		assert.Equal(t, []string{"same"}, result)
	})
}

func TestOrderArtefactsByIDs(t *testing.T) {
	artefacts := []*registry_dto.ArtefactMeta{
		{ID: "b"},
		{ID: "a"},
		{ID: "c"},
	}

	t.Run("Reorders to Match Request", func(t *testing.T) {
		requestedOrder := []string{"c", "a", "b"}

		ordered := registry_domain.OrderArtefactsByIDs(artefacts, requestedOrder)

		require.Len(t, ordered, 3)
		assert.Equal(t, "c", ordered[0].ID)
		assert.Equal(t, "a", ordered[1].ID)
		assert.Equal(t, "b", ordered[2].ID)
	})

	t.Run("Handles Missing IDs", func(t *testing.T) {
		requestedOrder := []string{"c", "missing", "a"}

		ordered := registry_domain.OrderArtefactsByIDs(artefacts, requestedOrder)

		require.Len(t, ordered, 2)
		assert.Equal(t, "c", ordered[0].ID)
		assert.Equal(t, "a", ordered[1].ID)
	})

	t.Run("Empty Artefacts", func(t *testing.T) {
		ordered := registry_domain.OrderArtefactsByIDs([]*registry_dto.ArtefactMeta{}, []string{"a", "b"})

		assert.Len(t, ordered, 0)
	})

	t.Run("Empty Requested IDs", func(t *testing.T) {
		ordered := registry_domain.OrderArtefactsByIDs(artefacts, []string{})

		assert.Len(t, ordered, 0)
	})
}

func TestGetOldSourceStorageKey(t *testing.T) {
	t.Run("Returns Key for Existing Source", func(t *testing.T) {
		artefact := NewArtefactBuilder("test").
			WithSourceVariant("hash123").
			Build()

		key := registry_domain.GetOldSourceStorageKey(artefact, false)

		assert.Equal(t, "source/hash123.js", key)
	})

	t.Run("Returns Empty for New Artefact", func(t *testing.T) {
		artefact := NewArtefactBuilder("test").Build()

		key := registry_domain.GetOldSourceStorageKey(artefact, true)

		assert.Equal(t, "", key)
	})

	t.Run("Returns Empty for Nil Artefact", func(t *testing.T) {
		key := registry_domain.GetOldSourceStorageKey(nil, false)

		assert.Equal(t, "", key)
	})

	t.Run("Returns Empty for Artefact Without Source Variant", func(t *testing.T) {
		artefact := NewArtefactBuilder("test").
			WithVariant("minified", "gen/min.js").
			Build()

		key := registry_domain.GetOldSourceStorageKey(artefact, false)

		assert.Equal(t, "", key)
	})
}

func TestBuildArtefactMeta(t *testing.T) {
	variants := []registry_dto.Variant{
		{VariantID: "source", StorageKey: "source/hash.js"},
	}
	profiles := []registry_dto.NamedProfile{
		{Name: "minified", Profile: registry_dto.DesiredProfile{CapabilityName: "minify"}},
	}

	t.Run("New Artefact", func(t *testing.T) {
		result := registry_domain.BuildArtefactMeta(
			"test-id",
			"path/to/file.js",
			variants,
			profiles,
			true,
			nil,
		)

		assert.Equal(t, "test-id", result.ID)
		assert.Equal(t, "path/to/file.js", result.SourcePath)
		assert.Equal(t, variants, result.ActualVariants)
		assert.Equal(t, profiles, result.DesiredProfiles)
		assert.WithinDuration(t, time.Now(), result.CreatedAt, time.Second)
		assert.WithinDuration(t, time.Now(), result.UpdatedAt, time.Second)
	})

	t.Run("Update Preserves CreatedAt", func(t *testing.T) {
		existingCreatedAt := time.Now().Add(-24 * time.Hour)
		existing := &registry_dto.ArtefactMeta{
			ID:        "test-id",
			CreatedAt: existingCreatedAt,
		}

		result := registry_domain.BuildArtefactMeta(
			"test-id",
			"path/to/file.js",
			variants,
			profiles,
			false,
			existing,
		)

		assert.Equal(t, existingCreatedAt, result.CreatedAt)
		assert.True(t, result.UpdatedAt.After(existingCreatedAt))
	})
}

func TestCreateSourceVariant(t *testing.T) {
	variant := registry_domain.CreateSourceVariant(
		1024,
		"tmp/uuid",
		"abc123def",
		"source/abc123def.js",
		"text/javascript",
		"local_disk_cache",
	)

	assert.Equal(t, "source", variant.VariantID)
	assert.Equal(t, "source/abc123def.js", variant.StorageKey)
	assert.Equal(t, "local_disk_cache", variant.StorageBackendID)
	assert.Equal(t, "text/javascript", variant.MimeType)
	assert.Equal(t, int64(1024), variant.SizeBytes)
	assert.Equal(t, "abc123def", variant.ContentHash)
	assert.Equal(t, registry_dto.VariantStatusReady, variant.Status)
	assert.Equal(t, "source", variant.MetadataTags.Get(registry_dto.TagType))
	assert.Equal(t, "abc123def", variant.MetadataTags.Get(registry_dto.TagHash))
}
func TestAddVariant_WithChunks(t *testing.T) {
	artefactID := "video.mp4"
	existingArtefact := &registry_dto.ArtefactMeta{
		ID: artefactID,
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source/video_hash.mp4"},
		},
	}

	hlsVariant := registry_dto.Variant{
		VariantID:        "hls_playlist",
		StorageKey:       "video/playlist.m3u8",
		StorageBackendID: "local",
		MimeType:         "application/x-mpegURL",
		SizeBytes:        2048,
		ContentHash:      "playlist_hash_abc",
		Chunks: []registry_dto.VariantChunk{
			{
				ChunkID:          "segment-0",
				StorageKey:       "video/segment-0.ts",
				StorageBackendID: "local",
				SizeBytes:        1024,
				ContentHash:      "chunk0_hash_def",
				SequenceNumber:   0,
				MimeType:         "video/MP2T",
				CreatedAt:        time.Now(),
			},
			{
				ChunkID:          "segment-1",
				StorageKey:       "video/segment-1.ts",
				StorageBackendID: "local",
				SizeBytes:        1024,
				ContentHash:      "chunk1_hash_ghi",
				SequenceNumber:   1,
				MimeType:         "video/MP2T",
				CreatedAt:        time.Now(),
			},
		},
		CreatedAt: time.Now(),
		Status:    registry_dto.VariantStatusReady,
	}

	t.Run("Increments Blob Ref Counts for Variant and All Chunks", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		var incrementedKeys []string
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			incrementedKeys = append(incrementedKeys, ref.StorageKey)
			return 1, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		artefact, err := f.service.AddVariant(f.testContext, artefactID, &hlsVariant)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Contains(t, incrementedKeys, "video/playlist.m3u8")
		assert.Contains(t, incrementedKeys, "video/segment-0.ts")
		assert.Contains(t, incrementedKeys, "video/segment-1.ts")
		assert.Equal(t, int64(3), atomic.LoadInt64(&f.metaStore.IncrementBlobRefCountCallCount))
	})
}

func TestAddVariant_ReplaceVariantWithChunks(t *testing.T) {
	artefactID := "video.mp4"

	oldHLSVariant := registry_dto.Variant{
		VariantID:        "hls_playlist",
		StorageKey:       "video/old_playlist.m3u8",
		StorageBackendID: "local",
		SizeBytes:        1500,
		ContentHash:      "old_playlist_hash",
		Chunks: []registry_dto.VariantChunk{
			{
				ChunkID:          "old-segment-0",
				StorageKey:       "video/old-segment-0.ts",
				StorageBackendID: "local",
				SizeBytes:        750,
				ContentHash:      "old_chunk0_hash",
				SequenceNumber:   0,
				MimeType:         "video/MP2T",
				CreatedAt:        time.Now(),
			},
			{
				ChunkID:          "old-segment-1",
				StorageKey:       "video/old-segment-1.ts",
				StorageBackendID: "local",
				SizeBytes:        750,
				ContentHash:      "old_chunk1_hash",
				SequenceNumber:   1,
				MimeType:         "video/MP2T",
				CreatedAt:        time.Now(),
			},
		},
	}

	existingArtefact := &registry_dto.ArtefactMeta{
		ID: artefactID,
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source/video.mp4"},
			oldHLSVariant,
		},
	}

	newHLSVariant := registry_dto.Variant{
		VariantID:        "hls_playlist",
		StorageKey:       "video/new_playlist.m3u8",
		StorageBackendID: "local",
		SizeBytes:        2000,
		MimeType:         "application/x-mpegURL",
		ContentHash:      "new_playlist_hash",
		Chunks: []registry_dto.VariantChunk{
			{
				ChunkID:          "new-segment-0",
				StorageKey:       "video/new-segment-0.ts",
				StorageBackendID: "local",
				SizeBytes:        1000,
				ContentHash:      "new_chunk0_hash",
				SequenceNumber:   0,
				MimeType:         "video/MP2T",
				CreatedAt:        time.Now(),
			},
		},
		CreatedAt: time.Now(),
		Status:    registry_dto.VariantStatusReady,
	}

	t.Run("Decrements Old Chunk Blob Ref Counts and Creates GC Hints", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 1, nil
		}
		f.metaStore.DecrementBlobRefCountFunc = func(_ context.Context, _ string) (int, bool, error) {
			return 0, true, nil
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		artefact, err := f.service.AddVariant(f.testContext, artefactID, &newHLSVariant)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		require.Len(t, capturedActions, 2)
		gcAction := capturedActions[1]
		assert.Equal(t, registry_dto.ActionTypeAddGCHints, gcAction.Type)
		require.Len(t, gcAction.GCHints, 3)
		storageKeys := make(map[string]bool)
		for _, hint := range gcAction.GCHints {
			storageKeys[hint.StorageKey] = true
		}
		assert.True(t, storageKeys["video/old_playlist.m3u8"])
		assert.True(t, storageKeys["video/old-segment-0.ts"])
		assert.True(t, storageKeys["video/old-segment-1.ts"])
		t.Logf("SUCCESS: Old chunks properly dereferenced and marked for GC")
	})
}

func TestAddVariant_ChunkDeduplication(t *testing.T) {
	artefactID := "video.mp4"
	sharedChunkHash := "shared_intro_hash"
	sharedChunkKey := "video/shared-intro.ts"

	existingArtefact := &registry_dto.ArtefactMeta{
		ID: artefactID,
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source/video.mp4"},
			{
				VariantID:        "hls_720p",
				StorageKey:       "video/720p.m3u8",
				StorageBackendID: "local",
				SizeBytes:        1024,
				ContentHash:      "720p_hash",
				Chunks: []registry_dto.VariantChunk{
					{
						ChunkID:          "intro",
						StorageKey:       sharedChunkKey,
						StorageBackendID: "local",
						SizeBytes:        512,
						ContentHash:      sharedChunkHash,
						SequenceNumber:   0,
						MimeType:         "video/MP2T",
						CreatedAt:        time.Now(),
					},
				},
			},
		},
	}

	newVariant := registry_dto.Variant{
		VariantID:        "hls_1080p",
		StorageKey:       "video/1080p.m3u8",
		StorageBackendID: "local",
		SizeBytes:        2048,
		ContentHash:      "1080p_hash",
		MimeType:         "application/x-mpegURL",
		Chunks: []registry_dto.VariantChunk{
			{
				ChunkID:          "intro",
				StorageKey:       sharedChunkKey,
				StorageBackendID: "local",
				SizeBytes:        512,
				ContentHash:      sharedChunkHash,
				SequenceNumber:   0,
				MimeType:         "video/MP2T",
				CreatedAt:        time.Now(),
			},
		},
		CreatedAt: time.Now(),
		Status:    registry_dto.VariantStatusReady,
	}

	t.Run("Shared Chunk Increments Ref Count Without Duplication", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		var sharedChunkRefCount int
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			if ref.StorageKey == sharedChunkKey {
				sharedChunkRefCount = 2
				return 2, nil
			}
			return 1, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		_, err := f.service.AddVariant(f.testContext, artefactID, &newVariant)

		require.NoError(t, err)
		assert.Equal(t, 2, sharedChunkRefCount)
		t.Logf("SUCCESS: Shared chunk blob ref count incremented to 2 (deduplication working)")
	})
}

func TestDeleteArtefact_WithChunks(t *testing.T) {
	artefactID := "video.mp4"
	artefactWithChunks := &registry_dto.ArtefactMeta{
		ID: artefactID,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:        "source",
				StorageBackendID: "local",
				StorageKey:       "source/video.mp4",
			},
			{
				VariantID:        "hls",
				StorageBackendID: "local",
				StorageKey:       "video/playlist.m3u8",
				SizeBytes:        1024,
				ContentHash:      "playlist_hash",
				Chunks: []registry_dto.VariantChunk{
					{
						ChunkID:          "chunk-0",
						StorageKey:       "video/chunk-0.ts",
						StorageBackendID: "local",
						SizeBytes:        512,
						ContentHash:      "chunk0_hash",
						SequenceNumber:   0,
						MimeType:         "video/MP2T",
						CreatedAt:        time.Now(),
					},
					{
						ChunkID:          "chunk-1",
						StorageKey:       "video/chunk-1.ts",
						StorageBackendID: "local",
						SizeBytes:        512,
						ContentHash:      "chunk1_hash",
						SequenceNumber:   1,
						MimeType:         "video/MP2T",
						CreatedAt:        time.Now(),
					},
				},
			},
		},
	}

	t.Run("Decrements All Blob Refs Including Chunks", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return artefactWithChunks, nil
		}
		f.metaStore.DecrementBlobRefCountFunc = func(_ context.Context, _ string) (int, bool, error) {
			return 0, true, nil
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		err := f.service.DeleteArtefact(f.testContext, artefactID)

		require.NoError(t, err)
		require.Len(t, capturedActions, 2)
		gcAction := capturedActions[1]
		assert.Equal(t, registry_dto.ActionTypeAddGCHints, gcAction.Type)
		assert.Len(t, gcAction.GCHints, 4)
		assert.Equal(t, int64(4), atomic.LoadInt64(&f.metaStore.DecrementBlobRefCountCallCount))
		t.Logf("SUCCESS: All chunk blobs dereferenced during artefact deletion")
	})
}

func TestIncrementChunkRefCounts_EdgeCases(t *testing.T) {
	artefactID := "test-artefact"

	t.Run("Variant With No Chunks", func(t *testing.T) {
		f := setupTest()
		variantWithoutChunks := registry_dto.Variant{
			VariantID:        "compiled",
			StorageKey:       "gen/compiled.js",
			StorageBackendID: "local",
			SizeBytes:        1024,
			ContentHash:      "compiled_hash",
			Chunks:           []registry_dto.VariantChunk{},
			MimeType:         "application/javascript",
		}

		existingArtefact := &registry_dto.ArtefactMeta{
			ID:             artefactID,
			ActualVariants: []registry_dto.Variant{{VariantID: "source", StorageKey: "src.js"}},
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			assert.Equal(t, "gen/compiled.js", ref.StorageKey)
			return 1, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		_, err := f.service.AddVariant(f.testContext, artefactID, &variantWithoutChunks)

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.metaStore.IncrementBlobRefCountCallCount))
		t.Logf("SUCCESS: Empty chunks array handled correctly")
	})

	t.Run("Chunk With Empty ContentHash Still Increments Ref", func(t *testing.T) {
		f := setupTest()
		variantWithEmptyHashChunk := registry_dto.Variant{
			VariantID:        "hls",
			StorageKey:       "video/playlist.m3u8",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/x-mpegURL",
			ContentHash:      "playlist_hash",
			Chunks: []registry_dto.VariantChunk{
				{
					ChunkID:          "chunk-0",
					StorageKey:       "video/chunk-0.ts",
					StorageBackendID: "local",
					SizeBytes:        512,
					ContentHash:      "",
					SequenceNumber:   0,
					MimeType:         "video/MP2T",
					CreatedAt:        time.Now(),
				},
			},
		}

		existingArtefact := &registry_dto.ArtefactMeta{
			ID:             artefactID,
			ActualVariants: []registry_dto.Variant{{VariantID: "source", StorageKey: "src.mp4"}},
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		var chunkRefIncremented bool
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			if ref.StorageKey == "video/chunk-0.ts" && ref.ContentHash == "" {
				chunkRefIncremented = true
			}
			return 1, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		_, err := f.service.AddVariant(f.testContext, artefactID, &variantWithEmptyHashChunk)

		require.NoError(t, err)
		assert.True(t, chunkRefIncremented)
		t.Logf("SUCCESS: Empty ContentHash handled gracefully (backward compatibility)")
	})
}

func TestAddVariant_ValidationErrors(t *testing.T) {
	artefactID := "test-artefact"
	existingArtefact := &registry_dto.ArtefactMeta{
		ID:             artefactID,
		ActualVariants: []registry_dto.Variant{{VariantID: "source", StorageKey: "src.js"}},
	}

	t.Run("Rejects Variant With Empty StorageKey", func(t *testing.T) {
		f := setupTest()
		invalidVariant := registry_dto.Variant{
			VariantID:        "compiled",
			StorageKey:       "",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/javascript",
			ContentHash:      "hash123",
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &invalidVariant)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty StorageKey")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.IncrementBlobRefCountCallCount))
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount))
	})

	t.Run("Rejects Variant With Zero SizeBytes", func(t *testing.T) {
		f := setupTest()
		invalidVariant := registry_dto.Variant{
			VariantID:        "compiled",
			StorageKey:       "gen/compiled.js",
			StorageBackendID: "local",
			SizeBytes:        0,
			MimeType:         "application/javascript",
			ContentHash:      "hash123",
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &invalidVariant)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SizeBytes")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.IncrementBlobRefCountCallCount))
	})

	t.Run("Rejects Variant With Empty MimeType", func(t *testing.T) {
		f := setupTest()
		invalidVariant := registry_dto.Variant{
			VariantID:        "compiled",
			StorageKey:       "gen/compiled.js",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "",
			ContentHash:      "hash123",
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &invalidVariant)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty MimeType")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.IncrementBlobRefCountCallCount))
	})

	t.Run("Rejects Chunk With Invalid Fields", func(t *testing.T) {
		f := setupTest()
		variantWithInvalidChunk := registry_dto.Variant{
			VariantID:        "hls",
			StorageKey:       "video/playlist.m3u8",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/x-mpegURL",
			ContentHash:      "playlist_hash",
			Chunks: []registry_dto.VariantChunk{
				{
					ChunkID:          "chunk-0",
					StorageKey:       "",
					StorageBackendID: "local",
					SizeBytes:        512,
					ContentHash:      "chunk_hash",
					SequenceNumber:   0,
					MimeType:         "video/MP2T",
					CreatedAt:        time.Now(),
				},
			},
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			if ref.StorageKey == "video/playlist.m3u8" {
				return 1, nil
			}
			return 0, errors.New("unexpected")
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &variantWithInvalidChunk)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty StorageKey")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount))
	})

	t.Run("Rejects Variant With Empty StorageBackendID", func(t *testing.T) {
		f := setupTest()
		invalidVariant := registry_dto.Variant{
			VariantID:        "compiled",
			StorageKey:       "gen/compiled.js",
			StorageBackendID: "",
			SizeBytes:        1024,
			MimeType:         "application/javascript",
			ContentHash:      "hash123",
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &invalidVariant)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty StorageBackendID")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.IncrementBlobRefCountCallCount))
	})

	t.Run("IncrementBlobRefCount Error", func(t *testing.T) {
		f := setupTest()
		validVariant := registry_dto.Variant{
			VariantID:        "compiled",
			StorageKey:       "gen/compiled.js",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/javascript",
			ContentHash:      "hash123",
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 0, errors.New("ref count error")
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &validVariant)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ref count")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount))
	})

	t.Run("Rejects Chunk With Empty StorageBackendID", func(t *testing.T) {
		f := setupTest()
		variantWithBadChunk := registry_dto.Variant{
			VariantID:        "hls",
			StorageKey:       "video/playlist.m3u8",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/x-mpegURL",
			ContentHash:      "playlist_hash",
			Chunks: []registry_dto.VariantChunk{
				{
					ChunkID:          "chunk-0",
					StorageKey:       "video/chunk-0.ts",
					StorageBackendID: "",
					SizeBytes:        512,
					ContentHash:      "chunk_hash",
					SequenceNumber:   0,
					MimeType:         "video/MP2T",
					CreatedAt:        time.Now(),
				},
			},
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			if ref.StorageKey == "video/playlist.m3u8" {
				return 1, nil
			}
			return 0, errors.New("unexpected")
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &variantWithBadChunk)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty StorageBackendID")
	})

	t.Run("Rejects Chunk With Zero SizeBytes", func(t *testing.T) {
		f := setupTest()
		variantWithBadChunk := registry_dto.Variant{
			VariantID:        "hls",
			StorageKey:       "video/playlist.m3u8",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/x-mpegURL",
			ContentHash:      "playlist_hash",
			Chunks: []registry_dto.VariantChunk{
				{
					ChunkID:          "chunk-0",
					StorageKey:       "video/chunk-0.ts",
					StorageBackendID: "local",
					SizeBytes:        0,
					ContentHash:      "chunk_hash",
					SequenceNumber:   0,
					MimeType:         "video/MP2T",
					CreatedAt:        time.Now(),
				},
			},
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			if ref.StorageKey == "video/playlist.m3u8" {
				return 1, nil
			}
			return 0, errors.New("unexpected")
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &variantWithBadChunk)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SizeBytes")
	})

	t.Run("Rejects Chunk With Empty MimeType", func(t *testing.T) {
		f := setupTest()
		variantWithBadChunk := registry_dto.Variant{
			VariantID:        "hls",
			StorageKey:       "video/playlist.m3u8",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/x-mpegURL",
			ContentHash:      "playlist_hash",
			Chunks: []registry_dto.VariantChunk{
				{
					ChunkID:          "chunk-0",
					StorageKey:       "video/chunk-0.ts",
					StorageBackendID: "local",
					SizeBytes:        512,
					ContentHash:      "chunk_hash",
					SequenceNumber:   0,
					MimeType:         "",
					CreatedAt:        time.Now(),
				},
			},
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			if ref.StorageKey == "video/playlist.m3u8" {
				return 1, nil
			}
			return 0, errors.New("unexpected")
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &variantWithBadChunk)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty MimeType")
	})

	t.Run("Chunk IncrementBlobRefCount Error", func(t *testing.T) {
		f := setupTest()
		variantWithChunk := registry_dto.Variant{
			VariantID:        "hls",
			StorageKey:       "video/playlist.m3u8",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/x-mpegURL",
			ContentHash:      "playlist_hash",
			Chunks: []registry_dto.VariantChunk{
				{
					ChunkID:          "chunk-0",
					StorageKey:       "video/chunk-0.ts",
					StorageBackendID: "local",
					SizeBytes:        512,
					ContentHash:      "chunk_hash",
					SequenceNumber:   0,
					MimeType:         "video/MP2T",
					CreatedAt:        time.Now(),
				},
			},
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, ref registry_domain.BlobReference) (int, error) {
			if ref.StorageKey == "video/playlist.m3u8" {
				return 1, nil
			}
			return 0, errors.New("chunk ref error")
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &variantWithChunk)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chunk")
	})
}

func TestRegistryService_HealthChecks(t *testing.T) {
	t.Run("Name returns RegistryService", func(t *testing.T) {
		f := setupTest()
		probe, ok := f.service.(healthprobe_domain.Probe)
		require.True(t, ok, "RegistryService should implement healthprobe_domain.Probe")
		assert.Equal(t, "RegistryService", probe.Name())
	})

	t.Run("ArtefactEventsPublished returns zero initially", func(t *testing.T) {
		f := setupTest()
		assert.Equal(t, int64(0), f.service.ArtefactEventsPublished())
	})

	t.Run("Check liveness with healthy metaStore", func(t *testing.T) {
		f := setupTest()
		probe, ok := f.service.(healthprobe_domain.Probe)
		require.True(t, ok, "expected service to implement healthprobe_domain.Probe")

		status := probe.Check(f.testContext, healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, "RegistryService", status.Name)
		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Equal(t, "Registry service is running", status.Message)
	})

	t.Run("Check liveness with nil metaStore", func(t *testing.T) {
		service := registry_domain.NewRegistryService(nil, nil, nil, nil)
		probe, ok := service.(healthprobe_domain.Probe)
		require.True(t, ok, "expected service to implement healthprobe_domain.Probe")

		status := probe.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Equal(t, "Metadata store is not initialised", status.Message)
	})

	t.Run("Check readiness all healthy", func(t *testing.T) {
		f := setupTestWithHealthyBlobStore()
		f.metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) { return []string{"a"}, nil }

		healthyBlobStore, ok := f.blobStores["local_disk_cache"].(*registry_domain.MockHealthyBlobStore)
		require.True(t, ok, "expected blob store to be *registry_domain.MockHealthyBlobStore")
		healthyBlobStore.NameFunc = func() string { return "TestBlobStore" }
		healthyBlobStore.CheckFunc = func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
			return healthprobe_dto.Status{
				Name:    "TestBlobStore",
				State:   healthprobe_dto.StateHealthy,
				Message: "Blob store is healthy",
			}
		}

		probe, ok := f.service.(healthprobe_domain.Probe)
		require.True(t, ok, "expected service to implement healthprobe_domain.Probe")
		status := probe.Check(f.testContext, healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Contains(t, status.Message, "1 blob store")
		assert.NotEmpty(t, status.Dependencies)
	})

	t.Run("Check readiness with unhealthy metaStore", func(t *testing.T) {
		f := setupTest()
		f.metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) {
			return nil, errors.New("db down")
		}

		probe, ok := f.service.(healthprobe_domain.Probe)
		require.True(t, ok, "expected service to implement healthprobe_domain.Probe")
		status := probe.Check(f.testContext, healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Contains(t, status.Message, "storage issues")
	})

	t.Run("Check readiness with blob store not supporting health checks", func(t *testing.T) {
		f := setupTest()
		f.metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) { return []string{}, nil }

		probe, ok := f.service.(healthprobe_domain.Probe)
		require.True(t, ok, "expected service to implement healthprobe_domain.Probe")
		status := probe.Check(f.testContext, healthprobe_dto.CheckTypeReadiness)

		foundSkipped := false
		for _, dependency := range status.Dependencies {
			if dependency.Message == "Blob store does not support health checks (skipped)" {
				foundSkipped = true
				assert.Equal(t, healthprobe_dto.StateHealthy, dependency.State)
			}
		}
		assert.True(t, foundSkipped, "Should have a skipped blob store dependency")
	})

	t.Run("Check readiness with unhealthy blob store", func(t *testing.T) {
		f := setupTestWithHealthyBlobStore()
		f.metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) { return []string{}, nil }

		healthyBlobStore, ok := f.blobStores["local_disk_cache"].(*registry_domain.MockHealthyBlobStore)
		require.True(t, ok, "expected blob store to be *registry_domain.MockHealthyBlobStore")
		healthyBlobStore.NameFunc = func() string { return "TestBlobStore" }
		healthyBlobStore.CheckFunc = func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
			return healthprobe_dto.Status{
				Name:    "TestBlobStore",
				State:   healthprobe_dto.StateUnhealthy,
				Message: "Blob store is down",
			}
		}

		probe, ok := f.service.(healthprobe_domain.Probe)
		require.True(t, ok, "expected service to implement healthprobe_domain.Probe")
		status := probe.Check(f.testContext, healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Contains(t, status.Message, "storage issues")
	})

	t.Run("ArtefactEventsPublished increments on successful publish", func(t *testing.T) {
		f := setupTest()
		wireNewArtefactUpsertMocks(&f)

		_, err := f.service.UpsertArtefact(f.testContext, "event-test", "file.js", strings.NewReader("data"), "local_disk_cache", nil)
		require.NoError(t, err)

		assert.Equal(t, int64(1), f.service.ArtefactEventsPublished())
	})
}
func TestAggregateState(t *testing.T) {
	testCases := []struct {
		name     string
		current  healthprobe_dto.State
		incoming healthprobe_dto.State
		expected healthprobe_dto.State
	}{
		{name: "healthy+healthy=healthy", current: healthprobe_dto.StateHealthy, incoming: healthprobe_dto.StateHealthy, expected: healthprobe_dto.StateHealthy},
		{name: "healthy+degraded=degraded", current: healthprobe_dto.StateHealthy, incoming: healthprobe_dto.StateDegraded, expected: healthprobe_dto.StateDegraded},
		{name: "healthy+unhealthy=unhealthy", current: healthprobe_dto.StateHealthy, incoming: healthprobe_dto.StateUnhealthy, expected: healthprobe_dto.StateUnhealthy},
		{name: "degraded+healthy=degraded", current: healthprobe_dto.StateDegraded, incoming: healthprobe_dto.StateHealthy, expected: healthprobe_dto.StateDegraded},
		{name: "degraded+degraded=degraded", current: healthprobe_dto.StateDegraded, incoming: healthprobe_dto.StateDegraded, expected: healthprobe_dto.StateDegraded},
		{name: "degraded+unhealthy=unhealthy", current: healthprobe_dto.StateDegraded, incoming: healthprobe_dto.StateUnhealthy, expected: healthprobe_dto.StateUnhealthy},
		{name: "unhealthy+healthy=unhealthy", current: healthprobe_dto.StateUnhealthy, incoming: healthprobe_dto.StateHealthy, expected: healthprobe_dto.StateUnhealthy},
		{name: "unhealthy+degraded=unhealthy", current: healthprobe_dto.StateUnhealthy, incoming: healthprobe_dto.StateDegraded, expected: healthprobe_dto.StateUnhealthy},
		{name: "unhealthy+unhealthy=unhealthy", current: healthprobe_dto.StateUnhealthy, incoming: healthprobe_dto.StateUnhealthy, expected: healthprobe_dto.StateUnhealthy},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := registry_domain.AggregateState(tc.current, tc.incoming)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRegistryService_UpsertArtefact_LifecycleErrors(t *testing.T) {
	artefactID := "test-artefact"
	sourcePath := "path/to/source.js"
	sourceData := "console.log('hello');"
	storageBackendID := "local_disk_cache"
	desiredProfiles := []registry_dto.NamedProfile{{Name: "minified", Profile: registry_dto.DesiredProfile{}}}

	t.Run("IncrementBlobRefCount failure", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		f.blobStore.PutFunc = func(_ context.Context, _ string, r io.Reader) error {
			_, _ = io.Copy(io.Discard, r)
			return nil
		}
		f.blobStore.ExistsFunc = func(_ context.Context, _ string) (bool, error) { return false, nil }
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 0, errors.New("ref count db error")
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ref count")
		assert.Nil(t, artefact)
	})

	t.Run("Blob existence check error falls through to upload", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		f.blobStore.PutFunc = func(_ context.Context, _ string, r io.Reader) error {
			_, _ = io.Copy(io.Discard, r)
			return nil
		}
		f.blobStore.ExistsFunc = func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("exists check failed")
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 1, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.NoError(t, err)
		require.NotNil(t, artefact)
	})

	t.Run("Blob deduplication hit deletes temp", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		f.blobStore.PutFunc = func(_ context.Context, _ string, r io.Reader) error {
			_, _ = io.Copy(io.Discard, r)
			return nil
		}
		f.blobStore.ExistsFunc = func(_ context.Context, _ string) (bool, error) { return true, nil }
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 2, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, strings.NewReader(sourceData), storageBackendID, desiredProfiles)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.blobStore.RenameCallCount))
		assert.True(t, atomic.LoadInt64(&f.blobStore.DeleteCallCount) > 0)
	})

	t.Run("Skips upsert when profiles match", func(t *testing.T) {
		f := setupTest()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID:              artefactID,
			SourcePath:      sourcePath,
			DesiredProfiles: desiredProfiles,
			ActualVariants:  []registry_dto.Variant{{VariantID: "source"}},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath, nil, storageBackendID, desiredProfiles)

		require.NoError(t, err)
		assert.Equal(t, existingArtefact, artefact)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount))
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})
}

func TestRegistryService_PublishEvent(t *testing.T) {
	t.Run("nil eventBus skips publication", func(t *testing.T) {
		metaStore := &registry_domain.MockMetadataStore{}
		blobStore := &registry_domain.MockBlobStore{}
		blobStores := map[string]registry_domain.BlobStore{"local_disk_cache": blobStore}

		service := registry_domain.NewRegistryService(metaStore, blobStores, nil, nil)

		metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		blobStore.PutFunc = func(_ context.Context, _ string, r io.Reader) error {
			_, _ = io.Copy(io.Discard, r)
			return nil
		}
		blobStore.ExistsFunc = func(_ context.Context, _ string) (bool, error) { return false, nil }
		metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 1, nil
		}
		metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		artefact, err := service.UpsertArtefact(context.Background(), "test", "file.js", strings.NewReader("data"), "local_disk_cache", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, int64(0), service.ArtefactEventsPublished())
	})

	t.Run("eventBus.Publish error does not fail operation", func(t *testing.T) {
		f := setupTest()
		wireNewArtefactUpsertMocks(&f)
		f.eventBus.PublishFunc = func(_ context.Context, _ string, _ orchestrator_domain.Event) error {
			return errors.New("event bus failure")
		}

		artefact, err := f.service.UpsertArtefact(f.testContext, "test", "file.js", strings.NewReader("data"), "local_disk_cache", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.eventBus.PublishCallCount))
	})
}

func TestRegistryService_AddVariant_MoreErrors(t *testing.T) {
	artefactID := "test-artefact"

	t.Run("GetArtefact failure returns error", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("db error")
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, new(NewVariantBuilder("compiled").Build()))

		require.Error(t, err)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.IncrementBlobRefCountCallCount))
	})

	t.Run("PersistVariantUpdate failure", func(t *testing.T) {
		f := setupTest()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID:             artefactID,
			ActualVariants: []registry_dto.Variant{{VariantID: "source", StorageKey: "src.js"}},
		}
		validVariant := registry_dto.Variant{
			VariantID:        "compiled",
			StorageKey:       "gen/compiled.js",
			StorageBackendID: "local",
			SizeBytes:        1024,
			MimeType:         "application/javascript",
			ContentHash:      "hash123",
		}

		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.IncrementBlobRefCountFunc = func(_ context.Context, _ registry_domain.BlobReference) (int, error) {
			return 1, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error {
			return errors.New("atomic update failed")
		}

		_, err := f.service.AddVariant(f.testContext, artefactID, &validVariant)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "atomic update")
	})
}

func TestRegistryService_DeleteArtefact_MoreErrors(t *testing.T) {
	artefactID := "test-artefact"

	t.Run("MetaStore error (not ErrArtefactNotFound)", func(t *testing.T) {
		f := setupTest()
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("db connection lost")
		}

		err := f.service.DeleteArtefact(f.testContext, artefactID)

		require.Error(t, err)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount))
	})

	t.Run("AtomicUpdate failure", func(t *testing.T) {
		f := setupTest()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID: artefactID,
			ActualVariants: []registry_dto.Variant{
				{VariantID: "source", StorageBackendID: "local", StorageKey: "key1"},
			},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.DecrementBlobRefCountFunc = func(_ context.Context, _ string) (int, bool, error) {
			return 0, true, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error {
			return errors.New("atomic failed")
		}

		err := f.service.DeleteArtefact(f.testContext, artefactID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "atomic")
	})

	t.Run("DecrementBlobRefCount error aborts", func(t *testing.T) {
		f := setupTest()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID: artefactID,
			ActualVariants: []registry_dto.Variant{
				{VariantID: "source", StorageBackendID: "local", StorageKey: "key1"},
				{VariantID: "minified", StorageBackendID: "local", StorageKey: "key2"},
			},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		var callNum int64
		f.metaStore.DecrementBlobRefCountFunc = func(_ context.Context, key string) (int, bool, error) {
			n := atomic.AddInt64(&callNum, 1)
			if n == 1 {
				return 0, false, errors.New("decrement failed")
			}
			return 0, true, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

		err := f.service.DeleteArtefact(f.testContext, artefactID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "decrement failed")
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.metaStore.AtomicUpdateCallCount),
			"AtomicUpdate should not be called when decrement fails")
	})

	t.Run("Blob not marked for deletion", func(t *testing.T) {
		f := setupTest()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID: artefactID,
			ActualVariants: []registry_dto.Variant{
				{VariantID: "source", StorageBackendID: "local", StorageKey: "key1"},
			},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.DecrementBlobRefCountFunc = func(_ context.Context, _ string) (int, bool, error) {
			return 1, false, nil
		}
		var capturedActions []registry_dto.AtomicAction
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
			capturedActions = actions
			return nil
		}

		err := f.service.DeleteArtefact(f.testContext, artefactID)

		require.NoError(t, err)
		require.Len(t, capturedActions, 1)
		assert.Equal(t, registry_dto.ActionTypeDeleteArtefact, capturedActions[0].Type)
	})

	t.Run("Delete with cache clears cache", func(t *testing.T) {
		f := setupTestWithCache()
		existingArtefact := &registry_dto.ArtefactMeta{
			ID:             artefactID,
			ActualVariants: []registry_dto.Variant{},
		}
		f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return existingArtefact, nil
		}
		f.metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }
		f.cache.DeleteFunc = func(_ context.Context, id string) {
			assert.Equal(t, artefactID, id)
		}

		err := f.service.DeleteArtefact(f.testContext, artefactID)

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&f.cache.DeleteCallCount))
	})
}

func TestRegistryService_GetVariantData_MoreErrors(t *testing.T) {
	t.Run("Empty StorageBackendID returns error", func(t *testing.T) {
		f := setupTest()
		variant := registry_dto.Variant{
			VariantID:        "source",
			StorageBackendID: "",
			StorageKey:       "source/key.js",
		}

		data, err := f.service.GetVariantData(f.testContext, &variant)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "empty StorageBackendID")
	})

	t.Run("Empty StorageKey returns error", func(t *testing.T) {
		f := setupTest()
		variant := registry_dto.Variant{
			VariantID:        "source",
			StorageBackendID: "local_disk_cache",
			StorageKey:       "",
		}

		data, err := f.service.GetVariantData(f.testContext, &variant)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "empty StorageKey")
	})

	t.Run("Store.Get non-BlobNotFound error", func(t *testing.T) {
		f := setupTest()
		f.blobStore.GetFunc = func(_ context.Context, _ string) (io.ReadCloser, error) {
			return nil, errors.New("disk I/O error")
		}

		data, err := f.service.GetVariantData(f.testContext, new(NewVariantBuilder("source").
			WithStorageKey("source/key.js").
			Build()))

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "disk I/O error")
	})
}

func TestRegistryService_GetVariantDataRange_MoreErrors(t *testing.T) {
	t.Run("Empty StorageBackendID returns error", func(t *testing.T) {
		f := setupTest()
		variant := registry_dto.Variant{
			VariantID:        "source",
			StorageBackendID: "",
			StorageKey:       "source/key.js",
		}

		data, err := f.service.GetVariantDataRange(f.testContext, &variant, 0, 100)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "empty StorageBackendID")
	})

	t.Run("Empty StorageKey returns error", func(t *testing.T) {
		f := setupTest()
		variant := registry_dto.Variant{
			VariantID:        "source",
			StorageBackendID: "local_disk_cache",
			StorageKey:       "",
		}

		data, err := f.service.GetVariantDataRange(f.testContext, &variant, 0, 100)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "empty StorageKey")
	})

	t.Run("RangeNotSatisfiable error from store", func(t *testing.T) {
		f := setupTest()
		f.blobStore.RangeGetFunc = func(_ context.Context, _ string, _ int64, _ int64) (io.ReadCloser, error) {
			return nil, registry_domain.ErrRangeNotSatisfiable
		}

		data, err := f.service.GetVariantDataRange(f.testContext, new(NewVariantBuilder("video").
			WithStorageKey("video/file.mp4").
			Build()), 0, 100)

		require.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
		assert.Nil(t, data)
	})

	t.Run("Non-specific store error", func(t *testing.T) {
		f := setupTest()
		f.blobStore.RangeGetFunc = func(_ context.Context, _ string, _ int64, _ int64) (io.ReadCloser, error) {
			return nil, errors.New("disk I/O error")
		}

		data, err := f.service.GetVariantDataRange(f.testContext, new(NewVariantBuilder("video").
			WithStorageKey("video/file.mp4").
			Build()), 0, 100)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "disk I/O error")
	})
}

func TestRegistryService_GetVariantChunk_MoreErrors(t *testing.T) {
	t.Run("Chunk with empty StorageBackendID", func(t *testing.T) {
		f := setupTest()
		variant := registry_dto.Variant{
			VariantID: "video",
			Chunks: []registry_dto.VariantChunk{
				{
					ChunkID:          "chunk-0",
					StorageKey:       "chunks/0.bin",
					StorageBackendID: "",
				},
			},
		}

		data, err := f.service.GetVariantChunk(f.testContext, &variant, "chunk-0")

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "empty StorageBackendID")
	})

	t.Run("Chunk with empty StorageKey", func(t *testing.T) {
		f := setupTest()
		variant := registry_dto.Variant{
			VariantID: "video",
			Chunks: []registry_dto.VariantChunk{
				{
					ChunkID:          "chunk-0",
					StorageKey:       "",
					StorageBackendID: "local_disk_cache",
				},
			},
		}

		data, err := f.service.GetVariantChunk(f.testContext, &variant, "chunk-0")

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "empty StorageKey")
	})

	t.Run("Chunk blob not found", func(t *testing.T) {
		f := setupTest()
		f.blobStore.GetFunc = func(_ context.Context, _ string) (io.ReadCloser, error) {
			return nil, registry_domain.ErrBlobNotFound
		}

		data, err := f.service.GetVariantChunk(f.testContext, new(NewVariantBuilder("video").
			WithChunk("chunk-0", "chunks/missing.bin", 0).
			Build()), "chunk-0")

		require.ErrorIs(t, err, registry_domain.ErrBlobNotFound)
		assert.Nil(t, data)
	})

	t.Run("Chunk store error (non-BlobNotFound)", func(t *testing.T) {
		f := setupTest()
		f.blobStore.GetFunc = func(_ context.Context, _ string) (io.ReadCloser, error) {
			return nil, errors.New("disk I/O error")
		}

		data, err := f.service.GetVariantChunk(f.testContext, new(NewVariantBuilder("video").
			WithChunk("chunk-0", "chunks/0.bin", 0).
			Build()), "chunk-0")

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "disk I/O error")
	})
}

func TestRegistryService_GetMultipleArtefacts_MoreErrors(t *testing.T) {
	t.Run("Store error without cache", func(t *testing.T) {
		f := setupTest()
		ids := []string{"art1", "art2"}

		f.metaStore.GetMultipleArtefactsFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("db error")
		}

		results, err := f.service.GetMultipleArtefacts(f.testContext, ids)

		require.Error(t, err)
		assert.Nil(t, results)
	})

	t.Run("Store error with cache misses", func(t *testing.T) {
		f := setupTestWithCache()
		ids := []string{"art1", "art2"}

		f.cache.GetMultipleFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, []string) {
			return []*registry_dto.ArtefactMeta{}, ids
		}
		f.metaStore.GetMultipleArtefactsFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("db error")
		}

		results, err := f.service.GetMultipleArtefacts(f.testContext, ids)

		require.Error(t, err)
		assert.Nil(t, results)
	})
}

func TestRegistryService_SearchArtefacts_MoreCases(t *testing.T) {
	t.Run("Raw RediSearch query", func(t *testing.T) {
		f := setupTestWithCache()
		query := registry_domain.SearchQuery{
			RawRediSearchQuery: "@category:{images}",
		}
		results := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("img1").Build(),
		}

		f.metaStore.SearchArtefactsFunc = func(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
			return results, nil
		}
		f.cache.SetMultipleFunc = func(_ context.Context, _ []*registry_dto.ArtefactMeta) {}

		artefacts, err := f.service.SearchArtefacts(f.testContext, query)

		require.NoError(t, err)
		assert.Len(t, artefacts, 1)
	})

	t.Run("Empty search query", func(t *testing.T) {
		f := setupTest()
		query := registry_domain.SearchQuery{}

		f.metaStore.SearchArtefactsFunc = func(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
			return []*registry_dto.ArtefactMeta{}, nil
		}

		artefacts, err := f.service.SearchArtefacts(f.testContext, query)

		require.NoError(t, err)
		assert.Len(t, artefacts, 0)
	})

	t.Run("Search error", func(t *testing.T) {
		f := setupTest()
		query := registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"key": "val"},
		}

		f.metaStore.SearchArtefactsFunc = func(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("search failed")
		}

		artefacts, err := f.service.SearchArtefacts(f.testContext, query)

		require.Error(t, err)
		assert.Nil(t, artefacts)
	})

	t.Run("No cache configured skips SetMultiple", func(t *testing.T) {
		f := setupTest()
		query := registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"category": "images"},
		}
		results := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("img1").Build(),
		}

		f.metaStore.SearchArtefactsFunc = func(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
			return results, nil
		}

		artefacts, err := f.service.SearchArtefacts(f.testContext, query)

		require.NoError(t, err)
		assert.Len(t, artefacts, 1)
	})
}

func TestRegistryService_FindArtefactByVariantStorageKey_StoreError(t *testing.T) {
	t.Run("Store error (not ErrArtefactNotFound)", func(t *testing.T) {
		f := setupTestWithCache()
		storageKey := "source/key.js"

		f.metaStore.FindArtefactByVariantStorageKeyFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("db error")
		}

		result, err := f.service.FindArtefactByVariantStorageKey(f.testContext, storageKey)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, int64(0), atomic.LoadInt64(&f.cache.SetCallCount))
	})
}

func TestRegistryService_SearchArtefactsByTagValues_MoreCases(t *testing.T) {
	t.Run("Store error", func(t *testing.T) {
		f := setupTestWithCache()

		f.metaStore.SearchArtefactsByTagValuesFunc = func(_ context.Context, _ string, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("search failed")
		}

		artefacts, err := f.service.SearchArtefactsByTagValues(f.testContext, "category", []string{"images"})

		require.Error(t, err)
		assert.Nil(t, artefacts)
	})

	t.Run("No cache configured skips SetMultiple", func(t *testing.T) {
		f := setupTest()
		results := []*registry_dto.ArtefactMeta{
			NewArtefactBuilder("img1").Build(),
		}

		f.metaStore.SearchArtefactsByTagValuesFunc = func(_ context.Context, _ string, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return results, nil
		}

		artefacts, err := f.service.SearchArtefactsByTagValues(f.testContext, "category", []string{"images"})

		require.NoError(t, err)
		assert.Len(t, artefacts, 1)
	})
}
func TestConcurrent_SingleflightDeduplicatesGetArtefact(t *testing.T) {
	f := setupTest()
	ctx := f.testContext

	artefact := NewArtefactBuilder("shared-artefact").
		WithSourceVariant("abc123").
		Build()

	var storeCallCount atomic.Int64
	f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		storeCallCount.Add(1)
		time.Sleep(5 * time.Millisecond)
		return artefact, nil
	}

	const goroutines = 50
	var wg sync.WaitGroup
	var ready sync.WaitGroup
	ready.Add(goroutines)

	results := make([]*registry_dto.ArtefactMeta, goroutines)
	errs := make([]error, goroutines)

	for i := range goroutines {
		index := i
		wg.Go(func() {
			ready.Done()
			ready.Wait()
			results[index], errs[index] = f.service.GetArtefact(ctx, "shared-artefact")
		})
	}

	wg.Wait()

	for i := range goroutines {
		require.NoError(t, errs[i], "goroutine %d should not error", i)
		require.NotNil(t, results[i], "goroutine %d should get a result", i)
		assert.Equal(t, "shared-artefact", results[i].ID, "goroutine %d should get the correct artefact", i)
	}

	calls := storeCallCount.Load()
	assert.Less(t, calls, int64(goroutines),
		"expected singleflight to deduplicate at least some calls, but store was called %d times for %d goroutines", calls, goroutines)
	t.Logf("Store was called %d time(s) for %d concurrent requests", calls, goroutines)
}

func TestConcurrent_SingleflightDistinctArtefactIDs(t *testing.T) {
	f := setupTest()
	ctx := f.testContext

	const goroutines = 20

	artefacts := make(map[string]*registry_dto.ArtefactMeta, goroutines)
	for i := range goroutines {
		id := fmt.Sprintf("artefact-%d", i)
		artefacts[id] = NewArtefactBuilder(id).WithSourceVariant("hash").Build()
	}
	f.metaStore.GetArtefactFunc = func(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
		return artefacts[id], nil
	}

	var wg sync.WaitGroup

	results := make([]*registry_dto.ArtefactMeta, goroutines)
	errs := make([]error, goroutines)

	for i := range goroutines {
		index := i
		wg.Go(func() {
			id := fmt.Sprintf("artefact-%d", index)
			results[index], errs[index] = f.service.GetArtefact(ctx, id)
		})
	}

	wg.Wait()

	for i := range goroutines {
		expectedID := fmt.Sprintf("artefact-%d", i)
		require.NoError(t, errs[i], "goroutine %d should not error", i)
		require.NotNil(t, results[i], "goroutine %d should get a result", i)
		assert.Equal(t, expectedID, results[i].ID, "goroutine %d should get the correct artefact", i)
	}
}

func TestConcurrent_SingleflightDeduplicatesFindByStorageKey(t *testing.T) {
	f := setupTest()
	ctx := f.testContext

	artefact := NewArtefactBuilder("found-artefact").
		WithSourceVariant("keyhash").
		Build()

	var storeCallCount atomic.Int64
	f.metaStore.FindArtefactByVariantStorageKeyFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		storeCallCount.Add(1)
		time.Sleep(5 * time.Millisecond)
		return artefact, nil
	}

	const goroutines = 30
	var wg sync.WaitGroup
	var ready sync.WaitGroup
	ready.Add(goroutines)

	results := make([]*registry_dto.ArtefactMeta, goroutines)
	errs := make([]error, goroutines)

	for i := range goroutines {
		index := i
		wg.Go(func() {
			ready.Done()
			ready.Wait()
			results[index], errs[index] = f.service.FindArtefactByVariantStorageKey(ctx, "blobs/keyhash.js")
		})
	}

	wg.Wait()

	for i := range goroutines {
		require.NoError(t, errs[i], "goroutine %d should not error", i)
		require.NotNil(t, results[i], "goroutine %d should get a result", i)
		assert.Equal(t, "found-artefact", results[i].ID)
	}

	calls := storeCallCount.Load()
	assert.Less(t, calls, int64(goroutines),
		"expected singleflight to deduplicate at least some calls, but store was called %d times for %d goroutines", calls, goroutines)
	t.Logf("Store was called %d time(s) for %d concurrent requests", calls, goroutines)
}

func TestConcurrent_MixedReadOperations(t *testing.T) {
	f := setupTestWithCache()
	ctx := f.testContext

	art1 := NewArtefactBuilder("art-1").WithSourceVariant("h1").Build()
	art2 := NewArtefactBuilder("art-2").WithSourceVariant("h2").Build()
	art3 := NewArtefactBuilder("art-3").WithSourceVariant("h3").Build()

	f.metaStore.GetArtefactFunc = func(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
		switch id {
		case "art-1":
			return art1, nil
		case "art-2":
			return art2, nil
		case "art-3":
			return art3, nil
		}
		return nil, registry_domain.ErrArtefactNotFound
	}
	f.metaStore.GetMultipleArtefactsFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
		return []*registry_dto.ArtefactMeta{art1, art2}, nil
	}
	f.metaStore.SearchArtefactsFunc = func(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
		return []*registry_dto.ArtefactMeta{art1}, nil
	}
	f.metaStore.FindArtefactByVariantStorageKeyFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return art3, nil
	}
	f.metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) {
		return []string{"art-1", "art-2", "art-3"}, nil
	}
	f.metaStore.SearchArtefactsByTagValuesFunc = func(_ context.Context, _ string, _ []string) ([]*registry_dto.ArtefactMeta, error) {
		return []*registry_dto.ArtefactMeta{art1, art2, art3}, nil
	}

	f.cache.GetFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrCacheMiss
	}
	f.cache.SetFunc = func(_ context.Context, _ *registry_dto.ArtefactMeta) {}
	f.cache.GetMultipleFunc = func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, []string) {
		return []*registry_dto.ArtefactMeta{}, []string{"art-1", "art-2"}
	}
	f.cache.SetMultipleFunc = func(_ context.Context, _ []*registry_dto.ArtefactMeta) {}

	var wg sync.WaitGroup
	const iterations = 50

	for range 5 {
		wg.Go(func() {
			for range iterations {
				_, _ = f.service.GetArtefact(ctx, "art-1")
				_, _ = f.service.GetArtefact(ctx, "art-2")
			}
		})
	}

	for range 5 {
		wg.Go(func() {
			for range iterations {
				_, _ = f.service.GetMultipleArtefacts(ctx, []string{"art-1", "art-2"})
			}
		})
	}

	for range 3 {
		wg.Go(func() {
			for range iterations {
				_, _ = f.service.SearchArtefacts(ctx, registry_domain.SearchQuery{
					SimpleTagQuery: map[string]string{"type": "source"},
				})
			}
		})
	}

	for range 3 {
		wg.Go(func() {
			for range iterations {
				_, _ = f.service.FindArtefactByVariantStorageKey(ctx, "blobs/h3.js")
			}
		})
	}

	for range 2 {
		wg.Go(func() {
			for range iterations {
				_, _ = f.service.ListAllArtefactIDs(ctx)
			}
		})
	}

	for range 2 {
		wg.Go(func() {
			for range iterations {
				_, _ = f.service.SearchArtefactsByTagValues(ctx, "type", []string{"source"})
			}
		})
	}

	wg.Wait()
}

func TestConcurrent_GetArtefactWithCacheAndSingleflight(t *testing.T) {
	f := setupTestWithCache()
	ctx := f.testContext

	artefact := NewArtefactBuilder("cached-art").WithSourceVariant("hash").Build()

	f.cache.GetFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrCacheMiss
	}
	f.cache.SetFunc = func(_ context.Context, _ *registry_dto.ArtefactMeta) {}
	f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return artefact, nil
	}

	const goroutines = 40
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			for range 10 {
				result, err := f.service.GetArtefact(ctx, "cached-art")
				assert.NoError(t, err)
				if result != nil {
					assert.Equal(t, "cached-art", result.ID)
				}
			}
		})
	}

	wg.Wait()
}

func TestConcurrent_EventCounterAccuracy(t *testing.T) {
	metaStore := &registry_domain.MockMetadataStore{}
	blobStore := &registry_domain.MockBlobStore{}
	eventBus := &registry_domain.MockEventBus{}
	blobStores := map[string]registry_domain.BlobStore{
		"local_disk_cache": blobStore,
	}

	service := registry_domain.NewRegistryService(metaStore, blobStores, eventBus, nil)
	ctx := context.Background()

	const goroutines = 20
	const iterations = 10

	metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrArtefactNotFound
	}
	metaStore.AtomicUpdateFunc = func(_ context.Context, _ []registry_dto.AtomicAction) error { return nil }

	var wg sync.WaitGroup

	for i := range goroutines {
		goroutineIndex := i
		wg.Go(func() {
			for j := range iterations {
				id := fmt.Sprintf("event-art-%d", goroutineIndex*iterations+j)
				_, _ = service.UpsertArtefact(ctx, id, "path/to/file.js", nil, "local_disk_cache", nil)
			}
		})
	}

	wg.Wait()

	counter := service.ArtefactEventsPublished()
	t.Logf("Published %d events from %d concurrent operations", counter, goroutines*iterations)
	assert.Equal(t, int64(goroutines*iterations), counter,
		"event counter should match the total number of successful upserts")
}

func TestConcurrent_HealthChecksDuringOperations(t *testing.T) {
	metaStore := &registry_domain.MockMetadataStore{}
	healthyBlobStore := &registry_domain.MockHealthyBlobStore{}
	eventBus := &registry_domain.MockEventBus{}
	blobStores := map[string]registry_domain.BlobStore{
		"local_disk_cache": healthyBlobStore,
	}

	service := registry_domain.NewRegistryService(metaStore, blobStores, eventBus, nil)
	ctx := context.Background()

	artefact := NewArtefactBuilder("hc-art").WithSourceVariant("hash").Build()

	metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return artefact, nil
	}
	metaStore.ListAllArtefactIDsFunc = func(_ context.Context) ([]string, error) {
		return []string{"hc-art"}, nil
	}
	healthyBlobStore.NameFunc = func() string { return "TestBlobStore" }
	healthyBlobStore.CheckFunc = func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
		return healthprobe_dto.Status{
			Name:    "TestBlobStore",
			State:   healthprobe_dto.StateHealthy,
			Message: "OK",
		}
	}

	probe, ok := service.(healthprobe_domain.Probe)
	require.True(t, ok)

	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			for range 50 {
				status := probe.Check(ctx, healthprobe_dto.CheckTypeReadiness)
				assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
			}
		})
	}

	for range 10 {
		wg.Go(func() {
			for range 50 {
				result, err := service.GetArtefact(ctx, "hc-art")
				assert.NoError(t, err)
				if result != nil {
					assert.Equal(t, "hc-art", result.ID)
				}
			}
		})
	}

	wg.Wait()
}
