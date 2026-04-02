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

package seo_domain

import (
	"cmp"
	"context"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/seo/seo_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultMaxURLsPerSitemap is the default limit for URLs in a single sitemap file.
	defaultMaxURLsPerSitemap = 5000

	// dateFormatISO is the ISO 8601 date format (YYYY-MM-DD) used in sitemaps.
	dateFormatISO = "2006-01-02"

	// urlPathSeparator is the forward slash used to separate parts of a URL path.
	urlPathSeparator = "/"

	// namespaceSitemap is the base XML namespace for sitemap documents.
	namespaceSitemap = "http://www.sitemaps.org/schemas/sitemap/0.9"

	// namespaceImage is the XML namespace for image sitemap extensions.
	namespaceImage = "http://www.google.com/schemas/sitemap-image/1.1"

	// namespaceXhtml is the XHTML namespace for alternate language links.
	namespaceXhtml = "http://www.w3.org/1999/xhtml"

	// namespaceVideo is the XML namespace for video sitemap extensions.
	namespaceVideo = "http://www.google.com/schemas/sitemap-video/1.1"

	// namespaceNews is the XML namespace for news sitemap extensions.
	namespaceNews = "http://www.google.com/schemas/sitemap-news/0.9"
)

// sitemapBuilder finds pages in the project and builds a complete sitemap
// with support for multiple languages and image discovery.
type sitemapBuilder struct {
	// dynamicURLSource fetches URLs that are created at runtime.
	dynamicURLSource DynamicURLSourcePort

	// sandboxFactory creates sandboxes when no sandbox is directly injected.
	// When non-nil and sandbox is nil, this factory is used instead of
	// safedisk.NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// sandbox is an optional file system sandbox for testing.
	// When nil, sandboxes are created per file's parent directory.
	sandbox safedisk.Sandbox

	// i18nDefaultLocale is the default locale code for building localised URLs.
	i18nDefaultLocale string

	// config holds the settings for sitemap generation.
	config config.SitemapConfig
}

// sitemapBuilderOption configures a sitemapBuilder during construction.
type sitemapBuilderOption func(*sitemapBuilder)

// pageDiscovery holds information about a page found during SEO discovery.
type pageDiscovery struct {
	// routePattern is the URL pattern for this page, used to build the sitemap URL.
	routePattern string

	// componentHash is a hash of the component content.
	componentHash string

	// sourcePath is the file path where this page was found.
	sourcePath string

	// metadata holds the SEO data extracted from the page.
	metadata seo_dto.PageSEOMetadata

	// isPublic indicates whether the page can be viewed by anyone.
	isPublic bool
}

// Build creates a sitemap from the project view.
//
// Takes view (*seo_dto.ProjectView) which provides the project data to build
// the sitemap from.
//
// Returns *seo_dto.SitemapBuildResult which contains either a single sitemap
// for small sites, or multiple sitemap files with an index for sites that
// exceed MaxURLsPerSitemap.
// Returns error when fetching dynamic URLs fails. The build still finishes
// with the URLs it has found.
func (b *sitemapBuilder) Build(ctx context.Context, view *seo_dto.ProjectView) (*seo_dto.SitemapBuildResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	pages := b.discoverPages(view)
	l.Trace("Discovered pages for sitemap", logger_domain.Int("count", len(pages)))

	discoveredURLs := make([]seo_dto.SitemapURL, 0, len(pages))
	for i := range pages {
		page := &pages[i]
		if b.shouldExclude(ctx, page.routePattern) {
			l.Trace("Excluding page from sitemap", logger_domain.String("route", page.routePattern))
			continue
		}

		if strings.Contains(strings.ToLower(page.metadata.RobotsRule), "noindex") {
			l.Trace("Skipping noindex page", logger_domain.String("route", page.routePattern))
			continue
		}

		url := b.buildSitemapURL(*page, view)
		discoveredURLs = append(discoveredURLs, url)
	}

	dynamicURLs, err := b.fetchDynamicURLs(ctx)
	if err != nil {
		l.Warn("Failed to fetch dynamic URLs", logger_domain.Error(err))
		dynamicURLs = []seo_dto.SitemapURL{}
	}

	allURLs := b.mergeAndDeduplicate(discoveredURLs, dynamicURLs)

	result := b.buildSitemapResult(ctx, allURLs)

	l.Trace("Generated sitemap",
		logger_domain.Int("total_urls", len(allURLs)),
		logger_domain.Int("sitemap_count", len(result.Sitemaps)),
		logger_domain.Bool("uses_index", result.Index != nil))
	return result, nil
}

// discoverPages finds all public pages in the project structure.
//
// Takes view (*seo_dto.ProjectView) which provides the project structure to
// search.
//
// Returns []pageDiscovery which contains an entry for each public page
// component. Returns an empty slice if view is nil.
func (*sitemapBuilder) discoverPages(view *seo_dto.ProjectView) []pageDiscovery {
	if view == nil {
		return []pageDiscovery{}
	}

	pages := make([]pageDiscovery, 0, len(view.Components))
	for i := range view.Components {
		component := &view.Components[i]
		if !component.IsPage || !component.IsPublic {
			continue
		}

		pages = append(pages, pageDiscovery{
			routePattern:  component.RoutePattern,
			componentHash: component.HashedName,
			sourcePath:    component.OriginalSourcePath,
			metadata:      component.SEO,
			isPublic:      component.IsPublic,
		})
	}

	return pages
}

// shouldExclude checks if a route pattern matches any exclusion pattern.
//
// Takes routePattern (string) which is the route to check.
//
// Returns bool which is true if the route matches any exclusion pattern.
func (b *sitemapBuilder) shouldExclude(ctx context.Context, routePattern string) bool {
	_, l := logger_domain.From(ctx, log)
	for _, pattern := range b.config.Exclude {
		matched, err := filepath.Match(pattern, routePattern)
		if err != nil {
			l.Warn("Invalid exclusion pattern", logger_domain.String("pattern", pattern), logger_domain.Error(err))
			continue
		}
		if matched {
			return true
		}

		if strings.Contains(pattern, "**") {
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(routePattern, prefix) {
				return true
			}
		}
	}
	return false
}

// buildSitemapURL creates a sitemap URL entry with all its data.
//
// Takes page (pageDiscovery) which provides the found page details including
// route and metadata.
// Takes view (*seo_dto.ProjectView) which supplies project data for finding
// images.
//
// Returns seo_dto.SitemapURL which is a complete sitemap URL with location,
// timestamps, priority, alternate language links, and linked images.
func (b *sitemapBuilder) buildSitemapURL(
	page pageDiscovery,
	view *seo_dto.ProjectView,
) seo_dto.SitemapURL {
	absoluteURL := b.buildAbsoluteURL(page.routePattern)

	lastMod := b.determineLastMod(page.metadata.LastModified, page.sourcePath)

	priority := fmt.Sprintf("%.1f", b.config.Defaults.Priority)
	changeFreq := b.config.Defaults.ChangeFreq

	alternates := b.buildAlternateLinks(page.routePattern, page.metadata.SupportedLocales)

	images := b.discoverImages(view, page.metadata.ImageURLs)

	return seo_dto.SitemapURL{
		Location:   absoluteURL,
		LastMod:    lastMod,
		ChangeFreq: changeFreq,
		Priority:   priority,
		Alternates: alternates,
		Images:     images,
	}
}

// buildAbsoluteURL creates a full URL from a route pattern.
//
// Takes routePattern (string) which is the path to add to the hostname.
//
// Returns string which is the full URL with the hostname and route pattern
// joined together.
func (b *sitemapBuilder) buildAbsoluteURL(routePattern string) string {
	hostname := strings.TrimSuffix(b.config.Hostname, urlPathSeparator)

	if !strings.HasPrefix(routePattern, urlPathSeparator) {
		routePattern = urlPathSeparator + routePattern
	}

	return hostname + routePattern
}

// determineLastMod determines the lastmod value, falling back to file
// modification time if needed.
//
// Takes explicitLastMod (*time.Time) which specifies an optional explicit
// timestamp to use.
// Takes sourcePath (string) which specifies the file path to check for
// modification time when explicitLastMod is nil.
//
// Returns string which is the lastmod value formatted as ISO date. Uses
// explicitLastMod if provided, otherwise the file modification time, or
// current time as fallback.
func (b *sitemapBuilder) determineLastMod(explicitLastMod *time.Time, sourcePath string) string {
	if explicitLastMod != nil {
		return explicitLastMod.Format(dateFormatISO)
	}

	if sourcePath == "" {
		return time.Now().Format(dateFormatISO)
	}

	if modTime := b.getFileModTime(sourcePath); modTime != nil {
		return modTime.Format(dateFormatISO)
	}

	return time.Now().Format(dateFormatISO)
}

// getFileModTime attempts to get the modification time for the given file path.
//
// Takes sourcePath (string) which specifies the file path to check.
//
// Returns *time.Time which contains the modification time, or nil if the file
// cannot be accessed.
func (b *sitemapBuilder) getFileModTime(sourcePath string) *time.Time {
	fileName := filepath.Base(sourcePath)

	if b.sandbox != nil {
		fileInfo, err := b.sandbox.Stat(fileName)
		if err != nil {
			return nil
		}
		return new(fileInfo.ModTime())
	}

	parentDir := filepath.Dir(sourcePath)

	var sandbox safedisk.Sandbox
	var err error
	if b.sandboxFactory != nil {
		sandbox, err = b.sandboxFactory.Create("sitemap-stat", parentDir, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil
	}
	defer func() { _ = sandbox.Close() }()

	fileInfo, err := sandbox.Stat(fileName)
	if err != nil {
		return nil
	}
	return new(fileInfo.ModTime())
}

// buildAlternateLinks generates hreflang alternate links for i18n pages.
//
// Takes routePattern (string) which specifies the URL pattern for the route.
// Takes locales ([]string) which provides the list of supported locales.
//
// Returns []seo_dto.AlternateLink which contains alternate links for all
// locales, or nil if there is only one locale or fewer.
func (b *sitemapBuilder) buildAlternateLinks(routePattern string, locales []string) []seo_dto.AlternateLink {
	if len(locales) <= 1 {
		return nil
	}

	alternates := make([]seo_dto.AlternateLink, 0, len(locales))
	for _, locale := range locales {
		localisedURL := b.buildLocalisedURL(routePattern, locale)

		alternates = append(alternates, seo_dto.AlternateLink{
			Rel:      "alternate",
			Hreflang: locale,
			Href:     localisedURL,
		})
	}

	return alternates
}

// buildLocalisedURL creates a full URL with a language prefix for
// non-default locales.
//
// Takes routePattern (string) which specifies the URL path pattern to append.
// Takes locale (string) which specifies the language code for localisation.
//
// Returns string which is the complete URL with the hostname and locale prefix
// applied when the locale differs from the default.
func (b *sitemapBuilder) buildLocalisedURL(routePattern string, locale string) string {
	hostname := strings.TrimSuffix(b.config.Hostname, urlPathSeparator)

	if locale == b.i18nDefaultLocale {
		if !strings.HasPrefix(routePattern, "/") {
			routePattern = "/" + routePattern
		}
		return hostname + routePattern
	}

	if !strings.HasPrefix(routePattern, urlPathSeparator) {
		routePattern = urlPathSeparator + routePattern
	}
	return hostname + "/" + locale + routePattern
}

// discoverImages finds images associated with a page from the asset manifest.
//
// Takes view (*seo_dto.ProjectView) which provides the project data containing
// the asset manifest.
// Takes explicitImages ([]string) which lists image URLs to include directly.
//
// Returns []seo_dto.ImageEntry for all discovered images, or nil when image
// discovery is disabled.
func (b *sitemapBuilder) discoverImages(
	view *seo_dto.ProjectView,
	explicitImages []string,
) []seo_dto.ImageEntry {
	if !b.config.DiscoverImages {
		return nil
	}

	imageURLs := make([]string, 0)
	imageURLs = append(imageURLs, explicitImages...)

	for _, asset := range view.FinalAssetManifest {
		if asset.AssetType == "img" {
			imageURL := b.buildAbsoluteURL("/_piko/assets/" + filepath.Base(asset.SourcePath))
			imageURLs = append(imageURLs, imageURL)
		}
	}

	images := make([]seo_dto.ImageEntry, 0, len(imageURLs))
	for _, url := range imageURLs {
		images = append(images, seo_dto.ImageEntry{Location: url})
	}

	return images
}

// fetchDynamicURLs retrieves URLs from all configured dynamic sources.
//
// Returns []seo_dto.SitemapURL which contains all successfully fetched URLs.
// Returns error when the context is cancelled.
//
// Individual source failures are logged and skipped rather than causing the
// entire fetch to fail.
func (b *sitemapBuilder) fetchDynamicURLs(ctx context.Context) ([]seo_dto.SitemapURL, error) {
	if len(b.config.Sources) == 0 {
		return []seo_dto.SitemapURL{}, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	allDynamicURLs := make([]seo_dto.SitemapURL, 0)

	for _, sourceURL := range b.config.Sources {
		inputs, err := b.dynamicURLSource.FetchURLs(ctx, sourceURL)
		if err != nil {
			l.Warn("Failed to fetch dynamic URLs from source",
				logger_domain.String("source", sourceURL),
				logger_domain.Error(err))
			continue
		}

		for i := range inputs {
			url := b.convertInputToURL(inputs[i])
			allDynamicURLs = append(allDynamicURLs, url)
		}
	}

	l.Trace("Fetched dynamic URLs", logger_domain.Int("count", len(allDynamicURLs)))
	return allDynamicURLs, nil
}

// convertInputToURL converts a SitemapURLInput into a SitemapURL.
//
// Takes input (seo_dto.SitemapURLInput) which contains the source URL data.
//
// Returns seo_dto.SitemapURL which contains the full location path and
// formatted image, video, and news entries.
func (b *sitemapBuilder) convertInputToURL(input seo_dto.SitemapURLInput) seo_dto.SitemapURL {
	location := input.Location
	if !strings.HasPrefix(location, "http") {
		location = b.buildAbsoluteURL(location)
	}

	images := b.convertInputImages(input)
	videos := convertInputVideos(input.Videos)
	news := convertInputNews(input.News)

	return seo_dto.SitemapURL{
		Location:   location,
		LastMod:    input.LastMod,
		ChangeFreq: input.ChangeFreq,
		Priority:   fmt.Sprintf("%.1f", input.Priority),
		Alternates: []seo_dto.AlternateLink{},
		Images:     images,
		Videos:     videos,
		News:       news,
	}
}

// convertInputImages builds image entries from a SitemapURLInput.
// Rich ImageEntries take precedence over the simple Images string list.
//
// Takes input (seo_dto.SitemapURLInput) which contains the image data.
//
// Returns []seo_dto.ImageEntry with the converted image entries.
func (*sitemapBuilder) convertInputImages(input seo_dto.SitemapURLInput) []seo_dto.ImageEntry {
	if len(input.ImageEntries) > 0 {
		images := make([]seo_dto.ImageEntry, 0, len(input.ImageEntries))
		for _, img := range input.ImageEntries {
			images = append(images, seo_dto.ImageEntry(img))
		}
		return images
	}

	images := make([]seo_dto.ImageEntry, 0, len(input.Images))
	for _, imgURL := range input.Images {
		images = append(images, seo_dto.ImageEntry{Location: imgURL})
	}
	return images
}

// mergeAndDeduplicate combines discovered and dynamic URLs, removing
// duplicates.
//
// Takes discovered ([]seo_dto.SitemapURL) which contains URLs found through
// crawling.
// Takes dynamic ([]seo_dto.SitemapURL) which contains programmatically
// generated URLs.
//
// Returns []seo_dto.SitemapURL which is a sorted, deduplicated slice of URLs.
// When duplicates exist, discovered URLs take precedence over dynamic ones.
func (*sitemapBuilder) mergeAndDeduplicate(discovered, dynamic []seo_dto.SitemapURL) []seo_dto.SitemapURL {
	urlMap := make(map[string]*seo_dto.SitemapURL, len(discovered)+len(dynamic))

	for i := range discovered {
		urlMap[discovered[i].Location] = &discovered[i]
	}

	for i := range dynamic {
		if _, exists := urlMap[dynamic[i].Location]; !exists {
			urlMap[dynamic[i].Location] = &dynamic[i]
		}
	}

	merged := make([]seo_dto.SitemapURL, 0, len(urlMap))
	for _, url := range urlMap {
		merged = append(merged, *url)
	}

	slices.SortFunc(merged, func(a, b seo_dto.SitemapURL) int {
		return cmp.Compare(a.Location, b.Location)
	})

	return merged
}

// buildSitemapResult creates either a single sitemap or multiple sitemaps
// with an index. If the total URLs exceed MaxURLsPerSitemap, the URLs are
// split across several sitemaps.
//
// Takes allURLs ([]seo_dto.SitemapURL) which contains all URLs to include.
//
// Returns *seo_dto.SitemapBuildResult which contains the sitemaps and an
// optional index when splitting was needed.
func (b *sitemapBuilder) buildSitemapResult(ctx context.Context, allURLs []seo_dto.SitemapURL) *seo_dto.SitemapBuildResult {
	if len(allURLs) <= b.config.MaxURLsPerSitemap {
		sitemap := buildSitemapNamespaces(allURLs)
		sitemap.URLs = allURLs

		return &seo_dto.SitemapBuildResult{
			Sitemaps: []seo_dto.Sitemap{sitemap},
			Index:    nil,
		}
	}

	sitemaps := b.splitIntoSitemaps(allURLs)

	index := b.buildSitemapIndex(len(sitemaps))

	_, l := logger_domain.From(ctx, log)
	l.Trace("Split sitemap into chunks",
		logger_domain.Int("total_urls", len(allURLs)),
		logger_domain.Int("chunk_count", len(sitemaps)),
		logger_domain.Int("max_per_sitemap", b.config.MaxURLsPerSitemap))

	return &seo_dto.SitemapBuildResult{
		Sitemaps: sitemaps,
		Index:    index,
	}
}

// splitIntoSitemaps divides URLs into multiple sitemap files based on
// MaxURLsPerSitemap.
//
// Takes allURLs ([]seo_dto.SitemapURL) which contains all URLs to distribute.
//
// Returns []seo_dto.Sitemap which contains sitemaps, each with at most
// MaxURLsPerSitemap URLs.
func (b *sitemapBuilder) splitIntoSitemaps(allURLs []seo_dto.SitemapURL) []seo_dto.Sitemap {
	chunkCount := (len(allURLs) + b.config.MaxURLsPerSitemap - 1) / b.config.MaxURLsPerSitemap
	sitemaps := make([]seo_dto.Sitemap, 0, chunkCount)

	for i := 0; i < len(allURLs); i += b.config.MaxURLsPerSitemap {
		end := min(i+b.config.MaxURLsPerSitemap, len(allURLs))

		chunk := allURLs[i:end]
		sitemap := buildSitemapNamespaces(chunk)
		sitemap.URLs = chunk

		sitemaps = append(sitemaps, sitemap)
	}

	return sitemaps
}

// buildSitemapIndex creates a sitemap index file with references to all
// sitemap chunks.
//
// Takes sitemapCount (int) which specifies the number of sitemap files to
// reference in the index.
//
// Returns *seo_dto.SitemapIndex containing references to numbered sitemap
// files.
func (b *sitemapBuilder) buildSitemapIndex(sitemapCount int) *seo_dto.SitemapIndex {
	hostname := strings.TrimSuffix(b.config.Hostname, urlPathSeparator)
	refs := make([]seo_dto.SitemapRef, 0, sitemapCount)

	for i := 1; i <= sitemapCount; i++ {
		ref := seo_dto.SitemapRef{
			Location: fmt.Sprintf("%s/sitemap-%d.xml", hostname, i),
			LastMod:  time.Now().Format("2006-01-02"),
		}
		refs = append(refs, ref)
	}

	return &seo_dto.SitemapIndex{
		XMLName:  xml.Name{},
		Xmlns:    "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: refs,
	}
}

// withSitemapSandbox sets a sandbox for testing file stat operations.
// The caller must close the sandbox when done.
//
// Takes sandbox (safedisk.Sandbox) which provides file system access.
//
// Returns sitemapBuilderOption which sets up the builder to use the sandbox.
func withSitemapSandbox(sandbox safedisk.Sandbox) sitemapBuilderOption {
	return func(b *sitemapBuilder) {
		b.sandbox = sandbox
	}
}

// withSitemapSandboxFactory sets a factory for creating sandboxes when no
// sandbox is directly injected.
//
// Takes factory (safedisk.Factory) which creates sandboxes for file stat
// operations.
//
// Returns sitemapBuilderOption which sets the factory on the builder.
func withSitemapSandboxFactory(factory safedisk.Factory) sitemapBuilderOption {
	return func(b *sitemapBuilder) {
		b.sandboxFactory = factory
	}
}

// newSitemapBuilder creates a new sitemap builder with the given settings.
//
// When MaxURLsPerSitemap is zero or negative, it defaults to
// defaultMaxURLsPerSitemap.
//
// Takes sitemapConfig (config.SitemapConfig) which provides the sitemap settings.
// Takes i18nDefaultLocale (string) which sets the default locale for URLs.
// Takes dynamicURLSource (DynamicURLSourcePort) which supplies dynamic URLs.
// Takes opts (...sitemapBuilderOption) which allows optional settings such as
// withSitemapSandbox for testing.
//
// Returns *sitemapBuilder which is ready for use.
func newSitemapBuilder(
	sitemapConfig config.SitemapConfig,
	i18nDefaultLocale string,
	dynamicURLSource DynamicURLSourcePort,
	opts ...sitemapBuilderOption,
) *sitemapBuilder {
	if sitemapConfig.MaxURLsPerSitemap <= 0 {
		sitemapConfig.MaxURLsPerSitemap = defaultMaxURLsPerSitemap
	}

	b := &sitemapBuilder{
		config:            sitemapConfig,
		i18nDefaultLocale: i18nDefaultLocale,
		dynamicURLSource:  dynamicURLSource,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// convertInputVideos builds video entries from dynamic source input.
//
// Takes inputs ([]seo_dto.VideoInputEntry) which contains the video data.
//
// Returns []seo_dto.VideoEntry with the converted video entries, or nil
// when the input is empty.
func convertInputVideos(inputs []seo_dto.VideoInputEntry) []seo_dto.VideoEntry {
	if len(inputs) == 0 {
		return nil
	}

	videos := make([]seo_dto.VideoEntry, 0, len(inputs))
	for i := range inputs {
		videos = append(videos, seo_dto.VideoEntry(inputs[i]))
	}
	return videos
}

// convertInputNews builds a news entry from dynamic source input.
//
// Takes input (*seo_dto.NewsInputEntry) which contains the news data.
//
// Returns *seo_dto.NewsEntry with the converted entry, or nil when
// the input is nil.
func convertInputNews(input *seo_dto.NewsInputEntry) *seo_dto.NewsEntry {
	if input == nil {
		return nil
	}

	return &seo_dto.NewsEntry{
		Publication: seo_dto.NewsPublication{
			Name:     input.PublicationName,
			Language: input.PublicationLanguage,
		},
		PublicationDate: input.PublicationDate,
		Title:           input.Title,
	}
}

// buildSitemapNamespaces determines which XML namespaces are needed based on
// the content of the URL entries. Only namespaces for entry types that are
// actually present are included.
//
// Takes urls ([]seo_dto.SitemapURL) which contains the URL entries to inspect.
//
// Returns seo_dto.Sitemap with the base namespace and any optional namespaces
// populated according to the URL content.
func buildSitemapNamespaces(urls []seo_dto.SitemapURL) seo_dto.Sitemap {
	sitemap := seo_dto.Sitemap{
		Xmlns: namespaceSitemap,
	}

	for i := range urls {
		url := &urls[i]
		if len(url.Images) > 0 {
			sitemap.XmlnsImage = namespaceImage
		}
		if len(url.Alternates) > 0 {
			sitemap.XmlnsXhtml = namespaceXhtml
		}
		if len(url.Videos) > 0 {
			sitemap.XmlnsVideo = namespaceVideo
		}
		if url.News != nil {
			sitemap.XmlnsNews = namespaceNews
		}

		if sitemap.XmlnsImage != "" && sitemap.XmlnsXhtml != "" &&
			sitemap.XmlnsVideo != "" && sitemap.XmlnsNews != "" {
			break
		}
	}

	return sitemap
}
