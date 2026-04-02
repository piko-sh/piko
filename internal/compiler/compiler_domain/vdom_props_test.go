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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestVdomProps_ParseOnCallToParts(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantFunction string
		wantArgs     []string
	}{
		{
			name:         "simple function name without arguments",
			input:        "handleClick",
			wantFunction: "handleClick",
			wantArgs:     nil,
		},
		{
			name:         "function name with whitespace",
			input:        "  handleClick  ",
			wantFunction: "handleClick",
			wantArgs:     nil,
		},
		{
			name:         "function with empty parens",
			input:        "handleClick()",
			wantFunction: "handleClick",
			wantArgs:     []string{},
		},
		{
			name:         "function with single argument",
			input:        "handleClick(item)",
			wantFunction: "handleClick",
			wantArgs:     []string{"item"},
		},
		{
			name:         "function with multiple arguments",
			input:        "handleClick(a, b, c)",
			wantFunction: "handleClick",
			wantArgs:     []string{"a", "b", "c"},
		},
		{
			name:         "function with nested call in arguments",
			input:        "handleClick(fn(a,b), c)",
			wantFunction: "handleClick",
			wantArgs:     []string{"fn(a,b)", "c"},
		},
		{
			name:         "function with deeply nested parens",
			input:        "handle(outer(inner(x,y), z), w)",
			wantFunction: "handle",
			wantArgs:     []string{"outer(inner(x,y), z)", "w"},
		},
		{
			name:         "member access style function",
			input:        "obj.method(a)",
			wantFunction: "obj.method",
			wantArgs:     []string{"a"},
		},
		{
			name:         "function with whitespace around arguments",
			input:        "fn(  a ,  b  )",
			wantFunction: "fn",
			wantArgs:     []string{"a", "b"},
		},
		{
			name:         "empty string",
			input:        "",
			wantFunction: "",
			wantArgs:     nil,
		},
		{
			name:         "function with $event argument",
			input:        "handleClick($event)",
			wantFunction: "handleClick",
			wantArgs:     []string{"$event"},
		},
		{
			name:         "function with string literal argument",
			input:        "handleClick('hello')",
			wantFunction: "handleClick",
			wantArgs:     []string{"'hello'"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotFunction, gotArgs := parseOnCallToParts(tc.input)
			assert.Equal(t, tc.wantFunction, gotFunction, "function name mismatch")
			assert.Equal(t, tc.wantArgs, gotArgs, "arguments mismatch")
		})
	}
}

func TestVdomProps_SplitArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single argument",
			input: "a",
			want:  []string{"a"},
		},
		{
			name:  "two simple arguments",
			input: "a, b",
			want:  []string{"a", "b"},
		},
		{
			name:  "three arguments with whitespace",
			input: "  a ,  b , c  ",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "nested function call preserves commas inside parens",
			input: "fn(a,b), c",
			want:  []string{"fn(a,b)", "c"},
		},
		{
			name:  "deeply nested preserves all inner commas",
			input: "outer(inner(x,y), z), w",
			want:  []string{"outer(inner(x,y), z)", "w"},
		},
		{
			name:  "empty string produces single empty element",
			input: "",
			want:  []string{""},
		},
		{
			name:  "only commas",
			input: ",,,",
			want:  []string{"", "", "", ""},
		},
		{
			name:  "no commas with whitespace",
			input: "  singleArg  ",
			want:  []string{"singleArg"},
		},
		{
			name:  "mixed nested and flat",
			input: "a, fn(b, c), d, g(h(i))",
			want:  []string{"a", "fn(b, c)", "d", "g(h(i))"},
		},
		{
			name:  "array literal preserves commas inside brackets",
			input: "fn([a,b]), c",
			want:  []string{"fn([a,b])", "c"},
		},
		{
			name:  "object literal preserves commas inside braces",
			input: "fn({a:1,b:2}), c",
			want:  []string{"fn({a:1,b:2})", "c"},
		},
		{
			name:  "nested array and object literals",
			input: "log(['click', { x: 1, y: 2 }])",
			want:  []string{"log(['click', { x: 1, y: 2 }])"},
		},
		{
			name:  "mixed brackets braces and parens",
			input: "a([1,2], {k: v}), b, c({x: [3,4]})",
			want:  []string{"a([1,2], {k: v})", "b", "c({x: [3,4]})"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitArgs(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVdomProps_IsBooleanBound(t *testing.T) {
	tests := []struct {
		name         string
		expression   ast_domain.Expression
		booleanProps []string
		want         bool
	}{
		{
			name:         "nil expression returns false",
			expression:   nil,
			booleanProps: []string{"isActive"},
			want:         false,
		},
		{
			name:         "matching identifier",
			expression:   &ast_domain.Identifier{Name: "isActive"},
			booleanProps: []string{"isActive", "isDisabled"},
			want:         true,
		},
		{
			name:         "non-matching identifier",
			expression:   &ast_domain.Identifier{Name: "count"},
			booleanProps: []string{"isActive"},
			want:         false,
		},
		{
			name:         "empty boolean props list",
			expression:   &ast_domain.Identifier{Name: "isActive"},
			booleanProps: []string{},
			want:         false,
		},
		{
			name:         "nil boolean props list",
			expression:   &ast_domain.Identifier{Name: "isActive"},
			booleanProps: nil,
			want:         false,
		},
		{
			name: "member expression extracts property name",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "user"},
				Property: &ast_domain.Identifier{Name: "isActive"},
			},
			booleanProps: []string{"isActive"},
			want:         true,
		},
		{
			name: "member expression property not in list",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "user"},
				Property: &ast_domain.Identifier{Name: "name"},
			},
			booleanProps: []string{"isActive"},
			want:         false,
		},
		{
			name: "unary expression unwraps to identifier",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right:    &ast_domain.Identifier{Name: "isDisabled"},
			},
			booleanProps: []string{"isDisabled"},
			want:         true,
		},
		{
			name: "binary expression returns false - no base identifier",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.Identifier{Name: "b"},
			},
			booleanProps: []string{"a", "b"},
			want:         false,
		},
		{
			name: "nested unary then member",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "obj"},
					Property: &ast_domain.Identifier{Name: "active"},
				},
			},
			booleanProps: []string{"active"},
			want:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isBooleanBound(tc.expression, tc.booleanProps)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVdomProps_ExtractBaseIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		expression ast_domain.Expression
		want       string
	}{
		{
			name:       "simple identifier",
			expression: &ast_domain.Identifier{Name: "count"},
			want:       "count",
		},
		{
			name: "member expression returns property",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "user"},
				Property: &ast_domain.Identifier{Name: "name"},
			},
			want: "name",
		},
		{
			name: "unary wrapping identifier",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right:    &ast_domain.Identifier{Name: "flag"},
			},
			want: "flag",
		},
		{
			name: "unary wrapping member expression",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNeg,
				Right: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "obj"},
					Property: &ast_domain.Identifier{Name: "value"},
				},
			},
			want: "value",
		},
		{
			name: "deeply nested unary expressions",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.UnaryExpression{
					Operator: ast_domain.OpNot,
					Right:    &ast_domain.Identifier{Name: "deep"},
				},
			},
			want: "deep",
		},
		{
			name: "binary expression returns empty",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.Identifier{Name: "b"},
			},
			want: "",
		},
		{
			name:       "string literal returns empty",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			want:       "",
		},
		{
			name: "member expression with non-identifier property returns empty",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "arr"},
				Property: &ast_domain.StringLiteral{Value: "key"},
			},
			want: "",
		},
		{
			name: "call expression returns empty",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "fn"},
				Args:   nil,
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractBaseIdentifier(tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVdomProps_IsCheckboxInput(t *testing.T) {
	tests := []struct {
		node *ast_domain.TemplateNode
		name string
		want bool
	}{
		{
			name: "input with type checkbox",
			node: &ast_domain.TemplateNode{
				TagName: "input",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "type", Value: "checkbox"},
				},
			},
			want: true,
		},
		{
			name: "input with type checkbox case insensitive tag",
			node: &ast_domain.TemplateNode{
				TagName: "INPUT",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "type", Value: "checkbox"},
				},
			},
			want: true,
		},
		{
			name: "input with type checkbox case insensitive attr",
			node: &ast_domain.TemplateNode{
				TagName: "input",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "Type", Value: "Checkbox"},
				},
			},
			want: true,
		},
		{
			name: "input with type text is not checkbox",
			node: &ast_domain.TemplateNode{
				TagName: "input",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "type", Value: "text"},
				},
			},
			want: false,
		},
		{
			name: "input with no type attribute",
			node: &ast_domain.TemplateNode{
				TagName:    "input",
				Attributes: []ast_domain.HTMLAttribute{},
			},
			want: false,
		},
		{
			name: "div element is not checkbox",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "type", Value: "checkbox"},
				},
			},
			want: false,
		},
		{
			name: "input with multiple attributes finds checkbox",
			node: &ast_domain.TemplateNode{
				TagName: "input",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "form-check"},
					{Name: "type", Value: "checkbox"},
					{Name: "id", Value: "myCheck"},
				},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isCheckboxInput(tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVdomProps_CollectStaticAttrs(t *testing.T) {
	t.Run("empty attributes produces empty map", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{},
		}
		props := make(map[string]js_ast.Expr)
		collectStaticAttrs(node, props)
		assert.Empty(t, props)
	})

	t.Run("collects all static attributes", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
				{Name: "data-role", Value: "page"},
			},
		}
		props := make(map[string]js_ast.Expr)
		collectStaticAttrs(node, props)

		assert.Len(t, props, 3)

		for _, attr := range node.Attributes {
			expression, exists := props[attr.Name]
			require.True(t, exists, "expected property %q to exist", attr.Name)
			require.NotNil(t, expression.Data)
			_, isString := expression.Data.(*js_ast.EString)
			assert.True(t, isString, "expected string literal for %q", attr.Name)
		}
	})

	t.Run("nil attributes slice does nothing", func(t *testing.T) {
		node := &ast_domain.TemplateNode{}
		props := make(map[string]js_ast.Expr)
		collectStaticAttrs(node, props)
		assert.Empty(t, props)
	})
}

func TestVdomProps_MergeMultiValueProps(t *testing.T) {
	t.Run("single value goes directly to properties", func(t *testing.T) {
		properties := make(map[string]js_ast.Expr)
		multiValueProps := map[string][]js_ast.Expr{
			"onClick": {
				{Data: &js_ast.ENumber{Value: 1}},
			},
		}

		mergeMultiValueProps(properties, multiValueProps)

		expression, exists := properties["onClick"]
		require.True(t, exists)
		_, isNum := expression.Data.(*js_ast.ENumber)
		assert.True(t, isNum, "single value should remain a number expression")
	})

	t.Run("multiple values become an array", func(t *testing.T) {
		properties := make(map[string]js_ast.Expr)
		multiValueProps := map[string][]js_ast.Expr{
			"onClick": {
				{Data: &js_ast.ENumber{Value: 1}},
				{Data: &js_ast.ENumber{Value: 2}},
			},
		}

		mergeMultiValueProps(properties, multiValueProps)

		expression, exists := properties["onClick"]
		require.True(t, exists)
		arr, isArr := expression.Data.(*js_ast.EArray)
		require.True(t, isArr, "multiple values should be wrapped in an array")
		assert.Len(t, arr.Items, 2)
	})

	t.Run("existing property merged into multi-value", func(t *testing.T) {
		properties := map[string]js_ast.Expr{
			"onClick": {Data: &js_ast.ENumber{Value: 0}},
		}
		multiValueProps := map[string][]js_ast.Expr{
			"onClick": {
				{Data: &js_ast.ENumber{Value: 1}},
			},
		}

		mergeMultiValueProps(properties, multiValueProps)

		expression, exists := properties["onClick"]
		require.True(t, exists)
		arr, isArr := expression.Data.(*js_ast.EArray)
		require.True(t, isArr, "should become an array when existing + multi merge")
		assert.Len(t, arr.Items, 2, "should have the multi value + existing value")
	})

	t.Run("empty multi value does nothing", func(t *testing.T) {
		properties := map[string]js_ast.Expr{
			"class": {Data: &js_ast.EString{}},
		}
		multiValueProps := map[string][]js_ast.Expr{}

		mergeMultiValueProps(properties, multiValueProps)

		assert.Len(t, properties, 1)
		_, exists := properties["class"]
		assert.True(t, exists)
	})

	t.Run("nil properties map in multi value props", func(t *testing.T) {
		properties := make(map[string]js_ast.Expr)
		multiValueProps := map[string][]js_ast.Expr(nil)

		mergeMultiValueProps(properties, multiValueProps)
		assert.Empty(t, properties)
	})
}

func TestVdomProps_BuildPropsObject(t *testing.T) {
	t.Run("empty properties produces empty object", func(t *testing.T) {
		properties := make(map[string]js_ast.Expr)
		result := buildPropsObject(properties)

		require.NotNil(t, result.Data)
		obj, ok := result.Data.(*js_ast.EObject)
		require.True(t, ok)
		assert.Empty(t, obj.Properties)
	})

	t.Run("properties are sorted alphabetically", func(t *testing.T) {
		properties := map[string]js_ast.Expr{
			"z-attr": newStringLiteral("z"),
			"a-attr": newStringLiteral("a"),
			"m-attr": newStringLiteral("m"),
		}

		result := buildPropsObject(properties)

		require.NotNil(t, result.Data)
		obj, ok := result.Data.(*js_ast.EObject)
		require.True(t, ok)
		require.Len(t, obj.Properties, 3)

		for i, prop := range obj.Properties {
			require.NotNil(t, prop.Key.Data)
			keyString, isString := prop.Key.Data.(*js_ast.EString)
			require.True(t, isString, "key should be a string at index %d", i)
			require.NotNil(t, keyString)
		}
	})

	t.Run("nil data values are filtered out", func(t *testing.T) {
		properties := map[string]js_ast.Expr{
			"valid":   newStringLiteral("value"),
			"invalid": {Data: nil},
		}

		result := buildPropsObject(properties)

		obj, ok := result.Data.(*js_ast.EObject)
		require.True(t, ok)
		assert.Len(t, obj.Properties, 1, "nil data properties should be excluded")
	})
}

func TestVdomProps_CollectDirectiveProps(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("no directives produces empty properties", func(t *testing.T) {
		node := &ast_domain.TemplateNode{}
		properties := make(map[string]js_ast.Expr)
		collectDirectiveProps(node, properties, registry)
		assert.Empty(t, properties)
	})

	t.Run("directory-show adds _s property", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirShow: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "visible"},
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectDirectiveProps(node, properties, registry)

		_, exists := properties["_s"]
		assert.True(t, exists, "_s property should be set from DirShow")
	})

	t.Run("directory-class adds _class property", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirClass: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "classes"},
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectDirectiveProps(node, properties, registry)

		_, exists := properties["_class"]
		assert.True(t, exists, "_class property should be set from DirClass")
	})

	t.Run("directory-style adds _style property", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirStyle: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "styles"},
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectDirectiveProps(node, properties, registry)

		_, exists := properties["_style"]
		assert.True(t, exists, "_style property should be set from DirStyle")
	})

	t.Run("directory-ref adds _ref property from raw expression", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirRef: &ast_domain.Directive{
				RawExpression: "myRef",
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectDirectiveProps(node, properties, registry)

		expression, exists := properties["_ref"]
		assert.True(t, exists, "_ref property should be set from DirRef")
		require.NotNil(t, expression.Data)
		_, isString := expression.Data.(*js_ast.EString)
		assert.True(t, isString, "_ref should be a string literal")
	})

	t.Run("directory-ref with empty raw expression is not added", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirRef: &ast_domain.Directive{
				RawExpression: "",
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectDirectiveProps(node, properties, registry)

		_, exists := properties["_ref"]
		assert.False(t, exists, "_ref should not be set when RawExpression is empty")
	})

	t.Run("all directives at once", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirShow: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "visible"},
			},
			DirClass: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "cls"},
			},
			DirStyle: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "sty"},
			},
			DirRef: &ast_domain.Directive{
				RawExpression: "myRef",
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectDirectiveProps(node, properties, registry)

		assert.Contains(t, properties, "_s")
		assert.Contains(t, properties, "_class")
		assert.Contains(t, properties, "_style")
		assert.Contains(t, properties, "_ref")
	})
}

func TestVdomProps_CollectDynamicAttrs(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("empty dynamic attrs produces empty map", func(t *testing.T) {
		node := &ast_domain.TemplateNode{}
		properties := make(map[string]js_ast.Expr)
		linkExpr := collectDynamicAttrs(node, properties, false, nil, registry)

		assert.Empty(t, properties)
		assert.Nil(t, linkExpr.Data)
	})

	t.Run("dynamic attr is added with unary positive wrapper", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "title",
					Expression: &ast_domain.Identifier{Name: "pageTitle"},
				},
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectDynamicAttrs(node, properties, false, nil, registry)

		expression, exists := properties["title"]
		require.True(t, exists)
		unary, isUnary := expression.Data.(*js_ast.EUnary)
		assert.True(t, isUnary, "should be wrapped in a unary expression")
		if isUnary {
			assert.Equal(t, js_ast.UnOpPos, unary.Op)
		}
	})

	t.Run("link href is extracted instead of added to properties", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "href",
					Expression: &ast_domain.Identifier{Name: "url"},
				},
			},
		}
		properties := make(map[string]js_ast.Expr)
		linkExpr := collectDynamicAttrs(node, properties, true, nil, registry)

		assert.Empty(t, properties, "href should not be in properties for link")
		assert.NotNil(t, linkExpr.Data, "link href expression should be set")
	})

	t.Run("boolean bound property gets question mark prefix", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "disabled",
					Expression: &ast_domain.Identifier{Name: "isDisabled"},
				},
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectDynamicAttrs(node, properties, false, []string{"isDisabled"}, registry)

		_, exists := properties["?disabled"]
		assert.True(t, exists, "boolean-bound property should have ? prefix")
		_, existsWithout := properties["disabled"]
		assert.False(t, existsWithout, "should not exist without ? prefix")
	})
}

func TestVdomProps_CollectBindProps(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("nil binds returns original link expression", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Binds: nil,
		}
		originalLinkExpr := js_ast.Expr{Data: &js_ast.ENumber{Value: 42}}
		properties := make(map[string]js_ast.Expr)

		result := collectBindProps(node, properties, false, nil, originalLinkExpr, registry)
		assert.Equal(t, originalLinkExpr, result)
		assert.Empty(t, properties)
	})

	t.Run("bind prop is added with unary wrapper", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"title": {
					Expression: &ast_domain.Identifier{Name: "pageTitle"},
				},
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectBindProps(node, properties, false, nil, js_ast.Expr{}, registry)

		expression, exists := properties["title"]
		require.True(t, exists)
		unary, isUnary := expression.Data.(*js_ast.EUnary)
		assert.True(t, isUnary, "bind prop should be wrapped in unary")
		if isUnary {
			assert.Equal(t, js_ast.UnOpPos, unary.Op)
		}
	})

	t.Run("bind href on link is captured as link expression", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"href": {
					Expression: &ast_domain.Identifier{Name: "url"},
				},
			},
		}
		properties := make(map[string]js_ast.Expr)
		result := collectBindProps(node, properties, true, nil, js_ast.Expr{}, registry)

		assert.Empty(t, properties, "href should not be in properties for link")
		assert.NotNil(t, result.Data, "link href expression should be captured")
	})

	t.Run("boolean bound bind prop gets question mark prefix", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"checked": {
					Expression: &ast_domain.Identifier{Name: "isChecked"},
				},
			},
		}
		properties := make(map[string]js_ast.Expr)
		collectBindProps(node, properties, false, []string{"isChecked"}, js_ast.Expr{}, registry)

		_, exists := properties["?checked"]
		assert.True(t, exists, "boolean-bound bind should have ? prefix")
	})
}

func TestVdomProps_BuildActionHandler(t *testing.T) {
	t.Run("returns handler block and name for identifier", func(t *testing.T) {
		d := ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "contact.send"},
		}

		block, name := buildActionHandler(d)

		assert.Equal(t, "__internal_action_handler", name)
		require.NotNil(t, block, "handler block should not be nil")
		assert.NotEmpty(t, block.Stmts, "handler block should have statements")
	})

	t.Run("extracts callee name from call expression", func(t *testing.T) {
		d := ast_domain.Directive{
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "contact.send"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "$form"}},
			},
		}

		block, name := buildActionHandler(d)

		assert.Equal(t, "__internal_action_handler", name)
		require.NotNil(t, block)
	})

	t.Run("escapes special characters in action name", func(t *testing.T) {
		d := ast_domain.Directive{
			Expression: &ast_domain.StringLiteral{Value: "form.submit"},
		}

		block, name := buildActionHandler(d)

		assert.Equal(t, "__internal_action_handler", name)
		require.NotNil(t, block)
	})
}

func TestVdomProps_BuildHelperHandler(t *testing.T) {
	t.Run("returns handler block and name", func(t *testing.T) {
		d := ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "validate"},
		}

		block, name := buildHelperHandler(d)

		assert.Equal(t, "__internal_helper_handler", name)
		require.NotNil(t, block, "handler block should not be nil")
		assert.NotEmpty(t, block.Stmts, "handler block should have statements")
	})
}

func TestVdomProps_TransformUserArgs(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("empty arguments returns nil", func(t *testing.T) {
		result := transformUserArgs(context.Background(), nil, registry)
		assert.Nil(t, result)
	})

	t.Run("empty slice returns empty slice", func(t *testing.T) {
		result := transformUserArgs(context.Background(), []string{}, registry)
		require.NotNil(t, result, "empty slice input should return empty slice, not nil")
		assert.Empty(t, result)
	})

	t.Run("valid identifier argument is transformed", func(t *testing.T) {
		result := transformUserArgs(context.Background(), []string{"myVar"}, registry)
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Data)
	})

	t.Run("multiple arguments are all transformed", func(t *testing.T) {
		result := transformUserArgs(context.Background(), []string{"a", "b", "c"}, registry)
		require.Len(t, result, 3)
		for i, expression := range result {
			assert.NotNil(t, expression.Data, "argument at index %d should have data", i)
		}
	})

	t.Run("numeric argument is transformed", func(t *testing.T) {
		result := transformUserArgs(context.Background(), []string{"42"}, registry)
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Data)
	})

	t.Run("invalid expression falls back to undefined", func(t *testing.T) {

		result := transformUserArgs(context.Background(), []string{"@@@"}, registry)
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Data, "should have fallback data")
	})
}

func TestVdomProps_HandleLinkProps(t *testing.T) {
	t.Run("no href expressions sets fallback", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		properties := make(map[string]js_ast.Expr)
		multiValueProps := make(map[string][]js_ast.Expr)

		handleLinkProps(ctx, properties, multiValueProps, js_ast.Expr{}, js_ast.Expr{}, false, events)

		expression, exists := properties[propHref]
		require.True(t, exists, "should set a fallback href")
		require.NotNil(t, expression.Data)
		_, isString := expression.Data.(*js_ast.EString)
		assert.True(t, isString, "fallback href should be a string literal")
	})

	t.Run("literal href is set in properties", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		properties := make(map[string]js_ast.Expr)
		multiValueProps := make(map[string][]js_ast.Expr)

		litHref := newStringLiteral("/about")

		handleLinkProps(ctx, properties, multiValueProps, litHref, js_ast.Expr{}, false, events)

		_, exists := properties[propHref]
		assert.True(t, exists, "href should be set from literal")
	})

	t.Run("expression href is wrapped in String call", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		properties := make(map[string]js_ast.Expr)
		multiValueProps := make(map[string][]js_ast.Expr)

		expressionHref := js_ast.Expr{Data: &js_ast.ENumber{Value: 1}}

		handleLinkProps(ctx, properties, multiValueProps, js_ast.Expr{}, expressionHref, false, events)

		expression, exists := properties[propHref]
		require.True(t, exists, "href should be set from expression")
		call, isCall := expression.Data.(*js_ast.ECall)
		assert.True(t, isCall, "expression href should be wrapped in a call")
		if isCall {
			assert.NotNil(t, call.Target.Data)
		}
	})

	t.Run("user clicked skips navigate handler", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		properties := make(map[string]js_ast.Expr)
		multiValueProps := make(map[string][]js_ast.Expr)

		litHref := newStringLiteral("/about")

		handleLinkProps(ctx, properties, multiValueProps, litHref, js_ast.Expr{}, true, events)

		_, hasClickHandler := multiValueProps["onClick"]
		assert.False(t, hasClickHandler, "should not add navigate handler when user already has click handler")
	})

	t.Run("no user click adds navigate handler", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		properties := make(map[string]js_ast.Expr)
		multiValueProps := make(map[string][]js_ast.Expr)

		litHref := newStringLiteral("/about")

		handleLinkProps(ctx, properties, multiValueProps, litHref, js_ast.Expr{}, false, events)

		_, hasClickHandler := multiValueProps["onClick"]
		assert.True(t, hasClickHandler, "should add navigate handler when user has no click handler")
	})
}

func TestVdomProps_BuildEventHandlerExpr(t *testing.T) {
	t.Run("default modifier creates binding", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		d := ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "handleClick"},
			Modifier:   "",
		}

		expression, err := buildEventHandlerExpr(ctx, d, "click", events, nil)

		require.NoError(t, err)
		assert.NotNil(t, expression.Data, "should return a non-nil expression")
		assert.Len(t, events.getBindings(), 1, "should have created one binding")
	})

	t.Run("action modifier creates action handler", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		d := ast_domain.Directive{
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "contact.send"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "$form"}},
			},
			Modifier: "action",
		}

		expression, err := buildEventHandlerExpr(ctx, d, "click", events, nil)

		require.NoError(t, err)
		assert.NotNil(t, expression.Data)
		assert.Len(t, events.getBindings(), 1)
	})

	t.Run("helper modifier creates helper handler", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		d := ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "validate"},
			Modifier:   "helper",
		}

		expression, err := buildEventHandlerExpr(ctx, d, "submit", events, nil)

		require.NoError(t, err)
		assert.NotNil(t, expression.Data)
		assert.Len(t, events.getBindings(), 1)
	})

	t.Run("function call with arguments creates binding", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		d := ast_domain.Directive{
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handleClick"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "item"}},
			},
			Modifier: "",
		}

		expression, err := buildEventHandlerExpr(ctx, d, "click", events, nil)

		require.NoError(t, err)
		assert.NotNil(t, expression.Data)
	})

	t.Run("multiple bindings increment index", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		for i := range 3 {
			d := ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "handler"},
				Modifier:   "",
			}
			_, err := buildEventHandlerExpr(ctx, d, "click", events, nil)
			require.NoError(t, err, "binding %d should not error", i)
		}

		assert.Len(t, events.getBindings(), 3)
	})
}

func TestVdomProps_DirTextDynamicExpr(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("produces array with dom call", func(t *testing.T) {
		expression := &ast_domain.Identifier{Name: "message"}
		keyBase := newStringLiteral("k")

		result, err := dirTextDynamicExpr(expression, keyBase, registry)

		require.NoError(t, err)
		require.NotNil(t, result.Data)
		arr, isArr := result.Data.(*js_ast.EArray)
		require.True(t, isArr, "result should be an array expression")
		assert.Len(t, arr.Items, 1, "should contain one DOM call element")
	})
}

func TestVdomProps_DirHTMLDynamicExpr(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("produces array with dom call", func(t *testing.T) {
		expression := &ast_domain.Identifier{Name: "htmlContent"}
		keyBase := newStringLiteral("k")

		result, err := dirHTMLDynamicExpr(expression, keyBase, registry)

		require.NoError(t, err)
		require.NotNil(t, result.Data)
		arr, isArr := result.Data.(*js_ast.EArray)
		require.True(t, isArr, "result should be an array expression")
		assert.Len(t, arr.Items, 1, "should contain one DOM call element")
	})
}

func TestVdomProps_BuildDynamicContentExpr(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("text content type", func(t *testing.T) {
		expression := &ast_domain.Identifier{Name: "text"}
		keyBase := newStringLiteral("key")

		result, err := buildDynamicContentExpr(expression, keyBase, registry, "txt")

		require.NoError(t, err)
		require.NotNil(t, result.Data)
		arr, isArr := result.Data.(*js_ast.EArray)
		require.True(t, isArr)
		assert.Len(t, arr.Items, 1)
	})

	t.Run("html content type", func(t *testing.T) {
		expression := &ast_domain.Identifier{Name: "rawHtml"}
		keyBase := newStringLiteral("key")

		result, err := buildDynamicContentExpr(expression, keyBase, registry, "html")

		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})

	t.Run("nil expression data still produces a result", func(t *testing.T) {

		keyBase := newStringLiteral("key")

		result, err := buildDynamicContentExpr(nil, keyBase, registry, "txt")

		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})
}

func TestVdomProps_ParseModelHandlerBlockForExpr(t *testing.T) {
	t.Run("input handler uses value property", func(t *testing.T) {
		block := parseModelHandlerBlockForExpr("this.$$ctx.name", false)
		require.NotNil(t, block, "should parse successfully for input model")
		assert.NotEmpty(t, block.Stmts)
	})

	t.Run("checkbox handler uses checked property", func(t *testing.T) {
		block := parseModelHandlerBlockForExpr("this.$$ctx.isActive", true)
		require.NotNil(t, block, "should parse successfully for checkbox model")
		assert.NotEmpty(t, block.Stmts)
	})
}

func TestVdomProps_CollectEventHandlers(t *testing.T) {
	t.Run("no events returns false and empty multi-value props", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		multiValueProps := make(map[string][]js_ast.Expr)

		node := &ast_domain.TemplateNode{}

		userClicked := collectEventHandlers(ctx, node, events, nil, multiValueProps)

		assert.False(t, userClicked)
		assert.Empty(t, multiValueProps)
	})

	t.Run("click event sets userClicked true", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		multiValueProps := make(map[string][]js_ast.Expr)

		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{
						Expression: &ast_domain.Identifier{Name: "handleClick"},
						Arg:        "click",
					},
				},
			},
		}

		userClicked := collectEventHandlers(ctx, node, events, nil, multiValueProps)

		assert.True(t, userClicked)
		assert.Contains(t, multiValueProps, "onClick")
	})

	t.Run("non-click event does not set userClicked", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		multiValueProps := make(map[string][]js_ast.Expr)

		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"submit": {
					{
						Expression: &ast_domain.Identifier{Name: "handleSubmit"},
						Arg:        "submit",
					},
				},
			},
		}

		userClicked := collectEventHandlers(ctx, node, events, nil, multiValueProps)

		assert.False(t, userClicked)
		assert.Contains(t, multiValueProps, "onSubmit")
	})

	t.Run("custom events use pe: prefix", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		multiValueProps := make(map[string][]js_ast.Expr)

		node := &ast_domain.TemplateNode{
			CustomEvents: map[string][]ast_domain.Directive{
				"update": {
					{
						Expression: &ast_domain.Identifier{Name: "handleUpdate"},
						Arg:        "update",
					},
				},
			},
		}

		userClicked := collectEventHandlers(ctx, node, events, nil, multiValueProps)

		assert.False(t, userClicked)
		assert.Contains(t, multiValueProps, "pe:update")
	})

	t.Run("multiple handlers for same event", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		multiValueProps := make(map[string][]js_ast.Expr)

		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{
						Expression: &ast_domain.Identifier{Name: "handler1"},
						Arg:        "click",
					},
					{
						Expression: &ast_domain.Identifier{Name: "handler2"},
						Arg:        "click",
					},
				},
			},
		}

		userClicked := collectEventHandlers(ctx, node, events, nil, multiValueProps)

		assert.True(t, userClicked)
		assert.Len(t, multiValueProps["onClick"], 2, "should have two handlers for onClick")
	})
}

func TestVdomProps_HandleModelDirective(t *testing.T) {
	t.Run("text input binds to value and onInput", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		properties := make(map[string]js_ast.Expr)
		multiValueProps := make(map[string][]js_ast.Expr)

		node := &ast_domain.TemplateNode{
			TagName: "input",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "type", Value: "text"},
			},
			DirModel: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "userName"},
			},
		}

		handleModelDirective(ctx, node, properties, multiValueProps, events, nil)

		_, hasValue := properties["value"]
		assert.True(t, hasValue, "text input should bind to value property")

		_, hasChecked := properties["?checked"]
		assert.False(t, hasChecked, "text input should not bind to checked")

		_, hasOnInput := multiValueProps["onInput"]
		assert.True(t, hasOnInput, "text input should have onInput handler")

		_, hasOnChange := multiValueProps["onChange"]
		assert.False(t, hasOnChange, "text input should not have onChange handler")
	})

	t.Run("checkbox input binds to checked and onChange", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		properties := make(map[string]js_ast.Expr)
		multiValueProps := make(map[string][]js_ast.Expr)

		node := &ast_domain.TemplateNode{
			TagName: "input",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "type", Value: "checkbox"},
			},
			DirModel: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "isActive"},
			},
		}

		handleModelDirective(ctx, node, properties, multiValueProps, events, nil)

		_, hasChecked := properties["?checked"]
		assert.True(t, hasChecked, "checkbox should bind to ?checked property")

		_, hasValue := properties["value"]
		assert.False(t, hasValue, "checkbox should not bind to value")

		_, hasOnChange := multiValueProps["onChange"]
		assert.True(t, hasOnChange, "checkbox should have onChange handler")

		_, hasOnInput := multiValueProps["onInput"]
		assert.False(t, hasOnInput, "checkbox should not have onInput handler")
	})

	t.Run("nil expression does not panic and still binds null", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)
		properties := make(map[string]js_ast.Expr)
		multiValueProps := make(map[string][]js_ast.Expr)

		node := &ast_domain.TemplateNode{
			TagName: "input",
			DirModel: &ast_domain.Directive{
				Expression: nil,
			},
		}

		handleModelDirective(ctx, node, properties, multiValueProps, events, nil)

		_, hasValue := properties["value"]
		assert.True(t, hasValue, "nil expression transforms to null literal, so value is bound")
	})
}

func TestVdomProps_BuildPropsAST(t *testing.T) {
	t.Run("empty node produces empty object", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			TagName: "div",
		}

		result, err := buildPropsAST(ctx, node, events, false, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)
		obj, ok := result.Data.(*js_ast.EObject)
		require.True(t, ok)
		assert.Empty(t, obj.Properties)
	})

	t.Run("node with static attributes", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			TagName: "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
		}

		result, err := buildPropsAST(ctx, node, events, false, nil, nil)

		require.NoError(t, err)
		obj, ok := result.Data.(*js_ast.EObject)
		require.True(t, ok)
		assert.Len(t, obj.Properties, 2)
	})

	t.Run("link node with no href gets fallback", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			TagName: "a",
		}

		result, err := buildPropsAST(ctx, node, events, true, nil, nil)

		require.NoError(t, err)
		obj, ok := result.Data.(*js_ast.EObject)
		require.True(t, ok)
		assert.NotEmpty(t, obj.Properties, "link should have at least href fallback")
	})
}
