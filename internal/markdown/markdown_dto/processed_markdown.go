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

package markdown_dto

import (
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
)

// ProcessedMarkdown holds all the build artefacts generated from a single
// .md source file. This is the primary return type of the MarkdownService.
type ProcessedMarkdown struct {
	// PageAST holds the parsed Markdown content as a Piko AST node tree.
	// It is ready to be passed to a <piko-ast-renderer> for display.
	PageAST *ast_domain.TemplateAST

	// ExcerptAST is a separate, optional renderable artefact used on other pages
	// (e.g., a blog index) to show a preview; nil if no excerpt is defined.
	ExcerptAST *ast_domain.TemplateAST

	// Diagnostics holds any warnings or errors found during transformation.
	Diagnostics []*ast_domain.Diagnostic

	// Metadata contains all the data about the page that is not part of the AST.
	// This data can be serialised and should be passed as props to the layout
	// component for use in expressions like `{{ state.Page.Title }}`.
	Metadata PageMetadata
}

// PageMetadata holds all extracted data for a page, excluding the AST.
type PageMetadata struct {
	// PublishDate is the publication date from the "date" frontmatter field.
	PublishDate time.Time

	// Title is the document heading taken from frontmatter.
	Title string

	// Description is a short summary of the page content from frontmatter.
	Description string

	// Navigation holds the navigation structure from frontmatter.
	Navigation *NavigationMetadata

	// Frontmatter holds custom key-value data from the document front matter.
	Frontmatter map[string]any

	// Tags contains labels used to group and filter content.
	Tags []string

	// Sections holds heading data used to build a table of contents.
	// The actual content for each section is stored in PageAST.
	Sections []SectionData

	// Images holds metadata for images found on the page.
	Images []ImageMeta

	// Links holds metadata for hyperlinks found in the documentation.
	Links []LinkMeta

	// ReadingTime is the estimated reading time in minutes.
	ReadingTime int

	// WordCount is the total number of words in the document.
	WordCount int

	// Draft indicates whether this content is a draft that should not be
	// published.
	Draft bool
}

// SectionData represents a logical part of the document, initiated by a heading.
// It is a data-only struct, primarily for building a Table of Contents.
//
// For hierarchical ToC structures, use collection_dto.SectionNode which
// is the provider-agnostic output type produced by BuildSectionTree.
type SectionData struct {
	// Title is the heading text for this section.
	Title string

	// Slug is the URL-safe identifier for this section.
	Slug string

	// Level is the heading depth from 1 to 6, matching h1 to h6.
	Level int
}

// ImageMeta holds data about an image tag found in a document.
type ImageMeta struct {
	// Src is the source URL or file path for the image.
	Src string

	// Alt is the alternative text for the image.
	Alt string
}

// LinkMeta holds details about an anchor tag found in a document.
type LinkMeta struct {
	// Href is the URL that the link points to.
	Href string

	// Text is the display text for the link.
	Text string
}
