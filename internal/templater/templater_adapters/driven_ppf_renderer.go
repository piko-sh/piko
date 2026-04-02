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
	"context"
	"net/http"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// DrivenRenderer is a production renderer implementation that delegates all
// rendering operations to the core render service. It implements RendererPort.
type DrivenRenderer struct {
	// renderService handles rendering of pages, partials, and emails.
	renderService render_domain.RenderService
}

// RenderPage delegates page rendering to the render service.
//
// Takes params (RenderPageParams) which contains the page definition, template
// AST, metadata, and styling options for rendering.
//
// Returns error when the underlying render service fails.
func (r *DrivenRenderer) RenderPage(ctx context.Context, params templater_domain.RenderPageParams) error {
	return r.renderService.RenderAST(ctx, params.Writer, params.ResponseWriter, params.Request, render_domain.RenderASTOptions{
		PageID:     params.PageDefinition.OriginalPath,
		Template:   params.TemplateAST,
		Metadata:   params.Metadata,
		IsFragment: params.IsFragment,
		Styling:    params.Styling,
		SiteConfig: params.Config,
		ProbeData:  params.ProbeData,
	})
}

// RenderPartial delegates partial rendering to the render service.
//
// Takes params (templater_domain.RenderPageParams) which contains the page
// definition, template AST, metadata, and styling for the partial render.
//
// Returns error when the underlying render service fails to render the AST.
func (r *DrivenRenderer) RenderPartial(ctx context.Context, params templater_domain.RenderPageParams) error {
	return r.renderService.RenderAST(ctx, params.Writer, params.ResponseWriter, params.Request, render_domain.RenderASTOptions{
		PageID:     params.PageDefinition.OriginalPath,
		Template:   params.TemplateAST,
		Metadata:   params.Metadata,
		IsFragment: params.IsFragment,
		Styling:    params.Styling,
		SiteConfig: params.Config,
		ProbeData:  params.ProbeData,
	})
}

// RenderEmail delegates email rendering to the render service.
//
// Takes params (templater_domain.RenderEmailParams) which contains the email
// template and rendering configuration.
//
// Returns error when the render service fails to produce the email output.
func (r *DrivenRenderer) RenderEmail(ctx context.Context, params templater_domain.RenderEmailParams) error {
	return r.renderService.RenderEmail(ctx, params.Writer, params.Request, render_domain.RenderEmailOptions{
		PageID:           params.PageID,
		Template:         params.TemplateAST,
		Metadata:         params.Metadata,
		Styling:          params.Styling,
		PremailerOptions: params.PremailerOptions,
		IsPreviewMode:    params.IsPreviewMode,
	})
}

// CollectMetadata extracts link headers and metadata from the internal metadata.
//
// Takes request (*http.Request) which provides the current HTTP request context.
// Takes metadata (*templater_dto.InternalMetadata) which contains the internal
// metadata to extract from.
// Takes websiteConfig (*config.WebsiteConfig) which specifies the
// website configuration.
//
// Returns []render_dto.LinkHeader which contains the extracted link
// headers.
// Returns *ProbeData which contains component probe metadata, or nil
// when unavailable.
// Returns error when metadata extraction fails.
func (r *DrivenRenderer) CollectMetadata(
	ctx context.Context,
	request *http.Request,
	metadata *templater_dto.InternalMetadata,
	websiteConfig *config.WebsiteConfig,
) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
	return r.renderService.CollectMetadata(ctx, request, metadata, websiteConfig)
}

// RenderASTToPlainText converts an AST to plain text for email rendering.
//
// Takes templateAST (*ast_domain.TemplateAST) which contains the parsed
// template structure to render.
//
// Returns string which is the rendered plain text output.
// Returns error when rendering fails.
func (r *DrivenRenderer) RenderASTToPlainText(
	ctx context.Context,
	templateAST *ast_domain.TemplateAST,
) (string, error) {
	return r.renderService.RenderASTToPlainText(ctx, templateAST)
}

// GetLastEmailAssetRequests retrieves the last batch of email asset requests
// for debugging.
//
// Returns []*email_dto.EmailAssetRequest which contains the most recent batch
// of asset requests made during email rendering.
func (r *DrivenRenderer) GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest {
	return r.renderService.GetLastEmailAssetRequests()
}

// NewDrivenRenderer creates a new production renderer adapter.
//
// Takes service (render_domain.RenderService) which provides the rendering
// functionality.
//
// Returns templater_domain.RendererPort which is the configured renderer ready
// for use.
func NewDrivenRenderer(service render_domain.RenderService) templater_domain.RendererPort {
	return &DrivenRenderer{
		renderService: service,
	}
}
