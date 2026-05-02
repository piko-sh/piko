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

package db_engine_conformance

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_clickhouse"
	"piko.sh/piko/wdk/db/db_engine_cockroachdb"
	"piko.sh/piko/wdk/db/db_engine_duckdb"
	"piko.sh/piko/wdk/db/db_engine_mariadb"
	"piko.sh/piko/wdk/db/db_engine_mysql"
	"piko.sh/piko/wdk/db/db_engine_postgres"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
	"piko.sh/piko/wdk/db/db_engine_timescaledb"
)

const (
	deepNestingDepth            = 100_000
	quotedWeirdName             = "weird name"
	expectedSplitStatementCount = 2
	beginAliasStatements        = "SELECT 1 AS begin; SELECT 2;"
)

type engineFixture struct {
	newEngine         func() querier_domain.EnginePort
	name              string
	baseMigration     string
	paramQuery        string
	splitterMigration string
	quotedMigration   string
	deepQuery         string
}

func deepNestedSelect(depth int) string {
	return "-- piko.query(name: Deep, command: one)\n" +
		"SELECT " + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth) + ";"
}

func postgresFamilyFixture(name string, newEngine func() querier_domain.EnginePort) engineFixture {
	return engineFixture{
		name:          name,
		newEngine:     newEngine,
		baseMigration: "CREATE TABLE items (id integer PRIMARY KEY, name text);",
		paramQuery: "-- piko.query(name: GetItem, command: one)\n" +
			"SELECT id, name FROM items WHERE id = $1;",
		splitterMigration: "CREATE TABLE alpha (id integer);\nCREATE TABLE beta (id integer);",
		quotedMigration:   `CREATE TABLE "weird name" (id integer);`,
		deepQuery:         deepNestedSelect(deepNestingDepth),
	}
}

func mysqlFamilyFixture(name string, newEngine func() querier_domain.EnginePort) engineFixture {
	return engineFixture{
		name:          name,
		newEngine:     newEngine,
		baseMigration: "CREATE TABLE items (id INT PRIMARY KEY, name VARCHAR(255));",
		paramQuery: "-- piko.query(name: GetItem, command: one)\n" +
			"SELECT id, name FROM items WHERE id = ?;",
		splitterMigration: "CREATE TABLE alpha (id INT);\nCREATE TABLE beta (id INT);",
		quotedMigration:   "CREATE TABLE `weird name` (id INT);",
		deepQuery:         deepNestedSelect(deepNestingDepth),
	}
}

func conformanceFixtures() []engineFixture {
	return []engineFixture{
		postgresFamilyFixture("postgres", func() querier_domain.EnginePort { return db_engine_postgres.NewPostgresEngine() }),
		postgresFamilyFixture("cockroachdb", func() querier_domain.EnginePort { return db_engine_cockroachdb.NewCockroachDBEngine() }),
		postgresFamilyFixture("timescaledb", func() querier_domain.EnginePort { return db_engine_timescaledb.NewTimescaleDBEngine() }),
		mysqlFamilyFixture("mysql", func() querier_domain.EnginePort { return db_engine_mysql.NewMySQLEngine() }),
		mysqlFamilyFixture("mariadb", func() querier_domain.EnginePort { return db_engine_mariadb.NewMariaDBEngine() }),
		{
			name:          "sqlite",
			newEngine:     func() querier_domain.EnginePort { return db_engine_sqlite.NewSQLiteEngine() },
			baseMigration: "CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT);",
			paramQuery: "-- piko.query(name: GetItem, command: one)\n" +
				"SELECT id, name FROM items WHERE id = ?;",
			splitterMigration: "CREATE TABLE alpha (id INTEGER);\nCREATE TABLE beta (id INTEGER);",
			quotedMigration:   `CREATE TABLE "weird name" (id INTEGER);`,
			deepQuery:         deepNestedSelect(deepNestingDepth),
		},
		{
			name:          "duckdb",
			newEngine:     func() querier_domain.EnginePort { return db_engine_duckdb.NewDuckDBEngine() },
			baseMigration: "CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT);",
			paramQuery: "-- piko.query(name: GetItem, command: one)\n" +
				"SELECT id, name FROM items WHERE id = $1;",
			splitterMigration: "CREATE TABLE alpha (id INTEGER);\nCREATE TABLE beta (id INTEGER);",
			quotedMigration:   `CREATE TABLE "weird name" (id INTEGER);`,
			deepQuery:         deepNestedSelect(deepNestingDepth),
		},
		{
			name:          "clickhouse",
			newEngine:     func() querier_domain.EnginePort { return db_engine_clickhouse.NewClickHouseEngine() },
			baseMigration: "CREATE TABLE items (id Int32, name String) ENGINE = MergeTree ORDER BY id;",
			paramQuery: "-- piko.query(name: GetItem, command: one)\n" +
				"SELECT id, name FROM items WHERE id = {id:Int32};",
			splitterMigration: "CREATE TABLE alpha (id Int32) ENGINE = MergeTree ORDER BY id;\n" +
				"CREATE TABLE beta (id Int32) ENGINE = MergeTree ORDER BY id;",
			quotedMigration: "CREATE TABLE `weird name` (id Int32) ENGINE = MergeTree ORDER BY id;",
			deepQuery:       deepNestedSelect(deepNestingDepth),
		},
	}
}

func TestConformance_StatementSplitting(t *testing.T) {
	t.Parallel()
	for _, fixture := range conformanceFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			emitter, _, generateError := runGenerate(
				context.Background(),
				fixture.newEngine(),
				map[string]string{"001_split.up.sql": fixture.splitterMigration},
				nil,
			)
			require.NoError(t, generateError, "generation should succeed")
			names := catalogueTableNames(emitter.catalogue)
			assert.Contains(t, names, "alpha", "first split statement should produce a table")
			assert.Contains(t, names, "beta", "second split statement should produce a table")
			assert.Len(t, names, expectedSplitStatementCount,
				"the two-statement migration must split into exactly two tables (no swallowed "+
					"or phantom statement), got %v", names)
		})
	}
}

func TestConformance_BeginAliasSplitsBothStatements(t *testing.T) {
	t.Parallel()
	for _, fixture := range conformanceFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			statements, parseError := fixture.newEngine().ParseStatements(beginAliasStatements)
			require.NoError(t, parseError,
				"parsing two statements with a begin alias should not error")
			assert.Len(t, statements, expectedSplitStatementCount,
				"%q must split into exactly two statements; a swallowed trailing statement means "+
					"the splitter mistook the begin alias for a procedural block opener",
				beginAliasStatements)
		})
	}
}

func TestConformance_ParameterTracking(t *testing.T) {
	t.Parallel()
	for _, fixture := range conformanceFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			emitter, _, generateError := runGenerate(
				context.Background(),
				fixture.newEngine(),
				map[string]string{"001_items.up.sql": fixture.baseMigration},
				map[string]string{"get_item.sql": fixture.paramQuery},
			)
			require.NoError(t, generateError, "generation should succeed")
			query := findQuery(emitter.queries, "GetItem")
			require.NotNil(t, query, "GetItem query should be analysed")
			require.Len(t, query.Parameters, 1, "exactly one parameter expected")
			assert.Equal(t, 1, query.Parameters[0].Number, "parameter numbering starts at 1")
		})
	}
}

func TestConformance_QuotedIdentifier(t *testing.T) {
	t.Parallel()
	for _, fixture := range conformanceFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			emitter, _, generateError := runGenerate(
				context.Background(),
				fixture.newEngine(),
				map[string]string{"001_quoted.up.sql": fixture.quotedMigration},
				nil,
			)
			require.NoError(t, generateError, "generation should succeed")
			names := catalogueTableNames(emitter.catalogue)
			assert.Contains(t, names, strings.ToLower(quotedWeirdName),
				"the delimited table name should appear in the catalogue")
		})
	}
}

func TestConformance_CommentStripping(t *testing.T) {
	t.Parallel()
	for _, fixture := range conformanceFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			migration := "/* leading block comment */\n" +
				"-- a line comment\n" +
				fixture.baseMigration + "\n" +
				"-- trailing comment\n"
			emitter, _, generateError := runGenerate(
				context.Background(),
				fixture.newEngine(),
				map[string]string{"001_items.up.sql": migration},
				nil,
			)
			require.NoError(t, generateError, "generation should succeed despite surrounding comments")
			names := catalogueTableNames(emitter.catalogue)
			assert.Contains(t, names, "items", "the table defined amid comments should appear in the catalogue")
		})
	}
}

func TestConformance_UnterminatedBlockCommentErrors(t *testing.T) {
	t.Parallel()
	const brokenQuery = "-- piko.query(name: Broken, command: one)\n" +
		"SELECT id FROM items /* unterminated"
	for _, fixture := range conformanceFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			_, result, generateError := runGenerate(
				context.Background(),
				fixture.newEngine(),
				map[string]string{"001_items.up.sql": fixture.baseMigration},
				map[string]string{"broken.sql": brokenQuery},
			)
			require.NoError(t, generateError, "generation itself should not fail fatally")
			require.NotNil(t, result, "a generation result is expected")
			require.NotEmpty(t, result.Diagnostics,
				"an unterminated block comment must produce a diagnostic in every dialect")
			found := false
			for _, diagnostic := range result.Diagnostics {
				if strings.Contains(diagnostic.Message, "unterminated block comment") {
					found = true
					break
				}
			}
			assert.True(t, found,
				"the diagnostic should name the unterminated block comment, got %+v", result.Diagnostics)
		})
	}
}

func TestConformance_DeepNestingTerminates(t *testing.T) {
	t.Parallel()
	for _, fixture := range conformanceFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			var emitter *recordingEmitter
			var result *querier_dto.GenerationResult
			var generateError error
			require.NotPanics(t, func() {
				emitter, result, generateError = runGenerate(
					context.Background(),
					fixture.newEngine(),
					map[string]string{"001_items.up.sql": fixture.baseMigration},
					map[string]string{"deep.sql": fixture.deepQuery},
				)
			}, "deeply nested input must be bounded by a guard, not overflow the stack")
			require.NoError(t, generateError,
				"a bounded depth guard must terminate generation cleanly, not fail fatally")
			require.NotNil(t, result, "a generation result is expected even for guard-bounded input")

			deepQuery := findQuery(emitter.queries, "Deep")
			if deepQuery == nil {

				return
			}
			require.LessOrEqual(t, len(deepQuery.OutputColumns), 1,
				"a guard-bounded nested expression must yield at most one output column")
			for _, column := range deepQuery.OutputColumns {
				assert.Equal(t, querier_dto.TypeCategoryUnknown, column.SQLType.Category,
					"the depth guard truncates the nested expression, so its column must degrade "+
						"to the Unknown category rather than resolving to a concrete type")
			}
		})
	}
}
