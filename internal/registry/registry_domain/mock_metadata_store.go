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
	"sync/atomic"

	"piko.sh/piko/internal/registry/registry_dto"
)

// MockMetadataStore is a test double for MetadataStore where nil
// function fields return zero values and call counts are tracked
// atomically.
type MockMetadataStore struct {
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

	// FindArtefactByVariantStorageKeyFunc is the function
	// called by FindArtefactByVariantStorageKey.
	FindArtefactByVariantStorageKeyFunc func(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error)

	// PopGCHintsFunc is the function called by
	// PopGCHints.
	PopGCHintsFunc func(ctx context.Context, limit int) ([]registry_dto.GCHint, error)

	// AtomicUpdateFunc is the function called by
	// AtomicUpdate.
	AtomicUpdateFunc func(ctx context.Context, actions []registry_dto.AtomicAction) error

	// IncrementBlobRefCountFunc is the function called
	// by IncrementBlobRefCount.
	IncrementBlobRefCountFunc func(ctx context.Context, blob BlobReference) (int, error)

	// DecrementBlobRefCountFunc is the function called
	// by DecrementBlobRefCount.
	DecrementBlobRefCountFunc func(ctx context.Context, storageKey string) (int, bool, error)

	// GetBlobRefCountFunc is the function called by
	// GetBlobRefCount.
	GetBlobRefCountFunc func(ctx context.Context, storageKey string) (int, error)

	// RunAtomicFunc is the function called by RunAtomic.
	RunAtomicFunc func(ctx context.Context, fn func(ctx context.Context, transactionStore MetadataStore) error) error

	// CloseFunc is the function called by Close.
	CloseFunc func() error

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

	// PopGCHintsCallCount tracks how many times
	// PopGCHints was called.
	PopGCHintsCallCount int64

	// AtomicUpdateCallCount tracks how many times
	// AtomicUpdate was called.
	AtomicUpdateCallCount int64

	// IncrementBlobRefCountCallCount tracks how many
	// times IncrementBlobRefCount was called.
	IncrementBlobRefCountCallCount int64

	// DecrementBlobRefCountCallCount tracks how many
	// times DecrementBlobRefCount was called.
	DecrementBlobRefCountCallCount int64

	// GetBlobRefCountCallCount tracks how many times
	// GetBlobRefCount was called.
	GetBlobRefCountCallCount int64

	// RunAtomicCallCount tracks how many times
	// RunAtomic was called.
	RunAtomicCallCount int64

	// CloseCallCount tracks how many times Close was
	// called.
	CloseCallCount int64
}

// GetArtefact retrieves artefact metadata by ID.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactID (string) which identifies the artefact to look up.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if GetArtefactFunc is nil.
func (m *MockMetadataStore) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
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
func (m *MockMetadataStore) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.GetMultipleArtefactsCallCount, 1)
	if m.GetMultipleArtefactsFunc != nil {
		return m.GetMultipleArtefactsFunc(ctx, artefactIDs)
	}
	return nil, nil
}

// ListAllArtefactIDs returns all artefact IDs in the store.
//
// Returns ([]string, error), or (nil, nil) if ListAllArtefactIDsFunc is nil.
func (m *MockMetadataStore) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
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
func (m *MockMetadataStore) SearchArtefacts(ctx context.Context, query SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
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
func (m *MockMetadataStore) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
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
func (m *MockMetadataStore) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.FindArtefactByVariantStorageKeyCallCount, 1)
	if m.FindArtefactByVariantStorageKeyFunc != nil {
		return m.FindArtefactByVariantStorageKeyFunc(ctx, storageKey)
	}
	return nil, nil
}

// PopGCHints pops garbage collection hints up to the given limit.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes limit (int) which is the maximum number of hints to return.
//
// Returns ([]GCHint, error), or (nil, nil) if PopGCHintsFunc is nil.
func (m *MockMetadataStore) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	atomic.AddInt64(&m.PopGCHintsCallCount, 1)
	if m.PopGCHintsFunc != nil {
		return m.PopGCHintsFunc(ctx, limit)
	}
	return nil, nil
}

// AtomicUpdate applies a batch of actions atomically.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes actions ([]registry_dto.AtomicAction) which lists the actions to apply.
//
// Returns error, or nil if AtomicUpdateFunc is nil.
func (m *MockMetadataStore) AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error {
	atomic.AddInt64(&m.AtomicUpdateCallCount, 1)
	if m.AtomicUpdateFunc != nil {
		return m.AtomicUpdateFunc(ctx, actions)
	}
	return nil
}

// IncrementBlobRefCount increments the reference count for a blob.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes blob (BlobReference) which identifies the blob to increment.
//
// Returns (int, error), or (0, nil) if IncrementBlobRefCountFunc is nil.
func (m *MockMetadataStore) IncrementBlobRefCount(ctx context.Context, blob BlobReference) (int, error) {
	atomic.AddInt64(&m.IncrementBlobRefCountCallCount, 1)
	if m.IncrementBlobRefCountFunc != nil {
		return m.IncrementBlobRefCountFunc(ctx, blob)
	}
	return 0, nil
}

// DecrementBlobRefCount decrements the reference count for a blob.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes storageKey (string) which identifies the blob to decrement.
//
// Returns (int, bool, error), or (0, false, nil) if
// DecrementBlobRefCountFunc is nil.
func (m *MockMetadataStore) DecrementBlobRefCount(ctx context.Context, storageKey string) (int, bool, error) {
	atomic.AddInt64(&m.DecrementBlobRefCountCallCount, 1)
	if m.DecrementBlobRefCountFunc != nil {
		return m.DecrementBlobRefCountFunc(ctx, storageKey)
	}
	return 0, false, nil
}

// GetBlobRefCount returns the current reference count for a blob.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes storageKey (string) which identifies the blob to query.
//
// Returns (int, error), or (0, nil) if GetBlobRefCountFunc is nil.
func (m *MockMetadataStore) GetBlobRefCount(ctx context.Context, storageKey string) (int, error) {
	atomic.AddInt64(&m.GetBlobRefCountCallCount, 1)
	if m.GetBlobRefCountFunc != nil {
		return m.GetBlobRefCountFunc(ctx, storageKey)
	}
	return 0, nil
}

// RunAtomic executes fn within a transaction.
//
// Takes fn which receives a transactional
// MetadataStore for atomic operations.
//
// Returns error, or nil if RunAtomicFunc is nil.
func (m *MockMetadataStore) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore MetadataStore) error) error {
	atomic.AddInt64(&m.RunAtomicCallCount, 1)
	if m.RunAtomicFunc != nil {
		return m.RunAtomicFunc(ctx, fn)
	}
	return fn(ctx, m)
}

// Close shuts down the metadata store.
//
// Returns error, or nil if CloseFunc is nil.
func (m *MockMetadataStore) Close() error {
	atomic.AddInt64(&m.CloseCallCount, 1)
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
