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

package registry_dal

import (
	"context"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

// MetadataDAL defines the interface for persisting and querying artefact
// metadata. It implements registry_domain.MetadataStore but is defined in
// the DAL layer to separate domain ports from data access implementation.
type MetadataDAL interface {
	// GetArtefact retrieves a single artefact by ID with all its variants and
	// profiles.
	//
	// Takes artefactID (string) which identifies the artefact to retrieve.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact with all its
	// variants and profiles.
	// Returns error when the artefact cannot be found or retrieval fails.
	GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)

	// GetMultipleArtefacts fetches several artefacts by their IDs.
	//
	// Takes artefactIDs ([]string) which lists the IDs of artefacts to fetch.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the retrieved artefacts.
	// Returns error when any artefact cannot be found or fetched.
	GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error)

	// ListAllArtefactIDs returns all artefact IDs in the store.
	//
	// Returns []string which contains all artefact IDs.
	// Returns error when the IDs cannot be retrieved.
	ListAllArtefactIDs(ctx context.Context) ([]string, error)

	// SearchArtefacts finds artefacts that match the given tag query.
	//
	// Takes query (registry_domain.SearchQuery) which specifies the search terms.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
	// Returns error when the search fails.
	SearchArtefacts(ctx context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error)

	// SearchArtefactsByTagValues searches for artefacts that have a specific tag
	// key with any of the given values.
	//
	// Takes tagKey (string) which specifies the tag key to match.
	// Takes tagValues ([]string) which lists the acceptable values for the tag.
	//
	// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
	// Returns error when the search fails.
	SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error)

	// FindArtefactByVariantStorageKey finds an artefact by the storage key of one
	// of its variants.
	//
	// Takes storageKey (string) which identifies the variant to search for.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
	// Returns error when the artefact cannot be found or retrieval fails.
	FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error)

	// PopGCHints retrieves and removes garbage collection hints from the store.
	//
	// Takes limit (int) which sets the maximum number of hints to return.
	//
	// Returns []registry_dto.GCHint which contains the removed hints.
	// Returns error when the operation fails.
	PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error)

	// AtomicUpdate runs a batch of actions within a single transaction.
	//
	// Takes actions ([]registry_dto.AtomicAction) which lists the operations to
	// perform.
	//
	// Returns error when any action in the batch fails.
	AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error

	// IncrementBlobRefCount atomically increments the reference count for a blob.
	// If the blob does not exist, it creates it with a reference count of one.
	//
	// Takes blob (registry_domain.BlobReference) which identifies the blob.
	//
	// Returns newRefCount (int) which is the reference count after the increment.
	// Returns error when the operation fails.
	IncrementBlobRefCount(ctx context.Context, blob registry_domain.BlobReference) (newRefCount int, err error)

	// DecrementBlobRefCount atomically decrements the reference count for a blob.
	//
	// Takes storageKey (string) which identifies the blob.
	//
	// Returns newRefCount (int) which is the updated reference count.
	// Returns shouldDelete (bool) which is true when the count reaches zero.
	// Returns error when the blob does not exist.
	DecrementBlobRefCount(ctx context.Context, storageKey string) (newRefCount int, shouldDelete bool, err error)

	// GetBlobRefCount returns the current reference count for a blob.
	// Returns 0 if the blob doesn't exist (not an error).
	GetBlobRefCount(ctx context.Context, storageKey string) (int, error)
}

// RegistryDAL provides the complete data access layer for the artefact registry.
// It combines MetadataDAL operations with health checks and resource management.
type RegistryDAL interface {
	MetadataDAL

	// HealthCheck checks that the database connection is working.
	//
	// Returns error when the connection is not healthy.
	HealthCheck(ctx context.Context) error

	// Close releases any resources held by the DAL.
	//
	// Returns error when the resources cannot be released.
	Close() error
}

// RegistryDALWithTx provides registry data access with transaction support.
type RegistryDALWithTx interface {
	RegistryDAL

	// RunAtomic executes fn within a transaction.
	//
	// The provided MetadataStore is scoped to the
	// transaction; all reads and writes through it are
	// atomic. If fn returns an error (or panics), all
	// mutations are rolled back.
	RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error
}
