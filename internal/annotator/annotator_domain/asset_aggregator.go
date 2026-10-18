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

// Aggregates static asset dependencies from multiple page results into a single
// project-wide manifest. Merges transformation parameters, de-duplicates entries, and
// produces deterministic sorted output for build consistency.

import (
	"cmp"
	"slices"
	"strings"
	"unicode"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// aggregatedAsset is a private intermediate data structure used for efficient merging and
// de-duplication of asset transformation parameters. The inner map acts as a set to store
// unique parameter values.
type aggregatedAsset struct {
	// Params holds transformation parameters grouped by name, where each name maps to a set
	// of values. For example, "widths" maps to {"300", "600"}.
	Params map[string]map[string]struct{}

	// SourcePath is the original file path of the asset.
	SourcePath string

	// AssetType is the kind of asset, such as image or stylesheet.
	AssetType string
}

// AggregateProjectAssets creates a single, de-duplicated, and merged asset manifest for
// the entire project from all page results. Acts as the final step in asset processing
// within the annotator hexagon.
//
// Takes pageResults ([]*annotator_dto.AnnotationResult) which contains the annotation
// results from all processed pages.
//
// Returns []*annotator_dto.FinalAssetDependency which is the merged and sorted asset
// manifest ready for output.
func AggregateProjectAssets(pageResults []*annotator_dto.AnnotationResult) []*annotator_dto.FinalAssetDependency {
	aggregated := collectAndMergeDependencies(pageResults)
	finalManifest := convertToFinalManifest(aggregated)
	sortManifestForDeterminism(finalManifest)
	return finalManifest
}

// collectAndMergeDependencies groups assets by their unique key (type and source path)
// and merges their settings into a single entry per asset.
//
// Takes pageResults ([]*annotator_dto.AnnotationResult) which contains the annotation
// results to process.
//
// Returns map[string]*aggregatedAsset which maps unique asset keys to their merged asset
// data.
func collectAndMergeDependencies(pageResults []*annotator_dto.AnnotationResult) map[string]*aggregatedAsset {
	aggregated := make(map[string]*aggregatedAsset)

	for _, pageResult := range pageResults {
		if pageResult == nil || pageResult.AssetDependencies == nil {
			continue
		}

		for _, dependency := range pageResult.AssetDependencies {
			agg := getOrCreateAggregatedAsset(aggregated, dependency)
			mergeTransformationParams(agg, dependency.TransformationParams)
		}
	}

	return aggregated
}

// getOrCreateAggregatedAsset finds or creates an aggregated asset entry.
//
// Takes aggregated (map[string]*aggregatedAsset) which holds the existing aggregated
// assets, keyed by type and source path.
// Takes dependency (*annotator_dto.StaticAssetDependency) which provides the asset
// details used to find or create the entry.
//
// Returns *aggregatedAsset which is the existing or newly created entry.
func getOrCreateAggregatedAsset(
	aggregated map[string]*aggregatedAsset,
	dependency *annotator_dto.StaticAssetDependency,
) *aggregatedAsset {
	key := dependency.AssetType + ":" + dependency.SourcePath

	agg, exists := aggregated[key]
	if !exists {
		agg = &aggregatedAsset{
			SourcePath: dependency.SourcePath,
			AssetType:  dependency.AssetType,
			Params:     make(map[string]map[string]struct{}),
		}
		aggregated[key] = agg
	}

	return agg
}

var (
	// listParamSeparators maps list-valued transformation parameters to their separators.
	listParamSeparators = map[string]func(rune) bool{
		"sizes":               isCommaSeparator,
		"widths":              isCommaSeparator,
		transformKeyDensities: isCommaOrSpaceSeparator,
		"format":              isCommaSeparator,
		"formats":             isCommaSeparator,
		"qualities":           isCommaSeparator,
	}
)

// mergeTransformationParams adds transformation parameters to an aggregated asset. When a
// parameter key already exists, the new values are joined with the existing ones.
//
// Takes agg (*aggregatedAsset) which is the target asset to merge into.
// Takes params (map[string]string) which holds the parameters to add.
func mergeTransformationParams(agg *aggregatedAsset, params map[string]string) {
	for paramKey, paramValue := range params {
		if _, ok := agg.Params[paramKey]; !ok {
			agg.Params[paramKey] = make(map[string]struct{})
		}

		addParamValues(agg.Params[paramKey], paramKey, paramValue)
	}
}

// addParamValues adds each list item or a scalar parameter's whole value to the set.
//
// Takes valueSet (map[string]struct{}) which receives the values.
// Takes paramKey (string) which names the parameter.
// Takes paramValue (string) which is the parameter's value.
func addParamValues(valueSet map[string]struct{}, paramKey, paramValue string) {
	separator, isList := listParamSeparators[paramKey]
	if !isList {
		if trimmedValue := strings.TrimSpace(paramValue); trimmedValue != "" {
			valueSet[trimmedValue] = struct{}{}
		}
		return
	}
	for _, item := range strings.FieldsFunc(paramValue, separator) {
		if trimmedValue := strings.TrimSpace(item); trimmedValue != "" {
			valueSet[trimmedValue] = struct{}{}
		}
	}
}

// isCommaSeparator reports whether r separates the items of a comma-separated list.
//
// Takes r (rune) which is the character to test.
//
// Returns bool which is true for a comma.
func isCommaSeparator(r rune) bool {
	return r == ','
}

// isCommaOrSpaceSeparator reports whether r separates the items of a list written with
// commas, spaces or both, as densities are ("x1 x2" or "x1, x2").
//
// Takes r (rune) which is the character to test.
//
// Returns bool which is true for a comma or white space.
func isCommaOrSpaceSeparator(r rune) bool {
	return r == ',' || unicode.IsSpace(r)
}

// convertToFinalManifest converts grouped asset data into the final output format.
//
// Takes aggregated (map[string]*aggregatedAsset) which contains asset data grouped by
// name.
//
// Returns []*annotator_dto.FinalAssetDependency which is the list of assets ready for
// output.
func convertToFinalManifest(aggregated map[string]*aggregatedAsset) []*annotator_dto.FinalAssetDependency {
	finalManifest := make([]*annotator_dto.FinalAssetDependency, 0, len(aggregated))

	for _, agg := range aggregated {
		finalDep := convertAggregatedToFinal(agg)
		finalManifest = append(finalManifest, finalDep)
	}

	return finalManifest
}

// convertAggregatedToFinal converts an aggregated asset into its final form.
//
// Takes agg (*aggregatedAsset) which holds the collected asset data.
//
// Returns *annotator_dto.FinalAssetDependency which contains the converted asset with
// sorted parameters for consistent output.
func convertAggregatedToFinal(agg *aggregatedAsset) *annotator_dto.FinalAssetDependency {
	finalDep := &annotator_dto.FinalAssetDependency{
		SourcePath:           agg.SourcePath,
		AssetType:            agg.AssetType,
		TransformationParams: make(map[string][]string),
	}

	for paramKey, valueSet := range agg.Params {
		finalDep.TransformationParams[paramKey] = sortedValuesFromSet(valueSet)
	}

	return finalDep
}

// sortedValuesFromSet extracts all keys from a set and returns them sorted.
//
// Takes valueSet (map[string]struct{}) which is the set to extract keys from.
//
// Returns []string which contains the sorted keys.
func sortedValuesFromSet(valueSet map[string]struct{}) []string {
	values := make([]string, 0, len(valueSet))
	for v := range valueSet {
		values = append(values, v)
	}
	slices.Sort(values)
	return values
}

// sortManifestForDeterminism sorts the manifest to give the same output each time. It
// sorts first by source path, then by asset type.
//
// Takes finalManifest ([]*annotator_dto.FinalAssetDependency) which is the manifest to
// sort in place.
func sortManifestForDeterminism(finalManifest []*annotator_dto.FinalAssetDependency) {
	slices.SortFunc(finalManifest, func(a, b *annotator_dto.FinalAssetDependency) int {
		return cmp.Or(
			cmp.Compare(a.SourcePath, b.SourcePath),
			cmp.Compare(a.AssetType, b.AssetType),
		)
	})
}
