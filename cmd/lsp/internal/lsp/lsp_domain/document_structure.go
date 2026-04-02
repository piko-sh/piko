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
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/safeconv"
)

// symbolReference identifies a symbol by its definition location in the source.
type symbolReference struct {
	// sourcePath is the file path where this symbol reference was found.
	sourcePath string

	// line is the 1-based line number where the symbol is defined.
	line int

	// column is the 1-based column position of the symbol in the source line.
	column int
}

// GetDocumentHighlights finds all uses of the symbol at the given cursor
// position within the current document. This shows the user where a variable
// or function is used in the file.
//
// Takes position (protocol.Position) which specifies the cursor location to find
// symbol uses for.
//
// Returns []protocol.DocumentHighlight which contains all locations where the
// symbol at the cursor position is used in the document.
// Returns error when the highlight operation fails.
func (d *document) GetDocumentHighlights(ctx context.Context, position protocol.Position) ([]protocol.DocumentHighlight, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return []protocol.DocumentHighlight{}, nil
	}

	targetRef := d.getSymbolReferenceAtPosition(ctx, position)
	if targetRef == nil {
		return []protocol.DocumentHighlight{}, nil
	}

	expressionRangeMap := buildExpressionRangeMap(d.AnnotationResult.AnnotatedAST, d.URI.Filename())
	return d.collectHighlights(targetRef, expressionRangeMap), nil
}

// getSymbolReferenceAtPosition finds the symbol reference at the cursor
// position.
//
// Takes position (protocol.Position) which specifies the cursor
// location to search.
//
// Returns *symbolReference which contains the location of the referenced
// symbol, or nil if no valid symbol reference exists at the position.
func (d *document) getSymbolReferenceAtPosition(ctx context.Context, position protocol.Position) *symbolReference {
	targetExpr, _ := findExpressionAtPosition(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetExpr == nil {
		return nil
	}

	targetAnn := targetExpr.GetGoAnnotation()
	if targetAnn == nil || targetAnn.Symbol == nil {
		return nil
	}

	defLocation := targetAnn.Symbol.ReferenceLocation
	if defLocation.IsSynthetic() {
		return nil
	}

	return &symbolReference{
		line:       defLocation.Line,
		column:     defLocation.Column,
		sourcePath: getSourcePath(targetAnn),
	}
}

// collectHighlights gathers all expressions that refer to the same symbol.
//
// Takes targetRef (*symbolReference) which identifies the symbol to find.
// Takes expressionRangeMap (map[ast_domain.Expression]protocol.Range) which maps
// expressions to their source locations.
//
// Returns []protocol.DocumentHighlight which contains all matching highlights.
func (d *document) collectHighlights(targetRef *symbolReference, expressionRangeMap map[ast_domain.Expression]protocol.Range) []protocol.DocumentHighlight {
	var highlights []protocol.DocumentHighlight

	d.AnnotationResult.AnnotatedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		ast_domain.WalkNodeExpressions(node, func(expression ast_domain.Expression) {
			if highlight := matchSymbolHighlight(expression, targetRef, expressionRangeMap); highlight != nil {
				highlights = append(highlights, *highlight)
			}
		})
		return true
	})

	return highlights
}

// GetFoldingRanges provides code folding regions for the document based on
// its structure, letting users collapse sections like elements, <template>,
// <script>, and <style> blocks.
//
// Returns []protocol.FoldingRange which contains the foldable regions.
// Returns error when the folding ranges cannot be computed.
func (d *document) GetFoldingRanges() ([]protocol.FoldingRange, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return []protocol.FoldingRange{}, nil
	}

	ranges := []protocol.FoldingRange{}

	d.AnnotationResult.AnnotatedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement {
			return true
		}

		if !node.ClosingTagRange.Start.IsSynthetic() {
			contentStartLine := node.OpeningTagRange.End.Line
			contentEndLine := node.ClosingTagRange.Start.Line

			if contentEndLine > contentStartLine {
				ranges = append(ranges, protocol.FoldingRange{
					StartLine: safeconv.IntToUint32(contentStartLine - 1),
					EndLine:   safeconv.IntToUint32(contentEndLine - 1),
					Kind:      protocol.RegionFoldingRange,
				})
			}
		}

		blockStartLine := node.NodeRange.Start.Line
		blockEndLine := node.NodeRange.End.Line

		if blockEndLine > blockStartLine {
			kind := protocol.RegionFoldingRange
			if node.TagName == "template" || node.TagName == "script" || node.TagName == "style" || node.TagName == "i18n" {
				kind = protocol.RegionFoldingRange
			}

			ranges = append(ranges, protocol.FoldingRange{
				StartLine: safeconv.IntToUint32(blockStartLine - 1),
				EndLine:   safeconv.IntToUint32(blockEndLine - 1),
				Kind:      kind,
			})
		}

		return true
	})

	return ranges, nil
}

// PrepareRename checks if the symbol at the cursor position is valid for
// renaming and returns its current range.
//
// Takes position (protocol.Position) which specifies the cursor location to check.
//
// Returns *protocol.Range which contains the symbol's range, or nil if the
// symbol cannot be renamed.
// Returns error when the operation fails.
func (d *document) PrepareRename(ctx context.Context, position protocol.Position) (*protocol.Range, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return nil, nil
	}

	targetExpr, targetRange := findExpressionAtPosition(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetExpr == nil {
		return nil, nil
	}

	ann := targetExpr.GetGoAnnotation()
	if ann == nil || ann.Symbol == nil {
		return nil, nil
	}

	symbolName := ann.Symbol.Name
	if symbolName == "" {
		return nil, nil
	}

	builtInNames := map[string]bool{
		"len": true, "cap": true, "append": true, "min": true, "max": true,
		"make": true, "new": true, "copy": true, "delete": true,
		"panic": true, "recover": true, "print": true, "println": true,
	}

	if builtInNames[symbolName] {
		return nil, nil
	}

	return &targetRange, nil
}

// GetDocumentSymbols extracts a hierarchical outline of the document's
// structure. This provides the document outline view in the IDE, showing
// elements with IDs, partial invocations, and other significant structural
// elements.
//
// Returns []any which contains the document symbols as protocol.DocumentSymbol
// values.
// Returns error when symbol extraction fails.
func (d *document) GetDocumentSymbols() ([]any, error) {
	if d.isPKCFile() {
		return d.getPKCDocumentSymbols()
	}

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return []any{}, nil
	}

	var buildSymbols func(node *ast_domain.TemplateNode) []protocol.DocumentSymbol
	buildSymbols = func(node *ast_domain.TemplateNode) []protocol.DocumentSymbol {
		symbols := []protocol.DocumentSymbol{}

		for _, child := range node.Children {
			symbol := d.extractSymbolFromNode(child)
			if symbol != nil {
				symbol.Children = buildSymbols(child)
				symbols = append(symbols, *symbol)
			}
		}

		return symbols
	}

	allSymbols := []protocol.DocumentSymbol{}
	for _, rootNode := range d.AnnotationResult.AnnotatedAST.RootNodes {
		rootSymbols := buildSymbols(rootNode)
		allSymbols = append(allSymbols, rootSymbols...)
	}

	result := make([]any, len(allSymbols))
	for i := range allSymbols {
		result[i] = allSymbols[i]
	}

	return result, nil
}

// symbolInfo holds details about a symbol taken from a syntax node.
type symbolInfo struct {
	// name is the symbol's name for display in the document outline.
	name string

	// detail provides extra context for the symbol, such as its type signature.
	detail string

	// kind specifies the LSP symbol kind for the document symbol.
	kind protocol.SymbolKind
}

// extractSymbolFromNode checks if a node should be represented as a symbol in
// the outline.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns *protocol.DocumentSymbol which is the symbol representation, or nil
// if the node should not appear in the outline.
func (*document) extractSymbolFromNode(node *ast_domain.TemplateNode) *protocol.DocumentSymbol {
	if node == nil || node.Location.Line == 0 || node.NodeType != ast_domain.NodeElement {
		return nil
	}

	info := extractSymbolInfo(node)
	if info == nil {
		return nil
	}

	return &protocol.DocumentSymbol{
		Name:           info.name,
		Detail:         info.detail,
		Kind:           info.kind,
		Range:          nodeRangeToLSP(node.NodeRange),
		SelectionRange: nodeRangeToLSP(node.OpeningTagRange),
	}
}

// structuralBlockSymbols maps tag names to symbol information for top-level SFC
// blocks.
var structuralBlockSymbols = map[string]symbolInfo{
	"template": {name: "<template>", kind: protocol.SymbolKindNamespace, detail: "template block"},
	"script":   {name: "<script>", kind: protocol.SymbolKindNamespace, detail: "script block"},
	"style":    {name: "<style>", kind: protocol.SymbolKindNamespace, detail: "style block"},
	"i18n":     {name: "<i18n>", kind: protocol.SymbolKindNamespace, detail: "i18n block"},
}

// getSourcePath extracts the source path from an annotation.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which contains the annotation
// to extract the path from.
//
// Returns string which is the original source path, or empty if none is set.
func getSourcePath(ann *ast_domain.GoGeneratorAnnotation) string {
	if ann.OriginalSourcePath != nil {
		return *ann.OriginalSourcePath
	}
	return ""
}

// matchSymbolHighlight checks if an expression refers to the
// target symbol and returns a highlight for it.
//
// Takes expression (ast_domain.Expression) which is the
// expression to check.
// Takes targetRef (*symbolReference) which is the symbol to
// match against.
// Takes expressionRangeMap
// (map[ast_domain.Expression]protocol.Range) which maps
// expressions to their source ranges.
//
// Returns *protocol.DocumentHighlight which holds the
// highlight range, or nil if the expression does not refer
// to the target symbol.
func matchSymbolHighlight(expression ast_domain.Expression, targetRef *symbolReference, expressionRangeMap map[ast_domain.Expression]protocol.Range) *protocol.DocumentHighlight {
	if expression == nil {
		return nil
	}

	ann := expression.GetGoAnnotation()
	if ann == nil || ann.Symbol == nil {
		return nil
	}

	defLocation := ann.Symbol.ReferenceLocation
	if defLocation.IsSynthetic() {
		return nil
	}

	if !isSameSymbol(defLocation, getSourcePath(ann), targetRef) {
		return nil
	}

	expressionRange, ok := expressionRangeMap[expression]
	if !ok {
		return nil
	}

	return &protocol.DocumentHighlight{
		Range: expressionRange,
		Kind:  protocol.DocumentHighlightKindText,
	}
}

// isSameSymbol checks if a location refers to the same symbol as the target.
//
// Takes defLocation (ast_domain.Location) which is the
// definition location to check.
// Takes sourcePath (string) which is the file path of the
// definition.
// Takes targetRef (*symbolReference) which is the reference to compare against.
//
// Returns bool which is true if the line, column, and path all match.
func isSameSymbol(defLocation ast_domain.Location, sourcePath string, targetRef *symbolReference) bool {
	return defLocation.Line == targetRef.line &&
		defLocation.Column == targetRef.column &&
		sourcePath == targetRef.sourcePath
}

// extractSymbolInfo gets symbol information from a node by trying several
// methods in turn.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns *symbolInfo which holds the symbol details, or nil if no symbol was
// found.
func extractSymbolInfo(node *ast_domain.TemplateNode) *symbolInfo {
	if info := extractIDSymbol(node); info != nil {
		return info
	}
	if info := extractIsAttrSymbol(node); info != nil {
		return info
	}
	if info := extractPartialInfoSymbol(node); info != nil {
		return info
	}
	return extractStructuralBlockSymbol(node)
}

// extractIDSymbol extracts symbol info from an element with an ID attribute.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns *symbolInfo which contains the ID value as name and tag as detail,
// or nil if the node has no ID attribute.
func extractIDSymbol(node *ast_domain.TemplateNode) *symbolInfo {
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if attr.Name == "id" {
			return &symbolInfo{
				name:   attr.Value,
				kind:   protocol.SymbolKindClass,
				detail: node.TagName,
			}
		}
	}
	return nil
}

// extractIsAttrSymbol extracts symbol info from a partial reference
// (is="..." attribute).
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns *symbolInfo which holds the partial reference info, or nil if the
// node has no "is" attribute.
func extractIsAttrSymbol(node *ast_domain.TemplateNode) *symbolInfo {
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if attr.Name == "is" {
			return &symbolInfo{
				name:   attr.Value,
				kind:   protocol.SymbolKindModule,
				detail: "partial: " + node.TagName,
			}
		}
	}
	return nil
}

// extractPartialInfoSymbol extracts symbol details from GoAnnotations partial
// info.
//
// Takes node (*ast_domain.TemplateNode) which contains the template node with
// possible partial info annotations.
//
// Returns *symbolInfo which contains the extracted symbol details, or nil if
// the node has no partial info or no valid name.
func extractPartialInfoSymbol(node *ast_domain.TemplateNode) *symbolInfo {
	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return nil
	}

	partialInfo := node.GoAnnotations.PartialInfo
	name := partialInfo.PartialAlias
	if name == "" {
		name = partialInfo.InvocationKey
	}
	if name == "" {
		return nil
	}

	return &symbolInfo{
		name:   name,
		kind:   protocol.SymbolKindModule,
		detail: "partial",
	}
}

// extractStructuralBlockSymbol gets symbol info for top-level structural
// blocks.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to get
// symbol info from.
//
// Returns *symbolInfo which holds the symbol info if the node's tag name
// matches a known structural block, or nil if there is no match.
func extractStructuralBlockSymbol(node *ast_domain.TemplateNode) *symbolInfo {
	if info, ok := structuralBlockSymbols[node.TagName]; ok {
		return &info
	}
	return nil
}

// nodeRangeToLSP converts an AST range to an LSP range.
//
// Takes r (ast_domain.Range) which specifies the source code position range.
//
// Returns protocol.Range which is the LSP-compatible range with zero-based
// line and column indices.
func nodeRangeToLSP(r ast_domain.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      safeconv.IntToUint32(r.Start.Line - 1),
			Character: safeconv.IntToUint32(r.Start.Column - 1),
		},
		End: protocol.Position{
			Line:      safeconv.IntToUint32(r.End.Line - 1),
			Character: safeconv.IntToUint32(r.End.Column - 1),
		},
	}
}
