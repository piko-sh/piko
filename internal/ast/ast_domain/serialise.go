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

package ast_domain

// Serialises TemplateAST structures into Go source code as struct literals for caching and code generation.
// Converts parsed templates into compilable Go code with proper formatting, imports, and package declarations.

import (
	"fmt"
	goast "go/ast"
	"go/format"
	"go/token"
)

// SerialiseASTToGoFileContent creates the full content of a Go file that
// stores the given TemplateAST as a variable. It formats the AST as a readable,
// multi-line literal.
//
// Takes tree (*TemplateAST) which is the AST to turn into Go code.
// Takes packageName (string) which sets the package name for the output file.
//
// Returns string which is the formatted Go file content.
func SerialiseASTToGoFileContent(tree *TemplateAST, packageName string) string {
	if tree == nil {
		return fmt.Sprintf("package %s\n\n// AST was nil\n", packageName)
	}

	rawASTLiteral := generateCompactLiteral(tree)
	tidyASTLiteral := formatCompactLiteral(rawASTLiteral)
	goFileContent := embedLiteralInGoFile(tidyASTLiteral, packageName)

	formattedGoCode, err := format.Source([]byte(goFileContent))
	if err != nil {
		return goFileContent
	}

	return string(formattedGoCode)
}

// SerialiseASTString converts a TemplateAST into a formatted Go source string.
//
// Takes tree (*TemplateAST) which is the template AST to convert.
//
// Returns string which is the formatted Go source code.
func SerialiseASTString(tree *TemplateAST) string {
	if tree == nil {
		return "/* AST is nil */"
	}
	return formatASTNode(SerialiseAST(tree))
}

// SerialiseNodeString converts a TemplateNode into formatted Go source code.
//
// Takes node (*TemplateNode) which is the template node to convert.
//
// Returns string which is the formatted Go source code. When node is nil,
// returns a comment that states the node is nil.
func SerialiseNodeString(node *TemplateNode) string {
	if node == nil {
		return "/* Node is nil */"
	}
	return formatASTNode(SerialiseNode(node))
}

// SerialiseAST converts a TemplateAST into a Go AST expression.
//
// When tree is nil, returns a nil identifier.
//
// Takes tree (*TemplateAST) which is the template AST to convert.
//
// Returns goast.Expr which is the Go AST expression for the template.
func SerialiseAST(tree *TemplateAST) goast.Expr {
	if tree == nil {
		return goast.NewIdent(identNil)
	}
	return buildIIFE("TemplateAST", buildTemplateASTLiteral(tree))
}

// SerialiseNode converts a TemplateNode into a Go AST expression.
//
// When node is nil, returns an identifier for nil.
//
// Takes node (*TemplateNode) which is the template node to convert.
//
// Returns goast.Expr which is the generated AST expression.
func SerialiseNode(node *TemplateNode) goast.Expr {
	if node == nil {
		return goast.NewIdent(identNil)
	}
	return buildIIFE(typeTemplateNode, buildNodeLiteral(node))
}

// generateCompactLiteral converts a template AST into its compact string form.
//
// Takes tree (*TemplateAST) which is the parsed template to convert.
//
// Returns string which is the compact string form of the AST.
func generateCompactLiteral(tree *TemplateAST) string {
	return SerialiseASTString(tree)
}

// formatCompactLiteral formats a compact literal by tidying its Go syntax.
//
// Takes compact (string) which is the compact literal to format.
//
// Returns string which is the tidied Go literal.
func formatCompactLiteral(compact string) string {
	return tidyGoLiteral(compact)
}

// embedLiteralInGoFile creates a Go source file containing an embedded AST
// literal.
//
// Takes literal (string) which is the AST literal to embed in the file.
// Takes packageName (string) which is the package name for the generated file.
//
// Returns string which is the complete Go source file content.
func embedLiteralInGoFile(literal, packageName string) string {
	const fileTemplate = `package %s

import (
	goast "go/ast"
	"go/parser"

	"piko.sh/piko/internal/ast/ast_domain"
)

var GeneratedAST = %s
`
	return fmt.Sprintf(fileTemplate, packageName, literal)
}

// astType returns a Go AST selector expression for the given type name.
//
// Takes name (string) which is the type to select from ast_domain.
//
// Returns goast.Expr which is ast_domain.name as a selector expression.
func astType(name string) goast.Expr {
	return &goast.SelectorExpr{X: goast.NewIdent("ast_domain"), Sel: goast.NewIdent(name)}
}

// newCompositeLit creates a new composite literal AST node with the given type
// and elements.
//
// Takes typ (goast.Expr) which specifies the type of the composite literal.
// Takes elts ([]goast.Expr) which provides the elements of the literal.
//
// Returns *goast.CompositeLit which is the constructed AST node.
func newCompositeLit(typ goast.Expr, elts []goast.Expr) *goast.CompositeLit {
	return &goast.CompositeLit{
		Type:       typ,
		Lbrace:     0,
		Elts:       elts,
		Rbrace:     0,
		Incomplete: false,
	}
}

// newArrayType creates a slice type AST node with the given element type.
//
// Takes elt (goast.Expr) which specifies the type for elements in the slice.
//
// Returns *goast.ArrayType which is the slice type node for use in the AST.
func newArrayType(elt goast.Expr) *goast.ArrayType {
	return &goast.ArrayType{
		Lbrack: 0,
		Len:    nil,
		Elt:    elt,
	}
}

// newStarExpr creates a pointer type expression wrapping the given expression.
//
// Takes x (goast.Expr) which is the expression to wrap as a pointer type.
//
// Returns *goast.StarExpr which is the pointer type expression.
func newStarExpr(x goast.Expr) *goast.StarExpr {
	return &goast.StarExpr{
		Star: 0,
		X:    x,
	}
}

// newUnaryExpr creates a Go unary expression with the given operator and
// operand.
//
// Takes operator (token.Token) which specifies the unary operator.
// Takes x (goast.Expr) which is the operand expression.
//
// Returns *goast.UnaryExpr which is the constructed unary expression.
func newUnaryExpr(operator token.Token, x goast.Expr) *goast.UnaryExpr {
	return &goast.UnaryExpr{
		OpPos: 0,
		Op:    operator,
		X:     x,
	}
}

// newMapType creates a map type AST node with the given key and value types.
//
// Takes key (goast.Expr) which specifies the type for map keys.
// Takes value (goast.Expr) which specifies the type for map values.
//
// Returns *goast.MapType which is the constructed map type node.
func newMapType(key, value goast.Expr) *goast.MapType {
	return &goast.MapType{
		Map:   0,
		Key:   key,
		Value: value,
	}
}

// newBasicLit creates a new basic literal AST node with the given token kind
// and value.
//
// Takes kind (token.Token) which specifies the literal type.
// Takes value (string) which is the literal value.
//
// Returns *goast.BasicLit which is the constructed AST node.
func newBasicLit(kind token.Token, value string) *goast.BasicLit {
	return &goast.BasicLit{
		ValuePos: 0,
		Kind:     kind,
		Value:    value,
	}
}

// newKeyValueExpr creates a key-value expression for use in composite literals.
//
// Takes key (goast.Expr) which is the key part of the pair.
// Takes value (goast.Expr) which is the value part of the pair.
//
// Returns *goast.KeyValueExpr which represents a key: value pair.
func newKeyValueExpr(key, value goast.Expr) *goast.KeyValueExpr {
	return &goast.KeyValueExpr{
		Key:   key,
		Colon: 0,
		Value: value,
	}
}

// newCallExpr creates a call expression AST node from a function and arguments.
//
// Takes fun (goast.Expr) which is the function or method to call.
// Takes arguments ([]goast.Expr) which are the arguments to pass to the function.
//
// Returns *goast.CallExpr which is the call expression node.
func newCallExpr(fun goast.Expr, arguments []goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun:      fun,
		Lparen:   0,
		Args:     arguments,
		Ellipsis: 0,
		Rparen:   0,
	}
}

// buildIIFE builds an immediately invoked function expression (IIFE) AST node.
//
// Takes returnTypeName (string) which sets the return type of the function.
// Takes bodyExpr (goast.Expr) which provides the expression for the function
// body.
//
// Returns *goast.CallExpr which is the complete IIFE call expression.
func buildIIFE(returnTypeName string, bodyExpr goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: buildIIFEFuncType(returnTypeName),
			Body: buildIIFEBody(bodyExpr),
		},
		Lparen:   0,
		Args:     nil,
		Ellipsis: 0,
		Rparen:   0,
	}
}

// buildIIFEFuncType builds a function type AST node for an IIFE that returns
// a pointer to the given type.
//
// Takes returnTypeName (string) which is the name of the type to return a
// pointer to.
//
// Returns *goast.FuncType which represents a function with no parameters and
// a single pointer return type.
func buildIIFEFuncType(returnTypeName string) *goast.FuncType {
	return &goast.FuncType{
		Func:       0,
		TypeParams: nil,
		Params:     nil,
		Results: &goast.FieldList{
			Opening: 0,
			List: []*goast.Field{
				{Names: nil, Type: &goast.StarExpr{Star: 0, X: astType(returnTypeName)}, Tag: nil, Comment: nil},
			},
			Closing: 0,
		},
	}
}

// buildIIFEBody builds the body of an immediately invoked function expression.
// It creates helper variable assignments and a return statement.
//
// Takes bodyExpr (goast.Expr) which is the expression to return from the IIFE.
//
// Returns *goast.BlockStmt which contains the helper assignments and return.
func buildIIFEBody(bodyExpr goast.Expr) *goast.BlockStmt {
	return &goast.BlockStmt{
		Lbrace: 0,
		List: []goast.Stmt{
			buildTypeExprFromStringHelperAssignment(),
			buildBlankAssignment(identTypeExprFromString),
			&goast.ReturnStmt{Return: 0, Results: []goast.Expr{bodyExpr}},
		},
		Rbrace: 0,
	}
}

// buildTypeExprFromStringHelperAssignment creates an assignment statement that
// defines the type expression helper function.
//
// Returns *goast.AssignStmt which assigns the helper function literal to its
// identifier.
func buildTypeExprFromStringHelperAssignment() *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs:    []goast.Expr{goast.NewIdent(identTypeExprFromString)},
		TokPos: 0,
		Tok:    token.DEFINE,
		Rhs:    []goast.Expr{buildTypeExprFromStringFuncLit()},
	}
}

// buildTypeExprFromStringFuncLit builds an AST function literal that parses a
// string into a Go expression.
//
// Returns *goast.FuncLit which takes a string and returns a goast.Expr, or nil
// if parsing fails.
func buildTypeExprFromStringFuncLit() *goast.FuncLit {
	return &goast.FuncLit{
		Type: &goast.FuncType{
			Func:       0,
			TypeParams: nil,
			Params:     newFieldList(goast.NewIdent(identS), goast.NewIdent(identString)),
			Results:    newFieldList(nil, &goast.SelectorExpr{X: goast.NewIdent("goast"), Sel: goast.NewIdent("Expr")}),
		},
		Body: &goast.BlockStmt{
			Lbrace: 0,
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs:    []goast.Expr{goast.NewIdent("expr"), goast.NewIdent("err")},
					TokPos: 0, Tok: token.DEFINE,
					Rhs: []goast.Expr{newCallExpr(&goast.SelectorExpr{X: goast.NewIdent("parser"), Sel: goast.NewIdent("ParseExpr")}, []goast.Expr{goast.NewIdent(identS)})},
				},
				&goast.IfStmt{
					If: 0, Init: nil,
					Cond: &goast.BinaryExpr{X: goast.NewIdent("err"), OpPos: 0, Op: token.NEQ, Y: goast.NewIdent(identNil)},
					Body: &goast.BlockStmt{Lbrace: 0, List: []goast.Stmt{&goast.ReturnStmt{Return: 0, Results: []goast.Expr{goast.NewIdent(identNil)}}}, Rbrace: 0},
					Else: nil,
				},
				&goast.ReturnStmt{Return: 0, Results: []goast.Expr{goast.NewIdent("expr")}},
			},
			Rbrace: 0,
		},
	}
}

// buildBlankAssignment creates an assignment statement that discards a value.
//
// Takes varName (string) which is the variable to assign to the blank
// identifier.
//
// Returns *goast.AssignStmt which represents "_ = varName".
func buildBlankAssignment(varName string) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs:    []goast.Expr{goast.NewIdent("_")},
		TokPos: 0,
		Tok:    token.ASSIGN,
		Rhs:    []goast.Expr{goast.NewIdent(varName)},
	}
}

// newFieldList creates a FieldList that holds a single field.
//
// Takes name (*goast.Ident) which is the field name, or nil for unnamed
// fields.
// Takes typ (goast.Expr) which is the type expression for the field.
//
// Returns *goast.FieldList which holds the new field.
func newFieldList(name *goast.Ident, typ goast.Expr) *goast.FieldList {
	var names []*goast.Ident
	if name != nil {
		names = []*goast.Ident{name}
	}
	return &goast.FieldList{
		Opening: 0,
		List:    []*goast.Field{{Names: names, Type: typ, Tag: nil, Comment: nil}},
		Closing: 0,
	}
}
