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

package generator_domain

import (
	"strings"

	parsejs "github.com/tdewolff/parse/v2/js"
)

// jsASTBuilder provides methods for building JavaScript AST nodes using the
// tdewolff parsejs library.
type jsASTBuilder struct{}

// newVar creates a variable reference.
//
// Takes name (string) which specifies the variable name.
//
// Returns *parsejs.Var which is the variable reference node.
func (*jsASTBuilder) newVar(name string) *parsejs.Var {
	return &parsejs.Var{Data: []byte(name)}
}

// newIdentifier creates an identifier expression.
//
// Takes name (string) which specifies the identifier text.
//
// Returns *parsejs.LiteralExpr which is the identifier expression node.
func (*jsASTBuilder) newIdentifier(name string) *parsejs.LiteralExpr {
	return &parsejs.LiteralExpr{TokenType: parsejs.IdentifierToken, Data: []byte(name)}
}

// newStringLiteral creates a string literal with proper quoting.
//
// Takes value (string) which is the unquoted string content.
//
// Returns *parsejs.LiteralExpr which is the quoted string literal expression.
func (*jsASTBuilder) newStringLiteral(value string) *parsejs.LiteralExpr {
	return &parsejs.LiteralExpr{TokenType: parsejs.StringToken, Data: []byte(`"` + value + `"`)}
}

// newCall creates a function call expression.
//
// Takes target (parsejs.IExpr) which specifies the function to call.
// Takes arguments (...parsejs.IExpr) which provides the arguments to pass.
//
// Returns *parsejs.CallExpr which is the constructed call expression.
func (*jsASTBuilder) newCall(target parsejs.IExpr, arguments ...parsejs.IExpr) *parsejs.CallExpr {
	argList := make([]parsejs.Arg, len(arguments))
	for i, argument := range arguments {
		argList[i] = parsejs.Arg{Value: argument}
	}
	return &parsejs.CallExpr{
		X:    target,
		Args: parsejs.Args{List: argList},
	}
}

// newCallWithSpread creates a function call with a spread argument at the end.
//
// Takes target (parsejs.IExpr) which is the function or method to call.
// Takes arguments ([]parsejs.IExpr) which are the regular arguments before the
// spread.
// Takes spreadArg (parsejs.IExpr) which is spread as the final argument.
//
// Returns *parsejs.CallExpr which is the complete call expression.
func (*jsASTBuilder) newCallWithSpread(target parsejs.IExpr, arguments []parsejs.IExpr, spreadArg parsejs.IExpr) *parsejs.CallExpr {
	argList := make([]parsejs.Arg, len(arguments)+1)
	for i, argument := range arguments {
		argList[i] = parsejs.Arg{Value: argument}
	}
	argList[len(arguments)] = parsejs.Arg{Value: spreadArg, Rest: true}
	return &parsejs.CallExpr{
		X:    target,
		Args: parsejs.Args{List: argList},
	}
}

// newDot creates a member access expression (a.b).
//
// Takes target (parsejs.IExpr) which is the expression to access a member on.
// Takes member (string) which is the name of the member to access.
//
// Returns *parsejs.DotExpr which represents the dot access expression.
func (b *jsASTBuilder) newDot(target parsejs.IExpr, member string) *parsejs.DotExpr {
	return &parsejs.DotExpr{
		X: target,
		Y: b.newIdentifier(member),
	}
}

// newMethodCall creates a method call expression (target.method(arguments...)).
//
// Takes target (parsejs.IExpr) which is the object to call the method on.
// Takes method (string) which is the name of the method to invoke.
// Takes arguments (...parsejs.IExpr) which are the arguments to
// pass to the method.
//
// Returns *parsejs.CallExpr which represents the constructed method call.
func (b *jsASTBuilder) newMethodCall(target parsejs.IExpr, method string, arguments ...parsejs.IExpr) *parsejs.CallExpr {
	return b.newCall(b.newDot(target, method), arguments...)
}

// newMethodCallWithSpread creates a method call expression with spread syntax
// (target.method(arguments..., ...spread)).
//
// Takes target (parsejs.IExpr) which is the object to call the method on.
// Takes method (string) which is the name of the method to call.
// Takes arguments ([]parsejs.IExpr) which are the standard arguments to pass.
// Takes spreadArg (parsejs.IExpr) which is the argument to spread.
//
// Returns *parsejs.CallExpr which is the built method call expression.
func (b *jsASTBuilder) newMethodCallWithSpread(target parsejs.IExpr, method string, arguments []parsejs.IExpr, spreadArg parsejs.IExpr) *parsejs.CallExpr {
	return b.newCallWithSpread(b.newDot(target, method), arguments, spreadArg)
}

// newNew creates a new expression AST node (new X(arguments)).
//
// Takes target (parsejs.IExpr) which is the constructor to call.
// Takes arguments (...parsejs.IExpr) which are the arguments to pass.
//
// Returns *parsejs.NewExpr which represents the new expression.
func (*jsASTBuilder) newNew(target parsejs.IExpr, arguments ...parsejs.IExpr) *parsejs.NewExpr {
	if len(arguments) == 0 {
		return &parsejs.NewExpr{X: target}
	}
	argList := make([]parsejs.Arg, len(arguments))
	for i, argument := range arguments {
		argList[i] = parsejs.Arg{Value: argument}
	}
	return &parsejs.NewExpr{
		X:    target,
		Args: &parsejs.Args{List: argList},
	}
}

// newBinary creates a binary expression (a op b).
//
// Takes op (parsejs.TokenType) which specifies the operator.
// Takes left (parsejs.IExpr) which is the left operand.
// Takes right (parsejs.IExpr) which is the right operand.
//
// Returns *parsejs.BinaryExpr which contains the constructed expression.
func (*jsASTBuilder) newBinary(op parsejs.TokenType, left, right parsejs.IExpr) *parsejs.BinaryExpr {
	return &parsejs.BinaryExpr{Op: op, X: left, Y: right}
}

// newGroup creates a grouping expression ((expression)).
//
// Takes expression (parsejs.IExpr) which is the expression to wrap in parentheses.
//
// Returns *parsejs.GroupExpr which contains the wrapped expression.
func (*jsASTBuilder) newGroup(expression parsejs.IExpr) *parsejs.GroupExpr {
	return &parsejs.GroupExpr{X: expression}
}

// newUnary creates a unary expression (op x).
//
// Takes op (parsejs.TokenType) which specifies the unary operator.
// Takes x (parsejs.IExpr) which is the operand expression.
//
// Returns *parsejs.UnaryExpr which represents the constructed unary expression.
func (*jsASTBuilder) newUnary(op parsejs.TokenType, x parsejs.IExpr) *parsejs.UnaryExpr {
	return &parsejs.UnaryExpr{Op: op, X: x}
}

// newAwait creates an await expression.
//
// Takes x (parsejs.IExpr) which is the expression to await.
//
// Returns *parsejs.UnaryExpr which wraps the expression with the await operator.
func (*jsASTBuilder) newAwait(x parsejs.IExpr) *parsejs.UnaryExpr {
	return &parsejs.UnaryExpr{Op: parsejs.AwaitToken, X: x}
}

// newObject creates an object literal.
//
// Takes properties (...parsejs.Property) which are the key-value pairs for the
// object.
//
// Returns *parsejs.ObjectExpr which is the constructed object expression.
func (*jsASTBuilder) newObject(properties ...parsejs.Property) *parsejs.ObjectExpr {
	return &parsejs.ObjectExpr{List: properties}
}

// newShorthandProperty creates a shorthand property (name is same as value).
//
// Takes name (string) which specifies the property and variable name.
//
// Returns parsejs.Property which is the shorthand property with matching name
// and value.
func (b *jsASTBuilder) newShorthandProperty(name string) parsejs.Property {
	v := b.newVar(name)
	return parsejs.Property{
		Name:  &parsejs.PropertyName{Literal: *b.newIdentifier(name)},
		Value: v,
	}
}

// newConstDecl creates a const declaration.
//
// Takes name (string) which specifies the constant identifier.
// Takes init (parsejs.IExpr) which provides the initialisation expression.
//
// Returns *parsejs.VarDecl which contains the const declaration node.
func (b *jsASTBuilder) newConstDecl(name string, init parsejs.IExpr) *parsejs.VarDecl {
	return &parsejs.VarDecl{
		TokenType: parsejs.ConstToken,
		List: []parsejs.BindingElement{
			{Binding: b.newVar(name), Default: init},
		},
	}
}

// newFunc creates a function declaration.
//
// Takes name (string) which specifies the function name, or empty for anonymous
// functions.
// Takes async (bool) which indicates whether the function is async.
// Takes params ([]string) which lists the parameter names.
// Takes body ([]parsejs.IStmt) which contains the function body statements.
//
// Returns *parsejs.FuncDecl which is the built function declaration.
func (b *jsASTBuilder) newFunc(name string, async bool, params []string, body []parsejs.IStmt) *parsejs.FuncDecl {
	paramBindings := make([]parsejs.BindingElement, len(params))
	for i, p := range params {
		paramBindings[i] = parsejs.BindingElement{Binding: b.newVar(p)}
	}

	var nameVar *parsejs.Var
	if name != "" {
		nameVar = b.newVar(name)
	}

	return &parsejs.FuncDecl{
		Async:  async,
		Name:   nameVar,
		Params: parsejs.Params{List: paramBindings},
		Body:   parsejs.BlockStmt{List: body},
	}
}

// newFuncWithRest creates a function declaration with a rest parameter.
//
// Takes name (string) which specifies the function name.
// Takes async (bool) which indicates whether the function is async.
// Takes params ([]string) which lists the regular parameter names.
// Takes restParam (string) which specifies the rest parameter name.
// Takes body ([]parsejs.IStmt) which contains the function body statements.
//
// Returns *parsejs.FuncDecl which is the function declaration with a rest
// parameter set.
func (b *jsASTBuilder) newFuncWithRest(name string, async bool, params []string, restParam string, body []parsejs.IStmt) *parsejs.FuncDecl {
	jsFunction := b.newFunc(name, async, params, body)
	jsFunction.Params.Rest = b.newVar(restParam)
	return jsFunction
}

// newReturn creates a return statement.
//
// Takes value (parsejs.IExpr) which specifies the expression to return.
//
// Returns *parsejs.ReturnStmt which is the constructed return statement.
func (*jsASTBuilder) newReturn(value parsejs.IExpr) *parsejs.ReturnStmt {
	return &parsejs.ReturnStmt{Value: value}
}

// newExprStmt creates an expression statement.
//
// Takes expression (parsejs.IExpr) which is the expression to wrap.
//
// Returns *parsejs.ExprStmt which contains the expression as a statement.
func (*jsASTBuilder) newExprStmt(expression parsejs.IExpr) *parsejs.ExprStmt {
	return &parsejs.ExprStmt{Value: expression}
}

// newIf creates an if statement.
//
// Takes condition (parsejs.IExpr) which is the condition to check.
// Takes body (parsejs.IStmt) which runs when the condition is true.
// Takes elseStmt (parsejs.IStmt) which runs when the condition is false.
//
// Returns *parsejs.IfStmt which is the built if statement.
func (*jsASTBuilder) newIf(condition parsejs.IExpr, body parsejs.IStmt, elseStmt parsejs.IStmt) *parsejs.IfStmt {
	return &parsejs.IfStmt{Cond: condition, Body: body, Else: elseStmt}
}

// newImport creates an import statement with named imports.
//
// Takes names ([]string) which lists the identifiers to import.
// Takes modulePath (string) which specifies the module to import from.
//
// Returns *parsejs.ImportStmt which is the built import statement.
func (*jsASTBuilder) newImport(names []string, modulePath string) *parsejs.ImportStmt {
	aliases := make([]parsejs.Alias, len(names))
	for i, name := range names {
		aliases[i] = parsejs.Alias{Binding: []byte(name)}
	}
	return &parsejs.ImportStmt{
		List:   aliases,
		Module: []byte(`"` + modulePath + `"`),
	}
}

// newExportFunc creates an exported function declaration.
//
// Takes name (string) which specifies the function name.
// Takes async (bool) which indicates if the function is asynchronous.
// Takes params ([]string) which lists the function parameter names.
// Takes restParam (string) which specifies the rest parameter name, if any.
// Takes body ([]parsejs.IStmt) which contains the function body statements.
//
// Returns *parsejs.ExportStmt which wraps the function in an export statement.
func (b *jsASTBuilder) newExportFunc(name string, async bool, params []string, restParam string, body []parsejs.IStmt) *parsejs.ExportStmt {
	jsFunction := b.newFuncWithRest(name, async, params, restParam, body)
	return &parsejs.ExportStmt{Decl: jsFunction}
}

// renderStmt converts a single statement to JavaScript source code.
//
// Takes statement (parsejs.IStmt) which is the statement to render.
//
// Returns string which contains the JavaScript source code.
func (*jsASTBuilder) renderStmt(statement parsejs.IStmt) string {
	var builder strings.Builder
	statement.JS(&builder)
	return builder.String()
}

// newJSASTBuilder creates a new JavaScript AST builder.
//
// Returns *jsASTBuilder which is an empty builder ready for use.
func newJSASTBuilder() *jsASTBuilder {
	return &jsASTBuilder{}
}
