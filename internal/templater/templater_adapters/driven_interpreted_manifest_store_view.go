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

package templater_adapters

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/templater/templater_domain"
)

// interpretedManifestStoreView implements ManifestStoreView as a lightweight
// read-only view over the InterpretedManifestRunner's cache. The RouterManager
// uses it in dev-i mode to mount routes after each JIT build.
type interpretedManifestStoreView struct {
	// r references the parent runner for cache and lock access.
	r *InterpretedManifestRunner
}

// GetKeys returns a stable, sorted list of all known component keys.
//
// Returns []string which contains all keys in sorted order.
//
// Safe for concurrent use.
func (v *interpretedManifestStoreView) GetKeys() []string {
	v.r.cacheLock.RLock()
	defer v.r.cacheLock.RUnlock()
	return slices.Sorted(maps.Keys(v.r.progCache))
}

// GetPageEntry returns a snapshot view of the page entry for the given path,
// if present.
//
// Takes path (string) which specifies the key to look up in the cache.
//
// Returns templater_domain.PageEntryView which is the cached page entry.
// Returns bool which indicates whether the entry was found.
//
// Safe for concurrent use; protected by a read lock.
func (v *interpretedManifestStoreView) GetPageEntry(path string) (templater_domain.PageEntryView, bool) {
	v.r.cacheLock.RLock()
	defer v.r.cacheLock.RUnlock()
	pe, ok := v.r.progCache[path]
	return pe, ok
}

// FindErrorPage looks up the most specific error page for the given
// HTTP status code and request path using the same three-tier
// fallback chain as the compiled ManifestStore: exact status-code
// match first, then range match, then catch-all. Within each tier
// the candidate with the longest matching ScopePath wins.
//
// Takes statusCode (int) which is the HTTP status code to match.
// Takes requestPath (string) which is the URL path being requested.
//
// Returns templater_domain.PageEntryView which is the matching error
// page entry.
// Returns bool which is true when a matching error page was found.
//
// Concurrency: read-locks the runner's progCache for the scan duration.
func (v *interpretedManifestStoreView) FindErrorPage(statusCode int, requestPath string) (templater_domain.PageEntryView, bool) {
	v.r.cacheLock.RLock()
	defer v.r.cacheLock.RUnlock()

	if entry, ok := findErrorPageTier(v.r.progCache, requestPath, func(d *ErrorPageDispatch) bool {
		return !d.IsCatchAll && d.StatusCodeMin == 0 && d.StatusCodeMax == 0 && d.StatusCode == statusCode
	}); ok {
		return entry, true
	}
	if entry, ok := findErrorPageTier(v.r.progCache, requestPath, func(d *ErrorPageDispatch) bool {
		return !d.IsCatchAll && d.StatusCodeMin > 0 && d.StatusCodeMax > 0 &&
			statusCode >= d.StatusCodeMin && statusCode <= d.StatusCodeMax
	}); ok {
		return entry, true
	}
	return findErrorPageTier(v.r.progCache, requestPath, func(d *ErrorPageDispatch) bool {
		return d.IsCatchAll
	})
}

// findErrorPageTier scans progCache for entries whose ErrorDispatch
// satisfies match and whose ScopePath is a prefix of requestPath,
// returning the entry with the longest matching ScopePath.
//
// Takes progCache (map[string]*PageEntry) holding the compiled
// entries to scan.
// Takes requestPath (string) which is the incoming request path.
// Takes match (func(*ErrorPageDispatch) bool) which selects entries
// belonging to the current tier.
//
// Returns templater_domain.PageEntryView and true when an entry was
// found; nil and false otherwise.
func findErrorPageTier(
	progCache map[string]*PageEntry,
	requestPath string,
	match func(*ErrorPageDispatch) bool,
) (templater_domain.PageEntryView, bool) {
	var best *PageEntry
	bestLen := -1
	for _, entry := range progCache {
		dispatch := entry.ErrorDispatch
		if dispatch == nil || !match(dispatch) {
			continue
		}
		if !strings.HasPrefix(requestPath, dispatch.ScopePath) {
			continue
		}
		if len(dispatch.ScopePath) > bestLen {
			best = entry
			bestLen = len(dispatch.ScopePath)
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

// GetCollectionFallbackRoutes is not supported in interpreted mode. Static
// collection expansion only happens during a compiled build.
//
// Returns nil always.
func (*interpretedManifestStoreView) GetCollectionFallbackRoutes() []templater_domain.CollectionFallbackRouteView {
	return nil
}

// ListPreviewEntries returns all entries with a Preview function from the
// interpreted runner's in-memory cache.
//
// Returns []templater_domain.PreviewCatalogueEntry which contains preview
// entries sorted by source path.
//
// Concurrency: acquires a read lock on the runner's cache.
func (v *interpretedManifestStoreView) ListPreviewEntries() []templater_domain.PreviewCatalogueEntry {
	v.r.cacheLock.RLock()
	defer v.r.cacheLock.RUnlock()

	var entries []templater_domain.PreviewCatalogueEntry
	for _, entry := range v.r.progCache {
		if !entry.HasPreview || entry.previewFunc == nil {
			continue
		}
		componentType := classifyComponentType(entry.OriginalSourcePath)
		scenarios := entry.previewFunc()
		entries = append(entries, templater_domain.PreviewCatalogueEntry{
			OriginalSourcePath: entry.OriginalSourcePath,
			ComponentType:      componentType,
			Scenarios:          scenarios,
		})
	}

	slices.SortFunc(entries, func(a, b templater_domain.PreviewCatalogueEntry) int {
		return cmp.Compare(a.OriginalSourcePath, b.OriginalSourcePath)
	})

	return entries
}

// classifyComponentType determines the component type from its source path
// prefix.
//
// Takes sourcePath (string) which is the source path to classify.
//
// Returns string which is the component type ("page", "partial", "email",
// "pdf", or "component").
func classifyComponentType(sourcePath string) string {
	switch {
	case strings.HasPrefix(sourcePath, "pages/") || strings.HasPrefix(sourcePath, "e2e/pages/"):
		return "page"
	case strings.HasPrefix(sourcePath, "partials/") || strings.HasPrefix(sourcePath, "e2e/partials/"):
		return "partial"
	case strings.HasPrefix(sourcePath, "emails/"):
		return "email"
	case strings.HasPrefix(sourcePath, "pdfs/"):
		return "pdf"
	default:
		return "component"
	}
}

// NewInterpretedManifestStoreView creates a ManifestStoreView that uses the
// interpreted runner's in-memory cache.
//
// Takes r (*InterpretedManifestRunner) which provides the in-memory cache for
// manifest data.
//
// Returns templater_domain.ManifestStoreView which gives read access to the
// cached manifest entries.
func NewInterpretedManifestStoreView(r *InterpretedManifestRunner) templater_domain.ManifestStoreView {
	return &interpretedManifestStoreView{r: r}
}
