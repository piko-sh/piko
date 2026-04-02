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

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/templater/templater_dto"
)

// MockTemplaterService is a test double for TemplaterService that returns
// zero values from nil function fields and tracks call counts atomically.
type MockTemplaterService struct {
	// ProbePageFunc is the function called by
	// ProbePage.
	ProbePageFunc func(ctx context.Context, page templater_dto.PageDefinition, request *http.Request, websiteConfig *config.WebsiteConfig) (*templater_dto.PageProbeResult, error)

	// RenderPageFunc is the function called by
	// RenderPage.
	RenderPageFunc func(ctx context.Context, request RenderRequest) error

	// ProbePartialFunc is the function called by
	// ProbePartial.
	ProbePartialFunc func(ctx context.Context, page templater_dto.PageDefinition, request *http.Request, websiteConfig *config.WebsiteConfig) (*templater_dto.PageProbeResult, error)

	// RenderPartialFunc is the function called by
	// RenderPartial.
	RenderPartialFunc func(ctx context.Context, request RenderRequest) error

	// SetRunnerFunc is the function called by
	// SetRunner.
	SetRunnerFunc func(r ManifestRunnerPort)

	// ProbePageCallCount tracks how many times
	// ProbePage was called.
	ProbePageCallCount int64

	// RenderPageCallCount tracks how many times
	// RenderPage was called.
	RenderPageCallCount int64

	// ProbePartialCallCount tracks how many times
	// ProbePartial was called.
	ProbePartialCallCount int64

	// RenderPartialCallCount tracks how many times
	// RenderPartial was called.
	RenderPartialCallCount int64

	// SetRunnerCallCount tracks how many times
	// SetRunner was called.
	SetRunnerCallCount int64
}

var _ TemplaterService = (*MockTemplaterService)(nil)

// ProbePage probes a page to gather metadata and validation information.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes page (templater_dto.PageDefinition) which describes the page to probe.
// Takes request (*http.Request) which is the incoming HTTP request.
// Takes websiteConfig (*config.WebsiteConfig) which provides the
// website configuration.
//
// Returns (*PageProbeResult, error), or (nil, nil) if ProbePageFunc is nil.
func (m *MockTemplaterService) ProbePage(
	ctx context.Context,
	page templater_dto.PageDefinition,
	request *http.Request,
	websiteConfig *config.WebsiteConfig,
) (*templater_dto.PageProbeResult, error) {
	atomic.AddInt64(&m.ProbePageCallCount, 1)
	if m.ProbePageFunc != nil {
		return m.ProbePageFunc(ctx, page, request, websiteConfig)
	}
	return nil, nil
}

// RenderPage renders a page template to the given writer.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes request (RenderRequest) which bundles all values needed for rendering.
//
// Returns error, or nil if RenderPageFunc is nil.
func (m *MockTemplaterService) RenderPage(ctx context.Context, request RenderRequest) error {
	atomic.AddInt64(&m.RenderPageCallCount, 1)
	if m.RenderPageFunc != nil {
		return m.RenderPageFunc(ctx, request)
	}
	return nil
}

// ProbePartial probes a page definition for partial template resolution.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes page (templater_dto.PageDefinition) which describes the partial to probe.
// Takes request (*http.Request) which is the incoming HTTP request.
// Takes websiteConfig (*config.WebsiteConfig) which provides the
// website configuration.
//
// Returns (*PageProbeResult, error), or (nil, nil) if ProbePartialFunc
// is nil.
func (m *MockTemplaterService) ProbePartial(
	ctx context.Context,
	page templater_dto.PageDefinition,
	request *http.Request,
	websiteConfig *config.WebsiteConfig,
) (*templater_dto.PageProbeResult, error) {
	atomic.AddInt64(&m.ProbePartialCallCount, 1)
	if m.ProbePartialFunc != nil {
		return m.ProbePartialFunc(ctx, page, request, websiteConfig)
	}
	return nil, nil
}

// RenderPartial renders a partial page template to the provided writer.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes request (RenderRequest) which bundles all values needed for rendering.
//
// Returns error, or nil if RenderPartialFunc is nil.
func (m *MockTemplaterService) RenderPartial(ctx context.Context, request RenderRequest) error {
	atomic.AddInt64(&m.RenderPartialCallCount, 1)
	if m.RenderPartialFunc != nil {
		return m.RenderPartialFunc(ctx, request)
	}
	return nil
}

// SetRunner assigns the manifest runner implementation.
//
// Takes r (ManifestRunnerPort) which is the runner to assign.
func (m *MockTemplaterService) SetRunner(r ManifestRunnerPort) {
	atomic.AddInt64(&m.SetRunnerCallCount, 1)
	if m.SetRunnerFunc != nil {
		m.SetRunnerFunc(r)
	}
}
