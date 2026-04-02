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

func TestChecksumMismatchError_Error(t *testing.T) {
	t.Parallel()

	err := &ChecksumMismatchError{
		Version:         1,
		Name:            "create_users",
		AppliedChecksum: "abc123",
		FileChecksum:    "def456",
	}

	msg := err.Error()
	assert.Contains(t, msg, "1")
	assert.Contains(t, msg, "create_users")
	assert.Contains(t, msg, "abc123")
	assert.Contains(t, msg, "def456")
	assert.Equal(t,
		"checksum mismatch for migration 1 (create_users): applied=abc123 file=def456",
		msg,
	)
}

func TestDownChecksumMismatchError_Error(t *testing.T) {
	t.Parallel()

	err := &DownChecksumMismatchError{
		Version:          2,
		Name:             "add_index",
		RecordedChecksum: "aaa111",
		FileChecksum:     "bbb222",
	}

	msg := err.Error()
	assert.Contains(t, msg, "2")
	assert.Contains(t, msg, "add_index")
	assert.Contains(t, msg, "aaa111")
	assert.Contains(t, msg, "bbb222")
	assert.Equal(t,
		"down checksum mismatch for migration 2 (add_index): recorded=aaa111 file=bbb222",
		msg,
	)
}

func TestMigrationExecutionError_Error(t *testing.T) {
	t.Parallel()

	cause := errors.New("syntax error near SELECT")

	tests := []struct {
		name           string
		direction      querier_dto.MigrationDirection
		expectedPrefix string
	}{
		{
			name:           "direction up starts with migration",
			direction:      querier_dto.MigrationDirectionUp,
			expectedPrefix: "migration",
		},
		{
			name:           "direction down starts with rollback",
			direction:      querier_dto.MigrationDirectionDown,
			expectedPrefix: "rollback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			execErr := &MigrationExecutionError{
				Version:   5,
				Name:      "add_column",
				Direction: tt.direction,
				Cause:     cause,
			}

			msg := execErr.Error()
			expected := fmt.Sprintf("%s 5 (add_column): syntax error near SELECT", tt.expectedPrefix)
			assert.Equal(t, expected, msg)
		})
	}
}

func TestMigrationExecutionError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection refused")
	err := &MigrationExecutionError{
		Version:   3,
		Name:      "create_index",
		Direction: querier_dto.MigrationDirectionUp,
		Cause:     cause,
	}

	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause), "errors.Is should match the wrapped cause")
}

func TestLockAcquisitionError_Error(t *testing.T) {
	t.Parallel()

	cause := errors.New("timeout exceeded")
	err := &LockAcquisitionError{
		Cause: cause,
	}

	expected := "acquiring migration lock: timeout exceeded"
	assert.Equal(t, expected, err.Error())
}

func TestLockAcquisitionError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection reset")
	err := &LockAcquisitionError{
		Cause: cause,
	}

	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause), "errors.Is should match the wrapped cause")
}

func TestMissingMigrationFileError_Error(t *testing.T) {
	t.Parallel()

	err := &MissingMigrationFileError{
		Version: 7,
		Name:    "drop_legacy_table",
	}

	msg := err.Error()
	assert.Contains(t, msg, "7")
	assert.Contains(t, msg, "drop_legacy_table")
	assert.Equal(t,
		"applied migration 7 (drop_legacy_table) has no corresponding file on disk",
		msg,
	)
}

func TestNoDownMigrationError_Error(t *testing.T) {
	t.Parallel()

	err := &NoDownMigrationError{
		Version: 12,
	}

	expected := "no down migration for version 12"
	assert.Equal(t, expected, err.Error())
	assert.Contains(t, err.Error(), "12")
}

func TestNoDownMigrationError_Is(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   error
		expected bool
	}{
		{
			name:     "matches ErrNoDownMigration sentinel",
			target:   ErrNoDownMigration,
			expected: true,
		},
		{
			name:     "does not match unrelated sentinel",
			target:   ErrLockNotAcquired,
			expected: false,
		},
		{
			name:     "does not match arbitrary error",
			target:   errors.New("something else entirely"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			noDownErr := &NoDownMigrationError{Version: 99}
			assert.Equal(t, tt.expected, errors.Is(noDownErr, tt.target))
		})
	}
}

func TestMigrationSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "ErrLockNotAcquired has correct message",
			err:             ErrLockNotAcquired,
			expectedMessage: "migration lock is already held",
		},
		{
			name:            "ErrNoDownMigration has correct message",
			err:             ErrNoDownMigration,
			expectedMessage: "no down migration file",
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
