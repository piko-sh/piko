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
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateFile(t *testing.T) {
	t.Parallel()

	pkg := ExtractedPackage{
		ImportPath: "strings",
		Name:       "strings",
		Symbols: []ExtractedSymbol{
			{Name: "ToUpper", Kind: SymbolFunc},
			{Name: "Contains", Kind: SymbolFunc},
			{Name: "Builder", Kind: SymbolType},
		},
	}

	source, err := GenerateFile(pkg, "test_output", PackageConfig{})
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err, "generated code should be valid Go:\n%s", src)

	require.Contains(t, src, `package test_output`)
	require.Contains(t, src, `Symbols["strings"]`)
	require.Contains(t, src, `"ToUpper"`)
	require.Contains(t, src, `reflect.ValueOf(strings.ToUpper)`)
	require.Contains(t, src, `reflect.ValueOf(strings.Contains)`)
	require.Contains(t, src, `reflect.ValueOf((*strings.Builder)(nil))`)
}

func TestGenerateFileNestedPackage(t *testing.T) {
	t.Parallel()

	pkg := ExtractedPackage{
		ImportPath: "encoding/json",
		Name:       "json",
		Symbols: []ExtractedSymbol{
			{Name: "Marshal", Kind: SymbolFunc},
			{Name: "Decoder", Kind: SymbolType},
		},
	}

	source, err := GenerateFile(pkg, "my_pkg", PackageConfig{})
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)

	require.Contains(t, src, `json "encoding/json"`)
	require.Contains(t, src, `Symbols["encoding/json"]`)
	require.Contains(t, src, `reflect.ValueOf(json.Marshal)`)
}

func TestGenerateFileVariable(t *testing.T) {
	t.Parallel()

	pkg := ExtractedPackage{
		ImportPath: "os",
		Name:       "os",
		Symbols: []ExtractedSymbol{
			{Name: "Args", Kind: SymbolVar},
			{Name: "Exit", Kind: SymbolFunc},
		},
	}

	source, err := GenerateFile(pkg, "test_pkg", PackageConfig{})
	require.NoError(t, err)

	src := string(source)

	require.Contains(t, src, `reflect.ValueOf(&os.Args).Elem()`)

	require.Contains(t, src, `reflect.ValueOf(os.Exit)`)
}

func TestGenerateFileSkipsUntypedConstants(t *testing.T) {
	t.Parallel()

	pkg := ExtractedPackage{
		ImportPath: "math",
		Name:       "math",
		Symbols: []ExtractedSymbol{
			{Name: "Pi", Kind: SymbolConst, IsUntypedConst: true},
			{Name: "Sqrt", Kind: SymbolFunc},
		},
	}

	source, err := GenerateFile(pkg, "test_pkg", PackageConfig{})
	require.NoError(t, err)

	src := string(source)

	require.NotContains(t, src, `"Pi"`)

	require.Contains(t, src, `"Sqrt"`)
}

func TestGenerateFileEmptyPackage(t *testing.T) {
	t.Parallel()

	pkg := ExtractedPackage{
		ImportPath: "empty",
		Name:       "empty",
		Symbols:    nil,
	}

	source, err := GenerateFile(pkg, "test_pkg", PackageConfig{})
	require.NoError(t, err)
	require.Nil(t, source, "empty packages should return nil source")
}

func TestOutputFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		expect string
	}{
		{input: "fmt", expect: "gen_fmt.go"},
		{input: "strings", expect: "gen_strings.go"},
		{input: "encoding/json", expect: "gen_encoding_json.go"},
		{input: "encoding/base64", expect: "gen_encoding_base64.go"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expect, OutputFileName(tt.input))
		})
	}
}

func TestExtractRealPackage(t *testing.T) {
	t.Parallel()

	packages, err := Extract([]string{"errors"}, nil)
	require.NoError(t, err)
	require.Len(t, packages, 1)

	pkg := packages[0]
	require.Equal(t, "errors", pkg.ImportPath)
	require.Equal(t, "errors", pkg.Name)

	var hasNew bool
	for _, sym := range pkg.Symbols {
		if sym.Name == "New" {
			hasNew = true
			require.Equal(t, SymbolFunc, sym.Kind)
		}
	}
	require.True(t, hasNew, "errors.New should be extracted")
}

func TestManifestParsing(t *testing.T) {
	t.Parallel()

	yaml := `
package: my_symbols
output: path/to/output
packages:
  - fmt
  - strings
  - encoding/json
`

	tmpFile := t.TempDir() + "/test-manifest.yaml"
	require.NoError(t, writeTestFile(tmpFile, yaml))

	manifest, err := LoadManifest(tmpFile)
	require.NoError(t, err)
	require.Equal(t, "my_symbols", manifest.Package)
	require.Equal(t, "path/to/output", manifest.Output)
	require.Equal(t, []string{"fmt", "strings", "encoding/json"}, manifest.ImportPaths())
}

func TestManifestMissingPackage(t *testing.T) {
	t.Parallel()

	yaml := `
output: path/to/output
packages:
  - fmt
`

	tmpFile := t.TempDir() + "/test-manifest.yaml"
	require.NoError(t, writeTestFile(tmpFile, yaml))

	_, err := LoadManifest(tmpFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "'package' is required")
}

func TestManifestPerFunctionOverrides(t *testing.T) {
	t.Parallel()

	yamlContent := `
package: test_symbols
output: path/to/output
packages:
  - slices:
      element_types: [string, int, int64, float64, byte, bool]
      functions:
        BinarySearch: [string, int, int64, float64, byte]
        Sort: [string, int]
  - maps:
      key_types: [string, int]
      value_types: [string, int, float64, bool, any]
      functions:
        Clone:
          key_types: [string]
          value_types: [string, int]
`

	tmpFile := t.TempDir() + "/test-manifest.yaml"
	require.NoError(t, writeTestFile(tmpFile, yamlContent))

	manifest, err := LoadManifest(tmpFile)
	require.NoError(t, err)

	configs := manifest.GenericConfigs()

	slicesConfig := configs["slices"]
	require.NotNil(t, slicesConfig.Functions)

	element, key, valueTypes := slicesConfig.TypesForFunc("Contains")
	require.Equal(t, []string{"string", "int", "int64", "float64", "byte", "bool"}, element)
	require.Empty(t, key)
	require.Empty(t, valueTypes)

	element, _, _ = slicesConfig.TypesForFunc("BinarySearch")
	require.Equal(t, []string{"string", "int", "int64", "float64", "byte"}, element)

	element, _, _ = slicesConfig.TypesForFunc("Sort")
	require.Equal(t, []string{"string", "int"}, element)

	mapsConfig := configs["maps"]
	_, key, valueTypes = mapsConfig.TypesForFunc("Equal")
	require.Equal(t, []string{"string", "int"}, key)
	require.Equal(t, []string{"string", "int", "float64", "bool", "any"}, valueTypes)

	_, key, valueTypes = mapsConfig.TypesForFunc("Clone")
	require.Equal(t, []string{"string"}, key)
	require.Equal(t, []string{"string", "int"}, valueTypes)
}

func writeTestFile(path, content string) error {
	content = strings.TrimSpace(content) + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}
