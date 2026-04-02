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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package markdown_ast defines the piko-native markdown AST used
// by parser adapters and the markdown domain.
//
// The tree is built from the [Node] interface and [BaseNode]
// embeddable struct, with concrete types for CommonMark and GFM
// constructs: headings, paragraphs, lists, code blocks, emphasis,
// links, images, tables, and more. [Walk] provides depth-first
// traversal, and [Attributable] allows nodes to carry key-value
// metadata such as auto-generated heading IDs.
package markdown_ast
