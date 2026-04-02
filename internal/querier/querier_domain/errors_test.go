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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestCatalogueError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *CatalogueError
		expected string
	}{
		{
			name: "line greater than zero includes line number",
			err: &CatalogueError{
				Filename: "0001_create_users.up.sql",
				Line:     42,
				Message:  "duplicate column definition",
			},
			expected: "migration 0001_create_users.up.sql:42: duplicate column definition",
		},
		{
			name: "line equal to zero omits line number",
			err: &CatalogueError{
				Filename: "0002_add_index.up.sql",
				Line:     0,
				Message:  "file could not be parsed",
			},
			expected: "migration 0002_add_index.up.sql: file could not be parsed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestCatalogueError_Unwrap(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("underlying cause")

	tests := []struct {
		name     string
		cause    error
		expected error
	}{
		{
			name:     "with cause returns the cause error",
			cause:    sentinel,
			expected: sentinel,
		},
		{
			name:     "with nil cause returns nil",
			cause:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			catalogueErr := &CatalogueError{
				Filename: "test.sql",
				Message:  "something went wrong",
				Cause:    tt.cause,
			}

			assert.Equal(t, tt.expected, catalogueErr.Unwrap())
		})
	}
}

func TestNewCatalogueError(t *testing.T) {
	t.Parallel()

	cause := errors.New("parse failure")
	err := NewCatalogueError("0003_alter_table.up.sql", 17, 3, "unexpected token", cause)

	require.NotNil(t, err)
	assert.Equal(t, "0003_alter_table.up.sql", err.Filename)
	assert.Equal(t, 17, err.Line)
	assert.Equal(t, 3, err.MigrationIndex)
	assert.Equal(t, "unexpected token", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestQueryAnalysisError_Error(t *testing.T) {
	t.Parallel()

	diagnostics := []querier_dto.SourceError{
		{Filename: "queries.sql", Message: "unknown column"},
		{Filename: "queries.sql", Message: "type mismatch"},
		{Filename: "queries.sql", Message: "ambiguous reference"},
	}

	err := &QueryAnalysisError{
		Filename:    "queries.sql",
		Diagnostics: diagnostics,
	}

	expected := "found 3 analysis errors in queries.sql"
	assert.Equal(t, expected, err.Error())
}

func TestNewQueryAnalysisError(t *testing.T) {
	t.Parallel()

	diagnostics := []querier_dto.SourceError{
		{Filename: "get_user.sql", Message: "column not found", Code: "Q001"},
	}

	err := NewQueryAnalysisError("get_user.sql", diagnostics)

	require.NotNil(t, err)
	assert.Equal(t, "get_user.sql", err.Filename)
	assert.Len(t, err.Diagnostics, 1)
	assert.Equal(t, diagnostics, err.Diagnostics)
}

func TestDirectiveSyntaxError_Error(t *testing.T) {
	t.Parallel()

	err := &DirectiveSyntaxError{
		Filename: "list_users.sql",
		Line:     5,
		Column:   12,
		Code:     "Q007",
		Message:  "malformed piko.name directive",
	}

	expected := "list_users.sql:5:12: Q007 malformed piko.name directive"
	assert.Equal(t, expected, err.Error())
}

func TestNewDirectiveSyntaxError(t *testing.T) {
	t.Parallel()

	err := NewDirectiveSyntaxError("insert_user.sql", 10, 3, "expected value after '='")

	require.NotNil(t, err)
	assert.Equal(t, "insert_user.sql", err.Filename)
	assert.Equal(t, 10, err.Line)
	assert.Equal(t, 3, err.Column)
	assert.Equal(t, "expected value after '='", err.Message)
	assert.Equal(t, "Q007", err.Code, "code should always be Q007")
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "ErrMissingEnginePort has correct message",
			err:             ErrMissingEnginePort,
			expectedMessage: "querier service requires an engine port",
		},
		{
			name:            "ErrMissingEmitterPort has correct message",
			err:             ErrMissingEmitterPort,
			expectedMessage: "querier service requires a code emitter port",
		},
		{
			name:            "ErrMissingFileReaderPort has correct message",
			err:             ErrMissingFileReaderPort,
			expectedMessage: "querier service requires a file reader port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tt.err)
			assert.Equal(t, tt.expectedMessage, tt.err.Error())

			wrapped := fmt.Errorf("wrapped: %w", tt.err)
			assert.True(t, errors.Is(wrapped, tt.err))
		})
	}
}
