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

package compiler_dto

// CompiledArtefact holds the output of compiling a component.
type CompiledArtefact struct {
	// Files maps output file paths to their compiled content.
	Files map[string]string

	// TagName is the tag label for this artefact.
	TagName string

	// ScaffoldHTML is the generated HTML scaffold template content.
	ScaffoldHTML string

	// SourceIdentifier is the unique identifier of the source component.
	SourceIdentifier string

	// BaseJSPath is the key in Files for the main JavaScript output.
	BaseJSPath string

	// JSDependencies holds the resolved JS import paths that need registry
	// registration. These are imports that used the @/ alias and were changed
	// to served paths.
	JSDependencies []JSDependency

	// Diagnostics carries non-fatal issues surfaced during compilation.
	Diagnostics []CompilationDiagnostic
}

// CompilationDiagnostic is a non-fatal compile-time issue.
//
// The message is suitable for surfacing directly to the playground or
// developer logs and avoids leaking framework error chains.
type CompilationDiagnostic struct {
	// Severity is "error", "warning", or "info".
	Severity string

	// Message is the human-readable description.
	Message string

	// SourceIdentifier echoes CompiledArtefact.SourceIdentifier so
	// downstream filters can route the diagnostic.
	SourceIdentifier string
}

// JSDependency represents a JavaScript file that a component imports.
// The file needs to be registered to the artefact registry for serving.
type JSDependency struct {
	// OriginalPath is the path as written in the source file
	// (e.g. "@/lib/utils.js").
	OriginalPath string `json:"originalPath"`

	// ResolvedPath is the module-qualified path (for example a GitHub-hosted
	// module path with a "/lib/utils.js" suffix).
	ResolvedPath string `json:"resolvedPath"`

	// ServedPath is the URL path for serving (for example a "/_piko/assets/"
	// prefix followed by a GitHub-hosted module path and a "/lib/utils.js"
	// suffix).
	ServedPath string `json:"servedPath"`
}
