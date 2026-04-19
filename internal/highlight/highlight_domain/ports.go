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

package highlight_domain

// Highlighter is the port interface for syntax highlighting code blocks.
// It is used by backends such as Chroma, highlight.js, or Prism.
type Highlighter interface {
	// Highlight takes source code and a language identifier, returning
	// syntax-highlighted HTML. If the language is not supported or highlighting
	// fails, implementations should return the original code wrapped in
	// appropriate HTML (e.g., <pre><code>...</code></pre>).
	//
	// Takes code (string) which is the source code to highlight.
	// Takes language (string) which is the language identifier (e.g., "go",
	// "piko", or a JavaScript identifier such as "js").
	//
	// Returns string which is the highlighted HTML ready for rendering.
	Highlight(code, language string) string
}
