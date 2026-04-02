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

// Resolves member access expressions including struct fields, methods, and embedded type navigation with full type inference.
// Handles pointer dereferencing, method set resolution, and provides detailed diagnostics for invalid member access attempts.

import (
	"context"
	"fmt"

	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// tryResolveField looks up a field on a struct type.
//
// This function handles full field resolution, including:
//   - switching context for external packages
//   - resolving type aliases
//   - working out if the type can be converted to a string
//   - mapping virtual locations to original locations
//
// Takes ctx (*AnalysisContext) which provides the current analysis state.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which is the annotation
// for the struct type to search.
// Takes propName (string) which is the name of the field to find.
// Takes location (ast_domain.Location) which is the source location for the
// result annotation.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the annotation for the
// found field, or nil if not found.
// Returns map[string]goast.Expr which contains any type parameters.
// Returns bool which shows whether the field was found.
func (tr *TypeResolver) tryResolveField(
	goCtx context.Context,
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	propName string,
	location ast_domain.Location,
) (*ast_domain.GoGeneratorAnnotation, map[string]goast.Expr, bool) {
	if baseAnn == nil || baseAnn.ResolvedType == nil || baseAnn.ResolvedType.TypeExpression == nil {
		return nil, nil, false
	}

	ctx.Logger.Trace("[tryResolveField] Starting field lookup",
		logger_domain.String("property_name", propName),
		logger_domain.String("on_base_type", goastutil.ASTToTypeString(baseAnn.ResolvedType.TypeExpression, baseAnn.ResolvedType.PackageAlias)),
		logger_domain.String("initial_context_pkg", ctx.CurrentGoFullPackagePath),
		logger_domain.String("initial_context_file", ctx.CurrentGoSourcePath),
	)

	importerPackagePath, importerFilePath := tr.determineFieldLookupContext(goCtx, ctx, baseAnn)

	resolvedBaseTypeAST, effectivePackagePath, effectiveFilePath := tr.resolveBaseType(goCtx, ctx, baseAnn, importerPackagePath, importerFilePath)

	fieldInfo := tr.inspectFieldInType(goCtx, ctx, resolvedBaseTypeAST, propName, effectivePackagePath, effectiveFilePath, baseAnn.ResolvedType.PackageAlias)
	if fieldInfo == nil {
		return nil, nil, false
	}

	return tr.buildFieldAnnotation(goCtx, ctx, baseAnn, fieldInfo, location)
}

// determineFieldLookupContext finds the package and file paths to use when
// looking up fields.
//
// Takes ctx (*AnalysisContext) which provides the current analysis state.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which describes the base
// type being analysed.
//
// Returns importerPackagePath (string) which is the package path to use for field
// lookups.
// Returns importerFilePath (string) which is the file path to use for field
// lookups.
func (tr *TypeResolver) determineFieldLookupContext(
	goCtx context.Context,
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
) (importerPackagePath, importerFilePath string) {
	importerPackagePath = ctx.CurrentGoFullPackagePath
	importerFilePath = ctx.CurrentGoSourcePath

	if baseAnn.ResolvedType.CanonicalPackagePath != "" {
		return tr.switchContextForExternalType(goCtx, ctx, baseAnn, importerPackagePath, importerFilePath)
	}

	ctx.Logger.Trace("[tryResolveField] No context switch needed; base type is in the current package.")
	return importerPackagePath, importerFilePath
}

// switchContextForExternalType switches the lookup context to an external
// package.
//
// This helper implements two strategies for determining the correct file
// context:
//  1. Find the type DTO and use its defining file (when TypeExpr is valid in
//     canonical context)
//  2. Use any file from the canonical package (when TypeExpr has qualifiers
//     only valid from parent context)
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which contains the
// resolved type information for the external type.
// Takes importerPackagePath (string) which is the package path of the importing
// context.
// Takes importerFilePath (string) which is the file path of the importing
// context.
//
// Returns newPackagePath (string) which is the package path to use for the new
// context.
// Returns newFilePath (string) which is the file path to use for the new
// context.
func (tr *TypeResolver) switchContextForExternalType(
	goCtx context.Context,
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	importerPackagePath, importerFilePath string,
) (newPackagePath, newFilePath string) {
	ctx.Logger.Trace("[tryResolveField] Context Switch needed for external package.",
		logger_domain.String("target_pkg", baseAnn.ResolvedType.CanonicalPackagePath))

	baseTypeDto, _ := tr.inspector.ResolveExprToNamedTypeWithMemoization(
		goCtx,
		baseAnn.ResolvedType.TypeExpression,
		baseAnn.ResolvedType.CanonicalPackagePath,
		importerFilePath,
	)
	if baseTypeDto != nil && baseTypeDto.DefinedInFilePath != "" {
		newPackagePath = baseAnn.ResolvedType.CanonicalPackagePath
		newFilePath = baseTypeDto.DefinedInFilePath
		if importerFilePath != newFilePath {
			ctx.Logger.Trace("[tryResolveField] Context Switch: Found authoritative defining file.",
				logger_domain.String("old_file", importerFilePath),
				logger_domain.String("new_file", newFilePath))
		}
		return newPackagePath, newFilePath
	}

	ctx.Logger.Trace("[tryResolveField] Context Switch: Could not find DTO for base type. Trying parent package context.",
		logger_domain.String("type_to_find", goastutil.ASTToTypeString(baseAnn.ResolvedType.TypeExpression, baseAnn.ResolvedType.PackageAlias)),
		logger_domain.String("canonical_pkg", baseAnn.ResolvedType.CanonicalPackagePath))

	pkgFiles := tr.inspector.GetFilesForPackage(baseAnn.ResolvedType.CanonicalPackagePath)
	if len(pkgFiles) > 0 {
		newPackagePath = baseAnn.ResolvedType.CanonicalPackagePath
		newFilePath = pkgFiles[0]
		ctx.Logger.Trace("[tryResolveField] Context Switch: Using file from canonical package.",
			logger_domain.String("new_pkg", newPackagePath),
			logger_domain.String("new_file", newFilePath))
		return newPackagePath, newFilePath
	}

	if definingFile := tr.findDefiningFileFromTypeData(baseAnn.ResolvedType.TypeExpression, baseAnn.ResolvedType.CanonicalPackagePath); definingFile != "" {
		newPackagePath = baseAnn.ResolvedType.CanonicalPackagePath
		ctx.Logger.Trace("[tryResolveField] Context Switch: Found defining file via type name lookup in canonical package.",
			logger_domain.String("new_pkg", newPackagePath),
			logger_domain.String("new_file", definingFile))
		return newPackagePath, definingFile
	}

	ctx.Logger.Trace("[tryResolveField] Context Switch: No files found for canonical package. Keeping original context.",
		logger_domain.String("keeping_pkg", importerPackagePath),
		logger_domain.String("keeping_file", importerFilePath))
	return importerPackagePath, importerFilePath
}

// findDefiningFileFromTypeData looks up a type by name in the canonical
// package's type data and returns the file where it is defined. This handles
// the case where external packages loaded from export data have no file
// imports registered, but their named types still carry the defining file
// path.
//
// Takes typeExpression (goast.Expr) which is the type expression to look up.
// Takes canonicalPackagePath (string) which is the canonical package path.
//
// Returns string which is the defining file path, or empty if not found.
func (tr *TypeResolver) findDefiningFileFromTypeData(typeExpression goast.Expr, canonicalPackagePath string) string {
	typeName, _, ok := inspector_domain.DeconstructTypeExpr(typeExpression)
	if !ok || typeName == "" {
		return ""
	}

	allPackages := tr.inspector.GetAllPackages()
	canonicalPackage, found := allPackages[canonicalPackagePath]
	if !found || canonicalPackage == nil {
		return ""
	}

	namedType, exists := canonicalPackage.NamedTypes[typeName]
	if !exists || namedType == nil {
		return ""
	}

	return namedType.DefinedInFilePath
}

// resolveBaseType resolves the base type through any aliases and determines
// the effective context.
//
// This helper extracts the logic of resolving type aliases and determining
// which context (package path and file path) should be used for subsequent
// inspector queries.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which specifies the type
// annotation to resolve.
// Takes importerPackagePath (string) which is the package path of the caller.
// Takes importerFilePath (string) which is the file path of the caller.
//
// Returns resolvedAST (goast.Expr) which is the resolved type expression.
// Returns effectivePackagePath (string) which is the package path to use for
// subsequent lookups.
// Returns effectiveFilePath (string) which is the file path to use for
// subsequent lookups.
func (tr *TypeResolver) resolveBaseType(
	goCtx context.Context,
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	importerPackagePath, importerFilePath string,
) (resolvedAST goast.Expr, effectivePackagePath, effectiveFilePath string) {
	resolvedBaseTypeAST, resolvedFilePath := tr.inspector.ResolveToUnderlyingASTWithContext(goCtx, baseAnn.ResolvedType.TypeExpression, importerFilePath)
	tr.logAliasResolution(ctx, baseAnn.ResolvedType.TypeExpression, resolvedBaseTypeAST, baseAnn.ResolvedType.PackageAlias)

	effectiveFilePath = importerFilePath
	effectivePackagePath = importerPackagePath
	if resolvedFilePath != "" && resolvedFilePath != importerFilePath {
		effectiveFilePath = resolvedFilePath
		if resolvedPackage := tr.inspector.PackagePathForFile(resolvedFilePath); resolvedPackage != "" {
			effectivePackagePath = resolvedPackage
		}
	}

	return resolvedBaseTypeAST, effectivePackagePath, effectiveFilePath
}

// inspectFieldInType calls the inspector to find field information and logs
// the result.
//
// This helper encapsulates the inspector call and result logging, making the
// main tryResolveField function more focused on the high-level flow.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes resolvedBaseTypeAST (goast.Expr) which is the resolved base type to
// search within.
// Takes propName (string) which is the name of the field to find.
// Takes effectivePackagePath (string) which is the package path for resolution.
// Takes effectiveFilePath (string) which is the file path for resolution.
// Takes packageAlias (string) which is the package alias for type string output.
//
// Returns *inspector_dto.FieldInfo which contains the field details, or nil
// if the field is not found.
func (tr *TypeResolver) inspectFieldInType(
	goCtx context.Context,
	ctx *AnalysisContext,
	resolvedBaseTypeAST goast.Expr,
	propName, effectivePackagePath, effectiveFilePath, packageAlias string,
) *inspector_dto.FieldInfo {
	resolvedTypeString := goastutil.ASTToTypeString(resolvedBaseTypeAST, packageAlias)
	ctx.Logger.Trace("[tryResolveField] Calling inspector.FindFieldInfo",
		logger_domain.String("property_name", propName),
		logger_domain.String("on_resolved_type", resolvedTypeString),
		logger_domain.String("using_pkg_context", effectivePackagePath),
		logger_domain.String("using_file_context", effectiveFilePath),
	)
	fieldInfo := tr.inspector.FindFieldInfo(
		goCtx,
		resolvedBaseTypeAST,
		propName,
		effectivePackagePath,
		effectiveFilePath,
	)

	if fieldInfo == nil {
		ctx.Logger.Trace("[tryResolveField] Inspector returned no FieldInfo. Field not found.")
		return nil
	}

	ctx.Logger.Trace("[tryResolveField] Inspector returned FieldInfo successfully.",
		logger_domain.String("found_field_name", fieldInfo.Name),
		logger_domain.String("field_type", goastutil.ASTToTypeString(fieldInfo.Type, fieldInfo.PackageAlias)),
		logger_domain.String("field_canonical_pkg", fieldInfo.CanonicalPackagePath),
	)
	return fieldInfo
}

// logAliasResolution logs whether a type was resolved through an alias.
//
// Takes ctx (*AnalysisContext) which provides the logger for trace output.
// Takes original (goast.Expr) which is the type expression before lookup.
// Takes resolved (goast.Expr) which is the type expression after lookup.
// Takes packageAlias (string) which is the package alias used for formatting.
func (*TypeResolver) logAliasResolution(
	ctx *AnalysisContext,
	original goast.Expr,
	resolved goast.Expr,
	packageAlias string,
) {
	originalTypeString := goastutil.ASTToTypeString(original, packageAlias)
	resolvedTypeString := goastutil.ASTToTypeString(resolved, packageAlias)
	if originalTypeString != resolvedTypeString {
		ctx.Logger.Trace("[tryResolveField] Alias Resolution: Base type was resolved through an alias.",
			logger_domain.String("original_type", originalTypeString),
			logger_domain.String("resolved_underlying_type", resolvedTypeString),
		)
	} else {
		ctx.Logger.Trace("[tryResolveField] Alias Resolution: Base type is not an alias.")
	}
}

// buildFieldAnnotation constructs the final annotation for a resolved field.
//
// Takes ctx (*AnalysisContext) which provides the analysis state,
// source paths, and logger.
// Takes fieldInfo (*inspector_dto.FieldInfo) which contains the
// resolved field details including type, location, and tags.
// Takes location (ast_domain.Location) which is the source location of the
// field access expression.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the complete
// annotation for the field with resolved type, symbol, and
// stringability information.
// Returns map[string]goast.Expr which contains any generic type
// parameter substitutions from the field's parent type.
// Returns bool which is always true since this method is only called
// when the field has been found.
func (tr *TypeResolver) buildFieldAnnotation(
	goCtx context.Context,
	ctx *AnalysisContext,
	_ *ast_domain.GoGeneratorAnnotation,
	fieldInfo *inspector_dto.FieldInfo,
	location ast_domain.Location,
) (*ast_domain.GoGeneratorAnnotation, map[string]goast.Expr, bool) {
	finalFieldTypeAST := tr.inspector.ResolveToUnderlyingAST(fieldInfo.Type, fieldInfo.DefiningFilePath)
	if goastutil.ASTToTypeString(finalFieldTypeAST) != goastutil.ASTToTypeString(fieldInfo.Type) {
		ctx.Logger.Trace("[tryResolveField] Alias Resolution: Resolved field's own type.",
			logger_domain.String("original_field_type", goastutil.ASTToTypeString(fieldInfo.Type, fieldInfo.PackageAlias)),
			logger_domain.String("final_field_type", goastutil.ASTToTypeString(finalFieldTypeAST)),
		)
	}

	finalCanonicalPath, finalPackageAlias := tr.correctFieldTypeContext(goCtx, ctx, fieldInfo, finalFieldTypeAST)

	resolvedTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          finalFieldTypeAST,
		PackageAlias:            finalPackageAlias,
		CanonicalPackagePath:    finalCanonicalPath,
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      ctx.CurrentGoFullPackagePath,
		InitialFilePath:         ctx.CurrentGoSourcePath,
	}

	stringability, isPointer := tr.determineStringability(ctx, resolvedTypeInfo)
	ctx.Logger.Trace("[tryResolveField] Determined stringability for field type.",
		logger_domain.Int("stringability_code", stringability),
		logger_domain.Bool("is_pointer_to_stringable", isPointer),
	)

	virtualLocation := ast_domain.Location{Line: fieldInfo.DefinitionLine, Column: fieldInfo.DefinitionColumn, Offset: 0}
	originalDefLocation := tr.unmapVirtualLocationToOriginal(ctx, virtualLocation)

	finalAnnotation := newAnnotationFull(resolvedTypeInfo, &ctx.SFCSourcePath, stringability)
	finalAnnotation.ParentTypeName = &fieldInfo.ParentTypeName
	finalAnnotation.GeneratedSourcePath = &fieldInfo.DefiningFilePath
	finalAnnotation.Symbol = &ast_domain.ResolvedSymbol{
		Name:                fieldInfo.Name,
		ReferenceLocation:   location,
		DeclarationLocation: originalDefLocation,
	}
	finalAnnotation.FieldTag = &fieldInfo.RawTag
	finalAnnotation.IsPointerToStringable = isPointer

	ctx.Logger.Trace("[tryResolveField] SUCCESS: Field resolved. Returning full annotation.",
		logger_domain.String("final_type", goastutil.ASTToTypeString(finalAnnotation.ResolvedType.TypeExpression, finalAnnotation.ResolvedType.PackageAlias)),
	)
	return finalAnnotation, fieldInfo.SubstMap, true
}

// correctFieldTypeContext fixes the package path and alias for a resolved
// field type when the original values are incorrect.
//
// Takes ctx (*AnalysisContext) which holds the analysis state and logger.
// Takes fieldInfo (*inspector_dto.FieldInfo) which contains the field details.
// Takes finalFieldTypeAST (goast.Expr) which is the AST node for the field
// type.
//
// Returns finalCanonicalPath (string) which is the fixed package path.
// Returns finalPackageAlias (string) which is the fixed package alias.
func (tr *TypeResolver) correctFieldTypeContext(
	goCtx context.Context,
	ctx *AnalysisContext,
	fieldInfo *inspector_dto.FieldInfo,
	finalFieldTypeAST goast.Expr,
) (finalCanonicalPath, finalPackageAlias string) {
	finalCanonicalPath = fieldInfo.CanonicalPackagePath
	finalPackageAlias = fieldInfo.PackageAlias

	if !tr.fieldTypeNeedsCorrection(finalFieldTypeAST, fieldInfo) {
		return finalCanonicalPath, finalPackageAlias
	}

	ctx.Logger.Trace("[tryResolveField] Correcting context for resolved alias.")
	correctedPath, correctedAlias := tr.resolveCorrectFieldContext(goCtx, ctx, fieldInfo, finalFieldTypeAST)

	if correctedPath != "" {
		finalCanonicalPath = correctedPath
	}
	if correctedAlias != "" {
		finalPackageAlias = correctedAlias
	}

	return finalCanonicalPath, finalPackageAlias
}

// fieldTypeNeedsCorrection checks if the resolved field type differs from the
// original.
//
// Takes finalFieldTypeAST (goast.Expr) which is the resolved type expression.
// Takes fieldInfo (*inspector_dto.FieldInfo) which contains the original type.
//
// Returns bool which is true when the types differ and need correction.
func (*TypeResolver) fieldTypeNeedsCorrection(finalFieldTypeAST goast.Expr, fieldInfo *inspector_dto.FieldInfo) bool {
	return goastutil.ASTToTypeString(finalFieldTypeAST) != goastutil.ASTToTypeString(fieldInfo.Type)
}

// resolveCorrectFieldContext finds the correct package path and alias for a
// resolved field type.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes fieldInfo (*inspector_dto.FieldInfo) which describes the field being
// resolved.
// Takes finalFieldTypeAST (goast.Expr) which is the AST expression for the
// field type.
//
// Returns canonicalPath (string) which is the full package path for the type,
// or empty if the type cannot be resolved.
// Returns packageAlias (string) which is the package alias used in the type
// expression, or empty if the type cannot be resolved.
func (tr *TypeResolver) resolveCorrectFieldContext(
	goCtx context.Context,
	ctx *AnalysisContext,
	fieldInfo *inspector_dto.FieldInfo,
	finalFieldTypeAST goast.Expr,
) (canonicalPath, packageAlias string) {
	underlyingTypeDTO, _ := tr.inspector.ResolveExprToNamedTypeWithMemoization(
		goCtx,
		finalFieldTypeAST,
		fieldInfo.DefiningPackagePath,
		fieldInfo.DefiningFilePath,
	)
	if underlyingTypeDTO == nil {
		return "", ""
	}

	trueCanonicalPath := tr.inspector.FindPackagePathForTypeDTO(underlyingTypeDTO)
	if trueCanonicalPath == "" {
		return "", ""
	}

	ctx.Logger.Trace("[tryResolveField] Found true canonical path for underlying type.",
		logger_domain.String("path", trueCanonicalPath))

	_, newAlias, _ := inspector_domain.DeconstructTypeExpr(finalFieldTypeAST)
	return trueCanonicalPath, newAlias
}

// tryResolveMethod looks up a method on a type.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which is the annotation
// for the type to search.
// Takes propName (string) which is the method name to find.
// Takes location (ast_domain.Location) which is the source location for errors.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the method annotation.
// Returns bool which is true when the method was found.
func (tr *TypeResolver) tryResolveMethod(
	goCtx context.Context,
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	propName string,
	location ast_domain.Location,
) (*ast_domain.GoGeneratorAnnotation, bool) {
	if baseAnn == nil || baseAnn.ResolvedType == nil || baseAnn.ResolvedType.TypeExpression == nil {
		return nil, false
	}

	ctx.Logger.Trace("[tryResolveMethod] Starting method lookup",
		logger_domain.String("property_name", propName),
		logger_domain.String("on_base_type", goastutil.ASTToTypeString(baseAnn.ResolvedType.TypeExpression, baseAnn.ResolvedType.PackageAlias)),
		logger_domain.String("initial_context_pkg", ctx.CurrentGoFullPackagePath),
	)

	importerPackagePath, importerFilePath := tr.determineMethodLookupContext(goCtx, ctx, baseAnn)

	ctx.Logger.Trace("[tryResolveMethod] Resolved final context for inspector call",
		logger_domain.String("using_pkg_context", importerPackagePath),
		logger_domain.String("using_file_context", importerFilePath),
	)

	methodInfo := tr.inspector.FindMethodInfo(
		baseAnn.ResolvedType.TypeExpression,
		propName,
		importerPackagePath,
		importerFilePath,
	)

	if methodInfo == nil {
		ctx.Logger.Trace("[tryResolveMethod] Inspector returned no method info. Method not found.")
		return nil, false
	}

	ctx.Logger.Trace("[tryResolveMethod] SUCCESS: Found method info.",
		logger_domain.String("signature", methodInfo.Signature.ToSignatureString()),
		logger_domain.Int("defLine", methodInfo.DefinitionLine),
		logger_domain.Int("defColumn", methodInfo.DefinitionColumn),
		logger_domain.String("defFile", methodInfo.DefinitionFilePath))

	return tr.buildMethodAnnotation(ctx, baseAnn, methodInfo, propName, location), true
}

// determineMethodLookupContext determines the correct context for method
// lookup.
//
// Takes ctx (*AnalysisContext) which provides the current analysis state.
// Takes baseAnn (*GoGeneratorAnnotation) which contains the base type info.
//
// Returns importerPackagePath (string) which is the package path to use.
// Returns importerFilePath (string) which is the file path to use.
func (tr *TypeResolver) determineMethodLookupContext(
	goCtx context.Context,
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
) (importerPackagePath, importerFilePath string) {
	importerPackagePath = ctx.CurrentGoFullPackagePath
	importerFilePath = ctx.CurrentGoSourcePath

	if baseAnn.ResolvedType.CanonicalPackagePath != "" {
		importerPackagePath = baseAnn.ResolvedType.CanonicalPackagePath

		baseTypeDto, _ := tr.inspector.ResolveExprToNamedTypeWithMemoization(
			goCtx,
			baseAnn.ResolvedType.TypeExpression,
			importerPackagePath,
			importerFilePath,
		)
		if baseTypeDto != nil && baseTypeDto.DefinedInFilePath != "" {
			importerFilePath = baseTypeDto.DefinedInFilePath
		}
	}

	return importerPackagePath, importerFilePath
}

// buildMethodAnnotation constructs an annotation for a resolved method.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and source
// path.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which supplies the base
// type information to inherit.
// Takes methodInfo (*inspector_dto.Method) which contains the method
// definition details.
// Takes propName (string) which specifies the property name for the symbol.
// Takes location (ast_domain.Location) which indicates the reference location.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the constructed
// annotation with resolved type and symbol information.
func (tr *TypeResolver) buildMethodAnnotation(
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	methodInfo *inspector_dto.Method,
	propName string,
	location ast_domain.Location,
) *ast_domain.GoGeneratorAnnotation {
	virtualLocation := ast_domain.Location{
		Line:   methodInfo.DefinitionLine,
		Column: methodInfo.DefinitionColumn,
		Offset: 0,
	}

	originalDefLocation := tr.unmapVirtualLocationToOriginal(ctx, virtualLocation)
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     new(methodInfo.DefinitionFilePath),
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          goast.NewIdent(typeFunction),
			PackageAlias:            baseAnn.ResolvedType.PackageAlias,
			CanonicalPackagePath:    baseAnn.ResolvedType.CanonicalPackagePath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		Symbol: &ast_domain.ResolvedSymbol{
			Name:                propName,
			ReferenceLocation:   location,
			DeclarationLocation: originalDefLocation,
		},
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      &ctx.SFCSourcePath,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           int(inspector_dto.StringableNone),
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

// handleUnknownMember creates a diagnostic for an unknown field or method
// access.
//
// Takes ctx (*AnalysisContext) which provides the analysis state for adding
// diagnostics.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which is the annotation of
// the base expression being accessed.
// Takes propName (string) which is the name of the unknown property.
// Takes n (*ast_domain.MemberExpression) which is the member expression node.
// Takes location (ast_domain.Location) which is the source location for the
// diagnostic.
//
// Returns *ast_domain.GoGeneratorAnnotation which is a fallback annotation for
// error recovery.
func (tr *TypeResolver) handleUnknownMember(
	goCtx context.Context,
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	propName string,
	n *ast_domain.MemberExpression,
	location ast_domain.Location,
) *ast_domain.GoGeneratorAnnotation {
	baseTypeName := goastutil.ASTToTypeString(baseAnn.ResolvedType.TypeExpression, baseAnn.ResolvedType.PackageAlias)
	message := fmt.Sprintf("Property '%s' does not exist on type '%s'", propName, baseTypeName)

	if propName == "length" && tr.isLenable(baseAnn.ResolvedType) {
		message += ". Did you mean to use the built-in len() function? (e.g., len(variable))"
		ctx.addDiagnosticForExpression(ast_domain.Error, message, n, location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeUndefinedMember)
		return newFallbackAnnotation()
	}

	importerPackagePath, importerFilePath := tr.determineMethodLookupContext(goCtx, ctx, baseAnn)

	suggestions := tr.inspector.GetAllFieldsAndMethods(
		baseAnn.ResolvedType.TypeExpression,
		importerPackagePath,
		importerFilePath,
	)

	if suggestion := findClosestMatch(propName, suggestions); suggestion != "" {
		message += fmt.Sprintf(". Did you mean '%s'?", suggestion)
	}

	ctx.addDiagnosticForExpression(ast_domain.Error, message, n, location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeUndefinedMember)
	return newFallbackAnnotation()
}

// findCallSignature finds the function signature for a method call
// expression.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes callee (*ast_domain.MemberExpression) which is the method
// call expression to look up.
//
// Returns *inspector_dto.FunctionSignature which is the found signature, or nil
// if not found.
// Returns *ast_domain.GoGeneratorAnnotation which is the base annotation from
// the callee.
// Returns *inspector_dto.Method which holds the method details if this is a
// method call, or nil for package functions.
// Returns bool which is true when the signature was found.
func (tr *TypeResolver) findCallSignature(
	goCtx context.Context,
	ctx *AnalysisContext,
	callee *ast_domain.MemberExpression,
) (*inspector_dto.FunctionSignature, *ast_domain.GoGeneratorAnnotation, *inspector_dto.Method, bool) {
	baseAnn := getAnnotationFromExpression(callee.Base)
	if baseAnn == nil {
		return nil, nil, nil, false
	}
	prop, ok := callee.Property.(*ast_domain.Identifier)
	if !ok {
		return nil, baseAnn, nil, false
	}

	ctx.Logger.Trace("[DEEP_DEBUG] findCallSignature",
		logger_domain.String(logKeyCallee, callee.String()),
		logger_domain.String("baseType", tr.logAnn(baseAnn)),
		logger_domain.String("methodName", prop.Name),
	)

	if baseAnn.ResolvedType != nil && baseAnn.ResolvedType.TypeExpression == nil {
		sig, ann, found := tr.findPackageFunctionSignature(ctx, baseAnn, prop.Name, callee)
		return sig, ann, nil, found
	}

	if baseAnn.ResolvedType != nil {
		return tr.findMethodOnTypeSignature(goCtx, ctx, baseAnn, prop.Name, callee)
	}

	ctx.Logger.Trace("  -> Outcome: Signature NOT found.", logger_domain.String(logKeyCallee, callee.String()))
	return nil, baseAnn, nil, false
}

// findPackageFunctionSignature looks up a function in a package.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which contains the
// resolved package information.
// Takes functionName (string) which is the name of the function to find.
// Takes callee (*ast_domain.MemberExpression) which is the member expression being
// resolved.
//
// Returns *inspector_dto.FunctionSignature which is the found signature, or
// nil if not found.
// Returns *ast_domain.GoGeneratorAnnotation which is the base annotation.
// Returns bool which is true if the function was found.
func (tr *TypeResolver) findPackageFunctionSignature(
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	functionName string,
	callee *ast_domain.MemberExpression,
) (*inspector_dto.FunctionSignature, *ast_domain.GoGeneratorAnnotation, bool) {
	ctx.Logger.Trace("  -> Path: Trying as package function.",
		logger_domain.String("packageAlias", baseAnn.ResolvedType.PackageAlias))

	sig := tr.inspector.FindFuncSignature(
		baseAnn.ResolvedType.PackageAlias,
		functionName,
		ctx.CurrentGoFullPackagePath,
		ctx.CurrentGoSourcePath,
	)

	if sig != nil {
		ctx.Logger.Trace("  -> Outcome: Found signature.", logger_domain.String(logKeyCallee, callee.String()))
		return sig, baseAnn, true
	}

	ctx.Logger.Trace("  -> Outcome: Signature NOT found.", logger_domain.String(logKeyCallee, callee.String()))
	return nil, baseAnn, false
}

// findMethodOnTypeSignature looks up a method on a type.
//
// Takes ctx (*AnalysisContext) which provides the current analysis state.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which holds the resolved
// base type details.
// Takes methodName (string) which specifies the method to find.
// Takes callee (*ast_domain.MemberExpression) which is the member expression being
// resolved.
//
// Returns *inspector_dto.FunctionSignature which is the method signature.
// Returns *ast_domain.GoGeneratorAnnotation which is the base annotation for
// context.
// Returns *inspector_dto.Method which provides extra method details.
// Returns bool which is true when the method was found.
func (tr *TypeResolver) findMethodOnTypeSignature(
	goCtx context.Context,
	ctx *AnalysisContext,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	methodName string,
	callee *ast_domain.MemberExpression,
) (*inspector_dto.FunctionSignature, *ast_domain.GoGeneratorAnnotation, *inspector_dto.Method, bool) {
	ctx.Logger.Trace("  -> Path: Trying as method on a type.",
		logger_domain.String("packageAlias", baseAnn.ResolvedType.PackageAlias))

	importerPackagePath := ctx.CurrentGoFullPackagePath
	importerFilePath := ctx.CurrentGoSourcePath

	ctx.Logger.Trace("[findCallSignature] Initial context for method lookup",
		logger_domain.String("pkg", importerPackagePath),
		logger_domain.String("file", importerFilePath),
	)

	if baseAnn.ResolvedType.CanonicalPackagePath != "" {
		ctx.Logger.Trace("[findCallSignature] Context Switch: Base type is from an external package.",
			logger_domain.String("old_pkg_path", importerPackagePath),
			logger_domain.String("new_pkg_path", baseAnn.ResolvedType.CanonicalPackagePath),
		)
		importerPackagePath = baseAnn.ResolvedType.CanonicalPackagePath

		baseTypeDto, _ := tr.inspector.ResolveExprToNamedTypeWithMemoization(
			goCtx,
			baseAnn.ResolvedType.TypeExpression,
			importerPackagePath,
			importerFilePath,
		)
		if baseTypeDto != nil && baseTypeDto.DefinedInFilePath != "" {
			ctx.Logger.Trace("[findCallSignature] Context Switch: Found specific defining file for base type.",
				logger_domain.String("old_file_path", importerFilePath),
				logger_domain.String("new_file_path", baseTypeDto.DefinedInFilePath),
			)
			importerFilePath = baseTypeDto.DefinedInFilePath
		} else {
			ctx.Logger.Trace("[findCallSignature] Context Switch: Could not find specific DTO for base type; proceeding with best-effort file path.")
		}
	} else {
		ctx.Logger.Trace("[findCallSignature] No context switch needed; base type is in the current package.")
	}

	methodInfo := tr.inspector.FindMethodInfo(
		baseAnn.ResolvedType.TypeExpression,
		methodName,
		importerPackagePath,
		importerFilePath,
	)

	if methodInfo != nil {
		ctx.Logger.Trace("  -> Outcome: Found method info.",
			logger_domain.String(logKeyCallee, callee.String()),
			logger_domain.String("definingPackage", methodInfo.DeclaringPackagePath),
			logger_domain.String("definingFile", methodInfo.DefinitionFilePath))
		return &methodInfo.Signature, baseAnn, methodInfo, true
	}

	ctx.Logger.Trace("  -> Outcome: Method NOT found.", logger_domain.String(logKeyCallee, callee.String()))
	return nil, baseAnn, nil, false
}

// findFuncDeclInCurrentContext finds a function declaration in the current
// component.
//
// Takes ctx (*AnalysisContext) which provides the package context to search.
// Takes functionName (string) which specifies the function name to find.
//
// Returns *goast.FuncDecl which is the matching declaration, or nil if not
// found.
func (tr *TypeResolver) findFuncDeclInCurrentContext(ctx *AnalysisContext, functionName string) *goast.FuncDecl {
	if ctx == nil {
		return nil
	}
	vc, ok := tr.virtualModule.ComponentsByGoPath[ctx.CurrentGoFullPackagePath]
	if !ok || vc == nil {
		return nil
	}

	if vc.Source.Script == nil || vc.Source.Script.AST == nil {
		return nil
	}

	for _, declaration := range vc.Source.Script.AST.Decls {
		if functionDeclaration, isFunc := declaration.(*goast.FuncDecl); isFunc {
			if functionDeclaration.Recv == nil && functionDeclaration.Name != nil && functionDeclaration.Name.Name == functionName {
				return functionDeclaration
			}
		}
	}
	return nil
}

// findUnexportedFuncDeclInCurrentContext searches for an unexported function
// with the given name in the current component's script AST.
//
// This helps provide useful error messages when users try to call unexported
// functions from templates. Unexported functions (names starting with a
// lowercase letter) are not registered as symbols but may exist in the source.
//
// Takes ctx (*AnalysisContext) which provides the package context to search.
// Takes functionName (string) which specifies the function name to find.
//
// Returns *goast.FuncDecl which is the matching unexported declaration, or nil
// if not found or if the function is exported.
func (tr *TypeResolver) findUnexportedFuncDeclInCurrentContext(ctx *AnalysisContext, functionName string) *goast.FuncDecl {
	if ctx == nil {
		return nil
	}
	vc, ok := tr.virtualModule.ComponentsByGoPath[ctx.CurrentGoFullPackagePath]
	if !ok || vc == nil {
		return nil
	}
	if vc.Source.Script == nil || vc.Source.Script.AST == nil {
		return nil
	}
	if !isUnexportedName(functionName) {
		return nil
	}

	for _, declaration := range vc.Source.Script.AST.Decls {
		if functionDeclaration := matchUnexportedFuncDecl(declaration, functionName); functionDeclaration != nil {
			return functionDeclaration
		}
	}
	return nil
}

// parseSignatureFromFuncDecl extracts a function signature from a Go AST
// function declaration.
//
// Takes functionDeclaration (*goast.FuncDecl) which is the function
// declaration to parse.
// Takes ctx (*AnalysisContext) which provides the current package
// context.
//
// Returns *inspector_dto.FunctionSignature which contains the parsed
// parameter and return types, or nil if functionDeclaration or
// functionDeclaration.Type is nil.
func (*TypeResolver) parseSignatureFromFuncDecl(functionDeclaration *goast.FuncDecl, ctx *AnalysisContext) *inspector_dto.FunctionSignature {
	if functionDeclaration == nil || functionDeclaration.Type == nil {
		return nil
	}
	return &inspector_dto.FunctionSignature{
		Params:  parseFieldListTypeStrings(functionDeclaration.Type.Params, ctx.CurrentGoPackageName),
		Results: parseFieldListTypeStrings(functionDeclaration.Type.Results, ctx.CurrentGoPackageName),
	}
}

// isUnexportedName reports whether the name starts with a lowercase letter.
//
// Takes name (string) which is the identifier to check.
//
// Returns bool which is true if the name begins with a lowercase ASCII letter.
func isUnexportedName(name string) bool {
	return len(name) > 0 && name[0] >= 'a' && name[0] <= 'z'
}

// matchUnexportedFuncDecl checks if a declaration is a matching unexported
// function.
//
// Takes declaration (goast.Decl) which is the declaration to check.
// Takes functionName (string) which is the function name to match.
//
// Returns *goast.FuncDecl which is the matching function, or nil if the
// declaration is not a function, is a method, or has a different name.
func matchUnexportedFuncDecl(declaration goast.Decl, functionName string) *goast.FuncDecl {
	functionDeclaration, isFunc := declaration.(*goast.FuncDecl)
	if !isFunc {
		return nil
	}
	if functionDeclaration.Recv != nil || functionDeclaration.Name == nil {
		return nil
	}
	if functionDeclaration.Name.Name != functionName {
		return nil
	}
	return functionDeclaration
}
