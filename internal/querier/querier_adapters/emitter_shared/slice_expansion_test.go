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
	"bytes"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestBuildOccurrenceOrderedFlatteningFollowsSQLOrder(t *testing.T) {
	t.Parallel()

	query := &querier_dto.AnalysedQuery{
		SQL: "SELECT id FROM t WHERE owner = ?2 AND tag IN (?1)",
		Parameters: []querier_dto.QueryParameter{
			{Number: 1, Name: "tags", IsSlice: true},
			{Number: 2, Name: "owner"},
		},
	}

	statements := buildOccurrenceOrderedFlattening(query)
	require.Len(t, statements, 2)

	fileSet := token.NewFileSet()
	var rendered strings.Builder
	for _, statement := range statements {
		var buffer bytes.Buffer
		require.NoError(t, printer.Fprint(&buffer, fileSet, statement))
		rendered.WriteString(buffer.String())
		rendered.WriteByte('\n')
	}
	source := rendered.String()

	ownerIndex := strings.Index(source, "params.Owner")
	tagsIndex := strings.Index(source, "params.Tags")
	require.GreaterOrEqual(t, ownerIndex, 0, "expected owner append in:\n%s", source)
	require.GreaterOrEqual(t, tagsIndex, 0, "expected tags range loop in:\n%s", source)
	assert.Less(t, ownerIndex, tagsIndex,
		"occurrence order must place ?2 (owner) before ?1 (tags):\n%s", source)
	assert.Contains(t, source, "range params.Tags", "the slice parameter must flatten via a range-append loop")
}

func TestBuildOccurrenceOrderedFlatteningNilWithoutPlaceholders(t *testing.T) {
	t.Parallel()

	query := &querier_dto.AnalysedQuery{
		SQL: "SELECT id FROM t",
		Parameters: []querier_dto.QueryParameter{
			{Number: 1, Name: "tags", IsSlice: true},
		},
	}

	assert.Nil(t, buildOccurrenceOrderedFlattening(query))
}
