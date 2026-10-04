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

package db_engine_sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNewSQLiteEngine(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	assert.Equal(t, "sqlite", engine.Dialect(), "dialect should be sqlite")
	assert.Equal(t, "main", engine.DefaultSchema(), "SQLite default schema should be main")
	assert.Equal(t, querier_dto.ParameterStyleQuestion, engine.ParameterStyle(), "SQLite uses question-mark parameters")
	assert.True(t, engine.SupportsReturning(), "SQLite supports RETURNING clauses")
	assert.NotNil(t, engine.BuiltinFunctions(), "function catalogue should be initialised")
	assert.NotNil(t, engine.BuiltinTypes(), "type catalogue should be initialised")
}

func TestNormaliseTypeName(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	tests := []struct {
		name           string
		input          string
		wantEngineName string
		wantCategory   querier_dto.SQLTypeCategory
	}{

		{
			name:           "INTEGER maps to int4",
			input:          "INTEGER",
			wantEngineName: "int4",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "REAL maps to real",
			input:          "REAL",
			wantEngineName: "real",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "TEXT maps to text",
			input:          "TEXT",
			wantEngineName: "text",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "BLOB maps to blob",
			input:          "BLOB",
			wantEngineName: "blob",
			wantCategory:   querier_dto.TypeCategoryBytea,
		},

		{
			name:           "int maps to int4 via built-in lookup",
			input:          "int",
			wantEngineName: "int4",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "bigint maps to int8",
			input:          "bigint",
			wantEngineName: "int8",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "smallint maps to int2",
			input:          "smallint",
			wantEngineName: "int2",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "varchar maps to text",
			input:          "varchar",
			wantEngineName: "text",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "float maps to real",
			input:          "float",
			wantEngineName: "real",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "double maps to real",
			input:          "double",
			wantEngineName: "real",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "boolean maps to boolean",
			input:          "boolean",
			wantEngineName: "boolean",
			wantCategory:   querier_dto.TypeCategoryBoolean,
		},

		{
			name:           "case insensitive normalisation",
			input:          "Integer",
			wantEngineName: "int4",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},

		{
			name:           "unknown type containing INT falls back to int4 affinity",
			input:          "myinttype",
			wantEngineName: "int4",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},

		{
			name:           "unknown type containing CHAR falls back to text affinity",
			input:          "nativechar",
			wantEngineName: "text",
			wantCategory:   querier_dto.TypeCategoryText,
		},

		{
			name:           "unknown type with no affinity match defaults to numeric",
			input:          "currency",
			wantEngineName: "numeric",
			wantCategory:   querier_dto.TypeCategoryDecimal,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := engine.NormaliseTypeName(testCase.input)

			assert.Equal(t, testCase.wantEngineName, result.EngineName, "engine name mismatch")
			assert.Equal(t, testCase.wantCategory, result.Category, "category mismatch")
		})
	}
}

func TestPromoteType(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	tests := []struct {
		name  string
		left  querier_dto.SQLType
		right querier_dto.SQLType
	}{
		{
			name:  "integer and integer returns left",
			left:  querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
			right: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		},
		{
			name:  "real and real returns left",
			left:  querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "real"},
			right: querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "real"},
		},
		{
			name:  "text and text returns left",
			left:  querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
			right: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		},
		{
			name:  "blob and blob returns left",
			left:  querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
			right: querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := engine.PromoteType(testCase.left, testCase.right)
			assert.Equal(t, testCase.left, result, "SQLite promotion should always return the left operand")
		})
	}
}

func TestCanImplicitCast(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	t.Run("numeric types are castable to wider numeric types", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryFloat),
			"integer to float should be allowed")
		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryDecimal),
			"integer to decimal should be allowed")
		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryDecimal, querier_dto.TypeCategoryFloat),
			"decimal to float should be allowed")
	})

	t.Run("text types are castable to text", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryText, querier_dto.TypeCategoryText),
			"text to text should be allowed")
	})

	t.Run("incompatible casts are rejected", func(t *testing.T) {
		t.Parallel()

		assert.False(t, engine.CanImplicitCast(querier_dto.TypeCategoryFloat, querier_dto.TypeCategoryInteger),
			"float to integer should not be allowed")
		assert.False(t, engine.CanImplicitCast(querier_dto.TypeCategoryText, querier_dto.TypeCategoryInteger),
			"text to integer should not be allowed")
	})
}

func TestDefaultSchema(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	assert.Equal(t, "main", engine.DefaultSchema(),
		"SQLite default schema should be main")
}

func TestBuiltinFunctions(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()
	catalogue := engine.BuiltinFunctions()

	require.NotNil(t, catalogue, "function catalogue must not be nil")
	require.NotNil(t, catalogue.Functions, "function map must not be nil")

	expectedFunctions := []string{"abs", "count", "length", "typeof"}
	for _, name := range expectedFunctions {
		signatures, exists := catalogue.Functions[name]
		assert.True(t, exists, "expected built-in function %q to be registered", name)
		assert.NotEmpty(t, signatures, "expected at least one signature for %q", name)
	}
}

func TestBuiltinTypes(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()
	catalogue := engine.BuiltinTypes()

	require.NotNil(t, catalogue, "type catalogue must not be nil")
	require.NotNil(t, catalogue.Types, "type map must not be nil")

	expectedTypes := []string{"integer", "real", "text", "blob", "boolean", "json", "varchar", "int"}
	for _, name := range expectedTypes {
		_, exists := catalogue.Types[name]
		assert.True(t, exists, "expected built-in type %q to be registered", name)
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "small value below 1 KiB",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "exactly 1 KiB",
			bytes: 1024,
			want:  "1.0 KiB",
		},
		{
			name:  "value in MiB range",
			bytes: 1024 * 1024 * 5,
			want:  "5.0 MiB",
		},
		{
			name:  "value in GiB range",
			bytes: 1024 * 1024 * 1024 * 2,
			want:  "2.0 GiB",
		},
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 B",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := formatBytes(testCase.bytes)
			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestNormaliseTypeName_AffinityRules(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	tests := []struct {
		name           string
		input          string
		modifiers      []int
		wantEngineName string
		wantCategory   querier_dto.SQLTypeCategory
	}{
		{
			name:           "type containing BLOB falls back to blob affinity",
			input:          "myblob",
			wantEngineName: "blob",
			wantCategory:   querier_dto.TypeCategoryBytea,
		},
		{
			name:           "type containing REAL falls back to real affinity",
			input:          "surreal",
			wantEngineName: "real",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "type containing FLOA falls back to real affinity",
			input:          "floating",
			wantEngineName: "real",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "type containing DOUB falls back to real affinity",
			input:          "redoubled",
			wantEngineName: "real",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "type containing CLOB falls back to text affinity",
			input:          "myclobtype",
			wantEngineName: "text",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "type containing TEXT falls back to text affinity",
			input:          "metatext",
			wantEngineName: "text",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "empty type name defaults to blob",
			input:          "",
			wantEngineName: "blob",
			wantCategory:   querier_dto.TypeCategoryBytea,
		},
		{
			name:           "decimal with precision modifier",
			input:          "numeric",
			modifiers:      []int{8},
			wantEngineName: "numeric",
			wantCategory:   querier_dto.TypeCategoryDecimal,
		},
		{
			name:           "decimal with precision and scale modifiers",
			input:          "decimal",
			modifiers:      []int{10, 2},
			wantEngineName: "numeric",
			wantCategory:   querier_dto.TypeCategoryDecimal,
		},
		{
			name:           "text with length modifier",
			input:          "varchar",
			modifiers:      []int{255},
			wantEngineName: "text",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "integer with modifier ignored",
			input:          "integer",
			modifiers:      []int{11},
			wantEngineName: "int4",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := engine.NormaliseTypeName(testCase.input, testCase.modifiers...)
			assert.Equal(t, testCase.wantEngineName, result.EngineName, "engine name mismatch")
			assert.Equal(t, testCase.wantCategory, result.Category, "category mismatch")
		})
	}
}

func TestAnalyseQuery_UnexpectedStatementType(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	_, err := engine.AnalyseQuery(nil, querier_dto.ParsedStatement{})
	assert.Error(t, err, "AnalyseQuery should return an error for nil Raw")
}

func TestApplyDDL_UnexpectedStatementType(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	_, err := engine.ApplyDDL(context.Background(), querier_dto.ParsedStatement{})
	assert.Error(t, err, "ApplyDDL should return an error for nil Raw")
}

func TestApplyDDLReturnsSyntaxErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		sql       string
		wantError string
	}{
		{name: "rename column without TO", sql: "ALTER TABLE t RENAME COLUMN a b", wantError: "expected keyword [TO]"},
		{name: "unterminated DEFAULT expression", sql: "CREATE TABLE t (a INTEGER DEFAULT (1", wantError: "unmatched parenthesis"},
		{name: "unterminated CHECK expression", sql: "CREATE TABLE t (a INTEGER CHECK (a > 0", wantError: "unmatched parenthesis"},
		{name: "unterminated REFERENCES column list", sql: "CREATE TABLE t (a INTEGER REFERENCES u(id", wantError: "unmatched parenthesis"},
		{name: "unterminated generated column expression", sql: "CREATE TABLE t (a INTEGER GENERATED ALWAYS AS (a + 1", wantError: "unmatched parenthesis"},
		{name: "unterminated table CHECK constraint", sql: "CREATE TABLE t (a INTEGER, CHECK (a > 0", wantError: "unmatched parenthesis"},
		{name: "unterminated virtual table arguments", sql: "CREATE VIRTUAL TABLE t USING fts5(a, b", wantError: "parsing virtual table arguments"},
		{name: "view without AS", sql: "CREATE VIEW v x SELECT 1", wantError: "expected keyword [AS]"},
		{name: "index without ON", sql: "CREATE INDEX i t (a)", wantError: "expected keyword [ON]"},
		{name: "trigger without ON", sql: "CREATE TRIGGER tr AFTER INSERT users BEGIN SELECT 1; END", wantError: "expected keyword [ON]"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewSQLiteEngine()
			statements, err := engine.ParseStatements(testCase.sql)
			require.NoError(t, err)
			require.NotEmpty(t, statements)

			mutation, err := engine.ApplyDDL(context.Background(), statements[0])

			require.Error(t, err)
			assert.Nil(t, mutation)
			assert.Contains(t, err.Error(), testCase.wantError)
			assert.NotContains(t, err.Error(), "panic")
		})
	}
}

func TestAnalyseQueryReturnsSyntaxErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantSentinel error
		name         string
		sql          string
		wantError    string
		options      []Option
	}{
		{name: "UPDATE without SET", sql: "UPDATE t a = 1", wantError: "expected keyword [SET]"},
		{name: "DELETE without FROM", sql: "DELETE t", wantError: "expected keyword [FROM]"},
		{name: "compound arm that is not a SELECT", sql: "SELECT 1 UNION VALUES (2)", wantError: "expected keyword [SELECT]"},
		{name: "unterminated USING list", sql: "SELECT * FROM a JOIN b USING (id", wantError: "unmatched parenthesis"},
		{name: "unterminated ON CONFLICT target", sql: "INSERT INTO t (a) VALUES (1) ON CONFLICT (a DO NOTHING", wantError: "unmatched parenthesis"},
		{name: "unterminated CAST type modifiers", sql: "SELECT CAST(? AS VARCHAR(10 FROM t", wantError: "parsing CAST type modifiers"},
		{
			name:         "expression nested past the cap inside a CTE",
			sql:          "WITH c AS (SELECT ((((((((((1)))))))))) AS x) SELECT x FROM c",
			options:      []Option{WithMaxParseDepth(8)},
			wantSentinel: errExpressionDepthExceeded,
		},
		{
			name:         "expression nested past the cap inside a scalar subquery",
			sql:          "SELECT (SELECT ((((((((((1))))))))))) FROM t",
			options:      []Option{WithMaxParseDepth(8)},
			wantSentinel: errExpressionDepthExceeded,
		},
		{
			name:         "expression nested past the cap inside a derived table",
			sql:          "SELECT * FROM (SELECT ((((((((((1)))))))))) AS x) d",
			options:      []Option{WithMaxParseDepth(8)},
			wantSentinel: errExpressionDepthExceeded,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewSQLiteEngine(testCase.options...)
			statements, err := engine.ParseStatements(testCase.sql)
			require.NoError(t, err)
			require.NotEmpty(t, statements)

			analysis, err := engine.AnalyseQuery(nil, statements[0])

			require.Error(t, err)
			assert.Nil(t, analysis)
			assert.NotContains(t, err.Error(), "panic")
			if testCase.wantSentinel != nil {
				assert.ErrorIs(t, err, testCase.wantSentinel)
				return
			}
			assert.Contains(t, err.Error(), testCase.wantError)
		})
	}
}

func TestApplyDDLRecoversFromHandlerPanic(t *testing.T) {
	original := ddlHandlers[statementKindCreateTable]
	ddlHandlers[statementKindCreateTable] = func(*parser, *SQLiteEngine) (*querier_dto.CatalogueMutation, error) {
		panic("handler bug")
	}
	t.Cleanup(func() { ddlHandlers[statementKindCreateTable] = original })

	engine := NewSQLiteEngine()
	statements, err := engine.ParseStatements("CREATE TABLE t (a INTEGER)")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])

	require.Error(t, err)
	assert.Nil(t, mutation)
	assert.Equal(t, "sqlite: ddl panic: handler bug", err.Error())
}

func TestAnalyseQueryRecoversFromAnalyserPanic(t *testing.T) {
	original := queryAnalysers[statementKindSelect]
	queryAnalysers[statementKindSelect] = func(*parser) (*querier_dto.RawQueryAnalysis, error) {
		panic("analyser bug")
	}
	t.Cleanup(func() { queryAnalysers[statementKindSelect] = original })

	engine := NewSQLiteEngine()
	statements, err := engine.ParseStatements("SELECT 1")
	require.NoError(t, err)

	analysis, err := engine.AnalyseQuery(nil, statements[0])

	require.Error(t, err)
	assert.Nil(t, analysis)
	assert.Equal(t, "sqlite: analyse panic: analyser bug", err.Error())
}

func TestWithMaxTokensPerStatement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "positive limit is applied", limit: 500, want: 500},
		{name: "zero keeps the default", limit: 0, want: defaultMaxTokensPerStatement},
		{name: "negative keeps the default", limit: -3, want: defaultMaxTokensPerStatement},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewSQLiteEngine(WithMaxTokensPerStatement(testCase.limit))

			assert.Equal(t, testCase.want, engine.dialect.resolvedMaxTokensPerStatement())
		})
	}
}

func TestTokenBudgetRejectsWalkedStatements(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		sql   string
		isDDL bool
	}{
		{name: "DDL statement over budget", sql: "CREATE TABLE t (a INTEGER, b TEXT, c BLOB)", isDDL: true},
		{name: "query over budget", sql: "SELECT a, b, c FROM t WHERE a = ?", isDDL: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewSQLiteEngine(WithMaxTokensPerStatement(5))
			statements, err := engine.ParseStatements(testCase.sql)
			require.NoError(t, err)

			if testCase.isDDL {
				_, err = engine.ApplyDDL(context.Background(), statements[0])
			} else {
				_, err = engine.AnalyseQuery(nil, statements[0])
			}

			require.ErrorIs(t, err, errTokenBudgetExceeded)
			assert.Contains(t, err.Error(), "exceeds the limit of 5")
		})
	}
}

func TestTokenBudgetIgnoresStatementsTheParserDoesNotWalk(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine(WithMaxTokensPerStatement(5))
	statements, err := engine.ParseStatements("INSERT INTO t (a, b, c) VALUES (1, 2, 3); PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.Len(t, statements, 2)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	assert.Nil(t, mutation)

	analysis, err := engine.AnalyseQuery(nil, statements[1])
	require.NoError(t, err)
	assert.Equal(t, &querier_dto.RawQueryAnalysis{}, analysis)
}

func TestTokenBudgetRejectsDeeplyNestedCTEsByDefault(t *testing.T) {
	t.Parallel()

	const nesting = 100_000
	engine := NewSQLiteEngine()
	sql := strings.Repeat("WITH a AS (", nesting) + "SELECT 1" + strings.Repeat(") SELECT 1", nesting)
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	_, err = engine.AnalyseQuery(nil, statements[0])

	require.ErrorIs(t, err, errTokenBudgetExceeded)
}

func TestAnalyseQueryHandlesNestedCTEsWithinBudget(t *testing.T) {
	t.Parallel()

	const nesting = 14_000
	engine := NewSQLiteEngine()
	sql := strings.Repeat("WITH a AS (", nesting) + "SELECT 1" + strings.Repeat(") SELECT 1", nesting)
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	analysis, err := engine.AnalyseQuery(nil, statements[0])

	require.NoError(t, err)
	require.NotNil(t, analysis)
	assert.Len(t, analysis.CTEDefinitions, 1)
}

func TestParseStatementsRecordsPerStatementByteLength(t *testing.T) {
	t.Parallel()

	const first = "SELECT 1"
	const second = "SELECT 22"
	sql := first + "; " + second

	engine := NewSQLiteEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 2)

	assert.Equal(t, 0, statements[0].Location)
	assert.Equal(t, len(first), statements[0].Length,
		"first statement Length should span only its own tokens, not the whole source")

	secondLocation := len(first) + len("; ")
	assert.Equal(t, secondLocation, statements[1].Location)
	assert.Equal(t, len(second), statements[1].Length,
		"second statement Length should span only its own tokens")
}

var (
	sampleDDLStatements = []string{
		"CREATE TABLE IF NOT EXISTS main.t (a INTEGER PRIMARY KEY AUTOINCREMENT, b TEXT NOT NULL DEFAULT (lower('x')) " +
			"CHECK (length(b) > 0) REFERENCES u(id) ON DELETE CASCADE, c TEXT COLLATE NOCASE CONSTRAINT n " +
			"GENERATED ALWAYS AS (b || 'x') STORED, CONSTRAINT k UNIQUE (b), FOREIGN KEY (c) REFERENCES u(id) " +
			"ON UPDATE SET NULL, CHECK (a > 0)) STRICT, WITHOUT ROWID",
		"DROP TABLE IF EXISTS main.t",
		"ALTER TABLE t RENAME COLUMN a TO b",
		"ALTER TABLE t ADD COLUMN d INTEGER DEFAULT 0 REFERENCES u(id)",
		"ALTER TABLE t RENAME TO u",
		"ALTER TABLE t DROP COLUMN d",
		"CREATE VIEW IF NOT EXISTS v (x) AS SELECT a FROM t",
		"DROP VIEW IF EXISTS v",
		"CREATE UNIQUE INDEX IF NOT EXISTS i ON t (a, b) WHERE a > 0",
		"DROP INDEX IF EXISTS i",
		"CREATE VIRTUAL TABLE f USING fts5(a, b, tokenize = 'porter')",
		"CREATE TRIGGER IF NOT EXISTS tr AFTER INSERT ON t BEGIN UPDATE t SET a = 1; END",
		"DROP TRIGGER IF EXISTS tr",
	}
	sampleQueryStatements = []string{
		"WITH c AS (SELECT a FROM t) SELECT DISTINCT a, CAST(? AS VARCHAR(10)) FROM c JOIN u USING (id) " +
			"WHERE a IN (SELECT a FROM t) AND EXISTS (SELECT 1 FROM (SELECT 1) d) GROUP BY a HAVING count(*) > 1 " +
			"UNION SELECT 1 ORDER BY 1 LIMIT ? OFFSET ?",
		"INSERT INTO t (a, b) VALUES (?, ?) ON CONFLICT (a) WHERE a > 0 DO UPDATE SET b = excluded.b WHERE a > 0 RETURNING a",
		"UPDATE OR IGNORE t SET a = ? FROM u WHERE t.id = u.id RETURNING a",
		"DELETE FROM t WHERE a = ? RETURNING a",
		"VALUES (1, ?)",
	}
)

func TestApplyDDLReturnsErrorsForTruncatedAndMismatchedStatements(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()
	for kind, handler := range ddlHandlers {
		if handler == nil {
			continue
		}
		for _, sql := range sampleDDLStatements {
			for _, tokens := range statementPrefixes(t, sql) {
				statement := querier_dto.ParsedStatement{Raw: &parsedStatement{tokens: tokens, kind: statementKind(kind)}}

				_, err := engine.ApplyDDL(context.Background(), statement)

				if err != nil {
					assert.NotContains(t, err.Error(), "panic", "kind %d over %d tokens of %q", kind, len(tokens), sql)
				}
			}
		}
	}
}

func TestAnalyseQueryReturnsErrorsForTruncatedAndMismatchedStatements(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()
	for kind, analyser := range queryAnalysers {
		if analyser == nil {
			continue
		}
		for _, sql := range append(sampleQueryStatements, sampleDDLStatements...) {
			for _, tokens := range statementPrefixes(t, sql) {
				statement := querier_dto.ParsedStatement{Raw: &parsedStatement{tokens: tokens, kind: statementKind(kind)}}

				_, err := engine.AnalyseQuery(nil, statement)

				if err != nil {
					assert.NotContains(t, err.Error(), "panic", "kind %d over %d tokens of %q", kind, len(tokens), sql)
				}
			}
		}
	}
}

func statementPrefixes(t *testing.T, sql string) [][]token {
	t.Helper()

	tokens, err := tokenise(sql)
	require.NoError(t, err)
	statements := splitStatements(tokens)
	require.NotEmpty(t, statements)
	whole := statements[0]

	prefixes := make([][]token, 0, len(whole)+1)
	for length := range len(whole) + 1 {
		prefixes = append(prefixes, whole[:length:length])
	}
	return prefixes
}
