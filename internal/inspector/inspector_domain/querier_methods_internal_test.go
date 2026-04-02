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

package inspector_domain

import (
	goast "go/ast"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func newQuerierWithMethods(t *testing.T) *TypeQuerier {
	t.Helper()

	td := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"my/pkg": {
				Name: "pkg",
				Path: "my/pkg",
				NamedTypes: map[string]*inspector_dto.Type{
					"User": {
						Name:              "User",
						PackagePath:       "my/pkg",
						DefinedInFilePath: "/src/user.go",
						DefinitionLine:    5,
						DefinitionColumn:  6,
						Methods: []*inspector_dto.Method{
							{
								Name:                 "GetName",
								DeclaringPackagePath: "my/pkg",
								DeclaringTypeName:    "User",
								IsPointerReceiver:    false,
								DefinitionFilePath:   "/src/user.go",
								DefinitionLine:       10,
								DefinitionColumn:     1,
								Signature: inspector_dto.FunctionSignature{
									Results: []string{"string"},
								},
							},
							{
								Name:                 "SetName",
								DeclaringPackagePath: "my/pkg",
								DeclaringTypeName:    "User",
								IsPointerReceiver:    true,
								DefinitionFilePath:   "/src/user.go",
								DefinitionLine:       15,
								DefinitionColumn:     1,
								Signature: inspector_dto.FunctionSignature{
									Params: []string{"string"},
								},
							},
						},
						Fields: []*inspector_dto.Field{
							{
								Name:       "Name",
								TypeString: "string",
							},
						},
					},
				},
				FileImports: map[string]map[string]string{
					"/src/main.go": {
						"pkg": "my/pkg",
					},
				},
			},
		},
		FileToPackage: map[string]string{
			"/src/user.go": "my/pkg",
			"/src/main.go": "my/pkg",
		},
	}

	return &TypeQuerier{
		typeData:           td,
		namedTypeCache:     sync.Map{},
		underlyingASTCache: sync.Map{},
	}
}

func TestFindMethodInfo(t *testing.T) {
	t.Parallel()
	t.Run("nil base type returns nil", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithMethods(t)
		result := querier.FindMethodInfo(nil, "GetName", "my/pkg", "/src/main.go")
		assert.Nil(t, result)
	})

	t.Run("finds value receiver method on type", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithMethods(t)
		baseType := goast.NewIdent("User")
		result := querier.FindMethodInfo(baseType, "GetName", "my/pkg", "/src/user.go")
		require.NotNil(t, result)
		assert.Equal(t, "GetName", result.Name)
		assert.Equal(t, 10, result.DefinitionLine)
	})

	t.Run("finds pointer receiver method on pointer type", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithMethods(t)
		baseType := &goast.StarExpr{X: goast.NewIdent("User")}
		result := querier.FindMethodInfo(baseType, "SetName", "my/pkg", "/src/user.go")
		require.NotNil(t, result)
		assert.Equal(t, "SetName", result.Name)
		assert.Equal(t, 15, result.DefinitionLine)
	})

	t.Run("method not found returns nil", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithMethods(t)
		baseType := goast.NewIdent("User")
		result := querier.FindMethodInfo(baseType, "NonExistent", "my/pkg", "/src/user.go")
		assert.Nil(t, result)
	})

	t.Run("finds pointer receiver method via addressable retry for external value", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithMethods(t)

		baseType := &goast.SelectorExpr{
			X:   goast.NewIdent("pkg"),
			Sel: goast.NewIdent("User"),
		}
		result := querier.FindMethodInfo(baseType, "SetName", "my/pkg", "/src/main.go")

		require.NotNil(t, result)
		assert.Equal(t, "SetName", result.Name)
	})
}

func TestIsSameType(t *testing.T) {
	t.Parallel()
	td := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"my/pkg": {
				Name: "pkg",
				Path: "my/pkg",
				NamedTypes: map[string]*inspector_dto.Type{
					"User":    {Name: "User", PackagePath: "my/pkg"},
					"Account": {Name: "Account", PackagePath: "my/pkg"},
				},
			},
			"other/pkg": {
				Name: "pkg",
				Path: "other/pkg",
				NamedTypes: map[string]*inspector_dto.Type{
					"User": {Name: "User", PackagePath: "other/pkg"},
				},
			},
		},
	}
	querier := &TypeQuerier{
		typeData:           td,
		namedTypeCache:     sync.Map{},
		underlyingASTCache: sync.Map{},
	}

	userA := td.Packages["my/pkg"].NamedTypes["User"]
	userB := td.Packages["my/pkg"].NamedTypes["User"]
	accountA := td.Packages["my/pkg"].NamedTypes["Account"]
	otherUser := td.Packages["other/pkg"].NamedTypes["User"]

	tests := []struct {
		a        *inspector_dto.Type
		b        *inspector_dto.Type
		name     string
		expected bool
	}{
		{name: "same type same package", a: userA, b: userB, expected: true},
		{name: "different type same package", a: userA, b: accountA, expected: false},
		{name: "same name different package", a: userA, b: otherUser, expected: false},
		{name: "nil a", a: nil, b: userA, expected: false},
		{name: "nil b", a: userA, b: nil, expected: false},
		{name: "both nil", a: nil, b: nil, expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := querier.isSameType(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindMethodReturnTypeInDTO(t *testing.T) {
	t.Parallel()
	querier := &TypeQuerier{}
	methods := []*inspector_dto.Method{
		{
			Name:      "GetName",
			Signature: inspector_dto.FunctionSignature{Results: []string{"string"}},
		},
		{
			Name:      "Save",
			Signature: inspector_dto.FunctionSignature{Results: []string{"error"}},
		},
		{
			Name:      "Close",
			Signature: inspector_dto.FunctionSignature{},
		},
	}

	tests := []struct {
		name       string
		methodName string
		expected   string
		expectNil  bool
	}{
		{name: "found with result", methodName: "GetName", expected: "string"},
		{name: "found with error result", methodName: "Save", expected: "error"},
		{name: "found but no results", methodName: "Close", expectNil: true},
		{name: "not found", methodName: "Unknown", expectNil: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := querier.findMethodReturnTypeInDTO(methods, tc.methodName)
			if tc.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expected, goastutil.ASTToTypeString(result))
			}
		})
	}

	t.Run("nil methods", func(t *testing.T) {
		t.Parallel()
		result := querier.findMethodReturnTypeInDTO(nil, "GetName")
		assert.Nil(t, result)
	})

	t.Run("empty methods", func(t *testing.T) {
		t.Parallel()
		result := querier.findMethodReturnTypeInDTO([]*inspector_dto.Method{}, "GetName")
		assert.Nil(t, result)
	})
}

func TestIsExternalValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression goast.Expr
		name       string
		expected   bool
	}{
		{
			name:       "selector expr is external",
			expression: &goast.SelectorExpr{X: goast.NewIdent("pkg"), Sel: goast.NewIdent("Type")},
			expected:   true,
		},
		{
			name:       "simple ident is not external",
			expression: goast.NewIdent("User"),
			expected:   false,
		},
		{
			name:       "star expr is not external",
			expression: &goast.StarExpr{X: goast.NewIdent("User")},
			expected:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isExternalValue(tc.expression)
			assert.Equal(t, tc.expected, result)
		})
	}
}
