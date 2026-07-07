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
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/seo/seo_dto"
	"piko.sh/piko/wdk/safeconv"
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
type ProjectViewTranslator struct {
	// i18nLocales is the full set of configured locale codes. A page that declares a
	// SupportedLocales() function opts into locale routing for this whole set (the annotator
	// records only that the function exists, not its return value), so the sitemap's
	// hreflang matches the routes the manifest fans out.
	i18nLocales []string
}

// NewProjectViewTranslator creates a new translator for project data.
//
// Takes i18nLocales ([]string) which is the full configured locale set, used to derive
// hreflang alternates for pages that declare a SupportedLocales() function.
//
// Returns *ProjectViewTranslator which is ready to use.
func NewProjectViewTranslator(i18nLocales []string) *ProjectViewTranslator {
	return &ProjectViewTranslator{i18nLocales: i18nLocales}
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
		return t.expandCollectionInstances(&view, component)
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
		IsAuthGated:        component.Source != nil && component.Source.Script != nil && component.Source.Script.HasAuthPolicy,
		OriginalSourcePath: "",
		RoutePattern:       "",
		SupportedLocales:   []string{},
		SEO:                seo_dto.PageSEOMetadata{},
	}

	if component.Source != nil {
		view.OriginalSourcePath = component.Source.SourcePath
		view.RoutePattern = t.deriveRouteFromPath(component.Source.SourcePath)

		if locales := t.supportedLocales(component.Source); len(locales) > 0 {
			view.SupportedLocales = locales
			view.SEO.SupportedLocales = locales
		}

		if !component.Source.HasCollection && component.Source.RouteSourceName != "" {
			view.RouteSourceName = component.Source.RouteSourceName
			view.RouteSourceParamName = component.Source.RouteSourceParamName
		}

		applySitemapOverrides(&view.SEO, component.Source)
	}

	if annotation := result.ComponentResults[hash]; annotation != nil {
		view.SEO.ImageURLs = annotation.SitemapImageURLs
	}

	return view
}

// applySitemapOverrides copies a page's declarative sitemap overrides (the p-noindex,
// p-sitemap-priority, p-sitemap-changefreq, and p-canonical template attributes) onto its
// SEO metadata. An invalid or non-finite priority string is ignored rather than failing
// the build.
//
// Takes seo (*seo_dto.PageSEOMetadata) which receives the overrides.
// Takes source (*annotator_dto.ParsedComponent) which carries the parsed attributes.
func applySitemapOverrides(seo *seo_dto.PageSEOMetadata, source *annotator_dto.ParsedComponent) {
	if source.SitemapNoindex {
		seo.RobotsRule = "noindex"
	}
	if source.SitemapChangeFrequency != "" {
		seo.ChangeFrequency = source.SitemapChangeFrequency
	}
	if source.SitemapCanonical != "" {
		seo.Canonical = source.SitemapCanonical
	}
	if source.SitemapPriority != "" {
		if priority, err := strconv.ParseFloat(source.SitemapPriority, 32); err == nil &&
			!math.IsNaN(priority) && !math.IsInf(priority, 0) {
			p := float32(priority)
			seo.Priority = &p
		}
	}
}

// supportedLocales returns the locales a page is available in, for hreflang generation.
//
// A page opts into locale routing only by declaring a SupportedLocales() function, which
// enrols it in the full configured locale set; the annotator records only that the
// function exists, matching how the manifest fans out per-locale routes. A page without
// that function is single-locale, so an empty slice is returned.
//
// Takes source (*annotator_dto.ParsedComponent) which is the page's parsed component.
//
// Returns []string which are the locale codes the page supports.
func (t *ProjectViewTranslator) supportedLocales(source *annotator_dto.ParsedComponent) []string {
	if source.Script != nil && source.Script.HasSupportedLocales && len(t.i18nLocales) > 0 {
		return slices.Clone(t.i18nLocales)
	}

	return nil
}

// expandCollectionInstances turns one collection-templated page component into one
// ComponentView per resolved collection item.
//
// Takes base (*seo_dto.ComponentView) which is the shared page-level view to clone per
// item.
// Takes component (*annotator_dto.VirtualComponent) which supplies the resolved
// instances.
//
// Returns []seo_dto.ComponentView which are the per-instance expanded views.
func (*ProjectViewTranslator) expandCollectionInstances(
	base *seo_dto.ComponentView,
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

		view := *base

		view.SEO.ImageURLs = slices.Clone(base.SEO.ImageURLs)
		view.RoutePattern = strings.ReplaceAll(base.RoutePattern, placeholder, seo_dto.EscapePathSegments(instance.Slug))
		if lastMod := extractInstanceLastModified(instance.InitialProps); lastMod != nil {
			view.SEO.LastModified = lastMod
		}
		applyInstanceSitemapOverrides(&view.SEO, instance.InitialProps)
		applyInstanceRichMedia(&view.SEO, instance.InitialProps)

		expanded = append(expanded, view)
	}
	return expanded
}

// applyInstanceSitemapOverrides reads per-item sitemap overrides from a collection
// instance's frontmatter (InitialProps["page"]), the conventional "noindex", "priority",
// "changefreq", and "canonical" keys, so a single markdown item can drop itself from the
// sitemap or tune its priority without affecting its siblings. Values with the wrong type
// are ignored.
//
// Takes seo (*seo_dto.PageSEOMetadata) which receives the overrides.
// Takes initialProps (map[string]any) which holds the instance's page metadata.
func applyInstanceSitemapOverrides(seo *seo_dto.PageSEOMetadata, initialProps map[string]any) {
	pageData, ok := initialProps["page"].(map[string]any)
	if !ok {
		return
	}

	if noindex, ok := pageData["noindex"].(bool); ok && noindex {
		seo.RobotsRule = "noindex"
	}
	if changeFreq, ok := pageData["changefreq"].(string); ok && changeFreq != "" {
		seo.ChangeFrequency = changeFreq
	}
	if canonical, ok := pageData["canonical"].(string); ok && canonical != "" {
		seo.Canonical = canonical
	}
	if priority, ok := sitemapPriorityValue(pageData["priority"]); ok {
		seo.Priority = &priority
	}
}

// sitemapPriorityValue coerces a frontmatter priority value (float64, int, or numeric
// string, as YAML/JSON decoders may yield) into a float32. A non-finite float (NaN or
// infinity, which a YAML .nan or an "Inf" string can produce) is rejected so it cannot
// reach the sitemap as an invalid <priority>.
//
// Takes value (any) which is the raw frontmatter value.
//
// Returns float32 which is the coerced priority.
// Returns bool which is true when coercion succeeded.
func sitemapPriorityValue(value any) (float32, bool) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return float32(v), true
	case float32:
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return 0, false
		}
		return v, true
	case int:
		return float32(v), true
	case string:
		if parsed, err := strconv.ParseFloat(v, 32); err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
			return float32(parsed), true
		}
	}
	return 0, false
}

// applyInstanceRichMedia populates the image, video, and news sitemap extensions for a
// collection item.
//
// The media is read from reserved, flat frontmatter keys on InitialProps["page"]. Flat
// scalar keys are used deliberately: the collection metadata encoder only supports scalar
// values, and one video / thumbnail per content item matches how markdown collections are
// authored. Recognised keys:
//   - sitemapImage: an image URL (comma-separated for several) -> <image:image>
//   - sitemapVideoTitle / ...Description / ...Thumbnail / ...Player / ...Content /
//     ...Duration / ...Date: a single <video:video> entry (built when a title and
//     thumbnail are present)
//   - sitemapNewsPublication / ...Language / ...Date / ...Title: a single <news:news>
//     entry
//
// Takes seo (*seo_dto.PageSEOMetadata) which receives the extracted media.
// Takes initialProps (map[string]any) which holds the instance's page metadata.
func applyInstanceRichMedia(seo *seo_dto.PageSEOMetadata, initialProps map[string]any) {
	pageData, ok := initialProps["page"].(map[string]any)
	if !ok {
		return
	}

	if images := splitCSV(propString(pageData, "sitemapImage")); len(images) > 0 {
		seo.ImageURLs = append(seo.ImageURLs, images...)
	}

	videoTitle := propString(pageData, "sitemapVideoTitle")
	videoThumbnail := propString(pageData, "sitemapVideoThumbnail")
	if videoTitle != "" && videoThumbnail != "" {
		seo.Videos = append(seo.Videos, seo_dto.VideoInputEntry{
			Title:             videoTitle,
			Description:       propString(pageData, "sitemapVideoDescription"),
			ThumbnailLocation: videoThumbnail,
			PlayerLocation:    propString(pageData, "sitemapVideoPlayer"),
			ContentLocation:   propString(pageData, "sitemapVideoContent"),
			PublicationDate:   propString(pageData, "sitemapVideoDate"),
			Duration:          clampVideoDuration(propInt(pageData, "sitemapVideoDuration")),
		})
	}

	newsPublication := propString(pageData, "sitemapNewsPublication")
	newsDate := propString(pageData, "sitemapNewsDate")
	if newsPublication != "" && newsDate != "" {
		seo.News = &seo_dto.NewsInputEntry{
			PublicationName:     newsPublication,
			PublicationLanguage: propString(pageData, "sitemapNewsLanguage"),
			PublicationDate:     newsDate,
			Title:               propString(pageData, "sitemapNewsTitle"),
		}
	}
}

// propString reads a scalar string frontmatter value, or "".
//
// Takes pageData (map[string]any) which holds the instance's page metadata.
// Takes key (string) which is the frontmatter key to read.
//
// Returns string which is the value, or "" when the key is absent or not a string.
func propString(pageData map[string]any, key string) string {
	if s, ok := pageData[key].(string); ok {
		return s
	}
	return ""
}

// propInt reads a scalar int frontmatter value, accepting int, int64, or float64.
//
// The value is guarded against a non-finite float, a negative value, and 32-bit overflow
// so a malformed frontmatter number cannot narrow to a garbage int.
//
// Takes pageData (map[string]any) which holds the instance's page metadata.
// Takes key (string) which is the frontmatter key to read.
//
// Returns int which is the value, or 0 when the key is absent, not numeric, or invalid.
func propInt(pageData map[string]any, key string) int {
	switch v := pageData[key].(type) {
	case int:
		return max(v, 0)
	case int64:
		return safeconv.Int64ToInt(max(v, 0))
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return int(math.Round(min(max(v, 0), float64(math.MaxInt32))))
	}
	return 0
}

// clampVideoDuration bounds a video duration to the range video sitemaps accept, so a
// garbage or oversized frontmatter value is capped rather than emitted. A non-positive
// value returns 0, which the omitempty tag then drops.
//
// Takes seconds (int) which is the raw duration.
//
// Returns int which is the bounded duration.
func clampVideoDuration(seconds int) int {
	const maxVideoDurationSeconds = 28800
	return min(max(seconds, 0), maxVideoDurationSeconds)
}

// splitCSV splits a comma-separated string into trimmed, non-empty parts.
//
// Takes value (string) which is the comma-separated input.
//
// Returns []string which are the trimmed, non-empty parts, or nil when the input is
// empty.
func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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

	isIndex := false
	if trimmed, found := strings.CutSuffix(basePath, "/index"); found {
		basePath = trimmed
		isIndex = true
	} else if basePath == "index" {
		basePath = ""
		isIndex = true
	}

	if basePath == "" {
		return "/"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	if isIndex && !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	return basePath
}
