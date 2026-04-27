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

import (
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// DynamicRenderRequest contains the parameters for a full dynamic render
// request. This combines code generation, interpretation, and HTML rendering
// into a single operation.
type DynamicRenderRequest struct {
	// Sources maps file paths to their PK template contents.
	Sources map[string]string `json:"sources"`

	// Props contains optional properties to pass to the template.
	Props map[string]any `json:"props,omitempty"`

	// ModuleName is the Go module name for the generated code.
	ModuleName string `json:"moduleName"`

	// RequestURL is the URL being rendered, used to determine which page
	// template to render and for request context.
	RequestURL string `json:"requestURL,omitempty"`
}

// DynamicRenderResponse contains the results of a dynamic render operation.
//
// Consumers compose a complete HTML document from these fields. HTML/CSS/JS
// are returned strictly separately so consumers can re-route imports
// (importmap rewrites, blob URLs, etc.) without parsing emitted markup.
type DynamicRenderResponse struct {
	// Error contains the error message if Success is false.
	Error string `json:"error,omitempty"`

	// HTML contains the rendered AST body markup. When the dynamic-render
	// pipeline composes its own document wrapper, only the body content
	// is returned here so consumers wrap as needed.
	HTML string `json:"html,omitempty"`

	// CSS contains the page's aggregated style block (page + all
	// transitively-rendered partials and components).
	CSS string `json:"css,omitempty"`

	// Scripts contains compiled client-side JavaScript modules.
	//
	// Each entry is a ScriptArtefact whose Path is the artefact ID
	// without URL prefix; the consumer wraps each Content in a Blob URL
	// (or fetches it via its own asset route) and exposes the mapping
	// via a <script type="importmap"> so the modules' absolute import
	// statements resolve. Modules cover the rendered page and any
	// partials or components it transitively references.
	Scripts []ScriptArtefact `json:"scripts,omitempty"`

	// RuntimeImports lists the framework-runtime URLs that compiled
	// component JavaScript imports. Consumers must make these URLs
	// resolvable inside the rendering context (typically by fetching
	// the framework bundles from the parent server, blob-wrapping them,
	// and adding entries to the same importmap used for Scripts).
	RuntimeImports []string `json:"runtimeImports,omitempty"`

	// Diagnostics contains any warnings or errors encountered during render.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Success indicates whether the render completed successfully.
	Success bool `json:"success"`
}

// InterpretRequest contains the parameters for interpreting generated Go code.
type InterpretRequest struct {
	// Dependencies maps package paths to their generated Go code.
	// These are additional packages needed by the main code.
	Dependencies map[string]string `json:"dependencies,omitempty"`

	// Props contains optional properties to pass to the template.
	Props map[string]any `json:"props,omitempty"`

	// GeneratedCode is the Go source code to interpret.
	GeneratedCode string `json:"generatedCode"`

	// PackagePath is the import path for the generated code.
	PackagePath string `json:"packagePath"`

	// RequestURL is the URL being rendered, used for request context.
	RequestURL string `json:"requestURL,omitempty"`
}

// InterpretResponse contains the results of code interpretation.
type InterpretResponse struct {
	// AST contains the template AST produced by calling BuildAST.
	AST *ast_domain.TemplateAST `json:"ast,omitempty"`

	// Metadata contains template metadata from the BuildAST call.
	Metadata *templater_dto.InternalMetadata `json:"metadata,omitempty"`

	// Error contains the error message if Success is false.
	Error string `json:"error,omitempty"`

	// Diagnostics contains any warnings or errors from interpretation.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Success indicates whether interpretation completed successfully.
	Success bool `json:"success"`
}

// RenderFromASTRequest contains the parameters for rendering a pre-built AST.
type RenderFromASTRequest struct {
	// AST is the template AST to render.
	AST *ast_domain.TemplateAST `json:"ast"`

	// Metadata contains template metadata such as title and description.
	Metadata *templater_dto.InternalMetadata `json:"metadata,omitempty"`

	// CSS contains any CSS styles to include in the output.
	CSS string `json:"css,omitempty"`
}

// RenderFromASTResponse contains the results of AST rendering.
type RenderFromASTResponse struct {
	// Error contains the error message if Success is false.
	Error string `json:"error,omitempty"`

	// HTML contains the rendered HTML output.
	HTML string `json:"html,omitempty"`

	// CSS contains the CSS styles included in the render.
	CSS string `json:"css,omitempty"`

	// Success indicates whether rendering completed successfully.
	Success bool `json:"success"`
}
