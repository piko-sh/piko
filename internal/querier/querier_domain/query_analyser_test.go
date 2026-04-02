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

package querier_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNewQueryAnalyser(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{}
	catalogue := newTestCatalogue("public")

	analyser := newQueryAnalyser(engine, catalogue)

	require.NotNil(t, analyser, "newQueryAnalyser must return a non-nil analyser")
	assert.Equal(t, engine, analyser.engine, "engine must be stored")
	assert.Equal(t, catalogue, analyser.catalogue, "catalogue must be stored")
	assert.NotNil(t, analyser.directiveParser, "directiveParser must be initialised")
	assert.NotNil(t, analyser.typeResolver, "typeResolver must be initialised")
	assert.NotNil(t, analyser.nullabilityPropagator, "nullabilityPropagator must be initialised")
	assert.NotNil(t, analyser.diagnosticAnalyser, "diagnosticAnalyser must be initialised")
	assert.NotNil(t, analyser.validator, "validator must be initialised")
}

func TestQueryAnalyser_AnalyseQuery(t *testing.T) {
	t.Parallel()

	buildCatalogue := func() *querier_dto.Catalogue {
		cat := newTestCatalogue("public")
		cat.Schemas["public"].Tables["users"] = newTestTable("users",
			querier_dto.Column{
				Name:    "id",
				SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			},
			querier_dto.Column{
				Name:    "name",
				SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
			},
		)
		return cat
	}

	buildEngine := func(
		parseFn func(string) ([]querier_dto.ParsedStatement, error),
		analyseFn func(*querier_dto.Catalogue, querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error),
	) *mockEngine {
		return &mockEngine{
			parseStatementsFn: parseFn,
			analyseQueryFn:    analyseFn,
			commentStyleFn: func() querier_dto.CommentStyle {
				return querier_dto.CommentStyle{LinePrefix: "--"}
			},
			supportedDirectivePrefixesFn: func() []querier_dto.DirectiveParameterPrefix {
				return []querier_dto.DirectiveParameterPrefix{{Prefix: '$', IsNamed: false}}
			},
		}
	}

	standardParseFn := func(sql string) ([]querier_dto.ParsedStatement, error) {
		return []querier_dto.ParsedStatement{{Location: 0, Length: len(sql)}}, nil
	}

	standardAnalyseFn := func(_ *querier_dto.Catalogue, _ querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error) {
		return &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "users", Schema: "public"},
			},
			OutputColumns: []querier_dto.RawOutputColumn{
				{
					Name:       "id",
					TableAlias: "users",
					ColumnName: "id",
					Expression: &querier_dto.ColumnRefExpression{TableAlias: "users", ColumnName: "id"},
				},
				{
					Name:       "name",
					TableAlias: "users",
					ColumnName: "name",
					Expression: &querier_dto.ColumnRefExpression{TableAlias: "users", ColumnName: "name"},
				},
			},
			ParameterReferences: []querier_dto.RawParameterReference{
				{
					Number:          1,
					ColumnReference: &querier_dto.ColumnReference{TableAlias: "users", ColumnName: "id"},
					Context:         querier_dto.ParameterContextComparison,
				},
			},
			ReadOnly: true,
		}, nil
	}

	tests := []struct {
		name string

		block queryBlock

		filename string

		parseFn   func(string) ([]querier_dto.ParsedStatement, error)
		analyseFn func(*querier_dto.Catalogue, querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error)

		wantNilQuery bool
		wantName     string
		wantCommand  querier_dto.QueryCommand

		wantDiagCodes []string

		wantOutputCount int
		wantParamCount  int
	}{
		{
			name: "simple SELECT with directives produces a valid AnalysedQuery",
			block: queryBlock{
				sql:       "-- piko.name: GetUser\n-- piko.command: one\n-- $1 as piko.param(user_id)\nSELECT id, name FROM users WHERE id = $1",
				startLine: 1,
			},
			filename:        "queries.sql",
			parseFn:         standardParseFn,
			analyseFn:       standardAnalyseFn,
			wantNilQuery:    false,
			wantName:        "GetUser",
			wantCommand:     querier_dto.QueryCommandOne,
			wantOutputCount: 2,
			wantParamCount:  1,
		},
		{
			name: "parse error returns diagnostic and nil query",
			block: queryBlock{
				sql:       "-- piko.name: BadQuery\n-- piko.command: one\nSELECT * FROM ???",
				startLine: 5,
			},
			filename: "queries.sql",
			parseFn: func(_ string) ([]querier_dto.ParsedStatement, error) {
				return nil, errors.New("syntax error at position 42")
			},
			analyseFn:     standardAnalyseFn,
			wantNilQuery:  true,
			wantDiagCodes: []string{"Q010"},
		},
		{
			name: "missing directives returns nil query with diagnostics",
			block: queryBlock{
				sql:       "SELECT 1",
				startLine: 1,
			},
			filename:      "queries.sql",
			parseFn:       standardParseFn,
			analyseFn:     standardAnalyseFn,
			wantNilQuery:  true,
			wantDiagCodes: nil,
		},
		{
			name: "engine analysis error returns diagnostic and nil query",
			block: queryBlock{
				sql:       "-- piko.name: FailAnalysis\n-- piko.command: many\nSELECT id FROM users",
				startLine: 10,
			},
			filename: "queries.sql",
			parseFn:  standardParseFn,
			analyseFn: func(_ *querier_dto.Catalogue, _ querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error) {
				return nil, errors.New("analysis failed: unsupported construct")
			},
			wantNilQuery:  true,
			wantDiagCodes: []string{"Q010"},
		},
		{
			name: "empty statement list returns diagnostic and nil query",
			block: queryBlock{
				sql:       "-- piko.name: EmptyQuery\n-- piko.command: exec\n",
				startLine: 1,
			},
			filename: "queries.sql",
			parseFn: func(_ string) ([]querier_dto.ParsedStatement, error) {
				return []querier_dto.ParsedStatement{}, nil
			},
			analyseFn:     standardAnalyseFn,
			wantNilQuery:  true,
			wantDiagCodes: []string{"Q010"},
		},
		{
			name: "multiple statements produces hint diagnostic and analyses last",
			block: queryBlock{
				sql:       "-- piko.name: MultiStmt\n-- piko.command: one\nCREATE TEMP TABLE t (x int);\nSELECT id, name FROM users WHERE id = $1",
				startLine: 1,
			},
			filename: "queries.sql",
			parseFn: func(sql string) ([]querier_dto.ParsedStatement, error) {

				return []querier_dto.ParsedStatement{
					{Location: 0, Length: 30},
					{Location: 31, Length: len(sql) - 31},
				}, nil
			},
			analyseFn:       standardAnalyseFn,
			wantNilQuery:    false,
			wantName:        "MultiStmt",
			wantCommand:     querier_dto.QueryCommandOne,
			wantDiagCodes:   []string{"Q012"},
			wantOutputCount: 2,
			wantParamCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parseFn := tt.parseFn
			if parseFn == nil {
				parseFn = standardParseFn
			}
			analyseFn := tt.analyseFn
			if analyseFn == nil {
				analyseFn = standardAnalyseFn
			}

			engine := buildEngine(parseFn, analyseFn)
			catalogue := buildCatalogue()
			analyser := newQueryAnalyser(engine, catalogue)

			ctx := context.Background()
			query, diagnostics := analyser.AnalyseQuery(ctx, tt.block, tt.filename)

			if tt.wantNilQuery {
				assert.Nil(t, query, "expected nil AnalysedQuery")
			} else {
				require.NotNil(t, query, "expected non-nil AnalysedQuery")
				assert.Equal(t, tt.wantName, query.Name, "query name mismatch")
				assert.Equal(t, tt.wantCommand, query.Command, "query command mismatch")
				assert.Equal(t, tt.filename, query.Filename, "filename must be propagated")
				assert.Equal(t, tt.block.sql, query.SQL, "SQL text must be propagated")
				assert.Equal(t, tt.block.startLine, query.Line, "start line must be propagated")

				if tt.wantOutputCount > 0 {
					assert.Len(t, query.OutputColumns, tt.wantOutputCount, "output column count mismatch")
				}
				if tt.wantParamCount > 0 {
					assert.Len(t, query.Parameters, tt.wantParamCount, "parameter count mismatch")
				}
			}

			if len(tt.wantDiagCodes) > 0 {
				require.NotEmpty(t, diagnostics, "expected diagnostics but got none")
				actualCodes := make([]string, len(diagnostics))
				for i, d := range diagnostics {
					actualCodes[i] = d.Code
				}
				for _, wantCode := range tt.wantDiagCodes {
					assert.Contains(t, actualCodes, wantCode,
						"expected diagnostic code %q in %v", wantCode, actualCodes)
				}
			}
		})
	}
}

func TestQueryAnalyser_AnalyseQuery_OutputColumnDetails(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["users"] = newTestTable("users",
		querier_dto.Column{
			Name:    "id",
			SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
		},
		querier_dto.Column{
			Name:     "email",
			SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
			Nullable: true,
		},
	)

	engine := &mockEngine{
		parseStatementsFn: func(sql string) ([]querier_dto.ParsedStatement, error) {
			return []querier_dto.ParsedStatement{{Location: 0, Length: len(sql)}}, nil
		},
		analyseQueryFn: func(_ *querier_dto.Catalogue, _ querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error) {
			return &querier_dto.RawQueryAnalysis{
				FromTables: []querier_dto.TableReference{
					{Name: "users", Schema: "public"},
				},
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Name:       "id",
						TableAlias: "users",
						ColumnName: "id",
						Expression: &querier_dto.ColumnRefExpression{TableAlias: "users", ColumnName: "id"},
					},
					{
						Name:       "email",
						TableAlias: "users",
						ColumnName: "email",
						Expression: &querier_dto.ColumnRefExpression{TableAlias: "users", ColumnName: "email"},
					},
				},
				ParameterReferences: []querier_dto.RawParameterReference{
					{
						Number:          1,
						ColumnReference: &querier_dto.ColumnReference{TableAlias: "users", ColumnName: "id"},
						Context:         querier_dto.ParameterContextComparison,
					},
				},
				ReadOnly: true,
			}, nil
		},
	}

	analyser := newQueryAnalyser(engine, catalogue)
	ctx := context.Background()

	block := queryBlock{
		sql:       "-- piko.name: GetUserEmail\n-- piko.command: one\n-- $1 as piko.param(user_id)\nSELECT id, email FROM users WHERE id = $1",
		startLine: 1,
	}

	query, diagnostics := analyser.AnalyseQuery(ctx, block, "test.sql")

	require.NotNil(t, query, "expected non-nil AnalysedQuery")
	assert.Equal(t, "GetUserEmail", query.Name)
	assert.Equal(t, querier_dto.QueryCommandOne, query.Command)
	assert.True(t, query.ReadOnly, "a SELECT query should be read-only")

	require.Len(t, query.OutputColumns, 2, "expected two output columns")
	assert.Equal(t, "id", query.OutputColumns[0].Name)
	assert.Equal(t, querier_dto.TypeCategoryInteger, query.OutputColumns[0].SQLType.Category)
	assert.Equal(t, "email", query.OutputColumns[1].Name)
	assert.Equal(t, querier_dto.TypeCategoryText, query.OutputColumns[1].SQLType.Category)

	assert.True(t, query.OutputColumns[1].Nullable, "email column should be nullable")

	require.Len(t, query.Parameters, 1, "expected one parameter")
	assert.Equal(t, "user_id", query.Parameters[0].Name)
	assert.Equal(t, 1, query.Parameters[0].Number)
	assert.Equal(t, querier_dto.TypeCategoryInteger, query.Parameters[0].SQLType.Category,
		"parameter type should be inferred from the compared column")

	for _, d := range diagnostics {
		assert.NotEqual(t, querier_dto.SeverityError, d.Severity,
			"unexpected error-severity diagnostic: %s (%s)", d.Message, d.Code)
	}
}

func TestQueryAnalyser_AnalyseQuery_ReadOnlyOverride(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["users"] = newTestTable("users",
		querier_dto.Column{
			Name:    "id",
			SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
		},
	)

	engine := &mockEngine{
		parseStatementsFn: func(sql string) ([]querier_dto.ParsedStatement, error) {
			return []querier_dto.ParsedStatement{{Location: 0, Length: len(sql)}}, nil
		},
		analyseQueryFn: func(_ *querier_dto.Catalogue, _ querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error) {
			return &querier_dto.RawQueryAnalysis{
				FromTables: []querier_dto.TableReference{
					{Name: "users", Schema: "public"},
				},
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Name:       "id",
						TableAlias: "users",
						ColumnName: "id",
						Expression: &querier_dto.ColumnRefExpression{TableAlias: "users", ColumnName: "id"},
					},
				},
				ReadOnly: false,
			}, nil
		},
	}

	analyser := newQueryAnalyser(engine, catalogue)
	ctx := context.Background()

	block := queryBlock{
		sql:       "-- piko.name: GetUsers\n-- piko.command: many\n-- piko.readonly: true\nSELECT id FROM users",
		startLine: 1,
	}

	query, _ := analyser.AnalyseQuery(ctx, block, "test.sql")

	require.NotNil(t, query, "expected non-nil AnalysedQuery")
	assert.True(t, query.ReadOnly, "piko.readonly: true should override engine detection")
}

func TestQueryAnalyser_AnalyseQuery_NameOnlyBlock(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["users"] = newTestTable("users",
		querier_dto.Column{
			Name:    "id",
			SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
		},
	)

	engine := &mockEngine{
		parseStatementsFn: func(sql string) ([]querier_dto.ParsedStatement, error) {
			return []querier_dto.ParsedStatement{{Location: 0, Length: len(sql)}}, nil
		},
		analyseQueryFn: func(_ *querier_dto.Catalogue, _ querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error) {
			return &querier_dto.RawQueryAnalysis{
				FromTables: []querier_dto.TableReference{
					{Name: "users", Schema: "public"},
				},
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Name:       "id",
						TableAlias: "users",
						ColumnName: "id",
						Expression: &querier_dto.ColumnRefExpression{TableAlias: "users", ColumnName: "id"},
					},
				},
				ReadOnly: true,
			}, nil
		},
	}

	analyser := newQueryAnalyser(engine, catalogue)
	ctx := context.Background()

	block := queryBlock{
		sql:       "-- piko.name: ListUsers\nSELECT id FROM users",
		startLine: 1,
	}

	query, _ := analyser.AnalyseQuery(ctx, block, "test.sql")

	require.NotNil(t, query, "a block with piko.name should still produce a query")
	assert.Equal(t, "ListUsers", query.Name)
}

func TestQueryAnalyser_AnalyseQuery_UnknownTableWarning(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("public")

	engine := &mockEngine{
		parseStatementsFn: func(sql string) ([]querier_dto.ParsedStatement, error) {
			return []querier_dto.ParsedStatement{{Location: 0, Length: len(sql)}}, nil
		},
		analyseQueryFn: func(_ *querier_dto.Catalogue, _ querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error) {
			return &querier_dto.RawQueryAnalysis{
				FromTables: []querier_dto.TableReference{
					{Name: "users", Schema: "public"},
				},
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Name:       "id",
						TableAlias: "users",
						ColumnName: "id",
						Expression: &querier_dto.ColumnRefExpression{TableAlias: "users", ColumnName: "id"},
					},
				},
				ReadOnly: true,
			}, nil
		},
	}

	analyser := newQueryAnalyser(engine, catalogue)
	ctx := context.Background()

	block := queryBlock{
		sql:       "-- piko.name: MissingTable\n-- piko.command: many\nSELECT id FROM users",
		startLine: 1,
	}

	query, diagnostics := analyser.AnalyseQuery(ctx, block, "test.sql")

	require.NotNil(t, query, "query should still be produced despite unresolved table")
	assert.Equal(t, "MissingTable", query.Name)

	var foundQ003 bool
	for _, d := range diagnostics {
		if d.Code == "Q003" {
			foundQ003 = true
			assert.Equal(t, querier_dto.SeverityWarning, d.Severity,
				"Q003 should be a warning, not an error")
			break
		}
	}
	assert.True(t, foundQ003, "expected Q003 warning for unresolved table reference")
}
