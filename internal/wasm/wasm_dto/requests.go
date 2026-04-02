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

// AnalyseRequest represents a request to analyse Go source code.
// This is the primary entry point for type analysis in the WASM REPL.
type AnalyseRequest struct {
	// Sources maps file paths to their Go source code content.
	// At minimum, include a main.go file.
	Sources map[string]string `json:"sources"`

	// ModuleName is the Go module name for the user's code.
	// Defaults to "playground" if not set.
	ModuleName string `json:"moduleName,omitempty"`
}

// CompletionRequest holds the data needed to request code completions at a
// cursor position.
type CompletionRequest struct {
	// Source is the Go source code to analyse for completions.
	Source string `json:"source"`

	// FilePath is the virtual file path for multi-file scenarios.
	FilePath string `json:"filePath,omitempty"`

	// ModuleName is the Go module name that provides context for completions.
	ModuleName string `json:"moduleName,omitempty"`

	// Line is the 1-indexed line number for the cursor position.
	Line int `json:"line"`

	// Column is the 1-indexed column number for the cursor position.
	Column int `json:"column"`
}

// HoverRequest holds the data needed to get hover information at a position
// in Go source code.
type HoverRequest struct {
	// Source is the Go source code to analyse.
	Source string `json:"source"`

	// FilePath is the path to the file being queried.
	FilePath string `json:"filePath,omitempty"`

	// ModuleName is the name of the Go module for extra context.
	ModuleName string `json:"moduleName,omitempty"`

	// Line is the 1-indexed line number in the source file.
	Line int `json:"line"`

	// Column is the 1-indexed column number within the line.
	Column int `json:"column"`
}

// ParseTemplateRequest represents a request to parse a PK template.
type ParseTemplateRequest struct {
	// Template is the PK template content to parse.
	Template string `json:"template"`

	// Script is the Go script block content. Optional; can be embedded in the
	// template.
	Script string `json:"script,omitempty"`

	// ModuleName is the Go module name used for context.
	ModuleName string `json:"moduleName,omitempty"`
}

// RenderPreviewRequest holds the data needed to render a template preview.
type RenderPreviewRequest struct {
	// Template is the primary key template content to render.
	Template string `json:"template"`

	// Script is the Go script content to render.
	Script string `json:"script,omitempty"`

	// PropsJSON is the JSON-encoded props to pass to the template.
	PropsJSON string `json:"propsJson,omitempty"`

	// ModuleName is the Go module name that gives context for the preview.
	ModuleName string `json:"moduleName,omitempty"`
}

// ValidateRequest represents a request to validate code without full analysis.
type ValidateRequest struct {
	// Source is the Go source code to check.
	Source string `json:"source"`

	// FilePath is the virtual path used in error messages and diagnostics.
	FilePath string `json:"filePath,omitempty"`
}
