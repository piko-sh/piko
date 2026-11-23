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

package counts

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
)

func TestGeneratedDatabaseQueries(t *testing.T) {
	const countParentsAndNotes = "SELECT (SELECT count(*) FROM readings WHERE id=99), (SELECT count(*) FROM notes)"
	const readParentNote = "SELECT p.label, n.body FROM readings AS p JOIN notes AS n ON p.id=n.parent_id"
	ctx := t.Context()
	root := os.Getenv("PIKO_DB_ROOT")
	dbPath := filepath.Join(t.TempDir(), "generated.db")
	d, err := sql.Open("sqlite", dbPath)
//line runtime_test.go:41
	require.NoError(t, err)
	defer d.Close()
	d.SetMaxOpenConns(1)
	_, err = d.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	svc, err := db.NewMigrator(db_engine_sqlite.SQLite(), d, os.DirFS(root), "migrations")
	require.NoError(t, err)
	for _, n := range []int{2, 0} {
		got, err := svc.Up(ctx)
		require.NoError(t, err)
		require.Equal(t, n, got, "PIKO_CHECK db-migration-count: migration idempotence")
	}
	states, err := svc.Status(ctx)
	require.NoError(t, err)
	require.Len(t, states, 2)
	for i, s := range states {
		require.EqualValues(t, i+1, s.Version, "PIKO_CHECK db-migration-version: ordered versions")
		require.False(t, s.AppliedAt.IsZero())
	}
	require.NoError(t, svc.Validate(ctx))
	type row struct {
		ID       int64
		Label    string
		Quantity string
	}
	rows := map[int64]row{}
	queries := New(d)
	draws, err := hex.DecodeString(os.Getenv("PIKO_FUZZ_DRAWS"))
	require.NoError(t, err)
	instructions := append([]byte{0, 3, 1, 4, 2}, draws...)
	for step, b := range instructions {
		id := int64((b/3)%6 + 1)
		r, present := rows[id]
		switch b % 3 {
		case 0:
			if !present {
				label := fmt.Sprintf("row-%d-%d", step, b)
				require.NoError(t, queries.Add(ctx, AddParams{ID: id, Label: label}))
				rows[id] = row{ID: id, Label: label, Quantity: "0"}
			}
		case 1:
			q := int64(int(b) - 80)
			require.NoError(t, queries.Change(ctx, ChangeParams{ID: id, Quantity: int32(q)}))
			if present {
				r.Quantity = fmt.Sprint(q)
				rows[id] = r
			}
		case 2:
			require.NoError(t, queries.Remove(ctx, id))
			delete(rows, id)
		}
		fetched, err := queries.ScanRows(ctx)
		require.NoError(t, err)
		got := []row{}
		for _, r := range fetched {
			got = append(got, row{ID: r.ID, Label: r.Label, Quantity: fmt.Sprint(r.Quantity)})
		}
		want := []row{}
		for _, r := range rows {
			want = append(want, r)
		}
		slices.SortFunc(want, func(a, b row) int { return int(a.ID - b.ID) })
		if os.Getenv("PIKO_DB_SORT") == "DESC" {
			slices.Reverse(want)
		}
		require.Equal(t, want, got, "PIKO_CHECK db-rows: step %d operation %d", step, b)
	}
	n, err := svc.Down(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	n, err = svc.Up(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	fetched, err := queries.ScanRows(ctx)
	require.NoError(t, err)
	require.Len(t, fetched, len(rows))
	for _, r := range fetched {
		require.Equal(t, rows[r.ID].Label, r.Label)
		require.Equal(t, "0", fmt.Sprint(r.Quantity), "PIKO_CHECK db-column-default: reapplying migration")
	}

	label := fmt.Sprintf("parent-%x", draws[:min(len(draws), 4)])
	note := "note-" + label
	require.Error(t, queries.RunInTx(ctx, d, func(q *Queries) error {
		if e := q.Add(ctx, AddParams{ID: 99, Label: label}); e != nil {
			return e
		}
		if e := q.AddNote(ctx, note); e != nil {
			return e
		}
		return q.AddNote(ctx, "")
	}), "PIKO_CHECK db-abort: child constraint rejects entire transaction")
	var parents, children int
	require.NoError(t, d.QueryRowContext(ctx, countParentsAndNotes).Scan(&parents, &children))
	require.Equal(t, [2]int{}, [2]int{parents, children}, "PIKO_CHECK db-rollback: neither write survived")
	require.Error(t, queries.AddNote(ctx, note), "PIKO_CHECK db-foreign-key: missing owner")
	require.NoError(t, queries.RunInTx(ctx, d, func(q *Queries) error {
		if e := q.Add(ctx, AddParams{ID: 99, Label: label}); e != nil {
			return e
		}
		return q.AddNote(ctx, note)
	}))
	require.NoError(t, d.Close())
	reopened, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer reopened.Close()
	var savedLabel, savedNote string
	require.NoError(t, reopened.QueryRowContext(ctx, readParentNote).Scan(&savedLabel, &savedNote))
	require.Equal(t, [2]string{label, note}, [2]string{savedLabel, savedNote}, "PIKO_CHECK db-reopen: committed relation and values")

	broken := filepath.Join(root, "migrations", "003_notes_index.up.sql")
	require.NoError(t, os.WriteFile(broken, []byte("CREATE INDEX notes_by_parent ON notes(parent_id); SELECT missing_column FROM notes;"), 0600))
	recover, err := db.NewMigrator(db_engine_sqlite.SQLite(), reopened, os.DirFS(root), "migrations")
	require.NoError(t, err)
	_, err = recover.Up(ctx)
	require.Error(t, err, "PIKO_CHECK db-failed-migration: SQL error returned")
	var indexes int
	require.NoError(t, reopened.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='index' AND name='notes_by_parent'").Scan(&indexes))
	require.Zero(t, indexes, "PIKO_CHECK db-migration-atomic: failing migration left its index visible")
	require.NoError(t, os.WriteFile(broken, []byte("CREATE INDEX notes_by_parent ON notes(parent_id);"), 0600))
	repaired, err := db.NewMigrator(db_engine_sqlite.SQLite(), reopened, os.DirFS(root), "migrations")
	require.NoError(t, err)
	n, err = repaired.Up(ctx)
	require.NoError(t, err, "PIKO_CHECK db-migration-repair: corrected migration")
	require.Equal(t, 1, n)
	require.NoError(t, reopened.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='index' AND name='notes_by_parent'").Scan(&indexes))
	require.Equal(t, 1, indexes, "PIKO_CHECK db-migration-repair: exactly one repaired index")
	require.NoError(t, repaired.Validate(ctx))
}
