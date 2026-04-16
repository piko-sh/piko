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

package templater_dto

import "time"

// AssetRef represents a static asset dependency of a component, such as an
// SVG or CSS file. This information can be used by the renderer to generate
// preload headers.
type AssetRef struct {
	// Kind is the asset type (e.g. "svg"); used with Path to identify the asset.
	Kind string `json:"kind"`

	// Path is the location of the asset; empty means no asset is needed.
	Path string `json:"path"`
}

// MetaTag represents an HTML meta tag used for page metadata.
type MetaTag struct {
	// Name is the value of the meta tag's name attribute.
	Name string `json:"name"`

	// Content is the value of the meta tag's content attribute.
	Content string `json:"content"`
}

// OGTag represents an Open Graph protocol <meta> tag (e.g., <meta
// property="og:title"...>).
type OGTag struct {
	// Property is the Open Graph property name (e.g. "og:title", "og:image").
	Property string `json:"property"`

	// Content is the value for the og:content meta tag attribute.
	Content string `json:"content"`
}

// CachePolicy controls how a component's output is cached at the CDN edge
// and in the browser. Field ordering is set for memory alignment.
type CachePolicy struct {
	// Key is an optional identifier combined with the URL to form the cache key.
	// Use for per-user caching, A/B testing, or localisation.
	Key string `json:"key,omitempty"`

	// MaxAgeSeconds sets the max-age value in seconds for the Cache-Control
	// header. It also sets how long the server-side AST cache stores entries.
	MaxAgeSeconds int `json:"maxAgeSeconds,omitempty"`

	// Enabled is the master switch for server-side AST caching. If false, the AST
	// is never cached.
	Enabled bool `json:"enabled"`

	// OnRender indicates whether the AST should be built once at build time
	// and stored as static content with a long cache time.
	OnRender bool `json:"onRender"`

	// Static enables full-page caching of rendered HTML; only suitable for pages
	// that do not vary per-user. If false, the system may still use the AST cache.
	Static bool `json:"static,omitempty"`

	// MustRevalidate corresponds to the `must-revalidate` Cache-Control directive,
	// forcing revalidation with the origin server.
	MustRevalidate bool `json:"mustRevalidate"`

	// NoStore corresponds to the `no-store` Cache-Control directive, indicating
	// that the response should not be stored in any cache.
	NoStore bool `json:"noStore"`
}

// Metadata holds SEO and page-level settings that a component's Render
// function can return to control the final HTML document's head element.
// Field ordering is optimised for memory alignment.
type Metadata struct {
	// LastModified is the last modification date for SEO purposes, used in the
	// <lastmod> tag in sitemap.xml. If nil, the source file's modification time
	// is used.
	LastModified *time.Time `json:"lastModified,omitempty"`

	// Language sets the lang attribute on the HTML tag.
	Language string `json:"language,omitempty"`

	// Title is the content of the HTML title tag.
	Title string `json:"title,omitempty"`

	// RobotsRule controls the robots meta tag (<meta name="robots" content="...">)
	// and the X-Robots-Tag HTTP header, where common values are "noindex,
	// nofollow" and "index, follow", defaulting to sensible environment-based
	// values when not set.
	RobotsRule string `json:"robotsRule,omitempty"`

	// ServerRedirect specifies a URL for server-side page rewriting.
	//
	// The server fetches and renders the specified page internally while the
	// browser URL remains unchanged. For example, requesting /login could
	// render /setup while the URL still shows /login.
	//
	// Maximum 3 hops are allowed to prevent infinite loops. If both
	// ServerRedirect and ClientRedirect are set, ServerRedirect takes
	// precedence.
	ServerRedirect string `json:"serverRedirect,omitempty"`

	// ClientRedirect is the URL for HTTP redirect responses.
	// Uses RedirectStatus for the status code; default is 302.
	ClientRedirect string `json:"clientRedirect,omitempty"`

	// StatusText is custom text for a non-200 HTTP status, such as "Not Found".
	StatusText string `json:"statusText,omitempty"`

	// CacheKey provides an optional, explicit key for server-side caching,
	// overriding the default request-based key. Useful for pages that vary on
	// something not in the URL.
	CacheKey string `json:"cacheKey,omitempty"`

	// Keywords sets the content for the `<meta name="keywords">` tag.
	Keywords string `json:"keywords,omitempty"`

	// CanonicalURL is the URL for the canonical link HTML element.
	CanonicalURL string `json:"canonicalUrl,omitempty"`

	// Description sets the content for the HTML meta description tag.
	Description string `json:"description,omitempty"`

	// OGTags is a list of Open Graph meta tags for social media sharing.
	OGTags []OGTag `json:"ogTags,omitempty"`

	// AlternateLinks holds link data for <link rel="alternate"> tags, used for
	// language versions (hreflang) or different media types.
	AlternateLinks []map[string]string `json:"alternateLinks,omitempty"`

	// MetaTags is a list of meta tags to add to the document head.
	MetaTags []MetaTag `json:"metaTags,omitempty"`

	// TwitterCards holds Twitter Card meta tags to render in the document head,
	// where each entry's Name field is a Twitter card property (e.g.
	// "twitter:card", "twitter:site", "twitter:title") and the Content field
	// holds the corresponding value.
	TwitterCards []MetaTag `json:"twitterCards,omitempty"`

	// StructuredData holds raw JSON-LD blocks rendered as
	// <script type="application/ld+json"> in the document head. Invalid JSON
	// entries are silently dropped with a warning log.
	StructuredData []string `json:"structuredData,omitempty"`

	// RedirectStatus specifies the HTTP status code for ClientRedirect, accepting
	// 301 (permanent), 302 (temporary, default), 303 (see other), or 307
	// (temporary, preserve method), defaulting to 302 (Found) when omitted or
	// invalid and ignored when ClientRedirect is empty.
	RedirectStatus int `json:"redirectStatus,omitempty"`

	// Status is the HTTP status code for the response (e.g., 200, 404, 500).
	Status int `json:"status,omitempty"`
}

// JSScriptMeta contains metadata about a single client-side JavaScript module
// needed by a page. This includes the script URL and, for partial scripts,
// the friendly partial name used for function scoping on the frontend.
type JSScriptMeta struct {
	// URL is the full URL path to the JavaScript module (e.g.,
	// "/_piko/assets/pk-js/partials/modal.js").
	URL string `json:"url"`

	// PartialName is the name of the partial this script belongs to
	// (e.g., "modal-wrapper"). Empty for page-level scripts.
	PartialName string `json:"partialName,omitempty"`

	// SRIHash is the Subresource Integrity hash for this script module.
	// Empty when SRI is disabled or the hash has not been computed.
	SRIHash string `json:"sriHash,omitempty"`
}

// InternalMetadata holds the complete metadata returned by the BuildAST
// function. It combines user-defined Metadata with system-level information
// derived during compilation and rendering, and is serialised to JSON for
// caching.
type InternalMetadata struct {
	// RenderError holds a typed error from the generated BuildAST function when
	// the page's rendering fails before producing an AST, preserving the
	// original error type (e.g., a collection-not-found 404) so the HTTP
	// handler can extract the correct status code and route to the appropriate
	// error page.
	//
	// Not serialised -- only used for in-process error propagation.
	RenderError error `json:"-"`

	// JSScriptMetas holds client-side JavaScript modules for this page and its
	// embedded partials. Each entry contains a URL and optional partial name for
	// scoping.
	JSScriptMetas []JSScriptMeta `json:"jsScriptMetas,omitempty"`

	// AssetRefs lists the static assets found during compilation that the
	// component tree needs.
	AssetRefs []AssetRef `json:"assetRefs,omitempty"`

	// CustomTags lists custom element tags (e.g. <my-component>) found in the
	// component tree, used for client-side hydration.
	CustomTags []string `json:"customTags,omitempty"`

	// SupportedLocales lists the locales this component supports.
	SupportedLocales []string `json:"supportedLocales,omitempty"`

	// CachePolicy is the embedded struct containing the caching rules for this
	// component. The fields will be flattened into this object during JSON
	// serialisation.
	CachePolicy

	// Metadata is the embedded struct containing SEO and page-level information.
	// The fields will be flattened into this object during JSON serialisation.
	Metadata

	// UsesCaptcha indicates this page or partial contains a piko:captcha element
	// and needs captcha provider scripts loaded.
	UsesCaptcha bool `json:"usesCaptcha,omitempty"`
}
