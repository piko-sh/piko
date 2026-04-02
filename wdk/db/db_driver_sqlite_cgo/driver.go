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

package db_driver_sqlite_cgo

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // register "sqlite3" database/sql driver
)

const (
	// intFormat is the fmt verb for formatting integers in PRAGMA values.
	intFormat = "%d"

	driverName = "sqlite3"

	defaultBusyTimeoutMs = 10_000

	defaultCachePages = -20_000

	defaultMmapSize = 64 * 1024 * 1024

	defaultJournalSizeLimit = 32 * 1024 * 1024

	poolSize = 1

	connMaxIdleTime = 5 * time.Minute

	connMaxLifetime = 1 * time.Hour

	directoryPermission = 0o750
)

// Config holds configuration for opening a SQLite database.
type Config struct {
	// BusyTimeoutMs is the timeout in milliseconds for SQLite busy waits.
	// Zero uses the default (10000).
	BusyTimeoutMs int

	// CachePages sets the number of pages to keep in the SQLite cache.
	// Zero uses the default (-20000, approximately 20 MB).
	CachePages int

	// MmapSize sets the memory-mapped I/O size in bytes.
	// Zero uses the default (64 MB).
	MmapSize int

	// JournalSizeLimit sets the maximum size in bytes for the WAL journal.
	// Zero uses the default (32 MB).
	JournalSizeLimit int
}

// Open opens a SQLite database at the given path with production-ready
// PRAGMAs applied. The returned *sql.DB is configured with a single
// connection (SQLite's single-writer model) and WAL mode enabled.
//
// Takes path (string) which is the filesystem path to the SQLite database file.
// Takes config (Config) which provides optional tuning parameters.
//
// Returns *sql.DB which is the configured database connection.
// Returns error when the database cannot be opened or PRAGMAs fail to apply.
func Open(path string, config Config) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("db_driver_sqlite_cgo: path must not be empty")
	}

	if directory := filepath.Dir(path); directory != "." && directory != "" {
		if err := os.MkdirAll(directory, directoryPermission); err != nil {
			return nil, fmt.Errorf("db_driver_sqlite_cgo: creating directory %q: %w", directory, err)
		}
	}

	busyTimeout := defaultBusyTimeoutMs
	if config.BusyTimeoutMs > 0 {
		busyTimeout = config.BusyTimeoutMs
	}
	cachePages := defaultCachePages
	if config.CachePages != 0 {
		cachePages = config.CachePages
	}
	mmapSize := defaultMmapSize
	if config.MmapSize > 0 {
		mmapSize = config.MmapSize
	}
	journalSizeLimit := defaultJournalSizeLimit
	if config.JournalSizeLimit > 0 {
		journalSizeLimit = config.JournalSizeLimit
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=%d&_foreign_keys=true", path, busyTimeout)

	database, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("db_driver_sqlite_cgo: opening database: %w", err)
	}

	database.SetMaxOpenConns(poolSize)
	database.SetMaxIdleConns(poolSize)
	database.SetConnMaxIdleTime(connMaxIdleTime)
	database.SetConnMaxLifetime(connMaxLifetime)

	if err := database.Ping(); err != nil {
		closeErr := database.Close()
		return nil, fmt.Errorf("db_driver_sqlite_cgo: pinging database: %w", errors.Join(err, closeErr))
	}

	if err := applyPragmas(database, busyTimeout, cachePages, mmapSize, journalSizeLimit); err != nil {
		closeErr := database.Close()
		return nil, fmt.Errorf("db_driver_sqlite_cgo: applying PRAGMAs: %w", errors.Join(err, closeErr))
	}

	return database, nil
}

// applyPragmas executes PRAGMA statements for performance and safety.
// modernc/sqlite does not parse DSN pragma parameters, so these are applied
// explicitly for consistency across both drivers.
func applyPragmas(database *sql.DB, busyTimeout, cachePages, mmapSize, journalSizeLimit int) error {
	pragmas := []struct {
		name  string
		value string
	}{
		{"journal_mode", "WAL"},
		{"wal_autocheckpoint", "1000"},
		{"busy_timeout", fmt.Sprintf(intFormat, busyTimeout)},
		{"synchronous", "NORMAL"},
		{"foreign_keys", "ON"},
		{"cell_size_check", "ON"},
		{"cache_size", fmt.Sprintf(intFormat, cachePages)},
		{"temp_store", "MEMORY"},
		{"mmap_size", fmt.Sprintf(intFormat, mmapSize)},
		{"journal_size_limit", fmt.Sprintf(intFormat, journalSizeLimit)},
		{"secure_delete", "OFF"},
	}

	for _, pragma := range pragmas {
		if _, err := database.Exec(fmt.Sprintf("PRAGMA %s = %s", pragma.name, pragma.value)); err != nil {
			return fmt.Errorf("PRAGMA %s: %w", pragma.name, err)
		}
	}

	return nil
}

// DriverName returns the database/sql driver name used by this package.
//
// Returns string which is "sqlite3".
func DriverName() string {
	return driverName
}
