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

package render_adapters

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/maypok86/otter/v2/stats"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
)

const (
	// defaultComponentCacheCapacity is the default capacity for the component
	// cache when no value is specified in configuration.
	defaultComponentCacheCapacity = 100

	// defaultComponentCacheTTLMinutes is the default time-to-live for cached
	// components in minutes.
	defaultComponentCacheTTLMinutes = 5

	// defaultSVGCacheCapacity is the default number of SVG entries to store
	// in the cache.
	defaultSVGCacheCapacity = 200

	// defaultSVGCacheTTLMinutes is the default time-to-live in minutes for
	// cached SVG data.
	defaultSVGCacheTTLMinutes = 30

	// cacheRefreshTTLMultiplier is the fraction of the cache TTL at which a
	// background refresh is triggered.
	cacheRefreshTTLMultiplier = 0.9

	// defaultSVGBufferCapacity is the initial buffer size in bytes for SVG data.
	defaultSVGBufferCapacity = 4096

	// sequentialSVGThreshold is the batch size at or below which SVGs are loaded
	// one at a time rather than in parallel. Small batches run faster without the
	// extra cost of starting goroutines.
	sequentialSVGThreshold = 5
)

var svgBufferPool = sync.Pool{
	New: func() any {
		return new(make([]byte, 0, defaultSVGBufferCapacity))
	},
}

// DataLoaderAdapterConfig holds configuration for DataLoaderRegistryAdapter.
// It specifies cache capacities and TTLs for both component and SVG caches.
type DataLoaderAdapterConfig struct {
	// ComponentCacheCapacity is the maximum number of entries in the component
	// cache; 0 uses the default capacity.
	ComponentCacheCapacity int

	// ComponentCacheTTL is the time-to-live for cached components; 0 uses the
	// default.
	ComponentCacheTTL time.Duration

	// SVGCacheCapacity is the maximum number of parsed SVG entries to cache.
	// Zero uses the default value.
	SVGCacheCapacity int

	// SVGCacheTTL is how long SVG cache entries are kept; 0 uses the default.
	SVGCacheTTL time.Duration
}

// DataLoaderRegistryAdapter implements the RegistryPort and
// RenderRegistryCachePort interfaces using Otter v2 caches. It loads and
// caches component metadata and SVG assets in batches.
type DataLoaderRegistryAdapter struct {
	// registryService provides the underlying registry operations.
	registryService registry_domain.RegistryService

	// componentBulkLoader fetches component metadata in batches on cache miss.
	componentBulkLoader otter.BulkLoader[string, *render_dto.ComponentMetadata]

	// svgBulkLoader loads and parses SVG data for multiple assets in one call.
	svgBulkLoader otter.BulkLoader[string, *render_domain.ParsedSvgData]

	// componentCache stores component metadata, keyed by component type.
	componentCache *otter.Cache[string, *render_dto.ComponentMetadata]

	// svgCache stores parsed SVG data with the asset ID as the key.
	svgCache *otter.Cache[string, *render_domain.ParsedSvgData]
}

// cacheMetrics groups the three metric counters used by the cache fast path.
type cacheMetrics struct {
	// ErrorCounter records empty-key and load errors.
	ErrorCounter metric.Int64Counter

	// HitCounter records cache hits.
	HitCounter metric.Int64Counter

	// MissCounter records cache misses.
	MissCounter metric.Int64Counter
}

// cacheSlowConfig holds settings for tracking cache slow path operations.
type cacheSlowConfig struct {
	// spanName is the name used for tracing spans.
	spanName string

	// durationHist records the time taken for cache operations in milliseconds.
	durationHist metric.Float64Histogram

	// errorCounter records the number of cache lookup failures.
	errorCounter metric.Int64Counter

	// errorMessage is the message used when reporting cache lookup errors.
	errorMessage string
}

// BulkGetComponentMetadata retrieves metadata for multiple component types
// in a single batch operation. Cache hits are returned immediately; cache
// misses are fetched together via the bulk loader.
//
// Takes componentTypes ([]string) which lists the component types to look up.
//
// Returns map[string]*render_dto.ComponentMetadata which maps each component
// type to its metadata.
// Returns error when the batch retrieval fails.
func (r *DataLoaderRegistryAdapter) BulkGetComponentMetadata(ctx context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "DataLoaderRegistryAdapter.BulkGetComponentMetadata")
	defer span.End()

	return r.componentCache.BulkGet(ctx, componentTypes, r.componentBulkLoader)
}

// GetComponentMetadata retrieves component metadata from the cache or loads
// it if not present. Uses a fast path for cache hits to avoid tracing and
// loader allocation overhead.
//
// Takes componentType (string) which specifies the type of component to load.
//
// Returns *render_dto.ComponentMetadata which contains the loaded metadata.
// Returns error when componentType is empty or loading fails.
func (r *DataLoaderRegistryAdapter) GetComponentMetadata(ctx context.Context, componentType string) (*render_dto.ComponentMetadata, error) {
	return getCacheWithFastPath(ctx, componentType, "cannot load component: empty componentType", r.componentCache, r.componentBulkLoader,
		cacheMetrics{
			ErrorCounter: componentLoaderErrorCount,
			HitCounter:   componentLoaderCacheHitCount,
			MissCounter:  componentLoaderCacheMissCount,
		},
		cacheSlowConfig{
			spanName:     "DataLoaderRegistryAdapter.GetComponentMetadata",
			durationHist: componentLoadDuration,
			errorCounter: componentLoaderErrorCount,
			errorMessage: "Failed to get component metadata",
		})
}

// GetAssetRawSVG retrieves parsed SVG data from the cache or loads it if not
// present. Uses a fast path for cache hits to avoid tracing and loader
// allocation overhead.
//
// Takes assetID (string) which identifies the SVG asset to retrieve.
//
// Returns *render_domain.ParsedSvgData which contains the parsed SVG content.
// Returns error when the assetID is empty or the asset cannot be loaded.
func (r *DataLoaderRegistryAdapter) GetAssetRawSVG(ctx context.Context, assetID string) (*render_domain.ParsedSvgData, error) {
	return getCacheWithFastPath(ctx, assetID, "cannot load SVG: empty assetID", r.svgCache, r.svgBulkLoader,
		cacheMetrics{
			ErrorCounter: svgLoaderErrorCount,
			HitCounter:   svgLoaderCacheHitCount,
			MissCounter:  svgLoaderCacheMissCount,
		},
		cacheSlowConfig{
			spanName:     "DataLoaderRegistryAdapter.GetAssetRawSVG",
			durationHist: svgLoadDuration,
			errorCounter: svgLoaderErrorCount,
			errorMessage: "Failed to get SVG asset",
		})
}

// BulkGetAssetRawSVG retrieves multiple SVG assets efficiently using bulk
// loading.
//
// Takes assetIDs ([]string) which specifies the asset identifiers to retrieve.
//
// Returns map[string]*render_domain.ParsedSvgData which maps asset IDs to their
// parsed SVG data.
// Returns error when the bulk retrieval fails.
func (r *DataLoaderRegistryAdapter) BulkGetAssetRawSVG(ctx context.Context, assetIDs []string) (map[string]*render_domain.ParsedSvgData, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "DataLoaderRegistryAdapter.BulkGetAssetRawSVG")
	defer span.End()

	return r.svgCache.BulkGet(ctx, assetIDs, r.svgBulkLoader)
}

// Close releases all resources held by the component and SVG caches.
func (r *DataLoaderRegistryAdapter) Close() {
	if r.componentCache != nil {
		r.componentCache.StopAllGoroutines()
	}
	if r.svgCache != nil {
		r.svgCache.StopAllGoroutines()
	}
}

// GetStats returns statistics about the cache sizes.
//
// Returns render_domain.RegistryAdapterStats which contains the estimated sizes
// of the component and SVG caches.
func (r *DataLoaderRegistryAdapter) GetStats() render_domain.RegistryAdapterStats {
	return render_domain.RegistryAdapterStats{
		ComponentCacheSize: r.componentCache.EstimatedSize(),
		SVGCacheSize:       r.svgCache.EstimatedSize(),
	}
}

// GetComponentCacheSize returns the estimated number of entries in the
// component metadata cache.
//
// Returns int which is the current estimated cache size.
func (r *DataLoaderRegistryAdapter) GetComponentCacheSize() int {
	return r.componentCache.EstimatedSize()
}

// GetSVGCacheSize returns the estimated number of entries in the SVG asset
// cache.
//
// Returns int which is the current estimated cache size.
func (r *DataLoaderRegistryAdapter) GetSVGCacheSize() int {
	return r.svgCache.EstimatedSize()
}

// ClearComponentCache removes a specific component type from the cache.
//
// Takes componentType (string) which specifies the component type to remove.
func (r *DataLoaderRegistryAdapter) ClearComponentCache(_ context.Context, componentType string) {
	if r.componentCache != nil {
		r.componentCache.Invalidate(componentType)
	}
}

// ClearSvgCache invalidates a specific SVG from the cache.
//
// Takes svgID (string) which identifies the SVG to remove from the cache.
func (r *DataLoaderRegistryAdapter) ClearSvgCache(_ context.Context, svgID string) {
	if r.svgCache != nil {
		r.svgCache.Invalidate(svgID)
	}
}

// UpsertArtefact registers a dynamic asset with metadata-only profiles.
// This delegates to the underlying registry service for metadata-only
// registration.
//
// Takes artefactID (string) which identifies the artefact to register.
// Takes sourcePath (string) which specifies the path to the source file.
// Takes sourceData (io.Reader) which provides the artefact content.
// Takes storageBackendID (string) which identifies the storage backend to use.
// Takes desiredProfiles ([]registry_dto.NamedProfile) which specifies the
// profiles to apply.
//
// Returns *registry_dto.ArtefactMeta which contains the registered artefact
// metadata.
// Returns error when registration fails.
func (r *DataLoaderRegistryAdapter) UpsertArtefact(
	ctx context.Context,
	artefactID string,
	sourcePath string,
	sourceData io.Reader,
	storageBackendID string,
	desiredProfiles []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	return r.registryService.UpsertArtefact(ctx, artefactID, sourcePath, sourceData, storageBackendID, desiredProfiles)
}

// NewDataLoaderRegistryAdapter creates a new DataLoaderRegistryAdapter with
// the specified configuration. It initialises caches and bulk loaders for
// component metadata and SVG assets.
//
// Takes registryService (registry_domain.RegistryService) which provides access
// to the component registry.
// Takes config (*DataLoaderAdapterConfig) which specifies cache and loader
// settings.
// Takes artefactServePath (string) which specifies the base path for serving
// component artefacts.
//
// Returns render_domain.RegistryPort which is the configured adapter ready
// for use.
func NewDataLoaderRegistryAdapter(
	registryService registry_domain.RegistryService,
	config *DataLoaderAdapterConfig,
	artefactServePath string,
) render_domain.RegistryPort {
	config = applyConfigDefaults(config)

	adapter := &DataLoaderRegistryAdapter{
		registryService:     registryService,
		componentBulkLoader: nil,
		svgBulkLoader:       nil,
		componentCache:      nil,
		svgCache:            nil,
	}

	adapter.componentBulkLoader = createComponentBulkLoader(registryService, artefactServePath)
	adapter.svgBulkLoader = createSVGBulkLoader(registryService)
	adapter.componentCache, adapter.svgCache = createCaches(config)

	return adapter
}

// applyConfigDefaults sets default values for nil or zero-valued fields.
//
// Takes config (*DataLoaderAdapterConfig) which is the configuration to update,
// or nil to create a new configuration with all defaults.
//
// Returns *DataLoaderAdapterConfig which is the configuration with defaults
// set.
func applyConfigDefaults(config *DataLoaderAdapterConfig) *DataLoaderAdapterConfig {
	if config == nil {
		config = &DataLoaderAdapterConfig{}
	}
	if config.ComponentCacheCapacity == 0 {
		config.ComponentCacheCapacity = defaultComponentCacheCapacity
	}
	if config.ComponentCacheTTL == 0 {
		config.ComponentCacheTTL = defaultComponentCacheTTLMinutes * time.Minute
	}
	if config.SVGCacheCapacity == 0 {
		config.SVGCacheCapacity = defaultSVGCacheCapacity
	}
	if config.SVGCacheTTL == 0 {
		config.SVGCacheTTL = defaultSVGCacheTTLMinutes * time.Minute
	}
	return config
}

// createComponentBulkLoader creates an Otter bulk loader for component
// metadata.
//
// Takes registryService (registry_domain.RegistryService) which provides
// access to artefact data.
// Takes artefactServePath (string) which is the base path for serving
// component artefacts.
//
// Returns otter.BulkLoader[string, *render_dto.ComponentMetadata] which
// fetches component metadata in batches.
func createComponentBulkLoader(
	registryService registry_domain.RegistryService,
	artefactServePath string,
) otter.BulkLoader[string, *render_dto.ComponentMetadata] {
	return otter.BulkLoaderFunc[string, *render_dto.ComponentMetadata](
		func(ctx context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error) {
			componentLoaderBatchCount.Add(ctx, 1)

			if len(componentTypes) == 0 {
				return map[string]*render_dto.ComponentMetadata{}, nil
			}

			artefacts, err := registryService.SearchArtefactsByTagValues(ctx, "tagName", componentTypes)
			if err != nil {
				return nil, fmt.Errorf("searching artefacts by tag name: %w", err)
			}

			requestedTypes := buildLookupSet(componentTypes)
			return buildComponentResults(artefacts, requestedTypes, artefactServePath), nil
		},
	)
}

// buildLookupSet creates a set for fast membership checks.
//
// Takes items ([]string) which contains the strings to include in the set.
//
// Returns map[string]struct{} which allows checking if a string is present.
func buildLookupSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}

// buildComponentResults processes artefacts and builds the component metadata
// results map.
//
// Takes artefacts ([]*registry_dto.ArtefactMeta) which provides the source
// artefact metadata to process.
// Takes requestedTypes (map[string]struct{}) which specifies which component
// types to include in the results.
// Takes artefactServePath (string) which is the base path for serving
// artefacts.
//
// Returns map[string]*render_dto.ComponentMetadata which maps tag names to
// their component metadata.
func buildComponentResults(
	artefacts []*registry_dto.ArtefactMeta,
	requestedTypes map[string]struct{},
	artefactServePath string,
) map[string]*render_dto.ComponentMetadata {
	results := make(map[string]*render_dto.ComponentMetadata, len(artefacts))

	for _, artefact := range artefacts {
		if meta := extractComponentMetadata(artefact, requestedTypes, artefactServePath); meta != nil {
			results[meta.TagName] = meta
		}
	}

	return results
}

// extractComponentMetadata extracts component metadata from an artefact if it
// matches requested types.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the artefact to
// extract metadata from.
// Takes requestedTypes (map[string]struct{}) which specifies the tag names to
// include.
// Takes artefactServePath (string) which is the base path for serving assets.
//
// Returns *render_dto.ComponentMetadata which contains the extracted metadata,
// or nil if the artefact has no JS variant, no tag name, or is not requested.
func extractComponentMetadata(
	artefact *registry_dto.ArtefactMeta,
	requestedTypes map[string]struct{},
	artefactServePath string,
) *render_dto.ComponentMetadata {
	jsVariant := findJSVariant(artefact.ActualVariants)
	if jsVariant == nil {
		return nil
	}

	tagName := jsVariant.MetadataTags.Get(registry_dto.TagTagName)
	if tagName == "" {
		return nil
	}

	if _, requested := requestedTypes[tagName]; !requested {
		return nil
	}

	serveVariant := jsVariant
	if daemon_frontend.IsSRIEnabled() {
		for i := range artefact.ActualVariants {
			if artefact.ActualVariants[i].VariantID == "minified" {
				serveVariant = &artefact.ActualVariants[i]
				break
			}
		}
	}

	return &render_dto.ComponentMetadata{
		TagName:         tagName,
		BaseJSPath:      path.Join(artefactServePath, serveVariant.StorageKey),
		DefaultCSS:      "",
		SRIHash:         daemon_frontend.FilterSRIHash(serveVariant.SRIHash),
		RequiredModules: nil,
	}
}

// createSVGBulkLoader creates an Otter bulk loader for SVG assets.
// Uses zero-copy buffer pooling and adaptive sequential/parallel processing.
//
// Takes registryService (registry_domain.RegistryService) which provides
// access to SVG artefact data.
//
// Returns otter.BulkLoader[string, *render_domain.ParsedSvgData] which
// fetches and parses SVGs in batches.
func createSVGBulkLoader(
	registryService registry_domain.RegistryService,
) otter.BulkLoader[string, *render_domain.ParsedSvgData] {
	return otter.BulkLoaderFunc[string, *render_domain.ParsedSvgData](
		func(ctx context.Context, artefactIDs []string) (map[string]*render_domain.ParsedSvgData, error) {
			ctx, l := logger_domain.From(ctx, log)

			svgLoaderBatchCount.Add(ctx, 1)

			artefacts, err := registryService.GetMultipleArtefacts(ctx, artefactIDs)
			if err != nil {
				svgLoaderErrorCount.Add(ctx, 1)
				return nil, fmt.Errorf("fetching SVG artefacts: %w", err)
			}

			artefactMap := make(map[string]*registry_dto.ArtefactMeta, len(artefacts))
			for _, artefact := range artefacts {
				artefactMap[artefact.ID] = artefact
			}

			results := make(map[string]*render_domain.ParsedSvgData, len(artefactIDs))

			frozenBuffers := make([]*[]byte, 0, len(artefactIDs))
			defer func() {
				for _, buffer := range frozenBuffers {
					svgBufferPool.Put(buffer)
				}
			}()

			var failedIDs []string
			if len(artefactIDs) <= sequentialSVGThreshold {
				failedIDs = processSVGsSequential(ctx, artefactIDs, artefactMap, results, &frozenBuffers, registryService)
			} else {
				failedIDs = processSVGsParallel(ctx, artefactIDs, artefactMap, results, &frozenBuffers, registryService)
			}

			if len(failedIDs) > 0 {
				svgLoaderItemFailureCount.Add(ctx, int64(len(failedIDs)))
				l.Warn("Some SVG assets failed to load",
					logger_domain.Int("failed_count", len(failedIDs)),
					logger_domain.Strings("failed_ids", failedIDs),
					logger_domain.Int("total_requested", len(artefactIDs)),
				)
			}

			return results, nil
		},
	)
}

// processSVGsSequential processes SVGs one at a time, used for
// small batches where additional overhead exceeds benefit.
//
// Takes artefactIDs ([]string) which lists the IDs to process.
// Takes artefactMap (map[string]*registry_dto.ArtefactMeta) which
// maps IDs to their artefact metadata.
// Takes results (map[string]*render_domain.ParsedSvgData) which
// receives the parsed SVG data.
// Takes frozenBuffers (*[]*[]byte) which collects buffers for
// deferred pool return.
// Takes registryService (registry_domain.RegistryService) which
// provides variant data access.
//
// Returns []string which contains IDs of failed SVG loads, or nil
// if none failed.
func processSVGsSequential(
	ctx context.Context,
	artefactIDs []string,
	artefactMap map[string]*registry_dto.ArtefactMeta,
	results map[string]*render_domain.ParsedSvgData,
	frozenBuffers *[]*[]byte,
	registryService registry_domain.RegistryService,
) []string {
	var failedIDs []string
	for _, artefactID := range artefactIDs {
		artefact, found := artefactMap[artefactID]
		if !found {
			continue
		}

		rawSvg, err := getRawSVGFromArtefactZeroCopy(ctx, registryService, artefact, frozenBuffers)
		if err != nil {
			failedIDs = append(failedIDs, artefactID)
			continue
		}

		tagContent, innerHTML := extractTagContent(rawSvg, "svg")
		attributes := render_domain.ParseSVGAttributes(tagContent)
		results[artefactID] = buildParsedSVGData(artefactID, attributes, innerHTML)
	}
	return failedIDs
}

// processSVGsParallel processes SVGs simultaneously for larger
// batches, limited to GOMAXPROCS workers.
//
// Takes artefactIDs ([]string) which lists the IDs to process.
// Takes artefactMap (map[string]*registry_dto.ArtefactMeta) which
// maps IDs to their artefact metadata.
// Takes results (map[string]*render_domain.ParsedSvgData) which
// receives the parsed SVG data.
// Takes frozenBuffers (*[]*[]byte) which collects buffers for
// deferred pool return.
// Takes registryService (registry_domain.RegistryService) which
// provides variant data access.
//
// Returns []string which contains IDs of failed SVG loads, or nil
// if none failed.
//
// Concurrent goroutines are spawned up to GOMAXPROCS via errgroup. Shared
// results and frozenBuffers are protected by local mutexes.
func processSVGsParallel(
	ctx context.Context,
	artefactIDs []string,
	artefactMap map[string]*registry_dto.ArtefactMeta,
	results map[string]*render_domain.ParsedSvgData,
	frozenBuffers *[]*[]byte,
	registryService registry_domain.RegistryService,
) []string {
	var mu sync.Mutex
	var bufMutex sync.Mutex
	var failedIDs []string

	g := new(errgroup.Group)
	g.SetLimit(runtime.GOMAXPROCS(0))

	for _, id := range artefactIDs {
		artefactID := id
		g.Go(func() error {
			artefact, found := artefactMap[artefactID]
			if !found {
				return nil
			}

			var localBuffers []*[]byte
			rawSvg, err := getRawSVGFromArtefactZeroCopy(ctx, registryService, artefact, &localBuffers)
			if err != nil {
				mu.Lock()
				failedIDs = append(failedIDs, artefactID)
				mu.Unlock()
				for _, buffer := range localBuffers {
					svgBufferPool.Put(buffer)
				}
				return nil
			}

			tagContent, innerHTML := extractTagContent(rawSvg, "svg")
			attributes := render_domain.ParseSVGAttributes(tagContent)
			parsedData := buildParsedSVGData(artefactID, attributes, innerHTML)

			mu.Lock()
			results[artefactID] = parsedData
			mu.Unlock()

			bufMutex.Lock()
			*frozenBuffers = append(*frozenBuffers, localBuffers...)
			bufMutex.Unlock()

			return nil
		})
	}

	_ = g.Wait()
	return failedIDs
}

// createCaches initialises both component and SVG caches with the specified
// configuration.
//
// Takes config (*DataLoaderAdapterConfig) which provides cache capacity
// and TTL settings.
//
// Returns *otter.Cache[string, *render_dto.ComponentMetadata] which is the
// component metadata cache.
// Returns *otter.Cache[string, *render_domain.ParsedSvgData] which is the
// SVG data cache.
func createCaches(
	config *DataLoaderAdapterConfig,
) (*otter.Cache[string, *render_dto.ComponentMetadata], *otter.Cache[string, *render_domain.ParsedSvgData]) {
	componentRefreshTTL := time.Duration(float64(config.ComponentCacheTTL) * cacheRefreshTTLMultiplier)
	svgRefreshTTL := time.Duration(float64(config.SVGCacheTTL) * cacheRefreshTTLMultiplier)

	componentCache := otter.Must(&otter.Options[string, *render_dto.ComponentMetadata]{
		MaximumSize:       config.ComponentCacheCapacity,
		ExpiryCalculator:  otter.ExpiryWriting[string, *render_dto.ComponentMetadata](config.ComponentCacheTTL),
		RefreshCalculator: otter.RefreshWriting[string, *render_dto.ComponentMetadata](componentRefreshTTL),
		StatsRecorder:     stats.NewCounter(),
		MaximumWeight:     0,
		InitialCapacity:   0,
		Weigher:           nil,
		OnDeletion:        nil,
		OnAtomicDeletion:  nil,
		Executor:          nil,
		Clock:             nil,
		Logger:            nil,
	})

	svgCache := otter.Must(&otter.Options[string, *render_domain.ParsedSvgData]{
		MaximumSize:       config.SVGCacheCapacity,
		ExpiryCalculator:  otter.ExpiryWriting[string, *render_domain.ParsedSvgData](config.SVGCacheTTL),
		RefreshCalculator: otter.RefreshWriting[string, *render_domain.ParsedSvgData](svgRefreshTTL),
		StatsRecorder:     stats.NewCounter(),
		MaximumWeight:     0,
		InitialCapacity:   0,
		Weigher:           nil,
		OnDeletion:        nil,
		OnAtomicDeletion:  nil,
		Executor:          nil,
		Clock:             nil,
		Logger:            nil,
	})

	return componentCache, svgCache
}

// getCacheWithFastPath combines an empty-key guard, a cache fast path, and the
// slow path into a single generic helper. It avoids tracing and loader
// allocation overhead on cache hits.
//
// Takes key (string) which identifies the cache entry to retrieve.
// Takes emptyKeyError (string) which is the error message returned when key is
// empty.
// Takes cache (*otter.Cache) which stores the cached values.
// Takes bulkLoader (otter.BulkLoader) which fetches values on cache miss.
// Takes counters (cacheMetrics) which holds the error, hit, and miss counters.
// Takes slowConfig (cacheSlowConfig) which configures span name, duration
// histogram, and error reporting for the slow path.
//
// Returns *T which is the cached or freshly loaded value.
// Returns error when key is empty or the slow path fails.
func getCacheWithFastPath[T any](
	ctx context.Context,
	key string,
	emptyKeyError string,
	cache *otter.Cache[string, *T],
	bulkLoader otter.BulkLoader[string, *T],
	counters cacheMetrics,
	slowConfig cacheSlowConfig,
) (*T, error) {
	if key == "" {
		counters.ErrorCounter.Add(ctx, 1)
		return nil, errors.New(emptyKeyError)
	}

	if result, ok := cache.GetIfPresent(key); ok {
		counters.HitCounter.Add(ctx, 1)
		return result, nil
	}

	counters.MissCounter.Add(ctx, 1)
	return getCacheSlow(ctx, key, cache, bulkLoader, slowConfig)
}

// getCacheSlow is a generic helper for cache miss handling with full tracing.
// It creates a span, records duration metrics, and handles errors consistently.
//
// Takes key (string) which identifies the cache entry to retrieve.
// Takes cache (*otter.Cache) which stores the cached values.
// Takes bulkLoader (otter.BulkLoader) which fetches values on cache miss.
// Takes config (cacheSlowConfig) which provides span name, metrics, and error
// message configuration.
//
// Returns T which is the cached or freshly loaded value.
// Returns error when the bulk loader fails or the key is not found.
func getCacheSlow[T any](
	ctx context.Context,
	key string,
	cache *otter.Cache[string, T],
	bulkLoader otter.BulkLoader[string, T],
	config cacheSlowConfig,
) (T, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, config.spanName)
	defer span.End()

	startTime := time.Now()

	loader := otter.LoaderFunc[string, T](func(ctx context.Context, k string) (T, error) {
		results, err := bulkLoader.BulkLoad(ctx, []string{k})
		if err != nil {
			var zero T
			return zero, err
		}
		if value, ok := results[k]; ok {
			return value, nil
		}
		var zero T
		return zero, otter.ErrNotFound
	})

	result, err := cache.Get(ctx, key, loader)

	duration := time.Since(startTime).Milliseconds()
	config.durationHist.Record(ctx, float64(duration))

	if err != nil {
		var zero T
		if !errors.Is(err, otter.ErrNotFound) {
			l.ReportError(span, err, config.errorMessage)
			config.errorCounter.Add(ctx, 1)
		}
		return zero, err
	}

	return result, nil
}

// findJSVariant finds the first variant with component-js type and entrypoint
// role.
//
// Takes variants ([]registry_dto.Variant) which is the list to search.
//
// Returns *registry_dto.Variant which is the matching variant, or nil if not
// found.
func findJSVariant(variants []registry_dto.Variant) *registry_dto.Variant {
	for i := range variants {
		v := &variants[i]
		if v.MetadataTags.Get(registry_dto.TagType) == "component-js" && v.MetadataTags.Get(registry_dto.TagRole) == "entrypoint" {
			return v
		}
	}
	return nil
}

// getRawSVGFromArtefactZeroCopy loads SVG content using pooled buffers and
// zero-copy string conversion.
//
// The frozenBuffers slice tracks buffers for deferred pool return after the
// result is processed.
//
// Takes registryService (RegistryService) which provides access to variant
// data.
// Takes artefact (*ArtefactMeta) which specifies the SVG artefact to load.
// Takes frozenBuffers (*[]*[]byte) which collects buffers for later pool
// return.
//
// Returns string which contains the raw SVG content.
// Returns error when the artefact is nil, no suitable variant exists, or
// reading fails.
func getRawSVGFromArtefactZeroCopy(
	ctx context.Context,
	registryService registry_domain.RegistryService,
	artefact *registry_dto.ArtefactMeta,
	frozenBuffers *[]*[]byte,
) (string, error) {
	if artefact == nil {
		return "", errors.New("cannot get SVG from a nil artefact")
	}

	bestVariant := findSVGVariant(artefact)
	if bestVariant == nil {
		return "", fmt.Errorf("no suitable variant (minified or source) found for SVG artefact '%s'", artefact.ID)
	}

	svgStream, err := registryService.GetVariantData(ctx, bestVariant)
	if err != nil {
		return "", fmt.Errorf("failed to get stream for SVG '%s': %w", artefact.ID, err)
	}
	defer func() { _ = svgStream.Close() }()

	buffer, ok := svgBufferPool.Get().(*[]byte)
	if !ok {
		buffer = new(make([]byte, 0, defaultSVGBufferCapacity))
	}
	*buffer = (*buffer)[:0]

	*buffer, err = io.ReadAll(svgStream)
	if err != nil {
		svgBufferPool.Put(buffer)
		return "", fmt.Errorf("failed to read from SVG stream for '%s': %w", artefact.ID, err)
	}

	*frozenBuffers = append(*frozenBuffers, buffer)

	return mem.String(*buffer), nil
}

// buildParsedSVGData creates a ParsedSvgData with all fields populated.
// Pre-computes the symbol string at load time to avoid per-request
// allocation overhead.
//
// Takes artefactID (string) which identifies the SVG artefact.
// Takes attributes ([]ast_domain.HTMLAttribute) which contains the SVG
// element attributes.
// Takes innerHTML (string) which holds the SVG content.
//
// Returns *render_domain.ParsedSvgData which is the fully populated SVG data
// with cached symbol.
func buildParsedSVGData(artefactID string, attributes []ast_domain.HTMLAttribute, innerHTML string) *render_domain.ParsedSvgData {
	data := &render_domain.ParsedSvgData{
		Attributes:   attributes,
		InnerHTML:    innerHTML,
		CachedSymbol: "",
	}
	data.CachedSymbol = render_domain.ComputeSymbolString(artefactID, data)
	return data
}

// findSVGVariant finds the best SVG variant from an artefact, preferring
// minified over source.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the variants to
// search.
//
// Returns *registry_dto.Variant which is the minified-svg variant if found,
// otherwise the source variant, or nil if neither exists.
func findSVGVariant(artefact *registry_dto.ArtefactMeta) *registry_dto.Variant {
	for i := range artefact.ActualVariants {
		v := &artefact.ActualVariants[i]
		if v.MetadataTags.Get(registry_dto.TagType) == "minified-svg" {
			return v
		}
	}
	for i := range artefact.ActualVariants {
		v := &artefact.ActualVariants[i]
		if v.MetadataTags.Get(registry_dto.TagType) == "source" {
			return v
		}
	}
	return nil
}

// indexFoldCase returns the index of the first case-insensitive match of
// substr in s.
//
// Takes s (string) which is the string to search within.
// Takes substr (string) which is the substring to find.
//
// Returns int which is the index of the match, or -1 if not found.
//
// This avoids allocating by not converting the entire string.
func indexFoldCase(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if strings.EqualFold(s[i:i+len(substr)], substr) {
			return i
		}
	}
	return -1
}

// lastIndexFoldCase finds the last occurrence of substr in s, ignoring case.
//
// Takes s (string) which is the string to search within.
// Takes substr (string) which is the substring to find.
//
// Returns int which is the index of the last match, or -1 if not found.
func lastIndexFoldCase(s, substr string) int {
	if len(substr) == 0 {
		return len(s)
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := len(s) - len(substr); i >= 0; i-- {
		if strings.EqualFold(s[i:i+len(substr)], substr) {
			return i
		}
	}
	return -1
}

// extractTagContent finds a named HTML tag and returns its parts.
//
// Takes rawHTML (string) which is the HTML source to search.
// Takes tagName (string) which is the tag name to find (e.g. "svg").
//
// Returns tagContent (string) which holds the tag's attributes.
// Returns innerContent (string) which holds the content between the opening and
// closing tags, or the full HTML if the tag is not found.
func extractTagContent(rawHTML, tagName string) (tagContent string, innerContent string) {
	tagOpen := "<" + tagName

	start := indexFoldCase(rawHTML, tagOpen)
	if start == -1 {
		return "", rawHTML
	}

	endOfOpenTag := strings.Index(rawHTML[start:], ">")
	if endOfOpenTag == -1 {
		return rawHTML[start+len(tagOpen):], ""
	}
	endOfOpenTag += start

	tagContent = rawHTML[start+len(tagOpen) : endOfOpenTag]

	closeTag := "</" + tagName + ">"
	closeTagStart := lastIndexFoldCase(rawHTML, closeTag)
	if closeTagStart == -1 || closeTagStart < endOfOpenTag {
		return tagContent, ""
	}

	innerContent = rawHTML[endOfOpenTag+1 : closeTagStart]
	return tagContent, innerContent
}
