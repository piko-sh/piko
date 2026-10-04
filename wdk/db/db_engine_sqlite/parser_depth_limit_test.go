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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserDepthLimitPreventsStackOverflow(t *testing.T) {
	t.Parallel()

	const depth = 100_000
	engine := NewSQLiteEngine(WithMaxTokensPerStatement(4 * depth))

	sql := "SELECT " + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth) + " FROM t"
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.NotEmpty(t, statements)

	analysis, err := engine.AnalyseQuery(nil, statements[0])

	require.ErrorIs(t, err, errExpressionDepthExceeded)
	assert.Nil(t, analysis)
}

func TestParserDepthLimitIsConfigurable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		nesting  int
		wantFail bool
	}{
		{name: "nesting within the cap analyses", nesting: 4, wantFail: false},
		{name: "nesting past the cap reports the depth error", nesting: 64, wantFail: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewSQLiteEngine(WithMaxParseDepth(8))
			sql := "SELECT " + strings.Repeat("(", testCase.nesting) + "1" + strings.Repeat(")", testCase.nesting) + " FROM t"
			statements, err := engine.ParseStatements(sql)
			require.NoError(t, err)
			require.NotEmpty(t, statements)

			_, err = engine.AnalyseQuery(nil, statements[0])

			if testCase.wantFail {
				require.ErrorIs(t, err, errExpressionDepthExceeded)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWithMaxParseDepthIgnoresNonPositiveValues(t *testing.T) {
	t.Parallel()

	for _, depth := range []int{0, -1} {
		engine := NewSQLiteEngine(WithMaxParseDepth(depth))

		assert.Equal(t, defaultMaxParseDepth, engine.dialect.resolvedMaxParseDepth())
	}
}
