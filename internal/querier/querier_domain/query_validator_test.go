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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestValidateDuplicateNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		queries            []*querier_dto.AnalysedQuery
		expectedCount      int
		expectedDiagnostic func(t *testing.T, diagnostics []querier_dto.SourceError)
	}{
		{
			name:          "no queries returns empty result",
			queries:       nil,
			expectedCount: 0,
			expectedDiagnostic: func(t *testing.T, diagnostics []querier_dto.SourceError) {
				assert.Empty(t, diagnostics)
			},
		},
		{
			name: "unique names returns empty result",
			queries: []*querier_dto.AnalysedQuery{
				{Name: "GetUser", Filename: "users.sql", Line: 1},
				{Name: "ListUsers", Filename: "users.sql", Line: 10},
				{Name: "CreateUser", Filename: "users.sql", Line: 20},
			},
			expectedCount: 0,
			expectedDiagnostic: func(t *testing.T, diagnostics []querier_dto.SourceError) {
				assert.Empty(t, diagnostics)
			},
		},
		{
			name: "one duplicate returns Q006 with correct filename and line",
			queries: []*querier_dto.AnalysedQuery{
				{Name: "GetUser", Filename: "users.sql", Line: 1},
				{Name: "GetUser", Filename: "admin.sql", Line: 5},
			},
			expectedCount: 1,
			expectedDiagnostic: func(t *testing.T, diagnostics []querier_dto.SourceError) {
				require.Len(t, diagnostics, 1)

				diag := diagnostics[0]
				assert.Equal(t, "admin.sql", diag.Filename,
					"diagnostic should reference the second (duplicate) file")
				assert.Equal(t, 5, diag.Line,
					"diagnostic should reference the duplicate's line")
				assert.Equal(t, 1, diag.Column)
				assert.Equal(t, "Q006", diag.Code)
				assert.Equal(t, querier_dto.SeverityError, diag.Severity)
				assert.Contains(t, diag.Message, `"GetUser"`,
					"message should include the duplicate query name")
				assert.Contains(t, diag.Message, "users.sql:1",
					"message should reference the first definition's location")
			},
		},
		{
			name: "multiple duplicates returns multiple Q006 errors",
			queries: []*querier_dto.AnalysedQuery{
				{Name: "GetUser", Filename: "users.sql", Line: 1},
				{Name: "ListPosts", Filename: "posts.sql", Line: 1},
				{Name: "GetUser", Filename: "admin.sql", Line: 10},
				{Name: "ListPosts", Filename: "dashboard.sql", Line: 20},
			},
			expectedCount: 2,
			expectedDiagnostic: func(t *testing.T, diagnostics []querier_dto.SourceError) {
				require.Len(t, diagnostics, 2)

				for _, diag := range diagnostics {
					assert.Equal(t, "Q006", diag.Code)
					assert.Equal(t, querier_dto.SeverityError, diag.Severity)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			validator := newQueryValidator()
			diagnostics := validator.ValidateDuplicateNames(tt.queries)

			assert.Len(t, diagnostics, tt.expectedCount)
			tt.expectedDiagnostic(t, diagnostics)
		})
	}
}

func TestCommandName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		command  querier_dto.QueryCommand
		expected string
	}{
		{
			name:     "QueryCommandOne returns one",
			command:  querier_dto.QueryCommandOne,
			expected: "one",
		},
		{
			name:     "QueryCommandMany returns many",
			command:  querier_dto.QueryCommandMany,
			expected: "many",
		},
		{
			name:     "QueryCommandExec returns exec",
			command:  querier_dto.QueryCommandExec,
			expected: "exec",
		},
		{
			name:     "QueryCommandExecResult returns execresult",
			command:  querier_dto.QueryCommandExecResult,
			expected: "execresult",
		},
		{
			name:     "QueryCommandExecRows returns execrows",
			command:  querier_dto.QueryCommandExecRows,
			expected: "execrows",
		},
		{
			name:     "QueryCommandBatch returns batch",
			command:  querier_dto.QueryCommandBatch,
			expected: "batch",
		},
		{
			name:     "QueryCommandStream returns stream",
			command:  querier_dto.QueryCommandStream,
			expected: "stream",
		},
		{
			name:     "QueryCommandCopyFrom returns copyfrom",
			command:  querier_dto.QueryCommandCopyFrom,
			expected: "copyfrom",
		},
		{
			name:     "unknown command value returns unknown",
			command:  querier_dto.QueryCommand(255),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, commandName(tt.command))
		})
	}
}
