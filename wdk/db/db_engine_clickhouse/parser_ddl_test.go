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

	"piko.sh/piko/internal/querier/querier_dto"
)

func applyDDL(t *testing.T, sql string) (*querier_dto.CatalogueMutation, error) {
	t.Helper()
	engine := NewClickHouseEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	return engine.ApplyDDL(t.Context(), statements[0])
}

func TestDDL_CreateTableSimpleMergeTree(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE users (
			id UInt64,
			email String,
			created_at DateTime
		) ENGINE = MergeTree() ORDER BY id
	`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Equal(t, "users", mutation.TableName)
	require.Len(t, mutation.Columns, 3)
	assert.Equal(t, "id", mutation.Columns[0].Name)
	assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.Columns[0].SQLType.Category)
	assert.False(t, mutation.Columns[0].Nullable)
	assert.Equal(t, "email", mutation.Columns[1].Name)
	assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[1].SQLType.Category)

	assert.Equal(t, []string{"id"}, mutation.PrimaryKey)
	assert.Contains(t, mutation.EngineSpecific, "ENGINE")
	assert.Contains(t, mutation.EngineSpecific, "ORDER_BY")
}

func TestDDL_CreateTableWithNullableColumns(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE accounts (
			id UInt64,
			label Nullable(String),
			balance Decimal(18, 4)
		) ENGINE = MergeTree() ORDER BY id
	`)
	require.NoError(t, err)
	require.Len(t, mutation.Columns, 3)
	assert.False(t, mutation.Columns[0].Nullable, "id not nullable")
	assert.True(t, mutation.Columns[1].Nullable, "Nullable(String) sets the flag")
	assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[1].SQLType.Category)
	assert.Equal(t, querier_dto.TypeCategoryDecimal, mutation.Columns[2].SQLType.Category)
}

func TestDDL_CreateTableWithArrayColumn(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE events (
			id UUID,
			tags Array(String)
		) ENGINE = MergeTree() ORDER BY id
	`)
	require.NoError(t, err)
	require.Len(t, mutation.Columns, 2)
	tagsColumn := mutation.Columns[1]
	assert.Equal(t, querier_dto.TypeCategoryArray, tagsColumn.SQLType.Category)
	require.NotNil(t, tagsColumn.SQLType.ElementType)
	assert.Equal(t, querier_dto.TypeCategoryText, tagsColumn.SQLType.ElementType.Category)
}

func TestDDL_CreateTableWithSchemaQualification(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE analytics.events (
			id UInt64,
			ts DateTime
		) ENGINE = MergeTree() ORDER BY (ts, id)
	`)
	require.NoError(t, err)
	assert.Equal(t, "analytics", mutation.SchemaName)
	assert.Equal(t, "events", mutation.TableName)
	assert.Equal(t, []string{"ts", "id"}, mutation.PrimaryKey)
}

func TestDDL_CreateTableWithPartitionByAndTTL(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE logs (
			id UInt64,
			ts DateTime
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(ts)
		ORDER BY id
		TTL ts + INTERVAL 30 DAY
	`)
	require.NoError(t, err)
	assert.Contains(t, mutation.EngineSpecific, "PARTITION_BY")
	assert.Contains(t, mutation.EngineSpecific, "TTL")
	assert.Contains(t, mutation.EngineSpecific, "ORDER_BY")
}

func TestDDL_CreateTableExplicitPrimaryKey(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE products (
			id UInt64,
			sku String,
			PRIMARY KEY (id)
		) ENGINE = MergeTree() ORDER BY id
	`)
	require.NoError(t, err)
	assert.Equal(t, []string{"id"}, mutation.PrimaryKey)
}

func TestDDL_CreateTableIfNotExists(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE IF NOT EXISTS items (
			id UInt64
		) ENGINE = Log
	`)
	require.NoError(t, err)
	assert.Equal(t, "items", mutation.TableName)
}

func TestDDL_CreateTableWithDefault(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE settings (
			id UInt64,
			created_at DateTime DEFAULT now(),
			label String DEFAULT 'unset'
		) ENGINE = MergeTree() ORDER BY id
	`)
	require.NoError(t, err)
	require.Len(t, mutation.Columns, 3)
	assert.False(t, mutation.Columns[0].HasDefault)
	assert.True(t, mutation.Columns[1].HasDefault)
	assert.True(t, mutation.Columns[2].HasDefault)
}

func TestDDL_DropTable(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "DROP TABLE IF EXISTS users")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
	assert.Equal(t, "users", mutation.TableName)
}

func TestDDL_AlterAddColumn(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE users ADD COLUMN email Nullable(String)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
	require.Len(t, mutation.Columns, 1)
	assert.Equal(t, "email", mutation.Columns[0].Name)
	assert.True(t, mutation.Columns[0].Nullable)
}

func TestDDL_AlterDropColumn(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE users DROP COLUMN email")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationAlterTableDropColumn, mutation.Kind)
	assert.Equal(t, "email", mutation.ColumnName)
}

func TestDDL_AlterModifyColumn(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE users MODIFY COLUMN id UInt64")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
	assert.Equal(t, "id", mutation.ColumnName)
}

func TestDDL_AlterRenameColumn(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE users RENAME COLUMN old_name TO new_name")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationAlterTableRenameColumn, mutation.Kind)
	assert.Equal(t, "old_name", mutation.ColumnName)
	assert.Equal(t, "new_name", mutation.NewName)
}

func TestDDL_AlterUpdate(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE users UPDATE name = 'unknown' WHERE id = 1")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAsyncDataUpdate, mutation.Kind)
	assert.Equal(t, "users", mutation.TableName)
	asyncBody := mutation.EngineSpecific["ASYNC_BODY"]
	assert.Contains(t, asyncBody, "name")
	assert.Contains(t, asyncBody, "WHERE")
}

func TestDDL_AlterDelete(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE users DELETE WHERE id = 1")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAsyncDataDelete, mutation.Kind)
	assert.Equal(t, "users", mutation.TableName)
	assert.Contains(t, mutation.EngineSpecific["ASYNC_BODY"], "WHERE")
}

func TestDDL_AlterTableOnCluster(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE users ON CLUSTER my_cluster ADD COLUMN age UInt8")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "my_cluster", mutation.EngineSpecific["ON_CLUSTER"])
}

func TestDDL_CreateTableOnCluster(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE TABLE replicated_users ON CLUSTER prod (id UInt64) ENGINE = MergeTree ORDER BY id")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "prod", mutation.EngineSpecific["ON_CLUSTER"])
}

func TestDDL_DropTableOnClusterWithSync(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "DROP TABLE IF EXISTS x ON CLUSTER c SYNC")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "c", mutation.EngineSpecific["ON_CLUSTER"])
}

func TestDDL_AlterTablePartitionOps(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		"ALTER TABLE events FREEZE PARTITION '2026-01'",
		"ALTER TABLE events UNFREEZE PARTITION '2026-01' WITH NAME 'backup'",
		"ALTER TABLE events DETACH PARTITION '2026-01'",
		"ALTER TABLE events ATTACH PARTITION '2026-01'",
		"ALTER TABLE events DROP PARTITION '2026-01'",
		"ALTER TABLE events MOVE PARTITION '2026-01' TO DISK 'cold'",
		"ALTER TABLE events FETCH PARTITION '2026-01' FROM '/clickhouse/tables/events'",
	} {
		_, err := applyDDL(t, source)
		assert.NoError(t, err, "failed to parse: %s", source)
	}
}

func TestDDL_AlterMaterializeVariants(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		"ALTER TABLE t MATERIALIZE COLUMN c",
		"ALTER TABLE t MATERIALIZE INDEX i",
		"ALTER TABLE t MATERIALIZE PROJECTION p",
		"ALTER TABLE t MATERIALIZE TTL",
		"ALTER TABLE t MATERIALIZE STATISTICS s",
	} {
		_, err := applyDDL(t, source)
		assert.NoError(t, err, "failed to parse: %s", source)
	}
}

func TestDDL_AlterTableMultiActionIndexAndColumn(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE t ADD INDEX i col TYPE bloom_filter GRANULARITY 1, ADD COLUMN new_col UInt32")
	require.NoError(t, err)
	require.NotNil(t, mutation)

	allMutations := append([]*querier_dto.CatalogueMutation{mutation}, mutation.AdditionalMutations...)
	foundColumn := false
	for _, m := range allMutations {
		if m == nil {
			continue
		}
		if m.Kind == querier_dto.MutationAlterTableAddColumn {
			foundColumn = true
			break
		}
	}
	assert.True(t, foundColumn, "ADD COLUMN action lost after ADD INDEX")
}

func TestDDL_CreateView(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE VIEW v AS SELECT id FROM t")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
	assert.Equal(t, "v", mutation.TableName)
}

func TestDDL_CreateMaterializedView(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE MATERIALIZED VIEW mv
		ENGINE = MergeTree() ORDER BY id
		AS SELECT id FROM t
	`)
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
	assert.Equal(t, "true", mutation.EngineSpecific["MATERIALIZED"])
}

func TestDDL_CreateDatabase(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE DATABASE analytics")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationCreateSchema, mutation.Kind)
	assert.Equal(t, "analytics", mutation.SchemaName)
}

func TestDDL_DropDatabase(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "DROP DATABASE IF EXISTS analytics")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationDropSchema, mutation.Kind)
	assert.Equal(t, "analytics", mutation.SchemaName)
}

func TestDDL_RenameTable(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "RENAME TABLE old TO new")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.MutationAlterTableRenameTable, mutation.Kind)
	assert.Equal(t, "old", mutation.TableName)
	assert.Equal(t, "new", mutation.NewName)
}

func TestDDL_TruncateOptimizeSystemNoOps(t *testing.T) {
	t.Parallel()

	for _, sql := range []string{
		"TRUNCATE TABLE t",
		"OPTIMIZE TABLE t FINAL",
		"SYSTEM FLUSH LOGS",
		"USE analytics",
		"SHOW TABLES",
		"SET max_threads = 4",
	} {
		t.Run(sql, func(t *testing.T) {
			t.Parallel()
			mutation, err := applyDDL(t, sql)
			require.NoError(t, err)
			assert.Nil(t, mutation, "expected catalogue no-op")
		})
	}
}

func TestDDL_CreateTableColumnNotNullSuffix(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE accounts (
			id UInt64 NOT NULL,
			label String NULL,
			email String
		) ENGINE = MergeTree() ORDER BY id
	`)
	require.NoError(t, err)
	require.Len(t, mutation.Columns, 3)
	assert.False(t, mutation.Columns[0].Nullable, "Int64 NOT NULL stays non-nullable")
	assert.True(t, mutation.Columns[1].Nullable, "String NULL promotes to nullable")
	assert.False(t, mutation.Columns[2].Nullable, "bare String remains non-nullable")
}

func TestDDL_CreateTableAsSelect(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE TABLE summary AS SELECT id, name FROM users")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "summary", mutation.TableName)
	require.Len(t, mutation.Columns, 2, "CTAS body should populate columns from SELECT")
	assert.Equal(t, "id", mutation.Columns[0].Name)
	assert.Equal(t, "name", mutation.Columns[1].Name)
}

func TestDDL_CreateTableAsSourceTable(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE TABLE clone AS analytics.events ENGINE = MergeTree() ORDER BY id")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "events", mutation.EngineSpecific["CTAS_SOURCE_TABLE"])
	assert.Equal(t, "analytics", mutation.EngineSpecific["CTAS_SOURCE_SCHEMA"])
}

func TestDDL_CreateTableReplacingMergeTreeCapturesVersion(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE events (
			id UInt64,
			ts DateTime,
			version UInt64,
			is_deleted UInt8
		) ENGINE = ReplacingMergeTree(version, is_deleted) ORDER BY id
	`)
	require.NoError(t, err)
	assert.Equal(t, "version", mutation.EngineSpecific["MERGETREE_VERSION_COLUMN"])
	assert.Equal(t, "is_deleted", mutation.EngineSpecific["MERGETREE_IS_DELETED_COLUMN"])
}

func TestDDL_CreateTableCollapsingMergeTreeCapturesSign(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE transactions (
			id UInt64,
			sign Int8
		) ENGINE = CollapsingMergeTree(sign) ORDER BY id
	`)
	require.NoError(t, err)
	assert.Equal(t, "sign", mutation.EngineSpecific["MERGETREE_SIGN_COLUMN"])
}

func TestDDL_CreateTableVersionedCollapsingMergeTree(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE transactions (
			id UInt64,
			sign Int8,
			ver UInt32
		) ENGINE = VersionedCollapsingMergeTree(sign, ver) ORDER BY id
	`)
	require.NoError(t, err)
	assert.Equal(t, "sign", mutation.EngineSpecific["MERGETREE_SIGN_COLUMN"])
	assert.Equal(t, "ver", mutation.EngineSpecific["MERGETREE_VERSION_COLUMN"])
}

func TestDDL_CreateTableSummingMergeTree(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE totals (
			id UInt64,
			revenue UInt64,
			impressions UInt64
		) ENGINE = SummingMergeTree((revenue, impressions)) ORDER BY id
	`)
	require.NoError(t, err)
	assert.Contains(t, mutation.EngineSpecific["MERGETREE_SUMMING_COLUMNS"], "revenue")
}

func TestDDL_CreateTableReplicatedMergeTree(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE TABLE shards (
			id UInt64
		) ENGINE = ReplicatedMergeTree('/clickhouse/tables/shard1', 'replica_a') ORDER BY id
	`)
	require.NoError(t, err)
	assert.Equal(t, "'/clickhouse/tables/shard1'", mutation.EngineSpecific["MERGETREE_ZOO_PATH"])
	assert.Equal(t, "'replica_a'", mutation.EngineSpecific["MERGETREE_REPLICA_NAME"])
}

func TestDDL_CreateDictionaryStructured(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE DICTIONARY country_codes (
			code UInt16,
			name String
		)
		PRIMARY KEY code
		SOURCE(HTTP(url 'http://example.com/codes'))
		LAYOUT(HASHED())
		LIFETIME(MIN 60 MAX 300)
	`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateDictionary, mutation.Kind)
	assert.Equal(t, "country_codes", mutation.TableName)
	require.Len(t, mutation.Columns, 2)
	assert.Equal(t, "code", mutation.Columns[0].Name)
	assert.Equal(t, []string{"code"}, mutation.PrimaryKey)
	assert.Equal(t, "code", mutation.EngineSpecific["DICTIONARY_PRIMARY_KEY"])
	assert.Contains(t, mutation.EngineSpecific["DICTIONARY_SOURCE"], "HTTP")
	assert.Contains(t, mutation.EngineSpecific["DICTIONARY_LAYOUT"], "HASHED")
	assert.Equal(t, "MIN 60 MAX 300", mutation.EngineSpecific["DICTIONARY_LIFETIME_SECONDS"])
}

func TestDDL_CreateDictionarySimpleLifetime(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE DICTIONARY codes (id UInt64, name String)
		PRIMARY KEY id
		SOURCE(FILE(path '/var/data.csv'))
		LAYOUT(FLAT())
		LIFETIME(300)
	`)
	require.NoError(t, err)
	assert.Equal(t, "300", mutation.EngineSpecific["DICTIONARY_LIFETIME_SECONDS"])
}

func TestDDL_DropDictionary(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "DROP DICTIONARY IF EXISTS country_codes")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationDropDictionary, mutation.Kind)
	assert.Equal(t, "country_codes", mutation.TableName)
}

func TestDDL_ExchangeTables(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "EXCHANGE TABLES analytics.events_new AND analytics.events_old")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationExchangeTables, mutation.Kind)
	assert.Equal(t, "analytics", mutation.SchemaName)
	assert.Equal(t, "events_new", mutation.TableName)
	assert.Equal(t, "analytics.events_old", mutation.EngineSpecific["EXCHANGE_TARGET"])
}

func TestDDL_ExchangeTablesOnCluster(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "EXCHANGE TABLES a AND b ON CLUSTER c")
	require.NoError(t, err)
	assert.Equal(t, "a", mutation.TableName)
	assert.Equal(t, "b", mutation.EngineSpecific["EXCHANGE_TARGET"])
	assert.Equal(t, "c", mutation.EngineSpecific["ON_CLUSTER"])
}

func TestCreateFunction_LambdaBody_Literal(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE FUNCTION fortytwo AS () -> 42")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	require.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
	require.NotNil(t, mutation.FunctionSignature)
	assert.Equal(t, "fortytwo", mutation.FunctionSignature.Name)
	assert.NotNil(t, mutation.FunctionSignature.BodyExpression)
	assert.Empty(t, mutation.FunctionSignature.BodyParameters)

	assert.Equal(t, querier_dto.TypeCategoryUnknown, mutation.FunctionSignature.ReturnType.Category)
}

func TestCreateFunction_LambdaBody_SingleParam(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE FUNCTION twice AS x -> x * 2")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	require.NotNil(t, mutation.FunctionSignature)
	assert.Equal(t, "twice", mutation.FunctionSignature.Name)
	assert.NotNil(t, mutation.FunctionSignature.BodyExpression)
	assert.Equal(t, []string{"x"}, mutation.FunctionSignature.BodyParameters)
}

func TestCreateFunction_LambdaBody_MultiParam(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE FUNCTION addxy AS (x, y) -> x + y")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	require.NotNil(t, mutation.FunctionSignature)
	assert.Equal(t, "addxy", mutation.FunctionSignature.Name)
	assert.NotNil(t, mutation.FunctionSignature.BodyExpression)
	assert.Equal(t, []string{"x", "y"}, mutation.FunctionSignature.BodyParameters)
}

func TestCreateFunction_LegacyNoAS_Backwards(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE FUNCTION legacy")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	require.NotNil(t, mutation.FunctionSignature)
	assert.Equal(t, "legacy", mutation.FunctionSignature.Name)
	assert.Nil(t, mutation.FunctionSignature.BodyExpression)
}

func TestCreateFunction_FunctionCallInBody(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE FUNCTION lengthof AS arr -> arrayLength(arr)")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	require.NotNil(t, mutation.FunctionSignature)
	require.NotNil(t, mutation.FunctionSignature.BodyExpression)
	assert.Equal(t, []string{"arr"}, mutation.FunctionSignature.BodyParameters)
	body, ok := mutation.FunctionSignature.BodyExpression.(*querier_dto.FunctionCallExpression)
	require.True(t, ok, "body should be a function call expression")
	assert.Equal(t, "arrayLength", body.FunctionName)
	require.Len(t, body.Arguments, 1)
	columnRef, ok := body.Arguments[0].(*querier_dto.ColumnRefExpression)
	require.True(t, ok, "argument should be a column reference")
	assert.Equal(t, "arr", columnRef.ColumnName)
}

func TestInferPrimaryKeyFromOrderBy_ParenAwareSplit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		orderBy  string
		expected []string
	}{
		{
			name:     "single column",
			orderBy:  "id",
			expected: []string{"id"},
		},
		{
			name:     "plain column list",
			orderBy:  "(a, b, c)",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "multi-arg function call kept intact",
			orderBy:  "(cityHash64(a, b), c)",
			expected: []string{"cityHash64(a,", "c"},
		},
		{
			name:     "direction qualifier dropped",
			orderBy:  "(a DESC, b ASC)",
			expected: []string{"a", "b"},
		},
		{
			name:     "nested function calls",
			orderBy:  "(toStartOfHour(ts), cityHash64(a, b))",
			expected: []string{"toStartOfHour(ts)", "cityHash64(a,"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(subtest *testing.T) {
			subtest.Parallel()
			assert.Equal(subtest, testCase.expected, inferPrimaryKeyFromOrderBy(testCase.orderBy))
		})
	}
}

func TestModifyColumn_MalformedSubActionErrors(t *testing.T) {
	t.Parallel()

	_, err := applyDDL(t, `ALTER TABLE t MODIFY COLUMN c RESET notsetting`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SETTING")
}

func TestDictionaryLifetime_MinWithoutMax(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE DICTIONARY d (id UInt64, name String)
		PRIMARY KEY id
		SOURCE(HTTP(url 'http://example.com/codes'))
		LAYOUT(HASHED())
		LIFETIME(MIN 300)
	`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	lifetime := mutation.EngineSpecific["DICTIONARY_LIFETIME_SECONDS"]
	assert.Equal(t, "MIN 300", lifetime)
	assert.NotContains(t, lifetime, "MAX ")
}
