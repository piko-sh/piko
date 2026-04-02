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

// Package markdown_domain orchestrates the transformation of Markdown content
// into structured build artefacts.
//
// The [MarkdownService] drives a multi-step pipeline that parses raw Markdown,
// extracts frontmatter metadata via [MarkdownParserPort], transforms the
// piko-native markdown AST into Piko's internal template AST format, and
// produces a ProcessedMarkdown DTO containing the page AST, excerpt AST, and
// structured metadata.
//
// # Processing pipeline
//
// The service processes markdown through these stages:
//
//  1. Parse raw markdown and extract frontmatter via [MarkdownParserPort]
//  2. Validate and structure frontmatter fields
//  3. Walk the piko markdown AST, transforming nodes to Piko template format
//  4. Collect metadata (sections, images, links, word count)
//  5. Build excerpt AST from content before <!--more--> separator
//  6. Assemble the final ProcessedMarkdown DTO
//
// # Shortcode support
//
// Fenced code blocks with the "piko" info string are treated as component
// shortcodes. The syntax ` ```piko ComponentName prop="value" ` creates
// an inline component invocation within markdown content.
package markdown_domain
