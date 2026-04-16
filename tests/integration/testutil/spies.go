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

package testutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
)

var _ render_domain.RegistryPort = (*SpyRegistryPort)(nil)

// SpyRegistryPort captures artefact lookup calls while delegating to an
// optional real implementation, allowing verification that the correct
// artefact IDs are being looked up during rendering.
type SpyRegistryPort struct {
	// delegate is the underlying registry port that this spy wraps.
	delegate render_domain.RegistryPort

	// svgData maps icon names to their parsed SVG data.
	svgData map[string]*render_domain.ParsedSvgData

	// componentData maps component names to their metadata.
	componentData map[string]*render_dto.ComponentMetadata

	// svgLookupCalls records the arguments passed to SVGLookup calls for test
	// verification.
	svgLookupCalls []string

	// componentLookupCalls records the component names passed to ComponentLookup.
	componentLookupCalls []string

	// mu guards access to the registry state.
	mu sync.Mutex
}

// NewSpyRegistryPort creates a new spy registry with optional mock data.
//
// Returns *SpyRegistryPort which is an uninitialised spy ready for test
// configuration.
func NewSpyRegistryPort() *SpyRegistryPort {
	return &SpyRegistryPort{
		delegate:             nil,
		svgData:              make(map[string]*render_domain.ParsedSvgData),
		componentData:        make(map[string]*render_dto.ComponentMetadata),
		svgLookupCalls:       nil,
		componentLookupCalls: nil,
		mu:                   sync.Mutex{},
	}
}

// NewSpyRegistryPortWithDelegate creates a spy that delegates to a real
// registry.
//
// Takes delegate (render_domain.RegistryPort) which provides the real registry
// to forward calls to.
//
// Returns *SpyRegistryPort which wraps the delegate with spy capabilities.
func NewSpyRegistryPortWithDelegate(delegate render_domain.RegistryPort) *SpyRegistryPort {
	spy := NewSpyRegistryPort()
	spy.delegate = delegate
	return spy
}

// SetSVGData configures mock SVG data for a given asset ID.
// Pre-computes CachedSymbol to match production behaviour.
//
// Takes assetID (string) which identifies the asset to configure.
// Takes data (*render_domain.ParsedSvgData) which provides the SVG data to
// store.
//
// Safe for concurrent use; protected by a mutex.
func (s *SpyRegistryPort) SetSVGData(assetID string, data *render_domain.ParsedSvgData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if data != nil && data.CachedSymbol == "" {
		data.CachedSymbol = render_domain.ComputeSymbolString(assetID, data)
	}
	s.svgData[assetID] = data
}

// SetComponentData configures mock component metadata for a given component
// type.
//
// Takes componentType (string) which identifies the component to configure.
// Takes data (*render_dto.ComponentMetadata) which provides the metadata to
// store.
//
// Safe for concurrent use; protected by mutex.
func (s *SpyRegistryPort) SetComponentData(componentType string, data *render_dto.ComponentMetadata) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.componentData[componentType] = data
}

// GetSVGLookupCalls returns all SVG asset IDs that have been looked up.
//
// Returns []string which contains a copy of the recorded lookup calls.
//
// Safe for concurrent use.
func (s *SpyRegistryPort) GetSVGLookupCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.svgLookupCalls))
	copy(result, s.svgLookupCalls)
	return result
}

// GetComponentLookupCalls returns all component types that have been looked up.
//
// Returns []string which contains a copy of all recorded lookup calls.
//
// Safe for concurrent use.
func (s *SpyRegistryPort) GetComponentLookupCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.componentLookupCalls))
	copy(result, s.componentLookupCalls)
	return result
}

// Reset clears all captured calls but preserves mock data.
//
// Safe for concurrent use.
func (s *SpyRegistryPort) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.svgLookupCalls = nil
	s.componentLookupCalls = nil
}

// GetAssetRawSVG implements render_domain.RegistryPort.
//
// Takes assetID (string) which identifies the SVG asset to retrieve.
//
// Returns *render_domain.ParsedSvgData which contains the parsed SVG data.
// Returns error when the asset is not found and no delegate is available.
//
// Safe for concurrent use; protected by a mutex.
func (s *SpyRegistryPort) GetAssetRawSVG(ctx context.Context, assetID string) (*render_domain.ParsedSvgData, error) {
	s.mu.Lock()
	s.svgLookupCalls = append(s.svgLookupCalls, assetID)

	if data, ok := s.svgData[assetID]; ok {
		s.mu.Unlock()
		return data, nil
	}
	s.mu.Unlock()

	if s.delegate != nil {
		return s.delegate.GetAssetRawSVG(ctx, assetID)
	}

	return nil, fmt.Errorf("SVG asset not found: %s", assetID)
}

// BulkGetAssetRawSVG implements render_domain.RegistryPort.
func (s *SpyRegistryPort) BulkGetAssetRawSVG(ctx context.Context, assetIDs []string) (map[string]*render_domain.ParsedSvgData, error) {
	if s.delegate != nil {
		return s.delegate.BulkGetAssetRawSVG(ctx, assetIDs)
	}

	return nil, nil
}

// GetComponentMetadata implements render_domain.RegistryPort.
//
// Takes componentType (string) which specifies the component to look up.
//
// Returns *render_dto.ComponentMetadata which contains the component details.
// Returns error when the component is not found.
//
// Safe for concurrent use. Uses a mutex to protect access to internal state.
func (s *SpyRegistryPort) GetComponentMetadata(ctx context.Context, componentType string) (*render_dto.ComponentMetadata, error) {
	s.mu.Lock()
	s.componentLookupCalls = append(s.componentLookupCalls, componentType)

	if data, ok := s.componentData[componentType]; ok {
		s.mu.Unlock()
		return data, nil
	}
	s.mu.Unlock()

	if s.delegate != nil {
		return s.delegate.GetComponentMetadata(ctx, componentType)
	}

	return nil, fmt.Errorf("component not found: %s", componentType)
}

// BulkGetComponentMetadata implements render_domain.RegistryPort.
func (s *SpyRegistryPort) BulkGetComponentMetadata(ctx context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error) {
	if s.delegate != nil {
		return s.delegate.BulkGetComponentMetadata(ctx, componentTypes)
	}

	return nil, nil
}

// GetStats implements render_domain.RegistryPort.
//
// Returns render_domain.RegistryAdapterStats which contains the registry
// statistics from the delegate, or an empty stats struct if no delegate is
// set.
func (s *SpyRegistryPort) GetStats() render_domain.RegistryAdapterStats {
	if s.delegate != nil {
		return s.delegate.GetStats()
	}
	return render_domain.RegistryAdapterStats{}
}

// ClearComponentCache implements render_domain.RegistryPort.
//
// Takes componentType (string) which specifies the type of component to clear
// from the cache.
func (s *SpyRegistryPort) ClearComponentCache(ctx context.Context, componentType string) {
	if s.delegate != nil {
		s.delegate.ClearComponentCache(ctx, componentType)
	}
}

// ClearSvgCache clears the cached SVG for the given identifier.
// Implements render_domain.RegistryPort.
//
// Takes svgID (string) which identifies the SVG to remove from the cache.
func (s *SpyRegistryPort) ClearSvgCache(ctx context.Context, svgID string) {
	if s.delegate != nil {
		s.delegate.ClearSvgCache(ctx, svgID)
	}
}

// GetArtefactServePath implements render_domain.RegistryPort.
//
// Takes artefactID (string) which identifies the artefact to look up.
//
// Returns string which is the serve path from the delegate, or empty if no
// delegate is set.
func (s *SpyRegistryPort) GetArtefactServePath(ctx context.Context, artefactID string) string {
	if s.delegate != nil {
		return s.delegate.GetArtefactServePath(ctx, artefactID)
	}
	return ""
}

// UpsertArtefact implements render_domain.RegistryPort.
//
// Takes artefactID (string) which identifies the artefact to create or update.
// Takes sourcePath (string) which specifies the path to the source content.
// Takes sourceData (io.Reader) which provides the artefact content to store.
// Takes storageBackendID (string) which identifies the storage backend to use.
// Takes desiredProfiles ([]registry_dto.NamedProfile) which specifies the
// profiles to apply to the artefact.
//
// Returns *registry_dto.ArtefactMeta which contains metadata for the artefact.
// Returns error when the delegate is nil or the operation fails.
func (s *SpyRegistryPort) UpsertArtefact(
	ctx context.Context,
	artefactID string,
	sourcePath string,
	sourceData io.Reader,
	storageBackendID string,
	desiredProfiles []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	if s.delegate != nil {
		return s.delegate.UpsertArtefact(ctx, artefactID, sourcePath, sourceData, storageBackendID, desiredProfiles)
	}
	return nil, errors.New("UpsertArtefact not implemented in spy")
}

// CacheSpy tracks cache hit and miss counts for testing.
type CacheSpy struct {
	// mu guards access to spy statistics during concurrent operations.
	mu sync.Mutex

	// hits is the number of cache hits recorded.
	hits int

	// misses counts the number of cache lookups that found no entry.
	misses int
}

// NewCacheSpy creates a new cache spy.
//
// Returns *CacheSpy which is an empty spy ready to record cache operations.
func NewCacheSpy() *CacheSpy {
	return &CacheSpy{}
}

// RecordHit records a cache hit.
//
// Safe for concurrent use.
func (c *CacheSpy) RecordHit() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hits++
}

// RecordMiss records a cache miss.
//
// Safe for concurrent use.
func (c *CacheSpy) RecordMiss() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.misses++
}

// Hits returns the number of cache hits.
//
// Returns int which is the current hit count.
//
// Safe for concurrent use.
func (c *CacheSpy) Hits() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits
}

// Misses returns the number of cache misses.
//
// Returns int which is the count of cache misses recorded.
//
// Safe for concurrent use.
func (c *CacheSpy) Misses() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.misses
}

// Reset clears the cache statistics.
//
// Safe for concurrent use.
func (c *CacheSpy) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hits = 0
	c.misses = 0
}
