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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"strconv"
	"strings"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// tagPikoVideo is the tag name for the piko:video custom element.
	tagPikoVideo = "piko:video"

	// defaultVideoServePath is the URL path prefix for serving HLS video files.
	defaultVideoServePath = "/_piko/video"

	// defaultSegmentDuration is the default HLS segment length in seconds.
	defaultSegmentDuration = 10

	// videoIDByteLength is the number of random bytes used to generate a video
	// element ID.
	videoIDByteLength = 8

	// attributeQuote is the closing quote character for HTML attribute values.
	attributeQuote = `"`
)

var (
	// defaultQualities is the default set of quality levels for HLS encoding.
	defaultQualities = []string{"1080p", "720p", "480p"}

	// standardVideoAttrs is a set of standard HTML video attributes handled
	// directly. Parser lowercases all attribute names.
	standardVideoAttrs = map[string]struct{}{
		"width":       {},
		"height":      {},
		"poster":      {},
		"preload":     {},
		"controls":    {},
		"autoplay":    {},
		"muted":       {},
		"loop":        {},
		"playsinline": {},
	}

	// videoQualityConfigs maps quality names to encoding configurations.
	videoQualityConfigs = map[string]videoQualityConfig{
		"2160p": {resolution: "3840x2160", bitrate: "15000k", bandwidth: 15000000},
		"1440p": {resolution: "2560x1440", bitrate: "8000k", bandwidth: 8000000},
		"1080p": {resolution: "1920x1080", bitrate: "5000k", bandwidth: 5000000},
		"720p":  {resolution: "1280x720", bitrate: "2500k", bandwidth: 2500000},
		"480p":  {resolution: "854x480", bitrate: "1000k", bandwidth: 1000000},
		"360p":  {resolution: "640x360", bitrate: "500k", bandwidth: 500000},
	}
)

// pikoVideoAttrs holds all attributes extracted from a piko:video element.
// Uses single-pass extraction to read node attributes only once.
type pikoVideoAttrs struct {
	// qualities is a comma-separated list of quality levels (e.g.
	// "1080p,720p,480p").
	qualities string

	// width is the video element width in pixels or CSS units.
	width string

	// posterWidths specifies the target widths for poster images (e.g.
	// "640,1280").
	posterWidths string

	// posterFormats specifies output formats as a comma-separated list
	// (e.g. "webp,jpeg").
	posterFormats string

	// posterDensities specifies screen density options (e.g. "1x,2x").
	posterDensities string

	// posterSizes specifies the CSS sizes attribute for a responsive poster image.
	posterSizes string

	// preload specifies when to load the video: auto, metadata, or none.
	preload string

	// thumbnailTime specifies when to capture the thumbnail, such as "5s".
	thumbnailTime string

	// poster is the URL of the image shown before the video plays.
	poster string

	// src is the video source URL or file path; empty triggers a warning.
	src string

	// segmentDuration is the HLS segment length in seconds.
	segmentDuration string

	// height is the video element height in pixels.
	height string

	// thumbnail enables automatic thumbnail generation from a video frame.
	thumbnail bool

	// controls indicates whether video playback controls are shown.
	controls bool

	// autoplay indicates whether the video should start playing on its own.
	autoplay bool

	// muted indicates whether the video starts with sound off.
	muted bool

	// loop indicates whether the video should restart when it ends.
	loop bool

	// playsInline indicates whether the video plays inline on mobile devices.
	playsInline bool
}

// hasQualityProfile returns true if quality-related attributes were set.
//
// Returns bool which is true when qualities or segmentDuration is set.
func (a *pikoVideoAttrs) hasQualityProfile() bool {
	return a.qualities != "" || a.segmentDuration != ""
}

// getQualities returns the parsed quality levels or the default values.
//
// Returns []string which contains the quality level names.
func (a *pikoVideoAttrs) getQualities() []string {
	if a.qualities == "" {
		return defaultQualities
	}
	return parseCommaSeparated(a.qualities)
}

// getSegmentDuration returns the segment duration in seconds.
//
// Returns int which is the parsed segment duration, or the default value if
// the duration is empty, not a valid number, or not greater than zero.
func (a *pikoVideoAttrs) getSegmentDuration() int {
	if a.segmentDuration == "" {
		return defaultSegmentDuration
	}
	if value, err := strconv.Atoi(a.segmentDuration); err == nil && value > 0 {
		return value
	}
	return defaultSegmentDuration
}

// hasPosterProfile returns true if any poster profile-related attributes were
// set, indicating the poster should be processed as an image asset.
//
// Returns bool which is true when posterWidths, posterFormats, posterDensities,
// or posterSizes is set.
func (a *pikoVideoAttrs) hasPosterProfile() bool {
	return a.posterWidths != "" || a.posterFormats != "" ||
		a.posterDensities != "" || a.posterSizes != ""
}

// needsPosterTransform reports whether the poster path needs to be changed.
// External URLs (http://, https://, //, data:) do not need changes.
//
// Returns bool which is true when the poster is a local path that needs
// processing.
func (a *pikoVideoAttrs) needsPosterTransform() bool {
	if a.poster == "" {
		return false
	}
	return assetpath.NeedsTransform(a.poster, assetpath.DefaultServePath)
}

// toPosterProfile converts poster attributes to an assetProfile for image
// processing, letting the poster be handled like a piko:img element.
//
// Returns *assetProfile which contains the parsed poster profile settings, or
// nil if no poster or poster profile attributes were found.
func (a *pikoVideoAttrs) toPosterProfile() *assetProfile {
	if a.poster == "" || !a.needsPosterTransform() {
		return nil
	}

	if !a.hasPosterProfile() {
		return nil
	}

	profile := &assetProfile{
		Sizes:     a.posterSizes,
		Densities: []string{"1x"},
		Formats:   []string{"webp"},
		Widths:    nil,
	}

	if a.posterDensities != "" {
		profile.Densities = parseCommaSeparated(a.posterDensities)
	}
	if a.posterFormats != "" {
		profile.Formats = parseCommaSeparated(a.posterFormats)
	}
	if a.posterWidths != "" {
		profile.Widths = parseIntList(a.posterWidths)
	}

	return profile
}

// registerDynamicVideoAsset registers a video asset with the registry for HLS
// encoding.
//
// Takes src (string) which is the original video source path.
// Takes qualities ([]string) which contains the desired quality levels.
// Takes segmentDuration (int) which is the HLS segment duration in seconds.
// Takes rctx (*renderContext) which provides the render context.
//
// Returns *registry_dto.ArtefactMeta which is the registered artefact metadata.
func (*RenderOrchestrator) registerDynamicVideoAsset(
	_ any,
	src string,
	qualities []string,
	segmentDuration int,
	rctx *renderContext,
) *registry_dto.ArtefactMeta {
	if rctx.registeredDynamicAssets == nil {
		rctx.registeredDynamicAssets = make(map[string]*registry_dto.ArtefactMeta)
	}

	cacheKey := buildVideoCacheKey(src, qualities, segmentDuration)
	if cached, exists := rctx.registeredDynamicAssets[cacheKey]; exists {
		return cached
	}

	desiredProfiles := buildVideoDesiredProfiles(qualities, segmentDuration)

	artefact := &registry_dto.ArtefactMeta{
		ID:              src,
		SourcePath:      src,
		Status:          registry_dto.VariantStatusPending,
		DesiredProfiles: desiredProfiles,
	}

	rctx.registeredDynamicAssets[cacheKey] = artefact
	return artefact
}

// videoQualityConfig holds the encoding settings for a video quality level.
type videoQualityConfig struct {
	// resolution specifies the video resolution in WxH format
	// (e.g. "1920x1080", "1280x720").
	resolution string

	// bitrate is the target video bitrate for encoding, such as "1500k".
	bitrate string

	// bandwidth is the target bit rate in bits per second.
	bandwidth int
}

// processPoster handles poster image changes for piko:video elements.
// It updates the poster path, optionally registers it as a dynamic asset
// for image optimisation, or generates an auto-thumbnail URL.
//
// Takes rctx (*renderContext) which provides the rendering context.
// Takes attrs (*pikoVideoAttrs) which contains the poster attributes.
// Takes transformedSrc (string) which is the updated video source path.
//
// Returns string which is the updated poster URL, or empty if no poster.
func (ro *RenderOrchestrator) processPoster(rctx *renderContext, attrs *pikoVideoAttrs, transformedSrc string) string {
	if attrs.poster != "" {
		return ro.processExplicitPoster(rctx, attrs)
	}

	if attrs.thumbnail {
		return buildThumbnailURL(transformedSrc, attrs.thumbnailTime)
	}

	return ""
}

// processExplicitPoster handles a user-provided poster image.
//
// Takes rctx (*renderContext) which provides the rendering context.
// Takes attrs (*pikoVideoAttrs) which contains the poster settings.
//
// Returns string which is the transformed poster URL, or the original if no
// transform is needed.
func (ro *RenderOrchestrator) processExplicitPoster(rctx *renderContext, attrs *pikoVideoAttrs) string {
	if !attrs.needsPosterTransform() {
		return attrs.poster
	}

	buffer := rctx.getBuffer()
	*buffer = assetpath.AppendTransformed(*buffer, attrs.poster, assetpath.DefaultServePath)
	transformedPoster := rctx.freezeToString(buffer)

	posterProfile := attrs.toPosterProfile()
	if posterProfile != nil {
		ro.registerDynamicAsset(rctx.originalCtx, attrs.poster, posterProfile, rctx)
	}

	return transformedPoster
}

// extractPikoVideoAttrs extracts all piko:video attributes in a single pass.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to extract
// attributes from.
//
// Returns pikoVideoAttrs which holds the extracted video attributes.
func extractPikoVideoAttrs(node *ast_domain.TemplateNode) pikoVideoAttrs {
	var result pikoVideoAttrs
	extractStaticPikoVideoAttrs(node, &result)
	if result.src == "" {
		result.src = extractDynamicSrc(node)
	}
	return result
}

// extractStaticPikoVideoAttrs fills result with values from static HTML
// attributes on the node.
//
// Takes node (*ast_domain.TemplateNode) which contains the attributes to read.
// Takes result (*pikoVideoAttrs) which receives the extracted values.
func extractStaticPikoVideoAttrs(node *ast_domain.TemplateNode, result *pikoVideoAttrs) {
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		assignPikoVideoAttr(attr.Name, attr.Value, result)
	}
}

// videoAttrSetters maps attribute names to setter functions
// that assign values on pikoVideoAttrs.
var videoAttrSetters = map[string]func(string, *pikoVideoAttrs){
	attributeSrc:       func(v string, a *pikoVideoAttrs) { a.src = v },
	"poster":           func(v string, a *pikoVideoAttrs) { a.poster = v },
	"poster-widths":    func(v string, a *pikoVideoAttrs) { a.posterWidths = v },
	"poster-width":     func(v string, a *pikoVideoAttrs) { a.posterWidths = v },
	"poster-formats":   func(v string, a *pikoVideoAttrs) { a.posterFormats = v },
	"poster-format":    func(v string, a *pikoVideoAttrs) { a.posterFormats = v },
	"poster-densities": func(v string, a *pikoVideoAttrs) { a.posterDensities = v },
	"poster-density":   func(v string, a *pikoVideoAttrs) { a.posterDensities = v },
	"poster-sizes":     func(v string, a *pikoVideoAttrs) { a.posterSizes = v },
	"thumbnail":        func(_ string, a *pikoVideoAttrs) { a.thumbnail = true },
	"thumbnail-time":   func(v string, a *pikoVideoAttrs) { a.thumbnailTime = v },
	"qualities":        func(v string, a *pikoVideoAttrs) { a.qualities = v },
	"segment-duration": func(v string, a *pikoVideoAttrs) { a.segmentDuration = v },
	"width":            func(v string, a *pikoVideoAttrs) { a.width = v },
	"height":           func(v string, a *pikoVideoAttrs) { a.height = v },
	"preload":          func(v string, a *pikoVideoAttrs) { a.preload = v },
	"controls":         func(_ string, a *pikoVideoAttrs) { a.controls = true },
	"autoplay":         func(_ string, a *pikoVideoAttrs) { a.autoplay = true },
	"muted":            func(_ string, a *pikoVideoAttrs) { a.muted = true },
	"loop":             func(_ string, a *pikoVideoAttrs) { a.loop = true },
	"playsinline":      func(_ string, a *pikoVideoAttrs) { a.playsInline = true },
}

// assignPikoVideoAttr sets a field in result based on the attribute name.
//
// The parser lowercases attribute names during parsing, so direct comparison
// is used. Dispatches through the videoAttrSetters lookup table.
//
// Takes name (string) which is the attribute name to match.
// Takes value (string) which is the value to assign.
// Takes result (*pikoVideoAttrs) which receives the assigned value.
func assignPikoVideoAttr(name, value string, result *pikoVideoAttrs) {
	if setter, ok := videoAttrSetters[name]; ok {
		setter(value, result)
	}
}

// isPikoVideoSpecialAttr checks whether an attribute name is a special
// piko:video attribute that should be removed from the output.
//
// Uses switch instead of map for faster lookup. Parser lowercases all
// attribute names, so direct comparison is safe.
//
// Takes name (string) which is the attribute name to check.
//
// Returns bool which is true when the name matches a special attribute.
func isPikoVideoSpecialAttr(name string) bool {
	switch name {
	case "qualities", "segment-duration",
		"poster-widths", "poster-width",
		"poster-formats", "poster-format",
		"poster-densities", "poster-density",
		"poster-sizes", "thumbnail", "thumbnail-time":
		return true
	default:
		return false
	}
}

// renderPikoVideo renders a <piko:video> component as a <video> tag with HLS.js
// integration using the direct-to-writer pattern.
//
// Takes ro (*RenderOrchestrator) which provides asset registration and
// element directive rendering.
// Takes node (*ast_domain.TemplateNode) which is the piko:video element to
// render.
// Takes qw (*qt.Writer) which is the quicktemplate writer for output.
// Takes rctx (*renderContext) which provides rendering state and diagnostics.
//
// Returns error when rendering fails.
func renderPikoVideo(ro *RenderOrchestrator, node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext) error {
	VideoTransformCount.Add(rctx.originalCtx, 1)

	attrs := extractPikoVideoAttrs(node)
	if attrs.src == "" {
		handleMissingVideoSrc(node.Attributes, qw, rctx)
		return nil
	}

	buffer := rctx.getBuffer()
	*buffer = assetpath.AppendTransformed(*buffer, attrs.src, assetpath.DefaultServePath)
	transformedSrc := rctx.freezeToString(buffer)

	var artefact *registry_dto.ArtefactMeta
	if attrs.hasQualityProfile() {
		artefact = ro.registerDynamicVideoAsset(
			rctx.originalCtx,
			attrs.src,
			attrs.getQualities(),
			attrs.getSegmentDuration(),
			rctx,
		)
	}
	transformedPoster := ro.processPoster(rctx, &attrs, transformedSrc)

	videoID := generateVideoID()

	manifestURL := buildManifestURL(transformedSrc)

	writeVideoElement(qw, videoID, &attrs, manifestURL, transformedPoster, artefact)

	writeHLSScript(qw, videoID, manifestURL)

	writePikoVideoStaticAttrsFiltered(node, qw)
	ro.writeElementDirectivesExcluding(node, qw, rctx, attributeSrc)

	return nil
}

// handleMissingVideoSrc handles a piko:video tag that has no src attribute by
// logging a warning and writing an error div to the output.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the tag attributes.
// Takes qw (*qt.Writer) which writes the error output.
// Takes rctx (*renderContext) which provides the render context.
func handleMissingVideoSrc(attrs []ast_domain.HTMLAttribute, qw *qt.Writer, rctx *renderContext) {
	userAttrs := extractVideoUserAttrs(attrs)
	rctx.diagnostics.AddWarning("renderPikoVideo",
		"piko:video tag missing src attribute",
		map[string]string{"pageID": rctx.pageID})
	VideoTransformErrorCount.Add(rctx.originalCtx, 1)
	writeErrorDiv(qw, userAttrs, "<!-- piko:video error: 'src' attribute is missing -->")
}

// extractVideoUserAttrs filters HTML attributes to keep only user-defined ones.
// It excludes the src attribute and piko:video specific attributes.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the original
// attributes to filter.
//
// Returns []ast_domain.HTMLAttribute which contains only the user-defined
// attributes.
func extractVideoUserAttrs(attrs []ast_domain.HTMLAttribute) []ast_domain.HTMLAttribute {
	userAttrCount := 0
	for i := range attrs {
		name := attrs[i].Name
		if name != attributeSrc && !isPikoVideoSpecialAttr(name) {
			userAttrCount++
		}
	}

	userAttrs := make([]ast_domain.HTMLAttribute, 0, userAttrCount)
	for i := range attrs {
		name := attrs[i].Name
		if name != attributeSrc && !isPikoVideoSpecialAttr(name) {
			userAttrs = append(userAttrs, attrs[i])
		}
	}
	return userAttrs
}

// generateVideoID creates a unique identifier for a video element.
//
// Returns string which is the unique video element identifier.
func generateVideoID() string {
	b := make([]byte, videoIDByteLength)
	_, _ = rand.Read(b)
	return "piko-video-" + hex.EncodeToString(b)
}

// buildManifestURL builds the HLS master manifest URL from a video path.
//
// Takes videoPath (string) which is the video source path.
//
// Returns string which is the full URL to the HLS manifest file.
func buildManifestURL(videoPath string) string {
	return defaultVideoServePath + "/" + videoPath + "/master.m3u8"
}

// writeVideoElement writes the <video> element with all attributes.
//
// Takes qw (*qt.Writer) which receives the HTML output.
// Takes videoID (string) which is the unique ID for the video element.
// Takes attrs (*pikoVideoAttrs) which contains the extracted video attributes.
// Takes manifestURL (string) which is the HLS manifest URL.
// Takes posterURL (string) which is the transformed poster URL (may be empty).
func writeVideoElement(qw *qt.Writer, videoID string, attrs *pikoVideoAttrs, manifestURL, posterURL string, _ *registry_dto.ArtefactMeta) {
	qw.N().S("<video")
	writeVideoAttr(qw, "id", videoID, false)
	writeVideoAttrIfSet(qw, "width", attrs.width)
	writeVideoAttrIfSet(qw, "height", attrs.height)
	writeVideoAttrIfSet(qw, "poster", posterURL)
	writeVideoPreloadAttr(qw, attrs.preload)
	writeVideoBooleanAttrs(qw, attrs)
	qw.N().S(`>`)
	writeVideoSource(qw, manifestURL)
	qw.N().S(`</video>`)
}

// writeVideoAttr writes a single HTML attribute to the output.
//
// Takes qw (*qt.Writer) which receives the attribute output.
// Takes name (string) which specifies the attribute name.
// Takes value (string) which provides the attribute value.
// Takes escape (bool) which controls whether to HTML-escape the value.
func writeVideoAttr(qw *qt.Writer, name, value string, escape bool) {
	qw.N().S(` `)
	qw.N().S(name)
	qw.N().S(`="`)
	if escape {
		qw.N().S(html.EscapeString(value))
	} else {
		qw.N().S(value)
	}
	qw.N().S(attributeQuote)
}

// writeVideoAttrIfSet writes an attribute only if the value is not empty.
//
// Takes qw (*qt.Writer) which receives the attribute output.
// Takes name (string) which specifies the attribute name.
// Takes value (string) which provides the attribute value to write.
func writeVideoAttrIfSet(qw *qt.Writer, name, value string) {
	if value != "" {
		writeVideoAttr(qw, name, value, true)
	}
}

// writeVideoPreloadAttr writes the preload attribute to the video element.
//
// When preload is empty, writes the default value "metadata".
//
// Takes qw (*qt.Writer) which is the template writer for output.
// Takes preload (string) which is the preload value, or empty for default.
func writeVideoPreloadAttr(qw *qt.Writer, preload string) {
	if preload != "" {
		writeVideoAttr(qw, "preload", preload, true)
	} else {
		qw.N().S(` preload="metadata"`)
	}
}

// writeVideoBooleanAttrs writes all boolean video attributes to the output.
//
// Takes qw (*qt.Writer) which receives the attribute output.
// Takes attrs (*pikoVideoAttrs) which holds the boolean attribute values.
func writeVideoBooleanAttrs(qw *qt.Writer, attrs *pikoVideoAttrs) {
	if attrs.controls {
		qw.N().S(` controls`)
	}
	if attrs.autoplay {
		qw.N().S(` autoplay`)
	}
	if attrs.muted {
		qw.N().S(` muted`)
	}
	if attrs.loop {
		qw.N().S(` loop`)
	}
	if attrs.playsInline {
		qw.N().S(` playsinline`)
	}
}

// writeVideoSource writes an HLS source element with fallback text.
//
// Takes qw (*qt.Writer) which receives the HTML output.
// Takes manifestURL (string) which specifies the HLS manifest location.
func writeVideoSource(qw *qt.Writer, manifestURL string) {
	qw.N().S(`<source src="`)
	qw.N().S(html.EscapeString(manifestURL))
	qw.N().S(`" type="application/x-mpegURL">`)
	qw.N().S(`Your browser does not support HLS video.`)
}

// writeHLSScript writes inline JavaScript that sets up HLS.js video playback.
//
// Takes qw (*qt.Writer) which receives the script output.
// Takes videoID (string) which is the HTML element ID of the video player.
// Takes manifestURL (string) which is the URL to the HLS manifest file.
func writeHLSScript(qw *qt.Writer, videoID, manifestURL string) {
	qw.N().S(`<script>`)
	qw.N().S(`(function(){`)
	qw.N().S(`var v=document.getElementById('`)
	qw.N().S(videoID)
	qw.N().S(`');`)
	qw.N().S(`if(typeof Hls!=='undefined'&&Hls.isSupported()){`)
	qw.N().S(`var h=new Hls();`)
	qw.N().S(`h.loadSource('`)
	qw.N().S(html.EscapeString(manifestURL))
	qw.N().S(`');`)
	qw.N().S(`h.attachMedia(v);`)
	qw.N().S(`}else if(v.canPlayType('application/vnd.apple.mpegurl')){`)
	qw.N().S(`v.src='`)
	qw.N().S(html.EscapeString(manifestURL))
	qw.N().S(`';`)
	qw.N().S(`}`)
	qw.N().S(`})();`)
	qw.N().S(`</script>`)
}

// writePikoVideoStaticAttrsFiltered writes static attributes to the output,
// skipping video-specific attributes and the src attribute.
//
// Takes node (*ast_domain.TemplateNode) which contains the attributes to write.
// Takes qw (*qt.Writer) which is the output writer for the attributes.
func writePikoVideoStaticAttrsFiltered(node *ast_domain.TemplateNode, qw *qt.Writer) {
	for i := range node.Attributes {
		attr := &node.Attributes[i]

		if attr.Name == attributeSrc || isPikoVideoSpecialAttr(attr.Name) {
			continue
		}

		if isStandardVideoAttr(attr.Name) {
			continue
		}

		qw.N().Z(space)
		qw.N().S(attr.Name)
		qw.N().Z(equalsQuote)
		qw.N().S(attr.Value)
		qw.N().Z(quote)
	}
}

// isStandardVideoAttr reports whether the given attribute name is a standard
// HTML video attribute that writeVideoElement handles directly.
//
// Takes name (string) which is the attribute name to check.
//
// Returns bool which is true if the attribute is standard.
func isStandardVideoAttr(name string) bool {
	_, ok := standardVideoAttrs[name]
	return ok
}

// buildVideoCacheKey creates a cache key for video asset registration.
//
// Takes src (string) which is the video source path.
// Takes qualities ([]string) which lists the quality levels available.
// Takes segmentDuration (int) which is the segment length in seconds.
//
// Returns string which is the cache key.
func buildVideoCacheKey(src string, qualities []string, segmentDuration int) string {
	return fmt.Sprintf("video:%s:%s:%d", src, strings.Join(qualities, ","), segmentDuration)
}

// buildVideoDesiredProfiles builds the desired profiles for HLS video encoding.
//
// Takes qualities ([]string) which contains the quality level names.
// Takes segmentDuration (int) which specifies the segment length in seconds.
//
// Returns []registry_dto.NamedProfile which contains the encoding profiles
// for HLS video output.
func buildVideoDesiredProfiles(qualities []string, segmentDuration int) []registry_dto.NamedProfile {
	profiles := make([]registry_dto.NamedProfile, 0, len(qualities))

	for _, quality := range qualities {
		config, ok := videoQualityConfigs[quality]
		if !ok {
			continue
		}

		profileName := "hls_" + quality

		var params registry_dto.ProfileParams
		params.SetByName("resolution", config.resolution)
		params.SetByName("bitrate", config.bitrate)
		params.SetByName("segment_duration", strconv.Itoa(segmentDuration))

		var deps registry_dto.Dependencies
		deps.Add("source")

		var resultingTags registry_dto.Tags
		resultingTags.SetByName("quality", quality)
		resultingTags.SetByName("bandwidth", strconv.Itoa(config.bandwidth))

		profiles = append(profiles, registry_dto.NamedProfile{
			Name: profileName,
			Profile: registry_dto.DesiredProfile{
				Priority:       registry_dto.PriorityNeed,
				CapabilityName: "video.encode.hls",
				DependsOn:      deps,
				Params:         params,
				ResultingTags:  resultingTags,
			},
		})
	}

	return profiles
}

// buildThumbnailURL builds a URL for a video thumbnail that is created
// automatically.
//
// Takes videoPath (string) which is the path to the video source file.
// Takes thumbnailTime (string) which is the time to extract the frame, or
// empty to use the default.
//
// Returns string which is the thumbnail URL.
func buildThumbnailURL(videoPath, thumbnailTime string) string {
	url := defaultVideoServePath + "/" + videoPath + "/thumbnail.jpg"
	if thumbnailTime != "" {
		url += "?t=" + thumbnailTime
	}
	return url
}
