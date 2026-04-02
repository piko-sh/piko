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

// The test file is in the `inspector_domain` package (not `_test`) to allow
// access to unexported functions via the public "Test Hook" variables.
package inspector_domain

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func setupTypesTest(t *testing.T, packagePath string) (types.Qualifier, func(name string) types.Type) {
	t.Helper()
	pkg, err := importer.Default().Import(packagePath)
	require.NoError(t, err, "Failed to import package for test setup: %s", packagePath)

	qualifier := func(p *types.Package) string {
		if p.Path() == "unsafe" {
			return "unsafe"
		}
		return p.Name()
	}

	lookup := func(name string) types.Type {
		obj := pkg.Scope().Lookup(name)
		require.NotNil(t, obj, "Type '%s' not found in package '%s'", name, packagePath)
		return obj.Type()
	}

	return qualifier, lookup
}

func setupGenericTypesTest(t *testing.T, src string) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", src, 0)
	require.NoError(t, err)

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("test/main", fset, []*ast.File{f}, nil)
	require.NoError(t, err, "Failed to type-check virtual package for test setup")
	require.NotNil(t, pkg)
	return pkg
}

func TestEncodeTypeName(t *testing.T) {
	t.Parallel()
	httpQualifier, httpLookup := setupTypesTest(t, "net/http")

	mainPackage := types.NewPackage("test/main", "main")
	mainQualifier := func(p *types.Package) string {
		if p.Path() == mainPackage.Path() {
			return p.Name()
		}
		return p.Name()
	}

	testCases := []struct {
		name      string
		inputType types.Type
		qualifier types.Qualifier
		expected  string
	}{

		{name: "Primitive Type - Integer", inputType: types.Typ[types.Int], qualifier: nil, expected: "int"},
		{name: "Composite Type - Pointer to Primitive", inputType: types.NewPointer(types.Typ[types.String]), qualifier: nil, expected: "*string"},
		{name: "Composite Type - Slice of Bytes", inputType: types.NewSlice(types.Typ[types.Byte]), qualifier: nil, expected: "[]uint8"},
		{name: "Composite Type - Fixed-Size Array of Runes", inputType: types.NewArray(types.Typ[types.Rune], 32), qualifier: nil, expected: "[32]int32"},
		{name: "Composite Type - Map of Primitives", inputType: types.NewMap(types.Typ[types.String], types.Typ[types.Bool]), qualifier: nil, expected: "map[string]bool"},
		{name: "Composite Type - Channel of Pointers to Primitives", inputType: types.NewChan(types.SendRecv, types.NewPointer(types.Typ[types.Int])), qualifier: nil, expected: "chan *int"},
		{name: "Composite Type - Pointer to a Map of Slices", inputType: types.NewPointer(types.NewMap(types.Typ[types.String], types.NewSlice(types.Typ[types.Int]))), qualifier: nil, expected: "*map[string][]int"},

		{name: "Named Type - Same Package Qualification", inputType: httpLookup("Request"), qualifier: httpQualifier, expected: "http.Request"},
		{name: "Named Type - Built-in Error", inputType: types.Universe.Lookup("error").Type(), qualifier: nil, expected: "error"},
		{name: "Named Type - unsafe.Pointer", inputType: types.Typ[types.UnsafePointer], qualifier: nil, expected: "unsafe.Pointer"},
		{name: "Named Type - Pointer to External Type", inputType: types.NewPointer(httpLookup("Request")), qualifier: httpQualifier, expected: "*http.Request"},

		{name: "Function Signature - No Params, No Results", inputType: types.NewSignatureType(nil, nil, nil, nil, nil, false), qualifier: nil, expected: "func()"},
		{name: "Function Signature - With Named Params and One Result", inputType: types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "p1", types.Typ[types.String]), types.NewVar(token.NoPos, nil, "p2", types.Typ[types.Int])), types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())), false), qualifier: nil, expected: "func(p1 string, p2 int) error"},
		{name: "Function Signature - With Multiple Results", inputType: types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]), types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false), qualifier: nil, expected: "func() (int, bool)"},
		{name: "Function Signature - Variadic Parameter", inputType: types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "format", types.Typ[types.String]), types.NewVar(token.NoPos, nil, "args", types.NewSlice(types.Universe.Lookup("any").Type()))), nil, true), qualifier: nil, expected: "func(format string, args ...any)"},

		{name: "Pathological - Anonymous Struct Field", inputType: types.NewStruct([]*types.Var{types.NewField(token.NoPos, mainPackage, "Name", types.Typ[types.String], false), types.NewField(token.NoPos, mainPackage, "Meta", types.NewStruct([]*types.Var{types.NewField(token.NoPos, mainPackage, "ID", types.Typ[types.Int], false)}, nil), false)}, nil), qualifier: mainQualifier, expected: "struct{Name string; Meta struct{ID int}}"},
		{name: "Pathological - Interface with Embedded `any` and Methods", inputType: types.NewInterfaceType([]*types.Func{types.NewFunc(token.NoPos, mainPackage, "Do", types.NewSignatureType(nil, nil, nil, nil, nil, false))}, []types.Type{types.Universe.Lookup("any").Type()}), qualifier: mainQualifier, expected: "interface{Do(); any}"},
		{name: "Pathological - Type Definition on `any`", inputType: types.NewNamed(types.NewTypeName(token.NoPos, mainPackage, "MyData", nil), types.Universe.Lookup("any").Type(), nil), qualifier: mainQualifier, expected: "main.MyData"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := encodeTypeName(tc.inputType, tc.qualifier)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestResolveUnderlyingType(t *testing.T) {
	t.Parallel()
	pkg := setupGenericTypesTest(t, `package main; type MyInt int; type UserID = string; type C = int; type B = C; type A = B`)

	testCases := []struct {
		inputType types.Type
		expected  types.Type
		name      string
	}{
		{name: "Alias Resolution - Resolve Underlying Type Definition", inputType: pkg.Scope().Lookup("MyInt").Type(), expected: types.Typ[types.Int]},
		{name: "Alias Resolution - Resolve Underlying Type Alias", inputType: pkg.Scope().Lookup("UserID").Type(), expected: types.Typ[types.String]},
		{name: "Alias Resolution - Resolve Chained Aliases", inputType: pkg.Scope().Lookup("A").Type(), expected: types.Typ[types.Int]},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := resolveUnderlyingType(tc.inputType)
			assert.True(t, types.Identical(tc.expected, actual), "Expected %s, got %s", tc.expected, actual)
		})
	}
}

func setupMultiPackageTest(t *testing.T, sources map[string]string) map[string]*types.Package {
	t.Helper()

	tempDir := t.TempDir()
	moduleName := "test"
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module "+moduleName+"\n\ngo 1.22\n"), 0644)
	require.NoError(t, err)

	overlay := make(map[string][]byte)

	patterns := make(map[string]bool)

	for path, content := range sources {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		overlay[fullPath] = []byte(content)

		pkgDir := filepath.Dir(path)
		if pkgDir == "." {
			patterns[moduleName] = true
		} else {

			patterns[moduleName+"/"+filepath.ToSlash(pkgDir)] = true
		}
	}

	loadPatterns := make([]string, 0, len(patterns))
	for p := range patterns {
		loadPatterns = append(loadPatterns, p)
	}

	config := &packages.Config{
		Mode:    packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:     tempDir,
		Overlay: overlay,
		Env:     os.Environ(),
		Tests:   false,
	}

	loadedPackages, err := packages.Load(config, loadPatterns...)
	require.NoError(t, err, "packages.Load failed")
	require.False(t, packages.PrintErrors(loadedPackages) > 0, "Found type-checking errors in virtual packages")

	result := make(map[string]*types.Package)
	for _, p := range loadedPackages {
		result[p.PkgPath] = p.Types
	}
	return result
}

func TestResolveAliasesWithinPackage(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"owner/main.go":    `package owner; import "test/external"; type InternalAlias = string; type Wrapper struct { IA InternalAlias; EA external.ExternalAlias }`,
		"external/defs.go": `package external; type ExternalAlias = bool`,
	}

	loadedPackages := setupMultiPackageTest(t, sources)

	ownerPackage := loadedPackages["test/owner"]
	externalPackage := loadedPackages["test/external"]
	require.NotNil(t, ownerPackage, "owner package not found")
	require.NotNil(t, externalPackage, "external package not found")

	internalAlias := ownerPackage.Scope().Lookup("InternalAlias").Type()
	externalAlias := externalPackage.Scope().Lookup("ExternalAlias").Type()
	wrapperUnderlying := ownerPackage.Scope().Lookup("Wrapper").Type().Underlying()
	wrapperStruct, ok := wrapperUnderlying.(*types.Struct)
	require.True(t, ok, "Wrapper underlying type should be *types.Struct")

	testCases := []struct {
		inputType        types.Type
		expectedType     types.Type
		name             string
		ownerPackagePath string
	}{
		{
			name:             "Alias Unwrapping - Unwrap Internal Alias",
			inputType:        internalAlias,
			ownerPackagePath: ownerPackage.Path(),
			expectedType:     types.Typ[types.String],
		},
		{
			name:             "Alias Preservation - Preserve External Alias",
			inputType:        externalAlias,
			ownerPackagePath: ownerPackage.Path(),
			expectedType:     externalAlias,
		},
		{
			name:             "Alias Preservation - Composite Type with Mixed Aliases",
			inputType:        wrapperStruct.Field(1).Type(),
			ownerPackagePath: ownerPackage.Path(),
			expectedType:     externalAlias,
		},
		{
			name:             "Alias Unwrapping - Composite Type with Mixed Aliases",
			inputType:        wrapperStruct.Field(0).Type(),
			ownerPackagePath: ownerPackage.Path(),
			expectedType:     types.Typ[types.String],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := resolveAliasesWithinPackage(tc.inputType, tc.ownerPackagePath)
			assert.True(t, types.Identical(tc.expectedType, actual), "Expected %s, got %s", tc.expectedType, actual)
		})
	}
}

func TestSubst(t *testing.T) {
	t.Parallel()
	pkg := setupGenericTypesTest(t, `package main; type Box[T any] struct{}; type Wrapper[T any] struct{};`)

	boxLookup := pkg.Scope().Lookup("Box").Type()
	boxType, ok := boxLookup.(*types.Named)
	require.True(t, ok, "Box should be *types.Named")
	wrapperLookup := pkg.Scope().Lookup("Wrapper").Type()
	wrapperType, ok := wrapperLookup.(*types.Named)
	require.True(t, ok, "Wrapper should be *types.Named")

	tParam := boxType.TypeParams().At(0)
	kParam := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "K", nil), types.NewInterfaceType(nil, nil))
	vParam := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "V", nil), types.NewInterfaceType(nil, nil))

	boxString, _ := types.Instantiate(nil, boxType, []types.Type{types.Typ[types.String]}, false)

	testCases := []struct {
		inputType types.Type
		expected  types.Type
		smap      map[*types.TypeParam]types.Type
		name      string
	}{
		{name: "Generic Substitution - Simple Type Parameter", inputType: tParam, smap: map[*types.TypeParam]types.Type{tParam: types.Typ[types.String]}, expected: types.Typ[types.String]},
		{name: "Generic Substitution - Composite Type", inputType: types.NewSlice(tParam), smap: map[*types.TypeParam]types.Type{tParam: types.Typ[types.Int]}, expected: types.NewSlice(types.Typ[types.Int])},
		{name: "Generic Substitution - Map with Two Parameters", inputType: types.NewMap(kParam, vParam), smap: map[*types.TypeParam]types.Type{kParam: types.Typ[types.String], vParam: types.Typ[types.Bool]}, expected: types.NewMap(types.Typ[types.String], types.Typ[types.Bool])},
		{name: "Generic Substitution - Nested Generic Type", inputType: wrapperType, smap: map[*types.TypeParam]types.Type{wrapperType.TypeParams().At(0): boxString}, expected: func() types.Type {
			t, _ := types.Instantiate(nil, wrapperType, []types.Type{boxString}, false)
			return t
		}()},
		{name: "Generic Substitution - Function Signature", inputType: types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", tParam)), types.NewTuple(types.NewVar(token.NoPos, nil, "", tParam)), false), smap: map[*types.TypeParam]types.Type{tParam: types.Typ[types.Bool]}, expected: types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)},
		{name: "Generic Substitution - Unused Type Parameter in Map", inputType: types.NewMap(types.Typ[types.String], vParam), smap: map[*types.TypeParam]types.Type{kParam: types.Typ[types.Int], vParam: types.Typ[types.Bool]}, expected: types.NewMap(types.Typ[types.String], types.Typ[types.Bool])},
		{name: "Generic Substitution - Free Type Parameter", inputType: types.NewSlice(tParam), smap: map[*types.TypeParam]types.Type{}, expected: types.NewSlice(tParam)},
		{name: "Generic Substitution - Substituting with `any`", inputType: types.NewSlice(tParam), smap: map[*types.TypeParam]types.Type{tParam: types.Universe.Lookup("any").Type()}, expected: types.NewSlice(types.Universe.Lookup("any").Type())},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := subst(tc.inputType, tc.smap)
			assert.True(t, types.Identical(tc.expected, actual), "Expected %s, got %s", tc.expected, actual)
		})
	}
}

func TestBaseNamedPackagePath(t *testing.T) {
	t.Parallel()
	_, httpLookup := setupTypesTest(t, "net/http")
	_, timeLookup := setupTypesTest(t, "time")

	genericPackage := setupGenericTypesTest(t, `package main; type Box[T any] struct{}`)
	boxLookup := genericPackage.Scope().Lookup("Box").Type()
	boxType, ok := boxLookup.(*types.Named)
	require.True(t, ok, "Box should be *types.Named")
	boxHttpReq, _ := types.Instantiate(nil, boxType, []types.Type{httpLookup("Request")}, false)

	testCases := []struct {
		name      string
		inputType types.Type
		expected  string
	}{
		{name: "Package Path - Simple Named Type", inputType: httpLookup("Request"), expected: "net/http"},
		{name: "Package Path - Composite Type", inputType: types.NewSlice(types.NewPointer(httpLookup("Cookie"))), expected: "net/http"},
		{name: "Package Path - Map Value Priority", inputType: types.NewMap(timeLookup("Time"), httpLookup("Request")), expected: "net/http"},
		{name: "Package Path - Map Key used if Value is primitive", inputType: types.NewMap(timeLookup("Time"), types.Typ[types.Int]), expected: "time"},
		{name: "Package Path - Generic with Named Type Argument", inputType: boxHttpReq, expected: "net/http"},
		{name: "Package Path - No Path for Primitives", inputType: types.NewMap(types.Typ[types.String], types.Typ[types.Int]), expected: ""},
		{name: "Package Path - Function with External Return Type", inputType: types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewPointer(httpLookup("Request")))), false), expected: "net/http"},
		{name: "Package Path - Function Returning `any`", inputType: types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("any").Type())), false), expected: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := baseNamedPackagePath(tc.inputType)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
