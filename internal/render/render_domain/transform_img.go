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
	"strconv"
	"strings"
	"sync"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// tagPikoImg is the tag name for the piko:img custom element.
	tagPikoImg = "piko:img"

	// sortBubbleThreshold is the largest slice size to sort with direct swaps.
	sortBubbleThreshold = 3

	// profileKeysInitialCapacity is the initial capacity for the profile keys
	// slice. 16 elements is a reasonable default for responsive image variants.
	profileKeysInitialCapacity = 16

	// baseDecimal is the base value for converting integers to decimal strings.
	baseDecimal = 10

	// attributeAlt is the HTML alt attribute name for image elements.
	attributeAlt = "alt"

	// attributeWidth is the HTML width attribute name.
	attributeWidth = "width"

	// attributeHeight is the HTML height attribute name for image dimensions.
	attributeHeight = "height"

	// defaultDensityCapacity is the initial slice capacity for image density variants.
	defaultDensityCapacity = 4

	// defaultWidthCapacity is the initial slice capacity for image width variants.
	defaultWidthCapacity = 8
)

// srcsetCacheKey is a composite map key for the srcset cache.
// Using a struct key avoids string joining when making cache keys.
type srcsetCacheKey struct {
	// artefactID is the unique identifier for the artefact.
	artefactID string

	// profileHash is a hash of the profile settings used for cache lookup.
	profileHash uint64
}

var (
	// sortedProfileKeysPool reuses string slices to reduce allocation pressure
	// during image profile key sorting.
	sortedProfileKeysPool = sync.Pool{
		New: func() any {
			return new(make([]string, 0, 16))
		},
	}

	// pikoImgAttrsPool pools pikoImgAttrs structs to avoid allocation per piko:img
	// element.
	pikoImgAttrsPool = sync.Pool{
		New: func() any { return new(pikoImgAttrs) },
	}

	// assetProfilePool reuses assetProfile instances to reduce allocation pressure
	// during responsive image transformation.
	assetProfilePool = sync.Pool{
		New: func() any {
			return &assetProfile{
				Densities: make([]string, 0, 4),
				Formats:   make([]string, 0, 4),
				Widths:    make([]int, 0, 8),
			}
		},
	}
)

// pikoImgAttrs holds all attributes from a piko:img element.
// Single-pass extraction avoids looping over node attributes more than once.
type pikoImgAttrs struct {
	// cmsMediaSource holds a CMS media object found through dynamic binding;
	// nil means no dynamic media was found.
	cmsMediaSource *cmsMediaWrapper

	// src is the image source URL or path; empty triggers a warning.
	src string

	// sizes is the HTML sizes attribute for responsive images.
	sizes string

	// densities is a comma-separated list of pixel density values (e.g. "1x,2x").
	densities string

	// formats is a comma-separated list of image output formats; defaults to webp.
	formats string

	// widths is a comma-separated list of image widths to generate.
	widths string

	// variant specifies the preferred CMS media variant name, such as "thumb_200".
	variant string

	// cmsMedia indicates whether the source is a CMS media reference.
	cmsMedia bool
}

// hasProfile returns true if any profile-related attributes were set.
//
// Returns bool which is true when sizes, densities, formats, or widths is set.
func (a *pikoImgAttrs) hasProfile() bool {
	return a.sizes != "" || a.densities != "" || a.formats != "" || a.widths != ""
}

// toAssetProfile converts the extracted attributes to an assetProfile.
//
// Returns *assetProfile which holds the parsed profile settings, or nil if no
// profile attributes were found.
func (a *pikoImgAttrs) toAssetProfile() *assetProfile {
	if !a.hasProfile() {
		return nil
	}

	profile := getAssetProfile()
	profile.Sizes = a.sizes
	profile.Densities = append(profile.Densities[:0], "1x")
	profile.Formats = append(profile.Formats[:0], "webp")

	if a.densities != "" {
		profile.Densities = appendCommaSeparated(profile.Densities[:0], a.densities)
	}
	if a.formats != "" {
		profile.Formats = appendCommaSeparated(profile.Formats[:0], a.formats)
	}
	if a.widths != "" {
		profile.Widths = appendIntList(profile.Widths[:0], a.widths)
	}

	return profile
}

// assetProfile represents the desired processing profile for a dynamic asset.
// Extracted from piko:img/piko:svg/piko:video attributes during rendering.
type assetProfile struct {
	// Sizes specifies the CSS sizes attribute for responsive images.
	Sizes string

	// Densities lists the pixel densities to generate (e.g. "1x", "2x", "3x").
	Densities []string

	// Formats lists the output image formats to generate (e.g. "webp", "avif").
	Formats []string

	// Widths lists the exact output widths to use instead of densities.
	Widths []int
}

// getPikoImgAttrs retrieves a pikoImgAttrs struct from the pool.
//
// Returns *pikoImgAttrs which is either a recycled instance from the pool or
// a new zero-value instance if the pool is empty.
func getPikoImgAttrs() *pikoImgAttrs {
	attrs, ok := pikoImgAttrsPool.Get().(*pikoImgAttrs)
	if !ok {
		return &pikoImgAttrs{}
	}
	return attrs
}

// putPikoImgAttrs returns a pikoImgAttrs struct to the pool after resetting it.
//
// Takes attrs (*pikoImgAttrs) which is the struct to reset and return to the
// pool.
func putPikoImgAttrs(attrs *pikoImgAttrs) {
	*attrs = pikoImgAttrs{}
	pikoImgAttrsPool.Put(attrs)
}

// extractPikoImgAttrs extracts all piko:img attributes from a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to extract
// attributes from.
//
// Returns pikoImgAttrs which contains the extracted image attributes.
func extractPikoImgAttrs(node *ast_domain.TemplateNode) pikoImgAttrs {
	var result pikoImgAttrs
	extractPikoImgAttrsInto(node, &result)
	return result
}

// extractPikoImgAttrsInto extracts all piko:img attributes into a pre-allocated
// struct. This avoids allocation when used with pooled structs.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to extract
// attributes from.
// Takes result (*pikoImgAttrs) which receives the extracted attributes.
func extractPikoImgAttrsInto(node *ast_domain.TemplateNode, result *pikoImgAttrs) {
	extractStaticPikoImgAttrs(node, result)
	if result.src == "" {
		result.src, result.cmsMediaSource = extractDynamicSrcOrMedia(node)
	}
}

// extractStaticPikoImgAttrs fills result with values from static HTML
// attributes on the node.
//
// Takes node (*ast_domain.TemplateNode) which contains the attributes to read.
// Takes result (*pikoImgAttrs) which receives the extracted values.
func extractStaticPikoImgAttrs(node *ast_domain.TemplateNode, result *pikoImgAttrs) {
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		assignPikoImgAttr(attr.Name, attr.Value, result)
	}
}

// assignPikoImgAttr assigns a piko-img attribute value to the corresponding
// field in result based on the attribute name.
//
// The parser lowercases attribute names during parsing, so direct comparison
// is used. Uses switch instead of map for faster dispatch on this small set.
//
// Takes name (string) which is the attribute name to match.
// Takes value (string) which is the value to assign.
// Takes result (*pikoImgAttrs) which receives the assigned value.
func assignPikoImgAttr(name, value string, result *pikoImgAttrs) {
	switch name {
	case attributeSrc:
		result.src = value
	case "sizes":
		result.sizes = value
	case "densities":
		result.densities = value
	case "formats":
		result.formats = value
	case "widths":
		result.widths = value
	case "variant":
		result.variant = value
	case "cms-media":
		result.cmsMedia = true
	}
}

// extractDynamicSrc looks through attribute writers for a dynamic src binding.
//
// Takes node (*ast_domain.TemplateNode) which holds the attribute writers to
// search.
//
// Returns string which is the dynamic src value, or empty if not found.
func extractDynamicSrc(node *ast_domain.TemplateNode) string {
	src, _ := extractDynamicSrcOrMedia(node)
	return src
}

// extractDynamicSrcOrMedia searches attribute writers for a dynamic src
// binding. If the bound value has CMS media methods (MediaURL, etc.), it wraps
// it for variant extraction.
//
// Takes node (*ast_domain.TemplateNode) which contains the attribute writers
// to search.
//
// Returns string which is the dynamic src value, or empty if not found.
// Returns *cmsMediaWrapper which wraps CMS media, or nil if not CMS media.
func extractDynamicSrcOrMedia(node *ast_domain.TemplateNode) (string, *cmsMediaWrapper) {
	for _, dw := range node.AttributeWriters {
		if dw == nil || dw.Name != attributeSrc {
			continue
		}
		return extractSrcFromWriter(dw)
	}
	return "", nil
}

// extractSrcFromWriter extracts the src value from a DirectWriter.
//
// Takes dw (*ast_domain.DirectWriter) which contains the source value to
// extract.
//
// Returns string which is the extracted source URL or string value.
// Returns *cmsMediaWrapper which wraps CMS media data, or nil if the source
// is not CMS media.
func extractSrcFromWriter(dw *ast_domain.DirectWriter) (string, *cmsMediaWrapper) {
	if dw.Len() == 1 {
		if wrapper := tryExtractCMSMedia(dw.Part(0)); wrapper != nil {
			url := wrapper.MediaURL()
			return url, wrapper
		}
	}
	if s, ok := dw.SingleStringValue(); ok {
		return s, nil
	}
	return dw.String(), nil
}

// tryExtractCMSMedia attempts to extract a CMS media wrapper from a writer
// part.
//
// Takes part (*ast_domain.WriterPart) which is the writer part to extract from.
//
// Returns *cmsMediaWrapper which is the extracted media wrapper, or nil if the
// part is nil or not of type WriterPartAny.
func tryExtractCMSMedia(part *ast_domain.WriterPart) *cmsMediaWrapper {
	if part == nil || part.Type != ast_domain.WriterPartAny {
		return nil
	}
	return tryCMSMediaWrapper(part.AnyValue)
}

// renderPikoImg renders a <piko:img> component as an <img> tag using the
// direct-to-writer pattern.
//
// Takes ro (*RenderOrchestrator) which provides asset registration and
// element directive rendering.
// Takes node (*ast_domain.TemplateNode) which is the piko:img element to
// render.
// Takes qw (*qt.Writer) which is the quicktemplate writer for output.
// Takes rctx (*renderContext) which provides rendering state and diagnostics.
//
// Returns error when rendering fails.
func renderPikoImg(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext) error {
	ImgTransformCount.Add(rctx.originalCtx, 1)

	attrs := getPikoImgAttrs()
	defer putPikoImgAttrs(attrs)
	extractPikoImgAttrsInto(node, attrs)

	if attrs.src == "" {
		rctx.diagnostics.AddWarning("renderPikoImg", "piko:img missing src", nil)
		writeImgTagWithoutSrc(ro, node, qw, rctx)
		return nil
	}

	if attrs.cmsMediaSource != nil || attrs.cmsMedia {
		return renderCMSMediaImg(ro, node, qw, rctx, attrs)
	}

	buffer := rctx.getBuffer()
	*buffer = assetpath.AppendTransformed(*buffer, attrs.src, assetpath.DefaultServePath)
	transformedSrc := rctx.freezeToString(buffer)

	profile := attrs.toAssetProfile()
	if profile != nil {
		defer putAssetProfile(profile)
	}
	var artefact *registry_dto.ArtefactMeta
	if profile != nil {
		if rctx.registry == nil {
			rctx.diagnostics.AddWarning(
				"renderPikoImg",
				"No image provider configured; responsive image features disabled",
				map[string]string{"src": attrs.src},
			)
		} else {
			artefact = ro.registerDynamicAsset(rctx.originalCtx, attrs.src, profile, rctx)
		}
	}
	hasSrcset := artefact != nil && len(artefact.DesiredProfiles) > 0

	qw.N().Z(openBracket)
	qw.N().S("img")
	writeSrcAttr(qw, transformedSrc)

	if hasSrcset {
		writeSrcsetAttribute(qw, artefact, transformedSrc, rctx)
		writeSizesAttr(qw, attrs.sizes)
	}

	writePikoImgStaticAttrsFilteredWithExclusions(node, qw, hasSrcset)
	if hasSrcset {
		ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc, attributeSrcset)
	} else {
		ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc)
	}
	qw.N().Z(selfClose)

	return nil
}

// renderCMSMediaImg renders a piko:img with a CMS media source. Uses
// pre-generated variants from the CMS instead of registering new profiles.
//
// Takes ro (*RenderOrchestrator) which provides element directive rendering.
// Takes node (*ast_domain.TemplateNode) which is the piko:img element to
// render.
// Takes qw (*qt.Writer) which is the quicktemplate writer for output.
// Takes rctx (*renderContext) which provides rendering state and diagnostics.
// Takes attrs (*pikoImgAttrs) which contains the extracted attributes.
//
// Returns error when rendering fails.
func renderCMSMediaImg(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext, attrs *pikoImgAttrs) error {
	media := attrs.cmsMediaSource
	if media == nil {
		rctx.diagnostics.AddWarning(
			"renderPikoImg",
			"cms-media attribute set but source is not a CMS media object",
			map[string]string{"src": attrs.src},
		)
		return nil
	}

	srcURL := media.MediaURL()
	if attrs.variant != "" {
		variant := media.MediaVariant(attrs.variant)
		if variant != nil && variant.VariantURL() != "" && variant.IsReady() {
			srcURL = variant.VariantURL()
		}
	}

	qw.N().Z(openBracket)
	qw.N().S("img")
	writeSrcAttr(qw, srcURL)

	variants := media.MediaVariants()
	hasSrcset := len(variants) > 0 && attrs.widths != ""
	if hasSrcset {
		writeCMSMediaSrcset(qw, variants, attrs.widths, rctx)
		writeSizesAttr(qw, attrs.sizes)
	}

	writeAltFromMedia(node, qw, media)

	writeMediaDimensions(node, qw, media)

	writePikoImgStaticAttrsFilteredWithExclusions(node, qw, hasSrcset)
	if hasSrcset {
		ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc, attributeSrcset, attributeAlt, attributeWidth, attributeHeight)
	} else {
		ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc, attributeAlt, attributeWidth, attributeHeight)
	}
	qw.N().Z(selfClose)

	return nil
}

// writeCMSMediaSrcset generates and writes a srcset attribute from CMS
// variants.
//
// Takes qw (*qt.Writer) which receives the srcset attribute.
// Takes variants (map[string]*variantWrapper) which contains the available
// variants.
// Takes widthsAttr (string) which specifies which widths to include
// (comma-separated).
// Takes rctx (*renderContext) which provides buffer pooling.
func writeCMSMediaSrcset(qw *qt.Writer, variants map[string]*variantWrapper, widthsAttr string, rctx *renderContext) {
	requestedWidths := parseIntList(widthsAttr)
	if len(requestedWidths) == 0 {
		return
	}

	buffer := rctx.getBuffer()
	needComma := false

	for _, width := range requestedWidths {
		variantKey := "w" + strconv.Itoa(width)
		variant, found := variants[variantKey]
		if !found {
			variantKey = strconv.Itoa(width)
			variant, found = variants[variantKey]
		}
		if !found || variant == nil || !variant.IsReady() {
			continue
		}

		if needComma {
			*buffer = append(*buffer, ", "...)
		}
		needComma = true

		*buffer = append(*buffer, variant.VariantURL()...)
		*buffer = append(*buffer, ' ')
		*buffer = strconv.AppendInt(*buffer, int64(variant.VariantWidth()), baseDecimal)
		*buffer = append(*buffer, 'w')
	}

	if len(*buffer) > 0 {
		srcset := rctx.freezeToString(buffer)
		qw.N().Z(space)
		qw.N().S("srcset")
		qw.N().Z(equalsQuote)
		qw.N().S(srcset)
		qw.N().Z(quote)
	}
}

// writeAltFromMedia writes the alt attribute from CMS media if not already set.
//
// Takes node (*ast_domain.TemplateNode) which holds the element attributes.
// Takes qw (*qt.Writer) which receives the attribute output.
// Takes media (*cmsMediaWrapper) which provides the media alt text.
func writeAltFromMedia(node *ast_domain.TemplateNode, qw *qt.Writer, media *cmsMediaWrapper) {
	for i := range node.Attributes {
		if node.Attributes[i].Name == attributeAlt {
			return
		}
	}
	for _, dw := range node.AttributeWriters {
		if dw != nil && dw.Name == attributeAlt {
			return
		}
	}

	altText := media.MediaAltText()
	if altText != "" {
		qw.N().Z(space)
		qw.N().S(attributeAlt)
		qw.N().Z(equalsQuote)
		qw.E().S(altText)
		qw.N().Z(quote)
	}
}

// writeMediaDimensions writes width and height attributes from CMS media if not
// set.
//
// Takes node (*ast_domain.TemplateNode) which contains the element attributes.
// Takes qw (*qt.Writer) which receives the attribute output.
// Takes media (*cmsMediaWrapper) which provides the media dimensions.
func writeMediaDimensions(node *ast_domain.TemplateNode, qw *qt.Writer, media *cmsMediaWrapper) {
	hasWidth, hasHeight := checkDimensionAttributes(node)

	if !hasWidth {
		writeIntAttr(qw, attributeWidth, media.MediaWidth())
	}
	if !hasHeight {
		writeIntAttr(qw, attributeHeight, media.MediaHeight())
	}
}

// checkDimensionAttributes checks if width and height attributes are already
// set on a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to inspect for
// dimension attributes.
//
// Returns hasWidth (bool) which indicates whether a width attribute exists.
// Returns hasHeight (bool) which indicates whether a height attribute exists.
func checkDimensionAttributes(node *ast_domain.TemplateNode) (hasWidth, hasHeight bool) {
	for i := range node.Attributes {
		name := node.Attributes[i].Name
		if name == attributeWidth {
			hasWidth = true
		}
		if name == attributeHeight {
			hasHeight = true
		}
	}

	for _, dw := range node.AttributeWriters {
		if dw == nil {
			continue
		}
		if dw.Name == attributeWidth {
			hasWidth = true
		}
		if dw.Name == attributeHeight {
			hasHeight = true
		}
	}
	return hasWidth, hasHeight
}

// writeIntAttr writes an integer attribute if the value is positive.
//
// Takes qw (*qt.Writer) which receives the formatted output.
// Takes name (string) which specifies the attribute name.
// Takes value (int) which is the integer to write.
func writeIntAttr(qw *qt.Writer, name string, value int) {
	if value <= 0 {
		return
	}
	qw.N().Z(space)
	qw.N().S(name)
	qw.N().Z(equalsQuote)
	qw.N().DL(int64(value))
	qw.N().Z(quote)
}

// writeSrcAttr writes an src attribute to the output.
//
// Takes qw (*qt.Writer) which receives the formatted attribute.
// Takes src (string) which specifies the source URL value.
func writeSrcAttr(qw *qt.Writer, src string) {
	qw.N().Z(space)
	qw.N().S(attributeSrc)
	qw.N().Z(equalsQuote)
	qw.N().S(src)
	qw.N().Z(quote)
}

// writeSizesAttr writes the sizes attribute if the value is not empty.
//
// Takes qw (*qt.Writer) which receives the attribute output.
// Takes sizes (string) which is the value to write.
func writeSizesAttr(qw *qt.Writer, sizes string) {
	if sizes == "" {
		return
	}
	qw.N().Z(space)
	qw.N().S("sizes")
	qw.N().Z(equalsQuote)
	qw.N().S(sizes)
	qw.N().Z(quote)
}

// writeImgTagWithoutSrc writes an img tag when the src attribute is missing.
//
// Takes ro (*RenderOrchestrator) which provides the rendering setup.
// Takes node (*ast_domain.TemplateNode) which holds the template node data.
// Takes qw (*qt.Writer) which receives the output HTML.
// Takes rctx (*renderContext) which holds the current render state.
func writeImgTagWithoutSrc(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext) {
	qw.N().Z(openBracket)
	qw.N().S("img")
	writePikoImgStaticAttrsFilteredWithExclusions(node, qw, false)
	ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc)
	qw.N().Z(selfClose)
}

// writeSrcsetAttribute writes the srcset attribute to the output. It uses a
// cache to avoid building the same string more than once for each image.
//
// Takes qw (*qt.Writer) which receives the rendered attribute output.
// Takes artefact (*registry_dto.ArtefactMeta) which provides the image
// profiles.
// Takes baseURL (string) which is the base path for image URLs.
// Takes rctx (*renderContext) which provides the srcset cache and buffer pool.
func writeSrcsetAttribute(qw *qt.Writer, artefact *registry_dto.ArtefactMeta, baseURL string, rctx *renderContext) {
	profileHash := hashDesiredProfiles(artefact.DesiredProfiles)
	cacheKey := srcsetCacheKey{
		artefactID:  artefact.ID,
		profileHash: profileHash,
	}

	if cached, exists := rctx.srcsetCache[cacheKey]; exists {
		qw.N().Z(space)
		qw.N().S("srcset")
		qw.N().Z(equalsQuote)
		qw.N().S(cached)
		qw.N().Z(quote)
		return
	}

	buffer := rctx.getBuffer()
	*buffer = appendSrcset(*buffer, artefact.DesiredProfiles, baseURL)

	if len(*buffer) > 0 {
		srcset := rctx.freezeToString(buffer)
		rctx.srcsetCache[cacheKey] = srcset

		qw.N().Z(space)
		qw.N().S("srcset")
		qw.N().Z(equalsQuote)
		qw.N().S(srcset)
		qw.N().Z(quote)
	}
}

// hashDesiredProfiles computes a hash of the desired profiles for cache key.
// Uses FNV-1a for fast hashing.
//
// Takes profiles ([]registry_dto.NamedProfile) which specifies the profiles
// to hash.
//
// Returns uint64 which is the computed hash value, or zero if profiles is
// empty.
func hashDesiredProfiles(profiles []registry_dto.NamedProfile) uint64 {
	if len(profiles) == 0 {
		return 0
	}

	h := getHasher()

	keys := getSortedProfileKeys(profiles)

	for _, key := range *keys {
		_, _ = h.Write(mem.Bytes(key))
		_, _ = h.Write([]byte{0})
		for i := range profiles {
			if profiles[i].Name == key {
				if w, ok := profiles[i].Profile.ResultingTags.GetByName("width"); ok {
					_, _ = h.Write(mem.Bytes(w))
				}
				if d, ok := profiles[i].Profile.ResultingTags.GetByName("density"); ok {
					_, _ = h.Write(mem.Bytes(d))
				}
				break
			}
		}
		_, _ = h.Write([]byte{0})
	}

	putSortedProfileKeys(keys)
	sum := h.Sum64()
	putHasher(h)
	return sum
}

// getSortedProfileKeys returns a pooled slice of sorted profile keys.
//
// Takes profiles ([]registry_dto.NamedProfile) which contains the profiles to
// extract and sort keys from.
//
// Returns *[]string which is a pooled slice containing the sorted profile
// names. The caller must return this slice to the pool when finished.
func getSortedProfileKeys(profiles []registry_dto.NamedProfile) *[]string {
	keys, ok := sortedProfileKeysPool.Get().(*[]string)
	if !ok {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("sortedProfileKeysPool returned unexpected type, allocating new instance")
		keys = new(make([]string, 0, profileKeysInitialCapacity))
	}

	*keys = (*keys)[:0]
	for i := range profiles {
		*keys = append(*keys, profiles[i].Name)
	}

	sortProfileKeys(keys, len(*keys))

	return keys
}

// putSortedProfileKeys returns a slice to the pool after clearing it.
//
// Takes keys (*[]string) which is the slice to clear and return to the pool.
func putSortedProfileKeys(keys *[]string) {
	for i := range *keys {
		(*keys)[i] = ""
	}
	*keys = (*keys)[:0]
	sortedProfileKeysPool.Put(keys)
}

// sortProfileKeys sorts profile keys in place using a method suited to the
// slice size. It uses a simple swap for two items, a three-way swap for three
// items, and insertion sort for larger slices.
//
// Takes keys (*[]string) which is the slice of profile keys to sort in place.
// Takes count (int) which specifies how many items from the slice to sort.
func sortProfileKeys(keys *[]string, count int) {
	if count <= 1 {
		return
	}
	if count == 2 {
		if (*keys)[0] > (*keys)[1] {
			(*keys)[0], (*keys)[1] = (*keys)[1], (*keys)[0]
		}
		return
	}
	if count == sortBubbleThreshold {
		if (*keys)[0] > (*keys)[1] {
			(*keys)[0], (*keys)[1] = (*keys)[1], (*keys)[0]
		}
		if (*keys)[1] > (*keys)[2] {
			(*keys)[1], (*keys)[2] = (*keys)[2], (*keys)[1]
		}
		if (*keys)[0] > (*keys)[1] {
			(*keys)[0], (*keys)[1] = (*keys)[1], (*keys)[0]
		}
		return
	}
	slice := (*keys)[:count]
	for i := 1; i < count; i++ {
		key := slice[i]
		j := i - 1
		for j >= 0 && slice[j] > key {
			slice[j+1] = slice[j]
			j--
		}
		slice[j+1] = key
	}
}

// findProfileByKey looks for a profile with the given name in a list.
//
// Takes profiles ([]registry_dto.NamedProfile) which is the list to search.
// Takes key (string) which is the name to find.
//
// Returns *registry_dto.DesiredProfile which is the matching profile, or nil
// if not found.
func findProfileByKey(profiles []registry_dto.NamedProfile, key string) *registry_dto.DesiredProfile {
	for i := range profiles {
		if profiles[i].Name == key {
			return &profiles[i].Profile
		}
	}
	return nil
}

// getProfileDescriptor returns the descriptor value and suffix for a srcset
// entry.
//
// Takes profile (*registry_dto.DesiredProfile) which contains the image
// profile to get descriptor details from.
//
// Returns value (string) which is the width or density descriptor value.
// Returns suffix (byte) which is 'w' for width descriptors, or 0 for density.
// Returns ok (bool) which is false when no valid descriptor is found.
func getProfileDescriptor(profile *registry_dto.DesiredProfile) (value string, suffix byte, ok bool) {
	if width, hasWidth := profile.ResultingTags.GetByName("width"); hasWidth {
		return width, 'w', true
	}
	if density, hasDensity := profile.ResultingTags.GetByName("density"); hasDensity {
		return density, 0, true
	}
	return "", 0, false
}

// appendSrcsetEntry appends a single srcset entry to the buffer.
//
// Each entry has the base URL with a version query parameter, then the
// descriptor (such as "2x" or "100w").
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes baseURL (string) which is the image URL.
// Takes profileKey (string) which is added as the version query parameter.
// Takes descriptor (string) which is the srcset descriptor value.
// Takes suffix (byte) which is appended after the descriptor if not zero.
//
// Returns []byte which is the buffer with the entry appended.
func appendSrcsetEntry(buffer []byte, baseURL, profileKey, descriptor string, suffix byte) []byte {
	buffer = append(buffer, baseURL...)
	buffer = append(buffer, "?v="...)
	buffer = append(buffer, profileKey...)
	buffer = append(buffer, ' ')
	buffer = append(buffer, descriptor...)
	if suffix != 0 {
		buffer = append(buffer, suffix)
	}
	return buffer
}

// appendSrcset builds a srcset string directly into a buffer.
// Profiles are processed in sorted order to ensure consistent output.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes profiles ([]registry_dto.NamedProfile) which contains the image
// profiles to include in the srcset.
// Takes baseURL (string) which is the base URL for image paths.
//
// Returns []byte which is the buffer with the srcset string appended.
func appendSrcset(buffer []byte, profiles []registry_dto.NamedProfile, baseURL string) []byte {
	if len(profiles) == 0 {
		return buffer
	}

	keys := getSortedProfileKeys(profiles)
	defer putSortedProfileKeys(keys)

	needComma := false
	for _, profileKey := range *keys {
		profile := findProfileByKey(profiles, profileKey)
		if profile == nil {
			continue
		}

		descriptor, suffix, ok := getProfileDescriptor(profile)
		if !ok {
			continue
		}

		if needComma {
			buffer = append(buffer, ", "...)
		}
		needComma = true

		buffer = appendSrcsetEntry(buffer, baseURL, profileKey, descriptor, suffix)
	}

	return buffer
}

// isPikoImgSpecialAttr checks whether a name is a piko:img special attribute
// that should not appear in the output.
//
// Uses switch instead of map for faster lookup on this small key set. Parser
// lowercases all attribute names, so direct comparison is safe.
//
// Takes name (string) which is the attribute name to check.
//
// Returns bool which is true when the name matches a special attribute.
func isPikoImgSpecialAttr(name string) bool {
	switch name {
	case "profile", "densities", "sizes", "formats", "widths", "variant", "cms-media":
		return true
	default:
		return false
	}
}

// getAssetProfile retrieves an assetProfile from the pool with pre-allocated
// slice backing arrays.
//
// Returns *assetProfile which is ready for use.
func getAssetProfile() *assetProfile {
	if p, ok := assetProfilePool.Get().(*assetProfile); ok {
		return p
	}
	return &assetProfile{
		Densities: make([]string, 0, defaultDensityCapacity),
		Formats:   make([]string, 0, defaultDensityCapacity),
		Widths:    make([]int, 0, defaultWidthCapacity),
	}
}

// putAssetProfile returns an assetProfile to the pool after resetting its
// slices.
//
// Takes p (*assetProfile) which is the profile to return to the pool.
func putAssetProfile(p *assetProfile) {
	p.Sizes = ""
	p.Densities = p.Densities[:0]
	p.Formats = p.Formats[:0]
	p.Widths = p.Widths[:0]
	assetProfilePool.Put(p)
}

// parseCommaSeparated splits a string by commas or spaces. It handles both
// "a,b,c" and "a b c" formats, which are common in HTML attributes.
//
// Takes value (string) which is the input to split.
//
// Returns []string which contains the split parts with empty strings removed.
func parseCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' '
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// appendCommaSeparated splits a string by commas or spaces and appends
// non-empty parts to dst, reusing its backing array.
//
// Takes dst ([]string) which is the slice to append to.
// Takes value (string) which contains the comma or space separated values.
//
// Returns []string which is dst with the parsed parts appended.
func appendCommaSeparated(dst []string, value string) []string {
	for value != "" {
		i := 0
		for i < len(value) && value[i] != ',' && value[i] != ' ' {
			i++
		}
		if i > 0 {
			dst = append(dst, value[:i])
		}
		value = value[i:]
		for len(value) > 0 && (value[0] == ',' || value[0] == ' ') {
			value = value[1:]
		}
	}
	return dst
}

// parseIntList parses a comma-separated list of integers.
//
// Takes value (string) which contains the comma-separated integers.
//
// Returns []int which contains the parsed integers. Values that are not valid
// integers are skipped.
func parseIntList(value string) []int {
	parts := parseCommaSeparated(value)
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		if value, err := strconv.Atoi(part); err == nil {
			result = append(result, value)
		}
	}
	return result
}

// appendIntList parses a comma-separated list of integers and appends them
// to dst, reusing its backing array.
//
// Takes dst ([]int) which is the slice to append to.
// Takes value (string) which contains the comma-separated integers.
//
// Returns []int which is dst with the parsed integers appended. Values that
// are not valid integers are skipped.
func appendIntList(dst []int, value string) []int {
	for value != "" {
		i := 0
		for i < len(value) && value[i] != ',' && value[i] != ' ' {
			i++
		}
		if i > 0 {
			if parsedValue, err := strconv.Atoi(value[:i]); err == nil {
				dst = append(dst, parsedValue)
			}
		}
		value = value[i:]
		for len(value) > 0 && (value[0] == ',' || value[0] == ' ') {
			value = value[1:]
		}
	}
	return dst
}

// writePikoImgStaticAttrsFilteredWithExclusions writes static attributes to the
// output, skipping profile-related attributes, src, and optionally srcset.
//
// Takes node (*ast_domain.TemplateNode) which holds the attributes to write.
// Takes qw (*qt.Writer) which receives the output.
// Takes excludeSrcset (bool) which when true also skips srcset attributes.
func writePikoImgStaticAttrsFilteredWithExclusions(node *ast_domain.TemplateNode, qw *qt.Writer, excludeSrcset bool) {
	for i := range node.Attributes {
		attr := &node.Attributes[i]

		if attr.Name == attributeSrc || isPikoImgSpecialAttr(attr.Name) {
			continue
		}

		if excludeSrcset && attr.Name == attributeSrcset {
			continue
		}

		qw.N().Z(space)
		qw.N().S(attr.Name)
		qw.N().Z(equalsQuote)
		qw.N().S(attr.Value)
		qw.N().Z(quote)
	}
}
