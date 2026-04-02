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

package templater_domain_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestNewTemplaterService(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)

	if fixture.Service == nil {
		t.Fatal("service should not be nil")
	}
}

func TestTemplaterService_SetRunner(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	newRunner := &templater_domain.MockManifestRunnerPort{}

	fixture.Service.SetRunner(newRunner)

	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := NewTestMetadata()

	var runPageCalled int64
	newRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		atomic.AddInt64(&runPageCalled, 1)
		return ast, metadata, "", nil
	}

	mockRenderer := &templater_domain.MockRendererPort{}

	service := templater_domain.NewTemplaterService(newRunner, mockRenderer, fixture.MockI18N)

	mockRenderer.RenderPageFunc = func(_ context.Context, _ templater_domain.RenderPageParams) error {
		return nil
	}

	var buffer bytes.Buffer
	err := service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}
	if atomic.LoadInt64(&runPageCalled) == 0 {
		t.Fatal("expected RunPage to be called on the new runner")
	}
}

func TestTemplaterService_RenderPage_Success(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := NewTestMetadata()

	fixture.MockRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "body { color: red; }", nil
	}

	fixture.MockRenderer.RenderPageFunc = func(_ context.Context, params templater_domain.RenderPageParams) error {
		if params.TemplateAST != ast {
			t.Error("expected TemplateAST to match")
		}
		if params.Styling != "body { color: red; }" {
			t.Errorf("expected styling %q, got %q", "body { color: red; }", params.Styling)
		}
		if params.IsFragment {
			t.Error("expected IsFragment to be false")
		}
		if params.PageDefinition.OriginalPath != "pages/home.pk" {
			t.Errorf("expected OriginalPath %q, got %q", "pages/home.pk", params.PageDefinition.OriginalPath)
		}
		return nil
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}
	if atomic.LoadInt64(&fixture.MockRunner.RunPageCallCount) == 0 {
		t.Fatal("expected RunPage to be called")
	}
	if atomic.LoadInt64(&fixture.MockRenderer.RenderPageCallCount) == 0 {
		t.Fatal("expected RenderPage to be called")
	}
}

func TestTemplaterService_RenderPage_RunnerError(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	fixture.MockRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return nil, templater_dto.InternalMetadata{}, "", errors.New("runner failure")
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	AssertError(t, err)
	AssertContains(t, err.Error(), "runner failure")
	AssertContains(t, err.Error(), "pages/home.pk")
}

func TestTemplaterService_RenderPage_ServerRedirect(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			ServerRedirect: "/new-location",
		},
	}

	fixture.MockRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "", nil
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	AssertError(t, err)
	redirectErr, ok := errors.AsType[*templater_dto.RedirectRequired](err)
	if !ok {
		t.Fatal("error should be RedirectRequired")
	}
	if redirectErr.Metadata.ServerRedirect != "/new-location" {
		t.Errorf("expected ServerRedirect %q, got %q", "/new-location", redirectErr.Metadata.ServerRedirect)
	}
}

func TestTemplaterService_RenderPage_ClientRedirect(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			ClientRedirect: "https://example.com/login",
		},
	}

	fixture.MockRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "", nil
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	AssertError(t, err)
	redirectErr, ok := errors.AsType[*templater_dto.RedirectRequired](err)
	if !ok {
		t.Fatal("error should be RedirectRequired")
	}
	if redirectErr.Metadata.ClientRedirect != "https://example.com/login" {
		t.Errorf("expected ClientRedirect %q, got %q", "https://example.com/login", redirectErr.Metadata.ClientRedirect)
	}
}

func TestTemplaterService_RenderPage_Fragment(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := NewTestMetadata()

	fixture.MockRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "", nil
	}

	fixture.MockRenderer.RenderPageFunc = func(_ context.Context, params templater_domain.RenderPageParams) error {
		if !params.IsFragment {
			t.Error("expected IsFragment to be true")
		}
		return nil
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    true,
		WebsiteConfig: websiteConfig,
	})

	AssertNoError(t, err)
}

func TestTemplaterService_RenderPage_RendererError(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := NewTestMetadata()

	fixture.MockRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "", nil
	}

	fixture.MockRenderer.RenderPageFunc = func(_ context.Context, _ templater_domain.RenderPageParams) error {
		return errors.New("renderer error")
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	AssertError(t, err)
	AssertContains(t, err.Error(), "renderer error")
}

func TestTemplaterService_RenderPartial_Success(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	partial := NewTestPartialDefinition("partials/header.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := NewTestMetadata()

	fixture.MockRunner.RunPartialFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "h1 { font-size: 2rem; }", nil
	}

	fixture.MockRenderer.RenderPartialFunc = func(_ context.Context, params templater_domain.RenderPageParams) error {
		if params.TemplateAST != ast {
			t.Error("expected TemplateAST to match")
		}
		if params.Styling != "h1 { font-size: 2rem; }" {
			t.Errorf("expected styling %q, got %q", "h1 { font-size: 2rem; }", params.Styling)
		}
		if params.PageDefinition.OriginalPath != "partials/header.pk" {
			t.Errorf("expected OriginalPath %q, got %q", "partials/header.pk", params.PageDefinition.OriginalPath)
		}
		return nil
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPartial(ctx, templater_domain.RenderRequest{
		Page:          partial,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	AssertNoError(t, err)
	if atomic.LoadInt64(&fixture.MockRunner.RunPartialCallCount) == 0 {
		t.Fatal("expected RunPartial to be called")
	}
	if atomic.LoadInt64(&fixture.MockRenderer.RenderPartialCallCount) == 0 {
		t.Fatal("expected RenderPartial to be called")
	}
}

func TestTemplaterService_RenderPartial_RunnerError(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	partial := NewTestPartialDefinition("partials/header.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	fixture.MockRunner.RunPartialFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return nil, templater_dto.InternalMetadata{}, "", errors.New("partial runner failure")
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPartial(ctx, templater_domain.RenderRequest{
		Page:          partial,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	AssertError(t, err)
	AssertContains(t, err.Error(), "partial runner failure")
}

func TestTemplaterService_RenderPartial_Redirect(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	partial := NewTestPartialDefinition("partials/header.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			ClientRedirect: "/redirected",
		},
	}

	fixture.MockRunner.RunPartialFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "", nil
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPartial(ctx, templater_domain.RenderRequest{
		Page:          partial,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	AssertError(t, err)
	if _, ok := errors.AsType[*templater_dto.RedirectRequired](err); !ok {
		t.Fatal("error should be RedirectRequired")
	}
}

func TestTemplaterService_ProbePage_Success(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	websiteConfig := NewTestConfig()

	staticMeta := &templater_dto.InternalMetadata{
		AssetRefs: []templater_dto.AssetRef{
			{Kind: "css", Path: "/assets/style.css"},
		},
	}
	mockEntry := &templater_domain.MockPageEntryView{
		GetStaticMetadataFunc: func() *templater_dto.InternalMetadata {
			return staticMeta
		},
	}

	fixture.MockRunner.GetPageEntryFunc = func(_ context.Context, manifestKey string) (templater_domain.PageEntryView, error) {
		if manifestKey != "pages/home.pk" {
			t.Errorf("expected manifestKey %q, got %q", "pages/home.pk", manifestKey)
		}
		return mockEntry, nil
	}

	linkHeaders := []render_dto.LinkHeader{
		{URL: "/assets/style.css", Rel: "preload", As: "style"},
	}
	fixture.MockRenderer.CollectMetadataFunc = func(
		_ context.Context,
		_ *http.Request,
		_ *templater_dto.InternalMetadata,
		_ *config.WebsiteConfig,
	) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
		return linkHeaders, nil, nil
	}

	result, err := fixture.Service.ProbePage(ctx, page, request, websiteConfig)

	AssertNoError(t, err)
	if result == nil {
		t.Fatal("expected result to not be nil")
	}
	if len(result.LinkHeaders) != 1 {
		t.Fatalf("expected 1 link header, got %d", len(result.LinkHeaders))
	}
	if result.LinkHeaders[0].URL != "/assets/style.css" {
		t.Errorf("expected URL %q, got %q", "/assets/style.css", result.LinkHeaders[0].URL)
	}
}

func TestTemplaterService_ProbePage_NotFound(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/nonexistent.pk")
	request := NewTestRequest("GET", "/")
	websiteConfig := NewTestConfig()

	fixture.MockRunner.GetPageEntryFunc = func(_ context.Context, _ string) (templater_domain.PageEntryView, error) {
		return nil, errors.New("not found")
	}

	result, err := fixture.Service.ProbePage(ctx, page, request, websiteConfig)

	AssertError(t, err)
	if result != nil {
		t.Fatal("expected result to be nil")
	}
	AssertContains(t, err.Error(), "not found in manifest")
}

func TestTemplaterService_ProbePage_MetadataCollectionError(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	websiteConfig := NewTestConfig()

	staticMeta := &templater_dto.InternalMetadata{}
	mockEntry := &templater_domain.MockPageEntryView{
		GetStaticMetadataFunc: func() *templater_dto.InternalMetadata {
			return staticMeta
		},
	}

	fixture.MockRunner.GetPageEntryFunc = func(_ context.Context, _ string) (templater_domain.PageEntryView, error) {
		return mockEntry, nil
	}

	fixture.MockRenderer.CollectMetadataFunc = func(
		_ context.Context,
		_ *http.Request,
		_ *templater_dto.InternalMetadata,
		_ *config.WebsiteConfig,
	) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
		return nil, nil, errors.New("collection error")
	}

	result, err := fixture.Service.ProbePage(ctx, page, request, websiteConfig)

	AssertNoError(t, err)
	if result == nil {
		t.Fatal("expected result to not be nil")
	}
}

func TestTemplaterService_ProbePartial_Success(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	partial := NewTestPartialDefinition("partials/header.pk")
	request := NewTestRequest("GET", "/")
	websiteConfig := NewTestConfig()

	staticMeta := &templater_dto.InternalMetadata{}
	mockEntry := &templater_domain.MockPageEntryView{
		GetStaticMetadataFunc: func() *templater_dto.InternalMetadata {
			return staticMeta
		},
	}

	fixture.MockRunner.GetPageEntryFunc = func(_ context.Context, _ string) (templater_domain.PageEntryView, error) {
		return mockEntry, nil
	}

	fixture.MockRenderer.CollectMetadataFunc = func(
		_ context.Context,
		_ *http.Request,
		_ *templater_dto.InternalMetadata,
		_ *config.WebsiteConfig,
	) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
		return nil, nil, nil
	}

	result, err := fixture.Service.ProbePartial(ctx, partial, request, websiteConfig)

	AssertNoError(t, err)
	if result == nil {
		t.Fatal("expected result to not be nil")
	}
}

func TestTemplaterService_ProbePartial_NotFound(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	partial := NewTestPartialDefinition("partials/missing.pk")
	request := NewTestRequest("GET", "/")
	websiteConfig := NewTestConfig()

	fixture.MockRunner.GetPageEntryFunc = func(_ context.Context, _ string) (templater_domain.PageEntryView, error) {
		return nil, errors.New("not found")
	}

	result, err := fixture.Service.ProbePartial(ctx, partial, request, websiteConfig)

	AssertError(t, err)
	if result != nil {
		t.Fatal("expected result to be nil")
	}
	AssertContains(t, err.Error(), "Partial not found in manifest")
}

func TestHasRedirect_NoRedirect(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := NewTestRequest("GET", "/")
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := templater_dto.InternalMetadata{}

	fixture.MockRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "", nil
	}

	fixture.MockRenderer.RenderPageFunc = func(_ context.Context, _ templater_domain.RenderPageParams) error {
		return nil
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})

	AssertNoError(t, err)
	if atomic.LoadInt64(&fixture.MockRenderer.RenderPageCallCount) == 0 {
		t.Fatal("expected RenderPage to be called")
	}
}

func TestTemplaterService_RenderPage_PassesAllParamsToRenderer(t *testing.T) {
	t.Parallel()

	fixture := NewTestFixture(t)
	ctx := context.Background()
	page := NewTestPageDefinition("pages/home.pk")
	request := httptest.NewRequest("GET", "/home", nil)
	response := NewTestResponseWriter()
	websiteConfig := NewTestConfig()

	ast := NewTestAST()
	metadata := NewTestMetadata()
	styling := "h1 { color: blue; }"

	fixture.MockRunner.RunPageFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, styling, nil
	}

	var capturedParams templater_domain.RenderPageParams
	fixture.MockRenderer.RenderPageFunc = func(_ context.Context, params templater_domain.RenderPageParams) error {
		capturedParams = params
		return nil
	}

	var buffer bytes.Buffer
	err := fixture.Service.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        &buffer,
		Response:      response,
		Request:       request,
		IsFragment:    true,
		WebsiteConfig: websiteConfig,
	})

	AssertNoError(t, err)
	if capturedParams.Writer != &buffer {
		t.Error("expected Writer to match")
	}
	if capturedParams.ResponseWriter != response {
		t.Error("expected ResponseWriter to match")
	}
	if capturedParams.Request != request {
		t.Error("expected Request to match")
	}
	if capturedParams.TemplateAST != ast {
		t.Error("expected TemplateAST to match")
	}
	if capturedParams.Styling != styling {
		t.Errorf("expected Styling %q, got %q", styling, capturedParams.Styling)
	}
	if !capturedParams.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if capturedParams.Config != websiteConfig {
		t.Error("expected Config to match")
	}
	if capturedParams.PageDefinition != page {
		t.Error("expected PageDefinition to match")
	}
}
