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
	//
	// Takes tx (*sql.Tx) which is the open transaction the returned driver runs in.
	//
	// Returns Driver which is a driver bound to that transaction.
	WithTx(tx *sql.Tx) Driver

	// GetArtefactLayers returns every tagged layer of a single artefact, ordered by release
	// id.
	//
	// Takes artefactID (string) which identifies the artefact.
	//
	// Returns []ArtefactLayerData which are the artefact's layers.
	// Returns error when the query fails.
	GetArtefactLayers(ctx context.Context, artefactID string) ([]ArtefactLayerData, error)

	// GetArtefactLayersForUpdate returns every tagged layer under a row-level lock held for
	// the enclosing transaction (Postgres FOR UPDATE; SQLite mirrors GetArtefactLayers
	// because its single-writer transaction already serialises writes).
	//
	// Takes artefactID (string) which identifies the artefact.
	//
	// Returns []ArtefactLayerData which are the locked artefact layers.
	// Returns error when the query fails.
	GetArtefactLayersForUpdate(ctx context.Context, artefactID string) ([]ArtefactLayerData, error)

	// GetMultipleArtefactLayers returns every layer of the given artefacts, tagged with the
	// artefact ID so the core can group and merge them.
	//
	// Takes artefactIDs ([]string) which are the artefacts to load.
	//
	// Returns []ArtefactLayerData which are the layers of the requested artefacts.
	// Returns error when the query fails.
	GetMultipleArtefactLayers(ctx context.Context, artefactIDs []string) ([]ArtefactLayerData, error)

	// ListAllArtefactsData returns the FlatBuffer payloads for every artefact.
	//
	// Returns [][]byte which are the FlatBuffer payloads.
	// Returns error when the query fails.
	ListAllArtefactsData(ctx context.Context) ([][]byte, error)

	// ListRecentArtefactsData returns the FlatBuffer payloads for the most recently updated
	// artefacts, most recent first.
	//
	// Takes limit (int) which caps how many payloads are returned.
	//
	// Returns [][]byte which are the FlatBuffer payloads.
	// Returns error when the query fails.
	ListRecentArtefactsData(ctx context.Context, limit int) ([][]byte, error)

	// ListAllArtefactIDs returns every artefact ID in the store.
	//
	// Returns []string which are the artefact IDs.
	// Returns error when the query fails.
	ListAllArtefactIDs(ctx context.Context) ([]string, error)

	// FindArtefactIDsByTag returns the IDs of artefacts carrying the given tag key and
	// value.
	//
	// Takes tagKey (string) which is the tag name to match.
	// Takes tagValue (string) which is the tag value to match.
	//
	// Returns []string which are the matching artefact IDs.
	// Returns error when the query fails.
	FindArtefactIDsByTag(ctx context.Context, tagKey, tagValue string) ([]string, error)

	// FindArtefactIDsByTagValues returns the IDs of artefacts whose tag key matches any of
	// the given values.
	//
	// Takes tagKey (string) which is the tag name to match.
	// Takes tagValues ([]string) which are the accepted tag values.
	//
	// Returns []string which are the matching artefact IDs.
	// Returns error when the query fails.
	FindArtefactIDsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]string, error)

	// FindArtefactIDByVariantStorageKey returns the artefact ID owning the variant stored at
	// storageKey, and a bool that is false when no variant references the key.
	//
	// Takes storageKey (string) which is the variant's storage key.
	//
	// Returns string which is the owning artefact ID.
	// Returns bool which reports whether a variant references the key.
	// Returns error when the query fails.
	FindArtefactIDByVariantStorageKey(ctx context.Context, storageKey string) (string, bool, error)

	// ListVariantStatusCounts returns variant counts grouped by status.
	//
	// Returns []VariantStatusCount which pair each status with its variant count.
	// Returns error when the query fails.
	ListVariantStatusCounts(ctx context.Context) ([]VariantStatusCount, error)

	// IncrementBlobRefCount increments (or creates) the reference count for a blob and
	// returns the new count.
	//
	// Takes params (IncrementBlobRefCountParams) which describe the blob and its reference
	// times.
	//
	// Returns int which is the new reference count.
	// Returns error when the write fails.
	IncrementBlobRefCount(ctx context.Context, params IncrementBlobRefCountParams) (int, error)

	// DecrementBlobRefCount decrements the reference count for a blob and returns the new
	// count.
	//
	// The bool is false when no row was decremented, because the blob is absent or its count
	// is already zero (the query guards on ref_count > 0).
	//
	// Takes storageKey (string) which identifies the blob in storage.
	// Takes lastReferencedAt (int64) which is the last-reference time in Unix seconds.
	//
	// Returns int which is the new reference count.
	// Returns bool which reports whether a row was decremented.
	// Returns error when the write fails.
	DecrementBlobRefCount(ctx context.Context, storageKey string, lastReferencedAt int64) (int, bool, error)

	// DeleteBlobReferenceIfZero deletes the blob reference record when its count has reached
	// zero.
	//
	// Takes storageKey (string) which identifies the blob in storage.
	//
	// Returns error when the delete fails.
	DeleteBlobReferenceIfZero(ctx context.Context, storageKey string) error

	// GetBlobRefCount returns the current reference count for a blob, and a bool that is
	// false when no reference record exists for the key.
	//
	// Takes storageKey (string) which identifies the blob in storage.
	//
	// Returns int which is the current reference count.
	// Returns bool which reports whether a reference record exists.
	// Returns error when the query fails.
	GetBlobRefCount(ctx context.Context, storageKey string) (int, bool, error)

	// PopGCHints returns up to limit garbage-collection hints.
	//
	// Takes limit (int) which caps how many hints are returned.
	//
	// Returns []GCHintRow which are the hints taken from the queue.
	// Returns error when the query fails.
	PopGCHints(ctx context.Context, limit int) ([]GCHintRow, error)

	// DeleteGCHints deletes the garbage-collection hints with the given row IDs.
	//
	// Takes ids ([]int64) which are the hint row identifiers to delete.
	//
	// Returns error when the delete fails.
	DeleteGCHints(ctx context.Context, ids []int64) error

	// AddGCHint records a garbage-collection hint for a storage key.
	//
	// Takes backendID (string) which identifies the storage backend.
	// Takes storageKey (string) which is the key marked for cleanup.
	// Takes createdAt (int64) which is the hint time in Unix seconds.
	//
	// Returns error when the insert fails.
	AddGCHint(ctx context.Context, backendID, storageKey string, createdAt int64) error

	// UpsertArtefact inserts or updates one layer of an artefact, keyed by (id, release_id).
	//
	// Takes params (UpsertArtefactParams) which carry the layer's identity and payload.
	//
	// Returns error when the write fails.
	UpsertArtefact(ctx context.Context, params UpsertArtefactParams) error

	// InsertArtefactLayerIfAbsent inserts one artefact layer only when its (id, release_id)
	// is not already present.
	//
	// It returns true when a row was inserted, so publish increments blob ref counts exactly
	// once per newly published layer and is idempotent across nodes.
	//
	// Takes params (UpsertArtefactParams) which carry the layer's identity and payload.
	//
	// Returns bool which reports whether a row was inserted.
	// Returns error when the insert fails.
	InsertArtefactLayerIfAbsent(ctx context.Context, params UpsertArtefactParams) (bool, error)

	// DeleteArtefact deletes every layer of an artefact by ID.
	//
	// Takes artefactID (string) which identifies the artefact.
	//
	// Returns error when the delete fails.
	DeleteArtefact(ctx context.Context, artefactID string) error

	// DeleteArtefactLayer deletes one artefact layer, keyed by (id, release_id).
	//
	// Takes artefactID (string) which identifies the artefact.
	// Takes releaseID (string) which identifies the layer's release.
	//
	// Returns error when the delete fails.
	DeleteArtefactLayer(ctx context.Context, artefactID, releaseID string) error

	// DeleteArtefactLayersForRelease deletes every artefact layer of a release, retiring it.
	//
	// Takes releaseID (string) which identifies the release.
	//
	// Returns error when the delete fails.
	DeleteArtefactLayersForRelease(ctx context.Context, releaseID string) error

	// ReclaimArtefactLayersForRelease deletes every artefact layer of a release and returns
	// the deleted layers in one statement, so exactly one of two racing reapers observes the
	// rows and decrements blob references.
	//
	// Takes releaseID (string) which identifies the release.
	//
	// Returns []ArtefactLayerData which are the layers this caller deleted.
	// Returns error when the delete fails.
	ReclaimArtefactLayersForRelease(ctx context.Context, releaseID string) ([]ArtefactLayerData, error)

	// DeleteVariantTagsForArtefact removes an artefact layer's variant tags.
	//
	// Takes artefactID (string) which identifies the artefact.
	// Takes releaseID (string) which identifies the layer's release.
	//
	// Returns error when the delete fails.
	DeleteVariantTagsForArtefact(ctx context.Context, artefactID, releaseID string) error

	// DeleteChunksForArtefact removes an artefact layer's chunks, across all its variants.
	//
	// It is used when clearing a layer's projection rows before a re-import, because the
	// incoming variant list may not name a variant whose chunks are still stored.
	//
	// Takes artefactID (string) which identifies the artefact.
	// Takes releaseID (string) which identifies the layer's release.
	//
	// Returns error when the delete fails.
	DeleteChunksForArtefact(ctx context.Context, artefactID, releaseID string) error

	// DeleteVariantsForArtefact removes an artefact layer's variants.
	//
	// Takes artefactID (string) which identifies the artefact.
	// Takes releaseID (string) which identifies the layer's release.
	//
	// Returns error when the delete fails.
	DeleteVariantsForArtefact(ctx context.Context, artefactID, releaseID string) error

	// DeleteDesiredProfilesForArtefact removes an artefact layer's desired profiles.
	//
	// Takes artefactID (string) which identifies the artefact.
	// Takes releaseID (string) which identifies the layer's release.
	//
	// Returns error when the delete fails.
	DeleteDesiredProfilesForArtefact(ctx context.Context, artefactID, releaseID string) error

	// InsertVariant stores a variant record.
	//
	// Takes params (InsertVariantParams) which carry the variant's identity and storage
	// details.
	//
	// Returns error when the insert fails.
	InsertVariant(ctx context.Context, params InsertVariantParams) error

	// InsertVariantTag stores a single metadata tag for a variant in a layer.
	//
	// Takes artefactID (string) which identifies the artefact.
	// Takes releaseID (string) which identifies the layer's release.
	// Takes variantID (string) which identifies the variant.
	// Takes tagKey (string) which is the tag name.
	// Takes tagValue (string) which is the tag value.
	//
	// Returns error when the insert fails.
	InsertVariantTag(ctx context.Context, artefactID, releaseID, variantID, tagKey, tagValue string) error

	// InsertVariantChunk stores a single chunk record for a variant.
	//
	// Takes params (InsertVariantChunkParams) which carry the chunk's identity and storage
	// details.
	//
	// Returns error when the insert fails.
	InsertVariantChunk(ctx context.Context, params InsertVariantChunkParams) error

	// InsertDesiredProfile stores a single desired-profile record.
	//
	// Takes params (InsertDesiredProfileParams) which carry the profile's name, capability
	// and parameters.
	//
	// Returns error when the insert fails.
	InsertDesiredProfile(ctx context.Context, params InsertDesiredProfileParams) error

	// ClaimRelease attempts to claim publishing rights for a release.
	//
	// It returns true when this caller won the claim, and false when another caller already
	// holds it.
	//
	// Takes params (ClaimReleaseParams) which carry the release, its digest and the claim
	// times.
	//
	// Returns bool which reports whether this caller won the claim.
	// Returns error when the write fails.
	ClaimRelease(ctx context.Context, params ClaimReleaseParams) (bool, error)

	// GetRelease returns a release lease, or false when the release is unknown.
	//
	// Takes releaseID (string) which identifies the release.
	//
	// Returns ReleaseLease which is the stored lease record.
	// Returns bool which reports whether the release is known.
	// Returns error when the query fails.
	GetRelease(ctx context.Context, releaseID string) (ReleaseLease, bool, error)

	// MarkReleasePublished flips a release lease to published and stamps its timestamps.
	//
	// Takes releaseID (string) which identifies the release.
	// Takes publishedAt (int64) which is the publish time in Unix seconds.
	// Takes heartbeatAt (int64) which is the heartbeat time in Unix seconds.
	//
	// Returns error when the write fails.
	MarkReleasePublished(ctx context.Context, releaseID string, publishedAt, heartbeatAt int64) error

	// HeartbeatRelease advances a release's heartbeat when the new value is more recent.
	//
	// Takes releaseID (string) which identifies the release.
	// Takes heartbeatAt (int64) which is the new heartbeat time in Unix seconds.
	//
	// Returns error when the write fails.
	HeartbeatRelease(ctx context.Context, releaseID string, heartbeatAt int64) error

	// ListExpiredReleases returns published releases whose heartbeat predates the cutoff,
	// excluding the caller's own release.
	//
	// Takes cutoff (int64) which is the heartbeat cutoff in Unix seconds.
	// Takes ownRelease (string) which is the caller's own release, left out of the result.
	//
	// Returns []string which are the expired release IDs.
	// Returns error when the query fails.
	ListExpiredReleases(ctx context.Context, cutoff int64, ownRelease string) ([]string, error)

	// DeleteReleaseLease removes a release lease row.
	//
	// Takes releaseID (string) which identifies the release.
	//
	// Returns error when the delete fails.
	DeleteReleaseLease(ctx context.Context, releaseID string) error

	// DeleteStalePublishingLease removes a publishing lease whose heartbeat predates
	// staleBefore, so a publish that died mid-flight can be re-claimed by another node.
	//
	// Takes releaseID (string) which identifies the release.
	// Takes staleBefore (int64) which is the heartbeat cutoff in Unix seconds.
	//
	// Returns error when the delete fails.
	DeleteStalePublishingLease(ctx context.Context, releaseID string, staleBefore int64) error
}

// ArtefactLayerData tags one artefact layer's payload with its artefact ID and its
// release.
type ArtefactLayerData struct {
	// ID is the artefact identifier.
	ID string

	// ReleaseID is the layer's release; empty for the runtime layer.
	ReleaseID string

	// Data is the layer's FlatBuffer payload.
	Data []byte
}

// ClaimReleaseParams carries the fields required to claim a release for publishing.
type ClaimReleaseParams struct {
	// ReleaseID identifies the release being claimed.
	ReleaseID string

	// PublishDigest is the digest of the payload this release will publish.
	PublishDigest string

	// FirstSeenAt is the claim time in Unix seconds.
	FirstSeenAt int64

	// HeartbeatAt is the initial heartbeat time in Unix seconds.
	HeartbeatAt int64
}

// ReleaseLease is one published-release record.
type ReleaseLease struct {
	// ReleaseID identifies the release.
	ReleaseID string

	// PublishDigest is the digest of the payload the release published.
	PublishDigest string

	// State is 'publishing' or 'published'.
	State string

	// FirstSeenAt is when the release was first claimed, in Unix seconds.
	FirstSeenAt int64

	// PublishedAt is when the release finished publishing, in Unix seconds.
	PublishedAt int64

	// HeartbeatAt is the release's most recent heartbeat, in Unix seconds.
	HeartbeatAt int64

	// RetiredAt is when the release was retired, in Unix seconds, or zero.
	RetiredAt int64
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

// UpsertArtefactParams carries the fields required to insert or update one artefact
// layer.
type UpsertArtefactParams struct {
	// ID is the artefact identifier.
	ID string

	// ReleaseID is the layer this payload belongs to (empty for the default runtime layer).
	ReleaseID string

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

	// ReleaseID is the layer this variant belongs to.
	ReleaseID string

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

	// ReleaseID is the layer this chunk belongs to.
	ReleaseID string

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

	// ReleaseID is the layer this profile belongs to.
	ReleaseID string

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
