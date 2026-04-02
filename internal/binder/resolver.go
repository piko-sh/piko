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

package binder

import (
	"errors"
	"fmt"
	"reflect"

	"piko.sh/piko/internal/ast/ast_domain"
)

// nestedMapAccessInfo holds the AST nodes for a nested map access pattern.
type nestedMapAccessInfo struct {
	// baseMember is the parent member expression for nested map access.
	baseMember *ast_domain.MemberExpression

	// indexExpr is the AST node for the map index access.
	indexExpr *ast_domain.IndexExpression
}

// isNestedMapAccess checks if the node contains a nested map access pattern.
//
// Takes currentVal (reflect.Value) which is the current struct value being
// checked.
// Takes node (*ast_domain.MemberExpression) which is the member
// expression to check.
// Takes maxPathDepth (int) which limits recursion depth for cache
// lookups to avoid repeated atomic loads.
//
// Returns nestedMapAccessInfo which contains the typed AST nodes if the
// pattern matches.
// Returns bool which is true if the node is a nested map access, false
// otherwise.
func (b *ASTBinder) isNestedMapAccess(currentVal reflect.Value, node *ast_domain.MemberExpression, maxPathDepth int) (nestedMapAccessInfo, bool) {
	baseMember, ok := node.Base.(*ast_domain.MemberExpression)
	if !ok {
		return nestedMapAccessInfo{}, false
	}

	indexExpr, ok := baseMember.Base.(*ast_domain.IndexExpression)
	if !ok {
		return nestedMapAccessInfo{}, false
	}

	baseIdent, ok := indexExpr.Base.(*ast_domain.Identifier)
	if !ok {
		return nestedMapAccessInfo{}, false
	}

	if currentVal.Kind() != reflect.Struct {
		return nestedMapAccessInfo{}, false
	}

	var fieldVal reflect.Value
	structMeta := b.cache.get(currentVal.Type(), maxPathDepth)
	if fi, ok := structMeta.Fields[baseIdent.Name]; ok {
		fieldVal = fieldByIndexSafe(currentVal, fi.Index)
	} else {
		fieldVal = currentVal.FieldByName(baseIdent.Name)
	}

	if !fieldVal.IsValid() || fieldVal.Kind() != reflect.Map {
		return nestedMapAccessInfo{}, false
	}

	return nestedMapAccessInfo{baseMember: baseMember, indexExpr: indexExpr}, true
}

// findTargetByAST dispatches navigation to a field using AST expression types
// with depth tracking.
//
// It separates the strategy (dispatching) from the implementation (handling
// each type). The limits are passed through to avoid repeated atomic loads in
// recursive calls.
//
// Takes currentVal (reflect.Value) which is the value to navigate
// from.
// Takes expression (ast_domain.Expression) which is the AST node to
// evaluate.
// Takes fullPath (string) which is the complete path for error
// messages.
// Takes depth (int) which tracks recursion depth.
// Takes limits (binderOptions) which provides recursion and path
// constraints.
//
// Returns reflect.Value which is the target field value.
// Returns error when the depth limit is exceeded or the path is
// invalid.
func (b *ASTBinder) findTargetByAST(currentVal reflect.Value, expression ast_domain.Expression, fullPath string, depth int, limits binderOptions) (reflect.Value, error) {
	if err := checkDepthLimit(fullPath, depth, limits.maxPathDepth); err != nil {
		return reflect.Value{}, err
	}

	currentVal = dereferencePointer(currentVal)

	switch node := expression.(type) {
	case *ast_domain.Identifier:
		if currentVal.Kind() != reflect.Struct {
			return reflect.Value{}, errSetField{err: fmt.Errorf("cannot look up field %q on non-struct type %s", node.Name, currentVal.Type()), path: fullPath, field: node.Name, fieldType: ""}
		}
		structMeta := b.cache.get(currentVal.Type(), limits.maxPathDepth)
		if fi, ok := structMeta.Fields[node.Name]; ok {
			return fieldByIndexSafe(currentVal, fi.Index), nil
		}
		return findFieldByName(currentVal, node.Name, fullPath)
	case *ast_domain.IndexExpression:
		return b.resolveIndexExpression(currentVal, node, fullPath, depth, limits)
	case *ast_domain.MemberExpression:
		return b.resolveMemberExpression(currentVal, node, fullPath, depth, limits)
	default:
		return reflect.Value{}, errInvalidPath{path: fullPath, err: errors.New("unsupported base expression type in form path: " + fmt.Sprintf("%T", node))}
	}
}

// resolveMemberExpression resolves a member expression by recursively
// processing base and property. The limits are passed through to avoid
// repeated atomic loads in recursive calls.
//
// Takes currentVal (reflect.Value) which is the current value being bound.
// Takes node (*ast_domain.MemberExpression) which is the member
// expression to resolve.
// Takes fullPath (string) which is the full path for error reporting.
// Takes depth (int) which tracks recursion depth.
// Takes limits (binderOptions) which provides binding constraints.
//
// Returns reflect.Value which is the resolved value of the member expression.
// Returns error when the base or property cannot be resolved.
func (b *ASTBinder) resolveMemberExpression(currentVal reflect.Value, node *ast_domain.MemberExpression, fullPath string, depth int, limits binderOptions) (reflect.Value, error) {
	baseVal, err := b.findTargetByAST(currentVal, node.Base, fullPath, depth+1, limits)
	if err != nil {
		return reflect.Value{}, err
	}
	return b.findTargetByAST(baseVal, node.Property, fullPath, depth+1, limits)
}

// resolveIndexExpression handles index access for slices, arrays, and maps.
// It recursively resolves the base expression to support nested paths like
// "parent.child[0]", then dispatches to the appropriate handler based on the
// resolved value's kind.
//
// Takes currentVal (reflect.Value) which is the starting value to resolve from.
// Takes node (*ast_domain.IndexExpression) which is the AST node for
// the index access.
// Takes fullPath (string) which is the dot-separated path for error messages.
// Takes depth (int) which tracks recursion depth during resolution.
// Takes limits (binderOptions) which provides bounds to avoid repeated atomic
// loads in recursive calls.
//
// Returns reflect.Value which is the value at the indexed position.
// Returns error when the base cannot be resolved or the field is not indexable.
func (b *ASTBinder) resolveIndexExpression(currentVal reflect.Value, node *ast_domain.IndexExpression, fullPath string, depth int, limits binderOptions) (reflect.Value, error) {
	sliceOrMapVal, err := b.findTargetByAST(currentVal, node.Base, fullPath, depth+1, limits)
	if err != nil {
		return reflect.Value{}, err
	}

	sliceOrMapVal = dereferenceIndirections(sliceOrMapVal)
	sliceOrMapVal = initialiseNilPointer(sliceOrMapVal)

	switch sliceOrMapVal.Kind() {
	case reflect.Slice:
		return resolveSliceIndex(sliceOrMapVal, node, fullPath, "sliceElement", limits.maxSliceSize)
	case reflect.Map:
		return resolveMapIndex(sliceOrMapVal, node, fullPath)
	case reflect.Struct:
		return resolveStructFieldByStringIndex(sliceOrMapVal, node, fullPath)
	default:
		return reflect.Value{}, errSetField{err: fmt.Errorf("field is not a slice, map, or struct, but got %s", sliceOrMapVal.Kind()), path: fullPath, field: "", fieldType: ""}
	}
}

// isDirectMapAccess checks if the node represents a direct map access pattern.
//
// Takes node (*ast_domain.MemberExpression) which is the member
// expression to check.
//
// Returns *ast_domain.IndexExpression which is the index expression
// if found, or nil.
// Returns bool which is true when the node is a direct map access.
func isDirectMapAccess(node *ast_domain.MemberExpression) (*ast_domain.IndexExpression, bool) {
	indexExpr, ok := node.Base.(*ast_domain.IndexExpression)
	return indexExpr, ok
}

// convertIndexToMapKey converts an AST index expression to a map key value.
//
// Takes indexExpr (*ast_domain.IndexExpression) which holds the index to convert.
// Takes keyType (reflect.Type) which specifies the target map key type.
// Takes fullPath (string) which provides context for error messages.
//
// Returns reflect.Value which holds the converted map key.
// Returns error when the index is not an integer or string literal, or when
// the key conversion fails.
func convertIndexToMapKey(indexExpr *ast_domain.IndexExpression, keyType reflect.Type, fullPath string) (reflect.Value, error) {
	var keyString string
	switch indexNode := indexExpr.Index.(type) {
	case *ast_domain.IntegerLiteral:
		keyString = indexNode.String()
	case *ast_domain.StringLiteral:
		keyString = indexNode.Value
	default:
		return reflect.Value{}, errInvalidPath{path: fullPath, err: fmt.Errorf("map index must be an integer or string literal, got %T", indexExpr.Index)}
	}

	mapKey, err := convertMapKey(keyString, keyType)
	if err != nil {
		return reflect.Value{}, errInvalidPath{path: fullPath, err: fmt.Errorf("could not convert map key: %w", err)}
	}
	return mapKey, nil
}

// getMapElementWorkingCopy creates a copy of a map element that can be changed.
//
// Takes mapVal (reflect.Value) which is the map to get the element from.
// Takes mapKey (reflect.Value) which is the key for the element.
// Takes elemType (reflect.Type) which is the type of the map elements.
//
// Returns reflect.Value which is a copy of the element that can be written to,
// or a new zero value if the element does not exist.
func getMapElementWorkingCopy(mapVal reflect.Value, mapKey reflect.Value, elemType reflect.Type) reflect.Value {
	element := mapVal.MapIndex(mapKey)

	if !element.IsValid() {
		if elemType.Kind() == reflect.Pointer {
			return reflect.New(elemType.Elem())
		}
		return reflect.New(elemType).Elem()
	}

	if elemType.Kind() == reflect.Pointer {
		if element.IsNil() {
			return reflect.New(elemType.Elem())
		}
		workingCopy := reflect.New(elemType.Elem())
		workingCopy.Elem().Set(element.Elem())
		return workingCopy.Elem().Addr()
	}

	workingCopy := reflect.New(elemType).Elem()
	workingCopy.Set(element)
	return workingCopy
}

// getTargetValueForMapElement returns the value to use when setting map
// elements in a nested structure.
//
// Takes workingCopy (reflect.Value) which holds the current element value.
// Takes elemType (reflect.Type) which specifies the element's type.
//
// Returns reflect.Value which is the inner value if elemType is a pointer,
// or the original workingCopy otherwise.
func getTargetValueForMapElement(workingCopy reflect.Value, elemType reflect.Type) reflect.Value {
	if elemType.Kind() == reflect.Pointer {
		return workingCopy.Elem()
	}
	return workingCopy
}

// findFieldByName finds a struct field by its name or json tag.
//
// Takes currentVal (reflect.Value) which is the struct to search in.
// Takes name (string) which is the field name or json tag to find.
// Takes fullPath (string) which is the full path used in error messages.
//
// Returns reflect.Value which is the field that was found.
// Returns error when no field matches the given name or json tag.
func findFieldByName(currentVal reflect.Value, name, fullPath string) (reflect.Value, error) {
	field := currentVal.FieldByName(name)
	if field.IsValid() {
		return field, nil
	}

	for structField, fieldValue := range currentVal.Fields() {
		if !structField.IsExported() {
			continue
		}

		tag := structField.Tag.Get("json")
		if tag == "-" {
			continue
		}

		tagName := tag
		for commaIndex := range len(tag) {
			if tag[commaIndex] == ',' {
				tagName = tag[:commaIndex]
				break
			}
		}

		if tagName == name {
			return fieldValue, nil
		}
	}

	return reflect.Value{}, errSetField{err: errors.New(errFieldNotFound), path: fullPath, field: name, fieldType: ""}
}

// resolveSliceIndex handles slice indexing with automatic growth.
//
// Takes sliceVal (reflect.Value) which is the slice to index into.
// Takes node (*ast_domain.IndexExpression) which contains the index expression.
// Takes fullPath (string) which is the path for error messages.
// Takes fieldName (string) which is the field name for error messages.
// Takes maxSliceSize (int) which limits slice growth to prevent memory issues.
//
// Returns reflect.Value which is the element at the given index.
// Returns error when the index is not an integer literal, is negative, or
// exceeds the maximum slice size.
func resolveSliceIndex(sliceVal reflect.Value, node *ast_domain.IndexExpression, fullPath string, fieldName string, maxSliceSize int) (reflect.Value, error) {
	indexLit, ok := node.Index.(*ast_domain.IntegerLiteral)
	if !ok {
		return reflect.Value{}, errInvalidPath{path: fullPath, err: errors.New("slice index must be an integer literal")}
	}

	index := int(indexLit.Value)
	if index < 0 {
		return reflect.Value{}, errInvalidPath{path: fullPath, err: errors.New("slice index cannot be negative")}
	}

	if err := growSliceToFitIndex(sliceVal, index, maxSliceSize); err != nil {
		return reflect.Value{}, errSetField{err: err, path: fullPath, field: fieldName, fieldType: ""}
	}

	return sliceVal.Index(index), nil
}

// resolveMapIndex finds or creates a map element at the given index.
// It creates the map if nil and sets up missing elements as needed.
//
// Takes mapVal (reflect.Value) which is the map to index into.
// Takes node (*ast_domain.IndexExpression) which contains the index expression.
// Takes fullPath (string) which is the path used in error messages.
//
// Returns reflect.Value which is the element at the given map index.
// Returns error when the index cannot be changed to a valid map key.
func resolveMapIndex(mapVal reflect.Value, node *ast_domain.IndexExpression, fullPath string) (reflect.Value, error) {
	if mapVal.IsNil() {
		mapVal.Set(reflect.MakeMap(mapVal.Type()))
	}

	mapType := mapVal.Type()
	keyType := mapType.Key()
	elemType := mapType.Elem()

	mapKey, err := convertIndexToMapKey(node, keyType, fullPath)
	if err != nil {
		return reflect.Value{}, err
	}

	element := mapVal.MapIndex(mapKey)
	if !element.IsValid() {
		element = createMapElement(elemType)
		mapVal.SetMapIndex(mapKey, element)
		element = mapVal.MapIndex(mapKey)
	}

	if element.Kind() == reflect.Interface && !element.IsNil() {
		element = element.Elem()
	}

	if element.Kind() == reflect.Pointer && element.IsNil() {
		newElem := reflect.New(element.Type().Elem())
		mapVal.SetMapIndex(mapKey, newElem)
		element = newElem
	}

	return element, nil
}

// resolveStructFieldByStringIndex handles bracket-notation access on a struct
// value where flattened JSON form data produces paths like
// shippingAddress['street'] and the bracket index corresponds to a struct
// field name or json tag.
//
// Takes structVal (reflect.Value) which is the struct to access.
// Takes node (*ast_domain.IndexExpression) which contains the string index.
// Takes fullPath (string) which is the path for error reporting.
//
// Returns reflect.Value which is the resolved struct field.
// Returns error when the index is not a string literal or the field is not
// found.
func resolveStructFieldByStringIndex(structVal reflect.Value, node *ast_domain.IndexExpression, fullPath string) (reflect.Value, error) {
	strIndex, ok := node.Index.(*ast_domain.StringLiteral)
	if !ok {
		return reflect.Value{}, errSetField{
			err:  fmt.Errorf("struct field access requires a string index, got %T", node.Index),
			path: fullPath,
		}
	}
	return findFieldByName(structVal, strIndex.Value, fullPath)
}

// createMapElement creates a new zero-value element for a map.
//
// When elemType is a pointer kind, returns a new pointer to the element type.
// Otherwise, returns the element value directly.
//
// Takes elemType (reflect.Type) which specifies the type of element to create.
//
// Returns reflect.Value which is the newly created zero-value element.
func createMapElement(elemType reflect.Type) reflect.Value {
	if elemType.Kind() == reflect.Pointer {
		return reflect.New(elemType.Elem())
	}
	return reflect.New(elemType).Elem()
}
