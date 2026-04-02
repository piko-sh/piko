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

package annotator_domain

// Provides utility functions for type resolution including stringability checks, type comparison, and diagnostic helpers.
// Supports the type resolver with common operations for determining type compatibility and formatting type information.

import (
	"fmt"

	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// unmapVirtualLocationToOriginal converts a position from a virtual .go file
// back to the original .pk file coordinates.
//
// Takes ctx (*AnalysisContext) which provides the current package path.
// Takes virtualLocation (ast_domain.Location) which specifies the position in the
// virtual file.
//
// Returns ast_domain.Location which contains the mapped position in the
// original file, or the unchanged location if no mapping exists.
func (tr *TypeResolver) unmapVirtualLocationToOriginal(
	ctx *AnalysisContext,
	virtualLocation ast_domain.Location,
) ast_domain.Location {
	vc, ok := tr.virtualModule.ComponentsByGoPath[ctx.CurrentGoFullPackagePath]
	if !ok || vc == nil || vc.Source == nil || vc.Source.Script == nil {
		return virtualLocation
	}

	scriptStart := vc.Source.Script.ScriptStartLocation

	return ast_domain.Location{
		Line:   scriptStart.Line + (virtualLocation.Line - 1),
		Column: virtualLocation.Column,
		Offset: 0,
	}
}

// logAnn returns a string form of an annotation for logging.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which is the annotation to
// format.
//
// Returns string which is the type as a string, "<nil>" if the annotation or
// its resolved type is nil, or "<unresolved>" if the type could not be turned
// into a string.
func (*TypeResolver) logAnn(ann *ast_domain.GoGeneratorAnnotation) string {
	if ann == nil || ann.ResolvedType == nil {
		return "<nil>"
	}
	typeString := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
	if typeString == "" {
		return "<unresolved>"
	}
	return typeString
}

// getPropertyName extracts the property name from a member expression.
//
// Takes expression (ast_domain.Expression) which is the expression to
// extract from.
//
// Returns string which is the property name, or empty if the expression is
// not a member expression with an identifier property.
func getPropertyName(expression ast_domain.Expression) string {
	if member, ok := expression.(*ast_domain.MemberExpression); ok {
		if identifier, isIdent := member.Property.(*ast_domain.Identifier); isIdent {
			return identifier.Name
		}
	}
	return ""
}

// setAnnotationOnExpression attaches an annotation to an expression by calling
// SetGoAnnotation on the Expression interface.
//
// When expression is nil, returns without doing anything.
//
// Takes expression (ast_domain.Expression) which is the target expression.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which is the annotation to
// attach.
func setAnnotationOnExpression(expression ast_domain.Expression, ann *ast_domain.GoGeneratorAnnotation) {
	if expression == nil {
		return
	}
	expression.SetGoAnnotation(ann)
}

// getAnnotationFromExpression extracts the Go generator annotation from an
// expression.
//
// Takes expression (ast_domain.Expression) which is the expression to
// extract from.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the annotation, or nil
// if expression is nil.
func getAnnotationFromExpression(expression ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	if expression == nil {
		return nil
	}
	return expression.GetGoAnnotation()
}

// handleUndefinedIdentifier creates a diagnostic for an undefined identifier.
//
// Takes ctx (*AnalysisContext) which provides the symbol table for
// suggestions and the diagnostics collector.
// Takes n (*ast_domain.Identifier) which is the undefined identifier
// that triggered the error.
// Takes location (ast_domain.Location) which specifies the source location
// of the identifier for the diagnostic.
//
// Returns *ast_domain.GoGeneratorAnnotation which is a fallback
// annotation with the "any" type for error recovery.
func handleUndefinedIdentifier(ctx *AnalysisContext, n *ast_domain.Identifier, location ast_domain.Location, _ int) *ast_domain.GoGeneratorAnnotation {
	message := fmt.Sprintf("Undefined variable: %s", n.Name)
	suggestion := findClosestMatch(n.Name, ctx.Symbols.AllSymbolNames())
	if suggestion != "" {
		message += fmt.Sprintf(". Did you mean '%s'?", suggestion)
	}
	ctx.Logger.Trace("Diagnostic", logger_domain.String("message", message))
	ctx.addDiagnostic(ast_domain.Error, message, n.Name, location.Add(n.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeUndefinedVariable)
	return newAnnotationWithTypeAndStringability(
		&ast_domain.ResolvedTypeInfo{
			TypeExpression:          goast.NewIdent(typeAny),
			PackageAlias:            "",
			CanonicalPackagePath:    "",
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		int(inspector_dto.StringableNone),
	)
}

// handleUnexportedFunctionAccess creates a diagnostic for an unexported function.
//
// This provides a helpful error message when users try to access an unexported
// (lowercase) function from a template. Template code is generated separately
// and can only access exported (capitalised) functions.
//
// Takes ctx (*AnalysisContext) which provides the diagnostics collector
// and logger.
// Takes n (*ast_domain.Identifier) which is the unexported identifier
// that triggered the error.
// Takes location (ast_domain.Location) which specifies the source location
// for the diagnostic.
//
// Returns *ast_domain.GoGeneratorAnnotation which is a fallback
// annotation with the "any" type for error recovery.
func handleUnexportedFunctionAccess(ctx *AnalysisContext, n *ast_domain.Identifier, location ast_domain.Location, _ int) *ast_domain.GoGeneratorAnnotation {
	capitalised := capitaliseFirstLetter(n.Name)
	message := fmt.Sprintf("Cannot access unexported function '%s' from template. Rename to '%s' (capitalised) to make it accessible.", n.Name, capitalised)
	ctx.Logger.Trace("Diagnostic", logger_domain.String("message", message))
	ctx.addDiagnostic(ast_domain.Error, message, n.Name, location.Add(n.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeUndefinedVariable)
	return newAnnotationWithTypeAndStringability(
		&ast_domain.ResolvedTypeInfo{
			TypeExpression:          goast.NewIdent(typeAny),
			PackageAlias:            "",
			CanonicalPackagePath:    "",
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		int(inspector_dto.StringableNone),
	)
}

// capitaliseFirstLetter returns the string with its first letter in uppercase.
//
// Takes s (string) which is the input string to change.
//
// Returns string which is the input with its first letter in uppercase, or the
// original string if empty or already uppercase.
func capitaliseFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}

// createBlankIdentifierAnnotation creates an annotation for the blank
// identifier "_".
//
// The blank identifier is a special Go syntax element used to discard values.
// In p-for expressions like "(_, item) in collection", users can use "_" to
// indicate they do not need the index variable.
//
// Returns *ast_domain.GoGeneratorAnnotation which represents an int type
// because loop indices are integers, but the actual value will be discarded.
func createBlankIdentifierAnnotation() *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      new("_"),
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          goast.NewIdent("int"),
			PackageAlias:            "",
			CanonicalPackagePath:    "",
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           int(inspector_dto.StringablePrimitive),
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}
