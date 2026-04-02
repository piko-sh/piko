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

package generator_helpers

import (
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/maths"
)

// baseDecimal is the base value for decimal number formatting.
const baseDecimal = 10

// RequestDataFieldMap maps snake_case field names used in templates to their
// corresponding Go method names on RequestData, so that template expressions
// like `r.path_params` are converted to `r.PathParams()` in generated code.
var RequestDataFieldMap = map[string]string{
	"context":      "Context",
	"method":       "Method",
	"host":         "Host",
	"path_params":  "PathParams",
	"query_params": "QueryParams",
	"form_data":    "FormData",
}

// GetContentAST extracts the content AST from collection data.
//
// This function is called at runtime by generated code when rendering
// <piko:content /> tags in collection page templates. The content AST
// contains the parsed markdown body that should be rendered as children
// of the content tag's parent element.
//
// When collectionData is nil or not a map[string]any, returns nil.
//
// Takes collectionData (any) which is the collection data from request data,
// expected to be a map[string]any containing a "contentAST" key.
//
// Returns *ast_domain.TemplateAST which is the content AST if found and
// valid, or nil otherwise.
//
// Architecture note: CollectionData is populated by BuildAST via
// GetStaticCollectionItem. The AST is deserialised on-demand from embedded
// FlatBuffer bytes with zero-copy access to the binary data.
func GetContentAST(collectionData any) *ast_domain.TemplateAST {
	if collectionData == nil {
		return nil
	}

	dataMap, ok := collectionData.(map[string]any)
	if !ok {
		return nil
	}

	contentASTRaw, exists := dataMap["contentAST"]
	if !exists {
		return nil
	}

	contentAST, ok := contentASTRaw.(*ast_domain.TemplateAST)
	if !ok || contentAST == nil {
		return nil
	}

	return contentAST
}

// ValueToString converts a value to its string form.
//
// It handles common Go types such as strings, numbers, and booleans in a fast
// way. For other types, it uses the Stringer interface if available, or falls
// back to fmt.Sprintf.
//
// Takes value (any) which is the value to convert.
//
// Returns string which is the string form of the value, or an empty string if
// value is nil.
func ValueToString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8, int16, int32, int64:
		return signedIntToString(value)
	case uint:
		return strconv.FormatUint(uint64(v), baseDecimal)
	case uint8, uint16, uint32, uint64:
		return unsignedIntToString(value)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case maths.Decimal, *maths.Decimal, maths.BigInt, *maths.BigInt, maths.Money, *maths.Money:
		return mathsTypeToString(value)
	}

	if stringer, ok := value.(fmt.Stringer); ok {
		return stringer.String()
	}

	return fmt.Sprintf("%v", value)
}

// PointerValueToString converts a pointer value to its string representation.
//
// It safely dereferences common pointer types and returns an empty string for
// nil. This is used by the template engine for rendering optional/nullable
// fields.
//
// Takes value (any) which is the pointer value to convert.
//
// Returns string which is the dereferenced value as a string, or an empty
// string if value is nil.
func PointerValueToString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case *string:
		if v == nil {
			return ""
		}
		return *v
	case *int:
		if v == nil {
			return ""
		}
		return strconv.Itoa(*v)
	case *int64:
		if v == nil {
			return ""
		}
		return strconv.FormatInt(*v, 10)
	case *bool:
		if v == nil {
			return ""
		}
		return strconv.FormatBool(*v)
	case *float64:
		if v == nil {
			return ""
		}
		return strconv.FormatFloat(*v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// CheckExpressionForIdentifierUsage recursively checks if an expression uses
// the given identifier name, either directly or as a prefix (e.g., "item.name"
// would match identifier "item"). This is used to determine variable
// dependencies.
//
// Takes expression (ast_domain.Expression) which is the expression to search.
// Takes identifierName (string) which is the identifier to look for.
//
// Returns bool which is true if the expression uses the identifier.
func CheckExpressionForIdentifierUsage(expression ast_domain.Expression, identifierName string) bool {
	if expression == nil {
		return false
	}
	switch e := expression.(type) {
	case *ast_domain.Identifier:
		return strings.HasPrefix(e.Name, identifierName+".") || e.Name == identifierName
	case *ast_domain.StringLiteral,
		*ast_domain.IntegerLiteral,
		*ast_domain.FloatLiteral,
		*ast_domain.BooleanLiteral,
		*ast_domain.NilLiteral:
		return false
	case *ast_domain.UnaryExpression:
		return CheckExpressionForIdentifierUsage(e.Right, identifierName)
	case *ast_domain.BinaryExpression:
		return CheckExpressionForIdentifierUsage(e.Left, identifierName) || CheckExpressionForIdentifierUsage(e.Right, identifierName)
	case *ast_domain.CallExpression:
		if CheckExpressionForIdentifierUsage(e.Callee, identifierName) {
			return true
		}
		for _, argument := range e.Args {
			if CheckExpressionForIdentifierUsage(argument, identifierName) {
				return true
			}
		}
		return false
	case *ast_domain.ForInExpression:
		return CheckExpressionForIdentifierUsage(e.Collection, identifierName)
	}
	return false
}

// IsLoopVarUsedInNode checks if a loop variable is used anywhere within a
// template node or its children. This is used to optimise code generation by
// avoiding unnecessary variable bindings when the loop variable is not
// referenced.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to search.
// Takes loopVarName (string) which is the name of the loop variable to find.
//
// Returns bool which is true if the loop variable is referenced in the node
// or any of its descendants.
func IsLoopVarUsedInNode(node *ast_domain.TemplateNode, loopVarName string) bool {
	if node == nil || loopVarName == "" {
		return false
	}

	if isLoopVarUsedInBuiltInDirectives(node, loopVarName) {
		return true
	}
	if isLoopVarUsedInCollections(node, loopVarName) {
		return true
	}

	for _, child := range node.Children {
		if IsLoopVarUsedInNode(child, loopVarName) {
			return true
		}
	}
	return false
}

// signedIntToString converts a signed integer value to its string form.
//
// Takes value (any) which is an int8, int16, int32, or int64 value.
//
// Returns string which is the decimal string representation.
func signedIntToString(value any) string {
	switch v := value.(type) {
	case int8:
		return strconv.FormatInt(int64(v), baseDecimal)
	case int16:
		return strconv.FormatInt(int64(v), baseDecimal)
	case int32:
		return strconv.FormatInt(int64(v), baseDecimal)
	case int64:
		return strconv.FormatInt(v, baseDecimal)
	default:
		return ""
	}
}

// unsignedIntToString converts an unsigned integer value to its string form.
//
// Takes value (any) which is a uint8, uint16, uint32, or uint64 value.
//
// Returns string which is the decimal string representation.
func unsignedIntToString(value any) string {
	switch v := value.(type) {
	case uint8:
		return strconv.FormatUint(uint64(v), baseDecimal)
	case uint16:
		return strconv.FormatUint(uint64(v), baseDecimal)
	case uint32:
		return strconv.FormatUint(uint64(v), baseDecimal)
	case uint64:
		return strconv.FormatUint(v, baseDecimal)
	default:
		return ""
	}
}

// mathsTypeToString converts a maths.Decimal, maths.BigInt, or maths.Money
// value (or pointer) to its string form.
//
// Takes value (any) which is a maths type value or pointer.
//
// Returns string which is the formatted string, or empty for nil pointers.
func mathsTypeToString(value any) string {
	switch v := value.(type) {
	case maths.Decimal:
		return v.MustString()
	case *maths.Decimal:
		if v == nil {
			return ""
		}
		return v.MustString()
	case maths.BigInt:
		return v.MustString()
	case *maths.BigInt:
		if v == nil {
			return ""
		}
		return v.MustString()
	case maths.Money:
		return v.MustString()
	case *maths.Money:
		if v == nil {
			return ""
		}
		return v.MustString()
	default:
		return ""
	}
}

// isLoopVarUsedInBuiltInDirectives checks if a loop variable is used in any
// built-in directive on the node, such as p-if, p-for, or p-text.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes loopVarName (string) which is the loop variable name to search for.
//
// Returns bool which is true if the loop variable appears in any directive.
func isLoopVarUsedInBuiltInDirectives(node *ast_domain.TemplateNode, loopVarName string) bool {
	expressions := collectDirectiveExpressions(node)
	for _, expression := range expressions {
		if CheckExpressionForIdentifierUsage(expression, loopVarName) {
			return true
		}
	}
	return false
}

// collectDirectiveExpressions gathers all expressions from a node's built-in
// directives.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to get
// expressions from.
//
// Returns []ast_domain.Expression which holds all directive expressions found
// on the node, including nil values for unset directives.
func collectDirectiveExpressions(node *ast_domain.TemplateNode) []ast_domain.Expression {
	var expressions []ast_domain.Expression

	if node.DirIf != nil {
		expressions = append(expressions, node.DirIf.Expression)
	}

	if node.DirElseIf != nil {
		expressions = append(expressions, node.DirElseIf.Expression)
	}
	if node.DirFor != nil {
		expressions = append(expressions, node.DirFor.Expression)
	}
	if node.DirShow != nil {
		expressions = append(expressions, node.DirShow.Expression)
	}
	if node.DirModel != nil {
		expressions = append(expressions, node.DirModel.Expression)
	}
	if node.DirClass != nil {
		expressions = append(expressions, node.DirClass.Expression)
	}
	if node.DirStyle != nil {
		expressions = append(expressions, node.DirStyle.Expression)
	}
	if node.DirText != nil {
		expressions = append(expressions, node.DirText.Expression)
	}
	if node.DirHTML != nil {
		expressions = append(expressions, node.DirHTML.Expression)
	}
	return expressions
}

// isLoopVarUsedInCollections checks if the loop variable is used in any of the
// node's collection-based fields (DynamicAttributes, Binds, OnEvents, etc.).
//
// Takes node (*ast_domain.TemplateNode) which is the template node to inspect.
// Takes loopVarName (string) which is the name of the loop variable to find.
//
// Returns bool which is true if the loop variable is found in any collection.
func isLoopVarUsedInCollections(node *ast_domain.TemplateNode, loopVarName string) bool {
	if checkDynamicAttributes(node.DynamicAttributes, loopVarName) {
		return true
	}
	if checkBindsMap(node.Binds, loopVarName) {
		return true
	}
	if checkEventMap(node.OnEvents, loopVarName) {
		return true
	}
	if checkEventMap(node.CustomEvents, loopVarName) {
		return true
	}
	return checkDirectives(node.Directives, loopVarName)
}

// checkDynamicAttributes checks whether the loop variable is used in any
// dynamic attribute.
//
// Takes attrs ([]ast_domain.DynamicAttribute) which is the list of dynamic
// attributes to search.
// Takes loopVarName (string) which is the loop variable name to find.
//
// Returns bool which is true if the loop variable is found in any attribute
// expression.
func checkDynamicAttributes(attrs []ast_domain.DynamicAttribute, loopVarName string) bool {
	for i := range attrs {
		if CheckExpressionForIdentifierUsage(attrs[i].Expression, loopVarName) {
			return true
		}
	}
	return false
}

// checkBindsMap checks whether a loop variable is used in any bind directive
// within a map of directives.
//
// Takes binds (map[string]*ast_domain.Directive) which contains the bind
// directives to search.
// Takes loopVarName (string) which is the name of the loop variable to find.
//
// Returns bool which is true if the loop variable is used in any directive.
func checkBindsMap(binds map[string]*ast_domain.Directive, loopVarName string) bool {
	for _, directive := range binds {
		if directive != nil && CheckExpressionForIdentifierUsage(directive.Expression, loopVarName) {
			return true
		}
	}
	return false
}

// checkEventMap checks whether the loop variable is used in an event map.
//
// Takes eventMap (map[string][]ast_domain.Directive) which holds the event
// directives to search.
// Takes loopVarName (string) which is the name of the loop variable to find.
//
// Returns bool which is true if the loop variable is used in any directive
// expression.
func checkEventMap(eventMap map[string][]ast_domain.Directive, loopVarName string) bool {
	for _, directiveSlice := range eventMap {
		for i := range directiveSlice {
			if CheckExpressionForIdentifierUsage(directiveSlice[i].Expression, loopVarName) {
				return true
			}
		}
	}
	return false
}

// checkDirectives checks if the loop variable is used in generic directives.
//
// Takes directives ([]ast_domain.Directive) which contains the directives to
// search through.
// Takes loopVarName (string) which specifies the loop variable name to find.
//
// Returns bool which is true if the loop variable is used in any directive
// expression.
func checkDirectives(directives []ast_domain.Directive, loopVarName string) bool {
	for i := range directives {
		if CheckExpressionForIdentifierUsage(directives[i].Expression, loopVarName) {
			return true
		}
	}
	return false
}
