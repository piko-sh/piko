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
	"errors"
	"path/filepath"
	"testing"

	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readPragma(t *testing.T, database *sql.DB, name string) string {
	t.Helper()

	var value string
	require.NoError(t, database.QueryRow("PRAGMA "+name).Scan(&value))

	return value
}

func TestOpenAppliesPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "open_pragmas.db")

	database, err := Open(context.Background(), path, Config{})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, database.Close()) })

	require.NoError(t, database.Ping())

	pragmas := map[string]string{
		"journal_mode":       "wal",
		"busy_timeout":       "10000",
		"foreign_keys":       "1",
		"synchronous":        "1",
		"cache_size":         "-20000",
		"temp_store":         "2",
		"mmap_size":          "67108864",
		"journal_size_limit": "33554432",
		"wal_autocheckpoint": "1000",
		"cell_size_check":    "1",
		"secure_delete":      "0",
	}
	for name, want := range pragmas {
		assert.Equalf(t, want, readPragma(t, database, name), "PRAGMA %s", name)
	}
}

func TestOpenResolvesConfigDefaults(t *testing.T) {
	tests := []struct {
		name                 string
		wantBusyTimeout      string
		wantCachePages       string
		wantMmapSize         string
		wantJournalSizeLimit string
		config               Config
	}{
		{
			name:                 "all defaults",
			config:               Config{},
			wantBusyTimeout:      "10000",
			wantCachePages:       "-20000",
			wantMmapSize:         "67108864",
			wantJournalSizeLimit: "33554432",
		},
		{
			name: "all overridden",
			config: Config{
				BusyTimeoutMs:    7777,
				CachePages:       -4096,
				MmapSize:         1 << 20,
				JournalSizeLimit: 1 << 21,
			},
			wantBusyTimeout:      "7777",
			wantCachePages:       "-4096",
			wantMmapSize:         "1048576",
			wantJournalSizeLimit: "2097152",
		},
		{
			name:                 "positive cache pages override",
			config:               Config{CachePages: 500},
			wantBusyTimeout:      "10000",
			wantCachePages:       "500",
			wantMmapSize:         "67108864",
			wantJournalSizeLimit: "33554432",
		},
		{
			name:                 "zero values keep defaults",
			config:               Config{},
			wantBusyTimeout:      "10000",
			wantCachePages:       "-20000",
			wantMmapSize:         "67108864",
			wantJournalSizeLimit: "33554432",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "defaults.db")
			database, err := Open(context.Background(), path, test.config)
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, database.Close()) })

			assert.Equal(t, test.wantBusyTimeout, readPragma(t, database, "busy_timeout"))
			assert.Equal(t, test.wantCachePages, readPragma(t, database, "cache_size"))
			assert.Equal(t, test.wantMmapSize, readPragma(t, database, "mmap_size"))
			assert.Equal(t, test.wantJournalSizeLimit, readPragma(t, database, "journal_size_limit"))
		})
	}
}

func TestOpenCreatesParentDirectory(t *testing.T) {

	path := filepath.Join(t.TempDir(), "nested", "child", "created.db")

	database, err := Open(context.Background(), path, Config{})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, database.Close()) })

	require.NoError(t, database.Ping())
	assert.FileExists(t, path)
}

func TestOpenEmptyPath(t *testing.T) {
	database, err := Open(context.Background(), "", Config{})
	require.Error(t, err)
	assert.Nil(t, database)
	assert.ErrorContains(t, err, "path must not be empty")
}

func TestOpenReappliesPragmasOnNewConnections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recycled.db")

	config := Config{
		BusyTimeoutMs:    4321,
		CachePages:       -1000,
		MmapSize:         4096,
		JournalSizeLimit: 8192,
	}
	database, err := Open(context.Background(), path, config)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, database.Close()) })

	database.SetMaxIdleConns(0)

	_, err = database.ExecContext(t.Context(), "CREATE TEMP TABLE connection_marker (value INTEGER)")
	require.NoError(t, err)

	var markerCount int
	require.NoError(t, database.QueryRowContext(t.Context(),
		"SELECT count(*) FROM sqlite_temp_master WHERE name = 'connection_marker'").Scan(&markerCount))
	require.Zero(t, markerCount, "the pool must hand out a fresh connection once idle connections are discarded")

	pragmas := map[string]string{
		"busy_timeout":       "4321",
		"cache_size":         "-1000",
		"mmap_size":          "4096",
		"journal_size_limit": "8192",
		"journal_mode":       "wal",
		"foreign_keys":       "1",
		"synchronous":        "1",
		"temp_store":         "2",
		"wal_autocheckpoint": "1000",
		"cell_size_check":    "1",
		"secure_delete":      "0",
	}
	for name, want := range pragmas {
		assert.Equalf(t, want, readPragma(t, database, name), "PRAGMA %s on a recycled connection", name)
	}
}

func TestOpenFailsWhenPathIsDirectory(t *testing.T) {
	database, err := Open(context.Background(), t.TempDir(), Config{})
	require.Error(t, err)
	assert.Nil(t, database)
	assert.ErrorContains(t, err, "opening first connection")
}

func TestResolveConfig(t *testing.T) {
	tests := []struct {
		name                 string
		config               Config
		wantBusyTimeout      int
		wantCachePages       int
		wantMmapSize         int
		wantJournalSizeLimit int
	}{
		{
			name:                 "all defaults",
			config:               Config{},
			wantBusyTimeout:      defaultBusyTimeoutMs,
			wantCachePages:       defaultCachePages,
			wantMmapSize:         defaultMmapSize,
			wantJournalSizeLimit: defaultJournalSizeLimit,
		},
		{
			name: "all overridden",
			config: Config{
				BusyTimeoutMs:    1234,
				CachePages:       -8192,
				MmapSize:         2048,
				JournalSizeLimit: 4096,
			},
			wantBusyTimeout:      1234,
			wantCachePages:       -8192,
			wantMmapSize:         2048,
			wantJournalSizeLimit: 4096,
		},
		{
			name:                 "negative sizes fall back to defaults",
			config:               Config{BusyTimeoutMs: -1, MmapSize: -1, JournalSizeLimit: -1},
			wantBusyTimeout:      defaultBusyTimeoutMs,
			wantCachePages:       defaultCachePages,
			wantMmapSize:         defaultMmapSize,
			wantJournalSizeLimit: defaultJournalSizeLimit,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			busyTimeout, cachePages, mmapSize, journalSizeLimit := resolveConfig(test.config)
			assert.Equal(t, test.wantBusyTimeout, busyTimeout)
			assert.Equal(t, test.wantCachePages, cachePages)
			assert.Equal(t, test.wantMmapSize, mmapSize)
			assert.Equal(t, test.wantJournalSizeLimit, journalSizeLimit)
		})
	}
}

func TestConnectionPragmasStartWithBusyTimeout(t *testing.T) {
	pragmas := connectionPragmas(1, 2, 3, 4)

	want := []pragmaAssignment{
		{name: "busy_timeout", value: "1"},
		{name: "journal_mode", value: "WAL"},
		{name: "wal_autocheckpoint", value: "1000"},
		{name: "synchronous", value: "NORMAL"},
		{name: "foreign_keys", value: "ON"},
		{name: "cell_size_check", value: "ON"},
		{name: "cache_size", value: "2"},
		{name: "temp_store", value: "MEMORY"},
		{name: "mmap_size", value: "3"},
		{name: "journal_size_limit", value: "4"},
		{name: "secure_delete", value: "OFF"},
	}
	assert.Equal(t, want, pragmas)
}

func TestApplyPragmasFailsOnClosedConnection(t *testing.T) {
	connection, err := (&sqlite3.SQLiteDriver{}).Open("file:" + filepath.Join(t.TempDir(), "closed.db"))
	require.NoError(t, err)
	sqliteConnection, ok := connection.(*sqlite3.SQLiteConn)
	require.True(t, ok)
	require.NoError(t, sqliteConnection.Close())

	err = applyPragmas(sqliteConnection, connectionPragmas(defaultBusyTimeoutMs, defaultCachePages, defaultMmapSize, defaultJournalSizeLimit))
	require.Error(t, err)
	assert.ErrorContains(t, err, "PRAGMA busy_timeout")
}

func TestPragmaConnector(t *testing.T) {
	connector := newPragmaConnector("file:"+filepath.Join(t.TempDir(), "connector.db"), connectionPragmas(250, defaultCachePages, defaultMmapSize, defaultJournalSizeLimit))

	assert.Same(t, connector.driver, connector.Driver())

	cancelledContext, cancel := context.WithCancelCause(t.Context())
	cancel(errors.New("caller gave up"))
	connection, err := connector.Connect(cancelledContext)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, connection)

	database := sql.OpenDB(connector)
	t.Cleanup(func() { assert.NoError(t, database.Close()) })
	assert.Equal(t, "250", readPragma(t, database, "busy_timeout"))
}

func TestSQLiteFilePathEscaper(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "plain path untouched", path: "/var/data/app.db", want: "/var/data/app.db"},
		{name: "memory form preserved", path: ":memory:", want: ":memory:"},
		{name: "percent encoded first", path: "a%b", want: "a%25b"},
		{name: "question mark encoded", path: "a?b", want: "a%3Fb"},
		{name: "hash encoded", path: "a#b", want: "a%23b"},
		{
			name: "all reserved characters",
			path: "/p/%?#.db",
			want: "/p/%25%3F%23.db",
		},
		{name: "colon preserved", path: "c:/db.sqlite", want: "c:/db.sqlite"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, sqliteFilePathEscaper.Replace(test.path))
		})
	}
}

func TestDriverName(t *testing.T) {
	assert.Equal(t, "sqlite3", DriverName())
}
