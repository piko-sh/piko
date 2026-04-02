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

package htmllexer

import "bytes"

// identifyTagOpening classifies the markup starting at '<' without consuming
// any bytes.
//
// Returns tagOpeningKind which indicates the type of markup, or tagOpeningNone
// if the '<' is not the start of any recognised markup.
func (l *Lexer) identifyTagOpening() tagOpeningKind {
	if l.isStartTagOpening() {
		return tagOpeningStartTag
	}

	if l.isEndTagOpening() {
		return tagOpeningEndTag
	}

	if l.isCommentOpening() {
		return tagOpeningComment
	}

	if l.isCDATAOpening() {
		return tagOpeningCDATA
	}

	if l.isDoctypeOpening() {
		return tagOpeningDoctype
	}

	if l.isMarkupDeclarationOpening() {
		return tagOpeningMarkupDeclaration
	}

	if l.isProcessingInstructionOpening() {
		return tagOpeningProcessingInstr
	}

	return tagOpeningNone
}

// dispatchTagOpening routes to the correct parser for the identified tag
// opening kind.
//
// Takes kind (tagOpeningKind) which specifies the type of tag opening to dispatch.
//
// Returns TokenType which indicates the kind of token produced by the parser.
func (l *Lexer) dispatchTagOpening(kind tagOpeningKind) TokenType {
	switch kind {
	case tagOpeningStartTag:
		return l.parseStartTag()
	case tagOpeningEndTag:
		return l.parseEndTag()
	case tagOpeningComment:
		return l.parseComment()
	case tagOpeningCDATA:
		return l.parseCDATA()
	case tagOpeningDoctype:
		return l.parseDoctype()
	case tagOpeningMarkupDeclaration, tagOpeningProcessingInstr:
		return l.parseBogusComment()
	default:
		l.advanceOne()

		return l.nextContent()
	}
}

// isStartTagOpening checks for < followed by an ASCII letter.
//
// Returns bool which indicates true when the cursor is at a start tag opening.
func (l *Lexer) isStartTagOpening() bool {
	return l.peek(1) != forwardSlash && isASCIILetter(l.peek(1))
}

// isEndTagOpening checks for </ followed by an ASCII letter.
//
// Returns bool which indicates true when the cursor is at an end tag opening.
func (l *Lexer) isEndTagOpening() bool {
	return l.peek(1) == forwardSlash && isASCIILetter(l.peek(2))
}

// isCommentOpening checks for the <!-- sequence.
//
// Returns bool which indicates true when the cursor is at a comment opening.
func (l *Lexer) isCommentOpening() bool {
	return l.peek(1) == exclamationMark &&
		l.peek(2) == hyphenMinus &&
		l.peek(commentOpenLength-1) == hyphenMinus
}

// isCDATAOpening checks for the <![CDATA[ sequence.
//
// Returns bool which indicates true when the cursor is at a CDATA opening.
func (l *Lexer) isCDATAOpening() bool {
	if l.peek(1) != exclamationMark || l.peek(2) != '[' {
		return false
	}

	remaining := l.source[l.cursor:]
	cdataPrefix := []byte("<![CDATA[")

	return len(remaining) >= len(cdataPrefix) &&
		bytes.Equal(remaining[:len(cdataPrefix)], cdataPrefix)
}

// isDoctypeOpening checks for <!DOCTYPE (case-insensitive).
//
// Returns bool which indicates true when the cursor is at a DOCTYPE opening.
func (l *Lexer) isDoctypeOpening() bool {
	if l.peek(1) != exclamationMark {
		return false
	}

	remaining := l.source[l.cursor+2:]
	if len(remaining) < doctypeKeywordLength {
		return false
	}

	return bytes.EqualFold(remaining[:doctypeKeywordLength], []byte("DOCTYPE"))
}

// isMarkupDeclarationOpening checks for <! that is not a comment, CDATA,
// or doctype.
//
// Returns bool which indicates true when the cursor is at a markup declaration opening.
func (l *Lexer) isMarkupDeclarationOpening() bool {
	return l.peek(1) == exclamationMark
}

// isProcessingInstructionOpening checks for <?.
//
// Returns bool which indicates true when the cursor is at a processing instruction opening.
func (l *Lexer) isProcessingInstructionOpening() bool {
	return l.peek(1) == questionMark
}

// parseStartTag reads a start tag name, classifies it, and returns the
// appropriate token type.
//
// Returns TokenType which indicates the kind of token produced.
func (l *Lexer) parseStartTag() TokenType {
	l.advanceOne()
	nameStart := l.cursor

	l.scanTagName()

	l.text = l.source[nameStart:l.cursor]

	tagKind := classifyTagName(l.text)

	if tagKind == foreignTagSVG {
		return l.parseForeignContent(SVGToken)
	}

	if tagKind == foreignTagMath {
		return l.parseForeignContent(MathToken)
	}

	if tagKind != foreignTagNone {
		l.rawTextTag = tagKindToRawText(tagKind)
	}

	l.insideTag = true
	l.finaliseToken()

	return StartTagToken
}

// parseEndTag reads a closing tag name and consumes through the closing >.
//
// Returns TokenType which is always EndTagToken.
func (l *Lexer) parseEndTag() TokenType {
	l.advanceCursor(2)
	nameStart := l.cursor

	for !l.atEnd() && l.source[l.cursor] != angleBracketClose {
		l.advanceOne()
	}

	nameEnd := l.cursor

	for nameEnd > nameStart && isHTMLWhitespace(l.source[nameEnd-1]) {
		nameEnd--
	}

	l.text = l.source[nameStart:nameEnd]

	if !l.atEnd() && l.source[l.cursor] == angleBracketClose {
		l.advanceOne()
	}

	l.finaliseToken()

	return EndTagToken
}

// parseComment consumes an HTML comment from <!-- through --> or --!>.
//
// Returns TokenType which is always CommentToken.
func (l *Lexer) parseComment() TokenType {
	l.advanceCursor(commentOpenLength)
	contentStart := l.cursor

	for !l.atEnd() {
		if l.matchesCommentClose() {
			l.text = l.source[contentStart:l.cursor]
			l.advanceCursor(commentCloseLength)
			l.finaliseToken()

			return CommentToken
		}

		if l.matchesBangCommentClose() {
			l.text = l.source[contentStart:l.cursor]
			l.advanceCursor(bangCommentCloseLength)
			l.finaliseToken()

			return CommentToken
		}

		l.advanceOne()
	}

	l.text = l.source[contentStart:l.cursor]
	l.finaliseToken()

	return CommentToken
}

// matchesBangCommentClose checks whether the cursor is at a --!> sequence.
//
// Returns bool which indicates true when the cursor is at the --!> sequence.
func (l *Lexer) matchesBangCommentClose() bool {
	return l.peek(0) == hyphenMinus &&
		l.peek(1) == hyphenMinus &&
		l.peek(2) == exclamationMark &&
		l.peek(bangCommentCloseLength-1) == angleBracketClose
}

// parseCDATA consumes a <![CDATA[...]]> section and returns it as TextToken.
//
// Returns TokenType which is always TextToken.
func (l *Lexer) parseCDATA() TokenType {
	l.advanceCursor(cdataPrefixLength)
	contentStart := l.cursor

	for !l.atEnd() {
		if l.peek(0) == ']' && l.peek(1) == ']' && l.peek(2) == angleBracketClose {
			l.text = l.source[contentStart:l.cursor]
			l.advanceCursor(cdataCloseLength)
			l.finaliseToken()

			return TextToken
		}

		l.advanceOne()
	}

	l.text = l.source[contentStart:l.cursor]
	l.finaliseToken()

	return TextToken
}

// parseDoctype consumes a <!DOCTYPE ...> declaration silently and continues
// scanning for the next real token.
//
// Returns TokenType which indicates the kind of token produced after the DOCTYPE.
func (l *Lexer) parseDoctype() TokenType {
	for !l.atEnd() && l.source[l.cursor] != angleBracketClose {
		l.advanceOne()
	}

	if !l.atEnd() {
		l.advanceOne()
	}

	return l.nextContent()
}

// parseBogusComment consumes a bogus comment (<?...> or <!...>) and returns
// it as CommentToken.
//
// Returns TokenType which is always CommentToken.
func (l *Lexer) parseBogusComment() TokenType {
	l.advanceCursor(2)
	contentStart := l.cursor

	for !l.atEnd() && l.source[l.cursor] != angleBracketClose {
		l.advanceOne()
	}

	l.text = l.source[contentStart:l.cursor]

	if !l.atEnd() {
		l.advanceOne()
	}

	l.finaliseToken()

	return CommentToken
}

// parseGenericRawText consumes raw text content for non-script elements
// until the matching closing tag.
//
// Takes contentStart (int) which specifies the byte offset where the raw text content began.
//
// Returns TokenType which indicates the kind of token produced.
func (l *Lexer) parseGenericRawText(contentStart int) TokenType {
	closingTag := rawTextTagNames[l.rawTextTag]

	for !l.atEnd() {
		if !l.isClosingTagForRawText(closingTag) {
			l.advanceOne()

			continue
		}

		if contentStart == l.cursor {
			l.rawTextTag = rawTextNone

			return l.nextContent()
		}

		return l.emitRawText(contentStart)
	}

	if contentStart == l.cursor {
		l.rawTextTag = rawTextNone
		l.finaliseToken()

		return ErrorToken
	}

	return l.emitRawText(contentStart)
}

// parseScriptContent consumes content inside a <script> element, handling
// the HTML5 spec's special rules for <!-- comments within scripts.
//
// Inside a script, <!-- starts a comment region where nested <script> tags
// are tracked. The closing </script> tag only ends the script element when
// it appears outside a nested script opened by <!-- comment rules. The
// comment region ends when --> is encountered.
//
// Takes contentStart (int) which specifies the byte offset where the script content began.
//
// Returns TokenType which indicates the kind of token produced.
func (l *Lexer) parseScriptContent(contentStart int) TokenType {
	for !l.atEnd() {
		if l.isClosingTagForRawText(rawTextTagNames[rawTextScript]) {
			if contentStart == l.cursor {
				l.rawTextTag = rawTextNone

				return l.nextContent()
			}

			return l.emitRawText(contentStart)
		}

		if l.matchesCommentOpen() {
			l.advanceCursor(commentOpenLength)
			l.parseScriptCommentRegion()

			if token, done := l.resolvePostScriptComment(contentStart); done {
				return token
			}

			continue
		}

		l.advanceOne()
	}

	if contentStart == l.cursor {
		l.rawTextTag = rawTextNone
		l.finaliseToken()

		return ErrorToken
	}

	return l.emitRawText(contentStart)
}

// resolvePostScriptComment checks whether the script comment region
// ended the script element entirely, and if so returns the
// appropriate token.
//
// Takes contentStart (int) which is the byte offset where the script
// content began.
//
// Returns TokenType which is the token to emit when done is true.
// Returns bool which reports whether the caller should return the
// token.
func (l *Lexer) resolvePostScriptComment(contentStart int) (TokenType, bool) {
	if l.rawTextTag != rawTextNone {
		return 0, false
	}
	if contentStart == l.cursor {
		return l.nextContent(), true
	}
	return l.emitRawTextAtCursor(contentStart), true
}

// parseScriptCommentRegion processes the interior of a <!-- comment inside a
// <script> element.
//
// When the outer </script> is found inside the comment without a nested
// script being open, sets rawTextTag to rawTextNone.
func (l *Lexer) parseScriptCommentRegion() {
	nestedScriptOpen := false

	for !l.atEnd() {
		if l.matchesCommentClose() {
			l.advanceCursor(commentCloseLength)

			return
		}

		if l.isNestedScriptOpen() {
			nestedScriptOpen = true
			l.advanceOne()

			continue
		}

		if l.isClosingTagForRawText(rawTextTagNames[rawTextScript]) {
			if nestedScriptOpen {
				nestedScriptOpen = false
				l.advanceOne()

				continue
			}

			l.rawTextTag = rawTextNone

			return
		}

		l.advanceOne()
	}
}

// isNestedScriptOpen checks whether the cursor is at a <script start tag
// opening inside a <!-- comment region.
//
// Returns bool which indicates true when the cursor is at a nested <script opening.
func (l *Lexer) isNestedScriptOpen() bool {
	if l.peek(0) != angleBracketOpen {
		return false
	}

	if l.peek(1) == forwardSlash {
		return false
	}

	remaining := l.source[l.cursor+1:]
	scriptTag := rawTextTagNames[rawTextScript]

	if len(remaining) < len(scriptTag) {
		return false
	}

	return bytes.EqualFold(remaining[:len(scriptTag)], scriptTag)
}

// isClosingTagForRawText checks whether the cursor is at a closing tag that
// matches the given tag name (case-insensitive).
//
// Takes tagName ([]byte) which specifies the expected tag name to match against.
//
// Returns bool which indicates true when a matching closing tag is found at the cursor.
func (l *Lexer) isClosingTagForRawText(tagName []byte) bool {
	if l.peek(0) != angleBracketOpen {
		return false
	}

	if l.peek(1) != forwardSlash {
		return false
	}

	remaining := l.source[l.cursor+2:]
	if len(remaining) < len(tagName) {
		return false
	}

	return bytes.EqualFold(remaining[:len(tagName)], tagName)
}

// matchesCommentOpen checks whether the cursor is at a <!-- sequence.
//
// Returns bool which indicates true when the cursor is at the <!-- sequence.
func (l *Lexer) matchesCommentOpen() bool {
	return l.peek(0) == angleBracketOpen &&
		l.peek(1) == exclamationMark &&
		l.peek(2) == hyphenMinus &&
		l.peek(commentOpenLength-1) == hyphenMinus
}

// matchesCommentClose checks whether the cursor is at a --> sequence.
//
// Returns bool which indicates true when the cursor is at the --> sequence.
func (l *Lexer) matchesCommentClose() bool {
	return l.peek(0) == hyphenMinus &&
		l.peek(1) == hyphenMinus &&
		l.peek(2) == angleBracketClose
}

// parseAttribute reads an attribute name and optional value from within a
// start tag.
//
// Returns TokenType which is always AttributeToken.
func (l *Lexer) parseAttribute() TokenType {
	nameStart := l.cursor

	l.scanAttributeName()

	nameEnd := l.cursor
	l.text = l.source[nameStart:nameEnd]

	l.skipWhitespace()

	if l.atEnd() || l.source[l.cursor] != equalsSign {
		l.finaliseToken()

		return AttributeToken
	}

	l.parseAttributeValue()
	l.finaliseToken()

	return AttributeToken
}

// parseAttributeValue reads the value portion of an attribute after the '='
// sign. Handles quoted (double or single), and unquoted values.
func (l *Lexer) parseAttributeValue() {
	l.advanceOne()
	l.skipWhitespace()

	if l.atEnd() {
		return
	}

	l.attributeValueStart = l.cursor
	current := l.source[l.cursor]

	if current == doubleQuote || current == singleQuote {
		l.parseQuotedAttributeValue(current)

		return
	}

	l.parseUnquotedAttributeValue()
}

// parseQuotedAttributeValue reads a quoted attribute value, consuming the
// opening and closing delimiter.
//
// Takes delimiter (byte) which specifies the quote character that delimits the value.
func (l *Lexer) parseQuotedAttributeValue(delimiter byte) {
	valueStart := l.cursor
	l.advanceOne()

	for !l.atEnd() && l.source[l.cursor] != delimiter {
		l.advanceOne()
	}

	if !l.atEnd() {
		l.advanceOne()
	}

	l.attributeValue = l.source[valueStart:l.cursor]
}

// parseUnquotedAttributeValue reads an unquoted attribute value until
// whitespace or >.
func (l *Lexer) parseUnquotedAttributeValue() {
	valueStart := l.cursor

	for !l.atEnd() {
		current := l.source[l.cursor]
		if isHTMLWhitespace(current) || current == angleBracketClose {
			break
		}

		l.advanceOne()
	}

	l.attributeValue = l.source[valueStart:l.cursor]
}

// parseForeignContent consumes an entire SVG or Math element including all
// nested content and the closing tag.
//
// Takes tokenType (TokenType) which specifies whether to emit SVGToken or MathToken.
//
// Returns TokenType which holds the token type passed as tokenType.
func (l *Lexer) parseForeignContent(tokenType TokenType) TokenType {
	blockStart := l.tokenStartOffset
	closingTagName := l.text
	var quoteChar byte

	for !l.atEnd() {
		current := l.source[l.cursor]

		if (current == doubleQuote || current == singleQuote) && quoteChar == 0 {
			quoteChar = current
			l.advanceOne()

			continue
		}

		if current == quoteChar {
			quoteChar = 0
			l.advanceOne()

			continue
		}

		if quoteChar != 0 {
			l.advanceOne()

			continue
		}

		if l.isForeignClosingTag(closingTagName) {
			l.consumeForeignClosingTag()

			break
		}

		l.advanceOne()
	}

	l.text = l.source[blockStart:l.cursor]
	l.insideTag = false
	l.finaliseToken()

	return tokenType
}

// isForeignClosingTag checks whether the cursor is at a closing tag matching
// the given name (case-insensitive).
//
// Takes tagName ([]byte) which specifies the tag name to match against.
//
// Returns bool which indicates true when a matching closing tag is found at the cursor.
func (l *Lexer) isForeignClosingTag(tagName []byte) bool {
	if l.peek(0) != angleBracketOpen || l.peek(1) != forwardSlash {
		return false
	}

	remaining := l.source[l.cursor+2:]
	if len(remaining) < len(tagName) {
		return false
	}

	return bytes.EqualFold(remaining[:len(tagName)], tagName)
}

// consumeForeignClosingTag advances the cursor past a </tag> closing tag.
func (l *Lexer) consumeForeignClosingTag() {
	for !l.atEnd() && l.source[l.cursor] != angleBracketClose {
		l.advanceOne()
	}

	if !l.atEnd() {
		l.advanceOne()
	}
}

// scanTagName advances the cursor past a tag name (ASCII letters, digits,
// hyphens, and colons).
func (l *Lexer) scanTagName() {
	for !l.atEnd() {
		current := l.source[l.cursor]
		if isHTMLWhitespace(current) ||
			current == angleBracketClose ||
			current == forwardSlash {
			break
		}

		l.advanceOne()
	}
}

// scanAttributeName advances the cursor past an attribute name.
//
// When a forward slash is encountered, scanning stops only if it is immediately
// followed by '>' (the self-closing '/>' sequence). A standalone '/' is consumed
// as part of the attribute name to avoid producing zero-length names on stray
// slashes, which would cause an infinite loop.
func (l *Lexer) scanAttributeName() {
	for !l.atEnd() {
		current := l.source[l.cursor]
		if isHTMLWhitespace(current) ||
			current == equalsSign ||
			current == angleBracketClose {
			break
		}

		if current == forwardSlash && l.peek(1) == angleBracketClose {
			break
		}

		l.advanceOne()
	}
}

// specialTagEntry pairs a tag name with its classification.
type specialTagEntry struct {
	// name holds the lowercase tag name bytes used for case-insensitive matching.
	name []byte

	// kind holds the classification of this tag as foreign content or raw text.
	kind tagKindType
}

// specialTagsByLength indexes tag candidates by their byte length, so
// classifyTagName can jump directly to the right bucket without scanning
// irrelevant entries. The array is sized to cover the longest special tag
// name (plaintext = 9 bytes), so index 0-9 are valid.
var specialTagsByLength = [10][]specialTagEntry{
	3: {{name: []byte("svg"), kind: foreignTagSVG}, {name: []byte("xmp"), kind: rawTagXmp}},
	4: {{name: []byte("math"), kind: foreignTagMath}},
	5: {{name: []byte("style"), kind: rawTagStyle}, {name: []byte("title"), kind: rawTagTitle}},
	6: {{name: []byte("script"), kind: rawTagScript}, {name: []byte("iframe"), kind: rawTagIframe}},
	8: {{name: []byte("textarea"), kind: rawTagTextarea}},
	9: {{name: []byte("plaintext"), kind: rawTagPlaintext}},
}

// classifyTagName checks whether a tag name is a special element (foreign
// content or raw text) using a length-indexed lookup to minimise comparisons.
//
// Takes name ([]byte) which specifies the tag name bytes to classify.
//
// Returns tagKindType which indicates the classification of the tag.
func classifyTagName(name []byte) tagKindType {
	length := len(name)
	if length >= len(specialTagsByLength) {
		return foreignTagNone
	}

	for i := range specialTagsByLength[length] {
		if bytes.EqualFold(name, specialTagsByLength[length][i].name) {
			return specialTagsByLength[length][i].kind
		}
	}

	return foreignTagNone
}

// tagKindToRawText converts a tagKindType to the corresponding rawTextKind.
//
// Takes kind (tagKindType) which specifies the tag classification to convert.
//
// Returns rawTextKind which holds the corresponding raw text element kind.
func tagKindToRawText(kind tagKindType) rawTextKind {
	switch kind {
	case rawTagScript:
		return rawTextScript
	case rawTagStyle:
		return rawTextStyle
	case rawTagTextarea:
		return rawTextTextarea
	case rawTagTitle:
		return rawTextTitle
	case rawTagXmp:
		return rawTextXmp
	case rawTagIframe:
		return rawTextIframe
	case rawTagPlaintext:
		return rawTextPlaintext
	default:
		return rawTextNone
	}
}

// isASCIILetter reports whether b is an ASCII letter (a-z or A-Z).
//
// Takes b (byte) which specifies the byte to test.
//
// Returns bool which indicates true when b is an ASCII letter.
func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// htmlWhitespace is a compile-time lookup table for HTML whitespace bytes.
var htmlWhitespace = [256]bool{
	' ':  true,
	'\t': true,
	'\n': true,
	'\r': true,
	'\f': true,
}

// isHTMLWhitespace reports whether b is an HTML whitespace character (space,
// tab, newline, carriage return, or form feed).
//
// Takes b (byte) which specifies the byte to test.
//
// Returns bool which indicates true when b is an HTML whitespace character.
func isHTMLWhitespace(b byte) bool {
	return htmlWhitespace[b]
}
