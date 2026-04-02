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

package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/goastutil"
)

func TestAnnotatorService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping annotator service integration tests in short mode")
	}

	goastutil.ResetDynamicCaches()
	ast_domain.ClearExpressionCache()
	ast_domain.ClearSelectorCache()
	ast_domain.ResetAllPools()
	compiler_domain.ClearIdentifierRegistry()
	compiler_domain.ClearBindingRegistry()
	compiler_domain.ClearLocRefRegistry()

	testdataRoot := "./testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("Critical test setup error: Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCaseName := entry.Name()

		tc := testCase{
			Name:      testCaseName,
			Path:      filepath.Join(testdataRoot, testCaseName),
			EntryFile: "main.pk",
		}

		t.Run(tc.Name, func(t *testing.T) {
			srcPath := filepath.Join(tc.Path, "src")
			specPath := filepath.Join(tc.Path, "testspec.json")

			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				t.Skipf("Skipping test case '%s': missing 'src' directory", tc.Name)
				return
			}
			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				t.Skipf("Skipping test case '%s': missing 'testspec.json' file", tc.Name)
				return
			}

			runTestCase(t, tc)
		})
	}
}
