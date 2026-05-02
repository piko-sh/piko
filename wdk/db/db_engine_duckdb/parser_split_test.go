// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package db_engine_duckdb

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
		contain []string
		want    int
	}{
		{

			name: "create_prefixed_begin_end_block",
			sql: `CREATE OR REPLACE MACRO m() AS BEGIN
    SELECT 1;
    SELECT 2;
END;
SELECT 3;`,
			want:    2,
			contain: []string{"CREATE OR REPLACE MACRO", "SELECT 3"},
		},
		{
			name: "bare_begin_is_transaction_marker",
			sql:  `BEGIN; SELECT 1; COMMIT;`,
			want: 3,
		},
		{

			name:    "begin_alias_does_not_swallow_next_statement",
			sql:     `SELECT 1 AS begin; SELECT 2;`,
			want:    2,
			contain: []string{"SELECT 1 AS begin", "SELECT 2"},
		},
		{

			name:    "begin_alias_quoted_identifier_does_not_open_block",
			sql:     `SELECT 1 AS "begin"; SELECT 2;`,
			want:    2,
			contain: []string{"SELECT 1 AS", "SELECT 2"},
		},
		{

			name: "macro_body_with_scalar_case_keeps_statement_intact",
			sql: `CREATE MACRO m(x) AS BEGIN
    SELECT CASE x WHEN 1 THEN 2 ELSE 3 END;
    SELECT 4;
END;
SELECT 5;`,
			want:    2,
			contain: []string{"CREATE MACRO", "SELECT 5"},
		},
		{
			name: "macro_body_with_end_if",
			sql: `CREATE MACRO m() AS BEGIN
    IF x = 1 THEN
        SELECT 1;
    END IF;
END;
SELECT 2;`,
			want:    2,
			contain: []string{"CREATE MACRO", "SELECT 2"},
		},
		{
			name: "macro_body_with_end_loop",
			sql: `CREATE MACRO m() AS BEGIN
    LOOP
        SELECT 1;
    END LOOP;
END;`,
			want:    1,
			contain: []string{"CREATE MACRO"},
		},
		{
			name: "macro_body_with_end_case",
			sql: `CREATE MACRO m() AS BEGIN
    CASE x
        WHEN 1 THEN SELECT 1;
        ELSE SELECT 0;
    END CASE;
END;`,
			want:    1,
			contain: []string{"CREATE MACRO"},
		},
		{
			name: "macro_body_with_end_while",
			sql: `CREATE MACRO m() AS BEGIN
    WHILE x < 10 DO
        SELECT 1;
    END WHILE;
END;`,
			want:    1,
			contain: []string{"CREATE MACRO"},
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
