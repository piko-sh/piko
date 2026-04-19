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

package storage_domain

import (
	"context"
	"io"
	"time"

	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

// Service is the primary port for the storage hexagon, implementing
// io.Closer, storage_domain.Service, and wdk/storage.Service.
//
// It defines the complete set of storage capabilities available to the rest
// of the application. Its method signatures use Data Transfer Objects (DTOs)
// to encapsulate parameters, which allows for future extension without
// breaking changes.
type Service interface {
	// NewUpload returns a builder for uploading new objects.
	//
	// Takes reader (io.Reader) which provides the data to upload.
	//
	// Returns *UploadBuilder which configures and executes the upload.
	NewUpload(reader io.Reader) *UploadBuilder

	// NewRequest returns a builder for operating on existing objects
	// (Get, Stat, Remove, Hash).
	//
	// Takes repo (string) which identifies the repository.
	// Takes key (string) which identifies the object within the repository.
	//
	// Returns *RequestBuilder which provides methods for object operations.
	NewRequest(repo string, key string) *RequestBuilder

	// RegisterProvider adds a new storage provider to the service.
	//
	// Takes ctx (context.Context) for cancellation and logging propagation.
	// Takes name (string) which identifies the provider
	// (e.g. "s3", "gcs", "disk").
	// Takes provider (StorageProviderPort) which
	// implements the storage operations.
	//
	// Returns error when the provider cannot be registered.
	RegisterProvider(ctx context.Context, name string, provider StorageProviderPort) error

	// SetDefaultProvider sets the provider to be used when no specific provider
	// is named in a call.
	//
	// Takes name (string) which identifies the provider to use as the default.
	//
	// Returns error when the named provider does not exist.
	SetDefaultProvider(name string) error

	// GetProviders returns a sorted list of all registered provider names.
	//
	// Takes ctx (context.Context) for cancellation and logging propagation.
	GetProviders(ctx context.Context) []string

	// HasProvider checks whether a provider with the given name is registered.
	//
	// Takes name (string) which is the provider name to look for.
	//
	// Returns bool which is true if the provider exists, false otherwise.
	HasProvider(name string) bool

	// ListProviders returns detailed information about all registered providers.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns []provider_domain.ProviderInfo which contains provider metadata,
	// health status, and capabilities.
	ListProviders(ctx context.Context) []provider_domain.ProviderInfo

	// RegisterTransformer adds a stream transformer for compression or
	// encryption.
	//
	// Takes ctx (context.Context) for cancellation and logging propagation.
	// Takes transformer (StreamTransformerPort) which processes the stream.
	//
	// Returns error when registration fails.
	RegisterTransformer(ctx context.Context, transformer StreamTransformerPort) error

	// GetTransformers returns a sorted list of all registered transformer names.
	GetTransformers() []string

	// HasTransformer checks if a transformer with the given name is registered.
	//
	// Takes name (string) which is the transformer name to look for.
	//
	// Returns bool which is true if the transformer exists, false otherwise.
	HasTransformer(name string) bool

	// RegisterDispatcher registers and starts a storage dispatcher for
	// asynchronous operations.
	//
	// Takes ctx (context.Context) for cancellation and logging propagation.
	// Takes dispatcher (StorageDispatcherPort) which handles storage operations.
	//
	// Returns error when registration or startup fails.
	RegisterDispatcher(ctx context.Context, dispatcher StorageDispatcherPort) error

	// FlushDispatcher forces an immediate flush of all queued operations in the
	// dispatcher.
	//
	//
	// Returns error when the flush operation fails.
	FlushDispatcher(ctx context.Context) error

	// PutObject uploads an object to storage.
	//
	// Takes providerName (string) which identifies the storage provider to use.
	// Takes params (*storage_dto.PutParams) which contains the object data and
	// metadata. A pointer is used to avoid copying a large struct.
	//
	// Returns error when the upload fails.
	PutObject(ctx context.Context, providerName string, params *storage_dto.PutParams) error

	// GetObject retrieves an object from storage as a readable stream.
	//
	// Takes providerName (string) which identifies the storage provider to use.
	// Takes params (storage_dto.GetParams) which specifies the object to retrieve.
	//
	// Returns io.ReadCloser which provides the object data as a stream.
	// Returns error when the object cannot be retrieved.
	GetObject(ctx context.Context, providerName string, params storage_dto.GetParams) (io.ReadCloser, error)

	// StatObject retrieves metadata about an object
	// without downloading its content.
	//
	// Takes providerName (string) which identifies the storage provider to query.
	// Takes params (storage_dto.GetParams) which specifies the object to look up.
	//
	// Returns *ObjectInfo which contains the object metadata.
	// Returns error when the object does not exist or the provider is unavailable.
	StatObject(ctx context.Context, providerName string, params storage_dto.GetParams) (*ObjectInfo, error)

	// CopyObject copies an object from one location to another on the server.
	//
	// Takes providerName (string) which identifies the storage provider.
	// Takes params (CopyParams) which specifies the source and destination.
	//
	// Returns error when the copy fails.
	CopyObject(ctx context.Context, providerName string, params storage_dto.CopyParams) error

	// RemoveObject deletes an object from storage.
	//
	// Takes providerName (string) which identifies the storage provider.
	// Takes params (storage_dto.GetParams) which specifies the object to remove.
	//
	// Returns error when the object cannot be deleted.
	RemoveObject(ctx context.Context, providerName string, params storage_dto.GetParams) error

	// PutObjects uploads multiple objects in a single operation.
	//
	// Takes providerName (string) which identifies the storage provider.
	// Takes params (*storage_dto.PutManyParams) which
	// contains the objects to upload.
	//
	// Returns error when the upload fails.
	PutObjects(ctx context.Context, providerName string, params *storage_dto.PutManyParams) error

	// RemoveObjects deletes multiple objects in a single operation.
	//
	// Takes providerName (string) which identifies the storage provider to use.
	// Takes params (storage_dto.RemoveManyParams) which specifies the objects to
	// delete.
	//
	// Returns error when the deletion fails.
	RemoveObjects(ctx context.Context, providerName string, params storage_dto.RemoveManyParams) error

	// GetObjectHash computes and returns the hash of an object, such as MD5.
	//
	// Takes providerName (string) which identifies the storage provider to use.
	// Takes params (storage_dto.GetParams) which specifies the object to hash.
	//
	// Returns string which is the computed hash value.
	// Returns error when the hash cannot be computed.
	GetObjectHash(ctx context.Context, providerName string, params storage_dto.GetParams) (string, error)

	// GeneratePresignedUploadURL creates a temporary, signed URL for direct
	// client-side uploads.
	//
	// Takes providerName (string) which identifies the
	// storage provider to use.
	// Takes params (storage_dto.PresignParams) which
	// specifies the upload details.
	//
	// Returns string which is the presigned URL for
	// uploading.
	// Returns error when URL generation fails.
	GeneratePresignedUploadURL(ctx context.Context, providerName string, params storage_dto.PresignParams) (string, error)

	// GeneratePresignedDownloadURL creates a temporary, signed URL for direct
	// client-side downloads.
	//
	// Takes providerName (string) which identifies the
	// storage provider to use.
	// Takes params (storage_dto.PresignDownloadParams)
	// which specifies the download details.
	//
	// Returns string which is the presigned URL for downloading.
	// Returns error when URL generation fails.
	GeneratePresignedDownloadURL(ctx context.Context, providerName string, params storage_dto.PresignDownloadParams) (string, error)

	// Migrate orchestrates the transfer of objects from one provider to another.
	//
	// Takes params (*storage_dto.MigrateParams) which specifies the migration
	// settings.
	//
	// Returns *storage_dto.BatchResult which contains the success or failure
	// status of each object migration.
	// Returns error when the migration cannot be completed.
	Migrate(ctx context.Context, params *storage_dto.MigrateParams) (*storage_dto.BatchResult, error)

	// Close gracefully shuts down the storage service and releases all provider
	// resources. It iterates through all registered providers and calls their
	// Close methods.
	//
	//
	// Returns error when any provider fails to close. Errors from individual
	// providers are collected and returned as a combined error.
	Close(ctx context.Context) error

	// GetStats returns a snapshot of current service statistics.
	GetStats(ctx context.Context) ServiceStats

	// RegisterPublicRepository registers a repository as publicly accessible.
	// Public repositories serve files via permanent URLs without authentication.
	//
	// Takes name (string) which identifies the repository.
	// Takes cacheControl (string) which specifies the Cache-Control header.
	RegisterPublicRepository(name string, cacheControl string)

	// RegisterPrivateRepository registers a repository as
	// requiring authentication.
	// Private repositories serve files via presigned URLs with HMAC tokens.
	//
	// Takes name (string) which identifies the repository.
	// Takes cacheControl (string) which specifies the Cache-Control header.
	RegisterPrivateRepository(name string, cacheControl string)

	// IsPublicRepository checks if a repository is marked as public.
	//
	// Takes name (string) which identifies the repository.
	//
	// Returns bool which is true if the repository allows public access.
	IsPublicRepository(name string) bool

	// GetRepositoryConfig retrieves the configuration for a repository.
	//
	// Takes name (string) which identifies the repository.
	//
	// Returns *RepositoryConfig which contains the repository metadata.
	// Returns bool which indicates whether the repository was found.
	GetRepositoryConfig(name string) (*RepositoryConfig, bool)

	// GetPublicBaseURL returns the base URL for public storage URLs.
	// Returns empty string if not configured (relative URLs will be used).
	//
	// Returns string which is the base URL (e.g., "http://localhost:8080").
	GetPublicBaseURL() string

	// BuildPublicURL constructs a public storage URL for the given path.
	//
	// If PublicFallbackBaseURL is set, returns an absolute URL. Otherwise,
	// returns a relative URL.
	//
	// Takes path (string) which is the URL path.
	//
	// Returns string which is the complete URL.
	BuildPublicURL(path string) string
}

// StorageProviderPort defines the driven port for storage adapters.
//
// Each concrete storage implementation (such as S3, GCS, or local disk) must
// satisfy the contract. Implements io.Closer and provides operations for
// object storage including upload, download, copy, and batch operations.
type StorageProviderPort interface {
	// Put uploads an object from a reader.
	//
	// Takes params (*storage_dto.PutParams) which is a pointer for efficiency.
	//
	// Returns error when the upload fails.
	Put(ctx context.Context, params *storage_dto.PutParams) error

	// Get retrieves an object as a readable stream.
	//
	// Takes params (storage_dto.GetParams) which specifies the object to retrieve
	// and any byte range options.
	//
	// Returns io.ReadCloser which provides the object data as a stream.
	// Returns error when the object cannot be retrieved.
	Get(ctx context.Context, params storage_dto.GetParams) (io.ReadCloser, error)

	// Stat retrieves metadata for an object, such as size and content type.
	//
	// Takes params (storage_dto.GetParams) which specifies which object to query.
	//
	// Returns *ObjectInfo which contains the object metadata.
	// Returns error when the object cannot be found or the request fails.
	Stat(ctx context.Context, params storage_dto.GetParams) (*ObjectInfo, error)

	// Copy performs a server-side copy within the same repository (e.g., same
	// S3 bucket).
	//
	// Takes srcRepo (string) which identifies the source repository.
	// Takes srcKey (string) which is the key of the source object.
	// Takes dstKey (string) which is the key for the destination object.
	//
	// Returns error when the copy operation fails.
	Copy(ctx context.Context, srcRepo string, srcKey, dstKey string) error

	// CopyToAnotherRepository performs a server-side copy between different
	// repositories.
	//
	// Takes srcRepo (string) which is the source repository name.
	// Takes srcKey (string) which is the key of the object to copy.
	// Takes dstRepo (string) which is the destination repository name.
	// Takes dstKey (string) which is the key for the copied object.
	//
	// Returns error when the copy operation fails.
	CopyToAnotherRepository(ctx context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error

	// Remove deletes an object. Should be idempotent (not return an error if
	// the object does not exist).
	//
	// Takes params (storage_dto.GetParams) which identifies the object to remove.
	//
	// Returns error when the removal operation fails.
	Remove(ctx context.Context, params storage_dto.GetParams) error

	// Rename atomically moves an object from oldKey to newKey within the same
	// repository. This is used for atomic writes: write to a temp key, then rename
	// to the final key.
	//
	// Takes repo (string) which identifies the repository.
	// Takes oldKey (string) which is the source object path.
	// Takes newKey (string) which is the destination object path.
	//
	// Returns error when the rename operation fails.
	//
	// Implementations:
	//   - Disk: Uses os.Rename which is atomic on POSIX systems.
	//   - S3/GCS/R2: Uses Copy + Delete (not truly atomic, but acceptable for
	//     temp file patterns).
	Rename(ctx context.Context, repo string, oldKey, newKey string) error

	// Exists checks if an object exists at the given key.
	//
	// This is more efficient than Stat for existence checks. On disk it uses
	// os.Stat which is already efficient. On S3 it uses HeadObject which is
	// cheaper than GetObject.
	//
	// Takes params (storage_dto.GetParams) which specifies the key to check.
	//
	// Returns bool which is true if the object exists, false otherwise.
	// Returns error when the existence check fails.
	Exists(ctx context.Context, params storage_dto.GetParams) (bool, error)

	// GetHash returns the hash of an object, preferring efficient metadata
	// lookups over downloading.
	//
	// Takes params (storage_dto.GetParams) which specifies the object to hash.
	//
	// Returns string which is the hash of the object.
	// Returns error when the hash cannot be retrieved.
	GetHash(ctx context.Context, params storage_dto.GetParams) (string, error)

	// PresignURL generates a signed URL for direct client uploads
	// (e.g., to S3 or GCS).
	//
	// Takes params (storage_dto.PresignParams) which specifies the upload details.
	//
	// Returns string which is the signed URL for uploading.
	// Returns error when URL generation fails.
	PresignURL(ctx context.Context, params storage_dto.PresignParams) (string, error)

	// PresignDownloadURL generates a signed URL for direct client downloads
	// (e.g., from S3 or GCS).
	//
	// Takes params (storage_dto.PresignDownloadParams)
	// which specifies the download details.
	//
	// Returns string which is the signed URL for
	// downloading.
	// Returns error when URL generation fails.
	PresignDownloadURL(ctx context.Context, params storage_dto.PresignDownloadParams) (string, error)

	// Close gracefully releases any resources held by the provider
	// (e.g., client connections).
	//
	//
	// Returns error when resources cannot be released cleanly.
	Close(ctx context.Context) error

	// SupportsMultipart returns true if the provider has
	// a native, efficient multipart upload API.
	// The service layer uses this to decide whether to
	// automatically enable multipart for large files.
	SupportsMultipart() bool

	// SupportsRetry returns true if the provider handles retries internally
	// with its own backoff logic. If false, the service wraps the provider with
	// RetryableOperation.
	//
	// Returns bool which indicates whether the provider manages its own retries.
	SupportsRetry() bool

	// SupportsCircuitBreaking returns true if the provider has built-in circuit
	// breaker functionality. If false, the service will wrap the provider with
	// CircuitBreakerWrapper.
	//
	// Returns bool which indicates whether the provider supports circuit breaking.
	SupportsCircuitBreaking() bool

	// SupportsRateLimiting returns true if the provider handles rate limiting
	// internally. If false, the service may apply rate limiting at the service
	// layer.
	//
	// Returns bool which indicates whether internal rate limiting is supported.
	SupportsRateLimiting() bool

	// PutMany uploads multiple objects in a batch.
	//
	// Providers can implement native batch APIs or fall back to sequential or
	// concurrent uploads.
	//
	// Takes params (*storage_dto.PutManyParams) which specifies the objects to
	// upload.
	//
	// Returns *storage_dto.BatchResult which contains per-object success or
	// failure details.
	// Returns error when the batch operation fails entirely.
	PutMany(ctx context.Context, params *storage_dto.PutManyParams) (*storage_dto.BatchResult, error)

	// RemoveMany deletes multiple objects in a batch.
	//
	// Providers should use native batch delete APIs where available (for example,
	// S3 DeleteObjects).
	//
	// Takes params (storage_dto.RemoveManyParams) which specifies the objects to
	// delete.
	//
	// Returns *storage_dto.BatchResult which contains per-object success or
	// failure details.
	// Returns error when the batch operation fails.
	RemoveMany(ctx context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error)

	// SupportsBatchOperations returns true if the provider has native, efficient
	// batch APIs. The service layer uses this to decide whether to use batch
	// methods or fall back to loops.
	//
	// Returns bool which indicates whether batch operations are supported.
	SupportsBatchOperations() bool

	// SupportsPresignedURLs reports whether the provider can generate native
	// presigned URLs (e.g. S3, GCS).
	//
	// Returns bool which is true for native presigned URL support, or false when
	// the service layer must generate HMAC-signed tokens and provide an HTTP
	// upload endpoint (e.g. disk providers).
	SupportsPresignedURLs() bool
}

// StorageDispatcherPort defines the interface for batched, asynchronous
// storage operations, typically for high-throughput, non-critical writes.
type StorageDispatcherPort interface {
	// QueuePut adds a Put operation to the processing queue.
	//
	// Takes params (*storage_dto.PutParams) which contains the data to store.
	//
	// Returns error when the operation cannot be queued.
	QueuePut(ctx context.Context, params *storage_dto.PutParams) error

	// QueueRemove adds a remove operation to the processing queue.
	//
	// Takes params (storage_dto.GetParams) which identifies the item to remove.
	//
	// Returns error when the operation cannot be queued.
	QueueRemove(ctx context.Context, params storage_dto.GetParams) error

	// Flush processes all queued operations straight away.
	//
	//
	// Returns error when the flush operation fails.
	Flush(ctx context.Context) error

	// SetBatchSize sets the number of operations to group together in a batch.
	//
	// Takes size (int) which specifies the new batch size.
	SetBatchSize(size int)

	// SetFlushInterval updates the time between automatic flushes.
	//
	// Takes interval (time.Duration) which sets the new flush interval.
	SetFlushInterval(interval time.Duration)

	// GetStats returns statistics about the dispatcher's work and queue state.
	//
	// Returns DispatcherStats which contains current performance metrics.
	GetStats() DispatcherStats

	// Start begins the dispatcher's background processing loops.
	//
	//
	// Returns error when the dispatcher cannot start.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the dispatcher, ensuring all queued items are
	// processed.
	//
	//
	// Returns error when the shutdown fails or the context is cancelled.
	Stop(ctx context.Context) error
}
