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

package migration_sql

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSplitStatements_SplitsAndTrims(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("CREATE TABLE foo (id INT);  INSERT INTO foo VALUES (1);  ;\n  ")

	require.NoError(t, err)
	require.Equal(t, []string{
		"CREATE TABLE foo (id INT)",
		"INSERT INTO foo VALUES (1)",
	}, got)
}

func TestSplitStatements_EmptyInputReturnsEmpty(t *testing.T) {
	t.Parallel()

	empty, err := splitStatements("")
	require.NoError(t, err)
	require.Empty(t, empty)

	emptyWithSemicolons, err := splitStatements(";   ;\n;")
	require.NoError(t, err)
	require.Empty(t, emptyWithSemicolons)
}

func TestSplitStatements_SingleStatementWithoutTrailingSemicolon(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("SELECT 1")

	require.NoError(t, err)
	require.Equal(t, []string{"SELECT 1"}, got)
}

func TestSplitStatements_HandlesStringLiteralsWithSemicolons(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("INSERT INTO t VALUES ('a;b'); SELECT 1;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"INSERT INTO t VALUES ('a;b')",
		"SELECT 1",
	}, got)
}

func TestSplitStatements_HandlesEscapedQuotesInStringLiterals(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("INSERT INTO t VALUES ('it''s; tricky'); SELECT 1;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"INSERT INTO t VALUES ('it''s; tricky')",
		"SELECT 1",
	}, got)
}

func TestSplitStatements_HandlesDollarQuotedBlocks(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("CREATE FUNCTION f() RETURNS INT AS $$ BEGIN RETURN 1; END $$; SELECT 1;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"CREATE FUNCTION f() RETURNS INT AS $$ BEGIN RETURN 1; END $$",
		"SELECT 1",
	}, got)
}

func TestSplitStatements_HandlesTaggedDollarQuotes(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("CREATE FUNCTION f() AS $body$ DECLARE x INT; BEGIN x := 1; END $body$; SELECT 2;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"CREATE FUNCTION f() AS $body$ DECLARE x INT; BEGIN x := 1; END $body$",
		"SELECT 2",
	}, got)
}

func TestSplitStatements_SkipsLineComments(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("-- comment with semicolon ; here\nSELECT 1; SELECT 2;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"-- comment with semicolon ; here\nSELECT 1",
		"SELECT 2",
	}, got)
}

func TestSplitStatements_SkipsBlockComments(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("/* skip; me; */ SELECT 1; SELECT 2;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"/* skip; me; */ SELECT 1",
		"SELECT 2",
	}, got)
}

func TestSplitStatements_RejectsUnterminatedString(t *testing.T) {
	t.Parallel()

	_, err := splitStatements("SELECT 'unterminated")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrMalformedSQLStatement)
}

func TestSplitStatements_RejectsUnterminatedDollarQuote(t *testing.T) {
	t.Parallel()

	_, err := splitStatements("CREATE FUNCTION f() AS $body$ BEGIN RETURN 1; END")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrMalformedSQLStatement)
}

func TestSplitStatements_RejectsUnterminatedBlockComment(t *testing.T) {
	t.Parallel()

	_, err := splitStatements("SELECT 1; /* never closed")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrMalformedSQLStatement)
}

func TestParseAppliedAt_HandlesNativeTimeTime(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)

	require.Equal(t, now, parseAppliedAt(now))
}

func TestParseAppliedAt_HandlesRFC3339(t *testing.T) {
	t.Parallel()

	got := parseAppliedAt("2026-01-02T15:04:05Z")
	require.Equal(t, 2026, got.Year())
	require.Equal(t, time.January, got.Month())
}

func TestParseAppliedAt_HandlesSQLiteFormat(t *testing.T) {
	t.Parallel()

	got := parseAppliedAt("2026-05-03 12:34:56")
	require.Equal(t, 2026, got.Year())
	require.Equal(t, time.May, got.Month())
}

func TestParseAppliedAt_HandlesUnixSecondsAsInt64(t *testing.T) {
	t.Parallel()

	got := parseAppliedAt(int64(1700000000))
	require.Equal(t, time.Unix(1700000000, 0).UTC(), got)
}

func TestParseAppliedAt_HandlesUnixSecondsAsFloat64(t *testing.T) {
	t.Parallel()

	got := parseAppliedAt(float64(1700000000))
	require.Equal(t, time.Unix(1700000000, 0).UTC(), got)
}

func TestParseAppliedAt_NilReturnsZeroTime(t *testing.T) {
	t.Parallel()

	require.True(t, parseAppliedAt(nil).IsZero())
}

func TestParseAppliedAt_UnsupportedTypeReturnsZeroTime(t *testing.T) {
	t.Parallel()

	require.True(t, parseAppliedAt([]byte("anything")).IsZero())
}

func TestParseAppliedAt_UnparseableStringReturnsZeroTime(t *testing.T) {
	t.Parallel()

	require.True(t, parseAppliedAt("not a date").IsZero())
}

func TestIsDuplicateColumnError_ReturnsTrueForKnownPhrases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("duplicate column name: dirty"), true},
		{errors.New("DUPLICATE COLUMN: foo"), true},
		{errors.New("column already exists"), true},
		{errors.New("ALREADY EXISTS"), true},
		{errors.New("syntax error near unknown"), false},
		{errors.New(""), false},
	}

	for _, tc := range cases {
		require.Equalf(t, tc.want, isDuplicateColumnError(tc.err), "error: %q", tc.err.Error())
	}
}

func TestIsLockNotAvailableError_ReturnsTrueForKnownPatterns(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("ERROR: 55P03 lock_not_available"), true},
		{errors.New("ERROR: could not obtain lock"), true},
		{errors.New("ERROR 1205 (HY000): Lock wait timeout exceeded"), true},
		{errors.New("Lock wait timeout exceeded"), true},
		{errors.New("connection refused"), false},
		{errors.New(""), false},
	}

	for _, tc := range cases {
		require.Equalf(t, tc.want, isLockNotAvailableError(tc.err), "error: %q", tc.err.Error())
	}
}

func TestNewExecutor_PreservesDialectConfig(t *testing.T) {
	t.Parallel()

	dialect := SQLiteDialect()

	executor := NewExecutor(nil, dialect)

	require.NotNil(t, executor)
	require.Equal(t, dialect.PlaceholderFunc(1), executor.dialectConfig.PlaceholderFunc(1))
}

func TestNewSeedExecutor_PreservesDialectConfig(t *testing.T) {
	t.Parallel()

	dialect := SQLiteDialect()

	executor := NewSeedExecutor(nil, dialect)

	require.NotNil(t, executor)
	require.Equal(t, dialect.PlaceholderFunc(1), executor.dialectConfig.PlaceholderFunc(1))
}

func TestSeedExecutor_EnsureSeedTable_RejectsEmptyDDL(t *testing.T) {
	t.Parallel()

	executor := NewSeedExecutor(nil, DialectConfig{})

	err := executor.EnsureSeedTable(t.Context())

	require.Error(t, err)
	require.Contains(t, err.Error(), "no seed table DDL")
}
