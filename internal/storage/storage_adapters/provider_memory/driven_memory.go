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

package provider_memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/contextaware"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// providerType is the stable, lowercase identifier reported for this provider.
	providerType = "memory"

	// keySeparator joins a repository and an object key into a single composite cache key. A
	// NUL byte is used because it cannot appear in a repository name or object key.
	keySeparator = "\x00"

	// entryOverheadBytes is the fixed allowance added to every object's footprint.
	entryOverheadBytes int64 = 160

	// metadataEntryOverheadBytes is the allowance charged per metadata pair.
	metadataEntryOverheadBytes int64 = 48
)

var (
	// ErrObjectNotFound is returned when a requested object is absent from the cache, either
	// because it was never written or because it has been evicted to stay within the byte
	// budget. Callers may match it with errors.Is.
	ErrObjectNotFound = errors.New("object not found")

	// ErrObjectTooLarge is returned by Put when a single object exceeds the configured byte
	// budget and therefore could never fit in the cache.
	ErrObjectTooLarge = errors.New("object exceeds the storage byte budget")

	// ErrPresignUnsupported is returned by the presign methods, which this provider cannot
	// satisfy because it has no external, addressable endpoint.
	ErrPresignUnsupported = errors.New("memory storage provider does not support presigned URLs")

	// ErrInvalidMaxBytes is returned by New when the configured byte budget is not positive.
	ErrInvalidMaxBytes = errors.New("maximum byte budget must be positive")

	// ErrNilReader is returned by Put when the parameters carry no reader.
	ErrNilReader = errors.New("reader cannot be nil for put operation")
)

// storedObject holds a single blob and its metadata inside the cache.
type storedObject struct {
	// lastModified is when this object was written or last replaced.
	lastModified time.Time

	// contentType is the MIME type recorded for the object.
	contentType string

	// metadata holds the custom key-value pairs stored alongside the object.
	metadata map[string]string

	// data holds the raw object bytes.
	data []byte
}

// Config holds the settings for constructing a memory storage provider.
type Config struct {
	// MaxBytes is the positive total byte budget for stored objects, beyond which writes
	// evict the least useful objects.
	MaxBytes int64
}

// Provider is an in-memory, byte-size-bounded storage provider backed by the cache
// hexagon's otter provider. It is safe for concurrent use.
type Provider struct {
	// cache holds the objects, bounded by weight (the byte length of each object's data).
	cache cache_domain.ProviderPort[string, *storedObject]

	// maxBytes is the configured byte budget, retained for validation and reporting.
	maxBytes int64
}

var (
	_ storage_domain.StorageProviderPort = (*Provider)(nil)

	_ provider_domain.ProviderMetadata = (*Provider)(nil)
)

// New constructs a memory storage provider from the supplied configuration.
//
// Takes config (Config) which specifies the byte budget for stored object data.
//
// Returns *Provider which is ready for use as a writable storage overlay.
// Returns error when the byte budget is not positive or the backing cache cannot be
// created.
func New(config Config) (*Provider, error) {
	if config.MaxBytes <= 0 {
		return nil, fmt.Errorf("memory storage provider byte budget %d is invalid: %w", config.MaxBytes, ErrInvalidMaxBytes)
	}

	cacheOptions := cache_dto.Options[string, *storedObject]{
		MaximumWeight: uint64(config.MaxBytes),
		Weigher: func(key string, value *storedObject) uint32 {
			return clampWeight(entryFootprint(key, value))
		},
	}

	cache, err := provider_otter.OtterProviderFactory(cacheOptions)
	if err != nil {
		return nil, fmt.Errorf("creating in-memory cache provider: %w", err)
	}

	return &Provider{
		cache:    cache,
		maxBytes: config.MaxBytes,
	}, nil
}

// GetProviderType returns the stable type identifier for this provider.
//
// Returns string which is always "memory".
func (*Provider) GetProviderType() string {
	return providerType
}

// GetProviderMetadata returns descriptive metadata about this provider for discovery and
// monitoring.
//
// Returns map[string]any which reports the provider type, that it is writable, the
// configured byte budget, and the current weighted size of the cache.
func (p *Provider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"type":         providerType,
		"writable":     true,
		"maxBytes":     p.maxBytes,
		"weightedSize": p.cache.WeightedSize(),
	}
}

// Put stores an object, reading its content from the supplied reader up to the byte
// budget.
//
// Reads are bounded by io.LimitReader and observe cancellation, so neither an unbounded
// nor a stalled source can hold the call open. An object whose size exceeds the
// configured budget is rejected because it could never fit, and a declared size beyond
// the budget is refused before any bytes are buffered.
//
// Takes params (*storage_dto.PutParams) which specifies the repository, key, reader,
// content type, and metadata for the object.
//
// Returns error when the reader is nil or fails, the context is cancelled, the object
// exceeds the byte budget, or the cache write fails.
func (p *Provider) Put(ctx context.Context, params *storage_dto.PutParams) error {
	if params.Reader == nil {
		return fmt.Errorf("storing object %q: %w", params.Key, ErrNilReader)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("storing object %q: %w", params.Key, err)
	}
	if params.Size > 0 && params.Size > p.maxBytes {
		return fmt.Errorf("object %q declares %d bytes against a budget of %d bytes: %w", params.Key, params.Size, p.maxBytes, ErrObjectTooLarge)
	}

	data, err := io.ReadAll(io.LimitReader(contextaware.NewReader(ctx, params.Reader), p.readLimit()))
	if err != nil {
		return fmt.Errorf("memory storage provider failed to read object %q: %w", params.Key, err)
	}

	key := compositeKey(params.Repository, params.Key)
	storedValue := &storedObject{
		lastModified: time.Now(),
		contentType:  params.ContentType,
		data:         slices.Clip(data),
		metadata:     maps.Clone(params.Metadata),
	}

	if err := p.checkFootprint(params.Key, key, storedValue); err != nil {
		return err
	}

	if err := p.cache.Set(ctx, key, storedValue); err != nil {
		return fmt.Errorf("memory storage provider failed to store object %q: %w", params.Key, err)
	}
	return nil
}

// Get retrieves an object as a readable stream, honouring any byte range in the
// parameters.
//
// Takes params (storage_dto.GetParams) which specifies the repository, key, and optional
// byte range.
//
// Returns io.ReadCloser which streams the object data.
// Returns error when the object is absent, has been evicted, or the cache read fails.
func (p *Provider) Get(ctx context.Context, params storage_dto.GetParams) (io.ReadCloser, error) {
	storedValue, found, err := p.cache.GetIfPresent(ctx, compositeKey(params.Repository, params.Key))
	if err != nil {
		return nil, fmt.Errorf("memory storage provider failed to read object %q: %w", params.Key, err)
	}
	if !found {
		return nil, fmt.Errorf("object %q in repository %q: %w", params.Key, params.Repository, ErrObjectNotFound)
	}

	data := storedValue.data
	if params.ByteRange != nil {
		data = sliceByteRange(data, params.ByteRange.Start, params.ByteRange.End)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

// Stat retrieves metadata for an object without streaming its content.
//
// Takes params (storage_dto.GetParams) which specifies the repository and key to query.
//
// Returns *storage_domain.ObjectInfo which describes the object.
// Returns error when the object is absent, has been evicted, or the cache read fails.
func (p *Provider) Stat(ctx context.Context, params storage_dto.GetParams) (*storage_domain.ObjectInfo, error) {
	storedValue, found, err := p.cache.GetIfPresent(ctx, compositeKey(params.Repository, params.Key))
	if err != nil {
		return nil, fmt.Errorf("memory storage provider failed to stat object %q: %w", params.Key, err)
	}
	if !found {
		return nil, fmt.Errorf("object %q in repository %q: %w", params.Key, params.Repository, ErrObjectNotFound)
	}

	return &storage_domain.ObjectInfo{
		LastModified: storedValue.lastModified,
		Metadata:     maps.Clone(storedValue.metadata),
		ContentType:  storedValue.contentType,
		ETag:         "",
		Size:         int64(len(storedValue.data)),
	}, nil
}

// Copy performs a copy within the same repository.
//
// Takes srcRepo (string) which identifies the repository holding both objects.
// Takes srcKey (string) which is the source object key.
// Takes dstKey (string) which is the destination object key.
//
// Returns error when the source object is absent, has been evicted, or the cache write
// fails.
func (p *Provider) Copy(ctx context.Context, srcRepo string, srcKey, dstKey string) error {
	return p.copyInternal(ctx, srcRepo, srcKey, srcRepo, dstKey)
}

// CopyToAnotherRepository performs a copy between two repositories.
//
// Takes srcRepo (string) which identifies the source repository.
// Takes srcKey (string) which is the source object key.
// Takes dstRepo (string) which identifies the destination repository.
// Takes dstKey (string) which is the destination object key.
//
// Returns error when the source object is absent, has been evicted, or the cache write
// fails.
func (p *Provider) CopyToAnotherRepository(ctx context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error {
	return p.copyInternal(ctx, srcRepo, srcKey, dstRepo, dstKey)
}

// Remove deletes an object. It is idempotent: removing an absent object is not an error.
//
// Takes params (storage_dto.GetParams) which identifies the object to remove.
//
// Returns error when the underlying cache invalidation fails, such as when the context
// has been cancelled.
func (p *Provider) Remove(ctx context.Context, params storage_dto.GetParams) error {
	if err := p.cache.Invalidate(ctx, compositeKey(params.Repository, params.Key)); err != nil {
		return fmt.Errorf("memory storage provider failed to remove object %q: %w", params.Key, err)
	}
	return nil
}

// Rename moves an object from one key to another within the same repository, supporting
// the temp-to-final atomic write pattern used by the content-addressed registry.
//
// The move is a set of the destination followed by an invalidation of the source. Because
// the registry is content-addressed and each key is written by a single writer, no extra
// locking is required.
//
// Takes repo (string) which identifies the repository.
// Takes oldKey (string) which is the current object key.
// Takes newKey (string) which is the destination object key.
//
// Returns error when the source object is absent, has been evicted, the destination
// cannot fit the budget, or a cache operation fails.
func (p *Provider) Rename(ctx context.Context, repo string, oldKey, newKey string) error {
	oldCompositeKey := compositeKey(repo, oldKey)
	storedValue, found, err := p.cache.GetIfPresent(ctx, oldCompositeKey)
	if err != nil {
		return fmt.Errorf("rename failed to read source object %q: %w", oldKey, err)
	}
	if !found {
		return fmt.Errorf("rename source object %q in repository %q: %w", oldKey, repo, ErrObjectNotFound)
	}

	newCompositeKey := compositeKey(repo, newKey)
	if err := p.checkFootprint(newKey, newCompositeKey, storedValue); err != nil {
		return fmt.Errorf("renaming %q to %q: %w", oldKey, newKey, err)
	}

	if err := p.cache.Set(ctx, newCompositeKey, storedValue); err != nil {
		return fmt.Errorf("rename failed to write destination object %q: %w", newKey, err)
	}
	if _, stored, checkErr := p.cache.GetIfPresent(ctx, newCompositeKey); checkErr != nil || !stored {
		return fmt.Errorf("rename destination object %q was not retained: %w", newKey, errors.Join(checkErr, ErrObjectNotFound))
	}
	if err := p.cache.Invalidate(ctx, oldCompositeKey); err != nil {
		return fmt.Errorf("rename failed to remove source object %q: %w", oldKey, err)
	}
	return nil
}

// Exists reports whether an object is present.
//
// Takes params (storage_dto.GetParams) which specifies the object to check.
//
// Returns bool which is true when the object is present, false otherwise.
// Returns error when the cache read fails.
func (p *Provider) Exists(ctx context.Context, params storage_dto.GetParams) (bool, error) {
	_, found, err := p.cache.GetIfPresent(ctx, compositeKey(params.Repository, params.Key))
	if err != nil {
		return false, fmt.Errorf("memory storage provider failed to check existence of object %q: %w", params.Key, err)
	}
	return found, nil
}

// GetHash returns the hex-encoded SHA-256 hash of an object's content.
//
// The registry is content-addressed, so the hash is a convenience computed on demand
// rather than stored.
//
// Takes params (storage_dto.GetParams) which specifies the object to hash.
//
// Returns string which is the hex-encoded SHA-256 hash of the object data.
// Returns error when the object is absent, has been evicted, or the cache read fails.
func (p *Provider) GetHash(ctx context.Context, params storage_dto.GetParams) (string, error) {
	storedValue, found, err := p.cache.GetIfPresent(ctx, compositeKey(params.Repository, params.Key))
	if err != nil {
		return "", fmt.Errorf("memory storage provider failed to hash object %q: %w", params.Key, err)
	}
	if !found {
		return "", fmt.Errorf("object %q in repository %q: %w", params.Key, params.Repository, ErrObjectNotFound)
	}

	hash := sha256.Sum256(storedValue.data)
	return hex.EncodeToString(hash[:]), nil
}

// PresignURL is unsupported because the provider has no external upload endpoint.
//
// Takes params (storage_dto.PresignParams) which is ignored.
//
// Returns string which is always empty.
// Returns error which is always ErrPresignUnsupported.
func (*Provider) PresignURL(_ context.Context, _ storage_dto.PresignParams) (string, error) {
	return "", fmt.Errorf("memory storage provider cannot presign uploads: %w", ErrPresignUnsupported)
}

// PresignDownloadURL is unsupported because the provider has no external download
// endpoint.
//
// Takes params (storage_dto.PresignDownloadParams) which is ignored.
//
// Returns string which is always empty.
// Returns error which is always ErrPresignUnsupported.
func (*Provider) PresignDownloadURL(_ context.Context, _ storage_dto.PresignDownloadParams) (string, error) {
	return "", fmt.Errorf("memory storage provider cannot presign downloads: %w", ErrPresignUnsupported)
}

// Close releases the resources held by the backing cache, including its background
// goroutines.
//
// Returns error when the underlying cache cannot be closed cleanly.
func (p *Provider) Close(ctx context.Context) error {
	if err := p.cache.Close(ctx); err != nil {
		return fmt.Errorf("closing in-memory cache provider: %w", err)
	}
	return nil
}

// SupportsMultipart reports whether the provider has a native multipart upload API.
//
// Returns bool which is always false.
func (*Provider) SupportsMultipart() bool {
	return false
}

// SupportsRetry reports whether the provider manages its own retries.
//
// Returns bool which is always false.
func (*Provider) SupportsRetry() bool {
	return false
}

// SupportsCircuitBreaking reports whether the provider has built-in circuit breaking.
//
// Returns bool which is always false.
func (*Provider) SupportsCircuitBreaking() bool {
	return false
}

// SupportsRateLimiting reports whether the provider handles rate limiting internally.
//
// Returns bool which is always false.
func (*Provider) SupportsRateLimiting() bool {
	return false
}

// SupportsBatchOperations reports whether the provider has native batch APIs.
//
// Returns bool which is always false; batch methods fall back to sequential operations.
func (*Provider) SupportsBatchOperations() bool {
	return false
}

// SupportsPresignedURLs reports whether the provider can generate native presigned URLs.
//
// Returns bool which is always false.
func (*Provider) SupportsPresignedURLs() bool {
	return false
}

// PutMany uploads several objects by applying Put to each in turn.
//
// Takes params (*storage_dto.PutManyParams) which specifies the repository and objects to
// upload.
//
// Returns *storage_dto.BatchResult which reports per-object success and failure.
// Returns error when the context is cancelled part way through; individual object
// failures are recorded in the result rather than returned.
func (p *Provider) PutMany(ctx context.Context, params *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	startTime := time.Now()
	result := &storage_dto.BatchResult{
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		TotalRequested:  len(params.Objects),
		TotalSuccessful: 0,
		TotalFailed:     0,
		ProcessingTime:  0,
	}

	for _, objectSpec := range params.Objects {
		if err := ctx.Err(); err != nil {
			result.ProcessingTime = time.Since(startTime)
			return result, fmt.Errorf("put-many cancelled after %d of %d objects: %w", result.TotalSuccessful+result.TotalFailed, result.TotalRequested, err)
		}
		putParams := &storage_dto.PutParams{
			Reader:               objectSpec.Reader,
			MultipartConfig:      nil,
			TransformConfig:      nil,
			Metadata:             nil,
			Key:                  objectSpec.Key,
			ContentType:          objectSpec.ContentType,
			HashAlgorithm:        "",
			ExpectedHash:         "",
			Repository:           params.Repository,
			Size:                 objectSpec.Size,
			UseContentAddressing: false,
		}
		if err := p.Put(ctx, putParams); err != nil {
			result.FailedKeys = append(result.FailedKeys, storage_dto.BatchFailure{
				Key:       objectSpec.Key,
				Error:     err.Error(),
				ErrorCode: "",
				Retryable: false,
			})
			result.TotalFailed++
			continue
		}
		result.SuccessfulKeys = append(result.SuccessfulKeys, objectSpec.Key)
		result.TotalSuccessful++
	}

	result.ProcessingTime = time.Since(startTime)
	return result, nil
}

// RemoveMany deletes several objects by applying Remove to each in turn. Because Remove
// is idempotent, every key succeeds unless the underlying cache invalidation fails.
//
// Takes params (storage_dto.RemoveManyParams) which specifies the repository and keys to
// delete.
//
// Returns *storage_dto.BatchResult which reports per-object success and failure.
// Returns error when the context is cancelled part way through; individual object
// failures are recorded in the result rather than returned.
func (p *Provider) RemoveMany(ctx context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	startTime := time.Now()
	result := &storage_dto.BatchResult{
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		TotalRequested:  len(params.Keys),
		TotalSuccessful: 0,
		TotalFailed:     0,
		ProcessingTime:  0,
	}

	for _, key := range params.Keys {
		if err := ctx.Err(); err != nil {
			result.ProcessingTime = time.Since(startTime)
			return result, fmt.Errorf("remove-many cancelled after %d of %d objects: %w", result.TotalSuccessful+result.TotalFailed, result.TotalRequested, err)
		}
		removeParams := storage_dto.GetParams{
			ByteRange:       nil,
			TransformConfig: nil,
			Key:             key,
			Repository:      params.Repository,
		}
		if err := p.Remove(ctx, removeParams); err != nil {
			result.FailedKeys = append(result.FailedKeys, storage_dto.BatchFailure{
				Key:       key,
				Error:     err.Error(),
				ErrorCode: "",
				Retryable: false,
			})
			result.TotalFailed++
			continue
		}
		result.SuccessfulKeys = append(result.SuccessfulKeys, key)
		result.TotalSuccessful++
	}

	result.ProcessingTime = time.Since(startTime)
	return result, nil
}

// checkFootprint rejects an object that cannot be admitted within the byte budget.
//
// The backing cache silently discards an entry whose weight exceeds the maximum, so every
// write path checks the footprint first rather than reporting success for an object that
// was never stored.
//
// Takes objectKey (string) which is the caller-facing key used in the error message.
// Takes key (string) which is the composite cache key charged to the object.
// Takes value (*storedObject) which is the object about to be written.
//
// Returns error wrapping ErrObjectTooLarge when the object cannot fit the budget.
func (p *Provider) checkFootprint(objectKey, key string, value *storedObject) error {
	footprint := entryFootprint(key, value)
	if footprint > p.maxBytes || footprint > math.MaxUint32 {
		return fmt.Errorf("object %q of %d bytes cannot fit budget of %d bytes: %w", objectKey, footprint, p.maxBytes, ErrObjectTooLarge)
	}
	return nil
}

// copyInternal stores a fresh copy of the source object at the destination.
//
// Takes sourceRepository (string) which holds the source object.
// Takes sourceKey (string) which is the source object key.
// Takes destinationRepository (string) which receives the copy.
// Takes destinationKey (string) which is the destination object key.
//
// Returns error when the source object is absent, has been evicted, or the cache write
// fails.
func (p *Provider) copyInternal(ctx context.Context, sourceRepository, sourceKey, destinationRepository, destinationKey string) error {
	storedValue, found, err := p.cache.GetIfPresent(ctx, compositeKey(sourceRepository, sourceKey))
	if err != nil {
		return fmt.Errorf("copy failed to read source object %q: %w", sourceKey, err)
	}
	if !found {
		return fmt.Errorf("copy source object %q in repository %q: %w", sourceKey, sourceRepository, ErrObjectNotFound)
	}

	destinationValue := &storedObject{
		lastModified: time.Now(),
		contentType:  storedValue.contentType,
		metadata:     maps.Clone(storedValue.metadata),
		data:         bytes.Clone(storedValue.data),
	}

	destinationCompositeKey := compositeKey(destinationRepository, destinationKey)
	if err := p.checkFootprint(destinationKey, destinationCompositeKey, destinationValue); err != nil {
		return fmt.Errorf("copying %q to %q: %w", sourceKey, destinationKey, err)
	}

	if err := p.cache.Set(ctx, destinationCompositeKey, destinationValue); err != nil {
		return fmt.Errorf("copy failed to write destination object %q: %w", destinationKey, err)
	}
	return nil
}

// readLimit reports the upper bound for reading an object's data. It is one byte beyond
// the budget so an over-budget object is detected, guarded against overflow when the
// budget is the maximum int64.
//
// Returns int64 which is the maximum number of bytes Put reads from a source.
func (p *Provider) readLimit() int64 {
	if p.maxBytes == math.MaxInt64 {
		return math.MaxInt64
	}
	return p.maxBytes + 1
}

// compositeKey joins a repository and object key into a single cache key.
//
// Takes repository (string) which namespaces the key.
// Takes key (string) which is the object key.
//
// Returns string which is the repository and key separated by a NUL byte.
func compositeKey(repository, key string) string {
	return repository + keySeparator + key
}

// entryFootprint reports a stored object's full byte cost against the budget.
//
// It charges the retained data capacity, composite key, content type, and metadata plus a
// fixed per-entry overhead, so the byte budget stays an honest bound and neither a
// small-data object carrying large keys or metadata nor a buffer holding spare capacity
// can escape it.
//
// Takes key (string) which is the composite cache key charged to the object.
// Takes value (*storedObject) which is the object whose cost is measured.
//
// Returns int64 which is the object's total footprint in bytes.
func entryFootprint(key string, value *storedObject) int64 {
	total := entryOverheadBytes + int64(len(key)) + int64(len(value.contentType)) + int64(cap(value.data))
	for metadataKey, metadataValue := range value.metadata {
		total += metadataEntryOverheadBytes + int64(len(metadataKey)) + int64(len(metadataValue))
	}
	return total
}

// clampWeight reduces a footprint to the uint32 the cache weigher requires, capping at
// math.MaxUint32. Put rejects any object whose footprint exceeds the budget or the uint32
// ceiling, so the clamp is only ever reached for a value that has already been refused.
//
// Takes footprint (int64) which is the object's full byte footprint.
//
// Returns uint32 which is the footprint capped at math.MaxUint32.
func clampWeight(footprint int64) uint32 {
	return safeconv.Int64ToUint32(footprint)
}

// sliceByteRange returns the sub-slice of data covered by the inclusive range, clamping
// the bounds exactly as the reference providers do. An end of -1 means the end of the
// data.
//
// Takes data ([]byte) which is the full object content.
// Takes start (int64) which is the inclusive start offset.
// Takes end (int64) which is the inclusive end offset, or -1 for the end of the data.
//
// Returns []byte which is the requested range, empty when the range is degenerate.
func sliceByteRange(data []byte, start, end int64) []byte {
	if start < 0 || start > int64(len(data)) {
		start = int64(len(data))
	}
	if end == -1 || end >= int64(len(data)) {
		end = int64(len(data)) - 1
	}
	if end < start {
		return []byte{}
	}
	return data[start : end+1]
}
