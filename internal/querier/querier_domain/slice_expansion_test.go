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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandSlicePlaceholders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		specs    []SliceExpansionSpec
		expected string
	}{

		{
			name:     "single slice with 3 elements",
			query:    "SELECT * FROM t WHERE id IN (?1)",
			specs:    []SliceExpansionSpec{{1, 3}},
			expected: "SELECT * FROM t WHERE id IN (?1,?2,?3)",
		},
		{
			name:     "single slice with 1 element",
			query:    "SELECT * FROM t WHERE id IN (?1)",
			specs:    []SliceExpansionSpec{{1, 1}},
			expected: "SELECT * FROM t WHERE id IN (?1)",
		},
		{
			name:     "single slice with 0 elements (empty)",
			query:    "SELECT * FROM t WHERE id IN (?1)",
			specs:    []SliceExpansionSpec{{1, 0}},
			expected: "SELECT * FROM t WHERE id IN (NULL)",
		},

		{
			name:     "slice then scalar",
			query:    "SELECT * FROM t WHERE status IN (?1) AND priority = ?2",
			specs:    []SliceExpansionSpec{{1, 3}, {2, 1}},
			expected: "SELECT * FROM t WHERE status IN (?1,?2,?3) AND priority = ?4",
		},
		{
			name:     "slice then two scalars",
			query:    "SELECT * FROM t WHERE status IN (?1) AND priority = ?2 AND active = ?3",
			specs:    []SliceExpansionSpec{{1, 3}, {2, 1}, {3, 1}},
			expected: "SELECT * FROM t WHERE status IN (?1,?2,?3) AND priority = ?4 AND active = ?5",
		},
		{
			name:     "slice then scalar then limit",
			query:    "SELECT * FROM t WHERE status IN (?1) AND priority = ?2 LIMIT ?3",
			specs:    []SliceExpansionSpec{{1, 2}, {2, 1}, {3, 1}},
			expected: "SELECT * FROM t WHERE status IN (?1,?2) AND priority = ?3 LIMIT ?4",
		},

		{
			name:     "scalar before slice",
			query:    "UPDATE t SET status = ?1 WHERE id IN (?2)",
			specs:    []SliceExpansionSpec{{1, 1}, {2, 3}},
			expected: "UPDATE t SET status = ?1 WHERE id IN (?2,?3,?4)",
		},
		{
			name:     "scalar before slice with scalar after",
			query:    "UPDATE t SET status = ?1 WHERE id IN (?2) AND active = ?3",
			specs:    []SliceExpansionSpec{{1, 1}, {2, 3}, {3, 1}},
			expected: "UPDATE t SET status = ?1 WHERE id IN (?2,?3,?4) AND active = ?5",
		},

		{
			name:     "scalar slice scalar",
			query:    "UPDATE t SET updated_at = ?1 WHERE status IN (?2) AND active = ?3",
			specs:    []SliceExpansionSpec{{1, 1}, {2, 3}, {3, 1}},
			expected: "UPDATE t SET updated_at = ?1 WHERE status IN (?2,?3,?4) AND active = ?5",
		},

		{
			name:     "two slices",
			query:    "SELECT * FROM t WHERE status IN (?1) AND id IN (?2)",
			specs:    []SliceExpansionSpec{{1, 2}, {2, 3}},
			expected: "SELECT * FROM t WHERE status IN (?1,?2) AND id IN (?3,?4,?5)",
		},
		{
			name:     "two slices with scalar between",
			query:    "SELECT * FROM t WHERE status IN (?1) AND priority = ?2 AND id IN (?3)",
			specs:    []SliceExpansionSpec{{1, 2}, {2, 1}, {3, 3}},
			expected: "SELECT * FROM t WHERE status IN (?1,?2) AND priority = ?3 AND id IN (?4,?5,?6)",
		},
		{
			name:     "two slices with scalar after",
			query:    "SELECT * FROM t WHERE status IN (?1) AND id IN (?2) LIMIT ?3",
			specs:    []SliceExpansionSpec{{1, 2}, {2, 3}, {3, 1}},
			expected: "SELECT * FROM t WHERE status IN (?1,?2) AND id IN (?3,?4,?5) LIMIT ?6",
		},
		{
			name:     "three slices",
			query:    "SELECT * FROM t WHERE a IN (?1) AND b IN (?2) AND c IN (?3)",
			specs:    []SliceExpansionSpec{{1, 2}, {2, 3}, {3, 1}},
			expected: "SELECT * FROM t WHERE a IN (?1,?2) AND b IN (?3,?4,?5) AND c IN (?6)",
		},

		{
			name:     "empty slice with scalar after",
			query:    "SELECT * FROM t WHERE id IN (?1) AND active = ?2",
			specs:    []SliceExpansionSpec{{1, 0}, {2, 1}},
			expected: "SELECT * FROM t WHERE id IN (NULL) AND active = ?1",
		},
		{
			name:     "empty slice between scalars",
			query:    "SELECT * FROM t WHERE a = ?1 AND id IN (?2) AND b = ?3",
			specs:    []SliceExpansionSpec{{1, 1}, {2, 0}, {3, 1}},
			expected: "SELECT * FROM t WHERE a = ?1 AND id IN (NULL) AND b = ?2",
		},

		{
			name:     "all scalars unchanged",
			query:    "SELECT * FROM t WHERE a = ?1 AND b = ?2",
			specs:    []SliceExpansionSpec{{1, 1}, {2, 1}},
			expected: "SELECT * FROM t WHERE a = ?1 AND b = ?2",
		},

		{
			name:     "same placeholder appears twice",
			query:    "SELECT * FROM t WHERE a = ?1 OR b = ?1",
			specs:    []SliceExpansionSpec{{1, 1}},
			expected: "SELECT * FROM t WHERE a = ?1 OR b = ?1",
		},
		{
			name:     "scalar repeated with slice renumbering",
			query:    "SELECT * FROM t WHERE a = ?2 OR b = ?2 AND status IN (?1)",
			specs:    []SliceExpansionSpec{{1, 3}, {2, 1}},
			expected: "SELECT * FROM t WHERE a = ?4 OR b = ?4 AND status IN (?1,?2,?3)",
		},

		{
			name:     "empty specs returns query unchanged",
			query:    "SELECT * FROM t WHERE a = ?1",
			specs:    nil,
			expected: "SELECT * FROM t WHERE a = ?1",
		},

		{
			name:     "standalone placeholder renumbered",
			query:    "INSERT INTO t (a, b) VALUES (?1, ?2)",
			specs:    []SliceExpansionSpec{{1, 1}, {2, 1}},
			expected: "INSERT INTO t (a, b) VALUES (?1, ?2)",
		},

		{
			name:     "unknown placeholder left unchanged",
			query:    "SELECT * FROM t WHERE a = ?1 AND b = ?99",
			specs:    []SliceExpansionSpec{{1, 1}},
			expected: "SELECT * FROM t WHERE a = ?1 AND b = ?99",
		},

		{
			name: "FetchDueTasks pattern",
			query: `SELECT id, status FROM tasks
WHERE status IN (?1) AND priority = ?2 AND execute_at <= ?3
ORDER BY priority DESC LIMIT ?4`,
			specs: []SliceExpansionSpec{{1, 2}, {2, 1}, {3, 1}, {4, 1}},
			expected: `SELECT id, status FROM tasks
WHERE status IN (?1,?2) AND priority = ?3 AND execute_at <= ?4
ORDER BY priority DESC LIMIT ?5`,
		},
		{
			name:     "MarkTasksAsProcessing pattern",
			query:    "UPDATE tasks SET updated_at = ?1 WHERE id IN (?2)",
			specs:    []SliceExpansionSpec{{1, 1}, {2, 4}},
			expected: "UPDATE tasks SET updated_at = ?1 WHERE id IN (?2,?3,?4,?5)",
		},
		{
			name:     "FindArtefactIDsByTagValues pattern",
			query:    "SELECT DISTINCT artefact_id FROM variant_tag WHERE tag_key = ?1 AND tag_value IN (?2)",
			specs:    []SliceExpansionSpec{{1, 1}, {2, 3}},
			expected: "SELECT DISTINCT artefact_id FROM variant_tag WHERE tag_key = ?1 AND tag_value IN (?2,?3,?4)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExpandSlicePlaceholders(tt.query, tt.specs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandSlicePlaceholdersLargeSlice(t *testing.T) {
	t.Parallel()

	const count = 150
	specs := []SliceExpansionSpec{{1, count}, {2, 1}}
	query := "SELECT * FROM t WHERE id IN (?1) AND active = ?2"

	result := ExpandSlicePlaceholders(query, specs)

	inStart := strings.Index(result, "IN (")
	inEnd := strings.Index(result[inStart:], ")") + inStart
	inClause := result[inStart+4 : inEnd]
	placeholders := strings.Split(inClause, ",")
	assert.Len(t, placeholders, count)

	assert.Equal(t, "?1", placeholders[0])
	assert.Equal(t, "?150", placeholders[count-1])

	assert.Contains(t, result, fmt.Sprintf("AND active = ?%d", count+1))
}

func TestExpandSlicePlaceholdersSingleElementSliceWithRenumbering(t *testing.T) {
	t.Parallel()

	query := "SELECT * FROM t WHERE id IN (?1) AND status = ?2"
	specs := []SliceExpansionSpec{{1, 1}, {2, 1}}

	result := ExpandSlicePlaceholders(query, specs)
	assert.Equal(t, "SELECT * FROM t WHERE id IN (?1) AND status = ?2", result)
}

func TestExpandSlicePlaceholdersSpecsOutOfOrder(t *testing.T) {
	t.Parallel()

	query := "SELECT * FROM t WHERE status IN (?1) AND priority = ?2"
	specs := []SliceExpansionSpec{{2, 1}, {1, 3}}

	result := ExpandSlicePlaceholders(query, specs)
	assert.Equal(t, "SELECT * FROM t WHERE status IN (?1,?2,?3) AND priority = ?4", result)
}

func TestExpandSlicePlaceholdersMultiDigitNumbers(t *testing.T) {
	t.Parallel()

	query := "SELECT * FROM t WHERE a = ?10 AND b IN (?11) AND c = ?12"
	specs := []SliceExpansionSpec{{10, 1}, {11, 3}, {12, 1}}

	result := ExpandSlicePlaceholders(query, specs)
	assert.Equal(t, "SELECT * FROM t WHERE a = ?1 AND b IN (?2,?3,?4) AND c = ?5", result)
}
