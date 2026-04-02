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

func newQuerierWithVariables() *TypeQuerier {
	return &TypeQuerier{
		typeData: &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
					Variables: map[string]*inspector_dto.Variable{
						"MaxRetries": {
							Name:       "MaxRetries",
							TypeString: "int",
						},
						"DefaultName": {
							Name:       "DefaultName",
							TypeString: "string",
						},
					},
					FileImports: map[string]map[string]string{
						"/src/main.go": {
							"pkg": "my/pkg",
						},
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

func TestFindPackageVariable(t *testing.T) {
	t.Parallel()
	t.Run("should find variable via alias resolution", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithVariables()
		v := querier.FindPackageVariable("pkg", "MaxRetries", "my/main", "/src/app.go")
		require.NotNil(t, v)
		assert.Equal(t, "MaxRetries", v.Name)
		assert.Equal(t, "int", v.TypeString)
	})

	t.Run("should capitalise lowercase variable name", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithVariables()
		v := querier.FindPackageVariable("pkg", "maxRetries", "my/main", "/src/app.go")
		require.NotNil(t, v)
		assert.Equal(t, "MaxRetries", v.Name)
	})

	t.Run("should return nil for nil typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: nil}
		assert.Nil(t, querier.FindPackageVariable("pkg", "MaxRetries", "my/main", "/src/app.go"))
	})

	t.Run("should return nil for nil Packages", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: &inspector_dto.TypeData{Packages: nil}}
		assert.Nil(t, querier.FindPackageVariable("pkg", "MaxRetries", "my/main", "/src/app.go"))
	})

	t.Run("should return nil when alias resolution fails", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithVariables()
		assert.Nil(t, querier.FindPackageVariable("unknown", "MaxRetries", "my/main", "/src/app.go"))
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
							"/src/app.go": {
								"missing": "my/missing",
							},
						},
					},
				},
			},
		}
		assert.Nil(t, querier.FindPackageVariable("missing", "Foo", "my/main", "/src/app.go"))
	})

	t.Run("should return nil for nil Variables map", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my/pkg": {
						Name:      "pkg",
						Path:      "my/pkg",
						Variables: nil,
						FileImports: map[string]map[string]string{
							"/src/main.go": {"pkg": "my/pkg"},
						},
					},
					"my/main": {
						Name: "main",
						Path: "my/main",
						FileImports: map[string]map[string]string{
							"/src/app.go": {"pkg": "my/pkg"},
						},
					},
				},
			},
		}
		assert.Nil(t, querier.FindPackageVariable("pkg", "Foo", "my/main", "/src/app.go"))
	})

	t.Run("should return nil for missing variable name", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithVariables()
		assert.Nil(t, querier.FindPackageVariable("pkg", "NonExistent", "my/main", "/src/app.go"))
	})
}

func TestFindPackageVariableType(t *testing.T) {
	t.Parallel()
	t.Run("should return type string for found variable", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithVariables()
		result := querier.FindPackageVariableType("pkg", "MaxRetries", "my/main", "/src/app.go")
		assert.Equal(t, "int", result)
	})

	t.Run("should return empty string when variable not found", func(t *testing.T) {
		t.Parallel()
		querier := newQuerierWithVariables()
		result := querier.FindPackageVariableType("pkg", "NonExistent", "my/main", "/src/app.go")
		assert.Equal(t, "", result)
	})

	t.Run("should return empty string for nil typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: nil}
		result := querier.FindPackageVariableType("pkg", "Foo", "my/main", "/src/app.go")
		assert.Equal(t, "", result)
	})
}
