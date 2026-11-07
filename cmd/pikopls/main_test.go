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

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/safedisk"
)

func TestResolveLogFile(t *testing.T) {
	t.Parallel()

	path, enabled := resolveLogFile(false, "")
	assert.False(t, enabled, "file logging is off by default")
	assert.Empty(t, path)

	path, enabled = resolveLogFile(false, " /var/log/../log/pikopls.log ")
	assert.True(t, enabled, "an explicit path enables file logging")
	assert.Equal(t, "/var/log/pikopls.log", path)

	path, enabled = resolveLogFile(true, "")
	assert.True(t, enabled)
	assert.Equal(t, os.TempDir(), filepath.Dir(path), "the default path lives in the OS temporary directory")
	assert.True(t, strings.HasPrefix(filepath.Base(path), "pikopls-"))
}

func TestProtocolLogPathFor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/var/log/pikopls.log.protocol", protocolLogPathFor("/var/log/pikopls.log"))

	matched, err := filepath.Match(staleLogPattern, filepath.Base(protocolLogPathFor("/tmp/pikopls-42.log")))
	require.NoError(t, err)
	assert.True(t, matched, "the protocol log must be covered by the stale log cleanup pattern")
}

func TestRemoveStaleLogsIn(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	now := time.Now()
	old := now.Add(-2 * staleLogMaxAge)

	writeLog := func(name string, modified time.Time) {
		t.Helper()
		path := filepath.Join(directory, name)
		require.NoError(t, os.WriteFile(path, []byte("log"), 0o600))
		require.NoError(t, os.Chtimes(path, modified, modified))
	}
	writeLog("pikopls-1.log", old)
	writeLog("pikopls-1.log.protocol", old)
	writeLog("pikopls-2.log", now)
	writeLog("unrelated.log", old)
	require.NoError(t, os.Mkdir(filepath.Join(directory, "pikopls-dir.log"), 0o700))

	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	removed, err := removeStaleLogsIn(sandbox, now.Add(-staleLogMaxAge))
	require.NoError(t, err)
	assert.Equal(t, 2, removed)

	for name, wantPresent := range map[string]bool{
		"pikopls-1.log":          false,
		"pikopls-1.log.protocol": false,
		"pikopls-2.log":          true,
		"unrelated.log":          true,
		"pikopls-dir.log":        true,
	} {
		_, statErr := os.Stat(filepath.Join(directory, name))
		assert.Equal(t, wantPresent, statErr == nil, "%s present", name)
	}
}

func TestRemoveStaleLogsIn_StopsAtRemovalCap(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	old := time.Now().Add(-2 * staleLogMaxAge)
	for index := range maxStaleLogRemovals + 3 {
		path := filepath.Join(directory, fmt.Sprintf("pikopls-%d.log", index))
		require.NoError(t, os.WriteFile(path, nil, 0o600))
		require.NoError(t, os.Chtimes(path, old, old))
	}

	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	removed, err := removeStaleLogsIn(sandbox, time.Now())
	require.NoError(t, err)
	assert.Equal(t, maxStaleLogRemovals, removed)
}

func TestRemoveStaleLogsIn_ReportsFailures(t *testing.T) {
	t.Parallel()

	listingFailure := safedisk.NewMockSandbox("/logs", safedisk.ModeReadWrite)
	listingFailure.ReadDirErr = errors.New("listing denied")
	_, err := removeStaleLogsIn(listingFailure, time.Now())
	require.ErrorIs(t, err, listingFailure.ReadDirErr)

	directory := t.TempDir()
	old := time.Now().Add(-2 * staleLogMaxAge)
	path := filepath.Join(directory, "pikopls-1.log")
	require.NoError(t, os.WriteFile(path, []byte("log"), 0o600))
	require.NoError(t, os.Chtimes(path, old, old))
	require.NoError(t, os.Chmod(directory, 0o500))
	t.Cleanup(func() { _ = os.Chmod(directory, 0o700) })
	if os.Geteuid() == 0 {
		t.Skip("directory permissions do not bind the root user")
	}

	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	removed, err := removeStaleLogsIn(sandbox, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "removing pikopls-1.log")
	assert.Zero(t, removed)
}

func TestValidateNumericFlags(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		port           int
		pprofPort      int
		maxConnections int
		wantErr        bool
	}{
		{name: "defaults are valid", port: defaultTCPPort, pprofPort: defaultPprofPort, maxConnections: 64},
		{name: "a single connection is valid", port: 1, pprofPort: 65535, maxConnections: 1},
		{name: "port below range", port: 0, pprofPort: defaultPprofPort, maxConnections: 1, wantErr: true},
		{name: "port above range", port: 65536, pprofPort: defaultPprofPort, maxConnections: 1, wantErr: true},
		{name: "pprof port out of range", port: defaultTCPPort, pprofPort: -1, maxConnections: 1, wantErr: true},
		{name: "no connections allowed", port: defaultTCPPort, pprofPort: defaultPprofPort, maxConnections: 0, wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateNumericFlags(testCase.port, testCase.pprofPort, testCase.maxConnections)
			if testCase.wantErr {
				require.ErrorIs(t, err, errInvalidFlag)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRemoveStaleLSPLogs(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	old := time.Now().Add(-2 * staleLogMaxAge)
	stale := filepath.Join(directory, "pikopls-7.log")
	require.NoError(t, os.WriteFile(stale, []byte("log"), 0o600))
	require.NoError(t, os.Chtimes(stale, old, old))

	removeStaleLSPLogs(directory, time.Now().Add(-staleLogMaxAge))

	_, err := os.Stat(stale)
	assert.ErrorIs(t, err, fs.ErrNotExist)

	blocker := filepath.Join(directory, "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	removeStaleLSPLogs(filepath.Join(blocker, "child"), time.Now())
}
