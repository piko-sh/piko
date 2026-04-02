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
	"errors"
	"fmt"
	"io"
	"slices"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

// GetVariantData retrieves the blob data for a specific variant.
//
// Takes variant (*registry_dto.Variant) which identifies the variant to fetch.
//
// Returns io.ReadCloser which provides access to the variant's blob data.
// Returns error when the variant has empty storage fields, the storage backend
// is not found, or the blob cannot be retrieved.
func (s *registryService) GetVariantData(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.GetVariantData",
		logger_domain.String(logKeyVariantID, variant.VariantID),
		logger_domain.String(logKeyStorageBackendID, variant.StorageBackendID),
		logger_domain.String(logKeyStorageKey, variant.StorageKey),
	)
	defer span.End()

	if variant.StorageBackendID == "" {
		err := fmt.Errorf("variant %s has empty StorageBackendID", variant.VariantID)
		l.ReportError(span, err, "Invalid variant: empty StorageBackendID")
		return nil, fmt.Errorf("getting variant data: %w", err)
	}
	if variant.StorageKey == "" {
		err := fmt.Errorf("variant %s has empty StorageKey", variant.VariantID)
		l.ReportError(span, err, "Invalid variant: empty StorageKey")
		return nil, fmt.Errorf("getting variant data: %w", err)
	}

	l.Trace("Looking up blob store for variant")
	store, ok := s.blobStores[variant.StorageBackendID]
	if !ok {
		l.ReportError(span, errors.New(errMessageStorageBackendNotFound), errStorageBackendNotFound,
			logger_domain.String(logKeyStorageBackendID, variant.StorageBackendID),
			logger_domain.String(logKeyVariantID, variant.VariantID))
		return nil, fmt.Errorf("storage backend '%s' not found for variant %s", variant.StorageBackendID, variant.VariantID)
	}

	l.Trace("Retrieving variant data from blob store")
	data, err := store.Get(ctx, variant.StorageKey)
	if err != nil {
		if errors.Is(err, ErrBlobNotFound) {
			l.Warn("Variant blob not found",
				logger_domain.String(logKeyStorageKey, variant.StorageKey))
		} else {
			l.ReportError(span, err, "Failed to retrieve variant data")
		}
		return nil, fmt.Errorf("retrieving variant %q data: %w", variant.VariantID, err)
	}

	l.Trace("Retrieved variant data successfully")
	return data, nil
}

// GetVariantChunk retrieves the data for a specific chunk within a variant.
//
// Takes variant (*registry_dto.Variant) which identifies the variant containing
// the chunk.
// Takes chunkID (string) which specifies the unique identifier of the chunk to
// retrieve.
//
// Returns io.ReadCloser which provides access to the chunk data and must be
// closed by the caller.
// Returns error when the chunk is not found, the storage backend is missing,
// or the blob cannot be retrieved.
func (s *registryService) GetVariantChunk(ctx context.Context, variant *registry_dto.Variant, chunkID string) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.GetVariantChunk",
		logger_domain.String(logKeyVariantID, variant.VariantID),
		logger_domain.String(logKeyChunkID, chunkID),
		logger_domain.String(logKeyStorageBackendID, variant.StorageBackendID),
	)
	defer span.End()

	chunk, err := findAndValidateChunk(variant, chunkID)
	if err != nil {
		l.Warn("Chunk lookup failed", logger_domain.Error(err),
			logger_domain.String(logKeyChunkID, chunkID),
			logger_domain.String(logKeyVariantID, variant.VariantID))
		return nil, fmt.Errorf("finding chunk %q in variant %q: %w", chunkID, variant.VariantID, err)
	}

	store, ok := s.blobStores[chunk.StorageBackendID]
	if !ok {
		l.ReportError(span, errors.New(errMessageStorageBackendNotFound), errStorageBackendNotFound,
			logger_domain.String(logKeyStorageBackendID, chunk.StorageBackendID),
			logger_domain.String(logKeyChunkID, chunkID))
		return nil, fmt.Errorf("storage backend '%s' not found for chunk %s", chunk.StorageBackendID, chunkID)
	}

	l.Trace("Retrieving chunk data from blob store",
		logger_domain.String(logKeyStorageKey, chunk.StorageKey),
		logger_domain.Int64("sizeBytes", chunk.SizeBytes))
	data, err := store.Get(ctx, chunk.StorageKey)
	if err != nil {
		if errors.Is(err, ErrBlobNotFound) {
			l.Warn("Chunk blob not found", logger_domain.String(logKeyStorageKey, chunk.StorageKey))
		} else {
			l.ReportError(span, err, "Failed to retrieve chunk data")
		}
		return nil, fmt.Errorf("retrieving chunk %q data: %w", chunkID, err)
	}

	l.Trace("Retrieved chunk data successfully", logger_domain.Int("sequenceNumber", chunk.SequenceNumber))
	return data, nil
}

// GetVariantDataRange retrieves a specific byte range from a variant's data.
// Use it for HTTP Range requests and streaming large files.
//
// Takes variant (*registry_dto.Variant) which identifies the stored data.
// Takes offset (int64) which specifies the starting byte position.
// Takes length (int64) which specifies the number of bytes to read.
//
// Returns io.ReadCloser which provides access to the requested byte range.
// Returns error when the range is invalid, the storage backend is not found,
// or the blob cannot be retrieved.
func (s *registryService) GetVariantDataRange(ctx context.Context, variant *registry_dto.Variant, offset int64, length int64) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.GetVariantDataRange",
		logger_domain.String(logKeyVariantID, variant.VariantID),
		logger_domain.String(logKeyStorageBackendID, variant.StorageBackendID),
		logger_domain.String(logKeyStorageKey, variant.StorageKey),
		logger_domain.Int64("offset", offset),
		logger_domain.Int64("length", length),
	)
	defer span.End()

	if offset < 0 || length <= 0 {
		l.Warn("Invalid range parameters",
			logger_domain.Int64("offset", offset),
			logger_domain.Int64("length", length))
		return nil, ErrRangeNotSatisfiable
	}

	if variant.StorageBackendID == "" {
		err := fmt.Errorf("variant %s has empty StorageBackendID", variant.VariantID)
		l.ReportError(span, err, "Invalid variant: empty StorageBackendID")
		return nil, fmt.Errorf("getting variant data range: %w", err)
	}
	if variant.StorageKey == "" {
		err := fmt.Errorf("variant %s has empty StorageKey", variant.VariantID)
		l.ReportError(span, err, "Invalid variant: empty StorageKey")
		return nil, fmt.Errorf("getting variant data range: %w", err)
	}

	l.Trace("Looking up blob store for variant range request")
	store, ok := s.blobStores[variant.StorageBackendID]
	if !ok {
		l.ReportError(span, errors.New(errMessageStorageBackendNotFound), errStorageBackendNotFound,
			logger_domain.String(logKeyStorageBackendID, variant.StorageBackendID),
			logger_domain.String(logKeyVariantID, variant.VariantID))
		return nil, fmt.Errorf("storage backend '%s' not found for variant %s", variant.StorageBackendID, variant.VariantID)
	}

	l.Trace("Retrieving variant data range from blob store")
	data, err := store.RangeGet(ctx, variant.StorageKey, offset, length)
	if err != nil {
		if errors.Is(err, ErrBlobNotFound) {
			l.Warn("Variant blob not found for range request",
				logger_domain.String(logKeyStorageKey, variant.StorageKey))
		} else if errors.Is(err, ErrRangeNotSatisfiable) {
			l.Warn("Range not satisfiable",
				logger_domain.Int64("offset", offset),
				logger_domain.Int64("length", length))
		} else {
			l.ReportError(span, err, "Failed to retrieve variant data range")
		}
		return nil, fmt.Errorf("retrieving variant %q data range (offset=%d, length=%d): %w", variant.VariantID, offset, length, err)
	}

	l.Trace("Retrieved variant data range successfully")
	return data, nil
}

// GetBlobStore retrieves a blob storage backend by its ID.
//
// Takes backendID (string) which identifies the storage backend to retrieve.
//
// Returns BlobStore which provides access to the blob storage operations.
// Returns error when the specified backend ID is not configured.
func (s *registryService) GetBlobStore(backendID string) (BlobStore, error) {
	_, span, l := log.Span(context.Background(), "RegistryService.GetBlobStore",
		logger_domain.String("backendID", backendID),
	)
	defer span.End()

	l.Trace("Looking up blob store by ID")
	store, ok := s.blobStores[backendID]
	if !ok {
		l.ReportError(span, errors.New(errMessageStorageBackendNotFound), errStorageBackendNotFound,
			logger_domain.String("backendID", backendID))
		return nil, fmt.Errorf("storage backend '%s' not configured or found", backendID)
	}

	l.Trace("Blob store found successfully")
	return store, nil
}

// PopGCHints retrieves and removes garbage collection hints from the
// metadata store.
//
// Takes limit (int) which specifies the maximum number of hints to retrieve.
//
// Returns []registry_dto.GCHint which contains the retrieved hints.
// Returns error when the metadata store operation fails.
func (s *registryService) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.PopGCHints",
		logger_domain.Int("requestedLimit", limit),
	)
	defer span.End()

	l.Trace("Retrieving GC hints from metadata store")
	hints, err := s.metaStore.PopGCHints(ctx, limit)
	if err != nil {
		l.ReportError(span, err, "Failed to pop GC hints")
		return nil, fmt.Errorf("popping GC hints: %w", err)
	}

	if len(hints) == 0 {
		l.Trace("No GC hints available")
	} else {
		l.Trace("Retrieved GC hints successfully", logger_domain.Int("hintCount", len(hints)))
	}

	return hints, nil
}

// ListBlobStoreIDs returns the identifiers of all registered blob storage
// backends, sorted alphabetically.
//
// Returns []string which contains all backend IDs.
func (s *registryService) ListBlobStoreIDs() []string {
	ids := make([]string, 0, len(s.blobStores))
	for id := range s.blobStores {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// findAndValidateChunk finds a chunk by ID and checks its storage fields.
//
// Takes variant (*registry_dto.Variant) which contains the chunks to search.
// Takes chunkID (string) which identifies the chunk to find.
//
// Returns *registry_dto.VariantChunk which is the matching chunk if found.
// Returns error when the chunk is not found or has empty storage fields.
func findAndValidateChunk(variant *registry_dto.Variant, chunkID string) (*registry_dto.VariantChunk, error) {
	for i := range variant.Chunks {
		if variant.Chunks[i].ChunkID == chunkID {
			chunk := &variant.Chunks[i]
			if chunk.StorageBackendID == "" {
				return nil, fmt.Errorf("chunk %s has empty StorageBackendID", chunkID)
			}
			if chunk.StorageKey == "" {
				return nil, fmt.Errorf("chunk %s has empty StorageKey", chunkID)
			}
			return chunk, nil
		}
	}
	return nil, ErrChunkNotFound
}
