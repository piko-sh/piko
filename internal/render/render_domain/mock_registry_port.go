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

package render_domain

import (
	"context"
	"io"
	"sync/atomic"

	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_dto"
)

// MockRegistryPort is a test double for RegistryPort that returns zero
// values from nil function fields and tracks call counts atomically.
type MockRegistryPort struct {
	// GetComponentMetadataFunc is the function called by
	// GetComponentMetadata.
	GetComponentMetadataFunc func(ctx context.Context, componentType string) (*render_dto.ComponentMetadata, error)

	// BulkGetComponentMetadataFunc is the function
	// called by BulkGetComponentMetadata.
	BulkGetComponentMetadataFunc func(ctx context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error)

	// GetAssetRawSVGFunc is the function called by
	// GetAssetRawSVG.
	GetAssetRawSVGFunc func(ctx context.Context, assetID string) (*ParsedSvgData, error)

	// BulkGetAssetRawSVGFunc is the function called by
	// BulkGetAssetRawSVG.
	BulkGetAssetRawSVGFunc func(ctx context.Context, assetIDs []string) (map[string]*ParsedSvgData, error)

	// GetStatsFunc is the function called by GetStats.
	GetStatsFunc func() RegistryAdapterStats

	// ClearComponentCacheFunc is the function called by
	// ClearComponentCache.
	ClearComponentCacheFunc func(ctx context.Context, componentType string)

	// ClearSvgCacheFunc is the function called by
	// ClearSvgCache.
	ClearSvgCacheFunc func(ctx context.Context, svgID string)

	// UpsertArtefactFunc is the function called by
	// UpsertArtefact.
	UpsertArtefactFunc func(
		ctx context.Context, artefactID string, sourcePath string, sourceData io.Reader,
		storageBackendID string, desiredProfiles []registry_dto.NamedProfile,
	) (*registry_dto.ArtefactMeta, error)

	// GetComponentMetadataCallCount tracks how many
	// times GetComponentMetadata was called.
	GetComponentMetadataCallCount int64

	// BulkGetComponentMetadataCallCount tracks how many
	// times BulkGetComponentMetadata was called.
	BulkGetComponentMetadataCallCount int64

	// GetAssetRawSVGCallCount tracks how many times
	// GetAssetRawSVG was called.
	GetAssetRawSVGCallCount int64

	// BulkGetAssetRawSVGCallCount tracks how many times
	// BulkGetAssetRawSVG was called.
	BulkGetAssetRawSVGCallCount int64

	// GetStatsCallCount tracks how many times GetStats
	// was called.
	GetStatsCallCount int64

	// ClearComponentCacheCallCount tracks how many
	// times ClearComponentCache was called.
	ClearComponentCacheCallCount int64

	// ClearSvgCacheCallCount tracks how many times
	// ClearSvgCache was called.
	ClearSvgCacheCallCount int64

	// UpsertArtefactCallCount tracks how many times
	// UpsertArtefact was called.
	UpsertArtefactCallCount int64
}

var _ RegistryPort = (*MockRegistryPort)(nil)

// GetComponentMetadata delegates to GetComponentMetadataFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes componentType (string) which identifies the component type to look up.
//
// Returns (nil, nil) if GetComponentMetadataFunc is nil.
func (m *MockRegistryPort) GetComponentMetadata(ctx context.Context, componentType string) (*render_dto.ComponentMetadata, error) {
	atomic.AddInt64(&m.GetComponentMetadataCallCount, 1)
	if m.GetComponentMetadataFunc != nil {
		return m.GetComponentMetadataFunc(ctx, componentType)
	}
	return nil, nil
}

// BulkGetComponentMetadata delegates to BulkGetComponentMetadataFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes componentTypes ([]string) which lists the component types to look up.
//
// Returns (nil, nil) if BulkGetComponentMetadataFunc is nil.
func (m *MockRegistryPort) BulkGetComponentMetadata(ctx context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error) {
	atomic.AddInt64(&m.BulkGetComponentMetadataCallCount, 1)
	if m.BulkGetComponentMetadataFunc != nil {
		return m.BulkGetComponentMetadataFunc(ctx, componentTypes)
	}
	return nil, nil
}

// GetAssetRawSVG delegates to GetAssetRawSVGFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes assetID (string) which identifies the SVG asset to retrieve.
//
// Returns (nil, nil) if GetAssetRawSVGFunc is nil.
func (m *MockRegistryPort) GetAssetRawSVG(ctx context.Context, assetID string) (*ParsedSvgData, error) {
	atomic.AddInt64(&m.GetAssetRawSVGCallCount, 1)
	if m.GetAssetRawSVGFunc != nil {
		return m.GetAssetRawSVGFunc(ctx, assetID)
	}
	return nil, nil
}

// BulkGetAssetRawSVG delegates to BulkGetAssetRawSVGFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes assetIDs ([]string) which lists the SVG asset IDs to retrieve.
//
// Returns (nil, nil) if BulkGetAssetRawSVGFunc is nil.
func (m *MockRegistryPort) BulkGetAssetRawSVG(ctx context.Context, assetIDs []string) (map[string]*ParsedSvgData, error) {
	atomic.AddInt64(&m.BulkGetAssetRawSVGCallCount, 1)
	if m.BulkGetAssetRawSVGFunc != nil {
		return m.BulkGetAssetRawSVGFunc(ctx, assetIDs)
	}
	return nil, nil
}

// GetStats delegates to GetStatsFunc if set.
//
// Returns the zero value if GetStatsFunc is nil.
func (m *MockRegistryPort) GetStats() RegistryAdapterStats {
	atomic.AddInt64(&m.GetStatsCallCount, 1)
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}
	return RegistryAdapterStats{}
}

// ClearComponentCache delegates to ClearComponentCacheFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes componentType (string) which identifies the
// component cache entry to clear.
//
// Does nothing if ClearComponentCacheFunc is nil.
func (m *MockRegistryPort) ClearComponentCache(ctx context.Context, componentType string) {
	atomic.AddInt64(&m.ClearComponentCacheCallCount, 1)
	if m.ClearComponentCacheFunc != nil {
		m.ClearComponentCacheFunc(ctx, componentType)
	}
}

// ClearSvgCache delegates to ClearSvgCacheFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes svgID (string) which identifies the SVG cache entry to clear.
//
// Does nothing if ClearSvgCacheFunc is nil.
func (m *MockRegistryPort) ClearSvgCache(ctx context.Context, svgID string) {
	atomic.AddInt64(&m.ClearSvgCacheCallCount, 1)
	if m.ClearSvgCacheFunc != nil {
		m.ClearSvgCacheFunc(ctx, svgID)
	}
}

// UpsertArtefact delegates to UpsertArtefactFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactID (string) which identifies the artefact to create or update.
// Takes sourcePath (string) which is the original path of the source file.
// Takes sourceData (io.Reader) which provides the source data to store.
// Takes storageBackendID (string) which identifies the storage backend to use.
// Takes desiredProfiles ([]registry_dto.NamedProfile)
// which lists the processing profiles to apply.
//
// Returns (nil, nil) if UpsertArtefactFunc is nil.
func (m *MockRegistryPort) UpsertArtefact(
	ctx context.Context, artefactID string, sourcePath string,
	sourceData io.Reader, storageBackendID string,
	desiredProfiles []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.UpsertArtefactCallCount, 1)
	if m.UpsertArtefactFunc != nil {
		return m.UpsertArtefactFunc(ctx, artefactID, sourcePath, sourceData, storageBackendID, desiredProfiles)
	}
	return nil, nil
}
