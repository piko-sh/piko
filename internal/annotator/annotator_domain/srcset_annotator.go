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

// Annotates image elements with srcset attributes for responsive images by generating density-based variants.
// Processes piko:img elements, applies transformation profiles, and creates srcset markup for optimised image delivery across devices.

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// AnnotateSrcsetForWebImages performs a final pass on the AST after the asset
// manifest is built. For each <piko:img> tag with responsive attributes, it
// finds all generated variants in the manifest and attaches srcset metadata to
// the AST node for use by the code generator.
//
// Takes ctx (context.Context) which carries the logger.
// Takes templateAST (*ast_domain.TemplateAST) which is the parsed template to
// annotate with srcset metadata.
// Takes assetDependencies ([]*annotator_dto.StaticAssetDependency) which
// contains the static asset manifest with generated image variants.
// Takes pathsConfig (AnnotatorPathsConfig) which provides path settings for
// URL generation.
func AnnotateSrcsetForWebImages(
	ctx context.Context,
	templateAST *ast_domain.TemplateAST,
	assetDependencies []*annotator_dto.StaticAssetDependency,
	pathsConfig AnnotatorPathsConfig,
) {
	_, l := logger_domain.From(ctx, log)
	if templateAST == nil || len(assetDependencies) == 0 {
		l.Trace("AnnotateSrcsetForWebImages: early return",
			logger_domain.Bool("ast_nil", templateAST == nil),
			logger_domain.Int("deps_count", len(assetDependencies)))
		return
	}

	l.Internal("AnnotateSrcsetForWebImages: starting",
		logger_domain.Int("total_deps", len(assetDependencies)))

	variantsBySource := make(map[string][]*annotator_dto.StaticAssetDependency)
	for _, dependency := range assetDependencies {
		if dependency.AssetType == "img" && dependency.TransformationParams[transformKeyDensity] != "" {
			variantsBySource[dependency.SourcePath] = append(variantsBySource[dependency.SourcePath], dependency)
			l.Trace("AnnotateSrcsetForWebImages: found density variant",
				logger_domain.String("src", dependency.SourcePath),
				logger_domain.String("density", dependency.TransformationParams[transformKeyDensity]))
		}
	}

	l.Internal("AnnotateSrcsetForWebImages: variants by source",
		logger_domain.Int("unique_sources", len(variantsBySource)))

	templateAST.Walk(func(node *ast_domain.TemplateNode) bool {
		return processPikoImgNode(ctx, node, variantsBySource, pathsConfig)
	})
}

// processPikoImgNode handles a single piko:img node by adding srcset data when
// there are image variants for its source.
//
// Takes ctx (context.Context) which carries the logger.
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes variantsBySource (map[string][]*annotator_dto.StaticAssetDependency)
// which maps source paths to their image variants.
// Takes pathsConfig (AnnotatorPathsConfig) which provides path settings.
//
// Returns bool which shows whether processing should continue.
func processPikoImgNode(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	variantsBySource map[string][]*annotator_dto.StaticAssetDependency,
	pathsConfig AnnotatorPathsConfig,
) bool {
	if node.NodeType != ast_domain.NodeElement || (node.TagName != "piko:img" && node.TagName != "piko:picture") {
		return true
	}

	src, hasSrc := node.GetAttribute("src")
	if !hasSrc || src == "" {
		return true
	}

	variants, hasVariants := variantsBySource[src]
	if !hasVariants || len(variants) == 0 {
		return true
	}

	srcsetMetadata := buildSrcsetMetadata(variants, pathsConfig)
	if len(srcsetMetadata) == 0 {
		return true
	}

	if node.GoAnnotations == nil {
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
	}
	node.GoAnnotations.Srcset = srcsetMetadata

	_, l := logger_domain.From(ctx, log)
	l.Trace("Annotated piko:img with srcset",
		logger_domain.String("src", src),
		logger_domain.Int("variant_count", len(srcsetMetadata)))

	return true
}

// buildSrcsetMetadata constructs srcset metadata from asset dependencies.
// It extracts width, density, and transformation parameters for each variant.
//
// Takes variants ([]*annotator_dto.StaticAssetDependency) which provides the
// asset variants to process.
// Takes pathsConfig (AnnotatorPathsConfig) which specifies the path settings for
// URL generation.
//
// Returns []ast_domain.ResponsiveVariantMetadata which contains the processed
// metadata sorted by width, then by density.
func buildSrcsetMetadata(
	variants []*annotator_dto.StaticAssetDependency,
	pathsConfig AnnotatorPathsConfig,
) []ast_domain.ResponsiveVariantMetadata {
	metadata := make([]ast_domain.ResponsiveVariantMetadata, 0, len(variants))

	for _, variant := range variants {
		widthString := variant.TransformationParams["width"]
		width := 0
		if widthString != "" {
			widthString = strings.TrimSuffix(widthString, "px")
			if parsed, err := strconv.Atoi(widthString); err == nil {
				width = parsed
			}
		}

		heightString := variant.TransformationParams["height"]
		height := 0
		if heightString != "" {
			heightString = strings.TrimSuffix(heightString, "px")
			if parsed, err := strconv.Atoi(heightString); err == nil {
				height = parsed
			}
		}

		density := cmp.Or(variant.TransformationParams[transformKeyDensity], "x1")

		variantKey := generateVariantKey(variant)

		url := generateAssetURL(variant, pathsConfig)

		metadata = append(metadata, ast_domain.ResponsiveVariantMetadata{
			Width:      width,
			Height:     height,
			Density:    density,
			VariantKey: variantKey,
			URL:        url,
		})
	}

	slices.SortFunc(metadata, func(a, b ast_domain.ResponsiveVariantMetadata) int {
		return cmp.Or(
			cmp.Compare(a.Width, b.Width),
			cmp.Compare(a.Density, b.Density),
		)
	})

	return metadata
}

// generateVariantKey creates a unique key for a variant that matches the
// registry's naming pattern. This key is used in the URL query parameter to
// request a specific variant.
//
// Takes variant (*annotator_dto.StaticAssetDependency) which contains the
// width and format settings for the image variant.
//
// Returns string which is the variant key in format "image_w{width}_{format}"
// matching the asset_pipeline.go profile naming pattern.
func generateVariantKey(variant *annotator_dto.StaticAssetDependency) string {
	width := variant.TransformationParams["width"]
	format := cmp.Or(variant.TransformationParams["format"], "webp")
	width = strings.TrimSuffix(width, "px")

	return fmt.Sprintf("image_w%s_%s", width, format)
}

// generateAssetURL creates the URL for serving a specific variant of an asset.
// The URL format is: {servePath}/{artefactID}?v={profileName}, so the asset
// server can look up the artefact and serve the correct variant.
//
// Takes variant (*annotator_dto.StaticAssetDependency) which specifies the
// asset variant to create a URL for.
// Takes pathsConfig (AnnotatorPathsConfig) which provides the asset serving path.
//
// Returns string which is the complete URL for serving the asset variant.
func generateAssetURL(variant *annotator_dto.StaticAssetDependency, pathsConfig AnnotatorPathsConfig) string {
	assetServePath := cmp.Or(pathsConfig.ArtefactServePath, "/_piko/assets")

	artefactID := variant.SourcePath

	profileName := generateVariantKey(variant)

	return fmt.Sprintf("%s/%s?v=%s", assetServePath, artefactID, profileName)
}
