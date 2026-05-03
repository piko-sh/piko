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

package generator_dto

import (
	"piko.sh/piko/internal/annotator/annotator_dto"
)

// VirtualPageInstanceData represents a single virtual page instance produced
// by collection-directive expansion. It is used when generating code for
// collection pages that share a single compiled package; the emitter gates
// the collection-loading prelude on len(request.VirtualInstances) > 0.
type VirtualPageInstanceData struct {
	// InitialProps holds the properties for this instance.
	InitialProps map[string]any

	// Slug identifies the collection item backing this instance.
	// Carried for tooling and direct lookups; at runtime the effective slug
	// comes from the request's path parameter rather than this value.
	Slug string
}

// ConvertVirtualInstances translates annotator-side virtual page instances
// into the emitter-side data type.
//
// Callers that skip this conversion will silently produce BuildAST functions
// that never populate r's CollectionData, leaving piko.GetData[T] to return
// zero values.
//
// Takes instances ([]annotator_dto.VirtualPageInstance) which is the input
// slice from the annotator stage.
//
// Returns []VirtualPageInstanceData which mirrors instances, or nil when no
// instances exist so the emitter's gate stays intact for non-collection
// pages.
func ConvertVirtualInstances(instances []annotator_dto.VirtualPageInstance) []VirtualPageInstanceData {
	if len(instances) == 0 {
		return nil
	}
	result := make([]VirtualPageInstanceData, len(instances))
	for index, instance := range instances {
		result[index] = VirtualPageInstanceData{
			Slug:         instance.Slug,
			InitialProps: instance.InitialProps,
		}
	}
	return result
}

// GenerateRequest encapsulates all the necessary information to process and
// generate a single Piko component file. It is created by an orchestrator
// (like the GeneratorService or a CLI driver) and consumed by the CodeEmitter.
type GenerateRequest struct {
	// SourcePath is the path to the source file for code generation.
	SourcePath string

	// OutputPath is the suggested file path for the generated output.
	OutputPath string

	// PackagePrefix is the package path prefix that filters which packages
	// are processed.
	PackagePrefix string

	// PackageName is the Go package name for the generated file.
	PackageName string

	// BaseDir is the root folder used to work out relative paths.
	BaseDir string

	// HashedName is the hashed component name used as a lookup key in the
	// component registry.
	HashedName string

	// CanonicalGoPackagePath is the fully qualified Go import path for the package.
	CanonicalGoPackagePath string

	// CollectionName specifies the collection name for virtual pages, used by
	// the generated BuildAST to call GetStaticCollectionItem; empty for
	// non-virtual pages.
	CollectionName string

	// CollectionParamName is the URL parameter name (from `p-param`, defaulting
	// to "slug") that the generated BuildAST reads to look up the collection
	// item for the current request, e.g. for `pages/blog/{slug}.pk` with the
	// default param BuildAST reads `r.PathParam("slug")` and passes the value
	// to GetStaticCollectionItem.
	CollectionParamName string

	// ModuleName is the Go module name from go.mod (e.g., a GitHub-hosted module path).
	// Used for @/ alias resolution in dynamic src attributes at runtime.
	ModuleName string

	// VirtualInstances holds virtual page instances for collection pages.
	// Each instance has its own route and data; when non-empty, the generator
	// creates a route-based data registry to serve all instances.
	VirtualInstances []VirtualPageInstanceData

	// IsPage indicates whether the source file is a page entry point.
	IsPage bool

	// VerifyGeneratedCode controls whether the emitter parses the generated Go code
	// as a sanity check. Disabling this provides ~20% faster builds but skips
	// syntactic validation of the output.
	VerifyGeneratedCode bool

	// IsEmail indicates whether this is an email template.
	// Email templates are never prerendered regardless of global settings,
	// as they require the full AST for CSS inlining and PML transformations.
	IsEmail bool

	// IsPdf indicates whether this is a PDF template.
	// PDF templates are never prerendered, as they require the full AST for
	// layout computation and PDF painting.
	IsPdf bool

	// EnablePrerendering controls static HTML prerendering; when false, nodes use
	// AST fallback rendering. Automatically set to false for email and PDF templates.
	EnablePrerendering bool

	// EnableStaticHoisting controls whether static nodes are hoisted to package-level
	// variables. When false, all nodes are built dynamically at render time.
	EnableStaticHoisting bool

	// StripHTMLComments controls whether HTML comments are omitted from output.
	StripHTMLComments bool

	// EnableDwarfLineDirectives controls whether the code generator emits valid
	// DWARF //line directives (no space) or plain comments (// line, with space).
	// When true, debuggers like Delve can map breakpoints back to .pk files.
	EnableDwarfLineDirectives bool
}
