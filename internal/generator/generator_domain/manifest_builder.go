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

package generator_domain

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/wdk/safedisk"
)

// rootURLPath is the root URL path prefix used when building route patterns.
const rootURLPath = "/"

// ManifestBuilder creates a complete Manifest from compiled and annotated
// artefacts. It applies rules to tell pages from partials and gathers their
// metadata into the final manifest structure.
type ManifestBuilder struct {
	// configSandbox provides file system access for reading the config file.
	// If nil, a sandbox is created from baseDir.
	configSandbox safedisk.Sandbox

	// configFactory creates sandboxes with validated paths. When set and
	// configSandbox is nil, the factory is used before falling back to
	// NewNoOpSandbox.
	configFactory safedisk.Factory

	// baseDir is the absolute path to the project root (where go.mod is found).
	// It is used to build paths relative to the project for manifest keys.
	baseDir string

	// pagesSourceDir is the directory name for page components.
	pagesSourceDir string

	// e2eSourceDir is the directory name for E2E test pages.
	e2eSourceDir string

	// baseServePath is the URL path prefix for serving pages.
	baseServePath string

	// i18nDefaultLocale is the default locale for i18n.
	i18nDefaultLocale string
}

// ManifestBuilderOption configures a ManifestBuilder during creation.
type ManifestBuilderOption func(*ManifestBuilder)

// NewManifestBuilder creates a new ManifestBuilder with the given settings.
//
// Takes pathsConfig (GeneratorPathsConfig) which provides path settings.
// Takes i18nDefaultLocale (string) which is the default locale for i18n.
// Takes baseDir (string) which specifies the base directory for calculating
// relative paths.
// Takes opts (...ManifestBuilderOption) which provides optional configuration
// such as WithConfigSandbox for testing.
//
// Returns *ManifestBuilder which is ready to build manifests.
func NewManifestBuilder(pathsConfig GeneratorPathsConfig, i18nDefaultLocale string, baseDir string, opts ...ManifestBuilderOption) *ManifestBuilder {
	mb := &ManifestBuilder{
		baseDir:           baseDir,
		pagesSourceDir:    pathsConfig.PagesSourceDir,
		e2eSourceDir:      pathsConfig.E2ESourceDir,
		baseServePath:     pathsConfig.BaseServePath,
		i18nDefaultLocale: i18nDefaultLocale,
		configSandbox:     nil,
	}

	for _, opt := range opts {
		opt(mb)
	}

	return mb
}

// Build creates a complete Manifest from the generated artefacts for a project.
// A pure transformation with no side effects.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains all
// generated artefacts for the project.
//
// Returns *generator_dto.Manifest which is the complete manifest object.
// Returns error when an artefact cannot be processed.
func (mb *ManifestBuilder) Build(artefacts []*generator_dto.GeneratedArtefact) (*generator_dto.Manifest, error) {
	manifest := &generator_dto.Manifest{
		Pages:      make(map[string]generator_dto.ManifestPageEntry),
		Partials:   make(map[string]generator_dto.ManifestPartialEntry),
		Emails:     make(map[string]generator_dto.ManifestEmailEntry),
		Pdfs:       make(map[string]generator_dto.ManifestPdfEntry),
		ErrorPages: make(map[string]generator_dto.ManifestErrorPageEntry),
	}

	jsResolver := newPartialJSDependencyResolver(artefacts)

	for _, artefact := range artefacts {
		if err := mb.processArtefact(artefact, manifest, jsResolver); err != nil {
			return nil, fmt.Errorf("processing artefact %q: %w", artefact.SuggestedPath, err)
		}
	}

	return manifest, nil
}

// processArtefact processes a single artefact and adds its entry to the
// manifest.
//
// Takes artefact (*generator_dto.GeneratedArtefact) which is the generated
// artefact to process.
// Takes manifest (*generator_dto.Manifest) which receives the new entry.
// Takes jsResolver (*partialJSDependencyResolver) which computes partial JS
// dependencies for pages.
//
// Returns error when the artefact is missing component metadata or the
// manifest key cannot be computed.
func (mb *ManifestBuilder) processArtefact(
	artefact *generator_dto.GeneratedArtefact,
	manifest *generator_dto.Manifest,
	jsResolver *partialJSDependencyResolver,
) error {
	vc := artefact.Component
	if vc == nil {
		return fmt.Errorf("failed to build manifest, artefact for path %s is missing component metadata", artefact.SuggestedPath)
	}

	manifestKey, err := mb.computeManifestKey(vc)
	if err != nil {
		return fmt.Errorf("computing manifest key for %q: %w", artefact.SuggestedPath, err)
	}

	if vc.IsErrorPage {
		mb.addErrorPageEntry(manifest, artefact, vc, manifestKey)
		return nil
	}

	switch vc.Source.ComponentType {
	case "page":
		mb.addPageEntry(manifest, artefact, vc, manifestKey, jsResolver)
	case "partial":
		mb.addPartialEntry(manifest, artefact, vc, manifestKey)
	case "email":
		mb.addEmailEntry(manifest, artefact, vc, manifestKey)
	case "pdf":
		mb.addPdfEntry(manifest, artefact, vc, manifestKey)
	}

	return nil
}

// computeManifestKey finds the unique key for a component in the manifest.
//
// Takes vc (*annotator_dto.VirtualComponent) which is the component to create
// a key for.
//
// Returns string which is the manifest key for the component.
// Returns error when the relative path cannot be computed for local components.
func (mb *ManifestBuilder) computeManifestKey(vc *annotator_dto.VirtualComponent) (string, error) {
	if vc.Source.IsExternal {
		return vc.Source.ModuleImportPath, nil
	}

	relPath, err := filepath.Rel(mb.baseDir, vc.Source.SourcePath)
	if err != nil {
		return "", fmt.Errorf("could not compute relative path for '%s': %w", vc.Source.SourcePath, err)
	}
	return filepath.ToSlash(relPath), nil
}

// addPageEntry creates and adds a page entry to the manifest. For virtual
// pages (created by collections), it makes one entry for each VirtualInstance.
//
// Takes manifest (*generator_dto.Manifest) which is the target manifest to
// populate.
// Takes artefact (*generator_dto.GeneratedArtefact) which provides the
// generated output data.
// Takes vc (*annotator_dto.VirtualComponent) which describes the component
// being added.
// Takes manifestKey (string) which identifies the entry in the manifest.
// Takes jsResolver (*partialJSDependencyResolver) which resolves partial JS
// dependencies.
func (mb *ManifestBuilder) addPageEntry(
	manifest *generator_dto.Manifest,
	artefact *generator_dto.GeneratedArtefact,
	vc *annotator_dto.VirtualComponent,
	manifestKey string,
	jsResolver *partialJSDependencyResolver,
) {
	if len(vc.VirtualInstances) > 0 {
		baseRoutePattern := mb.calculatePageRoutePattern(vc)
		for _, instance := range vc.VirtualInstances {
			mb.addVirtualPageEntry(manifest, artefact, vc, instance, jsResolver, baseRoutePattern)
		}
		if baseRoutePattern != "" {
			routePatterns, i18nStrategy := mb.computeRoutePatterns(baseRoutePattern, vc)
			manifest.CollectionFallbackRoutes = append(manifest.CollectionFallbackRoutes, generator_dto.CollectionFallbackRoute{
				RoutePatterns: routePatterns,
				I18nStrategy:  i18nStrategy,
			})
		}
		return
	}

	baseRoutePattern := mb.calculatePageRoutePattern(vc)
	routePatterns, i18nStrategy := mb.computeRoutePatterns(baseRoutePattern, vc)

	jsArtefactIDs := buildJSArtefactIDs(artefact.JSArtefactID, jsResolver.ResolveForPage(vc.HashedName))

	pageEntry := generator_dto.ManifestPageEntry{
		PackagePath:              vc.CanonicalGoPackagePath,
		OriginalSourcePath:       manifestKey,
		RoutePatterns:            routePatterns,
		I18nStrategy:             i18nStrategy,
		StyleBlock:               artefact.Result.StyleBlock,
		AssetRefs:                artefact.Result.AssetRefs,
		CustomTags:               artefact.Result.CustomTags,
		JSArtefactIDs:            jsArtefactIDs,
		HasCachePolicy:           vc.Source.Script.HasCachePolicy,
		CachePolicyFuncName:      vc.Source.Script.CachePolicyFuncName,
		HasMiddleware:            vc.Source.Script.HasMiddleware,
		MiddlewareFuncName:       vc.Source.Script.MiddlewaresFuncName,
		HasSupportedLocales:      vc.Source.Script.HasSupportedLocales,
		SupportedLocalesFuncName: vc.Source.Script.SupportedLocalesFuncName,
		LocalTranslations:        vc.Source.LocalTranslations,
		IsE2EOnly:                vc.IsE2EOnly,
		HasPreview:               vc.Source.Script.HasPreview,
		UsesCaptcha:              artefact.Result.UsesCaptcha,
	}
	manifest.Pages[manifestKey] = pageEntry
}

// addErrorPageEntry creates and adds an error page entry to the manifest.
// Error pages are not added to the Pages map  - they are stored separately
// in the ErrorPages map and rendered by the server when the corresponding
// HTTP error occurs.
//
// Takes manifest (*generator_dto.Manifest) which is the target manifest.
// Takes artefact (*generator_dto.GeneratedArtefact) which provides generated
// output data.
// Takes vc (*annotator_dto.VirtualComponent) which describes the error page
// component.
// Takes manifestKey (string) which identifies the entry in the manifest.
func (mb *ManifestBuilder) addErrorPageEntry(
	manifest *generator_dto.Manifest,
	artefact *generator_dto.GeneratedArtefact,
	vc *annotator_dto.VirtualComponent,
	manifestKey string,
) {
	scopePath := mb.calculateErrorPageScopePath(vc)

	entry := generator_dto.ManifestErrorPageEntry{
		PackagePath:        vc.CanonicalGoPackagePath,
		OriginalSourcePath: manifestKey,
		ScopePath:          scopePath,
		StyleBlock:         artefact.Result.StyleBlock,
		JSArtefactIDs:      buildJSArtefactIDs(artefact.JSArtefactID, nil),
		CustomTags:         artefact.Result.CustomTags,
		StatusCode:         vc.ErrorStatusCode,
		StatusCodeMin:      vc.ErrorStatusCodeMin,
		StatusCodeMax:      vc.ErrorStatusCodeMax,
		IsCatchAll:         vc.IsCatchAllError,
		IsE2EOnly:          vc.IsE2EOnly,
	}
	manifest.ErrorPages[manifestKey] = entry
}

// calculateErrorPageScopePath derives the URL path prefix that an error page
// covers, based on the error page's directory relative to the pages/ source
// directory.
//
// Takes vc (*annotator_dto.VirtualComponent) which is the error page component.
//
// Returns string which is the scope path prefix.
func (mb *ManifestBuilder) calculateErrorPageScopePath(vc *annotator_dto.VirtualComponent) string {
	var pagesSourceBase string
	if vc.IsE2EOnly {
		pagesSourceBase = filepath.Join(mb.baseDir, mb.e2eSourceDir, "pages")
	} else {
		pagesSourceBase = filepath.Join(mb.baseDir, mb.pagesSourceDir)
	}

	relToPages, err := filepath.Rel(pagesSourceBase, vc.Source.SourcePath)
	if err != nil {
		return "/"
	}

	directory := filepath.Dir(filepath.ToSlash(relToPages))
	if directory == "." || directory == "" {
		return "/"
	}

	return "/" + directory + "/"
}

// addVirtualPageEntry creates a manifest entry for a single virtual page
// instance.
//
// Takes manifest (*generator_dto.Manifest) which is the manifest to add the
// entry to.
// Takes artefact (*generator_dto.GeneratedArtefact) which provides the
// generated output data.
// Takes vc (*annotator_dto.VirtualComponent) which is the parent virtual
// component.
// Takes instance (annotator_dto.VirtualPageInstance) which specifies the page
// instance to register.
// Takes jsResolver (*partialJSDependencyResolver) which works out partial JS
// dependencies.
// Takes fallbackRoutePattern (string) which is the pre-computed route pattern
// to use when the instance has no explicit route.
func (mb *ManifestBuilder) addVirtualPageEntry(
	manifest *generator_dto.Manifest,
	artefact *generator_dto.GeneratedArtefact,
	vc *annotator_dto.VirtualComponent,
	instance annotator_dto.VirtualPageInstance,
	jsResolver *partialJSDependencyResolver,
	fallbackRoutePattern string,
) {
	baseRoutePattern := instance.Route
	if baseRoutePattern == "" {
		baseRoutePattern = fallbackRoutePattern
	}
	routePatterns, i18nStrategy := mb.computeRoutePatterns(baseRoutePattern, vc)

	jsArtefactIDs := buildJSArtefactIDs(artefact.JSArtefactID, jsResolver.ResolveForPage(vc.HashedName))

	pageEntry := generator_dto.ManifestPageEntry{
		PackagePath:              vc.CanonicalGoPackagePath,
		OriginalSourcePath:       instance.ManifestKey,
		RoutePatterns:            routePatterns,
		I18nStrategy:             i18nStrategy,
		StyleBlock:               artefact.Result.StyleBlock,
		AssetRefs:                artefact.Result.AssetRefs,
		CustomTags:               artefact.Result.CustomTags,
		JSArtefactIDs:            jsArtefactIDs,
		HasCachePolicy:           vc.Source.Script.HasCachePolicy,
		CachePolicyFuncName:      vc.Source.Script.CachePolicyFuncName,
		HasMiddleware:            vc.Source.Script.HasMiddleware,
		MiddlewareFuncName:       vc.Source.Script.MiddlewaresFuncName,
		HasSupportedLocales:      vc.Source.Script.HasSupportedLocales,
		SupportedLocalesFuncName: vc.Source.Script.SupportedLocalesFuncName,
		LocalTranslations:        vc.Source.LocalTranslations,
		IsE2EOnly:                vc.IsE2EOnly,
		HasPreview:               vc.Source.Script.HasPreview,
	}
	manifest.Pages[instance.ManifestKey] = pageEntry
}

// addPartialEntry creates and adds a partial entry to the manifest if the
// component is public.
//
// Takes manifest (*generator_dto.Manifest) which is the manifest to add the
// entry to.
// Takes artefact (*generator_dto.GeneratedArtefact) which provides the style
// and JavaScript artefact data.
// Takes vc (*annotator_dto.VirtualComponent) which is the virtual component
// to create an entry for.
// Takes manifestKey (string) which is the key to use in the manifest's
// partials map.
func (*ManifestBuilder) addPartialEntry(manifest *generator_dto.Manifest, artefact *generator_dto.GeneratedArtefact, vc *annotator_dto.VirtualComponent, manifestKey string) {
	if !vc.IsPublic {
		return
	}

	partialEntry := generator_dto.ManifestPartialEntry{
		PackagePath:        vc.CanonicalGoPackagePath,
		OriginalSourcePath: manifestKey,
		PartialName:        vc.PartialName,
		PartialSrc:         vc.PartialSrc,
		RoutePattern:       vc.PartialSrc,
		StyleBlock:         artefact.Result.StyleBlock,
		JSArtefactID:       artefact.JSArtefactID,
		IsE2EOnly:          vc.IsE2EOnly,
		HasPreview:         vc.Source != nil && vc.Source.Script != nil && vc.Source.Script.HasPreview,
	}
	manifest.Partials[manifestKey] = partialEntry
}

// addEmailEntry creates and adds an email entry to the manifest.
//
// Takes manifest (*generator_dto.Manifest) which is the manifest to add the
// entry to.
// Takes artefact (*generator_dto.GeneratedArtefact) which provides the style
// block for the entry.
// Takes vc (*annotator_dto.VirtualComponent) which provides the package path
// and locale settings.
// Takes manifestKey (string) which is the key used to store the email entry.
func (*ManifestBuilder) addEmailEntry(manifest *generator_dto.Manifest, artefact *generator_dto.GeneratedArtefact, vc *annotator_dto.VirtualComponent, manifestKey string) {
	emailEntry := generator_dto.ManifestEmailEntry{
		PackagePath:         vc.CanonicalGoPackagePath,
		OriginalSourcePath:  manifestKey,
		StyleBlock:          artefact.Result.StyleBlock,
		HasSupportedLocales: vc.Source.Script.HasSupportedLocales,
		LocalTranslations:   vc.Source.LocalTranslations,
		HasPreview:          vc.Source.Script.HasPreview,
	}
	manifest.Emails[manifestKey] = emailEntry
}

// addPdfEntry creates and adds a PDF entry to the manifest.
//
// Takes manifest (*generator_dto.Manifest) which is the manifest to add the
// entry to.
// Takes artefact (*generator_dto.GeneratedArtefact) which provides the style
// block for the entry.
// Takes vc (*annotator_dto.VirtualComponent) which provides the package path
// and locale settings.
// Takes manifestKey (string) which is the key used to store the PDF entry.
func (*ManifestBuilder) addPdfEntry(manifest *generator_dto.Manifest, artefact *generator_dto.GeneratedArtefact, vc *annotator_dto.VirtualComponent, manifestKey string) {
	pdfEntry := generator_dto.ManifestPdfEntry{
		PackagePath:         vc.CanonicalGoPackagePath,
		OriginalSourcePath:  manifestKey,
		StyleBlock:          artefact.Result.StyleBlock,
		HasSupportedLocales: vc.Source.Script.HasSupportedLocales,
		LocalTranslations:   vc.Source.LocalTranslations,
		HasPreview:          vc.Source.Script.HasPreview,
	}
	manifest.Pdfs[manifestKey] = pdfEntry
}

// calculatePageRoutePattern computes the Chi route pattern for a page
// component.
//
// Takes vc (*annotator_dto.VirtualComponent) which is the page component to
// calculate the route for.
//
// Returns string which is the URL route pattern, or empty if the component is
// not a public page.
func (mb *ManifestBuilder) calculatePageRoutePattern(vc *annotator_dto.VirtualComponent) string {
	if !vc.IsPage || !vc.IsPublic {
		return ""
	}

	var pagesSourceBase string
	if vc.IsE2EOnly {
		pagesSourceBase = filepath.Join(mb.baseDir, mb.e2eSourceDir, "pages")
	} else {
		pagesSourceBase = filepath.Join(mb.baseDir, mb.pagesSourceDir)
	}

	relToPages, err := filepath.Rel(pagesSourceBase, vc.Source.SourcePath)
	if err != nil {
		relToPages, _ = filepath.Rel(mb.baseDir, vc.Source.SourcePath)
	}
	relToPages = filepath.ToSlash(relToPages)

	route := strings.TrimSuffix(relToPages, ".pk")

	isIndex := false
	if strings.HasSuffix(route, "/index") {
		route = strings.TrimSuffix(route, "index")
		isIndex = true
	} else if route == "index" {
		route = ""
		isIndex = true
	}

	result := path.Join(rootURLPath, mb.baseServePath, route)

	if isIndex && !strings.HasSuffix(result, rootURLPath) {
		result += rootURLPath
	}

	return result
}

// computeRoutePatterns generates locale-specific route patterns for a page
// based on i18n configuration and the page's SupportedLocales setting.
//
// If the page has no SupportedLocales, returns a single pattern for the default
// locale only (opt-in model). If the page has SupportedLocales, generates
// patterns for all locales from WebsiteConfig. The runtime will call
// SupportedLocales to validate if a specific locale is actually supported.
//
// Takes basePattern (string) which is the base route pattern to localise.
// Takes vc (*annotator_dto.VirtualComponent) which provides the page metadata.
//
// Returns map[string]string which maps locale codes to their route patterns.
// Returns string which indicates the i18n strategy used (e.g. "disabled").
func (mb *ManifestBuilder) computeRoutePatterns(
	basePattern string,
	vc *annotator_dto.VirtualComponent,
) (map[string]string, string) {
	defaultLocale := mb.getDefaultLocale()

	websiteConfig, err := mb.loadWebsiteConfig()
	if err != nil || websiteConfig == nil || len(websiteConfig.I18n.Locales) == 0 {
		return map[string]string{defaultLocale: basePattern}, "disabled"
	}

	if websiteConfig.I18n.DefaultLocale != "" {
		defaultLocale = websiteConfig.I18n.DefaultLocale
	}

	if !vc.Source.Script.HasSupportedLocales {
		return map[string]string{defaultLocale: basePattern}, "disabled"
	}

	strategy := cmp.Or(websiteConfig.I18n.Strategy, "query-only")

	routePatterns := generateRoutesByStrategy(strategy, basePattern, defaultLocale, websiteConfig.I18n.Locales)
	return routePatterns, strategy
}

// getDefaultLocale returns the default locale for the manifest.
//
// Returns string which is the configured default locale, or "en" if none is
// set.
func (mb *ManifestBuilder) getDefaultLocale() string {
	if mb.i18nDefaultLocale != "" {
		return mb.i18nDefaultLocale
	}
	return "en"
}

// loadWebsiteConfig reads the website settings from the config.json file.
// This is needed at build time to find the i18n routing method.
//
// Returns *config.WebsiteConfig which holds the parsed settings, or nil if
// no config file exists.
// Returns error when the sandbox cannot be created or the config file cannot
// be read or parsed.
func (mb *ManifestBuilder) loadWebsiteConfig() (*config.WebsiteConfig, error) {
	const configFileName = "config.json"

	sandbox := mb.configSandbox
	var shouldClose bool
	if sandbox == nil {
		var err error
		if mb.configFactory != nil {
			sandbox, err = mb.configFactory.Create("manifest builder config", mb.baseDir, safedisk.ModeReadOnly)
		} else {
			sandbox, err = safedisk.NewNoOpSandbox(mb.baseDir, safedisk.ModeReadOnly)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create sandbox for base directory: %w", err)
		}
		shouldClose = true
	}
	if shouldClose {
		defer func() { _ = sandbox.Close() }()
	}

	data, err := sandbox.ReadFile(configFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config.json: %w", err)
	}

	var websiteConfig config.WebsiteConfig
	if err := json.Unmarshal(data, &websiteConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config.json: %w", err)
	}

	return &websiteConfig, nil
}

// WithConfigSandbox sets a custom sandbox for reading the config file. This
// allows tests to use mock sandboxes for filesystem operations.
//
// If not provided, a sandbox is created from baseDir when needed.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access for reading
// the config file.
//
// Returns ManifestBuilderOption which sets up the builder with the given
// sandbox.
func WithConfigSandbox(sandbox safedisk.Sandbox) ManifestBuilderOption {
	return func(mb *ManifestBuilder) {
		mb.configSandbox = sandbox
	}
}

// WithConfigFactory sets the sandbox factory for the manifest builder. When
// configSandbox is nil, the factory is tried before falling back to
// NewNoOpSandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes with validated
// paths.
//
// Returns ManifestBuilderOption which configures the builder with the factory.
func WithConfigFactory(factory safedisk.Factory) ManifestBuilderOption {
	return func(mb *ManifestBuilder) {
		mb.configFactory = factory
	}
}

// buildJSArtefactIDs combines the page's own JavaScript artefact ID with
// partial IDs into a single slice. The page's own script (if any) comes first,
// followed by partial scripts.
//
// Takes pageJSArtefactID (string) which is the page's JavaScript artefact ID.
// May be empty if the page has no client-side script.
// Takes partialJSArtefactIDs ([]string) which are the JavaScript artefact IDs
// from embedded partials.
//
// Returns []string containing all JavaScript artefact IDs, or nil if there
// are none.
func buildJSArtefactIDs(pageJSArtefactID string, partialJSArtefactIDs []string) []string {
	hasPageJS := pageJSArtefactID != ""
	hasPartialJS := len(partialJSArtefactIDs) > 0

	if !hasPageJS && !hasPartialJS {
		return nil
	}

	capacity := len(partialJSArtefactIDs)
	if hasPageJS {
		capacity++
	}

	result := make([]string, 0, capacity)

	if hasPageJS {
		result = append(result, pageJSArtefactID)
	}

	result = append(result, partialJSArtefactIDs...)

	return result
}

// generateRoutesByStrategy generates route patterns based on the i18n strategy.
// This is a pure function that does not need receiver state.
//
// Takes strategy (string) which specifies the i18n routing approach: "prefix",
// "prefix_except_default", or "query-only".
// Takes basePattern (string) which is the base URL path to generate routes for.
// Takes defaultLocale (string) which is the locale that may receive special
// treatment depending on strategy.
// Takes locales ([]string) which lists all locales to generate routes for.
//
// Returns map[string]string which maps each locale to its generated route
// pattern.
func generateRoutesByStrategy(strategy, basePattern, defaultLocale string, locales []string) map[string]string {
	routePatterns := make(map[string]string)

	switch strategy {
	case "prefix":
		for _, locale := range locales {
			routePatterns[locale] = path.Join(rootURLPath, locale, basePattern)
		}

	case "prefix_except_default":
		for _, locale := range locales {
			if locale == defaultLocale {
				routePatterns[locale] = basePattern
			} else {
				routePatterns[locale] = path.Join(rootURLPath, locale, basePattern)
			}
		}

	default:
		for _, locale := range locales {
			routePatterns[locale] = basePattern
		}
	}

	return routePatterns
}
