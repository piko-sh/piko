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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParserDepthLimitPreventsStackOverflow(t *testing.T) {
	t.Parallel()

	const depth = 100_000
	engine := NewClickHouseEngine()

	t.Run("nested expression parentheses", func(t *testing.T) {
		t.Parallel()
		sql := "SELECT " + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth) + " FROM t"
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)

		_, _ = engine.AnalyseQuery(nil, statements[0])
	})

	t.Run("nested subqueries", func(t *testing.T) {
		t.Parallel()

		sql := "SELECT " + strings.Repeat("(SELECT ", depth) + "1" + strings.Repeat(")", depth)
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		_, _ = engine.AnalyseQuery(nil, statements[0])
	})
}

func TestParserDepthLimitIsConfigurable(t *testing.T) {
	t.Parallel()

	const nesting = 64
	sql := "SELECT " + strings.Repeat("(", nesting) + "1" + strings.Repeat(")", nesting) + " FROM t"

	t.Run("small custom cap terminates without crashing", func(t *testing.T) {
		t.Parallel()
		engine := NewClickHouseEngine(WithMaxParseDepth(8))
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		_, _ = engine.AnalyseQuery(nil, statements[0])
	})

	t.Run("default cap admits the same nesting", func(t *testing.T) {
		t.Parallel()
		engine := NewClickHouseEngine()
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		_, _ = engine.AnalyseQuery(nil, statements[0])
	})
}

func TestResolvedMaxParseDepthDefault(t *testing.T) {
	t.Parallel()

	require.Equal(t, defaultMaxParseDepth, ClickHouseDialect{}.resolvedMaxParseDepth())
	require.Equal(t, 32, ClickHouseDialect{MaxParseDepth: 32}.resolvedMaxParseDepth())
}

func TestWithMaxParseDepthIgnoresNonPositive(t *testing.T) {
	t.Parallel()

	dialect := ClickHouseDialect{}
	WithMaxParseDepth(0)(&dialect)
	WithMaxParseDepth(-5)(&dialect)
	require.Equal(t, defaultMaxParseDepth, dialect.resolvedMaxParseDepth())

	WithMaxParseDepth(16)(&dialect)
	require.Equal(t, 16, dialect.resolvedMaxParseDepth())
}
