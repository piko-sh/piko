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

//go:build integration

package examples_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

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
	"piko.sh/piko/internal/shutdown"
	browserpkg "piko.sh/piko/wdk/browser"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safedisk"
)

type ExamplesHarness struct {
	T        *testing.T
	TestCase testCase
	Spec     *browserpkg.TestSpec

	TempDir   string
	SrcDir    string
	GoldenDir string

	ServerCmd  *exec.Cmd
	ServerPort int
	ServerURL  string

	PageHelper    *browserpkg.PageHelper
	IncognitoPage *browserpkg.IncognitoPage
	ActionCtx     *browserpkg.ActionContext
}

func NewExamplesHarness(t *testing.T, tc testCase) *ExamplesHarness {
	t.Helper()

	return &ExamplesHarness{
		T:         t,
		TestCase:  tc,
		SrcDir:    filepath.Join(tc.Path, "src"),
		GoldenDir: filepath.Join(tc.Path, "golden"),
	}
}

func (h *ExamplesHarness) Setup() error {

	resetGlobalStateForTestIsolation()

	specPath := filepath.Join(h.TestCase.Path, "testspec.json")
	spec, err := browserpkg.LoadTestSpec(specPath)
	if err != nil {
		return fmt.Errorf("loading test spec: %w", err)
	}
	h.Spec = spec

	h.TempDir = h.T.TempDir()

	return nil
}

func (h *ExamplesHarness) BuildServer() error {
	h.T.Helper()

	absSrcDir, err := filepath.Abs(h.SrcDir)
	if err != nil {
		return fmt.Errorf("getting absolute src path: %w", err)
	}

	pikoCache := filepath.Join(absSrcDir, ".piko")
	if _, err := os.Stat(pikoCache); err == nil {
		if err := os.RemoveAll(pikoCache); err != nil {
			return fmt.Errorf("removing .piko directory: %w", err)
		}
	}

	genCmd := exec.Command("go", "run", "./cmd/generator/", "all")
	genCmd.Dir = absSrcDir
	genCmd.Env = append(os.Environ(), "GOWORK=off")
	genOutput, genErr := genCmd.CombinedOutput()
	if genErr != nil {
		if h.Spec.ShouldError {
			errOutput := string(genOutput)
			if h.Spec.ErrorContains != "" && !strings.Contains(errOutput, h.Spec.ErrorContains) {
				return fmt.Errorf("expected error containing %q, got: %s", h.Spec.ErrorContains, errOutput)
			}
			return nil
		}
		return fmt.Errorf("generator failed: %w\nOutput: %s", genErr, string(genOutput))
	}

	if err := copyDir(absSrcDir, h.TempDir); err != nil {
		return fmt.Errorf("copying source to temp: %w", err)
	}

	pikoProjectRoot, err := findPikoProjectRoot()
	if err != nil {
		return fmt.Errorf("finding piko project root: %w", err)
	}

	goModPath := filepath.Join(h.TempDir, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	goModString := string(goModContent)

	pikoReplacePattern := regexp.MustCompile(`replace (piko\.sh/piko(?:/[^\s]+)?)\s+=>\s+\.\./[^\n]+`)
	updatedGoModContent := pikoReplacePattern.ReplaceAllStringFunc(goModString, func(match string) string {
		submatches := pikoReplacePattern.FindStringSubmatch(match)
		moduleName := submatches[1]
		subPath := strings.TrimPrefix(moduleName, "piko.sh/piko")
		localPath := filepath.Join(pikoProjectRoot, subPath)
		return fmt.Sprintf("replace %s => %s", moduleName, localPath)
	})

	if err := os.WriteFile(goModPath, []byte(updatedGoModContent), 0644); err != nil {
		return fmt.Errorf("updating go.mod: %w", err)
	}

	goWorkContent := fmt.Sprintf("go 1.26.0\n\nuse (\n\t.\n\t%s\n)\n", pikoProjectRoot)
	if err := os.WriteFile(filepath.Join(h.TempDir, "go.work"), []byte(goWorkContent), 0644); err != nil {
		return fmt.Errorf("creating go.work: %w", err)
	}

	buildSrvCmd := exec.Command("go", "build", "-o", "server", "./cmd/main/")
	buildSrvCmd.Dir = h.TempDir
	buildSrvCmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := buildSrvCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build server failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (h *ExamplesHarness) StartServer() error {
	h.T.Helper()

	port, err := browserpkg.FindAvailablePort()
	require.NoError(h.T, err)
	h.ServerPort = port

	scheme := "http"
	if h.Spec.TLS {
		scheme = "https"
	}
	h.ServerURL = fmt.Sprintf("%s://localhost:%d", scheme, h.ServerPort)

	h.ServerCmd = exec.Command("./server", "prod")
	h.ServerCmd.Dir = h.TempDir
	h.ServerCmd.Env = append(os.Environ(),
		fmt.Sprintf("PIKO_PORT=%d", h.ServerPort),
	)

	var stdout, stderr bytes.Buffer
	h.ServerCmd.Stdout = &stdout
	h.ServerCmd.Stderr = &stderr

	if err := h.ServerCmd.Start(); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	if err := browserpkg.WaitForServerReady(h.ServerPort, 30*time.Second); err != nil {
		return fmt.Errorf("server not ready: %w\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

func (h *ExamplesHarness) StopServer() {
	if h.ServerCmd != nil && h.ServerCmd.Process != nil {
		_ = h.ServerCmd.Process.Kill()
		_, _ = h.ServerCmd.Process.Wait()
	}
}

func (h *ExamplesHarness) SetupBrowser() error {
	h.T.Helper()

	incognitoPage, err := browser.NewIncognitoPage()
	if err != nil {
		return fmt.Errorf("creating incognito page: %w", err)
	}
	h.IncognitoPage = incognitoPage
	h.PageHelper = browserpkg.NewPageHelper(incognitoPage.Ctx)
	srcSandbox, err := safedisk.NewNoOpSandbox(filepath.Join(h.TestCase.Path, "src"), safedisk.ModeReadOnly)
	if err != nil {
		return fmt.Errorf("creating src sandbox: %w", err)
	}
	h.ActionCtx = &browserpkg.ActionContext{
		Ctx:        h.PageHelper.Ctx(),
		SrcSandbox: srcSandbox,
		PageHelper: h.PageHelper,
		ServerURL:  h.ServerURL,
	}

	return nil
}

func (h *ExamplesHarness) CloseBrowser() {
	if h.PageHelper != nil {
		h.PageHelper.Close()
	}
	if h.IncognitoPage != nil {
		_ = h.IncognitoPage.CloseContext()
	}
}

func (h *ExamplesHarness) NavigateToPage() error {
	h.T.Helper()

	url := h.ServerURL + h.Spec.RequestURL
	h.T.Logf("Navigating to %s", url)

	return h.PageHelper.Navigate(url)
}

func (h *ExamplesHarness) ReadConsoleLogs() []string {
	return h.PageHelper.ConsoleLogs()
}

func (h *ExamplesHarness) ClearConsoleLogs() {
	h.PageHelper.ClearConsoleLogs()
}

func (h *ExamplesHarness) GoldenPath(filename string) string {
	return filepath.Join(h.GoldenDir, filename)
}

func (h *ExamplesHarness) Cleanup() {
	h.CloseBrowser()
	h.StopServer()
}

func runExampleTestCase(t *testing.T, tc testCase) {
	t.Helper()

	h := NewExamplesHarness(t, tc)
	defer h.Cleanup()

	require.NoError(t, h.Setup(), "Failed to setup harness")

	err := h.BuildServer()
	if h.Spec.ShouldError && err == nil {
		return
	}
	require.NoError(t, err, "Failed to build server")

	require.NoError(t, h.StartServer(), "Failed to start server")

	require.NoError(t, h.SetupBrowser(), "Failed to setup browser")

	require.NoError(t, h.NavigateToPage(), "Failed to navigate to page")

	if len(h.Spec.BrowserSteps) > 0 {
		for i, step := range h.Spec.BrowserSteps {
			t.Logf("  Step %d: %s", i+1, step.Action)

			if err := executeStep(h, step); err != nil {
				if consoleLogs := h.ReadConsoleLogs(); len(consoleLogs) > 0 {
					t.Logf("  Browser console logs:")
					for _, log := range consoleLogs {
						t.Logf("    %s", log)
					}
				}
				t.Fatalf("Step %d (%s) failed: %v", i+1, step.Action, err)
			}
		}
	}
}

func executeStep(h *ExamplesHarness, step browserpkg.BrowserStep) error {

	if step.Action == "captureDOM" {
		return actionCaptureDOM(h, step)
	}

	if browserpkg.IsAssertionAction(step.Action) {
		return browserpkg.ExecuteAssertion(h.ActionCtx, &step)
	}

	return browserpkg.ExecuteStep(h.ActionCtx, &step)
}

func actionCaptureDOM(h *ExamplesHarness, step browserpkg.BrowserStep) error {

	time.Sleep(50 * time.Millisecond)

	html, err := browserpkg.CaptureDOM(h.ActionCtx, step.Selector, !step.ExcludeShadowRoots)
	if err != nil {
		return err
	}

	normalised := browserpkg.NormaliseDOM(html, browserpkg.DefaultNormaliseOptions())
	goldenPath := h.GoldenPath(step.GoldenFile)

	if *updateGoldenFiles {
		h.T.Logf("  Updating golden file: %s", goldenPath)
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			return fmt.Errorf("creating golden directory: %w", err)
		}
		if err := os.WriteFile(goldenPath, []byte(normalised), 0644); err != nil {
			return fmt.Errorf("writing golden file: %w", err)
		}
	} else {
		expected, err := os.ReadFile(goldenPath)
		if err != nil {
			return fmt.Errorf("reading golden file %s: %w (run with -update flag to create it)", goldenPath, err)
		}

		if !bytes.Equal(expected, []byte(normalised)) {
			diffCmd := exec.Command("diff", "-u", goldenPath, "-")
			diffCmd.Stdin = strings.NewReader(normalised)
			diffOutput, _ := diffCmd.CombinedOutput()

			h.T.Errorf("DOM mismatch for %s:\n%s", step.GoldenFile, string(diffOutput))
		}
	}

	h.T.Logf("  Captured DOM to '%s'", step.GoldenFile)
	return nil
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
