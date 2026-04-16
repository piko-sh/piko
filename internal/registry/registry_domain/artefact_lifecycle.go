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
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"mime"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// logKeyArtefactID is the logging key for artefact identifiers.
	logKeyArtefactID = "artefactID"

	// logKeyVariantID is the logging key for variant identifiers.
	logKeyVariantID = "variantID"

	// logKeyStorageBackendID is the log field key for the storage backend identifier.
	logKeyStorageBackendID = "storageBackendID"

	// logKeyStorageKey is the logging key for blob storage paths.
	logKeyStorageKey = "storageKey"

	// logKeyOldStorageKey is the log field key for the old storage key.
	logKeyOldStorageKey = "oldStorageKey"

	// logKeyChunkID is the log field key for chunk identifiers.
	logKeyChunkID = "chunkID"

	// logKeyNewRefCount is the log field key for the updated reference count.
	logKeyNewRefCount = "newRefCount"

	// logKeyShouldDelete is the log key that shows whether a blob should be deleted.
	logKeyShouldDelete = "shouldDelete"

	// logKeySource is the key used to mark source artefacts in variant metadata.
	logKeySource = "source"

	// errMessageStorageBackendNotFound is the error message used when a storage
	// backend ID is not found in the configured blob stores.
	errMessageStorageBackendNotFound = "storage backend not found"

	// errStorageBackendNotFound is the error code used when a storage backend ID
	// is not found in the configured blob stores.
	errStorageBackendNotFound = "Storage backend not found"
)

var (
	// sha256Pool reuses SHA-256 hash.Hash instances to reduce allocation pressure.
	sha256Pool = sync.Pool{New: func() any { return sha256.New() }}

	// sha384Pool reuses SHA-384 hash.Hash instances to reduce allocation pressure.
	sha384Pool = sync.Pool{New: func() any { return sha512.New384() }}
)

var (
	// errArtefactIDEmpty is returned when an artefact operation is attempted
	// with an empty artefact ID.
	errArtefactIDEmpty = errors.New("artefactID cannot be empty")

	// errSourcePathEmpty is returned when an artefact is ingested with an
	// empty source path.
	errSourcePathEmpty = errors.New("sourcePath cannot be empty")
)

// writeCounter implements io.Writer to count bytes written during blob upload.
type writeCounter struct {
	// total is the number of bytes written.
	total int64
}

// Write implements io.Writer and counts the total bytes written.
//
// Takes p ([]byte) which contains the bytes to count.
//
// Returns int which is the number of bytes processed, always equal to len(p).
// Returns error which is always nil as counting cannot fail.
func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total += int64(n)
	return n, nil
}

// publishEvent sends artefact lifecycle events to the event bus.
//
// Takes eventType (orchestrator_domain.EventType) which specifies the type of
// lifecycle event to send.
// Takes artefactID (string) which identifies the artefact the event relates to.
func (s *registryService) publishEvent(ctx context.Context, eventType orchestrator_domain.EventType, artefactID string) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.publishEvent",
		logger_domain.String("eventType", string(eventType)),
		logger_domain.String("artefactID", artefactID),
	)
	defer span.End()

	if s.eventBus == nil {
		l.Warn("Event bus is nil, skipping event publication")
		span.SetStatus(codes.Error, "Event bus is nil, skipping event publication")
		return
	}

	l.Trace("Creating event payload")

	payload := getEventPayload()
	payload["artefactID"] = artefactID
	payload["timestamp"] = time.Now().UTC()

	carrier := propagation.MapCarrier{}
	getTextMapPropagator().Inject(ctx, carrier)

	for k, v := range carrier {
		payload[k] = v
	}

	event := orchestrator_domain.Event{
		Type:    eventType,
		Payload: payload,
	}

	l.Trace("Publishing artefact event to topic",
		logger_domain.String("topic", string(eventType)),
		logger_domain.String("artefactID", artefactID))
	err := s.eventBus.Publish(ctx, string(eventType), event)

	putEventPayload(payload)

	if err != nil {
		l.ReportError(span, err, "Failed to publish artefact event")
	} else {
		l.Trace("Successfully published artefact event")
		span.SetStatus(codes.Ok, "Successfully published event")
		registryServiceEventPublishCount.Add(ctx, 1)
		s.artefactEventsPublished.Add(1)
	}
}

// upsertInput holds the checked input data for UpsertArtefact.
type upsertInput struct {
	// sourceData holds the raw artefact content to store; nil means no blob update.
	sourceData io.Reader

	// artefactID is the unique identifier of the artefact to upsert.
	artefactID string

	// sourcePath is the file path from which the source data was read.
	sourcePath string

	// storageBackendID specifies which storage backend to use for blob operations.
	storageBackendID string

	// desiredProfiles is the list of profiles to create or update.
	desiredProfiles []registry_dto.NamedProfile
}

// resolveVariantsForUpsert finds the final variants based on whether source
// data is available.
//
// Takes input (upsertInput) which contains the source data and metadata.
// Takes isNewArtefact (bool) which indicates if this is a new artefact.
// Takes existingArtefact (*registry_dto.ArtefactMeta) which provides the
// current artefact state for updates.
//
// Returns []registry_dto.Variant which contains the resolved variants.
// Returns error when blob processing fails.
func (s *registryService) resolveVariantsForUpsert(
	ctx context.Context,
	store MetadataStore,
	input upsertInput,
	isNewArtefact bool,
	existingArtefact *registry_dto.ArtefactMeta,
) ([]registry_dto.Variant, error) {
	ctx, l := logger_domain.From(ctx, log)
	if input.sourceData != nil {
		l.Trace("Source data provided, processing blob update")
		return s.processBlobUpdate(ctx, store, input.sourceData, input.sourcePath,
			input.storageBackendID, isNewArtefact, existingArtefact)
	}

	l.Trace("No source data provided, performing metadata-only update")
	if isNewArtefact {
		l.Trace("Metadata-only update for a new artefact; creating a placeholder record")
		return []registry_dto.Variant{}, nil
	}

	l.Trace("Preserving existing variants for metadata-only update",
		logger_domain.Int("variantCount", len(existingArtefact.ActualVariants)))
	return existingArtefact.ActualVariants, nil
}

// UpsertArtefact creates or updates an artefact in the registry.
//
// When sourceData is provided, it uploads the blob and creates or updates the
// source variant. When sourceData is nil, it performs a metadata-only update,
// such as updating desired profiles.
//
// For metadata-only updates where the artefact already exists with identical
// desired profiles, the update and event publication are skipped entirely.
// This prevents unnecessary event storms during rapid page reloads.
//
// Takes artefactID (string) which identifies the artefact to create or update.
// Takes sourcePath (string) which specifies the path to the source file.
// Takes sourceData (io.Reader) which provides the blob data to upload, or nil
// for metadata-only updates.
// Takes storageBackendID (string) which identifies the storage backend to use.
// Takes desiredProfiles ([]registry_dto.NamedProfile) which specifies the
// profiles to associate with the artefact.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the input is invalid or the operation fails.
func (s *registryService) UpsertArtefact(
	ctx context.Context,
	artefactID string,
	sourcePath string,
	sourceData io.Reader,
	storageBackendID string,
	desiredProfiles []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	startTime := time.Now()
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.UpsertArtefact",
		logger_domain.String(logKeyArtefactID, artefactID),
	)
	defer func() {
		span.End()
		registryServiceUpsertArtefactDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	if err := validateUpsertInput(artefactID, sourcePath); err != nil {
		l.ReportError(span, err, "Invalid input")
		registryServiceUpsertArtefactErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("validating upsert input for %q: %w", artefactID, err)
	}

	existingArtefact, err := s.getArtefactForUpsert(ctx, artefactID)
	isNewArtefact := errors.Is(err, ErrArtefactNotFound)
	if err != nil && !isNewArtefact {
		l.ReportError(span, err, "Failed to get existing artefact")
		registryServiceUpsertArtefactErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("failed to get existing artefact '%s': %w", artefactID, err)
	}

	if !isNewArtefact && sourceData == nil && profilesMatch(existingArtefact.DesiredProfiles, desiredProfiles) {
		l.Trace("Skipping upsert, artefact unchanged",
			logger_domain.Int("profileCount", len(desiredProfiles)))
		registryServiceUpsertArtefactSkippedCount.Add(ctx, 1)
		return existingArtefact, nil
	}

	input := upsertInput{
		desiredProfiles:  desiredProfiles,
		sourceData:       sourceData,
		artefactID:       artefactID,
		sourcePath:       sourcePath,
		storageBackendID: storageBackendID,
	}

	var finalArtefact *registry_dto.ArtefactMeta
	err = s.metaStore.RunAtomic(ctx, func(ctx context.Context, transactionStore MetadataStore) error {
		finalVariants, resolveErr := s.resolveVariantsForUpsert(ctx, transactionStore, input, isNewArtefact, existingArtefact)
		if resolveErr != nil {
			return fmt.Errorf("resolving variants for upsert of %q: %w", artefactID, resolveErr)
		}

		finalArtefact = buildArtefactMeta(artefactID, sourcePath, finalVariants, desiredProfiles,
			isNewArtefact, existingArtefact)

		return s.persistAndFinaliseUpsert(ctx, transactionStore, finalArtefact, isNewArtefact)
	})
	if err != nil {
		registryServiceUpsertArtefactErrorCount.Add(ctx, 1)
		return nil, err
	}

	return finalArtefact, nil
}

// getArtefactForUpsert retrieves an artefact using the service-level cache
// before falling back to the metadata store. This avoids a full SQLite read
// and FlatBuffer deserialisation when the artefact is already cached.
//
// Takes artefactID (string) which identifies the artefact to retrieve.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the artefact cannot be found or retrieval fails.
func (s *registryService) getArtefactForUpsert(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	if cached, found := s.tryGetFromCache(ctx, artefactID); found {
		return cached, nil
	}
	return s.metaStore.GetArtefact(ctx, artefactID)
}

// persistAndFinaliseUpsert saves the artefact to the store, updates cache,
// and publishes events.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact to save.
// Takes isNewArtefact (bool) which indicates if this is a new artefact or an
// update to an existing one.
//
// Returns error when the atomic update of metadata fails.
func (s *registryService) persistAndFinaliseUpsert(
	ctx context.Context,
	store MetadataStore,
	artefact *registry_dto.ArtefactMeta,
	isNewArtefact bool,
) error {
	actions := []registry_dto.AtomicAction{{
		Type:       registry_dto.ActionTypeUpsertArtefact,
		ArtefactID: artefact.ID,
		Artefact:   artefact,
		GCHints:    nil,
	}}

	if err := store.AtomicUpdate(ctx, actions); err != nil {
		return fmt.Errorf("atomic update of metadata failed: %w", err)
	}

	if s.cache != nil {
		s.cache.Set(ctx, artefact)
	}

	eventType := EventArtefactUpdated
	if isNewArtefact {
		eventType = EventArtefactCreated
	}
	s.publishEvent(ctx, eventType, artefact.ID)
	return nil
}

// blobUploadResult holds the outcome of uploading a blob to temporary storage.
type blobUploadResult struct {
	// tempKey is the storage key for the blob while it is being uploaded.
	tempKey string

	// hash is the content hash used for deduplication and integrity checks.
	hash string

	// sriHash is the SHA-384 Subresource Integrity hash in "sha384-<base64>"
	// format, computed alongside the content hash at zero additional cost.
	sriHash string

	// finalKey is the storage key that identifies where the blob is stored.
	finalKey string

	// mimeType is the content type of the uploaded blob (e.g. "image/png").
	mimeType string

	// size is the blob size in bytes.
	size int64
}

// incrementBlobRefCount increases the reference count for a blob.
//
// Takes upload (*blobUploadResult) which contains the blob details to track.
// Takes storageBackendID (string) which identifies the storage backend.
//
// Returns error when the reference count update fails.
func (*registryService) incrementBlobRefCount(
	ctx context.Context,
	store MetadataStore,
	upload *blobUploadResult,
	storageBackendID string,
) error {
	ctx, l := logger_domain.From(ctx, log)
	now := time.Now().UTC()
	blobRef := BlobReference{
		CreatedAt:        now,
		LastReferencedAt: now,
		StorageKey:       upload.finalKey,
		StorageBackendID: storageBackendID,
		ContentHash:      upload.hash,
		MimeType:         upload.mimeType,
		SizeBytes:        upload.size,
	}

	newRefCount, err := store.IncrementBlobRefCount(ctx, blobRef)
	if err != nil {
		l.Error("Failed to increment blob ref count",
			logger_domain.Error(err), logger_domain.String(logKeyStorageKey, upload.finalKey))
		return fmt.Errorf("failed to increment blob ref count for %s: %w", upload.finalKey, err)
	}
	l.Trace("Incremented blob ref count",
		logger_domain.String(logKeyStorageKey, upload.finalKey),
		logger_domain.Int(logKeyNewRefCount, newRefCount))
	registryServiceBlobRefCountIncrementCount.Add(ctx, 1)
	return nil
}

// decrementOldBlobRefCount lowers the reference count for a replaced blob.
//
// Takes oldStorageKey (string) which identifies the blob being replaced.
// Takes newStorageKey (string) which identifies the replacement blob.
//
// Returns error when the reference count cannot be decremented.
func (*registryService) decrementOldBlobRefCount(
	ctx context.Context,
	store MetadataStore,
	oldStorageKey, newStorageKey string,
) error {
	ctx, l := logger_domain.From(ctx, log)
	if oldStorageKey == "" || oldStorageKey == newStorageKey {
		return nil
	}
	newRefCount, shouldDelete, err := store.DecrementBlobRefCount(ctx, oldStorageKey)
	if errors.Is(err, ErrBlobReferenceNotFound) {
		l.Trace("Old blob reference not found, skipping decrement",
			logger_domain.String(logKeyOldStorageKey, oldStorageKey))
		return nil
	}
	if err != nil {
		return fmt.Errorf("decrementing old blob ref count for %q: %w", oldStorageKey, err)
	}
	l.Trace("Decremented old blob ref count",
		logger_domain.String(logKeyOldStorageKey, oldStorageKey),
		logger_domain.Int(logKeyNewRefCount, newRefCount),
		logger_domain.Bool(logKeyShouldDelete, shouldDelete))
	registryServiceBlobRefCountDecrementCount.Add(ctx, 1)
	return nil
}

// processBlobUpdate handles blob storage when source data is provided. It
// creates a new source variant and removes any variants that depend on the old
// source.
//
// Takes sourceData (io.Reader) which provides the blob content to store.
// Takes sourcePath (string) which specifies the original file path for MIME
// type detection.
// Takes storageBackendID (string) which identifies which blob storage to use.
// Takes isNewArtefact (bool) which indicates whether this is a new artefact.
// Takes existingArtefact (*registry_dto.ArtefactMeta) which provides the
// current artefact metadata for updates.
//
// Returns []registry_dto.Variant which contains the updated variant list.
// Returns error when blob storage fails or reference count update fails.
func (s *registryService) processBlobUpdate(
	ctx context.Context,
	store MetadataStore,
	sourceData io.Reader,
	sourcePath string,
	storageBackendID string,
	isNewArtefact bool,
	existingArtefact *registry_dto.ArtefactMeta,
) ([]registry_dto.Variant, error) {
	ctx, l := logger_domain.From(ctx, log)
	mimeType := detectMimeType(sourcePath)
	l.Trace("Detected MIME type for asset",
		logger_domain.String("sourcePath", sourcePath), logger_domain.String("mimeType", mimeType))

	blobStore, err := s.GetBlobStore(storageBackendID)
	if err != nil {
		l.Error("Failed to get blob store", logger_domain.Error(err))
		return nil, fmt.Errorf("getting blob store %q: %w", storageBackendID, err)
	}

	upload, err := uploadTempBlob(ctx, blobStore, sourceData, sourcePath, mimeType)
	if err != nil {
		l.Error("Failed to upload temp blob", logger_domain.Error(err))
		return nil, fmt.Errorf("uploading temp blob for %q: %w", sourcePath, err)
	}

	oldStorageKey := getOldSourceStorageKey(existingArtefact, isNewArtefact)
	if oldStorageKey == upload.finalKey {
		l.Trace("Source content unchanged (hash is identical), skipping blob update.",
			logger_domain.String(logKeyStorageKey, upload.finalKey))
		_ = blobStore.Delete(ctx, upload.tempKey)
		return existingArtefact.ActualVariants, nil
	}

	if err := finaliseBlobStorage(ctx, blobStore, upload.tempKey, upload.finalKey); err != nil {
		l.Error("Failed to finalise blob storage", logger_domain.Error(err))
		_ = blobStore.Delete(ctx, upload.tempKey)
		return nil, fmt.Errorf("finalising blob storage for %q: %w", sourcePath, err)
	}

	if err := s.incrementBlobRefCount(ctx, store, upload, storageBackendID); err != nil {
		l.Error("Failed to increment blob ref count", logger_domain.Error(err))
		return nil, fmt.Errorf("failed to register blob in ref count table: %w", err)
	}
	if err := s.decrementOldBlobRefCount(ctx, store, oldStorageKey, upload.finalKey); err != nil {
		return nil, fmt.Errorf("decrementing old blob ref count: %w", err)
	}

	newSourceVariant := buildSourceVariant(upload, storageBackendID)
	if isNewArtefact {
		return []registry_dto.Variant{newSourceVariant}, nil
	}
	return applyVariantInvalidation(ctx, existingArtefact, &newSourceVariant), nil
}

// variantReplacementInfo holds details about a variant being replaced.
type variantReplacementInfo struct {
	// oldStorageKey is the storage key of the variant being replaced.
	oldStorageKey string

	// backendID is the storage backend ID for the replaced variant.
	backendID string

	// oldChunks holds the chunk metadata from the variant being replaced.
	oldChunks []registry_dto.VariantChunk

	// found indicates whether an existing variant was replaced.
	found bool
}

// incrementVariantRefCounts increases the reference counts for a variant's
// blob and all its chunks.
//
// Takes variant (*registry_dto.Variant) which provides the blob and chunk data.
//
// Returns error when variant fields are missing or invalid, or when any
// reference count update fails.
func (s *registryService) incrementVariantRefCounts(
	ctx context.Context,
	store MetadataStore,
	variant *registry_dto.Variant,
) error {
	ctx, l := logger_domain.From(ctx, log)
	if variant.StorageKey == "" {
		return fmt.Errorf("variant %s has empty StorageKey", variant.VariantID)
	}
	if variant.StorageBackendID == "" {
		return fmt.Errorf("variant %s has empty StorageBackendID", variant.VariantID)
	}
	if variant.SizeBytes <= 0 {
		return fmt.Errorf("variant %s has invalid SizeBytes: %d", variant.VariantID, variant.SizeBytes)
	}
	if variant.MimeType == "" {
		return fmt.Errorf("variant %s has empty MimeType", variant.VariantID)
	}

	now := time.Now().UTC()
	blobRef := BlobReference{
		CreatedAt:        now,
		LastReferencedAt: now,
		StorageKey:       variant.StorageKey,
		StorageBackendID: variant.StorageBackendID,
		ContentHash:      variant.ContentHash,
		MimeType:         variant.MimeType,
		SizeBytes:        variant.SizeBytes,
	}

	newRefCount, err := store.IncrementBlobRefCount(ctx, blobRef)
	if err != nil {
		l.Error("Failed to increment blob ref count for new variant",
			logger_domain.Error(err), logger_domain.String(logKeyStorageKey, variant.StorageKey))
		return fmt.Errorf("failed to increment blob ref count for variant: %w", err)
	}
	l.Trace("Incremented blob ref count for new variant",
		logger_domain.String(logKeyStorageKey, variant.StorageKey),
		logger_domain.Int(logKeyNewRefCount, newRefCount))
	registryServiceBlobRefCountIncrementCount.Add(ctx, 1)

	return s.incrementChunkRefCounts(ctx, store, variant.Chunks)
}

// incrementChunkRefCounts increases the reference count for all chunk blobs.
// It stops on the first failure to avoid leaving reference counts in a partial
// state.
//
// Takes chunks ([]registry_dto.VariantChunk) which contains the chunks to
// update.
//
// Returns error when a chunk has missing or invalid fields, or when the
// reference count update fails.
func (*registryService) incrementChunkRefCounts(
	ctx context.Context,
	store MetadataStore,
	chunks []registry_dto.VariantChunk,
) error {
	ctx, l := logger_domain.From(ctx, log)
	for i := range chunks {
		chunk := &chunks[i]

		if chunk.StorageKey == "" {
			return fmt.Errorf("chunk %s has empty StorageKey", chunk.ChunkID)
		}
		if chunk.StorageBackendID == "" {
			return fmt.Errorf("chunk %s has empty StorageBackendID", chunk.ChunkID)
		}
		if chunk.SizeBytes <= 0 {
			return fmt.Errorf("chunk %s has invalid SizeBytes: %d", chunk.ChunkID, chunk.SizeBytes)
		}
		if chunk.MimeType == "" {
			return fmt.Errorf("chunk %s has empty MimeType", chunk.ChunkID)
		}

		now := time.Now().UTC()
		chunkBlobRef := BlobReference{
			CreatedAt:        now,
			LastReferencedAt: now,
			StorageKey:       chunk.StorageKey,
			StorageBackendID: chunk.StorageBackendID,
			ContentHash:      chunk.ContentHash,
			MimeType:         chunk.MimeType,
			SizeBytes:        chunk.SizeBytes,
		}

		chunkRefCount, err := store.IncrementBlobRefCount(ctx, chunkBlobRef)
		if err != nil {
			l.Error("Failed to increment blob ref count for chunk",
				logger_domain.Error(err), logger_domain.String(logKeyChunkID, chunk.ChunkID),
				logger_domain.String(logKeyStorageKey, chunk.StorageKey))
			return fmt.Errorf("failed to increment blob ref count for chunk %s: %w", chunk.ChunkID, err)
		}
		l.Trace("Incremented blob ref count for chunk",
			logger_domain.String(logKeyChunkID, chunk.ChunkID),
			logger_domain.String(logKeyStorageKey, chunk.StorageKey),
			logger_domain.Int(logKeyNewRefCount, chunkRefCount))
		registryServiceBlobRefCountIncrementCount.Add(ctx, 1)
	}
	return nil
}

// decrementOldVariantRefCounts lowers reference counts for a replaced variant
// and returns garbage collection hints.
//
// Takes info (variantReplacementInfo) which holds details about the replaced
// variant.
// Takes newStorageKey (string) which identifies the new storage location.
//
// Returns []registry_dto.GCHint which contains hints for garbage
// collection of blobs that are no longer needed.
// Returns error when a reference count update fails.
func (s *registryService) decrementOldVariantRefCounts(
	ctx context.Context,
	store MetadataStore,
	info variantReplacementInfo,
	newStorageKey string,
) ([]registry_dto.GCHint, error) {
	if !info.found {
		return nil, nil
	}

	mainHints, err := s.decrementMainBlobRefCount(ctx, store, info, newStorageKey)
	if err != nil {
		return nil, err
	}

	chunkHints, err := s.decrementOldChunkRefCounts(ctx, store, info.oldChunks)
	if err != nil {
		return mainHints, err
	}

	return append(mainHints, chunkHints...), nil
}

// decrementMainBlobRefCount lowers the reference count for the old blob and
// returns a garbage collection hint if the blob is no longer in use.
//
// Takes info (variantReplacementInfo) which contains the old storage key and
// backend ID.
// Takes newStorageKey (string) which is compared to skip updates where the
// key has not changed.
//
// Returns []registry_dto.GCHint which contains a hint for garbage
// collection when the old blob should be deleted, or nil otherwise.
// Returns error when the blob reference count cannot be decremented.
func (*registryService) decrementMainBlobRefCount(
	ctx context.Context,
	store MetadataStore,
	info variantReplacementInfo,
	newStorageKey string,
) ([]registry_dto.GCHint, error) {
	ctx, l := logger_domain.From(ctx, log)
	if info.oldStorageKey == "" || info.oldStorageKey == newStorageKey {
		return nil, nil
	}

	newRefCount, shouldDelete, err := store.DecrementBlobRefCount(ctx, info.oldStorageKey)
	if errors.Is(err, ErrBlobReferenceNotFound) {
		l.Trace("Old blob reference not found, skipping decrement",
			logger_domain.String(logKeyOldStorageKey, info.oldStorageKey))
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("decrementing main blob ref count for %q: %w", info.oldStorageKey, err)
	}

	l.Trace("Decremented old blob ref count",
		logger_domain.String(logKeyOldStorageKey, info.oldStorageKey),
		logger_domain.Int(logKeyNewRefCount, newRefCount),
		logger_domain.Bool(logKeyShouldDelete, shouldDelete))
	registryServiceBlobRefCountDecrementCount.Add(ctx, 1)

	if shouldDelete {
		l.Trace("Old variant blob no longer referenced, marking for GC",
			logger_domain.String(logKeyStorageKey, info.oldStorageKey))
		return []registry_dto.GCHint{{BackendID: info.backendID, StorageKey: info.oldStorageKey}}, nil
	}
	return nil, nil
}

// decrementOldChunkRefCounts lowers reference counts for old chunk blobs and
// returns hints for garbage collection.
//
// Takes oldChunks ([]registry_dto.VariantChunk) which contains the chunks to
// update.
//
// Returns []registry_dto.GCHint which contains hints for blobs that
// are no longer used and may be deleted.
// Returns error when a chunk reference count cannot be decremented.
func (*registryService) decrementOldChunkRefCounts(
	ctx context.Context,
	store MetadataStore,
	oldChunks []registry_dto.VariantChunk,
) ([]registry_dto.GCHint, error) {
	ctx, l := logger_domain.From(ctx, log)
	if len(oldChunks) == 0 {
		return nil, nil
	}

	l.Trace("Decrementing blob ref counts for old variant chunks", logger_domain.Int("chunkCount", len(oldChunks)))
	var hints []registry_dto.GCHint

	for i := range oldChunks {
		chunk := &oldChunks[i]
		chunkRefCount, shouldDelete, err := store.DecrementBlobRefCount(ctx, chunk.StorageKey)
		if errors.Is(err, ErrBlobReferenceNotFound) {
			l.Trace("Old chunk blob reference not found, skipping decrement",
				logger_domain.String(logKeyChunkID, chunk.ChunkID),
				logger_domain.String(logKeyStorageKey, chunk.StorageKey))
			continue
		}
		if err != nil {
			return hints, fmt.Errorf("decrementing chunk %q ref count: %w", chunk.ChunkID, err)
		}

		l.Trace("Decremented old chunk blob ref count",
			logger_domain.String(logKeyChunkID, chunk.ChunkID),
			logger_domain.String(logKeyStorageKey, chunk.StorageKey),
			logger_domain.Int(logKeyNewRefCount, chunkRefCount),
			logger_domain.Bool(logKeyShouldDelete, shouldDelete))
		registryServiceBlobRefCountDecrementCount.Add(ctx, 1)

		if shouldDelete {
			l.Trace("Old chunk blob no longer referenced, marking for GC",
				logger_domain.String(logKeyChunkID, chunk.ChunkID),
				logger_domain.String(logKeyStorageKey, chunk.StorageKey))
			hints = append(hints, registry_dto.GCHint{BackendID: chunk.StorageBackendID, StorageKey: chunk.StorageKey})
		}
	}
	return hints, nil
}

// persistVariantUpdate saves the updated artefact to the metadata store with
// optional GC hints.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the artefact
// metadata to persist.
// Takes hintsToAdd ([]registry_dto.GCHint) which specifies garbage collection
// hints to add atomically.
//
// Returns error when the atomic update fails.
func (s *registryService) persistVariantUpdate(
	ctx context.Context,
	store MetadataStore,
	artefact *registry_dto.ArtefactMeta,
	hintsToAdd []registry_dto.GCHint,
) error {
	actions := []registry_dto.AtomicAction{{
		Type:       registry_dto.ActionTypeUpsertArtefact,
		ArtefactID: artefact.ID,
		Artefact:   artefact,
		GCHints:    nil,
	}}

	if len(hintsToAdd) > 0 {
		actions = append(actions, registry_dto.AtomicAction{
			Type:       registry_dto.ActionTypeAddGCHints,
			ArtefactID: "",
			Artefact:   nil,
			GCHints:    hintsToAdd,
		})
	}

	if err := store.AtomicUpdate(ctx, actions); err != nil {
		return fmt.Errorf("failed to add variant via atomic update for artefact '%s': %w", artefact.ID, err)
	}

	if s.cache != nil {
		s.cache.Set(ctx, artefact)
	}
	return nil
}

// AddVariant adds or replaces a variant for an artefact. If a variant with
// the same ID already exists, it replaces it and creates a GC hint for the
// old blob.
//
// Takes artefactID (string) which identifies the artefact to modify.
// Takes newVariant (*registry_dto.Variant) which specifies the variant to add.
//
// Returns *registry_dto.ArtefactMeta which contains the updated artefact.
// Returns error when the artefact cannot be found or the update fails.
func (s *registryService) AddVariant(
	ctx context.Context,
	artefactID string,
	newVariant *registry_dto.Variant,
) (*registry_dto.ArtefactMeta, error) {
	startTime := time.Now()
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.AddVariant",
		logger_domain.String(logKeyArtefactID, artefactID),
		logger_domain.String(logKeyVariantID, newVariant.VariantID),
	)
	defer func() {
		span.End()
		registryServiceAddVariantDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	var artefact *registry_dto.ArtefactMeta
	var finalVariants []registry_dto.Variant

	err := s.metaStore.RunAtomic(ctx, func(ctx context.Context, transactionStore MetadataStore) error {
		original, getErr := transactionStore.GetArtefact(ctx, artefactID)
		if getErr != nil {
			return fmt.Errorf("getting artefact %q for variant addition: %w", artefactID, getErr)
		}

		artefact = new(registry_dto.ArtefactMeta)
		*artefact = *original

		var replacementInfo variantReplacementInfo
		finalVariants, replacementInfo = prepareVariantList(ctx, artefact.ActualVariants, newVariant)

		if incrementErr := s.incrementVariantRefCounts(ctx, transactionStore, newVariant); incrementErr != nil {
			return fmt.Errorf("failed to register variant blobs in ref count table: %w", incrementErr)
		}

		gcHints, decrementErr := s.decrementOldVariantRefCounts(ctx, transactionStore, replacementInfo, newVariant.StorageKey)
		if decrementErr != nil {
			return fmt.Errorf("decrementing old variant ref counts: %w", decrementErr)
		}

		artefact.ActualVariants = finalVariants
		artefact.UpdatedAt = time.Now().UTC()

		return s.persistVariantUpdate(ctx, transactionStore, artefact, gcHints)
	})
	if err != nil {
		l.ReportError(span, err, "Failed to add variant")
		registryServiceAddVariantErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("adding variant to %q: %w", artefactID, err)
	}

	l.Trace("Variant added successfully", logger_domain.Int("totalVariants", len(finalVariants)))
	s.publishEvent(ctx, EventArtefactUpdated, artefact.ID)
	return artefact, nil
}

// collectVariantGCHints lowers the reference counts for all variants and
// gathers garbage collection hints for blobs that are no longer in use.
//
// Takes variants ([]registry_dto.Variant) which contains the variants to
// process.
//
// Returns []registry_dto.GCHint which contains hints for blobs that
// should be removed.
// Returns error when a variant reference count cannot be decremented.
func (s *registryService) collectVariantGCHints(
	ctx context.Context,
	store MetadataStore,
	variants []registry_dto.Variant,
) ([]registry_dto.GCHint, error) {
	ctx, l := logger_domain.From(ctx, log)
	var hints []registry_dto.GCHint
	for i := range variants {
		v := &variants[i]
		newRefCount, shouldDelete, err := store.DecrementBlobRefCount(ctx, v.StorageKey)
		if errors.Is(err, ErrBlobReferenceNotFound) {
			l.Trace("Blob reference not found, skipping decrement",
				logger_domain.String(logKeyVariantID, v.VariantID),
				logger_domain.String(logKeyStorageKey, v.StorageKey))
			continue
		}
		if err != nil {
			return hints, fmt.Errorf("decrementing blob ref count for variant %q: %w", v.VariantID, err)
		}

		l.Trace("Decremented blob ref count",
			logger_domain.String(logKeyVariantID, v.VariantID),
			logger_domain.String(logKeyStorageKey, v.StorageKey),
			logger_domain.Int(logKeyNewRefCount, newRefCount),
			logger_domain.Bool(logKeyShouldDelete, shouldDelete))
		registryServiceBlobRefCountDecrementCount.Add(ctx, 1)

		if shouldDelete {
			l.Trace("Blob no longer referenced, marking for GC",
				logger_domain.String(logKeyStorageKey, v.StorageKey))
			hints = append(hints, registry_dto.GCHint{BackendID: v.StorageBackendID, StorageKey: v.StorageKey})
		}

		chunkHints, chunkErr := s.decrementOldChunkRefCounts(ctx, store, v.Chunks)
		if chunkErr != nil {
			return hints, chunkErr
		}
		hints = append(hints, chunkHints...)
	}
	return hints, nil
}

// persistArtefactDeletion executes the atomic deletion with optional GC hints.
//
// Takes artefactID (string) which identifies the artefact to delete.
// Takes gcHints ([]registry_dto.GCHint) which provides optional garbage
// collection hints to include in the atomic update.
//
// Returns error when the atomic update fails.
func (s *registryService) persistArtefactDeletion(
	ctx context.Context,
	store MetadataStore,
	artefactID string,
	gcHints []registry_dto.GCHint,
) error {
	actions := []registry_dto.AtomicAction{{
		Type:       registry_dto.ActionTypeDeleteArtefact,
		ArtefactID: artefactID,
		Artefact:   nil,
		GCHints:    nil,
	}}

	if len(gcHints) > 0 {
		actions = append(actions, registry_dto.AtomicAction{
			Type:       registry_dto.ActionTypeAddGCHints,
			ArtefactID: "",
			Artefact:   nil,
			GCHints:    gcHints,
		})
	}

	if err := store.AtomicUpdate(ctx, actions); err != nil {
		return fmt.Errorf("failed to delete artefact via atomic update for '%s': %w", artefactID, err)
	}

	if s.cache != nil {
		s.cache.Delete(ctx, artefactID)
	}
	return nil
}

// DeleteArtefact removes an artefact and all its variants from the registry.
// It uses reference counting to ensure blobs are only marked for garbage
// collection when no longer referenced.
//
// Takes artefactID (string) which identifies the artefact to remove.
//
// Returns error when fetching or deleting the artefact fails.
func (s *registryService) DeleteArtefact(ctx context.Context, artefactID string) error {
	startTime := time.Now()
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.DeleteArtefact",
		logger_domain.String(logKeyArtefactID, artefactID),
	)
	defer func() {
		span.End()
		registryServiceDeleteArtefactDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	artefact, err := s.metaStore.GetArtefact(ctx, artefactID)
	if err != nil {
		if errors.Is(err, ErrArtefactNotFound) {
			l.Trace("Artefact not found, nothing to delete", logger_domain.String(logKeyArtefactID, artefactID))
			return nil
		}
		l.ReportError(span, err, "Failed to get artefact")
		registryServiceDeleteArtefactErrorCount.Add(ctx, 1)
		return fmt.Errorf("getting artefact %q for deletion: %w", artefactID, err)
	}

	err = s.metaStore.RunAtomic(ctx, func(ctx context.Context, transactionStore MetadataStore) error {
		gcHints, gcErr := s.collectVariantGCHints(ctx, transactionStore, artefact.ActualVariants)
		if gcErr != nil {
			return fmt.Errorf("collecting GC hints for %q: %w", artefactID, gcErr)
		}
		return s.persistArtefactDeletion(ctx, transactionStore, artefactID, gcHints)
	})
	if err != nil {
		l.ReportError(span, err, "Failed to delete artefact")
		registryServiceDeleteArtefactErrorCount.Add(ctx, 1)
		return fmt.Errorf("deleting artefact %q: %w", artefactID, err)
	}

	l.Trace("Artefact deleted successfully", logger_domain.Int("variantCount", len(artefact.ActualVariants)))
	s.publishEvent(ctx, EventArtefactDeleted, artefactID)
	return nil
}

// BuildDependencyMap creates a map from profile names to their first
// dependency. Exported for testing.
//
// Takes profiles ([]registry_dto.NamedProfile) which contains the named
// profiles to get dependencies from.
//
// Returns map[string]string which maps each profile name to its first
// dependency. Profiles with no dependencies are not in the map.
func BuildDependencyMap(profiles []registry_dto.NamedProfile) map[string]string {
	depMap := make(map[string]string)
	for i := range profiles {
		if profiles[i].Profile.DependsOn.Len() > 0 {
			depMap[profiles[i].Name] = profiles[i].Profile.DependsOn.First()
		}
	}
	return depMap
}

// validateUpsertInput checks the input values for UpsertArtefact.
//
// Takes artefactID (string) which is the artefact to update or insert.
// Takes sourcePath (string) which is the path to the source file.
//
// Returns error when artefactID or sourcePath is empty.
func validateUpsertInput(artefactID, sourcePath string) error {
	if artefactID == "" {
		return errArtefactIDEmpty
	}
	if sourcePath == "" {
		return errSourcePathEmpty
	}
	return nil
}

// buildArtefactMeta builds the final artefact metadata.
//
// Takes artefactID (string) which is the unique identifier for the artefact.
// Takes sourcePath (string) which is the location of the source file.
// Takes variants ([]registry_dto.Variant) which lists the actual variants.
// Takes desiredProfiles ([]registry_dto.NamedProfile) which lists the target
// profiles.
// Takes isNewArtefact (bool) which is true when creating a new artefact.
// Takes existingArtefact (*registry_dto.ArtefactMeta) which provides the
// previous metadata when updating an existing artefact.
//
// Returns *registry_dto.ArtefactMeta which is the built metadata with a
// computed status.
func buildArtefactMeta(
	artefactID, sourcePath string,
	variants []registry_dto.Variant,
	desiredProfiles []registry_dto.NamedProfile,
	isNewArtefact bool,
	existingArtefact *registry_dto.ArtefactMeta,
) *registry_dto.ArtefactMeta {
	metaNow := time.Now().UTC()
	createdAt := metaNow
	if !isNewArtefact {
		createdAt = existingArtefact.CreatedAt
	}
	meta := &registry_dto.ArtefactMeta{
		CreatedAt:       createdAt,
		UpdatedAt:       metaNow,
		DesiredProfiles: desiredProfiles,
		ID:              artefactID,
		SourcePath:      sourcePath,
		ActualVariants:  variants,
		Status:          registry_dto.VariantStatusPending,
	}
	meta.Status = meta.ComputeStatus()
	return meta
}

// profilesMatch checks if two slices of NamedProfile contain the same profiles.
//
// Profiles are considered matching if they have the same names. This is
// sufficient because profile parameters are deterministically generated from
// source attributes.
//
// Takes existing ([]registry_dto.NamedProfile) which contains the current
// profiles.
// Takes incoming ([]registry_dto.NamedProfile) which contains the new profiles.
//
// Returns bool which is true when both slices contain the same profile names.
func profilesMatch(existing, incoming []registry_dto.NamedProfile) bool {
	if len(existing) != len(incoming) {
		return false
	}

	existingNames := make(map[string]struct{}, len(existing))
	for i := range existing {
		existingNames[existing[i].Name] = struct{}{}
	}

	for i := range incoming {
		if _, found := existingNames[incoming[i].Name]; !found {
			return false
		}
	}

	return true
}

// detectMimeType returns the MIME type for a file path.
//
// Takes sourcePath (string) which is the path to the file.
//
// Returns string which is the MIME type for the file extension, or
// "application/octet-stream" when the extension is not known.
func detectMimeType(sourcePath string) string {
	mimeType := mime.TypeByExtension(filepath.Ext(sourcePath))
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

// uploadTempBlob uploads source data to a temporary location and computes
// its hash.
//
// Takes blobStore (BlobStore) which provides storage for the blob data.
// Takes sourceData (io.Reader) which supplies the data to upload.
// Takes sourcePath (string) which provides the original file path for
// extension extraction.
// Takes mimeType (string) which specifies the content type of the blob.
//
// Returns *blobUploadResult which contains the temporary key, hash, size,
// final key, and MIME type.
// Returns error when the blob cannot be saved to the store.
func uploadTempBlob(
	ctx context.Context,
	blobStore BlobStore,
	sourceData io.Reader,
	sourcePath string,
	mimeType string,
) (*blobUploadResult, error) {
	tempKey := "tmp/" + uuid.NewString()

	sha256Hasher, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		return nil, errors.New("sha256Pool returned unexpected type")
	}
	sha384Hasher, ok := sha384Pool.Get().(hash.Hash)
	if !ok {
		sha256Pool.Put(sha256Hasher)
		return nil, errors.New("sha384Pool returned unexpected type")
	}
	sha256Hasher.Reset()
	sha384Hasher.Reset()

	counter := &writeCounter{}
	teeReader := io.TeeReader(sourceData, io.MultiWriter(sha256Hasher, sha384Hasher, counter))

	if err := blobStore.Put(ctx, tempKey, teeReader); err != nil {
		sha256Pool.Put(sha256Hasher)
		sha384Pool.Put(sha384Hasher)
		return nil, fmt.Errorf("failed to save source blob: %w", err)
	}

	contentHash := fmt.Sprintf("%x", sha256Hasher.Sum(nil))
	sriHash := "sha384-" + base64.StdEncoding.EncodeToString(sha384Hasher.Sum(nil))

	sha256Pool.Put(sha256Hasher)
	sha384Pool.Put(sha384Hasher)

	return &blobUploadResult{
		tempKey:  tempKey,
		hash:     contentHash,
		sriHash:  sriHash,
		size:     counter.total,
		finalKey: fmt.Sprintf("source/%s%s", contentHash, filepath.Ext(sourcePath)),
		mimeType: mimeType,
	}, nil
}

// finaliseBlobStorage moves a temporary blob to its final storage location.
// If a copy already exists, it removes the temporary blob instead.
//
// Takes blobStore (BlobStore) which provides blob storage operations.
// Takes tempKey (string) which is the temporary storage key for the blob.
// Takes finalKey (string) which is the target storage key for the blob.
//
// Returns error when the blob cannot be moved to its final location.
func finaliseBlobStorage(
	ctx context.Context,
	blobStore BlobStore,
	tempKey, finalKey string,
) error {
	ctx, l := logger_domain.From(ctx, log)
	blobExists, err := blobStore.Exists(ctx, finalKey)
	if err != nil {
		l.Warn("Failed to check blob existence, will attempt upload",
			logger_domain.Error(err), logger_domain.String(logKeyStorageKey, finalKey))
		blobExists = false
	}

	if blobExists {
		l.Trace("Blob already exists (deduplication), reusing existing blob",
			logger_domain.String(logKeyStorageKey, finalKey))
		_ = blobStore.Delete(ctx, tempKey)
		registryServiceBlobDeduplicationHitCount.Add(ctx, 1)
		return nil
	}

	l.Trace("Blob doesn't exist, moving from temp to final location",
		logger_domain.String("from", tempKey), logger_domain.String("to", finalKey))
	if err := blobStore.Rename(ctx, tempKey, finalKey); err != nil {
		_ = blobStore.Delete(ctx, tempKey)
		return fmt.Errorf("failed to rename temp blob: %w", err)
	}
	return nil
}

// buildSourceVariant creates a source variant from blob upload results.
//
// Takes upload (*blobUploadResult) which holds the blob upload details.
// Takes storageBackendID (string) which names the storage backend.
//
// Returns registry_dto.Variant which is the new source variant ready for use.
func buildSourceVariant(upload *blobUploadResult, storageBackendID string) registry_dto.Variant {
	var tags registry_dto.Tags
	tags.Set(registry_dto.TagType, logKeySource)
	tags.Set(registry_dto.TagHash, upload.hash)

	return registry_dto.Variant{
		CreatedAt:        time.Now().UTC(),
		MetadataTags:     tags,
		VariantID:        logKeySource,
		StorageBackendID: storageBackendID,
		StorageKey:       upload.finalKey,
		MimeType:         upload.mimeType,
		Status:           registry_dto.VariantStatusReady,
		SizeBytes:        upload.size,
		ContentHash:      upload.hash,
		SRIHash:          upload.sriHash,
		Chunks:           []registry_dto.VariantChunk{},
	}
}

// applyVariantInvalidation checks existing variants and marks stale ones.
//
// Takes existingArtefact (*registry_dto.ArtefactMeta) which holds the variants
// to check.
// Takes newSourceVariant (*registry_dto.Variant) which is the new source
// variant that starts the check.
//
// Returns []registry_dto.Variant which contains the new source variant
// followed by all existing variants, with stale ones marked.
func applyVariantInvalidation(
	ctx context.Context,
	existingArtefact *registry_dto.ArtefactMeta,
	newSourceVariant *registry_dto.Variant,
) []registry_dto.Variant {
	ctx, l := logger_domain.From(ctx, log)
	finalVariants := []registry_dto.Variant{*newSourceVariant}
	dependencyMap := BuildDependencyMap(existingArtefact.DesiredProfiles)
	invalidatedIDs := getInvalidatedVariants(dependencyMap, map[string]struct{}{"source": {}})

	var invalidatedCount int64
	for i := range existingArtefact.ActualVariants {
		oldVariant := &existingArtefact.ActualVariants[i]
		if oldVariant.VariantID == "source" {
			continue
		}
		if _, isInvalidated := invalidatedIDs[oldVariant.VariantID]; isInvalidated {
			l.Trace("Marking variant as STALE", logger_domain.String(logKeyVariantID, oldVariant.VariantID))
			oldVariant.Status = registry_dto.VariantStatusStale
			invalidatedCount++
		}
		finalVariants = append(finalVariants, *oldVariant)
	}

	if invalidatedCount > 0 {
		registryServiceVariantInvalidationCount.Add(ctx, invalidatedCount)
		l.Trace("Cascading invalidation complete", logger_domain.Int64("invalidatedCount", invalidatedCount))
	}
	return finalVariants
}

// getOldSourceStorageKey returns the storage key of the existing source
// variant if one exists.
//
// Takes existingArtefact (*registry_dto.ArtefactMeta) which holds the artefact
// metadata to search for a source variant.
// Takes isNewArtefact (bool) which indicates whether this is a new artefact.
//
// Returns string which is the storage key of the source variant, or an empty
// string if not found or if this is a new artefact.
func getOldSourceStorageKey(existingArtefact *registry_dto.ArtefactMeta, isNewArtefact bool) string {
	if isNewArtefact || existingArtefact == nil {
		return ""
	}
	oldSourceVariant := findVariantByID(existingArtefact.ActualVariants, "source")
	if oldSourceVariant == nil {
		return ""
	}
	return oldSourceVariant.StorageKey
}

// prepareVariantList builds a new variant list and finds any existing variant
// that will be replaced.
//
// Takes existingVariants ([]registry_dto.Variant) which contains the current
// list of variants.
// Takes newVariant (*registry_dto.Variant) which is the variant to add or
// replace.
//
// Returns []registry_dto.Variant which is the updated list with the new variant
// added.
// Returns variantReplacementInfo which holds details of any replaced variant.
func prepareVariantList(
	ctx context.Context,
	existingVariants []registry_dto.Variant,
	newVariant *registry_dto.Variant,
) ([]registry_dto.Variant, variantReplacementInfo) {
	ctx, l := logger_domain.From(ctx, log)
	var info variantReplacementInfo
	finalVariants := make([]registry_dto.Variant, 0, len(existingVariants)+1)

	for i := range existingVariants {
		v := &existingVariants[i]
		if v.VariantID == newVariant.VariantID {
			l.Trace("Found existing variant with same ID, will replace it",
				logger_domain.String("existingStorageKey", v.StorageKey),
				logger_domain.String("newStorageKey", newVariant.StorageKey))
			info.found = true
			info.oldStorageKey = v.StorageKey
			info.oldChunks = v.Chunks
			info.backendID = v.StorageBackendID
		} else {
			finalVariants = append(finalVariants, *v)
		}
	}

	newVariant.Status = registry_dto.VariantStatusReady
	finalVariants = append(finalVariants, *newVariant)
	return finalVariants, info
}

// getInvalidatedVariants finds all variants that depend on invalid variants
// using a breadth-first search.
//
// Takes dependencyMap (map[string]string) which maps variant names to the
// identifiers they depend on.
// Takes initiallyInvalidated (map[string]struct{}) which contains variants
// that are already known to be invalid.
//
// Returns map[string]struct{} which contains all invalid variants, including
// both the initial set and any that depend on them.
func getInvalidatedVariants(dependencyMap map[string]string, initiallyInvalidated map[string]struct{}) map[string]struct{} {
	invalidatedSet := make(map[string]struct{})
	queue := make([]string, 0, len(initiallyInvalidated))

	for id := range initiallyInvalidated {
		if _, exists := invalidatedSet[id]; !exists {
			invalidatedSet[id] = struct{}{}
			queue = append(queue, id)
		}
	}

	for i := 0; i < len(queue); i++ {
		invalidatedDep := queue[i]

		for profileName, depID := range dependencyMap {
			if depID == invalidatedDep {
				if _, alreadyInvalidated := invalidatedSet[profileName]; !alreadyInvalidated {
					invalidatedSet[profileName] = struct{}{}
					queue = append(queue, profileName)
				}
			}
		}
	}

	return invalidatedSet
}

// findVariantByID finds a variant by its ID in the given slice.
//
// Takes variants ([]registry_dto.Variant) which is the slice to search.
// Takes id (string) which is the ID to match.
//
// Returns *registry_dto.Variant which is the matching variant, or nil if not
// found.
func findVariantByID(variants []registry_dto.Variant, id string) *registry_dto.Variant {
	for i := range variants {
		if variants[i].VariantID == id {
			return &variants[i]
		}
	}
	return nil
}
