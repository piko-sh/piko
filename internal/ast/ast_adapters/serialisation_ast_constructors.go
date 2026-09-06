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

package ast_adapters

import (
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/ast/ast_schema/ast_schema_gen"
	"piko.sh/piko/internal/mem"
)

// newDecodedTemplateNode builds a template node from the scalar fields of a FlatBuffer
// node, leaving ranges, vectors, directives and annotations for the decoder to fill.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which is the FlatBuffer node to read.
//
// Returns *ast_domain.TemplateNode which holds the decoded scalar fields.
func newDecodedTemplateNode(fb *ast_schema_gen.TemplateNodeFB) *ast_domain.TemplateNode {
	node := ast_domain.TemplateNode{}
	node.NodeType = ast_domain.NodeType(fb.NodeType())
	node.TagName = mem.String(fb.TagName())
	node.TextContent = mem.String(fb.TextContent())
	node.InnerHTML = mem.String(fb.InnerHtml())
	node.IsContentEditable = fb.IsContentEditable()
	node.PreserveWhitespace = fb.PreserveWhitespace()
	node.PreferredFormat = ast_domain.FormatHint(fb.PreferredFormat())
	node.PrerenderedHTML = fb.PrerenderedHtmlBytes()
	return &node
}

// newDecodedDirective builds a directive from the scalar fields of a FlatBuffer
// directive, leaving expressions, annotations and locations for the decoder to fill.
//
// Takes fb (*ast_schema_gen.DirectiveFB) which is the FlatBuffer directive to read.
// Takes directiveType (ast_domain.DirectiveType) which is the already mapped directive
// type.
//
// Returns *ast_domain.Directive which holds the decoded scalar fields.
func newDecodedDirective(fb *ast_schema_gen.DirectiveFB, directiveType ast_domain.DirectiveType) *ast_domain.Directive {
	return &ast_domain.Directive{
		Type:           directiveType,
		Arg:            mem.String(fb.Argument()),
		Modifier:       mem.String(fb.Modifier()),
		RawExpression:  mem.String(fb.RawExpression()),
		IsStaticEvent:  fb.IsStaticEvent(),
		Expression:     nil,
		ChainKey:       nil,
		GoAnnotations:  nil,
		EventModifiers: nil,
		Location:       ast_domain.Location{},
		NameLocation:   ast_domain.Location{},
		AttributeRange: ast_domain.Range{},
	}
}
