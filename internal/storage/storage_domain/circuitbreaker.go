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
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safeconv"
)

// circuitBreakerBucketPeriod is the duration of each measurement bucket
// for tracking failure counts.
const circuitBreakerBucketPeriod = 10 * time.Second

// CircuitBreakerWrapper wraps a storage provider with circuit breaker
// functionality. It implements StorageProviderPort and prevents cascading
// failures by opening the circuit after consecutive failures.
type CircuitBreakerWrapper struct {
	// provider is the underlying storage provider wrapped by the circuit breaker.
	provider StorageProviderPort

	// breaker wraps storage operations to prevent cascading failures.
	breaker *gobreaker.CircuitBreaker[any]

	// name identifies this circuit breaker instance.
	name string
}

var _ StorageProviderPort = (*CircuitBreakerWrapper)(nil)

// ErrCircuitBreakerUnexpectedType indicates that the circuit breaker's
// Execute call returned a value that could not be type-asserted to the
// expected result type.
var ErrCircuitBreakerUnexpectedType = errors.New("circuit breaker returned unexpected type")

// NewCircuitBreakerWrapper creates a new circuit breaker wrapper for a storage
// provider.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes provider (StorageProviderPort) which is the storage to wrap.
// Takes config (CircuitBreakerConfig) which sets the circuit breaker options.
// Takes name (string) which identifies this circuit breaker in logs.
//
// Returns *CircuitBreakerWrapper which wraps the provider with circuit breaker
// protection.
func NewCircuitBreakerWrapper(ctx context.Context, provider StorageProviderPort, config CircuitBreakerConfig, name string) *CircuitBreakerWrapper {
	settings := gobreaker.Settings{
		Name:         fmt.Sprintf("storage-provider-%s", name),
		MaxRequests:  1,
		Interval:     config.Interval,
		Timeout:      config.Timeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= safeconv.IntToUint32(config.MaxConsecutiveFailures)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Circuit breaker state changed",
				logger_domain.String("provider", name),
				logger_domain.String("from", from.String()),
				logger_domain.String("to", to.String()))
		},
		IsSuccessful: nil,
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
	}

	return &CircuitBreakerWrapper{
		provider: provider,
		breaker:  gobreaker.NewCircuitBreaker[any](settings),
		name:     name,
	}
}

// Unwrap returns the underlying storage provider.
//
// Returns StorageProviderPort which is the wrapped provider.
func (cbw *CircuitBreakerWrapper) Unwrap() StorageProviderPort {
	return cbw.provider
}

// GetProviderType forwards the provider type query through the wrapper chain.
//
// Returns string which is the type from the inner provider, or "unknown" if
// no layer implements ProviderMetadata.
func (cbw *CircuitBreakerWrapper) GetProviderType() string {
	if meta, ok := cbw.provider.(provider_domain.ProviderMetadata); ok {
		return meta.GetProviderType()
	}
	return "unknown"
}

// GetProviderMetadata forwards the metadata query through the wrapper chain.
//
// Returns map[string]any which is the metadata from the inner provider, or nil
// if no layer implements ProviderMetadata.
func (cbw *CircuitBreakerWrapper) GetProviderMetadata() map[string]any {
	if meta, ok := cbw.provider.(provider_domain.ProviderMetadata); ok {
		return meta.GetProviderMetadata()
	}
	return nil
}

// Put executes a Put operation through the circuit breaker.
//
// Takes params (*storage_dto.PutParams) which specifies the storage parameters.
//
// Returns error when the circuit breaker is open or the underlying put fails.
func (cbw *CircuitBreakerWrapper) Put(ctx context.Context, params *storage_dto.PutParams) error {
	_, err := cbw.breaker.Execute(func() (any, error) {
		return nil, cbw.provider.Put(ctx, params)
	})
	if err != nil {
		return fmt.Errorf("executing put through circuit breaker: %w", err)
	}
	return nil
}

// Get retrieves content through the circuit breaker.
//
// Takes params (storage_dto.GetParams) which specifies what to retrieve.
//
// Returns io.ReadCloser which provides access to the retrieved content.
// Returns error when the circuit breaker is open, the operation fails, or the
// result is not the expected type.
func (cbw *CircuitBreakerWrapper) Get(ctx context.Context, params storage_dto.GetParams) (io.ReadCloser, error) {
	result, err := cbw.breaker.Execute(func() (any, error) {
		return cbw.provider.Get(ctx, params)
	})
	if err != nil {
		return nil, fmt.Errorf("executing get through circuit breaker: %w", err)
	}
	reader, ok := result.(io.ReadCloser)
	if !ok {
		return nil, fmt.Errorf("operation Get: %w", ErrCircuitBreakerUnexpectedType)
	}
	return reader, nil
}

// Stat executes a Stat operation through the circuit breaker.
//
// Takes params (storage_dto.GetParams) which specifies the object to query.
//
// Returns *ObjectInfo which contains metadata about the requested object.
// Returns error when the circuit breaker is open or the underlying operation
// fails.
func (cbw *CircuitBreakerWrapper) Stat(ctx context.Context, params storage_dto.GetParams) (*ObjectInfo, error) {
	result, err := cbw.breaker.Execute(func() (any, error) {
		return cbw.provider.Stat(ctx, params)
	})
	if err != nil {
		return nil, fmt.Errorf("executing stat through circuit breaker: %w", err)
	}
	info, ok := result.(*ObjectInfo)
	if !ok {
		return nil, fmt.Errorf("operation Stat: %w", ErrCircuitBreakerUnexpectedType)
	}
	return info, nil
}

// Copy executes a Copy operation through the circuit breaker.
//
// Takes srcRepo (string) which specifies the source repository.
// Takes srcKey (string) which identifies the source object.
// Takes dstKey (string) which identifies the destination object.
//
// Returns error when the circuit breaker is open or the copy fails.
func (cbw *CircuitBreakerWrapper) Copy(ctx context.Context, srcRepo string, srcKey string, dstKey string) error {
	_, err := cbw.breaker.Execute(func() (any, error) {
		return nil, cbw.provider.Copy(ctx, srcRepo, srcKey, dstKey)
	})
	if err != nil {
		return fmt.Errorf("executing copy through circuit breaker: %w", err)
	}
	return nil
}

// CopyToAnotherRepository executes a cross-repository copy operation through
// the circuit breaker.
//
// Takes srcRepo (string) which specifies the source repository name.
// Takes srcKey (string) which specifies the key of the item to copy.
// Takes dstRepo (string) which specifies the destination repository name.
// Takes dstKey (string) which specifies the key for the copied item.
//
// Returns error when the circuit breaker is open or the copy fails.
func (cbw *CircuitBreakerWrapper) CopyToAnotherRepository(ctx context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error {
	_, err := cbw.breaker.Execute(func() (any, error) {
		return nil, cbw.provider.CopyToAnotherRepository(ctx, srcRepo, srcKey, dstRepo, dstKey)
	})
	if err != nil {
		return fmt.Errorf("executing cross-repository copy through circuit breaker: %w", err)
	}
	return nil
}

// Remove executes a Remove operation through the circuit breaker.
//
// Takes params (storage_dto.GetParams) which specifies what to remove.
//
// Returns error when the circuit breaker is open or the removal fails.
func (cbw *CircuitBreakerWrapper) Remove(ctx context.Context, params storage_dto.GetParams) error {
	_, err := cbw.breaker.Execute(func() (any, error) {
		return nil, cbw.provider.Remove(ctx, params)
	})
	if err != nil {
		return fmt.Errorf("executing remove through circuit breaker: %w", err)
	}
	return nil
}

// GetHash executes a GetHash operation through the circuit breaker.
//
// Takes params (storage_dto.GetParams) which specifies what hash to retrieve.
//
// Returns string which is the retrieved hash value.
// Returns error when the circuit breaker is open or the underlying operation
// fails.
func (cbw *CircuitBreakerWrapper) GetHash(ctx context.Context, params storage_dto.GetParams) (string, error) {
	result, err := cbw.breaker.Execute(func() (any, error) {
		return cbw.provider.GetHash(ctx, params)
	})
	if err != nil {
		return "", fmt.Errorf("executing get-hash through circuit breaker: %w", err)
	}
	hash, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("operation GetHash: %w", ErrCircuitBreakerUnexpectedType)
	}
	return hash, nil
}

// PresignURL executes a PresignURL operation through the circuit breaker.
//
// Takes params (storage_dto.PresignParams) which specifies the presigning
// parameters.
//
// Returns string which is the presigned URL for the storage object.
// Returns error when the circuit breaker rejects the request or the underlying
// provider fails.
func (cbw *CircuitBreakerWrapper) PresignURL(ctx context.Context, params storage_dto.PresignParams) (string, error) {
	result, err := cbw.breaker.Execute(func() (any, error) {
		return cbw.provider.PresignURL(ctx, params)
	})
	if err != nil {
		return "", fmt.Errorf("executing presign-url through circuit breaker: %w", err)
	}
	url, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("operation PresignURL: %w", ErrCircuitBreakerUnexpectedType)
	}
	return url, nil
}

// PresignDownloadURL executes a presign download URL operation through
// the circuit breaker.
//
// Takes params (storage_dto.PresignDownloadParams) which specifies the download
// parameters.
//
// Returns string which is the presigned URL for downloading.
// Returns error when the circuit breaker rejects the request or the underlying
// provider fails.
func (cbw *CircuitBreakerWrapper) PresignDownloadURL(ctx context.Context, params storage_dto.PresignDownloadParams) (string, error) {
	result, err := cbw.breaker.Execute(func() (any, error) {
		return cbw.provider.PresignDownloadURL(ctx, params)
	})
	if err != nil {
		return "", fmt.Errorf("executing presign-download-url through circuit breaker: %w", err)
	}
	url, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("operation PresignDownloadURL: %w", ErrCircuitBreakerUnexpectedType)
	}
	return url, nil
}

// Close passes through to the underlying provider.
//
// Returns error when the underlying provider fails to close.
func (cbw *CircuitBreakerWrapper) Close(ctx context.Context) error {
	if err := cbw.provider.Close(ctx); err != nil {
		return fmt.Errorf("closing provider through circuit breaker: %w", err)
	}
	return nil
}

// SupportsMultipart passes through to the underlying provider.
//
// Returns bool which is true if the provider supports multipart requests.
func (cbw *CircuitBreakerWrapper) SupportsMultipart() bool {
	return cbw.provider.SupportsMultipart()
}

// RemoveMany deletes multiple items using the circuit breaker.
//
// Takes params (storage_dto.RemoveManyParams) which specifies which items to
// delete.
//
// Returns *storage_dto.BatchResult which contains the result of each delete.
// Returns error when the circuit breaker is open or the delete fails.
func (cbw *CircuitBreakerWrapper) RemoveMany(ctx context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	result, err := cbw.breaker.Execute(func() (any, error) {
		return cbw.provider.RemoveMany(ctx, params)
	})
	if err != nil {
		return nil, fmt.Errorf("executing remove-many through circuit breaker: %w", err)
	}
	batchResult, ok := result.(*storage_dto.BatchResult)
	if !ok {
		return nil, fmt.Errorf("operation RemoveMany: %w", ErrCircuitBreakerUnexpectedType)
	}
	return batchResult, nil
}

// PutMany executes batch upload through the circuit breaker.
//
// Takes params (*storage_dto.PutManyParams) which specifies the items to upload.
//
// Returns *storage_dto.BatchResult which contains the outcome of each item.
// Returns error when the circuit breaker fails or returns an unexpected type.
func (cbw *CircuitBreakerWrapper) PutMany(ctx context.Context, params *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	result, err := cbw.breaker.Execute(func() (any, error) {
		return cbw.provider.PutMany(ctx, params)
	})
	if err != nil {
		return nil, fmt.Errorf("executing put-many through circuit breaker: %w", err)
	}
	batchResult, ok := result.(*storage_dto.BatchResult)
	if !ok {
		return nil, fmt.Errorf("operation PutMany: %w", ErrCircuitBreakerUnexpectedType)
	}
	return batchResult, nil
}

// SupportsBatchOperations passes through to the underlying provider.
//
// Returns bool which is true if the underlying provider supports batch
// operations.
func (cbw *CircuitBreakerWrapper) SupportsBatchOperations() bool {
	return cbw.provider.SupportsBatchOperations()
}

// SupportsRetry passes through to the underlying provider.
//
// Returns bool which is true if the underlying provider supports retry.
func (cbw *CircuitBreakerWrapper) SupportsRetry() bool {
	return cbw.provider.SupportsRetry()
}

// SupportsCircuitBreaking passes through to the underlying provider.
//
// Returns bool which is true if the wrapped provider supports circuit breaking.
func (cbw *CircuitBreakerWrapper) SupportsCircuitBreaking() bool {
	return cbw.provider.SupportsCircuitBreaking()
}

// SupportsRateLimiting passes through to the underlying provider.
//
// Returns bool which is true if the underlying provider supports rate limiting.
func (cbw *CircuitBreakerWrapper) SupportsRateLimiting() bool {
	return cbw.provider.SupportsRateLimiting()
}

// SupportsPresignedURLs passes through to the underlying provider.
//
// Returns bool which is true if the underlying provider supports native
// presigned URLs.
func (cbw *CircuitBreakerWrapper) SupportsPresignedURLs() bool {
	return cbw.provider.SupportsPresignedURLs()
}

// Rename renames a blob from oldKey to newKey with circuit breaker protection.
//
// Takes repo (string) which identifies the repository.
// Takes oldKey (string) which specifies the current blob key.
// Takes newKey (string) which specifies the desired new blob key.
//
// Returns error when the rename fails or the circuit breaker is open.
func (cbw *CircuitBreakerWrapper) Rename(ctx context.Context, repo string, oldKey, newKey string) error {
	_, err := cbw.breaker.Execute(func() (any, error) {
		return nil, cbw.provider.Rename(ctx, repo, oldKey, newKey)
	})
	if err != nil {
		return fmt.Errorf("executing rename through circuit breaker: %w", err)
	}
	return nil
}

// Exists checks if a blob exists with circuit breaker protection.
//
// Takes params (storage_dto.GetParams) which specifies the blob to check.
//
// Returns bool which indicates whether the blob exists.
// Returns error when the circuit breaker fails or the underlying check fails.
func (cbw *CircuitBreakerWrapper) Exists(ctx context.Context, params storage_dto.GetParams) (bool, error) {
	result, err := cbw.breaker.Execute(func() (any, error) {
		return cbw.provider.Exists(ctx, params)
	})
	if err != nil {
		return false, fmt.Errorf("executing exists through circuit breaker: %w", err)
	}
	exists, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("operation Exists: %w", ErrCircuitBreakerUnexpectedType)
	}
	return exists, nil
}

// GetState returns the current state of the circuit breaker.
//
// Returns gobreaker.State which indicates whether the circuit is closed, open,
// or half-open.
func (cbw *CircuitBreakerWrapper) GetState() gobreaker.State {
	return cbw.breaker.State()
}

// GetCounts returns the current counts (successes, failures, etc.) of the
// circuit breaker.
//
// Returns gobreaker.Counts which contains the current state metrics.
func (cbw *CircuitBreakerWrapper) GetCounts() gobreaker.Counts {
	return cbw.breaker.Counts()
}
