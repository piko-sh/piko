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
	"piko.sh/piko/internal/templater/templater_dto"
)

// MockManifestRunnerPort is a test double for ManifestRunnerPort that
// returns zero values from nil function fields and tracks call counts
// atomically.
type MockManifestRunnerPort struct {
	// RunPageFunc is the function called by RunPage.
	RunPageFunc func(ctx context.Context, pageDefinition templater_dto.PageDefinition, request *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)

	// RunPartialFunc is the function called by
	// RunPartial.
	RunPartialFunc func(ctx context.Context, pageDefinition templater_dto.PageDefinition, request *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)

	// RunPartialWithPropsFunc is the function called by
	// RunPartialWithProps.
	RunPartialWithPropsFunc func(
		ctx context.Context, pageDefinition templater_dto.PageDefinition,
		request *http.Request, props any,
	) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)

	// GetPageEntryFunc is the function called by
	// GetPageEntry.
	GetPageEntryFunc func(ctx context.Context, manifestKey string) (PageEntryView, error)

	// RunPageCallCount tracks how many times RunPage
	// was called.
	RunPageCallCount int64

	// RunPartialCallCount tracks how many times
	// RunPartial was called.
	RunPartialCallCount int64

	// RunPartialWithPropsCallCount tracks how many
	// times RunPartialWithProps was called.
	RunPartialWithPropsCallCount int64

	// GetPageEntryCallCount tracks how many times
	// GetPageEntry was called.
	GetPageEntryCallCount int64
}

var _ ManifestRunnerPort = (*MockManifestRunnerPort)(nil)

// RunPage renders a page from the given definition and request.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes pageDefinition (templater_dto.PageDefinition)
// which describes the page to render.
// Takes request (*http.Request) which is the incoming HTTP request.
//
// Returns (*TemplateAST, InternalMetadata, string, error), or zero values if
// RunPageFunc is nil.
func (m *MockManifestRunnerPort) RunPage(
	ctx context.Context,
	pageDefinition templater_dto.PageDefinition,
	request *http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	atomic.AddInt64(&m.RunPageCallCount, 1)
	if m.RunPageFunc != nil {
		return m.RunPageFunc(ctx, pageDefinition, request)
	}
	return nil, templater_dto.InternalMetadata{}, "", nil
}

// RunPartial renders a partial template from the given page definition.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes pageDefinition (templater_dto.PageDefinition)
// which describes the partial to render.
// Takes request (*http.Request) which is the incoming
// HTTP request.
//
// Returns (*TemplateAST, InternalMetadata, string,
// error), or zero values if RunPartialFunc is nil.
func (m *MockManifestRunnerPort) RunPartial(
	ctx context.Context,
	pageDefinition templater_dto.PageDefinition,
	request *http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	atomic.AddInt64(&m.RunPartialCallCount, 1)
	if m.RunPartialFunc != nil {
		return m.RunPartialFunc(ctx, pageDefinition, request)
	}
	return nil, templater_dto.InternalMetadata{}, "", nil
}

// RunPartialWithProps runs a partial and passes props data through to the
// compiled template.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes pageDefinition (templater_dto.PageDefinition)
// which describes the partial to render.
// Takes request (*http.Request) which is the incoming
// HTTP request.
// Takes props (any) which contains the properties to
// pass to the template.
//
// Returns (*TemplateAST, InternalMetadata, string, error), or zero values if
// RunPartialWithPropsFunc is nil.
func (m *MockManifestRunnerPort) RunPartialWithProps(
	ctx context.Context,
	pageDefinition templater_dto.PageDefinition,
	request *http.Request,
	props any,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	atomic.AddInt64(&m.RunPartialWithPropsCallCount, 1)
	if m.RunPartialWithPropsFunc != nil {
		return m.RunPartialWithPropsFunc(ctx, pageDefinition, request, props)
	}
	return nil, templater_dto.InternalMetadata{}, "", nil
}

// GetPageEntry retrieves a page entry view by its manifest key.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes manifestKey (string) which identifies the page entry to look up.
//
// Returns (PageEntryView, error), or (nil, nil) if GetPageEntryFunc is nil.
func (m *MockManifestRunnerPort) GetPageEntry(
	ctx context.Context,
	manifestKey string,
) (PageEntryView, error) {
	atomic.AddInt64(&m.GetPageEntryCallCount, 1)
	if m.GetPageEntryFunc != nil {
		return m.GetPageEntryFunc(ctx, manifestKey)
	}
	return nil, nil
}
