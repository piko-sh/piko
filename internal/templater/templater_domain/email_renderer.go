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
	"strings"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/templater/templater_dto"
)

// emailTemplateService implements the EmailTemplateService interface.
type emailTemplateService struct {
	// runner handles running templates to render email content.
	runner ManifestRunnerPort

	// renderer handles conversion of email templates to HTML and plain text.
	renderer RendererPort
}

var _ EmailTemplateService = (*emailTemplateService)(nil)

// Render processes an email template and returns the rendered output.
//
// Takes request (*http.Request) which provides the HTTP context for template
// rendering.
// Takes templatePath (string) which specifies the path to the email template.
// Takes props (any) which contains the data to pass to the template.
// Takes premailerOptions (*premailer.Options) which controls CSS inlining.
//
// Returns *templater_dto.RenderedEmailContent which contains the rendered HTML
// and plain text versions of the email.
// Returns error when the template cannot be found, parsed, or rendered.
func (s *emailTemplateService) Render(
	ctx context.Context,
	request *http.Request,
	templatePath string,
	props any,
	premailerOptions *premailer.Options,
	isPreviewMode bool,
) (*templater_dto.RenderedEmailContent, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "EmailTemplateService.Render")
	defer span.End()

	pageDef := templater_dto.PageDefinition{
		OriginalPath:   templatePath,
		NormalisedPath: "",
		TemplateHTML:   "",
	}

	templateAST, metadata, styling, err := s.runner.RunPartialWithProps(ctx, pageDef, request, props)
	if err != nil {
		l.ReportError(span, err, "Failed to run email template via manifest runner")
		return nil, fmt.Errorf("failed to run email template '%s': %w", templatePath, err)
	}
	if templateAST == nil {
		err := fmt.Errorf("manifest runner returned a nil AST for template '%s'", templatePath)
		l.ReportError(span, err, "Cannot render nil AST")
		return nil, err
	}

	plainTextContent, err := s.renderer.RenderASTToPlainText(ctx, templateAST)
	if err != nil {
		l.ReportError(span, err, "Failed to convert AST to plain text for email")
		plainTextContent = ""
	}

	var htmlBuilder strings.Builder
	err = s.renderer.RenderEmail(ctx, RenderEmailParams{
		Writer:           &htmlBuilder,
		Request:          request,
		PageID:           pageDef.OriginalPath,
		TemplateAST:      templateAST,
		Metadata:         &metadata,
		Styling:          styling,
		PremailerOptions: premailerOptions,
		IsPreviewMode:    isPreviewMode,
	})
	if err != nil {
		l.ReportError(span, err, "Failed to render email AST to HTML string")
		return nil, fmt.Errorf("failed to render email template AST for '%s': %w", templatePath, err)
	}
	htmlContent := htmlBuilder.String()

	assetRequests := s.renderer.GetLastEmailAssetRequests()
	l.Trace("Retrieved email asset requests", logger_domain.Int("requestCount", len(assetRequests)))

	return buildRenderedEmailContent(ctx, htmlContent, plainTextContent, styling, assetRequests), nil
}

// NewEmailTemplateService creates a new email template rendering service.
//
// Takes runner (ManifestRunnerPort) which loads templates from the manifest.
// Takes renderer (RendererPort) which converts AST to HTML.
//
// Returns EmailTemplateService which is configured and ready for use.
func NewEmailTemplateService(runner ManifestRunnerPort, renderer RendererPort) EmailTemplateService {
	return &emailTemplateService{runner: runner, renderer: renderer}
}

// buildRenderedEmailContent builds the final email content and logs debug
// information about the result.
//
// Takes ctx (context.Context) which carries the logger and tracing data.
// Takes htmlContent (string) which provides the HTML body of the email.
// Takes plainTextContent (string) which provides the plain text body.
// Takes styling (string) which provides CSS styling for the email.
// Takes assetRequests ([]*email_dto.EmailAssetRequest) which lists assets to
// attach.
//
// Returns *templater_dto.RenderedEmailContent which contains the assembled
// email content ready for sending.
func buildRenderedEmailContent(
	ctx context.Context,
	htmlContent string,
	plainTextContent string,
	styling string,
	assetRequests []*email_dto.EmailAssetRequest,
) *templater_dto.RenderedEmailContent {
	ctx, l := logger_domain.From(ctx, log)
	result := &templater_dto.RenderedEmailContent{
		HTML:               htmlContent,
		PlainText:          plainTextContent,
		CSS:                styling,
		AttachmentRequests: assetRequests,
	}

	l.Trace("Successfully rendered email template with HTML and plain text parts",
		logger_domain.Int("html_size_bytes", len(result.HTML)),
		logger_domain.Int("plaintext_size_bytes", len(result.PlainText)),
		logger_domain.Int("css_size_bytes", len(result.CSS)),
		logger_domain.Int("asset_request_count", len(result.AttachmentRequests)),
	)

	return result
}
