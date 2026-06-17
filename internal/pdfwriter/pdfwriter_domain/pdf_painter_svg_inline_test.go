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

package pdfwriter_domain

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestSerialiseInlineSVG_DeepNestingTerminates(t *testing.T) {

	root := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "svg"}
	current := root
	const nesting = maxInlineSVGDepth + 64
	for range nesting {
		child := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "g"}
		current.Children = []*ast_domain.TemplateNode{child}
		current = child
	}

	markup := serialiseInlineSVG(context.Background(), root)

	require.NotEmpty(t, markup, "expected non-empty markup for deeply nested inline svg")
	assert.True(t, strings.HasPrefix(markup, "<svg"), "expected serialised markup to open with <svg, got %q", markup[:min(16, len(markup))])
	assert.True(t, strings.HasSuffix(markup, "</svg>"), "expected serialised markup to close with </svg>, got tail %q", markup[max(0, len(markup)-16):])

	assert.Equal(t, strings.Count(markup, "<g"), strings.Count(markup, "</g>"), "unbalanced <g> tags")
}
