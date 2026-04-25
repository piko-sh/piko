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

package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCrashOutput_NilContainer(t *testing.T) {
	t.Parallel()

	closeFn, err := InstallCrashOutput(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, closeFn)
}

func TestInstallCrashOutput_DisabledByDefault(t *testing.T) {
	t.Parallel()

	closeFn, err := InstallCrashOutput(context.Background(), &Container{})
	require.NoError(t, err)
	assert.Nil(t, closeFn, "no path → no close function")
}

func TestInstallCrashOutput_OpensConfiguredFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "crash.log")

	container := &Container{crashOutputPath: path}
	closeFn, err := InstallCrashOutput(context.Background(), container)
	require.NoError(t, err)
	require.NotNil(t, closeFn, "close function must be returned when path is set")

	t.Cleanup(closeFn)

	_, err = filepath.Abs(path)
	require.NoError(t, err)

}

func TestInstallCrashOutput_RejectsInvalidTracebackLevel(t *testing.T) {
	t.Parallel()

	container := &Container{crashTracebackLevel: "bogus"}
	closeFn, err := InstallCrashOutput(context.Background(), container)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCrashTracebackLevel)
	assert.Nil(t, closeFn)
}

func TestInstallCrashOutput_AcceptsValidTracebackLevels(t *testing.T) {
	t.Parallel()

	for _, level := range []string{"none", "single", "all", "system"} {
		container := &Container{crashTracebackLevel: level}
		closeFn, err := InstallCrashOutput(context.Background(), container)
		require.NoErrorf(t, err, "level %q should be accepted", level)
		assert.Nilf(t, closeFn, "no path supplied → no close function for %q", level)
	}
}

func TestInstallCrashOutput_FailsSilentlyOnUnwritablePath(t *testing.T) {
	t.Parallel()

	container := &Container{crashOutputPath: "/proc/this-cannot-be-created-anywhere"}
	closeFn, err := InstallCrashOutput(context.Background(), container)
	require.NoError(t, err, "open failure must not propagate")
	assert.Nil(t, closeFn, "no FD opened → no close function")
}
