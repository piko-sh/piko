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
	"net/http"
	"sync/atomic"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

// MockRendererPort is a test double for RendererPort that returns zero
// values from nil function fields and tracks call counts atomically.
type MockRendererPort struct {
	// CollectMetadataFunc is the function called by
	// CollectMetadata.
	CollectMetadataFunc func(
		ctx context.Context, request *http.Request,
		metadata *templater_dto.InternalMetadata,
		websiteConfig *config.WebsiteConfig,
	) ([]render_dto.LinkHeader, *render_dto.ProbeData, error)

	// RenderPageFunc is the function called by
	// RenderPage.
	RenderPageFunc func(ctx context.Context, params RenderPageParams) error

	// RenderPartialFunc is the function called by
	// RenderPartial.
	RenderPartialFunc func(ctx context.Context, params RenderPageParams) error

	// RenderEmailFunc is the function called by
	// RenderEmail.
	RenderEmailFunc func(ctx context.Context, params RenderEmailParams) error

	// RenderASTToPlainTextFunc is the function called
	// by RenderASTToPlainText.
	RenderASTToPlainTextFunc func(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error)

	// GetLastEmailAssetRequestsFunc is the function
	// called by GetLastEmailAssetRequests.
	GetLastEmailAssetRequestsFunc func() []*email_dto.EmailAssetRequest

	// CollectMetadataCallCount tracks how many times
	// CollectMetadata was called.
	CollectMetadataCallCount int64

	// RenderPageCallCount tracks how many times
	// RenderPage was called.
	RenderPageCallCount int64

	// RenderPartialCallCount tracks how many times
	// RenderPartial was called.
	RenderPartialCallCount int64

	// RenderEmailCallCount tracks how many times
	// RenderEmail was called.
	RenderEmailCallCount int64

	// RenderASTToPlainTextCallCount tracks how many
	// times RenderASTToPlainText was called.
	RenderASTToPlainTextCallCount int64

	// GetLastEmailAssetRequestsCallCount tracks how
	// many times GetLastEmailAssetRequests was called.
	GetLastEmailAssetRequestsCallCount int64
}

var _ RendererPort = (*MockRendererPort)(nil)

// CollectMetadata gathers metadata from the request and configuration.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes request (*http.Request) which is the incoming HTTP request.
// Takes metadata (*templater_dto.InternalMetadata) which holds the page metadata.
// Takes websiteConfig (*config.WebsiteConfig) which provides the
// website configuration.
//
// Returns ([]LinkHeader, error), or (nil, nil) if CollectMetadataFunc is nil.
func (m *MockRendererPort) CollectMetadata(
	ctx context.Context,
	request *http.Request,
	metadata *templater_dto.InternalMetadata,
	websiteConfig *config.WebsiteConfig,
) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
	atomic.AddInt64(&m.CollectMetadataCallCount, 1)
	if m.CollectMetadataFunc != nil {
		return m.CollectMetadataFunc(ctx, request, metadata, websiteConfig)
	}
	return nil, nil, nil
}

// RenderPage renders a page using the provided parameters.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes params (RenderPageParams) which contains the page rendering parameters.
//
// Returns error, or nil if RenderPageFunc is nil.
func (m *MockRendererPort) RenderPage(ctx context.Context, params RenderPageParams) error {
	atomic.AddInt64(&m.RenderPageCallCount, 1)
	if m.RenderPageFunc != nil {
		return m.RenderPageFunc(ctx, params)
	}
	return nil
}

// RenderPartial renders a partial page using the given parameters.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes params (RenderPageParams) which contains the partial rendering parameters.
//
// Returns error, or nil if RenderPartialFunc is nil.
func (m *MockRendererPort) RenderPartial(ctx context.Context, params RenderPageParams) error {
	atomic.AddInt64(&m.RenderPartialCallCount, 1)
	if m.RenderPartialFunc != nil {
		return m.RenderPartialFunc(ctx, params)
	}
	return nil
}

// RenderEmail creates an email from the given template and data.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes params (RenderEmailParams) which contains the email rendering parameters.
//
// Returns error, or nil if RenderEmailFunc is nil.
func (m *MockRendererPort) RenderEmail(ctx context.Context, params RenderEmailParams) error {
	atomic.AddInt64(&m.RenderEmailCallCount, 1)
	if m.RenderEmailFunc != nil {
		return m.RenderEmailFunc(ctx, params)
	}
	return nil
}

// RenderASTToPlainText converts a template AST into plain text.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes templateAST (*ast_domain.TemplateAST) which is the AST to convert.
//
// Returns (string, error), or ("", nil) if RenderASTToPlainTextFunc is nil.
func (m *MockRendererPort) RenderASTToPlainText(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error) {
	atomic.AddInt64(&m.RenderASTToPlainTextCallCount, 1)
	if m.RenderASTToPlainTextFunc != nil {
		return m.RenderASTToPlainTextFunc(ctx, templateAST)
	}
	return "", nil
}

// GetLastEmailAssetRequests returns the most recent email asset requests.
//
// Returns []*EmailAssetRequest, or nil if GetLastEmailAssetRequestsFunc
// is nil.
func (m *MockRendererPort) GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest {
	atomic.AddInt64(&m.GetLastEmailAssetRequestsCallCount, 1)
	if m.GetLastEmailAssetRequestsFunc != nil {
		return m.GetLastEmailAssetRequestsFunc()
	}
	return nil
}
