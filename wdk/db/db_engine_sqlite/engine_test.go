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

	_, err := engine.AnalyseQuery(nil, querier_dto.ParsedStatement{
		Raw: nil,
	})
	assert.Error(t, err, "AnalyseQuery should return an error for nil Raw")
}

func TestApplyDDL_UnexpectedStatementType(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	_, err := engine.ApplyDDL(context.Background(), querier_dto.ParsedStatement{
		Raw: nil,
	})
	assert.Error(t, err, "ApplyDDL should return an error for nil Raw")
}

func TestAnalyseQuery_RecoversFromParserPanic(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()
	statement := querier_dto.ParsedStatement{
		Raw: &parsedStatement{
			tokens: []token{{kind: tokenEOF}},
			kind:   statementKindSelect,
		},
	}

	analysis, err := engine.AnalyseQuery(nil, statement)

	require.Error(t, err)
	require.Nil(t, analysis)
	assert.Contains(t, err.Error(), "sqlite: analyse panic")
	assert.NotContains(t, err.Error(), "stack:")
}

func TestApplyDDL_RecoversWithStackTrace(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()
	statement := querier_dto.ParsedStatement{
		Raw: &parsedStatement{
			tokens: []token{{kind: tokenEOF}},
			kind:   statementKindCreateTable,
		},
	}

	mutation, err := engine.ApplyDDL(context.Background(), statement)

	require.Error(t, err)
	require.Nil(t, mutation)
	assert.Contains(t, err.Error(), "sqlite: ddl panic")
	assert.NotContains(t, err.Error(), "stack:")
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
