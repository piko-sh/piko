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

import "time"

const (
	// OperationPut is the operation type for put requests to object storage.
	OperationPut = "put"

	// OperationGet is the operation name for getting objects from storage.
	OperationGet = "get"

	// OperationStat is the operation name for object stat requests.
	OperationStat = "stat"

	// OperationCopy is the operation name for object copy actions in metrics.
	OperationCopy = "copy"

	// OperationRemove is the operation name for object removal metrics.
	OperationRemove = "remove"

	// OperationGetHash is the operation name for getting an object hash.
	OperationGetHash = "gethash"

	// OperationPresign is the operation name for generating presigned upload URLs.
	OperationPresign = "presign"

	// OperationPresignDownload is the operation name for generating presigned
	// download URLs.
	OperationPresignDownload = "presign_download"

	// OperationPutObjects is the metric label for batch object upload operations.
	OperationPutObjects = "put_objects"

	// OperationRemoveObjects is the operation name for bulk object removal.
	OperationRemoveObjects = "remove_objects"

	// LogFieldOperation is the metric attribute key for the operation name.
	LogFieldOperation = "operation"

	// LogFieldTransformers is the log field key for transformer names.
	LogFieldTransformers = "transformers"

	// ValidationFailedFmt is the format string for wrapping validation errors.
	ValidationFailedFmt = "validation failed: %w"

	// PathSeparator is the forward slash used to separate parts of a file path.
	PathSeparator = "/"

	// DefaultMaxRetries is the default number of retries for failed requests.
	DefaultMaxRetries = 3

	// DefaultInitialRetryDelay is the wait time before the first retry attempt.
	DefaultInitialRetryDelay = 1 * time.Second

	// DefaultMaxRetryDelay is the maximum time to wait between retry attempts.
	DefaultMaxRetryDelay = 30 * time.Second

	// DefaultBackoffFactor is the default multiplier for exponential backoff.
	DefaultBackoffFactor = 2.0

	// DefaultMaxConsecutiveFailures is the default number of failures before the
	// circuit opens.
	DefaultMaxConsecutiveFailures = 5

	// DefaultCircuitBreakerTimeout is the default time the circuit stays open.
	DefaultCircuitBreakerTimeout = 60 * time.Second

	// DefaultCircuitBreakerInterval is the default interval for resetting the
	// failure counter.
	DefaultCircuitBreakerInterval = 10 * time.Second

	// DefaultMaxUploadSize is the default maximum file upload size (100 MB).
	DefaultMaxUploadSize = 104857600

	// DefaultMaxBatchSize is the default maximum number of objects in a batch
	// operation.
	DefaultMaxBatchSize = 1000

	// DefaultSingleflightMemoryThreshold is the default size limit for
	// singleflight buffering (10 MB).
	DefaultSingleflightMemoryThreshold = 10485760

	// DefaultMultipartThreshold is the file size above which multipart uploads are
	// enabled (100 MB).
	DefaultMultipartThreshold = 100 * 1024 * 1024

	// MaxKeyLength is the maximum allowed length for object keys.
	MaxKeyLength = 1024

	// MaxRetries is the upper limit for retry attempts in multipart uploads.
	MaxRetries = 10

	// DefaultMigrationBatchSize is the number of items to process at once during
	// migration when no value is given.
	DefaultMigrationBatchSize = 10

	// MaxConcurrency is the maximum allowed concurrency for batch operations.
	MaxConcurrency = 100

	// SHA256HexLength is the length of a SHA-256 hash in hexadecimal format (32
	// bytes = 64 hex chars).
	SHA256HexLength = 64

	// MD5HexLength is the length of an MD5 hash in hexadecimal format (16 bytes =
	// 32 hex chars).
	MD5HexLength = 32

	// MaxMultipartConcurrency is the maximum allowed concurrency for multipart
	// uploads.
	MaxMultipartConcurrency = 100

	// MetadataKeyCacheControl is the metadata key for HTTP Cache-Control
	// directives.
	MetadataKeyCacheControl = "Cache-Control"

	// MetadataKeyContentDisposition is the metadata key for content disposition
	// preference, controlling how content is presented. Valid values are "inline"
	// or "attachment".
	MetadataKeyContentDisposition = "Content-Disposition"
)
