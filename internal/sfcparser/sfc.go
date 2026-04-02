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

package sfcparser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"piko.sh/piko/internal/htmllexer"
)

const (
	// tagNameTemplate is the HTML tag name for Piko template blocks.
	tagNameTemplate = "template"

	// tagNameScript is the tag name for script blocks in SFC files.
	tagNameScript = "script"

	// tagNameStyle is the tag name for style blocks in Piko SFC files.
	tagNameStyle = "style"

	// tagNameI18n is the tag name for i18n blocks in SFC files.
	tagNameI18n = "i18n"

	// tagNameTimeline is the tag name for piko:timeline blocks in SFC files.
	tagNameTimeline = "piko:timeline"
)

// parser processes HTML tokens and builds a structured parse result.
type parser struct {
	// lexer tokenises HTML input for parsing.
	lexer *htmllexer.Lexer

	// result holds the parsed template, script, and style blocks.
	result *ParseResult

	// tagHandlers maps tag names to their handler functions.
	tagHandlers map[string]func(Location) error
}

// initialiseTagHandlers sets up the map of tag handlers for the parser.
func (p *parser) initialiseTagHandlers() {
	p.tagHandlers = map[string]func(Location) error{
		tagNameTemplate: p.handleTemplateTag,
		tagNameScript:   p.handleScriptTag,
		tagNameStyle:    p.handleStyleTag,
		tagNameI18n:     p.handleI18nTag,
		tagNameTimeline: p.handleTimelineTag,
	}
}

// parse processes the token stream and returns the parsed result.
//
// Returns *ParseResult which contains the parsed output.
// Returns error when an error token is found or tag handling fails.
func (p *parser) parse() (*ParseResult, error) {
	for {
		tt := p.lexer.Next()

		switch tt {
		case htmllexer.ErrorToken:
			return p.handleErrorToken()

		case htmllexer.StartTagToken:
			result, err := p.handleStartTag()
			if err != nil {
				return nil, fmt.Errorf("handling SFC start tag: %w", err)
			}
			if result != nil {
				return result, nil
			}

		default:
		}
	}
}

// handleErrorToken processes an error token from the lexer.
//
// Returns *ParseResult which contains the parsed result when the error is EOF
// or when no error occurred.
// Returns error when the lexer encountered an error that is not EOF.
func (p *parser) handleErrorToken() (*ParseResult, error) {
	err := p.lexer.Err()
	if errors.Is(err, io.EOF) {
		return p.result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lexing SFC input: %w", err)
	}
	return p.result, nil
}

// handleStartTag processes a start tag found during parsing.
//
// Returns *ParseResult which is the parse result when the handler signals EOF.
// Returns error when the tag handler fails.
func (p *parser) handleStartTag() (*ParseResult, error) {
	line, column := p.lexer.PositionAt(p.lexer.TokenStart())

	handler, ok := p.tagHandlers[string(bytes.ToLower(p.lexer.Text()))]
	if !ok {
		return nil, nil
	}

	err := handler(Location{Line: line, Column: column})
	if err == nil {
		return nil, nil
	}
	if errors.Is(err, io.EOF) {
		return p.result, nil
	}
	return nil, fmt.Errorf("handling SFC tag: %w", err)
}

// handleTemplateTag processes a template tag and stores its content.
//
// Takes location (Location) which specifies where the tag begins.
//
// Returns error when attribute parsing fails.
func (p *parser) handleTemplateTag(location Location) error {
	tagName := tagNameTemplate
	if p.result.Template == "" {
		attrs, isVoid, err := p.parseAttributes()
		if err != nil {
			return fmt.Errorf("parsing template tag attributes: %w", err)
		}

		p.result.TemplateAttributes = attrs

		if isVoid {
			p.result.TemplateLocation = location
			return nil
		}

		contentLine, contentCol := p.lexer.PositionAt(p.lexer.TokenEnd())
		p.result.TemplateLocation = location
		p.result.TemplateContentLocation = Location{Line: contentLine, Column: contentCol}
		p.result.Template = p.extractArbitraryContent(tagName)
	} else {
		p.consumeAndDiscardContent(tagName)
	}
	return nil
}

// handleScriptTag parses a script tag and adds it to the result.
//
// Takes location (Location) which specifies where the script tag starts.
//
// Returns error when the script content cannot be parsed.
func (p *parser) handleScriptTag(location Location) error {
	attrs, content, contentLocation, err := p.parseBlockContent(tagNameScript)
	if err != nil {
		return fmt.Errorf("parsing script tag content: %w", err)
	}
	p.result.Scripts = append(p.result.Scripts, Script{
		Content:         content,
		Attributes:      attrs,
		Location:        location,
		ContentLocation: contentLocation,
	})
	return nil
}

// handleStyleTag parses a style tag and adds it to the parse result.
//
// Takes location (Location) which specifies the position of the style tag.
//
// Returns error when the block content cannot be parsed.
func (p *parser) handleStyleTag(location Location) error {
	attrs, content, contentLocation, err := p.parseBlockContent(tagNameStyle)
	if err != nil {
		return fmt.Errorf("parsing style tag content: %w", err)
	}
	p.result.Styles = append(p.result.Styles, Style{
		Content:         content,
		Attributes:      attrs,
		Location:        location,
		ContentLocation: contentLocation,
	})
	return nil
}

// handleI18nTag parses an i18n tag and adds it to the result.
//
// Takes location (Location) which specifies where the tag starts.
//
// Returns error when the block content cannot be parsed.
func (p *parser) handleI18nTag(location Location) error {
	attrs, content, contentLocation, err := p.parseBlockContent(tagNameI18n)
	if err != nil {
		return fmt.Errorf("parsing i18n tag content: %w", err)
	}
	p.result.I18nBlocks = append(p.result.I18nBlocks, I18nBlock{
		Content:         content,
		Attributes:      attrs,
		Location:        location,
		ContentLocation: contentLocation,
	})
	return nil
}

// handleTimelineTag parses a piko:timeline tag and appends it to the list of
// timeline blocks. Multiple blocks are supported so that each can target a
// different viewport via a media attribute.
//
// Takes location (Location) which specifies where the tag starts.
//
// Returns error when the block content cannot be parsed.
func (p *parser) handleTimelineTag(location Location) error {
	attrs, content, contentLocation, err := p.parseBlockContent(tagNameTimeline)
	if err != nil {
		return fmt.Errorf("parsing piko:timeline tag content: %w", err)
	}
	p.result.Timelines = append(p.result.Timelines, TimelineBlock{
		Content:         content,
		Attributes:      attrs,
		Location:        location,
		ContentLocation: contentLocation,
	})
	return nil
}

// parseBlockContent parses attributes and extracts content for a block tag.
//
// Takes tagName (string) which specifies the tag name for content extraction.
//
// Returns map[string]string which contains the parsed attributes.
// Returns string which holds the extracted content, or empty if the tag has no
// body.
// Returns Location which marks where the content starts in the source.
// Returns error when attribute parsing fails.
func (p *parser) parseBlockContent(tagName string) (map[string]string, string, Location, error) {
	attrs, isVoid, err := p.parseAttributes()
	if err != nil {
		return nil, "", Location{}, err
	}

	if isVoid {
		return attrs, "", Location{}, nil
	}

	contentLine, contentCol := p.lexer.PositionAt(p.lexer.TokenEnd())
	contentLocation := Location{Line: contentLine, Column: contentCol}
	content := p.extractArbitraryContent(tagName)

	return attrs, content, contentLocation, nil
}

// parseAttributes reads attribute tokens until the tag closes.
//
// Returns map[string]string which contains the parsed attribute key-value
// pairs.
// Returns bool which is true when the tag is self-closing.
// Returns error when the lexer encounters an error.
func (p *parser) parseAttributes() (map[string]string, bool, error) {
	attrs := make(map[string]string)
	for {
		tt := p.lexer.Next()
		switch tt {
		case htmllexer.AttributeToken:
			key := strings.ToLower(string(p.lexer.Text()))
			value := unquoteAttrVal(p.lexer.AttrVal())
			attrs[key] = value
		case htmllexer.StartTagCloseToken:
			return attrs, false, nil
		case htmllexer.StartTagVoidToken:
			return attrs, true, nil
		case htmllexer.ErrorToken:
			return attrs, true, p.lexer.Err()
		default:
		}
	}
}

// extractArbitraryContent extracts content from inside a tag based on its type.
//
// Takes parentTagName (string) which specifies the containing tag name.
//
// Returns string which is the extracted content.
func (p *parser) extractArbitraryContent(parentTagName string) string {
	startOffset := p.lexer.TokenEnd()
	isRawTextElement := parentTagName == tagNameScript || parentTagName == tagNameStyle

	if isRawTextElement {
		return p.extractRawTextContent(parentTagName, startOffset)
	}

	return p.extractNestableContent(parentTagName, startOffset)
}

// extractRawTextContent extracts the content of script and style blocks from
// the input starting at the given offset.
//
// Raw text elements have special parsing rules where the lexer stops at the
// first closing tag, even within strings. This method finds the correct closing
// tag and advances the lexer past the block.
//
// Takes tagName (string) which specifies the tag type (e.g. "script", "style").
// Takes startOffset (int) which indicates where to begin searching in the input.
//
// Returns string which contains the raw text content of the block.
func (p *parser) extractRawTextContent(tagName string, startOffset int) string {
	searchBytes := p.lexer.SourceBytes()[startOffset:]
	contentEndOffset := p.findRawTextEndOffset(tagName, startOffset, searchBytes)
	content := p.lexer.SourceBytes()[startOffset:contentEndOffset]
	p.advanceLexerPastBlock(contentEndOffset)
	return string(content)
}

// findRawTextEndOffset locates the closing tag for a raw text element. It
// searches for the last closing tag before the next opening tag to handle
// cases where the tag name appears in string content.
//
// Takes tagName (string) which specifies the HTML tag name to find.
// Takes startOffset (int) which is the position to calculate the final offset
// from.
// Takes searchBytes ([]byte) which contains the bytes to search within.
//
// Returns int which is the offset of the closing tag, or the input length if
// no closing tag is found.
func (p *parser) findRawTextEndOffset(tagName string, startOffset int, searchBytes []byte) int {
	endTag := []byte("</" + tagName)
	nextStartTag := []byte("<" + tagName)

	searchEndRange := p.determineSearchRange(searchBytes, nextStartTag)
	endOffsetInSlice := bytes.LastIndex(searchEndRange, endTag)

	if endOffsetInSlice == -1 {
		endOffsetInSlice = bytes.LastIndex(searchBytes, endTag)
	}

	if endOffsetInSlice == -1 {
		return len(p.lexer.SourceBytes())
	}

	return startOffset + endOffsetInSlice
}

// determineSearchRange limits the search to find the closing tag by looking
// only before the next opening tag. This means the correct closing tag is
// found for the current block.
//
// Takes searchBytes ([]byte) which contains the bytes to search within.
// Takes nextStartTag ([]byte) which is the next opening tag to look for.
//
// Returns []byte which is the part of searchBytes before nextStartTag, or the
// original searchBytes if nextStartTag is not found.
func (*parser) determineSearchRange(searchBytes, nextStartTag []byte) []byte {
	before, _, found := bytes.Cut(searchBytes, nextStartTag)
	if found {
		return before
	}
	return searchBytes
}

// advanceLexerPastBlock moves the lexer forward until it has passed the content
// block, keeping the lexer in sync with the parsed content.
//
// Takes contentEndOffset (int) which specifies the end position of the content.
func (p *parser) advanceLexerPastBlock(contentEndOffset int) {
	fullBlockEndOffset := p.findBlockEndOffset(contentEndOffset)

	for p.lexer.TokenEnd() < fullBlockEndOffset {
		token := p.lexer.Next()
		if token == htmllexer.ErrorToken {
			break
		}
	}
}

// findBlockEndOffset finds the end of the full block, including the closing
// tag.
//
// Takes contentEndOffset (int) which is the position where the content ends.
//
// Returns int which is the position after the closing bracket, or the end of
// the input if no closing bracket is found.
func (p *parser) findBlockEndOffset(contentEndOffset int) int {
	closingBracketIndex := bytes.IndexByte(p.lexer.SourceBytes()[contentEndOffset:], '>')
	if closingBracketIndex != -1 {
		return contentEndOffset + closingBracketIndex + 1
	}
	return len(p.lexer.SourceBytes())
}

// extractNestableContent extracts the content of template and i18n blocks.
// These elements can be nested, so we track the nesting level to find the
// matching closing tag.
//
// Takes parentTagName (string) which specifies the tag type to match.
// Takes startOffset (int) which marks the starting position in the input.
//
// Returns string which contains the content between the opening and closing
// tags.
func (p *parser) extractNestableContent(parentTagName string, startOffset int) string {
	level := 1
	for {
		tt := p.lexer.Next()

		if tt == htmllexer.ErrorToken {
			return string(p.lexer.SourceBytes()[startOffset:p.lexer.TokenStart()])
		}

		level = p.updateNestingLevel(tt, parentTagName, level)

		if level == 0 {
			endOffset := p.lexer.TokenStart()
			return string(p.lexer.SourceBytes()[startOffset:endOffset])
		}
	}
}

// updateNestingLevel adjusts the nesting level based on start and end tags.
//
// Takes tokenType (htmllexer.TokenType) which is the type of HTML token.
// Takes tagName (string) which is the name of the HTML tag.
// Takes currentLevel (int) which is the current nesting depth.
//
// Returns int which is the new nesting level after processing the tag.
func (p *parser) updateNestingLevel(tokenType htmllexer.TokenType, tagName string, currentLevel int) int {
	switch tokenType {
	case htmllexer.StartTagToken:
		if p.isMatchingTag(tagName) {
			return currentLevel + 1
		}
		return currentLevel
	case htmllexer.EndTagToken:
		if p.isMatchingTag(tagName) {
			return currentLevel - 1
		}
		return currentLevel
	default:
		return currentLevel
	}
}

// isMatchingTag checks if the current lexer token matches the given tag name.
//
// Takes tagName (string) which specifies the tag name to match against.
//
// Returns bool which is true if the token text, when lowercased, equals the
// tag name.
func (p *parser) isMatchingTag(tagName string) bool {
	return string(bytes.ToLower(p.lexer.Text())) == tagName
}

// consumeAndDiscardContent skips all tokens until the matching closing tag.
//
// Takes parentTagName (string) which specifies the tag name to match for
// tracking how deeply nested the tags are.
func (p *parser) consumeAndDiscardContent(parentTagName string) {
	level := 1
	for {
		tt := p.lexer.Next()
		switch tt {
		case htmllexer.ErrorToken:
			return
		case htmllexer.StartTagToken:
			if string(bytes.ToLower(p.lexer.Text())) == parentTagName {
				level++
			}
		case htmllexer.EndTagToken:
			if string(bytes.ToLower(p.lexer.Text())) == parentTagName {
				level--
			}
		default:
		}
		if level == 0 {
			return
		}
	}
}

// Parse extracts template, script, style, and i18n blocks from SFC content.
//
// Takes data ([]byte) which contains the raw SFC content to parse.
//
// Returns *ParseResult which holds the parsed template, scripts, styles,
// and i18n blocks with their locations.
// Returns error when parsing fails.
func Parse(data []byte) (*ParseResult, error) {
	p := &parser{
		lexer:       htmllexer.NewLexer(data),
		tagHandlers: nil,
		result: &ParseResult{
			Template:                "",
			TemplateAttributes:      nil,
			Scripts:                 make([]Script, 0),
			Styles:                  make([]Style, 0),
			I18nBlocks:              make([]I18nBlock, 0),
			TemplateLocation:        Location{},
			TemplateContentLocation: Location{},
		},
	}
	p.initialiseTagHandlers()
	return p.parse()
}

// unquoteAttrVal removes surrounding quotes from an attribute value.
//
// Takes valBytes ([]byte) which is the raw attribute value bytes.
//
// Returns string which is the value without quotes, or an empty string if nil.
func unquoteAttrVal(valBytes []byte) string {
	if valBytes == nil {
		return ""
	}
	if len(valBytes) >= 2 && (valBytes[0] == '"' || valBytes[0] == '\'') {
		return string(valBytes[1 : len(valBytes)-1])
	}
	return string(valBytes)
}
