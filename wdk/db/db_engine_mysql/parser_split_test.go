// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package db_engine_mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitStatements_BeginEndBlockHandling(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		sql     string
		want    int
		contain []string
	}{
		{
			name: "trigger_without_delimiter",
			sql: `CREATE TABLE accounts (id INT PRIMARY KEY);
CREATE TRIGGER tr_log
    AFTER INSERT ON accounts
    FOR EACH ROW
BEGIN
    INSERT INTO log VALUES (NEW.id);
    INSERT INTO log VALUES (NEW.id);
END;
SELECT 1;`,
			want:    3,
			contain: []string{"CREATE TABLE", "CREATE TRIGGER", "SELECT 1"},
		},
		{
			name: "bare_begin_is_transaction_marker",
			sql:  `BEGIN; SELECT 1; COMMIT;`,
			want: 3,
		},
		{
			name: "stored_procedure_with_end_if",
			sql: `CREATE PROCEDURE p()
BEGIN
    IF x = 1 THEN
        SELECT 1;
    END IF;
END;
SELECT 2;`,
			want:    2,
			contain: []string{"CREATE PROCEDURE", "SELECT 2"},
		},
		{
			name: "stored_procedure_with_end_loop",
			sql: `CREATE PROCEDURE p()
BEGIN
    LOOP
        SET x = x + 1;
    END LOOP;
END;`,
			want:    1,
			contain: []string{"CREATE PROCEDURE"},
		},
		{
			name: "stored_procedure_with_end_while",
			sql: `CREATE PROCEDURE p()
BEGIN
    WHILE x < 10 DO
        SET x = x + 1;
    END WHILE;
END;`,
			want:    1,
			contain: []string{"CREATE PROCEDURE"},
		},
		{
			name: "stored_procedure_with_end_repeat",
			sql: `CREATE PROCEDURE p()
BEGIN
    REPEAT
        SET x = x + 1;
    UNTIL x > 10 END REPEAT;
END;`,
			want:    1,
			contain: []string{"CREATE PROCEDURE"},
		},
		{
			name: "stored_procedure_with_end_case",
			sql: `CREATE PROCEDURE p()
BEGIN
    CASE x
        WHEN 1 THEN SELECT 'one';
        ELSE SELECT 'other';
    END CASE;
END;`,
			want:    1,
			contain: []string{"CREATE PROCEDURE"},
		},
		{

			name:    "begin_used_as_alias_does_not_open_block",
			sql:     `SELECT 1 AS begin; SELECT 2;`,
			want:    2,
			contain: []string{"SELECT 1", "SELECT 2"},
		},
		{

			name:    "begin_used_as_identifier_does_not_open_block",
			sql:     `SELECT begin FROM t; SELECT 2;`,
			want:    2,
			contain: []string{"SELECT begin FROM t", "SELECT 2"},
		},
		{

			name: "scalar_case_in_body_does_not_close_block_early",
			sql: `CREATE PROCEDURE p()
BEGIN
    SET y = CASE x WHEN 1 THEN 2 ELSE 3 END;
    INSERT INTO log VALUES (y);
END;
SELECT 9;`,
			want:    2,
			contain: []string{"CREATE PROCEDURE", "SELECT 9"},
		},
		{

			name: "scalar_case_in_function_return",
			sql: `CREATE FUNCTION f(x INT) RETURNS INT
BEGIN
    RETURN CASE x WHEN 1 THEN 10 ELSE 20 END;
END;
SELECT 7;`,
			want:    2,
			contain: []string{"CREATE FUNCTION", "SELECT 7"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tokens, err := tokenise(tc.sql)
			require.NoError(t, err)
			statements := splitStatements(tokens)
			require.Len(t, statements, tc.want)
			for i, want := range tc.contain {
				if i >= len(statements) {
					break
				}
				var buf strings.Builder
				for _, tok := range statements[i] {
					buf.WriteString(tok.value)
					buf.WriteRune(' ')
				}
				require.Contains(t, buf.String(), want)
			}
		})
	}
}
