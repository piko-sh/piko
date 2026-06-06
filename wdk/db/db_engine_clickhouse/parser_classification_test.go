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

package db_engine_clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func classify(t *testing.T, sql string) statementKind {
	t.Helper()
	tokens, err := tokenise(sql)
	require.NoError(t, err)
	statements := splitStatements(tokens)
	require.Len(t, statements, 1, "expected single statement")
	return classifyStatement(statements[0])
}

func TestClassify_Select(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindSelect, classify(t, "SELECT 1"))
	assert.Equal(t, statementKindSelect, classify(t, "select id from users"))
	assert.Equal(t, statementKindSelect, classify(t, "SELECT id, name FROM users WHERE id = {uid:UInt32}"))
}

func TestClassify_Insert(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindInsert, classify(t, "INSERT INTO users VALUES (1, 'a')"))
	assert.Equal(t, statementKindInsert, classify(t, "INSERT INTO users SELECT * FROM other"))
}

func TestClassify_CreateTable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindCreateTable, classify(t, "CREATE TABLE users (id UInt64) ENGINE = MergeTree() ORDER BY id"))
	assert.Equal(t, statementKindCreateTable, classify(t, "CREATE TABLE IF NOT EXISTS t (id UInt64) ENGINE = Log"))
	assert.Equal(t, statementKindCreateTable, classify(t, "CREATE OR REPLACE TABLE t (id UInt64) ENGINE = Log"))
	assert.Equal(t, statementKindCreateTable, classify(t, "CREATE TEMPORARY TABLE t (id UInt64) ENGINE = Memory"))
}

func TestClassify_CreateView(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindCreateView, classify(t, "CREATE VIEW v AS SELECT id FROM t"))
	assert.Equal(t, statementKindCreateView, classify(t, "CREATE OR REPLACE VIEW v AS SELECT id FROM t"))
}

func TestClassify_CreateMaterializedView(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindCreateMaterializedView, classify(t,
		"CREATE MATERIALIZED VIEW mv ENGINE = MergeTree() ORDER BY id AS SELECT id FROM t"))
}

func TestClassify_CreateDictionary(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindCreateDictionary, classify(t,
		"CREATE DICTIONARY dict (id UInt64, name String) PRIMARY KEY id"))
}

func TestClassify_CreateFunction(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindCreateFunction, classify(t,
		"CREATE FUNCTION double AS x -> x * 2"))
}

func TestClassify_CreateDatabase(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindCreateDatabase, classify(t, "CREATE DATABASE analytics"))
	assert.Equal(t, statementKindCreateDatabase, classify(t, "CREATE DATABASE IF NOT EXISTS analytics"))
}

func TestClassify_DropVariants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindDropTable, classify(t, "DROP TABLE t"))
	assert.Equal(t, statementKindDropView, classify(t, "DROP VIEW v"))
	assert.Equal(t, statementKindDropDictionary, classify(t, "DROP DICTIONARY d"))
	assert.Equal(t, statementKindDropFunction, classify(t, "DROP FUNCTION f"))
	assert.Equal(t, statementKindDropDatabase, classify(t, "DROP DATABASE analytics"))
}

func TestClassify_AlterTable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindAlterTable, classify(t, "ALTER TABLE t ADD COLUMN name String"))
	assert.Equal(t, statementKindAlterTable, classify(t, "ALTER TABLE t DROP COLUMN name"))
	assert.Equal(t, statementKindAlterTable, classify(t, "ALTER TABLE t MODIFY COLUMN id UInt64"))
	assert.Equal(t, statementKindAlterTable, classify(t, "ALTER TABLE t UPDATE count = count + 1 WHERE id = 1"))
	assert.Equal(t, statementKindAlterTable, classify(t, "ALTER TABLE t DELETE WHERE id = 1"))
}

func TestClassify_RenameTable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindRenameTable, classify(t, "RENAME TABLE old TO new"))
	assert.Equal(t, statementKindRenameTable, classify(t, "RENAME DICTIONARY d1 TO d2"))
	assert.Equal(t, statementKindRenameTable, classify(t, "RENAME DATABASE old TO new"))
}

func TestClassify_ExchangeTables(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindExchangeTables, classify(t, "EXCHANGE TABLES a AND b"))
	assert.Equal(t, statementKindExchangeTables, classify(t, "EXCHANGE TABLES analytics.events_new AND analytics.events_old"))
}

func TestClassify_TruncateOptimize(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindTruncate, classify(t, "TRUNCATE TABLE t"))
	assert.Equal(t, statementKindOptimize, classify(t, "OPTIMIZE TABLE t FINAL"))
}

func TestClassify_System(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindSystem, classify(t, "SYSTEM FLUSH LOGS"))
	assert.Equal(t, statementKindSystem, classify(t, "SYSTEM RELOAD DICTIONARY d"))
}

func TestClassify_UseShowSet(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindUse, classify(t, "USE analytics"))
	assert.Equal(t, statementKindShow, classify(t, "SHOW TABLES"))
	assert.Equal(t, statementKindSet, classify(t, "SET max_threads = 4"))
}

func TestClassify_WithCTE(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindSelect, classify(t, "WITH cte AS (SELECT id FROM t) SELECT * FROM cte"))
	assert.Equal(t, statementKindInsert, classify(t,
		"WITH cte AS (SELECT id FROM t) INSERT INTO dest SELECT * FROM cte"))
}

func TestClassify_Unknown(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindUnknown, classify(t, "UNKNOWN_STATEMENT"))
}

func TestSplitStatements_MultipleStatements(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("CREATE TABLE a (id UInt64) ENGINE = Log; CREATE TABLE b (id UInt64) ENGINE = Log;")
	require.NoError(t, err)
	statements := splitStatements(tokens)
	require.Len(t, statements, 2)
	assert.Equal(t, statementKindCreateTable, classifyStatement(statements[0]))
	assert.Equal(t, statementKindCreateTable, classifyStatement(statements[1]))
}

func TestSplitStatements_TrailingSemicolonDoesNotProduceEmptyStatement(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("SELECT 1;")
	require.NoError(t, err)
	statements := splitStatements(tokens)
	assert.Len(t, statements, 1)
}

func TestSplitStatements_EmptyInput(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("")
	require.NoError(t, err)
	assert.Empty(t, splitStatements(tokens))
}

func TestParser_ParseDatabaseQualifiedName(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("analytics.events")
	require.NoError(t, err)
	p := newParser(tokens)
	database, name, err := p.parseDatabaseQualifiedName()
	require.NoError(t, err)
	assert.Equal(t, "analytics", database)
	assert.Equal(t, "events", name)
}

func TestParser_ParseDatabaseQualifiedNameSingleIdent(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("events")
	require.NoError(t, err)
	p := newParser(tokens)
	database, name, err := p.parseDatabaseQualifiedName()
	require.NoError(t, err)
	assert.Equal(t, "", database)
	assert.Equal(t, "events", name)
}

func TestParser_MatchKeywordCaseInsensitive(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("Select id")
	require.NoError(t, err)
	p := newParser(tokens)
	assert.True(t, p.matchKeyword("SELECT"))
	assert.False(t, p.matchKeyword("FROM"))
}

func TestParser_SkipParenthesisedBalanced(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("(a, b, (c, d), e)")
	require.NoError(t, err)
	p := newParser(tokens)
	require.NoError(t, p.skipParenthesised())
	assert.True(t, p.atEnd())
}

func TestParser_CollectParenthesised(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("(a, b, c)")
	require.NoError(t, err)
	p := newParser(tokens)
	inner, err := p.collectParenthesised()
	require.NoError(t, err)

	assert.Len(t, inner, 5)
}

func TestParseStatements_EndToEnd(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	statements, err := engine.ParseStatements("CREATE TABLE t (id UInt64) ENGINE = MergeTree() ORDER BY id; SELECT * FROM t;")
	require.NoError(t, err)
	require.Len(t, statements, 2)

	first, firstOk := statements[0].Raw.(*parsedStatement)
	require.True(t, firstOk)
	assert.Equal(t, statementKindCreateTable, first.kind)

	second, secondOk := statements[1].Raw.(*parsedStatement)
	require.True(t, secondOk)
	assert.Equal(t, statementKindSelect, second.kind)
}
