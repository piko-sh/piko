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

package interpreted_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"piko.sh/piko/wdk/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko"
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
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/interp/interp_provider_piko"
	"piko.sh/piko/wdk/logger"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type TemplaterTestSpec struct {
	Description           string `json:"description"`
	RequestURL            string `json:"requestURL"`
	ErrorContains         string `json:"errorContains,omitempty"`
	ExpectedStatus        int    `json:"expectedStatus,omitempty"`
	IsFragment            bool   `json:"isFragment,omitempty"`
	ShouldError           bool   `json:"shouldError,omitempty"`
	ShouldGenerateError   bool   `json:"shouldGenerateError,omitempty"`
	RegisteredPackages    bool   `json:"registeredPackages,omitempty"`
	CrossPackageNamedType bool   `json:"crossPackageNamedType,omitempty"`
	CrossPackageTimeField bool   `json:"crossPackageTimeField,omitempty"`
}

type crossPackageIdentity struct {
	ID   uuid.UUID
	Name string
}

type crossPackageSession struct {
	Name      string
	ExpiresAt time.Time
}

func setupServer(t *testing.T, tc testCase, spec TemplaterTestSpec) (*piko.SSRServer, string, func()) {
	t.Helper()

	absTestCasePath, err := filepath.Abs(tc.Path)
	require.NoError(t, err)

	srcDir := filepath.Join(absTestCasePath, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(absSrcDir)
	require.NoError(t, err)

	cleanup := func() {
		_ = os.Chdir(originalWd)
	}

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
	)
	server.WithInterpreterProvider(interp_provider_piko.NewProvider())

	if spec.RegisteredPackages {
		moduleName := readModuleName(t, absSrcDir)
		server.WithSymbols(templater_domain.SymbolExports{
			moduleName + "/pkg/registered": {
				"GetName": reflect.ValueOf(func() string { return "Registered Package" }),
			},
		})
	}

	if spec.CrossPackageNamedType {
		moduleName := readModuleName(t, absSrcDir)
		server.WithSymbols(templater_domain.SymbolExports{
			"github.com/google/uuid": {
				"UUID":      reflect.ValueOf((*uuid.UUID)(nil)),
				"New":       reflect.ValueOf(uuid.New),
				"MustParse": reflect.ValueOf(uuid.MustParse),
			},
			moduleName + "/pkg/identity": {
				"Identity": reflect.ValueOf((*crossPackageIdentity)(nil)),
				"NewIdentity": reflect.ValueOf(func(name string) crossPackageIdentity {
					return crossPackageIdentity{
						ID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						Name: name,
					}
				}),
			},
		})
	}

	if spec.CrossPackageTimeField {
		moduleName := readModuleName(t, absSrcDir)
		server.WithSymbols(templater_domain.SymbolExports{
			moduleName + "/pkg/session": {
				"Session": reflect.ValueOf((*crossPackageSession)(nil)),
				"NewSession": reflect.ValueOf(func() crossPackageSession {
					return crossPackageSession{
						Name:      "test-session",
						ExpiresAt: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
					}
				}),
			},
		})
	}

	server.Configure(piko.PublicConfig{
		BaseDir:        ".",
		PagesSourceDir: "pages",
	})

	return server, absTestCasePath, cleanup
}

func readModuleName(t *testing.T, srcDir string) string {
	t.Helper()
	goModBytes, err := os.ReadFile(filepath.Join(srcDir, "go.mod"))
	require.NoError(t, err, "Failed to read go.mod for module name")
	for line := range bytes.SplitSeq(goModBytes, []byte("\n")) {
		if after, ok := bytes.CutPrefix(line, []byte("module ")); ok {
			return string(bytes.TrimSpace(after))
		}
	}
	t.Fatal("go.mod missing module directive")
	return ""
}

func runTestCase(t *testing.T, tc testCase) {
	resetGlobalStateForTestIsolation()

	spec := loadTestSpec(t, tc)
	server, absTestCasePath, cleanup := setupServer(t, tc, spec)
	defer cleanup()
	defer server.Close()

	err := server.Generate(context.Background(), piko.RunModeDevInterpreted)

	if spec.ShouldGenerateError {
		require.Error(t, err, "Expected generation to fail for: %s", tc.Name)
		if spec.ErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ErrorContains)
		}
		return
	}

	require.NoError(t, err, "Failed to generate project in dev-i mode")

	handler := server.GetHandler()
	require.NotNil(t, handler, "GetHandler returned nil - daemon not built correctly")

	request := httptest.NewRequest("GET", spec.RequestURL, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer func() { _ = response.Body.Close() }()

	expectedStatus := http.StatusOK
	if spec.ExpectedStatus != 0 {
		expectedStatus = spec.ExpectedStatus
	}

	if spec.ShouldError {
		assert.NotEqual(t, http.StatusOK, response.StatusCode, "Expected error status for: %s", tc.Name)
		return
	}

	assert.Equal(t, expectedStatus, response.StatusCode, "HTTP status code mismatch")

	htmlBytes, readErr := io.ReadAll(response.Body)
	require.NoError(t, readErr, "Failed to read response body")

	goldenPath := filepath.Join(absTestCasePath, "golden", "golden.html")
	assertGoldenFile(t, goldenPath, htmlBytes, "Generated HTML for %s", tc.Name)
}

func loadTestSpec(t *testing.T, tc testCase) TemplaterTestSpec {
	t.Helper()
	var spec TemplaterTestSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)

	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	return spec
}

var csrfTokenRegex = regexp.MustCompile(`<meta name="csrf-(token|ephemeral)" content="[^"]*">`)

func normaliseHTML(html []byte) []byte {
	return csrfTokenRegex.ReplaceAll(html, []byte(`<meta name="csrf-$1" content="[NORMALIZED]">`))
}

func assertGoldenFile(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
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
		t.Logf("--- EXPECTED (%s) ---\n%s\n--- ACTUAL (%s) ---\n%s\n", filepath.Base(goldenPath), string(normalisedExpected), filepath.Base(goldenPath), string(normalisedActual))
		assert.Fail(t, fmt.Sprintf("Golden file mismatch: %s. Run with -update if this change is intentional.", goldenPath), msgAndArgs...)
	}
}

func resetGlobalStateForTestIsolation() {
	shutdown.Reset()
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
