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

package builder_lite_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type TestSpec struct {
	Description   string          `json:"description"`
	ModuleName    string          `json:"moduleName"`
	ErrorContains string          `json:"errorContains"`
	Assertions    []JSONAssertion `json:"assertions"`
	ShouldError   bool            `json:"shouldError"`
}

type JSONAssertion struct {
	Expect      any    `json:"expect"`
	Description string `json:"description"`
	Select      string `json:"select"`
}

func TestLiteBuilder_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping lite builder tests in short mode")
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get test file path via runtime.Caller")
	}
	testdataRoot := filepath.Join(filepath.Dir(thisFile), "testdata")

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		t.Run(tc.Name, func(t *testing.T) {
			runLiteBuilderTestCase(t, tc)
		})
	}
}

func runLiteBuilderTestCase(t *testing.T, tc testCase) {

	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json")

	var spec TestSpec
	require.NoError(t, json.Unmarshal(specBytes, &spec), "Failed to unmarshal testspec.json")

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err, "Failed to get absolute path for src directory")

	sourceContents := getAllGoSource(t, absSrcDir)

	config := inspector_dto.Config{
		BaseDir:    absSrcDir,
		ModuleName: spec.ModuleName,
	}

	builder, err := inspector_domain.NewLiteBuilder(testStdlibData, config)
	require.NoError(t, err, "Failed to create LiteBuilder")

	err = builder.Build(context.Background(), sourceContents)

	if spec.ShouldError {
		require.Error(t, err, "Expected builder.Build() to fail")
		if spec.ErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ErrorContains)
		}
		return
	}
	require.NoError(t, err, "builder.Build() failed unexpectedly")

	typeData, err := builder.GetTypeData()
	require.NoError(t, err, "Failed to get TypeData")

	goldenDir := filepath.Join(tc.Path, "golden")
	require.NoError(t, os.MkdirAll(goldenDir, 0755))

	filteredData := filterTypeData(typeData, spec.ModuleName)

	testdataRoot, err := filepath.Abs(filepath.Dir(tc.Path))
	require.NoError(t, err)
	sanitiseTypeData(filteredData, testdataRoot)

	jsonBytes, err := json.ConfigStd.MarshalIndent(filteredData, "", "  ")
	require.NoError(t, err, "Failed to marshal TypeData to JSON")

	checkGoldenFile(t, tc.Name, filepath.Join(goldenDir, "types.json"), string(jsonBytes))

	var jsonData any
	require.NoError(t, json.Unmarshal(jsonBytes, &jsonData))

	for _, assertion := range spec.Assertions {
		t.Run(assertion.Description, func(t *testing.T) {
			assertOnJSON(t, jsonData, assertion)
		})
	}
}

func filterTypeData(data *inspector_dto.TypeData, prefix string) *inspector_dto.TypeData {
	filtered := &inspector_dto.TypeData{
		Packages: make(map[string]*inspector_dto.Package),
	}

	for path, pkg := range data.Packages {
		if strings.HasPrefix(path, prefix) {
			filtered.Packages[path] = pkg
		}
	}

	return filtered
}

func checkGoldenFile(t *testing.T, testName, goldenPath, actualContent string) {
	if !strings.HasSuffix(actualContent, "\n") {
		actualContent += "\n"
	}

	if *updateGoldenFiles {
		t.Logf("Updating golden file: %s", goldenPath)
		require.NoError(t, os.WriteFile(goldenPath, []byte(actualContent), 0644))
		return
	}

	expectedContent, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("Golden file not found: %s. Run with -update to generate it.", goldenPath)
	}
	require.NoError(t, err)
	assert.Equal(t, string(expectedContent), actualContent, "Golden file mismatch for %s", goldenPath)
}

func getAllGoSource(t *testing.T, srcDir string) map[string][]byte {
	sources := make(map[string][]byte)
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	err = filepath.Walk(absSrcDir, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			content, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			sources[path] = content
		}
		return nil
	})
	require.NoError(t, err)
	return sources
}

func sanitiseTypeData(td *inspector_dto.TypeData, prefix string) {
	for _, pkg := range td.Packages {
		sanitisedFileImports := make(map[string]map[string]string, len(pkg.FileImports))
		for path, imports := range pkg.FileImports {
			sanitisedFileImports[sanitisePath(path, prefix)] = imports
		}
		pkg.FileImports = sanitisedFileImports

		for _, typ := range pkg.NamedTypes {
			typ.DefinedInFilePath = sanitisePath(typ.DefinedInFilePath, prefix)
			for _, field := range typ.Fields {
				field.DefinitionFilePath = sanitisePath(field.DefinitionFilePath, prefix)
			}
			for _, method := range typ.Methods {
				method.DefinitionFilePath = sanitisePath(method.DefinitionFilePath, prefix)
			}
		}

		for _, inspectedFunction := range pkg.Funcs {
			inspectedFunction.DefinitionFilePath = sanitisePath(inspectedFunction.DefinitionFilePath, prefix)
		}

		for _, v := range pkg.Variables {
			v.DefinedInFilePath = sanitisePath(v.DefinedInFilePath, prefix)
		}
	}

	sanitisedFTP := make(map[string]string, len(td.FileToPackage))
	for path, pkg := range td.FileToPackage {
		sanitisedFTP[sanitisePath(path, prefix)] = pkg
	}
	td.FileToPackage = sanitisedFTP
}

func sanitisePath(path, prefix string) string {
	if path == "" || prefix == "" {
		return path
	}
	if sanitised, ok := strings.CutPrefix(path, prefix); ok {
		sanitised = strings.TrimPrefix(sanitised, string(filepath.Separator))
		return filepath.ToSlash(sanitised)
	}
	return filepath.ToSlash(path)
}

func assertOnJSON(t *testing.T, jsonData any, assertion JSONAssertion) {
	segments := strings.Split(assertion.Select, ".")
	var current = jsonData

	if len(segments) > 1 && segments[0] == "packages" {
		packagesObj, ok := jsonData.(map[string]any)
		if !ok {
			t.Fatalf("Top-level JSON data is not an object for selector '%s'", assertion.Select)
		}
		packagesMap, ok := packagesObj["packages"].(map[string]any)
		if !ok {
			t.Fatalf("JSON data does not contain a 'packages' map for selector '%s'", assertion.Select)
		}

		for i := len(segments); i > 1; i-- {
			pkgKey := strings.Join(segments[1:i], ".")
			if pkgData, keyExists := packagesMap[pkgKey]; keyExists {
				current = pkgData
				segments = segments[i:]
				break
			}
		}
	}

	for i, seg := range segments {
		if current == nil {
			t.Fatalf("Path is nil at segment #%d ('%s') for selector '%s'", i, seg, assertion.Select)
		}

		if currentMap, ok := current.(map[string]any); ok {
			var found bool
			current, found = currentMap[seg]
			if !found {
				t.Fatalf("Key not found at segment #%d ('%s') for selector '%s'", i, seg, assertion.Select)
			}
			continue
		}

		if currentArray, ok := current.([]any); ok {
			index, err := strconv.Atoi(seg)
			if err != nil {
				t.Fatalf("Expected integer index at segment #%d, got '%s' for selector '%s'", i, seg, assertion.Select)
			}
			if index < 0 || index >= len(currentArray) {
				t.Fatalf("Index %d out of bounds for array of length %d at segment #%d", index, len(currentArray), i)
			}
			current = currentArray[index]
			continue
		}

		t.Fatalf("Expected object or array at segment #%d, got %T for selector '%s'", i, current, assertion.Select)
	}

	if expectedNum, ok := assertion.Expect.(float64); ok {
		if actualNum, ok := current.(float64); ok {
			assert.InDelta(t, expectedNum, actualNum, 0.001, "Value mismatch for selector '%s'", assertion.Select)
			return
		}
	}

	assert.Equal(t, assertion.Expect, current, "Value mismatch for selector '%s'", assertion.Select)
}
