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
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const (
	// goldenDirMode is the file mode for golden test directories.
	goldenDirMode = 0755

	// goldenFileMode is the file permission mode for golden test files.
	goldenFileMode = 0644

	// goldenDir is the directory under the test working directory where golden
	// files are stored.
	goldenDir = "testdata/golden"
)

// compareGolden compares actual HTML against a golden file. If the
// PIKO_UPDATE_GOLDEN environment variable is set to "1", the golden file is
// created or overwritten instead.
//
// Golden files are stored at testdata/golden/<name>.html relative to the test
// working directory.
//
// Takes t (testing.TB) which is the test instance for logging.
// Takes name (string) which is the golden file name without extension.
// Takes actual (string) which is the HTML content to compare.
//
// Returns error when the golden file cannot be read or content does not match.
func compareGolden(t testing.TB, name, actual string) error {
	t.Helper()

	goldenPath := filepath.Join(goldenDir, name+".html")

	if os.Getenv("PIKO_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), goldenDirMode); err != nil {
			return fmt.Errorf("creating golden directory: %w", err)
		}
		if err := os.WriteFile(goldenPath, []byte(actual), goldenFileMode); err != nil {
			return fmt.Errorf("writing golden file: %w", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
	}

	expected, err := os.ReadFile(goldenPath) //nolint:gosec // test fixture path
	if err != nil {
		return fmt.Errorf("reading golden file %s (run with PIKO_UPDATE_GOLDEN=1 to create): %w", goldenPath, err)
	}

	if string(expected) != actual {
		t.Logf("--- EXPECTED (%s) ---\n%s", name, string(expected))
		t.Logf("--- ACTUAL ---\n%s", actual)
		return fmt.Errorf("golden file mismatch: %s (run with PIKO_UPDATE_GOLDEN=1 to update)", goldenPath)
	}

	return nil
}
