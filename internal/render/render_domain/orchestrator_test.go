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
	"strings"
	"testing"
	"time"

	"sync/atomic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestRenderNode_DispatchesToCorrectHandler(t *testing.T) {
	testCases := []struct {
		name           string
		node           *ast_domain.TemplateNode
		expectedOutput string
	}{
		{
			name: "text node renders content",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "Hello World",
			},
			expectedOutput: "Hello World",
		},
		{
			name: "comment node renders as HTML comment",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeComment,
				TextContent: "This is a comment",
			},
			expectedOutput: "<!--This is a comment-->",
		},
		{
			name: "element node renders with tag",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			expectedOutput: "<div></div>",
		},
		{
			name: "fragment node renders children only",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeFragment,
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Fragment content"},
				},
			},
			expectedOutput: "Fragment content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().Build()
			ro := NewTestOrchestratorBuilder().Build()

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			err := ro.renderNode(context.Background(), tc.node, qw, rctx)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedOutput, buffer.String())
		})
	}
}

func TestRenderElement_WritesAttributes(t *testing.T) {
	testCases := []struct {
		name           string
		attrs          []ast_domain.HTMLAttribute
		expectedOutput []string
	}{
		{
			name: "single attribute",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
			expectedOutput: []string{`<div class="container">`},
		},
		{
			name: "multiple attributes",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
				{Name: "data-value", Value: "test"},
			},
			expectedOutput: []string{
				`class="container"`,
				`id="main"`,
				`data-value="test"`,
			},
		},
		{
			name: "boolean attribute",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "disabled", Value: ""},
			},
			expectedOutput: []string{`disabled=""`},
		},
		{
			name: "attribute with special characters",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "data-json", Value: `{"key":"value"}`},
			},
			expectedOutput: []string{`data-json=`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().Build()
			ro := NewTestOrchestratorBuilder().Build()

			node := &ast_domain.TemplateNode{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: tc.attrs,
			}

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			err := ro.renderNode(context.Background(), node, qw, rctx)
			require.NoError(t, err)

			output := buffer.String()
			for _, expected := range tc.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestRenderElement_HandlesVoidElements(t *testing.T) {
	voidElements := []string{
		"area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr",
	}

	for _, tag := range voidElements {
		t.Run(tag, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().Build()
			ro := NewTestOrchestratorBuilder().Build()

			node := &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  tag,
			}

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			err := ro.renderNode(context.Background(), node, qw, rctx)
			require.NoError(t, err)

			output := buffer.String()

			assert.Contains(t, output, "<"+tag)
			assert.Contains(t, output, "/>")

			assert.NotContains(t, output, "</"+tag+">")
		})
	}
}

func TestRenderElement_WritesChildren(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Before "},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "span",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "nested"},
				},
			},
			{NodeType: ast_domain.NodeText, TextContent: " After"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	expected := "<div>Before <span>nested</span> After</div>"
	assert.Equal(t, expected, buffer.String())
}

func TestRenderElement_HandlesCSRF(t *testing.T) {
	mockCSRF := newTestCSRFMockWithTokens("ephemeral-123", []byte("action-456"))

	rctx := NewTestRenderContextBuilder().
		WithCSRFService(mockCSRF).
		WithHTTPRequest(testHTTPRequest()).
		Build()

	ro := NewTestOrchestratorBuilder().
		WithCSRFService(mockCSRF).
		Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "form",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "p-csrf", Value: ""},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, "<form")
	assert.Contains(t, output, "p-csrf")
}

func TestRenderNode_HandlesPikoSvg(t *testing.T) {
	mockReg := newTestRegistryBuilder().
		withSVG("test-icon", `<path d="M0 0"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "piko:svg",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "test-icon"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<svg")
	assert.Contains(t, output, "<use")
	assert.Contains(t, output, "#test-icon")
}

func TestRenderNode_HandlesPikoA(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "piko:a",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "href", Value: "/about"},
		},
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "About Us"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<a")
	assert.Contains(t, output, `href="/about"`)
	assert.Contains(t, output, "piko:a")
	assert.Contains(t, output, "About Us")
}

func TestRenderNode_HandlesPikoImg(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/assets/image.png"},
			{Name: "alt", Value: "Example image"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<img")
	assert.Contains(t, output, "/_piko/assets/")
	assert.Contains(t, output, `alt="Example image"`)
}

func TestRenderNode_HandlesEmptyTree(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderASTToWriter(&ast_domain.TemplateAST{RootNodes: nil}, qw, rctx)
	require.NoError(t, err)
	assert.Empty(t, buffer.String())
}

func TestRenderNode_HandlesDeeplyNestedStructure(t *testing.T) {

	depth := 10
	innermost := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "span",
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "deep"},
		},
	}

	current := innermost
	for range depth {
		current = &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{current},
		}
	}

	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), current, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<span>deep</span>")

	assert.Equal(t, depth, countSubstring(output, "<div>"))
	assert.Equal(t, depth, countSubstring(output, "</div>"))
}

func countSubstring(s, substr string) int {
	count := 0
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}

func TestWriteNodeAndFragmentAttributes_NodeAttributesOnly(t *testing.T) {
	nodeAttrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "container"},
		{Name: "id", Value: "main"},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeNodeAndFragmentAttributes(nodeAttrs, nil, nil, qw, nil)

	output := buffer.String()
	assert.Contains(t, output, `class="container"`)
	assert.Contains(t, output, `id="main"`)
}

func TestWriteNodeAndFragmentAttributes_FragmentAttributesOnly(t *testing.T) {
	fragmentAttrs := []ast_domain.HTMLAttribute{
		{Name: "data-fragment", Value: "true"},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeNodeAndFragmentAttributes(nil, fragmentAttrs, nil, qw, nil)

	output := buffer.String()
	assert.Contains(t, output, `data-fragment="true"`)
}

func TestWriteNodeAndFragmentAttributes_NodeOverridesFragment(t *testing.T) {
	nodeAttrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "node-class"},
	}
	fragmentAttrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "fragment-class"},
		{Name: "data-unique", Value: "fragment"},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeNodeAndFragmentAttributes(nodeAttrs, fragmentAttrs, nil, qw, nil)

	output := buffer.String()
	assert.Contains(t, output, `class="node-class"`)
	assert.NotContains(t, output, "fragment-class")
	assert.Contains(t, output, `data-unique="fragment"`)
}

func TestWriteEventDirectives_OnEvents(t *testing.T) {
	events := map[string][]ast_domain.Directive{
		"click": {
			{RawExpression: "handleClick", Modifier: ""},
		},
		"submit": {
			{RawExpression: "handleSubmit", Modifier: "prevent"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeEventDirectives(events, pOnPrefix, qw)

	output := buffer.String()
	assert.Contains(t, output, `p-on:click="handleClick"`)
	assert.Contains(t, output, `p-on:submit.prevent="handleSubmit"`)
}

func TestWriteEventDirectives_CustomEvents(t *testing.T) {
	events := map[string][]ast_domain.Directive{
		"custom-event": {
			{RawExpression: "handleCustom", Modifier: "once"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeEventDirectives(events, pEventPrefix, qw)

	output := buffer.String()
	assert.Contains(t, output, `p-event:custom-event.once="handleCustom"`)
}

func TestWriteEventDirectives_MultipleHandlersForSameEvent(t *testing.T) {
	events := map[string][]ast_domain.Directive{
		"click": {
			{RawExpression: "handler1", Modifier: ""},
			{RawExpression: "handler2", Modifier: "stop"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeEventDirectives(events, pOnPrefix, qw)

	output := buffer.String()
	assert.Contains(t, output, `p-on:click="handler1"`)
	assert.Contains(t, output, `p-on:click.stop="handler2"`)
}

func TestWriteEventDirectives_EmptyMap(t *testing.T) {
	events := map[string][]ast_domain.Directive{}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeEventDirectives(events, pOnPrefix, qw)

	assert.Empty(t, buffer.String())
}

func TestRenderNodeContent_WithInnerHTML(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType:  ast_domain.NodeElement,
		TagName:   "div",
		InnerHTML: "<span>inner content</span>",
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNodeContent(node, qw, rctx)
	require.NoError(t, err)

	assert.Equal(t, "<span>inner content</span>", buffer.String())
}

func TestRenderNodeContent_WithTextContent(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeElement,
		TagName:     "p",
		TextContent: "Plain text content",
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNodeContent(node, qw, rctx)
	require.NoError(t, err)

	assert.Equal(t, "Plain text content", buffer.String())
}

func TestRenderNodeContent_WithChildren(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Child text"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNodeContent(node, qw, rctx)
	require.NoError(t, err)

	assert.Equal(t, "Child text", buffer.String())
}

func TestRenderNodeContent_Empty(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNodeContent(node, qw, rctx)
	require.NoError(t, err)

	assert.Empty(t, buffer.String())
}

func TestRenderNodeContent_InnerHTMLTakesPriority(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeElement,
		TagName:     "div",
		InnerHTML:   "<strong>InnerHTML</strong>",
		TextContent: "TextContent",
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Children"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNodeContent(node, qw, rctx)
	require.NoError(t, err)

	assert.Equal(t, "<strong>InnerHTML</strong>", buffer.String())
}

func TestGetCSRFIfNeeded_NoAnnotations(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType:           ast_domain.NodeElement,
		TagName:            "form",
		RuntimeAnnotations: nil,
	}

	result := ro.getCSRFIfNeeded(node, rctx)
	assert.Nil(t, result)
}

func TestGetCSRFIfNeeded_CSRFNotNeeded(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "form",
		RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
			NeedsCSRF: false,
		},
	}

	result := ro.getCSRFIfNeeded(node, rctx)
	assert.Nil(t, result)
}

func TestGetCSRFIfNeeded_CSRFNeededButNoService(t *testing.T) {
	rctx := NewTestRenderContextBuilder().
		WithHTTPRequest(testHTTPRequest()).
		Build()
	rctx.csrfService = nil

	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "form",
		RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
			NeedsCSRF: true,
		},
	}

	result := ro.getCSRFIfNeeded(node, rctx)
	assert.Nil(t, result)
}

func TestGetCSRFIfNeeded_CSRFNeededButNoRequest(t *testing.T) {
	mockCSRF := newTestCSRFMock()

	rctx := NewTestRenderContextBuilder().
		WithCSRFService(mockCSRF).
		Build()
	rctx.httpRequest = nil

	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "form",
		RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
			NeedsCSRF: true,
		},
	}

	result := ro.getCSRFIfNeeded(node, rctx)
	assert.Nil(t, result)
}

func TestGetCSRFIfNeeded_GeneratesCSRF(t *testing.T) {
	mockCSRF := newTestCSRFMockWithTokens("ephemeral-token", []byte("action-token"))

	rctx := NewTestRenderContextBuilder().
		WithCSRFService(mockCSRF).
		WithHTTPRequest(testHTTPRequest()).
		Build()

	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "form",
		RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
			NeedsCSRF: true,
		},
	}

	result := ro.getCSRFIfNeeded(node, rctx)
	require.NotNil(t, result)
	assert.Equal(t, "ephemeral-token", result.RawEphemeralToken)
	assert.Equal(t, []byte("action-token"), result.ActionToken)
}

func TestGetCSRFIfNeeded_CachesResult(t *testing.T) {
	mockCSRF := newTestCSRFMockWithTokens("cached-ephemeral", []byte("cached-action"))

	rctx := NewTestRenderContextBuilder().
		WithCSRFService(mockCSRF).
		WithHTTPRequest(testHTTPRequest()).
		Build()

	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "form",
		RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
			NeedsCSRF: true,
		},
	}

	result1 := ro.getCSRFIfNeeded(node, rctx)
	require.NotNil(t, result1)

	result2 := ro.getCSRFIfNeeded(node, rctx)
	require.NotNil(t, result2)

	assert.Equal(t, result1.RawEphemeralToken, result2.RawEphemeralToken)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mockCSRF.GenerateCSRFPairCallCount))
}

func TestIsVoidElement_AllVoidElements(t *testing.T) {
	voidElements := []string{
		"area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr",
	}

	for _, tag := range voidElements {
		t.Run(tag, func(t *testing.T) {
			assert.True(t, isVoidElement(tag), "%s should be a void element", tag)
		})
	}
}

func TestIsVoidElement_NonVoidElements(t *testing.T) {
	nonVoidElements := []string{
		"div", "span", "p", "a", "button", "form", "table", "script", "style",
	}

	for _, tag := range nonVoidElements {
		t.Run(tag, func(t *testing.T) {
			assert.False(t, isVoidElement(tag), "%s should not be a void element", tag)
		})
	}
}

func TestWriteElementDirectives_WithCSRF(t *testing.T) {
	mockCSRF := newTestCSRFMockWithTokens("eph-token", []byte("action-token"))

	rctx := NewTestRenderContextBuilder().
		WithCSRFService(mockCSRF).
		WithHTTPRequest(testHTTPRequest()).
		Build()

	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "form",
		RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
			NeedsCSRF: true,
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	ro.writeElementDirectives(node, qw, rctx)

	output := buffer.String()

	assert.Contains(t, output, "eph-token")
	assert.Contains(t, output, "action-token")
}

func TestWriteElementDirectives_WithNoCSRF(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	ro.writeElementDirectives(node, qw, rctx)

	output := buffer.String()

	assert.Empty(t, output)
}

func TestLogCollectedDiagnostics_LogsWarnings(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	rctx.diagnostics.AddWarning("testLocation", "Test warning message", map[string]string{"key": "value"})
	rctx.diagnostics.AddWarning("anotherLocation", "Another warning", nil)

	assert.NotPanics(t, func() {
		logCollectedDiagnostics(context.Background(), rctx)
	})
}

func TestLogCollectedDiagnostics_LogsErrors(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	rctx.diagnostics.AddError("testLocation", errors.New("test error"), "Error message", map[string]string{"key": "value"})

	assert.NotPanics(t, func() {
		logCollectedDiagnostics(context.Background(), rctx)
	})
}

func TestLogCollectedDiagnostics_EmptyDiagnostics(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	assert.NotPanics(t, func() {
		logCollectedDiagnostics(context.Background(), rctx)
	})
}

func TestRenderNode_WithNestedElements(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "class", Value: "container"},
		},
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "span",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Hello"},
				},
			},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, `<div class="container">`)
	assert.Contains(t, output, "<span>Hello</span>")
	assert.Contains(t, output, "</div>")
}

func TestRenderNode_WithCommentNode(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeComment,
		TextContent: "This is a comment",
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<!--")
	assert.Contains(t, output, "This is a comment")
	assert.Contains(t, output, "-->")
}

func TestRenderNode_WithRawHTMLNode(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	rawContent := `<!--[if mso]><v:roundrect><![endif]-->`
	node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeRawHTML,
		TextContent: rawContent,
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Equal(t, rawContent, output)
}

func TestRenderNode_FragmentWithMixedChildren(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeFragment,
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Text before "},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "span",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "element"},
				},
			},
			{NodeType: ast_domain.NodeComment, TextContent: " comment "},
			{NodeType: ast_domain.NodeRawHTML, TextContent: "<raw>html</raw>"},
			{NodeType: ast_domain.NodeText, TextContent: " Text after"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "Text before ")
	assert.Contains(t, output, "<span>element</span>")
	assert.Contains(t, output, "<!-- comment -->")
	assert.Contains(t, output, "<raw>html</raw>")
	assert.Contains(t, output, " Text after")
}

func TestRenderNode_NestedFragments(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeFragment,
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeFragment,
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Nested fragment content"},
				},
			},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	assert.Equal(t, "Nested fragment content", buffer.String())
}

func TestRenderNode_ContextCancellation(t *testing.T) {

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	rctx := NewTestRenderContextBuilder().Build()
	rctx.originalCtx = ctx
	rctx.pageID = "test-page"

	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Should not render",
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rendering cancelled")
	assert.Contains(t, err.Error(), "test-page")
}

func TestRenderNode_FragmentPassesAttributesToChildren(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeFragment,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "data-fragment", Value: "true"},
			{Name: "class", Value: "from-fragment"},
		},
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Content"},
				},
			},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderNode(context.Background(), node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, "data-fragment")
	assert.Contains(t, output, "class=")
	assert.Contains(t, output, "Content")
}

func TestPopulateTagMap_PopulatesTags(t *testing.T) {
	dest := make(map[string]struct{})
	tags := []string{"my-component", "another-component", "third-component"}

	populateTagMap(dest, tags)

	assert.Len(t, dest, 3)
	_, exists := dest["my-component"]
	assert.True(t, exists)
	_, exists = dest["another-component"]
	assert.True(t, exists)
	_, exists = dest["third-component"]
	assert.True(t, exists)
}

func TestPopulateTagMap_EmptyTags(t *testing.T) {
	dest := make(map[string]struct{})

	populateTagMap(dest, []string{})

	assert.Empty(t, dest)
}

func TestPopulateTagMap_NilTags(t *testing.T) {
	dest := make(map[string]struct{})

	populateTagMap(dest, nil)

	assert.Empty(t, dest)
}

func TestRenderOrchestrator_Name(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	name := ro.Name()

	assert.Equal(t, "RenderService", name)
}

func TestGetLastEmailAssetRequests_ReturnsEmptySliceByDefault(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	requests := ro.GetLastEmailAssetRequests()

	assert.NotNil(t, requests)
	assert.Empty(t, requests)
}

func TestNewRenderOrchestrator_CreatesOrchestrator(t *testing.T) {
	mockReg := &MockRegistryPort{}
	mockCSRF := newTestCSRFMock()

	ro := NewRenderOrchestrator(nil, nil, mockReg, mockCSRF)

	assert.NotNil(t, ro)
	assert.Equal(t, mockReg, ro.registry)
	assert.Equal(t, mockCSRF, ro.csrfService)
}

func TestNewRenderOrchestrator_WithTransforms(t *testing.T) {
	mockReg := &MockRegistryPort{}
	transforms := []TransformationPort{}

	ro := NewRenderOrchestrator(nil, transforms, mockReg, nil)

	assert.NotNil(t, ro)
	assert.NotNil(t, ro.transformSteps)
}

func TestCheckLiveness_Healthy(t *testing.T) {
	mockReg := &MockRegistryPort{}
	ro := &RenderOrchestrator{
		pmlEngine: &pml_domain.MockTransformer{},
		registry:  mockReg,
	}

	status := ro.checkLiveness(time.Now())

	assert.Equal(t, "RenderService", status.Name)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Contains(t, status.Message, "running")
}

func TestCheckLiveness_UnhealthyNoPMLEngine(t *testing.T) {
	mockReg := &MockRegistryPort{}
	ro := &RenderOrchestrator{
		pmlEngine: nil,
		registry:  mockReg,
	}

	status := ro.checkLiveness(time.Now())

	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	assert.Contains(t, status.Message, "PML engine")
}

func TestCheckLiveness_UnhealthyNoRegistry(t *testing.T) {
	ro := &RenderOrchestrator{
		pmlEngine: &pml_domain.MockTransformer{},
		registry:  nil,
	}

	status := ro.checkLiveness(time.Now())

	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	assert.Contains(t, status.Message, "Registry")
}

func TestCheck_LivenessType(t *testing.T) {
	mockReg := &MockRegistryPort{}
	ro := &RenderOrchestrator{
		pmlEngine: &pml_domain.MockTransformer{},
		registry:  mockReg,
	}

	status := ro.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
}

func TestAddTransformPipelineStatus_NilTransforms(t *testing.T) {
	ro := &RenderOrchestrator{
		transformSteps: nil,
	}

	deps := make([]*healthprobe_dto.Status, 0)
	ro.addTransformPipelineStatus(&deps)

	assert.Len(t, deps, 1)
	assert.Equal(t, "TransformPipeline", deps[0].Name)
}

func TestAddTransformPipelineStatus_WithTransforms(t *testing.T) {
	ro := &RenderOrchestrator{
		transformSteps: []TransformationPort{},
	}

	deps := make([]*healthprobe_dto.Status, 0)
	ro.addTransformPipelineStatus(&deps)

	assert.Len(t, deps, 1)
	assert.Equal(t, healthprobe_dto.StateHealthy, deps[0].State)
}

func TestRenderASTToPlainText_NilAST(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	result, err := ro.RenderASTToPlainText(context.Background(), nil)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestRenderASTToPlainText_SimpleText(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Hello World"},
		},
	}

	result, err := ro.RenderASTToPlainText(context.Background(), ast)

	require.NoError(t, err)
	assert.Equal(t, "Hello World", result)
}

func TestRenderASTToPlainText_WithElements(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "p",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Paragraph text"},
				},
			},
		},
	}

	result, err := ro.RenderASTToPlainText(context.Background(), ast)

	require.NoError(t, err)
	assert.Contains(t, result, "Paragraph text")
}

func TestCheckReadiness_Healthy(t *testing.T) {
	mockReg := &MockRegistryPort{}
	ro := &RenderOrchestrator{
		pmlEngine:      &pml_domain.MockTransformer{},
		registry:       mockReg,
		transformSteps: []TransformationPort{},
	}

	status := ro.checkReadiness(context.Background(), healthprobe_dto.CheckTypeReadiness, time.Now())

	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Contains(t, status.Message, "operational")
	assert.NotEmpty(t, status.Dependencies)
}

func TestCheck_ReadinessType(t *testing.T) {
	mockReg := &MockRegistryPort{}
	ro := &RenderOrchestrator{
		pmlEngine:      &pml_domain.MockTransformer{},
		registry:       mockReg,
		transformSteps: []TransformationPort{},
	}

	status := ro.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
}

func TestCheckRegistryHealth_NoHealthProbeInterface(t *testing.T) {

	mockReg := &MockRegistryPort{}
	ro := &RenderOrchestrator{
		registry: mockReg,
	}

	deps := make([]*healthprobe_dto.Status, 0)
	result := ro.checkRegistryHealth(context.Background(), healthprobe_dto.CheckTypeReadiness, &deps, healthprobe_dto.StateHealthy)

	assert.Equal(t, healthprobe_dto.StateHealthy, result)
	assert.Empty(t, deps)
}

func TestCheckRegistryHealth_WithHealthProbeInterface(t *testing.T) {
	mockReg := &mockRegistryWithHealthProbe{
		status: healthprobe_dto.Status{
			Name:    "MockRegistry",
			State:   healthprobe_dto.StateHealthy,
			Message: "OK",
		},
	}
	ro := &RenderOrchestrator{
		registry: mockReg,
	}

	deps := make([]*healthprobe_dto.Status, 0)
	result := ro.checkRegistryHealth(context.Background(), healthprobe_dto.CheckTypeReadiness, &deps, healthprobe_dto.StateHealthy)

	assert.Equal(t, healthprobe_dto.StateHealthy, result)
	assert.Len(t, deps, 1)
}

func TestCheckRegistryHealth_UnhealthyRegistry(t *testing.T) {
	mockReg := &mockRegistryWithHealthProbe{
		status: healthprobe_dto.Status{
			Name:    "MockRegistry",
			State:   healthprobe_dto.StateUnhealthy,
			Message: "Database down",
		},
	}
	ro := &RenderOrchestrator{
		registry: mockReg,
	}

	result := ro.checkRegistryHealth(context.Background(), healthprobe_dto.CheckTypeReadiness, new(make([]*healthprobe_dto.Status, 0)), healthprobe_dto.StateHealthy)

	assert.Equal(t, healthprobe_dto.StateUnhealthy, result)
}

func TestCheckRegistryHealth_DegradedRegistry(t *testing.T) {
	mockReg := &mockRegistryWithHealthProbe{
		status: healthprobe_dto.Status{
			Name:    "MockRegistry",
			State:   healthprobe_dto.StateDegraded,
			Message: "Cache miss rate high",
		},
	}
	ro := &RenderOrchestrator{
		registry: mockReg,
	}

	result := ro.checkRegistryHealth(context.Background(), healthprobe_dto.CheckTypeReadiness, new(make([]*healthprobe_dto.Status, 0)), healthprobe_dto.StateHealthy)

	assert.Equal(t, healthprobe_dto.StateDegraded, result)
}

type mockRegistryWithHealthProbe struct {
	status healthprobe_dto.Status
	MockRegistryPort
}

func (m *mockRegistryWithHealthProbe) Name() string {
	return "MockRegistry"
}

func (m *mockRegistryWithHealthProbe) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return m.status
}

func TestGetRenderContext_InitialisesFields(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	request := testHTTPRequest()
	ctx := context.Background()

	rctx := ro.getRenderContext(ctx, "test-page", nil, request, nil)
	defer ro.putRenderContext(rctx)

	assert.NotNil(t, rctx.originalCtx)
	assert.Equal(t, "test-page", rctx.pageID)
	assert.NotNil(t, rctx.registry)
	assert.Equal(t, request, rctx.httpRequest)
	assert.Empty(t, rctx.collectedLinkHeaders)
	assert.Empty(t, rctx.diagnostics.Warnings)
	assert.Empty(t, rctx.diagnostics.Errors)
}

func TestGetRenderContext_ClearsPreviousState(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	ctx := context.Background()

	rctx := ro.getRenderContext(ctx, "page-1", nil, nil, nil)
	rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols, svgSymbolEntry{id: "icon-a"})
	rctx.collectedCustomComponents["comp-a"] = struct{}{}
	rctx.diagnostics.AddWarning("test", "warning", nil)

	ro.putRenderContext(rctx)

	rctx2 := ro.getRenderContext(ctx, "page-2", nil, nil, nil)
	defer ro.putRenderContext(rctx2)

	assert.Empty(t, rctx2.requiredSvgSymbols)
	assert.Empty(t, rctx2.collectedCustomComponents)
	assert.Empty(t, rctx2.diagnostics.Warnings)
	assert.Equal(t, "page-2", rctx2.pageID)
}

func TestPutRenderContext_ClearsNonPoolableFields(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	ctx := context.Background()
	request := testHTTPRequest()

	rctx := ro.getRenderContext(ctx, "test-page", nil, request, nil)

	rctx.currentLocale = "fr"
	rctx.i18nStrategy = "prefix"
	rctx.defaultLocale = "en"

	ro.putRenderContext(rctx)

	assert.Nil(t, rctx.originalCtx)
	assert.Nil(t, rctx.httpRequest)
	assert.Empty(t, rctx.currentLocale)
	assert.Empty(t, rctx.i18nStrategy)
	assert.Empty(t, rctx.defaultLocale)
}

func TestGetRenderContext_ExtractsLocaleFromRequest(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	request := testHTTPRequest()

	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		Locale: "de",
	})
	request = request.WithContext(ctx)

	rctx := ro.getRenderContext(context.Background(), "test-page", nil, request, nil)
	defer ro.putRenderContext(rctx)

	assert.Equal(t, "de", rctx.currentLocale)
}

func TestGetRenderContext_NoLocaleInRequest(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	request := testHTTPRequest()

	rctx := ro.getRenderContext(context.Background(), "test-page", nil, request, nil)
	defer ro.putRenderContext(rctx)

	assert.Empty(t, rctx.currentLocale)
}

func TestGetRenderContext_NilRequest(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	rctx := ro.getRenderContext(context.Background(), "test-page", nil, nil, nil)
	defer ro.putRenderContext(rctx)

	assert.Nil(t, rctx.httpRequest)
	assert.Empty(t, rctx.currentLocale)
}

func TestBuildSvgSpriteSheetIfNeeded_NoSymbols(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	result := ro.buildSvgSpriteSheetIfNeeded(context.Background(), rctx)

	assert.Empty(t, result)
}

func TestBuildSvgSpriteSheetIfNeeded_WithSymbols(t *testing.T) {
	svgData := &ParsedSvgData{
		InnerHTML:  `<path d="M10 20v-6h4v6"/>`,
		Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
	}
	svgData.CachedSymbol = ComputeSymbolString("icon-home", svgData)

	mockReg := newTestRegistryBuilder().
		withSVG("icon-home", `<path d="M10 20v-6h4v6"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		build()

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()
	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
		svgSymbolEntry{id: "icon-home", data: svgData})

	result := ro.buildSvgSpriteSheetIfNeeded(context.Background(), rctx)

	assert.Contains(t, result, "<svg")
	assert.Contains(t, result, "icon-home")
	assert.Contains(t, result, "</svg>")
}

func TestBuildSvgSpriteSheetIfNeeded_WithError(t *testing.T) {
	mockReg := newTestRegistryBuilder().
		withSVGError(errors.New("SVG load failed")).
		build()

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()
	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
		svgSymbolEntry{id: "missing-icon", data: nil})

	result := ro.buildSvgSpriteSheetIfNeeded(context.Background(), rctx)

	assert.True(t, result == "" || !strings.Contains(result, "<symbol"))
}

func TestRenderASTToWriter_NilAST(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderASTToWriter(nil, qw, rctx)

	assert.NoError(t, err)
	assert.Empty(t, buffer.String())
}

func TestRenderASTToWriter_EmptyAST(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderASTToWriter(ast, qw, rctx)

	assert.NoError(t, err)
	assert.Empty(t, buffer.String())
}

func TestRenderASTToWriter_SimpleContent(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "p",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Hello World"},
				},
			},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderASTToWriter(ast, qw, rctx)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "<p>Hello World</p>")
}

func TestRenderASTToWriter_MultipleRootNodes(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "div", Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "First"},
			}},
			{NodeType: ast_domain.NodeElement, TagName: "div", Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "Second"},
			}},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := ro.renderASTToWriter(ast, qw, rctx)

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "<div>First</div>")
	assert.Contains(t, output, "<div>Second</div>")
}

func TestGetBuffer_ReturnsValidBuffer(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	buffer := rctx.getBuffer()

	assert.NotNil(t, buffer)
	assert.NotNil(t, *buffer)
}

func TestFreezeToString_ConvertsBufferToString(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	buffer := rctx.getBuffer()
	*buffer = append(*buffer, "Hello, World!"...)

	result := rctx.freezeToString(buffer)

	assert.Equal(t, "Hello, World!", result)
	assert.Len(t, rctx.frozenBuffers, 1)
}

func TestFreezeToString_MultipleBuffers(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	buf1 := rctx.getBuffer()
	*buf1 = append(*buf1, "First"...)
	rctx.freezeToString(buf1)

	buf2 := rctx.getBuffer()
	*buf2 = append(*buf2, "Second"...)
	rctx.freezeToString(buf2)

	assert.Len(t, rctx.frozenBuffers, 2)
}

func TestRenderASTToString(t *testing.T) {
	testCases := []struct {
		buildOpts    func() *RenderOrchestrator
		opts         func() RenderASTToStringOptions
		name         string
		wantExact    string
		wantContains []string
		useExact     bool
		wantErr      bool
	}{
		{
			name: "nil template returns empty string and no error",
			buildOpts: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: func() RenderASTToStringOptions {
				return RenderASTToStringOptions{
					Template: nil,
				}
			},
			wantExact: "",
			useExact:  true,
		},
		{
			name: "simple text node renders text content",
			buildOpts: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: func() RenderASTToStringOptions {
				return RenderASTToStringOptions{
					Template: &ast_domain.TemplateAST{
						RootNodes: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "Hello World",
							},
						},
					},
				}
			},
			wantExact: "Hello World",
			useExact:  true,
		},
		{
			name: "element node renders tag with content",
			buildOpts: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: func() RenderASTToStringOptions {
				return RenderASTToStringOptions{
					Template: &ast_domain.TemplateAST{
						RootNodes: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "div",
								Children: []*ast_domain.TemplateNode{
									{
										NodeType:    ast_domain.NodeText,
										TextContent: "hello",
									},
								},
							},
						},
					},
				}
			},
			wantExact: "<div>hello</div>",
			useExact:  true,
		},
		{
			name: "void element renders self-closing",
			buildOpts: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: func() RenderASTToStringOptions {
				return RenderASTToStringOptions{
					Template: &ast_domain.TemplateAST{
						RootNodes: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "br",
							},
						},
					},
				}
			},
			wantExact: "<br />",
			useExact:  true,
		},
		{
			name: "IncludeDocumentWrapper wraps output in full HTML document",
			buildOpts: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: func() RenderASTToStringOptions {
				return RenderASTToStringOptions{
					Template: &ast_domain.TemplateAST{
						RootNodes: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "page content",
							},
						},
					},
					IncludeDocumentWrapper: true,
				}
			},
			wantContains: []string{
				"<!DOCTYPE html>",
				"<html>",
				"<head>",
				"<body>",
				"page content",
				"</body>",
				"</html>",
			},
		},
		{
			name: "IncludeDocumentWrapper with metadata includes title and language",
			buildOpts: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: func() RenderASTToStringOptions {
				return RenderASTToStringOptions{
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
							Title:    "Test",
							Language: "en",
						},
					},
					IncludeDocumentWrapper: true,
				}
			},
			wantContains: []string{
				"<title>Test</title>",
				`lang="en"`,
			},
		},
		{
			name: "IncludeDocumentWrapper with styling includes style tag",
			buildOpts: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			opts: func() RenderASTToStringOptions {
				return RenderASTToStringOptions{
					Template: &ast_domain.TemplateAST{
						RootNodes: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "styled content",
							},
						},
					},
					Styling:                "body { color: red; }",
					IncludeDocumentWrapper: true,
				}
			},
			wantContains: []string{
				"<style>",
				"body { color: red; }",
				"</style>",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := tc.buildOpts()
			opts := tc.opts()

			result, err := ro.RenderASTToString(context.Background(), opts)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tc.useExact {
				assert.Equal(t, tc.wantExact, result)
			}

			for _, want := range tc.wantContains {
				assert.Contains(t, result, want)
			}
		})
	}
}

func TestRenderStaticNode(t *testing.T) {
	testCases := []struct {
		buildOrchestrator func() *RenderOrchestrator
		node              *ast_domain.TemplateNode
		name              string
		wantOutput        string
		wantContains      []string
		useExact          bool
		wantErr           bool
	}{
		{
			name: "text node renders text",
			buildOrchestrator: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "static text",
			},
			wantOutput: "static text",
			useExact:   true,
		},
		{
			name: "element with children renders full tag with content",
			buildOrchestrator: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "span",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "inner text",
					},
				},
			},
			wantOutput: "<span>inner text</span>",
			useExact:   true,
		},
		{
			name: "void element renders self-closing",
			buildOrchestrator: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "br",
			},
			wantOutput: "<br />",
			useExact:   true,
		},
		{
			name: "comment node renders HTML comment",
			buildOrchestrator: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeComment,
				TextContent: "a comment",
			},
			wantOutput: "<!--a comment-->",
			useExact:   true,
		},
		{
			name: "comment stripped when orchestrator built with WithStripHTMLComments",
			buildOrchestrator: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().WithStripHTMLComments(true).Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeComment,
				TextContent: "stripped comment",
			},
			wantOutput: "",
			useExact:   true,
		},
		{
			name: "fragment node renders children only without wrapper tag",
			buildOrchestrator: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeFragment,
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "first",
					},
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "second",
					},
				},
			},
			wantOutput: "firstsecond",
			useExact:   true,
		},
		{
			name: "nested tree renders correctly",
			buildOrchestrator: func() *RenderOrchestrator {
				return NewTestOrchestratorBuilder().Build()
			},
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "p",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "nested content",
							},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "br",
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "span",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "class", Value: "highlight"},
						},
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "styled",
							},
						},
					},
				},
			},
			wantContains: []string{
				"<div>",
				"<p>nested content</p>",
				"<br />",
				`<span class="highlight">styled</span>`,
				"</div>",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := tc.buildOrchestrator()

			result, err := ro.RenderStaticNode(tc.node)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tc.useExact {
				assert.Equal(t, tc.wantOutput, string(result))
			}

			for _, want := range tc.wantContains {
				assert.Contains(t, string(result), want)
			}
		})
	}
}

func TestWithStripHTMLComments(t *testing.T) {
	testCases := []struct {
		name     string
		strip    bool
		expected bool
	}{
		{name: "enabled strips comments", strip: true, expected: true},
		{name: "disabled keeps comments", strip: false, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := NewRenderOrchestrator(nil, nil, nil, nil, WithStripHTMLComments(tc.strip))
			assert.Equal(t, tc.expected, ro.stripHTMLComments)
		})
	}
}

func TestNewRenderContext(t *testing.T) {
	rctx := newRenderContext()
	require.NotNil(t, rctx)

	assert.NotNil(t, rctx.collectedCustomComponents)
	assert.NotNil(t, rctx.requiredSvgSymbols)
	assert.NotNil(t, rctx.customTags)
	assert.NotNil(t, rctx.mergedAttrsCache)
	assert.NotNil(t, rctx.registeredDynamicAssets)
	assert.NotNil(t, rctx.srcsetCache)
	assert.NotNil(t, rctx.linkHeaderSet)
	assert.NotNil(t, rctx.collectedLinkHeaders)
	assert.NotNil(t, rctx.frozenBuffers)

	assert.Nil(t, rctx.registry)
	assert.Nil(t, rctx.csrfService)
	assert.Nil(t, rctx.httpRequest)
	assert.Nil(t, rctx.originalCtx)
	assert.Nil(t, rctx.csrfPair)
	assert.Nil(t, rctx.csrfError)

	assert.Empty(t, rctx.pageID)
	assert.Empty(t, rctx.currentLocale)
	assert.Empty(t, rctx.i18nStrategy)
	assert.Empty(t, rctx.defaultLocale)
}

func TestGetBufferedWriter_RoundTrip(t *testing.T) {
	var buffer bytes.Buffer
	bw := getBufferedWriter(&buffer)
	require.NotNil(t, bw)

	_, err := bw.WriteString("hello")
	require.NoError(t, err)
	require.NoError(t, bw.Flush())

	releaseBufferedWriter(bw)
	assert.Equal(t, "hello", buffer.String())
}

func TestGetBufferedWriter_MultipleRoundTrips(t *testing.T) {

	var buf1 bytes.Buffer
	bw := getBufferedWriter(&buf1)
	_, err := bw.WriteString("first")
	require.NoError(t, err)
	require.NoError(t, bw.Flush())
	releaseBufferedWriter(bw)

	var buf2 bytes.Buffer
	bw2 := getBufferedWriter(&buf2)
	_, err = bw2.WriteString("second")
	require.NoError(t, err)
	require.NoError(t, bw2.Flush())
	releaseBufferedWriter(bw2)

	assert.Equal(t, "first", buf1.String())
	assert.Equal(t, "second", buf2.String())
}

func TestEnsureCSRFForMeta(t *testing.T) {
	testCases := []struct {
		buildContext   func() *renderContext
		name           string
		checkEphemeral string
		expectedCount  int
		expectNil      bool
	}{
		{
			name: "full deps returns valid CSRF pair",
			buildContext: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithCSRFService(newTestCSRFMock()).
					WithHTTPRequest(httptest.NewRequest(http.MethodGet, "/test", nil)).
					Build()
			},
			expectNil:      false,
			expectedCount:  1,
			checkEphemeral: "test-ephemeral-token",
		},
		{
			name: "nil CSRF service returns nil",
			buildContext: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithHTTPRequest(httptest.NewRequest(http.MethodGet, "/test", nil)).
					WithHTTPResponse(httptest.NewRecorder()).
					Build()
			},
			expectNil: true,
		},
		{
			name: "nil HTTP request returns nil",
			buildContext: func() *renderContext {
				rctx := NewTestRenderContextBuilder().
					WithCSRFService(newTestCSRFMock()).
					Build()

				rctx.httpRequest = nil
				return rctx
			},
			expectNil: true,
		},
		{
			name: "nil HTTP response returns nil",
			buildContext: func() *renderContext {
				rctx := NewTestRenderContextBuilder().
					WithCSRFService(newTestCSRFMock()).
					WithHTTPRequest(httptest.NewRequest(http.MethodGet, "/test", nil)).
					Build()

				rctx.httpResponse = nil
				return rctx
			},
			expectNil: true,
		},
		{
			name: "service returns error yields nil",
			buildContext: func() *renderContext {
				return NewTestRenderContextBuilder().
					WithCSRFService(newTestCSRFMockWithError(errors.New("csrf failure"))).
					WithHTTPRequest(httptest.NewRequest(http.MethodGet, "/test", nil)).
					Build()
			},
			expectNil:     true,
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := tc.buildContext()
			ro := NewTestOrchestratorBuilder().Build()

			pair := ro.ensureCSRFForMeta(rctx)

			if tc.expectNil {
				assert.Nil(t, pair)
			} else {
				require.NotNil(t, pair)
				if tc.checkEphemeral != "" {
					assert.Equal(t, tc.checkEphemeral, pair.RawEphemeralToken)
				}
			}

			if tc.expectedCount > 0 {
				if mock, ok := rctx.csrfService.(*security_domain.MockCSRFTokenService); ok {
					assert.Equal(t, int64(tc.expectedCount), atomic.LoadInt64(&mock.GenerateCSRFPairCallCount))
				}
			}
		})
	}
}

func TestEnsureCSRFForMeta_CalledTwiceOnlyGeneratesOnce(t *testing.T) {
	mockCSRF := newTestCSRFMock()
	rctx := NewTestRenderContextBuilder().
		WithCSRFService(mockCSRF).
		WithHTTPRequest(httptest.NewRequest(http.MethodGet, "/test", nil)).
		Build()
	ro := NewTestOrchestratorBuilder().Build()

	pair1 := ro.ensureCSRFForMeta(rctx)
	require.NotNil(t, pair1)

	pair2 := ro.ensureCSRFForMeta(rctx)
	require.NotNil(t, pair2)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mockCSRF.GenerateCSRFPairCallCount), "GenerateCSRFPair should only be called once due to sync.Once")
	assert.Equal(t, pair1.RawEphemeralToken, pair2.RawEphemeralToken)
}

func TestFilterValidJSON(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input returns nil",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "all valid JSON passes through",
			input:    []string{`{"@type":"Article","name":"Test"}`, `{"@type":"WebPage"}`},
			expected: []string{`{"@type":"Article","name":"Test"}`, `{"@type":"WebPage"}`},
		},
		{
			name:     "invalid JSON is filtered out",
			input:    []string{`{"valid":true}`, `{invalid json`, `["also","valid"]`},
			expected: []string{`{"valid":true}`, `["also","valid"]`},
		},
		{
			name:     "all invalid returns empty slice",
			input:    []string{`{bad`, `not json`, `<html>`},
			expected: []string{},
		},
		{
			name:     "valid JSON-LD with nested objects",
			input:    []string{`{"@context":"https://schema.org","@type":"Organization","name":"Example","address":{"@type":"PostalAddress","addressLocality":"London"}}`},
			expected: []string{`{"@context":"https://schema.org","@type":"Organization","name":"Example","address":{"@type":"PostalAddress","addressLocality":"London"}}`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterValidJSON(context.Background(), tc.input)

			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}

			require.Len(t, result, len(tc.expected))
			for i, expected := range tc.expected {
				assert.Equal(t, expected, result[i])
			}
		})
	}
}
