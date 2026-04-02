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

// Provides validation and return type resolution for coercion built-in
// functions. These functions convert values between types (string, int, float,
// bool, decimal, bigint).

import (
	"fmt"
	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
)

var (
	// coercibleToString lists types that can be converted to a string.
	// Supported types include all primitives, maths.Decimal, maths.BigInt,
	// time.Time, and any.
	coercibleToString = map[string]bool{
		"string":        true,
		"int":           true,
		"int8":          true,
		"int16":         true,
		"int32":         true,
		"int64":         true,
		"uint":          true,
		"uint8":         true,
		"uint16":        true,
		"uint32":        true,
		"uint64":        true,
		"float32":       true,
		"float64":       true,
		"bool":          true,
		"byte":          true,
		"rune":          true,
		"maths.Decimal": true,
		"maths.BigInt":  true,
		"time.Time":     true,
		"any":           true,
		"interface{}":   true,
	}

	// coercibleToInt lists types that can be converted to int types.
	coercibleToInt = map[string]bool{
		"int":           true,
		"int8":          true,
		"int16":         true,
		"int32":         true,
		"int64":         true,
		"uint":          true,
		"uint8":         true,
		"uint16":        true,
		"uint32":        true,
		"uint64":        true,
		"float32":       true,
		"float64":       true,
		"bool":          true,
		"byte":          true,
		"rune":          true,
		"string":        true,
		"maths.Decimal": true,
		"maths.BigInt":  true,
		"any":           true,
		"interface{}":   true,
	}

	// coercibleToFloat lists Go types that can be changed to float types.
	coercibleToFloat = map[string]bool{
		"int":           true,
		"int8":          true,
		"int16":         true,
		"int32":         true,
		"int64":         true,
		"uint":          true,
		"uint8":         true,
		"uint16":        true,
		"uint32":        true,
		"uint64":        true,
		"float32":       true,
		"float64":       true,
		"bool":          true,
		"byte":          true,
		"rune":          true,
		"string":        true,
		"maths.Decimal": true,
		"maths.BigInt":  true,
		"any":           true,
		"interface{}":   true,
	}

	// coercibleToBool lists types that can be changed to bool.
	coercibleToBool = map[string]bool{
		"string":        true,
		"int":           true,
		"int8":          true,
		"int16":         true,
		"int32":         true,
		"int64":         true,
		"uint":          true,
		"uint8":         true,
		"uint16":        true,
		"uint32":        true,
		"uint64":        true,
		"float32":       true,
		"float64":       true,
		"bool":          true,
		"byte":          true,
		"rune":          true,
		"maths.Decimal": true,
		"maths.BigInt":  true,
		"time.Time":     true,
		"any":           true,
		"interface{}":   true,
	}

	// coercibleToDecimal lists types that can be converted to maths.Decimal.
	// bool and time.Time are NOT coercible to decimal.
	coercibleToDecimal = map[string]bool{
		"int":           true,
		"int8":          true,
		"int16":         true,
		"int32":         true,
		"int64":         true,
		"uint":          true,
		"uint8":         true,
		"uint16":        true,
		"uint32":        true,
		"uint64":        true,
		"float32":       true,
		"float64":       true,
		"byte":          true,
		"rune":          true,
		"string":        true,
		"maths.Decimal": true,
		"maths.BigInt":  true,
		"any":           true,
		"interface{}":   true,
	}

	// coercibleToBigInt lists types that can be converted to maths.BigInt.
	// float types, bool, and time.Time are NOT coercible to bigint.
	coercibleToBigInt = map[string]bool{
		"int":           true,
		"int8":          true,
		"int16":         true,
		"int32":         true,
		"int64":         true,
		"uint":          true,
		"uint8":         true,
		"uint16":        true,
		"uint32":        true,
		"uint64":        true,
		"byte":          true,
		"rune":          true,
		"string":        true,
		"maths.Decimal": true,
		"maths.BigInt":  true,
		"any":           true,
		"interface{}":   true,
	}
)

// validateStringCoercionArgs checks that arguments to the string() function
// can be converted to strings.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which holds the argument
// annotations.
// Takes baseLocation (ast_domain.Location) which specifies where to report
// errors.
func (*TypeResolver) validateStringCoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "string", "string", coercibleToString)
}

// validateIntCoercionArgs validates arguments to the int() coercion function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains argument
// annotations.
// Takes baseLocation (ast_domain.Location) which specifies the source position
// for errors.
func (*TypeResolver) validateIntCoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "int", "int", coercibleToInt)
}

// validateInt64CoercionArgs validates arguments to the int64() coercion
// function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which are the argument
// annotations.
// Takes baseLocation (ast_domain.Location) which is the source location for
// error reporting.
func (*TypeResolver) validateInt64CoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "int64", "int64", coercibleToInt)
}

// validateInt32CoercionArgs validates arguments to the int32() coercion
// function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call expression to
// validate.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains the
// argument annotations.
// Takes baseLocation (ast_domain.Location) which specifies where errors should
// be reported.
func (*TypeResolver) validateInt32CoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "int32", "int32", coercibleToInt)
}

// validateInt16CoercionArgs validates arguments to the int16() coercion
// function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which holds argument
// annotations.
// Takes baseLocation (ast_domain.Location) which specifies the error location.
func (*TypeResolver) validateInt16CoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "int16", "int16", coercibleToInt)
}

// validateFloatCoercionArgs validates arguments to the float() coercion
// function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call expression to
// validate.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains the
// argument annotations.
// Takes baseLocation (ast_domain.Location) which specifies the base location
// for error reporting.
func (*TypeResolver) validateFloatCoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "float", "float64", coercibleToFloat)
}

// validateFloat64CoercionArgs validates arguments to the float64() coercion
// function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains the
// argument annotations.
// Takes baseLocation (ast_domain.Location) which specifies the source location
// for diagnostics.
func (*TypeResolver) validateFloat64CoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "float64", "float64", coercibleToFloat)
}

// validateFloat32CoercionArgs checks the arguments for a float32 type
// conversion call.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which holds the argument
// annotations.
// Takes baseLocation (ast_domain.Location) which specifies where to report
// errors.
func (*TypeResolver) validateFloat32CoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "float32", "float32", coercibleToFloat)
}

// validateBoolCoercionArgs validates arguments to the bool() coercion
// function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which are the argument
// annotations.
// Takes baseLocation (ast_domain.Location) which is the source location for
// diagnostics.
func (*TypeResolver) validateBoolCoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "bool", "bool", coercibleToBool)
}

// validateDecimalCoercionArgs validates arguments to the decimal() coercion
// function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains the
// argument annotations.
// Takes baseLocation (ast_domain.Location) which specifies the source location
// for diagnostics.
func (*TypeResolver) validateDecimalCoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "decimal", "maths.Decimal", coercibleToDecimal)
}

// validateBigIntCoercionArgs validates arguments to the bigint() coercion
// function.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes callExpr (*ast_domain.CallExpression) which is the call
// expression to check.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which are the argument
// annotations.
// Takes baseLocation (ast_domain.Location) which is the source location for
// error reporting.
func (*TypeResolver) validateBigIntCoercionArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "bigint", "maths.BigInt", coercibleToBigInt)
}

// validateCoercionArg is a shared validator that checks argument count and type
// compatibility for coercion functions.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and
// diagnostics.
// Takes callExpr (*ast_domain.CallExpression) which is the coercion function call.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which provides type info
// for arguments.
// Takes baseLocation (ast_domain.Location) which is the base position for error
// reporting.
// Takes functionName (string) which is the name of the coercion function.
// Takes targetType (string) which is the target type name for error messages.
// Takes coercibleTypes (map[string]bool) which lists valid source types.
func validateCoercionArg(
	ctx *AnalysisContext,
	callExpr *ast_domain.CallExpression,
	argAnns []*ast_domain.GoGeneratorAnnotation,
	baseLocation ast_domain.Location,
	functionName, targetType string,
	coercibleTypes map[string]bool,
) {
	if len(argAnns) != 1 {
		finalLocation := baseLocation.Add(callExpr.GetRelativeLocation())
		msg := fmt.Sprintf("Built-in function '%s' expects exactly one argument", functionName)
		ctx.addDiagnosticForExpression(
			ast_domain.Error, msg, callExpr, finalLocation,
			callExpr.GoAnnotations, annotator_dto.CodeCoercionError,
		)
		return
	}

	argAnn := argAnns[0]
	if argAnn == nil || argAnn.ResolvedType == nil {
		return
	}

	typeName := goastutil.ASTToTypeString(argAnn.ResolvedType.TypeExpression, argAnn.ResolvedType.PackageAlias)

	if !coercibleTypes[typeName] {
		message := fmt.Sprintf("Cannot coerce type '%s' to %s", typeName, targetType)
		finalLocation := baseLocation.Add(callExpr.Args[0].GetRelativeLocation())
		ctx.addDiagnosticForExpression(ast_domain.Error, message, callExpr.Args[0], finalLocation, callExpr.GoAnnotations, annotator_dto.CodeCoercionError)
	}
}

// getStringReturnType returns the type information for a string type
// conversion.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the built-in string
// type.
func getStringReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent(typeString))
}

// getIntReturnType returns the type info for the int type conversion.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the built-in int type.
func getIntReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent(typeInt))
}

// getInt64ReturnType returns the resolved type info for int64.
//
// Returns *ast_domain.ResolvedTypeInfo which holds the int64 type.
func getInt64ReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent(typeInt64))
}

// getInt32ReturnType returns the type info for the int32 type.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the int32 type.
func getInt32ReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent("int32"))
}

// getInt16ReturnType returns the type info for the int16 type conversion.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the int16 type.
func getInt16ReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent("int16"))
}

// getFloatReturnType returns type information for the float() coercion
// function, which produces float64, the standard Go floating-point type.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the float64 type.
func getFloatReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent(typeFloat64))
}

// getFloat64ReturnType returns the type info for the float64 type.
//
// Returns *ast_domain.ResolvedTypeInfo which describes the float64 type.
func getFloat64ReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent(typeFloat64))
}

// getFloat32ReturnType returns the type info for the float32 type.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the float32 type.
func getFloat32ReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent("float32"))
}

// getBoolReturnType returns the type for the bool coercion function.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the built-in bool type.
func getBoolReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return newSimpleTypeInfo(goast.NewIdent(typeBool))
}

// getDecimalReturnType returns the type for the decimal() coercion function.
//
// Returns *ast_domain.ResolvedTypeInfo which describes the maths.Decimal type.
func getDecimalReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          &goast.SelectorExpr{X: goast.NewIdent(packageAliasMaths), Sel: goast.NewIdent("Decimal")},
		PackageAlias:            packageAliasMaths,
		CanonicalPackagePath:    "piko.sh/piko/pkg/maths",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// getBigIntReturnType returns the type for the bigint() coercion function.
//
// Returns *ast_domain.ResolvedTypeInfo which describes the maths.BigInt type.
func getBigIntReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          &goast.SelectorExpr{X: goast.NewIdent(packageAliasMaths), Sel: goast.NewIdent("BigInt")},
		PackageAlias:            packageAliasMaths,
		CanonicalPackagePath:    "piko.sh/piko/pkg/maths",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// suggestCoercionFunction suggests an appropriate coercion function when types
// mismatch.
//
// Takes type1 (string) which is the first type in the mismatch.
// Takes type2 (string) which is the second type in the mismatch.
//
// Returns string which is a suggestion message, or empty if no suggestion
// applies.
func suggestCoercionFunction(type1, type2 string) string {
	numericTypes := map[string]bool{
		typeNameInt: true, typeNameInt8: true, typeNameInt16: true, typeNameInt32: true, typeNameInt64: true,
		typeNameUint: true, typeNameUint8: true, typeNameUint16: true, typeNameUint32: true, typeNameUint64: true,
		typeNameFloat32: true, typeNameFloat64: true, typeNameByte: true, typeNameRune: true,
		typeNameMathsDecimal: true, typeNameMathsBigInt: true,
	}

	if type1 == typeNameString && numericTypes[type2] {
		return "; consider using string() to convert the numeric value"
	}
	if type2 == typeNameString && numericTypes[type1] {
		return "; consider using string() to convert the numeric value"
	}

	if type1 == typeNameBool && type2 == typeNameString {
		return "; consider using string() to convert the boolean value"
	}
	if type2 == typeNameBool && type1 == typeNameString {
		return "; consider using string() to convert the boolean value"
	}

	if numericTypes[type1] && numericTypes[type2] {
		return "; consider using an explicit type conversion"
	}

	return ""
}
