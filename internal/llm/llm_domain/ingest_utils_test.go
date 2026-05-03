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
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyMetadata_Nil(t *testing.T) {
	result := copyMetadata(nil)
	assert.Nil(t, result)
}

func TestCopyMetadata_Empty(t *testing.T) {
	src := map[string]any{}
	result := copyMetadata(src)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestCopyMetadata_CopiesValues(t *testing.T) {
	src := map[string]any{"key": "value", "num": 42}
	result := copyMetadata(src)
	assert.Equal(t, src, result)
}

func TestCopyMetadata_IndependentCopy(t *testing.T) {
	src := map[string]any{"key": "value"}
	result := copyMetadata(src)
	result["key"] = "modified"
	assert.Equal(t, "value", src["key"])
}

func TestSelectDone_NotDone(t *testing.T) {
	ctx := context.Background()
	assert.False(t, selectDone(ctx))
}

func TestSelectDone_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))
	assert.True(t, selectDone(ctx))
}

func TestNewRecursiveCharacterSplitter_ValidParams(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(100, 10)
	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, 100, s.chunkSize)
	assert.Equal(t, 10, s.overlap)
	assert.Equal(t, []string{"\n\n", "\n", " ", ""}, s.separators)
}

func TestNewRecursiveCharacterSplitter_ZeroChunkSize(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(0, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, s.chunkSize)
}

func TestNewRecursiveCharacterSplitter_NegativeChunkSize(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(-5, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, s.chunkSize)
}

func TestNewRecursiveCharacterSplitter_ErrorOnOverlapGEChunkSize(t *testing.T) {
	_, err := NewRecursiveCharacterSplitter(10, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overlap (10) must be less than chunkSize (10)")

	_, err = NewRecursiveCharacterSplitter(10, 15)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overlap (15) must be less than chunkSize (10)")
}

func TestSplit_ShortDocument(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(100, 0)
	require.NoError(t, err)
	document := Document{ID: "doc1", Content: "short text", Metadata: map[string]any{"source": "test"}}

	chunks := s.Split(document)
	require.Len(t, chunks, 1)
	assert.Equal(t, "doc1-chunk-0", chunks[0].ID)
	assert.Equal(t, "short text", chunks[0].Content)
	assert.Equal(t, map[string]any{"source": "test"}, chunks[0].Metadata)
}

func TestSplit_MetadataCopiedPerChunk(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(10, 0)
	require.NoError(t, err)
	document := Document{
		ID:       "doc1",
		Content:  "aaaaaaaaaa bbbbbbbbbb",
		Metadata: map[string]any{"source": "test"},
	}

	chunks := s.Split(document)
	require.True(t, len(chunks) > 1)

	chunks[0].Metadata["extra"] = "added"

	_, hasExtra := chunks[1].Metadata["extra"]
	assert.False(t, hasExtra)

	_, hasExtra = document.Metadata["extra"]
	assert.False(t, hasExtra)
}

func TestSplit_NilMetadata(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(100, 0)
	require.NoError(t, err)
	document := Document{ID: "doc1", Content: "text", Metadata: nil}

	chunks := s.Split(document)
	require.Len(t, chunks, 1)
	assert.Nil(t, chunks[0].Metadata)
}

func TestSplit_ChunkIDFormat(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(5, 0)
	require.NoError(t, err)
	document := Document{ID: "myDoc", Content: "aaaa bbbbb ccccc"}

	chunks := s.Split(document)
	for i := range chunks {
		assert.Contains(t, chunks[i].ID, "myDoc-chunk-")
	}
}

func TestSplit_ByDoubleNewline(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(20, 0)
	require.NoError(t, err)
	document := Document{ID: "doc", Content: "paragraph one\n\nparagraph two"}

	chunks := s.Split(document)
	require.True(t, len(chunks) >= 2)
	assert.Equal(t, "paragraph one", chunks[0].Content)
	assert.Equal(t, "paragraph two", chunks[1].Content)
}

func TestSplit_BySingleNewline(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(15, 0)
	require.NoError(t, err)
	document := Document{ID: "doc", Content: "line one\nline two\nline three"}

	chunks := s.Split(document)
	assert.True(t, len(chunks) >= 2)
}

func TestSplit_BySpace(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(10, 0)
	require.NoError(t, err)
	document := Document{ID: "doc", Content: "word1 word2 word3 word4 word5"}

	chunks := s.Split(document)
	assert.True(t, len(chunks) >= 2)
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len(chunk.Content), 10)
	}
}

func TestSplit_HardSplitFallback(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(5, 0)
	require.NoError(t, err)
	document := Document{ID: "doc", Content: "abcdefghij"}

	chunks := s.Split(document)
	assert.True(t, len(chunks) >= 2)
	assert.Equal(t, "abcde", chunks[0].Content)
	assert.Equal(t, "fghij", chunks[1].Content)
}

func TestSplit_WithOverlap(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(5, 2)
	require.NoError(t, err)
	document := Document{ID: "doc", Content: "abcdefghij"}

	chunks := s.Split(document)
	assert.True(t, len(chunks) >= 2)

}

func TestSplit_EmptyContent(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(100, 0)
	require.NoError(t, err)
	document := Document{ID: "doc", Content: ""}

	chunks := s.Split(document)
	require.Len(t, chunks, 1)
	assert.Equal(t, "", chunks[0].Content)
}

func TestNewFSLoader_DefaultPattern(t *testing.T) {
	fsys := fstest.MapFS{}
	loader := NewFSLoader(fsys)
	assert.Equal(t, []string{"*"}, loader.patterns)
}

func TestNewFSLoader_CustomPatterns(t *testing.T) {
	fsys := fstest.MapFS{}
	loader := NewFSLoader(fsys, "*.txt", "*.md")
	assert.Equal(t, []string{"*.txt", "*.md"}, loader.patterns)
}

func TestFSLoader_Load_Empty(t *testing.T) {
	fsys := fstest.MapFS{}
	loader := NewFSLoader(fsys, "*.txt")

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestFSLoader_Load_MatchingFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("Hello World")},
		"other.md":  &fstest.MapFile{Data: []byte("# Heading")},
	}
	loader := NewFSLoader(fsys, "*.txt")

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "hello.txt", docs[0].ID)
	assert.Equal(t, "Hello World", docs[0].Content)
	assert.Equal(t, "hello.txt", docs[0].Metadata["source"])
}

func TestFSLoader_Load_AllFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("aaa")},
		"b.txt": &fstest.MapFile{Data: []byte("bbb")},
	}
	loader := NewFSLoader(fsys)

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestFSLoader_Load_InvalidGlobPattern(t *testing.T) {
	fsys := fstest.MapFS{}
	loader := NewFSLoader(fsys, "[invalid")

	_, err := loader.Load(context.Background())
	assert.Error(t, err)
}

func TestFSLoader_Load_CancelledContext(t *testing.T) {
	fsys := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("content")},
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	loader := NewFSLoader(fsys, "*.txt")
	_, err := loader.Load(ctx)
	assert.Error(t, err)
}

func TestNewRecursiveFSLoader_DefaultPattern(t *testing.T) {
	fsys := fstest.MapFS{}
	loader := NewRecursiveFSLoader(fsys)
	assert.Equal(t, []string{"*"}, loader.patterns)
}

func TestNewRecursiveFSLoader_CustomPatterns(t *testing.T) {
	fsys := fstest.MapFS{}
	loader := NewRecursiveFSLoader(fsys, "*.md", "*.txt")
	assert.Equal(t, []string{"*.md", "*.txt"}, loader.patterns)
}

func TestRecursiveFSLoader_Load_Empty(t *testing.T) {
	fsys := fstest.MapFS{}
	loader := NewRecursiveFSLoader(fsys, "*.md")

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestRecursiveFSLoader_Load_NestedFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"top.md":             &fstest.MapFile{Data: []byte("top level")},
		"sub/nested.md":      &fstest.MapFile{Data: []byte("nested")},
		"sub/deep/deeper.md": &fstest.MapFile{Data: []byte("deep nested")},
		"sub/skip.txt":       &fstest.MapFile{Data: []byte("not markdown")},
	}
	loader := NewRecursiveFSLoader(fsys, "*.md")

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	require.Len(t, docs, 3)

	ids := make(map[string]bool)
	for _, document := range docs {
		ids[document.ID] = true
	}
	assert.True(t, ids["top.md"])
	assert.True(t, ids["sub/nested.md"])
	assert.True(t, ids["sub/deep/deeper.md"])
}

func TestRecursiveFSLoader_Load_MetadataSourceIsPath(t *testing.T) {
	fsys := fstest.MapFS{
		"a/b/file.md": &fstest.MapFile{Data: []byte("content")},
	}
	loader := NewRecursiveFSLoader(fsys, "*.md")

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "a/b/file.md", docs[0].Metadata["source"])
}

func TestRecursiveFSLoader_Load_MultiplePatterns(t *testing.T) {
	fsys := fstest.MapFS{
		"readme.md": &fstest.MapFile{Data: []byte("markdown")},
		"notes.txt": &fstest.MapFile{Data: []byte("text")},
		"image.png": &fstest.MapFile{Data: []byte("binary")},
	}
	loader := NewRecursiveFSLoader(fsys, "*.md", "*.txt")

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestRecursiveFSLoader_Load_CancelledContext(t *testing.T) {
	fsys := fstest.MapFS{
		"a/file.md": &fstest.MapFile{Data: []byte("content")},
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	loader := NewRecursiveFSLoader(fsys, "*.md")
	_, err := loader.Load(ctx)
	assert.Error(t, err)
}

func TestComputeOverlap_ZeroOverlapReturnsEmpty(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(100, 0)
	require.NoError(t, err)
	assert.Equal(t, "", s.computeOverlap("hello world"))
}

func TestComputeOverlap_TextShorterThanOverlapReturnsAll(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(100, 50)
	require.NoError(t, err)
	assert.Equal(t, "short", s.computeOverlap("short"))
}

func TestComputeOverlap_AsciiTrailingWindow(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(100, 5)
	require.NoError(t, err)
	assert.Equal(t, "world", s.computeOverlap("hello world"))
}

func TestComputeOverlap_MultiByteRunesProduceValidUtf8(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(1024, 7)
	require.NoError(t, err)
	input := "hello " + strings.Repeat("世界", 20)
	got := s.computeOverlap(input)
	assert.True(t, utf8.ValidString(got), "computeOverlap returned invalid UTF-8: %q", got)
	assert.LessOrEqual(t, len(got), 7, "overlap exceeds byte budget")
}

func TestComputeOverlap_BudgetSnapsForwardToRuneStart(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(1024, 4)
	require.NoError(t, err)
	input := "ab" + "世" + "cd"
	got := s.computeOverlap(input)
	assert.True(t, utf8.ValidString(got), "computeOverlap returned invalid UTF-8: %q", got)
	assert.LessOrEqual(t, len(got), 4, "overlap exceeds byte budget")
	assert.Equal(t, "cd", got)
}

func TestComputeOverlap_EmojiOverlapValidUtf8(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(1024, 9)
	require.NoError(t, err)
	input := "abc" + strings.Repeat("\U0001F600", 5)
	got := s.computeOverlap(input)
	assert.True(t, utf8.ValidString(got), "computeOverlap returned invalid UTF-8: %q", got)
	assert.LessOrEqual(t, len(got), 9, "overlap exceeds byte budget")
}
