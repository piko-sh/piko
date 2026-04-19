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
	"io"
	"sync/atomic"

	"piko.sh/piko/internal/registry/registry_dto"
)

// MockRegistryService is a test double for RegistryService where nil function fields
// return zero values and call counts are tracked atomically.
type MockRegistryService struct {
	// UpsertArtefactFunc is the function called by UpsertArtefact.
	UpsertArtefactFunc func(
		ctx context.Context, artefactID, sourcePath string, sourceData io.Reader,
		storageBackendID string, desiredProfiles []registry_dto.NamedProfile,
	) (*registry_dto.ArtefactMeta, error)

	// AddVariantFunc is the function called by AddVariant.
	AddVariantFunc func(ctx context.Context, artefactID string, newVariant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error)

	// DeleteArtefactFunc is the function called by DeleteArtefact.
	DeleteArtefactFunc func(ctx context.Context, artefactID string) error

	// GetArtefactFunc is the function called by GetArtefact.
	GetArtefactFunc func(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)

	// GetMultipleArtefactsFunc is the function called by GetMultipleArtefacts.
	GetMultipleArtefactsFunc func(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error)

	// ListAllArtefactIDsFunc is the function called by ListAllArtefactIDs.
	ListAllArtefactIDsFunc func(ctx context.Context) ([]string, error)

	// SearchArtefactsFunc is the function called by SearchArtefacts.
	SearchArtefactsFunc func(ctx context.Context, query SearchQuery) ([]*registry_dto.ArtefactMeta, error)

	// SearchArtefactsByTagValuesFunc is the function called by SearchArtefactsByTagValues.
	SearchArtefactsByTagValuesFunc func(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error)

	// FindArtefactByVariantStorageKeyFunc is the function called by
	// FindArtefactByVariantStorageKey.
	FindArtefactByVariantStorageKeyFunc func(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error)

	// GetVariantDataFunc is the function called by GetVariantData.
	GetVariantDataFunc func(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error)

	// GetVariantChunkFunc is the function called by GetVariantChunk.
	GetVariantChunkFunc func(ctx context.Context, variant *registry_dto.Variant, chunkID string) (io.ReadCloser, error)

	// GetVariantDataRangeFunc is the function called by GetVariantDataRange.
	GetVariantDataRangeFunc func(ctx context.Context, variant *registry_dto.Variant, offset, length int64) (io.ReadCloser, error)

	// GetBlobStoreFunc is the function called by GetBlobStore.
	GetBlobStoreFunc func(backendID string) (BlobStore, error)

	// PopGCHintsFunc is the function called by PopGCHints.
	PopGCHintsFunc func(ctx context.Context, limit int) ([]registry_dto.GCHint, error)

	// GetBlobRefCountFunc is the function called by GetBlobRefCount.
	GetBlobRefCountFunc func(ctx context.Context, storageKey string) (int, error)

	// ListBlobStoreIDsFunc is the function called by ListBlobStoreIDs.
	ListBlobStoreIDsFunc func() []string

	// ArtefactEventsPublishedFunc is the function called by ArtefactEventsPublished.
	ArtefactEventsPublishedFunc func() int64

	// UpsertArtefactCallCount tracks how many times UpsertArtefact was called.
	UpsertArtefactCallCount atomic.Int64

	// AddVariantCallCount tracks how many times AddVariant was called.
	AddVariantCallCount atomic.Int64

	// DeleteArtefactCallCount tracks how many times DeleteArtefact was called.
	DeleteArtefactCallCount atomic.Int64

	// GetArtefactCallCount tracks how many times GetArtefact was called.
	GetArtefactCallCount atomic.Int64

	// GetMultipleArtefactsCallCount tracks how many times GetMultipleArtefacts was called.
	GetMultipleArtefactsCallCount atomic.Int64

	// ListAllArtefactIDsCallCount tracks how many times ListAllArtefactIDs was called.
	ListAllArtefactIDsCallCount atomic.Int64

	// SearchArtefactsCallCount tracks how many times SearchArtefacts was called.
	SearchArtefactsCallCount atomic.Int64

	// SearchArtefactsByTagValuesCallCount tracks how many times SearchArtefactsByTagValues
	// was called.
	SearchArtefactsByTagValuesCallCount atomic.Int64

	// FindArtefactByVariantStorageKeyCallCount tracks how many times
	// FindArtefactByVariantStorageKey was called.
	FindArtefactByVariantStorageKeyCallCount atomic.Int64

	// GetVariantDataCallCount tracks how many times GetVariantData was called.
	GetVariantDataCallCount atomic.Int64

	// GetVariantChunkCallCount tracks how many times GetVariantChunk was called.
	GetVariantChunkCallCount atomic.Int64

	// GetVariantDataRangeCallCount tracks how many times GetVariantDataRange was called.
	GetVariantDataRangeCallCount atomic.Int64

	// GetBlobStoreCallCount tracks how many times GetBlobStore was called.
	GetBlobStoreCallCount atomic.Int64

	// PopGCHintsCallCount tracks how many times PopGCHints was called.
	PopGCHintsCallCount atomic.Int64

	// ListBlobStoreIDsCallCount tracks how many times ListBlobStoreIDs was called.
	ListBlobStoreIDsCallCount atomic.Int64

	// ArtefactEventsPublishedCallCount tracks how many times ArtefactEventsPublished was
	// called.
	ArtefactEventsPublishedCallCount atomic.Int64
}

// UpsertArtefact creates or updates an artefact with its source data.
//
// Takes artefactID (string) which identifies the artefact to create or update.
// Takes sourcePath (string) which is the original path of the source file.
// Takes sourceData (io.Reader) which provides the source data to store.
// Takes storageBackendID (string) which identifies the storage backend to use.
// Takes desiredProfiles ([]registry_dto.NamedProfile) which lists the processing profiles
// to apply.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if UpsertArtefactFunc is nil.
func (m *MockRegistryService) UpsertArtefact(
	ctx context.Context,
	artefactID string,
	sourcePath string,
	sourceData io.Reader,
	storageBackendID string,
	desiredProfiles []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	m.UpsertArtefactCallCount.Add(1)
	if m.UpsertArtefactFunc != nil {
		return m.UpsertArtefactFunc(ctx, artefactID, sourcePath, sourceData, storageBackendID, desiredProfiles)
	}
	return nil, nil
}

// AddVariant adds a new variant to an existing artefact.
//
// Takes artefactID (string) which identifies the artefact to add a variant to.
// Takes newVariant (*registry_dto.Variant) which is the variant to add.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if AddVariantFunc is nil.
func (m *MockRegistryService) AddVariant(ctx context.Context, artefactID string, newVariant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
	m.AddVariantCallCount.Add(1)
	if m.AddVariantFunc != nil {
		return m.AddVariantFunc(ctx, artefactID, newVariant)
	}
	return nil, nil
}

// DeleteArtefact removes an artefact by ID.
//
// Takes artefactID (string) which identifies the artefact to delete.
//
// Returns error, or nil if DeleteArtefactFunc is nil.
func (m *MockRegistryService) DeleteArtefact(ctx context.Context, artefactID string) error {
	m.DeleteArtefactCallCount.Add(1)
	if m.DeleteArtefactFunc != nil {
		return m.DeleteArtefactFunc(ctx, artefactID)
	}
	return nil
}

// GetArtefact retrieves artefact metadata by ID.
//
// Takes artefactID (string) which identifies the artefact to look up.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if GetArtefactFunc is nil.
func (m *MockRegistryService) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	m.GetArtefactCallCount.Add(1)
	if m.GetArtefactFunc != nil {
		return m.GetArtefactFunc(ctx, artefactID)
	}
	return nil, nil
}

// GetMultipleArtefacts retrieves metadata for multiple artefacts.
//
// Takes artefactIDs ([]string) which lists the artefact IDs to look up.
//
// Returns ([]*ArtefactMeta, error), or (nil, nil) if GetMultipleArtefactsFunc is nil.
func (m *MockRegistryService) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	m.GetMultipleArtefactsCallCount.Add(1)
	if m.GetMultipleArtefactsFunc != nil {
		return m.GetMultipleArtefactsFunc(ctx, artefactIDs)
	}
	return nil, nil
}

// ListAllArtefactIDs returns all artefact IDs.
//
// Returns ([]string, error), or (nil, nil) if ListAllArtefactIDsFunc is nil.
func (m *MockRegistryService) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	m.ListAllArtefactIDsCallCount.Add(1)
	if m.ListAllArtefactIDsFunc != nil {
		return m.ListAllArtefactIDsFunc(ctx)
	}
	return nil, nil
}

// SearchArtefacts searches for artefacts matching the query.
//
// Takes query (SearchQuery) which defines the search criteria.
//
// Returns ([]*ArtefactMeta, error), or (nil, nil) if SearchArtefactsFunc is nil.
func (m *MockRegistryService) SearchArtefacts(ctx context.Context, query SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	m.SearchArtefactsCallCount.Add(1)
	if m.SearchArtefactsFunc != nil {
		return m.SearchArtefactsFunc(ctx, query)
	}
	return nil, nil
}

// SearchArtefactsByTagValues finds artefacts by tag key and values.
//
// Takes tagKey (string) which is the tag key to filter by.
// Takes tagValues ([]string) which lists the tag values to match.
//
// Returns ([]*ArtefactMeta, error), or (nil, nil) if SearchArtefactsByTagValuesFunc is
// nil.
func (m *MockRegistryService) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	m.SearchArtefactsByTagValuesCallCount.Add(1)
	if m.SearchArtefactsByTagValuesFunc != nil {
		return m.SearchArtefactsByTagValuesFunc(ctx, tagKey, tagValues)
	}
	return nil, nil
}

// FindArtefactByVariantStorageKey looks up an artefact by variant storage key.
//
// Takes storageKey (string) which is the variant storage key to search for.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if FindArtefactByVariantStorageKeyFunc is
// nil.
func (m *MockRegistryService) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	m.FindArtefactByVariantStorageKeyCallCount.Add(1)
	if m.FindArtefactByVariantStorageKeyFunc != nil {
		return m.FindArtefactByVariantStorageKeyFunc(ctx, storageKey)
	}
	return nil, nil
}

// GetVariantData retrieves the full data for a variant.
//
// Takes variant (*registry_dto.Variant) which identifies the variant to retrieve.
//
// Returns (io.ReadCloser, error), or (nil, nil) if GetVariantDataFunc is nil.
func (m *MockRegistryService) GetVariantData(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
	m.GetVariantDataCallCount.Add(1)
	if m.GetVariantDataFunc != nil {
		return m.GetVariantDataFunc(ctx, variant)
	}
	return nil, nil
}

// GetVariantChunk retrieves a specific chunk of variant data.
//
// Takes variant (*registry_dto.Variant) which identifies the variant to read from.
// Takes chunkID (string) which identifies the specific chunk to retrieve.
//
// Returns (io.ReadCloser, error), or (nil, nil) if GetVariantChunkFunc is nil.
func (m *MockRegistryService) GetVariantChunk(ctx context.Context, variant *registry_dto.Variant, chunkID string) (io.ReadCloser, error) {
	m.GetVariantChunkCallCount.Add(1)
	if m.GetVariantChunkFunc != nil {
		return m.GetVariantChunkFunc(ctx, variant, chunkID)
	}
	return nil, nil
}

// GetVariantDataRange retrieves a byte range of variant data.
//
// Takes variant (*registry_dto.Variant) which identifies the variant to read from.
// Takes offset (int64) which is the byte position to start reading from.
// Takes length (int64) which is the number of bytes to read.
//
// Returns (io.ReadCloser, error), or (nil, nil) if GetVariantDataRangeFunc is nil.
func (m *MockRegistryService) GetVariantDataRange(ctx context.Context, variant *registry_dto.Variant, offset, length int64) (io.ReadCloser, error) {
	m.GetVariantDataRangeCallCount.Add(1)
	if m.GetVariantDataRangeFunc != nil {
		return m.GetVariantDataRangeFunc(ctx, variant, offset, length)
	}
	return nil, nil
}

// GetBlobStore returns the blob store for the given backend ID.
//
// Takes backendID (string) which identifies the storage backend.
//
// Returns (BlobStore, error), or (nil, nil) if GetBlobStoreFunc is nil.
func (m *MockRegistryService) GetBlobStore(backendID string) (BlobStore, error) {
	m.GetBlobStoreCallCount.Add(1)
	if m.GetBlobStoreFunc != nil {
		return m.GetBlobStoreFunc(backendID)
	}
	return nil, nil
}

// PopGCHints pops garbage collection hints up to the given limit.
//
// Takes limit (int) which is the maximum number of hints to return.
//
// Returns ([]GCHint, error), or (nil, nil) if PopGCHintsFunc is nil.
func (m *MockRegistryService) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	m.PopGCHintsCallCount.Add(1)
	if m.PopGCHintsFunc != nil {
		return m.PopGCHintsFunc(ctx, limit)
	}
	return nil, nil
}

// GetBlobRefCount returns the current reference count for a blob.
//
// Takes storageKey (string) which identifies the blob to count references for.
//
// Returns (int, error), or (0, nil) if GetBlobRefCountFunc is nil.
func (m *MockRegistryService) GetBlobRefCount(ctx context.Context, storageKey string) (int, error) {
	if m.GetBlobRefCountFunc != nil {
		return m.GetBlobRefCountFunc(ctx, storageKey)
	}
	return 0, nil
}

// ListBlobStoreIDs returns the identifiers of all registered blob storage backends.
//
// Returns []string, or nil if ListBlobStoreIDsFunc is nil.
func (m *MockRegistryService) ListBlobStoreIDs() []string {
	m.ListBlobStoreIDsCallCount.Add(1)
	if m.ListBlobStoreIDsFunc != nil {
		return m.ListBlobStoreIDsFunc()
	}
	return nil
}

// ArtefactEventsPublished returns the number of artefact events published.
//
// Returns int64, or 0 if ArtefactEventsPublishedFunc is nil.
func (m *MockRegistryService) ArtefactEventsPublished() int64 {
	m.ArtefactEventsPublishedCallCount.Add(1)
	if m.ArtefactEventsPublishedFunc != nil {
		return m.ArtefactEventsPublishedFunc()
	}
	return 0
}
