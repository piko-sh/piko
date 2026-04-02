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

package annotator_domain

// Transforms responsive image elements by applying media transformation profiles and generating appropriate asset dependencies.
// Processes piko:img elements with responsive profiles, extracts transformation parameters, and registers asset dependencies for image processing.

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/config"
)

// ResponsiveVariantSpec represents a single variant to be generated.
type ResponsiveVariantSpec struct {
	// Density is the pixel density descriptor, such as "1x" or "2x".
	Density string

	// Width is the target width in pixels.
	Width int
}

// expandResponsiveAssets takes assets marked as responsive and creates
// multiple assets for each required variant (one per width and density
// pair). Assets that are not responsive pass through unchanged.
//
// Takes dependencies ([]*annotator_dto.StaticAssetDependency) which is the
// list of asset dependencies to process.
// Takes assetsConfig (*config.AssetsConfig) which provides the responsive
// image settings.
//
// Returns []*annotator_dto.StaticAssetDependency which contains the expanded
// list with responsive variants added.
func expandResponsiveAssets(
	dependencies []*annotator_dto.StaticAssetDependency,
	assetsConfig *config.AssetsConfig,
) []*annotator_dto.StaticAssetDependency {
	var expanded []*annotator_dto.StaticAssetDependency

	for _, dependency := range dependencies {
		if dependency.TransformationParams[transformKeyResponsive] != "true" {
			expanded = append(expanded, dependency)
			continue
		}

		expanded = append(expanded, expandSingleResponsiveAsset(dependency, assetsConfig)...)
	}

	return expanded
}

// expandSingleResponsiveAsset expands a single responsive asset into multiple
// size variants for different screen densities.
//
// Takes dependency (*annotator_dto.StaticAssetDependency) which is the asset to
// expand.
// Takes assetsConfig (*config.AssetsConfig) which provides density and screen
// size settings.
//
// Returns []*annotator_dto.StaticAssetDependency which contains the expanded
// variants, or a slice with only the original asset if no variants apply.
func expandSingleResponsiveAsset(dependency *annotator_dto.StaticAssetDependency, assetsConfig *config.AssetsConfig) []*annotator_dto.StaticAssetDependency {
	widthString := strings.TrimSuffix(dependency.TransformationParams["width"], "px")
	baseWidth := 0
	if parsed, err := strconv.Atoi(widthString); err == nil {
		baseWidth = parsed
	}

	densitiesString := dependency.TransformationParams[transformKeyDensities]
	var densities []string
	if densitiesString != "" {
		densities = strings.Fields(densitiesString)
	}

	variants := calculateResponsiveVariants(
		baseWidth,
		dependency.TransformationParams["sizes"],
		densities,
		assetsConfig.Image.DefaultDensities,
		assetsConfig.Image.Screens,
	)

	if len(variants) == 0 {
		return []*annotator_dto.StaticAssetDependency{dependency}
	}

	result := make([]*annotator_dto.StaticAssetDependency, 0, len(variants))
	for _, variant := range variants {
		result = append(result, createVariantDependency(dependency, variant))
	}
	return result
}

// createVariantDependency creates a new asset dependency for a specific
// responsive variant.
//
// Takes baseDep (*annotator_dto.StaticAssetDependency) which provides the
// source dependency to copy settings from.
// Takes variant (ResponsiveVariantSpec) which specifies the width and density
// for the new variant.
//
// Returns *annotator_dto.StaticAssetDependency which is a new dependency with
// the variant settings applied.
func createVariantDependency(baseDep *annotator_dto.StaticAssetDependency, variant ResponsiveVariantSpec) *annotator_dto.StaticAssetDependency {
	variantDep := &annotator_dto.StaticAssetDependency{
		SourcePath:           baseDep.SourcePath,
		AssetType:            baseDep.AssetType,
		TransformationParams: make(map[string]string, len(baseDep.TransformationParams)),
		OriginComponentPath:  baseDep.OriginComponentPath,
		Location:             baseDep.Location,
	}

	maps.Copy(variantDep.TransformationParams, baseDep.TransformationParams)

	variantDep.TransformationParams["width"] = strconv.Itoa(variant.Width)
	variantDep.TransformationParams[transformKeyDensity] = variant.Density

	delete(variantDep.TransformationParams, "_responsive")
	delete(variantDep.TransformationParams, "densities")
	delete(variantDep.TransformationParams, "sizes")

	return variantDep
}

// calculateResponsiveVariants generates all width and density combinations
// for a responsive image, returning a deduplicated list of variants.
//
// Takes baseWidth (int) which specifies the explicit width attribute value.
// Takes sizesString (string) which contains the sizes attribute to parse.
// Takes densities ([]string) which lists the density multipliers to use.
// Takes defaultDensities ([]string) which provides fallback densities.
// Takes screens (map[string]int) which maps screen names to breakpoint widths.
//
// Returns []ResponsiveVariantSpec which contains the sorted, deduplicated
// variant specifications.
func calculateResponsiveVariants(
	baseWidth int,
	sizesString string,
	densities []string,
	defaultDensities []string,
	screens map[string]int,
) []ResponsiveVariantSpec {
	variantsMap := make(map[string]ResponsiveVariantSpec)

	activeDensities := densities
	if len(activeDensities) == 0 {
		activeDensities = defaultDensities
	}
	if len(activeDensities) == 0 {
		activeDensities = []string{"x1"}
	}

	var baseWidths []int
	if sizesString != "" {
		baseWidths = parseSizes(sizesString, screens)
	}

	if len(baseWidths) == 0 && baseWidth > 0 {
		baseWidths = []int{baseWidth}
	}

	if len(baseWidths) == 0 {
		baseWidths = defaultResponsiveBreakpoints
	}

	for _, width := range baseWidths {
		for _, density := range activeDensities {
			densityMultiplier := parseDensity(density)
			actualWidth := int(float64(width) * densityMultiplier)

			key := fmt.Sprintf("%d_%s", actualWidth, density)
			variantsMap[key] = ResponsiveVariantSpec{
				Width:   actualWidth,
				Density: density,
			}
		}
	}

	variants := make([]ResponsiveVariantSpec, 0, len(variantsMap))
	for _, variant := range variantsMap {
		variants = append(variants, variant)
	}

	slices.SortFunc(variants, func(a, b ResponsiveVariantSpec) int {
		return cmp.Or(
			cmp.Compare(a.Width, b.Width),
			cmp.Compare(a.Density, b.Density),
		)
	})

	return variants
}

// parseDensity converts a density string to a float64 value.
//
// It handles formats like "x1", "2x", or "x2" by removing the "x" prefix or
// suffix and parsing the number that remains.
//
// Takes density (string) which is the density value to parse.
//
// Returns float64 which is the parsed value, or 1.0 if the input is empty or
// cannot be parsed.
func parseDensity(density string) float64 {
	density = strings.ToLower(strings.TrimSpace(density))
	density = strings.TrimPrefix(density, "x")
	density = strings.TrimSuffix(density, "x")

	multiplier, err := strconv.ParseFloat(density, 64)
	if err != nil || multiplier <= 0 {
		return 1.0
	}

	return multiplier
}

// parseSizes reads a sizes attribute string and returns the pixel widths.
// It finds widths from breakpoints and viewport sizes, removes duplicates,
// and sorts the results.
//
// Input example: "100vw sm:50vw md:800px"
//
// Takes sizesString (string) which contains the sizes attribute to parse.
// Takes screens (map[string]int) which maps breakpoint names to pixel widths.
//
// Returns []int which is a sorted list of unique pixel widths.
func parseSizes(sizesString string, screens map[string]int) []int {
	if sizesString == "" {
		return nil
	}

	widthsMap := make(map[int]struct{})

	for part := range strings.FieldsSeq(sizesString) {
		if strings.Contains(part, ":") {
			segments := strings.SplitN(part, ":", 2)
			if len(segments) != 2 {
				continue
			}

			breakpoint := segments[0]
			sizeValue := segments[1]

			width := extractWidth(sizeValue, screens, breakpoint)
			if width > 0 {
				widthsMap[width] = struct{}{}
			}
		} else {
			width := extractWidth(part, screens, "")
			if width > 0 {
				widthsMap[width] = struct{}{}
			}
		}
	}

	widths := make([]int, 0, len(widthsMap))
	for w := range widthsMap {
		widths = append(widths, w)
	}
	slices.Sort(widths)

	return widths
}

// extractWidth parses a size value and returns the width in pixels. It handles
// pixel values like "800px", viewport values like "100vw", and percentage
// values like "50%".
//
// Takes sizeValue (string) which is the size value to parse.
// Takes screens (map[string]int) which maps breakpoint names to pixel widths.
// Takes breakpoint (string) which is the current screen size name.
//
// Returns int which is the width in pixels, or 0 if the value cannot be
// parsed.
func extractWidth(sizeValue string, screens map[string]int, breakpoint string) int {
	sizeValue = strings.TrimSpace(sizeValue)

	if width := extractPixelWidth(sizeValue); width > 0 {
		return width
	}
	if width := extractViewportWidth(sizeValue, screens, breakpoint); width > 0 {
		return width
	}
	if width := extractPercentageWidth(sizeValue, screens, breakpoint); width > 0 {
		return width
	}
	return 0
}

// extractPixelWidth parses a pixel size string and returns the width value.
//
// Takes sizeValue (string) which is the size to parse (e.g. "100px").
//
// Returns int which is the width in pixels, or 0 if the format is not valid.
func extractPixelWidth(sizeValue string) int {
	if !strings.HasSuffix(sizeValue, "px") {
		return 0
	}
	widthString := strings.TrimSuffix(sizeValue, "px")
	width, err := strconv.Atoi(widthString)
	if err != nil || width <= 0 {
		return 0
	}
	return width
}

// extractViewportWidth parses a viewport width value and returns the width in
// pixels.
//
// Takes sizeValue (string) which is the size value ending in "vw".
// Takes screens (map[string]int) which maps breakpoint names to pixel widths.
// Takes breakpoint (string) which is the name of the target screen size.
//
// Returns int which is the width in pixels, or 0 if the value is not valid.
func extractViewportWidth(sizeValue string, screens map[string]int, breakpoint string) int {
	if !strings.HasSuffix(sizeValue, "vw") {
		return 0
	}
	percentString := strings.TrimSuffix(sizeValue, "vw")
	percent, err := strconv.ParseFloat(percentString, 64)
	if err != nil {
		return 0
	}
	viewportWidth := getViewportWidth(screens, breakpoint)
	return int(float64(viewportWidth) * percent / 100.0)
}

// extractPercentageWidth converts a percentage value to a pixel width based on
// the viewport size for the given breakpoint.
//
// Takes sizeValue (string) which is the percentage value (e.g. "50%").
// Takes screens (map[string]int) which maps breakpoint names to pixel widths.
// Takes breakpoint (string) which is the name of the target screen size.
//
// Returns int which is the width in pixels, or 0 if the value is not a valid
// percentage.
func extractPercentageWidth(sizeValue string, screens map[string]int, breakpoint string) int {
	if !strings.HasSuffix(sizeValue, "%") {
		return 0
	}
	percentString := strings.TrimSuffix(sizeValue, "%")
	percent, err := strconv.ParseFloat(percentString, 64)
	if err != nil {
		return 0
	}
	viewportWidth := getViewportWidth(screens, breakpoint)
	return int(float64(viewportWidth) * percent / 100.0)
}

// getViewportWidth returns the viewport width for a given breakpoint.
//
// Takes screens (map[string]int) which maps breakpoint names to pixel widths.
// Takes breakpoint (string) which specifies the screen size to look up.
//
// Returns int which is the viewport width, or defaultViewportWidth if the
// breakpoint is empty or not found.
func getViewportWidth(screens map[string]int, breakpoint string) int {
	if breakpoint == "" {
		return defaultViewportWidth
	}
	if bpWidth, ok := screens[breakpoint]; ok {
		return bpWidth
	}
	return defaultViewportWidth
}
