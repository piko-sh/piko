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
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"

	"piko.sh/piko/wdk/safedisk"
)

const (
	// driverName is the database/sql driver registration name for SQLite cgo.
	driverName = "sqlite3"

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

// pragmaAssignment is one PRAGMA and the value assigned to it on every connection.
type pragmaAssignment struct {
	// name is the PRAGMA name.
	name string

	// value is the literal assigned to the PRAGMA, expressed as a fixed keyword or formatted
	// integer.
	value string
}

// pragmaConnector opens mattn/go-sqlite3 connections whose connect hook applies the
// PRAGMA assignments to each new connection before the pool sees it.
type pragmaConnector struct {
	// driver is the mattn driver carrying the connect hook.
	driver *sqlite3.SQLiteDriver

	// dsn is the file: URI each connection opens.
	dsn string
}

// Connect opens a new connection with every PRAGMA applied.
//
// Returns driver.Conn which is the configured connection.
// Returns error when ctx is already done, the database cannot be opened, or a PRAGMA
// fails.
func (connector *pragmaConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return connector.driver.Open(connector.dsn)
}

// Driver returns the underlying mattn driver.
//
// Returns driver.Driver which is the mattn driver carrying the connect hook.
func (connector *pragmaConnector) Driver() driver.Driver {
	return connector.driver
}

// Open opens a SQLite database at the given path with production-ready PRAGMAs applied.
// The returned *sql.DB is configured with a single connection (SQLite's single-writer
// model) and WAL mode enabled.
//
// The PRAGMAs run in a connect hook, so every connection the pool opens receives the full
// set. Most of them are per-connection settings, so applying them once through the pool
// would let them revert whenever the pool replaced a connection that had reached its idle
// or lifetime limit.
//
// Takes path (string) which is the filesystem path to the SQLite database file.
// Takes config (Config) which provides optional tuning parameters.
//
// Returns *sql.DB which is the configured database connection.
// Returns error when the database cannot be opened or PRAGMAs fail to apply.
func Open(ctx context.Context, path string, config Config) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("db_driver_sqlite_cgo: path must not be empty")
	}

	if directory := filepath.Dir(path); directory != "." && directory != "" {
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		if err != nil {
			return nil, fmt.Errorf("db_driver_sqlite_cgo: creating directory %q: %w", directory, err)
		}
		_ = sandbox.Close()
	}

	busyTimeout, cachePages, mmapSize, journalSizeLimit := resolveConfig(config)
	database := sql.OpenDB(newPragmaConnector(
		"file:"+sqliteFilePathEscaper.Replace(path),
		connectionPragmas(busyTimeout, cachePages, mmapSize, journalSizeLimit),
	))

	database.SetMaxOpenConns(poolSize)
	database.SetMaxIdleConns(poolSize)
	database.SetConnMaxIdleTime(connMaxIdleTime)
	database.SetConnMaxLifetime(connMaxLifetime)

	if err := database.PingContext(ctx); err != nil {
		closeErr := database.Close()
		return nil, fmt.Errorf("db_driver_sqlite_cgo: opening first connection with PRAGMAs: %w", errors.Join(err, closeErr))
	}

	return database, nil
}

// DriverName returns the database/sql driver name used for cgo SQLite.
//
// Returns string which is "sqlite3".
func DriverName() string {
	return driverName
}

// newPragmaConnector creates a connector that opens dsn and applies pragmas to every new
// connection.
//
// Takes dsn (string) which is the file: URI each connection opens.
// Takes pragmas ([]pragmaAssignment) which are applied in order to each connection.
//
// Returns *pragmaConnector which is ready for sql.OpenDB.
func newPragmaConnector(dsn string, pragmas []pragmaAssignment) *pragmaConnector {
	return &pragmaConnector{
		dsn: dsn,
		driver: &sqlite3.SQLiteDriver{
			ConnectHook: func(connection *sqlite3.SQLiteConn) error {
				return applyPragmas(connection, pragmas)
			},
		},
	}
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

// connectionPragmas lists the PRAGMA assignments applied to every connection, busy
// timeout first so the later assignments wait for locks rather than failing.
//
// Every value is either a fixed keyword or an integer formatted here, so no configured
// text reaches the PRAGMA statements.
//
// Takes busyTimeout (int) which sets PRAGMA busy_timeout in milliseconds.
// Takes cachePages (int) which sets PRAGMA cache_size in pages or KiB.
// Takes mmapSize (int) which sets PRAGMA mmap_size in bytes.
// Takes journalSizeLimit (int) which sets PRAGMA journal_size_limit in bytes.
//
// Returns []pragmaAssignment which holds the assignments in application order.
func connectionPragmas(busyTimeout, cachePages, mmapSize, journalSizeLimit int) []pragmaAssignment {
	return []pragmaAssignment{
		{name: "busy_timeout", value: strconv.Itoa(busyTimeout)},
		{name: "journal_mode", value: "WAL"},
		{name: "wal_autocheckpoint", value: "1000"},
		{name: "synchronous", value: "NORMAL"},
		{name: "foreign_keys", value: "ON"},
		{name: "cell_size_check", value: "ON"},
		{name: "cache_size", value: strconv.Itoa(cachePages)},
		{name: "temp_store", value: "MEMORY"},
		{name: "mmap_size", value: strconv.Itoa(mmapSize)},
		{name: "journal_size_limit", value: strconv.Itoa(journalSizeLimit)},
		{name: "secure_delete", value: "OFF"},
	}
}

// applyPragmas executes each PRAGMA assignment on a single connection.
//
// Takes connection (*sqlite3.SQLiteConn) which is the connection being configured.
// Takes pragmas ([]pragmaAssignment) which are applied in order.
//
// Returns error when any PRAGMA statement fails to execute.
func applyPragmas(connection *sqlite3.SQLiteConn, pragmas []pragmaAssignment) error {
	for _, pragma := range pragmas {
		if _, err := connection.Exec("PRAGMA "+pragma.name+" = "+pragma.value, nil); err != nil {
			return fmt.Errorf("PRAGMA %s: %w", pragma.name, err)
		}
	}
	return nil
}
