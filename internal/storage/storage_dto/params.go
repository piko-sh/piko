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

package storage_dto

import (
	"io"
	"time"
)

// TransformerType identifies the kind of stream transformer.
type TransformerType string

const (
	// defaultMultipartPartSizeMB is the default part size in megabytes (100 MB).
	defaultMultipartPartSizeMB = 100

	// defaultMultipartConcurrency is the number of parts to upload at once.
	defaultMultipartConcurrency = 5

	// defaultMultipartMaxRetries is the default number of retries for a failed
	// part.
	defaultMultipartMaxRetries = 3

	// bytesPerMB is the number of bytes in one megabyte (1024 x 1024).
	bytesPerMB = 1024 * 1024

	// TransformerCompression applies compression to data, such as zstd or gzip.
	TransformerCompression TransformerType = "compression"

	// TransformerEncryption is the transformer type for encryption operations.
	TransformerEncryption TransformerType = "encryption"

	// TransformerCustom is the transformer type for custom transformers.
	TransformerCustom TransformerType = "custom"
)

// TransformConfig configures stream transformations for Put and Get operations.
// Transformations are applied in priority order: ascending for Put, descending
// for Get.
type TransformConfig struct {
	// TransformerOptions maps transformer names to their settings.
	TransformerOptions map[string]any

	// EnabledTransformers lists the names of transformers to apply, in order.
	EnabledTransformers []string
}

// MultipartUploadConfig holds configuration for multipart uploads. Multipart
// uploads improve reliability and performance for large files by uploading
// parts concurrently, enabling retry of individual failed parts rather than
// the entire file, and supporting resume of interrupted uploads.
type MultipartUploadConfig struct {
	// PartSize is the size of each part in bytes, with a minimum of 5 MB and a
	// maximum of 5 GB. Default is 100 MB.
	PartSize int64

	// Concurrency is the number of parts to upload simultaneously.
	// Default: 5
	// Recommended: 5-10 for optimal throughput without overwhelming the network.
	Concurrency int

	// EnableChecksum enables integrity verification for each part after upload.
	// Default is true, which is recommended for data integrity.
	EnableChecksum bool

	// MaxRetries is the number of retries per part upload. Default is 3.
	MaxRetries int
}

// PutParams holds the parameters for uploading an object to storage.
type PutParams struct {
	// Reader provides the content to upload.
	Reader io.Reader

	// MultipartConfig sets the multipart upload behaviour for large files.
	MultipartConfig *MultipartUploadConfig

	// TransformConfig holds settings for stream transformations; nil uses
	// defaults.
	TransformConfig *TransformConfig

	// Metadata holds custom key-value pairs to store with the object.
	Metadata map[string]string

	// Key is the unique identifier for the object within the repository.
	Key string

	// ContentType is the MIME type of the object; must be a valid, non-empty
	// value.
	ContentType string

	// HashAlgorithm specifies which hash algorithm to use for content
	// verification. Defaults to "sha256" if empty.
	HashAlgorithm string

	// ExpectedHash is the hash value to check against after upload. If set, the
	// actual hash must match or the upload fails with an error.
	ExpectedHash string

	// Repository identifies the storage location for the content.
	Repository string

	// Size is the content size in bytes; -1 means unknown.
	Size int64

	// UseContentAddressing enables content-addressable storage mode.
	UseContentAddressing bool
}

// ByteRange specifies a byte range for partial object reads.
// This enables efficient partial downloads for use cases like video streaming,
// resuming interrupted downloads, or reading file metadata without fetching
// the entire file.
type ByteRange struct {
	// Start is the byte position where the range begins (0-based, inclusive).
	// Must be zero or greater.
	Start int64

	// End is the inclusive ending byte position, or -1 to read to the end of the
	// file. When not -1, must be >= Start.
	End int64
}

// GetParams holds parameters for operations that target a single existing
// object. It is used by Get, Stat, Remove, and GetHash operations across all
// storage providers.
type GetParams struct {
	// ByteRange specifies a byte range for partial object reads.
	ByteRange *ByteRange

	// TransformConfig specifies stream transformations such as decompression.
	TransformConfig *TransformConfig

	// Key is the unique identifier for the object within the repository.
	Key string

	// Repository identifies the storage location containing the object.
	Repository string
}

// CopyParams holds parameters for copying an object from a source to a
// destination. The source and destination can be in the same or different
// repositories.
type CopyParams struct {
	// SourceKey is the object key in the source repository to copy from.
	SourceKey string

	// DestinationKey is the object key where the copied data will be stored.
	DestinationKey string

	// SourceRepository is the name of the repository that holds the source object.
	SourceRepository string

	// DestinationRepository is the name of the target repository for the copy.
	DestinationRepository string
}

// RemoveManyParams holds parameters for bulk deletion of objects within a
// single repository.
type RemoveManyParams struct {
	// Repository identifies the storage location containing the objects.
	Repository string

	// Keys is the list of storage keys to remove.
	Keys []string

	// Concurrency controls parallel deletes for providers without native batching.
	// Default is 10.
	Concurrency int

	// ContinueOnError determines whether the batch continues after individual
	// failures. Default: true.
	ContinueOnError bool
}

// PutManyParams holds parameters for batch upload operations. It enables
// efficient bulk uploads with partial success handling via MultiError.
type PutManyParams struct {
	// TransformConfig sets the options for stream changes like compression.
	TransformConfig *TransformConfig

	// Repository identifies the storage location for the uploads.
	Repository string

	// Objects is the list of objects to upload in the batch operation.
	Objects []PutObjectSpec

	// Concurrency controls the number of parallel uploads. Default is 10.
	Concurrency int

	// ContinueOnError indicates whether to keep processing after a failure.
	ContinueOnError bool
}

// PutObjectSpec defines a single object to upload in a batch operation.
type PutObjectSpec struct {
	// Reader provides the content to upload.
	Reader io.Reader

	// Key is the storage path for this object.
	Key string

	// ContentType is the MIME type of the object; must not be empty.
	ContentType string

	// Size is the content size in bytes; -1 means unknown.
	Size int64
}

// PresignParams holds the settings for creating a presigned URL.
// A presigned URL gives direct, time-limited access for client-side uploads.
type PresignParams struct {
	// Key is the object path within the storage bucket.
	Key string

	// ContentType is the MIME type for the presigned URL; empty means no type.
	ContentType string

	// Repository specifies which storage location to use for the presigned URL.
	Repository string

	// ExpiresIn is how long the presigned URL stays valid.
	ExpiresIn time.Duration
}

// PresignDownloadParams holds settings for creating a presigned download URL.
// A presigned download URL gives direct, time-limited read access to an object.
type PresignDownloadParams struct {
	// Key is the object path within the storage bucket.
	Key string

	// Repository specifies the storage repository to use.
	Repository string

	// FileName is the suggested filename for the Content-Disposition header.
	FileName string

	// ContentType is the MIME type for the Content-Type header.
	ContentType string

	// ExpiresIn is how long the presigned URL remains valid.
	ExpiresIn time.Duration
}

// DefaultTransformConfig returns an empty transform configuration with no
// transformations enabled.
//
// Returns TransformConfig which is ready for use with empty transformer lists.
func DefaultTransformConfig() TransformConfig {
	return TransformConfig{
		EnabledTransformers: []string{},
		TransformerOptions:  make(map[string]any),
	}
}

// DefaultMultipartConfig returns sensible defaults for multipart uploads.
//
// Returns MultipartUploadConfig which contains the default settings for part
// size, concurrency, checksum checking, and retry behaviour.
func DefaultMultipartConfig() MultipartUploadConfig {
	return MultipartUploadConfig{
		PartSize:       defaultMultipartPartSizeMB * bytesPerMB,
		Concurrency:    defaultMultipartConcurrency,
		EnableChecksum: true,
		MaxRetries:     defaultMultipartMaxRetries,
	}
}
