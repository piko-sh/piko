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

//go:build crosslang

package crosslang

import (
	"os"
	"path/filepath"
	"testing"

	"piko.sh/piko/tests/benchmarks/cross_language/harness"
)

func TestCrossLanguage(testingHandle *testing.T) {
	if os.Getenv("RUN_CROSS_LANG_BENCH") != "1" {
		testingHandle.Skip("set RUN_CROSS_LANG_BENCH=1 to run cross-language suite")
	}
	benchmarksDirectory := mustResolveBenchmarksDir(testingHandle)
	suite := harness.Load(testingHandle, benchmarksDirectory)
	suite.Run(testingHandle)
}

func mustResolveBenchmarksDir(testingHandle *testing.T) string {
	testingHandle.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		testingHandle.Fatalf("getwd: %v", err)
	}
	return filepath.Join(cwd, "benchmarks")
}
