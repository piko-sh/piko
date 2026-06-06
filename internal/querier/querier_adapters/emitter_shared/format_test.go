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

package emitter_shared

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestStripDirectiveComments(t *testing.T) {
	input := "SELECT id, name\n-- piko.name: GetUser\nFROM users\n-- piko.command: :one\nWHERE id = ?1"
	expected := "SELECT id, name\nFROM users\nWHERE id = ?1"

	result := StripDirectiveComments(input)
	assert.Equal(t, expected, result)
}

func TestStripDirectiveCommentsStripsCallHeader(t *testing.T) {
	input := "-- piko.query(name: ListFailedTasks, command: many)\nSELECT id, name\nFROM tasks\nWHERE status = 'FAILED'"
	expected := "SELECT id, name\nFROM tasks\nWHERE status = 'FAILED'"

	result := StripDirectiveComments(input)
	assert.Equal(t, expected, result)
}

func TestStripDirectiveCommentsPreservesRegularComments(t *testing.T) {
	input := "SELECT id\n-- This is a regular comment\nFROM users\n-- Another comment about the query\nWHERE active = 1"
	expected := "SELECT id\n-- This is a regular comment\nFROM users\n-- Another comment about the query\nWHERE active = 1"

	result := StripDirectiveComments(input)
	assert.Equal(t, expected, result)
}

func TestStripDirectiveCommentsParameterDirectives(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "question mark parameter directive",
			input:    "SELECT id FROM users\n-- ?1 as piko.param(user_id)\nWHERE id = ?1",
			expected: "SELECT id FROM users\nWHERE id = ?1",
		},
		{
			name:     "dollar parameter directive",
			input:    "SELECT id FROM users\n-- $1 as piko.param(page_size)\nLIMIT $1",
			expected: "SELECT id FROM users\nLIMIT $1",
		},
		{
			name:     "colon parameter directive",
			input:    "SELECT id FROM users\n-- :email as piko.param\nWHERE email = :email",
			expected: "SELECT id FROM users\nWHERE email = :email",
		},
		{
			name:     "at parameter directive",
			input:    "SELECT id FROM users\n-- @name as piko.param\nWHERE name = @name",
			expected: "SELECT id FROM users\nWHERE name = @name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := StripDirectiveComments(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestRewriteNamedParameters(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		parameters []querier_dto.QueryParameter
		expected   string
	}{
		{
			name: "colon prefix",
			sql:  "SELECT id FROM users WHERE email = :email AND active = :active",
			parameters: []querier_dto.QueryParameter{
				{Name: "email", Number: 1},
				{Name: "active", Number: 2},
			},
			expected: "SELECT id FROM users WHERE email = ?1 AND active = ?2",
		},
		{
			name: "at prefix",
			sql:  "SELECT id FROM users WHERE name = @name",
			parameters: []querier_dto.QueryParameter{
				{Name: "name", Number: 1},
			},
			expected: "SELECT id FROM users WHERE name = ?1",
		},
		{
			name: "dollar prefix",
			sql:  "SELECT id FROM users WHERE id = $user_id",
			parameters: []querier_dto.QueryParameter{
				{Name: "user_id", Number: 1},
			},
			expected: "SELECT id FROM users WHERE id = ?1",
		},
		{
			name: "reused parameter gets same number",
			sql:  "SELECT id FROM users WHERE email = :email OR backup_email = :email",
			parameters: []querier_dto.QueryParameter{
				{Name: "email", Number: 1},
			},
			expected: "SELECT id FROM users WHERE email = ?1 OR backup_email = ?1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := RewriteNamedParameters(test.sql, test.parameters)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestRewriteBracedNamedToPositional(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		parameters []querier_dto.QueryParameter
		expected   string
	}{
		{
			name: "single braced placeholder",
			sql:  "SELECT id FROM products WHERE in_stock = {only_stocked:Bool}",
			parameters: []querier_dto.QueryParameter{
				{Name: "only_stocked", Number: 1},
			},
			expected: "SELECT id FROM products WHERE in_stock = $1",
		},
		{
			name: "multiple braced placeholders keep their numbers",
			sql:  "SELECT id FROM products WHERE price >= {min_price:UInt32} AND name = {term:String}",
			parameters: []querier_dto.QueryParameter{
				{Name: "min_price", Number: 1},
				{Name: "term", Number: 2},
			},
			expected: "SELECT id FROM products WHERE price >= $1 AND name = $2",
		},
		{
			name: "unknown name preserved verbatim",
			sql:  "SELECT id FROM products WHERE flag = {unknown:Bool}",
			parameters: []querier_dto.QueryParameter{
				{Name: "only_stocked", Number: 1},
			},
			expected: "SELECT id FROM products WHERE flag = {unknown:Bool}",
		},
		{
			name: "braces inside string literal untouched",
			sql:  "SELECT '{not:a placeholder}' AS label FROM products WHERE in_stock = {only_stocked:Bool}",
			parameters: []querier_dto.QueryParameter{
				{Name: "only_stocked", Number: 1},
			},
			expected: "SELECT '{not:a placeholder}' AS label FROM products WHERE in_stock = $1",
		},
		{
			name:       "no parameters leaves sql unchanged",
			sql:        "SELECT id FROM products",
			parameters: nil,
			expected:   "SELECT id FROM products",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := RewriteBracedNamedToPositional(test.sql, test.parameters)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestRewriteNamedParametersUnknownPreserved(t *testing.T) {

	sql := "SELECT id FROM users WHERE email = :email AND name = :unknown_param"
	parameters := []querier_dto.QueryParameter{
		{Name: "email", Number: 1},
	}

	result := RewriteNamedParameters(sql, parameters)
	assert.Contains(t, result, ":unknown_param")
	assert.Contains(t, result, "?1")
}

func TestRewriteNamedParametersNoChange(t *testing.T) {

	sql := "SELECT id FROM users WHERE id = ?1 AND active = ?2"
	parameters := []querier_dto.QueryParameter{
		{Name: "p1", Number: 1},
		{Name: "p2", Number: 2},
	}

	result := RewriteNamedParameters(sql, parameters)
	assert.Equal(t, sql, result)
}

func TestStripOrderByClause(t *testing.T) {
	input := "SELECT id, name FROM users ORDER BY name ASC"
	expected := "SELECT id, name FROM users"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClausePreservesLimit(t *testing.T) {
	input := "SELECT id, name FROM users ORDER BY name ASC LIMIT 10"
	expected := "SELECT id, name FROM users LIMIT 10"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClauseNoOrderBy(t *testing.T) {
	input := "SELECT id, name FROM users WHERE active = 1"

	result := StripOrderByClause(input)
	assert.Equal(t, input, result)
}

func TestStripOrderByClauseSkipsStringLiterals(t *testing.T) {
	input := "SELECT id, name FROM users ORDER BY 'before LIMIT 10' DESC LIMIT 5"
	expected := "SELECT id, name FROM users LIMIT 5"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClauseRespectsIdentifierBoundaries(t *testing.T) {
	input := "SELECT id FROM users ORDER BY for_each_user_id DESC"
	expected := "SELECT id FROM users"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClauseSkipsBlockComments(t *testing.T) {
	input := "SELECT id FROM users ORDER BY /* LIMIT inside comment */ created_at DESC"
	expected := "SELECT id FROM users"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClausePreservesSubqueryOrderByWhenOuterHasNone(t *testing.T) {
	input := "SELECT id FROM (SELECT id FROM users ORDER BY name ASC) AS recent"
	expected := "SELECT id FROM (SELECT id FROM users ORDER BY name ASC) AS recent"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClauseStripsOuterButKeepsSubquery(t *testing.T) {
	input := "SELECT id FROM (SELECT id FROM users ORDER BY name ASC) AS recent ORDER BY id DESC"
	expected := "SELECT id FROM (SELECT id FROM users ORDER BY name ASC) AS recent"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClauseIgnoresTerminatorInsideSubquery(t *testing.T) {
	input := "SELECT id FROM users WHERE id IN (SELECT id FROM posts LIMIT 5) ORDER BY name ASC"
	expected := "SELECT id FROM users WHERE id IN (SELECT id FROM posts LIMIT 5)"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestRenumberParametersExcluding(t *testing.T) {

	input := "SELECT id FROM users WHERE id = $1 AND name = $3"
	excluded := map[int]bool{2: true}

	result := RenumberParametersExcluding(input, excluded)
	assert.Equal(t, "SELECT id FROM users WHERE id = $1 AND name = $2", result)
}

func TestRenumberParametersExcludingNoExclusions(t *testing.T) {
	input := "SELECT id FROM users WHERE id = $1 AND name = $2"
	excluded := map[int]bool{}

	result := RenumberParametersExcluding(input, excluded)
	assert.Equal(t, input, result)
}

func TestRewriteNamedParametersSkipsLineComments(t *testing.T) {
	input := "SELECT id\n-- WHERE email = :email\nFROM users WHERE email = :email"
	parameters := []querier_dto.QueryParameter{
		{Name: "email", Number: 1},
	}

	result := RewriteNamedParameters(input, parameters)

	assert.Contains(t, result, "-- WHERE email = :email")
	assert.Contains(t, result, "WHERE email = ?1")
}

func TestRewriteNamedParametersSkipsBlockComments(t *testing.T) {
	input := "SELECT id /* lookup :email */ FROM users WHERE email = :email"
	parameters := []querier_dto.QueryParameter{
		{Name: "email", Number: 1},
	}

	result := RewriteNamedParameters(input, parameters)

	assert.Contains(t, result, "/* lookup :email */")
	assert.Contains(t, result, "WHERE email = ?1")
}

func TestRenumberParametersExcludingSkipsLineComments(t *testing.T) {
	input := "SELECT id FROM users\n-- old: $1 and $3\nWHERE id = $1 AND name = $3"
	excluded := map[int]bool{2: true}

	result := RenumberParametersExcluding(input, excluded)

	assert.Contains(t, result, "-- old: $1 and $3")
	assert.Contains(t, result, "WHERE id = $1 AND name = $2")
}

func TestRenumberParametersExcludingSkipsBlockComments(t *testing.T) {
	input := "SELECT id FROM users /* old: $1 and $3 */ WHERE id = $1 AND name = $3"
	excluded := map[int]bool{2: true}

	result := RenumberParametersExcluding(input, excluded)

	assert.Contains(t, result, "/* old: $1 and $3 */")
	assert.Contains(t, result, "WHERE id = $1 AND name = $2")
}

func TestStripOrderByClauseSkipsLineComments(t *testing.T) {
	input := "SELECT id FROM users ORDER BY name -- LIMIT 10 inside comment\nDESC"
	expected := "SELECT id FROM users"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}
