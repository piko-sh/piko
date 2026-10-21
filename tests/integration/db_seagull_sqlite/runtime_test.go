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
package db_seagull_sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"

	"piko.sh/piko/tests/integration/db_seagull_sqlite/store"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
	"piko.sh/piko/wdk/maths"
)

func openDatabase(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "library.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func migrateAndSeed(t *testing.T, database *sql.DB) db.MigrationService {
	t.Helper()

	ctx := context.Background()
	files := os.DirFS(filepath.Join("testdata", "db"))

	migrator, err := db.NewMigrator(db_engine_sqlite.SQLite(), database, files, "migrations")
	require.NoError(t, err)
	applied, err := migrator.Up(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, applied)

	seeder, err := db.NewSeeder(db_engine_sqlite.SQLite(), database, files, "seeds")
	require.NoError(t, err)
	seeded, err := seeder.Apply(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, seeded)

	return migrator
}

func TestGeneratedQueriesRunAgainstMigratedDatabase(t *testing.T) {
	ctx := context.Background()
	database := openDatabase(t)
	migrateAndSeed(t, database)
	queries := store.New(database)

	author, err := queries.GetAuthor(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "Ursula K. Le Guin", author.Name)
	require.NotNil(t, author.BornYear)
	require.Equal(t, int32(1929), *author.BornYear)

	authors, err := queries.ListAuthors(ctx)
	require.NoError(t, err)
	require.Len(t, authors, 2)
	require.Nil(t, authors[1].BornYear)

	rating := int32(5)
	book, err := queries.CreateBook(ctx, store.CreateBookParams{
		AuthorID: 1,
		Title:    "The Dispossessed",
		Price:    maths.NewDecimalFromString("12.50"),
		Rating:   &rating,
	})
	require.NoError(t, err)
	require.Equal(t, "The Dispossessed", book.Title)
	equal, err := book.Price.EqualsString("12.5")
	require.NoError(t, err)
	require.True(t, equal)

	rows, err := queries.ListAuthorsWithBooks(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.NotNil(t, rows[0].BookID)
	require.Equal(t, book.ID, *rows[0].BookID)
	require.Nil(t, rows[1].BookID)
	require.Nil(t, rows[1].Title)

	count, err := queries.CountBooksByAuthor(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), count.BookCount)

	require.NoError(t, queries.DeleteBook(ctx, book.ID))
	count, err = queries.CountBooksByAuthor(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, count.BookCount)
}

func TestMigrationsUsePikoHistoryTables(t *testing.T) {
	ctx := context.Background()
	database := openDatabase(t)
	migrator := migrateAndSeed(t, database)

	for _, table := range []string{"piko_migrations", "piko_seeds"} {
		var name string
		require.NoError(t, database.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name))
		require.Equal(t, table, name)
	}

	statuses, err := migrator.Status(ctx)
	require.NoError(t, err)
	require.Len(t, statuses, 2)
	for _, status := range statuses {
		require.False(t, status.AppliedAt.IsZero(), "%s not applied", status.Name)
	}
	require.NoError(t, migrator.Validate(ctx))

	reverted, err := migrator.Down(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, 2, reverted)

	var remaining int
	require.NoError(t, database.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name IN ('authors', 'books')").Scan(&remaining))
	require.Zero(t, remaining)
}

func TestMigrationHooksRun(t *testing.T) {
	ctx := context.Background()
	database := openDatabase(t)
	files := os.DirFS(filepath.Join("testdata", "db"))

	var before, after []int64
	migrator, err := db.NewMigrator(db_engine_sqlite.SQLite(), database, files, "migrations",
		db.WithBeforeMigration(func(_ context.Context, hook db.MigrationHookContext) error {
			before = append(before, hook.Version)
			return nil
		}),
		db.WithAfterMigration(func(_ context.Context, hook db.MigrationHookContext) error {
			after = append(after, hook.Version)
			return nil
		}),
	)
	require.NoError(t, err)

	_, err = migrator.Up(ctx)
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, before)
	require.Equal(t, []int64{1, 2}, after)
}
