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

package daemon_adapters

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/seo/seo_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

var (
	// localeRouteParamPattern matches a route-pattern parameter such as {slug} or {slug:.+}.
	localeRouteParamPattern = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::[^}]*)?\}`)
)

// redirectToCanonicalSlash redirects a request to its canonical trailing-slash form.
//
// It issues a permanent (308) redirect to the trailing-slash-toggled form of the request
// path when that form matches a registered route but the requested form does not. This
// gives every page a single canonical URL per locale: directory-index pages are
// slash-canonical and leaf pages are slashless, and the non-canonical form is redirected
// to the registered one symmetrically for all locales (so /fr/x redirects to /fr/x/
// exactly as /x redirects to /x/). The 308 status preserves the request method.
//
// Takes router (chi.Router) whose registered routes are matched against the toggled path.
// Takes w (http.ResponseWriter) which receives the redirect response.
// Takes request (*http.Request) which is the incoming request whose path is
// canonicalised.
//
// Returns bool which is true when a redirect response was written.
func redirectToCanonicalSlash(router chi.Router, w http.ResponseWriter, request *http.Request) bool {
	toggled := toggleTrailingSlash(request.URL.Path)
	if toggled == request.URL.Path {
		return false
	}

	if !strings.HasPrefix(toggled, "/") || strings.HasPrefix(toggled, "//") {
		return false
	}
	if !router.Match(chi.NewRouteContext(), request.Method, toggled) {
		return false
	}
	location := toggled
	if request.URL.RawQuery != "" {
		location += "?" + request.URL.RawQuery
	}
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusPermanentRedirect)
	return true
}

// toggleTrailingSlash returns the path with its trailing slash toggled: a trailing slash
// is stripped, or one is added when absent. The root path is returned unchanged.
//
// Takes path (string) which is the request path.
//
// Returns string which is the path with its trailing slash flipped.
func toggleTrailingSlash(path string) string {
	if path == "" || path == "/" {
		return path
	}
	if trimmed, found := strings.CutSuffix(path, "/"); found {
		return trimmed
	}
	return path + "/"
}

// computeAutoLocaleHead derives the locale SEO head for a localised page.
//
// It builds the head (language, canonical URL, and hreflang alternates) from the page's
// registered per-locale route patterns, which already carry the correct locale prefixes
// and trailing slashes. It returns nil when the page has fewer than two locale variants,
// leaving SEO head generation to the page itself.
//
// Takes request (*http.Request) which carries the active locale and matched route params.
// Takes entry (templater_domain.PageEntryView) which exposes the page's route patterns.
// Takes websiteConfig (*config.WebsiteConfig) which provides locales and the base URL.
//
// Returns *templater_domain.LocaleSEOHead which holds the derived head, or nil when the
// page is not localised.
func computeAutoLocaleHead(
	request *http.Request,
	entry templater_domain.PageEntryView,
	websiteConfig *config.WebsiteConfig,
) *templater_domain.LocaleSEOHead {
	if websiteConfig == nil {
		return nil
	}
	patterns := entry.GetRoutePatterns()
	if len(patterns) == 0 {
		return nil
	}

	currentLocale := ""
	if pctx := daemon_dto.PikoRequestCtxFromContext(request.Context()); pctx != nil {
		currentLocale = pctx.Locale
	}
	baseURL := canonicalBaseURL(websiteConfig, request)
	params := routePathParams(request)

	if len(patterns) < 2 {
		return singleLocaleHead(currentLocale, baseURL, patterns, params)
	}

	return localeAlternatesHead(currentLocale, websiteConfig.I18n.DefaultLocale, baseURL,
		websiteConfig.I18n.Locales, patterns, params)
}

// singleLocaleHead builds the SEO head for a page with a single locale variant.
//
// The head carries a self-referential canonical (every indexable page should declare one)
// but no hreflang alternates or x-default. The substituted path is escaped per segment
// (the same helper the build-time sitemap uses) so an attacker-controlled route param
// cannot break out of the href attribute, and the served canonical matches the sitemap
// URL.
//
// Takes currentLocale (string) which is the active request locale.
// Takes baseURL (string) which is the absolute site origin.
// Takes patterns (map[string]string) which holds the single locale route pattern.
// Takes params (map[string]string) which holds the matched route params.
//
// Returns *templater_domain.LocaleSEOHead, or nil when no canonical could be built.
func singleLocaleHead(currentLocale, baseURL string, patterns, params map[string]string) *templater_domain.LocaleSEOHead {
	head := &templater_domain.LocaleSEOHead{Language: currentLocale}
	for _, pattern := range patterns {
		head.CanonicalURL = baseURL + seo_dto.EscapePathSegments(substituteRouteParams(pattern, params))
		break
	}
	if head.CanonicalURL == "" {
		return nil
	}
	return head
}

// localeAlternatesHead builds the SEO head for a multi-locale page: a self-referential
// hreflang alternate for every locale, a canonical pointing at the default-locale
// variant, and an x-default. Each href is escaped per segment so it is both
// injection-safe and byte-identical to the build-time sitemap URL.
//
// Takes currentLocale (string) which is the active request locale.
// Takes defaultLocale (string) which selects the canonical variant.
// Takes baseURL (string) which is the absolute site origin.
// Takes orderedLocales ([]string) which orders the alternates; empty sorts the pattern
// keys.
// Takes patterns (map[string]string) which maps each locale to its route pattern.
// Takes params (map[string]string) which holds the matched route params.
//
// Returns *templater_domain.LocaleSEOHead which holds the derived head.
func localeAlternatesHead(currentLocale, defaultLocale, baseURL string, orderedLocales []string, patterns, params map[string]string) *templater_domain.LocaleSEOHead {
	ordered := orderedLocales
	if len(ordered) == 0 {
		ordered = make([]string, 0, len(patterns))
		for locale := range patterns {
			ordered = append(ordered, locale)
		}
		slices.Sort(ordered)
	}

	head := &templater_domain.LocaleSEOHead{Language: currentLocale}
	alternates := make([]map[string]string, 0, len(patterns)+1)
	for _, locale := range ordered {
		pattern, ok := patterns[locale]
		if !ok {
			continue
		}
		fullURL := baseURL + seo_dto.EscapePathSegments(substituteRouteParams(pattern, params))
		alternates = append(alternates, map[string]string{"hreflang": locale, "href": fullURL})
		if locale == defaultLocale {
			head.CanonicalURL = fullURL
		}
	}
	if head.CanonicalURL == "" && len(alternates) > 0 {
		head.CanonicalURL = alternates[0]["href"]
	}
	if head.CanonicalURL != "" {
		alternates = append(alternates, map[string]string{"hreflang": "x-default", "href": head.CanonicalURL})
	}
	head.AlternateLinks = alternates
	return head
}

// canonicalBaseURL returns the absolute site origin for canonical and hreflang URLs. It
// prefers the configured WebsiteConfig.CanonicalBaseURL (so cached pages do not bake in a
// stale request host) and falls back to the live request host, mirroring the request
// scheme, when unset.
//
// Takes websiteConfig (*config.WebsiteConfig) which may carry the configured base URL.
// Takes request (*http.Request) which provides the fallback host and scheme.
//
// Returns string which is the scheme+host origin with any trailing slash removed.
func canonicalBaseURL(websiteConfig *config.WebsiteConfig, request *http.Request) string {
	if websiteConfig != nil && websiteConfig.CanonicalBaseURL != "" {
		return strings.TrimRight(websiteConfig.CanonicalBaseURL, "/")
	}
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + request.Host
}

// routePathParams reads the matched path parameters from the request's chi route context.
//
// Takes request (*http.Request) which holds the matched chi route context.
//
// Returns map[string]string which maps each captured parameter name to its value.
func routePathParams(request *http.Request) map[string]string {
	routeCtx := chi.RouteContext(request.Context())
	if routeCtx == nil {
		return nil
	}
	params := make(map[string]string, len(routeCtx.URLParams.Keys))
	for i, key := range routeCtx.URLParams.Keys {
		if i < len(routeCtx.URLParams.Values) {
			params[key] = routeCtx.URLParams.Values[i]
		}
	}
	return params
}

// substituteRouteParams replaces {name} and {name:regex} placeholders in a route pattern
// with the matched parameter values to form a concrete path. Unknown placeholders stay.
//
// Takes pattern (string) which is the route pattern containing placeholders.
// Takes params (map[string]string) which maps placeholder names to their values.
//
// Returns string which is the pattern with known parameters substituted.
func substituteRouteParams(pattern string, params map[string]string) string {
	if len(params) == 0 {
		return pattern
	}
	return localeRouteParamPattern.ReplaceAllStringFunc(pattern, func(match string) string {
		name := match[1 : len(match)-1]
		if colon := strings.IndexByte(name, ':'); colon >= 0 {
			name = name[:colon]
		}
		if value, ok := params[name]; ok {
			return value
		}
		return match
	})
}
