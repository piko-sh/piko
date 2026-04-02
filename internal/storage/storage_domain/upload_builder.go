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
	"maps"

	"piko.sh/piko/internal/storage/storage_dto"
)

// UploadBuilder composes and executes object uploads. It sets upload parameters
// like metadata, transforms, and content-addressing.
//
// Usage:
// err := service.NewUpload(reader).
//
//	WithKey("path/to/image.jpg").
//	WithContentType("image/jpeg").
//	WithMetadata(map[string]string{"owner": "user-123"}).
//	WithTransformer("zstd").
//	Upload(ctx)
type UploadBuilder struct {
	// service provides access to upload operations for cloud storage.
	service *service

	// params holds the upload settings that builder methods change.
	params *storage_dto.PutParams

	// providerName is the storage provider to use; empty means the default provider.
	providerName string

	// useDispatcher indicates whether to use the dispatcher for async upload.
	useDispatcher bool
}

// NewUpload is the entry point for building a new upload operation.
//
// Takes reader (io.Reader) which contains the object's data to upload.
//
// Returns *UploadBuilder which provides a fluent interface for configuring
// the upload before execution.
func (s *service) NewUpload(reader io.Reader) *UploadBuilder {
	return &UploadBuilder{
		service:       s,
		providerName:  "",
		useDispatcher: false,
		params: &storage_dto.PutParams{
			Reader:               reader,
			Repository:           storage_dto.StorageRepositoryDefault,
			Key:                  "",
			ContentType:          "",
			Size:                 0,
			Metadata:             nil,
			MultipartConfig:      nil,
			TransformConfig:      nil,
			HashAlgorithm:        "",
			ExpectedHash:         "",
			UseContentAddressing: false,
		},
	}
}

// Key sets the key for the object.
//
// Takes key (string) which specifies the object key.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) Key(key string) *UploadBuilder {
	b.params.Key = key
	return b
}

// Repository sets the destination repository for the object.
//
// If this method is not called, the upload will go to the
// default repository (StorageRepositoryDefault).
//
// Takes repo (string) which specifies the repository name.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) Repository(repo string) *UploadBuilder {
	b.params.Repository = repo
	return b
}

// ContentType sets the MIME type of the object (e.g., "image/jpeg",
// "application/pdf").
//
// Takes contentType (string) which specifies the MIME type for the upload.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) ContentType(contentType string) *UploadBuilder {
	b.params.ContentType = contentType
	return b
}

// Size sets the total size of the object in bytes.
// This is important for optimising uploads (e.g., enabling multipart) and for
// validation.
//
// Takes size (int64) which specifies the object size in bytes.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) Size(size int64) *UploadBuilder {
	b.params.Size = size
	return b
}

// Metadata attaches custom key-value metadata to the object.
//
// Takes metadata (map[string]string) which contains the key-value pairs to
// attach.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) Metadata(metadata map[string]string) *UploadBuilder {
	b.params.Metadata = metadata
	return b
}

// Provider specifies a non-default storage provider for this upload.
// If not called, the service's default provider will be used.
//
// Takes name (string) which identifies the storage provider to use.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) Provider(name string) *UploadBuilder {
	b.providerName = name
	return b
}

// CAS enables Content-Addressable Storage for this upload. The object's
// key will be generated from its content hash; the key specified in To() will
// be ignored.
//
// Takes algorithm (string) which specifies the hash algorithm to use, either
// "sha256" (default) or "md5".
// Takes expectedHash (...string) which optionally provides a hash to verify
// against; the upload will fail if the computed hash does not match.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) CAS(algorithm string, expectedHash ...string) *UploadBuilder {
	b.params.UseContentAddressing = true
	b.params.HashAlgorithm = algorithm
	if len(expectedHash) > 0 {
		b.params.ExpectedHash = expectedHash[0]
	}
	return b
}

// Transformer adds a stream transformer to the upload pipeline by name.
// Call this method multiple times to chain transformers in order of priority.
//
// Takes name (string) which is the registered name of the transformer.
// Takes options (...any) which provides transformer-specific options
// such as an encryption key. Only the first option is used if provided.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) Transformer(name string, options ...any) *UploadBuilder {
	if b.params.TransformConfig == nil {
		b.params.TransformConfig = &storage_dto.TransformConfig{
			EnabledTransformers: []string{},
			TransformerOptions:  make(map[string]any),
		}
	}
	b.params.TransformConfig.EnabledTransformers = append(b.params.TransformConfig.EnabledTransformers, name)
	if len(options) > 0 && options[0] != nil {
		b.params.TransformConfig.TransformerOptions[name] = options[0]
	}
	return b
}

// Multipart configures and enables multipart uploading, which is ideal
// for large files. If not called, the service may still enable multipart
// automatically based on file size.
//
// Takes config (storage_dto.MultipartUploadConfig) which specifies the
// multipart upload settings.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) Multipart(config storage_dto.MultipartUploadConfig) *UploadBuilder {
	b.params.MultipartConfig = &config
	return b
}

// Dispatch queues the upload for asynchronous processing via the storage
// dispatcher. If no dispatcher is configured, this method has no effect and
// the upload remains synchronous.
//
// Returns *UploadBuilder which allows method chaining.
func (b *UploadBuilder) Dispatch() *UploadBuilder {
	b.useDispatcher = true
	return b
}

// Do runs the upload operation that was set up with this builder.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// By default, the operation runs at once and waits to finish. Use Dispatch to
// queue it for later instead.
//
// Returns error when the upload fails or the dispatcher cannot queue the
// operation.
func (b *UploadBuilder) Do(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("uploading object: %w", err)
	}

	s := b.service

	name := b.providerName
	if name == "" {
		name = storage_dto.StorageProviderDefault
	}

	if b.useDispatcher && s.dispatcher != nil {
		return s.dispatcher.QueuePut(ctx, b.params)
	}

	return s.PutObject(ctx, name, b.params)
}

// Build finalises the construction of the upload spec for bulk upload
// operations.
//
// This method consumes the reader and can only be called once.
//
// Returns storage_dto.PutObjectSpec which contains the configured upload
// specification.
// Returns error when the reader has already been consumed or was never
// provided.
func (b *UploadBuilder) Build() (storage_dto.PutObjectSpec, error) {
	if b.params.Reader == nil {
		return storage_dto.PutObjectSpec{}, errors.New("reader has already been consumed or was never provided")
	}
	spec := storage_dto.PutObjectSpec{
		Key:         b.params.Key,
		Reader:      b.params.Reader,
		ContentType: b.params.ContentType,
		Size:        b.params.Size,
	}
	b.params.Reader = nil
	return spec, nil
}

// Clone creates a deep copy of the UploadBuilder for use as a template.
//
// Use it to create an upload template that can be modified and used
// multiple times. The underlying io.Reader is not cloned and will be
// nil in the new builder. You must provide a new reader to the cloned
// builder before executing the upload.
//
// Returns *UploadBuilder which is the deep copy with a nil reader.
func (b *UploadBuilder) Clone() *UploadBuilder {
	clonedBuilder := *b

	paramsCopy := *b.params
	paramsCopy.Reader = nil

	if b.params.Metadata != nil {
		paramsCopy.Metadata = make(map[string]string, len(b.params.Metadata))
		maps.Copy(paramsCopy.Metadata, b.params.Metadata)
	}

	if b.params.TransformConfig != nil {
		tcCopy := *b.params.TransformConfig
		tcCopy.EnabledTransformers = append([]string(nil), tcCopy.EnabledTransformers...)
		tcCopy.TransformerOptions = make(map[string]any, len(b.params.TransformConfig.TransformerOptions))
		maps.Copy(tcCopy.TransformerOptions, b.params.TransformConfig.TransformerOptions)
		paramsCopy.TransformConfig = &tcCopy
	}

	clonedBuilder.params = &paramsCopy
	return &clonedBuilder
}
