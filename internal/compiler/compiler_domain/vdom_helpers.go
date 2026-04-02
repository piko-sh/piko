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
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// SscanfFloat parses a string as a float64.
//
// Takes s (string) which contains the text to parse.
//
// Returns float64 which is the parsed value.
// Returns error when the string cannot be parsed as a float.
func SscanfFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// buildDOMCall creates a call expression of the form dom.method(...arguments).
//
// Takes methodName (string) which names the method to call on the dom object.
// Takes arguments (...js_ast.Expr) which are the arguments to pass to the method.
//
// Returns js_ast.Expr which is the complete method call expression.
func buildDOMCall(methodName string, arguments ...js_ast.Expr) js_ast.Expr {
	return buildMethodCallOnExpr(newIdentifier("dom"), methodName, arguments...)
}

// buildMethodCallOnExpr creates a method call expression in the form
// target.method(...arguments).
//
// Takes target (js_ast.Expr) which is the object to call the method on.
// Takes methodName (string) which is the name of the method to call.
// Takes arguments (...js_ast.Expr) which are the arguments to pass to the method.
//
// Returns js_ast.Expr which is the complete method call expression.
func buildMethodCallOnExpr(target js_ast.Expr, methodName string, arguments ...js_ast.Expr) js_ast.Expr {
	return js_ast.Expr{Data: &js_ast.ECall{
		Target: js_ast.Expr{Data: &js_ast.EDot{
			Target: target,
			Name:   methodName,
		}},
		Args: arguments,
	}}
}

// newStringLiteral creates a JavaScript AST expression for a string value.
//
// Takes s (string) which is the value to convert to a UTF-16 string literal.
//
// Returns js_ast.Expr which wraps the string as an EString expression.
func newStringLiteral(s string) js_ast.Expr {
	return js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16(s)}}
}

// newIdentifier creates a JavaScript identifier expression and registers its
// name.
//
// Takes name (string) which specifies the identifier name to use.
//
// Returns js_ast.Expr which contains the registered identifier.
func newIdentifier(name string) js_ast.Expr {
	identifier := &js_ast.EIdentifier{}
	registerIdentifierName(identifier, name)
	return js_ast.Expr{Data: identifier}
}

// newBooleanLiteral creates a JavaScript boolean literal expression.
//
// Takes b (bool) which is the boolean value to wrap.
//
// Returns js_ast.Expr which holds the boolean literal.
func newBooleanLiteral(b bool) js_ast.Expr {
	return js_ast.Expr{Data: &js_ast.EBoolean{Value: b}}
}

// newNullLiteral creates a JavaScript null literal expression.
//
// Returns js_ast.Expr which holds the null value for use in AST building.
func newNullLiteral() js_ast.Expr {
	return js_ast.Expr{Data: js_ast.ENullShared}
}

// parseFloat64 parses a string to float64 for use in JavaScript number
// literals. Returns 0 if the string cannot be parsed.
//
// Takes s (string) which is the decimal string to parse.
//
// Returns float64 which is the parsed number value.
func parseFloat64(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64) //nolint:revive // parser-validated
	return f
}

// isNull checks whether the given expression is a null literal.
//
// Takes expression (js_ast.Expr) which is the expression to
// check.
//
// Returns bool which is true if the expression is a null literal.
func isNull(expression js_ast.Expr) bool {
	_, ok := expression.Data.(*js_ast.ENull)
	return ok
}

// parseSnippetAsExpr parses a JavaScript expression snippet.
//
// Takes snippet (string) which contains the JavaScript expression to parse.
//
// Returns js_ast.Expr which is the parsed expression with identifiers
// registered for later use.
// Returns error when the snippet fails to parse or is not a valid expression.
func parseSnippetAsExpr(snippet string) (js_ast.Expr, error) {
	wrappedSnippet := "(" + snippet + ")"

	parser := NewTypeScriptParser()
	parsedAST, err := parser.ParseTypeScript(wrappedSnippet, "snippet.ts")
	if err != nil {
		return js_ast.Expr{}, fmt.Errorf("failed to parse snippet '%s': %w", snippet, err)
	}
	statements := getStmtsFromAST(parsedAST)
	if parsedAST == nil || len(statements) == 0 {
		return js_ast.Expr{}, fmt.Errorf("parser returned empty AST for snippet '%s'", snippet)
	}

	expressionStatement, ok := statements[0].Data.(*js_ast.SExpr)
	if !ok {
		return js_ast.Expr{}, fmt.Errorf("snippet did not parse as an expression statement: '%s'", snippet)
	}

	registerExprIdentifiers(expressionStatement.Value, parsedAST.Symbols)

	return expressionStatement.Value, nil
}

// appendToKeyExpr adds a suffix to a key expression.
//
// For simple string keys, it joins the strings directly. For complex
// expressions (such as those with loop variables), it uses binary addition
// to keep the original expression intact.
//
// Takes keyExpr (js_ast.Expr) which is the key expression to extend.
// Takes suffix (string) which is the string to add to the key.
//
// Returns js_ast.Expr which is the expression with the suffix added.
func appendToKeyExpr(keyExpr js_ast.Expr, suffix string) js_ast.Expr {
	if keyExpr.Data == nil {
		return newStringLiteral(suffix)
	}

	if s, ok := keyExpr.Data.(*js_ast.EString); ok {
		return newStringLiteral(helpers.UTF16ToString(s.Value) + suffix)
	}

	return js_ast.Expr{Data: &js_ast.EBinary{
		Op:    js_ast.BinOpAdd,
		Left:  keyExpr,
		Right: newStringLiteral(suffix),
	}}
}

// expressionToJSValueString converts a JS expression to its
// string form.
//
// Takes expression (js_ast.Expr) which is the expression to
// convert.
//
// Returns string which is the value for strings, numbers, and
// booleans, or a type name like "null", "identifier", or "expr"
// for other expression types.
func expressionToJSValueString(expression js_ast.Expr) string {
	if expression.Data == nil {
		return "null"
	}
	switch e := expression.Data.(type) {
	case *js_ast.EString:
		return helpers.UTF16ToString(e.Value)
	case *js_ast.ENumber:
		return strconv.FormatFloat(e.Value, 'f', -1, 64)
	case *js_ast.EBoolean:
		if e.Value {
			return "true"
		}
		return "false"
	case *js_ast.ENull:
		return "null"
	case *js_ast.EIdentifier:
		return "identifier"
	default:
		return "expr"
	}
}

// cloneNode creates a deep copy of a template node.
//
// Takes original (*ast_domain.TemplateNode) which is the node to copy.
//
// Returns *ast_domain.TemplateNode which is a new node with all attributes,
// directives, events, and children copied recursively.
func cloneNode(original *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	copiedNode := *original
	if original.Attributes != nil {
		copiedNode.Attributes = make([]ast_domain.HTMLAttribute, len(original.Attributes))
		copy(copiedNode.Attributes, original.Attributes)
	}
	if original.DynamicAttributes != nil {
		copiedNode.DynamicAttributes = make([]ast_domain.DynamicAttribute, len(original.DynamicAttributes))
		copy(copiedNode.DynamicAttributes, original.DynamicAttributes)
	}
	if original.Directives != nil {
		copiedNode.Directives = make([]ast_domain.Directive, len(original.Directives))
		copy(copiedNode.Directives, original.Directives)
	}
	if original.OnEvents != nil {
		copiedNode.OnEvents = make(map[string][]ast_domain.Directive, len(original.OnEvents))
		for k, v := range original.OnEvents {
			sliceCopy := make([]ast_domain.Directive, len(v))
			copy(sliceCopy, v)
			copiedNode.OnEvents[k] = sliceCopy
		}
	}
	if original.CustomEvents != nil {
		copiedNode.CustomEvents = make(map[string][]ast_domain.Directive, len(original.CustomEvents))
		for k, v := range original.CustomEvents {
			sliceCopy := make([]ast_domain.Directive, len(v))
			copy(sliceCopy, v)
			copiedNode.CustomEvents[k] = sliceCopy
		}
	}
	if original.Children != nil {
		copiedNode.Children = make([]*ast_domain.TemplateNode, len(original.Children))
		for i, child := range original.Children {
			copiedNode.Children[i] = cloneNode(child)
		}
	}
	return &copiedNode
}

// filterOutKeyAttrs removes :key dynamic attributes from the given slice.
//
// Takes dyn ([]ast_domain.DynamicAttribute) which contains the attributes to
// filter.
//
// Returns []ast_domain.DynamicAttribute which contains all attributes except
// those named "key" (case-insensitive comparison).
func filterOutKeyAttrs(dyn []ast_domain.DynamicAttribute) []ast_domain.DynamicAttribute {
	var out []ast_domain.DynamicAttribute
	for dynamicAttributeIndex := range dyn {
		if !strings.EqualFold(dyn[dynamicAttributeIndex].Name, "key") {
			out = append(out, dyn[dynamicAttributeIndex])
		}
	}
	return out
}

// filterOutKeyAttrsHTML removes any HTML attribute named "key" from the list.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// filter.
//
// Returns []ast_domain.HTMLAttribute which contains all attributes except
// those named "key" (case-insensitive match).
func filterOutKeyAttrsHTML(attrs []ast_domain.HTMLAttribute) []ast_domain.HTMLAttribute {
	var out []ast_domain.HTMLAttribute
	for atIndex := range attrs {
		if !strings.EqualFold(attrs[atIndex].Name, "key") {
			out = append(out, attrs[atIndex])
		}
	}
	return out
}

// squashWhitespace replaces all whitespace sequences with single spaces.
//
// Takes s (string) which is the input text to process.
//
// Returns string which contains the input with all runs of whitespace
// (including newlines, carriage returns, and tabs) replaced by single spaces.
func squashWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\r", strSpace)
	s = strings.ReplaceAll(s, "\n", strSpace)
	s = strings.ReplaceAll(s, "\t", strSpace)
	return whitespacePattern.ReplaceAllString(s, strSpace)
}

// escapeString escapes special characters in a string for use in JavaScript.
//
// Takes raw (string) which is the input string to escape.
//
// Returns string which is the escaped string safe for use in JavaScript.
func escapeString(raw string) string {
	raw = strings.ReplaceAll(raw, `\`, `\\`)
	raw = strings.ReplaceAll(raw, `"`, `\"`)
	raw = strings.ReplaceAll(raw, "\n", `\n`)
	raw = strings.ReplaceAll(raw, "\r", `\r`)
	raw = strings.ReplaceAll(raw, "\t", `\t`)
	return raw
}

// isNumericOrBoolOrNull checks whether a string value represents a number,
// boolean, null, or undefined.
//
// Takes value (string) which is the value to check.
//
// Returns bool which is true if the value is a number, boolean, null, or
// undefined.
func isNumericOrBoolOrNull(value string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "true" || trimmed == "false" || trimmed == "null" || trimmed == "undefined" {
		return true
	}
	_, err := SscanfFloat(trimmed)
	return err == nil
}
