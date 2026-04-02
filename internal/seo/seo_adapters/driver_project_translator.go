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

package seo_adapters

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/seo/seo_dto"
)

const (
	// pagesDirPrefix is the standard prefix for page source files without a
	// leading slash.
	pagesDirPrefix = "pages/"

	// pagesDirPrefixLength is the number of characters in "pages/" for slicing.
	pagesDirPrefixLength = 6

	// pagesDirPrefixWithSlash is the standard prefix for page source files
	// with a leading slash.
	pagesDirPrefixWithSlash = "/pages/"

	// pagesDirPrefixWithSlashLength is the length of "/pages/" for substring
	// operations.
	pagesDirPrefixWithSlashLength = 7
)

// ProjectViewTranslator converts annotator domain objects into SEO domain
// objects. It acts as an anti-corruption layer between the SEO and annotator
// hexagons.
type ProjectViewTranslator struct{}

// NewProjectViewTranslator creates a new translator for project data.
//
// Returns *ProjectViewTranslator which is ready to use.
func NewProjectViewTranslator() *ProjectViewTranslator {
	return &ProjectViewTranslator{}
}

// Translate converts a ProjectAnnotationResult into a ProjectView for the SEO
// hexagon. This translation process extracts only the information relevant to
// SEO generation.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides the
// annotated project data to convert.
//
// Returns *seo_dto.ProjectView which contains the SEO-relevant project data.
func (t *ProjectViewTranslator) Translate(result *annotator_dto.ProjectAnnotationResult) *seo_dto.ProjectView {
	if result == nil || result.VirtualModule == nil {
		return &seo_dto.ProjectView{
			Components:         []seo_dto.ComponentView{},
			FinalAssetManifest: []seo_dto.AssetDependency{},
		}
	}

	components := make([]seo_dto.ComponentView, 0, len(result.VirtualModule.ComponentsByHash))
	for hash, component := range result.VirtualModule.ComponentsByHash {
		if !component.IsPage {
			continue
		}

		componentView := seo_dto.ComponentView{
			HashedName:         hash,
			IsPage:             component.IsPage,
			IsPublic:           component.IsPublic,
			OriginalSourcePath: "",
			RoutePattern:       "",
			SupportedLocales:   []string{},
			SEO:                seo_dto.PageSEOMetadata{},
		}

		if component.Source != nil {
			componentView.OriginalSourcePath = component.Source.SourcePath
			componentView.RoutePattern = t.deriveRouteFromPath(component.Source.SourcePath)

			if len(component.Source.LocalTranslations) > 0 {
				componentView.SupportedLocales = extractSupportedLocales(component.Source.LocalTranslations)
				componentView.SEO.SupportedLocales = componentView.SupportedLocales
			}
		}

		components = append(components, componentView)
	}

	assetManifest := make([]seo_dto.AssetDependency, 0, len(result.FinalAssetManifest))
	for _, asset := range result.FinalAssetManifest {
		assetManifest = append(assetManifest, seo_dto.AssetDependency{
			SourcePath: asset.SourcePath,
			AssetType:  asset.AssetType,
		})
	}

	return &seo_dto.ProjectView{
		Components:         components,
		FinalAssetManifest: assetManifest,
	}
}

// deriveRouteFromPath converts a source file path to a URL route.
// For example, "pages/about.pk" becomes "/about" and
// "pages/blog/post.pk" becomes "/blog/post".
//
// Takes sourcePath (string) which is the path to a template file.
//
// Returns string which is the URL route for the given file path.
func (*ProjectViewTranslator) deriveRouteFromPath(sourcePath string) string {
	basePath := strings.TrimSuffix(sourcePath, ".pk")

	basePath = filepath.ToSlash(basePath)
	if index := strings.Index(basePath, pagesDirPrefix); index != -1 {
		basePath = basePath[index+pagesDirPrefixLength:]
	} else if index := strings.Index(basePath, pagesDirPrefixWithSlash); index != -1 {
		basePath = basePath[index+pagesDirPrefixWithSlashLength:]
	}

	if trimmed, found := strings.CutSuffix(basePath, "/index"); found {
		basePath = trimmed
	} else if basePath == "index" {
		basePath = ""
	}

	if basePath == "" {
		return "/"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	return basePath
}

// extractSupportedLocales extracts locale codes from the i18n translations map.
//
// Takes translations (map[string]map[string]string) which contains the i18n
// translations keyed by locale code.
//
// Returns []string which contains the locale codes found in the translations
// map, or an empty slice if the map is empty.
func extractSupportedLocales(translations map[string]map[string]string) []string {
	if len(translations) == 0 {
		return []string{}
	}

	return slices.Collect(maps.Keys(translations))
}
