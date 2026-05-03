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
	"fmt"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// seedService implements SeedServicePort.
type seedService struct {
	// executor holds the database-specific seed operations adapter.
	executor SeedExecutorPort

	// fileReader holds the filesystem access adapter for reading seed files.
	fileReader FileReaderPort

	// directory holds the path to the directory containing seed files.
	directory string
}

// NewSeedService creates a seed service for applying database seed files.
// The service handles executing SQL seed files in version order with
// idempotency tracking via a history table.
//
// Takes executor (SeedExecutorPort) which provides database-specific seed
// operations.
// Takes fileReader (FileReaderPort) which provides filesystem access for
// reading seed SQL files.
// Takes directory (string) which is the path to the seed files within the
// filesystem.
//
// Returns SeedServicePort which is ready to apply seeds.
func NewSeedService(
	executor SeedExecutorPort,
	fileReader FileReaderPort,
	directory string,
) SeedServicePort {
	return &seedService{
		executor:   executor,
		fileReader: fileReader,
		directory:  directory,
	}
}

// Apply executes all pending seed files in version order, skipping those
// already applied and warning on checksum mismatches.
//
// Apply takes the dialect-specific seed advisory lock for the entire run so
// concurrent replicas cannot both observe a seed as pending and then race to
// insert duplicate history rows. The dialect's idempotent INSERT (e.g.
// "ON CONFLICT (version) DO NOTHING") provides a fallback for dialects
// without true advisory locking (such as SQLite).
//
// Returns int which is the number of seeds applied.
// Returns error when a seed fails to execute.
func (s *seedService) Apply(ctx context.Context) (int, error) {
	_, l := logger_domain.From(ctx, log)

	files, err := readSeedFiles(ctx, s.fileReader, s.directory)
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, nil
	}

	if lockErr := s.executor.AcquireSeedLock(ctx); lockErr != nil {
		return 0, fmt.Errorf("acquiring seed lock: %w", lockErr)
	}
	defer func() {
		if releaseErr := s.executor.ReleaseSeedLock(ctx); releaseErr != nil {
			l.Warn("Releasing seed lock failed", logger_domain.Error(releaseErr))
		}
	}()

	if err := s.executor.EnsureSeedTable(ctx); err != nil {
		return 0, fmt.Errorf("ensuring seed table: %w", err)
	}

	applied, err := s.executor.AppliedSeeds(ctx)
	if err != nil {
		return 0, fmt.Errorf("reading applied seeds: %w", err)
	}

	appliedByVersion := buildAppliedSeedMap(applied)
	pending := filterPendingSeeds(files, appliedByVersion, l)

	count := 0
	for _, seed := range pending {
		if ctx.Err() != nil {
			return count, ctx.Err()
		}

		record := querier_dto.SeedRecord{
			Version:  seed.Version,
			Name:     seed.Name,
			Checksum: seed.Checksum,
			Content:  seed.Content,
		}

		if execErr := s.executor.ExecuteSeed(ctx, record); execErr != nil {
			return count, &SeedExecutionError{
				Version: seed.Version,
				Name:    seed.Name,
				Cause:   execErr,
			}
		}

		l.Internal("Seed applied",
			logger_domain.Int64(logFieldVersion, seed.Version),
			logger_domain.String("name", seed.Name))
		count++
	}

	return count, nil
}

// Status returns the list of all known seeds and their applied state.
//
// Returns []querier_dto.SeedStatus which lists all seeds.
// Returns error when the status cannot be determined.
func (s *seedService) Status(ctx context.Context) ([]querier_dto.SeedStatus, error) {
	files, err := readSeedFiles(ctx, s.fileReader, s.directory)
	if err != nil {
		return nil, err
	}

	if err := s.executor.EnsureSeedTable(ctx); err != nil {
		return nil, fmt.Errorf("ensuring seed table: %w", err)
	}

	applied, err := s.executor.AppliedSeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading applied seeds: %w", err)
	}

	appliedByVersion := buildAppliedSeedMap(applied)
	statuses := make([]querier_dto.SeedStatus, 0, len(files))

	for _, file := range files {
		status := querier_dto.SeedStatus{
			Version:       file.Version,
			Name:          file.Name,
			Filename:      file.Filename,
			Applied:       false,
			ChecksumMatch: true,
		}

		if record, ok := appliedByVersion[file.Version]; ok {
			status.Applied = true
			status.AppliedAt = record.AppliedAt
			status.ChecksumMatch = record.Checksum == file.Checksum
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Reseed clears the seed history table and re-applies all seed files.
//
// Returns int which is the number of seeds applied.
// Returns error when clearing history or applying seeds fails.
func (s *seedService) Reseed(ctx context.Context) (int, error) {
	if err := s.executor.EnsureSeedTable(ctx); err != nil {
		return 0, fmt.Errorf("ensuring seed table: %w", err)
	}

	if err := s.executor.ClearSeedHistory(ctx); err != nil {
		return 0, fmt.Errorf("clearing seed history: %w", err)
	}

	return s.Apply(ctx)
}

// buildAppliedSeedMap creates a lookup map from version to applied seed.
//
// Takes applied ([]querier_dto.AppliedSeed) which holds the
// applied seed records.
//
// Returns map[int64]querier_dto.AppliedSeed which maps
// version numbers to their applied seed records.
func buildAppliedSeedMap(applied []querier_dto.AppliedSeed) map[int64]querier_dto.AppliedSeed {
	m := make(map[int64]querier_dto.AppliedSeed, len(applied))
	for _, a := range applied {
		m[a.Version] = a
	}
	return m
}

// filterPendingSeeds returns seed files that have not yet
// been applied, logging warnings for checksum mismatches.
//
// Takes files ([]querier_dto.SeedFile) which holds all seed
// files to filter.
// Takes appliedByVersion (map[int64]querier_dto.AppliedSeed)
// which holds the already-applied seeds.
// Takes l (logger_domain.Logger) which receives checksum
// mismatch warnings.
//
// Returns []querier_dto.SeedFile which holds only the
// pending seed files.
func filterPendingSeeds(
	files []querier_dto.SeedFile,
	appliedByVersion map[int64]querier_dto.AppliedSeed,
	l logger_domain.Logger,
) []querier_dto.SeedFile {
	pending := make([]querier_dto.SeedFile, 0, len(files))

	for _, file := range files {
		record, ok := appliedByVersion[file.Version]
		if !ok {
			pending = append(pending, file)
			continue
		}

		if record.Checksum != file.Checksum {
			l.Warn("Seed file checksum changed since last application",
				logger_domain.Int64(logFieldVersion, file.Version),
				logger_domain.String("name", file.Name),
				logger_domain.String("applied_checksum", record.Checksum),
				logger_domain.String("file_checksum", file.Checksum))
		}
	}

	return pending
}
