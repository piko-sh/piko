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

package render_domain

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"piko.sh/piko/internal/json"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/render/render_templates"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// profileKeyFormat is the key for image format in profile parameters and tags.
	profileKeyFormat = "format"

	// profileKeyWidth is the key for the image width in profile parameters.
	profileKeyWidth = "width"

	// profileKeyDensity is the key for setting image density in profile
	// parameters.
	profileKeyDensity = "density"

	// decimalBase is base 10, used when formatting numbers as strings.
	decimalBase = 10

	// tagStorageBackendID is the tag key for the storage backend ID.
	tagStorageBackendID = "storageBackendId"

	// tagType is the tag key for the image variant type.
	tagType = "type"

	// tagFileExtension is the tag name for the file extension of an output file.
	tagFileExtension = "fileExtension"

	// tagMimeType is the tag name for the MIME type of an asset variant.
	tagMimeType = "mimeType"

	// defaultStorageBackend is the storage backend used for image variant tags.
	defaultStorageBackend = "local_disk_cache"

	// defaultImageVariantType is the default type tag for image variant profiles.
	defaultImageVariantType = "image-variant"

	// defaultImageExtension is the file extension used for image variants.
	defaultImageExtension = ".img"

	// renderContextMapCapacity is the initial capacity for render context maps
	// tracking components, hints, symbols, and tags.
	renderContextMapCapacity = 16

	// renderContextCacheCapacity is the starting size for smaller render context
	// caches such as registered assets and srcset caches.
	renderContextCacheCapacity = 8

	// byteBufferInitialCapacity is the initial capacity for byte buffers used
	// during rendering.
	byteBufferInitialCapacity = 256

	// bufioWriterBufferSize is the buffer size for pooled bufio.Writer instances.
	// 32KB balances memory usage against syscall overhead for typical page sizes
	// (100-200KB), reducing ResponseWriter.Write calls from hundreds to around five
	// per page and removing mutex contention in net/http.
	bufioWriterBufferSize = 32 * 1024
)

var (
	// byteBufferPool reuses byte-slice buffers to reduce allocation pressure
	// during rendering.
	byteBufferPool = sync.Pool{
		New: func() any {
			return new(make([]byte, 0, byteBufferInitialCapacity))
		},
	}

	// csrfBufPool provides reusable bytes.Buffer instances for CSRF token
	// generation. The buffer is owned by renderContext for the request duration.
	csrfBufPool = sync.Pool{
		New: func() any {
			b := &bytes.Buffer{}
			b.Grow(128)
			return b
		},
	}

	// bufioWriterPool provides reusable buffered writers to reduce allocations
	// during rendering. Each writer has a 32KB buffer that batches small writes
	// before flushing to the underlying ResponseWriter.
	bufioWriterPool = sync.Pool{
		New: func() any {
			return bufio.NewWriterSize(nil, bufioWriterBufferSize)
		},
	}

	// selfClosingSVGElements holds the set of SVG element names that are rendered
	// as self-closing tags.
	selfClosingSVGElements = map[string]bool{
		"circle": true, "ellipse": true, "feBlend": true, "feColorMatrix": true, "feComponentTransfer": true,
		"feComposite": true, "feConvolveMatrix": true, "feDiffuseLighting": true, "feDisplacementMap": true,
		"feDistantLight": true, "feFlood": true, "feFuncA": true, "feFuncB": true, "feFuncG": true,
		"feFuncR": true, "feGaussianBlur": true, "feImage": true, "feMergeNode": true, "feMorphology": true,
		"feOffset": true, "fePointLight": true, "feSpecularLighting": true, "feSpotLight": true,
		"feTile": true, "feTurbulence": true, "image": true, "line": true, "path": true, "polygon": true,
		"polyline": true, "rect": true, "stop": true, "use": true, "view": true,
	}

	// openBracket is the pre-computed byte representation of "<".
	openBracket = []byte{'<'}

	// closeBracket is the pre-computed byte representation of ">".
	closeBracket = []byte{'>'}

	// space is the pre-computed byte representation of " ".
	space = []byte{' '}

	// quote is the pre-computed byte representation of a double-quote character.
	quote = []byte{'"'}

	// equalsQuote is the pre-computed byte representation of `="`.
	equalsQuote = []byte{'=', '"'}

	// selfClose is the pre-computed byte representation of " />".
	selfClose = []byte{' ', '/', '>'}

	// closeTagPrefix is the pre-computed byte representation of "</".
	closeTagPrefix = []byte{'<', '/'}

	// dot is the pre-computed byte representation of ".".
	dot = []byte{'.'}

	// pOnPrefix is the pre-computed byte representation of the p-on: attribute prefix.
	pOnPrefix = []byte(` p-on:`)

	// pEventPrefix is the pre-computed byte representation of the p-event: attribute prefix.
	pEventPrefix = []byte(` p-event:`)

	// pRefPrefix is the pre-computed byte representation of the p-ref attribute opening.
	pRefPrefix = []byte(` p-ref="`)
	// commentOpen is the pre-computed byte representation of "<!--".
	commentOpen = []byte("<!--")
	// commentClose is the pre-computed byte representation of "-->".
	commentClose = []byte("-->")
	// csrfEphemeralAttrName is the pre-computed byte representation of the CSRF
	// ephemeral token attribute name.
	csrfEphemeralAttrName = []byte("data-csrf-ephemeral-token")
	// csrfActionAttrName is the pre-computed byte representation of the CSRF
	// action token attribute name.
	csrfActionAttrName = []byte("data-csrf-action-token")
	// renderContextPool reuses renderContext instances to reduce allocation
	// pressure during page rendering.
	renderContextPool = sync.Pool{
		New: func() any {
			return new(newRenderContextFields())
		},
	}
)

// RenderASTOptions contains all options for rendering an AST to HTML output.
// Using an options struct reduces the argument count and makes the API more
// flexible.
type RenderASTOptions struct {
	// Template is the parsed AST to render. It is returned to the pool after use.
	Template *ast_domain.TemplateAST

	// Metadata holds page details such as title, description, and canonical URL.
	Metadata *templater_dto.InternalMetadata

	// SiteConfig holds website settings for fonts and favicons; nil skips these.
	SiteConfig *config.WebsiteConfig

	// ProbeData holds pre-fetched data from the probe phase. When non-nil,
	// RenderAST reuses it instead of fetching component metadata again.
	ProbeData *render_dto.ProbeData

	// PageID identifies the page for render context and error messages.
	PageID string

	// Styling specifies the CSS styles to apply to the rendered output.
	Styling string

	// IsFragment indicates whether to render a page fragment instead of a full
	// page.
	IsFragment bool
}

// RenderEmailOptions contains all options for rendering an email from an AST.
type RenderEmailOptions struct {
	// Template is the parsed template AST; nil uses a default template.
	Template *ast_domain.TemplateAST

	// Metadata holds the email metadata used when rendering the template.
	Metadata *templater_dto.InternalMetadata

	// PremailerOptions sets the CSS inlining settings; nil uses defaults.
	PremailerOptions *premailer.Options

	// PageID is the unique identifier for the email page to render.
	PageID string

	// Styling specifies the CSS styles to apply during email rendering.
	Styling string

	// IsPreviewMode indicates browser preview mode. When true, local image
	// paths are resolved to served asset URLs instead of CID references.
	IsPreviewMode bool
}

// linkHeaderKey identifies a link header for fast deduplication in a map.
// It uses URL, Rel, and As fields for comparison.
type linkHeaderKey struct {
	// URL is the target address from the link header.
	URL string

	// Rel is the link relation type.
	Rel string

	// As is the string form of this link header key.
	As string
}

// svgSymbolEntry pairs an SVG artefact ID with its pre-fetched data. Storing
// the data at registration time avoids a redundant cache lookup in
// buildSvgSpriteSheet.
type svgSymbolEntry struct {
	// data holds the pre-fetched SVG content for this symbol.
	data *ParsedSvgData

	// id is the artefact identifier for the SVG symbol.
	id string
}

// renderContext holds per-request state used throughout the rendering pipeline.
type renderContext struct {
	// httpResponse is the HTTP response writer used to set CSRF cookies.
	httpResponse http.ResponseWriter

	// registry provides access to component definitions and asset data.
	registry RegistryPort

	// csrfService generates CSRF token pairs to protect forms.
	csrfService security_domain.CSRFTokenService

	// csrfError holds any error from CSRF token generation; nil means success.
	csrfError error

	// originalCtx is the original request context for cancellation and deadlines.
	originalCtx context.Context

	// linkHeaderSet tracks which link headers have been added to prevent
	// duplicates.
	linkHeaderSet map[linkHeaderKey]struct{}

	// customTags holds the set of custom tag names that may appear in the
	// template.
	customTags map[string]struct{}

	// mergedAttrsCache stores merged SVG attribute strings for fast lookups.
	mergedAttrsCache map[svgCacheKey]string

	// registeredDynamicAssets caches assets that are registered during rendering.
	registeredDynamicAssets map[string]*registry_dto.ArtefactMeta

	// srcsetCache stores precomputed srcset attribute strings for piko:img
	// elements. The key is the artefact ID and profile hash.
	srcsetCache map[srcsetCacheKey]string

	// httpRequest holds the current HTTP request, used for CSRF token generation.
	httpRequest *http.Request

	// requiredSvgSymbols tracks which SVG symbols are needed for the sprite
	// sheet, together with their pre-fetched data from the registry cache.
	requiredSvgSymbols []svgSymbolEntry

	// csrfPair holds the CSRF token pair; nil until first needed.
	csrfPair *security_dto.CSRFPair

	// csrfBuf holds a pooled buffer for CSRF token generation.
	// CSRFPair.ActionToken is a slice into this buffer; the buffer is returned to
	// the pool on reset.
	csrfBuf *bytes.Buffer

	// collectedCustomComponents tracks custom component tags used during
	// rendering.
	collectedCustomComponents map[string]struct{}

	// probeData holds pre-fetched data from the probe phase. When non-nil,
	// putRenderContext releases it back to the pool.
	probeData *render_dto.ProbeData

	// componentMetadata stores bulk-fetched component metadata from the
	// preload phase so buildPreloadLogic can reuse it without a second fetch.
	componentMetadata map[string]*render_dto.ComponentMetadata

	// defaultLocale is the site's default locale; empty means no default is set.
	defaultLocale string

	// pageID identifies the page being rendered for error messages and
	// diagnostics.
	pageID string

	// currentLocale holds the locale for the current request; empty means not set.
	currentLocale string

	// i18nStrategy specifies how to handle localisation; empty or "disabled"
	// means no localisation, "prefix" adds the locale to URL paths.
	i18nStrategy string

	// diagnostics gathers errors and warnings found during rendering.
	diagnostics renderDiagnostics

	// frozenBuffers holds buffers that were turned into strings without copying.
	// These buffers must stay in memory until the request ends so the strings
	// remain valid.
	frozenBuffers []*[]byte

	// collectedLinkHeaders holds Link headers for preloading resources.
	collectedLinkHeaders []render_dto.LinkHeader

	// csrfOnce guards single execution of CSRF token generation per request.
	csrfOnce sync.Once

	// muCollectedLinkHeaders protects access to collectedLinkHeaders.
	muCollectedLinkHeaders sync.Mutex

	// muDiagnostics guards the diagnostics field during concurrent writes.
	muDiagnostics sync.Mutex

	// skipPrerenderedHTML when true causes renderNode to ignore PrerenderedHTML
	// and walk the full AST structure. Used for email rendering where the AST
	// must remain intact for CSS inlining and PML transformations.
	skipPrerenderedHTML bool

	// isEmailMode indicates whether email rendering is active. When true,
	// internal attributes like p-key and partial are removed from output.
	isEmailMode bool

	// stripHTMLComments controls whether HTML comments are removed from the
	// output.
	stripHTMLComments bool
}

// RenderOrchestrator coordinates the rendering of HTML from AST structures,
// managing transformations, component resolution, CSRF token generation, and
// SVG sprite sheet creation. It implements RenderService, StaticPrerenderer,
// and HealthProbe interfaces.
type RenderOrchestrator struct {
	// pmlEngine transforms PML templates into HTML for email output.
	pmlEngine pml_domain.Transformer

	// registry provides access to component metadata and asset loading.
	registry RegistryPort

	// csrfService creates and validates CSRF tokens.
	csrfService security_domain.CSRFTokenService

	// dynamicAssetCache stores artefact metadata across requests, keyed by
	// asset path. Profiles are deterministic from static template attributes,
	// so a registered artefact never changes during this process's lifetime.
	dynamicAssetCache sync.Map

	// cssResetCSS holds the CSS reset to prepend to theme output. When empty,
	// no CSS reset is included in the generated theme CSS.
	cssResetCSS string

	// transformSteps holds the registered transformation steps in the
	// pipeline.
	transformSteps []TransformationPort

	// lastEmailAssetRequests holds asset requests from the most recent email
	// render.
	lastEmailAssetRequests []*email_dto.EmailAssetRequest

	// stripHTMLComments controls whether HTML comments are removed from the
	// output.
	stripHTMLComments bool
}

// RenderOrchestratorOption configures a RenderOrchestrator.
type RenderOrchestratorOption func(*RenderOrchestrator)

// NewRenderOrchestrator creates a new render orchestrator with the provided
// dependencies.
//
// Takes pmlEngine (pml_domain.Transformer) which handles PML transformation.
// Takes transforms ([]TransformationPort) which defines the transformation
// steps to apply.
// Takes registry (RegistryPort) which provides access to registered components.
// Takes csrfService (security_domain.CSRFTokenService) which generates CSRF
// tokens for security.
// Takes opts (...RenderOrchestratorOption) which provides optional
// configuration.
//
// Returns *RenderOrchestrator which is ready to orchestrate email rendering.
func NewRenderOrchestrator(
	pmlEngine pml_domain.Transformer,
	transforms []TransformationPort,
	registry RegistryPort,
	csrfService security_domain.CSRFTokenService,
	opts ...RenderOrchestratorOption,
) *RenderOrchestrator {
	ro := &RenderOrchestrator{
		pmlEngine:              pmlEngine,
		transformSteps:         transforms,
		registry:               registry,
		csrfService:            csrfService,
		lastEmailAssetRequests: nil,
	}
	for _, opt := range opts {
		opt(ro)
	}
	return ro
}

// RenderAST renders an AST into HTML and writes it to the provided writer.
// It handles both full pages and fragments, manages component preloading,
// generates SVG sprite sheets, and applies i18n transformations.
//
// Takes w (io.Writer) which receives the rendered HTML output.
// Takes response (http.ResponseWriter) which provides response headers and state.
// Takes request (*http.Request) which supplies request context for rendering.
// Takes opts (RenderASTOptions) which configures rendering behaviour.
//
// Returns error when rendering fails for fragments or full pages.
//
// This function intentionally does not create its own span. The span
// should be created at the templater level (TemplaterService.RenderPage or
// RenderPartial) where it provides meaningful user-facing observability.
// This function is a low-level framework internal that should be
// "invisible fast."
func (ro *RenderOrchestrator) RenderAST(
	ctx context.Context,
	w io.Writer,
	response http.ResponseWriter,
	request *http.Request,
	opts RenderASTOptions,
) error {
	ctx, l := logger_domain.From(ctx, log)

	startTime := time.Now()
	RenderASTCount.Add(ctx, 1)
	renderCtx := ro.getRenderContext(ctx, opts.PageID, response, request, opts.SiteConfig)

	customTags := appendDevWidgetTag(opts.Metadata.CustomTags)
	populateTagMap(renderCtx.customTags, customTags)

	detectCapabilitiesFromAST(opts)
	if opts.ProbeData != nil && opts.ProbeData.ComponentMetadata != nil {
		renderCtx.componentMetadata = opts.ProbeData.ComponentMetadata
		renderCtx.probeData = opts.ProbeData
	} else {
		ro.ensureComponentMetadata(ctx, customTags, renderCtx)
	}
	preloadHTML, scriptHTML := ro.buildPreloadLogic(
		customTags, false, renderCtx)

	bufWriter := getBufferedWriter(w)
	defer releaseBufferedWriter(bufWriter)
	defer func() {
		if flushErr := bufWriter.Flush(); flushErr != nil {
			l.Warn("flushing buffered writer during render", logger_domain.Error(flushErr))
		}
	}()
	qw := qt.AcquireWriter(bufWriter)

	var renderErr error
	if opts.IsFragment {
		renderErr = ro.renderFragment(ctx, qw, renderCtx, opts, scriptHTML)
	} else {
		renderErr = ro.renderFullPage(ctx, qw, renderCtx, opts, preloadHTML, scriptHTML)
	}

	qt.ReleaseWriter(qw)
	if opts.Template != nil {
		ast_domain.PutTree(opts.Template)
	}

	if renderErr != nil {
		ro.putRenderContext(renderCtx)
		RenderASTDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
		RenderASTErrorCount.Add(ctx, 1)
		return renderErr
	}

	logCollectedDiagnostics(ctx, renderCtx)
	ro.putRenderContext(renderCtx)
	RenderASTDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	return nil
}

// RenderASTToPlainText converts an AST to plain text format. Use it to
// generate email plain-text alternatives.
//
// Takes templateAST (*ast_domain.TemplateAST) which is the parsed template to
// render.
//
// Returns string which is the plain text representation of the template.
// Returns error when the walking process fails.
func (*RenderOrchestrator) RenderASTToPlainText(
	_ context.Context,
	templateAST *ast_domain.TemplateAST,
) (string, error) {
	if templateAST == nil {
		return "", nil
	}

	walker := newPlainTextWalker()
	return walker.Walk(templateAST)
}

// GetLastEmailAssetRequests returns the asset requests collected during the
// most recent email rendering operation. This method should be called
// immediately after RenderEmail to retrieve assets that need to be embedded
// via Content-ID (CID).
//
// Returns []*email_dto.EmailAssetRequest which contains the asset requests.
// Returns an empty slice if no email has been rendered or if no assets were
// requested. The returned slice is not a copy; callers should not modify it.
func (ro *RenderOrchestrator) GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest {
	if ro.lastEmailAssetRequests == nil {
		return []*email_dto.EmailAssetRequest{}
	}
	return ro.lastEmailAssetRequests
}

// RenderASTToString renders an AST to an HTML string without requiring HTTP
// context. This is designed for headless rendering scenarios like WASM,
// testing, and static site generation.
//
// Unlike RenderAST, this method:
//   - Does not require http.Request or http.ResponseWriter
//   - Skips CSRF token generation
//   - Does not collect Link headers
//   - Does not extract locale from request headers
//
// Takes opts (RenderASTToStringOptions) which configures the rendering.
//
// Returns string which contains the rendered HTML.
// Returns error when rendering fails.
func (ro *RenderOrchestrator) RenderASTToString(
	ctx context.Context,
	opts RenderASTToStringOptions,
) (string, error) {
	if opts.Template == nil {
		return "", nil
	}

	rctx := ro.getHeadlessRenderContext(ctx)
	defer ro.putRenderContext(rctx)

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	if opts.IncludeDocumentWrapper {
		if err := ro.renderHeadlessFullPage(ctx, qw, rctx, opts); err != nil {
			return "", fmt.Errorf("rendering headless full page: %w", err)
		}
	} else {
		if err := ro.renderASTToWriter(opts.Template, qw, rctx); err != nil {
			return "", fmt.Errorf("rendering AST: %w", err)
		}
	}

	if opts.Template != nil {
		ast_domain.PutTree(opts.Template)
	}

	return buffer.String(), nil
}

// renderFragment renders a fragment page with header, content, and footer.
//
// Takes qw (*qt.Writer) which receives the rendered output.
// Takes rctx (*renderContext) which provides the rendering state.
// Takes opts (RenderASTOptions) which specifies page metadata and styling.
// Takes scriptHTML (string) which contains module script tags to inject.
//
// Returns error when streaming the AST content fails.
func (ro *RenderOrchestrator) renderFragment(
	ctx context.Context,
	qw *qt.Writer,
	rctx *renderContext,
	opts RenderASTOptions,
	scriptHTML string,
) error {
	csrfPair := ro.ensureCSRFForMeta(rctx)

	data := &render_templates.FragmentPageData{
		Title:           opts.Metadata.Title,
		Description:     opts.Metadata.Description,
		CanonicalURL:    opts.Metadata.CanonicalURL,
		PageID:          opts.PageID,
		ModuleScripts:   scriptHTML,
		RenderedContent: "",
		Styling:         opts.Styling,
		SvgSpriteSheet:  "",
		PKScriptMetas:   opts.Metadata.JSScriptMetas,
		AlternateLinks:  opts.Metadata.AlternateLinks,
		MetaTags:        opts.Metadata.MetaTags,
		OGTags:          opts.Metadata.OGTags,
		TwitterCards:    opts.Metadata.TwitterCards,
		StructuredData:  filterValidJSON(ctx, opts.Metadata.StructuredData),
	}

	if csrfPair != nil {
		data.CSRFActionToken = csrfPair.ActionToken
		data.CSRFEphemeralToken = csrfPair.RawEphemeralToken
	}
	render_templates.StreamFragmentPageHeader(qw, data)
	if err := ro.renderASTToWriter(opts.Template, qw, rctx); err != nil {
		return fmt.Errorf("streaming AST for fragment %s: %w", opts.PageID, err)
	}
	data.SvgSpriteSheet = ro.buildSvgSpriteSheetIfNeeded(ctx, rctx)
	render_templates.StreamFragmentPageFooter(qw, data)
	return nil
}

// renderFullPage renders a complete HTML page with header, content, and footer.
//
// Takes qw (*qt.Writer) which receives the streamed HTML output.
// Takes rctx (*renderContext) which provides the rendering context and state.
// Takes opts (RenderASTOptions) which specifies page metadata and
// configuration.
// Takes preloadHTML (string) which contains preload link tags for the page.
// Takes scriptHTML (string) which contains script tags to include.
//
// Returns error when streaming the AST content fails.
func (ro *RenderOrchestrator) renderFullPage(
	ctx context.Context,
	qw *qt.Writer,
	rctx *renderContext,
	opts RenderASTOptions,
	preloadHTML, scriptHTML string,
) error {
	var fonts, favicons []byte
	if opts.SiteConfig != nil {
		fonts = ro.buildFontLinks(opts.SiteConfig)
		favicons = ro.buildFaviconLinks(opts.SiteConfig)
	}

	modulePreloadHTML := getModulePreloadHTML()
	moduleConfigHTML := getModuleConfigHTML()
	moduleScriptHTML := getModuleScriptHTML()
	capabilityScriptHTML := buildCapabilityScriptHTML(opts.Metadata)
	fullPreloadHTML := modulePreloadHTML + preloadHTML
	fullScriptHTML := moduleConfigHTML + scriptHTML + moduleScriptHTML + capabilityScriptHTML

	csrfPair := ro.ensureCSRFForMeta(rctx)

	data := &render_templates.BasePageData{
		Style:            "",
		FontsHTML:        fonts,
		Title:            opts.Metadata.Title,
		PageID:           opts.PageID,
		Keywords:         opts.Metadata.Keywords,
		CanonicalURL:     opts.Metadata.CanonicalURL,
		SvgSpriteSheet:   "",
		Aesthetic:        "",
		Lang:             opts.Metadata.Language,
		FaviconsHTML:     favicons,
		Description:      opts.Metadata.Description,
		PreloadURLS:      fullPreloadHTML,
		ModuleScripts:    fullScriptHTML,
		RenderedContent:  "",
		Styling:          opts.Styling,
		PKScriptMetas:    opts.Metadata.JSScriptMetas,
		MetaTags:         opts.Metadata.MetaTags,
		OGTags:           opts.Metadata.OGTags,
		AlternateLinks:   opts.Metadata.AlternateLinks,
		TwitterCards:     opts.Metadata.TwitterCards,
		StructuredData:   filterValidJSON(ctx, opts.Metadata.StructuredData),
		DevWidgetHTML:    getDevWidgetHTML(),
		CoreJSSRIHash:    getCoreJSSRIHash(),
		ActionsJSSRIHash: getActionsJSSRIHash(),
		ThemeCSSSRIHash:  getThemeCSSSRIHash(),
	}

	if csrfPair != nil {
		data.CSRFActionToken = csrfPair.ActionToken
		data.CSRFEphemeralToken = csrfPair.RawEphemeralToken
	}

	render_templates.StreamBasePageHeader(qw, data)
	if err := ro.renderASTToWriter(opts.Template, qw, rctx); err != nil {
		return fmt.Errorf("streaming AST for page %s: %w", opts.PageID, err)
	}
	data.SvgSpriteSheet = ro.buildSvgSpriteSheetIfNeeded(ctx, rctx)
	render_templates.StreamBasePageFooter(qw, data)
	return nil
}

// getRenderContext obtains a render context from the pool and initialises it.
//
// Takes pageID (string) which identifies the page being rendered.
// Takes response (http.ResponseWriter) which receives the rendered output.
// Takes request (*http.Request) which provides the incoming request data.
// Takes siteConfig (*config.WebsiteConfig) which provides site settings.
//
// Returns *renderContext which is ready for template rendering.
func (ro *RenderOrchestrator) getRenderContext(
	ctx context.Context,
	pageID string,
	response http.ResponseWriter,
	request *http.Request,
	siteConfig *config.WebsiteConfig,
) *renderContext {
	ctx, l := logger_domain.From(ctx, log)

	rctx, ok := renderContextPool.Get().(*renderContext)
	if !ok {
		l.Error("renderContextPool returned unexpected type, allocating new instance")
		rctx = newRenderContext()
	}

	rctx.originalCtx = ctx
	rctx.pageID = pageID
	rctx.registry = ro.registry
	rctx.csrfService = ro.csrfService
	rctx.httpRequest = request
	rctx.httpResponse = response
	rctx.stripHTMLComments = ro.stripHTMLComments

	rctx.populateLocaleFromRequest(request)
	rctx.populateI18nFromConfig(siteConfig)
	rctx.clearCaches()
	rctx.resetDiagnosticsAndCSRF()

	return rctx
}

// buildSvgSpriteSheetIfNeeded builds an SVG sprite sheet if there are any
// required symbols.
//
// Takes rctx (*renderContext) which provides the required SVG symbols.
//
// Returns string which contains the sprite sheet, or empty if none is needed.
func (ro *RenderOrchestrator) buildSvgSpriteSheetIfNeeded(
	ctx context.Context,
	rctx *renderContext,
) string {
	ctx, l := logger_domain.From(ctx, log)

	if len(rctx.requiredSvgSymbols) == 0 {
		return ""
	}

	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	if err != nil {
		l.Error("Failed to build SVG sprite sheet", logger_domain.Error(err))
		BuildSvgSpriteSheetErrorCount.Add(ctx, 1)
		return ""
	}
	return spriteSheet
}

// renderASTToWriter renders the template AST nodes to the given writer.
//
// Takes tmplAST (*ast_domain.TemplateAST) which is the parsed template to
// render.
// Takes qw (*qt.Writer) which is the output destination.
// Takes rctx (*renderContext) which provides the rendering state.
//
// Returns error when any node fails to render.
func (ro *RenderOrchestrator) renderASTToWriter(tmplAST *ast_domain.TemplateAST, qw *qt.Writer, rctx *renderContext) error {
	if tmplAST == nil {
		return nil
	}
	for _, node := range tmplAST.RootNodes {
		if err := ro.renderNode(rctx.originalCtx, node, qw, rctx); err != nil {
			return fmt.Errorf("rendering AST node: %w", err)
		}
	}
	return nil
}

// renderNode renders a single template node to the writer.
//
// Takes node (*ast_domain.TemplateNode) which is the AST node to render.
// Takes qw (*qt.Writer) which receives the rendered output.
// Takes rctx (*renderContext) which provides the rendering state.
//
// Returns error when the context is cancelled or element rendering fails.
func (ro *RenderOrchestrator) renderNode(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	qw *qt.Writer,
	rctx *renderContext,
) error {
	select {
	case <-rctx.originalCtx.Done():
		return fmt.Errorf("rendering cancelled for page %s: %w", rctx.pageID, rctx.originalCtx.Err())
	default:
	}

	if len(node.PrerenderedHTML) > 0 && !rctx.skipPrerenderedHTML {
		qw.N().Z(node.PrerenderedHTML)
		return nil
	}

	switch node.NodeType {
	case ast_domain.NodeText:
		writeTextNode(node, qw)

	case ast_domain.NodeElement:
		return ro.renderElement(ctx, node, qw, rctx, nil)

	case ast_domain.NodeComment:
		if !rctx.stripHTMLComments {
			writeCommentNode(node.TextContent, qw)
		}

	case ast_domain.NodeRawHTML:
		qw.N().Z([]byte(node.TextContent))

	case ast_domain.NodeFragment:
		return ro.renderFragmentChildren(ctx, node, qw, rctx)
	}

	return nil
}

// renderFragmentChildren renders all children of a NodeFragment, passing
// attributes to child elements.
//
// Takes node (*ast_domain.TemplateNode) which provides the fragment whose
// children will be rendered.
// Takes qw (*qt.Writer) which receives the rendered output.
// Takes rctx (*renderContext) which holds the current rendering state.
//
// Returns error when any child fails to render.
func (ro *RenderOrchestrator) renderFragmentChildren(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	qw *qt.Writer,
	rctx *renderContext,
) error {
	for _, child := range node.Children {
		if err := ro.renderFragmentChild(ctx, child, node.Attributes, qw, rctx); err != nil {
			return fmt.Errorf("rendering fragment child: %w", err)
		}
	}
	return nil
}

// renderFragmentChild renders a single child of a NodeFragment.
//
// Takes child (*ast_domain.TemplateNode) which is the child node to render.
// Takes parentAttrs ([]ast_domain.HTMLAttribute) which contains attributes
// inherited from the parent fragment.
// Takes qw (*qt.Writer) which is the output writer for rendered content.
// Takes rctx (*renderContext) which provides the current rendering state.
//
// Returns error when rendering an element or nested fragment fails.
func (ro *RenderOrchestrator) renderFragmentChild(
	ctx context.Context,
	child *ast_domain.TemplateNode,
	parentAttrs []ast_domain.HTMLAttribute,
	qw *qt.Writer,
	rctx *renderContext,
) error {
	switch child.NodeType {
	case ast_domain.NodeElement:
		return ro.renderElement(ctx, child, qw, rctx, parentAttrs)

	case ast_domain.NodeText:
		writeTextNode(child, qw)

	case ast_domain.NodeComment:
		if !rctx.stripHTMLComments {
			writeCommentNode(child.TextContent, qw)
		}

	case ast_domain.NodeRawHTML:
		qw.N().Z([]byte(child.TextContent))

	case ast_domain.NodeFragment:
		return ro.renderNode(ctx, child, qw, rctx)
	}
	return nil
}

// renderElement writes a single HTML element node to the output writer.
//
// Takes node (*ast_domain.TemplateNode) which is the element to render.
// Takes qw (*qt.Writer) which receives the rendered output.
// Takes rctx (*renderContext) which holds the rendering state and settings.
// Takes fragmentAttrs ([]ast_domain.HTMLAttribute) which are extra attributes
// to add to the element.
//
// Returns error when rendering the element content fails.
func (ro *RenderOrchestrator) renderElement(
	_ context.Context,
	node *ast_domain.TemplateNode,
	qw *qt.Writer,
	rctx *renderContext,
	fragmentAttrs []ast_domain.HTMLAttribute,
) error {
	switch node.TagName {
	case tagPikoSvg:
		return renderPikoSvg(ro, node, qw, rctx)
	case tagPikoA:
		return renderPikoA(ro, node, qw, rctx)
	case tagPikoImg:
		return renderPikoImg(ro, node, qw, rctx)
	case tagPikoVideo:
		return renderPikoVideo(ro, node, qw, rctx)
	case tagPikoPicture:
		return renderPikoPicture(ro, node, qw, rctx)
	}

	if _, ok := rctx.customTags[node.TagName]; ok {
		rctx.collectedCustomComponents[node.TagName] = struct{}{}
	}

	qw.N().Z(openBracket)
	qw.N().S(node.TagName)

	writeNodeAndFragmentAttributes(node.Attributes, fragmentAttrs, node.AttributeWriters, qw, rctx)

	ro.writeElementDirectives(node, qw, rctx)

	isVoid := isVoidElement(node.TagName)
	shouldSelfClose := isVoid || (selfClosingSVGElements[node.TagName] && node.InnerHTML == "" && len(node.Children) == 0)

	if shouldSelfClose {
		qw.N().Z(selfClose)
	} else {
		qw.N().Z(closeBracket)
	}

	if shouldSelfClose {
		return nil
	}

	if err := ro.renderNodeContent(node, qw, rctx); err != nil {
		return fmt.Errorf("rendering content for element <%s>: %w", node.TagName, err)
	}

	qw.N().Z(closeTagPrefix)
	qw.N().S(node.TagName)
	qw.N().Z(closeBracket)

	return nil
}

// renderNodeContent renders the content inside an element (innerHTML,
// textContent, or children). This helper reduces nesting depth in
// renderElement.
//
// Takes node (*ast_domain.TemplateNode) which contains the content to render.
// Takes qw (*qt.Writer) which writes the rendered output.
// Takes rctx (*renderContext) which provides the rendering state.
//
// Returns error when child nodes cannot be rendered.
func (ro *RenderOrchestrator) renderNodeContent(node *ast_domain.TemplateNode, qw *qt.Writer, rctx *renderContext) error {
	if node.InnerHTML != "" {
		qw.N().S(node.InnerHTML)
		return nil
	}

	if node.TextContentWriter != nil && node.TextContentWriter.Len() > 0 {
		writeDirectWriterParts(node.TextContentWriter, qw)
		return nil
	}

	if node.TextContent != "" {
		qw.N().S(node.TextContent)
		return nil
	}

	if len(node.Children) == 0 {
		return nil
	}

	return ro.renderASTToWriter(&ast_domain.TemplateAST{
		SourcePath:        nil,
		ExpiresAtUnixNano: nil,
		Metadata:          nil,
		RootNodes:         node.Children,
		Diagnostics:       nil,
		SourceSize:        0,
		Tidied:            false,
	}, qw, rctx)
}

// writeElementDirectives writes all common directives for an element: CSRF,
// events, and p-ref. This is shared between renderElement and specialised
// renderers (renderPikoA, renderPikoImg, renderPikoSvg) to avoid code
// duplication and ensure consistent directive handling.
//
// Takes node (*ast_domain.TemplateNode) which provides the template node
// containing directives to write.
// Takes qw (*qt.Writer) which receives the rendered output.
// Takes rctx (*renderContext) which provides the current rendering context.
func (ro *RenderOrchestrator) writeElementDirectives(
	node *ast_domain.TemplateNode,
	qw *qt.Writer,
	rctx *renderContext,
) {
	csrfPair := ro.getCSRFIfNeeded(node, rctx)
	if csrfPair != nil {
		qw.N().Z(space)
		qw.N().Z(csrfEphemeralAttrName)
		qw.N().Z(equalsQuote)
		qw.N().S(csrfPair.RawEphemeralToken)
		qw.N().Z(quote)

		qw.N().Z(space)
		qw.N().Z(csrfActionAttrName)
		qw.N().Z(equalsQuote)
		qw.N().Z(csrfPair.ActionToken)
		qw.N().Z(quote)
	}

	writeEventDirectives(node.OnEvents, pOnPrefix, qw)
	writeEventDirectives(node.CustomEvents, pEventPrefix, qw)

	if node.DirRef != nil {
		qw.N().Z(pRefPrefix)
		qw.N().S(node.DirRef.Expression.String())
		qw.N().Z(quote)
	}

	writeAttributeWriters(node.AttributeWriters, qw)
}

// writeElementDirectivesExcluding is like writeElementDirectives but excludes
// the specified attributes from AttributeWriters. Used by piko:img which
// handles src and srcset specially.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to render.
// Takes qw (*qt.Writer) which is the output writer for the rendered content.
// Takes rctx (*renderContext) which provides the current render state.
// Takes excludeAttrs (...string) which lists attribute names to skip.
func (ro *RenderOrchestrator) writeElementDirectivesExcluding(
	node *ast_domain.TemplateNode,
	qw *qt.Writer,
	rctx *renderContext,
	excludeAttrs ...string,
) {
	csrfPair := ro.getCSRFIfNeeded(node, rctx)
	if csrfPair != nil {
		qw.N().Z(space)
		qw.N().Z(csrfEphemeralAttrName)
		qw.N().Z(equalsQuote)
		qw.N().S(csrfPair.RawEphemeralToken)
		qw.N().Z(quote)

		qw.N().Z(space)
		qw.N().Z(csrfActionAttrName)
		qw.N().Z(equalsQuote)
		qw.N().Z(csrfPair.ActionToken)
		qw.N().Z(quote)
	}

	writeEventDirectives(node.OnEvents, pOnPrefix, qw)
	writeEventDirectives(node.CustomEvents, pEventPrefix, qw)

	if node.DirRef != nil {
		qw.N().Z(pRefPrefix)
		qw.N().S(node.DirRef.Expression.String())
		qw.N().Z(quote)
	}

	writeAttributeWritersExcluding(node.AttributeWriters, qw, excludeAttrs...)
}

// getCSRFIfNeeded returns the CSRF pair if the node requires CSRF protection.
// Returns nil if CSRF is not needed or if generation fails.
//
// Takes node (*ast_domain.TemplateNode) which is checked for CSRF requirements.
// Takes rctx (*renderContext) which holds the request state and CSRF service.
//
// Returns *security_dto.CSRFPair which contains the token and cookie, or nil
// if CSRF is not required or generation failed.
//
// This function does not mutate the AST. Callers should write CSRF attributes
// directly to the output stream. CSRF generation is protected by sync.Once,
// ensuring it runs at most once per request.
//
// This function avoids creating OTEL spans. CSRF generation is a fast,
// once-per-request operation where span overhead (~40 allocations) would dwarf
// the actual work. Errors are logged directly rather than through span
// instrumentation.
func (*RenderOrchestrator) getCSRFIfNeeded(
	node *ast_domain.TemplateNode,
	rctx *renderContext,
) *security_dto.CSRFPair {
	if node.RuntimeAnnotations == nil || !node.RuntimeAnnotations.NeedsCSRF {
		return nil
	}

	generateCSRFOnce(rctx, "Failed to generate CSRF pair")

	if rctx.csrfError != nil || rctx.csrfPair == nil {
		return nil
	}

	return rctx.csrfPair
}

// RenderASTToStringOptions contains options for headless AST rendering.
type RenderASTToStringOptions struct {
	// Template is the parsed AST tree to render. If nil, no output is produced.
	Template *ast_domain.TemplateAST

	// Metadata holds page details such as title, description, and language.
	Metadata *templater_dto.InternalMetadata

	// Styling specifies the CSS styles to include in the output.
	Styling string

	// IncludeDocumentWrapper determines whether to wrap output in full HTML
	// document structure (<!DOCTYPE>, <html>, <head>, <body>). When false,
	// only the AST content is rendered.
	IncludeDocumentWrapper bool
}

// ensureCSRFForMeta generates CSRF tokens for page-level meta tag injection.
// Unlike getCSRFIfNeeded (which is lazy and per-node), this is called
// proactively at the start of page/fragment rendering to populate the CSRF
// meta tags.
//
// This enables the Global CSRF Token Strategy where tokens are available
// page-wide via <meta name="csrf-token"> and <meta name="csrf-ephemeral">,
// allowing client-side JavaScript to read tokens without per-element
// attributes.
//
// Takes rctx (*renderContext) which provides the request context including
// CSRF service and HTTP request/response.
//
// Returns *security_dto.CSRFPair which contains the generated CSRF tokens,
// or nil if generation fails.
func (*RenderOrchestrator) ensureCSRFForMeta(rctx *renderContext) *security_dto.CSRFPair {
	generateCSRFOnce(rctx, "Failed to generate CSRF pair for meta tags")

	if rctx.csrfError != nil || rctx.csrfPair == nil {
		return nil
	}

	return rctx.csrfPair
}

// getHeadlessRenderContext creates a render context for headless rendering
// without HTTP dependencies.
//
// Returns *renderContext which is configured for headless use.
func (ro *RenderOrchestrator) getHeadlessRenderContext(ctx context.Context) *renderContext {
	rctx, ok := renderContextPool.Get().(*renderContext)
	if !ok {
		rctx = newRenderContext()
	}

	rctx.originalCtx = ctx
	rctx.pageID = "headless"
	rctx.registry = ro.registry
	rctx.csrfService = nil
	rctx.httpRequest = nil
	rctx.httpResponse = nil

	rctx.clearCaches()
	rctx.resetDiagnosticsAndCSRF()

	return rctx
}

// renderHeadlessFullPage renders a complete HTML page for headless contexts.
//
// Takes qw (*qt.Writer) which receives the output.
// Takes rctx (*renderContext) which provides rendering state.
// Takes opts (RenderASTToStringOptions) which configures the page.
//
// Returns error when rendering fails.
func (ro *RenderOrchestrator) renderHeadlessFullPage(
	_ context.Context,
	qw *qt.Writer,
	rctx *renderContext,
	opts RenderASTToStringOptions,
) error {
	lang := ""
	title := ""
	description := ""

	if opts.Metadata != nil {
		lang = opts.Metadata.Language
		title = opts.Metadata.Title
		description = opts.Metadata.Description
	}

	qw.N().S("<!DOCTYPE html>\n<html")
	if lang != "" {
		qw.N().S(` lang="`)
		qw.E().S(lang)
		qw.N().S(`"`)
	}
	qw.N().S(">\n<head>\n")
	qw.N().S(`<meta charset="UTF-8">`)
	qw.N().S("\n")
	qw.N().S(`<meta name="viewport" content="width=device-width, initial-scale=1.0">`)
	qw.N().S("\n")

	if title != "" {
		qw.N().S("<title>")
		qw.E().S(title)
		qw.N().S("</title>\n")
	}

	if description != "" {
		qw.N().S(`<meta name="description" content="`)
		qw.E().S(description)
		qw.N().S(`">`)
		qw.N().S("\n")
	}

	if opts.Styling != "" {
		qw.N().S("<style>\n")
		qw.N().S(opts.Styling)
		qw.N().S("\n</style>\n")
	}

	qw.N().S("</head>\n<body>\n")

	if err := ro.renderASTToWriter(opts.Template, qw, rctx); err != nil {
		return fmt.Errorf("rendering AST content: %w", err)
	}

	qw.N().S("\n</body>\n</html>")

	return nil
}

// WithStripHTMLComments configures whether HTML comments (<!-- ... -->)
// are stripped from both prerendered and runtime-rendered output.
//
// Takes strip (bool) which specifies whether comments should be stripped.
//
// Returns RenderOrchestratorOption which applies the comment stripping setting.
func WithStripHTMLComments(strip bool) RenderOrchestratorOption {
	return func(ro *RenderOrchestrator) {
		ro.stripHTMLComments = strip
	}
}

// WithCSSResetCSS sets the CSS reset content to include in the generated theme
// CSS, appended after theme variables in BuildThemeCSS output. When empty, no
// CSS reset is included.
//
// Takes css (string) which is the CSS reset content to include.
//
// Returns RenderOrchestratorOption which configures the CSS reset.
func WithCSSResetCSS(css string) RenderOrchestratorOption {
	return func(ro *RenderOrchestrator) {
		ro.cssResetCSS = css
	}
}

// isVoidElement returns true for HTML void elements (elements that cannot
// have children). Uses a switch statement which Go optimises with length-based
// dispatch, avoiding map allocation.
//
// Takes tag (string) which is the HTML tag name to check.
//
// Returns bool which is true if the tag is a void element.
func isVoidElement(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

// newRenderContext creates a new render context with initialised maps and
// default values.
//
// This is called when pool retrieval fails, which should be rare as pools
// normally return the correct type.
//
// Returns *renderContext which is ready for use.
func newRenderContext() *renderContext {
	return new(newRenderContextFields())
}

// newRenderContextFields returns a renderContext value with all maps and slices
// pre-allocated to their default capacities. Shared by the pool constructor and
// the standalone fallback to avoid duplicating the field initialisations.
//
// Returns renderContext which is ready for use.
func newRenderContextFields() renderContext {
	return renderContext{
		collectedCustomComponents: make(map[string]struct{}, renderContextMapCapacity),
		requiredSvgSymbols:        make([]svgSymbolEntry, 0, renderContextMapCapacity),
		customTags:                make(map[string]struct{}, renderContextMapCapacity),
		mergedAttrsCache:          make(map[svgCacheKey]string, renderContextMapCapacity),
		registeredDynamicAssets:   make(map[string]*registry_dto.ArtefactMeta, renderContextCacheCapacity),
		srcsetCache:               make(map[srcsetCacheKey]string, renderContextCacheCapacity),
		linkHeaderSet:             make(map[linkHeaderKey]struct{}, renderContextMapCapacity),
		collectedLinkHeaders:      make([]render_dto.LinkHeader, 0, renderContextMapCapacity),
		frozenBuffers:             make([]*[]byte, 0, renderContextCacheCapacity),
	}
}

// getBufferedWriter gets a buffered writer from the pool and sets it to write
// to the given writer.
//
// Takes w (io.Writer) which is the target for buffered output.
//
// Returns *bufio.Writer which is ready to use with the given writer.
func getBufferedWriter(w io.Writer) *bufio.Writer {
	bw, ok := bufioWriterPool.Get().(*bufio.Writer)
	if !ok {
		bw = bufio.NewWriter(w)
	} else {
		bw.Reset(w)
	}
	return bw
}

// releaseBufferedWriter returns a buffered writer to the pool after clearing
// its link to the underlying writer. The caller must call Flush before
// releasing.
//
// Takes bw (*bufio.Writer) which is the writer to return to the pool.
func releaseBufferedWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	bufioWriterPool.Put(bw)
}

// populateTagMap fills the destination map with tag names.
// The destination map must be empty before calling this function.
//
// Takes dest (map[string]struct{}) which is the map to fill with tags.
// Takes tags ([]string) which contains the tag names to add.
func populateTagMap(dest map[string]struct{}, tags []string) {
	for _, tag := range tags {
		dest[tag] = struct{}{}
	}
}

// generateCSRFOnce generates a CSRF token pair exactly once per render context.
// It uses sync.Once to ensure the generation happens only on the first call,
// subsequent calls use the cached result.
//
// Takes rctx (*renderContext) which provides the HTTP context for CSRF
// generation.
// Takes errMessage (string) which is the log message prefix on failure.
func generateCSRFOnce(rctx *renderContext, errMessage string) {
	rctx.csrfOnce.Do(func() {
		if rctx.csrfService == nil || rctx.httpRequest == nil || rctx.httpResponse == nil {
			rctx.csrfError = errors.New("CSRF service, HTTP request, or HTTP response is nil")
			return
		}

		if rctx.csrfBuf == nil {
			if buffer, ok := csrfBufPool.Get().(*bytes.Buffer); ok {
				rctx.csrfBuf = buffer
			} else {
				rctx.csrfBuf = &bytes.Buffer{}
			}
		}
		rctx.csrfBuf.Reset()

		pair, err := rctx.csrfService.GenerateCSRFPair(rctx.httpResponse, rctx.httpRequest, rctx.csrfBuf)
		if err != nil {
			_, l := logger_domain.From(rctx.originalCtx, log)
			l.Error(errMessage, logger_domain.Error(err))
			rctx.csrfError = err
			return
		}

		rctx.csrfPair = &pair
	})
}

// filterValidJSON returns only the entries from input that are syntactically
// valid JSON. Invalid entries are logged as warnings and dropped to prevent
// malformed <script> blocks in the output.
//
// Takes input ([]string) which holds raw JSON-LD strings.
//
// Returns []string containing only the valid JSON entries.
func filterValidJSON(ctx context.Context, input []string) []string {
	if len(input) == 0 {
		return nil
	}

	_, l := logger_domain.From(ctx, log)

	result := make([]string, 0, len(input))
	for _, s := range input {
		if json.ValidString(s) {
			result = append(result, s)
		} else {
			l.Warn("Dropping invalid JSON-LD structured data block")
		}
	}
	return result
}
