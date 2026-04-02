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

// This file contains unit tests for the TypeQuerier's query logic.
// It operates on hand-crafted, in-memory TypeData structures to test the
// querier in complete isolation from the encoding process.

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

func newLocalFiles(t *testing.T, packageName string, content string) map[string]*ast.File {
	fset := token.NewFileSet()

	filePath := packageName + "/local.go"
	file, err := parser.ParseFile(fset, filePath, "package "+packageName+"\n"+content, 0)
	require.NoError(t, err)
	return map[string]*ast.File{filePath: file}
}

func TestTypeQuerier_Queries(t *testing.T) {
	t.Parallel()

	t.Run("FindFieldInfo and FindFieldType", func(t *testing.T) {
		t.Parallel()

		t.Run("Basic Field", func(t *testing.T) {
			t.Parallel()
			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"proj": importerPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:              "User",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "Name", TypeString: "string"}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("User"), "Name", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "Name", fieldInfo.Name)
			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldInfo.Type))
			assert.Equal(t, "Name", fieldInfo.PropName)
			assert.False(t, fieldInfo.IsRequired)
		})

		t.Run("should find field on an unqualified type identifier when given the correct package context", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "testcase_01_prop_validation"
			importerFilePath := "testcase_01_prop_validation/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "main",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"main": importerPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Response": {
								Name:              "Response",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields: []*inspector_dto.Field{
									{Name: "Title", TypeString: "string"},
									{Name: "Count", TypeString: "int"},
								},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})

			unqualifiedTypeExpr := ast.NewIdent("Response")
			fieldInfo := inspector.FindFieldInfo(context.Background(), unqualifiedTypeExpr, "Title", importerPackagePath, importerFilePath)

			require.NotNil(t, fieldInfo, "FindFieldInfo failed to resolve a field on an unqualified identifier using the provided package context")
			assert.Equal(t, "Title", fieldInfo.Name)
			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("Field with Struct Tags", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"proj": importerPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:              "User",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields: []*inspector_dto.Field{
									{Name: "FullName", TypeString: "string", RawTag: `prop:"name" validate:"required"`},
								},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("User"), "FullName", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.True(t, fieldInfo.IsRequired)
			assert.Equal(t, "name", fieldInfo.PropName)
		})

		t.Run("Promoted Field from Embedded Struct", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"proj": importerPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Base": {
								Name:              "Base",
								PackagePath:       importerPackagePath,
								TypeString:        "proj.Base",
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "ID", TypeString: "int"}},
							},
							"User": {
								Name:              "User",
								PackagePath:       importerPackagePath,
								TypeString:        "proj.User",
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "Base", TypeString: "proj.Base", IsEmbedded: true}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("User"), "ID", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "int", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("Shadowed Field", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"proj": importerPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Base": {
								Name:              "Base",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "ID", TypeString: "int"}},
							},
							"User": {
								Name:              "User",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields: []*inspector_dto.Field{
									{Name: "Base", TypeString: "proj.Base", IsEmbedded: true},
									{Name: "ID", TypeString: "string"},
								},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("User"), "ID", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("Field from Pointer to Embedded Struct", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"proj": {
						Name: "proj",
						Path: "proj",
						FileImports: map[string]map[string]string{
							importerFilePath: {"proj": importerPackagePath, "time": "time"},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Base": {
								Name:              "Base",
								PackagePath:       "proj",
								TypeString:        "proj.Base",
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "CreatedAt", TypeString: "time.Time"}},
							},
							"User": {
								Name:              "User",
								PackagePath:       "proj",
								TypeString:        "proj.User",
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "Base", TypeString: "*proj.Base", IsEmbedded: true}},
							},
						},
					},
					"time": {Name: "time", Path: "time"},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("User"), "CreatedAt", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "time.Time", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("Field with Pointer Type", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"proj": importerPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:              "User",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "Manager", TypeString: "*proj.User"}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldType := inspector.FindFieldType(ast.NewIdent("User"), "Manager", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldType)
			_, ok := fieldType.(*ast.StarExpr)
			assert.True(t, ok)
			assert.Equal(t, "*proj.User", goastutil.ASTToTypeString(fieldType))
		})

		t.Run("Field with Instantiated Generic Type", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			modelsPackagePath := "proj/models"
			modelsFilePath := "proj/models/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"models": modelsPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Container": {
								Name:              "Container",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "StringBox", TypeString: "models.Box[string]"}},
							},
						},
					},
					modelsPackagePath: {
						Name: "models",
						Path: modelsPackagePath,
						FileImports: map[string]map[string]string{
							modelsFilePath: {"models": modelsPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Box": {Name: "Box", PackagePath: modelsPackagePath, DefinedInFilePath: modelsFilePath},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldType := inspector.FindFieldType(ast.NewIdent("Container"), "StringBox", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldType)
			_, ok := fieldType.(*ast.IndexExpr)
			assert.True(t, ok, "Expected AST node to be an IndexExpr for generic type")
			assert.Equal(t, "models.Box[string]", goastutil.ASTToTypeString(fieldType))
		})

		t.Run("Not Found", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:              "User",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "Name", TypeString: "string"}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("User"), "Age", importerPackagePath, importerFilePath)
			assert.Nil(t, fieldInfo)
		})

		t.Run("Deeply Embedded Promoted Field", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"proj": importerPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"A": {Name: "A", PackagePath: importerPackagePath, TypeString: "proj.A", DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{{Name: "ID", TypeString: "int"}}},
							"B": {Name: "B", PackagePath: importerPackagePath, TypeString: "proj.B", DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{{Name: "A", TypeString: "proj.A", IsEmbedded: true}}},
							"C": {Name: "C", PackagePath: importerPackagePath, TypeString: "proj.C", DefinedInFilePath: importerFilePath, Fields: []*inspector_dto.Field{{Name: "B", TypeString: "proj.B", IsEmbedded: true}}},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("C"), "ID", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldInfo)
			assert.Equal(t, "ID", fieldInfo.Name)
			assert.Equal(t, "int", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("Field is a Function Type", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Runner": {
								Name:              "Runner",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields:            []*inspector_dto.Field{{Name: "Callback", TypeString: "func(int) error"}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			fieldType := inspector.FindFieldType(ast.NewIdent("Runner"), "Callback", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldType)
			_, ok := fieldType.(*ast.FuncType)
			assert.True(t, ok)
			assert.Equal(t, "func(int) error", goastutil.ASTToTypeString(fieldType))
		})
	})

	t.Run("FindMethodSignature and FindMethodReturnType", func(t *testing.T) {
		t.Parallel()

		importerPackagePath := "proj"
		importerFilePath := "proj/file.go"

		t.Run("Method on Value Receiver", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj", Path: importerPackagePath,
						FileImports: map[string]map[string]string{importerFilePath: {}},
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:              "User",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Methods:           []*inspector_dto.Method{{Name: "GetID", Signature: inspector_dto.FunctionSignature{Results: []string{"int"}}}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			sig := inspector.FindMethodSignature(ast.NewIdent("User"), "GetID", importerPackagePath, importerFilePath)
			require.NotNil(t, sig)
			assert.Equal(t, []string{"int"}, sig.Results)
		})

		t.Run("Method on Pointer Receiver", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj", Path: importerPackagePath,
						FileImports: map[string]map[string]string{importerFilePath: {}},
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:              "User",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Methods: []*inspector_dto.Method{{
									Name: "SetName", Signature: inspector_dto.FunctionSignature{Params: []string{"string"}}, IsPointerReceiver: true,
								}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})

			sigOnValue := inspector.FindMethodSignature(ast.NewIdent("User"), "SetName", importerPackagePath, importerFilePath)
			assert.Nil(t, sigOnValue, "Strict check should not find pointer method on value")

			sigOnPtr := inspector.FindMethodSignature(&ast.StarExpr{X: ast.NewIdent("User")}, "SetName", importerPackagePath, importerFilePath)
			require.NotNil(t, sigOnPtr)
			assert.Equal(t, []string{"string"}, sigOnPtr.Params)
		})

		t.Run("Promoted Method from Embedded Struct", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj", Path: importerPackagePath,
						FileImports: map[string]map[string]string{importerFilePath: {"proj": importerPackagePath}},
						NamedTypes: map[string]*inspector_dto.Type{
							"Base": {
								Name: "Base", PackagePath: importerPackagePath, TypeString: "proj.Base", DefinedInFilePath: importerFilePath,
								Methods: []*inspector_dto.Method{{Name: "Ping", Signature: inspector_dto.FunctionSignature{}}},
							},
							"Outer": {
								Name: "Outer", PackagePath: importerPackagePath, TypeString: "proj.Outer", DefinedInFilePath: importerFilePath,
								Fields: []*inspector_dto.Field{{Name: "Base", TypeString: "proj.Base", IsEmbedded: true}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			sig := inspector.FindMethodSignature(ast.NewIdent("Outer"), "Ping", importerPackagePath, importerFilePath)
			require.NotNil(t, sig)
		})

		t.Run("Method on an Interface Type", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "main"
			importerFilePath := "main/file.go"
			ioPackagePath := "io"
			ioFilePath := "io/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					ioPackagePath: {
						Name: "io", Path: ioPackagePath,
						FileImports: map[string]map[string]string{ioFilePath: {}},
						NamedTypes: map[string]*inspector_dto.Type{
							"Reader": {
								Name: "Reader", PackagePath: ioPackagePath, DefinedInFilePath: ioFilePath,
								Methods: []*inspector_dto.Method{{
									Name: "Read", Signature: inspector_dto.FunctionSignature{Params: []string{"[]uint8"}, Results: []string{"int", "error"}},
								}},
							},
						},
					},
					importerPackagePath: {
						Name: "main", Path: importerPackagePath,
						FileImports: map[string]map[string]string{importerFilePath: {"io": ioPackagePath}},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			sig := inspector.FindMethodSignature(&ast.SelectorExpr{X: ast.NewIdent("io"), Sel: ast.NewIdent("Reader")}, "Read", importerPackagePath, importerFilePath)
			require.NotNil(t, sig)
			assert.Equal(t, []string{"[]uint8"}, sig.Params)
			assert.Equal(t, []string{"int", "error"}, sig.Results)
		})

		t.Run("Method with Multiple Return Values", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj", Path: importerPackagePath,
						FileImports: map[string]map[string]string{importerFilePath: {"proj": importerPackagePath}},
						NamedTypes: map[string]*inspector_dto.Type{
							"Client": {
								Name: "Client", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath,
								Methods: []*inspector_dto.Method{{
									Name: "Connect", Signature: inspector_dto.FunctionSignature{Results: []string{"*proj.Conn", "error"}},
								}},
							},
							"Conn": {Name: "Conn", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			retType := inspector.FindMethodReturnType(ast.NewIdent("Client"), "Connect", importerPackagePath, importerFilePath)
			require.NotNil(t, retType)
			assert.Equal(t, "*proj.Conn", goastutil.ASTToTypeString(retType))
		})
	})

	t.Run("FindFuncSignature and FindFuncReturnType", func(t *testing.T) {
		t.Parallel()

		importerPackagePath := "proj/main"
		importerFilePath := "proj/main/file.go"
		utilsPackagePath := "proj/utils"
		utilsFilePath := "proj/utils/file.go"

		t.Run("Simple Package-Level Function", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					utilsPackagePath: {
						Name: "utils", Path: utilsPackagePath,
						FileImports: map[string]map[string]string{utilsFilePath: {}},
						Funcs: map[string]*inspector_dto.Function{
							"Now": {Name: "Now", Signature: inspector_dto.FunctionSignature{Results: []string{"time.Time"}}},
						},
					},
					importerPackagePath: {
						Name: "main", Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"utils": utilsPackagePath},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			sig := inspector.FindFuncSignature("utils", "Now", importerPackagePath, importerFilePath)
			require.NotNil(t, sig)
			assert.Equal(t, []string{"time.Time"}, sig.Results)
		})

		t.Run("Function in Aliased Package", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					utilsPackagePath: {
						Name: "utils", Path: utilsPackagePath,
						FileImports: map[string]map[string]string{utilsFilePath: {}},
						Funcs: map[string]*inspector_dto.Function{
							"Helper": {Name: "Helper", Signature: inspector_dto.FunctionSignature{}},
						},
					},
					importerPackagePath: {
						Name: "main", Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"u": utilsPackagePath},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			sig := inspector.FindFuncSignature("u", "Helper", importerPackagePath, importerFilePath)
			require.NotNil(t, sig)
		})

		t.Run("Not Found", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					utilsPackagePath:    {Name: "utils", Path: utilsPackagePath, Funcs: map[string]*inspector_dto.Function{}},
					importerPackagePath: {Name: "main", Path: importerPackagePath, FileImports: map[string]map[string]string{importerFilePath: {"utils": utilsPackagePath}}},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			sig := inspector.FindFuncSignature("utils", "DoesNotExist", importerPackagePath, importerFilePath)
			assert.Nil(t, sig)
		})
	})

	t.Run("ResolveToUnderlyingAST", func(t *testing.T) {
		t.Parallel()

		t.Run("Simple Alias", func(t *testing.T) {
			t.Parallel()
			files := newLocalFiles(t, "main", `type UserID = string`)
			inspector := inspector_domain.NewTypeQuerier(files, nil, inspector_dto.Config{})
			resolved := inspector.ResolveToUnderlyingAST(ast.NewIdent("UserID"), "main/local.go")
			assert.Equal(t, "string", goastutil.ASTToTypeString(resolved))
		})

		t.Run("Alias to Imported Type", func(t *testing.T) {
			t.Parallel()

			files := newLocalFiles(t, "main", `import "models"; type User = models.User`)
			inspector := inspector_domain.NewTypeQuerier(files, nil, inspector_dto.Config{})
			resolved := inspector.ResolveToUnderlyingAST(ast.NewIdent("User"), "main/local.go")
			assert.Equal(t, "models.User", goastutil.ASTToTypeString(resolved))
		})

		t.Run("Reaches recursion guard", func(t *testing.T) {
			t.Parallel()

			files := newLocalFiles(t, "main", `type A = B; type B = A`)
			inspector := inspector_domain.NewTypeQuerier(files, nil, inspector_dto.Config{})
			resolved := inspector.ResolveToUnderlyingAST(ast.NewIdent("A"), "main/local.go")
			assert.Contains(t, []string{"A", "B"}, goastutil.ASTToTypeString(resolved))
		})
	})

	t.Run("Other Inspector Methods", func(t *testing.T) {
		t.Parallel()

		t.Run("FindPropsType Found", func(t *testing.T) {
			t.Parallel()
			packageName := "main"
			filePath := packageName + "/local.go"
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, filePath, "package "+packageName+"\ntype Props struct{ Name string }", 0)
			require.NoError(t, err)
			localFiles := map[string]*ast.File{filePath: file}

			inspector := inspector_domain.NewTypeQuerier(localFiles, nil, inspector_dto.Config{})
			propsType := inspector.FindPropsType(filePath)
			require.NotNil(t, propsType)
			assert.Equal(t, "Props", goastutil.ASTToTypeString(propsType))
		})

		t.Run("FindRenderReturnType Found", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "main",
						Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"main": importerPackagePath, "proj": importerPackagePath},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Component": {Name: "Component", PackagePath: importerPackagePath, DefinedInFilePath: importerFilePath},
						},
						Funcs: map[string]*inspector_dto.Function{
							"Render": {Name: "Render", Signature: inspector_dto.FunctionSignature{Results: []string{"*proj.Component"}}},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			retType := inspector.FindRenderReturnType(importerPackagePath, importerFilePath)
			require.NotNil(t, retType)
			assert.Equal(t, "*proj.Component", goastutil.ASTToTypeString(retType))
		})

		t.Run("ToExportedName", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "Name", inspector_domain.ToExportedName("name"))
			assert.Equal(t, "URL", inspector_domain.ToExportedName("URL"))
			assert.Equal(t, "", inspector_domain.ToExportedName(""))
		})
	})

	t.Run("File-Scoped Import Resolution", func(t *testing.T) {
		t.Parallel()

		mainPackagePath := "proj/main"
		utilsPackagePath := "proj/utils"
		uuidPackagePath := "github.com/google/uuid"

		mainFileA := "proj/main/a.go"
		mainFileB := "proj/main/b.go"

		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				mainPackagePath: {
					Name: "main", Path: mainPackagePath,
					FileImports: map[string]map[string]string{

						mainFileA: {"u": uuidPackagePath},

						mainFileB: {"u": utilsPackagePath, ".": utilsPackagePath},
					},
					NamedTypes: map[string]*inspector_dto.Type{
						"Request": {Name: "Request", PackagePath: mainPackagePath, DefinedInFilePath: mainFileA},
					},
				},
				utilsPackagePath: {
					Name: "utils", Path: utilsPackagePath,
					FileImports: map[string]map[string]string{
						"proj/utils/util.go": {},
					},
					NamedTypes: map[string]*inspector_dto.Type{
						"Helper": {Name: "Helper", PackagePath: utilsPackagePath, DefinedInFilePath: "proj/utils/util.go", Fields: []*inspector_dto.Field{{Name: "Status", TypeString: "string"}}},
					},
					Funcs: map[string]*inspector_dto.Function{
						"GetStatus": {Name: "GetStatus", Signature: inspector_dto.FunctionSignature{Results: []string{"string"}}},
					},
				},
				uuidPackagePath: {
					Name: "uuid", Path: uuidPackagePath,
					FileImports: map[string]map[string]string{
						"github.com/google/uuid/uuid.go": {},
					},
					NamedTypes: map[string]*inspector_dto.Type{
						"UUID": {Name: "UUID", PackagePath: uuidPackagePath, DefinedInFilePath: "github.com/google/uuid/uuid.go"},
					},
				},
			},
		}
		inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})

		t.Run("should resolve alias correctly based on the calling file's context", func(t *testing.T) {
			t.Parallel()

			expressionA, err := parser.ParseExpr("u.UUID")
			require.NoError(t, err)
			typeFromA, _ := inspector.ResolveExprToNamedType(
				expressionA,
				mainPackagePath,
				mainFileA,
			)
			require.NotNil(t, typeFromA)
			assert.Equal(t, "UUID", typeFromA.Name)

			expressionB, err := parser.ParseExpr("u.Helper")
			require.NoError(t, err)
			typeFromB, _ := inspector.ResolveExprToNamedType(
				expressionB,
				mainPackagePath,
				mainFileB,
			)
			require.NotNil(t, typeFromB)
			assert.Equal(t, "Helper", typeFromB.Name)
		})

		t.Run("should fail to resolve alias when context is from the wrong file", func(t *testing.T) {
			t.Parallel()

			expressionWrong, err := parser.ParseExpr("u.UUID")
			require.NoError(t, err)
			typeFromB, _ := inspector.ResolveExprToNamedType(
				expressionWrong,
				mainPackagePath,
				mainFileB,
			)
			assert.Nil(t, typeFromB, "Should not find uuid.UUID via alias 'u' in file B's context")
		})

		t.Run("should resolve unqualified type from a dot import in the correct file context", func(t *testing.T) {
			t.Parallel()

			expressionDotB, err := parser.ParseExpr("Helper")
			require.NoError(t, err)
			typeFromDot, _ := inspector.ResolveExprToNamedType(
				expressionDotB,
				mainPackagePath,
				mainFileB,
			)
			require.NotNil(t, typeFromDot)
			assert.Equal(t, "Helper", typeFromDot.Name)

			expressionDotA, err := parser.ParseExpr("Helper")
			require.NoError(t, err)
			typeFromA, _ := inspector.ResolveExprToNamedType(
				expressionDotA,
				mainPackagePath,
				mainFileA,
			)
			assert.Nil(t, typeFromA, "Should not find Helper unqualified in file A's context")
		})

		t.Run("should find a dot-imported function signature from the correct file context", func(t *testing.T) {
			t.Parallel()

			pkgNameForDotImport := "."
			sig := inspector.FindFuncSignature(pkgNameForDotImport, "GetStatus", mainPackagePath, mainFileB)
			require.NotNil(t, sig)
			assert.Equal(t, []string{"string"}, sig.Results)
		})
	})

	t.Run("Advanced Interaction and Resilience", func(t *testing.T) {
		t.Parallel()

		t.Run("should find field by resolving a local alias to a cached type", func(t *testing.T) {
			t.Parallel()
			importerPackagePath := "proj"
			importerFilePath := "proj/main.go"
			modelsPackagePath := "proj/models"
			modelsFilePath := "proj/models/user.go"

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "main", Path: importerPackagePath,
						FileImports: map[string]map[string]string{
							importerFilePath: {"models": modelsPackagePath},
						},
					},
					modelsPackagePath: {
						Name: "models", Path: modelsPackagePath,
						FileImports: map[string]map[string]string{
							modelsFilePath: {},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:              "User",
								PackagePath:       modelsPackagePath,
								DefinedInFilePath: modelsFilePath,
								Fields:            []*inspector_dto.Field{{Name: "ID", TypeString: "int"}},
							},
						},
					},
				},
			}

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, importerFilePath, "package main\nimport \"proj/models\"\ntype APIUser = models.User", 0)
			require.NoError(t, err)
			localFiles := map[string]*ast.File{importerFilePath: file}

			inspector := inspector_domain.NewTypeQuerier(localFiles, typeData, inspector_dto.Config{})
			fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("APIUser"), "ID", importerPackagePath, importerFilePath)

			require.NotNil(t, fieldInfo)
			assert.Equal(t, "ID", fieldInfo.Name)
			assert.Equal(t, "int", goastutil.ASTToTypeString(fieldInfo.Type))
		})

		t.Run("should not panic when querying a field with a corrupt type string", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj", Path: importerPackagePath,
						FileImports: map[string]map[string]string{importerFilePath: {}},
						NamedTypes: map[string]*inspector_dto.Type{
							"Corrupted": {
								Name:              "Corrupted",
								PackagePath:       importerPackagePath,
								DefinedInFilePath: importerFilePath,
								Fields: []*inspector_dto.Field{
									{Name: "BadField", TypeString: "map[string]some garbage type"},
								},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			assert.NotPanics(t, func() {
				fieldInfo := inspector.FindFieldInfo(context.Background(), ast.NewIdent("Corrupted"), "BadField", importerPackagePath, importerFilePath)
				require.NotNil(t, fieldInfo)
				typeString := goastutil.ASTToTypeString(fieldInfo.Type)

				assert.Equal(t, "any /* failed to parse type string: map[string]some_garbage_type */", typeString)
			})
		})

		t.Run("should find promoted method from an embedded instantiated generic type's argument", func(t *testing.T) {
			t.Parallel()

			importerPackagePath := "proj"
			importerFilePath := "proj/file.go"
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					importerPackagePath: {
						Name: "proj", Path: importerPackagePath,
						FileImports: map[string]map[string]string{importerFilePath: {"proj": importerPackagePath}},
						NamedTypes: map[string]*inspector_dto.Type{
							"MyService": {
								Name: "MyService", PackagePath: importerPackagePath, TypeString: "proj.MyService", DefinedInFilePath: importerFilePath,
								Methods: []*inspector_dto.Method{{
									Name: "Ping", Signature: inspector_dto.FunctionSignature{Results: []string{"string"}},
								}},
							},
							"Box": {Name: "Box", PackagePath: importerPackagePath, TypeString: "proj.Box", DefinedInFilePath: importerFilePath},
							"ServiceContainer": {
								Name: "ServiceContainer", PackagePath: importerPackagePath, TypeString: "proj.ServiceContainer", DefinedInFilePath: importerFilePath,
								Fields: []*inspector_dto.Field{{
									Name: "Box", TypeString: "proj.Box[proj.MyService]", IsEmbedded: true,
								}},
							},
						},
					},
				},
			}
			inspector := inspector_domain.NewTypeQuerier(nil, typeData, inspector_dto.Config{})
			sig := inspector.FindMethodSignature(ast.NewIdent("ServiceContainer"), "Ping", importerPackagePath, importerFilePath)

			require.NotNil(t, sig, "Should have found the 'Ping' method through generic type argument traversal")
			require.Len(t, sig.Results, 1)
			assert.Equal(t, "string", sig.Results[0])
		})
	})
}
