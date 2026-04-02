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

	"piko.sh/piko/internal/querier/querier_dto"
)

// MigrationHookContext provides information about an individual migration
// being processed.
type MigrationHookContext struct {
	// Name holds the migration file name.
	Name string

	// Version holds the numeric version extracted from the migration filename.
	Version int64

	// Direction holds whether this migration is being applied up or rolled
	// back down.
	Direction querier_dto.MigrationDirection
}

// MigrationRunHookContext provides information about an entire migration run
// before it begins.
type MigrationRunHookContext struct {
	// PendingVersions holds the ordered list of migration versions that will
	// be applied during this run.
	PendingVersions []int64

	// PendingCount holds the number of migrations pending in this run.
	PendingCount int

	// Direction holds whether this run applies migrations up or rolls them
	// back down.
	Direction querier_dto.MigrationDirection
}

// BeforeMigrationHook is called before each individual migration executes.
// Returning an error cancels the migration run.
type BeforeMigrationHook func(ctx context.Context, hook MigrationHookContext) error

// AfterMigrationHook is called after each individual migration executes
// successfully.
type AfterMigrationHook func(ctx context.Context, hook MigrationHookContext) error

// BeforeRunHook is called before the migration run begins, after lock
// acquisition and pending migration computation. Returning an error cancels
// the run.
type BeforeRunHook func(ctx context.Context, hook MigrationRunHookContext) error

// AfterRunHook is called after the migration run completes successfully.
type AfterRunHook func(ctx context.Context, hook MigrationRunHookContext, applied int) error

// WithBeforeMigration registers a hook that runs before each individual
// migration.
//
// Takes hook (BeforeMigrationHook) which is the callback to invoke before
// each migration executes.
//
// Returns MigrationServiceOption which configures the migration service.
func WithBeforeMigration(hook BeforeMigrationHook) MigrationServiceOption {
	return func(service *migrationService) {
		service.beforeMigrationHooks = append(service.beforeMigrationHooks, hook)
	}
}

// WithAfterMigration registers a hook that runs after each individual
// migration succeeds.
//
// Takes hook (AfterMigrationHook) which is the callback to invoke after
// each migration executes successfully.
//
// Returns MigrationServiceOption which configures the migration service.
func WithAfterMigration(hook AfterMigrationHook) MigrationServiceOption {
	return func(service *migrationService) {
		service.afterMigrationHooks = append(service.afterMigrationHooks, hook)
	}
}

// WithBeforeRun registers a hook that runs before the migration run begins.
//
// Takes hook (BeforeRunHook) which is the callback to invoke before the
// migration run starts.
//
// Returns MigrationServiceOption which configures the migration service.
func WithBeforeRun(hook BeforeRunHook) MigrationServiceOption {
	return func(service *migrationService) {
		service.beforeRunHooks = append(service.beforeRunHooks, hook)
	}
}

// WithAfterRun registers a hook that runs after the migration run completes.
//
// Takes hook (AfterRunHook) which is the callback to invoke after the
// migration run completes successfully.
//
// Returns MigrationServiceOption which configures the migration service.
func WithAfterRun(hook AfterRunHook) MigrationServiceOption {
	return func(service *migrationService) {
		service.afterRunHooks = append(service.afterRunHooks, hook)
	}
}
