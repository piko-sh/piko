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

package driven_code_emitter_go_literal

import (
	goast "go/ast"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// isBoolType reports whether a Go AST type expression represents a bool type.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the expression is a bool type identifier.
func isBoolType(typeExpr goast.Expr) bool {
	if typeExpr == nil {
		return false
	}
	if identifier, ok := typeExpr.(*goast.Ident); ok {
		return identifier.Name == "bool"
	}
	return false
}

// isStringType checks whether a Go AST type expression is a string type.
//
// Takes expression (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the expression represents a string type.
func isStringType(expression goast.Expr) bool {
	if identifier, ok := expression.(*goast.Ident); ok {
		return identifier.Name == StringTypeName
	}
	return false
}

// isNillableType checks if a type expression represents a type that can be nil.
//
// Nillable types include pointers, slices, maps, functions, channels, and
// interfaces. Arrays are not nillable, but slices are.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type can be nil.
func isNillableType(typeExpr goast.Expr) bool {
	if typeExpr == nil {
		return true
	}
	switch t := typeExpr.(type) {
	case *goast.StarExpr, *goast.MapType, *goast.FuncType, *goast.ChanType, *goast.InterfaceType:
		return true
	case *goast.ArrayType:
		return t.Len == nil
	default:
		return false
	}
}

// isBasicGoType checks if a type expression is a built-in Go type that does
// not need an import (such as string, int, bool). This helps avoid adding
// imports for basic types.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is a built-in basic type.
func isBasicGoType(typeExpr goast.Expr) bool {
	if typeExpr == nil {
		return false
	}
	identifier, ok := typeExpr.(*goast.Ident)
	if !ok {
		return false
	}
	switch identifier.Name {
	case BoolTypeName, StringTypeName, "error", "any",
		IntTypeName, "int8", "int16", "int32", Int64TypeName,
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128",
		"byte", "rune", "uintptr":
		return true
	}
	return false
}

// isNumeric reports whether the given type is a numeric primitive.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the type to
// check.
//
// Returns bool which is true when the type is numeric, such as int, float, or
// byte.
func isNumeric(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	switch typeString {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "rune", "byte", "uintptr":
		return true
	}
	return false
}

// isExpressionStringType checks whether the given resolved type info
// represents a string type. This is the type-info version of isStringType.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type to check.
//
// Returns bool which is true if the type is a string, or false otherwise.
func isExpressionStringType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	if identifier, ok := typeInfo.TypeExpression.(*goast.Ident); ok {
		return identifier.Name == StringTypeName
	}
	return false
}

// shouldSkipEscaping checks if HTML escaping can be skipped for a value based
// on its type. Primitive non-string types like int and bool do not need HTML
// escaping because they cannot contain HTML special characters.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type information
// for the value being checked.
//
// Returns bool which is true when the value's type is a safe primitive that
// does not require HTML escaping.
func shouldSkipEscaping(ann *ast_domain.GoGeneratorAnnotation) bool {
	if ann == nil || ann.ResolvedType == nil {
		return false
	}
	if inspector_dto.StringabilityMethod(ann.Stringability) != inspector_dto.StringablePrimitive {
		return false
	}
	if typeIdent, ok := ann.ResolvedType.TypeExpression.(*goast.Ident); ok {
		switch typeIdent.Name {
		case StringTypeName, "rune":
			return false
		default:
			return true
		}
	}
	return false
}

// isSyntheticAnnotation checks whether an annotation contains a synthetic type.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the annotation
// to check.
//
// Returns bool which is true if the annotation's resolved type is synthetic.
func isSyntheticAnnotation(ann *ast_domain.GoGeneratorAnnotation) bool {
	if ann == nil || ann.ResolvedType == nil {
		return false
	}
	return ann.ResolvedType.IsSynthetic
}

// getSyntheticTypeName returns a type name suitable for error messages.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides
// the synthetic type.
//
// Returns string containing the type name for error reporting.
func getSyntheticTypeName(typeInfo *ast_domain.ResolvedTypeInfo) string {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return "unknown"
	}
	if identifier, ok := typeInfo.TypeExpression.(*goast.Ident); ok {
		return identifier.Name
	}
	return goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
}
