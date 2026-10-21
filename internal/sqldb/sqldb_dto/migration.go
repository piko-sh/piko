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

package sqldb_dto

import (
	"context"
	"time"
)

// MigrationDirection says whether a migration applies or reverts a change.
type MigrationDirection uint8

const (
	// DirectionUp is a forward migration that applies schema changes.
	DirectionUp MigrationDirection = iota

	// DirectionDown is a rollback migration that reverts schema changes.
	DirectionDown
)

// AppliedMigration is one row of the migration history table.
type AppliedMigration struct {
	// AppliedAt is when the migration finished.
	AppliedAt time.Time

	// LastStatement is the index of the last statement that completed, set only while a
	// migration is dirty.
	LastStatement *int

	// Name is the migration's name, taken from its filename.
	Name string

	// Checksum is the checksum of the up migration when it was applied.
	Checksum string

	// DownChecksum is the checksum of the down migration when the up migration was applied,
	// or empty when there was none.
	DownChecksum string

	// Version is the migration's version number.
	Version int64

	// DurationMs is how long the migration took, in milliseconds.
	DurationMs int64

	// Dirty reports whether the migration stopped part-way through.
	Dirty bool
}

// MigrationStatus describes one migration file and whether it has been applied.
type MigrationStatus struct {
	// AppliedAt is when the migration was applied, zero when it has not been.
	AppliedAt time.Time

	// LastStatement is the index of the last statement that completed, set only while a
	// migration is dirty.
	LastStatement *int

	// Name is the migration's name.
	Name string

	// Filename is the migration's file name.
	Filename string

	// Version is the migration's version number.
	Version int64

	// Applied reports whether the migration has been applied.
	Applied bool

	// ChecksumMatch reports whether the file still matches the applied checksum.
	ChecksumMatch bool

	// HasDownMigration reports whether a rollback file exists.
	HasDownMigration bool

	// Dirty reports whether the migration stopped part-way through.
	Dirty bool
}

// AppliedSeed is one row of the seed history table.
type AppliedSeed struct {
	// AppliedAt is when the seed finished.
	AppliedAt time.Time

	// Name is the seed's name, taken from its filename.
	Name string

	// Checksum is the checksum of the seed when it was applied.
	Checksum string

	// Version is the seed's version number.
	Version int64

	// DurationMs is how long the seed took, in milliseconds.
	DurationMs int64
}

// SeedStatus describes one seed file and whether it has been applied.
type SeedStatus struct {
	// AppliedAt is when the seed was applied, zero when it has not been.
	AppliedAt time.Time

	// Name is the seed's name.
	Name string

	// Filename is the seed's file name.
	Filename string

	// Version is the seed's version number.
	Version int64

	// Applied reports whether the seed has been applied.
	Applied bool

	// ChecksumMatch reports whether the file still matches the applied checksum.
	ChecksumMatch bool
}

// MigrationHookContext describes the migration a per-migration hook runs around.
type MigrationHookContext struct {
	// Name is the migration's name.
	Name string

	// Version is the migration's version number.
	Version int64

	// Direction says whether the migration is being applied or reverted.
	Direction MigrationDirection
}

// MigrationRunHookContext describes the batch of migrations a run hook runs around.
type MigrationRunHookContext struct {
	// PendingVersions lists the versions the run will apply or revert, in order.
	PendingVersions []int64

	// PendingCount is the number of pending migrations.
	PendingCount int

	// Direction says whether the run applies or reverts migrations.
	Direction MigrationDirection
}

// BeforeMigrationHook runs before each migration. Returning an error stops the run.
type BeforeMigrationHook func(ctx context.Context, hook MigrationHookContext) error

// AfterMigrationHook runs after each successful migration. Returning an error stops the
// run.
type AfterMigrationHook func(ctx context.Context, hook MigrationHookContext) error

// BeforeRunHook runs once before a batch of migrations. Returning an error stops the run.
type BeforeRunHook func(ctx context.Context, hook MigrationRunHookContext) error

// AfterRunHook runs once after a batch of migrations, with the number applied.
type AfterRunHook func(ctx context.Context, hook MigrationRunHookContext, applied int) error

// MigrationServiceConfig collects the options a migration service is built with.
type MigrationServiceConfig struct {
	// BeforeMigration holds the hooks run before each migration.
	BeforeMigration []BeforeMigrationHook

	// AfterMigration holds the hooks run after each migration.
	AfterMigration []AfterMigrationHook

	// BeforeRun holds the hooks run before a batch of migrations.
	BeforeRun []BeforeRunHook

	// AfterRun holds the hooks run after a batch of migrations.
	AfterRun []AfterRunHook

	// NonBlockingLock makes a run fail immediately when another process holds the migration
	// lock, instead of waiting for it.
	NonBlockingLock bool
}
