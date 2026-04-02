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

package lsp_domain

import (
	"context"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// GetMonikers returns unique identifiers for the symbol at the given position.
// These identifiers work across different code repositories and serve as stable
// references for code intelligence features.
//
// Takes position (protocol.Position) which specifies the cursor position to query.
//
// Returns []protocol.Moniker which contains the monikers for the symbol,
// typically one per symbol.
// Returns error when the moniker computation fails.
func (d *document) GetMonikers(ctx context.Context, position protocol.Position) ([]protocol.Moniker, error) {
	_, l := logger_domain.From(ctx, log)

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		l.Debug("GetMonikers: No annotated AST available")
		return []protocol.Moniker{}, nil
	}

	targetExpr, _ := findExpressionAtPosition(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetExpr == nil {
		l.Debug("GetMonikers: No expression found at position")
		return []protocol.Moniker{}, nil
	}

	ann := targetExpr.GetGoAnnotation()
	if ann == nil {
		l.Debug("GetMonikers: Expression has no Go annotation",
			logger_domain.String("expr", targetExpr.String()))
		return []protocol.Moniker{}, nil
	}

	return d.buildMonikerFromAnnotation(ctx, ann, targetExpr)
}

// buildMonikerFromAnnotation constructs a Moniker from a GoGeneratorAnnotation.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type
// and symbol info.
// Takes expression (ast_domain.Expression) which is the expression
// for name extraction.
//
// Returns []protocol.Moniker which contains the constructed moniker.
// Returns error when moniker construction fails.
func (*document) buildMonikerFromAnnotation(
	ctx context.Context,
	ann *ast_domain.GoGeneratorAnnotation,
	expression ast_domain.Expression,
) ([]protocol.Moniker, error) {
	_, l := logger_domain.From(ctx, log)

	symbolName := extractMonikerSymbolName(expression)
	if symbolName == "" {
		l.Debug("GetMonikers: Could not extract symbol name")
		return []protocol.Moniker{}, nil
	}

	packagePath := ""
	if ann.ResolvedType != nil {
		packagePath = ann.ResolvedType.CanonicalPackagePath
	}

	identifier := buildMonikerIdentifier(packagePath, symbolName)

	unique := determineMonikerUniqueness(ann.ResolvedType, symbolName)
	kind := determineMonikerKind(ann.ResolvedType, symbolName)

	l.Debug("GetMonikers: Built moniker",
		logger_domain.String("identifier", identifier),
		logger_domain.String("uniqueness", string(unique)),
		logger_domain.String("kind", string(kind)))

	return []protocol.Moniker{{
		Scheme:     "go",
		Identifier: identifier,
		Unique:     unique,
		Kind:       kind,
	}}, nil
}

// buildMonikerIdentifier constructs the moniker identifier string.
// Format: "package_path#symbol_name" (e.g., "github.com/user/pkg#MyType")
// or just "symbol_name" for local symbols without a package path.
//
// Takes packagePath (string) which is the canonical package path.
// Takes symbolName (string) which is the symbol name.
//
// Returns string which is the formatted moniker identifier.
func buildMonikerIdentifier(packagePath, symbolName string) string {
	if packagePath == "" {
		return symbolName
	}
	return packagePath + "#" + symbolName
}

// determineMonikerUniqueness finds the scope in which a moniker is unique.
//
// Takes resolvedType (*ast_domain.ResolvedTypeInfo) which provides package
// information.
// Takes symbolName (string) which is the symbol name to check for export
// status.
//
// Returns protocol.UniquenessLevel which shows the uniqueness scope: global
// for exported symbols from known packages, project for unexported symbols
// with a package path, or document for local variables.
func determineMonikerUniqueness(resolvedType *ast_domain.ResolvedTypeInfo, symbolName string) protocol.UniquenessLevel {
	if resolvedType != nil && resolvedType.CanonicalPackagePath != "" && isMonikerExportedName(symbolName) {
		return protocol.UniquenessLevelGlobal
	}
	if resolvedType != nil && resolvedType.CanonicalPackagePath != "" {
		return protocol.UniquenessLevelProject
	}
	return protocol.UniquenessLevelDocument
}

// determineMonikerKind determines if the symbol is exported, imported, or
// local.
//
// Takes resolvedType (*ast_domain.ResolvedTypeInfo) which provides export info.
// Takes symbolName (string) which is the symbol name to check export status.
//
// Returns protocol.MonikerKind which indicates the kind of moniker.
func determineMonikerKind(resolvedType *ast_domain.ResolvedTypeInfo, symbolName string) protocol.MonikerKind {
	if resolvedType != nil && resolvedType.IsExportedPackageSymbol {
		return protocol.MonikerKindExport
	}
	if isMonikerExportedName(symbolName) {
		return protocol.MonikerKindExport
	}
	return protocol.MonikerKindLocal
}

// extractMonikerSymbolName gets the symbol name from an expression.
//
// Takes expression (ast_domain.Expression) which is the expression
// to get the name from.
//
// Returns string which is the symbol name, or empty if not found.
func extractMonikerSymbolName(expression ast_domain.Expression) string {
	switch e := expression.(type) {
	case *ast_domain.Identifier:
		return e.Name
	case *ast_domain.MemberExpression:
		if prop, ok := e.Property.(*ast_domain.Identifier); ok {
			return prop.Name
		}
	}
	return ""
}

// isMonikerExportedName checks if a Go identifier is exported.
//
// An identifier is considered exported if it starts with an uppercase letter.
//
// Takes name (string) which is the identifier name to check.
//
// Returns bool which is true if the name is exported.
func isMonikerExportedName(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}
