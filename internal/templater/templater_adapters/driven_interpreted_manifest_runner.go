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
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// InterpretedManifestRunner implements ManifestRunnerPort for development
// environments. It provides live-reloading template execution via lazy JIT
// compilation, deferring work until pages are visited for fast feedback.
type InterpretedManifestRunner struct {
	// progCache maps manifest keys to their cached page entries.
	progCache map[string]*PageEntry

	// i18nService provides translation support for templates.
	i18nService i18n_domain.Service

	// orchestrator triggers lazy compilation and provides cached entries.
	orchestrator JITCompiler

	// defaultLocale is the fallback locale used when parsing requests.
	defaultLocale string

	// cacheLock guards access to progCache.
	cacheLock sync.RWMutex
}

// JITCompiler defines the contract for on-demand template compilation.
// InterpretedBuildOrchestrator implements the port.
type JITCompiler interface {
	// JITCompile compiles the file at the given path on demand.
	//
	// Takes relPath (string) which is the relative path to the file to compile.
	//
	// Returns error when compilation fails.
	JITCompile(ctx context.Context, relPath string) error

	// GetCachedEntry retrieves a compiled page entry from the cache.
	// Uses proper locking when reading shared state.
	//
	// Takes relPath (string) which is the path to look up in the cache.
	//
	// Returns *PageEntry which is the cached page, or nil if not found.
	// Returns bool which is true if the entry exists in the cache.
	GetCachedEntry(relPath string) (*PageEntry, bool)

	// GetAllCachedKeys returns all keys stored in the orchestrator's cache.
	//
	// Returns []string which contains all cached key names.
	GetAllCachedKeys() []string
}

// RunPage serves a page request in interpreted mode.
//
// It performs lazy JIT compilation if the requested page is dirty, then
// serves the page. It delegates to runPageWithRedirectLoop to handle
// ServerRedirect logic.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// render.
// Takes request (*http.Request) which provides the incoming HTTP request.
//
// Returns *ast_domain.TemplateAST which contains the parsed template tree.
// Returns templater_dto.InternalMetadata which holds internal rendering state.
// Returns string which contains the rendered page content.
// Returns error when compilation fails or the page cannot be served.
func (r *InterpretedManifestRunner) RunPage(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return r.runPageWithRedirectLoop(ctx, pageDef, request, 0)
}

// RunPartial delegates to RunPage as the lookup and execution logic is the
// same.
//
// Takes pageDef (templater_dto.PageDefinition) which defines the page to
// render.
// Takes request (*http.Request) which provides the HTTP request context.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns templater_dto.InternalMetadata which contains rendering metadata.
// Returns string which is the rendered output.
// Returns error when page execution fails.
func (r *InterpretedManifestRunner) RunPartial(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return r.RunPage(ctx, pageDef, request)
}

// RunPartialWithProps runs a partial and passes props through to the
// interpreted BuildAST function. It performs lazy JIT compilation if the
// requested partial is dirty.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// render.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes props (any) which contains properties to pass to the BuildAST function.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns templater_dto.InternalMetadata which contains page metadata.
// Returns string which is the styling content for the page.
// Returns error when the page is not found or request data cannot be prepared.
func (r *InterpretedManifestRunner) RunPartialWithProps(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
	props any,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	InterpretedManifestRunnerRunPageCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		InterpretedManifestRunnerRunPageDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	r.triggerJITCompilation(ctx, pageDef.OriginalPath)

	pageEntry, found := r.lookupPageEntry(pageDef.OriginalPath)
	if !found {
		InterpretedManifestRunnerRunPageErrorCount.Add(ctx, 1)
		return nil, templater_dto.InternalMetadata{}, "", os.ErrNotExist
	}

	reqData, err := r.prepareRequestData(request, pageDef.NormalisedPath, pageEntry)
	if err != nil {
		InterpretedManifestRunnerRunPageErrorCount.Add(ctx, 1)
		return nil, templater_dto.InternalMetadata{}, "", fmt.Errorf("preparing request data for %q: %w", pageDef.NormalisedPath, err)
	}

	l.Trace("Executing interpreted BuildAST function with props",
		logger_domain.String("path", pageDef.OriginalPath))
	astRoot, internalMeta := pageEntry.GetASTRootWithProps(reqData, props)

	internalMeta.JSScriptMetas = pageEntry.GetJSScriptMetas()

	if internalMeta.RenderError != nil {
		reqData.Release()
		return nil, templater_dto.InternalMetadata{}, "", internalMeta.RenderError
	}

	reqData.Release()
	return astRoot, internalMeta, pageEntry.GetStyling(), nil
}

// GetPageEntry provides a simple lookup for component metadata.
//
// Takes manifestKey (string) which identifies the page entry to retrieve.
//
// Returns templater_domain.PageEntryView which contains the page metadata.
// Returns error when the manifest key is not found.
//
// Safe for concurrent use. Uses a read lock when accessing the local cache.
func (r *InterpretedManifestRunner) GetPageEntry(_ context.Context, manifestKey string) (templater_domain.PageEntryView, error) {
	if r.orchestrator != nil {
		entry, found := r.orchestrator.GetCachedEntry(manifestKey)
		if found {
			return entry, nil
		}
	}

	r.cacheLock.RLock()
	defer r.cacheLock.RUnlock()

	entry, found := r.progCache[manifestKey]
	if !found {
		return nil, os.ErrNotExist
	}
	return entry, nil
}

// GetKeys returns all known component paths in the cache.
//
// Keys are sorted by route specificity for correct Chi router matching,
// ensuring static routes are registered before dynamic routes.
//
// Returns []string which contains the sorted component paths.
//
// Safe for concurrent use when an orchestrator is present. When using the
// local cache fallback, a read lock protects access.
func (r *InterpretedManifestRunner) GetKeys() []string {
	var keys []string
	var entryLookup func(string) *PageEntry

	if r.orchestrator != nil {
		keys = r.orchestrator.GetAllCachedKeys()
		entryLookup = func(key string) *PageEntry {
			entry, _ := r.orchestrator.GetCachedEntry(key)
			return entry
		}
	} else {
		r.cacheLock.RLock()
		defer r.cacheLock.RUnlock()

		keys = make([]string, 0, len(r.progCache))
		for key := range r.progCache {
			keys = append(keys, key)
		}
		entryLookup = func(key string) *PageEntry {
			return r.progCache[key]
		}
	}

	sortKeysByRouteSpecificityWithLookup(keys, entryLookup)

	return keys
}

// GetPageEntryByPath provides lookup by path for ManifestStoreView interface.
//
// Takes path (string) which specifies the page path to look up.
//
// Returns templater_domain.PageEntryView which is the page entry if found.
// Returns bool which indicates whether the path was found.
//
// Safe for concurrent use. Uses read lock when falling back to local cache.
func (r *InterpretedManifestRunner) GetPageEntryByPath(path string) (templater_domain.PageEntryView, bool) {
	if r.orchestrator != nil {
		entry, found := r.orchestrator.GetCachedEntry(path)
		return entry, found
	}

	r.cacheLock.RLock()
	defer r.cacheLock.RUnlock()

	entry, found := r.progCache[path]
	return entry, found
}

// runPageWithRedirectLoop handles page execution with server-side redirect
// support. It recursively follows ServerRedirect metadata up to
// maxServerRedirectHops times.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// execute.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes hopCount (int) which tracks the current redirect depth.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns templater_dto.InternalMetadata which contains page metadata.
// Returns string which is the page styling content.
// Returns error when the page is not found or request preparation fails.
func (r *InterpretedManifestRunner) runPageWithRedirectLoop(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
	hopCount int,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	if hopCount >= maxServerRedirectHops {
		return r.handleRedirectLoopError(ctx, pageDef.OriginalPath)
	}

	InterpretedManifestRunnerRunPageCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		InterpretedManifestRunnerRunPageDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	r.triggerJITCompilation(ctx, pageDef.OriginalPath)

	pageEntry, found := r.lookupPageEntry(pageDef.OriginalPath)
	if !found {
		InterpretedManifestRunnerRunPageErrorCount.Add(ctx, 1)
		return nil, templater_dto.InternalMetadata{}, "", os.ErrNotExist
	}

	reqData, err := r.prepareRequestData(request, pageDef.NormalisedPath, pageEntry)
	if err != nil {
		InterpretedManifestRunnerRunPageErrorCount.Add(ctx, 1)
		return nil, templater_dto.InternalMetadata{}, "", fmt.Errorf("preparing request data for %q: %w", pageDef.NormalisedPath, err)
	}
	defer reqData.Release()

	l.Trace("Executing interpreted BuildAST function",
		logger_domain.String("path", pageDef.OriginalPath))
	astRoot, internalMeta := pageEntry.GetASTRoot(reqData)
	internalMeta.JSScriptMetas = pageEntry.GetJSScriptMetas()

	if internalMeta.RenderError != nil {
		return nil, templater_dto.InternalMetadata{}, "", internalMeta.RenderError
	}

	if internalMeta.ServerRedirect != "" {
		return r.handleServerRedirect(ctx, internalMeta.ServerRedirect, request, hopCount)
	}

	return astRoot, internalMeta, pageEntry.GetStyling(), nil
}

// handleRedirectLoopError returns an error when the redirect hop limit is
// reached.
//
// Takes originalPath (string) which is the path that caused the loop.
//
// Returns *ast_domain.TemplateAST which is always nil on error.
// Returns templater_dto.InternalMetadata which is always empty on error.
// Returns string which is always empty on error.
// Returns error when the redirect hop limit is reached.
func (*InterpretedManifestRunner) handleRedirectLoopError(
	ctx context.Context,
	originalPath string,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	err := fmt.Errorf("maximum server redirect hops (%d) exceeded for %s", maxServerRedirectHops, originalPath)
	l.Error("Server redirect loop detected",
		logger_domain.Error(err),
		logger_domain.String("path", originalPath))
	InterpretedManifestRunnerRunPageErrorCount.Add(ctx, 1)
	return nil, templater_dto.InternalMetadata{}, "", err
}

// triggerJITCompilation starts JIT compilation if the orchestrator is set.
//
// Takes path (string) which specifies the component path to compile.
func (r *InterpretedManifestRunner) triggerJITCompilation(ctx context.Context, path string) {
	if r.orchestrator == nil {
		return
	}
	ctx, l := logger_domain.From(ctx, log)
	if err := r.orchestrator.JITCompile(ctx, path); err != nil {
		l.Error("Failed to JIT compile component",
			logger_domain.Error(err),
			logger_domain.String(logFieldPath, path))
	}
}

// lookupPageEntry retrieves a page entry from the cache.
//
// Takes path (string) which specifies the cache key to look up.
//
// Returns *PageEntry which is the cached entry, or nil if not found.
// Returns bool which indicates whether the entry was found.
//
// Safe for concurrent use; protected by a read lock.
func (r *InterpretedManifestRunner) lookupPageEntry(path string) (*PageEntry, bool) {
	r.cacheLock.RLock()
	defer r.cacheLock.RUnlock()
	pageEntry, found := r.progCache[path]
	return pageEntry, found
}

// prepareRequestData parses request data and injects i18n and translations.
//
// Takes request (*http.Request) which provides the HTTP request to parse.
// Takes normalisedPath (string) which identifies the request path for errors.
// Takes pageEntry (*PageEntry) which provides local translations for the page.
//
// Returns *templater_dto.RequestData which contains the parsed request with
// i18n support.
// Returns error when the request data cannot be parsed.
func (r *InterpretedManifestRunner) prepareRequestData(
	request *http.Request,
	normalisedPath string,
	pageEntry *PageEntry,
) (*templater_dto.RequestData, error) {
	reqData, err := templater_domain.ParseRequestData(request, r.defaultLocale)
	if err != nil {
		return nil, fmt.Errorf("bad request for %s: %w", normalisedPath, err)
	}

	if r.i18nService != nil {
		if store := r.i18nService.GetStore(); store != nil {
			reqData.SetI18n(store, pageEntry.GetLocalStore(), r.i18nService.GetStrBufPool())
		}
	}

	return reqData, nil
}

// handleServerRedirect processes a server-side redirect by fetching the target
// page.
//
// Takes redirectURL (string) which specifies the target URL.
// Takes request (*http.Request) which provides the original request context.
// Takes hopCount (int) which tracks the current redirect depth.
//
// Returns *ast_domain.TemplateAST which contains the parsed template from the
// redirected page.
// Returns templater_dto.InternalMetadata which provides metadata from the
// redirected page.
// Returns string which contains the final resolved path.
// Returns error when the redirect fails or the hop limit is exceeded.
func (r *InterpretedManifestRunner) handleServerRedirect(
	ctx context.Context,
	redirectURL string,
	request *http.Request,
	hopCount int,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Trace("ServerRedirect detected, fetching alternate page",
		logger_domain.String("targetURL", redirectURL),
		logger_domain.Int("currentHop", hopCount))

	redirectPath := normaliseServerRedirectPathInterpreted(redirectURL)
	newPageDef := templater_dto.PageDefinition{
		OriginalPath:   redirectPath,
		NormalisedPath: redirectURL,
		TemplateHTML:   "",
	}

	return r.runPageWithRedirectLoop(ctx, newPageDef, request, hopCount+1)
}

// NewInterpretedManifestRunner creates a runner with a shared cache and JIT
// compiler for on-demand compilation.
//
// This constructor is called by DaemonService after the initial build to create
// the runner that will serve requests with lazy compilation support. The
// progCache is shared with the orchestrator and will be updated as components
// are lazily compiled.
//
// Takes i18nService (i18n_domain.Service) which provides translation support.
// Takes progCache (map[string]*PageEntry) which stores compiled page entries.
// Takes orchestrator (JITCompiler) which handles lazy compilation of components.
// Takes defaultLocale (string) which specifies the fallback locale.
//
// Returns templater_domain.ManifestRunnerPort which serves requests with lazy
// compilation support.
func NewInterpretedManifestRunner(
	i18nService i18n_domain.Service,
	progCache map[string]*PageEntry,
	orchestrator JITCompiler,
	defaultLocale string,
) templater_domain.ManifestRunnerPort {
	return &InterpretedManifestRunner{
		progCache:     progCache,
		i18nService:   i18nService,
		orchestrator:  orchestrator,
		defaultLocale: defaultLocale,
		cacheLock:     sync.RWMutex{},
	}
}

// normaliseServerRedirectPathInterpreted converts a URL path to a manifest key.
//
// Takes urlPath (string) which is the URL path to convert.
//
// Returns string which is the manifest key for the given path.
func normaliseServerRedirectPathInterpreted(urlPath string) string {
	path := cmp.Or(strings.TrimPrefix(urlPath, "/"), "index")

	return "pages/" + path + ".pk"
}
