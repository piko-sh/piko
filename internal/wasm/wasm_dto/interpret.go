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
type DynamicRenderResponse struct {
	// Error contains the error message if Success is false.
	Error string `json:"error,omitempty"`

	// HTML contains the rendered HTML output.
	HTML string `json:"html,omitempty"`

	// CSS contains any CSS extracted from the templates.
	CSS string `json:"css,omitempty"`

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
