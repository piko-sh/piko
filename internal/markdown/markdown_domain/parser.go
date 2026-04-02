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

package markdown_domain

import (
	"cmp"
	"context"
	"fmt"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

// markdownService orchestrates converting raw Markdown into structured build
// artefacts. It implements MarkdownService.
type markdownService struct {
	// parser parses raw markdown content and extracts frontmatter.
	parser MarkdownParserPort

	// highlighter applies syntax highlighting to code blocks.
	highlighter Highlighter
}

var _ MarkdownService = (*markdownService)(nil)

// Process is the main entry point for the service that converts raw markdown
// content into a fully processed DTO containing distinct build artefacts
// (Page AST, Excerpt AST, Metadata). It follows a multi-step pipeline to
// ensure correctness and maintainability.
//
// Takes content ([]byte) which contains the raw bytes of a markdown file.
// Takes sourcePath (string) which is the file path used for error messages.
//
// Returns *markdown_dto.ProcessedMarkdown which contains the processed build
// artefacts ready for rendering.
// Returns error when parsing, frontmatter processing, or AST transformation
// fails.
func (s *markdownService) Process(ctx context.Context, content []byte, sourcePath string) (*markdown_dto.ProcessedMarkdown, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "MarkdownService.Process",
		logger_domain.String("sourcePath", sourcePath),
	)
	defer span.End()

	l.Internal("Parsing raw markdown content and frontmatter...")
	doc, rawFrontmatter, err := s.parser.Parse(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("markdown parsing failed for %s: %w", sourcePath, err)
	}
	l.Internal("Raw parsing successful.", logger_domain.Int("frontmatter_keys", len(rawFrontmatter)))

	l.Internal("Processing and validating frontmatter...")
	frontmatter, err := ParseFrontmatter(rawFrontmatter)
	if err != nil {
		return nil, fmt.Errorf("frontmatter processing failed for %s: %w", sourcePath, err)
	}
	l.Internal("Frontmatter processed successfully.")

	l.Internal("Walking markdown AST to transform into Piko build artefacts...")
	processedData, err := transformMarkdownAST(ctx, doc, content, sourcePath, s.highlighter)
	if err != nil {
		return nil, fmt.Errorf("markdown AST transformation failed for %s: %w", sourcePath, err)
	}
	l.Internal("AST transformation successful.",
		logger_domain.Int("sections", len(processedData.Metadata.Sections)),
		logger_domain.Int("word_count", processedData.Metadata.WordCount),
		logger_domain.Int("diagnostics", len(processedData.Diagnostics)),
	)

	l.Internal("Performing final analysis (e.g., calculating reading time)...")
	readingTime := calculateReadingTime(processedData.Metadata.WordCount)
	l.Internal("Analysis complete.", logger_domain.Int("reading_time_minutes", readingTime))

	l.Internal("Assembling final ProcessedMarkdown DTO...")
	result := assembleFinalResult(processedData, frontmatter, readingTime)
	l.Internal("Markdown processing complete.", logger_domain.String("title", result.Metadata.Title))
	return result, nil
}

// NewMarkdownService creates a new markdown service with the given parser and
// highlighter. It decouples the service from any specific Markdown library
// (like Goldmark) by depending on a MarkdownParserPort.
//
// Takes parser (MarkdownParserPort) which handles the initial parsing of
// markdown content.
// Takes highlighter (Highlighter) which provides syntax highlighting for code
// blocks. May be nil if highlighting is not needed.
//
// Returns MarkdownService which is the configured service ready for use.
func NewMarkdownService(parser MarkdownParserPort, highlighter Highlighter) MarkdownService {
	return &markdownService{
		parser:      parser,
		highlighter: highlighter,
	}
}

// calculateReadingTime estimates the reading time in minutes based on word
// count. Uses an average reading speed of 225 words per minute.
//
// Takes wordCount (int) which is the number of words in the content.
//
// Returns int which is the reading time in minutes, with a minimum of 1.
func calculateReadingTime(wordCount int) int {
	const averageWordsPerMinute = 225
	if wordCount == 0 {
		return 0
	}
	return cmp.Or((wordCount+averageWordsPerMinute-1)/averageWordsPerMinute, 1)
}

// assembleFinalResult combines the processed parts into the final result.
//
// Takes processedData (*markdown_dto.ProcessedMarkdown) which provides the
// base result from the markdown walker.
// Takes frontmatter (*Frontmatter) which contains the parsed front matter.
// Takes readingTime (int) which specifies the estimated reading time.
//
// Returns *markdown_dto.ProcessedMarkdown which is the combined result with
// metadata fields set from the frontmatter.
func assembleFinalResult(processedData *markdown_dto.ProcessedMarkdown, frontmatter *Frontmatter, readingTime int) *markdown_dto.ProcessedMarkdown {
	result := processedData
	result.Metadata.Title = frontmatter.Title
	result.Metadata.Description = frontmatter.Description
	result.Metadata.Navigation = frontmatter.Navigation
	result.Metadata.Frontmatter = frontmatter.Custom
	result.Metadata.Tags = frontmatter.Tags
	result.Metadata.PublishDate = frontmatter.PublishDate
	result.Metadata.Draft = frontmatter.Draft
	result.Metadata.ReadingTime = readingTime
	return result
}
