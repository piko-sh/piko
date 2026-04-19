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
	"go/token"

	"piko.sh/piko/internal/goastutil"
)

// nilLiteral is the string "nil" used to create nil expressions in Go code.
const nilLiteral = "nil"

// defineAndAssign creates a short variable declaration statement (varName :=
// rightHandSide). Used to create new local variables.
//
// Takes varName (string) which is the name for the new variable.
// Takes rightHandSide (goast.Expr) which is the expression to assign to it.
//
// Returns *goast.AssignStmt which is the declaration statement.
func defineAndAssign(varName string, rightHandSide goast.Expr) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{cachedIdent(varName)},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{rightHandSide},
	}
}

// assignExpression creates a Go assignment statement that assigns a value to a
// variable.
//
// Takes varName (string) which is the name of the variable to assign to.
// Takes rightHandSide (goast.Expr) which is the expression to assign.
//
// Returns *goast.AssignStmt which is the assignment statement.
func assignExpression(varName string, rightHandSide goast.Expr) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{cachedIdent(varName)},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{rightHandSide},
	}
}

// appendToSlice creates a Go assignment statement of the form
// `slice = append(slice, element)` for adding items to slices. It accepts
// goast.Expr for both parameters, allowing appends to complex expressions
// like `tempVar1.Children` or `tempVar2.Attributes`.
//
// Takes sliceExpr (goast.Expr) which is the slice to append to.
// Takes elemExpr (goast.Expr) which is the element to add.
//
// Returns *goast.AssignStmt which is the generated assignment statement.
func appendToSlice(sliceExpr goast.Expr, elemExpr goast.Expr) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{sliceExpr},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{
			&goast.CallExpr{
				Fun:  cachedIdent("append"),
				Args: []goast.Expr{sliceExpr, elemExpr},
			},
		},
	}
}

// strLit creates a Go AST string literal from a string value.
// This is a thin wrapper around goastutil.StrLit.
//
// Takes s (string) which is the value to convert to an AST literal.
//
// Returns *goast.BasicLit which is the AST node for the string.
func strLit(s string) *goast.BasicLit {
	return goastutil.StrLit(s)
}

// intLit creates a Go AST integer literal from an int value.
// This is a thin wrapper around goastutil.IntLit.
//
// Takes i (int) which is the integer value to convert.
//
// Returns *goast.BasicLit which is the AST node for the integer.
func intLit(i int) *goast.BasicLit {
	return goastutil.IntLit(i)
}

// callHelper creates a function call to a helper in the generator_helpers
// package. This is the standard way to call runtime helper functions.
//
// Takes functionName (string) which is the name of the helper function to call.
// Takes arguments (...goast.Expr) which are the arguments to pass to the function.
//
// Returns *goast.CallExpr which is the built function call expression.
func callHelper(functionName string, arguments ...goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(runtimePackageName),
			Sel: cachedIdent(functionName),
		},
		Args: arguments,
	}
}

// callHelperArena constructs a call to a pikoruntime arena-aware helper.
// It automatically prepends the arena variable as the first argument.
//
// This is used for arena-based ByteBuf functions that take arena as their
// first parameter to eliminate sync.Pool allocations.
//
// Takes functionName (string) which is the name of the
// arena-aware helper function.
// Takes arguments (...goast.Expr) which are the arguments after
// the arena parameter.
//
// Returns *goast.CallExpr which is the built function call expression.
func callHelperArena(functionName string, arguments ...goast.Expr) *goast.CallExpr {
	arenaArgs := make([]goast.Expr, 0, len(arguments)+1)
	arenaArgs = append(arenaArgs, cachedIdent(arenaVarName))
	arenaArgs = append(arenaArgs, arguments...)
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(runtimePackageName),
			Sel: cachedIdent(functionName),
		},
		Args: arenaArgs,
	}
}

// wrapInTruthinessCall wraps an expression in a runtime helper call for
// JavaScript-like truthiness checks.
//
// The control flow emitter decides when to use this based on type notes. If
// the type is already a bool, the wrapper is not called.
//
// Takes expression (goast.Expr) which is the expression to wrap.
//
// Returns goast.Expr which is a call expression that invokes
// EvaluateTruthiness.
func wrapInTruthinessCall(expression goast.Expr) goast.Expr {
	return callHelper("EvaluateTruthiness", expression)
}

// getZeroValueExpr returns the Go AST node for the zero value of a type.
//
// Takes typeExpr (goast.Expr) which specifies the type to get a zero value for.
//
// Returns goast.Expr which is the AST node for the zero value. For pointer,
// map, function, channel, and interface types, this returns nil. For basic
// types like string, bool, and numeric types, this returns the correct zero
// value. For other types, this returns an empty composite literal.
func getZeroValueExpr(typeExpr goast.Expr) goast.Expr {
	if typeExpr == nil {
		return cachedIdent(nilLiteral)
	}
	switch t := typeExpr.(type) {
	case *goast.StarExpr, *goast.MapType, *goast.FuncType, *goast.ChanType, *goast.InterfaceType:
		return cachedIdent(nilLiteral)
	case *goast.ArrayType:
		if t.Len == nil {
			return cachedIdent(nilLiteral)
		}

	case *goast.Ident:
		switch t.Name {
		case "string":
			return strLit("")
		case "bool":
			return cachedIdent("false")
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"float32", "float64", "rune", "byte", "complex64", "complex128":
			return intLit(0)
		case "any", "error":
			return cachedIdent(nilLiteral)
		}
	}

	return &goast.CompositeLit{Type: typeExpr}
}

// createHTMLAttributeLiteral creates an HTMLAttribute struct literal for use
// in Go AST code generation.
//
// Takes name (string) which is the attribute name.
// Takes value (string) which is the attribute value.
//
// Returns *goast.CompositeLit which is the HTMLAttribute literal ready for
// code generation.
func createHTMLAttributeLiteral(name, value string) *goast.CompositeLit {
	return &goast.CompositeLit{
		Type: cachedIdent("pikoruntime.HTMLAttribute"),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(name)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(value)},
		},
	}
}
