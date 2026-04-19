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

// AnalyseResponse holds the results from analysing Go source code.
type AnalyseResponse struct {
	// Error contains the error message when Success is false; empty otherwise.
	Error string `json:"error,omitempty"`

	// Types contains the type definitions found in the package.
	Types []TypeInfo `json:"types,omitempty"`

	// Functions holds the function details found in the analysed package.
	Functions []FunctionInfo `json:"functions,omitempty"`

	// Imports lists all import statements found in the analysed code.
	Imports []ImportInfo `json:"imports,omitempty"`

	// Diagnostics contains any warnings or errors from the analysis.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Success indicates whether the analysis completed without errors.
	Success bool `json:"success"`
}

// TypeInfo describes a named type found in the source code.
type TypeInfo struct {
	// Name is the name of the type.
	Name string `json:"name"`

	// Kind is the type category (struct, interface, alias, etc.).
	Kind string `json:"kind"`

	// Documentation is the doc comment for the type.
	Documentation string `json:"documentation,omitempty"`

	// Fields contains the struct fields; empty for non-struct types.
	Fields []FieldInfo `json:"fields,omitempty"`

	// Methods holds the methods defined on the type.
	Methods []MethodInfo `json:"methods,omitempty"`

	// Location is where the type is defined in the source code.
	Location Location `json:"location"`

	// IsExported indicates whether the type is exported.
	IsExported bool `json:"isExported"`
}

// FieldInfo describes a struct field.
type FieldInfo struct {
	// Name is the field name.
	Name string `json:"name"`

	// TypeString is the field type expressed as a string.
	TypeString string `json:"typeString"`

	// Tag is the struct tag, if present.
	Tag string `json:"tag,omitempty"`

	// Documentation is the doc comment for the field.
	Documentation string `json:"documentation,omitempty"`

	// IsEmbedded indicates whether the field is embedded in its parent struct.
	IsEmbedded bool `json:"isEmbedded,omitempty"`
}

// MethodInfo holds details about a single method on a type.
type MethodInfo struct {
	// Name is the name of the method.
	Name string `json:"name"`

	// Signature is the method signature, such as "func(ctx context.Context) error".
	Signature string `json:"signature"`

	// Documentation is the doc comment for the method.
	Documentation string `json:"documentation,omitempty"`

	// IsPointerReceiver indicates whether the method has a pointer receiver.
	IsPointerReceiver bool `json:"isPointerReceiver,omitempty"`
}

// FunctionInfo holds details about a function defined at package level.
type FunctionInfo struct {
	// Name is the name of the function.
	Name string `json:"name"`

	// Signature is the full function signature as a string.
	Signature string `json:"signature"`

	// Documentation is the doc comment for the function.
	Documentation string `json:"documentation,omitempty"`

	// Location specifies where the function is defined in the source code.
	Location Location `json:"location"`

	// IsExported indicates whether the function is exported.
	IsExported bool `json:"isExported"`
}

// ImportInfo holds details about a single import statement.
type ImportInfo struct {
	// Path is the import path for the package.
	Path string `json:"path"`

	// Alias is the import alias; empty if none is used.
	Alias string `json:"alias,omitempty"`

	// IsUsed indicates whether the import is used in the code.
	IsUsed bool `json:"isUsed"`
}

// Location represents a position in source code.
type Location struct {
	// FilePath is the path to the file where this location occurs.
	FilePath string `json:"filePath"`

	// Line is the 1-indexed line number; 0 means no real source position.
	Line int `json:"line"`

	// Column is the column number, starting from 1.
	Column int `json:"column"`
}

// Diagnostic represents a problem found during code analysis.
type Diagnostic struct {
	// Severity indicates the importance level: error, warning, info, or hint.
	Severity string `json:"severity"`

	// Message is the text that describes the issue found.
	Message string `json:"message"`

	// Code is an optional identifier for this diagnostic type; empty means none.
	Code string `json:"code,omitempty"`

	// Location specifies the position in the source file where the issue occurs.
	Location Location `json:"location"`
}

// CompletionResponse holds the result of a code completion request.
type CompletionResponse struct {
	// Error holds the error message when Success is false.
	Error string `json:"error,omitempty"`

	// Items holds the list of completion suggestions.
	Items []CompletionItem `json:"items,omitempty"`

	// Success indicates whether completions were generated.
	Success bool `json:"success"`
}

// CompletionItem represents a single suggestion in an autocomplete list.
type CompletionItem struct {
	// Label is the text displayed in the completion list.
	Label string `json:"label"`

	// Kind is the type of completion item (function, type, variable, etc.).
	Kind string `json:"kind"`

	// Detail provides extra information about this item, such as its scope.
	Detail string `json:"detail,omitempty"`

	// Documentation is a detailed description of this item.
	Documentation string `json:"documentation,omitempty"`

	// InsertText is the text to insert when it differs from Label.
	InsertText string `json:"insertText,omitempty"`

	// SortText is used for sorting when it differs from Label.
	SortText string `json:"sortText,omitempty"`
}

// HoverResponse holds the hover data returned when a user hovers over a symbol.
type HoverResponse struct {
	// Range specifies the text span in the source to which the hover applies.
	Range *Range `json:"range,omitempty"`

	// Content is the hover information formatted as Markdown.
	Content string `json:"content,omitempty"`

	// Error contains the error message when Success is false.
	Error string `json:"error,omitempty"`

	// Success indicates whether hover information was found.
	Success bool `json:"success"`
}

// Range represents a span of text in source code, defined by start and end
// positions.
type Range struct {
	// Start is the beginning position of the range.
	Start Position `json:"start"`

	// End is the final position of the range.
	End Position `json:"end"`
}

// Position represents a position in source code.
type Position struct {
	// Line is the 1-indexed line number in the source file.
	Line int `json:"line"`

	// Column is the column number within the line, starting at 1.
	Column int `json:"column"`
}

// ParseTemplateResponse holds the result of parsing a PK template.
type ParseTemplateResponse struct {
	// AST holds a simplified form of the template syntax tree.
	AST *TemplateAST `json:"ast,omitempty"`

	// Error contains the error message when Success is false.
	Error string `json:"error,omitempty"`

	// Diagnostics holds any warnings or errors from parsing.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Success indicates whether parsing succeeded.
	Success bool `json:"success"`
}

// TemplateAST holds the parsed form of a PK template.
type TemplateAST struct {
	// ScriptBlock contains parsed information about the Go script block from
	// the template.
	ScriptBlock *ScriptBlockInfo `json:"scriptBlock,omitempty"`

	// Nodes holds the top-level template nodes.
	Nodes []TemplateNode `json:"nodes"`
}

// TemplateNode represents a single node in the template syntax tree.
type TemplateNode struct {
	// Type is the kind of node (element, text, expression, etc.).
	Type string `json:"type"`

	// Name is the element or component name for element nodes.
	Name string `json:"name,omitempty"`

	// Content is the text for text nodes.
	Content string `json:"content,omitempty"`

	// Attributes holds the HTML element attributes as name-value pairs.
	Attributes map[string]string `json:"attributes,omitempty"`

	// Children holds the nested nodes within this element or fragment.
	Children []TemplateNode `json:"children,omitempty"`

	// Location is the line and column where this node starts in the source.
	Location Location `json:"location"`
}

// ScriptBlockInfo holds details about a template's script block.
type ScriptBlockInfo struct {
	// PropsType is the name of the props type, if one is defined.
	PropsType string `json:"propsType,omitempty"`

	// Types lists the names of types defined in this script block.
	Types []string `json:"types,omitempty"`

	// HasInit indicates whether the file contains an init function.
	HasInit bool `json:"hasInit,omitempty"`
}

// RenderPreviewResponse holds the result of a template preview render.
type RenderPreviewResponse struct {
	// HTML is the rendered HTML output.
	HTML string `json:"html,omitempty"`

	// CSS contains any extracted CSS styles; empty if not used.
	CSS string `json:"css,omitempty"`

	// Error holds the error message when Success is false.
	Error string `json:"error,omitempty"`

	// Diagnostics holds any warnings produced during rendering.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Success indicates whether rendering succeeded.
	Success bool `json:"success"`
}

// ValidateResponse holds the results of a validation check.
type ValidateResponse struct {
	// Diagnostics holds validation errors and warnings.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Valid indicates whether the code is valid.
	Valid bool `json:"valid"`
}

// RuntimeInfo contains information about the WASM runtime.
type RuntimeInfo struct {
	// Version is the Piko server version string.
	Version string `json:"version"`

	// GoVersion is the Go version used to build the binary.
	GoVersion string `json:"goVersion"`

	// StdlibPackages lists the standard library packages that are available.
	StdlibPackages []string `json:"stdlibPackages"`

	// Capabilities lists the features that are available.
	Capabilities []string `json:"capabilities"`
}
