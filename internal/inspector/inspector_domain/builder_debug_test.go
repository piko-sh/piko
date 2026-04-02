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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestSanitisePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		path           string
		prefix         string
		modCachePrefix string
		expected       string
	}{
		{
			name:     "empty path",
			path:     "",
			prefix:   "/home/user",
			expected: "",
		},
		{
			name:     "path with matching prefix",
			path:     "/home/user/project/src/main.go",
			prefix:   "/home/user/project",
			expected: "src/main.go",
		},
		{
			name:           "path with matching mod cache prefix",
			path:           "/home/user/gomodcache/pkg/mod/github.com/foo/bar@v1.0.0/baz.go",
			prefix:         "/home/user/project",
			modCachePrefix: "/home/user/gomodcache/pkg/mod",
			expected:       "$GOMODCACHE/github.com/foo/bar@v1.0.0/baz.go",
		},
		{
			name:           "path with no matching prefix",
			path:           "/other/path/file.go",
			prefix:         "/home/user",
			modCachePrefix: "/modcache",
			expected:       "/other/path/file.go",
		},
		{
			name:           "empty mod cache prefix",
			path:           "/home/user/project/file.go",
			prefix:         "/home/user/project",
			modCachePrefix: "",
			expected:       "file.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := sanitisePath(tc.path, tc.prefix, tc.modCachePrefix, "")
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFilterDTO(t *testing.T) {
	t.Parallel()
	t.Run("should return original when no prefixes", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"a/b": {Name: "b", Path: "a/b"},
			},
		}
		result := filterDTO(td, nil)
		assert.Equal(t, td, result)
	})

	t.Run("should filter by prefix", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
					FileImports: map[string]map[string]string{
						"/src/main.go": {"fmt": "fmt"},
					},
				},
				"other/pkg": {
					Name: "other",
					Path: "other/pkg",
				},
			},
			FileToPackage: map[string]string{},
		}
		result := filterDTO(td, []string{"my/"})
		require.Len(t, result.Packages, 1)
		_, ok := result.Packages["my/pkg"]
		assert.True(t, ok)
		_, ok = result.FileToPackage["/src/main.go"]
		assert.True(t, ok)
	})

	t.Run("should exclude non-matching packages", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"a/b": {Name: "b", Path: "a/b"},
				"c/d": {Name: "d", Path: "c/d"},
			},
			FileToPackage: map[string]string{},
		}
		result := filterDTO(td, []string{"x/"})
		assert.Empty(t, result.Packages)
	})
}

func TestSanitiseDTO(t *testing.T) {
	t.Parallel()
	t.Run("should sanitise all file paths in TypeData", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
					FileImports: map[string]map[string]string{
						"/home/user/project/src/main.go": {"fmt": "fmt"},
					},
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {
							Name:              "User",
							DefinedInFilePath: "/home/user/project/src/user.go",
							Fields: []*inspector_dto.Field{
								{
									Name:               "Name",
									DefinitionFilePath: "/home/user/project/src/user.go",
								},
							},
							Methods: []*inspector_dto.Method{
								{
									Name:               "GetName",
									DefinitionFilePath: "/home/user/project/src/user.go",
								},
							},
						},
					},
					Funcs: map[string]*inspector_dto.Function{
						"NewUser": {
							Name:               "NewUser",
							DefinitionFilePath: "/home/user/project/src/user.go",
						},
					},
				},
			},
			FileToPackage: map[string]string{
				"/home/user/project/src/main.go": "my/pkg",
			},
		}

		sanitiseDTO(td, "/home/user/project", "", "")

		_, ok := td.FileToPackage["src/main.go"]
		assert.True(t, ok, "FileToPackage paths should be sanitised")

		pkg := td.Packages["my/pkg"]
		_, ok = pkg.FileImports["src/main.go"]
		assert.True(t, ok, "FileImports paths should be sanitised")

		userType := pkg.NamedTypes["User"]
		assert.Equal(t, "src/user.go", userType.DefinedInFilePath)

		assert.Equal(t, "src/user.go", userType.Fields[0].DefinitionFilePath)

		assert.Equal(t, "src/user.go", userType.Methods[0].DefinitionFilePath)

		assert.Equal(t, "src/user.go", pkg.Funcs["NewUser"].DefinitionFilePath)
	})
}

func TestDumpReadable(t *testing.T) {
	t.Parallel()
	t.Run("should produce readable output", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {
							Name:       "User",
							TypeString: "User",
							Fields: []*inspector_dto.Field{
								{Name: "Name", TypeString: "string"},
							},
							Methods: []*inspector_dto.Method{
								{
									Name:                 "GetName",
									DeclaringPackagePath: "my/pkg",
									DeclaringTypeName:    "User",
									Signature:            inspector_dto.FunctionSignature{Results: []string{"string"}},
								},
							},
						},
					},
					Funcs: map[string]*inspector_dto.Function{
						"NewUser": {
							Name:      "NewUser",
							Signature: inspector_dto.FunctionSignature{Results: []string{"*User"}},
						},
					},
				},
			},
		}

		output := dumpReadable(td)
		assert.Contains(t, output, "my/pkg")
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "GetName")
		assert.Contains(t, output, "NewUser")
	})

	t.Run("should handle empty TypeData", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		}
		output := dumpReadable(td)
		assert.Contains(t, output, "EMPTY")
	})
}

func TestFormatMethodsReadable(t *testing.T) {
	t.Parallel()
	t.Run("should return nil for empty methods", func(t *testing.T) {
		t.Parallel()
		result := formatMethodsReadable(nil, "User", "my/pkg")
		assert.Nil(t, result)
	})

	t.Run("should format methods with receiver types", func(t *testing.T) {
		t.Parallel()
		methods := []*inspector_dto.Method{
			{
				Name:                 "GetName",
				IsPointerReceiver:    false,
				DeclaringPackagePath: "my/pkg",
				DeclaringTypeName:    "User",
				Signature:            inspector_dto.FunctionSignature{Results: []string{"string"}},
			},
			{
				Name:                 "SetName",
				IsPointerReceiver:    true,
				DeclaringPackagePath: "my/pkg",
				DeclaringTypeName:    "User",
				Signature:            inspector_dto.FunctionSignature{Params: []string{"string"}},
			},
		}
		result := formatMethodsReadable(methods, "User", "my/pkg")
		require.NotNil(t, result)
		output := strings.Join(result, "\n")
		assert.Contains(t, output, "(T)")
		assert.Contains(t, output, "(*T)")
		assert.Contains(t, output, "GetName")
		assert.Contains(t, output, "SetName")
	})

	t.Run("should mark promoted methods", func(t *testing.T) {
		t.Parallel()
		methods := []*inspector_dto.Method{
			{
				Name:                 "BaseMethod",
				DeclaringPackagePath: "other/pkg",
				DeclaringTypeName:    "Base",
				Signature:            inspector_dto.FunctionSignature{},
			},
		}
		result := formatMethodsReadable(methods, "User", "my/pkg")
		require.NotNil(t, result)
		output := strings.Join(result, "\n")
		assert.Contains(t, output, "Promoted from")
		assert.Contains(t, output, "other/pkg.Base")
	})
}

func TestFormatFieldsReadable(t *testing.T) {
	t.Parallel()
	t.Run("should format fields with composite types", func(t *testing.T) {
		t.Parallel()
		fields := []*inspector_dto.Field{
			{
				Name:          "Data",
				TypeString:    "map[string]int",
				CompositeType: inspector_dto.CompositeTypeMap,
				CompositeParts: []*inspector_dto.CompositePart{
					{Role: "key", TypeString: "string"},
					{Role: "value", TypeString: "int"},
				},
			},
			{
				Name:       "Name",
				TypeString: "string",
			},
		}
		result := formatFieldsReadable(fields)
		require.NotNil(t, result)
		output := strings.Join(result, "\n")
		assert.Contains(t, output, "Data")
		assert.Contains(t, output, "Name")
	})

	t.Run("should return nil for empty fields", func(t *testing.T) {
		t.Parallel()
		result := formatFieldsReadable(nil)
		assert.Nil(t, result)
	})
}
