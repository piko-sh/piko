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

package wal_domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncMode_String(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		mode     SyncMode
	}{
		{
			name:     "SyncModeNone",
			mode:     SyncModeNone,
			expected: "NONE",
		},
		{
			name:     "SyncModeEveryWrite",
			mode:     SyncModeEveryWrite,
			expected: "EVERY_WRITE",
		},
		{
			name:     "SyncModeBatched",
			mode:     SyncModeBatched,
			expected: "BATCHED",
		},
		{
			name:     "unknown mode",
			mode:     SyncMode(99),
			expected: "UNKNOWN",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.mode.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name:      "valid config with defaults",
			config:    DefaultConfig("/tmp/wal"),
			expectErr: false,
		},
		{
			name: "valid minimal config",
			config: Config{
				Dir:              "/tmp/wal",
				CompressionLevel: 3,
			},
			expectErr: false,
		},
		{
			name: "empty directory",
			config: Config{
				Dir:              "",
				CompressionLevel: 3,
			},
			expectErr: true,
		},
		{
			name: "compression level too low",
			config: Config{
				Dir:              "/tmp/wal",
				CompressionLevel: 0,
			},
			expectErr: true,
		},
		{
			name: "compression level too high",
			config: Config{
				Dir:              "/tmp/wal",
				CompressionLevel: 20,
			},
			expectErr: true,
		},
		{
			name: "negative batch sync interval",
			config: Config{
				Dir:               "/tmp/wal",
				CompressionLevel:  3,
				BatchSyncInterval: -1 * time.Millisecond,
			},
			expectErr: true,
		},
		{
			name: "negative batch sync count",
			config: Config{
				Dir:              "/tmp/wal",
				CompressionLevel: 3,
				BatchSyncCount:   -1,
			},
			expectErr: true,
		},
		{
			name: "zero batch sync interval is valid",
			config: Config{
				Dir:               "/tmp/wal",
				CompressionLevel:  3,
				BatchSyncInterval: 0,
			},
			expectErr: false,
		},
		{
			name: "zero batch sync count is valid",
			config: Config{
				Dir:              "/tmp/wal",
				CompressionLevel: 3,
				BatchSyncCount:   0,
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	t.Run("applies all defaults to empty config", func(t *testing.T) {
		config := Config{Dir: "/tmp/wal"}.WithDefaults()

		assert.Equal(t, "/tmp/wal", config.Dir)
		assert.Equal(t, DefaultSyncMode, config.SyncMode)
		assert.Equal(t, DefaultBatchSyncInterval, config.BatchSyncInterval)
		assert.Equal(t, DefaultBatchSyncCount, config.BatchSyncCount)
		assert.Equal(t, DefaultSnapshotThreshold, config.SnapshotThreshold)
		assert.Equal(t, int64(DefaultMaxWALSize), config.MaxWALSize)
		assert.Equal(t, DefaultCompressionLevel, config.CompressionLevel)
		assert.Equal(t, "data.wal", config.WALFileName)
		assert.Equal(t, "snapshot.piko", config.SnapshotFileName)
	})

	t.Run("preserves explicitly set values", func(t *testing.T) {
		config := Config{
			Dir:               "/custom/directory",
			SyncMode:          SyncModeEveryWrite,
			BatchSyncInterval: 500 * time.Millisecond,
			BatchSyncCount:    50,
			SnapshotThreshold: 5000,
			MaxWALSize:        128 * 1024 * 1024,
			CompressionLevel:  5,
			WALFileName:       "custom.wal",
			SnapshotFileName:  "custom.snap",
		}.WithDefaults()

		assert.Equal(t, "/custom/directory", config.Dir)
		assert.Equal(t, SyncModeEveryWrite, config.SyncMode)
		assert.Equal(t, 500*time.Millisecond, config.BatchSyncInterval)
		assert.Equal(t, 50, config.BatchSyncCount)
		assert.Equal(t, 5000, config.SnapshotThreshold)
		assert.Equal(t, int64(128*1024*1024), config.MaxWALSize)
		assert.Equal(t, 5, config.CompressionLevel)
		assert.Equal(t, "custom.wal", config.WALFileName)
		assert.Equal(t, "custom.snap", config.SnapshotFileName)
	})

	t.Run("SyncModeNone is preserved as zero value", func(t *testing.T) {

		config := Config{
			Dir:      "/tmp/wal",
			SyncMode: SyncModeNone,
		}.WithDefaults()

		assert.Equal(t, DefaultSyncMode, config.SyncMode)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Run("creates config with expected defaults", func(t *testing.T) {
		config := DefaultConfig("/var/data/wal")

		assert.Equal(t, "/var/data/wal", config.Dir)
		assert.Equal(t, SyncModeBatched, config.SyncMode)
		assert.Equal(t, 100*time.Millisecond, config.BatchSyncInterval)
		assert.Equal(t, 100, config.BatchSyncCount)
		assert.Equal(t, 10000, config.SnapshotThreshold)
		assert.Equal(t, int64(64*1024*1024), config.MaxWALSize)
		assert.True(t, config.EnableCompression)
		assert.Equal(t, 3, config.CompressionLevel)
		assert.Equal(t, "data.wal", config.WALFileName)
		assert.Equal(t, "snapshot.piko", config.SnapshotFileName)
	})

	t.Run("default config is valid", func(t *testing.T) {
		config := DefaultConfig("/tmp/wal")
		err := config.Validate()
		require.NoError(t, err)
	})
}

func TestDefaultConfigNamed(t *testing.T) {
	t.Run("creates config with .piko/wal/{name} directory", func(t *testing.T) {
		config := DefaultConfigNamed("users")

		assert.Equal(t, ".piko/wal/users", config.Dir)
		assert.Equal(t, SyncModeBatched, config.SyncMode)
		assert.Equal(t, "data.wal", config.WALFileName)
		assert.Equal(t, "snapshot.piko", config.SnapshotFileName)
	})

	t.Run("handles nested names", func(t *testing.T) {
		config := DefaultConfigNamed("myapp/sessions")

		assert.Equal(t, ".piko/wal/myapp/sessions", config.Dir)
	})

	t.Run("named config is valid", func(t *testing.T) {
		config := DefaultConfigNamed("cache")
		err := config.Validate()
		require.NoError(t, err)
	})
}
