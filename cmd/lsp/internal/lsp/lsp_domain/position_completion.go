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
	"slices"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/typegen/typegen_domain"
)

// isAttrPrefixLen is the length of the `is="` prefix used when parsing partial
// alias attributes.
const isAttrPrefixLen = 4

// completionContext holds state about what kind of completion the user is
// asking for at a given cursor position.
type completionContext struct {
	// BaseExpression is the text before the dot when completing member access.
	BaseExpression string

	// Prefix is the text typed so far, used to filter completions.
	Prefix string

	// DirectiveType is the directive type found in the comment, if any.
	DirectiveType string

	// Namespace is the sub-namespace for piko completions (e.g., "nav", "form").
	Namespace string

	// TriggerKind indicates what caused the completion request.
	TriggerKind completionTriggerKind

	// InDirective indicates whether the cursor is inside a directive attribute.
	InDirective bool

	// InPartialAlias indicates whether we are inside an is="..." attribute.
	InPartialAlias bool
}

// completionTriggerKind shows what caused a completion request to start.
type completionTriggerKind int

const (
	// triggerScope shows that completion was triggered at the start of a scope.
	triggerScope completionTriggerKind = iota

	// triggerMemberAccess indicates a dot-triggered completion for accessing a
	// struct field or method.
	triggerMemberAccess

	// triggerPartialAlias indicates completion triggered within a partial alias.
	triggerPartialAlias

	// triggerDirective indicates completion was triggered by a directive prefix.
	triggerDirective

	// triggerDirectiveValue indicates completion inside a directive's value.
	triggerDirectiveValue

	// triggerEventHandler indicates completion was triggered for an event handler.
	triggerEventHandler

	// triggerPartialName indicates a completion triggered by a partial name match.
	triggerPartialName

	// triggerRefAccess indicates a completion triggered by accessing a reference.
	triggerRefAccess

	// triggerStateAccessJS indicates a completion trigger for state field access
	// in JavaScript expressions.
	triggerStateAccessJS

	// triggerPropsAccessJS indicates completion triggered by props access in JS.
	triggerPropsAccessJS

	// triggerPikoNamespace indicates completion triggered by piko. prefix.
	triggerPikoNamespace

	// triggerPikoSubNamespace indicates completion inside a piko sub-namespace.
	triggerPikoSubNamespace

	// triggerActionNamespace indicates completion triggered by action. prefix.
	triggerActionNamespace

	// triggerCSSClassValue indicates completion inside a class="" attribute value.
	triggerCSSClassValue
)

// analyseCompletionContext checks the document text near the cursor to work
// out what type of completion to offer.
//
// Takes d (*document) which gives access to content and SFC parsing.
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns completionContext which describes the detected completion type and
// any prefix or context information found.
func analyseCompletionContext(d *document, position protocol.Position) completionContext {
	ctx := completionContext{
		TriggerKind: triggerScope,
	}

	line, found := getLineAtPosition(d.Content, position.Line)
	if !found {
		return ctx
	}
	if int(position.Character) > len(line) {
		return ctx
	}

	textBeforeCursor := line[:position.Character]
	lineString := string(line)
	cursorPosition := int(position.Character)

	if tryPikoNamespaceContext(&ctx, textBeforeCursor) {
		return ctx
	}
	if tryActionNamespaceContext(&ctx, textBeforeCursor) {
		return ctx
	}
	if tryMemberAccessContext(&ctx, textBeforeCursor) {
		return ctx
	}
	if tryDirectiveContext(&ctx, textBeforeCursor) {
		return ctx
	}
	if tryCSSClassValueContext(&ctx, lineString, cursorPosition) {
		return ctx
	}
	if tryDirectiveValueContext(&ctx, textBeforeCursor) {
		return ctx
	}
	if tryPartialAliasContext(&ctx, lineString, cursorPosition) {
		return ctx
	}
	if tryEventHandlercompletionContext(&ctx, lineString, cursorPosition) {
		return ctx
	}
	if tryPartialNamecompletionContext(&ctx, lineString, cursorPosition) {
		return ctx
	}
	if tryRefsAccesscompletionContext(&ctx, lineString, cursorPosition) {
		return ctx
	}
	if tryStateAccesscompletionContext(&ctx, d, position, lineString, cursorPosition) {
		return ctx
	}
	if tryPropsAccesscompletionContext(&ctx, d, position, lineString, cursorPosition) {
		return ctx
	}

	return ctx
}

// tryMemberAccessContext checks for member access (contains a dot).
// Handles both "state." (cursor right after dot) and "state.us" (partial name
// after dot).
//
// Takes ctx (*completionContext) which receives the trigger kind, base
// expression, and prefix when member access is detected.
// Takes textBeforeCursor ([]byte) which is the text to check for member
// access.
//
// Returns bool which is true when member access was detected and the context
// was updated.
func tryMemberAccessContext(ctx *completionContext, textBeforeCursor []byte) bool {
	if len(textBeforeCursor) == 0 {
		return false
	}

	if textBeforeCursor[len(textBeforeCursor)-1] == '.' {
		ctx.TriggerKind = triggerMemberAccess
		ctx.BaseExpression = extractBaseExpression(textBeforeCursor[:len(textBeforeCursor)-1])
		ctx.Prefix = ""
		return true
	}

	dotIndex := findLastDotIndex(textBeforeCursor)
	if dotIndex == -1 {
		return false
	}

	partialName := textBeforeCursor[dotIndex+1:]
	if !isValidIdentifierPrefix(partialName) {
		return false
	}

	ctx.TriggerKind = triggerMemberAccess
	ctx.BaseExpression = extractBaseExpression(textBeforeCursor[:dotIndex])
	ctx.Prefix = string(partialName)
	return true
}

// findLastDotIndex finds the position of the last dot in a byte slice.
//
// Takes text ([]byte) which is the byte slice to search.
//
// Returns int which is the index of the last dot, or -1 if not found.
func findLastDotIndex(text []byte) int {
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == '.' {
			return i
		}
	}
	return -1
}

// isValidIdentifierPrefix checks whether all bytes in the given text are valid
// identifier characters.
//
// Takes text ([]byte) which is the byte slice to check.
//
// Returns bool which is true if every byte is a valid identifier character,
// or false otherwise.
func isValidIdentifierPrefix(text []byte) bool {
	for _, b := range text {
		if !isIdentChar(b) {
			return false
		}
	}
	return true
}

// tryDirectiveContext checks for directive completion patterns like "p-".
// Handles both "p-" (cursor right after the dash) and "p-sh" (partial
// directive name).
//
// Takes ctx (*completionContext) which receives the trigger kind and prefix
// if a directive pattern is found.
// Takes textBeforeCursor ([]byte) which contains the text to check for a
// directive prefix.
//
// Returns bool which is true if a directive context was found and ctx was
// updated.
func tryDirectiveContext(ctx *completionContext, textBeforeCursor []byte) bool {
	if len(textBeforeCursor) < 2 {
		return false
	}

	pDashIndex := findLastPDash(textBeforeCursor)
	if pDashIndex == -1 {
		return false
	}

	if pDashIndex > 0 && !isAttrBoundary(textBeforeCursor[pDashIndex-1]) {
		return false
	}

	afterPDash := textBeforeCursor[pDashIndex+2:]

	if len(afterPDash) > 0 && !isValidDirectivePrefix(afterPDash) {
		return false
	}

	ctx.TriggerKind = triggerDirective
	ctx.Prefix = string(afterPDash)
	return true
}

// findLastPDash finds the last "p-" in the given byte slice.
//
// Takes text ([]byte) which is the byte slice to search.
//
// Returns int which is the index of 'p' in "p-", or -1 if not found.
func findLastPDash(text []byte) int {
	for i := len(text) - 2; i >= 0; i-- {
		if text[i] == 'p' && text[i+1] == '-' {
			return i
		}
	}
	return -1
}

// isAttrBoundary reports whether a byte marks a valid boundary before an
// attribute name.
//
// Takes b (byte) which is the character to check.
//
// Returns bool which is true if b is whitespace, a newline, or a tag start
// character.
func isAttrBoundary(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '<'
}

// isValidDirectivePrefix checks if all bytes are valid directive name
// characters. Directive names can contain letters, numbers, and hyphens.
//
// Takes text ([]byte) which is the bytes to check.
//
// Returns bool which is true if all bytes are valid.
func isValidDirectivePrefix(text []byte) bool {
	for _, b := range text {
		if !isDirectiveNameChar(b) {
			return false
		}
	}
	return true
}

// isDirectiveNameChar reports whether a byte is valid in a directive name.
//
// Takes b (byte) which is the character to check.
//
// Returns bool which is true if b is a letter, digit, or hyphen.
func isDirectiveNameChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-'
}

// tryDirectiveValueContext checks if the cursor is inside a directive attribute
// value (e.g., p-if="<cursor>", p-show="sta<cursor>").
//
// Takes ctx (*completionContext) which receives the trigger kind and prefix
// when inside a directive value.
// Takes textBeforeCursor ([]byte) which contains the text to check.
//
// Returns bool which is true when the cursor is inside a directive value.
func tryDirectiveValueContext(ctx *completionContext, textBeforeCursor []byte) bool {
	index := findDirectiveValueStart(textBeforeCursor)
	if index == -1 {
		return false
	}

	afterQuote := textBeforeCursor[index:]
	prefix, insideQuotes := extractDirectiveValuePrefix(afterQuote)
	if !insideQuotes {
		return false
	}

	ctx.TriggerKind = triggerDirectiveValue
	ctx.Prefix = prefix
	ctx.InDirective = true
	return true
}

// findDirectiveValueStart finds the start position after the opening quote of
// a directive attribute value such as p-if=" or p-show=".
//
// Takes text ([]byte) which is the text to search backwards through.
//
// Returns int which is the index after the opening quote, or -1 if not found.
func findDirectiveValueStart(text []byte) int {
	for i := len(text) - 1; i >= 2; i-- {
		if text[i] == '"' && text[i-1] == '=' {
			if isDirectiveAttribute(text[:i-1]) {
				return i + 1
			}
		}
	}
	return -1
}

// isDirectiveAttribute checks if the text ends with a directive attribute name
// such as "p-if", "p-show", "p-for", or "p-bind:class".
//
// Takes text ([]byte) which is the byte slice to check.
//
// Returns bool which is true if the text ends with a "p-" directive name.
func isDirectiveAttribute(text []byte) bool {
	end := len(text)
	start := end

	for start > 0 {
		character := text[start-1]
		if !isDirectiveNameChar(character) && character != ':' {
			break
		}
		start--
	}

	if start >= end {
		return false
	}

	attributeName := text[start:end]

	if len(attributeName) < 2 {
		return false
	}
	if attributeName[0] != 'p' || attributeName[1] != '-' {
		return false
	}

	if start > 0 && !isAttrBoundary(text[start-1]) {
		return false
	}

	return true
}

// extractDirectiveValuePrefix extracts the text typed so far inside a
// directive value, stopping at any closing quote.
//
// Takes text ([]byte) which is the text after the opening quote.
//
// Returns string which is the prefix to use for completion.
// Returns bool which is true if the cursor is inside the quotes, meaning no
// closing quote was found.
func extractDirectiveValuePrefix(text []byte) (string, bool) {
	if slices.Contains(text, '"') {
		return "", false
	}
	return string(text), true
}

// tryPartialAliasContext checks if the cursor is inside an is="..." attribute
// and sets up the completion context for partial alias matching.
//
// Takes ctx (*completionContext) which receives the trigger kind and prefix
// when a partial alias is found.
// Takes lineString (string) which contains the current line text.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns bool which is true when the cursor is inside an is="..." attribute.
func tryPartialAliasContext(ctx *completionContext, lineString string, cursorPosition int) bool {
	index := findLastOccurrence(lineString[:cursorPosition], `is="`)
	if index == -1 {
		return false
	}
	textBetween := lineString[index+isAttrPrefixLen : cursorPosition]
	if hasClosingQuote(textBetween) {
		return false
	}
	ctx.TriggerKind = triggerPartialAlias
	ctx.Prefix = textBetween
	return true
}

// tryEventHandlercompletionContext checks if the current line matches an event
// handler pattern and updates the completion context if found.
//
// Takes ctx (*completionContext) which receives updates when a match is found.
// Takes lineString (string) which contains the current line of code.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns bool which is true when an event handler context was found.
func tryEventHandlercompletionContext(ctx *completionContext, lineString string, cursorPosition int) bool {
	index, prefix := findEventHandlerContext(lineString, cursorPosition)
	if index == -1 {
		return false
	}
	ctx.TriggerKind = triggerEventHandler
	ctx.Prefix = prefix
	return true
}

// tryPartialNamecompletionContext checks if there is a partial name at the
// cursor that can be completed.
//
// Takes ctx (*completionContext) which stores the completion trigger details.
// Takes lineString (string) which contains the current line of text.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns bool which is true when a partial name context was found.
func tryPartialNamecompletionContext(ctx *completionContext, lineString string, cursorPosition int) bool {
	index, prefix := findPartialNameContext(lineString, cursorPosition)
	if index == -1 {
		return false
	}
	ctx.TriggerKind = triggerPartialName
	ctx.Prefix = prefix
	return true
}

// tryRefsAccesscompletionContext checks if a line contains refs access context.
//
// Takes ctx (*completionContext) which receives the trigger kind and prefix
// when refs access context is found.
// Takes lineString (string) which contains the line of text to check.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns bool which indicates whether refs access context was found.
func tryRefsAccesscompletionContext(ctx *completionContext, lineString string, cursorPosition int) bool {
	index, prefix := findRefsAccessContext(lineString, cursorPosition)
	if index == -1 {
		return false
	}
	ctx.TriggerKind = triggerRefAccess
	ctx.Prefix = prefix
	return true
}

// tryStateAccesscompletionContext checks for state access in a JavaScript
// script block.
//
// Takes ctx (*completionContext) which receives the completion context to
// populate if a state access is found.
// Takes d (*document) which provides access to SFC parsing for script
// detection.
// Takes position (protocol.Position) which specifies the cursor position in the
// document.
// Takes lineString (string) which contains the current line text.
// Takes cursorPosition (int) which indicates the cursor position within the line.
//
// Returns bool which is true when a state access context was found and the
// context was populated.
func tryStateAccesscompletionContext(ctx *completionContext, d *document, position protocol.Position, lineString string, cursorPosition int) bool {
	index, prefix := findStateAccessContext(lineString, cursorPosition)
	if index == -1 {
		return false
	}
	if !d.isPositionInClientScript(position) {
		return false
	}
	ctx.TriggerKind = triggerStateAccessJS
	ctx.Prefix = prefix
	return true
}

// tryPropsAccesscompletionContext checks for props access in a JavaScript
// script block.
//
// Takes ctx (*completionContext) which receives the completion context to
// update if props access is found.
// Takes d (*document) which provides access to SFC parsing to find the script.
// Takes position (protocol.Position) which specifies the cursor position.
// Takes lineString (string) which contains the current line text.
// Takes cursorPosition (int) which indicates the cursor offset within the line.
//
// Returns bool which is true if props access context was found and ctx was
// updated.
func tryPropsAccesscompletionContext(ctx *completionContext, d *document, position protocol.Position, lineString string, cursorPosition int) bool {
	index, prefix := findPropsAccessContext(lineString, cursorPosition)
	if index == -1 {
		return false
	}
	if !d.isPositionInClientScript(position) {
		return false
	}
	ctx.TriggerKind = triggerPropsAccessJS
	ctx.Prefix = prefix
	return true
}

// splitLines splits byte content into separate lines.
//
// Takes content ([]byte) which is the raw bytes to split.
//
// Returns [][]byte which contains each line with newline characters removed.
func splitLines(content []byte) [][]byte {
	var lines [][]byte
	var currentLine []byte

	for _, b := range content {
		if b == '\n' {
			lines = append(lines, currentLine)
			currentLine = nil
		} else if b != '\r' {
			currentLine = append(currentLine, b)
		}
	}
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

// getLineAtPosition returns the line at the given zero-based line number as a
// sub-slice of the original content. This is zero-copy and does not allocate.
//
// Takes content ([]byte) which is the raw document bytes.
// Takes lineNumber (uint32) which is the zero-based line index.
//
// Returns ([]byte, bool) where the byte slice is the line content without
// newline characters, and the bool indicates whether the line was found.
func getLineAtPosition(content []byte, lineNumber uint32) ([]byte, bool) {
	currentLine := uint32(0)
	lineStart := 0

	for i, b := range content {
		if b == '\n' {
			if currentLine == lineNumber {
				end := i
				if end > lineStart && content[end-1] == '\r' {
					end--
				}
				return content[lineStart:end], true
			}
			currentLine++
			lineStart = i + 1
		}
	}

	if currentLine == lineNumber {
		end := len(content)
		if end > lineStart && content[end-1] == '\r' {
			end--
		}
		return content[lineStart:end], true
	}
	return nil, false
}

// extractBaseExpression walks backwards from a position to find the base
// expression before a dot. It handles nested member access, array or slice
// indexing, and function calls.
//
// Takes text ([]byte) which contains the source text to extract from.
//
// Returns string which is the base expression, or empty if none is found.
func extractBaseExpression(text []byte) string {
	if len(text) == 0 {
		return ""
	}

	startIndex := findExpressionStart(text)
	if startIndex < len(text) {
		return string(text[startIndex:])
	}
	return ""
}

// findExpressionStart walks backwards through text to find where an expression
// begins. It tracks nesting depth to handle brackets and parentheses.
//
// Takes text ([]byte) which contains the source text to scan backwards.
//
// Returns int which is the index where the expression starts.
func findExpressionStart(text []byte) int {
	i := len(text) - 1
	depth := 0

	for i >= 0 && i < len(text) { // #nosec G602 -- i is always in bounds; explicit check satisfies static analysis
		character := text[i]

		switch character {
		case ')', ']':
			depth++
			i--
		case '(', '[':
			if depth == 0 {
				return i + 1
			}
			depth--
			i--
		case '.':
			i--
		default:
			if depth == 0 && !isIdentChar(character) {
				return i + 1
			}
			i--
		}
	}

	return 0
}

// isIdentChar reports whether a byte is a valid identifier character.
//
// Takes b (byte) which is the byte to check.
//
// Returns bool which is true if b is a letter, digit, or underscore.
func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// findLastOccurrence finds the last position of a substring within a string.
//
// Takes s (string) which is the string to search within.
// Takes substr (string) which is the substring to find.
//
// Returns int which is the index of the last match, or -1 if not found.
func findLastOccurrence(s, substr string) int {
	index := -1
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			index = i
		}
	}
	return index
}

// findEventHandlerContext checks if the cursor is inside a p-on:*="" attribute
// value.
//
// Takes line (string) which contains the text to search.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns int which is the start index of the handler value, or -1 if not
// found.
// Returns string which is the text after the pattern up to the cursor.
func findEventHandlerContext(line string, cursorPosition int) (int, string) {
	patterns := []string{
		`p-on:click="`, `p-on:change="`, `p-on:submit="`, `p-on:input="`,
		`p-on:focus="`, `p-on:blur="`, `p-on:keydown="`, `p-on:keyup="`,
		`p-on:mouseenter="`, `p-on:mouseleave="`, `p-on:scroll="`,
	}

	for _, pattern := range patterns {
		index := findLastOccurrence(line[:cursorPosition], pattern)
		if index == -1 {
			continue
		}

		afterPattern := line[index+len(pattern) : cursorPosition]
		if hasClosingQuote(afterPattern) {
			continue
		}

		return index + len(pattern), afterPattern
	}

	return -1, ""
}

// hasClosingQuote checks whether the text contains a double quote character.
//
// Takes text (string) which is the text to search.
//
// Returns bool which is true if a double quote is found.
func hasClosingQuote(text string) bool {
	for i := range len(text) {
		if text[i] == '"' {
			return true
		}
	}
	return false
}

// findPartialNameContext checks if the cursor is inside a reloadPartial or
// reloadGroup call and extracts the partial name being typed.
//
// Takes line (string) which is the text line to search within.
// Takes cursorPosition (int) which is the cursor position within the line.
//
// Returns int which is the start index of the partial name, or -1 if not found.
// Returns string which is the text typed so far, or empty if not found.
func findPartialNameContext(line string, cursorPosition int) (int, string) {
	patterns := []string{`reloadPartial('`, `reloadPartial("`, `reloadGroup('`, `reloadGroup("`}

	for _, pattern := range patterns {
		index := findLastOccurrence(line[:cursorPosition], pattern)
		if index == -1 {
			continue
		}

		quoteChar := pattern[len(pattern)-1]
		afterPattern := line[index+len(pattern) : cursorPosition]

		hasClose := false
		for i := range len(afterPattern) {
			if afterPattern[i] == quoteChar {
				hasClose = true
				break
			}
		}

		if !hasClose {
			return index + len(pattern), afterPattern
		}
	}

	return -1, ""
}

// findRefsAccessContext checks if the cursor is after "refs." in a JavaScript
// context.
//
// Takes line (string) which contains the text to search.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns int which is the start index of the prefix, or -1 if not found.
// Returns string which is the prefix after "refs.", or empty if not found.
func findRefsAccessContext(line string, cursorPosition int) (int, string) {
	pattern := "refs."
	index := findLastOccurrence(line[:cursorPosition], pattern)
	if index == -1 {
		return -1, ""
	}

	prefixStart := index + len(pattern)
	if prefixStart > cursorPosition {
		return -1, ""
	}

	prefix := line[prefixStart:cursorPosition]

	for i := range len(prefix) {
		if !isIdentChar(prefix[i]) {
			return -1, ""
		}
	}

	return prefixStart, prefix
}

// buildExpressionRangeMap creates a map of all expressions in the AST to their
// absolute LSP ranges. DocumentHighlight and References use this map to get
// correct absolute positions.
//
// Takes tree (*ast_domain.TemplateAST) which contains the parsed template.
// Takes docPath (string) which specifies the document path to filter by.
//
// Returns map[ast_domain.Expression]protocol.Range which maps each expression
// to its absolute LSP range.
func buildExpressionRangeMap(tree *ast_domain.TemplateAST, docPath string) map[ast_domain.Expression]protocol.Range {
	rangeMap := make(map[ast_domain.Expression]protocol.Range)

	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.GoAnnotations == nil || node.GoAnnotations.OriginalSourcePath == nil {
			return true
		}
		if *node.GoAnnotations.OriginalSourcePath != docPath {
			return true
		}

		processExpressionsInNode(node, rangeMap)
		return true
	})

	return rangeMap
}

// processExpressionsInNode finds all expressions in a template node and
// records their positions in the document.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes rangeMap (map[ast_domain.Expression]protocol.Range) which stores the
// position range for each expression found.
func processExpressionsInNode(node *ast_domain.TemplateNode, rangeMap map[ast_domain.Expression]protocol.Range) {
	processNodeDynamicAttrs(node, rangeMap)
	processNodeDirectives(node, rangeMap)
	processNodeRichText(node, rangeMap)
}

// processNodeDynamicAttrs adds all dynamic attribute expressions from a node
// to the range map.
//
// Takes node (*ast_domain.TemplateNode) which holds the dynamic attributes to
// process.
// Takes rangeMap (map[ast_domain.Expression]protocol.Range) which maps
// expressions to their source ranges.
func processNodeDynamicAttrs(node *ast_domain.TemplateNode, rangeMap map[ast_domain.Expression]protocol.Range) {
	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		addExpressionTreeToRangeMap(attr.Expression, attr.Location, rangeMap)
	}
}

// processNodeDirectives adds all directive expressions from a template node
// to the range map.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes rangeMap (map[ast_domain.Expression]protocol.Range) which maps each
// expression to its position in the source file.
func processNodeDirectives(node *ast_domain.TemplateNode, rangeMap map[ast_domain.Expression]protocol.Range) {
	for _, directive := range [...]*ast_domain.Directive{
		node.DirIf, node.DirElseIf, node.DirFor, node.DirShow, node.DirModel,
		node.DirText, node.DirHTML, node.DirClass, node.DirStyle,
		node.DirKey, node.DirContext, node.DirScaffold,
	} {
		if directive != nil && !directive.AttributeRange.Start.IsSynthetic() {
			addExpressionTreeToRangeMap(directive.Expression, directive.Location, rangeMap)
		}
	}
	for _, directive := range node.Binds {
		if directive != nil && !directive.AttributeRange.Start.IsSynthetic() {
			addExpressionTreeToRangeMap(directive.Expression, directive.Location, rangeMap)
		}
	}
	for _, directives := range node.OnEvents {
		for i := range directives {
			directive := &directives[i]
			if !directive.AttributeRange.Start.IsSynthetic() {
				addExpressionTreeToRangeMap(directive.Expression, directive.Location, rangeMap)
			}
		}
	}
}

// collectNodeDirectives gathers all directives from a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to collect
// directives from.
//
// Returns []*ast_domain.Directive which contains all directives found on the
// node. This includes conditional, loop, binding, and event directives.
func collectNodeDirectives(node *ast_domain.TemplateNode) []*ast_domain.Directive {
	directives := []*ast_domain.Directive{
		node.DirIf, node.DirElseIf, node.DirFor, node.DirShow, node.DirModel,
		node.DirText, node.DirHTML, node.DirClass, node.DirStyle,
		node.DirKey, node.DirContext, node.DirScaffold,
	}

	for _, directive := range node.Binds {
		directives = append(directives, directive)
	}

	for _, dirs := range node.OnEvents {
		for i := range dirs {
			directives = append(directives, &dirs[i])
		}
	}

	return directives
}

// processNodeRichText adds all rich text expressions to the range map.
//
// Takes node (*ast_domain.TemplateNode) which contains the rich text parts to
// process.
// Takes rangeMap (map[ast_domain.Expression]protocol.Range) which maps
// expressions to their source ranges.
func processNodeRichText(node *ast_domain.TemplateNode, rangeMap map[ast_domain.Expression]protocol.Range) {
	for i := range node.RichText {
		part := &node.RichText[i]
		if !part.IsLiteral {
			addExpressionTreeToRangeMap(part.Expression, part.Location, rangeMap)
		}
	}
}

// addExpressionTreeToRangeMap adds an expression and all its
// child nodes to the range map.
//
// Takes expression (ast_domain.Expression) which is the root
// expression to process.
// Takes baseLocation (ast_domain.Location) which provides
// the base position for working out ranges.
// Takes rangeMap (map[ast_domain.Expression]protocol.Range)
// which stores the worked out range for each expression.
func addExpressionTreeToRangeMap(expression ast_domain.Expression, baseLocation ast_domain.Location, rangeMap map[ast_domain.Expression]protocol.Range) {
	visitExpressionTree(expression, func(e ast_domain.Expression) {
		rangeMap[e] = calculateExpressionRange(e, baseLocation)
	})
}

// findStateAccessContext checks if the cursor is after "state." in a
// JavaScript context.
//
// Takes line (string) which contains the text to search within.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns int which is the start index of the prefix, or -1 if not found.
// Returns string which is the prefix after "state.", or empty if not found.
func findStateAccessContext(line string, cursorPosition int) (int, string) {
	pattern := "state."
	index := findLastOccurrence(line[:cursorPosition], pattern)
	if index == -1 {
		return -1, ""
	}

	prefixStart := index + len(pattern)
	if prefixStart > cursorPosition {
		return -1, ""
	}

	prefix := line[prefixStart:cursorPosition]

	for i := range len(prefix) {
		if !isIdentChar(prefix[i]) {
			return -1, ""
		}
	}

	return prefixStart, prefix
}

// findPropsAccessContext checks if the cursor is after "props." in a
// JavaScript context.
//
// Takes line (string) which contains the source line to search.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns int which is the start index of the prefix, or -1 if not found.
// Returns string which is the prefix after "props.", or empty if not found.
func findPropsAccessContext(line string, cursorPosition int) (int, string) {
	pattern := "props."
	index := findLastOccurrence(line[:cursorPosition], pattern)
	if index == -1 {
		return -1, ""
	}

	prefixStart := index + len(pattern)
	if prefixStart > cursorPosition {
		return -1, ""
	}

	prefix := line[prefixStart:cursorPosition]

	for i := range len(prefix) {
		if !isIdentChar(prefix[i]) {
			return -1, ""
		}
	}

	return prefixStart, prefix
}

// tryPikoNamespaceContext checks for piko namespace completion contexts.
// Handles "piko.", "piko.na", "piko.nav.", and "piko.nav.na" patterns.
//
// Takes ctx (*completionContext) which receives the trigger kind and namespace.
// Takes textBeforeCursor ([]byte) which is the text before the cursor.
//
// Returns bool which is true when a piko namespace context was detected.
func tryPikoNamespaceContext(ctx *completionContext, textBeforeCursor []byte) bool {
	for _, namespace := range typegen_domain.PikoSubNamespaces {
		pattern := "piko." + namespace + "."
		if index := findPatternEnd(textBeforeCursor, pattern); index != -1 {
			prefix := string(textBeforeCursor[index:])
			if isValidIdentifierPrefix([]byte(prefix)) {
				ctx.TriggerKind = triggerPikoSubNamespace
				ctx.Namespace = namespace
				ctx.Prefix = prefix
				return true
			}
		}
	}

	pikoPattern := "piko."
	if index := findPatternEnd(textBeforeCursor, pikoPattern); index != -1 {
		prefix := string(textBeforeCursor[index:])
		if isValidIdentifierPrefix([]byte(prefix)) {
			ctx.TriggerKind = triggerPikoNamespace
			ctx.Prefix = prefix
			return true
		}
	}

	return false
}

// tryActionNamespaceContext checks for action namespace completion contexts.
// It handles patterns like "action." and "action.cus".
//
// Takes ctx (*completionContext) which receives the trigger kind and prefix.
// Takes textBeforeCursor ([]byte) which is the text before the cursor.
//
// Returns bool which is true when an action namespace context was found.
func tryActionNamespaceContext(ctx *completionContext, textBeforeCursor []byte) bool {
	actionPattern := "action."
	if index := findPatternEnd(textBeforeCursor, actionPattern); index != -1 {
		prefix := string(textBeforeCursor[index:])
		if isValidActionPrefix([]byte(prefix)) {
			ctx.TriggerKind = triggerActionNamespace
			ctx.Prefix = prefix
			return true
		}
	}

	return false
}

// findPatternEnd searches for a pattern in text and returns the position just
// after the match. The search works backwards from the end of the text and
// checks that the pattern starts at a word boundary.
//
// Takes text ([]byte) which is the text to search through.
// Takes pattern (string) which is the pattern to find.
//
// Returns int which is the index after the pattern, or -1 if not found.
func findPatternEnd(text []byte, pattern string) int {
	patternBytes := []byte(pattern)
	for i := len(text) - len(patternBytes); i >= 0; i-- {
		match := true
		for j := range len(patternBytes) {
			if text[i+j] != patternBytes[j] {
				match = false
				break
			}
		}
		if match {
			if i > 0 && isIdentChar(text[i-1]) {
				continue
			}
			return i + len(patternBytes)
		}
	}
	return -1
}

// isValidActionPrefix checks whether all bytes are valid for an action name
// prefix. Action names may contain letters, numbers, underscores, and dots.
//
// Takes text ([]byte) which is the bytes to check.
//
// Returns bool which is true if all bytes are valid action name characters.
func isValidActionPrefix(text []byte) bool {
	for _, b := range text {
		if !isIdentChar(b) && b != '.' {
			return false
		}
	}
	return true
}
