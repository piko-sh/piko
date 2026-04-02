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
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildImportAliasMap_DefaultAlias(t *testing.T) {
	t.Parallel()

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

import "github.com/example/content_domain"

func main() { _ = content_domain.Foo }
`,
	})

	aliases := buildImportAliasMap(files)
	assert.Equal(t, "github.com/example/content_domain", aliases["content_domain"])
}

func TestBuildImportAliasMap_ExplicitAlias(t *testing.T) {
	t.Parallel()

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

import cd "github.com/example/content_domain"

func main() { _ = cd.Foo }
`,
	})

	aliases := buildImportAliasMap(files)
	assert.Equal(t, "github.com/example/content_domain", aliases["cd"])
	_, hasDefault := aliases["content_domain"]
	assert.False(t, hasDefault, "default alias should not be present when explicit alias is used")
}

func TestBuildImportAliasMap_BlankAndDotImports(t *testing.T) {
	t.Parallel()

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

import _ "github.com/example/sideeffect"
import . "github.com/example/dotpkg"
import "github.com/example/normalpkg"

func main() {}
`,
	})

	aliases := buildImportAliasMap(files)
	_, hasBlank := aliases["_"]
	_, hasDot := aliases["."]
	assert.False(t, hasBlank, "blank imports should be excluded")
	assert.False(t, hasDot, "dot imports should be excluded")
	assert.Equal(t, "github.com/example/normalpkg", aliases["normalpkg"])
}

func TestEnrichTypeCheckError_UndefinedInNativePackage(t *testing.T) {
	t.Parallel()

	service := NewService()
	service.RegisterPackage("github.com/example/content_domain", map[string]reflect.Value{
		"SomeFunc": reflect.ValueOf(func() string { return "" }),
	})

	origErr := types.Error{
		Fset: token.NewFileSet(),
		Msg:  "undefined: content_domain.AnnotatedField",
	}

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

import "github.com/example/content_domain"

func main() { _ = content_domain.AnnotatedField{} }
`,
	})

	enriched := service.enrichTypeCheckError(origErr, files, nil)
	assert.NotEqual(t, origErr.Error(), enriched.Error(), "error should be enriched")
	assert.Contains(t, enriched.Error(), `symbol "AnnotatedField" is not registered`)
	assert.Contains(t, enriched.Error(), `"github.com/example/content_domain"`)
	assert.Contains(t, enriched.Error(), `piko extract`)
}

func TestEnrichTypeCheckError_UndefinedInInterpretedPackage(t *testing.T) {
	t.Parallel()

	service := NewService()

	origErr := types.Error{
		Fset: token.NewFileSet(),
		Msg:  "undefined: mylib.MissingFunc",
	}

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

import "testmod/mylib"

func main() { mylib.MissingFunc() }
`,
	})

	interpretedPaths := map[string]bool{
		"testmod/mylib": true,
	}

	enriched := service.enrichTypeCheckError(origErr, files, interpretedPaths)
	assert.Equal(t, origErr.Error(), enriched.Error(), "interpreted packages should not get extract hint")
}

func TestEnrichTypeCheckError_NonUndefinedMessage(t *testing.T) {
	t.Parallel()

	service := NewService()
	service.RegisterPackage("github.com/example/pkg", map[string]reflect.Value{
		"Foo": reflect.ValueOf(func() {}),
	})

	origErr := types.Error{
		Fset: token.NewFileSet(),
		Msg:  "cannot use x (variable of type int) as string value",
	}

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

import "github.com/example/pkg"

func main() { pkg.Foo() }
`,
	})

	enriched := service.enrichTypeCheckError(origErr, files, nil)
	assert.Equal(t, origErr.Error(), enriched.Error(), "non-undefined errors should not be enriched")
}

func TestEnrichTypeCheckError_UndefinedLocalVariable(t *testing.T) {
	t.Parallel()

	service := NewService()

	origErr := types.Error{
		Fset: token.NewFileSet(),
		Msg:  "undefined: myVariable",
	}

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})

	enriched := service.enrichTypeCheckError(origErr, files, nil)
	assert.Equal(t, origErr.Error(), enriched.Error(), "local variable undefined errors should not be enriched")
}

func TestEnrichTypeCheckError_PackageNotInRegistry(t *testing.T) {
	t.Parallel()

	service := NewService()

	origErr := types.Error{
		Fset: token.NewFileSet(),
		Msg:  "undefined: unknownpkg.Foo",
	}

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

import "github.com/example/unknownpkg"

func main() { unknownpkg.Foo() }
`,
	})

	enriched := service.enrichTypeCheckError(origErr, files, nil)
	assert.Equal(t, origErr.Error(), enriched.Error(), "unregistered packages should not get extract hint")
}

func TestEnrichTypeCheckError_NilSymbols(t *testing.T) {
	t.Parallel()

	service := &Service{}

	origErr := types.Error{
		Fset: token.NewFileSet(),
		Msg:  "undefined: content_domain.AnnotatedField",
	}

	enriched := service.enrichTypeCheckError(origErr, nil, nil)
	assert.Equal(t, origErr.Error(), enriched.Error(), "nil symbols should not panic")
}

func TestEnrichTypeCheckError_NotTypesError(t *testing.T) {
	t.Parallel()

	service := NewService()

	origErr := errors.New("some random error")

	enriched := service.enrichTypeCheckError(origErr, nil, nil)
	assert.Equal(t, origErr.Error(), enriched.Error(), "non-types.Error should not be enriched")
}

func TestEnrichTypeCheckError_SymbolAlreadyRegistered(t *testing.T) {
	t.Parallel()

	service := NewService()
	service.RegisterPackage("github.com/example/content_domain", map[string]reflect.Value{
		"AnnotatedField": reflect.ValueOf((*struct{})(nil)),
	})

	origErr := types.Error{
		Fset: token.NewFileSet(),
		Msg:  "undefined: content_domain.AnnotatedField",
	}

	files := parseTestFiles(t, map[string]string{
		"main.go": `package main

import "github.com/example/content_domain"

func main() { _ = content_domain.AnnotatedField{} }
`,
	})

	enriched := service.enrichTypeCheckError(origErr, files, nil)
	assert.Equal(t, origErr.Error(), enriched.Error(),
		"should not add extract hint when the symbol IS registered")
}

func TestCompileProgram_StaleSymbolRegistryHint(t *testing.T) {
	t.Parallel()

	service := NewService()
	service.RegisterPackage("custom/content_domain", map[string]reflect.Value{
		"SomeFunc": reflect.ValueOf(func() string { return "hello" }),
	})

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "custom/content_domain"

func entrypoint() bool {
	return content_domain.MissingType{} != content_domain.MissingType{}
}

func main() {}
`,
		},
	}

	_, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.Error(t, err)
	require.ErrorIs(t, err, errTypeCheck)
	assert.Contains(t, err.Error(), `piko extract`,
		"error should contain extract hint for missing symbol in registered package")
	assert.Contains(t, err.Error(), `"MissingType" is not registered`)
}

func TestCompileProgram_InterpretedPackageMissingSymbol_NoHint(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/mylib"

func entrypoint() int {
	return mylib.Nonexistent()
}

func main() {}
`,
		},
		"mylib": {
			"mylib.go": `package mylib

func Add(a, b int) int { return a + b }
`,
		},
	}

	service := NewService()
	_, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.Error(t, err)
	require.ErrorIs(t, err, errTypeCheck)
	assert.NotContains(t, err.Error(), `piko extract`,
		"interpreted packages should not trigger extract hint")
}

func TestCompileFileSet_StaleSymbolRegistryHint(t *testing.T) {
	t.Parallel()

	service := NewService()
	service.RegisterPackage("custom/mypkg", map[string]reflect.Value{
		"Hello": reflect.ValueOf(func() string { return "world" }),
	})

	sources := map[string]string{
		"main.go": `package main

import "custom/mypkg"

func run() string {
	return mypkg.Missing()
}

func main() {}
`,
	}

	_, err := service.CompileFileSet(context.Background(), sources)
	require.Error(t, err)
	require.ErrorIs(t, err, errTypeCheck)
	assert.Contains(t, err.Error(), `piko extract`,
		"CompileFileSet should also enrich type check errors")
	assert.Contains(t, err.Error(), `"Missing" is not registered`)
}

func parseTestFiles(t *testing.T, sources map[string]string) []*ast.File {
	t.Helper()
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(sources))
	for name, src := range sources {
		file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		require.NoError(t, err, "parsing %s", name)
		files = append(files, file)
	}
	return files
}
