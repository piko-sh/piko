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
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func threeSeedFileReader() *mockFileReader {
	return &mockFileReader{
		dirs: map[string][]os.DirEntry{
			"seeds": {
				&mockDirEntry{name: "001_authors.sql"},
				&mockDirEntry{name: "002_posts.sql"},
				&mockDirEntry{name: "003_comments.sql"},
			},
		},
		files: map[string][]byte{
			"seeds/001_authors.sql":  []byte("INSERT INTO authors (name) VALUES ('Alice');"),
			"seeds/002_posts.sql":    []byte("INSERT INTO posts (title) VALUES ('Hello');"),
			"seeds/003_comments.sql": []byte("INSERT INTO comments (body) VALUES ('Nice');"),
		},
	}
}

func emptySeedFileReader() *mockFileReader {
	return &mockFileReader{
		dirs: map[string][]os.DirEntry{
			"seeds": {},
		},
		files: map[string][]byte{},
	}
}

func TestSeedService_Apply_NoPendingSeeds(t *testing.T) {
	reader := emptySeedFileReader()
	executor := &mockSeedExecutor{}
	svc := NewSeedService(executor, reader, "seeds")

	applied, err := svc.Apply(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, applied)
}

func TestSeedService_Apply_ThreePendingSeeds(t *testing.T) {
	reader := threeSeedFileReader()
	var executedVersions []int64
	executor := &mockSeedExecutor{
		executeSeedFn: func(_ context.Context, seed querier_dto.SeedRecord) error {
			executedVersions = append(executedVersions, seed.Version)
			return nil
		},
	}
	svc := NewSeedService(executor, reader, "seeds")

	applied, err := svc.Apply(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, applied)
	assert.Equal(t, []int64{1, 2, 3}, executedVersions)
}

func TestSeedService_Apply_SkipsAlreadyApplied(t *testing.T) {
	reader := threeSeedFileReader()
	content1 := reader.files["seeds/001_authors.sql"]
	content2 := reader.files["seeds/002_posts.sql"]

	var executedVersions []int64
	executor := &mockSeedExecutor{
		appliedSeedsFn: func(_ context.Context) ([]querier_dto.AppliedSeed, error) {
			return []querier_dto.AppliedSeed{
				{
					Version:   1,
					Name:      "authors",
					Checksum:  testChecksum(content1),
					AppliedAt: time.Now(),
				},
				{
					Version:   2,
					Name:      "posts",
					Checksum:  testChecksum(content2),
					AppliedAt: time.Now(),
				},
			}, nil
		},
		executeSeedFn: func(_ context.Context, seed querier_dto.SeedRecord) error {
			executedVersions = append(executedVersions, seed.Version)
			return nil
		},
	}
	svc := NewSeedService(executor, reader, "seeds")

	applied, err := svc.Apply(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, applied)
	assert.Equal(t, []int64{3}, executedVersions)
}

func TestSeedService_Apply_WarnsOnChecksumMismatch(t *testing.T) {
	reader := threeSeedFileReader()
	executor := &mockSeedExecutor{
		appliedSeedsFn: func(_ context.Context) ([]querier_dto.AppliedSeed, error) {
			return []querier_dto.AppliedSeed{
				{
					Version:   1,
					Name:      "authors",
					Checksum:  "stale-checksum-that-does-not-match",
					AppliedAt: time.Now(),
				},
			}, nil
		},
	}
	svc := NewSeedService(executor, reader, "seeds")

	applied, err := svc.Apply(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, applied)
}

func TestSeedService_Apply_ExecutionError(t *testing.T) {
	reader := threeSeedFileReader()
	expectedErr := errors.New("database is read-only")
	executor := &mockSeedExecutor{
		executeSeedFn: func(_ context.Context, seed querier_dto.SeedRecord) error {
			if seed.Version == 2 {
				return expectedErr
			}
			return nil
		},
	}
	svc := NewSeedService(executor, reader, "seeds")

	applied, err := svc.Apply(context.Background())
	assert.Equal(t, 1, applied)
	require.Error(t, err)

	var seedErr *SeedExecutionError
	require.ErrorAs(t, err, &seedErr)
	assert.Equal(t, int64(2), seedErr.Version)
	assert.Equal(t, "posts", seedErr.Name)
	assert.ErrorIs(t, seedErr, expectedErr)
}

func TestSeedService_Apply_ContextCancellation(t *testing.T) {
	reader := threeSeedFileReader()
	var callCount atomic.Int32
	executor := &mockSeedExecutor{
		executeSeedFn: func(_ context.Context, _ querier_dto.SeedRecord) error {
			callCount.Add(1)
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	executor.executeSeedFn = func(_ context.Context, seed querier_dto.SeedRecord) error {
		callCount.Add(1)
		if seed.Version == 1 {
			cancel()
		}
		return nil
	}

	svc := NewSeedService(executor, reader, "seeds")
	applied, err := svc.Apply(ctx)

	assert.Equal(t, 1, applied)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSeedService_Reseed(t *testing.T) {
	reader := threeSeedFileReader()
	var cleared bool
	var executedVersions []int64
	executor := &mockSeedExecutor{
		clearSeedHistoryFn: func(_ context.Context) error {
			cleared = true
			return nil
		},
		executeSeedFn: func(_ context.Context, seed querier_dto.SeedRecord) error {
			executedVersions = append(executedVersions, seed.Version)
			return nil
		},
	}
	svc := NewSeedService(executor, reader, "seeds")

	applied, err := svc.Reseed(context.Background())
	require.NoError(t, err)
	assert.True(t, cleared)
	assert.Equal(t, 3, applied)
	assert.Equal(t, []int64{1, 2, 3}, executedVersions)
}

func TestSeedService_Status(t *testing.T) {
	reader := threeSeedFileReader()
	content1 := reader.files["seeds/001_authors.sql"]
	executor := &mockSeedExecutor{
		appliedSeedsFn: func(_ context.Context) ([]querier_dto.AppliedSeed, error) {
			return []querier_dto.AppliedSeed{
				{
					Version:   1,
					Name:      "authors",
					Checksum:  testChecksum(content1),
					AppliedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	}
	svc := NewSeedService(executor, reader, "seeds")

	statuses, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.Len(t, statuses, 3)

	assert.True(t, statuses[0].Applied)
	assert.True(t, statuses[0].ChecksumMatch)
	assert.Equal(t, int64(1), statuses[0].Version)

	assert.False(t, statuses[1].Applied)
	assert.True(t, statuses[1].ChecksumMatch)
	assert.Equal(t, int64(2), statuses[1].Version)

	assert.False(t, statuses[2].Applied)
	assert.Equal(t, int64(3), statuses[2].Version)
}

func TestSeedService_Status_ChecksumMismatch(t *testing.T) {
	reader := threeSeedFileReader()
	executor := &mockSeedExecutor{
		appliedSeedsFn: func(_ context.Context) ([]querier_dto.AppliedSeed, error) {
			return []querier_dto.AppliedSeed{
				{
					Version:  1,
					Name:     "authors",
					Checksum: "old-checksum",
				},
			}, nil
		},
	}
	svc := NewSeedService(executor, reader, "seeds")

	statuses, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.Len(t, statuses, 3)

	assert.True(t, statuses[0].Applied)
	assert.False(t, statuses[0].ChecksumMatch)
}
