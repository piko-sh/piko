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

package compiled_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"piko.sh/piko/internal/json"
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
	"piko.sh/piko/wdk/logger"
)

var (
	updateGoldenFiles  = flag.Bool("update", false, "Update golden files")
	mainGoTemplate     = template.Must(template.ParseFiles("main.go.tmpl"))
	csrfTokenRegex     = regexp.MustCompile(`(<meta name="csrf-token" content=")[^"]*(")`)
	csrfEphemeralRegex = regexp.MustCompile(`(<meta name="csrf-ephemeral" content=")[^"]*(")`)
)

type TemplaterTestSpec struct {
	AdditionalGoldenFiles map[string]string `json:"additionalGoldenFiles,omitempty"`
	Description           string            `json:"description"`
	RequestURL            string            `json:"requestURL"`
	ErrorContains         string            `json:"errorContains,omitempty"`
	ExpectedStatus        int               `json:"expectedStatus,omitempty"`
	IsFragment            bool              `json:"isFragment,omitempty"`
	ShouldError           bool              `json:"shouldError,omitempty"`
	SkipGoldenComparison  bool              `json:"skipGoldenComparison,omitempty"`
}

func runTestCase(t *testing.T, tc testCase) {

	resetGlobalStateForTestIsolation()

	spec := loadTestSpec(t, tc)
	tempDir := t.TempDir()

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	pikoDir := filepath.Join(absSrcDir, ".piko")
	if _, err := os.Stat(pikoDir); err == nil {
		require.NoError(t, os.RemoveAll(pikoDir), "Failed to remove .piko directory")
	}

	srcDistDir := filepath.Join(absSrcDir, "dist")
	if _, err := os.Stat(srcDistDir); err == nil {
		require.NoError(t, os.RemoveAll(srcDistDir), "Failed to remove dist directory")
	}

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(absSrcDir)
	require.NoError(t, err)

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
	)

	err = server.Generate(context.Background(), piko.GenerateModeAll)

	shutdown.Cleanup(context.Background(), 5*time.Second)

	restoreErr := os.Chdir(originalWd)
	require.NoError(t, restoreErr, "Failed to restore original working directory")

	if spec.ShouldError {
		require.Error(t, err, "Expected piko.Generate to fail, but it succeeded for: %s", tc.Name)
		if spec.ErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ErrorContains, "The build-time error message did not contain the expected text")
		}
		return
	}
	require.NoError(t, err, "piko.Generate failed unexpectedly for test case: %s", tc.Name)

	require.NoError(t, copyDir(absSrcDir, tempDir), "Failed to copy test source to temp directory")

	depFile := filepath.Join(tempDir, "piko_dependency.go")
	if _, err := os.Stat(depFile); err == nil {
		require.NoError(t, os.Remove(depFile), "Failed to remove piko_dependency.go")
	}

	distDir := filepath.Join(tempDir, "dist")
	_, err = os.Stat(distDir)
	require.NoError(t, err, "dist directory was not found after generation")

	registerFilePath := filepath.Join(distDir, "piko_register.go")
	_, err = os.Stat(registerFilePath)
	require.NoError(t, err, "piko_register.go was not created in the dist directory")

	pikoProjectRoot, err := findPikoProjectRoot()
	require.NoError(t, err, "Failed to find piko project root")

	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	require.NoError(t, err, "Failed to read go.mod")
	updatedGoModContent := strings.ReplaceAll(string(goModContent),
		"replace piko.sh/piko => ../../../../../../../",
		fmt.Sprintf("replace piko.sh/piko => %s", pikoProjectRoot))
	require.NoError(t, os.WriteFile(goModPath, []byte(updatedGoModContent), 0644), "Failed to update go.mod")

	goWorkContent := fmt.Sprintf("go 1.25.1\n\nuse (\n\t.\n\t%s\n)\n", pikoProjectRoot)
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.work"), []byte(goWorkContent), 0644))

	mainGoPath := filepath.Join(tempDir, "main.go")
	mainGoFile, err := os.Create(mainGoPath)
	require.NoError(t, err)

	moduleNameLine := ""
	for line := range strings.SplitSeq(updatedGoModContent, "\n") {
		if trimmed, found := strings.CutPrefix(line, "module "); found {
			moduleNameLine = strings.TrimSpace(trimmed)
			break
		}
	}
	require.NotEmpty(t, moduleNameLine, "Could not find module name in go.mod")

	err = mainGoTemplate.Execute(mainGoFile, map[string]string{"ModuleName": moduleNameLine})
	require.NoError(t, err)
	_ = mainGoFile.Close()

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyCmd.Env = append(os.Environ(), "GOWORK=off")
	tidyOutput, err := tidyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to `go mod tidy` in temp directory.\nOutput:\n%s", string(tidyOutput))

	buildCmd := exec.Command("go", "build", "-o", "server", ".")
	buildCmd.Dir = tempDir
	buildCmd.Env = append(os.Environ(), "GOWORK=off")
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to `go build` the test server.\nBuild Output:\n%s", string(buildOutput))

	port := findAvailablePort(t)
	serverURL := fmt.Sprintf("http://localhost:%d%s", port, spec.RequestURL)

	serverCmd := exec.Command("./server")
	serverCmd.Dir = tempDir

	serverCmd.Env = append(os.Environ(),
		fmt.Sprintf("PIKO_TEST_PORT=%d", port),
		fmt.Sprintf("PIKO_PORT=%d", port),
	)

	var serverStdout, serverStderr bytes.Buffer
	serverCmd.Stdout = &serverStdout
	serverCmd.Stderr = &serverStderr
	require.NoError(t, serverCmd.Start(), "Failed to start compiled server process")

	t.Cleanup(func() {
		_ = serverCmd.Process.Kill()
		_, _ = serverCmd.Process.Wait()
	})

	if err := waitForServerReady(t, port, 30*time.Second); err != nil {
		t.Fatalf("Server did not become ready in time: %v\nServer Stdout:\n%s\nServer Stderr:\n%s", err, serverStdout.String(), serverStderr.String())
	}

	response, err := http.Get(serverURL)
	if err != nil {
		t.Fatalf("Failed to make request to test server: %v\nServer Stdout:\n%s\nServer Stderr:\n%s", err, serverStdout.String(), serverStderr.String())
	}
	defer func() { _ = response.Body.Close() }()

	expectedStatus := http.StatusOK
	if spec.ExpectedStatus != 0 {
		expectedStatus = spec.ExpectedStatus
	}
	assert.Equal(t, expectedStatus, response.StatusCode, "HTTP status code mismatch")

	htmlBytes, readErr := io.ReadAll(response.Body)
	require.NoError(t, readErr, "Failed to read response body from test server")

	if len(htmlBytes) == 0 {
		t.Logf("WARNING: Received empty response body from server")
		t.Logf("Server Stdout:\n%s", serverStdout.String())
		t.Logf("Server Stderr:\n%s", serverStderr.String())
	}

	goldenPath := filepath.Join(tc.Path, "golden", "golden.html")
	if !spec.SkipGoldenComparison {
		assertGoldenFile(t, goldenPath, htmlBytes, "Generated HTML for %s", tc.Name)
	}

	if len(spec.AdditionalGoldenFiles) > 0 {
		baseURL := fmt.Sprintf("http://localhost:%d", port)
		for urlPath, goldenFileName := range spec.AdditionalGoldenFiles {
			additionalURL := baseURL + urlPath
			additionalRes, err := http.Get(additionalURL)
			if err != nil {
				t.Errorf("Failed to fetch additional asset %s: %v", urlPath, err)
				continue
			}
			additionalBytes, err := io.ReadAll(additionalRes.Body)
			_ = additionalRes.Body.Close()
			if err != nil {
				t.Errorf("Failed to read additional asset %s: %v", urlPath, err)
				continue
			}
			if additionalRes.StatusCode != http.StatusOK {
				t.Errorf("Additional asset %s returned status %d, expected 200", urlPath, additionalRes.StatusCode)
				continue
			}
			additionalGoldenPath := filepath.Join(tc.Path, "golden", goldenFileName)
			assertGoldenFile(t, additionalGoldenPath, additionalBytes, "Additional asset %s for %s", urlPath, tc.Name)
		}
	}
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

func normaliseCSRFTokens(data []byte) []byte {
	result := csrfTokenRegex.ReplaceAll(data, []byte(`${1}NORMALISED${2}`))
	result = csrfEphemeralRegex.ReplaceAll(result, []byte(`${1}NORMALISED${2}`))
	return result
}

func assertGoldenFile(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
	t.Helper()

	normActual := normaliseCSRFTokens(actualBytes)

	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
		require.NoError(t, os.WriteFile(goldenPath, normActual, 0644))
	}
	expectedBytes, readErr := os.ReadFile(goldenPath)
	require.NoError(t, readErr, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	normExpected := normaliseCSRFTokens(expectedBytes)

	if !bytes.Equal(normExpected, normActual) {
		diffCmd := exec.Command("diff", "-u", goldenPath, "-")
		diffCmd.Stdin = bytes.NewReader(normActual)
		diffOutput, _ := diffCmd.CombinedOutput()

		assert.Fail(t, fmt.Sprintf(
			"Golden file mismatch: %s."+
				" Run with -update if this change is"+
				" intentional.\n--- Diff ---\n%s",
			goldenPath, string(diffOutput),
		), msgAndArgs...)
	}
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = srcFile.Close() }()
		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer func() { _ = destFile.Close() }()
		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

func findAvailablePort(t *testing.T) int {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)
	l, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForServerReady(t *testing.T, port int, timeout time.Duration) error {
	t.Helper()
	ctx, cancel := context.WithTimeoutCause(context.Background(), timeout, fmt.Errorf("test: server readiness probe timed out"))
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for server on port %d to become ready", port)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 50*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}

func findPikoProjectRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(directory, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {

			content, readErr := os.ReadFile(goModPath)
			if readErr == nil && strings.Contains(string(content), "module piko.sh/piko") {
				return directory, nil
			}
		}

		parentDir := filepath.Dir(directory)
		if parentDir == directory {

			return "", errors.New("could not find piko project root")
		}
		directory = parentDir
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
