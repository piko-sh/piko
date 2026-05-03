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
	"bytes"
	"cmp"
	"context"
	"fmt"
	"slices"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// errorFormatEnsuringMigrationTable holds the format string for wrapping
	// errors from EnsureMigrationTable.
	errorFormatEnsuringMigrationTable = "ensuring migration table: %w"

	// errorFormatReadingAppliedVersions holds the format string for wrapping
	// errors from AppliedVersions.
	errorFormatReadingAppliedVersions = "reading applied versions: %w"

	// logFieldVersion holds the structured log field name for migration
	// version numbers.
	logFieldVersion = "version"
)

// noTransactionDirective is the marker that opts a migration out of
// transaction wrapping. If this appears as the first line of a migration
// file, the executor runs the SQL without BEGIN/COMMIT.
var noTransactionDirective = []byte("-- piko:no-transaction")

// MigrationServiceOption configures optional behaviour of the migration
// service.
type MigrationServiceOption func(*migrationService)

// WithNonBlockingLock configures the migration service to use a non-blocking
// lock acquisition. If the lock is already held, operations return
// ErrLockNotAcquired immediately instead of waiting.
//
// Returns MigrationServiceOption which sets the nonBlockingLock flag on the
// service.
func WithNonBlockingLock() MigrationServiceOption {
	return func(service *migrationService) {
		service.nonBlockingLock = true
	}
}

// migrationService implements MigrationServicePort.
type migrationService struct {
	// executor holds the database-specific migration operations adapter.
	executor MigrationExecutorPort

	// fileReader holds the filesystem access adapter for reading migration files.
	fileReader FileReaderPort

	// directory holds the path to the directory containing migration files.
	directory string

	// beforeMigrationHooks holds hooks invoked before each individual migration.
	beforeMigrationHooks []BeforeMigrationHook

	// afterMigrationHooks holds hooks invoked after each individual migration.
	afterMigrationHooks []AfterMigrationHook

	// beforeRunHooks holds hooks invoked before a batch of migrations begins.
	beforeRunHooks []BeforeRunHook

	// afterRunHooks holds hooks invoked after a batch of migrations completes.
	afterRunHooks []AfterRunHook

	// nonBlockingLock indicates whether lock acquisition should be non-blocking.
	nonBlockingLock bool
}

// NewMigrationService creates a new migration service.
//
// Takes executor (MigrationExecutorPort) which provides database-specific
// migration operations.
// Takes fileReader (FileReaderPort) which provides filesystem access.
// Takes directory (string) which is the path to the migration files.
// Takes options (...MigrationServiceOption) which configure optional
// behaviour such as non-blocking lock acquisition.
//
// Returns MigrationServicePort which is ready to apply or roll back
// migrations.
func NewMigrationService(
	executor MigrationExecutorPort,
	fileReader FileReaderPort,
	directory string,
	options ...MigrationServiceOption,
) MigrationServicePort {
	service := &migrationService{
		executor:   executor,
		fileReader: fileReader,
		directory:  directory,
	}
	for _, option := range options {
		option(service)
	}
	return service
}

// Up applies all pending up migrations in version order.
//
// Returns int which is the number of migrations applied.
// Returns error when migration reading, checksum
// validation, lock acquisition, or execution fails.
func (service *migrationService) Up(ctx context.Context) (int, error) {
	return service.applyUpMigrations(ctx, nil)
}

// UpTo applies pending up migrations up to and including the target version.
//
// Takes targetVersion (int64) which specifies the maximum version to apply.
//
// Returns int which is the number of migrations applied.
// Returns error when migration reading, checksum
// validation, lock acquisition, or execution fails.
func (service *migrationService) UpTo(ctx context.Context, targetVersion int64) (int, error) {
	return service.applyUpMigrations(ctx, &targetVersion)
}

// Down rolls back the last n applied migrations in reverse version order.
//
// Takes steps (int) which specifies how many migrations to roll back.
//
// Returns int which is the number of migrations rolled
// back.
// Returns error when migration reading, lock acquisition,
// or rollback execution fails.
func (service *migrationService) Down(ctx context.Context, steps int) (int, error) {
	return service.rollbackMigrations(ctx, &steps, nil)
}

// DownTo rolls back applied migrations down to (but not including) the target
// version.
//
// Takes targetVersion (int64) which specifies the version to roll back to.
//
// Returns int which is the number of migrations rolled
// back.
// Returns error when migration reading, lock acquisition,
// or rollback execution fails.
func (service *migrationService) DownTo(ctx context.Context, targetVersion int64) (int, error) {
	return service.rollbackMigrations(ctx, nil, &targetVersion)
}

// Status returns the list of all known migrations and their applied state.
//
// Returns []querier_dto.MigrationStatus which holds the
// status of each known migration.
// Returns error when reading files or querying applied
// versions fails.
func (service *migrationService) Status(ctx context.Context) ([]querier_dto.MigrationStatus, error) {
	ctx, _ = logger_domain.From(ctx, log)
	ctx, span, _ := log.Span(ctx, "MigrationService.Status")
	defer span.End()

	allFiles, readError := readMigrationFilesVersioned(ctx, service.fileReader, service.directory)
	if readError != nil {
		return nil, readError
	}

	if ensureError := service.executor.EnsureMigrationTable(ctx); ensureError != nil {
		return nil, fmt.Errorf(errorFormatEnsuringMigrationTable, ensureError)
	}

	applied, appliedError := service.executor.AppliedVersions(ctx)
	if appliedError != nil {
		return nil, fmt.Errorf(errorFormatReadingAppliedVersions, appliedError)
	}

	appliedByVersion := make(map[int64]querier_dto.AppliedMigration, len(applied))
	for _, migration := range applied {
		appliedByVersion[migration.Version] = migration
	}

	upFiles := filterByDirection(allFiles, querier_dto.MigrationDirectionUp)
	downVersions := buildDownVersionSet(allFiles)

	statuses := make([]querier_dto.MigrationStatus, 0, len(upFiles))
	for _, file := range upFiles {
		status := querier_dto.MigrationStatus{
			Version:          file.Version,
			Name:             file.Name,
			Filename:         file.Filename,
			HasDownMigration: downVersions[file.Version],
		}

		if appliedMigration, found := appliedByVersion[file.Version]; found {
			status.Applied = true
			status.AppliedAt = appliedMigration.AppliedAt
			status.ChecksumMatch = appliedMigration.Checksum == file.Checksum
			status.Dirty = appliedMigration.Dirty
			status.LastStatement = appliedMigration.LastStatement
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Validate checks that all applied migration checksums match their on-disk
// files without executing anything.
//
// Returns error when file reading, table initialisation, version querying,
// or checksum validation fails.
func (service *migrationService) Validate(ctx context.Context) error {
	ctx, _ = logger_domain.From(ctx, log)
	ctx, span, _ := log.Span(ctx, "MigrationService.Validate")
	defer span.End()

	allFiles, readError := readMigrationFilesVersioned(ctx, service.fileReader, service.directory)
	if readError != nil {
		return readError
	}

	upFiles := filterByDirection(allFiles, querier_dto.MigrationDirectionUp)

	if ensureError := service.executor.EnsureMigrationTable(ctx); ensureError != nil {
		return fmt.Errorf(errorFormatEnsuringMigrationTable, ensureError)
	}

	applied, appliedError := service.executor.AppliedVersions(ctx)
	if appliedError != nil {
		return fmt.Errorf(errorFormatReadingAppliedVersions, appliedError)
	}

	return validateChecksums(upFiles, applied)
}

// applyUpMigrations reads migration files, validates checksums, acquires the
// advisory lock, and executes all pending up migrations up to the optional
// target version.
//
// Takes targetVersion (*int64) which specifies the maximum version to apply,
// or nil to apply all pending migrations.
//
// Returns int which is the number of migrations applied.
// Returns error when any step fails.
func (service *migrationService) applyUpMigrations(
	ctx context.Context,
	targetVersion *int64,
) (int, error) {
	ctx, _ = logger_domain.From(ctx, log)
	ctx, span, _ := log.Span(ctx, "MigrationService.Up")
	defer span.End()

	allFiles, readError := readMigrationFilesVersioned(ctx, service.fileReader, service.directory)
	if readError != nil {
		return 0, readError
	}

	upFiles := filterByDirection(allFiles, querier_dto.MigrationDirectionUp)

	if ensureError := service.executor.EnsureMigrationTable(ctx); ensureError != nil {
		return 0, fmt.Errorf(errorFormatEnsuringMigrationTable, ensureError)
	}

	applied, appliedError := service.executor.AppliedVersions(ctx)
	if appliedError != nil {
		return 0, fmt.Errorf(errorFormatReadingAppliedVersions, appliedError)
	}

	pending := computePending(upFiles, applied)
	pending = filterByTargetVersion(pending, targetVersion)
	if len(pending) == 0 {
		return 0, nil
	}

	if checksumError := validateChecksums(upFiles, applied); checksumError != nil {
		return 0, checksumError
	}

	service.warnSkippedMigrations(ctx, pending, applied)

	if lockError := service.acquireLock(ctx); lockError != nil {
		return 0, &LockAcquisitionError{Cause: lockError}
	}
	defer service.releaseLock(context.WithoutCancel(ctx))

	applied, appliedError = service.executor.AppliedVersions(ctx)
	if appliedError != nil {
		return 0, fmt.Errorf("reading applied versions under lock: %w", appliedError)
	}

	pending = computePending(upFiles, applied)
	pending = filterByTargetVersion(pending, targetVersion)
	if len(pending) == 0 {
		return 0, nil
	}

	downChecksumsByVersion := buildDownChecksumMap(allFiles)

	return service.executePendingUp(ctx, pending, downChecksumsByVersion)
}

// rollbackMigrations reads migration files, acquires the advisory lock, and
// rolls back applied migrations by step count or down to a target version.
//
// Takes steps (*int) which specifies how many migrations to roll back, or nil
// to use targetVersion instead.
// Takes targetVersion (*int64) which specifies the version to roll back to, or
// nil to use steps instead.
//
// Returns int which is the number of migrations rolled
// back.
// Returns error when any step fails.
func (service *migrationService) rollbackMigrations(
	ctx context.Context,
	steps *int,
	targetVersion *int64,
) (int, error) {
	ctx, _ = logger_domain.From(ctx, log)
	ctx, span, _ := log.Span(ctx, "MigrationService.Down")
	defer span.End()

	allFiles, readError := readMigrationFilesVersioned(ctx, service.fileReader, service.directory)
	if readError != nil {
		return 0, readError
	}

	downFilesByVersion := buildDownFileMap(allFiles)

	if ensureError := service.executor.EnsureMigrationTable(ctx); ensureError != nil {
		return 0, fmt.Errorf(errorFormatEnsuringMigrationTable, ensureError)
	}

	applied, appliedError := service.executor.AppliedVersions(ctx)
	if appliedError != nil {
		return 0, fmt.Errorf(errorFormatReadingAppliedVersions, appliedError)
	}

	if len(applied) == 0 {
		return 0, nil
	}

	rollbackCount := computeRollbackSteps(applied, steps, targetVersion)
	if rollbackCount == 0 {
		return 0, nil
	}

	if lockError := service.acquireLock(ctx); lockError != nil {
		return 0, &LockAcquisitionError{Cause: lockError}
	}
	defer service.releaseLock(context.WithoutCancel(ctx))

	return service.executeRollbacks(ctx, applied, rollbackCount, downFilesByVersion)
}

// warnSkippedMigrations logs a warning for each pending migration whose
// version is earlier than the maximum applied version.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations to
// check.
// Takes applied ([]querier_dto.AppliedMigration) which holds the already
// applied migrations.
func (*migrationService) warnSkippedMigrations(
	ctx context.Context,
	pending []querier_dto.MigrationFile,
	applied []querier_dto.AppliedMigration,
) {
	skippedVersions := detectSkippedMigrations(pending, applied)
	if len(skippedVersions) == 0 {
		return
	}
	_, l := logger_domain.From(ctx, log)
	for _, version := range skippedVersions {
		l.Warn("applying skipped migration",
			logger_domain.Int64(logFieldVersion, version),
		)
	}
}

// executePendingUp runs before-run hooks, applies each pending migration with
// its before/after hooks, and runs after-run hooks. Before processing pending
// migrations, any dirty migration is detected and either retried (if it
// matches the next pending version) or reported as a blocking error.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations to
// apply.
// Takes downChecksumsByVersion (map[int64]string) which maps versions to their
// down-migration checksums.
//
// Returns int which is the number of migrations applied.
// Returns error when any hook or migration execution fails, or a dirty
// migration from a different version blocks progress.
func (service *migrationService) executePendingUp(
	ctx context.Context,
	pending []querier_dto.MigrationFile,
	downChecksumsByVersion map[int64]string,
) (int, error) {
	applied, appliedError := service.executor.AppliedVersions(ctx)
	if appliedError != nil {
		return 0, fmt.Errorf(errorFormatReadingAppliedVersions, appliedError)
	}

	dirtyMigration, skipUpTo := findDirtyMigration(applied)

	if dirtyMigration != nil {
		retryResult, retryError := service.handleDirtyMigration(
			ctx, pending, downChecksumsByVersion, *dirtyMigration, skipUpTo,
		)
		if retryError != nil {
			return retryResult, retryError
		}

		pending = removePendingVersion(pending, dirtyMigration.Version)
		if retryResult > 0 && len(pending) == 0 {
			return retryResult, nil
		}
	}

	if hookError := service.runBeforeRunHooks(ctx, pending, querier_dto.MigrationDirectionUp); hookError != nil {
		return 0, hookError
	}

	count := 0
	if dirtyMigration != nil {
		count = 1
	}

	migrated, applyError := service.applyPendingUpMigrations(ctx, pending, downChecksumsByVersion)
	return count + migrated, applyError
}

// applyPendingUpMigrations iterates through pending migrations, running hooks
// and executing each in sequence.
//
// Takes pending ([]querier_dto.MigrationFile) which holds
// the migrations to apply.
// Takes downChecksumsByVersion (map[int64]string) which maps
// versions to their down-migration checksums.
//
// Returns int which is the number of migrations successfully applied.
// Returns error when a hook or migration execution fails.
func (service *migrationService) applyPendingUpMigrations(
	ctx context.Context,
	pending []querier_dto.MigrationFile,
	downChecksumsByVersion map[int64]string,
) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	count := 0

	for _, migration := range pending {
		if ctx.Err() != nil {
			return count, ctx.Err()
		}

		hookContext := MigrationHookContext{
			Version:   migration.Version,
			Name:      migration.Name,
			Direction: querier_dto.MigrationDirectionUp,
		}

		if hookError := service.runBeforeMigrationHooks(ctx, hookContext); hookError != nil {
			return count, hookError
		}

		useTransaction := !hasNoTransactionDirective(migration.Content)
		record := querier_dto.MigrationRecord{
			Version:      migration.Version,
			Name:         migration.Name,
			Content:      migration.Content,
			Checksum:     migration.Checksum,
			DownChecksum: downChecksumsByVersion[migration.Version],
			SkipUpTo:     -1,
		}

		executeError := service.executor.ExecuteMigration(
			ctx, record, querier_dto.MigrationDirectionUp, useTransaction,
		)
		if executeError != nil {
			return count, &MigrationExecutionError{
				Cause:     executeError,
				Name:      migration.Name,
				Version:   migration.Version,
				Direction: querier_dto.MigrationDirectionUp,
			}
		}
		count++

		if hookError := service.runAfterMigrationHooks(ctx, hookContext); hookError != nil {
			return count, hookError
		}

		l.Trace("applied migration",
			logger_domain.Int64(logFieldVersion, migration.Version),
			logger_domain.String("name", migration.Name),
		)
	}

	if hookError := service.runAfterRunHooks(ctx, pending, querier_dto.MigrationDirectionUp, count); hookError != nil {
		return count, hookError
	}

	return count, nil
}

// findDirtyMigration scans applied migrations for one marked as dirty.
//
// Takes applied ([]querier_dto.AppliedMigration) which holds the applied
// migration records.
//
// Returns *querier_dto.AppliedMigration which is the dirty
// migration, or nil if none is dirty.
// Returns int which is the last completed statement index to
// skip on retry (-1 if no statements completed).
func findDirtyMigration(
	applied []querier_dto.AppliedMigration,
) (*querier_dto.AppliedMigration, int) {
	for i := range applied {
		if applied[i].Dirty {
			skipUpTo := -1
			if applied[i].LastStatement != nil {
				skipUpTo = *applied[i].LastStatement
			}
			return &applied[i], skipUpTo
		}
	}
	return nil, -1
}

// handleDirtyMigration checks whether a dirty migration can be retried
// (because it matches the next pending version) and retries it if so.
// If the dirty migration does not match the next pending version, a
// DirtyMigrationError is returned.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the pending
// migrations.
// Takes downChecksumsByVersion (map[int64]string) which maps versions to
// their down-migration checksums.
// Takes dirtyMigration (querier_dto.AppliedMigration) which is the dirty
// migration record.
// Takes skipUpTo (int) which is the last completed statement index.
//
// Returns int which is 1 if the retry succeeded, 0 otherwise.
// Returns error when the dirty migration cannot be retried or execution fails.
func (service *migrationService) handleDirtyMigration(
	ctx context.Context,
	pending []querier_dto.MigrationFile,
	downChecksumsByVersion map[int64]string,
	dirtyMigration querier_dto.AppliedMigration,
	skipUpTo int,
) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	if len(pending) == 0 {
		lastStatement := -1
		if dirtyMigration.LastStatement != nil {
			lastStatement = *dirtyMigration.LastStatement
		}
		return 0, &DirtyMigrationError{
			Version:       dirtyMigration.Version,
			LastStatement: lastStatement,
		}
	}

	nextPending := pending[0]
	if dirtyMigration.Version != nextPending.Version {
		lastStatement := -1
		if dirtyMigration.LastStatement != nil {
			lastStatement = *dirtyMigration.LastStatement
		}
		return 0, &DirtyMigrationError{
			Version:       dirtyMigration.Version,
			LastStatement: lastStatement,
		}
	}

	l.Trace("retrying dirty migration",
		logger_domain.Int64(logFieldVersion, dirtyMigration.Version),
		logger_domain.Int("skip_up_to", skipUpTo),
	)

	useTransaction := !hasNoTransactionDirective(nextPending.Content)
	record := querier_dto.MigrationRecord{
		Version:      nextPending.Version,
		Name:         nextPending.Name,
		Content:      nextPending.Content,
		Checksum:     nextPending.Checksum,
		DownChecksum: downChecksumsByVersion[nextPending.Version],
		SkipUpTo:     skipUpTo,
	}

	executeError := service.executor.ExecuteMigration(
		ctx, record, querier_dto.MigrationDirectionUp, useTransaction,
	)
	if executeError != nil {
		return 0, &MigrationExecutionError{
			Cause:     executeError,
			Name:      nextPending.Name,
			Version:   nextPending.Version,
			Direction: querier_dto.MigrationDirectionUp,
		}
	}

	l.Trace("retried dirty migration successfully",
		logger_domain.Int64(logFieldVersion, dirtyMigration.Version),
	)

	return 1, nil
}

// removePendingVersion removes a specific version from the pending list.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the pending
// migrations.
// Takes version (int64) which is the version to remove.
//
// Returns []querier_dto.MigrationFile without the specified version.
func removePendingVersion(
	pending []querier_dto.MigrationFile,
	version int64,
) []querier_dto.MigrationFile {
	filtered := make([]querier_dto.MigrationFile, 0, len(pending))
	for _, file := range pending {
		if file.Version != version {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// executeRollbacks runs before-run hooks, rolls back the specified number of
// migrations in reverse order, and runs after-run hooks.
//
// Takes applied ([]querier_dto.AppliedMigration) which holds all applied
// migrations in version order.
// Takes steps (int) which specifies how many migrations to roll back from the
// end.
// Takes downFilesByVersion (map[int64]querier_dto.MigrationFile) which maps
// versions to their down-migration files.
//
// Returns int which is the number of migrations rolled
// back.
// Returns error when any hook or rollback execution fails.
func (service *migrationService) executeRollbacks(
	ctx context.Context,
	applied []querier_dto.AppliedMigration,
	steps int,
	downFilesByVersion map[int64]querier_dto.MigrationFile,
) (int, error) {
	rollbackVersions := make([]int64, 0, steps)
	for i := len(applied) - 1; i >= len(applied)-steps; i-- {
		rollbackVersions = append(rollbackVersions, applied[i].Version)
	}

	if hookError := service.runBeforeRunHooksFromVersions(ctx, rollbackVersions, querier_dto.MigrationDirectionDown); hookError != nil {
		return 0, hookError
	}

	count := 0
	for i := len(applied) - 1; i >= len(applied)-steps; i-- {
		if ctx.Err() != nil {
			return count, ctx.Err()
		}

		downFile, exists := downFilesByVersion[applied[i].Version]
		if !exists {
			return count, &NoDownMigrationError{Version: applied[i].Version}
		}

		if rollbackError := service.executeSingleRollback(ctx, applied[i], downFile); rollbackError != nil {
			return count, rollbackError
		}
		count++
	}

	if hookError := service.runAfterRunHooksFromVersions(ctx, rollbackVersions, querier_dto.MigrationDirectionDown, count); hookError != nil {
		return count, hookError
	}

	return count, nil
}

// executeSingleRollback validates the down checksum, runs before/after
// migration hooks, and executes a single rollback migration.
//
// Takes appliedMigration (querier_dto.AppliedMigration) which holds the
// applied migration record to roll back.
// Takes downFile (querier_dto.MigrationFile) which holds the down migration
// file content.
//
// Returns error when checksum validation, hook execution, or migration
// execution fails.
func (service *migrationService) executeSingleRollback(
	ctx context.Context,
	appliedMigration querier_dto.AppliedMigration,
	downFile querier_dto.MigrationFile,
) error {
	ctx, l := logger_domain.From(ctx, log)
	hookContext := MigrationHookContext{
		Version:   downFile.Version,
		Name:      downFile.Name,
		Direction: querier_dto.MigrationDirectionDown,
	}

	if hookError := service.runBeforeMigrationHooks(ctx, hookContext); hookError != nil {
		return hookError
	}

	if checksumError := validateDownChecksum(ctx, appliedMigration, downFile); checksumError != nil {
		return checksumError
	}

	useTransaction := !hasNoTransactionDirective(downFile.Content)
	record := querier_dto.MigrationRecord{
		Version:  downFile.Version,
		Name:     downFile.Name,
		Content:  downFile.Content,
		Checksum: downFile.Checksum,
		SkipUpTo: -1,
	}

	executeError := service.executor.ExecuteMigration(
		ctx, record, querier_dto.MigrationDirectionDown, useTransaction,
	)
	if executeError != nil {
		return &MigrationExecutionError{
			Cause:     executeError,
			Name:      downFile.Name,
			Version:   downFile.Version,
			Direction: querier_dto.MigrationDirectionDown,
		}
	}

	if hookError := service.runAfterMigrationHooks(ctx, hookContext); hookError != nil {
		return hookError
	}

	l.Trace("rolled back migration",
		logger_domain.Int64(logFieldVersion, downFile.Version),
		logger_domain.String("name", downFile.Name),
	)
	return nil
}

// validateDownChecksum checks that the down migration file checksum matches
// the checksum recorded when the up migration was applied.
//
// Takes appliedMigration (querier_dto.AppliedMigration) which holds the
// recorded down checksum.
// Takes downFile (querier_dto.MigrationFile) which holds the current file
// checksum.
//
// Returns error when the recorded checksum does not match the file checksum.
func validateDownChecksum(
	ctx context.Context,
	appliedMigration querier_dto.AppliedMigration,
	downFile querier_dto.MigrationFile,
) error {
	recordedDownChecksum := appliedMigration.DownChecksum
	if recordedDownChecksum != "" && recordedDownChecksum != downFile.Checksum {
		return &DownChecksumMismatchError{
			Version:          downFile.Version,
			Name:             downFile.Name,
			RecordedChecksum: recordedDownChecksum,
			FileChecksum:     downFile.Checksum,
		}
	}
	if recordedDownChecksum == "" {
		_, l := logger_domain.From(ctx, log)
		l.Warn("no recorded down checksum for migration, skipping validation",
			logger_domain.Int64(logFieldVersion, downFile.Version),
		)
	}
	return nil
}

// runBeforeRunHooks invokes all registered before-run hooks with a context
// built from the pending migration files.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations about
// to be applied.
// Takes direction (querier_dto.MigrationDirection) which indicates whether the
// run is up or down.
//
// Returns error when any hook returns an error.
func (service *migrationService) runBeforeRunHooks(
	ctx context.Context,
	pending []querier_dto.MigrationFile,
	direction querier_dto.MigrationDirection,
) error {
	if len(service.beforeRunHooks) == 0 {
		return nil
	}
	versions := make([]int64, len(pending))
	for i, file := range pending {
		versions[i] = file.Version
	}
	hookContext := MigrationRunHookContext{
		Direction:       direction,
		PendingCount:    len(pending),
		PendingVersions: versions,
	}
	for _, hook := range service.beforeRunHooks {
		if hookError := hook(ctx, hookContext); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runBeforeRunHooksFromVersions invokes all registered before-run hooks with
// a context built from explicit version numbers.
//
// Takes versions ([]int64) which holds the migration versions about to be
// processed.
// Takes direction (querier_dto.MigrationDirection) which indicates whether the
// run is up or down.
//
// Returns error when any hook returns an error.
func (service *migrationService) runBeforeRunHooksFromVersions(
	ctx context.Context,
	versions []int64,
	direction querier_dto.MigrationDirection,
) error {
	if len(service.beforeRunHooks) == 0 {
		return nil
	}
	hookContext := MigrationRunHookContext{
		Direction:       direction,
		PendingCount:    len(versions),
		PendingVersions: versions,
	}
	for _, hook := range service.beforeRunHooks {
		if hookError := hook(ctx, hookContext); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runAfterRunHooks invokes all registered after-run hooks with a context
// built from the pending migration files and the count of applied migrations.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations that
// were processed.
// Takes direction (querier_dto.MigrationDirection) which indicates whether the
// run was up or down.
// Takes applied (int) which is the number of migrations that were applied.
//
// Returns error when any hook returns an error.
func (service *migrationService) runAfterRunHooks(
	ctx context.Context,
	pending []querier_dto.MigrationFile,
	direction querier_dto.MigrationDirection,
	applied int,
) error {
	if len(service.afterRunHooks) == 0 {
		return nil
	}
	versions := make([]int64, len(pending))
	for i, file := range pending {
		versions[i] = file.Version
	}
	hookContext := MigrationRunHookContext{
		Direction:       direction,
		PendingCount:    len(pending),
		PendingVersions: versions,
	}
	for _, hook := range service.afterRunHooks {
		if hookError := hook(ctx, hookContext, applied); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runAfterRunHooksFromVersions invokes all registered after-run hooks with
// a context built from explicit version numbers and the count of applied
// migrations.
//
// Takes versions ([]int64) which holds the migration versions that were
// processed.
// Takes direction (querier_dto.MigrationDirection) which indicates whether the
// run was up or down.
// Takes applied (int) which is the number of migrations that were applied.
//
// Returns error when any hook returns an error.
func (service *migrationService) runAfterRunHooksFromVersions(
	ctx context.Context,
	versions []int64,
	direction querier_dto.MigrationDirection,
	applied int,
) error {
	if len(service.afterRunHooks) == 0 {
		return nil
	}
	hookContext := MigrationRunHookContext{
		Direction:       direction,
		PendingCount:    len(versions),
		PendingVersions: versions,
	}
	for _, hook := range service.afterRunHooks {
		if hookError := hook(ctx, hookContext, applied); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runBeforeMigrationHooks invokes all registered before-migration hooks for a
// single migration.
//
// Takes hookContext (MigrationHookContext) which holds the version, name, and
// direction of the migration.
//
// Returns error when any hook returns an error.
func (service *migrationService) runBeforeMigrationHooks(
	ctx context.Context,
	hookContext MigrationHookContext,
) error {
	for _, hook := range service.beforeMigrationHooks {
		if hookError := hook(ctx, hookContext); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runAfterMigrationHooks invokes all registered after-migration hooks for a
// single migration.
//
// Takes hookContext (MigrationHookContext) which holds the version, name, and
// direction of the migration.
//
// Returns error when any hook returns an error.
func (service *migrationService) runAfterMigrationHooks(
	ctx context.Context,
	hookContext MigrationHookContext,
) error {
	for _, hook := range service.afterMigrationHooks {
		if hookError := hook(ctx, hookContext); hookError != nil {
			return hookError
		}
	}
	return nil
}

// acquireLock acquires the migration advisory lock, using either blocking or
// non-blocking mode depending on the service configuration.
//
// Returns error when the lock cannot be acquired.
func (service *migrationService) acquireLock(ctx context.Context) error {
	if service.nonBlockingLock {
		return service.executor.TryAcquireLock(ctx)
	}
	return service.executor.AcquireLock(ctx)
}

// releaseLock releases the migration advisory lock, logging an error if
// release fails.
func (service *migrationService) releaseLock(ctx context.Context) {
	if releaseError := service.executor.ReleaseLock(ctx); releaseError != nil {
		_, l := logger_domain.From(ctx, log)
		l.Error("failed to release migration lock",
			logger_domain.Error(releaseError),
		)
	}
}

// filterByDirection returns only files matching the given direction.
//
// Takes files ([]querier_dto.MigrationFile) which holds all migration files.
// Takes direction (querier_dto.MigrationDirection) which specifies the
// direction to filter by.
//
// Returns []querier_dto.MigrationFile which holds only files matching the
// specified direction.
func filterByDirection(
	files []querier_dto.MigrationFile,
	direction querier_dto.MigrationDirection,
) []querier_dto.MigrationFile {
	var filtered []querier_dto.MigrationFile
	for _, file := range files {
		if file.Direction == direction {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// buildDownFileMap indexes down migration files by version.
//
// Takes files ([]querier_dto.MigrationFile) which holds all migration files.
//
// Returns map[int64]querier_dto.MigrationFile which maps version numbers to
// their corresponding down migration files.
func buildDownFileMap(
	files []querier_dto.MigrationFile,
) map[int64]querier_dto.MigrationFile {
	result := make(map[int64]querier_dto.MigrationFile)
	for _, file := range files {
		if file.Direction == querier_dto.MigrationDirectionDown {
			result[file.Version] = file
		}
	}
	return result
}

// buildDownChecksumMap returns a map of version to down-migration checksum for
// all versions that have a .down.sql file.
//
// Takes files ([]querier_dto.MigrationFile) which holds all migration files.
//
// Returns map[int64]string which maps version numbers to their down-migration
// checksums.
func buildDownChecksumMap(files []querier_dto.MigrationFile) map[int64]string {
	result := make(map[int64]string)
	for _, file := range files {
		if file.Direction == querier_dto.MigrationDirectionDown {
			result[file.Version] = file.Checksum
		}
	}
	return result
}

// buildDownVersionSet returns a set of versions that have down migration files.
//
// Takes files ([]querier_dto.MigrationFile) which holds all migration files.
//
// Returns map[int64]bool which contains true for each version that has a down
// migration file.
func buildDownVersionSet(files []querier_dto.MigrationFile) map[int64]bool {
	result := make(map[int64]bool)
	for _, file := range files {
		if file.Direction == querier_dto.MigrationDirectionDown {
			result[file.Version] = true
		}
	}
	return result
}

// filterByTargetVersion filters pending migrations to only include those up
// to and including the target version.
//
// If targetVersion is nil, all pending migrations are returned.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations to
// filter.
// Takes targetVersion (*int64) which specifies the maximum version to include,
// or nil to include all.
//
// Returns []querier_dto.MigrationFile which holds the filtered migrations.
func filterByTargetVersion(
	pending []querier_dto.MigrationFile,
	targetVersion *int64,
) []querier_dto.MigrationFile {
	if targetVersion == nil {
		return pending
	}
	var filtered []querier_dto.MigrationFile
	for _, file := range pending {
		if file.Version <= *targetVersion {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// computeRollbackSteps determines how many migrations to roll back.
//
// If steps is provided, that value is used (clamped to applied length). If
// targetVersion is provided, it counts applied migrations with version greater
// than targetVersion.
//
// Takes applied ([]querier_dto.AppliedMigration) which holds all applied
// migrations.
// Takes steps (*int) which specifies the number of migrations to roll back, or
// nil to use targetVersion.
// Takes targetVersion (*int64) which specifies the version to roll back to, or
// nil to use steps.
//
// Returns int which is the number of migrations to roll back.
func computeRollbackSteps(
	applied []querier_dto.AppliedMigration,
	steps *int,
	targetVersion *int64,
) int {
	if steps != nil {
		if *steps > len(applied) {
			return len(applied)
		}
		return *steps
	}
	if targetVersion != nil {
		count := 0
		for _, migration := range applied {
			if migration.Version > *targetVersion {
				count++
			}
		}
		return count
	}
	return 0
}

// computePending returns up migration files that have not been successfully
// applied yet, sorted by version ascending. Dirty (partially-applied)
// migrations are treated as pending since they need to be retried.
//
// Takes upFiles ([]querier_dto.MigrationFile) which holds all up migration
// files.
// Takes applied ([]querier_dto.AppliedMigration) which holds already applied
// migrations.
//
// Returns []querier_dto.MigrationFile which holds the unapplied or dirty
// migrations sorted by version.
func computePending(
	upFiles []querier_dto.MigrationFile,
	applied []querier_dto.AppliedMigration,
) []querier_dto.MigrationFile {
	appliedSet := make(map[int64]struct{}, len(applied))
	for _, migration := range applied {
		if !migration.Dirty {
			appliedSet[migration.Version] = struct{}{}
		}
	}

	var pending []querier_dto.MigrationFile
	for _, file := range upFiles {
		if _, alreadyApplied := appliedSet[file.Version]; !alreadyApplied {
			pending = append(pending, file)
		}
	}

	slices.SortFunc(pending, func(a, b querier_dto.MigrationFile) int {
		return cmp.Compare(a.Version, b.Version)
	})

	return pending
}

// validateChecksums verifies that all applied migrations have matching
// checksums with on-disk files.
//
// Takes upFiles ([]querier_dto.MigrationFile) which holds the on-disk
// migration files.
// Takes applied ([]querier_dto.AppliedMigration) which holds the applied
// migration records.
//
// Returns error when a file is missing or its checksum does not match the
// applied record.
func validateChecksums(
	upFiles []querier_dto.MigrationFile,
	applied []querier_dto.AppliedMigration,
) error {
	filesByVersion := make(map[int64]querier_dto.MigrationFile, len(upFiles))
	for _, file := range upFiles {
		filesByVersion[file.Version] = file
	}

	for _, migration := range applied {
		file, exists := filesByVersion[migration.Version]
		if !exists {
			return &MissingMigrationFileError{
				Version: migration.Version,
				Name:    migration.Name,
			}
		}
		if file.Checksum != migration.Checksum {
			return &ChecksumMismatchError{
				Version:         migration.Version,
				Name:            migration.Name,
				AppliedChecksum: migration.Checksum,
				FileChecksum:    file.Checksum,
			}
		}
	}

	return nil
}

// detectSkippedMigrations finds pending migrations whose version is earlier
// than the maximum applied version.
//
// These are migrations that were added after later migrations were already
// applied, for example from branch merges.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the pending
// migrations to check.
// Takes applied ([]querier_dto.AppliedMigration) which holds the already
// applied migrations.
//
// Returns []int64 which holds the version numbers of skipped migrations.
func detectSkippedMigrations(
	pending []querier_dto.MigrationFile,
	applied []querier_dto.AppliedMigration,
) []int64 {
	if len(applied) == 0 {
		return nil
	}

	maxApplied := applied[len(applied)-1].Version

	var skipped []int64
	for _, migration := range pending {
		if migration.Version < maxApplied {
			skipped = append(skipped, migration.Version)
		}
	}

	return skipped
}

// hasNoTransactionDirective checks whether the migration content starts with
// the -- piko:no-transaction directive.
//
// Takes content ([]byte) which holds the migration file content.
//
// Returns bool which is true if the first line matches the no-transaction
// directive.
func hasNoTransactionDirective(content []byte) bool {
	firstLine, _, _ := bytes.Cut(content, []byte{'\n'})
	return bytes.Equal(bytes.TrimSpace(firstLine), noTransactionDirective)
}
