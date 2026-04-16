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

import (
	"bytes"
	"io"
	"sort"
	"unicode/utf8"
)

const (
	// angleBracketOpen holds the '<' byte used to detect tag openings.
	angleBracketOpen byte = '<'

	// angleBracketClose holds the '>' byte used to detect tag closings.
	angleBracketClose byte = '>'

	// forwardSlash holds the '/' byte used in end tags and self-closing tags.
	forwardSlash byte = '/'

	// equalsSign holds the '=' byte used to separate attribute names from values.
	equalsSign byte = '='

	// doubleQuote holds the '"' byte used to delimit attribute values.
	doubleQuote byte = '"'

	// singleQuote holds the single-quote byte used to delimit attribute values.
	singleQuote byte = '\''

	// exclamationMark holds the '!' byte used in comments, CDATA, and doctype openings.
	exclamationMark byte = '!'

	// questionMark holds the '?' byte used in processing instruction openings.
	questionMark byte = '?'

	// hyphenMinus holds the '-' byte used in comment delimiters.
	hyphenMinus byte = '-'

	// lineFeed holds the newline byte used for line tracking.
	lineFeed byte = '\n'

	// utf8ContinuationMask and utf8ContinuationValue identify UTF-8
	// continuation bytes (10xxxxxx), used by advanceOne to avoid
	// incrementing the column counter for non-leading bytes.
	utf8ContinuationMask byte = 0xC0

	// utf8ContinuationValue holds the bit pattern that identifies a UTF-8 continuation byte.
	utf8ContinuationValue byte = 0x80
)

const (
	// commentOpenLength holds the byte length of the "<!--" opening sequence.
	commentOpenLength = 4

	// commentCloseLength holds the byte length of the "-->" closing sequence.
	commentCloseLength = 3

	// bangCommentCloseLength holds the byte length of the "--!>" closing sequence.
	bangCommentCloseLength = 4

	// cdataPrefixLength holds the byte length of the "<![CDATA[" opening sequence.
	cdataPrefixLength = 9

	// cdataCloseLength holds the byte length of the "]]>" closing sequence.
	cdataCloseLength = 3

	// doctypeKeywordLength holds the byte length of the "DOCTYPE" keyword.
	doctypeKeywordLength = 7
)

// rawTextKind identifies the type of raw text element the lexer is currently
// inside. Raw text elements consume their content verbatim until the matching
// closing tag.
type rawTextKind uint8

const (
	// rawTextNone indicates the lexer is not inside any raw text element.
	rawTextNone rawTextKind = iota

	// rawTextScript indicates the lexer is inside a <script> element.
	rawTextScript

	// rawTextStyle indicates the lexer is inside a <style> element.
	rawTextStyle

	// rawTextTextarea indicates the lexer is inside a <textarea> element.
	rawTextTextarea

	// rawTextTitle indicates the lexer is inside a <title> element.
	rawTextTitle

	// rawTextXmp indicates the lexer is inside a <xmp> element.
	rawTextXmp

	// rawTextIframe indicates the lexer is inside an <iframe> element.
	rawTextIframe

	// rawTextPlaintext indicates the lexer is inside a <plaintext> element.
	rawTextPlaintext
)

// tagOpeningKind classifies what kind of markup a '<' character introduces.
type tagOpeningKind uint8

const (
	// tagOpeningNone indicates the '<' is not the start of any recognised tag.
	tagOpeningNone tagOpeningKind = iota

	// tagOpeningStartTag indicates a start tag opening such as <tagname.
	tagOpeningStartTag

	// tagOpeningEndTag indicates an end tag opening such as </tagname>.
	tagOpeningEndTag

	// tagOpeningComment indicates a comment opening via the <!-- sequence.
	tagOpeningComment

	// tagOpeningCDATA indicates a CDATA section opening via <![CDATA[.
	tagOpeningCDATA

	// tagOpeningDoctype indicates a DOCTYPE declaration opening via <!DOCTYPE.
	tagOpeningDoctype

	// tagOpeningMarkupDeclaration indicates a markup declaration opening via <!.
	tagOpeningMarkupDeclaration

	// tagOpeningProcessingInstr indicates a processing instruction opening via <?.
	tagOpeningProcessingInstr
)

// tagKindType classifies a tag name into raw text, foreign, or normal.
type tagKindType uint8

const (
	// foreignTagNone indicates the tag is a normal HTML element.
	foreignTagNone tagKindType = iota

	// foreignTagSVG indicates the tag is an <svg> foreign content element.
	foreignTagSVG

	// foreignTagMath indicates the tag is a <math> foreign content element.
	foreignTagMath

	// rawTagScript indicates the tag is a <script> raw text element.
	rawTagScript

	// rawTagStyle indicates the tag is a <style> raw text element.
	rawTagStyle

	// rawTagTextarea indicates the tag is a <textarea> raw text element.
	rawTagTextarea

	// rawTagTitle indicates the tag is a <title> raw text element.
	rawTagTitle

	// rawTagXmp indicates the tag is a <xmp> raw text element.
	rawTagXmp

	// rawTagIframe indicates the tag is an <iframe> raw text element.
	rawTagIframe

	// rawTagPlaintext indicates the tag is a <plaintext> raw text element.
	rawTagPlaintext
)

// rawTextTagNames maps each rawTextKind to its tag name bytes for closing tag
// matching.
var rawTextTagNames = [...][]byte{
	rawTextNone:      nil,
	rawTextScript:    []byte("script"),
	rawTextStyle:     []byte("style"),
	rawTextTextarea:  []byte("textarea"),
	rawTextTitle:     []byte("title"),
	rawTextXmp:       []byte("xmp"),
	rawTextIframe:    []byte("iframe"),
	rawTextPlaintext: []byte("plaintext"),
}

// Lexer tokenises HTML source into a stream of tokens. It is a streaming,
// single-pass tokeniser that returns one token per call to Next().
//
// The lexer never modifies the source buffer. All byte slices returned by
// Text() and AttrVal() are sub-slices of the original source. Callers must
// not mutate these slices.
type Lexer struct {
	// Pointer-containing fields grouped first to minimise GC scan region.
	// The error interface (2 pointer words) is placed between slices so that
	// the last pointer word falls at offset 88 rather than 104, reducing GC
	// scan from 112 to 96 bytes.
	source []byte

	// newlineOffsets holds the byte offsets of all newline characters in the source.
	newlineOffsets []int

	// err holds the current error state of the lexer.
	err error

	// text holds the tag name, attribute key, or comment content for the current token.
	text []byte

	// attributeValue holds the raw attribute value bytes for the current attribute token.
	attributeValue []byte

	// cursor holds the current byte offset into the source buffer.
	cursor int

	// currentLine holds the 1-based line number at the cursor position.
	currentLine int

	// currentColumn holds the 1-based column number at the cursor position.
	currentColumn int

	// lastNewlinePos holds the byte offset of the most recently seen newline character.
	lastNewlinePos int

	// tokenStartOffset holds the byte offset where the current token begins.
	tokenStartOffset int

	// tokenEndOffset holds the byte offset where the current token ends.
	tokenEndOffset int

	// tokenLine holds the 1-based line number where the current token begins.
	tokenLine int

	// tokenColumn holds the 1-based column number where the current token begins.
	tokenColumn int

	// attributeValueStart holds the byte offset where the attribute value begins, or -1.
	attributeValueStart int

	// Small fields packed at the end to minimise padding.
	insideTag bool

	// rawTextTag holds the kind of raw text element the lexer is currently inside.
	rawTextTag rawTextKind
}

// NewLexer creates a lexer for the given HTML source. The source byte slice
// is retained by reference and must not be modified during lexing.
//
// Takes source ([]byte) which is the HTML content to tokenise.
//
// Returns *Lexer which is ready to produce tokens via Next().
func NewLexer(source []byte) *Lexer {
	newlineCount := bytes.Count(source, []byte{lineFeed})
	offsets := make([]int, 0, newlineCount)

	for i, b := range source {
		if b == lineFeed {
			offsets = append(offsets, i)
		}
	}

	return &Lexer{
		source:              source,
		cursor:              0,
		newlineOffsets:      offsets,
		insideTag:           false,
		rawTextTag:          rawTextNone,
		currentLine:         1,
		currentColumn:       1,
		lastNewlinePos:      -1,
		tokenStartOffset:    0,
		tokenEndOffset:      0,
		tokenLine:           1,
		tokenColumn:         1,
		text:                nil,
		attributeValue:      nil,
		attributeValueStart: -1,
		err:                 nil,
	}
}

// Next advances the lexer to the next token and returns its type. Token data
// is available via the accessor methods (Text, AttrVal, etc.) until the next
// call to Next().
//
// Returns TokenType which indicates the kind of token produced.
func (l *Lexer) Next() TokenType {
	l.resetTokenState()

	if l.insideTag {
		return l.nextInsideTag()
	}

	if l.rawTextTag != rawTextNone {
		return l.nextRawText()
	}

	return l.nextContent()
}

// Text returns the tag name, attribute key, or comment content for the current token.
//
// For CommentToken, returns the inner content without delimiters. For
// SVGToken/MathToken, returns the entire foreign block. The returned slice
// is a sub-slice of the source and must not be mutated.
//
// Returns []byte which holds the text content associated with the current token.
func (l *Lexer) Text() []byte {
	return l.text
}

// AttrVal returns the attribute value bytes for the current AttributeToken.
//
// The value may include surrounding quotes.
//
// Returns []byte which holds the attribute value, or nil for boolean attributes.
func (l *Lexer) AttrVal() []byte {
	return l.attributeValue
}

// TokenStart returns the byte offset in the source where the current token
// begins.
//
// Returns int which holds the starting byte offset of the current token.
func (l *Lexer) TokenStart() int {
	return l.tokenStartOffset
}

// TokenEnd returns the byte offset in the source where the current token ends.
//
// Returns int which holds the offset of the first byte after the token.
func (l *Lexer) TokenEnd() int {
	return l.tokenEndOffset
}

// TokenLine returns the 1-based line number at the start of the current
// token.
//
// Returns int which holds the line number where the current token begins.
func (l *Lexer) TokenLine() int {
	return l.tokenLine
}

// TokenCol returns the 1-based column number at the start of the current token.
//
// Returns int which holds the column number measured in Unicode runes, not bytes.
func (l *Lexer) TokenCol() int {
	return l.tokenColumn
}

// AttrValStart returns the byte offset where the attribute value begins in
// the source.
//
// Returns int which holds the byte offset of the attribute value, or -1 when
// the current token has no attribute value.
func (l *Lexer) AttrValStart() int {
	return l.attributeValueStart
}

// PositionAt converts an arbitrary byte offset in the source to a 1-based
// line and column position, measured in Unicode runes using a precomputed
// newline index for O(log n) lookup.
//
// Takes offset (int) which is the byte offset to convert.
//
// Returns line (int) which is the 1-based line number.
// Returns column (int) which is the 1-based column in runes.
func (l *Lexer) PositionAt(offset int) (line int, column int) {
	if offset < 0 {
		offset = 0
	}

	if offset > len(l.source) {
		offset = len(l.source)
	}

	lineIndex := sort.SearchInts(l.newlineOffsets, offset)
	line = lineIndex + 1

	columnStartOffset := 0
	if lineIndex > 0 {
		columnStartOffset = l.newlineOffsets[lineIndex-1] + 1
	}

	column = utf8.RuneCount(l.source[columnStartOffset:offset]) + 1

	return line, column
}

// SourceBytes returns the full source buffer. This allows consumers to slice
// raw content by byte offsets obtained from TokenStart() and TokenEnd().
//
// Returns []byte which holds the complete HTML source passed to NewLexer.
func (l *Lexer) SourceBytes() []byte {
	return l.source
}

// Err returns the lexer's error state.
//
// Returns error which is io.EOF when all input is consumed, or nil when no
// error has occurred.
func (l *Lexer) Err() error {
	return l.err
}

// resetTokenState clears per-token fields before producing a new token.
func (l *Lexer) resetTokenState() {
	l.text = nil
	l.attributeValue = nil
	l.attributeValueStart = -1
}

// recordTokenPosition captures the current cursor position as the start of
// a new token, computing line and column from the incremental tracking state.
func (l *Lexer) recordTokenPosition() {
	l.tokenStartOffset = l.cursor
	l.tokenLine = l.currentLine
	l.tokenColumn = l.currentColumn
}

// finaliseToken records the cursor as the end of the current token.
func (l *Lexer) finaliseToken() {
	l.tokenEndOffset = l.cursor
}

// advanceCursor moves the cursor forward by n bytes, updating the incremental
// line and column tracking.
//
// Takes n (int) which specifies the number of bytes to advance.
//
// For single-byte advances, prefer advanceOne which avoids slice creation
// and bytes.Count overhead.
func (l *Lexer) advanceCursor(n int) {
	end := min(l.cursor+n, len(l.source))

	segment := l.source[l.cursor:end]

	newlineCount := bytes.Count(segment, []byte{lineFeed})
	if newlineCount > 0 {
		l.currentLine += newlineCount
		lastLF := bytes.LastIndexByte(segment, lineFeed)
		l.lastNewlinePos = l.cursor + lastLF
		l.currentColumn = utf8.RuneCount(l.source[l.lastNewlinePos+1:end]) + 1
	} else {
		l.currentColumn += utf8.RuneCount(segment)
	}

	l.cursor = end
}

// advanceOne moves the cursor forward by exactly one byte with minimal
// overhead. Unlike advanceCursor(1), it avoids slice creation, bytes.Count,
// and utf8.RuneCount calls.
//
// For multi-byte UTF-8 sequences, only the leading byte increments the
// column counter; continuation bytes (10xxxxxx) are skipped. This correctly
// counts columns in runes rather than bytes.
func (l *Lexer) advanceOne() {
	if l.cursor >= len(l.source) {
		return
	}

	b := l.source[l.cursor]
	l.cursor++

	if b == lineFeed {
		l.currentLine++
		l.lastNewlinePos = l.cursor - 1
		l.currentColumn = 1

		return
	}

	if b < utf8.RuneSelf {
		l.currentColumn++

		return
	}

	if b&utf8ContinuationMask != utf8ContinuationValue {
		l.currentColumn++
	}
}

// atEnd reports whether the cursor has reached the end of the source.
//
// Returns bool which indicates true when no more bytes remain.
func (l *Lexer) atEnd() bool {
	return l.cursor >= len(l.source)
}

// peek returns the byte at the given offset from the cursor, or 0 if the
// offset is out of bounds.
//
// Takes offset (int) which specifies the distance from the cursor in bytes.
//
// Returns byte which holds the byte at that position, or 0 if out of bounds.
func (l *Lexer) peek(offset int) byte {
	pos := l.cursor + offset
	if pos < 0 || pos >= len(l.source) {
		return 0
	}

	return l.source[pos]
}

// skipWhitespace advances the cursor past any HTML whitespace characters
// (space, tab, newline, carriage return, form feed).
func (l *Lexer) skipWhitespace() {
	for !l.atEnd() && isHTMLWhitespace(l.source[l.cursor]) {
		l.advanceOne()
	}
}

// nextInsideTag handles the state where the lexer is inside a start tag,
// reading attributes or the tag close.
//
// Returns TokenType which indicates the kind of token produced.
func (l *Lexer) nextInsideTag() TokenType {
	l.skipWhitespace()
	l.recordTokenPosition()

	if l.atEnd() {
		l.err = io.EOF
		l.finaliseToken()

		return ErrorToken
	}

	current := l.source[l.cursor]

	if current == forwardSlash && l.peek(1) == angleBracketClose {
		return l.emitTagVoidClose()
	}

	if current == angleBracketClose {
		return l.emitTagClose()
	}

	return l.parseAttribute()
}

// nextRawText handles the state where the lexer is inside a raw text element
// and must consume content until the matching closing tag.
//
// Returns TokenType which indicates the kind of token produced.
func (l *Lexer) nextRawText() TokenType {
	l.recordTokenPosition()
	contentStart := l.cursor

	if l.rawTextTag == rawTextPlaintext {
		l.advanceCursor(len(l.source) - l.cursor)
		l.text = l.source[contentStart:l.cursor]
		l.rawTextTag = rawTextNone
		l.finaliseToken()

		return TextToken
	}

	if l.rawTextTag == rawTextScript {
		return l.parseScriptContent(contentStart)
	}

	return l.parseGenericRawText(contentStart)
}

// nextContent handles the default state: scanning text and detecting tag
// openings.
//
// Returns TokenType which indicates the kind of token produced.
func (l *Lexer) nextContent() TokenType {
	l.recordTokenPosition()
	textStart := l.cursor

	for !l.atEnd() {
		if l.source[l.cursor] != angleBracketOpen {
			l.advanceOne()

			continue
		}

		kind := l.identifyTagOpening()
		if kind == tagOpeningNone {
			l.advanceOne()

			continue
		}

		if l.cursor > textStart {
			return l.emitText(textStart)
		}

		l.recordTokenPosition()

		return l.dispatchTagOpening(kind)
	}

	if l.cursor > textStart {
		return l.emitText(textStart)
	}

	l.recordTokenPosition()
	l.err = io.EOF
	l.finaliseToken()

	return ErrorToken
}

// emitTagClose emits a StartTagCloseToken for the > character.
//
// Returns TokenType which is always StartTagCloseToken.
func (l *Lexer) emitTagClose() TokenType {
	l.advanceOne()
	l.insideTag = false
	l.finaliseToken()

	return StartTagCloseToken
}

// emitTagVoidClose emits a StartTagVoidToken for the /> sequence.
//
// Returns TokenType which is always StartTagVoidToken.
func (l *Lexer) emitTagVoidClose() TokenType {
	l.advanceCursor(2)
	l.insideTag = false
	l.finaliseToken()

	return StartTagVoidToken
}

// emitText packages accumulated text content as a TextToken.
//
// Takes textStart (int) which specifies the byte offset where the text began.
//
// Returns TokenType which is always TextToken.
func (l *Lexer) emitText(textStart int) TokenType {
	l.text = l.source[textStart:l.cursor]
	l.finaliseToken()

	return TextToken
}

// emitRawText packages the raw text content and returns a TextToken.
//
// Takes contentStart (int) which specifies the byte offset where the raw text began.
//
// Returns TokenType which is always TextToken.
func (l *Lexer) emitRawText(contentStart int) TokenType {
	l.text = l.source[contentStart:l.cursor]
	l.rawTextTag = rawTextNone
	l.finaliseToken()

	return TextToken
}

// emitRawTextAtCursor is like emitRawText but used when rawTextTag has
// already been cleared by a sub-handler.
//
// Takes contentStart (int) which specifies the byte offset where the raw text began.
//
// Returns TokenType which is always TextToken.
func (l *Lexer) emitRawTextAtCursor(contentStart int) TokenType {
	l.text = l.source[contentStart:l.cursor]
	l.finaliseToken()

	return TextToken
}
