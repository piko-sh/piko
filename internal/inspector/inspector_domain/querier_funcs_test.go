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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func newQuerierWithFuncs() *TypeQuerier {
	return &TypeQuerier{
		typeData: &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
					Funcs: map[string]*inspector_dto.Function{
						"NewUser": {
							Name:               "NewUser",
							DefinitionLine:     10,
							DefinitionColumn:   6,
							DefinitionFilePath: "/src/user.go",
							Signature: inspector_dto.FunctionSignature{
								Params:  []string{"string", "int"},
								Results: []string{"*User", "error"},
							},
						},
						"NoReturn": {
							Name:               "NoReturn",
							DefinitionLine:     20,
							DefinitionColumn:   6,
							DefinitionFilePath: "/src/user.go",
							Signature: inspector_dto.FunctionSignature{
								Params: []string{"string"},
							},
						},
					},
					FileImports: map[string]map[string]string{
						"/src/user.go": {"pkg": "my/pkg"},
					},
				},
				"my/main": {
					Name: "main",
					Path: "my/main",
					FileImports: map[string]map[string]string{
						"/src/app.go": {
							"pkg": "my/pkg",
						},
					},
				},
			},
		},
	}
}

func TestFindFuncSignature(t *testing.T) {
	t.Parallel()

	t.Run("should find function signature via alias resolution", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithFuncs()
		sig := querier.FindFuncSignature("pkg", "NewUser", "my/main", "/src/app.go")
		require.NotNil(t, sig)
		assert.Equal(t, []string{"string", "int"}, sig.Params)
		assert.Equal(t, []string{"*User", "error"}, sig.Results)
	})

	t.Run("should capitalise lowercase function name", func(t *testing.T) {
		t.Parallel()

		querier := newQuerierWithFuncs()
		sig := querier.FindFuncSignature("pkg", "newUser", "my/main", "/src/app.go")
		require.NotNil(t, sig)
		assert.Equal(t, []string{"*User", "error"}, sig.Results)
	})

	t.Run("should return nil for nil typeData", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{typeData: nil}
		assert.Nil(t, querier.FindFuncSignature("pkg", "NewUser", "my/main", "/src/app.go"))
	})

	t.Run("should return nil when alias resolution fails", func(t *testing.T) {
		t.Parallel()

		querier := newQuerierWithFuncs()
		assert.Nil(t, querier.FindFuncSignature("unknown", "NewUser", "my/main", "/src/app.go"))
	})

	t.Run("should return nil when target package not found", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my/main": {
						Name: "main",
						Path: "my/main",
						FileImports: map[string]map[string]string{
							"/src/app.go": {"missing": "my/missing"},
						},
					},
				},
			},
		}
		assert.Nil(t, querier.FindFuncSignature("missing", "Foo", "my/main", "/src/app.go"))
	})

	t.Run("should return nil for missing function", func(t *testing.T) {
		t.Parallel()

		querier := newQuerierWithFuncs()
		assert.Nil(t, querier.FindFuncSignature("pkg", "NonExistent", "my/main", "/src/app.go"))
	})
}

func TestFindFuncReturnType(t *testing.T) {
	t.Parallel()

	t.Run("should return AST for first return type", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithFuncs()
		result := querier.FindFuncReturnType("pkg", "NewUser", "my/main", "/src/app.go")
		assert.NotNil(t, result)
	})

	t.Run("should return nil for function with no returns", func(t *testing.T) {
		t.Parallel()

		querier := newQuerierWithFuncs()
		result := querier.FindFuncReturnType("pkg", "NoReturn", "my/main", "/src/app.go")
		assert.Nil(t, result)
	})

	t.Run("should return nil when function not found", func(t *testing.T) {
		t.Parallel()

		querier := newQuerierWithFuncs()
		result := querier.FindFuncReturnType("pkg", "NonExistent", "my/main", "/src/app.go")
		assert.Nil(t, result)
	})

	t.Run("should return nil for nil typeData", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{typeData: nil}
		assert.Nil(t, querier.FindFuncReturnType("pkg", "NewUser", "my/main", "/src/app.go"))
	})
}

func TestFindFuncInfo(t *testing.T) {
	t.Parallel()

	t.Run("should return full Function DTO with location", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithFuncs()
		inspectedFunction := querier.FindFuncInfo("pkg", "NewUser", "my/main", "/src/app.go")
		require.NotNil(t, inspectedFunction)
		assert.Equal(t, "NewUser", inspectedFunction.Name)
		assert.Equal(t, 10, inspectedFunction.DefinitionLine)
		assert.Equal(t, 6, inspectedFunction.DefinitionColumn)
		assert.Equal(t, "/src/user.go", inspectedFunction.DefinitionFilePath)
	})

	t.Run("should capitalise lowercase function name", func(t *testing.T) {
		t.Parallel()

		querier := newQuerierWithFuncs()
		inspectedFunction := querier.FindFuncInfo("pkg", "newUser", "my/main", "/src/app.go")
		require.NotNil(t, inspectedFunction)
		assert.Equal(t, "NewUser", inspectedFunction.Name)
	})

	t.Run("should return nil for nil typeData", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{typeData: nil}
		assert.Nil(t, querier.FindFuncInfo("pkg", "NewUser", "my/main", "/src/app.go"))
	})

	t.Run("should return nil when alias resolution fails", func(t *testing.T) {
		t.Parallel()

		querier := newQuerierWithFuncs()
		assert.Nil(t, querier.FindFuncInfo("unknown", "NewUser", "my/main", "/src/app.go"))
	})

	t.Run("should return nil for missing function", func(t *testing.T) {
		t.Parallel()

		querier := newQuerierWithFuncs()
		assert.Nil(t, querier.FindFuncInfo("pkg", "NonExistent", "my/main", "/src/app.go"))
	})
}
