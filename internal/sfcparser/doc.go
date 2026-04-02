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

// Package sfcparser provides parsing for Piko Single File Components (SFC).
//
// It extracts template, script, style, and internationalisation blocks
// from .pk files, handling both raw text elements (script and style)
// which cannot be nested, and nestable elements (template and i18n)
// which support arbitrary HTML content. It sits between file loading
// and the AST parser in the processing pipeline.
//
// # Usage
//
//	result, err := sfcparser.Parse(sfcBytes)
//	if err != nil {
//	    return err
//	}
//
//	// Access the template content
//	templateHTML := result.Template
//
//	// Find the Go script block
//	if goScript, ok := result.GoScript(); ok {
//	    processGoCode(goScript.Content)
//	}
//
//	// Iterate over styles
//	for _, style := range result.Styles {
//	    processCSS(style.Content, style.Attributes)
//	}
//
// # Supported block types
//
// The parser recognises four top-level block types:
//
//   - <template>: HTML content for the component (only first is used)
//   - <script>: Code blocks with type/lang detection for Go, JS, or TypeScript
//   - <style>: CSS blocks with optional global/scoped attributes
//   - <i18n>: Internationalisation content with lang attribute
//
// # Thread safety
//
// The Parse function is safe for concurrent use. Each call creates its own
// parser instance with no shared state.
package sfcparser
