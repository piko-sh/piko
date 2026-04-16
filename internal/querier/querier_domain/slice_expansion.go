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
	"slices"
	"strconv"
	"strings"
)

// SliceExpansionSpec describes how one query parameter expands at runtime.
type SliceExpansionSpec struct {
	// Placeholder is the original ?N number in the SQL template (1-based).
	Placeholder int

	// Count is the number of bind positions this parameter occupies (1 for
	// scalars, len(slice) for slices, 0 for empty slices).
	Count int
}

// placeholderMapping records the new starting bind position and element count
// for an original placeholder after slice expansion.
type placeholderMapping struct {
	// newStart is the first bind position assigned to this placeholder after
	// renumbering.
	newStart int

	// count is the number of bind positions this placeholder occupies.
	count int
}

// placeholderOccurrence records the location and metadata of a single ?N
// placeholder found in a query string.
type placeholderOccurrence struct {
	// start is the byte offset of the '?' character in the query string.
	start int

	// end is the byte offset immediately after the last digit of the placeholder.
	end int

	// originalNum is the parsed placeholder number (1-based).
	originalNum int

	// inParens indicates whether the placeholder is wrapped in parentheses.
	inParens bool
}

// buildPlaceholderRemap sorts the specs by placeholder number and computes
// contiguous new bind positions for each original placeholder.
//
// Takes specs ([]SliceExpansionSpec) which describes the expansion for each
// parameter.
//
// Returns map[int]placeholderMapping which maps original placeholder numbers
// to their new positions.
func buildPlaceholderRemap(specs []SliceExpansionSpec) map[int]placeholderMapping {
	sorted := make([]SliceExpansionSpec, len(specs))
	copy(sorted, specs)
	slices.SortFunc(sorted, func(a, b SliceExpansionSpec) int {
		return a.Placeholder - b.Placeholder
	})

	remap := make(map[int]placeholderMapping, len(sorted))
	pos := 1
	for _, spec := range sorted {
		remap[spec.Placeholder] = placeholderMapping{newStart: pos, count: spec.Count}
		if spec.Count > 0 {
			pos += spec.Count
		}
	}

	return remap
}

// findPlaceholderOccurrences scans the query for all ?N tokens and returns
// their byte positions and parsed placeholder numbers.
//
// Takes query (string) which is the SQL query to scan.
//
// Returns []placeholderOccurrence which contains the found placeholder
// positions.
func findPlaceholderOccurrences(query string) []placeholderOccurrence {
	var occurrences []placeholderOccurrence
	i := 0
	for i < len(query) {
		if query[i] == '?' && i+1 < len(query) && query[i+1] >= '1' && query[i+1] <= '9' {
			start := i
			i++
			numStart := i
			for i < len(query) && query[i] >= '0' && query[i] <= '9' {
				i++
			}
			n, _ := strconv.Atoi(query[numStart:i])

			inParens := start > 0 && query[start-1] == '(' && i < len(query) && query[i] == ')'
			occurrences = append(occurrences, placeholderOccurrence{
				start:       start,
				end:         i,
				originalNum: n,
				inParens:    inParens,
			})
		} else {
			i++
		}
	}

	return occurrences
}

// ExpandSlicePlaceholders rewrites ?N placeholders in a SQL query for
// slice parameter expansion with correct renumbering. For each
// parameter whose Count differs from 1, the placeholder is expanded
// (or collapsed to NULL) and all subsequent ?M placeholders are
// renumbered to maintain contiguous bind positions.
//
// Example: given SQL "WHERE status IN (?1) AND priority = ?2" with
// specs [{1, 3}, {2, 1}], the result is
// "WHERE status IN (?1,?2,?3) AND priority = ?4".
//
// Empty slices (Count=0) produce (NULL) which matches no rows in an
// IN clause.
//
// Takes query (string) which is the SQL template containing ?N
// placeholders to rewrite.
// Takes specs ([]SliceExpansionSpec) which describes the expansion
// count for each placeholder.
//
// Returns string which is the rewritten SQL with renumbered and
// expanded placeholders.
func ExpandSlicePlaceholders(query string, specs []SliceExpansionSpec) string {
	if len(specs) == 0 {
		return query
	}

	remap := buildPlaceholderRemap(specs)
	occurrences := findPlaceholderOccurrences(query)

	if len(occurrences) == 0 {
		return query
	}

	var b strings.Builder
	b.Grow(len(query) + len(occurrences)*4)

	prevEnd := 0
	for _, occ := range occurrences {
		m, ok := remap[occ.originalNum]
		if !ok {
			continue
		}

		replStart := occ.start
		replEnd := occ.end

		var replacement string
		switch {
		case m.count == 0 && occ.inParens:
			replacement = "(NULL)"
			replStart--
			replEnd++

		case m.count > 1 && occ.inParens:
			replacement = expandSlice(m.newStart, m.count)
			replStart--
			replEnd++

		default:
			replacement = "?" + strconv.Itoa(m.newStart)
		}

		b.WriteString(query[prevEnd:replStart])
		b.WriteString(replacement)
		prevEnd = replEnd
	}
	b.WriteString(query[prevEnd:])

	return b.String()
}

// expandSlice builds a parenthesised list of numbered placeholders:
// (?start,?start+1,...,?start+count-1).
//
// Takes start (int) which is the first bind position number in the
// expanded list.
// Takes count (int) which is the number of placeholders to generate.
//
// Returns string which is the parenthesised comma-separated
// placeholder list.
func expandSlice(start, count int) string {
	var b strings.Builder
	b.Grow(count*4 + 2)
	b.WriteByte('(')
	for i := range count {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('?')
		b.WriteString(strconv.Itoa(start + i))
	}
	b.WriteByte(')')
	return b.String()
}
