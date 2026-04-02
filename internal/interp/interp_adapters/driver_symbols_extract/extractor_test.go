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
	"go/constant"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractMathPackage(t *testing.T) {
	t.Parallel()

	pkgs, err := Extract([]string{"math"}, nil)
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.Equal(t, "math", pkg.ImportPath)
	require.Equal(t, "math", pkg.Name)

	symMap := make(map[string]ExtractedSymbol)
	for _, sym := range pkg.Symbols {
		symMap[sym.Name] = sym
	}

	pi, ok := symMap["Pi"]
	require.True(t, ok, "math.Pi should be extracted")
	require.True(t, pi.IsUntypedConst, "math.Pi should be untyped")
	require.Equal(t, SymbolConst, pi.Kind)

	sqrt, ok := symMap["Sqrt"]
	require.True(t, ok, "math.Sqrt should be extracted")
	require.Equal(t, SymbolFunc, sqrt.Kind)

	_, ok = symMap["MaxFloat64"]
	require.True(t, ok, "math.MaxFloat64 should be extracted")
}

func TestExtractOsPackage(t *testing.T) {
	t.Parallel()

	pkgs, err := Extract([]string{"os"}, nil)
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]

	symMap := make(map[string]ExtractedSymbol)
	for _, sym := range pkg.Symbols {
		symMap[sym.Name] = sym
	}

	stdin, ok := symMap["Stdin"]
	require.True(t, ok, "os.Stdin should be extracted")
	require.Equal(t, SymbolVar, stdin.Kind)

	file, ok := symMap["File"]
	require.True(t, ok, "os.File should be extracted")
	require.Equal(t, SymbolType, file.Kind)

	exit, ok := symMap["Exit"]
	require.True(t, ok, "os.Exit should be extracted")
	require.Equal(t, SymbolFunc, exit.Kind)
}

func TestExtractInvalidPackage(t *testing.T) {
	t.Parallel()

	_, err := Extract([]string{"nonexistent/fake/package"}, nil)
	require.Error(t, err)
}

func TestExtractMultiplePackages(t *testing.T) {
	t.Parallel()

	pkgs, err := Extract([]string{"fmt", "strings", "math"}, nil)
	require.NoError(t, err)
	require.Len(t, pkgs, 3)

	paths := make(map[string]bool)
	for _, pkg := range pkgs {
		paths[pkg.ImportPath] = true
	}
	require.True(t, paths["fmt"])
	require.True(t, paths["strings"])
	require.True(t, paths["math"])
}

func TestFormatConstantLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  constant.Value
		expect string
	}{
		{"Int", constant.MakeInt64(42), "42"},
		{"String", constant.MakeString("hello"), `"hello"`},
		{"Zero", constant.MakeInt64(0), "0"},
		{"Bool", constant.MakeBool(true), "true"},
		{"NegativeInt", constant.MakeInt64(-7), "-7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expect, FormatConstantLiteral(tt.input))
		})
	}
}

func TestGenerateFileConstSymbol(t *testing.T) {
	t.Parallel()

	pkg := ExtractedPackage{
		ImportPath: "math",
		Name:       "math",
		Symbols: []ExtractedSymbol{
			{Name: "MaxFloat64", Kind: SymbolConst, ConstValue: "math.MaxFloat64"},
			{Name: "Sqrt", Kind: SymbolFunc},
		},
	}

	source, err := GenerateFile(pkg, "test_pkg", PackageConfig{})
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, parseErr, "generated code should be valid Go:\n%s", src)

	require.Contains(t, src, `reflect.ValueOf(math.MaxFloat64)`)
}

func TestImportAliasReflectCollision(t *testing.T) {
	t.Parallel()

	pkg := ExtractedPackage{
		ImportPath: "some/reflect",
		Name:       "reflect",
		Symbols: []ExtractedSymbol{
			{Name: "Something", Kind: SymbolFunc},
		},
	}

	source, err := GenerateFile(pkg, "test_pkg", PackageConfig{})
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	require.Contains(t, src, `_reflect "some/reflect"`)
}

func TestGenerateTypesLoaderFile(t *testing.T) {
	t.Parallel()

	source, err := GenerateTypesLoaderFile([]string{"slices", "maps"}, "test_pkg")
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	require.Contains(t, src, "//go:build !js || !wasm")

	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	require.NoError(t, parseErr, "types loader code should be valid Go:\n%s", src)

	require.Contains(t, src, `"go/importer"`)
	require.Contains(t, src, `"go/types"`)

	require.Contains(t, src, "TypesPackages")

	require.Contains(t, src, `"slices"`)
	require.Contains(t, src, `"maps"`)
}

func TestGenerateTypesLoaderWASMFile(t *testing.T) {
	t.Parallel()

	source, err := GenerateTypesLoaderWASMFile(nil, "test_pkg")
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	require.Contains(t, src, "//go:build js && wasm")

	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	require.NoError(t, parseErr, "WASM loader code should be valid Go:\n%s", src)

	require.Contains(t, src, "TypesPackages")

	require.Contains(t, src, `"go/types"`)
}

func TestGenerateTypesLoaderFileEmptyPaths(t *testing.T) {
	t.Parallel()

	source, err := GenerateTypesLoaderFile(nil, "test_pkg")
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	require.NoError(t, parseErr, "empty types loader should be valid Go:\n%s", src)
}

func TestExtractEncodingJsonPackage(t *testing.T) {
	t.Parallel()

	pkgs, err := Extract([]string{"encoding/json"}, nil)
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.Equal(t, "encoding/json", pkg.ImportPath)
	require.Equal(t, "json", pkg.Name)

	symMap := make(map[string]ExtractedSymbol)
	for _, sym := range pkg.Symbols {
		symMap[sym.Name] = sym
	}

	marshal, ok := symMap["Marshal"]
	require.True(t, ok, "json.Marshal should be extracted")
	require.Equal(t, SymbolFunc, marshal.Kind)

	decoder, ok := symMap["Decoder"]
	require.True(t, ok, "json.Decoder should be extracted")
	require.Equal(t, SymbolType, decoder.Kind)
}

func TestExtractSortPackage(t *testing.T) {
	t.Parallel()

	pkgs, err := Extract([]string{"sort"}, nil)
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]

	symMap := make(map[string]ExtractedSymbol)
	for _, sym := range pkg.Symbols {
		symMap[sym.Name] = sym
	}

	iface, ok := symMap["Interface"]
	require.True(t, ok, "sort.Interface should be extracted")
	require.Equal(t, SymbolType, iface.Kind)

	stringsFunc, ok := symMap["Strings"]
	require.True(t, ok, "sort.Strings should be extracted")
	require.Equal(t, SymbolFunc, stringsFunc.Kind)
}

func TestExtractWithGenericConfigPopulatesTypesPackage(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ImportPath:   "slices",
		ElementTypes: []string{"string"},
	}

	pkgs, err := Extract([]string{"slices"}, map[string]PackageConfig{"slices": config})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.NotNil(t, pkg.TypesPackage, "TypesPackage should be populated for generic packages")
	require.Equal(t, "slices", pkg.TypesPackage.Name())
}
