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

// Package formatter_domain handles formatting of Piko template files
// (.pk), covering all Single File Component (SFC) blocks (template,
// script, style, and i18n) with consistent, opinionated rules that
// preserve semantic meaning while improving readability. It supports
// full-file and range-based formatting for LSP integration.
//
// # Block formatting
//
// Each SFC block type uses specialised formatting:
//
//   - Template: AST-based pretty-printing with intelligent inline vs block
//     decisions, attribute sorting, and whitespace normalisation
//   - Script (Go): Formatted using go/format for standard Go style
//   - Style (CSS): Formatted using esbuild's CSS parser and printer
//   - i18n (JSON): Formatted with consistent 2-space indentation
//
// # Usage
//
// Create a formatter service and format a .pk file:
//
//	service := formatter_domain.NewFormatterService()
//	formatted, err := service.Format(ctx, source)
//
// For custom options:
//
//	opts := &formatter_domain.FormatOptions{
//	    IndentSize:     4,
//	    MaxLineLength:  120,
//	    SortAttributes: true,
//	}
//	formatted, err := service.FormatWithOptions(ctx, source, opts)
//
// # Thread safety
//
// formatterServiceImpl instances are safe for concurrent use. Each Format call
// creates its own internal state and does not modify shared mutable state.
package formatter_domain
