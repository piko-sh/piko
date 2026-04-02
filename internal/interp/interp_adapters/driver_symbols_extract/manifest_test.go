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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadManifestMissingOutput(t *testing.T) {
	t.Parallel()

	yamlContent := `
package: test
packages:
  - fmt
`

	tmpFile := t.TempDir() + "/test-manifest.yaml"
	require.NoError(t, writeTestFile(tmpFile, yamlContent))

	_, err := LoadManifest(tmpFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "'output' is required")
}

func TestLoadManifestNonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := LoadManifest("/nonexistent/path/manifest.yaml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "reading manifest")
}

func TestLoadManifestEmptyPackages(t *testing.T) {
	t.Parallel()

	yamlContent := `
package: test
output: path/to/output
packages: []
`

	tmpFile := t.TempDir() + "/test-manifest.yaml"
	require.NoError(t, writeTestFile(tmpFile, yamlContent))

	_, err := LoadManifest(tmpFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must list at least one package")
}

func TestLoadManifestInvalidYAML(t *testing.T) {
	t.Parallel()

	yamlContent := `
package: test
output: path
packages:
  - [invalid: nested
`

	tmpFile := t.TempDir() + "/test-manifest.yaml"
	require.NoError(t, writeTestFile(tmpFile, yamlContent))

	_, err := LoadManifest(tmpFile)
	require.Error(t, err)
}

func TestManifestGenericConfigs(t *testing.T) {
	t.Parallel()

	yamlContent := `
package: test_symbols
output: path/to/output
packages:
  - fmt
  - strings
  - slices:
      element_types: [string, int]
  - maps:
      key_types: [string]
      value_types: [int]
`

	tmpFile := t.TempDir() + "/test-manifest.yaml"
	require.NoError(t, writeTestFile(tmpFile, yamlContent))

	manifest, err := LoadManifest(tmpFile)
	require.NoError(t, err)

	configs := manifest.GenericConfigs()

	require.Len(t, configs, 2)
	require.Contains(t, configs, "slices")
	require.Contains(t, configs, "maps")
	require.NotContains(t, configs, "fmt")
	require.NotContains(t, configs, "strings")
}

func TestManifestImportPaths(t *testing.T) {
	t.Parallel()

	yamlContent := `
package: test_symbols
output: path/to/output
packages:
  - fmt
  - encoding/json
  - slices:
      element_types: [string]
`

	tmpFile := t.TempDir() + "/test-manifest.yaml"
	require.NoError(t, writeTestFile(tmpFile, yamlContent))

	manifest, err := LoadManifest(tmpFile)
	require.NoError(t, err)

	paths := manifest.ImportPaths()
	require.Equal(t, []string{"fmt", "encoding/json", "slices"}, paths)
}

func TestPackageConfigIsGeneric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config PackageConfig
		expect bool
	}{
		{"empty", PackageConfig{}, false},
		{"element_types", PackageConfig{ElementTypes: []string{"int"}}, true},
		{"key_types", PackageConfig{KeyTypes: []string{"string"}}, true},
		{"functions_only", PackageConfig{Functions: map[string]FunctionConfig{"Sort": {}}}, true},
		{"import_path_only", PackageConfig{ImportPath: "fmt"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expect, tt.config.IsGeneric())
		})
	}
}

func TestPackageConfigTypesForFunc(t *testing.T) {
	t.Parallel()

	config := PackageConfig{
		ElementTypes: []string{"string", "int"},
		KeyTypes:     []string{"string"},
		ValueTypes:   []string{"int", "bool"},
		Functions: map[string]FunctionConfig{
			"SpecialFunc": {
				ElementTypes: []string{"float64"},
				KeyTypes:     []string{"int"},
				ValueTypes:   []string{"string"},
			},
			"PartialOverride": {
				ElementTypes: []string{"byte"},
			},
		},
	}

	element, key, valueTypes := config.TypesForFunc("RegularFunc")
	require.Equal(t, []string{"string", "int"}, element)
	require.Equal(t, []string{"string"}, key)
	require.Equal(t, []string{"int", "bool"}, valueTypes)

	element, key, valueTypes = config.TypesForFunc("SpecialFunc")
	require.Equal(t, []string{"float64"}, element)
	require.Equal(t, []string{"int"}, key)
	require.Equal(t, []string{"string"}, valueTypes)

	element, key, valueTypes = config.TypesForFunc("PartialOverride")
	require.Equal(t, []string{"byte"}, element)
	require.Equal(t, []string{"string"}, key)
	require.Equal(t, []string{"int", "bool"}, valueTypes)
}

func TestLoadManifestWithFunctionsMapping(t *testing.T) {
	t.Parallel()

	yamlContent := `
package: test_symbols
output: path/to/output
packages:
  - maps:
      key_types: [string, int]
      value_types: [string, int, float64]
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
	mapsConfig := configs["maps"]

	_, key, valueTypes := mapsConfig.TypesForFunc("Clone")
	require.Equal(t, []string{"string"}, key)
	require.Equal(t, []string{"string", "int"}, valueTypes)

	_, key, valueTypes = mapsConfig.TypesForFunc("Equal")
	require.Equal(t, []string{"string", "int"}, key)
	require.Equal(t, []string{"string", "int", "float64"}, valueTypes)
}
