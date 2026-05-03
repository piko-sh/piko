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

package inspector_domain

// This file is dedicated to a very specific and critical workaround: "cleaning"
// the internal representation of types that the go/types library produces for
// methods on instantiated generic types.
//
// The Problem:
// The go/types library can represent the type `string` inside a method signature
// like `func (b Box[string]) Method() string` as a special *types.TypeParam object
// whose String() method returns "string/* type parameter */".
//
// The Solution:
// This code detects this annotation, extracts the clean type string (e.g., "string"),
// and then uses the official go/parser and the full type-checking context to
// resolve that string back into its proper, canonical *types.Type. This approach
// handles primitives, named types, and complex composite types.

import (
	"go/ast"
	"go/parser"
	"go/types"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

// astResolverPool reuses astResolver instances to reduce allocation pressure
// during generic type cleaning.
var astResolverPool = sync.Pool{
	New: func() any {
		return &astResolver{}
	},
}

// cleaningContext holds the data needed to resolve a type string back to a
// types.Type within a specific file's scope.
type cleaningContext struct {
	// aliasToPath maps import aliases to their full import paths for the
	// file being processed.
	aliasToPath map[string]string

	// allPackages maps import paths to loaded packages for type resolution.
	allPackages map[string]*packages.Package

	// currentPackage is the package where the type is being resolved.
	currentPackage *types.Package
}

// astResolver holds the state needed to convert an ast.Expr to a types.Type.
type astResolver struct {
	// ctx holds the cleaning context for resolving types during AST traversal.
	ctx *cleaningContext
}

// resolve is the main dispatcher for the astResolver. It delegates to
// specific methods based on the AST node type.
//
// Takes e (ast.Expr) which is the expression node to resolve.
//
// Returns types.Type which is the resolved type, or nil if the node type is
// not supported.
func (r *astResolver) resolve(e ast.Expr) types.Type {
	switch node := e.(type) {
	case *ast.Ident:
		return r.resolveIdent(node)
	case *ast.SelectorExpr:
		return r.resolveSelectorExpr(node)
	case *ast.StarExpr:
		return r.resolveStarExpr(node)
	case *ast.ArrayType:
		return r.resolveArrayType(node)
	}
	return nil
}

// resolveIdent finds the type for an identifier.
//
// Takes node (*ast.Ident) which is the identifier to look up.
//
// Returns types.Type which is the found type, or nil if not found.
func (r *astResolver) resolveIdent(node *ast.Ident) types.Type {
	if typeObject := types.Universe.Lookup(node.Name); typeObject != nil {
		return typeObject.Type()
	}
	if typeObject := r.ctx.currentPackage.Scope().Lookup(node.Name); typeObject != nil {
		return typeObject.Type()
	}
	return nil
}

// resolveSelectorExpr resolves a selector expression to its type.
//
// Takes node (*ast.SelectorExpr) which is the selector expression to resolve.
//
// Returns types.Type which is the resolved type, or nil if the selector cannot
// be resolved.
func (r *astResolver) resolveSelectorExpr(node *ast.SelectorExpr) types.Type {
	pkgIdent, ok := node.X.(*ast.Ident)
	if !ok {
		return nil
	}

	pkgAlias := pkgIdent.Name
	typeName := node.Sel.Name

	packagePath, ok := r.ctx.aliasToPath[pkgAlias]
	if !ok {
		return nil
	}

	targetPackage, ok := r.ctx.allPackages[packagePath]
	if !ok || targetPackage.Types == nil {
		return nil
	}

	if typeObject := targetPackage.Types.Scope().Lookup(typeName); typeObject != nil {
		return typeObject.Type()
	}

	return nil
}

// resolveStarExpr resolves a pointer type expression to its types.Type.
//
// Takes node (*ast.StarExpr) which is the pointer type expression to resolve.
//
// Returns types.Type which is the pointer type, or nil if the element type
// cannot be resolved.
func (r *astResolver) resolveStarExpr(node *ast.StarExpr) types.Type {
	if elemType := r.resolve(node.X); elemType != nil {
		return types.NewPointer(elemType)
	}
	return nil
}

// resolveArrayType converts an array type AST node to its types.Type form.
//
// Takes node (*ast.ArrayType) which is the array or slice type node to convert.
//
// Returns types.Type which is a slice type if the node has no length, an array
// type with length zero if a length is present, or nil if the element type
// cannot be resolved.
func (r *astResolver) resolveArrayType(node *ast.ArrayType) types.Type {
	elemType := r.resolve(node.Elt)
	if elemType == nil {
		return nil
	}

	if node.Len == nil {
		return types.NewSlice(elemType)
	}

	return types.NewArray(elemType, 0)
}

// getAstResolver gets an astResolver from the pool and sets it up for use.
//
// Takes ctx (*cleaningContext) which provides the cleaning context for the
// resolver.
//
// Returns *astResolver which is ready to use with the given context.
func getAstResolver(ctx *cleaningContext) *astResolver {
	r, ok := astResolverPool.Get().(*astResolver)
	if !ok {
		r = &astResolver{}
	}
	r.ctx = ctx
	return r
}

// putAstResolver resets the given resolver and returns it to the pool.
//
// Takes r (*astResolver) which is the resolver to reset and return.
func putAstResolver(r *astResolver) {
	r.ctx = nil
	astResolverPool.Put(r)
}

// cleanAnnotatedSignature walks through a signature and cleans any annotated
// type parameters within it.
//
// Takes sig (*types.Signature) which is the signature to clean.
// Takes ctx (*cleaningContext) which holds the cleaning state.
//
// Returns *types.Signature which is the cleaned signature, or the original if
// no changes were needed.
func cleanAnnotatedSignature(sig *types.Signature, ctx *cleaningContext) *types.Signature {
	if sig == nil {
		return nil
	}

	paramsChanged := false
	resultsChanged := false

	var newParams *types.Tuple
	if sig.Params() != nil && sig.Params().Len() > 0 {
		newParams, paramsChanged = cleanAnnotatedTuple(sig.Params(), ctx)
	}

	var newResults *types.Tuple
	if sig.Results() != nil && sig.Results().Len() > 0 {
		newResults, resultsChanged = cleanAnnotatedTuple(sig.Results(), ctx)
	}

	if !paramsChanged && !resultsChanged {
		return sig
	}

	return types.NewSignatureType(sig.Recv(), nil, nil, newParams, newResults, sig.Variadic())
}

// cleanAnnotatedTuple cleans each type within a tuple of function parameters
// or results.
//
// Takes tuple (*types.Tuple) which is the tuple to clean.
// Takes ctx (*cleaningContext) which provides the cleaning settings.
//
// Returns *types.Tuple which is the cleaned tuple.
// Returns bool which is true if any types were changed.
func cleanAnnotatedTuple(tuple *types.Tuple, ctx *cleaningContext) (*types.Tuple, bool) {
	return transformTuple(tuple, func(t types.Type) types.Type {
		return cleanAnnotatedType(t, ctx)
	})
}

// cleanAnnotatedType cleans a type by resolving annotated type parameters or
// passing composite types to their respective cleaners.
//
// Takes typ (types.Type) which is the type to clean.
// Takes ctx (*cleaningContext) which provides the cleaning state and cache.
//
// Returns types.Type which is the cleaned type, or the original if no cleaning
// is needed.
func cleanAnnotatedType(typ types.Type, ctx *cleaningContext) types.Type {
	tp, ok := typ.(*types.TypeParam)
	if !ok || !isAnnotatedTypeParam(tp) {
		return cleanCompositeType(typ, ctx)
	}

	baseTypeName := extractBaseTypeName(tp)
	if resolvedType := resolveCleanedTypeFromString(baseTypeName, ctx); resolvedType != nil {
		return resolvedType
	}

	return typ
}

// cleanCompositeType picks and runs the correct cleaner for a composite type.
// Splits the logic of cleanAnnotatedType into smaller parts.
//
// Takes typ (types.Type) which is the type to clean.
// Takes ctx (*cleaningContext) which tracks visited types and cleaning state.
//
// Returns types.Type which is the cleaned type, or the original if the type is
// not a handled composite type.
func cleanCompositeType(typ types.Type, ctx *cleaningContext) types.Type {
	switch t := typ.(type) {
	case *types.Pointer:
		return cleanPointer(t, ctx)
	case *types.Slice:
		return cleanSlice(t, ctx)
	case *types.Array:
		return cleanArray(t, ctx)
	case *types.Map:
		return cleanMap(t, ctx)
	case *types.Chan:
		return cleanChan(t, ctx)
	case *types.Named:
		return cleanNamed(t, ctx)
	}
	return typ
}

// cleanPointer cleans the element type of a pointer and returns a new pointer
// if the element was changed.
//
// Takes t (*types.Pointer) which is the pointer type to clean.
// Takes ctx (*cleaningContext) which provides the cleaning context.
//
// Returns types.Type which is a new pointer if the element changed, or the
// original pointer if unchanged.
func cleanPointer(t *types.Pointer, ctx *cleaningContext) types.Type {
	if element := cleanAnnotatedType(t.Elem(), ctx); element != t.Elem() {
		return types.NewPointer(element)
	}
	return t
}

// cleanSlice returns a slice type with its element type cleaned of annotations.
//
// Takes t (*types.Slice) which is the slice type to clean.
// Takes ctx (*cleaningContext) which tracks the cleaning state.
//
// Returns types.Type which is the original slice if unchanged, or a new slice
// with a cleaned element type.
func cleanSlice(t *types.Slice, ctx *cleaningContext) types.Type {
	if element := cleanAnnotatedType(t.Elem(), ctx); element != t.Elem() {
		return types.NewSlice(element)
	}
	return t
}

// cleanArray returns a cleaned copy of the array type if its element type
// needs cleaning, otherwise returns the original array.
//
// Takes t (*types.Array) which is the array type to clean.
// Takes ctx (*cleaningContext) which provides the cleaning settings.
//
// Returns types.Type which is either a new array with a cleaned element type
// or the original array if no cleaning was needed.
func cleanArray(t *types.Array, ctx *cleaningContext) types.Type {
	if element := cleanAnnotatedType(t.Elem(), ctx); element != t.Elem() {
		return types.NewArray(element, t.Len())
	}
	return t
}

// cleanMap returns a map type with cleaned key and element types.
//
// Takes t (*types.Map) which is the map type to clean.
// Takes ctx (*cleaningContext) which holds the cleaning state.
//
// Returns types.Type which is the original map if unchanged, or a new map
// with cleaned key and element types.
func cleanMap(t *types.Map, ctx *cleaningContext) types.Type {
	key := cleanAnnotatedType(t.Key(), ctx)
	element := cleanAnnotatedType(t.Elem(), ctx)
	if key != t.Key() || element != t.Elem() {
		return types.NewMap(key, element)
	}
	return t
}

// cleanChan removes annotations from a channel's element type.
//
// Takes t (*types.Chan) which is the channel type to clean.
// Takes ctx (*cleaningContext) which tracks the cleaning state.
//
// Returns types.Type which is a new channel with a cleaned element type, or
// the original if no cleaning was needed.
func cleanChan(t *types.Chan, ctx *cleaningContext) types.Type {
	if element := cleanAnnotatedType(t.Elem(), ctx); element != t.Elem() {
		return types.NewChan(t.Dir(), element)
	}
	return t
}

// cleanNamed removes type annotations from the type arguments of a named
// generic type.
//
// When the type is not generic, returns it unchanged.
//
// Takes t (*types.Named) which is the named type to clean.
// Takes ctx (*cleaningContext) which provides the cleaning context.
//
// Returns types.Type which is the cleaned type with annotations removed from
// its type arguments.
func cleanNamed(t *types.Named, ctx *cleaningContext) types.Type {
	if t.TypeArgs() == nil || t.TypeArgs().Len() == 0 {
		return t
	}

	newArgs := make([]types.Type, t.TypeArgs().Len())
	changed := false
	for i := range t.TypeArgs().Len() {
		argument := t.TypeArgs().At(i)
		newArg := cleanAnnotatedType(argument, ctx)
		if newArg != argument {
			changed = true
		}
		newArgs[i] = newArg
	}

	if changed {
		if inst, err := types.Instantiate(nil, t.Origin(), newArgs, false); err == nil {
			return inst
		}
	}
	return t
}

// isAnnotatedTypeParam checks if a type parameter has the special annotation.
//
// Takes tp (*types.TypeParam) which is the type parameter to check.
//
// Returns bool which is true if the type parameter contains the annotation.
func isAnnotatedTypeParam(tp *types.TypeParam) bool {
	return strings.Contains(tp.String(), "/* type parameter */")
}

// extractBaseTypeName extracts the base type name from a type parameter.
// It removes any trailing annotation comments from the type string.
//
// Takes tp (*types.TypeParam) which is the type parameter to extract from.
//
// Returns string which is the type name without annotations.
func extractBaseTypeName(tp *types.TypeParam) string {
	return strings.TrimSpace(strings.Split(tp.String(), "/*")[0])
}

// resolveCleanedTypeFromString parses a type string and returns the matching
// Go type. It uses the AST resolver to handle parsing and type lookup.
//
// Takes typeString (string) which is the type expression to parse.
// Takes ctx (*cleaningContext) which provides context for type resolution.
//
// Returns types.Type which is the resolved type, or nil if parsing fails.
func resolveCleanedTypeFromString(typeString string, ctx *cleaningContext) types.Type {
	expression, err := parser.ParseExpr(typeString)
	if err != nil {
		return nil
	}

	resolver := getAstResolver(ctx)
	defer putAstResolver(resolver)
	return resolver.resolve(expression)
}
