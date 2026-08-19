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
	"errors"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/wdk/goroutine"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/seo/seo_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultMaxURLsPerSitemap is the default limit for URLs in a single sitemap file.
	defaultMaxURLsPerSitemap = 50_000

	// maxSitemapBytesBudget is the uncompressed byte budget for one sitemap file.
	maxSitemapBytesBudget = 49 * 1024 * 1024

	// maxProviderSitemapURLs bounds how many URLs a build-time SitemapURLProvider may
	// contribute in a single build. It is a safety valve against a runaway or buggy
	// provider, set far above realistic use; any truncation beyond it is logged, never
	// silent.
	maxProviderSitemapURLs = 100_000

	// maxRouteSourceSitemapURLs bounds how many URLs all build-time RouteSources may
	// contribute in a single build, counted after the per-locale fan-out.
	maxRouteSourceSitemapURLs = 100_000

	// dateFormatISO is the ISO 8601 date format (YYYY-MM-DD) used in sitemaps.
	dateFormatISO = "2006-01-02"

	// gitLastModTimeout bounds a single `git log` invocation used to derive a page's
	// lastmod, so a slow or hung git call cannot stall the build; on timeout the mtime
	// fallback is used.
	gitLastModTimeout = 3 * time.Second

	// urlPathSeparator is the forward slash used to separate parts of a URL path.
	urlPathSeparator = "/"

	// logFieldRoute is the log field key for a page route pattern.
	logFieldRoute = "route"

	// logFieldSource is the log field key for a route-source name.
	logFieldSource = "source"

	// logFieldCount is the log field key for a URL count.
	logFieldCount = "count"

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

var (
	// errGitLastModTimeout is the cause reported when the git last-commit lookup exceeds
	// gitLastModTimeout, distinguishing a timed-out lookup from an ordinary git failure.
	errGitLastModTimeout = errors.New("git last-commit lookup timed out")

	// validChangeFreqs is the set of values a sitemap <changefreq> may carry. A resolved
	// value outside this set is dropped rather than emitted, since the element is optional.
	validChangeFreqs = map[string]struct{}{
		"always":  {},
		"hourly":  {},
		"daily":   {},
		"weekly":  {},
		"monthly": {},
		"yearly":  {},
		"never":   {},
	}
)

// sitemapBuilder finds pages in the project and builds a complete sitemap with support
// for multiple languages and image discovery.
type sitemapBuilder struct {
	// dynamicURLSource fetches URLs that are created at runtime.
	dynamicURLSource DynamicURLSourcePort

	// urlProvider supplies additional URLs at build time, in-process. Optional; nil means no
	// extra build-time URLs.
	urlProvider SitemapURLProvider

	// routeSources enumerate the concrete URLs for pages bound to a p-route-source
	// directive. Each is matched to its pages by Name.
	routeSources []RouteSource

	// sandboxFactory creates sandboxes when no sandbox is directly injected. When non-nil
	// and sandbox is nil, this factory is used instead of safedisk.NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// sandbox is an optional file system sandbox for testing. When nil, sandboxes are
	// created per file's parent directory.
	sandbox safedisk.Sandbox

	// i18nDefaultLocale is the default locale code for building localised URLs.
	i18nDefaultLocale string

	// i18nStrategy is the locale-routing strategy, one of i18n_domain.Strategy*.
	//
	// It governs how localised URLs and hreflang alternates are built so the sitemap matches
	// the runtime router. Empty behaves as query-only/disabled (bare patterns).
	i18nStrategy string

	// i18nLocales is the full configured locale set, supplied to a RouteSource via
	// RouteContext when a RouteURL does not name its own locales.
	i18nLocales []string

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

	// isAuthGated indicates whether the page is behind an AuthPolicy.
	isAuthGated bool
}

// Build creates a sitemap from the project view.
//
// Takes view (*seo_dto.ProjectView) which provides the project data to build the sitemap
// from.
//
// Returns *seo_dto.SitemapBuildResult which contains either a single sitemap for small
// sites, or multiple sitemap files with an index for sites that exceed MaxURLsPerSitemap.
// Returns error when the build context is cancelled; individual source failures (dynamic,
// provider, route-source) are logged and skipped, never surfaced.
func (b *sitemapBuilder) Build(ctx context.Context, view *seo_dto.ProjectView) (*seo_dto.SitemapBuildResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	pages := b.discoverPages(view)
	l.Trace("Discovered pages for sitemap", logger_domain.Int(logFieldCount, len(pages)))

	discoveredURLs := make([]seo_dto.SitemapURL, 0, len(pages))
	for i := range pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page := &pages[i]

		if isNonIndexableRoute(page.routePattern) {
			l.Trace("Skipping non-indexable route", logger_domain.String(logFieldRoute, page.routePattern))
			continue
		}

		rule := b.matchRouteRule(ctx, page.routePattern)
		if b.shouldExcludePage(ctx, page.routePattern, page.isAuthGated, page.metadata, rule) {
			continue
		}

		url := b.buildSitemapURL(ctx, *page, rule)
		discoveredURLs = append(discoveredURLs, url)
	}

	dynamicURLs := b.fetchDynamicURLs(ctx)

	providedURLs := b.fetchProvidedURLs(ctx)

	routeSourceURLs := b.fetchRouteSourceURLs(ctx, view)

	extraURLs := make([]seo_dto.SitemapURL, 0, len(providedURLs)+len(dynamicURLs)+len(routeSourceURLs))
	extraURLs = append(extraURLs, providedURLs...)
	extraURLs = append(extraURLs, dynamicURLs...)
	extraURLs = append(extraURLs, routeSourceURLs...)

	allURLs := b.mergeAndDeduplicate(discoveredURLs, extraURLs)

	result := b.buildSitemapResult(ctx, allURLs)

	l.Trace("Generated sitemap",
		logger_domain.Int("total_urls", len(allURLs)),
		logger_domain.Int("sitemap_count", len(result.Sitemaps)),
		logger_domain.Bool("uses_index", result.Index != nil))
	return result, nil
}

// discoverPages finds all public pages in the project structure.
//
// Takes view (*seo_dto.ProjectView) which provides the project structure to search.
//
// Returns []pageDiscovery which contains an entry for each public page component.
// Returns an empty slice if view is nil.
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
			isAuthGated:   component.IsAuthGated,
		})
	}

	return pages
}

// isNonIndexableRoute reports whether a route pattern must be kept out of the sitemap.
//
// It rejects any pattern still containing an unexpanded brace placeholder (e.g.
// "/blog/{slug}"), a dynamic template that was never resolved to a concrete slug, and any
// pattern whose slash-delimited segment begins with "!" (the convention-based special
// pages such as "/!404" and "/!error"). The root "/" is not flagged: its segments are
// empty and contain neither a brace nor a bang.
//
// Takes routePattern (string) which is the route pattern to classify.
//
// Returns bool which is true when the route must be excluded from the sitemap.
func isNonIndexableRoute(routePattern string) bool {
	if strings.Contains(routePattern, "{") {
		return true
	}
	for segment := range strings.SplitSeq(routePattern, urlPathSeparator) {
		if strings.HasPrefix(segment, "!") {
			return true
		}
	}
	return false
}

// shouldExclude checks if a route pattern matches any exclusion pattern.
//
// Takes routePattern (string) which is the route to check.
//
// Returns bool which is true if the route matches any exclusion pattern.
func (b *sitemapBuilder) shouldExclude(ctx context.Context, routePattern string) bool {
	for _, pattern := range b.config.Exclude {
		if matchRoutePattern(ctx, pattern, routePattern) {
			return true
		}
	}
	return false
}

// shouldExcludePage reports whether a page must be kept out of the sitemap.
//
// A page is excluded because it is config-excluded, auth-gated without opt-in, marked
// noindex, or covered by an excluding route rule. It logs the specific reason at trace
// level. Both the discovered-page loop and the route-source path use it so the two cannot
// drift. It deliberately does not apply isNonIndexableRoute, whose brace check only suits
// fully expanded route patterns and would reject every route-source pattern.
//
// Takes routePattern (string) which is the page route being classified.
// Takes isAuthGated (bool) which reports whether the page declares an AuthPolicy.
// Takes metadata (seo_dto.PageSEOMetadata) which may carry a noindex robots rule.
// Takes rule (*config.SitemapRouteRule) which is the matching route rule, or nil.
//
// Returns bool which is true when the page must be excluded.
func (b *sitemapBuilder) shouldExcludePage(
	ctx context.Context,
	routePattern string,
	isAuthGated bool,
	metadata seo_dto.PageSEOMetadata,
	rule *config.SitemapRouteRule,
) bool {
	_, l := logger_domain.From(ctx, log)

	if b.shouldExclude(ctx, routePattern) {
		l.Trace("Excluding page from sitemap", logger_domain.String(logFieldRoute, routePattern))
		return true
	}
	if isAuthGated && !b.config.IncludeAuthGatedPages {
		l.Trace("Skipping auth-gated page", logger_domain.String(logFieldRoute, routePattern))
		return true
	}
	if strings.Contains(strings.ToLower(metadata.RobotsRule), "noindex") {
		l.Trace("Skipping noindex page", logger_domain.String(logFieldRoute, routePattern))
		return true
	}
	if rule != nil && (rule.Exclude || strings.Contains(strings.ToLower(rule.Robots), "noindex")) {
		l.Trace("Skipping page excluded by route rule", logger_domain.String(logFieldRoute, routePattern))
		return true
	}
	return false
}

// matchRoutePattern reports whether a route matches a glob pattern.
//
// It supports both filepath.Match semantics and a trailing "**" prefix wildcard (e.g.
// "/blog/**"). An invalid pattern is logged and treated as non-matching.
//
// Takes pattern (string) which is the glob pattern.
// Takes routePattern (string) which is the route to test.
//
// Returns bool which is true when the route matches.
func matchRoutePattern(ctx context.Context, pattern, routePattern string) bool {
	if strings.Contains(pattern, "**") {
		prefix := strings.TrimSuffix(pattern, "**")
		if strings.HasPrefix(routePattern, prefix) {
			return true
		}
	}
	matched, err := filepath.Match(pattern, routePattern)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Invalid sitemap route pattern", logger_domain.String("pattern", pattern), logger_domain.Error(err))
		return false
	}
	return matched
}

// matchRouteRule returns the first configured route rule whose pattern matches the route,
// or nil when none match. Route rules assign per-route
// priority/changefreq/robots/exclusion without editing pages.
//
// Takes routePattern (string) which is the route to match.
//
// Returns *config.SitemapRouteRule which is the matching rule, or nil.
func (b *sitemapBuilder) matchRouteRule(ctx context.Context, routePattern string) *config.SitemapRouteRule {
	for i := range b.config.RouteRules {
		if matchRoutePattern(ctx, b.config.RouteRules[i].Pattern, routePattern) {
			return &b.config.RouteRules[i]
		}
	}
	return nil
}

// buildSitemapURL creates a sitemap URL entry with all its data.
//
// Takes page (pageDiscovery) which provides the found page details including route and
// metadata.
//
// Returns seo_dto.SitemapURL which is a complete sitemap URL with location, timestamps,
// priority, alternate language links, and linked images.
func (b *sitemapBuilder) buildSitemapURL(ctx context.Context, page pageDiscovery, rule *config.SitemapRouteRule) seo_dto.SitemapURL {
	var absoluteURL string
	if len(page.metadata.SupportedLocales) > 1 {
		absoluteURL = b.buildLocalisedURL(page.routePattern, b.i18nDefaultLocale)
	} else {
		absoluteURL = b.buildAbsoluteURL(page.routePattern)
	}

	lastMod := b.determineLastMod(ctx, page.metadata.LastModified, page.sourcePath)

	priority := fmt.Sprintf("%.1f", b.resolvePriority(page.metadata, rule))
	changeFreq := b.resolveChangeFreq(ctx, page.routePattern, page.metadata, rule)

	alternates := b.buildAlternateLinks(page.routePattern, page.metadata.SupportedLocales)

	images := b.discoverImages(page.metadata.ImageURLs)
	videos := convertInputVideos(page.metadata.Videos)
	news := convertInputNews(page.metadata.News)

	return seo_dto.SitemapURL{
		Location:   absoluteURL,
		LastMod:    lastMod,
		ChangeFreq: changeFreq,
		Priority:   priority,
		Alternates: alternates,
		Images:     images,
		Videos:     videos,
		News:       news,
	}
}

// resolvePriority picks the sitemap priority using the precedence: the page's own
// override, then the matching route rule, then the configured default.
//
// Takes metadata (seo_dto.PageSEOMetadata) which may carry a per-page priority.
// Takes rule (*config.SitemapRouteRule) which may carry a per-route priority; nil for
// none.
//
// Returns float32 which is the resolved priority.
func (b *sitemapBuilder) resolvePriority(metadata seo_dto.PageSEOMetadata, rule *config.SitemapRouteRule) float32 {
	priority := b.config.Defaults.Priority
	switch {
	case metadata.Priority != nil:
		priority = *metadata.Priority
	case rule != nil && rule.Priority != nil:
		priority = *rule.Priority
	}

	return clampPriority(priority, b.config.Defaults.Priority)
}

// clampPriority coerces a sitemap priority into the valid 0.0-1.0 range. A non-finite
// value (NaN or infinity, which author input or a provider can produce) falls back to
// def, and a non-finite def falls back to 0, so the formatted <priority> can never be
// "NaN" or out of range.
//
// Takes v (float32) which is the candidate priority.
// Takes def (float32) which is the fallback when v is non-finite.
//
// Returns float32 which is the clamped, finite priority.
func clampPriority(v, def float32) float32 {
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		v = def
	}
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		v = 0
	}
	return min(max(v, 0), 1)
}

// resolveChangeFreq picks the sitemap changefreq using the precedence: the page's own
// override, then the matching route rule, then the configured default. Each candidate is
// validated so an invalid value is skipped rather than emitted.
//
// Takes routePattern (string) which identifies the page for logging.
// Takes metadata (seo_dto.PageSEOMetadata) which may carry a per-page changefreq.
// Takes rule (*config.SitemapRouteRule) which may carry a per-route changefreq, or nil.
//
// Returns string which is the resolved changefreq, or "" when no candidate is valid.
func (b *sitemapBuilder) resolveChangeFreq(ctx context.Context, routePattern string, metadata seo_dto.PageSEOMetadata, rule *config.SitemapRouteRule) string {
	var ruleFreq string
	if rule != nil {
		ruleFreq = rule.ChangeFreq
	}
	return b.resolveValidChangeFreq(ctx, routePattern, metadata.ChangeFrequency, ruleFreq, b.config.Defaults.ChangeFreq)
}

// resolveValidChangeFreq returns the first non-empty candidate that is a valid
// changefreq.
//
// The candidate is matched case-insensitively and returned lowercased so the emitted
// <changefreq> is always schema-valid. An invalid non-empty candidate is logged and
// skipped; when no candidate is valid it returns "" so the optional element is omitted
// rather than an invalid value shown.
//
// Takes routePattern (string) which identifies the page for logging.
// Takes candidates (...string) which are the changefreq values in precedence order.
//
// Returns string which is the resolved lowercase changefreq, or "".
func (*sitemapBuilder) resolveValidChangeFreq(ctx context.Context, routePattern string, candidates ...string) string {
	_, l := logger_domain.From(ctx, log)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		lower := strings.ToLower(candidate)
		if _, ok := validChangeFreqs[lower]; ok {
			return lower
		}
		l.Warn("Ignoring invalid sitemap changefreq value",
			logger_domain.String(logFieldRoute, routePattern),
			logger_domain.String("changefreq", candidate))
	}
	return ""
}

// buildAbsoluteURL creates a full URL from a route pattern.
//
// Takes routePattern (string) which is the path to add to the hostname.
//
// Returns string which is the full URL with the hostname and route pattern joined
// together.
func (b *sitemapBuilder) buildAbsoluteURL(routePattern string) string {
	hostname := strings.TrimSuffix(b.config.Hostname, urlPathSeparator)

	if !strings.HasPrefix(routePattern, urlPathSeparator) {
		routePattern = urlPathSeparator + routePattern
	}

	return hostname + routePattern
}

// determineLastMod determines the lastmod value. It prefers an explicit timestamp (a
// collection item's content date); otherwise, for a page with a source file, it uses the
// git last-commit date when SitemapConfig.GitLastMod is enabled (stable across unrelated
// edits), then the file modification time, and finally the current time.
//
// Takes explicitLastMod (*time.Time) which specifies an optional explicit timestamp to
// use.
// Takes sourcePath (string) which specifies the file path to check when explicitLastMod
// is nil.
//
// Returns string which is the lastmod value formatted as an ISO date.
func (b *sitemapBuilder) determineLastMod(ctx context.Context, explicitLastMod *time.Time, sourcePath string) string {
	if explicitLastMod != nil {
		return explicitLastMod.Format(dateFormatISO)
	}

	if sourcePath == "" {
		return time.Now().Format(dateFormatISO)
	}

	if b.config.GitLastMod {
		if gitDate, ok := gitLastCommitDate(ctx, sourcePath); ok {
			return gitDate
		}
	}

	if modTime := b.getFileModTime(sourcePath); modTime != nil {
		return modTime.Format(dateFormatISO)
	}

	return time.Now().Format(dateFormatISO)
}

// gitLastCommitDate returns the committer date of the most recent git commit for a file.
//
// The date is in YYYY-MM-DD form for the most recent git commit that touched sourcePath.
// It runs `git log` scoped to the file's directory with a short timeout derived from the
// build context and returns ok=false on any failure (git missing, not a repository, an
// untracked file, a cancelled build, or a timeout), so the caller falls back to the file
// mtime. The commit date ignores uncommitted working-tree changes, which is exactly why
// it is a stabler lastmod than mtime.
//
// Takes sourcePath (string) which is the page's source file path.
//
// Returns string which is the ISO date.
// Returns bool which is true when a date was obtained.
func gitLastCommitDate(ctx context.Context, sourcePath string) (string, bool) {
	ctx, cancel := context.WithTimeoutCause(ctx, gitLastModTimeout, errGitLastModTimeout)
	defer cancel()

	//nolint:gosec // G204: fixed git subcommand; only a build-time project source path varies, passed after -- as a pathspec.
	cmd := exec.CommandContext(ctx, "git", "-C", filepath.Dir(sourcePath),
		"log", "-1", "--format=%cs", "--", filepath.Base(sourcePath))

	cmd.WaitDelay = gitLastModTimeout
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}

	date := strings.TrimSpace(string(out))
	if date == "" {
		return "", false
	}
	return date, true
}

// getFileModTime attempts to get the modification time for the given file path.
//
// Takes sourcePath (string) which specifies the file path to check.
//
// Returns *time.Time which contains the modification time, or nil if the file cannot be
// accessed.
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
// Returns []seo_dto.AlternateLink which contains a self-referential alternate for every
// locale plus an x-default pointing at the default-locale variant, or nil if there is
// only one locale or fewer. Each localised page lists every alternate (including itself)
// plus an x-default; the hrefs are built through the same strategy helper the runtime
// router uses, so the sitemap can never disagree with the served routes.
func (b *sitemapBuilder) buildAlternateLinks(routePattern string, locales []string) []seo_dto.AlternateLink {
	if len(locales) <= 1 {
		return nil
	}

	alternates := make([]seo_dto.AlternateLink, 0, len(locales)+1)
	for _, locale := range locales {
		alternates = append(alternates, seo_dto.AlternateLink{
			Rel:      "alternate",
			Hreflang: locale,
			Href:     b.buildLocalisedURL(routePattern, locale),
		})
	}

	alternates = append(alternates, seo_dto.AlternateLink{
		Rel:      "alternate",
		Hreflang: "x-default",
		Href:     b.buildLocalisedURL(routePattern, b.i18nDefaultLocale),
	})

	return alternates
}

// buildLocalisedURL creates a full URL for a route pattern under a specific locale.
//
// It applies the configured i18n strategy (prefix / prefix_except_default / query-only /
// disabled). It delegates path construction to i18n_domain.RoutesByStrategy, the exact
// helper the manifest builder uses to register the runtime chi routes, so a localised
// <loc> or hreflang href always matches the route the server actually serves, including
// trailing-slash handling.
//
// Takes routePattern (string) which specifies the default-locale URL path pattern.
// Takes locale (string) which specifies the language code for localisation.
//
// Returns string which is the complete localised URL.
func (b *sitemapBuilder) buildLocalisedURL(routePattern string, locale string) string {
	hostname := strings.TrimSuffix(b.config.Hostname, urlPathSeparator)

	if !strings.HasPrefix(routePattern, urlPathSeparator) {
		routePattern = urlPathSeparator + routePattern
	}

	localised := i18n_domain.RoutesByStrategy(b.i18nStrategy, routePattern, b.i18nDefaultLocale, []string{locale})[locale]

	return hostname + localised
}

// discoverImages converts a page's opted-in image URLs into sitemap image entries.
//
// The URLs come from the page's PageSEOMetadata.ImageURLs: images the author explicitly
// marked for the sitemap (e.g. via the <piko:img sitemap> attribute), collected per page
// during annotation. They are root-relative serve paths, so each is made absolute against
// the configured hostname (an <image:loc> must be an absolute URL).
//
// Takes explicitImages ([]string) which lists the page's opted-in image URLs.
//
// Returns []seo_dto.ImageEntry for the page's images, or nil when image discovery is
// disabled.
func (b *sitemapBuilder) discoverImages(explicitImages []string) []seo_dto.ImageEntry {
	if b.config.DiscoverImages != nil && !*b.config.DiscoverImages {
		return nil
	}

	images := make([]seo_dto.ImageEntry, 0, len(explicitImages))
	for _, img := range explicitImages {
		images = append(images, seo_dto.ImageEntry{Location: b.buildAbsoluteURL(img)})
	}

	return images
}

// fetchDynamicURLs retrieves URLs from all configured dynamic sources.
//
// Returns []seo_dto.SitemapURL which contains all successfully fetched URLs. Individual
// source failures are logged and skipped rather than causing the entire fetch to fail.
func (b *sitemapBuilder) fetchDynamicURLs(ctx context.Context) []seo_dto.SitemapURL {
	if len(b.config.Sources) == 0 {
		return []seo_dto.SitemapURL{}
	}

	ctx, l := logger_domain.From(ctx, log)
	allDynamicURLs := make([]seo_dto.SitemapURL, 0)

	for _, sourceURL := range b.config.Sources {
		inputs, err := b.dynamicURLSource.FetchURLs(ctx, sourceURL)
		if err != nil {
			l.Warn("Failed to fetch dynamic URLs from source",
				logger_domain.String(logFieldSource, sourceURL),
				logger_domain.Error(err))
			continue
		}

		for i := range inputs {
			url := b.convertInputToURL(ctx, inputs[i])
			allDynamicURLs = append(allDynamicURLs, url)
		}
	}

	l.Trace("Fetched dynamic URLs", logger_domain.Int(logFieldCount, len(allDynamicURLs)))
	return allDynamicURLs
}

// fetchProvidedURLs collects URLs from the optional build-time SitemapURLProvider.
//
// Unlike fetchDynamicURLs (which fetches over HTTP and therefore yields nothing during
// the offline generator build), this runs in-process. A nil provider, a provider error,
// or a provider panic yields no URLs; the error is logged rather than failing the build.
// The provider is invoked synchronously, so it must honour ctx: there is no watchdog
// beyond the context it is handed. Its contribution is capped at maxProviderSitemapURLs.
//
// Returns []seo_dto.SitemapURL which contains the converted provider URLs.
func (b *sitemapBuilder) fetchProvidedURLs(ctx context.Context) []seo_dto.SitemapURL {
	if b.urlProvider == nil {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)

	inputs, err := goroutine.SafeCall1(ctx, "seo.sitemap.url_provider", func() ([]seo_dto.SitemapURLInput, error) {
		return b.urlProvider.SitemapURLs(ctx)
	})
	if err != nil {
		l.Warn("Failed to get URLs from sitemap URL provider", logger_domain.Error(err))
		return nil
	}

	if len(inputs) > maxProviderSitemapURLs {
		l.Warn("Sitemap URL provider returned more URLs than the allowed maximum; truncating",
			logger_domain.Int("returned", len(inputs)),
			logger_domain.Int("max", maxProviderSitemapURLs))
		inputs = inputs[:maxProviderSitemapURLs]
	}

	urls := make([]seo_dto.SitemapURL, 0, len(inputs))
	for i := range inputs {
		urls = append(urls, b.convertInputToURL(ctx, inputs[i]))
	}

	l.Trace("Collected build-time provider URLs", logger_domain.Int(logFieldCount, len(urls)))
	return urls
}

// fetchRouteSourceURLs enumerates the concrete URLs for every page bound to a route
// source.
//
// For each page bound to a p-route-source directive it builds a RouteContext from the
// page's real route pattern and the i18n config, invokes the matching registered source
// (inside a panic-safe boundary), and converts each returned RouteURL into one or more
// concrete, already-expanded SitemapURLs. A page whose source name is not registered is
// logged loudly and skipped, never silently dropped.
//
// Takes view (*seo_dto.ProjectView) which lists the project's components.
//
// Returns []seo_dto.SitemapURL which are the enumerated, expanded URLs.
func (b *sitemapBuilder) fetchRouteSourceURLs(ctx context.Context, view *seo_dto.ProjectView) []seo_dto.SitemapURL {
	if view == nil || len(b.routeSources) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)
	byName := b.routeSourcesByName()

	var urls []seo_dto.SitemapURL
	for i := range view.Components {
		if err := ctx.Err(); err != nil {
			l.Warn("Route-source enumeration cancelled", logger_domain.Error(err))
			break
		}

		rc, source, ok := b.resolveRouteSourcePage(ctx, &view.Components[i], byName)
		if !ok {
			continue
		}

		routeURLs, err := goroutine.SafeCall1(ctx, "seo.sitemap.route_source", func() ([]RouteURL, error) {
			return source.Enumerate(ctx, rc)
		})
		if err != nil {
			l.Warn("Route source failed to enumerate URLs; skipping",
				logger_domain.String(logFieldSource, rc.SourceName),
				logger_domain.Error(err))
			continue
		}

		expanded, capped := b.appendCappedRouteURLs(ctx, urls, rc, routeURLs)
		urls = expanded
		if capped {
			l.Warn("Route sources returned more URLs than the allowed maximum; truncating",
				logger_domain.String(logFieldSource, rc.SourceName),
				logger_domain.Int("collected", len(urls)),
				logger_domain.Int("max", maxRouteSourceSitemapURLs))
			break
		}
	}

	l.Trace("Collected route-source URLs", logger_domain.Int(logFieldCount, len(urls)))
	return urls
}

// appendCappedRouteURLs expands routeURLs onto urls, stopping and reporting capped=true
// once the running total reaches maxRouteSourceSitemapURLs so a single pathological
// source cannot blow past the cap during its per-locale fan-out.
//
// Takes urls ([]seo_dto.SitemapURL) which is the running accumulator.
// Takes rc (RouteContext) which provides the pattern and i18n config.
// Takes routeURLs ([]RouteURL) which are the enumerated URLs for one page.
//
// Returns []SitemapURL which is the extended accumulator.
// Returns bool which is true when the cap was reached.
func (b *sitemapBuilder) appendCappedRouteURLs(ctx context.Context, urls []seo_dto.SitemapURL, rc RouteContext, routeURLs []RouteURL) ([]seo_dto.SitemapURL, bool) {
	for j := range routeURLs {
		urls = append(urls, b.routeURLToSitemapURLs(ctx, rc, routeURLs[j])...)
		if len(urls) >= maxRouteSourceSitemapURLs {
			return urls[:maxRouteSourceSitemapURLs], true
		}
	}
	return urls, false
}

// routeSourcesByName indexes the registered route sources by Name for lookup.
//
// Returns map[string]RouteSource which maps each source name to its source.
func (b *sitemapBuilder) routeSourcesByName() map[string]RouteSource {
	byName := make(map[string]RouteSource, len(b.routeSources))
	for _, source := range b.routeSources {
		byName[source.Name()] = source
	}
	return byName
}

// resolveRouteSourcePage resolves a component's route-source binding into a context.
//
// It returns ok=false (logging the reason when a declared source cannot be satisfied)
// when the component is not a route-source page, lacks a p-param, is excluded by a page
// filter, or names an unregistered source. isNonIndexableRoute is not applied here: a
// route-source pattern always retains its {param} brace, which that check would reject.
//
// Takes component (*seo_dto.ComponentView) which is the candidate page.
// Takes byName (map[string]RouteSource) which indexes the registered sources.
//
// Returns RouteContext which carries the pattern and i18n config for enumeration.
// Returns RouteSource which is the registered source to enumerate.
// Returns bool which is true when the page is bound to a usable source.
func (b *sitemapBuilder) resolveRouteSourcePage(ctx context.Context, component *seo_dto.ComponentView, byName map[string]RouteSource) (RouteContext, RouteSource, bool) {
	ctx, l := logger_domain.From(ctx, log)

	if !component.IsPage || !component.IsPublic || component.RouteSourceName == "" {
		return RouteContext{}, nil, false
	}

	if component.RouteSourceParamName == "" {
		l.Warn("Page declares p-route-source without a p-param; its URLs are absent from the sitemap (add p-param to enumerate the dynamic segment)",
			logger_domain.String(logFieldRoute, component.RoutePattern),
			logger_domain.String(logFieldSource, component.RouteSourceName))
		return RouteContext{}, nil, false
	}

	rule := b.matchRouteRule(ctx, component.RoutePattern)
	if b.shouldExcludePage(ctx, component.RoutePattern, component.IsAuthGated, component.SEO, rule) {
		return RouteContext{}, nil, false
	}

	source, ok := byName[component.RouteSourceName]
	if !ok {
		l.Warn("Page declares an unregistered route source; its URLs are absent from the sitemap",
			logger_domain.String(logFieldRoute, component.RoutePattern),
			logger_domain.String(logFieldSource, component.RouteSourceName))
		return RouteContext{}, nil, false
	}

	return RouteContext{
		SourceName:    component.RouteSourceName,
		RoutePattern:  component.RoutePattern,
		ParamName:     component.RouteSourceParamName,
		DefaultLocale: b.i18nDefaultLocale,
		Locales:       b.i18nLocales,
		Strategy:      b.i18nStrategy,
	}, source, true
}

// routeURLToSitemapURLs expands one RouteURL into the concrete sitemap URLs it
// represents.
//
// It yields one URL per locale it supports, each with its param value substituted into
// the real route pattern. A single-locale RouteURL yields one URL with no hreflang; a
// multi-locale one yields cross-linked URLs carrying self-referential alternates plus an
// x-default (unless the RouteURL supplied explicit alternates).
//
// Takes rc (RouteContext) which provides the pattern and i18n config.
// Takes routeURL (RouteURL) which is the enumerated URL and its SEO.
//
// Returns []seo_dto.SitemapURL which are the concrete URLs.
func (b *sitemapBuilder) routeURLToSitemapURLs(ctx context.Context, rc RouteContext, routeURL RouteURL) []seo_dto.SitemapURL {
	if !isEnumerableParamValue(routeURL.ParamValue) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Route source produced a param value that does not name a real page; skipping its URL",
			logger_domain.String(logFieldSource, rc.SourceName),
			logger_domain.String("paramValue", routeURL.ParamValue))
		return nil
	}

	locales := routeURL.Locales
	if len(locales) == 0 {
		locales = rc.Locales
	}
	if len(locales) == 0 {
		locales = []string{rc.DefaultLocale}
	}

	var alternates []seo_dto.AlternateLink
	if len(routeURL.Alternates) > 0 {
		alternates = routeURL.Alternates
	} else if len(locales) > 1 {
		alternates = b.routeURLAlternates(rc, routeURL.ParamValue, locales)
	}

	urls := make([]seo_dto.SitemapURL, 0, len(locales))
	for _, locale := range locales {
		input := routeURL.SEO
		input.Location = rc.Expand(routeURL.ParamValue, locale)
		input.Alternates = alternates
		urls = append(urls, b.convertInputToURL(ctx, input))
	}
	return urls
}

// routeURLAlternates builds the hreflang alternate set for a multi-locale route URL: a
// self-referential alternate for every locale plus an x-default pointing at the
// default-locale variant, each resolved to an absolute URL through the shared strategy
// helper so they match the served routes.
//
// Takes rc (RouteContext) which provides the pattern and i18n config.
// Takes paramValue (string) which is substituted into the route pattern.
// Takes locales ([]string) which are the locales to cross-link.
//
// Returns []seo_dto.AlternateLink which are the alternates.
func (b *sitemapBuilder) routeURLAlternates(rc RouteContext, paramValue string, locales []string) []seo_dto.AlternateLink {
	alternates := make([]seo_dto.AlternateLink, 0, len(locales)+1)
	for _, locale := range locales {
		alternates = append(alternates, seo_dto.AlternateLink{
			Rel:      "alternate",
			Hreflang: locale,
			Href:     b.buildAbsoluteURL(rc.Expand(paramValue, locale)),
		})
	}
	alternates = append(alternates, seo_dto.AlternateLink{
		Rel:      "alternate",
		Hreflang: "x-default",
		Href:     b.buildAbsoluteURL(rc.Expand(paramValue, rc.DefaultLocale)),
	})
	return alternates
}

// convertInputToURL converts a SitemapURLInput into a SitemapURL.
//
// Takes input (seo_dto.SitemapURLInput) which contains the source URL data.
//
// Returns seo_dto.SitemapURL which contains the full location path and formatted image,
// video, and news entries.
func (b *sitemapBuilder) convertInputToURL(ctx context.Context, input seo_dto.SitemapURLInput) seo_dto.SitemapURL {
	location := input.Location
	if !isAbsoluteURL(location) {
		location = b.buildAbsoluteURL(location)
	}

	images := b.convertInputImages(input)
	videos := convertInputVideos(input.Videos)
	news := convertInputNews(input.News)

	priority := input.Priority
	if priority == 0 {
		priority = b.config.Defaults.Priority
	}
	priority = clampPriority(priority, b.config.Defaults.Priority)

	changeFreq := b.resolveValidChangeFreq(ctx, location, input.ChangeFreq, b.config.Defaults.ChangeFreq)

	alternates := input.Alternates
	if alternates == nil {
		alternates = []seo_dto.AlternateLink{}
	}

	return seo_dto.SitemapURL{
		Location:   location,
		LastMod:    b.normaliseLastMod(ctx, location, input.LastMod),
		ChangeFreq: changeFreq,
		Priority:   fmt.Sprintf("%.1f", priority),
		Alternates: alternates,
		Images:     images,
		Videos:     videos,
		News:       news,
	}
}

// normaliseLastMod converts a supplied lastmod into the ISO date form used everywhere
// else.
//
// The ISO form (YYYY-MM-DD) is used so both the emitted <lastmod> and the lexicographic
// comparison in latestLastMod operate on a single canonical format. An empty value passes
// through as empty; a value matching none of the accepted layouts is dropped (no
// <lastmod>) and logged.
//
// Takes location (string) which identifies the URL for logging.
// Takes lastMod (string) which is the raw lastmod value.
//
// Returns string which is the ISO date, or "" when absent or unparseable.
func (*sitemapBuilder) normaliseLastMod(ctx context.Context, location, lastMod string) string {
	if lastMod == "" {
		return ""
	}
	for _, layout := range []string{dateFormatISO, time.RFC3339, "2006-01-02T15:04:05", "2006/01/02"} {
		if parsed, err := time.Parse(layout, lastMod); err == nil {
			return parsed.Format(dateFormatISO)
		}
	}

	_, l := logger_domain.From(ctx, log)
	l.Warn("Dropping unparseable sitemap lastmod value",
		logger_domain.String("location", location),
		logger_domain.String("lastmod", lastMod))
	return ""
}

// isAbsoluteURL reports whether a sitemap location already carries an explicit http or
// https scheme and so must not be resolved against the configured hostname.
//
// A relative path is not absolute even when its first segment begins with "http" (e.g.
// the route slug "http-headers-guide"), which a bare "http" prefix check would
// misclassify.
//
// Takes location (string) which is the sitemap location to classify.
//
// Returns bool which is true when the location is an absolute http(s) URL.
func isAbsoluteURL(location string) bool {
	return strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")
}

// isEnumerableParamValue reports whether a route-source param value yields a well-formed
// URL segment. Empty, ".", and ".." are rejected because they void or navigate a path
// segment rather than naming a real page, so a URL built from them would not resolve.
//
// Takes value (string) which is the route-source param value.
//
// Returns bool which is true when the value names a real page segment.
func isEnumerableParamValue(value string) bool {
	return value != "" && value != "." && value != ".."
}

// convertInputImages builds image entries from a SitemapURLInput. Rich ImageEntries take
// precedence over the simple Images string list.
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

// mergeAndDeduplicate combines discovered and dynamic URLs, removing duplicates.
//
// Takes discovered ([]seo_dto.SitemapURL) which contains URLs found through crawling.
// Takes dynamic ([]seo_dto.SitemapURL) which contains programmatically generated URLs.
//
// Returns []seo_dto.SitemapURL which is a sorted, deduplicated slice of URLs. When
// duplicates exist, discovered URLs take precedence over dynamic ones.
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

// buildSitemapResult creates either a single sitemap or multiple sitemaps with an index.
// If the total URLs exceed MaxURLsPerSitemap, the URLs are split across several sitemaps.
//
// Takes allURLs ([]seo_dto.SitemapURL) which contains all URLs to include.
//
// Returns *seo_dto.SitemapBuildResult which contains the sitemaps and an optional index
// when splitting was needed.
func (b *sitemapBuilder) buildSitemapResult(ctx context.Context, allURLs []seo_dto.SitemapURL) *seo_dto.SitemapBuildResult {
	chunks := b.chunkURLs(ctx, allURLs)

	if len(chunks) <= 1 {
		urls := allURLs
		if len(chunks) == 1 {
			urls = chunks[0]
		}
		sitemap := buildSitemapNamespaces(urls)
		sitemap.URLs = urls

		return &seo_dto.SitemapBuildResult{
			Sitemaps: []seo_dto.Sitemap{sitemap},
			Index:    nil,
		}
	}

	sitemaps := make([]seo_dto.Sitemap, 0, len(chunks))
	for _, chunk := range chunks {
		sitemap := buildSitemapNamespaces(chunk)
		sitemap.URLs = chunk
		sitemaps = append(sitemaps, sitemap)
	}

	index := b.buildSitemapIndex(sitemaps)

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

// chunkURLs divides URLs into sitemap-sized groups.
//
// It cuts a new chunk whenever the next URL would push the current chunk past either the
// URL-count limit (MaxURLsPerSitemap) or the byte budget (maxSitemapBytesBudget). A URL
// is never split across chunks, so a single URL whose own metadata exceeds the budget
// still produces one over-budget file; that case is logged rather than silently emitted.
//
// Both result slices are preallocated. The chunk count is at least
// ceil(len(allURLs)/maxCount), and the byte budget can only ever split a chunk further,
// never merge two, so that count is a safe lower-bound reservation. Each chunk holds at
// most maxCount URLs and is bounded by the URLs that remain; because a completed chunk is
// retained by the result, a new chunk starts a fresh backing array rather than reslicing
// the previous one.
//
// Takes allURLs ([]seo_dto.SitemapURL) which contains all URLs to distribute.
//
// Returns [][]seo_dto.SitemapURL which is the ordered list of chunks; empty when there
// are no URLs.
func (b *sitemapBuilder) chunkURLs(ctx context.Context, allURLs []seo_dto.SitemapURL) [][]seo_dto.SitemapURL {
	_, l := logger_domain.From(ctx, log)

	maxCount := b.config.MaxURLsPerSitemap
	if maxCount <= 0 {
		maxCount = defaultMaxURLsPerSitemap
	}

	chunks := make([][]seo_dto.SitemapURL, 0, (len(allURLs)+maxCount-1)/maxCount)
	current := make([]seo_dto.SitemapURL, 0, min(maxCount, len(allURLs)))
	currentBytes := 0

	for i := range allURLs {
		size := estimateSitemapURLSize(&allURLs[i])

		exceedsCount := len(current) >= maxCount
		exceedsBytes := len(current) > 0 && currentBytes+size > maxSitemapBytesBudget
		if exceedsCount || exceedsBytes {
			chunks = append(chunks, current)
			current = make([]seo_dto.SitemapURL, 0, min(maxCount, len(allURLs)-i))
			currentBytes = 0
		}

		if len(current) == 0 && size > maxSitemapBytesBudget {
			l.Warn("Sitemap URL exceeds the single-file byte budget on its own; emitting an over-budget file",
				logger_domain.String("location", allURLs[i].Location),
				logger_domain.Int("estimated_bytes", size),
				logger_domain.Int("budget", maxSitemapBytesBudget))
		}

		current = append(current, allURLs[i])
		currentBytes += size
	}

	if len(current) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}

// estimateSitemapURLSize returns the approximate uncompressed XML byte size of one <url>.
//
// Takes url (*seo_dto.SitemapURL) which is the entry to size.
//
// Returns int which is the estimated byte size, or 0 on a marshal error.
func estimateSitemapURLSize(url *seo_dto.SitemapURL) int {
	encoded, err := xml.MarshalIndent(url, "", "  ")
	if err != nil {
		return 0
	}
	return len(encoded)
}

// buildSitemapIndex creates a sitemap index file referencing every sitemap chunk.
//
// Takes sitemaps ([]seo_dto.Sitemap) which are the chunk sitemaps, in order.
//
// Returns *seo_dto.SitemapIndex containing references to numbered sitemap files.
func (b *sitemapBuilder) buildSitemapIndex(sitemaps []seo_dto.Sitemap) *seo_dto.SitemapIndex {
	hostname := strings.TrimSuffix(b.config.Hostname, urlPathSeparator)
	refs := make([]seo_dto.SitemapRef, 0, len(sitemaps))

	for i := range sitemaps {
		ref := seo_dto.SitemapRef{
			Location: fmt.Sprintf("%s/sitemap-%d.xml", hostname, i+1),
			LastMod:  latestLastMod(sitemaps[i].URLs),
		}
		refs = append(refs, ref)
	}

	return &seo_dto.SitemapIndex{
		XMLName:  xml.Name{},
		Xmlns:    "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: refs,
	}
}

// latestLastMod returns the newest <lastmod> among the given URLs, or "" when none carry
// one.
//
// Takes urls ([]seo_dto.SitemapURL) which are the URLs to scan.
//
// Returns string which is the newest lastmod, or "" when none carry one.
func latestLastMod(urls []seo_dto.SitemapURL) string {
	latest := ""
	for i := range urls {
		if urls[i].LastMod > latest {
			latest = urls[i].LastMod
		}
	}
	return latest
}

// withSitemapSandbox sets a sandbox for testing file stat operations. The caller must
// close the sandbox when done.
//
// Takes sandbox (safedisk.Sandbox) which provides file system access.
//
// Returns sitemapBuilderOption which sets up the builder to use the sandbox.
func withSitemapSandbox(sandbox safedisk.Sandbox) sitemapBuilderOption {
	return func(b *sitemapBuilder) {
		b.sandbox = sandbox
	}
}

// withSitemapSandboxFactory sets a factory for creating sandboxes when no sandbox is
// directly injected.
//
// Takes factory (safedisk.Factory) which creates sandboxes for file stat operations.
//
// Returns sitemapBuilderOption which sets the factory on the builder.
func withSitemapSandboxFactory(factory safedisk.Factory) sitemapBuilderOption {
	return func(b *sitemapBuilder) {
		b.sandboxFactory = factory
	}
}

// withSitemapURLProvider sets a build-time, in-process provider of additional sitemap
// URLs.
//
// Takes provider (SitemapURLProvider) which enumerates the extra URLs during generation.
//
// Returns sitemapBuilderOption which sets the provider on the builder.
func withSitemapURLProvider(provider SitemapURLProvider) sitemapBuilderOption {
	return func(b *sitemapBuilder) {
		b.urlProvider = provider
	}
}

// withSitemapI18nStrategy sets the locale-routing strategy used to build localised URLs
// and hreflang alternates.
//
// Takes strategy (string) which is one of the i18n_domain.Strategy* values.
//
// Returns sitemapBuilderOption which sets the strategy on the builder.
func withSitemapI18nStrategy(strategy string) sitemapBuilderOption {
	return func(b *sitemapBuilder) {
		b.i18nStrategy = strategy
	}
}

// withSitemapI18nLocales sets the full configured locale set used when a RouteURL does
// not name its own locales.
//
// Takes locales ([]string) which is the configured locale set.
//
// Returns sitemapBuilderOption which sets the locale set on the builder.
func withSitemapI18nLocales(locales []string) sitemapBuilderOption {
	return func(b *sitemapBuilder) {
		b.i18nLocales = locales
	}
}

// withSitemapRouteSources sets the build-time route sources that enumerate URLs for pages
// bound to a p-route-source directive.
//
// Takes sources ([]RouteSource) which are the registered route sources.
//
// Returns sitemapBuilderOption which sets the sources on the builder.
func withSitemapRouteSources(sources []RouteSource) sitemapBuilderOption {
	return func(b *sitemapBuilder) {
		b.routeSources = sources
	}
}

// newSitemapBuilder creates a new sitemap builder with the given settings.
//
// When MaxURLsPerSitemap is zero or negative, it defaults to defaultMaxURLsPerSitemap.
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
// Returns []seo_dto.VideoEntry with the converted video entries, or nil when the input is
// empty.
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
// Returns *seo_dto.NewsEntry with the converted entry, or nil when the input is nil.
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

// buildSitemapNamespaces determines which XML namespaces are needed based on the content
// of the URL entries. Only namespaces for entry types that are actually present are
// included.
//
// Takes urls ([]seo_dto.SitemapURL) which contains the URL entries to inspect.
//
// Returns seo_dto.Sitemap with the base namespace and any optional namespaces populated
// according to the URL content.
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
