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

package querier_sqlite

import (
	"context"
	"database/sql"

	"piko.sh/piko/internal/registry/registry_dal"
	"piko.sh/piko/internal/registry/registry_dal/dalcore"
	registry_db "piko.sh/piko/internal/registry/registry_dal/querier_sqlite/db"
)

// driver is the SQLite implementation of dalcore.Driver. It translates dalcore's
// dialect-neutral flat structs into the generated SQLite query calls.
type driver struct {
	// queries provides access to the generated SQLite query methods.
	queries *registry_db.Queries
}

var (
	_ dalcore.Driver = (*driver)(nil)
)

// New creates a registry DAL backed by the given SQLite database connection.
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

// GetArtefactLayers returns every layer of an artefact tagged with its release id, in
// release id order so the runtime layer is first.
//
// Takes artefactID (string) which identifies the artefact to load.
//
// Returns []dalcore.ArtefactLayerData which holds each layer, tagged and ordered.
// Returns error when the query fails.
func (d *driver) GetArtefactLayers(ctx context.Context, artefactID string) ([]dalcore.ArtefactLayerData, error) {
	rows, err := d.queries.GetArtefact(ctx, artefactID)
	if err != nil {
		return nil, err
	}
	layers := make([]dalcore.ArtefactLayerData, len(rows))
	for i := range rows {
		layers[i] = dalcore.ArtefactLayerData{ID: artefactID, ReleaseID: rows[i].ReleaseID, Data: rows[i].DataFbs}
	}
	return layers, nil
}

// GetArtefactLayersForUpdate returns every tagged layer.
//
// Takes artefactID (string) which identifies the artefact to lock and load.
//
// Returns []dalcore.ArtefactLayerData which holds each layer, tagged and ordered.
// Returns error when the query fails.
func (d *driver) GetArtefactLayersForUpdate(ctx context.Context, artefactID string) ([]dalcore.ArtefactLayerData, error) {
	rows, err := d.queries.GetArtefactForUpdate(ctx, artefactID)
	if err != nil {
		return nil, err
	}
	layers := make([]dalcore.ArtefactLayerData, len(rows))
	for i := range rows {
		layers[i] = dalcore.ArtefactLayerData{ID: artefactID, ReleaseID: rows[i].ReleaseID, Data: rows[i].DataFbs}
	}
	return layers, nil
}

// GetMultipleArtefactLayers returns every layer of the given artefacts, tagged with the
// artefact ID.
//
// Takes artefactIDs ([]string) which identify the artefacts to load.
//
// Returns []dalcore.ArtefactLayerData which holds each layer's ID and FlatBuffer payload.
// Returns error when the query fails.
func (d *driver) GetMultipleArtefactLayers(ctx context.Context, artefactIDs []string) ([]dalcore.ArtefactLayerData, error) {
	rows, err := d.queries.GetMultipleArtefacts(ctx, registry_db.GetMultipleArtefactsParams{IDs: artefactIDs})
	if err != nil {
		return nil, err
	}
	layers := make([]dalcore.ArtefactLayerData, len(rows))
	for i := range rows {
		layers[i] = dalcore.ArtefactLayerData{ID: rows[i].ID, ReleaseID: rows[i].ReleaseID, Data: rows[i].DataFbs}
	}
	return layers, nil
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
func (d *driver) FindArtefactIDByVariantStorageKey(ctx context.Context, storageKey string) (string, bool, error) {
	row, found, err := d.queries.FindArtefactByVariantStorageKey(ctx, storageKey)
	if err != nil {
		return "", false, err
	}
	return row.ArtefactID, found, nil
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
func (d *driver) DecrementBlobRefCount(ctx context.Context, storageKey string, lastReferencedAt int64) (int, bool, error) {
	row, found, err := d.queries.DecrementBlobRefCount(ctx, registry_db.DecrementBlobRefCountParams{
		LastReferencedAt: lastReferencedAt,
		StorageKey:       storageKey,
	})
	if err != nil {
		return 0, false, err
	}
	return int(row.RefCount), found, nil
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
func (d *driver) GetBlobRefCount(ctx context.Context, storageKey string) (int, bool, error) {
	row, found, err := d.queries.GetBlobRefCount(ctx, storageKey)
	if err != nil {
		return 0, false, err
	}
	return int(row.RefCount), found, nil
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
		ReleaseID:  params.ReleaseID,
		SourcePath: params.SourcePath,
		CreatedAt:  params.CreatedAt,
		UpdatedAt:  params.UpdatedAt,
		DataFbs:    params.DataFbs,
	})
}

// InsertArtefactLayerIfAbsent inserts one artefact layer only when it is not already
// present.
//
// Takes params (dalcore.UpsertArtefactParams) which describes the layer to insert.
//
// Returns bool which is true when a row was inserted.
// Returns error when the insert fails.
func (d *driver) InsertArtefactLayerIfAbsent(ctx context.Context, params dalcore.UpsertArtefactParams) (bool, error) {
	_, inserted, err := d.queries.InsertArtefactLayerIfAbsent(ctx, registry_db.InsertArtefactLayerIfAbsentParams{
		ID:         params.ID,
		ReleaseID:  params.ReleaseID,
		SourcePath: params.SourcePath,
		CreatedAt:  params.CreatedAt,
		UpdatedAt:  params.UpdatedAt,
		DataFbs:    params.DataFbs,
	})
	if err != nil {
		return false, err
	}
	return inserted, nil
}

// DeleteArtefact deletes every layer of an artefact by ID.
//
// Takes artefactID (string) which identifies the artefact to delete.
//
// Returns error when the delete fails.
func (d *driver) DeleteArtefact(ctx context.Context, artefactID string) error {
	return d.queries.DeleteArtefact(ctx, artefactID)
}

// DeleteArtefactLayer deletes one artefact layer, keyed by (id, release_id).
//
// Takes artefactID (string) and releaseID (string) which identify the layer.
//
// Returns error when the delete fails.
func (d *driver) DeleteArtefactLayer(ctx context.Context, artefactID, releaseID string) error {
	return d.queries.DeleteArtefactLayer(ctx, registry_db.DeleteArtefactLayerParams{ID: artefactID, ReleaseID: releaseID})
}

// DeleteArtefactLayersForRelease deletes every artefact layer of a release.
//
// Takes releaseID (string) which identifies the release to retire.
//
// Returns error when the delete fails.
func (d *driver) DeleteArtefactLayersForRelease(ctx context.Context, releaseID string) error {
	return d.queries.DeleteArtefactLayersForRelease(ctx, releaseID)
}

// ReclaimArtefactLayersForRelease deletes every artefact layer of a release and returns
// the deleted layers, tagged with their identity.
//
// Takes releaseID (string) which identifies the release to retire.
//
// Returns []dalcore.ArtefactLayerData which holds the deleted layers.
// Returns error when the statement fails.
func (d *driver) ReclaimArtefactLayersForRelease(ctx context.Context, releaseID string) ([]dalcore.ArtefactLayerData, error) {
	rows, err := d.queries.ReclaimArtefactLayersForRelease(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	layers := make([]dalcore.ArtefactLayerData, len(rows))
	for i := range rows {
		layers[i] = dalcore.ArtefactLayerData{ID: rows[i].ID, ReleaseID: rows[i].ReleaseID, Data: rows[i].DataFbs}
	}
	return layers, nil
}

// DeleteVariantTagsForArtefact removes an artefact layer's variant tags.
//
// Takes artefactID (string) and releaseID (string) which identify the layer.
//
// Returns error when the delete fails.
func (d *driver) DeleteVariantTagsForArtefact(ctx context.Context, artefactID, releaseID string) error {
	return d.queries.DeleteVariantTagsForArtefact(ctx, registry_db.DeleteVariantTagsForArtefactParams{ArtefactID: artefactID, ReleaseID: releaseID})
}

// DeleteChunksForArtefact removes every chunk belonging to an artefact.
//
// Takes artefactID (string) which identifies the owning artefact.
//
// Returns error when the delete fails.
func (d *driver) DeleteChunksForArtefact(ctx context.Context, artefactID, releaseID string) error {
	return d.queries.DeleteChunksForArtefact(ctx, registry_db.DeleteChunksForArtefactParams{ArtefactID: artefactID, ReleaseID: releaseID})
}

// DeleteVariantsForArtefact removes an artefact layer's variants.
//
// Takes artefactID (string) and releaseID (string) which identify the layer.
//
// Returns error when the delete fails.
func (d *driver) DeleteVariantsForArtefact(ctx context.Context, artefactID, releaseID string) error {
	return d.queries.DeleteVariantsForArtefact(ctx, registry_db.DeleteVariantsForArtefactParams{ArtefactID: artefactID, ReleaseID: releaseID})
}

// DeleteDesiredProfilesForArtefact removes an artefact layer's desired profiles.
//
// Takes artefactID (string) and releaseID (string) which identify the layer.
//
// Returns error when the delete fails.
func (d *driver) DeleteDesiredProfilesForArtefact(ctx context.Context, artefactID, releaseID string) error {
	return d.queries.DeleteDesiredProfilesForArtefact(ctx, registry_db.DeleteDesiredProfilesForArtefactParams{ArtefactID: artefactID, ReleaseID: releaseID})
}

// InsertVariant stores a variant record.
//
// Takes params (dalcore.InsertVariantParams) which describes the variant to store.
//
// Returns error when the insert fails.
func (d *driver) InsertVariant(ctx context.Context, params dalcore.InsertVariantParams) error {
	return d.queries.InsertVariant(ctx, registry_db.InsertVariantParams{
		ArtefactID:       params.ArtefactID,
		ReleaseID:        params.ReleaseID,
		VariantID:        params.VariantID,
		StorageKey:       params.StorageKey,
		StorageBackendID: params.StorageBackendID,
		MimeType:         params.MimeType,
		SizeBytes:        params.SizeBytes,
		Status:           params.Status,
		CreatedAt:        params.CreatedAt,
	})
}

// InsertVariantTag stores a single metadata tag for a variant in a layer.
//
// Takes artefactID (string) and releaseID (string) which identify the layer.
// Takes variantID (string) which identifies the owning variant.
// Takes tagKey (string) and tagValue (string) which are the tag to store.
//
// Returns error when the insert fails.
func (d *driver) InsertVariantTag(ctx context.Context, artefactID, releaseID, variantID, tagKey, tagValue string) error {
	return d.queries.InsertVariantTag(ctx, registry_db.InsertVariantTagParams{
		ArtefactID: artefactID,
		ReleaseID:  releaseID,
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
		ReleaseID:        params.ReleaseID,
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
		ReleaseID:      params.ReleaseID,
		Name:           params.Name,
		CapabilityName: params.CapabilityName,
		Priority:       params.Priority,
		ParamsJSON:     params.ParamsJSON,
		TagsJSON:       params.TagsJSON,
		DependsOnJSON:  params.DependsOnJSON,
	})
}

// ClaimRelease attempts to claim publishing rights for a release.
//
// Takes params (dalcore.ClaimReleaseParams) which describes the release to claim.
//
// Returns bool which is true when this caller won the claim.
// Returns error when the query fails.
func (d *driver) ClaimRelease(ctx context.Context, params dalcore.ClaimReleaseParams) (bool, error) {
	_, claimed, err := d.queries.ClaimRelease(ctx, registry_db.ClaimReleaseParams{
		ReleaseID:     params.ReleaseID,
		PublishDigest: params.PublishDigest,
		FirstSeenAt:   params.FirstSeenAt,
		HeartbeatAt:   params.HeartbeatAt,
	})
	if err != nil {
		return false, err
	}
	return claimed, nil
}

// GetRelease returns a release lease, or false when the release is unknown.
//
// Takes releaseID (string) which identifies the release.
//
// Returns dalcore.ReleaseLease which is the lease when found.
// Returns bool which is true when the release exists.
// Returns error when the query fails.
func (d *driver) GetRelease(ctx context.Context, releaseID string) (dalcore.ReleaseLease, bool, error) {
	row, found, err := d.queries.GetRelease(ctx, releaseID)
	if err != nil {
		return dalcore.ReleaseLease{}, false, err
	}
	if !found {
		return dalcore.ReleaseLease{}, false, nil
	}
	return dalcore.ReleaseLease{
		ReleaseID:     row.ReleaseID,
		PublishDigest: row.PublishDigest,
		State:         row.State,
		FirstSeenAt:   row.FirstSeenAt,
		PublishedAt:   row.PublishedAt,
		HeartbeatAt:   row.HeartbeatAt,
		RetiredAt:     row.RetiredAt,
	}, true, nil
}

// MarkReleasePublished flips a release lease to published and stamps its timestamps.
//
// Takes releaseID (string), publishedAt (int64) and heartbeatAt (int64).
//
// Returns error when the update fails.
func (d *driver) MarkReleasePublished(ctx context.Context, releaseID string, publishedAt, heartbeatAt int64) error {
	return d.queries.MarkReleasePublished(ctx, registry_db.MarkReleasePublishedParams{
		ReleaseID:   releaseID,
		PublishedAt: publishedAt,
		HeartbeatAt: heartbeatAt,
	})
}

// HeartbeatRelease advances a release's heartbeat when the new value is more recent. The
// new heartbeat feeds both the SET value and the comparison, which makes the update
// monotonic.
//
// Takes releaseID (string) and heartbeatAt (int64).
//
// Returns error when the update fails.
func (d *driver) HeartbeatRelease(ctx context.Context, releaseID string, heartbeatAt int64) error {
	return d.queries.HeartbeatRelease(ctx, registry_db.HeartbeatReleaseParams{
		ReleaseID:    releaseID,
		HeartbeatAt:  heartbeatAt,
		HeartbeatAt2: heartbeatAt,
	})
}

// ListExpiredReleases returns published releases whose heartbeat predates the cutoff.
//
// Takes cutoff (int64) and ownRelease (string) which is excluded from the result.
//
// Returns []string which are the expired release IDs.
// Returns error when the query fails.
func (d *driver) ListExpiredReleases(ctx context.Context, cutoff int64, ownRelease string) ([]string, error) {
	rows, err := d.queries.ListExpiredReleases(ctx, registry_db.ListExpiredReleasesParams{
		HeartbeatAt: cutoff,
		ReleaseID:   ownRelease,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i := range rows {
		ids[i] = rows[i].ReleaseID
	}
	return ids, nil
}

// DeleteReleaseLease removes a release lease row.
//
// Takes releaseID (string) which identifies the release.
//
// Returns error when the delete fails.
func (d *driver) DeleteReleaseLease(ctx context.Context, releaseID string) error {
	return d.queries.DeleteReleaseLease(ctx, releaseID)
}

// DeleteStalePublishingLease removes a publishing lease whose heartbeat predates
// staleBefore.
//
// Takes releaseID (string) which identifies the release, and staleBefore (int64) which is
// the staleness cutoff in Unix seconds.
//
// Returns error when the delete fails.
func (d *driver) DeleteStalePublishingLease(ctx context.Context, releaseID string, staleBefore int64) error {
	return d.queries.DeleteStalePublishingLease(ctx, registry_db.DeleteStalePublishingLeaseParams{
		ReleaseID:   releaseID,
		HeartbeatAt: staleBefore,
	})
}
