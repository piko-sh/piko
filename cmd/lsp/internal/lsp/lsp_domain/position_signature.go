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
	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/safeconv"
)

// callExprFinder is a visitor that finds the smallest CallExpr containing a
// target position within its parentheses.
type callExprFinder struct {
	// bestMatch holds the most specific call expression that contains the cursor.
	bestMatch *ast_domain.CallExpression

	// bestMatchRange is the source range of the current best match.
	bestMatchRange protocol.Range

	// targetPosition is the position to find within call expressions.
	targetPosition protocol.Position
}

// visit checks if the given call expression contains the target
// position and updates bestMatch if this is the most specific
// match found so far.
//
// Takes expression (ast_domain.Expression) which is the
// expression to check.
func (f *callExprFinder) visit(expression ast_domain.Expression) {
	if expression == nil {
		return
	}

	callExpr, ok := expression.(*ast_domain.CallExpression)
	if !ok {
		return
	}

	leftParenLocation := callExpr.LparenLocation
	rightParenLocation := callExpr.RparenLocation

	callRange := protocol.Range{
		Start: protocol.Position{
			Line:      safeconv.IntToUint32(leftParenLocation.Line - 1),
			Character: safeconv.IntToUint32(leftParenLocation.Column - 1),
		},
		End: protocol.Position{
			Line:      safeconv.IntToUint32(rightParenLocation.Line - 1),
			Character: safeconv.IntToUint32(rightParenLocation.Column),
		},
	}

	if !isPositionInRange(f.targetPosition, callRange) {
		return
	}

	if f.bestMatch == nil || isRangeSmaller(callRange, f.bestMatchRange) {
		f.bestMatch = callExpr
		f.bestMatchRange = callRange
	}
}

// commaCounter tracks state while counting top-level commas in text.
type commaCounter struct {
	// count is the number of top-level commas found.
	count int

	// depth tracks how deeply nested brackets are; commas only count at depth 0.
	depth int

	// inString indicates whether the parser is inside a string literal.
	inString bool

	// inRawString indicates whether the scanner is inside a raw string literal.
	inRawString bool

	// escapeNext indicates the next character should be treated as escaped.
	escapeNext bool
}

// processChar handles a single character in the comma counting state machine.
//
// Takes character (byte) which is the character to process.
func (c *commaCounter) processChar(character byte) {
	if c.handleEscape() {
		return
	}
	if c.handleRawString(character) {
		return
	}
	if c.handleString(character) {
		return
	}
	c.handleNesting(character)
}

// handleEscape processes the next character if it was marked for escaping.
//
// Returns bool which is true if the character was consumed by an escape
// sequence.
func (c *commaCounter) handleEscape() bool {
	if c.escapeNext {
		c.escapeNext = false
		return true
	}
	return false
}

// handleRawString processes raw string delimiters.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if inside a raw string.
func (c *commaCounter) handleRawString(character byte) bool {
	if character == '`' {
		c.inRawString = !c.inRawString
		return true
	}
	return c.inRawString
}

// handleString processes regular string delimiters.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if the character is part of a string.
func (c *commaCounter) handleString(character byte) bool {
	if character == '"' || character == '\'' {
		c.inString = !c.inString
		return true
	}
	if c.inString {
		if character == '\\' {
			c.escapeNext = true
		}
		return true
	}
	return false
}

// handleNesting tracks bracket depth and counts top-level commas.
//
// Takes character (byte) which is the character to check for nesting or comma.
func (c *commaCounter) handleNesting(character byte) {
	switch character {
	case '(', '[', '{':
		c.depth++
	case ')', ']', '}':
		if c.depth > 0 {
			c.depth--
		}
	case ',':
		if c.depth == 0 {
			c.count++
		}
	}
}

// findEnclosingCallExpr finds the innermost call expression that contains
// the given position. This is used for signature help to find which function
// call the user is typing arguments for.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template to search.
// Takes position (protocol.Position) which is the cursor position to find.
// Takes sourceContent ([]byte) which is the source text for counting parameters.
//
// Returns *ast_domain.CallExpression which is the enclosing
// call expression, or nil.
// Returns int which is the zero-based index of the active parameter.
func findEnclosingCallExpr(tree *ast_domain.TemplateAST, position protocol.Position, sourceContent []byte) (*ast_domain.CallExpression, int) {
	finder := &callExprFinder{
		targetPosition: position,
	}

	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		ast_domain.WalkNodeExpressions(node, finder.visit)
		return true
	})

	if finder.bestMatch != nil {
		activeParam := countActiveParameter(finder.bestMatch, position, sourceContent)
		return finder.bestMatch, activeParam
	}

	return nil, 0
}

// countActiveParameter finds which parameter the cursor is on within a call
// expression. It counts top-level commas in the source text between the
// opening parenthesis and the cursor position.
//
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes position (protocol.Position) which is the cursor position.
// Takes sourceContent ([]byte) which is the source file content.
//
// Returns int which is the zero-based parameter index at the cursor position.
func countActiveParameter(callExpr *ast_domain.CallExpression, position protocol.Position, sourceContent []byte) int {
	if callExpr == nil {
		return 0
	}

	leftParenLocation := callExpr.LparenLocation
	lparenPos := protocol.Position{
		Line:      safeconv.IntToUint32(leftParenLocation.Line - 1),
		Character: safeconv.IntToUint32(leftParenLocation.Column - 1),
	}

	if position.Line < lparenPos.Line || (position.Line == lparenPos.Line && position.Character <= lparenPos.Character) {
		return 0
	}

	text := extractTextBetweenPositions(sourceContent, lparenPos, position)
	if text == "" {
		return 0
	}

	commaCount := countTopLevelCommas(text)
	return commaCount
}

// extractTextBetweenPositions extracts text between two positions in the
// source content.
//
// Takes content ([]byte) which is the source content to extract from.
// Takes start (protocol.Position) which is the starting position.
// Takes end (protocol.Position) which is the ending position.
//
// Returns string which contains the extracted text, or empty if positions are
// out of bounds.
func extractTextBetweenPositions(content []byte, start protocol.Position, end protocol.Position) string {
	startOffset := positionToByteOffset(content, start)
	endOffset := positionToByteOffset(content, end)
	if startOffset < 0 || endOffset < 0 || startOffset > endOffset {
		return ""
	}
	return string(content[startOffset:endOffset])
}

// positionToByteOffset converts an LSP line/character position to a byte
// offset within the content. Returns -1 if the position is out of bounds.
//
// Takes content ([]byte) which is the raw document bytes.
// Takes position (protocol.Position) which is the LSP position to convert.
//
// Returns int which is the byte offset, or -1 if out of bounds.
func positionToByteOffset(content []byte, position protocol.Position) int {
	currentLine := uint32(0)
	lineStart := 0

	for i, b := range content {
		if b == '\n' {
			if currentLine == position.Line {
				return byteOffsetWithinLine(content, lineStart, i, position.Character)
			}
			currentLine++
			lineStart = i + 1
		}
	}

	if currentLine == position.Line {
		return byteOffsetWithinLine(content, lineStart, len(content), position.Character)
	}
	return -1
}

// byteOffsetWithinLine returns the byte offset of the given character position
// within the line delimited by [lineStart, lineEnd), stripping a trailing '\r'
// before checking bounds.
//
// Takes content ([]byte) which is the full document bytes.
// Takes lineStart (int) which is the byte index of the first byte on the line.
// Takes lineEnd (int) which is the byte index one past the last byte on the line.
// Takes character (uint32) which is the zero-based character offset requested.
//
// Returns int which is the absolute byte offset, or -1 if out of bounds.
func byteOffsetWithinLine(content []byte, lineStart, lineEnd int, character uint32) int {
	if lineEnd > lineStart && content[lineEnd-1] == '\r' {
		lineEnd--
	}
	charOffset := int(character)
	if charOffset > lineEnd-lineStart {
		return -1
	}
	return lineStart + charOffset
}

// countTopLevelCommas counts commas that are not inside nested brackets,
// braces, parentheses, or strings.
//
// Takes text (string) which is the input to scan for top-level commas.
//
// Returns int which is the number of commas found at the top level.
func countTopLevelCommas(text string) int {
	c := &commaCounter{}
	for i := range len(text) {
		c.processChar(text[i])
	}
	return c.count
}
