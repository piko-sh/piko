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
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

const (
	// dotSeparator is the dot character used to join package and symbol names.
	dotSeparator = "."

	// printerTabWidth is the tab width used when formatting AST expressions.
	printerTabWidth = 8
)

var (
	// sharedPrintFileSet is a package-level FileSet used by printing utilities.
	// Since we don't use position information from the printed output, we can
	// safely reuse a single FileSet instance to avoid allocation overhead (~15-20MB
	// savings).
	sharedPrintFileSet = token.NewFileSet()

	// primitiveASTCache contains pre-parsed AST expressions for all Go primitive
	// types and pre-declared identifiers. This avoids repeated parser.ParseExpr
	// calls for these extremely common type strings, significantly reducing CPU
	// overhead.
	primitiveASTCache = func() map[string]ast.Expr {
		primitives := []string{
			"bool",
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"byte", "rune",
			"float32", "float64", "complex64", "complex128",
			"string",
			"any", "error", "nil", "true", "false", "comparable",
		}
		cache := make(map[string]ast.Expr, len(primitives))
		for _, p := range primitives {
			cache[p] = ast.NewIdent(p)
		}
		return cache
	}()

	// primitiveAndBuiltinSet is a set of all primitive types, pre-declared
	// identifiers, and built-in type keywords that should not be qualified with a
	// package name.
	primitiveAndBuiltinSet = map[string]bool{
		"any": true, "error": true, "comparable": true, "function": true, "struct": true, "builtin_function": true,
		"bool": true,
		"int":  true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
		"byte": true, "rune": true,
		"float32": true, "float64": true, "complex64": true, "complex128": true,
		"string": true,
		"nil":    true, "true": true, "false": true,
		"interface{}": true,
		"map":         true, "slice": true, "chan": true, "func": true,
	}
)

// TypeStringToAST parses a Go type string into its matching AST expression.
//
// For primitive types, it returns a new identifier to avoid callers changing
// shared cached nodes. When parsing fails, it returns a placeholder "any"
// identifier.
//
// Takes typeString (string) which specifies the Go type to parse.
//
// Returns ast.Expr which is the parsed AST expression, or nil if typeString is
// empty.
func TypeStringToAST(typeString string) ast.Expr {
	if typeString == "" {
		return nil
	}

	if _, isPrimitive := primitiveASTCache[typeString]; isPrimitive {
		return ast.NewIdent(typeString)
	}

	expression, err := parser.ParseExpr(typeString)
	if err != nil {
		return ast.NewIdent("any /* failed to parse type string: " + strings.ReplaceAll(typeString, " ", "_") + " */")
	}
	return expression
}

// ASTToTypeString converts an AST expression back into its Go type string
// representation.
//
// Takes expression (ast.Expr) which is the AST expression to
// convert.
// Takes pkgAlias (...string) which optionally qualifies unqualified
// identifiers in the AST with the given package alias.
//
// Returns string which is the Go type representation of the
// expression.
func ASTToTypeString(expression ast.Expr, pkgAlias ...string) string {
	if expression == nil {
		return ""
	}

	pAlias := ""
	if len(pkgAlias) > 0 {
		pAlias = pkgAlias[0]
	}

	if result, ok := tryFastPathConversion(expression, pAlias); ok {
		return result
	}

	return slowPathConversion(expression, pAlias)
}

// IsPrimitiveOrBuiltin reports whether a type name is a Go primitive, a
// pre-declared identifier, or follows the naming pattern for a generic type
// parameter. These types should not have a package name prefix.
//
// Takes name (string) which is the type name to check.
//
// Returns bool which is true if the name is a primitive, built-in, or generic
// type parameter.
func IsPrimitiveOrBuiltin(name string) bool {
	return primitiveAndBuiltinSet[name]
}

// UnqualifyTypeExpr removes package qualifiers from a type expression.
//
// It takes an expression such as pkg.Type or *pkg.Type and returns the
// unqualified form such as Type or *Type. It recurses through pointers,
// slices, maps, generics, and function types.
//
// Takes expression (ast.Expr) which is the type expression to
// unqualify.
//
// Returns ast.Expr which is the unqualified type expression, or the
// original expression if no qualification was present.
func UnqualifyTypeExpr(expression ast.Expr) ast.Expr {
	if expression == nil {
		return nil
	}

	switch n := expression.(type) {
	case *ast.SelectorExpr:
		return n.Sel
	case *ast.StarExpr:
		return unqualifyStarExpr(n)
	case *ast.ArrayType:
		return unqualifyArrayType(n)
	case *ast.MapType:
		return unqualifyMapType(n)
	case *ast.IndexExpr:
		return unqualifyIndexExpr(n)
	case *ast.IndexListExpr:
		return unqualifyIndexListExpr(n)
	case *ast.FuncType:
		return unqualifyFuncType(n)
	default:
		return expression
	}
}

// tryFastPathConversion tries to convert simple AST expressions without
// full AST work.
//
// Takes expression (ast.Expr) which is the expression to convert.
// Takes pAlias (string) which overrides the package alias if not
// empty.
//
// Returns string which is the converted type string, or empty on
// failure.
// Returns bool which is true if the fast path worked, false
// otherwise.
func tryFastPathConversion(expression ast.Expr, pAlias string) (string, bool) {
	if result, ok := tryIdentFastPath(expression, pAlias); ok {
		return result, true
	}

	if pAlias != "" {
		return "", false
	}

	if result, ok := trySelectorFastPath(expression); ok {
		return result, true
	}

	if result, ok := tryPointerSelectorFastPath(expression); ok {
		return result, true
	}

	if result, ok := trySliceSelectorFastPath(expression); ok {
		return result, true
	}

	return "", false
}

// tryIdentFastPath handles simple identifier expressions.
//
// Takes expression (ast.Expr) which is the expression to check.
// Takes pAlias (string) which is the package alias to add if needed.
//
// Returns string which is the formatted identifier name.
// Returns bool which is true when the expression was a simple
// identifier.
func tryIdentFastPath(expression ast.Expr, pAlias string) (string, bool) {
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return "", false
	}

	if pAlias == "" || IsPrimitiveOrBuiltin(identifier.Name) {
		return identifier.Name, true
	}
	return pAlias + dotSeparator + identifier.Name, true
}

// trySelectorFastPath handles selector expressions that are already in
// qualified form (pkg.Type).
//
// Takes expression (ast.Expr) which is the expression to check for
// selector form.
//
// Returns string which is the qualified name in "pkg.Type" format.
// Returns bool which indicates whether the expression was a valid
// selector.
func trySelectorFastPath(expression ast.Expr) (string, bool) {
	selectorExpression, ok := expression.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	xIdent, ok := selectorExpression.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	return xIdent.Name + dotSeparator + selectorExpression.Sel.Name, true
}

// tryPointerSelectorFastPath handles pointers to selector expressions
// (*pkg.Type).
//
// Takes expression (ast.Expr) which is the expression to check and
// convert.
//
// Returns string which is the formatted type string (e.g.
// "*pkg.Type").
// Returns bool which is true when the fast path conversion
// succeeded.
func tryPointerSelectorFastPath(expression ast.Expr) (string, bool) {
	star, ok := expression.(*ast.StarExpr)
	if !ok {
		return "", false
	}

	selectorExpression, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	xIdent, ok := selectorExpression.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	return "*" + xIdent.Name + dotSeparator + selectorExpression.Sel.Name, true
}

// trySliceSelectorFastPath handles slices of selector expressions ([]pkg.Type).
//
// Takes expression (ast.Expr) which is the expression to check for
// the slice pattern.
//
// Returns string which contains the formatted type string if
// successful.
// Returns bool which indicates whether the fast path was used.
func trySliceSelectorFastPath(expression ast.Expr) (string, bool) {
	arr, ok := expression.(*ast.ArrayType)
	if !ok || arr.Len != nil {
		return "", false
	}

	selectorExpression, ok := arr.Elt.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	xIdent, ok := selectorExpression.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	return "[]" + xIdent.Name + dotSeparator + selectorExpression.Sel.Name, true
}

// slowPathConversion handles complex types that require full AST manipulation.
//
// Takes expression (ast.Expr) which is the expression to convert to
// a string.
// Takes pAlias (string) which is the package alias to qualify
// identifiers.
//
// Returns string which is the printed representation of the
// expression.
func slowPathConversion(expression ast.Expr, pAlias string) string {
	expressionCopy := deepCopyAST(expression)

	if pAlias != "" {
		expressionCopy = qualifyAST(expressionCopy, pAlias)
	}

	var buffer strings.Builder
	config := printer.Config{Mode: 0, Tabwidth: printerTabWidth}
	err := config.Fprint(&buffer, sharedPrintFileSet, expressionCopy)
	if err != nil {
		return "/* error printing ast node */"
	}

	return buffer.String()
}

// qualifyAST recursively traverses an AST expression and prepends a package
// alias to any unqualified identifiers that are not primitives or built-ins.
//
// When node is nil or pkgAlias is empty, returns node unchanged.
//
// Takes node (ast.Expr) which is the expression to qualify.
// Takes pkgAlias (string) which is the package alias to prepend.
//
// Returns ast.Expr which is the qualified expression tree.
func qualifyAST(node ast.Expr, pkgAlias string) ast.Expr {
	if node == nil || pkgAlias == "" {
		return node
	}

	switch n := node.(type) {
	case *ast.Ident:
		return qualifyIdent(n, pkgAlias)
	case *ast.StarExpr:
		return &ast.StarExpr{X: qualifyAST(n.X, pkgAlias)}
	case *ast.ArrayType:
		return &ast.ArrayType{Len: n.Len, Elt: qualifyAST(n.Elt, pkgAlias)}
	case *ast.Ellipsis:
		n.Elt = qualifyAST(n.Elt, pkgAlias)
		return n
	case *ast.MapType:
		return &ast.MapType{Key: qualifyAST(n.Key, pkgAlias), Value: qualifyAST(n.Value, pkgAlias)}
	case *ast.ChanType:
		return &ast.ChanType{Dir: n.Dir, Value: qualifyAST(n.Value, pkgAlias)}
	case *ast.ParenExpr:
		return &ast.ParenExpr{Lparen: n.Lparen, X: qualifyAST(n.X, pkgAlias), Rparen: n.Rparen}
	case *ast.FuncType:
		return qualifyFuncType(n, pkgAlias)
	case *ast.InterfaceType:
		return qualifyInterfaceType(n, pkgAlias)
	case *ast.StructType:
		return qualifyStructType(n, pkgAlias)
	case *ast.SelectorExpr:
		return n
	case *ast.IndexExpr:
		return qualifyIndexExpr(n, pkgAlias)
	case *ast.IndexListExpr:
		return qualifyIndexListExpr(n, pkgAlias)
	case *ast.TypeAssertExpr:
		return qualifyTypeAssertExpr(n, pkgAlias)
	default:
		return n
	}
}

// qualifyIdent adds a package alias to an identifier when needed.
//
// Takes n (*ast.Ident) which is the identifier to qualify.
// Takes pkgAlias (string) which is the package alias to add as a prefix.
//
// Returns ast.Expr which is a selector expression with the package alias, or
// the original identifier if it is a primitive or built-in type.
func qualifyIdent(n *ast.Ident, pkgAlias string) ast.Expr {
	if !IsPrimitiveOrBuiltin(n.Name) {
		return &ast.SelectorExpr{X: ast.NewIdent(pkgAlias), Sel: n}
	}
	return n
}

// qualifyFuncType adds package qualifiers to all types in a function type.
//
// Takes n (*ast.FuncType) which is the function type to qualify.
// Takes pkgAlias (string) which is the package alias to add before types.
//
// Returns *ast.FuncType which is the same node with all types qualified.
func qualifyFuncType(n *ast.FuncType, pkgAlias string) *ast.FuncType {
	if n.Params != nil {
		for _, f := range n.Params.List {
			f.Type = qualifyAST(f.Type, pkgAlias)
		}
	}
	if n.Results != nil {
		for _, f := range n.Results.List {
			f.Type = qualifyAST(f.Type, pkgAlias)
		}
	}
	return n
}

// qualifyInterfaceType adds a package alias to all types in an interface type.
//
// Takes n (*ast.InterfaceType) which is the interface type to update.
// Takes pkgAlias (string) which is the package alias to add to types.
//
// Returns *ast.InterfaceType which is the same interface with updated types.
func qualifyInterfaceType(n *ast.InterfaceType, pkgAlias string) *ast.InterfaceType {
	if n.Methods != nil {
		for _, f := range n.Methods.List {
			f.Type = qualifyAST(f.Type, pkgAlias)
		}
	}
	return n
}

// qualifyStructType adds the package alias to all field types in a struct.
//
// Takes n (*ast.StructType) which is the struct type to process.
// Takes pkgAlias (string) which is the alias to add before each type name.
//
// Returns *ast.StructType which is the same struct with its field types
// updated.
func qualifyStructType(n *ast.StructType, pkgAlias string) *ast.StructType {
	if n.Fields != nil {
		for _, f := range n.Fields.List {
			f.Type = qualifyAST(f.Type, pkgAlias)
		}
	}
	return n
}

// qualifyIndexExpr adds a package alias to a generic type with one type
// parameter.
//
// Takes n (*ast.IndexExpr) which is the index expression to qualify.
// Takes pkgAlias (string) which is the package alias to add.
//
// Returns *ast.IndexExpr which is the qualified index expression.
func qualifyIndexExpr(n *ast.IndexExpr, pkgAlias string) *ast.IndexExpr {
	n.X = qualifyAST(n.X, pkgAlias)
	n.Index = qualifyAST(n.Index, pkgAlias)
	return n
}

// qualifyIndexListExpr adds a package alias to a generic type that has more
// than one type parameter.
//
// Takes n (*ast.IndexListExpr) which is the generic type expression to update.
// Takes pkgAlias (string) which is the package alias to add before identifiers.
//
// Returns *ast.IndexListExpr which is the updated expression with the package
// alias added to identifiers.
func qualifyIndexListExpr(n *ast.IndexListExpr, pkgAlias string) *ast.IndexListExpr {
	n.X = qualifyAST(n.X, pkgAlias)
	for i, index := range n.Indices {
		n.Indices[i] = qualifyAST(index, pkgAlias)
	}
	return n
}

// qualifyTypeAssertExpr adds a package alias to the type in a type assertion.
//
// Takes n (*ast.TypeAssertExpr) which is the type assertion to update.
// Takes pkgAlias (string) which is the package alias to add to type names.
//
// Returns *ast.TypeAssertExpr which is the updated expression.
func qualifyTypeAssertExpr(n *ast.TypeAssertExpr, pkgAlias string) *ast.TypeAssertExpr {
	if n.Type != nil {
		n.Type = qualifyAST(n.Type, pkgAlias)
	}
	return n
}

// deepCopyAST creates a deep copy of an AST expression by printing the node to
// a string and parsing it again. This prevents side effects, as AST nodes are
// mutable pointers.
//
// Takes node (ast.Expr) which is the expression to copy.
//
// Returns ast.Expr which is a new, independent copy of the input expression.
func deepCopyAST(node ast.Expr) ast.Expr {
	var buffer bytes.Buffer
	err := printer.Fprint(&buffer, sharedPrintFileSet, node)
	if err != nil {
		return ast.NewIdent("any /* internal copy error */")
	}

	newNode, err := parser.ParseExpr(buffer.String())
	if err != nil {
		return ast.NewIdent("any")
	}
	return newNode
}

// unqualifyStarExpr removes package qualifiers from pointer types.
// For example, *pkg.Type becomes *Type.
//
// Takes n (*ast.StarExpr) which is the pointer type expression to process.
//
// Returns ast.Expr which is the pointer expression without package qualifiers,
// or the original if no change was needed.
func unqualifyStarExpr(n *ast.StarExpr) ast.Expr {
	unqualifiedX := UnqualifyTypeExpr(n.X)
	if unqualifiedX != n.X {
		return &ast.StarExpr{X: unqualifiedX}
	}
	return n
}

// unqualifyArrayType removes package qualifiers from the element type of an
// array or slice, converting []pkg.Type to []Type.
//
// Takes n (*ast.ArrayType) which is the array or slice type to process.
//
// Returns ast.Expr which is the type with qualifiers removed, or the original
// if no changes were needed.
func unqualifyArrayType(n *ast.ArrayType) ast.Expr {
	unqualifiedElt := UnqualifyTypeExpr(n.Elt)
	if unqualifiedElt != n.Elt {
		return &ast.ArrayType{Len: n.Len, Elt: unqualifiedElt}
	}
	return n
}

// unqualifyMapType removes package qualifiers from a map type's key and value.
//
// Takes n (*ast.MapType) which is the map type to process.
//
// Returns ast.Expr which is the map type with unqualified key and value types,
// or the original if no changes were needed.
func unqualifyMapType(n *ast.MapType) ast.Expr {
	newKey := UnqualifyTypeExpr(n.Key)
	newValue := UnqualifyTypeExpr(n.Value)
	if newKey != n.Key || newValue != n.Value {
		return &ast.MapType{Key: newKey, Value: newValue}
	}
	return n
}

// unqualifyIndexExpr removes package qualifiers from a generic type expression.
// For example, Box[pkg.User] becomes Box[User] by removing the package prefix
// from both the base type and the type argument.
//
// Takes n (*ast.IndexExpr) which is the generic type expression to process.
//
// Returns ast.Expr which is the expression without package qualifiers, or n
// unchanged if no qualifiers were present.
func unqualifyIndexExpr(n *ast.IndexExpr) ast.Expr {
	newX := UnqualifyTypeExpr(n.X)
	newIndex := UnqualifyTypeExpr(n.Index)
	if newX != n.X || newIndex != n.Index {
		return &ast.IndexExpr{X: newX, Index: newIndex}
	}
	return n
}

// unqualifyIndexListExpr removes package names from generic type expressions
// such as Map[pkg.Key, pkg.Value].
//
// Takes n (*ast.IndexListExpr) which is the generic type expression to process.
//
// Returns ast.Expr which is the expression with package names removed, or the
// original if no changes were needed.
func unqualifyIndexListExpr(n *ast.IndexListExpr) ast.Expr {
	newX := UnqualifyTypeExpr(n.X)
	changed := newX != n.X
	newIndices := make([]ast.Expr, len(n.Indices))
	for i, index := range n.Indices {
		newIndices[i] = UnqualifyTypeExpr(index)
		if newIndices[i] != index {
			changed = true
		}
	}
	if changed {
		return &ast.IndexListExpr{X: newX, Indices: newIndices}
	}
	return n
}

// unqualifyFuncType removes package qualifiers from all types in a function
// type signature.
//
// Takes n (*ast.FuncType) which is the function type to process.
//
// Returns *ast.FuncType which is the same node with all parameter and result
// types unqualified.
func unqualifyFuncType(n *ast.FuncType) *ast.FuncType {
	if n.Params != nil {
		for _, f := range n.Params.List {
			unqualifiedType := UnqualifyTypeExpr(f.Type)
			if unqualifiedType != f.Type {
				f.Type = unqualifiedType
			}
		}
	}
	if n.Results != nil {
		for _, f := range n.Results.List {
			unqualifiedType := UnqualifyTypeExpr(f.Type)
			if unqualifiedType != f.Type {
				f.Type = unqualifiedType
			}
		}
	}
	return n
}
