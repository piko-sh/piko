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
)

func TestNewComponentLinker(t *testing.T) {
	t.Run("CreatesLinkerWithDependencies", func(t *testing.T) {
		resolver := &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		}
		linker := NewComponentLinker(resolver)

		require.NotNil(t, linker, "Expected non-nil linker")
		if linker.typeResolver != resolver {
			t.Error("Expected resolver to be set")
		}

	})
}

func TestComponentLinker_Link(t *testing.T) {
	t.Run("EmptyAST", func(t *testing.T) {
		linker := NewComponentLinker(&TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		})
		expansionResult := &annotator_dto.ExpansionResult{
			FlattenedAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{},
			},
			CombinedCSS: "/* empty */",
		}
		vm := createTestVirtualModule()

		result, diagnostics, err := linker.Link(context.Background(), expansionResult, vm, "/test/main.piko")

		if err != nil {
			t.Errorf("Expected no error for empty AST, got: %v", err)
		}
		require.NotNil(t, result, "Expected non-nil result")
		if len(diagnostics) != 0 {
			t.Errorf("Expected no diagnostics for empty AST, got %d", len(diagnostics))
		}
		if len(result.UniqueInvocations) != 0 {
			t.Errorf("Expected no invocations for empty AST, got %d", len(result.UniqueInvocations))
		}
	})

	t.Run("NilAST", func(t *testing.T) {
		linker := NewComponentLinker(&TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		})
		expansionResult := &annotator_dto.ExpansionResult{
			FlattenedAST: nil,
			CombinedCSS:  "",
		}
		vm := createTestVirtualModule()

		result, diagnostics, err := linker.Link(context.Background(), expansionResult, vm, "/test/main.piko")

		if err != nil {
			t.Errorf("Expected no error for nil AST, got: %v", err)
		}
		require.NotNil(t, result, "Expected non-nil result")
		if len(diagnostics) != 0 {
			t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
		}
	})

	t.Run("EntryPointNotFound", func(t *testing.T) {
		linker := NewComponentLinker(&TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		})
		expansionResult := &annotator_dto.ExpansionResult{
			FlattenedAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{TagName: "div"},
				},
			},
		}
		vm := createTestVirtualModule()

		_, _, err := linker.Link(context.Background(), expansionResult, vm, "/nonexistent/path.piko")

		if err == nil {
			t.Error("Expected error for nonexistent entry point")
		}
	})
}

func TestGetValidPropNames(t *testing.T) {
	t.Run("EmptyMap", func(t *testing.T) {
		validProps := make(map[string]validPropInfo)

		names := getValidPropNames(validProps)

		if len(names) != 0 {
			t.Errorf("Expected empty slice, got %d names", len(names))
		}
	})

	t.Run("SingleProp", func(t *testing.T) {
		validProps := map[string]validPropInfo{
			"title": {GoFieldName: "Title"},
		}

		names := getValidPropNames(validProps)

		if len(names) != 1 {
			t.Fatalf("Expected 1 name, got %d", len(names))
		}
		if names[0] != "title" {
			t.Errorf("Expected 'title', got '%s'", names[0])
		}
	})

	t.Run("MultipleProps", func(t *testing.T) {
		validProps := map[string]validPropInfo{
			"title":   {GoFieldName: "Title"},
			"content": {GoFieldName: "Content"},
			"count":   {GoFieldName: "Count"},
		}

		names := getValidPropNames(validProps)

		if len(names) != 3 {
			t.Fatalf("Expected 3 names, got %d", len(names))
		}

		nameSet := make(map[string]bool)
		for _, name := range names {
			nameSet[name] = true
		}
		expectedNames := []string{"title", "content", "count"}
		for _, expected := range expectedNames {
			if !nameSet[expected] {
				t.Errorf("Expected name '%s' not found in result", expected)
			}
		}
	})
}

func TestCalculateCanonicalKey(t *testing.T) {
	t.Run("NoProps", func(t *testing.T) {
		partialAlias := "myPartial"
		props := make(map[string]ast_domain.PropValue)

		key := calculateCanonicalKey(partialAlias, props, "")

		if key == "" {
			t.Error("Expected non-empty key")
		}
	})

	t.Run("SingleProp", func(t *testing.T) {
		partialAlias := "myPartial"
		props := map[string]ast_domain.PropValue{
			"title": {
				Expression: &ast_domain.StringLiteral{Value: "Hello"},
			},
		}

		key := calculateCanonicalKey(partialAlias, props, "")

		if key == "" {
			t.Error("Expected non-empty key")
		}
	})

	t.Run("MultipleProps", func(t *testing.T) {
		partialAlias := "myPartial"
		props := map[string]ast_domain.PropValue{
			"title": {
				Expression: &ast_domain.StringLiteral{Value: "Hello"},
			},
			"count": {
				Expression: &ast_domain.IntegerLiteral{Value: 42},
			},
		}

		key := calculateCanonicalKey(partialAlias, props, "")

		if key == "" {
			t.Error("Expected non-empty key")
		}
	})

	t.Run("DeterministicOrdering", func(t *testing.T) {
		partialAlias := "myPartial"
		props1 := map[string]ast_domain.PropValue{
			"a": {Expression: &ast_domain.StringLiteral{Value: "1"}},
			"b": {Expression: &ast_domain.StringLiteral{Value: "2"}},
		}
		props2 := map[string]ast_domain.PropValue{
			"b": {Expression: &ast_domain.StringLiteral{Value: "2"}},
			"a": {Expression: &ast_domain.StringLiteral{Value: "1"}},
		}

		key1 := calculateCanonicalKey(partialAlias, props1, "")
		key2 := calculateCanonicalKey(partialAlias, props2, "")

		if key1 != key2 {
			t.Errorf("Expected identical keys for same props, got '%s' and '%s'", key1, key2)
		}
	})
}

func TestCoercePropType(t *testing.T) {
	tests := []struct {
		sourceExpression ast_domain.Expression
		destType         goast.Expr
		name             string
		expectType       string
		expectChange     bool
	}{
		{
			name:             "StringToInt",
			sourceExpression: &ast_domain.StringLiteral{Value: "42"},
			destType:         goast.NewIdent("int"),
			expectChange:     true,
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "StringToInt64",
			sourceExpression: &ast_domain.StringLiteral{Value: "123"},
			destType:         goast.NewIdent("int64"),
			expectChange:     true,
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "StringToUint",
			sourceExpression: &ast_domain.StringLiteral{Value: "42"},
			destType:         goast.NewIdent("uint"),
			expectChange:     true,
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "StringToFloat",
			sourceExpression: &ast_domain.StringLiteral{Value: "3.14"},
			destType:         goast.NewIdent("float64"),
			expectChange:     true,
			expectType:       "*ast_domain.FloatLiteral",
		},
		{
			name:             "StringToBoolTrue",
			sourceExpression: &ast_domain.StringLiteral{Value: "true"},
			destType:         goast.NewIdent("bool"),
			expectChange:     true,
			expectType:       "*ast_domain.BooleanLiteral",
		},
		{
			name:             "StringToBoolFalse",
			sourceExpression: &ast_domain.StringLiteral{Value: "false"},
			destType:         goast.NewIdent("bool"),
			expectChange:     true,
			expectType:       "*ast_domain.BooleanLiteral",
		},
		{
			name:             "InvalidStringToInt",
			sourceExpression: &ast_domain.StringLiteral{Value: "not a number"},
			destType:         goast.NewIdent("int"),
			expectChange:     false,
			expectType:       "*ast_domain.StringLiteral",
		},
		{
			name:             "NonStringSource",
			sourceExpression: &ast_domain.IntegerLiteral{Value: 42},
			destType:         goast.NewIdent("string"),
			expectChange:     false,
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "StringToString",
			sourceExpression: &ast_domain.StringLiteral{Value: "hello"},
			destType:         goast.NewIdent("string"),
			expectChange:     false,
			expectType:       "*ast_domain.StringLiteral",
		},
		{
			name:             "NonIdentDestType",
			sourceExpression: &ast_domain.StringLiteral{Value: "42"},
			destType:         &goast.ArrayType{Elt: goast.NewIdent("int")},
			expectChange:     false,
			expectType:       "*ast_domain.StringLiteral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coercePropType(tt.sourceExpression, tt.destType)

			resultType := getExpressionTypeName(result)
			if resultType != tt.expectType {
				t.Errorf("Expected type %s, got %s", tt.expectType, resultType)
			}

			if tt.expectChange && result == tt.sourceExpression {
				t.Error("Expected coercion to create new expression")
			}
			if !tt.expectChange && result != tt.sourceExpression {
				t.Error("Expected no coercion, but expression changed")
			}
		})
	}
}

func TestParseDefaultValue(t *testing.T) {
	tests := []struct {
		checkValue func(ast_domain.Expression) bool
		name       string
		input      string
		expectType string
	}{
		{
			name:       "BoolTrue",
			input:      "true",
			expectType: "*ast_domain.BooleanLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				b, ok := e.(*ast_domain.BooleanLiteral)
				return ok && b.Value == true
			},
		},
		{
			name:       "BoolTrueUppercase",
			input:      "TRUE",
			expectType: "*ast_domain.BooleanLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				b, ok := e.(*ast_domain.BooleanLiteral)
				return ok && b.Value == true
			},
		},
		{
			name:       "BoolFalse",
			input:      "false",
			expectType: "*ast_domain.BooleanLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				b, ok := e.(*ast_domain.BooleanLiteral)
				return ok && b.Value == false
			},
		},
		{
			name:       "Nil",
			input:      "nil",
			expectType: "*ast_domain.NilLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				_, ok := e.(*ast_domain.NilLiteral)
				return ok
			},
		},
		{
			name:       "Integer",
			input:      "42",
			expectType: "*ast_domain.IntegerLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				i, ok := e.(*ast_domain.IntegerLiteral)
				return ok && i.Value == 42
			},
		},
		{
			name:       "NegativeInteger",
			input:      "-123",
			expectType: "*ast_domain.IntegerLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				i, ok := e.(*ast_domain.IntegerLiteral)
				return ok && i.Value == -123
			},
		},
		{
			name:       "Float",
			input:      "3.14",
			expectType: "*ast_domain.FloatLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				f, ok := e.(*ast_domain.FloatLiteral)
				return ok && f.Value == 3.14
			},
		},
		{
			name:       "PlainString",
			input:      "hello",
			expectType: "*ast_domain.StringLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				s, ok := e.(*ast_domain.StringLiteral)
				return ok && s.Value == "hello"
			},
		},
		{
			name:       "EmptyString",
			input:      "",
			expectType: "*ast_domain.StringLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				s, ok := e.(*ast_domain.StringLiteral)
				return ok && s.Value == ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDefaultValue(context.Background(), tt.input, "test")

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			require.NotNil(t, result, "Expected non-nil result")

			resultType := getExpressionTypeName(result)
			if resultType != tt.expectType {
				t.Errorf("Expected type %s, got %s", tt.expectType, resultType)
			}

			if tt.checkValue != nil && !tt.checkValue(result) {
				t.Errorf("Value check failed for input '%s'", tt.input)
			}
		})
	}
}

func TestIsTypeCheckable(t *testing.T) {
	tests := []struct {
		ann      *ast_domain.GoGeneratorAnnotation
		name     string
		expected bool
	}{
		{
			name:     "NilAnnotation",
			ann:      nil,
			expected: false,
		},
		{
			name:     "NilResolvedType",
			ann:      &ast_domain.GoGeneratorAnnotation{EffectiveKeyExpression: nil, DynamicCollectionInfo: nil, StaticCollectionLiteral: nil, ParentTypeName: nil, BaseCodeGenVarName: nil, GeneratedSourcePath: nil, DynamicAttributeOrigins: nil, ResolvedType: nil, Symbol: nil, PartialInfo: nil, PropDataSource: nil, OriginalSourcePath: nil, OriginalPackageAlias: nil, FieldTag: nil, SourceInvocationKey: nil, StaticCollectionData: nil, Srcset: nil, Stringability: 0, IsStatic: false, NeedsCSRF: false, NeedsRuntimeSafetyCheck: false, IsStructurallyStatic: false, IsPointerToStringable: false, IsCollectionCall: false, IsHybridCollection: false, IsMapAccess: false},
			expected: false,
		},
		{
			name: "NilTypeExpr",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: nil, PackageAlias: "", CanonicalPackagePath: "", IsSynthetic: false, IsExportedPackageSymbol: false, InitialPackagePath: "", InitialFilePath: ""},
			},
			expected: false,
		},
		{
			name: "AnyType",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:          goast.NewIdent("any"),
					PackageAlias:            "",
					CanonicalPackagePath:    "",
					IsSynthetic:             false,
					IsExportedPackageSymbol: false,
					InitialPackagePath:      "",
					InitialFilePath:         "",
				},
			},
			expected: false,
		},
		{
			name: "StringType",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:          goast.NewIdent("string"),
					PackageAlias:            "",
					CanonicalPackagePath:    "",
					IsSynthetic:             false,
					IsExportedPackageSymbol: false,
					InitialPackagePath:      "",
					InitialFilePath:         "",
				},
			},
			expected: true,
		},
		{
			name: "IntType",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:          goast.NewIdent("int"),
					PackageAlias:            "",
					CanonicalPackagePath:    "",
					IsSynthetic:             false,
					IsExportedPackageSymbol: false,
					InitialPackagePath:      "",
					InitialFilePath:         "",
				},
			},
			expected: true,
		},
		{
			name: "ComplexType",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.ArrayType{
						Elt: goast.NewIdent("string"),
					},
					PackageAlias:            "",
					CanonicalPackagePath:    "",
					IsSynthetic:             false,
					IsExportedPackageSymbol: false,
					InitialPackagePath:      "",
					InitialFilePath:         "",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTypeCheckable(tt.ann)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func getExpressionTypeName(expression ast_domain.Expression) string {
	switch expression.(type) {
	case *ast_domain.StringLiteral:
		return "*ast_domain.StringLiteral"
	case *ast_domain.IntegerLiteral:
		return "*ast_domain.IntegerLiteral"
	case *ast_domain.FloatLiteral:
		return "*ast_domain.FloatLiteral"
	case *ast_domain.BooleanLiteral:
		return "*ast_domain.BooleanLiteral"
	case *ast_domain.NilLiteral:
		return "*ast_domain.NilLiteral"
	case *ast_domain.Identifier:
		return "*ast_domain.Identifier"
	default:
		return "unknown"
	}
}

func TestSortInvocationsByOrder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		uniqueInvocs    map[string]*annotator_dto.PartialInvocation
		invocationOrder []string
		expectedOrder   []string
	}{
		{
			name:            "empty state returns empty slice",
			uniqueInvocs:    map[string]*annotator_dto.PartialInvocation{},
			invocationOrder: []string{},
			expectedOrder:   []string{},
		},
		{
			name: "single invocation",
			uniqueInvocs: map[string]*annotator_dto.PartialInvocation{
				"key1": {InvocationKey: "key1"},
			},
			invocationOrder: []string{"key1"},
			expectedOrder:   []string{"key1"},
		},
		{
			name: "preserves insertion order",
			uniqueInvocs: map[string]*annotator_dto.PartialInvocation{
				"first":  {InvocationKey: "first"},
				"second": {InvocationKey: "second"},
				"third":  {InvocationKey: "third"},
			},
			invocationOrder: []string{"first", "second", "third"},
			expectedOrder:   []string{"first", "second", "third"},
		},
		{
			name: "order differs from map iteration",
			uniqueInvocs: map[string]*annotator_dto.PartialInvocation{
				"z_last":   {InvocationKey: "z_last"},
				"a_first":  {InvocationKey: "a_first"},
				"m_middle": {InvocationKey: "m_middle"},
			},
			invocationOrder: []string{"m_middle", "z_last", "a_first"},
			expectedOrder:   []string{"m_middle", "z_last", "a_first"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := &linkingSharedState{
				uniqueInvocations: tc.uniqueInvocs,
				invocationOrder:   tc.invocationOrder,
			}

			result := sortInvocationsByOrder(state)

			if len(result) != len(tc.expectedOrder) {
				t.Fatalf("Expected %d invocations, got %d", len(tc.expectedOrder), len(result))
			}

			for i, expected := range tc.expectedOrder {
				if result[i].InvocationKey != expected {
					t.Errorf("Position %d: expected key '%s', got '%s'", i, expected, result[i].InvocationKey)
				}
			}
		})
	}
}

func TestCreateEmptyLinkingResult(t *testing.T) {
	t.Parallel()

	t.Run("returns result with empty invocations", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
		}
		css := "body { color: red; }"
		vm := createTestVirtualModule()
		diagnostics := []*ast_domain.Diagnostic{
			{Message: "test warning", Severity: ast_domain.Warning},
		}

		result, returnedDiags, err := createEmptyLinkingResult(ast, css, vm, diagnostics)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		require.NotNil(t, result, "Expected non-nil result")
		if result.LinkedAST != ast {
			t.Error("Expected LinkedAST to be the same as input")
		}
		if result.CombinedCSS != css {
			t.Errorf("Expected CombinedCSS '%s', got '%s'", css, result.CombinedCSS)
		}
		if len(result.UniqueInvocations) != 0 {
			t.Errorf("Expected empty UniqueInvocations, got %d", len(result.UniqueInvocations))
		}
		if result.VirtualModule != vm {
			t.Error("Expected VirtualModule to be the same as input")
		}
		if len(returnedDiags) != len(diagnostics) {
			t.Errorf("Expected %d diagnostics, got %d", len(diagnostics), len(returnedDiags))
		}
	})

	t.Run("handles nil inputs", func(t *testing.T) {
		t.Parallel()

		result, diagnostics, err := createEmptyLinkingResult(nil, "", nil, nil)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		require.NotNil(t, result, "Expected non-nil result")
		if result.LinkedAST != nil {
			t.Error("Expected nil LinkedAST")
		}
		if result.CombinedCSS != "" {
			t.Error("Expected empty CombinedCSS")
		}
		if result.VirtualModule != nil {
			t.Error("Expected nil VirtualModule")
		}
		if len(diagnostics) != 0 {
			t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
		}
	})
}

func TestCalculateCanonicalKey_WithInvokerInvocationKey(t *testing.T) {
	t.Parallel()

	t.Run("includes invoker invocation key in canonical key", func(t *testing.T) {
		t.Parallel()

		partialAlias := "card"
		props := map[string]ast_domain.PropValue{
			"title": {Expression: &ast_domain.StringLiteral{Value: "Hello"}},
		}

		keyWithoutInvoker := calculateCanonicalKey(partialAlias, props, "")
		keyWithInvoker := calculateCanonicalKey(partialAlias, props, "parent_inv_key")

		assert.NotEqual(t, keyWithoutInvoker, keyWithInvoker,
			"keys should differ when invoker invocation key is present")
	})

	t.Run("different invoker keys produce different canonical keys", func(t *testing.T) {
		t.Parallel()

		partialAlias := "card"
		props := map[string]ast_domain.PropValue{
			"title": {Expression: &ast_domain.StringLiteral{Value: "Hello"}},
		}

		key1 := calculateCanonicalKey(partialAlias, props, "invoker_A")
		key2 := calculateCanonicalKey(partialAlias, props, "invoker_B")

		assert.NotEqual(t, key1, key2,
			"different invoker keys should produce different canonical keys")
	})
}

func TestSetupRootContextForLinking(t *testing.T) {
	t.Parallel()

	t.Run("returns error when entry point not in graph", func(t *testing.T) {
		t.Parallel()

		linker := NewComponentLinker(&TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		})
		vm := createTestVirtualModule()
		_, _, err := linker.setupRootContextForLinking(context.Background(), vm, "/nonexistent/path.piko", new([]*ast_domain.Diagnostic))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "/nonexistent/path.piko")
	})

	t.Run("returns error when hashed name not found in components", func(t *testing.T) {
		t.Parallel()

		linker := NewComponentLinker(&TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		})
		vm := &annotator_dto.VirtualModule{
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{
					"/test/orphan.piko": "orphan_hash",
				},
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		}
		_, _, err := linker.setupRootContextForLinking(context.Background(), vm, "/test/orphan.piko", new([]*ast_domain.Diagnostic))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "orphan_hash")
	})

	t.Run("creates root context successfully for valid entry point", func(t *testing.T) {
		t.Parallel()

		linker := NewComponentLinker(&TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		})
		vm := createTestVirtualModule()
		rootCtx, mainVC, err := linker.setupRootContextForLinking(context.Background(), vm, "/test/main.piko", new([]*ast_domain.Diagnostic))

		require.NoError(t, err)
		assert.NotNil(t, rootCtx)
		assert.NotNil(t, mainVC)
		assert.Equal(t, "test/main", rootCtx.CurrentGoFullPackagePath)
		assert.Equal(t, "main", rootCtx.CurrentGoPackageName)
		assert.Equal(t, "/virtual/main.go", rootCtx.CurrentGoSourcePath)
		assert.Equal(t, "/test/main.piko", rootCtx.SFCSourcePath)
	})
}

func TestCoercePropType_AdditionalCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		sourceExpression ast_domain.Expression
		destType         goast.Expr
		expectType       string
	}{
		{
			name:             "string to int8",
			sourceExpression: &ast_domain.StringLiteral{Value: "127"},
			destType:         goast.NewIdent("int8"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to int16",
			sourceExpression: &ast_domain.StringLiteral{Value: "1000"},
			destType:         goast.NewIdent("int16"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to int32",
			sourceExpression: &ast_domain.StringLiteral{Value: "100000"},
			destType:         goast.NewIdent("int32"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to rune",
			sourceExpression: &ast_domain.StringLiteral{Value: "65"},
			destType:         goast.NewIdent("rune"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to uint8",
			sourceExpression: &ast_domain.StringLiteral{Value: "255"},
			destType:         goast.NewIdent("uint8"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to uint16",
			sourceExpression: &ast_domain.StringLiteral{Value: "65535"},
			destType:         goast.NewIdent("uint16"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to uint32",
			sourceExpression: &ast_domain.StringLiteral{Value: "4294967295"},
			destType:         goast.NewIdent("uint32"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to uint64",
			sourceExpression: &ast_domain.StringLiteral{Value: "100"},
			destType:         goast.NewIdent("uint64"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to byte",
			sourceExpression: &ast_domain.StringLiteral{Value: "200"},
			destType:         goast.NewIdent("byte"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to uintptr",
			sourceExpression: &ast_domain.StringLiteral{Value: "42"},
			destType:         goast.NewIdent("uintptr"),
			expectType:       "*ast_domain.IntegerLiteral",
		},
		{
			name:             "string to float32",
			sourceExpression: &ast_domain.StringLiteral{Value: "1.5"},
			destType:         goast.NewIdent("float32"),
			expectType:       "*ast_domain.FloatLiteral",
		},
		{
			name:             "invalid string to uint stays string",
			sourceExpression: &ast_domain.StringLiteral{Value: "not_a_number"},
			destType:         goast.NewIdent("uint"),
			expectType:       "*ast_domain.StringLiteral",
		},
		{
			name:             "invalid string to float stays string",
			sourceExpression: &ast_domain.StringLiteral{Value: "not_a_float"},
			destType:         goast.NewIdent("float64"),
			expectType:       "*ast_domain.StringLiteral",
		},
		{
			name:             "invalid string to bool stays string",
			sourceExpression: &ast_domain.StringLiteral{Value: "maybe"},
			destType:         goast.NewIdent("bool"),
			expectType:       "*ast_domain.StringLiteral",
		},
		{
			name:             "string to unknown type stays string",
			sourceExpression: &ast_domain.StringLiteral{Value: "42"},
			destType:         goast.NewIdent("complex128"),
			expectType:       "*ast_domain.StringLiteral",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := coercePropType(tc.sourceExpression, tc.destType)

			resultType := getExpressionTypeName(result)
			assert.Equal(t, tc.expectType, resultType)
		})
	}
}

func TestParseDefaultValue_AdditionalCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		checkValue func(ast_domain.Expression) bool
		name       string
		input      string
		expectType string
	}{
		{
			name:       "zero integer",
			input:      "0",
			expectType: "*ast_domain.IntegerLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				i, ok := e.(*ast_domain.IntegerLiteral)
				return ok && i.Value == 0
			},
		},
		{
			name:       "negative float",
			input:      "-2.5",
			expectType: "*ast_domain.FloatLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				f, ok := e.(*ast_domain.FloatLiteral)
				return ok && f.Value == -2.5
			},
		},
		{
			name:       "nil literal",
			input:      "nil",
			expectType: "*ast_domain.NilLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				_, ok := e.(*ast_domain.NilLiteral)
				return ok
			},
		},
		{
			name:       "mixed case TRUE",
			input:      "True",
			expectType: "*ast_domain.BooleanLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				b, ok := e.(*ast_domain.BooleanLiteral)
				return ok && b.Value == true
			},
		},
		{
			name:       "mixed case FALSE",
			input:      "False",
			expectType: "*ast_domain.BooleanLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				b, ok := e.(*ast_domain.BooleanLiteral)
				return ok && b.Value == false
			},
		},
		{
			name:       "string with spaces",
			input:      "hello world",
			expectType: "*ast_domain.StringLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				s, ok := e.(*ast_domain.StringLiteral)
				return ok && s.Value == "hello world"
			},
		},
		{
			name:       "string with special characters",
			input:      "a+b=c",
			expectType: "*ast_domain.StringLiteral",
			checkValue: func(e ast_domain.Expression) bool {
				s, ok := e.(*ast_domain.StringLiteral)
				return ok && s.Value == "a+b=c"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseDefaultValue(context.Background(), tc.input, "test")

			require.NoError(t, err)
			require.NotNil(t, result)

			resultType := getExpressionTypeName(result)
			assert.Equal(t, tc.expectType, resultType)

			if tc.checkValue != nil {
				assert.True(t, tc.checkValue(result), "value check failed for input %q", tc.input)
			}
		})
	}
}

func TestLink_WithSimpleAST(t *testing.T) {
	t.Parallel()

	t.Run("successful link with simple elements", func(t *testing.T) {
		t.Parallel()

		linker := NewComponentLinker(&TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		})
		expansionResult := &annotator_dto.ExpansionResult{
			FlattenedAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
					},
				},
			},
			CombinedCSS: "body { color: red; }",
		}
		vm := createTestVirtualModule()

		result, diagnostics, err := linker.Link(context.Background(), expansionResult, vm, "/test/main.piko")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "body { color: red; }", result.CombinedCSS)
		assert.Equal(t, vm, result.VirtualModule)
		assert.Empty(t, diagnostics)
		assert.Empty(t, result.UniqueInvocations)
	})
}

func createTestVirtualModule() *annotator_dto.VirtualModule {
	return &annotator_dto.VirtualModule{
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{
				"/test/main.piko": "main_abc123",
			},
			HashedNameToPath: map[string]string{
				"main_abc123": "/test/main.piko",
			},
		},
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"main_abc123": {
				HashedName:             "main_abc123",
				CanonicalGoPackagePath: "test/main",
				VirtualGoFilePath:      "/virtual/main.go",
				RewrittenScriptAST: &goast.File{
					Name: goast.NewIdent("main"),
				},
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/test/main.piko",
				},
			},
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			"test/main": {
				HashedName: "main_abc123",
			},
		},
	}
}
