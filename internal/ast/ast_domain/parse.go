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

package ast_domain

// Parses HTML templates into TemplateAST structures by tokenizing input and
// building a tree of nodes with directives and expressions. Handles element
// parsing, attribute processing, directive recognition, text interpolation, and
// string interning for performance optimisation.

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"piko.sh/piko/internal/htmllexer"
)

const (
	// textPartPreallocCapacity is the initial slice capacity for TextPart slices.
	// Sized for the common pattern: literal-expr-literal-expr.
	textPartPreallocCapacity = 4
)

var (
	// flagDirectives defines directives that are treated as boolean flags.
	// If the attribute is present, its value is considered "true" unless
	// explicitly set to "false".
	flagDirectives = map[DirectiveType]bool{
		DirectiveScaffold: true,
	}

	// internedStrings contains pre-allocated strings for common tag and attribute
	// names. Using interning reduces allocations when parsing templates with
	// repeated elements.
	internedStrings = map[string]string{
		"div": "div", "span": "span", "p": "p", "a": "a", "img": "img",
		"ul": "ul", "ol": "ol", "li": "li", "table": "table", "tr": "tr",
		"td": "td", "th": "th", "thead": "thead", "tbody": "tbody", "tfoot": "tfoot",
		"form": "form", "input": "input", "button": "button", "label": "label",
		"select": "select", "option": "option", "textarea": "textarea",
		"h1": "h1", "h2": "h2", "h3": "h3", "h4": "h4", "h5": "h5", "h6": "h6",
		"header": "header", "footer": "footer", "nav": "nav", "main": "main",
		"section": "section", "article": "article", "aside": "aside",
		"figure": "figure", "figcaption": "figcaption",
		"script": "script", "style": "style", "link": "link", "meta": "meta",
		"title": "title", "head": "head", "body": "body", "html": "html",
		"br": "br", "hr": "hr", "strong": "strong", "em": "em", "b": "b", "i": "i",
		"code": "code", "pre": "pre", "blockquote": "blockquote",
		"iframe": "iframe", "video": "video", "audio": "audio", "source": "source",
		"canvas": "canvas", "svg": "svg", "path": "path", "g": "g",
		"fragment": "fragment", "template": "template", "slot": "slot", "component": "component",
		"class": "class", "id": "id", "href": "href", "src": "src",
		"alt": "alt", "type": "type", "name": "name", "value": "value",
		"placeholder": "placeholder", "disabled": "disabled", "readonly": "readonly",
		"checked": "checked", "selected": "selected", "hidden": "hidden",
		"width": "width", "height": "height", "rel": "rel", "target": "target",
		"action": "action", "method": "method", "enctype": "enctype",
		"for": "for", "autocomplete": "autocomplete", "autofocus": "autofocus",
		"required": "required", "pattern": "pattern", "min": "min", "max": "max",
		"step": "step", "rows": "rows", "cols": "cols", "wrap": "wrap",
		"role": "role", "tabindex": "tabindex", "contenteditable": "contenteditable",
		"draggable": "draggable", "spellcheck": "spellcheck",
		"data-id": "data-id", "data-value": "data-value", "data-type": "data-type",
		"aria-label": "aria-label", "aria-hidden": "aria-hidden", "aria-expanded": "aria-expanded",
	}

	// svgCaseSensitiveAttrs is the set of SVG attribute names that
	// must preserve their original case, sourced from the HTML spec's
	// "adjust SVG attributes" step.
	//
	// All other attributes on SVG elements are lowercased for
	// performance so the renderer can use exact string comparison
	// instead of equalFold.
	svgCaseSensitiveAttrs = map[string]struct{}{
		"attributeName":       {},
		"attributeType":       {},
		"baseFrequency":       {},
		"baseProfile":         {},
		"calcMode":            {},
		"clipPathUnits":       {},
		"diffuseConstant":     {},
		"edgeMode":            {},
		"filterUnits":         {},
		"glyphRef":            {},
		"gradientTransform":   {},
		"gradientUnits":       {},
		"kernelMatrix":        {},
		"kernelUnitLength":    {},
		"keyPoints":           {},
		"keySplines":          {},
		"keyTimes":            {},
		"lengthAdjust":        {},
		"limitingConeAngle":   {},
		"markerHeight":        {},
		"markerUnits":         {},
		"markerWidth":         {},
		"maskContentUnits":    {},
		"maskUnits":           {},
		"numOctaves":          {},
		"pathLength":          {},
		"patternContentUnits": {},
		"patternTransform":    {},
		"patternUnits":        {},
		"pointsAtX":           {},
		"pointsAtY":           {},
		"pointsAtZ":           {},
		"preserveAlpha":       {},
		"preserveAspectRatio": {},
		"primitiveUnits":      {},
		"refX":                {},
		"refY":                {},
		"repeatCount":         {},
		"repeatDur":           {},
		"specularConstant":    {},
		"specularExponent":    {},
		"spreadMethod":        {},
		"startOffset":         {},
		"stdDeviation":        {},
		"stitchTiles":         {},
		"surfaceScale":        {},
		"tableValues":         {},
		"targetX":             {},
		"targetY":             {},
		"textLength":          {},
		"viewBox":             {},
		"viewTarget":          {},
		"xChannelSelector":    {},
		"yChannelSelector":    {},
		"zoomAndPan":          {},
	}

	// interpolationScannerPool provides pooled scanner instances to reduce
	// allocations.
	interpolationScannerPool = sync.Pool{
		New: func() any {
			return &interpolationScanner{}
		},
	}
)

// ParseOptions configures the behaviour of the HTML parser.
type ParseOptions struct {
	// RawMode treats all attributes as plain HTML attributes and skips directive
	// parsing. Useful when formatting rendered HTML output that contains p-key
	// attributes with values that are not valid expressions.
	RawMode bool
}

// Parser transforms raw HTML input into a preliminary TemplateAST. It uses a
// high-performance HTML lexer to tokenise the input and build a tree structure,
// keeping directive expressions as raw strings at this stage.
type Parser struct {
	// lexer breaks the HTML input into tokens for parsing.
	lexer *htmllexer.Lexer

	// tree holds the template AST being built during parsing.
	tree *TemplateAST

	// opts holds the parse options; nil means default behaviour.
	opts *ParseOptions

	// sourcePath is the file path of the source being parsed.
	sourcePath string

	// nodeStack holds the stack of parent nodes during parsing.
	nodeStack []*TemplateNode

	// startLine is the line number where parsing begins.
	startLine int

	// startCol is the starting column in the source file for offset calculations.
	startCol int

	// lowerBuf is a scratch buffer for lowercasing tag names and attribute
	// names without allocating. 64 bytes covers all standard HTML names.
	lowerBuf [64]byte
}

// toLower lowercases src into the parser's scratch buffer and returns the
// result as a byte slice. If src is longer than the scratch buffer, it falls
// back to bytes.ToLower which allocates.
//
// Takes src ([]byte) which is the byte slice to lowercase.
//
// Returns []byte which is the lowercased result.
func (p *Parser) toLower(src []byte) []byte {
	if len(src) > len(p.lowerBuf) {
		return bytes.ToLower(src)
	}
	dst := p.lowerBuf[:len(src)]
	for i, b := range src {
		if b >= 'A' && b <= 'Z' {
			dst[i] = b + ('a' - 'A')
		} else {
			dst[i] = b
		}
	}
	return dst
}

// adjustLocation converts a position within a template to a position
// in the original source file.
//
// Takes line (int) which is the line number within the template.
// Takes column (int) which is the column number within the template.
//
// Returns Location which holds the line, column, and offset in the original
// source file.
func (p *Parser) adjustLocation(line, column int) Location {
	finalLine := p.startLine + line - 1

	finalColumn := column
	if line == 1 {
		finalColumn = p.startCol + column - 1
	}

	return Location{Line: finalLine, Column: finalColumn, Offset: 0}
}

// run executes the main parsing loop, reading tokens from the lexer.
//
// Returns error when the lexer returns an error other than end of file.
func (p *Parser) run() error {
	head := &TemplateNode{}
	p.nodeStack = []*TemplateNode{head}

	for {
		tt := p.lexer.Next()

		if tt == htmllexer.ErrorToken {
			err := p.lexer.Err()
			if errors.Is(err, io.EOF) {
				p.tree.RootNodes = head.Children
				return nil
			}
			return fmt.Errorf("lexer error: %w", err)
		}

		switch tt {
		case htmllexer.TextToken:
			p.processText(p.lexer.Text())
		case htmllexer.StartTagToken:
			p.processElement()
		case htmllexer.EndTagToken:
			p.processEndTag()
		case htmllexer.CommentToken:
			p.processComment()
		case htmllexer.SVGToken, htmllexer.MathToken:
			p.processForeignToken()
		default:
		}
	}
}

// currentParent returns the node at the top of the parsing stack.
//
// Returns *TemplateNode which is the current parent node for nested elements.
func (p *Parser) currentParent() *TemplateNode {
	return p.nodeStack[len(p.nodeStack)-1]
}

// processText sends text content to the correct handler based on its type.
//
// Takes data ([]byte) which contains the text to process.
func (p *Parser) processText(data []byte) {
	if isRawTextParent(p.currentParent()) {
		p.handleRawText(data)
	} else if bytes.Contains(data, []byte("{{")) {
		p.handleInterpolatedText(data)
	} else {
		p.handlePlainText(data)
	}
}

// handleRawText processes text within elements like textarea or script,
// where the content should be kept as-is without parsing.
//
// Takes data ([]byte) which contains the raw text to process.
func (p *Parser) handleRawText(data []byte) {
	node := &TemplateNode{
		Key:                nil,
		DirKey:             nil,
		DirHTML:            nil,
		GoAnnotations:      nil,
		RuntimeAnnotations: nil,
		AttributeWriters:   nil,
		TextContentWriter:  nil,
		CustomEvents:       nil,
		OnEvents:           nil,
		Binds:              nil,
		DirContext:         nil,
		DirElse:            nil,
		DirText:            nil,
		DirStyle:           nil,
		DirClass:           nil,
		DirIf:              nil,
		DirElseIf:          nil,
		DirFor:             nil,
		DirShow:            nil,
		DirRef:             nil,
		DirModel:           nil,
		DirScaffold:        nil,
		TagName:            "",
		TextContent:        html.UnescapeString(string(data)),
		InnerHTML:          "",
		Children:           nil,
		RichText:           nil,
		Attributes:         nil,
		Diagnostics:        nil,
		DynamicAttributes:  nil,
		Directives:         nil,
		Location:           Location{},
		NodeType:           NodeText,
		IsPooled:           false,
		IsContentEditable:  false,
		PreserveWhitespace: true,
		NodeRange:          Range{},
		OpeningTagRange:    Range{},
		ClosingTagRange:    Range{},
		PreferredFormat:    FormatAuto,
	}
	p.currentParent().Children = append(p.currentParent().Children, node)
}

// interpolationScanner holds state while parsing `{{...}}` expressions.
type interpolationScanner struct {
	// p is the parent parser used to track input positions.
	p *Parser

	// data holds the raw input bytes being scanned.
	data []byte

	// cursor is the current read position in the input byte slice.
	cursor int

	// tokenStartOffset is the byte position where the current token starts.
	tokenStartOffset int
}

// handleInterpolatedText parses text that contains `{{...}}` expressions and
// builds a RichText node.
//
// Takes data ([]byte) which contains the text with interpolation expressions.
func (p *Parser) handleInterpolatedText(data []byte) {
	scanner := p.acquireScanner(data)
	parts, hasContent := scanner.scanAllParts()
	releaseScanner(scanner)

	if hasContent {
		baseLine, baseCol := p.lexer.TokenLine(), p.lexer.TokenCol()
		node := newRichTextNode(parts, p.adjustLocation(baseLine, baseCol))
		p.currentParent().Children = append(p.currentParent().Children, node)
	}
}

// acquireScanner gets a scanner from the pool and sets it up for use.
//
// Takes data ([]byte) which contains the input to scan.
//
// Returns *interpolationScanner which is ready to scan the input.
func (p *Parser) acquireScanner(data []byte) *interpolationScanner {
	scanner, ok := interpolationScannerPool.Get().(*interpolationScanner)
	if !ok {
		scanner = &interpolationScanner{}
	}
	scanner.p = p
	scanner.data = data
	scanner.cursor = 0
	scanner.tokenStartOffset = p.lexer.TokenStart()
	return scanner
}

// scanAllParts reads through the text data and splits it into parts.
//
// Returns []TextPart which contains the parsed literal and expression parts.
// Returns bool which is true when any parts were found.
func (s *interpolationScanner) scanAllParts() ([]TextPart, bool) {
	parts := make([]TextPart, 0, textPartPreallocCapacity)

	for s.cursor < len(s.data) {
		remainingBytes := s.data[s.cursor:]
		openDelimOffset := bytes.Index(remainingBytes, []byte("{{"))

		if openDelimOffset == -1 {
			if len(remainingBytes) > 0 {
				absLiteralStartOffset := s.tokenStartOffset + s.cursor
				line, column := s.p.lexer.PositionAt(absLiteralStartOffset)
				finalLocation := s.p.adjustLocation(line, column)

				parts = append(parts, TextPart{
					Expression:    nil,
					GoAnnotations: nil,
					Literal:       html.UnescapeString(string(remainingBytes)),
					RawExpression: "",
					Location:      finalLocation,
					IsLiteral:     true,
				})
			}
			break
		}

		if openDelimOffset > 0 {
			literalPartBytes := remainingBytes[:openDelimOffset]

			absLiteralStartOffset := s.tokenStartOffset + s.cursor
			line, column := s.p.lexer.PositionAt(absLiteralStartOffset)
			finalLocation := s.p.adjustLocation(line, column)

			parts = append(parts, TextPart{
				Expression:    nil,
				GoAnnotations: nil,
				Literal:       html.UnescapeString(string(literalPartBytes)),
				RawExpression: "",
				Location:      finalLocation,
				IsLiteral:     true,
			})
		}

		part, consumed, ok := s.scanExpressionPart(openDelimOffset)
		if ok {
			parts = append(parts, part)
		}
		s.cursor += consumed
	}
	return parts, len(parts) > 0
}

// scanExpressionPart finds and processes a single `{{...}}` block.
//
// Takes openDelimOffset (int) which is the byte offset to the opening
// delimiter within the current data slice.
//
// Returns TextPart which contains the parsed expression or literal text.
// Returns int which is the number of bytes read from the input.
// Returns bool which indicates whether a part was found.
func (s *interpolationScanner) scanExpressionPart(openDelimOffset int) (TextPart, int, bool) {
	expressionStartRelativeOffset := openDelimOffset + 2
	expressionStartAbsoluteOffset := s.cursor + expressionStartRelativeOffset
	closeDelimOffset := bytes.Index(s.data[expressionStartAbsoluteOffset:], []byte("}}"))

	if closeDelimOffset == -1 {
		absOpenDelimOffset := s.tokenStartOffset + s.cursor + openDelimOffset
		errRelLine, errRelCol := s.p.lexer.PositionAt(absOpenDelimOffset)
		errAbsLocation := s.p.adjustLocation(errRelLine, errRelCol)
		remainingString := string(s.data[s.cursor+openDelimOffset:])
		s.p.tree.Diagnostics = append(s.p.tree.Diagnostics, NewDiagnosticWithCode(
			Error, "Unterminated text interpolation: missing '}}'", remainingString,
			CodeUnterminatedInterpolation, errAbsLocation, s.p.sourcePath,
		))
		return TextPart{Expression: nil, GoAnnotations: nil, Literal: html.UnescapeString(remainingString), RawExpression: "", Location: Location{}, IsLiteral: true}, len(s.data) - s.cursor, true
	}

	expressionBytesWithWhitespace := s.data[expressionStartAbsoluteOffset : expressionStartAbsoluteOffset+closeDelimOffset]
	expressionBytes := bytes.TrimSpace(expressionBytesWithWhitespace)
	rawExprString := string(expressionBytes)
	consumedBytes := expressionStartRelativeOffset + closeDelimOffset + 2

	if len(expressionBytes) > 0 {
		leadingWhitespaceLen := bytes.Index(expressionBytesWithWhitespace, expressionBytes)
		expressionStartPosition := s.tokenStartOffset + expressionStartAbsoluteOffset + leadingWhitespaceLen
		expressionRelativeLine, expressionRelativeColumn := s.p.lexer.PositionAt(expressionStartPosition)

		return TextPart{
			Expression:    nil,
			GoAnnotations: nil,
			Literal:       "",
			RawExpression: rawExprString,
			Location:      s.p.adjustLocation(expressionRelativeLine, expressionRelativeColumn),
			IsLiteral:     false,
		}, consumedBytes, true
	}
	return TextPart{}, consumedBytes, false
}

// handlePlainText turns raw text into a text node after trimming extra spaces.
//
// Takes data ([]byte) which contains the raw text to process.
func (p *Parser) handlePlainText(data []byte) {
	tokenStartOffset := p.lexer.TokenStart()
	line, column := p.lexer.PositionAt(tokenStartOffset)
	finalLocation := p.adjustLocation(line, column)

	start, end := 0, len(data)
	for start < end && isSpace(rune(data[start])) {
		start++
	}
	for end > start && isSpace(rune(data[end-1])) {
		end--
	}

	if start == end {
		if bytes.ContainsRune(data, ' ') && !bytes.ContainsRune(data, '\n') {
			p.currentParent().Children = append(p.currentParent().Children, newTextNode(" ", finalLocation))
		}
		return
	}

	finalText := collapseWhitespace(data, start, end)
	if finalText != "" {
		p.currentParent().Children = append(p.currentParent().Children, newTextNode(html.UnescapeString(finalText), finalLocation))
	}
}

// processElement handles a start tag token and creates a new template node.
func (p *Parser) processElement() {
	tagNameBytes := p.lexer.Text()
	tagName := internBytes(p.toLower(tagNameBytes))

	if skipTopWrapper(tagName) {
		p.consumeUntilTagClose()
		return
	}

	line, column := p.lexer.TokenLine(), p.lexer.TokenCol()
	startLocation := p.adjustLocation(line, column)

	node := newElementNode(tagName, startLocation)

	isVoid := p.collectDirectivesAndAttrs(node)

	if tagName == "fragment" || (tagName == "template" && !hasAttribute(node, "shadowrootmode")) {
		node.NodeType = NodeFragment
	}

	p.currentParent().Children = append(p.currentParent().Children, node)

	if !isVoid {
		p.nodeStack = append(p.nodeStack, node)
	}
}

// processEndTag handles an end tag token by removing the top node from the
// stack.
func (p *Parser) processEndTag() {
	tagName := string(p.toLower(p.lexer.Text()))
	if skipTopWrapper(tagName) {
		return
	}

	if len(p.nodeStack) > 1 {
		closingNode := p.nodeStack[len(p.nodeStack)-1]

		startLine, startCol := p.lexer.PositionAt(p.lexer.TokenStart())
		endLine, endCol := p.lexer.PositionAt(p.lexer.TokenEnd())

		closingNode.ClosingTagRange = Range{
			Start: p.adjustLocation(startLine, startCol),
			End:   p.adjustLocation(endLine, endCol),
		}

		closingNode.NodeRange.End = closingNode.ClosingTagRange.End

		p.detectFormattingHint(closingNode)

		p.nodeStack = p.nodeStack[:len(p.nodeStack)-1]
	}
}

// detectFormattingHint checks how a node is laid out in the source code and
// sets PreferredFormat to keep the user's chosen style.
//
// Takes node (*TemplateNode) which is the element node to check.
//
// The method uses these rules:
//   - Same line for open and close tags -> FormatInline
//   - Children on separate lines from the opening tag -> FormatBlock
//   - Multiple children spread across several lines -> FormatBlock
//   - Otherwise -> FormatAuto (let the formatter decide)
func (*Parser) detectFormattingHint(node *TemplateNode) {
	if node == nil || node.NodeType != NodeElement {
		return
	}

	openingLine := node.OpeningTagRange.End.Line
	closingLine := node.ClosingTagRange.Start.Line

	if openingLine == closingLine {
		node.PreferredFormat = FormatInline
		return
	}

	if len(node.Children) > 0 {
		firstChild := node.Children[0]
		lastChild := node.Children[len(node.Children)-1]

		if firstChild.Location.Line > openingLine {
			node.PreferredFormat = FormatBlock
			return
		}

		if len(node.Children) > 1 {
			firstChildLine := firstChild.Location.Line
			lastChildLine := lastChild.NodeRange.End.Line
			if lastChildLine > firstChildLine {
				node.PreferredFormat = FormatBlock
				return
			}
		}
	}

	node.PreferredFormat = FormatAuto
}

// processComment handles a CommentToken by creating a comment node.
func (p *Parser) processComment() {
	commentContent := string(p.lexer.Text())
	line, column := p.lexer.PositionAt(p.lexer.TokenStart())
	endLine, endColumn := p.lexer.PositionAt(p.lexer.TokenEnd())
	startLocation := p.adjustLocation(line, column)
	endLocation := p.adjustLocation(endLine, endColumn)
	node := &TemplateNode{
		Key:                nil,
		DirKey:             nil,
		DirHTML:            nil,
		GoAnnotations:      nil,
		RuntimeAnnotations: nil,
		AttributeWriters:   nil,
		TextContentWriter:  nil,
		CustomEvents:       nil,
		OnEvents:           nil,
		Binds:              nil,
		DirContext:         nil,
		DirElse:            nil,
		DirText:            nil,
		DirStyle:           nil,
		DirClass:           nil,
		DirIf:              nil,
		DirElseIf:          nil,
		DirFor:             nil,
		DirShow:            nil,
		DirRef:             nil,
		DirModel:           nil,
		DirScaffold:        nil,
		TagName:            "",
		TextContent:        commentContent,
		InnerHTML:          "",
		Children:           nil,
		RichText:           nil,
		Attributes:         nil,
		Diagnostics:        nil,
		DynamicAttributes:  nil,
		Directives:         nil,
		Location:           p.adjustLocation(line, column),
		NodeRange:          Range{Start: startLocation, End: endLocation},
		OpeningTagRange:    Range{},
		ClosingTagRange:    Range{},
		NodeType:           NodeComment,
		IsPooled:           false,
		IsContentEditable:  false,
		PreferredFormat:    FormatAuto,
	}
	p.currentParent().Children = append(p.currentParent().Children, node)
}

// processForeignToken handles SVGToken and MathToken by re-parsing the raw
// XML content with encoding/xml and building a TemplateNode tree. This
// supports Piko directives (p-if, p-show, etc.) and dynamic attributes on
// SVG/MathML elements.
func (p *Parser) processForeignToken() {
	data := p.lexer.Text()
	line, column := p.lexer.TokenLine(), p.lexer.TokenCol()
	baseLocation := p.adjustLocation(line, column)

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false

	var stack []*TemplateNode

	foreignParent := func() *TemplateNode {
		if len(stack) > 0 {
			return stack[len(stack)-1]
		}
		return p.currentParent()
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			node := buildForeignElementNode(t, baseLocation)
			foreignParent().Children = append(foreignParent().Children, node)
			stack = append(stack, node)

		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			trimmed := strings.TrimSpace(string(t))
			if trimmed != "" {
				textNode := newTextNode(trimmed, baseLocation)
				foreignParent().Children = append(foreignParent().Children, textNode)
			}
		}
	}
}

// collectDirectivesAndAttrs goes through the attributes of an element.
//
// Takes tn (*TemplateNode) which receives the collected directives and
// attributes.
//
// Returns isVoidTag (bool) which indicates whether the element is a void tag.
func (p *Parser) collectDirectivesAndAttrs(tn *TemplateNode) (isVoidTag bool) {
	for {
		tt := p.lexer.Next()
		switch tt {
		case htmllexer.AttributeToken:
			p.processAttribute(tn)
		case htmllexer.StartTagVoidToken, htmllexer.ErrorToken:
			endLine, endCol := p.lexer.PositionAt(p.lexer.TokenEnd())
			endLocation := p.adjustLocation(endLine, endCol)

			tn.OpeningTagRange.End = endLocation
			tn.NodeRange.End = endLocation
			return true
		case htmllexer.StartTagCloseToken:
			endLine, endCol := p.lexer.PositionAt(p.lexer.TokenEnd())
			endLocation := p.adjustLocation(endLine, endCol)
			tn.OpeningTagRange.End = endLocation

			return isVoidElement(tn.TagName)
		default:
		}
	}
}

// processAttribute parses a single attribute and stores it in the node.
//
// Takes tn (*TemplateNode) which receives the parsed attribute data.
func (p *Parser) processAttribute(tn *TemplateNode) {
	keyBytes := p.lexer.Text()
	keyLine, keyCol := p.lexer.TokenLine(), p.lexer.TokenCol()
	keyLocation := p.adjustLocation(keyLine, keyCol)

	rawVal, valLocation := p.unquoteAttrVal()

	endOffset := p.lexer.TokenEnd()
	endLine, endCol := p.lexer.PositionAt(endOffset)
	attributeEndLocation := p.adjustLocation(endLine, endCol)

	attributeRange := Range{Start: keyLocation, End: attributeEndLocation}

	if bytes.EqualFold(keyBytes, []byte("contenteditable")) {
		if html.UnescapeString(rawVal) != "false" {
			tn.IsContentEditable = true
		}
	}

	if p.opts != nil && p.opts.RawMode {
		tn.Attributes = append(tn.Attributes, HTMLAttribute{
			Name:           internBytes(p.toLower(keyBytes)),
			Value:          rawVal,
			Location:       valLocation,
			NameLocation:   keyLocation,
			AttributeRange: attributeRange,
		})
		return
	}

	if d, found := interpretDirective(keyBytes, rawVal, valLocation); found {
		d.NameLocation = keyLocation
		d.AttributeRange = attributeRange
		tn.Directives = append(tn.Directives, d)
	} else if bytes.HasPrefix(keyBytes, []byte(":")) {
		tn.DynamicAttributes = append(tn.DynamicAttributes, DynamicAttribute{
			Expression:     nil,
			GoAnnotations:  nil,
			Name:           internBytes(keyBytes[1:]),
			RawExpression:  rawVal,
			Location:       valLocation,
			NameLocation:   keyLocation,
			AttributeRange: attributeRange,
		})
	} else {
		tn.Attributes = append(tn.Attributes, HTMLAttribute{
			Name:           internBytes(p.toLower(keyBytes)),
			Value:          rawVal,
			Location:       valLocation,
			NameLocation:   keyLocation,
			AttributeRange: attributeRange,
		})
	}
}

// unquoteAttrVal extracts the attribute value, removes surrounding quotes if
// present, and finds its exact position in the source.
//
// Returns string which is the attribute value with quotes removed.
// Returns Location which is the exact position of the value in the source.
func (p *Parser) unquoteAttrVal() (string, Location) {
	valBytes := p.lexer.AttrVal()
	valStartOffset := p.lexer.AttrValStart()
	var relLine, relCol int

	if valStartOffset != -1 {
		relLine, relCol = p.lexer.PositionAt(valStartOffset)
	} else {
		relLine, relCol = p.lexer.TokenLine(), p.lexer.TokenCol()
	}

	if len(valBytes) > 1 && (valBytes[0] == '"' || valBytes[0] == '\'') {
		relCol++
		return string(valBytes[1 : len(valBytes)-1]), p.adjustLocation(relLine, relCol)
	}
	return string(valBytes), p.adjustLocation(relLine, relCol)
}

// consumeUntilTagClose reads tokens from the lexer until it finds the end of
// the current start tag.
func (p *Parser) consumeUntilTagClose() {
	for {
		tt := p.lexer.Next()
		if tt == htmllexer.StartTagCloseToken || tt == htmllexer.StartTagVoidToken || tt == htmllexer.ErrorToken {
			break
		}
	}
}

// ParseAndTransform is the main entry point for the AST pipeline. It parses
// the template source, validates it, and applies all transformation passes.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes raw (string) which contains the template source to parse.
// Takes sourcePath (string) which identifies the file for error messages.
//
// Returns *TemplateAST which is the validated and transformed syntax tree.
// Returns error when parsing fails.
func ParseAndTransform(ctx context.Context, raw string, sourcePath string) (*TemplateAST, error) {
	newAst, err := Parse(ctx, raw, sourcePath, &Location{Line: 1, Column: 1, Offset: 0})
	if err != nil {
		return newAst, fmt.Errorf("parsing template: %w", err)
	}
	ValidateAST(newAst)
	TidyAST(ctx, newAst)
	return newAst, nil
}

// Parse parses a raw string into a TemplateAST.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes raw (string) which contains the template content to parse.
// Takes sourcePath (string) which identifies the source file for error
// messages.
// Takes startLocation (*Location) which offsets all line and column
// numbers if set.
//
// Returns *TemplateAST which contains the parsed template structure.
// Returns error when the template has invalid syntax.
func Parse(ctx context.Context, raw string, sourcePath string, startLocation *Location) (*TemplateAST, error) {
	return ParseWithOptions(ctx, raw, sourcePath, startLocation, nil)
}

// ParseWithOptions performs the initial parsing with configurable options.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes raw (string) which contains the template content to parse.
// Takes sourcePath (string) which identifies the source file for diagnostics.
// Takes startLocation (*Location) which optionally offsets all
// generated locations.
// Takes opts (*ParseOptions) which configures parser behaviour; nil uses
// defaults.
//
// Returns *TemplateAST which contains the parsed template structure.
// Returns error when parsing fails due to invalid syntax.
func ParseWithOptions(ctx context.Context, raw string, sourcePath string, startLocation *Location, opts *ParseOptions) (*TemplateAST, error) {
	var location Location
	if startLocation != nil {
		location = *startLocation
	} else {
		location = Location{Line: 1, Column: 1, Offset: 0}
	}

	p := &Parser{
		lexer: htmllexer.NewLexer([]byte(raw)),
		tree: &TemplateAST{
			SourcePath:        &sourcePath,
			ExpiresAtUnixNano: nil,
			Metadata:          nil,
			queryContext:      nil,
			RootNodes:         nil,
			Diagnostics:       nil,
			SourceSize:        int64(len(raw)),
			Tidied:            false,
			isPooled:          false,
		},
		sourcePath: sourcePath,
		nodeStack:  nil,
		startLine:  location.Line,
		startCol:   location.Column,
		opts:       opts,
	}

	err := p.run()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("running parser: %w", err)
	}

	applyTemplateTransformations(ctx, p.tree)
	return p.tree, nil
}

// normaliseForeignAttrName preserves SVG-spec case-sensitive attribute names
// and lowercases everything else for fast comparison at runtime.
//
// Takes raw (string) which is the attribute name to normalise.
//
// Returns string which is the original name if it is case-sensitive per the
// SVG spec, or the lowercased name otherwise.
func normaliseForeignAttrName(raw string) string {
	if _, ok := svgCaseSensitiveAttrs[raw]; ok {
		return raw
	}
	return strings.ToLower(raw)
}

// releaseScanner clears references and returns the scanner to the pool.
//
// Takes scanner (*interpolationScanner) which is the scanner to release.
func releaseScanner(scanner *interpolationScanner) {
	scanner.p = nil
	scanner.data = nil
	interpolationScannerPool.Put(scanner)
}

// newRichTextNode creates a text TemplateNode with rich text content.
//
// Takes parts ([]TextPart) which holds the rich text segments to include.
// Takes location (Location) which sets the source location for the node.
//
// Returns *TemplateNode which is a text node with the RichText field set.
func newRichTextNode(parts []TextPart, location Location) *TemplateNode {
	return &TemplateNode{
		Key:                nil,
		DirKey:             nil,
		DirHTML:            nil,
		GoAnnotations:      nil,
		RuntimeAnnotations: nil,
		AttributeWriters:   nil,
		TextContentWriter:  nil,
		CustomEvents:       nil,
		OnEvents:           nil,
		Binds:              nil,
		DirContext:         nil,
		DirElse:            nil,
		DirText:            nil,
		DirStyle:           nil,
		DirClass:           nil,
		DirIf:              nil,
		DirElseIf:          nil,
		DirFor:             nil,
		DirShow:            nil,
		DirRef:             nil,
		DirModel:           nil,
		DirScaffold:        nil,
		TagName:            "",
		TextContent:        "",
		InnerHTML:          "",
		Children:           nil,
		RichText:           parts,
		Attributes:         nil,
		Diagnostics:        nil,
		DynamicAttributes:  nil,
		Directives:         nil,
		Location:           location,
		NodeType:           NodeText,
		IsPooled:           false,
		IsContentEditable:  false,
		NodeRange:          Range{},
		OpeningTagRange:    Range{},
		ClosingTagRange:    Range{},
		PreferredFormat:    FormatAuto,
	}
}

// newTextNode creates a text TemplateNode with the given content and location.
//
// Takes textContent (string) which provides the text to display.
// Takes location (Location) which specifies where the node appears in the source.
//
// Returns *TemplateNode which is a text node ready for use in the template.
func newTextNode(textContent string, location Location) *TemplateNode {
	return &TemplateNode{
		Key: nil, DirKey: nil, DirHTML: nil, GoAnnotations: nil, RuntimeAnnotations: nil,
		AttributeWriters: nil, TextContentWriter: nil,
		CustomEvents: nil, OnEvents: nil, Binds: nil, DirContext: nil,
		DirElse: nil, DirText: nil, DirStyle: nil, DirClass: nil, DirIf: nil, DirElseIf: nil,
		DirFor: nil, DirShow: nil, DirRef: nil, DirSlot: nil, DirModel: nil, DirScaffold: nil,
		TagName: "", TextContent: textContent, InnerHTML: "", Children: nil,
		RichText: nil, Attributes: nil, Diagnostics: nil, DynamicAttributes: nil, Directives: nil,
		Location: location, NodeType: NodeText, IsPooled: false, IsContentEditable: false,
		NodeRange: Range{}, OpeningTagRange: Range{}, ClosingTagRange: Range{}, PreferredFormat: FormatAuto,
	}
}

// collapseWhitespace extracts text from a byte slice and replaces any group of
// whitespace characters with a single space.
//
// Takes data ([]byte) which is the source bytes to process.
// Takes start (int) which is the position of the first byte to include.
// Takes end (int) which is the position after the last byte to include.
//
// Returns string which is the text with whitespace groups replaced by single
// spaces.
func collapseWhitespace(data []byte, start, end int) string {
	var b strings.Builder
	b.Grow(end - start)

	inWhitespace := false
	for i := start; i < end; {
		r, size := utf8.DecodeRune(data[i:])
		if isSpace(r) {
			if !inWhitespace {
				_ = b.WriteByte(' ')
				inWhitespace = true
			}
		} else {
			_, _ = b.WriteRune(r)
			inWhitespace = false
		}
		i += size
	}
	collapsedText := b.String()

	if start > 0 && isSpace(rune(data[0])) {
		collapsedText = " " + collapsedText
	}
	if end < len(data) && isSpace(rune(data[len(data)-1])) {
		collapsedText = collapsedText + " "
	}

	return collapsedText
}

// hasAttribute returns true if the node has an HTML attribute with the given
// name (case-insensitive comparison).
//
// Takes node (*TemplateNode) which is the node to search.
// Takes name (string) which is the attribute name to look for.
//
// Returns bool which is true if a matching attribute is found.
func hasAttribute(node *TemplateNode, name string) bool {
	for i := range node.Attributes {
		if strings.EqualFold(node.Attributes[i].Name, name) {
			return true
		}
	}
	return false
}

// newElementNode creates an element TemplateNode with the given tag name and
// location.
//
// Takes tagName (string) which specifies the HTML element tag name.
// Takes location (Location) which specifies the source location of the element.
//
// Returns *TemplateNode which is the new element node with default values set.
func newElementNode(tagName string, location Location) *TemplateNode {
	return &TemplateNode{
		Key: nil, DirKey: nil, DirHTML: nil, GoAnnotations: nil, RuntimeAnnotations: nil,
		AttributeWriters: nil, TextContentWriter: nil,
		CustomEvents: nil, OnEvents: nil, Binds: nil, DirContext: nil,
		DirElse: nil, DirText: nil, DirStyle: nil, DirClass: nil, DirIf: nil, DirElseIf: nil,
		DirFor: nil, DirShow: nil, DirRef: nil, DirSlot: nil, DirModel: nil, DirScaffold: nil,
		TagName: tagName, TextContent: "", InnerHTML: "", Children: nil,
		RichText: nil, Attributes: nil, Diagnostics: nil, DynamicAttributes: nil, Directives: nil,
		Location: location, NodeType: NodeElement, IsPooled: false, IsContentEditable: false,
		NodeRange: Range{Start: location, End: location}, OpeningTagRange: Range{Start: location, End: location},
		ClosingTagRange: Range{}, PreferredFormat: FormatAuto,
	}
}

// buildForeignElementNode creates a TemplateNode from an XML start element,
// classifying each attribute as a directive, dynamic binding, or static
// attribute.
//
// Takes element (xml.StartElement) which provides the tag name and attributes.
// Takes baseLocation (Location) which provides the source location for all parsed
// items.
//
// Returns *TemplateNode which is the fully populated element node.
func buildForeignElementNode(element xml.StartElement, baseLocation Location) *TemplateNode {
	tagName := internBytes([]byte(strings.ToLower(element.Name.Local)))
	node := newElementNode(tagName, baseLocation)

	for _, attr := range element.Attr {
		classifyForeignAttribute(node, attr, baseLocation)
	}

	return node
}

// classifyForeignAttribute adds a single XML attribute to the node as either
// a directive, dynamic attribute, or static HTML attribute.
//
// Takes node (*TemplateNode) which receives the classified attribute.
// Takes attr (xml.Attr) which is the attribute to classify.
// Takes baseLocation (Location) which provides the source location.
func classifyForeignAttribute(node *TemplateNode, attr xml.Attr, baseLocation Location) {
	attributeName := attr.Name.Local
	attributeValue := attr.Value
	keyBytes := []byte(attributeName)

	if d, found := interpretDirective(keyBytes, attributeValue, baseLocation); found {
		d.NameLocation = baseLocation
		node.Directives = append(node.Directives, d)
	} else if len(attributeName) > 0 && attributeName[0] == ':' {
		node.DynamicAttributes = append(node.DynamicAttributes, DynamicAttribute{
			Name:          internBytes([]byte(attributeName[1:])),
			RawExpression: attributeValue,
			Location:      baseLocation,
		})
	} else {
		node.Attributes = append(node.Attributes, HTMLAttribute{
			Name:  normaliseForeignAttrName(attributeName),
			Value: attributeValue,
		})
	}
}

// isRawTextParent checks if the given node needs its text content treated as
// raw text.
//
// Takes node (*TemplateNode) which is the node to check.
//
// Returns bool which is true if the node is a pre, code, or textarea element,
// or if it has contenteditable set.
func isRawTextParent(node *TemplateNode) bool {
	return node.TagName == "pre" || node.TagName == "code" || node.TagName == "textarea" || node.IsContentEditable
}

// isSpace reports whether a rune is a whitespace character.
//
// Takes r (rune) which is the character to check.
//
// Returns bool which is true if r is a space, tab, newline, or carriage
// return.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// skipTopWrapper reports whether a tag is a top-level document wrapper that
// should be skipped during processing.
//
// Takes tagName (string) which is the HTML tag name to check.
//
// Returns bool which is true when the tag is html, head, or body.
func skipTopWrapper(tagName string) bool {
	return tagName == "html" || tagName == "head" || tagName == "body"
}

// isVoidElement reports whether a tag is a void element (e.g. input, br, img).
//
// Takes tagName (string) which is the HTML tag name to check.
//
// Returns bool which is true if the tag is a void element.
func isVoidElement(tagName string) bool {
	switch tagName {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}

// interpretDirective finds a Piko directive from an HTML attribute.
//
// Takes key ([]byte) which is the attribute name to check for directives.
// Takes value (string) which is the attribute value with the expression.
// Takes location (Location) which is the source location for error reporting.
//
// Returns Directive which is the parsed directive if one was found.
// Returns bool which is true when a valid directive was found.
func interpretDirective(key []byte, value string, location Location) (Directive, bool) {
	if d, ok := parsePrefixedDirective(key, value, location); ok {
		return d, true
	}

	if dirType, found := resolveDirectiveType(key); found {
		d := Directive{
			Expression:     nil,
			ChainKey:       nil,
			GoAnnotations:  nil,
			Arg:            "",
			Modifier:       "",
			RawExpression:  value,
			Location:       location,
			NameLocation:   Location{},
			AttributeRange: Range{},
			Type:           dirType,
		}

		if d.Type == DirectiveElse {
			d.RawExpression = ""
		} else if flagDirectives[d.Type] {
			handleFlagDirective(&d)
		}

		return d, true
	}

	return Directive{}, false
}

// handleFlagDirective sets a directive's raw expression to a boolean string.
//
// When the value is empty or "true", it sets the expression to "true". When the
// value is "false", it sets it to "false". Any other value is left unchanged so
// it can be parsed as a dynamic expression.
//
// Takes d (*Directive) which is the directive to process.
func handleFlagDirective(d *Directive) {
	trimmedVal := strings.TrimSpace(d.RawExpression)
	lowerVal := strings.ToLower(trimmedVal)

	switch lowerVal {
	case "", "true":
		d.RawExpression = "true"
	case "false":
		d.RawExpression = "false"
	}
}

// parsePrefixedDirective checks for and parses directive prefixes such as
// p-on:, p-event:, and p-bind: from an attribute name.
//
// Takes key ([]byte) which is the attribute name to check for a prefix.
// Takes value (string) which is the expression value for the directive.
// Takes location (Location) which specifies where the directive appears.
//
// Returns Directive which contains the parsed directive with its type and
// argument.
// Returns bool which is true when a prefixed directive was found.
func parsePrefixedDirective(key []byte, value string, location Location) (Directive, bool) {
	d := Directive{
		Expression:     nil,
		ChainKey:       nil,
		GoAnnotations:  nil,
		Arg:            "",
		Modifier:       "",
		RawExpression:  value,
		Location:       location,
		NameLocation:   Location{},
		AttributeRange: Range{},
		Type:           0,
	}

	switch {
	case bytes.HasPrefix(key, []byte("p-on:")):
		base := string(key[5:])
		parts := strings.Split(base, ".")
		d.Type, d.Arg = DirectiveOn, parts[0]
		if len(parts) > 1 {
			d.EventModifiers = parts[1:]
		}
		return d, true
	case bytes.HasPrefix(key, []byte("p-event:")):
		base := string(key[8:])
		parts := strings.Split(base, ".")
		d.Type, d.Arg = DirectiveEvent, parts[0]
		if len(parts) > 1 {
			d.EventModifiers = parts[1:]
		}
		return d, true
	case bytes.HasPrefix(key, []byte("p-bind:")):
		d.Type = DirectiveBind
		d.Arg = string(key[7:])
		return d, true
	case bytes.HasPrefix(key, []byte("p-timeline:")):
		d.Type = DirectiveTimeline
		d.Arg = string(key[11:])
		return d, true
	}

	return Directive{}, false
}
