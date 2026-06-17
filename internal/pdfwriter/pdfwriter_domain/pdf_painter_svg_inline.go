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
	"html"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxInlineSVGDepth bounds the recursion depth when serialising an inline <svg> subtree.
	//
	// A maliciously or accidentally deeply nested template would otherwise overflow the
	// stack; past this depth the subtree is truncated and a warning is logged. The limit is
	// high enough that any genuine SVG stays well within it.
	maxInlineSVGDepth = 256
)

// serialiseInlineSVG serialises an inline <svg> AST subtree back into SVG XML so it can
// be handed to the SVG writer for native vector painting.
//
// An xmlns is added to the root <svg> when missing. Returns "" for non-svg nodes.
//
// Takes ctx (context.Context) which carries the logger used to warn on truncation.
// Takes node (*ast_domain.TemplateNode) which is the inline <svg> subtree root.
//
// Returns string which holds the serialised SVG markup, or "" for non-svg nodes.
func serialiseInlineSVG(ctx context.Context, node *ast_domain.TemplateNode) string {
	if node == nil || node.TagName != "svg" {
		return ""
	}
	var sb strings.Builder
	writeSVGNode(ctx, &sb, node, true, 0)
	return sb.String()
}

// writeSVGNode recursively serialises an SVG element subtree, stopping past
// maxInlineSVGDepth. On early return the element is still closed so the emitted markup
// stays well formed.
//
// Takes ctx (context.Context) which carries the logger used to warn on truncation.
// Takes sb (*strings.Builder) which receives the serialised markup.
// Takes node (*ast_domain.TemplateNode) which is the node to serialise.
// Takes isRoot (bool) which marks the root <svg> element for xmlns injection.
// Takes depth (int) which is the current recursion depth.
func writeSVGNode(ctx context.Context, sb *strings.Builder, node *ast_domain.TemplateNode, isRoot bool, depth int) {
	if node == nil {
		return
	}
	if node.NodeType == ast_domain.NodeText {
		sb.WriteString(html.EscapeString(node.TextContent))
		return
	}
	if node.NodeType != ast_domain.NodeElement || node.TagName == "" {
		return
	}

	sb.WriteString("<")
	sb.WriteString(node.TagName)

	hasXmlns := false
	for i := range node.Attributes {
		name := node.Attributes[i].Name
		if name == "xmlns" {
			hasXmlns = true
		}
		sb.WriteString(" ")
		sb.WriteString(name)
		sb.WriteString(`="`)
		sb.WriteString(html.EscapeString(node.Attributes[i].Value))
		sb.WriteString(`"`)
	}
	if isRoot && !hasXmlns {
		sb.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
	}
	sb.WriteString(">")

	if depth >= maxInlineSVGDepth {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Inline SVG nesting exceeds maximum depth; truncating subtree",
			logger_domain.Int("max_depth", maxInlineSVGDepth))
	} else {
		for _, child := range node.Children {
			writeSVGNode(ctx, sb, child, false, depth+1)
		}
	}

	sb.WriteString("</")
	sb.WriteString(node.TagName)
	sb.WriteString(">")
}
