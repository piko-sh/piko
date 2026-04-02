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

package inspector_domain_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func astExpr(t *testing.T, expression string) ast.Expr {
	e, err := parser.ParseExpr(expression)
	require.NoError(t, err)
	return e
}

func setupInspector(t *testing.T, sources map[string]string, typeData *inspector_dto.TypeData) *inspector_domain.TypeQuerier {
	localPackageFiles := make(map[string]*ast.File)
	fset := token.NewFileSet()

	for filePath, content := range sources {
		file, err := parser.ParseFile(fset, filePath, content, parser.AllErrors)
		require.NoError(t, err)

		localPackageFiles[filePath] = file
	}

	return inspector_domain.NewTypeQuerier(localPackageFiles, typeData, inspector_dto.Config{})
}

func TestFieldSearcher(t *testing.T) {
	t.Parallel()
	importerPackagePath := "proj"
	importerFilePath := "proj/main.go"
	modelsPackagePath := "proj/models"
	modelsFilePath := "proj/models/models.go"

	t.Run("Basic Lookups", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				importerPackagePath: {
					Path: importerPackagePath, Name: "main",
					FileImports: map[string]map[string]string{importerFilePath: {}},
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {
							Name:              "User",
							PackagePath:       importerPackagePath,
							DefinedInFilePath: importerFilePath,
							Fields: []*inspector_dto.Field{
								{Name: "Name", TypeString: "string"},
								{Name: "age", TypeString: "int"},
							},
						},
					},
				},
			},
		}
		inspector := setupInspector(t, nil, typeData)

		t.Run("should find exported field on a value type", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "User"), "Name", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "Name", fieldInfo.Name)
			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("should find exported field on a pointer type", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "*User"), "Name", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "Name", fieldInfo.Name)
		})

		t.Run("should return nil for non-existent field", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "User"), "Address", importerPackagePath, importerFilePath)
			assert.Nil(t, fieldInfo)
		})
	})

	t.Run("Embedding and Shadowing", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				importerPackagePath: {
					Path: importerPackagePath, Name: "main",
					FileImports: map[string]map[string]string{importerFilePath: {"main": importerPackagePath}},
					NamedTypes: map[string]*inspector_dto.Type{
						"Base": {
							Name: "Base", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Fields: []*inspector_dto.Field{{Name: "ID", TypeString: "int"}},
						},
						"Mixin": {
							Name: "Mixin", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Fields: []*inspector_dto.Field{{Name: "Status", TypeString: "string"}},
						},
						"Model": {
							Name: "Model", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Fields: []*inspector_dto.Field{
								{Name: "Base", TypeString: "main.Base", IsEmbedded: true},
								{Name: "Mixin", TypeString: "*main.Mixin", IsEmbedded: true},
							},
						},
						"ShadowModel": {
							Name: "ShadowModel", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Fields: []*inspector_dto.Field{
								{Name: "Base", TypeString: "main.Base", IsEmbedded: true},
								{Name: "ID", TypeString: "string"},
							},
						},
					},
				},
			},
		}
		inspector := setupInspector(t, nil, typeData)

		t.Run("should find promoted field from embedded value type", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "Model"), "ID", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "ID", fieldInfo.Name)
			assert.Equal(t, "int", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("should find promoted field from embedded pointer type", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "Model"), "Status", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "Status", fieldInfo.Name)
			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("should find the shadowing field, not the promoted one", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "ShadowModel"), "ID", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "ID", fieldInfo.Name)
			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldInfo.Type), "Should have found the string shadow field, not the int from Base")
		})
	})

	t.Run("Local Alias Resolution", func(t *testing.T) {
		t.Parallel()
		sources := map[string]string{
			importerFilePath: `
                package main
                import "proj/models"
                type UserAlias = models.User
            `,
		}
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				importerPackagePath: {Path: importerPackagePath, Name: "main",
					FileImports: map[string]map[string]string{importerFilePath: {"models": modelsPackagePath}},
				},
				modelsPackagePath: {Path: modelsPackagePath, Name: "models",
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {Name: "User", PackagePath: modelsPackagePath, DefinedInFilePath: modelsFilePath,
							Fields: []*inspector_dto.Field{{Name: "ID", TypeString: "int"}},
						},
					},
				},
			},
		}
		inspector := setupInspector(t, sources, typeData)

		t.Run("should find field through a simple type alias", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "UserAlias"), "ID", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "ID", fieldInfo.Name)
			assert.Equal(t, "int", goastutil.ASTToTypeString(fieldInfo.Type))
		})
	})

	t.Run("Generics", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				importerPackagePath: {
					Path: importerPackagePath, Name: "main",
					FileImports: map[string]map[string]string{importerFilePath: {"main": importerPackagePath}},
					NamedTypes: map[string]*inspector_dto.Type{
						"Cache": {
							Name: "Cache", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							TypeParams: []string{"T"},
							Fields:     []*inspector_dto.Field{{Name: "Value", TypeString: "T"}},
						},
						"Container": {
							Name: "Container", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Fields: []*inspector_dto.Field{
								{Name: "Cache", TypeString: "main.Cache[string]", IsEmbedded: true},
							},
						},
					},
				},
			},
		}
		inspector := setupInspector(t, nil, typeData)

		t.Run("should find field on generic type instantiation", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "Cache[string]"), "Value", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "Value", fieldInfo.Name)

			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("should find promoted field from embedded generic type", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "Container"), "Value", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "Value", fieldInfo.Name)
			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldInfo.Type))
		})
	})

	t.Run("Cycle Prevention and Diamond Embedding", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				importerPackagePath: {
					Path: importerPackagePath, Name: "main",
					FileImports: map[string]map[string]string{importerFilePath: {"main": importerPackagePath}},
					NamedTypes: map[string]*inspector_dto.Type{
						"D": {Name: "D", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{{Name: "Value", TypeString: "float64"}}},
						"B": {Name: "B", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{{Name: "D", TypeString: "main.D", IsEmbedded: true}}},
						"C": {Name: "C", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{{Name: "D", TypeString: "main.D", IsEmbedded: true}}},
						"A": {Name: "A", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{
							{Name: "B", TypeString: "main.B", IsEmbedded: true},
							{Name: "C", TypeString: "main.C", IsEmbedded: true},
						}},
					},
				},
			},
		}
		inspector := setupInspector(t, nil, typeData)

		t.Run("should find field through diamond embedding without error", func(t *testing.T) {
			t.Parallel()
			fieldInfo := inspector.FindFieldInfo(context.Background(), astExpr(t, "A"), "Value", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "Value", fieldInfo.Name)
			assert.Equal(t, "float64", goastutil.ASTToTypeString(fieldInfo.Type))
		})
	})
}

func TestMethodSearcher(t *testing.T) {
	t.Parallel()
	importerPackagePath := "proj"
	importerFilePath := "proj/main.go"
	modelsPackagePath := "proj/models"
	modelsFilePath := "proj/models/models.go"

	t.Run("Receiver Type Correctness", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				importerPackagePath: {Path: importerPackagePath, Name: "main",
					FileImports: map[string]map[string]string{
						importerFilePath: {"models": modelsPackagePath},
					},
					NamedTypes: map[string]*inspector_dto.Type{
						"Controller": {Name: "Controller", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Methods: []*inspector_dto.Method{
								{Name: "GetValue", IsPointerReceiver: false, Signature: inspector_dto.FunctionSignature{Results: []string{"string"}}},
								{Name: "SetValue", IsPointerReceiver: true, Signature: inspector_dto.FunctionSignature{Params: []string{"string"}}},
							},
						},
					},
				},
				modelsPackagePath: {Path: modelsPackagePath, Name: "models",
					NamedTypes: map[string]*inspector_dto.Type{
						"Repo": {Name: "Repo", PackagePath: modelsPackagePath, DefinedInFilePath: modelsFilePath,
							Methods: []*inspector_dto.Method{
								{Name: "Find", IsPointerReceiver: true, Signature: inspector_dto.FunctionSignature{}},
							},
						},
					},
				},
			},
		}
		inspector := setupInspector(t, nil, typeData)

		t.Run("should find value receiver method on value type", func(t *testing.T) {
			t.Parallel()
			sig := inspector.FindMethodSignature(astExpr(t, "Controller"), "GetValue", importerPackagePath, importerFilePath)
			assert.NotNil(t, sig)
		})

		t.Run("should find value receiver method on pointer type", func(t *testing.T) {
			t.Parallel()
			sig := inspector.FindMethodSignature(astExpr(t, "*Controller"), "GetValue", importerPackagePath, importerFilePath)
			assert.NotNil(t, sig)
		})

		t.Run("should NOT find pointer receiver method on value type in same package", func(t *testing.T) {
			t.Parallel()
			sig := inspector.FindMethodSignature(astExpr(t, "Controller"), "SetValue", importerPackagePath, importerFilePath)
			assert.Nil(t, sig, "A value type does not have pointer receiver methods in its method set")
		})

		t.Run("should find pointer receiver method on pointer type", func(t *testing.T) {
			t.Parallel()
			sig := inspector.FindMethodSignature(astExpr(t, "*Controller"), "SetValue", importerPackagePath, importerFilePath)
			assert.NotNil(t, sig)
		})

		t.Run("should find pointer receiver method on addressable value type from another package", func(t *testing.T) {
			t.Parallel()

			sig := inspector.FindMethodSignature(astExpr(t, "models.Repo"), "Find", importerPackagePath, importerFilePath)
			assert.NotNil(t, sig, "Should find pointer method on addressable value from another package")
		})
	})

	t.Run("Embedding and Shadowing", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				importerPackagePath: {Path: importerPackagePath, Name: "main",
					FileImports: map[string]map[string]string{importerFilePath: {"main": importerPackagePath}},
					NamedTypes: map[string]*inspector_dto.Type{
						"Logger": {Name: "Logger", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Methods: []*inspector_dto.Method{{Name: "Log", IsPointerReceiver: false, Signature: inspector_dto.FunctionSignature{}}},
						},
						"DB": {Name: "DB", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Methods: []*inspector_dto.Method{{Name: "Query", IsPointerReceiver: true, Signature: inspector_dto.FunctionSignature{}}},
						},
						"Service": {Name: "Service", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
							Fields: []*inspector_dto.Field{
								{Name: "Logger", TypeString: "main.Logger", IsEmbedded: true},
								{Name: "DB", TypeString: "*main.DB", IsEmbedded: true},
							},
							Methods: []*inspector_dto.Method{{Name: "Log", IsPointerReceiver: false, Signature: inspector_dto.FunctionSignature{Params: []string{"int"}}}},
						},
					},
				},
			},
		}
		inspector := setupInspector(t, nil, typeData)

		t.Run("should find promoted pointer receiver method", func(t *testing.T) {
			t.Parallel()
			sig := inspector.FindMethodSignature(astExpr(t, "*Service"), "Query", importerPackagePath, importerFilePath)
			require.NotNil(t, sig)
		})

		t.Run("should find shadowing method, not promoted one", func(t *testing.T) {
			t.Parallel()
			sig := inspector.FindMethodSignature(astExpr(t, "Service"), "Log", importerPackagePath, importerFilePath)
			require.NotNil(t, sig)
			require.Len(t, sig.Params, 1, "Should find the shadowing Log(int) method")
			assert.Equal(t, "int", sig.Params[0])
		})
	})

	t.Run("Cycle Prevention and Diamond Embedding", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				importerPackagePath: {
					Path: importerPackagePath, Name: "main",
					FileImports: map[string]map[string]string{importerFilePath: {"main": importerPackagePath}},
					NamedTypes: map[string]*inspector_dto.Type{
						"D": {Name: "D", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath, Methods: []*inspector_dto.Method{{Name: "Exec", Signature: inspector_dto.FunctionSignature{}}}},
						"B": {Name: "B", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{{Name: "D", TypeString: "main.D", IsEmbedded: true}}},
						"C": {Name: "C", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{{Name: "D", TypeString: "main.D", IsEmbedded: true}}},
						"A": {Name: "A", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{
							{Name: "B", TypeString: "main.B", IsEmbedded: true},
							{Name: "C", TypeString: "main.C", IsEmbedded: true},
						}},
					},
				},
			},
		}
		inspector := setupInspector(t, nil, typeData)

		t.Run("should find method through diamond embedding without error", func(t *testing.T) {
			t.Parallel()
			sig := inspector.FindMethodSignature(astExpr(t, "A"), "Exec", importerPackagePath, importerFilePath)
			require.NotNil(t, sig, "The walker should successfully find the promoted method through the diamond path")
		})
	})
}
