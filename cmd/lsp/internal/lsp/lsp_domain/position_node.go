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
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// logKeyName is the structured logging key for name attributes.
	logKeyName = "name"

	// logKeyPosLine is the log key for the cursor position line number.
	logKeyPosLine = "posLine"

	// logKeyPosChar is the log key for the character position.
	logKeyPosChar = "posChar"

	// logKeyRangeStartLine is the log key for the start line of a range.
	logKeyRangeStartLine = "rangeStartLine"

	// logKeyRangeStartCol is the log key for the starting column of a range.
	logKeyRangeStartCol = "rangeStartCol"

	// logKeyRangeStartChar is the log key for a range start character position.
	logKeyRangeStartChar = "rangeStartChar"

	// logKeyRangeEndLine is the log key for the ending line of a range.
	logKeyRangeEndLine = "rangeEndLine"

	// logKeyRangeEndCol is the log key for the ending column of a range.
	logKeyRangeEndCol = "rangeEndCol"

	// logKeyRangeEndChar is the log key for a range ending character position.
	logKeyRangeEndChar = "rangeEndChar"
)

// nodeFindState tracks the state when searching for a node at a given position.
type nodeFindState struct {
	// bestMatch holds the most specific node that contains the position.
	bestMatch *ast_domain.TemplateNode

	// bestRange is the source range of the current best matching node.
	bestRange protocol.Range

	// foundInTag is true when the position was found inside an opening tag.
	foundInTag bool
}

// processNode evaluates a single node during the position search.
//
// Takes node (*ast_domain.TemplateNode) which is the node to evaluate.
// Takes position (protocol.Position) which is the position being searched for.
// Takes docPath (string) which is the path of the document being searched.
//
// Returns bool which is false to stop traversal when found, true to continue.
func (s *nodeFindState) processNode(node *ast_domain.TemplateNode, position protocol.Position, docPath string) bool {
	if s.foundInTag {
		return false
	}

	if !isNodeFromDocument(node, docPath) {
		return true
	}

	if node.NodeRange.Start.IsSynthetic() {
		return true
	}

	if s.checkOpeningTag(node, position) {
		return false
	}

	s.checkNodeRange(node, position)
	return true
}

// checkOpeningTag checks if the position is within the node's opening tag.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes position (protocol.Position) which is the position to locate.
//
// Returns bool which is true if found, indicating the search should stop.
func (s *nodeFindState) checkOpeningTag(node *ast_domain.TemplateNode, position protocol.Position) bool {
	if node.OpeningTagRange.Start.IsSynthetic() {
		return false
	}

	openingTagRange := astRangeToLSPRange(node.OpeningTagRange)
	if isPositionInRange(position, openingTagRange) {
		s.bestMatch = node
		s.foundInTag = true
		return true
	}
	return false
}

// checkNodeRange checks if the position is within the node's overall range
// and updates best match.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes position (protocol.Position) which is the position to test against.
func (s *nodeFindState) checkNodeRange(node *ast_domain.TemplateNode, position protocol.Position) {
	nodeRange := astRangeToLSPRange(node.NodeRange)
	if !isPositionInRange(position, nodeRange) {
		return
	}

	if s.bestMatch == nil || isRangeSmaller(nodeRange, s.bestRange) {
		s.bestMatch = node
		s.bestRange = nodeRange
	}
}

// findNodeAtPosition finds the smallest node that contains the given cursor
// position and comes from the specified document.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template to search.
// Takes position (protocol.Position) which specifies the cursor position to find.
// Takes docPath (string) which filters nodes by their source document.
//
// Returns *ast_domain.TemplateNode which is the innermost node at the position,
// or nil if no matching node is found.
func findNodeAtPosition(tree *ast_domain.TemplateAST, position protocol.Position, docPath string) *ast_domain.TemplateNode {
	state := &nodeFindState{}

	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		return state.processNode(node, position, docPath)
	})

	return state.bestMatch
}

// isNodeFromDocument checks if a node comes from the specified document.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes docPath (string) which is the path of the document to match against.
//
// Returns bool which is true if the node's original source path matches
// docPath, or if the node has no source path annotation. Nodes without this
// annotation are included because they likely belong to the current document.
func isNodeFromDocument(node *ast_domain.TemplateNode, docPath string) bool {
	if node.GoAnnotations == nil || node.GoAnnotations.OriginalSourcePath == nil {
		return true
	}
	return *node.GoAnnotations.OriginalSourcePath == docPath
}

// findNodeAtPartialInvocationSite finds a node whose merged invoker attributes
// contain the given cursor position. This handles the case where the cursor is
// at a partial invocation site but the expanded node's OriginalSourcePath points
// to the partial definition file.
//
// When a partial is expanded, its attributes are merged from the invoker node,
// preserving their original AttributeRange locations from the invoking document.
// The merged attributes can then be checked against the cursor position to
// locate partial invocation nodes.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template to search.
// Takes position (protocol.Position) which is the cursor position to locate.
//
// Returns *ast_domain.TemplateNode which is the partial invocation node at the
// position, or nil if no match is found.
func findNodeAtPartialInvocationSite(ctx context.Context, tree *ast_domain.TemplateAST, position protocol.Position) *ast_domain.TemplateNode {
	_, l := logger_domain.From(ctx, log)

	var result *ast_domain.TemplateNode

	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		if !isPartialInvocationNode(node) {
			return true
		}

		pInfo := node.GoAnnotations.PartialInfo
		var passedPropNames []string
		for k := range pInfo.PassedProps {
			passedPropNames = append(passedPropNames, k)
		}
		l.Debug("findNodeAtPartialInvocationSite: Checking partial node",
			logger_domain.String("tagName", node.TagName),
			logger_domain.String("partialAlias", pInfo.PartialAlias),
			logger_domain.Int("numAttrs", len(node.Attributes)),
			logger_domain.Int("numDynAttrs", len(node.DynamicAttributes)),
			logger_domain.Int("numPassedProps", len(pInfo.PassedProps)),
			logger_domain.Strings("passedPropNames", passedPropNames),
			logger_domain.Int("invocationLine", pInfo.Location.Line),
			logger_domain.Int("invocationCol", pInfo.Location.Column))

		if hasAttributeAtPosition(ctx, node, position) {
			l.Debug("findNodeAtPartialInvocationSite: Found matching partial",
				logger_domain.String("tagName", node.TagName))
			result = node
			return false
		}

		return true
	})

	return result
}

// isPartialInvocationNode checks if a node represents an expanded partial
// invocation by checking whether it has PartialInfo attached.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node has PartialInfo, meaning it is an
// expanded partial invocation.
func isPartialInvocationNode(node *ast_domain.TemplateNode) bool {
	return node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil
}

// hasAttributeAtPosition checks if any attribute or directive on the node has
// a range that contains the given position. This works because merged invoker
// attributes and directives preserve their original AttributeRange from the
// invoking document.
//
// Takes node (*ast_domain.TemplateNode) which is the node whose attributes to
// check.
// Takes position (protocol.Position) which is the cursor position to match.
//
// Returns bool which is true if any attribute's or directive's range contains
// the position.
func hasAttributeAtPosition(ctx context.Context, node *ast_domain.TemplateNode, position protocol.Position) bool {
	for i := range node.Attributes {
		attribute := &node.Attributes[i]
		if checkAttrRangeMatch(ctx, attribute.Name, &attribute.AttributeRange, position, "static attr") {
			return true
		}
	}

	for i := range node.DynamicAttributes {
		attribute := &node.DynamicAttributes[i]
		if checkAttrRangeMatch(ctx, attribute.Name, &attribute.AttributeRange, position, "dynamic attr") {
			return true
		}
	}

	if hasDirectiveAtPosition(ctx, node, position) {
		return true
	}

	if hasPassedPropAtPosition(ctx, node, position) {
		return true
	}

	return false
}

// checkAttrRangeMatch checks if the given position is within an attribute's
// range. It logs debug information about the check and returns true if a match
// is found.
//
// Takes name (string) which identifies the attribute being checked.
// Takes attributeRange (*ast_domain.Range) which defines the attribute boundaries.
// Takes position (protocol.Position) which specifies the position to check.
// Takes attributeType (string) which describes the type of attribute for logging.
//
// Returns bool which is true if the position falls within the attribute range.
func checkAttrRangeMatch(ctx context.Context, name string, attributeRange *ast_domain.Range, position protocol.Position, attributeType string) bool {
	_, l := logger_domain.From(ctx, log)

	l.Debug("hasAttributeAtPosition: Checking "+attributeType,
		logger_domain.String(logKeyName, name),
		logger_domain.Int(logKeyRangeStartLine, attributeRange.Start.Line),
		logger_domain.Int(logKeyRangeStartCol, attributeRange.Start.Column),
		logger_domain.Int(logKeyRangeEndLine, attributeRange.End.Line),
		logger_domain.Int(logKeyRangeEndCol, attributeRange.End.Column),
		logger_domain.Int(logKeyPosLine, int(position.Line)),
		logger_domain.Int(logKeyPosChar, int(position.Character)))
	if isPositionInAttributeRange(position, attributeRange) {
		l.Debug("hasAttributeAtPosition: Match found on "+attributeType, logger_domain.String(logKeyName, name))
		return true
	}
	return false
}

// hasPassedPropAtPosition checks if any PassedProp on a partial invocation
// node has a location that matches the given position.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes position (protocol.Position) which is the cursor position to match.
//
// Returns bool which is true if any PassedProp's location matches.
func hasPassedPropAtPosition(ctx context.Context, node *ast_domain.TemplateNode, position protocol.Position) bool {
	_, l := logger_domain.From(ctx, log)

	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return false
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
			l.Debug("hasPassedPropAtPosition: Match found",
				logger_domain.String("propName", propName),
				logger_domain.Int("propLine", nameLocation.Line),
				logger_domain.Int("propCol", nameLocation.Column),
				logger_domain.Int(logKeyPosLine, int(position.Line)),
				logger_domain.Int(logKeyPosChar, int(position.Character)))
			return true
		}
	}

	return false
}

// hasDirectiveAtPosition checks if any directive on the node contains the
// given position.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes position (protocol.Position) which is the cursor position to match.
//
// Returns bool which is true if any directive's range contains the position.
func hasDirectiveAtPosition(ctx context.Context, node *ast_domain.TemplateNode, position protocol.Position) bool {
	if hasStandardDirectiveAtPosition(ctx, node, position) {
		return true
	}
	if hasBindDirectiveAtPosition(ctx, node.Binds, position) {
		return true
	}
	return hasEventDirectiveAtPosition(ctx, node.OnEvents, position)
}

// hasStandardDirectiveAtPosition checks standard directives (if, for, etc.)
// for a position match.
//
// Takes node (*ast_domain.TemplateNode) which contains the directives to check.
// Takes position (protocol.Position) which specifies the position
// to match against.
//
// Returns bool which is true if the position falls within any
// directive range.
func hasStandardDirectiveAtPosition(ctx context.Context, node *ast_domain.TemplateNode, position protocol.Position) bool {
	_, l := logger_domain.From(ctx, log)

	directives := []*ast_domain.Directive{
		node.DirIf, node.DirElseIf, node.DirFor, node.DirShow, node.DirModel,
		node.DirText, node.DirHTML, node.DirClass, node.DirStyle,
		node.DirKey, node.DirContext, node.DirScaffold,
	}

	for index, directive := range directives {
		if directive == nil || directive.AttributeRange.Start.IsSynthetic() {
			continue
		}
		l.Debug("hasDirectiveAtPosition: Checking directive",
			logger_domain.Int("index", index),
			logger_domain.Int(logKeyRangeStartLine, directive.AttributeRange.Start.Line),
			logger_domain.Int(logKeyRangeStartCol, directive.AttributeRange.Start.Column),
			logger_domain.Int(logKeyRangeEndLine, directive.AttributeRange.End.Line),
			logger_domain.Int(logKeyRangeEndCol, directive.AttributeRange.End.Column),
			logger_domain.Int(logKeyPosLine, int(position.Line)),
			logger_domain.Int(logKeyPosChar, int(position.Character)))
		if isPositionInAttributeRange(position, &directive.AttributeRange) {
			l.Debug("hasDirectiveAtPosition: Match found on directive", logger_domain.Int("index", index))
			return true
		}
	}
	return false
}

// hasBindDirectiveAtPosition checks bind directives for a position match.
//
// Takes ctx (context.Context) which carries the request context for logging.
// Takes binds (map[string]*ast_domain.Directive) which contains the bind
// directives to search.
// Takes position (protocol.Position) which specifies the position to check.
//
// Returns bool which is true if the position falls within any bind directive's
// attribute range.
func hasBindDirectiveAtPosition(ctx context.Context, binds map[string]*ast_domain.Directive, position protocol.Position) bool {
	_, l := logger_domain.From(ctx, log)

	for _, directive := range binds {
		if directive == nil || directive.AttributeRange.Start.IsSynthetic() {
			continue
		}
		if isPositionInAttributeRange(position, &directive.AttributeRange) {
			l.Debug("hasDirectiveAtPosition: Match found on bind directive", logger_domain.String("arg", directive.Arg))
			return true
		}
	}
	return false
}

// hasEventDirectiveAtPosition checks event directives for a position match.
//
// Takes onEvents (map[string][]ast_domain.Directive) which contains the event
// directives to search through.
// Takes position (protocol.Position) which specifies the position
// to match against.
//
// Returns bool which is true if the position falls within any
// non-synthetic
// event directive's attribute range.
func hasEventDirectiveAtPosition(ctx context.Context, onEvents map[string][]ast_domain.Directive, position protocol.Position) bool {
	_, l := logger_domain.From(ctx, log)

	for _, dirs := range onEvents {
		for i := range dirs {
			directive := &dirs[i]
			if directive.AttributeRange.Start.IsSynthetic() {
				continue
			}
			if isPositionInAttributeRange(position, &directive.AttributeRange) {
				l.Debug("hasDirectiveAtPosition: Match found on event directive", logger_domain.String("arg", directive.Arg))
				return true
			}
		}
	}
	return false
}

// isPositionInAttributeRange checks if a position falls within an attribute's
// range.
//
// Takes position (protocol.Position) which is the position to check.
// Takes attributeRange (*ast_domain.Range) which defines the attribute's range.
//
// Returns bool which is true if the position is within the attribute range,
// or false if the range is synthetic.
func isPositionInAttributeRange(position protocol.Position, attributeRange *ast_domain.Range) bool {
	if attributeRange.Start.IsSynthetic() {
		return false
	}
	r := astRangeToLSPRange(*attributeRange)
	return isPositionInRange(position, r)
}

// astRangeToLSPRange converts a 1-based ast_domain.Range to a 0-based
// protocol.Range. Lines and columns in the source range start at 1, while the
// LSP protocol uses positions that start at 0.
//
// Takes astRange (ast_domain.Range) which specifies the source range to
// convert.
//
// Returns protocol.Range which is the converted range with 0-based positions.
func astRangeToLSPRange(astRange ast_domain.Range) protocol.Range {
	if astRange.Start.IsSynthetic() {
		return protocol.Range{}
	}

	end := astRange.End
	if end.IsSynthetic() {
		end = astRange.Start
	}

	return protocol.Range{
		Start: protocol.Position{
			Line:      safeconv.IntToUint32(astRange.Start.Line - 1),
			Character: safeconv.IntToUint32(astRange.Start.Column - 1),
		},
		End: protocol.Position{
			Line:      safeconv.IntToUint32(end.Line - 1),
			Character: safeconv.IntToUint32(end.Column - 1),
		},
	}
}

// isPositionInRange checks if a position falls within a given range.
//
// Takes position (protocol.Position) which is the position to check.
// Takes r (protocol.Range) which defines the range boundaries.
//
// Returns bool which is true if the position is within the range, inclusive.
func isPositionInRange(position protocol.Position, r protocol.Range) bool {
	return (position.Line > r.Start.Line || (position.Line == r.Start.Line && position.Character >= r.Start.Character)) &&
		(position.Line < r.End.Line || (position.Line == r.End.Line && position.Character <= r.End.Character))
}

// isRangeSmaller checks if the first range is smaller than the second.
//
// Takes r1 (protocol.Range) which is the first range to compare.
// Takes r2 (protocol.Range) which is the second range to compare.
//
// Returns bool which is true when r1 covers fewer characters than r2.
func isRangeSmaller(r1, r2 protocol.Range) bool {
	size1 := (r1.End.Line-r1.Start.Line)*lineSizeMultiplier + (r1.End.Character - r1.Start.Character)
	size2 := (r2.End.Line-r2.Start.Line)*lineSizeMultiplier + (r2.End.Character - r2.Start.Character)
	return size1 < size2
}
