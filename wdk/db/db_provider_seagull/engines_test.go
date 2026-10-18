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

package db_provider_seagull

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/sqldb/sqldb_dto"
)

func TestJoinUseNullsDecidesWhetherOuterJoinColumnsArePointers(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"migrations/001_books.up.sql": {Data: []byte(
			"CREATE TABLE authors (id UInt64, name String) ENGINE = MergeTree() ORDER BY id;\n" +
				"CREATE TABLE books (id UInt64, author_id UInt64, title String) ENGINE = MergeTree() ORDER BY id;\n",
		)},
		"queries/books.sql": {Data: []byte(
			"-- piko.query(ListAuthorsWithBooks, many)\n" +
				"SELECT authors.name, books.title FROM authors LEFT JOIN books ON books.author_id = authors.id;\n",
		)},
	}
	enabled := true
	disabled := false

	testCases := []struct {
		joinUseNulls *bool
		name         string
		expected     string
	}{
		{name: "server default", joinUseNulls: nil, expected: "Title string"},
		{name: "off", joinUseNulls: &disabled, expected: "Title string"},
		{name: "on", joinUseNulls: &enabled, expected: "Title *string"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewBuiltinEngine(DialectOptions{Dialect: "clickhouse", JoinUseNulls: testCase.joinUseNulls})
			result, err := engine.Toolchain().Generate(context.Background(), sqldb_dto.GenerateRequest{
				Files:              files,
				MigrationDirectory: "migrations",
				QueryDirectory:     "queries",
				Driver:             sqldb_dto.DriverDatabaseSQL,
			})
			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)

			queries := generatedFile(t, result, "books.sql.go")
			require.Contains(t, strings.Join(strings.Fields(queries), " "), testCase.expected)
		})
	}
}

func TestJoinUseNullsIsAClickHouseOption(t *testing.T) {
	t.Parallel()

	enabled := true
	engine := NewBuiltinEngine(DialectOptions{Dialect: "postgres", JoinUseNulls: &enabled})

	_, err := engine.Toolchain().Generate(context.Background(), sqldb_dto.GenerateRequest{})

	require.ErrorContains(t, err, "JoinUseNulls applies to clickhouse")
}
