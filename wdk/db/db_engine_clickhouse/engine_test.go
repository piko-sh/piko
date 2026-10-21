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

package db_engine_clickhouse

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/db"
)

func TestWithJoinUseNullsMakesOuterJoinColumnsPointers(t *testing.T) {
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

	testCases := []struct {
		name     string
		options  []Option
		expected string
	}{
		{name: "server default", options: nil, expected: "Title string"},
		{name: "join_use_nulls off", options: []Option{WithJoinUseNulls(false)}, expected: "Title string"},
		{name: "join_use_nulls on", options: []Option{WithJoinUseNulls(true)}, expected: "Title *string"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := NewClickHouseEngine(testCase.options...).Toolchain().Generate(context.Background(), db.GenerateRequest{
				Files:              files,
				PackageName:        "generated",
				MigrationDirectory: "migrations",
				QueryDirectory:     "queries",
				TypeOverrides:      nil,
				CustomFunctions:    nil,
				Driver:             db.DriverClickHouseNative,
			})
			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)

			var queries string
			for _, file := range result.Files {
				if file.Name == "books.sql.go" {
					queries = string(file.Content)
				}
			}
			require.Contains(t, strings.Join(strings.Fields(queries), " "), testCase.expected)
		})
	}
}
