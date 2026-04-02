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
	"maps"
	"slices"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// binaryOpMap maps our custom binary operators to esbuild's OpCode.
var binaryOpMap = map[ast_domain.BinaryOp]js_ast.OpCode{
	ast_domain.OpAnd:      js_ast.BinOpLogicalAnd,
	ast_domain.OpOr:       js_ast.BinOpLogicalOr,
	ast_domain.OpEq:       js_ast.BinOpStrictEq,
	ast_domain.OpNe:       js_ast.BinOpStrictNe,
	ast_domain.OpLooseEq:  js_ast.BinOpLooseEq,
	ast_domain.OpLooseNe:  js_ast.BinOpLooseNe,
	ast_domain.OpGt:       js_ast.BinOpGt,
	ast_domain.OpLt:       js_ast.BinOpLt,
	ast_domain.OpGe:       js_ast.BinOpGe,
	ast_domain.OpLe:       js_ast.BinOpLe,
	ast_domain.OpMinus:    js_ast.BinOpSub,
	ast_domain.OpMul:      js_ast.BinOpMul,
	ast_domain.OpDiv:      js_ast.BinOpDiv,
	ast_domain.OpMod:      js_ast.BinOpRem,
	ast_domain.OpPlus:     js_ast.BinOpAdd,
	ast_domain.OpCoalesce: js_ast.BinOpNullishCoalescing,
}

// transformOurASTtoJSAST converts our AST expression to esbuild's JS AST.
// It dispatches to specialised handlers for each category of expression.
//
// Takes ourExpr (ast_domain.Expression) which is the expression to transform.
// Takes registry (*RegistryContext) which provides symbol resolution context.
//
// Returns js_ast.Expr which is the equivalent esbuild AST expression.
// Returns error when the expression type is unhandled or parsing fails.
func transformOurASTtoJSAST(ourExpr ast_domain.Expression, registry *RegistryContext) (js_ast.Expr, error) {
	if ourExpr == nil {
		return newNullLiteral(), nil
	}

	if expression, ok, err := tryTransformLiteral(ourExpr); ok {
		return expression, err
	}
	if expression, ok, err := tryTransformIdentifier(ourExpr, registry); ok {
		return expression, err
	}
	if expression, ok, err := tryTransformAccessor(ourExpr, registry); ok {
		return expression, err
	}
	if expression, ok, err := tryTransformOperation(ourExpr, registry); ok {
		return expression, err
	}
	if expression, ok, err := tryTransformCallExpr(ourExpr, registry); ok {
		return expression, err
	}
	if expression, ok, err := tryTransformComplexLiteral(ourExpr, registry); ok {
		return expression, err
	}

	if _, isForIn := ourExpr.(*ast_domain.ForInExpression); isForIn {
		return js_ast.Expr{}, errors.New("forInExpr should be handled by the VDOM builder's loop logic")
	}

	return parseSnippetAsExpr(ourExpr.String())
}

// tryTransformLiteral attempts to convert a literal expression to JavaScript.
//
// Takes expression (ast_domain.Expression) which is the expression to
// transform.
//
// Returns js_ast.Expr which is the transformed JavaScript AST expression.
// Returns bool which is true if the expression was a supported literal type.
// Returns error when transformation fails.
func tryTransformLiteral(expression ast_domain.Expression) (js_ast.Expr, bool, error) {
	switch n := expression.(type) {
	case *ast_domain.StringLiteral:
		return newStringLiteral(n.Value), true, nil
	case *ast_domain.IntegerLiteral:
		return js_ast.Expr{Data: &js_ast.ENumber{Value: float64(n.Value)}}, true, nil
	case *ast_domain.FloatLiteral:
		return js_ast.Expr{Data: &js_ast.ENumber{Value: n.Value}}, true, nil
	case *ast_domain.BooleanLiteral:
		return newBooleanLiteral(n.Value), true, nil
	case *ast_domain.NilLiteral:
		return newNullLiteral(), true, nil
	case *ast_domain.DecimalLiteral:
		return js_ast.Expr{Data: &js_ast.ENumber{Value: parseFloat64(n.Value)}}, true, nil
	case *ast_domain.BigIntLiteral:
		return newStringLiteral(n.Value), true, nil
	case *ast_domain.RuneLiteral:
		return newStringLiteral(string(n.Value)), true, nil
	case *ast_domain.DateLiteral:
		return newStringLiteral(n.Value), true, nil
	case *ast_domain.TimeLiteral:
		return newStringLiteral(n.Value), true, nil
	case *ast_domain.DateTimeLiteral:
		return newStringLiteral(n.Value), true, nil
	case *ast_domain.DurationLiteral:
		return newStringLiteral(n.Value), true, nil
	default:
		return js_ast.Expr{}, false, nil
	}
}

// tryTransformIdentifier converts an identifier expression to JavaScript AST
// form.
//
// Takes expression (ast_domain.Expression) which is the expression to
// transform.
// Takes registry (*RegistryContext) which provides identifier creation.
//
// Returns js_ast.Expr which is the transformed JavaScript expression.
// Returns bool which indicates whether the expression was an identifier.
// Returns error when transformation fails.
func tryTransformIdentifier(expression ast_domain.Expression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	n, ok := expression.(*ast_domain.Identifier)
	if !ok {
		return js_ast.Expr{}, false, nil
	}

	if shouldRewriteToContext(n.Name) {
		return buildContextAccessExpr(n.Name), true, nil
	}
	return registry.MakeIdentifierExpr(n.Name), true, nil
}

// shouldRewriteToContext checks if an identifier should be rewritten to
// this.$$ctx.<name>.
//
// Takes name (string) which is the identifier name to check.
//
// Returns bool which is true if the identifier should be rewritten.
func shouldRewriteToContext(name string) bool {
	return name == "state"
}

// buildContextAccessExpr creates a this.$$ctx.<name> expression.
//
// Takes name (string) which specifies the property to access on the context.
//
// Returns js_ast.Expr which is the built dot-access expression.
func buildContextAccessExpr(name string) js_ast.Expr {
	thisExpr := js_ast.Expr{Data: &js_ast.EThis{}}
	ctxAccess := js_ast.Expr{Data: &js_ast.EDot{
		Target: thisExpr,
		Name:   "$$ctx",
	}}
	return js_ast.Expr{Data: &js_ast.EDot{
		Target: ctxAccess,
		Name:   name,
	}}
}

// tryTransformAccessor handles member and index access expressions.
//
// Takes expression (ast_domain.Expression) which is the expression to
// transform.
// Takes registry (*RegistryContext) which provides the transformation
// context.
//
// Returns js_ast.Expr which is the transformed JavaScript expression.
// Returns bool which shows whether the expression was transformed.
// Returns error when the transformation fails.
func tryTransformAccessor(expression ast_domain.Expression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	switch n := expression.(type) {
	case *ast_domain.MemberExpression:
		return transformMemberExpr(n, registry)
	case *ast_domain.IndexExpression:
		return transformIndexExpr(n, registry)
	default:
		return js_ast.Expr{}, false, nil
	}
}

// transformMemberExpr changes a member expression AST node into its JavaScript
// form.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression to change.
// Takes registry (*RegistryContext) which provides the context for the change.
//
// Returns js_ast.Expr which is the JavaScript expression that was made.
// Returns bool which shows whether the change was tried.
// Returns error when the base expression fails to change or when the property
// is not an identifier.
func transformMemberExpr(n *ast_domain.MemberExpression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	base, err := transformOurASTtoJSAST(n.Base, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	prop, ok := n.Property.(*ast_domain.Identifier)
	if !ok {
		return js_ast.Expr{}, true, errors.New("unsupported non-identifier property in MemberExpr")
	}

	optionalChain := js_ast.OptionalChainNone
	if n.Optional {
		optionalChain = js_ast.OptionalChainStart
	}

	return js_ast.Expr{Data: &js_ast.EDot{
		Target:        base,
		Name:          prop.Name,
		OptionalChain: optionalChain,
	}}, true, nil
}

// transformIndexExpr converts an index expression from the AST domain to a
// JavaScript AST index expression.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression to convert.
// Takes registry (*RegistryContext) which provides the conversion context.
//
// Returns js_ast.Expr which is the resulting JavaScript index expression.
// Returns bool which shows whether the conversion was handled.
// Returns error when the base or index expression fails to convert.
func transformIndexExpr(n *ast_domain.IndexExpression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	base, err := transformOurASTtoJSAST(n.Base, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	index, err := transformOurASTtoJSAST(n.Index, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	return js_ast.Expr{Data: &js_ast.EIndex{
		Target:        base,
		Index:         index,
		OptionalChain: js_ast.OptionalChainNone,
	}}, true, nil
}

// tryTransformOperation handles unary, binary, and ternary operations.
//
// Takes expression (ast_domain.Expression) which is the expression to
// transform.
// Takes registry (*RegistryContext) which provides the transformation
// context.
//
// Returns js_ast.Expr which is the transformed JavaScript expression.
// Returns bool which shows whether the transformation was handled.
// Returns error when the transformation fails.
func tryTransformOperation(expression ast_domain.Expression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	switch n := expression.(type) {
	case *ast_domain.UnaryExpression:
		return transformUnaryExpr(n, registry)
	case *ast_domain.BinaryExpression:
		return transformBinaryExpr(n, registry)
	case *ast_domain.TernaryExpression:
		return transformTernaryExpr(n, registry)
	default:
		return js_ast.Expr{}, false, nil
	}
}

// transformUnaryExpr converts a unary expression from the AST to JavaScript.
//
// Takes n (*ast_domain.UnaryExpression) which is the unary expression to convert.
// Takes registry (*RegistryContext) which provides the conversion context.
//
// Returns js_ast.Expr which is the resulting JavaScript expression.
// Returns bool which indicates whether the conversion was handled.
// Returns error when the operand conversion fails.
func transformUnaryExpr(n *ast_domain.UnaryExpression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	right, err := transformOurASTtoJSAST(n.Right, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}

	if n.Operator == ast_domain.OpTruthy {
		return js_ast.Expr{Data: &js_ast.EUnary{
			Op: js_ast.UnOpNot,
			Value: js_ast.Expr{Data: &js_ast.EUnary{
				Op:    js_ast.UnOpNot,
				Value: right,
			}},
		}}, true, nil
	}

	return js_ast.Expr{Data: &js_ast.EUnary{
		Op:    toJSUnaryOpCode(n.Operator),
		Value: right,
	}}, true, nil
}

// transformBinaryExpr converts a binary expression to its JavaScript AST form.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to convert.
// Takes registry (*RegistryContext) which provides the transformation context.
//
// Returns js_ast.Expr which is the resulting JavaScript binary expression.
// Returns bool which shows whether the transformation was handled.
// Returns error when the left or right operand fails to transform.
func transformBinaryExpr(n *ast_domain.BinaryExpression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	left, err := transformOurASTtoJSAST(n.Left, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	right, err := transformOurASTtoJSAST(n.Right, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	return js_ast.Expr{Data: &js_ast.EBinary{
		Op:    toJSBinaryOpCode(n.Operator),
		Left:  left,
		Right: right,
	}}, true, nil
}

// transformTernaryExpr converts a ternary expression from the internal AST to
// a JavaScript conditional expression.
//
// Takes n (*ast_domain.TernaryExpression) which is the ternary
// expression to convert.
// Takes registry (*RegistryContext) which provides context for the conversion.
//
// Returns js_ast.Expr which is the resulting JavaScript conditional expression.
// Returns bool which shows whether the conversion was handled.
// Returns error when any part of the expression fails to convert.
func transformTernaryExpr(n *ast_domain.TernaryExpression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	condition, err := transformOurASTtoJSAST(n.Condition, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	cons, err := transformOurASTtoJSAST(n.Consequent, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	alt, err := transformOurASTtoJSAST(n.Alternate, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	return js_ast.Expr{Data: &js_ast.EIf{Test: condition, Yes: cons, No: alt}}, true, nil
}

// tryTransformCallExpr transforms a function call expression to JavaScript.
//
// Takes expression (ast_domain.Expression) which is the expression to
// transform.
// Takes registry (*RegistryContext) which provides the transformation
// context.
//
// Returns js_ast.Expr which is the transformed JavaScript call expression.
// Returns bool which shows whether the expression was a call expression.
// Returns error when transforming the callee or arguments fails.
func tryTransformCallExpr(expression ast_domain.Expression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	n, ok := expression.(*ast_domain.CallExpression)
	if !ok {
		return js_ast.Expr{}, false, nil
	}

	callee, err := transformOurASTtoJSAST(n.Callee, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}

	jsArgs, err := transformExprList(n.Args, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}

	return js_ast.Expr{Data: &js_ast.ECall{
		Target:        callee,
		Args:          jsArgs,
		OptionalChain: js_ast.OptionalChainNone,
	}}, true, nil
}

// transformExprList converts a list of expressions to JavaScript AST form.
//
// Takes exprs ([]ast_domain.Expression) which contains the expressions to
// convert.
// Takes registry (*RegistryContext) which provides the conversion context.
//
// Returns []js_ast.Expr which contains the converted JavaScript expressions.
// Returns error when any expression in the list fails to convert.
func transformExprList(exprs []ast_domain.Expression, registry *RegistryContext) ([]js_ast.Expr, error) {
	result := make([]js_ast.Expr, len(exprs))
	for i, expression := range exprs {
		jsExpr, err := transformOurASTtoJSAST(expression, registry)
		if err != nil {
			return nil, fmt.Errorf("transforming expression at index %d: %w", i, err)
		}
		result[i] = jsExpr
	}
	return result, nil
}

// tryTransformComplexLiteral handles array, object, and template literals.
//
// Takes expression (ast_domain.Expression) which is the expression to
// transform.
// Takes registry (*RegistryContext) which provides the transformation
// context.
//
// Returns js_ast.Expr which is the transformed JavaScript AST expression.
// Returns bool which is true if the expression was transformed.
// Returns error when the transformation fails.
func tryTransformComplexLiteral(expression ast_domain.Expression, registry *RegistryContext) (js_ast.Expr, bool, error) {
	switch n := expression.(type) {
	case *ast_domain.ArrayLiteral:
		return transformArrayLiteral(n, registry)
	case *ast_domain.ObjectLiteral:
		return transformObjectLiteral(n, registry)
	case *ast_domain.TemplateLiteral:
		result, err := transformTemplateLiteral(n, registry)
		return result, true, err
	default:
		return js_ast.Expr{}, false, nil
	}
}

// transformArrayLiteral converts a domain array literal to a JavaScript array
// expression.
//
// Takes n (*ast_domain.ArrayLiteral) which is the array literal to convert.
// Takes registry (*RegistryContext) which provides context for the conversion.
//
// Returns js_ast.Expr which is the resulting JavaScript array expression.
// Returns bool which shows whether the conversion was handled.
// Returns error when element conversion fails.
func transformArrayLiteral(n *ast_domain.ArrayLiteral, registry *RegistryContext) (js_ast.Expr, bool, error) {
	jsElements, err := transformExprList(n.Elements, registry)
	if err != nil {
		return js_ast.Expr{}, true, err
	}
	return js_ast.Expr{Data: &js_ast.EArray{Items: jsElements}}, true, nil
}

// transformObjectLiteral converts an ObjectLiteral AST node to a JavaScript
// object expression. The keys are sorted in alphabetical order for consistent
// output.
//
// Takes n (*ast_domain.ObjectLiteral) which is the object literal to convert.
// Takes registry (*RegistryContext) which provides context for the transform.
//
// Returns js_ast.Expr which is the resulting JavaScript object expression.
// Returns bool which indicates whether the transform was handled.
// Returns error when a nested value cannot be transformed.
func transformObjectLiteral(n *ast_domain.ObjectLiteral, registry *RegistryContext) (js_ast.Expr, bool, error) {
	jsProperties := make([]js_ast.Property, 0, len(n.Pairs))

	keys := slices.Sorted(maps.Keys(n.Pairs))

	for _, k := range keys {
		jsVal, err := transformOurASTtoJSAST(n.Pairs[k], registry)
		if err != nil {
			return js_ast.Expr{}, true, err
		}
		jsProperties = append(jsProperties, js_ast.Property{
			Key:        newStringLiteral(k),
			ValueOrNil: jsVal,
			Kind:       js_ast.PropertyField,
		})
	}
	return js_ast.Expr{Data: &js_ast.EObject{Properties: jsProperties}}, true, nil
}

// toJSUnaryOpCode converts a unary operator to its esbuild OpCode.
//
// Takes op (ast_domain.UnaryOp) which is the unary operator to convert.
//
// Returns js_ast.OpCode which is the matching esbuild operator code.
func toJSUnaryOpCode(op ast_domain.UnaryOp) js_ast.OpCode {
	switch op {
	case ast_domain.OpNot:
		return js_ast.UnOpNot
	case ast_domain.OpNeg:
		return js_ast.UnOpNeg
	default:
		return js_ast.UnOpPos
	}
}

// toJSBinaryOpCode maps a binary operator to the matching esbuild OpCode.
//
// Takes op (ast_domain.BinaryOp) which is the binary operator to convert.
//
// Returns js_ast.OpCode which is the esbuild operator code. Returns BinOpAdd
// for unknown operators.
func toJSBinaryOpCode(op ast_domain.BinaryOp) js_ast.OpCode {
	if jsOp, ok := binaryOpMap[op]; ok {
		return jsOp
	}
	return js_ast.BinOpAdd
}

// transformTemplateLiteral converts a template literal into a chain of string
// values joined with the add operator.
//
// Takes n (*ast_domain.TemplateLiteral) which is the template literal to
// convert.
// Takes registry (*RegistryContext) which provides the context for conversion.
//
// Returns js_ast.Expr which is the resulting JavaScript expression.
// Returns error when the conversion fails.
func transformTemplateLiteral(n *ast_domain.TemplateLiteral, registry *RegistryContext) (js_ast.Expr, error) {
	if len(n.Parts) == 0 {
		return newStringLiteral(""), nil
	}

	jsExprParts := make([]js_ast.Expr, 0, len(n.Parts))
	for _, part := range n.Parts {
		if partExpr := transformTemplatePart(part, registry); partExpr.Data != nil {
			jsExprParts = append(jsExprParts, partExpr)
		}
	}

	if len(jsExprParts) == 0 {
		return newStringLiteral(""), nil
	}
	if len(jsExprParts) == 1 {
		return jsExprParts[0], nil
	}
	return chainWithAddOperator(jsExprParts), nil
}

// transformTemplatePart converts a single template literal part into a
// JavaScript expression.
//
// Takes part (ast_domain.TemplateLiteralPart) which is the template part to
// convert.
// Takes registry (*RegistryContext) which provides the conversion context.
//
// Returns js_ast.Expr which is the resulting JavaScript expression, or an
// empty expression when the part has no content.
func transformTemplatePart(part ast_domain.TemplateLiteralPart, registry *RegistryContext) js_ast.Expr {
	if part.IsLiteral {
		if part.Literal != "" {
			return newStringLiteral(part.Literal)
		}
		return js_ast.Expr{}
	}

	if part.Expression == nil {
		return js_ast.Expr{}
	}

	jsExpr, err := transformOurASTtoJSAST(part.Expression, registry)
	if err != nil {
		return newStringLiteral("/*EXPR_ERR*/")
	}

	return js_ast.Expr{Data: &js_ast.ECall{
		Target: newIdentifier("String"),
		Args: []js_ast.Expr{{Data: &js_ast.EBinary{
			Op:    js_ast.BinOpNullishCoalescing,
			Left:  jsExpr,
			Right: newStringLiteral(""),
		}}},
	}}
}

// chainWithAddOperator joins multiple expressions using the '+' operator.
//
// Takes exprs ([]js_ast.Expr) which contains the expressions to join.
//
// Returns js_ast.Expr which is the combined binary expression tree.
func chainWithAddOperator(exprs []js_ast.Expr) js_ast.Expr {
	finalExpr := exprs[0]
	for i := 1; i < len(exprs); i++ {
		finalExpr = js_ast.Expr{Data: &js_ast.EBinary{
			Op:    js_ast.BinOpAdd,
			Left:  finalExpr,
			Right: exprs[i], //nolint:gosec // G602: bounds guaranteed by loop condition
		}}
	}
	return finalExpr
}
