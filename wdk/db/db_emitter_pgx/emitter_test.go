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

package db_emitter_pgx

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

func TestPgxEmitQuerier(t *testing.T) {
	emitter := NewPgxEmitter()
	result, err := emitter.EmitQuerier("testpkg", 0)
	require.NoError(t, err)

	source := string(result.Content)
	assert.Equal(t, "querier.go", result.Name)

	requireValidGo(t, "querier.go", source)

	assert.Contains(t, source, "DBTX")
	assert.Contains(t, source, "Exec")
	assert.Contains(t, source, "Query")
	assert.Contains(t, source, "QueryRow")
	assert.Contains(t, source, "SendBatch")
	assert.Contains(t, source, "CopyFrom")
}

func TestPgxEmitQuerierNoDatabaseSQL(t *testing.T) {
	emitter := NewPgxEmitter()
	result, err := emitter.EmitQuerier("testpkg", 0)
	require.NoError(t, err)

	source := string(result.Content)

	assert.NotContains(t, source, "ExecContext")
	assert.NotContains(t, source, "QueryContext")
	assert.NotContains(t, source, "QueryRowContext")
	assert.NotContains(t, source, "sql.Result")
	assert.NotContains(t, source, "sql.Row")
	assert.NotContains(t, source, "sql.Rows")
	assert.NotContains(t, source, "database/sql")
}

func TestPgxEmitModels(t *testing.T) {
	emitter := NewPgxEmitter()
	catalogue := &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name: "public",
				Tables: map[string]*querier_dto.Table{
					"users": {
						Name:   "users",
						Schema: "public",
						Columns: []querier_dto.Column{
							{
								Name:    "id",
								SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
							},
							{
								Name:     "email",
								SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
								Nullable: true,
							},
						},
					},
				},
			},
		},
	}

	files, err := emitter.EmitModels("testpkg", catalogue, defaultMappings())
	require.NoError(t, err)
	require.Len(t, files, 1)

	source := string(files[0].Content)

	requireValidGo(t, "models.go", source)

	assert.Contains(t, source, "User")
}

func TestPgxEmitQueriesOne(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "get_user",
			SQL:      "SELECT id, name FROM users WHERE id = $1",
			Command:  querier_dto.QueryCommandOne,
			Filename: "users.sql",
			Parameters: []querier_dto.QueryParameter{
				{
					Name:    "id",
					Number:  1,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
			},
			OutputColumns: []querier_dto.OutputColumn{
				{
					Name:    "id",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
				{
					Name:    "name",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "QueryRow")
	assert.Contains(t, source, "Scan")
}

func TestPgxEmitQueriesOptionalOne(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "get_user",
			SQL:      "SELECT id, name FROM users WHERE id = $1",
			Command:  querier_dto.QueryCommandOne,
			Optional: true,
			Filename: "users.sql",
			Parameters: []querier_dto.QueryParameter{
				{
					Name:    "id",
					Number:  1,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
			},
			OutputColumns: []querier_dto.OutputColumn{
				{
					Name:    "id",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
				{
					Name:    "name",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "pgx.ErrNoRows",
		"pgx optional one query must test the pgx no-rows sentinel")
	assert.Contains(t, source, "errors.Is",
		"optional one query must distinguish no rows from a real error via errors.Is")
	assert.Contains(t, source, "(get_userRow, bool, error)",
		"optional one query must widen the return signature to (row, bool, error)")

	requireTypeChecks(t, files)
}

func TestPgxEmitQueriesMany(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "list_users",
			SQL:      "SELECT id, name FROM users",
			Command:  querier_dto.QueryCommandMany,
			Filename: "users.sql",
			OutputColumns: []querier_dto.OutputColumn{
				{
					Name:    "id",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
				{
					Name:    "name",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "Query")
	assert.Contains(t, source, "for")
}

func TestPgxEmitQueriesExec(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "delete_user",
			SQL:      "DELETE FROM users WHERE id = $1",
			Command:  querier_dto.QueryCommandExec,
			Filename: "users.sql",
			Parameters: []querier_dto.QueryParameter{
				{
					Name:    "id",
					Number:  1,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "Exec")
}

func TestPgxEmitQueriesExecRows(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "delete_inactive_users",
			SQL:      "DELETE FROM users WHERE active = false",
			Command:  querier_dto.QueryCommandExecRows,
			Filename: "users.sql",
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "func (queries *Queries)")
	assert.Contains(t, source, "Exec(")
	assert.NotContains(t, source, "ExecContext")
	assert.Contains(t, source, "RowsAffected()")
	assert.Contains(t, source, "(int64, error)")
	assert.NotContains(t, source, "sql.Result")
}

func TestPgxEmitQueriesExecResult(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "update_user_email",
			SQL:      "UPDATE users SET email = $1 WHERE id = $2",
			Command:  querier_dto.QueryCommandExecResult,
			Filename: "users.sql",
			Parameters: []querier_dto.QueryParameter{
				{
					Name:    "email",
					Number:  1,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
				{
					Name:    "id",
					Number:  2,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "CommandTag")
	assert.NotContains(t, source, "sql.Result")
}

func TestPgxEmitQueriesStream(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "stream_users",
			SQL:      "SELECT id, name FROM users",
			Command:  querier_dto.QueryCommandStream,
			Filename: "users.sql",
			OutputColumns: []querier_dto.OutputColumn{
				{
					Name:    "id",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
				{
					Name:    "name",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "yield")
	assert.Contains(t, source, "func(")
}

func TestPgxEmitQueriesBatch(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "batch_insert_users",
			SQL:      "INSERT INTO users (name, email) VALUES ($1, $2)",
			Command:  querier_dto.QueryCommandBatch,
			Filename: "users.sql",
			Parameters: []querier_dto.QueryParameter{
				{
					Name:    "name",
					Number:  1,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
				{
					Name:    "email",
					Number:  2,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
			},
			OutputColumns: []querier_dto.OutputColumn{
				{
					Name:    "id",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				},
			},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "Batch")
	assert.Contains(t, source, "SendBatch")
}

func TestPgxEmitQueriesCopyFrom(t *testing.T) {
	emitter := NewPgxEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "copy_users",
			SQL:      "INSERT INTO users (name, email) VALUES ($1, $2)",
			Command:  querier_dto.QueryCommandCopyFrom,
			Filename: "users.sql",
			Parameters: []querier_dto.QueryParameter{
				{
					Name:    "name",
					Number:  1,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
				{
					Name:    "email",
					Number:  2,
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
			},
			InsertTable:   "users",
			InsertColumns: []string{"name", "email"},
		},
	}

	files, err := emitter.EmitQueries("testpkg", queries, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	source := string(files[0].Content)

	requireValidGo(t, "users.sql.go", source)

	assert.Contains(t, source, "CopyFrom")
	assert.Contains(t, source, "CopyFromSlice")
}

func TestPgxEmitPreparedEmpty(t *testing.T) {
	emitter := NewPgxEmitter()
	result, err := emitter.EmitPrepared("testpkg", nil)
	require.NoError(t, err)

	assert.Empty(t, result.Content)
	assert.Empty(t, result.Name)
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

func dynamicRuntimeQuery() *querier_dto.AnalysedQuery {
	return &querier_dto.AnalysedQuery{
		Name:                    "SearchPosts",
		Filename:                "posts.sql",
		SQL:                     "SELECT id, title FROM posts WHERE environment_id = $1",
		CountSQL:                "SELECT COUNT(*) FROM posts WHERE environment_id = $1",
		Command:                 querier_dto.QueryCommandMany,
		DynamicRuntime:          true,
		ReadOnly:                true,
		BaseQueryHasWhereClause: true,
		Parameters: []querier_dto.QueryParameter{
			{
				Name:    "environment_id",
				Number:  1,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
			},
		},
		AllowedColumns: []querier_dto.AllowedColumn{
			{Name: "title", SourceExpression: "posts.title", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			{Name: "created_at", SourceExpression: "posts.created_at", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
		},
		OutputColumns: []querier_dto.OutputColumn{
			{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
			{Name: "title", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
		},
	}
}

func TestPgxDynamicRuntimeEmitsNumberedPlaceholders(t *testing.T) {
	emitter := NewPgxEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicRuntimeQuery()}, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	queryFile := findFile(t, files, "posts.sql.go")
	source := string(queryFile.Content)
	requireValidGo(t, queryFile.Name, source)

	assert.Contains(t, source, `query += " LIMIT $" + strconv.Itoa(parameterCount)`,
		"pgx must emit numbered LIMIT placeholders")
	assert.Contains(t, source, `query += " OFFSET $" + strconv.Itoa(parameterCount)`,
		"pgx must emit numbered OFFSET placeholders")
	assert.NotContains(t, source, `query += " LIMIT ?"`, "pgx must not emit anonymous LIMIT placeholders")
	assert.NotContains(t, source, `query += " OFFSET ?"`, "pgx must not emit anonymous OFFSET placeholders")

	helperFile := findFile(t, files, "runtime_helpers.go")
	helperSource := string(helperFile.Content)
	requireValidGo(t, helperFile.Name, helperSource)
	assert.Contains(t, helperSource, `return "$" + strconv.Itoa(`,
		"pgx runtime WHERE-fragment helper must build numbered placeholders")
}

func TestPgxDynamicRuntimeEmitsColumnAndDirectionAllowLists(t *testing.T) {
	emitter := NewPgxEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicRuntimeQuery()}, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	queryFile := findFile(t, files, "posts.sql.go")
	source := string(queryFile.Content)

	assert.Contains(t, source, "searchpostsAllowedColumns = map[string]string{",
		"runtime builder must emit a per-query column allow-list")
	assert.Contains(t, source, `"title": "\"posts\".\"title\""`, "column allow-list maps the output name to its quoted qualified source")
	assert.Contains(t, source, `"created_at": "\"posts\".\"created_at\""`, "column allow-list maps the output name to its quoted qualified source")
	assert.Contains(t, source, "resolvedColumn, columnAllowed := searchpostsAllowedColumns[columnRoot]",
		"Where/OrderBy must consult the column allow-list")

	helperFile := findFile(t, files, "runtime_helpers.go")
	helperSource := string(helperFile.Content)

	assert.Contains(t, helperSource, "pikoAllowedDirections = map[string]bool{",
		"runtime helpers must emit the sort-direction allow-list")
	assert.Contains(t, helperSource, `"ASC":`, "direction allow-list must include ASC")
	assert.Contains(t, helperSource, `"DESC":`, "direction allow-list must include DESC")
	assert.Contains(t, helperSource, "pikoAllowedOperators = map[string]bool{",
		"runtime helpers must emit the operator allow-list")
}

func TestPgxDynamicRuntimeTypeChecks(t *testing.T) {
	emitter := NewPgxEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicRuntimeQuery()}, defaultMappings())
	require.NoError(t, err)
	requireTypeChecks(t, files)
}

func dynamicSortableQuery() *querier_dto.AnalysedQuery {
	return &querier_dto.AnalysedQuery{
		Name:      "ListPosts",
		Filename:  "list_posts.sql",
		SQL:       "SELECT id, title FROM posts WHERE author = $1 ORDER BY created_at LIMIT $2",
		Command:   querier_dto.QueryCommandMany,
		IsDynamic: true,
		ReadOnly:  true,
		Parameters: []querier_dto.QueryParameter{
			{
				Name:    "author",
				Number:  1,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				Kind:    querier_dto.ParameterDirectiveParam,
			},
			{
				Name:         "page_size",
				Number:       2,
				SQLType:      querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
				Context:      querier_dto.ParameterContextLimit,
				DefaultLimit: new(20),
				MaxLimit:     new(100),
			},
			{
				Name:            "sort",
				Number:          3,
				SQLType:         querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				Kind:            querier_dto.ParameterDirectiveSortable,
				SortableColumns: []string{"created_at", "title"},
			},
		},
		OutputColumns: []querier_dto.OutputColumn{
			{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
			{Name: "title", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
		},
	}
}

func TestPgxDynamicSortableEmitsAllowListedOrderBy(t *testing.T) {
	emitter := NewPgxEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicSortableQuery()}, defaultMappings())
	require.NoError(t, err)
	require.NotEmpty(t, files)

	queryFile := findFile(t, files, "list_posts.sql.go")
	source := string(queryFile.Content)
	requireValidGo(t, queryFile.Name, source)

	assert.Contains(t, source, "type ListPostsOrderBy string",
		"sortable query must emit a typed ORDER BY enum")
	assert.Contains(t, source, `ListPostsOrderByCreatedAt ListPostsOrderBy = "created_at"`,
		"ORDER BY enum must allow-list the declared sortable column")
	assert.Contains(t, source, `ListPostsOrderByTitle`, "ORDER BY enum must allow-list the declared sortable column")
	assert.Contains(t, source, `ListPostsOrderBy = "title"`, "ORDER BY enum must carry the column value")
	assert.Contains(t, source, `case "created_at", "title":`,
		"sortable ORDER BY switch must allow-list exactly the declared columns")
	assert.Contains(t, source, `query = query + (" ORDER BY " + string(params.Sort))`,
		"sortable query must splice the allow-listed ORDER BY into the SQL")
}

func TestPgxDynamicSortableTypeChecks(t *testing.T) {
	emitter := NewPgxEmitter()
	files, err := emitter.EmitQueries("testpkg", []*querier_dto.AnalysedQuery{dynamicSortableQuery()}, defaultMappings())
	require.NoError(t, err)
	requireTypeChecks(t, files)
}

func TestPgxQueriesTypeCheck(t *testing.T) {
	tests := []struct {
		name    string
		queries []*querier_dto.AnalysedQuery
	}{
		{
			name: "one",
			queries: []*querier_dto.AnalysedQuery{
				{
					Name:     "GetUser",
					SQL:      "SELECT id, name FROM users WHERE id = $1",
					Command:  querier_dto.QueryCommandOne,
					Filename: "users.sql",
					Parameters: []querier_dto.QueryParameter{
						{Name: "id", Number: 1, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"}},
					},
					OutputColumns: []querier_dto.OutputColumn{
						{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"}},
						{Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
					},
				},
			},
		},
		{
			name: "many",
			queries: []*querier_dto.AnalysedQuery{
				{
					Name:     "ListUsers",
					SQL:      "SELECT id, name FROM users",
					Command:  querier_dto.QueryCommandMany,
					Filename: "users.sql",
					OutputColumns: []querier_dto.OutputColumn{
						{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"}},
						{Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
					},
				},
			},
		},
	}

	emitter := NewPgxEmitter()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			files, err := emitter.EmitQueries("testpkg", test.queries, defaultMappings())
			require.NoError(t, err)
			requireTypeChecks(t, files)
		})
	}
}
