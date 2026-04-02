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

package formatter_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/formatter/formatter_domain"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type TestSpec struct {
	Description   string `json:"description"`
	ErrorContains string `json:"errorContains,omitempty"`
	FileFormat    string `json:"fileFormat,omitempty"`
	ShouldError   bool   `json:"shouldError,omitempty"`
	RawHTMLMode   bool   `json:"rawHTMLMode,omitempty"`
}

func runTestCase(t *testing.T, tc testCase) {
	spec := loadTestSpec(t, tc)

	srcDir := filepath.Join(tc.Path, "src")
	goldenDir := filepath.Join(tc.Path, "golden")

	pkFiles := discoverPkFiles(t, srcDir)
	require.NotEmpty(t, pkFiles, "No .pk or .html files found in %s", srcDir)

	formatter := formatter_domain.NewFormatterService()

	ctx := context.Background()

	for _, srcPath := range pkFiles {
		relPath, err := filepath.Rel(srcDir, srcPath)
		require.NoError(t, err)

		sourceBytes, err := os.ReadFile(srcPath)
		require.NoError(t, err, "Failed to read source file: %s", srcPath)

		formatted, formatErr := formatSource(ctx, formatter, sourceBytes, spec)
		err = formatErr

		if spec.ShouldError {
			require.Error(t, err, "Expected formatting to fail for %s, but it succeeded", relPath)
			if spec.ErrorContains != "" {
				assert.Contains(t, err.Error(), spec.ErrorContains, "Error message didn't contain expected text")
			}
			continue
		}

		require.NoError(t, err, "Formatting failed unexpectedly for %s", relPath)
		require.NotNil(t, formatted, "Formatted output should not be nil")

		goldenPath := filepath.Join(goldenDir, relPath)
		assertGoldenFile(t, goldenPath, formatted, "Formatted output for %s", relPath)
	}
}

func formatSource(ctx context.Context, formatter formatter_domain.FormatterService, source []byte, spec TestSpec) ([]byte, error) {
	if spec.FileFormat != "" || spec.RawHTMLMode {
		opts := formatter_domain.DefaultFormatOptions()
		if spec.FileFormat == "html" {
			opts.FileFormat = formatter_domain.FormatHTML
		}
		opts.RawHTMLMode = spec.RawHTMLMode
		return formatter.FormatWithOptions(ctx, source, opts)
	}
	return formatter.Format(ctx, source)
}

func discoverPkFiles(t *testing.T, directory string) []string {
	t.Helper()
	var files []string

	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		require.NoError(t, err)

		if !d.IsDir() {
			lowerName := strings.ToLower(d.Name())
			if strings.HasSuffix(lowerName, ".pk") || strings.HasSuffix(lowerName, ".html") {
				files = append(files, path)
			}
		}

		return nil
	})

	require.NoError(t, err, "Failed to walk directory %s", directory)
	return files
}

func loadTestSpec(t *testing.T, tc testCase) TestSpec {
	t.Helper()
	var spec TestSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	if os.IsNotExist(err) {
		return TestSpec{
			Description: "Default formatter test case",
			ShouldError: false,
		}
	}
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	return spec
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
		t.Logf("--- EXPECTED (%s) ---\n%s\n--- ACTUAL (%s) ---\n%s\n",
			filepath.Base(goldenPath), string(expectedBytes),
			filepath.Base(goldenPath), string(actualBytes))
		assert.Fail(t, "Golden file mismatch: "+goldenPath+". Run with -update if this change is intentional.", msgAndArgs...)
	}
}
