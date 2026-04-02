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
			input:    "SELECT id FROM users\n-- $1 as piko.limit(page_size)\nLIMIT $1",
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
