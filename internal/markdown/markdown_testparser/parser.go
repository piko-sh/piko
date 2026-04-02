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

package markdown_testparser

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"

	"piko.sh/piko/internal/markdown/markdown_ast"
	"piko.sh/piko/internal/markdown/markdown_domain"
)

const (
	// frontmatterDelimiterLength is the byte length of the "---\n" opener.
	frontmatterDelimiterLength = 4

	// frontmatterCloserLength is the byte length of the "---" closer.
	frontmatterCloserLength = 3

	// frontmatterBodySkip is the number of bytes to skip past "\n---\n".
	frontmatterBodySkip = 5

	// maxATXHeadingLevel is the maximum permitted heading depth (h6).
	maxATXHeadingLevel = 6

	// minHorizontalRuleCharCount is the minimum repeated characters for a
	// horizontal rule.
	minHorizontalRuleCharCount = 3

	// carriageReturnNewline is the CRLF line ending sequence.
	carriageReturnNewline = "\r\n"

	// singleSpace is a single ASCII space character.
	singleSpace = " "
)

// Parser is a lightweight markdown parser that converts markdown source into
// piko-native AST. It satisfies [markdown_domain.MarkdownParserPort].
type Parser struct{}

var _ markdown_domain.MarkdownParserPort = (*Parser)(nil)

// NewParser creates a new test parser.
//
// Returns *Parser which is the new lightweight markdown parser.
func NewParser() *Parser { return &Parser{} }

// Parse parses markdown content, extracting YAML frontmatter and building
// a piko AST document.
//
// Takes content ([]byte) which is the raw markdown input.
//
// Returns *markdown_ast.Document which is the root of the parsed AST.
// Returns map[string]any which contains the frontmatter metadata.
// Returns error which is always nil for this parser.
func (*Parser) Parse(_ context.Context, content []byte) (*markdown_ast.Document, map[string]any, error) {
	body, frontmatter := extractFrontmatter(content)
	doc := markdown_ast.NewDocument()
	parseBlocks(doc, body, 0)
	return doc, frontmatter, nil
}

// extractFrontmatter splits content into the YAML frontmatter map and the
// remaining markdown body.
//
// Takes content ([]byte) which is the full markdown source.
//
// Returns body ([]byte) which is the content after the frontmatter block.
// Returns fm (map[string]any) which contains the parsed YAML metadata.
func extractFrontmatter(content []byte) (body []byte, fm map[string]any) {
	fm = make(map[string]any)
	if !bytes.HasPrefix(content, []byte("---\n")) && !bytes.HasPrefix(content, []byte("---\r\n")) {
		return content, fm
	}

	rest := content[frontmatterDelimiterLength:]
	idx := bytes.Index(rest, []byte("\n---\n"))
	if idx < 0 {
		idx = bytes.Index(rest, []byte("\n---\r\n"))
	}
	if idx < 0 {
		if bytes.HasSuffix(bytes.TrimRight(rest, carriageReturnNewline), []byte("---")) {
			fmBlock := bytes.TrimRight(rest, carriageReturnNewline)
			fmBlock = fmBlock[:len(fmBlock)-frontmatterCloserLength]
			_ = yaml.Unmarshal(fmBlock, &fm)
			stringifyTimes(fm)
			return nil, fm
		}
		return content, fm
	}

	fmBlock := rest[:idx]
	_ = yaml.Unmarshal(fmBlock, &fm)
	stringifyTimes(fm)

	bodyStart := frontmatterDelimiterLength + idx + frontmatterBodySkip
	bodyStart = min(bodyStart, len(content))
	return content[bodyStart:], fm
}

// stringifyTimes converts time.Time values in a frontmatter map to strings,
// matching the behaviour of goldmark-meta which returns dates as strings.
//
// Takes m (map[string]any) which is the frontmatter map to mutate in place.
func stringifyTimes(m map[string]any) {
	for k, v := range m {
		switch val := v.(type) {
		case time.Time:
			m[k] = val.Format("2006-01-02")
		case map[string]any:
			stringifyTimes(val)
		}
	}
}

// blockLine holds a line of source text with its byte offset in the original
// content.
type blockLine struct {
	// text is the line content including the trailing newline.
	text string

	// offset is the byte position of this line in the original content.
	offset int
}

// isBlockquoteLine reports whether trimmed begins with a blockquote marker.
//
// Takes trimmed (string) which is the line with trailing whitespace removed.
//
// Returns bool which is true when the line starts with "> " or is bare ">".
func isBlockquoteLine(trimmed string) bool {
	stripped := strings.TrimLeft(trimmed, singleSpace)
	return strings.HasPrefix(stripped, "> ") || stripped == ">"
}

// parseBlocks splits source into block-level constructs and appends them as
// children of parent.
//
// Takes parent (markdown_ast.Node) which receives the parsed blocks.
// Takes source ([]byte) which is the markdown content to parse.
// Takes baseOffset (int) which is the byte offset of source within the
// original document.
func parseBlocks(parent markdown_ast.Node, source []byte, baseOffset int) {
	lines := splitLines(source, baseOffset)
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimRight(line.text, carriageReturnNewline)

		if strings.TrimSpace(trimmed) == "" {
			i++
			continue
		}

		if isFenceLine(trimmed) {
			i = parseFencedCodeBlock(parent, lines, i, source, baseOffset)
			continue
		}

		if level, _ := atxHeading(trimmed); level > 0 {
			parseHeading(parent, line, level, source, baseOffset)
			i++
			continue
		}

		if isHorizontalRule(trimmed) {
			i++
			continue
		}

		if isBlockquoteLine(trimmed) {
			i = parseBlockquote(parent, lines, i, source, baseOffset)
			continue
		}

		if isUnorderedListItem(trimmed) {
			i = parseList(parent, lines, i, false, source, baseOffset)
			continue
		}

		if isOrderedListItem(trimmed) {
			i = parseList(parent, lines, i, true, source, baseOffset)
			continue
		}

		i = parseParagraph(parent, lines, i, source, baseOffset)
	}
}

// splitLines divides source into individual lines, preserving byte offsets.
//
// Takes source ([]byte) which is the content to split.
// Takes baseOffset (int) which is added to each line's offset.
//
// Returns []blockLine which holds each line with its absolute offset.
func splitLines(source []byte, baseOffset int) []blockLine {
	var lines []blockLine
	offset := 0
	for offset < len(source) {
		nlIdx := bytes.IndexByte(source[offset:], '\n')
		var lineEnd int
		if nlIdx < 0 {
			lineEnd = len(source)
		} else {
			lineEnd = offset + nlIdx + 1
		}
		lines = append(lines, blockLine{
			text:   string(source[offset:lineEnd]),
			offset: baseOffset + offset,
		})
		offset = lineEnd
	}
	return lines
}

// atxHeading tries to parse an ATX heading from line.
//
// Takes line (string) which is the trimmed source line.
//
// Returns level (int) which is the heading depth (1-6), or 0 if not a heading.
// Returns text (string) which is the heading content after the hashes.
func atxHeading(line string) (level int, text string) {
	trimmed := strings.TrimLeft(line, singleSpace)
	if len(trimmed) == 0 || trimmed[0] != '#' {
		return 0, ""
	}
	lvl := 0
	for lvl < len(trimmed) && trimmed[lvl] == '#' {
		lvl++
	}
	if lvl > maxATXHeadingLevel {
		return 0, ""
	}
	if lvl == len(trimmed) {
		return lvl, ""
	}
	if trimmed[lvl] != ' ' && trimmed[lvl] != '\t' {
		return 0, ""
	}
	rest := strings.TrimSpace(trimmed[lvl:])

	rest = strings.TrimRight(rest, singleSpace)
	if idx := strings.LastIndex(rest, singleSpace); idx >= 0 {
		candidate := rest[idx+1:]
		if len(candidate) > 0 && candidate == strings.Repeat("#", len(candidate)) {
			rest = strings.TrimRight(rest[:idx], singleSpace)
		}
	} else if rest == strings.Repeat("#", len(rest)) {
		rest = ""
	}
	return lvl, rest
}

// parseHeading creates a heading node from line and appends it to parent.
//
// Takes parent (markdown_ast.Node) which receives the heading.
// Takes line (blockLine) which is the source line containing the heading.
// Takes level (int) which is the heading depth (1-6).
func parseHeading(parent markdown_ast.Node, line blockLine, level int, _ []byte, _ int) {
	_, headingText := atxHeading(strings.TrimRight(line.text, carriageReturnNewline))

	h := markdown_ast.NewHeading(level)

	id := generateHeadingID(headingText)
	h.SetAttributeString("id", id)

	lineStart := line.offset
	lineEnd := line.offset + len(strings.TrimRight(line.text, carriageReturnNewline))
	h.SetLines(markdown_ast.NewSegments(markdown_ast.Segment{
		Start: lineStart + level + 1,
		Stop:  lineEnd,
	}))

	parseInlines(h, []byte(headingText), line.offset+level+1)

	parent.AppendChild(h)
}

// generateHeadingID produces a lowercase kebab-case ID from heading text,
// matching the goldmark auto-heading-id behaviour.
//
// Takes text (string) which is the heading content.
//
// Returns string which is the generated slug, or "heading" if empty.
func generateHeadingID(text string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(text) {
		if r == ' ' || r == '-' || r == '_' {
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			_, _ = b.WriteRune(r)
			prevDash = false
		}
	}
	result := strings.TrimRight(b.String(), "-")
	if result == "" {
		return "heading"
	}
	return result
}

var fencePattern = regexp.MustCompile(`^(\s{0,3})(` + "```" + `+|~~~+)(.*)$`)

// isFenceLine reports whether line is a fenced code block delimiter.
//
// Takes line (string) which is the trimmed source line.
//
// Returns bool which is true when the line matches the fence pattern.
func isFenceLine(line string) bool {
	return fencePattern.MatchString(line)
}

// isClosingFenceLine reports whether trimmed is a closing fence that matches
// the opening fence character and length.
//
// Takes trimmed (string) which is the line with trailing whitespace removed.
// Takes fenceChar (byte) which is the opening fence character.
// Takes fenceLen (int) which is the minimum fence length to match.
//
// Returns bool which is true when the line closes the fence.
func isClosingFenceLine(trimmed string, fenceChar byte, fenceLen int) bool {
	stripped := strings.TrimLeft(trimmed, singleSpace)
	if len(stripped) < fenceLen || stripped[0] != fenceChar {
		return false
	}
	for _, c := range stripped {
		if c != rune(fenceChar) {
			return false
		}
	}
	return true
}

// extractFenceLanguage returns the language token from a fence info string.
//
// Takes info (string) which is the info string after the opening fence.
//
// Returns string which is the language identifier.
func extractFenceLanguage(info string) string {
	if language, _, found := strings.Cut(info, singleSpace); found {
		return language
	}
	return info
}

// parseFencedCodeBlock parses a fenced code block starting at lines[start]
// and appends the node to parent.
//
// Takes parent (markdown_ast.Node) which receives the code block.
// Takes lines ([]blockLine) which is the full set of source lines.
// Takes start (int) which is the index of the opening fence line.
//
// Returns int which is the next line index to process.
func parseFencedCodeBlock(parent markdown_ast.Node, lines []blockLine, start int, _ []byte, _ int) int {
	m := fencePattern.FindStringSubmatch(strings.TrimRight(lines[start].text, carriageReturnNewline))
	if m == nil {
		return start + 1
	}
	fenceChar := m[2][0]
	fenceLen := len(m[2])
	info := strings.TrimSpace(m[3])

	fcb := markdown_ast.NewFencedCodeBlock()
	if info != "" {
		fcb.Info = info
		fcb.Language = extractFenceLanguage(info)
	}

	i := start + 1
	for i < len(lines) {
		trimmed := strings.TrimRight(lines[i].text, carriageReturnNewline)
		if isClosingFenceLine(trimmed, fenceChar, fenceLen) {
			i++
			break
		}
		fcb.Content = append(fcb.Content, []byte(lines[i].text))
		i++
	}

	parent.AppendChild(fcb)
	return i
}

// extractBlockquoteLine strips the blockquote marker from trimmed and
// returns the inner content.
//
// Takes trimmed (string) which is the line with trailing whitespace removed.
//
// Returns content ([]byte) which is the line without the marker.
// Returns shouldBreak (bool) which is true when the blockquote ends.
func extractBlockquoteLine(trimmed string) (content []byte, shouldBreak bool) {
	stripped := strings.TrimLeft(trimmed, singleSpace)
	if strings.HasPrefix(stripped, "> ") {
		return []byte(stripped[2:] + "\n"), false
	}
	if stripped == ">" {
		return []byte("\n"), false
	}
	if strings.TrimSpace(trimmed) == "" {
		return nil, true
	}
	return []byte(trimmed + "\n"), false
}

// parseBlockquote parses a blockquote starting at lines[start] and appends
// the node to parent.
//
// Takes parent (markdown_ast.Node) which receives the blockquote.
// Takes lines ([]blockLine) which is the full set of source lines.
// Takes start (int) which is the index of the first blockquote line.
//
// Returns int which is the next line index to process.
func parseBlockquote(parent markdown_ast.Node, lines []blockLine, start int, _ []byte, _ int) int {
	bq := markdown_ast.NewBlockquote()

	var innerLines []byte
	i := start
	for i < len(lines) {
		trimmed := strings.TrimRight(lines[i].text, carriageReturnNewline)
		content, shouldBreak := extractBlockquoteLine(trimmed)
		if shouldBreak {
			break
		}
		innerLines = append(innerLines, content...)
		i++
	}

	parseBlocks(bq, innerLines, 0)
	parent.AppendChild(bq)
	return i
}

// isUnorderedListItem reports whether line starts with an unordered list
// marker (-, *, or +).
//
// Takes line (string) which is the source line.
//
// Returns bool which is true when the line is an unordered list item.
func isUnorderedListItem(line string) bool {
	trimmed := strings.TrimLeft(line, singleSpace)
	if len(trimmed) < 2 {
		return false
	}
	return (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') && trimmed[1] == ' '
}

// isOrderedListItem reports whether line starts with an ordered list marker
// (digits followed by . or )).
//
// Takes line (string) which is the source line.
//
// Returns bool which is true when the line is an ordered list item.
func isOrderedListItem(line string) bool {
	trimmed := strings.TrimLeft(line, singleSpace)
	for i, c := range trimmed {
		if c >= '0' && c <= '9' {
			continue
		}
		if (c == '.' || c == ')') && i > 0 && i+1 < len(trimmed) && trimmed[i+1] == ' ' {
			return true
		}
		return false
	}
	return false
}

// extractListItemContent strips the list marker from trimmed and returns
// the item text.
//
// Takes trimmed (string) which is the line with trailing whitespace removed.
// Takes ordered (bool) which is true for ordered list markers.
//
// Returns string which is the item content after the marker.
// Returns bool which is true when the line matched a list item.
func extractListItemContent(trimmed string, ordered bool) (string, bool) {
	if ordered && isOrderedListItem(trimmed) {
		stripped := strings.TrimLeft(trimmed, singleSpace)
		dotIdx := strings.IndexAny(stripped, ".)")
		if dotIdx >= 0 && dotIdx+1 < len(stripped) {
			return stripped[dotIdx+2:], true
		}
		return "", true
	}
	if !ordered && isUnorderedListItem(trimmed) {
		stripped := strings.TrimLeft(trimmed, singleSpace)
		return stripped[2:], true
	}
	return "", false
}

// collectListItemContinuation gathers continuation lines for a list item
// until a blank line or new item is reached.
//
// Takes lines ([]blockLine) which is the full set of source lines.
// Takes start (int) which is the index of the first continuation line.
//
// Returns []byte which is the concatenated continuation content.
// Returns int which is the next line index to process.
func collectListItemContinuation(lines []blockLine, start int) ([]byte, int) {
	var content []byte
	i := start
	for i < len(lines) {
		nextTrimmed := strings.TrimRight(lines[i].text, carriageReturnNewline)
		if strings.TrimSpace(nextTrimmed) == "" {
			i++
			break
		}
		if isUnorderedListItem(nextTrimmed) || isOrderedListItem(nextTrimmed) {
			break
		}
		content = append(content, []byte(strings.TrimLeft(nextTrimmed, " \t")+"\n")...)
		i++
	}
	return content, i
}

// parseList parses consecutive list items starting at lines[start] and
// appends the list node to parent.
//
// Takes parent (markdown_ast.Node) which receives the list.
// Takes lines ([]blockLine) which is the full set of source lines.
// Takes start (int) which is the index of the first list item.
// Takes ordered (bool) which is true for ordered lists.
//
// Returns int which is the next line index to process.
func parseList(parent markdown_ast.Node, lines []blockLine, start int, ordered bool, _ []byte, _ int) int {
	list := markdown_ast.NewList(ordered)
	i := start
	for i < len(lines) {
		trimmed := strings.TrimRight(lines[i].text, carriageReturnNewline)
		itemContent, isItem := extractListItemContent(trimmed, ordered)
		if !isItem {
			break
		}

		li := markdown_ast.NewListItem()
		content := []byte(itemContent + "\n")
		i++

		continuation, nextIndex := collectListItemContinuation(lines, i)
		content = append(content, continuation...)
		i = nextIndex

		parseBlocks(li, content, 0)
		list.AppendChild(li)
	}
	parent.AppendChild(list)
	return i
}

// isHorizontalRule reports whether line is a horizontal rule (three or more
// -, *, or _ characters).
//
// Takes line (string) which is the source line.
//
// Returns bool which is true when the line is a horizontal rule.
func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < minHorizontalRuleCharCount {
		return false
	}
	ch := trimmed[0]
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	count := 0
	for _, c := range trimmed {
		if c == rune(ch) {
			count++
		} else if c != ' ' {
			return false
		}
	}
	return count >= minHorizontalRuleCharCount
}

// isBlockBoundary reports whether trimmed starts a new block construct,
// terminating the current paragraph.
//
// Takes trimmed (string) which is the line with trailing whitespace removed.
//
// Returns bool which is true when the line is a block boundary.
func isBlockBoundary(trimmed string) bool {
	if strings.TrimSpace(trimmed) == "" {
		return true
	}
	if isFenceLine(trimmed) {
		return true
	}
	if level, _ := atxHeading(trimmed); level > 0 {
		return true
	}
	if isHorizontalRule(trimmed) {
		return true
	}
	if isBlockquoteLine(trimmed) {
		return true
	}
	if isUnorderedListItem(trimmed) {
		return true
	}
	return isOrderedListItem(trimmed)
}

// parseParagraph collects consecutive non-block lines into a paragraph
// and appends it to parent.
//
// Takes parent (markdown_ast.Node) which receives the paragraph.
// Takes lines ([]blockLine) which is the full set of source lines.
// Takes start (int) which is the index of the first paragraph line.
//
// Returns int which is the next line index to process.
func parseParagraph(parent markdown_ast.Node, lines []blockLine, start int, _ []byte, _ int) int {
	var paraText []byte
	paraStart := lines[start].offset
	i := start
	for i < len(lines) {
		trimmed := strings.TrimRight(lines[i].text, carriageReturnNewline)
		if isBlockBoundary(trimmed) {
			break
		}
		if len(paraText) > 0 {
			paraText = append(paraText, ' ')
		}
		paraText = append(paraText, []byte(strings.TrimSpace(trimmed))...)
		i++
	}

	if len(paraText) > 0 {
		p := markdown_ast.NewParagraph()
		p.SetLines(markdown_ast.NewSegments(markdown_ast.Segment{
			Start: paraStart,
			Stop:  paraStart + len(paraText),
		}))
		parseInlines(p, paraText, paraStart)
		parent.AppendChild(p)
	}
	return i
}

// inlineParser holds the state for a single inline-level parsing pass.
type inlineParser struct {
	// parent is the node that receives parsed inline children.
	parent markdown_ast.Node

	// text is the raw inline content to parse.
	text []byte

	// baseOffset is the byte offset of text within the original document.
	baseOffset int

	// position is the current scan position within text.
	position int

	// textStart is the start of the current uncommitted text run.
	textStart int
}

// flushText emits any accumulated plain text from textStart up to end as a
// Text node.
//
// Takes end (int) which is the exclusive end position of the text run.
func (p *inlineParser) flushText(end int) {
	if end > p.textStart {
		segment := p.text[p.textStart:end]
		t := markdown_ast.NewText(segment)
		t.Segment = markdown_ast.Segment{
			Start: p.baseOffset + p.textStart,
			Stop:  p.baseOffset + end,
		}
		p.parent.AppendChild(t)
	}
}

// tryCodeSpan attempts to parse a code span at the current position.
//
// Returns bool which is true when a code span was successfully parsed.
func (p *inlineParser) tryCodeSpan() bool {
	end := findClosingBacktick(p.text, p.position)
	if end <= p.position {
		return false
	}
	p.flushText(p.position)
	cs := markdown_ast.NewCodeSpan()
	inner := p.text[p.position+1 : end]
	ct := markdown_ast.NewText(inner)
	ct.Segment = markdown_ast.Segment{
		Start: p.baseOffset + p.position + 1,
		Stop:  p.baseOffset + end,
	}
	cs.AppendChild(ct)
	p.parent.AppendChild(cs)
	p.position = end + 1
	p.textStart = p.position
	return true
}

// tryEmphasis attempts to parse emphasis or strong at the current position.
//
// Returns bool which is true when emphasis was successfully parsed.
func (p *inlineParser) tryEmphasis() bool {
	consumed := parseEmphasis(p.parent, p.text, p.position, p.baseOffset, func(end int) {
		p.flushText(end)
	})
	if consumed <= 0 {
		return false
	}
	p.position += consumed
	p.textStart = p.position
	return true
}

// tryLink attempts to parse an inline link at the current position.
//
// Returns bool which is true when a link was successfully parsed.
func (p *inlineParser) tryLink() bool {
	consumed := parseLink(p.parent, p.text, p.position, p.baseOffset, func(end int) {
		p.flushText(end)
	})
	if consumed <= 0 {
		return false
	}
	p.position += consumed
	p.textStart = p.position
	return true
}

// tryImage attempts to parse an inline image at the current position.
//
// Returns bool which is true when an image was successfully parsed.
func (p *inlineParser) tryImage() bool {
	consumed := parseImage(p.parent, p.text, p.position, p.baseOffset, func(end int) {
		p.flushText(end)
	})
	if consumed <= 0 {
		return false
	}
	p.position += consumed
	p.textStart = p.position
	return true
}

// parseInlines scans text for inline constructs (code spans, emphasis,
// links, images) and appends them as children of parent.
//
// Takes parent (markdown_ast.Node) which receives the inline nodes.
// Takes text ([]byte) which is the inline content to parse.
// Takes baseOffset (int) which is the byte offset of text within the
// original document.
func parseInlines(parent markdown_ast.Node, text []byte, baseOffset int) {
	p := &inlineParser{
		parent:     parent,
		text:       text,
		baseOffset: baseOffset,
	}

	for p.position < len(text) {
		ch := text[p.position]

		if ch == '`' && p.tryCodeSpan() {
			continue
		}

		if (ch == '*' || ch == '_') && p.position+1 < len(text) && p.tryEmphasis() {
			continue
		}

		if ch == '[' && p.tryLink() {
			continue
		}

		if ch == '!' && p.position+1 < len(text) && text[p.position+1] == '[' && p.tryImage() {
			continue
		}

		p.position++
	}
	p.flushText(len(text))
}

// findClosingBacktick returns the index of the next backtick after start.
//
// Takes text ([]byte) which is the inline content.
// Takes start (int) which is the position of the opening backtick.
//
// Returns int which is the index of the closing backtick, or -1 if not
// found.
func findClosingBacktick(text []byte, start int) int {
	for i := start + 1; i < len(text); i++ {
		if text[i] == '`' {
			return i
		}
	}
	return -1
}

// parseEmphasis parses an emphasis or strong span starting at pos and
// appends the node to parent.
//
// Takes parent (markdown_ast.Node) which receives the emphasis node.
// Takes text ([]byte) which is the inline content.
// Takes pos (int) which is the position of the opening delimiter.
// Takes baseOffset (int) which is the byte offset of text in the document.
// Takes flushBefore (func(int)) which flushes accumulated text up to an
// offset.
//
// Returns int which is the number of bytes consumed, or 0 on failure.
func parseEmphasis(parent markdown_ast.Node, text []byte, pos int, baseOffset int, flushBefore func(int)) int {
	ch := text[pos]
	level := 0
	for pos+level < len(text) && text[pos+level] == ch {
		level++
	}
	if level > 2 {
		level = 2
	}

	closer := findClosingEmphasis(text, pos+level, ch, level)
	if closer < 0 {
		return 0
	}

	flushBefore(pos)

	em := markdown_ast.NewEmphasis(level)
	inner := text[pos+level : closer]
	parseInlines(em, inner, baseOffset+pos+level)
	parent.AppendChild(em)

	return closer + level - pos
}

// findClosingEmphasis locates the closing emphasis delimiter matching the
// given character and level.
//
// Takes text ([]byte) which is the inline content.
// Takes start (int) which is the position to begin searching.
// Takes ch (byte) which is the delimiter character (* or _).
// Takes level (int) which is the number of consecutive delimiter characters.
//
// Returns int which is the index of the closing delimiter, or -1 if not
// found.
func findClosingEmphasis(text []byte, start int, ch byte, level int) int {
	needle := bytes.Repeat([]byte{ch}, level)
	idx := bytes.Index(text[start:], needle)
	if idx < 0 {
		return -1
	}
	return start + idx
}

// findMatchingCloseBracket finds the ] that matches the [ at pos, handling
// nested brackets.
//
// Takes text ([]byte) which is the inline content.
// Takes pos (int) which is the position of the opening bracket.
//
// Returns int which is the index of the matching ], or -1 if not found.
func findMatchingCloseBracket(text []byte, pos int) int {
	depth := 0
	for i := pos; i < len(text); i++ {
		if text[i] == '[' {
			depth++
		}
		if text[i] == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// parseLink parses an inline link starting at pos and appends the node to
// parent.
//
// Takes parent (markdown_ast.Node) which receives the link node.
// Takes text ([]byte) which is the inline content.
// Takes pos (int) which is the position of the opening [.
// Takes baseOffset (int) which is the byte offset of text in the document.
// Takes flushBefore (func(int)) which flushes accumulated text up to an
// offset.
//
// Returns int which is the number of bytes consumed, or 0 on failure.
func parseLink(parent markdown_ast.Node, text []byte, pos int, baseOffset int, flushBefore func(int)) int {
	closeBracket := findMatchingCloseBracket(text, pos)
	if closeBracket < 0 || closeBracket+1 >= len(text) || text[closeBracket+1] != '(' {
		return 0
	}

	closeParen := bytes.IndexByte(text[closeBracket+2:], ')')
	if closeParen < 0 {
		return 0
	}
	closeParen += closeBracket + 2

	flushBefore(pos)

	linkText := text[pos+1 : closeBracket]
	urlAndTitle := text[closeBracket+2 : closeParen]
	url, title := parseURLTitle(urlAndTitle)

	link := markdown_ast.NewLink([]byte(url), []byte(title))
	parseInlines(link, linkText, baseOffset+pos+1)
	parent.AppendChild(link)

	return closeParen + 1 - pos
}

// parseImage parses an inline image starting at pos and appends the node
// to parent.
//
// Takes parent (markdown_ast.Node) which receives the image node.
// Takes text ([]byte) which is the inline content.
// Takes pos (int) which is the position of the opening !.
// Takes baseOffset (int) which is the byte offset of text in the document.
// Takes flushBefore (func(int)) which flushes accumulated text up to an
// offset.
//
// Returns int which is the number of bytes consumed, or 0 on failure.
func parseImage(parent markdown_ast.Node, text []byte, pos int, baseOffset int, flushBefore func(int)) int {
	if pos+1 >= len(text) || text[pos+1] != '[' {
		return 0
	}

	closeBracket := bytes.IndexByte(text[pos+2:], ']')
	if closeBracket < 0 {
		return 0
	}
	closeBracket += pos + 2

	if closeBracket+1 >= len(text) || text[closeBracket+1] != '(' {
		return 0
	}

	closeParen := bytes.IndexByte(text[closeBracket+2:], ')')
	if closeParen < 0 {
		return 0
	}
	closeParen += closeBracket + 2

	flushBefore(pos)

	altText := text[pos+2 : closeBracket]
	urlAndTitle := text[closeBracket+2 : closeParen]
	url, title := parseURLTitle(urlAndTitle)

	img := markdown_ast.NewImage([]byte(url), []byte(title))

	parseInlines(img, altText, baseOffset+pos+2)
	parent.AppendChild(img)

	return closeParen + 1 - pos
}

// parseURLTitle splits raw into a URL and optional quoted title.
//
// Takes raw ([]byte) which is the content between ( and ) in a link or
// image.
//
// Returns url (string) which is the destination URL.
// Returns title (string) which is the quoted title, or empty if absent.
func parseURLTitle(raw []byte) (url, title string) {
	trimmed := bytes.TrimSpace(raw)
	s := string(trimmed)

	for _, q := range []byte{'"', '\''} {
		if lastQ := bytes.LastIndexByte(trimmed, q); lastQ > 0 {
			firstQ := bytes.IndexByte(trimmed[:lastQ], q)
			if firstQ > 0 {
				url = strings.TrimSpace(string(trimmed[:firstQ]))
				title = string(trimmed[firstQ+1 : lastQ])
				return url, title
			}
		}
	}

	return s, ""
}
