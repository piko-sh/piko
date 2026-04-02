//go:build integration

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

package wasm_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/json"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type WASMTestSpec struct {
	Description   string `json:"description"`
	ModuleName    string `json:"moduleName,omitempty"`
	ErrorContains string `json:"errorContains,omitempty"`
	ShouldError   bool   `json:"shouldError,omitempty"`
}

func TestWASMGenerator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping WASM generator integration tests in short mode")
	}

	if wasmTestDir == "" {
		t.Fatal("wasmTestDir not set - TestMain did not run correctly")
	}

	testdataRoot := "testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("Critical test setup error: Failed to read testdata directory at '%s': %v", testdataRoot, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCaseName := entry.Name()
		tc := testCase{
			Name: testCaseName,
			Path: filepath.Join(testdataRoot, testCaseName),
		}

		t.Run(tc.Name, func(t *testing.T) {
			srcPath := filepath.Join(tc.Path, "src")
			specPath := filepath.Join(tc.Path, "testspec.json")

			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				t.Skipf("Skipping test case '%s': missing 'src' directory", tc.Name)
				return
			}
			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				t.Skipf("Skipping test case '%s': missing 'testspec.json' file", tc.Name)
				return
			}

			runTestCase(t, tc)
		})
	}
}

func runTestCase(t *testing.T, tc testCase) {
	spec := loadTestSpec(t, tc)

	sources := loadSources(t, filepath.Join(tc.Path, "src"))

	moduleName := spec.ModuleName
	if moduleName == "" {
		moduleName = "testmodule"
	}

	executor := NewWASMExecutor(wasmTestDir)

	response, err := executor.Generate(context.Background(), &wasm_dto.GenerateFromSourcesRequest{
		Sources:    sources,
		ModuleName: moduleName,
	})

	if spec.ShouldError {
		if err != nil {
			if spec.ErrorContains != "" {
				assert.Contains(t, err.Error(), spec.ErrorContains,
					"The error message did not contain the expected text")
			}
			return
		}
		require.False(t, response.Success, "Expected generation to fail, but it succeeded for: %s", tc.Name)
		if spec.ErrorContains != "" {
			assert.Contains(t, response.Error, spec.ErrorContains,
				"The error message did not contain the expected text")
		}
		return
	}

	require.NoError(t, err, "WASM execution failed unexpectedly")
	require.True(t, response.Success, "Generation failed: %s", response.Error)

	absTestCasePath, _ := filepath.Abs(tc.Path)
	srcDir := filepath.Join(absTestCasePath, "src")
	for _, artefact := range response.Artefacts {

		artefactPath := artefact.Path
		if filepath.IsAbs(artefactPath) {
			relPath, err := filepath.Rel(srcDir, artefactPath)
			if err == nil {
				artefactPath = relPath
			}
		}
		goldenPath := filepath.Join(tc.Path, "golden", artefactPath)
		assertGoldenFile(t, goldenPath, []byte(artefact.Content), "Generated artefact %s", artefactPath)
	}

	if response.Manifest != nil {
		manifestPath := filepath.Join(tc.Path, "golden", "manifest.json")
		manifestBytes, jsonErr := json.StdConfig().MarshalIndent(response.Manifest, "", "  ")
		require.NoError(t, jsonErr, "Failed to marshal manifest to JSON")
		assertGoldenFileJSON(t, manifestPath, manifestBytes, "Generated manifest")
	}
}

func loadTestSpec(t *testing.T, tc testCase) WASMTestSpec {
	t.Helper()
	var spec WASMTestSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)

	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	return spec
}

func loadSources(t *testing.T, srcDir string) map[string]string {
	t.Helper()
	sources := make(map[string]string)

	err := filepath.WalkDir(srcDir, func(absPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(d.Name()), ".pk") {
			return nil
		}

		relPath, relErr := filepath.Rel(srcDir, absPath)
		if relErr != nil {
			return relErr
		}

		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return readErr
		}

		key := filepath.ToSlash(relPath)
		sources[key] = string(content)

		return nil
	})
	require.NoError(t, err, "Failed to walk source directory")

	return sources
}

func assertGoldenFile(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
	t.Helper()

	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
		require.NoError(t, os.WriteFile(goldenPath, actualBytes, 0644))
	}

	expectedBytes, readErr := os.ReadFile(goldenPath)
	require.NoError(t, readErr, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	if !bytes.Equal(expectedBytes, actualBytes) {
		t.Logf("--- EXPECTED (%s) ---\n%s\n--- ACTUAL (%s) ---\n%s\n",
			filepath.Base(goldenPath), string(expectedBytes),
			filepath.Base(goldenPath), string(actualBytes))
		assert.Fail(t, fmt.Sprintf("Golden file mismatch: %s. Run with -update if this change is intentional.", goldenPath), msgAndArgs...)
	}
}

func assertGoldenFileJSON(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
	t.Helper()

	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
		require.NoError(t, os.WriteFile(goldenPath, actualBytes, 0644))
	}

	expectedBytes, readErr := os.ReadFile(goldenPath)
	require.NoError(t, readErr, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	assert.JSONEq(t, string(expectedBytes), string(actualBytes), msgAndArgs...)
}
