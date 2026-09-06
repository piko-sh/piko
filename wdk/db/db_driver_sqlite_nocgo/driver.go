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

package db_driver_sqlite_nocgo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // register "sqlite" database/sql driver

	"piko.sh/piko/wdk/safedisk"
)

const (
	// driverName is the database/sql driver registration name for modernc SQLite.
	driverName = "sqlite"

	// defaultBusyTimeoutMs caps the SQLite busy wait at 10 seconds.
	defaultBusyTimeoutMs = 10_000

	// defaultCachePages requests a page cache of approximately 20 MB (negative values denote
	// kibibytes in SQLite cache_size).
	defaultCachePages = -20_000

	// defaultMmapSize sets memory-mapped I/O to 64 MiB.
	defaultMmapSize = 64 * 1024 * 1024

	// defaultJournalSizeLimit caps the WAL journal at 32 MiB.
	defaultJournalSizeLimit = 32 * 1024 * 1024

	// poolSize fixes the connection pool to one open connection, matching SQLite's
	// single-writer model.
	poolSize = 1

	// connMaxIdleTime is how long idle connections may remain in the pool.
	connMaxIdleTime = 5 * time.Minute

	// connMaxLifetime caps the lifetime of any individual connection.
	connMaxLifetime = 1 * time.Hour

	// pragmaParameter is the modernc DSN query parameter whose values the driver executes as
	// PRAGMA statements on every connection it opens.
	pragmaParameter = "_pragma"
)

var (
	// sqliteFilePathEscaper percent-encodes the characters that would otherwise corrupt a
	// SQLite file: URI when interpolated into the DSN path component.
	//
	// "%" is encoded first (and strings.NewReplacer performs a single non-overlapping pass,
	// so the encoded sequences are not re-encoded). "/" and ":" are preserved so ordinary
	// paths and the ":memory:" form continue to work. SQLite percent-decodes the path,
	// restoring the original bytes.
	sqliteFilePathEscaper = strings.NewReplacer(
		"%", "%25",
		"?", "%3F",
		"#", "%23",
	)
)

// Config holds configuration for opening a SQLite database.
type Config struct {
	// BusyTimeoutMs is the timeout in milliseconds for SQLite busy waits. Zero uses the
	// default (10000).
	BusyTimeoutMs int

	// CachePages sets the number of pages to keep in the SQLite cache. Zero uses the default
	// (-20000, approximately 20 MB).
	CachePages int

	// MmapSize sets the memory-mapped I/O size in bytes. Zero uses the default (64 MB).
	MmapSize int

	// JournalSizeLimit sets the maximum size in bytes for the WAL journal. Zero uses the
	// default (32 MB).
	JournalSizeLimit int
}

// Open opens a SQLite database at the given path with production-ready PRAGMAs applied.
// The returned *sql.DB is configured with a single connection (SQLite's single-writer
// model) and WAL mode enabled.
//
// Every PRAGMA is carried in the DSN, so the driver applies the full set to each
// connection it opens. Most of them are per-connection settings, so applying them once
// through the pool would let them revert whenever the pool replaced a connection that had
// reached its idle or lifetime limit.
//
// Takes path (string) which is the filesystem path to the SQLite database file.
// Takes config (Config) which provides optional tuning parameters.
//
// Returns *sql.DB which is the configured database connection.
// Returns error when the database cannot be opened or PRAGMAs fail to apply.
func Open(ctx context.Context, path string, config Config) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("db_driver_sqlite_nocgo: path must not be empty")
	}

	if directory := filepath.Dir(path); directory != "." && directory != "" {
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		if err != nil {
			return nil, fmt.Errorf("db_driver_sqlite_nocgo: creating directory %q: %w", directory, err)
		}
		_ = sandbox.Close()
	}

	database, err := sql.Open(driverName, buildDSN(path, connectionPragmas(resolveConfig(config))))
	if err != nil {
		return nil, fmt.Errorf("db_driver_sqlite_nocgo: opening database: %w", err)
	}

	database.SetMaxOpenConns(poolSize)
	database.SetMaxIdleConns(poolSize)
	database.SetConnMaxIdleTime(connMaxIdleTime)
	database.SetConnMaxLifetime(connMaxLifetime)

	if err := database.PingContext(ctx); err != nil {
		closeErr := database.Close()
		return nil, fmt.Errorf("db_driver_sqlite_nocgo: opening first connection with PRAGMAs: %w", errors.Join(err, closeErr))
	}

	return database, nil
}

// resolveConfig applies the package defaults to any unset Config fields.
//
// Takes config (Config) which provides optional tuning parameters.
//
// Returns busyTimeout (int) which is the resolved busy timeout in milliseconds.
// Returns cachePages (int) which is the resolved cache size in pages or KiB.
// Returns mmapSize (int) which is the resolved memory-mapped I/O size in bytes.
// Returns journalSizeLimit (int) which is the resolved WAL journal size limit in bytes.
func resolveConfig(config Config) (busyTimeout, cachePages, mmapSize, journalSizeLimit int) {
	busyTimeout = defaultBusyTimeoutMs
	if config.BusyTimeoutMs > 0 {
		busyTimeout = config.BusyTimeoutMs
	}
	cachePages = defaultCachePages
	if config.CachePages != 0 {
		cachePages = config.CachePages
	}
	mmapSize = defaultMmapSize
	if config.MmapSize > 0 {
		mmapSize = config.MmapSize
	}
	journalSizeLimit = defaultJournalSizeLimit
	if config.JournalSizeLimit > 0 {
		journalSizeLimit = config.JournalSizeLimit
	}

	return busyTimeout, cachePages, mmapSize, journalSizeLimit
}

// connectionPragmas lists the PRAGMA assignments applied to every connection, in the form
// modernc's _pragma DSN parameter executes verbatim.
//
// Every value is either a fixed keyword or an integer formatted here, so no configured
// text reaches the PRAGMA statements.
//
// Takes busyTimeout (int) which sets PRAGMA busy_timeout in milliseconds.
// Takes cachePages (int) which sets PRAGMA cache_size in pages or KiB.
// Takes mmapSize (int) which sets PRAGMA mmap_size in bytes.
// Takes journalSizeLimit (int) which sets PRAGMA journal_size_limit in bytes.
//
// Returns []string which holds one name(value) assignment per PRAGMA.
func connectionPragmas(busyTimeout, cachePages, mmapSize, journalSizeLimit int) []string {
	return []string{
		fmt.Sprintf("busy_timeout(%d)", busyTimeout),
		"journal_mode(WAL)",
		"wal_autocheckpoint(1000)",
		"synchronous(NORMAL)",
		"foreign_keys(ON)",
		"cell_size_check(ON)",
		fmt.Sprintf("cache_size(%d)", cachePages),
		"temp_store(MEMORY)",
		fmt.Sprintf("mmap_size(%d)", mmapSize),
		fmt.Sprintf("journal_size_limit(%d)", journalSizeLimit),
		"secure_delete(OFF)",
	}
}

// buildDSN builds the modernc file: URI for path with each PRAGMA as a _pragma parameter.
//
// modernc runs busy_timeout first and the remaining PRAGMAs in name order whenever it
// opens a connection.
//
// Takes path (string) which is the database file path.
// Takes pragmas ([]string) which holds the name(value) assignments to apply.
//
// Returns string which is the DSN passed to sql.Open.
func buildDSN(path string, pragmas []string) string {
	query := url.Values{pragmaParameter: pragmas}
	return "file:" + sqliteFilePathEscaper.Replace(path) + "?" + query.Encode()
}

// DriverName returns the database/sql driver name used for nocgo SQLite.
//
// Returns string which is "sqlite".
func DriverName() string {
	return driverName
}
