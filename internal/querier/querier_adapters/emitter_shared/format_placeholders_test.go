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
)

func TestPlaceholderOccurrenceOrderReturnsBindNumbersInSourceOrder(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []int
	}{
		{
			name:     "single numbered placeholder",
			sql:      "SELECT id FROM users WHERE id = ?1",
			expected: []int{1},
		},
		{
			name:     "repeated number preserved per occurrence",
			sql:      "SELECT id FROM users WHERE id = ?1 OR backup = ?1 AND active = ?2",
			expected: []int{1, 1, 2},
		},
		{
			name:     "dollar placeholders are recognised",
			sql:      "SELECT id FROM users WHERE id = $1 AND name = $2",
			expected: []int{1, 2},
		},
		{
			name:     "multi-digit placeholder decoded",
			sql:      "SELECT id FROM users WHERE id = ?10 AND seq = ?2",
			expected: []int{10, 2},
		},
		{
			name:     "no placeholders yields nil",
			sql:      "SELECT id FROM users",
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := PlaceholderOccurrenceOrder(test.sql)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestPlaceholderOccurrenceOrderNumbersBareQuestionMarks(t *testing.T) {
	result := PlaceholderOccurrenceOrder("SELECT id FROM users WHERE id = ? AND name = ?")
	assert.Equal(t, []int{1, 2}, result)
}

func TestPlaceholderOccurrenceOrderSkipsLiteralsAndComments(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []int
	}{
		{
			name:     "question mark inside string literal ignored",
			sql:      "SELECT '?1 literal' FROM users WHERE id = ?1",
			expected: []int{1},
		},
		{
			name:     "placeholder inside line comment ignored",
			sql:      "SELECT id\n-- old ?9 reference\nFROM users WHERE id = ?2",
			expected: []int{2},
		},
		{
			name:     "placeholder inside block comment ignored",
			sql:      "SELECT id /* ?9 hidden */ FROM users WHERE id = ?3",
			expected: []int{3},
		},
		{
			name:     "lone dollar without digit is ignored",
			sql:      "SELECT cost$ FROM products WHERE id = $1",
			expected: []int{1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := PlaceholderOccurrenceOrder(test.sql)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestPlaceholderOccurrenceOrderBoundsOverlongDigitRun(t *testing.T) {

	result := PlaceholderOccurrenceOrder("SELECT id FROM users WHERE id = $999999999999999999999")
	assert.Equal(t, []int{defaultMaxBindVariablesFallback + 1}, result)
}

func TestRenumberParametersExcludingLeavesOverlongMarkerVerbatim(t *testing.T) {

	input := "WHERE a = $1 AND b = $999999999999999999999 AND c = $3"
	expected := "WHERE a = $1 AND b = $999999999999999999999 AND c = $2"
	excluded := map[int]bool{2: true}

	result := RenumberParametersExcluding(input, excluded)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClauseCutsAtEarliestTerminator(t *testing.T) {
	input := "SELECT id FROM users ORDER BY name ASC OFFSET 5 LIMIT 10"
	expected := "SELECT id FROM users OFFSET 5 LIMIT 10"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClauseStopsAtFetchKeyword(t *testing.T) {
	input := "SELECT id FROM users ORDER BY name DESC FETCH FIRST 5 ROWS ONLY"
	expected := "SELECT id FROM users FETCH FIRST 5 ROWS ONLY"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestStripOrderByClauseUsesLastOrderByOccurrence(t *testing.T) {
	input := "SELECT id FROM (SELECT id FROM users ORDER BY created_at) sub ORDER BY id ASC"
	expected := "SELECT id FROM (SELECT id FROM users ORDER BY created_at) sub"

	result := StripOrderByClause(input)
	assert.Equal(t, expected, result)
}

func TestRenumberParametersExcludingClosesMultipleGaps(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		excluded map[int]bool
		expected string
	}{
		{
			name:     "dollar markers shift down by two",
			sql:      "WHERE a = $1 AND b = $4 AND c = $5",
			excluded: map[int]bool{2: true, 3: true},
			expected: "WHERE a = $1 AND b = $2 AND c = $3",
		},
		{
			name:     "question markers shift down by one",
			sql:      "WHERE a = ?1 AND b = ?3",
			excluded: map[int]bool{2: true},
			expected: "WHERE a = ?1 AND b = ?2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := RenumberParametersExcluding(test.sql, test.excluded)
			assert.Equal(t, test.expected, result)
		})
	}
}
