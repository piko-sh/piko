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

// Defines the core type system structures for representing Go types within
// template expressions during compilation. Provides type information models
// including resolved types, stringability levels, and type metadata used
// throughout semantic analysis.

import (
	"context"
	"fmt"
	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// typeAny is the string for the Go "any" type keyword.
	typeAny = "any"

	// typeBool is the Go type name for boolean values.
	typeBool = "bool"

	// typeString is the name of the built-in string type in Go.
	typeString = "string"

	// typeInt is the name of the built-in int type.
	typeInt = "int"

	// typeInt64 is the Go type name for a signed 64-bit integer.
	typeInt64 = "int64"

	// typeFloat64 is the type name for float64 values.
	typeFloat64 = "float64"

	// typeNil is the type name used for nil values in type expressions.
	typeNil = "nil"

	// typeRune is the type name for rune values.
	typeRune = "rune"

	// typeFunction is the type name used to mark function declarations.
	typeFunction = "function"

	// rankNone is the rank for types that cannot be compared numerically.
	rankNone = 0

	// rankBool is the ranking weight for boolean type fields.
	rankBool = 1

	// rankSmallInt is the sort rank for small integer types.
	rankSmallInt = 2

	// rankMediumSmall is the ranking value for items with medium-small priority.
	rankMediumSmall = 3

	// rankMedium is the middle priority rank value.
	rankMedium = 4

	// rankStandard is the default ranking value for standard matches.
	rankStandard = 5

	// rankLargeInt is the sort rank for large integer types.
	rankLargeInt = 6

	// rankFloat32 is the sort order for the float32 type.
	rankFloat32 = 9

	// rankFloat64 is the priority rank for float64 when comparing types.
	rankFloat64 = 10

	// rankBigInt is the sort rank for big integer values.
	rankBigInt = 11

	// rankDecimal is the rank for decimal number literals in type precedence.
	rankDecimal = 12
)

const (
	// familyNone marks a type that is not numeric.
	familyNone = iota

	// familyStandard is the font family for standard text output.
	familyStandard

	// familyBigInt represents the type family for arbitrary-precision integers.
	familyBigInt

	// familyDecimal is the number family for decimal values.
	familyDecimal
)

// builtInHandler holds the validation and return type logic for a built-in
// function.
type builtInHandler struct {
	// ValidateArgs checks the arguments passed to a built-in function call.
	ValidateArgs func(tr *TypeResolver, ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location)

	// GetReturnType returns the result type for this built-in function handler.
	GetReturnType func(tr *TypeResolver, ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo
}

var (
	// builtInFunctions is the central registry for all special system functions.
	builtInFunctions = map[string]builtInHandler{
		"len": {
			ValidateArgs:  (*TypeResolver).validateLenCapArgs,
			GetReturnType: getLenCapReturnType,
		},
		"cap": {
			ValidateArgs:  (*TypeResolver).validateLenCapArgs,
			GetReturnType: getLenCapReturnType,
		},
		"append": {
			ValidateArgs:  (*TypeResolver).validateAppendArgs,
			GetReturnType: (*TypeResolver).getAppendReturnType,
		},
		"min": {
			ValidateArgs:  (*TypeResolver).validateMinMaxArgs,
			GetReturnType: getMinMaxReturnType,
		},
		"max": {
			ValidateArgs:  (*TypeResolver).validateMinMaxArgs,
			GetReturnType: getMinMaxReturnType,
		},
		"T": {
			ValidateArgs:  (*TypeResolver).validateTranslationFuncArgs,
			GetReturnType: getTranslationFuncReturnType,
		},
		"LT": {
			ValidateArgs:  (*TypeResolver).validateTranslationFuncArgs,
			GetReturnType: getTranslationFuncReturnType,
		},
		"F": {
			ValidateArgs:  (*TypeResolver).validateFormatFuncArgs,
			GetReturnType: getFormatFuncReturnType,
		},
		"LF": {
			ValidateArgs:  (*TypeResolver).validateFormatFuncArgs,
			GetReturnType: getFormatFuncReturnType,
		},
		"string": {
			ValidateArgs:  (*TypeResolver).validateStringCoercionArgs,
			GetReturnType: getStringReturnType,
		},
		"int": {
			ValidateArgs:  (*TypeResolver).validateIntCoercionArgs,
			GetReturnType: getIntReturnType,
		},
		"int64": {
			ValidateArgs:  (*TypeResolver).validateInt64CoercionArgs,
			GetReturnType: getInt64ReturnType,
		},
		"int32": {
			ValidateArgs:  (*TypeResolver).validateInt32CoercionArgs,
			GetReturnType: getInt32ReturnType,
		},
		"int16": {
			ValidateArgs:  (*TypeResolver).validateInt16CoercionArgs,
			GetReturnType: getInt16ReturnType,
		},
		"float": {
			ValidateArgs:  (*TypeResolver).validateFloatCoercionArgs,
			GetReturnType: getFloatReturnType,
		},
		"float64": {
			ValidateArgs:  (*TypeResolver).validateFloat64CoercionArgs,
			GetReturnType: getFloat64ReturnType,
		},
		"float32": {
			ValidateArgs:  (*TypeResolver).validateFloat32CoercionArgs,
			GetReturnType: getFloat32ReturnType,
		},
		"bool": {
			ValidateArgs:  (*TypeResolver).validateBoolCoercionArgs,
			GetReturnType: getBoolReturnType,
		},
		"decimal": {
			ValidateArgs:  (*TypeResolver).validateDecimalCoercionArgs,
			GetReturnType: getDecimalReturnType,
		},
		"bigint": {
			ValidateArgs:  (*TypeResolver).validateBigIntCoercionArgs,
			GetReturnType: getBigIntReturnType,
		},
	}

	// numericRankMap provides O(1) lookup for type promotion ranks.
	numericRankMap = map[string]int{
		"maths.Decimal": rankDecimal,
		"maths.BigInt":  rankBigInt,
		"float64":       rankFloat64,
		"float32":       rankFloat32,
		"int64":         rankLargeInt,
		"uint64":        rankLargeInt,
		"int":           rankStandard,
		"uint":          rankStandard,
		"uintptr":       rankStandard,
		"int32":         rankMedium,
		"uint32":        rankMedium,
		"rune":          rankMedium,
		"int16":         rankMediumSmall,
		"uint16":        rankMediumSmall,
		"int8":          rankSmallInt,
		"uint8":         rankSmallInt,
		"byte":          rankSmallInt,
		"bool":          rankBool,
	}
)

// determineInspectorContext finds the correct package and file context for
// inspector lookups.
//
// Takes ctx (*AnalysisContext) which provides the current analysis state.
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which contains the resolved
// type details to look up.
//
// Returns packagePath (string) which is the package path to use for lookups.
// Returns filePath (string) which is the file path where the type is defined.
func (tr *TypeResolver) determineInspectorContext(ctx *AnalysisContext, typeInfo *ast_domain.ResolvedTypeInfo) (packagePath, filePath string) {
	importerPackagePath := typeInfo.CanonicalPackagePath
	importerFilePath := ctx.CurrentGoSourcePath

	ctx.Logger.Trace("[stringability] Determining inspector context...",
		logger_domain.String("initial_pkg_path", importerPackagePath),
		logger_domain.String("initial_file_path", importerFilePath),
	)

	if importerPackagePath == "" {
		importerPackagePath = ctx.CurrentGoFullPackagePath
		ctx.Logger.Trace("[stringability] No canonical path on typeInfo, falling back to current context.",
			logger_domain.String("using_pkg_path", importerPackagePath),
		)
		return importerPackagePath, importerFilePath
	}

	ctx.Logger.Trace("[stringability] Canonical path found. Looking up type DTO to find its defining file.",
		logger_domain.String("type_to_find", goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)),
		logger_domain.String("in_pkg", importerPackagePath),
	)
	dto, _ := tr.inspector.ResolveExprToNamedTypeWithMemoization(context.Background(), typeInfo.TypeExpression, importerPackagePath, importerFilePath)
	if dto != nil && dto.DefinedInFilePath != "" {
		importerFilePath = dto.DefinedInFilePath
		ctx.Logger.Trace("[stringability] Found type DTO and its defining file.",
			logger_domain.String("resolved_file_path", importerFilePath),
		)
	} else {
		ctx.Logger.Trace("[stringability] Could not find specific DTO for type, proceeding with best-effort file path.",
			logger_domain.String("best_effort_file_path", importerFilePath),
		)
	}

	return importerPackagePath, importerFilePath
}

// checkPointerStringability checks if a pointer type can be converted to a
// string by unwrapping the pointer and checking its base type.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which contains the resolved
// type details for the pointer.
// Takes starExpr (*goast.StarExpr) which is the pointer type expression to
// check.
//
// Returns stringability (int) which shows the stringability level found.
// Returns isPointer (bool) which is true when the type is a pointer.
// Returns isStringable (bool) which is true when the base type can be turned
// into a string.
func (tr *TypeResolver) checkPointerStringability(ctx *AnalysisContext, typeInfo *ast_domain.ResolvedTypeInfo, starExpr *goast.StarExpr) (stringability int, isPointer, isStringable bool) {
	ctx.Logger.Trace("[stringability] Type is a pointer, unwrapping and recursing.",
		logger_domain.String("pointer_type", goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)),
	)

	elementTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          starExpr.X,
		PackageAlias:            getPackageAliasFromType(starExpr.X, typeInfo.PackageAlias),
		CanonicalPackagePath:    typeInfo.CanonicalPackagePath,
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}

	baseStringability, _ := tr.determineStringability(ctx, elementTypeInfo)
	if baseStringability != int(inspector_dto.StringableNone) {
		ctx.Logger.Trace("[stringability] Found stringability on pointer's base type.",
			logger_domain.Int("stringability_code", baseStringability),
		)
		return baseStringability, true, true
	}

	ctx.Logger.Trace("[stringability] Pointer's base type was not stringable, continuing check on pointer itself.")
	return 0, false, false
}

// determineStringability checks whether a type can be turned into a string.
// It looks for types that implement the Stringer interface or are otherwise
// convertible to a string.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which describes the type to
// check.
//
// Returns int which shows the stringability level of the type.
// Returns bool which is true when stringability is via a pointer receiver.
func (tr *TypeResolver) determineStringability(ctx *AnalysisContext, typeInfo *ast_domain.ResolvedTypeInfo) (int, bool) {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return int(inspector_dto.StringableNone), false
	}

	ctx.Logger.Trace("[stringability] Starting check",
		logger_domain.String("type_expr", goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)),
		logger_domain.String("type_info_pkg_alias", typeInfo.PackageAlias),
		logger_domain.String("type_info_canonical_path", typeInfo.CanonicalPackagePath),
	)

	if starExpr, isPointer := typeInfo.TypeExpression.(*goast.StarExpr); isPointer {
		if stringability, isViaPointer, found := tr.checkPointerStringability(ctx, typeInfo, starExpr); found {
			return stringability, isViaPointer
		}
	}

	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	if goastutil.IsPrimitiveOrBuiltin(typeString) {
		ctx.Logger.Trace("[stringability] Type is a primitive or built-in.",
			logger_domain.String("type", typeString),
		)
		return int(inspector_dto.StringablePrimitive), false
	}

	if tr.isJSONStringableType(ctx, typeInfo.TypeExpression) {
		ctx.Logger.Trace("[stringability] Type is a map or slice, using JSON stringability.",
			logger_domain.String("type", typeString),
		)
		return int(inspector_dto.StringableViaJSON), false
	}

	importerPackagePath, importerFilePath := tr.determineInspectorContext(ctx, typeInfo)

	ctx.Logger.Trace("[stringability] Calling ResolveExprToNamedTypeWithMemoization for final lookup.",
		logger_domain.String("type_expr", goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)),
		logger_domain.String("using_pkg_context", importerPackagePath),
		logger_domain.String("using_file_context", importerFilePath),
	)
	namedType, packageName := tr.inspector.ResolveExprToNamedTypeWithMemoization(
		context.Background(),
		typeInfo.TypeExpression,
		importerPackagePath,
		importerFilePath,
	)

	if namedType != nil {
		ctx.Logger.Trace("[stringability] SUCCESS: Inspector found named type DTO.",
			logger_domain.String("found_type_name", namedType.Name),
			logger_domain.String("found_in_pkg_alias", packageName),
			logger_domain.Int("stringability_from_dto", int(namedType.Stringability)),
		)
		return int(namedType.Stringability), false
	}

	ctx.Logger.Trace("[stringability] FAILURE: Inspector could not resolve a named type DTO. Type is not stringable.",
		logger_domain.String("type_expr", goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)),
	)
	return int(inspector_dto.StringableNone), false
}

// validateLenCapArgs checks arguments for len and cap built-in calls.
//
// Takes ctx (*AnalysisContext) which collects diagnostics.
// Takes callExpr (*ast_domain.CallExpression) which is the call to validate.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which holds argument
// types.
// Takes baseLocation (ast_domain.Location) which anchors diagnostic positions.
func (tr *TypeResolver) validateLenCapArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	functionName := callExpr.Callee.String()
	if len(argAnns) != 1 {
		finalLocation := baseLocation.Add(callExpr.GetRelativeLocation())
		msg := fmt.Sprintf("Built-in function '%s' expects exactly one argument", functionName)
		ctx.addDiagnosticForExpression(
			ast_domain.Error, msg, callExpr, finalLocation,
			callExpr.GoAnnotations, annotator_dto.CodeBuiltinFunctionError,
		)
		return
	}
	argAnn := argAnns[0]
	if argAnn == nil || argAnn.ResolvedType == nil {
		return
	}
	if !tr.isLenable(argAnn.ResolvedType) {
		typeName := goastutil.ASTToTypeString(argAnn.ResolvedType.TypeExpression, argAnn.ResolvedType.PackageAlias)
		message := fmt.Sprintf("Invalid argument for '%s': type '%s' is not an array, slice, map, or string", functionName, typeName)
		finalLocation := baseLocation.Add(callExpr.Args[0].GetRelativeLocation())
		ctx.addDiagnosticForExpression(ast_domain.Error, message, callExpr.Args[0], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeBuiltinFunctionError)
	}
}

// validateMinMaxArgs checks that min/max built-in calls have valid arguments.
// It verifies at least one argument is present, the first argument is an
// ordered type, and all subsequent arguments match the first argument's type.
//
// Takes ctx (*AnalysisContext) which collects diagnostics during validation.
// Takes callExpr (*ast_domain.CallExpression) which is the min or max
// call to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which provides type info
// for each argument.
// Takes baseLocation (ast_domain.Location) which is the base position for
// error reporting.
func (*TypeResolver) validateMinMaxArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	functionName := callExpr.Callee.String()
	if len(argAnns) < 1 {
		finalLocation := baseLocation.Add(callExpr.GetRelativeLocation())
		msg := fmt.Sprintf("Built-in function '%s' requires at least one argument", functionName)
		ctx.addDiagnosticForExpression(
			ast_domain.Error, msg, callExpr, finalLocation,
			callExpr.GoAnnotations, annotator_dto.CodeBuiltinFunctionError,
		)
		return
	}
	firstArgAnn := argAnns[0]
	if firstArgAnn == nil || firstArgAnn.ResolvedType == nil {
		return
	}
	if !areComparableForOrdering(firstArgAnn.ResolvedType, firstArgAnn.ResolvedType) {
		typeName := goastutil.ASTToTypeString(firstArgAnn.ResolvedType.TypeExpression, firstArgAnn.ResolvedType.PackageAlias)
		message := fmt.Sprintf("Invalid argument for '%s': type '%s' is not an ordered type (e.g., number or string)", functionName, typeName)
		finalLocation := baseLocation.Add(callExpr.Args[0].GetRelativeLocation())
		ctx.addDiagnosticForExpression(ast_domain.Error, message, callExpr.Args[0], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeBuiltinFunctionError)
		return
	}
	for i := 1; i < len(argAnns); i++ {
		nextArgAnn := argAnns[i] //nolint:gosec // loop bounded
		if !isAssignable(nextArgAnn.ResolvedType, firstArgAnn.ResolvedType) {
			sourceType := goastutil.ASTToTypeString(nextArgAnn.ResolvedType.TypeExpression, nextArgAnn.ResolvedType.PackageAlias)
			destType := goastutil.ASTToTypeString(firstArgAnn.ResolvedType.TypeExpression, firstArgAnn.ResolvedType.PackageAlias)
			message := fmt.Sprintf(
				"Mismatched types in call to '%s': all arguments must be of the same type, "+
					"but argument %d is type '%s' while first argument is type '%s'",
				functionName, i+1, sourceType, destType)
			finalLocation := baseLocation.Add(callExpr.Args[i].GetRelativeLocation())
			ctx.addDiagnosticForExpression(ast_domain.Error, message, callExpr.Args[i], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeBuiltinFunctionError)
		}
	}
}

// getAppendReturnType returns the return type for the built-in append
// function, which is the type of the first argument (the slice).
// Falls back to "any" when no argument type information is available.
//
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which provides
// resolved type information for each argument.
//
// Returns *ast_domain.ResolvedTypeInfo which is the slice type from the
// first argument, or "any" as a fallback.
func (*TypeResolver) getAppendReturnType(_ *AnalysisContext, _ *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	if len(argAnns) > 0 && argAnns[0] != nil && argAnns[0].ResolvedType != nil {
		return argAnns[0].ResolvedType
	}
	return newSimpleTypeInfo(goast.NewIdent(typeAny))
}

// validateAppendArgs checks that arguments to the built-in append function are
// valid.
//
// Takes ctx (*AnalysisContext) which collects diagnostics.
// Takes callExpr (*ast_domain.CallExpression) which is the append
// call to validate.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which provides resolved
// type information for each argument.
// Takes baseLocation (ast_domain.Location) which is the base position for error
// reporting.
func (tr *TypeResolver) validateAppendArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	if len(argAnns) < 1 {
		finalLocation := baseLocation.Add(callExpr.GetRelativeLocation())
		ctx.addDiagnosticForExpression(
			ast_domain.Error, "Built-in function 'append' requires at least one argument",
			callExpr, finalLocation,
			callExpr.GoAnnotations, annotator_dto.CodeBuiltinFunctionError,
		)
		return
	}
	sliceAnn := argAnns[0]
	sliceElementType, ok := tr.getSliceElementType(sliceAnn.ResolvedType)
	if !ok {
		typeName := goastutil.ASTToTypeString(sliceAnn.ResolvedType.TypeExpression, sliceAnn.ResolvedType.PackageAlias)
		message := fmt.Sprintf("Invalid first argument for 'append': type '%s' is not a slice", typeName)
		finalLocation := baseLocation.Add(callExpr.Args[0].GetRelativeLocation())
		ctx.addDiagnosticForExpression(ast_domain.Error, message, callExpr.Args[0], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeBuiltinFunctionError)
		return
	}
	for i := 1; i < len(argAnns); i++ {
		elementAnn := argAnns[i] //nolint:gosec // loop bounded
		if !isAssignable(elementAnn.ResolvedType, sliceElementType) {
			sourceType := goastutil.ASTToTypeString(elementAnn.ResolvedType.TypeExpression, elementAnn.ResolvedType.PackageAlias)
			destType := goastutil.ASTToTypeString(sliceElementType.TypeExpression, sliceElementType.PackageAlias)
			message := fmt.Sprintf("Cannot use type '%s' as a value of type '%s' in argument to 'append'", sourceType, destType)
			finalLocation := baseLocation.Add(callExpr.Args[i].GetRelativeLocation())
			ctx.addDiagnosticForExpression(ast_domain.Error, message, callExpr.Args[i], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeBuiltinFunctionError)
		}
	}
}

// isLenable reports whether the given type supports the len built-in.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which describes the type to
// check.
//
// Returns bool which is true for arrays, maps, and strings.
func (tr *TypeResolver) isLenable(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	underlyingType := tr.inspector.ResolveToUnderlyingAST(typeInfo.TypeExpression, typeInfo.PackageAlias)
	switch t := underlyingType.(type) {
	case *goast.ArrayType, *goast.MapType:
		return true
	case *goast.Ident:
		return t.Name == typeString
	}
	return false
}

// getSliceElementType extracts the element type from a slice type.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which describes the type to
// inspect.
//
// Returns *ast_domain.ResolvedTypeInfo which describes the slice element type.
// Returns bool which indicates whether typeInfo was a slice type.
func (tr *TypeResolver) getSliceElementType(typeInfo *ast_domain.ResolvedTypeInfo) (*ast_domain.ResolvedTypeInfo, bool) {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return nil, false
	}
	underlyingType := tr.inspector.ResolveToUnderlyingAST(typeInfo.TypeExpression, typeInfo.PackageAlias)
	if arrayType, isSlice := underlyingType.(*goast.ArrayType); isSlice && arrayType.Len == nil {
		packageAlias := getPackageAliasFromType(arrayType.Elt, typeInfo.PackageAlias)
		return &ast_domain.ResolvedTypeInfo{
			TypeExpression:          arrayType.Elt,
			PackageAlias:            packageAlias,
			CanonicalPackagePath:    typeInfo.CanonicalPackagePath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		}, true
	}
	return nil, false
}

// validateTranslationFuncArgs checks that T() and LT() are called with at
// least one string argument. These functions accept variadic arguments: the
// first is the translation key, and later arguments are fallback values if
// the key is not found.
//
// If a TranslationKeySet is available in the context, it also checks that
// the translation key exists and emits a warning if not.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and
// diagnostics collector.
// Takes callExpr (*ast_domain.CallExpression) which is the function call to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains type
// annotations for each argument.
// Takes baseLocation (ast_domain.Location) which is the base location for
// working out diagnostic positions.
func (*TypeResolver) validateTranslationFuncArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	functionName := callExpr.Callee.String()
	if len(argAnns) < 1 {
		finalLocation := baseLocation.Add(callExpr.GetRelativeLocation())
		msg := fmt.Sprintf("Built-in function '%s' expects at least one argument", functionName)
		ctx.addDiagnosticForExpression(
			ast_domain.Error, msg, callExpr, finalLocation,
			callExpr.GoAnnotations, annotator_dto.CodeTranslationFunctionError,
		)
		return
	}
	for i, argAnn := range argAnns {
		if argAnn == nil || argAnn.ResolvedType == nil {
			continue
		}
		if !isStringType(argAnn.ResolvedType) {
			typeName := goastutil.ASTToTypeString(argAnn.ResolvedType.TypeExpression, argAnn.ResolvedType.PackageAlias)
			argLabel := "key"
			if i > 0 {
				argLabel = fmt.Sprintf("fallback argument %d", i)
			}
			message := fmt.Sprintf("Invalid %s for '%s': expected a string, but got type '%s'", argLabel, functionName, typeName)
			finalLocation := baseLocation.Add(callExpr.Args[i].GetRelativeLocation())
			ctx.addDiagnosticForExpression(ast_domain.Error, message, callExpr.Args[i], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeTranslationFunctionError)
		}
	}

	validateTranslationKeyExists(ctx, callExpr, functionName, baseLocation)
}

// isJSONStringableType checks if a type expression is a map or slice that
// can be safely turned into JSON for use in HTML attributes.
//
// The method checks for safety by making sure that:
//   - Map keys must be strings (JSON requires this)
//   - Leaf values must be simple types OR types that can be turned into JSON
//
// Any nesting depth is accepted as long as the leaves are safe:
//   - map[string]map[string][]string is allowed (simple leaves)
//   - map[string]time.Time is allowed (time.Time has json.Marshaler)
//   - []uuid.UUID is allowed (uuid.UUID has json.Marshaler)
//
// But blocks recursive or unsafe structures:
//   - map[int]string is not allowed (key is not a string)
//   - []SomeStruct is not allowed (struct without json.Marshaler could loop)
//   - map[string]chan int is not allowed (channels cannot be turned into JSON)
//
// Takes ctx (*AnalysisContext) which provides inspector lookups.
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type can be safely turned into JSON.
func (tr *TypeResolver) isJSONStringableType(ctx *AnalysisContext, typeExpr goast.Expr) bool {
	if typeExpr == nil {
		return false
	}
	switch t := typeExpr.(type) {
	case *goast.MapType:
		if !isJSONSafeKeyType(t.Key) {
			return false
		}
		return tr.isSafeJSONLeafOrCollection(ctx, t.Value)
	case *goast.ArrayType:
		return tr.isSafeJSONLeafOrCollection(ctx, t.Elt)
	case *goast.StarExpr:
		return tr.isJSONStringableType(ctx, t.X)
	}
	return false
}

// isSafeJSONLeafOrCollection checks if a type can be safely turned into JSON.
//
// This checks for basic types, JSON-safe named types, or safe collections.
// It calls itself to check nested structures at any depth. Named types are
// checked through the inspector to see if they support JSON-compatible string
// output (TextMarshaler, PikoFormatter, and similar).
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type can be safely turned into JSON.
func (tr *TypeResolver) isSafeJSONLeafOrCollection(ctx *AnalysisContext, typeExpr goast.Expr) bool {
	if typeExpr == nil {
		return false
	}
	switch t := typeExpr.(type) {
	case *goast.Ident:
		if isJSONPrimitive(t.Name) {
			return true
		}
		return tr.isNamedTypeJSONSafe(ctx, typeExpr)
	case *goast.SelectorExpr:
		return tr.isNamedTypeJSONSafe(ctx, typeExpr)
	case *goast.StarExpr:
		return tr.isSafeJSONLeafOrCollection(ctx, t.X)
	case *goast.ArrayType:
		return tr.isSafeJSONLeafOrCollection(ctx, t.Elt)
	case *goast.MapType:
		if isJSONSafeKeyType(t.Key) {
			return tr.isSafeJSONLeafOrCollection(ctx, t.Value)
		}
	}
	return false
}

// isNamedTypeJSONSafe checks if a named type is safe for JSON serialisation.
// A type is safe if it implements json.Marshaler, TextMarshaler, or is a
// special Piko type with a known formatter.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and scope.
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is JSON-safe.
func (tr *TypeResolver) isNamedTypeJSONSafe(ctx *AnalysisContext, typeExpr goast.Expr) bool {
	importerPackagePath, importerFilePath := tr.determineInspectorContext(ctx, &ast_domain.ResolvedTypeInfo{
		TypeExpression:          typeExpr,
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	})

	namedType, _ := tr.inspector.ResolveExprToNamedTypeWithMemoization(
		context.Background(),
		typeExpr,
		importerPackagePath,
		importerFilePath,
	)

	if namedType == nil {
		return false
	}

	switch namedType.Stringability {
	case inspector_dto.StringableViaTextMarshaler,
		inspector_dto.StringableViaPikoFormatter,
		inspector_dto.StringableViaJSON:
		return true
	default:
		return false
	}
}

// newSimpleTypeInfo creates a ResolvedTypeInfo with only the type expression
// set and all other fields left at their zero values.
//
// Takes typeExpr (goast.Expr) which is the AST expression for the type.
//
// Returns *ast_domain.ResolvedTypeInfo with the type expression set.
func newSimpleTypeInfo(typeExpr goast.Expr) *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          typeExpr,
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// newSimpleTypeInfoWithAlias creates a ResolvedTypeInfo with a type expression
// and package alias.
//
// Takes typeExpr (goast.Expr) which is the AST expression for the type.
// Takes packageAlias (string) which is the package alias for the type.
//
// Returns *ast_domain.ResolvedTypeInfo with the type expression and alias set.
func newSimpleTypeInfoWithAlias(typeExpr goast.Expr, packageAlias string) *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          typeExpr,
		PackageAlias:            packageAlias,
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// getLenCapReturnType returns the type for the built-in len and cap
// functions.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the int type.
func getLenCapReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent(typeInt))
}

// getMinMaxReturnType defines the return type for the built-in min() and max()
// functions. The return type is always the type of the first argument.
//
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which provides
// type annotations for the function arguments.
//
// Returns *ast_domain.ResolvedTypeInfo which is the type of the first
// argument, or "any" if no arguments are available.
func getMinMaxReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	if len(argAnns) > 0 && argAnns[0] != nil && argAnns[0].ResolvedType != nil {
		return argAnns[0].ResolvedType
	}
	return newSimpleTypeInfo(goast.NewIdent(typeAny))
}

// substituteType walks a type expression and replaces generic type parameters
// with their concrete types from the substitution map.
//
// When expression is nil or substMap is empty, returns the original expression.
//
// Takes expression (goast.Expr) which is the type expression to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns goast.Expr which is the expression with type parameters replaced,
// or the original expression if no substitution applies.
func substituteType(expression goast.Expr, substMap map[string]goast.Expr) goast.Expr {
	if expression == nil || len(substMap) == 0 {
		return expression
	}

	switch n := expression.(type) {
	case *goast.Ident:
		return substituteIdent(n, substMap)
	case *goast.StarExpr:
		return substituteStarExpr(n, substMap)
	case *goast.ArrayType:
		return substituteArrayType(n, substMap)
	case *goast.MapType:
		return substituteMapType(n, substMap)
	case *goast.ChanType:
		return substituteChanType(n, substMap)
	case *goast.FuncType:
		return substituteFuncType(n, substMap)
	case *goast.IndexExpr:
		return substituteIndexExpr(n, substMap)
	case *goast.IndexListExpr:
		return substituteIndexListExpr(n, substMap)
	}
	return expression
}

// substituteIdent replaces a type parameter name with its concrete type.
//
// Takes n (*goast.Ident) which is the identifier to check.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their concrete type expressions.
//
// Returns goast.Expr which is the concrete type if found in the map, or the
// original identifier if not found.
func substituteIdent(n *goast.Ident, substMap map[string]goast.Expr) goast.Expr {
	if replacement, ok := substMap[n.Name]; ok {
		return replacement
	}
	return n
}

// substituteStarExpr replaces type parameters in a pointer type.
//
// Takes n (*goast.StarExpr) which is the pointer type to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns goast.Expr which is a new pointer with the inner type replaced, or
// the original if no change was needed.
func substituteStarExpr(n *goast.StarExpr, substMap map[string]goast.Expr) goast.Expr {
	newX := substituteType(n.X, substMap)
	if newX != n.X {
		return &goast.StarExpr{X: newX}
	}
	return n
}

// substituteArrayType replaces type parameters in array and slice types.
//
// Takes n (*goast.ArrayType) which is the array type node to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns goast.Expr which is a new array type with the element type replaced,
// or the original node if no change was needed.
func substituteArrayType(n *goast.ArrayType, substMap map[string]goast.Expr) goast.Expr {
	newElt := substituteType(n.Elt, substMap)
	if newElt != n.Elt {
		return &goast.ArrayType{Len: n.Len, Elt: newElt}
	}
	return n
}

// substituteMapType replaces type parameters in a map type with their concrete
// types.
//
// Takes n (*goast.MapType) which is the map type node to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns goast.Expr which is the map type with replaced key and value types,
// or the original node if no changes were made.
func substituteMapType(n *goast.MapType, substMap map[string]goast.Expr) goast.Expr {
	newKey := substituteType(n.Key, substMap)
	newValue := substituteType(n.Value, substMap)
	if newKey != n.Key || newValue != n.Value {
		return &goast.MapType{Key: newKey, Value: newValue}
	}
	return n
}

// substituteChanType replaces type parameters in a channel type with their
// concrete types.
//
// Takes n (*goast.ChanType) which is the channel type to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns goast.Expr which is a new channel type with the element type
// replaced, or the original if no change was needed.
func substituteChanType(n *goast.ChanType, substMap map[string]goast.Expr) goast.Expr {
	newValue := substituteType(n.Value, substMap)
	if newValue != n.Value {
		return &goast.ChanType{Dir: n.Dir, Value: newValue}
	}
	return n
}

// substituteFuncType replaces type parameters in a function type.
//
// Takes n (*goast.FuncType) which is the function type to update.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns goast.Expr which is a new function type with the replacements
// applied, or the original if no changes were needed.
func substituteFuncType(n *goast.FuncType, substMap map[string]goast.Expr) goast.Expr {
	newParams, paramsChanged := substituteFieldList(n.Params, substMap)
	newResults, resultsChanged := substituteFieldList(n.Results, substMap)

	if paramsChanged || resultsChanged {
		return &goast.FuncType{Func: n.Func, Params: newParams, Results: newResults}
	}
	return n
}

// substituteFieldList replaces types in a field list used for function
// parameters or results.
//
// Takes fieldList (*goast.FieldList) which is the list of fields to process.
// Takes substMap (map[string]goast.Expr) which maps type names to their
// replacement types.
//
// Returns *goast.FieldList which is the new field list with replacements.
// Returns bool which indicates whether any replacements were made.
func substituteFieldList(fieldList *goast.FieldList, substMap map[string]goast.Expr) (*goast.FieldList, bool) {
	if fieldList == nil {
		return nil, false
	}

	newFieldList := &goast.FieldList{Opening: fieldList.Opening, Closing: fieldList.Closing}
	changed := false

	for _, field := range fieldList.List {
		newField := *field
		newField.Type = substituteType(field.Type, substMap)
		if newField.Type != field.Type {
			changed = true
		}
		newFieldList.List = append(newFieldList.List, &newField)
	}
	return newFieldList, changed
}

// substituteIndexExpr replaces type parameters in a generic index expression
// with one type argument, such as Box[T].
//
// Takes n (*goast.IndexExpr) which is the index expression to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their replacement expressions.
//
// Returns goast.Expr which is a new expression with replacements applied, or
// the original if nothing changed.
func substituteIndexExpr(n *goast.IndexExpr, substMap map[string]goast.Expr) goast.Expr {
	newX := substituteType(n.X, substMap)
	newIndex := substituteType(n.Index, substMap)
	if newX != n.X || newIndex != n.Index {
		return &goast.IndexExpr{X: newX, Lbrack: n.Lbrack, Index: newIndex, Rbrack: n.Rbrack}
	}
	return n
}

// substituteIndexListExpr replaces type parameters in an index list
// expression for generic types with more than one parameter, such as
// Map[K, V].
//
// Takes n (*goast.IndexListExpr) which is the index list expression to
// process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns goast.Expr which is a new expression with the replacements applied,
// or the original if no changes were needed.
func substituteIndexListExpr(n *goast.IndexListExpr, substMap map[string]goast.Expr) goast.Expr {
	newX := substituteType(n.X, substMap)
	changed := newX != n.X
	newIndices := make([]goast.Expr, len(n.Indices))

	for i, index := range n.Indices {
		newIndices[i] = substituteType(index, substMap)
		if newIndices[i] != index {
			changed = true
		}
	}

	if changed {
		return &goast.IndexListExpr{X: newX, Lbrack: n.Lbrack, Indices: newIndices, Rbrack: n.Rbrack}
	}
	return n
}

// isAssignable checks whether a source type can be assigned to a destination
// type.
//
// Takes source (*ast_domain.ResolvedTypeInfo) which is the type being
// assigned.
// Takes destination (*ast_domain.ResolvedTypeInfo) which is the target type.
//
// Returns bool which is true if the assignment is valid.
func isAssignable(source, destination *ast_domain.ResolvedTypeInfo) bool {
	if source == nil || destination == nil || source.TypeExpression == nil || destination.TypeExpression == nil {
		return false
	}
	if isNilType(source) {
		return isComparableWithNil(destination)
	}
	if destIdent, ok := destination.TypeExpression.(*goast.Ident); ok && (destIdent.Name == typeAny || destIdent.Name == "interface{}") {
		return true
	}
	if isTypeParameter(destination) {
		return true
	}
	if isArithmeticType(source, destination) {
		return true
	}
	return goastutil.ASTToTypeString(source.TypeExpression, source.PackageAlias) == goastutil.ASTToTypeString(destination.TypeExpression, destination.PackageAlias)
}

// isTypeParameter checks whether the given type information represents a
// generic type parameter. Type parameters are usually single uppercase letters
// like T, E, S, K, V, or names starting with a tilde (~) for underlying type
// constraints.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type to check.
//
// Returns bool which is true if the type appears to be a type parameter.
func isTypeParameter(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}

	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	if len(typeString) > 0 && typeString[0] == '~' {
		return true
	}

	identifier, ok := typeInfo.TypeExpression.(*goast.Ident)
	if !ok {
		return false
	}

	name := identifier.Name
	if len(name) == 0 {
		return false
	}

	if len(name) == 1 && name[0] >= 'A' && name[0] <= 'Z' {
		return true
	}

	return false
}

// isStringType checks whether the given type is a string type.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type to check.
//
// Returns bool which is true if the type is a string, false otherwise.
func isStringType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	if identifier, ok := typeInfo.TypeExpression.(*goast.Ident); ok {
		return identifier.Name == typeString
	}
	return false
}

// isBoolLike checks whether the given type is a boolean type.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type to check.
//
// Returns bool which is true if the type is bool, false otherwise.
func isBoolLike(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	if identifier, ok := typeInfo.TypeExpression.(*goast.Ident); ok {
		return identifier.Name == "bool"
	}
	return false
}

// isMoneyType checks whether the given type is maths.Money.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type information to check.
//
// Returns bool which is true if the type is maths.Money, false otherwise.
func isMoneyType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	return goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias) == "maths.Money"
}

// isComparableWithNil checks whether the given type can be compared with nil.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which describes the type to
// check.
//
// Returns bool which is true for pointer, slice, map, interface, function, and
// channel types.
func isComparableWithNil(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	switch typeInfo.TypeExpression.(type) {
	case *goast.StarExpr, *goast.ArrayType, *goast.MapType, *goast.InterfaceType, *goast.FuncType, *goast.ChanType:
		return true
	}
	return false
}

// isNilType checks whether the given type represents the nil type.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the type to
// check.
//
// Returns bool which is true if the type is nil, false otherwise.
func isNilType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	if identifier, ok := typeInfo.TypeExpression.(*goast.Ident); ok {
		return identifier.Name == typeNil
	}
	return false
}

// getPackageAliasFromType extracts the package alias from a type expression.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
// Takes fallback (string) which is returned when no package alias is found.
//
// Returns string which is the package alias or the fallback value.
func getPackageAliasFromType(typeExpr goast.Expr, fallback string) string {
	if star, ok := typeExpr.(*goast.StarExpr); ok {
		return getPackageAliasFromType(star.X, fallback)
	}
	if selectorExpression, ok := typeExpr.(*goast.SelectorExpr); ok {
		if identifier, isIdent := selectorExpression.X.(*goast.Ident); isIdent {
			return identifier.Name
		}
	}
	return fallback
}

// getNumericFamily returns the numeric family for a type. It is used to check
// if types are compatible with each other.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type information to classify.
//
// Returns int which is the numeric family: familyDecimal, familyBigInt,
// familyStandard, or familyNone if the type is not numeric.
func getNumericFamily(typeInfo *ast_domain.ResolvedTypeInfo) int {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return familyNone
	}
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	switch typeString {
	case "maths.Decimal":
		return familyDecimal
	case "maths.BigInt":
		return familyBigInt
	}
	if isNumericType(typeInfo) {
		return familyStandard
	}
	return familyNone
}

// isNumericType checks whether a type is a standard Go numeric primitive,
// such as int, float64, or similar built-in number types.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type information to check.
//
// Returns bool which is true if the type is a numeric primitive.
func isNumericType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	identifier, ok := typeInfo.TypeExpression.(*goast.Ident)
	if !ok {
		return false
	}
	return goastutil.IsPrimitiveOrBuiltin(identifier.Name) && identifier.Name != typeString && identifier.Name != "bool"
}

// isStandardInteger reports whether a type is a standard Go integer type.
// It returns false for floating-point types (float32 and float64).
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type information to check.
//
// Returns bool which is true if the type is a standard integer type.
func isStandardInteger(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	switch typeString {
	case "float64", "float32":
		return false
	}
	return isNumericType(typeInfo)
}

// isArithmeticType checks if two types can be used together in arithmetic.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the type of the left
// operand.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the type of the right
// operand.
//
// Returns bool which is true if the two types can be safely used in an
// arithmetic operation.
func isArithmeticType(left, right *ast_domain.ResolvedTypeInfo) bool {
	if isMoneyType(left) || isMoneyType(right) {
		return areMoneyTypesCompatible(left, right)
	}

	return areNumericTypesCompatible(left, right)
}

// areNumericTypesCompatible checks if two numeric types can be used together
// in operations. It handles standard numeric types, BigInt, and Decimal
// families, but not Money.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the first type to check.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the second type to
// check.
//
// Returns bool which is true when the types can work together in numeric
// operations.
func areNumericTypesCompatible(left, right *ast_domain.ResolvedTypeInfo) bool {
	leftIsNumLike := isNumericLike(left)
	rightIsNumLike := isNumericLike(right)

	if leftIsNumLike && rightIsNumLike {
		return true
	}

	leftFamily := getNumericFamily(left)
	rightFamily := getNumericFamily(right)

	if leftFamily > familyNone && leftFamily == rightFamily {
		return true
	}

	if (leftFamily == familyDecimal && rightFamily > familyNone) || (rightFamily == familyDecimal && leftFamily > familyNone) {
		return true
	}

	if (leftFamily == familyBigInt && isStandardInteger(right)) || (rightFamily == familyBigInt && isStandardInteger(left)) {
		return true
	}

	return false
}

// areMoneyTypesCompatible checks if two types can be used together in
// arithmetic when at least one is a Money type.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the first type to check.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the second type to check.
//
// Returns bool which is true if the types can be used together for Money
// arithmetic. This is true when both types are Money, or when one is Money
// and the other is a number type.
func areMoneyTypesCompatible(left, right *ast_domain.ResolvedTypeInfo) bool {
	isLeftMoney := isMoneyType(left)
	isRightMoney := isMoneyType(right)

	if isLeftMoney && isRightMoney {
		return true
	}

	isLeftNumeric := getNumericFamily(left) > familyNone
	isRightNumeric := getNumericFamily(right) > familyNone
	if (isLeftMoney && isRightNumeric) || (isRightMoney && isLeftNumeric) {
		return true
	}

	return false
}

// areComparableForOrdering checks if two types can be compared using ordering
// operators such as >, <, >=, and <=.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the first type to check.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the second type to check.
//
// Returns bool which is true if both types support ordering comparisons.
func areComparableForOrdering(left, right *ast_domain.ResolvedTypeInfo) bool {
	if isStringType(left) && isStringType(right) {
		return true
	}
	return isArithmeticType(left, right)
}

// areComparableForEquality checks if two types can be compared using == or !=.
//
// Takes operator (ast_domain.BinaryOp) which specifies the equality operator
// being used.
// Takes left (*ast_domain.ResolvedTypeInfo) which is the left-hand type.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the right-hand type.
//
// Returns bool which is true if the types can be compared for equality.
func areComparableForEquality(operator ast_domain.BinaryOp, left, right *ast_domain.ResolvedTypeInfo) bool {
	if isNilComparisonValid(left, right) {
		return true
	}

	if operator == ast_domain.OpEq || operator == ast_domain.OpNe {
		return isAssignable(left, right) || isAssignable(right, left)
	}

	if areNumericLikeComparable(left, right) {
		return true
	}

	return isAssignable(left, right) || isAssignable(right, left)
}

// isNilComparisonValid checks whether a nil comparison between two operands
// is valid.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the type of the left
// operand.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the type of the right
// operand.
//
// Returns bool which is true when one operand is nil and the other can be
// compared with nil.
func isNilComparisonValid(left, right *ast_domain.ResolvedTypeInfo) bool {
	return (isNilType(left) && isComparableWithNil(right)) ||
		(isNilType(right) && isComparableWithNil(left))
}

// areNumericLikeComparable checks if two types can be compared as numbers.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the first type to check.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the second type to check.
//
// Returns bool which is true if both types are numeric, or if one is a string
// and the other is numeric for loose equality checks.
func areNumericLikeComparable(left, right *ast_domain.ResolvedTypeInfo) bool {
	if isNumericLike(left) && isNumericLike(right) {
		return true
	}
	return (isStringType(left) && isNumericLike(right)) ||
		(isNumericLike(left) && isStringType(right))
}

// isNumericLike reports whether the type is numeric or boolean-like.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which describes the type to
// check.
//
// Returns bool which is true if the type is numeric or boolean-like.
func isNumericLike(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	return isNumeric(typeInfo) || isBoolLike(typeInfo)
}

// isNumeric reports whether the given type is a numeric type.
//
// Takes typeInfo (*ResolvedTypeInfo) which describes the type to check.
//
// Returns bool which is true if the type is a built-in numeric type such as
// int, float64, or byte.
func isNumeric(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression)
	switch typeString {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "rune", "byte", "uintptr":
		return true
	}
	return false
}

// getTranslationFuncReturnType returns the type information for T() and LT()
// translation functions. These functions always return a string.
//
// Returns *ast_domain.ResolvedTypeInfo which holds the string type.
func getTranslationFuncReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent(typeString))
}

// validateTranslationKeyExists checks if a translation key exists in the
// loaded translations.
//
// When translation keys are not loaded, returns without checking. When the
// first argument is not a string literal, returns without checking.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and
// translation keys.
// Takes callExpr (*ast_domain.CallExpression) which is the function call to check.
// Takes functionName (string) which is the name of the translation function
// (T or LT).
// Takes baseLocation (ast_domain.Location) which is the source location for
// diagnostics.
func validateTranslationKeyExists(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, functionName string, baseLocation ast_domain.Location) {
	if ctx.TranslationKeys == nil {
		return
	}

	if len(callExpr.Args) == 0 {
		return
	}

	stringLit, ok := callExpr.Args[0].(*ast_domain.StringLiteral)
	if !ok {
		return
	}

	key := stringLit.Value
	hasFallback := len(callExpr.Args) > 1

	var keyExists bool
	var keySource string
	if functionName == "LT" {
		keyExists = ctx.TranslationKeys.HasLocalKey(key)
		keySource = "local"
	} else {
		keyExists = ctx.TranslationKeys.HasGlobalKey(key)
		if !keyExists {
			keyExists = ctx.TranslationKeys.HasLocalKey(key)
		}
		keySource = "global"
	}

	if !keyExists {
		finalLocation := baseLocation.Add(callExpr.Args[0].GetRelativeLocation())
		if hasFallback {
			message := fmt.Sprintf("Translation key %q not found in %s translations; fallback will be used", key, keySource)
			ctx.addDiagnosticForExpression(ast_domain.Info, message, callExpr.Args[0], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeTranslationFunctionError)
		} else {
			message := fmt.Sprintf("Translation key %q not found in %s translations; consider providing a fallback", key, keySource)
			ctx.addDiagnosticForExpression(ast_domain.Warning, message, callExpr.Args[0], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeTranslationFunctionError)
		}
	}
}

// isJSONPrimitive checks if a type name is a primitive that is safe for JSON.
// The type `any` is not included because it can hold values that cannot be
// turned into JSON at runtime, such as channels, functions, and complex
// numbers.
//
// Takes name (string) which is the type name to check.
//
// Returns bool which is true if the type is a JSON-safe primitive.
func isJSONPrimitive(name string) bool {
	switch name {
	case typeString, "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "byte", "rune":
		return true
	}
	return false
}

// isJSONSafeKeyType checks whether a type can be used as a JSON map key.
// JSON keys are always strings, but Go's json.Marshal converts some types
// on its own: strings are used as they are, integers become decimal strings,
// and types that use encoding.TextMarshaler call their MarshalText method.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is safe for use as a JSON map key.
func isJSONSafeKeyType(typeExpr goast.Expr) bool {
	identifier, ok := typeExpr.(*goast.Ident)
	if !ok {
		return false
	}
	switch identifier.Name {
	case typeString,
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	}
	return false
}

// isPointerToType checks whether destination is a pointer to the source type.
//
// Takes source (*ast_domain.ResolvedTypeInfo) which is the base type to check.
// Takes destination (*ast_domain.ResolvedTypeInfo) which is the type that may
// be a pointer to source.
//
// Returns bool which is true if destination is a pointer to the source type.
func isPointerToType(source, destination *ast_domain.ResolvedTypeInfo) bool {
	if source == nil || destination == nil || source.TypeExpression == nil || destination.TypeExpression == nil {
		return false
	}

	starExpr, ok := destination.TypeExpression.(*goast.StarExpr)
	if !ok {
		return false
	}

	sourceTypeString := goastutil.ASTToTypeString(source.TypeExpression, source.PackageAlias)

	destBaseTypeString := goastutil.ASTToTypeString(starExpr.X, destination.PackageAlias)

	return sourceTypeString == destBaseTypeString
}

// getNumericRank returns the rank for numeric type promotion.
// This is called after isArithmeticType has confirmed the operation is valid.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type information to rank.
//
// Returns int which is the numeric rank, or rankNone if the type is nil or
// not found in the rank map.
func getNumericRank(typeInfo *ast_domain.ResolvedTypeInfo) int {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return rankNone
	}

	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	if rank, ok := numericRankMap[typeString]; ok {
		return rank
	}
	return rankNone
}

// promoteNumericTypes returns the type with the higher numeric rank.
//
// It assumes the types have already been checked for compatibility by
// isArithmeticType.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the first type to compare.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the second type to
// compare.
//
// Returns *ast_domain.ResolvedTypeInfo which is the type with the higher rank.
func promoteNumericTypes(left, right *ast_domain.ResolvedTypeInfo) *ast_domain.ResolvedTypeInfo {
	if getNumericRank(left) >= getNumericRank(right) {
		return left
	}
	return right
}
