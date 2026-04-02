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

package lifecycle_domain

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// defaultImageQuality is the default JPEG quality setting used when no quality
	// value is given.
	defaultImageQuality = 80

	// defaultStorageBackendID is the storage backend identifier for asset storage.
	defaultStorageBackendID = "local_disk_cache"

	// imageProfileNamePrefix is the prefix for generated image variant profile
	// names.
	imageProfileNamePrefix = "image"

	// imageVariantTypeTag is the tag value that marks a profile as an image
	// variant.
	imageVariantTypeTag = "image-variant"

	// imageCapabilityName is the name used for image transform profiles.
	imageCapabilityName = capabilities_dto.CapabilityImageTransform

	// assetDependencySource is the dependency key for the original source asset.
	assetDependencySource = "source"

	// defaultImagePriority is the priority level for image variant profiles.
	defaultImagePriority = registry_dto.PriorityWant

	// defaultImageFileExtension is the fallback file extension for image variants.
	defaultImageFileExtension = ".img"

	// fieldAssetPath is the logging field name for asset paths.
	fieldAssetPath = "asset_path"

	// fieldFormat is the parameter key for the image format setting.
	fieldFormat = "format"

	// percentMax is the value used for percentage calculations.
	percentMax = 100

	// paramPlaceholder is the parameter key for enabling placeholder generation.
	paramPlaceholder = "placeholder"

	// paramPlaceholderWidth is the parameter key for the placeholder image width.
	paramPlaceholderWidth = "placeholder-width"

	// paramPlaceholderHeight is the parameter name for the placeholder image
	// height.
	paramPlaceholderHeight = "placeholder-height"

	// paramPlaceholderQuality is the parameter name for placeholder image quality.
	paramPlaceholderQuality = "placeholder-quality"

	// paramPlaceholderBlur is the parameter name for placeholder image blur
	// radius.
	paramPlaceholderBlur = "placeholder-blur"

	// videoProfileNamePrefix is the prefix used in video variant profile names.
	videoProfileNamePrefix = "hls"

	// videoVariantTypeTag is the tag value that marks video variant profiles.
	videoVariantTypeTag = "video-variant"

	// videoCapabilityName is the capability name for video transcoding.
	videoCapabilityName = capabilities_dto.CapabilityVideoTranscode

	// fallbackVideoSegmentDuration is the fallback HLS segment duration when not
	// configured.
	fallbackVideoSegmentDuration = 10
)

var (
	// standardViewportWidths defines viewport widths used to resolve 'vw' units
	// into concrete pixel values.
	standardViewportWidths = []int{320, 640, 768, 1024, 1280, 1536, 1920}

	// defaultScreens defines standard responsive breakpoints for converting
	// viewport-relative sizes (like "sm:50vw") to concrete pixel widths. These
	// match common CSS framework conventions.
	defaultScreens = map[string]int{
		"sm":  640,
		"md":  768,
		"lg":  1024,
		"xl":  1280,
		"2xl": 1536,
	}

	// _ references defaultScreens to satisfy staticcheck, as it is reserved for
	// future use.
	_ = defaultScreens

	// fallbackVideoQualities defines the fallback quality levels when not
	// configured.
	fallbackVideoQualities = []string{"1080p", "720p", "480p"}

	// videoQualityConfigs maps quality names to their encoding settings.
	videoQualityConfigs = map[string]videoQualityConfig{
		"1080p": {resolution: "1920x1080", bitrate: "5000k", bandwidth: 5000000},
		"720p":  {resolution: "1280x720", bitrate: "2500k", bandwidth: 2500000},
		"480p":  {resolution: "854x480", bitrate: "1000k", bandwidth: 1000000},
		"360p":  {resolution: "640x360", bitrate: "500k", bandwidth: 500000},
	}
)

// AssetPipelineOrchestrator translates high-level asset requirements from a
// build manifest into detailed processing instructions for the registry. It
// implements AssetPipelinePort, acting as the bridge between the annotator's
// output and the orchestrator's input.
type AssetPipelineOrchestrator struct {
	// registryService stores and updates artefacts in the registry.
	registryService registry_domain.RegistryService

	// assetsConfig holds asset profiles and video defaults. Separate from
	// ServerConfig because assets are configured programmatically.
	assetsConfig *config.AssetsConfig
}

// NewAssetPipelineOrchestrator creates a new asset pipeline orchestrator.
//
// Takes registry (RegistryService) which provides access to registry services.
// Takes assetsConfig (*AssetsConfig) which holds asset profiles and video
// defaults; may be nil for default behaviour.
//
// Returns *AssetPipelineOrchestrator which is ready for use.
func NewAssetPipelineOrchestrator(registry registry_domain.RegistryService, assetsConfig *config.AssetsConfig) *AssetPipelineOrchestrator {
	return &AssetPipelineOrchestrator{
		registryService: registry,
		assetsConfig:    assetsConfig,
	}
}

// ProcessBuildResult is the main entry point for asset processing. It iterates
// through the final asset manifest from a build, generates the necessary
// DesiredProfiles for each asset, and upserts them into the registry.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// build output including the final asset manifest to process.
//
// Returns error when processing fails, though individual asset failures are
// logged and do not halt the pipeline.
func (p *AssetPipelineOrchestrator) ProcessBuildResult(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error {
	ctx, span, l := log.Span(ctx, "AssetPipelineOrchestrator.ProcessBuildResult")
	defer span.End()

	if result == nil || len(result.FinalAssetManifest) == 0 {
		l.Trace("No asset manifest found in build result, nothing to process.")
		return nil
	}

	l.Trace("Processing build asset manifest to generate registry profiles...", logger_domain.Int("asset_count", len(result.FinalAssetManifest)))

	for _, asset := range result.FinalAssetManifest {
		l.Trace("Processing asset", logger_domain.String(fieldAssetPath, asset.SourcePath), logger_domain.String("asset_type", asset.AssetType))

		var desiredProfiles []registry_dto.NamedProfile

		switch asset.AssetType {
		case "img":
			desiredProfiles = p.generateImageProfiles(asset)
		case "video":
			desiredProfiles = p.generateVideoProfiles(asset)
		default:
			desiredProfiles = GetProfilesForFile(asset.SourcePath, nil)
		}

		if len(desiredProfiles) == 0 {
			l.Trace("No desired profiles generated for asset, skipping.", logger_domain.String(fieldAssetPath, asset.SourcePath))
			continue
		}

		l.Trace("Generated desired profiles for asset", logger_domain.Int("profile_count", len(desiredProfiles)), logger_domain.String(fieldAssetPath, asset.SourcePath))

		_, err := p.registryService.UpsertArtefact(
			ctx,
			asset.SourcePath,
			asset.SourcePath,
			nil,
			defaultStorageBackendID,
			desiredProfiles,
		)
		if err != nil {
			l.Error("Failed to upsert asset with generated profiles", logger_domain.Error(err), logger_domain.String(fieldAssetPath, asset.SourcePath))
		}
	}

	l.Trace("Finished processing asset manifest.")
	return nil
}

// imageProfileConfig holds parsed configuration for generating image profiles.
type imageProfileConfig struct {
	// standardParams holds standard transformation parameters such as fit and
	// aspectratio.
	standardParams map[string]string

	// modifiers are provider-specific passthrough modifiers (greyscale, blur,
	// sharpen, etc.).
	modifiers map[string]string

	// requiredWidths is the list of pixel widths to generate.
	requiredWidths []int

	// formats lists the output image formats such as "webp", "avif", or "jpeg".
	formats []string

	// quality is the image compression level from 1 to 100.
	quality int
}

// generateImageProfiles handles piko:img assets by parsing rich transformation
// parameters (sizes, densities, formats) and generating NamedProfiles for
// every required image variant.
//
// Takes asset (*annotator_dto.FinalAssetDependency) which provides the asset
// metadata and transformation parameters to process.
//
// Returns []registry_dto.NamedProfile which contains all generated image
// variant profiles, including placeholder if configured.
func (p *AssetPipelineOrchestrator) generateImageProfiles(asset *annotator_dto.FinalAssetDependency) []registry_dto.NamedProfile {
	params := asset.TransformationParams
	imageConfig := p.parseImageConfig(params, asset.SourcePath)

	profiles := p.generateVariantProfiles(imageConfig)

	if p.shouldGeneratePlaceholder(params) {
		if profile := p.generatePlaceholderProfile(params, imageConfig.quality, imageConfig.standardParams, imageConfig.modifiers); profile != nil {
			profiles = append(profiles, *profile)
		}
	}

	return profiles
}

// parseImageConfig extracts all configuration needed for image profile
// generation.
//
// Takes params (map[string][]string) which contains the image parameters such
// as sizes, densities, quality, and format settings.
// Takes sourcePath (string) which is the path to the source image file.
//
// Returns imageProfileConfig which contains the parsed configuration including
// required widths, quality settings, formats, and modifiers.
func (p *AssetPipelineOrchestrator) parseImageConfig(params map[string][]string, sourcePath string) imageProfileConfig {
	return imageProfileConfig{
		requiredWidths: p.calculateRequiredWidths(params["sizes"], params["densities"]),
		quality:        p.parseQuality(params["quality"]),
		formats:        p.parseFormats(params[fieldFormat], sourcePath),
		standardParams: p.extractStandardParams(params),
		modifiers:      p.getPassthroughModifiers(params),
	}
}

// generateVariantProfiles creates a NamedProfile for each width/format
// combination.
//
// Takes imageConfig (imageProfileConfig) which specifies the required widths and
// formats for variant generation.
//
// Returns []registry_dto.NamedProfile which contains the generated profiles
// for all width/format combinations.
func (*AssetPipelineOrchestrator) generateVariantProfiles(imageConfig imageProfileConfig) []registry_dto.NamedProfile {
	profiles := make([]registry_dto.NamedProfile, 0, len(imageConfig.formats)*len(imageConfig.requiredWidths))
	for _, width := range imageConfig.requiredWidths {
		for _, format := range imageConfig.formats {
			profiles = append(profiles, createVariantProfile(width, format, imageConfig))
		}
	}
	return profiles
}

// shouldGeneratePlaceholder checks if placeholder generation is enabled in
// the params.
//
// Takes params (map[string][]string) which contains the request parameters.
//
// Returns bool which is true when placeholder generation is explicitly enabled
// via values like "true", "yes", "1", or "enabled".
func (*AssetPipelineOrchestrator) shouldGeneratePlaceholder(params map[string][]string) bool {
	if placeholderValues, ok := params[paramPlaceholder]; ok && len(placeholderValues) > 0 {
		value := strings.ToLower(placeholderValues[0])
		return value == "true" || value == "yes" || value == "1" || value == "enabled"
	}
	return false
}

// generatePlaceholderProfile creates a profile for generating a LQIP
// (Low Quality Image Placeholder). The unnamed parameters are reserved
// for future use when placeholder quality and modifiers are
// customisable.
//
// Takes params (map[string][]string) which provides the query
// parameters for placeholder configuration.
// Takes standardParams (map[string]string) which provides standard
// rendering parameters to include in the profile.
//
// Returns *registry_dto.NamedProfile which is the configured
// placeholder profile ready for image processing.
func (*AssetPipelineOrchestrator) generatePlaceholderProfile(
	params map[string][]string,
	_ int,
	standardParams map[string]string,
	_ map[string]string,
) *registry_dto.NamedProfile {
	var capabilityParams registry_dto.ProfileParams
	capabilityParams.SetByName(paramPlaceholder, "true")

	if widthValues, ok := params[paramPlaceholderWidth]; ok && len(widthValues) > 0 {
		capabilityParams.SetByName(paramPlaceholderWidth, widthValues[0])
	} else {
		capabilityParams.SetByName(paramPlaceholderWidth, "20")
	}

	if heightValues, ok := params[paramPlaceholderHeight]; ok && len(heightValues) > 0 {
		capabilityParams.SetByName(paramPlaceholderHeight, heightValues[0])
	}

	if qualityValues, ok := params[paramPlaceholderQuality]; ok && len(qualityValues) > 0 {
		capabilityParams.SetByName(paramPlaceholderQuality, qualityValues[0])
	} else {
		capabilityParams.SetByName(paramPlaceholderQuality, "10")
	}

	if blurValues, ok := params[paramPlaceholderBlur]; ok && len(blurValues) > 0 {
		capabilityParams.SetByName(paramPlaceholderBlur, blurValues[0])
	} else {
		capabilityParams.SetByName(paramPlaceholderBlur, "5.0")
	}

	for k, v := range standardParams {
		capabilityParams.SetByName(k, v)
	}

	if formatValues, ok := params[fieldFormat]; ok && len(formatValues) > 0 {
		formats := strings.Split(formatValues[0], ",")
		if len(formats) > 0 {
			capabilityParams.SetByName(fieldFormat, strings.TrimSpace(formats[0]))
		}
	} else {
		capabilityParams.SetByName(fieldFormat, "webp")
	}

	var resultingTags registry_dto.Tags
	resultingTags.SetByName("type", "image-placeholder")
	resultingTags.SetByName("storageBackendId", defaultStorageBackendID)
	resultingTags.SetByName("fileExtension", ".placeholder.img")

	var deps registry_dto.Dependencies
	deps.Add(assetDependencySource)

	return &registry_dto.NamedProfile{
		Name: "placeholder",
		Profile: registry_dto.DesiredProfile{
			CapabilityName: string(imageCapabilityName),
			Params:         capabilityParams,
			DependsOn:      deps,
			Priority:       registry_dto.PriorityNeed,
			ResultingTags:  resultingTags,
		},
	}
}

// extractStandardParams extracts known standard transformation parameters that
// should be passed to every variant, such as fit mode and aspect ratio.
//
// Takes params (map[string][]string) which contains the raw query parameters.
//
// Returns map[string]string which contains the first value for each known
// standard parameter key.
func (*AssetPipelineOrchestrator) extractStandardParams(params map[string][]string) map[string]string {
	standard := make(map[string]string)

	standardKeys := []string{
		"fit", "crop", "aspectratio", "withoutenlargement",
		"background", "provider", "height",
	}

	for _, key := range standardKeys {
		if values, ok := params[key]; ok && len(values) > 0 {
			standard[key] = values[0]
		}
	}

	return standard
}

// calculateRequiredWidths parses size and density values to produce a sorted
// list of unique pixel widths that need to be generated.
//
// Takes sizes ([]string) which contains the size values to parse.
// Takes densities ([]string) which contains the density multipliers to apply.
//
// Returns []int which contains the sorted, unique pixel widths to generate.
func (*AssetPipelineOrchestrator) calculateRequiredWidths(sizes, densities []string) []int {
	baseWidths := parseSizesToWidths(sizes)
	if len(baseWidths) == 0 {
		return nil
	}

	finalWidths := applyDensitiesToWidths(baseWidths, densities)
	return widthSetToSortedSlice(finalWidths)
}

// parseQuality extracts a quality value from the given slice of strings.
//
// Takes qualityValues ([]string) which contains quality settings to parse.
//
// Returns int which is the parsed quality value, or a default if parsing
// fails or the value is outside the valid range.
func (*AssetPipelineOrchestrator) parseQuality(qualityValues []string) int {
	if len(qualityValues) > 0 {
		q, err := strconv.Atoi(qualityValues[0])
		if err == nil && q >= 1 && q <= percentMax {
			return q
		}
	}
	return defaultImageQuality
}

// parseFormats determines which output formats to use.
//
// Takes formatValues ([]string) which specifies the desired output formats.
// Takes sourcePath (string) which provides the original file path for its
// extension.
//
// Returns []string which contains the output formats. When formatValues is
// empty, returns webp, avif, and the original file format.
func (*AssetPipelineOrchestrator) parseFormats(formatValues []string, sourcePath string) []string {
	if len(formatValues) > 0 {
		return formatValues
	}
	originalExt := strings.TrimPrefix(filepath.Ext(sourcePath), ".")
	return []string{"webp", "avif", originalExt}
}

// getPassthroughModifiers collects extra image parameters that are not part
// of the standard sizing or format settings, so they can be passed directly to
// the image transformer.
//
// This excludes all standard parameters and placeholder parameters, leaving
// only provider-specific image modifiers such as greyscale, blur, or sharpen.
//
// Takes params (map[string][]string) which contains all query parameters from
// the request.
//
// Returns map[string]string which contains only the provider-specific
// modifiers, with each key mapped to its first value.
func (*AssetPipelineOrchestrator) getPassthroughModifiers(params map[string][]string) map[string]string {
	modifiers := make(map[string]string)

	reservedKeys := map[string]struct{}{
		"src":       {},
		"sizes":     {},
		"densities": {},
		fieldFormat: {},
		"quality":   {},

		"width":              {},
		"height":             {},
		"fit":                {},
		"crop":               {},
		"aspectratio":        {},
		"withoutenlargement": {},
		"background":         {},
		"provider":           {},

		paramPlaceholder:        {},
		paramPlaceholderWidth:   {},
		paramPlaceholderHeight:  {},
		paramPlaceholderQuality: {},
		paramPlaceholderBlur:    {},
	}

	for key, values := range params {
		if _, isReserved := reservedKeys[key]; !isReserved && len(values) > 0 {
			modifiers[key] = values[0]
		}
	}

	return modifiers
}

// videoQualityConfig holds the encoding settings for a video quality level.
type videoQualityConfig struct {
	// resolution specifies the video dimensions in "WIDTHxHEIGHT" format,
	// for example "1920x1080".
	resolution string

	// bitrate specifies the video encoding bit rate, for example "5000k".
	bitrate string

	// bandwidth is the target bitrate in bits per second, for example 5000000.
	bandwidth int
}

// generateVideoProfiles handles piko:video assets by parsing quality parameters
// and generating NamedProfiles for every required HLS variant.
//
// Takes asset (*annotator_dto.FinalAssetDependency) which provides the asset
// metadata and transformation parameters to process.
//
// Returns []registry_dto.NamedProfile which contains all generated video
// variant profiles.
func (p *AssetPipelineOrchestrator) generateVideoProfiles(
	asset *annotator_dto.FinalAssetDependency,
) []registry_dto.NamedProfile {
	params := asset.TransformationParams

	qualities := p.parseVideoQualities(params)
	segmentDuration := p.parseSegmentDuration(params)

	return p.buildVideoProfiles(qualities, segmentDuration)
}

// parseVideoQualities extracts the quality levels from video parameters.
//
// Takes params (map[string][]string) which contains the video parameters.
//
// Returns []string which contains the quality levels to generate.
func (p *AssetPipelineOrchestrator) parseVideoQualities(
	params map[string][]string,
) []string {
	if qualityParams, ok := params["qualities"]; ok && len(qualityParams) > 0 {
		qualities := strings.Split(qualityParams[0], ",")
		result := make([]string, 0, len(qualities))
		for _, q := range qualities {
			q = strings.TrimSpace(q)
			if _, ok := videoQualityConfigs[q]; ok {
				result = append(result, q)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	if p.assetsConfig != nil && len(p.assetsConfig.Video.DefaultQualities) > 0 {
		return p.assetsConfig.Video.DefaultQualities
	}
	return fallbackVideoQualities
}

// parseSegmentDuration gets the HLS segment duration from video parameters.
//
// Takes params (map[string][]string) which holds the video parameters.
//
// Returns int which is the segment duration in seconds, using the config
// default or a fallback value if not specified.
func (p *AssetPipelineOrchestrator) parseSegmentDuration(
	params map[string][]string,
) int {
	if durationParams, ok := params["segment-duration"]; ok && len(durationParams) > 0 {
		if duration, err := strconv.Atoi(durationParams[0]); err == nil && duration > 0 {
			return duration
		}
	}

	if p.assetsConfig != nil && p.assetsConfig.Video.DefaultSegmentDuration > 0 {
		return p.assetsConfig.Video.DefaultSegmentDuration
	}
	return fallbackVideoSegmentDuration
}

// buildVideoProfiles creates named profiles for each video quality level.
//
// Takes qualities ([]string) which contains the quality levels to create.
// Takes segmentDuration (int) which sets the HLS segment length in seconds.
//
// Returns []registry_dto.NamedProfile which holds the HLS encoding profiles.
func (*AssetPipelineOrchestrator) buildVideoProfiles(
	qualities []string,
	segmentDuration int,
) []registry_dto.NamedProfile {
	profiles := make([]registry_dto.NamedProfile, 0, len(qualities))

	for _, quality := range qualities {
		qualityConfig, ok := videoQualityConfigs[quality]
		if !ok {
			continue
		}

		profileName := videoProfileNamePrefix + "_" + quality

		profiles = append(profiles, makeProfile(
			profileName,
			registry_dto.PriorityNeed,
			videoCapabilityName.String(),
			assetDependencySource,
			map[string]string{
				"Type":     videoVariantTypeTag,
				"quality":  quality,
				"mimeType": "application/x-mpegURL",
			},
			map[string]string{
				"resolution":       qualityConfig.resolution,
				"bitrate":          qualityConfig.bitrate,
				"segment_duration": strconv.Itoa(segmentDuration),
			},
		))
	}

	return profiles
}

// createVariantProfile builds a single image variant profile for the given
// width and format.
//
// Takes width (int) which sets the target image width in pixels.
// Takes format (string) which sets the output image format.
// Takes imageConfig (imageProfileConfig) which provides the profile settings.
//
// Returns registry_dto.NamedProfile which contains the configured variant
// profile ready for registration.
func createVariantProfile(width int, format string, imageConfig imageProfileConfig) registry_dto.NamedProfile {
	profileName := fmt.Sprintf("%s_w%d_%s", imageProfileNamePrefix, width, format)

	capabilityParams := buildVariantParams(width, format, imageConfig)

	var resultingTags registry_dto.Tags
	resultingTags.SetByName("type", imageVariantTypeTag)
	resultingTags.SetByName("storageBackendId", defaultStorageBackendID)
	resultingTags.SetByName("fileExtension", defaultImageFileExtension)

	var deps registry_dto.Dependencies
	deps.Add(assetDependencySource)

	return registry_dto.NamedProfile{
		Name: profileName,
		Profile: registry_dto.DesiredProfile{
			CapabilityName: string(imageCapabilityName),
			Params:         capabilityParams,
			DependsOn:      deps,
			Priority:       defaultImagePriority,
			ResultingTags:  resultingTags,
		},
	}
}

// buildVariantParams builds the parameters for an image variant.
//
// Takes width (int) which sets the target width in pixels.
// Takes format (string) which sets the output image format.
// Takes imageConfig (imageProfileConfig) which provides
// quality and extra settings.
//
// Returns registry_dto.ProfileParams which holds the assembled parameters.
func buildVariantParams(width int, format string, imageConfig imageProfileConfig) registry_dto.ProfileParams {
	var params registry_dto.ProfileParams
	params.SetByName("width", strconv.Itoa(width))
	params.SetByName(fieldFormat, format)
	params.SetByName("quality", strconv.Itoa(imageConfig.quality))

	for k, v := range imageConfig.standardParams {
		params.SetByName(k, v)
	}
	for k, v := range imageConfig.modifiers {
		params.SetByName(k, v)
	}
	return params
}

// parseSizesToWidths converts size specifications to a set of pixel widths.
//
// Takes sizes ([]string) which contains the size specifications to parse.
//
// Returns map[int]struct{} which contains the unique pixel widths from the
// size specifications.
func parseSizesToWidths(sizes []string) map[int]struct{} {
	widthSet := make(map[int]struct{})
	for _, size := range sizes {
		sizeValue := extractSizeValue(size)
		addWidthsForSizeValue(sizeValue, widthSet)
	}
	return widthSet
}

// extractSizeValue gets the size value from a string that may have a prefix.
// For example, "sm:50vw" returns "50vw", and "400px" returns "400px".
//
// Takes size (string) which is the size string, possibly with a breakpoint
// prefix separated by a colon.
//
// Returns string which is the size value with any whitespace removed.
func extractSizeValue(size string) string {
	parts := strings.Split(size, ":")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(parts[0])
}

// addWidthsForSizeValue parses a size value and adds the resulting widths to
// the set.
//
// Takes sizeValue (string) which is a size ending in "px" or "vw".
// Takes widthSet (map[int]struct{}) which collects the resulting widths.
func addWidthsForSizeValue(sizeValue string, widthSet map[int]struct{}) {
	if pxValue, found := strings.CutSuffix(sizeValue, "px"); found {
		addPixelWidth(pxValue, widthSet)
		return
	}
	if vwValue, found := strings.CutSuffix(sizeValue, "vw"); found {
		addViewportWidths(vwValue, widthSet)
	}
}

// addPixelWidth parses a pixel width value and adds it to the width set.
//
// Takes pxValue (string) which is the pixel width to parse.
// Takes widthSet (map[int]struct{}) which collects valid pixel widths.
func addPixelWidth(pxValue string, widthSet map[int]struct{}) {
	width, err := strconv.Atoi(pxValue)
	if err == nil && width > 0 {
		widthSet[width] = struct{}{}
	}
}

// addViewportWidths calculates pixel widths from a viewport width percentage
// across standard screen widths.
//
// Takes vwValue (string) which is the viewport width as a percentage.
// Takes widthSet (map[int]struct{}) which collects the calculated widths.
func addViewportWidths(vwValue string, widthSet map[int]struct{}) {
	vw, err := strconv.Atoi(vwValue)
	if err != nil || vw <= 0 {
		return
	}
	for _, screenWidth := range standardViewportWidths {
		calculatedWidth := (screenWidth * vw) / percentMax
		if calculatedWidth > 0 {
			widthSet[calculatedWidth] = struct{}{}
		}
	}
}

// applyDensitiesToWidths creates width values scaled by density factors.
//
// Takes baseWidths (map[int]struct{}) which holds the starting width values.
// Takes densities ([]string) which lists density factors such as "2x" or "3x".
//
// Returns map[int]struct{} which holds all widths, including the original
// values and any new widths created by applying the density factors.
func applyDensitiesToWidths(baseWidths map[int]struct{}, densities []string) map[int]struct{} {
	finalWidthSet := make(map[int]struct{}, len(baseWidths))
	for w := range baseWidths {
		finalWidthSet[w] = struct{}{}
	}

	for _, densityString := range densities {
		multiplier, err := parseDensity(densityString)
		if err != nil || multiplier <= 1.0 {
			continue
		}
		applyDensityMultiplier(baseWidths, finalWidthSet, multiplier)
	}
	return finalWidthSet
}

// applyDensityMultiplier scales each base width by the given multiplier and
// adds the result to the final width set.
//
// Takes baseWidths (map[int]struct{}) which contains the original widths.
// Takes finalWidthSet (map[int]struct{}) which stores the scaled widths.
// Takes multiplier (float64) which is the density scale factor.
func applyDensityMultiplier(baseWidths, finalWidthSet map[int]struct{}, multiplier float64) {
	for w := range baseWidths {
		denseWidth := int(float64(w) * multiplier)
		if denseWidth > 0 {
			finalWidthSet[denseWidth] = struct{}{}
		}
	}
}

// widthSetToSortedSlice converts a width set to a sorted slice for
// consistent output.
//
// Takes widthSet (map[int]struct{}) which contains the widths to convert.
//
// Returns []int which contains the widths in ascending order, or nil if empty.
func widthSetToSortedSlice(widthSet map[int]struct{}) []int {
	if len(widthSet) == 0 {
		return nil
	}
	sortedWidths := make([]int, 0, len(widthSet))
	for w := range widthSet {
		sortedWidths = append(sortedWidths, w)
	}
	slices.Sort(sortedWidths)
	return sortedWidths
}

// parseDensity converts a density string (e.g. "x1", "2x", "3") into a float
// multiplier.
//
// Takes density (string) which is the density value to parse.
//
// Returns float64 which is the parsed density multiplier.
// Returns error when the density string cannot be parsed as a number.
func parseDensity(density string) (float64, error) {
	d := strings.ToLower(strings.TrimSpace(density))
	d = strings.TrimPrefix(d, "x")
	d = strings.TrimSuffix(d, "x")
	return strconv.ParseFloat(d, 64)
}
