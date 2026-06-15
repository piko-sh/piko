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
	"sync"
	"sync/atomic"
)

const (
	// dotSeparator is the dot character used to join package and symbol names.
	dotSeparator = "."

	// printerTabWidth is the tab width used when formatting AST expressions.
	printerTabWidth = 8

	// maxParseFileSetBase bounds FileSet reuse so pooled FileSets cannot grow without bound.
	//
	// A recycled FileSet accumulates base offset and file records on every parse, so it is
	// retired once its base passes ~1 GiB. (token.Pos is an int, so this guards retained
	// memory, not 32-bit position overflow.)
	maxParseFileSetBase = 1 << 30

	// typeASTCacheMaxEntries bounds typeASTCache so a long-lived process (watch daemon)
	// cannot accrete type-string templates without limit. On overflow the cache is wiped
	// wholesale; type strings recur, so a wipe only costs a re-parse of those still in use.
	typeASTCacheMaxEntries = 10000
)

var (
	// sharedPrintFileSet is a package-level FileSet used by printing utilities. Since we
	// don't use position information from the printed output, we can safely reuse a single
	// FileSet instance to avoid allocation overhead (~15-20MB savings).
	sharedPrintFileSet = token.NewFileSet()

	// primitiveASTCache contains pre-parsed AST expressions for all Go primitive types and
	// pre-declared identifiers. This avoids repeated parser.ParseExpr calls for these
	// extremely common type strings, significantly reducing CPU overhead.
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

	// primitiveAndBuiltinSet is a set of all primitive types, pre-declared identifiers, and
	// built-in type keywords that should not be qualified with a package name.
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

	// parseFileSetPool recycles FileSets across expression parses. parser.ParseExpr
	// allocates a fresh FileSet on every call purely for position bookkeeping that goastutil
	// never reads (printing uses the separate sharedPrintFileSet), so the FileSet is pure
	// churn; pooling it removes that per-parse allocation.
	parseFileSetPool = sync.Pool{New: func() any { return token.NewFileSet() }}

	// typeASTCache memoises parsed type-string ASTs (string -> ast.Expr template). Only
	// position-independent shapes (see cloneExprStructural) are stored; the template is
	// never returned directly, every caller gets an independent clone, preserving the
	// copy-on-handout invariant the qualify*/unqualify* passes (which mutate in place) rely
	// on.
	typeASTCache sync.Map

	// typeASTCacheEntries counts live typeASTCache entries so it can be wiped wholesale once
	// it passes typeASTCacheMaxEntries.
	typeASTCacheEntries atomic.Int64
)

// parseTypeExpr parses a Go expression against a pooled FileSet.
//
// This reuses a pooled FileSet instead of the fresh one parser.ParseExpr mints per call.
// Safe because goastutil never reads position information from the result (positions are
// int offsets, not pointers into the FileSet), and the pool hands each parse an
// exclusively-owned FileSet.
//
// Takes src (string) which is the Go expression source to parse.
//
// Returns ast.Expr which is the parsed expression.
// Returns error which is non-nil when src fails to parse.
func parseTypeExpr(src string) (ast.Expr, error) {
	fset, ok := parseFileSetPool.Get().(*token.FileSet)
	if !ok || fset == nil {
		fset = token.NewFileSet()
	}
	expr, err := parser.ParseExprFrom(fset, "", src, 0)
	if fset.Base() < maxParseFileSetBase {
		parseFileSetPool.Put(fset)
	}
	return expr, err
}

// TypeStringToAST parses a Go type string into its matching AST expression.
//
// For primitive types, it returns a new identifier to avoid callers changing shared
// cached nodes. When parsing fails, it returns a placeholder "any" identifier.
//
// Takes typeString (string) which specifies the Go type to parse.
//
// Returns ast.Expr which is the parsed AST expression, or nil if typeString is empty.
func TypeStringToAST(typeString string) ast.Expr {
	if typeString == "" {
		return nil
	}

	if _, isPrimitive := primitiveASTCache[typeString]; isPrimitive {
		return ast.NewIdent(typeString)
	}

	if template, ok := typeASTCache.Load(typeString); ok {
		if clone := cloneExprStructural(template.(ast.Expr)); clone != nil {
			return clone
		}
	}

	expression, err := parseTypeExpr(typeString)
	if err != nil {
		return ast.NewIdent("any /* failed to parse type string: " + strings.ReplaceAll(typeString, " ", "_") + " */")
	}
	if clone := cloneExprStructural(expression); clone != nil {
		if _, loaded := typeASTCache.LoadOrStore(typeString, expression); !loaded {
			if typeASTCacheEntries.Add(1) > typeASTCacheMaxEntries {
				typeASTCache.Clear()
				typeASTCacheEntries.Store(0)
			}
		}
		return clone
	}
	return expression
}

// cloneExprStructural deep-clones expression shapes with position-independent output.
//
// It returns nil for any shape whose printed form does depend on token positions (chan,
// fixed-length array, func, struct, interface, and anything containing one). A nil result
// signals the caller to use a fresh parse instead, so output stays byte-identical to
// go/format.Source.
//
// Takes e (ast.Expr) which is the expression to clone structurally.
//
// Returns ast.Expr which is the position-independent clone, or nil when e is not a
// position-independent shape and the caller should re-parse.
func cloneExprStructural(e ast.Expr) ast.Expr {
	switch n := e.(type) {
	case *ast.Ident:
		return ast.NewIdent(n.Name)
	case *ast.SelectorExpr:
		if x := cloneExprStructural(n.X); x != nil {
			return &ast.SelectorExpr{X: x, Sel: ast.NewIdent(n.Sel.Name)}
		}
	case *ast.StarExpr:
		if x := cloneExprStructural(n.X); x != nil {
			return &ast.StarExpr{X: x}
		}
	case *ast.ArrayType:
		return cloneArrayTypeStructural(n)
	case *ast.MapType:
		return cloneMapTypeStructural(n)
	case *ast.IndexExpr:
		return cloneIndexExprStructural(n)
	case *ast.IndexListExpr:
		return cloneIndexListExprStructural(n)
	}
	return nil
}

// cloneArrayTypeStructural clones a slice type ([]T).
//
// Fixed-length arrays ([N]T) depend on the length expression's position, so they are
// refused (nil) to force a fresh parse.
//
// Takes n (*ast.ArrayType) which is the slice or array type to clone.
//
// Returns ast.Expr which is the cloned slice type, or nil when n is not a
// position-independent shape and the caller should re-parse.
func cloneArrayTypeStructural(n *ast.ArrayType) ast.Expr {
	if n.Len != nil {
		return nil
	}
	elt := cloneExprStructural(n.Elt)
	if elt == nil {
		return nil
	}
	return &ast.ArrayType{Elt: elt}
}

// cloneMapTypeStructural clones a map type.
//
// It refuses (nil) when either the key or value is a non-cacheable (position-dependent)
// shape.
//
// Takes n (*ast.MapType) which is the map type to clone.
//
// Returns ast.Expr which is the cloned map type, or nil when n is not a
// position-independent shape and the caller should re-parse.
func cloneMapTypeStructural(n *ast.MapType) ast.Expr {
	key, value := cloneExprStructural(n.Key), cloneExprStructural(n.Value)
	if key == nil || value == nil {
		return nil
	}
	return &ast.MapType{Key: key, Value: value}
}

// cloneIndexExprStructural clones a single-type-argument generic instantiation (T[A]).
//
// Takes n (*ast.IndexExpr) which is the generic instantiation to clone.
//
// Returns ast.Expr which is the cloned instantiation, or nil when n is not a
// position-independent shape and the caller should re-parse.
func cloneIndexExprStructural(n *ast.IndexExpr) ast.Expr {
	x, index := cloneExprStructural(n.X), cloneExprStructural(n.Index)
	if x == nil || index == nil {
		return nil
	}
	return &ast.IndexExpr{X: x, Index: index}
}

// cloneIndexListExprStructural clones a multi-argument generic instantiation (T[A, B]).
//
// Takes n (*ast.IndexListExpr) which is the generic instantiation to clone.
//
// Returns ast.Expr which is the cloned instantiation, or nil when n is not a
// position-independent shape and the caller should re-parse.
func cloneIndexListExprStructural(n *ast.IndexListExpr) ast.Expr {
	x := cloneExprStructural(n.X)
	if x == nil {
		return nil
	}
	indices := make([]ast.Expr, len(n.Indices))
	for i, index := range n.Indices {
		clone := cloneExprStructural(index)
		if clone == nil {
			return nil
		}
		indices[i] = clone
	}
	return &ast.IndexListExpr{X: x, Indices: indices}
}

// ASTToTypeString converts an AST expression back into its Go type string representation.
//
// Takes expression (ast.Expr) which is the AST expression to convert.
// Takes pkgAlias (...string) which optionally qualifies unqualified identifiers in the
// AST with the given package alias.
//
// Returns string which is the Go type representation of the expression.
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

// IsPrimitiveOrBuiltin reports whether a type name is a Go primitive, a pre-declared
// identifier, or follows the naming pattern for a generic type parameter. These types
// should not have a package name prefix.
//
// Takes name (string) which is the type name to check.
//
// Returns bool which is true if the name is a primitive, built-in, or generic type
// parameter.
func IsPrimitiveOrBuiltin(name string) bool {
	return primitiveAndBuiltinSet[name]
}

// UnqualifyTypeExpr removes package qualifiers from a type expression.
//
// It takes an expression such as pkg.Type or *pkg.Type and returns the unqualified form
// such as Type or *Type. It recurses through pointers, slices, maps, generics, and
// function types.
//
// Takes expression (ast.Expr) which is the type expression to unqualify.
//
// Returns ast.Expr which is the unqualified type expression, or the original expression
// if no qualification was present.
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

// tryFastPathConversion tries to convert simple AST expressions without full AST work.
//
// Takes expression (ast.Expr) which is the expression to convert.
// Takes pAlias (string) which overrides the package alias if not empty.
//
// Returns string which is the converted type string, or empty on failure.
// Returns bool which is true if the fast path worked, false otherwise.
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
// Returns bool which is true when the expression was a simple identifier.
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

// trySelectorFastPath handles selector expressions that are already in qualified form
// (pkg.Type).
//
// Takes expression (ast.Expr) which is the expression to check for selector form.
//
// Returns string which is the qualified name in "pkg.Type" format.
// Returns bool which indicates whether the expression was a valid selector.
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

// tryPointerSelectorFastPath handles pointers to selector expressions (*pkg.Type).
//
// Takes expression (ast.Expr) which is the expression to check and convert.
//
// Returns string which is the formatted type string (e.g. "*pkg.Type").
// Returns bool which is true when the fast path conversion succeeded.
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
// Takes expression (ast.Expr) which is the expression to check for the slice pattern.
//
// Returns string which contains the formatted type string if successful.
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
// Takes expression (ast.Expr) which is the expression to convert to a string.
// Takes pAlias (string) which is the package alias to qualify identifiers.
//
// Returns string which is the printed representation of the expression.
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

// qualifyAST recursively traverses an AST expression and prepends a package alias to any
// unqualified identifiers that are not primitives or built-ins.
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
// Returns ast.Expr which is a selector expression with the package alias, or the original
// identifier if it is a primitive or built-in type.
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
// Returns *ast.StructType which is the same struct with its field types updated.
func qualifyStructType(n *ast.StructType, pkgAlias string) *ast.StructType {
	if n.Fields != nil {
		for _, f := range n.Fields.List {
			f.Type = qualifyAST(f.Type, pkgAlias)
		}
	}
	return n
}

// qualifyIndexExpr adds a package alias to a generic type with one type parameter.
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

// qualifyIndexListExpr adds a package alias to a generic type that has more than one type
// parameter.
//
// Takes n (*ast.IndexListExpr) which is the generic type expression to update.
// Takes pkgAlias (string) which is the package alias to add before identifiers.
//
// Returns *ast.IndexListExpr which is the updated expression with the package alias added
// to identifiers.
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

// deepCopyAST creates a deep copy of an AST expression.
//
// It copies by printing the node to a string and parsing it again. This prevents side
// effects, as AST nodes are mutable pointers. The reparse also re-establishes the
// position information go/printer relies on to keep composite types (structs, interfaces)
// on a single line, which a structural clone would lose. Parsing goes through the pooled
// FileSet to avoid the per-call FileSet allocation.
//
// Takes node (ast.Expr) which is the expression to copy.
//
// Returns ast.Expr which is a new, independent copy of the input expression.
func deepCopyAST(node ast.Expr) ast.Expr {
	var buffer bytes.Buffer
	if err := printer.Fprint(&buffer, sharedPrintFileSet, node); err != nil {
		return ast.NewIdent("any /* internal copy error */")
	}

	newNode, err := parseTypeExpr(buffer.String())
	if err != nil {
		return ast.NewIdent("any")
	}
	return newNode
}

// unqualifyStarExpr removes package qualifiers from pointer types. For example, *pkg.Type
// becomes *Type.
//
// Takes n (*ast.StarExpr) which is the pointer type expression to process.
//
// Returns ast.Expr which is the pointer expression without package qualifiers, or the
// original if no change was needed.
func unqualifyStarExpr(n *ast.StarExpr) ast.Expr {
	unqualifiedX := UnqualifyTypeExpr(n.X)
	if unqualifiedX != n.X {
		return &ast.StarExpr{X: unqualifiedX}
	}
	return n
}

// unqualifyArrayType removes package qualifiers from the element type of an array or
// slice, converting []pkg.Type to []Type.
//
// Takes n (*ast.ArrayType) which is the array or slice type to process.
//
// Returns ast.Expr which is the type with qualifiers removed, or the original if no
// changes were needed.
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
// Returns ast.Expr which is the map type with unqualified key and value types, or the
// original if no changes were needed.
func unqualifyMapType(n *ast.MapType) ast.Expr {
	newKey := UnqualifyTypeExpr(n.Key)
	newValue := UnqualifyTypeExpr(n.Value)
	if newKey != n.Key || newValue != n.Value {
		return &ast.MapType{Key: newKey, Value: newValue}
	}
	return n
}

// unqualifyIndexExpr removes package qualifiers from a generic type expression. For
// example, Box[pkg.User] becomes Box[User] by removing the package prefix from both the
// base type and the type argument.
//
// Takes n (*ast.IndexExpr) which is the generic type expression to process.
//
// Returns ast.Expr which is the expression without package qualifiers, or n unchanged if
// no qualifiers were present.
func unqualifyIndexExpr(n *ast.IndexExpr) ast.Expr {
	newX := UnqualifyTypeExpr(n.X)
	newIndex := UnqualifyTypeExpr(n.Index)
	if newX != n.X || newIndex != n.Index {
		return &ast.IndexExpr{X: newX, Index: newIndex}
	}
	return n
}

// unqualifyIndexListExpr removes package names from generic type expressions such as
// Map[pkg.Key, pkg.Value].
//
// Takes n (*ast.IndexListExpr) which is the generic type expression to process.
//
// Returns ast.Expr which is the expression with package names removed, or the original if
// no changes were needed.
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

// unqualifyFuncType removes package qualifiers from all types in a function type
// signature.
//
// Takes n (*ast.FuncType) which is the function type to process.
//
// Returns *ast.FuncType which is the same node with all parameter and result types
// unqualified.
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
