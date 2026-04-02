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

package llm_domain

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/markdown/markdown_ast"
	"piko.sh/piko/internal/markdown/markdown_domain"
)

const (
	// maxHeadingLevel is the highest heading level (h1-h6) recognised by the
	// markdown splitter.
	maxHeadingLevel = 6

	// fenceMinLength is the minimum length of a fence delimiter (``` or ~~~)
	// for code block detection.
	fenceMinLength = 3

	// metadataKeyHeading is the metadata key used to store the section heading
	// in split document chunks.
	metadataKeyHeading = "heading"

	// minMergeChunkSize is the minimum content length below which a chunk is
	// force-merged into an adjacent chunk regardless of maxSize constraints.
	minMergeChunkSize = 50
)

// MarkdownSplitterOption configures optional behaviour of [MarkdownSplitter].
type MarkdownSplitterOption func(*MarkdownSplitter)

// MarkdownSplitter splits markdown documents at heading boundaries using the
// goldmark AST parser. It implements SplitterPort and uses
// RecursiveCharacterSplitter for sections that exceed the chunk size.
type MarkdownSplitter struct {
	// parser is an optional markdown parser for AST-based heading detection.
	// When nil, a simple line-based heuristic is used instead.
	parser markdown_domain.MarkdownParserPort

	// fallback splits content that does not match markdown boundaries.
	fallback *RecursiveCharacterSplitter

	// chunkSize is the maximum size in characters for each chunk.
	chunkSize int

	// overlap is the number of characters shared between consecutive chunks.
	overlap int

	// minChunkSize is the minimum chunk size; smaller chunks get merged.
	// Zero disables merging.
	minChunkSize int

	// maxSplitLevel specifies the deepest heading level (1-6) used for splitting.
	maxSplitLevel int
}

// NewMarkdownSplitter creates a new MarkdownSplitter.
//
// Takes chunkSize (int) which specifies the maximum chunk size in bytes.
// Takes overlap (int) which specifies the character overlap for sub-split
// chunks. Must be less than chunkSize.
// Takes opts (...MarkdownSplitterOption) which provides optional settings.
// Use [WithMaxSplitLevel] to change the heading split threshold.
//
// Returns *MarkdownSplitter which implements [SplitterPort].
// Returns error when the overlap is greater than or equal to chunkSize.
func NewMarkdownSplitter(chunkSize, overlap int, opts ...MarkdownSplitterOption) (*MarkdownSplitter, error) {
	fallback, err := NewRecursiveCharacterSplitter(chunkSize, overlap)
	if err != nil {
		return nil, fmt.Errorf("creating fallback splitter: %w", err)
	}
	s := &MarkdownSplitter{
		chunkSize:     chunkSize,
		overlap:       overlap,
		maxSplitLevel: 2,
		fallback:      fallback,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// Split divides a markdown document into chunks at heading
// boundaries. Each chunk inherits the parent document's metadata
// with an additional metadataKeyHeading key set to the text of
// the heading that introduces the section.
//
// Takes document (Document) which is the markdown document to split.
//
// Returns []Document which contains the resulting chunks with unique IDs.
func (s *MarkdownSplitter) Split(document Document) []Document {
	sections := s.splitOnHeadings(document.Content, s.maxSplitLevel)

	var chunks []Document
	for _, sec := range sections {
		content := strings.TrimSpace(sec.content)
		if content == "" {
			continue
		}

		if len(content) <= s.chunkSize {
			chunks = s.appendSmallSection(chunks, document, sec, content)
		} else {
			chunks = s.splitOversizedSection(chunks, document, sec, content)
		}
	}

	if len(chunks) == 0 {
		return []Document{{
			ID:       document.ID + "-chunk-0",
			Content:  document.Content,
			Metadata: copyMetadata(document.Metadata),
		}}
	}

	if s.minChunkSize > 0 {
		chunks = mergeSmallChunks(chunks, s.minChunkSize, s.chunkSize)
		for i := range chunks {
			chunks[i].ID = document.ID + fmt.Sprintf("-chunk-%d", i)
		}
	}

	return chunks
}

// appendSmallSection appends a section that fits within the chunk size as a
// single document chunk.
//
// Takes chunks ([]Document) which is the accumulator for document chunks.
// Takes document (Document) which provides the parent document ID and metadata.
// Takes sec (section) which provides the heading text.
// Takes content (string) which is the trimmed section content.
//
// Returns []Document which contains the updated chunks slice.
func (*MarkdownSplitter) appendSmallSection(chunks []Document, document Document, sec section, content string) []Document {
	meta := metadataWithHeading(document.Metadata, sec.heading)
	return append(chunks, Document{
		ID:       document.ID + fmt.Sprintf("-chunk-%d", len(chunks)),
		Content:  content,
		Metadata: meta,
	})
}

// splitOversizedSection splits a section that exceeds the chunk size using
// code-fence-aware segmentation and fallback character splitting.
//
// Takes chunks ([]Document) which is the accumulator for document chunks.
// Takes document (Document) which provides the parent document ID and metadata.
// Takes sec (section) which provides the heading text.
// Takes content (string) which is the trimmed section content.
//
// Returns []Document which contains the updated chunks slice.
func (s *MarkdownSplitter) splitOversizedSection(chunks []Document, document Document, sec section, content string) []Document {
	segments := splitAroundCodeFences(content)
	subTexts := groupSegments(segments, s.chunkSize, s.minChunkSize)

	for _, txt := range subTexts {
		if len(txt) <= s.chunkSize || startsWithCodeFence(txt) {
			meta := metadataWithHeading(document.Metadata, sec.heading)
			chunks = append(chunks, Document{
				ID:       document.ID + fmt.Sprintf("-chunk-%d", len(chunks)),
				Content:  txt,
				Metadata: meta,
			})
		} else {
			chunks = s.fallbackSplitSubText(chunks, document, sec, txt)
		}
	}
	return chunks
}

// fallbackSplitSubText uses the recursive character splitter on a sub-text
// that exceeds the chunk size and is not a code fence.
//
// Takes chunks ([]Document) which is the accumulator for document chunks.
// Takes document (Document) which provides the parent document ID and metadata.
// Takes sec (section) which provides the heading text.
// Takes txt (string) which is the text to split.
//
// Returns []Document which contains the updated chunks slice.
func (s *MarkdownSplitter) fallbackSplitSubText(chunks []Document, document Document, sec section, txt string) []Document {
	subDoc := Document{
		ID:       document.ID,
		Content:  txt,
		Metadata: document.Metadata,
	}
	for _, sc := range s.fallback.Split(subDoc) {
		sc.Metadata = metadataWithHeading(sc.Metadata, sec.heading)
		sc.ID = document.ID + fmt.Sprintf("-chunk-%d", len(chunks))
		chunks = append(chunks, sc)
	}
	return chunks
}

// section represents a heading-delimited section of a markdown document.
type section struct {
	// heading is the markdown heading text for this section.
	heading string

	// content holds the text content of this markdown section.
	content string
}

// headingPos records a heading's location and text within the source.
type headingPos struct {
	// heading is the text content of the markdown heading.
	heading string

	// offset is the byte position of this heading in the source content.
	offset int
}

// segment represents a contiguous piece of a markdown section, tagged as either
// prose or a fenced code block.
type segment struct {
	// text holds the raw text content of the segment.
	text string

	// isCode indicates whether this segment is a code block.
	isCode bool
}

// WithMaxSplitLevel sets the maximum heading level that acts as a split
// boundary.
//
// For example, level 3 splits on h1, h2, and h3 but groups h4-h6 content
// with their parent section. The default is 2.
//
// Takes level (int) which specifies the maximum heading level (1-6).
//
// Returns MarkdownSplitterOption which applies the setting.
func WithMaxSplitLevel(level int) MarkdownSplitterOption {
	return func(s *MarkdownSplitter) {
		if level >= 1 && level <= maxHeadingLevel {
			s.maxSplitLevel = level
		}
	}
}

// WithSplitterMarkdownParser sets the markdown parser for AST-based heading
// detection. When nil (default), the splitter returns the entire document as
// a single section - users must provide a parser for heading-based splitting.
//
// Takes parser (markdown_domain.MarkdownParserPort) which parses markdown into
// piko AST.
//
// Returns MarkdownSplitterOption which applies the setting.
func WithSplitterMarkdownParser(parser markdown_domain.MarkdownParserPort) MarkdownSplitterOption {
	return func(s *MarkdownSplitter) {
		s.parser = parser
	}
}

// WithMinChunkSize sets the minimum chunk size in bytes.
//
// When chunks are smaller than this threshold, they are merged with an
// adjacent chunk. The merge prefers forward merge into the next chunk, and
// falls back to backward merge into the previous chunk. This eliminates
// orphaned intro text, tables without context, and other undersized fragments
// that are useless for vector search.
//
// When the value is greater than 0, horizontal-rule-only chunks are filtered
// entirely.
//
// When the value is 0 (the default), the merge pass is disabled for backward
// compatibility.
//
// Takes size (int) which specifies the minimum chunk size in bytes.
//
// Returns MarkdownSplitterOption which applies the setting.
func WithMinChunkSize(size int) MarkdownSplitterOption {
	return func(s *MarkdownSplitter) {
		if size >= 0 {
			s.minChunkSize = size
		}
	}
}

// metadataWithHeading copies the source metadata and sets the heading key
// when the heading is non-empty.
//
// Takes src (map[string]any) which is the metadata to copy.
// Takes heading (string) which is the heading text to set.
//
// Returns map[string]any which is the new metadata map.
func metadataWithHeading(src map[string]any, heading string) map[string]any {
	meta := copyMetadata(src)
	if meta == nil {
		meta = make(map[string]any)
	}
	if heading != "" {
		meta[metadataKeyHeading] = heading
	}
	return meta
}

// splitOnHeadings splits markdown content into sections at heading boundaries.
//
// Headings with level <= maxSplitLevel act as split boundaries. Text before the
// first qualifying heading becomes a section with an empty heading. Heading
// identification uses the goldmark parser so that headings inside fenced code
// blocks, HTML blocks, and other constructs are correctly ignored.
//
// Takes content (string) which is the markdown text to split.
// Takes maxSplitLevel (int) which sets the maximum heading level to act as a
// split boundary.
//
// Returns []section which contains the resulting sections, each with a heading
// and its content.
func (s *MarkdownSplitter) splitOnHeadings(content string, maxSplitLevel int) []section {
	source := []byte(content)

	if s.parser == nil {
		return []section{{heading: "", content: content}}
	}

	doc, _, err := s.parser.Parse(context.Background(), source)
	if err != nil || doc == nil {
		return []section{{heading: "", content: content}}
	}

	var headings []headingPos
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		h, ok := child.(*markdown_ast.Heading)
		if !ok || h.Level > maxSplitLevel {
			continue
		}

		headings = append(headings, headingPos{
			offset:  headingLineStart(source, h),
			heading: extractNodeText(h),
		})
	}

	if len(headings) == 0 {
		return []section{{heading: "", content: content}}
	}

	var sections []section

	if headings[0].offset > 0 {
		sections = append(sections, section{
			heading: "",
			content: string(source[:headings[0].offset]),
		})
	}

	for i, hp := range headings {
		var end int
		if i+1 < len(headings) {
			end = headings[i+1].offset
		} else {
			end = len(source)
		}

		body := stripHeadingLine(source[hp.offset:end])
		sections = append(sections, section{
			heading: hp.heading,
			content: string(body),
		})
	}

	return sections
}

// headingLineStart returns the byte offset of the start of the line containing
// the heading node. It walks backward from the heading's content position to
// find the beginning of the line, including the # prefix.
//
// Takes source ([]byte) which is the document source text.
// Takes h (*ast.Heading) which is the heading node to locate.
//
// Returns int which is the byte offset of the line start, or 0 if not found.
func headingLineStart(source []byte, h *markdown_ast.Heading) int {
	if h.Lines().Len() > 0 {
		start := h.Lines().At(0).Start
		for start > 0 && source[start-1] != '\n' {
			start--
		}
		return start
	}

	if h.HasChildren() {
		if t, ok := h.FirstChild().(*markdown_ast.Text); ok {
			start := t.Segment.Start
			for start > 0 && source[start-1] != '\n' {
				start--
			}
			return start
		}
	}

	return 0
}

// stripHeadingLine removes the first line (the heading syntax) from a section's
// bytes, returning the remaining body content.
//
// Takes section ([]byte) which contains the section bytes including the heading.
//
// Returns []byte which is the content after the first line, or nil if no
// newline is found.
func stripHeadingLine(section []byte) []byte {
	_, after, found := bytes.Cut(section, []byte{'\n'})
	if !found {
		return nil
	}
	return after
}

// splitAroundCodeFences divides content into alternating prose and code-fence
// segments.
//
// Code fences (``` or ~~~) are kept intact so they are never broken across
// chunks. Concatenating the segment texts reproduces the original content.
//
// Takes content (string) which is the text to split around code fences.
//
// Returns []segment which contains segments in document order, alternating
// between prose and code sections.
func splitAroundCodeFences(content string) []segment {
	var segments []segment
	lines := strings.Split(content, "\n")

	var prose strings.Builder
	var code strings.Builder
	inFence := false
	fencePrefix := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		isLast := i == len(lines)-1

		if !inFence {
			segments, inFence, fencePrefix = handleProseLineInFenceSplit(
				trimmed, line, isLast, &prose, &code, segments)
		} else {
			inFence = handleCodeLineInFenceSplit(
				trimmed, line, isLast, fencePrefix, &code, &segments)
		}
	}

	if code.Len() > 0 {
		segments = append(segments, segment{text: code.String(), isCode: true})
	}
	if prose.Len() > 0 {
		segments = append(segments, segment{text: prose.String(), isCode: false})
	}

	return segments
}

// handleProseLineInFenceSplit processes a line when not inside a code fence.
// If the line opens a code fence, it flushes accumulated prose and begins
// code accumulation.
//
// Takes trimmed (string) which is the whitespace-trimmed line.
// Takes line (string) which is the original line text.
// Takes isLast (bool) which indicates whether this is the final line.
// Takes prose (*strings.Builder) which accumulates prose content.
// Takes code (*strings.Builder) which accumulates code content.
// Takes segments ([]segment) which is the current segment list.
//
// Returns []segment which is the updated segment list.
// Returns bool which indicates whether we are now inside a code fence.
// Returns string which is the fence prefix if a fence was opened.
func handleProseLineInFenceSplit(
	trimmed, line string, isLast bool,
	prose, code *strings.Builder, segments []segment,
) ([]segment, bool, string) {
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		if prose.Len() > 0 {
			segments = append(segments, segment{text: prose.String(), isCode: false})
			prose.Reset()
		}
		code.WriteString(line)
		if !isLast {
			code.WriteByte('\n')
		}
		return segments, true, trimmed[:fenceMinLength]
	}

	prose.WriteString(line)
	if !isLast {
		prose.WriteByte('\n')
	}
	return segments, false, ""
}

// handleCodeLineInFenceSplit processes a line when inside a code fence.
// If the line closes the fence, it flushes the code segment.
//
// Takes trimmed (string) which is the whitespace-trimmed line.
// Takes line (string) which is the original line text.
// Takes isLast (bool) which indicates whether this is the final line.
// Takes fencePrefix (string) which is the opening fence marker.
// Takes code (*strings.Builder) which accumulates code content.
// Takes segments (*[]segment) which is the current segment list.
//
// Returns bool which indicates whether we are still inside a code fence.
func handleCodeLineInFenceSplit(
	trimmed, line string, isLast bool,
	fencePrefix string, code *strings.Builder, segments *[]segment,
) bool {
	code.WriteString(line)
	if !isLast {
		code.WriteByte('\n')
	}
	if strings.HasPrefix(trimmed, fencePrefix) && strings.TrimSpace(strings.TrimLeft(trimmed, fencePrefix[:1])) == "" {
		*segments = append(*segments, segment{text: code.String(), isCode: true})
		code.Reset()
		return false
	}
	return true
}

// groupSegments merges consecutive segments into chunks within the given size
// limit.
//
// When a single code block exceeds chunkSize, it becomes its own chunk rather
// than being split. When minSegmentSize is positive, small prose segments stay
// attached to the following code block.
//
// Takes segments ([]segment) which contains the text segments to group.
// Takes chunkSize (int) which specifies the maximum size for each chunk.
// Takes minSegmentSize (int) which defines when small prose attaches to code.
//
// Returns []string which contains the grouped chunks of text.
func groupSegments(segments []segment, chunkSize, minSegmentSize int) []string {
	if len(segments) == 0 {
		return nil
	}

	var result []string
	var current strings.Builder

	for _, seg := range segments {
		segText := strings.TrimSpace(seg.text)
		if segText == "" {
			continue
		}

		if seg.isCode && len(segText) > chunkSize {
			result = flushOversizedCode(&current, segText, result, minSegmentSize)
			continue
		}

		result = appendSegmentText(&current, segText, seg.isCode, result, chunkSize, minSegmentSize)
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result
}

// flushOversizedCode handles a code segment that exceeds the chunk size by
// flushing or merging the current prose buffer, then emitting the code block.
//
// Takes current (*strings.Builder) which holds accumulated text.
// Takes segText (string) which is the oversized code block text.
// Takes result ([]string) which is the accumulated chunk list.
// Takes minSegmentSize (int) which is the minimum segment size for merging.
//
// Returns []string which is the updated chunk list.
func flushOversizedCode(current *strings.Builder, segText string, result []string, minSegmentSize int) []string {
	if current.Len() == 0 {
		return append(result, segText)
	}

	if minSegmentSize > 0 && current.Len() < minSegmentSize {
		current.WriteString("\n\n")
		current.WriteString(segText)
		result = append(result, strings.TrimSpace(current.String()))
		current.Reset()
		return result
	}

	result = append(result, strings.TrimSpace(current.String()))
	current.Reset()
	return append(result, segText)
}

// appendSegmentText adds a normal-sized segment to the current builder,
// flushing it first when the combined size would exceed the chunk limit.
//
// Takes current (*strings.Builder) which holds accumulated text.
// Takes segText (string) which is the segment text to append.
// Takes isCode (bool) which indicates whether the segment is a code block.
// Takes result ([]string) which is the accumulated chunk list.
// Takes chunkSize (int) which is the maximum chunk size.
// Takes minSegmentSize (int) which is the minimum segment size for merging.
//
// Returns []string which is the updated chunk list.
func appendSegmentText(current *strings.Builder, segText string, isCode bool, result []string, chunkSize, minSegmentSize int) []string {
	needed := len(segText)
	if current.Len() > 0 {
		needed += 2
	}

	if current.Len()+needed > chunkSize && current.Len() > 0 {
		if minSegmentSize <= 0 || isCode || current.Len() >= minSegmentSize {
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}

	if current.Len() > 0 {
		current.WriteString("\n\n")
	}
	current.WriteString(segText)
	return result
}

// startsWithCodeFence reports whether content begins with a fenced code block
// marker (``` or ~~~). Used to detect oversized code blocks that should be
// kept intact rather than character-split.
//
// Takes bodyText (string) which is the text to check for a code fence prefix.
//
// Returns bool which is true if the text starts with ``` or ~~~.
func startsWithCodeFence(bodyText string) bool {
	trimmed := strings.TrimSpace(bodyText)
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

// isJunkContent reports whether content is structurally empty and should be
// filtered from the chunk output. Horizontal rules, whitespace-only content,
// and other purely decorative elements contribute nothing to vector search.
//
// Takes content (string) which is the text to check for junk patterns.
//
// Returns bool which is true when the content is junk and should be filtered.
func isJunkContent(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return true
	}
	stripped := strings.Map(func(r rune) rune {
		if r == '-' || r == '*' || r == '_' {
			return -1
		}
		return r
	}, trimmed)
	return strings.TrimSpace(stripped) == "" && len(trimmed) >= fenceMinLength
}

// mergeSmallChunks merges undersized chunks with their neighbours to eliminate
// orphaned intro text, context-free tables, and other fragments that are too
// small for useful vector search.
//
// Takes chunks ([]Document) which contains the document chunks to process.
// Takes minSize (int) which specifies the minimum acceptable chunk size.
// Takes maxSize (int) which specifies the maximum allowed chunk size.
//
// Returns []Document which contains the merged chunks with junk filtered out.
//
// The algorithm:
//
//  1. Filter junk content (horizontal rules, whitespace).
//  2. Forward merge: if a chunk is smaller than minSize and a next chunk exists,
//     merge into the next chunk (prepend). Very small chunks (< 50 bytes) merge
//     even if the combined size exceeds maxSize.
//  3. Backward merge: if a chunk is still smaller than minSize and a previous
//     chunk exists and the combined size fits within maxSize, merge into the
//     previous chunk (append).
func mergeSmallChunks(chunks []Document, minSize, maxSize int) []Document {
	filtered := filterJunkChunks(chunks)
	if len(filtered) == 0 {
		return filtered
	}

	merged := make([]bool, len(filtered))
	forwardMergePass(filtered, merged, minSize, maxSize)
	backwardMergePass(filtered, merged, minSize, maxSize)

	result := make([]Document, 0, len(filtered))
	for i, c := range filtered {
		if !merged[i] {
			result = append(result, c)
		}
	}
	return result
}

// filterJunkChunks removes chunks whose content is structurally empty.
//
// Takes chunks ([]Document) which contains the chunks to filter.
//
// Returns []Document which contains only non-junk chunks.
func filterJunkChunks(chunks []Document) []Document {
	filtered := make([]Document, 0, len(chunks))
	for _, c := range chunks {
		if !isJunkContent(c.Content) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// forwardMergePass merges undersized chunks into the next unmerged chunk.
//
// Takes filtered ([]Document) which contains the chunks to process.
// Takes merged ([]bool) which tracks which chunks have been merged away.
// Takes minSize (int) which is the minimum acceptable chunk size.
// Takes maxSize (int) which is the maximum allowed chunk size.
func forwardMergePass(filtered []Document, merged []bool, minSize, maxSize int) {
	for i := range len(filtered) - 1 {
		if merged[i] || len(filtered[i].Content) >= minSize {
			continue
		}

		next := findNextUnmerged(merged, i+1, len(filtered))
		if next < 0 {
			break
		}

		combined := len(filtered[i].Content) + 2 + len(filtered[next].Content)
		if combined <= maxSize || len(filtered[i].Content) < minMergeChunkSize {
			mergeChunkInto(&filtered[next], &filtered[i], true)
			merged[i] = true
		}
	}
}

// backwardMergePass merges undersized chunks into the previous unmerged chunk.
//
// Takes filtered ([]Document) which contains the chunks to process.
// Takes merged ([]bool) which tracks which chunks have been merged away.
// Takes minSize (int) which is the minimum acceptable chunk size.
// Takes maxSize (int) which is the maximum allowed chunk size.
func backwardMergePass(filtered []Document, merged []bool, minSize, maxSize int) {
	for i := len(filtered) - 1; i > 0; i-- {
		if merged[i] || len(filtered[i].Content) >= minSize {
			continue
		}

		previous := findPrevUnmerged(merged, i-1)
		if previous < 0 {
			continue
		}

		combined := len(filtered[previous].Content) + 2 + len(filtered[i].Content)
		if combined <= maxSize || len(filtered[i].Content) < minMergeChunkSize {
			mergeChunkInto(&filtered[previous], &filtered[i], false)
			merged[i] = true
		}
	}
}

// findNextUnmerged returns the index of the next unmerged chunk starting from
// start, or -1 if none exists.
//
// Takes merged ([]bool) which tracks which chunks have been merged.
// Takes start (int) which is the first index to check.
// Takes limit (int) which is the exclusive upper bound.
//
// Returns int which is the index of the next unmerged chunk, or -1.
func findNextUnmerged(merged []bool, start, limit int) int {
	for j := start; j < limit; j++ {
		if !merged[j] {
			return j
		}
	}
	return -1
}

// findPrevUnmerged returns the index of the previous unmerged chunk starting
// from start scanning backwards, or -1 if none exists.
//
// Takes merged ([]bool) which tracks which chunks have been merged.
// Takes start (int) which is the first index to check.
//
// Returns int which is the index of the previous unmerged chunk, or -1.
func findPrevUnmerged(merged []bool, start int) int {
	for j := start; j >= 0; j-- {
		if !merged[j] {
			return j
		}
	}
	return -1
}

// mergeChunkInto merges the content of src into dst and propagates the heading
// metadata when the destination lacks one.
//
// Takes dst (*Document) which receives the merged content.
// Takes src (*Document) which provides the content to merge.
// Takes prepend (bool) which when true places src before dst; otherwise appends.
func mergeChunkInto(dst, src *Document, prepend bool) {
	if prepend {
		dst.Content = src.Content + "\n\n" + dst.Content
	} else {
		dst.Content = dst.Content + "\n\n" + src.Content
	}
	if h, ok := src.Metadata[metadataKeyHeading].(string); ok && h != "" {
		if existing, ok2 := dst.Metadata[metadataKeyHeading].(string); !ok2 || existing == "" {
			if dst.Metadata == nil {
				dst.Metadata = make(map[string]any)
			}
			dst.Metadata[metadataKeyHeading] = h
		}
	}
}

// extractNodeText extracts the text content from a piko markdown AST node by
// recursively walking its inline children.
//
// Takes node (markdown_ast.Node) which is the AST node to extract text from.
//
// Returns string which is the combined text content of the node's children.
func extractNodeText(node markdown_ast.Node) string {
	var result strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*markdown_ast.Text); ok {
			result.Write(textNode.Value)
		} else {
			result.WriteString(extractNodeText(child))
		}
	}
	return result.String()
}
