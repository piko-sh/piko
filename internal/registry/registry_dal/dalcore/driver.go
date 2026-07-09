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

package dalcore

import (
	"context"
	"database/sql"
)

// Driver is the per-dialect seam between Core and a generated query package.
//
// Its implementations live in each querier_<dialect> package and perform the trivial
// translation between the dialect-neutral flat structs below and the generated database
// types. Read methods return the raw sql.ErrNoRows so Core can map it to the appropriate
// domain sentinel.
type Driver interface {
	// WithTx returns a driver whose queries run inside tx.
	WithTx(tx *sql.Tx) Driver

	// GetArtefactData returns the FlatBuffer payload for a single artefact.
	GetArtefactData(ctx context.Context, artefactID string) ([]byte, error)

	// GetMultipleArtefactsData returns the FlatBuffer payloads for the given artefact IDs.
	GetMultipleArtefactsData(ctx context.Context, artefactIDs []string) ([][]byte, error)

	// ListAllArtefactsData returns the FlatBuffer payloads for every artefact.
	ListAllArtefactsData(ctx context.Context) ([][]byte, error)

	// ListRecentArtefactsData returns the FlatBuffer payloads for the most recently updated
	// artefacts, most recent first.
	ListRecentArtefactsData(ctx context.Context, limit int) ([][]byte, error)

	// ListAllArtefactIDs returns every artefact ID in the store.
	ListAllArtefactIDs(ctx context.Context) ([]string, error)

	// FindArtefactIDsByTag returns the IDs of artefacts carrying the given tag key and
	// value.
	FindArtefactIDsByTag(ctx context.Context, tagKey, tagValue string) ([]string, error)

	// FindArtefactIDsByTagValues returns the IDs of artefacts whose tag key matches any of
	// the given values.
	FindArtefactIDsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]string, error)

	// FindArtefactIDByVariantStorageKey returns the artefact ID owning the variant stored at
	// storageKey.
	FindArtefactIDByVariantStorageKey(ctx context.Context, storageKey string) (string, error)

	// ListVariantStatusCounts returns variant counts grouped by status.
	ListVariantStatusCounts(ctx context.Context) ([]VariantStatusCount, error)

	// IncrementBlobRefCount increments (or creates) the reference count for a blob and
	// returns the new count.
	IncrementBlobRefCount(ctx context.Context, params IncrementBlobRefCountParams) (int, error)

	// DecrementBlobRefCount decrements the reference count for a blob and returns the new
	// count.
	DecrementBlobRefCount(ctx context.Context, storageKey string, lastReferencedAt int64) (int, error)

	// DeleteBlobReferenceIfZero deletes the blob reference record when its count has reached
	// zero.
	DeleteBlobReferenceIfZero(ctx context.Context, storageKey string) error

	// GetBlobRefCount returns the current reference count for a blob.
	GetBlobRefCount(ctx context.Context, storageKey string) (int, error)

	// PopGCHints returns up to limit garbage-collection hints.
	PopGCHints(ctx context.Context, limit int) ([]GCHintRow, error)

	// DeleteGCHints deletes the garbage-collection hints with the given row IDs.
	DeleteGCHints(ctx context.Context, ids []int64) error

	// AddGCHint records a garbage-collection hint for a storage key.
	AddGCHint(ctx context.Context, backendID, storageKey string, createdAt int64) error

	// UpsertArtefact inserts or updates an artefact record.
	UpsertArtefact(ctx context.Context, params UpsertArtefactParams) error

	// DeleteArtefact deletes an artefact by ID.
	DeleteArtefact(ctx context.Context, artefactID string) error

	// DeleteVariantTagsForArtefact removes all variant tags belonging to an artefact.
	DeleteVariantTagsForArtefact(ctx context.Context, artefactID string) error

	// DeleteChunksForVariant removes all chunks belonging to a variant.
	DeleteChunksForVariant(ctx context.Context, artefactID, variantID string) error

	// DeleteVariantsForArtefact removes all variants belonging to an artefact.
	DeleteVariantsForArtefact(ctx context.Context, artefactID string) error

	// DeleteDesiredProfilesForArtefact removes all desired profiles belonging to an
	// artefact.
	DeleteDesiredProfilesForArtefact(ctx context.Context, artefactID string) error

	// InsertVariant stores a variant record.
	InsertVariant(ctx context.Context, params InsertVariantParams) error

	// InsertVariantTag stores a single metadata tag for a variant.
	InsertVariantTag(ctx context.Context, artefactID, variantID, tagKey, tagValue string) error

	// InsertVariantChunk stores a single chunk record for a variant.
	InsertVariantChunk(ctx context.Context, params InsertVariantChunkParams) error

	// InsertDesiredProfile stores a single desired-profile record.
	InsertDesiredProfile(ctx context.Context, params InsertDesiredProfileParams) error
}

// VariantStatusCount pairs a variant status with the number of variants in that status.
type VariantStatusCount struct {
	// Status is the variant status.
	Status string

	// Count is the number of variants in the status.
	Count int64
}

// GCHintRow is a garbage-collection hint row.
type GCHintRow struct {
	// BackendID identifies the storage backend the key belongs to.
	BackendID string

	// StorageKey is the storage key marked for cleanup.
	StorageKey string

	// ID is the hint row identifier used for deletion.
	ID int64
}

// UpsertArtefactParams carries the fields required to insert or update an artefact.
type UpsertArtefactParams struct {
	// ID is the artefact identifier.
	ID string

	// SourcePath is the artefact's source path.
	SourcePath string

	// DataFbs is the FlatBuffer-encoded artefact payload.
	DataFbs []byte

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64
}

// IncrementBlobRefCountParams carries the fields required to increment (or create) a blob
// reference.
type IncrementBlobRefCountParams struct {
	// StorageKey identifies the blob in storage.
	StorageKey string

	// StorageBackendID identifies the storage backend.
	StorageBackendID string

	// ContentHash is the blob's content hash.
	ContentHash string

	// MimeType is the blob's MIME type.
	MimeType string

	// SizeBytes is the blob size in bytes.
	SizeBytes int64

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// LastReferencedAt is the last-reference time in Unix seconds.
	LastReferencedAt int64
}

// InsertVariantParams carries the fields required to insert a variant record.
type InsertVariantParams struct {
	// ArtefactID identifies the parent artefact.
	ArtefactID string

	// VariantID identifies the variant.
	VariantID string

	// StorageKey is the variant's storage key.
	StorageKey string

	// StorageBackendID identifies the storage backend.
	StorageBackendID string

	// MimeType is the variant's MIME type.
	MimeType string

	// Status is the variant status.
	Status string

	// SizeBytes is the variant size in bytes.
	SizeBytes int64

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64
}

// InsertVariantChunkParams carries the fields required to insert a variant chunk record.
type InsertVariantChunkParams struct {
	// DurationSeconds is the chunk's duration in seconds, when applicable.
	DurationSeconds *float64

	// ArtefactID identifies the parent artefact.
	ArtefactID string

	// VariantID identifies the parent variant.
	VariantID string

	// ChunkID identifies the chunk.
	ChunkID string

	// StorageKey is the chunk's storage key.
	StorageKey string

	// StorageBackendID identifies the storage backend.
	StorageBackendID string

	// ContentHash is the chunk's content hash.
	ContentHash string

	// MimeType is the chunk's MIME type.
	MimeType string

	// SizeBytes is the chunk size in bytes.
	SizeBytes int64

	// SequenceNumber is the chunk's ordinal position.
	SequenceNumber int64

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64
}

// InsertDesiredProfileParams carries the fields required to insert a desired-profile
// record.
type InsertDesiredProfileParams struct {
	// ArtefactID identifies the parent artefact.
	ArtefactID string

	// Name is the desired profile name.
	Name string

	// CapabilityName is the capability the profile targets.
	CapabilityName string

	// Priority is the profile priority.
	Priority string

	// ParamsJSON is the JSON-encoded profile parameters.
	ParamsJSON string

	// TagsJSON is the JSON-encoded resulting tags.
	TagsJSON string

	// DependsOnJSON is the JSON-encoded dependency list.
	DependsOnJSON string
}
