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

// GenerateFromSourcesRequest contains the input for generating code from
// in-memory sources via WASM.
type GenerateFromSourcesRequest struct {
	// Sources maps file paths to their contents. The paths should be relative
	// to the project root (e.g., "pages/main.pk", "partials/header.pk").
	Sources map[string]string `json:"sources"`

	// ModuleName is the Go module name for the generated code
	// (e.g., "example.com/myproject").
	ModuleName string `json:"moduleName"`

	// BaseDir is the virtual base directory for path resolution.
	// If empty, defaults to the current directory.
	BaseDir string `json:"baseDir,omitempty"`
}

// GenerateFromSourcesResponse contains the results of code generation via WASM.
type GenerateFromSourcesResponse struct {
	// Manifest contains the project manifest with page and partial metadata.
	Manifest *GeneratedManifest `json:"manifest,omitempty"`

	// Error contains the error message when Success is false.
	Error string `json:"error,omitempty"`

	// Artefacts contains the generated code files.
	Artefacts []GeneratedArtefact `json:"artefacts,omitempty"`

	// Diagnostics contains any warnings or errors from generation.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Success indicates whether generation completed without errors.
	Success bool `json:"success"`
}

// GeneratedArtefact represents a single generated file from the code
// generation process.
type GeneratedArtefact struct {
	// Path is the relative path where the file should be written
	// (e.g., "dist/pages/pages_main_abc123/main.go").
	Path string `json:"path"`

	// Content is the generated file content as a string.
	Content string `json:"content"`

	// Type indicates what kind of artefact this is.
	Type ArtefactType `json:"type"`

	// SourcePath is the original source file path that produced this artefact.
	SourcePath string `json:"sourcePath,omitempty"`
}

// ArtefactType identifies the kind of generated artefact.
type ArtefactType string

const (
	// ArtefactTypePage indicates a generated page component.
	ArtefactTypePage ArtefactType = "page"

	// ArtefactTypePartial indicates a generated partial component.
	ArtefactTypePartial ArtefactType = "partial"

	// ArtefactTypeAction indicates a generated action handler.
	ArtefactTypeAction ArtefactType = "action"

	// ArtefactTypeRegister indicates a generated component register file.
	ArtefactTypeRegister ArtefactType = "register"

	// ArtefactTypeManifest indicates a generated manifest file.
	ArtefactTypeManifest ArtefactType = "manifest"

	// ArtefactTypeJS indicates a generated JavaScript file.
	ArtefactTypeJS ArtefactType = "js"
)

// GeneratedManifest contains metadata about all generated pages and partials.
type GeneratedManifest struct {
	// Pages maps page identifiers to their manifest entries.
	Pages map[string]ManifestPageEntry `json:"pages,omitempty"`

	// Partials maps partial identifiers to their manifest entries.
	Partials map[string]ManifestPartialEntry `json:"partials,omitempty"`
}

// ManifestPageEntry contains metadata for a generated page.
type ManifestPageEntry struct {
	// RoutePatterns maps locale codes to route patterns.
	RoutePatterns map[string]string `json:"routePatterns,omitempty"`

	// CachePolicy contains caching configuration for this page.
	CachePolicy *CachePolicy `json:"cachePolicy,omitempty"`

	// PackagePath is the Go package import path for this page.
	PackagePath string `json:"packagePath"`

	// SourcePath is the original .pk source file path.
	SourcePath string `json:"sourcePath"`

	// StyleBlock contains the aggregated CSS for the page and its partials.
	StyleBlock string `json:"styleBlock,omitempty"`

	// JSArtefactIDs lists the client-side JavaScript artefact IDs for this page.
	JSArtefactIDs []string `json:"jsArtefactIds,omitempty"`

	// HasGetData indicates whether the page has a GetData function.
	HasGetData bool `json:"hasGetData,omitempty"`

	// HasRender indicates whether the page has a Render function.
	HasRender bool `json:"hasRender,omitempty"`
}

// ManifestPartialEntry contains metadata for a generated partial.
type ManifestPartialEntry struct {
	// PackagePath is the Go package import path for this partial.
	PackagePath string `json:"packagePath"`

	// SourcePath is the original .pk source file path.
	SourcePath string `json:"sourcePath"`

	// PropsTypeName is the name of the Props type, if any.
	PropsTypeName string `json:"propsTypeName,omitempty"`

	// HasProps indicates whether the partial accepts props.
	HasProps bool `json:"hasProps,omitempty"`
}

// CachePolicy defines caching behaviour for a page.
type CachePolicy struct {
	// Mode specifies the cache strategy (e.g., "none", "swr", "ttl").
	Mode string `json:"mode"`

	// TTL is the time-to-live in seconds for TTL-based caching.
	TTL int `json:"ttl,omitempty"`

	// StaleWhileRevalidate is the stale-while-revalidate window in seconds.
	StaleWhileRevalidate int `json:"staleWhileRevalidate,omitempty"`
}
