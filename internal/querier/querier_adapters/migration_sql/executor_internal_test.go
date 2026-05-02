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
	"fmt"
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

func TestSplitStatements_CaseEndInsideTriggerBodyDoesNotSplit(t *testing.T) {
	t.Parallel()

	body := "CREATE TRIGGER t AFTER INSERT ON x BEGIN " +
		"UPDATE x SET v = CASE WHEN new.a > 0 THEN 1 ELSE 2 END; " +
		"END; SELECT 1;"
	got, err := splitStatements(body)

	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Contains(t, got[0], "CREATE TRIGGER")
	require.Contains(t, got[0], "END")
	require.Equal(t, "SELECT 1", got[1])
}

func TestSplitStatements_LeadingWhitespaceThenStandaloneBegin(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("\n\n   BEGIN;\nSELECT 1;\nCOMMIT;")

	require.NoError(t, err)
	require.Equal(t, []string{"BEGIN", "SELECT 1", "COMMIT"}, got)
}

func TestSplitStatements_BackslashEscapeOnlyHonouredWhenEnabled(t *testing.T) {
	t.Parallel()

	withEscapes, err := splitStatementsWithOptions(`INSERT INTO t VALUES ('a\'; b'); SELECT 1;`, true)
	require.NoError(t, err)
	require.Len(t, withEscapes, 2)
	require.Equal(t, `INSERT INTO t VALUES ('a\'; b')`, withEscapes[0])
	require.Equal(t, "SELECT 1", withEscapes[1])
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

func TestSplitStatements_HandlesSQLiteTriggerBody(t *testing.T) {
	t.Parallel()

	sql := `
CREATE TABLE accounts (id TEXT PRIMARY KEY);

CREATE TRIGGER tr_accounts_disallow_modifications
    BEFORE UPDATE ON accounts
    FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'Cannot UPDATE operation on table accounts');
END;

CREATE TRIGGER tr_accounts_disallow_deletes
    BEFORE DELETE ON accounts
    FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'Cannot DELETE operation on table accounts');
END;

INSERT INTO accounts (id) VALUES ('seed-1');`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 4, "should split into table-create + two triggers + insert")
	require.Contains(t, got[0], "CREATE TABLE accounts")
	require.Contains(t, got[1], "BEGIN")
	require.Contains(t, got[1], "END")
	require.Contains(t, got[1], "Cannot UPDATE")
	require.Contains(t, got[2], "Cannot DELETE")
	require.Contains(t, got[3], "INSERT INTO accounts")
}

func TestSplitStatements_HandlesNestedBlocks(t *testing.T) {
	t.Parallel()

	sql := `
CREATE TRIGGER tr_nested
    BEFORE INSERT ON foo
BEGIN
    BEGIN
        SELECT 1;
    END;
    SELECT 2;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1, "nested BEGIN..END must keep the outer statement intact")
	require.Contains(t, got[0], "CREATE TRIGGER tr_nested")
}

func TestSplitStatements_BareBeginIsTransaction(t *testing.T) {
	t.Parallel()

	sql := `BEGIN;
SELECT 1;
COMMIT;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Equal(t, []string{"BEGIN", "SELECT 1", "COMMIT"}, got)
}

func TestSplitStatements_BeginEndCaseInsensitive(t *testing.T) {
	t.Parallel()

	sql := `CREATE TRIGGER t
    BEFORE INSERT ON foo
begin
    select raise(abort, 'no');
End;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "CREATE TRIGGER t")
}

func TestSplitStatements_EndInsideIdentifierIsSafe(t *testing.T) {
	t.Parallel()

	sql := `CREATE TRIGGER t
    BEFORE INSERT ON endpoints
BEGIN
    SELECT * FROM endpoints WHERE id = NEW.endpoint_id;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1, "END as substring of an identifier must not close the block")
}

func TestSplitStatements_TriggerBodyWithCaseExpression(t *testing.T) {
	t.Parallel()

	sql := `CREATE TRIGGER tr_case
    BEFORE UPDATE ON x
BEGIN
    UPDATE y SET v = CASE WHEN x.id = 1 THEN 1 ELSE 2 END FROM x;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1, "CASE..END FROM inside trigger body must not close BEGIN..END")
	require.Contains(t, got[0], "CREATE TRIGGER tr_case")
	require.Contains(t, got[0], "ELSE 2 END FROM x")
}

func TestSplitStatements_TriggerBodyWithEndIf(t *testing.T) {
	t.Parallel()

	sql := `CREATE TRIGGER tr_if
    BEFORE UPDATE ON x
BEGIN
    IF NEW.value = 1 THEN
        SELECT 1;
    END IF;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "END IF")
}

func TestSplitStatements_TriggerBodyWithEndLoop(t *testing.T) {
	t.Parallel()

	sql := `CREATE PROCEDURE p()
BEGIN
    LOOP
        SET x = x + 1;
    END LOOP;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "END LOOP")
}

func TestSplitStatements_TriggerBodyWithEndCase(t *testing.T) {
	t.Parallel()

	sql := `CREATE PROCEDURE p()
BEGIN
    CASE x
        WHEN 1 THEN SELECT 'one';
        ELSE SELECT 'other';
    END CASE;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "END CASE")
}

func TestSplitStatements_TriggerBodyWithEndWhile(t *testing.T) {
	t.Parallel()

	sql := `CREATE PROCEDURE p()
BEGIN
    WHILE x < 10 DO
        SET x = x + 1;
    END WHILE;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "END WHILE")
}

func TestSplitStatements_TriggerBodyWithEndRepeat(t *testing.T) {
	t.Parallel()

	sql := `CREATE PROCEDURE p()
BEGIN
    REPEAT
        SET x = x + 1;
    UNTIL x > 10 END REPEAT;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "END REPEAT")
}

func TestSplitStatements_StandaloneEndStillClosesBlock(t *testing.T) {
	t.Parallel()

	sql := `CREATE TRIGGER t BEFORE INSERT ON foo BEGIN SELECT 1; END;
SELECT 2;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Equal(t, []string{
		"CREATE TRIGGER t BEFORE INSERT ON foo BEGIN SELECT 1; END",
		"SELECT 2",
	}, got)
}

func TestSplitStatements_NestedEndIfInsideOuterBegin(t *testing.T) {
	t.Parallel()

	sql := `CREATE PROCEDURE p()
BEGIN
    IF a THEN
        IF b THEN
            SELECT 1;
        END IF;
    END IF;
    SELECT 2;
END;`

	got, err := splitStatements(sql)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "END IF;\n    END IF")
}

func TestSplitStatements_LeadingLineCommentBeforeStandaloneBegin(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("-- wrap the next change in a transaction\nBEGIN;\nSELECT 1;\nCOMMIT;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"-- wrap the next change in a transaction\nBEGIN",
		"SELECT 1",
		"COMMIT",
	}, got)
}

func TestSplitStatements_LeadingBlockCommentBeforeStandaloneBegin(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("/* migrate users table */ BEGIN; SELECT 1; COMMIT;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"/* migrate users table */ BEGIN",
		"SELECT 1",
		"COMMIT",
	}, got)
}

func TestSplitStatements_RejectsUnbalancedBegin(t *testing.T) {
	t.Parallel()

	_, err := splitStatements("CREATE TRIGGER t BEFORE INSERT ON foo BEGIN SELECT 1; SELECT 2;")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrMalformedSQLStatement)
}

func TestSplitStatements_RejectsUnbalancedCase(t *testing.T) {
	t.Parallel()

	_, err := splitStatements("CREATE PROCEDURE p() BEGIN UPDATE x SET v = CASE WHEN a THEN 1; END;")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrMalformedSQLStatement)
}

func TestSplitStatements_BalancedBeginEndDoesNotErrorAtEOF(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("CREATE TRIGGER t BEFORE INSERT ON foo BEGIN SELECT 1; END")

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "CREATE TRIGGER t")
}

func TestSplitStatements_DollarOneDollarIsParameterNotTag(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("UPDATE t SET a = $1, b = $1 + 1; SELECT 2;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"UPDATE t SET a = $1, b = $1 + 1",
		"SELECT 2",
	}, got)
}

func TestSplitStatements_NamedDollarTagStillOpensBlock(t *testing.T) {
	t.Parallel()

	got, err := splitStatements("CREATE FUNCTION f() AS $tag1$ SELECT 1; $tag1$; SELECT 2;")

	require.NoError(t, err)
	require.Equal(t, []string{
		"CREATE FUNCTION f() AS $tag1$ SELECT 1; $tag1$",
		"SELECT 2",
	}, got)
}

func TestReadDollarQuoteTag_RejectsDigitLeadingTag(t *testing.T) {
	t.Parallel()

	_, _, ok := readDollarQuoteTag([]rune("$1$"), 0)
	require.False(t, ok, "a digit-leading tag must not be treated as a dollar-quote opener")
}

func TestReadDollarQuoteTag_AcceptsEmptyAndLetterLeadingTags(t *testing.T) {
	t.Parallel()

	tag, advance, ok := readDollarQuoteTag([]rune("$$body$$"), 0)
	require.True(t, ok)
	require.Empty(t, tag)
	require.Equal(t, 2, advance)

	tag, advance, ok = readDollarQuoteTag([]rune("$body$x$body$"), 0)
	require.True(t, ok)
	require.Equal(t, "body", tag)
	require.Equal(t, 6, advance)

	tag, advance, ok = readDollarQuoteTag([]rune("$_priv$x$_priv$"), 0)
	require.True(t, ok)
	require.Equal(t, "_priv", tag)
	require.Equal(t, 7, advance)
}

func TestParseAppliedAt_HandlesNativeTimeTime(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)

	require.Equal(t, now, parseAppliedAt(t.Context(), now, 1))
}

func TestParseAppliedAt_HandlesRFC3339(t *testing.T) {
	t.Parallel()

	got := parseAppliedAt(t.Context(), "2026-01-02T15:04:05Z", 1)
	require.Equal(t, 2026, got.Year())
	require.Equal(t, time.January, got.Month())
}

func TestParseAppliedAt_HandlesSQLiteFormat(t *testing.T) {
	t.Parallel()

	got := parseAppliedAt(t.Context(), "2026-05-03 12:34:56", 1)
	require.Equal(t, 2026, got.Year())
	require.Equal(t, time.May, got.Month())
}

func TestParseAppliedAt_HandlesUnixSecondsAsInt64(t *testing.T) {
	t.Parallel()

	got := parseAppliedAt(t.Context(), int64(1700000000), 1)
	require.Equal(t, time.Unix(1700000000, 0).UTC(), got)
}

func TestParseAppliedAt_HandlesUnixSecondsAsFloat64(t *testing.T) {
	t.Parallel()

	got := parseAppliedAt(t.Context(), float64(1700000000), 1)
	require.Equal(t, time.Unix(1700000000, 0).UTC(), got)
}

func TestParseAppliedAt_NilReturnsZeroTime(t *testing.T) {
	t.Parallel()

	require.True(t, parseAppliedAt(t.Context(), nil, 1).IsZero())
}

func TestParseAppliedAt_UnsupportedTypeReturnsZeroTime(t *testing.T) {
	t.Parallel()

	require.True(t, parseAppliedAt(t.Context(), []byte("anything"), 1).IsZero())
}

func TestParseAppliedAt_UnparseableStringReturnsZeroTime(t *testing.T) {
	t.Parallel()

	require.True(t, parseAppliedAt(t.Context(), "not a date", 1).IsZero())
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

type fakePostgresSQLStateError struct {
	sqlState string
}

func (e fakePostgresSQLStateError) Error() string    { return "postgres server error" }
func (e fakePostgresSQLStateError) SQLState() string { return e.sqlState }

func TestIsLockNotAvailableError_DetectsTypedPostgresSQLState(t *testing.T) {
	t.Parallel()

	lockErr := fakePostgresSQLStateError{sqlState: postgresErrorCodeLockNotAvailable}
	require.True(t, isLockNotAvailableError(lockErr), "typed 55P03 must be detected")

	wrapped := fmt.Errorf("acquire migration lock: %w", lockErr)
	require.True(t, isLockNotAvailableError(wrapped), "wrapped typed 55P03 must be detected")

	other := fakePostgresSQLStateError{sqlState: "23505"}
	require.False(t, isLockNotAvailableError(other), "unrelated SQLSTATE must not match")
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
