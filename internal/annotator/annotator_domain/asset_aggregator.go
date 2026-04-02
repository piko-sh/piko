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

// Aggregates static asset dependencies from multiple page results into a single project-wide manifest.
// Merges transformation parameters, de-duplicates entries, and produces deterministic sorted output for build consistency.

import (
	"cmp"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// aggregatedAsset is a private intermediate data structure used for efficient
// merging and de-duplication of asset transformation parameters. The inner map
// acts as a set to store unique parameter values.
type aggregatedAsset struct {
	// Params holds transformation parameters grouped by name, where each name maps
	// to a set of values. For example, "widths" maps to {"300", "600"}.
	Params map[string]map[string]struct{}

	// SourcePath is the original file path of the asset.
	SourcePath string

	// AssetType is the kind of asset, such as image or stylesheet.
	AssetType string
}

// AggregateProjectAssets creates a single, de-duplicated, and merged asset
// manifest for the entire project from all page results. This function is the
// final step in asset processing within the annotator hexagon.
//
// Takes pageResults ([]*annotator_dto.AnnotationResult) which contains the
// annotation results from all processed pages.
//
// Returns []*annotator_dto.FinalAssetDependency which is the merged and sorted
// asset manifest ready for output.
func AggregateProjectAssets(pageResults []*annotator_dto.AnnotationResult) []*annotator_dto.FinalAssetDependency {
	aggregated := collectAndMergeDependencies(pageResults)
	finalManifest := convertToFinalManifest(aggregated)
	sortManifestForDeterminism(finalManifest)
	return finalManifest
}

// collectAndMergeDependencies groups assets by their unique key (type and
// source path) and merges their settings into a single entry per asset.
//
// Takes pageResults ([]*annotator_dto.AnnotationResult) which contains the
// annotation results to process.
//
// Returns map[string]*aggregatedAsset which maps unique asset keys to their
// merged asset data.
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
// Takes aggregated (map[string]*aggregatedAsset) which holds the existing
// aggregated assets, keyed by type and source path.
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

// mergeTransformationParams adds transformation parameters to an aggregated
// asset. When a parameter key already exists, the new values are joined with
// the existing ones.
//
// Takes agg (*aggregatedAsset) which is the target asset to merge into.
// Takes params (map[string]string) which holds the parameters to add.
func mergeTransformationParams(agg *aggregatedAsset, params map[string]string) {
	for paramKey, paramValue := range params {
		if _, ok := agg.Params[paramKey]; !ok {
			agg.Params[paramKey] = make(map[string]struct{})
		}

		addCommaSeparatedValues(agg.Params[paramKey], paramValue)
	}
}

// addCommaSeparatedValues splits a comma-separated string and adds each
// trimmed, non-empty value to the given set.
//
// Takes valueSet (map[string]struct{}) which receives the parsed values.
// Takes paramValue (string) which contains the comma-separated values to parse.
func addCommaSeparatedValues(valueSet map[string]struct{}, paramValue string) {
	for v := range strings.SplitSeq(paramValue, ",") {
		trimmedValue := strings.TrimSpace(v)
		if trimmedValue != "" {
			valueSet[trimmedValue] = struct{}{}
		}
	}
}

// convertToFinalManifest converts grouped asset data into the final output
// format.
//
// Takes aggregated (map[string]*aggregatedAsset) which contains asset data
// grouped by name.
//
// Returns []*annotator_dto.FinalAssetDependency which is the list of assets
// ready for output.
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
// Returns *annotator_dto.FinalAssetDependency which contains the converted
// asset with sorted parameters for consistent output.
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

// sortManifestForDeterminism sorts the manifest to give the same output each
// time. It sorts first by source path, then by asset type.
//
// Takes finalManifest ([]*annotator_dto.FinalAssetDependency) which is the
// manifest to sort in place.
func sortManifestForDeterminism(finalManifest []*annotator_dto.FinalAssetDependency) {
	slices.SortFunc(finalManifest, func(a, b *annotator_dto.FinalAssetDependency) int {
		return cmp.Or(
			cmp.Compare(a.SourcePath, b.SourcePath),
			cmp.Compare(a.AssetType, b.AssetType),
		)
	})
}
