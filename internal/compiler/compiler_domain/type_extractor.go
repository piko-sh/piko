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

package compiler_domain

import (
	"errors"
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// TypeExtractor defines the interface for extracting component metadata
// from an AST. It implements compiler_domain.TypeExtractor.
type TypeExtractor interface {
	// ExtractMetadata retrieves metadata from the component.
	//
	// Returns *ComponentMetadata which contains the extracted metadata.
	// Returns error when metadata extraction fails.
	ExtractMetadata() (*ComponentMetadata, error)
}

// typeExtractor walks an esbuild AST to extract component metadata.
// It implements TypeExtractor and handles TypeScript type assertions.
type typeExtractor struct {
	// ast holds the parsed JavaScript AST used to extract type metadata.
	ast *js_ast.AST

	// typeAssertions maps property names to their type assertions.
	typeAssertions map[string]TypeAssertion
}

var _ TypeExtractor = (*typeExtractor)(nil)

// ExtractMetadata walks the esbuild AST and extracts type information from
// literal value inference, user-defined functions, and state bindings.
//
// Returns *ComponentMetadata which contains the extracted type information.
// Returns error when extracting state properties fails.
func (e *typeExtractor) ExtractMetadata() (*ComponentMetadata, error) {
	metadata := NewComponentMetadata()

	for _, statement := range getStmtsFromAST(e.ast) {
		if local, ok := statement.Data.(*js_ast.SLocal); ok {
			for i, declaration := range local.Decls {
				if e.isStateBinding(declaration.Binding) {
					if err := e.extractStateProperties(&local.Decls[i], metadata); err != nil {
						return nil, fmt.Errorf("extracting state properties: %w", err)
					}
				}
			}
		}
	}

	e.extractMethods(metadata)

	return metadata, nil
}

// isStateBinding checks if a binding represents the "state" variable.
//
// Takes binding (js_ast.Binding) which is the AST binding to check.
//
// Returns bool which is true if the binding is an identifier with the original
// name "state".
func (e *typeExtractor) isStateBinding(binding js_ast.Binding) bool {
	if identifier, ok := binding.Data.(*js_ast.BIdentifier); ok {
		if int(identifier.Ref.InnerIndex) < len(e.ast.Symbols) {
			return e.ast.Symbols[identifier.Ref.InnerIndex].OriginalName == "state"
		}
	}
	return false
}

// extractStateProperties reads property data from a state variable and adds
// it to the component metadata.
//
// Takes declaration (*js_ast.Decl) which is the state variable to parse.
// Takes metadata (*ComponentMetadata) which receives the extracted properties.
//
// Returns error when the declaration has no value or is not an object literal.
func (e *typeExtractor) extractStateProperties(declaration *js_ast.Decl, metadata *ComponentMetadata) error {
	if declaration.ValueOrNil.Data == nil {
		return errors.New("state declaration has no value")
	}

	valueExpr := declaration.ValueOrNil

	jsObject, ok := valueExpr.Data.(*js_ast.EObject)
	if !ok {
		return errors.New("state value is not an object literal")
	}

	for i := range jsObject.Properties {
		prop := &jsObject.Properties[i]
		propName := e.getPropertyName(prop.Key)

		if propName == "" {
			continue
		}

		propMeta := e.extractPropertyMetadata(prop.ValueOrNil, propName)
		if propMeta != nil {
			metadata.StateProperties[propName] = propMeta

			if propMeta.IsBoolean() {
				metadata.BooleanProps = append(metadata.BooleanProps, propName)
			}
		}
	}

	return nil
}

// extractPropertyMetadata extracts type information from a property value.
// Uses pre-extracted type assertions when available, otherwise infers from the
// literal value.
//
// Takes expression (js_ast.Expr) which is the expression to
// extract type info from.
// Takes propName (string) which is the name of the property being
// processed.
//
// Returns *PropertyMetadata which contains the extracted type
// information, or a default metadata with JSType "any" if no type
// could be determined.
func (e *typeExtractor) extractPropertyMetadata(expression js_ast.Expr, propName string) *PropertyMetadata {
	meta := &PropertyMetadata{
		Name:         propName,
		JSType:       "",
		ElementType:  "",
		KeyType:      "",
		ValueType:    "",
		InitialValue: "",
		Location:     ast_domain.Location{},
		IsNullable:   false,
	}

	if assertion, found := e.typeAssertions[propName]; found {
		parsedType := ParseTypeString(assertion.TypeString)
		meta.JSType = parsedType.JSType
		meta.ElementType = parsedType.ElementType
		meta.KeyType = parsedType.KeyType
		meta.ValueType = parsedType.ValueType
		meta.IsNullable = parsedType.IsNullable
		meta.InitialValue = e.expressionToString(expression)
		return meta
	}

	inferredMeta := e.inferTypeFromLiteral(expression)
	if inferredMeta != nil {
		inferredMeta.Name = propName
		inferredMeta.InitialValue = e.expressionToString(expression)
		return inferredMeta
	}

	meta.JSType = "any"
	meta.InitialValue = e.expressionToString(expression)
	return meta
}

// inferTypeFromLiteral infers the TypeScript type from a JavaScript literal
// expression.
//
// Takes expression (js_ast.Expr) which is the literal expression
// to analyse.
//
// Returns *PropertyMetadata which contains the inferred type
// information, or nil when the expression type is not recognised.
func (e *typeExtractor) inferTypeFromLiteral(expression js_ast.Expr) *PropertyMetadata {
	switch v := expression.Data.(type) {
	case *js_ast.ENumber:
		return newPropertyMeta("number", "", "", "")
	case *js_ast.EString:
		return newPropertyMeta("string", "", "", "")
	case *js_ast.EBoolean:
		return newPropertyMeta("boolean", "", "", "")
	case *js_ast.EArray:
		return e.inferArrayType(v)
	case *js_ast.EObject:
		return newPropertyMeta(typeObject, "", "", "")
	case *js_ast.ENull, *js_ast.EUndefined:
		return newPropertyMeta("any", "", "", "")
	default:
		return nil
	}
}

// inferArrayType works out the element type from an array literal.
//
// Takes arr (*js_ast.EArray) which is the array expression to analyse.
//
// Returns *PropertyMetadata which contains the inferred array type metadata.
func (e *typeExtractor) inferArrayType(arr *js_ast.EArray) *PropertyMetadata {
	if len(arr.Items) > 0 {
		if elemMeta := e.inferTypeFromLiteral(arr.Items[0]); elemMeta != nil {
			return newPropertyMeta(typeArray, elemMeta.JSType, "", "")
		}
	}
	return newPropertyMeta(typeArray, "", "", "")
}

// getPropertyName extracts the property name from a key expression.
//
// Takes key (js_ast.Expr) which is the expression to extract the name from.
//
// Returns string which is the property name, or empty if extraction fails.
func (e *typeExtractor) getPropertyName(key js_ast.Expr) string {
	switch k := key.Data.(type) {
	case *js_ast.EString:
		return helpers.UTF16ToString(k.Value)
	case *js_ast.EIdentifier:
		if int(k.Ref.InnerIndex) < len(e.ast.Symbols) {
			return e.ast.Symbols[k.Ref.InnerIndex].OriginalName
		}
	}
	return ""
}

// extractMethods finds user-defined function declarations in the AST.
//
// Takes metadata (*ComponentMetadata) which stores the discovered methods.
func (e *typeExtractor) extractMethods(metadata *ComponentMetadata) {
	for _, statement := range getStmtsFromAST(e.ast) {
		if funcDecl, ok := statement.Data.(*js_ast.SFunction); ok {
			if funcDecl.Fn.Name != nil {
				methodName := ""
				if e.ast.Symbols != nil && int(funcDecl.Fn.Name.Ref.InnerIndex) < len(e.ast.Symbols) {
					methodName = e.ast.Symbols[funcDecl.Fn.Name.Ref.InnerIndex].OriginalName
				}
				if methodName != "" {
					metadata.Methods[methodName] = &MethodMetadata{
						Name:     methodName,
						Location: ast_domain.Location{},
					}
				}
			}
		}
	}
}

// expressionToString converts an esbuild expression to its string form.
// Used to keep the starting value for defaultProps.
//
// Takes expression (js_ast.Expr) which is the expression to
// convert.
//
// Returns string which is the string form of the expression.
func (e *typeExtractor) expressionToString(expression js_ast.Expr) string {
	switch v := expression.Data.(type) {
	case *js_ast.ENumber:
		return fmt.Sprintf("%v", v.Value)
	case *js_ast.EString:
		return fmt.Sprintf("%q", helpers.UTF16ToString(v.Value))
	case *js_ast.EBoolean:
		return e.booleanToString(v.Value)
	case *js_ast.EArray:
		return e.arrayToString(v)
	case *js_ast.EObject:
		return e.objectToString(v)
	case *js_ast.ENull:
		return "null"
	case *js_ast.EUndefined:
		return "undefined"
	default:
		return "null"
	}
}

// booleanToString converts a boolean value to its string representation.
//
// Takes value (bool) which is the boolean to convert.
//
// Returns string which is "true" or "false".
func (*typeExtractor) booleanToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// arrayToString converts an array expression to a string.
//
// Takes v (*js_ast.EArray) which is the array expression to convert.
//
// Returns string which is the formatted array as "[elem1, elem2, ...]" or
// "[]" for empty arrays.
func (e *typeExtractor) arrayToString(v *js_ast.EArray) string {
	if len(v.Items) == 0 {
		return "[]"
	}
	elements := make([]string, len(v.Items))
	for i, item := range v.Items {
		elements[i] = e.expressionToString(item)
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// objectToString converts an object expression to its string form.
//
// Takes v (*js_ast.EObject) which is the object expression to convert.
//
// Returns string which is the formatted object as text.
func (e *typeExtractor) objectToString(v *js_ast.EObject) string {
	if len(v.Properties) == 0 {
		return "{}"
	}
	props := make([]string, 0, len(v.Properties))
	for _, prop := range v.Properties {
		key := e.extractPropertyKey(prop.Key)
		if key == "" {
			continue
		}
		value := e.expressionToString(prop.ValueOrNil)
		props = append(props, fmt.Sprintf("%q: %s", key, value))
	}
	return "{" + strings.Join(props, ", ") + "}"
}

// extractPropertyKey gets the key name from a property key expression.
//
// Takes keyExpr (js_ast.Expr) which is the property key expression.
//
// Returns string which is the key name, or an empty string if the key is not
// a string literal.
func (*typeExtractor) extractPropertyKey(keyExpr js_ast.Expr) string {
	if strKey, ok := keyExpr.Data.(*js_ast.EString); ok {
		return helpers.UTF16ToString(strKey.Value)
	}
	return ""
}

// NewTypeExtractor creates a new type extractor for the given AST.
//
// The typeAssertions should be pre-extracted from raw source before esbuild
// parsing, since esbuild strips type information.
//
// Takes ast (*js_ast.AST) which is the parsed JavaScript AST to extract from.
// Takes typeAssertions (map[string]typeAssertion) which provides pre-extracted
// type assertion data.
//
// Returns TypeExtractor which is ready to extract type information.
func NewTypeExtractor(ast *js_ast.AST, typeAssertions map[string]TypeAssertion) TypeExtractor {
	return &typeExtractor{
		ast:            ast,
		typeAssertions: typeAssertions,
	}
}

// newPropertyMeta creates a PropertyMetadata with common defaults.
//
// Takes jsType (string) which specifies the JavaScript type name.
// Takes elemType (string) which specifies the element type for arrays.
// Takes keyType (string) which specifies the key type for maps.
// Takes valueType (string) which specifies the value type for maps.
//
// Returns *PropertyMetadata which contains the type info with default values.
func newPropertyMeta(jsType, elemType, keyType, valueType string) *PropertyMetadata {
	return &PropertyMetadata{
		Name:         "",
		JSType:       jsType,
		ElementType:  elemType,
		KeyType:      keyType,
		ValueType:    valueType,
		InitialValue: "",
		Location:     ast_domain.Location{},
		IsNullable:   false,
	}
}
