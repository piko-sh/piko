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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	parsejs "github.com/tdewolff/parse/v2/js"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/logger"
)

func TestConvertBinding(t *testing.T) {
	t.Parallel()

	t.Run("nil binding data returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		result, err := converter.convertBinding(js_ast.Binding{Data: nil})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("BIdentifier binding returns Var", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("x")
		converter := NewASTConverter(nil, nil, registry)

		result, err := converter.convertBinding(binding)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "x", string(v.Data))
	})

	t.Run("unknown binding type returns fallback var", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		result, err := converter.convertBinding(js_ast.Binding{Data: &js_ast.BMissing{}})
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "binding", string(v.Data))
	})
}

func TestConvertBIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("resolves from registry", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		bind := &js_ast.BIdentifier{Ref: ast.Ref{}}
		registry.RegisterBindingName(bind, "registeredName")
		converter := NewASTConverter(nil, nil, registry)

		result, err := converter.convertBIdentifier(bind)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "registeredName", string(v.Data))
	})

	t.Run("resolves from symbols fallback", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "symBinding"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		bind := &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 0}}

		result, err := converter.convertBIdentifier(bind)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "symBinding", string(v.Data))
	})

	t.Run("falls back to binding when no name found", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		bind := &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 999}}

		result, err := converter.convertBIdentifier(bind)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "binding", string(v.Data))
	})
}

func TestConvertBArray(t *testing.T) {
	t.Parallel()

	t.Run("array binding with default values", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding1 := registry.MakeBinding("a")
		binding2 := registry.MakeBinding("b")
		converter := NewASTConverter(nil, nil, registry)

		b := &js_ast.BArray{
			Items: []js_ast.ArrayBinding{
				{
					Binding:           binding1,
					DefaultValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
				},
				{
					Binding:           binding2,
					DefaultValueOrNil: js_ast.Expr{Data: nil},
				},
			},
		}

		result, err := converter.convertBArray(b)
		require.NoError(t, err)
		require.NotNil(t, result)

		bindingArray, ok := result.(*parsejs.BindingArray)
		require.True(t, ok)
		assert.Len(t, bindingArray.List, 2)
		require.NotNil(t, bindingArray.List[0].Default)
		assert.Nil(t, bindingArray.List[1].Default)
	})

	t.Run("empty array binding", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		b := &js_ast.BArray{Items: []js_ast.ArrayBinding{}}

		result, err := converter.convertBArray(b)
		require.NoError(t, err)
		require.NotNil(t, result)

		bindingArray, ok := result.(*parsejs.BindingArray)
		require.True(t, ok)
		assert.Empty(t, bindingArray.List)
	})
}

func TestConvertBObject(t *testing.T) {
	t.Parallel()

	t.Run("object binding with default values", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		valueBinding := registry.MakeBinding("val")
		keyIdent := registry.MakeIdentifier("key")
		converter := NewASTConverter(nil, nil, registry)

		b := &js_ast.BObject{
			Properties: []js_ast.PropertyBinding{
				{
					Key:               js_ast.Expr{Data: keyIdent},
					Value:             valueBinding,
					DefaultValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 10}},
				},
			},
			IsSingleLine: false,
		}

		result, err := converter.convertBObject(b)
		require.NoError(t, err)
		require.NotNil(t, result)

		bindingObj, ok := result.(*parsejs.BindingObject)
		require.True(t, ok)
		assert.Len(t, bindingObj.List, 1)
		require.NotNil(t, bindingObj.List[0].Value.Default)
	})

	t.Run("object binding without default", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		valueBinding := registry.MakeBinding("val")
		keyIdent := registry.MakeIdentifier("key")
		converter := NewASTConverter(nil, nil, registry)

		b := &js_ast.BObject{
			Properties: []js_ast.PropertyBinding{
				{
					Key:               js_ast.Expr{Data: keyIdent},
					Value:             valueBinding,
					DefaultValueOrNil: js_ast.Expr{Data: nil},
				},
			},
			IsSingleLine: false,
		}

		result, err := converter.convertBObject(b)
		require.NoError(t, err)
		require.NotNil(t, result)

		bindingObj, ok := result.(*parsejs.BindingObject)
		require.True(t, ok)
		assert.Len(t, bindingObj.List, 1)
		assert.Nil(t, bindingObj.List[0].Value.Default)
	})
}

func TestConvertBindingPropertyKey(t *testing.T) {
	t.Parallel()

	t.Run("identifier key", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "myKey"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		key := js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 0}}}

		result := converter.convertBindingPropertyKey(key)
		require.NotNil(t, result)
		assert.Equal(t, parsejs.IdentifierToken, result.Literal.TokenType)
		assert.Equal(t, "myKey", string(result.Literal.Data))
	})

	t.Run("identifier key with empty name falls back to key", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		key := js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 999}}}

		result := converter.convertBindingPropertyKey(key)
		require.NotNil(t, result)
		assert.Equal(t, "key", string(result.Literal.Data))
	})

	t.Run("string key", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		key := js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'n', 'a', 'm', 'e'}}}

		result := converter.convertBindingPropertyKey(key)
		require.NotNil(t, result)
		assert.Equal(t, parsejs.StringToken, result.Literal.TokenType)
	})

	t.Run("unsupported key type returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		key := js_ast.Expr{Data: &js_ast.ENumber{Value: 42}}

		result := converter.convertBindingPropertyKey(key)
		assert.Nil(t, result)
	})
}

func TestConvertParams(t *testing.T) {
	t.Parallel()

	t.Run("params with default value", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("x")
		converter := NewASTConverter(nil, nil, registry)

		arguments := []js_ast.Arg{
			{
				Binding:      binding,
				DefaultOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 10}},
				Decorators:   nil,
			},
		}

		result, err := converter.convertParams(arguments)
		require.NoError(t, err)
		assert.Len(t, result.List, 1)
		require.NotNil(t, result.List[0].Default)
	})

	t.Run("params without default value", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("y")
		converter := NewASTConverter(nil, nil, registry)

		arguments := []js_ast.Arg{
			{
				Binding:      binding,
				DefaultOrNil: js_ast.Expr{Data: nil},
				Decorators:   nil,
			},
		}

		result, err := converter.convertParams(arguments)
		require.NoError(t, err)
		assert.Len(t, result.List, 1)
		assert.Nil(t, result.List[0].Default)
	})

	t.Run("empty params", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		result, err := converter.convertParams([]js_ast.Arg{})
		require.NoError(t, err)
		assert.Empty(t, result.List)
	})
}

func TestConvertProperty(t *testing.T) {
	t.Parallel()

	t.Run("spread property", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		valIdent := registry.MakeIdentifier("other")
		converter := NewASTConverter(nil, nil, registry)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: nil},
			ValueOrNil:       js_ast.Expr{Data: valIdent},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertySpread,
			Flags:            0,
		}

		result, err := converter.convertProperty(prop)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Spread)
	})

	t.Run("shorthand property", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "x"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		identifier := &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 0}}
		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: identifier},
			ValueOrNil:       js_ast.Expr{Data: identifier},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            js_ast.PropertyWasShorthand,
		}

		result, err := converter.convertProperty(prop)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Name)
		assert.Equal(t, "x", string(result.Name.Literal.Data))
	})

	t.Run("regular property with name and value", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "myProp"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 0}}},
			ValueOrNil:       js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}

		result, err := converter.convertProperty(prop)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Name)
		require.NotNil(t, result.Value)
	})

	t.Run("property with nil value", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "myProp"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 0}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}

		result, err := converter.convertProperty(prop)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.Value)
	})
}

func TestTryConvertShorthandProperty(t *testing.T) {
	t.Parallel()

	t.Run("non-shorthand property returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 0}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}

		result := converter.tryConvertShorthandProperty(prop)
		assert.Nil(t, result)
	})

	t.Run("shorthand with non-identifier key returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            js_ast.PropertyWasShorthand,
		}

		result := converter.tryConvertShorthandProperty(prop)
		assert.Nil(t, result)
	})

	t.Run("shorthand with empty name uses prop fallback", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 999}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            js_ast.PropertyWasShorthand,
		}

		result := converter.tryConvertShorthandProperty(prop)
		require.NotNil(t, result)
		assert.Equal(t, "prop", string(result.Name.Literal.Data))
		assert.Equal(t, "prop", string(result.Value.(*parsejs.Var).Data))
	})
}

func TestConvertPropertyName(t *testing.T) {
	t.Parallel()

	t.Run("string key", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		key := js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'h', 'i'}}}

		result, err := converter.convertPropertyName(key)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, parsejs.StringToken, result.Literal.TokenType)
	})

	t.Run("identifier key", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "myProp"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		key := js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 0}}}

		result, err := converter.convertPropertyName(key)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, parsejs.IdentifierToken, result.Literal.TokenType)
		assert.Equal(t, "myProp", string(result.Literal.Data))
	})

	t.Run("identifier key with empty name falls back to property", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		key := js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 999}}}

		result, err := converter.convertPropertyName(key)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "property", string(result.Literal.Data))
	})

	t.Run("number key", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		key := js_ast.Expr{Data: &js_ast.ENumber{Value: 42}}

		result, err := converter.convertPropertyName(key)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, parsejs.DecimalToken, result.Literal.TokenType)
		assert.Equal(t, "42", string(result.Literal.Data))
	})

	t.Run("computed key", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		key := js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}

		result, err := converter.convertPropertyName(key)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Computed)
	})
}

func TestGetClassElementName(t *testing.T) {
	t.Parallel()

	t.Run("string key for method uses identifier token", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'r', 'u', 'n'}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyMethod,
			Flags:            0,
		}

		result := converter.getClassElementName(prop)
		assert.Equal(t, parsejs.IdentifierToken, result.Literal.TokenType)
		assert.Equal(t, "run", string(result.Literal.Data))
	})

	t.Run("string key for getter uses identifier token", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'v', 'a', 'l'}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyGetter,
			Flags:            0,
		}

		result := converter.getClassElementName(prop)
		assert.Equal(t, parsejs.IdentifierToken, result.Literal.TokenType)
	})

	t.Run("string key for field uses string token", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'f', 'l', 'd'}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}

		result := converter.getClassElementName(prop)
		assert.Equal(t, parsejs.StringToken, result.Literal.TokenType)
	})

	t.Run("identifier key with registry lookup", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		identifier := registry.MakeIdentifier("memberName")
		converter := NewASTConverter(nil, nil, registry)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: identifier},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}

		result := converter.getClassElementName(prop)
		assert.Equal(t, parsejs.IdentifierToken, result.Literal.TokenType)
		assert.Equal(t, "memberName", string(result.Literal.Data))
	})

	t.Run("identifier key falls back to member", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 999}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}

		result := converter.getClassElementName(prop)
		assert.Equal(t, "member", string(result.Literal.Data))
	})

	t.Run("unknown key type returns empty ClassElementName", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}

		result := converter.getClassElementName(prop)
		assert.Empty(t, result.Literal.Data)
	})
}

func TestConvertClassMethod(t *testing.T) {
	t.Parallel()

	t.Run("getter method", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		jsFunction := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Name:         nil,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{Start: 0}}, Loc: logger.Loc{Start: 0}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{Start: 0},
				IsAsync:      false,
				IsGenerator:  false,
			},
		}

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'v', 'a', 'l'}}},
			ValueOrNil:       js_ast.Expr{Data: jsFunction},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyGetter,
			Flags:            0,
		}
		elemName := converter.getClassElementName(prop)

		result, err := converter.convertClassMethod(prop, jsFunction, elemName)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Method)
		assert.True(t, result.Method.Get)
		assert.False(t, result.Method.Set)
	})

	t.Run("setter method", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("newVal")
		converter := NewASTConverter(nil, nil, registry)

		jsFunction := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Name: nil,
				Args: []js_ast.Arg{
					{
						Binding:      binding,
						DefaultOrNil: js_ast.Expr{Data: nil},
						Decorators:   nil,
					},
				},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{Start: 0}}, Loc: logger.Loc{Start: 0}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{Start: 0},
				IsAsync:      false,
				IsGenerator:  false,
			},
		}

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'v', 'a', 'l'}}},
			ValueOrNil:       js_ast.Expr{Data: jsFunction},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertySetter,
			Flags:            0,
		}
		elemName := converter.getClassElementName(prop)

		result, err := converter.convertClassMethod(prop, jsFunction, elemName)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Method)
		assert.False(t, result.Method.Get)
		assert.True(t, result.Method.Set)
	})

	t.Run("static async method", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		jsFunction := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Name:         nil,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{Start: 0}}, Loc: logger.Loc{Start: 0}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{Start: 0},
				IsAsync:      true,
				IsGenerator:  false,
			},
		}

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'r', 'u', 'n'}}},
			ValueOrNil:       js_ast.Expr{Data: jsFunction},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyMethod,
			Flags:            js_ast.PropertyIsStatic,
		}
		elemName := converter.getClassElementName(prop)

		result, err := converter.convertClassMethod(prop, jsFunction, elemName)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Method)
		assert.True(t, result.Method.Static)
		assert.True(t, result.Method.Async)
	})
}

func TestConvertClassField(t *testing.T) {
	t.Parallel()

	t.Run("field with initialiser", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'x'}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}
		elemName := converter.getClassElementName(prop)

		result, err := converter.convertClassField(prop, elemName)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Init)
	})

	t.Run("field with value but no initialiser", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'y'}}},
			ValueOrNil:       js_ast.Expr{Data: &js_ast.ENumber{Value: 10}},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}
		elemName := converter.getClassElementName(prop)

		result, err := converter.convertClassField(prop, elemName)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Init)
	})

	t.Run("field without initialiser or value", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'z'}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: nil},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            0,
		}
		elemName := converter.getClassElementName(prop)

		result, err := converter.convertClassField(prop, elemName)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.Init)
	})

	t.Run("static field with initialiser", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		prop := js_ast.Property{
			ClassStaticBlock: nil,
			Key:              js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'s'}}},
			ValueOrNil:       js_ast.Expr{Data: nil},
			InitializerOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 99}},
			Decorators:       nil,
			Loc:              logger.Loc{Start: 0},
			CloseBracketLoc:  logger.Loc{Start: 0},
			Kind:             js_ast.PropertyField,
			Flags:            js_ast.PropertyIsStatic,
		}
		elemName := converter.getClassElementName(prop)

		result, err := converter.convertClassField(prop, elemName)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Static)
		require.NotNil(t, result.Init)
	})
}

func TestConvertBinaryOp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		op       js_ast.OpCode
		expected parsejs.TokenType
	}{
		{name: "add", op: js_ast.BinOpAdd, expected: parsejs.AddToken},
		{name: "sub", op: js_ast.BinOpSub, expected: parsejs.SubToken},
		{name: "mul", op: js_ast.BinOpMul, expected: parsejs.MulToken},
		{name: "div", op: js_ast.BinOpDiv, expected: parsejs.DivToken},
		{name: "strict eq", op: js_ast.BinOpStrictEq, expected: parsejs.EqEqEqToken},
		{name: "in", op: js_ast.BinOpIn, expected: parsejs.InToken},
		{name: "instanceof", op: js_ast.BinOpInstanceof, expected: parsejs.InstanceofToken},
		{name: "unknown op defaults to add", op: js_ast.OpCode(255), expected: parsejs.AddToken},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := convertBinaryOp(tc.op)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertUnaryOp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		op       js_ast.OpCode
		expected parsejs.TokenType
	}{
		{name: "negate", op: js_ast.UnOpNeg, expected: parsejs.SubToken},
		{name: "not", op: js_ast.UnOpNot, expected: parsejs.NotToken},
		{name: "typeof", op: js_ast.UnOpTypeof, expected: parsejs.TypeofToken},
		{name: "void", op: js_ast.UnOpVoid, expected: parsejs.VoidToken},
		{name: "delete", op: js_ast.UnOpDelete, expected: parsejs.DeleteToken},
		{name: "pre increment", op: js_ast.UnOpPreInc, expected: parsejs.PreIncrToken},
		{name: "post decrement", op: js_ast.UnOpPostDec, expected: parsejs.PostDecrToken},
		{name: "unknown op defaults to not", op: js_ast.OpCode(255), expected: parsejs.NotToken},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := convertUnaryOp(tc.op)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetOpPrecedence(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		op       js_ast.OpCode
		expected int
	}{
		{name: "assign has precedence 1", op: js_ast.BinOpAssign, expected: 1},
		{name: "add has precedence 11", op: js_ast.BinOpAdd, expected: 11},
		{name: "mul has precedence 12", op: js_ast.BinOpMul, expected: 12},
		{name: "pow has precedence 13", op: js_ast.BinOpPow, expected: 13},
		{name: "unknown op has precedence 0", op: js_ast.OpCode(255), expected: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getOpPrecedence(tc.op)
			assert.Equal(t, tc.expected, result)
		})
	}
}
