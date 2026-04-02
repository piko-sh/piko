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

// MockRegistryService is a test double for RegistryService where nil
// function fields return zero values and call counts are tracked
// atomically.
type MockRegistryService struct {
	// UpsertArtefactFunc is the function called by
	// UpsertArtefact.
	UpsertArtefactFunc func(
		ctx context.Context, artefactID, sourcePath string, sourceData io.Reader,
		storageBackendID string, desiredProfiles []registry_dto.NamedProfile,
	) (*registry_dto.ArtefactMeta, error)

	// AddVariantFunc is the function called by
	// AddVariant.
	AddVariantFunc func(ctx context.Context, artefactID string, newVariant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error)

	// DeleteArtefactFunc is the function called by
	// DeleteArtefact.
	DeleteArtefactFunc func(ctx context.Context, artefactID string) error

	// GetArtefactFunc is the function called by
	// GetArtefact.
	GetArtefactFunc func(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)

	// GetMultipleArtefactsFunc is the function called by
	// GetMultipleArtefacts.
	GetMultipleArtefactsFunc func(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error)

	// ListAllArtefactIDsFunc is the function called by
	// ListAllArtefactIDs.
	ListAllArtefactIDsFunc func(ctx context.Context) ([]string, error)

	// SearchArtefactsFunc is the function called by
	// SearchArtefacts.
	SearchArtefactsFunc func(ctx context.Context, query SearchQuery) ([]*registry_dto.ArtefactMeta, error)

	// SearchArtefactsByTagValuesFunc is the function
	// called by SearchArtefactsByTagValues.
	SearchArtefactsByTagValuesFunc func(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error)

	// FindArtefactByVariantStorageKeyFunc is the
	// function called by FindArtefactByVariantStorageKey.
	FindArtefactByVariantStorageKeyFunc func(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error)

	// GetVariantDataFunc is the function called by
	// GetVariantData.
	GetVariantDataFunc func(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error)

	// GetVariantChunkFunc is the function called by
	// GetVariantChunk.
	GetVariantChunkFunc func(ctx context.Context, variant *registry_dto.Variant, chunkID string) (io.ReadCloser, error)

	// GetVariantDataRangeFunc is the function called by
	// GetVariantDataRange.
	GetVariantDataRangeFunc func(ctx context.Context, variant *registry_dto.Variant, offset, length int64) (io.ReadCloser, error)

	// GetBlobStoreFunc is the function called by
	// GetBlobStore.
	GetBlobStoreFunc func(backendID string) (BlobStore, error)

	// PopGCHintsFunc is the function called by
	// PopGCHints.
	PopGCHintsFunc func(ctx context.Context, limit int) ([]registry_dto.GCHint, error)

	// ListBlobStoreIDsFunc is the function called by
	// ListBlobStoreIDs.
	ListBlobStoreIDsFunc func() []string

	// ArtefactEventsPublishedFunc is the function called
	// by ArtefactEventsPublished.
	ArtefactEventsPublishedFunc func() int64

	// UpsertArtefactCallCount tracks how many times
	// UpsertArtefact was called.
	UpsertArtefactCallCount int64

	// AddVariantCallCount tracks how many times
	// AddVariant was called.
	AddVariantCallCount int64

	// DeleteArtefactCallCount tracks how many times
	// DeleteArtefact was called.
	DeleteArtefactCallCount int64

	// GetArtefactCallCount tracks how many times
	// GetArtefact was called.
	GetArtefactCallCount int64

	// GetMultipleArtefactsCallCount tracks how many
	// times GetMultipleArtefacts was called.
	GetMultipleArtefactsCallCount int64

	// ListAllArtefactIDsCallCount tracks how many times
	// ListAllArtefactIDs was called.
	ListAllArtefactIDsCallCount int64

	// SearchArtefactsCallCount tracks how many times
	// SearchArtefacts was called.
	SearchArtefactsCallCount int64

	// SearchArtefactsByTagValuesCallCount tracks how
	// many times SearchArtefactsByTagValues was called.
	SearchArtefactsByTagValuesCallCount int64

	// FindArtefactByVariantStorageKeyCallCount tracks
	// how many times FindArtefactByVariantStorageKey
	// was called.
	FindArtefactByVariantStorageKeyCallCount int64

	// GetVariantDataCallCount tracks how many times
	// GetVariantData was called.
	GetVariantDataCallCount int64

	// GetVariantChunkCallCount tracks how many times
	// GetVariantChunk was called.
	GetVariantChunkCallCount int64

	// GetVariantDataRangeCallCount tracks how many
	// times GetVariantDataRange was called.
	GetVariantDataRangeCallCount int64

	// GetBlobStoreCallCount tracks how many times
	// GetBlobStore was called.
	GetBlobStoreCallCount int64

	// PopGCHintsCallCount tracks how many times
	// PopGCHints was called.
	PopGCHintsCallCount int64

	// ListBlobStoreIDsCallCount tracks how many times
	// ListBlobStoreIDs was called.
	ListBlobStoreIDsCallCount int64

	// ArtefactEventsPublishedCallCount tracks how many
	// times ArtefactEventsPublished was called.
	ArtefactEventsPublishedCallCount int64
}

// UpsertArtefact creates or updates an artefact with its source data.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactID (string) which identifies the artefact to create or update.
// Takes sourcePath (string) which is the original path of the source file.
// Takes sourceData (io.Reader) which provides the source data to store.
// Takes storageBackendID (string) which identifies the storage backend to use.
// Takes desiredProfiles ([]registry_dto.NamedProfile)
// which lists the processing profiles to apply.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if
// UpsertArtefactFunc is nil.
func (m *MockRegistryService) UpsertArtefact(
	ctx context.Context,
	artefactID string,
	sourcePath string,
	sourceData io.Reader,
	storageBackendID string,
	desiredProfiles []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.UpsertArtefactCallCount, 1)
	if m.UpsertArtefactFunc != nil {
		return m.UpsertArtefactFunc(ctx, artefactID, sourcePath, sourceData, storageBackendID, desiredProfiles)
	}
	return nil, nil
}

// AddVariant adds a new variant to an existing artefact.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactID (string) which identifies the artefact to add a variant to.
// Takes newVariant (*registry_dto.Variant) which is the variant to add.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if AddVariantFunc is nil.
func (m *MockRegistryService) AddVariant(ctx context.Context, artefactID string, newVariant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.AddVariantCallCount, 1)
	if m.AddVariantFunc != nil {
		return m.AddVariantFunc(ctx, artefactID, newVariant)
	}
	return nil, nil
}

// DeleteArtefact removes an artefact by ID.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactID (string) which identifies the artefact to delete.
//
// Returns error, or nil if DeleteArtefactFunc is nil.
func (m *MockRegistryService) DeleteArtefact(ctx context.Context, artefactID string) error {
	atomic.AddInt64(&m.DeleteArtefactCallCount, 1)
	if m.DeleteArtefactFunc != nil {
		return m.DeleteArtefactFunc(ctx, artefactID)
	}
	return nil
}

// GetArtefact retrieves artefact metadata by ID.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactID (string) which identifies the artefact to look up.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if GetArtefactFunc is nil.
func (m *MockRegistryService) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.GetArtefactCallCount, 1)
	if m.GetArtefactFunc != nil {
		return m.GetArtefactFunc(ctx, artefactID)
	}
	return nil, nil
}

// GetMultipleArtefacts retrieves metadata for multiple artefacts.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactIDs ([]string) which lists the artefact IDs to look up.
//
// Returns ([]*ArtefactMeta, error), or (nil, nil) if
// GetMultipleArtefactsFunc is nil.
func (m *MockRegistryService) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.GetMultipleArtefactsCallCount, 1)
	if m.GetMultipleArtefactsFunc != nil {
		return m.GetMultipleArtefactsFunc(ctx, artefactIDs)
	}
	return nil, nil
}

// ListAllArtefactIDs returns all artefact IDs.
//
// Returns ([]string, error), or (nil, nil) if ListAllArtefactIDsFunc is nil.
func (m *MockRegistryService) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	atomic.AddInt64(&m.ListAllArtefactIDsCallCount, 1)
	if m.ListAllArtefactIDsFunc != nil {
		return m.ListAllArtefactIDsFunc(ctx)
	}
	return nil, nil
}

// SearchArtefacts searches for artefacts matching the query.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes query (SearchQuery) which defines the search criteria.
//
// Returns ([]*ArtefactMeta, error), or (nil, nil) if SearchArtefactsFunc
// is nil.
func (m *MockRegistryService) SearchArtefacts(ctx context.Context, query SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.SearchArtefactsCallCount, 1)
	if m.SearchArtefactsFunc != nil {
		return m.SearchArtefactsFunc(ctx, query)
	}
	return nil, nil
}

// SearchArtefactsByTagValues finds artefacts by tag key and values.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes tagKey (string) which is the tag key to filter by.
// Takes tagValues ([]string) which lists the tag values to match.
//
// Returns ([]*ArtefactMeta, error), or (nil, nil) if
// SearchArtefactsByTagValuesFunc is nil.
func (m *MockRegistryService) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.SearchArtefactsByTagValuesCallCount, 1)
	if m.SearchArtefactsByTagValuesFunc != nil {
		return m.SearchArtefactsByTagValuesFunc(ctx, tagKey, tagValues)
	}
	return nil, nil
}

// FindArtefactByVariantStorageKey looks up an artefact by variant storage key.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes storageKey (string) which is the variant storage key to search for.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if
// FindArtefactByVariantStorageKeyFunc is nil.
func (m *MockRegistryService) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.FindArtefactByVariantStorageKeyCallCount, 1)
	if m.FindArtefactByVariantStorageKeyFunc != nil {
		return m.FindArtefactByVariantStorageKeyFunc(ctx, storageKey)
	}
	return nil, nil
}

// GetVariantData retrieves the full data for a variant.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes variant (*registry_dto.Variant) which
// identifies the variant to retrieve.
//
// Returns (io.ReadCloser, error), or (nil, nil) if
// GetVariantDataFunc is nil.
func (m *MockRegistryService) GetVariantData(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
	atomic.AddInt64(&m.GetVariantDataCallCount, 1)
	if m.GetVariantDataFunc != nil {
		return m.GetVariantDataFunc(ctx, variant)
	}
	return nil, nil
}

// GetVariantChunk retrieves a specific chunk of variant data.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes variant (*registry_dto.Variant) which
// identifies the variant to read from.
// Takes chunkID (string) which identifies the specific
// chunk to retrieve.
//
// Returns (io.ReadCloser, error), or (nil, nil) if GetVariantChunkFunc
// is nil.
func (m *MockRegistryService) GetVariantChunk(ctx context.Context, variant *registry_dto.Variant, chunkID string) (io.ReadCloser, error) {
	atomic.AddInt64(&m.GetVariantChunkCallCount, 1)
	if m.GetVariantChunkFunc != nil {
		return m.GetVariantChunkFunc(ctx, variant, chunkID)
	}
	return nil, nil
}

// GetVariantDataRange retrieves a byte range of variant data.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes variant (*registry_dto.Variant) which
// identifies the variant to read from.
// Takes offset (int64) which is the byte position to
// start reading from.
// Takes length (int64) which is the number of bytes to read.
//
// Returns (io.ReadCloser, error), or (nil, nil) if GetVariantDataRangeFunc
// is nil.
func (m *MockRegistryService) GetVariantDataRange(ctx context.Context, variant *registry_dto.Variant, offset, length int64) (io.ReadCloser, error) {
	atomic.AddInt64(&m.GetVariantDataRangeCallCount, 1)
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
	atomic.AddInt64(&m.GetBlobStoreCallCount, 1)
	if m.GetBlobStoreFunc != nil {
		return m.GetBlobStoreFunc(backendID)
	}
	return nil, nil
}

// PopGCHints pops garbage collection hints up to the given limit.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes limit (int) which is the maximum number of hints to return.
//
// Returns ([]GCHint, error), or (nil, nil) if PopGCHintsFunc is nil.
func (m *MockRegistryService) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	atomic.AddInt64(&m.PopGCHintsCallCount, 1)
	if m.PopGCHintsFunc != nil {
		return m.PopGCHintsFunc(ctx, limit)
	}
	return nil, nil
}

// ListBlobStoreIDs returns the identifiers of all registered blob storage
// backends.
//
// Returns []string, or nil if ListBlobStoreIDsFunc is nil.
func (m *MockRegistryService) ListBlobStoreIDs() []string {
	atomic.AddInt64(&m.ListBlobStoreIDsCallCount, 1)
	if m.ListBlobStoreIDsFunc != nil {
		return m.ListBlobStoreIDsFunc()
	}
	return nil
}

// ArtefactEventsPublished returns the number of artefact events published.
//
// Returns int64, or 0 if ArtefactEventsPublishedFunc is nil.
func (m *MockRegistryService) ArtefactEventsPublished() int64 {
	atomic.AddInt64(&m.ArtefactEventsPublishedCallCount, 1)
	if m.ArtefactEventsPublishedFunc != nil {
		return m.ArtefactEventsPublishedFunc()
	}
	return 0
}
