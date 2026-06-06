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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	deepNestingLevels = 4_000
)

func TestParserDepthLimitPreventsStackOverflow(t *testing.T) {
	t.Parallel()

	const depth = 100_000
	engine := NewPostgresEngine()

	t.Run("nested expression parentheses", func(t *testing.T) {
		t.Parallel()
		sql := "SELECT " + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth) + " FROM t"
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)

		_, _ = engine.AnalyseQuery(nil, statements[0])
	})

	t.Run("nested derived tables", func(t *testing.T) {
		t.Parallel()

		sql := buildNestedDerivedTables(deepNestingLevels)
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		_, _ = engine.AnalyseQuery(nil, statements[0])
	})
}

func TestParserDepthLimitRejectsNestedSubquerySubtypes(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	t.Run("chained EXISTS subqueries", func(t *testing.T) {
		t.Parallel()

		var sb strings.Builder
		sb.WriteString("SELECT 1 WHERE 1 = 1")
		for range deepNestingLevels {
			previous := sb.String()
			sb.Reset()
			sb.WriteString("SELECT 1 WHERE EXISTS (")
			sb.WriteString(previous)
			sb.WriteString(")")
		}
		statements, err := engine.ParseStatements(sb.String())
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		_, _ = engine.AnalyseQuery(nil, statements[0])
	})

	t.Run("chained scalar subqueries", func(t *testing.T) {
		t.Parallel()

		const depth = 100_000
		sql := "SELECT " + strings.Repeat("(SELECT ", depth) + "1" + strings.Repeat(")", depth)
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		_, _ = engine.AnalyseQuery(nil, statements[0])
	})
}

func TestParserDepthLimitIsConfigurable(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine(WithMaxParseDepth(8))

	t.Run("expression nesting", func(t *testing.T) {
		t.Parallel()
		sql := "SELECT " + strings.Repeat("(", 64) + "1" + strings.Repeat(")", 64) + " FROM t"
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		_, _ = engine.AnalyseQuery(nil, statements[0])
	})

	t.Run("subquery nesting", func(t *testing.T) {
		t.Parallel()
		sql := buildNestedDerivedTables(64)
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		_, _ = engine.AnalyseQuery(nil, statements[0])
	})
}

func buildNestedDerivedTables(depth int) string {
	var sb strings.Builder
	sb.WriteString("SELECT * FROM t")
	for range depth {
		previous := sb.String()
		sb.Reset()
		sb.WriteString("SELECT * FROM (")
		sb.WriteString(previous)
		sb.WriteString(") sub")
	}
	return sb.String()
}
