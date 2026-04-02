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
	"time"

	"piko.sh/piko/internal/registry/registry_dto"
)

// SearchQuery holds the settings for finding artefacts in the registry.
// It allows both simple tag-based lookups and raw RediSearch queries.
type SearchQuery struct {
	// SimpleTagQuery maps tag names to their values for basic filtering.
	SimpleTagQuery map[string]string

	// RawRediSearchQuery is a raw RediSearch query string; empty means not used.
	RawRediSearchQuery string
}

// BlobReference represents metadata about a stored blob for reference counting.
// It tracks shared blobs across multiple artefacts to enable safe content
// deduplication.
type BlobReference struct {
	// CreatedAt is when the blob reference was created.
	CreatedAt time.Time

	// LastReferencedAt is when the blob was last used; zero means never.
	LastReferencedAt time.Time

	// StorageKey is the unique identifier for locating this blob in storage.
	StorageKey string

	// StorageBackendID identifies where the blob is stored.
	StorageBackendID string

	// ContentHash is the hash of the blob content, used to check data integrity.
	ContentHash string

	// MimeType is the MIME type of the blob content.
	MimeType string

	// SizeBytes is the size of the blob in bytes.
	SizeBytes int64
}

// MetadataStore defines the interface for persisting and querying artefact
// metadata. Implementations handle storage of artefact records and blob
// reference counting.
type MetadataStore interface {
	// GetArtefact retrieves metadata for the specified artefact.
	//
	// Takes artefactID (string) which identifies the artefact to retrieve.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
	// Returns error when the artefact cannot be found or retrieval fails.
	GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)

	// GetMultipleArtefacts retrieves metadata for multiple artefacts by their IDs.
	//
	// Takes artefactIDs ([]string) which contains the IDs of artefacts to fetch.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the metadata for each
	// found artefact.
	// Returns error when the retrieval fails.
	GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error)

	// ListAllArtefactIDs retrieves all artefact identifiers from the store.
	//
	// Returns []string which contains all known artefact identifiers.
	// Returns error when the retrieval fails.
	ListAllArtefactIDs(ctx context.Context) ([]string, error)

	// SearchArtefacts finds artefacts that match the given query.
	//
	// Takes query (SearchQuery) which specifies the search criteria.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
	// Returns error when the search fails.
	SearchArtefacts(ctx context.Context, query SearchQuery) ([]*registry_dto.ArtefactMeta, error)

	// SearchArtefactsByTagValues retrieves artefacts that match any of the given
	// tag values for a specific tag key.
	//
	// Takes tagKey (string) which identifies the tag to search by.
	// Takes tagValues ([]string) which lists the values to match against.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
	// Returns error when the search operation fails.
	SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error)

	// FindArtefactByVariantStorageKey retrieves artefact metadata by its variant
	// storage key.
	//
	// Takes storageKey (string) which identifies the variant in storage.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
	// Returns error when the artefact cannot be found or retrieval fails.
	FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error)

	// PopGCHints retrieves garbage collection hints from the registry.
	//
	// Takes limit (int) which specifies the maximum number of hints to return.
	//
	// Returns []registry_dto.GCHint which contains the retrieved hints.
	// Returns error when the operation fails.
	PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error)

	// AtomicUpdate applies a set of actions as a single atomic operation.
	//
	// Takes actions ([]registry_dto.AtomicAction) which specifies the operations
	// to perform atomically.
	//
	// Returns error when any action fails, rolling back all changes.
	AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error

	// IncrementBlobRefCount atomically increments the reference count for a blob.
	// If the blob does not exist, it creates it with a reference count of one.
	//
	// Takes blob (BlobReference) which identifies the blob to increment.
	//
	// Returns newRefCount (int) which is the reference count after incrementing.
	// Returns error when the operation fails.
	IncrementBlobRefCount(ctx context.Context, blob BlobReference) (newRefCount int, err error)

	// DecrementBlobRefCount atomically decrements the reference count for a blob.
	//
	// Takes storageKey (string) which identifies the blob.
	//
	// Returns newRefCount (int) which is the updated reference count.
	// Returns shouldDelete (bool) which is true when the blob should be deleted.
	// Returns error when the blob does not exist.
	DecrementBlobRefCount(ctx context.Context, storageKey string) (newRefCount int, shouldDelete bool, err error)

	// GetBlobRefCount returns the current reference count for a blob.
	// Returns 0 if the blob doesn't exist (not an error).
	GetBlobRefCount(ctx context.Context, storageKey string) (int, error)

	// RunAtomic executes fn within a transaction.
	//
	// The provided MetadataStore is scoped to the
	// transaction; all reads and writes through it are
	// atomic. If fn returns an error (or panics), all
	// mutations are rolled back.
	//
	// Takes fn which receives a transactional
	// MetadataStore. The caller MUST use this
	// transactional store for all operations that should
	// be atomic.
	//
	// Returns error when fn returns an error or the
	// transaction fails to commit.
	RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore MetadataStore) error) error

	// Close releases resources held by this store.
	//
	// Returns error when the close operation fails.
	Close() error
}

// MetadataCache provides an in-memory cache for artefact metadata.
// It gives fast access to metadata that is used often.
type MetadataCache interface {
	// Get retrieves the metadata for an artefact by its identifier.
	//
	// Takes artefactID (string) which is the unique identifier of the artefact.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
	// Returns error when the artefact cannot be found or retrieval fails.
	Get(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)

	// GetMultiple retrieves metadata for multiple artefacts by their IDs.
	//
	// Takes artefactIDs ([]string) which lists the artefact IDs to look up.
	//
	// Returns hits ([]*registry_dto.ArtefactMeta) which contains the found
	// artefact metadata.
	// Returns misses ([]string) which lists the IDs that were not found.
	GetMultiple(ctx context.Context, artefactIDs []string) (hits []*registry_dto.ArtefactMeta, misses []string)

	// Set stores the given artefact metadata.
	//
	// Takes artefact (*registry_dto.ArtefactMeta) which contains the metadata to
	// store.
	Set(ctx context.Context, artefact *registry_dto.ArtefactMeta)

	// SetMultiple stores several artefact metadata entries at once.
	//
	// Takes artefacts ([]*registry_dto.ArtefactMeta) which holds the metadata
	// entries to store.
	SetMultiple(ctx context.Context, artefacts []*registry_dto.ArtefactMeta)

	// Delete removes the artefact with the given identifier.
	//
	// Takes artefactID (string) which identifies the artefact to remove.
	Delete(ctx context.Context, artefactID string)

	// Close releases resources held by this object.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	//
	// Returns error when the close operation fails.
	Close(ctx context.Context) error
}

// BlobStore defines the interface for storing and retrieving binary blob data.
// Implementations handle storage of artefact variant content on disk, S3, or
// other backends.
type BlobStore interface {
	// Put stores data from the reader under the given key.
	//
	// Takes key (string) which identifies the stored data.
	// Takes data (io.Reader) which provides the content to store.
	//
	// Returns error when the storage operation fails.
	Put(ctx context.Context, key string, data io.Reader) error

	// Get retrieves the value for the given key.
	//
	// Takes key (string) which identifies the value to retrieve.
	//
	// Returns io.ReadCloser which provides access to the stored data.
	// Returns error when the key does not exist or retrieval fails.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// RangeGet retrieves a portion of data for the given key.
	//
	// Takes key (string) which identifies the data to retrieve.
	// Takes offset (int64) which specifies the starting byte position.
	// Takes length (int64) which specifies the number of bytes to read.
	//
	// Returns io.ReadCloser which provides access to the requested byte range.
	// Returns error when the key does not exist or the range is invalid.
	RangeGet(ctx context.Context, key string, offset int64, length int64) (io.ReadCloser, error)

	// Delete removes the value for the given key.
	//
	// Takes key (string) which identifies the value to remove.
	//
	// Returns error when the deletion fails.
	Delete(ctx context.Context, key string) error

	// Rename moves a value from a temporary key to its final key.
	//
	// Takes tempKey (string) which is the temporary key holding the value.
	// Takes key (string) which is the final destination key.
	//
	// Returns error when the rename operation fails.
	Rename(ctx context.Context, tempKey string, key string) error

	// Exists checks whether a value with the given key is present.
	//
	// Takes key (string) which identifies the value to look for.
	//
	// Returns bool which is true if the key exists, false otherwise.
	// Returns error when the check fails.
	Exists(ctx context.Context, key string) (bool, error)

	// ListKeys returns all storage keys present in this blob store.
	// Used by garbage collection to detect orphaned blobs that are no longer
	// referenced by any artefact.
	//
	// Returns []string which contains all keys in the store.
	// Returns error when the listing operation fails or is not supported.
	ListKeys(ctx context.Context) ([]string, error)
}

// RegistryService defines the interface for the artefact registry service.
// It provides operations for managing artefacts, variants, and blob storage.
type RegistryService interface {
	// UpsertArtefact creates or updates an artefact with the given source data
	// and profiles.
	//
	// Takes artefactID (string) which identifies the artefact to create or update.
	// Takes sourcePath (string) which specifies the path of the source file.
	// Takes sourceData (io.Reader) which provides the content to store.
	// Takes storageBackendID (string) which identifies the storage backend to use.
	// Takes desiredProfiles ([]registry_dto.NamedProfile) which specifies the
	// profiles to apply.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
	// Returns error when the operation fails.
	UpsertArtefact(
		ctx context.Context,
		artefactID string,
		sourcePath string,
		sourceData io.Reader,
		storageBackendID string,
		desiredProfiles []registry_dto.NamedProfile,
	) (*registry_dto.ArtefactMeta, error)

	// AddVariant adds a new variant to an existing artefact.
	//
	// Takes artefactID (string) which identifies the target artefact.
	// Takes newVariant (*registry_dto.Variant) which contains the variant data.
	//
	// Returns *registry_dto.ArtefactMeta which is the updated artefact metadata.
	// Returns error when the variant cannot be added.
	AddVariant(ctx context.Context, artefactID string, newVariant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error)

	// DeleteArtefact removes an artefact by its identifier.
	//
	// Takes artefactID (string) which identifies the artefact to remove.
	//
	// Returns error when the deletion fails.
	DeleteArtefact(ctx context.Context, artefactID string) error

	// GetArtefact retrieves metadata for an artefact by its identifier.
	//
	// Takes artefactID (string) which identifies the artefact to retrieve.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
	// Returns error when the artefact cannot be found or retrieval fails.
	GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)

	// GetMultipleArtefacts retrieves metadata for multiple artefacts by their IDs.
	//
	// Takes artefactIDs ([]string) which specifies the artefact identifiers to
	// look up.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the metadata for each
	// found artefact.
	// Returns error when the lookup fails.
	GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error)

	// ListAllArtefactIDs retrieves all artefact identifiers from the store.
	//
	// Returns []string which contains all known artefact IDs.
	// Returns error when the retrieval fails.
	ListAllArtefactIDs(ctx context.Context) ([]string, error)

	// SearchArtefacts finds artefacts matching the given query.
	//
	// Takes query (SearchQuery) which specifies the search criteria.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
	// Returns error when the search fails.
	SearchArtefacts(ctx context.Context, query SearchQuery) ([]*registry_dto.ArtefactMeta, error)

	// SearchArtefactsByTagValues retrieves artefacts that match any of the given
	// tag values for a specific tag key.
	//
	// Takes tagKey (string) which identifies the tag to search by.
	// Takes tagValues ([]string) which lists the values to match against.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
	// Returns error when the search fails.
	SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error)

	// FindArtefactByVariantStorageKey retrieves artefact metadata by its variant
	// storage key.
	//
	// Takes storageKey (string) which identifies the variant in storage.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
	// Returns error when the artefact cannot be found or retrieval fails.
	FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error)

	// GetVariantData retrieves the data for the given variant.
	//
	// Takes variant (*registry_dto.Variant) which specifies the variant to fetch.
	//
	// Returns io.ReadCloser which provides access to the variant data.
	// Returns error when the variant data cannot be retrieved.
	GetVariantData(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error)

	// GetVariantChunk retrieves a chunk of data for the given variant.
	//
	// Takes variant (*registry_dto.Variant) which identifies the variant to read.
	// Takes chunkID (string) which specifies which chunk to retrieve.
	//
	// Returns io.ReadCloser which provides access to the chunk data.
	// Returns error when the chunk cannot be retrieved.
	GetVariantChunk(ctx context.Context, variant *registry_dto.Variant, chunkID string) (io.ReadCloser, error)

	// GetVariantDataRange retrieves a portion of the variant's data as a stream.
	//
	// Takes variant (*registry_dto.Variant) which identifies the variant to read.
	// Takes offset (int64) which specifies the starting byte position.
	// Takes length (int64) which specifies the number of bytes to read.
	//
	// Returns io.ReadCloser which provides the requested data range.
	// Returns error when the variant cannot be found or the range is invalid.
	GetVariantDataRange(ctx context.Context, variant *registry_dto.Variant, offset, length int64) (io.ReadCloser, error)

	// GetBlobStore retrieves the blob store for the given backend.
	//
	// Takes backendID (string) which identifies the storage backend.
	//
	// Returns BlobStore which provides access to blob storage operations.
	// Returns error when the backend cannot be found or accessed.
	GetBlobStore(backendID string) (BlobStore, error)

	// PopGCHints retrieves garbage collection hints from the registry.
	//
	// Takes limit (int) which specifies the maximum number of hints to return.
	//
	// Returns []registry_dto.GCHint which contains the retrieved hints.
	// Returns error when the hints cannot be retrieved.
	PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error)

	// ListBlobStoreIDs returns the identifiers of all registered blob storage
	// backends. Used by garbage collection to enumerate backends for orphan
	// scanning.
	//
	// Returns []string which contains all backend IDs, sorted alphabetically.
	ListBlobStoreIDs() []string

	// ArtefactEventsPublished returns the number of artefact events published.
	// Used for pipeline flush detection to check that all events are processed
	// before the system is considered idle.
	//
	// Returns int64 which is the total count of published artefact events.
	ArtefactEventsPublished() int64
}
