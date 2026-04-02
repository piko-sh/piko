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

package driver_symbols_extract

import (
	"go/parser"
	gotoken "go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateFileSlicesGenericDispatch(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "slices",
		ElementTypes: []string{"string", "int"},
	}

	pkgs, err := Extract([]string{"slices"}, map[string]PackageConfig{"slices": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.NotEmpty(t, pkg.GenericFuncs, "slices should have generic funcs")

	source, err := GenerateFile(pkg, "test_pkg", config)
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := gotoken.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, parseErr, "generated slices code should be valid Go:\n%s", src)

	require.Contains(t, src, "wrappedSlices")

	require.Contains(t, src, "[]string")
	require.Contains(t, src, "[]int")

	require.Contains(t, src, `"fmt"`)
}

func TestGenerateFileMapsGenericDispatch(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath: "maps",
		KeyTypes:   []string{"string"},
		ValueTypes: []string{"int"},
	}

	pkgs, err := Extract([]string{"maps"}, map[string]PackageConfig{"maps": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.NotEmpty(t, pkg.GenericFuncs, "maps should have generic funcs")

	source, err := GenerateFile(pkg, "test_pkg", config)
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := gotoken.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, parseErr, "generated maps code should be valid Go:\n%s", src)

	require.Contains(t, src, "map[string]int")
	require.Contains(t, src, "wrappedMaps")
}

func TestGenerateFileCmpScalarDispatch(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "cmp",
		ElementTypes: []string{"string", "int", "float64"},
	}

	pkgs, err := Extract([]string{"cmp"}, map[string]PackageConfig{"cmp": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.NotEmpty(t, pkg.GenericFuncs, "cmp should have generic funcs")

	source, err := GenerateFile(pkg, "test_pkg", config)
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := gotoken.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, parseErr, "generated cmp code should be valid Go:\n%s", src)

	require.Contains(t, src, "wrappedCmp")

	require.Contains(t, src, "string")
	require.Contains(t, src, "int")
	require.Contains(t, src, "float64")
}

func TestGenerateFileSlicesWithFuncParams(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "slices",
		ElementTypes: []string{"string"},
	}

	pkgs, err := Extract([]string{"slices"}, map[string]PackageConfig{"slices": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]

	var hasSortFunc bool
	for _, gf := range pkg.GenericFuncs {
		if gf.Name == "SortFunc" {
			hasSortFunc = true
			break
		}
	}
	require.True(t, hasSortFunc, "slices.SortFunc should be in GenericFuncs")

	source, err := GenerateFile(pkg, "test_pkg", config)
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := gotoken.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, parseErr, "generated code with func params should be valid Go:\n%s", src)

	require.Contains(t, src, "wrappedSlicesSortFunc")
}

func TestGenerateFileSlicesConcatVariadic(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "slices",
		ElementTypes: []string{"string", "int"},
	}

	pkgs, err := Extract([]string{"slices"}, map[string]PackageConfig{"slices": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]

	var hasConcat bool
	for _, gf := range pkg.GenericFuncs {
		if gf.Name == "Concat" {
			hasConcat = true
			break
		}
	}
	require.True(t, hasConcat, "slices.Concat should be in GenericFuncs")

	source, err := GenerateFile(pkg, "test_pkg", config)
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := gotoken.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, parseErr, "generated code with variadic params should be valid Go:\n%s", src)

	require.Contains(t, src, "wrappedSlicesConcat")
}

func TestGenerateFileEmptyGenericConfig(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "cmp",
		ElementTypes: nil,
	}

	pkgs, err := Extract([]string{"cmp"}, map[string]PackageConfig{"cmp": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]

	emptyGenConfig := PackageConfig{}
	source, err := GenerateFile(pkg, "test_pkg", emptyGenConfig)
	require.NoError(t, err)

	if source != nil {
		src := string(source)
		fset := gotoken.NewFileSet()
		_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
		require.NoError(t, parseErr, "generated code should be valid Go:\n%s", src)
	}
}

func TestGenerateFilePerFunctionOverrides(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "slices",
		ElementTypes: []string{"string", "int", "float64"},
		Functions: map[string]FunctionConfig{
			"Sort": {ElementTypes: []string{"string"}},
		},
	}

	pkgs, err := Extract([]string{"slices"}, map[string]PackageConfig{"slices": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]

	source, err := GenerateFile(pkg, "test_pkg", config)
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := gotoken.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, parseErr, "generated code with overrides should be valid Go:\n%s", src)
}

func TestExtractKeyValTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input     string
		expectKey string
		expectVal string
	}{
		{"map[string]int", "string", "int"},
		{"map[int]bool", "int", "bool"},
		{"map[string]float64", "string", "float64"},

		{"int", "", "int"},
		{"[]string", "", "[]string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			k, v := extractKeyValTypes(tt.input)
			require.Equal(t, tt.expectKey, k)
			require.Equal(t, tt.expectVal, v)
		})
	}
}

func TestParseTypeExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		expect string
	}{
		{"int", "int"},
		{"string", "string"},
		{"bool", "bool"},
		{"float64", "float64"},
		{"any", "any"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			expression := parseTypeExpr(tt.input)
			require.NotNil(t, expression)
		})
	}
}

func TestParseTypeExprSlice(t *testing.T) {
	t.Parallel()

	expression := parseTypeExpr("[]string")
	require.NotNil(t, expression)
}

func TestParseTypeExprMap(t *testing.T) {
	t.Parallel()

	expression := parseTypeExpr("map[string]int")
	require.NotNil(t, expression)
}

func TestParseTypeExprPointer(t *testing.T) {
	t.Parallel()

	expression := parseTypeExpr("*int")
	require.NotNil(t, expression)
}

func TestParseTypeExprQualified(t *testing.T) {
	t.Parallel()

	expression := parseTypeExpr("time.Duration")
	require.NotNil(t, expression)
}

func TestTitleCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		expect string
	}{
		{"", ""},
		{"a", "A"},
		{"slices", "Slices"},
		{"hello", "Hello"},
		{"ABC", "ABC"},
		{"cmp", "Cmp"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expect, titleCase(tt.input))
		})
	}
}

func TestSanitiseParamName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		index  int
		expect string
	}{
		{"x", 0, "x"},
		{"", 0, "p0"},
		{"_", 3, "p3"},
		{"slices", 0, "slicesArg"},
		{"maps", 1, "mapsArg"},
		{"cmp", 2, "cmpArg"},
		{"fmt", 0, "fmtArg"},
		{"reflect", 0, "reflectArg"},
		{"strings", 0, "stringsArg"},
		{"myParam", 5, "myParam"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.expect, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expect, sanitiseParamName(tt.name, tt.index))
		})
	}
}

func TestConcreteTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		typ    types.Type
		expect string
	}{
		{"Int", types.Typ[types.Int], "int"},
		{"String", types.Typ[types.String], "string"},
		{"Bool", types.Typ[types.Bool], "bool"},
		{"Float64", types.Typ[types.Float64], "float64"},
		{"Byte", types.Typ[types.Byte], "uint8"},
		{"SliceOfInt", types.NewSlice(types.Typ[types.Int]), "[]int"},
		{"SliceOfString", types.NewSlice(types.Typ[types.String]), "[]string"},
		{"MapStringInt", types.NewMap(types.Typ[types.String], types.Typ[types.Int]), "map[string]int"},
		{"PointerToInt", types.NewPointer(types.Typ[types.Int]), "*int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expect, concreteTypeName(tt.typ))
		})
	}
}

func TestSliceTypeExpr(t *testing.T) {
	t.Parallel()

	expression := sliceTypeExpr("int")
	require.NotNil(t, expression)
}

func TestMapTypeExpr(t *testing.T) {
	t.Parallel()

	expression := mapTypeExpr("string", "int")
	require.NotNil(t, expression)
}

func TestCoerceCallAST(t *testing.T) {
	t.Parallel()

	expression := coerceCallAST("string", "x")
	require.NotNil(t, expression)
}

func TestExtractSlicesGenericFuncs(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "slices",
		ElementTypes: []string{"string"},
	}

	pkgs, err := Extract([]string{"slices"}, map[string]PackageConfig{"slices": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.NotEmpty(t, pkg.GenericFuncs, "slices should have generic funcs")
	require.NotNil(t, pkg.TypesPackage, "slices should have TypesPackage populated")

	functionNames := make(map[string]bool)
	for _, gf := range pkg.GenericFuncs {
		functionNames[gf.Name] = true

		require.NotNil(t, gf.Signature, "signature for %s should not be nil", gf.Name)
	}
	require.True(t, functionNames["Sort"], "slices.Sort should be in GenericFuncs")
	require.True(t, functionNames["Contains"], "slices.Contains should be in GenericFuncs")
}

func TestExtractCmpGenericFuncs(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "cmp",
		ElementTypes: []string{"int"},
	}

	pkgs, err := Extract([]string{"cmp"}, map[string]PackageConfig{"cmp": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.NotEmpty(t, pkg.GenericFuncs, "cmp should have generic funcs")

	functionNames := make(map[string]bool)
	for _, gf := range pkg.GenericFuncs {
		functionNames[gf.Name] = true
	}
	require.True(t, functionNames["Compare"], "cmp.Compare should be in GenericFuncs")
}

func TestExtractMapsGenericFuncs(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath: "maps",
		KeyTypes:   []string{"string"},
		ValueTypes: []string{"int"},
	}

	pkgs, err := Extract([]string{"maps"}, map[string]PackageConfig{"maps": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.NotEmpty(t, pkg.GenericFuncs, "maps should have generic funcs")
}

func TestGenerateFileAllThreeDispatchKinds(t *testing.T) {
	t.Parallel()

	configs := map[string]PackageConfig{
		"slices": {ImportPath: "slices", ElementTypes: []string{"string"}},
		"maps":   {ImportPath: "maps", KeyTypes: []string{"string"}, ValueTypes: []string{"int"}},
		"cmp":    {ImportPath: "cmp", ElementTypes: []string{"int"}},
	}

	pkgs, err := Extract([]string{"slices", "maps", "cmp"}, configs)
	require.NoError(t, err)
	require.Len(t, pkgs, 3)

	for _, pkg := range pkgs {
		config := configs[pkg.ImportPath]
		source, genErr := GenerateFile(pkg, "test_pkg", config)
		require.NoError(t, genErr, "generating %s", pkg.ImportPath)
		if source != nil {
			fset := gotoken.NewFileSet()
			_, parseErr := parser.ParseFile(fset, "test.go", string(source), 0)
			require.NoError(t, parseErr, "generated %s code should be valid Go:\n%s", pkg.ImportPath, string(source))
		}
	}
}

func TestGenerateFileSlicesMultipleTypes(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "slices",
		ElementTypes: []string{"string", "int", "int64", "float64", "byte", "bool"},
	}

	pkgs, err := Extract([]string{"slices"}, map[string]PackageConfig{"slices": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	source, err := GenerateFile(pkg, "test_pkg", config)
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := gotoken.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, parseErr, "generated code should be valid Go:\n%s", src)

	for _, et := range config.ElementTypes {
		require.True(t, strings.Contains(src, et),
			"expected type %s to appear in generated code", et)
	}
}
