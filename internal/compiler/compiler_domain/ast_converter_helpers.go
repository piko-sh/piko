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
	"math"

	parsejs "github.com/tdewolff/parse/v2/js"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// convertBinding converts an esbuild binding to a tdewolff binding.
//
// An array hole is only meaningful inside an array pattern, so convertBArray handles it
// before it reaches here.
//
// Takes binding (js_ast.Binding) which is the esbuild binding to convert.
//
// Returns parsejs.IBinding which is the converted tdewolff binding, or nil when binding
// has no data.
// Returns error when the binding kind cannot be printed, so it fails the build instead of
// declaring a placeholder name.
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
		return nil, fmt.Errorf("binding %T: %w", binding.Data, errUnsupportedExpression)
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
// A hole such as the first slot of `[, b]` becomes an elided element, and when the
// pattern ends in a rest element such as `...others` the last item becomes the pattern's
// rest binding.
//
// Takes b (*js_ast.BArray) which is the array binding pattern to convert.
//
// Returns parsejs.IBinding which is the converted array binding.
// Returns error when any element binding or default expression fails to convert, or the
// rest element is missing or has a default.
func (c *ASTConverter) convertBArray(b *js_ast.BArray) (parsejs.IBinding, error) {
	items := b.Items
	var rest parsejs.IBinding
	if b.HasSpread {
		if len(items) == 0 {
			return nil, fmt.Errorf("array binding rest element: %w", errUnsupportedExpression)
		}
		var err error
		rest, err = c.convertBArrayRest(items[len(items)-1])
		if err != nil {
			return nil, err
		}
		items = items[:len(items)-1]
	}

	elements := make([]parsejs.BindingElement, 0, len(items))
	for _, item := range items {
		element, err := c.convertArrayBindingElement(item)
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)
	}
	return &parsejs.BindingArray{List: elements, Rest: rest}, nil
}

// convertArrayBindingElement converts one non-rest slot of an array destructuring
// pattern.
//
// Takes item (js_ast.ArrayBinding) which is the slot to convert.
//
// Returns parsejs.BindingElement which is the converted slot, with a nil binding for a
// hole.
// Returns error when the binding or its default expression fails to convert.
func (c *ASTConverter) convertArrayBindingElement(item js_ast.ArrayBinding) (parsejs.BindingElement, error) {
	if _, isHole := item.Binding.Data.(*js_ast.BMissing); isHole {
		return parsejs.BindingElement{}, nil
	}

	converted, err := c.convertBinding(item.Binding)
	if err != nil {
		return parsejs.BindingElement{}, fmt.Errorf("converting array binding element: %w", err)
	}

	var defaultExpr parsejs.IExpr
	if item.DefaultValueOrNil.Data != nil {
		defaultExpr, err = c.convertExpression(item.DefaultValueOrNil)
		if err != nil {
			return parsejs.BindingElement{}, fmt.Errorf("converting array binding default value: %w", err)
		}
	}

	return parsejs.BindingElement{Binding: converted, Default: defaultExpr}, nil
}

// convertBArrayRest converts the rest element of an array destructuring pattern.
//
// Takes item (js_ast.ArrayBinding) which is the pattern's last item.
//
// Returns parsejs.IBinding which is the binding that receives the remaining elements; it
// may itself be a pattern, as in `[a, ...[b, c]]`.
// Returns error when the rest target is a hole, carries a default, or fails to convert.
func (c *ASTConverter) convertBArrayRest(item js_ast.ArrayBinding) (parsejs.IBinding, error) {
	if item.DefaultValueOrNil.Data != nil {
		return nil, fmt.Errorf("array binding rest element with a default: %w", errUnsupportedExpression)
	}
	if _, isHole := item.Binding.Data.(*js_ast.BMissing); isHole || item.Binding.Data == nil {
		return nil, fmt.Errorf("array binding rest element without a target: %w", errUnsupportedExpression)
	}
	rest, err := c.convertBinding(item.Binding)
	if err != nil {
		return nil, fmt.Errorf("converting array binding rest element: %w", err)
	}
	return rest, nil
}

// convertBObject converts an object destructuring binding.
//
// Takes b (*js_ast.BObject) which is the object binding to convert.
//
// Returns parsejs.IBinding which is the converted binding object.
// Returns error when a property value or default expression fails to convert.
func (c *ASTConverter) convertBObject(b *js_ast.BObject) (parsejs.IBinding, error) {
	props := make([]parsejs.BindingObjectItem, 0, len(b.Properties))
	var rest *parsejs.Var
	for _, prop := range b.Properties {
		if prop.IsSpread {
			var err error
			rest, err = c.convertBObjectRest(prop)
			if err != nil {
				return nil, err
			}
			continue
		}

		key, err := c.convertBindingKey(prop)
		if err != nil {
			return nil, err
		}

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
	return &parsejs.BindingObject{List: props, Rest: rest}, nil
}

// convertBObjectRest converts the rest element of an object destructuring pattern.
//
// Takes prop (js_ast.PropertyBinding) which is the spread property holding the rest
// target.
//
// Returns *parsejs.Var which names the variable that receives the remaining properties.
// Returns error when the rest target is not a plain identifier, which is the only form
// JavaScript allows in an object pattern.
func (c *ASTConverter) convertBObjectRest(prop js_ast.PropertyBinding) (*parsejs.Var, error) {
	binding, err := c.convertBinding(prop.Value)
	if err != nil {
		return nil, fmt.Errorf("converting object binding rest: %w", err)
	}
	rest, ok := binding.(*parsejs.Var)
	if !ok {
		return nil, fmt.Errorf("object binding rest %T: %w", binding, errUnsupportedExpression)
	}
	return rest, nil
}

// convertBindingKey converts the key of one object destructuring property.
//
// A computed key keeps its brackets so the binding reads the property the expression
// names, not a property spelled like the expression.
//
// Takes prop (js_ast.PropertyBinding) which is the destructuring property.
//
// Returns *parsejs.PropertyName which is the converted key.
// Returns error when the key cannot be converted.
func (c *ASTConverter) convertBindingKey(prop js_ast.PropertyBinding) (*parsejs.PropertyName, error) {
	if prop.IsComputed {
		return c.computedPropertyName(prop.Key)
	}
	return c.convertBindingPropertyKey(prop.Key)
}

// convertBindingPropertyKey converts a non-computed destructuring key into a
// PropertyName.
//
// Takes key (js_ast.Expr) which is the AST expression for the property key.
//
// Returns *parsejs.PropertyName which is the converted property name.
// Returns error when the key is not an identifier, string or number, so an unsupported
// key fails the build instead of printing as a shorthand binding.
func (c *ASTConverter) convertBindingPropertyKey(key js_ast.Expr) (*parsejs.PropertyName, error) {
	switch k := key.Data.(type) {
	case *js_ast.EIdentifier:
		name := c.resolveRef(k.Ref)
		if name == "" {
			name = "key"
		}
		return &parsejs.PropertyName{
			Literal: parsejs.LiteralExpr{TokenType: parsejs.IdentifierToken, Data: []byte(name)},
		}, nil
	case *js_ast.EString:
		return &parsejs.PropertyName{
			Literal: parsejs.LiteralExpr{TokenType: parsejs.StringToken, Data: helpers.QuoteForJSON(helpers.UTF16ToString(k.Value), false)},
		}, nil
	case *js_ast.ENumber:
		return numericPropertyName(k.Value), nil
	default:
		return nil, fmt.Errorf("object binding key %T: %w", key.Data, errUnsupportedExpression)
	}
}

// convertParams converts function arguments to parameter bindings.
//
// Takes arguments ([]js_ast.Arg) which contains the function arguments to convert.
// Takes hasRestArg (bool) which is true when the last argument is a rest parameter such
// as `...values`, as recorded on the function or arrow.
//
// Returns parsejs.Params which contains the converted binding elements and, when
// hasRestArg is set, the rest parameter.
// Returns error when a binding or default value cannot be converted, or the rest
// parameter is missing or has a default.
func (c *ASTConverter) convertParams(arguments []js_ast.Arg, hasRestArg bool) (parsejs.Params, error) {
	var rest parsejs.IBinding
	if hasRestArg {
		if len(arguments) == 0 {
			return parsejs.Params{}, fmt.Errorf("rest parameter: %w", errUnsupportedExpression)
		}
		var err error
		rest, err = c.convertRestParam(arguments[len(arguments)-1])
		if err != nil {
			return parsejs.Params{}, err
		}
		arguments = arguments[:len(arguments)-1]
	}

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

	return parsejs.Params{List: elements, Rest: rest}, nil
}

// convertRestParam converts a function's rest parameter.
//
// Takes argument (js_ast.Arg) which is the function's last argument.
//
// Returns parsejs.IBinding which is the binding that receives the remaining arguments.
// Returns error when the rest parameter has a default or no binding, or its binding fails
// to convert.
func (c *ASTConverter) convertRestParam(argument js_ast.Arg) (parsejs.IBinding, error) {
	if argument.DefaultOrNil.Data != nil {
		return nil, fmt.Errorf("rest parameter with a default: %w", errUnsupportedExpression)
	}
	rest, err := c.convertBinding(argument.Binding)
	if err != nil {
		return nil, fmt.Errorf("converting rest parameter binding: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("rest parameter without a binding: %w", errUnsupportedExpression)
	}
	return rest, nil
}

// convertFunctionBody converts a function body to a block statement.
//
// Takes body (js_ast.FnBody) which contains the function body to convert.
//
// Returns *parsejs.BlockStmt which contains the converted statements.
// Returns error when a statement conversion fails.
func (c *ASTConverter) convertFunctionBody(body js_ast.FnBody) (*parsejs.BlockStmt, error) {
	statements, err := c.convertStatementList(body.Block.Stmts)
	if err != nil {
		return nil, fmt.Errorf("converting function body: %w", err)
	}
	return &parsejs.BlockStmt{List: statements}, nil
}

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

	var name *parsejs.PropertyName
	var err error
	if prop.Flags.Has(js_ast.PropertyIsComputed) {
		name, err = c.computedPropertyName(prop.Key)
	} else {
		name, err = c.convertPropertyName(prop.Key)
	}
	if err != nil {
		return nil, fmt.Errorf("converting property name: %w", err)
	}

	if accessor, ok, err := c.tryConvertAccessorProperty(prop, name); err != nil {
		return nil, err
	} else if ok {
		return accessor, nil
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

// tryConvertAccessorProperty converts a getter, setter or concise method of an object
// literal into a method declaration.
//
// Takes prop (js_ast.Property) which is the property to convert.
// Takes name (*parsejs.PropertyName) which is the already-converted key.
//
// Returns *parsejs.Property which holds the method.
// Returns bool which is true when the property was an accessor or a method.
// Returns error when the function body cannot be converted.
func (c *ASTConverter) tryConvertAccessorProperty(
	prop js_ast.Property,
	name *parsejs.PropertyName,
) (*parsejs.Property, bool, error) {
	if prop.Kind != js_ast.PropertyGetter && prop.Kind != js_ast.PropertySetter && prop.Kind != js_ast.PropertyMethod {
		return nil, false, nil
	}
	jsFunction, ok := prop.ValueOrNil.Data.(*js_ast.EFunction)
	if !ok {
		return nil, false, nil
	}

	params, err := c.convertParams(jsFunction.Fn.Args, jsFunction.Fn.HasRestArg)
	if err != nil {
		return nil, false, fmt.Errorf("converting object method parameters: %w", err)
	}
	body, err := c.convertFunctionBody(jsFunction.Fn.Body)
	if err != nil {
		return nil, false, fmt.Errorf("converting object method body: %w", err)
	}

	method := &parsejs.MethodDecl{
		Async:     jsFunction.Fn.IsAsync,
		Generator: jsFunction.Fn.IsGenerator,
		Get:       prop.Kind == js_ast.PropertyGetter,
		Set:       prop.Kind == js_ast.PropertySetter,
		Params:    params,
		Body:      *body,
	}
	if name != nil {
		method.Name = parsejs.ClassElementName{PropertyName: *name}
	}

	return &parsejs.Property{Value: method}, true, nil
}

// tryConvertShorthandProperty tries to convert a shorthand property.
//
// Takes prop (js_ast.Property) which is the property to convert.
//
// Returns *parsejs.Property which is the converted property, or nil if the property is
// not a shorthand property or cannot be converted.
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
				Data:      helpers.QuoteForJSON(helpers.UTF16ToString(k.Value), false),
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
		return numericPropertyName(k.Value), nil

	default:
		return c.computedPropertyName(key)
	}
}

// computedPropertyName converts a key expression into the bracketed computed-name slot.
//
// Takes key (js_ast.Expr) which is the key expression written inside the brackets.
//
// Returns *parsejs.PropertyName which holds the converted key expression.
// Returns error when the key expression cannot be converted.
func (c *ASTConverter) computedPropertyName(key js_ast.Expr) (*parsejs.PropertyName, error) {
	computed, err := c.convertExpression(key)
	if err != nil {
		return nil, fmt.Errorf("converting computed property name: %w", err)
	}
	if computed == nil {
		return nil, fmt.Errorf("computed property name: %w", errUnsupportedExpression)
	}
	return &parsejs.PropertyName{Computed: computed}, nil
}

// numericPropertyName converts a numeric literal key into a PropertyName.
//
// Takes value (float64) which is the key's numeric value.
//
// Returns *parsejs.PropertyName which prints the number as written, or as the non-finite
// key literal for NaN and the infinities.
func numericPropertyName(value float64) *parsejs.PropertyName {
	if lit, ok := nonFiniteKeyLiteral(value); ok {
		return &parsejs.PropertyName{Literal: lit}
	}
	return &parsejs.PropertyName{
		Literal: parsejs.LiteralExpr{
			TokenType: parsejs.DecimalToken,
			Data:      fmt.Appendf(nil, "%g", value),
		},
	}
}

// convertClassProperty converts a class property or method to a ClassElement.
//
// Takes prop (js_ast.Property) which is the property to convert.
//
// Returns *parsejs.ClassElement which is the converted element.
// Returns error when the conversion fails.
func (c *ASTConverter) convertClassProperty(prop js_ast.Property) (*parsejs.ClassElement, error) {
	if prop.Kind == js_ast.PropertyClassStaticBlock {
		return c.convertClassStaticBlock(prop)
	}

	elemName, err := c.getClassElementName(prop)
	if err != nil {
		return nil, fmt.Errorf("converting class element name: %w", err)
	}

	if jsFunction, ok := prop.ValueOrNil.Data.(*js_ast.EFunction); ok {
		return c.convertClassMethod(prop, jsFunction, elemName)
	}

	return c.convertClassField(prop, elemName)
}

// convertClassStaticBlock converts a class static initialisation block.
//
// Takes prop (js_ast.Property) which carries the static block.
//
// Returns *parsejs.ClassElement which holds the block.
// Returns error when a statement in the block cannot be converted.
func (c *ASTConverter) convertClassStaticBlock(prop js_ast.Property) (*parsejs.ClassElement, error) {
	block := &parsejs.BlockStmt{}
	if prop.ClassStaticBlock == nil {
		return &parsejs.ClassElement{StaticBlock: block}, nil
	}

	statements, err := c.convertStatementList(prop.ClassStaticBlock.Block.Stmts)
	if err != nil {
		return nil, fmt.Errorf("converting class static block: %w", err)
	}
	block.List = statements
	return &parsejs.ClassElement{StaticBlock: block}, nil
}

// getClassElementName extracts the name from a class element property.
//
// Takes prop (js_ast.Property) which contains the class element to process.
//
// Returns parsejs.ClassElementName which holds the extracted name.
// Returns error when the key cannot be represented, so a dropped name surfaces instead of
// emitting a class body with a blank member.
func (c *ASTConverter) getClassElementName(prop js_ast.Property) (parsejs.ClassElementName, error) {
	if prop.Key.Data == nil {
		return parsejs.ClassElementName{}, nil
	}

	if prop.Flags.Has(js_ast.PropertyIsComputed) {
		return c.computedClassElementName(prop)
	}

	if str, ok := prop.Key.Data.(*js_ast.EString); ok {
		strValue := helpers.UTF16ToString(str.Value)
		isMember := prop.Kind == js_ast.PropertyMethod || prop.Kind == js_ast.PropertyGetter ||
			prop.Kind == js_ast.PropertySetter
		if isMember && js_ast.IsIdentifier(strValue) {
			return parsejs.ClassElementName{
				Literal: parsejs.LiteralExpr{
					TokenType: parsejs.IdentifierToken,
					Data:      []byte(strValue),
				},
			}, nil
		}
		return parsejs.ClassElementName{
			Literal: parsejs.LiteralExpr{
				TokenType: parsejs.StringToken,
				Data:      helpers.QuoteForJSON(strValue, false),
			},
		}, nil
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
			Literal: parsejs.LiteralExpr{
				TokenType: parsejs.IdentifierToken,
				Data:      []byte(name),
			},
		}, nil
	}

	return c.computedClassElementName(prop)
}

// computedClassElementName converts a class member key into the computed-name slot.
//
// Takes prop (js_ast.Property) which owns the key.
//
// Returns parsejs.ClassElementName which holds the converted key expression.
// Returns error when the key expression cannot be converted.
func (c *ASTConverter) computedClassElementName(prop js_ast.Property) (parsejs.ClassElementName, error) {
	key, err := c.convertExpression(prop.Key)
	if err != nil {
		return parsejs.ClassElementName{}, fmt.Errorf("converting computed class member key: %w", err)
	}
	if key == nil {
		return parsejs.ClassElementName{}, fmt.Errorf("computed class member key: %w", errUnsupportedExpression)
	}
	return parsejs.ClassElementName{
		Computed: key,
	}, nil
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
	params, err := c.convertParams(jsFunction.Fn.Args, jsFunction.Fn.HasRestArg)
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
		Static: prop.Flags.Has(js_ast.PropertyIsStatic),
		Name:   elemName,
		Init:   init,
	}, nil
}

// Operator conversion using dispatch tables

var (
	// esbuildBinaryOpToTdewolff maps esbuild binary operators to tdewolff tokens.
	esbuildBinaryOpToTdewolff = map[js_ast.OpCode]parsejs.TokenType{ //nolint:exhaustive // unary OpCodes handled by esbuildUnaryOpToTdewolff
		js_ast.BinOpAdd:                     parsejs.AddToken,
		js_ast.BinOpSub:                     parsejs.SubToken,
		js_ast.BinOpMul:                     parsejs.MulToken,
		js_ast.BinOpDiv:                     parsejs.DivToken,
		js_ast.BinOpRem:                     parsejs.ModToken,
		js_ast.BinOpPow:                     parsejs.ExpToken,
		js_ast.BinOpStrictEq:                parsejs.EqEqEqToken,
		js_ast.BinOpStrictNe:                parsejs.NotEqEqToken,
		js_ast.BinOpLooseEq:                 parsejs.EqEqToken,
		js_ast.BinOpLooseNe:                 parsejs.NotEqToken,
		js_ast.BinOpLt:                      parsejs.LtToken,
		js_ast.BinOpGt:                      parsejs.GtToken,
		js_ast.BinOpLe:                      parsejs.LtEqToken,
		js_ast.BinOpGe:                      parsejs.GtEqToken,
		js_ast.BinOpLogicalAnd:              parsejs.AndToken,
		js_ast.BinOpLogicalOr:               parsejs.OrToken,
		js_ast.BinOpAssign:                  parsejs.EqToken,
		js_ast.BinOpAddAssign:               parsejs.AddEqToken,
		js_ast.BinOpSubAssign:               parsejs.SubEqToken,
		js_ast.BinOpMulAssign:               parsejs.MulEqToken,
		js_ast.BinOpDivAssign:               parsejs.DivEqToken,
		js_ast.BinOpRemAssign:               parsejs.ModEqToken,
		js_ast.BinOpPowAssign:               parsejs.ExpEqToken,
		js_ast.BinOpShlAssign:               parsejs.LtLtEqToken,
		js_ast.BinOpShrAssign:               parsejs.GtGtEqToken,
		js_ast.BinOpUShrAssign:              parsejs.GtGtGtEqToken,
		js_ast.BinOpBitwiseOrAssign:         parsejs.BitOrEqToken,
		js_ast.BinOpBitwiseAndAssign:        parsejs.BitAndEqToken,
		js_ast.BinOpBitwiseXorAssign:        parsejs.BitXorEqToken,
		js_ast.BinOpNullishCoalescingAssign: parsejs.NullishEqToken,
		js_ast.BinOpLogicalOrAssign:         parsejs.OrEqToken,
		js_ast.BinOpLogicalAndAssign:        parsejs.AndEqToken,
		js_ast.BinOpNullishCoalescing:       parsejs.NullishToken,
		js_ast.BinOpBitwiseAnd:              parsejs.BitAndToken,
		js_ast.BinOpBitwiseOr:               parsejs.BitOrToken,
		js_ast.BinOpBitwiseXor:              parsejs.BitXorToken,
		js_ast.BinOpShl:                     parsejs.LtLtToken,
		js_ast.BinOpShr:                     parsejs.GtGtToken,
		js_ast.BinOpUShr:                    parsejs.GtGtGtToken,
		js_ast.BinOpIn:                      parsejs.InToken,
		js_ast.BinOpInstanceof:              parsejs.InstanceofToken,
		js_ast.BinOpComma:                   parsejs.CommaToken,
	}

	// esbuildUnaryOpToTdewolff maps esbuild unary operators to tdewolff tokens.
	esbuildUnaryOpToTdewolff = map[js_ast.OpCode]parsejs.TokenType{ //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are errors in convertUnaryOp
		js_ast.UnOpNeg:     parsejs.NegToken,
		js_ast.UnOpPos:     parsejs.PosToken,
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
)

// convertBinaryOp converts an esbuild binary operator to a tdewolff token.
//
// Takes op (js_ast.OpCode) which specifies the esbuild binary operator.
//
// Returns parsejs.TokenType which is the matching tdewolff token.
// Returns error when the operator is not in the mapping, since silently returning a
// default token would miscompile user source.
func convertBinaryOp(op js_ast.OpCode) (parsejs.TokenType, error) {
	if token, ok := esbuildBinaryOpToTdewolff[op]; ok {
		return token, nil
	}
	return 0, fmt.Errorf("unsupported JS binary operator %v", op)
}

// convertUnaryOp converts an esbuild unary operator to a tdewolff token type.
//
// Takes op (js_ast.OpCode) which specifies the esbuild unary operator to convert.
//
// Returns parsejs.TokenType which is the matching tdewolff token.
// Returns error when no mapping exists for the operator.
func convertUnaryOp(op js_ast.OpCode) (parsejs.TokenType, error) {
	if token, ok := esbuildUnaryOpToTdewolff[op]; ok {
		return token, nil
	}
	return 0, fmt.Errorf("unsupported JS unary operator %v", op)
}

// nonFiniteKeyLiteral returns the JavaScript literal for a non-finite numeric object key.
//
// +Inf and NaN become the Infinity/NaN identifiers; -Inf becomes the string key a
// -Infinity numeric key coerces to.
//
// Takes value (float64) which is the constant-folded numeric key.
//
// Returns parsejs.LiteralExpr which is the key literal when the bool is true.
// Returns bool which is true for a non-finite value and false for a finite one.
func nonFiniteKeyLiteral(value float64) (parsejs.LiteralExpr, bool) {
	switch {
	case math.IsInf(value, 1):
		return parsejs.LiteralExpr{TokenType: parsejs.IdentifierToken, Data: []byte("Infinity")}, true
	case math.IsInf(value, -1):
		return parsejs.LiteralExpr{TokenType: parsejs.StringToken, Data: []byte(`"-Infinity"`)}, true
	case math.IsNaN(value):
		return parsejs.LiteralExpr{TokenType: parsejs.IdentifierToken, Data: []byte("NaN")}, true
	default:
		return parsejs.LiteralExpr{}, false
	}
}
