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

package goastutil_test

import (
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/goastutil"
)

func TestIsPrimitive(t *testing.T) {
	testCases := []struct {
		typ      types.Type
		name     string
		expected bool
	}{
		{name: "int", typ: types.Typ[types.Int], expected: true},
		{name: "int8", typ: types.Typ[types.Int8], expected: true},
		{name: "int16", typ: types.Typ[types.Int16], expected: true},
		{name: "int32", typ: types.Typ[types.Int32], expected: true},
		{name: "int64", typ: types.Typ[types.Int64], expected: true},
		{name: "uint", typ: types.Typ[types.Uint], expected: true},
		{name: "uint8", typ: types.Typ[types.Uint8], expected: true},
		{name: "uint16", typ: types.Typ[types.Uint16], expected: true},
		{name: "uint32", typ: types.Typ[types.Uint32], expected: true},
		{name: "uint64", typ: types.Typ[types.Uint64], expected: true},
		{name: "uintptr", typ: types.Typ[types.Uintptr], expected: true},
		{name: "float32", typ: types.Typ[types.Float32], expected: true},
		{name: "float64", typ: types.Typ[types.Float64], expected: true},
		{name: "complex64", typ: types.Typ[types.Complex64], expected: true},
		{name: "complex128", typ: types.Typ[types.Complex128], expected: true},
		{name: "bool", typ: types.Typ[types.Bool], expected: true},
		{name: "string", typ: types.Typ[types.String], expected: true},
		{name: "byte alias for uint8", typ: types.Typ[types.Byte], expected: true},
		{name: "rune alias for int32", typ: types.Typ[types.Rune], expected: true},
		{name: "empty interface", typ: types.NewInterfaceType(nil, nil), expected: true},
		{name: "nil", typ: nil, expected: false},
		{name: "function signature", typ: types.NewSignatureType(nil, nil, nil, nil, nil, false), expected: true},
		{name: "slice of functions", typ: types.NewSlice(types.NewSignatureType(nil, nil, nil, nil, nil, false)), expected: true},
		{name: "map with function value", typ: types.NewMap(types.Typ[types.String], types.NewSignatureType(nil, nil, nil, nil, nil, false)), expected: true},
		{name: "pointer to function", typ: types.NewPointer(types.NewSignatureType(nil, nil, nil, nil, nil, false)), expected: true},
		{name: "channel of functions", typ: types.NewChan(types.SendRecv, types.NewSignatureType(nil, nil, nil, nil, nil, false)), expected: true},
		{name: "array of functions", typ: types.NewArray(types.NewSignatureType(nil, nil, nil, nil, nil, false), 5), expected: true},
		{name: "slice of int", typ: types.NewSlice(types.Typ[types.Int]), expected: false},
		{name: "map of string to int", typ: types.NewMap(types.Typ[types.String], types.Typ[types.Int]), expected: false},
		{name: "pointer to int", typ: types.NewPointer(types.Typ[types.Int]), expected: false},
		{name: "channel of int", typ: types.NewChan(types.SendRecv, types.Typ[types.Int]), expected: false},
		{name: "array of int", typ: types.NewArray(types.Typ[types.Int], 5), expected: false},
		{name: "interface with methods", typ: createInterfaceWithMethods(), expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := goastutil.IsPrimitive(tc.typ)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPrimitive_NamedTypes(t *testing.T) {
	pkg := types.NewPackage("example.com/test", "test")

	t.Run("Named type with package is not primitive", func(t *testing.T) {
		namedType := types.NewNamed(
			types.NewTypeName(0, pkg, "MyInt", nil),
			types.Typ[types.Int],
			nil,
		)
		assert.False(t, goastutil.IsPrimitive(namedType))
	})

	t.Run("Named type without package (builtin error) is not primitive due to methods", func(t *testing.T) {
		errorType := types.Universe.Lookup("error").Type()
		assert.False(t, goastutil.IsPrimitive(errorType))
	})

	t.Run("Named type without package (builtin any) is primitive", func(t *testing.T) {
		anyType := types.Universe.Lookup("any").Type()
		assert.True(t, goastutil.IsPrimitive(anyType))
	})
}

func TestIsPrimitive_TypeParam(t *testing.T) {
	t.Run("Type parameter is not primitive", func(t *testing.T) {
		typeParam := types.NewTypeParam(
			types.NewTypeName(0, nil, "T", nil),
			types.NewInterfaceType(nil, nil),
		)
		assert.False(t, goastutil.IsPrimitive(typeParam))
	})
}

func TestIsPrimitive_MapWithFunctionKey(t *testing.T) {
	t.Run("Map with function key is primitive", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		mapType := types.NewMap(funcType, types.Typ[types.String])
		assert.True(t, goastutil.IsPrimitive(mapType))
	})
}

func TestIsPrimitive_NestedComposites(t *testing.T) {
	t.Run("Nested slice containing function", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		innerSlice := types.NewSlice(funcType)
		outerSlice := types.NewSlice(innerSlice)
		assert.True(t, goastutil.IsPrimitive(outerSlice))
	})

	t.Run("Map containing slice of functions", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		sliceOfFunc := types.NewSlice(funcType)
		mapType := types.NewMap(types.Typ[types.String], sliceOfFunc)
		assert.True(t, goastutil.IsPrimitive(mapType))
	})

	t.Run("Pointer to pointer to function", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		ptrToFunc := types.NewPointer(funcType)
		ptrToPtrToFunc := types.NewPointer(ptrToFunc)
		assert.True(t, goastutil.IsPrimitive(ptrToPtrToFunc))
	})

	t.Run("Channel of channel of functions", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		chanOfFunc := types.NewChan(types.SendRecv, funcType)
		chanOfChanOfFunc := types.NewChan(types.SendRecv, chanOfFunc)
		assert.True(t, goastutil.IsPrimitive(chanOfChanOfFunc))
	})

	t.Run("Array of pointers to functions", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		ptrToFunc := types.NewPointer(funcType)
		arrayOfPtrToFunc := types.NewArray(ptrToFunc, 10)
		assert.True(t, goastutil.IsPrimitive(arrayOfPtrToFunc))
	})
}

func createInterfaceWithMethods() *types.Interface {
	sig := types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])), false)
	method := types.NewFunc(0, nil, "String", sig)

	iface := types.NewInterfaceType([]*types.Func{method}, nil)
	iface.Complete()
	return iface
}

func findType(t *testing.T, pkg *packages.Package, name string) types.Type {
	t.Helper()
	obj := pkg.Types.Scope().Lookup(name)
	require.NotNil(t, obj, "type '%s' not found in package '%s'", name, pkg.Name)
	return obj.Type()
}

func findPackageByName(t *testing.T, pkgs []*packages.Package, name string) *packages.Package {
	t.Helper()
	for _, p := range pkgs {
		if p.Name == name {
			return p
		}
	}
	require.Fail(t, "package with name '%s' not found in loaded packages", name)
	return nil
}

func TestIsGoInternalType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test that requires package loading in short mode")
	}

	tempDir := t.TempDir()

	goModContent := `module testmodule

go 1.19

require github.com/google/uuid v1.6.0
`
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	const sourceCode = `package main

import (
	"context"
	"strings"
	"unsafe"
	
	"github.com/google/uuid"
)

type MyStruct struct{ ID int }
type MyContext = context.Context
type MyInt int
type MyFunc func()

var (
	_ MyStruct
	_ MyContext
	_ MyInt
	_ MyFunc
	_ strings.Builder
	_ error
	_ unsafe.Pointer
	_ int
	_ map[string]bool
	_ uuid.UUID
)
`
	err = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(sourceCode), 0644)
	require.NoError(t, err)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyOutput, err := tidyCmd.CombinedOutput()
	require.NoError(t, err, "go mod tidy failed: %s", string(tidyOutput))

	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedModule | packages.NeedImports,
		Dir:  tempDir,
	}
	loadedPackages, err := packages.Load(config, "./...")
	require.NoError(t, err, "failed to load packages")
	require.False(t, packages.PrintErrors(loadedPackages) > 0, "packages.Load returned errors")
	require.NotEmpty(t, loadedPackages, "no packages were loaded")

	allPackages := make(map[string]*packages.Package)
	packages.Visit(loadedPackages, func(p *packages.Package) bool {
		allPackages[p.PkgPath] = p
		return true
	}, nil)

	mainPackage := findPackageByName(t, loadedPackages, "main")

	contextPackage := allPackages["context"]
	stringsPackage := allPackages["strings"]
	unsafePackage := allPackages["unsafe"]
	uuidPackage := allPackages["github.com/google/uuid"]

	require.NotNil(t, mainPackage, "'main' package not found")
	require.NotNil(t, contextPackage, "'context' package not found")
	require.NotNil(t, stringsPackage, "'strings' package not found")
	require.NotNil(t, unsafePackage, "'unsafe' package not found")
	require.NotNil(t, uuidPackage, "'uuid' package not found")

	testCases := []struct {
		typ      types.Type
		name     string
		expected bool
	}{
		{name: "Local User-defined Type", typ: findType(t, mainPackage, "MyStruct"), expected: false},
		{name: "Pointer to Local User-defined Type", typ: types.NewPointer(findType(t, mainPackage, "MyStruct")), expected: false},
		{name: "Standard Library Type", typ: findType(t, contextPackage, "Context"), expected: true},
		{name: "Standard Library Other Type", typ: findType(t, stringsPackage, "Builder"), expected: true},
		{name: "Unsafe Pointer", typ: findType(t, unsafePackage, "Pointer"), expected: true},
		{name: "Pre-declared: error", typ: types.Universe.Lookup("error").Type(), expected: true},
		{name: "Pre-declared: any", typ: types.Universe.Lookup("any").Type(), expected: true},
		{name: "Primitive: int", typ: types.Universe.Lookup("int").Type(), expected: true},
		{name: "Local Alias of Standard Library Type", typ: findType(t, mainPackage, "MyContext"), expected: true},
		{name: "Local Type Definition on Primitive", typ: findType(t, mainPackage, "MyInt"), expected: false},
		{name: "Third-Party Dependency Type", typ: findType(t, uuidPackage, "UUID"), expected: false},
		{name: "Composite: Slice of Local Type", typ: types.NewSlice(findType(t, mainPackage, "MyStruct")), expected: false},
		{name: "Composite: Slice of Standard Library Type", typ: types.NewSlice(findType(t, contextPackage, "Context")), expected: true},
		{name: "Composite: Map with Third-Party and Local Types", typ: types.NewMap(findType(t, uuidPackage, "UUID"), findType(t, mainPackage, "MyStruct")), expected: false},
		{name: "Function Type", typ: findType(t, mainPackage, "MyFunc"), expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := goastutil.IsGoInternalType(tc.typ, allPackages)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
