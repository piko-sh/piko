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
	"path/filepath"
	"testing"

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
		config               Config
		wantBusyTimeout      string
		wantCachePages       string
		wantMmapSize         string
		wantJournalSizeLimit string
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
			config:               Config{BusyTimeoutMs: 0, CachePages: 0, MmapSize: 0, JournalSizeLimit: 0},
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

func TestApplyPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apply_pragmas.db")

	database, err := sql.Open(driverName, "file:"+path)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, database.Close()) })
	require.NoError(t, database.Ping())

	require.NoError(t, applyPragmas(context.Background(), database, 4321, -1000, 4096, 8192))

	assert.Equal(t, "4321", readPragma(t, database, "busy_timeout"))
	assert.Equal(t, "-1000", readPragma(t, database, "cache_size"))
	assert.Equal(t, "4096", readPragma(t, database, "mmap_size"))
	assert.Equal(t, "8192", readPragma(t, database, "journal_size_limit"))
	assert.Equal(t, "wal", readPragma(t, database, "journal_mode"))
}

func TestApplyPragmasClosedDatabase(t *testing.T) {

	path := filepath.Join(t.TempDir(), "closed.db")

	database, err := sql.Open(driverName, "file:"+path)
	require.NoError(t, err)
	require.NoError(t, database.Ping())
	require.NoError(t, database.Close())

	err = applyPragmas(context.Background(), database, defaultBusyTimeoutMs, defaultCachePages, defaultMmapSize, defaultJournalSizeLimit)
	require.Error(t, err)
	assert.ErrorContains(t, err, "PRAGMA")
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
