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

// This file holds the shared logic for recursively walking and transforming
// type hierarchies, such as applying generic type substitutions.

import (
	"go/ast"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// typeWalker provides a base for recursive AST walkers that track state.
type typeWalker struct {
	// querier provides type information lookups for the current package.
	querier *TypeQuerier
}

// applyGenericSubstitutions replaces generic type parameters in an AST node.
// It sends composite types to dedicated helper functions.
//
// Takes typeAST (ast.Expr) which is the type expression to transform.
// Takes substMap (map[string]ast.Expr) which maps type parameter names to
// their concrete replacements.
//
// Returns ast.Expr which is the transformed type with substitutions applied,
// or the original if no substitution was needed.
func applyGenericSubstitutions(typeAST ast.Expr, substMap map[string]ast.Expr) ast.Expr {
	if typeAST == nil || len(substMap) == 0 {
		return typeAST
	}

	switch t := typeAST.(type) {
	case *ast.Ident:
		if replacement, ok := substMap[t.Name]; ok {
			return replacement
		}
	case *ast.StarExpr:
		return substituteInStarExpr(t, substMap)
	case *ast.ArrayType:
		return substituteInArrayType(t, substMap)
	case *ast.MapType:
		return substituteInMapType(t, substMap)
	case *ast.IndexExpr:
		return substituteInIndexExpr(t, substMap)
	case *ast.IndexListExpr:
		return substituteInIndexListExpr(t, substMap)
	}
	return typeAST
}

// substituteInStarExpr applies type substitutions to a pointer type.
//
// Takes t (*ast.StarExpr) which is the pointer type to process.
// Takes substMap (map[string]ast.Expr) which maps type names to their
// replacement expressions.
//
// Returns ast.Expr which is a new StarExpr with the substituted inner type,
// or the original expression if no change was needed.
func substituteInStarExpr(t *ast.StarExpr, substMap map[string]ast.Expr) ast.Expr {
	newX := applyGenericSubstitutions(t.X, substMap)
	if newX != t.X {
		return &ast.StarExpr{X: newX}
	}
	return t
}

// substituteInArrayType replaces type parameters in an array or slice type.
//
// Takes t (*ast.ArrayType) which is the array or slice type to process.
// Takes substMap (map[string]ast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns ast.Expr which is a new array type with substitutions applied, or
// the original type if no changes were needed.
func substituteInArrayType(t *ast.ArrayType, substMap map[string]ast.Expr) ast.Expr {
	newElt := applyGenericSubstitutions(t.Elt, substMap)
	if newElt != t.Elt {
		return &ast.ArrayType{Len: t.Len, Elt: newElt}
	}
	return t
}

// substituteInMapType applies type substitutions to a map's key and value
// types.
//
// Takes t (*ast.MapType) which is the map type to transform.
// Takes substMap (map[string]ast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns ast.Expr which is a new map type with substitutions applied, or the
// original if no changes were needed.
func substituteInMapType(t *ast.MapType, substMap map[string]ast.Expr) ast.Expr {
	newKey := applyGenericSubstitutions(t.Key, substMap)
	newVal := applyGenericSubstitutions(t.Value, substMap)
	if newKey != t.Key || newVal != t.Value {
		return &ast.MapType{Key: newKey, Value: newVal}
	}
	return t
}

// substituteInIndexExpr applies type substitutions to an index expression.
//
// Takes t (*ast.IndexExpr) which is the index expression to change.
// Takes substMap (map[string]ast.Expr) which maps type parameter names to
// their concrete types.
//
// Returns ast.Expr which is the updated expression, or the original if no
// substitutions were made.
func substituteInIndexExpr(t *ast.IndexExpr, substMap map[string]ast.Expr) ast.Expr {
	newX := applyGenericSubstitutions(t.X, substMap)
	newIndex := applyGenericSubstitutions(t.Index, substMap)
	if newX != t.X || newIndex != t.Index {
		return &ast.IndexExpr{X: newX, Index: newIndex}
	}
	return t
}

// substituteInIndexListExpr applies type substitutions to a generic type that
// has more than one type argument.
//
// Takes t (*ast.IndexListExpr) which is the generic type expression to process.
// Takes substMap (map[string]ast.Expr) which maps type parameter names to their
// replacement expressions.
//
// Returns ast.Expr which is a new expression with substitutions applied, or
// the original if no changes were needed.
func substituteInIndexListExpr(t *ast.IndexListExpr, substMap map[string]ast.Expr) ast.Expr {
	newX := applyGenericSubstitutions(t.X, substMap)
	changed := newX != t.X

	newIndices := make([]ast.Expr, len(t.Indices))
	for i, index := range t.Indices {
		newIndex := applyGenericSubstitutions(index, substMap)
		if newIndex != index {
			changed = true
		}
		newIndices[i] = newIndex
	}

	if changed {
		return &ast.IndexListExpr{X: newX, Indices: newIndices}
	}
	return t
}

// extractGenericTypeArguments gets the type arguments from a generic type
// AST node (for example, Box[T] or Map[K, V]).
//
// Takes expression (ast.Expr) which is the expression to get type
// arguments from.
//
// Returns []ast.Expr which holds the type arguments, or nil if the
// expression is not a generic type.
func extractGenericTypeArguments(expression ast.Expr) []ast.Expr {
	currentExpr := expression
	if star, ok := currentExpr.(*ast.StarExpr); ok {
		currentExpr = star.X
	}

	switch t := currentExpr.(type) {
	case *ast.IndexExpr:
		return []ast.Expr{t.Index}
	case *ast.IndexListExpr:
		return t.Indices
	default:
		return nil
	}
}

// findDirectMethodForValue finds a method in the value receiver method set.
// It skips methods that only have a pointer receiver.
//
// Takes namedType (*inspector_dto.Type) which contains the methods to search.
// Takes searcher (*methodSearcher) which specifies the method name to find.
//
// Returns *inspector_dto.FunctionSignature which is the matching method
// signature, or nil if not found.
// Returns *inspector_dto.Method which is the matching method, or nil if not
// found.
func findDirectMethodForValue(namedType *inspector_dto.Type, searcher *methodSearcher) (*inspector_dto.FunctionSignature, *inspector_dto.Method) {
	for _, method := range namedType.Methods {
		if method.Name != searcher.exportedMethodName {
			continue
		}

		if method.IsPointerReceiver {
			continue
		}

		return &method.Signature, method
	}

	return nil, nil
}

// findDirectMethodForPointer finds a method in the full method set, which
// includes both value and pointer receivers.
//
// Takes namedType (*inspector_dto.Type) which holds the methods to search.
// Takes searcher (*methodSearcher) which gives the method name to find.
//
// Returns *inspector_dto.FunctionSignature which is the matching method's
// signature, or nil if not found.
// Returns *inspector_dto.Method which is the matching method, or nil if not
// found.
func findDirectMethodForPointer(namedType *inspector_dto.Type, searcher *methodSearcher) (*inspector_dto.FunctionSignature, *inspector_dto.Method) {
	for _, method := range namedType.Methods {
		if method.Name == searcher.exportedMethodName {
			return &method.Signature, method
		}
	}
	return nil, nil
}
