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

package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerUsesPhaseTempDir(t *testing.T) {
	if os.Getenv("PIKO_FUZZ_TEST_TEMP_CHILD") == "1" {
		require.Equal(t, os.Getenv("GOTMPDIR"), os.TempDir(), "Go and process temporary roots differ")
		require.NoError(t, os.WriteFile(filepath.Join(os.TempDir(), "overlay"), []byte("test"), 0o600))
		return
	}
	executable, err := os.Executable()
	require.NoError(t, err)
	temporary := t.TempDir()
	output, err := Run(WithTempDir(t.Context(), temporary), t.TempDir(), []string{"PIKO_FUZZ_TEST_TEMP_CHILD=1"}, executable, "-test.run=^TestWorkerUsesPhaseTempDir$")
	require.NoError(t, err, output)
	_, err = os.Stat(filepath.Join(temporary, "overlay"))
	require.NoError(t, err)
}

func TestRunReportsCancellationCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	cause := errors.New("stop requested")
	cancel(cause)
	_, err := Run(ctx, t.TempDir(), nil, "true")
	require.ErrorIs(t, err, cause)
}

func TestRunObservedCopiesOutput(t *testing.T) {
	var observed bytes.Buffer
	output, err := RunObserved(t.Context(), t.TempDir(), nil, &observed, "sh", "-c", "printf observed")
	require.NoError(t, err)
	assert.Equal(t, "observed", output)
	assert.Equal(t, "observed", observed.String())
}

func TestOutputLogRetainsBoundedHeadAndTail(t *testing.T) {
	var log outputLog
	_, err := log.Write([]byte("start-"))
	require.NoError(t, err)
	_, err = log.Write(bytes.Repeat([]byte("x"), logLimit))
	require.NoError(t, err)
	_, err = log.Write([]byte("-end"))
	require.NoError(t, err)
	rendered := log.String()
	assert.True(t, strings.HasPrefix(rendered, "start-"))
	assert.True(t, strings.HasSuffix(rendered, "-end"))
	assert.Contains(t, rendered, "log bytes omitted")
	assert.Less(t, len(rendered), logLimit+100)
}

func TestOutputLogTrimsCutRunes(t *testing.T) {
	var log outputLog
	_, err := log.Write(bytes.Repeat([]byte("a"), logLimit/2-1))
	require.NoError(t, err)
	_, err = log.Write(bytes.Repeat([]byte("é"), logLimit))
	require.NoError(t, err)
	_, err = log.Write([]byte("z"))
	require.NoError(t, err)

	rendered := log.String()

	assert.True(t, utf8.ValidString(rendered))
	assert.True(t, strings.HasSuffix(rendered, "éz"))
	assert.Contains(t, rendered, fmt.Sprintf("\n... %d log bytes omitted ...\n", logLimit*3/2+2))
}

func TestBuildWorkerReusesCachedBinary(t *testing.T) {
	workers := t.TempDir()
	cached := filepath.Join(workers, "helper-tag")
	require.NoError(t, os.WriteFile(cached, []byte("binary"), 0o700))
	binary, err := BuildWorker(t.Context(), t.TempDir(), workers, "helper", "tag")
	require.NoError(t, err)
	assert.Equal(t, cached, binary)
}

func TestRunObservedReturnsExitFailureWithOutput(t *testing.T) {
	output, err := Run(t.Context(), t.TempDir(), nil, "sh", "-c", "echo diagnostic; exit 3")

	exitErr, ok := errors.AsType[*exec.ExitError](err)
	require.True(t, ok, "error %v is not an exit error", err)
	assert.Equal(t, 3, exitErr.ExitCode())
	assert.Equal(t, "diagnostic\n", output)
}

func TestRunObservedReportsLeakedOutputPipe(t *testing.T) {
	output, err := Run(t.Context(), t.TempDir(), nil, "sh", "-c", "sleep 60 & echo started")

	require.ErrorIs(t, err, exec.ErrWaitDelay)
	assert.ErrorContains(t, err, "sh left output pipes open")
	assert.Equal(t, "started\n", output)
}

func TestRunObservedRefusesCancelledContextBeforeStarting(t *testing.T) {
	directory := t.TempDir()
	marker := filepath.Join(directory, "started")
	ctx, cancel := context.WithCancelCause(t.Context())
	cause := errors.New("already stopped")
	cancel(cause)

	output, err := RunObserved(ctx, directory, nil, nil, "touch", marker)

	require.ErrorIs(t, err, cause)
	assert.Empty(t, output)
	assert.NoFileExists(t, marker)
}

func TestRunObservedReportsMissingExecutable(t *testing.T) {
	_, err := Run(t.Context(), t.TempDir(), nil, filepath.Join(t.TempDir(), "absent"))

	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func writeWorkerRepository(t *testing.T, source string) string {
	t.Helper()
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
	repository := t.TempDir()
	module := filepath.Join(repository, "tests", "fuzz")
	require.NoError(t, os.MkdirAll(filepath.Join(module, "cmd", "helper"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.test/helper\n\ngo 1.25\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(module, "cmd", "helper", "main.go"), []byte(source), 0o600))
	return repository
}

func TestBuildWorkerCompilesTaggedHelper(t *testing.T) {
	repository := writeWorkerRepository(t, "//go:build special\n\npackage main\n\nfunc main() {}\n")
	workers := t.TempDir()

	binary, err := BuildWorker(t.Context(), repository, workers, "helper", "special")

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(workers, "helper-special"), binary)
	assert.FileExists(t, binary)
	assert.NoFileExists(t, binary+".tmp")
}

func TestBuildCoveredWorkerCompilesInstrumentedHelper(t *testing.T) {
	repository := writeWorkerRepository(t, "package main\n\nfunc main() {}\n")
	workers := t.TempDir()

	binary, err := BuildCoveredWorker(t.Context(), repository, workers, "helper", "example.test/helper/...")

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(workers, "helper-covered"), binary)
	assert.FileExists(t, binary)
}

func TestBuildWorkerReportsCompileFailureAndLeavesNoStagedBinary(t *testing.T) {
	repository := writeWorkerRepository(t, "package main\n\nfunc main() { undefinedCall() }\n")
	workers := t.TempDir()

	binary, err := BuildWorker(t.Context(), repository, workers, "helper", "")

	require.Error(t, err)
	assert.Empty(t, binary)
	assert.ErrorContains(t, err, "worker-build/helper")
	assert.ErrorContains(t, err, "undefinedCall")
	assert.NoFileExists(t, filepath.Join(workers, "helper.tmp"))
	assert.NoFileExists(t, filepath.Join(workers, "helper"))
}

func TestBuildWorkerReportsUninspectableCache(t *testing.T) {
	workers := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(workers, []byte("not a directory"), 0o600))

	_, err := BuildWorker(t.Context(), t.TempDir(), workers, "helper", "")

	assert.ErrorContains(t, err, "inspect cached binary")
}

func TestRemoveStagedIgnoresAbsentFileAndReportsFailure(t *testing.T) {
	require.NoError(t, removeStaged(filepath.Join(t.TempDir(), "absent")))

	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "child"), []byte("x"), 0o600))
	assert.ErrorContains(t, removeStaged(directory), "remove staged worker")
}
