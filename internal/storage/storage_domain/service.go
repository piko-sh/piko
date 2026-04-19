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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// logFieldSize is the log field key for object size values.
	logFieldSize = "size"

	// logFieldThreshold is the log field key for the memory threshold value.
	logFieldThreshold = "threshold"

	// serviceName is the name used to identify the storage service in health
	// probes.
	serviceName = "StorageService"

	// presignUploadPath is the URL path for presigned upload requests.
	presignUploadPath = "/_piko/storage/upload"

	// presignDownloadPath is the URL path for presigned download requests.
	presignDownloadPath = "/_piko/storage/download"

	// queryParamToken is the query parameter key for presigned URL tokens.
	queryParamToken = "token"

	// queryParamProvider is the query parameter key for provider names in
	// presigned URLs.
	queryParamProvider = "provider"

	// urlQuerySeparator is the character that separates a URL path from its
	// query string.
	urlQuerySeparator = "?"
)

// service provides the main storage service implementation.
// It implements storage.Service, io.Closer, and healthprobe.Probe.
type service struct {
	// dispatcher queues notifications for background sending; nil means send now.
	dispatcher StorageDispatcherPort

	// clock provides the time source for tracking operation durations.
	clock clock.Clock

	// tempSandbox provides sandboxed filesystem access for temporary files.
	tempSandbox safedisk.Sandbox

	// registry holds the provider registry for managing storage backends.
	registry *provider_domain.StandardRegistry[StorageProviderPort]

	// transformerRegistry holds stream transformers that change data during
	// transfer.
	transformerRegistry *TransformerRegistry

	// repositoryRegistry stores repository settings and controls access.
	repositoryRegistry *RepositoryRegistry

	// cancelFunc cancels the service's internal context, stopping background
	// goroutines such as the presign RID cache cleanup loop.
	cancelFunc context.CancelCauseFunc

	// getGroup stops repeated fetch requests for the same cache key.
	getGroup singleflight.Group

	// stats holds counters for service operations, updated atomically.
	stats ServiceStats

	// config holds the service settings used for validation.
	config ServiceConfig

	// totalBytesStored tracks the approximate total bytes stored for soft
	// quota enforcement, updated atomically on Put/Remove and reset on
	// restart.
	totalBytesStored int64

	// mu guards access to the providers map.
	mu sync.RWMutex
}

var _ Service = (*service)(nil)

// PutObject handles the full upload lifecycle, including validation,
// content-addressing, multipart decision logic, and delegation to the
// appropriate provider.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (*storage_dto.PutParams) which specifies the upload settings.
//
// Returns error when validation fails, the provider is not found, or the
// upload operation fails.
func (s *service) PutObject(ctx context.Context, providerName string, params *storage_dto.PutParams) error {
	startTime := s.clock.Now()
	atomic.AddInt64(&s.stats.TotalOperations, 1)

	if err := validatePutParams(params, &s.config); err != nil {
		s.recordOperationFailure(ctx, OperationPut)
		return fmt.Errorf("validating put params for %q: %w", params.Key, err)
	}

	if err := s.checkStorageQuota(params.Size); err != nil {
		s.recordOperationFailure(ctx, OperationPut)
		return fmt.Errorf("checking storage quota for key %q: %w", params.Key, err)
	}

	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		s.recordOperationFailure(ctx, OperationPut)
		return fmt.Errorf("resolving provider %q for put: %w", providerName, err)
	}

	if params.UseContentAddressing {
		casResult, err := handleContentAddressing(ctx, provider, params, s.tempSandbox)
		if err != nil {
			if errors.Is(err, errCASDeduplicated) {
				s.recordOperationSuccess(ctx, OperationPut, startTime)
				return nil
			}
			s.recordOperationFailure(ctx, OperationPut)
			return fmt.Errorf("handling content-addressable storage for put: %w", err)
		}

		defer casResult.Cleanup()
		defer func() { _ = casResult.Reader.Close() }()

		providerParams := *params
		providerParams.Reader = casResult.Reader
		providerParams.Key = casResult.Key

		params = &providerParams
	}

	enableMultipartIfNeeded(ctx, provider, params, providerName)

	if err := goroutine.SafeCall(ctx, "storage.Put", func() error { return provider.Put(ctx, params) }); err != nil {
		s.recordOperationFailure(ctx, OperationPut)
		return fmt.Errorf("uploading object %q to provider %q: %w", params.Key, providerName, err)
	}

	if params.Size > 0 {
		atomic.AddInt64(&s.totalBytesStored, params.Size)
	}

	s.recordOperationSuccess(ctx, OperationPut, startTime)
	return nil
}

// GetObject orchestrates object retrieval, intelligently deciding whether to
// use singleflight caching for small files or direct streaming for large ones.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (storage_dto.GetParams) which specifies the object to retrieve.
//
// Returns io.ReadCloser which provides access to the object data.
// Returns error when params are invalid, provider is not found, or retrieval
// fails.
func (s *service) GetObject(ctx context.Context, providerName string, params storage_dto.GetParams) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := s.clock.Now()

	if err := validateGetParams(params); err != nil {
		s.recordOperationFailure(ctx, OperationGet)
		return nil, fmt.Errorf("validating get params for %q: %w", params.Key, err)
	}

	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		s.recordOperationFailure(ctx, OperationGet)
		return nil, fmt.Errorf("resolving provider %q for get: %w", providerName, err)
	}

	if params.ByteRange != nil {
		reader, err := s.getObjectViaStream(ctx, provider, params)
		if err != nil {
			s.recordOperationFailure(ctx, OperationGet)
			return nil, fmt.Errorf("streaming object %q with byte range: %w", params.Key, err)
		}
		s.recordOperationSuccess(ctx, OperationGet, startTime)
		return reader, nil
	}

	info, err := goroutine.SafeCall1(ctx, "storage.Stat", func() (*ObjectInfo, error) { return provider.Stat(ctx, params) })
	if err != nil {
		s.recordOperationFailure(ctx, OperationGet)
		return nil, fmt.Errorf("statting object %q before get: %w", params.Key, err)
	}

	if s.config.EnableSingleflight && info.Size <= s.config.SingleflightMemoryThreshold {
		reader, err := s.getObjectViaSingleflight(ctx, provider, providerName, params, info.Size)
		if err != nil {
			s.recordOperationFailure(ctx, OperationGet)
			return nil, fmt.Errorf("getting object %q via singleflight: %w", params.Key, err)
		}
		s.recordOperationSuccess(ctx, OperationGet, startTime)
		return reader, nil
	}

	l.Trace("File exceeds singleflight threshold, streaming directly",
		logger_domain.String(logFieldKey, params.Key),
		logger_domain.Int64(logFieldSize, info.Size),
		logger_domain.Int64(logFieldThreshold, s.config.SingleflightMemoryThreshold))

	reader, err := s.getObjectViaStream(ctx, provider, params)
	if err != nil {
		s.recordOperationFailure(ctx, OperationGet)
		return nil, fmt.Errorf("streaming object %q directly: %w", params.Key, err)
	}
	s.recordOperationSuccess(ctx, OperationGet, startTime)
	return reader, nil
}

// StatObject gets metadata about an object without downloading its content.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (storage_dto.GetParams) which specifies the object to query.
//
// Returns *ObjectInfo which contains the object metadata.
// Returns error when params are invalid, provider is not found, or the stat
// operation fails.
func (s *service) StatObject(ctx context.Context, providerName string, params storage_dto.GetParams) (*ObjectInfo, error) {
	startTime := s.clock.Now()

	if err := validateGetParams(params); err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationStat)))
		return nil, fmt.Errorf("validating stat params for %q: %w", params.Key, err)
	}
	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationStat)))
		return nil, fmt.Errorf("resolving provider %q for stat: %w", providerName, err)
	}
	info, err := goroutine.SafeCall1(ctx, "storage.Stat", func() (*ObjectInfo, error) { return provider.Stat(ctx, params) })
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationStat)))
		return nil, fmt.Errorf("statting object %q on provider %q: %w", params.Key, providerName, err)
	}
	duration := s.clock.Now().Sub(startTime).Milliseconds()
	operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, OperationStat)))
	operationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationStat)))
	return info, nil
}

// CopyObject copies an object from source to destination, optionally across
// repositories.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (storage_dto.CopyParams) which specifies source and destination.
//
// Returns error when params are invalid, the provider is not found, or the
// copy operation fails.
func (s *service) CopyObject(ctx context.Context, providerName string, params storage_dto.CopyParams) error {
	startTime := s.clock.Now()

	if err := validateCopyParams(params); err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationCopy)))
		return fmt.Errorf("validating copy params: %w", err)
	}
	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationCopy)))
		return fmt.Errorf("resolving provider %q for copy: %w", providerName, err)
	}

	var copyErr error
	if params.SourceRepository == params.DestinationRepository {
		copyErr = goroutine.SafeCall(ctx, "storage.Copy", func() error {
			return provider.Copy(ctx, params.SourceRepository, params.SourceKey, params.DestinationKey)
		})
	} else {
		copyErr = goroutine.SafeCall(ctx, "storage.Copy", func() error {
			return provider.CopyToAnotherRepository(ctx, params.SourceRepository, params.SourceKey, params.DestinationRepository, params.DestinationKey)
		})
	}

	if copyErr != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationCopy)))
		return fmt.Errorf("copying object %q to %q on provider %q: %w", params.SourceKey, params.DestinationKey, providerName, copyErr)
	}

	duration := s.clock.Now().Sub(startTime).Milliseconds()
	operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, OperationCopy)))
	operationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationCopy)))
	return nil
}

// RemoveObject deletes an object from storage.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (storage_dto.GetParams) which specifies the object to remove.
//
// Returns error when params are invalid, provider is not found, or removal
// fails.
func (s *service) RemoveObject(ctx context.Context, providerName string, params storage_dto.GetParams) error {
	startTime := s.clock.Now()

	if err := validateGetParams(params); err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemove)))
		return fmt.Errorf("validating remove params for %q: %w", params.Key, err)
	}
	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemove)))
		return fmt.Errorf("resolving provider %q for remove: %w", providerName, err)
	}
	var removedSize int64
	if s.config.MaxStorageBytes > 0 {
		if info, statErr := goroutine.SafeCall1(ctx, "storage.Stat", func() (*ObjectInfo, error) { return provider.Stat(ctx, params) }); statErr == nil {
			removedSize = info.Size
		}
	}

	err = goroutine.SafeCall(ctx, "storage.Remove", func() error { return provider.Remove(ctx, params) })
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemove)))
		return fmt.Errorf("removing object %q from provider %q: %w", params.Key, providerName, err)
	}

	if removedSize > 0 {
		atomic.AddInt64(&s.totalBytesStored, -removedSize)
	}

	duration := s.clock.Now().Sub(startTime).Milliseconds()
	operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemove)))
	operationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemove)))
	return nil
}

// GetObjectHash retrieves the hash (ETag or content hash) of an object.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (storage_dto.GetParams) which specifies the object to retrieve.
//
// Returns string which is the hash value of the object.
// Returns error when the provider is not found or the operation fails.
func (s *service) GetObjectHash(ctx context.Context, providerName string, params storage_dto.GetParams) (string, error) {
	return s.executeStringOperation(ctx, providerName, OperationGetHash, func(provider StorageProviderPort) (string, error) {
		return goroutine.SafeCall1(ctx, "storage.GetHash", func() (string, error) { return provider.GetHash(ctx, params) })
	})
}

// GeneratePresignedUploadURL creates a presigned URL for uploading an object.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (storage_dto.PresignParams) which specifies the upload
// parameters.
//
// Returns string which is the presigned URL for uploading.
// Returns error when the URL cannot be generated.
func (s *service) GeneratePresignedUploadURL(ctx context.Context, providerName string, params storage_dto.PresignParams) (string, error) {
	return s.generatePresignedURLGeneric(ctx, providerName, params.Key, OperationPresign,
		func(p StorageProviderPort) (string, error) {
			return goroutine.SafeCall1(ctx, "storage.PresignURL", func() (string, error) { return p.PresignURL(ctx, params) })
		},
		func() (string, error) { return s.generateFallbackPresignedURL(ctx, providerName, params) },
	)
}

// GeneratePresignedDownloadURL creates a time-limited URL for direct client
// downloads.
//
// If the provider supports native presigned URLs (e.g., S3, GCS), it delegates
// to the provider. Otherwise, it generates a service-level presigned URL using
// HMAC-signed tokens that can be validated by the HTTP download handler.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (storage_dto.PresignDownloadParams) which specifies the download
// settings.
//
// Returns string which is the presigned URL for downloading.
// Returns error when the provider is not found or URL generation fails.
func (s *service) GeneratePresignedDownloadURL(ctx context.Context, providerName string, params storage_dto.PresignDownloadParams) (string, error) {
	return s.generatePresignedURLGeneric(ctx, providerName, params.Key, OperationPresignDownload,
		func(p StorageProviderPort) (string, error) {
			return goroutine.SafeCall1(ctx, "storage.PresignDownloadURL", func() (string, error) { return p.PresignDownloadURL(ctx, params) })
		},
		func() (string, error) { return s.generateFallbackPresignedDownloadURL(ctx, providerName, params) },
	)
}

// Name returns the service identifier for health probe discovery.
// Implements the healthprobe_domain.Probe interface.
//
// Returns string which is the constant "StorageService".
func (*service) Name() string {
	return serviceName
}

// GetPresignConfig returns the presigned URL configuration for this service,
// giving bootstrap code access to the presign settings for creating the HTTP
// upload handler.
//
// Returns PresignConfig which contains the presign settings including secret,
// expiry, and size limits.
func (s *service) GetPresignConfig() PresignConfig {
	return s.config.PresignConfig
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies that storage providers are available and operational.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which contains the health state, message,
// timing information, and dependency statuses.
func (s *service) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := s.clock.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(ctx, startTime)
	}

	providers := s.registry.ListProviders(ctx)
	providerCount := len(providers)
	defaultProvider := s.registry.GetDefaultProvider()

	overallState, dependencies := s.checkProviderHealth(ctx, checkType, defaultProvider)

	message := fmt.Sprintf("Storage service operational with %d provider(s)", providerCount)
	if overallState != healthprobe_dto.StateHealthy {
		message = "Storage service has provider issues"
	}

	return healthprobe_dto.Status{
		Name:         serviceName,
		State:        overallState,
		Message:      message,
		Timestamp:    s.clock.Now(),
		Duration:     s.clock.Now().Sub(startTime).String(),
		Dependencies: dependencies,
	}
}

// Close shuts down the storage service and releases all provider resources.
// It goes through all registered providers and calls their Close methods
// via the registry.
//
// Returns error when one or more providers fail to close; errors from
// individual providers are collected and returned as a combined error.
func (s *service) Close(ctx context.Context) error {
	if s.cancelFunc != nil {
		s.cancelFunc(errors.New("storage service shutdown"))
	}
	if err := s.registry.CloseAll(ctx); err != nil {
		return fmt.Errorf("closing all storage providers: %w", err)
	}
	return nil
}

// RegisterPublicRepository registers a repository as publicly accessible.
// Public repositories serve files via permanent URLs without authentication.
//
// Takes name (string) which identifies the repository.
// Takes cacheControl (string) which specifies the Cache-Control header for
// files.
func (s *service) RegisterPublicRepository(name string, cacheControl string) {
	s.repositoryRegistry.Register(&RepositoryConfig{
		Name:         name,
		IsPublic:     true,
		CacheControl: cacheControl,
	})
}

// RegisterPrivateRepository registers a repository as requiring authentication.
// Private repositories serve files via presigned URLs with HMAC tokens.
//
// Takes name (string) which identifies the repository.
// Takes cacheControl (string) which specifies the Cache-Control header for
// files.
func (s *service) RegisterPrivateRepository(name string, cacheControl string) {
	s.repositoryRegistry.Register(&RepositoryConfig{
		Name:         name,
		IsPublic:     false,
		CacheControl: cacheControl,
	})
}

// IsPublicRepository checks if a repository is marked as public.
//
// Takes name (string) which identifies the repository.
//
// Returns bool which is true if the repository allows public access.
func (s *service) IsPublicRepository(name string) bool {
	return s.repositoryRegistry.IsPublic(name)
}

// GetRepositoryConfig retrieves the configuration for a repository.
//
// Takes name (string) which identifies the repository.
//
// Returns *RepositoryConfig which contains the repository metadata.
// Returns bool which indicates whether the repository was found.
func (s *service) GetRepositoryConfig(name string) (*RepositoryConfig, bool) {
	return s.repositoryRegistry.Get(name)
}

// GetPublicBaseURL returns the base URL for public storage URLs.
// Returns empty string if not configured (relative URLs will be used).
//
// Returns string which is the base URL (e.g., "http://localhost:8080").
func (s *service) GetPublicBaseURL() string {
	return s.config.PublicFallbackBaseURL
}

// BuildPublicURL constructs a public storage URL for the given path.
//
// If PublicFallbackBaseURL is set, returns an absolute URL. Otherwise,
// returns a relative URL.
//
// Takes path (string) which is the URL path for the storage resource.
//
// Returns string which is the complete URL.
func (s *service) BuildPublicURL(path string) string {
	if s.config.PublicFallbackBaseURL != "" {
		return s.config.PublicFallbackBaseURL + path
	}
	return path
}

// generatePresignedURLGeneric creates a time-limited URL
// for direct client uploads.
//
// If the provider supports native presigned URLs (e.g., S3, GCS), it delegates
// to the provider. Otherwise, it generates a service-level presigned URL using
// HMAC-signed tokens that can be validated by the HTTP upload handler.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes key (string) which is the object key for the presigned URL.
// Takes operation (string) which names the operation for metrics.
// Takes presignNative (func(StorageProviderPort) (string, error)) which
// generates a native presigned URL via the provider.
// Takes presignFallback (func() (string, error)) which generates a
// service-level presigned URL when the provider lacks native support.
//
// Returns string which is the presigned URL for uploading.
// Returns error when the provider is not found or URL generation fails.
func (s *service) generatePresignedURLGeneric(
	ctx context.Context,
	providerName string,
	key string,
	operation string,
	presignNative func(StorageProviderPort) (string, error),
	presignFallback func() (string, error),
) (string, error) {
	startTime := s.clock.Now()

	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
		return "", fmt.Errorf("resolving provider %q for %s URL: %w", providerName, operation, err)
	}

	var result string
	if goroutine.SafeCallValue(ctx, "storage.SupportsPresignedURLs", func() bool { return provider.SupportsPresignedURLs() }) {
		result, err = presignNative(provider)
		if err != nil {
			operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
			return "", fmt.Errorf("generating native %s URL for %q: %w", operation, key, err)
		}
	} else {
		result, err = presignFallback()
		if err != nil {
			operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
			return "", fmt.Errorf("generating fallback %s URL for %q: %w", operation, key, err)
		}
	}

	duration := s.clock.Now().Sub(startTime).Milliseconds()
	operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
	operationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
	return result, nil
}

// generateFallbackPresignedDownloadURL creates a service-level presigned URL
// for providers that do not support native presigned URLs (e.g., disk
// provider).
//
// The generated URL points to the framework's HTTP download handler at
// /_piko/storage/download and includes an HMAC-signed token containing the
// download parameters.
//
// Takes providerName (string) which identifies the target storage provider.
// Takes params (storage_dto.PresignDownloadParams) which specifies the download
// settings.
//
// Returns string which is the presigned URL with a signed token.
// Returns error when token generation fails.
func (s *service) generateFallbackPresignedDownloadURL(ctx context.Context, providerName string, params storage_dto.PresignDownloadParams) (string, error) {
	_, l := logger_domain.From(ctx, log)
	if err := s.config.PresignConfig.EnsureSecret(); err != nil {
		return "", fmt.Errorf("failed to initialise presign secret: %w", err)
	}

	rid, err := GeneratePresignRID()
	if err != nil {
		return "", fmt.Errorf("failed to generate random identifier: %w", err)
	}

	expiry := s.config.PresignConfig.ClampExpiry(params.ExpiresIn)

	tokenData := PresignDownloadTokenData{
		Key:         params.Key,
		Repository:  params.Repository,
		FileName:    params.FileName,
		ContentType: params.ContentType,
		ExpiresAt:   s.clock.Now().Add(expiry).Unix(),
		RID:         rid,
	}

	token, err := GeneratePresignDownloadToken(s.config.PresignConfig.Secret, tokenData)
	if err != nil {
		return "", fmt.Errorf("failed to generate presign download token: %w", err)
	}

	downloadURL := s.buildPresignedDownloadURL(token, providerName)

	l.Debug("Generated service-level presigned download URL",
		logger_domain.String(logFieldProvider, providerName),
		logger_domain.String("key", params.Key),
		logger_domain.Duration("expiry", expiry))

	return downloadURL, nil
}

// buildPresignedDownloadURL builds the full URL for the presigned download
// endpoint.
//
// Takes token (string) which is the signed token to include as a query
// parameter.
// Takes providerName (string) which identifies the storage provider.
//
// Returns string which is the complete presigned download URL.
func (s *service) buildPresignedDownloadURL(token string, providerName string) string {
	baseURL := s.config.PresignFallbackBaseURL
	if baseURL == "" {
		baseURL = ""
	}

	downloadPath := presignDownloadPath
	params := url.Values{}
	params.Set(queryParamToken, token)
	params.Set(queryParamProvider, providerName)

	if baseURL != "" {
		return baseURL + downloadPath + urlQuerySeparator + params.Encode()
	}
	return downloadPath + urlQuerySeparator + params.Encode()
}

// generateFallbackPresignedURL creates a service-level presigned URL for
// providers that do not support native presigned URLs (e.g., disk provider).
//
// The generated URL points to the framework's HTTP upload handler at
// /_piko/storage/upload and includes an HMAC-signed token containing the upload
// parameters.
//
// Takes providerName (string) which identifies the target storage provider.
// Takes params (storage_dto.PresignParams) which specifies the upload settings.
//
// Returns string which is the presigned URL with a signed token.
// Returns error when token generation fails.
func (s *service) generateFallbackPresignedURL(ctx context.Context, providerName string, params storage_dto.PresignParams) (string, error) {
	_, l := logger_domain.From(ctx, log)
	if err := s.config.PresignConfig.EnsureSecret(); err != nil {
		return "", fmt.Errorf("failed to initialise presign secret: %w", err)
	}

	tempKey := params.Key

	rid, err := GeneratePresignRID()
	if err != nil {
		return "", fmt.Errorf("failed to generate random identifier: %w", err)
	}

	expiry := s.config.PresignConfig.ClampExpiry(params.ExpiresIn)
	maxSize := s.config.PresignConfig.ClampMaxSize(s.config.PresignConfig.DefaultMaxSize)

	tokenData := PresignTokenData{
		TempKey:     tempKey,
		Repository:  params.Repository,
		ContentType: params.ContentType,
		MaxSize:     maxSize,
		ExpiresAt:   s.clock.Now().Add(expiry).Unix(),
		RID:         rid,
	}

	token, err := GeneratePresignToken(s.config.PresignConfig.Secret, tokenData)
	if err != nil {
		return "", fmt.Errorf("failed to generate presign token: %w", err)
	}

	uploadURL := s.buildPresignedUploadURL(token, providerName)

	l.Debug("Generated service-level presigned URL",
		logger_domain.String(logFieldProvider, providerName),
		logger_domain.String("temp_key", tempKey),
		logger_domain.Duration("expiry", expiry))

	return uploadURL, nil
}

// buildPresignedUploadURL builds the full URL for the presigned upload
// endpoint.
//
// Takes token (string) which is the signed token to include as a query
// parameter.
// Takes providerName (string) which identifies the storage provider.
//
// Returns string which is the complete presigned upload URL.
func (s *service) buildPresignedUploadURL(token string, providerName string) string {
	baseURL := s.config.PresignFallbackBaseURL
	if baseURL == "" {
		baseURL = ""
	}

	uploadPath := presignUploadPath
	params := url.Values{}
	params.Set(queryParamToken, token)
	params.Set(queryParamProvider, providerName)

	if baseURL != "" {
		return baseURL + uploadPath + urlQuerySeparator + params.Encode()
	}
	return uploadPath + urlQuerySeparator + params.Encode()
}

// checkStorageQuota returns an error if adding size bytes would exceed the
// configured MaxStorageBytes limit. When MaxStorageBytes is 0 (the default) or
// size is unknown (0), the check is skipped.
//
// Takes size (int64) which is the number of bytes about to be stored.
//
// Returns error when the upload would exceed the storage quota.
func (s *service) checkStorageQuota(size int64) error {
	if s.config.MaxStorageBytes <= 0 || size <= 0 {
		return nil
	}
	current := atomic.LoadInt64(&s.totalBytesStored)
	if current+size > s.config.MaxStorageBytes {
		return fmt.Errorf("storage quota exceeded: current %d bytes + %d bytes would exceed limit of %d bytes",
			current, size, s.config.MaxStorageBytes)
	}
	return nil
}

// recordOperationSuccess records metrics for a successful operation.
//
// Takes operation (string) which names the operation being recorded.
// Takes startTime (time.Time) which marks when the operation began.
func (s *service) recordOperationSuccess(ctx context.Context, operation string, startTime time.Time) {
	atomic.AddInt64(&s.stats.SuccessfulOperations, 1)
	duration := s.clock.Now().Sub(startTime).Milliseconds()
	operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
	operationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
}

// recordOperationFailure records metrics when an operation fails.
//
// Takes operation (string) which names the failed operation.
func (s *service) recordOperationFailure(ctx context.Context, operation string) {
	atomic.AddInt64(&s.stats.FailedOperations, 1)
	operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
}

// executeStringOperation runs a provider operation that returns a string and
// records metrics for the operation.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes operation (string) which names the operation for metrics labels.
// Takes callback (func(...)) which performs the actual operation on the provider.
//
// Returns string which is the result from the provider operation.
// Returns error when the provider cannot be found or the operation fails.
func (s *service) executeStringOperation(
	ctx context.Context, providerName string, operation string,
	callback func(provider StorageProviderPort) (string, error),
) (string, error) {
	startTime := s.clock.Now()

	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
		return "", fmt.Errorf("resolving provider %q for %s: %w", providerName, operation, err)
	}
	result, err := callback(provider)
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
		return "", fmt.Errorf("executing %s on provider %q: %w", operation, providerName, err)
	}
	duration := s.clock.Now().Sub(startTime).Milliseconds()
	operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
	operationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, operation)))
	return result, nil
}

// getObjectViaStream retrieves an object directly from the provider without
// caching.
//
// Takes provider (StorageProviderPort) which provides access to storage.
// Takes params (storage_dto.GetParams) which specifies the object to retrieve.
//
// Returns io.ReadCloser which provides streaming access to the object data.
// Returns error when the provider fails to retrieve the object.
func (*service) getObjectViaStream(
	ctx context.Context, provider StorageProviderPort, params storage_dto.GetParams,
) (io.ReadCloser, error) {
	reader, err := goroutine.SafeCall1(ctx, "storage.Get", func() (io.ReadCloser, error) { return provider.Get(ctx, params) })
	if err != nil {
		return nil, fmt.Errorf("streaming object %q from provider: %w", params.Key, err)
	}
	return reader, nil
}

// getObjectViaSingleflight uses a singleflight group to deduplicate
// concurrent requests for the same object.
//
// The singleflight buffer is bounded by SingleflightMemoryThreshold (the same
// gating threshold used to decide whether to use this path). A provider that
// reports a small object via Stat but streams more bytes than the threshold
// is rejected with ErrSingleflightObjectTooLarge so an oversized payload
// cannot dominate memory.
//
// Takes provider (StorageProviderPort) which fetches the object data.
// Takes providerName (string) which identifies the provider for cache keys.
// Takes params (storage_dto.GetParams) which specifies the object to fetch.
// Takes size (int64) which indicates the expected object size for logging.
//
// Returns io.ReadCloser which provides access to the buffered object data.
// Returns error when the underlying provider fails or returns unexpected data.
func (s *service) getObjectViaSingleflight(
	ctx context.Context, provider StorageProviderPort,
	providerName string, params storage_dto.GetParams, size int64,
) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	cacheKey := fmt.Sprintf("%s:%s:%s", providerName, params.Repository, params.Key)

	maxBytes := s.config.SingleflightMemoryThreshold
	if maxBytes <= 0 {
		maxBytes = DefaultSingleflightMemoryThreshold
	}

	result, err, _ := s.getGroup.Do(cacheKey, func() (any, error) {
		l.Trace("Singleflight cache miss: fetching from underlying provider",
			logger_domain.String(logFieldKey, params.Key), logger_domain.Int64(logFieldSize, size))

		reader, getErr := goroutine.SafeCall1(ctx, "storage.Get", func() (io.ReadCloser, error) { return provider.Get(ctx, params) })
		if getErr != nil {
			return nil, getErr
		}
		defer func() { _ = reader.Close() }()

		data, readErr := io.ReadAll(io.LimitReader(reader, maxBytes+1))
		if readErr != nil {
			return nil, readErr
		}
		if int64(len(data)) > maxBytes {
			return nil, fmt.Errorf("%w: read at least %d bytes, threshold %d",
				ErrSingleflightObjectTooLarge, len(data), maxBytes)
		}
		return data, nil
	})

	if err != nil {
		return nil, fmt.Errorf("singleflight operation failed for key '%s': %w", params.Key, err)
	}

	data, ok := result.([]byte)
	if !ok {
		return nil, fmt.Errorf("singleflight operation returned unexpected type for key '%s'", params.Key)
	}
	l.Trace("Singleflight cache hit: returning buffered data",
		logger_domain.String(logFieldKey, params.Key), logger_domain.Int(logFieldSize, len(data)))
	return io.NopCloser(bytes.NewReader(data)), nil
}

// checkLiveness performs a basic liveness health check.
// It only verifies that the service is initialised with at least one provider.
//
// Takes startTime (time.Time) which records when the health check began.
//
// Returns healthprobe_dto.Status which contains the liveness state and timing
// information.
func (s *service) checkLiveness(ctx context.Context, startTime time.Time) healthprobe_dto.Status {
	providers := s.registry.ListProviders(ctx)
	providerCount := len(providers)

	state := healthprobe_dto.StateHealthy
	message := "Storage service is running"

	if providerCount == 0 {
		state = healthprobe_dto.StateUnhealthy
		message = "No storage providers registered"
	}

	return healthprobe_dto.Status{
		Name:         serviceName,
		State:        state,
		Message:      message,
		Timestamp:    s.clock.Now(),
		Duration:     s.clock.Now().Sub(startTime).String(),
		Dependencies: nil,
	}
}

// checkProviderHealth checks the health of all registered storage providers.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to run.
// Takes defaultProvider (string) which names the default provider for state
// aggregation.
//
// Returns healthprobe_dto.State which is the combined health state across all
// providers.
// Returns []*healthprobe_dto.Status which contains the status of each
// provider.
func (s *service) checkProviderHealth(
	ctx context.Context, checkType healthprobe_dto.CheckType, defaultProvider string,
) (healthprobe_dto.State, []*healthprobe_dto.Status) {
	dependencies := make([]*healthprobe_dto.Status, 0)
	overallState := healthprobe_dto.StateHealthy

	providerInfos := s.registry.ListProviders(ctx)

	for _, info := range providerInfos {
		provider, err := s.registry.GetProvider(ctx, info.Name)
		if err != nil {
			continue
		}

		isDefault := info.Name == defaultProvider

		rawProvider := unwrapStorageProvider(provider)
		if probe, ok := rawProvider.(interface {
			Name() string
			Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status
		}); ok {
			providerStatus := probe.Check(ctx, checkType)
			dependencies = append(dependencies, &providerStatus)

			overallState = s.aggregateProviderState(overallState, providerStatus.State, isDefault)
		} else {
			providerLabel := fmt.Sprintf("StorageProvider (%s)", info.Name)
			if isDefault {
				providerLabel += " [default]"
			}
			dependencies = append(dependencies, &healthprobe_dto.Status{
				Name:         providerLabel,
				State:        healthprobe_dto.StateHealthy,
				Message:      "Provider does not support health checks (skipped)",
				Timestamp:    s.clock.Now(),
				Duration:     "0s",
				Dependencies: nil,
			})
		}
	}

	return overallState, dependencies
}

// aggregateProviderState determines the overall health state based on
// individual provider states.
//
// Takes currentState (healthprobe_dto.State) which is the current
// aggregated state.
// Takes providerState (healthprobe_dto.State) which is the state of the
// provider being processed.
// Takes isDefault (bool) which indicates if this is the default provider.
//
// Returns healthprobe_dto.State which is the new aggregated state.
func (*service) aggregateProviderState(
	currentState, providerState healthprobe_dto.State, isDefault bool,
) healthprobe_dto.State {
	if providerState == healthprobe_dto.StateUnhealthy {
		if isDefault {
			return healthprobe_dto.StateUnhealthy
		}
		if currentState == healthprobe_dto.StateHealthy {
			return healthprobe_dto.StateDegraded
		}
		return currentState
	}

	if providerState == healthprobe_dto.StateDegraded && currentState == healthprobe_dto.StateHealthy {
		return healthprobe_dto.StateDegraded
	}

	return currentState
}

// NewService creates a new storage service with the given options.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes opts (...ServiceOption) which sets the service behaviour.
//
// Returns Service which is the storage service ready for use.
func NewService(ctx context.Context, opts ...ServiceOption) Service {
	config := defaultServiceConfig()
	for _, opt := range opts {
		opt(&config)
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	ctx, l := logger_domain.From(ctx, log)

	serviceCtx, serviceCancel := context.WithCancelCause(ctx)

	if err := config.PresignConfig.EnsureSecret(); err != nil {
		l.Warn("Failed to initialise presign secret, presigned uploads disabled",
			logger_domain.Error(err))
	}
	config.PresignConfig.EnsureRIDCache(serviceCtx, DefaultRIDCleanupInterval)

	tempSandbox := resolveStorageTempSandbox(&config, l)

	return &service{
		registry:            provider_domain.NewStandardRegistry[StorageProviderPort]("storage"),
		transformerRegistry: NewTransformerRegistry(),
		repositoryRegistry:  NewRepositoryRegistry(),
		config:              config,
		tempSandbox:         tempSandbox,
		clock:               clk,
		cancelFunc:          serviceCancel,
		dispatcher:          nil,
		getGroup:            singleflight.Group{},
		mu:                  sync.RWMutex{},
		stats: ServiceStats{
			StartTime:            clk.Now(),
			TotalOperations:      0,
			SuccessfulOperations: 0,
			FailedOperations:     0,
			RetryAttempts:        0,
			CacheHits:            0,
			CacheMisses:          0,
			DLQEntries:           0,
		},
	}
}

// NewServiceWithDefaultProvider creates a new storage service with a specified
// default provider name. The provider itself must be registered separately via
// RegisterProvider.
//
// Takes opts (...ServiceOption) which configures the service behaviour.
//
// Returns Service which is the configured storage service ready for use.
func NewServiceWithDefaultProvider(_ string, opts ...ServiceOption) Service {
	config := defaultServiceConfig()
	for _, opt := range opts {
		opt(&config)
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	_, l := logger_domain.From(context.Background(), log)

	serviceCtx, serviceCancel := context.WithCancelCause(context.Background())

	if err := config.PresignConfig.EnsureSecret(); err != nil {
		l.Warn("Failed to initialise presign secret, presigned uploads disabled",
			logger_domain.Error(err))
	}
	config.PresignConfig.EnsureRIDCache(serviceCtx, DefaultRIDCleanupInterval)

	tempSandbox := resolveStorageTempSandbox(&config, l)

	return &service{
		registry:            provider_domain.NewStandardRegistry[StorageProviderPort]("storage"),
		transformerRegistry: NewTransformerRegistry(),
		repositoryRegistry:  NewRepositoryRegistry(),
		config:              config,
		tempSandbox:         tempSandbox,
		clock:               clk,
		cancelFunc:          serviceCancel,
		dispatcher:          nil,
		getGroup:            singleflight.Group{},
		mu:                  sync.RWMutex{},
		stats: ServiceStats{
			StartTime:            clk.Now(),
			TotalOperations:      0,
			SuccessfulOperations: 0,
			FailedOperations:     0,
			RetryAttempts:        0,
			CacheHits:            0,
			CacheMisses:          0,
			DLQEntries:           0,
		},
	}
}

// resolveStorageTempSandbox returns a temp sandbox for CAS operations,
// preferring the injected sandbox, then the factory, then a no-op fallback.
//
// It logs a warning and returns nil when no sandbox can be created.
//
// Takes config (*ServiceConfig) which provides the sandbox, factory, and
// related settings.
// Takes l (logger_domain.Logger) which logs warnings on failure.
//
// Returns safedisk.Sandbox which provides write access to the temp directory,
// or nil when creation fails.
func resolveStorageTempSandbox(config *ServiceConfig, l logger_domain.Logger) safedisk.Sandbox {
	if config.TempSandbox != nil {
		return config.TempSandbox
	}
	if config.TempSandboxFactory != nil {
		sandbox, err := config.TempSandboxFactory.Create("storage-temp", os.TempDir(), safedisk.ModeReadWrite)
		if err == nil {
			return sandbox
		}
		l.Warn("Failed to create temp sandbox via factory, CAS operations may fail",
			logger_domain.Error(err))
	}
	sandbox, err := safedisk.NewNoOpSandbox(os.TempDir(), safedisk.ModeReadWrite)
	if err != nil {
		l.Warn("Failed to create temp sandbox, CAS operations may fail",
			logger_domain.Error(err))
		return nil
	}
	return sandbox
}

// enableMultipartIfNeeded turns on multipart upload for large files when the
// provider supports it.
//
// Takes provider (StorageProviderPort) which is checked for multipart support.
// Takes params (*storage_dto.PutParams) which is updated with a default
// multipart config if needed.
// Takes providerName (string) which identifies the provider in log messages.
func enableMultipartIfNeeded(
	ctx context.Context,
	provider StorageProviderPort,
	params *storage_dto.PutParams, providerName string,
) {
	if params.MultipartConfig != nil {
		return
	}
	if params.Size <= DefaultMultipartThreshold {
		return
	}
	if !goroutine.SafeCallValue(ctx, "storage.SupportsMultipart", func() bool { return provider.SupportsMultipart() }) {
		return
	}

	_, l := logger_domain.From(ctx, log)
	params.MultipartConfig = new(storage_dto.DefaultMultipartConfig())
	l.Trace("Automatically enabling multipart upload",
		logger_domain.String(logFieldKey, params.Key),
		logger_domain.Int64(logFieldSize, params.Size),
		logger_domain.String(logFieldProvider, providerName))
}

// unwrapStorageProvider walks through wrapper layers (TransformerWrapper,
// retryableOperation, CircuitBreakerWrapper) to retrieve the underlying
// storage provider for type assertions such as health check support.
//
// Takes provider (StorageProviderPort) which may be a wrapped provider.
//
// Returns StorageProviderPort which is the innermost unwrapped provider.
func unwrapStorageProvider(provider StorageProviderPort) StorageProviderPort {
	type unwrapper interface {
		Unwrap() StorageProviderPort
	}
	for {
		u, ok := provider.(unwrapper)
		if !ok {
			return provider
		}
		provider = u.Unwrap()
	}
}
