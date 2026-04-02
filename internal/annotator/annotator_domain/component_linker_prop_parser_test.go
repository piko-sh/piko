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
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestGetValidPropsForComponent(t *testing.T) {
	t.Run("NilVirtualComponent", func(t *testing.T) {
		inspector := &inspector_domain.MockTypeQuerier{}
		ctx := createPropTestContext()

		props, err := getValidPropsForComponent(nil, inspector, ctx)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(props) != 0 {
			t.Errorf("Expected empty props map for nil component, got %d", len(props))
		}
	})

	t.Run("ComponentWithoutScript", func(t *testing.T) {
		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{},
		}
		inspector := &inspector_domain.MockTypeQuerier{}
		ctx := createPropTestContext()

		props, err := getValidPropsForComponent(vc, inspector, ctx)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(props) != 0 {
			t.Errorf("Expected empty props map, got %d", len(props))
		}
	})

	t.Run("ComponentWithoutPropsType", func(t *testing.T) {
		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					PropsTypeExpression: nil,
				},
			},
		}
		inspector := &inspector_domain.MockTypeQuerier{}
		ctx := createPropTestContext()

		props, err := getValidPropsForComponent(vc, inspector, ctx)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(props) != 0 {
			t.Errorf("Expected empty props map, got %d", len(props))
		}
	})
}

func TestPropParser_ParseFieldAsProp(t *testing.T) {
	tests := []struct {
		field         *inspector_dto.Field
		checkPropInfo func(*testing.T, validPropInfo)
		name          string
		expectedName  string
		expectError   bool
	}{
		{
			name: "SimpleField",
			field: &inspector_dto.Field{
				Name:       "Title",
				TypeString: "string",
				RawTag:     "",
			},
			expectedName: "title",
			expectError:  false,
			checkPropInfo: func(t *testing.T, info validPropInfo) {
				if info.GoFieldName != "Title" {
					t.Errorf("Expected GoFieldName 'Title', got '%s'", info.GoFieldName)
				}
				if info.IsRequired {
					t.Error("Expected IsRequired to be false by default")
				}
				if info.ShouldCoerce {
					t.Error("Expected ShouldCoerce to be false by default")
				}
			},
		},
		{
			name: "FieldWithPropTag",
			field: &inspector_dto.Field{
				Name:       "UserName",
				TypeString: "string",
				RawTag:     `prop:"username"`,
			},
			expectedName: "username",
			expectError:  false,
			checkPropInfo: func(t *testing.T, info validPropInfo) {
				if info.GoFieldName != "UserName" {
					t.Errorf("Expected GoFieldName 'UserName', got '%s'", info.GoFieldName)
				}
			},
		},
		{
			name: "RequiredField",
			field: &inspector_dto.Field{
				Name:       "Email",
				TypeString: "string",
				RawTag:     `validate:"required"`,
			},
			expectedName: "email",
			expectError:  false,
			checkPropInfo: func(t *testing.T, info validPropInfo) {
				if !info.IsRequired {
					t.Error("Expected IsRequired to be true")
				}
			},
		},
		{
			name: "FieldWithDefault",
			field: &inspector_dto.Field{
				Name:       "Count",
				TypeString: "int",
				RawTag:     `default:"42"`,
			},
			expectedName: "count",
			expectError:  false,
			checkPropInfo: func(t *testing.T, info validPropInfo) {
				if info.DefaultValue == nil {
					t.Error("Expected DefaultValue to be set")
				} else if *info.DefaultValue != "42" {
					t.Errorf("Expected DefaultValue '42', got '%s'", *info.DefaultValue)
				}
			},
		},
		{
			name: "FieldWithFactory",
			field: &inspector_dto.Field{
				Name:       "Items",
				TypeString: "[]string",
				RawTag:     `factory:"NewItems"`,
			},
			expectedName: "items",
			expectError:  false,
			checkPropInfo: func(t *testing.T, info validPropInfo) {
				if info.FactoryFuncName != "NewItems" {
					t.Errorf("Expected FactoryFuncName 'NewItems', got '%s'", info.FactoryFuncName)
				}
			},
		},
		{
			name: "FieldWithCoerceTrue",
			field: &inspector_dto.Field{
				Name:       "Port",
				TypeString: "int",
				RawTag:     `coerce:"true"`,
			},
			expectedName: "port",
			expectError:  false,
			checkPropInfo: func(t *testing.T, info validPropInfo) {
				if !info.ShouldCoerce {
					t.Error("Expected ShouldCoerce to be true")
				}
			},
		},
		{
			name: "FieldWithCoerceEmpty",
			field: &inspector_dto.Field{
				Name:       "Timeout",
				TypeString: "int",
				RawTag:     `coerce:""`,
			},
			expectedName: "timeout",
			expectError:  false,
			checkPropInfo: func(t *testing.T, info validPropInfo) {
				if !info.ShouldCoerce {
					t.Error("Expected ShouldCoerce to be true for empty coerce tag")
				}
			},
		},
		{
			name: "FieldWithBothDefaultAndFactory",
			field: &inspector_dto.Field{
				Name:       "Data",
				TypeString: "[]byte",
				RawTag:     `default:"test" factory:"NewData"`,
			},
			expectedName: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := createTestPropParser()

			propName, propInfo, err := parser.parseFieldAsProp(tt.field)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if propName != tt.expectedName {
				t.Errorf("Expected prop name '%s', got '%s'", tt.expectedName, propName)
			}

			if tt.checkPropInfo != nil {
				tt.checkPropInfo(t, propInfo)
			}
		})
	}
}

func createPropTestContext() *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoPackageName:     "pkg",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            "/test/file.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}
}

func createTestPropParser() *propParser {
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoPackageName:     "pkg",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            "/test/file.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	return &propParser{
		inspector: &inspector_domain.MockTypeQuerier{},
		vc: &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/component.piko",
			},
		},
		validProps: make(map[string]validPropInfo),
		ctx:        ctx,
	}
}

func TestIsQueryCompatibleType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeExpr string
		expected bool
	}{
		{
			name:     "string type is compatible",
			typeExpr: "string",
			expected: true,
		},
		{
			name:     "pointer to string is compatible",
			typeExpr: "*string",
			expected: true,
		},
		{
			name:     "int type is not compatible",
			typeExpr: "int",
			expected: false,
		},
		{
			name:     "bool type is not compatible",
			typeExpr: "bool",
			expected: false,
		},
		{
			name:     "float64 type is not compatible",
			typeExpr: "float64",
			expected: false,
		},
		{
			name:     "pointer to int is not compatible",
			typeExpr: "*int",
			expected: false,
		},
		{
			name:     "slice of string is not compatible",
			typeExpr: "[]string",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			typeExpr := parseTypeExpr(tc.typeExpr)
			result := isQueryCompatibleType(typeExpr)

			if result != tc.expected {
				t.Errorf("Expected %v for type '%s', got %v", tc.expected, tc.typeExpr, result)
			}
		})
	}
}

func TestIsSliceOrMapType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeExpr string
		expected bool
	}{
		{
			name:     "slice type returns true",
			typeExpr: "[]string",
			expected: true,
		},
		{
			name:     "map type returns true",
			typeExpr: "map[string]int",
			expected: true,
		},
		{
			name:     "pointer to slice returns true",
			typeExpr: "*[]string",
			expected: true,
		},
		{
			name:     "pointer to map returns true",
			typeExpr: "*map[string]int",
			expected: true,
		},
		{
			name:     "string type returns false",
			typeExpr: "string",
			expected: false,
		},
		{
			name:     "int type returns false",
			typeExpr: "int",
			expected: false,
		},
		{
			name:     "pointer to string returns false",
			typeExpr: "*string",
			expected: false,
		},
		{
			name:     "struct type returns false",
			typeExpr: "MyStruct",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			typeExpr := parseTypeExpr(tc.typeExpr)
			result := isSliceOrMapType(typeExpr)

			if result != tc.expected {
				t.Errorf("Expected %v for type '%s', got %v", tc.expected, tc.typeExpr, result)
			}
		})
	}
}

func TestPropParser_CollectProps_WithFields(t *testing.T) {
	t.Parallel()

	t.Run("collects fields from resolved type", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		parser.inspector = &inspector_domain.MockTypeQuerier{
			ResolveExprToNamedTypeFunc: func(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name: "Props",
					Fields: []*inspector_dto.Field{
						{
							Name:       "Title",
							TypeString: "string",
							RawTag:     "",
							IsEmbedded: false,
						},
						{
							Name:       "Count",
							TypeString: "int",
							RawTag:     `default:"10"`,
							IsEmbedded: false,
						},
					},
				}, "pkg"
			},
		}

		typeExpr := goast.NewIdent("Props")
		err := parser.collectProps(typeExpr, "test/pkg", "/test/file.go")

		require.NoError(t, err)
		assert.Len(t, parser.validProps, 2)
		assert.Contains(t, parser.validProps, "title")
		assert.Contains(t, parser.validProps, "count")
		assert.Equal(t, "Title", parser.validProps["title"].GoFieldName)
		assert.Equal(t, "Count", parser.validProps["count"].GoFieldName)
	})

	t.Run("returns nil when type cannot be resolved", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		parser.inspector = &inspector_domain.MockTypeQuerier{
			ResolveExprToNamedTypeFunc: func(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return nil, ""
			},
		}

		typeExpr := goast.NewIdent("Unknown")
		err := parser.collectProps(typeExpr, "test/pkg", "/test/file.go")

		require.NoError(t, err)
		assert.Empty(t, parser.validProps)
	})

	t.Run("detects duplicate prop names", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		parser.inspector = &inspector_domain.MockTypeQuerier{
			ResolveExprToNamedTypeFunc: func(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name: "Props",
					Fields: []*inspector_dto.Field{
						{
							Name:       "Title",
							TypeString: "string",
							RawTag:     "",
							IsEmbedded: false,
						},
						{
							Name:       "AnotherTitle",
							TypeString: "string",
							RawTag:     `prop:"title"`,
							IsEmbedded: false,
						},
					},
				}, "pkg"
			},
		}

		typeExpr := goast.NewIdent("Props")
		err := parser.collectProps(typeExpr, "test/pkg", "/test/file.go")

		require.NoError(t, err)
		assert.Len(t, parser.validProps, 1, "duplicate prop should not be added twice")
		require.NotEmpty(t, *parser.ctx.Diagnostics, "should emit a diagnostic for duplicate prop")
		assert.Contains(t, (*parser.ctx.Diagnostics)[0].Message, "Duplicate prop name")
	})

	t.Run("handles embedded field", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		parser := createTestPropParser()
		parser.inspector = &inspector_domain.MockTypeQuerier{
			ResolveExprToNamedTypeFunc: func(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				callCount++
				if callCount == 1 {
					return &inspector_dto.Type{
						Name: "Props",
						Fields: []*inspector_dto.Field{
							{
								Name:       "CommonProps",
								TypeString: "CommonProps",
								RawTag:     "",
								IsEmbedded: true,
							},
						},
					}, "pkg"
				}
				return &inspector_dto.Type{
					Name: "CommonProps",
					Fields: []*inspector_dto.Field{
						{
							Name:       "ID",
							TypeString: "string",
							RawTag:     "",
							IsEmbedded: false,
						},
					},
				}, "pkg"
			},
		}

		typeExpr := goast.NewIdent("Props")
		err := parser.collectProps(typeExpr, "test/pkg", "/test/file.go")

		require.NoError(t, err)
		assert.Contains(t, parser.validProps, "id", "embedded field should be collected")
	})

	t.Run("emits diagnostic for invalid field tag", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		parser.inspector = &inspector_domain.MockTypeQuerier{
			ResolveExprToNamedTypeFunc: func(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name: "Props",
					Fields: []*inspector_dto.Field{
						{
							Name:       "Data",
							TypeString: "[]byte",
							RawTag:     `default:"test" factory:"NewData"`,
							IsEmbedded: false,
						},
					},
				}, "pkg"
			},
		}

		typeExpr := goast.NewIdent("Props")
		err := parser.collectProps(typeExpr, "test/pkg", "/test/file.go")

		require.NoError(t, err)
		assert.Empty(t, parser.validProps, "field with invalid tags should not be added")
		require.NotEmpty(t, *parser.ctx.Diagnostics, "should emit diagnostic for ambiguous tag")
	})
}

func TestPropParser_ParseQueryTag(t *testing.T) {
	t.Parallel()

	t.Run("stores query parameter name", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Search",
			TypeString: "string",
		}
		tag := map[string]string{
			"query": "q",
		}
		destTypeExpr := goast.NewIdent("string")
		result := &propParseResult{}

		parser.parseQueryTag(field, tag, destTypeExpr, result)

		assert.Equal(t, "q", result.queryParamName)
		assert.Empty(t, *parser.ctx.Diagnostics)
	})

	t.Run("emits error for empty query tag", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Search",
			TypeString: "string",
		}
		tag := map[string]string{
			"query": "",
		}
		destTypeExpr := goast.NewIdent("string")
		result := &propParseResult{}

		parser.parseQueryTag(field, tag, destTypeExpr, result)

		assert.Equal(t, "", result.queryParamName)
		require.NotEmpty(t, *parser.ctx.Diagnostics)
		assert.Contains(t, (*parser.ctx.Diagnostics)[0].Message, "empty query tag")
	})

	t.Run("does nothing when query tag is absent", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Search",
			TypeString: "string",
		}
		tag := map[string]string{}
		destTypeExpr := goast.NewIdent("string")
		result := &propParseResult{}

		parser.parseQueryTag(field, tag, destTypeExpr, result)

		assert.Equal(t, "", result.queryParamName)
		assert.Empty(t, *parser.ctx.Diagnostics)
	})
}

func TestPropParser_ValidateQueryType(t *testing.T) {
	t.Parallel()

	t.Run("warns when non-string type without coerce", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Port",
			TypeString: "int",
		}
		destTypeExpr := goast.NewIdent("int")
		result := &propParseResult{shouldCoerce: false}

		parser.validateQueryType(field, "port", destTypeExpr, result)

		require.NotEmpty(t, *parser.ctx.Diagnostics)
		assert.Contains(t, (*parser.ctx.Diagnostics)[0].Message, "non-string type")
	})

	t.Run("no warning when non-string type with coerce", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Port",
			TypeString: "int",
		}
		destTypeExpr := goast.NewIdent("int")
		result := &propParseResult{shouldCoerce: true}

		parser.validateQueryType(field, "port", destTypeExpr, result)

		assert.Empty(t, *parser.ctx.Diagnostics)
	})

	t.Run("errors when type is slice", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Tags",
			TypeString: "[]string",
		}
		destTypeExpr := goastutil.TypeStringToAST("[]string")
		result := &propParseResult{shouldCoerce: false, queryParamName: "tags"}

		parser.validateQueryType(field, "tags", destTypeExpr, result)

		require.NotEmpty(t, *parser.ctx.Diagnostics)
		assert.Contains(t, (*parser.ctx.Diagnostics)[len(*parser.ctx.Diagnostics)-1].Message, "not supported for slice or map")
		assert.Equal(t, "", result.queryParamName, "queryParamName should be cleared for unsupported types")
	})

	t.Run("errors when type is map", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Metadata",
			TypeString: "map[string]string",
		}
		destTypeExpr := goastutil.TypeStringToAST("map[string]string")
		result := &propParseResult{shouldCoerce: false, queryParamName: "metadata"}

		parser.validateQueryType(field, "metadata", destTypeExpr, result)

		require.NotEmpty(t, *parser.ctx.Diagnostics)
		assert.Contains(t, (*parser.ctx.Diagnostics)[len(*parser.ctx.Diagnostics)-1].Message, "not supported for slice or map")
		assert.Equal(t, "", result.queryParamName)
	})

	t.Run("no warning for string type", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Name",
			TypeString: "string",
		}
		destTypeExpr := goast.NewIdent("string")
		result := &propParseResult{shouldCoerce: false}

		parser.validateQueryType(field, "name", destTypeExpr, result)

		assert.Empty(t, *parser.ctx.Diagnostics)
	})

	t.Run("no warning for pointer to string type", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "OptName",
			TypeString: "*string",
		}
		destTypeExpr := goastutil.TypeStringToAST("*string")
		result := &propParseResult{shouldCoerce: false}

		parser.validateQueryType(field, "name", destTypeExpr, result)

		assert.Empty(t, *parser.ctx.Diagnostics)
	})
}

func TestGetValidPropsForComponent_FullPath(t *testing.T) {
	t.Parallel()

	t.Run("collects props from component with script and props type", func(t *testing.T) {
		t.Parallel()

		propsTypeExpr := goast.NewIdent("Props")
		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "test/pkg",
			VirtualGoFilePath:      "/virtual/test.go",
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/component.piko",
				Script: &annotator_dto.ParsedScript{
					PropsTypeExpression: propsTypeExpr,
				},
			},
		}

		mockInspector := &inspector_domain.MockTypeQuerier{
			ResolveExprToNamedTypeFunc: func(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name: "Props",
					Fields: []*inspector_dto.Field{
						{
							Name:       "Title",
							TypeString: "string",
							RawTag:     `validate:"required"`,
							IsEmbedded: false,
						},
						{
							Name:       "Count",
							TypeString: "int",
							RawTag:     `default:"5" coerce:""`,
							IsEmbedded: false,
						},
					},
				}, "pkg"
			},
		}

		ctx := createPropTestContext()

		props, err := getValidPropsForComponent(vc, mockInspector, ctx)

		require.NoError(t, err)
		assert.Len(t, props, 2)

		titleProp, ok := props["title"]
		require.True(t, ok)
		assert.Equal(t, "Title", titleProp.GoFieldName)
		assert.True(t, titleProp.IsRequired)

		countProp, ok := props["count"]
		require.True(t, ok)
		assert.Equal(t, "Count", countProp.GoFieldName)
		assert.True(t, countProp.ShouldCoerce)
		require.NotNil(t, countProp.DefaultValue)
		assert.Equal(t, "5", *countProp.DefaultValue)
	})
}

func TestPropParser_ParseFieldAsProp_WithQueryTag(t *testing.T) {
	t.Parallel()

	t.Run("field with query tag stores query param name", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "SearchQuery",
			TypeString: "string",
			RawTag:     `query:"q"`,
		}

		propName, propInfo, err := parser.parseFieldAsProp(field)

		require.NoError(t, err)
		assert.Equal(t, "searchquery", propName)
		assert.Equal(t, "q", propInfo.QueryParamName)
	})

	t.Run("field with query and coerce tags", func(t *testing.T) {
		t.Parallel()

		parser := createTestPropParser()
		field := &inspector_dto.Field{
			Name:       "Page",
			TypeString: "int",
			RawTag:     `query:"page" coerce:""`,
		}

		propName, propInfo, err := parser.parseFieldAsProp(field)

		require.NoError(t, err)
		assert.Equal(t, "page", propName)
		assert.Equal(t, "page", propInfo.QueryParamName)
		assert.True(t, propInfo.ShouldCoerce)
	})
}

func TestPropParser_ParsePropName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		fieldName    string
		rawTag       string
		expectedName string
	}{
		{
			name:         "no tag uses lowercase field name",
			fieldName:    "MyField",
			rawTag:       "",
			expectedName: "myfield",
		},
		{
			name:         "prop tag overrides name",
			fieldName:    "MyField",
			rawTag:       `prop:"custom_name"`,
			expectedName: "custom_name",
		},
		{
			name:         "prop tag with comma uses first part",
			fieldName:    "MyField",
			rawTag:       `prop:"alias,omitempty"`,
			expectedName: "alias",
		},
		{
			name:         "prop tag with empty value uses field name",
			fieldName:    "MyField",
			rawTag:       `prop:""`,
			expectedName: "myfield",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser := createTestPropParser()
			field := &inspector_dto.Field{
				Name:   tc.fieldName,
				RawTag: tc.rawTag,
			}
			tag := inspector_dto.ParseStructTag(field.RawTag)

			result := parser.parsePropName(field, tag)

			assert.Equal(t, tc.expectedName, result)
		})
	}
}

func TestIsQueryCompatibleType_StarExprNonIdent(t *testing.T) {
	t.Parallel()

	t.Run("pointer to non-ident is not compatible", func(t *testing.T) {
		t.Parallel()

		typeExpr := &goast.StarExpr{
			X: &goast.ArrayType{
				Elt: goast.NewIdent("string"),
			},
		}

		result := isQueryCompatibleType(typeExpr)

		assert.False(t, result)
	})
}

func parseTypeExpr(typeString string) goast.Expr {
	return goastutil.TypeStringToAST(typeString)
}
