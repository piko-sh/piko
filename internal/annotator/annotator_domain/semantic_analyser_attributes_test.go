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

package annotator_domain

import (
	"context"
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestAttributeAnalyser_AnalyseNodeAttributes(t *testing.T) {
	t.Run("SimpleElementWithNoDirectives", func(t *testing.T) {
		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		analyser := newAttributeAnalyser(nil, nil, contextManager, "", nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		analyser.AnalyseNodeAttributes(context.Background(), node, ctx, nil, nil)
	})

	t.Run("ElementWithStaticAttributes", func(t *testing.T) {
		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		analyser := newAttributeAnalyser(nil, nil, contextManager, "", nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
		}

		analyser.AnalyseNodeAttributes(context.Background(), node, ctx, nil, nil)
	})
}

func TestAttributeAnalyser_resolveObjectLiteralValues(t *testing.T) {
	t.Run("NilExpression", func(t *testing.T) {
		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		analyser := newAttributeAnalyser(nil, nil, contextManager, "", nil)
		loc := ast_domain.Location{Line: 1, Column: 1, Offset: 0}

		analyser.resolveObjectLiteralValues(context.Background(), ctx, nil, loc)
	})

	t.Run("NonObjectLiteralExpression", func(t *testing.T) {
		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		analyser := newAttributeAnalyser(nil, nil, contextManager, "", nil)
		loc := ast_domain.Location{Line: 1, Column: 1, Offset: 0}
		expression := &ast_domain.StringLiteral{Value: "test"}

		analyser.resolveObjectLiteralValues(context.Background(), ctx, expression, loc)
	})
}

func TestValidateAttributeTypeIsStringable(t *testing.T) {
	t.Run("NilAnnotation", func(t *testing.T) {
		ctx := createTestContext()
		attr := &ast_domain.DynamicAttribute{
			Name:          "title",
			RawExpression: "state.Title",
		}

		validateAttributeTypeIsStringable(ctx, attr)

		if len(*ctx.Diagnostics) != 0 {
			t.Errorf("Expected no diagnostics, got %d", len(*ctx.Diagnostics))
		}
	})

	t.Run("StringableType", func(t *testing.T) {
		ctx := createTestContext()
		attr := &ast_domain.DynamicAttribute{
			Name:          "title",
			RawExpression: "state.Title",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Stringability: 1,
			},
		}

		validateAttributeTypeIsStringable(ctx, attr)

		if len(*ctx.Diagnostics) != 0 {
			t.Errorf("Expected no diagnostics for stringable type, got %d", len(*ctx.Diagnostics))
		}
	})
}

func TestValidateClassAttribute(t *testing.T) {
	t.Run("NilAnnotation", func(t *testing.T) {
		ctx := createTestContext()
		attr := &ast_domain.DynamicAttribute{
			Name:          "class",
			RawExpression: "state.Classes",
		}

		validateClassAttribute(ctx, attr)

		if len(*ctx.Diagnostics) != 0 {
			t.Errorf("Expected no diagnostics, got %d", len(*ctx.Diagnostics))
		}
	})
}

func TestValidateConditionalDirective(t *testing.T) {
	t.Run("NilExpression", func(t *testing.T) {
		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveIf,
			RawExpression: "state.IsActive",
		}

		validateConditionalDirective(directive, ctx)
	})
}

func TestValidateModelDirective(t *testing.T) {
	t.Run("ValidIdentifier", func(t *testing.T) {
		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveModel,
			RawExpression: "state.Name",
			Expression: &ast_domain.Identifier{
				Name: "state",
			},
		}

		validateModelDirective(directive, ctx)

		if len(*ctx.Diagnostics) != 0 {
			t.Errorf("Expected no diagnostics for valid identifier, got %d", len(*ctx.Diagnostics))
		}
	})

	t.Run("ValidMemberExpression", func(t *testing.T) {
		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveModel,
			RawExpression: "state.Name",
			Expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "state"},
				Property: &ast_domain.Identifier{Name: "Name"},
			},
		}

		validateModelDirective(directive, ctx)

		if len(*ctx.Diagnostics) != 0 {
			t.Errorf("Expected no diagnostics for valid member expression, got %d", len(*ctx.Diagnostics))
		}
	})

	t.Run("InvalidLiteral", func(t *testing.T) {
		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveModel,
			RawExpression: "\"literal\"",
			Expression: &ast_domain.StringLiteral{
				Value: "literal",
			},
		}

		validateModelDirective(directive, ctx)

		if len(*ctx.Diagnostics) != 1 {
			t.Fatalf("Expected 1 diagnostic, got %d", len(*ctx.Diagnostics))
		}
		if (*ctx.Diagnostics)[0].Severity != ast_domain.Error {
			t.Errorf("Expected Error severity, got %v", (*ctx.Diagnostics)[0].Severity)
		}
	})
}

func TestValidateEventDirective(t *testing.T) {
	t.Run("ValidCallExpression", func(t *testing.T) {
		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveOn,
			RawExpression: "handleClick()",
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handleClick"},
			},
		}

		validateEventDirective(directive, ctx)

		if len(*ctx.Diagnostics) != 0 {
			t.Errorf("Expected no diagnostics for valid call expression, got %d", len(*ctx.Diagnostics))
		}
	})

	t.Run("InvalidIdentifier", func(t *testing.T) {
		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveOn,
			RawExpression: "handleClick",
			Expression: &ast_domain.Identifier{
				Name: "handleClick",
			},
		}

		validateEventDirective(directive, ctx)

		if len(*ctx.Diagnostics) != 1 {
			t.Fatalf("Expected 1 diagnostic, got %d", len(*ctx.Diagnostics))
		}
		if (*ctx.Diagnostics)[0].Severity != ast_domain.Error {
			t.Errorf("Expected Error severity, got %v", (*ctx.Diagnostics)[0].Severity)
		}
	})
}

func createTestContext() *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/package",
		CurrentGoPackageName:     "test",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            "/test/file.phtml",
		Logger:                   logger_domain.GetLogger("test"),
	}
}

func createMinimalVirtualModule() *annotator_dto.VirtualModule {
	return &annotator_dto.VirtualModule{
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: make(map[string]string),
		},
		ComponentsByHash:   make(map[string]*annotator_dto.VirtualComponent),
		ComponentsByGoPath: make(map[string]*annotator_dto.VirtualComponent),
	}
}

func TestIsCallExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		expected   bool
	}{
		{
			name: "call expression returns true",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "foo"},
			},
			expected: true,
		},
		{
			name:       "identifier returns false",
			expression: &ast_domain.Identifier{Name: "foo"},
			expected:   false,
		},
		{
			name:       "member expression returns false",
			expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "obj"}},
			expected:   false,
		},
		{
			name:       "string literal returns false",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isCallExpr(tc.expression)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContainsEventPlaceholder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		expected   bool
	}{
		{
			name:       "$event identifier returns true",
			expression: &ast_domain.Identifier{Name: "$event"},
			expected:   true,
		},
		{
			name:       "other identifier returns false",
			expression: &ast_domain.Identifier{Name: "foo"},
			expected:   false,
		},
		{
			name: "$event in call arguments returns true",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handler"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "$event"}},
			},
			expected: true,
		},
		{
			name: "nested $event returns true",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "$event"},
				Property: &ast_domain.Identifier{Name: "target"},
			},
			expected: true,
		},
		{
			name:       "string literal returns false",
			expression: &ast_domain.StringLiteral{Value: "$event"},
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := containsEventPlaceholder(tc.expression)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContainsFormPlaceholder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		expected   bool
	}{
		{
			name:       "$form identifier returns true",
			expression: &ast_domain.Identifier{Name: "$form"},
			expected:   true,
		},
		{
			name:       "other identifier returns false",
			expression: &ast_domain.Identifier{Name: "form"},
			expected:   false,
		},
		{
			name: "$form in call arguments returns true",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "submit"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "$form"}},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := containsFormPlaceholder(tc.expression)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindEventPropertyAccess(t *testing.T) {
	t.Parallel()

	t.Run("finds $event property access", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "$event"},
			Property: &ast_domain.Identifier{Name: "target"},
		}

		result := findEventPropertyAccess(expression)

		require.NotNil(t, result)
		assert.Equal(t, "$event", result.Base.(*ast_domain.Identifier).Name)
	})

	t.Run("returns nil for non-$event member", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "state"},
			Property: &ast_domain.Identifier{Name: "value"},
		}

		result := findEventPropertyAccess(expression)

		assert.Nil(t, result)
	})

	t.Run("returns nil for simple identifier", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "$event"}

		result := findEventPropertyAccess(expression)

		assert.Nil(t, result)
	})
}

func TestFindFormPropertyAccess(t *testing.T) {
	t.Parallel()

	t.Run("finds $form property access", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "$form"},
			Property: &ast_domain.Identifier{Name: "data"},
		}

		result := findFormPropertyAccess(expression)

		require.NotNil(t, result)
		assert.Equal(t, "$form", result.Base.(*ast_domain.Identifier).Name)
	})

	t.Run("returns nil for non-$form member", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "form"},
			Property: &ast_domain.Identifier{Name: "data"},
		}

		result := findFormPropertyAccess(expression)

		assert.Nil(t, result)
	})
}

func TestFindLegacyEventIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("finds bare event identifier", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "event"}

		result := findLegacyEventIdentifier(expression)

		require.NotNil(t, result)
		assert.Equal(t, "event", result.Name)
	})

	t.Run("returns nil for $event", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "$event"}

		result := findLegacyEventIdentifier(expression)

		assert.Nil(t, result)
	})

	t.Run("returns nil for other identifiers", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "foo"}

		result := findLegacyEventIdentifier(expression)

		assert.Nil(t, result)
	})
}

func TestIsClassBindingType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil type info returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name: "nil type expr returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: nil,
			},
			expected: false,
		},
		{
			name: "string type returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			expected: true,
		},
		{
			name: "slice type returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
			expected: true,
		},
		{
			name: "map[string]bool returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{
					Key:   goast.NewIdent("string"),
					Value: goast.NewIdent("bool"),
				},
			},
			expected: true,
		},
		{
			name: "int type returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isClassBindingType(tc.typeInfo)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsStyleBindingType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil type info returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name: "string type returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			expected: true,
		},
		{
			name: "map[string]string returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{
					Key:   goast.NewIdent("string"),
					Value: goast.NewIdent("string"),
				},
			},
			expected: true,
		},
		{
			name: "slice type returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isStyleBindingType(tc.typeInfo)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsComplexType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil type info returns true",
			typeInfo: nil,
			expected: true,
		},
		{
			name: "nil type expr returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: nil,
			},
			expected: true,
		},
		{
			name: "string is not complex",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			expected: false,
		},
		{
			name: "int is not complex",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			expected: false,
		},
		{
			name: "bool is not complex",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("bool"),
			},
			expected: false,
		},
		{
			name: "custom struct name is complex",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyStruct"),
			},
			expected: true,
		},
		{
			name: "struct type is complex",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StructType{},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isComplexType(tc.typeInfo)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExpressionHasDynamicScopeRefs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		expected   bool
	}{
		{
			name:       "nil expression returns false",
			expression: nil,
			expected:   false,
		},
		{
			name:       "$event returns false (client-side)",
			expression: &ast_domain.Identifier{Name: "$event"},
			expected:   false,
		},
		{
			name:       "$form returns false (client-side)",
			expression: &ast_domain.Identifier{Name: "$form"},
			expected:   false,
		},
		{
			name:       "regular identifier returns true",
			expression: &ast_domain.Identifier{Name: "item"},
			expected:   true,
		},
		{
			name: "member expression returns true",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "state"},
				Property: &ast_domain.Identifier{Name: "value"},
			},
			expected: true,
		},
		{
			name:       "string literal returns false",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			expected:   false,
		},
		{
			name:       "integer literal returns false",
			expression: &ast_domain.IntegerLiteral{Value: 42},
			expected:   false,
		},
		{
			name:       "boolean literal returns false",
			expression: &ast_domain.BooleanLiteral{Value: true},
			expected:   false,
		},
		{
			name: "call with $event only returns false",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handler"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "$event"}},
			},
			expected: false,
		},
		{
			name: "call with dynamic argument returns true",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handler"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "item"}},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := expressionHasDynamicScopeRefs(tc.expression)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractActionNameFromDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		rawExpression string
		expected      string
	}{
		{
			name:          "simple action name",
			rawExpression: "action.email.send",
			expected:      "email.send",
		},
		{
			name:          "action with function call",
			rawExpression: "action.email.send($form)",
			expected:      "email.send",
		},
		{
			name:          "action with multiple arguments",
			rawExpression: "action.user.create(name, email)",
			expected:      "user.create",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := &ast_domain.Directive{RawExpression: tc.rawExpression}

			result := extractActionNameFromDirective(d)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewSyntheticTypeInfo(t *testing.T) {
	t.Parallel()

	t.Run("creates type info with given name", func(t *testing.T) {
		t.Parallel()

		result := newSyntheticTypeInfo("js.Event")

		require.NotNil(t, result)
		require.NotNil(t, result.TypeExpression)

		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "js.Event", identifier.Name)
		assert.True(t, result.IsSynthetic)
	})
}

func TestNewSyntheticAnyTypeInfo(t *testing.T) {
	t.Parallel()

	t.Run("creates any type info", func(t *testing.T) {
		t.Parallel()

		result := newSyntheticAnyTypeInfo()

		require.NotNil(t, result)
		require.NotNil(t, result.TypeExpression)

		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
		assert.False(t, result.IsSynthetic)
	})
}

func TestAttributeAnalyser_isActionCall(t *testing.T) {
	t.Parallel()

	aa := &AttributeAnalyser{}

	testCases := []struct {
		name          string
		rawExpression string
		expected      bool
	}{
		{
			name:          "action call returns true",
			rawExpression: "action.email.send($form)",
			expected:      true,
		},
		{
			name:          "action without arguments returns true",
			rawExpression: "action.user.logout",
			expected:      true,
		},
		{
			name:          "non-action returns false",
			rawExpression: "handleClick($event)",
			expected:      false,
		},
		{
			name:          "remote action style returns false",
			rawExpression: "emailSend($form)",
			expected:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := &ast_domain.Directive{RawExpression: tc.rawExpression}

			result := aa.isActionCall(d)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAttributeAnalyser_determineKeyLocation(t *testing.T) {
	t.Parallel()

	aa := &AttributeAnalyser{}

	t.Run("returns DirKey location when present", func(t *testing.T) {
		t.Parallel()

		keyLocation := ast_domain.Location{Line: 10, Column: 5}
		node := &ast_domain.TemplateNode{
			DirKey:   &ast_domain.Directive{Location: keyLocation},
			Location: ast_domain.Location{Line: 1, Column: 1},
		}

		result := aa.determineKeyLocation(node)

		assert.Equal(t, keyLocation, result)
	})

	t.Run("returns DirFor location when DirKey is nil", func(t *testing.T) {
		t.Parallel()

		forLocation := ast_domain.Location{Line: 20, Column: 3}
		node := &ast_domain.TemplateNode{
			DirFor:   &ast_domain.Directive{Location: forLocation},
			Location: ast_domain.Location{Line: 1, Column: 1},
		}

		result := aa.determineKeyLocation(node)

		assert.Equal(t, forLocation, result)
	})

	t.Run("returns node location as fallback", func(t *testing.T) {
		t.Parallel()

		nodeLocation := ast_domain.Location{Line: 30, Column: 10}
		node := &ast_domain.TemplateNode{
			Location: nodeLocation,
		}

		result := aa.determineKeyLocation(node)

		assert.Equal(t, nodeLocation, result)
	})
}

type mockActionInfo struct {
	method string
}

func (m mockActionInfo) Method() string {
	return m.method
}

func TestAttributeAnalyser_lookupActionCaseInsensitive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		actions    map[string]ActionInfoProvider
		name       string
		lookup     string
		wantMethod string
		wantNil    bool
	}{
		{
			name: "exact match returns action",
			actions: map[string]ActionInfoProvider{
				"email.Send": mockActionInfo{method: "POST"},
			},
			lookup:     "email.Send",
			wantNil:    false,
			wantMethod: "POST",
		},
		{
			name: "case-insensitive match returns action",
			actions: map[string]ActionInfoProvider{
				"email.Send": mockActionInfo{method: "POST"},
			},
			lookup:     "email.send",
			wantNil:    false,
			wantMethod: "POST",
		},
		{
			name: "uppercase lookup matches lowercase action",
			actions: map[string]ActionInfoProvider{
				"user.logout": mockActionInfo{method: "GET"},
			},
			lookup:     "User.Logout",
			wantNil:    false,
			wantMethod: "GET",
		},
		{
			name: "no match returns nil",
			actions: map[string]ActionInfoProvider{
				"email.Send": mockActionInfo{method: "POST"},
			},
			lookup:  "user.Delete",
			wantNil: true,
		},
		{
			name:    "empty actions map returns nil",
			actions: map[string]ActionInfoProvider{},
			lookup:  "anything",
			wantNil: true,
		},
		{
			name: "exact match preferred over case-insensitive",
			actions: map[string]ActionInfoProvider{
				"email.send": mockActionInfo{method: "GET"},
				"email.Send": mockActionInfo{method: "POST"},
			},
			lookup:     "email.Send",
			wantNil:    false,
			wantMethod: "POST",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			aa := &AttributeAnalyser{
				actions: tc.actions,
			}

			result := aa.lookupActionCaseInsensitive(tc.lookup)

			if tc.wantNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.wantMethod, result.Method())
			}
		})
	}
}

func TestValidateAttributeTypeIsStringable_NonStringableType(t *testing.T) {
	t.Parallel()

	t.Run("non-stringable type produces warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		attr := &ast_domain.DynamicAttribute{
			Name:          "data-value",
			RawExpression: "state.Data",
			Location:      ast_domain.Location{Line: 5, Column: 10},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Stringability:         0,
				IsPointerToStringable: false,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("MyStruct"),
				},
			},
		}

		validateAttributeTypeIsStringable(ctx, attr)

		require.Equal(t, 1, len(*ctx.Diagnostics))
		assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "MyStruct")
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "not directly renderable")
	})

	t.Run("pointer to stringable type produces no warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		attr := &ast_domain.DynamicAttribute{
			Name:          "title",
			RawExpression: "state.Title",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Stringability:         0,
				IsPointerToStringable: true,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
		}

		validateAttributeTypeIsStringable(ctx, attr)

		assert.Equal(t, 0, len(*ctx.Diagnostics))
	})

	t.Run("skips when error diagnostic already exists for same location", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		loc := ast_domain.Location{Line: 3, Column: 7}
		rawExpr := "state.BadField"

		*ctx.Diagnostics = append(*ctx.Diagnostics, &ast_domain.Diagnostic{
			Severity:   ast_domain.Error,
			Message:    "existing error",
			Location:   loc,
			Expression: rawExpr,
		})

		attr := &ast_domain.DynamicAttribute{
			Name:          "title",
			RawExpression: rawExpr,
			Location:      loc,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Stringability:         0,
				IsPointerToStringable: false,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("MyStruct"),
				},
			},
		}

		validateAttributeTypeIsStringable(ctx, attr)

		assert.Equal(t, 1, len(*ctx.Diagnostics))
		assert.Equal(t, "existing error", (*ctx.Diagnostics)[0].Message)
	})
}

func TestAttributeAnalyser_analyseConditionalDirectives(t *testing.T) {
	t.Parallel()

	t.Run("all nil directives produces no diagnostics", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		resolver := NewTypeResolver(nil, vm, nil)
		aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		r := &attributeContextResolver{
			aa:  aa,
			ctx: ctx,
		}

		aa.analyseConditionalDirectives(context.Background(), node, r)

		assert.Equal(t, 0, len(*ctx.Diagnostics))
	})

	t.Run("directive with nil expression is skipped", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		resolver := NewTypeResolver(nil, vm, nil)
		aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirIf: &ast_domain.Directive{
				Type:          ast_domain.DirectiveIf,
				RawExpression: "state.Active",
				Expression:    nil,
			},
		}

		r := &attributeContextResolver{
			aa:  aa,
			ctx: ctx,
		}

		aa.analyseConditionalDirectives(context.Background(), node, r)

		assert.Equal(t, 0, len(*ctx.Diagnostics))
	})
}

func TestAttributeAnalyser_getValidatorForContext(t *testing.T) {
	t.Parallel()

	t.Run("nil activePInfo returns main validator", func(t *testing.T) {
		t.Parallel()

		mainValidator := NewPKValidator("console.log('hello')", "/main.pk")
		aa := &AttributeAnalyser{
			pkValidators:      map[string]*PKValidator{"main": mainValidator},
			mainComponentHash: "main",
		}

		result := aa.getValidatorForContext(nil)

		assert.Equal(t, mainValidator, result)
	})

	t.Run("empty partial pkg name returns main validator", func(t *testing.T) {
		t.Parallel()

		mainValidator := NewPKValidator("console.log('hello')", "/main.pk")
		aa := &AttributeAnalyser{
			pkValidators:      map[string]*PKValidator{"main": mainValidator},
			mainComponentHash: "main",
		}

		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "",
		}

		result := aa.getValidatorForContext(pInfo)

		assert.Equal(t, mainValidator, result)
	})

	t.Run("unknown partial without component returns nil", func(t *testing.T) {
		t.Parallel()

		vm := createMinimalVirtualModule()
		resolver := NewTypeResolver(nil, vm, nil)
		aa := &AttributeAnalyser{
			typeResolver:      resolver,
			pkValidators:      map[string]*PKValidator{},
			mainComponentHash: "main",
		}

		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "unknown_partial",
		}

		result := aa.getValidatorForContext(pInfo)

		assert.Nil(t, result)
	})
}

func TestAttributeAnalyser_MarkPartialRendered(t *testing.T) {
	t.Parallel()

	t.Run("nil analyser does not panic", func(t *testing.T) {
		t.Parallel()

		var aa *AttributeAnalyser

		aa.MarkPartialRendered("some-alias")
	})

	t.Run("empty alias is ignored", func(t *testing.T) {
		t.Parallel()

		aa := &AttributeAnalyser{
			pkValidators:      map[string]*PKValidator{},
			mainComponentHash: "main",
		}

		aa.MarkPartialRendered("")
	})
}

func TestValidateClassAttribute_NonClassType(t *testing.T) {
	t.Parallel()

	t.Run("integer type produces error", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		attr := &ast_domain.DynamicAttribute{
			Name:          "class",
			RawExpression: "state.Count",
			Location:      ast_domain.Location{Line: 2, Column: 5},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("int"),
				},
			},
		}

		validateClassAttribute(ctx, attr)

		require.Equal(t, 1, len(*ctx.Diagnostics))
		assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "int")
	})

	t.Run("string type produces no error", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		attr := &ast_domain.DynamicAttribute{
			Name:          "class",
			RawExpression: "state.ClassName",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
		}

		validateClassAttribute(ctx, attr)

		assert.Equal(t, 0, len(*ctx.Diagnostics))
	})

	t.Run("skips when error diagnostic already exists for same location", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		loc := ast_domain.Location{Line: 4, Column: 8}
		rawExpr := "state.Invalid"

		*ctx.Diagnostics = append(*ctx.Diagnostics, &ast_domain.Diagnostic{
			Severity:   ast_domain.Error,
			Message:    "pre-existing error",
			Location:   loc,
			Expression: rawExpr,
		})

		attr := &ast_domain.DynamicAttribute{
			Name:          "class",
			RawExpression: rawExpr,
			Location:      loc,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("int"),
				},
			},
		}

		validateClassAttribute(ctx, attr)

		assert.Equal(t, 1, len(*ctx.Diagnostics))
		assert.Equal(t, "pre-existing error", (*ctx.Diagnostics)[0].Message)
	})
}

func TestValidateKeyDirective(t *testing.T) {
	t.Parallel()

	t.Run("complex type produces warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveKey,
			RawExpression: "state.User",
			Expression: &ast_domain.Identifier{
				Name: "user",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("User"),
					},
				},
			},
		}

		validateKeyDirective(directive, ctx)

		require.Equal(t, 1, len(*ctx.Diagnostics))
		assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "complex type")
	})

	t.Run("primitive type produces no warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveKey,
			RawExpression: "item.ID",
			Expression: &ast_domain.Identifier{
				Name: "id",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				},
			},
		}

		validateKeyDirective(directive, ctx)

		assert.Equal(t, 0, len(*ctx.Diagnostics))
	})
}

func TestValidateContextDirective(t *testing.T) {
	t.Parallel()

	t.Run("non-string type produces warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveContext,
			RawExpression: "state.Count",
			Expression: &ast_domain.Identifier{
				Name: "count",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				},
			},
		}

		validateContextDirective(directive, ctx)

		require.Equal(t, 1, len(*ctx.Diagnostics))
		assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "string")
	})

	t.Run("string type produces no warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveContext,
			RawExpression: "state.Key",
			Expression: &ast_domain.Identifier{
				Name: "key",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("string"),
					},
				},
			},
		}

		validateContextDirective(directive, ctx)

		assert.Equal(t, 0, len(*ctx.Diagnostics))
	})
}

func TestValidateStyleDirective(t *testing.T) {
	t.Parallel()

	t.Run("invalid type produces warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveStyle,
			RawExpression: "state.Count",
			Expression: &ast_domain.Identifier{
				Name: "count",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				},
			},
		}

		validateStyleDirective(directive, ctx)

		require.Equal(t, 1, len(*ctx.Diagnostics))
		assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
	})

	t.Run("string type produces no warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		directive := &ast_domain.Directive{
			Type:          ast_domain.DirectiveStyle,
			RawExpression: "state.Style",
			Expression: &ast_domain.Identifier{
				Name: "style",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("string"),
					},
				},
			},
		}

		validateStyleDirective(directive, ctx)

		assert.Equal(t, 0, len(*ctx.Diagnostics))
	})
}

func TestRejectEventPlaceholderInDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		directive       *ast_domain.Directive
		name            string
		wantDiagMessage string
		wantReject      bool
	}{
		{
			name:       "nil directive returns false",
			directive:  nil,
			wantReject: false,
		},
		{
			name: "nil expression returns false",
			directive: &ast_domain.Directive{
				Expression: nil,
			},
			wantReject: false,
		},
		{
			name: "directive with $event returns true",
			directive: &ast_domain.Directive{
				RawExpression: "$event",
				Expression:    &ast_domain.Identifier{Name: "$event"},
			},
			wantReject:      true,
			wantDiagMessage: "$event can only be used in p-on or p-event handlers",
		},
		{
			name: "directive without $event returns false",
			directive: &ast_domain.Directive{
				RawExpression: "state.Active",
				Expression:    &ast_domain.Identifier{Name: "state"},
			},
			wantReject: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createTestContext()

			result := rejectEventPlaceholderInDirective(tc.directive, ctx)

			assert.Equal(t, tc.wantReject, result)
			if tc.wantDiagMessage != "" {
				require.Equal(t, 1, len(*ctx.Diagnostics))
				assert.Equal(t, tc.wantDiagMessage, (*ctx.Diagnostics)[0].Message)
			}
		})
	}
}

func TestRejectFormPlaceholderInDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		directive       *ast_domain.Directive
		name            string
		wantDiagMessage string
		wantReject      bool
	}{
		{
			name:       "nil directive returns false",
			directive:  nil,
			wantReject: false,
		},
		{
			name: "directive with $form returns true",
			directive: &ast_domain.Directive{
				RawExpression: "$form",
				Expression:    &ast_domain.Identifier{Name: "$form"},
			},
			wantReject:      true,
			wantDiagMessage: "$form can only be used in p-on or p-event handlers",
		},
		{
			name: "directive without $form returns false",
			directive: &ast_domain.Directive{
				RawExpression: "state.Name",
				Expression:    &ast_domain.Identifier{Name: "state"},
			},
			wantReject: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createTestContext()

			result := rejectFormPlaceholderInDirective(tc.directive, ctx)

			assert.Equal(t, tc.wantReject, result)
			if tc.wantDiagMessage != "" {
				require.Equal(t, 1, len(*ctx.Diagnostics))
				assert.Equal(t, tc.wantDiagMessage, (*ctx.Diagnostics)[0].Message)
			}
		})
	}
}

func TestCreateEventContextWithSymbols(t *testing.T) {
	t.Parallel()

	t.Run("defines $event and $form symbols", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()

		eventCtx := createEventContextWithSymbols(ctx)

		require.NotNil(t, eventCtx)

		eventSym, found := eventCtx.Symbols.Find("$event")
		require.True(t, found)
		assert.Equal(t, "$event", eventSym.Name)
		require.NotNil(t, eventSym.TypeInfo)
		identifier, ok := eventSym.TypeInfo.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "js.Event", identifier.Name)

		formSym, found := eventCtx.Symbols.Find("$form")
		require.True(t, found)
		assert.Equal(t, "$form", formSym.Name)
		require.NotNil(t, formSym.TypeInfo)
		formIdent, ok := formSym.TypeInfo.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "pk.FormData", formIdent.Name)
	})
}

func TestValidateConditionalDirective_NonBoolType(t *testing.T) {
	t.Parallel()

	t.Run("integer type produces error", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		expression := &ast_domain.Identifier{Name: "count"}
		expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
		})

		d := &ast_domain.Directive{
			Type:          ast_domain.DirectiveIf,
			RawExpression: "count",
			Expression:    expression,
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateConditionalDirective(d, ctx)

		require.Len(t, *ctx.Diagnostics, 1)
		assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "must be a boolean")
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "int")
	})

	t.Run("bool type produces no error", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		expression := &ast_domain.Identifier{Name: "isActive"}
		expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("bool"),
			},
		})

		d := &ast_domain.Directive{
			Type:          ast_domain.DirectiveIf,
			RawExpression: "isActive",
			Expression:    expression,
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateConditionalDirective(d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("nil resolved type produces no error", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		expression := &ast_domain.Identifier{Name: "unknown"}
		expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: nil,
		})

		d := &ast_domain.Directive{
			Type:          ast_domain.DirectiveIf,
			RawExpression: "unknown",
			Expression:    expression,
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateConditionalDirective(d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("nil type expr produces no error", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		expression := &ast_domain.Identifier{Name: "x"}
		expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: nil,
			},
		})

		d := &ast_domain.Directive{
			Type:          ast_domain.DirectiveIf,
			RawExpression: "x",
			Expression:    expression,
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateConditionalDirective(d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("string type produces error", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		expression := &ast_domain.Identifier{Name: "name"}
		expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		})

		d := &ast_domain.Directive{
			Type:          ast_domain.DirectiveIf,
			RawExpression: "name",
			Expression:    expression,
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateConditionalDirective(d, ctx)

		require.Len(t, *ctx.Diagnostics, 1)
		assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "string")
	})
}

func TestValidateConditionalDirective_RejectsEventPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveIf,
		RawExpression: "$event",
		Expression:    &ast_domain.Identifier{Name: "$event"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateConditionalDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$event can only be used in p-on or p-event handlers")
}

func TestValidateConditionalDirective_RejectsFormPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveIf,
		RawExpression: "$form",
		Expression:    &ast_domain.Identifier{Name: "$form"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateConditionalDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$form can only be used in p-on or p-event handlers")
}

func TestValidateModelDirective_RejectsEventPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveModel,
		RawExpression: "$event",
		Expression:    &ast_domain.Identifier{Name: "$event"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateModelDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$event can only be used in p-on or p-event handlers")
}

func TestValidateModelDirective_RejectsFormPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveModel,
		RawExpression: "$form",
		Expression:    &ast_domain.Identifier{Name: "$form"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateModelDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$form can only be used in p-on or p-event handlers")
}

func TestValidateModelDirective_IndexExprIsValid(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveModel,
		RawExpression: "items[0]",
		Expression: &ast_domain.IndexExpression{
			Base:  &ast_domain.Identifier{Name: "items"},
			Index: &ast_domain.IntegerLiteral{Value: 0},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateModelDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateModelDirective_NilExpression(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveModel,
		RawExpression: "",
		Expression:    nil,
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateModelDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateModelDirective_CallExprIsInvalid(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveModel,
		RawExpression: "getValue()",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "getValue"},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateModelDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "assignable variable")
}

func TestValidateClassDirective_ValidTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr goast.Expr
		name     string
	}{
		{
			name:     "string type",
			typeExpr: goast.NewIdent("string"),
		},
		{
			name:     "slice type",
			typeExpr: &goast.ArrayType{Elt: goast.NewIdent("string")},
		},
		{
			name: "map[string]bool type",
			typeExpr: &goast.MapType{
				Key:   goast.NewIdent("string"),
				Value: goast.NewIdent("bool"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createTestContext()
			expression := &ast_domain.Identifier{Name: "classes"}
			expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: tc.typeExpr,
				},
			})

			d := &ast_domain.Directive{
				Type:          ast_domain.DirectiveClass,
				RawExpression: "classes",
				Expression:    expression,
				Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			}

			validateClassDirective(d, ctx)

			assert.Empty(t, *ctx.Diagnostics)
		})
	}
}

func TestValidateClassDirective_InvalidType(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	expression := &ast_domain.Identifier{Name: "count"}
	expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		},
	})

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveClass,
		RawExpression: "count",
		Expression:    expression,
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateClassDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "int")
}

func TestValidateClassDirective_RejectsEventPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveClass,
		RawExpression: "$event",
		Expression:    &ast_domain.Identifier{Name: "$event"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateClassDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$event can only be used in p-on or p-event handlers")
}

func TestValidateClassDirective_RejectsFormPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveClass,
		RawExpression: "$form",
		Expression:    &ast_domain.Identifier{Name: "$form"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateClassDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$form can only be used in p-on or p-event handlers")
}

func TestValidateClassDirective_NilAnnotation(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveClass,
		RawExpression: "classes",
		Expression:    &ast_domain.Identifier{Name: "classes"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateClassDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateStyleDirective_RejectsEventPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveStyle,
		RawExpression: "$event",
		Expression:    &ast_domain.Identifier{Name: "$event"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateStyleDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$event can only be used in p-on or p-event handlers")
}

func TestValidateStyleDirective_RejectsFormPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveStyle,
		RawExpression: "$form",
		Expression:    &ast_domain.Identifier{Name: "$form"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateStyleDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$form can only be used in p-on or p-event handlers")
}

func TestValidateStyleDirective_MapStringType(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	expression := &ast_domain.Identifier{Name: "styles"}
	expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.MapType{
				Key:   goast.NewIdent("string"),
				Value: goast.NewIdent("string"),
			},
		},
	})

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveStyle,
		RawExpression: "styles",
		Expression:    expression,
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateStyleDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateStyleDirective_NilAnnotation(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveStyle,
		RawExpression: "myStyle",
		Expression:    &ast_domain.Identifier{Name: "myStyle"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateStyleDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateStyleDirective_SliceTypeProducesWarning(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	expression := &ast_domain.Identifier{Name: "styles"}
	expression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
		},
	})

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveStyle,
		RawExpression: "styles",
		Expression:    expression,
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateStyleDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
}

func TestValidateKeyDirective_RejectsEventPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveKey,
		RawExpression: "$event",
		Expression:    &ast_domain.Identifier{Name: "$event"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateKeyDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$event can only be used in p-on or p-event handlers")
}

func TestValidateKeyDirective_RejectsFormPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveKey,
		RawExpression: "$form",
		Expression:    &ast_domain.Identifier{Name: "$form"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateKeyDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$form can only be used in p-on or p-event handlers")
}

func TestValidateKeyDirective_NilAnnotation(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveKey,
		RawExpression: "myKey",
		Expression:    &ast_domain.Identifier{Name: "myKey"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateKeyDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateContextDirective_RejectsEventPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveContext,
		RawExpression: "$event",
		Expression:    &ast_domain.Identifier{Name: "$event"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateContextDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$event can only be used in p-on or p-event handlers")
}

func TestValidateContextDirective_RejectsFormPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveContext,
		RawExpression: "$form",
		Expression:    &ast_domain.Identifier{Name: "$form"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateContextDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$form can only be used in p-on or p-event handlers")
}

func TestValidateContextDirective_NilAnnotation(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveContext,
		RawExpression: "myContext",
		Expression:    &ast_domain.Identifier{Name: "myContext"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateContextDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateEventDirective_NilExpression(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "",
		Expression:    nil,
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateEventDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateEventDirective_EventPropertyAccess(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "handler($event.target)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handler"},
			Args: []ast_domain.Expression{
				&ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "$event"},
					Property: &ast_domain.Identifier{Name: "target"},
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateEventDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$event property access is not supported")
}

func TestValidateEventDirective_FormPropertyAccess(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "handler($form.data)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handler"},
			Args: []ast_domain.Expression{
				&ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "$form"},
					Property: &ast_domain.Identifier{Name: "data"},
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateEventDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$form property access is not supported")
}

func TestValidateEventDirective_LegacyEventIdentifier(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "handler(event)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handler"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "event"},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateEventDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "use $event instead of event")
}

func TestValidateEventDirective_ValidCallWithEventArg(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "handler($event)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handler"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$event"},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateEventDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateEventDirective_ValidCallWithFormArg(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "submit($form)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "submit"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$form"},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateEventDirective(d, ctx)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateEventDirective_MemberExprIsInvalid(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "state.handler",
		Expression: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "state"},
			Property: &ast_domain.Identifier{Name: "handler"},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateEventDirective(d, ctx)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "must be a function or method call")
}

func TestValidateDynamicAttribute_EventPlaceholder(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

	attr := &ast_domain.DynamicAttribute{
		Name:          "title",
		RawExpression: "$event",
		Expression:    &ast_domain.Identifier{Name: "$event"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.validateDynamicAttribute(ctx, attr)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "$event can only be used in p-on or p-event handlers")
}

func TestValidateDynamicAttribute_ClassAttributeValidation(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

	attr := &ast_domain.DynamicAttribute{
		Name:          "class",
		RawExpression: "state.Count",
		Expression:    &ast_domain.Identifier{Name: "count"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
		},
	}

	aa.validateDynamicAttribute(ctx, attr)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "Invalid type for :class binding")
}

func TestValidateDynamicAttribute_ServerPrefix(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

	attr := &ast_domain.DynamicAttribute{
		Name:          "server.data",
		RawExpression: "state.Data",
		Expression:    &ast_domain.Identifier{Name: "data"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			Stringability:         0,
			IsPointerToStringable: false,
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyStruct"),
			},
		},
	}

	aa.validateDynamicAttribute(ctx, attr)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateDynamicAttribute_RequestPrefix(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

	attr := &ast_domain.DynamicAttribute{
		Name:          "request.header",
		RawExpression: "state.Header",
		Expression:    &ast_domain.Identifier{Name: "header"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			Stringability:         0,
			IsPointerToStringable: false,
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyStruct"),
			},
		},
	}

	aa.validateDynamicAttribute(ctx, attr)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestExpressionHasDynamicScopeRefs_AdditionalCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		expected   bool
	}{
		{
			name:       "nil literal returns false",
			expression: &ast_domain.NilLiteral{},
			expected:   false,
		},
		{
			name:       "float literal returns false",
			expression: &ast_domain.FloatLiteral{Value: 3.14},
			expected:   false,
		},
		{
			name:       "rune literal returns false",
			expression: &ast_domain.RuneLiteral{Value: 'a'},
			expected:   false,
		},
		{
			name:       "decimal literal returns false",
			expression: &ast_domain.DecimalLiteral{Value: "100.50"},
			expected:   false,
		},
		{
			name:       "big int literal returns false",
			expression: &ast_domain.BigIntLiteral{Value: "999999999999"},
			expected:   false,
		},
		{
			name:       "duration literal returns false",
			expression: &ast_domain.DurationLiteral{Value: "5s"},
			expected:   false,
		},
		{
			name:       "date literal returns false",
			expression: &ast_domain.DateLiteral{Value: "2026-01-01"},
			expected:   false,
		},
		{
			name:       "time literal returns false",
			expression: &ast_domain.TimeLiteral{Value: "12:00:00"},
			expected:   false,
		},
		{
			name:       "datetime literal returns false",
			expression: &ast_domain.DateTimeLiteral{Value: "2026-01-01T12:00:00Z"},
			expected:   false,
		},
		{
			name: "index expression returns true",
			expression: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "items"},
				Index: &ast_domain.IntegerLiteral{Value: 0},
			},
			expected: true,
		},
		{
			name: "call with no arguments returns false",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handler"},
				Args:   nil,
			},
			expected: false,
		},
		{
			name: "call with only $form argument returns false",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "submit"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "$form"}},
			},
			expected: false,
		},
		{
			name: "call with static literal argument returns false",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handler"},
				Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "static"}},
			},
			expected: false,
		},
		{
			name: "nested call with dynamic argument returns true",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "outer"},
				Args: []ast_domain.Expression{
					&ast_domain.CallExpression{
						Callee: &ast_domain.Identifier{Name: "inner"},
						Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "item"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "call with mixed static and dynamic arguments returns true",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handler"},
				Args: []ast_domain.Expression{
					&ast_domain.StringLiteral{Value: "static"},
					&ast_domain.Identifier{Name: "item"},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := expressionHasDynamicScopeRefs(tc.expression)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAttributeAnalyser_analyseEventDirective_DefaultModifier(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "handleClick()",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handleClick"},
		},
		Location: ast_domain.Location{Line: 1, Column: 10, Offset: 9},
	}

	aa.analyseEventDirective(context.Background(), node, d, ctx, nil)

	assert.NotNil(t, d.GoAnnotations)
}

func TestAttributeAnalyser_analyseEventDirective_SetsIsStaticEvent(t *testing.T) {
	t.Parallel()

	t.Run("static event with no dynamic refs", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		mockInsp := &inspector_domain.MockTypeQuerier{}
		contextManager := newContextManager(nil, vm)
		resolver := NewTypeResolver(mockInsp, vm, nil)
		aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

		node := &ast_domain.TemplateNode{
			TagName:  "button",
			NodeType: ast_domain.NodeElement,
		}

		d := &ast_domain.Directive{
			Type:          ast_domain.DirectiveOn,
			RawExpression: "handleClick($event)",
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handleClick"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "$event"}},
			},
			Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		aa.analyseEventDirective(context.Background(), node, d, ctx, nil)

		assert.True(t, d.IsStaticEvent)
	})

	t.Run("dynamic event with scope refs", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		mockInsp := &inspector_domain.MockTypeQuerier{}
		contextManager := newContextManager(nil, vm)
		resolver := NewTypeResolver(mockInsp, vm, nil)
		aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

		node := &ast_domain.TemplateNode{
			TagName:  "button",
			NodeType: ast_domain.NodeElement,
		}

		d := &ast_domain.Directive{
			Type:          ast_domain.DirectiveOn,
			RawExpression: "handleClick(item.id)",
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handleClick"},
				Args: []ast_domain.Expression{
					&ast_domain.MemberExpression{
						Base:     &ast_domain.Identifier{Name: "item"},
						Property: &ast_domain.Identifier{Name: "id"},
					},
				},
			},
			Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		aa.analyseEventDirective(context.Background(), node, d, ctx, nil)

		assert.False(t, d.IsStaticEvent)
	})
}

func TestAttributeAnalyser_analyseEventDirectives_EmptyMap(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
	}

	aa.analyseEventDirectives(context.Background(), node, nil, ctx, nil)
	aa.analyseEventDirectives(context.Background(), node, map[string][]ast_domain.Directive{}, ctx, nil)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestAttributeAnalyser_getValidatorForContext_CachedPartial(t *testing.T) {
	t.Parallel()

	partialValidator := NewPKValidator("export function click() {}", "/partial.pk")
	aa := &AttributeAnalyser{
		pkValidators: map[string]*PKValidator{
			"main":        NewPKValidator("console.log('main')", "/main.pk"),
			"partial_pkg": partialValidator,
		},
		mainComponentHash: "main",
	}

	pInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName: "partial_pkg",
	}

	result := aa.getValidatorForContext(pInfo)

	assert.Same(t, partialValidator, result)
}

func TestAttributeAnalyser_getValidatorForContext_PartialWithClientScript(t *testing.T) {
	t.Parallel()

	vm := createMinimalVirtualModule()
	vm.ComponentsByHash["partial_pkg"] = &annotator_dto.VirtualComponent{
		HashedName: "partial_pkg",
		Source: &annotator_dto.ParsedComponent{
			ClientScript: "export function partialClick() {}",
			SourcePath:   "/partial.pk",
		},
	}
	resolver := NewTypeResolver(nil, vm, nil)

	aa := &AttributeAnalyser{
		typeResolver:      resolver,
		pkValidators:      map[string]*PKValidator{},
		mainComponentHash: "main",
	}

	pInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName: "partial_pkg",
	}

	result := aa.getValidatorForContext(pInfo)

	require.NotNil(t, result)
	assert.Contains(t, aa.pkValidators, "partial_pkg")
}

func TestAttributeAnalyser_getValidatorForContext_PartialWithoutClientScript(t *testing.T) {
	t.Parallel()

	vm := createMinimalVirtualModule()
	vm.ComponentsByHash["partial_pkg"] = &annotator_dto.VirtualComponent{
		HashedName: "partial_pkg",
		Source: &annotator_dto.ParsedComponent{
			ClientScript: "",
			SourcePath:   "/partial.pk",
		},
	}
	resolver := NewTypeResolver(nil, vm, nil)

	aa := &AttributeAnalyser{
		typeResolver:      resolver,
		pkValidators:      map[string]*PKValidator{},
		mainComponentHash: "main",
	}

	pInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName: "partial_pkg",
	}

	result := aa.getValidatorForContext(pInfo)

	assert.Nil(t, result)
}

func TestAttributeContextResolver_ForAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("nil annotation returns default context", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		partialCtx := createTestContext()
		r := &attributeContextResolver{
			aa:                 &AttributeAnalyser{},
			ctx:                ctx,
			partialSelfContext: partialCtx,
		}

		result := r.forAnnotation(nil)

		assert.Same(t, partialCtx, result)
	})

	t.Run("annotation with nil pkg alias returns default context", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		r := &attributeContextResolver{
			aa:  &AttributeAnalyser{},
			ctx: ctx,
		}

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalPackageAlias: nil,
			OriginalSourcePath:   new("/test.pk"),
		}

		result := r.forAnnotation(ann)

		assert.Same(t, ctx, result)
	})

	t.Run("annotation with nil source path returns default context", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		r := &attributeContextResolver{
			aa:  &AttributeAnalyser{},
			ctx: ctx,
		}

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalPackageAlias: new("pkg"),
			OriginalSourcePath:   nil,
		}

		result := r.forAnnotation(ann)

		assert.Same(t, ctx, result)
	})

	t.Run("annotation matching current SFC path returns ctx", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		r := &attributeContextResolver{
			aa:  &AttributeAnalyser{},
			ctx: ctx,
		}

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalPackageAlias: new("pkg"),
			OriginalSourcePath:   new(ctx.SFCSourcePath),
		}

		result := r.forAnnotation(ann)

		assert.Same(t, ctx, result)
	})
}

func TestAttributeContextResolver_ForDirective(t *testing.T) {
	t.Parallel()

	t.Run("nil directive returns ctx", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		r := &attributeContextResolver{
			aa:  &AttributeAnalyser{},
			ctx: ctx,
		}

		result := r.forDirective(nil)

		assert.Same(t, ctx, result)
	})

	t.Run("directive with nil annotations returns default", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		partialCtx := createTestContext()
		r := &attributeContextResolver{
			aa:                 &AttributeAnalyser{},
			ctx:                ctx,
			partialSelfContext: partialCtx,
		}

		d := &ast_domain.Directive{
			GoAnnotations: nil,
		}

		result := r.forDirective(d)

		assert.Same(t, partialCtx, result)
	})
}

func TestAttributeContextResolver_DefaultContext(t *testing.T) {
	t.Parallel()

	t.Run("returns partial self context when set", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		partialCtx := createTestContext()
		r := &attributeContextResolver{
			aa:                 &AttributeAnalyser{},
			ctx:                ctx,
			partialSelfContext: partialCtx,
		}

		result := r.defaultContext()

		assert.Same(t, partialCtx, result)
	})

	t.Run("returns ctx when partial self context is nil", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		r := &attributeContextResolver{
			aa:                 &AttributeAnalyser{},
			ctx:                ctx,
			partialSelfContext: nil,
		}

		result := r.defaultContext()

		assert.Same(t, ctx, result)
	})
}

func TestAttributeAnalyser_ResolveObjectLiteralValues_ObjectLiteral(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

	objLit := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{
			"active": &ast_domain.Identifier{Name: "isActive"},
		},
	}
	loc := ast_domain.Location{Line: 1, Column: 1, Offset: 0}

	aa.resolveObjectLiteralValues(context.Background(), ctx, objLit, loc)
}

func TestNewAttributeAnalyser_WithValidator(t *testing.T) {
	t.Parallel()

	pkValidator := NewPKValidator("export function click() {}", "/test.pk")
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "main_hash", pkValidator)

	require.NotNil(t, aa)
	assert.Contains(t, aa.pkValidators, "main_hash")
	assert.Same(t, pkValidator, aa.pkValidators["main_hash"])
}

func TestNewAttributeAnalyser_WithoutValidator(t *testing.T) {
	t.Parallel()

	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "main_hash", nil)

	require.NotNil(t, aa)
	assert.Empty(t, aa.pkValidators)
}

func TestNewAttributeAnalyser_EmptyComponentHash(t *testing.T) {
	t.Parallel()

	pkValidator := NewPKValidator("export function click() {}", "/test.pk")
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", pkValidator)

	require.NotNil(t, aa)
	assert.Empty(t, aa.pkValidators)
}

func TestAttributeAnalyser_MarkPartialRendered_ValidAlias(t *testing.T) {
	t.Parallel()

	mainValidator := NewPKValidator("export function handleClick() {}", "/main.pk")
	mainValidator.RegisterImportedPartials([]string{"card"})

	aa := &AttributeAnalyser{
		pkValidators:      map[string]*PKValidator{"main_hash": mainValidator},
		mainComponentHash: "main_hash",
	}

	aa.MarkPartialRendered("card")
}

func TestAttributeAnalyser_MarkPartialRendered_NoMainValidator(t *testing.T) {
	t.Parallel()

	aa := &AttributeAnalyser{
		pkValidators:      map[string]*PKValidator{},
		mainComponentHash: "main_hash",
	}

	assert.NotPanics(t, func() {
		aa.MarkPartialRendered("card")
	})
}

func TestAttributeAnalyser_analyseOtherDirectives_AllNil(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
	}

	r := &attributeContextResolver{
		aa:  aa,
		ctx: ctx,
	}

	aa.analyseOtherDirectives(context.Background(), node, r)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestAttributeAnalyser_analyseNodeKey_NilKey(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
		Key:      nil,
	}

	r := &attributeContextResolver{
		aa:  aa,
		ctx: ctx,
	}

	aa.analyseNodeKey(context.Background(), node, r)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestAttributeAnalyser_analyseNodeKey_WithKey(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "li",
		NodeType: ast_domain.NodeElement,
		Key:      &ast_domain.Identifier{Name: "itemKey"},
		Location: ast_domain.Location{Line: 5, Column: 1, Offset: 0},
	}

	r := &attributeContextResolver{
		aa:  aa,
		ctx: ctx,
	}

	aa.analyseNodeKey(context.Background(), node, r)
}

func TestAttributeAnalyser_resolveClientEventHandlerArgs_NilDirective(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

	aa.resolveClientEventHandlerArgs(context.Background(), nil, ctx, nil)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestAttributeAnalyser_resolveClientEventHandlerArgs_NilExpression(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

	d := &ast_domain.Directive{
		Type:       ast_domain.DirectiveOn,
		Expression: nil,
	}

	aa.resolveClientEventHandlerArgs(context.Background(), d, ctx, nil)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestAttributeAnalyser_resolveClientEventHandlerArgs_CallExpr(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "", nil)

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "handler($event)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handler"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$event"},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.resolveClientEventHandlerArgs(context.Background(), d, h.Context, nil)

	require.NotNil(t, d.GoAnnotations)
}

func TestAttributeAnalyser_resolveClientEventHandlerArgs_NonCallExpr(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "someIdentifier",
		Expression:    &ast_domain.Identifier{Name: "someIdentifier"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.resolveClientEventHandlerArgs(context.Background(), d, ctx, nil)

	require.NotNil(t, d.GoAnnotations)
}

func TestAttributeAnalyser_analyseClassAndStyleDirectives(t *testing.T) {
	t.Parallel()

	t.Run("both nil directives", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		resolver := NewTypeResolver(nil, vm, nil)
		aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

		node := &ast_domain.TemplateNode{
			TagName:  "div",
			NodeType: ast_domain.NodeElement,
		}

		r := &attributeContextResolver{
			aa:  aa,
			ctx: ctx,
		}

		aa.analyseClassAndStyleDirectives(context.Background(), node, r)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("with class directive", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		resolver := NewTypeResolver(nil, vm, nil)
		aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

		node := &ast_domain.TemplateNode{
			TagName:  "div",
			NodeType: ast_domain.NodeElement,
			DirClass: &ast_domain.Directive{
				Type:          ast_domain.DirectiveClass,
				RawExpression: "classes",
				Expression:    &ast_domain.Identifier{Name: "classes"},
				Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		}

		r := &attributeContextResolver{
			aa:  aa,
			ctx: ctx,
		}

		aa.analyseClassAndStyleDirectives(context.Background(), node, r)
	})

	t.Run("with style directive", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		resolver := NewTypeResolver(nil, vm, nil)
		aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

		node := &ast_domain.TemplateNode{
			TagName:  "div",
			NodeType: ast_domain.NodeElement,
			DirStyle: &ast_domain.Directive{
				Type:          ast_domain.DirectiveStyle,
				RawExpression: "myStyles",
				Expression:    &ast_domain.Identifier{Name: "myStyles"},
				Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		}

		r := &attributeContextResolver{
			aa:  aa,
			ctx: ctx,
		}

		aa.analyseClassAndStyleDirectives(context.Background(), node, r)
	})
}

func TestAttributeAnalyser_analyseBindAndModelDirectives(t *testing.T) {
	t.Parallel()

	t.Run("with binds map", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		resolver := NewTypeResolver(nil, vm, nil)
		aa := newAttributeAnalyser(resolver, nil, contextManager, "", nil)

		node := &ast_domain.TemplateNode{
			TagName:  "input",
			NodeType: ast_domain.NodeElement,
			Binds: map[string]*ast_domain.Directive{
				"value": {
					Type:          ast_domain.DirectiveBind,
					RawExpression: "state.Name",
					Expression:    &ast_domain.Identifier{Name: "name"},
					Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				},
			},
		}

		r := &attributeContextResolver{
			aa:  aa,
			ctx: ctx,
		}

		aa.analyseBindAndModelDirectives(context.Background(), node, r, ctx)
	})
}

func TestIsStyleBindingType_AdditionalCases(t *testing.T) {
	t.Parallel()

	t.Run("nil type expr returns false", func(t *testing.T) {
		t.Parallel()

		result := isStyleBindingType(&ast_domain.ResolvedTypeInfo{TypeExpression: nil})

		assert.False(t, result)
	})

	t.Run("int type returns false", func(t *testing.T) {
		t.Parallel()

		result := isStyleBindingType(&ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")})

		assert.False(t, result)
	})

	t.Run("bool type returns false", func(t *testing.T) {
		t.Parallel()

		result := isStyleBindingType(&ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")})

		assert.False(t, result)
	})
}

func TestIsClassBindingType_AdditionalCases(t *testing.T) {
	t.Parallel()

	t.Run("map[int]string returns false", func(t *testing.T) {
		t.Parallel()

		result := isClassBindingType(&ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.MapType{
				Key:   goast.NewIdent("int"),
				Value: goast.NewIdent("string"),
			},
		})

		assert.False(t, result)
	})

	t.Run("bool type returns false", func(t *testing.T) {
		t.Parallel()

		result := isClassBindingType(&ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")})

		assert.False(t, result)
	})
}

func TestIsComplexType_AdditionalCases(t *testing.T) {
	t.Parallel()

	t.Run("float64 is not complex", func(t *testing.T) {
		t.Parallel()

		result := isComplexType(&ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")})

		assert.False(t, result)
	})

	t.Run("byte is not complex", func(t *testing.T) {
		t.Parallel()

		result := isComplexType(&ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("byte")})

		assert.False(t, result)
	})

	t.Run("map type is complex", func(t *testing.T) {
		t.Parallel()

		result := isComplexType(&ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.MapType{
				Key:   goast.NewIdent("string"),
				Value: goast.NewIdent("int"),
			},
		})

		assert.True(t, result)
	})

	t.Run("array type is complex", func(t *testing.T) {
		t.Parallel()

		result := isComplexType(&ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("int")},
		})

		assert.True(t, result)
	})
}

func TestValidateClassAttribute_StringType(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	attr := &ast_domain.DynamicAttribute{
		Name:          "class",
		RawExpression: "state.ClassName",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		},
	}

	validateClassAttribute(ctx, attr)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateClassAttribute_SliceType(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	attr := &ast_domain.DynamicAttribute{
		Name:          "class",
		RawExpression: "state.ClassList",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
		},
	}

	validateClassAttribute(ctx, attr)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateClassAttribute_MapType(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	attr := &ast_domain.DynamicAttribute{
		Name:          "class",
		RawExpression: "state.ClassMap",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{
					Key:   goast.NewIdent("string"),
					Value: goast.NewIdent("bool"),
				},
			},
		},
	}

	validateClassAttribute(ctx, attr)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateClassAttribute_NilResolvedType(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	attr := &ast_domain.DynamicAttribute{
		Name:          "class",
		RawExpression: "state.Class",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: nil,
		},
	}

	validateClassAttribute(ctx, attr)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestContainsEventPlaceholder_NilExpression(t *testing.T) {
	t.Parallel()

	result := containsEventPlaceholder(nil)
	assert.False(t, result)
}

func TestContainsFormPlaceholder_NilExpression(t *testing.T) {
	t.Parallel()

	result := containsFormPlaceholder(nil)
	assert.False(t, result)
}

func TestFindEventPropertyAccess_NilExpression(t *testing.T) {
	t.Parallel()

	result := findEventPropertyAccess(nil)
	assert.Nil(t, result)
}

func TestFindFormPropertyAccess_NilExpression(t *testing.T) {
	t.Parallel()

	result := findFormPropertyAccess(nil)
	assert.Nil(t, result)
}

func TestFindLegacyEventIdentifier_NilExpression(t *testing.T) {
	t.Parallel()

	result := findLegacyEventIdentifier(nil)
	assert.Nil(t, result)
}

func TestAttributeAnalyser_isActionCall_EmptyExpression(t *testing.T) {
	t.Parallel()

	aa := &AttributeAnalyser{}
	d := &ast_domain.Directive{RawExpression: ""}

	result := aa.isActionCall(d)

	assert.False(t, result)
}

func TestExtractActionNameFromDirective_NoArgs(t *testing.T) {
	t.Parallel()

	d := &ast_domain.Directive{RawExpression: "action.namespace.Name"}

	result := extractActionNameFromDirective(d)

	assert.Equal(t, "namespace.Name", result)
}

func TestExtractActionNameFromDirective_WithNestedParens(t *testing.T) {
	t.Parallel()

	d := &ast_domain.Directive{RawExpression: "action.ns.Method(foo(bar))"}

	result := extractActionNameFromDirective(d)

	assert.Equal(t, "ns.Method", result)
}

func TestBuildOriginContext_UsesPartialInvocationMap(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := h.Resolver.virtualModule

	originVC := &annotator_dto.VirtualComponent{
		HashedName:             "origin_hash",
		CanonicalGoPackagePath: "origin/pkg",
		VirtualGoFilePath:      "/origin_virtual.go",
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/origin.pk",
			Script:     &annotator_dto.ParsedScript{},
		},
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("originpkg"),
		},
	}
	vm.ComponentsByHash["origin_hash"] = originVC

	pInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName: "origin_hash",
		InvocationKey:      "inv_key_1",
	}

	r := &attributeContextResolver{
		aa:  &AttributeAnalyser{typeResolver: h.Resolver},
		ctx: h.Context,
		partialInvocationMap: map[string]*ast_domain.PartialInvocationInfo{
			"origin_hash": pInfo,
		},
	}

	result := r.buildOriginContext(originVC, "/origin.pk")

	require.NotNil(t, result)
	assert.Equal(t, "origin/pkg", result.CurrentGoFullPackagePath)
	assert.Equal(t, "originpkg", result.CurrentGoPackageName)
	assert.Equal(t, "/origin_virtual.go", result.CurrentGoSourcePath)
	assert.Equal(t, "/origin.pk", result.SFCSourcePath)
}

func TestBuildOriginContext_UsesActivePInfoWhenHashMatches(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	originVC := &annotator_dto.VirtualComponent{
		HashedName:             "partial_hash",
		CanonicalGoPackagePath: "partial/pkg",
		VirtualGoFilePath:      "/partial_virtual.go",
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/partial.pk",
			Script:     &annotator_dto.ParsedScript{},
		},
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("partialpkg"),
		},
	}

	activePInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName: "partial_hash",
		InvocationKey:      "active_key",
	}

	r := &attributeContextResolver{
		aa:                   &AttributeAnalyser{typeResolver: h.Resolver},
		ctx:                  h.Context,
		activePInfo:          activePInfo,
		partialInvocationMap: map[string]*ast_domain.PartialInvocationInfo{},
	}

	result := r.buildOriginContext(originVC, "/partial.pk")

	require.NotNil(t, result)
	assert.Equal(t, "partial/pkg", result.CurrentGoFullPackagePath)
	assert.Equal(t, "/partial.pk", result.SFCSourcePath)
}

func TestBuildOriginContext_FallsBackToPopulateRootContext(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	originVC := &annotator_dto.VirtualComponent{
		HashedName:             "root_hash",
		CanonicalGoPackagePath: "root/pkg",
		VirtualGoFilePath:      "/root_virtual.go",
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/root.pk",
			Script:     &annotator_dto.ParsedScript{},
		},
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("rootpkg"),
		},
	}

	r := &attributeContextResolver{
		aa:                   &AttributeAnalyser{typeResolver: h.Resolver},
		ctx:                  h.Context,
		activePInfo:          nil,
		partialInvocationMap: map[string]*ast_domain.PartialInvocationInfo{},
	}

	result := r.buildOriginContext(originVC, "/root.pk")

	require.NotNil(t, result)
	assert.Equal(t, "root/pkg", result.CurrentGoFullPackagePath)
	assert.Equal(t, "rootpkg", result.CurrentGoPackageName)
}

func TestBuildOriginContext_ActivePInfoDoesNotMatchHash(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	originVC := &annotator_dto.VirtualComponent{
		HashedName:             "origin_hash",
		CanonicalGoPackagePath: "origin/pkg",
		VirtualGoFilePath:      "/origin_virtual.go",
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/origin.pk",
			Script:     &annotator_dto.ParsedScript{},
		},
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("originpkg"),
		},
	}

	activePInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName: "different_hash",
		InvocationKey:      "some_key",
	}

	r := &attributeContextResolver{
		aa:                   &AttributeAnalyser{typeResolver: h.Resolver},
		ctx:                  h.Context,
		activePInfo:          activePInfo,
		partialInvocationMap: map[string]*ast_domain.PartialInvocationInfo{},
	}

	result := r.buildOriginContext(originVC, "/origin.pk")

	require.NotNil(t, result)
	assert.Equal(t, "origin/pkg", result.CurrentGoFullPackagePath)
}

func TestForAnnotation_ReachingBuildOriginContext(t *testing.T) {
	t.Parallel()

	t.Run("annotation with different SFC and matched hash reaches buildOriginContext", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		vm := h.Resolver.virtualModule

		originVC := &annotator_dto.VirtualComponent{
			HashedName:             "origin_comp",
			CanonicalGoPackagePath: "origin/comp/pkg",
			VirtualGoFilePath:      "/origin_comp_virtual.go",
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/origin_comp.pk",
				Script:     &annotator_dto.ParsedScript{},
			},
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("origincomppkg"),
			},
		}
		vm.ComponentsByHash["origin_comp"] = originVC

		r := &attributeContextResolver{
			aa:                   &AttributeAnalyser{typeResolver: h.Resolver},
			ctx:                  h.Context,
			partialInvocationMap: map[string]*ast_domain.PartialInvocationInfo{},
		}

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalPackageAlias: new("origin_comp"),
			OriginalSourcePath:   new("/origin_comp.pk"),
		}

		result := r.forAnnotation(ann)

		require.NotNil(t, result)
		assert.Equal(t, "origin/comp/pkg", result.CurrentGoFullPackagePath)
	})

	t.Run("annotation with matching current VC hashed name returns ctx", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		vm := h.Resolver.virtualModule

		currentVC := &annotator_dto.VirtualComponent{
			HashedName: "current_hash",
		}
		vm.ComponentsByGoPath[h.Context.CurrentGoFullPackagePath] = currentVC

		r := &attributeContextResolver{
			aa:  &AttributeAnalyser{typeResolver: h.Resolver},
			ctx: h.Context,
		}

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalPackageAlias: new("current_hash"),
			OriginalSourcePath:   new("/some_different.pk"),
		}

		result := r.forAnnotation(ann)

		assert.Same(t, h.Context, result)
	})

	t.Run("annotation with unknown hash returns default context", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		r := &attributeContextResolver{
			aa:  &AttributeAnalyser{typeResolver: h.Resolver},
			ctx: h.Context,
		}

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalPackageAlias: new("nonexistent_hash"),
			OriginalSourcePath:   new("/nonexistent.pk"),
		}

		result := r.forAnnotation(ann)

		assert.Same(t, h.Context, result)
	})
}

func TestResolveActionCall_WithKnownAction(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	actions := map[string]ActionInfoProvider{
		"email.send": mockActionInfo{method: "POST"},
	}
	aa := newAttributeAnalyser(h.Resolver, actions, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "action.email.send($form)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.MemberExpression{
				Base: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "action"},
					Property: &ast_domain.Identifier{Name: "email"},
				},
				Property: &ast_domain.Identifier{Name: "send"},
			},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$form"},
			},
		},
		Location: ast_domain.Location{Line: 5, Column: 10, Offset: 50},
	}

	aa.resolveActionCall(context.Background(), node, d, h.Context)

	assert.Equal(t, "action", d.Modifier)
	require.NotNil(t, d.GoAnnotations)
	require.NotNil(t, node.GoAnnotations)
	assert.True(t, node.GoAnnotations.NeedsCSRF)

	callExpr, ok := d.Expression.(*ast_domain.CallExpression)
	require.True(t, ok)
	identifier, ok := callExpr.Callee.(*ast_domain.Identifier)
	require.True(t, ok)
	assert.Equal(t, "email.send", identifier.Name)
}

func TestResolveActionCall_WithUnknownAction(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	aa := newAttributeAnalyser(h.Resolver, map[string]ActionInfoProvider{}, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "action.unknown.handler($form)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "action.unknown.handler"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$form"},
			},
		},
		Location: ast_domain.Location{Line: 3, Column: 5, Offset: 20},
	}

	aa.resolveActionCall(context.Background(), node, d, h.Context)

	assert.Equal(t, "action", d.Modifier)
	require.NotNil(t, d.GoAnnotations)

	require.GreaterOrEqual(t, len(*h.Diagnostics), 1)
	assert.Contains(t, (*h.Diagnostics)[0].Message, "not found for action call")
}

func TestResolveActionCall_WithoutCallExpr(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	actions := map[string]ActionInfoProvider{
		"email.logout": mockActionInfo{method: "GET"},
	}
	aa := newAttributeAnalyser(h.Resolver, actions, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "action.email.logout",
		Expression:    &ast_domain.Identifier{Name: "action.email.logout"},
		Location:      ast_domain.Location{Line: 2, Column: 1, Offset: 10},
	}

	aa.resolveActionCall(context.Background(), node, d, h.Context)

	assert.Equal(t, "action", d.Modifier)
	require.NotNil(t, d.GoAnnotations)
	require.NotNil(t, node.GoAnnotations)
	assert.True(t, node.GoAnnotations.NeedsCSRF)
}

func TestResolveActionCall_CaseInsensitiveMatch(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	actions := map[string]ActionInfoProvider{
		"email.Send": mockActionInfo{method: "PUT"},
	}
	aa := newAttributeAnalyser(h.Resolver, actions, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "action.email.send($form)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "action.email.send"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$form"},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.resolveActionCall(context.Background(), node, d, h.Context)

	assert.Equal(t, "action", d.Modifier)
	require.NotNil(t, node.GoAnnotations)
	assert.True(t, node.GoAnnotations.NeedsCSRF)

	found := false
	for _, attr := range node.Attributes {
		if attr.Name == "data-pk-action-method" && attr.Value == "PUT" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected data-pk-action-method attribute with value PUT")
}

func TestAddCSRFAndMethodAttributes_ActionFound(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	actions := map[string]ActionInfoProvider{
		"email.contact": mockActionInfo{method: "POST"},
	}
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, actions, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "form",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		RawExpression: "action.email.contact($form)",
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.addCSRFAndMethodAttributes(node, d, "email.contact", actions["email.contact"], ctx)

	require.NotNil(t, node.GoAnnotations)
	assert.True(t, node.GoAnnotations.NeedsCSRF)
	require.Len(t, node.Attributes, 1)
	assert.Equal(t, "data-pk-action-method", node.Attributes[0].Name)
	assert.Equal(t, "POST", node.Attributes[0].Value)
	assert.Empty(t, *ctx.Diagnostics)
}

func TestAddCSRFAndMethodAttributes_ActionNotFound(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, map[string]ActionInfoProvider{}, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "form",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		RawExpression: "action.unknown.handler($form)",
		Location:      ast_domain.Location{Line: 4, Column: 8, Offset: 30},
	}

	aa.addCSRFAndMethodAttributes(node, d, "unknown.handler", nil, ctx)

	assert.Nil(t, node.GoAnnotations)
	assert.Empty(t, node.Attributes)
	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "Action 'unknown.handler' not found for action call")
}

func TestAddCSRFAndMethodAttributes_PreservesExistingAnnotations(t *testing.T) {
	t.Parallel()

	ctx := createTestContext()
	actions := map[string]ActionInfoProvider{
		"user.update": mockActionInfo{method: "PATCH"},
	}
	vm := createMinimalVirtualModule()
	contextManager := newContextManager(nil, vm)
	resolver := NewTypeResolver(nil, vm, nil)
	aa := newAttributeAnalyser(resolver, actions, contextManager, "", nil)

	existingAnn := &ast_domain.GoGeneratorAnnotation{
		NeedsCSRF: false,
	}
	node := &ast_domain.TemplateNode{
		TagName:       "button",
		NodeType:      ast_domain.NodeElement,
		GoAnnotations: existingAnn,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "class", Value: "btn"},
		},
	}

	d := &ast_domain.Directive{
		RawExpression: "action.user.update()",
		Location:      ast_domain.Location{Line: 2, Column: 3, Offset: 15},
	}

	aa.addCSRFAndMethodAttributes(node, d, "user.update", actions["user.update"], ctx)

	assert.Same(t, existingAnn, node.GoAnnotations)
	assert.True(t, node.GoAnnotations.NeedsCSRF)
	require.Len(t, node.Attributes, 2)
	assert.Equal(t, "class", node.Attributes[0].Name)
	assert.Equal(t, "data-pk-action-method", node.Attributes[1].Name)
	assert.Equal(t, "PATCH", node.Attributes[1].Value)
}

func TestTransformActionCallExpr_TransformsCallee(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "", nil)

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "action"},
				Property: &ast_domain.Identifier{Name: "email"},
			},
			Property: &ast_domain.Identifier{Name: "send"},
		},
		Args: []ast_domain.Expression{
			&ast_domain.Identifier{Name: "$form"},
		},
	}

	location := ast_domain.Location{Line: 5, Column: 10, Offset: 50}

	aa.transformActionCallExpr(context.Background(), callExpr, "email.send", nil, h.Context, location)

	identifier, ok := callExpr.Callee.(*ast_domain.Identifier)
	require.True(t, ok)
	assert.Equal(t, "email.send", identifier.Name)
	assert.Equal(t, len("email.send"), identifier.SourceLength)
	assert.Nil(t, identifier.GoAnnotations)
}

func TestTransformActionCallExpr_ResolvesArgs(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "", nil)

	formArg := &ast_domain.Identifier{Name: "$form"}
	eventArg := &ast_domain.Identifier{Name: "$event"}

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "action.email.send"},
		Args:   []ast_domain.Expression{formArg, eventArg},
	}

	location := ast_domain.Location{Line: 1, Column: 1, Offset: 0}

	aa.transformActionCallExpr(context.Background(), callExpr, "email.send", nil, h.Context, location)

	identifier, ok := callExpr.Callee.(*ast_domain.Identifier)
	require.True(t, ok)
	assert.Equal(t, "email.send", identifier.Name)
}

func TestTransformActionCallExpr_NoArgs(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "", nil)

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "action.user.logout"},
		Args:   nil,
	}

	location := ast_domain.Location{Line: 1, Column: 1, Offset: 0}

	aa.transformActionCallExpr(context.Background(), callExpr, "user.logout", nil, h.Context, location)

	identifier, ok := callExpr.Callee.(*ast_domain.Identifier)
	require.True(t, ok)
	assert.Equal(t, "user.logout", identifier.Name)
}

func TestResolveDefaultEventDirective_ActionCall(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	actions := map[string]ActionInfoProvider{
		"email.send": mockActionInfo{method: "POST"},
	}
	aa := newAttributeAnalyser(h.Resolver, actions, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "action.email.send($form)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "action.email.send"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$form"},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.resolveDefaultEventDirective(context.Background(), node, d, h.Context, nil)

	assert.Equal(t, "action", d.Modifier)
	require.NotNil(t, d.GoAnnotations)
	require.NotNil(t, node.GoAnnotations)
	assert.True(t, node.GoAnnotations.NeedsCSRF)
}

func TestResolveDefaultEventDirective_WithValidator(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	pkValidator := NewPKValidator("export function handleClick() {}", "/test.pk")
	aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "main_hash", pkValidator)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "handleClick($event)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handleClick"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$event"},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.resolveDefaultEventDirective(context.Background(), node, d, h.Context, aa.pkValidators["main_hash"])

	require.NotNil(t, d.GoAnnotations)
}

func TestResolveDefaultEventDirective_WithNilValidator(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "handleClick()",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handleClick"},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.resolveDefaultEventDirective(context.Background(), node, d, h.Context, nil)

	require.NotNil(t, d.GoAnnotations)
}

func TestAnalyseEventDirective_ActionCallSetsModifierAndStaticEvent(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	actions := map[string]ActionInfoProvider{
		"contact.submit": mockActionInfo{method: "POST"},
	}
	aa := newAttributeAnalyser(h.Resolver, actions, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "form",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		RawExpression: "action.contact.submit($form)",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "action.contact.submit"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$form"},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.analyseEventDirective(context.Background(), node, d, h.Context, nil)

	assert.Equal(t, "action", d.Modifier)
	require.NotNil(t, d.GoAnnotations)

	assert.True(t, d.IsStaticEvent)
}

func TestAnalyseEventDirectives_MultipleEvents(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	eventMap := map[string][]ast_domain.Directive{
		"click": {
			{
				Type:          ast_domain.DirectiveOn,
				RawExpression: "handleClick()",
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "handleClick"},
				},
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		},
		"submit": {
			{
				Type:          ast_domain.DirectiveOn,
				RawExpression: "handleSubmit()",
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "handleSubmit"},
				},
				Location: ast_domain.Location{Line: 2, Column: 1, Offset: 10},
			},
		},
	}

	aa.analyseEventDirectives(context.Background(), node, eventMap, h.Context, nil)

	for _, directives := range eventMap {
		for _, d := range directives {
			assert.NotNil(t, d.GoAnnotations, "Expected GoAnnotations for directive: %s", d.RawExpression)
		}
	}
}

func TestLookupActionCaseInsensitive_NilActionsMap(t *testing.T) {
	t.Parallel()

	aa := &AttributeAnalyser{
		actions: nil,
	}

	result := aa.lookupActionCaseInsensitive("anything")

	assert.Nil(t, result)
}

func TestNewSyntheticTypeInfo_Synthetic(t *testing.T) {
	t.Parallel()

	result := newSyntheticTypeInfo("pk.FormData")

	require.NotNil(t, result)
	assert.True(t, result.IsSynthetic)
	assert.Empty(t, result.PackageAlias)
	assert.Empty(t, result.CanonicalPackagePath)
	assert.False(t, result.IsExportedPackageSymbol)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "pk.FormData", identifier.Name)
}

func TestAnalyseEventDirective_ValidModifiersProduceNoError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		modifiers []string
	}{
		{name: "prevent only", modifiers: []string{"prevent"}},
		{name: "stop only", modifiers: []string{"stop"}},
		{name: "once only", modifiers: []string{"once"}},
		{name: "self only", modifiers: []string{"self"}},
		{name: "passive only", modifiers: []string{"passive"}},
		{name: "capture only", modifiers: []string{"capture"}},
		{name: "prevent and stop", modifiers: []string{"prevent", "stop"}},
		{name: "prevent and once", modifiers: []string{"prevent", "once"}},
		{name: "passive and once", modifiers: []string{"passive", "once"}},
		{name: "capture and stop", modifiers: []string{"capture", "stop"}},
		{name: "capture and passive", modifiers: []string{"capture", "passive"}},
		{name: "all handler modifiers", modifiers: []string{"prevent", "stop", "once", "self"}},
		{name: "all listener modifiers", modifiers: []string{"capture", "passive"}},
		{name: "handler and listener modifiers combined", modifiers: []string{"stop", "once", "capture"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			vm := newTestHarnessVirtualModule()
			contextManager := newContextManager(h.Resolver, vm)
			actions := map[string]ActionInfoProvider{
				"contact.Send": mockActionInfo{method: "POST"},
			}
			aa := newAttributeAnalyser(h.Resolver, actions, contextManager, "", nil)

			node := &ast_domain.TemplateNode{
				TagName:  "form",
				NodeType: ast_domain.NodeElement,
			}

			d := &ast_domain.Directive{
				Type:           ast_domain.DirectiveOn,
				Arg:            "submit",
				EventModifiers: tc.modifiers,
				RawExpression:  "action.contact.Send($form)",
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "action.contact.Send"},
					Args: []ast_domain.Expression{
						&ast_domain.Identifier{Name: "$form"},
					},
				},
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			}

			aa.analyseEventDirective(context.Background(), node, d, h.Context, nil)

			assert.Empty(t, *h.Diagnostics, "valid modifiers %v should not produce diagnostics", tc.modifiers)
			assert.Equal(t, "action", d.Modifier, "internal modifier should be set to action")
		})
	}
}

func TestAnalyseEventDirective_UnknownModifierProducesError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		wantMessage string
		modifiers   []string
	}{
		{
			name:        "invalid modifier",
			modifiers:   []string{"invalid"},
			wantMessage: "Unknown event modifier .invalid. Supported modifiers: .prevent, .stop, .once, .self, .passive, .capture",
		},
		{
			name:        "throttle modifier",
			modifiers:   []string{"throttle"},
			wantMessage: "Unknown event modifier .throttle. Supported modifiers: .prevent, .stop, .once, .self, .passive, .capture",
		},
		{
			name:        "valid then unknown modifier",
			modifiers:   []string{"prevent", "throttle"},
			wantMessage: "Unknown event modifier .throttle. Supported modifiers: .prevent, .stop, .once, .self, .passive, .capture",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			vm := newTestHarnessVirtualModule()
			contextManager := newContextManager(h.Resolver, vm)
			aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "", nil)

			node := &ast_domain.TemplateNode{
				TagName:  "button",
				NodeType: ast_domain.NodeElement,
			}

			d := &ast_domain.Directive{
				Type:           ast_domain.DirectiveOn,
				Arg:            "click",
				EventModifiers: tc.modifiers,
				RawExpression:  "handleClick()",
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "handleClick"},
				},
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			}

			aa.analyseEventDirective(context.Background(), node, d, h.Context, nil)

			require.Len(t, *h.Diagnostics, 1)
			assert.Equal(t, ast_domain.Error, (*h.Diagnostics)[0].Severity)
			assert.Contains(t, (*h.Diagnostics)[0].Message, tc.wantMessage)
		})
	}
}

func TestAnalyseEventDirective_PassivePreventConflictProducesError(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	aa := newAttributeAnalyser(h.Resolver, nil, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:           ast_domain.DirectiveOn,
		Arg:            "wheel",
		EventModifiers: []string{"passive", "prevent"},
		RawExpression:  "handleWheel()",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "handleWheel"},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.analyseEventDirective(context.Background(), node, d, h.Context, nil)

	require.Len(t, *h.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*h.Diagnostics)[0].Severity)
	assert.Contains(t, (*h.Diagnostics)[0].Message, "Modifiers .passive and .prevent are incompatible")
}

func TestAnalyseEventDirective_NoModifiersProducesNoError(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	vm := newTestHarnessVirtualModule()
	contextManager := newContextManager(h.Resolver, vm)
	actions := map[string]ActionInfoProvider{
		"contact.Send": mockActionInfo{method: "POST"},
	}
	aa := newAttributeAnalyser(h.Resolver, actions, contextManager, "", nil)

	node := &ast_domain.TemplateNode{
		TagName:  "button",
		NodeType: ast_domain.NodeElement,
	}

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveOn,
		Arg:           "click",
		RawExpression: "action.contact.Send()",
		Expression: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "action.contact.Send"},
		},
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	aa.analyseEventDirective(context.Background(), node, d, h.Context, nil)

	assert.Empty(t, *h.Diagnostics, "no modifiers should not produce diagnostics")
	assert.Equal(t, "action", d.Modifier)
}

func TestNewSyntheticAnyTypeInfo_Fields(t *testing.T) {
	t.Parallel()

	result := newSyntheticAnyTypeInfo()

	require.NotNil(t, result)
	assert.False(t, result.IsSynthetic)
	assert.Empty(t, result.PackageAlias)
	assert.Empty(t, result.CanonicalPackagePath)
	assert.False(t, result.IsExportedPackageSymbol)
	assert.Empty(t, result.InitialPackagePath)
	assert.Empty(t, result.InitialFilePath)
}

func TestExtractHelperNameFromDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		raw      string
		expected string
	}{
		{
			name:     "simple helper call",
			raw:      "helpers.doSomething()",
			expected: "doSomething",
		},
		{
			name:     "helper with arguments",
			raw:      "helpers.doSomething($event, data)",
			expected: "doSomething",
		},
		{
			name:     "helper without parens",
			raw:      "helpers.myHelper",
			expected: "myHelper",
		},
		{
			name:     "nested parentheses",
			raw:      "helpers.calc(fn(1,2))",
			expected: "calc",
		},
		{
			name:     "just prefix",
			raw:      "helpers.",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			directive := &ast_domain.Directive{
				RawExpression: tc.raw,
				Type:          ast_domain.DirectiveEvent,
			}

			result := extractHelperNameFromDirective(directive)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClassifyGoTypeCategory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		goType   string
		expected string
	}{
		{"string", categoryString},
		{"bool", categoryBoolean},
		{"int", categoryNumber},
		{"int8", categoryNumber},
		{"int16", categoryNumber},
		{"int32", categoryNumber},
		{"int64", categoryNumber},
		{"uint", categoryNumber},
		{"uint8", categoryNumber},
		{"uint16", categoryNumber},
		{"uint32", categoryNumber},
		{"uint64", categoryNumber},
		{"float32", categoryNumber},
		{"float64", categoryNumber},
		{"any", categoryAny},
		{"interface{}", categoryAny},
		{"*string", categoryString},
		{"*int", categoryNumber},
		{"*bool", categoryBoolean},
		{"MyStruct", categoryObject},
		{"[]string", categoryObject},
		{"map[string]int", categoryObject},
	}

	for _, tc := range testCases {
		t.Run(tc.goType, func(t *testing.T) {
			t.Parallel()

			result := classifyGoTypeCategory(tc.goType)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClassifyResolvedExprCategory(t *testing.T) {
	t.Parallel()

	t.Run("nil annotation returns any", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "x"}

		result := classifyResolvedExprCategory(expression)

		assert.Equal(t, categoryAny, result)
	})

	t.Run("nil resolved type returns any", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "x"}
		expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}

		result := classifyResolvedExprCategory(expression)

		assert.Equal(t, categoryAny, result)
	})

	t.Run("nil type expr returns any", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "x"}
		expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{},
		}

		result := classifyResolvedExprCategory(expression)

		assert.Equal(t, categoryAny, result)
	})

	t.Run("string type returns string", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "x"}
		expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}

		result := classifyResolvedExprCategory(expression)

		assert.Equal(t, categoryString, result)
	})

	t.Run("int type returns number", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "x"}
		expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
		}

		result := classifyResolvedExprCategory(expression)

		assert.Equal(t, categoryNumber, result)
	})

	t.Run("bool type returns boolean", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "x"}
		expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("bool"),
			},
		}

		result := classifyResolvedExprCategory(expression)

		assert.Equal(t, categoryBoolean, result)
	})

	t.Run("struct type returns object", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "x"}
		expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyStruct"),
			},
		}

		result := classifyResolvedExprCategory(expression)

		assert.Equal(t, categoryObject, result)
	})
}

func TestValidateClientHandlerArgs(t *testing.T) {
	t.Parallel()

	t.Run("nil validator skips validation", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}

		aa.validateClientHandlerArgs(callExpr, "fn", nil, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("nil client exports skips validation", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("unknown function skips validation", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "unknown"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"other": {Name: "other"},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "unknown", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("function with no params skips validation", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {Name: "fn"},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("too many arguments emits warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "greet"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "a"},
				&ast_domain.Identifier{Name: "b"},
			},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"greet": {
						Name: "greet",
						Params: []ParamInfo{
							{Name: "name", Category: categoryString},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "greet", validator, d, ctx)

		require.Len(t, *ctx.Diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "expects 1 argument(s), but 2 provided")
	})

	t.Run("too few arguments emits warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {
						Name: "fn",
						Params: []ParamInfo{
							{Name: "a", Category: categoryString},
							{Name: "b", Category: categoryNumber},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		require.Len(t, *ctx.Diagnostics, 1)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "expects 2 argument(s), but 0 provided")
	})

	t.Run("type mismatch emits warning", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		argument := &ast_domain.Identifier{Name: "x"}
		argument.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{argument},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {
						Name: "fn",
						Params: []ParamInfo{
							{Name: "count", Category: categoryNumber},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		require.Len(t, *ctx.Diagnostics, 1)
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "Type mismatch for parameter 'count'")
		assert.Contains(t, (*ctx.Diagnostics)[0].Message, "got 'string', expected 'number'")
	})

	t.Run("any category skips type check", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		argument := &ast_domain.Identifier{Name: "x"}
		argument.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{argument},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {
						Name: "fn",
						Params: []ParamInfo{
							{Name: "x", Category: categoryAny},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("unresolved argument category skips type check", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		argument := &ast_domain.Identifier{Name: "x"}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{argument},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {
						Name: "fn",
						Params: []ParamInfo{
							{Name: "x", Category: categoryNumber},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("matching types produce no diagnostic", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		argument := &ast_domain.Identifier{Name: "x"}
		argument.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{argument},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {
						Name: "fn",
						Params: []ParamInfo{
							{Name: "x", Category: categoryString},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("optional params allow fewer arguments", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		argument := &ast_domain.Identifier{Name: "x"}
		argument.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{argument},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {
						Name: "fn",
						Params: []ParamInfo{
							{Name: "a", Category: categoryString},
							{Name: "b", Category: categoryNumber, Optional: true},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("rest params allow extra arguments", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "a"},
				&ast_domain.Identifier{Name: "b"},
				&ast_domain.Identifier{Name: "c"},
			},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {
						Name: "fn",
						Params: []ParamInfo{
							{Name: "first", Category: categoryString},
							{Name: "rest", Category: categoryObject, IsRest: true},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})

	t.Run("more arguments than params stops checking at param count", func(t *testing.T) {
		t.Parallel()

		ctx := createTestContext()
		vm := createMinimalVirtualModule()
		contextManager := newContextManager(nil, vm)
		aa := newAttributeAnalyser(nil, nil, contextManager, "", nil)

		arg1 := &ast_domain.Identifier{Name: "x"}
		arg1.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}
		arg2 := &ast_domain.Identifier{Name: "y"}
		arg2.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{arg1, arg2},
		}
		d := &ast_domain.Directive{Location: ast_domain.Location{Line: 1}}
		validator := &PKValidator{
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"fn": {
						Name: "fn",
						Params: []ParamInfo{
							{Name: "a", Category: categoryString},
							{Name: "rest", Category: categoryAny, IsRest: true},
						},
					},
				},
			},
		}

		aa.validateClientHandlerArgs(callExpr, "fn", validator, d, ctx)

		assert.Empty(t, *ctx.Diagnostics)
	})
}
