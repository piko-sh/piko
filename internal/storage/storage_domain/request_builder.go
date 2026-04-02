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
	"fmt"
	"io"
	"maps"

	"piko.sh/piko/internal/storage/storage_dto"
)

// RequestBuilder provides a fluent API for working with storage objects.
// It is the main entry point for Get, Stat, Remove, and Hash actions.
//
// Usage:
// // Get an object
// reader, err := service.NewRequest(repo, key).Get(ctx)
// // Remove an object
// err := service.NewRequest(repo, key).Remove(ctx)
// // Stat an object
// info, err := service.NewRequest(repo, key).Stat(ctx)
type RequestBuilder struct {
	// service provides access to object storage operations.
	service *service

	// params holds the options for getting objects, such as byte range and transforms.
	params storage_dto.GetParams

	// providerName specifies the storage provider; empty uses the default.
	providerName string

	// useDispatcher enables the dispatch queue for Remove operations.
	useDispatcher bool
}

// NewRequest creates a request builder for an operation on an existing object.
//
// Takes repo (string) which specifies the repository name.
// Takes key (string) which identifies the object within the repository.
//
// Returns *RequestBuilder which is ready to be configured and executed.
func (s *service) NewRequest(repo string, key string) *RequestBuilder {
	return &RequestBuilder{
		service:       s,
		providerName:  "",
		useDispatcher: false,
		params: storage_dto.GetParams{
			Repository:      repo,
			Key:             key,
			ByteRange:       nil,
			TransformConfig: nil,
		},
	}
}

// Provider sets the storage provider to use for this request.
//
// Takes name (string) which identifies the storage provider.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) Provider(name string) *RequestBuilder {
	b.providerName = name
	return b
}

// ByteRange specifies a byte range for a partial download.
//
// This only applies to Get operations. Use end=-1 to read to the end of the
// file.
//
// Takes start (int64) which is the starting byte offset (inclusive).
// Takes end (int64) which is the ending byte offset (inclusive).
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) ByteRange(start int64, end int64) *RequestBuilder {
	b.params.ByteRange = &storage_dto.ByteRange{
		Start: start,
		End:   end,
	}
	return b
}

// Transformer sets options for reversing a transformation.
//
// This is typically only needed when the reversal cannot be worked out
// from metadata alone, for example when providing a decryption key.
//
// Takes name (string) which identifies the transformer to set up.
// Takes options (any) which specifies the settings for that transformer.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) Transformer(name string, options any) *RequestBuilder {
	if b.params.TransformConfig == nil {
		b.params.TransformConfig = &storage_dto.TransformConfig{
			EnabledTransformers: []string{},
			TransformerOptions:  make(map[string]any),
		}
	}
	b.params.TransformConfig.TransformerOptions[name] = options
	return b
}

// DispatchRemove queues the Remove operation for asynchronous processing.
// This only applies to the Remove() terminal method.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) DispatchRemove() *RequestBuilder {
	b.useDispatcher = true
	return b
}

// Get executes the Get operation and returns a readable stream of the object's
// content. The caller must close the returned io.ReadCloser.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns io.ReadCloser which provides the object's content as a stream.
// Returns error when the storage provider cannot retrieve the object.
func (b *RequestBuilder) Get(ctx context.Context) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("getting object %q from repo %q: %w", b.params.Key, b.params.Repository, err)
	}

	name := b.providerName
	if name == "" {
		name = storage_dto.StorageProviderDefault
	}

	return b.service.GetObject(ctx, name, b.params)
}

// Stat runs the stat operation and returns the object's metadata without its
// content.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns *ObjectInfo which contains the object's metadata.
// Returns error when the stat operation fails.
func (b *RequestBuilder) Stat(ctx context.Context) (*ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("statting object %q from repo %q: %w", b.params.Key, b.params.Repository, err)
	}

	name := b.providerName
	if name == "" {
		name = storage_dto.StorageProviderDefault
	}

	return b.service.StatObject(ctx, b.providerName, b.params)
}

// Remove deletes the object from storage.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Use DispatchRemove before calling this to make the operation run in the
// background.
//
// Returns error when the removal fails or the dispatcher rejects the request.
func (b *RequestBuilder) Remove(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("removing object %q from repo %q: %w", b.params.Key, b.params.Repository, err)
	}

	name := b.providerName
	if name == "" {
		name = storage_dto.StorageProviderDefault
	}

	s := b.service
	if b.useDispatcher && s.dispatcher != nil {
		return s.dispatcher.QueueRemove(ctx, b.params)
	}
	return s.RemoveObject(ctx, name, b.params)
}

// Hash returns a hash of the object's content.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns string which is the hash of the object's content.
// Returns error when the hash operation fails.
func (b *RequestBuilder) Hash(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	name := b.providerName
	if name == "" {
		name = storage_dto.StorageProviderDefault
	}

	return b.service.GetObjectHash(ctx, name, b.params)
}

// Clone creates a deep copy of the RequestBuilder.
//
// Use it to create a template for performing multiple actions on the same
// object.
//
// Returns *RequestBuilder which is an independent copy with its own params.
func (b *RequestBuilder) Clone() *RequestBuilder {
	clonedBuilder := *b

	paramsCopy := b.params

	if b.params.ByteRange != nil {
		paramsCopy.ByteRange = new(*b.params.ByteRange)
	}

	if b.params.TransformConfig != nil {
		tcCopy := *b.params.TransformConfig
		tcCopy.TransformerOptions = make(map[string]any, len(b.params.TransformConfig.TransformerOptions))
		maps.Copy(tcCopy.TransformerOptions, b.params.TransformConfig.TransformerOptions)
		paramsCopy.TransformConfig = &tcCopy
	}

	clonedBuilder.params = paramsCopy
	return &clonedBuilder
}
