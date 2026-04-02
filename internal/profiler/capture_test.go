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

package profiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartCapture_CreatesAllProfileFiles(t *testing.T) {

	outputDirectory := t.TempDir()

	cleanup, err := StartCapture(Config{
		OutputDir: outputDirectory,
	})
	require.NoError(t, err)
	require.NotNil(t, cleanup)

	cpuFile := filepath.Join(outputDirectory, "cpu.pprof")
	traceFile := filepath.Join(outputDirectory, "trace.out")
	assertFileExists(t, cpuFile)
	assertFileExists(t, traceFile)

	cleanup()

	expectedFiles := []string{
		"heap.pprof",
		"block.pprof",
		"mutex.pprof",
		"goroutine.pprof",
		"allocs.pprof",
	}
	for _, filename := range expectedFiles {
		assertFileExists(t, filepath.Join(outputDirectory, filename))
	}

	cpuInfo, err := os.Stat(cpuFile)
	require.NoError(t, err)
	assert.Greater(t, cpuInfo.Size(), int64(0))

	heapFile := filepath.Join(outputDirectory, "heap.pprof")
	heapInfo, err := os.Stat(heapFile)
	require.NoError(t, err)
	assert.Greater(t, heapInfo.Size(), int64(0))
}

func TestStartCapture_FailsWithEmptyOutputDir(t *testing.T) {
	t.Parallel()

	cleanup, err := StartCapture(Config{
		OutputDir: "",
	})

	assert.Error(t, err)
	assert.Nil(t, cleanup)
}

func TestStartCapture_FailsWithNonExistentDirectory(t *testing.T) {
	t.Parallel()

	cleanup, err := StartCapture(Config{
		OutputDir: "/nonexistent/path/that/does/not/exist",
	})

	assert.Error(t, err)
	assert.Nil(t, cleanup)
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.NoError(t, err, "expected file to exist: %s", path)
}
