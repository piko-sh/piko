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
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func newTestResolver() *liteTypeResolver {
	reg := newTypeRegistry(minimalStdlib())
	resolver := newLiteTypeResolver(reg)
	resolver.SetContext("test/pkg", "/test.go", map[string]string{
		"time": "time",
		"http": "net/http",
		"fmt":  "fmt",
	})
	return resolver
}

func TestTypeExprToString(t *testing.T) {
	t.Parallel()
	resolver := newTestResolver()

	t.Run("ident", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.Ident{Name: "string"})
		require.NoError(t, err)
		assert.Equal(t, "string", result)
	})

	t.Run("selector expr", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.SelectorExpr{
			X:   &ast.Ident{Name: "time"},
			Sel: &ast.Ident{Name: "Time"},
		})
		require.NoError(t, err)
		assert.Equal(t, "time.Time", result)
	})

	t.Run("selector expr with non-ident base", func(t *testing.T) {
		t.Parallel()
		_, err := resolver.TypeExprToString(&ast.SelectorExpr{
			X:   &ast.StarExpr{X: &ast.Ident{Name: "foo"}},
			Sel: &ast.Ident{Name: "Bar"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected selector base")
	})

	t.Run("star expr", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.StarExpr{
			X: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "*int", result)
	})

	t.Run("array type (slice)", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.ArrayType{
			Elt: &ast.Ident{Name: "string"},
		})
		require.NoError(t, err)
		assert.Equal(t, "[]string", result)
	})

	t.Run("array type (fixed)", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.ArrayType{
			Len: &ast.BasicLit{Value: "3"},
			Elt: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "[...]int", result)
	})

	t.Run("map type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.MapType{
			Key:   &ast.Ident{Name: "string"},
			Value: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "map[string]int", result)
	})

	t.Run("map type with complex value", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.MapType{
			Key: &ast.Ident{Name: "string"},
			Value: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "time"},
				Sel: &ast.Ident{Name: "Time"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "map[string]time.Time", result)
	})

	t.Run("ellipsis", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.Ellipsis{
			Elt: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "...int", result)
	})

	t.Run("interface type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.InterfaceType{})
		require.NoError(t, err)
		assert.Equal(t, "interface{}", result)
	})

	t.Run("struct type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.StructType{})
		require.NoError(t, err)
		assert.Equal(t, "struct{}", result)
	})

	t.Run("func type no params or results", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.FuncType{})
		require.NoError(t, err)
		assert.Equal(t, "func()", result)
	})

	t.Run("func type with params", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "string"}},
					{Type: &ast.Ident{Name: "int"}},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "func(string, int)", result)
	})

	t.Run("func type with single result", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "error"}},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "func() error", result)
	})

	t.Run("func type with multiple results", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "string"}},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "int"}},
					{Type: &ast.Ident{Name: "error"}},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "func(string) (int, error)", result)
	})

	t.Run("chan type bidirectional", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.ChanType{
			Dir:   ast.SEND | ast.RECV,
			Value: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "chan int", result)
	})

	t.Run("chan type send only", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.ChanType{
			Dir:   ast.SEND,
			Value: &ast.Ident{Name: "string"},
		})
		require.NoError(t, err)
		assert.Equal(t, "chan<- string", result)
	})

	t.Run("chan type recv only", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.ChanType{
			Dir:   ast.RECV,
			Value: &ast.Ident{Name: "bool"},
		})
		require.NoError(t, err)
		assert.Equal(t, "<-chan bool", result)
	})

	t.Run("paren expr", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.TypeExprToString(&ast.ParenExpr{
			X: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "int", result)
	})

	t.Run("unsupported expr type", func(t *testing.T) {
		t.Parallel()
		_, err := resolver.TypeExprToString(&ast.BadExpr{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type expression")
	})
}

func TestResolveTypeExpr(t *testing.T) {
	t.Parallel()
	resolver := newTestResolver()

	t.Run("primitive type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.Ident{Name: "string"})
		require.NoError(t, err)
		assert.Equal(t, "string", result.TypeString)
		assert.True(t, result.IsUnderlyingPrimitive)
		assert.Equal(t, "", result.PackagePath)
	})

	t.Run("known internal type", func(t *testing.T) {
		t.Parallel()

		reg := newTypeRegistry(minimalStdlib())
		reg.RegisterPackage(&inspector_dto.Package{
			Name: "pkg",
			Path: "test/pkg",
			NamedTypes: map[string]*inspector_dto.Type{
				"MyType": {Name: "MyType"},
			},
		})
		r := newLiteTypeResolver(reg)
		r.SetContext("test/pkg", "/test.go", map[string]string{})

		result, err := r.ResolveTypeExpr(&ast.Ident{Name: "MyType"})
		require.NoError(t, err)
		assert.Equal(t, "MyType", result.TypeString)
		assert.True(t, result.IsInternalType)
		assert.Equal(t, "test/pkg", result.PackagePath)
	})

	t.Run("unknown ident falls through to current package", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.Ident{Name: "Unknown"})
		require.NoError(t, err)
		assert.Equal(t, "Unknown", result.TypeString)
		assert.True(t, result.IsInternalType)
		assert.Equal(t, "test/pkg", result.PackagePath)
	})

	t.Run("selector expr resolves imported type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.SelectorExpr{
			X:   &ast.Ident{Name: "time"},
			Sel: &ast.Ident{Name: "Time"},
		})
		require.NoError(t, err)
		assert.Equal(t, "time.Time", result.TypeString)
		assert.Equal(t, "time", result.PackagePath)
	})

	t.Run("selector expr with unknown alias", func(t *testing.T) {
		t.Parallel()
		_, err := resolver.ResolveTypeExpr(&ast.SelectorExpr{
			X:   &ast.Ident{Name: "unknown"},
			Sel: &ast.Ident{Name: "Type"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown package alias")
	})

	t.Run("pointer type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.StarExpr{
			X: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "*int", result.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypePointer, result.CompositeType)
		require.Len(t, result.CompositeParts, 1)
		assert.Equal(t, "element", result.CompositeParts[0].Role)
	})

	t.Run("slice type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.ArrayType{
			Elt: &ast.Ident{Name: "string"},
		})
		require.NoError(t, err)
		assert.Equal(t, "[]string", result.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypeSlice, result.CompositeType)
	})

	t.Run("array type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.ArrayType{
			Len: &ast.BasicLit{Value: "5"},
			Elt: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "[...]int", result.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypeArray, result.CompositeType)
	})

	t.Run("map type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.MapType{
			Key:   &ast.Ident{Name: "string"},
			Value: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "map[string]int", result.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypeMap, result.CompositeType)
		require.Len(t, result.CompositeParts, 2)
		assert.Equal(t, "key", result.CompositeParts[0].Role)
		assert.Equal(t, "value", result.CompositeParts[1].Role)
	})

	t.Run("ellipsis resolves as array", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.Ellipsis{
			Elt: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "[]int", result.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypeSlice, result.CompositeType)
	})

	t.Run("interface type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.InterfaceType{})
		require.NoError(t, err)
		assert.Equal(t, "interface{}", result.TypeString)
	})

	t.Run("struct type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.StructType{})
		require.NoError(t, err)
		assert.Equal(t, "struct{}", result.TypeString)
	})

	t.Run("func type", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "string"}},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "error"}},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "func(string) error", result.TypeString)
	})

	t.Run("chan type returns error", func(t *testing.T) {
		t.Parallel()
		_, err := resolver.ResolveTypeExpr(&ast.ChanType{
			Dir:   ast.SEND | ast.RECV,
			Value: &ast.Ident{Name: "int"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel types not supported")
	})

	t.Run("paren expr unwraps", func(t *testing.T) {
		t.Parallel()
		result, err := resolver.ResolveTypeExpr(&ast.ParenExpr{
			X: &ast.Ident{Name: "int"},
		})
		require.NoError(t, err)
		assert.Equal(t, "int", result.TypeString)
	})

	t.Run("unsupported expr", func(t *testing.T) {
		t.Parallel()
		_, err := resolver.ResolveTypeExpr(&ast.BadExpr{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type expression")
	})
}

func TestNewSimpleResolvedType(t *testing.T) {
	t.Parallel()
	result := newSimpleresolvedType("interface{}")
	assert.Equal(t, "interface{}", result.TypeString)
	assert.Equal(t, "interface{}", result.UnderlyingTypeString)
	assert.Equal(t, "", result.PackagePath)
	assert.Nil(t, result.CompositeParts)
	assert.False(t, result.IsInternalType)
	assert.False(t, result.IsUnderlyingPrimitive)
}
