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

package goastutil

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachedIdent_StaticCacheHit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
	}{
		{name: "string"},
		{name: "int"},
		{name: "err"},
		{name: "nil"},
		{name: "true"},
		{name: "false"},
		{name: "append"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identifier := CachedIdent(tc.name)
			require.NotNil(t, identifier)
			assert.Equal(t, tc.name, identifier.Name)
			assert.Same(t, staticIdentCache[tc.name], identifier, "should return the static cache entry")
		})
	}
}

func TestCachedIdent_DynamicCacheHit(t *testing.T) {
	t.Parallel()

	name := "testDynCacheIdent_unique_42"
	dynamicIdentCache.Delete(name)

	first := CachedIdent(name)
	require.NotNil(t, first)
	assert.Equal(t, name, first.Name)

	second := CachedIdent(name)
	assert.Same(t, first, second, "repeat call should return the same pointer from dynamic cache")
}

func TestCachedIdent_NonCachedName(t *testing.T) {
	t.Parallel()

	name := "testNonCached_unique_99"
	dynamicIdentCache.Delete(name)

	identifier := CachedIdent(name)
	require.NotNil(t, identifier)
	assert.Equal(t, name, identifier.Name)
}

func TestRegisterIdent(t *testing.T) {
	t.Run("registers new ident", func(t *testing.T) {
		name := "testRegisterNew_unique_77"
		delete(staticIdentCache, name)

		RegisterIdent(name)

		cached, ok := staticIdentCache[name]
		require.True(t, ok)
		assert.Equal(t, name, cached.Name)

		delete(staticIdentCache, name)
	})

	t.Run("does not overwrite existing ident", func(t *testing.T) {
		existing := staticIdentCache["string"]
		require.NotNil(t, existing)

		RegisterIdent("string")

		assert.Same(t, existing, staticIdentCache["string"])
	})
}

func TestStrLit(t *testing.T) {
	t.Parallel()

	t.Run("empty string returns cached empty literal", func(t *testing.T) {
		t.Parallel()

		lit := StrLit("")
		require.NotNil(t, lit)
		assert.Equal(t, token.STRING, lit.Kind)
		assert.Equal(t, `""`, lit.Value)
		assert.Same(t, litEmptyString, lit)
	})

	t.Run("non-empty string returns correct quoted value", func(t *testing.T) {
		t.Parallel()

		lit := StrLit("hello")
		require.NotNil(t, lit)
		assert.Equal(t, token.STRING, lit.Kind)
		assert.Equal(t, `"hello"`, lit.Value)
	})

	t.Run("same string returns same pointer via cache", func(t *testing.T) {
		t.Parallel()

		first := StrLit("strlit_cache_test_unique")
		second := StrLit("strlit_cache_test_unique")
		assert.Same(t, first, second)
	})

	t.Run("different strings return different literals", func(t *testing.T) {
		t.Parallel()

		a := StrLit("alpha_strlit")
		b := StrLit("beta_strlit")
		assert.NotSame(t, a, b)
		assert.NotEqual(t, a.Value, b.Value)
	})
}

func TestIntLit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		input    int
		cached   bool
	}{
		{name: "zero returns cached literal", input: 0, expected: "0", cached: true},
		{name: "one returns cached literal", input: 1, expected: "1", cached: true},
		{name: "other value returns new literal", input: 42, expected: "42", cached: false},
		{name: "negative value", input: -7, expected: "-7", cached: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lit := IntLit(tc.input)
			require.NotNil(t, lit)
			assert.Equal(t, token.INT, lit.Kind)
			assert.Equal(t, tc.expected, lit.Value)

			if tc.cached {
				second := IntLit(tc.input)
				assert.Same(t, lit, second, "cached value should return the same pointer")
			}
		})
	}
}

func TestBoolIdent(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		identifier := BoolIdent(true)
		require.NotNil(t, identifier)
		assert.Equal(t, "true", identifier.Name)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		identifier := BoolIdent(false)
		require.NotNil(t, identifier)
		assert.Equal(t, "false", identifier.Name)
	})
}

func TestSelectorExpr(t *testing.T) {
	t.Parallel()

	selectorExpression := SelectorExpr("fmt", "Println")
	require.NotNil(t, selectorExpression)

	xIdent, ok := selectorExpression.X.(*ast.Ident)
	require.True(t, ok)
	assert.Equal(t, "fmt", xIdent.Name)
	assert.Equal(t, "Println", selectorExpression.Sel.Name)
}

func TestSelectorExprFrom(t *testing.T) {
	t.Parallel()

	x := CachedIdent("ctx")
	selectorExpression := SelectorExprFrom(x, "Value")
	require.NotNil(t, selectorExpression)
	assert.Same(t, x, selectorExpression.X)
	assert.Equal(t, "Value", selectorExpression.Sel.Name)
}

func TestCallExpr(t *testing.T) {
	t.Parallel()

	t.Run("no arguments", func(t *testing.T) {
		t.Parallel()

		call := CallExpr(CachedIdent("foo"))
		require.NotNil(t, call)
		assert.Empty(t, call.Args)
	})

	t.Run("single argument", func(t *testing.T) {
		t.Parallel()

		argument := CachedIdent("x")
		call := CallExpr(CachedIdent("foo"), argument)
		require.Len(t, call.Args, 1)
		assert.Same(t, argument, call.Args[0])
	})

	t.Run("multiple arguments", func(t *testing.T) {
		t.Parallel()

		a := CachedIdent("a")
		b := CachedIdent("b")
		call := CallExpr(CachedIdent("foo"), a, b)
		require.Len(t, call.Args, 2)
	})
}

func TestDefineStmt(t *testing.T) {
	t.Parallel()

	rightHandSide := CachedIdent("bar")
	statement := DefineStmt("foo", rightHandSide)
	require.NotNil(t, statement)
	assert.Equal(t, token.DEFINE, statement.Tok)
	require.Len(t, statement.Lhs, 1)

	leftHandSideIdent, ok := statement.Lhs[0].(*ast.Ident)
	require.True(t, ok)
	assert.Equal(t, "foo", leftHandSideIdent.Name)

	require.Len(t, statement.Rhs, 1)
	assert.Same(t, rightHandSide, statement.Rhs[0])
}

func TestDefineStmtMulti(t *testing.T) {
	t.Parallel()

	rightHandSide := CachedIdent("expression")
	statement := DefineStmtMulti([]string{"a", "b", "c"}, rightHandSide)
	require.NotNil(t, statement)
	assert.Equal(t, token.DEFINE, statement.Tok)
	require.Len(t, statement.Lhs, 3)

	for i, name := range []string{"a", "b", "c"} {
		identifier, ok := statement.Lhs[i].(*ast.Ident)
		require.True(t, ok)
		assert.Equal(t, name, identifier.Name)
	}

	require.Len(t, statement.Rhs, 1)
	assert.Same(t, rightHandSide, statement.Rhs[0])
}

func TestAssignStmt(t *testing.T) {
	t.Parallel()

	leftHandSide := CachedIdent("x")
	rightHandSide := CachedIdent("y")
	statement := AssignStmt(leftHandSide, rightHandSide)
	require.NotNil(t, statement)
	assert.Equal(t, token.ASSIGN, statement.Tok)
	require.Len(t, statement.Lhs, 1)
	assert.Same(t, leftHandSide, statement.Lhs[0])
	require.Len(t, statement.Rhs, 1)
	assert.Same(t, rightHandSide, statement.Rhs[0])
}

func TestReturnStmt(t *testing.T) {
	t.Parallel()

	t.Run("no results", func(t *testing.T) {
		t.Parallel()

		statement := ReturnStmt()
		require.NotNil(t, statement)
		assert.Empty(t, statement.Results)
	})

	t.Run("single result", func(t *testing.T) {
		t.Parallel()

		r := CachedIdent("nil")
		statement := ReturnStmt(r)
		require.Len(t, statement.Results, 1)
		assert.Same(t, r, statement.Results[0])
	})

	t.Run("multiple results", func(t *testing.T) {
		t.Parallel()

		a := CachedIdent("result")
		b := CachedIdent("nil")
		statement := ReturnStmt(a, b)
		require.Len(t, statement.Results, 2)
	})
}

func TestVarDecl(t *testing.T) {
	t.Parallel()

	typ := CachedIdent("string")
	declStmt := VarDecl("name", typ)
	require.NotNil(t, declStmt)

	genDecl, ok := declStmt.Decl.(*ast.GenDecl)
	require.True(t, ok)
	assert.Equal(t, token.VAR, genDecl.Tok)
	require.Len(t, genDecl.Specs, 1)

	valueSpec, ok := genDecl.Specs[0].(*ast.ValueSpec)
	require.True(t, ok)
	require.Len(t, valueSpec.Names, 1)
	assert.Equal(t, "name", valueSpec.Names[0].Name)
	assert.Same(t, typ, valueSpec.Type)
}

func TestCompositeLit(t *testing.T) {
	t.Parallel()

	t.Run("with type and elements", func(t *testing.T) {
		t.Parallel()

		typ := CachedIdent("MyStruct")
		elt := CachedIdent("val")
		lit := CompositeLit(typ, elt)
		require.NotNil(t, lit)
		assert.Same(t, typ, lit.Type)
		require.Len(t, lit.Elts, 1)
		assert.Same(t, elt, lit.Elts[0])
	})

	t.Run("nil type no elements", func(t *testing.T) {
		t.Parallel()

		lit := CompositeLit(nil)
		require.NotNil(t, lit)
		assert.Nil(t, lit.Type)
		assert.Empty(t, lit.Elts)
	})
}

func TestKeyValueExpr(t *testing.T) {
	t.Parallel()

	key := CachedIdent("key")
	value := CachedIdent("value")
	kv := KeyValueExpr(key, value)
	require.NotNil(t, kv)
	assert.Same(t, key, kv.Key)
	assert.Same(t, value, kv.Value)
}

func TestKeyValueIdent(t *testing.T) {
	t.Parallel()

	value := CachedIdent("value")
	kv := KeyValueIdent("Name", value)
	require.NotNil(t, kv)

	keyIdent, ok := kv.Key.(*ast.Ident)
	require.True(t, ok)
	assert.Equal(t, "Name", keyIdent.Name)
	assert.Same(t, value, kv.Value)
}

func TestStarExpr(t *testing.T) {
	t.Parallel()

	x := CachedIdent("int")
	star := StarExpr(x)
	require.NotNil(t, star)
	assert.Same(t, x, star.X)
}

func TestAddressExpr(t *testing.T) {
	t.Parallel()

	x := CachedIdent("myVar")
	addr := AddressExpr(x)
	require.NotNil(t, addr)
	assert.Equal(t, token.AND, addr.Op)
	assert.Same(t, x, addr.X)
}

func TestIndexExpr(t *testing.T) {
	t.Parallel()

	x := CachedIdent("arr")
	index := IntLit(0)
	indexExpr := IndexExpr(x, index)
	require.NotNil(t, indexExpr)
	assert.Same(t, x, indexExpr.X)
	assert.Same(t, index, indexExpr.Index)
}

func TestTypeAssertExpr(t *testing.T) {
	t.Parallel()

	x := CachedIdent("val")
	typ := CachedIdent("string")
	ta := TypeAssertExpr(x, typ)
	require.NotNil(t, ta)
	assert.Same(t, x, ta.X)
	assert.Same(t, typ, ta.Type)
}

func TestMapType(t *testing.T) {
	t.Parallel()

	key := CachedIdent("string")
	value := CachedIdent("int")
	m := MapType(key, value)
	require.NotNil(t, m)
	assert.Same(t, key, m.Key)
	assert.Same(t, value, m.Value)
}

func TestFuncType(t *testing.T) {
	t.Parallel()

	params := FieldList(Field("x", CachedIdent("int")))
	results := FieldList(Field("", CachedIdent("error")))
	ft := FuncType(params, results)
	require.NotNil(t, ft)
	assert.Same(t, params, ft.Params)
	assert.Same(t, results, ft.Results)
}

func TestFieldList(t *testing.T) {
	t.Parallel()

	f1 := Field("a", CachedIdent("int"))
	f2 := Field("b", CachedIdent("string"))
	fl := FieldList(f1, f2)
	require.NotNil(t, fl)
	require.Len(t, fl.List, 2)
	assert.Same(t, f1, fl.List[0])
	assert.Same(t, f2, fl.List[1])
}

func TestField(t *testing.T) {
	t.Parallel()

	t.Run("named field", func(t *testing.T) {
		t.Parallel()

		typ := CachedIdent("string")
		f := Field("name", typ)
		require.NotNil(t, f)
		require.Len(t, f.Names, 1)
		assert.Equal(t, "name", f.Names[0].Name)
		assert.Same(t, typ, f.Type)
	})

	t.Run("anonymous field has nil names", func(t *testing.T) {
		t.Parallel()

		typ := CachedIdent("io.Reader")
		f := Field("", typ)
		require.NotNil(t, f)
		assert.Nil(t, f.Names)
		assert.Same(t, typ, f.Type)
	})
}

func TestFuncLit(t *testing.T) {
	t.Parallel()

	ft := FuncType(nil, nil)
	body := BlockStmt()
	fl := FuncLit(ft, body)
	require.NotNil(t, fl)
	assert.Same(t, ft, fl.Type)
	assert.Same(t, body, fl.Body)
}

func TestBlockStmt(t *testing.T) {
	t.Parallel()

	t.Run("empty block", func(t *testing.T) {
		t.Parallel()

		block := BlockStmt()
		require.NotNil(t, block)
		assert.Empty(t, block.List)
	})

	t.Run("with statements", func(t *testing.T) {
		t.Parallel()

		s1 := ExprStmt(CachedIdent("x"))
		s2 := &ast.ReturnStmt{}
		block := BlockStmt(s1, s2)
		require.Len(t, block.List, 2)
		assert.Same(t, s1, block.List[0])
		assert.Same(t, s2, block.List[1])
	})
}

func TestExprStmt(t *testing.T) {
	t.Parallel()

	expression := CachedIdent("foo")
	statement := ExprStmt(expression)
	require.NotNil(t, statement)
	assert.Same(t, expression, statement.X)
}

func TestIfStmt(t *testing.T) {
	t.Parallel()

	t.Run("with init", func(t *testing.T) {
		t.Parallel()

		init := ExprStmt(CachedIdent("init"))
		cond := CachedIdent("ok")
		body := BlockStmt()
		statement := IfStmt(init, cond, body)
		require.NotNil(t, statement)
		assert.Same(t, init, statement.Init)
		assert.Same(t, cond, statement.Cond)
		assert.Same(t, body, statement.Body)
	})

	t.Run("without init", func(t *testing.T) {
		t.Parallel()

		cond := CachedIdent("ok")
		body := BlockStmt()
		statement := IfStmt(nil, cond, body)
		require.NotNil(t, statement)
		assert.Nil(t, statement.Init)
		assert.Same(t, cond, statement.Cond)
	})
}

func TestFuncDecl(t *testing.T) {
	t.Parallel()

	params := FieldList(Field("x", CachedIdent("int")))
	results := FieldList(Field("", CachedIdent("error")))
	body := BlockStmt(ReturnStmt(CachedIdent("nil")))

	declaration := FuncDecl("myFunc", params, results, body)
	require.NotNil(t, declaration)
	assert.Equal(t, "myFunc", declaration.Name.Name)
	assert.Same(t, params, declaration.Type.Params)
	assert.Same(t, results, declaration.Type.Results)
	assert.Same(t, body, declaration.Body)
}

func TestGenDeclType(t *testing.T) {
	t.Parallel()

	typ := CachedIdent("int")
	declaration := GenDeclType("MyInt", typ)
	require.NotNil(t, declaration)
	assert.Equal(t, token.TYPE, declaration.Tok)
	require.Len(t, declaration.Specs, 1)

	typeSpec, ok := declaration.Specs[0].(*ast.TypeSpec)
	require.True(t, ok)
	assert.Equal(t, "MyInt", typeSpec.Name.Name)
	assert.Same(t, typ, typeSpec.Type)
}

func TestStructType(t *testing.T) {
	t.Parallel()

	f1 := Field("Name", CachedIdent("string"))
	f2 := Field("Age", CachedIdent("int"))
	st := StructType(f1, f2)
	require.NotNil(t, st)
	require.NotNil(t, st.Fields)
	require.Len(t, st.Fields.List, 2)
	assert.Same(t, f1, st.Fields.List[0])
	assert.Same(t, f2, st.Fields.List[1])
}

func TestAddImport(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file := &ast.File{
		Name:  ast.NewIdent("main"),
		Decls: []ast.Decl{},
	}

	AddImport(fset, file, "fmt")

	found := false
	for _, imp := range file.Imports {
		if imp.Path.Value == `"fmt"` {
			found = true
			break
		}
	}
	assert.True(t, found, "expected import 'fmt' to be present")
}

func TestAddNamedImport(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file := &ast.File{
		Name:  ast.NewIdent("main"),
		Decls: []ast.Decl{},
	}

	AddNamedImport(fset, file, "myfmt", "fmt")

	found := false
	for _, imp := range file.Imports {
		if imp.Path.Value == `"fmt"` && imp.Name != nil && imp.Name.Name == "myfmt" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected named import 'myfmt \"fmt\"' to be present")
}

func TestFormatAST(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file := &ast.File{
		Name: ast.NewIdent("main"),
		Decls: []ast.Decl{
			FuncDecl("main", nil, nil, BlockStmt()),
		},
	}

	out, err := FormatAST(fset, file)
	require.NoError(t, err)
	require.NotEmpty(t, out)

	src := string(out)
	assert.True(t, strings.Contains(src, "package main"), "output should contain package declaration")
	assert.True(t, strings.Contains(src, "func main()"), "output should contain function declaration")
}
