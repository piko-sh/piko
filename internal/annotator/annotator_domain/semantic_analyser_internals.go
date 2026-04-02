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

package annotator_domain

// Provides internal helper functions for semantic analysis including text
// content validation and node processing. Contains utility methods used by the
// semantic analyser to validate template structure and content during AST
// traversal.

import (
	"context"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// InternalsAnalyser handles the analysis of expressions internal to a
// component. These are the "inner" parts of a node: text interpolations,
// HTML content directives, and similar constructs.
type InternalsAnalyser struct {
	// resolver looks up types for expressions during analysis.
	resolver *TypeResolver
}

// AnalyseInternalExpressions analyses all expressions that are internal to a
// component's template scope. This includes {{...}} interpolations in rich text
// and directives like p-text and p-html that render content from the
// component's state.
//
// Takes node (*ast_domain.TemplateNode) which is the template node
// whose internal expressions are to be analysed.
// Takes ctx (*AnalysisContext) which provides the analysis state, symbol
// table, and diagnostic collector.
func (ia *InternalsAnalyser) AnalyseInternalExpressions(
	goCtx context.Context,
	node *ast_domain.TemplateNode,
	ctx *AnalysisContext,
	_ *ast_domain.PartialInvocationInfo,
) {
	if node == nil {
		return
	}

	resolveAndValidate(goCtx, node.DirText, ctx, ia.resolver, validateTextContentDirective)
	resolveAndValidate(goCtx, node.DirHTML, ctx, ia.resolver, validateTextContentDirective)

	for i := range node.RichText {
		part := &node.RichText[i]
		if !part.IsLiteral {
			if containsEventPlaceholder(part.Expression) {
				ctx.addDiagnostic(
					ast_domain.Error,
					"$event can only be used in p-on or p-event handlers",
					part.RawExpression,
					part.Location,
					part.GoAnnotations,
					annotator_dto.CodeEventPlaceholderMisuse,
				)
				continue
			}
			part.GoAnnotations = ia.resolver.Resolve(goCtx, ctx, part.Expression, part.Location)
			if part.GoAnnotations != nil && part.GoAnnotations.OriginalSourcePath == nil {
				part.GoAnnotations.OriginalSourcePath = &ctx.SFCSourcePath
			}
		}
	}
}

// newInternalsAnalyser creates a new InternalsAnalyser.
//
// Takes resolver (*TypeResolver) which resolves types during analysis.
//
// Returns *InternalsAnalyser which is ready to analyse internal structures.
func newInternalsAnalyser(resolver *TypeResolver) *InternalsAnalyser {
	return &InternalsAnalyser{resolver: resolver}
}

// validateTextContentDirective checks that text content directives (p-text,
// p-html) do not use $event.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes ctx (*AnalysisContext) which collects any errors found.
func validateTextContentDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	rejectEventPlaceholderInDirective(d, ctx)
	rejectFormPlaceholderInDirective(d, ctx)
}
