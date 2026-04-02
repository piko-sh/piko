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
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

const (
	analysisTimeout = 60 * time.Second
	requestTimeout  = 30 * time.Second
	testTimeout     = 5 * time.Minute
)

var (
	lspBinaryOnce sync.Once
	lspBinaryPath string
	lspBuildErr   error
)

func buildLSPBinary(t *testing.T) string {
	t.Helper()

	lspBinaryOnce.Do(func() {
		tmpDir := os.TempDir()
		lspBinaryPath = filepath.Join(tmpDir, "piko-lsp-stress-test")

		lspSrcDir := filepath.Join("..", "..", "..", "cmd", "lsp")
		absLspSrcDir, err := filepath.Abs(lspSrcDir)
		if err != nil {
			lspBuildErr = err
			return
		}

		cmd := exec.Command("go", "build", "-o", lspBinaryPath, ".")
		cmd.Dir = absLspSrcDir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			lspBuildErr = &buildError{output: output, err: err}
		}
	})

	require.NoError(t, lspBuildErr, "building piko-lsp binary")
	t.Cleanup(func() {

	})

	return lspBinaryPath
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
	t.Helper()
	buildLSPBinary(t)

	return &stressHarness{
		t:      t,
		srcDir: copyFixtureToTemp(t),
	}
}

func (h *stressHarness) startSession() (*stressClient, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, lspBinaryPath)
	cmd.Dir = h.srcDir
	cmd.Stderr = os.Stderr

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
	_, initErr := client.Initialize(ctx, rootURI)
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

func copyFixtureToTemp(t *testing.T) string {
	t.Helper()

	fixtureDir := filepath.Join("testdata", "stress_project", "src")
	tmpDir := t.TempDir()
	dstDir := filepath.Join(tmpDir, "src")

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
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}
