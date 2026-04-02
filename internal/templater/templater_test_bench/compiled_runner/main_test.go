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

//go:build bench

package compiled_bench_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompiledRunner_Benchmark(t *testing.T) {
	if !*runBenchmarks {
		t.Skip("Skipping compiled benchmarks. Use the -run-bench flag to enable.")
	}

	testdataRoot := "testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("Critical test setup error: Failed to read testdata directory at '%s': %v", testdataRoot, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		benchCaseName := entry.Name()
		bc := benchmarkCase{
			Name: benchCaseName,
			Path: filepath.Join(testdataRoot, benchCaseName),
		}

		t.Run(bc.Name, func(t *testing.T) {
			b := testing.B{N: 1}
			runBenchmarkCase(&b, bc)
		})
	}
}
