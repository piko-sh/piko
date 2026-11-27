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

package daemon_adapters

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

type stubEmailTemplateService struct {
	err  error
	html string
}

func (s *stubEmailTemplateService) Render(
	_ context.Context,
	_ *http.Request,
	_ string,
	_ any,
	_ *premailer.Options,
	_ bool,
) (*templater_dto.RenderedEmailContent, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &templater_dto.RenderedEmailContent{HTML: s.html}, nil
}

type stubPdfWriterService struct {
	pdfwriter_domain.PdfWriterService
	err     error
	content []byte
}

func (s *stubPdfWriterService) Render(
	_ context.Context,
	_ *http.Request,
	_ string,
	_ any,
	_ pdfwriter_dto.PdfConfig,
) (*pdfwriter_dto.PdfResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &pdfwriter_dto.PdfResult{Content: s.content}, nil
}

type previewHandlerDependencies struct {
	runner       templater_domain.ManifestRunnerPort
	renderer     templater_domain.RendererPort
	emailService templater_domain.EmailTemplateService
	pdfService   pdfwriter_domain.PdfWriterService
	registry     render_domain.RegistryPort
	scenarios    map[string][]templater_dto.PreviewScenario
}

func newPreviewRouter(t *testing.T, dependencies previewHandlerDependencies) *chi.Mux {
	t.Helper()

	store := &templater_domain.MockManifestStoreView{
		GetPageEntryFunc: func(path string) (templater_domain.PageEntryView, bool) {
			scenarios, ok := dependencies.scenarios[path]
			if !ok {
				return nil, false
			}
			return &templater_domain.MockPageEntryView{
				GetPreviewScenariosFunc: func() []templater_dto.PreviewScenario { return scenarios },
			}, true
		},
	}

	handler := NewDevPreviewHandler(store, dependencies.runner, dependencies.renderer,
		dependencies.emailService, dependencies.pdfService, dependencies.registry)
	router := chi.NewRouter()
	handler.Mount(router)
	return router
}

func servePreview(t *testing.T, router http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func singleScenario(path string) map[string][]templater_dto.PreviewScenario {
	return map[string][]templater_dto.PreviewScenario{
		path: {
			{Name: "default", Description: "", Props: map[string]any{"name": "Ada"}},
			{Name: "empty", Description: "No items", Props: nil},
		},
	}
}

func TestDevPreviewHandler_ListPreviews(t *testing.T) {
	t.Parallel()

	store := &templater_domain.MockManifestStoreView{
		ListPreviewEntriesFunc: func() []templater_domain.PreviewCatalogueEntry {
			return []templater_domain.PreviewCatalogueEntry{
				{OriginalSourcePath: "pdfs/invoice.pk", ComponentType: "pdf", Scenarios: []templater_dto.PreviewScenario{{Name: "paid"}}},
				{OriginalSourcePath: "emails/welcome.pk", ComponentType: "email", Scenarios: []templater_dto.PreviewScenario{{Name: "new", Description: "First visit"}}},
				{OriginalSourcePath: "pages/home.pk", ComponentType: "page", Scenarios: nil},
				{OriginalSourcePath: "other/thing.pk", ComponentType: "unknown", Scenarios: nil},
			}
		},
	}
	router := chi.NewRouter()
	NewDevPreviewHandler(store, nil, nil, nil, nil, nil).Mount(router)

	recorder := servePreview(t, router, "/_piko/dev/api/previews")

	require.Equal(t, http.StatusOK, recorder.Code)
	var body previewListResponse
	require.NoError(t, json.ConfigDefault.Unmarshal(recorder.Body.Bytes(), &body))

	types := make([]string, 0, len(body.Groups))
	for _, group := range body.Groups {
		types = append(types, group.Type)
	}
	assert.Equal(t, []string{"page", "email", "pdf"}, types, "groups follow a fixed order and unknown types are dropped")
	assert.Equal(t, "emails/welcome.pk", body.Groups[1].Templates[0].SourcePath)
	assert.Equal(t, []previewScenario{{Name: "new", Description: "First visit"}}, body.Groups[1].Templates[0].Scenarios)
}

func TestDevPreviewHandler_RenderPreviewScenarioLookup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenarios  map[string][]templater_dto.PreviewScenario
		name       string
		target     string
		wantBody   string
		wantStatus int
	}{
		{
			name:       "unknown template",
			scenarios:  nil,
			target:     "/_piko/dev/preview/email/missing",
			wantStatus: http.StatusNotFound,
			wantBody:   "template not found: emails/missing.pk",
		},
		{
			name:       "template without scenarios",
			scenarios:  map[string][]templater_dto.PreviewScenario{"emails/welcome.pk": nil},
			target:     "/_piko/dev/preview/email/welcome",
			wantStatus: http.StatusNotFound,
			wantBody:   "has no preview scenarios",
		},
		{
			name:       "unknown scenario",
			scenarios:  singleScenario("emails/welcome.pk"),
			target:     "/_piko/dev/preview/email/welcome?scenario=nope",
			wantStatus: http.StatusNotFound,
			wantBody:   `scenario "nope" not found`,
		},
		{
			name:       "unknown component type",
			scenarios:  singleScenario("widgets/clock"),
			target:     "/_piko/dev/preview/widgets/widgets/clock",
			wantStatus: http.StatusBadRequest,
			wantBody:   "unknown component type",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			router := newPreviewRouter(t, previewHandlerDependencies{scenarios: testCase.scenarios})

			recorder := servePreview(t, router, testCase.target)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), testCase.wantBody)
		})
	}
}

func TestDevPreviewHandler_RenderEmailPreview(t *testing.T) {
	t.Parallel()

	t.Run("injects the copy button before the closing body tag", func(t *testing.T) {
		t.Parallel()

		emailHTML := "<html><body><p>Hello</p></body></html>"
		router := newPreviewRouter(t, previewHandlerDependencies{
			scenarios:    singleScenario("emails/welcome.pk"),
			emailService: &stubEmailTemplateService{html: emailHTML},
		})

		recorder := servePreview(t, router, "/_piko/dev/preview/email/welcome?scenario=empty")

		require.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		assert.Equal(t, contentTypeHTML, recorder.Header().Get(headerContentType))
		assert.Contains(t, body, base64.StdEncoding.EncodeToString([]byte(emailHTML)))
		assert.Less(t, strings.Index(body, `id="piko-copy-email"`), strings.Index(body, "</body>"))
	})

	t.Run("appends the copy button when there is no body tag", func(t *testing.T) {
		t.Parallel()

		router := newPreviewRouter(t, previewHandlerDependencies{
			scenarios:    singleScenario("emails/welcome.pk"),
			emailService: &stubEmailTemplateService{html: "<p>Hello</p>"},
		})

		recorder := servePreview(t, router, "/_piko/dev/preview/email/welcome")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.True(t, strings.HasPrefix(recorder.Body.String(), "<p>Hello</p>"))
		assert.Contains(t, recorder.Body.String(), `id="piko-copy-email"`)
	})

	t.Run("answers 503 without an email service", func(t *testing.T) {
		t.Parallel()

		router := newPreviewRouter(t, previewHandlerDependencies{scenarios: singleScenario("emails/welcome.pk")})

		recorder := servePreview(t, router, "/_piko/dev/preview/email/welcome")

		assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	})

	t.Run("answers 500 when rendering fails", func(t *testing.T) {
		t.Parallel()

		router := newPreviewRouter(t, previewHandlerDependencies{
			scenarios:    singleScenario("emails/welcome.pk"),
			emailService: &stubEmailTemplateService{err: errors.New("template broke")},
		})

		recorder := servePreview(t, router, "/_piko/dev/preview/email/welcome")

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "email render failed")
	})
}

func TestDevPreviewHandler_RenderPdfPreview(t *testing.T) {
	t.Parallel()

	t.Run("wrapper page embeds the render URL", func(t *testing.T) {
		t.Parallel()

		router := newPreviewRouter(t, previewHandlerDependencies{scenarios: singleScenario("pdfs/invoice.pk")})

		recorder := servePreview(t, router, "/_piko/dev/preview/pdf/invoice?scenario=paid+in+full")

		assert.Equal(t, http.StatusNotFound, recorder.Code, "the scenario must exist before the wrapper is served")

		recorder = servePreview(t, router, "/_piko/dev/preview/pdf/invoice?scenario=empty")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `src="/_piko/dev/preview/pdf/invoice/render?scenario=empty"`)
		assert.Contains(t, recorder.Body.String(), "PDF Preview: pdfs/invoice.pk")
	})

	t.Run("wrapper page without a scenario has no query", func(t *testing.T) {
		t.Parallel()

		router := newPreviewRouter(t, previewHandlerDependencies{scenarios: singleScenario("pdfs/invoice.pk")})

		recorder := servePreview(t, router, "/_piko/dev/preview/pdf/invoice")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `src="/_piko/dev/preview/pdf/invoice/render"`)
	})

	t.Run("render returns the PDF bytes", func(t *testing.T) {
		t.Parallel()

		router := newPreviewRouter(t, previewHandlerDependencies{
			scenarios:  singleScenario("pdfs/invoice.pk"),
			pdfService: &stubPdfWriterService{content: []byte("%PDF-1.7")},
		})

		recorder := servePreview(t, router, "/_piko/dev/preview/pdf/invoice/render")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/pdf", recorder.Header().Get(headerContentType))
		assert.Equal(t, "SAMEORIGIN", recorder.Header().Get("X-Frame-Options"))
		assert.Equal(t, "%PDF-1.7", recorder.Body.String())
	})

	t.Run("render answers 503 without a PDF service", func(t *testing.T) {
		t.Parallel()

		router := newPreviewRouter(t, previewHandlerDependencies{scenarios: singleScenario("pdfs/invoice.pk")})

		recorder := servePreview(t, router, "/_piko/dev/preview/pdf/invoice/render")

		assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	})

	t.Run("render answers 500 when rendering fails", func(t *testing.T) {
		t.Parallel()

		router := newPreviewRouter(t, previewHandlerDependencies{
			scenarios:  singleScenario("pdfs/invoice.pk"),
			pdfService: &stubPdfWriterService{err: errors.New("layout broke")},
		})

		recorder := servePreview(t, router, "/_piko/dev/preview/pdf/invoice/render")

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "PDF render failed")
	})
}

func TestDevPreviewHandler_RenderComponentPreview(t *testing.T) {
	t.Parallel()

	runnerWith := func(err error) *templater_domain.MockManifestRunnerPort {
		return &templater_domain.MockManifestRunnerPort{
			RunPartialWithPropsFunc: func(
				context.Context, templater_dto.PageDefinition, *http.Request, any,
			) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
				return nil, templater_dto.InternalMetadata{}, "", err
			},
		}
	}

	testCases := []struct {
		runner       templater_domain.ManifestRunnerPort
		renderer     templater_domain.RendererPort
		name         string
		target       string
		wantBody     string
		wantStatus   int
		wantFragment bool
	}{
		{
			name:       "page renders in full",
			runner:     runnerWith(nil),
			target:     "/_piko/dev/preview/page/home",
			wantStatus: http.StatusOK,
			wantBody:   "rendered",
		},
		{
			name:         "partial renders as a fragment",
			runner:       runnerWith(nil),
			target:       "/_piko/dev/preview/partial/card",
			wantStatus:   http.StatusOK,
			wantBody:     "rendered",
			wantFragment: true,
		},
		{
			name:       "missing runner answers 503",
			runner:     nil,
			target:     "/_piko/dev/preview/page/home",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "not available",
		},
		{
			name:       "runner failure answers 500",
			runner:     runnerWith(errors.New("runner broke")),
			target:     "/_piko/dev/preview/page/home",
			wantStatus: http.StatusInternalServerError,
			wantBody:   "render failed: runner broke",
		},
		{
			name:   "renderer failure answers 500",
			runner: runnerWith(nil),
			renderer: &templater_domain.MockRendererPort{
				RenderPageFunc: func(context.Context, templater_domain.RenderPageParams) error {
					return errors.New("renderer broke")
				},
			},
			target:     "/_piko/dev/preview/page/home",
			wantStatus: http.StatusInternalServerError,
			wantBody:   "render failed: renderer broke",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var gotFragment bool
			renderer := testCase.renderer
			if renderer == nil {
				renderer = &templater_domain.MockRendererPort{
					RenderPageFunc: func(_ context.Context, params templater_domain.RenderPageParams) error {
						gotFragment = params.IsFragment
						_, err := params.Writer.Write([]byte("rendered"))
						return err
					},
				}
			}

			scenarios := singleScenario("pages/home.pk")
			scenarios["partials/card.pk"] = scenarios["pages/home.pk"]
			router := newPreviewRouter(t, previewHandlerDependencies{
				scenarios: scenarios,
				runner:    testCase.runner,
				renderer:  renderer,
			})

			recorder := servePreview(t, router, testCase.target)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), testCase.wantBody)
			assert.Equal(t, testCase.wantFragment, gotFragment)
		})
	}
}

func TestBuildSourcePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		componentType string
		path          string
		want          string
	}{
		{componentType: "email", path: "/welcome", want: "emails/welcome.pk"},
		{componentType: "pdf", path: "billing/invoice", want: "pdfs/billing/invoice.pk"},
		{componentType: "page", path: "home", want: "pages/home.pk"},
		{componentType: "partial", path: "card", want: "partials/card.pk"},
		{componentType: "other", path: "/raw/path", want: "raw/path"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.componentType, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, buildSourcePath(testCase.componentType, testCase.path))
		})
	}
}

func TestAddCopyScriptHashToCSP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		csp     string
		wantCSP string
	}{
		{
			name:    "no policy is left alone",
			csp:     "",
			wantCSP: "",
		},
		{
			name:    "existing script-src-elem gains the hash",
			csp:     "default-src 'self'; script-src-elem 'self'",
			wantCSP: "default-src 'self'; script-src-elem 'self' " + copyButtonScriptCSPHash,
		},
		{
			name:    "missing script-src-elem is added",
			csp:     "default-src 'self'",
			wantCSP: "default-src 'self'; script-src-elem 'self' " + copyButtonScriptCSPHash,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			if testCase.csp != "" {
				recorder.Header().Set("Content-Security-Policy", testCase.csp)
			}

			addCopyScriptHashToCSP(recorder)

			assert.Equal(t, testCase.wantCSP, recorder.Header().Get("Content-Security-Policy"))
		})
	}
}

func TestDevPreviewHandler_DevWidgetSnippet(t *testing.T) {
	daemon_frontend.SetDevWidgetHTML("<piko-dev-widget></piko-dev-widget>")
	t.Cleanup(func() { daemon_frontend.SetDevWidgetHTML("") })

	testCases := []struct {
		registry      render_domain.RegistryPort
		name          string
		wantComponent bool
	}{
		{name: "without a registry", registry: nil, wantComponent: false},
		{
			name: "with the widget component registered",
			registry: &render_domain.MockRegistryPort{
				GetComponentMetadataFunc: func(context.Context, string) (*render_dto.ComponentMetadata, error) {
					return &render_dto.ComponentMetadata{BaseJSPath: "/_piko/widget.js"}, nil
				},
			},
			wantComponent: true,
		},
		{
			name: "with a registry lookup that fails",
			registry: &render_domain.MockRegistryPort{
				GetComponentMetadataFunc: func(context.Context, string) (*render_dto.ComponentMetadata, error) {
					return nil, errors.New("not found")
				},
			},
			wantComponent: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := NewDevPreviewHandler(nil, nil, nil, nil, nil, testCase.registry)

			snippet := handler.devWidgetSnippet(t.Context())

			assert.True(t, strings.HasPrefix(snippet, "<piko-dev-widget></piko-dev-widget>"))
			assert.Contains(t, snippet, `"`+daemon_frontend.CoreModuleImportURL()+`"`)
			assert.Equal(t, testCase.wantComponent, strings.Contains(snippet, "/_piko/widget.js"))
		})
	}
}

func TestDevPreviewHandler_DevWidgetSnippetFollowsTheCSPToken(t *testing.T) {
	daemon_frontend.SetDevWidgetHTML("<piko-dev-widget></piko-dev-widget>")
	t.Cleanup(func() { daemon_frontend.SetDevWidgetHTML("") })

	testCases := []struct {
		name     string
		cspToken string
	}{
		{name: "every script carries the request token under a token-based policy", cspToken: "abc+/="},
		{name: "no script carries a token when the policy uses none", cspToken: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			registry := &render_domain.MockRegistryPort{
				GetComponentMetadataFunc: func(context.Context, string) (*render_dto.ComponentMetadata, error) {
					return &render_dto.ComponentMetadata{BaseJSPath: "/_piko/widget.js"}, nil
				},
			}
			handler := NewDevPreviewHandler(nil, nil, nil, nil, nil, registry)
			ctx := daemon_dto.WithPikoRequestCtx(t.Context(), &daemon_dto.PikoRequestCtx{CSPToken: testCase.cspToken})

			snippet := handler.devWidgetSnippet(ctx)

			scripts := moduleScriptTagPattern.FindAllString(snippet, -1)
			require.Len(t, scripts, 4)
			for _, script := range scripts {
				if testCase.cspToken == "" {
					assert.NotContains(t, script, "nonce=")
					continue
				}
				assert.Contains(t, script, ` nonce="`+testCase.cspToken+`"`)
			}
		})
	}
}

func TestDevPreviewHandler_DevWidgetSnippetIsEmptyOutsideDevMode(t *testing.T) {
	t.Parallel()

	handler := NewDevPreviewHandler(nil, nil, nil, nil, nil, nil)

	assert.Empty(t, handler.devWidgetSnippet(t.Context()))
}

var (
	esImportPattern        = regexp.MustCompile(`import\s*\{[^}]*\}\s*from\s*"([^"]+)"`)
	moduleScriptTagPattern = regexp.MustCompile(`<script type="module"[^>]*>`)
	moduleScriptPattern    = regexp.MustCompile(`<script type="module" src="([^"]+)"`)
)

func frameworkRuntimeURLs(pattern *regexp.Regexp, text string) []string {
	var urls []string
	for _, match := range pattern.FindAllStringSubmatch(text, -1) {
		bundle := strings.TrimPrefix(daemon_frontend.StripRuntimeVersion(match[1]), "/_piko/dist/")
		if strings.HasPrefix(bundle, "ppframework.core.") || strings.HasPrefix(bundle, "ppframework.components.") {
			urls = append(urls, match[1])
		}
	}
	return urls
}

func compiledComponentRuntimeURLs(t *testing.T) []string {
	t.Helper()

	source := []byte(`<template name="runtime-probe"><div>Probe</div></template>` +
		`<script lang="ts">const state = {};</script>`)
	artefact, err := compiler_domain.NewSFCCompiler("", nil).CompileSFC(t.Context(), "runtime-probe.pkc", source)
	require.NoError(t, err)

	urls := frameworkRuntimeURLs(esImportPattern, artefact.Files[artefact.BaseJSPath])
	require.NotEmpty(t, urls, "a compiled component must import the framework runtime")
	return urls
}

func TestDevPreviewHandler_LoadsTheRuntimeThatComponentsImport(t *testing.T) {
	daemon_frontend.SetDevWidgetHTML("<piko-dev-widget></piko-dev-widget>")
	t.Cleanup(func() { daemon_frontend.SetDevWidgetHTML("") })

	compiledURLs := compiledComponentRuntimeURLs(t)
	router := newPreviewRouter(t, previewHandlerDependencies{
		scenarios: map[string][]templater_dto.PreviewScenario{
			"emails/welcome.pk": {{Name: "default"}},
			"pdfs/invoice.pk":   {{Name: "default"}},
		},
		emailService: &stubEmailTemplateService{html: "<html><body><p>Hello</p></body></html>"},
	})

	for _, target := range []string{"/_piko/dev/preview/email/welcome", "/_piko/dev/preview/pdf/invoice"} {
		t.Run(target, func(t *testing.T) {
			recorder := servePreview(t, router, target)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.ElementsMatch(t, compiledURLs, frameworkRuntimeURLs(moduleScriptPattern, recorder.Body.String()),
				"the preview must load each runtime bundle once, at the URL compiled components import")
		})
	}
}

func TestDevPreviewHandler_PdfPreviewStylesFollowTheCSPToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		cspToken  string
		wantStyle string
	}{
		{name: "the page style carries the request token under a token-based policy", cspToken: "abc", wantStyle: `<style nonce="abc">body{margin:0;height:100vh}</style>`},
		{name: "the page style carries no token when the policy uses none", cspToken: "", wantStyle: `<style>body{margin:0;height:100vh}</style>`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			handler := NewDevPreviewHandler(nil, nil, nil, nil, nil, nil)
			request := httptest.NewRequest(http.MethodGet, "/_piko/dev/preview/pdf/invoice", nil)
			request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{CSPToken: testCase.cspToken}))
			recorder := httptest.NewRecorder()

			handler.renderPdfPreview(recorder, request, "pdfs/invoice.pk", nil, "")

			body := recorder.Body.String()
			assert.Contains(t, body, testCase.wantStyle)
			assert.NotContains(t, body, ` style="`)
		})
	}
}
