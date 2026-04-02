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

// Provides code emission for coercion built-in functions (string, int, float,
// etc.). Generates optimised Go code based on source type, falling back to
// runtime helpers for any/interface{}.

import (
	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// coercionRuntimePackagePath is the import path for the piko runtime package.
	coercionRuntimePackagePath = "piko.sh/piko/wdk/runtime"
)

// CoercionEmitter handles code output for type conversion functions.
type CoercionEmitter struct {
	// ee provides access to the expression emitter for adding imports.
	ee *expressionEmitter
}

// emitCoercionCall generates Go code for a coercion function call.
//
// Takes functionName (string) which is the coercion function name.
// Takes argExpr (goast.Expr) which is the already-emitted argument expression.
// Takes argAnn (*ast_domain.GoGeneratorAnnotation) which provides type info for
// the argument.
//
// Returns goast.Expr which is the generated Go expression for the coercion.
func (ce *CoercionEmitter) emitCoercionCall(
	functionName string,
	_ *ast_domain.CallExpression,
	argExpr goast.Expr,
	argAnn *ast_domain.GoGeneratorAnnotation,
) goast.Expr {
	sourceType := ce.getSourceType(argAnn)

	switch functionName {
	case StringTypeName:
		return ce.emitStringCoercion(argExpr, sourceType)
	case IntTypeName:
		return ce.emitIntCoercion(argExpr, sourceType)
	case Int64TypeName:
		return ce.emitInt64Coercion(argExpr, sourceType)
	case Int32TypeName:
		return ce.emitInt32Coercion(argExpr, sourceType)
	case Int16TypeName:
		return ce.emitInt16Coercion(argExpr, sourceType)
	case "float", Float64TypeName:
		return ce.emitFloat64Coercion(argExpr, sourceType)
	case Float32TypeName:
		return ce.emitFloat32Coercion(argExpr, sourceType)
	case BoolTypeName:
		return ce.emitBoolCoercion(argExpr, sourceType)
	case "decimal":
		return ce.emitDecimalCoercion(argExpr, sourceType)
	case "bigint":
		return ce.emitBigIntCoercion(argExpr, sourceType)
	default:
		return argExpr
	}
}

// getSourceType extracts the type name from an annotation.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the annotation
// to extract the type from.
//
// Returns string which is the extracted type name, or "any" if the annotation
// is nil or has no type information.
func (*CoercionEmitter) getSourceType(ann *ast_domain.GoGeneratorAnnotation) string {
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return "any"
	}

	switch t := ann.ResolvedType.TypeExpression.(type) {
	case *goast.Ident:
		return t.Name
	case *goast.SelectorExpr:
		if x, ok := t.X.(*goast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return "any"
}

// emitStringCoercion generates code to convert a value to a string.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which specifies the source value's type.
//
// Returns goast.Expr which is the generated conversion expression.
func (ce *CoercionEmitter) emitStringCoercion(argExpr goast.Expr, sourceType string) goast.Expr {
	ce.ee.emitter.addImport(pkgStrconv, "")

	switch sourceType {
	case StringTypeName:
		return argExpr
	case IntTypeName:
		return ce.strconvCall(strconvItoa, argExpr)
	case Int64TypeName:
		return ce.strconvFormatIntCall(argExpr)
	case Int32TypeName, Int16TypeName, Int8TypeName:
		return ce.strconvFormatIntCall(ce.castTo(Int64TypeName, argExpr))
	case UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName:
		return ce.strconvFormatUintCall(ce.castTo(Uint64TypeName, argExpr))
	case Float64TypeName:
		return ce.strconvFormatFloat64Call(argExpr)
	case Float32TypeName:
		return ce.strconvFormatFloat32Call(argExpr)
	case BoolTypeName:
		return ce.strconvCall(strconvFormatBool, argExpr)
	case mathsDecimalTypeName, mathsBigIntTypeName:
		return ce.methodCall(argExpr, mathsMustString)
	default:
		return ce.emitRuntimeCoercionCall("CoerceToString", argExpr)
	}
}

// emitIntCoercion generates code to convert a value to int.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which identifies the type to convert from.
//
// Returns goast.Expr which is the converted expression.
func (ce *CoercionEmitter) emitIntCoercion(argExpr goast.Expr, sourceType string) goast.Expr {
	switch sourceType {
	case IntTypeName:
		return argExpr
	case Int64TypeName, Int32TypeName, Int16TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName,
		Float64TypeName, Float32TypeName:
		return ce.castTo(IntTypeName, argExpr)
	case BoolTypeName:
		return ce.emitBoolToIntIIFE(argExpr, IntTypeName)
	case StringTypeName:
		return ce.emitStringParseIIFE(argExpr, IntTypeName, bitSize64)
	case mathsDecimalTypeName, mathsBigIntTypeName:
		return ce.castTo(IntTypeName, ce.methodCall(argExpr, mathsMustInt64))
	default:
		return ce.emitRuntimeCoercionCall("CoerceToInt", argExpr)
	}
}

// emitInt64Coercion generates code to convert a value to int64.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which identifies the type being converted from.
//
// Returns goast.Expr which is the converted expression.
func (ce *CoercionEmitter) emitInt64Coercion(argExpr goast.Expr, sourceType string) goast.Expr {
	switch sourceType {
	case Int64TypeName:
		return argExpr
	case IntTypeName, Int32TypeName, Int16TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName,
		Float64TypeName, Float32TypeName:
		return ce.castTo(Int64TypeName, argExpr)
	case BoolTypeName:
		return ce.emitBoolToIntIIFE(argExpr, Int64TypeName)
	case StringTypeName:
		return ce.emitStringParseIIFE(argExpr, Int64TypeName, bitSize64)
	case mathsDecimalTypeName, mathsBigIntTypeName:
		return ce.methodCall(argExpr, mathsMustInt64)
	default:
		return ce.emitRuntimeCoercionCall("CoerceToInt64", argExpr)
	}
}

// emitInt32Coercion creates code to convert a value to int32.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which is the type to convert from.
//
// Returns goast.Expr which is the conversion expression.
func (ce *CoercionEmitter) emitInt32Coercion(argExpr goast.Expr, sourceType string) goast.Expr {
	switch sourceType {
	case Int32TypeName:
		return argExpr
	case IntTypeName, Int64TypeName, Int16TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName,
		Float64TypeName, Float32TypeName:
		return ce.castTo(Int32TypeName, argExpr)
	case BoolTypeName:
		return ce.emitBoolToIntIIFE(argExpr, Int32TypeName)
	case StringTypeName:
		return ce.emitStringParseIIFE(argExpr, Int32TypeName, bitSize32)
	case mathsDecimalTypeName, mathsBigIntTypeName:
		return ce.castTo(Int32TypeName, ce.methodCall(argExpr, mathsMustInt64))
	default:
		return ce.emitRuntimeCoercionCall("CoerceToInt32", argExpr)
	}
}

// emitInt16Coercion generates code to convert a value to int16.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which identifies the type being converted from.
//
// Returns goast.Expr which is the converted expression.
func (ce *CoercionEmitter) emitInt16Coercion(argExpr goast.Expr, sourceType string) goast.Expr {
	switch sourceType {
	case Int16TypeName:
		return argExpr
	case IntTypeName, Int64TypeName, Int32TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName,
		Float64TypeName, Float32TypeName:
		return ce.castTo(Int16TypeName, argExpr)
	case BoolTypeName:
		return ce.emitBoolToIntIIFE(argExpr, Int16TypeName)
	case StringTypeName:
		return ce.emitStringParseIIFE(argExpr, Int16TypeName, bitSize16)
	case mathsDecimalTypeName, mathsBigIntTypeName:
		return ce.castTo(Int16TypeName, ce.methodCall(argExpr, mathsMustInt64))
	default:
		return ce.emitRuntimeCoercionCall("CoerceToInt16", argExpr)
	}
}

// emitFloat64Coercion generates code to convert a value to float64.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which identifies the type being converted from.
//
// Returns goast.Expr which is the converted expression.
func (ce *CoercionEmitter) emitFloat64Coercion(argExpr goast.Expr, sourceType string) goast.Expr {
	switch sourceType {
	case Float64TypeName:
		return argExpr
	case IntTypeName, Int64TypeName, Int32TypeName, Int16TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName,
		Float32TypeName:
		return ce.castTo(Float64TypeName, argExpr)
	case BoolTypeName:
		return ce.emitBoolToFloatIIFE(argExpr, Float64TypeName)
	case StringTypeName:
		return ce.emitStringParseToFloatIIFE(argExpr, Float64TypeName, bitSize64)
	case mathsDecimalTypeName:
		return ce.methodCall(argExpr, mathsMustFloat64)
	case mathsBigIntTypeName:
		return ce.castTo(Float64TypeName, ce.methodCall(argExpr, mathsMustInt64))
	default:
		return ce.emitRuntimeCoercionCall("CoerceToFloat64", argExpr)
	}
}

// emitFloat32Coercion creates code to convert a value to float32.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which identifies the type to convert from.
//
// Returns goast.Expr which is the converted expression.
func (ce *CoercionEmitter) emitFloat32Coercion(argExpr goast.Expr, sourceType string) goast.Expr {
	switch sourceType {
	case Float32TypeName:
		return argExpr
	case IntTypeName, Int64TypeName, Int32TypeName, Int16TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName,
		Float64TypeName:
		return ce.castTo(Float32TypeName, argExpr)
	case BoolTypeName:
		return ce.emitBoolToFloatIIFE(argExpr, Float32TypeName)
	case StringTypeName:
		return ce.emitStringParseToFloatIIFE(argExpr, Float32TypeName, bitSize32)
	case mathsDecimalTypeName:
		return ce.castTo(Float32TypeName, ce.methodCall(argExpr, mathsMustFloat64))
	case mathsBigIntTypeName:
		return ce.castTo(Float32TypeName, ce.methodCall(argExpr, mathsMustInt64))
	default:
		return ce.emitRuntimeCoercionCall("CoerceToFloat32", argExpr)
	}
}

// emitBoolCoercion generates code to convert a value to bool.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which identifies the type being converted from.
//
// Returns goast.Expr which is the converted boolean expression.
func (ce *CoercionEmitter) emitBoolCoercion(argExpr goast.Expr, sourceType string) goast.Expr {
	switch sourceType {
	case BoolTypeName:
		return argExpr
	case StringTypeName:
		return ce.emitStringParseToBoolIIFE(argExpr)
	case IntTypeName, Int64TypeName, Int32TypeName, Int16TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName:
		return &goast.BinaryExpr{X: argExpr, Op: token.NEQ, Y: intLit(IntValueZero)}
	case Float64TypeName, Float32TypeName:
		return &goast.BinaryExpr{X: argExpr, Op: token.NEQ, Y: &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}}
	default:
		return ce.emitRuntimeCoercionCall("CoerceToBool", argExpr)
	}
}

// emitDecimalCoercion generates code to convert a value to maths.Decimal.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which identifies the type being converted from.
//
// Returns goast.Expr which is the conversion expression for the target type.
func (ce *CoercionEmitter) emitDecimalCoercion(argExpr goast.Expr, sourceType string) goast.Expr {
	ce.ee.emitter.addImport(mathsPackagePath, pkgMaths)

	switch sourceType {
	case mathsDecimalTypeName:
		return argExpr
	case Int64TypeName:
		return ce.mathsNewDecimalFromInt(argExpr)
	case IntTypeName, Int32TypeName, Int16TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName:
		return ce.mathsNewDecimalFromInt(ce.castTo(Int64TypeName, argExpr))
	case Float64TypeName:
		return ce.mathsNewDecimalFromFloat(argExpr)
	case Float32TypeName:
		return ce.mathsNewDecimalFromFloat(ce.castTo(Float64TypeName, argExpr))
	case StringTypeName:
		return ce.mathsConstructorCall(mathsNewDecimalFromString, argExpr)
	case mathsBigIntTypeName:
		return ce.methodCall(argExpr, mathsToDecimal)
	default:
		return ce.emitRuntimeCoercionCall("CoerceToDecimal", argExpr)
	}
}

// emitBigIntCoercion generates code to convert a value to maths.BigInt.
//
// Takes argExpr (goast.Expr) which is the expression to convert.
// Takes sourceType (string) which identifies the type being converted from.
//
// Returns goast.Expr which is the converted expression.
func (ce *CoercionEmitter) emitBigIntCoercion(argExpr goast.Expr, sourceType string) goast.Expr {
	ce.ee.emitter.addImport(mathsPackagePath, pkgMaths)

	switch sourceType {
	case mathsBigIntTypeName:
		return argExpr
	case Int64TypeName:
		return ce.mathsNewBigIntFromInt(argExpr)
	case IntTypeName, Int32TypeName, Int16TypeName, Int8TypeName,
		UintTypeName, Uint64TypeName, Uint32TypeName, Uint16TypeName, Uint8TypeName, ByteTypeName:
		return ce.mathsNewBigIntFromInt(ce.castTo(Int64TypeName, argExpr))
	case StringTypeName:
		return ce.mathsConstructorCall(mathsNewBigIntFromString, argExpr)
	case mathsDecimalTypeName:
		return ce.methodCall(argExpr, mathsToBigInt)
	default:
		return ce.emitRuntimeCoercionCall("CoerceToBigInt", argExpr)
	}
}

// emitRuntimeCoercionCall builds a call to a runtime coercion helper.
//
// Takes helperName (string) which is the name of the runtime helper function.
// Takes argExpr (goast.Expr) which is the expression to pass as the argument.
//
// Returns goast.Expr which is the call expression for the helper.
func (ce *CoercionEmitter) emitRuntimeCoercionCall(helperName string, argExpr goast.Expr) goast.Expr {
	ce.ee.emitter.addImport(coercionRuntimePackagePath, runtimePackageName)
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(helperName)},
		Args: []goast.Expr{argExpr},
	}
}

// castTo creates a type conversion expression.
//
// Takes targetType (string) which is the name of the type to convert to.
// Takes argExpr (goast.Expr) which is the expression to convert.
//
// Returns *goast.CallExpr which wraps the argument in a type conversion call.
func (*CoercionEmitter) castTo(targetType string, argExpr goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{Fun: cachedIdent(targetType), Args: []goast.Expr{argExpr}}
}

// methodCall generates a method call on an expression.
//
// Takes receiver (goast.Expr) which is the expression to call the method on.
// Takes method (string) which is the name of the method to call.
//
// Returns *goast.CallExpr which is the generated method call expression.
func (*CoercionEmitter) methodCall(receiver goast.Expr, method string) *goast.CallExpr {
	return &goast.CallExpr{Fun: &goast.SelectorExpr{X: receiver, Sel: cachedIdent(method)}}
}

// strconvCall generates a strconv function call.
//
// Takes functionName (string) which specifies the strconv function to call.
// Takes arguments (...goast.Expr) which provides the arguments for the call.
//
// Returns *goast.CallExpr which is the constructed function call expression.
func (*CoercionEmitter) strconvCall(functionName string, arguments ...goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent(functionName)},
		Args: arguments,
	}
}

// strconvFormatIntCall generates strconv.FormatInt(x, 10).
//
// Takes argExpr (goast.Expr) which is the expression to format as an integer.
//
// Returns *goast.CallExpr which is the generated function call AST node.
func (ce *CoercionEmitter) strconvFormatIntCall(argExpr goast.Expr) *goast.CallExpr {
	return ce.strconvCall(strconvFormatInt, argExpr, intLit(numericBaseDecimal))
}

// strconvFormatUintCall generates strconv.FormatUint(x, 10).
//
// Takes argExpr (goast.Expr) which is the unsigned integer expression to
// format.
//
// Returns *goast.CallExpr which is the generated function call.
func (ce *CoercionEmitter) strconvFormatUintCall(argExpr goast.Expr) *goast.CallExpr {
	return ce.strconvCall(strconvFormatUint, argExpr, intLit(numericBaseDecimal))
}

// strconvFormatFloat64Call generates strconv.FormatFloat(x, 'f', -1, 64).
//
// Takes argExpr (goast.Expr) which is the float64 value to format.
//
// Returns *goast.CallExpr which is the generated FormatFloat call.
func (ce *CoercionEmitter) strconvFormatFloat64Call(argExpr goast.Expr) *goast.CallExpr {
	return ce.strconvCall(strconvFormatFloat, argExpr,
		&goast.BasicLit{Kind: token.CHAR, Value: "'f'"},
		&goast.UnaryExpr{Op: token.SUB, X: intLit(IntValueOne)},
		intLit(bitSize64))
}

// strconvFormatFloat32Call generates strconv.FormatFloat(float64(x), 'f', -1,
// 32).
//
// Takes argExpr (goast.Expr) which is the expression to format as a float32.
//
// Returns *goast.CallExpr which is the generated strconv.FormatFloat call.
func (ce *CoercionEmitter) strconvFormatFloat32Call(argExpr goast.Expr) *goast.CallExpr {
	return ce.strconvCall(strconvFormatFloat, ce.castTo(Float64TypeName, argExpr),
		&goast.BasicLit{Kind: token.CHAR, Value: "'f'"},
		&goast.UnaryExpr{Op: token.SUB, X: intLit(IntValueOne)},
		intLit(bitSize32))
}

// mathsNewDecimalFromInt creates a maths.NewDecimalFromInt(x) call expression.
//
// Takes argExpr (goast.Expr) which is the integer expression to convert.
//
// Returns *goast.CallExpr which is the created constructor call.
func (ce *CoercionEmitter) mathsNewDecimalFromInt(argExpr goast.Expr) *goast.CallExpr {
	return ce.mathsConstructorCall(mathsNewDecimalFromInt, argExpr)
}

// mathsNewDecimalFromFloat creates a maths.NewDecimalFromFloat(x) call.
//
// Takes argExpr (goast.Expr) which is the float expression to wrap.
//
// Returns *goast.CallExpr which is the generated constructor call.
func (ce *CoercionEmitter) mathsNewDecimalFromFloat(argExpr goast.Expr) *goast.CallExpr {
	return ce.mathsConstructorCall(mathsNewDecimalFromFloat, argExpr)
}

// mathsNewBigIntFromInt generates maths.NewBigIntFromInt(x).
//
// Takes argExpr (goast.Expr) which is the integer expression to wrap.
//
// Returns *goast.CallExpr which is the generated constructor call.
func (ce *CoercionEmitter) mathsNewBigIntFromInt(argExpr goast.Expr) *goast.CallExpr {
	return ce.mathsConstructorCall(mathsNewBigIntFromInt, argExpr)
}

// mathsConstructorCall builds a call expression for a maths package function.
//
// Takes functionName (string) which specifies the function name to call.
// Takes argExpr (goast.Expr) which provides the argument to pass.
//
// Returns *goast.CallExpr which is the built call expression.
func (*CoercionEmitter) mathsConstructorCall(functionName string, argExpr goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(pkgMaths), Sel: cachedIdent(functionName)},
		Args: []goast.Expr{argExpr},
	}
}

// emitBoolToIntIIFE generates an IIFE for bool to int conversion.
// Generates: func() T { if b { return 1 } return 0 }().
//
// Takes argExpr (goast.Expr) which is the boolean expression to convert.
// Takes targetType (string) which specifies the target integer type name.
//
// Returns goast.Expr which is the IIFE call expression.
func (*CoercionEmitter) emitBoolToIntIIFE(argExpr goast.Expr, targetType string) goast.Expr {
	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{Results: &goast.FieldList{List: []*goast.Field{{Type: cachedIdent(targetType)}}}},
			Body: &goast.BlockStmt{List: []goast.Stmt{
				&goast.IfStmt{
					Cond: argExpr,
					Body: &goast.BlockStmt{List: []goast.Stmt{
						&goast.ReturnStmt{Results: []goast.Expr{intLit(IntValueOne)}},
					}},
				},
				&goast.ReturnStmt{Results: []goast.Expr{intLit(IntValueZero)}},
			}},
		},
	}
}

// emitBoolToFloatIIFE generates an IIFE for bool to float conversion.
//
// Takes argExpr (goast.Expr) which is the boolean expression to convert.
// Takes targetType (string) which specifies the target float type name.
//
// Returns goast.Expr which is the IIFE that returns 1.0 for true or 0.0 for
// false.
func (*CoercionEmitter) emitBoolToFloatIIFE(argExpr goast.Expr, targetType string) goast.Expr {
	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{Results: &goast.FieldList{List: []*goast.Field{{Type: cachedIdent(targetType)}}}},
			Body: &goast.BlockStmt{List: []goast.Stmt{
				&goast.IfStmt{
					Cond: argExpr,
					Body: &goast.BlockStmt{List: []goast.Stmt{
						&goast.ReturnStmt{Results: []goast.Expr{&goast.BasicLit{Kind: token.FLOAT, Value: "1.0"}}},
					}},
				},
				&goast.ReturnStmt{Results: []goast.Expr{&goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}}},
			}},
		},
	}
}

// emitStringParseIIFE generates an immediately invoked function expression that
// parses a string to an integer type.
//
// Takes argExpr (goast.Expr) which is the string expression to parse.
// Takes targetType (string) which specifies the target integer type name.
// Takes bitSize (int) which specifies the bit size for strconv.ParseInt.
//
// Returns goast.Expr which represents an IIFE in the form:
// func() T { v, _ := strconv.ParseInt(s, 10, bitSize); return T(v) }().
func (ce *CoercionEmitter) emitStringParseIIFE(argExpr goast.Expr, targetType string, bitSize int) goast.Expr {
	ce.ee.emitter.addImport(pkgStrconv, "")

	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{Results: &goast.FieldList{List: []*goast.Field{{Type: cachedIdent(targetType)}}}},
			Body: &goast.BlockStmt{List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cachedIdent(varNameV), cachedIdent(BlankIdentifier)},
					Tok: token.DEFINE,
					Rhs: []goast.Expr{ce.strconvCall(strconvParseInt, argExpr,
						intLit(numericBaseDecimal), intLit(bitSize))},
				},
				&goast.ReturnStmt{Results: []goast.Expr{ce.castTo(targetType, cachedIdent(varNameV))}},
			}},
		},
	}
}

// emitStringParseToFloatIIFE generates an IIFE for string parsing to float
// types. Generates: func() T { v, _ := strconv.ParseFloat(s, bitSize);
// return T(v) }().
//
// Takes argExpr (goast.Expr) which is the string expression to parse.
// Takes targetType (string) which specifies the target float type name.
// Takes bitSize (int) which specifies the bit size for parsing (32 or 64).
//
// Returns goast.Expr which is the IIFE call expression.
func (ce *CoercionEmitter) emitStringParseToFloatIIFE(argExpr goast.Expr, targetType string, bitSize int) goast.Expr {
	ce.ee.emitter.addImport(pkgStrconv, "")

	returnExpr := goast.Expr(cachedIdent(varNameV))
	if targetType == Float32TypeName {
		returnExpr = ce.castTo(targetType, cachedIdent(varNameV))
	}

	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{Results: &goast.FieldList{List: []*goast.Field{{Type: cachedIdent(targetType)}}}},
			Body: &goast.BlockStmt{List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cachedIdent(varNameV), cachedIdent(BlankIdentifier)},
					Tok: token.DEFINE,
					Rhs: []goast.Expr{ce.strconvCall(strconvParseFloat, argExpr, intLit(bitSize))},
				},
				&goast.ReturnStmt{Results: []goast.Expr{returnExpr}},
			}},
		},
	}
}

// emitStringParseToBoolIIFE builds an IIFE that converts a string to a bool.
//
// Takes argExpr (goast.Expr) which is the string expression to parse.
//
// Returns goast.Expr which is the IIFE that parses the string to a bool.
func (ce *CoercionEmitter) emitStringParseToBoolIIFE(argExpr goast.Expr) goast.Expr {
	ce.ee.emitter.addImport(pkgStrconv, "")

	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{Results: &goast.FieldList{List: []*goast.Field{{Type: cachedIdent(BoolTypeName)}}}},
			Body: &goast.BlockStmt{List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cachedIdent(varNameV), cachedIdent(BlankIdentifier)},
					Tok: token.DEFINE,
					Rhs: []goast.Expr{ce.strconvCall(strconvParseBool, argExpr)},
				},
				&goast.ReturnStmt{Results: []goast.Expr{cachedIdent(varNameV)}},
			}},
		},
	}
}

// newCoercionEmitter creates a new coercion emitter.
//
// Takes ee (*expressionEmitter) which provides the expression emitter to wrap.
//
// Returns *CoercionEmitter which is ready for use.
func newCoercionEmitter(ee *expressionEmitter) *CoercionEmitter {
	return &CoercionEmitter{ee: ee}
}
