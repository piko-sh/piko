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
)

// directiveSlot names one directive field, and says how to read it from the buffer and
// where to put it on the node.
type directiveSlot struct {
	// fetch reads the serialised directive out of the node buffer.
	fetch func(*ast_schema_gen.TemplateNodeFB, *ast_schema_gen.DirectiveFB) *ast_schema_gen.DirectiveFB

	// target points at the node field that receives the directive.
	target func(*ast_domain.TemplateNode) **ast_domain.Directive

	// name identifies the field when a decode fails.
	name string
}

var (
	// directiveSlots lists every directive stored as its own field on a template node. The
	// control-flow directives come first, then the ones that bind data or behaviour.
	directiveSlots = []directiveSlot{
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveIf,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirIf },
			name:   "DirIf",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveElseIf,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirElseIf },
			name:   "DirElseIf",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveElse,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirElse },
			name:   "DirElse",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveFor,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirFor },
			name:   "DirFor",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveShow,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirShow },
			name:   "DirShow",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveKey,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirKey },
			name:   "DirKey",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveMemo,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirMemo },
			name:   "DirMemo",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveContext,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirContext },
			name:   "DirContext",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveModel,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirModel },
			name:   "DirModel",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveRef,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirRef },
			name:   "DirRef",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveSlot,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirSlot },
			name:   "DirSlot",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveClass,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirClass },
			name:   "DirClass",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveStyle,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirStyle },
			name:   "DirStyle",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveText,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirText },
			name:   "DirText",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveHtml,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirHTML },
			name:   "DirHTML",
		},
		{
			fetch:  (*ast_schema_gen.TemplateNodeFB).DirectiveScaffold,
			target: func(node *ast_domain.TemplateNode) **ast_domain.Directive { return &node.DirScaffold },
			name:   "DirScaffold",
		},
	}
)
