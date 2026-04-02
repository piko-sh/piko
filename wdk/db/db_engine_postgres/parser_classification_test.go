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

package db_engine_postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatements_Classification(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	type testCase struct {
		description   string
		sql           string
		expectedCount int
	}

	cases := []testCase{
		{
			description:   "SELECT returns one statement",
			sql:           "SELECT 1",
			expectedCount: 1,
		},
		{
			description:   "INSERT returns one statement",
			sql:           "INSERT INTO t VALUES (1)",
			expectedCount: 1,
		},
		{
			description:   "UPDATE returns one statement",
			sql:           "UPDATE t SET x = 1",
			expectedCount: 1,
		},
		{
			description:   "DELETE returns one statement",
			sql:           "DELETE FROM t",
			expectedCount: 1,
		},
		{
			description:   "CREATE TABLE returns one statement",
			sql:           "CREATE TABLE t (id int)",
			expectedCount: 1,
		},
		{
			description:   "ALTER TABLE returns one statement",
			sql:           "ALTER TABLE t ADD COLUMN x int",
			expectedCount: 1,
		},
		{
			description:   "DROP TABLE returns one statement",
			sql:           "DROP TABLE t",
			expectedCount: 1,
		},
		{
			description:   "CREATE INDEX returns one statement",
			sql:           "CREATE INDEX idx ON t (x)",
			expectedCount: 1,
		},
		{
			description:   "CREATE VIEW returns one statement",
			sql:           "CREATE VIEW v AS SELECT 1",
			expectedCount: 1,
		},
		{
			description:   "WITH CTE returns one statement",
			sql:           "WITH cte AS (SELECT 1) SELECT * FROM cte",
			expectedCount: 1,
		},
		{
			description:   "COMMENT returns one statement",
			sql:           "COMMENT ON TABLE t IS 'test'",
			expectedCount: 1,
		},
		{
			description:   "empty string returns zero statements",
			sql:           "",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			statements, err := engine.ParseStatements(tc.sql)
			require.NoError(t, err)
			assert.Len(t, statements, tc.expectedCount, "statement count")
		})
	}
}

func TestParseStatements_MultipleStatements(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	type testCase struct {
		description   string
		sql           string
		expectedCount int
	}

	cases := []testCase{
		{
			description:   "two SELECT statements separated by semicolon",
			sql:           "SELECT 1; SELECT 2",
			expectedCount: 2,
		},
		{
			description:   "CREATE TABLE followed by INSERT",
			sql:           "CREATE TABLE t (id int); INSERT INTO t VALUES (1)",
			expectedCount: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			statements, err := engine.ParseStatements(tc.sql)
			require.NoError(t, err)
			assert.Len(t, statements, tc.expectedCount, "statement count")
		})
	}
}

func TestParseStatements_SyntaxErrors(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	type testCase struct {
		description string
		sql         string
	}

	cases := []testCase{
		{
			description: "unterminated string literal returns error",
			sql:         "SELECT 'unterminated",
		},
		{
			description: "unterminated quoted identifier returns error",
			sql:         `SELECT "unclosed`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			_, err := engine.ParseStatements(tc.sql)
			assert.Error(t, err, "expected a tokenisation error")
		})
	}
}
