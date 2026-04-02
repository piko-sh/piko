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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func newRequestWithChiCtx(method, target string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	rctx := chi.NewRouteContext()
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))
}

type mockRenderService struct {
	renderASTFunction             func(ctx context.Context, w io.Writer, response http.ResponseWriter, request *http.Request, opts render_domain.RenderASTOptions) error
	renderEmailFunction           func(ctx context.Context, w io.Writer, request *http.Request, opts render_domain.RenderEmailOptions) error
	collectMetadataFunction       func(ctx context.Context, request *http.Request, metadata *templater_dto.InternalMetadata, siteConfig *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error)
	renderASTToPlainTextFunction  func(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error)
	getLastEmailAssetReqsFunction func() []*email_dto.EmailAssetRequest
	buildThemeCSSFunction         func(ctx context.Context, websiteConfig *config.WebsiteConfig) ([]byte, error)
}

func (m *mockRenderService) BuildThemeCSS(ctx context.Context, websiteConfig *config.WebsiteConfig) ([]byte, error) {
	if m.buildThemeCSSFunction != nil {
		return m.buildThemeCSSFunction(ctx, websiteConfig)
	}
	return nil, nil
}

func (m *mockRenderService) RenderAST(ctx context.Context, w io.Writer, response http.ResponseWriter, request *http.Request, opts render_domain.RenderASTOptions) error {
	if m.renderASTFunction != nil {
		return m.renderASTFunction(ctx, w, response, request, opts)
	}
	return nil
}

func (m *mockRenderService) RenderEmail(ctx context.Context, w io.Writer, request *http.Request, opts render_domain.RenderEmailOptions) error {
	if m.renderEmailFunction != nil {
		return m.renderEmailFunction(ctx, w, request, opts)
	}
	return nil
}

func (m *mockRenderService) CollectMetadata(ctx context.Context, request *http.Request, metadata *templater_dto.InternalMetadata, siteConfig *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
	if m.collectMetadataFunction != nil {
		return m.collectMetadataFunction(ctx, request, metadata, siteConfig)
	}
	return nil, nil, nil
}

func (m *mockRenderService) RenderASTToPlainText(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error) {
	if m.renderASTToPlainTextFunction != nil {
		return m.renderASTToPlainTextFunction(ctx, templateAST)
	}
	return "", nil
}

func (m *mockRenderService) GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest {
	if m.getLastEmailAssetReqsFunction != nil {
		return m.getLastEmailAssetReqsFunction()
	}
	return nil
}

func TestDrivenRenderer_RenderPage_DelegatesToService(t *testing.T) {
	t.Parallel()

	var capturedOpts render_domain.RenderASTOptions
	service := &mockRenderService{
		renderASTFunction: func(_ context.Context, _ io.Writer, _ http.ResponseWriter, _ *http.Request, opts render_domain.RenderASTOptions) error {
			capturedOpts = opts
			return nil
		},
	}

	renderer := NewDrivenRenderer(service)
	var buffer bytes.Buffer

	params := templater_domain.RenderPageParams{
		Writer: &buffer,
		PageDefinition: templater_dto.PageDefinition{
			OriginalPath: "pages/about.pk",
		},
		TemplateAST: &ast_domain.TemplateAST{},
		Metadata:    &templater_dto.InternalMetadata{},
		IsFragment:  true,
		Styling:     ".test { color: red; }",
	}

	err := renderer.RenderPage(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, "pages/about.pk", capturedOpts.PageID)
	assert.True(t, capturedOpts.IsFragment)
	assert.Equal(t, ".test { color: red; }", capturedOpts.Styling)
}

func TestDrivenRenderer_RenderPartial_DelegatesToService(t *testing.T) {
	t.Parallel()

	var capturedOpts render_domain.RenderASTOptions
	service := &mockRenderService{
		renderASTFunction: func(_ context.Context, _ io.Writer, _ http.ResponseWriter, _ *http.Request, opts render_domain.RenderASTOptions) error {
			capturedOpts = opts
			return nil
		},
	}

	renderer := NewDrivenRenderer(service)
	var buffer bytes.Buffer

	params := templater_domain.RenderPageParams{
		Writer: &buffer,
		PageDefinition: templater_dto.PageDefinition{
			OriginalPath: "partials/card.pk",
		},
		TemplateAST: &ast_domain.TemplateAST{},
		Metadata:    &templater_dto.InternalMetadata{},
	}

	err := renderer.RenderPartial(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, "partials/card.pk", capturedOpts.PageID)
}

func TestDrivenRenderer_RenderEmail_DelegatesToService(t *testing.T) {
	t.Parallel()

	var capturedOpts render_domain.RenderEmailOptions
	service := &mockRenderService{
		renderEmailFunction: func(_ context.Context, _ io.Writer, _ *http.Request, opts render_domain.RenderEmailOptions) error {
			capturedOpts = opts
			return nil
		},
	}

	renderer := NewDrivenRenderer(service)
	var buffer bytes.Buffer

	params := templater_domain.RenderEmailParams{
		Writer:   &buffer,
		PageID:   "emails/welcome",
		Metadata: &templater_dto.InternalMetadata{},
		Styling:  ".email { color: blue; }",
	}

	err := renderer.RenderEmail(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, "emails/welcome", capturedOpts.PageID)
	assert.Equal(t, ".email { color: blue; }", capturedOpts.Styling)
}

func TestDrivenRenderer_CollectMetadata_DelegatesToService(t *testing.T) {
	t.Parallel()

	expectedHeaders := []render_dto.LinkHeader{{Rel: "preload", URL: "/style.css"}}
	service := &mockRenderService{
		collectMetadataFunction: func(_ context.Context, _ *http.Request, _ *templater_dto.InternalMetadata, _ *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
			return expectedHeaders, nil, nil
		},
	}

	renderer := NewDrivenRenderer(service)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	headers, _, err := renderer.CollectMetadata(context.Background(), request, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, expectedHeaders, headers)
}

func TestDrivenRenderer_RenderASTToPlainText_DelegatesToService(t *testing.T) {
	t.Parallel()

	service := &mockRenderService{
		renderASTToPlainTextFunction: func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
			return "plain text output", nil
		},
	}

	renderer := NewDrivenRenderer(service)
	text, err := renderer.RenderASTToPlainText(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "plain text output", text)
}

func TestDrivenRenderer_GetLastEmailAssetRequests_DelegatesToService(t *testing.T) {
	t.Parallel()

	expectedRequests := []*email_dto.EmailAssetRequest{{}}
	service := &mockRenderService{
		getLastEmailAssetReqsFunction: func() []*email_dto.EmailAssetRequest {
			return expectedRequests
		},
	}

	renderer := NewDrivenRenderer(service)
	requests := renderer.GetLastEmailAssetRequests()
	assert.Equal(t, expectedRequests, requests)
}

func TestCompiledManifestRunner_GetPageEntry_Found(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/home.pk"

	entries := map[string]*PageEntry{"pages/home.pk": entry}
	store := &templater_domain.MockManifestStoreView{
		GetPageEntryFunc: func(path string) (templater_domain.PageEntryView, bool) {
			e, ok := entries[path]
			if !ok {
				return nil, false
			}
			return e, true
		},
	}

	runner := NewCompiledManifestRunner(store, nil, "en")
	result, err := runner.GetPageEntry(context.Background(), "pages/home.pk")
	require.NoError(t, err)
	assert.Equal(t, "pages/home.pk", result.GetOriginalPath())
}

func TestCompiledManifestRunner_GetPageEntry_NotFound(t *testing.T) {
	t.Parallel()

	store := &templater_domain.MockManifestStoreView{}

	runner := NewCompiledManifestRunner(store, nil, "en")
	_, err := runner.GetPageEntry(context.Background(), "pages/missing.pk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in manifest")
}

func TestCompiledManifestRunner_RunPage_PageNotFound(t *testing.T) {
	t.Parallel()

	store := &templater_domain.MockManifestStoreView{}

	runner := NewCompiledManifestRunner(store, nil, "en")
	request := newRequestWithChiCtx(http.MethodGet, "/about")
	pageDef := templater_dto.PageDefinition{
		OriginalPath: "pages/about.pk",
	}

	_, _, _, err := runner.RunPage(context.Background(), pageDef, request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pages/about.pk")
}

func TestCompiledManifestRunner_RunPartial_DelegatesToRunPage(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	entry.OriginalSourcePath = "partials/card.pk"
	entry.RoutePatterns = map[string]string{"": "/_piko/partial/card"}
	entry.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	})

	entries := map[string]*PageEntry{"partials/card.pk": entry}
	store := &templater_domain.MockManifestStoreView{
		GetPageEntryFunc: func(path string) (templater_domain.PageEntryView, bool) {
			e, ok := entries[path]
			if !ok {
				return nil, false
			}
			return e, true
		},
	}

	runner := NewCompiledManifestRunner(store, nil, "en")
	request := newRequestWithChiCtx(http.MethodGet, "/_piko/partial/card")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "partials/card.pk",
		NormalisedPath: "/_piko/partial/card",
	}

	ast, _, _, err := runner.RunPartial(context.Background(), pageDef, request)
	require.NoError(t, err)
	require.NotNil(t, ast)
}

func TestCompiledManifestRunner_RunPartialWithProps_PageNotFound(t *testing.T) {
	t.Parallel()

	store := &templater_domain.MockManifestStoreView{}

	runner := NewCompiledManifestRunner(store, nil, "en")
	request := newRequestWithChiCtx(http.MethodGet, "/test")
	pageDef := templater_dto.PageDefinition{
		OriginalPath: "partials/missing.pk",
	}

	_, _, _, err := runner.RunPartialWithProps(context.Background(), pageDef, request, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "partials/missing.pk")
}

func TestCompiledManifestRunner_RunPartialWithProps_Success(t *testing.T) {
	t.Parallel()

	var capturedProps any
	expectedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "div"},
		},
	}

	entry := &PageEntry{}
	entry.OriginalSourcePath = "partials/card.pk"
	entry.SetASTFunc(func(_ *templater_dto.RequestData, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		capturedProps = props
		return expectedAST, templater_dto.InternalMetadata{CustomTags: []string{"card"}}, nil
	})

	entries := map[string]*PageEntry{"partials/card.pk": entry}
	store := &templater_domain.MockManifestStoreView{
		GetPageEntryFunc: func(path string) (templater_domain.PageEntryView, bool) {
			e, ok := entries[path]
			if !ok {
				return nil, false
			}
			return e, true
		},
	}

	runner := NewCompiledManifestRunner(store, nil, "en")
	request := newRequestWithChiCtx(http.MethodGet, "/_piko/partial/card")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "partials/card.pk",
		NormalisedPath: "/_piko/partial/card",
	}

	myProps := map[string]string{"title": "Test"}
	ast, meta, _, err := runner.RunPartialWithProps(context.Background(), pageDef, request, myProps)
	require.NoError(t, err)
	assert.Equal(t, expectedAST, ast)
	assert.Equal(t, []string{"card"}, meta.CustomTags)
	assert.Equal(t, myProps, capturedProps)
}

func TestCompiledManifestRunner_RunPage_WithRedirect(t *testing.T) {
	t.Parallel()

	firstEntry := &PageEntry{}
	firstEntry.OriginalSourcePath = "pages/old.pk"
	firstEntry.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{ServerRedirect: "/new"},
		}, nil
	})

	secondEntry := &PageEntry{}
	secondEntry.OriginalSourcePath = "pages/new.pk"
	secondEntry.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
			},
		}, templater_dto.InternalMetadata{}, nil
	})

	entries := map[string]*PageEntry{
		"pages/old.pk": firstEntry,
		"pages/new.pk": secondEntry,
	}
	store := &templater_domain.MockManifestStoreView{
		GetPageEntryFunc: func(path string) (templater_domain.PageEntryView, bool) {
			e, ok := entries[path]
			if !ok {
				return nil, false
			}
			return e, true
		},
	}

	runner := NewCompiledManifestRunner(store, nil, "en")
	request := newRequestWithChiCtx(http.MethodGet, "/old")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "pages/old.pk",
		NormalisedPath: "/old",
	}

	ast, _, _, err := runner.RunPage(context.Background(), pageDef, request)
	require.NoError(t, err)
	require.NotNil(t, ast)
	assert.Len(t, ast.RootNodes, 1)
}

func TestCompiledManifestRunner_RunPage_RedirectLoop(t *testing.T) {
	t.Parallel()

	loopEntry := &PageEntry{}
	loopEntry.OriginalSourcePath = "pages/loop.pk"
	loopEntry.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{ServerRedirect: "/loop"},
		}, nil
	})

	entries := map[string]*PageEntry{"pages/loop.pk": loopEntry}
	store := &templater_domain.MockManifestStoreView{
		GetPageEntryFunc: func(path string) (templater_domain.PageEntryView, bool) {
			e, ok := entries[path]
			if !ok {
				return nil, false
			}
			return e, true
		},
	}

	runner := NewCompiledManifestRunner(store, nil, "en")
	request := newRequestWithChiCtx(http.MethodGet, "/loop")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "pages/loop.pk",
		NormalisedPath: "/loop",
	}

	_, _, _, err := runner.RunPage(context.Background(), pageDef, request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum server redirect hops")
}

type mockJITCompiler struct {
	entries map[string]*PageEntry
	keys    []string
}

func (m *mockJITCompiler) JITCompile(_ context.Context, _ string) error {
	return nil
}

func (m *mockJITCompiler) GetCachedEntry(relPath string) (*PageEntry, bool) {
	entry, found := m.entries[relPath]
	return entry, found
}

func (m *mockJITCompiler) GetAllCachedKeys() []string {
	return m.keys
}

func TestInterpretedManifestRunner_GetPageEntry_FromOrchestrator(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/home.pk"

	jit := &mockJITCompiler{
		entries: map[string]*PageEntry{
			"pages/home.pk": entry,
		},
	}

	runner := &InterpretedManifestRunner{
		progCache:    map[string]*PageEntry{},
		orchestrator: jit,
	}

	result, err := runner.GetPageEntry(context.Background(), "pages/home.pk")
	require.NoError(t, err)
	assert.Equal(t, "pages/home.pk", result.GetOriginalPath())
}

func TestInterpretedManifestRunner_GetPageEntry_FallbackToCache(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/local.pk"

	jit := &mockJITCompiler{
		entries: map[string]*PageEntry{},
	}

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/local.pk": entry,
		},
		orchestrator: jit,
	}

	result, err := runner.GetPageEntry(context.Background(), "pages/local.pk")
	require.NoError(t, err)
	assert.Equal(t, "pages/local.pk", result.GetOriginalPath())
}

func TestInterpretedManifestRunner_GetPageEntry_NotFound(t *testing.T) {
	t.Parallel()

	jit := &mockJITCompiler{
		entries: map[string]*PageEntry{},
	}

	runner := &InterpretedManifestRunner{
		progCache:    map[string]*PageEntry{},
		orchestrator: jit,
	}

	_, err := runner.GetPageEntry(context.Background(), "pages/missing.pk")
	require.Error(t, err)
}

func TestInterpretedManifestRunner_GetKeys_WithOrchestrator(t *testing.T) {
	t.Parallel()

	jit := &mockJITCompiler{
		keys: []string{"pages/b.pk", "pages/a.pk"},
		entries: map[string]*PageEntry{
			"pages/a.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/a"},
				},
			},
			"pages/b.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/b"},
				},
			},
		},
	}

	runner := &InterpretedManifestRunner{
		progCache:    map[string]*PageEntry{},
		orchestrator: jit,
	}

	keys := runner.GetKeys()
	require.Len(t, keys, 2)
	assert.Equal(t, "pages/a.pk", keys[0])
	assert.Equal(t, "pages/b.pk", keys[1])
}

func TestInterpretedManifestRunner_GetPageEntryByPath_WithOrchestrator(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/test.pk"

	jit := &mockJITCompiler{
		entries: map[string]*PageEntry{
			"pages/test.pk": entry,
		},
	}

	runner := &InterpretedManifestRunner{
		progCache:    map[string]*PageEntry{},
		orchestrator: jit,
	}

	t.Run("found in orchestrator", func(t *testing.T) {
		t.Parallel()
		pe, found := runner.GetPageEntryByPath("pages/test.pk")
		assert.True(t, found)
		assert.Equal(t, entry, pe)
	})

	t.Run("not found in orchestrator", func(t *testing.T) {
		t.Parallel()
		pe, found := runner.GetPageEntryByPath("pages/missing.pk")
		assert.False(t, found)
		assert.Nil(t, pe)
	})
}

func TestInterpretedManifestRunner_RunPage_PageNotFound(t *testing.T) {
	t.Parallel()

	runner := &InterpretedManifestRunner{
		progCache:    map[string]*PageEntry{},
		orchestrator: nil,
	}

	request := newRequestWithChiCtx(http.MethodGet, "/missing")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "pages/missing.pk",
		NormalisedPath: "/missing",
	}

	_, _, _, err := runner.RunPage(context.Background(), pageDef, request)
	require.Error(t, err)
}

func TestInterpretedManifestRunner_RunPage_Success(t *testing.T) {
	t.Parallel()

	expectedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "h1"},
		},
	}

	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/home.pk"
	entry.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return expectedAST, templater_dto.InternalMetadata{}, nil
	})

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/home.pk": entry,
		},
		orchestrator:  nil,
		defaultLocale: "en",
	}

	request := newRequestWithChiCtx(http.MethodGet, "/home")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "pages/home.pk",
		NormalisedPath: "/home",
	}

	ast, _, _, err := runner.RunPage(context.Background(), pageDef, request)
	require.NoError(t, err)
	assert.Equal(t, expectedAST, ast)
}

func TestInterpretedManifestRunner_RunPartialWithProps_PageNotFound(t *testing.T) {
	t.Parallel()

	runner := &InterpretedManifestRunner{
		progCache:    map[string]*PageEntry{},
		orchestrator: nil,
	}

	request := newRequestWithChiCtx(http.MethodGet, "/test")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "partials/missing.pk",
		NormalisedPath: "/_piko/partial/missing",
	}

	_, _, _, err := runner.RunPartialWithProps(context.Background(), pageDef, request, nil)
	require.Error(t, err)
}

func TestInterpretedManifestRunner_RunPartialWithProps_Success(t *testing.T) {
	t.Parallel()

	var capturedProps any
	entry := &PageEntry{}
	entry.OriginalSourcePath = "partials/card.pk"
	entry.SetASTFunc(func(_ *templater_dto.RequestData, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		capturedProps = props
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	})

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"partials/card.pk": entry,
		},
		orchestrator:  nil,
		defaultLocale: "en",
	}

	request := newRequestWithChiCtx(http.MethodGet, "/_piko/partial/card")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "partials/card.pk",
		NormalisedPath: "/_piko/partial/card",
	}

	myProps := map[string]string{"key": "val"}
	_, _, _, err := runner.RunPartialWithProps(context.Background(), pageDef, request, myProps)
	require.NoError(t, err)
	assert.Equal(t, myProps, capturedProps)
}

func TestInterpretedManifestRunner_RunPage_WithRedirect(t *testing.T) {
	t.Parallel()

	firstEntry := &PageEntry{}
	firstEntry.OriginalSourcePath = "pages/old.pk"
	firstEntry.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{ServerRedirect: "/new"},
		}, nil
	})

	secondEntry := &PageEntry{}
	secondEntry.OriginalSourcePath = "pages/new.pk"
	secondEntry.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "new page"},
			},
		}, templater_dto.InternalMetadata{}, nil
	})

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/old.pk": firstEntry,
			"pages/new.pk": secondEntry,
		},
		orchestrator:  nil,
		defaultLocale: "en",
	}

	request := newRequestWithChiCtx(http.MethodGet, "/old")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "pages/old.pk",
		NormalisedPath: "/old",
	}

	ast, _, _, err := runner.RunPage(context.Background(), pageDef, request)
	require.NoError(t, err)
	require.NotNil(t, ast)
	assert.Len(t, ast.RootNodes, 1)
}

func TestInterpretedManifestRunner_RunPage_RedirectLoop(t *testing.T) {
	t.Parallel()

	loopEntry := &PageEntry{}
	loopEntry.OriginalSourcePath = "pages/loop.pk"
	loopEntry.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{ServerRedirect: "/loop"},
		}, nil
	})

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/loop.pk": loopEntry,
		},
		orchestrator:  nil,
		defaultLocale: "en",
	}

	request := newRequestWithChiCtx(http.MethodGet, "/loop")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "pages/loop.pk",
		NormalisedPath: "/loop",
	}

	_, _, _, err := runner.RunPage(context.Background(), pageDef, request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum server redirect hops")
}

func TestInterpretedManifestRunner_TriggerJITCompilation_WithOrchestrator(t *testing.T) {
	t.Parallel()

	jit := &mockJITCompiler{
		entries: map[string]*PageEntry{},
	}

	runner := &InterpretedManifestRunner{
		progCache:    map[string]*PageEntry{},
		orchestrator: jit,
	}

	runner.triggerJITCompilation(context.Background(), "pages/test.pk")
}

func TestInterpretedManifestRunner_PrepareRequestData_NilRequest(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	runner := &InterpretedManifestRunner{
		defaultLocale: "en",
	}

	reqData, err := runner.prepareRequestData(nil, "/test", entry)
	require.NoError(t, err)
	require.NotNil(t, reqData)
	reqData.Release()
}

type mockManifestRunner struct {
	runPageFunction             func(ctx context.Context, pageDef templater_dto.PageDefinition, request *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)
	runPartialFunction          func(ctx context.Context, pageDef templater_dto.PageDefinition, request *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)
	runPartialWithPropsFunction func(ctx context.Context, pageDef templater_dto.PageDefinition, request *http.Request, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)
	getPageEntryFunction        func(ctx context.Context, manifestKey string) (templater_domain.PageEntryView, error)
}

func (m *mockManifestRunner) RunPage(ctx context.Context, pageDef templater_dto.PageDefinition, request *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	if m.runPageFunction != nil {
		return m.runPageFunction(ctx, pageDef, request)
	}
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockManifestRunner) RunPartial(ctx context.Context, pageDef templater_dto.PageDefinition, request *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	if m.runPartialFunction != nil {
		return m.runPartialFunction(ctx, pageDef, request)
	}
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockManifestRunner) RunPartialWithProps(ctx context.Context, pageDef templater_dto.PageDefinition, request *http.Request, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	if m.runPartialWithPropsFunction != nil {
		return m.runPartialWithPropsFunction(ctx, pageDef, request, props)
	}
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockManifestRunner) GetPageEntry(ctx context.Context, manifestKey string) (templater_domain.PageEntryView, error) {
	if m.getPageEntryFunction != nil {
		return m.getPageEntryFunction(ctx, manifestKey)
	}
	return nil, nil
}

func TestCachingManifestRunner_GetPageEntry_Delegates(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/test.pk"

	next := &mockManifestRunner{
		getPageEntryFunction: func(_ context.Context, key string) (templater_domain.PageEntryView, error) {
			if key == "pages/test.pk" {
				return entry, nil
			}
			return nil, nil
		},
	}

	runner := NewCachingManifestRunner(next, nil)
	result, err := runner.GetPageEntry(context.Background(), "pages/test.pk")
	require.NoError(t, err)
	assert.Equal(t, "pages/test.pk", result.GetOriginalPath())
}

func TestCachingManifestRunner_RunPartialWithProps_BypassesCache(t *testing.T) {
	t.Parallel()

	var capturedProps any
	next := &mockManifestRunner{
		runPartialWithPropsFunction: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
			capturedProps = props
			return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, "", nil
		},
	}

	runner := NewCachingManifestRunner(next, nil)
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	pageDef := templater_dto.PageDefinition{
		OriginalPath: "partials/card.pk",
	}

	myProps := "test-props"
	_, _, _, err := runner.RunPartialWithProps(context.Background(), pageDef, request, myProps)
	require.NoError(t, err)
	assert.Equal(t, myProps, capturedProps)
}

func TestPageEntry_GetCachePolicy_WithFunc(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.SetCachePolicyFunc(func(_ *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{Enabled: true, MaxAgeSeconds: 300}
	})
	policy := pe.GetCachePolicy(nil)
	assert.True(t, policy.Enabled)
	assert.Equal(t, 300, policy.MaxAgeSeconds)
}

func TestPageEntry_GetCachePolicy_DisabledByDefault(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.SetCachePolicyFunc(func(_ *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{}
	})
	policy := pe.GetCachePolicy(nil)
	assert.False(t, policy.Enabled)
}

func TestPageEntry_GetMiddlewares_EmptyList(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.SetMiddlewareFunc(func() []func(http.Handler) http.Handler {
		return nil
	})

	middlewares := pe.GetMiddlewares()
	assert.Nil(t, middlewares)
}

func TestPageEntry_GetMiddlewares_WithFunc(t *testing.T) {
	t.Parallel()

	handler := func(next http.Handler) http.Handler {
		return next
	}
	pe := &PageEntry{}
	pe.SetMiddlewareFunc(func() []func(http.Handler) http.Handler {
		return []func(http.Handler) http.Handler{handler}
	})

	middlewares := pe.GetMiddlewares()
	require.Len(t, middlewares, 1)
}

func TestPageEntry_GetSupportedLocales_EmptyList(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.SetSupportedLocalesFunc(func() []string {
		return nil
	})
	locales := pe.GetSupportedLocales()
	assert.Nil(t, locales)
}

func TestPageEntry_GetSupportedLocales_WithLocales(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.SetSupportedLocalesFunc(func() []string {
		return []string{"en", "fr", "de"}
	})
	locales := pe.GetSupportedLocales()
	assert.Equal(t, []string{"en", "fr", "de"}, locales)
}

func TestCachingManifestRunner_ASTRoundTripThroughDisk(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping AST disk round-trip test in short mode")
	}

	t.Parallel()

	tempDir := t.TempDir()
	l1TTL := 150 * time.Millisecond

	cacheService, err := ast_adapters.NewASTCacheService(context.Background(), ast_adapters.ASTCacheConfig{
		L1CacheCapacity: 10,
		L1CacheTTL:      l1TTL,
		L2CacheBaseDir:  tempDir,
	})
	require.NoError(t, err)
	t.Cleanup(func() { cacheService.Shutdown(context.Background()) })

	complexAST := &ast_domain.TemplateAST{
		SourcePath: new("/app/pages/cached.pk"),
		Tidied:     true,
		SourceSize: 8000,
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "id", Value: "content"},
					{Name: "class", Value: "page-wrapper"},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "h1",
						DirText: &ast_domain.Directive{
							Type:          ast_domain.DirectiveText,
							RawExpression: "state.Title",
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "ul",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "li",
								DirFor: &ast_domain.Directive{
									Type:          ast_domain.DirectiveFor,
									Arg:           "item",
									RawExpression: "item in state.Items",
								},
								DirKey: &ast_domain.Directive{
									Type:          ast_domain.DirectiveKey,
									RawExpression: "item.ID",
								},
								RichText: []ast_domain.TextPart{
									{IsLiteral: false, RawExpression: "item.Name"},
								},
							},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "button",
						OnEvents: map[string][]ast_domain.Directive{
							"click": {{Type: ast_domain.DirectiveOn, Arg: "click", RawExpression: "handleSubmit"}},
						},
						Children: []*ast_domain.TemplateNode{
							{NodeType: ast_domain.NodeText, TextContent: "Submit"},
						},
					},
				},
			},
		},
	}

	expectedMetadata := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title: "Cached Page",
		},
		CustomTags: []string{"card"},
	}

	missCallCount := 0
	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/cached.pk"
	entry.HasCachePolicy = true
	entry.SetCachePolicyFunc(func(*templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{Enabled: true, MaxAgeSeconds: 3600}
	})

	next := &mockManifestRunner{
		getPageEntryFunction: func(_ context.Context, key string) (templater_domain.PageEntryView, error) {
			if key == "pages/cached.pk" {
				return entry, nil
			}
			return nil, fmt.Errorf("not found: %s", key)
		},
		runPageFunction: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
			missCallCount++
			return complexAST.DeepClone(), expectedMetadata, "", nil
		},
	}

	runner := NewCachingManifestRunner(next, cacheService)
	request := newRequestWithChiCtx(http.MethodGet, "/cached")
	pageDef := templater_dto.PageDefinition{
		OriginalPath:   "pages/cached.pk",
		NormalisedPath: "/cached",
	}

	ast1, meta1, _, err := runner.RunPage(context.Background(), pageDef, request)
	require.NoError(t, err)
	require.NotNil(t, ast1)
	assert.Equal(t, 1, missCallCount)
	assert.Equal(t, expectedMetadata.Title, meta1.Title)
	assert.Equal(t, expectedMetadata.CustomTags, meta1.CustomTags)

	require.Len(t, ast1.RootNodes, 1)
	assert.Equal(t, "div", ast1.RootNodes[0].TagName)
	require.Len(t, ast1.RootNodes[0].Children, 3)
	assert.Equal(t, "h1", ast1.RootNodes[0].Children[0].TagName)

	time.Sleep(l1TTL + 200*time.Millisecond)

	req2 := newRequestWithChiCtx(http.MethodGet, "/cached")
	ast2, meta2, _, err := runner.RunPage(context.Background(), pageDef, req2)
	require.NoError(t, err)
	require.NotNil(t, ast2)
	assert.Equal(t, 1, missCallCount)

	assert.Equal(t, meta1.Title, meta2.Title)
	assert.Equal(t, meta1.CustomTags, meta2.CustomTags)

	require.Len(t, ast2.RootNodes, 1)
	root := ast2.RootNodes[0]
	assert.Equal(t, "div", root.TagName)
	require.Len(t, root.Attributes, 2)
	assert.Equal(t, "id", root.Attributes[0].Name)
	assert.Equal(t, "content", root.Attributes[0].Value)
	assert.Equal(t, "class", root.Attributes[1].Name)
	assert.Equal(t, "page-wrapper", root.Attributes[1].Value)

	require.Len(t, root.Children, 3)

	h1 := root.Children[0]
	assert.Equal(t, "h1", h1.TagName)
	require.NotNil(t, h1.DirText)
	assert.Equal(t, "state.Title", h1.DirText.RawExpression)

	ul := root.Children[1]
	assert.Equal(t, "ul", ul.TagName)
	require.Len(t, ul.Children, 1)
	li := ul.Children[0]
	require.NotNil(t, li.DirFor)
	assert.Equal(t, "item", li.DirFor.Arg)
	assert.Equal(t, "item in state.Items", li.DirFor.RawExpression)
	require.NotNil(t, li.DirKey)
	assert.Equal(t, "item.ID", li.DirKey.RawExpression)
	require.Len(t, li.RichText, 1)
	assert.Equal(t, "item.Name", li.RichText[0].RawExpression)

	button := root.Children[2]
	assert.Equal(t, "button", button.TagName)
	require.Contains(t, button.OnEvents, "click")
	assert.Equal(t, "handleSubmit", button.OnEvents["click"][0].RawExpression)
	require.Len(t, button.Children, 1)
	assert.Equal(t, "Submit", button.Children[0].TextContent)
}
