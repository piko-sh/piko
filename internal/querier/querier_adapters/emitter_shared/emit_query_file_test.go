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

package emitter_shared

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

type indexedStrategy struct {
	preservesIndices bool
}

func (*indexedStrategy) ConnectionField(query *querier_dto.AnalysedQuery) string {
	return ConnectionField(query)
}

func (*indexedStrategy) DBCall(field string, method string, args []ast.Expr) *ast.CallExpr {
	return goastutil.CallExpr(
		goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), field),
			method,
		),
		args...,
	)
}

func (*indexedStrategy) QueryMethod() string     { return "QueryContext" }
func (*indexedStrategy) QueryRowMethod() string  { return "QueryRowContext" }
func (*indexedStrategy) ExecMethod() string      { return "ExecContext" }
func (*indexedStrategy) ExecReturnsResult() bool { return true }

func (*indexedStrategy) QueriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(IdentQueries))),
	)
}

func (*indexedStrategy) ExecResultReturnType() ast.Expr {
	return goastutil.SelectorExpr("sql", "Result")
}

func (*indexedStrategy) ExecResultImport(tracker *ImportTracker) {
	tracker.AddImport("database/sql")
}

func (strategy *indexedStrategy) BuildExecRowsBody(queryArgs []ast.Expr, field string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{IdentResults, IdentErr},
			strategy.DBCall(field, "ExecContext", queryArgs),
		),
		BuildErrCheck(goastutil.IntLit(0)),
		goastutil.ReturnStmt(
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentResults), "RowsAffected"),
			),
		),
	}
}

func (*indexedStrategy) BuilderQueryCall(argsExpr ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), IdentQueriesReceiver),
				IdentReader,
			),
			"QueryContext",
		),
		Args:     []ast.Expr{goastutil.CachedIdent(IdentCtx), goastutil.CachedIdent(IdentQuery), argsExpr},
		Ellipsis: 1,
	}
}

func (*indexedStrategy) BuilderQueryRowCall(argsExpr ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), IdentQueriesReceiver),
				IdentReader,
			),
			"QueryRowContext",
		),
		Args:     []ast.Expr{goastutil.CachedIdent(IdentCtx), goastutil.CachedIdent(IdentQuery), argsExpr},
		Ellipsis: 1,
	}
}

func (*indexedStrategy) RuntimeBuilderImports(tracker *ImportTracker) {
	tracker.AddImport("database/sql")
}

func (*indexedStrategy) NeedsSliceExpansion() bool          { return true }
func (*indexedStrategy) PlaceholderMarker() rune            { return '?' }
func (*indexedStrategy) ArrayJSONWrapFunc() string          { return "" }
func (*indexedStrategy) QuoteIdentifier(name string) string { return `"` + name + `"` }
func (*indexedStrategy) MaxBindVariables() int              { return 999 }
func (*indexedStrategy) UsesNumberedParams() bool           { return true }
func (strategy *indexedStrategy) PreservesPlaceholderIndices() bool {
	return strategy.preservesIndices
}

func (strategy *indexedStrategy) RuntimeBuilderUsesNumberedPlaceholders() bool {
	return strategy.preservesIndices
}

func (*indexedStrategy) WrapParameterAccess(access ast.Expr, _ string) ast.Expr { return access }
func (*indexedStrategy) UsesBracedNamedPlaceholders() bool                      { return false }
func (*indexedStrategy) ParameterAccessImports() []string                       { return nil }

func (*indexedStrategy) ParameterAccessHelperFile(_ string) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{}, nil
}

func textMappings() *querier_dto.TypeMappingTable {
	return &querier_dto.TypeMappingTable{Mappings: []querier_dto.TypeMapping{
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
	}}
}

func textColumn(name string, nullable bool) querier_dto.OutputColumn {
	return querier_dto.OutputColumn{
		Name:     name,
		SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
		Nullable: nullable,
	}
}

func textParam(name string, number int) querier_dto.QueryParameter {
	return querier_dto.QueryParameter{
		Name:    name,
		Number:  number,
		SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
		Kind:    querier_dto.ParameterDirectiveParam,
	}
}

func requireValidGo(t *testing.T, content []byte) {
	t.Helper()
	fileSet := token.NewFileSet()
	_, err := parser.ParseFile(fileSet, "generated.go", content, parser.AllErrors)
	require.NoErrorf(t, err, "emitted source must be valid Go:\n%s", content)
}

func TestEmitQueryFileEmitsCompilableMethodPerCommand(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	mappings := textMappings()

	tests := []struct {
		name          string
		query         *querier_dto.AnalysedQuery
		wantSubstrs   []string
		unwantSubstrs []string
	}{
		{
			name: "one method scans single row",
			query: &querier_dto.AnalysedQuery{
				Name:          "GetUser",
				Filename:      "users.sql",
				SQL:           "SELECT id, name FROM users WHERE id = ?1",
				Command:       querier_dto.QueryCommandOne,
				ReadOnly:      true,
				Parameters:    []querier_dto.QueryParameter{textParam("id", 1)},
				OutputColumns: []querier_dto.OutputColumn{textColumn("id", false), textColumn("name", false)},
			},
			wantSubstrs: []string{
				"func (queries *Queries) GetUser(ctx context.Context, id string) (GetUserRow, error)",
				"type GetUserRow struct",
				"const getuser = `SELECT id, name FROM users WHERE id = ?1`",
				"queries.reader.QueryRowContext(ctx, getuser, id)",
			},
		},
		{
			name: "many method iterates rows",
			query: &querier_dto.AnalysedQuery{
				Name:          "ListUsers",
				Filename:      "users.sql",
				SQL:           "SELECT id FROM users",
				Command:       querier_dto.QueryCommandMany,
				ReadOnly:      true,
				OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
			},
			wantSubstrs: []string{
				"func (queries *Queries) ListUsers(ctx context.Context) ([]ListUsersRow, error)",
				"for rows.Next()",
				"rows.Close()",
			},
		},
		{
			name: "exec method returns error only",
			query: &querier_dto.AnalysedQuery{
				Name:       "DeleteUser",
				Filename:   "users.sql",
				SQL:        "DELETE FROM users WHERE id = ?1",
				Command:    querier_dto.QueryCommandExec,
				Parameters: []querier_dto.QueryParameter{textParam("id", 1)},
			},
			wantSubstrs: []string{
				"func (queries *Queries) DeleteUser(ctx context.Context, id string) error",
				"queries.writer.ExecContext(ctx, deleteuser, id)",
			},
		},
		{
			name: "execresult method returns sql.Result",
			query: &querier_dto.AnalysedQuery{
				Name:       "InsertUser",
				Filename:   "users.sql",
				SQL:        "INSERT INTO users (name) VALUES (?1)",
				Command:    querier_dto.QueryCommandExecResult,
				Parameters: []querier_dto.QueryParameter{textParam("name", 1)},
			},
			wantSubstrs: []string{
				"(sql.Result, error)",
				"return queries.writer.ExecContext(ctx, insertuser, name)",
			},
		},
		{
			name: "execrows method returns int64",
			query: &querier_dto.AnalysedQuery{
				Name:       "TouchUsers",
				Filename:   "users.sql",
				SQL:        "UPDATE users SET seen = 1",
				Command:    querier_dto.QueryCommandExecRows,
				Parameters: nil,
			},
			wantSubstrs: []string{
				"(int64, error)",
				"results.RowsAffected()",
			},
		},
		{
			name: "stream method returns iterator",
			query: &querier_dto.AnalysedQuery{
				Name:          "StreamUsers",
				Filename:      "users.sql",
				SQL:           "SELECT id FROM users",
				Command:       querier_dto.QueryCommandStream,
				ReadOnly:      true,
				OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
			},
			wantSubstrs: []string{
				"func(yield func(StreamUsersRow, error) bool)",
				"yield(StreamUsersRow{}, err)",
			},
		},
		{
			name: "asyncexec method carries doc comment",
			query: &querier_dto.AnalysedQuery{
				Name:     "PurgeUsers",
				Filename: "users.sql",
				SQL:      "DELETE FROM users WHERE stale = 1",
				Command:  querier_dto.QueryCommandAsyncExec,
			},
			wantSubstrs: []string{
				"func (queries *Queries) PurgeUsers(ctx context.Context) error",
				"asynchronous mutation",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file, err := EmitQueryFile("mypkg", test.query.Filename, []*querier_dto.AnalysedQuery{test.query}, mappings, strategy, nil)
			require.NoError(t, err)
			require.Equal(t, "users.sql.go", file.Name)

			source := string(file.Content)
			requireValidGo(t, file.Content)

			for _, want := range test.wantSubstrs {
				assert.Containsf(t, source, want, "emitted source missing %q\n%s", want, source)
			}
			for _, unwanted := range test.unwantSubstrs {
				assert.NotContainsf(t, source, unwanted, "emitted source unexpectedly contains %q", unwanted)
			}
		})
	}
}

func TestEmitQueryFileMultiParameterUsesParamsStruct(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:     "FindUser",
		Filename: "users.sql",
		SQL:      "SELECT id FROM users WHERE name = ?1 AND city = ?2",
		Command:  querier_dto.QueryCommandOne,
		ReadOnly: true,
		Parameters: []querier_dto.QueryParameter{
			textParam("name", 1),
			textParam("city", 2),
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)
	assert.Contains(t, source, "type FindUserParams struct")
	assert.Contains(t, source, "params FindUserParams")
	assert.Contains(t, source, "params.Name")
	assert.Contains(t, source, "params.City")
}

func TestEmitQueryFileSliceParameterEmitsExpansionPreamble(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:     "UsersByIDs",
		Filename: "users.sql",
		SQL:      "SELECT id FROM users WHERE id IN (?1)",
		Command:  querier_dto.QueryCommandMany,
		ReadOnly: true,
		Parameters: []querier_dto.QueryParameter{
			{
				Name:    "ids",
				Number:  1,
				IsSlice: true,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
				Kind:    querier_dto.ParameterDirectiveParam,
			},
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)
	assert.Contains(t, source, "pikoExpandSlicePlaceholders(usersbyids,")
	assert.Contains(t, source, "expansionError")
	assert.Contains(t, source, "for _, v := range params.IDs")
	assert.Contains(t, source, "queries.reader.QueryContext(ctx, query, args...)")
}

func TestEmitQueryFileAnonymousPlaceholderEngineCollapsesIndices(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: false}
	query := &querier_dto.AnalysedQuery{
		Name:     "GetByEmail",
		Filename: "users.sql",
		SQL:      "SELECT id FROM users WHERE email = ?1 OR backup = ?1",
		Command:  querier_dto.QueryCommandOne,
		ReadOnly: true,
		Parameters: []querier_dto.QueryParameter{
			textParam("email", 1),
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Contains(t, source, "WHERE email = ? OR backup = ?")
	assert.Contains(t, source, "queries.reader.QueryRowContext(ctx, getbyemail, email, email)")
}
