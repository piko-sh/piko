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

package storage

import (
	"errors"
	"fmt"
	"io"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// StorageNameDefault is the name of the default storage provider used when no
	// specific provider is specified.
	StorageNameDefault = storage_dto.StorageProviderDefault

	// StorageNameSystem is the name of the system storage provider used for
	// internal framework operations.
	StorageNameSystem = storage_dto.StorageProviderSystem

	// StorageRepositoryDefault is the name of the default storage repository.
	StorageRepositoryDefault = storage_dto.StorageRepositoryDefault

	// TransformerCompression is the transformer type for compression operations.
	TransformerCompression = storage_dto.TransformerCompression

	// TransformerEncryption is a transformer type that encrypts data.
	TransformerEncryption = storage_dto.TransformerEncryption

	// TransformerCustom is a custom data transformer type.
	TransformerCustom = storage_dto.TransformerCustom
)

// Service manages storage providers and operations.
type Service = storage_domain.Service

// ServiceOption is a functional option for setting up the storage service.
type ServiceOption = storage_domain.ServiceOption

// ProviderPort defines the interface that all storage providers must implement.
// Implement it to create custom or mock providers.
type ProviderPort = storage_domain.StorageProviderPort

// DispatcherPort defines the interface for storage dispatchers that handle
// operations in the background.
type DispatcherPort = storage_domain.StorageDispatcherPort

// StreamTransformerPort represents the interface for stream transformers
// such as compression and encryption.
type StreamTransformerPort = storage_domain.StreamTransformerPort

// PutParams holds all parameters needed to upload an object.
type PutParams = storage_dto.PutParams

// GetParams holds the parameters needed to retrieve or work with an object.
type GetParams = storage_dto.GetParams

// CopyParams holds all parameters needed to copy an object.
type CopyParams = storage_dto.CopyParams

// PresignParams holds the parameters needed to create a presigned URL.
type PresignParams = storage_dto.PresignParams

// PresignDownloadParams holds all parameters needed to generate a presigned
// download URL.
type PresignDownloadParams = storage_dto.PresignDownloadParams

// PutManyParams holds parameters for batch upload operations.
type PutManyParams = storage_dto.PutManyParams

// RemoveManyParams holds settings for deleting several items at once.
type RemoveManyParams = storage_dto.RemoveManyParams

// MigrateParams holds parameters for moving objects between storage providers.
type MigrateParams = storage_dto.MigrateParams

// BatchResult represents the result of a batch operation with per-object
// success or failure details.
type BatchResult = storage_dto.BatchResult

// ObjectInfo contains metadata about a stored object.
type ObjectInfo = storage_domain.ObjectInfo

// DispatcherConfig holds settings for the background storage dispatcher.
type DispatcherConfig = storage_domain.DispatcherConfig

// DispatcherStats provides runtime statistics for monitoring the storage
// dispatcher.
type DispatcherStats = storage_domain.DispatcherStats

// RetryConfig holds settings for retry behaviour.
type RetryConfig = storage_domain.RetryConfig

// CircuitBreakerConfig holds settings for circuit breaker behaviour.
type CircuitBreakerConfig = storage_domain.CircuitBreakerConfig

// ProviderOption is a functional option for configuring provider-specific
// settings.
type ProviderOption = storage_domain.ProviderOption

// UploadBuilder provides a builder for uploading new objects to storage.
type UploadBuilder = storage_domain.UploadBuilder

// RequestBuilder provides methods to build and modify storage requests.
type RequestBuilder = storage_domain.RequestBuilder

// ProviderRateLimiter provides rate limiting for storage providers.
type ProviderRateLimiter = storage_domain.ProviderRateLimiter

// ProviderRateLimitConfig holds rate limiting configuration for a storage
// provider.
type ProviderRateLimitConfig = storage_domain.ProviderRateLimitConfig

// ProviderOptions holds settings for storage providers.
type ProviderOptions = storage_domain.ProviderOptions

// TransformerType identifies the kind of stream transformer.
type TransformerType = storage_dto.TransformerType

// TransformConfig sets up stream changes for Put and Get actions.
type TransformConfig = storage_dto.TransformConfig

// MultipartUploadConfig holds settings for multipart uploads.
type MultipartUploadConfig = storage_dto.MultipartUploadConfig

// ByteRange specifies a range of bytes for reading part of an object.
type ByteRange = storage_dto.ByteRange

// BatchFailure represents a single failed operation within a batch.
type BatchFailure = storage_dto.BatchFailure

// PutObjectSpec defines a single object to upload in a batch operation.
type PutObjectSpec = storage_dto.PutObjectSpec

var (
	// DefaultTransformConfig returns an empty transform configuration.
	DefaultTransformConfig = storage_dto.DefaultTransformConfig

	// DefaultMultipartConfig provides the standard settings for multipart uploads.
	DefaultMultipartConfig = storage_dto.DefaultMultipartConfig

	// ApplyProviderOptions applies functional options to create a rate limiter.
	//
	// Takes defaults (ProviderRateLimitConfig) which provides the base
	// configuration.
	// Takes opts (...ProviderOption) which modifies the rate limiting settings.
	//
	// Returns *ProviderRateLimiter which is the configured rate limiter ready for
	// use.
	ApplyProviderOptions = storage_domain.ApplyProviderOptions

	// IsRetryableError checks whether an error is temporary and worth retrying.
	//
	// Takes err (error) which is the error to check.
	//
	// Returns bool which is true when the error is retryable.
	IsRetryableError = storage_domain.IsRetryableError
)

// NewService creates a new storage service instance.
//
// Takes defaultProviderName (string) which specifies the provider to use when
// none is specified.
// Takes opts (...ServiceOption) which configures service limits and behaviour.
//
// Returns Service which is the configured storage service ready for use.
//
// Example:
//
//	service := storage.NewService("disk")
//	provider, _ := storage_provider_disk.NewProvider(ctx, config)
//	service.RegisterProvider(ctx, "disk", provider)
func NewService(defaultProviderName string, opts ...ServiceOption) Service {
	return storage_domain.NewServiceWithDefaultProvider(defaultProviderName, opts...)
}

// GetDefaultService returns the storage service initialised by the framework.
//
// Returns Service which is the initialised storage service instance.
// Returns error when the framework has not been bootstrapped.
func GetDefaultService() (Service, error) {
	service, err := bootstrap.GetStorageService()
	if err != nil {
		return nil, fmt.Errorf("storage: get default service: %w", err)
	}
	return service, nil
}

// NewUploadBuilder creates an upload builder for composing and uploading
// objects.
//
// Takes service (Service) which is the storage service to use for uploads.
// Takes reader (io.Reader) which provides the content to upload. The caller
// retains ownership of the reader and is responsible for closing it after Do()
// returns.
//
// Returns *UploadBuilder which provides a fluent interface for uploading
// objects.
// Returns error when service or reader is nil.
//
// Example:
//
//	service := storage.NewService("disk")
//	builder, err := storage.NewUploadBuilder(service, reader)
//	if err != nil {
//	    return err
//	}
//	err = builder.
//	    Key("documents/report.pdf").
//	    ContentType("application/pdf").
//	    Do(ctx)
func NewUploadBuilder(service Service, reader io.Reader) (*UploadBuilder, error) {
	if service == nil {
		return nil, errors.New("storage: service must not be nil")
	}
	if reader == nil {
		return nil, errors.New("storage: reader must not be nil")
	}
	return service.NewUpload(reader), nil
}

// NewUploadBuilderFromDefault creates a new upload builder using the
// framework's bootstrapped service.
//
// Takes reader (io.Reader) which provides the content to upload.
//
// Returns *UploadBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped or reader is nil.
//
// Example:
//
//	builder, err := storage.NewUploadBuilderFromDefault(reader)
//	if err != nil {
//	    return err
//	}
//	err = builder.
//	    Key("documents/report.pdf").
//	    ContentType("application/pdf").
//	    Do(ctx)
func NewUploadBuilderFromDefault(reader io.Reader) (*UploadBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf("storage: get default service for upload: %w", err)
	}
	return NewUploadBuilder(service, reader)
}

// NewRequestBuilder creates a new request builder for operating on existing
// objects.
//
// Takes service (Service) which is the storage service to use for operations.
// Takes repository (string) which identifies the storage repository.
// Takes key (string) which identifies the object within the repository.
//
// Returns *RequestBuilder which provides a fluent interface for object
// operations.
// Returns error when service is nil or repository/key is empty.
func NewRequestBuilder(service Service, repository, key string) (*RequestBuilder, error) {
	if service == nil {
		return nil, errors.New("storage: service must not be nil")
	}
	if repository == "" {
		return nil, errors.New("storage: repository must not be empty")
	}
	if key == "" {
		return nil, errors.New("storage: key must not be empty")
	}
	return service.NewRequest(repository, key), nil
}

// NewRequestBuilderFromDefault creates a new request builder using the
// framework's bootstrapped service.
//
// Takes repository (string) which identifies the storage repository.
// Takes key (string) which identifies the object within the repository.
//
// Returns *RequestBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped or parameters
// are invalid.
//
// Example:
//
//	builder, err := storage.NewRequestBuilderFromDefault("default", "documents/report.pdf")
//	if err != nil {
//	    return err
//	}
//	data, err := builder.Get(ctx)
func NewRequestBuilderFromDefault(repository, key string) (*RequestBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf("storage: get default service for request: %w", err)
	}
	return NewRequestBuilder(service, repository, key)
}
