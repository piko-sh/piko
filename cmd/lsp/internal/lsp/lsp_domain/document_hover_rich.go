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
	"fmt"
	goast "go/ast"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxFieldsPreview is the maximum number of fields to show in type previews.
	maxFieldsPreview = 10

	// fieldNameWidth is the minimum column width for field names in hover output.
	fieldNameWidth = 20

	// codeBlockEnd is the closing marker for Markdown code blocks.
	codeBlockEnd = "\n```"

	// maxDisplayValueLen is the maximum length for displayed values in hovers.
	maxDisplayValueLen = 50

	// truncatedDisplayLen is the length to truncate display values to before
	// adding "...".
	truncatedDisplayLen = 47

	// logKeyTypeExpr is the log field key for type expressions.
	logKeyTypeExpr = "typeExpr"

	// fieldPadding is used to right-pad field names to fieldNameWidth columns.
	fieldPadding = "                    "
)

// formatHoverContentsEnhanced creates rich hover tooltips with better
// categorization and links. This provides improved DX with clearer symbol
// information and external package links.
//
// Takes expression (ast_domain.Expression) which is the expression to hover over.
// Takes memberContext (*ast_domain.MemberExpression) which is the containing
// MemberExpr when the cursor is on a method property identifier
// (e.g., "String" in "x.String()").
//
// Returns string which is the formatted markdown hover content, or an
// empty string if the expression has no type annotation.
func (d *document) formatHoverContentsEnhanced(ctx context.Context, expression ast_domain.Expression, _ protocol.Position, memberContext *ast_domain.MemberExpression) string {
	if tl, ok := expression.(*ast_domain.TemplateLiteral); ok {
		return d.formatTemplateLiteralHover(tl)
	}

	ann := expression.GetGoAnnotation()
	if ann == nil || ann.ResolvedType == nil {
		return ""
	}

	if identifier, ok := expression.(*ast_domain.Identifier); ok && identifier.Name == "state" {
		return d.getStateTypeHover()
	}

	if ann.ResolvedType.IsSynthetic {
		return d.getSyntheticTypeHover(expression, ann)
	}

	symbolKind, displayName := d.categoriseSymbol(expression, ann)
	typeString := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)

	if symbolKind == "field" {
		return d.formatFieldHover(ctx, ann, displayName, typeString)
	}

	return d.formatNonFieldHover(ctx, expression, ann, symbolKind, displayName, typeString, memberContext)
}

// formatTemplateLiteralHover formats hover content for template literal
// expressions. Template literals are always strings.
//
// Takes tl (*ast_domain.TemplateLiteral) which is the template literal to
// format.
//
// Returns string which is the formatted hover content.
func (*document) formatTemplateLiteralHover(tl *ast_domain.TemplateLiteral) string {
	displayValue := tl.String()
	if len(displayValue) > maxDisplayValueLen {
		displayValue = displayValue[:truncatedDisplayLen] + "..."
	}

	return fmt.Sprintf("```go\n(value) %s: string\n```", displayValue)
}

// formatFieldHover formats hover content for struct field symbols.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the field
// annotation data including optional field tags.
// Takes displayName (string) which specifies the field name to display.
// Takes typeString (string) which specifies the field type as a string.
//
// Returns string which contains the formatted markdown hover content.
func (d *document) formatFieldHover(ctx context.Context, ann *ast_domain.GoGeneratorAnnotation, displayName, typeString string) string {
	var b strings.Builder
	b.WriteString("```go\n")

	if ann.FieldTag != nil && *ann.FieldTag != "" {
		_, _ = fmt.Fprintf(&b, "field %s %s `%s`", displayName, typeString, *ann.FieldTag)
	} else {
		_, _ = fmt.Fprintf(&b, "field %s %s", displayName, typeString)
	}
	b.WriteString("\n```")

	if typePreview := d.getTypePreview(ctx, ann, maxFieldsPreview); typePreview != "" {
		b.WriteString("\n\n```go\n")
		b.WriteString(typePreview)
		b.WriteString(codeBlockEnd)
	}

	d.addPackageLinkWithRule(&b, ann, typeString)
	return b.String()
}

// formatNonFieldHover formats hover content for non-field symbols.
//
// Takes expression (ast_domain.Expression) which is the expression being hovered.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the symbol
// annotation data.
// Takes symbolKind (string) which specifies the kind of symbol being formatted.
// Takes displayName (string) which is the name to display in the hover.
// Takes typeString (string) which is the type signature to show.
// Takes memberContext (*ast_domain.MemberExpression) which is the containing
// MemberExpr
// when hovering over a method property identifier.
//
// Returns string which is the formatted hover content with code blocks.
func (d *document) formatNonFieldHover(
	ctx context.Context, expression ast_domain.Expression, ann *ast_domain.GoGeneratorAnnotation,
	symbolKind, displayName, typeString string, memberContext *ast_domain.MemberExpression,
) string {
	var b strings.Builder
	b.WriteString("```go\n")

	if symbolKind == "function" || symbolKind == "method" {
		if signature := d.getFunctionSignatureForHover(expression, displayName, ann, memberContext); signature != "" {
			typeString = signature
		}
	}

	b.WriteString(formatSymbolDeclaration(symbolKind, displayName, typeString))
	b.WriteString("\n```")

	if typePreview := d.getTypePreviewForAnySymbol(ctx, ann, maxFieldsPreview); typePreview != "" {
		b.WriteString("\n\n```go\n")
		b.WriteString(typePreview)
		b.WriteString(codeBlockEnd)
	}

	d.addPackageLink(&b, ann)
	return b.String()
}

// getSyntheticTypeHover generates hover documentation for synthetic types.
// Synthetic types are JavaScript-only placeholders (like $event) that exist for
// type-checking purposes but do not correspond to real Go types.
//
// Takes expression (ast_domain.Expression) which is the expression being hovered.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the type
// annotation.
//
// Returns string which is the formatted markdown hover content explaining
// the synthetic type and its valid usage context.
func (*document) getSyntheticTypeHover(expression ast_domain.Expression, ann *ast_domain.GoGeneratorAnnotation) string {
	typeString := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
	name := expression.String()

	var b strings.Builder
	b.WriteString("```typescript\n")
	_, _ = fmt.Fprintf(&b, "(synthetic) %s: %s", name, typeString)
	b.WriteString("\n```\n\n")

	switch typeString {
	case "js.Event":
		b.WriteString("**JavaScript Browser Event**\n\n")
		b.WriteString("`$event` passes the native browser event object to your handler function. ")
		b.WriteString("It is only available in client-side event handlers (`p-on:*`).\n\n")
		b.WriteString("**Usage:**\n")
		b.WriteString("- Template: `p-on:click=\"handleClick($event)\"`\n")
		b.WriteString("- Handler: `function handleClick(e) { e.preventDefault(); }`\n\n")
		b.WriteString("**Invalid usage:**\n")
		b.WriteString("- Property access in template: `$event.target`\n")
		b.WriteString("- Arithmetic: `$event + 1`\n")
	case "pk.FormData":
		b.WriteString("**Form Data Handle**\n\n")
		b.WriteString("`$form` passes form data from the closest ancestor `<form>` element to your handler. ")
		b.WriteString("It is only available in client-side event handlers (`p-on:*`).\n\n")
		b.WriteString("**Usage:**\n")
		b.WriteString("- Template: `p-on:submit=\"handleSubmit($form)\"`\n")
		b.WriteString("- Handler: `function handleSubmit(form) { form.get('email'); }`\n\n")
		b.WriteString("**Invalid usage:**\n")
		b.WriteString("- Method calls in template: `$form.get('email')`\n")
		b.WriteString("- Outside form: Using `$form` when not inside a `<form>` element\n")
	default:
		b.WriteString("**Synthetic Type**\n\n")
		b.WriteString("This is a placeholder type that exists only for type-checking purposes. ")
		b.WriteString("It does not correspond to a real Go type and cannot be used in server-side code.")
	}

	return b.String()
}

// categoriseSymbol determines what kind of symbol this is for better hover
// display.
//
// Takes expression (ast_domain.Expression) which is the expression to categorise.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides symbol metadata.
//
// Returns kind (string) which is the symbol category (e.g. "attribute",
// "function", "value").
// Returns displayName (string) which is the human-readable name for display.
func (d *document) categoriseSymbol(expression ast_domain.Expression, ann *ast_domain.GoGeneratorAnnotation) (kind, displayName string) {
	if isAttributeSymbol(ann) {
		return "attribute", ann.Symbol.Name
	}

	displayName = expression.String()

	if d.isFunctionType(ann.ResolvedType) {
		return d.categoriseFunctionSymbol(expression, displayName)
	}

	if ann.Symbol != nil {
		return d.categoriseNamedSymbol(expression, ann, displayName)
	}

	return "value", displayName
}

// categoriseFunctionSymbol sorts a function or method symbol into its type.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
// Takes displayName (string) which is the name to include in the result.
//
// Returns kind (string) which is either "method" or "function".
// Returns name (string) which is the given display name.
func (*document) categoriseFunctionSymbol(expression ast_domain.Expression, displayName string) (kind, name string) {
	if _, isMemberExpr := expression.(*ast_domain.MemberExpression); isMemberExpr {
		return "method", displayName
	}
	return "function", displayName
}

// categoriseNamedSymbol categorises a symbol that has a name.
//
// Takes expression (ast_domain.Expression) which is the expression to categorise.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides symbol metadata.
// Takes displayName (string) which is the name to use in the result.
//
// Returns kind (string) which is the category: "property", "field", or
// "variable".
// Returns name (string) which is the display name passed through unchanged.
func (d *document) categoriseNamedSymbol(expression ast_domain.Expression, ann *ast_domain.GoGeneratorAnnotation, displayName string) (kind, name string) {
	if d.isPropUsage(expression) {
		return "property", displayName
	}

	if isFrameworkIdentifier(ann.Symbol.Name) {
		return "variable", displayName
	}

	if d.isFieldSymbol(ann) {
		return "field", displayName
	}

	return "variable", displayName
}

// isFieldSymbol checks whether a symbol represents a struct field.
//
// Takes ann (*GoGeneratorAnnotation) which provides the symbol annotation to
// check.
//
// Returns bool which is true when the symbol is a field reference.
func (d *document) isFieldSymbol(ann *ast_domain.GoGeneratorAnnotation) bool {
	if ann.Symbol.ReferenceLocation.IsSynthetic() || ann.OriginalSourcePath == nil {
		return false
	}

	defPath := *ann.OriginalSourcePath
	currentPath := d.URI.Filename()

	if defPath != currentPath && !strings.HasSuffix(currentPath, ".pk") {
		return true
	}

	return strings.HasSuffix(currentPath, ".pk") && strings.HasSuffix(defPath, ".go")
}

// getStateTypeHover creates hover text for the special "state" variable.
//
// Returns string which contains the formatted hover text, including the state
// variable type and a preview of the type definition when one is available.
func (d *document) getStateTypeHover() string {
	scriptResult, err := d.parseOriginalScriptBlock()
	if err != nil || scriptResult == nil {
		return "(variable) state: Response"
	}

	returnTypeName := findRenderReturnType(scriptResult.AST)
	if returnTypeName == "" {
		return "(variable) state: Response"
	}

	typeDef := findTypeDefinitionInAST(scriptResult.AST, scriptResult.Fset, returnTypeName)
	if typeDef == nil {
		return fmt.Sprintf("(variable) state: %s", returnTypeName)
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "```go\n(variable) state: %s\n```\n\n", returnTypeName)

	if preview := getLocalTypePreview(scriptResult.AST, scriptResult.Fset, returnTypeName, maxFieldsPreview); preview != "" {
		b.WriteString("```go\n")
		b.WriteString(preview)
		b.WriteString(codeBlockEnd)
	}

	return b.String()
}

// getTypePreviewForAnySymbol generates a type preview for any symbol by
// resolving type information and extracting element types from slices.
// Unlike getTypePreview, this does not require OriginalSourcePath to be set.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which contains the resolved
// type information for the symbol.
// Takes maxFields (int) which limits the number of fields shown in the
// preview.
//
// Returns string which contains the formatted type preview, or empty if the
// type cannot be resolved.
func (d *document) getTypePreviewForAnySymbol(ctx context.Context, ann *ast_domain.GoGeneratorAnnotation, maxFields int) string {
	_, l := logger_domain.From(ctx, log)

	if d.TypeInspector == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return ""
	}

	typePackagePath := ann.ResolvedType.CanonicalPackagePath
	if typePackagePath == "" {
		return ""
	}

	resolutionPackagePath := typePackagePath
	resolutionFilePath := d.getResolvedFilePath(ann)
	if ann.ResolvedType.InitialPackagePath != "" && ann.ResolvedType.InitialFilePath != "" {
		resolutionPackagePath = ann.ResolvedType.InitialPackagePath
		resolutionFilePath = ann.ResolvedType.InitialFilePath
	}

	typeExpr, isSlice := extractElementType(ann.ResolvedType.TypeExpression)

	typeExprString := goastutil.ASTToTypeString(typeExpr, ann.ResolvedType.PackageAlias)
	l.Trace("getTypePreviewForAnySymbol: Attempting type resolution",
		logger_domain.String(logKeyTypeExpr, typeExprString),
		logger_domain.String("typePackagePath", typePackagePath),
		logger_domain.String("resolutionPackagePath", resolutionPackagePath),
		logger_domain.String("resolutionFilePath", resolutionFilePath),
		logger_domain.String("initialPackagePath", ann.ResolvedType.InitialPackagePath),
		logger_domain.String("initialFilePath", ann.ResolvedType.InitialFilePath),
		logger_domain.Bool("isSlice", isSlice),
	)

	typeDTO, _ := d.TypeInspector.ResolveExprToNamedType(typeExpr, resolutionPackagePath, resolutionFilePath)
	if typeDTO == nil {
		typeDTO = resolveTypeViaCanonicalPath(d.TypeInspector, ann.ResolvedType)
	}
	if typeDTO == nil {
		l.Trace("getTypePreviewForAnySymbol: Type resolution returned nil",
			logger_domain.String(logKeyTypeExpr, typeExprString),
		)
		return ""
	}

	if len(typeDTO.Fields) > 0 {
		return buildTypePreviewString(typeDTO, isSlice, maxFields)
	}
	return buildMethodPreviewString(typeDTO, maxFields)
}

// getResolvedFilePath returns the file path for type resolution.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the annotation
// that may contain an original source path.
//
// Returns string which is the original source path from the annotation if set,
// otherwise the filename from the document URI.
func (d *document) getResolvedFilePath(ann *ast_domain.GoGeneratorAnnotation) string {
	if ann.OriginalSourcePath != nil {
		return *ann.OriginalSourcePath
	}
	return d.URI.Filename()
}

// getTypePreview builds a preview of a struct type's fields from the inspector.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the type info
// and source path for resolution.
// Takes maxFields (int) which limits how many fields to show in the preview.
//
// Returns string which contains the formatted struct preview, or an empty
// string when the type cannot be resolved or has no fields.
func (d *document) getTypePreview(ctx context.Context, ann *ast_domain.GoGeneratorAnnotation, maxFields int) string {
	_, l := logger_domain.From(ctx, log)

	if d.TypeInspector == nil || d.AnalysisMap == nil || ann.OriginalSourcePath == nil {
		return ""
	}
	if ann.ResolvedType.CanonicalPackagePath == "" {
		return ""
	}

	resolutionPackagePath, resolutionFilePath := d.getTypeResolutionContext(ann)
	typeExprString := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
	l.Trace("getTypePreview: Attempting type resolution",
		logger_domain.String(logKeyTypeExpr, typeExprString),
		logger_domain.String("typePackagePath", ann.ResolvedType.CanonicalPackagePath),
		logger_domain.String("resolutionPackagePath", resolutionPackagePath),
		logger_domain.String("resolutionFilePath", resolutionFilePath),
		logger_domain.String("initialPackagePath", ann.ResolvedType.InitialPackagePath),
		logger_domain.String("initialFilePath", ann.ResolvedType.InitialFilePath),
	)

	typeDTO, _ := d.TypeInspector.ResolveExprToNamedType(ann.ResolvedType.TypeExpression, resolutionPackagePath, resolutionFilePath)
	if typeDTO == nil {
		typeDTO = resolveTypeViaCanonicalPath(d.TypeInspector, ann.ResolvedType)
	}
	if typeDTO == nil {
		l.Trace("getTypePreview: Type resolution returned nil",
			logger_domain.String(logKeyTypeExpr, typeExprString))
		return ""
	}

	if len(typeDTO.Fields) > 0 {
		return formatInspectorStructPreview(typeDTO, maxFields)
	}
	return buildMethodPreviewString(typeDTO, maxFields)
}

// getTypeResolutionContext determines the package path and file path to use
// for resolving a type expression. It prefers the initial context if available.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the type
// resolution information.
//
// Returns packagePath (string) which is the canonical or initial package path.
// Returns filePath (string) which is the original or initial source file path.
func (*document) getTypeResolutionContext(ann *ast_domain.GoGeneratorAnnotation) (packagePath, filePath string) {
	packagePath = ann.ResolvedType.CanonicalPackagePath
	filePath = *ann.OriginalSourcePath
	if ann.ResolvedType.InitialPackagePath != "" && ann.ResolvedType.InitialFilePath != "" {
		packagePath = ann.ResolvedType.InitialPackagePath
		filePath = ann.ResolvedType.InitialFilePath
	}
	return packagePath, filePath
}

// addPackageLinkWithRule adds a gopls-style package link with a horizontal
// rule and the full type path.
//
// Takes b (*strings.Builder) which receives the formatted output.
// Takes ann (*GoGeneratorAnnotation) which provides the resolved type info.
// Takes typeString (string) which specifies the display text for the link.
func (d *document) addPackageLinkWithRule(b *strings.Builder, ann *ast_domain.GoGeneratorAnnotation, typeString string) {
	if !d.shouldShowPackageLink(ann.ResolvedType.CanonicalPackagePath) {
		return
	}

	b.WriteString("\n\n---\n\n")

	_, _ = fmt.Fprintf(b, "[%s on pkg.go.dev](https://pkg.go.dev/%s)",
		typeString,
		ann.ResolvedType.CanonicalPackagePath)
}

// addPackageLink appends a pkg.go.dev link to the builder for non-field
// hovers.
//
// Takes b (*strings.Builder) which receives the formatted link text.
// Takes ann (*GoGeneratorAnnotation) which provides the resolved type info.
func (d *document) addPackageLink(b *strings.Builder, ann *ast_domain.GoGeneratorAnnotation) {
	if !d.shouldShowPackageLink(ann.ResolvedType.CanonicalPackagePath) {
		return
	}

	b.WriteString("\n\n")
	_, _ = fmt.Fprintf(b, "[`%s` on pkg.go.dev](https://pkg.go.dev/%s)",
		ann.ResolvedType.PackageAlias,
		ann.ResolvedType.CanonicalPackagePath)
}

// shouldShowPackageLink checks if a pkg.go.dev link should be shown for a
// package.
//
// Takes packagePath (string) which is the import path of the package to check.
//
// Returns bool which is true for external packages with a domain in the path.
// Returns false for local project packages, standard library packages, and
// relative paths.
func (d *document) shouldShowPackageLink(packagePath string) bool {
	if packagePath == "" {
		return false
	}

	if d.Resolver != nil {
		moduleName := d.Resolver.GetModuleName()
		if moduleName != "" && strings.HasPrefix(packagePath, moduleName) {
			return false
		}
	}

	if !strings.Contains(packagePath, "/") {
		return false
	}

	if strings.HasPrefix(packagePath, ".") {
		return false
	}

	return true
}

// isFunctionType checks whether the given type info represents a function type.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which holds the resolved type
// to check.
//
// Returns bool which is true if the type is a function type or has the name
// "function".
func (*document) isFunctionType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	if identifier, ok := typeInfo.TypeExpression.(*goast.Ident); ok {
		return identifier.Name == "function"
	}
	_, isFuncType := typeInfo.TypeExpression.(*goast.FuncType)
	return isFuncType
}

// isPropUsage checks whether the expression refers to the props identifier.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if the expression's base identifier is "props".
func (*document) isPropUsage(expression ast_domain.Expression) bool {
	identifier := extractBaseIdentifier(expression)
	return identifier != nil && identifier.Name == "props"
}

// getFunctionSignatureForHover attempts to get a full function signature for
// hover display. It first tries the local script block, then falls back to the
// TypeQuerier for package functions or method lookups.
//
// Takes expression (ast_domain.Expression) which is the expression being hovered.
// Takes functionName (string) which is the name of the function to look up.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides package context.
// Takes memberContext (*ast_domain.MemberExpression) which is the containing
// MemberExpr
// when hovering over a method property identifier.
//
// Returns string which is the full function signature, or empty string if not
// found.
func (d *document) getFunctionSignatureForHover(expression ast_domain.Expression, functionName string, ann *ast_domain.GoGeneratorAnnotation, memberContext *ast_domain.MemberExpression) string {
	if signature := d.getLocalFunctionSignature(functionName); signature != "" {
		return signature
	}

	if memberExpr, ok := expression.(*ast_domain.MemberExpression); ok {
		if signature := d.getMethodSignatureFromInspector(memberExpr, functionName, ann); signature != "" {
			return signature
		}
	} else if memberContext != nil {
		if signature := d.getMethodSignatureFromInspector(memberContext, functionName, ann); signature != "" {
			return signature
		}
	}

	if signature := d.getFunctionSignatureFromInspector(functionName, ann); signature != "" {
		return signature
	}

	return ""
}

// getLocalFunctionSignature looks up a function in the local script block and
// returns its signature string.
//
// Takes functionName (string) which is the name of the function to find.
//
// Returns string which is the function signature, or an empty string if not
// found.
func (d *document) getLocalFunctionSignature(functionName string) string {
	scriptResult, err := d.parseOriginalScriptBlock()
	if err != nil || scriptResult == nil {
		return ""
	}

	for _, declaration := range scriptResult.AST.Decls {
		funcDecl, ok := declaration.(*goast.FuncDecl)
		if !ok || funcDecl.Name == nil || funcDecl.Name.Name != functionName {
			continue
		}

		if funcDecl.Type != nil {
			return goastutil.ASTToTypeString(funcDecl.Type, "")
		}
	}
	return ""
}

// getFunctionSignatureFromInspector looks up a function signature using the
// TypeQuerier.
//
// Takes functionName (string) which is the function name to look up.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides package context.
//
// Returns string which is the function signature from the inspector, or empty
// string if not found.
func (d *document) getFunctionSignatureFromInspector(functionName string, ann *ast_domain.GoGeneratorAnnotation) string {
	if d.TypeInspector == nil || ann.ResolvedType == nil {
		return ""
	}

	pkgAlias := ann.ResolvedType.PackageAlias
	packagePath := ann.ResolvedType.CanonicalPackagePath
	filePath := d.getResolvedFilePath(ann)

	signature := d.TypeInspector.FindFuncSignature(pkgAlias, functionName, packagePath, filePath)
	if signature != nil {
		return signature.ToSignatureString()
	}
	return ""
}

// getMethodSignatureFromInspector looks up a method signature using the
// TypeInspector based on the base expression's type.
//
// Takes memberExpr (*ast_domain.MemberExpression) which is the method call
// expression.
// Takes methodName (string) which is the name of the method to look up.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type context.
//
// Returns string which is the method signature, or empty if not found.
func (d *document) getMethodSignatureFromInspector(memberExpr *ast_domain.MemberExpression, methodName string, ann *ast_domain.GoGeneratorAnnotation) string {
	if d.TypeInspector == nil {
		return ""
	}

	baseAnn := memberExpr.Base.GetGoAnnotation()
	if baseAnn == nil || baseAnn.ResolvedType == nil || baseAnn.ResolvedType.TypeExpression == nil {
		return ""
	}

	filePath := d.getResolvedFilePath(ann)
	packagePath := ""
	if ann != nil && ann.ResolvedType != nil {
		packagePath = ann.ResolvedType.CanonicalPackagePath
	}

	methodInfo := d.TypeInspector.FindMethodInfo(
		baseAnn.ResolvedType.TypeExpression,
		methodName,
		packagePath,
		filePath,
	)

	if methodInfo != nil {
		return methodInfo.Signature.ToSignatureString()
	}
	return ""
}

// formatSymbolDeclaration formats a symbol declaration line based on its kind.
//
// Takes symbolKind (string) which is the type of symbol, such as "property".
// Takes displayName (string) which is the name shown for the symbol.
// Takes typeString (string) which holds the type details for the symbol.
//
// Returns string which is the formatted declaration line.
func formatSymbolDeclaration(symbolKind, displayName, typeString string) string {
	switch symbolKind {
	case "property":
		return fmt.Sprintf("(property) %s: %s", displayName, typeString)
	case "attribute":
		return fmt.Sprintf("(attribute) %s", displayName)
	default:
		return fmt.Sprintf("(%s) %s: %s", symbolKind, displayName, typeString)
	}
}

// isAttributeSymbol checks if the annotation represents an attribute symbol.
//
// Takes ann (*GoGeneratorAnnotation) which is the annotation to check.
//
// Returns bool which is true if the annotation has a symbol but no base
// code generation variable name.
func isAttributeSymbol(ann *ast_domain.GoGeneratorAnnotation) bool {
	return ann.Symbol != nil && ann.BaseCodeGenVarName == nil
}

// isFrameworkIdentifier checks if a symbol name is a special framework
// identifier.
//
// Takes name (string) which is the symbol name to check.
//
// Returns bool which is true if name is "state" or "props".
func isFrameworkIdentifier(name string) bool {
	return name == "state" || name == "props"
}

// extractElementType gets the element type from an array or slice type.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns goast.Expr which is the element type if typeExpr is an array or
// slice, or the original typeExpr if not.
// Returns bool which is true if typeExpr was an array or slice type.
func extractElementType(typeExpr goast.Expr) (goast.Expr, bool) {
	if arrayType, ok := typeExpr.(*goast.ArrayType); ok {
		return arrayType.Elt, true
	}
	return typeExpr, false
}

// buildMethodPreviewString formats a non-struct named type as a preview
// showing its underlying type and methods. Used for types like
// `type RichText []Paragraph` where there are no struct fields to show.
//
// Takes typeDTO (*inspector_dto.Type) which provides the type data.
// Takes maxMethods (int) which limits how many methods to show.
//
// Returns string which contains the formatted preview, or empty if the type
// has no methods.
func buildMethodPreviewString(typeDTO *inspector_dto.Type, maxMethods int) string {
	if len(typeDTO.Methods) == 0 {
		return ""
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "type %s %s", typeDTO.Name, typeDTO.TypeString)

	methodsToShow := min(len(typeDTO.Methods), maxMethods)
	for i := range methodsToShow {
		method := typeDTO.Methods[i]
		_, _ = fmt.Fprintf(&b, "\nfunc (%s) %s%s", typeDTO.Name, method.Name, formatMethodParams(method))
	}

	if len(typeDTO.Methods) > maxMethods {
		_, _ = fmt.Fprintf(&b, "\n... (%d more methods)", len(typeDTO.Methods)-maxMethods)
	}

	return b.String()
}

// formatMethodParams formats a method's parameters and return types for
// display in a method preview.
//
// Takes method (*inspector_dto.Method) which provides the signature.
//
// Returns string which contains the formatted parameter list and return type
// (e.g. "(data []byte) error").
func formatMethodParams(method *inspector_dto.Method) string {
	params := strings.Join(method.Signature.Params, ", ")
	results := strings.Join(method.Signature.Results, ", ")
	if len(method.Signature.Results) > 1 {
		results = "(" + results + ")"
	}
	if results != "" {
		return fmt.Sprintf("(%s) %s", params, results)
	}
	return fmt.Sprintf("(%s)", params)
}

// buildTypePreviewString formats a type DTO as a struct preview string.
//
// Takes typeDTO (*inspector_dto.Type) which provides the type data to format.
// Takes isSlice (bool) which indicates whether to show slice element context.
// Takes maxFields (int) which limits how many fields to show in the preview.
//
// Returns string which contains the formatted struct preview with fields.
func buildTypePreviewString(typeDTO *inspector_dto.Type, isSlice bool, maxFields int) string {
	var b strings.Builder

	if isSlice {
		_, _ = fmt.Fprintf(&b, "// element type of []%s\n", typeDTO.Name)
	}
	_, _ = fmt.Fprintf(&b, "type %s struct {", typeDTO.Name)

	fieldsToShow := min(len(typeDTO.Fields), maxFields)
	for i := range fieldsToShow {
		b.WriteString("\n    ")
		writeInspectorFieldLine(&b, typeDTO.Fields[i])
	}

	if len(typeDTO.Fields) > maxFields {
		_, _ = fmt.Fprintf(&b, "\n    ... (%d more fields)", len(typeDTO.Fields)-maxFields)
	}

	b.WriteString("\n}")
	return b.String()
}

// formatInspectorStructPreview formats a struct type as a preview with limited
// fields.
//
// Takes typeDTO (*inspector_dto.Type) which provides the struct type to format.
// Takes maxFields (int) which limits how many fields to show in the preview.
//
// Returns string which contains the formatted struct preview with type
// declaration and fields.
func formatInspectorStructPreview(typeDTO *inspector_dto.Type, maxFields int) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "type %s struct {", typeDTO.Name)

	fieldsToShow := min(len(typeDTO.Fields), maxFields)
	for i := range fieldsToShow {
		b.WriteString("\n    ")
		writeInspectorFieldLine(&b, typeDTO.Fields[i])
	}

	if len(typeDTO.Fields) > maxFields {
		_, _ = fmt.Fprintf(&b, "\n    ... (%d more fields)", len(typeDTO.Fields)-maxFields)
	}
	b.WriteString("\n}")
	return b.String()
}

// formatInspectorFieldLine formats a struct field for the type preview display.
//
// Takes field (*inspector_dto.Field) which contains the field data to format.
//
// Returns string which is the formatted line with padded name, type, and tag.
func formatInspectorFieldLine(field *inspector_dto.Field) string {
	var builder strings.Builder
	writeInspectorFieldLine(&builder, field)
	return builder.String()
}

// writeInspectorFieldLine writes a single struct field line to the
// builder with aligned name, type, and optional tag.
//
// Takes builder (*strings.Builder) which accumulates the output text.
// Takes field (*inspector_dto.Field) which holds the field name, type, and raw tag.
func writeInspectorFieldLine(builder *strings.Builder, field *inspector_dto.Field) {
	builder.WriteString(field.Name)
	if padding := fieldNameWidth - len(field.Name); padding > 0 {
		builder.WriteString(fieldPadding[:padding])
	}
	builder.WriteByte(' ')
	builder.WriteString(field.TypeString)
	if field.RawTag != "" {
		builder.WriteString(" `")
		builder.WriteString(field.RawTag)
		builder.WriteByte('`')
	}
}

// extractBaseIdentifier walks an expression tree to find the base identifier.
// It follows MemberExpr, IndexExpr, and CallExpr nodes by checking their base
// or callee fields until it finds an identifier.
//
// Takes expression (ast_domain.Expression) which is the expression tree to search.
//
// Returns *ast_domain.Identifier which is the base identifier, or nil if no
// identifier is found.
func extractBaseIdentifier(expression ast_domain.Expression) *ast_domain.Identifier {
	current := expression
	for current != nil {
		switch n := current.(type) {
		case *ast_domain.Identifier:
			return n
		case *ast_domain.MemberExpression:
			current = n.Base
		case *ast_domain.IndexExpression:
			current = n.Base
		case *ast_domain.CallExpression:
			current = n.Callee
		default:
			return nil
		}
	}
	return nil
}
