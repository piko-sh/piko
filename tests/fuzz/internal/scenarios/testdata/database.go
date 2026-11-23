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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
)

func TestGeneratedPikoScenario(t *testing.T) {
	const table = "readings"
	const direction = "ASC"
	const kind = "INTEGER"
	files := fstest.MapFS{}
	put := func(name, body string) { files[name] = &fstest.MapFile{Data: []byte(body)} }
	put("migrations/001_table.up.sql", fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, label TEXT NOT NULL); CREATE TABLE notes(parent_id INTEGER NOT NULL REFERENCES %s(id), body TEXT NOT NULL CHECK(length(body)>0));", table, table))
	put("migrations/001_table.down.sql", "DROP TABLE notes; DROP TABLE "+table+";")
	put("migrations/002_count.up.sql", fmt.Sprintf("ALTER TABLE %s ADD COLUMN quantity %s NOT NULL DEFAULT 0;", table, kind))
	put("migrations/002_count.down.sql", "ALTER TABLE "+table+" DROP COLUMN quantity;")
	put("queries/counts.sql", fmt.Sprintf(`
-- piko.query(Add, exec)
INSERT INTO %[1]s(id,label) VALUES (?,?);
-- piko.query(AddNote, exec)
INSERT INTO notes(parent_id,body) VALUES (99,?);
-- piko.query(Change, exec)
UPDATE %[1]s SET quantity=? WHERE id=?;
-- piko.query(Remove, exec)
DELETE FROM %[1]s WHERE id=?;
-- piko.query(ScanRows, many)
SELECT id,label,quantity FROM %[1]s ORDER BY id %[2]s;
`, table, direction))
	engine := db_engine_sqlite.SQLite()
	result, err := db.Generate(t.Context(), engine, db.GenerateRequest{Files: files, PackageName: "counts", MigrationDirectory: "migrations", QueryDirectory: "queries", Driver: db.DriverDatabaseSQL})
//line piko_fuzz_scenario_test.go:63
	require.NoError(t, err, "PIKO_CHECK db-generate: schema and query analysis")
	for _, d := range result.Diagnostics {
		require.NotEqual(t, db.SeverityError, d.Severity, "PIKO_CHECK db-diagnostic: %v", d)
	}
	require.Equal(t, 5, result.QueryCount, "PIKO_CHECK db-query-count: all typed methods")
	tmp := t.TempDir()
	args := []string{"test", "-count=1", "-v", "-run=^TestGeneratedDatabaseQueries$"}
	for _, file := range result.Files {
		p := filepath.Join(tmp, file.Name)
		require.NoError(t, os.WriteFile(p, file.Content, 0600))
		args = append(args, p)
	}
	for p, f := range files {
		dst := filepath.Join(tmp, p)
		require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0700))
		require.NoError(t, os.WriteFile(dst, f.Data, 0600))
	}
	fixture, err := os.ReadFile(os.Getenv("PIKO_FUZZ_DATABASE_RUNTIME"))
	require.NoError(t, err)
	fixturePath := filepath.Join(tmp, "runtime_test.go")
//line piko_fuzz_scenario_test.go:89
	require.NoError(t, os.WriteFile(fixturePath, fixture, 0600))
	cmd := exec.CommandContext(t.Context(), "go", append(args, fixturePath)...)
	cmd.Env = append(os.Environ(), "PIKO_DB_ROOT="+tmp, "PIKO_DB_SORT="+direction)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running emitted queries: %v\n%s", err, b)
	}
	require.Contains(t, string(b), "--- PASS: TestGeneratedDatabaseQueries", "PIKO_CHECK db-execution: actual generated code must run")
}
