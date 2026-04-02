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
	"fmt"
	"net/http"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// logFieldOriginalPath is the log field key for the original request path.
	logFieldOriginalPath = "originalPath"
)

// templaterService provides template rendering and page probing functions.
// It implements TemplaterService and TemplaterRunnerSwapper interfaces.
type templaterService struct {
	// runner runs page and partial templates through the manifest.
	runner ManifestRunnerPort

	// renderer handles page and partial rendering and metadata collection.
	renderer RendererPort

	// i18nService provides translation and localisation support.
	i18nService i18n_domain.Service
}

// RenderPage renders a page template to the given writer.
//
// Takes req (RenderRequest) which bundles all values needed for rendering.
//
// Returns error when rendering fails.
func (t *templaterService) RenderPage(ctx context.Context, req RenderRequest) error {
	return t.renderGeneric(ctx, "Page", req, t.runner.RunPage, t.renderer.RenderPage)
}

// RenderPartial renders a partial page template to the given writer.
//
// Takes req (RenderRequest) which bundles all values needed for rendering.
//
// Returns error when rendering fails.
func (t *templaterService) RenderPartial(ctx context.Context, req RenderRequest) error {
	return t.renderGeneric(ctx, "Partial", req, t.runner.RunPartial, t.renderer.RenderPartial)
}

// ProbePage checks whether a page template can be rendered successfully.
//
// Takes page (templater_dto.PageDefinition) which specifies the page to probe.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes websiteConfig (*config.WebsiteConfig) which contains the website settings.
//
// Returns *templater_dto.PageProbeResult which contains the probe outcome.
// Returns error when the probe fails.
func (t *templaterService) ProbePage(
	ctx context.Context,
	page templater_dto.PageDefinition,
	request *http.Request,
	websiteConfig *config.WebsiteConfig,
) (*templater_dto.PageProbeResult, error) {
	return t.probeGeneric(ctx, "Page", page, request, websiteConfig)
}

// ProbePartial gets metadata for a partial template without rendering it.
//
// Takes partial (templater_dto.PageDefinition) which identifies the partial to
// probe.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes websiteConfig (*config.WebsiteConfig) which specifies the
// website settings.
//
// Returns *templater_dto.PageProbeResult which contains link headers and
// metadata for the partial.
// Returns error when the partial is not found in the manifest.
func (t *templaterService) ProbePartial(
	ctx context.Context,
	partial templater_dto.PageDefinition,
	request *http.Request,
	websiteConfig *config.WebsiteConfig,
) (*templater_dto.PageProbeResult, error) {
	return t.probeGeneric(ctx, "Partial", partial, request, websiteConfig)
}

// SetRunner assigns the manifest runner used for template execution.
//
// Takes r (ManifestRunnerPort) which provides the runner implementation.
func (t *templaterService) SetRunner(r ManifestRunnerPort) {
	t.runner = r
}

// renderGeneric renders a page definition to the provided writer.
//
// Takes label (string) which identifies the render operation for logging.
// Takes req (RenderRequest) which bundles all values needed for rendering.
// Takes run (func(...)) which executes the manifest runner for the page.
// Takes render (func(...)) which performs the final template rendering.
//
// Returns error when the runner fails or rendering cannot complete.
func (*templaterService) renderGeneric(
	ctx context.Context,
	label string,
	req RenderRequest,
	run func(context.Context, templater_dto.PageDefinition, *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error),
	render func(context.Context, RenderPageParams) error,
) error {
	ctx, l := logger_domain.From(ctx, log)

	cookieAccumulator := templater_dto.NewCookieAccumulator()
	runRequest := req.Request.WithContext(templater_dto.WithCookieAccumulator(req.Request.Context(), cookieAccumulator))

	l.Trace("Running manifest runner",
		logger_domain.String("phase", label),
		logger_domain.String(logFieldOriginalPath, req.Page.OriginalPath))
	ast, internalMeta, styling, err := run(ctx, req.Page, runRequest)

	for _, cookie := range cookieAccumulator.GetCookies() {
		http.SetCookie(req.Response, cookie)
	}

	if err != nil {
		l.Error("Manifest runner failed",
			logger_domain.String("phase", label),
			logger_domain.Error(err),
			logger_domain.String(logFieldOriginalPath, req.Page.OriginalPath))
		return fmt.Errorf("render%s: run%s failed for %s: %w", label, label, req.Page.OriginalPath, err)
	}

	if hasRedirect(&internalMeta) {
		l.Trace("Redirect detected in metadata, returning RedirectRequired",
			logger_domain.String("serverRedirect", internalMeta.ServerRedirect),
			logger_domain.String("clientRedirect", internalMeta.ClientRedirect))
		return &templater_dto.RedirectRequired{Metadata: internalMeta}
	}

	l.Trace("Rendering through renderer",
		logger_domain.String("phase", label),
		logger_domain.String(logFieldOriginalPath, req.Page.OriginalPath))
	return render(ctx, RenderPageParams{
		Writer:         req.Writer,
		ResponseWriter: req.Response,
		Request:        req.Request,
		PageDefinition: req.Page,
		TemplateAST:    ast,
		Metadata:       &internalMeta,
		IsFragment:     req.IsFragment,
		Config:         req.WebsiteConfig,
		Styling:        styling,
		ProbeData:      req.ProbeData,
	})
}

// probeGeneric gets page metadata and link headers without rendering the page.
//
// Takes label (string) which identifies the page type for logging.
// Takes page (templater_dto.PageDefinition) which identifies the page to probe.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes websiteConfig (*config.WebsiteConfig) which specifies the
// website settings.
//
// Returns *templater_dto.PageProbeResult which contains the link headers for
// the page.
// Returns error when the page is not found in the manifest.
func (t *templaterService) probeGeneric(
	ctx context.Context,
	label string,
	page templater_dto.PageDefinition,
	request *http.Request,
	websiteConfig *config.WebsiteConfig,
) (*templater_dto.PageProbeResult, error) {
	ctx, l := logger_domain.From(ctx, log)

	entry, err := t.runner.GetPageEntry(ctx, page.OriginalPath)
	if err != nil {
		l.Error(label+" not found in manifest",
			logger_domain.Error(err),
			logger_domain.String(logFieldOriginalPath, page.OriginalPath))
		return nil, fmt.Errorf("probe%s: %s not found in manifest: %s", label, label, page.OriginalPath)
	}

	staticMeta := entry.GetStaticMetadata()

	linkHeaders, probeData, err := t.renderer.CollectMetadata(ctx, request, staticMeta, websiteConfig)
	if err != nil {
		l.Warn("Failed to collect static link headers for "+label,
			logger_domain.Error(err))
	}

	return &templater_dto.PageProbeResult{
		LinkHeaders: linkHeaders,
		ProbeData:   probeData,
	}, nil
}

// NewTemplaterService creates a new templater service that orchestrates
// template rendering.
//
// Takes runner (ManifestRunnerPort) which executes manifest operations.
// Takes renderer (RendererPort) which renders templates.
// Takes i18nService (i18n_domain.Service) which provides internationalisation.
//
// Returns TemplaterService which is ready for use.
func NewTemplaterService(runner ManifestRunnerPort, renderer RendererPort, i18nService i18n_domain.Service) TemplaterService {
	return &templaterService{
		runner:      runner,
		renderer:    renderer,
		i18nService: i18nService,
	}
}

// hasRedirect checks if metadata contains any redirect directive.
//
// Takes meta (*templater_dto.InternalMetadata) which contains the page
// metadata to check.
//
// Returns bool which is true if ServerRedirect or ClientRedirect is set.
func hasRedirect(meta *templater_dto.InternalMetadata) bool {
	return meta.ServerRedirect != "" || meta.ClientRedirect != ""
}
