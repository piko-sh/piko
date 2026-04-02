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

	parsejs "github.com/tdewolff/parse/v2/js"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// jsImportKeyword is the JavaScript import keyword used in dynamic imports.
const jsImportKeyword = "import"

// convertExpression converts an esbuild expression to a tdewolff
// expression.
//
// Takes expression (js_ast.Expr) which is the esbuild expression
// to convert.
//
// Returns parsejs.IExpr which is the converted tdewolff
// expression.
// Returns error when conversion of a nested expression fails.
func (c *ASTConverter) convertExpression(expression js_ast.Expr) (parsejs.IExpr, error) {
	if expression.Data == nil {
		return nil, nil
	}

	if annotation, ok := expression.Data.(*js_ast.EAnnotation); ok {
		return c.convertExpression(annotation.Value)
	}

	if result := c.tryConvertPrimitiveLiteral(expression); result != nil {
		return result, nil
	}

	if result, handled, err := c.tryConvertValueExpr(expression); handled {
		return result, err
	}

	if result, handled, err := c.tryConvertCallExpr(expression); handled {
		return result, err
	}

	return c.convertOperatorOrFunctionExpr(expression)
}

// tryConvertPrimitiveLiteral converts simple esbuild literal
// expressions to their tdewolff equivalents.
//
// Takes expression (js_ast.Expr) which is the expression to
// convert.
//
// Returns parsejs.IExpr which is the converted literal
// expression, or nil if not a primitive literal.
func (*ASTConverter) tryConvertPrimitiveLiteral(expression js_ast.Expr) parsejs.IExpr {
	switch expression.Data.(type) {
	case *js_ast.ENull:
		return &parsejs.LiteralExpr{TokenType: parsejs.NullToken, Data: []byte("null")}
	case *js_ast.EUndefined:
		return &parsejs.Var{Data: []byte("undefined")}
	case *js_ast.EThis:
		return &parsejs.LiteralExpr{TokenType: parsejs.ThisToken, Data: []byte("this")}
	case *js_ast.ESuper:
		return &parsejs.LiteralExpr{TokenType: parsejs.SuperToken, Data: []byte("super")}
	default:
		return nil
	}
}

// tryConvertValueExpr handles identifiers, literals, and
// collection expressions.
//
// Takes expression (js_ast.Expr) which is the expression node to
// convert.
//
// Returns parsejs.IExpr which is the converted expression, or
// nil for elision.
// Returns bool which shows whether this method handled the
// expression type.
// Returns error when conversion of a supported expression type
// fails.
func (c *ASTConverter) tryConvertValueExpr(expression js_ast.Expr) (parsejs.IExpr, bool, error) {
	switch e := expression.Data.(type) {
	case *js_ast.EIdentifier:
		r, err := c.convertEIdentifier(e)
		return r, true, err
	case *js_ast.EString:
		r, err := c.convertEString(e)
		return r, true, err
	case *js_ast.ENumber:
		r, err := c.convertENumber(e)
		return r, true, err
	case *js_ast.EBoolean:
		r, err := c.convertEBoolean(e)
		return r, true, err
	case *js_ast.EArray:
		r, err := c.convertEArray(e)
		return r, true, err
	case *js_ast.EObject:
		r, err := c.convertEObject(e)
		return r, true, err
	case *js_ast.ETemplate:
		r, err := c.convertETemplate(e)
		return r, true, err
	case *js_ast.ERegExp:
		r, err := c.convertERegExp(e)
		return r, true, err
	case *js_ast.EBigInt:
		r, err := c.convertEBigInt(e)
		return r, true, err
	case *js_ast.EMissing:
		return nil, true, nil
	case *js_ast.EClass:
		r, err := c.convertEClass(e)
		return r, true, err
	case *js_ast.EPrivateIdentifier:
		r, err := c.convertEPrivateIdentifier(e)
		return r, true, err
	case *js_ast.EImportIdentifier:
		r, err := c.convertEImportIdentifier(e)
		return r, true, err
	default:
		return nil, false, nil
	}
}

// tryConvertCallExpr handles call and member access expressions.
//
// Takes expression (js_ast.Expr) which is the expression to
// convert.
//
// Returns parsejs.IExpr which is the converted expression, or
// nil if not handled.
// Returns bool which indicates whether this converter handled
// the expression.
// Returns error when conversion fails.
func (c *ASTConverter) tryConvertCallExpr(expression js_ast.Expr) (parsejs.IExpr, bool, error) {
	switch e := expression.Data.(type) {
	case *js_ast.ECall:
		r, err := c.convertECall(e)
		return r, true, err
	case *js_ast.ENew:
		r, err := c.convertENew(e)
		return r, true, err
	case *js_ast.EDot:
		r, err := c.convertEDot(e)
		return r, true, err
	case *js_ast.EIndex:
		r, err := c.convertEIndex(e)
		return r, true, err
	default:
		return nil, false, nil
	}
}

// convertOperatorOrFunctionExpr handles operators, control flow,
// and function expressions.
//
// Takes expression (js_ast.Expr) which is the expression to
// convert.
//
// Returns parsejs.IExpr which is the converted expression.
// Returns error when conversion of the underlying expression
// fails.
func (c *ASTConverter) convertOperatorOrFunctionExpr(expression js_ast.Expr) (parsejs.IExpr, error) {
	var result parsejs.IExpr
	var err error

	switch e := expression.Data.(type) {
	case *js_ast.EBinary:
		result, err = c.convertEBinary(e)
	case *js_ast.EUnary:
		result, err = c.convertEUnary(e)
	case *js_ast.EIf:
		result, err = c.convertEIf(e)
	case *js_ast.EArrow:
		result, err = c.convertEArrow(e)
	case *js_ast.EFunction:
		result, err = c.convertEFunction(e)
	case *js_ast.ESpread:
		result, err = c.convertESpread(e)
	case *js_ast.EAwait:
		result, err = c.convertEAwait(e)
	case *js_ast.EYield:
		result, err = c.convertEYield(e)
	case *js_ast.EImportCall:
		result, err = c.convertEImportCall(e)
	case *js_ast.EImportString:
		result, err = c.convertEImportString(e)
	case *js_ast.ENewTarget:
		result, err = c.convertENewTarget()
	case *js_ast.EImportMeta:
		result, err = c.convertEImportMeta()
	default:
		return &parsejs.Var{Data: []byte("/* unsupported */")}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("converting %T expression: %w", expression.Data, err)
	}
	return result, nil
}

// convertEIdentifier converts an identifier expression to a variable.
//
// Takes e (*js_ast.EIdentifier) which is the identifier to convert.
//
// Returns parsejs.IExpr which is the converted variable expression.
// Returns error when conversion fails.
func (c *ASTConverter) convertEIdentifier(e *js_ast.EIdentifier) (parsejs.IExpr, error) {
	var name string
	if c.registry != nil {
		name = c.registry.LookupIdentifierName(e)
	}
	if name == "" {
		name = c.resolveRef(e.Ref)
	}
	if name == "" {
		name = "unknown"
	}
	return &parsejs.Var{Data: []byte(name)}, nil
}

// convertEPrivateIdentifier converts a private identifier expression
// (e.g., #field) to its parsed form.
//
// Takes e (*js_ast.EPrivateIdentifier) which is the private identifier to
// convert.
//
// Returns parsejs.IExpr which is the converted literal expression.
// Returns error when conversion fails.
func (c *ASTConverter) convertEPrivateIdentifier(e *js_ast.EPrivateIdentifier) (parsejs.IExpr, error) {
	name := c.resolveRef(e.Ref)
	if name == "" {
		name = "private"
	}
	if len(name) == 0 || name[0] != '#' {
		name = "#" + name
	}
	return &parsejs.LiteralExpr{
		TokenType: parsejs.PrivateIdentifierToken,
		Data:      []byte(name),
	}, nil
}

// convertEImportIdentifier converts an ES6 import identifier expression.
// This is similar to EIdentifier but refers to an imported symbol rather than
// a local one.
//
// Takes e (*js_ast.EImportIdentifier) which is the import identifier to
// convert.
//
// Returns parsejs.IExpr which is the converted variable expression.
// Returns error when conversion fails.
func (c *ASTConverter) convertEImportIdentifier(e *js_ast.EImportIdentifier) (parsejs.IExpr, error) {
	name := c.resolveRef(e.Ref)
	if name == "" {
		name = jsImportKeyword
	}
	return &parsejs.Var{Data: []byte(name)}, nil
}

// convertEString converts a string literal expression to its parsed form.
//
// Takes e (*js_ast.EString) which is the string expression to convert.
//
// Returns parsejs.IExpr which is the converted literal expression.
// Returns error when conversion fails.
func (*ASTConverter) convertEString(e *js_ast.EString) (parsejs.IExpr, error) {
	str := helpers.UTF16ToString(e.Value)
	return &parsejs.LiteralExpr{
		TokenType: parsejs.StringToken,
		Data:      fmt.Appendf(nil, fmtQuotedString, str),
	}, nil
}

// convertENumber converts a number literal.
//
// Takes e (*js_ast.ENumber) which contains the numeric value to convert.
//
// Returns parsejs.IExpr which is the converted literal expression.
// Returns error when conversion fails.
func (*ASTConverter) convertENumber(e *js_ast.ENumber) (parsejs.IExpr, error) {
	return &parsejs.LiteralExpr{
		TokenType: parsejs.DecimalToken,
		Data:      fmt.Appendf(nil, "%g", e.Value),
	}, nil
}

// convertEBoolean converts a boolean literal to a literal expression.
//
// Takes e (*js_ast.EBoolean) which contains the boolean value to convert.
//
// Returns parsejs.IExpr which is the converted literal expression.
// Returns error when conversion fails.
func (*ASTConverter) convertEBoolean(e *js_ast.EBoolean) (parsejs.IExpr, error) {
	if e.Value {
		return &parsejs.LiteralExpr{TokenType: parsejs.TrueToken, Data: []byte("true")}, nil
	}
	return &parsejs.LiteralExpr{TokenType: parsejs.FalseToken, Data: []byte("false")}, nil
}

// convertERegExp converts a regular expression literal.
//
// Takes e (*js_ast.ERegExp) which contains the regular expression value.
//
// Returns parsejs.IExpr which is the literal expression for the regexp.
// Returns error when conversion fails.
func (*ASTConverter) convertERegExp(e *js_ast.ERegExp) (parsejs.IExpr, error) {
	return &parsejs.LiteralExpr{
		TokenType: parsejs.RegExpToken,
		Data:      []byte(e.Value),
	}, nil
}

// convertEArray converts an array literal expression.
//
// Takes e (*js_ast.EArray) which is the array expression to convert.
//
// Returns parsejs.IExpr which is the converted array expression.
// Returns error when any element in the array fails to convert.
func (c *ASTConverter) convertEArray(e *js_ast.EArray) (parsejs.IExpr, error) {
	elements := make([]parsejs.Element, 0, len(e.Items))
	for i, item := range e.Items {
		converted, err := c.convertExpression(item)
		if err != nil {
			return nil, fmt.Errorf("converting array element %d: %w", i, err)
		}
		elements = append(elements, parsejs.Element{Value: converted})
	}
	return &parsejs.ArrayExpr{List: elements}, nil
}

// convertEObject converts an object literal expression.
//
// Takes e (*js_ast.EObject) which is the esbuild object expression to convert.
//
// Returns parsejs.IExpr which is the converted object expression.
// Returns error when a property conversion fails.
func (c *ASTConverter) convertEObject(e *js_ast.EObject) (parsejs.IExpr, error) {
	properties := make([]parsejs.Property, 0, len(e.Properties))
	for i, prop := range e.Properties {
		convertedProp, err := c.convertProperty(prop)
		if err != nil {
			return nil, fmt.Errorf("converting object property %d: %w", i, err)
		}
		if convertedProp != nil {
			properties = append(properties, *convertedProp)
		}
	}
	return &parsejs.ObjectExpr{List: properties}, nil
}

// convertECall converts a function call expression.
//
// Takes e (*js_ast.ECall) which is the esbuild call expression to convert.
//
// Returns parsejs.IExpr which is the converted call expression.
// Returns error when the target or any argument cannot be converted.
func (c *ASTConverter) convertECall(e *js_ast.ECall) (parsejs.IExpr, error) {
	target, err := c.convertExpression(e.Target)
	if err != nil {
		return nil, fmt.Errorf("converting call target: %w", err)
	}

	arguments := make([]parsejs.Arg, 0, len(e.Args))
	for i, argument := range e.Args {
		converted, err := c.convertExpression(argument)
		if err != nil {
			return nil, fmt.Errorf("converting call argument %d: %w", i, err)
		}
		arguments = append(arguments, parsejs.Arg{Value: converted})
	}

	target = wrapIIFETarget(e.Target, target)

	return &parsejs.CallExpr{
		X:        target,
		Args:     parsejs.Args{List: arguments},
		Optional: e.OptionalChain == js_ast.OptionalChainStart,
	}, nil
}

// convertENew converts a new expression.
//
// Takes e (*js_ast.ENew) which is the new expression to convert.
//
// Returns parsejs.IExpr which is the converted new expression.
// Returns error when the target or any argument fails to convert.
func (c *ASTConverter) convertENew(e *js_ast.ENew) (parsejs.IExpr, error) {
	target, err := c.convertExpression(e.Target)
	if err != nil {
		return nil, fmt.Errorf("converting new target: %w", err)
	}

	arguments := make([]parsejs.Arg, 0, len(e.Args))
	for i, argument := range e.Args {
		converted, err := c.convertExpression(argument)
		if err != nil {
			return nil, fmt.Errorf("converting new argument %d: %w", i, err)
		}
		arguments = append(arguments, parsejs.Arg{Value: converted})
	}

	return &parsejs.NewExpr{
		X:    target,
		Args: &parsejs.Args{List: arguments},
	}, nil
}

// convertEDot converts a dot/member expression to the internal AST format.
// It wraps the target in GroupExpr when needed to preserve correct operator
// precedence, e.g., (a || []).map(...) must not become a || [].map(...).
//
// Takes e (*js_ast.EDot) which is the dot expression to convert.
//
// Returns parsejs.IExpr which is the converted dot expression.
// Returns error when the target expression cannot be converted.
func (c *ASTConverter) convertEDot(e *js_ast.EDot) (parsejs.IExpr, error) {
	target, err := c.convertExpression(e.Target)
	if err != nil {
		return nil, fmt.Errorf("converting dot target for %q: %w", e.Name, err)
	}

	target = wrapLowPrecedenceForMemberAccess(e.Target, target)

	return &parsejs.DotExpr{
		X: target,
		Y: parsejs.LiteralExpr{
			TokenType: parsejs.IdentifierToken,
			Data:      []byte(e.Name),
		},
		Optional: e.OptionalChain == js_ast.OptionalChainStart,
	}, nil
}

// convertEIndex converts a bracket index expression (e.g. obj[key]).
//
// Takes e (*js_ast.EIndex) which is the index expression to convert.
//
// Returns parsejs.IExpr which is the converted index expression.
// Returns error when the target or index expression cannot be converted.
func (c *ASTConverter) convertEIndex(e *js_ast.EIndex) (parsejs.IExpr, error) {
	target, err := c.convertExpression(e.Target)
	if err != nil {
		return nil, fmt.Errorf("converting index target: %w", err)
	}
	index, err := c.convertExpression(e.Index)
	if err != nil {
		return nil, fmt.Errorf("converting index key: %w", err)
	}

	return &parsejs.IndexExpr{
		X:        target,
		Y:        index,
		Optional: e.OptionalChain == js_ast.OptionalChainStart,
	}, nil
}

// convertEBinary converts a binary expression. It wraps child expressions in
// GroupExpr when needed to keep the correct order of operations, since the
// printer does not add brackets on its own.
//
// Takes e (*js_ast.EBinary) which is the binary expression to convert.
//
// Returns parsejs.IExpr which is the converted binary expression.
// Returns error when a child expression fails to convert.
func (c *ASTConverter) convertEBinary(e *js_ast.EBinary) (parsejs.IExpr, error) {
	left, err := c.convertExpression(e.Left)
	if err != nil {
		return nil, fmt.Errorf("converting binary left operand: %w", err)
	}
	right, err := c.convertExpression(e.Right)
	if err != nil {
		return nil, fmt.Errorf("converting binary right operand: %w", err)
	}

	parentPrec := getOpPrecedence(e.Op)

	if leftBin, ok := e.Left.Data.(*js_ast.EBinary); ok {
		leftPrec := getOpPrecedence(leftBin.Op)
		if leftPrec < parentPrec {
			left = &parsejs.GroupExpr{X: left}
		}
	}

	if rightBin, ok := e.Right.Data.(*js_ast.EBinary); ok {
		rightPrec := getOpPrecedence(rightBin.Op)
		if rightPrec < parentPrec || (rightPrec == parentPrec && e.Op.IsLeftAssociative()) {
			right = &parsejs.GroupExpr{X: right}
		}
	}

	if _, ok := e.Right.Data.(*js_ast.EIf); ok {
		right = &parsejs.GroupExpr{X: right}
	}
	if _, ok := e.Left.Data.(*js_ast.EIf); ok {
		left = &parsejs.GroupExpr{X: left}
	}

	return &parsejs.BinaryExpr{
		Op: convertBinaryOp(e.Op),
		X:  left,
		Y:  right,
	}, nil
}

// convertEUnary converts a unary expression.
// UnOpPos (unary plus) is used as a marker for dynamic attribute values
// and should be converted to a GroupExpr (parentheses), not a literal +.
//
// Takes e (*js_ast.EUnary) which is the unary expression to convert.
//
// Returns parsejs.IExpr which is the converted expression.
// Returns error when the inner expression conversion fails.
func (c *ASTConverter) convertEUnary(e *js_ast.EUnary) (parsejs.IExpr, error) {
	value, err := c.convertExpression(e.Value)
	if err != nil {
		return nil, fmt.Errorf("converting unary expression operand: %w", err)
	}

	if e.Op == js_ast.UnOpPos {
		return &parsejs.GroupExpr{X: value}, nil
	}

	return &parsejs.UnaryExpr{
		Op: convertUnaryOp(e.Op),
		X:  value,
	}, nil
}

// convertEIf converts a ternary conditional expression.
//
// Takes e (*js_ast.EIf) which is the conditional expression to convert.
//
// Returns parsejs.IExpr which is the converted conditional expression.
// Returns error when any part of the expression fails to convert.
func (c *ASTConverter) convertEIf(e *js_ast.EIf) (parsejs.IExpr, error) {
	test, err := c.convertExpression(e.Test)
	if err != nil {
		return nil, fmt.Errorf("converting ternary condition: %w", err)
	}
	yes, err := c.convertExpression(e.Yes)
	if err != nil {
		return nil, fmt.Errorf("converting ternary consequent: %w", err)
	}
	no, err := c.convertExpression(e.No)
	if err != nil {
		return nil, fmt.Errorf("converting ternary alternate: %w", err)
	}

	return &parsejs.CondExpr{
		Cond: test,
		X:    yes,
		Y:    no,
	}, nil
}

// convertEArrow converts an arrow function expression.
//
// Takes e (*js_ast.EArrow) which is the arrow function expression to convert.
//
// Returns parsejs.IExpr which is the converted arrow function.
// Returns error when parameter or body conversion fails.
func (c *ASTConverter) convertEArrow(e *js_ast.EArrow) (parsejs.IExpr, error) {
	params, err := c.convertParams(e.Args)
	if err != nil {
		return nil, fmt.Errorf("converting arrow function parameters: %w", err)
	}

	body, err := c.convertFunctionBody(e.Body)
	if err != nil {
		return nil, fmt.Errorf("converting arrow function body: %w", err)
	}

	return &parsejs.ArrowFunc{
		Async:  e.IsAsync,
		Params: params,
		Body:   *body,
	}, nil
}

// convertEFunction converts a function expression to a function declaration.
//
// Takes e (*js_ast.EFunction) which is the function expression to convert.
//
// Returns parsejs.IExpr which is the converted function declaration.
// Returns error when parameter or body conversion fails.
func (c *ASTConverter) convertEFunction(e *js_ast.EFunction) (parsejs.IExpr, error) {
	params, err := c.convertParams(e.Fn.Args)
	if err != nil {
		return nil, fmt.Errorf("converting function expression parameters: %w", err)
	}

	body, err := c.convertFunctionBody(e.Fn.Body)
	if err != nil {
		return nil, fmt.Errorf("converting function expression body: %w", err)
	}

	var name *parsejs.Var
	if e.Fn.Name != nil {
		if resolved := c.resolveRef(e.Fn.Name.Ref); resolved != "" {
			name = &parsejs.Var{Data: []byte(resolved)}
		}
	}

	return &parsejs.FuncDecl{
		Async:     e.Fn.IsAsync,
		Generator: e.Fn.IsGenerator,
		Name:      name,
		Params:    params,
		Body:      *body,
	}, nil
}

// convertETemplate converts a template literal expression.
//
// Takes e (*js_ast.ETemplate) which is the template expression to convert.
//
// Returns parsejs.IExpr which is the converted template expression.
// Returns error when the conversion fails.
func (c *ASTConverter) convertETemplate(e *js_ast.ETemplate) (parsejs.IExpr, error) {
	if len(e.Parts) == 0 {
		return c.convertSimpleTemplate(e)
	}
	return c.convertInterpolatedTemplate(e)
}

// convertSimpleTemplate converts a template literal with no interpolations.
//
// Takes e (*js_ast.ETemplate) which is the template expression to convert.
//
// Returns parsejs.IExpr which is the converted template expression.
// Returns error when the conversion fails.
func (*ASTConverter) convertSimpleTemplate(e *js_ast.ETemplate) (parsejs.IExpr, error) {
	headString := ""
	if e.HeadCooked != nil {
		headString = helpers.UTF16ToString(e.HeadCooked)
	}
	return &parsejs.TemplateExpr{
		List: nil,
		Tail: []byte("`" + headString + "`"),
	}, nil
}

// convertInterpolatedTemplate converts a template literal with embedded
// expressions into an internal template expression.
//
// Takes e (*js_ast.ETemplate) which is the template expression to convert.
//
// Returns parsejs.IExpr which is the converted template expression.
// Returns error when a nested expression fails to convert.
func (c *ASTConverter) convertInterpolatedTemplate(e *js_ast.ETemplate) (parsejs.IExpr, error) {
	parts := make([]parsejs.TemplatePart, 0, len(e.Parts))

	headString := ""
	if e.HeadCooked != nil {
		headString = helpers.UTF16ToString(e.HeadCooked)
	}
	previousString := headString

	var finalTail string
	for i, part := range e.Parts {
		expression, err := c.convertExpression(part.Value)
		if err != nil {
			return nil, fmt.Errorf("converting template part %d: %w", i, err)
		}

		var prefix string
		if i == 0 {
			prefix = "`" + previousString + "${"
		} else {
			prefix = "}" + previousString + "${"
		}

		parts = append(parts, parsejs.TemplatePart{
			Value: []byte(prefix),
			Expr:  expression,
		})

		previousString = ""
		if part.TailCooked != nil {
			previousString = helpers.UTF16ToString(part.TailCooked)
		}
		finalTail = previousString
	}

	return &parsejs.TemplateExpr{
		List: parts,
		Tail: []byte("}" + finalTail + "`"),
	}, nil
}

// convertESpread converts a spread expression to a unary expression.
//
// Takes e (*js_ast.ESpread) which is the spread expression to convert.
//
// Returns parsejs.IExpr which is the converted unary expression.
// Returns error when the inner value cannot be converted.
func (c *ASTConverter) convertESpread(e *js_ast.ESpread) (parsejs.IExpr, error) {
	value, err := c.convertExpression(e.Value)
	if err != nil {
		return nil, fmt.Errorf("converting spread expression value: %w", err)
	}
	return &parsejs.UnaryExpr{Op: parsejs.EllipsisToken, X: value}, nil
}

// convertEAwait converts an await expression to a unary expression.
//
// Takes e (*js_ast.EAwait) which holds the expression being awaited.
//
// Returns parsejs.IExpr which is the converted unary await expression.
// Returns error when the inner expression cannot be converted.
func (c *ASTConverter) convertEAwait(e *js_ast.EAwait) (parsejs.IExpr, error) {
	value, err := c.convertExpression(e.Value)
	if err != nil {
		return nil, fmt.Errorf("converting await expression value: %w", err)
	}
	return &parsejs.UnaryExpr{
		Op: parsejs.AwaitToken,
		X:  value,
	}, nil
}

// convertEYield converts a yield expression.
//
// Takes e (*js_ast.EYield) which specifies the yield expression to convert.
//
// Returns parsejs.IExpr which is the converted yield expression.
// Returns error when the value expression cannot be converted.
func (c *ASTConverter) convertEYield(e *js_ast.EYield) (parsejs.IExpr, error) {
	var value parsejs.IExpr
	var err error
	if e.ValueOrNil.Data != nil {
		value, err = c.convertExpression(e.ValueOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting yield expression value: %w", err)
		}
	}

	if e.IsStar {
		return &parsejs.UnaryExpr{
			Op: parsejs.YieldToken,
			X: &parsejs.UnaryExpr{
				Op: parsejs.MulToken,
				X:  value,
			},
		}, nil
	}

	return &parsejs.UnaryExpr{
		Op: parsejs.YieldToken,
		X:  value,
	}, nil
}

// convertEBigInt converts a BigInt literal.
//
// Takes e (*js_ast.EBigInt) which contains the BigInt value without the 'n'
// suffix.
//
// Returns parsejs.IExpr which is the literal expression with the 'n' suffix
// appended.
// Returns error when conversion fails.
func (*ASTConverter) convertEBigInt(e *js_ast.EBigInt) (parsejs.IExpr, error) {
	return &parsejs.LiteralExpr{
		TokenType: parsejs.IntegerToken,
		Data:      []byte(e.Value + "n"),
	}, nil
}

// convertEClass converts a class expression to a ClassDecl.
//
// Takes e (*js_ast.EClass) which is the class expression to convert.
//
// Returns parsejs.IExpr which is the converted class declaration.
// Returns error when the extends clause or properties cannot be converted.
func (c *ASTConverter) convertEClass(e *js_ast.EClass) (parsejs.IExpr, error) {
	var extends parsejs.IExpr
	if e.Class.ExtendsOrNil.Data != nil {
		var err error
		extends, err = c.convertExpression(e.Class.ExtendsOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting class expression extends: %w", err)
		}
	}

	elements := make([]parsejs.ClassElement, 0, len(e.Class.Properties))
	for i, prop := range e.Class.Properties {
		element, err := c.convertClassProperty(prop)
		if err != nil {
			return nil, fmt.Errorf("converting class expression property %d: %w", i, err)
		}
		if element != nil {
			elements = append(elements, *element)
		}
	}

	var name *parsejs.Var
	if e.Class.Name != nil {
		var className string
		if c.registry != nil {
			className = c.registry.LookupLocRefName(e.Class.Name)
		}
		if className == "" {
			className = c.resolveRef(e.Class.Name.Ref)
		}
		if className != "" {
			name = &parsejs.Var{Data: []byte(className)}
		}
	}

	return &parsejs.ClassDecl{
		Name:    name,
		Extends: extends,
		List:    elements,
	}, nil
}

// convertEImportCall converts a dynamic import() expression.
//
// Takes e (*js_ast.EImportCall) which contains the import expression to
// convert.
//
// Returns parsejs.IExpr which is the converted call expression.
// Returns error when the inner expression conversion fails.
func (c *ASTConverter) convertEImportCall(e *js_ast.EImportCall) (parsejs.IExpr, error) {
	argument, err := c.convertExpression(e.Expr)
	if err != nil {
		return nil, fmt.Errorf("converting dynamic import argument: %w", err)
	}

	arguments := []parsejs.Arg{{Value: argument}}

	if e.OptionsOrNil.Data != nil {
		options, err := c.convertExpression(e.OptionsOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting dynamic import options: %w", err)
		}
		arguments = append(arguments, parsejs.Arg{Value: options})
	}

	return &parsejs.CallExpr{
		X:    &parsejs.Var{Data: []byte(jsImportKeyword)},
		Args: parsejs.Args{List: arguments},
	}, nil
}

// convertEImportString converts a dynamic import with a string literal path.
// EImportString is used by esbuild when the import path is a known string.
//
// Takes e (*js_ast.EImportString) which holds the import record index.
//
// Returns parsejs.IExpr which is the converted dynamic import call.
// Returns error when the conversion fails.
func (c *ASTConverter) convertEImportString(e *js_ast.EImportString) (parsejs.IExpr, error) {
	var path string
	if int(e.ImportRecordIndex) < len(c.importRecords) {
		path = c.importRecords[e.ImportRecordIndex].Path.Text
	} else {
		path = "unknown"
	}

	return &parsejs.CallExpr{
		X: &parsejs.Var{Data: []byte(jsImportKeyword)},
		Args: parsejs.Args{
			List: []parsejs.Arg{{
				Value: &parsejs.LiteralExpr{
					TokenType: parsejs.StringToken,
					Data:      fmt.Appendf(nil, "%q", path),
				},
			}},
		},
	}, nil
}

// convertENewTarget converts a new.target expression to an AST node.
//
// Returns parsejs.IExpr which is a dot expression for new.target.
// Returns error when conversion fails.
func (*ASTConverter) convertENewTarget() (parsejs.IExpr, error) {
	return &parsejs.DotExpr{
		X: &parsejs.LiteralExpr{
			TokenType: parsejs.NewToken,
			Data:      []byte("new"),
		},
		Y: parsejs.LiteralExpr{
			TokenType: parsejs.IdentifierToken,
			Data:      []byte("target"),
		},
	}, nil
}

// convertEImportMeta converts an import.meta expression to its AST form.
//
// Returns parsejs.IExpr which is the AST node for import.meta.
// Returns error when the conversion fails.
func (*ASTConverter) convertEImportMeta() (parsejs.IExpr, error) {
	return &parsejs.DotExpr{
		X: &parsejs.LiteralExpr{
			TokenType: parsejs.ImportToken,
			Data:      []byte(jsImportKeyword),
		},
		Y: parsejs.LiteralExpr{
			TokenType: parsejs.IdentifierToken,
			Data:      []byte("meta"),
		},
	}, nil
}

// wrapIIFETarget wraps arrow or function expressions in brackets for IIFE
// calls.
//
// Takes original (js_ast.Expr) which is the original JavaScript AST expression.
// Takes converted (parsejs.IExpr) which is the converted expression to wrap.
//
// Returns parsejs.IExpr which is the wrapped expression if original is an arrow
// or function expression, otherwise returns converted unchanged.
func wrapIIFETarget(original js_ast.Expr, converted parsejs.IExpr) parsejs.IExpr {
	switch original.Data.(type) {
	case *js_ast.EArrow, *js_ast.EFunction:
		return &parsejs.GroupExpr{X: converted}
	}
	return converted
}

// wrapLowPrecedenceForMemberAccess wraps expressions that have lower
// precedence than member access in parentheses. This means members are
// accessed in the correct order.
//
// Takes original (js_ast.Expr) which is the source expression to check.
// Takes converted (parsejs.IExpr) which is the already-converted expression.
//
// Returns parsejs.IExpr which is the converted expression, wrapped in
// parentheses if needed.
func wrapLowPrecedenceForMemberAccess(original js_ast.Expr, converted parsejs.IExpr) parsejs.IExpr {
	switch original.Data.(type) {
	case *js_ast.EBinary, *js_ast.EIf:
		return &parsejs.GroupExpr{X: converted}
	}
	return converted
}
