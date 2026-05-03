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

package migration_sql_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_domain"
)

func TestNewFSFileReader_ImplementsFileReaderPort(t *testing.T) {
	t.Parallel()

	var _ querier_domain.FileReaderPort = migration_sql.NewFSFileReader(fstest.MapFS{})
}

func TestFSFileReader_ReadFile_ReturnsContents(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"migrations/0001_init.up.sql": &fstest.MapFile{Data: []byte("CREATE TABLE foo (id INT);")},
	}
	reader := migration_sql.NewFSFileReader(fs)

	got, err := reader.ReadFile(context.Background(), "migrations/0001_init.up.sql")

	require.NoError(t, err)
	require.Equal(t, []byte("CREATE TABLE foo (id INT);"), got)
}

func TestFSFileReader_ReadFile_ReturnsErrorWhenMissing(t *testing.T) {
	t.Parallel()

	reader := migration_sql.NewFSFileReader(fstest.MapFS{})

	_, err := reader.ReadFile(context.Background(), "missing.sql")

	require.Error(t, err)
}

func TestFSFileReader_ReadDir_ReturnsSortedEntries(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"migrations/0002_alter.up.sql": &fstest.MapFile{Data: []byte("ALTER TABLE")},
		"migrations/0001_init.up.sql":  &fstest.MapFile{Data: []byte("CREATE TABLE")},
		"migrations/0003_data.up.sql":  &fstest.MapFile{Data: []byte("INSERT INTO")},
	}
	reader := migration_sql.NewFSFileReader(fs)

	entries, err := reader.ReadDir(context.Background(), "migrations")

	require.NoError(t, err)
	require.Len(t, entries, 3)
	require.Equal(t, "0001_init.up.sql", entries[0].Name())
	require.Equal(t, "0002_alter.up.sql", entries[1].Name())
	require.Equal(t, "0003_data.up.sql", entries[2].Name())
}

func TestFSFileReader_ReadDir_ReturnsErrorWhenDirectoryMissing(t *testing.T) {
	t.Parallel()

	reader := migration_sql.NewFSFileReader(fstest.MapFS{})

	_, err := reader.ReadDir(context.Background(), "no-such-directory")

	require.Error(t, err)
	require.Contains(t, err.Error(), "no-such-directory")
}

func TestMigrationFileReader_RejectsOversizedFile(t *testing.T) {
	t.Parallel()

	oversizedContent := make([]byte, 4096)
	for i := range oversizedContent {
		oversizedContent[i] = 'A'
	}

	fs := fstest.MapFS{
		"migrations/0001_huge.up.sql": &fstest.MapFile{Data: oversizedContent},
	}
	reader := migration_sql.NewFSFileReader(fs, migration_sql.WithMaxMigrationFileBytes(1024))

	_, err := reader.ReadFile(context.Background(), "migrations/0001_huge.up.sql")

	require.Error(t, err)
	require.ErrorIs(t, err, migration_sql.ErrMigrationFileTooLarge)
}

func TestMigrationFileReader_AcceptsSmallFileAtBoundary(t *testing.T) {
	t.Parallel()

	atCap := make([]byte, 1024)
	for i := range atCap {
		atCap[i] = 'A'
	}

	fs := fstest.MapFS{
		"migrations/0001_at_cap.up.sql": &fstest.MapFile{Data: atCap},
	}
	reader := migration_sql.NewFSFileReader(fs, migration_sql.WithMaxMigrationFileBytes(1024))

	got, err := reader.ReadFile(context.Background(), "migrations/0001_at_cap.up.sql")

	require.NoError(t, err)
	require.Len(t, got, 1024)
}

func TestMigrationFileReader_NonPositiveOptionFallsBackToDefault(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"migrations/0001_init.up.sql": &fstest.MapFile{Data: []byte("SELECT 1")},
	}
	reader := migration_sql.NewFSFileReader(fs, migration_sql.WithMaxMigrationFileBytes(0))

	got, err := reader.ReadFile(context.Background(), "migrations/0001_init.up.sql")

	require.NoError(t, err)
	require.Equal(t, "SELECT 1", string(got))
}
