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

package annotator_domain

import (
	goast "go/ast"
)

// substituteType replaces generic type parameters in a Go type expression.
//
// Substitution keeps the input tree intact. Unchanged nodes and replacements are shared.
//
// Takes expression (goast.Expr) which is the type expression to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// concrete types.
//
// Returns goast.Expr which contains the substitutions or the original expression.
func substituteType(expression goast.Expr, substMap map[string]goast.Expr) goast.Expr {
	if expression == nil || len(substMap) == 0 {
		return expression
	}

	switch n := expression.(type) {
	case *goast.Ident:
		return substituteIdent(n, substMap)
	case *goast.StarExpr:
		return substituteStarExpr(n, substMap)
	case *goast.ArrayType:
		return substituteArrayType(n, substMap)
	case *goast.MapType:
		return substituteMapType(n, substMap)
	case *goast.ChanType:
		return substituteChanType(n, substMap)
	case *goast.FuncType:
		return substituteFuncType(n, substMap)
	case *goast.IndexExpr:
		return substituteIndexExpr(n, substMap)
	case *goast.IndexListExpr:
		return substituteIndexListExpr(n, substMap)
	}
	return expression
}

// substituteIdent replaces a type parameter name with its concrete type.
//
// Takes n (*goast.Ident) which is the identifier to check.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// concrete type expressions.
//
// Returns goast.Expr which is the concrete type if found in the map, or the original
// identifier if not found.
func substituteIdent(n *goast.Ident, substMap map[string]goast.Expr) goast.Expr {
	if replacement, ok := substMap[n.Name]; ok {
		return replacement
	}
	return n
}

// substituteStarExpr replaces type parameters in a pointer type.
//
// Takes n (*goast.StarExpr) which is the pointer type to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// concrete types.
//
// Returns goast.Expr which is a new pointer with the inner type replaced, or the original
// if no change was needed.
func substituteStarExpr(n *goast.StarExpr, substMap map[string]goast.Expr) goast.Expr {
	newX := substituteType(n.X, substMap)
	if newX != n.X {
		return &goast.StarExpr{X: newX}
	}
	return n
}

// substituteArrayType replaces type parameters in array and slice types.
//
// Takes n (*goast.ArrayType) which is the array type node to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// concrete types.
//
// Returns goast.Expr which is a new array type with the element type replaced, or the
// original node if no change was needed.
func substituteArrayType(n *goast.ArrayType, substMap map[string]goast.Expr) goast.Expr {
	newElt := substituteType(n.Elt, substMap)
	if newElt != n.Elt {
		return &goast.ArrayType{Len: n.Len, Elt: newElt}
	}
	return n
}

// substituteMapType replaces type parameters in a map type with their concrete types.
//
// Takes n (*goast.MapType) which is the map type node to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// concrete types.
//
// Returns goast.Expr which is the map type with replaced key and value types, or the
// original node if no changes were made.
func substituteMapType(n *goast.MapType, substMap map[string]goast.Expr) goast.Expr {
	newKey := substituteType(n.Key, substMap)
	newValue := substituteType(n.Value, substMap)
	if newKey != n.Key || newValue != n.Value {
		return &goast.MapType{Key: newKey, Value: newValue}
	}
	return n
}

// substituteChanType replaces type parameters in a channel type with their concrete
// types.
//
// Takes n (*goast.ChanType) which is the channel type to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// concrete types.
//
// Returns goast.Expr which is a new channel type with the element type replaced, or the
// original if no change was needed.
func substituteChanType(n *goast.ChanType, substMap map[string]goast.Expr) goast.Expr {
	newValue := substituteType(n.Value, substMap)
	if newValue != n.Value {
		return &goast.ChanType{Dir: n.Dir, Value: newValue}
	}
	return n
}

// substituteFuncType replaces type parameters in a function type.
//
// Takes n (*goast.FuncType) which is the function type to update.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// concrete types.
//
// Returns goast.Expr which is a new function type with the replacements applied, or the
// original if no changes were needed.
func substituteFuncType(n *goast.FuncType, substMap map[string]goast.Expr) goast.Expr {
	newParams, paramsChanged := substituteFieldList(n.Params, substMap)
	newResults, resultsChanged := substituteFieldList(n.Results, substMap)

	if paramsChanged || resultsChanged {
		return &goast.FuncType{Func: n.Func, Params: newParams, Results: newResults}
	}
	return n
}

// substituteFieldList replaces generic field types in a copy of the field list.
//
// Takes fieldList (*goast.FieldList) which is the list of fields to process.
// Takes substMap (map[string]goast.Expr) which maps type names to their replacement
// types.
//
// Returns *goast.FieldList which is the copied field list, or nil for nil input.
// Returns bool which reports whether any field type changed.
func substituteFieldList(fieldList *goast.FieldList, substMap map[string]goast.Expr) (*goast.FieldList, bool) {
	if fieldList == nil {
		return nil, false
	}

	newFieldList := &goast.FieldList{Opening: fieldList.Opening, Closing: fieldList.Closing}
	changed := false

	for _, field := range fieldList.List {
		newField := *field
		newField.Type = substituteType(field.Type, substMap)
		if newField.Type != field.Type {
			changed = true
		}
		newFieldList.List = append(newFieldList.List, &newField)
	}
	return newFieldList, changed
}

// substituteIndexExpr replaces type parameters in a generic index expression with one
// type argument, such as Box[T].
//
// Takes n (*goast.IndexExpr) which is the index expression to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// replacement expressions.
//
// Returns goast.Expr which is a new expression with replacements applied, or the original
// if nothing changed.
func substituteIndexExpr(n *goast.IndexExpr, substMap map[string]goast.Expr) goast.Expr {
	newX := substituteType(n.X, substMap)
	newIndex := substituteType(n.Index, substMap)
	if newX != n.X || newIndex != n.Index {
		return &goast.IndexExpr{X: newX, Lbrack: n.Lbrack, Index: newIndex, Rbrack: n.Rbrack}
	}
	return n
}

// substituteIndexListExpr replaces type parameters in an index list expression for
// generic types with more than one parameter, such as Map[K, V].
//
// Takes n (*goast.IndexListExpr) which is the index list expression to process.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to their
// concrete types.
//
// Returns goast.Expr which is a new expression with the replacements applied, or the
// original if no changes were needed.
func substituteIndexListExpr(n *goast.IndexListExpr, substMap map[string]goast.Expr) goast.Expr {
	newX := substituteType(n.X, substMap)
	changed := newX != n.X
	newIndices := make([]goast.Expr, len(n.Indices))

	for i, index := range n.Indices {
		newIndices[i] = substituteType(index, substMap)
		if newIndices[i] != index {
			changed = true
		}
	}

	if changed {
		return &goast.IndexListExpr{X: newX, Lbrack: n.Lbrack, Indices: newIndices, Rbrack: n.Rbrack}
	}
	return n
}
