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

package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// BaseHarness provides common setup and helper functions for integration tests.
type BaseHarness struct {
	// T is the test context used to run subtests and log messages.
	T *testing.T

	// Spec holds the test specification for this harness.
	Spec *TestSpec

	// CleanupFunc is called after each test to release resources.
	CleanupFunc func()

	// TestCase holds the test case data for this harness.
	TestCase TestCase

	// TempDir is the path to the temporary directory for test files.
	TempDir string

	// SrcDir is the path to the source directory for the test harness.
	SrcDir string

	// GoldenDir is the path to the directory that holds golden test files.
	GoldenDir string
}

// NewBaseHarness creates a new base harness for a test case.
//
// Takes t (*testing.T) which is the test context for reporting failures.
// Takes tc (TestCase) which defines the test case settings and paths.
//
// Returns *BaseHarness which is a partly set up harness ready for further
// setup via LoadSpec and SetupTempDir.
func NewBaseHarness(t *testing.T, tc TestCase) *BaseHarness {
	t.Helper()

	h := &BaseHarness{
		T:           t,
		Spec:        nil,
		CleanupFunc: nil,
		TestCase:    tc,
		TempDir:     "",
		SrcDir:      filepath.Join(tc.Path, "src"),
		GoldenDir:   filepath.Join(tc.Path, "golden"),
	}

	return h
}

// LoadSpec loads the testspec.json file for this test case.
//
// Returns error when the spec file cannot be loaded.
func (h *BaseHarness) LoadSpec() error {
	specPath := filepath.Join(h.TestCase.Path, "testspec.json")
	spec, err := LoadTestSpec(specPath)
	if err != nil {
		return err
	}
	h.Spec = spec
	return nil
}

// SetupTempDir creates a temporary directory and copies the source files.
//
// Returns error when the temporary directory cannot be created or when copying
// the source files fails.
func (h *BaseHarness) SetupTempDir() error {
	h.T.Helper()

	tempDir, err := os.MkdirTemp("", "integration-test-*")
	if err != nil {
		return err
	}

	h.TempDir = tempDir
	h.CleanupFunc = func() {
		if h.T.Failed() && KeepStaged != nil && *KeepStaged {
			h.T.Logf("Test failed, keeping temp directory: %s", tempDir)
			return
		}
		_ = os.RemoveAll(tempDir)
	}

	return CopyDir(h.SrcDir, tempDir)
}

// Close releases all resources held by the harness.
func (h *BaseHarness) Close() {
	if h.CleanupFunc != nil {
		h.CleanupFunc()
	}
}

// AbsSrcDir returns the absolute path to the source directory.
//
// Returns string which is the resolved absolute path.
func (h *BaseHarness) AbsSrcDir() string {
	absPath, err := filepath.Abs(h.SrcDir)
	require.NoError(h.T, err)
	return absPath
}

// AbsTempDir returns the absolute path to the temp directory.
//
// Returns string which is the absolute path, or empty if TempDir is not set.
func (h *BaseHarness) AbsTempDir() string {
	if h.TempDir == "" {
		return ""
	}
	absPath, err := filepath.Abs(h.TempDir)
	require.NoError(h.T, err)
	return absPath
}

// GoldenPath returns the path to a golden file.
//
// Takes filename (string) which specifies the golden file name.
//
// Returns string which is the full path to the golden file.
func (h *BaseHarness) GoldenPath(filename string) string {
	return filepath.Join(h.GoldenDir, filename)
}

// AssertGolden compares actual bytes against a golden file.
//
// Takes filename (string) which identifies the golden file to compare against.
// Takes actual ([]byte) which contains the bytes to compare.
// Takes msgAndArgs (...any) which provides optional message and
// format arguments on failure.
func (h *BaseHarness) AssertGolden(filename string, actual []byte, msgAndArgs ...any) {
	h.T.Helper()
	AssertGoldenFile(h.T, h.GoldenPath(filename), actual, msgAndArgs...)
}

// AssertGoldenHTML compares actual HTML against a golden file.
//
// Takes filename (string) which is the name of the golden file to compare
// against.
// Takes actual ([]byte) which is the HTML content to verify.
// Takes msgAndArgs (...any) which provides optional message and format
// arguments for failure output.
func (h *BaseHarness) AssertGoldenHTML(filename string, actual []byte, msgAndArgs ...any) {
	h.T.Helper()
	AssertGoldenHTML(h.T, h.GoldenPath(filename), actual, msgAndArgs...)
}

// ReadFile reads a file from the source directory.
//
// Takes filePath (string) which specifies the path relative to the source
// directory.
//
// Returns []byte which contains the file contents.
// Returns error when the file cannot be read.
func (h *BaseHarness) ReadFile(relPath string) []byte {
	h.T.Helper()
	content, err := os.ReadFile(filepath.Join(h.SrcDir, relPath))
	require.NoError(h.T, err, "Failed to read file: %s", relPath)
	return content
}

// ReadTempFile reads a file from the temp directory.
//
// Takes relPath (string) which specifies the path relative to the temp
// directory.
//
// Returns []byte which contains the file contents.
func (h *BaseHarness) ReadTempFile(relPath string) []byte {
	h.T.Helper()
	require.NotEmpty(h.T, h.TempDir, "Temp directory not set up")
	content, err := os.ReadFile(filepath.Join(h.TempDir, relPath))
	require.NoError(h.T, err, "Failed to read temp file: %s", relPath)
	return content
}

// FileExists checks if a file exists in the source directory.
//
// Takes relPath (string) which specifies the path relative to the source
// directory.
//
// Returns bool which is true if the file exists, false otherwise.
func (h *BaseHarness) FileExists(relPath string) bool {
	_, err := os.Stat(filepath.Join(h.SrcDir, relPath))
	return err == nil
}

// TempFileExists checks if a file exists in the temp directory.
//
// Takes relPath (string) which specifies the path relative to the temp
// directory.
//
// Returns bool which is true if the file exists and is accessible.
func (h *BaseHarness) TempFileExists(relPath string) bool {
	if h.TempDir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(h.TempDir, relPath))
	return err == nil
}

// FSReader provides file system reading for integration tests.
type FSReader struct{}

// ReadFile reads a file from the file system.
//
// Takes filePath (string) which specifies the path to the file to read.
//
// Returns []byte which contains the file contents.
// Returns error when the file cannot be read.
func (*FSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// FSWriter is a no-op file system writer for integration tests.
// It stops any writes to disk while tests run.
type FSWriter struct{}

// WriteFile does nothing and always returns nil.
//
// Returns error which is always nil.
func (*FSWriter) WriteFile(_ context.Context, _ string, _ []byte) error {
	return nil
}
