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

package layouter_test

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

type layouterHarness struct {
	t               *testing.T
	testCase        testCase
	spec            *layouterTestSpec
	tempDirectory   string
	sourceDirectory string
	serverCommand   *exec.Cmd
	serverPort      int
	serverURL       string
	incognitoPage   *browserpkg.IncognitoPage
	pageHelper      *browserpkg.PageHelper
	actionContext   *browserpkg.ActionContext
}

var serverMainTemplate = template.Must(template.New("main.go").Parse(`package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"piko.sh/piko"
	"piko.sh/piko/wdk/runtime"
	_ "{{.ModuleName}}/dist"
)

func main() {
	if os.Getenv("PIKO_EXTRACT_POSITIONS") == "1" {
		extractPositions()
		return
	}

	portString := os.Getenv("PIKO_TEST_PORT")
	if portString == "" {
		portString = "8080"
	}
	port, _ := strconv.Atoi(portString)

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
	)
	server.Configure(piko.PublicConfig{
		BaseDir:        ".",
		PagesSourceDir: "pages",
		Port:           port,
	})

	fmt.Printf("Test server starting on port %s\n", portString)
	if err := server.Run(piko.RunModeProd); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func extractPositions() {
	manifestPath := os.Getenv("PIKO_MANIFEST_PATH")
	requestPath := os.Getenv("PIKO_REQUEST_PATH")
	fontPath := os.Getenv("PIKO_FONT_PATH")
	viewportWidth, _ := strconv.Atoi(os.Getenv("PIKO_VIEWPORT_WIDTH"))
	viewportHeight, _ := strconv.Atoi(os.Getenv("PIKO_VIEWPORT_HEIGHT"))

	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read font file: %v\n", err)
		os.Exit(1)
	}

	config := runtime.LayoutPositionConfig{
		ManifestPath:   manifestPath,
		RequestPath:    requestPath,
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
		FontData:       fontData,
		AttributeName:  "data-layout-id",
	}

	if os.Getenv("PIKO_PAGINATE") == "1" {
		config.Paginate = true
		config.PageWidthPx, _ = strconv.ParseFloat(os.Getenv("PIKO_PAGE_WIDTH_PX"), 64)
		config.PageHeightPx, _ = strconv.ParseFloat(os.Getenv("PIKO_PAGE_HEIGHT_PX"), 64)
		config.PageMarginPx, _ = strconv.ParseFloat(os.Getenv("PIKO_PAGE_MARGIN_PX"), 64)
	}

	if extraCSS := os.Getenv("PIKO_EXTRA_CSS"); extraCSS != "" {
		config.ExtraStylesheets = append(config.ExtraStylesheets, extraCSS)
	}

	positions, err := runtime.ExtractLayoutPositions(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "layout extraction failed: %v\n", err)
		os.Exit(1)
	}

	if err := json.NewEncoder(os.Stdout).Encode(positions); err != nil {
		fmt.Fprintf(os.Stderr, "JSON encode failed: %v\n", err)
		os.Exit(1)
	}
}
`))

func newLayouterHarness(t *testing.T, tc testCase) *layouterHarness {
	t.Helper()

	return &layouterHarness{
		t:               t,
		testCase:        tc,
		sourceDirectory: filepath.Join(tc.Path, "src"),
	}
}

func (h *layouterHarness) setup() error {
	resetGlobalStateForTestIsolation()

	specPath := filepath.Join(h.testCase.Path, "testspec.json")
	spec, err := loadTestSpec(specPath)
	if err != nil {
		return fmt.Errorf("loading test spec: %w", err)
	}
	h.spec = spec

	h.tempDirectory = h.t.TempDir()

	return nil
}

func (h *layouterHarness) buildServer() error {
	h.t.Helper()

	absoluteSourceDirectory, err := filepath.Abs(h.sourceDirectory)
	if err != nil {
		return fmt.Errorf("getting absolute src path: %w", err)
	}

	pikoDirectory := filepath.Join(absoluteSourceDirectory, ".piko")
	if _, err := os.Stat(pikoDirectory); err == nil {
		if err := os.RemoveAll(pikoDirectory); err != nil {
			return fmt.Errorf("removing .piko directory: %w", err)
		}
	}

	sourceDistDirectory := filepath.Join(absoluteSourceDirectory, "dist")
	if _, err := os.Stat(sourceDistDirectory); err == nil {
		if err := os.RemoveAll(sourceDistDirectory); err != nil {
			return fmt.Errorf("removing dist directory: %w", err)
		}
	}

	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := os.Chdir(absoluteSourceDirectory); err != nil {
		return fmt.Errorf("changing to source directory: %w", err)
	}

	sourceTidyCommand := exec.Command("go", "mod", "tidy")
	sourceTidyCommand.Dir = absoluteSourceDirectory
	sourceTidyCommand.Env = append(os.Environ(), "GOWORK=off")
	if output, err := sourceTidyCommand.CombinedOutput(); err != nil {
		_ = os.Chdir(originalWorkingDirectory)
		return fmt.Errorf("go mod tidy in source directory failed: %w\nOutput: %s", err, string(output))
	}

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
	)
	err = server.Generate(context.Background(), piko.GenerateModeAll)
	server.Close()

	if restoreError := os.Chdir(originalWorkingDirectory); restoreError != nil {
		return fmt.Errorf("restoring working directory: %w", restoreError)
	}

	if err != nil {
		return fmt.Errorf("piko.Generate failed: %w", err)
	}

	if err := copyDirectory(absoluteSourceDirectory, h.tempDirectory); err != nil {
		return fmt.Errorf("copying source to temp: %w", err)
	}

	dependencyFile := filepath.Join(h.tempDirectory, "piko_dependency.go")
	if _, err := os.Stat(dependencyFile); err == nil {
		if err := os.Remove(dependencyFile); err != nil {
			return fmt.Errorf("removing piko_dependency.go: %w", err)
		}
	}

	pikoProjectRoot, err := findPikoProjectRoot()
	if err != nil {
		return fmt.Errorf("finding piko project root: %w", err)
	}

	goModPath := filepath.Join(h.tempDirectory, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	goModString := string(goModContent)
	replacePattern := regexp.MustCompile(`replace piko\.sh/piko => \.\./[^\n]+`)
	updatedGoModContent := replacePattern.ReplaceAllString(goModString,
		fmt.Sprintf("replace piko.sh/piko => %s", pikoProjectRoot))

	if err := os.WriteFile(goModPath, []byte(updatedGoModContent), 0644); err != nil {
		return fmt.Errorf("updating go.mod: %w", err)
	}

	goWorkContent := fmt.Sprintf("go 1.25.1\n\nuse (\n\t.\n\t%s\n)\n", pikoProjectRoot)
	if err := os.WriteFile(filepath.Join(h.tempDirectory, "go.work"), []byte(goWorkContent), 0644); err != nil {
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

	mainGoPath := filepath.Join(h.tempDirectory, "main.go")
	mainGoFile, err := os.Create(mainGoPath)
	if err != nil {
		return fmt.Errorf("creating main.go: %w", err)
	}
	defer func() { _ = mainGoFile.Close() }()

	if err := serverMainTemplate.Execute(mainGoFile, map[string]string{"ModuleName": moduleName}); err != nil {
		return fmt.Errorf("executing main.go template: %w", err)
	}

	tidyCommand := exec.Command("go", "mod", "tidy")
	tidyCommand.Dir = h.tempDirectory
	tidyCommand.Env = append(os.Environ(), "GOWORK=off")
	if output, err := tidyCommand.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w\nOutput: %s", err, string(output))
	}

	buildCommand := exec.Command("go", "build", "-o", "server", ".")
	buildCommand.Dir = h.tempDirectory
	buildCommand.Env = append(os.Environ(), "GOWORK=off")
	if output, err := buildCommand.CombinedOutput(); err != nil {
		return fmt.Errorf("go build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (h *layouterHarness) startServer() error {
	h.t.Helper()

	port, err := browserpkg.FindAvailablePort()
	require.NoError(h.t, err)
	h.serverPort = port
	h.serverURL = fmt.Sprintf("http://localhost:%d", h.serverPort)

	h.serverCommand = exec.Command("./server")
	h.serverCommand.Dir = h.tempDirectory
	h.serverCommand.Env = append(os.Environ(),
		fmt.Sprintf("PIKO_TEST_PORT=%d", h.serverPort),
		fmt.Sprintf("PIKO_PORT=%d", h.serverPort),
		"PIKO_RATE_LIMIT_ENABLED=true",
	)

	var stdout, stderr bytes.Buffer
	h.serverCommand.Stdout = &stdout
	h.serverCommand.Stderr = &stderr

	if err := h.serverCommand.Start(); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	if err := browserpkg.WaitForServerReady(h.serverPort, 30*time.Second); err != nil {
		return fmt.Errorf("server not ready: %w\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

func (h *layouterHarness) setupBrowser() error {
	h.t.Helper()

	incognitoPage, err := browser.NewIncognitoPage()
	if err != nil {
		return fmt.Errorf("creating incognito page: %w", err)
	}
	h.incognitoPage = incognitoPage
	h.pageHelper = browserpkg.NewPageHelper(incognitoPage.Ctx)

	sourceDirectory := filepath.Join(h.testCase.Path, "src")
	sourceSandbox, err := safedisk.NewNoOpSandbox(sourceDirectory, safedisk.ModeReadOnly)
	if err != nil {
		return fmt.Errorf("creating source sandbox: %w", err)
	}

	h.actionContext = &browserpkg.ActionContext{
		Ctx:        h.pageHelper.Ctx(),
		SrcSandbox: sourceSandbox,
		PageHelper: h.pageHelper,
		ServerURL:  h.serverURL,
	}

	return nil
}

func (h *layouterHarness) navigateToPage() error {
	h.t.Helper()

	url := h.serverURL + h.spec.RequestURL
	h.t.Logf("navigating to %s", url)

	return h.pageHelper.Navigate(url)
}

func (h *layouterHarness) cleanup() {
	if h.pageHelper != nil {
		h.pageHelper.Close()
	}
	if h.incognitoPage != nil {
		_ = h.incognitoPage.CloseContext()
	}
	if h.serverCommand != nil && h.serverCommand.Process != nil {
		_ = h.serverCommand.Process.Kill()
		_, _ = h.serverCommand.Process.Wait()
	}
}

func copyDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, directory os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destinationPath := filepath.Join(destination, relativePath)
		if directory.IsDir() {
			return os.MkdirAll(destinationPath, 0755)
		}
		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = sourceFile.Close() }()
		destinationFile, err := os.Create(destinationPath)
		if err != nil {
			return err
		}
		defer func() { _ = destinationFile.Close() }()
		_, err = io.Copy(destinationFile, sourceFile)
		return err
	})
}

func findPikoProjectRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(directory, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			content, readError := os.ReadFile(goModPath)
			if readError == nil && strings.Contains(string(content), "module piko.sh/piko\n") {
				return directory, nil
			}
		}

		parentDirectory := filepath.Dir(directory)
		if parentDirectory == directory {
			return "", errors.New("could not find piko project root")
		}
		directory = parentDirectory
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
