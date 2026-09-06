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

package wasm_domain

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

const (
	// minCatchAllSegmentLength is the minimum length of a catch-all route segment like {x*}
	// (opening brace + at least one char + star + closing brace).
	minCatchAllSegmentLength = 3
)

var (
	// defaultRuntimeImports lists the framework-runtime URLs.
	defaultRuntimeImports = []string{
		generator_domain.PKFrameworkURL,
		generator_domain.PKComponentsURL,
		generator_domain.PKActionsGenURL,
	}
)

// routeCandidate is one manifest page whose route pattern matches a request URL, ranked
// by how specific that pattern is.
type routeCandidate struct {
	// pattern is the route pattern that matched.
	pattern string

	// page is the matching manifest entry.
	page wasm_dto.ManifestPageEntry

	// dynamicSegments counts the pattern's single-segment parameters.
	dynamicSegments int

	// segments counts the pattern's path segments.
	segments int

	// catchAll is true when the pattern ends in a catch-all parameter.
	catchAll bool
}

// findPageArtefactForURL finds the page artefact that matches the given URL.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains all generated files.
// Takes manifest (*wasm_dto.GeneratedManifest) which contains page metadata and package
// paths.
// Takes requestURL (string) which is the URL to match against page routes.
// Takes moduleName (string) which is the Go module name prefix for partial import paths.
//
// Returns *wasm_dto.GeneratedArtefact which is the matching page artefact.
// Returns string which is the package path for the page.
// Returns map[string]string which contains dependency paths mapped to content.
func findPageArtefactForURL(artefacts []wasm_dto.GeneratedArtefact, manifest *wasm_dto.GeneratedManifest, requestURL, moduleName string) (*wasm_dto.GeneratedArtefact, string, map[string]string) {
	dependencies := collectPartialDependencies(artefacts, moduleName)

	if manifest != nil && manifest.Pages != nil {
		if artefact, packagePath := findMatchingPageFromManifest(artefacts, manifest, requestURL); artefact != nil {
			return artefact, packagePath, dependencies
		}
	}

	pageArtefact, packagePath := findFirstPageArtefact(artefacts, manifest)
	return pageArtefact, packagePath, dependencies
}

// collectPartialDependencies builds a dependency map from partial artefacts so the
// interpreter can resolve partial imports during type checking.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains all generated files
// including partial artefacts.
// Takes moduleName (string) which is the Go module name prefix for import paths.
//
// Returns map[string]string which maps partial package paths to their generated source
// code.
func collectPartialDependencies(artefacts []wasm_dto.GeneratedArtefact, moduleName string) map[string]string {
	dependencies := make(map[string]string)

	for i := range artefacts {
		if artefacts[i].Type != wasm_dto.ArtefactTypePartial {
			continue
		}
		if !strings.HasSuffix(artefacts[i].Path, ".go") {
			continue
		}

		directory := artefactDirectory(artefacts[i].Path)
		if directory == "" || !strings.Contains(directory, "/partials/") {
			continue
		}
		dependencies[moduleName+"/"+directory] = artefacts[i].Content
	}

	return dependencies
}

// artefactDirectory returns the directory portion of an artefact path.
//
// Takes path (string) which is the artefact file path (e.g.,
// "dist/partials/partials_info_card_14ceb24d/generated.go").
//
// Returns string which is the directory (e.g.,
// "dist/partials/partials_info_card_14ceb24d"), or empty if the path has no directory
// separator.
func artefactDirectory(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash <= 0 {
		return ""
	}
	return path[:lastSlash]
}

// findMatchingPageFromManifest finds the most specific manifest page whose route matches
// the request URL and whose artefact was generated.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains all generated files.
// Takes manifest (*wasm_dto.GeneratedManifest) which contains page metadata.
// Takes requestURL (string) which is the URL to match.
//
// Returns *wasm_dto.GeneratedArtefact which is the matching artefact, or nil.
// Returns string which is the package path for the page.
func findMatchingPageFromManifest(artefacts []wasm_dto.GeneratedArtefact, manifest *wasm_dto.GeneratedManifest, requestURL string) (*wasm_dto.GeneratedArtefact, string) {
	candidates := rankRouteCandidates(manifest, requestURL)
	for i := range candidates {
		if artefact := findArtefactBySourcePath(artefacts, candidates[i].page.SourcePath); artefact != nil {
			return artefact, candidates[i].page.PackagePath
		}
	}
	return nil, ""
}

// rankRouteCandidates returns every manifest page with a route pattern matching the
// request URL, most specific first, so overlapping routes resolve the same way on every
// call.
//
// Takes manifest (*wasm_dto.GeneratedManifest) which contains page metadata.
// Takes requestURL (string) which is the URL to match.
//
// Returns []routeCandidate which holds the matching pages in priority order.
func rankRouteCandidates(manifest *wasm_dto.GeneratedManifest, requestURL string) []routeCandidate {
	if manifest == nil {
		return nil
	}
	var candidates []routeCandidate
	for _, pageKey := range slices.Sorted(maps.Keys(manifest.Pages)) {
		page := manifest.Pages[pageKey]
		var best *routeCandidate
		for _, pattern := range page.RoutePatterns {
			if !matchesRoute(pattern, requestURL) {
				continue
			}
			candidate := newRouteCandidate(page, pattern)
			if best == nil || compareRouteCandidates(candidate, *best) < 0 {
				best = &candidate
			}
		}
		if best != nil {
			candidates = append(candidates, *best)
		}
	}
	slices.SortFunc(candidates, compareRouteCandidates)
	return candidates
}

// newRouteCandidate measures a matching route pattern's specificity.
//
// Takes page (wasm_dto.ManifestPageEntry) which is the page the pattern belongs to.
// Takes pattern (string) which is the matching route pattern.
//
// Returns routeCandidate which carries the pattern's specificity measures.
func newRouteCandidate(page wasm_dto.ManifestPageEntry, pattern string) routeCandidate {
	candidate := routeCandidate{pattern: pattern, page: page, dynamicSegments: 0, segments: 0, catchAll: false}
	trimmed := strings.Trim(pattern, "/")
	if trimmed == "" {
		return candidate
	}
	for segment := range strings.SplitSeq(trimmed, "/") {
		candidate.segments++
		switch {
		case isCatchAllSegment(segment):
			candidate.catchAll = true
		case isDynamicSegment(segment):
			candidate.dynamicSegments++
		}
	}
	return candidate
}

// compareRouteCandidates orders route candidates from most to least specific.
//
// Takes left (routeCandidate) which is the first candidate.
// Takes right (routeCandidate) which is the second candidate.
//
// Returns int which is negative when left is more specific, positive when right is, and
// zero only for identical patterns on the same source.
func compareRouteCandidates(left, right routeCandidate) int {
	if left.catchAll != right.catchAll {
		if left.catchAll {
			return 1
		}
		return -1
	}
	return cmp.Or(
		cmp.Compare(left.dynamicSegments, right.dynamicSegments),
		cmp.Compare(right.segments, left.segments),
		cmp.Compare(left.pattern, right.pattern),
		cmp.Compare(left.page.SourcePath, right.page.SourcePath),
	)
}

// findArtefactBySourcePath finds a page artefact with the given source path.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains the list of artefacts to
// search through.
// Takes sourcePath (string) which is the path to match against.
//
// Returns *wasm_dto.GeneratedArtefact which is the matching artefact, or nil if no match
// is found.
func findArtefactBySourcePath(artefacts []wasm_dto.GeneratedArtefact, sourcePath string) *wasm_dto.GeneratedArtefact {
	for i := range artefacts {
		if artefacts[i].Type == wasm_dto.ArtefactTypePage && artefacts[i].SourcePath == sourcePath {
			return &artefacts[i]
		}
	}
	return nil
}

// findFirstPageArtefact finds the first page artefact to use as a fallback.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains the artefacts to search
// through.
// Takes manifest (*wasm_dto.GeneratedManifest) which provides package path lookup.
//
// Returns *wasm_dto.GeneratedArtefact which is the first page artefact found, or nil if
// none exists.
// Returns string which is the package path for the page, or empty if not found.
func findFirstPageArtefact(artefacts []wasm_dto.GeneratedArtefact, manifest *wasm_dto.GeneratedManifest) (*wasm_dto.GeneratedArtefact, string) {
	for i := range artefacts {
		if artefacts[i].Type != wasm_dto.ArtefactTypePage {
			continue
		}
		packagePath := lookupPackagePath(manifest, artefacts[i].SourcePath)
		return &artefacts[i], packagePath
	}
	return nil, ""
}

// lookupPackagePath finds the package path for an artefact in the manifest.
//
// Takes manifest (*wasm_dto.GeneratedManifest) which holds page metadata.
// Takes sourcePath (string) which is the artefact source path to look up.
//
// Returns string which is the package path, or empty if not found.
func lookupPackagePath(manifest *wasm_dto.GeneratedManifest, sourcePath string) string {
	if manifest == nil || manifest.Pages == nil {
		return ""
	}
	for _, pageKey := range slices.Sorted(maps.Keys(manifest.Pages)) {
		if page := manifest.Pages[pageKey]; page.SourcePath == sourcePath {
			return page.PackagePath
		}
	}
	return ""
}

// matchesRoute checks if a URL matches a route pattern.
//
// Takes pattern (string) which is the route pattern to match against.
// Takes url (string) which is the URL to check.
//
// Returns bool which is true if the URL matches the pattern.
func matchesRoute(pattern, url string) bool {
	if i := strings.IndexAny(url, "?#"); i >= 0 {
		url = url[:i]
	}
	pattern = strings.TrimSuffix(pattern, "/")
	url = strings.TrimSuffix(url, "/")

	if pattern == url {
		return true
	}

	if pattern == "" && (url == "" || url == "/") {
		return true
	}

	patternSegments := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	urlSegments := strings.Split(strings.TrimPrefix(url, "/"), "/")

	return matchSegments(patternSegments, urlSegments)
}

// matchSegments compares pattern segments against URL segments.
//
// Takes patternSegments ([]string) which contains the pattern path segments.
// Takes urlSegments ([]string) which contains the URL path segments.
//
// Returns bool which is true if all segments match.
func matchSegments(patternSegments, urlSegments []string) bool {
	for i, patternSegment := range patternSegments {
		if isCatchAllSegment(patternSegment) {
			return true
		}

		if i >= len(urlSegments) {
			return false
		}

		if isDynamicSegment(patternSegment) {
			if urlSegments[i] == "" {
				return false
			}

			continue
		}

		if patternSegment != urlSegments[i] {
			return false
		}
	}

	return len(patternSegments) == len(urlSegments)
}

// isDynamicSegment reports whether a path segment is a Chi-style dynamic parameter like
// {slug}.
//
// Takes segment (string) which is the path segment to check.
//
// Returns bool which is true if the segment is a dynamic parameter.
func isDynamicSegment(segment string) bool {
	return len(segment) > 2 && segment[0] == '{' && segment[len(segment)-1] == '}' && !strings.HasSuffix(segment, "*}")
}

// isCatchAllSegment reports whether a path segment is a Chi-style catch-all parameter
// like {path*}.
//
// Takes segment (string) which is the path segment to check.
//
// Returns bool which is true if the segment is a catch-all parameter.
func isCatchAllSegment(segment string) bool {
	return len(segment) > minCatchAllSegmentLength && segment[0] == '{' && strings.HasSuffix(segment, "*}")
}

// collectScriptArtefacts pulls every JavaScript artefact out of a generator response so
// the dynamic-render path can surface them to the consumer.
//
// Takes genResp (*wasm_dto.GenerateFromSourcesResponse) which carries every artefact
// captured during generation.
//
// Returns []wasm_dto.ScriptArtefact where each entry is one compiled ES module.
// Returns nil when genResp is nil or no JS artefacts were emitted; JSON-omitempty hides
// the field from the response in that case.
func collectScriptArtefacts(genResp *wasm_dto.GenerateFromSourcesResponse) []wasm_dto.ScriptArtefact {
	if genResp == nil || len(genResp.Artefacts) == 0 {
		return nil
	}

	scripts := make([]wasm_dto.ScriptArtefact, 0, len(genResp.Artefacts))
	for _, artefact := range genResp.Artefacts {
		if artefact.Type != wasm_dto.ArtefactTypeJS {
			continue
		}
		scripts = append(scripts, wasm_dto.ScriptArtefact{
			Path:    artefact.Path,
			Content: artefact.Content,
		})
	}
	if len(scripts) == 0 {
		return nil
	}
	return scripts
}

// findPageStyleBlock returns the aggregated CSS for the most specific page that matches
// requestURL. The block already includes CSS from every transitively referenced partial
// (the manifest builder collapses them at generation time), so it can be used verbatim as
// the page's <style> contents.
//
// Takes genResp (*wasm_dto.GenerateFromSourcesResponse) which carries the manifest
// produced this run.
// Takes requestURL (string) which is matched against each page entry's route patterns.
//
// Returns string which is the matched page's StyleBlock, or empty when no page matches or
// the manifest is absent.
func findPageStyleBlock(genResp *wasm_dto.GenerateFromSourcesResponse, requestURL string) string {
	if genResp == nil {
		return ""
	}
	candidates := rankRouteCandidates(genResp.Manifest, requestURL)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].page.StyleBlock
}
