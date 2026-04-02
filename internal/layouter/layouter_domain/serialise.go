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

package layouter_domain

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/format"
	"go/printer"
	"go/token"
)

const (
	// identLayouterDomain is the package qualifier for
	// layouter_domain types in generated code.
	identLayouterDomain = "layouter_domain"

	// identWithStyle is the local variable name for the style
	// override helper function.
	identWithStyle = "withStyle"

	// identOverrides is the parameter name for the style
	// override callback.
	identOverrides = "overrides"

	// identStyle is the local variable name for the computed
	// style being built.
	identStyle = "style"

	// identS is the short parameter name used in generated
	// style override closures.
	identS = "s"

	// identDefaultComputedStyle is the function name for
	// constructing a default computed style.
	identDefaultComputedStyle = "DefaultComputedStyle"

	// typeLayoutBox is the type name for LayoutBox in generated
	// code.
	typeLayoutBox = "LayoutBox"

	// typeComputedStyle is the type name for ComputedStyle in
	// generated code.
	typeComputedStyle = "ComputedStyle"

	// typeBoxEdges is the type name for BoxEdges in generated
	// code.
	typeBoxEdges = "BoxEdges"

	// printTabWidth is the tab width used when printing AST
	// expressions.
	printTabWidth = 4
)

// goldenFileTemplate is the Go source file template used to embed
// a serialised LayoutBox tree as a package-level variable.
const goldenFileTemplate = `package %s

import "piko.sh/piko/internal/layouter/layouter_domain"

var GeneratedLayoutBox = %s
`

// SerialiseLayoutBoxToGoFileContent creates a compilable Go file
// that constructs the given LayoutBox tree as a variable. Style
// properties are expressed as overrides from
// DefaultComputedStyle for readability.
//
// Takes root (*LayoutBox) which is the root of the layout tree
// to serialise.
// Takes packageName (string) which is the Go package name for
// the generated file.
//
// Returns the formatted Go source file content as a string.
func SerialiseLayoutBoxToGoFileContent(root *LayoutBox, packageName string) string {
	if root == nil {
		return fmt.Sprintf("package %s\n\n// LayoutBox was nil\n", packageName)
	}

	boxExpr := buildLayoutBoxExpr(root)
	iifeExpr := buildIIFE(boxExpr)
	compact := printExpr(iifeExpr)
	tidy := tidyGoLiteral(compact)
	source := fmt.Sprintf(goldenFileTemplate, packageName, tidy)

	formatted, err := format.Source([]byte(source))
	if err != nil {
		return source
	}

	return string(formatted)
}

// buildIIFE wraps the given expression in an immediately
// invoked function expression that declares the withStyle
// helper.
//
// Takes bodyExpr (goast.Expr) which is the expression to
// wrap in the IIFE body.
//
// Returns *goast.CallExpr which is the IIFE call node.
func buildIIFE(bodyExpr goast.Expr) *goast.CallExpr {
	return newCallExpr(
		&goast.FuncLit{
			Type: &goast.FuncType{
				Results: &goast.FieldList{
					List: []*goast.Field{
						{Type: newStarExpr(layouterType(typeLayoutBox))},
					},
				},
			},
			Body: &goast.BlockStmt{
				List: []goast.Stmt{
					buildWithStyleAssignment(),
					buildBlankAssignment(identWithStyle),
					&goast.ReturnStmt{Results: []goast.Expr{bodyExpr}},
				},
			},
		},
		nil,
	)
}

// buildWithStyleAssignment builds the short variable
// declaration that assigns the withStyle function literal.
//
// Returns *goast.AssignStmt which is the declaration
// statement.
func buildWithStyleAssignment() *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{goast.NewIdent(identWithStyle)},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{buildWithStyleFuncLit()},
	}
}

// buildWithStyleFuncLit constructs the function literal
// that applies style overrides to a DefaultComputedStyle.
//
// Returns *goast.FuncLit which is the function literal
// node.
func buildWithStyleFuncLit() *goast.FuncLit {
	return &goast.FuncLit{
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{goast.NewIdent(identOverrides)},
						Type: &goast.FuncType{
							Params: &goast.FieldList{
								List: []*goast.Field{
									{Type: newStarExpr(layouterType(typeComputedStyle))},
								},
							},
						},
					},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: layouterType(typeComputedStyle)},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{goast.NewIdent(identStyle)},
					Tok: token.DEFINE,
					Rhs: []goast.Expr{newCallExpr(layouterType(identDefaultComputedStyle), nil)},
				},
				&goast.ExprStmt{
					X: newCallExpr(goast.NewIdent(identOverrides), []goast.Expr{
						newUnaryExpr(token.AND, goast.NewIdent(identStyle)),
					}),
				},
				&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent(identStyle)}},
			},
		},
	}
}

// buildBlankAssignment creates a blank identifier
// assignment to suppress unused variable warnings.
//
// Takes varName (string) which is the variable name to
// assign to the blank identifier.
//
// Returns *goast.AssignStmt which is the blank
// assignment statement.
func buildBlankAssignment(varName string) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{goast.NewIdent("_")},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{goast.NewIdent(varName)},
	}
}

// layouterType builds a qualified selector expression for
// a type in the layouter_domain package.
//
// Takes name (string) which is the type or function name
// to qualify.
//
// Returns goast.Expr which is the selector expression.
func layouterType(name string) goast.Expr {
	return &goast.SelectorExpr{
		X:   goast.NewIdent(identLayouterDomain),
		Sel: goast.NewIdent(name),
	}
}

// newCompositeLit creates a composite literal AST node
// with the given type and elements.
//
// Takes typ (goast.Expr) which is the literal's type
// expression.
// Takes elements ([]goast.Expr) which is the list of
// element expressions.
//
// Returns *goast.CompositeLit which is the composite
// literal node.
func newCompositeLit(typ goast.Expr, elements []goast.Expr) *goast.CompositeLit {
	return &goast.CompositeLit{Type: typ, Elts: elements}
}

// newKeyValueExpr creates a key-value expression AST
// node.
//
// Takes key (goast.Expr) which is the key expression.
// Takes value (goast.Expr) which is the value expression.
//
// Returns *goast.KeyValueExpr which is the key-value
// node.
func newKeyValueExpr(key, value goast.Expr) *goast.KeyValueExpr {
	return &goast.KeyValueExpr{Key: key, Value: value}
}

// newBasicLit creates a basic literal AST node with the
// given token kind and value.
//
// Takes kind (token.Token) which is the literal's token
// type.
// Takes value (string) which is the literal's source
// text.
//
// Returns *goast.BasicLit which is the literal node.
func newBasicLit(kind token.Token, value string) *goast.BasicLit {
	return &goast.BasicLit{Kind: kind, Value: value}
}

// newCallExpr creates a function call expression AST
// node.
//
// Takes fun (goast.Expr) which is the function to call.
// Takes arguments ([]goast.Expr) which is the list of
// call arguments.
//
// Returns *goast.CallExpr which is the call expression
// node.
func newCallExpr(fun goast.Expr, arguments []goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{Fun: fun, Args: arguments}
}

// newStarExpr creates a pointer dereference or pointer
// type AST node.
//
// Takes expression (goast.Expr) which is the operand
// expression.
//
// Returns *goast.StarExpr which is the star expression
// node.
func newStarExpr(expression goast.Expr) *goast.StarExpr {
	return &goast.StarExpr{X: expression}
}

// newUnaryExpr creates a unary expression AST node with
// the given operator and operand.
//
// Takes operator (token.Token) which is the unary
// operator.
// Takes expression (goast.Expr) which is the operand
// expression.
//
// Returns *goast.UnaryExpr which is the unary expression
// node.
func newUnaryExpr(operator token.Token, expression goast.Expr) *goast.UnaryExpr {
	return &goast.UnaryExpr{Op: operator, X: expression}
}

// printExpr formats an AST expression to its Go source
// representation as a string.
//
// Takes node (goast.Expr) which is the expression to
// print.
//
// Returns string which is the formatted Go source text.
func printExpr(node goast.Expr) string {
	var buffer bytes.Buffer
	config := printer.Config{
		Mode:     printer.TabIndent | printer.UseSpaces,
		Tabwidth: printTabWidth,
		Indent:   0,
	}

	fileSet := token.NewFileSet()
	if err := config.Fprint(&buffer, fileSet, node); err != nil {
		return fmt.Sprintf("/* error printing AST: %v */", err)
	}

	return buffer.String()
}
