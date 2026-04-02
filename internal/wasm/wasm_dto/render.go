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

package wasm_dto

// RenderFromSourcesRequest contains the input for rendering HTML from
// in-memory sources via WASM.
type RenderFromSourcesRequest struct {
	// Sources maps file paths to their contents. The paths should be relative
	// to the project root (e.g., "pages/main.pk", "partials/header.pk").
	Sources map[string]string `json:"sources"`

	// ModuleName is the Go module name for the generated code
	// (e.g., "example.com/myproject"). Defaults to "playground" if not set.
	ModuleName string `json:"moduleName,omitempty"`

	// EntryPoint is the page to render (e.g., "pages/main.pk").
	// If not specified and there's only one page, that page is used.
	EntryPoint string `json:"entryPoint,omitempty"`
}

// RenderFromSourcesResponse contains the results of HTML rendering via WASM.
type RenderFromSourcesResponse struct {
	// Error contains the error message when Success is false.
	Error string `json:"error,omitempty"`

	// HTML is the rendered HTML output.
	HTML string `json:"html,omitempty"`

	// CSS contains the aggregated CSS styles for the rendered page.
	CSS string `json:"css,omitempty"`

	// Diagnostics contains any warnings or errors from rendering.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Success indicates whether rendering completed without errors.
	Success bool `json:"success"`

	// IsStaticOnly indicates whether the template was rendered in static-only
	// mode (no Go code execution). When true, dynamic expressions and handlers
	// are not evaluated.
	IsStaticOnly bool `json:"isStaticOnly"`
}
