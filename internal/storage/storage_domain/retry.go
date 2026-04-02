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
	"errors"
	"fmt"
	"io"
	"os"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// logFieldOperation is the key for the operation name in structured logs.
	logFieldOperation = "operation"

	// logFieldAttempt is the log field key for the retry attempt number.
	logFieldAttempt = "attempt"

	// logFieldAttempts is the log field key for the number of retry attempts.
	logFieldAttempts = "attempts"

	// logFieldWait is the log field key for retry wait duration.
	logFieldWait = "wait_duration"

	// logOpGet is the operation name for get requests in retry logs.
	logOpGet = "Get"
)

// errorClassifier classifies errors for retry decisions, treating filesystem
// and IO errors as permanent in addition to the shared defaults.
var errorClassifier = retry.NewErrorClassifier(
	retry.WithPermanentErrors(os.ErrNotExist, os.ErrPermission, io.EOF, io.ErrUnexpectedEOF),
)

// retryableOperation wraps a storage provider with retry logic using
// exponential backoff. It implements StorageProviderPort and transparently
// handles transient failures for key operations.
type retryableOperation struct {
	// provider is the storage backend that performs the actual operations.
	provider StorageProviderPort

	// clock provides time functions for retry delay calculations.
	clock clock.Clock

	// name identifies this operation for logging and error messages.
	name string

	// config holds the retry policy settings.
	config RetryConfig
}

// Unwrap returns the underlying storage provider.
//
// Returns StorageProviderPort which is the wrapped provider.
func (r *retryableOperation) Unwrap() StorageProviderPort {
	return r.provider
}

// GetProviderType forwards the provider type query through the wrapper chain.
//
// Returns string which is the type from the inner provider, or "unknown" if
// no layer implements ProviderMetadata.
func (r *retryableOperation) GetProviderType() string {
	if meta, ok := r.provider.(provider_domain.ProviderMetadata); ok {
		return meta.GetProviderType()
	}
	return "unknown"
}

// GetProviderMetadata forwards the metadata query through the wrapper chain.
//
// Returns map[string]any which is the metadata from the inner provider, or nil
// if no layer implements ProviderMetadata.
func (r *retryableOperation) GetProviderMetadata() map[string]any {
	if meta, ok := r.provider.(provider_domain.ProviderMetadata); ok {
		return meta.GetProviderMetadata()
	}
	return nil
}

// Put implements StorageProviderPort, wrapping the underlying provider's Put
// call with retry logic.
//
// Takes params (*storage_dto.PutParams) which specifies the key and data to
// store.
//
// Returns error when the put operation fails after all retry attempts.
func (r *retryableOperation) Put(ctx context.Context, params *storage_dto.PutParams) error {
	operation := func() (any, error) {
		return nil, r.provider.Put(ctx, params)
	}
	_, err := r.executeWithRetry(ctx, logOpPut, params.Key, operation)
	if err != nil {
		return fmt.Errorf("putting object %q with retry: %w", params.Key, err)
	}
	return nil
}

// Get implements StorageProviderPort, wrapping the underlying provider's Get
// call with retry logic.
//
// Takes params (storage_dto.GetParams) which specifies the storage key and
// retrieval options.
//
// Returns io.ReadCloser which provides access to the retrieved content.
// Returns error when all retry attempts fail or the result has an unexpected
// type.
func (r *retryableOperation) Get(ctx context.Context, params storage_dto.GetParams) (io.ReadCloser, error) {
	operation := func() (any, error) {
		return r.provider.Get(ctx, params)
	}
	result, err := r.executeWithRetry(ctx, logOpGet, params.Key, operation)
	if err != nil {
		return nil, fmt.Errorf("getting object %q with retry: %w", params.Key, err)
	}
	reader, ok := result.(io.ReadCloser)
	if !ok {
		return nil, errors.New("retry operation returned unexpected type for Get")
	}
	return reader, nil
}

// Remove implements StorageProviderPort, wrapping the underlying provider's
// Remove call with retry logic.
//
// Takes params (storage_dto.GetParams) which identifies the item to remove.
//
// Returns error when the removal fails after all retry attempts.
func (r *retryableOperation) Remove(ctx context.Context, params storage_dto.GetParams) error {
	operation := func() (any, error) {
		return nil, r.provider.Remove(ctx, params)
	}
	_, err := r.executeWithRetry(ctx, logOpRemove, params.Key, operation)
	if err != nil {
		return fmt.Errorf("removing object %q with retry: %w", params.Key, err)
	}
	return nil
}

// Stat implements StorageProviderPort by passing through to the underlying
// provider. Metadata operations are usually fast and are not retried to avoid
// hiding bigger problems.
//
// Takes params (storage_dto.GetParams) which specifies the object to query.
//
// Returns *ObjectInfo which contains the object metadata.
// Returns error when the underlying provider fails.
func (r *retryableOperation) Stat(ctx context.Context, params storage_dto.GetParams) (*ObjectInfo, error) {
	info, err := r.provider.Stat(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("statting object %q through retry wrapper: %w", params.Key, err)
	}
	return info, nil
}

// Copy implements StorageProviderPort by passing through to the underlying
// provider. Server-side copies are generally not retried at this layer.
//
// Takes srcRepo (string) which specifies the source repository.
// Takes srcKey (string) which identifies the source object.
// Takes dstKey (string) which identifies the destination object.
//
// Returns error when the underlying provider copy fails.
func (r *retryableOperation) Copy(ctx context.Context, srcRepo string, srcKey, dstKey string) error {
	if err := r.provider.Copy(ctx, srcRepo, srcKey, dstKey); err != nil {
		return fmt.Errorf("copying object %q to %q through retry wrapper: %w", srcKey, dstKey, err)
	}
	return nil
}

// CopyToAnotherRepository implements StorageProviderPort by passing through
// to the underlying provider.
//
// Takes srcRepo (string) which specifies the source repository name.
// Takes srcKey (string) which identifies the object to copy.
// Takes dstRepo (string) which specifies the destination repository name.
// Takes dstKey (string) which identifies the destination object key.
//
// Returns error when the underlying provider copy operation fails.
func (r *retryableOperation) CopyToAnotherRepository(ctx context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error {
	if err := r.provider.CopyToAnotherRepository(ctx, srcRepo, srcKey, dstRepo, dstKey); err != nil {
		return fmt.Errorf("copying object %q to repository %q through retry wrapper: %w", srcKey, dstRepo, err)
	}
	return nil
}

// GetHash implements StorageProviderPort by passing through to the underlying
// provider.
//
// Takes params (storage_dto.GetParams) which specifies what to retrieve.
//
// Returns string which is the hash value from the underlying provider.
// Returns error when the underlying provider fails.
func (r *retryableOperation) GetHash(ctx context.Context, params storage_dto.GetParams) (string, error) {
	hash, err := r.provider.GetHash(ctx, params)
	if err != nil {
		return "", fmt.Errorf("getting hash for object %q through retry wrapper: %w", params.Key, err)
	}
	return hash, nil
}

// PresignURL implements StorageProviderPort by passing through to the
// underlying provider.
//
// Takes params (storage_dto.PresignParams) which contains the presign request
// details.
//
// Returns string which is the presigned URL for the storage object.
// Returns error when the underlying provider fails to create the URL.
func (r *retryableOperation) PresignURL(ctx context.Context, params storage_dto.PresignParams) (string, error) {
	url, err := r.provider.PresignURL(ctx, params)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL for %q through retry wrapper: %w", params.Key, err)
	}
	return url, nil
}

// PresignDownloadURL implements StorageProviderPort by passing through to the
// underlying provider. Presigned URLs are not retried as they do not involve
// network calls to storage.
//
// Takes params (storage_dto.PresignDownloadParams) which specifies the download
// details.
//
// Returns string which is the signed URL for downloading.
// Returns error when URL generation fails.
func (r *retryableOperation) PresignDownloadURL(ctx context.Context, params storage_dto.PresignDownloadParams) (string, error) {
	url, err := r.provider.PresignDownloadURL(ctx, params)
	if err != nil {
		return "", fmt.Errorf("generating presigned download URL for %q through retry wrapper: %w", params.Key, err)
	}
	return url, nil
}

// Close implements StorageProviderPort by passing through to the underlying
// provider.
//
// Returns error when the underlying provider fails to close.
func (r *retryableOperation) Close(ctx context.Context) error {
	if err := r.provider.Close(ctx); err != nil {
		return fmt.Errorf("closing provider through retry wrapper: %w", err)
	}
	return nil
}

// SupportsMultipart implements StorageProviderPort by passing through to the
// underlying provider.
//
// Returns bool which indicates whether multipart uploads are supported.
func (r *retryableOperation) SupportsMultipart() bool {
	return r.provider.SupportsMultipart()
}

// RemoveMany implements batch delete with retry logic. Retries the entire batch
// if it fails with a retryable error.
//
// Takes params (storage_dto.RemoveManyParams) which specifies the keys to
// delete.
//
// Returns *storage_dto.BatchResult which contains the outcome of each deletion.
// Returns error when all retry attempts fail or the result type is unexpected.
func (r *retryableOperation) RemoveMany(ctx context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	operation := func() (any, error) {
		return r.provider.RemoveMany(ctx, params)
	}

	result, err := r.executeWithRetry(ctx, "RemoveMany", fmt.Sprintf("%d keys", len(params.Keys)), operation)
	if err != nil {
		return nil, fmt.Errorf("removing %d objects with retry: %w", len(params.Keys), err)
	}
	batchResult, ok := result.(*storage_dto.BatchResult)
	if !ok {
		return nil, errors.New("retry operation returned unexpected type for RemoveMany")
	}
	return batchResult, nil
}

// PutMany implements batch upload with retry logic.
// Retries the entire batch if it fails with a retryable error.
//
// Takes params (*storage_dto.PutManyParams) which contains the objects to
// upload.
//
// Returns *storage_dto.BatchResult which contains the outcome of each upload.
// Returns error when all retry attempts fail or the operation returns an
// unexpected type.
func (r *retryableOperation) PutMany(ctx context.Context, params *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	operation := func() (any, error) {
		return r.provider.PutMany(ctx, params)
	}

	result, err := r.executeWithRetry(ctx, "PutMany", fmt.Sprintf("%d objects", len(params.Objects)), operation)
	if err != nil {
		return nil, fmt.Errorf("putting %d objects with retry: %w", len(params.Objects), err)
	}
	batchResult, ok := result.(*storage_dto.BatchResult)
	if !ok {
		return nil, errors.New("retry operation returned unexpected type for PutMany")
	}
	return batchResult, nil
}

// SupportsBatchOperations passes through to the underlying provider.
//
// Returns bool which is true if the provider supports batch operations.
func (r *retryableOperation) SupportsBatchOperations() bool {
	return r.provider.SupportsBatchOperations()
}

// SupportsRetry passes through to the underlying provider.
//
// Returns bool which is true if the provider supports retry operations.
func (r *retryableOperation) SupportsRetry() bool {
	return r.provider.SupportsRetry()
}

// SupportsCircuitBreaking passes through to the underlying provider.
//
// Returns bool which is true if the provider supports circuit breaking.
func (r *retryableOperation) SupportsCircuitBreaking() bool {
	return r.provider.SupportsCircuitBreaking()
}

// SupportsRateLimiting passes through to the underlying provider.
//
// Returns bool which is true if the provider supports rate limiting.
func (r *retryableOperation) SupportsRateLimiting() bool {
	return r.provider.SupportsRateLimiting()
}

// SupportsPresignedURLs passes through to the underlying provider.
//
// Returns bool which is true if the provider supports native presigned URLs.
func (r *retryableOperation) SupportsPresignedURLs() bool {
	return r.provider.SupportsPresignedURLs()
}

// Rename changes a blob's key from oldKey to newKey with retry logic.
//
// Takes repo (string) which identifies the repository.
// Takes oldKey (string) which is the current key for the blob.
// Takes newKey (string) which is the new key for the blob.
//
// Returns error when the rename fails after all retry attempts.
func (r *retryableOperation) Rename(ctx context.Context, repo string, oldKey, newKey string) error {
	operation := func() (any, error) {
		return nil, r.provider.Rename(ctx, repo, oldKey, newKey)
	}
	_, err := r.executeWithRetry(ctx, "Rename", oldKey, operation)
	if err != nil {
		return fmt.Errorf("renaming object %q to %q with retry: %w", oldKey, newKey, err)
	}
	return nil
}

// Exists checks if a blob exists, retrying on transient failures.
//
// Takes params (storage_dto.GetParams) which specifies the blob to check.
//
// Returns bool which indicates whether the blob exists.
// Returns error when the check fails after all retry attempts.
func (r *retryableOperation) Exists(ctx context.Context, params storage_dto.GetParams) (bool, error) {
	operation := func() (any, error) {
		return r.provider.Exists(ctx, params)
	}
	result, err := r.executeWithRetry(ctx, "Exists", params.Key, operation)
	if err != nil {
		return false, fmt.Errorf("checking existence of object %q with retry: %w", params.Key, err)
	}
	exists, ok := result.(bool)
	if !ok {
		return false, errors.New("retry operation returned unexpected type for Exists")
	}
	return exists, nil
}

var _ StorageProviderPort = (*retryableOperation)(nil)

// executeWithRetry runs a storage operation with automatic retries on failure.
//
// Takes opName (string) which names the operation for logging.
// Takes key (string) which identifies the storage key being used.
// Takes operation (func() (any, error)) which runs the storage operation.
//
// Returns any which is the result from a successful operation.
// Returns error when the operation fails with a non-retryable error, goes past
// the maximum number of retries, or the context is cancelled.
func (r *retryableOperation) executeWithRetry(
	ctx context.Context,
	opName, key string,
	operation func() (any, error),
) (any, error) {
	ctx, l := logger_domain.From(ctx, log)
	var lastErr error
	for attempt := 1; ; attempt++ {
		result, err := operation()
		if err == nil {
			if attempt > 1 {
				l.Trace("Storage operation succeeded after retry",
					logger_domain.String(logFieldOperation, opName),
					logger_domain.String(logFieldKey, key),
					logger_domain.Int(logFieldAttempt, attempt))
			}
			return result, nil
		}

		lastErr = err

		if !IsRetryableError(err) {
			l.Trace("Storage operation failed with non-retryable error",
				logger_domain.String(logFieldOperation, opName), logger_domain.String(logFieldKey, key), logger_domain.Error(err))
			return nil, fmt.Errorf("non-retryable storage error during %s on key %q: %w", opName, key, err)
		}

		if !r.config.ShouldRetry(attempt) {
			l.Warn("Storage operation failed after max retries",
				logger_domain.String(logFieldOperation, opName), logger_domain.String(logFieldKey, key),
				logger_domain.Int(logFieldAttempts, attempt), logger_domain.Error(lastErr))
			return nil, fmt.Errorf("operation %s on key %s failed after %d attempts: %w", opName, key, attempt, lastErr)
		}

		now := r.clock.Now()
		nextRetryTime := r.config.CalculateNextRetry(attempt, now)
		waitDuration := nextRetryTime.Sub(now)
		l.Trace("Storage operation failed, scheduling retry",
			logger_domain.String(logFieldOperation, opName), logger_domain.String(logFieldKey, key),
			logger_domain.Int(logFieldAttempt, attempt), logger_domain.Duration(logFieldWait, waitDuration), logger_domain.Error(err))

		timer := r.clock.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("operation cancelled during retry wait: %w", ctx.Err())
		case <-timer.C():
		}
	}
}

// IsRetryableError reports whether an error is temporary and worth retrying.
// It checks for network errors, system call errors, and known retryable
// messages, whilst filtering out permanent errors including filesystem and
// IO errors.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error can be retried, false otherwise.
func IsRetryableError(err error) bool {
	return errorClassifier.IsRetryable(err)
}

// newRetryableOperation creates a new retryable operation wrapper.
//
// If clk is nil, the real system clock is used.
//
// Takes config (RetryConfig) which specifies the retry behaviour settings.
// Takes provider (StorageProviderPort) which provides storage access.
// Takes name (string) which identifies the operation for logging.
// Takes clk (clock.Clock) which provides time functions, or nil for the
// system clock.
//
// Returns *retryableOperation which is ready to execute with retry logic.
func newRetryableOperation(config RetryConfig, provider StorageProviderPort, name string, clk clock.Clock) *retryableOperation {
	if clk == nil {
		clk = clock.RealClock()
	}
	return &retryableOperation{
		config:   config,
		provider: provider,
		name:     name,
		clock:    clk,
	}
}
