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

package provider_union

import (
	"context"
	"errors"
	"io"

	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

var (
	// ErrReadOnly is returned by every write when the union has no writable overlay.
	ErrReadOnly = errors.New("union storage provider has no writable overlay")

	// ErrNoLayer is returned by a read when the union has neither a base nor an overlay.
	ErrNoLayer = errors.New("union storage provider has neither a base nor an overlay")
)

// keyLister is the optional key-listing capability, used by garbage collection. It is not
// on StorageProviderPort, so the union forwards it by type assertion.
type keyLister interface {
	// ListKeys returns every storage key in the repository.
	//
	// Takes repository (string) which names the repository to scan.
	//
	// Returns []string which are the storage keys found.
	// Returns error when the listing fails.
	ListKeys(ctx context.Context, repository string) ([]string, error)
}

// manyRemover is the optional batch-remove capability, forwarded by type assertion.
type manyRemover interface {
	// RemoveMany deletes the batch of keys and reports the outcome.
	//
	// Takes params (storage_dto.RemoveManyParams) which describes the objects to remove.
	//
	// Returns *storage_dto.BatchResult which reports the outcome of each removal.
	// Returns error when the batch removal fails.
	RemoveMany(ctx context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error)
}

// providerDescriber is the optional provider-metadata capability, forwarded by assertion.
type providerDescriber interface {
	// GetProviderType names the underlying provider.
	//
	// Returns string which is the provider name.
	GetProviderType() string

	// GetProviderMetadata describes the underlying provider.
	//
	// Returns map[string]any which holds the provider's descriptive fields.
	GetProviderMetadata() map[string]any
}

// batchSupporter is the optional batch-operations capability, forwarded by type
// assertion.
type batchSupporter interface {
	// SupportsBatchOperations reports whether the provider has native batch operations.
	//
	// Returns bool which reports whether native batch operations exist.
	SupportsBatchOperations() bool
}

// presignSupporter is the optional presigned-URL capability, forwarded by type assertion.
type presignSupporter interface {
	// SupportsPresignedURLs reports whether the provider can generate presigned URLs.
	//
	// Returns bool which reports whether presigned URLs can be generated.
	SupportsPresignedURLs() bool
}

// UnionProvider presents a read-only base and a writable overlay as one storage provider.
type UnionProvider struct {
	// base is the read-only layer (the embedded filesystem), or nil.
	base storage_domain.StorageProviderPort

	// overlay is the writable layer (disk or object store), or nil for a read-only
	// deployment.
	overlay storage_domain.StorageProviderPort
}

var (
	_ storage_domain.StorageProviderPort = (*UnionProvider)(nil)
)

// New composes a base and an overlay into one storage provider.
//
// At least one layer should be present. With no base it returns the overlay untouched;
// with no overlay every write returns ErrReadOnly and reads serve the base alone. As a
// guard against a deferred nil dereference, New(nil, nil) returns a degenerate read-only
// provider whose reads return ErrNoLayer and whose writes return ErrReadOnly, never a nil
// interface.
//
// Takes base (storage_domain.StorageProviderPort) which is the read-only layer, or nil.
// Takes overlay (storage_domain.StorageProviderPort) which is the writable layer, or nil.
//
// Returns storage_domain.StorageProviderPort which is the composed provider, never nil.
func New(base, overlay storage_domain.StorageProviderPort) storage_domain.StorageProviderPort {
	if base == nil {
		if overlay == nil {
			return &UnionProvider{base: nil, overlay: nil}
		}
		return overlay
	}
	return &UnionProvider{base: base, overlay: overlay}
}

// Get returns the object, from the base when it has it, otherwise from the overlay.
//
// Takes params (storage_dto.GetParams) which names the object.
//
// Returns io.ReadCloser which streams the object.
// Returns error when neither layer has it.
func (p *UnionProvider) Get(ctx context.Context, params storage_dto.GetParams) (io.ReadCloser, error) {
	if p.baseHas(ctx, params) {
		return p.base.Get(ctx, params)
	}
	if p.overlay != nil {
		return p.overlay.Get(ctx, params)
	}
	if p.base != nil {
		return p.base.Get(ctx, params)
	}
	return nil, ErrNoLayer
}

// Stat returns object metadata, from the base when it has it, otherwise from the overlay.
//
// Takes params (storage_dto.GetParams) which names the object.
//
// Returns *storage_domain.ObjectInfo which describes the object.
// Returns error when neither layer has it.
func (p *UnionProvider) Stat(ctx context.Context, params storage_dto.GetParams) (*storage_domain.ObjectInfo, error) {
	if p.baseHas(ctx, params) {
		return p.base.Stat(ctx, params)
	}
	if p.overlay != nil {
		return p.overlay.Stat(ctx, params)
	}
	if p.base != nil {
		return p.base.Stat(ctx, params)
	}
	return nil, ErrNoLayer
}

// Exists reports whether either layer holds the object.
//
// Takes params (storage_dto.GetParams) which names the object.
//
// Returns bool which is true when either layer has it.
// Returns error when the overlay existence check fails.
func (p *UnionProvider) Exists(ctx context.Context, params storage_dto.GetParams) (bool, error) {
	if p.baseHas(ctx, params) {
		return true, nil
	}
	if p.overlay == nil {
		return false, nil
	}
	return p.overlay.Exists(ctx, params)
}

// GetHash returns the object's hash, from the base when it has it, otherwise the overlay.
//
// Takes params (storage_dto.GetParams) which names the object.
//
// Returns string which is the object's hash.
// Returns error when neither layer has it.
func (p *UnionProvider) GetHash(ctx context.Context, params storage_dto.GetParams) (string, error) {
	if p.baseHas(ctx, params) {
		return p.base.GetHash(ctx, params)
	}
	if p.overlay != nil {
		return p.overlay.GetHash(ctx, params)
	}
	if p.base != nil {
		return p.base.GetHash(ctx, params)
	}
	return "", ErrNoLayer
}

// ListKeys returns the deduplicated union of both layers' keys for a repository.
//
// Takes repository (string) which is the repository to list.
//
// Returns []string which is every key across both layers, once each.
// Returns error when a layer listing fails.
func (p *UnionProvider) ListKeys(ctx context.Context, repository string) ([]string, error) {
	seen := make(map[string]struct{})
	keys := make([]string, 0)

	appendUnique := func(source keyLister) error {
		sourceKeys, err := source.ListKeys(ctx, repository)
		if err != nil {
			return err
		}
		for _, key := range sourceKeys {
			if _, done := seen[key]; !done {
				seen[key] = struct{}{}
				keys = append(keys, key)
			}
		}
		return nil
	}

	if lister, ok := p.base.(keyLister); ok {
		if err := appendUnique(lister); err != nil {
			return nil, err
		}
	}
	if lister, ok := p.overlay.(keyLister); ok {
		if err := appendUnique(lister); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// Put writes an object to the overlay.
//
// Takes params (*storage_dto.PutParams) describing the object to write.
//
// Returns error which is ErrReadOnly when there is no overlay.
func (p *UnionProvider) Put(ctx context.Context, params *storage_dto.PutParams) error {
	if p.overlay == nil {
		return ErrReadOnly
	}
	return p.overlay.Put(ctx, params)
}

// Copy copies an object within a repository on the overlay.
//
// Takes repository (string), sourceKey (string) and destinationKey (string) naming the
// copy.
//
// Returns error which is ErrReadOnly when there is no overlay.
func (p *UnionProvider) Copy(ctx context.Context, repository, sourceKey, destinationKey string) error {
	if p.overlay == nil {
		return ErrReadOnly
	}
	return p.overlay.Copy(ctx, repository, sourceKey, destinationKey)
}

// CopyToAnotherRepository copies an object across repositories on the overlay.
//
// Takes sourceRepository, sourceKey, destinationRepository and destinationKey naming a
// copy.
//
// Returns error which is ErrReadOnly when there is no overlay.
func (p *UnionProvider) CopyToAnotherRepository(ctx context.Context, sourceRepository, sourceKey, destinationRepository, destinationKey string) error {
	if p.overlay == nil {
		return ErrReadOnly
	}
	return p.overlay.CopyToAnotherRepository(ctx, sourceRepository, sourceKey, destinationRepository, destinationKey)
}

// Remove deletes an object from the overlay, and is a no-op for a base-owned object.
//
// A blob the binary ships lives only in the read-only base and is never deleted; a remove
// request for such a key (a mistaken GC hint, say) is a safe no-op rather than an error.
//
// Takes params (storage_dto.GetParams) naming the object to remove.
//
// Returns error which is ErrReadOnly with no overlay and a non-base-owned object.
func (p *UnionProvider) Remove(ctx context.Context, params storage_dto.GetParams) error {
	if p.baseHas(ctx, params) {
		return nil
	}
	if p.overlay == nil {
		return ErrReadOnly
	}
	return p.overlay.Remove(ctx, params)
}

// Rename renames an object within a repository on the overlay.
//
// Takes repository, sourceKey and destinationKey (strings) naming the rename.
//
// Returns error which is ErrReadOnly when there is no overlay.
func (p *UnionProvider) Rename(ctx context.Context, repository, sourceKey, destinationKey string) error {
	if p.overlay == nil {
		return ErrReadOnly
	}
	return p.overlay.Rename(ctx, repository, sourceKey, destinationKey)
}

// PresignURL delegates to the overlay: a base object has no external URL to sign.
//
// Takes params (storage_dto.PresignParams) describing the upload to presign.
//
// Returns string which is the presigned URL.
// Returns error which is ErrReadOnly when there is no overlay.
func (p *UnionProvider) PresignURL(ctx context.Context, params storage_dto.PresignParams) (string, error) {
	if p.overlay == nil {
		return "", ErrReadOnly
	}
	return p.overlay.PresignURL(ctx, params)
}

// PresignDownloadURL delegates to the overlay: a base object has no external URL to sign.
//
// Takes params (storage_dto.PresignDownloadParams) describing the download to presign.
//
// Returns string which is the presigned URL.
// Returns error which is ErrReadOnly when there is no overlay.
func (p *UnionProvider) PresignDownloadURL(ctx context.Context, params storage_dto.PresignDownloadParams) (string, error) {
	if p.overlay == nil {
		return "", ErrReadOnly
	}
	return p.overlay.PresignDownloadURL(ctx, params)
}

// PutMany writes several objects to the overlay.
//
// Takes params (*storage_dto.PutManyParams) describing the objects to write.
//
// Returns *storage_dto.BatchResult which describes the outcome.
// Returns error which is ErrReadOnly when there is no overlay.
func (p *UnionProvider) PutMany(ctx context.Context, params *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	if p.overlay == nil {
		return nil, ErrReadOnly
	}
	return p.overlay.PutMany(ctx, params)
}

// RemoveMany deletes several objects from the overlay, skipping base-owned keys.
//
// Base-owned keys are filtered out before the batch reaches the overlay, so a batch
// mixing baked and runtime keys does not fail spuriously on the baked ones; this mirrors
// the single-object Remove no-op. When nothing remains for the overlay the result is an
// empty success rather than ErrReadOnly, matching that no-op semantics.
//
// Takes params (storage_dto.RemoveManyParams) describing the objects to remove.
//
// Returns *storage_dto.BatchResult which describes the outcome.
// Returns error which is ErrReadOnly when there is no overlay and overlay-owned keys
// remain.
func (p *UnionProvider) RemoveMany(ctx context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	params.Keys = p.overlayRemovableKeys(ctx, params.Repository, params.Keys)
	remover, ok := p.overlay.(manyRemover)
	if ok {
		return remover.RemoveMany(ctx, params)
	}
	if len(params.Keys) == 0 {
		return new(storage_dto.BatchResult), nil
	}
	return nil, ErrReadOnly
}

// Close closes the overlay; the base is in-memory.
//
// Returns error when the overlay close fails.
func (p *UnionProvider) Close(ctx context.Context) error {
	if p.overlay == nil {
		return nil
	}
	return p.overlay.Close(ctx)
}

// SupportsMultipart reports the overlay's multipart support, or false with no overlay.
//
// Returns bool which is true when the overlay supports multipart uploads.
func (p *UnionProvider) SupportsMultipart() bool {
	return p.overlay != nil && p.overlay.SupportsMultipart()
}

// SupportsBatchOperations reports the overlay's batch support, or false with no overlay.
//
// Returns bool which is true when the overlay supports batch operations.
func (p *UnionProvider) SupportsBatchOperations() bool {
	supporter, ok := p.overlay.(batchSupporter)
	return ok && supporter.SupportsBatchOperations()
}

// SupportsRetry reports the overlay's retry support, or false with no overlay.
//
// Returns bool which is true when the overlay supports retry.
func (p *UnionProvider) SupportsRetry() bool {
	return p.overlay != nil && p.overlay.SupportsRetry()
}

// SupportsCircuitBreaking reports whether the overlay supports circuit breaking.
//
// Returns bool which is true when the overlay supports circuit breaking.
func (p *UnionProvider) SupportsCircuitBreaking() bool {
	return p.overlay != nil && p.overlay.SupportsCircuitBreaking()
}

// SupportsRateLimiting reports whether the overlay supports rate limiting.
//
// Returns bool which is true when the overlay supports rate limiting.
func (p *UnionProvider) SupportsRateLimiting() bool {
	return p.overlay != nil && p.overlay.SupportsRateLimiting()
}

// SupportsPresignedURLs reports whether the overlay supports presigned URLs.
//
// Returns bool which is true when the overlay supports presigned URLs.
func (p *UnionProvider) SupportsPresignedURLs() bool {
	supporter, ok := p.overlay.(presignSupporter)
	return ok && supporter.SupportsPresignedURLs()
}

// GetProviderType names the composed provider.
//
// Returns string which identifies the union provider.
func (*UnionProvider) GetProviderType() string {
	return "union"
}

// GetProviderMetadata describes the union and its two layers.
//
// Returns map[string]any which describes the provider and whether it has a writable
// overlay.
func (p *UnionProvider) GetProviderMetadata() map[string]any {
	metadata := map[string]any{
		"type":     "union",
		"writable": p.overlay != nil,
	}
	if describer, ok := p.base.(providerDescriber); ok {
		metadata["base"] = describer.GetProviderType()
	}
	if describer, ok := p.overlay.(providerDescriber); ok {
		metadata["overlay"] = describer.GetProviderType()
	}
	return metadata
}

// baseHas reports whether the base holds the object named by params, without reading it.
//
// Reading via a cheap Exists first, rather than sniffing a Get error, keeps the
// fall-through decision independent of each provider's error strings, and the base is an
// in-memory embedded filesystem where Exists is nearly free and free of a time-of-check
// race.
//
// Takes params (storage_dto.GetParams) which names the object.
//
// Returns bool which is true when the base holds the object.
func (p *UnionProvider) baseHas(ctx context.Context, params storage_dto.GetParams) bool {
	if p.base == nil {
		return false
	}
	exists, err := p.base.Exists(ctx, params)
	return err == nil && exists
}

// overlayRemovableKeys drops the keys the base owns from a remove batch. A base-owned
// blob lives only in the read-only base and must never reach the overlay, where it does
// not exist and the delete would fail spuriously; this mirrors the single-object Remove
// no-op.
//
// Takes repository (string) which names the repository the keys belong to.
// Takes keys ([]string) which are the requested keys to remove.
//
// Returns []string which are only the keys the overlay may safely remove.
func (p *UnionProvider) overlayRemovableKeys(ctx context.Context, repository string, keys []string) []string {
	removable := make([]string, 0, len(keys))
	for _, key := range keys {
		params := storage_dto.GetParams{
			ByteRange:       nil,
			TransformConfig: nil,
			Key:             key,
			Repository:      repository,
		}
		if p.baseHas(ctx, params) {
			continue
		}
		removable = append(removable, key)
	}
	return removable
}
