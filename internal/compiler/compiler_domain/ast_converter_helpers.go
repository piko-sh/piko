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

// Binding conversion

// convertBinding converts an esbuild binding to a tdewolff binding.
//
// Takes binding (js_ast.Binding) which is the esbuild binding to convert.
//
// Returns parsejs.IBinding which is the converted tdewolff binding.
// Returns error when the conversion fails.
func (c *ASTConverter) convertBinding(binding js_ast.Binding) (parsejs.IBinding, error) {
	if binding.Data == nil {
		return nil, nil
	}

	switch b := binding.Data.(type) {
	case *js_ast.BIdentifier:
		return c.convertBIdentifier(b)
	case *js_ast.BArray:
		return c.convertBArray(b)
	case *js_ast.BObject:
		return c.convertBObject(b)
	default:
		return &parsejs.Var{Data: []byte("binding")}, nil
	}
}

// convertBIdentifier converts an identifier binding to a variable binding.
//
// Takes b (*js_ast.BIdentifier) which is the identifier binding to convert.
//
// Returns parsejs.IBinding which is the converted variable binding.
// Returns error when the conversion fails.
func (c *ASTConverter) convertBIdentifier(b *js_ast.BIdentifier) (parsejs.IBinding, error) {
	var name string
	if c.registry != nil {
		name = c.registry.LookupBindingName(b)
	}
	if name == "" {
		name = c.resolveRef(b.Ref)
	}
	if name == "" {
		name = "binding"
	}
	return &parsejs.Var{Data: []byte(name)}, nil
}

// convertBArray converts an array destructuring binding.
//
// Takes b (*js_ast.BArray) which is the array binding pattern to convert.
//
// Returns parsejs.IBinding which is the converted array binding.
// Returns error when any element binding or default expression fails to
// convert.
func (c *ASTConverter) convertBArray(b *js_ast.BArray) (parsejs.IBinding, error) {
	elements := make([]parsejs.BindingElement, 0, len(b.Items))
	for _, item := range b.Items {
		converted, err := c.convertBinding(item.Binding)
		if err != nil {
			return nil, fmt.Errorf("converting array binding element: %w", err)
		}

		var defaultExpr parsejs.IExpr
		if item.DefaultValueOrNil.Data != nil {
			defaultExpr, err = c.convertExpression(item.DefaultValueOrNil)
			if err != nil {
				return nil, fmt.Errorf("converting array binding default value: %w", err)
			}
		}

		elements = append(elements, parsejs.BindingElement{
			Binding: converted,
			Default: defaultExpr,
		})
	}
	return &parsejs.BindingArray{List: elements}, nil
}

// convertBObject converts an object destructuring binding.
//
// Takes b (*js_ast.BObject) which is the object binding to convert.
//
// Returns parsejs.IBinding which is the converted binding object.
// Returns error when a property value or default expression fails to convert.
func (c *ASTConverter) convertBObject(b *js_ast.BObject) (parsejs.IBinding, error) {
	props := make([]parsejs.BindingObjectItem, 0, len(b.Properties))
	for _, prop := range b.Properties {
		key := c.convertBindingPropertyKey(prop.Key)

		value, err := c.convertBinding(prop.Value)
		if err != nil {
			return nil, fmt.Errorf("converting object binding value: %w", err)
		}

		var defaultExpr parsejs.IExpr
		if prop.DefaultValueOrNil.Data != nil {
			defaultExpr, err = c.convertExpression(prop.DefaultValueOrNil)
			if err != nil {
				return nil, fmt.Errorf("converting object binding default value: %w", err)
			}
		}

		props = append(props, parsejs.BindingObjectItem{
			Key:   key,
			Value: parsejs.BindingElement{Binding: value, Default: defaultExpr},
		})
	}
	return &parsejs.BindingObject{List: props}, nil
}

// convertBindingPropertyKey converts an AST expression for a property key into
// a PropertyName.
//
// Takes key (js_ast.Expr) which is the AST expression for the property key.
//
// Returns *parsejs.PropertyName which is the converted property name, or nil
// if the key type is not supported. Handles identifier and string expressions.
func (c *ASTConverter) convertBindingPropertyKey(key js_ast.Expr) *parsejs.PropertyName {
	if identifier, ok := key.Data.(*js_ast.EIdentifier); ok {
		name := c.resolveRef(identifier.Ref)
		if name == "" {
			name = "key"
		}
		return &parsejs.PropertyName{
			Literal: parsejs.LiteralExpr{TokenType: parsejs.IdentifierToken, Data: []byte(name)},
		}
	}
	if str, ok := key.Data.(*js_ast.EString); ok {
		return &parsejs.PropertyName{
			Literal: parsejs.LiteralExpr{TokenType: parsejs.StringToken, Data: fmt.Appendf(nil, fmtQuotedString, helpers.UTF16ToString(str.Value))},
		}
	}
	return nil
}

// convertParams converts function arguments to parameter bindings.
//
// Takes arguments ([]js_ast.Arg) which contains the function arguments to convert.
//
// Returns parsejs.Params which contains the converted binding elements.
// Returns error when a binding or default value cannot be converted.
func (c *ASTConverter) convertParams(arguments []js_ast.Arg) (parsejs.Params, error) {
	elements := make([]parsejs.BindingElement, 0, len(arguments))
	for _, argument := range arguments {
		binding, err := c.convertBinding(argument.Binding)
		if err != nil {
			return parsejs.Params{}, fmt.Errorf("converting parameter binding: %w", err)
		}

		var defaultExpr parsejs.IExpr
		if argument.DefaultOrNil.Data != nil {
			defaultExpr, err = c.convertExpression(argument.DefaultOrNil)
			if err != nil {
				return parsejs.Params{}, fmt.Errorf("converting parameter default value: %w", err)
			}
		}

		elements = append(elements, parsejs.BindingElement{
			Binding: binding,
			Default: defaultExpr,
		})
	}

	return parsejs.Params{List: elements}, nil
}

// convertFunctionBody converts a function body to a block statement.
//
// Takes body (js_ast.FnBody) which contains the function body to convert.
//
// Returns *parsejs.BlockStmt which contains the converted statements.
// Returns error when a statement conversion fails.
func (c *ASTConverter) convertFunctionBody(body js_ast.FnBody) (*parsejs.BlockStmt, error) {
	statements := make([]parsejs.IStmt, 0, len(body.Block.Stmts))
	for _, statement := range body.Block.Stmts {
		converted, err := c.convertStatement(statement)
		if err != nil {
			return nil, fmt.Errorf("converting function body statement: %w", err)
		}
		if converted != nil {
			statements = append(statements, converted)
		}
	}

	return &parsejs.BlockStmt{List: statements}, nil
}

// Property conversion

// convertProperty converts a JavaScript object property to internal form.
//
// Takes prop (js_ast.Property) which is the property to convert.
//
// Returns *parsejs.Property which is the converted property.
// Returns error when the property key or value cannot be converted.
func (c *ASTConverter) convertProperty(prop js_ast.Property) (*parsejs.Property, error) {
	if prop.Kind == js_ast.PropertySpread {
		value, err := c.convertExpression(prop.ValueOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting spread property value: %w", err)
		}
		return &parsejs.Property{
			Spread: true,
			Value:  value,
		}, nil
	}

	if shorthandProp := c.tryConvertShorthandProperty(prop); shorthandProp != nil {
		return shorthandProp, nil
	}

	name, err := c.convertPropertyName(prop.Key)
	if err != nil {
		return nil, fmt.Errorf("converting property name: %w", err)
	}

	var value parsejs.IExpr
	if prop.ValueOrNil.Data != nil {
		value, err = c.convertExpression(prop.ValueOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting property value: %w", err)
		}
	}

	return &parsejs.Property{
		Name:  name,
		Value: value,
	}, nil
}

// tryConvertShorthandProperty tries to convert a shorthand property.
//
// Takes prop (js_ast.Property) which is the property to convert.
//
// Returns *parsejs.Property which is the converted property, or nil if the
// property is not a shorthand property or cannot be converted.
func (c *ASTConverter) tryConvertShorthandProperty(prop js_ast.Property) *parsejs.Property {
	if !prop.Flags.Has(js_ast.PropertyWasShorthand) {
		return nil
	}
	identifier, ok := prop.Key.Data.(*js_ast.EIdentifier)
	if !ok {
		return nil
	}
	name := c.resolveRef(identifier.Ref)
	if name == "" {
		name = "prop"
	}
	return &parsejs.Property{
		Name:  &parsejs.PropertyName{Literal: parsejs.LiteralExpr{TokenType: parsejs.IdentifierToken, Data: []byte(name)}},
		Value: &parsejs.Var{Data: []byte(name)},
	}
}

// convertPropertyName converts a property key expression to a PropertyName.
//
// Takes key (js_ast.Expr) which is the property key expression to convert.
//
// Returns *parsejs.PropertyName which represents the converted property name.
// Returns error when the key expression cannot be converted.
func (c *ASTConverter) convertPropertyName(key js_ast.Expr) (*parsejs.PropertyName, error) {
	switch k := key.Data.(type) {
	case *js_ast.EString:
		return &parsejs.PropertyName{
			Literal: parsejs.LiteralExpr{
				TokenType: parsejs.StringToken,
				Data:      fmt.Appendf(nil, fmtQuotedString, helpers.UTF16ToString(k.Value)),
			},
		}, nil

	case *js_ast.EIdentifier:
		propName := c.resolveRef(k.Ref)
		if propName == "" {
			propName = "property"
		}
		return &parsejs.PropertyName{
			Literal: parsejs.LiteralExpr{
				TokenType: parsejs.IdentifierToken,
				Data:      []byte(propName),
			},
		}, nil

	case *js_ast.ENumber:
		return &parsejs.PropertyName{
			Literal: parsejs.LiteralExpr{
				TokenType: parsejs.DecimalToken,
				Data:      fmt.Appendf(nil, "%g", k.Value),
			},
		}, nil

	default:
		computed, err := c.convertExpression(key)
		if err != nil {
			return nil, fmt.Errorf("converting computed property name: %w", err)
		}
		return &parsejs.PropertyName{Computed: computed}, nil
	}
}

// convertClassProperty converts a class property or method to a ClassElement.
//
// Takes prop (js_ast.Property) which is the property to convert.
//
// Returns *parsejs.ClassElement which is the converted element.
// Returns error when the conversion fails.
func (c *ASTConverter) convertClassProperty(prop js_ast.Property) (*parsejs.ClassElement, error) {
	elemName := c.getClassElementName(prop)

	if jsFunction, ok := prop.ValueOrNil.Data.(*js_ast.EFunction); ok {
		return c.convertClassMethod(prop, jsFunction, elemName)
	}

	return c.convertClassField(prop, elemName)
}

// getClassElementName extracts the name from a class element property.
//
// Takes prop (js_ast.Property) which contains the class element to process.
//
// Returns parsejs.ClassElementName which holds the extracted name.
func (c *ASTConverter) getClassElementName(prop js_ast.Property) parsejs.ClassElementName {
	if str, ok := prop.Key.Data.(*js_ast.EString); ok {
		strValue := helpers.UTF16ToString(str.Value)
		if prop.Kind == js_ast.PropertyMethod || prop.Kind == js_ast.PropertyGetter || prop.Kind == js_ast.PropertySetter {
			return parsejs.ClassElementName{
				PropertyName: parsejs.PropertyName{
					Literal: parsejs.LiteralExpr{
						TokenType: parsejs.IdentifierToken,
						Data:      []byte(strValue),
					},
				},
			}
		}
		return parsejs.ClassElementName{
			PropertyName: parsejs.PropertyName{
				Literal: parsejs.LiteralExpr{
					TokenType: parsejs.StringToken,
					Data:      fmt.Appendf(nil, fmtQuotedString, strValue),
				},
			},
		}
	}

	if identifier, ok := prop.Key.Data.(*js_ast.EIdentifier); ok {
		var name string
		if c.registry != nil {
			name = c.registry.LookupIdentifierName(identifier)
		}
		if name == "" {
			name = c.resolveRef(identifier.Ref)
		}
		if name == "" {
			name = "member"
		}
		return parsejs.ClassElementName{
			PropertyName: parsejs.PropertyName{
				Literal: parsejs.LiteralExpr{
					TokenType: parsejs.IdentifierToken,
					Data:      []byte(name),
				},
			},
		}
	}

	return parsejs.ClassElementName{}
}

// convertClassMethod converts a class method from AST property format.
//
// Takes prop (js_ast.Property) which holds the method flags and kind.
// Takes jsFunction (*js_ast.EFunction) which is the function to convert.
// Takes elemName (parsejs.ClassElementName) which is the method name.
//
// Returns *parsejs.ClassElement which wraps the converted method.
// Returns error when parameter or body conversion fails.
func (c *ASTConverter) convertClassMethod(prop js_ast.Property, jsFunction *js_ast.EFunction, elemName parsejs.ClassElementName) (*parsejs.ClassElement, error) {
	params, err := c.convertParams(jsFunction.Fn.Args)
	if err != nil {
		return nil, fmt.Errorf("converting class method parameters: %w", err)
	}

	body, err := c.convertFunctionBody(jsFunction.Fn.Body)
	if err != nil {
		return nil, fmt.Errorf("converting class method body: %w", err)
	}

	method := &parsejs.MethodDecl{
		Static: prop.Flags.Has(js_ast.PropertyIsStatic),
		Async:  jsFunction.Fn.IsAsync,
		Name:   elemName,
		Params: params,
		Body:   *body,
	}

	switch prop.Kind {
	case js_ast.PropertyGetter:
		method.Get = true
	case js_ast.PropertySetter:
		method.Set = true
	default:
	}

	return &parsejs.ClassElement{Method: method}, nil
}

// convertClassField converts a class field property to a class element.
//
// Takes prop (js_ast.Property) which contains the field property data.
// Takes elemName (parsejs.ClassElementName) which specifies the field name.
//
// Returns *parsejs.ClassElement which is the converted class field element.
// Returns error when the initialiser expression cannot be converted.
func (c *ASTConverter) convertClassField(prop js_ast.Property, elemName parsejs.ClassElementName) (*parsejs.ClassElement, error) {
	var init parsejs.IExpr
	var err error

	if prop.InitializerOrNil.Data != nil {
		init, err = c.convertExpression(prop.InitializerOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting class field initialiser: %w", err)
		}
	} else if prop.ValueOrNil.Data != nil {
		init, err = c.convertExpression(prop.ValueOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting class field value: %w", err)
		}
	}

	return &parsejs.ClassElement{
		Field: parsejs.Field{
			Static: prop.Flags.Has(js_ast.PropertyIsStatic),
			Name:   elemName,
			Init:   init,
		},
	}, nil
}

// Operator conversion using dispatch tables

var (
	// esbuildBinaryOpToTdewolff maps esbuild binary operators to tdewolff tokens.
	esbuildBinaryOpToTdewolff = map[js_ast.OpCode]parsejs.TokenType{
		js_ast.BinOpAdd:               parsejs.AddToken,
		js_ast.BinOpSub:               parsejs.SubToken,
		js_ast.BinOpMul:               parsejs.MulToken,
		js_ast.BinOpDiv:               parsejs.DivToken,
		js_ast.BinOpRem:               parsejs.ModToken,
		js_ast.BinOpPow:               parsejs.ExpToken,
		js_ast.BinOpStrictEq:          parsejs.EqEqEqToken,
		js_ast.BinOpStrictNe:          parsejs.NotEqEqToken,
		js_ast.BinOpLooseEq:           parsejs.EqEqToken,
		js_ast.BinOpLooseNe:           parsejs.NotEqToken,
		js_ast.BinOpLt:                parsejs.LtToken,
		js_ast.BinOpGt:                parsejs.GtToken,
		js_ast.BinOpLe:                parsejs.LtEqToken,
		js_ast.BinOpGe:                parsejs.GtEqToken,
		js_ast.BinOpLogicalAnd:        parsejs.AndToken,
		js_ast.BinOpLogicalOr:         parsejs.OrToken,
		js_ast.BinOpAssign:            parsejs.EqToken,
		js_ast.BinOpAddAssign:         parsejs.AddEqToken,
		js_ast.BinOpSubAssign:         parsejs.SubEqToken,
		js_ast.BinOpMulAssign:         parsejs.MulEqToken,
		js_ast.BinOpDivAssign:         parsejs.DivEqToken,
		js_ast.BinOpNullishCoalescing: parsejs.NullishToken,
		js_ast.BinOpBitwiseAnd:        parsejs.BitAndToken,
		js_ast.BinOpBitwiseOr:         parsejs.BitOrToken,
		js_ast.BinOpBitwiseXor:        parsejs.BitXorToken,
		js_ast.BinOpShl:               parsejs.LtLtToken,
		js_ast.BinOpShr:               parsejs.GtGtToken,
		js_ast.BinOpUShr:              parsejs.GtGtGtToken,
		js_ast.BinOpIn:                parsejs.InToken,
		js_ast.BinOpInstanceof:        parsejs.InstanceofToken,
	}

	// esbuildUnaryOpToTdewolff maps esbuild unary operators to tdewolff tokens.
	esbuildUnaryOpToTdewolff = map[js_ast.OpCode]parsejs.TokenType{
		js_ast.UnOpNeg:     parsejs.SubToken,
		js_ast.UnOpPos:     parsejs.AddToken,
		js_ast.UnOpNot:     parsejs.NotToken,
		js_ast.UnOpCpl:     parsejs.BitNotToken,
		js_ast.UnOpTypeof:  parsejs.TypeofToken,
		js_ast.UnOpVoid:    parsejs.VoidToken,
		js_ast.UnOpDelete:  parsejs.DeleteToken,
		js_ast.UnOpPreInc:  parsejs.PreIncrToken,
		js_ast.UnOpPreDec:  parsejs.PreDecrToken,
		js_ast.UnOpPostInc: parsejs.PostIncrToken,
		js_ast.UnOpPostDec: parsejs.PostDecrToken,
	}

	// esbuildOpPrecedence is the operator precedence table for esbuild operators
	// where higher values mean tighter binding, based on esbuild's js_ast.L*
	// constants.
	esbuildOpPrecedence = map[js_ast.OpCode]int{
		js_ast.BinOpAssign:            1,
		js_ast.BinOpAddAssign:         1,
		js_ast.BinOpSubAssign:         1,
		js_ast.BinOpMulAssign:         1,
		js_ast.BinOpDivAssign:         1,
		js_ast.BinOpNullishCoalescing: 2,
		js_ast.BinOpLogicalOr:         3,
		js_ast.BinOpLogicalAnd:        4,
		js_ast.BinOpBitwiseOr:         5,
		js_ast.BinOpBitwiseXor:        6,
		js_ast.BinOpBitwiseAnd:        7,
		js_ast.BinOpLooseEq:           8,
		js_ast.BinOpLooseNe:           8,
		js_ast.BinOpStrictEq:          8,
		js_ast.BinOpStrictNe:          8,
		js_ast.BinOpLt:                9,
		js_ast.BinOpLe:                9,
		js_ast.BinOpGt:                9,
		js_ast.BinOpGe:                9,
		js_ast.BinOpIn:                9,
		js_ast.BinOpInstanceof:        9,
		js_ast.BinOpShl:               10,
		js_ast.BinOpShr:               10,
		js_ast.BinOpUShr:              10,
		js_ast.BinOpAdd:               11,
		js_ast.BinOpSub:               11,
		js_ast.BinOpMul:               12,
		js_ast.BinOpDiv:               12,
		js_ast.BinOpRem:               12,
		js_ast.BinOpPow:               13,
	}
)

// convertBinaryOp converts an esbuild binary operator to a tdewolff token.
//
// Takes op (js_ast.OpCode) which specifies the esbuild binary operator.
//
// Returns parsejs.TokenType which is the matching tdewolff token, or AddToken
// if no mapping exists.
func convertBinaryOp(op js_ast.OpCode) parsejs.TokenType {
	if token, ok := esbuildBinaryOpToTdewolff[op]; ok {
		return token
	}
	return parsejs.AddToken
}

// convertUnaryOp converts an esbuild unary operator to a tdewolff token type.
//
// Takes op (js_ast.OpCode) which specifies the esbuild unary operator to
// convert.
//
// Returns parsejs.TokenType which is the matching tdewolff token, or
// parsejs.NotToken if no mapping exists.
func convertUnaryOp(op js_ast.OpCode) parsejs.TokenType {
	if token, ok := esbuildUnaryOpToTdewolff[op]; ok {
		return token
	}
	return parsejs.NotToken
}

// getOpPrecedence returns the precedence level for an esbuild operator.
//
// Takes op (js_ast.OpCode) which is the operator code to look up.
//
// Returns int which is the precedence level, or 0 if the operator is unknown.
func getOpPrecedence(op js_ast.OpCode) int {
	if prec, ok := esbuildOpPrecedence[op]; ok {
		return prec
	}
	return 0
}
