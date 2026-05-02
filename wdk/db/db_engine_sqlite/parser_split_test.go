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
			name: "trigger_body_with_inner_semicolon",
			sql: `CREATE TABLE accounts (id TEXT);
CREATE TRIGGER tr_no_update
    BEFORE UPDATE ON accounts
    FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'no');
END;
INSERT INTO accounts VALUES ('a');`,
			want:    3,
			contain: []string{"CREATE TABLE", "CREATE TRIGGER", "INSERT"},
		},
		{
			name: "multi_statement_trigger_body",
			sql: `CREATE TRIGGER tr_log
    AFTER INSERT ON accounts
BEGIN
    INSERT INTO log VALUES (NEW.id);
    INSERT INTO log VALUES (NEW.id);
END;`,
			want:    1,
			contain: []string{"CREATE TRIGGER"},
		},
		{
			name: "bare_begin_is_transaction_marker",
			sql:  `BEGIN; SELECT 1; COMMIT;`,
			want: 3,
		},
		{
			name:    "begin_as_column_alias_does_not_swallow_next_statement",
			sql:     `SELECT 1 AS begin; SELECT 2;`,
			want:    2,
			contain: []string{"SELECT 1 AS begin", "SELECT 2"},
		},
		{
			name:    "begin_as_identifier_in_non_procedural_statement",
			sql:     `SELECT begin FROM t; SELECT 2;`,
			want:    2,
			contain: []string{"SELECT begin FROM t", "SELECT 2"},
		},
		{
			name: "case_insensitive_begin_end",
			sql: `CREATE TRIGGER t
    BEFORE INSERT ON foo
begin
    select raise(abort, 'no');
End;`,
			want: 1,
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
