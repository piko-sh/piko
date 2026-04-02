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

// setByAST is the entry point for the recursive AST walker on the slow
// path. It initialises depth tracking and delegates to
// setByASTRecursive.
//
// Takes currentVal (reflect.Value) which is the value to be modified.
// Takes expression (ast_domain.Expression) which is the parsed AST
// expression.
// Takes valueToSet (string) which is the value to assign.
// Takes fullPath (string) which is the full path for error messages.
// Takes limits (binderOptions) which provides recursion and binding
// limits.
//
// Returns error when the value cannot be set or the expression is
// invalid.
func (b *ASTBinder) setByAST(currentVal reflect.Value, expression ast_domain.Expression, valueToSet string, fullPath string, limits binderOptions) error {
	return b.setByASTRecursive(currentVal, expression, valueToSet, fullPath, 0, limits)
}

// setByASTRecursive recursively walks the AST to set a field value at
// the specified path, tracking depth to prevent unbounded recursion.
//
// Takes currentVal (reflect.Value) which is the current struct being
// traversed.
// Takes expression (ast_domain.Expression) which is the AST node to
// process.
// Takes valueToSet (string) which is the value to assign to the
// target field.
// Takes fullPath (string) which is the complete path for error
// reporting.
// Takes depth (int) which tracks recursion depth for limit checking.
// Takes limits (binderOptions) which provides depth limits to avoid
// repeated atomic loads in recursive calls.
//
// Returns error when the depth limit is exceeded, the current value
// is not a struct, or the expression type is unsupported.
func (b *ASTBinder) setByASTRecursive(currentVal reflect.Value, expression ast_domain.Expression, valueToSet string, fullPath string, depth int, limits binderOptions) error {
	if err := checkDepthLimit(fullPath, depth, limits.maxPathDepth); err != nil {
		return fmt.Errorf("checking depth limit for %q: %w", fullPath, err)
	}

	currentVal = dereferencePointer(currentVal)

	if currentVal.Kind() != reflect.Struct {
		return errSetField{err: errors.New("cannot set field on non-struct type: " + currentVal.Type().String()), path: fullPath, field: "", fieldType: ""}
	}

	switch node := expression.(type) {
	case *ast_domain.Identifier:
		return b.handleIdentifierNode(currentVal, node, valueToSet, fullPath, limits)
	case *ast_domain.IndexExpression:
		return b.handleIndexExprNode(currentVal, node, valueToSet, fullPath, depth, limits)
	case *ast_domain.MemberExpression:
		return b.handleMemberExprNode(currentVal, node, valueToSet, fullPath, depth, limits)
	default:
		return errInvalidPath{path: fullPath, err: errors.New("unsupported expression type in form path: " + fmt.Sprintf("%T", node))}
	}
}

// handleIdentifierNode processes an Identifier AST node.
//
// Takes currentVal (reflect.Value) which is the struct value being bound.
// Takes node (*ast_domain.Identifier) which is the identifier node to process.
// Takes valueToSet (string) which is the raw value to assign.
// Takes fullPath (string) which is the path for error messages.
// Takes limits (binderOptions) which controls binding behaviour.
//
// Returns error when field lookup or value conversion fails.
func (b *ASTBinder) handleIdentifierNode(currentVal reflect.Value, node *ast_domain.Identifier, valueToSet string, fullPath string, limits binderOptions) error {
	structMeta := b.cache.get(currentVal.Type(), limits.maxPathDepth)
	fi, foundInCache := structMeta.Fields[node.Name]

	if !foundInCache {
		return b.handleUncachedField(currentVal, node.Name, valueToSet, fullPath, limits.ignoreUnknownKeys)
	}

	if fi.CanDirect {
		return b.convertAndSetDirect(currentVal, valueToSet, fullPath, fi)
	}

	field := fieldByIndexSafe(currentVal, fi.Index)
	return b.convertAndSet(field, valueToSet, fullPath, fi)
}

// handleIndexExprNode processes an IndexExpr AST node (e.g., items[0] or
// config["key"]).
//
// Dispatches to specific handlers based on the resolved value's kind.
//
// Takes currentVal (reflect.Value) which is the current reflection value being
// processed.
// Takes node (*ast_domain.IndexExpression) which is the index expression AST node.
// Takes valueToSet (string) which is the value to assign at the indexed
// position.
// Takes fullPath (string) which is the dot-separated path for error reporting.
// Takes depth (int) which tracks recursion depth for nested expressions.
// Takes limits (binderOptions) which provides size limits and binding options.
//
// Returns error when the base value is not a slice or map, or when the index
// operation fails.
func (b *ASTBinder) handleIndexExprNode(currentVal reflect.Value, node *ast_domain.IndexExpression, valueToSet string, fullPath string, depth int, limits binderOptions) error {
	if innerIndex, ok := node.Base.(*ast_domain.IndexExpression); ok {
		if shouldUseChainedIndexHandler(currentVal, innerIndex, limits.maxPathDepth) {
			return b.handleChainedIndexExpr(currentVal, innerIndex, node.Index, valueToSet, fullPath, depth, limits)
		}
	}

	baseVal, err := b.findTargetByAST(currentVal, node.Base, fullPath, depth+1, limits)
	if err != nil {
		return fmt.Errorf("resolving index expression base for %q: %w", fullPath, err)
	}

	baseVal = dereferenceIndirections(baseVal)
	baseVal = initialiseNilPointer(baseVal)

	switch baseVal.Kind() {
	case reflect.Slice:
		return b.handleSliceIndexExpr(baseVal, node, valueToSet, fullPath, limits.maxSliceSize)
	case reflect.Map:
		return b.handleMapIndexExpr(baseVal, node, valueToSet, fullPath)
	case reflect.Struct:
		return b.handleStructIndexExpr(baseVal, node, valueToSet, fullPath, limits)
	default:
		return errSetField{err: fmt.Errorf("field is not a slice, map, or struct, but got %s", baseVal.Kind()), path: fullPath, field: "", fieldType: ""}
	}
}

// handleSliceIndexExpr processes slice index access for terminal value setting.
//
// Takes sliceVal (reflect.Value) which is the slice to index into.
// Takes node (*ast_domain.IndexExpression) which contains the index expression.
// Takes valueToSet (string) which is the value to assign to the element.
// Takes fullPath (string) which is the path for error reporting.
// Takes maxSliceSize (int) which limits slice expansion.
//
// Returns error when the slice cannot be indexed or the element cannot be set.
func (b *ASTBinder) handleSliceIndexExpr(sliceVal reflect.Value, node *ast_domain.IndexExpression, valueToSet string, fullPath string, maxSliceSize int) error {
	if !sliceVal.CanSet() && !sliceVal.CanAddr() {
		return errInvalidPath{
			path: fullPath,
			err:  errors.New("cannot index into slice obtained from map value (pattern like map[key][index] is not supported)"),
		}
	}

	element, err := resolveSliceIndex(sliceVal, node, fullPath, "sliceElement", maxSliceSize)
	if err != nil {
		return fmt.Errorf("resolving slice index for %q: %w", fullPath, err)
	}

	if !element.CanSet() {
		return errInvalidPath{
			path: fullPath,
			err:  errors.New("cannot set value on non-addressable slice element (may be from a map value)"),
		}
	}

	fi := newFieldInfoForType(fullPath, element.Type())
	return b.convertAndSet(element, valueToSet, fullPath, fi)
}

// handleMapIndexExpr sets a value at a map index.
//
// Takes mapVal (reflect.Value) which is the map to change.
// Takes node (*ast_domain.IndexExpression) which holds the index expression.
// Takes valueToSet (string) which is the value to store at the index.
// Takes fullPath (string) which is the field path used in error messages.
//
// Returns error when chained indexing is not allowed, the map cannot be set
// up, key conversion fails, or value conversion fails.
func (b *ASTBinder) handleMapIndexExpr(mapVal reflect.Value, node *ast_domain.IndexExpression, valueToSet string, fullPath string) error {
	mapType := mapVal.Type()
	elemType := mapType.Elem()

	if err := validateChainedIndexing(mapVal, elemType, fullPath); err != nil {
		return fmt.Errorf("validating chained indexing for %q: %w", fullPath, err)
	}

	if err := initialiseMapIfNil(mapVal, mapType, fullPath); err != nil {
		return fmt.Errorf("initialising map for %q: %w", fullPath, err)
	}

	mapKey, err := convertIndexToMapKey(node, mapType.Key(), fullPath)
	if err != nil {
		return fmt.Errorf("converting index to map key for %q: %w", fullPath, err)
	}

	fi := newFieldInfoForType(fullPath, elemType)
	convertedVal, err := b.convertToType(valueToSet, fi)
	if err != nil {
		return errSetField{path: fullPath, field: fi.Path, fieldType: fi.Type.String(), err: err}
	}

	setMapIndexValue(mapVal, mapKey, convertedVal, elemType)
	return nil
}

// handleStructIndexExpr handles bracket-notation access on a struct field
// where flattened JSON form data produces paths like
// shippingAddress['street'] and the bracket index is a string literal that
// maps to a struct field name or json tag.
//
// Takes structVal (reflect.Value) which is the struct to access.
// Takes node (*ast_domain.IndexExpression) which contains the string index.
// Takes valueToSet (string) which is the value to assign.
// Takes fullPath (string) which is the path for error reporting.
// Takes limits (binderOptions) which provides binding constraints.
//
// Returns error when the index is not a string literal, the field is not
// found, or the value cannot be set.
func (b *ASTBinder) handleStructIndexExpr(structVal reflect.Value, node *ast_domain.IndexExpression, valueToSet string, fullPath string, limits binderOptions) error {
	strIndex, ok := node.Index.(*ast_domain.StringLiteral)
	if !ok {
		return errSetField{
			err:  fmt.Errorf("struct field access requires a string index, got %T", node.Index),
			path: fullPath,
		}
	}
	return b.handleIdentifierNode(structVal, &ast_domain.Identifier{Name: strIndex.Value}, valueToSet, fullPath, limits)
}

// handleChainedIndexExpr handles chained index expressions like
// fields['key1']['key2']. This is needed for map[string]any where intermediate
// values may not exist and need to be created as maps.
//
// Takes currentVal (reflect.Value) which is the struct containing the base map.
// Takes innerIndex (*ast_domain.IndexExpression) which is the inner
// index expression.
// Takes outerKey (ast_domain.Expression) which is the outer key.
// Takes valueToSet (string) which is the value to assign.
// Takes fullPath (string) which is the full path for error reporting.
// Takes depth (int) which tracks recursion depth.
// Takes limits (binderOptions) which provides binding constraints.
//
// Returns error when the chain cannot be resolved or the value cannot be set.
func (b *ASTBinder) handleChainedIndexExpr(
	currentVal reflect.Value,
	innerIndex *ast_domain.IndexExpression,
	outerKey ast_domain.Expression,
	valueToSet string,
	fullPath string,
	depth int,
	limits binderOptions,
) error {
	baseMapVal, err := b.resolveAndValidateBaseMap(currentVal, innerIndex.Base, fullPath, depth, limits)
	if err != nil {
		return fmt.Errorf("resolving base map for chained index at %q: %w", fullPath, err)
	}

	innerKey, err := convertIndexToMapKey(innerIndex, baseMapVal.Type().Key(), fullPath)
	if err != nil {
		return fmt.Errorf("converting inner key for chained index at %q: %w", fullPath, err)
	}

	intermediateVal, err := getOrCreateIntermediateValue(baseMapVal, innerKey, fullPath)
	if err != nil {
		return fmt.Errorf("getting intermediate value for chained index at %q: %w", fullPath, err)
	}

	return b.setValueInIntermediateMap(intermediateVal, outerKey, valueToSet, fullPath)
}

// resolveAndValidateBaseMap resolves the base map from an expression and
// validates it.
//
// Takes currentVal (reflect.Value) which is the current value being bound.
// Takes base (ast_domain.Expression) which is the expression to resolve.
// Takes fullPath (string) which is the full path for error reporting.
// Takes depth (int) which is the current recursion depth.
// Takes limits (binderOptions) which provides binding constraints.
//
// Returns reflect.Value which is the resolved and validated map value.
// Returns error when resolution fails or the value is not a map.
func (b *ASTBinder) resolveAndValidateBaseMap(
	currentVal reflect.Value,
	base ast_domain.Expression,
	fullPath string,
	depth int,
	limits binderOptions,
) (reflect.Value, error) {
	baseMapVal, err := b.findTargetByAST(currentVal, base, fullPath, depth+1, limits)
	if err != nil {
		return reflect.Value{}, err
	}

	baseMapVal = dereferenceIndirections(baseMapVal)
	if baseMapVal.Kind() != reflect.Map {
		return reflect.Value{}, errSetField{
			err:   fmt.Errorf("expected map for chained index access, got %s", baseMapVal.Kind()),
			path:  fullPath,
			field: "",
		}
	}

	if err := initialiseMapIfNil(baseMapVal, baseMapVal.Type(), fullPath); err != nil {
		return reflect.Value{}, err
	}

	return baseMapVal, nil
}

// setValueInIntermediateMap sets a value in the intermediate map using the
// outer key.
//
// Takes intermediateVal (reflect.Value) which is the map to update.
// Takes outerKey (ast_domain.Expression) which identifies the map key.
// Takes valueToSet (string) which is the value to convert and store.
// Takes fullPath (string) which identifies the field location for errors.
//
// Returns error when the key conversion fails or the value cannot be
// converted to the map's element type.
func (b *ASTBinder) setValueInIntermediateMap(
	intermediateVal reflect.Value,
	outerKey ast_domain.Expression,
	valueToSet string,
	fullPath string,
) error {
	outerKeyVal, err := convertOuterKey(outerKey, intermediateVal.Type().Key(), fullPath)
	if err != nil {
		return fmt.Errorf("converting outer key for intermediate map at %q: %w", fullPath, err)
	}

	intermediateElemType := intermediateVal.Type().Elem()
	fi := newFieldInfoForType(fullPath, intermediateElemType)
	convertedVal, err := b.convertToType(valueToSet, fi)
	if err != nil {
		return errSetField{path: fullPath, field: fi.Path, fieldType: fi.Type.String(), err: err}
	}

	setMapIndexValue(intermediateVal, outerKeyVal, convertedVal, intermediateElemType)
	return nil
}

// handleUncachedField processes a field that was not found in the cache.
//
// Takes currentVal (reflect.Value) which is the struct value containing the
// field to set.
// Takes fieldName (string) which is the name of the field to process.
// Takes valueToSet (string) which is the string value to convert and assign.
// Takes fullPath (string) which is the full path for error reporting.
// Takes ignoreUnknown (bool) which controls whether unknown fields are
// silently ignored.
//
// Returns error when the field is not found and ignoreUnknown is false, or
// when the value cannot be converted and set.
func (b *ASTBinder) handleUncachedField(currentVal reflect.Value, fieldName, valueToSet, fullPath string, ignoreUnknown bool) error {
	field := currentVal.FieldByName(fieldName)
	if !field.IsValid() {
		if ignoreUnknown {
			return nil
		}
		return errSetField{err: errors.New(errFieldNotFound), path: fullPath, field: fieldName, fieldType: ""}
	}

	fi := newFieldInfoForType(fieldName, field.Type())
	return b.convertAndSet(field, valueToSet, fullPath, fi)
}

// handleMemberExprNode processes a MemberExpr AST node.
//
// Takes currentVal (reflect.Value) which is the current value being traversed.
// Takes node (*ast_domain.MemberExpression) which is the member
// expression to process.
// Takes valueToSet (string) which is the value to assign at the target path.
// Takes fullPath (string) which is the complete path for error messages.
// Takes depth (int) which tracks recursion depth.
// Takes limits (binderOptions) which controls traversal behaviour.
//
// Returns error when map access fails or the target path cannot be resolved.
func (b *ASTBinder) handleMemberExprNode(currentVal reflect.Value, node *ast_domain.MemberExpression, valueToSet string, fullPath string, depth int, limits binderOptions) error {
	if indexExpr, ok := isDirectMapAccess(node); ok {
		return b.setByASTWithMapSupport(currentVal, indexExpr, node.Property, valueToSet, fullPath, depth, limits)
	}

	if info, ok := b.isNestedMapAccess(currentVal, node, limits.maxPathDepth); ok {
		return b.handleNestedMapAccess(currentVal, node, valueToSet, fullPath, depth, info, limits)
	}

	baseVal, err := b.findTargetByAST(currentVal, node.Base, fullPath, depth+1, limits)
	if err != nil {
		return fmt.Errorf("resolving member expression base for %q: %w", fullPath, err)
	}
	return b.setByASTRecursive(baseVal, node.Property, valueToSet, fullPath, depth+1, limits)
}

// handleNestedMapAccess processes nested map access patterns by combining
// property expressions and delegating to the map support handler.
//
// Takes currentVal (reflect.Value) which is the current value being processed.
// Takes node (*ast_domain.MemberExpression) which is the member
// expression to access.
// Takes valueToSet (string) which is the value to assign at the target path.
// Takes fullPath (string) which is the complete path for error reporting.
// Takes depth (int) which tracks recursion depth.
// Takes info (nestedMapAccessInfo) which contains the base member and index.
// Takes limits (binderOptions) which provides configuration limits.
//
// Returns error when the nested map access or value setting fails.
func (b *ASTBinder) handleNestedMapAccess(
	currentVal reflect.Value,
	node *ast_domain.MemberExpression,
	valueToSet string,
	fullPath string,
	depth int,
	info nestedMapAccessInfo,
	limits binderOptions,
) error {
	combinedProperty := &ast_domain.MemberExpression{
		Base:             info.baseMember.Property,
		Property:         node.Property,
		GoAnnotations:    nil,
		Optional:         false,
		Computed:         false,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     0,
	}
	return b.setByASTWithMapSupport(currentVal, info.indexExpr, combinedProperty, valueToSet, fullPath, depth, limits)
}

// setByASTWithMapSupport handles the special case of setting a field on a
// map element using a get-modify-set pattern because MapIndex returns a
// non-addressable copy.
//
// Takes currentVal (reflect.Value) which is the current value being traversed.
// Takes indexExpr (*ast_domain.IndexExpression) which is the index expression to
// resolve.
// Takes property (ast_domain.Expression) which is the property to set on the
// resolved element.
// Takes valueToSet (string) which is the value to assign.
// Takes fullPath (string) which is the full path for error reporting.
// Takes depth (int) which tracks recursion depth.
// Takes limits (binderOptions) which provides limits to avoid repeated atomic
// loads in recursive calls.
//
// Returns error when the base cannot be resolved, index is out of range, or
// the field is not a slice or map.
func (b *ASTBinder) setByASTWithMapSupport(
	currentVal reflect.Value,
	indexExpr *ast_domain.IndexExpression,
	property ast_domain.Expression,
	valueToSet string,
	fullPath string,
	depth int,
	limits binderOptions,
) error {
	fieldVal, err := b.findTargetByAST(currentVal, indexExpr.Base, fullPath, depth+1, limits)
	if err != nil {
		return fmt.Errorf("resolving map support base for %q: %w", fullPath, err)
	}

	fieldVal = dereferenceIndirections(fieldVal)
	fieldVal = initialiseNilPointer(fieldVal)

	switch fieldVal.Kind() {
	case reflect.Slice:
		element, err := resolveSliceIndex(fieldVal, indexExpr, fullPath, "sliceElement", limits.maxSliceSize)
		if err != nil {
			return fmt.Errorf("resolving slice index in map support for %q: %w", fullPath, err)
		}
		return b.setByASTRecursive(element, property, valueToSet, fullPath, depth+1, limits)

	case reflect.Map:
		return b.setMapElement(fieldVal, indexExpr, property, valueToSet, fullPath, depth, limits)

	case reflect.Struct:
		structField, err := resolveStructFieldByStringIndex(fieldVal, indexExpr, fullPath)
		if err != nil {
			return fmt.Errorf("resolving struct field in map support for %q: %w", fullPath, err)
		}
		return b.setByASTRecursive(structField, property, valueToSet, fullPath, depth+1, limits)

	default:
		return errSetField{err: fmt.Errorf("field is not a slice, map, or struct, but got %s", fieldVal.Kind()), path: fullPath, field: "", fieldType: ""}
	}
}

// setMapElement implements the get-modify-set pattern for map elements.
//
// Takes mapVal (reflect.Value) which is the map to modify.
// Takes indexExpr (*ast_domain.IndexExpression) which specifies the
// key expression.
// Takes property (ast_domain.Expression) which is the nested property to set.
// Takes valueToSet (string) which is the value to assign.
// Takes fullPath (string) which is the path for error messages.
// Takes depth (int) which tracks recursion depth.
// Takes limits (binderOptions) which avoids repeated atomic loads in recursive
// calls.
//
// Returns error when index conversion fails or recursive setting fails.
func (b *ASTBinder) setMapElement(
	mapVal reflect.Value,
	indexExpr *ast_domain.IndexExpression,
	property ast_domain.Expression,
	valueToSet string,
	fullPath string,
	depth int,
	limits binderOptions,
) error {
	if mapVal.IsNil() {
		mapVal.Set(reflect.MakeMap(mapVal.Type()))
	}

	mapType := mapVal.Type()
	keyType := mapType.Key()
	elemType := mapType.Elem()

	mapKey, err := convertIndexToMapKey(indexExpr, keyType, fullPath)
	if err != nil {
		return fmt.Errorf("converting map element key for %q: %w", fullPath, err)
	}

	workingCopy := getMapElementWorkingCopy(mapVal, mapKey, elemType)

	targetVal := getTargetValueForMapElement(workingCopy, elemType)

	err = b.setByASTRecursive(targetVal, property, valueToSet, fullPath, depth+1, limits)
	if err != nil {
		return fmt.Errorf("setting recursive value on map element for %q: %w", fullPath, err)
	}

	mapVal.SetMapIndex(mapKey, workingCopy)
	return nil
}

// checkDepthLimit checks if the current depth is within the allowed limit.
// The maxDepth is passed as a parameter to avoid repeated atomic loads.
//
// Takes fullPath (string) which is the path being checked.
// Takes depth (int) which is the current depth level.
// Takes maxDepth (int) which is the maximum allowed depth.
//
// Returns error when the depth is greater than the maximum allowed limit.
func checkDepthLimit(fullPath string, depth int, maxDepth int) error {
	if maxDepth > 0 && depth > maxDepth {
		return errInvalidPath{path: fullPath, err: fmt.Errorf("path depth exceeds maximum limit of %d", maxDepth)}
	}
	return nil
}

// shouldUseChainedIndexHandler checks if the chained index expression should use
// the special handler for map[string]any. Returns true only when the base map
// has interface{} element type.
//
// Takes currentVal (reflect.Value) which is the struct value containing the
// base map field.
// Takes innerIndex (*ast_domain.IndexExpression) which is the inner index
// expression to inspect for a base identifier.
//
// Returns bool which is true when the base map has an interface{} element
// type, indicating chained index handling is needed.
func shouldUseChainedIndexHandler(currentVal reflect.Value, innerIndex *ast_domain.IndexExpression, _ int) bool {
	baseIdent, ok := innerIndex.Base.(*ast_domain.Identifier)
	if !ok {
		return false
	}

	currentVal = dereferencePointer(currentVal)
	if currentVal.Kind() != reflect.Struct {
		return false
	}

	field := findFieldByNameOrTag(currentVal, baseIdent.Name)
	if !field.IsValid() || field.Kind() != reflect.Map {
		return false
	}

	return field.Type().Elem().Kind() == reflect.Interface
}

// findFieldByNameOrTag finds a struct field by name or by its json tag.
//
// Takes structVal (reflect.Value) which is the struct to search within.
// Takes name (string) which is the field name or json tag to find.
//
// Returns reflect.Value which is the matching field, or an invalid value if
// not found.
func findFieldByNameOrTag(structVal reflect.Value, name string) reflect.Value {
	field := structVal.FieldByName(name)
	if field.IsValid() {
		return field
	}
	return findFieldByJSONTag(structVal, name)
}

// findFieldByJSONTag searches for a field by its JSON tag name.
//
// Takes structVal (reflect.Value) which is the struct to search within.
// Takes name (string) which is the JSON tag name to find.
//
// Returns reflect.Value which is the matching field, or an invalid Value if
// not found.
func findFieldByJSONTag(structVal reflect.Value, name string) reflect.Value {
	for sf, field := range structVal.Fields() {
		tag := sf.Tag.Get("json")
		if tag == "-" {
			continue
		}
		tagName := extractJSONTagName(tag)
		if tagName == name {
			return field
		}
	}
	return reflect.Value{}
}

// extractJSONTagName extracts the field name from a json tag, excluding options.
//
// Takes tag (string) which is the raw json struct tag value.
//
// Returns string which is the field name portion before any comma.
func extractJSONTagName(tag string) string {
	for index := range len(tag) {
		if tag[index] == ',' {
			return tag[:index]
		}
	}
	return tag
}

// getOrCreateIntermediateValue gets or creates the intermediate value for
// chained access.
//
// Takes baseMapVal (reflect.Value) which is the map to retrieve or create in.
// Takes innerKey (reflect.Value) which is the key for the intermediate value.
// Takes fullPath (string) which is the full path for error reporting.
//
// Returns reflect.Value which is the dereferenced intermediate map value.
// Returns error when the intermediate value is not a map type.
func getOrCreateIntermediateValue(baseMapVal, innerKey reflect.Value, fullPath string) (reflect.Value, error) {
	elemType := baseMapVal.Type().Elem()
	intermediateVal := baseMapVal.MapIndex(innerKey)

	if needsIntermediateCreation(intermediateVal, elemType) {
		intermediateVal = createIntermediateMapValue(baseMapVal, innerKey, elemType)
	}

	intermediateVal = dereferenceIndirections(intermediateVal)
	if intermediateVal.Kind() != reflect.Map {
		return reflect.Value{}, errSetField{
			err:   fmt.Errorf("intermediate value is not a map, got %s", intermediateVal.Kind()),
			path:  fullPath,
			field: "",
		}
	}

	return intermediateVal, nil
}

// needsIntermediateCreation checks if an intermediate value needs to be created.
//
// Takes intermediateVal (reflect.Value) which is the current intermediate value
// to check.
// Takes elemType (reflect.Type) which is the expected element type.
//
// Returns bool which is true if the value is invalid or is a nil interface.
func needsIntermediateCreation(intermediateVal reflect.Value, elemType reflect.Type) bool {
	if !intermediateVal.IsValid() {
		return true
	}
	if elemType.Kind() == reflect.Interface && intermediateVal.IsNil() {
		return true
	}
	return false
}

// createIntermediateMapValue creates an intermediate map value for chained
// access.
//
// Takes baseMapVal (reflect.Value) which is the parent map to modify.
// Takes innerKey (reflect.Value) which is the key for the new entry.
// Takes elemType (reflect.Type) which specifies the type of element to create.
//
// Returns reflect.Value which is the newly created map element.
func createIntermediateMapValue(baseMapVal, innerKey reflect.Value, elemType reflect.Type) reflect.Value {
	if elemType.Kind() == reflect.Interface {
		newMap := reflect.MakeMap(reflect.TypeFor[map[string]any]())
		baseMapVal.SetMapIndex(innerKey, newMap)
	} else {
		newElem := createMapElement(elemType)
		baseMapVal.SetMapIndex(innerKey, newElem)
	}
	return baseMapVal.MapIndex(innerKey)
}

// convertOuterKey converts an AST expression to a map key value.
//
// Takes outerKey (ast_domain.Expression) which is the AST expression to
// convert.
// Takes keyType (reflect.Type) which specifies the target map key type.
// Takes fullPath (string) which provides the path for error messages.
//
// Returns reflect.Value which is the converted key ready for map access.
// Returns error when the expression type is unsupported or conversion fails.
func convertOuterKey(outerKey ast_domain.Expression, keyType reflect.Type, fullPath string) (reflect.Value, error) {
	var keyVal reflect.Value
	var err error

	switch k := outerKey.(type) {
	case *ast_domain.StringLiteral:
		keyVal, err = convertMapKey(k.Value, keyType)
	case *ast_domain.IntegerLiteral:
		keyVal, err = convertMapKey(k.String(), keyType)
	default:
		return reflect.Value{}, errInvalidPath{path: fullPath, err: fmt.Errorf("unsupported outer key type: %T", outerKey)}
	}

	if err != nil {
		return reflect.Value{}, errInvalidPath{path: fullPath, err: fmt.Errorf("could not convert outer key: %w", err)}
	}
	return keyVal, nil
}

// newFieldInfoForType creates a fieldInfo struct for a given type with
// unmarshaler detection.
//
// This function is used to reduce repeated code in handleSliceIndexExpr and
// handleMapIndexExpr. The Offset, Kind, and CanDirect fields are set to zero
// or false because these are for dynamic paths, not fixed struct fields that
// could use unsafe direct access.
//
// Takes path (string) which specifies the field path for error messages.
// Takes t (reflect.Type) which specifies the type to create field info for.
//
// Returns *fieldInfo which contains type metadata for binding operations.
func newFieldInfoForType(path string, t reflect.Type) *fieldInfo {
	effectiveType := t
	if effectiveType.Kind() == reflect.Pointer {
		effectiveType = effectiveType.Elem()
	}
	unmarshaler, _ := implementsTextUnmarshaler(effectiveType)
	return &fieldInfo{
		Type:        t,
		unmarshaler: unmarshaler,
		Path:        path,
		Index:       nil,
		Offset:      0,
		Kind:        t.Kind(),
		CanDirect:   false,
	}
}

// validateChainedIndexing checks for unsupported chained indexing patterns.
//
// Takes mapVal (reflect.Value) which is the map value being checked.
// Takes elemType (reflect.Type) which is the element type of the map.
// Takes fullPath (string) which is the path used for error messages.
//
// Returns error when chained indexing into maps or slices is found, such as
// map[key1][key2] or map[key][index] patterns.
func validateChainedIndexing(mapVal reflect.Value, elemType reflect.Type, fullPath string) error {
	effectiveElemType := elemType
	if effectiveElemType.Kind() == reflect.Pointer {
		effectiveElemType = effectiveElemType.Elem()
	}

	isCollection := effectiveElemType.Kind() == reflect.Map || effectiveElemType.Kind() == reflect.Slice
	if isCollection && !mapVal.CanSet() {
		return errInvalidPath{
			path: fullPath,
			err:  errors.New("chained indexing into maps or slices is not supported (pattern like map[key1][key2] or map[key][index])"),
		}
	}
	return nil
}

// initialiseMapIfNil creates a new map if the given map value is nil.
//
// Takes mapVal (reflect.Value) which is the map value to check and set.
// Takes mapType (reflect.Type) which is the type to use when making the map.
// Takes fullPath (string) which is the path shown in error messages.
//
// Returns error when the map is nil but cannot be set. This happens when the
// map is accessed through another map, as chained map access is not supported.
func initialiseMapIfNil(mapVal reflect.Value, mapType reflect.Type, fullPath string) error {
	if !mapVal.IsNil() {
		return nil
	}
	if !mapVal.CanSet() {
		return errInvalidPath{
			path: fullPath,
			err:  errors.New("cannot initialise map obtained from another map value (chained map access is not supported)"),
		}
	}
	mapVal.Set(reflect.MakeMap(mapType))
	return nil
}

// setMapIndexValue sets a value in a map and handles pointer types correctly.
//
// Takes mapVal (reflect.Value) which is the map to change.
// Takes mapKey (reflect.Value) which is the key where the value will be set.
// Takes convertedVal (reflect.Value) which is the value to store.
// Takes elemType (reflect.Type) which is the expected element type of the map.
func setMapIndexValue(mapVal reflect.Value, mapKey, convertedVal reflect.Value, elemType reflect.Type) {
	if elemType.Kind() == reflect.Pointer {
		ptr := reflect.New(elemType.Elem())
		ptr.Elem().Set(convertedVal)
		mapVal.SetMapIndex(mapKey, ptr)
		return
	}
	if !convertedVal.Type().AssignableTo(elemType) {
		convertedVal = convertedVal.Convert(elemType)
	}
	mapVal.SetMapIndex(mapKey, convertedVal)
}
