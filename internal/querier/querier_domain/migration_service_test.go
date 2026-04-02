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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func testChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func setupMigrationService(
	t *testing.T,
	executor *mockMigrationExecutor,
	reader *mockFileReader,
	opts ...MigrationServiceOption,
) *migrationService {
	t.Helper()
	svc := NewMigrationService(executor, reader, "/migrations", opts...)
	concrete, ok := svc.(*migrationService)
	require.True(t, ok, "NewMigrationService must return *migrationService")
	return concrete
}

func threeFileMigrationReader() *mockFileReader {
	return &mockFileReader{
		dirs: map[string][]os.DirEntry{
			"/migrations": {
				&mockDirEntry{name: "001_create_users.up.sql"},
				&mockDirEntry{name: "001_create_users.down.sql"},
				&mockDirEntry{name: "002_create_posts.up.sql"},
				&mockDirEntry{name: "002_create_posts.down.sql"},
				&mockDirEntry{name: "003_add_email.up.sql"},
				&mockDirEntry{name: "003_add_email.down.sql"},
			},
		},
		files: map[string][]byte{
			"/migrations/001_create_users.up.sql":   []byte("CREATE TABLE users (id int);"),
			"/migrations/001_create_users.down.sql": []byte("DROP TABLE users;"),
			"/migrations/002_create_posts.up.sql":   []byte("CREATE TABLE posts (id int);"),
			"/migrations/002_create_posts.down.sql": []byte("DROP TABLE posts;"),
			"/migrations/003_add_email.up.sql":      []byte("ALTER TABLE users ADD COLUMN email text;"),
			"/migrations/003_add_email.down.sql":    []byte("ALTER TABLE users DROP COLUMN email;"),
		},
	}
}

func appliedFromReader(reader *mockFileReader, versions ...int64) []querier_dto.AppliedMigration {
	nameByVersion := map[int64]string{
		1: "create_users",
		2: "create_posts",
		3: "add_email",
	}
	filenameByVersion := map[int64]string{
		1: "/migrations/001_create_users.up.sql",
		2: "/migrations/002_create_posts.up.sql",
		3: "/migrations/003_add_email.up.sql",
	}
	downFilenameByVersion := map[int64]string{
		1: "/migrations/001_create_users.down.sql",
		2: "/migrations/002_create_posts.down.sql",
		3: "/migrations/003_add_email.down.sql",
	}

	result := make([]querier_dto.AppliedMigration, 0, len(versions))
	for _, version := range versions {
		upContent := reader.files[filenameByVersion[version]]
		downContent := reader.files[downFilenameByVersion[version]]
		result = append(result, querier_dto.AppliedMigration{
			Version:      version,
			Name:         nameByVersion[version],
			Checksum:     testChecksum(upContent),
			DownChecksum: testChecksum(downContent),
			AppliedAt:    time.Date(2026, 1, int(version), 0, 0, 0, 0, time.UTC),
		})
	}
	return result
}

func TestNewMigrationService(t *testing.T) {
	t.Parallel()

	t.Run("creates with required arguments", func(t *testing.T) {
		t.Parallel()
		executor := &mockMigrationExecutor{}
		reader := &mockFileReader{}

		svc := NewMigrationService(executor, reader, "/migrations")

		require.NotNil(t, svc, "service must not be nil")
		concrete, ok := svc.(*migrationService)
		require.True(t, ok)
		assert.Equal(t, "/migrations", concrete.directory)
		assert.False(t, concrete.nonBlockingLock, "non-blocking lock should default to false")
	})

	t.Run("options are applied", func(t *testing.T) {
		t.Parallel()
		executor := &mockMigrationExecutor{}
		reader := &mockFileReader{}

		svc := NewMigrationService(executor, reader, "/migrations", WithNonBlockingLock())

		concrete, ok := svc.(*migrationService)
		require.True(t, ok, "expected *migrationService")
		assert.True(t, concrete.nonBlockingLock, "WithNonBlockingLock should set the flag")
	})
}

func TestMigrationService_Up(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		reader       *mockFileReader
		applied      []querier_dto.AppliedMigration
		executorErr  error
		ensureErr    error
		appliedErr   error
		lockErr      error
		wantCount    int
		wantErr      bool
		wantErrType  any
		wantExecuted []int64
	}{
		{
			name:         "applies all pending migrations",
			reader:       threeFileMigrationReader(),
			applied:      nil,
			wantCount:    3,
			wantExecuted: []int64{1, 2, 3},
		},
		{
			name:         "skips already applied migrations",
			reader:       threeFileMigrationReader(),
			applied:      appliedFromReader(threeFileMigrationReader(), 1, 2),
			wantCount:    1,
			wantExecuted: []int64{3},
		},
		{
			name:      "no pending returns zero",
			reader:    threeFileMigrationReader(),
			applied:   appliedFromReader(threeFileMigrationReader(), 1, 2, 3),
			wantCount: 0,
		},
		{
			name:   "checksum mismatch returns error",
			reader: threeFileMigrationReader(),
			applied: []querier_dto.AppliedMigration{
				{
					Version:  1,
					Name:     "create_users",
					Checksum: "wrong-checksum",
				},
			},
			wantErr:     true,
			wantErrType: &ChecksumMismatchError{},
		},
		{
			name:        "executor error is propagated",
			reader:      threeFileMigrationReader(),
			applied:     nil,
			executorErr: errors.New("database exploded"),
			wantErr:     true,
		},
		{
			name:      "ensure migration table error is propagated",
			reader:    threeFileMigrationReader(),
			applied:   nil,
			ensureErr: errors.New("cannot create table"),
			wantErr:   true,
		},
		{
			name:       "applied versions error is propagated",
			reader:     threeFileMigrationReader(),
			applied:    nil,
			appliedErr: errors.New("cannot read versions"),
			wantErr:    true,
		},
		{
			name:        "lock acquisition failure returns LockAcquisitionError",
			reader:      threeFileMigrationReader(),
			applied:     nil,
			lockErr:     errors.New("lock held"),
			wantErr:     true,
			wantErrType: &LockAcquisitionError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var executedVersions []int64
			executor := &mockMigrationExecutor{
				ensureMigrationTableFn: func(_ context.Context) error {
					return tt.ensureErr
				},
				appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
					if tt.appliedErr != nil {
						return nil, tt.appliedErr
					}
					return tt.applied, nil
				},
				acquireLockFn: func(_ context.Context) error {
					return tt.lockErr
				},
				releaseLockFn: func(_ context.Context) error {
					return nil
				},
				executeMigrationFn: func(_ context.Context, migration querier_dto.MigrationRecord, _ querier_dto.MigrationDirection, _ bool) error {
					if tt.executorErr != nil {
						return tt.executorErr
					}
					executedVersions = append(executedVersions, migration.Version)
					return nil
				},
			}

			svc := setupMigrationService(t, executor, tt.reader)
			count, err := svc.Up(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorAs(t, err, &tt.wantErrType,
						"error should be of the expected type")
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
			if tt.wantExecuted != nil {
				assert.Equal(t, tt.wantExecuted, executedVersions,
					"migrations should be executed in version order")
			}
		})
	}
}

func TestMigrationService_Up_lock_acquired_and_released(t *testing.T) {
	t.Parallel()

	reader := threeFileMigrationReader()
	var lockAcquired, lockReleased bool

	executor := &mockMigrationExecutor{
		appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
			return nil, nil
		},
		acquireLockFn: func(_ context.Context) error {
			lockAcquired = true
			return nil
		},
		releaseLockFn: func(_ context.Context) error {
			lockReleased = true
			return nil
		},
	}

	svc := setupMigrationService(t, executor, reader)
	_, err := svc.Up(context.Background())

	require.NoError(t, err)
	assert.True(t, lockAcquired, "lock should be acquired during Up")
	assert.True(t, lockReleased, "lock should be released after Up")
}

func TestMigrationService_Down(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		reader         *mockFileReader
		applied        []querier_dto.AppliedMigration
		steps          int
		executorErr    error
		wantCount      int
		wantErr        bool
		wantErrType    any
		wantRolledBack []int64
	}{
		{
			name:           "rolls back N migrations in reverse order",
			reader:         threeFileMigrationReader(),
			applied:        appliedFromReader(threeFileMigrationReader(), 1, 2, 3),
			steps:          2,
			wantCount:      2,
			wantRolledBack: []int64{3, 2},
		},
		{
			name:           "rolls back all when N exceeds applied count",
			reader:         threeFileMigrationReader(),
			applied:        appliedFromReader(threeFileMigrationReader(), 1, 2),
			steps:          5,
			wantCount:      2,
			wantRolledBack: []int64{2, 1},
		},
		{
			name:      "returns zero when nothing applied",
			reader:    threeFileMigrationReader(),
			applied:   nil,
			steps:     3,
			wantCount: 0,
		},
		{
			name: "no down file returns NoDownMigrationError",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "001_create_users.up.sql"},
					},
				},
				files: map[string][]byte{
					"/migrations/001_create_users.up.sql": []byte("CREATE TABLE users (id int);"),
				},
			},
			applied: []querier_dto.AppliedMigration{
				{
					Version:  1,
					Name:     "create_users",
					Checksum: testChecksum([]byte("CREATE TABLE users (id int);")),
				},
			},
			steps:       1,
			wantErr:     true,
			wantErrType: &NoDownMigrationError{},
		},
		{
			name:        "executor error during rollback is propagated",
			reader:      threeFileMigrationReader(),
			applied:     appliedFromReader(threeFileMigrationReader(), 1, 2, 3),
			steps:       1,
			executorErr: errors.New("rollback failed"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var rolledBackVersions []int64
			executor := &mockMigrationExecutor{
				appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
					return tt.applied, nil
				},
				acquireLockFn: func(_ context.Context) error {
					return nil
				},
				releaseLockFn: func(_ context.Context) error {
					return nil
				},
				executeMigrationFn: func(_ context.Context, migration querier_dto.MigrationRecord, _ querier_dto.MigrationDirection, _ bool) error {
					if tt.executorErr != nil {
						return tt.executorErr
					}
					rolledBackVersions = append(rolledBackVersions, migration.Version)
					return nil
				},
			}

			svc := setupMigrationService(t, executor, tt.reader)
			count, err := svc.Down(context.Background(), tt.steps)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorAs(t, err, &tt.wantErrType,
						"error should be of the expected type")
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
			if tt.wantRolledBack != nil {
				assert.Equal(t, tt.wantRolledBack, rolledBackVersions,
					"migrations should be rolled back in reverse version order")
			}
		})
	}
}

func TestMigrationService_UpTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		reader        *mockFileReader
		applied       []querier_dto.AppliedMigration
		targetVersion int64
		wantCount     int
		wantExecuted  []int64
	}{
		{
			name:          "applies up to target version",
			reader:        threeFileMigrationReader(),
			applied:       nil,
			targetVersion: 2,
			wantCount:     2,
			wantExecuted:  []int64{1, 2},
		},
		{
			name:          "target below all pending applies none",
			reader:        threeFileMigrationReader(),
			applied:       nil,
			targetVersion: 0,
			wantCount:     0,
		},
		{
			name:          "target above all pending applies all",
			reader:        threeFileMigrationReader(),
			applied:       nil,
			targetVersion: 100,
			wantCount:     3,
			wantExecuted:  []int64{1, 2, 3},
		},
		{
			name:          "respects already applied when targeting",
			reader:        threeFileMigrationReader(),
			applied:       appliedFromReader(threeFileMigrationReader(), 1),
			targetVersion: 2,
			wantCount:     1,
			wantExecuted:  []int64{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var executedVersions []int64
			executor := &mockMigrationExecutor{
				appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
					return tt.applied, nil
				},
				acquireLockFn: func(_ context.Context) error {
					return nil
				},
				releaseLockFn: func(_ context.Context) error {
					return nil
				},
				executeMigrationFn: func(_ context.Context, migration querier_dto.MigrationRecord, _ querier_dto.MigrationDirection, _ bool) error {
					executedVersions = append(executedVersions, migration.Version)
					return nil
				},
			}

			svc := setupMigrationService(t, executor, tt.reader)
			count, err := svc.UpTo(context.Background(), tt.targetVersion)

			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
			if tt.wantExecuted != nil {
				assert.Equal(t, tt.wantExecuted, executedVersions)
			}
		})
	}
}

func TestMigrationService_DownTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		reader         *mockFileReader
		applied        []querier_dto.AppliedMigration
		targetVersion  int64
		wantCount      int
		wantRolledBack []int64
	}{
		{
			name:           "rolls back to target version exclusive",
			reader:         threeFileMigrationReader(),
			applied:        appliedFromReader(threeFileMigrationReader(), 1, 2, 3),
			targetVersion:  1,
			wantCount:      2,
			wantRolledBack: []int64{3, 2},
		},
		{
			name:          "target at highest applied rolls back nothing",
			reader:        threeFileMigrationReader(),
			applied:       appliedFromReader(threeFileMigrationReader(), 1, 2, 3),
			targetVersion: 3,
			wantCount:     0,
		},
		{
			name:           "target below all applied rolls back everything",
			reader:         threeFileMigrationReader(),
			applied:        appliedFromReader(threeFileMigrationReader(), 1, 2, 3),
			targetVersion:  0,
			wantCount:      3,
			wantRolledBack: []int64{3, 2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var rolledBackVersions []int64
			executor := &mockMigrationExecutor{
				appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
					return tt.applied, nil
				},
				acquireLockFn: func(_ context.Context) error {
					return nil
				},
				releaseLockFn: func(_ context.Context) error {
					return nil
				},
				executeMigrationFn: func(_ context.Context, migration querier_dto.MigrationRecord, _ querier_dto.MigrationDirection, _ bool) error {
					rolledBackVersions = append(rolledBackVersions, migration.Version)
					return nil
				},
			}

			svc := setupMigrationService(t, executor, tt.reader)
			count, err := svc.DownTo(context.Background(), tt.targetVersion)

			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
			if tt.wantRolledBack != nil {
				assert.Equal(t, tt.wantRolledBack, rolledBackVersions,
					"migrations should be rolled back in reverse version order")
			}
		})
	}
}

func TestMigrationService_Status(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reader     *mockFileReader
		applied    []querier_dto.AppliedMigration
		appliedErr error
		ensureErr  error
		wantCount  int
		wantErr    bool
		verify     func(t *testing.T, statuses []querier_dto.MigrationStatus)
	}{
		{
			name:      "combines files and applied state",
			reader:    threeFileMigrationReader(),
			applied:   appliedFromReader(threeFileMigrationReader(), 1, 2),
			wantCount: 3,
			verify: func(t *testing.T, statuses []querier_dto.MigrationStatus) {
				t.Helper()

				assert.True(t, statuses[0].Applied, "version 1 should be applied")
				assert.True(t, statuses[0].ChecksumMatch, "version 1 checksum should match")
				assert.True(t, statuses[0].HasDownMigration, "version 1 should have down migration")
				assert.Equal(t, int64(1), statuses[0].Version)
				assert.Equal(t, "create_users", statuses[0].Name)

				assert.True(t, statuses[1].Applied, "version 2 should be applied")
				assert.True(t, statuses[1].ChecksumMatch, "version 2 checksum should match")

				assert.False(t, statuses[2].Applied, "version 3 should be pending")
				assert.False(t, statuses[2].ChecksumMatch, "version 3 checksum match should be false when not applied")
				assert.True(t, statuses[2].HasDownMigration, "version 3 should have down migration")
			},
		},
		{
			name:   "checksum mismatch flagged correctly",
			reader: threeFileMigrationReader(),
			applied: []querier_dto.AppliedMigration{
				{
					Version:  1,
					Name:     "create_users",
					Checksum: "wrong-checksum",
				},
			},
			wantCount: 3,
			verify: func(t *testing.T, statuses []querier_dto.MigrationStatus) {
				t.Helper()
				assert.True(t, statuses[0].Applied, "version 1 should be applied")
				assert.False(t, statuses[0].ChecksumMatch, "version 1 checksum should NOT match")
			},
		},
		{
			name:      "ensure migration table error is propagated",
			reader:    threeFileMigrationReader(),
			ensureErr: errors.New("table creation failed"),
			wantErr:   true,
		},
		{
			name:       "applied versions error is propagated",
			reader:     threeFileMigrationReader(),
			appliedErr: errors.New("query failed"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := &mockMigrationExecutor{
				ensureMigrationTableFn: func(_ context.Context) error {
					return tt.ensureErr
				},
				appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
					if tt.appliedErr != nil {
						return nil, tt.appliedErr
					}
					return tt.applied, nil
				},
			}

			svc := setupMigrationService(t, executor, tt.reader)
			statuses, err := svc.Status(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, statuses, tt.wantCount)
			if tt.verify != nil {
				tt.verify(t, statuses)
			}
		})
	}
}

func TestMigrationService_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		reader      *mockFileReader
		applied     []querier_dto.AppliedMigration
		ensureErr   error
		appliedErr  error
		wantErr     bool
		wantErrType any
	}{
		{
			name:    "all checksums match returns nil",
			reader:  threeFileMigrationReader(),
			applied: appliedFromReader(threeFileMigrationReader(), 1, 2, 3),
			wantErr: false,
		},
		{
			name:   "checksum mismatch returns ChecksumMismatchError",
			reader: threeFileMigrationReader(),
			applied: []querier_dto.AppliedMigration{
				{
					Version:  1,
					Name:     "create_users",
					Checksum: "tampered-checksum",
				},
			},
			wantErr:     true,
			wantErrType: &ChecksumMismatchError{},
		},
		{
			name: "missing file for applied migration returns MissingMigrationFileError",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "001_create_users.up.sql"},
					},
				},
				files: map[string][]byte{
					"/migrations/001_create_users.up.sql": []byte("CREATE TABLE users (id int);"),
				},
			},
			applied: []querier_dto.AppliedMigration{
				{
					Version:  1,
					Name:     "create_users",
					Checksum: testChecksum([]byte("CREATE TABLE users (id int);")),
				},
				{
					Version:  2,
					Name:     "create_posts",
					Checksum: "some-checksum",
				},
			},
			wantErr:     true,
			wantErrType: &MissingMigrationFileError{},
		},
		{
			name:      "ensure migration table error is propagated",
			reader:    threeFileMigrationReader(),
			ensureErr: errors.New("broken table"),
			wantErr:   true,
		},
		{
			name:       "applied versions error is propagated",
			reader:     threeFileMigrationReader(),
			appliedErr: errors.New("query broken"),
			wantErr:    true,
		},
		{
			name:    "no applied migrations validates successfully",
			reader:  threeFileMigrationReader(),
			applied: nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := &mockMigrationExecutor{
				ensureMigrationTableFn: func(_ context.Context) error {
					return tt.ensureErr
				},
				appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
					if tt.appliedErr != nil {
						return nil, tt.appliedErr
					}
					return tt.applied, nil
				},
			}

			svc := setupMigrationService(t, executor, tt.reader)
			err := svc.Validate(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorAs(t, err, &tt.wantErrType,
						"error should be of the expected type")
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMigrationService_Hooks(t *testing.T) {
	t.Parallel()

	t.Run("before migration hook called for each migration", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		var hookedVersions []int64

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		hook := WithBeforeMigration(func(_ context.Context, hookCtx MigrationHookContext) error {
			hookedVersions = append(hookedVersions, hookCtx.Version)
			assert.Equal(t, querier_dto.MigrationDirectionUp, hookCtx.Direction)
			return nil
		})

		svc := setupMigrationService(t, executor, reader, hook)
		count, err := svc.Up(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.Equal(t, []int64{1, 2, 3}, hookedVersions,
			"before hook should fire for each pending migration in order")
	})

	t.Run("after migration hook called for each migration", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		var hookedVersions []int64

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		hook := WithAfterMigration(func(_ context.Context, hookCtx MigrationHookContext) error {
			hookedVersions = append(hookedVersions, hookCtx.Version)
			return nil
		})

		svc := setupMigrationService(t, executor, reader, hook)
		count, err := svc.Up(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.Equal(t, []int64{1, 2, 3}, hookedVersions,
			"after hook should fire for each applied migration in order")
	})

	t.Run("before run hook called before execution", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		var hookContext MigrationRunHookContext
		hookCalled := false

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		hook := WithBeforeRun(func(_ context.Context, ctx MigrationRunHookContext) error {
			hookCalled = true
			hookContext = ctx
			return nil
		})

		svc := setupMigrationService(t, executor, reader, hook)
		_, err := svc.Up(context.Background())

		require.NoError(t, err)
		assert.True(t, hookCalled, "before run hook should be called")
		assert.Equal(t, 3, hookContext.PendingCount)
		assert.Equal(t, querier_dto.MigrationDirectionUp, hookContext.Direction)
		assert.Equal(t, []int64{1, 2, 3}, hookContext.PendingVersions)
	})

	t.Run("after run hook called after execution", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		var appliedCount int
		hookCalled := false

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		hook := WithAfterRun(func(_ context.Context, _ MigrationRunHookContext, applied int) error {
			hookCalled = true
			appliedCount = applied
			return nil
		})

		svc := setupMigrationService(t, executor, reader, hook)
		_, err := svc.Up(context.Background())

		require.NoError(t, err)
		assert.True(t, hookCalled, "after run hook should be called")
		assert.Equal(t, 3, appliedCount, "after run hook should receive correct applied count")
	})

	t.Run("before migration hook error cancels the run", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		hookErr := errors.New("hook vetoed the migration")

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		hook := WithBeforeMigration(func(_ context.Context, _ MigrationHookContext) error {
			return hookErr
		})

		svc := setupMigrationService(t, executor, reader, hook)
		count, err := svc.Up(context.Background())

		require.ErrorIs(t, err, hookErr)
		assert.Equal(t, 0, count, "no migrations should be applied when hook returns error")
	})

	t.Run("before run hook error cancels the run", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		hookErr := errors.New("run vetoed")

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		hook := WithBeforeRun(func(_ context.Context, _ MigrationRunHookContext) error {
			return hookErr
		})

		svc := setupMigrationService(t, executor, reader, hook)
		count, err := svc.Up(context.Background())

		require.ErrorIs(t, err, hookErr)
		assert.Equal(t, 0, count)
	})

	t.Run("hooks fire during rollback with correct direction", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		var beforeVersions []int64
		var afterVersions []int64
		var runDirection querier_dto.MigrationDirection

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return appliedFromReader(threeFileMigrationReader(), 1, 2, 3), nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		svc := setupMigrationService(t, executor, reader,
			WithBeforeMigration(func(_ context.Context, hookCtx MigrationHookContext) error {
				beforeVersions = append(beforeVersions, hookCtx.Version)
				assert.Equal(t, querier_dto.MigrationDirectionDown, hookCtx.Direction)
				return nil
			}),
			WithAfterMigration(func(_ context.Context, hookCtx MigrationHookContext) error {
				afterVersions = append(afterVersions, hookCtx.Version)
				return nil
			}),
			WithBeforeRun(func(_ context.Context, ctx MigrationRunHookContext) error {
				runDirection = ctx.Direction
				return nil
			}),
		)

		count, err := svc.Down(context.Background(), 2)

		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Equal(t, []int64{3, 2}, beforeVersions)
		assert.Equal(t, []int64{3, 2}, afterVersions)
		assert.Equal(t, querier_dto.MigrationDirectionDown, runDirection)
	})
}

func TestMigrationService_NonBlockingLock(t *testing.T) {
	t.Parallel()

	t.Run("uses TryAcquireLock instead of AcquireLock", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		var acquireCalled atomic.Bool
		var tryAcquireCalled atomic.Bool

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				acquireCalled.Store(true)
				return nil
			},
			tryAcquireLockFn: func(_ context.Context) error {
				tryAcquireCalled.Store(true)
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		svc := setupMigrationService(t, executor, reader, WithNonBlockingLock())
		_, err := svc.Up(context.Background())

		require.NoError(t, err)
		assert.True(t, tryAcquireCalled.Load(),
			"TryAcquireLock should be called when non-blocking lock is configured")
		assert.False(t, acquireCalled.Load(),
			"AcquireLock should NOT be called when non-blocking lock is configured")
	})

	t.Run("blocking lock uses AcquireLock by default", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		var acquireCalled atomic.Bool
		var tryAcquireCalled atomic.Bool

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				acquireCalled.Store(true)
				return nil
			},
			tryAcquireLockFn: func(_ context.Context) error {
				tryAcquireCalled.Store(true)
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		svc := setupMigrationService(t, executor, reader)
		_, err := svc.Up(context.Background())

		require.NoError(t, err)
		assert.True(t, acquireCalled.Load(),
			"AcquireLock should be called when non-blocking lock is NOT configured")
		assert.False(t, tryAcquireCalled.Load(),
			"TryAcquireLock should NOT be called when non-blocking lock is NOT configured")
	})

	t.Run("non-blocking lock failure on Down returns LockAcquisitionError", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return appliedFromReader(threeFileMigrationReader(), 1, 2), nil
			},
			tryAcquireLockFn: func(_ context.Context) error {
				return ErrLockNotAcquired
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
		}

		svc := setupMigrationService(t, executor, reader, WithNonBlockingLock())
		_, err := svc.Down(context.Background(), 1)

		require.Error(t, err)
		var lockErr *LockAcquisitionError
		assert.ErrorAs(t, err, &lockErr, "should wrap as LockAcquisitionError")
		assert.ErrorIs(t, lockErr, ErrLockNotAcquired)
	})
}

func TestMigrationService_NoTransactionDirective(t *testing.T) {
	t.Parallel()

	t.Run("migration with no-transaction directive runs without transaction", func(t *testing.T) {
		t.Parallel()

		noTxContent := []byte("-- piko:no-transaction\nCREATE INDEX CONCURRENTLY idx ON users (email);")
		reader := &mockFileReader{
			dirs: map[string][]os.DirEntry{
				"/migrations": {
					&mockDirEntry{name: "001_add_index.up.sql"},
				},
			},
			files: map[string][]byte{
				"/migrations/001_add_index.up.sql": noTxContent,
			},
		}

		var capturedUseTransaction bool
		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
			executeMigrationFn: func(_ context.Context, _ querier_dto.MigrationRecord, _ querier_dto.MigrationDirection, useTransaction bool) error {
				capturedUseTransaction = useTransaction
				return nil
			},
		}

		svc := setupMigrationService(t, executor, reader)
		count, err := svc.Up(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.False(t, capturedUseTransaction,
			"migration with no-transaction directive should run without transaction wrapping")
	})

	t.Run("migration without directive runs with transaction", func(t *testing.T) {
		t.Parallel()

		reader := &mockFileReader{
			dirs: map[string][]os.DirEntry{
				"/migrations": {
					&mockDirEntry{name: "001_create_table.up.sql"},
				},
			},
			files: map[string][]byte{
				"/migrations/001_create_table.up.sql": []byte("CREATE TABLE foo (id int);"),
			},
		}

		var capturedUseTransaction bool
		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
			executeMigrationFn: func(_ context.Context, _ querier_dto.MigrationRecord, _ querier_dto.MigrationDirection, useTransaction bool) error {
				capturedUseTransaction = useTransaction
				return nil
			},
		}

		svc := setupMigrationService(t, executor, reader)
		count, err := svc.Up(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.True(t, capturedUseTransaction,
			"migration without the directive should run with transaction wrapping")
	})
}

func TestMigrationService_DownChecksumMismatch(t *testing.T) {
	t.Parallel()

	reader := threeFileMigrationReader()

	applied := appliedFromReader(threeFileMigrationReader(), 1, 2, 3)
	applied[2].DownChecksum = "tampered-down-checksum"

	executor := &mockMigrationExecutor{
		appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
			return applied, nil
		},
		acquireLockFn: func(_ context.Context) error {
			return nil
		},
		releaseLockFn: func(_ context.Context) error {
			return nil
		},
	}

	svc := setupMigrationService(t, executor, reader)
	_, err := svc.Down(context.Background(), 1)

	require.Error(t, err)
	var mismatchErr *DownChecksumMismatchError
	assert.ErrorAs(t, err, &mismatchErr,
		"should return DownChecksumMismatchError when the down file checksum differs")
	assert.Equal(t, int64(3), mismatchErr.Version)
}

func TestMigrationService_ContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("cancelled context during Up stops early", func(t *testing.T) {
		t.Parallel()

		reader := threeFileMigrationReader()
		var executedCount int

		ctx, cancel := context.WithCancel(context.Background())

		executor := &mockMigrationExecutor{
			appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {
				return nil, nil
			},
			acquireLockFn: func(_ context.Context) error {
				return nil
			},
			releaseLockFn: func(_ context.Context) error {
				return nil
			},
			executeMigrationFn: func(_ context.Context, _ querier_dto.MigrationRecord, _ querier_dto.MigrationDirection, _ bool) error {
				executedCount++
				if executedCount == 1 {
					cancel()
				}
				return nil
			},
		}

		svc := setupMigrationService(t, executor, reader)
		count, err := svc.Up(ctx)

		require.Error(t, err)
		assert.Equal(t, 1, count, "should stop after the first migration")
	})
}

func TestMigrationService_MigrationRecordContent(t *testing.T) {
	t.Parallel()

	reader := threeFileMigrationReader()
	var capturedRecord querier_dto.MigrationRecord

	executor := &mockMigrationExecutor{
		appliedVersionsFn: func(_ context.Context) ([]querier_dto.AppliedMigration, error) {

			return appliedFromReader(threeFileMigrationReader(), 1, 2), nil
		},
		acquireLockFn: func(_ context.Context) error {
			return nil
		},
		releaseLockFn: func(_ context.Context) error {
			return nil
		},
		executeMigrationFn: func(_ context.Context, migration querier_dto.MigrationRecord, _ querier_dto.MigrationDirection, _ bool) error {
			capturedRecord = migration
			return nil
		},
	}

	svc := setupMigrationService(t, executor, reader)
	count, err := svc.Up(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, int64(3), capturedRecord.Version)
	assert.Equal(t, "add_email", capturedRecord.Name)
	assert.Equal(t,
		testChecksum([]byte("ALTER TABLE users ADD COLUMN email text;")),
		capturedRecord.Checksum,
		"up checksum should match the file content",
	)
	assert.Equal(t,
		testChecksum([]byte("ALTER TABLE users DROP COLUMN email;")),
		capturedRecord.DownChecksum,
		"down checksum should be included in the record",
	)
}

func TestMigrationService_ReadDirError(t *testing.T) {
	t.Parallel()

	reader := &mockFileReader{
		readDirErr: map[string]error{
			"/migrations": errors.New("permission denied"),
		},
	}
	executor := &mockMigrationExecutor{}

	svc := setupMigrationService(t, executor, reader)

	_, upErr := svc.Up(context.Background())
	require.Error(t, upErr, "Up should propagate directory read errors")

	_, downErr := svc.Down(context.Background(), 1)
	require.Error(t, downErr, "Down should propagate directory read errors")

	_, statusErr := svc.Status(context.Background())
	require.Error(t, statusErr, "Status should propagate directory read errors")

	validateErr := svc.Validate(context.Background())
	require.Error(t, validateErr, "Validate should propagate directory read errors")
}
