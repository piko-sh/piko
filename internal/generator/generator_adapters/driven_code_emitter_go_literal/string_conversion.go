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
	"fmt"
	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// StringConverter provides conversion of Go expressions to their string
// representations. It enables mocking and testing of string conversion.
type StringConverter interface {
	// valueToString converts a Go expression to its string representation.
	//
	// Takes goExpr (goast.Expr) which is the expression to convert.
	// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides conversion context.
	//
	// Returns goast.Expr which is the string representation of the input expression.
	valueToString(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr
}

// stringConverter turns values into Go string literals.
// It implements the StringConverter interface.
type stringConverter struct {
	// A reference to the parent emitter is not needed here, as this component
	// is a pure, functional transformer of a `goast.Expr`.
}

var _ StringConverter = (*stringConverter)(nil)

// valueToString is the main entry point for this component. It generates
// Go code to convert an expression to a string, using type information from
// the annotator to select the best conversion method.
//
// Takes goExpr (goast.Expr) which is the expression to convert to a string.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type information
// for selecting the conversion method.
//
// Returns goast.Expr which is the generated Go code for the string conversion.
func (sc *stringConverter) valueToString(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	if ann == nil || ann.ResolvedType == nil {
		return callHelper(helperValueToString, goExpr)
	}

	if ann.IsPointerToStringable {
		return sc.emitPointerToStringerIIFE(goExpr, ann)
	}

	stringability := inspector_dto.StringabilityMethod(ann.Stringability)

	switch stringability {
	case inspector_dto.StringablePrimitive:
		return sc.convertPrimitiveToString(goExpr, ann.ResolvedType)

	case inspector_dto.StringableViaStringer:
		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: goExpr, Sel: cachedIdent("String")},
		}

	case inspector_dto.StringableViaTextMarshaler:
		return sc.emitTextMarshalerIIFE(goExpr)

	case inspector_dto.StringableViaPikoFormatter:
		return sc.emitPikoFormatterDefault(goExpr)

	case inspector_dto.StringableViaJSON:
		fallback := determineJSONFallback(ann.ResolvedType)
		return sc.emitJSONMarshalIIFE(goExpr, fallback)

	default:
		return callHelper(helperValueToString, goExpr)
	}
}

// emitPikoFormatterDefault generates the default string conversion for Piko
// maths types (Decimal, BigInt, Money). This emits expr.MustString().
//
// Takes goExpr (goast.Expr) which is the expression to convert.
//
// Returns goast.Expr which is the method call expression.
func (*stringConverter) emitPikoFormatterDefault(goExpr goast.Expr) goast.Expr {
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: goExpr, Sel: cachedIdent(mathsMustString)},
	}
}

// emitPointerToStringerIIFE generates a self-executing anonymous function
// that performs a nil-check before converting the underlying value to a
// string. This is the reflection-free implementation.
//
// Takes goExpr (goast.Expr) which is the pointer expression to convert.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type metadata.
//
// Returns goast.Expr which is the IIFE call expression.
//
// Panics if ann.ResolvedType.TypeExpr is not a pointer type.
func (sc *stringConverter) emitPointerToStringerIIFE(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	starExpr, ok := ann.ResolvedType.TypeExpression.(*goast.StarExpr)
	if !ok {
		panic(fmt.Sprintf("emitPointerToStringerIIFE called with non-pointer type: %T", ann.ResolvedType.TypeExpression))
	}

	ptrIdent := cachedIdent("_ptr")

	underlyingAnn := createUnderlyingTypeAnnotation(starExpr, ann)
	dereferencedExpr := &goast.UnaryExpr{Op: token.MUL, X: ptrIdent}
	stringifiedValueExpr := sc.valueToString(dereferencedExpr, underlyingAnn)

	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{
				Params:  &goast.FieldList{},
				Results: &goast.FieldList{List: []*goast.Field{{Type: cachedIdent(StringTypeName)}}},
			},
			Body: &goast.BlockStmt{
				List: []goast.Stmt{
					&goast.AssignStmt{
						Lhs: []goast.Expr{ptrIdent},
						Tok: token.DEFINE,
						Rhs: []goast.Expr{goExpr},
					},
					&goast.IfStmt{
						Cond: &goast.BinaryExpr{X: ptrIdent, Op: token.EQL, Y: cachedIdent("nil")},
						Body: &goast.BlockStmt{List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{strLit("")}}}},
					},
					&goast.ReturnStmt{Results: []goast.Expr{stringifiedValueExpr}},
				},
			},
		},
	}
}

// convertPrimitiveToString selects the correct, highly optimised strconv
// function or direct type cast for a given primitive Go type.
//
// Takes expression (goast.Expr) which is the expression to convert.
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which describes the primitive
// type being converted.
//
// Returns goast.Expr which is the conversion call expression.
func (*stringConverter) convertPrimitiveToString(expression goast.Expr, typeInfo *ast_domain.ResolvedTypeInfo) goast.Expr {
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression)
	if converter, ok := primitiveStringerMap[typeString]; ok {
		return converter(expression)
	}

	return callHelper(helperValueToString, expression)
}

// emitTextMarshalerIIFE generates a self-executing anonymous function (IIFE)
// to safely call MarshalText(), which returns ([]byte, error), and convert
// the result to a string.
//
// The generated code pattern is:
//
//	func() string {
//	    data, err := (goExpr).MarshalText()
//	    if err != nil {
//	        return ""
//	    }
//	    return string(data)
//	}()
//
// Takes goExpr (goast.Expr) which is the expression implementing TextMarshaler.
//
// Returns goast.Expr which is the IIFE call expression.
func (*stringConverter) emitTextMarshalerIIFE(goExpr goast.Expr) goast.Expr {
	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{
				Params:  &goast.FieldList{},
				Results: &goast.FieldList{List: []*goast.Field{{Type: cachedIdent(StringTypeName)}}},
			},
			Body: &goast.BlockStmt{
				List: []goast.Stmt{
					&goast.AssignStmt{
						Lhs: []goast.Expr{cachedIdent(varNameData), cachedIdent(varNameErr)},
						Tok: token.DEFINE,
						Rhs: []goast.Expr{&goast.CallExpr{Fun: &goast.SelectorExpr{X: goExpr, Sel: cachedIdent("MarshalText")}}},
					},
					&goast.IfStmt{
						Cond: &goast.BinaryExpr{X: cachedIdent(varNameErr), Op: token.NEQ, Y: cachedIdent("nil")},
						Body: &goast.BlockStmt{List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{strLit("")}}}},
					},
					&goast.ReturnStmt{Results: []goast.Expr{&goast.CallExpr{Fun: cachedIdent(StringTypeName), Args: []goast.Expr{cachedIdent(varNameData)}}}},
				},
			},
		},
	}
}

// emitJSONMarshalIIFE generates a self-executing anonymous function (IIFE)
// to safely call json.Marshal and convert the result to a string. This is used
// for maps and slices that need to be serialised to JSON for use in HTML
// attributes.
//
// The generated code pattern is:
//
//	func() string {
//	    data, err := json.Marshal(goExpr)
//	    if err != nil {
//	        return "{}" // or "[]" for slices
//	    }
//	    return string(data)
//	}()
//
// Takes goExpr (goast.Expr) which is the Go expression to marshal to JSON.
// Takes fallbackValue (string) which is the value to return on error, such as
// "{}" for maps or "[]" for slices.
//
// Returns goast.Expr which is the AST for the generated IIFE.
func (*stringConverter) emitJSONMarshalIIFE(goExpr goast.Expr, fallbackValue string) goast.Expr {
	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{
				Params:  &goast.FieldList{},
				Results: &goast.FieldList{List: []*goast.Field{{Type: cachedIdent(StringTypeName)}}},
			},
			Body: &goast.BlockStmt{
				List: []goast.Stmt{
					&goast.AssignStmt{
						Lhs: []goast.Expr{cachedIdent(varNameData), cachedIdent(varNameErr)},
						Tok: token.DEFINE,
						Rhs: []goast.Expr{&goast.CallExpr{
							Fun:  &goast.SelectorExpr{X: cachedIdent(pkgJSON), Sel: cachedIdent("Marshal")},
							Args: []goast.Expr{goExpr},
						}},
					},
					&goast.IfStmt{
						Cond: &goast.BinaryExpr{X: cachedIdent(varNameErr), Op: token.NEQ, Y: cachedIdent("nil")},
						Body: &goast.BlockStmt{List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{strLit(fallbackValue)}}}},
					},
					&goast.ReturnStmt{Results: []goast.Expr{&goast.CallExpr{Fun: cachedIdent(StringTypeName), Args: []goast.Expr{cachedIdent(varNameData)}}}},
				},
			},
		},
	}
}

// primitiveStringerMap is a dispatch table that maps a primitive Go type name
// to a function that generates the optimal go/ast expression for converting it
// to a string. This applies the Replace Conditional with Dispatch Table pattern.
var primitiveStringerMap = map[string]func(goast.Expr) goast.Expr{
	"string": func(e goast.Expr) goast.Expr { return e },
	"int": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatInt", &goast.CallExpr{Fun: cachedIdent(Int64TypeName), Args: []goast.Expr{e}}, intLit(10))
	},
	"int8": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatInt", &goast.CallExpr{Fun: cachedIdent(Int64TypeName), Args: []goast.Expr{e}}, intLit(10))
	},
	"int16": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatInt", &goast.CallExpr{Fun: cachedIdent(Int64TypeName), Args: []goast.Expr{e}}, intLit(10))
	},
	"int32": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatInt", &goast.CallExpr{Fun: cachedIdent(Int64TypeName), Args: []goast.Expr{e}}, intLit(10))
	},
	"int64": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatInt", e, intLit(10))
	},
	"uint": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatUint", &goast.CallExpr{Fun: cachedIdent("uint64"), Args: []goast.Expr{e}}, intLit(10))
	},
	"uint8": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatUint", &goast.CallExpr{Fun: cachedIdent("uint64"), Args: []goast.Expr{e}}, intLit(10))
	},
	"byte": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatUint", &goast.CallExpr{Fun: cachedIdent("uint64"), Args: []goast.Expr{e}}, intLit(10))
	},
	"uint16": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatUint", &goast.CallExpr{Fun: cachedIdent("uint64"), Args: []goast.Expr{e}}, intLit(10))
	},
	"uint32": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatUint", &goast.CallExpr{Fun: cachedIdent("uint64"), Args: []goast.Expr{e}}, intLit(10))
	},
	"uint64": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatUint", e, intLit(10))
	},
	"uintptr": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatUint", &goast.CallExpr{Fun: cachedIdent("uint64"), Args: []goast.Expr{e}}, intLit(10))
	},
	"float32": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatFloat", &goast.CallExpr{Fun: cachedIdent("float64"), Args: []goast.Expr{e}}, &goast.BasicLit{Kind: token.CHAR, Value: "'f'"}, intLit(-1), intLit(32))
	},
	"float64": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatFloat", e, &goast.BasicLit{Kind: token.CHAR, Value: "'f'"}, intLit(-1), intLit(64))
	},
	"bool": func(e goast.Expr) goast.Expr {
		return callStrconv("FormatBool", e)
	},
	"rune": func(e goast.Expr) goast.Expr {
		return &goast.CallExpr{Fun: cachedIdent(StringTypeName), Args: []goast.Expr{e}}
	},
}

// newStringConverter creates a new string converter.
//
// Returns *stringConverter which is the initialised converter instance.
func newStringConverter() *stringConverter {
	return &stringConverter{}
}

// createUnderlyingTypeAnnotation creates an annotation for the type that a
// pointer points to.
//
// Takes starExpr (*goast.StarExpr) which provides the pointer expression to
// extract the underlying type from.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the parent
// annotation to copy stringability settings from.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the type information
// for the underlying type.
func createUnderlyingTypeAnnotation(starExpr *goast.StarExpr, ann *ast_domain.GoGeneratorAnnotation) *ast_domain.GoGeneratorAnnotation {
	underlyingTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:       starExpr.X,
		PackageAlias:         getPackageAliasFromType(starExpr.X, ann.ResolvedType.PackageAlias),
		CanonicalPackagePath: "",
		IsSynthetic:          false,
	}

	return &ast_domain.GoGeneratorAnnotation{
		ResolvedType:            underlyingTypeInfo,
		Stringability:           ann.Stringability,
		EffectiveKeyExpression:  nil,
		PropDataSource:          nil,
		BaseCodeGenVarName:      nil,
		ParentTypeName:          nil,
		GeneratedSourcePath:     nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		OriginalPackageAlias:    nil,
		OriginalSourcePath:      nil,
		DynamicAttributeOrigins: nil,
		Symbol:                  nil,
		PartialInfo:             nil,
		Srcset:                  nil,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		StaticCollectionLiteral: nil,
		StaticCollectionData:    nil,
		DynamicCollectionInfo:   nil,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
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

// determineJSONFallback returns the correct empty JSON value for a given type.
// Maps return "{}", slices and arrays return "[]", and json.Marshaler types
// return "null" because the actual JSON type they produce is not known.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type details used to pick the fallback value.
//
// Returns string which is the JSON fallback value ("{}", "[]", or "null").
func determineJSONFallback(typeInfo *ast_domain.ResolvedTypeInfo) string {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return "null"
	}
	return getJSONFallbackForExpr(typeInfo.TypeExpression)
}

// getJSONFallbackForExpr returns a default JSON value for a type expression.
// It follows pointers to find the base type.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns string which is the JSON fallback value: "{}" for maps, "[]" for
// arrays, or "null" for other types.
func getJSONFallbackForExpr(typeExpr goast.Expr) string {
	switch t := typeExpr.(type) {
	case *goast.MapType:
		return "{}"
	case *goast.ArrayType:
		return "[]"
	case *goast.StarExpr:
		return getJSONFallbackForExpr(t.X)
	default:
		return "null"
	}
}

// callStrconv builds a call expression for a strconv package function.
//
// Takes functionName (string) which names the strconv function to call.
// Takes arguments (...goast.Expr) which are the arguments to pass to the function.
//
// Returns *goast.CallExpr which is the constructed function call expression.
func callStrconv(functionName string, arguments ...goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent(functionName)},
		Args: arguments,
	}
}
