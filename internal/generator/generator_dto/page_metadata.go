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
	"go/ast"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// PageMetadata is a flattened, serialisable representation of a compiled
// component. It contains all information required by the manifest emitter to
// create the final project manifest, serving as the data contract between the
// main generator service and the manifest emission adapter.
type PageMetadata struct {
	// PropsTypeExpression is the AST expression for the component's props type.
	PropsTypeExpression ast.Expr

	// LocalTranslations holds translation data for this page only.
	LocalTranslations i18n_domain.Translations

	// CachePolicyFuncName is the name of the function that sets the cache policy.
	CachePolicyFuncName string

	// PackagePath is the full import path of the package.
	PackagePath string

	// OriginalFsPathForDisplay is the original file system path
	// for display purposes.
	OriginalFsPathForDisplay string

	// RoutePatterns maps locale codes to their matching route patterns.
	RoutePatterns map[string]string

	// I18nStrategy specifies the URL strategy for multiple languages; valid values
	// are "prefix", "prefix_except_default", "query-only", or "disabled".
	I18nStrategy string

	// RenderFuncName is the name of the function that renders this page.
	RenderFuncName string

	// ASTFuncName is the function name as it appears in the abstract syntax tree.
	ASTFuncName string

	// StyleBlock contains inline CSS styles for the page.
	StyleBlock string

	// PackageAlias is the import alias used for this package; empty means no alias.
	PackageAlias string

	// MiddlewaresFuncName is the name of the function that provides middleware.
	MiddlewaresFuncName string

	// SupportedLocalesFuncName is the name of the function
	// returning supported locales.
	SupportedLocalesFuncName string

	// AuthPolicyFuncName is the name of the function that declares auth
	// requirements for this page.
	AuthPolicyFuncName string

	// OriginalPath is the file path before any processing.
	OriginalPath string

	// AssetRefs lists the external assets used by this page.
	AssetRefs []templater_dto.AssetRef

	// CustomTags lists the custom tags that may appear in documentation.
	CustomTags []string

	// HasSupportedLocales indicates whether the page has translations available.
	HasSupportedLocales bool

	// HasCachePolicy indicates whether the page sets a cache policy.
	HasCachePolicy bool

	// HasMiddleware indicates whether the page uses middleware.
	HasMiddleware bool

	// HasAuthPolicy indicates whether the page declares auth requirements.
	HasAuthPolicy bool

	// IsPage indicates whether this metadata is for a page rather than a section.
	IsPage bool
}
