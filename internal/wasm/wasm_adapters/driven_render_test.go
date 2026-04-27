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

package wasm_adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/internal/wasm/wasm_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

type captureHeadlessRenderer struct {
	lastOptions wasm_domain.HeadlessRenderOptions
	html        string
	err         error
}

func (c *captureHeadlessRenderer) RenderASTToString(_ context.Context, options wasm_domain.HeadlessRenderOptions) (string, error) {
	c.lastOptions = options
	return c.html, c.err
}

func TestRenderAdapter_RenderFromAST_OmitsDocumentWrapper(t *testing.T) {
	t.Parallel()

	capture := &captureHeadlessRenderer{html: "<p>x</p>"}
	adapter := NewRenderAdapter(WithHeadlessRenderer(capture))

	response, err := adapter.RenderFromAST(context.Background(), &wasm_dto.RenderFromASTRequest{
		AST:      &ast_domain.TemplateAST{},
		Metadata: &templater_dto.InternalMetadata{},
		CSS:      "p { color: red; }",
	})
	require.NoError(t, err)
	require.True(t, response.Success)
	assert.False(t, capture.lastOptions.IncludeDocumentWrapper,
		"dynamic-render contract requires the body-only option")
	assert.Equal(t, "p { color: red; }", capture.lastOptions.Styling,
		"CSS must reach the headless renderer for AST-aware styling logic")
	assert.Equal(t, "<p>x</p>", response.HTML)
	assert.Equal(t, "p { color: red; }", response.CSS, "response CSS echoes the request")
}

func TestRenderAdapter_RenderFromAST_NilASTReturnsBlankHTML(t *testing.T) {
	t.Parallel()

	adapter := NewRenderAdapter()

	response, err := adapter.RenderFromAST(context.Background(), &wasm_dto.RenderFromASTRequest{
		AST: nil, CSS: "x",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Empty(t, response.HTML)
	assert.Equal(t, "x", response.CSS)
}

func TestRenderAdapter_RenderFromAST_NoRendererConfigured(t *testing.T) {
	t.Parallel()

	adapter := NewRenderAdapter()
	response, err := adapter.RenderFromAST(context.Background(), &wasm_dto.RenderFromASTRequest{
		AST: &ast_domain.TemplateAST{},
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "headless renderer not configured")
}
