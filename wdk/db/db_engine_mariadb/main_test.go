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

package db_engine_mariadb_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestMariaDBEngine_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping querier MariaDB integration tests in short mode")
	}

	testdataRoot := "./testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}

	var testDirectories []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCasePath := filepath.Join(testdataRoot, entry.Name())
		specPath := filepath.Join(testCasePath, "testspec.json")

		if _, err := os.Stat(specPath); os.IsNotExist(err) {
			continue
		}

		testDirectories = append(testDirectories, entry.Name())
	}
	sort.Strings(testDirectories)

	if len(testDirectories) == 0 {
		t.Fatal("no test directories found in testdata/")
	}

	for _, directory := range testDirectories {
		t.Run(directory, func(t *testing.T) {
			runTestCase(t, filepath.Join(testdataRoot, directory))
		})
	}
}
