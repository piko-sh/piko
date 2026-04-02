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
	"strings"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// CompiledManifestRunner is a stateless adapter that handles requests in
// production mode. It implements ManifestRunnerPort by querying a pre-loaded,
// in-memory ManifestStore.
type CompiledManifestRunner struct {
	// store provides read access to manifest entries for page lookup.
	store templater_domain.ManifestStoreView

	// i18nService provides translation stores for request data; nil disables
	// translations.
	i18nService i18n_domain.Service

	// defaultLocale is the fallback locale for request parsing when none is
	// specified.
	defaultLocale string
}

var _ templater_domain.ManifestRunnerPort = (*CompiledManifestRunner)(nil)

const (
	// maxServerRedirectHops is the maximum number of server redirect hops allowed
	// before returning an error to prevent infinite loops.
	maxServerRedirectHops = 3
)

// RunPage handles a request for a page in production.
//
// It looks up the component, prepares the request context, and executes the
// component's compiled BuildAST function. It delegates to
// runPageWithRedirectLoop to handle ServerRedirect logic.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// render.
// Takes request (*http.Request) which provides the incoming HTTP request data.
//
// Returns *ast_domain.TemplateAST which is the rendered template structure.
// Returns templater_dto.InternalMetadata which contains internal page metadata.
// Returns string which is the final resolved page path.
// Returns error when the page cannot be rendered or a redirect loop is
// detected.
func (r *CompiledManifestRunner) RunPage(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return r.runPageWithRedirectLoop(ctx, pageDef, request, 0)
}

// RunPartial handles a request for a partial component. In the compiled
// runner, the lookup and execution logic is the same as for a page.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the partial
// component to render.
// Takes request (*http.Request) which provides the HTTP request context.
//
// Returns *ast_domain.TemplateAST which is the parsed template structure.
// Returns templater_dto.InternalMetadata which contains rendering metadata.
// Returns string which is the rendered output.
// Returns error when the partial component cannot be processed.
func (r *CompiledManifestRunner) RunPartial(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Trace("Delegating partial run to page run logic",
		logger_domain.String(logFieldPath, pageDef.OriginalPath))
	return r.RunPage(ctx, pageDef, request)
}

// RunPartialWithProps runs a partial and passes props to the compiled
// BuildAST function.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// render.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes props (any) which contains the properties to pass to the template.
//
// Returns *ast_domain.TemplateAST which is the rendered template AST.
// Returns templater_dto.InternalMetadata which contains page metadata.
// Returns string which is the page styling content.
// Returns error when the page entry cannot be found or request parsing fails.
func (r *CompiledManifestRunner) RunPartialWithProps(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
	props any,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	CompiledManifestRunnerRunPageCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		CompiledManifestRunnerRunPageDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	pageEntry, err := r.GetPageEntry(ctx, pageDef.OriginalPath)
	if err != nil {
		l.Error("Failed to get page entry from manifest store",
			logger_domain.Error(err),
			logger_domain.String(logFieldPath, pageDef.OriginalPath))
		CompiledManifestRunnerRunPageErrorCount.Add(ctx, 1)
		return nil, templater_dto.InternalMetadata{}, "", fmt.Errorf("getting page entry for %q: %w", pageDef.OriginalPath, err)
	}

	reqData, err := templater_domain.ParseRequestData(request, r.defaultLocale)
	if err != nil {
		l.Error("Failed to parse incoming request data",
			logger_domain.Error(err),
			logger_domain.String("path", pageDef.NormalisedPath))
		CompiledManifestRunnerRunPageErrorCount.Add(ctx, 1)
		return nil, templater_dto.InternalMetadata{}, "", fmt.Errorf("bad request for %s: %w", pageDef.NormalisedPath, err)
	}
	defer reqData.Release()

	if r.i18nService != nil {
		if store := r.i18nService.GetStore(); store != nil {
			reqData.SetI18n(store, pageEntry.GetLocalStore(), r.i18nService.GetStrBufPool())
		}
	}

	astRoot, internalMetadata := pageEntry.GetASTRootWithProps(reqData, props)

	internalMetadata.JSScriptMetas = pageEntry.GetJSScriptMetas()

	if internalMetadata.RenderError != nil {
		return nil, templater_dto.InternalMetadata{}, "", internalMetadata.RenderError
	}

	return astRoot, internalMetadata, pageEntry.GetStyling(), nil
}

// GetPageEntry retrieves a view of a single component's metadata from the
// store.
//
// Takes manifestKey (string) which identifies the component to retrieve.
//
// Returns templater_domain.PageEntryView which contains the component metadata.
// Returns error when the component with the given key is not found.
func (r *CompiledManifestRunner) GetPageEntry(ctx context.Context, manifestKey string) (templater_domain.PageEntryView, error) {
	CompiledManifestRunnerGetPageEntryCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		CompiledManifestRunnerGetPageEntryDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	ctx, l := logger_domain.From(ctx, log)
	pageEntry, found := r.store.GetPageEntry(manifestKey)
	if !found {
		err := fmt.Errorf("component with key '%s' not found in manifest", manifestKey)
		l.Warn("Component not found in manifest store",
			logger_domain.Error(err),
			logger_domain.String("key", manifestKey))
		CompiledManifestRunnerGetPageEntryErrorCount.Add(ctx, 1)
		return nil, err
	}
	return pageEntry, nil
}

// runPageWithRedirectLoop runs a page and follows server-side redirects.
// It calls itself again for each redirect, up to maxServerRedirectHops times.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to run.
// Takes request (*http.Request) which provides the HTTP request data.
// Takes hopCount (int) which tracks the current redirect depth.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns templater_dto.InternalMetadata which contains page metadata.
// Returns string which is the page styling content.
// Returns error when page preparation fails or too many redirects occur.
func (r *CompiledManifestRunner) runPageWithRedirectLoop(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
	hopCount int,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	if hopCount >= maxServerRedirectHops {
		return r.handleRedirectLoopError(ctx, pageDef.OriginalPath)
	}

	CompiledManifestRunnerRunPageCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		CompiledManifestRunnerRunPageDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	pageEntry, reqData, err := r.preparePageExecution(ctx, pageDef, request)
	if err != nil {
		return nil, templater_dto.InternalMetadata{}, "", fmt.Errorf("preparing page execution for %q: %w", pageDef.OriginalPath, err)
	}
	defer reqData.Release()

	l.Trace("Executing compiled BuildAST function for page",
		logger_domain.String(logFieldPath, pageDef.OriginalPath))
	astRoot, internalMetadata := pageEntry.GetASTRoot(reqData)
	internalMetadata.JSScriptMetas = pageEntry.GetJSScriptMetas()

	if internalMetadata.RenderError != nil {
		return nil, templater_dto.InternalMetadata{}, "", internalMetadata.RenderError
	}

	if internalMetadata.ServerRedirect != "" {
		return r.handleServerRedirect(ctx, internalMetadata.ServerRedirect, request, hopCount)
	}

	return astRoot, internalMetadata, pageEntry.GetStyling(), nil
}

// handleRedirectLoopError returns an error when the maximum redirect hops are
// exceeded.
//
// Takes originalPath (string) which is the path that caused the loop.
//
// Returns *ast_domain.TemplateAST which is always nil on error.
// Returns templater_dto.InternalMetadata which is always empty on error.
// Returns string which is always empty on error.
// Returns error when the redirect hop limit has been exceeded.
func (*CompiledManifestRunner) handleRedirectLoopError(
	ctx context.Context,
	originalPath string,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	err := fmt.Errorf("maximum server redirect hops (%d) exceeded for %s", maxServerRedirectHops, originalPath)
	l.Error("Server redirect loop detected",
		logger_domain.Error(err),
		logger_domain.String("path", originalPath))
	CompiledManifestRunnerRunPageErrorCount.Add(ctx, 1)
	return nil, templater_dto.InternalMetadata{}, "", err
}

// preparePageExecution retrieves the page entry and prepares request data.
//
// Takes pageDef (templater_dto.PageDefinition) which defines the page to load.
// Takes request (*http.Request) which provides the incoming HTTP request data.
//
// Returns templater_domain.PageEntryView which is the loaded page entry.
// Returns *templater_dto.RequestData which contains parsed request information.
// Returns error when the page entry cannot be found or request parsing fails.
func (r *CompiledManifestRunner) preparePageExecution(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
) (templater_domain.PageEntryView, *templater_dto.RequestData, error) {
	ctx, l := logger_domain.From(ctx, log)

	pageEntry, err := r.GetPageEntry(ctx, pageDef.OriginalPath)
	if err != nil {
		l.Error("Failed to get page entry from manifest store",
			logger_domain.Error(err),
			logger_domain.String(logFieldPath, pageDef.OriginalPath))
		CompiledManifestRunnerRunPageErrorCount.Add(ctx, 1)
		return nil, nil, fmt.Errorf("getting page entry for %q: %w", pageDef.OriginalPath, err)
	}

	reqData, err := templater_domain.ParseRequestData(request, r.defaultLocale)
	if err != nil {
		l.Error("Failed to parse incoming request data",
			logger_domain.Error(err),
			logger_domain.String("path", pageDef.NormalisedPath))
		CompiledManifestRunnerRunPageErrorCount.Add(ctx, 1)
		return nil, nil, fmt.Errorf("bad request for %s: %w", pageDef.NormalisedPath, err)
	}

	if r.i18nService != nil {
		if store := r.i18nService.GetStore(); store != nil {
			reqData.SetI18n(store, pageEntry.GetLocalStore(), r.i18nService.GetStrBufPool())
		}
	}

	return pageEntry, reqData, nil
}

// handleServerRedirect handles a server-side redirect by fetching the target
// page.
//
// Takes redirectURL (string) which is the target URL to redirect to.
// Takes request (*http.Request) which provides the original request context.
// Takes hopCount (int) which tracks the current redirect depth.
//
// Returns *ast_domain.TemplateAST which contains the parsed template from the
// redirected page.
// Returns templater_dto.InternalMetadata which holds metadata from processing.
// Returns string which provides the final resolved path.
// Returns error when the redirect fails or exceeds the maximum hop count.
func (r *CompiledManifestRunner) handleServerRedirect(
	ctx context.Context,
	redirectURL string,
	request *http.Request,
	hopCount int,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Trace("ServerRedirect detected, fetching alternate page",
		logger_domain.String("targetURL", redirectURL),
		logger_domain.Int("currentHop", hopCount))

	redirectPath := normaliseServerRedirectPath(redirectURL)
	newPageDef := templater_dto.PageDefinition{
		OriginalPath:   redirectPath,
		NormalisedPath: redirectURL,
		TemplateHTML:   "",
	}

	return r.runPageWithRedirectLoop(ctx, newPageDef, request, hopCount+1)
}

// NewCompiledManifestRunner creates a new production runner. Its sole
// dependency is the ManifestStoreView, which has already loaded and linked all
// components.
//
// Takes store (templater_domain.ManifestStoreView) which provides
// read access to compiled manifest entries.
// Takes i18nService (i18n_domain.Service) which provides translation
// stores for request data; nil disables translations.
// Takes defaultLocale (string) which is the fallback locale for
// request parsing when none is specified.
//
// Returns templater_domain.ManifestRunnerPort which is the configured
// runner ready for handling production requests.
func NewCompiledManifestRunner(store templater_domain.ManifestStoreView, i18nService i18n_domain.Service, defaultLocale string) templater_domain.ManifestRunnerPort {
	return &CompiledManifestRunner{
		store:         store,
		i18nService:   i18nService,
		defaultLocale: defaultLocale,
	}
}

// normaliseServerRedirectPath converts a URL path to a manifest key by
// removing the leading slash and adding the pages folder prefix with the .pk
// file extension.
//
// Takes urlPath (string) which is the URL path to convert (e.g. "/about" or
// "/").
//
// Returns string which is the manifest key (e.g. "pages/about.pk" or
// "pages/index.pk").
func normaliseServerRedirectPath(urlPath string) string {
	path := cmp.Or(strings.TrimPrefix(urlPath, "/"), "index")

	return "pages/" + path + ".pk"
}
