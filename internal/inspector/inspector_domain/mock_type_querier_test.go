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
	"context"
	goast "go/ast"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestMockTypeQuerier_ResolveExprToNamedType(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolveExprToNamedTypeFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got, name := m.ResolveExprToNamedType(&goast.Ident{Name: "X"}, "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, "", name)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to ResolveExprToNamedTypeFunc", func(t *testing.T) {
		t.Parallel()
		wantType := &inspector_dto.Type{Name: "Foo"}
		m := &MockTypeQuerier{
			ResolveExprToNamedTypeFunc: func(expression goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
				assert.Equal(t, "pkg/a", importerPackagePath)
				assert.Equal(t, "a.go", importerFilePath)
				return wantType, "Foo"
			},
		}

		got, name := m.ResolveExprToNamedType(&goast.Ident{Name: "Foo"}, "pkg/a", "a.go")

		require.NotNil(t, got)
		assert.Equal(t, "Foo", got.Name)
		assert.Equal(t, "Foo", name)
	})
}

func TestMockTypeQuerier_ResolveExprToNamedTypeWithMemoization(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolveExprToNamedTypeWithMemoizationFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got, name := m.ResolveExprToNamedTypeWithMemoization(context.Background(), &goast.Ident{Name: "X"}, "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, "", name)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to ResolveExprToNamedTypeWithMemoizationFunc", func(t *testing.T) {
		t.Parallel()
		wantType := &inspector_dto.Type{Name: "Bar"}
		m := &MockTypeQuerier{
			ResolveExprToNamedTypeWithMemoizationFunc: func(ctx context.Context, typeExpr goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
				assert.Equal(t, "pkg/b", importerPackagePath)
				assert.Equal(t, "b.go", importerFilePath)
				return wantType, "Bar"
			},
		}

		got, name := m.ResolveExprToNamedTypeWithMemoization(context.Background(), &goast.Ident{Name: "Bar"}, "pkg/b", "b.go")

		require.NotNil(t, got)
		assert.Equal(t, "Bar", got.Name)
		assert.Equal(t, "Bar", name)
	})
}

func TestMockTypeQuerier_ResolveToUnderlyingAST(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolveToUnderlyingASTFunc returns typeExpr unchanged", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		input := &goast.Ident{Name: "Original"}
		got := m.ResolveToUnderlyingAST(input, "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Same(t, input, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to ResolveToUnderlyingASTFunc", func(t *testing.T) {
		t.Parallel()
		resolved := &goast.Ident{Name: "Resolved"}
		m := &MockTypeQuerier{
			ResolveToUnderlyingASTFunc: func(typeExpr goast.Expr, currentFilePath string) goast.Expr {
				assert.Equal(t, "src.go", currentFilePath)
				return resolved
			},
		}

		got := m.ResolveToUnderlyingAST(&goast.Ident{Name: "Alias"}, "src.go")

		assert.Same(t, resolved, got)
	})
}

func TestMockTypeQuerier_ResolveToUnderlyingASTWithContext(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolveToUnderlyingASTWithContextFunc returns typeExpr and empty string", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		input := &goast.Ident{Name: "T"}
		got, filePath := m.ResolveToUnderlyingASTWithContext(context.Background(), input, "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Same(t, input, got)
		assert.Equal(t, "", filePath)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to ResolveToUnderlyingASTWithContextFunc", func(t *testing.T) {
		t.Parallel()
		resolved := &goast.Ident{Name: "Underlying"}
		m := &MockTypeQuerier{
			ResolveToUnderlyingASTWithContextFunc: func(ctx context.Context, typeExpr goast.Expr, currentFilePath string) (goast.Expr, string) {
				assert.Equal(t, "ctx.go", currentFilePath)
				return resolved, "/abs/ctx.go"
			},
		}

		got, fp := m.ResolveToUnderlyingASTWithContext(context.Background(), &goast.Ident{Name: "Alias"}, "ctx.go")

		assert.Same(t, resolved, got)
		assert.Equal(t, "/abs/ctx.go", fp)
	})
}

func TestMockTypeQuerier_GetAllSymbols(t *testing.T) {
	t.Parallel()

	t.Run("nil GetAllSymbolsFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.GetAllSymbols()
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to GetAllSymbolsFunc", func(t *testing.T) {
		t.Parallel()
		want := []inspector_dto.WorkspaceSymbol{
			{Name: "Foo", Kind: "type", PackagePath: "pkg/a"},
		}
		m := &MockTypeQuerier{
			GetAllSymbolsFunc: func() []inspector_dto.WorkspaceSymbol {
				return want
			},
		}

		got := m.GetAllSymbols()

		require.Len(t, got, 1)
		assert.Equal(t, "Foo", got[0].Name)
	})
}

func TestMockTypeQuerier_GetImplementationIndex(t *testing.T) {
	t.Parallel()

	t.Run("nil GetImplementationIndexFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.GetImplementationIndex()
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to GetImplementationIndexFunc", func(t *testing.T) {
		t.Parallel()
		want := NewImplementationIndex(&inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		})
		m := &MockTypeQuerier{
			GetImplementationIndexFunc: func() *ImplementationIndex {
				return want
			},
		}

		got := m.GetImplementationIndex()

		assert.Same(t, want, got)
	})
}

func TestMockTypeQuerier_GetTypeHierarchyIndex(t *testing.T) {
	t.Parallel()

	t.Run("nil GetTypeHierarchyIndexFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.GetTypeHierarchyIndex()
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to GetTypeHierarchyIndexFunc", func(t *testing.T) {
		t.Parallel()
		want := NewTypeHierarchyIndex(&inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		})
		m := &MockTypeQuerier{
			GetTypeHierarchyIndexFunc: func() *TypeHierarchyIndex {
				return want
			},
		}

		got := m.GetTypeHierarchyIndex()

		assert.Same(t, want, got)
	})
}

func TestMockTypeQuerier_FindFieldInfo(t *testing.T) {
	t.Parallel()

	t.Run("nil FindFieldInfoFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindFieldInfo(context.Background(), &goast.Ident{Name: "T"}, "Name", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindFieldInfoFunc", func(t *testing.T) {
		t.Parallel()
		want := &inspector_dto.FieldInfo{Name: "ID", PropName: "id"}
		m := &MockTypeQuerier{
			FindFieldInfoFunc: func(ctx context.Context, baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) *inspector_dto.FieldInfo {
				assert.Equal(t, "ID", fieldName)
				assert.Equal(t, "pkg/a", importerPackagePath)
				assert.Equal(t, "a.go", importerFilePath)
				return want
			},
		}

		got := m.FindFieldInfo(context.Background(), &goast.Ident{Name: "MyStruct"}, "ID", "pkg/a", "a.go")

		require.NotNil(t, got)
		assert.Equal(t, "ID", got.Name)
		assert.Equal(t, "id", got.PropName)
	})
}

func TestMockTypeQuerier_FindFieldType(t *testing.T) {
	t.Parallel()

	t.Run("nil FindFieldTypeFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindFieldType(&goast.Ident{Name: "T"}, "Name", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindFieldTypeFunc", func(t *testing.T) {
		t.Parallel()
		want := &goast.Ident{Name: "string"}
		m := &MockTypeQuerier{
			FindFieldTypeFunc: func(baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) goast.Expr {
				assert.Equal(t, "Title", fieldName)
				return want
			},
		}

		got := m.FindFieldType(&goast.Ident{Name: "Post"}, "Title", "pkg/b", "b.go")

		assert.Same(t, want, got)
	})
}

func TestMockTypeQuerier_FindFuncSignature(t *testing.T) {
	t.Parallel()

	t.Run("nil FindFuncSignatureFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindFuncSignature("fmt", "Println", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindFuncSignatureFunc", func(t *testing.T) {
		t.Parallel()
		want := &inspector_dto.FunctionSignature{
			Params:  []string{"string"},
			Results: []string{"int"},
		}
		m := &MockTypeQuerier{
			FindFuncSignatureFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
				assert.Equal(t, "strings", pkgAlias)
				assert.Equal(t, "Count", functionName)
				return want
			},
		}

		got := m.FindFuncSignature("strings", "Count", "pkg/c", "c.go")

		require.NotNil(t, got)
		assert.Equal(t, []string{"string"}, got.Params)
		assert.Equal(t, []string{"int"}, got.Results)
	})
}

func TestMockTypeQuerier_FindFuncReturnType(t *testing.T) {
	t.Parallel()

	t.Run("nil FindFuncReturnTypeFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindFuncReturnType("fmt", "Sprintf", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindFuncReturnTypeFunc", func(t *testing.T) {
		t.Parallel()
		want := &goast.Ident{Name: "string"}
		m := &MockTypeQuerier{
			FindFuncReturnTypeFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) goast.Expr {
				assert.Equal(t, "fmt", pkgAlias)
				assert.Equal(t, "Sprintf", functionName)
				return want
			},
		}

		got := m.FindFuncReturnType("fmt", "Sprintf", "pkg/d", "d.go")

		assert.Same(t, want, got)
	})
}

func TestMockTypeQuerier_FindFuncInfo(t *testing.T) {
	t.Parallel()

	t.Run("nil FindFuncInfoFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindFuncInfo("pkg", "DoWork", "pkg/a", "a.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindFuncInfoFunc", func(t *testing.T) {
		t.Parallel()
		want := &inspector_dto.Function{Name: "DoWork", TypeString: "func() error"}
		m := &MockTypeQuerier{
			FindFuncInfoFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.Function {
				assert.Equal(t, "DoWork", functionName)
				return want
			},
		}

		got := m.FindFuncInfo("myapp", "DoWork", "pkg/e", "e.go")

		require.NotNil(t, got)
		assert.Equal(t, "DoWork", got.Name)
	})
}

func TestMockTypeQuerier_FindMethodSignature(t *testing.T) {
	t.Parallel()

	t.Run("nil FindMethodSignatureFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindMethodSignature(&goast.Ident{Name: "T"}, "Read", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindMethodSignatureFunc", func(t *testing.T) {
		t.Parallel()
		want := &inspector_dto.FunctionSignature{
			Params:  []string{"[]byte"},
			Results: []string{"int", "error"},
		}
		m := &MockTypeQuerier{
			FindMethodSignatureFunc: func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
				assert.Equal(t, "Read", methodName)
				return want
			},
		}

		got := m.FindMethodSignature(&goast.Ident{Name: "Reader"}, "Read", "pkg/f", "f.go")

		require.NotNil(t, got)
		assert.Equal(t, []string{"[]byte"}, got.Params)
	})
}

func TestMockTypeQuerier_FindMethodReturnType(t *testing.T) {
	t.Parallel()

	t.Run("nil FindMethodReturnTypeFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindMethodReturnType(&goast.Ident{Name: "T"}, "Close", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindMethodReturnTypeFunc", func(t *testing.T) {
		t.Parallel()
		want := &goast.Ident{Name: "error"}
		m := &MockTypeQuerier{
			FindMethodReturnTypeFunc: func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) goast.Expr {
				assert.Equal(t, "Close", methodName)
				return want
			},
		}

		got := m.FindMethodReturnType(&goast.Ident{Name: "Conn"}, "Close", "pkg/g", "g.go")

		assert.Same(t, want, got)
	})
}

func TestMockTypeQuerier_FindMethodInfo(t *testing.T) {
	t.Parallel()

	t.Run("nil FindMethodInfoFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindMethodInfo(&goast.Ident{Name: "T"}, "String", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindMethodInfoFunc", func(t *testing.T) {
		t.Parallel()
		want := &inspector_dto.Method{Name: "String", TypeString: "func() string"}
		m := &MockTypeQuerier{
			FindMethodInfoFunc: func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.Method {
				assert.Equal(t, "String", methodName)
				return want
			},
		}

		got := m.FindMethodInfo(&goast.Ident{Name: "MyType"}, "String", "pkg/h", "h.go")

		require.NotNil(t, got)
		assert.Equal(t, "String", got.Name)
	})
}

func TestMockTypeQuerier_ResolvePackageAlias(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolvePackageAliasFunc returns empty string", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.ResolvePackageAlias("fmt", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to ResolvePackageAliasFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{
			ResolvePackageAliasFunc: func(aliasToResolve, importerPackagePath, importerFilePath string) string {
				assert.Equal(t, "fmt", aliasToResolve)
				assert.Equal(t, "pkg/i", importerPackagePath)
				return "fmt"
			},
		}

		got := m.ResolvePackageAlias("fmt", "pkg/i", "i.go")

		assert.Equal(t, "fmt", got)
	})
}

func TestMockTypeQuerier_FindPackageVariable(t *testing.T) {
	t.Parallel()

	t.Run("nil FindPackageVariableFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindPackageVariable("os", "Stdin", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindPackageVariableFunc", func(t *testing.T) {
		t.Parallel()
		want := &inspector_dto.Variable{Name: "Stdin", TypeString: "*os.File"}
		m := &MockTypeQuerier{
			FindPackageVariableFunc: func(pkgAlias, varName, importerPackagePath, importerFilePath string) *inspector_dto.Variable {
				assert.Equal(t, "os", pkgAlias)
				assert.Equal(t, "Stdin", varName)
				return want
			},
		}

		got := m.FindPackageVariable("os", "Stdin", "pkg/j", "j.go")

		require.NotNil(t, got)
		assert.Equal(t, "Stdin", got.Name)
	})
}

func TestMockTypeQuerier_FindPackageVariableType(t *testing.T) {
	t.Parallel()

	t.Run("nil FindPackageVariableTypeFunc returns empty string", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindPackageVariableType("os", "Args", "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindPackageVariableTypeFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{
			FindPackageVariableTypeFunc: func(pkgAlias, varName, importerPackagePath, importerFilePath string) string {
				assert.Equal(t, "os", pkgAlias)
				assert.Equal(t, "Args", varName)
				return "[]string"
			},
		}

		got := m.FindPackageVariableType("os", "Args", "pkg/k", "k.go")

		assert.Equal(t, "[]string", got)
	})
}

func TestMockTypeQuerier_GetAllPackages(t *testing.T) {
	t.Parallel()

	t.Run("nil GetAllPackagesFunc returns empty map", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.GetAllPackages()
		atomic.AddInt64(&calls, 1)

		require.NotNil(t, got)
		assert.Empty(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to GetAllPackagesFunc", func(t *testing.T) {
		t.Parallel()
		want := map[string]*inspector_dto.Package{
			"pkg/x": {Name: "x", Path: "pkg/x"},
		}
		m := &MockTypeQuerier{
			GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
				return want
			},
		}

		got := m.GetAllPackages()

		require.Len(t, got, 1)
		assert.Equal(t, "x", got["pkg/x"].Name)
	})
}

func TestMockTypeQuerier_FindRenderReturnType(t *testing.T) {
	t.Parallel()

	t.Run("nil FindRenderReturnTypeFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindRenderReturnType("pkg/comp", "comp.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindRenderReturnTypeFunc", func(t *testing.T) {
		t.Parallel()
		want := &goast.Ident{Name: "Node"}
		m := &MockTypeQuerier{
			FindRenderReturnTypeFunc: func(componentPackagePath, componentFilePath string) goast.Expr {
				assert.Equal(t, "pkg/comp", componentPackagePath)
				assert.Equal(t, "comp.go", componentFilePath)
				return want
			},
		}

		got := m.FindRenderReturnType("pkg/comp", "comp.go")

		assert.Same(t, want, got)
	})
}

func TestMockTypeQuerier_GetImportsForFile(t *testing.T) {
	t.Parallel()

	t.Run("nil GetImportsForFileFunc returns empty map", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.GetImportsForFile("pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		require.NotNil(t, got)
		assert.Empty(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to GetImportsForFileFunc", func(t *testing.T) {
		t.Parallel()
		want := map[string]string{"fmt": "fmt", "io": "io"}
		m := &MockTypeQuerier{
			GetImportsForFileFunc: func(importerPackagePath, importerFilePath string) map[string]string {
				assert.Equal(t, "pkg/l", importerPackagePath)
				return want
			},
		}

		got := m.GetImportsForFile("pkg/l", "l.go")

		assert.Equal(t, want, got)
	})
}

func TestMockTypeQuerier_FindPropsType(t *testing.T) {
	t.Parallel()

	t.Run("nil FindPropsTypeFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindPropsType("comp.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindPropsTypeFunc", func(t *testing.T) {
		t.Parallel()
		want := &goast.Ident{Name: "Props"}
		m := &MockTypeQuerier{
			FindPropsTypeFunc: func(filePath string) goast.Expr {
				assert.Equal(t, "button.go", filePath)
				return want
			},
		}

		got := m.FindPropsType("button.go")

		assert.Same(t, want, got)
	})
}

func TestMockTypeQuerier_GetAllFieldsAndMethods(t *testing.T) {
	t.Parallel()

	t.Run("nil GetAllFieldsAndMethodsFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.GetAllFieldsAndMethods(&goast.Ident{Name: "T"}, "pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to GetAllFieldsAndMethodsFunc", func(t *testing.T) {
		t.Parallel()
		want := []string{"Name", "Age", "String"}
		m := &MockTypeQuerier{
			GetAllFieldsAndMethodsFunc: func(baseType goast.Expr, importerPackagePath, importerFilePath string) []string {
				assert.Equal(t, "pkg/m", importerPackagePath)
				return want
			},
		}

		got := m.GetAllFieldsAndMethods(&goast.Ident{Name: "Person"}, "pkg/m", "m.go")

		assert.Equal(t, want, got)
	})
}

func TestMockTypeQuerier_FindFileWithImportAlias(t *testing.T) {
	t.Parallel()

	t.Run("nil FindFileWithImportAliasFunc returns empty string", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindFileWithImportAlias("pkg/n", "n", "pkg/n")
		atomic.AddInt64(&calls, 1)

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindFileWithImportAliasFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{
			FindFileWithImportAliasFunc: func(packagePath, alias, canonicalPath string) string {
				assert.Equal(t, "pkg/n", packagePath)
				assert.Equal(t, "n", alias)
				assert.Equal(t, "pkg/n", canonicalPath)
				return "/abs/n.go"
			},
		}

		got := m.FindFileWithImportAlias("pkg/n", "n", "pkg/n")

		assert.Equal(t, "/abs/n.go", got)
	})
}

func TestMockTypeQuerier_FindPackagePathForTypeDTO(t *testing.T) {
	t.Parallel()

	t.Run("nil FindPackagePathForTypeDTOFunc returns empty string", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.FindPackagePathForTypeDTO(&inspector_dto.Type{Name: "X"})
		atomic.AddInt64(&calls, 1)

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to FindPackagePathForTypeDTOFunc", func(t *testing.T) {
		t.Parallel()
		target := &inspector_dto.Type{Name: "Widget", PackagePath: "pkg/ui"}
		m := &MockTypeQuerier{
			FindPackagePathForTypeDTOFunc: func(tgt *inspector_dto.Type) string {
				assert.Same(t, target, tgt)
				return "pkg/ui"
			},
		}

		got := m.FindPackagePathForTypeDTO(target)

		assert.Equal(t, "pkg/ui", got)
	})
}

func TestMockTypeQuerier_Debug(t *testing.T) {
	t.Parallel()

	t.Run("nil DebugFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.Debug("pkg", "file.go")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to DebugFunc", func(t *testing.T) {
		t.Parallel()
		want := []string{"debug line 1", "debug line 2"}
		m := &MockTypeQuerier{
			DebugFunc: func(importerPackagePath, importerFilePath string) []string {
				assert.Equal(t, "pkg/o", importerPackagePath)
				return want
			},
		}

		got := m.Debug("pkg/o", "o.go")

		assert.Equal(t, want, got)
	})
}

func TestMockTypeQuerier_PackagePathForFile(t *testing.T) {
	t.Parallel()

	t.Run("nil PackagePathForFileFunc returns empty string", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.PackagePathForFile("file.go")
		atomic.AddInt64(&calls, 1)

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to PackagePathForFileFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{
			PackagePathForFileFunc: func(filePath string) string {
				assert.Equal(t, "/src/main.go", filePath)
				return "pkg/main"
			},
		}

		got := m.PackagePathForFile("/src/main.go")

		assert.Equal(t, "pkg/main", got)
	})
}

func TestMockTypeQuerier_GetFilesForPackage(t *testing.T) {
	t.Parallel()

	t.Run("nil GetFilesForPackageFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.GetFilesForPackage("pkg/p")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to GetFilesForPackageFunc", func(t *testing.T) {
		t.Parallel()
		want := []string{"a.go", "b.go", "c.go"}
		m := &MockTypeQuerier{
			GetFilesForPackageFunc: func(packagePath string) []string {
				assert.Equal(t, "pkg/p", packagePath)
				return want
			},
		}

		got := m.GetFilesForPackage("pkg/p")

		assert.Equal(t, want, got)
	})
}

func TestMockTypeQuerier_DebugDTO(t *testing.T) {
	t.Parallel()

	t.Run("nil DebugDTOFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.DebugDTO()
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to DebugDTOFunc", func(t *testing.T) {
		t.Parallel()
		want := map[string][]string{
			"pkg/q": {"type Foo", "func Bar"},
		}
		m := &MockTypeQuerier{
			DebugDTOFunc: func() map[string][]string {
				return want
			},
		}

		got := m.DebugDTO()

		assert.Equal(t, want, got)
	})
}

func TestMockTypeQuerier_DebugPackageDTO(t *testing.T) {
	t.Parallel()

	t.Run("nil DebugPackageDTOFunc returns nil", func(t *testing.T) {
		t.Parallel()
		m := &MockTypeQuerier{}
		var calls int64
		got := m.DebugPackageDTO("pkg/r")
		atomic.AddInt64(&calls, 1)

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
	})

	t.Run("delegates to DebugPackageDTOFunc", func(t *testing.T) {
		t.Parallel()
		want := []string{"type Alpha", "func Beta"}
		m := &MockTypeQuerier{
			DebugPackageDTOFunc: func(packagePath string) []string {
				assert.Equal(t, "pkg/r", packagePath)
				return want
			},
		}

		got := m.DebugPackageDTO("pkg/r")

		assert.Equal(t, want, got)
	})
}

func TestMockTypeQuerier_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockTypeQuerier{}
	identifier := &goast.Ident{Name: "T"}
	typeDTO := &inspector_dto.Type{Name: "X"}

	typ, name := m.ResolveExprToNamedType(identifier, "", "")
	assert.Nil(t, typ)
	assert.Equal(t, "", name)

	typ2, name2 := m.ResolveExprToNamedTypeWithMemoization(context.Background(), identifier, "", "")
	assert.Nil(t, typ2)
	assert.Equal(t, "", name2)

	got := m.ResolveToUnderlyingAST(identifier, "")
	assert.Same(t, identifier, got)

	gotExpr, gotFile := m.ResolveToUnderlyingASTWithContext(context.Background(), identifier, "")
	assert.Same(t, identifier, gotExpr)
	assert.Equal(t, "", gotFile)

	assert.Nil(t, m.GetAllSymbols())
	assert.Nil(t, m.GetAllFieldsAndMethods(identifier, "", ""))
	assert.Nil(t, m.GetFilesForPackage(""))
	assert.Nil(t, m.Debug("", ""))
	assert.Nil(t, m.DebugPackageDTO(""))

	assert.Nil(t, m.GetImplementationIndex())
	assert.Nil(t, m.GetTypeHierarchyIndex())
	assert.Nil(t, m.FindFieldInfo(context.Background(), identifier, "", "", ""))
	assert.Nil(t, m.FindFieldType(identifier, "", "", ""))
	assert.Nil(t, m.FindFuncSignature("", "", "", ""))
	assert.Nil(t, m.FindFuncReturnType("", "", "", ""))
	assert.Nil(t, m.FindFuncInfo("", "", "", ""))
	assert.Nil(t, m.FindMethodSignature(identifier, "", "", ""))
	assert.Nil(t, m.FindMethodReturnType(identifier, "", "", ""))
	assert.Nil(t, m.FindMethodInfo(identifier, "", "", ""))
	assert.Nil(t, m.FindPackageVariable("", "", "", ""))
	assert.Nil(t, m.FindRenderReturnType("", ""))
	assert.Nil(t, m.FindPropsType(""))

	assert.Equal(t, "", m.ResolvePackageAlias("", "", ""))
	assert.Equal(t, "", m.FindPackageVariableType("", "", "", ""))
	assert.Equal(t, "", m.FindFileWithImportAlias("", "", ""))
	assert.Equal(t, "", m.FindPackagePathForTypeDTO(typeDTO))
	assert.Equal(t, "", m.PackagePathForFile(""))

	allPackages := m.GetAllPackages()
	require.NotNil(t, allPackages)
	assert.Empty(t, allPackages)

	imports := m.GetImportsForFile("", "")
	require.NotNil(t, imports)
	assert.Empty(t, imports)

	assert.Nil(t, m.DebugDTO())
}

func TestMockTypeQuerier_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	identifier := &goast.Ident{Name: "Concurrent"}
	typeDTO := &inspector_dto.Type{Name: "C"}

	m := &MockTypeQuerier{
		ResolveExprToNamedTypeFunc: func(expression goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
			return &inspector_dto.Type{Name: "R"}, "R"
		},
		ResolveExprToNamedTypeWithMemoizationFunc: func(ctx context.Context, typeExpr goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
			return &inspector_dto.Type{Name: "RM"}, "RM"
		},
		ResolveToUnderlyingASTFunc: func(typeExpr goast.Expr, currentFilePath string) goast.Expr {
			return typeExpr
		},
		ResolveToUnderlyingASTWithContextFunc: func(ctx context.Context, typeExpr goast.Expr, currentFilePath string) (goast.Expr, string) {
			return typeExpr, "ctx"
		},
		GetAllSymbolsFunc: func() []inspector_dto.WorkspaceSymbol {
			return []inspector_dto.WorkspaceSymbol{{Name: "S"}}
		},
		GetImplementationIndexFunc: func() *ImplementationIndex {
			return nil
		},
		GetTypeHierarchyIndexFunc: func() *TypeHierarchyIndex {
			return nil
		},
		FindFieldInfoFunc: func(ctx context.Context, baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) *inspector_dto.FieldInfo {
			return &inspector_dto.FieldInfo{Name: fieldName}
		},
		FindFieldTypeFunc: func(baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) goast.Expr {
			return baseType
		},
		FindFuncSignatureFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
			return &inspector_dto.FunctionSignature{}
		},
		FindFuncReturnTypeFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) goast.Expr {
			return identifier
		},
		FindFuncInfoFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.Function {
			return &inspector_dto.Function{Name: functionName}
		},
		FindMethodSignatureFunc: func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
			return &inspector_dto.FunctionSignature{}
		},
		FindMethodReturnTypeFunc: func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) goast.Expr {
			return baseType
		},
		FindMethodInfoFunc: func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.Method {
			return &inspector_dto.Method{Name: methodName}
		},
		ResolvePackageAliasFunc: func(aliasToResolve, importerPackagePath, importerFilePath string) string {
			return aliasToResolve
		},
		FindPackageVariableFunc: func(pkgAlias, varName, importerPackagePath, importerFilePath string) *inspector_dto.Variable {
			return &inspector_dto.Variable{Name: varName}
		},
		FindPackageVariableTypeFunc: func(pkgAlias, varName, importerPackagePath, importerFilePath string) string {
			return "string"
		},
		GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
			return map[string]*inspector_dto.Package{"p": {Name: "p"}}
		},
		FindRenderReturnTypeFunc: func(componentPackagePath, componentFilePath string) goast.Expr {
			return identifier
		},
		GetImportsForFileFunc: func(importerPackagePath, importerFilePath string) map[string]string {
			return map[string]string{"fmt": "fmt"}
		},
		FindPropsTypeFunc: func(filePath string) goast.Expr {
			return identifier
		},
		GetAllFieldsAndMethodsFunc: func(baseType goast.Expr, importerPackagePath, importerFilePath string) []string {
			return []string{"field"}
		},
		FindFileWithImportAliasFunc: func(packagePath, alias, canonicalPath string) string {
			return "found.go"
		},
		FindPackagePathForTypeDTOFunc: func(target *inspector_dto.Type) string {
			return "pkg/found"
		},
		DebugFunc: func(importerPackagePath, importerFilePath string) []string {
			return []string{"dbg"}
		},
		PackagePathForFileFunc: func(filePath string) string {
			return "pkg/file"
		},
		GetFilesForPackageFunc: func(packagePath string) []string {
			return []string{"a.go"}
		},
		DebugDTOFunc: func() map[string][]string {
			return map[string][]string{"k": {"v"}}
		},
		DebugPackageDTOFunc: func(packagePath string) []string {
			return []string{"info"}
		},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			m.ResolveExprToNamedType(identifier, "p", "f")
			m.ResolveExprToNamedTypeWithMemoization(context.Background(), identifier, "p", "f")
			m.ResolveToUnderlyingAST(identifier, "f")
			m.ResolveToUnderlyingASTWithContext(context.Background(), identifier, "f")
			m.GetAllSymbols()
			m.GetImplementationIndex()
			m.GetTypeHierarchyIndex()
			m.FindFieldInfo(context.Background(), identifier, "n", "p", "f")
			m.FindFieldType(identifier, "n", "p", "f")
			m.FindFuncSignature("a", "fn", "p", "f")
			m.FindFuncReturnType("a", "fn", "p", "f")
			m.FindFuncInfo("a", "fn", "p", "f")
			m.FindMethodSignature(identifier, "m", "p", "f")
			m.FindMethodReturnType(identifier, "m", "p", "f")
			m.FindMethodInfo(identifier, "m", "p", "f")
			m.ResolvePackageAlias("a", "p", "f")
			m.FindPackageVariable("a", "v", "p", "f")
			m.FindPackageVariableType("a", "v", "p", "f")
			m.GetAllPackages()
			m.FindRenderReturnType("p", "f")
			m.GetImportsForFile("p", "f")
			m.FindPropsType("f")
			m.GetAllFieldsAndMethods(identifier, "p", "f")
			m.FindFileWithImportAlias("p", "a", "c")
			m.FindPackagePathForTypeDTO(typeDTO)
			m.Debug("p", "f")
			m.PackagePathForFile("f")
			m.GetFilesForPackage("p")
			m.DebugDTO()
			m.DebugPackageDTO("p")
		}()
	}

	wg.Wait()
}
