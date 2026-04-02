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

package registry_blob_adapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"

	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

// BlobStoreAdapter implements registry_domain.BlobStore by wrapping a
// StorageProviderPort. It binds the provider to a specific repository and
// provides a simpler interface.
type BlobStoreAdapter struct {
	// provider stores and retrieves blobs through Put, Get, Remove, and Rename.
	provider storage_domain.StorageProviderPort

	// repository is the name of the storage repository for blob operations.
	repository string
}

var _ registry_domain.BlobStore = (*BlobStoreAdapter)(nil)
var _ provider_domain.ProviderMetadata = (*BlobStoreAdapter)(nil)

// Config holds settings for creating a BlobStoreAdapter.
type Config struct {
	// Provider is the storage backend that this adapter wraps; nil is not allowed.
	Provider storage_domain.StorageProviderPort

	// Repository is the name of the storage bucket for all operations.
	Repository string
}

// NewBlobStoreAdapter creates a new adapter that wraps a StorageProviderPort.
//
// Takes config (Config) which specifies the provider, repository, and logger.
//
// Returns *BlobStoreAdapter which is ready to handle blob storage operations.
// Returns error when the provider is nil.
func NewBlobStoreAdapter(config Config) (*BlobStoreAdapter, error) {
	if config.Provider == nil {
		return nil, errors.New("provider is required")
	}

	return &BlobStoreAdapter{
		provider:   config.Provider,
		repository: config.Repository,
	}, nil
}

// GetProviderType returns the type of the underlying storage provider.
//
// Returns string which is the provider type, or "unknown" if the provider
// does not implement ProviderMetadata.
func (a *BlobStoreAdapter) GetProviderType() string {
	if meta, ok := a.provider.(provider_domain.ProviderMetadata); ok {
		return meta.GetProviderType()
	}
	return "unknown"
}

// GetProviderMetadata returns metadata about this blob store adapter,
// including the repository name and any metadata from the underlying provider.
//
// Returns map[string]any which describes the adapter configuration.
func (a *BlobStoreAdapter) GetProviderMetadata() map[string]any {
	result := map[string]any{
		"repository": a.repository,
	}
	if meta, ok := a.provider.(provider_domain.ProviderMetadata); ok {
		maps.Copy(result, meta.GetProviderMetadata())
	}
	return result
}

// Put stores data at the given key.
//
// Takes key (string) which identifies where to store the data.
// Takes data (io.Reader) which provides the content to store.
//
// Returns error when the storage provider fails to write the data.
func (a *BlobStoreAdapter) Put(ctx context.Context, key string, data io.Reader) error {
	params := &storage_dto.PutParams{
		Repository:           a.repository,
		Key:                  key,
		Reader:               data,
		Size:                 -1,
		ContentType:          "application/octet-stream",
		Metadata:             nil,
		TransformConfig:      nil,
		MultipartConfig:      nil,
		HashAlgorithm:        "",
		ExpectedHash:         "",
		UseContentAddressing: false,
	}

	if err := a.provider.Put(ctx, params); err != nil {
		return fmt.Errorf("storage provider put failed: %w", err)
	}
	return nil
}

// Get retrieves data for the given key.
//
// Takes key (string) which identifies the blob to retrieve.
//
// Returns io.ReadCloser which provides access to the blob data.
// Returns error when the blob is not found or the storage provider fails.
func (a *BlobStoreAdapter) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	params := storage_dto.GetParams{
		Repository:      a.repository,
		Key:             key,
		ByteRange:       nil,
		TransformConfig: nil,
	}

	reader, err := a.provider.Get(ctx, params)
	if err != nil {
		if isNotFoundError(err) {
			return nil, registry_domain.ErrBlobNotFound
		}
		return nil, fmt.Errorf("storage provider get failed: %w", err)
	}
	return reader, nil
}

// RangeGet retrieves a range of bytes from the given key.
//
// Takes key (string) which identifies the blob to read from.
// Takes offset (int64) which specifies the starting byte position.
// Takes length (int64) which specifies the number of bytes to read.
//
// Returns io.ReadCloser which provides access to the requested byte range.
// Returns error when offset is negative, length is not positive, or the blob
// is not found.
func (a *BlobStoreAdapter) RangeGet(ctx context.Context, key string, offset int64, length int64) (io.ReadCloser, error) {
	if offset < 0 || length <= 0 {
		return nil, registry_domain.ErrRangeNotSatisfiable
	}

	params := storage_dto.GetParams{
		Repository: a.repository,
		Key:        key,
		ByteRange: &storage_dto.ByteRange{
			Start: offset,
			End:   offset + length - 1,
		},
		TransformConfig: nil,
	}

	reader, err := a.provider.Get(ctx, params)
	if err != nil {
		if isNotFoundError(err) {
			return nil, registry_domain.ErrBlobNotFound
		}
		return nil, fmt.Errorf("storage provider range get failed: %w", err)
	}
	return reader, nil
}

// Delete removes the data at the given key.
//
// Takes key (string) which identifies the data to remove.
//
// Returns error when the storage provider fails to remove the data.
func (a *BlobStoreAdapter) Delete(ctx context.Context, key string) error {
	params := storage_dto.GetParams{
		Repository:      a.repository,
		Key:             key,
		ByteRange:       nil,
		TransformConfig: nil,
	}

	if err := a.provider.Remove(ctx, params); err != nil {
		return fmt.Errorf("storage provider remove failed: %w", err)
	}
	return nil
}

// Rename moves data from a temporary location to its final location.
//
// Takes tempKey (string) which is the temporary storage location.
// Takes key (string) which is the final storage location.
//
// Returns error when the storage provider rename fails.
func (a *BlobStoreAdapter) Rename(ctx context.Context, tempKey string, key string) error {
	if err := a.provider.Rename(ctx, a.repository, tempKey, key); err != nil {
		return fmt.Errorf("storage provider rename failed: %w", err)
	}
	return nil
}

// Exists checks if data exists at the given key.
//
// Takes key (string) which specifies the storage location to check.
//
// Returns bool which is true if data exists at the key.
// Returns error when the storage provider check fails.
func (a *BlobStoreAdapter) Exists(ctx context.Context, key string) (bool, error) {
	params := storage_dto.GetParams{
		Repository:      a.repository,
		Key:             key,
		ByteRange:       nil,
		TransformConfig: nil,
	}

	exists, err := a.provider.Exists(ctx, params)
	if err != nil {
		return false, fmt.Errorf("storage provider exists failed: %w", err)
	}
	return exists, nil
}

// keyLister is an optional interface that storage providers can implement to
// support key enumeration for garbage collection.
type keyLister interface {
	// ListKeys returns all storage keys in the given repository.
	ListKeys(ctx context.Context, repository string) ([]string, error)
}

// ListKeys returns all storage keys present in this blob store by delegating
// to the underlying provider if it supports key listing.
//
// Returns []string which contains all keys in the store.
// Returns error when the provider does not support listing or the operation
// fails.
func (a *BlobStoreAdapter) ListKeys(ctx context.Context) ([]string, error) {
	lister, ok := a.provider.(keyLister)
	if !ok {
		return nil, errors.New("storage provider does not support key listing")
	}
	return lister.ListKeys(ctx, a.repository)
}

// isNotFoundError checks if an error shows that the object was not found.
// It handles the different "not found" error patterns from various storage
// providers.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error shows a not found condition.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, os.ErrNotExist) {
		return true
	}

	errString := err.Error()
	notFoundPatterns := []string{
		"not found",
		"NoSuchKey",
		"NotFound",
		"does not exist",
		"ErrObjectNotExist",
	}
	for _, pattern := range notFoundPatterns {
		if strings.Contains(errString, pattern) {
			return true
		}
	}

	return false
}
