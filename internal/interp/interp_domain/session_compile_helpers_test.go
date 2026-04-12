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

package interp_domain

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func parseFirstDecl(t *testing.T, source string) ast.Decl {
	t.Helper()
	wrapped := "package main\n" + source
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", wrapped, parser.ParseComments)
	require.NoError(t, err)
	require.NotEmpty(t, file.Decls, "test source has no top-level declarations")
	return file.Decls[0]
}

func TestDeclNamesIn_FuncDeclSingle(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `func foo() {}`)
	got := declNamesIn(decl)
	require.Equal(t, []SessionDecl{{Name: "foo", Kind: SessionDeclFunc}}, got)
}

func TestDeclNamesIn_InitFuncReturnsNil(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `func init() {}`)
	require.Nil(t, declNamesIn(decl), "init must not be reported as a named declaration")
}

func TestDeclNamesIn_VarSingle(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `var x int`)
	require.Equal(t, []SessionDecl{{Name: "x", Kind: SessionDeclVar}}, declNamesIn(decl))
}

func TestDeclNamesIn_VarBlock(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `var (
	x int
	y string
)`)
	require.Equal(t,
		[]SessionDecl{
			{Name: "x", Kind: SessionDeclVar},
			{Name: "y", Kind: SessionDeclVar},
		},
		declNamesIn(decl),
	)
}

func TestDeclNamesIn_ConstBlock(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `const a = 1`)
	require.Equal(t, []SessionDecl{{Name: "a", Kind: SessionDeclConst}}, declNamesIn(decl))
}

func TestDeclNamesIn_TypeDecl(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `type Point struct { X, Y int }`)
	require.Equal(t, []SessionDecl{{Name: "Point", Kind: SessionDeclType}}, declNamesIn(decl))
}

func TestDeclNamesIn_BlankIdentSkipped(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `var _ = 42`)
	require.Empty(t, declNamesIn(decl), "blank ident var must not be reported")
}

func TestDeclNamesIn_ImportReturnsNil(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `import "fmt"`)
	require.Nil(t, declNamesIn(decl), "imports should produce no SessionDecl entries")
}

func TestDeclNamesIn_BadDeclReturnsNil(t *testing.T) {
	t.Parallel()
	var bad ast.Decl = &ast.BadDecl{}
	require.Nil(t, declNamesIn(bad))
}

func TestIsInitFuncDecl_PositiveAndNegatives(t *testing.T) {
	t.Parallel()

	init := parseFirstDecl(t, `func init() {}`)
	require.True(t, isInitFuncDecl(init))

	notInit := parseFirstDecl(t, `func foo() {}`)
	require.False(t, isInitFuncDecl(notInit))

	notFunc := parseFirstDecl(t, `var x int`)
	require.False(t, isInitFuncDecl(notFunc))

	require.False(t, isInitFuncDecl(&ast.BadDecl{}),
		"bad decl with no name must not be classified as init")
}

func TestInitDeclSignature_StableFingerprint(t *testing.T) {
	t.Parallel()
	first := parseFirstDecl(t, `func init() { _ = 1 }`)
	second := parseFirstDecl(t, `func init() { _ = 1 }`)
	require.Equal(t, initDeclSignature(first), initDeclSignature(second))
}

func TestInitDeclSignature_DifferentBodiesDifferentSig(t *testing.T) {
	t.Parallel()
	a := parseFirstDecl(t, `func init() { _ = 1 }`)
	b := parseFirstDecl(t, `func init() { _ = 2 }`)
	require.NotEqual(t, initDeclSignature(a), initDeclSignature(b))
}

func TestInitDeclSignature_NonInitReturnsEmpty(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `func foo() {}`)
	require.Empty(t, initDeclSignature(decl))
}

func TestImportPathsIn_SingleImport(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `import "fmt"`)
	require.Equal(t, []string{"fmt"}, importPathsIn(decl))
}

func TestImportPathsIn_ImportBlock(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `import (
	"fmt"
	"strings"
)`)
	got := importPathsIn(decl)
	require.Equal(t, []string{"fmt", "strings"}, got)
}

func TestImportPathsIn_NonImportReturnsNil(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `var x int`)
	require.Nil(t, importPathsIn(decl))
}

func TestImportPathsIn_FuncDeclReturnsNil(t *testing.T) {
	t.Parallel()
	decl := parseFirstDecl(t, `func foo() {}`)
	require.Nil(t, importPathsIn(decl))
}

func TestImportPathsIn_BacktickPathIsStripped(t *testing.T) {
	t.Parallel()
	wrapped := "package main\nimport `fmt`\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", wrapped, 0)
	require.NoError(t, err)
	got := importPathsIn(file.Decls[0])
	require.Equal(t, []string{"fmt"}, got)
}

func TestFuncDeclNames_AnonymousReturnsNil(t *testing.T) {
	t.Parallel()
	decl := &ast.FuncDecl{Name: nil}
	require.Nil(t, funcDeclNames(decl))
}

func TestValueSpecNames_VarVsConst(t *testing.T) {
	t.Parallel()
	spec := &ast.ValueSpec{
		Names: []*ast.Ident{{Name: "a"}, {Name: "b"}},
	}

	varNames := valueSpecNames(token.VAR, spec)
	require.Equal(t,
		[]SessionDecl{
			{Name: "a", Kind: SessionDeclVar},
			{Name: "b", Kind: SessionDeclVar},
		},
		varNames,
	)

	constNames := valueSpecNames(token.CONST, spec)
	require.Equal(t,
		[]SessionDecl{
			{Name: "a", Kind: SessionDeclConst},
			{Name: "b", Kind: SessionDeclConst},
		},
		constNames,
	)
}

func TestValueSpecNames_BlankSkipped(t *testing.T) {
	t.Parallel()
	spec := &ast.ValueSpec{
		Names: []*ast.Ident{{Name: "_"}, {Name: "x"}, nil, {Name: "_"}},
	}
	got := valueSpecNames(token.VAR, spec)
	require.Equal(t, []SessionDecl{{Name: "x", Kind: SessionDeclVar}}, got)
}

func TestInitDeclSignature_NonInitWithEmptyBody(t *testing.T) {
	t.Parallel()
	decl := &ast.FuncDecl{Name: &ast.Ident{Name: "init2"}}
	got := initDeclSignature(decl)
	require.Empty(t, got, "non-init name must produce empty signature")
	require.False(t, strings.HasPrefix(got, "func"))
}
