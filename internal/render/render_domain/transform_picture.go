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
	"path"
	"strings"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// tagPikoPicture is the tag name for the piko:picture custom element.
	tagPikoPicture = "piko:picture"

	// elementPicture is the HTML picture element name.
	elementPicture = "picture"

	// widthParseBase is the base for parsing width values from profile tags.
	widthParseBase = 10

	// formatJPG is the JPEG format identifier for fallback source selection.
	formatJPG = "jpg"
)

// imgElementParams groups the parameters for writeImgElement.
type imgElementParams struct {
	// Artefact is the registered image artefact metadata, or nil when unregistered.
	Artefact *registry_dto.ArtefactMeta

	// Attrs holds the extracted piko:picture attributes from the source node.
	Attrs *pikoImgAttrs

	// TransformedSrc is the resolved image source URL after asset path transformation.
	TransformedSrc string

	// FallbackFormat is the image format used for the fallback img src attribute.
	FallbackFormat string

	// HasProfiles indicates whether image profiles are
	// registered for srcset generation.
	HasProfiles bool
}

// renderPikoPicture renders a <piko:picture> component as a <picture> element
// with per-format <source> elements and a fallback <img>.
//
// Takes ro (*RenderOrchestrator) which provides asset registration and
// element directive rendering.
// Takes node (*ast_domain.TemplateNode) which is the piko:picture element to
// render.
// Takes qw (*qt.Writer) which is the quicktemplate writer for output.
// Takes rctx (*renderContext) which provides rendering state and diagnostics.
//
// Returns error when rendering fails.
func renderPikoPicture(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext) error {
	PictureTransformCount.Add(rctx.originalCtx, 1)

	attrs := getPikoImgAttrs()
	defer putPikoImgAttrs(attrs)
	extractPikoImgAttrsInto(node, attrs)

	if attrs.src == "" {
		rctx.diagnostics.AddWarning("renderPikoPicture", "piko:picture missing src", nil)
		writePictureTagWithoutSrc(ro, node, qw, rctx)
		return nil
	}

	if attrs.cmsMediaSource != nil || attrs.cmsMedia {
		return renderCMSMediaPicture(ro, node, qw, rctx, attrs)
	}

	return renderStaticPicture(ro, node, qw, rctx, attrs)
}

// renderStaticPicture renders a piko:picture with a static asset source,
// handling format negotiation, profile registration, and source element
// generation.
//
// Takes ro (*RenderOrchestrator) which provides asset registration and
// element directive rendering.
// Takes node (*ast_domain.TemplateNode) which is the piko:picture element.
// Takes qw (*qt.Writer) which is the quicktemplate writer for output.
// Takes rctx (*renderContext) which provides rendering state.
// Takes attrs (*pikoImgAttrs) which contains the extracted attributes.
//
// Returns error when rendering fails.
func renderStaticPicture(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext, attrs *pikoImgAttrs) error {
	buffer := rctx.getBuffer()
	*buffer = assetpath.AppendTransformed(*buffer, attrs.src, assetpath.DefaultServePath)
	transformedSrc := rctx.freezeToString(buffer)

	formats := []string{"webp"}
	if attrs.formats != "" {
		formats = parseCommaSeparated(attrs.formats)
	}
	fallbackFormat := inferFallbackFormat(attrs.src)

	artefact := registerPictureProfile(ro, rctx, attrs, formats, fallbackFormat)
	hasProfiles := artefact != nil && len(artefact.DesiredProfiles) > 0

	writePictureOpen(qw)

	if hasProfiles {
		for _, format := range formats {
			writePictureSourceElement(qw, artefact, transformedSrc, format, attrs.sizes, rctx)
		}
	}

	writeImgElement(ro, node, qw, rctx, imgElementParams{
		Artefact:       artefact,
		TransformedSrc: transformedSrc,
		FallbackFormat: fallbackFormat,
		Attrs:          attrs,
		HasProfiles:    hasProfiles,
	})

	writePictureClose(qw)

	return nil
}

// registerPictureProfile creates and registers the asset profile for a static
// picture element. Returns nil when no profile is configured or no registry is
// available.
//
// Takes ro (*RenderOrchestrator) which provides asset registration.
// Takes rctx (*renderContext) which provides rendering state and diagnostics.
// Takes attrs (*pikoImgAttrs) which contains the extracted image attributes.
// Takes formats ([]string) which lists the requested image formats.
// Takes fallbackFormat (string) which is the fallback format to include.
//
// Returns *registry_dto.ArtefactMeta which is the registered artefact, or nil.
func registerPictureProfile(ro *RenderOrchestrator, rctx *renderContext, attrs *pikoImgAttrs, formats []string, fallbackFormat string) *registry_dto.ArtefactMeta {
	profile := attrs.toAssetProfile()
	if profile == nil {
		return nil
	}
	defer putAssetProfile(profile)
	profile.Formats = ensureFallbackFormat(formats, fallbackFormat)

	if rctx.registry == nil {
		rctx.diagnostics.AddWarning(
			"renderPikoPicture",
			"No image provider configured; responsive image features disabled",
			map[string]string{"src": attrs.src},
		)
		return nil
	}

	return ro.registerDynamicAsset(rctx.originalCtx, attrs.src, profile, rctx)
}

// writePictureOpen writes the opening <picture> tag.
//
// Takes qw (*qt.Writer) which receives the output.
func writePictureOpen(qw *qt.Writer) {
	qw.N().Z(openBracket)
	qw.N().S(elementPicture)
	qw.N().Z(closeBracket)
}

// writePictureClose writes the closing </picture> tag.
//
// Takes qw (*qt.Writer) which receives the output.
func writePictureClose(qw *qt.Writer) {
	qw.N().Z(closeTagPrefix)
	qw.N().S(elementPicture)
	qw.N().Z(closeBracket)
}

// writeImgElement writes the fallback <img> element with appropriate src,
// srcset, sizes, and directive attributes.
//
// Takes ro (*RenderOrchestrator) which provides directive writing.
// Takes node (*ast_domain.TemplateNode) which holds the element attributes.
// Takes qw (*qt.Writer) which receives the output.
// Takes rctx (*renderContext) which provides rendering state.
// Takes p (imgElementParams) which groups the artefact, transformed source,
// fallback format, image attributes, and profile flag.
func writeImgElement(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext, p imgElementParams) {
	qw.N().Z(openBracket)
	qw.N().S("img")

	if p.HasProfiles {
		writeFallbackImgSrc(qw, p.Artefact, p.TransformedSrc, p.FallbackFormat, rctx)
		writeSrcsetAttributeForFormat(qw, p.Artefact, p.TransformedSrc, p.FallbackFormat, rctx)
		writeSizesAttr(qw, p.Attrs.sizes)
	} else {
		writeSrcAttr(qw, p.TransformedSrc)
	}

	writePikoImgStaticAttrsFilteredWithExclusions(node, qw, p.HasProfiles)
	if p.HasProfiles {
		ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc, attributeSrcset)
	} else {
		ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc)
	}
	qw.N().Z(selfClose)
}

// writePictureTagWithoutSrc writes a picture element when the src attribute is
// missing.
//
// Takes ro (*RenderOrchestrator) which provides the rendering setup.
// Takes node (*ast_domain.TemplateNode) which holds the template node data.
// Takes qw (*qt.Writer) which receives the output HTML.
// Takes rctx (*renderContext) which holds the current render state.
func writePictureTagWithoutSrc(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext) {
	qw.N().Z(openBracket)
	qw.N().S(elementPicture)
	qw.N().Z(closeBracket)

	qw.N().Z(openBracket)
	qw.N().S("img")
	writePikoImgStaticAttrsFilteredWithExclusions(node, qw, false)
	ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc)
	qw.N().Z(selfClose)

	qw.N().Z(closeTagPrefix)
	qw.N().S(elementPicture)
	qw.N().Z(closeBracket)
}

// renderCMSMediaPicture renders a piko:picture with a CMS media source.
//
// Takes ro (*RenderOrchestrator) which provides element directive rendering.
// Takes node (*ast_domain.TemplateNode) which is the piko:picture element to
// render.
// Takes qw (*qt.Writer) which is the quicktemplate writer for output.
// Takes rctx (*renderContext) which provides rendering state and diagnostics.
// Takes attrs (*pikoImgAttrs) which contains the extracted attributes.
//
// Returns error when rendering fails.
func renderCMSMediaPicture(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext, attrs *pikoImgAttrs) error {
	media := attrs.cmsMediaSource
	if media == nil {
		rctx.diagnostics.AddWarning(
			"renderPikoPicture",
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
	qw.N().S(elementPicture)
	qw.N().Z(closeBracket)

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

	qw.N().Z(closeTagPrefix)
	qw.N().S(elementPicture)
	qw.N().Z(closeBracket)

	return nil
}

// writePictureSourceElement writes a <source> element with type, srcset,
// and sizes attributes for a specific image format.
//
// Takes qw (*qt.Writer) which receives the rendered output.
// Takes artefact (*registry_dto.ArtefactMeta) which provides the image profiles.
// Takes baseURL (string) which is the base path for image URLs.
// Takes format (string) which is the image format for this source element.
// Takes sizes (string) which is the sizes attribute value.
// Takes rctx (*renderContext) which provides the buffer pool.
func writePictureSourceElement(qw *qt.Writer, artefact *registry_dto.ArtefactMeta, baseURL, format, sizes string, rctx *renderContext) {
	buffer := rctx.getBuffer()
	*buffer = appendSrcsetForFormat(*buffer, artefact.DesiredProfiles, baseURL, format)

	if len(*buffer) == 0 {
		return
	}

	srcset := rctx.freezeToString(buffer)

	qw.N().Z(openBracket)
	qw.N().S("source")

	qw.N().Z(space)
	qw.N().S("type")
	qw.N().Z(equalsQuote)
	qw.N().S(formatToMIMEType(format))
	qw.N().Z(quote)

	qw.N().Z(space)
	qw.N().S("srcset")
	qw.N().Z(equalsQuote)
	qw.N().S(srcset)
	qw.N().Z(quote)

	writeSizesAttr(qw, sizes)

	qw.N().Z(selfClose)
}

// writeSrcsetAttributeForFormat writes the srcset attribute filtered to a
// specific image format.
//
// Takes qw (*qt.Writer) which receives the rendered attribute output.
// Takes artefact (*registry_dto.ArtefactMeta) which provides the image profiles.
// Takes baseURL (string) which is the base path for image URLs.
// Takes format (string) which is the image format to filter by.
// Takes rctx (*renderContext) which provides the buffer pool.
func writeSrcsetAttributeForFormat(qw *qt.Writer, artefact *registry_dto.ArtefactMeta, baseURL, format string, rctx *renderContext) {
	buffer := rctx.getBuffer()
	*buffer = appendSrcsetForFormat(*buffer, artefact.DesiredProfiles, baseURL, format)

	if len(*buffer) == 0 {
		return
	}

	srcset := rctx.freezeToString(buffer)

	qw.N().Z(space)
	qw.N().S("srcset")
	qw.N().Z(equalsQuote)
	qw.N().S(srcset)
	qw.N().Z(quote)
}

// writeFallbackImgSrc writes the src attribute for the fallback <img> element.
// Uses the largest width variant for the fallback format, or the base URL if
// no matching profile is found.
//
// Takes qw (*qt.Writer) which receives the rendered attribute output.
// Takes artefact (*registry_dto.ArtefactMeta) which provides the image profiles.
// Takes baseURL (string) which is the base path for image URLs.
// Takes format (string) which is the fallback format.
func writeFallbackImgSrc(qw *qt.Writer, artefact *registry_dto.ArtefactMeta, baseURL, format string, _ *renderContext) {
	bestKey := selectLargestProfileForFormat(artefact, format)

	if bestKey != "" {
		qw.N().Z(space)
		qw.N().S(attributeSrc)
		qw.N().Z(equalsQuote)
		qw.N().S(baseURL)
		qw.N().S("?v=")
		qw.N().S(bestKey)
		qw.N().Z(quote)
	} else {
		writeSrcAttr(qw, baseURL)
	}
}

// selectLargestProfileForFormat finds the profile with the largest width for
// the given image format. Returns the profile key, or empty string if none
// matches.
//
// Takes artefact (*registry_dto.ArtefactMeta) which provides the image profiles.
// Takes format (string) which is the image format to filter by.
//
// Returns string which is the profile key with the largest width, or empty.
func selectLargestProfileForFormat(artefact *registry_dto.ArtefactMeta, format string) string {
	formatSuffix := "_" + format

	var bestKey string
	var bestWidth int

	for i := range artefact.DesiredProfiles {
		p := &artefact.DesiredProfiles[i]
		if !strings.HasSuffix(p.Name, formatSuffix) {
			continue
		}

		w, ok := p.Profile.ResultingTags.GetByName("width")
		if !ok {
			if bestKey == "" {
				bestKey = p.Name
			}
			continue
		}

		wInt := parseDecimalDigits(w)
		if wInt > bestWidth {
			bestWidth = wInt
			bestKey = p.Name
		}
	}

	return bestKey
}

// parseDecimalDigits extracts a non-negative integer from a string by reading
// consecutive ASCII digit characters.
//
// Takes s (string) which contains the digits to parse.
//
// Returns int which is the parsed integer value.
func parseDecimalDigits(s string) int {
	n := 0
	for _, character := range s {
		if character >= '0' && character <= '9' {
			n = n*widthParseBase + int(character-'0')
		}
	}
	return n
}

// appendSrcsetForFormat builds a srcset string filtered to a specific image
// format. Only profiles whose name ends with "_{format}" are included.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes profiles ([]registry_dto.NamedProfile) which contains the image profiles.
// Takes baseURL (string) which is the base URL for image paths.
// Takes format (string) which is the format to filter by.
//
// Returns []byte which is the buffer with the filtered srcset string appended.
func appendSrcsetForFormat(buffer []byte, profiles []registry_dto.NamedProfile, baseURL, format string) []byte {
	if len(profiles) == 0 {
		return buffer
	}

	formatSuffix := "_" + format

	keys := getSortedProfileKeys(profiles)
	defer putSortedProfileKeys(keys)

	needComma := false
	for _, profileKey := range *keys {
		if !strings.HasSuffix(profileKey, formatSuffix) {
			continue
		}

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

// ensureFallbackFormat returns a new formats slice that includes the fallback
// format if not already present. The fallback is appended at the end.
//
// Takes formats ([]string) which is the list of requested formats.
// Takes fallback (string) which is the fallback format to ensure is present.
//
// Returns []string which contains all formats including the fallback.
func ensureFallbackFormat(formats []string, fallback string) []string {
	for _, f := range formats {
		if f == fallback {
			return formats
		}

		if (f == formatJPG && fallback == "jpeg") || (f == "jpeg" && fallback == formatJPG) {
			return formats
		}
	}
	result := make([]string, len(formats)+1)
	copy(result, formats)
	result[len(formats)] = fallback
	return result
}

// formatToMIMEType maps an image format name to its MIME type.
//
// Takes format (string) which is the format name (e.g. "avif", "webp", "jpg").
//
// Returns string which is the corresponding MIME type.
func formatToMIMEType(format string) string {
	switch format {
	case "avif":
		return "image/avif"
	case "webp":
		return "image/webp"
	case formatJPG, "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	default:
		return "image/" + format
	}
}

// inferFallbackFormat determines the appropriate fallback image format based on
// the source file extension. Transparent formats (.png, .gif, .webp) fall back
// to png; all others fall back to jpg.
//
// Takes src (string) which is the source image path.
//
// Returns string which is the fallback format name.
func inferFallbackFormat(src string) string {
	ext := strings.ToLower(path.Ext(src))
	switch ext {
	case ".png", ".gif", ".webp":
		return "png"
	default:
		return formatJPG
	}
}
