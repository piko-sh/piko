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

package formatter_domain

import (
	"maps"
	"slices"
	"strings"

	"golang.org/x/net/html"

	"piko.sh/piko/internal/sfcparser"
)

const (
	// templateIndent is the prefix used to indent each line in a template block.
	templateIndent = "  "

	// newline is the line separator used when reassembling SFC blocks.
	newline = "\n"

	// space is the space character used between HTML tag attributes.
	space = " "

	// quote is the quotation mark used to wrap attribute values.
	quote = `"`
	// tagEnd is the character that closes an HTML start tag.
	tagEnd = ">"
	// attributeAssign is the sign placed between an attribute name and its value.
	attributeAssign = "="
)

// reassembleSFC joins formatted blocks into a complete .pk file.
//
// Takes source (*sfcparser.ParseResult) which supplies wrapper attributes and timeline
// blocks.
// Takes formattedTemplate (string) which is the formatted template HTML.
// Takes formattedScripts ([]sfcparser.Script) which contains the formatted script blocks.
// Takes formattedStyles ([]sfcparser.Style) which contains the formatted style blocks.
// Takes formattedI18nBlocks ([]sfcparser.I18nBlock) which contains the formatted
// translation blocks.
//
// Returns string which is the reassembled .pk file content.
func reassembleSFC(
	source *sfcparser.ParseResult,
	formattedTemplate string,
	formattedScripts []sfcparser.Script,
	formattedStyles []sfcparser.Style,
	formattedI18nBlocks []sfcparser.I18nBlock,
) string {
	var builder strings.Builder

	writeTemplateBlock(&builder, formattedTemplate, source.TemplateAttributes)
	writeScriptBlocks(&builder, formattedScripts)
	writeStyleBlocks(&builder, formattedStyles)
	writeI18nBlocks(&builder, formattedI18nBlocks)
	for _, timeline := range source.Timelines {
		builder.WriteString(newline + "<piko:timeline")
		writeAttributes(&builder, timeline.Attributes)
		builder.WriteString(tagEnd)
		builder.WriteString(timeline.Content)
		builder.WriteString("</piko:timeline>" + newline)
	}

	return builder.String()
}

// writeTemplateBlock writes the formatted template content wrapped in <template> tags.
//
// Wrapper indentation preserves continuation lines in whitespace-sensitive elements. Tags
// inside interpolations count as expression text when indenting the markup.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes formattedTemplate (string) which contains the template content to wrap.
// Takes attributes (map[string]string) which contains the template wrapper attributes.
func writeTemplateBlock(builder *strings.Builder, formattedTemplate string, attributes map[string]string) {
	if formattedTemplate == "" {
		return
	}

	builder.WriteString("<template")
	writeAttributes(builder, attributes)
	builder.WriteString(tagEnd + newline)

	source := strings.TrimSpace(formattedTemplate)
	nextLine := true
	interpolating := false
	var preserved []string
	tokens := html.NewTokenizer(strings.NewReader(source))
	offset := 0
	for {
		kind := tokens.Next()
		end := offset + len(tokens.Raw())
		raw := source[offset:end]
		offset = end
		inTag := !interpolating && (kind == html.StartTagToken || kind == html.SelfClosingTagToken)
		nextLine = writeIndentedToken(builder, raw, nextLine, len(preserved) == 0, inTag)
		if kind == html.ErrorToken {
			break
		}
		switch {
		case interpolating:
			interpolating = nextInterpolationState(raw, true, kind == html.TextToken)
		case kind == html.TextToken && len(preserved) == 0:
			interpolating = nextInterpolationState(raw, false, true)
		default:
			preserved = trackPreservedElements(tokens, kind, preserved)
		}
	}
	builder.WriteString(newline + "</template>" + newline)
}

// writeIndentedToken writes one token, indenting each line that starts outside a quoted
// attribute value, so a value spanning several lines is never re-indented.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes raw (string) which is the token's source text.
// Takes nextLine (bool) which reports whether the output is at the start of a line.
// Takes indent (bool) which reports whether lines may be indented here.
// Takes inTag (bool) which reports whether the token is a start tag with attributes.
//
// Returns bool which reports whether the output ends at the start of a line.
func writeIndentedToken(builder *strings.Builder, raw string, nextLine, indent, inTag bool) bool {
	var quote byte
	for index := range len(raw) {
		character := raw[index]
		if nextLine && character != '\n' {
			if indent && quote == 0 {
				builder.WriteString(templateIndent)
			}
			nextLine = false
		}
		builder.WriteByte(character)
		if inTag {
			quote = nextQuoteState(quote, character)
		}
		if character == '\n' {
			nextLine = true
		}
	}
	return nextLine
}

// nextQuoteState tracks whether a start tag is inside a quoted attribute value.
//
// Takes quote (byte) which is the open quote character, or zero outside a value.
// Takes character (byte) which is the next character of the tag.
//
// Returns byte which is the quote state after the character.
func nextQuoteState(quote, character byte) byte {
	switch {
	case quote == 0 && (character == '"' || character == '\''):
		return character
	case character == quote:
		return 0
	default:
		return quote
	}
}

// nextInterpolationState tracks whether the template source is inside a "{{ ... }}"
// interpolation after a token.
//
// Takes raw (string) which is the token's source text.
// Takes open (bool) which reports whether an interpolation is open before the token.
// Takes canOpen (bool) which reports whether the token is text, where "{{" opens an
// interpolation. In a tag, braces belong to an attribute value.
//
// Returns bool which reports whether an interpolation is open after the token.
func nextInterpolationState(raw string, open, canOpen bool) bool {
	for {
		if open {
			_, after, found := strings.Cut(raw, "}}")
			if !found {
				return true
			}
			raw, open = after, false
		}
		if !canOpen {
			return false
		}
		_, after, found := strings.Cut(raw, "{{")
		if !found {
			return false
		}
		raw, open = after, true
	}
}

// trackPreservedElements updates the stack of open elements whose content keeps its own
// layout.
//
// Takes tokens (*html.Tokenizer) which is positioned on the current token.
// Takes kind (html.TokenType) which is the current token's type.
// Takes preserved ([]string) which holds the open preserved element names.
//
// Returns []string which is the updated stack.
func trackPreservedElements(tokens *html.Tokenizer, kind html.TokenType, preserved []string) []string {
	switch kind {
	case html.StartTagToken:
		name, hasAttributes := tokens.TagName()
		tag := string(name)
		if len(preserved) > 0 {
			if tag == preserved[len(preserved)-1] {
				preserved = append(preserved, tag)
			}
			return preserved
		}
		if isWhitespaceSensitive(tag) || isRawTextElement(tag) || (hasAttributes && hasContentEditable(tokens)) {
			preserved = append(preserved, tag)
		}
	case html.EndTagToken:
		name, _ := tokens.TagName()
		if len(preserved) > 0 && string(name) == preserved[len(preserved)-1] {
			preserved = preserved[:len(preserved)-1]
		}
	default:
	}
	return preserved
}

// hasContentEditable reports whether the current start tag makes its element editable.
//
// Takes tokens (*html.Tokenizer) which is positioned on a start tag with attributes.
//
// Returns bool which is true when contenteditable is present and not "false".
func hasContentEditable(tokens *html.Tokenizer) bool {
	for {
		key, value, more := tokens.TagAttr()
		if strings.EqualFold(string(key), "contenteditable") && !strings.EqualFold(string(value), "false") {
			return true
		}
		if !more {
			return false
		}
	}
}

// writeScriptBlocks writes all script blocks with their attributes and content to the
// output.
//
// Takes builder (*strings.Builder) which collects the output text.
// Takes formattedScripts ([]sfcparser.Script) which holds the script blocks to write.
func writeScriptBlocks(builder *strings.Builder, formattedScripts []sfcparser.Script) {
	for i := range formattedScripts {
		script := &formattedScripts[i]
		builder.WriteString(newline + "<script")
		writeAttributes(builder, script.Attributes)
		builder.WriteString(tagEnd + newline)
		writeBlockContent(builder, script.Content)
		builder.WriteString("</script>" + newline)
	}
}

// writeStyleBlocks writes all style blocks with their attributes and content.
//
// Takes builder (*strings.Builder) which collects the output text.
// Takes formattedStyles ([]sfcparser.Style) which holds the style blocks to write.
func writeStyleBlocks(builder *strings.Builder, formattedStyles []sfcparser.Style) {
	for i := range formattedStyles {
		style := &formattedStyles[i]
		builder.WriteString(newline + "<style")
		writeAttributes(builder, style.Attributes)
		builder.WriteString(tagEnd + newline)
		writeBlockContent(builder, style.Content)
		builder.WriteString("</style>" + newline)
	}
}

// writeI18nBlocks writes all i18n blocks with their attributes and content.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes formattedI18nBlocks ([]sfcparser.I18nBlock) which contains the i18n blocks to
// write.
func writeI18nBlocks(builder *strings.Builder, formattedI18nBlocks []sfcparser.I18nBlock) {
	for i := range formattedI18nBlocks {
		i18n := &formattedI18nBlocks[i]
		builder.WriteString(newline + "<i18n")
		writeAttributes(builder, i18n.Attributes)
		builder.WriteString(tagEnd + newline)
		writeBlockContent(builder, i18n.Content)
		builder.WriteString("</i18n>" + newline)
	}
}

// writeAttributes writes HTML attributes to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted attributes.
// Takes attributes (map[string]string) which contains the key-value pairs to write.
func writeAttributes(builder *strings.Builder, attributes map[string]string) {
	for _, key := range slices.Sorted(maps.Keys(attributes)) {
		value := attributes[key]
		if value == "" {
			builder.WriteString(space)
			builder.WriteString(key)
		} else {
			builder.WriteString(space)
			builder.WriteString(key)
			builder.WriteString(attributeAssign)
			builder.WriteString(formatRawAttributeValue(value))
		}
	}
}

// formatRawAttributeValue quotes an attribute without changing its source spelling.
//
// A value holding both quote characters, such as an unquoted HTML attribute or a decoded
// SVG or MathML attribute, is double-quoted with each double quote written as "&quot;".
// Expression values are decoded before parsing, so the reference keeps their meaning.
//
// Takes value (string) which is the raw parsed attribute value.
//
// Returns string which is the quoted value.
func formatRawAttributeValue(value string) string {
	if !strings.Contains(value, quote) {
		return quote + value + quote
	}
	if !strings.Contains(value, "'") {
		return "'" + value + "'"
	}
	return quote + strings.ReplaceAll(value, quote, "&quot;") + quote
}

// writeBlockContent writes block content to the builder, making sure it ends with a
// newline.
//
// Takes builder (*strings.Builder) which receives the formatted content.
// Takes content (string) which is the raw content to write.
func writeBlockContent(builder *strings.Builder, content string) {
	if content == "" {
		return
	}

	trimmed := strings.TrimSpace(content)
	builder.WriteString(trimmed)

	if !strings.HasSuffix(trimmed, newline) {
		builder.WriteString(newline)
	}
}
