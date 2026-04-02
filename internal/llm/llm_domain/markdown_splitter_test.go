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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/markdown/markdown_ast"
	"piko.sh/piko/internal/markdown/markdown_testparser"
)

func TestMarkdownSplitter_SplitsOnHeadings(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# Introduction\n\nSome intro text.\n\n## Getting Started\n\nSetup instructions here.",
		Metadata: map[string]any{
			"source": "guide.md",
		},
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 2)

	assert.Equal(t, "Some intro text.", chunks[0].Content)
	assert.Equal(t, "Introduction", chunks[0].Metadata["heading"])
	assert.Equal(t, "guide.md", chunks[0].Metadata["source"])

	assert.Equal(t, "Setup instructions here.", chunks[1].Content)
	assert.Equal(t, "Getting Started", chunks[1].Metadata["heading"])
	assert.Equal(t, "guide.md", chunks[1].Metadata["source"])
}

func TestMarkdownSplitter_TextBeforeFirstHeading(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "Preamble text.\n\n# First Heading\n\nContent.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 2)

	assert.Equal(t, "Preamble text.", chunks[0].Content)
	assert.Empty(t, chunks[0].Metadata["heading"], "preamble should have no heading")

	assert.Equal(t, "Content.", chunks[1].Content)
	assert.Equal(t, "First Heading", chunks[1].Metadata["heading"])
}

func TestMarkdownSplitter_OversizedSectionFallsBack(t *testing.T) {
	long := strings.Repeat("word ", 100)
	document := Document{
		ID:      "doc",
		Content: "# Big Section\n\n" + long,
	}

	splitter, err := NewMarkdownSplitter(100, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	assert.Greater(t, len(chunks), 1, "oversized section should be sub-split")

	for _, chunk := range chunks {
		assert.Equal(t, "Big Section", chunk.Metadata["heading"],
			"all sub-chunks should inherit the heading")
		assert.LessOrEqual(t, len(chunk.Content), 110,
			"sub-chunks should respect chunk size (with some tolerance)")
	}
}

func TestMarkdownSplitter_SequentialChunkIDs(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# A\n\nFirst.\n\n# B\n\nSecond.\n\n# C\n\nThird.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 3)
	assert.Equal(t, "doc-chunk-0", chunks[0].ID)
	assert.Equal(t, "doc-chunk-1", chunks[1].ID)
	assert.Equal(t, "doc-chunk-2", chunks[2].ID)
}

func TestMarkdownSplitter_EmptyContent(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 1, "empty doc returns single chunk")
	assert.Equal(t, "", chunks[0].Content)
}

func TestMarkdownSplitter_NoHeadings(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "Just a plain paragraph.\n\nAnother paragraph.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 1)
	assert.Equal(t, "Just a plain paragraph.\n\nAnother paragraph.", chunks[0].Content)
}

func TestMarkdownSplitter_MultiLevelHeadings(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# Top Level\n\nIntro.\n\n## Sub Section\n\nSub content.\n\n### Deep\n\nDeep content.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 2)
	assert.Equal(t, "Top Level", chunks[0].Metadata["heading"])
	assert.Equal(t, "Intro.", chunks[0].Content)
	assert.Equal(t, "Sub Section", chunks[1].Metadata["heading"])
	assert.Contains(t, chunks[1].Content, "Sub content.")
	assert.Contains(t, chunks[1].Content, "### Deep")
	assert.Contains(t, chunks[1].Content, "Deep content.")
}

func TestMarkdownSplitter_EmptySectionsSkipped(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# A\n\nContent A.\n\n# B\n\n# C\n\nContent C.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 2)
	assert.Equal(t, "A", chunks[0].Metadata["heading"])
	assert.Equal(t, "C", chunks[1].Metadata["heading"])
}

func TestMarkdownSplitter_MetadataIsolation(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# A\n\nFirst.\n\n# B\n\nSecond.",
		Metadata: map[string]any{
			"source": "file.md",
		},
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 2)

	chunks[0].Metadata["extra"] = "test"
	assert.Nil(t, chunks[1].Metadata["extra"])
}

func TestMarkdownSplitter_NilMetadata(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# Hello\n\nContent.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 1)
	assert.Equal(t, "Hello", chunks[0].Metadata["heading"])
}

func TestMarkdownSplitter_ImplementsSplitterPort(t *testing.T) {
	var _ SplitterPort = (*MarkdownSplitter)(nil)
}

func TestMarkdownSplitter_WithMaxSplitLevel(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# Top\n\nIntro.\n\n## Mid\n\nMid content.\n\n### Deep\n\nDeep content.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()), WithMaxSplitLevel(3))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 3)
	assert.Equal(t, "Top", chunks[0].Metadata["heading"])
	assert.Equal(t, "Mid", chunks[1].Metadata["heading"])
	assert.Equal(t, "Deep", chunks[2].Metadata["heading"])
}

func TestMarkdownSplitter_WithMaxSplitLevel6(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# A\n\nA body.\n\n#### D\n\nD body.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()), WithMaxSplitLevel(6))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 2)
	assert.Equal(t, "A", chunks[0].Metadata["heading"])
	assert.Equal(t, "D", chunks[1].Metadata["heading"])
}

func TestMarkdownSplitter_CodeBlockHeadingNotSplit(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# Real Heading\n\nText before code.\n\n```bash\n# This is a comment\necho hello\n```\n\nText after code.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()), WithMaxSplitLevel(6))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 1, "code block comment should not create a split")
	assert.Equal(t, "Real Heading", chunks[0].Metadata["heading"])
	assert.Contains(t, chunks[0].Content, "# This is a comment")
	assert.Contains(t, chunks[0].Content, "echo hello")
}

func TestMarkdownSplitter_DefaultMaxSplitLevel(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# H1\n\nH1 body.\n\n### H3\n\nH3 body.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	require.Len(t, chunks, 1, "h3 should not split with default maxSplitLevel=2")
	assert.Equal(t, "H1", chunks[0].Metadata["heading"])
	assert.Contains(t, chunks[0].Content, "### H3")
}

func TestSplitOnHeadings(t *testing.T) {
	content := "Preamble.\n\n# First\n\nContent 1.\n\n## Second\n\nContent 2."
	s := &MarkdownSplitter{
		parser:        markdown_testparser.NewParser(),
		maxSplitLevel: 2,
	}
	sections := s.splitOnHeadings(content, 2)

	require.Len(t, sections, 3)
	assert.Empty(t, sections[0].heading)
	assert.Contains(t, sections[0].content, "Preamble.")
	assert.Equal(t, "First", sections[1].heading)
	assert.Contains(t, sections[1].content, "Content 1.")
	assert.Equal(t, "Second", sections[2].heading)
	assert.Contains(t, sections[2].content, "Content 2.")
}

func TestExtractNodeText(t *testing.T) {
	heading := markdown_ast.NewHeading(1)
	heading.AppendChild(markdown_ast.NewText([]byte("Hello World")))

	got := extractNodeText(heading)
	assert.Equal(t, "Hello World", got)
}

func TestExtractNodeText_WithInlineFormatting(t *testing.T) {
	heading := markdown_ast.NewHeading(1)
	heading.AppendChild(markdown_ast.NewText([]byte("Hello ")))
	strong := markdown_ast.NewEmphasis(2)
	strong.AppendChild(markdown_ast.NewText([]byte("World")))
	heading.AppendChild(strong)

	got := extractNodeText(heading)
	assert.Equal(t, "Hello World", got)
}

func TestHeadingLineStart(t *testing.T) {
	source := []byte("preamble\n\n## Section\n\nbody")

	heading := markdown_ast.NewHeading(2)
	heading.SetLines(markdown_ast.NewSegments(markdown_ast.Segment{Start: 13, Stop: 20}))

	offset := headingLineStart(source, heading)
	assert.Equal(t, 10, offset, "heading should start at byte offset 10 (after 'preamble\\n\\n')")
}

func TestSplitAroundCodeFences_Basic(t *testing.T) {
	content := "Some prose.\n\n```go\nfunc main() {}\n```\n\nMore prose."
	segments := splitAroundCodeFences(content)

	require.Len(t, segments, 3)
	assert.False(t, segments[0].isCode)
	assert.Contains(t, segments[0].text, "Some prose.")
	assert.True(t, segments[1].isCode)
	assert.Contains(t, segments[1].text, "func main() {}")
	assert.False(t, segments[2].isCode)
	assert.Contains(t, segments[2].text, "More prose.")
}

func TestSplitAroundCodeFences_MultipleFences(t *testing.T) {
	content := "Intro.\n\n```\ncode1\n```\n\nMiddle.\n\n```\ncode2\n```\n\nEnd."
	segments := splitAroundCodeFences(content)

	var codeCount int
	for _, seg := range segments {
		if seg.isCode {
			codeCount++
		}
	}
	assert.Equal(t, 2, codeCount)
}

func TestSplitAroundCodeFences_UnclosedFence(t *testing.T) {
	content := "Prose.\n\n```\nunclosed code block\nmore code"
	segments := splitAroundCodeFences(content)

	var hasCode bool
	for _, seg := range segments {
		if seg.isCode {
			hasCode = true
			assert.Contains(t, seg.text, "unclosed code block")
		}
	}
	assert.True(t, hasCode)
}

func TestSplitAroundCodeFences_NoFences(t *testing.T) {
	content := "Just some plain text.\n\nWith paragraphs."
	segments := splitAroundCodeFences(content)

	require.Len(t, segments, 1)
	assert.False(t, segments[0].isCode)
	assert.Equal(t, content, segments[0].text)
}

func TestSplitAroundCodeFences_TildeFences(t *testing.T) {
	content := "Prose.\n\n~~~bash\necho hello\n~~~\n\nMore."
	segments := splitAroundCodeFences(content)

	var codeSegment segment
	for _, seg := range segments {
		if seg.isCode {
			codeSegment = seg
			break
		}
	}
	assert.Contains(t, codeSegment.text, "echo hello")
}

func TestGroupSegments_FitsInOne(t *testing.T) {
	segments := []segment{
		{text: "short prose", isCode: false},
		{text: "```\ncode\n```", isCode: true},
	}

	result := groupSegments(segments, 1000, 0)
	require.Len(t, result, 1)
	assert.Contains(t, result[0], "short prose")
	assert.Contains(t, result[0], "code")
}

func TestGroupSegments_SplitsWhenOversized(t *testing.T) {
	segments := []segment{
		{text: strings.Repeat("a", 50), isCode: false},
		{text: "```\n" + strings.Repeat("b", 50) + "\n```", isCode: true},
	}

	result := groupSegments(segments, 60, 0)
	require.Len(t, result, 2, "prose and code should be separate chunks")
}

func TestGroupSegments_OversizedCodeBlockStaysIntact(t *testing.T) {
	bigCode := "```\n" + strings.Repeat("x", 200) + "\n```"
	segments := []segment{
		{text: "intro", isCode: false},
		{text: bigCode, isCode: true},
		{text: "outro", isCode: false},
	}

	result := groupSegments(segments, 100, 0)

	var foundBigBlock bool
	for _, chunk := range result {
		if strings.Contains(chunk, strings.Repeat("x", 200)) {
			foundBigBlock = true
		}
	}
	assert.True(t, foundBigBlock, "oversized code block should be kept intact")
}

func TestMarkdownSplitter_PreservesCodeBlocks(t *testing.T) {

	code := "```\n" + strings.Repeat("line\n", 20) + "```"
	document := Document{
		ID:      "doc",
		Content: "# Section\n\nSome introductory text.\n\n" + code + "\n\nFollowing paragraph.",
	}

	splitter, err := NewMarkdownSplitter(100, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	for _, chunk := range chunks {
		openCount := strings.Count(chunk.Content, "```")

		assert.True(t, openCount == 0 || openCount == 2,
			"chunk should contain complete code fences, got %d backtick-triplets in:\n%s",
			openCount, chunk.Content)
	}
}

func TestMarkdownSplitter_ASCIIDiagramKeptIntact(t *testing.T) {

	diagram := "```\n" +
		"┌─────────────────────────────────────────────────────────────┐\n" +
		"│                                                              │\n" +
		"│  ┌──────────────┐     ┌──────────────────────────────────┐  │\n" +
		"│  │  Plaintext    │     │  Encrypted Data Key + Data       │  │\n" +
		"│  └──────────────┘     └──────────────────────────────────┘  │\n" +
		"│                                                              │\n" +
		"└─────────────────────────────────────────────────────────────┘\n" +
		"```"
	document := Document{
		ID:      "crypto",
		Content: "# Encryption\n\nThe following diagram shows the flow:\n\n" + diagram,
	}

	splitter, err := NewMarkdownSplitter(200, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	var diagramChunk string
	for _, chunk := range chunks {
		if strings.Contains(chunk.Content, "Plaintext") {
			diagramChunk = chunk.Content
			break
		}
	}
	require.NotEmpty(t, diagramChunk, "diagram should appear in a chunk")
	assert.Contains(t, diagramChunk, "┌─────")
	assert.Contains(t, diagramChunk, "└─────")
}

func TestIsJunkContent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "empty", in: "", want: true},
		{name: "whitespace", in: "   \n  \t  ", want: true},
		{name: "horizontal rule dashes", in: "---", want: true},
		{name: "horizontal rule stars", in: "***", want: true},
		{name: "horizontal rule underscores", in: "___", want: true},
		{name: "long dashes", in: "----------", want: true},
		{name: "spaced dashes", in: " --- ", want: true},
		{name: "real content", in: "This is actual content", want: false},
		{name: "code fence", in: "```\ncode\n```", want: false},
		{name: "heading", in: "# Title", want: false},
		{name: "short text", in: "ok", want: false},
		{name: "dash in content", in: "use --- for rules", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isJunkContent(tt.in))
		})
	}
}

func TestMergeSmallChunks_ForwardMerge(t *testing.T) {

	chunks := []Document{
		{ID: "d-chunk-0", Content: "Here's the code:", Metadata: map[string]any{"heading": "Examples"}},
		{ID: "d-chunk-1", Content: "```go\nfunc main() { fmt.Println(\"hello\") }\n```", Metadata: map[string]any{"heading": "Examples"}},
	}

	result := mergeSmallChunks(chunks, 100, 2000)

	require.Len(t, result, 1)
	assert.Contains(t, result[0].Content, "Here's the code:")
	assert.Contains(t, result[0].Content, "func main()")
}

func TestMergeSmallChunks_BackwardMerge(t *testing.T) {

	chunks := []Document{
		{ID: "d-chunk-0", Content: strings.Repeat("a", 150), Metadata: map[string]any{"heading": "Main"}},
		{ID: "d-chunk-1", Content: "See also: other page.", Metadata: map[string]any{"heading": "Main"}},
	}

	result := mergeSmallChunks(chunks, 100, 2000)

	require.Len(t, result, 1)
	assert.Contains(t, result[0].Content, strings.Repeat("a", 150))
	assert.Contains(t, result[0].Content, "See also: other page.")
}

func TestMergeSmallChunks_JunkFiltered(t *testing.T) {
	chunks := []Document{
		{ID: "d-chunk-0", Content: "Real content here.", Metadata: map[string]any{}},
		{ID: "d-chunk-1", Content: "---", Metadata: map[string]any{}},
		{ID: "d-chunk-2", Content: "More real content.", Metadata: map[string]any{}},
	}

	result := mergeSmallChunks(chunks, 100, 2000)

	for _, c := range result {
		trimmed := strings.TrimSpace(c.Content)
		assert.NotEqual(t, "---", trimmed, "horizontal rule should be filtered")
	}
}

func TestMergeSmallChunks_NoMergeIfBothLarge(t *testing.T) {
	chunks := []Document{
		{ID: "d-chunk-0", Content: strings.Repeat("a", 200), Metadata: map[string]any{}},
		{ID: "d-chunk-1", Content: strings.Repeat("b", 200), Metadata: map[string]any{}},
	}

	result := mergeSmallChunks(chunks, 100, 2000)

	require.Len(t, result, 2, "both chunks are large enough, no merge")
}

func TestMergeSmallChunks_VerySmallMergesEvenIfOversize(t *testing.T) {

	smallIntro := "Intro:"
	bigCode := strings.Repeat("x", 300)
	chunks := []Document{
		{ID: "d-chunk-0", Content: smallIntro, Metadata: map[string]any{}},
		{ID: "d-chunk-1", Content: bigCode, Metadata: map[string]any{}},
	}

	result := mergeSmallChunks(chunks, 100, 200)

	require.Len(t, result, 1, "very small chunk should merge even if oversize")
	assert.Contains(t, result[0].Content, smallIntro)
	assert.Contains(t, result[0].Content, bigCode)
}

func TestMergeSmallChunks_HeadingPreservation(t *testing.T) {

	chunks := []Document{
		{ID: "d-chunk-0", Content: "Short intro.", Metadata: map[string]any{"heading": "Setup"}},
		{ID: "d-chunk-1", Content: strings.Repeat("b", 200), Metadata: map[string]any{}},
	}

	result := mergeSmallChunks(chunks, 100, 2000)

	require.Len(t, result, 1)
	assert.Equal(t, "Setup", result[0].Metadata["heading"])
}

func TestMergeSmallChunks_IDResequenced(t *testing.T) {

	document := Document{
		ID:      "doc",
		Content: "# A\n\n---\n\n# B\n\nReal content B.\n\n# C\n\nReal content C.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()), WithMinChunkSize(100))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	for i, c := range chunks {
		expected := fmt.Sprintf("doc-chunk-%d", i)
		assert.Equal(t, expected, c.ID, "chunk IDs should be sequential after merge")
	}
}

func TestMarkdownSplitter_WithMinChunkSize(t *testing.T) {

	document := Document{
		ID:      "doc",
		Content: "# Intro\n\nShort.\n\n# Main\n\n" + strings.Repeat("a", 500) + "\n\n# Footer\n\nBrief.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()), WithMinChunkSize(100))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	assert.Less(t, len(chunks), 3, "small chunks should be merged")

	var all strings.Builder
	for _, c := range chunks {
		all.WriteString(c.Content + " ")
	}
	assert.Contains(t, all.String(), "Short.")
	assert.Contains(t, all.String(), strings.Repeat("a", 500))
	assert.Contains(t, all.String(), "Brief.")
}

func TestMarkdownSplitter_HorizontalRuleFiltered(t *testing.T) {
	document := Document{
		ID:      "doc",
		Content: "# A\n\nContent A.\n\n---\n\n# B\n\nContent B.",
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()), WithMinChunkSize(50))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	for _, c := range chunks {
		trimmed := strings.TrimSpace(c.Content)
		assert.NotEqual(t, "---", trimmed, "horizontal rule should not be a chunk")
	}
}

func TestMarkdownSplitter_ProseStaysWithCode(t *testing.T) {

	code := "```go\n" + strings.Repeat("x := 1\n", 30) + "```"
	document := Document{
		ID:      "doc",
		Content: "# Example\n\nHere's the full code:\n\n" + code,
	}

	splitter, err := NewMarkdownSplitter(2000, 0, WithSplitterMarkdownParser(markdown_testparser.NewParser()), WithMinChunkSize(100))
	require.NoError(t, err)
	chunks := splitter.Split(document)

	if len(chunks) == 1 {
		assert.Contains(t, chunks[0].Content, "Here's the full code:")
		assert.Contains(t, chunks[0].Content, "x := 1")
	} else {

		found := false
		for _, c := range chunks {
			if strings.Contains(c.Content, "Here's the full code:") &&
				strings.Contains(c.Content, "x := 1") {
				found = true
				break
			}
		}
		assert.True(t, found, "intro prose should be in same chunk as code block")
	}
}

func TestGroupSegments_SmallProseStaysWithCode(t *testing.T) {

	segments := []segment{
		{text: "Intro:", isCode: false},
		{text: "```\n" + strings.Repeat("x", 500) + "\n```", isCode: true},
	}

	result := groupSegments(segments, 600, 100)

	require.Len(t, result, 1, "tiny prose should stay with following code")
	assert.Contains(t, result[0], "Intro:")
	assert.Contains(t, result[0], strings.Repeat("x", 500))
}

func TestStripHeadingLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "section with heading",
			in:   []byte("# Heading\nBody content here."),
			want: []byte("Body content here."),
		},
		{
			name: "section with no newline",
			in:   []byte("# Heading only"),
			want: nil,
		},
		{
			name: "empty section",
			in:   []byte{},
			want: nil,
		},
		{
			name: "newline at start",
			in:   []byte("\nBody starts here."),
			want: []byte("Body starts here."),
		},
		{
			name: "multiple newlines",
			in:   []byte("# Title\nLine 2\nLine 3"),
			want: []byte("Line 2\nLine 3"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := stripHeadingLine(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStartsWithCodeFence(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want bool
	}{
		{name: "backtick fence", text: "```go\nfunc main() {}", want: true},
		{name: "tilde fence", text: "~~~bash\necho hello", want: true},
		{name: "backtick fence with leading whitespace", text: "  ```\ncode", want: true},
		{name: "no fence", text: "Just normal text", want: false},
		{name: "empty string", text: "", want: false},
		{name: "single backtick", text: "`code`", want: false},
		{name: "double backtick", text: "``code``", want: false},
		{name: "tilde in middle", text: "some ~~~ text", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := startsWithCodeFence(tc.text)
			assert.Equal(t, tc.want, got)
		})
	}
}
