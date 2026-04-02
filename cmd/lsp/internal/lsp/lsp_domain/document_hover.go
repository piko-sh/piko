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

package lsp_domain

import (
	"context"

	"go.lsp.dev/protocol"
)

// GetHoverInfo finds the AST node at the given position and returns hover
// information for the symbol at that location.
//
// It uses the AnalysisMap to get the semantic context at that location. It also
// provides PK-specific hover information for handlers, partials, and refs.
//
// Takes position (protocol.Position) which specifies the cursor location in the
// document.
//
// Returns *protocol.Hover which contains the formatted hover contents and
// range, or nil if no hover information is available.
// Returns error when hover lookup fails.
func (d *document) GetHoverInfo(ctx context.Context, position protocol.Position) (*protocol.Hover, error) {
	if pkHover, err := d.GetPKHoverInfo(ctx, position); err == nil && pkHover != nil {
		return pkHover, nil
	}

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return nil, nil
	}

	findResult := findExpressionAtPositionWithContext(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if findResult.bestMatch == nil {
		return nil, nil
	}

	hoverContents := d.formatHoverContentsEnhanced(ctx, findResult.bestMatch, position, findResult.memberContext)
	if hoverContents == "" {
		return nil, nil
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: hoverContents,
		},
		Range: &findResult.bestRange,
	}, nil
}
