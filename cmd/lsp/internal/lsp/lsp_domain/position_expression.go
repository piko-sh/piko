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
	goast "go/ast"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// maxRangeValue is a large value used to represent an unbounded range
	// endpoint. Used when initialising a "worst case" range that any valid range
	// will be smaller than.
	maxRangeValue = 999999

	// lineSizeMultiplier is used in range size calculations to weight line
	// differences more heavily than character differences, ensuring multi-line
	// ranges are compared correctly.
	lineSizeMultiplier = 10000
)

// expressionFindResult holds the result of finding an expression at a position.
type expressionFindResult struct {
	// memberContext is the MemberExpr whose property contains the cursor, if any.
	// It provides method lookup context when the best match is an Identifier that
	// is the property of a MemberExpr.
	memberContext *ast_domain.MemberExpression

	// bestMatch is the most specific expression found at the target position.
	bestMatch ast_domain.Expression

	// bestRange is the LSP range of the most specific expression found.
	bestRange protocol.Range
}

// findExpressionAtPosition searches the AST to find the most specific
// expression node at the given cursor position.
//
// The search works in steps: first it finds the template node that contains
// the cursor, then it looks for the attribute or interpolation block within
// that node, and finally it searches down through the expression tree to find
// the most specific sub-expression.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template to search.
// Takes position (protocol.Position) which is the cursor position to find.
// Takes docPath (string) which is the document path for node matching.
//
// Returns ast_domain.Expression which is the most specific expression at the
// position, or nil if none is found.
// Returns protocol.Range which is the range of the found expression, or an
// empty range if none is found.
func findExpressionAtPosition(ctx context.Context, tree *ast_domain.TemplateAST, position protocol.Position, docPath string) (ast_domain.Expression, protocol.Range) {
	result := findExpressionAtPositionWithContext(ctx, tree, position, docPath)
	return result.bestMatch, result.bestRange
}

// findExpressionAtPositionWithContext searches the AST to find the most
// specific expression node at a cursor position and tracks method context
// for method signature lookups.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template to search.
// Takes position (protocol.Position) which is the cursor position to find.
// Takes docPath (string) which is the document path for node matching.
//
// Returns expressionFindResult which contains the most specific expression,
// its range, and any MemberExpr context for method lookups.
func findExpressionAtPositionWithContext(ctx context.Context, tree *ast_domain.TemplateAST, position protocol.Position, docPath string) expressionFindResult {
	_, l := logger_domain.From(ctx, log)

	targetNode := findTargetNodeWithFallback(ctx, tree, position, docPath)
	if targetNode == nil {
		return expressionFindResult{}
	}

	logTargetNodeInfo(ctx, targetNode)
	topLevelExpr, baseLocation := findExpressionOnNodeOrChildren(ctx, tree, targetNode, position)

	if topLevelExpr == nil {
		l.Debug("findExpressionAtPosition: No top-level expression found on node or children")
		return expressionFindResult{}
	}

	return findMostSpecificExpression(topLevelExpr, baseLocation, position)
}

// findTargetNodeWithFallback finds the target node at the given position,
// trying partial invocation fallback if the primary search fails.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template to search.
// Takes position (protocol.Position) which specifies the cursor position.
// Takes docPath (string) which identifies the document being searched.
//
// Returns *ast_domain.TemplateNode which is the found node, or nil if no node
// exists at the position.
func findTargetNodeWithFallback(ctx context.Context, tree *ast_domain.TemplateAST, position protocol.Position, docPath string) *ast_domain.TemplateNode {
	_, l := logger_domain.From(ctx, log)

	targetNode := findNodeAtPosition(tree, position, docPath)
	if targetNode != nil {
		return targetNode
	}

	l.Debug("findExpressionAtPosition: Primary search returned nil, trying partial invocation fallback",
		logger_domain.Int("line", int(position.Line)), logger_domain.Int("char", int(position.Character)))

	targetNode = findNodeAtPartialInvocationSite(ctx, tree, position)
	if targetNode != nil {
		l.Debug("findExpressionAtPosition: Found node via partial invocation fallback",
			logger_domain.String("tagName", targetNode.TagName))
	} else {
		l.Debug("findExpressionAtPosition: No target node found",
			logger_domain.Int("line", int(position.Line)), logger_domain.Int("char", int(position.Character)))
	}
	return targetNode
}

// logTargetNodeInfo logs debug information about a target node.
//
// Takes targetNode (*ast_domain.TemplateNode) which is the node to log details
// for.
func logTargetNodeInfo(ctx context.Context, targetNode *ast_domain.TemplateNode) {
	_, l := logger_domain.From(ctx, log)

	hasPartialInfo := targetNode.GoAnnotations != nil && targetNode.GoAnnotations.PartialInfo != nil
	sourcePath := ""
	if targetNode.GoAnnotations != nil && targetNode.GoAnnotations.OriginalSourcePath != nil {
		sourcePath = *targetNode.GoAnnotations.OriginalSourcePath
	}
	l.Debug("findExpressionAtPosition: Found target node",
		logger_domain.Int("nodeType", int(targetNode.NodeType)),
		logger_domain.String("tagName", targetNode.TagName),
		logger_domain.Int("richTextLen", len(targetNode.RichText)),
		logger_domain.Int("childrenLen", len(targetNode.Children)),
		logger_domain.Bool("hasPartialInfo", hasPartialInfo),
		logger_domain.String("sourcePath", sourcePath),
		logger_domain.Int("numAttrs", len(targetNode.Attributes)),
		logger_domain.Int("numDynAttrs", len(targetNode.DynamicAttributes)))
}

// findExpressionOnNodeOrChildren finds an expression on the node or its
// children.
//
// Takes tree (*ast_domain.TemplateAST) which provides the full template AST.
// Takes targetNode (*ast_domain.TemplateNode) which is the node to search.
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns ast_domain.Expression which is the found expression, or nil if none.
// Returns ast_domain.Location which is the location of the found expression.
func findExpressionOnNodeOrChildren(ctx context.Context, tree *ast_domain.TemplateAST, targetNode *ast_domain.TemplateNode, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	topLevelExpr, baseLocation := findTopLevelExpressionOnNode(ctx, targetNode, position)

	if topLevelExpr == nil {
		topLevelExpr, baseLocation = findExprInTextChildren(ctx, targetNode, position)
	}

	if topLevelExpr == nil {
		topLevelExpr, baseLocation = tryPartialInvocationFallback(ctx, tree, position)
	}

	return topLevelExpr, baseLocation
}

// findExprInTextChildren searches text node children for rich text expressions.
//
// Takes targetNode (*ast_domain.TemplateNode) which is the node whose children
// to search.
// Takes position (protocol.Position) which specifies the position
// to match against.
//
// Returns ast_domain.Expression which is the found expression, or nil if none.
// Returns ast_domain.Location which is the location of the expression, or an
// empty location if none found.
func findExprInTextChildren(ctx context.Context, targetNode *ast_domain.TemplateNode, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	_, l := logger_domain.From(ctx, log)

	for _, child := range targetNode.Children {
		if child.NodeType != ast_domain.NodeText || len(child.RichText) == 0 {
			continue
		}
		l.Debug("findExpressionAtPosition: Checking text child for RichText",
			logger_domain.Int("childRichTextLen", len(child.RichText)))
		if expression, location := findExprInRichText(ctx, child.RichText, position); expression != nil {
			return expression, location
		}
	}
	return nil, ast_domain.Location{}
}

// tryPartialInvocationFallback tries to find an expression on a partial
// invocation node when the primary node lookup fails.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template to search.
// Takes position (protocol.Position) which is the cursor position to check.
//
// Returns ast_domain.Expression which is the found expression, or nil if none.
// Returns ast_domain.Location which is the location of the expression.
func tryPartialInvocationFallback(ctx context.Context, tree *ast_domain.TemplateAST, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("findExpressionAtPosition: No expression on primary node, trying partial invocation fallback")
	partialNode := findNodeAtPartialInvocationSite(ctx, tree, position)
	if partialNode == nil {
		return nil, ast_domain.Location{}
	}

	l.Debug("findExpressionAtPosition: Found partial invocation node",
		logger_domain.String("tagName", partialNode.TagName))

	expression, location := findTopLevelExpressionOnNode(ctx, partialNode, position)
	if expression != nil {
		l.Debug("findExpressionAtPosition: Found expression on partial invocation node")
	}
	return expression, location
}

// findMostSpecificExpression searches an expression tree to find the smallest
// expression that contains the given position.
//
// Takes expression (ast_domain.Expression) which is the root
// expression to search.
// Takes baseLocation (ast_domain.Location) which provides the base
// offset for range calculations.
// Takes position (protocol.Position) which is the position to find
// within the tree.
//
// Returns expressionFindResult which contains the most specific
// expression found at the position, or an empty result if no match
// exists.
func findMostSpecificExpression(expression ast_domain.Expression, baseLocation ast_domain.Location, position protocol.Position) expressionFindResult {
	result := expressionFindResult{
		bestRange: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: maxRangeValue, Character: maxRangeValue},
		},
	}

	visitExpressionTreeWithContext(expression, baseLocation, position, &result)

	return result
}

// visitExpressionTreeWithContext walks an expression tree and tracks both the
// most specific expression and any MemberExpr context for method lookups.
//
// Takes expression (ast_domain.Expression) which is the expression
// to visit.
// Takes baseLocation (ast_domain.Location) which provides position
// offsets.
// Takes position (protocol.Position) which is the target cursor
// position.
// Takes result (*expressionFindResult) which stores the search
// results.
func visitExpressionTreeWithContext(expression ast_domain.Expression, baseLocation ast_domain.Location, position protocol.Position, result *expressionFindResult) {
	if expression == nil {
		return
	}

	expressionRange := calculateExpressionRange(expression, baseLocation)
	if isPositionInRange(position, expressionRange) && isRangeSmaller(expressionRange, result.bestRange) {
		result.bestMatch = expression
		result.bestRange = expressionRange
	}

	if memberExpr, ok := expression.(*ast_domain.MemberExpression); ok {
		visitExpressionTreeWithContext(memberExpr.Base, baseLocation, position, result)

		if memberExpr.Property != nil {
			propRange := calculateExpressionRange(memberExpr.Property, baseLocation)
			if isPositionInRange(position, propRange) {
				result.memberContext = memberExpr
			}
		}
		visitExpressionTreeWithContext(memberExpr.Property, baseLocation, position, result)
		return
	}

	visitExpressionChildrenWithContext(expression, baseLocation, position, result)
}

// visitExpressionChildrenWithContext visits child expressions with context
// tracking.
//
// Takes expression (ast_domain.Expression) which is the parent
// expression to visit.
// Takes baseLocation (ast_domain.Location) which is the base
// position for calculating child locations.
// Takes position (protocol.Position) which is the cursor position
// being searched.
// Takes result (*expressionFindResult) which accumulates matching
// expressions.
func visitExpressionChildrenWithContext(expression ast_domain.Expression, baseLocation ast_domain.Location, position protocol.Position, result *expressionFindResult) {
	switch n := expression.(type) {
	case *ast_domain.IndexExpression:
		visitExpressionTreeWithContext(n.Base, baseLocation, position, result)
		visitExpressionTreeWithContext(n.Index, baseLocation, position, result)
	case *ast_domain.CallExpression:
		visitExpressionTreeWithContext(n.Callee, baseLocation, position, result)
		for _, arg := range n.Args {
			visitExpressionTreeWithContext(arg, baseLocation, position, result)
		}
	case *ast_domain.BinaryExpression:
		visitExpressionTreeWithContext(n.Left, baseLocation, position, result)
		visitExpressionTreeWithContext(n.Right, baseLocation, position, result)
	case *ast_domain.UnaryExpression:
		visitExpressionTreeWithContext(n.Right, baseLocation, position, result)
	case *ast_domain.TernaryExpression:
		visitExpressionTreeWithContext(n.Condition, baseLocation, position, result)
		visitExpressionTreeWithContext(n.Consequent, baseLocation, position, result)
		visitExpressionTreeWithContext(n.Alternate, baseLocation, position, result)
	case *ast_domain.TemplateLiteral:
		for _, part := range n.Parts {
			if !part.IsLiteral && part.Expression != nil {
				expressionBase := baseLocation.Add(part.RelativeLocation)
				expressionBase.Column += 2
				visitExpressionTreeWithContext(part.Expression, expressionBase, position, result)
			}
		}
	case *ast_domain.ObjectLiteral:
		for _, value := range n.Pairs {
			visitExpressionTreeWithContext(value, baseLocation, position, result)
		}
	case *ast_domain.ArrayLiteral:
		for _, el := range n.Elements {
			visitExpressionTreeWithContext(el, baseLocation, position, result)
		}
	case *ast_domain.ForInExpression:
		visitForInExprChildrenWithContext(n, baseLocation, position, result)
	case *ast_domain.LinkedMessageExpression:
		visitExpressionTreeWithContext(n.Path, baseLocation, position, result)
	}
}

// visitForInExprChildrenWithContext visits the child nodes of a ForInExpr.
//
// Takes n (*ast_domain.ForInExpression) which is the for-in expression to visit.
// Takes baseLocation (ast_domain.Location) which is the base location for
// position calculations.
// Takes position (protocol.Position) which is the position to match against.
// Takes result (*expressionFindResult) which accumulates matching expressions.
func visitForInExprChildrenWithContext(n *ast_domain.ForInExpression, baseLocation ast_domain.Location, position protocol.Position, result *expressionFindResult) {
	if n.IndexVariable != nil {
		visitExpressionTreeWithContext(n.IndexVariable, baseLocation, position, result)
	}
	if n.ItemVariable != nil {
		visitExpressionTreeWithContext(n.ItemVariable, baseLocation, position, result)
	}
	visitExpressionTreeWithContext(n.Collection, baseLocation, position, result)
}

// calculateExpressionRange computes the LSP range for an expression given its
// base location.
//
// Takes expression (ast_domain.Expression) which is the expression
// to compute the range for.
// Takes baseLocation (ast_domain.Location) which is the base
// position to offset from.
//
// Returns protocol.Range which is the computed range with start and
// end positions.
func calculateExpressionRange(expression ast_domain.Expression, baseLocation ast_domain.Location) protocol.Range {
	finalStart := baseLocation.Add(expression.GetRelativeLocation())
	expressionLength := expression.GetSourceLength()

	return protocol.Range{
		Start: protocol.Position{
			Line:      safeconv.IntToUint32(finalStart.Line - 1),
			Character: safeconv.IntToUint32(finalStart.Column - 1),
		},
		End: protocol.Position{
			Line:      safeconv.IntToUint32(finalStart.Line - 1),
			Character: safeconv.IntToUint32(finalStart.Column - 1 + expressionLength),
		},
	}
}

// visitExpressionTree walks an expression tree and calls the visitor function
// for each node.
//
// Takes expression (ast_domain.Expression) which is the root
// expression to walk.
// Takes visitor (func(...)) which is called for each node in the
// tree.
func visitExpressionTree(expression ast_domain.Expression, visitor func(ast_domain.Expression)) {
	if expression == nil {
		return
	}
	visitor(expression)
	visitExpressionChildren(expression, visitor)
}

// visitExpressionChildren visits each child node of the given expression.
//
// Takes expression (ast_domain.Expression) which is the parent
// expression to visit.
// Takes visitor (func(...)) which is called for each child
// expression.
func visitExpressionChildren(expression ast_domain.Expression, visitor func(ast_domain.Expression)) {
	switch n := expression.(type) {
	case *ast_domain.MemberExpression:
		visitExpressionTree(n.Base, visitor)
		visitExpressionTree(n.Property, visitor)
	case *ast_domain.IndexExpression:
		visitExpressionTree(n.Base, visitor)
		visitExpressionTree(n.Index, visitor)
	case *ast_domain.CallExpression:
		visitExpressionTree(n.Callee, visitor)
		for _, arg := range n.Args {
			visitExpressionTree(arg, visitor)
		}
	case *ast_domain.BinaryExpression:
		visitExpressionTree(n.Left, visitor)
		visitExpressionTree(n.Right, visitor)
	case *ast_domain.UnaryExpression:
		visitExpressionTree(n.Right, visitor)
	case *ast_domain.TernaryExpression:
		visitExpressionTree(n.Condition, visitor)
		visitExpressionTree(n.Consequent, visitor)
		visitExpressionTree(n.Alternate, visitor)
	case *ast_domain.TemplateLiteral:
		for _, part := range n.Parts {
			if !part.IsLiteral {
				visitExpressionTree(part.Expression, visitor)
			}
		}
	case *ast_domain.ObjectLiteral:
		for _, value := range n.Pairs {
			visitExpressionTree(value, visitor)
		}
	case *ast_domain.ArrayLiteral:
		for _, el := range n.Elements {
			visitExpressionTree(el, visitor)
		}
	case *ast_domain.ForInExpression:
		visitForInExprChildren(n, visitor)
	case *ast_domain.LinkedMessageExpression:
		visitExpressionTree(n.Path, visitor)
	}
}

// visitForInExprChildren visits the child nodes of a ForInExpr node.
//
// Takes n (*ast_domain.ForInExpression) which is the for-in
// expression to traverse.
// Takes visitor (func(...)) which is called for each child expression found.
func visitForInExprChildren(n *ast_domain.ForInExpression, visitor func(ast_domain.Expression)) {
	if n.IndexVariable != nil {
		visitExpressionTree(n.IndexVariable, visitor)
	}
	if n.ItemVariable != nil {
		visitExpressionTree(n.ItemVariable, visitor)
	}
	visitExpressionTree(n.Collection, visitor)
}

// findTopLevelExpressionOnNode searches a template node to find the expression
// container at the given position. It checks dynamic attributes, directives,
// rich text interpolations, and static attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to search.
// Takes position (protocol.Position) which is the position to
// find an expression at.
//
// Returns ast_domain.Expression which is the found expression, or nil if none.
// Returns ast_domain.Location which is the location of the expression.
func findTopLevelExpressionOnNode(ctx context.Context, node *ast_domain.TemplateNode, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	if expression, location := findExprInDynamicAttrs(node.DynamicAttributes, position); expression != nil {
		return expression, location
	}

	if expression, location := findExprInNodeDirectives(node, position); expression != nil {
		return expression, location
	}

	if expression, location := findExprInRichText(ctx, node.RichText, position); expression != nil {
		return expression, location
	}

	if expression, location := findExprInStaticAttrs(node.Attributes, position); expression != nil {
		return expression, location
	}

	if expression, location := findExprInPassedProps(ctx, node, position); expression != nil {
		return expression, location
	}

	return nil, ast_domain.Location{}
}

// findExprInPassedProps searches passed props for an expression at the given
// position. This handles server.* prefixed attributes on partial invocations.
//
// Takes ctx (context.Context) which carries the request context for logging.
// Takes node (*ast_domain.TemplateNode) which is the node to search.
// Takes position (protocol.Position) which is the position to find.
//
// Returns ast_domain.Expression which is the expression at the position, or
// nil if not found.
// Returns ast_domain.Location which is the location of the expression.
func findExprInPassedProps(ctx context.Context, node *ast_domain.TemplateNode, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	_, l := logger_domain.From(ctx, log)

	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return nil, ast_domain.Location{}
	}

	pInfo := node.GoAnnotations.PartialInfo
	for propName, propValue := range pInfo.PassedProps {
		nameLocation := propValue.NameLocation
		if nameLocation.IsSynthetic() {
			continue
		}

		propLine := safeconv.IntToUint32(nameLocation.Line - 1)
		if position.Line != propLine {
			continue
		}

		propCol := safeconv.IntToUint32(nameLocation.Column - 1)
		if position.Character >= propCol {
			l.Debug("findExprInPassedProps: Found expression",
				logger_domain.String("propName", propName),
				logger_domain.Int("propLine", nameLocation.Line),
				logger_domain.Int("propCol", nameLocation.Column))

			if propValue.InvokerAnnotation != nil && propValue.Expression != nil {
				copyAnnotationToExpression(propValue.Expression, propValue.InvokerAnnotation)
			}

			return propValue.Expression, propValue.Location
		}
	}

	return nil, ast_domain.Location{}
}

// copyAnnotationToExpression copies relevant fields from an annotation to
// the expression's own GoAnnotations. This is needed for PassedProps where
// the type info is stored in InvokerAnnotation but hover code expects it
// on the expression itself.
//
// Takes expression (ast_domain.Expression) which is the target
// expression to update.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the
// annotation data to copy.
func copyAnnotationToExpression(expression ast_domain.Expression, ann *ast_domain.GoGeneratorAnnotation) {
	if expression == nil || ann == nil {
		return
	}

	expressionAnnotation := expression.GetGoAnnotation()
	if expressionAnnotation == nil {
		expression.SetGoAnnotation(ann)
		return
	}

	if expressionAnnotation.ResolvedType == nil && ann.ResolvedType != nil {
		expressionAnnotation.ResolvedType = ann.ResolvedType
	}
	if expressionAnnotation.Symbol == nil && ann.Symbol != nil {
		expressionAnnotation.Symbol = ann.Symbol
	}
}

// findExprInDynamicAttrs searches dynamic attributes for an expression at the
// given position.
//
// Takes attrs ([]ast_domain.DynamicAttribute) which is the list of dynamic
// attributes to search.
// Takes position (protocol.Position) which is the position to find.
//
// Returns ast_domain.Expression which is the expression at the position, or
// nil if not found.
// Returns ast_domain.Location which is the location of the expression, or an
// empty location if not found.
func findExprInDynamicAttrs(attrs []ast_domain.DynamicAttribute, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	for i := range attrs {
		attribute := &attrs[i]
		attributeRange := protocol.Range{
			Start: protocol.Position{Line: safeconv.IntToUint32(attribute.AttributeRange.Start.Line - 1), Character: safeconv.IntToUint32(attribute.AttributeRange.Start.Column - 1)},
			End:   protocol.Position{Line: safeconv.IntToUint32(attribute.AttributeRange.End.Line - 1), Character: safeconv.IntToUint32(attribute.AttributeRange.End.Column - 1)},
		}
		if isPositionInRange(position, attributeRange) {
			return attribute.Expression, attribute.Location
		}
	}
	return nil, ast_domain.Location{}
}

// findExprInNodeDirectives searches all directive fields on a node for an
// expression at the given position.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to search.
// Takes position (protocol.Position) which specifies the position to find.
//
// Returns ast_domain.Expression which is the expression found at the position,
// or nil if none exists.
// Returns ast_domain.Location which is the location of the found expression,
// or an empty location if none exists.
func findExprInNodeDirectives(node *ast_domain.TemplateNode, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	directives := []*ast_domain.Directive{
		node.DirIf, node.DirElseIf, node.DirFor, node.DirShow, node.DirModel,
		node.DirText, node.DirHTML, node.DirClass, node.DirStyle,
		node.DirKey, node.DirContext, node.DirScaffold,
	}

	for _, directive := range directives {
		if expression, location, ok := checkDirectiveAtPos(directive, position); ok {
			return expression, location
		}
	}

	for _, directive := range node.Binds {
		if expression, location, ok := checkDirectiveAtPos(directive, position); ok {
			return expression, location
		}
	}

	for _, dirs := range node.OnEvents {
		for i := range dirs {
			if expression, location, ok := checkDirectiveAtPos(&dirs[i], position); ok {
				return expression, location
			}
		}
	}

	return nil, ast_domain.Location{}
}

// checkDirectiveAtPos checks if a directive contains the given position.
//
// Takes directive (*ast_domain.Directive) which is the directive to check.
// Takes position (protocol.Position) which is the position to look for.
//
// Returns ast_domain.Expression which is the directive's expression if found.
// Returns ast_domain.Location which is the directive's location if found.
// Returns bool which indicates whether the position is within the directive.
func checkDirectiveAtPos(directive *ast_domain.Directive, position protocol.Position) (ast_domain.Expression, ast_domain.Location, bool) {
	if directive == nil || directive.AttributeRange.Start.IsSynthetic() {
		return nil, ast_domain.Location{}, false
	}
	directiveRange := protocol.Range{
		Start: protocol.Position{Line: safeconv.IntToUint32(directive.AttributeRange.Start.Line - 1), Character: safeconv.IntToUint32(directive.AttributeRange.Start.Column - 1)},
		End:   protocol.Position{Line: safeconv.IntToUint32(directive.AttributeRange.End.Line - 1), Character: safeconv.IntToUint32(directive.AttributeRange.End.Column - 1)},
	}
	if isPositionInRange(position, directiveRange) {
		return directive.Expression, directive.Location, true
	}
	return nil, ast_domain.Location{}, false
}

// findExprInRichText searches rich text interpolations for an expression at
// the given position.
//
// Takes parts ([]ast_domain.TextPart) which contains the rich text parts to
// search through.
// Takes position (protocol.Position) which specifies the cursor position to match.
//
// Returns ast_domain.Expression which is the matched expression, or nil if no
// match is found.
// Returns ast_domain.Location which is the location of the matched expression,
// or an empty location if no match is found.
func findExprInRichText(ctx context.Context, parts []ast_domain.TextPart, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("findExprInRichText: Searching",
		logger_domain.Int("partsLen", len(parts)),
		logger_domain.Int(logKeyPosLine, int(position.Line)),
		logger_domain.Int(logKeyPosChar, int(position.Character)))

	for i := range parts {
		part := &parts[i]
		if part.IsLiteral {
			continue
		}
		partRange := protocol.Range{
			Start: protocol.Position{Line: safeconv.IntToUint32(part.Location.Line - 1), Character: safeconv.IntToUint32(part.Location.Column - 1 - 2)},
			End:   protocol.Position{Line: safeconv.IntToUint32(part.Location.Line - 1), Character: safeconv.IntToUint32(part.Location.Column - 1 + len(part.RawExpression) + 2)},
		}

		l.Debug("findExprInRichText: Checking part",
			logger_domain.Int("partIndex", i),
			logger_domain.Int("partLocLine", part.Location.Line),
			logger_domain.Int("partLocCol", part.Location.Column),
			logger_domain.String("rawExpr", part.RawExpression),
			logger_domain.Int(logKeyRangeStartLine, int(partRange.Start.Line)),
			logger_domain.Int(logKeyRangeStartChar, int(partRange.Start.Character)),
			logger_domain.Int(logKeyRangeEndLine, int(partRange.End.Line)),
			logger_domain.Int(logKeyRangeEndChar, int(partRange.End.Character)))

		if isPositionInRange(position, partRange) {
			l.Debug("findExprInRichText: Found matching part")
			return part.Expression, part.Location
		}
	}
	return nil, ast_domain.Location{}
}

// findExprInStaticAttrs creates a synthetic string literal for a static
// attribute value at the given cursor position.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// search through.
// Takes position (protocol.Position) which specifies the cursor position to match.
//
// Returns ast_domain.Expression which is a synthetic string literal for the
// matched attribute, or nil if no match is found.
// Returns ast_domain.Location which is the location of the matched attribute,
// or an empty location if no match is found.
func findExprInStaticAttrs(attrs []ast_domain.HTMLAttribute, position protocol.Position) (ast_domain.Expression, ast_domain.Location) {
	for i := range attrs {
		attribute := &attrs[i]
		if !isPositionInAttributeValue(position, &attribute.AttributeRange) {
			continue
		}

		sourceLen := calculateAttrSourceLen(attribute)
		stringLit := createSyntheticStringLiteral(attribute, sourceLen)
		return stringLit, attribute.Location
	}
	return nil, ast_domain.Location{}
}

// calculateAttrSourceLen works out the source length of an attribute value.
//
// Takes attribute (*ast_domain.HTMLAttribute) which contains the attribute to
// measure.
//
// Returns int which is the length of the attribute value in source code. For
// attributes on a single line, this is the column difference. For attributes
// that span multiple lines, this is the string length.
func calculateAttrSourceLen(attribute *ast_domain.HTMLAttribute) int {
	if attribute.AttributeRange.Start.Line == attribute.AttributeRange.End.Line {
		return attribute.AttributeRange.End.Column - attribute.Location.Column - 1
	}
	return len(attribute.Value)
}

// createSyntheticStringLiteral creates a StringLiteral expression for a static
// HTML attribute.
//
// Takes attribute (*ast_domain.HTMLAttribute) which provides the attribute value
// and name for the string literal.
// Takes sourceLen (int) which sets the source length for the literal.
//
// Returns *ast_domain.StringLiteral which contains the created literal with
// Go annotations for hover provider support.
func createSyntheticStringLiteral(attribute *ast_domain.HTMLAttribute, sourceLen int) *ast_domain.StringLiteral {
	stringLit := &ast_domain.StringLiteral{
		Value:            attribute.Value,
		RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
		SourceLength:     sourceLen,
	}

	stringLit.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
		Symbol: &ast_domain.ResolvedSymbol{
			Name: attribute.Name,
		},
		Stringability: int(inspector_dto.StringablePrimitive),
	}

	return stringLit
}

// isPositionInAttributeValue checks if a position is inside the value part of
// an attribute range.
//
// Takes position (protocol.Position) which specifies the position to check.
// Takes attributeRange (*ast_domain.Range) which defines the attribute's range.
//
// Returns bool which is true if the position is within the attribute value.
func isPositionInAttributeValue(position protocol.Position, attributeRange *ast_domain.Range) bool {
	if attributeRange.Start.IsSynthetic() {
		return false
	}
	r := protocol.Range{
		Start: protocol.Position{Line: safeconv.IntToUint32(attributeRange.Start.Line - 1), Character: safeconv.IntToUint32(attributeRange.Start.Column - 1)},
		End:   protocol.Position{Line: safeconv.IntToUint32(attributeRange.End.Line - 1), Character: safeconv.IntToUint32(attributeRange.End.Column - 1)},
	}
	return isPositionInRange(position, r)
}
