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
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestIsUnexportedName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "lowercase letter is unexported",
			input:    "myFunc",
			expected: true,
		},
		{
			name:     "uppercase letter is exported",
			input:    "MyFunc",
			expected: false,
		},
		{
			name:     "empty string is not unexported",
			input:    "",
			expected: false,
		},
		{
			name:     "underscore prefix is not lowercase",
			input:    "_hidden",
			expected: false,
		},
		{
			name:     "number prefix is not lowercase",
			input:    "123func",
			expected: false,
		},
		{
			name:     "single lowercase letter",
			input:    "a",
			expected: true,
		},
		{
			name:     "single uppercase letter",
			input:    "A",
			expected: false,
		},
		{
			name:     "lowercase z boundary",
			input:    "zVariable",
			expected: true,
		},
		{
			name:     "lowercase a boundary",
			input:    "aVariable",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isUnexportedName(tc.input)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchUnexportedFuncDecl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		declaration  goast.Decl
		functionName string
		wantMatch    bool
	}{
		{
			name: "matching unexported function",
			declaration: &goast.FuncDecl{
				Name: goast.NewIdent("myFunc"),
				Recv: nil,
			},
			functionName: "myFunc",
			wantMatch:    true,
		},
		{
			name: "non-matching function name",
			declaration: &goast.FuncDecl{
				Name: goast.NewIdent("otherFunc"),
				Recv: nil,
			},
			functionName: "myFunc",
			wantMatch:    false,
		},
		{
			name: "method receiver present",
			declaration: &goast.FuncDecl{
				Name: goast.NewIdent("myFunc"),
				Recv: &goast.FieldList{
					List: []*goast.Field{
						{Type: goast.NewIdent("MyType")},
					},
				},
			},
			functionName: "myFunc",
			wantMatch:    false,
		},
		{
			name: "nil function name",
			declaration: &goast.FuncDecl{
				Name: nil,
				Recv: nil,
			},
			functionName: "myFunc",
			wantMatch:    false,
		},
		{
			name:         "non-function declaration",
			declaration:  &goast.GenDecl{},
			functionName: "myFunc",
			wantMatch:    false,
		},
		{
			name:         "nil declaration",
			declaration:  nil,
			functionName: "myFunc",
			wantMatch:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := matchUnexportedFuncDecl(tc.declaration, tc.functionName)

			if tc.wantMatch {
				assert.NotNil(t, result)
				assert.Equal(t, tc.functionName, result.Name.Name)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestFieldTypeNeedsCorrection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		finalTypeAST    goast.Expr
		fieldInfoType   goast.Expr
		name            string
		expectedNeedsIt bool
	}{
		{
			name:            "same type does not need correction",
			finalTypeAST:    goast.NewIdent("string"),
			fieldInfoType:   goast.NewIdent("string"),
			expectedNeedsIt: false,
		},
		{
			name:            "different types need correction",
			finalTypeAST:    goast.NewIdent("MyAlias"),
			fieldInfoType:   goast.NewIdent("string"),
			expectedNeedsIt: true,
		},
		{
			name: "pointer vs non-pointer needs correction",
			finalTypeAST: &goast.StarExpr{
				X: goast.NewIdent("MyType"),
			},
			fieldInfoType:   goast.NewIdent("MyType"),
			expectedNeedsIt: true,
		},
		{
			name: "same pointer type does not need correction",
			finalTypeAST: &goast.StarExpr{
				X: goast.NewIdent("MyType"),
			},
			fieldInfoType: &goast.StarExpr{
				X: goast.NewIdent("MyType"),
			},
			expectedNeedsIt: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tr := &TypeResolver{}
			fieldInfo := &inspector_dto.FieldInfo{
				Type: tc.fieldInfoType,
			}

			result := tr.fieldTypeNeedsCorrection(tc.finalTypeAST, fieldInfo)

			assert.Equal(t, tc.expectedNeedsIt, result)
		})
	}
}

func TestDetermineFieldLookupContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		setupHarness        func(*typeResolverTestHarness)
		baseAnn             *ast_domain.GoGeneratorAnnotation
		expectedPackagePath string
		expectedFilePath    string
		setupCanonicalPath  string
	}{
		{
			name:         "no canonical path uses current context",
			setupHarness: func(_ *typeResolverTestHarness) {},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("MyType"),
					CanonicalPackagePath: "",
				},
			},
			expectedPackagePath: "test/pkg",
			expectedFilePath:    "/test.go",
		},
		{
			name: "canonical path triggers context switch with DTO",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.ResolveExprToNamedTypeWithMemoizationFunc = func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
					return &inspector_dto.Type{
						DefinedInFilePath: "/external/type.go",
					}, ""
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("ExternalType"),
					CanonicalPackagePath: "external/pkg",
				},
			},
			expectedPackagePath: "external/pkg",
			expectedFilePath:    "/external/type.go",
		},
		{
			name: "canonical path with no DTO falls back to package files",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.ResolveExprToNamedTypeWithMemoizationFunc = func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
					return nil, ""
				}
				h.Inspector.GetFilesForPackageFunc = func(_ string) []string {
					return []string{"/fallback/file.go"}
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("ExternalType"),
					CanonicalPackagePath: "external/pkg",
				},
			},
			expectedPackagePath: "external/pkg",
			expectedFilePath:    "/fallback/file.go",
		},
		{
			name: "canonical path with no files keeps original context",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.ResolveExprToNamedTypeWithMemoizationFunc = func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
					return nil, ""
				}
				h.Inspector.GetFilesForPackageFunc = func(_ string) []string {
					return []string{}
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("ExternalType"),
					CanonicalPackagePath: "external/pkg",
				},
			},
			expectedPackagePath: "test/pkg",
			expectedFilePath:    "/test.go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			tc.setupHarness(h)

			packagePath, filePath := h.Resolver.determineFieldLookupContext(context.Background(), h.Context, tc.baseAnn)

			assert.Equal(t, tc.expectedPackagePath, packagePath)
			assert.Equal(t, tc.expectedFilePath, filePath)
		})
	}
}

func TestTryResolveField(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setupHarness   func(*typeResolverTestHarness)
		baseAnn        *ast_domain.GoGeneratorAnnotation
		name           string
		propName       string
		expectedType   string
		checkFieldName string
		expectedFound  bool
	}{
		{
			name:          "nil base annotation returns not found",
			setupHarness:  func(_ *typeResolverTestHarness) {},
			baseAnn:       nil,
			propName:      "Field",
			expectedFound: false,
		},
		{
			name:         "nil resolved type returns not found",
			setupHarness: func(_ *typeResolverTestHarness) {},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: nil,
			},
			propName:      "Field",
			expectedFound: false,
		},
		{
			name:         "nil type expr returns not found",
			setupHarness: func(_ *typeResolverTestHarness) {},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: nil,
				},
			},
			propName:      "Field",
			expectedFound: false,
		},
		{
			name: "field not found in type",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.FindFieldInfoFunc = func(_ context.Context, _ goast.Expr, _, _, _ string) *inspector_dto.FieldInfo {
					return nil
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("MyStruct"),
				},
			},
			propName:      "NonExistent",
			expectedFound: false,
		},
		{
			name: "field found successfully",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.FindFieldInfoFunc = func(_ context.Context, _ goast.Expr, fieldName, _, _ string) *inspector_dto.FieldInfo {
					if fieldName == "Name" {
						return &inspector_dto.FieldInfo{
							Name:                 "Name",
							Type:                 goast.NewIdent("string"),
							ParentTypeName:       "User",
							CanonicalPackagePath: "test/pkg",
							DefiningFilePath:     "/test.go",
							DefiningPackagePath:  "test/pkg",
						}
					}
					return nil
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("User"),
				},
			},
			propName:       "Name",
			expectedFound:  true,
			expectedType:   "string",
			checkFieldName: "Name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			tc.setupHarness(h)

			result, _, found := h.Resolver.tryResolveField(
				context.Background(),
				h.Context,
				tc.baseAnn,
				tc.propName,
				ast_domain.Location{Line: 1, Column: 1},
			)

			assert.Equal(t, tc.expectedFound, found)

			if tc.expectedFound {
				require.NotNil(t, result)
				require.NotNil(t, result.ResolvedType)
				require.NotNil(t, result.ResolvedType.TypeExpression)

				if identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident); ok {
					assert.Equal(t, tc.expectedType, identifier.Name)
				}

				if tc.checkFieldName != "" && result.Symbol != nil {
					assert.Equal(t, tc.checkFieldName, result.Symbol.Name)
				}
			}
		})
	}
}

func TestTryResolveMethod(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		setupHarness  func(*typeResolverTestHarness)
		baseAnn       *ast_domain.GoGeneratorAnnotation
		methodName    string
		expectedFound bool
	}{
		{
			name:          "nil base annotation returns not found",
			setupHarness:  func(_ *typeResolverTestHarness) {},
			baseAnn:       nil,
			methodName:    "Method",
			expectedFound: false,
		},
		{
			name:         "nil resolved type returns not found",
			setupHarness: func(_ *typeResolverTestHarness) {},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: nil,
			},
			methodName:    "Method",
			expectedFound: false,
		},
		{
			name:         "nil type expr returns not found",
			setupHarness: func(_ *typeResolverTestHarness) {},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: nil,
				},
			},
			methodName:    "Method",
			expectedFound: false,
		},
		{
			name: "method not found in type",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.FindMethodInfoFunc = func(_ goast.Expr, _, _, _ string) *inspector_dto.Method {
					return nil
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("MyStruct"),
				},
			},
			methodName:    "NonExistent",
			expectedFound: false,
		},
		{
			name: "method found successfully",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.FindMethodInfoFunc = func(_ goast.Expr, methodName, _, _ string) *inspector_dto.Method {
					if methodName == "String" {
						return &inspector_dto.Method{
							Name: "String",
							Signature: inspector_dto.FunctionSignature{
								Params:  []string{},
								Results: []string{"string"},
							},
							DefinitionLine:     10,
							DefinitionColumn:   1,
							DefinitionFilePath: "/test.go",
						}
					}
					return nil
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("MyStruct"),
				},
			},
			methodName:    "String",
			expectedFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			tc.setupHarness(h)

			result, found := h.Resolver.tryResolveMethod(
				context.Background(),
				h.Context,
				tc.baseAnn,
				tc.methodName,
				ast_domain.Location{Line: 1, Column: 1},
			)

			assert.Equal(t, tc.expectedFound, found)

			if tc.expectedFound {
				require.NotNil(t, result)
				require.NotNil(t, result.ResolvedType)
				require.NotNil(t, result.Symbol)
				assert.Equal(t, tc.methodName, result.Symbol.Name)
			}
		})
	}
}

func TestHandleUnknownMember(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		setupHarness       func(*typeResolverTestHarness)
		baseTypeName       string
		propName           string
		expectedDiagSubstr string
	}{
		{
			name:               "unknown property generates diagnostic",
			setupHarness:       func(_ *typeResolverTestHarness) {},
			baseTypeName:       "MyStruct",
			propName:           "unknownProp",
			expectedDiagSubstr: "Property 'unknownProp' does not exist on type 'MyStruct'",
		},
		{
			name:               "length property on slice suggests len function",
			setupHarness:       func(_ *typeResolverTestHarness) {},
			baseTypeName:       "[]string",
			propName:           "length",
			expectedDiagSubstr: "Did you mean to use the built-in len() function",
		},
		{
			name: "similar property name suggests correction",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.GetAllFieldsAndMethodsFunc = func(_ goast.Expr, _, _ string) []string {
					return []string{"Name", "Email", "Age"}
				}
			},
			baseTypeName:       "User",
			propName:           "Naem",
			expectedDiagSubstr: "Did you mean 'Name'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			tc.setupHarness(h)

			var typeExpr goast.Expr
			if tc.baseTypeName == "[]string" {

				typeExpr = &goast.ArrayType{
					Elt: goast.NewIdent("string"),
				}
			} else {
				typeExpr = goast.NewIdent(tc.baseTypeName)
			}

			baseAnn := &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: typeExpr,
				},
			}

			memberExpr := &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "base"},
				Property: &ast_domain.Identifier{Name: tc.propName},
			}

			result := h.Resolver.handleUnknownMember(
				context.Background(),
				h.Context,
				baseAnn,
				tc.propName,
				memberExpr,
				ast_domain.Location{Line: 1, Column: 1},
			)

			require.NotNil(t, result)
			require.True(t, h.HasDiagnostics())

			diagnostic := h.GetFirstDiagnostic()
			assert.Contains(t, diagnostic.Message, tc.expectedDiagSubstr)
		})
	}
}

func TestDetermineMethodLookupContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		setupHarness        func(*typeResolverTestHarness)
		baseAnn             *ast_domain.GoGeneratorAnnotation
		expectedPackagePath string
		expectedFilePath    string
	}{
		{
			name:         "no canonical path uses current context",
			setupHarness: func(_ *typeResolverTestHarness) {},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("MyType"),
					CanonicalPackagePath: "",
				},
			},
			expectedPackagePath: "test/pkg",
			expectedFilePath:    "/test.go",
		},
		{
			name: "canonical path with DTO switches context",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.ResolveExprToNamedTypeWithMemoizationFunc = func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
					return &inspector_dto.Type{
						DefinedInFilePath: "/external/type.go",
					}, ""
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("ExternalType"),
					CanonicalPackagePath: "external/pkg",
				},
			},
			expectedPackagePath: "external/pkg",
			expectedFilePath:    "/external/type.go",
		},
		{
			name: "canonical path without DTO uses current file",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.ResolveExprToNamedTypeWithMemoizationFunc = func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
					return nil, ""
				}
			},
			baseAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("ExternalType"),
					CanonicalPackagePath: "external/pkg",
				},
			},
			expectedPackagePath: "external/pkg",
			expectedFilePath:    "/test.go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			tc.setupHarness(h)

			packagePath, filePath := h.Resolver.determineMethodLookupContext(context.Background(), h.Context, tc.baseAnn)

			assert.Equal(t, tc.expectedPackagePath, packagePath)
			assert.Equal(t, tc.expectedFilePath, filePath)
		})
	}
}

func TestFindCallSignature(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setupHarness  func(*typeResolverTestHarness)
		callee        *ast_domain.MemberExpression
		name          string
		expectedFound bool
	}{
		{
			name:         "nil base annotation returns not found",
			setupHarness: func(_ *typeResolverTestHarness) {},
			callee: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "unknown"},
				Property: &ast_domain.Identifier{Name: "Method"},
			},
			expectedFound: false,
		},
		{
			name: "package function found",
			setupHarness: func(h *typeResolverTestHarness) {
				h.DefineSymbol("fmt", nil)
				h.Context.Symbols.Define(Symbol{
					Name: "fmt",
					TypeInfo: &ast_domain.ResolvedTypeInfo{
						TypeExpression: nil,
						PackageAlias:   "fmt",
					},
				})
				h.Inspector.FindFuncSignatureFunc = func(packageAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
					if packageAlias == "fmt" && functionName == "Println" {
						return &inspector_dto.FunctionSignature{
							Params:  []string{"...any"},
							Results: []string{"int", "error"},
						}
					}
					return nil
				}
			},
			callee: &ast_domain.MemberExpression{
				Base: &ast_domain.Identifier{
					Name: "fmt",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: nil,
							PackageAlias:   "fmt",
						},
					},
				},
				Property: &ast_domain.Identifier{Name: "Println"},
			},
			expectedFound: true,
		},
		{
			name: "method on type found",
			setupHarness: func(h *typeResolverTestHarness) {
				h.Inspector.FindMethodInfoFunc = func(_ goast.Expr, methodName, _, _ string) *inspector_dto.Method {
					if methodName == "String" {
						return &inspector_dto.Method{
							Name: "String",
							Signature: inspector_dto.FunctionSignature{
								Params:  []string{},
								Results: []string{"string"},
							},
						}
					}
					return nil
				}
			},
			callee: &ast_domain.MemberExpression{
				Base: &ast_domain.Identifier{
					Name: "obj",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: goast.NewIdent("MyType"),
						},
					},
				},
				Property: &ast_domain.Identifier{Name: "String"},
			},
			expectedFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			tc.setupHarness(h)

			sig, _, _, found := h.Resolver.findCallSignature(context.Background(), h.Context, tc.callee)

			assert.Equal(t, tc.expectedFound, found)
			if tc.expectedFound {
				assert.NotNil(t, sig)
			}
		})
	}
}

func TestParseSignatureFromFuncDecl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		funcDecl        *goast.FuncDecl
		name            string
		expectedParams  int
		expectedResults int
		expectedNil     bool
	}{
		{
			name:        "nil function declaration",
			funcDecl:    nil,
			expectedNil: true,
		},
		{
			name: "nil function type",
			funcDecl: &goast.FuncDecl{
				Name: goast.NewIdent("myFunc"),
				Type: nil,
			},
			expectedNil: true,
		},
		{
			name: "function with no params or results",
			funcDecl: &goast.FuncDecl{
				Name: goast.NewIdent("myFunc"),
				Type: &goast.FuncType{
					Params:  nil,
					Results: nil,
				},
			},
			expectedNil:     false,
			expectedParams:  0,
			expectedResults: 0,
		},
		{
			name: "function with params and results",
			funcDecl: &goast.FuncDecl{
				Name: goast.NewIdent("myFunc"),
				Type: &goast.FuncType{
					Params: &goast.FieldList{
						List: []*goast.Field{
							{Names: []*goast.Ident{goast.NewIdent("a")}, Type: goast.NewIdent("int")},
							{Names: []*goast.Ident{goast.NewIdent("b")}, Type: goast.NewIdent("string")},
						},
					},
					Results: &goast.FieldList{
						List: []*goast.Field{
							{Type: goast.NewIdent("error")},
						},
					},
				},
			},
			expectedNil:     false,
			expectedParams:  2,
			expectedResults: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			result := h.Resolver.parseSignatureFromFuncDecl(tc.funcDecl, h.Context)

			if tc.expectedNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Len(t, result.Params, tc.expectedParams)
				assert.Len(t, result.Results, tc.expectedResults)
			}
		})
	}
}

func TestFindFuncDeclInCurrentContext(t *testing.T) {
	t.Parallel()

	t.Run("nil context returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := h.Resolver.findFuncDeclInCurrentContext(nil, "someFunc")
		assert.Nil(t, result)
	})

	t.Run("no virtual component for package returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := h.Resolver.findFuncDeclInCurrentContext(h.Context, "someFunc")
		assert.Nil(t, result)
	})

	t.Run("virtual component with nil source script returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		vm := h.Resolver.virtualModule
		vm.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: nil,
			},
		}

		result := h.Resolver.findFuncDeclInCurrentContext(h.Context, "someFunc")
		assert.Nil(t, result)
	})

	t.Run("finds matching function declaration", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		funcDecl := &goast.FuncDecl{
			Name: goast.NewIdent("MyFunc"),
			Type: &goast.FuncType{},
		}
		vm := h.Resolver.virtualModule
		vm.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name:  goast.NewIdent("testpkg"),
						Decls: []goast.Decl{funcDecl},
					},
				},
			},
		}

		result := h.Resolver.findFuncDeclInCurrentContext(h.Context, "MyFunc")
		require.NotNil(t, result)
		assert.Equal(t, "MyFunc", result.Name.Name)
	})

	t.Run("skips method declarations with receiver", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		methodDecl := &goast.FuncDecl{
			Name: goast.NewIdent("MyFunc"),
			Type: &goast.FuncType{},
			Recv: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("MyType")},
				},
			},
		}
		vm := h.Resolver.virtualModule
		vm.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name:  goast.NewIdent("testpkg"),
						Decls: []goast.Decl{methodDecl},
					},
				},
			},
		}

		result := h.Resolver.findFuncDeclInCurrentContext(h.Context, "MyFunc")
		assert.Nil(t, result)
	})

	t.Run("returns nil when name does not match", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		funcDecl := &goast.FuncDecl{
			Name: goast.NewIdent("OtherFunc"),
			Type: &goast.FuncType{},
		}
		vm := h.Resolver.virtualModule
		vm.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name:  goast.NewIdent("testpkg"),
						Decls: []goast.Decl{funcDecl},
					},
				},
			},
		}

		result := h.Resolver.findFuncDeclInCurrentContext(h.Context, "MyFunc")
		assert.Nil(t, result)
	})
}

func TestFindUnexportedFuncDeclInCurrentContext(t *testing.T) {
	t.Parallel()

	t.Run("nil context returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := h.Resolver.findUnexportedFuncDeclInCurrentContext(nil, "myFunc")
		assert.Nil(t, result)
	})

	t.Run("no virtual component returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := h.Resolver.findUnexportedFuncDeclInCurrentContext(h.Context, "myFunc")
		assert.Nil(t, result)
	})

	t.Run("exported name returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		funcDecl := &goast.FuncDecl{
			Name: goast.NewIdent("MyFunc"),
			Type: &goast.FuncType{},
		}
		vm := h.Resolver.virtualModule
		vm.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name:  goast.NewIdent("testpkg"),
						Decls: []goast.Decl{funcDecl},
					},
				},
			},
		}

		result := h.Resolver.findUnexportedFuncDeclInCurrentContext(h.Context, "MyFunc")
		assert.Nil(t, result)
	})

	t.Run("finds matching unexported function", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		funcDecl := &goast.FuncDecl{
			Name: goast.NewIdent("helperFunc"),
			Type: &goast.FuncType{},
		}
		vm := h.Resolver.virtualModule
		vm.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name:  goast.NewIdent("testpkg"),
						Decls: []goast.Decl{funcDecl},
					},
				},
			},
		}

		result := h.Resolver.findUnexportedFuncDeclInCurrentContext(h.Context, "helperFunc")
		require.NotNil(t, result)
		assert.Equal(t, "helperFunc", result.Name.Name)
	})

	t.Run("nil script returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		vm := h.Resolver.virtualModule
		vm.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: nil,
			},
		}

		result := h.Resolver.findUnexportedFuncDeclInCurrentContext(h.Context, "myFunc")
		assert.Nil(t, result)
	})
}

func TestResolveBaseType(t *testing.T) {
	t.Parallel()

	t.Run("returns same file when no redirect", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Inspector.ResolveToUnderlyingASTWithContextFunc = func(_ context.Context, typeExpr goast.Expr, _ string) (goast.Expr, string) {
			return typeExpr, ""
		}

		baseAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyType"),
				PackageAlias:   "testpkg",
			},
		}

		resolvedAST, packagePath, filePath := h.Resolver.resolveBaseType(
			context.Background(), h.Context, baseAnn, "test/pkg", "/test.go",
		)

		require.NotNil(t, resolvedAST)
		assert.Equal(t, "test/pkg", packagePath)
		assert.Equal(t, "/test.go", filePath)
	})

	t.Run("switches context when resolved to different file", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		resolvedType := goast.NewIdent("OtherType")
		h.Inspector.ResolveToUnderlyingASTWithContextFunc = func(_ context.Context, _ goast.Expr, _ string) (goast.Expr, string) {
			return resolvedType, "/other/file.go"
		}
		h.Inspector.PackagePathForFileFunc = func(_ string) string {
			return "other/pkg"
		}

		baseAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyType"),
				PackageAlias:   "testpkg",
			},
		}

		resolvedAST, packagePath, filePath := h.Resolver.resolveBaseType(
			context.Background(), h.Context, baseAnn, "test/pkg", "/test.go",
		)

		assert.Equal(t, resolvedType, resolvedAST)
		assert.Equal(t, "other/pkg", packagePath)
		assert.Equal(t, "/other/file.go", filePath)
	})

	t.Run("keeps importer pkg when PackagePathForFile returns empty", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Inspector.ResolveToUnderlyingASTWithContextFunc = func(_ context.Context, typeExpr goast.Expr, _ string) (goast.Expr, string) {
			return typeExpr, "/different/file.go"
		}
		h.Inspector.PackagePathForFileFunc = func(_ string) string {
			return ""
		}

		baseAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyType"),
				PackageAlias:   "testpkg",
			},
		}

		_, packagePath, filePath := h.Resolver.resolveBaseType(
			context.Background(), h.Context, baseAnn, "test/pkg", "/test.go",
		)

		assert.Equal(t, "test/pkg", packagePath)
		assert.Equal(t, "/different/file.go", filePath)
	})
}

func TestFindPackageFunctionSignature(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no signature found", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		callee := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "pkg"},
			Property: &ast_domain.Identifier{Name: "Func"},
		}
		baseAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("pkg"),
				PackageAlias:   "pkg",
			},
		}

		sig, _, found := h.Resolver.findPackageFunctionSignature(h.Context, baseAnn, "Func", callee)

		assert.Nil(t, sig)
		assert.False(t, found)
	})

	t.Run("returns signature when found", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Inspector.FindFuncSignatureFunc = func(packageAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
			if packageAlias == "fmt" && functionName == "Println" {
				return &inspector_dto.FunctionSignature{
					Params:  []string{"...any"},
					Results: []string{"int", "error"},
				}
			}
			return nil
		}

		callee := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "fmt"},
			Property: &ast_domain.Identifier{Name: "Println"},
		}
		baseAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("fmt"),
				PackageAlias:   "fmt",
			},
		}

		sig, _, found := h.Resolver.findPackageFunctionSignature(h.Context, baseAnn, "Println", callee)

		require.NotNil(t, sig)
		assert.True(t, found)
		assert.Len(t, sig.Params, 1)
	})
}

func TestFindCallSignature_AdditionalPaths(t *testing.T) {
	t.Parallel()

	t.Run("returns false when base annotation is nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		callee := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "obj"},
			Property: &ast_domain.Identifier{Name: "Method"},
		}

		sig, baseAnn, methodInfo, found := h.Resolver.findCallSignature(context.Background(), h.Context, callee)

		assert.Nil(t, sig)
		assert.Nil(t, baseAnn)
		assert.Nil(t, methodInfo)
		assert.False(t, found)
	})

	t.Run("returns false when property is not an identifier", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		base := &ast_domain.Identifier{Name: "obj"}
		setAnnotationOnExpression(base, &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyStruct"),
			},
		})

		callee := &ast_domain.MemberExpression{
			Base:     base,
			Property: &ast_domain.StringLiteral{Value: "computed"},
		}

		sig, baseAnn, methodInfo, found := h.Resolver.findCallSignature(context.Background(), h.Context, callee)

		assert.Nil(t, sig)
		assert.NotNil(t, baseAnn)
		assert.Nil(t, methodInfo)
		assert.False(t, found)
	})

	t.Run("tries package function path when TypeExpr is nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		h.Inspector.FindFuncSignatureFunc = func(packageName, functionName, packagePath, filePath string) *inspector_dto.FunctionSignature {
			if functionName == "Sprintf" && packageName == "fmt" {
				return &inspector_dto.FunctionSignature{
					Params: []string{"string", "...any"},
				}
			}
			return nil
		}

		base := &ast_domain.Identifier{Name: "fmt"}
		setAnnotationOnExpression(base, &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: nil,
				PackageAlias:   "fmt",
			},
		})

		callee := &ast_domain.MemberExpression{
			Base:     base,
			Property: &ast_domain.Identifier{Name: "Sprintf"},
		}

		sig, _, _, found := h.Resolver.findCallSignature(context.Background(), h.Context, callee)

		require.NotNil(t, sig)
		assert.True(t, found)
		assert.Equal(t, []string{"string", "...any"}, sig.Params)
	})

	t.Run("returns false when ResolvedType is nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		base := &ast_domain.Identifier{Name: "obj"}
		setAnnotationOnExpression(base, &ast_domain.GoGeneratorAnnotation{
			ResolvedType: nil,
		})

		callee := &ast_domain.MemberExpression{
			Base:     base,
			Property: &ast_domain.Identifier{Name: "Method"},
		}

		sig, baseAnn, methodInfo, found := h.Resolver.findCallSignature(context.Background(), h.Context, callee)

		assert.Nil(t, sig)
		assert.NotNil(t, baseAnn)
		assert.Nil(t, methodInfo)
		assert.False(t, found)
	})

	t.Run("finds method on type via FindMethodSignature", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		h.Inspector.FindMethodInfoFunc = func(typeExpr goast.Expr, methodName, packagePath, filePath string) *inspector_dto.Method {
			if methodName == "Len" {
				return &inspector_dto.Method{
					Name: "Len",
					Signature: inspector_dto.FunctionSignature{
						Params:  nil,
						Results: []string{"int"},
					},
				}
			}
			return nil
		}

		base := &ast_domain.Identifier{Name: "col"}
		setAnnotationOnExpression(base, &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression:       goast.NewIdent("MyCollection"),
				CanonicalPackagePath: "test/pkg",
			},
		})

		callee := &ast_domain.MemberExpression{
			Base:     base,
			Property: &ast_domain.Identifier{Name: "Len"},
		}

		sig, _, methodInfo, found := h.Resolver.findCallSignature(context.Background(), h.Context, callee)

		require.NotNil(t, sig)
		assert.True(t, found)
		assert.Equal(t, []string{"int"}, sig.Results)
		assert.Equal(t, "Len", methodInfo.Name)
	})
}

func TestCorrectFieldTypeContext_NoCorrection(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	fieldInfo := &inspector_dto.FieldInfo{
		Name:                 "Name",
		Type:                 goast.NewIdent("string"),
		PackageAlias:         "",
		CanonicalPackagePath: "",
	}

	canonPath, packageAlias := h.Resolver.correctFieldTypeContext(context.Background(), h.Context, fieldInfo, goast.NewIdent("string"))

	assert.Empty(t, canonPath)
	assert.Empty(t, packageAlias)
}

func TestCorrectFieldTypeContext_TypeDiffers_ButUnresolvable(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	fieldInfo := &inspector_dto.FieldInfo{
		Name:                 "Value",
		Type:                 goast.NewIdent("MyAlias"),
		PackageAlias:         "mypkg",
		CanonicalPackagePath: "original/pkg",
		DefiningPackagePath:  "defining/pkg",
		DefiningFilePath:     "/defining/file.go",
	}

	resolvedType := goast.NewIdent("UnderlyingType")

	canonPath, packageAlias := h.Resolver.correctFieldTypeContext(context.Background(), h.Context, fieldInfo, resolvedType)

	assert.Equal(t, "original/pkg", canonPath)
	assert.Equal(t, "mypkg", packageAlias)
}
