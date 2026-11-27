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

//go:build !js

package daemon_frontend

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// EmbeddedFrontendTemplates contains the embedded frontend JavaScript and CSS assets.
	//
	//go:embed built/*
	EmbeddedFrontendTemplates embed.FS

	// assetStore holds the in-memory cache of embedded frontend assets keyed by path.
	assetStore map[string]*EmbeddedAsset
)

// EmbeddedAsset represents a cached frontend asset with its metadata.
type EmbeddedAsset struct {
	// ETag is the entity tag used for cache validation with If-None-Match headers.
	ETag string

	// MimeType is the MIME type used for the Content-Type header.
	MimeType string

	// Encoding is the content encoding (e.g. "br", "gzip") or empty if uncompressed.
	Encoding string

	// Content holds the raw bytes of the embedded asset.
	Content []byte

	// Embedded is true when the asset is one of the framework bundles built into the binary,
	// whose bytes the runtime version fingerprints.
	//
	// Only these may be served under a versioned URL. Custom modules registered at startup
	// change with each app release, so they are false.
	Embedded bool
}

// InitAssetStore sets up the in-memory asset cache by walking the embedded filesystem and
// loading all assets with their ETags and MIME types.
//
// Returns error when the embedded filesystem cannot be walked.
func InitAssetStore(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "InitEmbeddedAssetCache",
		logger_domain.String(logger_domain.FieldStrComponent, "EmbeddedAssetCache"),
	)
	defer span.End()

	assetCacheInitCount.Add(ctx, 1)
	l.Internal("Initialising embedded asset cache via init()...")

	store := make(map[string]*EmbeddedAsset)

	err := l.RunInSpan(ctx, "WalkEmbeddedFS", func(ctx context.Context, _ logger_domain.Logger) error {
		return fs.WalkDir(EmbeddedFrontendTemplates, "built", func(assetPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("walking embedded asset path %q: %w", assetPath, err)
			}
			if d.IsDir() || strings.HasSuffix(assetPath, ".etag") || strings.HasSuffix(assetPath, ".sri") {
				return nil
			}

			asset, loadErr := loadEmbeddedAsset(ctx, assetPath)
			if loadErr != nil {
				return nil
			}
			store[assetPath] = asset
			return nil
		})
	})

	if err != nil {
		assetCacheInitErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "Failed to walk embedded filesystem to build asset cache")
		return fmt.Errorf("walking embedded filesystem to build asset cache: %w", err)
	}

	assetStore = store

	populateSRIHashes(ctx, store)

	if versionErr := runtimeVersionError(); versionErr != nil {
		l.Warn("Cannot compute the framework runtime version, so pages will use unversioned bundle URLs",
			logger_domain.Error(versionErr),
		)
	}

	assetCacheSize.Add(ctx, int64(len(assetStore)))
	l.Internal("Embedded asset cache initialised successfully",
		logger_domain.Int("itemCount", len(assetStore)),
	)

	return nil
}

// populateSRIHashes computes and stores SRI hashes for uncompressed, non-source-map
// assets.
//
// Takes store (map[string]*EmbeddedAsset) which provides the assets to compute hashes
// for.
func populateSRIHashes(ctx context.Context, store map[string]*EmbeddedAsset) {
	for assetPath, asset := range store {
		if asset.Encoding == "" && !strings.HasSuffix(assetPath, ".map") {
			SetSRIHash(assetPath, loadAssetSRI(ctx, assetPath, asset.Content))
		}
	}
}

// GetAsset retrieves an asset from the cache by its path.
//
// Takes assetPath (string) which is the path to look up in the cache.
//
// Returns *EmbeddedAsset which is the cached asset, or nil if not found.
// Returns bool which is true if the asset exists, false otherwise.
func GetAsset(ctx context.Context, assetPath string) (*EmbeddedAsset, bool) {
	ctx, span, l := log.Span(ctx, "GetAsset",
		logger_domain.String(logger_domain.FieldStrPath, assetPath),
	)
	defer span.End()

	assetCacheReadCount.Add(ctx, 1)

	asset, found := assetStore[assetPath]
	if !found {
		assetCacheMissCount.Add(ctx, 1)
		l.Trace("Asset cache miss", logger_domain.String(logger_domain.FieldStrPath, assetPath))
	}

	return asset, found
}

// DetermineBestAssetPath selects the optimal compressed variant of an asset based on the
// client's Accept-Encoding header. Prefers Brotli, then Gzip, then falls back to the
// uncompressed version.
//
// Takes basePath (string) which is the path to the uncompressed asset.
// Takes acceptEncoding (string) which is the client's Accept-Encoding header.
//
// Returns string which is the path to the best available asset variant.
func DetermineBestAssetPath(ctx context.Context, basePath, acceptEncoding string) string {
	ctx, span, l := log.Span(ctx, "DetermineBestAssetPath",
		logger_domain.String(logger_domain.FieldStrPath, basePath),
		logger_domain.String("acceptEncoding", acceptEncoding),
	)
	defer span.End()

	if strings.Contains(acceptEncoding, "br") {
		brPath := basePath + ".br"
		if _, ok := assetStore[brPath]; ok {
			l.Trace("Using Brotli compressed asset",
				logger_domain.String(logger_domain.FieldStrPath, brPath),
			)
			return brPath
		}
	}

	if strings.Contains(acceptEncoding, "gzip") || strings.Contains(acceptEncoding, "gz") {
		gzPath := basePath + ".gz"
		if _, ok := assetStore[gzPath]; ok {
			l.Trace("Using Gzip compressed asset",
				logger_domain.String(logger_domain.FieldStrPath, gzPath),
			)
			return gzPath
		}
	}

	l.Trace("Using uncompressed asset",
		logger_domain.String(logger_domain.FieldStrPath, basePath),
	)
	return basePath
}

// RegisterCustomModule adds a custom frontend module to the asset store.
//
// Call this during setup, after InitAssetStore. The module will be served at
// /_piko/dist/ppframework.{name}.min.js
//
// Takes name (string) which identifies the module in the URL path.
// Takes content ([]byte) which provides the JavaScript source code.
// Takes etag (string) which sets the ETag header for cache checking.
//
// Returns error when the asset store has not been initialised.
func RegisterCustomModule(ctx context.Context, name string, content []byte, etag string) error {
	ctx, span, l := log.Span(ctx, "RegisterCustomModule",
		logger_domain.String("module_name", name),
	)
	defer span.End()

	if assetStore == nil {
		err := errors.New("asset store not initialised; call InitAssetStore first")
		l.ReportError(span, err, "Cannot register custom module")
		return fmt.Errorf("registering custom module %q: %w", name, err)
	}

	assetPath := fmt.Sprintf("built/ppframework.%s.min.js", name)
	mimeType := "application/javascript; charset=utf-8"

	assetStore[assetPath] = &EmbeddedAsset{
		Content:  content,
		ETag:     etag,
		MimeType: mimeType,
		Encoding: "",
		Embedded: false,
	}

	SetSRIHash(assetPath, ComputeSRIHash(content))

	l.Internal("Custom frontend module registered",
		logger_domain.String("asset_path", assetPath),
		logger_domain.Int("content_size", len(content)),
	)

	return nil
}

// loadEmbeddedAsset reads and parses a single asset from the embedded file system.
//
// Takes assetPath (string) which specifies the path to the embedded asset.
//
// Returns *EmbeddedAsset which contains the asset content, ETag, MIME type, and encoding.
// Returns error when the asset cannot be read from the embedded file system.
func loadEmbeddedAsset(ctx context.Context, assetPath string) (*EmbeddedAsset, error) {
	ctx, l := logger_domain.From(ctx, log)

	contentBytes, readErr := EmbeddedFrontendTemplates.ReadFile(assetPath)
	if readErr != nil {
		l.Warn("Failed to read embedded asset, skipping cache",
			logger_domain.String(logger_domain.FieldStrPath, assetPath),
			logger_domain.String(logger_domain.FieldStrError, readErr.Error()),
		)
		assetCacheReadErrorCount.Add(ctx, 1)
		return nil, readErr
	}

	etag := loadAssetETag(ctx, EmbeddedFrontendTemplates, assetPath, contentBytes)
	uncompressedPath := strings.TrimSuffix(strings.TrimSuffix(assetPath, ".br"), ".gz")
	mimeType := getMimeType(filepath.Ext(uncompressedPath))
	encoding := getEncodingFromPath(assetPath)

	return &EmbeddedAsset{
		Content:  contentBytes,
		ETag:     etag,
		MimeType: mimeType,
		Encoding: encoding,
		Embedded: true,
	}, nil
}

// loadAssetETag reads the ETag for an asset from its matching .etag file.
//
// When the file is missing or empty, the ETag is derived from the asset's bytes instead.
// Browsers revalidate these assets on every use, so the tag must change whenever the
// bytes do. A tag built from the file name would answer 304 to a copy from an earlier
// release.
//
// Takes assetFS (fs.ReadFileFS) which holds the asset and its sidecar files.
// Takes assetPath (string) which is the path to the asset.
// Takes content ([]byte) which is the asset's bytes, used when no sidecar can be read.
//
// Returns string which is the ETag from the sidecar or one derived from the content.
func loadAssetETag(ctx context.Context, assetFS fs.ReadFileFS, assetPath string, content []byte) string {
	_, l := logger_domain.From(ctx, log)

	etagBytes, err := assetFS.ReadFile(assetPath + ".etag")
	if err != nil {
		l.Warn("Missing .etag file for embedded asset, deriving it from the content",
			logger_domain.String(logger_domain.FieldStrPath, assetPath),
			logger_domain.Error(err),
		)
		return computeETag(content)
	}

	etag := strings.TrimSpace(string(etagBytes))
	if etag == "" {
		l.Warn("Empty .etag file for embedded asset, deriving it from the content",
			logger_domain.String(logger_domain.FieldStrPath, assetPath),
		)
		return computeETag(content)
	}
	return etag
}

// loadAssetSRI reads the SRI hash for an asset from its matching .sri sidecar file. Falls
// back to computing the hash at runtime if the .sri file is missing.
//
// Takes assetPath (string) which is the path to the embedded asset.
// Takes content ([]byte) which is the uncompressed asset content used as fallback when no
// .sri file exists.
//
// Returns string which is the SRI hash from the file or a computed one.
func loadAssetSRI(ctx context.Context, assetPath string, content []byte) string {
	ctx, l := logger_domain.From(ctx, log)

	sriPath := assetPath + ".sri"
	sriBytes, sriErr := EmbeddedFrontendTemplates.ReadFile(sriPath)
	if sriErr == nil {
		return string(sriBytes)
	}

	l.Warn("Missing .sri file for embedded asset, computing at runtime",
		logger_domain.String(logger_domain.FieldStrPath, assetPath),
	)
	return ComputeSRIHash(content)
}

// getMimeType returns the MIME type for a file extension.
//
// Takes fileExt (string) which is the file extension including the leading dot (e.g.
// ".js", ".css").
//
// Returns string which is the MIME type with charset where needed, or
// "application/octet-stream" for unknown extensions.
func getMimeType(fileExt string) string {
	switch strings.ToLower(fileExt) {
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// getEncodingFromPath returns the content encoding based on the file extension.
//
// Takes p (string) which is the file path to check.
//
// Returns string which is "br" for Brotli files, "gzip" for gzip files, or empty for
// files without compression.
func getEncodingFromPath(p string) string {
	if strings.HasSuffix(p, ".br") {
		return "br"
	}
	if strings.HasSuffix(p, ".gz") {
		return "gzip"
	}
	return ""
}
