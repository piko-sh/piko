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

package templater_adapters

import (
	"cmp"
	"context"
	"fmt"
	"html"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/generator/generator_helpers"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// routeType groups route patterns by how specific they are.
// Static routes are most specific, then dynamic segments, then catch-alls.
type routeType int

const (
	// routeTypeStatic represents routes with no dynamic segments (e.g., "/docs",
	// "/about").
	routeTypeStatic routeType = iota

	// routeTypeDynamic represents routes with dynamic segments (e.g.,
	// "/docs/{id}").
	routeTypeDynamic

	// routeTypeCatchAll represents catch-all routes (e.g., "/docs/{path}*").
	routeTypeCatchAll
)

// pathSeparator is the separator used between path segments.
const pathSeparator = "/"

// ManifestStore is the runtime representation of the compiled manifest.
// It implements ManifestStoreView and loads static data from a provider
// (JSON or FlatBuffers), dynamically linking it with the live function
// pointers registered by the compiled components' init() functions.
type ManifestStore struct {
	// registry provides template functions; uses the global registry if nil.
	registry templater_domain.FunctionRegistry

	// pages maps source paths to page entries for template lookup.
	pages map[string]*PageEntry

	// partials maps partial names to their page entries.
	partials map[string]*PageEntry

	// emails maps email template paths to their page entries.
	emails map[string]*PageEntry

	// pdfs maps PDF template paths to their page entries.
	pdfs map[string]*PageEntry

	// errorPages maps HTTP status codes to sorted lists of error page entries.
	// Each list is sorted by scope specificity (longest ScopePath first) so
	// the most specific error page for a request path is found first.
	errorPages map[int][]*errorPageEntry

	// rangeErrorPages holds range-based error pages (e.g., !400-499.pk).
	// Sorted by scope specificity (longest ScopePath first).
	rangeErrorPages []*rangeErrorPageEntry

	// catchAllErrorPages holds catch-all error pages (!error.pk).
	// Sorted by scope specificity (longest ScopePath first).
	catchAllErrorPages []*errorPageEntry

	// jsArtefactToPartialName maps JS artefact IDs to their partial names,
	// enabling the frontend to scope partial JS functions by their friendly
	// partial name (e.g., "modal-wrapper") rather than the hashed ID. Only
	// populated for partials that have JS scripts.
	jsArtefactToPartialName map[string]string

	// collectionFallbackRoutes preserves the original dynamic route patterns
	// from static collections. The daemon registers these as low-priority
	// fallback routes that return "collection item not found" error pages.
	collectionFallbackRoutes []generator_dto.CollectionFallbackRoute

	// baseDir is the project root folder used to build full paths from relative
	// paths in runtime error messages.
	baseDir string

	// keys holds all template keys sorted by route specificity.
	keys []string
}

// errorPageEntry pairs an error page's scope path with its linked PageEntry
// for runtime rendering.
type errorPageEntry struct {
	// pageEntry is the linked page entry used for rendering.
	pageEntry *PageEntry

	// scopePath is the URL path prefix this error page covers (e.g., "/", "/app/").
	scopePath string
}

// rangeErrorPageEntry extends errorPageEntry with status code range bounds.
type rangeErrorPageEntry struct {
	errorPageEntry

	// statusCodeMin is the lower bound of the range (inclusive).
	statusCodeMin int

	// statusCodeMax is the upper bound of the range (inclusive).
	statusCodeMax int
}

// PageEntry is the concrete, in-memory implementation of the PageEntryView
// interface. It holds both the static data loaded from the manifest file and
// the dynamic function pointers retrieved from the runtime registry.
type PageEntry struct {
	// ModTime is when the page was last modified.
	ModTime time.Time

	// registry is the function registry used to look up compiled functions.
	// It is injected from the ManifestStore; nil uses the default registry.
	registry templater_domain.FunctionRegistry

	// astFunc holds the compiled AST builder for this component, set by LinkFuncs
	// or SetASTFunc.
	astFunc templater_domain.ASTFunc

	// cachePolicyFunc returns the cache policy for a given request.
	cachePolicyFunc templater_domain.CachePolicyFunc

	// middlewareFunc returns the middleware chain for this page.
	middlewareFunc templater_domain.MiddlewareFunc

	// supportedLocalesFunc returns the list of supported locales for this page.
	supportedLocalesFunc templater_domain.SupportedLocalesFunc

	// authPolicyFunc returns the auth requirements for this page.
	authPolicyFunc templater_domain.AuthPolicyFunc

	// previewFunc returns preview scenarios for dev-mode previewing.
	previewFunc templater_domain.PreviewFunc

	// jsArtefactToPartialName maps JS artefact IDs to partial names. Used by
	// GetJSScriptMetas for frontend function scoping; an empty string means a
	// page-level script.
	jsArtefactToPartialName map[string]string

	// localStore holds the pre-built translation Store for this page's local
	// translations, built once at manifest load time from LocalTranslations.
	localStore *i18n_domain.Store

	// baseDir is the project root folder used to make relative paths into full
	// paths when showing runtime errors.
	baseDir string

	// cachedJSScriptMetas holds pre-computed JS script metadata to avoid
	// per-request allocations. Computed once during manifest loading.
	cachedJSScriptMetas []templater_dto.JSScriptMeta

	// ManifestPageEntry contains static metadata loaded from the manifest file.
	generator_dto.ManifestPageEntry

	// cachedStaticMetadata holds pre-computed static metadata (AssetRefs,
	// CustomTags, SupportedLocales) to avoid per-request allocations.
	cachedStaticMetadata templater_dto.InternalMetadata
}

var _ templater_domain.ManifestStoreView = (*ManifestStore)(nil)

var _ templater_domain.PageEntryView = (*PageEntry)(nil)

// ManifestStoreOption is a function type that sets options for a ManifestStore.
type ManifestStoreOption func(*ManifestStore)

// NewManifestStore creates a runtime manifest store from the given provider.
//
// It loads the manifest data, then links that data to the registered
// functions. Static routes are sorted before dynamic routes to ensure
// exact matches take precedence over pattern matches.
//
// Optional configuration can be provided via functional options:
//   - withRegistry: Inject a custom registry (useful for testing)
//
// Takes provider (ManifestProviderPort) which loads the manifest data.
// Takes opts (ManifestStoreOption) which configures the store behaviour.
//
// Returns *ManifestStore which contains the linked manifest entries.
// Returns error when the manifest cannot be loaded from the provider.
func NewManifestStore(ctx context.Context, provider generator_domain.ManifestProviderPort, opts ...ManifestStoreOption) (*ManifestStore, error) {
	manifest, err := provider.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	store := &ManifestStore{
		pages:                   make(map[string]*PageEntry, len(manifest.Pages)),
		partials:                make(map[string]*PageEntry, len(manifest.Partials)),
		emails:                  make(map[string]*PageEntry, len(manifest.Emails)),
		pdfs:                    make(map[string]*PageEntry, len(manifest.Pdfs)),
		errorPages:              make(map[int][]*errorPageEntry),
		keys:                    make([]string, 0, len(manifest.Pages)+len(manifest.Partials)+len(manifest.Emails)+len(manifest.Pdfs)),
		registry:                templater_domain.GetDefaultRegistry(),
		jsArtefactToPartialName: make(map[string]string, len(manifest.Partials)),
	}

	for _, opt := range opts {
		opt(store)
	}

	processPartials(store, manifest.Partials)
	processPages(store, manifest.Pages)
	processEmails(store, manifest.Emails)
	processPdfs(store, manifest.Pdfs)
	processErrorPages(store, manifest.ErrorPages)
	store.collectionFallbackRoutes = manifest.CollectionFallbackRoutes

	sortKeysByRouteSpecificity(store)
	warnUnlinkedPages(store)
	return store, nil
}

// warnUnlinkedPages checks whether any pages failed to link their AST
// functions from the runtime registry and logs a diagnostic warning.
// If the registry is entirely empty it hints at a missing dist import.
//
// Takes store (*ManifestStore) which is the manifest store to inspect for
// unlinked pages.
func warnUnlinkedPages(store *ManifestStore) {
	var unlinked []string
	for key, entry := range store.pages {
		if entry.astFunc == nil {
			unlinked = append(unlinked, key)
		}
	}
	if len(unlinked) == 0 {
		return
	}

	registry := store.registry
	if registry == nil {
		registry = templater_domain.GetDefaultRegistry()
	}

	if len(registry.List()) == 0 {
		log.Warn(
			"No component functions are registered in the runtime registry. "+
				"This usually means the generated dist package is not imported. "+
				"Add a blank import to your main.go:  _ \"<module>/dist\"",
			logger_domain.Int("unlinked_page_count", len(unlinked)),
		)
		_, _ = fmt.Fprintf(os.Stderr,
			"\n⚠ Piko: %d page(s) have no registered AST function because the runtime registry is empty.\n"+
				"  Add a blank import to your main.go:\n\n"+
				"      import _ \"<module>/dist\"\n\n"+
				"  This ensures the generated init() functions run and register your components.\n\n",
			len(unlinked),
		)
	} else {
		log.Warn(
			"Some pages could not be linked to their compiled AST functions. "+
				"The generated code in dist/ may be out of date; try re-running the generator.",
			logger_domain.Int("unlinked_page_count", len(unlinked)),
			logger_domain.String("example", unlinked[0]),
		)
	}
}

// FindErrorPage looks up the most specific error page for the given
// HTTP status code and request path, using a three-tier fallback chain
// (exact match, range match, catch-all) where the longest matching
// ScopePath takes priority within each tier.
//
// Takes statusCode (int) which is the HTTP status code to find a page for.
// Takes requestPath (string) which is the URL path being requested.
//
// Returns PageEntryView which is the matching error page entry.
// Returns bool which is true if a matching error page was found.
func (s *ManifestStore) FindErrorPage(statusCode int, requestPath string) (templater_domain.PageEntryView, bool) {
	if entry, ok := s.findExactErrorPage(statusCode, requestPath); ok {
		return entry, true
	}
	if entry, ok := s.findRangeErrorPage(statusCode, requestPath); ok {
		return entry, true
	}
	return s.findCatchAllErrorPage(requestPath)
}

// LinkFuncs connects a static manifest entry to the compiled functions that
// were registered during the application's init phase. This method is exported
// so that InterpretedBuildOrchestrator can call it.
func (pe *PageEntry) LinkFuncs() {
	registry := pe.registry
	if registry == nil {
		registry = templater_domain.GetDefaultRegistry()
	}

	pe.astFunc, _ = registry.GetASTFunc(pe.PackagePath)
	pe.cachePolicyFunc = registry.GetCachePolicyFunc(pe.PackagePath)
	pe.middlewareFunc = registry.GetMiddlewareFunc(pe.PackagePath)
	pe.supportedLocalesFunc = registry.GetSupportedLocalesFunc(pe.PackagePath)
	pe.authPolicyFunc = registry.GetAuthPolicyFunc(pe.PackagePath)
	pe.previewFunc, _ = registry.GetPreviewFunc(pe.PackagePath)
	pe.initialiseCachedJSScriptMetas()
	pe.initialiseCachedStaticMetadata()
}

// InitialiseCachedMetadata rebuilds the pre-computed JS script metadata and
// static metadata caches.
//
// Call this after linking functions from a registry when LinkFuncs is not used
// (e.g. the JIT interpreted build path). The supportedLocalesFunc must be set
// before calling this method because initialiseCachedStaticMetadata invokes it.
func (pe *PageEntry) InitialiseCachedMetadata() {
	pe.initialiseCachedJSScriptMetas()
	pe.initialiseCachedStaticMetadata()
}

// SetBaseDir sets the project root directory used to build full paths from
// relative paths in runtime error diagnostics.
//
// Takes dir (string) which specifies the project root directory path.
func (pe *PageEntry) SetBaseDir(dir string) {
	pe.baseDir = dir
}

// SetJSArtefactToPartialNameMap sets the shared mapping from JS artefact IDs
// to partial names. This is used by initialiseCachedJSScriptMetas to populate
// the PartialName field for frontend function scoping.
//
// Takes m (map[string]string) which maps artefact IDs to partial names.
func (pe *PageEntry) SetJSArtefactToPartialNameMap(m map[string]string) {
	pe.jsArtefactToPartialName = m
}

// InitialiseLocalStore builds the pre-computed translation store from
// LocalTranslations if any are present. This must be called during entry
// setup for pages and emails that use component-scoped i18n.
func (pe *PageEntry) InitialiseLocalStore() {
	if len(pe.LocalTranslations) > 0 {
		pe.localStore = i18n_domain.NewStoreFromTranslations(pe.LocalTranslations, "")
	}
}

// GetStaticMetadata returns a pointer to the pre-computed static metadata.
// This avoids per-request allocations for probe operations.
//
// Returns *templater_dto.InternalMetadata which contains the cached static
// metadata. The caller MUST NOT modify this data.
func (pe *PageEntry) GetStaticMetadata() *templater_dto.InternalMetadata {
	return &pe.cachedStaticMetadata
}

// SetASTFunc sets the function used to build the AST.
//
// Takes registryFunction (templater_domain.ASTFunc) which is the
// function to build the AST.
func (pe *PageEntry) SetASTFunc(registryFunction templater_domain.ASTFunc) {
	pe.astFunc = registryFunction
}

// SetCachePolicyFunc sets the cache policy function for this page entry.
//
// Takes registryFunction (CachePolicyFunc) which specifies the cache
// policy to apply.
func (pe *PageEntry) SetCachePolicyFunc(registryFunction templater_domain.CachePolicyFunc) {
	pe.cachePolicyFunc = registryFunction
}

// SetMiddlewareFunc sets the middleware function pointer directly.
//
// Takes registryFunction (MiddlewareFunc) which provides the
// middleware handling logic.
func (pe *PageEntry) SetMiddlewareFunc(registryFunction templater_domain.MiddlewareFunc) {
	pe.middlewareFunc = registryFunction
}

// SetSupportedLocalesFunc sets the function used to look up supported locales.
//
// Takes registryFunction (SupportedLocalesFunc) which provides the
// locale lookup function.
func (pe *PageEntry) SetSupportedLocalesFunc(registryFunction templater_domain.SupportedLocalesFunc) {
	pe.supportedLocalesFunc = registryFunction
}

// GetASTRoot executes the compiled BuildAST function for the component.
//
// If the function was not found in the registry, it returns a diagnostic AST
// to display the error.
//
// Takes r (*templater_dto.RequestData) which provides the request context for
// AST generation.
//
// Returns *ast_domain.TemplateAST which is the generated AST for the component.
// Returns templater_dto.InternalMetadata which contains metadata from the AST
// function.
func (pe *PageEntry) GetASTRoot(r *templater_dto.RequestData) (*ast_domain.TemplateAST, templater_dto.InternalMetadata) {
	return pe.GetASTRootWithProps(r, nil)
}

// GetASTRootWithProps executes the compiled BuildAST function with the
// provided props data.
//
// Takes r (*templater_dto.RequestData) which provides the request context.
// Takes props (any) which contains the properties to pass to the AST function.
//
// Returns *ast_domain.TemplateAST which is the generated template AST, or an
// error AST if the function is not registered.
// Returns templater_dto.InternalMetadata which contains metadata from the AST
// generation.
func (pe *PageEntry) GetASTRootWithProps(r *templater_dto.RequestData, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata) {
	if pe.astFunc == nil {
		registry := pe.registry
		if registry == nil {
			registry = templater_domain.GetDefaultRegistry()
		}

		var err error
		if len(registry.List()) == 0 {
			err = fmt.Errorf(
				"AST function for component '%s' (package: %s) not found: the runtime registry is empty. "+
					"Add a blank import to your main.go:  _ \"<module>/dist\"",
				pe.OriginalSourcePath,
				pe.PackagePath,
			)
		} else {
			err = fmt.Errorf(
				"AST function for component '%s' (package: %s) not found in runtime registry; "+
					"the generated code in dist/ may be out of date; try re-running the generator",
				pe.OriginalSourcePath,
				pe.PackagePath,
			)
		}
		return buildErrorAST(err, pe.OriginalSourcePath), templater_dto.InternalMetadata{}
	}

	ast, metadata, diagnostics := pe.astFunc(r, props)
	if len(diagnostics) > 0 {
		formattedErrors := generator_helpers.FormatRuntimeDiagnostics(diagnostics, pe.baseDir)
		_, l := logger_domain.From(r.Context(), log)
		l.Error("Error running AST function", logger_domain.Int("diagnostic_count", len(diagnostics)))
		_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", formattedErrors)
	}
	return ast, metadata
}

// GetKeys returns a sorted list of all known component paths, including both
// pages and partials.
//
// Returns []string which contains the component paths in sorted order.
func (s *ManifestStore) GetKeys() []string {
	return s.keys
}

// GetCollectionFallbackRoutes returns the original dynamic route patterns from
// static collections. The daemon registers these as low-priority fallback
// routes.
//
// Returns []templater_domain.CollectionFallbackRouteView which contains the
// fallback route patterns, or nil if there are none.
func (s *ManifestStore) GetCollectionFallbackRoutes() []templater_domain.CollectionFallbackRouteView {
	if len(s.collectionFallbackRoutes) == 0 {
		return nil
	}
	views := make([]templater_domain.CollectionFallbackRouteView, len(s.collectionFallbackRoutes))
	for i, r := range s.collectionFallbackRoutes {
		views[i] = templater_domain.CollectionFallbackRouteView{RoutePatterns: r.RoutePatterns}
	}
	return views
}

// GetPageEntry provides a unified lookup for pages, partials, and emails
// by their original source path.
//
// Takes path (string) which is the original source path to look up.
//
// Returns templater_domain.PageEntryView which is the found entry.
// Returns bool which indicates whether the entry was found.
func (s *ManifestStore) GetPageEntry(path string) (templater_domain.PageEntryView, bool) {
	if entry, ok := s.pages[path]; ok {
		return entry, true
	}
	if entry, ok := s.partials[path]; ok {
		return entry, true
	}
	if entry, ok := s.emails[path]; ok {
		return entry, true
	}
	if entry, ok := s.pdfs[path]; ok {
		return entry, true
	}
	return nil, false
}

// GetIsPage determines if the entry is a page or a partial based on its
// route patterns.
//
// Returns bool which is true if the entry is a page, false if it is a partial.
func (pe *PageEntry) GetIsPage() bool {
	for _, pattern := range pe.RoutePatterns {
		return pattern != "" && !strings.HasPrefix(pattern, "/_piko/")
	}
	return false
}

// GetCachePolicy executes the compiled CachePolicy function for the component.
//
// Takes r (*templater_dto.RequestData) which provides the request context.
//
// Returns templater_dto.CachePolicy which specifies the caching behaviour.
func (pe *PageEntry) GetCachePolicy(r *templater_dto.RequestData) templater_dto.CachePolicy {
	return pe.cachePolicyFunc(r)
}

// GetMiddlewares executes the compiled Middlewares function for the component.
//
// Returns []func(http.Handler) http.Handler which contains the middleware chain
// for this page entry.
func (pe *PageEntry) GetMiddlewares() []func(http.Handler) http.Handler {
	return pe.middlewareFunc()
}

// GetHasAuthPolicy reports whether the page declares auth requirements.
//
// Returns bool which is true when an AuthPolicy function is registered.
func (pe *PageEntry) GetHasAuthPolicy() bool {
	return pe.HasAuthPolicy
}

// GetAuthPolicy executes the compiled AuthPolicy function for the component.
//
// Takes r (*templater_dto.RequestData) which provides request details.
//
// Returns daemon_dto.AuthPolicy which specifies the auth requirements.
func (pe *PageEntry) GetAuthPolicy(r *templater_dto.RequestData) daemon_dto.AuthPolicy {
	return pe.authPolicyFunc(r)
}

// GetSupportedLocales executes the compiled SupportedLocales function.
//
// Returns []string which contains the locale codes supported by this page.
func (pe *PageEntry) GetSupportedLocales() []string {
	return pe.supportedLocalesFunc()
}

// GetStyling returns the combined and scoped CSS for the component.
//
// Returns string which contains the CSS style block for this page entry.
func (pe *PageEntry) GetStyling() string {
	return pe.StyleBlock
}

// GetAssetRefs returns the static list of asset references.
//
// Returns []templater_dto.AssetRef which contains the page's asset references.
func (pe *PageEntry) GetAssetRefs() []templater_dto.AssetRef {
	return pe.AssetRefs
}

// GetCustomTags returns the static list of potential custom tags.
//
// Returns []string which contains the allowed custom tag names.
func (pe *PageEntry) GetCustomTags() []string {
	return pe.CustomTags
}

// GetHasCachePolicy returns the static boolean flag from the manifest.
//
// Returns bool which is true if the page has a cache policy defined.
func (pe *PageEntry) GetHasCachePolicy() bool {
	return pe.HasCachePolicy
}

// GetHasMiddleware returns the static boolean flag from the manifest.
//
// Returns bool which indicates whether the page uses middleware.
func (pe *PageEntry) GetHasMiddleware() bool {
	return pe.HasMiddleware
}

// GetHasSupportedLocales returns the static boolean flag from the manifest.
//
// Returns bool which indicates whether the page has supported locales defined.
func (pe *PageEntry) GetHasSupportedLocales() bool {
	return pe.HasSupportedLocales
}

// GetHasPreview reports whether this component defines a Preview function.
func (pe *PageEntry) GetHasPreview() bool {
	return pe.HasPreview
}

// GetPreviewScenarios returns the preview scenarios for this component.
// Returns nil if the component has no Preview function.
func (pe *PageEntry) GetPreviewScenarios() []templater_dto.PreviewScenario {
	if pe.previewFunc == nil {
		return nil
	}
	return pe.previewFunc()
}

// GetLocalStore returns the pre-built translation Store for this page's local
// translations. Returns nil if the page has no local translations.
//
// Returns *i18n_domain.Store which provides pre-parsed, zero-allocation lookups
// for component-scoped translations.
func (pe *PageEntry) GetLocalStore() *i18n_domain.Store {
	return pe.localStore
}

// GetJSScriptMetas returns metadata for all client-side JavaScript modules
// needed by this page.
//
// This includes the page's own script plus scripts from all embedded partials.
// Each artefact ID is resolved to the standard asset serving URL pattern.
// Partial scripts include their friendly partial name for frontend function
// scoping.
//
// Returns []templater_dto.JSScriptMeta which contains script URLs and partial
// names, or nil if there are no scripts.
func (pe *PageEntry) GetJSScriptMetas() []templater_dto.JSScriptMeta {
	return pe.cachedJSScriptMetas
}

// GetOriginalPath returns the project-relative source path.
//
// Returns string which is the path to the original source file.
func (pe *PageEntry) GetOriginalPath() string {
	return pe.OriginalSourcePath
}

// GetRoutePattern returns the first route pattern from RoutePatterns.
//
// This is primarily used for partials (which have a single pattern) and for
// logging. For pages with i18n routing, use GetRoutePatterns instead.
//
// Returns string which is the first route pattern, or empty if none exist.
func (pe *PageEntry) GetRoutePattern() string {
	for _, pattern := range pe.RoutePatterns {
		return pattern
	}
	return ""
}

// GetRoutePatterns returns the full map of route patterns with their locale
// keys.
//
// Returns map[string]string which maps locale keys to their route patterns.
func (pe *PageEntry) GetRoutePatterns() map[string]string {
	return pe.RoutePatterns
}

// GetI18nStrategy returns the internationalisation strategy for the page.
//
// Returns string which is the strategy name from the manifest entry.
func (pe *PageEntry) GetI18nStrategy() string {
	return pe.I18nStrategy
}

// GetMiddlewareFuncName returns the original name of the middleware function,
// used for runtime logging, debugging, and observability.
//
// Returns string which is the middleware function name from the manifest.
func (pe *PageEntry) GetMiddlewareFuncName() string {
	return pe.MiddlewareFuncName
}

// GetCachePolicyFuncName returns the original name of the cache policy
// function, useful for runtime logging and debugging.
//
// Returns string which is the function name from the manifest.
func (pe *PageEntry) GetCachePolicyFuncName() string {
	return pe.CachePolicyFuncName
}

// GetIsE2EOnly reports whether this entry is from the e2e/ directory.
// E2E components are only served when Build.E2EMode is enabled at runtime.
//
// Returns bool which is true if this is an E2E-only component.
func (pe *PageEntry) GetIsE2EOnly() bool {
	return pe.IsE2EOnly
}

// initialiseCachedJSScriptMetas pre-computes JS script
// metadata once during manifest load to avoid per-request
// allocations.
func (pe *PageEntry) initialiseCachedJSScriptMetas() {
	ids := pe.JSArtefactIDs
	if len(ids) == 0 {
		return
	}
	pe.cachedJSScriptMetas = make([]templater_dto.JSScriptMeta, len(ids))
	for i, id := range ids {
		pe.cachedJSScriptMetas[i] = templater_dto.JSScriptMeta{
			URL:         "/_piko/assets/" + id,
			PartialName: pe.jsArtefactToPartialName[id],
		}
	}
}

// initialiseCachedStaticMetadata pre-computes static metadata once during manifest
// load to avoid per-request struct allocations in ProbePage/ProbePartial.
func (pe *PageEntry) initialiseCachedStaticMetadata() {
	pe.cachedStaticMetadata = templater_dto.InternalMetadata{
		AssetRefs:        pe.AssetRefs,
		CustomTags:       pe.CustomTags,
		SupportedLocales: pe.supportedLocalesFunc(),
	}
}

// WithBaseDir sets the project root directory for the manifest store.
// This is used to build full paths from relative paths in runtime
// diagnostics, making error messages easier to use for IDE navigation.
//
// Takes baseDir (string) which specifies the project root directory path.
//
// Returns ManifestStoreOption which configures the base directory setting.
func WithBaseDir(baseDir string) ManifestStoreOption {
	return func(store *ManifestStore) {
		store.baseDir = baseDir
	}
}

// withRegistry sets a custom function registry, mainly for testing.
// If not set, the store uses the global default registry.
//
// Takes registry (FunctionRegistry) which provides the custom registry to use.
//
// Returns ManifestStoreOption which sets the registry on the store.
func withRegistry(registry templater_domain.FunctionRegistry) ManifestStoreOption {
	return func(store *ManifestStore) {
		store.registry = registry
	}
}

// processPages loads and links all page entries from the manifest.
//
// Takes store (*ManifestStore) which receives the processed page entries.
// Takes pages (map[string]generator_dto.ManifestPageEntry) which contains the
// raw manifest page data to process.
func processPages(store *ManifestStore, pages map[string]generator_dto.ManifestPageEntry) {
	for key := range pages {
		pageData := pages[key]
		entry := &PageEntry{
			ManifestPageEntry:       pageData,
			astFunc:                 nil,
			cachePolicyFunc:         nil,
			middlewareFunc:          nil,
			supportedLocalesFunc:    nil,
			ModTime:                 time.Time{},
			registry:                store.registry,
			jsArtefactToPartialName: store.jsArtefactToPartialName,
			baseDir:                 store.baseDir,
		}
		if len(pageData.LocalTranslations) > 0 {
			entry.localStore = i18n_domain.NewStoreFromTranslations(pageData.LocalTranslations, "")
		}
		entry.LinkFuncs()
		store.pages[key] = entry
		store.keys = append(store.keys, key)
	}
}

// processPartials loads and links all partial entries, adapting them to the
// PageEntry structure.
//
// Partials use single-pattern routing with query-only locale detection. This
// function also builds the jsArtefactToPartialName mapping for frontend
// function scoping.
//
// Takes store (*ManifestStore) which receives the processed partial entries.
// Takes partials (map[string]generator_dto.ManifestPartialEntry) which
// provides the raw partial data to process.
func processPartials(store *ManifestStore, partials map[string]generator_dto.ManifestPartialEntry) {
	for key, partialData := range partials {
		if partialData.JSArtefactID != "" {
			store.jsArtefactToPartialName[partialData.JSArtefactID] = partialData.PartialName
		}

		entry := &PageEntry{
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				PackagePath:              partialData.PackagePath,
				OriginalSourcePath:       partialData.OriginalSourcePath,
				RoutePatterns:            map[string]string{"": partialData.PartialSrc},
				I18nStrategy:             "",
				StyleBlock:               partialData.StyleBlock,
				AssetRefs:                nil,
				CustomTags:               nil,
				JSArtefactIDs:            partialJSArtefactIDsToSlice(partialData.JSArtefactID),
				HasCachePolicy:           false,
				CachePolicyFuncName:      "",
				HasMiddleware:            false,
				MiddlewareFuncName:       "",
				HasSupportedLocales:      false,
				SupportedLocalesFuncName: "",
				LocalTranslations:        nil,
			},
			astFunc:                 nil,
			cachePolicyFunc:         nil,
			middlewareFunc:          nil,
			supportedLocalesFunc:    nil,
			ModTime:                 time.Time{},
			registry:                store.registry,
			jsArtefactToPartialName: store.jsArtefactToPartialName,
			baseDir:                 store.baseDir,
		}
		entry.LinkFuncs()
		store.partials[key] = entry
		store.keys = append(store.keys, key)
	}
}

// localisableManifestData holds the fields shared between email and PDF
// manifest entries, used to avoid duplicating PageEntry construction logic.
type localisableManifestData struct {
	// localTranslations holds the per-locale translation key-value pairs.
	localTranslations i18n_domain.Translations

	// packagePath holds the Go package path for the entry.
	packagePath string

	// originalSourcePath holds the source file path relative to the project root.
	originalSourcePath string

	// styleBlock holds the scoped CSS for this entry.
	styleBlock string

	// hasSupportedLocales indicates whether the entry defines a supported locales function.
	hasSupportedLocales bool

	// hasPreview indicates whether the entry defines a Preview function.
	hasPreview bool
}

// buildLocalisablePageEntry constructs a PageEntry from the common fields of
// email and PDF manifest entries.
//
// Takes store (*ManifestStore) which provides the registry and base directory.
// Takes data (localisableManifestData) which holds the shared entry fields.
//
// Returns *PageEntry which is fully linked and ready for rendering.
func buildLocalisablePageEntry(store *ManifestStore, data localisableManifestData) *PageEntry {
	entry := &PageEntry{
		ManifestPageEntry: generator_dto.ManifestPageEntry{
			PackagePath:         data.packagePath,
			OriginalSourcePath:  data.originalSourcePath,
			StyleBlock:          data.styleBlock,
			HasSupportedLocales: data.hasSupportedLocales,
			LocalTranslations:   data.localTranslations,
			HasPreview:          data.hasPreview,
		},
		ModTime:  time.Time{},
		registry: store.registry,
		baseDir:  store.baseDir,
	}
	entry.LinkFuncs()
	return entry
}

// processEmails loads and links all email entries, adapting them to the
// PageEntry structure.
//
// Takes store (*ManifestStore) which receives the processed email entries.
// Takes emails (map[string]generator_dto.ManifestEmailEntry) which provides
// the raw email manifest data to process.
func processEmails(store *ManifestStore, emails map[string]generator_dto.ManifestEmailEntry) {
	for key, emailData := range emails {
		entry := buildLocalisablePageEntry(store, localisableManifestData{
			packagePath:         emailData.PackagePath,
			originalSourcePath:  emailData.OriginalSourcePath,
			styleBlock:          emailData.StyleBlock,
			hasSupportedLocales: emailData.HasSupportedLocales,
			localTranslations:   emailData.LocalTranslations,
			hasPreview:          emailData.HasPreview,
		})
		store.emails[key] = entry
		store.keys = append(store.keys, key)
	}
}

// processPdfs loads and links all PDF entries, adapting them to the
// PageEntry structure.
//
// Takes store (*ManifestStore) which receives the processed PDF entries.
// Takes pdfs (map[string]generator_dto.ManifestPdfEntry) which provides
// the raw PDF manifest data to process.
func processPdfs(store *ManifestStore, pdfs map[string]generator_dto.ManifestPdfEntry) {
	for key, pdfData := range pdfs {
		entry := buildLocalisablePageEntry(store, localisableManifestData{
			packagePath:         pdfData.PackagePath,
			originalSourcePath:  pdfData.OriginalSourcePath,
			styleBlock:          pdfData.StyleBlock,
			hasSupportedLocales: pdfData.HasSupportedLocales,
			localTranslations:   pdfData.LocalTranslations,
			hasPreview:          pdfData.HasPreview,
		})
		store.pdfs[key] = entry
		store.keys = append(store.keys, key)
	}
}

// ListPreviewEntries returns all manifest entries that have a Preview function
// defined. Iterates all four component maps (pages, partials, emails, PDFs),
// filters by HasPreview with a non-nil previewFunc, and returns entries sorted
// by OriginalSourcePath.
func (s *ManifestStore) ListPreviewEntries() []templater_domain.PreviewCatalogueEntry {
	var entries []templater_domain.PreviewCatalogueEntry

	collectFrom := func(componentMap map[string]*PageEntry, componentType string) {
		for _, entry := range componentMap {
			if !entry.HasPreview || entry.previewFunc == nil {
				continue
			}
			scenarios := entry.previewFunc()
			entries = append(entries, templater_domain.PreviewCatalogueEntry{
				OriginalSourcePath: entry.OriginalSourcePath,
				ComponentType:      componentType,
				Scenarios:          scenarios,
			})
		}
	}

	collectFrom(s.pages, "page")
	collectFrom(s.partials, "partial")
	collectFrom(s.emails, "email")
	collectFrom(s.pdfs, "pdf")

	slices.SortFunc(entries, func(a, b templater_domain.PreviewCatalogueEntry) int {
		return cmp.Compare(a.OriginalSourcePath, b.OriginalSourcePath)
	})

	return entries
}

// findExactErrorPage searches for an error page matching the exact status code
// and request path scope.
//
// Takes statusCode (int) which is the HTTP status code to match.
// Takes requestPath (string) which is the URL path being requested.
//
// Returns templater_domain.PageEntryView which is the matching entry.
// Returns bool which is true when a match is found.
func (s *ManifestStore) findExactErrorPage(statusCode int, requestPath string) (templater_domain.PageEntryView, bool) {
	entries, ok := s.errorPages[statusCode]
	if !ok {
		return nil, false
	}
	for _, epe := range entries {
		if strings.HasPrefix(requestPath, epe.scopePath) {
			return epe.pageEntry, true
		}
	}
	return nil, false
}

// findRangeErrorPage searches for an error page whose status code range
// contains the given code and whose scope path matches the request path.
//
// Takes statusCode (int) which is the HTTP status code to match.
// Takes requestPath (string) which is the URL path being requested.
//
// Returns templater_domain.PageEntryView which is the matching entry.
// Returns bool which is true when a match is found.
func (s *ManifestStore) findRangeErrorPage(statusCode int, requestPath string) (templater_domain.PageEntryView, bool) {
	for _, rpe := range s.rangeErrorPages {
		if statusCode >= rpe.statusCodeMin && statusCode <= rpe.statusCodeMax {
			if strings.HasPrefix(requestPath, rpe.scopePath) {
				return rpe.pageEntry, true
			}
		}
	}
	return nil, false
}

// findCatchAllErrorPage searches for a catch-all error page whose scope path
// matches the request path.
//
// Takes requestPath (string) which is the URL path being requested.
//
// Returns templater_domain.PageEntryView which is the matching entry.
// Returns bool which is true when a match is found.
func (s *ManifestStore) findCatchAllErrorPage(requestPath string) (templater_domain.PageEntryView, bool) {
	for _, epe := range s.catchAllErrorPages {
		if strings.HasPrefix(requestPath, epe.scopePath) {
			return epe.pageEntry, true
		}
	}
	return nil, false
}

// processErrorPages loads and links error page entries from the manifest.
// Error pages are grouped by status code and sorted by scope specificity
// (longest scope path first) so the most specific error page is matched.
//
// Takes store (*ManifestStore) which receives the processed error page entries.
// Takes errorPages (map[string]generator_dto.ManifestErrorPageEntry) which
// contains the raw error page data to process.
func processErrorPages(store *ManifestStore, errorPages map[string]generator_dto.ManifestErrorPageEntry) {
	for key := range errorPages {
		epData := errorPages[key]
		entry := &PageEntry{
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				PackagePath:        epData.PackagePath,
				OriginalSourcePath: epData.OriginalSourcePath,
				StyleBlock:         epData.StyleBlock,
				JSArtefactIDs:      epData.JSArtefactIDs,
				CustomTags:         epData.CustomTags,
				IsE2EOnly:          epData.IsE2EOnly,
			},
			registry: store.registry,
			baseDir:  store.baseDir,
		}
		entry.LinkFuncs()

		baseEntry := &errorPageEntry{
			scopePath: epData.ScopePath,
			pageEntry: entry,
		}

		switch {
		case epData.IsCatchAll:
			store.catchAllErrorPages = append(store.catchAllErrorPages, baseEntry)
		case epData.StatusCodeMin > 0 && epData.StatusCodeMax > 0:
			store.rangeErrorPages = append(store.rangeErrorPages, &rangeErrorPageEntry{
				errorPageEntry: *baseEntry,
				statusCodeMin:  epData.StatusCodeMin,
				statusCodeMax:  epData.StatusCodeMax,
			})
		default:
			store.errorPages[epData.StatusCode] = append(store.errorPages[epData.StatusCode], baseEntry)
		}

		store.keys = append(store.keys, key)
		store.pages[key] = entry
	}

	for code := range store.errorPages {
		entries := store.errorPages[code]
		slices.SortFunc(entries, func(a, b *errorPageEntry) int {
			return cmp.Compare(len(b.scopePath), len(a.scopePath))
		})
	}
	slices.SortFunc(store.rangeErrorPages, func(a, b *rangeErrorPageEntry) int {
		return cmp.Compare(len(b.scopePath), len(a.scopePath))
	})
	slices.SortFunc(store.catchAllErrorPages, func(a, b *errorPageEntry) int {
		return cmp.Compare(len(b.scopePath), len(a.scopePath))
	})
}

// partialJSArtefactIDsToSlice converts a single JS artefact ID to a slice. This
// handles the partial manifest entry format where partials have a single
// JSArtefactID field that needs to be converted to the unified JSArtefactIDs
// slice.
//
// Takes id (string) which is the JS artefact ID, or empty if no script exists.
//
// Returns []string which contains the ID as a single-element slice, or nil if
// empty.
func partialJSArtefactIDsToSlice(id string) []string {
	if id == "" {
		return nil
	}
	return []string{id}
}

// sortKeysByRouteSpecificity sorts manifest keys so that static routes come
// before dynamic routes. This order is needed for correct matching in Chi's
// radix tree router.
//
// Sort priority (highest to lowest):
//   - Static routes (no dynamic parts) - e.g., "/docs", "/about"
//   - Dynamic segment routes - e.g., "/docs/{id}", "/{slug}"
//   - Catch-all routes - e.g., "/docs/{path}*"
//
// Within each group, routes with more path segments come first. Alphabetical
// order is used as a tiebreaker for consistent results.
//
// Takes store (*ManifestStore) which contains the keys to sort in place.
func sortKeysByRouteSpecificity(store *ManifestStore) {
	slices.SortFunc(store.keys, func(a, b string) int {
		patternA := getPrimaryRoutePattern(store, a)
		patternB := getPrimaryRoutePattern(store, b)

		return cmp.Or(
			cmp.Compare(classifyRoutePattern(patternA), classifyRoutePattern(patternB)),
			cmp.Compare(countPathSegments(patternB), countPathSegments(patternA)),
			cmp.Compare(countStaticSegments(patternB), countStaticSegments(patternA)),
			cmp.Compare(a, b),
		)
	})
}

// sortKeysByRouteSpecificityWithLookup sorts route keys by how specific they
// are, with more specific routes first. This version accepts a lookup function
// instead of a store reference, so it works with both ManifestStore (compiled
// mode) and InterpretedManifestRunner (interpreted mode).
//
// Takes keys ([]string) which contains the route keys to sort in place.
// Takes lookup (func(string) *PageEntry) which returns the PageEntry for a
// given key, or nil if not found.
func sortKeysByRouteSpecificityWithLookup(keys []string, lookup func(string) *PageEntry) {
	slices.SortFunc(keys, func(a, b string) int {
		patternA := getPatternFromEntry(lookup(a))
		patternB := getPatternFromEntry(lookup(b))

		return cmp.Or(
			cmp.Compare(classifyRoutePattern(patternA), classifyRoutePattern(patternB)),
			cmp.Compare(countPathSegments(patternB), countPathSegments(patternA)),
			cmp.Compare(countStaticSegments(patternB), countStaticSegments(patternA)),
			cmp.Compare(a, b),
		)
	})
}

// getPatternFromEntry returns the main route pattern from a page entry.
//
// Takes entry (*PageEntry) which is the page entry to extract the pattern from.
//
// Returns string which is the canonical route pattern, or empty if entry is
// nil.
func getPatternFromEntry(entry *PageEntry) string {
	if entry == nil {
		return ""
	}
	return selectCanonicalRoutePattern(entry.RoutePatterns)
}

// getPrimaryRoutePattern returns the main route pattern for a manifest key.
//
// For pages, this returns a pattern from RoutePatterns, choosing the default
// locale when several are present. For partials, this is the single
// RoutePattern. For emails, returns an empty string as emails have no routes.
//
// When RoutePatterns has many entries (for example, with many locales), this
// function picks a pattern using locale order: empty key (default) first, then
// "en", then the first locale in alphabetical order.
//
// Takes store (*ManifestStore) which contains the manifest entries to search.
// Takes key (string) which identifies the manifest entry.
//
// Returns string which is the chosen route pattern, or empty if not found.
func getPrimaryRoutePattern(store *ManifestStore, key string) string {
	var patterns map[string]string

	if entry, ok := store.pages[key]; ok {
		patterns = entry.RoutePatterns
	} else if entry, ok := store.partials[key]; ok {
		patterns = entry.RoutePatterns
	}

	return selectCanonicalRoutePattern(patterns)
}

// selectCanonicalRoutePattern returns a consistent route pattern from a map of
// locale codes to patterns.
//
// It selects the pattern in this order: empty key (default locale), then "en",
// then the first locale in alphabetical order. This produces the same result
// each time, even though Go maps do not have a fixed iteration order.
//
// Takes patterns (map[string]string) which maps locale codes to route patterns.
//
// Returns string which is the selected pattern, or empty if the map is empty.
func selectCanonicalRoutePattern(patterns map[string]string) string {
	if len(patterns) == 0 {
		return ""
	}

	if pattern, ok := patterns[""]; ok {
		return pattern
	}

	if pattern, ok := patterns["en"]; ok {
		return pattern
	}

	locales := slices.Sorted(maps.Keys(patterns))

	return patterns[locales[0]]
}

// classifyRoutePattern finds what kind of route a pattern is based on its
// content.
//
// Takes pattern (string) which is the route pattern to check.
//
// Returns routeType which shows if the pattern is static, dynamic, or
// catch-all.
func classifyRoutePattern(pattern string) routeType {
	if pattern == "" {
		return routeTypeStatic
	}

	if strings.Contains(pattern, "*") {
		return routeTypeCatchAll
	}

	if strings.Contains(pattern, "{") {
		return routeTypeDynamic
	}

	return routeTypeStatic
}

// countPathSegments counts the number of path segments in a route pattern.
// For example, "/docs/api/v1" has 3 segments, while "/" has 0 segments.
//
// Takes pattern (string) which is the route pattern to count.
//
// Returns int which is the number of non-empty path segments.
func countPathSegments(pattern string) int {
	if pattern == "" || pattern == pathSeparator {
		return 0
	}

	trimmed := strings.Trim(pattern, pathSeparator)
	if trimmed == "" {
		return 0
	}

	return len(strings.Split(trimmed, pathSeparator))
}

// countStaticSegments counts the number of static (non-dynamic) segments in a
// route pattern. For example, "/docs/{id}/edit" has 2 static segments ("docs"
// and "edit").
//
// Takes pattern (string) which is the route pattern to analyse.
//
// Returns int which is the count of segments that do not contain placeholders.
func countStaticSegments(pattern string) int {
	if pattern == "" || pattern == pathSeparator {
		return 0
	}

	trimmed := strings.Trim(pattern, pathSeparator)
	if trimmed == "" {
		return 0
	}

	segments := strings.Split(trimmed, pathSeparator)
	count := 0
	for _, seg := range segments {
		if !strings.Contains(seg, "{") {
			count++
		}
	}
	return count
}

// buildErrorAST creates an AST that displays an error message in the browser,
// showing runtime failures to the user.
//
// Takes err (error) which is the error to display.
// Takes filePath (string) which is the name of the component that failed.
//
// Returns *ast_domain.TemplateAST which contains the styled error overlay.
func buildErrorAST(err error, filePath string) *ast_domain.TemplateAST {
	errorHTML := fmt.Sprintf(
		`<div style="z-index: 99999; position: fixed; top: 0; left: 0; right: 0; bottom: 0; `+
			`background: rgba(0,0,0,0.8); display: flex; align-items: center; justify-content: center;">`+
			`<div style="width: 80vw; max-width: 1200px; padding: 2rem; border-radius: 8px; `+
			`border: 2px solid #e11d48; background: #1c1917; font-family: ui-monospace, `+
			`SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace; `+
			`color: #f8fafc; max-height: 80vh; overflow-y: auto;">`+
			`<h3 style="color: #f43f5e; margin-bottom: 1rem; font-size: 1.5rem; `+
			`border-bottom: 1px solid #3f3f46; padding-bottom: 0.5rem;">Runtime Error</h3>`+
			`<p style="margin-bottom: 0.5rem; font-size: 1.1rem; color: #a1a1aa;">`+
			`Failed to execute component: <strong style="color: #e5e7eb;">%s</strong></p>`+
			`<pre style="background: #27272a; padding: 1rem; border-radius: 4px; `+
			`white-space: pre-wrap; word-wrap: break-word; font-size: 0.9rem; color: #d4d4d8; `+
			`border: 1px solid #3f3f46;">%s</pre>`+
			`</div>`+
			`</div>`,
		html.EscapeString(filePath),
		html.EscapeString(err.Error()),
	)

	return &ast_domain.TemplateAST{
		SourcePath:        nil,
		ExpiresAtUnixNano: nil,
		Metadata:          nil,
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:  ast_domain.NodeElement,
				TagName:   "div",
				InnerHTML: errorHTML,
			},
		},
		Diagnostics: nil,
		SourceSize:  0,
		Tidied:      false,
	}
}
