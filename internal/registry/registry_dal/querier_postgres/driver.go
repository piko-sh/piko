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

package querier_postgres

import (
	"context"
	"database/sql"

	"piko.sh/piko/internal/registry/registry_dal"
	"piko.sh/piko/internal/registry/registry_dal/dalcore"
	registry_db "piko.sh/piko/internal/registry/registry_dal/querier_postgres/db"
)

// driver is the PostgreSQL implementation of dalcore.Driver. It translates dalcore's
// dialect-neutral flat structs into the generated PostgreSQL query calls.
type driver struct {
	// queries provides access to the generated PostgreSQL query methods.
	queries *registry_db.Queries
}

var (
	_ dalcore.Driver = (*driver)(nil)
)

// New creates a registry DAL backed by the given PostgreSQL database connection.
//
// Takes database (*sql.DB) which provides the database connection.
//
// Returns registry_dal.RegistryDALWithTx which is ready for use.
func New(database *sql.DB) registry_dal.RegistryDALWithTx {
	return dalcore.New(database, &driver{queries: registry_db.New(database)})
}

// WithTx returns a driver whose queries run inside tx.
//
// Takes tx (*sql.Tx) which is the transaction to scope queries to.
//
// Returns dalcore.Driver scoped to the transaction.
func (d *driver) WithTx(tx *sql.Tx) dalcore.Driver {
	return &driver{queries: d.queries.WithTx(tx)}
}

// GetArtefactData returns the FlatBuffer payload for a single artefact.
//
// Takes artefactID (string) which identifies the artefact to load.
//
// Returns []byte which is the artefact's FlatBuffer payload.
// Returns error when the query fails.
func (d *driver) GetArtefactData(ctx context.Context, artefactID string) ([]byte, error) {
	row, err := d.queries.GetArtefact(ctx, artefactID)
	if err != nil {
		return nil, err
	}
	return row.DataFbs, nil
}

// GetMultipleArtefactsData returns the FlatBuffer payloads for the given artefact IDs.
//
// Takes artefactIDs ([]string) which identify the artefacts to load.
//
// Returns [][]byte which holds each artefact's FlatBuffer payload.
// Returns error when the query fails.
func (d *driver) GetMultipleArtefactsData(ctx context.Context, artefactIDs []string) ([][]byte, error) {
	rows, err := d.queries.GetMultipleArtefacts(ctx, registry_db.GetMultipleArtefactsParams{IDs: artefactIDs})
	if err != nil {
		return nil, err
	}
	blobs := make([][]byte, len(rows))
	for i := range rows {
		blobs[i] = rows[i].DataFbs
	}
	return blobs, nil
}

// ListAllArtefactsData returns the FlatBuffer payloads for every artefact.
//
// Returns [][]byte which holds every artefact's FlatBuffer payload.
// Returns error when the query fails.
func (d *driver) ListAllArtefactsData(ctx context.Context) ([][]byte, error) {
	rows, err := d.queries.ListAllArtefactsWithData(ctx)
	if err != nil {
		return nil, err
	}
	blobs := make([][]byte, len(rows))
	for i := range rows {
		blobs[i] = rows[i].DataFbs
	}
	return blobs, nil
}

// ListRecentArtefactsData returns the FlatBuffer payloads for the most recently updated
// artefacts.
//
// Takes limit (int) which caps the number of artefacts returned.
//
// Returns [][]byte which holds the most recent artefacts' FlatBuffer payloads.
// Returns error when the query fails.
func (d *driver) ListRecentArtefactsData(ctx context.Context, limit int) ([][]byte, error) {
	rows, err := d.queries.ListRecentArtefactsWithData(ctx, limit)
	if err != nil {
		return nil, err
	}
	blobs := make([][]byte, len(rows))
	for i := range rows {
		blobs[i] = rows[i].DataFbs
	}
	return blobs, nil
}

// ListAllArtefactIDs returns every artefact ID in the store.
//
// Returns []string which holds every artefact ID in the store.
// Returns error when the query fails.
func (d *driver) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	rows, err := d.queries.ListAllArtefactIDs(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids, nil
}

// FindArtefactIDsByTag returns the IDs of artefacts carrying the given tag key and value.
//
// Takes tagKey (string) which is the tag key to match.
// Takes tagValue (string) which is the tag value to match.
//
// Returns []string which holds the IDs of the matching artefacts.
// Returns error when the query fails.
func (d *driver) FindArtefactIDsByTag(ctx context.Context, tagKey, tagValue string) ([]string, error) {
	rows, err := d.queries.FindArtefactIDsByTag(ctx, registry_db.FindArtefactIDsByTagParams{
		TagKey:   tagKey,
		TagValue: tagValue,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ArtefactID
	}
	return ids, nil
}

// FindArtefactIDsByTagValues returns the IDs of artefacts whose tag key matches any of
// the given values.
//
// Takes tagKey (string) which is the tag key to match.
// Takes tagValues ([]string) which are the tag values to match against.
//
// Returns []string which holds the IDs of the matching artefacts.
// Returns error when the query fails.
func (d *driver) FindArtefactIDsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]string, error) {
	rows, err := d.queries.FindArtefactIDsByTagValues(ctx, registry_db.FindArtefactIDsByTagValuesParams{
		TagKey:    tagKey,
		TagValues: tagValues,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ArtefactID
	}
	return ids, nil
}

// FindArtefactIDByVariantStorageKey returns the artefact ID owning the variant stored at
// storageKey.
//
// Takes storageKey (string) which is the variant storage key to look up.
//
// Returns string which is the owning artefact ID.
// Returns error when the query fails.
func (d *driver) FindArtefactIDByVariantStorageKey(ctx context.Context, storageKey string) (string, error) {
	row, err := d.queries.FindArtefactByVariantStorageKey(ctx, storageKey)
	if err != nil {
		return "", err
	}
	return row.ArtefactID, nil
}

// ListVariantStatusCounts returns variant counts grouped by status.
//
// Returns []dalcore.VariantStatusCount which holds the per-status variant counts.
// Returns error when the query fails.
func (d *driver) ListVariantStatusCounts(ctx context.Context) ([]dalcore.VariantStatusCount, error) {
	rows, err := d.queries.ListVariantStatusCounts(ctx)
	if err != nil {
		return nil, err
	}
	counts := make([]dalcore.VariantStatusCount, len(rows))
	for i, row := range rows {
		counts[i] = dalcore.VariantStatusCount{Status: row.Status, Count: row.VariantCount}
	}
	return counts, nil
}

// IncrementBlobRefCount increments (or creates) the reference count for a blob.
//
// Takes params (dalcore.IncrementBlobRefCountParams) which describes the blob to
// reference.
//
// Returns int which is the blob's new reference count.
// Returns error when the query fails.
func (d *driver) IncrementBlobRefCount(ctx context.Context, params dalcore.IncrementBlobRefCountParams) (int, error) {
	row, err := d.queries.IncrementBlobRefCount(ctx, registry_db.IncrementBlobRefCountParams{
		StorageKey:       params.StorageKey,
		StorageBackendID: params.StorageBackendID,
		ContentHash:      params.ContentHash,
		SizeBytes:        params.SizeBytes,
		MimeType:         params.MimeType,
		CreatedAt:        params.CreatedAt,
		LastReferencedAt: params.LastReferencedAt,
	})
	if err != nil {
		return 0, err
	}
	return int(row.RefCount), nil
}

// DecrementBlobRefCount decrements the reference count for a blob.
//
// Takes storageKey (string) which identifies the blob to dereference.
// Takes lastReferencedAt (int64) which is the Unix timestamp to record on the blob.
//
// Returns int which is the blob's new reference count.
// Returns error when the query fails.
func (d *driver) DecrementBlobRefCount(ctx context.Context, storageKey string, lastReferencedAt int64) (int, error) {
	row, err := d.queries.DecrementBlobRefCount(ctx, registry_db.DecrementBlobRefCountParams{
		LastReferencedAt: lastReferencedAt,
		StorageKey:       storageKey,
	})
	if err != nil {
		return 0, err
	}
	return int(row.RefCount), nil
}

// DeleteBlobReferenceIfZero deletes the blob reference record when its count has reached
// zero.
//
// Takes storageKey (string) which identifies the blob reference to delete.
//
// Returns error when the delete fails.
func (d *driver) DeleteBlobReferenceIfZero(ctx context.Context, storageKey string) error {
	return d.queries.DeleteBlobReferenceIfZero(ctx, storageKey)
}

// GetBlobRefCount returns the current reference count for a blob.
//
// Takes storageKey (string) which identifies the blob to inspect.
//
// Returns int which is the blob's current reference count.
// Returns error when the query fails.
func (d *driver) GetBlobRefCount(ctx context.Context, storageKey string) (int, error) {
	row, err := d.queries.GetBlobRefCount(ctx, storageKey)
	if err != nil {
		return 0, err
	}
	return int(row.RefCount), nil
}

// PopGCHints returns up to limit garbage-collection hints.
//
// Takes limit (int) which caps the number of hints returned.
//
// Returns []dalcore.GCHintRow which holds the popped garbage-collection hints.
// Returns error when the query fails.
func (d *driver) PopGCHints(ctx context.Context, limit int) ([]dalcore.GCHintRow, error) {
	rows, err := d.queries.PopGCHints(ctx, limit)
	if err != nil {
		return nil, err
	}
	hints := make([]dalcore.GCHintRow, len(rows))
	for i, row := range rows {
		hints[i] = dalcore.GCHintRow{ID: row.ID, BackendID: row.BackendID, StorageKey: row.StorageKey}
	}
	return hints, nil
}

// DeleteGCHints deletes the garbage-collection hints with the given row IDs.
//
// Takes ids ([]int64) which are the row IDs of the hints to delete.
//
// Returns error when the delete fails.
func (d *driver) DeleteGCHints(ctx context.Context, ids []int64) error {
	return d.queries.DeleteGCHints(ctx, registry_db.DeleteGCHintsParams{IDs: ids})
}

// AddGCHint records a garbage-collection hint for a storage key.
//
// Takes backendID (string) which identifies the storage backend.
// Takes storageKey (string) which is the storage key to record for collection.
// Takes createdAt (int64) which is the Unix timestamp to record on the hint.
//
// Returns error when the insert fails.
func (d *driver) AddGCHint(ctx context.Context, backendID, storageKey string, createdAt int64) error {
	return d.queries.AddGCHint(ctx, registry_db.AddGCHintParams{
		BackendID:  backendID,
		StorageKey: storageKey,
		CreatedAt:  createdAt,
	})
}

// UpsertArtefact inserts or updates an artefact record.
//
// Takes params (dalcore.UpsertArtefactParams) which describes the artefact to store.
//
// Returns error when the upsert fails.
func (d *driver) UpsertArtefact(ctx context.Context, params dalcore.UpsertArtefactParams) error {
	return d.queries.UpsertArtefact(ctx, registry_db.UpsertArtefactParams{
		ID:         params.ID,
		SourcePath: params.SourcePath,
		CreatedAt:  params.CreatedAt,
		UpdatedAt:  params.UpdatedAt,
		DataFbs:    params.DataFbs,
	})
}

// DeleteArtefact deletes an artefact by ID.
//
// Takes artefactID (string) which identifies the artefact to delete.
//
// Returns error when the delete fails.
func (d *driver) DeleteArtefact(ctx context.Context, artefactID string) error {
	return d.queries.DeleteArtefact(ctx, artefactID)
}

// DeleteVariantTagsForArtefact removes all variant tags belonging to an artefact.
//
// Takes artefactID (string) which identifies the artefact whose variant tags are removed.
//
// Returns error when the delete fails.
func (d *driver) DeleteVariantTagsForArtefact(ctx context.Context, artefactID string) error {
	return d.queries.DeleteVariantTagsForArtefact(ctx, artefactID)
}

// DeleteChunksForVariant removes all chunks belonging to a variant.
//
// Takes artefactID (string) which identifies the owning artefact.
// Takes variantID (string) which identifies the variant whose chunks are removed.
//
// Returns error when the delete fails.
func (d *driver) DeleteChunksForVariant(ctx context.Context, artefactID, variantID string) error {
	return d.queries.DeleteChunksForVariant(ctx, registry_db.DeleteChunksForVariantParams{
		ArtefactID: artefactID,
		VariantID:  variantID,
	})
}

// DeleteVariantsForArtefact removes all variants belonging to an artefact.
//
// Takes artefactID (string) which identifies the artefact whose variants are removed.
//
// Returns error when the delete fails.
func (d *driver) DeleteVariantsForArtefact(ctx context.Context, artefactID string) error {
	return d.queries.DeleteVariantsForArtefact(ctx, artefactID)
}

// DeleteDesiredProfilesForArtefact removes all desired profiles belonging to an artefact.
//
// Takes artefactID (string) which identifies the artefact whose desired profiles are
// removed.
//
// Returns error when the delete fails.
func (d *driver) DeleteDesiredProfilesForArtefact(ctx context.Context, artefactID string) error {
	return d.queries.DeleteDesiredProfilesForArtefact(ctx, artefactID)
}

// InsertVariant stores a variant record.
//
// Takes params (dalcore.InsertVariantParams) which describes the variant to store.
//
// Returns error when the insert fails.
func (d *driver) InsertVariant(ctx context.Context, params dalcore.InsertVariantParams) error {
	return d.queries.InsertVariant(ctx, registry_db.InsertVariantParams{
		ArtefactID:       params.ArtefactID,
		VariantID:        params.VariantID,
		StorageKey:       params.StorageKey,
		StorageBackendID: params.StorageBackendID,
		MimeType:         params.MimeType,
		SizeBytes:        params.SizeBytes,
		Status:           params.Status,
		CreatedAt:        params.CreatedAt,
	})
}

// InsertVariantTag stores a single metadata tag for a variant.
//
// Takes artefactID (string) which identifies the owning artefact.
// Takes variantID (string) which identifies the owning variant.
// Takes tagKey (string) which is the tag key to store.
// Takes tagValue (string) which is the tag value to store.
//
// Returns error when the insert fails.
func (d *driver) InsertVariantTag(ctx context.Context, artefactID, variantID, tagKey, tagValue string) error {
	return d.queries.InsertVariantTag(ctx, registry_db.InsertVariantTagParams{
		ArtefactID: artefactID,
		VariantID:  variantID,
		TagKey:     tagKey,
		TagValue:   tagValue,
	})
}

// InsertVariantChunk stores a single chunk record for a variant.
//
// Takes params (dalcore.InsertVariantChunkParams) which describes the chunk to store.
//
// Returns error when the insert fails.
func (d *driver) InsertVariantChunk(ctx context.Context, params dalcore.InsertVariantChunkParams) error {
	return d.queries.InsertVariantChunk(ctx, registry_db.InsertVariantChunkParams{
		ArtefactID:       params.ArtefactID,
		VariantID:        params.VariantID,
		ChunkID:          params.ChunkID,
		StorageKey:       params.StorageKey,
		StorageBackendID: params.StorageBackendID,
		SizeBytes:        params.SizeBytes,
		ContentHash:      params.ContentHash,
		SequenceNumber:   params.SequenceNumber,
		MimeType:         params.MimeType,
		CreatedAt:        params.CreatedAt,
		DurationSeconds:  params.DurationSeconds,
	})
}

// InsertDesiredProfile stores a single desired-profile record.
//
// Takes params (dalcore.InsertDesiredProfileParams) which describes the profile to store.
//
// Returns error when the insert fails.
func (d *driver) InsertDesiredProfile(ctx context.Context, params dalcore.InsertDesiredProfileParams) error {
	return d.queries.InsertDesiredProfile(ctx, registry_db.InsertDesiredProfileParams{
		ArtefactID:     params.ArtefactID,
		Name:           params.Name,
		CapabilityName: params.CapabilityName,
		Priority:       params.Priority,
		ParamsJSON:     params.ParamsJSON,
		TagsJSON:       params.TagsJSON,
		DependsOnJSON:  params.DependsOnJSON,
	})
}
