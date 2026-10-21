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

package sqldb_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/sqldb/sqldb_dto"
)

func TestBuildMigrationServiceConfigAppliesOptionsInOrder(t *testing.T) {
	t.Parallel()

	var calls []string
	before := func(context.Context, sqldb_dto.MigrationHookContext) error {
		calls = append(calls, "before")
		return nil
	}
	after := func(context.Context, sqldb_dto.MigrationHookContext) error {
		calls = append(calls, "after")
		return nil
	}
	beforeRun := func(context.Context, sqldb_dto.MigrationRunHookContext) error { return nil }
	afterRun := func(context.Context, sqldb_dto.MigrationRunHookContext, int) error { return nil }

	config := BuildMigrationServiceConfig(
		WithNonBlockingLock(),
		WithBeforeMigration(before),
		WithAfterMigration(after),
		WithBeforeRun(beforeRun),
		WithAfterRun(afterRun),
	)

	require.True(t, config.NonBlockingLock)
	require.Len(t, config.BeforeMigration, 1)
	require.Len(t, config.AfterMigration, 1)
	require.Len(t, config.BeforeRun, 1)
	require.Len(t, config.AfterRun, 1)
	require.NoError(t, config.BeforeMigration[0](context.Background(), sqldb_dto.MigrationHookContext{}))
	require.NoError(t, config.AfterMigration[0](context.Background(), sqldb_dto.MigrationHookContext{}))
	require.Equal(t, []string{"before", "after"}, calls)
}

func TestBuildMigrationServiceConfigDefaults(t *testing.T) {
	t.Parallel()

	config := BuildMigrationServiceConfig()

	require.False(t, config.NonBlockingLock)
	require.Empty(t, config.BeforeMigration)
}

func TestTranslatedErrorMatchesBothVocabularies(t *testing.T) {
	t.Parallel()

	original := errors.New("toolchain: migration lock is already held")
	translated := &TranslatedError{Piko: ErrLockNotAcquired, Original: original}

	require.ErrorIs(t, translated, ErrLockNotAcquired)
	require.ErrorIs(t, translated, original)
	require.Equal(t, original.Error(), translated.Error())
}

func TestTranslatedErrorExposesTypedError(t *testing.T) {
	t.Parallel()

	typed := &ChecksumMismatchError{AppliedChecksum: "a", FileChecksum: "b", Name: "init", Version: 1}
	translated := &TranslatedError{Piko: typed, Original: errors.New("checksum mismatch")}

	var target *ChecksumMismatchError
	require.ErrorAs(t, translated, &target)
	require.Equal(t, int64(1), target.Version)
}

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")
	cases := []struct {
		err      error
		name     string
		expected string
	}{
		{name: "checksum", err: &ChecksumMismatchError{AppliedChecksum: "a", FileChecksum: "b", Name: "init", Version: 1}, expected: "checksum mismatch for migration 1 (init): applied=a file=b"},
		{name: "down checksum", err: &DownChecksumMismatchError{RecordedChecksum: "a", FileChecksum: "b", Name: "init", Version: 1}, expected: "down checksum mismatch for migration 1 (init): recorded=a file=b"},
		{name: "execution up", err: &MigrationExecutionError{Cause: cause, Name: "init", Version: 1, Direction: sqldb_dto.DirectionUp}, expected: "migration 1 (init): boom"},
		{name: "execution down", err: &MigrationExecutionError{Cause: cause, Name: "init", Version: 1, Direction: sqldb_dto.DirectionDown}, expected: "rollback 1 (init): boom"},
		{name: "lock", err: &LockAcquisitionError{Cause: cause}, expected: "acquiring migration lock: boom"},
		{name: "missing file", err: &MissingMigrationFileError{Name: "init", Version: 1}, expected: "applied migration 1 (init) has no corresponding file on disk"},
		{name: "no down", err: &NoDownMigrationError{Version: 3}, expected: "no down migration for version 3"},
		{name: "duplicate up", err: &DuplicateMigrationVersionError{FirstFilename: "a", SecondFilename: "b", Version: 2, Direction: sqldb_dto.DirectionUp}, expected: "duplicate up migration for version 2: a and b"},
		{name: "duplicate down", err: &DuplicateMigrationVersionError{FirstFilename: "a", SecondFilename: "b", Version: 2, Direction: sqldb_dto.DirectionDown}, expected: "duplicate down migration for version 2: a and b"},
		{name: "dirty", err: &DirtyMigrationError{Version: 4, LastStatement: 2}, expected: "migration 4 is dirty (last completed statement: 2); resolve manually or retry"},
		{name: "duplicate seed", err: &DuplicateSeedVersionError{FirstFilename: "a", SecondFilename: "b", Version: 5}, expected: "duplicate seed for version 5: a and b"},
		{name: "seed execution", err: &SeedExecutionError{Cause: cause, Name: "users", Version: 6}, expected: "seed 6 (users): boom"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, testCase.err.Error())
		})
	}
}

func TestUnwrapExposesCauses(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")

	require.ErrorIs(t, &MigrationExecutionError{Cause: cause}, cause)
	require.ErrorIs(t, &LockAcquisitionError{Cause: cause}, cause)
	require.ErrorIs(t, &SeedExecutionError{Cause: cause}, cause)
	require.ErrorIs(t, &NoDownMigrationError{Version: 1}, ErrNoDownMigration)
}
