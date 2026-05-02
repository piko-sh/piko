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

package querier_domain

import (
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestRewriteSelectAsCount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		sql         string
		analysis    *querier_dto.RawQueryAnalysis
		wantSQL     string
		wantWrapped bool
		wantErr     bool
	}{
		{
			name:        "plain select rewrites in place",
			sql:         "SELECT id, name, email FROM users",
			analysis:    &querier_dto.RawQueryAnalysis{},
			wantSQL:     "SELECT COUNT(*) FROM users",
			wantWrapped: false,
		},
		{
			name:        "select with where preserved",
			sql:         "SELECT id, name FROM users WHERE active = true",
			analysis:    &querier_dto.RawQueryAnalysis{HasWhereClause: true},
			wantSQL:     "SELECT COUNT(*) FROM users WHERE active = true",
			wantWrapped: false,
		},
		{
			name:        "order by is stripped",
			sql:         "SELECT id, name FROM users WHERE active = true ORDER BY name ASC",
			analysis:    &querier_dto.RawQueryAnalysis{HasWhereClause: true},
			wantSQL:     "SELECT COUNT(*) FROM users WHERE active = true",
			wantWrapped: false,
		},
		{
			name:        "limit and offset are stripped",
			sql:         "SELECT id FROM users LIMIT 10 OFFSET 5",
			analysis:    &querier_dto.RawQueryAnalysis{},
			wantSQL:     "SELECT COUNT(*) FROM users",
			wantWrapped: false,
		},
		{
			name: "group by wraps in subquery",
			sql:  "SELECT category, COUNT(*) FROM posts GROUP BY category",
			analysis: &querier_dto.RawQueryAnalysis{
				GroupByColumns: []querier_dto.ColumnReference{{ColumnName: "category"}},
			},
			wantSQL:     "SELECT COUNT(*) FROM (SELECT category, COUNT(*) FROM posts GROUP BY category) sub",
			wantWrapped: true,
		},
		{
			name:        "distinct wraps in subquery",
			sql:         "SELECT DISTINCT category FROM posts",
			analysis:    &querier_dto.RawQueryAnalysis{},
			wantSQL:     "SELECT COUNT(*) FROM (SELECT DISTINCT category FROM posts) sub",
			wantWrapped: true,
		},
		{
			name: "top-level union wraps in subquery",
			sql:  "SELECT a, b FROM t UNION SELECT c, d FROM u",
			analysis: &querier_dto.RawQueryAnalysis{
				CompoundBranches: []querier_dto.RawCompoundBranch{
					{Query: &querier_dto.RawQueryAnalysis{}},
				},
			},
			wantSQL:     "SELECT COUNT(*) FROM (SELECT a, b FROM t UNION SELECT c, d FROM u) sub",
			wantWrapped: true,
		},
		{
			name:        "window function wraps in subquery",
			sql:         "SELECT id, ROW_NUMBER() OVER (PARTITION BY category ORDER BY id) AS rn FROM posts",
			analysis:    &querier_dto.RawQueryAnalysis{},
			wantSQL:     "SELECT COUNT(*) FROM (SELECT id, ROW_NUMBER() OVER (PARTITION BY category ORDER BY id) AS rn FROM posts) sub",
			wantWrapped: true,
		},
		{
			name:        "join preserved",
			sql:         "SELECT p.id, b.name FROM posts p JOIN blueprints b ON b.id = p.blueprint_id WHERE p.environment_id = $1",
			analysis:    &querier_dto.RawQueryAnalysis{HasWhereClause: true},
			wantSQL:     "SELECT COUNT(*) FROM posts p JOIN blueprints b ON b.id = p.blueprint_id WHERE p.environment_id = $1",
			wantWrapped: false,
		},
		{
			name:        "subquery select is ignored when looking for top-level select",
			sql:         "SELECT id FROM users WHERE id IN (SELECT user_id FROM events)",
			analysis:    &querier_dto.RawQueryAnalysis{HasWhereClause: true},
			wantSQL:     "SELECT COUNT(*) FROM users WHERE id IN (SELECT user_id FROM events)",
			wantWrapped: false,
		},
		{
			name:     "empty input is rejected",
			sql:      "",
			analysis: &querier_dto.RawQueryAnalysis{},
			wantErr:  true,
		},
		{
			name:     "non-select input is rejected",
			sql:      "DELETE FROM users",
			analysis: &querier_dto.RawQueryAnalysis{},
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotSQL, gotWrapped, err := RewriteSelectAsCount(tc.sql, tc.analysis)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantSQL, gotSQL)
			require.Equal(t, tc.wantWrapped, gotWrapped)
		})
	}
}

func TestRewriteSelectAsCountIgnoresLiterals(t *testing.T) {
	t.Parallel()

	sql := `SELECT id, "select", category FROM posts WHERE title = 'GROUP BY tutorials'`
	got, wrapped, err := RewriteSelectAsCount(sql, &querier_dto.RawQueryAnalysis{HasWhereClause: true})

	require.NoError(t, err)
	require.False(t, wrapped, "quoted identifiers and string literals must not trigger wrapping")
	require.Equal(t, `SELECT COUNT(*) FROM posts WHERE title = 'GROUP BY tutorials'`, got)
}

func TestRewriteSelectAsCount_DoubledQuoteEscapeInLiteral(t *testing.T) {
	t.Parallel()

	sql := `SELECT id FROM users WHERE name = 'O''Brien'`
	got, wrapped, err := RewriteSelectAsCount(sql, &querier_dto.RawQueryAnalysis{HasWhereClause: true})

	require.NoError(t, err)
	require.False(t, wrapped)
	require.Equal(t, `SELECT COUNT(*) FROM users WHERE name = 'O''Brien'`, got)
}

func TestRewriteSelectAsCount_DoubledQuoteEscapeAcrossKeyword(t *testing.T) {
	t.Parallel()

	sql := `SELECT id FROM posts WHERE title = 'has ''GROUP BY'' in middle'`
	got, wrapped, err := RewriteSelectAsCount(sql, &querier_dto.RawQueryAnalysis{HasWhereClause: true})

	require.NoError(t, err)
	require.False(t, wrapped, "GROUP BY inside a doubled-quote literal must not trigger wrapping")
	require.Equal(t, `SELECT COUNT(*) FROM posts WHERE title = 'has ''GROUP BY'' in middle'`, got)
}

func TestRewriteSelectAsCount_DoesNotSplitOnUTF8LeadByte(t *testing.T) {
	t.Parallel()

	sql := `SELECT id FROM users WHERE name = 'caf` + "é" + `'`
	got, wrapped, err := RewriteSelectAsCount(sql, &querier_dto.RawQueryAnalysis{HasWhereClause: true})

	require.NoError(t, err)
	require.False(t, wrapped)
	require.Contains(t, got, "'café'")
}

func TestRewriteSelectAsCount_PreservesWherePlaceholders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sql         string
		wantSQL     string
		wantWrapped bool
	}{
		{
			name:        "where_order_limit_offset",
			sql:         "SELECT a FROM t WHERE x = $1 AND y = $2 ORDER BY z LIMIT $3 OFFSET $4",
			wantSQL:     "SELECT COUNT(*) FROM t WHERE x = $1 AND y = $2",
			wantWrapped: false,
		},
		{
			name:        "where_limit_only",
			sql:         "SELECT a FROM t WHERE x = $1 LIMIT $2",
			wantSQL:     "SELECT COUNT(*) FROM t WHERE x = $1",
			wantWrapped: false,
		},
		{
			name:        "distinct_wrap_keeps_inner_where_placeholders",
			sql:         "SELECT DISTINCT a FROM t WHERE x = $1 ORDER BY z LIMIT $2",
			wantSQL:     "SELECT COUNT(*) FROM (SELECT DISTINCT a FROM t WHERE x = $1) sub",
			wantWrapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, wrapped, err := RewriteSelectAsCount(tt.sql, &querier_dto.RawQueryAnalysis{HasWhereClause: true})
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, got)
			require.Equal(t, tt.wantWrapped, wrapped)
		})
	}
}

func TestStripTrailingOrderLimitOffsetPreservesCompoundBranches(t *testing.T) {
	t.Parallel()

	got := stripTrailingOrderLimitOffset(
		"SELECT a FROM t1 ORDER BY a UNION SELECT b FROM t2 ORDER BY b",
	)
	require.Equal(t, "SELECT a FROM t1 ORDER BY a UNION SELECT b FROM t2", got)

	gotLimit := stripTrailingOrderLimitOffset(
		"SELECT a FROM t1 UNION SELECT b FROM t2 ORDER BY b LIMIT 10 OFFSET 5",
	)
	require.Equal(t, "SELECT a FROM t1 UNION SELECT b FROM t2", gotLimit)
}
