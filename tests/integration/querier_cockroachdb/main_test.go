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

package querier_cockroachdb_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/testutil/leakcheck"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

func TestMain(m *testing.M) {
	flag.Parse()

	ctx := context.Background()

	var err error
	cockroachContainer, testConnectionString, err = startCockroachDBContainer(ctx)
	if err != nil {
		log.Fatalf("starting cockroachdb container: %v", err)
	}

	exitCode := m.Run()

	if cockroachContainer != nil {
		_ = cockroachContainer.Terminate(context.Background())
	}

	if exitCode == 0 {
		if err := leakcheck.FindLeaks(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(exitCode)
}

func TestQuerierCockroachDBIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDirectory := filepath.Join("testdata")
	entries, err := os.ReadDir(testdataDirectory)
	if err != nil {
		t.Fatalf("reading testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCaseDirectory := filepath.Join(testdataDirectory, entry.Name())
		specPath := filepath.Join(testCaseDirectory, "testspec.json")
		if _, statError := os.Stat(specPath); os.IsNotExist(statError) {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			runTestCase(t, testCaseDirectory)
		})
	}
}
