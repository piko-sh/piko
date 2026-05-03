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

package driver_asset_registrar

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultStorageBackendID matches the identifier used by the build-time
	// asset pipeline so collection-discovered assets share the same blob
	// store as component-level assets. Hardcoding is intentional: each
	// hexagon owns the value locally to avoid a cross-hexagon import.
	defaultStorageBackendID = "local_disk_cache"

	// artefactIDPrefix namespaces collection-discovered artefact IDs so
	// they cannot collide with compile-time asset IDs. Compile-time assets
	// use their source path directly, which never starts with "collection/".
	artefactIDPrefix = "collection"

	// defaultMaxAssetSizeBytes caps how large a single collection-discovered
	// asset may be. 64 MiB is generous for diagrams and docs imagery while
	// still bounding worst-case memory during build-time registration.
	defaultMaxAssetSizeBytes int64 = 64 * 1024 * 1024
)

var (
	// ErrRegistryNotConfigured is returned when the registrar was constructed
	// without a registry.
	ErrRegistryNotConfigured = errors.New("driver_asset_registrar: registry not configured")

	// ErrSandboxRequired is returned when a caller passes a nil sandbox.
	ErrSandboxRequired = errors.New("driver_asset_registrar: sandbox required")

	// ErrEmptyPath is returned when a caller passes an empty sandbox-relative
	// path.
	ErrEmptyPath = errors.New("driver_asset_registrar: sandbox-relative path is empty")

	// ErrEmptyCollectionName is returned when a caller passes an empty
	// collection name.
	ErrEmptyCollectionName = errors.New("driver_asset_registrar: collection name is empty")

	// ErrInvalidCollectionName is returned when a collection name contains
	// path separators or traversal sequences that would bleed into the
	// artefactID namespace.
	ErrInvalidCollectionName = errors.New("driver_asset_registrar: collection name contains path separator or traversal sequence")

	// ErrAssetTooLarge is returned when the sandboxed file exceeds the
	// configured size cap. The original limit is included in the wrapped
	// error message.
	ErrAssetTooLarge = errors.New("driver_asset_registrar: asset exceeds maximum allowed size")

	// ErrNoServableVariant is returned when the registry accepts the upsert
	// but produces no servable variant for the artefactID. This points at a
	// registry-side misconfiguration rather than a caller mistake.
	ErrNoServableVariant = errors.New("driver_asset_registrar: no servable variant resolved")
)

// Option configures a RegistryBackedRegistrar at construction time.
type Option func(*RegistryBackedRegistrar)

// WithMaxAssetSizeBytes overrides the per-asset size cap. Values less than
// or equal to zero leave the default in place.
//
// Takes limit (int64) which is the new cap in bytes.
//
// Returns Option which is applied during NewRegistryBackedRegistrar.
func WithMaxAssetSizeBytes(limit int64) Option {
	return func(r *RegistryBackedRegistrar) {
		if limit > 0 {
			r.maxAssetSizeBytes = limit
		}
	}
}

// RegistryBackedRegistrar implements collection_domain.AssetRegistrar by
// streaming the file through the supplied sandbox (with a hard size cap)
// and upserting its bytes into the artefact registry. It then asks the
// registry for the servable URL that rewritten src attributes should
// point at.
//
// The adapter is safe for concurrent use: the underlying render_domain.RegistryPort
// and safedisk.Sandbox are required to be concurrency-safe by their own
// contracts, and the registrar adds no mutable state of its own.
type RegistryBackedRegistrar struct {
	// registry provides artefact upsert plus serve-path resolution.
	registry render_domain.RegistryPort

	// maxAssetSizeBytes caps the size of any single asset registered. A
	// stream read above this cap fails with ErrAssetTooLarge so we never
	// buffer unbounded bytes per file during build.
	maxAssetSizeBytes int64
}

// NewRegistryBackedRegistrar creates a RegistryBackedRegistrar.
//
// Takes registry (render_domain.RegistryPort) which provides access to the
// artefact registry. The same port already used by the render layer for
// piko:img dynamic asset registration is suitable here.
// Takes options (...Option) which tune the registrar; see WithMaxAssetSizeBytes.
//
// Returns *RegistryBackedRegistrar which is ready for use.
func NewRegistryBackedRegistrar(registry render_domain.RegistryPort, options ...Option) *RegistryBackedRegistrar {
	r := &RegistryBackedRegistrar{
		registry:          registry,
		maxAssetSizeBytes: defaultMaxAssetSizeBytes,
	}
	for _, option := range options {
		option(r)
	}
	return r
}

// RegisterCollectionAsset reads sandboxRelPath through sandbox with a hard
// size cap, upserts the bytes into the registry under a deterministic
// artefactID, and returns the serve URL. See the AssetRegistrar port for
// the full contract.
//
// The artefactID format is "collection/{collectionName}/{sandboxRelPath}"
// which is stable across builds for the same input and namespaced away
// from compile-time asset IDs.
//
// Takes sandbox (safedisk.Sandbox) which provides kernel-enforced read
// access to the source content tree.
// Takes sandboxRelPath (string) which is the cleaned path of the asset
// file relative to the sandbox root.
// Takes collectionName (string) which identifies the owning collection
// and scopes the artefactID.
//
// Returns serveURL (string) which is the absolute URL for rewritten src
// attributes.
// Returns error when validation fails, the file cannot be read, the file
// exceeds maxAssetSizeBytes, the registry upsert fails, or the registered
// artefact has no servable variant. Sentinel errors are exported above.
func (r *RegistryBackedRegistrar) RegisterCollectionAsset(
	ctx context.Context,
	sandbox safedisk.Sandbox,
	sandboxRelPath string,
	collectionName string,
) (string, error) {
	ctx, l := logger_domain.From(ctx, log)

	if r.registry == nil {
		return "", ErrRegistryNotConfigured
	}
	if sandbox == nil {
		return "", ErrSandboxRequired
	}
	if sandboxRelPath == "" {
		return "", ErrEmptyPath
	}
	if collectionName == "" {
		return "", ErrEmptyCollectionName
	}
	if strings.ContainsAny(collectionName, "/\\") || strings.Contains(collectionName, "..") {
		return "", fmt.Errorf("%w: %q", ErrInvalidCollectionName, collectionName)
	}

	data, err := r.readBoundedAsset(ctx, sandbox, sandboxRelPath)
	if err != nil {
		return "", err
	}

	artefactID := buildArtefactID(collectionName, sandboxRelPath)

	_, err = r.registry.UpsertArtefact(
		ctx,
		artefactID,
		sandboxRelPath,
		bytes.NewReader(data),
		defaultStorageBackendID,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("upserting artefact %q (source %q, collection %q): %w",
			artefactID, sandboxRelPath, collectionName, err)
	}

	serveURL := r.registry.GetArtefactServePath(ctx, artefactID)
	if serveURL == "" {
		return "", fmt.Errorf("%w: artefact %q", ErrNoServableVariant, artefactID)
	}

	l.Trace("Registered collection asset",
		logger_domain.String("collection", collectionName),
		logger_domain.String("source_path", sandboxRelPath),
		logger_domain.String("artefact_id", artefactID),
		logger_domain.String("serve_url", serveURL),
		logger_domain.Int("bytes", len(data)))

	return serveURL, nil
}

// readBoundedAsset streams a file from the sandbox through an io.LimitReader
// sized at maxAssetSizeBytes+1. A read that consumes the extra byte means
// the file was larger than the cap, so we fail with ErrAssetTooLarge
// rather than silently truncating.
//
// Takes sandbox (safedisk.Sandbox) which provides the file handle.
// Takes sandboxRelPath (string) which is the file to read.
//
// Returns []byte which contains the entire file when it fits under the cap.
// Returns error when the file cannot be opened, read, or exceeds the cap.
func (r *RegistryBackedRegistrar) readBoundedAsset(
	ctx context.Context,
	sandbox safedisk.Sandbox,
	sandboxRelPath string,
) ([]byte, error) {
	_, l := logger_domain.From(ctx, log)

	handle, err := sandbox.Open(sandboxRelPath)
	if err != nil {
		return nil, fmt.Errorf("opening asset %q from sandbox: %w", sandboxRelPath, err)
	}
	defer func() {
		if closeErr := handle.Close(); closeErr != nil {
			l.Warn("Failed to close sandboxed asset handle",
				logger_domain.String("source_path", sandboxRelPath),
				logger_domain.Error(closeErr))
		}
	}()

	limit := r.maxAssetSizeBytes
	if limit <= 0 {
		limit = defaultMaxAssetSizeBytes
	}

	data, err := io.ReadAll(io.LimitReader(handle, limit+1))
	if err != nil {
		return nil, fmt.Errorf("reading asset %q from sandbox: %w", sandboxRelPath, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %q exceeds %d bytes", ErrAssetTooLarge, sandboxRelPath, limit)
	}

	return data, nil
}

// buildArtefactID composes a deterministic artefactID from the collection
// name and the sandbox-relative path. Leading slashes on either part are
// stripped so the ID has a clean "collection/<name>/<path>" shape
// regardless of caller input.
//
// Takes collectionName (string) which identifies the owning collection.
// Takes sandboxRelPath (string) which is the cleaned path of the asset
// inside the sandbox.
//
// Returns string which is the artefact identifier.
func buildArtefactID(collectionName, sandboxRelPath string) string {
	collectionName = strings.Trim(collectionName, "/")
	sandboxRelPath = strings.TrimLeft(sandboxRelPath, "/")
	if collectionName == "" {
		return path.Join(artefactIDPrefix, sandboxRelPath)
	}
	return path.Join(artefactIDPrefix, collectionName, sandboxRelPath)
}

var _ collection_domain.AssetRegistrar = (*RegistryBackedRegistrar)(nil)
