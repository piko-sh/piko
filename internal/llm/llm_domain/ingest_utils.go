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
	"io"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FSLoader loads documents from a file system.
type FSLoader struct {
	// fsys is the file system used to read files matching the patterns.
	fsys fs.FS

	// patterns contains the glob patterns used to match files for loading.
	patterns []string
}

// NewFSLoader creates a new FSLoader with the given file system and patterns.
//
// Takes fsys (fs.FS) which provides the file system to load from.
// Takes patterns (...string) which specifies the glob patterns to match. If
// empty, defaults to "*" to match all files.
//
// Returns *FSLoader which is ready to load matching files from the file system.
func NewFSLoader(fsys fs.FS, patterns ...string) *FSLoader {
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}
	return &FSLoader{
		fsys:     fsys,
		patterns: patterns,
	}
}

// Load retrieves documents from the file system matching the patterns.
// Files that cannot be opened or read are skipped, and a summary error is
// returned alongside any successfully loaded documents.
//
// Returns []Document which contains all successfully loaded documents.
// Returns error when a glob pattern is invalid, the context is cancelled,
// or one or more files could not be read.
func (l *FSLoader) Load(ctx context.Context) ([]Document, error) {
	var docs []Document
	var skipped []string

	for _, pattern := range l.patterns {
		matches, err := fs.Glob(l.fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("globbing pattern %q: %w", pattern, err)
		}

		for _, match := range matches {
			if selectDone(ctx) {
				return nil, ctx.Err()
			}

			file, err := l.fsys.Open(match)
			if err != nil {
				skipped = append(skipped, match)
				continue
			}

			content, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				skipped = append(skipped, match)
				continue
			}

			docs = append(docs, Document{
				ID:      match,
				Content: string(content),
				Metadata: map[string]any{
					"source": match,
				},
			})
		}
	}

	if len(skipped) > 0 {
		return docs, fmt.Errorf("failed to load %d file(s): %s", len(skipped), strings.Join(skipped, ", "))
	}

	return docs, nil
}

// RecursiveFSLoader loads documents by walking a file system tree recursively.
// Unlike FSLoader which uses fs.Glob (single-level matching), this loader
// traverses all subdirectories to find files whose names match the patterns.
type RecursiveFSLoader struct {
	// fsys is the file system to walk recursively.
	fsys fs.FS

	// patterns contains the glob patterns matched against file names.
	patterns []string
}

// NewRecursiveFSLoader creates a new RecursiveFSLoader with the given file
// system and patterns. Patterns are matched against file names (not paths)
// using filepath.Match.
//
// Takes fsys (fs.FS) which provides the file system to walk.
// Takes patterns (...string) which specifies the glob patterns to match file
// names against. If empty, defaults to "*" to match all files.
//
// Returns *RecursiveFSLoader which is ready to load matching files.
func NewRecursiveFSLoader(fsys fs.FS, patterns ...string) *RecursiveFSLoader {
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}
	return &RecursiveFSLoader{
		fsys:     fsys,
		patterns: patterns,
	}
}

// Load walks the file system tree and retrieves all files whose names match
// the configured patterns. Files that cannot be read are skipped.
//
// Returns []Document which contains all successfully loaded documents.
// Returns error when the walk fails or the context is cancelled.
func (l *RecursiveFSLoader) Load(ctx context.Context) ([]Document, error) {
	var docs []Document
	var skipped []string

	err := fs.WalkDir(l.fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if selectDone(ctx) {
			return ctx.Err()
		}
		if !l.matchesAnyPattern(path) {
			return nil
		}

		document, ok := l.readFileAsDocument(path)
		if !ok {
			skipped = append(skipped, path)
			return nil
		}

		docs = append(docs, document)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking file system: %w", err)
	}

	if len(skipped) > 0 {
		return docs, fmt.Errorf("failed to load %d file(s): %s", len(skipped), strings.Join(skipped, ", "))
	}

	return docs, nil
}

// matchesAnyPattern reports whether the file at path matches any of the
// configured glob patterns. Patterns are matched against the base file name.
//
// Takes path (string) which is the full file path to check.
//
// Returns bool which is true when at least one pattern matches.
func (l *RecursiveFSLoader) matchesAnyPattern(path string) bool {
	name := filepath.Base(path)
	for _, pattern := range l.patterns {
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
	}
	return false
}

// readFileAsDocument reads a file from the file system and returns it as a
// Document. Returns false when the file cannot be opened or read.
//
// Takes path (string) which is the file path to read.
//
// Returns Document which contains the file content and metadata.
// Returns bool which is false when the file could not be read.
func (l *RecursiveFSLoader) readFileAsDocument(path string) (Document, bool) {
	file, err := l.fsys.Open(path)
	if err != nil {
		return Document{}, false
	}

	content, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		return Document{}, false
	}

	return Document{
		ID:      path,
		Content: string(content),
		Metadata: map[string]any{
			"source": path,
		},
	}, true
}

// RecursiveCharacterSplitter splits text into chunks by recursively trying
// a list of separators. It aims to keep chunks within a target size while
// maintaining overlap for context.
type RecursiveCharacterSplitter struct {
	// separators lists the strings used to split text, tried in order.
	separators []string

	// chunkSize is the maximum length in bytes for each text chunk.
	chunkSize int

	// overlap is the number of characters to repeat from the previous chunk.
	overlap int
}

// NewRecursiveCharacterSplitter creates a new RecursiveCharacterSplitter.
//
// Takes chunkSize (int) which specifies the target size for each chunk.
// Takes overlap (int) which specifies the number of characters to overlap
// between chunks. Must be less than chunkSize.
//
// Returns *RecursiveCharacterSplitter which is configured and ready for use.
// Returns error when overlap is greater than or equal to chunkSize as this
// would cause an infinite loop during hard splitting.
func NewRecursiveCharacterSplitter(chunkSize, overlap int) (*RecursiveCharacterSplitter, error) {
	if chunkSize <= 0 {
		chunkSize = 1
	}
	if overlap >= chunkSize {
		return nil, fmt.Errorf("overlap (%d) must be less than chunkSize (%d)", overlap, chunkSize)
	}
	return &RecursiveCharacterSplitter{
		separators: []string{"\n\n", "\n", " ", ""},
		chunkSize:  chunkSize,
		overlap:    overlap,
	}, nil
}

// Split divides a document into chunks. Each chunk receives its own copy of
// the parent document's metadata to prevent shared-reference mutations.
//
// Takes document (Document) which is the document to split into smaller chunks.
//
// Returns []Document which contains the resulting chunks with unique IDs.
func (s *RecursiveCharacterSplitter) Split(document Document) []Document {
	chunks := s.splitText(document.Content, s.separators)

	finalDocuments := make([]Document, 0, len(chunks))
	for i, chunk := range chunks {
		finalDocuments = append(finalDocuments, Document{
			ID:       document.ID + fmt.Sprintf("-chunk-%d", i),
			Content:  chunk,
			Metadata: copyMetadata(document.Metadata),
		})
	}
	return finalDocuments
}

// splitText divides text into chunks using recursive separator matching.
//
// Takes text (string) which is the content to split into chunks.
// Takes separators ([]string) which lists separators to try in order.
//
// Returns []string which contains chunks no larger than the configured size.
func (s *RecursiveCharacterSplitter) splitText(text string, separators []string) []string {
	if len(text) <= s.chunkSize {
		return []string{text}
	}

	if len(separators) == 0 {
		return s.hardSplit(text)
	}

	separator := separators[0]
	parts := strings.Split(text, separator)
	finalChunks := s.assembleChunks(parts, separator)

	return s.refineChunks(finalChunks, separators[1:])
}

// assembleChunks joins parts back into chunks that fit within the configured
// size, inserting the original separator between parts and applying overlap
// when a chunk boundary is reached.
//
// Takes parts ([]string) which are the text segments to reassemble.
// Takes separator (string) which is the delimiter placed between parts.
//
// Returns []string which contains the assembled chunks.
func (s *RecursiveCharacterSplitter) assembleChunks(parts []string, separator string) []string {
	var finalChunks []string
	var currentChunk strings.Builder

	for _, part := range parts {
		if currentChunk.Len()+len(part)+len(separator) > s.chunkSize && currentChunk.Len() > 0 {
			finalChunks = append(finalChunks, currentChunk.String())
			overlapText := s.computeOverlap(currentChunk.String())
			currentChunk.Reset()
			currentChunk.WriteString(overlapText)
		}

		if currentChunk.Len() > 0 && separator != "" {
			currentChunk.WriteString(separator)
		}
		currentChunk.WriteString(part)
	}

	if currentChunk.Len() > 0 {
		finalChunks = append(finalChunks, currentChunk.String())
	}

	return finalChunks
}

// computeOverlap returns the trailing portion of text to carry forward into the next chunk.
//
// The slice boundary is snapped forward to the nearest UTF-8 rune start so the
// result is always valid UTF-8, even when the byte budget would otherwise fall
// inside a multi-byte sequence. The returned slice may therefore be slightly
// shorter than s.overlap bytes.
//
// Takes text (string) which is the completed chunk text.
//
// Returns string which is the trailing overlap portion, or empty if overlap
// is zero.
func (s *RecursiveCharacterSplitter) computeOverlap(text string) string {
	if s.overlap <= 0 {
		return ""
	}
	if len(text) <= s.overlap {
		return text
	}
	start := len(text) - s.overlap
	for start < len(text) && !utf8.RuneStart(text[start]) {
		start++
	}
	return text[start:]
}

// refineChunks recursively splits any oversized chunks using the remaining
// separators.
//
// Takes chunks ([]string) which are the chunks to refine.
// Takes remainingSeparators ([]string) which are the separators still
// available.
//
// Returns []string which contains chunks no larger than the configured size.
func (s *RecursiveCharacterSplitter) refineChunks(chunks []string, remainingSeparators []string) []string {
	var refined []string
	for _, chunk := range chunks {
		if len(chunk) > s.chunkSize {
			refined = append(refined, s.splitText(chunk, remainingSeparators)...)
		} else {
			refined = append(refined, chunk)
		}
	}
	return refined
}

// hardSplit splits text into fixed-size chunks when no separators work.
//
// Both the start and end of each chunk are snapped to the nearest rune
// boundary so chunks remain valid UTF-8 even when the byte budget would
// otherwise cut mid-rune. Resulting chunks may therefore be slightly smaller
// than s.chunkSize.
//
// Takes text (string) which is the content to split into chunks.
//
// Returns []string which contains the text divided into overlapping chunks
// of approximately the configured size in bytes.
func (s *RecursiveCharacterSplitter) hardSplit(text string) []string {
	var chunks []string
	stride := max(s.chunkSize-s.overlap, 1)
	for i := 0; i < len(text); i += stride {
		start := snapForwardToRune(text, i)
		if start >= len(text) {
			break
		}
		end := snapBackwardToRune(text, start, start+s.chunkSize)
		if end <= start {
			break
		}
		chunks = append(chunks, text[start:end])
		if end == len(text) {
			break
		}
	}
	return chunks
}

// snapForwardToRune returns the smallest index >= start that begins a UTF-8 rune.
//
// Takes text (string) which is the source string to scan.
// Takes start (int) which is the candidate index to snap forward from.
//
// Returns int which is the snapped index, or len(text) if no rune-start byte
// exists at or after start.
func snapForwardToRune(text string, start int) int {
	for start < len(text) && !utf8.RuneStart(text[start]) {
		start++
	}
	return start
}

// snapBackwardToRune clamps end to len(text) then walks back to the nearest
// rune boundary.
//
// The walk stops once end either equals start or sits on a rune-start byte,
// keeping the returned slice bounds valid UTF-8.
//
// Takes text (string) which is the source string to scan.
// Takes start (int) which is the lower bound that end must not cross.
// Takes end (int) which is the candidate end index to snap backward.
//
// Returns int which is the snapped end index, clamped to len(text) and never
// less than start.
func snapBackwardToRune(text string, start, end int) int {
	if end >= len(text) {
		return len(text)
	}
	for end > start && !utf8.RuneStart(text[end]) {
		end--
	}
	return end
}

// copyMetadata creates a shallow copy of a metadata map so that each chunk
// gets its own map instance.
//
// Takes src (map[string]any) which is the metadata map to copy.
//
// Returns map[string]any which is a new map containing the same key-value
// pairs, or nil if src is nil.
func copyMetadata(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	maps.Copy(dst, src)
	return dst
}

// selectDone checks whether the context has been cancelled.
//
// Returns bool which is true if the context is done, false otherwise.
func selectDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
