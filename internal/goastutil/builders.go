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

package goastutil

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strconv"
	"sync"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

// defaultTabWidth is the number of spaces used for tab indentation in Go code.
const defaultTabWidth = 4

var (
	// staticIdentCache provides a cache of commonly-used *ast.Ident instances.
	// Since Ident structs are immutable in our usage (we only set Name), we can
	// safely share the same Ident instance across multiple locations in the AST.
	staticIdentCache = make(map[string]*ast.Ident, 128)

	// dynamicIdentCache is an atomic cache for identifiers not in the static cache.
	dynamicIdentCache sync.Map

	// strLitCache is a dynamic cache for string literals that appear multiple times.
	strLitCache sync.Map

	// litIntZero is a shared AST literal node for the integer value 0.
	litIntZero = &ast.BasicLit{Kind: token.INT, Value: "0"}

	// litIntOne is a shared AST literal node for the integer value 1.
	litIntOne = &ast.BasicLit{Kind: token.INT, Value: "1"}

	// litEmptyString is a shared AST literal node for an empty string.
	litEmptyString = &ast.BasicLit{Kind: token.STRING, Value: `""`}
)

// ResetDynamicCaches clears the dynamic identifier and string literal caches.
// This is intended for test isolation between iterations.
func ResetDynamicCaches() {
	dynamicIdentCache = sync.Map{}
	strLitCache = sync.Map{}
}

// RegisterIdent adds an identifier to the static cache for reuse.
// Call this at package init time to pre-fill domain-specific identifiers.
//
// Takes name (string) which specifies the identifier name to register.
func RegisterIdent(name string) {
	if _, exists := staticIdentCache[name]; !exists {
		staticIdentCache[name] = &ast.Ident{Name: name}
	}
}

// CachedIdent returns a cached *ast.Ident for the given name. Use this instead
// of ast.NewIdent for identifier names that appear often to reduce memory use.
//
// Takes name (string) which specifies the identifier name to look up or cache.
//
// Returns *ast.Ident which is a cached identifier node for the given name.
func CachedIdent(name string) *ast.Ident {
	if cached, ok := staticIdentCache[name]; ok {
		return cached
	}
	if cached, ok := dynamicIdentCache.Load(name); ok {
		if identifier, isIdent := cached.(*ast.Ident); isIdent {
			return identifier
		}
	}
	newIdent := &ast.Ident{Name: name}
	actual, _ := dynamicIdentCache.LoadOrStore(name, newIdent)
	if result, isIdent := actual.(*ast.Ident); isIdent {
		return result
	}
	return newIdent
}

// StrLit creates a Go AST string literal from a string value.
//
// It uses a two-level cache to reduce memory use during code generation: a
// fixed cache for the empty string and a shared cache for strings that appear
// more than once.
//
// Takes s (string) which is the value to convert to an AST literal.
//
// Returns *ast.BasicLit which is the AST node for the quoted string.
func StrLit(s string) *ast.BasicLit {
	if s == "" {
		return litEmptyString
	}

	quoted := strconv.Quote(s)
	if cached, ok := strLitCache.Load(quoted); ok {
		if lit, isBasicLit := cached.(*ast.BasicLit); isBasicLit {
			return lit
		}
	}

	lit := &ast.BasicLit{Kind: token.STRING, Value: quoted}
	actual, _ := strLitCache.LoadOrStore(quoted, lit)
	if result, isBasicLit := actual.(*ast.BasicLit); isBasicLit {
		return result
	}
	return lit
}

// IntLit creates a Go AST integer literal from an int value. Common values
// (0, 1) are cached to reduce memory use.
//
// Takes i (int) which is the integer value to convert.
//
// Returns *ast.BasicLit which is the AST node for the integer.
func IntLit(i int) *ast.BasicLit {
	switch i {
	case 0:
		return litIntZero
	case 1:
		return litIntOne
	default:
		return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)}
	}
}

// BoolIdent returns the identifier for a boolean value.
//
// Takes b (bool) which is the boolean value to convert.
//
// Returns *ast.Ident which is the cached identifier ("true" or "false").
func BoolIdent(b bool) *ast.Ident {
	if b {
		return CachedIdent("true")
	}
	return CachedIdent("false")
}

// SelectorExpr creates a selector expression for accessing a field or method.
//
// Takes x (string) which is the left-hand identifier.
// Takes selector (string) which is the field or method name to select.
//
// Returns *ast.SelectorExpr which represents the expression x.selector.
func SelectorExpr(x, selector string) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   CachedIdent(x),
		Sel: CachedIdent(selector),
	}
}

// SelectorExprFrom creates a selector expression from an existing expression.
//
// Takes x (ast.Expr) which is the left-hand side expression.
// Takes selector (string) which is the name of the field or method to select.
//
// Returns *ast.SelectorExpr which represents the expression x.selector.
func SelectorExprFrom(x ast.Expr, selector string) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   x,
		Sel: CachedIdent(selector),
	}
}

// CallExpr creates a function call expression.
//
// Takes fun (ast.Expr) which is the function to call.
// Takes arguments (...ast.Expr) which are the arguments to pass.
//
// Returns *ast.CallExpr which represents fun(arguments...).
func CallExpr(fun ast.Expr, arguments ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: fun, Args: arguments}
}

// DefineStmt creates a short variable declaration statement
// (name := rightHandSide).
//
// Takes name (string) which is the variable name.
// Takes rightHandSide (ast.Expr) which is the expression to assign.
//
// Returns *ast.AssignStmt which is the declaration statement.
func DefineStmt(name string, rightHandSide ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{CachedIdent(name)},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{rightHandSide},
	}
}

// DefineStmtMulti creates a short variable declaration with multiple names on
// the left side (a, b := rightHandSide).
//
// Takes names ([]string) which are the variable names to declare.
// Takes rightHandSide (ast.Expr) which is the expression to assign.
//
// Returns *ast.AssignStmt which is the declaration statement.
func DefineStmtMulti(names []string, rightHandSide ast.Expr) *ast.AssignStmt {
	leftHandSide := make([]ast.Expr, len(names))
	for i, name := range names {
		leftHandSide[i] = CachedIdent(name)
	}
	return &ast.AssignStmt{
		Lhs: leftHandSide,
		Tok: token.DEFINE,
		Rhs: []ast.Expr{rightHandSide},
	}
}

// AssignStmt creates an assignment statement (leftHandSide = rightHandSide).
//
// Takes leftHandSide (ast.Expr) which is the left-hand side of the assignment.
// Takes rightHandSide (ast.Expr) which is the right-hand side of the assignment.
//
// Returns *ast.AssignStmt which is the new assignment statement.
func AssignStmt(leftHandSide, rightHandSide ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{leftHandSide},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{rightHandSide},
	}
}

// ReturnStmt creates a return statement.
//
// Takes results (...ast.Expr) which are the values to return.
//
// Returns *ast.ReturnStmt which is the return statement.
func ReturnStmt(results ...ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{Results: results}
}

// VarDecl creates a variable declaration statement (var name type).
//
// Takes name (string) which is the variable name.
// Takes typ (ast.Expr) which is the type expression.
//
// Returns *ast.DeclStmt which contains the variable declaration.
func VarDecl(name string, typ ast.Expr) *ast.DeclStmt {
	return &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{CachedIdent(name)},
					Type:  typ,
				},
			},
		},
	}
}

// CompositeLit creates a composite literal expression ({...}).
//
// Takes typ (ast.Expr) which is the type of the literal, or nil for type
// inference.
// Takes elts (...ast.Expr) which are the elements of the literal.
//
// Returns *ast.CompositeLit which is the new literal expression.
func CompositeLit(typ ast.Expr, elts ...ast.Expr) *ast.CompositeLit {
	return &ast.CompositeLit{Type: typ, Elts: elts}
}

// KeyValueExpr creates a key-value expression (key: value).
//
// Takes key (ast.Expr) which is the key part of the pair.
// Takes value (ast.Expr) which is the value part of the pair.
//
// Returns *ast.KeyValueExpr which is the constructed key-value pair.
func KeyValueExpr(key, value ast.Expr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{Key: key, Value: value}
}

// KeyValueIdent creates a key-value expression with an identifier as the key.
//
// Takes key (string) which is the name for the key identifier.
// Takes value (ast.Expr) which is the expression for the value.
//
// Returns *ast.KeyValueExpr which is the resulting key-value pair.
func KeyValueIdent(key string, value ast.Expr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{Key: CachedIdent(key), Value: value}
}

// StarExpr creates a pointer type expression (*T).
//
// Takes x (ast.Expr) which is the base type.
//
// Returns *ast.StarExpr which is the pointer type expression.
func StarExpr(x ast.Expr) *ast.StarExpr {
	return &ast.StarExpr{X: x}
}

// AddressExpr creates an address-of expression (&x).
//
// Takes x (ast.Expr) which is the expression to take the address of.
//
// Returns *ast.UnaryExpr which is the address expression.
func AddressExpr(x ast.Expr) *ast.UnaryExpr {
	return &ast.UnaryExpr{Op: token.AND, X: x}
}

// IndexExpr creates an index expression (x[index]).
//
// Takes x (ast.Expr) which is the expression being indexed.
// Takes index (ast.Expr) which is the index value.
//
// Returns *ast.IndexExpr which is the resulting index expression.
func IndexExpr(x ast.Expr, index ast.Expr) *ast.IndexExpr {
	return &ast.IndexExpr{X: x, Index: index}
}

// TypeAssertExpr creates a type assertion expression (x.(T)).
//
// Takes x (ast.Expr) which is the expression being asserted.
// Takes typ (ast.Expr) which is the type to assert.
//
// Returns *ast.TypeAssertExpr which is the type assertion expression.
func TypeAssertExpr(x ast.Expr, typ ast.Expr) *ast.TypeAssertExpr {
	return &ast.TypeAssertExpr{X: x, Type: typ}
}

// MapType creates a map type (map[key]value).
//
// Takes key (ast.Expr) which is the key type.
// Takes value (ast.Expr) which is the value type.
//
// Returns *ast.MapType which is the map type expression.
func MapType(key, value ast.Expr) *ast.MapType {
	return &ast.MapType{Key: key, Value: value}
}

// FuncType creates a function type with the given parameters and return types.
//
// Takes params (*ast.FieldList) which specifies the function parameters.
// Takes results (*ast.FieldList) which specifies the return types.
//
// Returns *ast.FuncType which is the function type.
func FuncType(params, results *ast.FieldList) *ast.FuncType {
	return &ast.FuncType{Params: params, Results: results}
}

// FieldList creates a field list from the given fields.
//
// Takes fields (...*ast.Field) which are the AST field nodes to include.
//
// Returns *ast.FieldList which contains the provided fields.
func FieldList(fields ...*ast.Field) *ast.FieldList {
	return &ast.FieldList{List: fields}
}

// Field creates an AST field with an optional name.
//
// When name is empty, creates an anonymous field (embedded type).
//
// Takes name (string) which is the field name, or empty for anonymous.
// Takes typ (ast.Expr) which is the type expression for the field.
//
// Returns *ast.Field which is the field definition.
func Field(name string, typ ast.Expr) *ast.Field {
	var names []*ast.Ident
	if name != "" {
		names = []*ast.Ident{CachedIdent(name)}
	}
	return &ast.Field{Names: names, Type: typ}
}

// FuncLit creates a function literal from a type and body.
//
// Takes typ (*ast.FuncType) which defines the function signature.
// Takes body (*ast.BlockStmt) which contains the function statements.
//
// Returns *ast.FuncLit which is the complete function literal.
func FuncLit(typ *ast.FuncType, body *ast.BlockStmt) *ast.FuncLit {
	return &ast.FuncLit{Type: typ, Body: body}
}

// BlockStmt creates a block statement from the given statements.
//
// Takes statements (...ast.Stmt) which are the statements to include in the block.
//
// Returns *ast.BlockStmt which is the new block statement.
func BlockStmt(statements ...ast.Stmt) *ast.BlockStmt {
	return &ast.BlockStmt{List: statements}
}

// ExprStmt wraps an expression in a statement.
//
// Takes expression (ast.Expr) which is the expression to wrap.
//
// Returns *ast.ExprStmt which is the wrapped expression statement.
func ExprStmt(expression ast.Expr) *ast.ExprStmt {
	return &ast.ExprStmt{X: expression}
}

// IfStmt creates an if statement AST node.
//
// Takes init (ast.Stmt) which is the init statement, or nil if not needed.
// Takes condition (ast.Expr) which is the condition to check.
// Takes body (*ast.BlockStmt) which is the block to run when true.
//
// Returns *ast.IfStmt which is the new if statement node.
func IfStmt(init ast.Stmt, condition ast.Expr, body *ast.BlockStmt) *ast.IfStmt {
	return &ast.IfStmt{Init: init, Cond: condition, Body: body}
}

// FuncDecl creates a function declaration AST node.
//
// Takes name (string) which is the function name.
// Takes params (*ast.FieldList) which holds the parameters, or nil if none.
// Takes results (*ast.FieldList) which holds the return types, or nil if none.
// Takes body (*ast.BlockStmt) which is the function body.
//
// Returns *ast.FuncDecl which is the complete function declaration.
func FuncDecl(name string, params, results *ast.FieldList, body *ast.BlockStmt) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: CachedIdent(name),
		Type: &ast.FuncType{Params: params, Results: results},
		Body: body,
	}
}

// GenDeclType creates a type declaration with the given name and type.
//
// Takes name (string) which is the name for the new type.
// Takes typ (ast.Expr) which is the type expression.
//
// Returns *ast.GenDecl which is the complete type declaration.
func GenDeclType(name string, typ ast.Expr) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: CachedIdent(name),
				Type: typ,
			},
		},
	}
}

// StructType creates a struct type from the given fields.
//
// Takes fields (...*ast.Field) which are the fields to include in the struct.
//
// Returns *ast.StructType which is the new struct type.
func StructType(fields ...*ast.Field) *ast.StructType {
	return &ast.StructType{
		Fields: &ast.FieldList{List: fields},
	}
}

// AddImport adds an import statement to a Go source file.
//
// Takes fset (*token.FileSet) which provides position information for the file.
// Takes file (*ast.File) which is the AST of the file to modify.
// Takes path (string) which is the import path to add.
func AddImport(fset *token.FileSet, file *ast.File, path string) {
	astutil.AddImport(fset, file, path)
}

// AddNamedImport adds a named import to a file.
//
// Takes fset (*token.FileSet) which holds position data for the file.
// Takes file (*ast.File) which is the file to change.
// Takes name (string) which is the alias for the import.
// Takes path (string) which is the import path to add.
func AddNamedImport(fset *token.FileSet, file *ast.File, name, path string) {
	astutil.AddNamedImport(fset, file, name, path)
}

// FormatAST formats an AST file to Go source code with goimports processing.
//
// Takes fset (*token.FileSet) which is the file set.
// Takes file (*ast.File) which is the file to format.
//
// Returns []byte which is the formatted Go source code.
// Returns error when formatting or import processing fails.
func FormatAST(fset *token.FileSet, file *ast.File) ([]byte, error) {
	buffer := new(bytes.Buffer)
	printerConfig := printer.Config{
		Mode:     printer.TabIndent | printer.UseSpaces,
		Tabwidth: defaultTabWidth,
	}
	if err := printerConfig.Fprint(buffer, fset, file); err != nil {
		return nil, fmt.Errorf("printing Go AST: %w", err)
	}
	return imports.Process("", buffer.Bytes(), nil)
}

func init() {
	commonIdents := []string{
		"append", "nil", "ok", "err", "true", "false", "make", "len", "cap",
		"any", "string", "int", "int64", "int32", "int16", "int8",
		"uint", "uint64", "uint32", "uint16", "uint8", "byte",
		"float64", "float32", "bool", "error", "rune",
		"func", "return", "var", "if", "for", "range", "map", "struct",
		"ctx", "args", "result", "data", "raw", "bytes",
	}
	for _, name := range commonIdents {
		staticIdentCache[name] = &ast.Ident{Name: name}
	}
}
