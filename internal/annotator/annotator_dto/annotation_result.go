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

package annotator_dto

import (
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// AnnotationResult holds the output of template annotation, including the
// annotated AST, extracted styles, asset references, and analysis metadata
// used by the LSP for context-aware intelligence features.
type AnnotationResult struct {
	// AnalysisMap links each TemplateNode to its active AnalysisContext.
	//
	// It enables the LSP to provide context-aware intelligence features
	// (hover, completion, definition) by accessing the symbol table and
	// type information at each template location. This field uses any to
	// avoid a circular dependency; the LSP type-asserts it to
	// AnalysisContext from annotator_domain.
	AnalysisMap any

	// EntryPointStyleBlocks contains the style blocks from the entry
	// point component being analysed (the file being edited), with
	// their original position information for LSP colour picker
	// ranges.
	//
	// Only populated for the entry point component, not for imported
	// partials.
	EntryPointStyleBlocks any

	// AnnotatedAST is the parsed template with type and code generation
	// annotations applied.
	AnnotatedAST *ast_domain.TemplateAST

	// VirtualModule holds the complete Go module structure. It provides access to
	// the component graph and hash mappings needed for dependency lookups.
	VirtualModule *VirtualModule

	// StyleBlock is the combined CSS content from the component tree.
	StyleBlock string

	// ClientScript contains the client-side JavaScript/TypeScript
	// code extracted from the PK file's <script> block (without
	// type="application/go"), transpiled and served separately for
	// client-side interactivity.
	//
	// Empty if no client script block is present.
	ClientScript string

	// AssetRefs holds the asset references found in the template.
	AssetRefs []templater_dto.AssetRef

	// CustomTags lists the names of custom tags that this component may use.
	CustomTags []string

	// UniqueInvocations holds the partial invocations found during
	// template expansion.
	UniqueInvocations []*PartialInvocation

	// AssetDependencies lists the static assets that this component needs.
	AssetDependencies []*StaticAssetDependency

	// UsesCaptcha indicates the template contains a piko:captcha element
	// and needs captcha provider scripts loaded at runtime.
	UsesCaptcha bool
}
