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

package migration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	"piko.sh/piko/internal/querier/querier_dto"
)

type mockExecutor struct {
	applied             []querier_dto.AppliedMigration
	preApplied          []querier_dto.AppliedMigration
	tryLockError        error
	locked              bool
	tableCreated        bool
	lastUsedTransaction bool
}

func (*mockExecutor) EnsureMigrationTable(_ context.Context) error {
	return nil
}

func (executor *mockExecutor) AcquireLock(_ context.Context) error {
	executor.locked = true
	return nil
}

func (executor *mockExecutor) TryAcquireLock(_ context.Context) error {
	if executor.tryLockError != nil {
		return executor.tryLockError
	}
	executor.locked = true
	return nil
}

func (executor *mockExecutor) ReleaseLock(_ context.Context) error {
	executor.locked = false
	return nil
}

func (executor *mockExecutor) AppliedVersions(
	_ context.Context,
) ([]querier_dto.AppliedMigration, error) {
	all := make([]querier_dto.AppliedMigration, 0, len(executor.preApplied)+len(executor.applied))
	all = append(all, executor.preApplied...)
	all = append(all, executor.applied...)

	sort.Slice(all, func(i, j int) bool {
		return all[i].Version < all[j].Version
	})

	return all, nil
}

func (executor *mockExecutor) ExecuteMigration(
	_ context.Context,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	useTransaction bool,
) error {
	executor.lastUsedTransaction = useTransaction

	if direction == querier_dto.MigrationDirectionUp {
		executor.applied = append(executor.applied, querier_dto.AppliedMigration{
			Version:      migration.Version,
			Name:         migration.Name,
			Checksum:     migration.Checksum,
			DownChecksum: migration.DownChecksum,
			AppliedAt:    time.Now(),
			DurationMs:   1,
		})
		return nil
	}

	filtered := make([]querier_dto.AppliedMigration, 0, len(executor.applied))
	for _, applied := range executor.applied {
		if applied.Version != migration.Version {
			filtered = append(filtered, applied)
		}
	}
	executor.applied = filtered

	return nil
}

func computeTestChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
