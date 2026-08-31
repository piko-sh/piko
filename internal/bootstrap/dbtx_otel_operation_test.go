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

package bootstrap

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveOperationName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		query string
		want  string
	}{
		{"select", "SELECT id, name FROM artefacts WHERE id = ?", "SELECT artefacts"},
		{"select lowercase", "select * from tasks order by id", "SELECT tasks"},
		{"select schema qualified", "SELECT * FROM public.artefacts", "SELECT artefacts"},
		{"select quoted", `SELECT * FROM "artefacts" WHERE x = 1`, "SELECT artefacts"},
		{"select backticked", "SELECT * FROM `tasks`", "SELECT tasks"},
		{"insert", "INSERT INTO artefacts (id) VALUES (?)", "INSERT artefacts"},
		{"update", "UPDATE tasks SET status = ? WHERE id = ?", "UPDATE tasks"},
		{"delete", "DELETE FROM tasks WHERE id = ?", "DELETE tasks"},
		{"replace", "REPLACE INTO cache (k) VALUES (?)", "REPLACE cache"},
		{"multiline", "SELECT\n  id\nFROM\n  artefacts", "SELECT artefacts"},
		{"subquery still names the outer table", "SELECT * FROM a WHERE id IN (SELECT id FROM b)", "SELECT a"},
		{"verb only when no table found", "SELECT 1", "SELECT"},
		{"unrecognised verb", "PRAGMA journal_mode=WAL", "UNKNOWN"},
		{"empty", "", "UNKNOWN"},
		{"whitespace", "   \n\t ", "UNKNOWN"},
		{
			name:  "sub-query in the FROM clause names the table it reads",
			query: "SELECT * FROM (SELECT id FROM b) AS t",
			want:  "SELECT b",
		},
		{
			name:  "common table expression",
			query: "WITH x AS (SELECT 1) SELECT * FROM x",
			want:  "WITH x",
		},
		{
			name:  "leading block comment does not hide the verb",
			query: "/* traceparent=00-abc */ SELECT * FROM artefacts",
			want:  "SELECT artefacts",
		},
		{
			name:  "unterminated leading comment is left alone",
			query: "/* never closed SELECT * FROM artefacts",
			want:  "UNKNOWN",
		},
		{
			name:  "postgres ONLY qualifier is stepped over",
			query: "SELECT * FROM ONLY tbl",
			want:  "SELECT tbl",
		},
		{
			name:  "explain names the explained table",
			query: "EXPLAIN SELECT * FROM t",
			want:  "EXPLAIN t",
		},
		{
			name:  "values list is not a table",
			query: "SELECT * FROM (VALUES (1)) AS t",
			want:  "SELECT",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, deriveOperationName(tc.query))
		})
	}
}

func TestDeriveOperationName_NeverEchoesValues(t *testing.T) {
	t.Parallel()

	require.Equal(t, "SELECT artefacts",
		deriveOperationName("SELECT * FROM artefacts WHERE secret = 'hunter2'"))
}

func TestResolveOperation_StopsMintingLabelsPastTheBound(t *testing.T) {
	t.Parallel()

	wrapper := newOTelDBTX(nil, "sqlite", "test", nil, nil)

	for index := range maxOperationLabels {
		label := wrapper.resolveOperation(fmt.Sprintf("SELECT * FROM events_%d", index))
		require.Equal(t, fmt.Sprintf("SELECT events_%d", index), label)
	}

	assert.Equal(t, "SELECT", wrapper.resolveOperation("SELECT * FROM one_table_too_many"),
		"past the bound the verb alone is reported, which cannot grow")
}

func TestResolveOperation_DerivesEachStatementOnce(t *testing.T) {
	t.Parallel()

	wrapper := newOTelDBTX(nil, "sqlite", "test", nil, nil)
	const query = "SELECT * FROM artefacts WHERE id = ?"

	require.Equal(t, "SELECT artefacts", wrapper.resolveOperation(query))
	require.Equal(t, "SELECT artefacts", wrapper.resolveOperation(query))

	assert.Equal(t, int64(1), wrapper.operationLabels.count.Load(),
		"a repeated statement reuses its label rather than deriving another")
}

func TestWrapDBTX_SharesTheLabelBudgetWithItsParent(t *testing.T) {
	t.Parallel()

	wrapper := newOTelDBTX(nil, "sqlite", "test", nil, nil)
	require.Equal(t, "SELECT artefacts", wrapper.resolveOperation("SELECT * FROM artefacts"))

	clone, ok := wrapper.WrapDBTX(&stubDBTX{}).(*otelDBTX)
	require.True(t, ok, "a DBTX is wrapped")

	assert.Same(t, wrapper.operationLabels, clone.operationLabels)
	assert.Same(t, wrapper.reportOnce, clone.reportOnce)
}

func TestDeriveOperationName_SkipsLeadingComments(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"line comment":            "-- name: GetArtefact :one\nSELECT * FROM artefacts",
		"block comment":           "/* tracer */ SELECT * FROM artefacts",
		"both, stacked":           "-- name: GetArtefact :one\n/* tracer */\nSELECT * FROM artefacts",
		"comment with no newline": "-- only a comment",
	}

	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			label := deriveOperationName(query)
			if name == "comment with no newline" {
				assert.Equal(t, "UNKNOWN", label)

				return
			}
			assert.Equal(t, "SELECT artefacts", label)
		})
	}
}

func TestDeriveOperationName_BoundsStackedComments(t *testing.T) {
	t.Parallel()

	query := strings.Repeat("-- comment\n", 10_000) + "SELECT * FROM artefacts"

	assert.Equal(t, "UNKNOWN", deriveOperationName(query),
		"past the comment cap the statement is not parsed rather than scanned without limit")
}

func TestDeriveOperationName_RejectsAnOverlongIdentifier(t *testing.T) {
	t.Parallel()

	query := "SELECT * FROM " + strings.Repeat("a", maxOperationIdentifierLen+1)

	assert.Equal(t, "SELECT", deriveOperationName(query))
}
