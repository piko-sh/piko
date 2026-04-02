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

package pikotest_domain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"piko.sh/piko/internal/pikotest/pikotest_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// ComponentTester orchestrates the testing of a compiled Piko component.
// It handles calling the BuildAST function, managing the render service for
// HTML output, and providing a clean API for assertions.
type ComponentTester struct {
	// tb is the testing context for reporting errors and marking test helpers.
	tb testing.TB

	// renderer is the service that executes templates during tests.
	renderer render_domain.RenderService

	// buildAST builds the AST from request data and props for rendering.
	buildAST pikotest_dto.BuildASTFunc

	// httpServer is the test HTTP server for handling requests.
	httpServer *httptest.Server

	// pageID is the unique identifier for the page passed to the renderer.
	pageID string
}

// NewComponentTester creates a new ComponentTester for the given component's
// BuildAST function.
//
// Takes tb (testing.TB) which provides the test context for assertions.
// Takes buildAST (BuildASTFunc) which builds the component's AST for testing.
// Takes opts (...ComponentOption) which configures the tester behaviour.
//
// Returns *ComponentTester which is ready to render and test components.
func NewComponentTester(tb testing.TB, buildAST pikotest_dto.BuildASTFunc, opts ...pikotest_dto.ComponentOption) *ComponentTester {
	config := pikotest_dto.DefaultComponentConfig()

	for _, opt := range opts {
		opt(&config)
	}

	return &ComponentTester{
		tb:         tb,
		renderer:   config.Renderer,
		buildAST:   buildAST,
		httpServer: nil,
		pageID:     config.PageID,
	}
}

// Render runs the component with the given request and props.
//
// This is the main entry point for testing a component.
//
// Takes request (*templater_dto.RequestData) which contains the request data for
// the component.
// Takes props (any) which provides the component properties.
//
// Returns *TestView which provides assertion methods for checking the rendered
// output.
func (ct *ComponentTester) Render(request *templater_dto.RequestData, props any) *TestView {
	ct.tb.Helper()

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Milliseconds()
		TestRenderDuration.Record(context.Background(), float64(duration))
		TestRenderCount.Add(context.Background(), 1)
	}()

	ast, metadata, diagnostics := ct.buildAST(request, props)

	if len(diagnostics) > 0 {
		ct.tb.Helper()
		for _, diagnostic := range diagnostics {
			ct.tb.Errorf("Component diagnostic [%s]: %s (at %s:%d:%d) - %s",
				diagnostic.Severity.String(), diagnostic.Message, diagnostic.SourcePath, diagnostic.Line, diagnostic.Column, diagnostic.Expression)
		}

		if ast == nil {
			ct.tb.Fatal("Component render failed with fatal diagnostics")
		}
	}

	httpReq := ct.makeHTTPRequest(request)

	return newTestView(
		ct.tb,
		nil,
		&metadata,
		ast,
		ct.renderer,
		httpReq,
		ct.pageID,
	)
}

// Benchmark runs a benchmark of the component render operation.
// This measures the time to execute BuildAST, excluding initial setup.
//
// Takes request (*templater_dto.RequestData) which provides the request context.
// Takes props (any) which supplies the component properties.
func (ct *ComponentTester) Benchmark(request *templater_dto.RequestData, props any) {
	b, ok := ct.tb.(*testing.B)
	if !ok {
		ct.tb.Fatal("Benchmark can only be called from *testing.B")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = ct.buildAST(request, props)
	}
}

// makeHTTPRequest creates an http.Request from RequestData for use with
// RenderService.
//
// The RenderService expects an *http.Request, so this builds one from the
// test RequestData.
//
// Takes reqData (*templater_dto.RequestData) which provides the request
// details to convert.
//
// Returns *http.Request which is the built request ready for rendering.
func (*ComponentTester) makeHTTPRequest(reqData *templater_dto.RequestData) *http.Request {
	method := reqData.Method()
	if method == "" {
		method = "GET"
	}

	path := "/"
	if reqURL := reqData.URL(); reqURL != nil {
		path = reqURL.String()
	}

	httpReq := httptest.NewRequest(method, path, nil)
	httpReq = httpReq.WithContext(reqData.Context())

	if reqData.Host() != "" {
		httpReq.Host = reqData.Host()
	}

	return httpReq
}
