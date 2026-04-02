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

func TestTypeHierarchyIndex(t *testing.T) {
	t.Parallel()
	t.Run("should track simple same-package embedding", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Base": {
							Name:              "Base",
							DefinitionLine:    5,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/base.go",
						},
						"Derived": {
							Name:              "Derived",
							DefinitionLine:    15,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/derived.go",
							Fields: []*inspector_dto.Field{
								{
									Name:        "",
									TypeString:  "Base",
									IsEmbedded:  true,
									PackagePath: "pkg/a",
								},
							},
						},
					},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)

		supertypes := index.GetSupertypes("pkg/a", "Derived")
		require.Len(t, supertypes, 1)
		assert.Equal(t, "Base", supertypes[0].TypeName)
		assert.Equal(t, "pkg/a", supertypes[0].PackagePath)
		assert.Equal(t, "/src/base.go", supertypes[0].FilePath)
		assert.Equal(t, 5, supertypes[0].Line)
		assert.Equal(t, 6, supertypes[0].Col)

		subtypes := index.GetSubtypes("pkg/a", "Base")
		require.Len(t, subtypes, 1)
		assert.Equal(t, "Derived", subtypes[0].TypeName)
		assert.Equal(t, "pkg/a", subtypes[0].PackagePath)
		assert.Equal(t, "/src/derived.go", subtypes[0].FilePath)
		assert.Equal(t, 15, subtypes[0].Line)
		assert.Equal(t, 6, subtypes[0].Col)
	})

	t.Run("should track cross-package embedding", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/base": {
					Name: "base",
					Path: "pkg/base",
					NamedTypes: map[string]*inspector_dto.Type{
						"Model": {
							Name:              "Model",
							DefinitionLine:    3,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/model.go",
						},
					},
				},
				"pkg/user": {
					Name: "user",
					Path: "pkg/user",
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {
							Name:              "User",
							DefinitionLine:    10,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/user.go",
							Fields: []*inspector_dto.Field{
								{
									Name:        "",
									TypeString:  "base.Model",
									IsEmbedded:  true,
									PackagePath: "pkg/base",
								},
							},
						},
					},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)

		supertypes := index.GetSupertypes("pkg/user", "User")
		require.Len(t, supertypes, 1)
		assert.Equal(t, "Model", supertypes[0].TypeName)
		assert.Equal(t, "pkg/base", supertypes[0].PackagePath)

		subtypes := index.GetSubtypes("pkg/base", "Model")
		require.Len(t, subtypes, 1)
		assert.Equal(t, "User", subtypes[0].TypeName)
		assert.Equal(t, "pkg/user", subtypes[0].PackagePath)
	})

	t.Run("should handle pointer embedding", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Base": {
							Name:              "Base",
							DefinitionLine:    1,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/a.go",
						},
						"Child": {
							Name:              "Child",
							DefinitionLine:    5,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/a.go",
							Fields: []*inspector_dto.Field{
								{
									Name:        "",
									TypeString:  "*Base",
									IsEmbedded:  true,
									PackagePath: "pkg/a",
								},
							},
						},
					},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)

		supertypes := index.GetSupertypes("pkg/a", "Child")
		require.Len(t, supertypes, 1)
		assert.Equal(t, "Base", supertypes[0].TypeName)
	})

	t.Run("should handle generic embedding", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Box": {
							Name:              "Box",
							DefinitionLine:    1,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/a.go",
						},
						"Container": {
							Name:              "Container",
							DefinitionLine:    5,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/a.go",
							Fields: []*inspector_dto.Field{
								{
									Name:        "",
									TypeString:  "Box[string]",
									IsEmbedded:  true,
									PackagePath: "pkg/a",
								},
							},
						},
					},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)

		supertypes := index.GetSupertypes("pkg/a", "Container")
		require.Len(t, supertypes, 1)
		assert.Equal(t, "Box", supertypes[0].TypeName)
	})

	t.Run("should handle multiple embeddings in one type", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Alpha": {
							Name:              "Alpha",
							DefinitionLine:    1,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/a.go",
						},
						"Beta": {
							Name:              "Beta",
							DefinitionLine:    5,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/a.go",
						},
						"Combo": {
							Name:              "Combo",
							DefinitionLine:    10,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/a.go",
							Fields: []*inspector_dto.Field{
								{
									Name:        "",
									TypeString:  "Alpha",
									IsEmbedded:  true,
									PackagePath: "pkg/a",
								},
								{
									Name:        "",
									TypeString:  "Beta",
									IsEmbedded:  true,
									PackagePath: "pkg/a",
								},
							},
						},
					},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)

		supertypes := index.GetSupertypes("pkg/a", "Combo")
		require.Len(t, supertypes, 2)

		names := make(map[string]bool)
		for _, st := range supertypes {
			names[st.TypeName] = true
		}
		assert.True(t, names["Alpha"])
		assert.True(t, names["Beta"])

		alphaSubtypes := index.GetSubtypes("pkg/a", "Alpha")
		require.Len(t, alphaSubtypes, 1)
		assert.Equal(t, "Combo", alphaSubtypes[0].TypeName)
	})

	t.Run("should return empty for type with no embeddings", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Plain": {
							Name: "Plain",
							Fields: []*inspector_dto.Field{
								{Name: "Name", TypeString: "string", IsEmbedded: false},
							},
						},
					},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)
		assert.Empty(t, index.GetSupertypes("pkg/a", "Plain"))
	})

	t.Run("should return empty for non-existent type", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name:       "a",
					Path:       "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)
		assert.Empty(t, index.GetSupertypes("pkg/a", "Missing"))
		assert.Empty(t, index.GetSubtypes("pkg/a", "Missing"))
	})

	t.Run("should handle nil TypeData", func(t *testing.T) {
		t.Parallel()
		index := NewTypeHierarchyIndex(nil)
		assert.Empty(t, index.GetSupertypes("pkg/a", "Anything"))
		assert.Empty(t, index.GetSubtypes("pkg/a", "Anything"))
	})

	t.Run("should handle TypeData with nil Packages", func(t *testing.T) {
		t.Parallel()
		index := NewTypeHierarchyIndex(&inspector_dto.TypeData{Packages: nil})
		assert.Empty(t, index.GetSupertypes("pkg/a", "Anything"))
	})

	t.Run("should handle nil package in Packages map", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/nil": nil,
			},
		}
		index := NewTypeHierarchyIndex(typeData)
		assert.NotNil(t, index)
	})

	t.Run("should handle package with nil NamedTypes", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {Name: "a", Path: "pkg/a", NamedTypes: nil},
			},
		}
		index := NewTypeHierarchyIndex(typeData)
		assert.NotNil(t, index)
	})

	t.Run("should handle nil type info in NamedTypes", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"NilType": nil,
					},
				},
			},
		}
		index := NewTypeHierarchyIndex(typeData)
		assert.NotNil(t, index)
	})

	t.Run("should handle nil and non-embedded fields", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"WithFields": {
							Name: "WithFields",
							Fields: []*inspector_dto.Field{
								nil,
								{Name: "Name", TypeString: "string", IsEmbedded: false},
							},
						},
					},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)
		assert.Empty(t, index.GetSupertypes("pkg/a", "WithFields"))
	})

	t.Run("should handle embedded type not found in TypeData", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Orphan": {
							Name:              "Orphan",
							DefinitionLine:    1,
							DefinitionColumn:  6,
							DefinedInFilePath: "/src/orphan.go",
							Fields: []*inspector_dto.Field{
								{
									Name:        "",
									TypeString:  "MissingBase",
									IsEmbedded:  true,
									PackagePath: "pkg/a",
								},
							},
						},
					},
				},
			},
		}

		index := NewTypeHierarchyIndex(typeData)

		supertypes := index.GetSupertypes("pkg/a", "Orphan")
		require.Len(t, supertypes, 1)
		assert.Equal(t, "MissingBase", supertypes[0].TypeName)
		assert.Equal(t, 0, supertypes[0].Line)
		assert.Equal(t, "", supertypes[0].FilePath)
	})
}
