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
	"context"

	"piko.sh/piko/internal/highlight/highlight_domain"
	"piko.sh/piko/internal/markdown/markdown_ast"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

// MarkdownService defines the contract for processing markdown files into
// structured build artefacts.
type MarkdownService interface {
	// Process transforms raw markdown content into a structured representation.
	//
	// Takes content ([]byte) which is the raw markdown to process.
	// Takes sourcePath (string) which identifies the source file for error
	// reporting.
	//
	// Returns *markdown_dto.ProcessedMarkdown which contains the parsed result.
	// Returns error when the markdown content cannot be processed.
	Process(ctx context.Context, content []byte, sourcePath string) (*markdown_dto.ProcessedMarkdown, error)
}

// MarkdownParserPort defines the adapter contract for parsing raw markdown
// content into a piko-native AST.
type MarkdownParserPort interface {
	// Parse processes the content and extracts the AST and frontmatter.
	//
	// Takes content ([]byte) which is the raw input to parse.
	//
	// Returns doc (*markdown_ast.Document) which is the root of the parsed
	// syntax tree.
	// Returns frontmatter (map[string]any) which contains metadata from the
	// content.
	// Returns error when parsing fails.
	Parse(ctx context.Context, content []byte) (doc *markdown_ast.Document, frontmatter map[string]any, err error)
}

// HTMLConverter defines the contract for converting raw markdown bytes to HTML.
type HTMLConverter interface {
	// Convert renders markdown to HTML with raw HTML stripped for safety.
	//
	// Takes markdown ([]byte) which is the markdown content to convert.
	//
	// Returns []byte which is the rendered HTML output.
	// Returns error when conversion fails.
	Convert(markdown []byte) ([]byte, error)

	// ConvertUnsafe renders markdown to HTML with raw HTML preserved.
	//
	// Takes markdown ([]byte) which is the markdown content to convert.
	//
	// Returns []byte which is the rendered HTML output.
	// Returns error when conversion fails.
	//
	// WARNING: Only use with fully trusted content.
	ConvertUnsafe(markdown []byte) ([]byte, error)
}

// Highlighter is a type alias for highlight_domain.Highlighter.
type Highlighter = highlight_domain.Highlighter
