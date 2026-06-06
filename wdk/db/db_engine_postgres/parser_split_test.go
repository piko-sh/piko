// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package db_engine_postgres

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
			name: "do_block_with_inner_semicolons",
			sql: `CREATE TABLE accounts (id UUID PRIMARY KEY);
DO $$
BEGIN
    INSERT INTO accounts VALUES (gen_random_uuid());
    INSERT INTO accounts VALUES (gen_random_uuid());
END;
$$;
SELECT 1;`,
			want:    3,
			contain: []string{"CREATE TABLE", "DO", "SELECT 1"},
		},
		{
			name: "bare_begin_is_transaction_marker",
			sql:  `BEGIN; SELECT 1; COMMIT;`,
			want: 3,
		},
		{

			name:    "begin_as_column_alias_does_not_merge",
			sql:     `SELECT 1 AS begin; SELECT 2;`,
			want:    2,
			contain: []string{"begin", "SELECT 2"},
		},
		{
			name:    "begin_as_column_name_does_not_merge",
			sql:     `SELECT begin FROM events; CREATE TABLE x (id INT);`,
			want:    2,
			contain: []string{"begin", "CREATE TABLE"},
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
