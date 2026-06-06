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

func analyse(t *testing.T, sql string) *querier_dto.RawQueryAnalysis {
	t.Helper()
	engine := NewClickHouseEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	analysis, err := engine.AnalyseQuery(nil, statements[0])
	require.NoError(t, err)
	require.NotNil(t, analysis)
	return analysis
}

func TestAnalyseQuery_AlterAsyncMutationSetsAsyncBodyMarker(t *testing.T) {
	t.Parallel()

	deleteAnalysis := analyse(t, "ALTER TABLE events DELETE WHERE ts < {cutoff:DateTime}")
	require.NotNil(t, deleteAnalysis.EngineSpecific)
	_, deletePresent := deleteAnalysis.EngineSpecific[engineKeyAsyncBody]
	assert.True(t, deletePresent, "ALTER ... DELETE must carry the ASYNC_BODY marker")

	updateAnalysis := analyse(t, "ALTER TABLE events UPDATE payload = {p:String} WHERE id = {id:UInt64}")
	_, updatePresent := updateAnalysis.EngineSpecific[engineKeyAsyncBody]
	assert.True(t, updatePresent, "ALTER ... UPDATE must carry the ASYNC_BODY marker")

	assert.Len(t, updateAnalysis.ParameterReferences, 2)

	addColumn := analyse(t, "ALTER TABLE events ON CLUSTER c ADD COLUMN flag UInt8")
	_, addPresent := addColumn.EngineSpecific[engineKeyAsyncBody]
	assert.False(t, addPresent, "ALTER ... ADD COLUMN must not be marked async")
}

func TestDML_SelectBasic(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id, name FROM users")
	require.Len(t, analysis.OutputColumns, 2)
	assert.Equal(t, "id", analysis.OutputColumns[0].ColumnName)
	assert.Equal(t, "name", analysis.OutputColumns[1].ColumnName)
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
}

func TestDML_SelectStarExpansion(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT * FROM events")
	require.Len(t, analysis.OutputColumns, 1)
	assert.True(t, analysis.OutputColumns[0].IsStar)
}

func TestDML_SelectQualifiedStar(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT users.* FROM users")
	require.Len(t, analysis.OutputColumns, 1)
	assert.True(t, analysis.OutputColumns[0].IsStar)
	assert.Equal(t, "users", analysis.OutputColumns[0].TableAlias)
}

func TestDML_SelectWithSchemaQualifiedTable(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM analytics.events")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "analytics", analysis.FromTables[0].Schema)
	assert.Equal(t, "events", analysis.FromTables[0].Name)
}

func TestDML_SelectWithAlias(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT e.id FROM events AS e")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "events", analysis.FromTables[0].Name)
	assert.Equal(t, "e", analysis.FromTables[0].Alias)
}

func TestDML_SelectInnerJoin(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u INNER JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.FromTables, 2)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
	assert.Equal(t, "accounts", analysis.FromTables[1].Name)
}

func TestDML_SelectLeftJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u LEFT JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinLeft, analysis.JoinClauses[0].Kind)
	assert.Equal(t, "accounts", analysis.JoinClauses[0].Table.Name)
}

func TestDML_SelectAsofJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT q.symbol FROM quotes q ASOF JOIN trades t ON q.symbol = t.symbol AND q.ts <= t.ts")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinAsof, analysis.JoinClauses[0].Kind)
}

func TestDML_SelectSemiJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u SEMI JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinSemi, analysis.JoinClauses[0].Kind)
}

func TestDML_SelectAntiJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u ANTI JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinAnti, analysis.JoinClauses[0].Kind)
}

func TestDML_SelectLeftSemiJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u LEFT SEMI JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinLeftSemi, analysis.JoinClauses[0].Kind)
}

func TestDML_SelectLeftAntiJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u LEFT ANTI JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinLeftAnti, analysis.JoinClauses[0].Kind)
}

func TestDML_SelectRightSemiJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u RIGHT SEMI JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinRightSemi, analysis.JoinClauses[0].Kind)
}

func TestDML_SelectRightAntiJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u RIGHT ANTI JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinRightAnti, analysis.JoinClauses[0].Kind)
}

func TestDML_SelectInfixProjectionCorrupt(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT a + b AS c, d FROM t")
	require.Len(t, analysis.OutputColumns, 2)
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "t", analysis.FromTables[0].Name)
}

func TestDML_SelectWindowClauseAfterFrom(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id, row_number() OVER w FROM users WINDOW w AS (ORDER BY id)")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
}

func TestDML_SelectInSubqueryDoesNotMisparse(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users WHERE id IN (SELECT user_id FROM admins)")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
}

func TestDML_SelectInWrappedSubqueryDoesNotMisparse(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users WHERE id IN ((SELECT user_id FROM admins))")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
}

func TestDML_SelectPrewhereSetsHasWhereClause(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users PREWHERE active")
	assert.True(t, analysis.HasWhereClause, "PREWHERE-only queries should still set HasWhereClause for count rewriting")
}

func TestDML_SelectRecursiveCTEFlag(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "WITH RECURSIVE r AS (SELECT 1 AS n) SELECT * FROM r")
	require.Len(t, analysis.CTEDefinitions, 1)
	assert.True(t, analysis.CTEDefinitions[0].IsRecursive)
}

func TestDML_SelectLimitCommaForm(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users LIMIT 10, 5")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
}

func TestDML_SelectWhereWithCurlyParameter(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users WHERE id = {uid:UInt32}")
	require.Len(t, analysis.ParameterReferences, 1)
	ref := analysis.ParameterReferences[0]
	assert.Equal(t, "uid", ref.Name)
	assert.Equal(t, 1, ref.Number)
	require.NotNil(t, ref.CastType)
	assert.Equal(t, querier_dto.TypeCategoryInteger, ref.CastType.Category)
	assert.True(t, analysis.HasWhereClause)
}

func TestDML_UnrecognisedParameterTypeTagRecordsDiagnostic(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	statements, parseErr := engine.ParseStatements("SELECT id FROM users WHERE id = {uid:Strign}")
	require.NoError(t, parseErr)
	require.Len(t, statements, 1)

	analysis, analyseErr := engine.AnalyseQuery(nil, statements[0])
	require.Error(t, analyseErr, "an unrecognised parameter type tag must no longer be silently accepted")
	assert.Contains(t, analyseErr.Error(), "unrecognised type tag")

	require.NotNil(t, analysis)
	require.Len(t, analysis.ParameterReferences, 1)
	ref := analysis.ParameterReferences[0]
	assert.Equal(t, "uid", ref.Name)
	require.NotNil(t, ref.CastType, "the binding is retained with the Unknown cast, not dropped")
	assert.Equal(t, querier_dto.TypeCategoryUnknown, ref.CastType.Category)
}

func TestDML_FunctionArgumentParameterIsSelfTyped(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT toString({uid:UInt32}) FROM users")
	require.Len(t, analysis.ParameterReferences, 1)
	ref := analysis.ParameterReferences[0]
	assert.Equal(t, "uid", ref.Name)

	require.NotNil(t, ref.CastType, "function-arg placeholder must carry its explicit type tag")
	assert.Equal(t, querier_dto.TypeCategoryInteger, ref.CastType.Category)

	assert.Empty(t, ref.EnclosingFunctionName, "clickhouse does not record the enclosing function name")
	assert.Zero(t, ref.ArgumentOrdinal, "clickhouse does not record an argument ordinal")
}

func TestDML_FromTableFunctionArgumentIsNotCaptured(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT number FROM numbers({n:UInt64})")
	require.Len(t, analysis.RawTableValuedFunctions, 1)
	assert.Equal(t, "numbers", analysis.RawTableValuedFunctions[0].FunctionName)
	assert.Empty(t, analysis.ParameterReferences, "table-function arguments are consumed opaquely")
}

func TestDML_ParameterReuseDedupes(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users WHERE id = {uid:UInt32} OR parent_id = {uid:UInt32}")

	require.Len(t, analysis.ParameterReferences, 2)
	assert.Equal(t, analysis.ParameterReferences[0].Number, analysis.ParameterReferences[1].Number)
}

func TestDML_GroupByCapturesColumns(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT count(*), country FROM users GROUP BY country")
	require.Len(t, analysis.GroupByColumns, 1)
	assert.Equal(t, "country", analysis.GroupByColumns[0].ColumnName)
}

func TestDML_GroupByWithRollup(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT country, region, count(*) FROM users GROUP BY country, region WITH ROLLUP")
	require.Len(t, analysis.GroupByColumns, 2)
}

func TestDML_OrderByLimit(t *testing.T) {
	t.Parallel()

	analyse(t, "SELECT id FROM users ORDER BY created_at DESC LIMIT 10 OFFSET 20")
}

func TestDML_PrewhereAccepted(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users PREWHERE active = 1 WHERE id > 0")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
	assert.True(t, analysis.HasWhereClause, "PREWHERE + WHERE should set HasWhereClause")
}

func TestDML_FinalModifierAccepted(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users FINAL WHERE id = 1")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
	assert.True(t, analysis.HasWhereClause)
}

func TestDML_SampleAccepted(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users SAMPLE 0.1 WHERE id = 1")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "users", analysis.FromTables[0].Name)
	assert.True(t, analysis.HasWhereClause)
}

func TestDML_ArrayJoinAccepted(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT tag FROM events ARRAY JOIN tags AS tag WHERE id = 1")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "events", analysis.FromTables[0].Name)
	require.Len(t, analysis.OutputColumns, 1)
	assert.Equal(t, "tag", analysis.OutputColumns[0].Name)
}

func TestDML_LeftArrayJoinAccepted(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT tag FROM events LEFT ARRAY JOIN tags AS tag WHERE id = 1")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "events", analysis.FromTables[0].Name)
}

func TestDML_UnionAllRecordsCompoundBranches(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM a UNION ALL SELECT id FROM b")
	require.Len(t, analysis.CompoundBranches, 1)
	assert.Equal(t, querier_dto.CompoundUnionAll, analysis.CompoundBranches[0].Operator)
}

func TestDML_IntersectExcept(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM a INTERSECT SELECT id FROM b EXCEPT SELECT id FROM c")

	require.Len(t, analysis.CompoundBranches, 1)
	assert.Equal(t, querier_dto.CompoundIntersect, analysis.CompoundBranches[0].Operator)
	require.Len(t, analysis.CompoundBranches[0].Query.CompoundBranches, 1)
	assert.Equal(t, querier_dto.CompoundExcept, analysis.CompoundBranches[0].Query.CompoundBranches[0].Operator)
}

func TestDML_CTERecordsDefinitions(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "WITH active AS (SELECT id FROM users WHERE active = 1) SELECT id FROM active")
	require.Len(t, analysis.CTEDefinitions, 1)
	assert.Equal(t, "active", analysis.CTEDefinitions[0].Name)
}

func TestDML_DerivedTableSubquery(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT t.id FROM (SELECT id FROM users WHERE active = 1) t")
	require.Len(t, analysis.RawDerivedTables, 1)
	assert.Equal(t, "t", analysis.RawDerivedTables[0].Alias)
}

func TestDML_TableValuedFunctionCall(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT number FROM numbers(100)")
	require.Len(t, analysis.RawTableValuedFunctions, 1)
	assert.Equal(t, "numbers", analysis.RawTableValuedFunctions[0].FunctionName)
}

func TestDML_InsertValues(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "INSERT INTO users (id, name) VALUES (1, 'alice'), (2, 'bob')")
	assert.Equal(t, "users", analysis.InsertTable)
	assert.Equal(t, []string{"id", "name"}, analysis.InsertColumns)
	assert.False(t, analysis.ReadOnly)
}

func TestDML_InsertSelect(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "INSERT INTO dest (id, name) SELECT id, name FROM source WHERE active = 1")
	assert.Equal(t, "dest", analysis.InsertTable)
	assert.Equal(t, []string{"id", "name"}, analysis.InsertColumns)

	require.NotNil(t, analysis.InsertSelect)
	require.NotEmpty(t, analysis.InsertSelect.FromTables)
	assert.Equal(t, "source", analysis.InsertSelect.FromTables[0].Name)
}

func TestDML_InsertSelectWithJoinAndParams(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "INSERT INTO dest (id, name) "+
		"SELECT u.id, o.name FROM users u INNER JOIN orgs o ON o.id = u.org_id "+
		"WHERE u.id > {min_id:UInt32} AND o.active = {active:UInt8}")
	assert.Equal(t, "dest", analysis.InsertTable)

	require.NotNil(t, analysis.InsertSelect)
	require.Len(t, analysis.InsertSelect.FromTables, 2)
	assert.Equal(t, "users", analysis.InsertSelect.FromTables[0].Name)
	assert.Equal(t, "orgs", analysis.InsertSelect.FromTables[1].Name)
	require.Len(t, analysis.InsertSelect.ParameterReferences, 2)
	require.Len(t, analysis.ParameterReferences, 2)
}

func TestDML_InsertFormat(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "INSERT INTO logs (data) FORMAT JSONEachRow")
	assert.Equal(t, "logs", analysis.InsertTable)
}

func TestDML_SchemaQualifiedInsert(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "INSERT INTO analytics.events (id) VALUES (1)")
	assert.Equal(t, "events", analysis.InsertTable)
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "analytics", analysis.FromTables[0].Schema)
}

func TestDML_DeeplyNestedSubquery(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, `
		WITH agg AS (
			SELECT id, count() AS c FROM events GROUP BY id
		)
		SELECT a.id, a.c FROM agg a
		WHERE a.c > {threshold:UInt32}
		ORDER BY a.c DESC
		LIMIT {page_size:UInt32}
	`)
	require.Len(t, analysis.CTEDefinitions, 1)
	assert.Equal(t, "agg", analysis.CTEDefinitions[0].Name)
	require.GreaterOrEqual(t, len(analysis.ParameterReferences), 2, "should capture both threshold and page_size parameters")
	paramNames := map[string]bool{}
	for _, ref := range analysis.ParameterReferences {
		paramNames[ref.Name] = true
	}
	assert.True(t, paramNames["threshold"], "threshold parameter must be captured")
	assert.True(t, paramNames["page_size"], "page_size parameter must be captured")
}

func TestDML_GroupByAllRecordsModifier(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT region, count(*) FROM users GROUP BY ALL")
	require.NotNil(t, analysis.EngineSpecific)
	assert.Equal(t, "true", analysis.EngineSpecific["GROUP_BY_ALL"])
	assert.Empty(t, analysis.GroupByColumns, "GROUP BY ALL should not populate the column list")
}

func TestDML_GroupByGroupingSets(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT country, city, sum(val) FROM sales GROUP BY GROUPING SETS ((country, city), (country), ())")
	require.NotNil(t, analysis.EngineSpecific)
	captured, ok := analysis.EngineSpecific["GROUPING_SETS"]
	require.True(t, ok, "GROUPING SETS body should be captured")
	assert.Contains(t, captured, "country")
	assert.Contains(t, captured, "city")
}

func TestDML_QualifyClauseRecorded(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id, row_number() OVER (PARTITION BY user_id ORDER BY ts) AS rn FROM events QUALIFY rn = 1")
	require.NotNil(t, analysis.EngineSpecific)
	assert.Equal(t, "true", analysis.EngineSpecific["QUALIFY"])
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "events", analysis.FromTables[0].Name)
}

func TestDML_AnyJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u ANY LEFT JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)

	assert.Equal(t, querier_dto.JoinLeft, analysis.JoinClauses[0].Kind)
}

func TestDML_AllJoinRecordsKind(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u ALL INNER JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.FromTables, 2)
	assert.Equal(t, "accounts", analysis.FromTables[1].Name)
}

func TestDML_GlobalJoinPrefixAccepted(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u GLOBAL LEFT JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinLeft, analysis.JoinClauses[0].Kind)
}

func TestDML_PlainAnyJoinResolvesAny(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u ANY JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinAny, analysis.JoinClauses[0].Kind)
}

func TestDML_PlainGlobalJoinResolvesGlobal(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT u.id FROM users u GLOBAL JOIN accounts a ON a.user_id = u.id")
	require.Len(t, analysis.JoinClauses, 1)
	assert.Equal(t, querier_dto.JoinGlobal, analysis.JoinClauses[0].Kind)
}

func TestDML_CTEMaterializedQualifierCaptured(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "WITH active AS (SELECT id FROM users) MATERIALIZED SELECT * FROM active")
	require.Len(t, analysis.CTEDefinitions, 1)
	require.NotNil(t, analysis.CTEDefinitions[0].EngineSpecific)
	assert.Equal(t, "true", analysis.CTEDefinitions[0].EngineSpecific["CTE_MATERIALIZED"])
}

func TestDML_CTENotMaterializedQualifierCaptured(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "WITH active AS (SELECT id FROM users) NOT MATERIALIZED SELECT * FROM active")
	require.Len(t, analysis.CTEDefinitions, 1)
	require.NotNil(t, analysis.CTEDefinitions[0].EngineSpecific)
	assert.Equal(t, "false", analysis.CTEDefinitions[0].EngineSpecific["CTE_MATERIALIZED"])
}

func TestDML_CTEWithoutMaterializedHasNoEngineSpecific(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "WITH active AS (SELECT id FROM users) SELECT * FROM active")
	require.Len(t, analysis.CTEDefinitions, 1)
	assert.Nil(t, analysis.CTEDefinitions[0].EngineSpecific, "plain CTE should not allocate EngineSpecific")
}

func TestDML_LimitWithFillBoundariesCaptured(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM events ORDER BY id LIMIT 10 WITH FILL FROM 1 TO 100 STEP 2")
	require.NotNil(t, analysis.EngineSpecific)
	assert.Equal(t, "1", analysis.EngineSpecific["LIMIT_FILL_FROM"])
	assert.Equal(t, "100", analysis.EngineSpecific["LIMIT_FILL_TO"])
	assert.Equal(t, "2", analysis.EngineSpecific["LIMIT_FILL_STEP"])
}

func TestDML_OrderByCapturesDirectionAndNulls(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM users ORDER BY created_at DESC NULLS LAST, name ASC NULLS FIRST")
	require.Len(t, analysis.OrderByColumns, 2)
	assert.Equal(t, "created_at", analysis.OrderByColumns[0].Expression)
	assert.Equal(t, querier_dto.OrderDirectionDesc, analysis.OrderByColumns[0].Direction)
	assert.Equal(t, querier_dto.OrderNullsLast, analysis.OrderByColumns[0].Nulls)
	assert.Equal(t, "name", analysis.OrderByColumns[1].Expression)
	assert.Equal(t, querier_dto.OrderDirectionAsc, analysis.OrderByColumns[1].Direction)
	assert.Equal(t, querier_dto.OrderNullsFirst, analysis.OrderByColumns[1].Nulls)
}

func TestDML_OrderByCapturesFillSettings(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM events ORDER BY ts WITH FILL FROM 0 TO 10 STEP 1")
	require.Len(t, analysis.OrderByColumns, 1)
	column := analysis.OrderByColumns[0]
	assert.Equal(t, "ts", column.Expression)
	assert.True(t, column.HasFill)
	assert.Equal(t, "0", column.FillFrom)
	assert.Equal(t, "10", column.FillTo)
	assert.Equal(t, "1", column.FillStep)
}

func TestDML_OrderByEmptyDefaults(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM events ORDER BY ts")
	require.Len(t, analysis.OrderByColumns, 1)
	column := analysis.OrderByColumns[0]
	assert.Equal(t, querier_dto.OrderDirectionUnspecified, column.Direction)
	assert.Equal(t, querier_dto.OrderNullsUnspecified, column.Nulls)
	assert.False(t, column.HasFill)
}

func TestDML_ArrayJoinAcceptsLiteralArraySource(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT x FROM events ARRAY JOIN [1, 2, 3] AS x WHERE id = 1")
	require.Len(t, analysis.ArrayJoinClauses, 1)
	assert.Empty(t, analysis.ArrayJoinClauses[0].SourceColumn)
	assert.NotEmpty(t, analysis.ArrayJoinClauses[0].SourceExpression)
	assert.Equal(t, "x", analysis.ArrayJoinClauses[0].Alias)
}

func TestDML_ArrayJoinAcceptsFunctionCallSource(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT y FROM events ARRAY JOIN arrayMap(v -> v * 2, src) AS y")
	require.Len(t, analysis.ArrayJoinClauses, 1)
	assert.Empty(t, analysis.ArrayJoinClauses[0].SourceColumn)
	assert.NotEmpty(t, analysis.ArrayJoinClauses[0].SourceExpression)
	assert.Equal(t, "y", analysis.ArrayJoinClauses[0].Alias)
}

func TestDML_ArrayJoinBareColumnPreservesSourceColumn(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT tag FROM events ARRAY JOIN tags AS tag WHERE id = 1")
	require.Len(t, analysis.ArrayJoinClauses, 1)
	assert.Equal(t, "tags", analysis.ArrayJoinClauses[0].SourceColumn)
	assert.Empty(t, analysis.ArrayJoinClauses[0].SourceExpression, "bare column form must not populate SourceExpression")
	assert.Equal(t, "tag", analysis.ArrayJoinClauses[0].Alias)
}

func TestDML_ExistsSubqueryRegistersBodyParameter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "exists in select list",
			sql:  "SELECT EXISTS(SELECT 1 FROM orchestrator_tasks WHERE workflow_id = {workflow_id:String}) AS has_incomplete",
		},
		{
			name: "in subquery in where",
			sql:  "SELECT id FROM workflows WHERE id IN (SELECT workflow_id FROM orchestrator_tasks WHERE workflow_id = {workflow_id:String})",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			analysis := analyse(t, test.sql)
			require.Len(t, analysis.ParameterReferences, 1)
			ref := analysis.ParameterReferences[0]
			assert.Equal(t, "workflow_id", ref.Name)
			require.NotNil(t, ref.CastType, "self-typed placeholder must keep its cast type")
			assert.Equal(t, "String", ref.CastType.EngineName)
		})
	}
}

func TestAnalyse_ParametersInProjectionAndGroupBy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		sql  string
		want int
	}{
		{name: "projection arithmetic", sql: `SELECT id + {bonus:UInt32} AS total FROM users`, want: 1},
		{name: "projection function arg", sql: `SELECT greatest(x, {y:UInt32}) FROM t`, want: 1},
		{name: "group by expression", sql: `SELECT count() FROM t GROUP BY toStartOfInterval(ts, {step:UInt32})`, want: 1},
		{name: "where still works", sql: `SELECT id FROM users WHERE id = {uid:UInt32}`, want: 1},
		{name: "projection and where", sql: `SELECT id + {bonus:UInt32} FROM users WHERE id = {uid:UInt32}`, want: 2},
	}
	for index := range cases {
		testCase := cases[index]
		t.Run(testCase.name, func(testRunner *testing.T) {
			testRunner.Parallel()
			analysis := analyse(testRunner, testCase.sql)
			require.Len(testRunner, analysis.ParameterReferences, testCase.want,
				"parameters in projection/GROUP BY expressions must be registered")
			for _, ref := range analysis.ParameterReferences {
				assert.NotEmpty(testRunner, ref.Name)
			}
		})
	}
}
