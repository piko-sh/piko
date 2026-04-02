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

package driven_code_emitter_go_literal

import (
	goast "go/ast"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

// EmitterConfig holds settings for code generation. It is created once per
// EmitCode call and never changed, with field ordering set for memory
// alignment.
type EmitterConfig struct {
	// SourcePathHasClientScript maps source file paths to whether they contain
	// client-side scripts. When a path is present, its boolean value is returned
	// directly; when absent, the script detection falls back to other methods.
	SourcePathHasClientScript map[string]bool

	// CanonicalGoPackagePath is the full import path of the package being generated.
	CanonicalGoPackagePath string

	// BaseDir is the root directory used to compute relative file paths.
	BaseDir string

	// PackageName is the name of the package being documented.
	PackageName string

	// SourcePath is the path to the source file being processed.
	SourcePath string

	// HashedName is the component hash for partial identification; empty disables it.
	HashedName string

	// ModuleName is the Go module name used to resolve import paths.
	ModuleName string

	// VirtualInstances holds virtual page instance data used in code generation.
	VirtualInstances []generator_dto.VirtualPageInstanceData

	// IsPage indicates whether the node is a full page rather than a partial.
	IsPage bool

	// HasClientScript indicates whether the template has client-side scripts.
	HasClientScript bool

	// EnablePrerendering controls whether static nodes are prerendered to HTML bytes.
	// When false, all nodes use AST fallback rendering.
	EnablePrerendering bool

	// EnableStaticHoisting controls whether static nodes are hoisted to package-level
	// variables. When false, all nodes are built dynamically at render time.
	EnableStaticHoisting bool

	// StripHTMLComments controls whether comment nodes are omitted from output.
	StripHTMLComments bool

	// EnableDwarfLineDirectives controls whether //line directives are valid
	// DWARF directives (true) or plain comments with a space (false).
	EnableDwarfLineDirectives bool
}

// LoopIterableInfo holds information about an extracted p-for collection
// variable. This enables accurate child slice capacity calculation and avoids
// double-evaluation of collection expressions (which could be method calls
// with side effects).
type LoopIterableInfo struct {
	// CollectionExpression is the Go AST expression for the collection being iterated.
	CollectionExpression goast.Expr

	// CollectionAnn is the type annotation for the collection, used for
	// nil-check decisions and map type detection.
	CollectionAnn *ast_domain.GoGeneratorAnnotation

	// VarName is the generated Go variable name for this loop iterable
	// (e.g., "loopIter_1").
	VarName string

	// IsNillable indicates whether the collection needs a nil check before iteration.
	IsNillable bool
}

// EmitterContext holds state that changes during code generation.
// Fields are ordered for better memory alignment.
type EmitterContext struct {
	// requiredImports maps package paths to their aliases for generated code.
	// The key is the full package path; the value is the alias or empty if none.
	requiredImports map[string]string

	// usedAliases tracks import aliases in use to detect name conflicts.
	// Key: alias name, Value: canonical package path using this alias.
	usedAliases map[string]string

	// customTagsVarName holds the name of the CustomTags
	// variable at package level; set when static variables are built.
	customTagsVarName string

	// fetcherDecls holds the generated functions that fetch collections.
	fetcherDecls []goast.Decl

	// userCodeLineDirectives collects //line directives for user-authored
	// declarations that need to be injected during post-processing.
	userCodeLineDirectives []userCodeLineDirective

	// tempVarCtr is a counter for generating unique temporary variable names.
	tempVarCtr int64

	// staticVarCtr is an atomic counter used to generate unique static variable names.
	staticVarCtr int64

	// staticAttrVarCtr is an atomic counter for creating unique static attribute
	// slice variable names.
	staticAttrVarCtr int64

	// fetcherCtr counts fetcher functions to generate unique names.
	fetcherCtr int64

	// loopIterCtr is an atomic counter for creating unique loop iterator
	// variable names.
	loopIterCtr int64

	// aliasCtr counts alias suffixes for creating unique names when imports clash.
	aliasCtr int
}

// NewEmitterContext creates a new context for a code generation operation.
//
// Returns *EmitterContext which has empty import and alias maps with all
// counters set to zero.
func NewEmitterContext() *EmitterContext {
	return &EmitterContext{
		requiredImports:   make(map[string]string),
		usedAliases:       make(map[string]string),
		fetcherDecls:      make([]goast.Decl, 0),
		tempVarCtr:        0,
		staticVarCtr:      0,
		staticAttrVarCtr:  0,
		fetcherCtr:        0,
		customTagsVarName: "",
		aliasCtr:          0,
		loopIterCtr:       0,
	}
}
