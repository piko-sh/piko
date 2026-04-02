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
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestDeconstructTypeExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		expression       goast.Expr
		wantTypeName     string
		wantPackageAlias string
		wantOk           bool
	}{
		{
			name:             "simple identifier",
			expression:       &goast.Ident{Name: "User"},
			wantTypeName:     "User",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name:             "pointer type",
			expression:       &goast.StarExpr{X: &goast.Ident{Name: "User"}},
			wantTypeName:     "User",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name:             "slice type",
			expression:       &goast.ArrayType{Elt: &goast.Ident{Name: "User"}},
			wantTypeName:     "User",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name: "array type",
			expression: &goast.ArrayType{
				Len: &goast.BasicLit{Kind: token.INT, Value: "5"},
				Elt: &goast.Ident{Name: "User"},
			},
			wantTypeName:     "User",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name:             "channel type",
			expression:       &goast.ChanType{Value: &goast.Ident{Name: "Event"}},
			wantTypeName:     "Event",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name: "map type",
			expression: &goast.MapType{
				Key:   &goast.Ident{Name: "string"},
				Value: &goast.Ident{Name: "int"},
			},
			wantTypeName:     "map",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name: "selector expression (qualified type)",
			expression: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "http"},
				Sel: &goast.Ident{Name: "Request"},
			},
			wantTypeName:     "Request",
			wantPackageAlias: "http",
			wantOk:           true,
		},
		{
			name: "generic index expression",
			expression: &goast.IndexExpr{
				X: &goast.Ident{Name: "Box"},
			},
			wantTypeName:     "Box",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name: "generic multi-param index expression",
			expression: &goast.IndexListExpr{
				X: &goast.Ident{Name: "Pair"},
			},
			wantTypeName:     "Pair",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name:             "parenthesised expression",
			expression:       &goast.ParenExpr{X: &goast.Ident{Name: "MyType"}},
			wantTypeName:     "MyType",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name:             "function type",
			expression:       &goast.FuncType{},
			wantTypeName:     "function",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name:             "interface type",
			expression:       &goast.InterfaceType{},
			wantTypeName:     "interface{}",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name:             "struct type",
			expression:       &goast.StructType{},
			wantTypeName:     "struct",
			wantPackageAlias: "",
			wantOk:           true,
		},
		{
			name: "nested pointer to slice of qualified type",
			expression: &goast.StarExpr{
				X: &goast.ArrayType{
					Elt: &goast.SelectorExpr{
						X:   &goast.Ident{Name: "models"},
						Sel: &goast.Ident{Name: "User"},
					},
				},
			},
			wantTypeName:     "User",
			wantPackageAlias: "models",
			wantOk:           true,
		},
		{
			name: "selector with non-ident X",
			expression: &goast.SelectorExpr{
				X:   &goast.StarExpr{X: &goast.Ident{Name: "pkg"}},
				Sel: &goast.Ident{Name: "Type"},
			},
			wantTypeName:     "",
			wantPackageAlias: "",
			wantOk:           false,
		},
		{
			name:             "unsupported expression (Ellipsis)",
			expression:       &goast.Ellipsis{Elt: &goast.Ident{Name: "int"}},
			wantTypeName:     "",
			wantPackageAlias: "",
			wantOk:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			typeName, pkgAlias, ok := DeconstructTypeExpr(tc.expression)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.wantTypeName, typeName)
			assert.Equal(t, tc.wantPackageAlias, pkgAlias)
		})
	}
}

func TestToExportedName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "already capitalised", input: "Name", want: "Name"},
		{name: "lowercase", input: "name", want: "Name"},
		{name: "single char lowercase", input: "a", want: "A"},
		{name: "single char uppercase", input: "Z", want: "Z"},
		{name: "unicode lowercase", input: "\u00e9cole", want: "\u00c9cole"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, ToExportedName(tc.input))
		})
	}
}

func TestGetAllPackages(t *testing.T) {
	t.Parallel()

	t.Run("should return empty map for nil typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: nil}
		pkgs := querier.GetAllPackages()
		assert.NotNil(t, pkgs)
		assert.Empty(t, pkgs)
	})

	t.Run("should return empty map for nil Packages", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{typeData: &inspector_dto.TypeData{Packages: nil}}
		pkgs := querier.GetAllPackages()
		assert.NotNil(t, pkgs)
		assert.Empty(t, pkgs)
	})

	t.Run("should return packages from typeData", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {Name: "a", Path: "pkg/a"},
					"pkg/b": {Name: "b", Path: "pkg/b"},
				},
			},
		}
		pkgs := querier.GetAllPackages()
		assert.Len(t, pkgs, 2)
		assert.Contains(t, pkgs, "pkg/a")
		assert.Contains(t, pkgs, "pkg/b")
	})
}

func TestGetImportsForFile(t *testing.T) {
	t.Parallel()

	t.Run("should return empty map for nil typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: nil}
		imports := querier.GetImportsForFile("pkg/a", "/src/a.go")
		assert.NotNil(t, imports)
		assert.Empty(t, imports)
	})

	t.Run("should return empty map for empty importerFilePath", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {Name: "a", Path: "pkg/a"},
				},
			},
		}
		imports := querier.GetImportsForFile("pkg/a", "")
		assert.NotNil(t, imports)
		assert.Empty(t, imports)
	})

	t.Run("should return empty map for missing package", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		imports := querier.GetImportsForFile("pkg/missing", "/src/a.go")
		assert.Empty(t, imports)
	})

	t.Run("should return empty map for nil FileImports", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {Name: "a", Path: "pkg/a", FileImports: nil},
				},
			},
		}
		imports := querier.GetImportsForFile("pkg/a", "/src/a.go")
		assert.Empty(t, imports)
	})

	t.Run("should return empty map for missing file", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						FileImports: map[string]map[string]string{
							"/src/other.go": {"fmt": "fmt"},
						},
					},
				},
			},
		}
		imports := querier.GetImportsForFile("pkg/a", "/src/a.go")
		assert.Empty(t, imports)
	})

	t.Run("should return imports with identity and package name mappings", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my/pkg": {
						Name: "pkg",
						Path: "my/pkg",
						FileImports: map[string]map[string]string{
							"/src/a.go": {
								"fmt":  "fmt",
								"http": "net/http",
							},
						},
					},
				},
			},
		}
		imports := querier.GetImportsForFile("my/pkg", "/src/a.go")

		assert.Equal(t, "fmt", imports["fmt"])
		assert.Equal(t, "net/http", imports["http"])
		assert.Equal(t, "net/http", imports["net/http"])
		assert.Equal(t, "my/pkg", imports["my/pkg"])
		assert.Equal(t, "my/pkg", imports["pkg"])
	})
}

func TestFindFileWithImportAlias(t *testing.T) {
	t.Parallel()

	t.Run("should return empty for nil typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: nil}
		assert.Equal(t, "", querier.FindFileWithImportAlias("pkg/a", "fmt", "fmt"))
	})

	t.Run("should return empty for missing package", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		assert.Equal(t, "", querier.FindFileWithImportAlias("pkg/missing", "fmt", "fmt"))
	})

	t.Run("should return empty for nil FileImports", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {Name: "a", Path: "pkg/a", FileImports: nil},
				},
			},
		}
		assert.Equal(t, "", querier.FindFileWithImportAlias("pkg/a", "fmt", "fmt"))
	})

	t.Run("should find file with matching alias and path", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						FileImports: map[string]map[string]string{
							"/src/a.go": {"h": "net/http"},
							"/src/b.go": {"fmt": "fmt"},
						},
					},
				},
			},
		}
		result := querier.FindFileWithImportAlias("pkg/a", "h", "net/http")
		assert.Equal(t, "/src/a.go", result)
	})

	t.Run("should return empty when alias maps to wrong path", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						FileImports: map[string]map[string]string{
							"/src/a.go": {"h": "net/http"},
						},
					},
				},
			},
		}
		result := querier.FindFileWithImportAlias("pkg/a", "h", "fmt")
		assert.Equal(t, "", result)
	})
}

func TestFindPackagePathForTypeDTO(t *testing.T) {
	t.Parallel()

	querier := &TypeQuerier{}

	t.Run("should return empty for nil target", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", querier.FindPackagePathForTypeDTO(nil))
	})

	t.Run("should return PackagePath from type", func(t *testing.T) {
		t.Parallel()

		typ := &inspector_dto.Type{PackagePath: "my/pkg/path"}
		assert.Equal(t, "my/pkg/path", querier.FindPackagePathForTypeDTO(typ))
	})

	t.Run("should return empty when PackagePath is empty", func(t *testing.T) {
		t.Parallel()

		typ := &inspector_dto.Type{PackagePath: ""}
		assert.Equal(t, "", querier.FindPackagePathForTypeDTO(typ))
	})
}

func TestGetFilesForPackage(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for nil querier state", func(t *testing.T) {
		t.Parallel()
		var querier *TypeQuerier
		assert.Nil(t, querier.GetFilesForPackage("pkg/a"))
	})

	t.Run("should return nil for nil typeData", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{typeData: nil}
		assert.Nil(t, querier.GetFilesForPackage("pkg/a"))
	})

	t.Run("should return nil for missing package", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		assert.Nil(t, querier.GetFilesForPackage("pkg/missing"))
	})

	t.Run("should return nil for nil FileImports", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {Name: "a", Path: "pkg/a", FileImports: nil},
				},
			},
		}
		assert.Nil(t, querier.GetFilesForPackage("pkg/a"))
	})

	t.Run("should return file paths from FileImports keys", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						FileImports: map[string]map[string]string{
							"/src/a.go": {},
							"/src/b.go": {},
						},
					},
				},
			},
		}
		files := querier.GetFilesForPackage("pkg/a")
		sort.Strings(files)
		assert.Equal(t, []string{"/src/a.go", "/src/b.go"}, files)
	})
}

func TestPackagePathForFile(t *testing.T) {
	t.Parallel()

	t.Run("should return empty for empty file path", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{}
		assert.Equal(t, "", querier.PackagePathForFile(""))
	})

	t.Run("should return from FileToPackage map", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				FileToPackage: map[string]string{
					"/src/a.go": "my/pkg",
				},
			},
		}
		assert.Equal(t, "my/pkg", querier.PackagePathForFile("/src/a.go"))
	})

	t.Run("should derive from BaseDir and ModuleName", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				FileToPackage: map[string]string{},
			},
			Config: inspector_dto.Config{
				BaseDir:    "/project",
				ModuleName: "my/module",
			},
		}
		assert.Equal(t, "my/module/sub/pkg", querier.PackagePathForFile("/project/sub/pkg/a.go"))
	})

	t.Run("should return module name for files in base directory", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				FileToPackage: map[string]string{},
			},
			Config: inspector_dto.Config{
				BaseDir:    "/project",
				ModuleName: "my/module",
			},
		}
		assert.Equal(t, "my/module", querier.PackagePathForFile("/project/a.go"))
	})

	t.Run("should return empty when no BaseDir or ModuleName", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				FileToPackage: map[string]string{},
			},
			Config: inspector_dto.Config{},
		}
		assert.Equal(t, "", querier.PackagePathForFile("/some/file.go"))
	})
}

func TestGetNamedTypeByPackageAndName(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for nil querier", func(t *testing.T) {
		t.Parallel()
		var querier *TypeQuerier
		assert.Nil(t, querier.getNamedTypeByPackageAndName("pkg/a", "User"))
	})

	t.Run("should return nil for nil typeData", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{typeData: nil}
		assert.Nil(t, querier.getNamedTypeByPackageAndName("pkg/a", "User"))
	})

	t.Run("should return nil for empty packagePath", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		assert.Nil(t, querier.getNamedTypeByPackageAndName("", "User"))
	})

	t.Run("should return nil for empty typeName", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		assert.Nil(t, querier.getNamedTypeByPackageAndName("pkg/a", ""))
	})

	t.Run("should return nil for missing package", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		assert.Nil(t, querier.getNamedTypeByPackageAndName("pkg/missing", "User"))
	})

	t.Run("should return nil for missing type", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name:       "a",
						Path:       "pkg/a",
						NamedTypes: map[string]*inspector_dto.Type{},
					},
				},
			},
		}
		assert.Nil(t, querier.getNamedTypeByPackageAndName("pkg/a", "Missing"))
	})

	t.Run("should return type when found", func(t *testing.T) {
		t.Parallel()

		expected := &inspector_dto.Type{Name: "User"}
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": expected,
						},
					},
				},
			},
		}
		assert.Equal(t, expected, querier.getNamedTypeByPackageAndName("pkg/a", "User"))
	})
}

func TestDebug(t *testing.T) {
	t.Parallel()

	t.Run("should return debug output with empty typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: &inspector_dto.TypeData{}}
		lines := querier.Debug("", "")
		output := strings.Join(lines, "\n")
		assert.Contains(t, output, "TYPE INSPECTOR STATE")
		assert.Contains(t, output, "GLOBAL CONTEXT")
		assert.Contains(t, output, "(None)")
	})

	t.Run("should return debug output with packages", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {Name: "a", Path: "pkg/a"},
				},
			},
		}
		lines := querier.Debug("", "")
		output := strings.Join(lines, "\n")
		assert.Contains(t, output, "pkg/a")
	})

	t.Run("should include focused context when importerPackagePath is set", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name: "a",
						Path: "pkg/a",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {Name: "User"},
						},
						Funcs: map[string]*inspector_dto.Function{
							"NewUser": {Name: "NewUser"},
						},
						FileImports: map[string]map[string]string{
							"/src/a.go": {"fmt": "fmt"},
						},
					},
				},
			},
		}
		lines := querier.Debug("pkg/a", "/src/a.go")
		output := strings.Join(lines, "\n")
		assert.Contains(t, output, "FOCUSED CONTEXT FOR PACKAGE: pkg/a")
		assert.Contains(t, output, "FOCUSED CONTEXT FOR FILE")
		assert.Contains(t, output, "type User")
		assert.Contains(t, output, "func NewUser")
	})

	t.Run("should handle focused context for missing package", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		lines := querier.Debug("pkg/missing", "")
		output := strings.Join(lines, "\n")
		assert.Contains(t, output, "Could not find cached package data")
	})
}

func TestDebugDTO(t *testing.T) {
	t.Parallel()

	t.Run("should return error map for nil typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: nil}
		result := querier.DebugDTO()
		require.Contains(t, result, "error")
		assert.Contains(t, result["error"][0], "nil")
	})

	t.Run("should return dump for each package", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {Name: "a", Path: "pkg/a"},
					"pkg/b": {Name: "b", Path: "pkg/b"},
				},
			},
		}
		result := querier.DebugDTO()
		assert.Contains(t, result, "pkg/a")
		assert.Contains(t, result, "pkg/b")
	})
}

func TestDebugPackageDTO(t *testing.T) {
	t.Parallel()

	t.Run("should handle missing package", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		lines := querier.DebugPackageDTO("pkg/missing")
		output := strings.Join(lines, "\n")
		assert.Contains(t, output, "ERROR: Package not found")
	})

	t.Run("should dump package with types, funcs, imports, and version", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"pkg/a": {
						Name:    "a",
						Path:    "pkg/a",
						Version: "v1.2.3",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:       "User",
								TypeString: "pkg/a.User",
								Fields: []*inspector_dto.Field{
									{Name: "Name", TypeString: "string"},
								},
								Methods: []*inspector_dto.Method{
									{
										Name:                 "GetName",
										DeclaringPackagePath: "pkg/a",
										DeclaringTypeName:    "User",
										Signature: inspector_dto.FunctionSignature{
											Results: []string{"string"},
										},
									},
									{
										Name:                 "SetName",
										IsPointerReceiver:    true,
										DeclaringPackagePath: "pkg/a",
										DeclaringTypeName:    "User",
										Signature: inspector_dto.FunctionSignature{
											Params: []string{"string"},
										},
									},
									{
										Name:                 "Promoted",
										DeclaringPackagePath: "other/pkg",
										DeclaringTypeName:    "Base",
										Signature:            inspector_dto.FunctionSignature{},
									},
								},
							},
						},
						Funcs: map[string]*inspector_dto.Function{
							"NewUser": {Name: "NewUser"},
						},
						FileImports: map[string]map[string]string{
							"/src/a.go": {"fmt": "fmt"},
						},
					},
				},
			},
		}
		lines := querier.DebugPackageDTO("pkg/a")
		output := strings.Join(lines, "\n")
		assert.Contains(t, output, "Name:    a")
		assert.Contains(t, output, "Path:    pkg/a")
		assert.Contains(t, output, "Version: v1.2.3")
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "NewUser")
		assert.Contains(t, output, "/src/a.go")
		assert.Contains(t, output, "GetName")
		assert.Contains(t, output, "SetName")
		assert.Contains(t, output, "(*T)")
		assert.Contains(t, output, "Promoted from")
	})
}

func TestFindPropsType(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for missing file", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{localPackageFiles: map[string]*goast.File{}}
		assert.Nil(t, querier.FindPropsType("/missing.go"))
	})

	t.Run("should return nil when Props type not found", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		src := `package main
type User struct { Name string }
`
		file, err := parser.ParseFile(fset, "main.go", src, 0)
		require.NoError(t, err)

		querier := &TypeQuerier{
			localPackageFiles: map[string]*goast.File{
				"/src/main.go": file,
			},
		}
		assert.Nil(t, querier.FindPropsType("/src/main.go"))
	})

	t.Run("should find Props type", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		src := `package main
type Props struct { Title string }
`
		file, err := parser.ParseFile(fset, "main.go", src, 0)
		require.NoError(t, err)

		querier := &TypeQuerier{
			localPackageFiles: map[string]*goast.File{
				"/src/main.go": file,
			},
		}
		result := querier.FindPropsType("/src/main.go")
		require.NotNil(t, result)
		identifier, ok := result.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "Props", identifier.Name)
	})
}

func TestFindRenderReturnType(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for nil typeData", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{typeData: nil}
		assert.Nil(t, querier.FindRenderReturnType("pkg/a", "/src/a.go"))
	})

	t.Run("should return nil for missing package", func(t *testing.T) {
		t.Parallel()

		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		assert.Nil(t, querier.FindRenderReturnType("pkg/missing", "/src/a.go"))
	})

	t.Run("should find Render function return type", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my/component": {
						Name: "component",
						Path: "my/component",
						Funcs: map[string]*inspector_dto.Function{
							"Render": {
								Name: "Render",
								Signature: inspector_dto.FunctionSignature{
									Results: []string{"*html.Node"},
								},
							},
						},
						FileImports: map[string]map[string]string{
							"/src/component.go": {
								"component": "my/component",
							},
						},
					},
				},
			},
		}
		result := querier.FindRenderReturnType("my/component", "/src/component.go")
		assert.NotNil(t, result)
	})
}

func TestGetAllFieldsAndMethods(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for nil baseType", func(t *testing.T) {
		t.Parallel()
		querier := &TypeQuerier{
			typeData: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}
		assert.Nil(t, querier.GetAllFieldsAndMethods(nil, "pkg/a", "/src/a.go"))
	})
}
