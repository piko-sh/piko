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
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestNewEmailTemplateService(t *testing.T) {
	t.Parallel()

	runner := &templater_domain.MockManifestRunnerPort{}
	renderer := &templater_domain.MockRendererPort{}

	service := templater_domain.NewEmailTemplateService(runner, renderer)

	if service == nil {
		t.Fatal("service should not be nil")
	}
}

func TestEmailTemplateService_Render_Success(t *testing.T) {
	t.Parallel()

	runner := &templater_domain.MockManifestRunnerPort{}
	renderer := &templater_domain.MockRendererPort{}
	service := templater_domain.NewEmailTemplateService(runner, renderer)

	ctx := context.Background()
	request := NewTestRequest("GET", "/")
	templatePath := "emails/welcome.pk"
	props := map[string]string{"name": "Alice"}
	premailerOpts := &premailer.Options{}

	ast := NewTestAST()
	metadata := NewTestMetadata()
	styling := "body { font-family: Arial; }"

	runner.RunPartialWithPropsFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
		_ any,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, styling, nil
	}

	renderer.RenderASTToPlainTextFunc = func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
		return "Welcome, Alice!", nil
	}

	renderer.RenderEmailFunc = func(_ context.Context, params templater_domain.RenderEmailParams) error {
		if params.PageID != templatePath {
			t.Errorf("expected PageID %q, got %q", templatePath, params.PageID)
		}
		if params.TemplateAST != ast {
			t.Error("expected TemplateAST to match")
		}
		if params.Styling != styling {
			t.Errorf("expected Styling %q, got %q", styling, params.Styling)
		}
		_, _ = io.WriteString(params.Writer, "<html><body>Welcome, Alice!</body></html>")
		return nil
	}

	assetRequests := []*email_dto.EmailAssetRequest{
		{SourcePath: "assets/logo.png"},
	}
	renderer.GetLastEmailAssetRequestsFunc = func() []*email_dto.EmailAssetRequest {
		return assetRequests
	}

	result, err := service.Render(ctx, request, templatePath, props, premailerOpts, false)

	AssertNoError(t, err)
	if result == nil {
		t.Fatal("expected result to not be nil")
	}
	if result.HTML != "<html><body>Welcome, Alice!</body></html>" {
		t.Errorf("expected HTML %q, got %q", "<html><body>Welcome, Alice!</body></html>", result.HTML)
	}
	if result.PlainText != "Welcome, Alice!" {
		t.Errorf("expected PlainText %q, got %q", "Welcome, Alice!", result.PlainText)
	}
	if result.CSS != styling {
		t.Errorf("expected CSS %q, got %q", styling, result.CSS)
	}
	if len(result.AttachmentRequests) != 1 {
		t.Fatalf("expected 1 attachment request, got %d", len(result.AttachmentRequests))
	}
	if result.AttachmentRequests[0].SourcePath != "assets/logo.png" {
		t.Errorf("expected SourcePath %q, got %q", "assets/logo.png", result.AttachmentRequests[0].SourcePath)
	}
}

func TestEmailTemplateService_Render_RunnerError(t *testing.T) {
	t.Parallel()

	runner := &templater_domain.MockManifestRunnerPort{}
	renderer := &templater_domain.MockRendererPort{}
	service := templater_domain.NewEmailTemplateService(runner, renderer)

	ctx := context.Background()
	request := NewTestRequest("GET", "/")
	templatePath := "emails/broken.pk"

	runner.RunPartialWithPropsFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
		_ any,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return nil, templater_dto.InternalMetadata{}, "", errors.New("template not found")
	}

	result, err := service.Render(ctx, request, templatePath, nil, nil, false)

	AssertError(t, err)
	if result != nil {
		t.Fatal("expected result to be nil")
	}
	AssertContains(t, err.Error(), "failed to run email template")
	AssertContains(t, err.Error(), templatePath)
}

func TestEmailTemplateService_Render_NilAST(t *testing.T) {
	t.Parallel()

	runner := &templater_domain.MockManifestRunnerPort{}
	renderer := &templater_domain.MockRendererPort{}
	service := templater_domain.NewEmailTemplateService(runner, renderer)

	ctx := context.Background()
	request := NewTestRequest("GET", "/")
	templatePath := "emails/empty.pk"

	runner.RunPartialWithPropsFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
		_ any,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return nil, templater_dto.InternalMetadata{}, "", nil
	}

	result, err := service.Render(ctx, request, templatePath, nil, nil, false)

	AssertError(t, err)
	if result != nil {
		t.Fatal("expected result to be nil")
	}
	AssertContains(t, err.Error(), "nil AST")
}

func TestEmailTemplateService_Render_PlainTextError(t *testing.T) {
	t.Parallel()

	runner := &templater_domain.MockManifestRunnerPort{}
	renderer := &templater_domain.MockRendererPort{}
	service := templater_domain.NewEmailTemplateService(runner, renderer)

	ctx := context.Background()
	request := NewTestRequest("GET", "/")
	templatePath := "emails/welcome.pk"

	ast := NewTestAST()
	metadata := NewTestMetadata()

	runner.RunPartialWithPropsFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
		_ any,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "", nil
	}

	renderer.RenderASTToPlainTextFunc = func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
		return "", errors.New("plain text conversion failed")
	}

	renderer.RenderEmailFunc = func(_ context.Context, params templater_domain.RenderEmailParams) error {
		_, _ = io.WriteString(params.Writer, "<html>content</html>")
		return nil
	}

	renderer.GetLastEmailAssetRequestsFunc = func() []*email_dto.EmailAssetRequest {
		return nil
	}

	result, err := service.Render(ctx, request, templatePath, nil, nil, false)

	AssertNoError(t, err)
	if result == nil {
		t.Fatal("expected result to not be nil")
	}
	if result.HTML != "<html>content</html>" {
		t.Errorf("expected HTML %q, got %q", "<html>content</html>", result.HTML)
	}
	if result.PlainText != "" {
		t.Errorf("expected empty PlainText, got %q", result.PlainText)
	}
}

func TestEmailTemplateService_Render_HTMLRenderError(t *testing.T) {
	t.Parallel()

	runner := &templater_domain.MockManifestRunnerPort{}
	renderer := &templater_domain.MockRendererPort{}
	service := templater_domain.NewEmailTemplateService(runner, renderer)

	ctx := context.Background()
	request := NewTestRequest("GET", "/")
	templatePath := "emails/broken-render.pk"

	ast := NewTestAST()
	metadata := NewTestMetadata()

	runner.RunPartialWithPropsFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
		_ any,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "css", nil
	}

	renderer.RenderASTToPlainTextFunc = func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
		return "plain text", nil
	}

	renderer.RenderEmailFunc = func(_ context.Context, _ templater_domain.RenderEmailParams) error {
		return errors.New("HTML render failed")
	}

	result, err := service.Render(ctx, request, templatePath, nil, nil, false)

	AssertError(t, err)
	if result != nil {
		t.Fatal("expected result to be nil")
	}
	AssertContains(t, err.Error(), "failed to render email template AST")
}

func TestEmailTemplateService_Render_NoAssets(t *testing.T) {
	t.Parallel()

	runner := &templater_domain.MockManifestRunnerPort{}
	renderer := &templater_domain.MockRendererPort{}
	service := templater_domain.NewEmailTemplateService(runner, renderer)

	ctx := context.Background()
	request := NewTestRequest("GET", "/")
	templatePath := "emails/simple.pk"

	ast := NewTestAST()
	metadata := NewTestMetadata()

	runner.RunPartialWithPropsFunc = func(
		_ context.Context,
		_ templater_dto.PageDefinition,
		_ *http.Request,
		_ any,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
		return ast, metadata, "", nil
	}

	renderer.RenderASTToPlainTextFunc = func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
		return "simple text", nil
	}

	renderer.RenderEmailFunc = func(_ context.Context, params templater_domain.RenderEmailParams) error {
		_, _ = io.WriteString(params.Writer, "<html>simple</html>")
		return nil
	}

	renderer.GetLastEmailAssetRequestsFunc = func() []*email_dto.EmailAssetRequest {
		return nil
	}

	result, err := service.Render(ctx, request, templatePath, nil, nil, false)

	AssertNoError(t, err)
	if result == nil {
		t.Fatal("expected result to not be nil")
	}
	if result.AttachmentRequests != nil {
		t.Error("expected AttachmentRequests to be nil")
	}
	if result.CSS != "" {
		t.Errorf("expected empty CSS, got %q", result.CSS)
	}
}
