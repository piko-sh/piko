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
	"go/parser"
	"go/token"
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

func TestPgxEmitQuerier(t *testing.T) {
	emitter := NewPgxEmitter()
	result, err := emitter.EmitQuerier("testpkg", 0)
	require.NoError(t, err)

	source := string(result.Content)
	assert.Equal(t, "querier.go", result.Name)

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "querier.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated querier code must be valid Go")

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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "models.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated models code must be valid Go")

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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "users.sql.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated :one query code must be valid Go")

	assert.Contains(t, source, "QueryRow")
	assert.Contains(t, source, "Scan")
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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "users.sql.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated :many query code must be valid Go")

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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "users.sql.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated :exec query code must be valid Go")

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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "users.sql.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated :execrows query code must be valid Go")

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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "users.sql.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated :execresult query code must be valid Go")

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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "users.sql.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated :stream query code must be valid Go")

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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "users.sql.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated :batch query code must be valid Go")

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

	file_set := token.NewFileSet()
	_, parse_err := parser.ParseFile(file_set, "users.sql.go", source, parser.AllErrors)
	require.NoError(t, parse_err, "generated :copyfrom query code must be valid Go")

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
