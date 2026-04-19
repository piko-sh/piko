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

func TestCommandOutputPassConflictDoNothingReturning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		command      querier_dto.QueryCommand
		sql          string
		hasReturning bool
		optional     bool
		wantWarning  bool
	}{
		{
			name:         "one with DO NOTHING and RETURNING warns",
			command:      querier_dto.QueryCommandOne,
			sql:          "INSERT INTO t (id) VALUES ($1) ON CONFLICT (id) DO NOTHING RETURNING id",
			hasReturning: true,
			wantWarning:  true,
		},
		{
			name:         "optional suppresses the warning for one with DO NOTHING and RETURNING",
			command:      querier_dto.QueryCommandOne,
			sql:          "INSERT INTO t (id) VALUES ($1) ON CONFLICT (id) DO NOTHING RETURNING id",
			hasReturning: true,
			optional:     true,
			wantWarning:  false,
		},
		{
			name:         "DO UPDATE does not warn",
			command:      querier_dto.QueryCommandOne,
			sql:          "INSERT INTO t (id) VALUES ($1) ON CONFLICT (id) DO UPDATE SET id = $1 RETURNING id",
			hasReturning: true,
			wantWarning:  false,
		},
		{
			name:         "many with DO NOTHING does not warn",
			command:      querier_dto.QueryCommandMany,
			sql:          "INSERT INTO t (id) VALUES ($1) ON CONFLICT (id) DO NOTHING RETURNING id",
			hasReturning: true,
			wantWarning:  false,
		},
		{
			name:         "one with DO NOTHING but no RETURNING does not warn",
			command:      querier_dto.QueryCommandOne,
			sql:          "INSERT INTO t (id) VALUES ($1) ON CONFLICT (id) DO NOTHING",
			hasReturning: false,
			wantWarning:  false,
		},
		{
			name:         "lowercase do nothing across newlines and multiple spaces warns",
			command:      querier_dto.QueryCommandOne,
			sql:          "insert into t (id) values ($1)\non conflict (id)\ndo   nothing\nreturning id",
			hasReturning: true,
			wantWarning:  true,
		},
		{
			name:         "DO NOTHING only inside a string literal without ON CONFLICT does not warn",
			command:      querier_dto.QueryCommandOne,
			sql:          "INSERT INTO logs (action) VALUES ('DO NOTHING') RETURNING id",
			hasReturning: true,
			wantWarning:  false,
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
					SQL:           tt.sql,
					Optional:      tt.optional,
					OutputColumns: []querier_dto.OutputColumn{{Name: "id"}},
					Line:          3,
				},
				RawAnalysis: &querier_dto.RawQueryAnalysis{HasReturning: tt.hasReturning},
			}

			var found bool
			for _, diagnostic := range pass.Analyse(context) {
				if diagnostic.Code == querier_dto.CodeConflictDoNothingReturning {
					found = true
					assert.Equal(t, querier_dto.SeverityWarning, diagnostic.Severity)
					assert.Contains(t, diagnostic.Message, "sql.ErrNoRows")
				}
			}
			assert.Equal(t, tt.wantWarning, found)
		})
	}
}

func TestCommandOutputPassOptionalMisuse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		command        querier_dto.QueryCommand
		optional       bool
		isDynamic      bool
		dynamicRuntime bool
		wantError      bool
	}{
		{
			name:      "optional on command:many is rejected",
			command:   querier_dto.QueryCommandMany,
			optional:  true,
			wantError: true,
		},
		{
			name:      "optional on a command:one with dynamic predicate parameters is rejected",
			command:   querier_dto.QueryCommandOne,
			optional:  true,
			isDynamic: true,
			wantError: true,
		},
		{
			name:           "optional on a command:one with dynamic runtime builder is rejected",
			command:        querier_dto.QueryCommandOne,
			optional:       true,
			dynamicRuntime: true,
			wantError:      true,
		},
		{
			name:      "optional on a static command:one is valid",
			command:   querier_dto.QueryCommandOne,
			optional:  true,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pass := &commandOutputPass{}
			context := &diagnosticContext{
				Filename: "test.sql",
				Query: &querier_dto.AnalysedQuery{
					Name:           "TestQuery",
					Command:        tt.command,
					Optional:       tt.optional,
					IsDynamic:      tt.isDynamic,
					DynamicRuntime: tt.dynamicRuntime,
					OutputColumns:  []querier_dto.OutputColumn{{Name: "id"}},
					Line:           3,
				},
				RawAnalysis: &querier_dto.RawQueryAnalysis{},
			}

			var found bool
			for _, diagnostic := range pass.Analyse(context) {
				if diagnostic.Code == querier_dto.CodeOptionalNonOneCommand {
					found = true
					assert.Equal(t, querier_dto.SeverityError, diagnostic.Severity)
					assert.Contains(t, diagnostic.Message, `"TestQuery"`)
				}
			}
			assert.Equal(t, tt.wantError, found)
		})
	}
}
