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

func TestCommandOutputPass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		command       querier_dto.QueryCommand
		outputColumns []querier_dto.OutputColumn
		expectedCount int
	}{
		{
			name:    ":one with output columns produces no diagnostic",
			command: querier_dto.QueryCommandOne,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 0,
		},
		{
			name:          ":one without columns produces warning",
			command:       querier_dto.QueryCommandOne,
			outputColumns: nil,
			expectedCount: 1,
		},
		{
			name:    ":exec with columns produces warning",
			command: querier_dto.QueryCommandExec,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 1,
		},
		{
			name:          ":exec without columns produces no diagnostic",
			command:       querier_dto.QueryCommandExec,
			outputColumns: nil,
			expectedCount: 0,
		},
		{
			name:    ":many with columns produces no diagnostic",
			command: querier_dto.QueryCommandMany,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
				{Name: "email"},
			},
			expectedCount: 0,
		},
		{
			name:    ":stream with columns produces no diagnostic",
			command: querier_dto.QueryCommandStream,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 0,
		},
		{
			name:          ":many without columns produces warning",
			command:       querier_dto.QueryCommandMany,
			outputColumns: nil,
			expectedCount: 1,
		},
		{
			name:    ":execresult with columns produces warning",
			command: querier_dto.QueryCommandExecResult,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 1,
		},
		{
			name:    ":execrows with columns produces warning",
			command: querier_dto.QueryCommandExecRows,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pass := &commandOutputPass{}
			context := &diagnosticContext{
				Filename: "test.sql",
				Query: &querier_dto.AnalysedQuery{
					Name:          "TestQuery",
					Command:       tt.command,
					OutputColumns: tt.outputColumns,
					Line:          3,
				},
				RawAnalysis: &querier_dto.RawQueryAnalysis{},
			}

			diagnostics := pass.Analyse(context)

			require.Len(t, diagnostics, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, "Q009", diagnostics[0].Code)
				assert.Equal(t, querier_dto.SeverityWarning, diagnostics[0].Severity)
				assert.Contains(t, diagnostics[0].Message, `"TestQuery"`)
			}
		})
	}
}
