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
	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/sfcparser"
)

// ComponentGraph is the output of the GraphBuilder stage.
//
// It represents the complete, interconnected graph of all parsed .pk
// components in their raw, Piko-native state, before any Go module
// virtualisation. It is the authoritative, in-memory representation of the
// Piko project structure.
type ComponentGraph struct {
	// Components maps hashed file paths to their parsed representations.
	Components map[string]*ParsedComponent

	// AllSourceContents maps absolute file paths to their source content.
	// Used to create detailed diagnostics that include the original source code.
	AllSourceContents map[string][]byte

	// PathToHashedName maps absolute file paths to stable hashed names.
	// For example: "/path/to/project/components/card.pk" maps to "card_abc123".
	PathToHashedName map[string]string

	// HashedNameToPath maps a stable hashed name back to its original absolute
	// file path. It aids debugging and generating error messages that refer to
	// the original file (e.g., "card_abc123" -> "/path/to/card.pk").
	HashedNameToPath map[string]string
}

// ParsedComponent is the direct result of parsing a single .pk file. It
// contains only information that can be derived directly from that one file's
// content and location, without knowledge of other components or the wider
// project.
type ParsedComponent struct {
	// Script holds the parsed script data; nil if no script is present.
	Script *ParsedScript

	// LocalTranslations holds the translation strings defined in this component.
	LocalTranslations i18n_domain.Translations

	// Template is the parsed template AST; nil if this is not a template file.
	Template *ast_domain.TemplateAST

	// VisibilityOverride holds the explicit visibility directive from the
	// template tag, where nil means use the default for the component type
	// and non-nil overrides it.
	VisibilityOverride *bool

	// ClientScript contains the raw JavaScript or TypeScript code from the PK
	// file's script block without type="application/go". Empty if absent.
	ClientScript string

	// SourcePath is the absolute path to the .pk file. For external modules,
	// this may point to a location in GOMODCACHE.
	SourcePath string

	// ModuleImportPath is the full import path for this component
	// (e.g. "github.com/ui/lib/button.pk").
	ModuleImportPath string

	// CollectionName is the name from the p-collection attribute.
	CollectionName string

	// CollectionProvider is the name of the data provider from the p-provider
	// attribute. Defaults to "markdown".
	CollectionProvider string

	// ComponentType is the kind of component: "page", "email", "partial", or
	// "component".
	ComponentType string

	// CollectionParamName is the URL parameter name from p-param
	// attribute (defaults to "slug"), used by collection providers to
	// extract the chi URL parameter for content lookup.
	CollectionParamName string

	// ContentModulePath is the resolved Go module import path for content
	// sourcing. Set when the p-collection-source attribute references an import
	// alias, this allows collection providers to fetch markdown content from
	// external Go modules rather than the local project's content directory.
	ContentModulePath string

	// StyleBlocks holds the parsed style sections from the component.
	StyleBlocks []sfcparser.Style

	// PikoImports lists the Piko framework imports declared in this component.
	PikoImports []PikoImport

	// HasCollection indicates whether this component has a p-collection attribute
	// on its template element.
	HasCollection bool

	// IsExternal indicates whether the component is from an external Go module
	// (located in GOMODCACHE) rather than the current project.
	IsExternal bool
}

// ParsedScript contains the pre-analysed metadata extracted from a component's
// Go <script> block. Caching this information here prevents later stages from
// needing to reparse or re-inspect the Go AST for common details.
type ParsedScript struct {
	// PropsTypeExpression is the AST expression for the props type.
	// Nil means no props.
	PropsTypeExpression goast.Expr

	// RenderReturnTypeExpression is the AST expression for the return type of
	// the Render function; nil if no Render function is defined.
	RenderReturnTypeExpression goast.Expr

	// AST is the parsed Go syntax tree for the script block.
	AST *goast.File

	// Fset is the file set that holds position data for parsed files.
	Fset *token.FileSet

	// SupportedLocalesFuncName is the name of the function that returns
	// supported locales.
	SupportedLocalesFuncName string

	// GoPackageName is the Go package name extracted from the script;
	// defaults to "piko_default" if not specified.
	GoPackageName string

	// MiddlewaresFuncName holds the name of the function that returns middleware.
	MiddlewaresFuncName string

	// CachePolicyFuncName is the name of the cache policy function for this
	// script.
	CachePolicyFuncName string

	// ProvisionalGoPackagePath is the expected Go package path before parsing.
	ProvisionalGoPackagePath string

	// AuthPolicyFuncName is the name of the auth policy function for this
	// script.
	AuthPolicyFuncName string

	// PreviewFuncName is the name of the Preview convention function for
	// dev-mode component previewing.
	PreviewFuncName string

	// ScriptStartLocation is where the script content begins in the source file.
	ScriptStartLocation ast_domain.Location

	// HasMiddleware indicates whether the script defines a middlewares function.
	HasMiddleware bool

	// HasCachePolicy indicates whether the script defines a cache policy function.
	HasCachePolicy bool

	// HasSupportedLocales indicates whether the script defines a SupportedLocales
	// function for locale-aware routing.
	HasSupportedLocales bool

	// HasAuthPolicy indicates whether the script defines an AuthPolicy function
	// for page-level authentication requirements.
	HasAuthPolicy bool

	// HasPreview indicates whether the script defines a Preview function
	// for dev-mode component previewing.
	HasPreview bool
}

// PikoImport represents an `import "..."` statement found in a <script> block
// that points to another .pk file, forming an edge in the ComponentGraph.
type PikoImport struct {
	// Alias is the local name for the import (e.g., "card" in
	// `import card "..."`). It can be empty for a standard import or "_" for a
	// side effect import.
	Alias string

	// Path is the import path as written, before it is resolved to a full path.
	Path string

	// Location is the position of the import statement in the source file.
	Location ast_domain.Location
}
