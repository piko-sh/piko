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

package generator_dto

import (
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// Manifest is the root object for the project manifest file. It is the
// definitive, serialisable phone book for the entire compiled Piko application,
// containing all static metadata needed by the runtime server.
type Manifest struct {
	// Pages maps project-relative source paths to page entries. The runtime router
	// uses this map to match incoming requests to the correct compiled component.
	Pages map[string]ManifestPageEntry `json:"pages"`

	// Partials maps project-relative source paths (e.g. "partials/card.pk") to
	// their manifest entries for non-routable, reusable components.
	Partials map[string]ManifestPartialEntry `json:"partials"`

	// Emails maps manifest keys to email template entries.
	Emails map[string]ManifestEmailEntry `json:"emails"`

	// Pdfs maps manifest keys to PDF template entries.
	Pdfs map[string]ManifestPdfEntry `json:"pdfs"`

	// ErrorPages maps project-relative source paths to error page
	// entries.
	//
	// Paths use the ! prefix convention (e.g. "pages/!404.pk")
	// and are rendered when the corresponding HTTP error occurs,
	// with hierarchical scoping.
	ErrorPages map[string]ManifestErrorPageEntry `json:"errorPages,omitempty"`

	// CollectionFallbackRoutes holds the original dynamic route patterns
	// from static collections, preserving patterns like {slug} so the
	// daemon can register low-priority fallback routes that return a
	// "collection item not found" error page instead of a generic 404.
	CollectionFallbackRoutes []CollectionFallbackRoute `json:"collectionFallbackRoutes,omitempty"`
}

// ManifestPageEntry contains all the pre-compiled, static metadata for a single
// routable page.
type ManifestPageEntry struct {
	// RoutePatterns maps locale identifiers to their Chi-compatible route
	// patterns. If empty, falls back to RoutePattern for backwards compatibility.
	RoutePatterns map[string]string `json:"routePatterns,omitempty"`

	// LocalTranslations holds translations defined in <i18n> blocks.
	// The runtime uses this to add to the global translation map for
	// requests to this page.
	LocalTranslations i18n_domain.Translations `json:"localTranslations,omitempty"`

	// CachePolicyFuncName is the original name of the CachePolicy function from
	// the source <script> block. It aids logging and debugging.
	CachePolicyFuncName string `json:"cachePolicyFuncName,omitempty"`

	// I18nStrategy stores the strategy ("prefix",
	// "prefix_except_default", "query-only", or "disabled") used to
	// generate RoutePatterns, enabling the runtime to understand how
	// routes were constructed.
	I18nStrategy string `json:"i18nStrategy,omitempty"`

	// StyleBlock is the complete, scoped, and minified CSS block for this page and
	// all its dependent partials. The runtime injects this into the response.
	StyleBlock string `json:"styleBlock,omitempty"`

	// JSArtefactIDs contains all client-side JavaScript artefact IDs
	// needed by this page, including the page's own script and the
	// transitive closure of all partial scripts embedded via
	// <partial is="..."> syntax.
	//
	// The runtime resolves each ID to a URL via the /_piko/assets/ route
	// and loads them all as ES modules. e.g.,
	// ["pk-js/pages/checkout.js", "pk-js/partials/cart.js"]
	JSArtefactIDs []string `json:"jsArtefactIds,omitempty"`

	// PackagePath is the canonical Go package path of the generated
	// page. The runtime uses this path to find and call the compiled
	// functions (BuildAST, Render, etc.), for example
	// "my-project/dist/pages/home_page_abcdef12".
	PackagePath string `json:"packagePath"`

	// MiddlewareFuncName is the name of the middleware function from the source
	// script block.
	MiddlewareFuncName string `json:"middlewareFuncName,omitempty"`

	// SupportedLocalesFuncName is the name of the function that returns supported
	// locales, taken from the source script block.
	SupportedLocalesFuncName string `json:"supportedLocalesFuncName,omitempty"`

	// AuthPolicyFuncName is the name of the AuthPolicy function from the source
	// script block.
	AuthPolicyFuncName string `json:"authPolicyFuncName,omitempty"`

	// OriginalSourcePath is the project-relative source file path, used for
	// display and debugging. For example, "pages/home.pk".
	OriginalSourcePath string `json:"originalSourcePath"`

	// AssetRefs lists static assets (such as SVGs) needed by the page. The runtime
	// probe step uses this to create link headers for early hints.
	AssetRefs []templater_dto.AssetRef `json:"assetRefs,omitempty"`

	// CustomTags lists custom element tags found in the template. The
	// frontend uses this to choose which components to hydrate.
	CustomTags []string `json:"customTags,omitempty"`

	// HasCachePolicy is true if the page script defines a CachePolicy function.
	// The runtime checks this to decide whether to call the function or use
	// a default policy.
	HasCachePolicy bool `json:"hasCachePolicy"`

	// HasMiddleware is true if the component script defines a Middlewares
	// function. The daemon uses this to wrap the page handler in custom
	// middleware.
	HasMiddleware bool `json:"hasMiddleware"`

	// HasSupportedLocales is true if the component script defines a
	// SupportedLocales function. Used by the i18n system at runtime.
	HasSupportedLocales bool `json:"hasSupportedLocales"`

	// HasAuthPolicy is true if the component script defines an AuthPolicy
	// function. The runtime checks this to enforce page-level auth requirements.
	HasAuthPolicy bool `json:"hasAuthPolicy,omitempty"`

	// IsE2EOnly indicates this page is from the e2e/ directory. E2E pages are
	// only served when Build.E2EMode is enabled; the runtime guard middleware
	// returns 404 for E2E pages when E2EMode is disabled.
	IsE2EOnly bool `json:"isE2EOnly,omitempty"`

	// HasPreview is true if the component script defines a Preview function
	// for dev-mode component previewing.
	HasPreview bool `json:"hasPreview,omitempty"`

	// UsesCaptcha indicates the template contains a piko:captcha element
	// and needs captcha provider scripts loaded at runtime.
	UsesCaptcha bool `json:"usesCaptcha,omitempty"`
}

// ManifestPartialEntry contains pre-compiled, static metadata for a single
// reusable partial.
type ManifestPartialEntry struct {
	// PackagePath is the Go package path for the generated partial.
	// For example, "my-project/dist/partials/card_component_abcdef12".
	PackagePath string `json:"packagePath"`

	// OriginalSourcePath is the path to the source file relative to the project
	// root, for example "partials/card.pk".
	OriginalSourcePath string `json:"originalSourcePath"`

	// PartialName is a unique identifier derived from the file path, used for
	// client-side hydration (e.g., "partials-card").
	PartialName string `json:"partialName"`

	// PartialSrc is the server-routable URL for fetching this partial's
	// client-side assets. It is also the route pattern for client-side
	// navigation; e.g., "/_piko/partial/partials-card".
	PartialSrc string `json:"partialSrc"`

	// RoutePattern is the URL route pattern for this partial. The Chi router uses
	// this value, which matches the PartialSrc value.
	RoutePattern string `json:"routePattern"`

	// StyleBlock holds the scoped and minified CSS for this partial and its
	// dependent partials. The runtime uses this for fragment renders.
	StyleBlock string `json:"styleBlock,omitempty"`

	// JSArtefactID is the registry artefact ID for this partial's client-side
	// JavaScript. Loaded when the partial is rendered as a fragment (e.g., via
	// HTMX).
	JSArtefactID string `json:"jsArtefactId,omitempty"`

	// IsE2EOnly is true if this partial comes from the e2e/ directory.
	// These partials are only served when Build.E2EMode is enabled at runtime.
	IsE2EOnly bool `json:"isE2EOnly,omitempty"`

	// HasPreview is true if the component script defines a Preview function
	// for dev-mode component previewing.
	HasPreview bool `json:"hasPreview,omitempty"`
}

// ManifestEmailEntry contains all the pre-compiled, static metadata for a
// single email template.
type ManifestEmailEntry struct {
	// LocalTranslations holds translations from <i18n> blocks in the email
	// template.
	LocalTranslations i18n_domain.Translations `json:"localTranslations,omitempty"`

	// PackagePath is the Go package path for the generated email component.
	PackagePath string `json:"packagePath"`

	// OriginalSourcePath is the path to the source file, relative to the project
	// root.
	OriginalSourcePath string `json:"originalSourcePath"`

	// StyleBlock is the full scoped and minified CSS for this email.
	StyleBlock string `json:"styleBlock,omitempty"`

	// HasSupportedLocales is true when the component's script defines a
	// SupportedLocales function.
	HasSupportedLocales bool `json:"hasSupportedLocales"`

	// HasPreview is true if the component script defines a Preview function
	// for dev-mode component previewing.
	HasPreview bool `json:"hasPreview,omitempty"`
}

// ManifestPdfEntry contains all the pre-compiled, static metadata for a
// single PDF template.
type ManifestPdfEntry struct {
	// LocalTranslations holds translations from <i18n> blocks in the PDF
	// template.
	LocalTranslations i18n_domain.Translations `json:"localTranslations,omitempty"`

	// PackagePath is the Go package path for the generated PDF component.
	PackagePath string `json:"packagePath"`

	// OriginalSourcePath is the path to the source file, relative to the project
	// root.
	OriginalSourcePath string `json:"originalSourcePath"`

	// StyleBlock is the full scoped and minified CSS for this PDF.
	StyleBlock string `json:"styleBlock,omitempty"`

	// HasSupportedLocales is true when the component's script defines a
	// SupportedLocales function.
	HasSupportedLocales bool `json:"hasSupportedLocales"`

	// HasPreview is true if the component script defines a Preview function
	// for dev-mode component previewing.
	HasPreview bool `json:"hasPreview,omitempty"`
}

// ManifestErrorPageEntry contains pre-compiled metadata for an error
// page that uses the ! prefix convention (e.g., !404.pk, !500.pk),
// supporting hierarchical scoping where a !404.pk in pages/app/ handles
// 404s for routes under /app/.
type ManifestErrorPageEntry struct {
	// PackagePath is the Go package path for the generated error page component.
	PackagePath string `json:"packagePath"`

	// OriginalSourcePath is the path to the source file relative to the project
	// root, for example "pages/!404.pk" or "pages/app/!500.pk".
	OriginalSourcePath string `json:"originalSourcePath"`

	// ScopePath is the URL path prefix this error page covers,
	// derived from the error page's directory relative to pages/.
	ScopePath string `json:"scopePath"`

	// StyleBlock is the scoped and minified CSS for this error page.
	StyleBlock string `json:"styleBlock,omitempty"`

	// JSArtefactIDs contains client-side JavaScript artefact IDs
	// needed by this error page.
	JSArtefactIDs []string `json:"jsArtefactIds,omitempty"`

	// CustomTags lists custom element tags found in the template.
	CustomTags []string `json:"customTags,omitempty"`

	// StatusCode is the HTTP status code this error page handles (e.g., 404, 500).
	// Zero for catch-all and range error pages.
	StatusCode int `json:"statusCode"`

	// StatusCodeMin is the lower bound of a range error page (e.g., 400 for
	// !400-499.pk). Zero when the page is not a range.
	StatusCodeMin int `json:"statusCodeMin,omitempty"`

	// StatusCodeMax is the upper bound of a range error page (e.g., 499 for
	// !400-499.pk). Zero when the page is not a range.
	StatusCodeMax int `json:"statusCodeMax,omitempty"`

	// IsCatchAll is true for !error.pk pages that handle all status codes.
	IsCatchAll bool `json:"isCatchAll,omitempty"`

	// IsE2EOnly indicates this error page is from the e2e/ directory.
	IsE2EOnly bool `json:"isE2EOnly,omitempty"`
}

// CollectionFallbackRoute holds the original dynamic route pattern for
// a static collection, preserving patterns like {slug} so the daemon
// can register a fallback route that returns a "collection item not
// found" error page instead of a generic 404.
type CollectionFallbackRoute struct {
	// RoutePatterns maps locale identifiers to their Chi-compatible route
	// patterns for this collection fallback.
	RoutePatterns map[string]string `json:"routePatterns,omitempty"`

	// I18nStrategy stores the strategy used to generate RoutePatterns.
	I18nStrategy string `json:"i18nStrategy,omitempty"`
}
