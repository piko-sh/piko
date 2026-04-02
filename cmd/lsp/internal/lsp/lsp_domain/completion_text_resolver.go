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
	"go/ast"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/logger/logger_domain"
)

// resolveExpressionFromText resolves a dot-separated expression by walking
// through each segment. This is used when the AST is broken (parse errors) but
// we still want autocomplete.
//
// Example: "state.Domain.CustomerItem" resolves to CustomerDto type.
//
// Algorithm:
// 1. Split by dots: ["state", "Domain", "CustomerItem"]
// 2. Resolve "state" -> Look in scope or use special handling -> type: Response
// 3. Resolve "Domain" -> Field on Response -> type: CustomerModal
// 4. Resolve "CustomerItem" -> Field on CustomerModal -> type: CustomerDto
// 5. Return: CustomerDto type info for autocomplete
//
// Takes expressionText (string) which is the dot-separated expression to resolve.
// Takes position (protocol.Position) which specifies where in the document to
// resolve from.
//
// Returns *ast_domain.ResolvedTypeInfo which is the resolved type information,
// or nil if resolution fails at any segment.
func (d *document) resolveExpressionFromText(ctx context.Context, expressionText string, position protocol.Position) *ast_domain.ResolvedTypeInfo {
	_, l := logger_domain.From(ctx, log)

	if expressionText == "" {
		return nil
	}

	l.Trace("resolveExpressionFromText: Starting text-based resolution",
		logger_domain.String("expression", expressionText))

	segments := strings.Split(expressionText, ".")

	currentType := d.resolveFirstSegment(ctx, segments[0], position)
	if currentType == nil {
		l.Debug("resolveExpressionFromText: Could not resolve first segment",
			logger_domain.String(keySegment, segments[0]))
		return nil
	}

	l.Trace("resolveExpressionFromText: Resolved first segment",
		logger_domain.String(keySegment, segments[0]),
		logger_domain.String("type", goastutil.ASTToTypeString(currentType.TypeExpression, currentType.PackageAlias)))

	for i := 1; i < len(segments); i++ {
		fieldName := segments[i]

		fieldType := d.resolveFieldOnType(ctx, currentType, fieldName)
		if fieldType == nil {
			l.Trace("resolveExpressionFromText: Could not resolve segment as field",
				logger_domain.String(keySegment, fieldName),
				logger_domain.String("onType", goastutil.ASTToTypeString(currentType.TypeExpression, currentType.PackageAlias)))
			return nil
		}

		currentType = fieldType
		l.Trace("resolveExpressionFromText: Resolved segment",
			logger_domain.String(keySegment, fieldName),
			logger_domain.String("type", goastutil.ASTToTypeString(currentType.TypeExpression, currentType.PackageAlias)))
	}

	l.Trace("resolveExpressionFromText: Successfully resolved complete expression",
		logger_domain.String("finalType", goastutil.ASTToTypeString(currentType.TypeExpression, currentType.PackageAlias)))

	return currentType
}

// resolveFirstSegment resolves the first identifier in an expression.
// Handles special cases like "state" and "props", then falls back to scope
// lookup.
//
// Takes identifier (string) which is the first segment of the expression to
// resolve.
// Takes position (protocol.Position) which specifies the source location for scope
// lookup.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the resolved type, or
// nil if the identifier cannot be resolved.
func (d *document) resolveFirstSegment(ctx context.Context, identifier string, position protocol.Position) *ast_domain.ResolvedTypeInfo {
	if identifier == "state" {
		return d.resolveStateType()
	}

	if identifier == "props" {
		return d.resolvePropsType()
	}

	return d.resolveIdentifierFromScope(ctx, identifier, position)
}

// resolveIdentifierFromScope looks up an identifier in the symbol table at
// the given position. This is used for text-based completion when the AST has
// parse errors.
//
// Takes identifier (string) which is the name to look up.
// Takes position (protocol.Position) which specifies the location in the document.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the resolved type, or
// nil if the identifier cannot be found.
func (d *document) resolveIdentifierFromScope(ctx context.Context, identifier string, position protocol.Position) *ast_domain.ResolvedTypeInfo {
	if d.AnalysisMap == nil || d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return nil
	}

	targetNode := findNodeAtPosition(d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetNode == nil {
		return d.resolveIdentifierFromFallbackContext(ctx, identifier)
	}

	return d.resolveIdentifierFromNode(ctx, identifier, targetNode)
}

// resolveIdentifierFromFallbackContext searches all analysis contexts for an
// identifier. Used when the target node cannot be found at the cursor position.
//
// Takes identifier (string) which is the name to search for across contexts.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the resolved type, or nil
// if not found.
func (d *document) resolveIdentifierFromFallbackContext(ctx context.Context, identifier string) *ast_domain.ResolvedTypeInfo {
	_, l := logger_domain.From(ctx, log)

	for _, actx := range d.AnalysisMap {
		if actx == nil || actx.Symbols == nil {
			continue
		}

		symbol, found := actx.Symbols.Find(identifier)
		if found && symbol.TypeInfo != nil {
			l.Debug("resolveIdentifierFromScope: Found symbol in fallback context",
				logger_domain.String("identifier", identifier))
			return symbol.TypeInfo
		}
	}
	return nil
}

// resolveIdentifierFromNode looks up an identifier in a node's symbol table.
//
// Takes identifier (string) which is the name to find.
// Takes targetNode (*ast_domain.TemplateNode) which provides the analysis
// context for the lookup.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the type information for
// the identifier, or nil if the identifier is not found.
func (d *document) resolveIdentifierFromNode(ctx context.Context, identifier string, targetNode *ast_domain.TemplateNode) *ast_domain.ResolvedTypeInfo {
	_, l := logger_domain.From(ctx, log)

	analysisCtx, exists := d.AnalysisMap[targetNode]
	if !exists || analysisCtx == nil || analysisCtx.Symbols == nil {
		return nil
	}

	symbol, found := analysisCtx.Symbols.Find(identifier)
	if !found || symbol.TypeInfo == nil {
		l.Debug("resolveIdentifierFromScope: Symbol not found in scope",
			logger_domain.String("identifier", identifier))
		return nil
	}

	l.Debug("resolveIdentifierFromScope: Successfully resolved symbol",
		logger_domain.String("identifier", identifier),
		logger_domain.String("type", goastutil.ASTToTypeString(symbol.TypeInfo.TypeExpression, symbol.TypeInfo.PackageAlias)))

	return symbol.TypeInfo
}

// resolveStateType resolves the "state" identifier to the Render return type.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the resolved type, or nil
// when the script block cannot be parsed or the return type cannot be found.
func (d *document) resolveStateType() *ast_domain.ResolvedTypeInfo {
	scriptResult, err := d.parseOriginalScriptBlock()
	if err != nil || scriptResult == nil {
		return nil
	}

	returnTypeName := findRenderReturnType(scriptResult.AST)
	if returnTypeName == "" {
		return nil
	}

	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          ast.NewIdent(returnTypeName),
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// resolvePropsType resolves the "props" identifier to the component's props
// type by extracting it from the Render function's second parameter.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the resolved props type,
// or nil if the props type cannot be determined.
func (d *document) resolvePropsType() *ast_domain.ResolvedTypeInfo {
	scriptResult, err := d.parseOriginalScriptBlock()
	if err != nil || scriptResult == nil {
		return nil
	}

	propsType := findRenderPropsType(scriptResult.AST)
	if propsType == nil {
		return nil
	}

	return propsType
}

// resolveFieldOnType resolves a field name on a given type using the inspector.
//
// Takes baseType (*ast_domain.ResolvedTypeInfo) which is the type to search
// for the field on.
// Takes fieldName (string) which is the name of the field to resolve.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the resolved field type
// information, or nil if the field cannot be found.
func (d *document) resolveFieldOnType(ctx context.Context, baseType *ast_domain.ResolvedTypeInfo, fieldName string) *ast_domain.ResolvedTypeInfo {
	if d.TypeInspector == nil || baseType == nil || baseType.TypeExpression == nil {
		return nil
	}

	var packagePath, filePath string
	for _, ctx := range d.AnalysisMap {
		if ctx != nil {
			packagePath = ctx.CurrentGoFullPackagePath
			filePath = ctx.CurrentGoSourcePath
			break
		}
	}

	if packagePath == "" {
		return nil
	}

	fieldInfo := d.TypeInspector.FindFieldInfo(
		ctx,
		baseType.TypeExpression,
		fieldName,
		packagePath,
		filePath,
	)

	if fieldInfo == nil {
		return nil
	}

	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          fieldInfo.Type,
		PackageAlias:            fieldInfo.PackageAlias,
		CanonicalPackagePath:    fieldInfo.CanonicalPackagePath,
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}
