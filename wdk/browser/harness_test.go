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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHarnessBeforeSetup(t *testing.T) {
	harness := NewHarness(WithInteractive(true))
	t.Cleanup(harness.Cleanup)
	assert.Same(t, harness, getGlobalHarness())
	assert.True(t, harness.IsInteractive())
	assert.Empty(t, harness.ServerURL())

	_, err := harness.NewSession()
	require.ErrorContains(t, err, "harness not set up")
	_, err = harness.newPage(t)
	require.ErrorContains(t, err, "harness not set up")

	harness.Cleanup()
	assert.Nil(t, getGlobalHarness())
}

func TestHarnessSetupReportsMissingServerCommand(t *testing.T) {
	harness := NewHarness(
		WithProjectDir(t.TempDir()),
		WithSkipBuild(true),
		WithPort(1),
		WithServerCommand("wdk-browser-no-such-server-command"),
	)
	t.Cleanup(harness.Cleanup)
	err := harness.Setup()
	require.ErrorContains(t, err, `command "wdk-browser-no-such-server-command" not found`)
	assert.Equal(t, "http://localhost:1", harness.ServerURL())
	require.ErrorIs(t, harness.Setup(), err, "a failed setup is not retried")

	temporary := harness.tempDir
	require.DirExists(t, temporary)
	harness.Cleanup()
	assert.NoDirExists(t, temporary)
}

func TestHarnessSetupReportsBuildFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script in place of the go command")
	}
	tools := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tools, "go"), []byte("#!/bin/sh\nexit 3\n"), 0o700))
	t.Setenv("PATH", tools)

	harness := NewHarness(WithProjectDir(t.TempDir()))
	t.Cleanup(harness.Cleanup)
	require.ErrorContains(t, harness.Setup(), "piko generate failed")
}
