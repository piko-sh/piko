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

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestParserDepthLimitPreventsStackOverflow(t *testing.T) {
	t.Parallel()

	const depth = 100_000
	engine := NewMySQLEngine()

	t.Run("nested expression parentheses", func(t *testing.T) {
		t.Parallel()
		sql := "SELECT " + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth) + " FROM t"
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.NotEmpty(t, statements)

		_, _ = engine.AnalyseQuery(nil, statements[0])
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

	engine := NewMySQLEngine(WithMaxParseDepth(8))
	sql := "SELECT " + strings.Repeat("(", 64) + "1" + strings.Repeat(")", 64) + " FROM t"
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.NotEmpty(t, statements)
	_, _ = engine.AnalyseQuery(nil, statements[0])
}
