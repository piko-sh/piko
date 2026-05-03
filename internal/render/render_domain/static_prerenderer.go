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
	"bufio"
	"bytes"
	"context"
	"fmt"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
)

// RenderStaticNode renders a fully-static template node subtree to HTML bytes.
// Implements generator_domain.StaticPrerenderer for generation-time use,
// enabling precomputation of HTML for static subtrees.
//
// The node must have IsFullyPrerenderable=true, meaning its entire subtree
// contains no piko:svg, piko:img, piko:a, or piko:video tags that require
// runtime processing.
//
// Uses a minimal render context with no registry, CSRF service, or HTTP
// request/response, since those are not needed for static content.
//
// Takes node (*ast_domain.TemplateNode) which is the root of the static
// subtree to render.
//
// Returns []byte which contains the rendered HTML.
// Returns error when rendering fails or the buffer cannot be flushed.
func (ro *RenderOrchestrator) RenderStaticNode(node *ast_domain.TemplateNode) ([]byte, error) {
	var buffer bytes.Buffer
	const staticNodeBufferSize = 256
	buffer.Grow(staticNodeBufferSize)

	bufWriter := bufio.NewWriter(&buffer)
	qw := qt.AcquireWriter(bufWriter)
	defer qt.ReleaseWriter(qw)

	rctx := &renderContext{
		originalCtx:       context.Background(),
		stripHTMLComments: ro.stripHTMLComments,
	}

	if err := ro.renderNode(context.Background(), node, qw, rctx); err != nil {
		return nil, fmt.Errorf("rendering static node: %w", err)
	}

	if err := bufWriter.Flush(); err != nil {
		return nil, fmt.Errorf("flushing static node buffer: %w", err)
	}

	return buffer.Bytes(), nil
}
