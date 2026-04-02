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

package browser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
	"piko.sh/piko/wdk/safedisk"
)

var (
	globalHarness *Harness

	globalMu sync.RWMutex
)

// Harness manages the test environment for end-to-end browser tests.
// Create one in TestMain and call Setup before running tests.
type Harness struct {
	// setupErr stores any error that occurs during setup; protected by setupOnce.
	setupErr error

	// browser is the headless browser for UI testing; nil until Setup is called.
	browser *browser_provider_chromedp.Browser

	// serverCommand is the running server process.
	serverCommand *exec.Cmd

	// serverURL is the base URL of the running test server.
	serverURL string

	// tempDir is the path to a temporary directory for build files.
	tempDir string

	// srcSandbox provides sandboxed access to the project source directory.
	srcSandbox safedisk.Sandbox

	// opts holds the settings for this harness instance.
	opts harnessOptions

	// serverPort is the TCP port number for the test server.
	serverPort int

	// setupOnce guards single execution of Setup, even with concurrent calls.
	setupOnce sync.Once

	// mu guards the browser and its state during cleanup and page creation.
	mu sync.Mutex
}

// NewHarness creates a new end-to-end test harness.
//
// Takes opts (...HarnessOption) which sets the harness behaviour.
//
// Returns *Harness which is the test harness ready for use.
//
// Safe for concurrent use. Sets itself as the global harness using a mutex.
func NewHarness(opts ...HarnessOption) *Harness {
	options := defaultHarnessOptions()
	for _, opt := range opts {
		opt(&options)
	}

	h := &Harness{
		setupErr:      nil,
		browser:       nil,
		serverCommand: nil,
		serverURL:     "",
		tempDir:       "",
		opts:          options,
		serverPort:    0,
		setupOnce:     sync.Once{},
		mu:            sync.Mutex{},
	}

	globalMu.Lock()
	globalHarness = h
	globalMu.Unlock()

	return h
}

// Setup builds the project and starts the server.
// Safe to call many times; uses sync.Once internally.
//
// Returns error when the build or server start fails.
func (h *Harness) Setup() error {
	h.setupOnce.Do(func() {
		h.setupErr = h.doSetup()
	})
	return h.setupErr
}

// Cleanup stops the server and closes the browser.
// Call this in defer after NewHarness.
//
// Safe for concurrent use. Acquires the harness mutex before releasing
// resources.
func (h *Harness) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.browser != nil {
		h.browser.Close()
		h.browser = nil
	}

	h.stopServer()

	if h.srcSandbox != nil {
		_ = h.srcSandbox.Close()
		h.srcSandbox = nil
	}

	if h.tempDir != "" {
		_ = os.RemoveAll(h.tempDir)
		h.tempDir = ""
	}

	globalMu.Lock()
	if globalHarness == h {
		globalHarness = nil
	}
	globalMu.Unlock()
}

// ServerURL returns the base URL of the running server.
//
// Returns string which is the base URL for making requests to the test server.
func (h *Harness) ServerURL() string {
	return h.serverURL
}

// IsInteractive returns whether interactive mode is enabled.
//
// Returns bool which is true when interactive mode is enabled.
func (h *Harness) IsInteractive() bool {
	return h.opts.interactive
}

// doSetup performs the actual setup work.
//
// Returns error when the project path cannot be resolved, the temp directory
// cannot be created, the project build fails, no port is available, the server
// fails to start, or the browser fails to start.
func (h *Harness) doSetup() error {
	absProjectDir, err := filepath.Abs(h.opts.projectDir)
	if err != nil {
		return fmt.Errorf("getting absolute project path: %w", err)
	}

	if h.opts.sandboxFactory != nil {
		h.srcSandbox, err = h.opts.sandboxFactory.Create("browser-harness-source", absProjectDir, safedisk.ModeReadOnly)
	} else {
		h.srcSandbox, err = safedisk.NewNoOpSandbox(absProjectDir, safedisk.ModeReadOnly)
	}
	if err != nil {
		return fmt.Errorf("creating source sandbox: %w", err)
	}

	h.tempDir, err = os.MkdirTemp("", "piko-e2e-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}

	if !h.opts.skipBuild {
		if err := h.buildProject(absProjectDir); err != nil {
			return fmt.Errorf("building project: %w", err)
		}
	}

	if h.opts.port == 0 {
		h.serverPort, err = browser_provider_chromedp.FindAvailablePort()
		if err != nil {
			return fmt.Errorf("finding available port: %w", err)
		}
	} else {
		h.serverPort = h.opts.port
	}
	h.serverURL = fmt.Sprintf("http://localhost:%d", h.serverPort)

	if err := h.startServer(absProjectDir); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	browserOpts := browser_provider_chromedp.BrowserOptions{
		Headless: h.opts.headless,
	}
	h.browser, err = browser_provider_chromedp.NewBrowser(browserOpts)
	if err != nil {
		h.stopServer()
		return fmt.Errorf("starting browser: %w", err)
	}

	return nil
}

// buildProject runs the piko generate command using go run.
//
// Takes projectDir (string) which specifies the path to the project directory.
//
// Returns error when the piko generate command fails.
func (*Harness) buildProject(projectDir string) error {
	command := exec.Command("go", "run", "./cmd/generator/main.go", "all")
	command.Dir = projectDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	command.Env = append(os.Environ(), "PIKO_E2E_MODE=true")

	if err := command.Run(); err != nil {
		return fmt.Errorf("piko generate failed: %w", err)
	}

	return nil
}

// startServer starts the piko server.
//
// Takes projectDir (string) which specifies the project root folder.
//
// Returns error when the server fails to start or does not become ready.
func (h *Harness) startServer(projectDir string) error {
	var commandArgs []string

	if len(h.opts.serverCommand) > 0 {
		commandArgs = h.opts.serverCommand
	} else {
		binaryPath := filepath.Join(projectDir, "server")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			commandArgs = []string{"go", "run", "./cmd/server"}
		} else {
			commandArgs = []string{binaryPath}
		}
	}

	commandArgs = append(commandArgs, h.opts.serverArgs...)

	execPath, err := exec.LookPath(commandArgs[0])
	if err != nil {
		return fmt.Errorf("command %q not found: %w", commandArgs[0], err)
	}

	h.serverCommand = exec.Command(execPath, commandArgs[1:]...) //nolint:gosec // test harness, developer-controlled path
	h.serverCommand.Dir = projectDir

	h.serverCommand.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	env := append(os.Environ(),
		"PIKO_E2E_MODE=true",
		fmt.Sprintf("PIKO_PORT=%d", h.serverPort),
	)
	for key, value := range h.opts.env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	h.serverCommand.Env = env

	if err := h.serverCommand.Start(); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	if err := browser_provider_chromedp.WaitForServerReady(h.serverPort, 30*time.Second); err != nil {
		h.stopServer()
		return fmt.Errorf("server not ready: %w", err)
	}

	return nil
}

// stopServer stops the server process and all its child processes.
func (h *Harness) stopServer() {
	if h.serverCommand != nil && h.serverCommand.Process != nil {
		pgid, err := syscall.Getpgid(h.serverCommand.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = h.serverCommand.Process.Kill()
		}
		_, _ = h.serverCommand.Process.Wait()
	}
}

// newPage creates a new isolated browser page for a test.
//
// Takes t (testing.TB) which receives test failures and logs.
//
// Returns *Page which is the configured page ready for browser interaction.
// Returns error when the harness has not been set up or page creation fails.
//
// Safe for concurrent use. Uses a mutex to protect access to the browser.
func (h *Harness) newPage(t testing.TB) (*Page, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.browser == nil {
		return nil, errors.New("harness not set up - call Setup() first")
	}

	incognitoPage, err := h.browser.NewIncognitoPage()
	if err != nil {
		return nil, fmt.Errorf("creating browser page: %w", err)
	}

	var outputSandbox safedisk.Sandbox
	if h.opts.sandboxFactory != nil {
		outputSandbox, err = h.opts.sandboxFactory.Create("browser-harness-output", h.opts.outputDir, safedisk.ModeReadWrite)
	} else {
		outputSandbox, err = safedisk.NewNoOpSandbox(h.opts.outputDir, safedisk.ModeReadWrite)
	}
	if err != nil {
		_ = incognitoPage.CloseContext()
		return nil, fmt.Errorf("creating output sandbox: %w", err)
	}

	page := &Page{
		t:                 t,
		interactiveRunner: nil,
		incognitoPage:     incognitoPage,
		harness:           h,
		pageHelper:        browser_provider_chromedp.NewPageHelper(incognitoPage.Ctx),
		dialogHandler:     nil,
		outputSandbox:     outputSandbox,
		baseURL:           h.serverURL,
		currentPath:       "",
	}

	if h.opts.interactive {
		page.enableInteractive(h.opts.interactiveTUI)
	}

	return page, nil
}

// New creates a browser page for a test using the global harness set up in
// TestMain. Each test should call this to get its own page.
//
// Takes t (testing.TB) which is the test context for error reporting.
//
// Returns *Page which is the browser page for the test.
//
// Safe for use by multiple goroutines at the same time.
func New(t testing.TB) *Page {
	globalMu.RLock()
	h := globalHarness
	globalMu.RUnlock()

	if h == nil {
		t.Fatal("browser: no harness available - create one in TestMain and call Setup()")
	}

	page, err := h.newPage(t)
	if err != nil {
		t.Fatalf("browser: failed to create page: %v", err)
	}

	return page
}
