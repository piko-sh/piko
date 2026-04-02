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

func TestGetAllSymbols(t *testing.T) {
	t.Parallel()
	t.Run("should collect type, method, field, and function symbols", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:              "User",
								DefinitionLine:    5,
								DefinitionColumn:  6,
								DefinedInFilePath: "/src/user.go",
								Fields: []*inspector_dto.Field{
									{
										Name:               "Name",
										TypeString:         "string",
										DefinitionFilePath: "/src/user.go",
										DefinitionLine:     6,
										DefinitionColumn:   2,
									},
									{
										Name:       "secret",
										TypeString: "string",
									},
								},
								Methods: []*inspector_dto.Method{
									{
										Name:               "GetName",
										DefinitionFilePath: "/src/user.go",
										DefinitionLine:     10,
										DefinitionColumn:   6,
									},
								},
							},
						},
						Funcs: map[string]*inspector_dto.Function{
							"NewUser": {
								Name:               "NewUser",
								DefinitionFilePath: "/src/user.go",
								DefinitionLine:     15,
								DefinitionColumn:   6,
							},
						},
					},
				},
			},
		}

		symbols := querier.GetAllSymbols()
		require.NotEmpty(t, symbols)

		symbolsByKind := make(map[string][]inspector_dto.WorkspaceSymbol)
		for _, s := range symbols {
			symbolsByKind[s.Kind] = append(symbolsByKind[s.Kind], s)
		}

		require.Len(t, symbolsByKind["type"], 1)
		assert.Equal(t, "User", symbolsByKind["type"][0].Name)
		assert.Equal(t, "pkg/a", symbolsByKind["type"][0].PackagePath)
		assert.Equal(t, "a", symbolsByKind["type"][0].PackageName)
		assert.Equal(t, 5, symbolsByKind["type"][0].Line)

		require.Len(t, symbolsByKind["method"], 1)
		assert.Equal(t, "GetName", symbolsByKind["method"][0].Name)
		assert.Equal(t, "User", symbolsByKind["method"][0].ContainerName)

		require.Len(t, symbolsByKind["field"], 1, "unexported field 'secret' should be excluded")
		assert.Equal(t, "Name", symbolsByKind["field"][0].Name)
		assert.Equal(t, "User", symbolsByKind["field"][0].ContainerName)

		require.Len(t, symbolsByKind["function"], 1)
		assert.Equal(t, "NewUser", symbolsByKind["function"][0].Name)
		assert.Equal(t, "", symbolsByKind["function"][0].ContainerName)
	})

	t.Run("should return empty for nil typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: nil}
		symbols := querier.GetAllSymbols()
		assert.Empty(t, symbols)
	})

	t.Run("should return empty for nil Packages", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{Packages: nil},
		}
		symbols := querier.GetAllSymbols()
		assert.Empty(t, symbols)
	})

	t.Run("should skip nil package", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/nil": nil,
				},
			},
		}
		symbols := querier.GetAllSymbols()
		assert.Empty(t, symbols)
	})

	t.Run("should skip nil type info", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						NamedTypes: map[string]*inspector_dto.Type{
							"NilType": nil,
						},
					},
				},
			},
		}
		symbols := querier.GetAllSymbols()
		assert.Empty(t, symbols)
	})

	t.Run("should skip nil method and nil field", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						NamedTypes: map[string]*inspector_dto.Type{
							"T": {
								Name: "T",
								Methods: []*inspector_dto.Method{
									nil,
									{Name: "Valid"},
								},
								Fields: []*inspector_dto.Field{
									nil,
									{Name: "Valid"},
								},
							},
						},
					},
				},
			},
		}
		symbols := querier.GetAllSymbols()
		methodCount := 0
		fieldCount := 0
		for _, s := range symbols {
			if s.Kind == "method" {
				methodCount++
			}
			if s.Kind == "field" {
				fieldCount++
			}
		}
		assert.Equal(t, 1, methodCount)
		assert.Equal(t, 1, fieldCount)
	})

	t.Run("should skip nil function info", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						Funcs: map[string]*inspector_dto.Function{
							"NilFunc": nil,
							"Good":    {Name: "Good"},
						},
					},
				},
			},
		}
		symbols := querier.GetAllSymbols()
		funcCount := 0
		for _, s := range symbols {
			if s.Kind == "function" {
				funcCount++
			}
		}
		assert.Equal(t, 1, funcCount)
	})
}

func TestGetImplementationIndex(t *testing.T) {
	t.Parallel()
	t.Run("should lazily build and cache the index", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						NamedTypes: map[string]*inspector_dto.Type{
							"Doer": {
								Name:                 "Doer",
								UnderlyingTypeString: "interface{Do()}",
								Methods: []*inspector_dto.Method{
									{Name: "Do", Signature: inspector_dto.FunctionSignature{}},
								},
							},
						},
					},
				},
			},
		}

		idx1 := querier.GetImplementationIndex()
		require.NotNil(t, idx1)

		idx2 := querier.GetImplementationIndex()
		assert.Equal(t, idx1, idx2, "should return the same index on second call")
	})
}

func TestGetTypeHierarchyIndex(t *testing.T) {
	t.Parallel()
	t.Run("should lazily build and cache the index", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						NamedTypes: map[string]*inspector_dto.Type{
							"Base":    {Name: "Base"},
							"Derived": {Name: "Derived"},
						},
					},
				},
			},
		}

		idx1 := querier.GetTypeHierarchyIndex()
		require.NotNil(t, idx1)

		idx2 := querier.GetTypeHierarchyIndex()
		assert.Equal(t, idx1, idx2, "should return the same index on second call")
	})
}
