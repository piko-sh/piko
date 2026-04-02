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

package render_domain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestRenderAST(t *testing.T) {
	testCases := []struct {
		buildRO        func() *RenderOrchestrator
		name           string
		opts           RenderASTOptions
		wantContains   []string
		wantNotContain []string
		withReq        bool
		wantErr        bool
	}{
		{
			name: "nil template renders without error for fragment",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: RenderASTOptions{
				Template:   nil,
				Metadata:   &templater_dto.InternalMetadata{},
				PageID:     "nil-frag",
				IsFragment: true,
			},
			withReq: true,
		},
		{
			name: "nil template renders without error for full page",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: RenderASTOptions{
				Template:   nil,
				Metadata:   &templater_dto.InternalMetadata{},
				PageID:     "nil-full",
				IsFragment: false,
			},
			withReq: true,
		},
		{
			name: "fragment mode renders content without DOCTYPE",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "div",
							Children: []*ast_domain.TemplateNode{
								{
									NodeType:    ast_domain.NodeText,
									TextContent: "fragment body content",
								},
							},
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title: "Fragment Page",
					},
				},
				PageID:     "frag-test",
				IsFragment: true,
			},
			withReq:        true,
			wantContains:   []string{"fragment body content"},
			wantNotContain: []string{"<!DOCTYPE html>"},
		},
		{
			name: "full page mode includes DOCTYPE and html structure",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "full page body",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title:    "Full Page",
						Language: "en",
					},
				},
				PageID:     "full-test",
				IsFragment: false,
			},
			withReq: true,
			wantContains: []string{
				"<!DOCTYPE html>",
				"<html",
				"full page body",
				"</html>",
			},
		},
		{
			name: "full page with title includes title tag",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "titled page",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title: "My Test Title",
					},
				},
				PageID:     "titled-page",
				IsFragment: false,
			},
			withReq: true,
			wantContains: []string{
				"<title>My Test Title</title>",
			},
		},
		{
			name: "fragment with styling includes style content",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "styled fragment",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{},
				PageID:   "styled-frag",
				Styling:  "body { color: red; }",

				IsFragment: true,
			},
			withReq: true,
			wantContains: []string{
				"styled fragment",
				"body { color: red; }",
			},
		},
		{
			name: "renders without HTTP request and response (nil)",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "no-request content",
						},
					},
				},
				Metadata:   &templater_dto.InternalMetadata{},
				PageID:     "no-request",
				IsFragment: true,
			},
			withReq:      false,
			wantContains: []string{"no-request content"},
		},
		{
			name: "full page with empty AST renders structure only",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title: "Empty Body",
					},
				},
				PageID:     "empty-body",
				IsFragment: false,
			},
			withReq: true,
			wantContains: []string{
				"<!DOCTYPE html>",
				"</html>",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := tc.buildRO()
			var buffer bytes.Buffer

			var response http.ResponseWriter
			var request *http.Request
			if tc.withReq {
				response = httptest.NewRecorder()
				request = httptest.NewRequest(http.MethodGet, "/test", nil)
			}

			err := ro.RenderAST(context.Background(), &buffer, response, request, tc.opts)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			output := buffer.String()
			for _, want := range tc.wantContains {
				assert.Contains(t, output, want, "output should contain %q", want)
			}
			for _, notWant := range tc.wantNotContain {
				assert.NotContains(t, output, notWant, "output should not contain %q", notWant)
			}
		})
	}
}

func TestRenderAST_ContextCancellation(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	var buffer bytes.Buffer
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/cancelled", nil)

	err := ro.RenderAST(ctx, &buffer, response, request, RenderASTOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "should not render",
				},
			},
		},
		Metadata:   &templater_dto.InternalMetadata{},
		PageID:     "cancelled-page",
		IsFragment: true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled-page")
}

func TestRenderAST_WithHTTPRequest(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/with-http", nil)

	var buffer bytes.Buffer
	err := ro.RenderAST(context.Background(), &buffer, recorder, request, RenderASTOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "p",
					Children: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "http request content",
						},
					},
				},
			},
		},
		Metadata: &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{
				Title: "HTTP Request Test",
			},
		},
		PageID:     "http-page",
		IsFragment: false,
	})

	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "http request content")
	assert.Contains(t, output, "<!DOCTYPE html>")
}

func TestRenderFragment(t *testing.T) {
	testCases := []struct {
		name           string
		buildRO        func() *RenderOrchestrator
		buildRCtx      func() *renderContext
		opts           RenderASTOptions
		scriptHTML     string
		wantContains   []string
		wantNotContain []string
		wantErr        bool
	}{
		{
			name: "renders text content in fragment",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Hello fragment",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title: "Fragment Title",
					},
				},
				PageID: "frag-text",
			},
			wantContains: []string{"Hello fragment"},
		},
		{
			name: "includes module scripts when provided",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "with scripts",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{},
				PageID:   "frag-scripts",
			},
			scriptHTML:   `<script type="module" src="/app.js"></script>`,
			wantContains: []string{"with scripts"},
		},
		{
			name: "nil template renders header and footer only",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: nil,
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title: "Empty Fragment",
					},
				},
				PageID: "nil-frag",
			},
			wantNotContain: []string{"<!DOCTYPE html>"},
		},
		{
			name: "renders nested elements",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "div",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "class", Value: "wrapper"},
							},
							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "span",
									Children: []*ast_domain.TemplateNode{
										{
											NodeType:    ast_domain.NodeText,
											TextContent: "nested",
										},
									},
								},
							},
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{},
				PageID:   "nested-frag",
			},
			wantContains: []string{
				`class="wrapper"`,
				"<span>nested</span>",
			},
		},
		{
			name: "fragment with CSRF service injects CSRF meta tokens",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().
					WithCSRFService(newTestCSRFMock()).
					Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithCSRFService(newTestCSRFMock()).
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "csrf fragment",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{},
				PageID:   "csrf-frag",
			},
			wantContains: []string{"csrf fragment"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := tc.buildRO()
			rctx := tc.buildRCtx()

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			err := ro.renderFragment(context.Background(), qw, rctx, tc.opts, tc.scriptHTML)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			output := buffer.String()
			for _, want := range tc.wantContains {
				assert.Contains(t, output, want, "output should contain %q", want)
			}
			for _, notWant := range tc.wantNotContain {
				assert.NotContains(t, output, notWant, "output should not contain %q", notWant)
			}
		})
	}
}

func TestRenderFullPage(t *testing.T) {
	testCases := []struct {
		name         string
		buildRO      func() *RenderOrchestrator
		buildRCtx    func() *renderContext
		opts         RenderASTOptions
		preloadHTML  string
		scriptHTML   string
		wantContains []string
		wantErr      bool
	}{
		{
			name: "renders full HTML document structure",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "full page body content",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title:    "Full Page Title",
						Language: "en",
					},
				},
				PageID: "full-page",
			},
			wantContains: []string{
				"<!DOCTYPE html>",
				"<html",
				"full page body content",
				"</html>",
			},
		},
		{
			name: "includes title in head",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "content",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title: "My Page Title",
					},
				},
				PageID: "title-page",
			},
			wantContains: []string{
				"<title>My Page Title</title>",
			},
		},
		{
			name: "nil template renders page structure without body content",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: nil,
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title: "Empty Body Page",
					},
				},
				PageID: "empty-full",
			},
			wantContains: []string{
				"<!DOCTYPE html>",
				"</html>",
			},
		},
		{
			name: "includes styling when provided",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "styled page",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{},
				PageID:   "styled-full",
				Styling:  ".container { max-width: 1200px; }",
			},
			wantContains: []string{
				".container { max-width: 1200px; }",
				"styled page",
			},
		},
		{
			name: "renders with multiple root nodes",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "h1",
							Children: []*ast_domain.TemplateNode{
								{
									NodeType:    ast_domain.NodeText,
									TextContent: "Heading",
								},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "p",
							Children: []*ast_domain.TemplateNode{
								{
									NodeType:    ast_domain.NodeText,
									TextContent: "Paragraph",
								},
							},
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{},
				PageID:   "multi-root",
			},
			wantContains: []string{
				"<h1>Heading</h1>",
				"<p>Paragraph</p>",
			},
		},
		{
			name: "renders full page with description metadata",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			opts: RenderASTOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "described page",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{
						Title:       "Described Page",
						Description: "A page with a description",
					},
				},
				PageID: "described-page",
			},
			wantContains: []string{
				"A page with a description",
				"described page",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := tc.buildRO()
			rctx := tc.buildRCtx()

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			err := ro.renderFullPage(
				context.Background(), qw, rctx, tc.opts, tc.preloadHTML, tc.scriptHTML,
			)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			output := buffer.String()
			for _, want := range tc.wantContains {
				assert.Contains(t, output, want, "output should contain %q", want)
			}
		})
	}
}

func TestRenderFullPage_ContextCancellation(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	rctx := NewTestRenderContextBuilder().
		WithContext(ctx).
		WithHTTPRequest(testHTTPRequest()).
		Build()

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderFullPage(context.Background(), qw, rctx, RenderASTOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "should not render",
				},
			},
		},
		Metadata: &templater_dto.InternalMetadata{},
		PageID:   "cancel-full",
	}, "", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancel-full")
}

func TestRenderFragment_ContextCancellation(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	rctx := NewTestRenderContextBuilder().
		WithContext(ctx).
		WithHTTPRequest(testHTTPRequest()).
		Build()

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderFragment(context.Background(), qw, rctx, RenderASTOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "should not render",
				},
			},
		},
		Metadata: &templater_dto.InternalMetadata{},
		PageID:   "cancel-frag",
	}, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancel-frag")
}

func TestWriteElementDirectivesExcluding(t *testing.T) {
	testCases := []struct {
		name           string
		buildRO        func() *RenderOrchestrator
		buildRCtx      func() *renderContext
		node           *ast_domain.TemplateNode
		excludeAttrs   []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "no directives produces empty output",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			excludeAttrs: nil,
		},
		{
			name: "on-events are written",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "button",
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{RawExpression: "handleClick"},
					},
				},
			},
			excludeAttrs: nil,
			wantContains: []string{`p-on:click="handleClick"`},
		},
		{
			name: "custom events are written",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				CustomEvents: map[string][]ast_domain.Directive{
					"my-event": {
						{RawExpression: "handleMyEvent"},
					},
				},
			},
			excludeAttrs: nil,
			wantContains: []string{`p-event:my-event="handleMyEvent"`},
		},
		{
			name: "p-ref directive is written",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "input",
				DirRef: &ast_domain.Directive{
					Expression: &ast_domain.Identifier{Name: "myRef"},
				},
			},
			excludeAttrs: nil,
			wantContains: []string{`p-ref="myRef"`},
		},
		{
			name: "CSRF attributes are written when node needs CSRF",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithCSRFService(newTestCSRFMockWithTokens("eph-excl", []byte("act-excl"))).
					WithHTTPRequest(testHTTPRequest()).
					Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "form",
				RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
					NeedsCSRF: true,
				},
			},
			excludeAttrs: nil,
			wantContains: []string{
				"eph-excl",
				"act-excl",
				"data-csrf-ephemeral-token",
				"data-csrf-action-token",
			},
		},
		{
			name: "attribute writers with exclusion skip excluded attributes",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().Build()
			},
			node: func() *ast_domain.TemplateNode {
				srcWriter := &ast_domain.DirectWriter{Name: "src"}
				srcWriter.AppendString("image.png")

				altWriter := &ast_domain.DirectWriter{Name: "alt"}
				altWriter.AppendString("an image")

				return &ast_domain.TemplateNode{
					NodeType:         ast_domain.NodeElement,
					TagName:          "img",
					AttributeWriters: []*ast_domain.DirectWriter{srcWriter, altWriter},
				}
			}(),
			excludeAttrs:   []string{"src"},
			wantContains:   []string{`alt="an image"`},
			wantNotContain: []string{`src="image.png"`},
		},
		{
			name: "multiple exclusions skip all listed attributes",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().Build()
			},
			node: func() *ast_domain.TemplateNode {
				srcWriter := &ast_domain.DirectWriter{Name: "src"}
				srcWriter.AppendString("photo.jpg")

				srcsetWriter := &ast_domain.DirectWriter{Name: "srcset"}
				srcsetWriter.AppendString("photo-2x.jpg 2x")

				classWriter := &ast_domain.DirectWriter{Name: "class"}
				classWriter.AppendString("photo-class")

				return &ast_domain.TemplateNode{
					NodeType:         ast_domain.NodeElement,
					TagName:          "img",
					AttributeWriters: []*ast_domain.DirectWriter{srcWriter, srcsetWriter, classWriter},
				}
			}(),
			excludeAttrs:   []string{"src", "srcset"},
			wantContains:   []string{`class="photo-class"`},
			wantNotContain: []string{`src=`, `srcset=`},
		},
		{
			name: "no exclusions writes all attribute writers",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().Build()
			},
			node: func() *ast_domain.TemplateNode {
				srcWriter := &ast_domain.DirectWriter{Name: "src"}
				srcWriter.AppendString("file.png")

				altWriter := &ast_domain.DirectWriter{Name: "alt"}
				altWriter.AppendString("description")

				return &ast_domain.TemplateNode{
					NodeType:         ast_domain.NodeElement,
					TagName:          "img",
					AttributeWriters: []*ast_domain.DirectWriter{srcWriter, altWriter},
				}
			}(),
			excludeAttrs: nil,
			wantContains: []string{
				`src="file.png"`,
				`alt="description"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := tc.buildRO()
			rctx := tc.buildRCtx()

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			ro.writeElementDirectivesExcluding(tc.node, qw, rctx, tc.excludeAttrs...)

			output := buffer.String()
			for _, want := range tc.wantContains {
				assert.Contains(t, output, want, "output should contain %q", want)
			}
			for _, notWant := range tc.wantNotContain {
				assert.NotContains(t, output, notWant, "output should not contain %q", notWant)
			}
		})
	}
}

func TestBuildSvgSpriteSheetIfNeeded_AdditionalCases(t *testing.T) {
	testCases := []struct {
		name         string
		buildRO      func() *RenderOrchestrator
		buildRCtx    func() *renderContext
		wantContains []string
		wantEmpty    bool
	}{
		{
			name: "no required symbols returns empty",
			buildRO: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			buildRCtx: func() *renderContext {
				return NewTestRenderContextBuilder().Build()
			},
			wantEmpty: true,
		},
		{
			name: "single symbol produces sprite sheet with symbol",
			buildRO: func() *RenderOrchestrator {
				mockReg := newTestRegistryBuilder().
					withSVG("icon-star", `<path d="M12 2L15 9"/>`, ast_domain.HTMLAttribute{
						Name: "viewBox", Value: "0 0 24 24",
					}).build()
				return NewTestOrchestratorBuilder().
					WithRegistry(mockReg).
					Build()
			},
			buildRCtx: func() *renderContext {
				svgStar := &ParsedSvgData{
					InnerHTML:  `<path d="M12 2L15 9"/>`,
					Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
				}
				svgStar.CachedSymbol = ComputeSymbolString("icon-star", svgStar)
				mockReg := newTestRegistryBuilder().
					withSVG("icon-star", `<path d="M12 2L15 9"/>`, ast_domain.HTMLAttribute{
						Name: "viewBox", Value: "0 0 24 24",
					}).build()
				rctx := NewTestRenderContextBuilder().
					WithRegistry(mockReg).
					Build()
				rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
					svgSymbolEntry{id: "icon-star", data: svgStar})
				return rctx
			},
			wantContains: []string{
				"<svg",
				"icon-star",
				"</svg>",
			},
		},
		{
			name: "multiple symbols produces sprite sheet with all symbols",
			buildRO: func() *RenderOrchestrator {
				mockReg := newTestRegistryBuilder().
					withSVG("icon-a", `<circle cx="10" cy="10" r="5"/>`, ast_domain.HTMLAttribute{
						Name: "viewBox", Value: "0 0 20 20",
					}).
					withSVG("icon-b", `<rect width="10" height="10"/>`, ast_domain.HTMLAttribute{
						Name: "viewBox", Value: "0 0 10 10",
					}).build()
				return NewTestOrchestratorBuilder().
					WithRegistry(mockReg).
					Build()
			},
			buildRCtx: func() *renderContext {
				svgA := &ParsedSvgData{
					InnerHTML:  `<circle cx="10" cy="10" r="5"/>`,
					Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 20 20"}},
				}
				svgA.CachedSymbol = ComputeSymbolString("icon-a", svgA)
				svgB := &ParsedSvgData{
					InnerHTML:  `<rect width="10" height="10"/>`,
					Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 10 10"}},
				}
				svgB.CachedSymbol = ComputeSymbolString("icon-b", svgB)
				mockReg := newTestRegistryBuilder().
					withSVG("icon-a", `<circle cx="10" cy="10" r="5"/>`, ast_domain.HTMLAttribute{
						Name: "viewBox", Value: "0 0 20 20",
					}).
					withSVG("icon-b", `<rect width="10" height="10"/>`, ast_domain.HTMLAttribute{
						Name: "viewBox", Value: "0 0 10 10",
					}).build()
				rctx := NewTestRenderContextBuilder().
					WithRegistry(mockReg).
					Build()
				rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
					svgSymbolEntry{id: "icon-a", data: svgA},
					svgSymbolEntry{id: "icon-b", data: svgB},
				)
				return rctx
			},
			wantContains: []string{
				"<svg",
				"icon-a",
				"icon-b",
				"</svg>",
			},
		},
		{
			name: "registry error produces wrapper without symbol definitions",
			buildRO: func() *RenderOrchestrator {
				mockReg := newTestRegistryBuilder().
					withSVGError(errors.New("registry failure")).
					build()
				return NewTestOrchestratorBuilder().
					WithRegistry(mockReg).
					Build()
			},
			buildRCtx: func() *renderContext {
				mockReg := newTestRegistryBuilder().
					withSVGError(errors.New("registry failure")).
					build()
				rctx := NewTestRenderContextBuilder().
					WithRegistry(mockReg).
					Build()
				rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
					svgSymbolEntry{id: "missing-icon", data: nil})
				return rctx
			},

			wantContains: []string{"<svg"},
		},
		{
			name: "symbol not in registry returns empty or omits symbol",
			buildRO: func() *RenderOrchestrator {
				mockReg := &MockRegistryPort{}
				return NewTestOrchestratorBuilder().
					WithRegistry(mockReg).
					Build()
			},
			buildRCtx: func() *renderContext {
				mockReg := &MockRegistryPort{}
				rctx := NewTestRenderContextBuilder().
					WithRegistry(mockReg).
					Build()
				rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
					svgSymbolEntry{id: "nonexistent-icon", data: nil})
				return rctx
			},

			wantEmpty: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := tc.buildRO()
			rctx := tc.buildRCtx()

			result := ro.buildSvgSpriteSheetIfNeeded(context.Background(), rctx)

			if tc.wantEmpty {
				assert.Empty(t, result)
				return
			}

			for _, want := range tc.wantContains {
				assert.Contains(t, result, want, "result should contain %q", want)
			}
		})
	}
}

func TestRenderAST_FragmentVsFullPage(t *testing.T) {
	template := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "section",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "shared content",
					},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		wantContains   []string
		wantNotContain []string
		isFragment     bool
	}{
		{
			name:       "fragment mode omits DOCTYPE",
			isFragment: true,
			wantContains: []string{
				"shared content",
			},
			wantNotContain: []string{
				"<!DOCTYPE html>",
			},
		},
		{
			name:       "full page mode includes DOCTYPE",
			isFragment: false,
			wantContains: []string{
				"<!DOCTYPE html>",
				"shared content",
				"</html>",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := NewTestOrchestratorBuilder().Build()

			localTemplate := &ast_domain.TemplateAST{
				RootNodes: template.RootNodes,
			}

			var buffer bytes.Buffer
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/test", nil)

			err := ro.RenderAST(context.Background(), &buffer, response, request, RenderASTOptions{
				Template:   localTemplate,
				Metadata:   &templater_dto.InternalMetadata{},
				PageID:     "frag-vs-full",
				IsFragment: tc.isFragment,
			})

			require.NoError(t, err)

			output := buffer.String()
			for _, want := range tc.wantContains {
				assert.Contains(t, output, want)
			}
			for _, notWant := range tc.wantNotContain {
				assert.NotContains(t, output, notWant)
			}
		})
	}
}

func TestRenderAST_NilMetadata(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	var buffer bytes.Buffer
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/nil-meta", nil)

	err := ro.RenderAST(context.Background(), &buffer, response, request, RenderASTOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "nil metadata content",
				},
			},
		},
		Metadata:   &templater_dto.InternalMetadata{},
		PageID:     "nil-meta-page",
		IsFragment: true,
	})

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "nil metadata content")
}

func TestRenderAST_EmptyPageID(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	var buffer bytes.Buffer
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/empty-id", nil)

	err := ro.RenderAST(context.Background(), &buffer, response, request, RenderASTOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "empty id content",
				},
			},
		},
		Metadata:   &templater_dto.InternalMetadata{},
		PageID:     "",
		IsFragment: true,
	})

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "empty id content")
}

func TestWriteElementDirectivesExcluding_CombinesDirectivesAndExclusion(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	srcWriter := &ast_domain.DirectWriter{Name: "src"}
	srcWriter.AppendString("image.png")

	classWriter := &ast_domain.DirectWriter{Name: "class"}
	classWriter.AppendString("my-class")

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "img",
		OnEvents: map[string][]ast_domain.Directive{
			"load": {
				{RawExpression: "onLoad"},
			},
		},
		DirRef: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "imgRef"},
		},
		AttributeWriters: []*ast_domain.DirectWriter{srcWriter, classWriter},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	ro.writeElementDirectivesExcluding(node, qw, rctx, "src")

	output := buffer.String()

	assert.Contains(t, output, `p-on:load="onLoad"`)

	assert.Contains(t, output, `p-ref="imgRef"`)

	assert.Contains(t, output, `class="my-class"`)

	assert.NotContains(t, output, `src="image.png"`)
}
