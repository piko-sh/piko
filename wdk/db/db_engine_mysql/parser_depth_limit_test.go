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

package db_engine_mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestParserDepthLimitPreventsStackOverflow(t *testing.T) {
	t.Parallel()

	const depth = 100_000
	engine := NewMySQLEngine(WithMaxTokensPerStatement(4 * depth))

	t.Run("nested expression parentheses", func(t *testing.T) {
		t.Parallel()
		sql := "SELECT " + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth) + " FROM t"
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)

		analysis, err := engine.AnalyseQuery(nil, statements[0])

		require.ErrorIs(t, err, errExpressionDepthExceeded)
		assert.Nil(t, analysis)
	})
}

func TestDataModifyingAnalysersHonourDepthGuard(t *testing.T) {
	t.Parallel()

	analysers := map[string]func(*parser) (*querier_dto.RawQueryAnalysis, error){
		"insert": (*parser).analyseInsert,
		"update": (*parser).analyseUpdate,
		"delete": (*parser).analyseDelete,
		"values": (*parser).analyseValues,
	}
	for name, analyse := range analysers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := newParser(nil)
			p.maxParseDepth = 4
			p.analysisDepth = p.maxParseDepth
			_, err := analyse(p)
			require.ErrorIs(t, err, errAnalysisDepthExceeded)
		})
	}
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

			engine := NewMySQLEngine(WithMaxParseDepth(8))
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

func TestAnalyseQueryCollectsCompoundArmsIntoOneFlatList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		sql           string
		wantOperators []querier_dto.CompoundOperator
	}{
		{
			name:          "three arms with mixed operators",
			sql:           "SELECT id FROM a UNION SELECT id FROM b INTERSECT SELECT id FROM c ORDER BY id LIMIT ?",
			wantOperators: []querier_dto.CompoundOperator{querier_dto.CompoundUnion, querier_dto.CompoundIntersect},
		},
		{
			name:          "union all followed by except with a locking clause",
			sql:           "SELECT id FROM a UNION ALL SELECT id FROM b EXCEPT SELECT id FROM c FOR UPDATE",
			wantOperators: []querier_dto.CompoundOperator{querier_dto.CompoundUnionAll, querier_dto.CompoundExcept},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewMySQLEngine()
			statements, err := engine.ParseStatements(testCase.sql)
			require.NoError(t, err)

			analysis, err := engine.AnalyseQuery(nil, statements[0])

			require.NoError(t, err)
			require.Len(t, analysis.CompoundBranches, len(testCase.wantOperators))
			for index, branch := range analysis.CompoundBranches {
				assert.Equal(t, testCase.wantOperators[index], branch.Operator)
				require.NotNil(t, branch.Query)
				assert.Empty(t, branch.Query.CompoundBranches, "arms must not nest inside one another")
			}
		})
	}
}

func TestAnalyseQueryHandlesLongCompoundChains(t *testing.T) {
	t.Parallel()

	const arms = 10_000
	engine := NewMySQLEngine()
	statements, err := engine.ParseStatements("SELECT 1" + strings.Repeat(" UNION SELECT 1", arms-1))
	require.NoError(t, err)

	analysis, err := engine.AnalyseQuery(nil, statements[0])

	require.NoError(t, err)
	assert.Len(t, analysis.CompoundBranches, arms-1)
}
