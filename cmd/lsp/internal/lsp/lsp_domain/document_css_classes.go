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
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/wdk/safeconv"
)

// cssClassDefinition holds a CSS class name and its absolute position in the
// document. Positions are 0-based for direct use as LSP protocol values.
type cssClassDefinition struct {
	// Name is the CSS class name without the leading dot.
	Name string

	// Line is the 0-based line number in the document.
	Line int

	// Column is the 0-based column where the dot prefix starts.
	Column int

	// EndColumn is the 0-based column after the class name ends.
	EndColumn int
}

// cssClassMatch holds a matched class selector within a CSS content string.
type cssClassMatch struct {
	// Name is the class name without the leading dot.
	Name string

	// DotOffset is the byte offset of the dot character in the content.
	DotOffset int
}

// pClassShorthandPrefix is the text before the class name in a p-class
// shorthand attribute.
const pClassShorthandPrefix = "p-class:"

// findCSSClassDefinitions scans all style blocks for CSS class selectors and
// returns a map of class name to its first definition position.
//
// Returns map[string]cssClassDefinition which maps each class name to the
// position of its first occurrence across all style blocks.
func (d *document) findCSSClassDefinitions() map[string]cssClassDefinition {
	if d.isPKCFile() {
		meta := d.getPKCMetadata()
		if meta == nil {
			return nil
		}
		return meta.CSSClasses
	}

	if d.AnnotationResult == nil || d.AnnotationResult.EntryPointStyleBlocks == nil {
		return nil
	}

	styleBlocks, ok := d.AnnotationResult.EntryPointStyleBlocks.([]sfcparser.Style)
	if !ok {
		return nil
	}

	result := make(map[string]cssClassDefinition)

	for _, block := range styleBlocks {
		if block.Content == "" {
			continue
		}
		collectClassDefinitions(block, result)
	}

	return result
}

// findCSSClassDefinitionByName finds a specific CSS class definition by name
// in the style blocks.
//
// Takes className (string) which is the class name to search for.
//
// Returns *cssClassDefinition which holds the position of the class, or nil
// if not found.
func (d *document) findCSSClassDefinitionByName(className string) *cssClassDefinition {
	definitions := d.findCSSClassDefinitions()
	if definitions == nil {
		return nil
	}
	definition, exists := definitions[className]
	if !exists {
		return nil
	}
	return &definition
}

// findCSSClassDefinitionLocation finds the location of a CSS class definition
// in the style blocks and returns it as an LSP location.
//
// Takes className (string) which is the class name to search for.
//
// Returns []protocol.Location which contains the definition location.
// Returns error which is always nil.
func (d *document) findCSSClassDefinitionLocation(className string) ([]protocol.Location, error) {
	definition := d.findCSSClassDefinitionByName(className)
	if definition == nil {
		return []protocol.Location{}, nil
	}

	return []protocol.Location{{
		URI: d.URI,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(definition.Line),
				Character: safeconv.IntToUint32(definition.Column),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(definition.Line),
				Character: safeconv.IntToUint32(definition.EndColumn),
			},
		},
	}}, nil
}

// getCSSClassCompletions returns completion suggestions for CSS class names
// defined in the style blocks, filtered by the given prefix.
//
// Takes prefix (string) which filters class names to those containing this
// substring (case-insensitive). An empty string means no filtering.
//
// Returns *protocol.CompletionList which contains the matching CSS class
// completions.
// Returns error which is always nil.
func (d *document) getCSSClassCompletions(prefix string) (*protocol.CompletionList, error) {
	definitions := d.findCSSClassDefinitions()
	if definitions == nil {
		return emptyCompletionList(), nil
	}

	items := make([]protocol.CompletionItem, 0, len(definitions))
	for name := range definitions {
		if prefix != "" && !containsSubstring(name, prefix) {
			continue
		}
		items = append(items, protocol.CompletionItem{
			Label:  name,
			Kind:   protocol.CompletionItemKindValue,
			Detail: "CSS class",
		})
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// checkCSSClassDefinitionContext checks whether the cursor is on a CSS class
// name reference in the template section.
//
// Handles four patterns:
//   - class="foo bar" - static class attribute, extracts word under cursor
//   - p-class:active="..." - shorthand directive, extracts suffix
//   - p-class="{ 'x': true }" - string literal in directive value
//   - :class="'x'" - string literal in bind value
//
// Takes line (string) which is the current line of text being analysed.
// Takes cursor (int) which is the character position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKDefinitionContext which provides the CSS class definition context
// if found, or nil if the cursor is not on a class reference.
func (*document) checkCSSClassDefinitionContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	if ctx := tryExtractStaticClassContext(line, cursor, position); ctx != nil {
		return ctx
	}
	if ctx := tryExtractPClassShorthandContext(line, cursor, position); ctx != nil {
		return ctx
	}
	return tryExtractDirectiveClassStringContext(line, cursor, position)
}

// collectClassDefinitions scans a single style block for CSS class selectors
// and adds them to the result map. Only the first occurrence of each class
// name is recorded.
//
// Takes block (sfcparser.Style) which provides the CSS content and its base
// position in the source file.
// Takes result (map[string]cssClassDefinition) which collects the found class
// definitions.
func collectClassDefinitions(block sfcparser.Style, result map[string]cssClassDefinition) {
	baseLine := block.ContentLocation.Line - 1
	baseCol := block.ContentLocation.Column - 1

	matches := scanCSSClassSelectors(block.Content)
	for _, match := range matches {
		if _, exists := result[match.Name]; exists {
			continue
		}

		relLine, relCol := convertCharPosToLineColumn(block.Content, match.DotOffset)
		absLine := baseLine + relLine
		absCol := relCol
		if relLine == 0 {
			absCol += baseCol
		}

		result[match.Name] = cssClassDefinition{
			Name:      match.Name,
			Line:      absLine,
			Column:    absCol,
			EndColumn: absCol + 1 + len(match.Name),
		}
	}
}

// scanCSSClassSelectors scans CSS content for class selectors (.className)
// and returns all matches with their byte offsets.
//
// Skips dots inside comments (/* ... */), single-quoted strings, and
// double-quoted strings. Also skips dots preceded by identifier characters
// or digits to avoid matching numeric values like 0.5em.
//
// Takes content (string) which is the raw CSS text to scan.
//
// Returns []cssClassMatch which contains all class selector matches found.
func scanCSSClassSelectors(content string) []cssClassMatch {
	var matches []cssClassMatch
	i := 0

	for i < len(content) {
		character := content[i]

		if character == '/' && i+1 < len(content) && content[i+1] == '*' {
			i = skipBlockComment(content, i)
			continue
		}

		if character == '\'' || character == '"' {
			i = skipString(content, i, character)
			continue
		}

		if character == '.' {
			if match := tryMatchClassSelector(content, i); match != nil {
				matches = append(matches, *match)
				i += 1 + len(match.Name)
				continue
			}
		}

		i++
	}

	return matches
}

// skipBlockComment advances past a CSS block comment (/* ... */).
//
// Takes content (string) which is the CSS text being scanned.
// Takes position (int) which is the position of the opening slash.
//
// Returns int which is the position after the closing */, or the end of
// content if no closing is found.
func skipBlockComment(content string, position int) int {
	i := position + 2
	for i < len(content)-1 {
		if content[i] == '*' && content[i+1] == '/' {
			return i + 2
		}
		i++
	}
	return len(content)
}

// skipString advances past a quoted string in CSS content.
//
// Takes content (string) which is the CSS text being scanned.
// Takes position (int) which is the position of the opening quote.
// Takes quote (byte) which is the quote character to match.
//
// Returns int which is the position after the closing quote, or the end of
// content if no closing quote is found.
func skipString(content string, position int, quote byte) int {
	i := position + 1
	for i < len(content) {
		if content[i] == '\\' {
			i += 2
			continue
		}
		if content[i] == quote {
			return i + 1
		}
		i++
	}
	return len(content)
}

// tryMatchClassSelector checks whether a dot at the given position starts a
// valid CSS class selector.
//
// A valid class selector requires:
// - The dot is not preceded by an identifier character or digit
// - The dot is followed by a letter or underscore (start of identifier)
//
// Takes content (string) which is the CSS text being scanned.
// Takes dotPosition (int) which is the position of the dot character.
//
// Returns *cssClassMatch which holds the matched class name, or nil if the
// dot does not start a valid class selector.
func tryMatchClassSelector(content string, dotPosition int) *cssClassMatch {
	if dotPosition > 0 && isDigit(content[dotPosition-1]) {
		return nil
	}

	nameStart := dotPosition + 1
	if nameStart >= len(content) {
		return nil
	}

	first := content[nameStart]
	if !isCSSIdentStart(first) {
		return nil
	}

	nameEnd := nameStart + 1
	for nameEnd < len(content) && isCSSIdentChar(content[nameEnd]) {
		nameEnd++
	}

	return &cssClassMatch{
		Name:      content[nameStart:nameEnd],
		DotOffset: dotPosition,
	}
}

// isCSSIdentStart reports whether a byte can start a CSS identifier name.
// CSS identifiers start with a letter, underscore, or hyphen (followed by
// another valid character, but we check only the first byte here).
//
// Takes b (byte) which is the character to check.
//
// Returns bool which is true if b can start a CSS identifier.
func isCSSIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || b == '-'
}

// isCSSIdentChar reports whether a byte is valid inside a CSS identifier.
//
// Takes b (byte) which is the character to check.
//
// Returns bool which is true if b is a letter, digit, underscore, or hyphen.
func isCSSIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '_' || b == '-'
}

// isDigit reports whether a byte is a decimal digit. Used to check the
// character before a dot to exclude numeric values like 0.5em while allowing
// compound selectors like div.active.
//
// Takes b (byte) which is the character to check.
//
// Returns bool which is true if b is a digit (0-9).
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// tryExtractStaticClassContext checks for class="foo bar" and extracts the
// word under the cursor.
//
// Takes line (string) which is the current line of text.
// Takes cursor (int) which is the cursor position in the line.
// Takes position (protocol.Position) which is the LSP position.
//
// Returns *PKDefinitionContext which holds the class name, or nil if the
// cursor is not inside a static class attribute value.
func tryExtractStaticClassContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	for _, pattern := range []string{`class="`, `class='`} {
		index := findLastOccurrence(line[:min(cursor+patternSearchRadius, len(line))], pattern)
		if index == -1 || index > cursor {
			continue
		}

		if index > 0 && line[index-1] != ' ' && line[index-1] != '\t' && line[index-1] != '<' {
			continue
		}

		quoteChar := pattern[len(pattern)-1]
		valueStart := index + len(pattern)
		valueEnd := findQuoteEndPositionNav(line, valueStart, quoteChar)
		if valueEnd == -1 || cursor < valueStart || cursor > valueEnd {
			continue
		}

		className := findWordAtOffset(line[valueStart:valueEnd], cursor-valueStart)
		if className == "" {
			continue
		}

		return &PKDefinitionContext{
			Kind:     PKDefCSSClass,
			Name:     className,
			Position: position,
		}
	}
	return nil
}

// findWordAtOffset finds the whitespace-delimited word at the given offset
// within a string.
//
// Takes text (string) which is the text to search in.
// Takes offset (int) which is the character offset within the text.
//
// Returns string which is the word at the offset, or empty if the offset is
// on whitespace or out of bounds.
func findWordAtOffset(text string, offset int) string {
	if offset < 0 || offset > len(text) {
		return ""
	}
	if offset == len(text) {
		offset--
	}
	if offset < 0 || text[offset] == ' ' || text[offset] == '\t' {
		return ""
	}

	start := offset
	for start > 0 && text[start-1] != ' ' && text[start-1] != '\t' {
		start--
	}
	end := offset
	for end < len(text) && text[end] != ' ' && text[end] != '\t' {
		end++
	}
	return text[start:end]
}

// tryExtractPClassShorthandContext checks for p-class:active="..." and
// extracts the class name suffix.
//
// Takes line (string) which is the current line of text.
// Takes cursor (int) which is the cursor position in the line.
// Takes position (protocol.Position) which is the LSP position.
//
// Returns *PKDefinitionContext which holds the class name, or nil if the
// cursor is not on a p-class shorthand attribute.
func tryExtractPClassShorthandContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	index := findLastOccurrence(line[:min(cursor+patternSearchRadius, len(line))], pClassShorthandPrefix)
	if index == -1 || index > cursor {
		return nil
	}

	nameStart := index + len(pClassShorthandPrefix)
	nameEnd := nameStart
	for nameEnd < len(line) && isCSSIdentChar(line[nameEnd]) {
		nameEnd++
	}

	if nameStart == nameEnd {
		return nil
	}

	if cursor < index || cursor > nameEnd {
		return nil
	}

	return &PKDefinitionContext{
		Kind:     PKDefCSSClass,
		Name:     line[nameStart:nameEnd],
		Position: position,
	}
}

// tryExtractDirectiveClassStringContext checks for p-class="{ 'x': true }"
// and :class="'x'" patterns. Extracts the string literal under the cursor.
//
// Takes line (string) which is the current line of text.
// Takes cursor (int) which is the cursor position in the line.
// Takes position (protocol.Position) which is the LSP position.
//
// Returns *PKDefinitionContext which holds the class name, or nil if the
// cursor is not on a string literal inside a class directive value.
func tryExtractDirectiveClassStringContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	for _, pattern := range []string{`p-class="`, `:class="`} {
		index := findLastOccurrence(line[:min(cursor+patternSearchRadius, len(line))], pattern)
		if index == -1 || index > cursor {
			continue
		}

		valueStart := index + len(pattern)
		valueEnd := findQuoteEndPositionNav(line, valueStart, '"')
		if valueEnd == -1 || cursor < valueStart || cursor > valueEnd {
			continue
		}

		className := extractStringLiteralAtCursor(line[valueStart:valueEnd], cursor-valueStart)
		if className == "" {
			continue
		}

		return &PKDefinitionContext{
			Kind:     PKDefCSSClass,
			Name:     className,
			Position: position,
		}
	}
	return nil
}

// extractStringLiteralAtCursor finds the single-quoted string literal under
// the cursor within a directive value.
//
// Takes text (string) which is the directive value content.
// Takes cursor (int) which is the cursor position within the text.
//
// Returns string which is the string literal content, or empty if the cursor
// is not inside a single-quoted string.
func extractStringLiteralAtCursor(text string, cursor int) string {
	if cursor < 0 || cursor >= len(text) {
		return ""
	}

	start := -1
	for i := range len(text) {
		if text[i] == '\'' {
			if start == -1 {
				start = i + 1
			} else {
				if cursor >= start && cursor < i {
					return text[start:i]
				}
				start = -1
			}
		}
	}
	return ""
}

// tryCSSClassValueContext checks if the cursor is inside a class="..."
// attribute value and sets up the completion context for CSS class matching.
//
// Takes ctx (*completionContext) which receives the trigger kind and prefix
// when a class attribute value is found.
// Takes lineString (string) which contains the current line text.
// Takes cursorPosition (int) which specifies the cursor position in the line.
//
// Returns bool which is true when the cursor is inside a class="..." value.
func tryCSSClassValueContext(ctx *completionContext, lineString string, cursorPosition int) bool {
	for _, pattern := range []string{`class="`, `class='`} {
		index := findLastOccurrence(lineString[:cursorPosition], pattern)
		if index == -1 {
			continue
		}

		if index > 0 && lineString[index-1] != ' ' && lineString[index-1] != '\t' && lineString[index-1] != '<' {
			continue
		}

		quoteChar := pattern[len(pattern)-1]
		valueStart := index + len(pattern)
		textBetween := lineString[valueStart:cursorPosition]

		if containsByte(textBetween, quoteChar) {
			continue
		}

		ctx.TriggerKind = triggerCSSClassValue
		ctx.Prefix = extractLastWord(textBetween)
		return true
	}
	return false
}

// containsByte reports whether s contains the byte c.
//
// Takes s (string) which is the string to search within.
// Takes c (byte) which is the byte to search for.
//
// Returns bool which is true if c is found in s, false otherwise.
func containsByte(s string, c byte) bool {
	for i := range len(s) {
		if s[i] == c {
			return true
		}
	}
	return false
}

// extractLastWord extracts the last whitespace-delimited word from a string.
// This is the partial class name being typed.
//
// Takes text (string) which is the text to extract from.
//
// Returns string which is the last word, or empty if the text ends with
// whitespace.
func extractLastWord(text string) string {
	end := len(text)
	if end == 0 {
		return ""
	}

	if text[end-1] == ' ' || text[end-1] == '\t' {
		return ""
	}

	start := end - 1
	for start > 0 && text[start-1] != ' ' && text[start-1] != '\t' {
		start--
	}
	return text[start:end]
}
