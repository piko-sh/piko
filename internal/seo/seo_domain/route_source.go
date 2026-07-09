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
	"context"
	"strings"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/seo/seo_dto"
)

// RouteSource is a build-time, in-process enumerator of the concrete URLs for a page.
type RouteSource interface {
	// Name is the identifier a page's p-route-source directive refers to.
	//
	// Returns string which is the unique source name.
	Name() string

	// Enumerate returns every concrete URL (with per-URL SEO) for the page bound to this
	// source. It runs during generation, in-process; it must honour ctx and should be
	// deterministic across builds so the sitemap stays stable.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes rc (RouteContext) which carries the page's real route pattern, dynamic param
	// name, and the i18n configuration needed to build correct localised URLs.
	//
	// Returns []RouteURL which are the concrete URLs to add to the sitemap.
	// Returns error when enumeration fails; the build logs it and continues without these
	// URLs.
	Enumerate(ctx context.Context, rc RouteContext) ([]RouteURL, error)
}

// RouteSourceFunc adapts a bare closure to the RouteSource interface. Unlike the
// single-method SitemapURLProviderFunc, RouteSource has two methods (Name and Enumerate),
// so the adapter is a struct carrying the name alongside the closure.
type RouteSourceFunc struct {
	// Fn is invoked by Enumerate.
	Fn func(ctx context.Context, rc RouteContext) ([]RouteURL, error)

	// SourceName is returned by Name and binds the source to a p-route-source directive.
	SourceName string
}

// Name returns the configured source name.
//
// Returns string which is the configured source name.
func (f RouteSourceFunc) Name() string { return f.SourceName }

// Enumerate invokes the wrapped closure.
//
// Takes rc (RouteContext) which carries the route pattern and i18n configuration for the
// page.
//
// Returns []RouteURL which are the concrete URLs to add to the sitemap.
// Returns error when the wrapped closure fails.
func (f RouteSourceFunc) Enumerate(ctx context.Context, rc RouteContext) ([]RouteURL, error) {
	return f.Fn(ctx, rc)
}

// RouteContext gives a RouteSource the authoritative route pattern and i18n configuration
// for the page it enumerates, so it substitutes param values into the real route and its
// URLs match the routes the runtime router serves.
type RouteContext struct {
	// SourceName is the p-route-source value that bound this source to the page.
	SourceName string

	// RoutePattern is the page's brace-retaining route pattern, e.g.
	// "/managed-cloud-services{locationslug}/kubernetes".
	RoutePattern string

	// ParamName is the dynamic segment the source enumerates, e.g. "locationslug".
	ParamName string

	// DefaultLocale is the site's default locale.
	DefaultLocale string

	// Strategy is the i18n routing strategy (one of i18n_domain.Strategy*).
	Strategy string

	// Locales is the full configured locale set, used when a RouteURL does not specify its
	// own locales.
	Locales []string
}

// Expand builds the root-relative URL path for one locale.
//
// Takes paramValue (string) which replaces the {ParamName} placeholder.
// Takes locale (string) which selects the locale variant.
//
// Returns string which is the localised, escaped, root-relative URL path, or "".
func (rc RouteContext) Expand(paramValue, locale string) string {
	if rc.ParamName == "" {
		return ""
	}

	pattern := rc.RoutePattern
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	localisedPattern := i18n_domain.RoutesByStrategy(rc.Strategy, pattern, rc.DefaultLocale, []string{locale})[locale]
	substituted := strings.ReplaceAll(localisedPattern, "{"+rc.ParamName+"}", paramValue)
	return seo_dto.EscapePathSegments(substituted)
}

// RouteURL is one concrete URL enumerated by a RouteSource, with its per-URL SEO. It
// reuses SitemapURLInput for the metadata payload so any SitemapURLInput capability
// (images, videos, news, priority, changefreq, lastmod) is available to sources for free.
type RouteURL struct {
	// ParamValue is substituted into the route pattern's {param}, e.g. "-jersey".
	ParamValue string

	// Locales lists the locales this URL is available in.
	Locales []string

	// Alternates, when set, are used verbatim as this URL's hreflang alternates instead of
	// the automatic cross-linking. Hrefs may be relative (resolved against the hostname) or
	// absolute.
	Alternates []seo_dto.AlternateLink

	// SEO carries lastmod / changefreq / priority / images / videos / news for the URL.
	SEO seo_dto.SitemapURLInput
}
