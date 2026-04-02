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
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

var _ registry_domain.MetadataStore = (*MockMetadataStore)(nil)

// MockMetadataStore is a thread-safe, in-memory implementation of MetadataStore
// and MetadataDAL for testing. It does not save data to disk.
type MockMetadataStore struct {
	// artefacts maps artefact IDs to their metadata.
	artefacts map[string]*registry_dto.ArtefactMeta

	// variantIndex maps variant storage keys to artefact IDs.
	variantIndex map[string]string

	// blobRefs maps storage keys to their reference count data.
	blobRefs map[string]*blobRefData

	// mu guards concurrent access to the artefacts map.
	mu sync.RWMutex
}

// blobRefData holds a blob reference with its current reference count.
type blobRefData struct {
	// Metadata holds the blob reference details including the last referenced time.
	Metadata registry_domain.BlobReference

	// RefCount tracks how many manifests reference this blob; when it reaches
	// zero the blob may be deleted.
	RefCount int
}

// NewMockMetadataStore creates a new in-memory metadata store for testing.
//
// Returns *MockMetadataStore which is an empty store ready for use.
func NewMockMetadataStore() *MockMetadataStore {
	return &MockMetadataStore{
		artefacts:    make(map[string]*registry_dto.ArtefactMeta),
		variantIndex: make(map[string]string),
		blobRefs:     make(map[string]*blobRefData),
		mu:           sync.RWMutex{},
	}
}

// GetArtefact retrieves a single artefact by ID from the mock store.
//
// Takes artefactID (string) which identifies the artefact to retrieve.
//
// Returns *registry_dto.ArtefactMeta which is a cloned copy of the artefact.
// Returns error when the artefact does not exist.
//
// Safe for concurrent use.
func (m *MockMetadataStore) GetArtefact(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	artefact, exists := m.artefacts[artefactID]
	if !exists {
		return nil, fmt.Errorf("artefact not found: %s", artefactID)
	}

	return cloneArtefactMeta(artefact), nil
}

// GetMultipleArtefacts retrieves multiple artefacts by their IDs.
//
// Takes artefactIDs ([]string) which specifies the artefact IDs to retrieve.
//
// Returns []*registry_dto.ArtefactMeta which contains the found artefacts.
// Returns error when retrieval fails.
//
// Safe for concurrent use.
func (m *MockMetadataStore) GetMultipleArtefacts(_ context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	for _, id := range artefactIDs {
		if artefact, exists := m.artefacts[id]; exists {
			result = append(result, cloneArtefactMeta(artefact))
		}
	}

	return result, nil
}

// ListAllArtefactIDs returns all artefact IDs in the store.
//
// Returns []string which contains all artefact IDs currently stored.
// Returns error when retrieval fails.
//
// Safe for concurrent use.
func (m *MockMetadataStore) ListAllArtefactIDs(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.artefacts))
	for id := range m.artefacts {
		ids = append(ids, id)
	}

	return ids, nil
}

// SearchArtefacts searches for artefacts matching the given query.
// This mock implementation returns all artefacts regardless of query.
//
// Returns []*registry_dto.ArtefactMeta which contains cloned copies of all
// stored artefacts.
// Returns error which is always nil in this mock implementation.
//
// Safe for concurrent use. Uses a read lock to protect access to the
// internal artefacts map.
func (m *MockMetadataStore) SearchArtefacts(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*registry_dto.ArtefactMeta, 0, len(m.artefacts))
	for _, artefact := range m.artefacts {
		result = append(result, cloneArtefactMeta(artefact))
	}

	return result, nil
}

// SearchArtefactsByTagValues searches for artefacts with variants matching
// the given tag key and values.
//
// Takes tagKey (string) which specifies the metadata tag name to match.
// Takes tagValues ([]string) which lists the acceptable values for the tag.
//
// Returns []*registry_dto.ArtefactMeta which contains matching artefacts.
// Returns error when the search fails.
//
// Safe for concurrent use; holds a read lock during the search.
func (m *MockMetadataStore) SearchArtefactsByTagValues(_ context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	valueSet := make(map[string]bool)
	for _, v := range tagValues {
		valueSet[v] = true
	}

	result := make([]*registry_dto.ArtefactMeta, 0)
	for _, artefact := range m.artefacts {
		for i := range artefact.ActualVariants {
			variant := &artefact.ActualVariants[i]
			if tagValue, exists := variant.MetadataTags.GetByName(tagKey); exists && valueSet[tagValue] {
				result = append(result, cloneArtefactMeta(artefact))
				break
			}
		}
	}

	return result, nil
}

// FindArtefactByVariantStorageKey finds an artefact by a variant's storage key.
//
// Takes storageKey (string) which identifies the variant to look up.
//
// Returns *registry_dto.ArtefactMeta which is a clone of the matching artefact.
// Returns error when no variant exists with the given key or when the internal
// index is corrupted.
//
// Safe for concurrent use.
func (m *MockMetadataStore) FindArtefactByVariantStorageKey(_ context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	artefactID, exists := m.variantIndex[storageKey]
	if !exists {
		return nil, fmt.Errorf("no artefact found with variant storage key: %s", storageKey)
	}

	artefact, exists := m.artefacts[artefactID]
	if !exists {
		return nil, fmt.Errorf("artefact index corrupted: variant points to non-existent artefact %s", artefactID)
	}

	return cloneArtefactMeta(artefact), nil
}

// PopGCHints returns and removes garbage collection hints.
// This mock implementation always returns an empty slice.
//
// Returns []registry_dto.GCHint which is always empty in this mock.
// Returns error which is always nil in this mock.
func (*MockMetadataStore) PopGCHints(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
	return []registry_dto.GCHint{}, nil
}

// AtomicUpdate performs a batch of atomic operations on the store.
//
// Takes actions ([]registry_dto.AtomicAction) which specifies the operations
// to perform.
//
// Returns error when an unknown action type is encountered.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockMetadataStore) AtomicUpdate(_ context.Context, actions []registry_dto.AtomicAction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, action := range actions {
		switch action.Type {
		case registry_dto.ActionTypeUpsertArtefact:
			artefact := action.Artefact
			m.artefacts[artefact.ID] = cloneArtefactMeta(artefact)

			for i := range artefact.ActualVariants {
				variant := &artefact.ActualVariants[i]
				m.variantIndex[variant.StorageKey] = artefact.ID
			}

		case registry_dto.ActionTypeDeleteArtefact:
			if artefact, exists := m.artefacts[action.ArtefactID]; exists {
				for i := range artefact.ActualVariants {
					variant := &artefact.ActualVariants[i]
					delete(m.variantIndex, variant.StorageKey)
				}
				delete(m.artefacts, action.ArtefactID)
			}

		case registry_dto.ActionTypeAddGCHints:

		default:
			return fmt.Errorf("unknown action type: %s", action.Type)
		}
	}

	return nil
}

// IncrementBlobRefCount increments the reference count for a blob.
//
// Takes blob (registry_domain.BlobReference) which identifies the blob to
// increment.
//
// Returns int which is the new reference count after incrementing.
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockMetadataStore) IncrementBlobRefCount(_ context.Context, blob registry_domain.BlobReference) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ref, exists := m.blobRefs[blob.StorageKey]
	if !exists {
		m.blobRefs[blob.StorageKey] = &blobRefData{
			Metadata: blob,
			RefCount: 1,
		}
		return 1, nil
	}

	ref.RefCount++
	ref.Metadata.LastReferencedAt = time.Now()
	return ref.RefCount, nil
}

// DecrementBlobRefCount decrements the reference count for a blob.
//
// Takes storageKey (string) which identifies the blob to decrement.
//
// Returns int which is the new reference count, or zero if deleted.
// Returns bool which is true when the blob was deleted.
// Returns error when the blob does not exist.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockMetadataStore) DecrementBlobRefCount(_ context.Context, storageKey string) (int, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ref, exists := m.blobRefs[storageKey]
	if !exists {
		return 0, false, registry_domain.ErrBlobReferenceNotFound
	}

	ref.RefCount--
	if ref.RefCount <= 0 {
		delete(m.blobRefs, storageKey)
		return 0, true, nil
	}

	return ref.RefCount, false, nil
}

// GetBlobRefCount returns the current reference count for a blob.
//
// Takes storageKey (string) which identifies the blob in storage.
//
// Returns int which is the reference count, or zero if the blob does not exist.
// Returns error when the lookup fails.
//
// Safe for concurrent use.
func (m *MockMetadataStore) GetBlobRefCount(_ context.Context, storageKey string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ref, exists := m.blobRefs[storageKey]
	if !exists {
		return 0, nil
	}

	return ref.RefCount, nil
}

// Close releases resources held by this store.
// This mock implementation is a no-op.
//
// Returns error which is always nil in this mock.
func (*MockMetadataStore) Close() error {
	return nil
}

// RunAtomic executes fn within a transaction.
//
// For the mock store, this passes itself as the transaction store.
// Each method already guards its own access, so no additional lock
// is needed here.
//
// Takes fn (func(ctx context.Context,
// transactionStore MetadataStore) error) which receives the
// store to use within the transaction.
//
// Returns error when fn returns an error.
func (m *MockMetadataStore) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	return fn(ctx, m)
}

// cloneArtefactMeta creates a deep copy of artefact metadata to prevent
// external modifications.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the metadata to clone.
//
// Returns *registry_dto.ArtefactMeta which is an independent copy of the input,
// or nil if the input is nil.
func cloneArtefactMeta(artefact *registry_dto.ArtefactMeta) *registry_dto.ArtefactMeta {
	if artefact == nil {
		return nil
	}

	artCopy := &registry_dto.ArtefactMeta{
		ID:              artefact.ID,
		SourcePath:      artefact.SourcePath,
		ActualVariants:  make([]registry_dto.Variant, len(artefact.ActualVariants)),
		CreatedAt:       artefact.CreatedAt,
		UpdatedAt:       artefact.UpdatedAt,
		DesiredProfiles: make([]registry_dto.NamedProfile, len(artefact.DesiredProfiles)),
	}

	copy(artCopy.DesiredProfiles, artefact.DesiredProfiles)

	for i := range artefact.ActualVariants {
		v := &artefact.ActualVariants[i]
		artCopy.ActualVariants[i] = registry_dto.Variant{
			VariantID:        v.VariantID,
			StorageBackendID: v.StorageBackendID,
			StorageKey:       v.StorageKey,
			MimeType:         v.MimeType,
			SizeBytes:        v.SizeBytes,
			CreatedAt:        v.CreatedAt,
			Status:           v.Status,
			MetadataTags:     v.MetadataTags.Clone(),
			ContentHash:      v.ContentHash,
			Chunks:           v.Chunks,
		}
	}

	return artCopy
}
