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
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/seo/seo_dto"
)

const (
	// pagesDirPrefix is the standard prefix for page source files without a leading slash.
	pagesDirPrefix = "pages/"

	// pagesDirPrefixLength is the number of characters in "pages/" for slicing.
	pagesDirPrefixLength = 6

	// pagesDirPrefixWithSlash is the standard prefix for page source files with a leading
	// slash.
	pagesDirPrefixWithSlash = "/pages/"

	// pagesDirPrefixWithSlashLength is the length of "/pages/" for substring operations.
	pagesDirPrefixWithSlashLength = 7
)

// ProjectViewTranslator converts annotator domain objects into SEO domain objects. It
// acts as an anti-corruption layer between the SEO and annotator hexagons.
type ProjectViewTranslator struct{}

// NewProjectViewTranslator creates a new translator for project data.
//
// Returns *ProjectViewTranslator which is ready to use.
func NewProjectViewTranslator() *ProjectViewTranslator {
	return &ProjectViewTranslator{}
}

// Translate converts a ProjectAnnotationResult into a ProjectView for the SEO hexagon.
// This translation process extracts only the information relevant to SEO generation.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides the annotated
// project data to convert.
//
// Returns *seo_dto.ProjectView which contains the SEO-relevant project data.
func (t *ProjectViewTranslator) Translate(result *annotator_dto.ProjectAnnotationResult) *seo_dto.ProjectView {
	if result == nil || result.VirtualModule == nil {
		return &seo_dto.ProjectView{
			Components: []seo_dto.ComponentView{},
		}
	}

	components := make([]seo_dto.ComponentView, 0, len(result.VirtualModule.ComponentsByHash))
	for hash, component := range result.VirtualModule.ComponentsByHash {
		components = append(components, t.componentViews(hash, component, result)...)
	}

	return &seo_dto.ProjectView{
		Components: components,
	}
}

// componentViews builds the SEO component views contributed by one annotated component.
//
// A page yields a single view, or one per resolved instance when it is a collection
// template; non-page components yield nothing.
//
// Takes hash (string) which is the component's stable hashed name.
// Takes component (*annotator_dto.VirtualComponent) which is the annotated component.
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides per-component
// annotation data such as opted-in sitemap images.
//
// Returns []seo_dto.ComponentView which are the views this component contributes.
func (t *ProjectViewTranslator) componentViews(
	hash string,
	component *annotator_dto.VirtualComponent,
	result *annotator_dto.ProjectAnnotationResult,
) []seo_dto.ComponentView {
	if !component.IsPage {
		return nil
	}

	view := t.baseComponentView(hash, component, result)

	if component.Source != nil && component.Source.HasCollection && len(component.VirtualInstances) > 0 {
		return t.expandCollectionInstances(view, component)
	}

	return []seo_dto.ComponentView{view}
}

// baseComponentView builds the page-level view shared by a static page and, for a
// collection template, by every expanded instance.
//
// Takes hash (string) which is the component's stable hashed name.
// Takes component (*annotator_dto.VirtualComponent) which is the annotated component.
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides per-component
// annotation data.
//
// Returns seo_dto.ComponentView which is the shared base view.
func (t *ProjectViewTranslator) baseComponentView(
	hash string,
	component *annotator_dto.VirtualComponent,
	result *annotator_dto.ProjectAnnotationResult,
) seo_dto.ComponentView {
	view := seo_dto.ComponentView{
		HashedName:         hash,
		IsPage:             component.IsPage,
		IsPublic:           component.IsPublic,
		OriginalSourcePath: "",
		RoutePattern:       "",
		SupportedLocales:   []string{},
		SEO:                seo_dto.PageSEOMetadata{},
	}

	if component.Source != nil {
		view.OriginalSourcePath = component.Source.SourcePath
		view.RoutePattern = t.deriveRouteFromPath(component.Source.SourcePath)

		if len(component.Source.LocalTranslations) > 0 {
			view.SupportedLocales = extractSupportedLocales(component.Source.LocalTranslations)
			view.SEO.SupportedLocales = view.SupportedLocales
		}
	}

	if annotation := result.ComponentResults[hash]; annotation != nil {
		view.SEO.ImageURLs = annotation.SitemapImageURLs
	}

	return view
}

// expandCollectionInstances turns one collection-templated page component into one
// ComponentView per resolved collection item.
//
// Takes base (seo_dto.ComponentView) which is the shared page-level view to clone per
// item.
// Takes component (*annotator_dto.VirtualComponent) which supplies the resolved
// instances.
//
// Returns []seo_dto.ComponentView which are the per-instance expanded views.
func (*ProjectViewTranslator) expandCollectionInstances(
	base seo_dto.ComponentView,
	component *annotator_dto.VirtualComponent,
) []seo_dto.ComponentView {
	param := component.Source.CollectionParamName
	if param == "" {
		param = "slug"
	}
	placeholder := "{" + param + "}"

	expanded := make([]seo_dto.ComponentView, 0, len(component.VirtualInstances))
	for _, instance := range component.VirtualInstances {
		if instance.Slug == "" {
			continue
		}

		view := base
		view.RoutePattern = strings.ReplaceAll(base.RoutePattern, placeholder, escapePathSegments(instance.Slug))
		if lastMod := extractInstanceLastModified(instance.InitialProps); lastMod != nil {
			view.SEO.LastModified = lastMod
		}

		expanded = append(expanded, view)
	}
	return expanded
}

// extractInstanceLastModified pulls a last-modified timestamp from a collection
// instance's page metadata (InitialProps["page"]), preferring UpdatedAt then PublishedAt.
//
// Takes initialProps (map[string]any) which holds the instance's page metadata.
//
// Returns *time.Time which is the parsed timestamp, or nil when none is available.
func extractInstanceLastModified(initialProps map[string]any) *time.Time {
	pageData, ok := initialProps["page"].(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range []string{collection_dto.MetaKeyUpdatedAt, collection_dto.MetaKeyPublishedAt} {
		switch v := pageData[key].(type) {
		case time.Time:
			if !v.IsZero() {
				return new(v)
			}
		case *time.Time:
			if v != nil && !v.IsZero() {
				return v
			}
		case string:
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				return &parsed
			}
		}
	}
	return nil
}

// escapePathSegments percent-encodes each slash-separated segment of a URL path so that a
// content slug containing spaces or non-ASCII characters yields a valid sitemap URL.
//
// Takes path (string) which is the URL path to encode segment by segment.
//
// Returns string which is the path with each segment percent-encoded.
func escapePathSegments(path string) string {
	if path == "" {
		return path
	}

	segments := strings.Split(path, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	return strings.Join(segments, "/")
}

// deriveRouteFromPath converts a source file path to a URL route. For example,
// "pages/about.pk" becomes "/about" and "pages/blog/post.pk" becomes "/blog/post".
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
// Takes translations (map[string]map[string]string) which contains the i18n translations
// keyed by locale code.
//
// Returns []string which contains the locale codes found in the translations map, or an
// empty slice if the map is empty.
func extractSupportedLocales(translations map[string]map[string]string) []string {
	if len(translations) == 0 {
		return []string{}
	}

	return slices.Collect(maps.Keys(translations))
}
