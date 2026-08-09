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

package lsp_stress_test

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/uri"
)

const (
	analysisTimeout = 60 * time.Second
	requestTimeout  = 30 * time.Second
	testTimeout     = 5 * time.Minute
)

var (
	lspBinaryOnce sync.Once
	lspBinaryDir  string
	lspBinaryPath string
	lspBuildErr   error
)

func buildLSPBinary(t *testing.T) string {
	t.Helper()

	lspBinaryOnce.Do(func() {
		lspBinaryDir, lspBuildErr = os.MkdirTemp("", "pikopls-stress-test")
		if lspBuildErr != nil {
			return
		}
		lspBinaryPath = filepath.Join(lspBinaryDir, "pikopls")

		repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
		if err != nil {
			lspBuildErr = err
			return
		}
		absLspSrcDir := filepath.Join(repoRoot, "cmd", "pikopls")
		goWorkPath := filepath.Join(repoRoot, "go.work")

		cmd := exec.Command("go", "build", "-o", lspBinaryPath, ".")
		cmd.Dir = absLspSrcDir

		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK="+goWorkPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			lspBuildErr = &buildError{output: output, err: err}
		}
	})

	require.NoError(t, lspBuildErr, "building pikopls binary")

	return lspBinaryPath
}

func removeLSPBinary() {
	if lspBinaryDir == "" {
		return
	}
	if err := os.RemoveAll(lspBinaryDir); err != nil {
		fmt.Fprintf(os.Stderr, "removing pikopls build directory %s: %v\n", lspBinaryDir, err)
	}
}

type buildError struct {
	output []byte
	err    error
}

func (e *buildError) Error() string {
	return e.err.Error() + "\n" + string(e.output)
}

type stressHarness struct {
	t      *testing.T
	srcDir string
}

func newStressHarness(t *testing.T) *stressHarness {
	return newStressHarnessForFixture(t, "stress_project", t.TempDir())
}

func newStressHarnessForFixture(t *testing.T, fixtureName, destRoot string) *stressHarness {
	t.Helper()
	buildLSPBinary(t)

	return &stressHarness{
		t:      t,
		srcDir: copyFixtureToTemp(t, fixtureName, destRoot),
	}
}

var (
	fakeGoplsOnce sync.Once
	fakeGoplsPath string
	fakeGoplsErr  error
)

func requireGopls(t *testing.T) string {
	t.Helper()
	goplsPath, err := exec.LookPath("gopls")
	if err != nil {
		t.Skip("gopls not found on PATH; skipping real-gopls bridge test")
	}
	return goplsPath
}

func buildFakeGoplsBinary(t *testing.T) string {
	t.Helper()
	fakeGoplsOnce.Do(func() {
		fakeGoplsPath = filepath.Join(os.TempDir(), "piko-fakegopls")
		cmd := exec.Command("go", "build", "-o", fakeGoplsPath, "./testdata/fakegopls")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fakeGoplsErr = &buildError{output: output, err: err}
		}
	})
	require.NoError(t, fakeGoplsErr, "building fake gopls binary")
	return fakeGoplsPath
}

func spacedTempDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "dir with space")
	require.NoError(t, os.MkdirAll(dir, 0755))
	return dir
}

type sessionOptions struct {
	Args        []string
	Env         []string
	InitOptions map[string]any
}

var (
	goplsSessionSlots = make(chan struct{}, goplsTestConcurrency())
)

func goplsTestConcurrency() int {
	if raw := os.Getenv("PIKO_TEST_GOPLS_CONCURRENCY"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 4
}

func sessionUsesGoplsBridge(opts sessionOptions) bool {
	for _, arg := range opts.Args {
		if strings.HasPrefix(arg, "--gopls-bridge") {
			return true
		}
	}
	enabled, isBool := opts.InitOptions["goBridge"].(bool)
	return isBool && enabled
}

func (h *stressHarness) startSession() (*stressClient, func()) {
	return h.startSessionWithOptions(sessionOptions{})
}

func (h *stressHarness) startSessionWithOptions(opts sessionOptions) (*stressClient, func()) {
	if sessionUsesGoplsBridge(opts) {

		goplsSessionSlots <- struct{}{}
		h.t.Cleanup(func() { <-goplsSessionSlots })
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, lspBinaryPath, opts.Args...)
	cmd.Dir = h.srcDir
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), opts.Env...)

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 10 * time.Second

	stdin, err := cmd.StdinPipe()
	require.NoError(h.t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(h.t, err)

	require.NoError(h.t, cmd.Start(), "starting LSP subprocess")

	stream := jsonrpc2.NewStream(&stdioRWC{
		reader: stdout,
		writer: stdin,
	})

	client := newStressClient(h.t, stream)

	time.Sleep(200 * time.Millisecond)

	rootURI := protocol.DocumentURI(uri.File(h.srcDir))
	_, initErr := client.InitializeWithOptions(ctx, rootURI, opts.InitOptions)
	require.NoError(h.t, initErr, "LSP initialise failed")
	require.NoError(h.t, client.Initialized(ctx), "LSP initialised notification failed")

	cleanup := func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		_ = client.Shutdown(shutdownCtx)
		_ = client.Exit(shutdownCtx)
		_ = client.Close()

		cancel()
		_ = cmd.Wait()
	}

	return client, cleanup
}

func (h *stressHarness) startTCPBridgeServer(goplsPath string) (int, func()) {
	goplsSessionSlots <- struct{}{}
	h.t.Cleanup(func() { <-goplsSessionSlots })

	port := freeTCPPort(h.t)
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, lspBinaryPath,
		"--tcp", "--port", strconv.Itoa(port),
		"--gopls-bridge=true", "--gopls-path="+goplsPath)
	cmd.Dir = h.srcDir
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 10 * time.Second

	require.NoError(h.t, cmd.Start(), "starting LSP TCP server")
	waitForTCPListen(h.t, port)

	return port, func() {
		cancel()
		_ = cmd.Wait()
	}
}

func (h *stressHarness) dialTCPClient(port int) (*stressClient, func()) {
	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	require.NoError(h.t, err)

	client := newStressClient(h.t, jsonrpc2.NewStream(conn))

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	rootURI := protocol.DocumentURI(uri.File(h.srcDir))
	_, initErr := client.InitializeWithOptions(ctx, rootURI, map[string]any{"goBridge": true})
	require.NoError(h.t, initErr, "TCP client initialise failed")
	require.NoError(h.t, client.Initialized(ctx), "TCP client initialised failed")

	return client, func() { _ = client.Close() }
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok, "listener address is not TCP")
	require.NoError(t, listener.Close())
	return port.Port
}

func waitForTCPListen(t *testing.T, port int) {
	t.Helper()
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 30*time.Second, 100*time.Millisecond, "LSP TCP server did not start listening")
}

func (h *stressHarness) fileURI(relPath string) protocol.DocumentURI {
	return protocol.DocumentURI(uri.File(filepath.Join(h.srcDir, relPath)))
}

func (h *stressHarness) readFile(relPath string) string {
	data, err := os.ReadFile(filepath.Join(h.srcDir, relPath))
	require.NoError(h.t, err)
	return string(data)
}

func (h *stressHarness) writeFile(relPath string, content string) {
	err := os.WriteFile(filepath.Join(h.srcDir, relPath), []byte(content), 0644)
	require.NoError(h.t, err)
}

type stdioRWC struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (s *stdioRWC) Read(p []byte) (int, error)  { return s.reader.Read(p) }
func (s *stdioRWC) Write(p []byte) (int, error) { return s.writer.Write(p) }
func (s *stdioRWC) Close() error {
	rErr := s.reader.Close()
	wErr := s.writer.Close()
	if rErr != nil {
		return rErr
	}
	return wErr
}

func copyFixtureToTemp(t *testing.T, fixtureName, destRoot string) string {
	t.Helper()

	fixtureDir := filepath.Join("testdata", fixtureName, "src")
	dstDir := filepath.Join(destRoot, "src")

	err := filepath.WalkDir(fixtureDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, relErr := filepath.Rel(fixtureDir, path)
		if relErr != nil {
			return relErr
		}
		dstPath := filepath.Join(dstDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(dstPath, data, 0644)
	})
	require.NoError(t, err, "copying fixture to temp dir")

	repoRoot, absErr := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, absErr)

	goModPath := filepath.Join(dstDir, "go.mod")
	goModData, readErr := os.ReadFile(goModPath)
	require.NoError(t, readErr)

	fixedGoMod := replaceGoModDirective(string(goModData), "piko.sh/piko", repoRoot)
	require.NoError(t, os.WriteFile(goModPath, []byte(fixedGoMod), 0644))

	return dstDir
}

func replaceGoModDirective(goMod string, module string, absPath string) string {
	lines := splitLines(goMod)
	prefix := "replace " + module + " =>"

	var result []string
	for _, line := range lines {
		if len(line) >= len(prefix) && line[:len(prefix)] == prefix {
			result = append(result, "replace "+module+" => "+absPath)
		} else {
			result = append(result, line)
		}
	}
	return joinLines(result)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
