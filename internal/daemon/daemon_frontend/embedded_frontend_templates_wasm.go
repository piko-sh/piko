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

//go:build js

package daemon_frontend

import "context"

// EmbeddedAsset is the WASM-build counterpart of the disk-build type.
//
// It exists so daemon_adapters and other callers that only dispatch on
// (asset, found) results compile in WASM mode without needing their own
// build-tag splits; the stubs below always return (nil, false), so
// callers fall through to whatever 404/default path they already have.
type EmbeddedAsset struct {
	// ETag is the HTTP ETag header value identifying the asset version.
	ETag string

	// MimeType is the asset's IANA media type (e.g. "text/javascript").
	MimeType string

	// Encoding identifies a content-encoding (e.g. "br", "gzip", "")
	// when the asset is pre-compressed.
	Encoding string

	// Content holds the asset bytes ready to be served.
	Content []byte
}

// GetAsset is a no-op in WASM builds because the asset store
// (InitAssetStore) and the heavier embed.FS that backs it both live
// behind `//go:build !js` to keep the WASM binary lean. The framework
// runtime sources that the playground iframe needs are exposed
// separately via FrameworkCore and FrameworkComponents in
// runtime_source.go.
//
// Returns *EmbeddedAsset which is always nil.
// Returns bool which is always false to signal "not found".
func GetAsset(_ context.Context, _ string) (*EmbeddedAsset, bool) {
	return nil, false
}

// DetermineBestAssetPath returns the base path unchanged in WASM builds.
// The full implementation negotiates Accept-Encoding against
// pre-compressed .br/.gz variants in the asset store; that store doesn't
// exist in WASM (see GetAsset).
//
// Takes basePath (string) which is returned as-is.
//
// Returns string equal to basePath.
func DetermineBestAssetPath(_ context.Context, basePath, _ string) string {
	return basePath
}

// InitAssetStore is a no-op in WASM builds; the asset store is part of
// the disk-build embedded_frontend_templates.go file (`//go:build !js`).
//
// Returns error which is always nil.
func InitAssetStore(_ context.Context) error {
	return nil
}

// RegisterCustomModule is a no-op in WASM builds; there is no asset
// store to register against.
//
// Returns error which is always nil.
func RegisterCustomModule(_ context.Context, _ string, _ []byte, _ string) error {
	return nil
}
