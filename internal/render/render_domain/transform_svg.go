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
	"cmp"
	"context"
	"fmt"
	"hash"
	"hash/fnv"
	"html"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/mem"
)

const (
	// classAttrOverhead is the number of extra bytes needed for a class attribute.
	classAttrOverhead = 9

	// attributeOverhead is the extra bytes for each HTML attribute: the space before
	// the name and the quotes around its value.
	attributeOverhead = 4

	// sortThresholdTwo is the count at which inline sorting handles two elements.
	sortThresholdTwo = 2

	// sortThresholdThree is the count at which manual three-element sorting is
	// used.
	sortThresholdThree = 3

	// defaultClassBufCapacity is the initial capacity for class collection
	// buffers.
	defaultClassBufCapacity = 256

	// maxClassBufCapacity is the maximum buffer size to keep in the pool.
	// Buffers larger than this are discarded to prevent memory bloat.
	maxClassBufCapacity = 4096

	// intConvBufferSize is the buffer size for converting integers to strings.
	intConvBufferSize = 20

	// floatConvBufferSize is the buffer size in bytes for float to string
	// conversion.
	floatConvBufferSize = 32

	// floatConvBitSize is the bit size for float conversion (64 for float64).
	floatConvBitSize = 64

	// intConvBase is the number base used when converting integers to strings.
	intConvBase = 10

	// maxCombinedAttrCount is the maximum number of static and dynamic attributes
	// that can use the fast path without heap allocation.
	maxCombinedAttrCount = 16
)

var (
	// nullSeparator is a single-byte separator for hash input.
	// Declared at package level to avoid repeated allocation.
	nullSeparator = []byte{0}

	// classSetPool reuses maps for class deduplication to reduce GC pressure.
	classSetPool = sync.Pool{
		New: func() any {
			return make(map[string]struct{}, 16)
		},
	}

	// sortedKeysPool reuses string slices to reduce allocation pressure during
	// SVG attribute key sorting.
	sortedKeysPool = sync.Pool{
		New: func() any {
			return new(make([]string, 16))
		},
	}

	// hashPool is a sync.Pool that reuses FNV-1a hashers to reduce allocation
	// overhead. This eliminates approximately 50MB of hasher allocations and
	// provides a 5-10% speedup.
	hashPool = sync.Pool{
		New: func() any {
			return fnv.New64a()
		},
	}

	// classCollectionBufPool provides reusable byte buffers for collecting class
	// values during SVG attribute merging. This avoids the strings.Builder
	// allocations that were causing ~0.55GB of allocations.
	classCollectionBufPool = sync.Pool{
		New: func() any {
			return &classCollectionBufWrapper{
				buffer: make([]byte, 0, defaultClassBufCapacity),
			}
		},
	}

	// svgNonUserAttrs is a set of attributes that are not user attributes.
	// Parser lowercases all attribute names.
	svgNonUserAttrs = map[string]struct{}{
		attributeSrc: {},
		tagPikoSvg:   {},
	}
)

// svgCacheKey is a struct-based map key for the merged attributes cache.
// Using a struct key eliminates the []byte allocation required by string
// key generation.
type svgCacheKey struct {
	// artefactID is the unique identifier for the SVG artefact.
	artefactID string

	// userAttrsHash is a hash of the attributes set by the user.
	userAttrsHash uint64
}

// classCollectionBufWrapper wraps a byte buffer to avoid heap escape when
// pooling. Storing *[]byte causes the slice header to escape; wrapping in a
// struct with a pre-set slice avoids this.
type classCollectionBufWrapper struct {
	// buffer holds collected class names for reuse through sync.Pool.
	buffer []byte
}

// attributeReference holds a reference to an attribute, either static or dynamic.
// It avoids copying strings into a new struct.
type attributeReference struct {
	// writer holds the DirectWriter for dynamic attributes; nil if static.
	writer *ast_domain.DirectWriter

	// name is the attribute name.
	name string

	// value holds the attribute value for static attributes.
	value string
}

// classAnalysis holds the results from parsing a class string.
type classAnalysis struct {
	// estimatedCapacity is the expected number of classes based on separator
	// count.
	estimatedCapacity int

	// firstSpaceIndex is the index of the first space character; -1 if none.
	firstSpaceIndex int

	// hasSeparator indicates whether the string contains whitespace characters.
	hasSeparator bool

	// onlySpaces is true when all separators are spaces, not tabs or newlines.
	onlySpaces bool
}

// stringTokeniser provides a way to loop through words in a string without
// using extra memory. It is faster than strings.Fields.
type stringTokeniser struct {
	// input is the input string to tokenise.
	input string

	// current holds the current parsed token value.
	current string

	// position is the current byte position in the string being tokenised.
	position int
}

// Next moves the tokeniser forward to the next token.
//
// Returns bool which is true if a token was found, or false if there are no
// more tokens.
func (t *stringTokeniser) Next() bool {
	for t.position < len(t.input) {
		r, size := utf8.DecodeRuneInString(t.input[t.position:])
		if !unicode.IsSpace(r) {
			break
		}
		t.position += size
	}

	if t.position >= len(t.input) {
		t.current = ""
		return false
	}

	start := t.position
	for t.position < len(t.input) {
		r, size := utf8.DecodeRuneInString(t.input[t.position:])
		if unicode.IsSpace(r) {
			break
		}
		t.position += size
	}

	t.current = t.input[start:t.position]
	return true
}

// Token returns the current token.
//
// Returns string which is the token at the current position.
func (t *stringTokeniser) Token() string {
	return t.current
}

// getClassCollectionBufWrapper gets a buffer wrapper from the pool.
//
// Returns *classCollectionBufWrapper which is a reset buffer ready for use.
func getClassCollectionBufWrapper() *classCollectionBufWrapper {
	if w, ok := classCollectionBufPool.Get().(*classCollectionBufWrapper); ok {
		w.buffer = w.buffer[:0]
		return w
	}
	return &classCollectionBufWrapper{buffer: make([]byte, 0, defaultClassBufCapacity)}
}

// putClassCollectionBufWrapper returns a buffer wrapper to the pool.
//
// Takes w (*classCollectionBufWrapper) which is the wrapper to return.
func putClassCollectionBufWrapper(w *classCollectionBufWrapper) {
	if w == nil || cap(w.buffer) > maxClassBufCapacity {
		return
	}
	w.buffer = w.buffer[:0]
	classCollectionBufPool.Put(w)
}

// renderPikoSvg is a specialised renderer that writes <piko:svg> directly to
// the output stream without mutating the AST. This implements the
// "direct-to-writer" pattern for maximum performance.
//
// Takes n (*ast_domain.TemplateNode) which is the piko:svg node to
// render.
// Takes qw (*qt.Writer) which is the output writer for the rendered
// SVG markup.
// Takes rctx (*renderContext) which provides registry access, caching,
// and rendering state.
//
// Returns error which is always nil; SVG errors are rendered inline as
// HTML comments.
//
// Performance benefits over transformPpSvg:
//   - Eliminates AST mutation overhead (no n.Attributes = ... assignment)
//   - Single-pass rendering (no mutate-then-iterate pattern)
//   - Reduces allocations by writing cached attributes directly to stream
//
// This function handles the complete rendering of a <piko:svg> element,
// including:
//   - Loading SVG data from registry
//   - Merging attributes with caching
//   - Writing final <svg><use></use></svg> structure directly
//
// NOTE: SVGs don't currently support CSRF, event directives, or p-ref because:
//  1. The output structure is <svg><use></use></svg> - a self-contained inline
//     element
//  2. Attribute caching would be invalidated by per-request CSRF tokens
//  3. Event handlers on inline SVGs are uncommon; use wrapper elements if
//     needed
func renderPikoSvg(_ *RenderOrchestrator, n *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext) error {
	SVGTransformCount.Add(rctx.originalCtx, 1)

	svgSrc := getSvgSrcOnly(n.Attributes, n.AttributeWriters)
	if svgSrc == "" {
		handleMissingSVGSrc(n.Attributes, qw, rctx)
		return nil
	}

	artefactID := svgSrc

	parsedData, err := rctx.registry.GetAssetRawSVG(rctx.originalCtx, artefactID)
	if err != nil {
		handleSVGLoadError(n.Attributes, artefactID, err, qw, rctx)
		return nil
	}

	registerSVGSymbol(artefactID, parsedData, rctx)

	cacheKey := svgCacheKey{
		artefactID:    artefactID,
		userAttrsHash: hashUserAttrsDirect(n.Attributes, n.AttributeWriters),
	}
	initialiseCacheIfNeeded(rctx)

	if cachedAttrString, exists := rctx.mergedAttrsCache[cacheKey]; exists {
		writeSVGWithAttrs(qw, cachedAttrString, artefactID)
		return nil
	}

	attributeString := mergeAndCacheAttrs(n.Attributes, n.AttributeWriters, parsedData.Attributes, cacheKey, rctx)
	writeSVGWithAttrs(qw, attributeString, artefactID)

	return nil
}

// handleMissingSVGSrc handles a piko:svg tag that has no source attribute.
// It logs a warning, adds to the error count, and writes an error div.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the tag attributes.
// Takes qw (*qt.Writer) which writes the error output.
// Takes rctx (*renderContext) which provides the rendering context.
func handleMissingSVGSrc(attrs []ast_domain.HTMLAttribute, qw *qt.Writer, rctx *renderContext) {
	userAttrs := extractUserAttrsOnly(attrs)
	rctx.diagnostics.AddWarning("renderPikoSvg",
		"piko:svg tag missing source attribute",
		map[string]string{"pageID": rctx.pageID})
	SVGTransformErrorCount.Add(rctx.originalCtx, 1)
	writeErrorDiv(qw, userAttrs, "<!-- piko:svg error: 'source' attribute is missing -->")
}

// handleSVGLoadError writes an error div when an SVG fails to load.
//
// It logs the error to diagnostics, updates error metrics, and writes an
// HTML error div with a comment showing the failure details.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the HTML attributes
// from the original element.
// Takes artefactID (string) which identifies the SVG that failed to load.
// Takes err (error) which is the original load error.
// Takes qw (*qt.Writer) which is the output writer for the error div.
// Takes rctx (*renderContext) which provides diagnostics and metrics context.
func handleSVGLoadError(attrs []ast_domain.HTMLAttribute, artefactID string, err error, qw *qt.Writer, rctx *renderContext) {
	userAttrs := extractUserAttrsOnly(attrs)
	rctx.diagnostics.AddError("renderPikoSvg", err,
		"Failed to get parsed SVG data from registry",
		map[string]string{"artefactID": artefactID})
	SVGTransformErrorCount.Add(rctx.originalCtx, 1)
	errMessage := fmt.Sprintf("<!-- piko:svg error: failed to load/parse '%s': %v -->",
		html.EscapeString(artefactID), err)
	writeErrorDiv(qw, userAttrs, errMessage)
}

// registerSVGSymbol records an SVG artefact and its pre-fetched data so that
// buildSvgSpriteSheet can read CachedSymbol directly without re-fetching.
// Duplicate IDs are silently ignored (linear scan; typical N < 20).
//
// Takes artefactID (string) which identifies the SVG symbol to add.
// Takes data (*ParsedSvgData) which holds the pre-fetched SVG content.
// Takes rctx (*renderContext) which holds the render state.
func registerSVGSymbol(artefactID string, data *ParsedSvgData, rctx *renderContext) {
	for i := range rctx.requiredSvgSymbols {
		if rctx.requiredSvgSymbols[i].id == artefactID {
			return
		}
	}
	rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols, svgSymbolEntry{
		id:   artefactID,
		data: data,
	})
}

// initialiseCacheIfNeeded sets up the merged attributes cache if it has not been
// created yet.
//
// Takes rctx (*renderContext) which holds the cache to set up.
func initialiseCacheIfNeeded(rctx *renderContext) {
	if rctx.mergedAttrsCache == nil {
		rctx.mergedAttrsCache = make(map[svgCacheKey]string, standardSortedKeysSize)
	}
}

// mergeAndCacheAttrs merges user and loaded attributes, caches the result, and
// returns the attribute string.
//
// Uses request-level buffer pooling with zero-copy conversion. The buffer is
// kept alive until the request ends (in putRenderContext), making the zero-copy
// string conversion safe. Uses inline filtering to avoid slice allocation.
//
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which contains the user-defined
// static attributes from the SVG node.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which contains dynamic
// attributes from the SVG node.
// Takes loadedAttrs ([]ast_domain.HTMLAttribute) which contains the attributes
// loaded from the SVG file.
// Takes cacheKey (svgCacheKey) which identifies this attribute combination for
// caching.
// Takes rctx (*renderContext) which provides the buffer pool and cache storage.
//
// Returns string which contains the merged attributes ready for output.
func mergeAndCacheAttrs(nodeAttrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter, loadedAttrs []ast_domain.HTMLAttribute, cacheKey svgCacheKey, rctx *renderContext) string {
	buffer := rctx.getBuffer()

	size := estimateMergedAttrsSizeInline(loadedAttrs, nodeAttrs, attributeWriters)
	if cap(*buffer) < size {
		newBuf := make([]byte, 0, size)
		*buffer = newBuf
	}

	*buffer = appendMergedSvgAttributesInline(*buffer, loadedAttrs, nodeAttrs, attributeWriters)

	attributeString := rctx.freezeToString(buffer)
	rctx.mergedAttrsCache[cacheKey] = attributeString
	return attributeString
}

// writeSVGWithAttrs writes a complete SVG element with the given attributes.
//
// Takes qw (*qt.Writer) which receives the SVG output.
// Takes attributeString (string) which contains the prepared SVG attributes.
// Takes artefactID (string) which identifies the SVG symbol to reference.
func writeSVGWithAttrs(qw *qt.Writer, attributeString, artefactID string) {
	qw.N().S("<svg")
	qw.N().S(attributeString)
	qw.N().S(`><use href="#`)
	qw.N().S(html.EscapeString(artefactID))
	qw.N().S(`"></use></svg>`)
}

// estimateMergedAttrsSizeInline works out the buffer size needed for merged
// attributes by checking node attributes one by one. This avoids making a new
// slice, unlike extractUserAttrsOnly.
//
// Takes loadedAttrs ([]ast_domain.HTMLAttribute) which contains the attributes
// from the symbol definition.
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which contains the fixed
// attributes from the current node being rendered.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which contains the changing
// attributes from the current node.
//
// Returns int which is the estimated total buffer size for the merged
// attributes.
func estimateMergedAttrsSizeInline(loadedAttrs, nodeAttrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter) int {
	size, classSize := estimateLoadedAttrsSize(loadedAttrs)
	userSize, userClassSize := estimateUserAttrsSize(nodeAttrs, classSize)
	size += userSize

	for _, dw := range attributeWriters {
		if dw == nil || !isUserAttr(dw.Name) {
			continue
		}
		if dw.Name == attributeClass {
			if userClassSize > 0 {
				userClassSize++
			}
			userClassSize += writerValueLen(dw)
		} else {
			size += attributeOverhead + len(dw.Name) + writerValueLen(dw)
		}
	}

	if userClassSize > 0 {
		size += classAttrOverhead + userClassSize
	}

	return size
}

// estimateLoadedAttrsSize works out the size of loaded SVG attributes.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// measure.
//
// Returns size (int) which is the total size of non-class attributes,
// including overhead per attribute.
// Returns classSize (int) which is the total length of class attribute values,
// with spaces added between multiple class values.
func estimateLoadedAttrsSize(attrs []ast_domain.HTMLAttribute) (size, classSize int) {
	for i := range attrs {
		if attrs[i].Name == attributeClass {
			if classSize > 0 {
				classSize++
			}
			classSize += len(attrs[i].Value)
		} else {
			size += attributeOverhead + len(attrs[i].Name) + len(attrs[i].Value)
		}
	}
	return size, classSize
}

// estimateUserAttrsSize works out the size of user attributes, skipping source
// and piko:svg attributes.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the user attributes
// to measure.
// Takes classSize (int) which is the current class attribute size.
//
// Returns size (int) which is the total size of non-class attributes.
// Returns finalClassSize (int) which is the updated class size, including any
// class values from the user attributes.
func estimateUserAttrsSize(attrs []ast_domain.HTMLAttribute, classSize int) (size, finalClassSize int) {
	finalClassSize = classSize
	for i := range attrs {
		name := attrs[i].Name
		if name == attributeSrc || name == tagPikoSvg {
			continue
		}
		if name == attributeClass {
			if finalClassSize > 0 {
				finalClassSize++
			}
			finalClassSize += len(attrs[i].Value)
		} else {
			size += attributeOverhead + len(name) + len(attrs[i].Value)
		}
	}
	return size, finalClassSize
}

// appendMergedSvgAttributesInline merges attributes and filters them inline.
// This removes the need for slice allocation from extractUserAttrsOnly.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes loadedAttrs ([]ast_domain.HTMLAttribute) which contains attributes
// parsed from the SVG file.
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which contains fixed attributes
// from the AST node.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which contains dynamic
// attributes from the AST node.
//
// Returns []byte which is the buffer with merged attributes appended.
//
// Note: Uses ordered merge instead of map and sort for stable output.
//   - Stable order comes from fixed merge order: loadedAttrs, then nodeAttrs,
//     then attributeWriters.
//   - Removes 3.12s CPU cost from sorting (benchmarked).
//   - Removes map allocation (pooled map no longer needed).
//   - Removes 4+ defer calls (explicit cleanup).
//
// Output order is stable because:
//  1. loadedAttrs order is fixed (parsed from SVG file in source order).
//  2. nodeAttrs order is fixed (from AST in source order).
//  3. attributeWriters order is fixed (from generated code order).
//  4. Merge rule: nodeAttrs and attributeWriters override loadedAttrs with the
//     same name.
func appendMergedSvgAttributesInline(buffer []byte, loadedAttrs, nodeAttrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter) []byte {
	buffer = appendMergedClassAttribute(buffer, loadedAttrs, nodeAttrs, attributeWriters)
	buffer = appendLoadedAttrsFiltered(buffer, loadedAttrs, nodeAttrs, attributeWriters)
	buffer = appendNodeAttrsFiltered(buffer, nodeAttrs)
	buffer = appendWriterAttrsFiltered(buffer, attributeWriters)
	return buffer
}

// appendMergedClassAttribute gathers class values from all attribute sources,
// removes duplicates, and appends the merged class attribute to the buffer.
//
// Takes buffer ([]byte) which is the buffer to append the class attribute to.
// Takes loadedAttrs ([]ast_domain.HTMLAttribute) which contains attributes from
// loaded sources.
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which contains attributes from
// the node itself.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which provides dynamic class
// values.
//
// Returns []byte which is the buffer with the merged class attribute added, or
// the same buffer if no classes were found.
func appendMergedClassAttribute(buffer []byte, loadedAttrs, nodeAttrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter) []byte {
	wrapper := getClassCollectionBufWrapper()
	wrapper.buffer = collectClassesToBuf(wrapper.buffer, loadedAttrs)
	wrapper.buffer = collectClassesToBufFiltered(wrapper.buffer, nodeAttrs)
	wrapper.buffer = collectClassesToBufFromWriters(wrapper.buffer, attributeWriters)

	if len(wrapper.buffer) > 0 {
		buffer = append(buffer, ` class="`...)
		buffer = appendDeduplicatedClassesToBufFromBytes(buffer, wrapper.buffer)
		buffer = append(buffer, '"')
	}
	putClassCollectionBufWrapper(wrapper)
	return buffer
}

// appendLoadedAttrsFiltered appends loaded SVG attributes to a buffer, but
// skips any that are replaced by node attributes or attribute writers.
//
// Takes buffer ([]byte) which is the buffer to append attributes to.
// Takes loadedAttrs ([]ast_domain.HTMLAttribute) which contains the loaded SVG
// attributes to filter and append.
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which contains node attributes
// that may replace loaded attributes.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which contains writers that
// may replace loaded attributes.
//
// Returns []byte which is the buffer with filtered attributes appended.
func appendLoadedAttrsFiltered(buffer []byte, loadedAttrs, nodeAttrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter) []byte {
	for i := range loadedAttrs {
		if shouldSkipLoadedAttr(&loadedAttrs[i], nodeAttrs, attributeWriters) {
			continue
		}
		buffer = appendSingleAttribute(buffer, &loadedAttrs[i])
	}
	return buffer
}

// shouldSkipLoadedAttr checks whether a loaded attribute should be skipped.
// It returns true when the attribute is a class attribute, or when the user
// has already set it, or when a direct writer will set it.
//
// Takes attr (*ast_domain.HTMLAttribute) which is the loaded attribute to
// check.
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which are the attributes the
// user has set on the node.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which are writers that may
// set attributes directly.
//
// Returns bool which is true if the attribute should be skipped.
func shouldSkipLoadedAttr(attr *ast_domain.HTMLAttribute, nodeAttrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter) bool {
	if attr.Name == attributeClass {
		return true
	}
	if hasSvgUserAttrByName(nodeAttrs, attr.Name) {
		return true
	}
	return hasSvgUserAttrWriterByName(attributeWriters, attr.Name)
}

// appendNodeAttrsFiltered appends HTML attributes to a buffer, skipping class,
// source, and piko:svg attributes.
//
// Takes buffer ([]byte) which is the buffer to append attributes to.
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which contains the attributes
// to filter and append.
//
// Returns []byte which is the buffer with filtered attributes added.
func appendNodeAttrsFiltered(buffer []byte, nodeAttrs []ast_domain.HTMLAttribute) []byte {
	for i := range nodeAttrs {
		if isSvgSkippedAttr(nodeAttrs[i].Name) {
			continue
		}
		buffer = appendSingleAttribute(buffer, &nodeAttrs[i])
	}
	return buffer
}

// appendWriterAttrsFiltered appends dynamic attribute writers to a buffer,
// skipping class, source, and piko:svg attributes.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which provides the attribute
// writers to filter and append.
//
// Returns []byte which is the buffer with the filtered attributes appended.
func appendWriterAttrsFiltered(buffer []byte, attributeWriters []*ast_domain.DirectWriter) []byte {
	for _, dw := range attributeWriters {
		if dw == nil || isSvgSkippedAttr(dw.Name) {
			continue
		}
		buffer = appendWriterAttribute(buffer, dw)
	}
	return buffer
}

// isSvgSkippedAttr reports whether the attribute name should be skipped during
// SVG attribute merging. Skipped attributes are class, source, and piko:svg.
//
// Uses switch instead of map for faster lookup. Parser lowercases all
// attribute names, so direct comparison is safe.
//
// Takes name (string) which is the attribute name to check.
//
// Returns bool which is true if the attribute should be skipped.
func isSvgSkippedAttr(name string) bool {
	switch name {
	case attributeClass, attributeSrc, tagPikoSvg:
		return true
	default:
		return false
	}
}

// hasSvgUserAttrWriterByName checks if a writer with the given name exists in
// the list, skipping reserved names during the search. Parser lowercases
// attribute names, so direct comparison is used.
//
// Takes attributeWriters ([]*ast_domain.DirectWriter) which is the list of writers
// to search.
// Takes name (string) which is the attribute name to find.
//
// Returns bool which is true if a matching writer exists.
func hasSvgUserAttrWriterByName(attributeWriters []*ast_domain.DirectWriter, name string) bool {
	for _, dw := range attributeWriters {
		if dw == nil {
			continue
		}
		if dw.Name == attributeSrc || dw.Name == tagPikoSvg {
			continue
		}
		if dw.Name == name {
			return true
		}
	}
	return false
}

// writerValueLen returns the byte length of a DirectWriter's value without
// allocating memory. Uses fast paths for single string or bytes parts.
//
// Takes dw (*ast_domain.DirectWriter) which is the writer to measure.
//
// Returns int which is the byte length of the rendered value.
func writerValueLen(dw *ast_domain.DirectWriter) int {
	if s, ok := dw.SingleStringValue(); ok {
		return len(s)
	}
	if b, ok := dw.SingleBytesValue(); ok {
		return len(b)
	}
	return dw.RenderedLen()
}

// appendWriterAttribute appends a dynamic attribute writer to the buffer.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes dw (*ast_domain.DirectWriter) which is the attribute writer to add.
//
// Returns []byte which is the buffer with the attribute appended.
func appendWriterAttribute(buffer []byte, dw *ast_domain.DirectWriter) []byte {
	buffer = append(buffer, ' ')
	buffer = append(buffer, dw.Name...)
	buffer = append(buffer, `="`...)
	if s, ok := dw.SingleStringValue(); ok {
		buffer = append(buffer, s...)
	} else {
		buffer = dw.WriteTo(buffer)
	}
	buffer = append(buffer, '"')
	return buffer
}

// collectClassesToBuf adds class attribute values from HTML attributes to a
// byte buffer without extra memory use.
//
// Takes buffer ([]byte) which is the target buffer.
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the HTML attributes
// to scan for class values.
//
// Returns []byte which is the buffer with class values added.
func collectClassesToBuf(buffer []byte, attrs []ast_domain.HTMLAttribute) []byte {
	for i := range attrs {
		if attrs[i].Name == attributeClass {
			if len(buffer) > 0 {
				buffer = append(buffer, ' ')
			}
			buffer = append(buffer, attrs[i].Value...)
		}
	}
	return buffer
}

// collectClassesToBufFiltered appends class attribute values to a buffer,
// skipping source and piko:svg attributes.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// filter.
//
// Returns []byte which is the buffer with class values appended.
func collectClassesToBufFiltered(buffer []byte, attrs []ast_domain.HTMLAttribute) []byte {
	for i := range attrs {
		name := attrs[i].Name
		if name == attributeSrc || name == tagPikoSvg {
			continue
		}
		if name == attributeClass {
			if len(buffer) > 0 {
				buffer = append(buffer, ' ')
			}
			buffer = append(buffer, attrs[i].Value...)
		}
	}
	return buffer
}

// collectClassesToBufFromWriters gathers class attribute values from dynamic
// writers and adds them to a byte buffer.
//
// Takes buffer ([]byte) which is the buffer to write to.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which holds the dynamic
// attributes to check for class values.
//
// Returns []byte which is the buffer with any class values added.
func collectClassesToBufFromWriters(buffer []byte, attributeWriters []*ast_domain.DirectWriter) []byte {
	for _, dw := range attributeWriters {
		if dw == nil {
			continue
		}
		if dw.Name == attributeSrc || dw.Name == tagPikoSvg {
			continue
		}
		if dw.Name == attributeClass {
			if len(buffer) > 0 {
				buffer = append(buffer, ' ')
			}
			if s, ok := dw.SingleStringValue(); ok {
				buffer = append(buffer, s...)
			} else if b, ok := dw.SingleBytesValue(); ok {
				buffer = append(buffer, b...)
			} else {
				buffer = dw.WriteTo(buffer)
			}
		}
	}
	return buffer
}

// appendDeduplicatedClassesToBufFromBytes appends unique CSS classes from a
// byte slice to the output buffer. This version takes bytes directly and does
// not allocate memory.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes classBytes ([]byte) which contains CSS class names separated by spaces.
//
// Returns []byte which is the buffer with unique classes appended.
func appendDeduplicatedClassesToBufFromBytes(buffer []byte, classBytes []byte) []byte {
	if len(classBytes) == 0 {
		return buffer
	}

	classString := mem.String(classBytes)

	analysis := analyseClassString(classString)

	if !analysis.hasSeparator {
		return append(buffer, classBytes...)
	}

	if len(classString) <= maxFastPathClassStringLen && analysis.estimatedCapacity <= 2 && analysis.onlySpaces {
		if result, ok := tryFastPathTwoClasses(classString, analysis.firstSpaceIndex); ok {
			return append(buffer, result...)
		}
	}

	classSet := getClassSet()

	tokeniser := newStringTokeniser(classString)

	needsSpace := false
	for tokeniser.Next() {
		class := tokeniser.Token()
		if _, exists := classSet[class]; !exists {
			classSet[class] = struct{}{}
			if needsSpace {
				buffer = append(buffer, ' ')
			}
			buffer = append(buffer, class...)
			needsSpace = true
		}
	}

	putClassSet(classSet)
	return buffer
}

// hasSvgUserAttrByName checks if an attribute with the given name exists.
// Parser lowercases attribute names, so direct comparison is used.
//
// For small slices (typically 3-8 attributes), linear search is faster than
// map lookup. The check skips source and piko:svg attributes.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which is the list of attributes to
// search.
// Takes name (string) which is the attribute name to find.
//
// Returns bool which is true if a matching attribute exists.
func hasSvgUserAttrByName(attrs []ast_domain.HTMLAttribute, name string) bool {
	for i := range attrs {
		attributeName := attrs[i].Name
		if attributeName == attributeSrc || attributeName == tagPikoSvg {
			continue
		}
		if attributeName == name {
			return true
		}
	}
	return false
}

// appendSingleAttribute adds a single HTML attribute to a byte buffer.
//
// Takes buffer ([]byte) which is the buffer to write to.
// Takes attr (*ast_domain.HTMLAttribute) which is the attribute to add.
//
// Returns []byte which is the buffer with the attribute added.
func appendSingleAttribute(buffer []byte, attr *ast_domain.HTMLAttribute) []byte {
	buffer = append(buffer, ' ')
	buffer = append(buffer, attr.Name...)
	buffer = append(buffer, `="`...)
	buffer = append(buffer, attr.Value...)
	buffer = append(buffer, '"')
	return buffer
}

// getSvgSrcOnly extracts the source attribute value without allocating a slice,
// providing a fast path for cache key computation.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which holds static HTML attributes.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which provides
// :source bindings.
//
// Returns string which is the source attribute value, or empty if not found.
//
// Defers userAttrs extraction until cache miss for optimisation.
func getSvgSrcOnly(attrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter) string {
	for i := range attrs {
		if attrs[i].Name == attributeSrc {
			return attrs[i].Value
		}
	}
	for _, dw := range attributeWriters {
		if dw != nil && dw.Name == attributeSrc {
			if s, ok := dw.SingleStringValue(); ok {
				return s
			}
			return dw.String()
		}
	}
	return ""
}

// extractUserAttrsOnly filters HTML attributes to return only user-defined
// attributes, excluding the source attribute and piko:svg tag name.
//
// Called only on cache miss or error paths. Counts matching attributes first
// to set the correct slice capacity.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which is the full list of
// attributes to filter.
//
// Returns []ast_domain.HTMLAttribute which contains only the user-defined
// attributes.
func extractUserAttrsOnly(attrs []ast_domain.HTMLAttribute) []ast_domain.HTMLAttribute {
	userAttrCount := 0
	for i := range attrs {
		if isUserAttr(attrs[i].Name) {
			userAttrCount++
		}
	}

	userAttrs := make([]ast_domain.HTMLAttribute, 0, userAttrCount)

	for i := range attrs {
		if isUserAttr(attrs[i].Name) {
			userAttrs = append(userAttrs, attrs[i])
		}
	}
	return userAttrs
}

// hashUserAttrsDirect computes a hash of user attributes directly from the
// original slice, without extracting them first. It skips "src" and "piko:svg"
// attributes and includes dynamic attribute writers in the hash.
//
// This function must produce the same hash as hashUserAttrs for the same
// inputs. The ordering is based on the original attribute order (not sorted),
// which is stable because the AST always keeps source order.
//
// This avoids creating a new []HTMLAttribute slice for cache key work.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the HTML attributes
// to hash.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which provides dynamic
// attribute writers to include in the hash.
//
// Returns uint64 which is the computed hash value, or zero when no user
// attributes or writers are present.
func hashUserAttrsDirect(attrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter) uint64 {
	indices, n := collectUserAttrIndices(attrs)
	writerCount := countUserAttrWriters(attributeWriters)

	if n == 0 && writerCount == 0 {
		return 0
	}

	if writerCount > 0 {
		return hashAttrsWithWriters(attrs, indices[:min(n, maxFastPathAttrCount)], n, attributeWriters, writerCount)
	}

	if n <= maxFastPathAttrCount {
		sortIndicesByAttrName(attrs, indices[:n])
		return hashAttrsByIndices(attrs, indices[:n])
	}

	userAttrs := extractUserAttrsOnly(attrs)
	return hashUserAttrs(userAttrs)
}

// countUserAttrWriters counts the attribute writers that are user attributes,
// excluding source and piko:svg attributes.
//
// Takes attributeWriters ([]*ast_domain.DirectWriter) which contains the attribute
// writers to check.
//
// Returns int which is the number of user attribute writers found.
func countUserAttrWriters(attributeWriters []*ast_domain.DirectWriter) int {
	count := 0
	for _, dw := range attributeWriters {
		if dw != nil && isUserAttr(dw.Name) {
			count++
		}
	}
	return count
}

// hashAttrsWithWriters computes a hash from both static attributes and
// dynamic writers. Uses sorted order to give the same result each time.
//
// Uses a stack-based array for the common case of 16 or fewer combined
// attributes. Uses insertion sort to avoid the overhead from sort.Slice.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the static HTML
// attributes to hash.
// Takes indices ([]int) which specifies the sorted order of static attributes.
// Takes staticCount (int) which is the number of static attributes to include.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which contains the dynamic
// attribute writers to include in the hash.
// Takes writerCount (int) which is the pre-computed count of user attribute
// writers, avoiding a redundant recount.
//
// Returns uint64 which is the computed hash of all combined attributes.
func hashAttrsWithWriters(attrs []ast_domain.HTMLAttribute, indices []int, staticCount int, attributeWriters []*ast_domain.DirectWriter, writerCount int) uint64 {
	totalCount := staticCount + writerCount

	if totalCount <= maxCombinedAttrCount {
		return hashAttrsWithWritersFast(attrs, indices, staticCount, attributeWriters)
	}

	return hashAttrsWithWritersSlow(attrs, staticCount, attributeWriters)
}

// hashAttrsWithWritersFast handles the common case with stack-allocated array.
// Zero heap allocations for <=16 combined attributes.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the static
// attributes to hash.
// Takes indices ([]int) which maps positions of user attributes within
// attrs.
// Takes staticCount (int) which limits how many static attributes to
// include.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which provides dynamic
// attribute writers to include in the hash.
//
// Returns uint64 which is the computed hash of the combined attributes.
func hashAttrsWithWritersFast(attrs []ast_domain.HTMLAttribute, indices []int, staticCount int, attributeWriters []*ast_domain.DirectWriter) uint64 {
	var refs [maxCombinedAttrCount]attributeReference
	n := collectAttrRefs(&refs, attrs, indices, staticCount, attributeWriters)
	insertionSortAttrRefs(refs[:n])
	return hashSortedAttrRefs(refs[:n])
}

// collectAttrRefs fills the refs array with static attributes and dynamic
// writers.
//
// Takes refs (*[...]attributeReference) which is the array to fill with attribute
// references.
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the source
// attributes.
// Takes indices ([]int) which specifies which attributes to include.
// Takes staticCount (int) which limits how many static attributes to add.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which provides dynamic
// attribute writers.
//
// Returns int which is the number of attribute references added to refs.
func collectAttrRefs(refs *[maxCombinedAttrCount]attributeReference, attrs []ast_domain.HTMLAttribute, indices []int, staticCount int, attributeWriters []*ast_domain.DirectWriter) int {
	n := 0
	for i := 0; i < len(indices) && i < staticCount && n < maxCombinedAttrCount; i++ {
		index := indices[i] //nolint:gosec // index bounded upstream
		refs[n] = attributeReference{name: attrs[index].Name, value: attrs[index].Value}
		n++
	}
	for _, dw := range attributeWriters {
		if dw == nil || !isUserAttr(dw.Name) || n >= maxCombinedAttrCount {
			continue
		}
		refs[n] = attributeReference{name: dw.Name, writer: dw}
		n++
	}
	return n
}

// insertionSortAttrRefs sorts attribute refs by name using insertion sort.
// This avoids reflection and memory allocation.
//
// Takes refs ([]attributeReference) which is the slice to sort in place.
func insertionSortAttrRefs(refs []attributeReference) {
	for i := 1; i < len(refs); i++ {
		key := refs[i]
		j := i - 1
		for j >= 0 && refs[j].name > key.name {
			refs[j+1] = refs[j]
			j--
		}
		refs[j+1] = key
	}
}

// hashSortedAttrRefs computes a hash from sorted attribute references.
//
// Takes refs ([]attributeReference) which contains the sorted
// attribute references to hash.
//
// Returns uint64 which is the combined hash of all attribute references.
func hashSortedAttrRefs(refs []attributeReference) uint64 {
	h := getHasher()
	for i := range refs {
		hashSingleAttrRef(h, &refs[i])
	}
	sum := h.Sum64()
	putHasher(h)
	return sum
}

// hashSingleAttrRef writes a single attribute reference to the given hasher.
//
// Takes h (hash.Hash64) which receives the hashed attribute data.
// Takes ref (*attributeReference) which is the attribute reference to hash.
func hashSingleAttrRef(h hash.Hash64, ref *attributeReference) {
	_, _ = h.Write(mem.Bytes(ref.name))
	_, _ = h.Write(nullSeparator)
	if ref.writer != nil {
		hashWriterValue(h, ref.writer)
	} else {
		_, _ = h.Write(mem.Bytes(ref.value))
	}
	_, _ = h.Write(nullSeparator)
}

// hashWriterValue writes a DirectWriter's value to the hash using fast paths.
//
// Takes h (hash.Hash64) which receives the hashed bytes.
// Takes writer (*ast_domain.DirectWriter) which provides the value to hash.
func hashWriterValue(h hash.Hash64, writer *ast_domain.DirectWriter) {
	if s, ok := writer.SingleStringValue(); ok {
		_, _ = h.Write(mem.Bytes(s))
		return
	}
	hashWriterParts(h, writer)
}

// hashWriterParts writes DirectWriter parts to the hasher without creating a
// string. This avoids memory allocation for common part types.
//
// Takes h (hash.Hash64) which receives the hash data.
// Takes dw (*ast_domain.DirectWriter) which provides the parts to hash.
func hashWriterParts(h hash.Hash64, dw *ast_domain.DirectWriter) {
	for i := range dw.Len() {
		part := dw.Part(i)
		if part == nil {
			continue
		}
		switch part.Type {
		case ast_domain.WriterPartString:
			_, _ = h.Write(mem.Bytes(part.StringValue))
		case ast_domain.WriterPartBytes:
			if len(part.BytesValue) > 0 {
				_, _ = h.Write(part.BytesValue)
			}
		case ast_domain.WriterPartInt:
			var buffer [intConvBufferSize]byte
			s := strconv.AppendInt(buffer[:0], part.IntValue, intConvBase)
			_, _ = h.Write(s)
		case ast_domain.WriterPartFloat:
			var buffer [floatConvBufferSize]byte
			s := strconv.AppendFloat(buffer[:0], part.FloatValue, 'f', -1, floatConvBitSize)
			_, _ = h.Write(s)
		case ast_domain.WriterPartBool:
			if part.BoolValue {
				_, _ = h.Write([]byte("true"))
			} else {
				_, _ = h.Write([]byte("false"))
			}
		default:
		}
	}
}

// hashAttrsWithWritersSlow handles the rare case of >16 combined attributes.
// Falls back to heap allocation but still avoids sort.Slice reflection.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the static
// attributes to hash.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which provides dynamic
// attribute writers to include in the hash.
//
// Returns uint64 which is the computed hash of the combined attributes.
func hashAttrsWithWritersSlow(attrs []ast_domain.HTMLAttribute, _ int, attributeWriters []*ast_domain.DirectWriter) uint64 {
	userAttrs := extractUserAttrsOnly(attrs)

	for _, dw := range attributeWriters {
		if dw == nil || !isUserAttr(dw.Name) {
			continue
		}
		var value string
		if s, ok := dw.SingleStringValue(); ok {
			value = s
		} else {
			value = dw.String()
		}
		userAttrs = append(userAttrs, ast_domain.HTMLAttribute{Name: dw.Name, Value: value})
	}

	return hashUserAttrs(userAttrs)
}

// collectUserAttrIndices collects indices of user attributes,
// excluding source and piko:svg.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which is the attribute list
// to scan.
//
// Returns [maxFastPathAttrCount]int which is a fixed-size array holding the
// indices of user attributes found, filled up to the returned count.
// Returns int which is the total count of user attributes found.
func collectUserAttrIndices(attrs []ast_domain.HTMLAttribute) ([maxFastPathAttrCount]int, int) {
	var indices [maxFastPathAttrCount]int
	n := 0

	for i := range attrs {
		if isUserAttr(attrs[i].Name) {
			if n < maxFastPathAttrCount {
				indices[n] = i
			}
			n++
		}
	}

	return indices, n
}

// isUserAttr reports whether the given attribute name is a user attribute. User
// attributes are those not reserved by the system, such as source or piko:svg.
//
// Takes name (string) which is the attribute name to check.
//
// Returns bool which is true if the attribute is a user attribute.
func isUserAttr(name string) bool {
	_, notUser := svgNonUserAttrs[name]
	return !notUser
}

// sortIndicesByAttrName sorts the given indices slice by the attribute names
// they point to. Uses insertion sort, which works well for small slices.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the attribute data
// for name comparisons.
// Takes indices ([]int) which holds the positions to sort in place.
func sortIndicesByAttrName(attrs []ast_domain.HTMLAttribute, indices []int) {
	for j := 1; j < len(indices); j++ {
		key := indices[j] //nolint:gosec // loop bounded
		k := j - 1
		for k >= 0 && attrs[indices[k]].Name > attrs[key].Name { //nolint:gosec // G602: k >= 0 guarantees bounds
			indices[k+1] = indices[k]
			k--
		}
		indices[k+1] = key
	}
}

// hashAttrsByIndices computes a hash of attributes at the given indices.
// Uses mem.Bytes for zero-copy string to byte slice conversion and explicit
// cleanup instead of defer for hot path optimisation.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the attributes to
// hash.
// Takes indices ([]int) which specifies which attribute positions to include.
//
// Returns uint64 which is the computed hash value.
func hashAttrsByIndices(attrs []ast_domain.HTMLAttribute, indices []int) uint64 {
	h := getHasher()

	for _, index := range indices {
		_, _ = h.Write(mem.Bytes(attrs[index].Name))
		_, _ = h.Write(nullSeparator)
		_, _ = h.Write(mem.Bytes(attrs[index].Value))
		_, _ = h.Write(nullSeparator)
	}

	sum := h.Sum64()
	putHasher(h)
	return sum
}

// analyseClassString scans a class string in one pass to find separators and
// estimate capacity. This cuts CPU work by 32% for two-class cases.
//
// Takes classString (string) which contains the CSS class names to scan.
//
// Returns classAnalysis which holds separator positions and capacity estimates.
func analyseClassString(classString string) classAnalysis {
	analysis := classAnalysis{
		estimatedCapacity: 1,
		firstSpaceIndex:   -1,
		hasSeparator:      false,
		onlySpaces:        true,
	}

	for i := range len(classString) {
		c := classString[i]
		switch c {
		case ' ':
			analysis.hasSeparator = true
			analysis.estimatedCapacity++
			if analysis.firstSpaceIndex == -1 {
				analysis.firstSpaceIndex = i
			}
		case '\t', '\n', '\r':
			analysis.hasSeparator = true
			analysis.onlySpaces = false
			analysis.estimatedCapacity++
		}
	}

	return analysis
}

// tryFastPathTwoClasses handles the common case of exactly two classes.
//
// Takes classString (string) which contains the space-separated class names.
// Takes firstSpaceIndex (int) which is the position of the first space character.
//
// Returns string which is the result with duplicates removed, or empty if not
// used.
// Returns bool which is true if the fast path was applied.
func tryFastPathTwoClasses(classString string, firstSpaceIndex int) (string, bool) {
	if firstSpaceIndex <= 0 {
		return "", false
	}

	secondSpaceIndex := -1
	for i := firstSpaceIndex + 1; i < len(classString); i++ {
		if classString[i] == ' ' {
			secondSpaceIndex = i
			break
		}
	}

	if secondSpaceIndex != -1 {
		return "", false
	}

	class1 := classString[:firstSpaceIndex]
	class2 := classString[firstSpaceIndex+1:]
	if class1 == class2 {
		return class1, true
	}
	return classString, true
}

// appendDeduplicatedClassesToBuf appends unique CSS classes to a byte buffer.
//
// This is a zero-copy version for the hot path in
// appendMergedSvgAttributesInline. It uses explicit cleanup instead of defer
// to reduce CPU overhead.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes classString (string) which contains space-separated CSS class names.
//
// Returns []byte which is the buffer with unique classes appended.
func appendDeduplicatedClassesToBuf(buffer []byte, classString string) []byte {
	if classString == "" {
		return buffer
	}

	analysis := analyseClassString(classString)

	if !analysis.hasSeparator {
		return append(buffer, classString...)
	}

	if len(classString) <= maxFastPathClassStringLen && analysis.estimatedCapacity <= 2 && analysis.onlySpaces {
		if result, ok := tryFastPathTwoClasses(classString, analysis.firstSpaceIndex); ok {
			return append(buffer, result...)
		}
	}

	classSet := getClassSet()

	tokeniser := newStringTokeniser(classString)

	needsSpace := false
	for tokeniser.Next() {
		class := tokeniser.Token()
		if _, exists := classSet[class]; !exists {
			classSet[class] = struct{}{}
			if needsSpace {
				buffer = append(buffer, ' ')
			}
			buffer = append(buffer, class...)
			needsSpace = true
		}
	}

	putClassSet(classSet)
	return buffer
}

// getClassSet retrieves a reusable map from the pool for collecting class
// names.
//
// Returns map[string]struct{} which is an empty map for storing class names.
func getClassSet() map[string]struct{} {
	if m, ok := classSetPool.Get().(map[string]struct{}); ok {
		return m
	}
	_, l := logger_domain.From(context.Background(), log)
	l.Error("classSetPool returned unexpected type, allocating new instance")
	return make(map[string]struct{})
}

// putClassSet returns a class set map to the pool after clearing it.
//
// Takes m (map[string]struct{}) which is the class set to return.
func putClassSet(m map[string]struct{}) {
	clear(m)
	classSetPool.Put(m)
}

// hashUserAttrs creates a stable hash of user attributes for use in cache keys.
// Attributes are sorted by name so that the same set of attributes always
// produces the same hash, regardless of their original order.
//
// For small attribute lists (8 or fewer), this uses manual insertion sort
// to avoid heap allocation.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// hash.
//
// Returns uint64 which is the computed hash value, or zero if attrs is empty.
func hashUserAttrs(attrs []ast_domain.HTMLAttribute) uint64 {
	if len(attrs) == 0 {
		return 0
	}

	if len(attrs) <= maxFastPathAttrCount {
		return hashUserAttrsSmall(attrs)
	}

	return hashUserAttrsLarge(attrs)
}

// hashUserAttrsSmall hashes small attribute lists of eight or fewer items.
// Uses a stack-based array and insertion sort to avoid heap allocation.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// hash.
//
// Returns uint64 which is the computed hash of the sorted attributes.
func hashUserAttrsSmall(attrs []ast_domain.HTMLAttribute) uint64 {
	var sorted [maxFastPathAttrCount]ast_domain.HTMLAttribute
	n := copy(sorted[:], attrs)

	sortSlice := sorted[:n]
	sortAttrsInPlace(sortSlice)

	return hashAttrSlice(sortSlice)
}

// hashUserAttrsLarge computes a hash for large attribute lists with more than
// eight items.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// hash.
//
// Returns uint64 which is the hash value after sorting the attributes by name.
func hashUserAttrsLarge(attrs []ast_domain.HTMLAttribute) uint64 {
	sortedSlice := make([]ast_domain.HTMLAttribute, len(attrs))
	copy(sortedSlice, attrs)
	slices.SortFunc(sortedSlice, func(a, b ast_domain.HTMLAttribute) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return hashAttrSlice(sortedSlice)
}

// sortAttrsInPlace sorts attributes by name using insertion sort.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which is the slice to sort in place.
func sortAttrsInPlace(attrs []ast_domain.HTMLAttribute) {
	for i := 1; i < len(attrs); i++ {
		key := attrs[i] //nolint:gosec // loop bounded
		j := i - 1
		for j >= 0 && attrs[j].Name > key.Name { //nolint:gosec // G602: j >= 0 guarantees bounds
			attrs[j+1] = attrs[j]
			j--
		}
		attrs[j+1] = key //nolint:gosec // j+1 in range [0, i]
	}
}

// hashAttrSlice computes a hash value for a slice of HTML attributes.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// hash.
//
// Returns uint64 which is the computed hash value.
//
// Uses explicit cleanup instead of defer to improve performance on this hot
// path.
func hashAttrSlice(attrs []ast_domain.HTMLAttribute) uint64 {
	h := getHasher()

	for i := range attrs {
		_, _ = h.Write(mem.Bytes(attrs[i].Name))
		_, _ = h.Write(nullSeparator)
		_, _ = h.Write(mem.Bytes(attrs[i].Value))
		_, _ = h.Write(nullSeparator)
	}

	sum := h.Sum64()
	putHasher(h)
	return sum
}

// getHasher gets a Hash64 from the shared pool for hashing.
//
// Returns hash.Hash64 which is ready for use.
func getHasher() hash.Hash64 {
	if h, ok := hashPool.Get().(hash.Hash64); ok {
		return h
	}
	_, l := logger_domain.From(context.Background(), log)
	l.Error("hashPool returned unexpected type, allocating new instance")
	return fnv.New64a()
}

// putHasher returns a hasher to the pool after resetting it.
//
// Takes h (hash.Hash64) which is the hasher to return to the pool.
func putHasher(h hash.Hash64) {
	h.Reset()
	hashPool.Put(h)
}

// newStringTokeniser creates a tokeniser for the given string.
//
// Takes s (string) which is the input to split into tokens.
//
// Returns stringTokeniser which is ready to iterate over tokens.
func newStringTokeniser(s string) stringTokeniser {
	return stringTokeniser{
		input:    s,
		current:  "",
		position: 0,
	}
}

// getBuilder retrieves a reusable strings.Builder from the pool.
//
// Returns *strings.Builder which is ready for use.
func getBuilder() *strings.Builder {
	if b, ok := stringBuilderPool.Get().(*strings.Builder); ok {
		return b
	}
	_, l := logger_domain.From(context.Background(), log)
	l.Error("stringBuilderPool returned unexpected type, allocating new instance")
	return &strings.Builder{}
}

// putBuilder returns a strings.Builder to the pool after resetting it.
//
// Takes b (*strings.Builder) which is the builder to return.
func putBuilder(b *strings.Builder) {
	b.Reset()
	stringBuilderPool.Put(b)
}

// getSortedKeysBuffer retrieves a string slice buffer from the pool or creates
// a new one.
//
// Takes neededCap (int) which is the minimum capacity required.
//
// Returns *[]string which points to a reusable or newly created slice.
func getSortedKeysBuffer(neededCap int) *[]string {
	if neededCap <= standardSortedKeysSize {
		if p, ok := sortedKeysPool.Get().(*[]string); ok {
			return p
		}
		_, l := logger_domain.From(context.Background(), log)
		l.Error("sortedKeysPool returned unexpected type, allocating new instance")
		return new(make([]string, 0, standardSortedKeysSize))
	}
	return new(make([]string, neededCap))
}

// putSortedKeysBuffer returns a sorted keys buffer to the pool.
//
// Takes s (*[]string) which is the buffer to return.
func putSortedKeysBuffer(s *[]string) {
	if cap(*s) == standardSortedKeysSize {
		for i := range standardSortedKeysSize {
			(*s)[i] = ""
		}
		sortedKeysPool.Put(s)
	}
}

// sortAttributeKeys sorts the first count elements of sortedKeys in place.
//
// Takes sortedKeys (*[]string) which is the slice to sort.
// Takes count (int) which specifies how many elements to sort from the start.
//
// Uses a manual sort for three or fewer elements, insertion sort for four to
// five elements, and the standard library sort for larger sets.
func sortAttributeKeys(sortedKeys *[]string, count int) {
	if count <= sortThresholdManual {
		sortManual(sortedKeys, count)
		return
	}
	if count <= sortThresholdInsertion {
		sortInsertion(sortedKeys, count)
		return
	}
	slices.Sort((*sortedKeys)[:count])
}

// sortManual sorts two or three elements using the fewest swaps.
//
// Takes sortedKeys (*[]string) which is the slice to sort in place.
// Takes count (int) which specifies how many elements to sort.
func sortManual(sortedKeys *[]string, count int) {
	switch count {
	case sortThresholdTwo:
		if (*sortedKeys)[0] > (*sortedKeys)[1] {
			(*sortedKeys)[0], (*sortedKeys)[1] = (*sortedKeys)[1], (*sortedKeys)[0]
		}
	case sortThresholdThree:
		if (*sortedKeys)[0] > (*sortedKeys)[1] {
			(*sortedKeys)[0], (*sortedKeys)[1] = (*sortedKeys)[1], (*sortedKeys)[0]
		}
		if (*sortedKeys)[1] > (*sortedKeys)[2] {
			(*sortedKeys)[1], (*sortedKeys)[2] = (*sortedKeys)[2], (*sortedKeys)[1]
		}
		if (*sortedKeys)[0] > (*sortedKeys)[1] {
			(*sortedKeys)[0], (*sortedKeys)[1] = (*sortedKeys)[1], (*sortedKeys)[0]
		}
	}
}

// sortInsertion sorts a small slice of strings using insertion sort.
// This is 18.5% faster than sort.Strings for 4-5 elements, based on benchmarks.
//
// Takes sortedKeys (*[]string) which is the slice to sort in place.
// Takes count (int) which is the number of elements to sort.
func sortInsertion(sortedKeys *[]string, count int) {
	keys := (*sortedKeys)[:count]
	for index := 1; index < count; index++ {
		key := keys[index]
		j := index - 1
		for j >= 0 && keys[j] > key {
			keys[j+1] = keys[j]
			j--
		}
		keys[j+1] = key
	}
}
