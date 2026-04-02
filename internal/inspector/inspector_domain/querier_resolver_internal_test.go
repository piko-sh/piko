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

func TestResolvePackageAlias(t *testing.T) {
	t.Parallel()
	td := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"my/pkg": {
				Name: "pkg",
				Path: "my/pkg",
				FileImports: map[string]map[string]string{
					"/src/main.go": {
						"fmt":    "fmt",
						"models": "my/models",
					},
				},
			},
			"my/models": {
				Name: "models",
				Path: "my/models",
			},
			"fmt": {
				Name: "fmt",
				Path: "fmt",
			},
		},
		FileToPackage: map[string]string{
			"/src/main.go": "my/pkg",
		},
	}
	querier := &TypeQuerier{
		typeData:           td,
		namedTypeCache:     sync.Map{},
		underlyingASTCache: sync.Map{},
	}

	tests := []struct {
		name                string
		alias               string
		importerPackagePath string
		importerFilePath    string
		expected            string
	}{
		{
			name:                "resolve known alias from file imports",
			alias:               "models",
			importerPackagePath: "my/pkg",
			importerFilePath:    "/src/main.go",
			expected:            "my/models",
		},
		{
			name:                "resolve stdlib alias",
			alias:               "fmt",
			importerPackagePath: "my/pkg",
			importerFilePath:    "/src/main.go",
			expected:            "fmt",
		},
		{
			name:                "unknown alias returns empty",
			alias:               "unknown",
			importerPackagePath: "my/pkg",
			importerFilePath:    "/src/main.go",
			expected:            "",
		},
		{
			name:                "unknown package path returns empty",
			alias:               "models",
			importerPackagePath: "nonexistent/pkg",
			importerFilePath:    "/src/main.go",
			expected:            "",
		},
		{
			name:                "empty file path falls back to resolveWithFallbacks",
			alias:               "models",
			importerPackagePath: "my/pkg",
			importerFilePath:    "",
			expected:            "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := querier.ResolvePackageAlias(tc.alias, tc.importerPackagePath, tc.importerFilePath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindCorrectAliasForPath(t *testing.T) {
	t.Parallel()
	pkg := &inspector_dto.Package{
		Name: "main",
		Path: "main",
		FileImports: map[string]map[string]string{
			"/src/main.go": {
				"models": "my/models",
				".":      "my/dotpkg",
				"_":      "my/sideeffect",
				"http":   "net/http",
			},
		},
	}

	tests := []struct {
		name          string
		filePath      string
		canonicalPath string
		expected      string
	}{
		{
			name:          "finds alias for known path",
			filePath:      "/src/main.go",
			canonicalPath: "my/models",
			expected:      "models",
		},
		{
			name:          "skips dot import",
			filePath:      "/src/main.go",
			canonicalPath: "my/dotpkg",
			expected:      "",
		},
		{
			name:          "skips blank import",
			filePath:      "/src/main.go",
			canonicalPath: "my/sideeffect",
			expected:      "",
		},
		{
			name:          "returns empty for unknown path",
			filePath:      "/src/main.go",
			canonicalPath: "unknown/pkg",
			expected:      "",
		},
		{
			name:          "returns empty for unknown file",
			filePath:      "/other/file.go",
			canonicalPath: "my/models",
			expected:      "",
		},
		{
			name:          "finds stdlib alias",
			filePath:      "/src/main.go",
			canonicalPath: "net/http",
			expected:      "http",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := findCorrectAliasForPath(pkg, tc.filePath, tc.canonicalPath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHandleUnqualifiedIdentifier(t *testing.T) {
	t.Parallel()
	t.Run("non-ident expr returns unchanged", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"main": {
					Name: "main",
					Path: "main",
				},
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		selectorExpr := &goast.SelectorExpr{
			X:   goast.NewIdent("models"),
			Sel: goast.NewIdent("User"),
		}
		pkg := td.Packages["main"]

		finalAST, finalPath, finalAlias := querier.handleUnqualifiedIdentifier(
			selectorExpr, pkg, "/src/main.go", "my/models", "models",
		)

		assert.Equal(t, "models.User", goastutil.ASTToTypeString(finalAST))
		assert.Equal(t, "my/models", finalPath)
		assert.Equal(t, "models", finalAlias)
	})

	t.Run("ident for a primitive type returns unchanged", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"main": {
					Name: "main",
					Path: "main",
				},
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		identExpr := goast.NewIdent("string")
		pkg := td.Packages["main"]

		finalAST, finalPath, finalAlias := querier.handleUnqualifiedIdentifier(
			identExpr, pkg, "/src/main.go", "main", "main",
		)

		assert.Equal(t, "string", goastutil.ASTToTypeString(finalAST))
		assert.Equal(t, "main", finalPath)
		assert.Equal(t, "main", finalAlias)
	})

	t.Run("ident from a dot-imported package creates SelectorExpr", func(t *testing.T) {
		t.Parallel()
		pkg := &inspector_dto.Package{
			Name: "main",
			Path: "main",
			FileImports: map[string]map[string]string{
				"/src/main.go": {
					".": "my/models",
				},
			},
		}
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"main": pkg,
				"my/models": {
					Name: "models",
					Path: "my/models",
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {Name: "User"},
					},
				},
			},
			FileToPackage: map[string]string{
				"/src/main.go": "main",
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		identExpr := goast.NewIdent("User")

		finalAST, finalPath, finalAlias := querier.handleUnqualifiedIdentifier(
			identExpr, pkg, "/src/main.go", "main", "main",
		)

		assert.Equal(t, "models.User", goastutil.ASTToTypeString(finalAST))
		assert.Equal(t, "my/models", finalPath)
		assert.Equal(t, "models", finalAlias)
	})

	t.Run("ident where canonicalPath differs and alias found creates SelectorExpr", func(t *testing.T) {
		t.Parallel()
		pkg := &inspector_dto.Package{
			Name: "main",
			Path: "main",
			FileImports: map[string]map[string]string{
				"/src/main.go": {
					"models": "my/models",
				},
			},
		}
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"main": pkg,
				"my/models": {
					Name: "models",
					Path: "my/models",
				},
			},
			FileToPackage: map[string]string{
				"/src/main.go": "main",
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		identExpr := goast.NewIdent("User")

		finalAST, finalPath, finalAlias := querier.handleUnqualifiedIdentifier(
			identExpr, pkg, "/src/main.go", "my/models", "models",
		)

		assert.Equal(t, "models.User", goastutil.ASTToTypeString(finalAST))
		assert.Equal(t, "my/models", finalPath)
		assert.Equal(t, "models", finalAlias)
	})

	t.Run("ident where no alias found returns unchanged", func(t *testing.T) {
		t.Parallel()
		pkg := &inspector_dto.Package{
			Name: "main",
			Path: "main",
			FileImports: map[string]map[string]string{
				"/src/main.go": {
					"http": "net/http",
				},
			},
		}
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"main": pkg,
			},
			FileToPackage: map[string]string{
				"/src/main.go": "main",
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		identExpr := goast.NewIdent("User")

		finalAST, finalPath, finalAlias := querier.handleUnqualifiedIdentifier(
			identExpr, pkg, "/src/main.go", "unknown/pkg", "unknown",
		)

		assert.Equal(t, "User", goastutil.ASTToTypeString(finalAST))
		assert.Equal(t, "unknown/pkg", finalPath)
		assert.Equal(t, "unknown", finalAlias)
	})
}

func TestFindContextByResolvingType(t *testing.T) {
	t.Parallel()
	t.Run("happy path resolves type successfully", func(t *testing.T) {
		t.Parallel()
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
						},
					},
				},
			},
			FileToPackage: map[string]string{
				"/src/user.go": "my/pkg",
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		info := &inspector_dto.FieldInfo{
			Type:                goastutil.TypeStringToAST("User"),
			DefiningPackagePath: "my/pkg",
			DefiningFilePath:    "/src/user.go",
		}

		pkg, file, ok := querier.findContextByResolvingType(info)

		require.True(t, ok)
		assert.Equal(t, "my/pkg", pkg)
		assert.Equal(t, "/src/user.go", file)
	})

	t.Run("namedType is nil returns false", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
				},
			},
			FileToPackage: map[string]string{},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		info := &inspector_dto.FieldInfo{
			Type:                goastutil.TypeStringToAST("NonExistent"),
			DefiningPackagePath: "my/pkg",
			DefiningFilePath:    "/src/main.go",
		}

		pkg, file, ok := querier.findContextByResolvingType(info)

		assert.False(t, ok)
		assert.Empty(t, pkg)
		assert.Empty(t, file)
	})

	t.Run("namedType has empty DefinedInFilePath returns false", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {
							Name:              "User",
							PackagePath:       "my/pkg",
							DefinedInFilePath: "",
						},
					},
				},
			},
			FileToPackage: map[string]string{},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		info := &inspector_dto.FieldInfo{
			Type:                goastutil.TypeStringToAST("User"),
			DefiningPackagePath: "my/pkg",
			DefiningFilePath:    "/src/main.go",
		}

		pkg, file, ok := querier.findContextByResolvingType(info)

		assert.False(t, ok)
		assert.Empty(t, pkg)
		assert.Empty(t, file)
	})

	t.Run("FindPackagePathForTypeDTO returns empty falls back to lookupPackagePathForFile", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
					NamedTypes: map[string]*inspector_dto.Type{
						"Widget": {
							Name:              "Widget",
							PackagePath:       "",
							DefinedInFilePath: "/src/widget.go",
						},
					},
				},
			},
			FileToPackage: map[string]string{
				"/src/widget.go": "my/pkg",
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		info := &inspector_dto.FieldInfo{
			Type:                goastutil.TypeStringToAST("Widget"),
			DefiningPackagePath: "my/pkg",
			DefiningFilePath:    "/src/main.go",
		}

		pkg, file, ok := querier.findContextByResolvingType(info)

		require.True(t, ok)
		assert.Equal(t, "my/pkg", pkg)
		assert.Equal(t, "/src/widget.go", file)
	})

	t.Run("both FindPackagePathForTypeDTO and lookupPackagePathForFile return empty", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my/pkg": {
					Name: "pkg",
					Path: "my/pkg",
					NamedTypes: map[string]*inspector_dto.Type{
						"Orphan": {
							Name:              "Orphan",
							PackagePath:       "",
							DefinedInFilePath: "/unknown/orphan.go",
						},
					},
				},
			},
			FileToPackage: map[string]string{},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		info := &inspector_dto.FieldInfo{
			Type:                goastutil.TypeStringToAST("Orphan"),
			DefiningPackagePath: "my/pkg",
			DefiningFilePath:    "/src/main.go",
		}

		pkg, file, ok := querier.findContextByResolvingType(info)

		assert.False(t, ok)
		assert.Empty(t, pkg)
		assert.Empty(t, file)
	})
}

func TestResolveWithFallbacksCommandLineArguments(t *testing.T) {
	t.Parallel()
	t.Run("absolute path matching importerPackagePath resolves to command-line-arguments", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"command-line-arguments": {
					Name: "main",
					Path: "command-line-arguments",
				},
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		result, ok := querier.resolveWithFallbacks("/absolute/path", nil, "/absolute/path")

		require.True(t, ok)
		assert.Equal(t, "command-line-arguments", result)
	})

	t.Run("absolute path not matching importerPackagePath does not resolve", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"command-line-arguments": {
					Name: "main",
					Path: "command-line-arguments",
				},
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		result, ok := querier.resolveWithFallbacks("/absolute/path", nil, "/different/path")

		assert.False(t, ok)
		assert.Empty(t, result)
	})

	t.Run("relative path does not trigger command-line-arguments fallback", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"command-line-arguments": {
					Name: "main",
					Path: "command-line-arguments",
				},
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		result, ok := querier.resolveWithFallbacks("relative/path", nil, "relative/path")

		assert.False(t, ok)
		assert.Empty(t, result)
	})

	t.Run("alias matches importer package name with nil pkg returns importerPackagePath", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		result, ok := querier.resolveWithFallbacks("unknown", nil, "some/pkg")

		assert.False(t, ok)
		assert.Empty(t, result)
	})

	t.Run("alias matches importer package name returns package path", func(t *testing.T) {
		t.Parallel()
		importerPackage := &inspector_dto.Package{
			Name: "mypkg",
			Path: "my/mypkg",
		}
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		result, ok := querier.resolveWithFallbacks("mypkg", importerPackage, "my/mypkg")

		require.True(t, ok)
		assert.Equal(t, "my/mypkg", result)
	})

	t.Run("alias matches importer package name with empty path returns importerPackagePath", func(t *testing.T) {
		t.Parallel()
		importerPackage := &inspector_dto.Package{
			Name: "mypkg",
			Path: "",
		}
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		result, ok := querier.resolveWithFallbacks("mypkg", importerPackage, "fallback/path")

		require.True(t, ok)
		assert.Equal(t, "fallback/path", result)
	})

	t.Run("alias is a known package path resolves directly", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"known/pkg": {
					Name: "pkg",
					Path: "known/pkg",
				},
			},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		result, ok := querier.resolveWithFallbacks("known/pkg", nil, "some/importer")

		require.True(t, ok)
		assert.Equal(t, "known/pkg", result)
	})

	t.Run("absolute path with no command-line-arguments package returns false", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		}
		querier := &TypeQuerier{
			typeData:           td,
			namedTypeCache:     sync.Map{},
			underlyingASTCache: sync.Map{},
		}

		result, ok := querier.resolveWithFallbacks("/absolute/path", nil, "/absolute/path")

		assert.False(t, ok)
		assert.Empty(t, result)
	})
}
