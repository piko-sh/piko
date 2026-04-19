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

package templater_domain

import (
	"context"
	"io"
	"net/http"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

// RenderPageParams holds all values needed to render a page or partial.
// Groups related fields to reduce the number of function arguments.
type RenderPageParams struct {
	// Writer is the destination for the rendered output.
	Writer io.Writer

	// ResponseWriter is the HTTP response writer for sending rendered output.
	ResponseWriter http.ResponseWriter

	// Request holds the HTTP request being rendered.
	Request *http.Request

	// TemplateAST is the parsed template to render; nil skips AST-based rendering.
	TemplateAST *ast_domain.TemplateAST

	// Metadata holds page data used for rendering and JSON output.
	Metadata *templater_dto.InternalMetadata

	// Config holds website settings used when rendering pages.
	Config *config.WebsiteConfig

	// ProbeData holds pre-fetched data from the probe phase. When non-nil, the
	// renderer reuses it instead of re-fetching component metadata.
	ProbeData *render_dto.ProbeData

	// PageDefinition contains the page metadata including paths used for
	// rendering.
	PageDefinition templater_dto.PageDefinition

	// Styling specifies the CSS or style settings for the rendered page.
	Styling string

	// IsFragment indicates whether to render a page fragment instead of a full
	// page.
	IsFragment bool
}

// RenderEmailParams holds the values needed to render an email template.
type RenderEmailParams struct {
	// Writer receives the rendered email output.
	Writer io.Writer

	// Request is the HTTP request used when rendering the email template.
	Request *http.Request

	// TemplateAST is the parsed template structure; nil means no template content.
	TemplateAST *ast_domain.TemplateAST

	// Metadata holds the data to be rendered into the template output.
	Metadata *templater_dto.InternalMetadata

	// PremailerOptions specifies CSS inlining settings; nil uses defaults.
	PremailerOptions *premailer.Options

	// PageID identifies the page for which the email is being rendered.
	PageID string

	// Styling contains CSS styles to apply to the rendered email.
	Styling string

	// IsPreviewMode indicates browser preview mode. When true, local image
	// paths are resolved to served asset URLs instead of CID references.
	IsPreviewMode bool
}

// ManifestStoreView provides read-only access to the runtime manifest.
// It lists all registered pages and partials, hiding the underlying storage.
type ManifestStoreView interface {
	// GetKeys returns a sorted list of all unique component source paths
	// in the manifest.
	//
	// Returns []string which contains paths such as "pages/home.pk".
	GetKeys() []string

	// GetPageEntry retrieves the unified view for a single component by its
	// original source path.
	//
	// Takes path (string) which is the original source path of the component.
	//
	// Returns PageEntryView which contains the unified component data.
	// Returns bool which is true if the entry was found, false otherwise.
	GetPageEntry(path string) (PageEntryView, bool)

	// FindErrorPage looks up the most specific error page for the given HTTP
	// status code and request path. Error pages are matched by scope: a !404.pk
	// in pages/app/ handles 404s for routes under /app/.
	//
	// Takes statusCode (int) which is the HTTP status code to find a page for.
	// Takes requestPath (string) which is the URL path being requested.
	//
	// Returns PageEntryView which is the matching error page entry.
	// Returns bool which is true if a matching error page was found.
	FindErrorPage(statusCode int, requestPath string) (PageEntryView, bool)

	// ListPreviewEntries returns all manifest entries that have a Preview
	// function defined. Used by the dev API to build the preview catalogue.
	//
	// Returns []PreviewCatalogueEntry which contains the previewable templates.
	ListPreviewEntries() []PreviewCatalogueEntry
}

// PreviewCatalogueEntry represents a template that has a Preview function,
// used by the dev API to build the preview catalogue.
type PreviewCatalogueEntry struct {
	// OriginalSourcePath is the source path of the template (e.g.,
	// "emails/welcome.pk").
	OriginalSourcePath string

	// ComponentType is the kind of component: "page", "partial", "email",
	// or "pdf".
	ComponentType string

	// Scenarios holds the preview scenarios returned by the Preview function.
	Scenarios []templater_dto.PreviewScenario
}

// ManifestRunnerPort provides access to the compiled template manifest,
// enabling execution of pages and partials. It manages the lifecycle of
// template ASTs and their metadata.
type ManifestRunnerPort interface {
	// RunPage renders a page from the given definition and request.
	//
	// Takes pageDefinition (templater_dto.PageDefinition) which describes the page
	// to render.
	// Takes request (*http.Request) which provides the HTTP request context.
	//
	// Returns *ast_domain.TemplateAST which is the parsed template structure.
	// Returns templater_dto.InternalMetadata which contains rendering metadata.
	// Returns string which is the rendered page content.
	// Returns error when rendering fails.
	RunPage(ctx context.Context, pageDefinition templater_dto.PageDefinition, request *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)

	// RunPartial renders a partial template from the given page definition.
	//
	// Takes pageDefinition (templater_dto.PageDefinition) which defines the
	// template to render.
	// Takes request (*http.Request) which provides the HTTP request context.
	//
	// Returns *ast_domain.TemplateAST which is the parsed template structure.
	// Returns templater_dto.InternalMetadata which contains rendering metadata.
	// Returns string which is the rendered output.
	// Returns error when rendering fails.
	RunPartial(ctx context.Context, pageDefinition templater_dto.PageDefinition, request *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)

	// RunPartialWithProps runs a partial and passes props data through to the
	// compiled template.
	//
	// Takes pageDefinition (templater_dto.PageDefinition) which specifies the
	// partial to render.
	// Takes request (*http.Request) which provides the HTTP request context.
	// Takes props (any) which contains data to pass to the template.
	//
	// Returns *ast_domain.TemplateAST which is the compiled template tree.
	// Returns templater_dto.InternalMetadata which contains rendering metadata.
	// Returns string which is the rendered output.
	// Returns error when the partial cannot be compiled or rendered.
	RunPartialWithProps(ctx context.Context, pageDefinition templater_dto.PageDefinition, request *http.Request, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)

	// GetPageEntry retrieves a page entry view by its manifest key.
	//
	// Takes manifestKey (string) which identifies the page entry to retrieve.
	//
	// Returns PageEntryView which contains the page entry data.
	// Returns error when the page entry cannot be found or retrieved.
	GetPageEntry(ctx context.Context, manifestKey string) (PageEntryView, error)
}

// RendererPort defines the interface for rendering templates to HTML output.
// It handles pages, partials, and email templates.
type RendererPort interface {
	// CollectMetadata gathers metadata from the request and configuration.
	//
	// Takes request (*http.Request) which provides the incoming request data.
	// Takes metadata (*templater_dto.InternalMetadata) which stores collected
	// metadata.
	// Takes websiteConfig (*config.WebsiteConfig) which provides website settings.
	//
	// Returns []render_dto.LinkHeader which contains link headers for the
	// response.
	// Returns *render_dto.ProbeData which holds pre-fetched data for the
	// render phase.
	// Returns error when metadata collection fails.
	CollectMetadata(
		ctx context.Context,
		request *http.Request,
		metadata *templater_dto.InternalMetadata,
		websiteConfig *config.WebsiteConfig,
	) ([]render_dto.LinkHeader, *render_dto.ProbeData, error)

	// RenderPage renders a page using the provided parameters.
	//
	// Takes params (RenderPageParams) which specifies the page rendering options.
	//
	// Returns error when rendering fails.
	RenderPage(ctx context.Context, params RenderPageParams) error

	// RenderPartial renders a partial page using the given parameters.
	//
	// Takes params (RenderPageParams) which specifies the page rendering options.
	//
	// Returns error when rendering fails.
	RenderPartial(ctx context.Context, params RenderPageParams) error

	// RenderEmail creates an email from the given template and data.
	//
	// Takes params (RenderEmailParams) which holds the email template settings.
	//
	// Returns error when the email cannot be created.
	RenderEmail(ctx context.Context, params RenderEmailParams) error

	// RenderASTToPlainText converts a template AST into plain text.
	//
	// Takes templateAST (*ast_domain.TemplateAST) which is the parsed template to
	// render.
	//
	// Returns string which is the rendered plain text output.
	// Returns error when rendering fails.
	RenderASTToPlainText(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error)

	// GetLastEmailAssetRequests returns the most recent email asset requests.
	//
	// Returns []*EmailAssetRequest which contains the last batch of asset
	// requests.
	GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest
}

// RenderRequest holds all values needed to render a page or partial via
// TemplaterService.
type RenderRequest struct {
	// Writer receives the rendered output.
	Writer io.Writer

	// Response provides HTTP response controls.
	Response http.ResponseWriter

	// Request contains the incoming HTTP request data.
	Request *http.Request

	// WebsiteConfig provides website settings used during rendering.
	WebsiteConfig *config.WebsiteConfig

	// ProbeData holds pre-fetched data from the probe phase.
	ProbeData *render_dto.ProbeData

	// Page defines the page or partial to render.
	Page templater_dto.PageDefinition

	// IsFragment indicates whether to render a partial page fragment.
	IsFragment bool
}

// TemplaterService is the main orchestration service for template rendering. It
// coordinates between the manifest runner and renderer to produce final HTML
// output.
type TemplaterService interface {
	// ProbePage probes a page to gather metadata and
	// validation information.
	//
	// Takes page (templater_dto.PageDefinition) which
	// defines the page to probe.
	// Takes request (*http.Request) which provides the HTTP
	// request context.
	// Takes websiteConfig (*config.WebsiteConfig) which
	// specifies the website settings.
	//
	// Returns *templater_dto.PageProbeResult which contains
	// the probe findings.
	// Returns error when the probe fails.
	ProbePage(ctx context.Context, page templater_dto.PageDefinition, request *http.Request, websiteConfig *config.WebsiteConfig) (*templater_dto.PageProbeResult, error)

	// RenderPage renders a page template to the given writer.
	//
	// Takes request (RenderRequest) which bundles all values needed for rendering.
	//
	// Returns error when rendering fails.
	RenderPage(ctx context.Context, request RenderRequest) error

	// ProbePartial probes a page definition for partial
	// template resolution.
	//
	// Takes page (templater_dto.PageDefinition) which
	// defines the page to probe.
	// Takes request (*http.Request) which provides the HTTP
	// request context.
	// Takes websiteConfig (*config.WebsiteConfig) which
	// specifies the website settings.
	//
	// Returns *templater_dto.PageProbeResult which contains
	// the probe findings.
	// Returns error when probing fails.
	ProbePartial(ctx context.Context, page templater_dto.PageDefinition, request *http.Request, websiteConfig *config.WebsiteConfig) (*templater_dto.PageProbeResult, error)

	// RenderPartial renders a partial page template to the provided writer.
	//
	// Takes request (RenderRequest) which bundles all values needed for rendering.
	//
	// Returns error when rendering fails.
	RenderPartial(ctx context.Context, request RenderRequest) error

	// SetRunner assigns the manifest runner implementation for executing checks.
	//
	// Takes r (ManifestRunnerPort) which provides the runner to use.
	SetRunner(r ManifestRunnerPort)
}

// EmailTemplateService renders templates for email delivery. It implements
// templater_domain.EmailTemplateService and produces HTML with inlined CSS
// and plain text alternatives optimised for email clients.
type EmailTemplateService interface {
	// Render runs the full Piko pipeline for a single component.
	//
	// Takes request (*http.Request) which provides the HTTP context for rendering.
	// Takes templatePath (string) which is the path to the template file.
	// Takes props (any) which contains the data to pass to the template.
	// Takes premailerOptions (*premailer.Options) which configures CSS inlining.
	// Takes isPreviewMode (bool) which when true resolves images to served URLs
	// instead of CID references.
	//
	// Returns *templater_dto.RenderedEmailContent which contains the rendered HTML
	// and CSS, ready for email inlining.
	// Returns error when rendering or CSS processing fails.
	Render(
		ctx context.Context,
		request *http.Request,
		templatePath string,
		props any,
		premailerOptions *premailer.Options,
		isPreviewMode bool,
	) (*templater_dto.RenderedEmailContent, error)
}

// PageEntryView provides a unified, read-only view of a compiled page or
// partial. It exposes all metadata, rendering functions, and configuration
// needed to execute the template.
type PageEntryView interface {
	// GetHasMiddleware reports whether the handler has middleware attached.
	GetHasMiddleware() bool

	// GetMiddlewareFuncName returns the name of the middleware function.
	//
	// Returns string which is the middleware function name.
	GetMiddlewareFuncName() string

	// GetHasCachePolicy reports whether the directive has a cache policy set.
	//
	// Returns bool which is true if a cache policy exists, false otherwise.
	GetHasCachePolicy() bool

	// GetCachePolicy returns the cache policy for the given request.
	//
	// Takes r (*templater_dto.RequestData) which contains the request details.
	//
	// Returns templater_dto.CachePolicy which specifies how the response should
	// be cached.
	GetCachePolicy(r *templater_dto.RequestData) templater_dto.CachePolicy

	// GetCachePolicyFuncName returns the name of the cache policy function.
	//
	// Returns string which is the function name used for cache policy choices.
	GetCachePolicyFuncName() string

	// GetMiddlewares returns the middleware chain for the router.
	//
	// Returns []func(http.Handler) http.Handler which contains the middleware
	// functions to wrap handlers.
	GetMiddlewares() []func(http.Handler) http.Handler

	// GetHasAuthPolicy reports whether the page declares auth requirements.
	GetHasAuthPolicy() bool

	// GetAuthPolicy returns the auth policy for the given request.
	//
	// Takes r (*templater_dto.RequestData) which contains the request details.
	//
	// Returns daemon_dto.AuthPolicy which specifies the auth requirements.
	GetAuthPolicy(r *templater_dto.RequestData) daemon_dto.AuthPolicy

	// GetIsPage reports whether the element represents a page.
	GetIsPage() bool

	// GetRoutePattern returns the first route pattern for partials and logging.
	//
	// Returns string which is the primary pattern from RoutePatterns.
	GetRoutePattern() string

	// GetRoutePatterns returns the route patterns for the HTTP handler.
	//
	// Returns map[string]string which maps route names to their URL patterns.
	GetRoutePatterns() map[string]string

	// GetI18nStrategy returns the internationalisation strategy identifier.
	//
	// Returns string which specifies the i18n handling approach for this rule.
	GetI18nStrategy() string

	// GetOriginalPath returns the original file path before any changes.
	//
	// Returns string which is the unchanged path.
	GetOriginalPath() string

	// GetASTRoot retrieves the root AST node for the given request data.
	//
	// Takes *templater_dto.RequestData which contains the template request
	// details.
	//
	// Returns *ast_domain.TemplateAST which is the parsed template syntax tree.
	// Returns templater_dto.InternalMetadata which provides processing metadata.
	GetASTRoot(*templater_dto.RequestData) (*ast_domain.TemplateAST, templater_dto.InternalMetadata)

	// GetASTRootWithProps returns the AST root with props for email rendering
	// and similar use-cases.
	//
	// Takes *templater_dto.RequestData which provides the request context.
	// Takes any which supplies the props for template rendering.
	//
	// Returns *ast_domain.TemplateAST which is the parsed template tree.
	// Returns templater_dto.InternalMetadata which contains rendering metadata.
	GetASTRootWithProps(*templater_dto.RequestData, any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata)

	// GetStyling returns the styling settings as a string.
	GetStyling() string

	// GetAssetRefs returns the asset references associated with this entity.
	//
	// Returns []templater_dto.AssetRef which contains the asset references.
	GetAssetRefs() []templater_dto.AssetRef

	// GetCustomTags returns the list of custom tags that may appear in
	// documentation comments.
	//
	// Returns []string which contains the recognised custom tag names.
	GetCustomTags() []string

	// GetSupportedLocales returns the list of locale codes that this provider
	// supports.
	//
	// Returns []string which contains the supported locale codes.
	GetSupportedLocales() []string

	// GetLocalStore returns the pre-built translation Store for this page's
	// local translations. Returns nil if the page has no local translations.
	//
	// Returns *i18n_domain.Store which provides pre-parsed lookups for
	// component-scoped translations.
	GetLocalStore() *i18n_domain.Store

	// GetJSScriptMetas returns metadata for all client-side JavaScript modules
	// needed by this page.
	//
	// Returns []JSScriptMeta which contains the page's own script (if any) plus
	// scripts from all embedded partials. Each entry contains the URL and optional
	// partial name for frontend function scoping. Returns nil if there are no
	// client scripts.
	GetJSScriptMetas() []templater_dto.JSScriptMeta

	// GetIsE2EOnly reports whether this entry is from the e2e/ directory.
	// E2E components are only served when Build.E2EMode is enabled at runtime.
	//
	// Returns bool which is true if this is an E2E-only component.
	GetIsE2EOnly() bool

	// GetStaticMetadata returns a pointer to pre-computed static metadata
	// (AssetRefs, CustomTags, SupportedLocales) to avoid per-request allocations.
	// The caller MUST NOT modify this data.
	//
	// Returns *templater_dto.InternalMetadata which contains the cached static
	// metadata for probe operations.
	GetStaticMetadata() *templater_dto.InternalMetadata

	// GetHasPreview reports whether this component defines a Preview function
	// for dev-mode previewing.
	GetHasPreview() bool

	// GetPreviewScenarios returns the preview scenarios for this component.
	// Returns nil if the component has no Preview function.
	GetPreviewScenarios() []templater_dto.PreviewScenario
}
