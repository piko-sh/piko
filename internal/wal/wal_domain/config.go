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
	"path/filepath"
	"time"
)

// SyncMode controls when data is fsynced to disk.
type SyncMode uint8

const (
	// SyncModeNone performs no explicit sync; the OS decides when to flush.
	SyncModeNone SyncMode = iota

	// SyncModeEveryWrite performs fsync after every write operation. This is the
	// safest mode but also the slowest, so use it when data loss is unacceptable.
	SyncModeEveryWrite

	// SyncModeBatched performs fsync periodically based on time interval
	// and/or write count thresholds. This provides a balance between
	// durability and performance.
	//
	// DATA LOSS WINDOW: With SyncModeBatched, there is a small window where
	// data could be lost on crash:
	//   - Up to BatchSyncCount entries pending in memory
	//   - Up to BatchSyncInterval duration (~50us default) between flushes
	//
	// This is acceptable for cache persistence where:
	//   - Data can be reconstructed from source
	//   - Occasional re-computation on recovery is tolerable
	//   - High throughput is prioritised over zero-data-loss guarantees
	//
	// For applications requiring zero data loss, use SyncModeEveryWrite
	// (at the cost of ~10x lower throughput due to per-write fsync).
	SyncModeBatched
)

const (
	// DefaultBaseDir is the default base directory for WAL storage.
	// WAL files are stored in subdirectories named after the cache/WAL name.
	DefaultBaseDir = ".piko/wal"

	// DefaultSyncMode is the default sync mode for new WAL instances.
	DefaultSyncMode = SyncModeBatched

	// DefaultBatchSyncInterval is the default time between syncs in batched mode.
	DefaultBatchSyncInterval = 100 * time.Millisecond

	// DefaultBatchSyncCount is the default number of writes that trigger a sync.
	DefaultBatchSyncCount = 100

	// DefaultSnapshotThreshold is the default number of WAL entries before
	// an automatic snapshot is triggered.
	DefaultSnapshotThreshold = 10000

	// DefaultCompressionLevel is the default zstd compression level (1-19).
	// Level 3 provides a good balance between speed and compression ratio.
	DefaultCompressionLevel = 3

	// MaxCompressionLevel is the maximum allowed zstd compression level.
	MaxCompressionLevel = 19

	// DefaultMaxWALSize is the default maximum WAL file size before compaction.
	DefaultMaxWALSize = 64 * 1024 * 1024
)

// String returns the string representation of the sync mode.
//
// Returns string which is the mode name such as "NONE", "EVERY_WRITE", or
// "BATCHED". Returns "UNKNOWN" for undefined mode values.
func (m SyncMode) String() string {
	switch m {
	case SyncModeNone:
		return "NONE"
	case SyncModeEveryWrite:
		return "EVERY_WRITE"
	case SyncModeBatched:
		return "BATCHED"
	default:
		return "UNKNOWN"
	}
}

// Config configures the WAL behaviour.
type Config struct {
	// Dir is the required directory for WAL and snapshot files, created
	// automatically if it does not exist.
	Dir string

	// WALFileName is the name of the write-ahead log file within Dir.
	// Default: "data.wal".
	WALFileName string

	// SnapshotFileName is the name of the snapshot file within Dir.
	// Default: "snapshot.piko".
	SnapshotFileName string

	// BatchSyncInterval is the maximum time between syncs in
	// batched mode, where a sync occurs when this interval
	// elapses or when BatchSyncCount writes have been reached,
	// whichever comes first (default: 100ms).
	BatchSyncInterval time.Duration

	// BatchSyncCount is the number of writes that trigger a sync
	// in batched mode, where a sync also occurs when
	// BatchSyncInterval elapses, whichever comes first
	// (default: 100).
	BatchSyncCount int

	// SnapshotThreshold is the number of WAL entries that trigger an automatic
	// snapshot followed by WAL truncation, where 0 disables automatic
	// snapshots.
	// Default: 10000.
	SnapshotThreshold int

	// MaxWALSize is the advisory maximum WAL file size in bytes
	// before compaction is recommended, where the WAL may
	// temporarily exceed this limit (default: 64MB).
	MaxWALSize int64

	// CompressionLevel sets the zstd compression level (1-19),
	// where higher levels provide better compression at the cost
	// of speed (default: 3).
	CompressionLevel int

	// SyncMode controls the durability versus performance trade-off for writes.
	// Default is SyncModeBatched.
	SyncMode SyncMode

	// EnableCompression enables zstd compression for snapshot
	// files, where WAL entries remain uncompressed to maintain
	// append performance (default: true).
	EnableCompression bool

	// DisableAlignedWrites disables 4KB-aligned writes.
	//
	// By default (zero value), writes are padded to 4KB sector
	// boundaries to reduce SSD write amplification. Set to true
	// for HDD or RAM-backed filesystems where alignment overhead
	// is unnecessary.
	DisableAlignedWrites bool
}

// Validate reports whether the configuration is valid.
//
// Returns error when any required fields are missing or invalid.
func (c Config) Validate() error {
	if c.Dir == "" {
		return ErrInvalidConfig
	}
	if c.CompressionLevel < 1 || c.CompressionLevel > MaxCompressionLevel {
		return ErrInvalidConfig
	}
	if c.BatchSyncInterval < 0 {
		return ErrInvalidConfig
	}
	if c.BatchSyncCount < 0 {
		return ErrInvalidConfig
	}
	return nil
}

// WithDefaults returns a copy of the config with default values applied
// for any zero-value fields.
//
// Returns Config which is the configuration with defaults set for any
// unspecified fields.
func (c Config) WithDefaults() Config {
	if c.SyncMode == 0 {
		c.SyncMode = DefaultSyncMode
	}
	if c.BatchSyncInterval == 0 {
		c.BatchSyncInterval = DefaultBatchSyncInterval
	}
	if c.BatchSyncCount == 0 {
		c.BatchSyncCount = DefaultBatchSyncCount
	}
	if c.SnapshotThreshold == 0 {
		c.SnapshotThreshold = DefaultSnapshotThreshold
	}
	if c.MaxWALSize == 0 {
		c.MaxWALSize = DefaultMaxWALSize
	}
	if c.CompressionLevel == 0 {
		c.CompressionLevel = DefaultCompressionLevel
	}
	if c.WALFileName == "" {
		c.WALFileName = "data.wal"
	}
	if c.SnapshotFileName == "" {
		c.SnapshotFileName = "snapshot.piko"
	}
	return c
}

// DefaultConfig returns a Config with sensible default values.
//
// Takes directory (string) which specifies where WAL and snapshot
// files are stored.
//
// Returns Config which contains the initialised configuration.
func DefaultConfig(directory string) Config {
	return Config{
		Dir:               directory,
		SyncMode:          DefaultSyncMode,
		BatchSyncInterval: DefaultBatchSyncInterval,
		BatchSyncCount:    DefaultBatchSyncCount,
		SnapshotThreshold: DefaultSnapshotThreshold,
		MaxWALSize:        DefaultMaxWALSize,
		EnableCompression: true,
		CompressionLevel:  DefaultCompressionLevel,
		WALFileName:       "data.wal",
		SnapshotFileName:  "snapshot.piko",
	}
}

// DefaultConfigNamed returns a Config using the default .piko/wal/{name}
// directory. This is a convenience function for creating WAL configs with
// sensible defaults.
//
// Takes name (string) which specifies the subdirectory name within the default
// WAL base directory.
//
// Returns Config which contains the WAL settings with the directory set to
// .piko/wal/{name}.
func DefaultConfigNamed(name string) Config {
	return DefaultConfig(filepath.Join(DefaultBaseDir, name))
}
