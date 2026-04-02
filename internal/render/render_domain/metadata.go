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
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/maypok86/otter/v2"
	"golang.org/x/net/html"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

// devWidgetTag is the custom element tag name for the dev tools overlay widget.
const devWidgetTag = "piko-dev-widget"

const (
	// svgSymbolOverhead is the fixed character count for SVG symbol tags.
	// Format: <symbol id="ID" viewBox="VALUE">INNERHTML</symbol>
	// Breakdown: 13 (<symbol id=") + 1 (") + 11 ( viewBox=") + 1 (") + 1 (>) +
	// 9 (</symbol>) = 36 characters.
	svgSymbolOverhead = 36

	// defaultSVGIDSliceCapacity is the starting capacity for SVG ID slices.
	// Set to 32 elements, which is enough for most pages.
	defaultSVGIDSliceCapacity = 32
)

// ParsedSvgData contains the parsed content and attributes of an SVG asset.
// CachedSymbol is pre-computed at load time to avoid per-request allocation
// overhead.
type ParsedSvgData struct {
	// InnerHTML is the content between the opening and closing SVG tags.
	InnerHTML string

	// CachedSymbol is the pre-computed SVG symbol string for this asset.
	CachedSymbol string

	// Attributes holds the parsed SVG element attributes such as viewBox.
	Attributes []ast_domain.HTMLAttribute
}

// appendDevWidgetTag returns customTags with the dev widget tag appended when
// the dev widget is enabled.
//
// Takes tags ([]string) which holds the existing custom tags.
//
// Returns []string which contains the original tags plus the dev widget tag,
// or the original slice unmodified when the widget is disabled.
func appendDevWidgetTag(tags []string) []string {
	if getDevWidgetHTML() == "" {
		return tags
	}
	if slices.Contains(tags, devWidgetTag) {
		return tags
	}
	return append(tags, devWidgetTag)
}

var (
	svgIDSlicePool = sync.Pool{
		New: func() any {
			return new(make([]string, 0, defaultSVGIDSliceCapacity))
		},
	}

	// getSpriteSheetCache returns the lazily initialised sprite sheet cache.
	getSpriteSheetCache = sync.OnceValue(func() *otter.Cache[uint64, string] {
		return otter.Must(&otter.Options[uint64, string]{
			MaximumSize:      500,
			InitialCapacity:  32,
			ExpiryCalculator: otter.ExpiryWriting[uint64, string](30 * time.Minute),
		})
	})
)

// addLinkHeaderIfUnique adds a link header to the collection if it is not
// already present. Uses O(1) map-based deduplication instead of O(n) linear
// search.
//
// Takes lh (render_dto.LinkHeader) which is the link header to add.
//
// Safe for concurrent use.
func (rctx *renderContext) addLinkHeaderIfUnique(lh render_dto.LinkHeader) {
	rctx.muCollectedLinkHeaders.Lock()
	defer rctx.muCollectedLinkHeaders.Unlock()

	key := linkHeaderKey{
		URL: lh.URL,
		Rel: lh.Rel,
		As:  lh.As,
	}

	if _, exists := rctx.linkHeaderSet[key]; exists {
		return
	}

	rctx.linkHeaderSet[key] = struct{}{}
	rctx.collectedLinkHeaders = append(rctx.collectedLinkHeaders, lh)
}

// CollectMetadata gathers and generates Link headers for preloading assets.
// It preloads component JavaScript modules, SVG assets, fonts, and generates
// appropriate preconnect headers for external resources.
//
// Takes request (*http.Request) which provides request context for asset paths.
// Takes metadata (*templater_dto.InternalMetadata) which contains asset refs
// and custom tags to process.
// Takes siteConfig (*config.WebsiteConfig) which provides font configurations;
// may be nil.
//
// Returns []render_dto.LinkHeader which contains the collected Link
// headers for preloading.
// Returns *ProbeData which contains component probe metadata, or nil
// when no component metadata is present.
// Returns error when metadata collection fails.
func (ro *RenderOrchestrator) CollectMetadata(
	ctx context.Context,
	request *http.Request,
	metadata *templater_dto.InternalMetadata,
	siteConfig *config.WebsiteConfig,
) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
	CollectMetadataCount.Add(ctx, 1)

	tempRenderCtx := ro.getRenderContext(ctx, "", nil, request, nil)

	ro.preloadAssetsAndComponentsForTags(
		ctx, appendDevWidgetTag(metadata.CustomTags), tempRenderCtx)
	addStandardLinkHeaders(tempRenderCtx)
	addComponentsExtensionLinkHeader(tempRenderCtx)
	addJSLinkHeaders(metadata.JSScriptMetas, tempRenderCtx)

	if siteConfig != nil {
		processFontConfigurations(siteConfig.Fonts, tempRenderCtx)
	}

	var probeData *render_dto.ProbeData
	if tempRenderCtx.componentMetadata != nil {
		probeData = render_dto.AcquireProbeData()
		probeData.ComponentMetadata = tempRenderCtx.componentMetadata
		tempRenderCtx.componentMetadata = nil
	}

	headers := tempRenderCtx.collectedLinkHeaders
	ro.putRenderContext(tempRenderCtx)

	return headers, probeData, nil
}

// preloadAssetsAndComponentsForTags fetches component metadata in bulk and
// adds modulepreload link headers for each component with a JS path.
//
// Takes customTags ([]string) which lists the component tags to preload.
// Takes rctx (*renderContext) which provides the registry and collects
// link headers.
func (ro *RenderOrchestrator) preloadAssetsAndComponentsForTags(
	ctx context.Context,
	customTags []string,
	rctx *renderContext,
) {
	ctx, l := logger_domain.From(ctx, log)

	if ro.registry == nil || len(customTags) == 0 {
		return
	}

	results, err := ro.registry.BulkGetComponentMetadata(ctx, customTags)
	if err != nil {
		l.Warn("Error bulk-fetching component metadata for preload",
			logger_domain.Error(err))
		return
	}

	rctx.componentMetadata = results

	for _, compMeta := range results {
		if compMeta != nil && compMeta.BaseJSPath != "" {
			rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
				URL:         compMeta.BaseJSPath,
				Rel:         "modulepreload",
				As:          "script",
				Type:        "",
				CrossOrigin: "",
			})
		}
	}
}

// ensureComponentMetadata populates rctx.componentMetadata if it has not
// already been set. This is needed because CollectMetadata and RenderAST
// use separate render contexts, so the bulk fetch from
// preloadAssetsAndComponentsForTags does not carry over.
//
// Takes ctx (context.Context) which provides the request context.
// Takes customTags ([]string) which lists the component tags to fetch.
// Takes rctx (*renderContext) which receives the fetched metadata.
func (ro *RenderOrchestrator) ensureComponentMetadata(ctx context.Context, customTags []string, rctx *renderContext) {
	if rctx.componentMetadata != nil || ro.registry == nil || len(customTags) == 0 {
		return
	}
	ctx, l := logger_domain.From(ctx, log)
	results, err := ro.registry.BulkGetComponentMetadata(ctx, customTags)
	if err != nil {
		l.Warn("Error bulk-fetching component metadata for render",
			logger_domain.Error(err))
		return
	}
	rctx.componentMetadata = results
}

// buildPreloadLogic generates HTML for preloading component scripts,
// reading from rctx.componentMetadata which must be populated by
// ensureComponentMetadata before this call.
//
// Takes componentTypes ([]string) which lists the component tags to process.
// Takes noPreloadsGlobally (bool) which disables all preloading when true.
// Takes rctx (*renderContext) which provides pooled buffers for HTML
// generation.
//
// Returns preloadHTML (string) which contains link rel="modulepreload" tags.
// Returns scriptHTML (string) which contains script tags for eager loading.
func (*RenderOrchestrator) buildPreloadLogic(
	componentTypes []string,
	noPreloadsGlobally bool,
	rctx *renderContext,
) (preloadHTML string, scriptHTML string) {
	if noPreloadsGlobally || len(componentTypes) == 0 {
		return "", ""
	}

	if rctx.componentMetadata == nil {
		return "", ""
	}

	sortedCompTags := make([]string, 0, len(componentTypes))
	sortedCompTags = append(sortedCompTags, componentTypes...)
	slices.SortFunc(sortedCompTags, strings.Compare)

	preloadBuf := rctx.getBuffer()
	scriptBuf := rctx.getBuffer()

	for _, compTag := range sortedCompTags {
		meta, ok := rctx.componentMetadata[compTag]
		if !ok || meta == nil || meta.BaseJSPath == "" {
			continue
		}
		appendPreloadTags(preloadBuf, scriptBuf, meta.BaseJSPath, meta.SRIHash)
	}

	preloadHTML = rctx.freezeToString(preloadBuf)
	scriptHTML = rctx.freezeToString(scriptBuf)

	return preloadHTML, scriptHTML
}

// buildSvgSpriteSheet generates an SVG sprite sheet from required symbols.
// Results are cached by the hash of sorted symbol IDs for efficient reuse.
//
// Takes rctx (*renderContext) which provides the symbols to include.
//
// Returns string which is the assembled sprite sheet markup, or empty if no
// symbols are required.
// Returns error when sprite sheet generation fails.
func (*RenderOrchestrator) buildSvgSpriteSheet(rctx *renderContext) (string, error) {
	BuildSvgSpriteSheetCount.Add(rctx.originalCtx, 1)

	if len(rctx.requiredSvgSymbols) == 0 {
		return "", nil
	}

	svgIDsPtr := getSvgIDSlice()
	extractSVGIDs(rctx.requiredSvgSymbols, svgIDsPtr)
	svgIDs := *svgIDsPtr
	defer putSvgIDSlice(svgIDsPtr)

	cacheKey := computeSpriteSheetKey(svgIDs)
	cache := getSpriteSheetCache()
	if cached, ok := cache.GetIfPresent(cacheKey); ok {
		SpriteSheetCacheHitCount.Add(rctx.originalCtx, 1)
		return cached, nil
	}

	symbols := make([]string, len(rctx.requiredSvgSymbols))
	for i := range rctx.requiredSvgSymbols {
		if rctx.requiredSvgSymbols[i].data != nil {
			symbols[i] = rctx.requiredSvgSymbols[i].data.CachedSymbol
			SVGSymbolCount.Add(rctx.originalCtx, 1)
		}
	}

	result := assembleSpriteSheet(symbols, rctx)

	cache.Set(cacheKey, strings.Clone(result))
	return result, nil
}

// ShutdownSpriteSheetCache stops the sprite sheet cache's background
// goroutines. Call during application shutdown or test cleanup.
func ShutdownSpriteSheetCache() {
	getSpriteSheetCache().StopAllGoroutines()
}

// ClearSpriteSheetCacheForTesting clears the sprite sheet cache.
// Only for use in tests.
func ClearSpriteSheetCacheForTesting() {
	getSpriteSheetCache().InvalidateAll()
}

// ComputeSymbolString builds an SVG symbol string from parsed SVG data.
// This is called once when the SVG loads and is cached in
// ParsedSvgData.CachedSymbol to avoid repeated memory use per request.
//
// Takes id (string) which specifies the symbol identifier.
// Takes parsedData (*ParsedSvgData) which provides the parsed SVG content.
//
// Returns string which contains the formatted symbol element, or an empty
// string if parsedData is nil.
//
// Output format: <symbol id="escaped-id" viewBox="value">innerHTML</symbol>
func ComputeSymbolString(id string, parsedData *ParsedSvgData) string {
	if parsedData == nil {
		return ""
	}

	var viewBox string
	for i := range parsedData.Attributes {
		if parsedData.Attributes[i].Name == "viewBox" {
			viewBox = parsedData.Attributes[i].Value
			break
		}
	}

	builder := getBuilder()

	estimatedSize := svgSymbolOverhead + len(id) + len(viewBox) + len(parsedData.InnerHTML)
	builder.Grow(estimatedSize)

	builder.WriteString(`<symbol id="`)
	builder.WriteString(html.EscapeString(id))
	_ = builder.WriteByte('"')

	if viewBox != "" {
		builder.WriteString(` viewBox="`)
		builder.WriteString(html.EscapeString(viewBox))
		_ = builder.WriteByte('"')
	}

	_ = builder.WriteByte('>')
	builder.WriteString(parsedData.InnerHTML)
	builder.WriteString(`</symbol>`)

	result := builder.String()
	putBuilder(builder)
	return result
}

// formatComponentNotFoundError creates an error message for when a component
// cannot be found.
//
// Takes componentTag (string) which is the name of the missing component.
//
// Returns string which contains the error message with steps to fix the issue.
func formatComponentNotFoundError(componentTag string) string {
	return fmt.Sprintf(
		"Component '%s' is referenced in your page but does not exist.\n"+
			"To fix this:\n"+
			"  1. Create the component file: components/%s.pkc\n"+
			"  2. Or remove the <%s> tag from your page if it's not needed\n"+
			"  3. Run the build generator to register the component\n\n"+
			"See: https://piko.sh/docs/guide/client-components",
		componentTag, componentTag, componentTag,
	)
}

// addStandardLinkHeaders adds the standard framework preload headers to the
// render context. These headers tell the browser to fetch key assets early.
//
// Takes rctx (*renderContext) which receives the link headers.
func addStandardLinkHeaders(rctx *renderContext) {
	rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
		URL:         "/_piko/dist/ppframework.core.es.js",
		Rel:         "preload",
		As:          "script",
		Type:        "",
		CrossOrigin: "",
	})
	rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
		URL:         "/theme.css",
		Rel:         "preload",
		As:          "style",
		Type:        "",
		CrossOrigin: "",
	})
}

// addComponentsExtensionLinkHeader adds a modulepreload link header for the
// components extension when the page contains PKC components. This allows HTTP
// 103 Early Hints to fetch the component runtime before the HTML body arrives.
//
// Takes rctx (*renderContext) which provides the component metadata and
// receives the link header.
func addComponentsExtensionLinkHeader(rctx *renderContext) {
	if rctx.componentMetadata == nil {
		return
	}
	rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
		URL:         "/_piko/dist/ppframework.components.es.js",
		Rel:         "modulepreload",
		As:          "script",
		Type:        "",
		CrossOrigin: "",
	})
}

// addJSLinkHeaders adds modulepreload link headers for client-side JavaScript
// needed by the page. This includes the page's own script and scripts from
// embedded partials, allowing HTTP 103 Early Hints for parallel downloads.
//
// Takes jsMetas ([]templater_dto.JSScriptMeta) which contains the script
// metadata.
// Takes rctx (*renderContext) which receives the link headers.
func addJSLinkHeaders(jsMetas []templater_dto.JSScriptMeta, rctx *renderContext) {
	for _, meta := range jsMetas {
		rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
			URL:         meta.URL,
			Rel:         "modulepreload",
			As:          "script",
			Type:        "",
			CrossOrigin: "",
		})
	}
}

// processFontConfigurations adds link headers for font preloading and
// preconnect to Google Fonts servers.
//
// Takes fonts ([]config.FontDefinition) which contains the font definitions to
// process.
// Takes rctx (*renderContext) which receives the link headers that are added.
func processFontConfigurations(fonts []config.FontDefinition, rctx *renderContext) {
	for _, font := range fonts {
		if isGoogleFontsURL(font.URL) {
			rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
				URL:         "https://fonts.googleapis.com",
				Rel:         "preconnect",
				As:          "",
				Type:        "",
				CrossOrigin: "",
			})
			rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
				URL:         "https://fonts.gstatic.com",
				Rel:         "preconnect",
				As:          "",
				Type:        "",
				CrossOrigin: "anonymous",
			})
		}

		asType, fontType := determineFontAssetType(font.URL)
		rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
			URL:         font.URL,
			Rel:         "preload",
			As:          asType,
			Type:        fontType,
			CrossOrigin: "anonymous",
		})
	}
}

// isGoogleFontsURL checks whether the given URL points to a Google Fonts
// domain.
//
// Takes url (string) which is the URL to check.
//
// Returns bool which is true if the URL contains fonts.googleapis.com or
// fonts.gstatic.com.
func isGoogleFontsURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, "fonts.googleapis.com") ||
		strings.Contains(lower, "fonts.gstatic.com")
}

// determineFontAssetType returns the asset type and MIME type for a font URL.
//
// Takes url (string) which is the font URL to check.
//
// Returns asType (string) which is "font" for woff2 files or "style" for
// other files.
// Returns fontType (string) which is the MIME type for woff2 files or empty
// for other files.
func determineFontAssetType(url string) (asType, fontType string) {
	if strings.HasSuffix(url, ".woff2") {
		return "font", "font/woff2"
	}
	return "style", ""
}

// appendPreloadTags appends modulepreload and script tags for a JavaScript
// file to byte buffers, with optional SRI integrity attributes.
//
// Takes preload (*[]byte) which receives the modulepreload link tag.
// Takes script (*[]byte) which receives the script module tag.
// Takes jsFile (string) which is the path to the JavaScript file.
// Takes sriHash (string) which is the SRI integrity hash, or empty to omit.
//
// Uses escapeIfNeeded to avoid memory use when the path has no special
// characters. Uses the []byte append pattern for zero-copy freeze support.
func appendPreloadTags(preload, script *[]byte, jsFile, sriHash string) {
	escaped := escapeIfNeeded(jsFile)

	*preload = append(*preload, `<link rel="modulepreload" href="`...)
	*preload = append(*preload, escaped...)
	*preload = append(*preload, `"`...)
	if sriHash != "" {
		*preload = append(*preload, ` integrity="`...)
		*preload = append(*preload, sriHash...)
		*preload = append(*preload, `" crossorigin="anonymous"`...)
	}
	*preload = append(*preload, `>`...)

	*script = append(*script, `<script type="module" src="`...)
	*script = append(*script, escaped...)
	*script = append(*script, `"`...)
	if sriHash != "" {
		*script = append(*script, ` integrity="`...)
		*script = append(*script, sriHash...)
		*script = append(*script, `" crossorigin="anonymous"`...)
	}
	*script = append(*script, `></script>`...)
}

// escapeIfNeeded returns the HTML-escaped version of s only if s contains
// characters that need escaping.
//
// For clean paths (no &, <, >, ", ') this avoids the memory allocation from
// html.EscapeString.
//
// Takes s (string) which is the input to check and escape if needed.
//
// Returns string which is either the original string or the escaped version.
func escapeIfNeeded(s string) string {
	if needsHTMLEscape(s) {
		return html.EscapeString(s)
	}
	return s
}

// needsHTMLEscape checks if a string contains characters that need HTML
// escaping. It looks for the five characters that html.EscapeString handles:
// &, <, >, ", and '.
//
// Takes s (string) which is the text to check.
//
// Returns bool which is true if the string contains any of these characters.
func needsHTMLEscape(s string) bool {
	for i := range len(s) {
		switch s[i] {
		case '&', '<', '>', '"', '\'':
			return true
		}
	}
	return false
}

// getSvgIDSlice retrieves a string slice from the pool.
//
// Returns *[]string which is a slice ready for use, either from the pool or
// newly allocated.
func getSvgIDSlice() *[]string {
	s, ok := svgIDSlicePool.Get().(*[]string)
	if !ok {
		return new(make([]string, 0, defaultSVGIDSliceCapacity))
	}
	return s
}

// putSvgIDSlice returns a string slice to the pool after clearing it.
//
// Takes s (*[]string) which is the slice to clear and return to the pool.
func putSvgIDSlice(s *[]string) {
	*s = (*s)[:0]
	svgIDSlicePool.Put(s)
}

// computeSpriteSheetKey creates a stable hash of sorted symbol IDs.
// The IDs must be pre-sorted to ensure consistent keys.
//
// Takes svgIDs ([]string) which is the sorted list of SVG identifiers.
//
// Returns uint64 which is the FNV-64a hash of the concatenated IDs.
func computeSpriteSheetKey(svgIDs []string) uint64 {
	h := getHasher()
	for _, id := range svgIDs {
		_, _ = h.Write(mem.Bytes(id))
		_, _ = h.Write([]byte{0})
	}
	sum := h.Sum64()
	putHasher(h)
	return sum
}

// extractSVGIDs copies IDs from symbol entries into a string slice.
//
// Takes entries ([]svgSymbolEntry) which holds the SVG entries.
// Takes result (*[]string) which receives the extracted IDs.
func extractSVGIDs(entries []svgSymbolEntry, result *[]string) {
	*result = (*result)[:0]
	for i := range entries {
		*result = append(*result, entries[i].id)
	}
}

// collectAndSortSVGIDs extracts SVG IDs from symbol entries and returns them
// sorted for consistent output ordering.
//
// Takes entries ([]svgSymbolEntry) which holds the SVG entries to collect from.
//
// Returns []string which holds the sorted SVG IDs.
func collectAndSortSVGIDs(entries []svgSymbolEntry) []string {
	svgIDs := make([]string, 0, len(entries))
	for i := range entries {
		svgIDs = append(svgIDs, entries[i].id)
	}
	slices.SortFunc(svgIDs, strings.Compare)
	return svgIDs
}

// assembleSpriteSheet builds the final SVG sprite sheet from processed symbols.
//
// Uses request-level buffer pooling with zero-copy string conversion. The
// buffer is kept alive until the request ends, making the conversion safe.
//
// Takes symbols ([]string) which contains the processed SVG symbol elements.
// Takes rctx (*renderContext) which provides the pooled buffer and conversion.
//
// Returns string which is the complete SVG sprite sheet document.
func assembleSpriteSheet(symbols []string, rctx *renderContext) string {
	var totalSize int
	for _, s := range symbols {
		totalSize += len(s)
	}
	totalSize += len(`<svg xmlns="http://www.w3.org/2000/svg" id="sprite" style="display: none;"></svg>`)

	buffer := rctx.getBuffer()

	if cap(*buffer) < totalSize {
		newBuf := make([]byte, 0, totalSize)
		*buffer = newBuf
	}

	*buffer = append(*buffer, `<svg xmlns="http://www.w3.org/2000/svg" id="sprite" style="display: none;">`...)
	for _, symbol := range symbols {
		if symbol != "" {
			*buffer = append(*buffer, symbol...)
		}
	}
	*buffer = append(*buffer, `</svg>`...)

	return rctx.freezeToString(buffer)
}
