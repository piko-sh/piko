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
	"time"

	"piko.sh/piko/internal/retry"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

// ObjectInfo holds details about a stored object, returned by Stat methods.
type ObjectInfo struct {
	// LastModified is when the object was last changed.
	LastModified time.Time

	// Metadata holds custom key-value pairs for the object.
	Metadata map[string]string

	// ContentType is the MIME type of the object content.
	ContentType string

	// ETag is the entity tag for the object; used for MD5 hash extraction.
	ETag string

	// Size is the total size of the object in bytes.
	Size int64
}

// RetryConfig is an alias for the shared retry configuration.
type RetryConfig = retry.Config

// CircuitBreakerConfig holds settings for the circuit breaker pattern.
type CircuitBreakerConfig struct {
	// MaxConsecutiveFailures is the number of consecutive failures before the
	// circuit opens.
	MaxConsecutiveFailures int

	// Timeout is how long the circuit stays open before it tries to recover.
	Timeout time.Duration

	// Interval is the time between resets of the failure counter when closed.
	Interval time.Duration
}

// ServiceConfig holds configuration for the storage service, including limits
// for security.
type ServiceConfig struct {
	// TempSandbox provides sandboxed filesystem access for temporary files
	// used during content-addressable storage operations. If nil, a default
	// sandbox rooted at the system temporary directory is created.
	TempSandbox safedisk.Sandbox

	// TempSandboxFactory creates sandboxes for temporary file operations.
	// When non-nil and TempSandbox is nil, this factory is used instead of
	// falling back to safedisk.NewNoOpSandbox.
	TempSandboxFactory safedisk.Factory

	// Clock provides time operations for the service, defaulting to the
	// real system clock when nil (primarily used for testing to make
	// time-based logic deterministic).
	Clock clock.Clock

	// PresignFallbackBaseURL is the base URL for generating presigned upload URLs.
	// If empty, the URL is generated relative to the request origin.
	PresignFallbackBaseURL string

	// PublicFallbackBaseURL is the base URL for generating public storage
	// URLs, where an empty value produces relative paths and a non-empty
	// value produces absolute URLs (e.g.,
	// "http://localhost:8080/_piko/storage/public/...").
	PublicFallbackBaseURL string

	// RetryConfig holds settings for retry behaviour with exponential backoff.
	RetryConfig RetryConfig

	// PresignConfig holds settings for service-level presigned URLs.
	// Used when a storage provider does not support native presigned URLs.
	PresignConfig PresignConfig

	// CircuitBreakerConfig holds settings for the circuit breaker pattern.
	CircuitBreakerConfig CircuitBreakerConfig

	// MaxUploadSizeBytes is the maximum size in bytes for a
	// single file upload, protecting against resource exhaustion
	// attacks (default: 104857600, 100 MB).
	MaxUploadSizeBytes int64

	// SingleflightMemoryThreshold is the maximum file size in
	// bytes for singleflight buffering, where larger files are
	// streamed directly without deduplication (default:
	// 10485760, 10 MB).
	SingleflightMemoryThreshold int64

	// MaxStorageBytes is the soft limit on total bytes stored across all
	// providers, where PutObject rejects uploads that would exceed this
	// threshold when non-zero (default: 0, unlimited).
	//
	// The counter resets on process restart; for persistent quotas,
	// implement tracking in the storage provider.
	MaxStorageBytes int64

	// MaxBatchSize is the maximum number of objects allowed in a
	// batch operation, preventing resource exhaustion from overly
	// large batch requests (default: 1000).
	MaxBatchSize int

	// EnableRetry controls whether retry logic is used for operations.
	// Default is true.
	EnableRetry bool

	// EnableCircuitBreaker enables circuit breaker protection for storage
	// operations. Default is true.
	EnableCircuitBreaker bool

	// EnableSingleflight determines whether singleflight is
	// enabled for read deduplication, where concurrent reads of
	// the same object are combined (default: true).
	EnableSingleflight bool
}

// ServiceOption is a function that modifies a ServiceConfig, enabling the
// functional options pattern.
type ServiceOption func(*ServiceConfig)

// ServiceStats holds counts and timing data for storage service activity.
// All counters use atomic operations for thread-safe updates.
type ServiceStats struct {
	// StartTime is when the service was created; zero means not started.
	StartTime time.Time

	// TotalOperations is the total number of storage operations attempted.
	TotalOperations int64

	// SuccessfulOperations is the count of operations that finished without error.
	SuccessfulOperations int64

	// FailedOperations is the count of operations that did not complete
	// successfully.
	FailedOperations int64

	// RetryAttempts is the total number of retry attempts made.
	RetryAttempts int64

	// CacheHits is the number of singleflight cache hits.
	CacheHits int64

	// CacheMisses is the number of singleflight cache misses.
	CacheMisses int64

	// DLQEntries is the number of entries in the dead letter queue.
	DLQEntries int64
}

// Uptime returns the duration since the service started.
//
// Uses the system clock; for testing, use UptimeAt with a controlled time.
//
// Returns time.Duration which is the elapsed time since service start.
func (s *ServiceStats) Uptime() time.Duration {
	return s.UptimeAt(time.Now())
}

// UptimeAt returns the duration between StartTime and the provided time.
// Use it for testing with a mock clock.
//
// Takes now (time.Time) which specifies the current time to calculate against.
//
// Returns time.Duration which is the elapsed time since the service started.
func (s *ServiceStats) UptimeAt(now time.Time) time.Duration {
	if s.StartTime.IsZero() {
		return 0
	}
	return now.Sub(s.StartTime)
}

// DefaultRetryConfig returns a retry configuration with sensible default
// values for storage operations.
//
// Returns RetryConfig which contains default values for retry behaviour.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    DefaultMaxRetries,
		InitialDelay:  DefaultInitialRetryDelay,
		MaxDelay:      DefaultMaxRetryDelay,
		BackoffFactor: DefaultBackoffFactor,
	}
}

// DefaultCircuitBreakerConfig returns a circuit breaker configuration with
// sensible default values.
//
// Returns CircuitBreakerConfig which contains default settings for failure
// threshold, timeout, and interval.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxConsecutiveFailures: DefaultMaxConsecutiveFailures,
		Timeout:                DefaultCircuitBreakerTimeout,
		Interval:               DefaultCircuitBreakerInterval,
	}
}

// WithMaxUploadSizeBytes sets the maximum upload size limit for the service.
//
// Takes maxBytes (int64) which specifies the size limit in bytes. Values of
// zero or less are ignored.
//
// Returns ServiceOption which sets the upload size limit when applied.
func WithMaxUploadSizeBytes(maxBytes int64) ServiceOption {
	return func(config *ServiceConfig) {
		if maxBytes > 0 {
			config.MaxUploadSizeBytes = maxBytes
		}
	}
}

// WithMaxBatchSize sets the maximum batch size for bulk operations.
//
// Takes maxSize (int) which specifies the maximum number of items per batch.
// Values less than or equal to zero are ignored.
//
// Returns ServiceOption which configures the batch size limit.
func WithMaxBatchSize(maxSize int) ServiceOption {
	return func(config *ServiceConfig) {
		if maxSize > 0 {
			config.MaxBatchSize = maxSize
		}
	}
}

// WithRetryConfig sets the retry configuration for the service.
//
// Takes retryConfig (RetryConfig) which specifies the retry behaviour settings.
//
// Returns ServiceOption which configures the retry settings on a service.
func WithRetryConfig(retryConfig RetryConfig) ServiceOption {
	return func(config *ServiceConfig) {
		config.RetryConfig = retryConfig
	}
}

// WithCircuitBreakerConfig sets the circuit breaker settings for the service.
//
// Takes circuitBreakerConfig (CircuitBreakerConfig) which specifies the circuit
// breaker settings.
//
// Returns ServiceOption which applies the circuit breaker settings to the
// service.
func WithCircuitBreakerConfig(circuitBreakerConfig CircuitBreakerConfig) ServiceOption {
	return func(config *ServiceConfig) {
		config.CircuitBreakerConfig = circuitBreakerConfig
	}
}

// WithRetryEnabled sets whether retry logic is active.
//
// Takes enabled (bool) which turns retry on or off.
//
// Returns ServiceOption which sets the retry behaviour.
func WithRetryEnabled(enabled bool) ServiceOption {
	return func(config *ServiceConfig) {
		config.EnableRetry = enabled
	}
}

// WithCircuitBreakerEnabled enables or disables circuit breaker.
//
// Takes enabled (bool) which specifies whether the circuit breaker is active.
//
// Returns ServiceOption which configures the circuit breaker setting.
func WithCircuitBreakerEnabled(enabled bool) ServiceOption {
	return func(config *ServiceConfig) {
		config.EnableCircuitBreaker = enabled
	}
}

// WithSingleflightEnabled sets whether singleflight is used for read requests.
// When enabled, duplicate reads are grouped together so the work is done once.
//
// Takes enabled (bool) which specifies whether singleflight is active.
//
// Returns ServiceOption which configures the singleflight setting.
func WithSingleflightEnabled(enabled bool) ServiceOption {
	return func(config *ServiceConfig) {
		config.EnableSingleflight = enabled
	}
}

// WithSingleflightMemoryThreshold sets the maximum file size for singleflight
// buffering. Files larger than this threshold are streamed directly without
// deduplication.
//
// Takes threshold (int64) which specifies the size limit in bytes.
//
// Returns ServiceOption which configures the threshold when applied.
func WithSingleflightMemoryThreshold(threshold int64) ServiceOption {
	return func(config *ServiceConfig) {
		if threshold > 0 {
			config.SingleflightMemoryThreshold = threshold
		}
	}
}

// WithClock sets a custom clock for time operations.
// This is primarily used for testing to make time-based logic deterministic.
//
// Takes c (clock.Clock) which provides the clock implementation to use.
//
// Returns ServiceOption which configures the service to use the given clock.
func WithClock(c clock.Clock) ServiceOption {
	return func(config *ServiceConfig) {
		config.Clock = c
	}
}

// WithPresignConfig sets the presigned URL settings for the service. Use this
// when a storage provider does not support native presigned URLs.
//
// Takes presignConfig (PresignConfig) which specifies the presigned URL
// settings.
//
// Returns ServiceOption which applies the presigned URL settings.
func WithPresignConfig(presignConfig PresignConfig) ServiceOption {
	return func(config *ServiceConfig) {
		config.PresignConfig = presignConfig
	}
}

// WithPresignFallbackBaseURL sets the base URL for service-level presigned
// URLs. This URL is used when generating fallback presigned URLs for providers
// that do not support native presigned URLs (e.g., disk provider).
//
// Takes baseURL (string) which specifies the base URL, such as
// "https://example.com".
//
// Returns ServiceOption which configures the presigned URL base URL.
func WithPresignFallbackBaseURL(baseURL string) ServiceOption {
	return func(config *ServiceConfig) {
		config.PresignFallbackBaseURL = baseURL
	}
}

// WithMaxStorageBytes sets a soft limit on total bytes stored, causing
// PutObject to return an error when the tracked total would exceed
// this value (set to 0 to disable the quota).
//
// The counter is best-effort and resets on process restart.
//
// Takes maxBytes (int64) which specifies the storage cap in bytes. Values of
// zero or less disable the quota.
//
// Returns ServiceOption which configures the storage quota when applied.
func WithMaxStorageBytes(maxBytes int64) ServiceOption {
	return func(config *ServiceConfig) {
		if maxBytes > 0 {
			config.MaxStorageBytes = maxBytes
		}
	}
}

// WithPublicFallbackBaseURL sets the base URL for public storage URLs,
// enabling absolute URL generation required when the website and
// CMS/API run on different ports or hosts.
//
// Takes baseURL (string) which specifies the base URL (e.g.,
// "http://localhost:8080").
//
// Returns ServiceOption which configures the public URL base URL.
func WithPublicFallbackBaseURL(baseURL string) ServiceOption {
	return func(config *ServiceConfig) {
		config.PublicFallbackBaseURL = baseURL
	}
}

// WithTempSandbox sets the sandbox used for temporary files during
// content-addressable storage operations.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed filesystem access
// for temporary files. When nil, the option is ignored.
//
// Returns ServiceOption which configures the temp sandbox when applied.
func WithTempSandbox(sandbox safedisk.Sandbox) ServiceOption {
	return func(config *ServiceConfig) {
		if sandbox != nil {
			config.TempSandbox = sandbox
		}
	}
}

// WithTempSandboxFactory sets the factory used to create temporary file
// sandboxes when no TempSandbox is directly injected.
//
// Takes factory (safedisk.Factory) which creates sandboxes for temporary
// file operations.
//
// Returns ServiceOption which configures the factory when applied.
func WithTempSandboxFactory(factory safedisk.Factory) ServiceOption {
	return func(config *ServiceConfig) {
		config.TempSandboxFactory = factory
	}
}

// defaultServiceConfig returns a ServiceConfig with sensible default values.
//
// Returns ServiceConfig which contains defaults ready for production use,
// with retry enabled and circuit breaker disabled.
func defaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxUploadSizeBytes:          DefaultMaxUploadSize,
		MaxBatchSize:                DefaultMaxBatchSize,
		RetryConfig:                 DefaultRetryConfig(),
		CircuitBreakerConfig:        DefaultCircuitBreakerConfig(),
		EnableRetry:                 true,
		EnableCircuitBreaker:        false,
		EnableSingleflight:          false,
		SingleflightMemoryThreshold: DefaultSingleflightMemoryThreshold,
		PresignConfig:               DefaultPresignConfig(),
		PresignFallbackBaseURL:      "",
		PublicFallbackBaseURL:       "",
	}
}
