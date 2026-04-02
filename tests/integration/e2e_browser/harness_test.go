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

package e2e_browser_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/markdown/markdown_provider_goldmark"
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

type E2EBrowserHarness struct {
	T             *testing.T
	TestCase      testCase
	Spec          *E2ETestSpec
	TempDir       string
	SrcDir        string
	GoldenDir     string
	ServerCmd     *exec.Cmd
	ServerPort    int
	ServerURL     string
	PageHelper    *browserpkg.PageHelper
	IncognitoPage *browserpkg.IncognitoPage
	ActionCtx     *browserpkg.ActionContext
}

var mainGoTemplate = template.Must(template.New("main.go").Parse(`package main

import (
	"fmt"
	"os"
	"strconv"

	"piko.sh/piko"
	_ "{{.ModuleName}}/dist"
{{- if .RequiresMarkdown}}
	"piko.sh/piko/wdk/markdown/markdown_provider_goldmark"
{{- end}}
)

func main() {
	portString := os.Getenv("PIKO_TEST_PORT")
	if portString == "" {
		portString = "8080"
	}
	port, _ := strconv.Atoi(portString)

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
{{- if .RequiresMarkdown}}
		piko.WithMarkdownParser(markdown_provider_goldmark.NewParser()),
{{- end}}
	)
	server.Configure(piko.PublicConfig{
		BaseDir:        ".",
		PagesSourceDir: "pages",
		Port:           port,
	})

	fmt.Printf("Test server starting on port %s\n", portString)
	// Use Run() with prod mode which sets up all routes properly
	if err := server.Run(piko.RunModeProd); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
`))

func NewE2EBrowserHarness(t *testing.T, tc testCase) *E2EBrowserHarness {
	t.Helper()

	return &E2EBrowserHarness{
		T:         t,
		TestCase:  tc,
		SrcDir:    filepath.Join(tc.Path, "src"),
		GoldenDir: filepath.Join(tc.Path, "golden"),
	}
}

func (h *E2EBrowserHarness) Setup() error {

	resetGlobalStateForTestIsolation()

	specPath := filepath.Join(h.TestCase.Path, "testspec.json")
	spec, err := LoadE2ETestSpec(specPath)
	if err != nil {
		return fmt.Errorf("loading test spec: %w", err)
	}
	h.Spec = spec

	h.TempDir = h.T.TempDir()

	return nil
}

func (h *E2EBrowserHarness) BuildServer() error {
	h.T.Helper()

	absSrcDir, err := filepath.Abs(h.SrcDir)
	if err != nil {
		return fmt.Errorf("getting absolute src path: %w", err)
	}

	pikoDir := filepath.Join(absSrcDir, ".piko")
	if _, err := os.Stat(pikoDir); err == nil {
		if err := os.RemoveAll(pikoDir); err != nil {
			return fmt.Errorf("removing .piko directory: %w", err)
		}
	}

	srcDistDir := filepath.Join(absSrcDir, "dist")
	if _, err := os.Stat(srcDistDir); err == nil {
		if err := os.RemoveAll(srcDistDir); err != nil {
			return fmt.Errorf("removing dist directory: %w", err)
		}
	}

	originalWd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := os.Chdir(absSrcDir); err != nil {
		return fmt.Errorf("changing to source directory: %w", err)
	}

	srcTidyCmd := exec.Command("go", "mod", "tidy")
	srcTidyCmd.Dir = absSrcDir
	srcTidyCmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := srcTidyCmd.CombinedOutput(); err != nil {
		_ = os.Chdir(originalWd)
		return fmt.Errorf("go mod tidy in source directory failed: %w\nOutput: %s", err, string(output))
	}

	serverOptions := []piko.Option{
		piko.WithCSSReset(piko.WithCSSResetComplete()),
	}
	if h.Spec.RequiresMarkdown {
		serverOptions = append(serverOptions, piko.WithMarkdownParser(markdown_provider_goldmark.NewParser()))
	}
	server := piko.New(serverOptions...)
	err = server.Generate(context.Background(), piko.GenerateModeAll)
	server.Close()

	if restoreErr := os.Chdir(originalWd); restoreErr != nil {
		return fmt.Errorf("restoring working directory: %w", restoreErr)
	}

	if err != nil {
		if h.Spec.ShouldError {
			if h.Spec.ErrorContains != "" && !strings.Contains(err.Error(), h.Spec.ErrorContains) {
				return fmt.Errorf("expected error containing %q, got: %w", h.Spec.ErrorContains, err)
			}
			return nil
		}
		return fmt.Errorf("piko.Generate failed: %w", err)
	}

	if err := copyDir(absSrcDir, h.TempDir); err != nil {
		return fmt.Errorf("copying source to temp: %w", err)
	}

	depFile := filepath.Join(h.TempDir, "piko_dependency.go")
	if _, err := os.Stat(depFile); err == nil {
		if err := os.Remove(depFile); err != nil {
			return fmt.Errorf("removing piko_dependency.go: %w", err)
		}
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
	replacePattern := regexp.MustCompile(`replace piko\.sh/piko => \.\./[^\n]+`)
	updatedGoModContent := replacePattern.ReplaceAllString(goModString,
		fmt.Sprintf("replace piko.sh/piko => %s", pikoProjectRoot))

	if h.Spec.RequiresMarkdown {
		goldmarkModPath := filepath.Join(pikoProjectRoot, "wdk", "markdown", "markdown_provider_goldmark")
		updatedGoModContent += "\nrequire piko.sh/piko/wdk/markdown/markdown_provider_goldmark v0.0.0\n"
		updatedGoModContent += fmt.Sprintf("replace piko.sh/piko/wdk/markdown/markdown_provider_goldmark => %s\n", goldmarkModPath)
	}

	if err := os.WriteFile(goModPath, []byte(updatedGoModContent), 0644); err != nil {
		return fmt.Errorf("updating go.mod: %w", err)
	}

	goWorkUses := fmt.Sprintf("\t.\n\t%s", pikoProjectRoot)
	if h.Spec.RequiresMarkdown {
		goldmarkPath := filepath.Join(pikoProjectRoot, "wdk", "markdown", "markdown_provider_goldmark")
		goWorkUses += fmt.Sprintf("\n\t%s", goldmarkPath)
	}
	goWorkContent := fmt.Sprintf("go 1.25.1\n\nuse (\n%s\n)\n", goWorkUses)
	if err := os.WriteFile(filepath.Join(h.TempDir, "go.work"), []byte(goWorkContent), 0644); err != nil {
		return fmt.Errorf("creating go.work: %w", err)
	}

	moduleName := ""
	for line := range strings.SplitSeq(updatedGoModContent, "\n") {
		if trimmed, found := strings.CutPrefix(line, "module "); found {
			moduleName = strings.TrimSpace(trimmed)
			break
		}
	}
	if moduleName == "" {
		return errors.New("could not find module name in go.mod")
	}

	mainGoPath := filepath.Join(h.TempDir, "main.go")
	mainGoFile, err := os.Create(mainGoPath)
	if err != nil {
		return fmt.Errorf("creating main.go: %w", err)
	}
	defer func() { _ = mainGoFile.Close() }()

	if err := mainGoTemplate.Execute(mainGoFile, map[string]any{"ModuleName": moduleName, "RequiresMarkdown": h.Spec.RequiresMarkdown}); err != nil {
		return fmt.Errorf("executing main.go template: %w", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = h.TempDir
	tidyCmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w\nOutput: %s", err, string(output))
	}

	buildCmd := exec.Command("go", "build", "-o", "server", ".")
	buildCmd.Dir = h.TempDir
	buildCmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (h *E2EBrowserHarness) StartServer() error {
	h.T.Helper()

	h.ServerPort = findAvailablePort(h.T)
	h.ServerURL = fmt.Sprintf("http://localhost:%d", h.ServerPort)

	h.ServerCmd = exec.Command("./server")
	h.ServerCmd.Dir = h.TempDir
	h.ServerCmd.Env = append(os.Environ(),
		fmt.Sprintf("PIKO_TEST_PORT=%d", h.ServerPort),
		fmt.Sprintf("PIKO_PORT=%d", h.ServerPort),
		"PIKO_RATE_LIMIT_ENABLED=true",
	)

	var stdout, stderr bytes.Buffer
	h.ServerCmd.Stdout = &stdout
	h.ServerCmd.Stderr = &stderr

	if err := h.ServerCmd.Start(); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	if err := waitForServerReady(h.T, h.ServerPort, 30*time.Second); err != nil {
		return fmt.Errorf("server not ready: %w\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

func (h *E2EBrowserHarness) StopServer() {
	if h.ServerCmd != nil && h.ServerCmd.Process != nil {
		_ = h.ServerCmd.Process.Kill()
		_, _ = h.ServerCmd.Process.Wait()
	}
}

func (h *E2EBrowserHarness) SetupBrowser() error {
	h.T.Helper()

	incognitoPage, err := browser.NewIncognitoPage()
	if err != nil {
		return fmt.Errorf("creating incognito page: %w", err)
	}
	h.IncognitoPage = incognitoPage
	h.PageHelper = browserpkg.NewPageHelper(incognitoPage.Ctx)
	srcSandbox, err := safedisk.NewNoOpSandbox(filepath.Join(h.TestCase.Path, "src"), safedisk.ModeReadOnly)
	if err != nil {
		return fmt.Errorf("creating source sandbox: %w", err)
	}

	h.ActionCtx = &browserpkg.ActionContext{
		Ctx:        h.PageHelper.Ctx(),
		SrcSandbox: srcSandbox,
		PageHelper: h.PageHelper,
		ServerURL:  h.ServerURL,
	}

	return nil
}

func (h *E2EBrowserHarness) CloseBrowser() {
	if h.PageHelper != nil {
		h.PageHelper.Close()
	}
	if h.IncognitoPage != nil {
		_ = h.IncognitoPage.CloseContext()
	}
}

func (h *E2EBrowserHarness) NavigateToPage() error {
	h.T.Helper()

	url := h.ServerURL + h.Spec.RequestURL
	h.T.Logf("Navigating to %s", url)

	return h.PageHelper.Navigate(url)
}

func (h *E2EBrowserHarness) ReadConsoleLogs() []string {
	return h.PageHelper.ConsoleLogs()
}

func (h *E2EBrowserHarness) ClearConsoleLogs() {
	h.PageHelper.ClearConsoleLogs()
}

func (h *E2EBrowserHarness) GoldenPath(filename string) string {
	return filepath.Join(h.GoldenDir, filename)
}

func (h *E2EBrowserHarness) Cleanup() {
	h.CloseBrowser()
	h.StopServer()
}

func runE2ETestCase(t *testing.T, tc testCase) {
	t.Helper()

	h := NewE2EBrowserHarness(t, tc)
	defer h.Cleanup()

	require.NoError(t, h.Setup(), "Failed to setup harness")

	err := h.BuildServer()
	if h.Spec.ShouldError && err == nil {
		return
	}
	require.NoError(t, err, "Failed to build server")

	require.NoError(t, h.StartServer(),
		"Failed to start server")

	require.NoError(t, h.SetupBrowser(),
		"Failed to setup browser")

	require.NoError(t, h.NavigateToPage(),
		"Failed to navigate to page")

	if len(h.Spec.BrowserSteps) > 0 {

		if interactive {

			if err := runInteractiveMode(h); err != nil {

				if err.Error() == "test aborted by user" {
					t.Skip("Test aborted by user")
				}
				t.Fatalf("Interactive mode failed: %v", err)
			}
		} else {

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
	port, err := browserpkg.FindAvailablePort()
	require.NoError(t, err)
	return port
}

func waitForServerReady(t *testing.T, port int, timeout time.Duration) error {
	t.Helper()
	return browserpkg.WaitForServerReady(port, timeout)
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

			if readErr == nil && strings.Contains(string(content), "module piko.sh/piko\n") {
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
