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

package wasm_runner_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/json"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/caller"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/internal/generator/generator_helpers"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
	"piko.sh/piko/wdk/logger"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type WASMRunnerTestSpec struct {
	Description      string `json:"description"`
	ModuleName       string `json:"moduleName,omitempty"`
	RequestURL       string `json:"requestURL"`
	ExpectedStatus   int    `json:"expectedStatus,omitempty"`
	ShouldError      bool   `json:"shouldError,omitempty"`
	ErrorContains    string `json:"errorContains,omitempty"`
	UseDynamicRender bool   `json:"useDynamicRender,omitempty"`
}

func TestWASMRunner_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping WASM runner integration tests in short mode")
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
	resetGlobalStateForTestIsolation()

	spec := loadTestSpec(t, tc)

	absTestCasePath, err := filepath.Abs(tc.Path)
	require.NoError(t, err)
	srcDir := filepath.Join(absTestCasePath, "src")

	sources := loadSources(t, srcDir)

	moduleName := spec.ModuleName
	if moduleName == "" {
		moduleName = "testmodule"
	}

	executor := NewWASMExecutor(wasmTestDir)

	genResp, err := executor.Generate(context.Background(), &wasm_dto.GenerateFromSourcesRequest{
		Sources:    sources,
		ModuleName: moduleName,
	})

	if spec.ShouldError {
		if err != nil {
			if spec.ErrorContains != "" {
				assert.Contains(t, err.Error(), spec.ErrorContains)
			}
			return
		}
		require.False(t, genResp.Success, "Expected generation to fail, but it succeeded")
		if spec.ErrorContains != "" {
			assert.Contains(t, genResp.Error, spec.ErrorContains)
		}
		return
	}

	require.NoError(t, err, "WASM execution failed unexpectedly")
	require.True(t, genResp.Success, "WASM generation failed: %s", genResp.Error)

	var htmlBytes []byte

	renderResp, renderErr := executor.DynamicRender(context.Background(), &wasm_dto.DynamicRenderRequest{
		Sources:    sources,
		ModuleName: moduleName,
		RequestURL: spec.RequestURL,
	})
	require.NoError(t, renderErr, "WASM dynamic render failed")
	require.True(t, renderResp.Success, "WASM render failed: %s", renderResp.Error)

	htmlBytes = []byte(renderResp.HTML)

	goldenHTMLPath := filepath.Join(absTestCasePath, "golden", "rendered.html")
	assertGoldenHTML(t, goldenHTMLPath, htmlBytes, "Rendered HTML for %s", tc.Name)
}

func loadTestSpec(t *testing.T, tc testCase) WASMRunnerTestSpec {
	t.Helper()
	var spec WASMRunnerTestSpec
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

var csrfTokenRegex = regexp.MustCompile(`<meta name="csrf-(token|ephemeral)" content="[^"]*">`)

func normaliseHTML(html []byte) []byte {
	return csrfTokenRegex.ReplaceAll(html, []byte(`<meta name="csrf-$1" content="[NORMALIZED]">`))
}

func assertGoldenHTML(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
	t.Helper()

	normalisedActual := normaliseHTML(actualBytes)

	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
		require.NoError(t, os.WriteFile(goldenPath, normalisedActual, 0644))
	}

	expectedBytes, readErr := os.ReadFile(goldenPath)
	require.NoError(t, readErr, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	normalisedExpected := normaliseHTML(expectedBytes)

	if !bytes.Equal(normalisedExpected, normalisedActual) {
		t.Logf("--- EXPECTED (%s) ---\n%s\n--- ACTUAL (%s) ---\n%s\n",
			filepath.Base(goldenPath), string(normalisedExpected),
			filepath.Base(goldenPath), string(normalisedActual))
		assert.Fail(t, fmt.Sprintf("Golden file mismatch: %s. Run with -update if this change is intentional.", goldenPath), msgAndArgs...)
	}
}

func resetGlobalStateForTestIsolation() {
	logger.ResetLogger()

	config_domain.ResetGlobalFlagCoordinator()
	config_domain.ResetGlobalResolverRegistry()
	config_domain.ResetDotEnvCache()
	config_domain.ResetSecretManager()

	ast_domain.ClearExpressionCache()
	ast_domain.ResetAllPools()

	compiler_domain.ClearIdentifierRegistry()
	compiler_domain.ClearBindingRegistry()
	compiler_domain.ClearLocRefRegistry()

	collection_domain.ResetStaticCollectionRegistry()
	collection_domain.ResetRuntimeProviderRegistry()
	collection_domain.ResetHybridRegistry()

	goastutil.ResetDynamicCaches()
	ast_domain.ClearSelectorCache()
	i18n_domain.ClearExpressionCache()
	generator_helpers.ClearModulePathCaches()
	render_domain.ClearHTMLLinksCache()
	caller.ResetFrameCache()
	logger_domain.ResetCallerCache()
}
