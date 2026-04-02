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
	"go/printer"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// printerTabWidth is the tab width used when formatting type expressions.
const printerTabWidth = 4

// GetInlayHints returns inlay hints for the given range.
// Inlay hints show type annotations and parameter names inline in the editor.
//
// Takes textRange (protocol.Range) which specifies the visible
// range to get hints for.
//
// Returns []InlayHint which contains the hints to display.
// Returns error when hint generation fails.
func (d *document) GetInlayHints(ctx context.Context, textRange protocol.Range) ([]InlayHint, error) {
	_, l := logger_domain.From(ctx, log)

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil || d.AnalysisMap == nil {
		l.Debug("GetInlayHints: No analysis data available")
		return []InlayHint{}, nil
	}

	var hints []InlayHint

	for node, analysisCtx := range d.AnalysisMap {
		if analysisCtx == nil || analysisCtx.Symbols == nil {
			continue
		}

		nodeHints := d.collectForLoopTypeHints(node, analysisCtx, textRange)
		hints = append(hints, nodeHints...)
	}

	l.Debug("GetInlayHints: Generated hints",
		logger_domain.Int("count", len(hints)))

	return hints, nil
}

// collectForLoopTypeHints collects type hints for p-for loop variables.
//
// Takes node (*ast_domain.TemplateNode) which contains the for loop to analyse.
// Takes analysisCtx (*annotator_domain.AnalysisContext) which provides symbol
// information for type resolution.
// Takes textRange (protocol.Range) which limits hints to the visible range.
//
// Returns []InlayHint which contains type hints for each loop variable.
func (d *document) collectForLoopTypeHints(
	node *ast_domain.TemplateNode,
	analysisCtx *annotator_domain.AnalysisContext,
	textRange protocol.Range,
) []InlayHint {
	if node == nil || node.DirFor == nil {
		return nil
	}

	directive := node.DirFor

	loopVars := extractLoopVariables(directive)
	hints := make([]InlayHint, 0, len(loopVars))

	for _, varName := range loopVars {
		symbol, found := analysisCtx.Symbols.Find(varName)
		if !found || symbol.TypeInfo == nil || symbol.TypeInfo.TypeExpression == nil {
			continue
		}

		position := getDirectiveHintPosition(directive, varName, d.URI.Filename())
		if !inlayHintPositionInRange(position, textRange) {
			continue
		}

		typeString := formatInlayTypeExpr(symbol.TypeInfo)
		if typeString == "" {
			continue
		}

		hints = append(hints, InlayHint{
			Position:     position,
			Label:        ": " + typeString,
			Kind:         InlayHintKindType,
			PaddingLeft:  false,
			PaddingRight: true,
		})
	}

	return hints
}

// extractLoopVariables extracts variable names from a p-for directive.
//
// Takes directive (*ast_domain.Directive) which contains the loop expression
// to parse.
//
// Returns []string which contains the extracted variable names, excluding
// blank identifiers. Returns nil when the directive is nil, has no expression,
// or has no valid assignment.
func extractLoopVariables(directive *ast_domain.Directive) []string {
	if directive == nil || directive.RawExpression == "" {
		return nil
	}

	expression := directive.RawExpression
	varsPart, _, found := strings.Cut(expression, ":=")
	if !found {
		return nil
	}

	varsPart = strings.TrimSpace(varsPart)
	if varsPart == "" {
		return nil
	}

	parts := strings.Split(varsPart, ",")
	var vars []string
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v != "" && v != "_" {
			vars = append(vars, v)
		}
	}

	return vars
}

// getDirectiveHintPosition returns the position where a type hint should appear
// for a loop variable in a p-for directive.
//
// Takes directive (*ast_domain.Directive) which is the p-for directive
// containing the loop variable.
// Takes varName (string) which is the name of the loop variable to
// locate.
//
// Returns protocol.Position which is the position immediately after the
// variable name where the type hint should be inserted. Returns a zero
// position if the directive is nil or the variable is not found.
func getDirectiveHintPosition(directive *ast_domain.Directive, varName string, _ string) protocol.Position {
	if directive == nil {
		return protocol.Position{}
	}

	location := directive.Location
	if location.Line == 0 {
		return protocol.Position{}
	}

	expression := directive.RawExpression
	index := strings.Index(expression, varName)
	if index < 0 {
		return protocol.Position{}
	}

	return protocol.Position{
		Line:      safeconv.IntToUint32(location.Line - 1),
		Character: safeconv.IntToUint32(location.Column - 1 + index + len(varName)),
	}
}

// formatInlayTypeExpr formats a ResolvedTypeInfo for display as a hint.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type information to format.
//
// Returns string which contains the formatted type expression, or an empty
// string if the input is nil or formatting fails.
func formatInlayTypeExpr(typeInfo *ast_domain.ResolvedTypeInfo) string {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return ""
	}

	var buffer strings.Builder
	printerConfig := printer.Config{Mode: printer.UseSpaces, Tabwidth: printerTabWidth}
	if err := printerConfig.Fprint(&buffer, nil, typeInfo.TypeExpression); err != nil {
		return ""
	}

	return buffer.String()
}

// inlayHintPositionInRange checks if a position is within the given range.
//
// Takes position (protocol.Position) which specifies the position to check.
// Takes textRange (protocol.Range) which defines the range boundaries.
//
// Returns bool which is true if the position falls within the range.
func inlayHintPositionInRange(position protocol.Position, textRange protocol.Range) bool {
	if position.Line < textRange.Start.Line {
		return false
	}
	if position.Line == textRange.Start.Line && position.Character < textRange.Start.Character {
		return false
	}

	if position.Line > textRange.End.Line {
		return false
	}
	if position.Line == textRange.End.Line && position.Character > textRange.End.Character {
		return false
	}

	return true
}
