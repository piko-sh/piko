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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/json"
	"github.com/stretchr/testify/require"
)

const (
	// fileModeDir is the Unix permission mode for directories (rwxr-xr-x).
	fileModeDir = 0755

	// fileModeFile is the file permission mode for regular files (owner read/write,
	// group and others read-only).
	fileModeFile = 0644
)

var (
	// UpdateGolden controls whether golden files are regenerated during tests.
	// It is set via the -update flag and initialised in flags.go.
	UpdateGolden *bool

	// KeepStaged is set via the -keep-staged flag to preserve staged directories.
	// This is initialised in flags.go.
	KeepStaged *bool
)

// AssertGoldenFile compares actual bytes against a golden file.
// If UpdateGolden is true, it updates the golden file instead.
//
// Takes t (*testing.T) which is the test context for assertions.
// Takes goldenPath (string) which is the path to the golden file.
// Takes actual ([]byte) which is the content to compare against the golden.
// Takes msgAndArgs (...any) which provides optional failure message arguments.
func AssertGoldenFile(t *testing.T, goldenPath string, actual []byte, msgAndArgs ...any) {
	t.Helper()

	if UpdateGolden != nil && *UpdateGolden {
		t.Logf("Updating golden file: %s", goldenPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), fileModeDir))
		require.NoError(t, os.WriteFile(goldenPath, actual, fileModeFile))
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	if !bytes.Equal(expected, actual) {
		t.Logf("--- EXPECTED (%s) ---\n%s", filepath.Base(goldenPath), string(expected))
		t.Logf("--- ACTUAL ---\n%s", string(actual))
		assert.Fail(t, fmt.Sprintf("Golden file mismatch: %s. Run with -update if this change is intentional.", goldenPath), msgAndArgs...)
	}
}

// AssertGoldenJSON compares actual JSON bytes against a golden file.
// It uses JSON-aware comparison to ignore formatting differences.
//
// Takes t (*testing.T) which is the test context for assertions and logging.
// Takes goldenPath (string) which is the path to the golden file.
// Takes actual ([]byte) which is the JSON bytes to compare against the golden
// file.
// Takes msgAndArgs (...any) which provides optional message and arguments for
// assertion failures.
func AssertGoldenJSON(t *testing.T, goldenPath string, actual []byte, msgAndArgs ...any) {
	t.Helper()

	if UpdateGolden != nil && *UpdateGolden {
		t.Logf("Updating golden file: %s", goldenPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), fileModeDir))
		require.NoError(t, os.WriteFile(goldenPath, actual, fileModeFile))
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	assert.JSONEq(t, string(expected), string(actual), msgAndArgs...)
}

// AssertGoldenHTML compares actual HTML bytes against a golden file.
// This is an alias for AssertGoldenFile but provides semantic clarity.
//
// Takes t (*testing.T) which is the test context for reporting failures.
// Takes goldenPath (string) which is the path to the golden file.
// Takes actual ([]byte) which contains the HTML bytes to compare.
// Takes msgAndArgs (...any) which provides optional failure message details.
func AssertGoldenHTML(t *testing.T, goldenPath string, actual []byte, msgAndArgs ...any) {
	AssertGoldenFile(t, goldenPath, actual, msgAndArgs...)
}

// WriteGoldenFile writes content to a golden file, creating directories as
// needed.
//
// Takes t (*testing.T) which is the test context for reporting failures.
// Takes goldenPath (string) which is the file path to write the golden file.
// Takes content ([]byte) which is the data to write to the file.
func WriteGoldenFile(t *testing.T, goldenPath string, content []byte) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), fileModeDir))
	require.NoError(t, os.WriteFile(goldenPath, content, fileModeFile))
}

// ReadGoldenFile reads a golden file and returns its contents.
//
// Takes t (*testing.T) which is the test context for error reporting.
// Takes goldenPath (string) which is the path to the golden file.
//
// Returns []byte which contains the file contents.
func ReadGoldenFile(t *testing.T, goldenPath string) []byte {
	t.Helper()
	content, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "Failed to read golden file: %s", goldenPath)
	return content
}

// PrettyPrintJSON formats JSON bytes with indentation for better readability.
//
// Takes data ([]byte) which contains the JSON to format.
//
// Returns []byte which contains the formatted JSON with two-space indentation.
// Returns error when the input is not valid JSON.
func PrettyPrintJSON(data []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return json.StdConfig().MarshalIndent(v, "", "  ")
}
