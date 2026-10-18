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
	"piko.sh/piko/internal/sqldb/sqldb_dto"
)

// MigrationServiceOption configures a migration service.
type MigrationServiceOption func(config *sqldb_dto.MigrationServiceConfig)

// WithNonBlockingLock makes a run fail with ErrLockNotAcquired when another process holds
// the migration lock, instead of waiting for it.
//
// Returns MigrationServiceOption which enables non-blocking locking.
func WithNonBlockingLock() MigrationServiceOption {
	return func(config *sqldb_dto.MigrationServiceConfig) {
		config.NonBlockingLock = true
	}
}

// WithBeforeMigration adds a hook that runs before each migration.
//
// Takes hook (sqldb_dto.BeforeMigrationHook) which may stop the run by returning an
// error.
//
// Returns MigrationServiceOption which registers the hook.
func WithBeforeMigration(hook sqldb_dto.BeforeMigrationHook) MigrationServiceOption {
	return func(config *sqldb_dto.MigrationServiceConfig) {
		config.BeforeMigration = append(config.BeforeMigration, hook)
	}
}

// WithAfterMigration adds a hook that runs after each successful migration.
//
// Takes hook (sqldb_dto.AfterMigrationHook) which may stop the run by returning an error.
//
// Returns MigrationServiceOption which registers the hook.
func WithAfterMigration(hook sqldb_dto.AfterMigrationHook) MigrationServiceOption {
	return func(config *sqldb_dto.MigrationServiceConfig) {
		config.AfterMigration = append(config.AfterMigration, hook)
	}
}

// WithBeforeRun adds a hook that runs once before a batch of migrations.
//
// Takes hook (sqldb_dto.BeforeRunHook) which may stop the run by returning an error.
//
// Returns MigrationServiceOption which registers the hook.
func WithBeforeRun(hook sqldb_dto.BeforeRunHook) MigrationServiceOption {
	return func(config *sqldb_dto.MigrationServiceConfig) {
		config.BeforeRun = append(config.BeforeRun, hook)
	}
}

// WithAfterRun adds a hook that runs once after a batch of migrations.
//
// Takes hook (sqldb_dto.AfterRunHook) which receives the number applied.
//
// Returns MigrationServiceOption which registers the hook.
func WithAfterRun(hook sqldb_dto.AfterRunHook) MigrationServiceOption {
	return func(config *sqldb_dto.MigrationServiceConfig) {
		config.AfterRun = append(config.AfterRun, hook)
	}
}

// BuildMigrationServiceConfig applies options to an empty configuration.
//
// Takes options (...MigrationServiceOption) which are applied in order.
//
// Returns sqldb_dto.MigrationServiceConfig which holds the combined settings.
func BuildMigrationServiceConfig(options ...MigrationServiceOption) sqldb_dto.MigrationServiceConfig {
	config := sqldb_dto.MigrationServiceConfig{
		BeforeMigration: nil,
		AfterMigration:  nil,
		BeforeRun:       nil,
		AfterRun:        nil,
		NonBlockingLock: false,
	}
	for _, option := range options {
		option(&config)
	}
	return config
}
