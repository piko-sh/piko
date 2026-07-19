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

package db_emitter_clickhouse

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func defaultMappings() *querier_dto.TypeMappingTable {
	return &querier_dto.TypeMappingTable{
		Mappings: []querier_dto.TypeMapping{
			{
				SQLCategory: querier_dto.TypeCategoryInteger,
				NotNull:     querier_dto.GoType{Name: "int64"},
				Nullable:    querier_dto.GoType{Name: "*int64"},
			},
			{
				SQLCategory: querier_dto.TypeCategoryText,
				NotNull:     querier_dto.GoType{Name: "string"},
				Nullable:    querier_dto.GoType{Name: "*string"},
			},
			{
				SQLCategory: querier_dto.TypeCategoryBoolean,
				NotNull:     querier_dto.GoType{Name: "bool"},
				Nullable:    querier_dto.GoType{Name: "*bool"},
			},
		},
	}
}

func requireValidGo(t *testing.T, name, source string) {
	t.Helper()
	fileSet := token.NewFileSet()
	_, err := parser.ParseFile(fileSet, name, source, parser.AllErrors)
	require.NoErrorf(t, err, "generated source must be valid Go:\n%s", source)
}

const (
	typeCheckPrelude = `package testpkg

import "context"

type pikoTestRows struct{}

func (pikoTestRows) Close()            {}
func (pikoTestRows) Next() bool        { return false }
func (pikoTestRows) Scan(...any) error { return nil }
func (pikoTestRows) Err() error        { return nil }

type pikoTestRow struct{}

func (pikoTestRow) Scan(...any) error { return nil }

type pikoTestDB struct{}

func (pikoTestDB) Query(ctx context.Context, query string, args ...any) (pikoTestRows, error) {
	return pikoTestRows{}, nil
}

func (pikoTestDB) QueryRow(ctx context.Context, query string, args ...any) pikoTestRow {
	return pikoTestRow{}
}

func (pikoTestDB) Exec(ctx context.Context, query string, args ...any) error { return nil }

type Queries struct {
	db pikoTestDB
}
`
)

func requireTypeChecks(t *testing.T, files []querier_dto.GeneratedFile) {
	t.Helper()
	fileSet := token.NewFileSet()

	preludeFile, parseError := parser.ParseFile(fileSet, "prelude_test.go", typeCheckPrelude, parser.AllErrors)
	require.NoError(t, parseError, "type-check prelude must parse")
	astFiles := []*ast.File{preludeFile}

	for _, file := range files {
		parsed, fileError := parser.ParseFile(fileSet, file.Name, string(file.Content), parser.AllErrors)
		require.NoErrorf(t, fileError, "generated %s must parse:\n%s", file.Name, file.Content)
		astFiles = append(astFiles, parsed)
	}

	config := types.Config{Importer: importer.ForCompiler(fileSet, "source", nil)}
	_, checkError := config.Check("testpkg", fileSet, astFiles, nil)
	require.NoError(t, checkError, "generated files must type-check, not merely parse")
}

func findFile(t *testing.T, files []querier_dto.GeneratedFile, name string) querier_dto.GeneratedFile {
	t.Helper()
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	t.Fatalf("generated file %q not found", name)
	return querier_dto.GeneratedFile{}
}

func TestEmitQuerierShape(t *testing.T) {
	emitter := NewClickHouseEmitter()
	result, err := emitter.EmitQuerier("testpkg", 0)
	require.NoError(t, err)
	assert.Equal(t, "querier.go", result.Name)

	source := string(result.Content)
	requireValidGo(t, "querier.go", source)

	assert.Contains(t, source, "DBTX")
	assert.Contains(t, source, "PrepareBatch(")
	assert.Contains(t, source, "driver.Batch")
	assert.Contains(t, source, "driver.Rows")
	assert.Contains(t, source, "driver.Row")
	assert.Contains(t, source, importDriver)

	assert.Contains(t, source, "Exec(ctx context.Context, query string, args ...any) error")

	assert.NotContains(t, source, "database/sql")
	assert.NotContains(t, source, "ExecContext")
	assert.NotContains(t, source, "sql.Result")
	assert.NotContains(t, source, "pgx")
	assert.NotContains(t, source, "SendBatch")
	assert.NotContains(t, source, "CopyFrom")
	assert.NotContains(t, source, "WithTx")
	assert.NotContains(t, source, "RunInTx")
}

func TestEmitQueriesExecSingleValue(t *testing.T) {
	emitter := NewClickHouseEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "delete_event",
			SQL:      "DELETE FROM events WHERE id = ?",
			Command:  querier_dto.QueryCommandExec,
			Filename: "events.sql",
			Parameters: []querier_dto.QueryParameter{
				{Name: "id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)
	requireValidGo(t, "events.sql.go", source)

	assert.Contains(t, source, "(ctx context.Context, id int64) error")
	assert.Contains(t, source, "return queries.db.Exec(ctx, deleteEvent, id)")
	assert.NotContains(t, source, "_, err := queries.db.Exec(")
	assert.NotContains(t, source, "pikoClickHouseFormat")
	assert.NotContains(t, source, "clickhouse.Named")
}

func TestEmitQueriesBatchNative(t *testing.T) {
	emitter := NewClickHouseEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "insert_events",
			SQL:      "INSERT INTO events (host_id, kind) VALUES (?, ?)",
			Command:  querier_dto.QueryCommandBatch,
			Filename: "events.sql",
			Parameters: []querier_dto.QueryParameter{
				{Name: "host_id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
				{Name: "kind", Number: 2, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)
	requireValidGo(t, "events.sql.go", source)

	assert.Contains(t, source, "batch, err := queries.db.PrepareBatch(ctx, insertEvents)")
	assert.Contains(t, source, "batch.Append(item.HostID, item.Kind)")
	assert.Contains(t, source, "return batch.Send()")

	assert.NotContains(t, source, "SendBatch")
	assert.NotContains(t, source, "pgx")
	assert.NotContains(t, source, "pikoClickHouseFormat")
	assert.NotContains(t, source, "clickhouse.Named")
}

func TestEmitQueriesBatchNativeHonoursNullableParams(t *testing.T) {
	emitter := NewClickHouseEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "insert_samples",
			SQL:      "INSERT INTO samples (id, pid, label) VALUES (?, ?, ?)",
			Command:  querier_dto.QueryCommandBatch,
			Filename: "samples.sql",
			Parameters: []querier_dto.QueryParameter{
				{Name: "id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}},
				{Name: "pid", Number: 2, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64", Nullable: true}},
				{Name: "label", Number: 3, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String", Nullable: true}},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)
	requireValidGo(t, "samples.sql.go", source)

	assert.Regexp(t, `ID\s+int64`, source, "non-nullable param stays a value field")
	assert.Regexp(t, `Pid\s+\*int64`, source, "Nullable(Int64) param must be a pointer field")
	assert.Regexp(t, `Label\s+\*string`, source, "Nullable(String) param must be a pointer field")
}

func TestEmitQueriesCopyFromMapsToBatch(t *testing.T) {
	emitter := NewClickHouseEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "copy_events",
			SQL:      "INSERT INTO events (host_id) VALUES (?)",
			Command:  querier_dto.QueryCommandCopyFrom,
			Filename: "events.sql",
			Parameters: []querier_dto.QueryParameter{
				{Name: "host_id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)
	requireValidGo(t, "events.sql.go", source)
	assert.Contains(t, source, "PrepareBatch(ctx, copyEvents)")
	assert.Contains(t, source, "return batch.Send()")
}

func TestEmitQueriesRejectsExecRows(t *testing.T) {
	emitter := NewClickHouseEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "purge_old",
			SQL:      "DELETE FROM events WHERE ts < ?",
			Command:  querier_dto.QueryCommandExecRows,
			Filename: "events.sql",
		},
	}
	_, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "purge_old")
	assert.Contains(t, err.Error(), "execrows")
}

func TestEmitQueriesRejectsExecResult(t *testing.T) {
	emitter := NewClickHouseEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "update_event",
			SQL:      "ALTER TABLE events UPDATE kind = ? WHERE id = ?",
			Command:  querier_dto.QueryCommandExecResult,
			Filename: "events.sql",
		},
	}
	_, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update_event")
	assert.Contains(t, err.Error(), "execresult")
}

func TestEmitQueriesManyReads(t *testing.T) {
	emitter := NewClickHouseEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "list_events",
			SQL:      "SELECT host_id, kind FROM events WHERE host_id = ?",
			Command:  querier_dto.QueryCommandMany,
			ReadOnly: true,
			Filename: "events.sql",
			Parameters: []querier_dto.QueryParameter{
				{Name: "host_id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
			},
			OutputColumns: []querier_dto.OutputColumn{
				{Name: "host_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
				{Name: "kind", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)
	requireValidGo(t, "events.sql.go", source)
	assert.Contains(t, source, "queries.db.Query(ctx, listEvents, hostID)")
	assert.Contains(t, source, "rows.Scan(")
	assert.NotContains(t, source, "database/sql")
	assert.NotContains(t, source, "pikoClickHouseFormat")
}

func TestEmitQueriesOptionalOne(t *testing.T) {
	emitter := NewClickHouseEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "GetEvent",
			SQL:      "SELECT host_id, kind FROM events WHERE id = ?",
			Command:  querier_dto.QueryCommandOne,
			Optional: true,
			Filename: "events.sql",
			Parameters: []querier_dto.QueryParameter{
				{Name: "id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}},
			},
			OutputColumns: []querier_dto.OutputColumn{
				{Name: "host_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
				{Name: "kind", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	queryFile := findFile(t, files, "events.sql.go")
	source := string(queryFile.Content)
	requireValidGo(t, queryFile.Name, source)

	assert.Contains(t, source, "sql.ErrNoRows",
		"ClickHouse optional one query must test the database/sql no-rows sentinel")
	assert.Contains(t, source, `"database/sql"`,
		"ClickHouse optional one query must import database/sql for sql.ErrNoRows")
	assert.Contains(t, source, "errors.Is",
		"optional one query must distinguish no rows from a real error via errors.Is")
	assert.Contains(t, source, "(GetEventRow, bool, error)",
		"optional one query must widen the return signature to (row, bool, error)")

	requireTypeChecks(t, files)
}

func TestEmitModelsDelegates(t *testing.T) {
	emitter := NewClickHouseEmitter()
	catalogue := &querier_dto.Catalogue{
		DefaultSchema: "default",
		Schemas: map[string]*querier_dto.Schema{
			"default": {
				Name: "default",
				Tables: map[string]*querier_dto.Table{
					"events": {
						Name:   "events",
						Schema: "default",
						Columns: []querier_dto.Column{
							{Name: "host_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
						},
					},
				},
			},
		},
	}

	files, err := emitter.EmitModels("testpkg", catalogue, defaultMappings())
	require.NoError(t, err)
	require.Len(t, files, 1)
	requireValidGo(t, "models.go", string(files[0].Content))
	assert.Contains(t, string(files[0].Content), "Event")
}

func TestEmitPreparedEmpty(t *testing.T) {
	emitter := NewClickHouseEmitter()
	file, err := emitter.EmitPrepared("testpkg", nil)
	require.NoError(t, err)
	assert.Empty(t, file.Name)
	assert.Empty(t, file.Content)
}

func dynamicRuntimeQuery() *querier_dto.AnalysedQuery {
	return &querier_dto.AnalysedQuery{
		Name:                    "SearchEvents",
		Filename:                "events.sql",
		SQL:                     "SELECT host_id, kind FROM events WHERE environment_id = ?",
		CountSQL:                "SELECT COUNT(*) FROM events WHERE environment_id = ?",
		Command:                 querier_dto.QueryCommandMany,
		DynamicRuntime:          true,
		ReadOnly:                true,
		BaseQueryHasWhereClause: true,
		Parameters: []querier_dto.QueryParameter{
			{
				Name:    "environment_id",
				Number:  1,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"},
			},
		},
		AllowedColumns: []querier_dto.AllowedColumn{
			{Name: "kind", SourceExpression: "events.kind", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			{Name: "created_at", SourceExpression: "events.created_at", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
		},
		OutputColumns: []querier_dto.OutputColumn{
			{Name: "host_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
			{Name: "kind", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
		},
	}
}

func TestClickHouseDynamicRuntimeEmitsAnonymousPlaceholders(t *testing.T) {
	emitter := NewClickHouseEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicRuntimeQuery()}, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	queryFile := findFile(t, files, "events.sql.go")
	source := string(queryFile.Content)
	requireValidGo(t, queryFile.Name, source)

	assert.Contains(t, source, `query += " LIMIT ?"`, "ClickHouse must emit anonymous LIMIT placeholders")
	assert.Contains(t, source, `query += " OFFSET ?"`, "ClickHouse must emit anonymous OFFSET placeholders")
	assert.NotContains(t, source, `query += " LIMIT $"`, "ClickHouse must not emit numbered LIMIT placeholders")
	assert.NotContains(t, source, `query += " OFFSET $"`, "ClickHouse must not emit numbered OFFSET placeholders")
	assert.NotContains(t, source, "strconv.Itoa", "ClickHouse runtime builder must not number placeholders")

	helperFile := findFile(t, files, "runtime_helpers.go")
	helperSource := string(helperFile.Content)
	requireValidGo(t, helperFile.Name, helperSource)
	assert.Contains(t, helperSource, `return "?"`,
		"ClickHouse runtime WHERE-fragment helper must build anonymous placeholders")
	assert.NotContains(t, helperSource, `return "$" + strconv.Itoa(`,
		"ClickHouse runtime WHERE-fragment helper must not number placeholders")
	assert.NotContains(t, helperSource, `"strconv"`,
		"ClickHouse runtime helpers must not import strconv for placeholder numbering")
}

func TestClickHouseDynamicRuntimeEmitsColumnAndDirectionAllowLists(t *testing.T) {
	emitter := NewClickHouseEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicRuntimeQuery()}, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	queryFile := findFile(t, files, "events.sql.go")
	source := string(queryFile.Content)

	assert.Contains(t, source, "searcheventsAllowedColumns = map[string]string{",
		"runtime builder must emit a per-query column allow-list")
	assert.Contains(t, source, "\"kind\": \"`events`.`kind`\"", "column allow-list maps the output name to its backtick-quoted qualified source")
	assert.Contains(t, source, "\"created_at\": \"`events`.`created_at`\"", "column allow-list maps the output name to its backtick-quoted qualified source")
	assert.Contains(t, source, "resolvedColumn, columnAllowed := searcheventsAllowedColumns[columnRoot]",
		"Where/OrderBy must consult the column allow-list")

	helperFile := findFile(t, files, "runtime_helpers.go")
	helperSource := string(helperFile.Content)

	assert.Contains(t, helperSource, "pikoAllowedDirections = map[string]bool{",
		"runtime helpers must emit the sort-direction allow-list")
	assert.Contains(t, helperSource, `"ASC":`, "direction allow-list must include ASC")
	assert.Contains(t, helperSource, `"DESC":`, "direction allow-list must include DESC")
	assert.Contains(t, helperSource, "pikoAllowedOperators = map[string]bool{",
		"runtime helpers must emit the operator allow-list")
	assert.NotContains(t, helperSource, `"?|":`,
		"ClickHouse must not allow-list the Postgres JSONB existence operators (anonymous-marker engine)")
}

func TestClickHouseDynamicRuntimeTypeChecks(t *testing.T) {
	emitter := NewClickHouseEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicRuntimeQuery()}, defaultMappings())
	require.NoError(t, err)
	requireTypeChecks(t, files)
}

func dynamicSortableQuery() *querier_dto.AnalysedQuery {
	defaultLimit := 20
	maxLimit := 100
	return &querier_dto.AnalysedQuery{
		Name:      "ListEvents",
		Filename:  "list_events.sql",
		SQL:       "SELECT host_id, kind FROM events WHERE host_id = ? ORDER BY created_at LIMIT ?",
		Command:   querier_dto.QueryCommandMany,
		IsDynamic: true,
		ReadOnly:  true,
		Parameters: []querier_dto.QueryParameter{
			{
				Name:    "host_id",
				Number:  1,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"},
				Kind:    querier_dto.ParameterDirectiveParam,
			},
			{
				Name:         "page_size",
				Number:       2,
				SQLType:      querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"},
				Context:      querier_dto.ParameterContextLimit,
				DefaultLimit: &defaultLimit,
				MaxLimit:     &maxLimit,
			},
			{
				Name:            "sort",
				Number:          3,
				SQLType:         querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"},
				Kind:            querier_dto.ParameterDirectiveSortable,
				SortableColumns: []string{"created_at", "kind"},
			},
		},
		OutputColumns: []querier_dto.OutputColumn{
			{Name: "host_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
			{Name: "kind", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
		},
	}
}

func TestClickHouseDynamicSortableEmitsAllowListedOrderBy(t *testing.T) {
	emitter := NewClickHouseEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicSortableQuery()}, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	queryFile := findFile(t, files, "list_events.sql.go")
	source := string(queryFile.Content)
	requireValidGo(t, queryFile.Name, source)

	assert.Contains(t, source, "type ListEventsOrderBy string",
		"sortable query must emit a typed ORDER BY enum")
	assert.Contains(t, source, `ListEventsOrderByCreatedAt ListEventsOrderBy = "created_at"`,
		"ORDER BY enum must allow-list the declared sortable column")
	assert.Contains(t, source, `ListEventsOrderByKind`, "ORDER BY enum must allow-list the declared sortable column")
	assert.Contains(t, source, `ListEventsOrderBy = "kind"`, "ORDER BY enum must carry the column value")
	assert.Contains(t, source, `case "created_at", "kind":`,
		"sortable ORDER BY switch must allow-list exactly the declared columns")
	assert.Contains(t, source, `query = query + (" ORDER BY " + string(params.Sort))`,
		"sortable query must splice the allow-listed ORDER BY into the SQL")
}

func TestClickHouseDynamicSortableTypeChecks(t *testing.T) {
	emitter := NewClickHouseEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicSortableQuery()}, defaultMappings())
	require.NoError(t, err)
	requireTypeChecks(t, files)
}

func TestClickHouseQueriesTypeCheck(t *testing.T) {
	tests := []struct {
		name    string
		queries []*querier_dto.AnalysedQuery
	}{
		{
			name: "one",
			queries: []*querier_dto.AnalysedQuery{
				{
					Name:     "GetEvent",
					SQL:      "SELECT host_id, kind FROM events WHERE id = ?",
					Command:  querier_dto.QueryCommandOne,
					Filename: "events.sql",
					Parameters: []querier_dto.QueryParameter{
						{Name: "id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}},
					},
					OutputColumns: []querier_dto.OutputColumn{
						{Name: "host_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
						{Name: "kind", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
					},
				},
			},
		},
		{
			name: "many",
			queries: []*querier_dto.AnalysedQuery{
				{
					Name:     "ListEventsStatic",
					SQL:      "SELECT host_id, kind FROM events WHERE host_id = ?",
					Command:  querier_dto.QueryCommandMany,
					Filename: "events.sql",
					Parameters: []querier_dto.QueryParameter{
						{Name: "host_id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
					},
					OutputColumns: []querier_dto.OutputColumn{
						{Name: "host_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
						{Name: "kind", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
					},
				},
			},
		},
	}

	emitter := NewClickHouseEmitter()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			files, err := emitter.EmitQueries("testpkg", test.queries, defaultMappings())
			require.NoError(t, err)
			requireTypeChecks(t, files)
		})
	}
}
