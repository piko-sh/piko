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

package cache_rendering_test

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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/json"
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
	"piko.sh/piko/wdk/interp/interp_provider_piko"
	"piko.sh/piko/wdk/logger"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type CacheRenderingTestSpec struct {
	Pages          map[string]string `json:"pages"`
	Description    string            `json:"description"`
	ExpectedStatus int               `json:"expectedStatus,omitempty"`
}

type requestResult struct {
	headers http.Header
	body    []byte
	status  int
}

func setupServer(t *testing.T, tc testCase) (*piko.SSRServer, string, func()) {
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

	server.Configure(piko.PublicConfig{
		BaseDir:        ".",
		PagesSourceDir: "pages",
	})

	return server, absTestCasePath, cleanup
}

func makeRequest(t *testing.T, handler http.Handler, url string) requestResult {
	t.Helper()

	request := httptest.NewRequest("GET", url, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err, "Failed to read response body for %s", url)

	return requestResult{
		body:    body,
		headers: response.Header,
		status:  response.StatusCode,
	}
}

func runTestCase(t *testing.T, tc testCase) {
	resetGlobalStateForTestIsolation()

	spec := loadTestSpec(t, tc)

	server, absTestCasePath, cleanup := setupServer(t, tc)
	defer cleanup()
	defer server.Close()

	err := server.Generate(context.Background(), piko.RunModeDevInterpreted)
	require.NoError(t, err, "Failed to generate project in dev-i mode")

	handler := server.GetHandler()
	require.NotNil(t, handler, "GetHandler returned nil - daemon not built correctly")

	expectedStatus := http.StatusOK
	if spec.ExpectedStatus != 0 {
		expectedStatus = spec.ExpectedStatus
	}

	nocacheURL := spec.Pages["nocache"]
	astcacheURL := spec.Pages["astcache"]
	staticcacheURL := spec.Pages["staticcache"]

	require.NotEmpty(t, nocacheURL, "testspec missing pages.nocache")
	require.NotEmpty(t, astcacheURL, "testspec missing pages.astcache")
	require.NotEmpty(t, staticcacheURL, "testspec missing pages.staticcache")

	resultNoCache := makeRequest(t, handler, nocacheURL)
	assert.Equal(t, expectedStatus, resultNoCache.status, "nocache: HTTP status mismatch")

	resultASTMiss := makeRequest(t, handler, astcacheURL)
	assert.Equal(t, expectedStatus, resultASTMiss.status, "astcache (miss): HTTP status mismatch")

	resultASTHit := makeRequest(t, handler, astcacheURL)
	assert.Equal(t, expectedStatus, resultASTHit.status, "astcache (hit): HTTP status mismatch")

	resultStaticMiss := makeRequest(t, handler, staticcacheURL)
	assert.Equal(t, expectedStatus, resultStaticMiss.status, "staticcache (miss): HTTP status mismatch")

	resultStaticHit := makeRequest(t, handler, staticcacheURL)
	assert.Equal(t, expectedStatus, resultStaticHit.status, "staticcache (hit): HTTP status mismatch")

	normNoCache := normaliseForComparison(resultNoCache.body)
	normASTMiss := normaliseForComparison(resultASTMiss.body)
	normASTHit := normaliseForComparison(resultASTHit.body)
	normStaticMiss := normaliseForComparison(resultStaticMiss.body)
	normStaticHit := normaliseForComparison(resultStaticHit.body)

	goldenPath := filepath.Join(absTestCasePath, "golden", "golden.html")
	assertGoldenFile(t, goldenPath, normNoCache, "Golden file for %s", tc.Name)

	assertHTMLEqual(t, normNoCache, normASTMiss, "nocache vs astcache(miss)")
	assertHTMLEqual(t, normNoCache, normASTHit, "nocache vs astcache(hit)")
	assertHTMLEqual(t, normNoCache, normStaticMiss, "nocache vs staticcache(miss)")
	assertHTMLEqual(t, normNoCache, normStaticHit, "nocache vs staticcache(hit)")

	verifyHeaders(t, resultStaticMiss, resultStaticHit)
}

func verifyHeaders(t *testing.T, staticMiss, staticHit requestResult) {
	t.Helper()

	missStatus := staticMiss.headers.Get("X-Cache-Status")
	if missStatus != "" {
		assert.Equal(t, "MISS", missStatus, "staticcache first request should be MISS (or absent)")
	}

	hitStatus := staticHit.headers.Get("X-Cache-Status")
	if hitStatus != "" {
		assert.Equal(t, "HIT", hitStatus, "staticcache second request should be HIT")
	}
}

func loadTestSpec(t *testing.T, tc testCase) CacheRenderingTestSpec {
	t.Helper()
	var spec CacheRenderingTestSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)

	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	return spec
}

var (
	csrfMetaRegex = regexp.MustCompile(`<meta name="csrf-(token|ephemeral)" content="[^"]*">`)
	csrfAttrRegex = regexp.MustCompile(`data-csrf-(ephemeral-token|action-token)="[^"]*"`)
	partialRegex  = regexp.MustCompile(`partial="pages_(nocache|astcache|staticcache)_[a-f0-9]+"`)
	pageidRegex   = regexp.MustCompile(`data-pageid="pages/(nocache|astcache|staticcache)\.pk"`)
)

func normaliseForComparison(html []byte) []byte {
	result := csrfMetaRegex.ReplaceAll(html, []byte(`<meta name="csrf-$1" content="[NORMALISED]">`))
	result = csrfAttrRegex.ReplaceAll(result, []byte(`data-csrf-$1="[NORMALISED]"`))
	result = partialRegex.ReplaceAll(result, []byte(`partial="pages_[PAGE]_[HASH]"`))
	result = pageidRegex.ReplaceAll(result, []byte(`data-pageid="pages/[PAGE].pk"`))
	return result
}

func assertHTMLEqual(t *testing.T, expected, actual []byte, label string) {
	t.Helper()
	if !bytes.Equal(expected, actual) {
		t.Logf("--- EXPECTED (%s) ---\n%s\n--- ACTUAL (%s) ---\n%s\n", label, string(expected), label, string(actual))
		assert.Fail(t, fmt.Sprintf("HTML mismatch: %s", label))
	}
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
		t.Logf("--- EXPECTED (%s) ---\n%s\n--- ACTUAL (%s) ---\n%s\n", filepath.Base(goldenPath), string(expectedBytes), filepath.Base(goldenPath), string(actualBytes))
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
