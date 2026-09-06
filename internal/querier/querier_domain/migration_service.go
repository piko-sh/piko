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
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// errorFormatEnsuringMigrationTable holds the format string for wrapping errors from
	// EnsureMigrationTable.
	errorFormatEnsuringMigrationTable = "ensuring migration table: %w"

	// errorFormatReadingAppliedVersions holds the format string for wrapping errors from
	// AppliedVersions.
	errorFormatReadingAppliedVersions = "reading applied versions: %w"

	// logFieldVersion holds the structured log field name for migration version numbers.
	logFieldVersion = "version"
)

// MigrationServiceOption configures optional behaviour of the migration service.
type MigrationServiceOption func(*migrationService)

// WithNonBlockingLock configures the migration service to use a non-blocking lock
// acquisition. If the lock is already held, operations return ErrLockNotAcquired
// immediately instead of waiting.
//
// Returns MigrationServiceOption which sets the nonBlockingLock flag on the service.
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
// Takes executor (MigrationExecutorPort) which provides database-specific migration
// operations.
// Takes fileReader (FileReaderPort) which provides filesystem access.
// Takes directory (string) which is the path to the migration files.
// Takes options (...MigrationServiceOption) which configure optional behaviour such as
// non-blocking lock acquisition.
//
// Returns MigrationServicePort which is ready to apply or roll back migrations.
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
// Returns error when migration reading, checksum validation, lock acquisition, or
// execution fails.
func (service *migrationService) Up(ctx context.Context) (int, error) {
	return service.applyUpMigrations(ctx, nil)
}

// UpTo applies pending up migrations up to and including the target version.
//
// Takes targetVersion (int64) which specifies the maximum version to apply.
//
// Returns int which is the number of migrations applied.
// Returns error when migration reading, checksum validation, lock acquisition, or
// execution fails.
func (service *migrationService) UpTo(ctx context.Context, targetVersion int64) (int, error) {
	return service.applyUpMigrations(ctx, &targetVersion)
}

// Down rolls back the last n applied migrations in reverse version order.
//
// Takes steps (int) which specifies how many migrations to roll back.
//
// Returns int which is the number of migrations rolled back.
// Returns error when migration reading, lock acquisition, or rollback execution fails.
func (service *migrationService) Down(ctx context.Context, steps int) (int, error) {
	return service.rollbackMigrations(ctx, &steps, nil)
}

// DownTo rolls back applied migrations down to (but not including) the target version.
//
// Takes targetVersion (int64) which specifies the version to roll back to.
//
// Returns int which is the number of migrations rolled back.
// Returns error when migration reading, lock acquisition, or rollback execution fails.
func (service *migrationService) DownTo(ctx context.Context, targetVersion int64) (int, error) {
	return service.rollbackMigrations(ctx, nil, &targetVersion)
}

// Status returns the list of all known migrations and their applied state.
//
// Returns []querier_dto.MigrationStatus which holds the status of each known migration.
// Returns error when reading files or querying applied versions fails.
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

// Validate checks that all applied migration checksums match their on-disk files without
// executing anything.
//
// Returns error when file reading, table initialisation, version querying, or checksum
// validation fails.
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

// applyUpMigrations reads migration files, validates checksums, acquires the advisory
// lock, and executes all pending up migrations up to the optional target version.
//
// Takes targetVersion (*int64) which specifies the maximum version to apply, or nil to
// apply all pending migrations.
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

	if lockError := service.acquireLock(ctx); lockError != nil {
		return 0, &LockAcquisitionError{Cause: lockError}
	}
	defer service.releaseLock(context.WithoutCancel(ctx))

	applied, appliedError = service.executor.AppliedVersions(ctx)
	if appliedError != nil {
		return 0, fmt.Errorf("reading applied versions under lock: %w", appliedError)
	}

	if checksumError := validateChecksums(upFiles, applied); checksumError != nil {
		return 0, checksumError
	}

	pending = computePending(upFiles, applied)
	pending = filterByTargetVersion(pending, targetVersion)
	if len(pending) == 0 {
		return 0, nil
	}

	service.warnSkippedMigrations(ctx, pending, applied)

	downChecksumsByVersion := buildDownChecksumMap(allFiles)

	return service.executePendingUp(ctx, pending, downChecksumsByVersion)
}

// rollbackMigrations reads migration files, acquires the advisory lock, and rolls back
// applied migrations by step count or down to a target version.
//
// Takes steps (*int) which specifies how many migrations to roll back, or nil to use
// targetVersion instead.
// Takes targetVersion (*int64) which specifies the version to roll back to, or nil to use
// steps instead.
//
// Returns int which is the number of migrations rolled back.
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

	if lockError := service.acquireLock(ctx); lockError != nil {
		return 0, &LockAcquisitionError{Cause: lockError}
	}
	defer service.releaseLock(context.WithoutCancel(ctx))

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

	return service.executeRollbacks(ctx, applied, rollbackCount, downFilesByVersion)
}

// warnSkippedMigrations logs a warning for each pending migration whose version is
// earlier than the maximum applied version.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations to check.
// Takes applied ([]querier_dto.AppliedMigration) which holds the already applied
// migrations.
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

// executePendingUp runs before-run hooks, applies each pending migration with its
// before/after hooks, and runs after-run hooks. Before processing pending migrations, any
// dirty migration is detected and either retried (if it matches the next pending version)
// or reported as a blocking error.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations to apply.
// Takes downChecksumsByVersion (map[int64]string) which maps versions to their
// down-migration checksums.
//
// Returns int which is the number of migrations applied.
// Returns error when any hook or migration execution fails, or a dirty migration from a
// different version blocks progress.
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

// applyPendingUpMigrations iterates through pending migrations, running hooks and
// executing each in sequence.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations to apply.
// Takes downChecksumsByVersion (map[int64]string) which maps versions to their
// down-migration checksums.
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
// Takes applied ([]querier_dto.AppliedMigration) which holds the applied migration
// records.
//
// Returns *querier_dto.AppliedMigration which is the dirty migration, or nil if none is
// dirty.
// Returns int which is the last completed statement index to skip on retry (-1 if no
// statements completed).
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

// handleDirtyMigration checks whether a dirty migration can be retried (because it
// matches the next pending version) and retries it if so. If the dirty migration does not
// match the next pending version, a DirtyMigrationError is returned.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the pending migrations.
// Takes downChecksumsByVersion (map[int64]string) which maps versions to their
// down-migration checksums.
// Takes dirtyMigration (querier_dto.AppliedMigration) which is the dirty migration
// record.
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
// Takes pending ([]querier_dto.MigrationFile) which holds the pending migrations.
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

// executeRollbacks runs before-run hooks, rolls back the specified number of migrations
// in reverse order, and runs after-run hooks.
//
// Takes applied ([]querier_dto.AppliedMigration) which holds all applied migrations in
// version order.
// Takes steps (int) which specifies how many migrations to roll back from the end.
// Takes downFilesByVersion (map[int64]querier_dto.MigrationFile) which maps versions to
// their down-migration files.
//
// Returns int which is the number of migrations rolled back.
// Returns error when any hook or rollback execution fails.
func (service *migrationService) executeRollbacks(
	ctx context.Context,
	applied []querier_dto.AppliedMigration,
	steps int,
	downFilesByVersion map[int64]querier_dto.MigrationFile,
) (int, error) {
	applied = slices.Clone(applied)
	slices.SortFunc(applied, func(a, b querier_dto.AppliedMigration) int {
		return cmp.Compare(a.Version, b.Version)
	})

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

// executeSingleRollback validates the down checksum, runs before/after migration hooks,
// and executes a single rollback migration.
//
// Takes appliedMigration (querier_dto.AppliedMigration) which holds the applied migration
// record to roll back.
// Takes downFile (querier_dto.MigrationFile) which holds the down migration file content.
//
// Returns error when checksum validation, hook execution, or migration execution fails.
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

// validateDownChecksum checks that the down migration file checksum matches the checksum
// recorded when the up migration was applied.
//
// Takes appliedMigration (querier_dto.AppliedMigration) which holds the recorded down
// checksum.
// Takes downFile (querier_dto.MigrationFile) which holds the current file checksum.
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

// runBeforeRunHooks invokes all registered before-run hooks with a context built from the
// pending migration files.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations about to be
// applied.
// Takes direction (querier_dto.MigrationDirection) which indicates whether the run is up
// or down.
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
		if hookError := goroutine.SafeCall(ctx, "migration before-run-hook", func() error {
			return hook(ctx, hookContext)
		}); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runBeforeRunHooksFromVersions invokes all registered before-run hooks with a context
// built from explicit version numbers.
//
// Takes versions ([]int64) which holds the migration versions about to be processed.
// Takes direction (querier_dto.MigrationDirection) which indicates whether the run is up
// or down.
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
		if hookError := goroutine.SafeCall(ctx, "migration before-run-hook", func() error {
			return hook(ctx, hookContext)
		}); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runAfterRunHooks invokes all registered after-run hooks with a context built from the
// pending migration files and the count of applied migrations.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations that were
// processed.
// Takes direction (querier_dto.MigrationDirection) which indicates whether the run was up
// or down.
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
		if hookError := goroutine.SafeCall(ctx, "migration after-run-hook", func() error {
			return hook(ctx, hookContext, applied)
		}); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runAfterRunHooksFromVersions invokes all registered after-run hooks with a context
// built from explicit version numbers and the count of applied migrations.
//
// Takes versions ([]int64) which holds the migration versions that were processed.
// Takes direction (querier_dto.MigrationDirection) which indicates whether the run was up
// or down.
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
		if hookError := goroutine.SafeCall(ctx, "migration after-run-hook", func() error {
			return hook(ctx, hookContext, applied)
		}); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runBeforeMigrationHooks invokes all registered before-migration hooks for a single
// migration.
//
// Takes hookContext (MigrationHookContext) which holds the version, name, and direction
// of the migration.
//
// Returns error when any hook returns an error.
func (service *migrationService) runBeforeMigrationHooks(
	ctx context.Context,
	hookContext MigrationHookContext,
) error {
	for _, hook := range service.beforeMigrationHooks {
		if hookError := goroutine.SafeCall(ctx, "migration before-hook", func() error {
			return hook(ctx, hookContext)
		}); hookError != nil {
			return hookError
		}
	}
	return nil
}

// runAfterMigrationHooks invokes all registered after-migration hooks for a single
// migration.
//
// Takes hookContext (MigrationHookContext) which holds the version, name, and direction
// of the migration.
//
// Returns error when any hook returns an error.
func (service *migrationService) runAfterMigrationHooks(
	ctx context.Context,
	hookContext MigrationHookContext,
) error {
	for _, hook := range service.afterMigrationHooks {
		if hookError := goroutine.SafeCall(ctx, "migration after-hook", func() error {
			return hook(ctx, hookContext)
		}); hookError != nil {
			return hookError
		}
	}
	return nil
}

// acquireLock acquires the migration advisory lock, using either blocking or non-blocking
// mode depending on the service configuration.
//
// Returns error when the lock cannot be acquired.
func (service *migrationService) acquireLock(ctx context.Context) error {
	if service.nonBlockingLock {
		return service.executor.TryAcquireLock(ctx)
	}
	return service.executor.AcquireLock(ctx)
}

// releaseLock releases the migration advisory lock, logging an error if release fails.
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
// Takes direction (querier_dto.MigrationDirection) which specifies the direction to
// filter by.
//
// Returns []querier_dto.MigrationFile which holds only files matching the specified
// direction.
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
// Returns map[int64]querier_dto.MigrationFile which maps version numbers to their
// corresponding down migration files.
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

// buildDownChecksumMap returns a map of version to down-migration checksum for all
// versions that have a .down.sql file.
//
// Takes files ([]querier_dto.MigrationFile) which holds all migration files.
//
// Returns map[int64]string which maps version numbers to their down-migration checksums.
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
// Returns map[int64]bool which contains true for each version that has a down migration
// file.
func buildDownVersionSet(files []querier_dto.MigrationFile) map[int64]bool {
	result := make(map[int64]bool)
	for _, file := range files {
		if file.Direction == querier_dto.MigrationDirectionDown {
			result[file.Version] = true
		}
	}
	return result
}

// filterByTargetVersion filters pending migrations to only include those up to and
// including the target version.
//
// If targetVersion is nil, all pending migrations are returned.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the migrations to filter.
// Takes targetVersion (*int64) which specifies the maximum version to include, or nil to
// include all.
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
// If steps is provided, that value is used (clamped to applied length). If targetVersion
// is provided, it counts applied migrations with version greater than targetVersion.
//
// Takes applied ([]querier_dto.AppliedMigration) which holds all applied migrations.
// Takes steps (*int) which specifies the number of migrations to roll back, or nil to use
// targetVersion.
// Takes targetVersion (*int64) which specifies the version to roll back to, or nil to use
// steps.
//
// Returns int which is the number of migrations to roll back.
func computeRollbackSteps(
	applied []querier_dto.AppliedMigration,
	steps *int,
	targetVersion *int64,
) int {
	if steps != nil {
		if *steps < 0 {
			return 0
		}
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

// computePending returns up migration files that have not been successfully applied yet,
// sorted by version ascending. Dirty (partially-applied) migrations are treated as
// pending since they need to be retried.
//
// Takes upFiles ([]querier_dto.MigrationFile) which holds all up migration files.
// Takes applied ([]querier_dto.AppliedMigration) which holds already applied migrations.
//
// Returns []querier_dto.MigrationFile which holds the unapplied or dirty migrations
// sorted by version.
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

// validateChecksums verifies that all applied migrations have matching checksums with
// on-disk files.
//
// Takes upFiles ([]querier_dto.MigrationFile) which holds the on-disk migration files.
// Takes applied ([]querier_dto.AppliedMigration) which holds the applied migration
// records.
//
// Returns error when a file is missing or its checksum does not match the applied record.
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

// detectSkippedMigrations finds pending migrations whose version is earlier than the
// maximum applied version.
//
// These are migrations that were added after later migrations were already applied, for
// example from branch merges.
//
// Takes pending ([]querier_dto.MigrationFile) which holds the pending migrations to
// check.
// Takes applied ([]querier_dto.AppliedMigration) which holds the already applied
// migrations.
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

// hasNoTransactionDirective reports whether a migration must run outside a transaction.
//
// This is true when any statement carries an explicit `-- piko.migration(no_transaction:
// true)` directive, or when a statement is auto-detected as non-transactional such as
// CREATE INDEX CONCURRENTLY or VACUUM. A migration with no such statement runs in a
// single transaction.
//
// Detection is lexically aware: the content is scanned with string literals,
// dollar-quoted bodies, quoted identifiers, and comments removed before keywords are
// matched, so a literal or a dollar-quoted function body that only mentions CONCURRENTLY
// or begins a line with VACUUM cannot strip transactional safety. Only the code text of
// each statement is inspected, and VACUUM is matched only as the first keyword of a
// statement.
//
// Takes content ([]byte) which holds the migration file content.
//
// Returns bool which is true if the migration must run without BEGIN/COMMIT.
func hasNoTransactionDirective(content []byte) bool {
	source := string(content)
	if scanMigrationCommentDirective(source) {
		return true
	}
	return slices.ContainsFunc(scanMigrationStatementCode(source), statementRequiresNoTransaction)
}

// scanMigrationCommentDirective reports whether any line comment in the migration carries
// an explicit `piko.migration(no_transaction: true)` directive. Comments are inspected
// per physical line so a single-line directive is recognised regardless of the
// surrounding statement structure.
//
// Takes source (string) which is the migration file content.
//
// Returns bool which is true when an explicit no-transaction directive is present.
func scanMigrationCommentDirective(source string) bool {
	for line := range strings.SplitSeq(source, "\n") {
		body, isComment := migrationCommentBody(strings.TrimSpace(line))
		if !isComment {
			continue
		}
		if value, matched := migrationDirectiveBool(body, "no_transaction"); matched && value {
			return true
		}
	}
	return false
}

// scanMigrationStatementCode splits the migration into its executable statements with
// string literals, dollar-quoted bodies, quoted identifiers, and comments removed.
//
// Each statement is returned as upper-cased code text. Removing literal and comment text
// first means a CONCURRENTLY mention inside a string or a dollar-quoted body, or a VACUUM
// that only begins a line inside such a body, can never be mistaken for a statement
// keyword.
//
// Takes source (string) which is the migration file content.
//
// Returns []string which holds the upper-cased, literal-free code of each non-empty
// statement.
func scanMigrationStatementCode(source string) []string {
	var statements []string
	var current strings.Builder
	index := 0
	for index < len(source) {
		consumed, isTerminator := skipMigrationNonCode(source, index)
		if consumed > 0 {
			current.WriteByte(' ')
			index += consumed
			continue
		}
		if isTerminator {
			appendStatementCode(&statements, &current)
			index++
			continue
		}
		current.WriteByte(byteToUpper(source[index]))
		index++
	}
	appendStatementCode(&statements, &current)
	return statements
}

// skipMigrationNonCode reports how many bytes of non-code text begin at index in source.
//
// A non-zero count covers a line comment, block comment, single-quoted string,
// double-quoted identifier, or dollar-quoted block; the second return reports whether
// index marks a top-level statement terminator (`;`). A zero count with a false
// terminator means index is ordinary code.
//
// Takes source (string) which is the migration file content.
// Takes index (int) which is the byte offset to inspect.
//
// Returns int which is the number of non-code bytes consumed (zero when index is code).
// Returns bool which is true when index marks a statement terminator.
func skipMigrationNonCode(source string, index int) (int, bool) {
	character := source[index]
	switch {
	case character == ';':
		return 0, true
	case character == '-' && index+1 < len(source) && source[index+1] == '-':
		return skipToLineEnd(source, index) - index, false
	case character == '#':
		return skipToLineEnd(source, index) - index, false
	case character == '/' && index+1 < len(source) && source[index+1] == '*':
		return skipBlockComment(source, index) - index, false
	case character == '\'' || character == '"':
		end, _ := skipStringLiteral(source, index, character)
		return end - index, false
	case character == '$':
		if end, ok := skipDollarQuotedBlock(source, index); ok {
			return end - index, false
		}
		return 0, false
	default:
		return 0, false
	}
}

// appendStatementCode trims and stores the accumulated statement code when non-empty,
// then resets the builder for the next statement.
//
// Takes statements (*[]string) which collects each statement's code text.
// Takes current (*strings.Builder) which holds the in-progress statement code.
func appendStatementCode(statements *[]string, current *strings.Builder) {
	trimmed := strings.TrimSpace(current.String())
	if trimmed != "" {
		*statements = append(*statements, trimmed)
	}
	current.Reset()
}

// skipToLineEnd returns the index of the newline that ends the comment beginning at
// start, or the length of source when the comment runs to end of input.
//
// Takes source (string) which is the migration file content.
// Takes start (int) which is the byte offset of the comment opener.
//
// Returns int which is the index just past the comment body (at the newline or end).
func skipToLineEnd(source string, start int) int {
	index := start
	for index < len(source) && source[index] != '\n' {
		index++
	}
	return index
}

// skipBlockComment returns the index just past a `/* ... */` block comment beginning at
// start.
//
// An unterminated block comment is treated as running to end of input.
//
// Takes source (string) which is the migration file content.
// Takes start (int) which is the byte offset of the opening `/*`.
//
// Returns int which is the index just past the closing `*/` (or end of input).
func skipBlockComment(source string, start int) int {
	index := start + 2
	for index+1 < len(source) {
		if source[index] == '*' && source[index+1] == '/' {
			return index + 2
		}
		index++
	}
	return len(source)
}

// skipDollarQuotedBlock returns the index just past a Postgres-style dollar-quoted block
// (`$$...$$` or `$tag$...$tag$`) beginning at start, and whether start indeed opened one.
// An unterminated block is treated as running to end of input.
//
// Takes source (string) which is the migration file content.
// Takes start (int) which is the byte offset of the leading '$'.
//
// Returns int which is the index just past the closing tag (or end of input).
// Returns bool which is true when start opened a dollar-quoted block.
func skipDollarQuotedBlock(source string, start int) (int, bool) {
	tag, openLen, ok := readDollarTag(source, start)
	if !ok {
		return start, false
	}
	closer := "$" + tag + "$"
	index := start + openLen
	for index < len(source) {
		if source[index] == '$' && strings.HasPrefix(source[index:], closer) {
			return index + len(closer), true
		}
		index++
	}
	return len(source), true
}

// readDollarTag inspects the dollar-quote opener at start and returns its inner tag, the
// number of bytes the opener spans, and whether start marks a valid opener. A valid tag
// is a possibly-empty run of identifier characters bounded by '$' on each side, so `$1`
// (a positional parameter) is rejected.
//
// Takes source (string) which is the migration file content.
// Takes start (int) which is the byte offset of the leading '$'.
//
// Returns string which is the inner tag (empty for `$$`).
// Returns int which is the number of bytes the opener spans.
// Returns bool which is true when start marks a valid dollar-quote opener.
func readDollarTag(source string, start int) (string, int, bool) {
	if start >= len(source) || source[start] != '$' {
		return "", 0, false
	}
	end := start + 1
	for end < len(source) && source[end] != '$' {
		if !isIdentifierPart(source[end]) {
			return "", 0, false
		}
		end++
	}
	if end >= len(source) {
		return "", 0, false
	}
	return source[start+1 : end], end - start + 1, true
}

// byteToUpper upper-cases an ASCII letter, leaving every other byte unchanged.
//
// Takes character (byte) which is the byte to fold.
//
// Returns byte which is the upper-cased ASCII letter or the original byte.
func byteToUpper(character byte) byte {
	if character >= 'a' && character <= 'z' {
		return character - ('a' - 'A')
	}
	return character
}

// migrationCommentBody returns the directive text of a line comment (after a -- or #
// prefix) and whether the line was a comment.
//
// Takes trimmed (string) which is the whitespace-trimmed source line.
//
// Returns string which is the comment body after the prefix.
// Returns bool which is true when the line was a comment.
func migrationCommentBody(trimmed string) (string, bool) {
	if rest, ok := strings.CutPrefix(trimmed, "--"); ok {
		return strings.TrimSpace(rest), true
	}
	if rest, ok := strings.CutPrefix(trimmed, "#"); ok {
		return strings.TrimSpace(rest), true
	}
	return "", false
}

// statementRequiresNoTransaction reports whether an upper-cased, literal-free statement
// is one of the known statements that cannot run inside a transaction.
//
// The CONCURRENTLY keyword covers CREATE/DROP INDEX CONCURRENTLY and REINDEX
// CONCURRENTLY, and VACUUM cannot run inside a transaction either. The keyword is matched
// at word boundaries, and VACUUM only as the leading keyword, so an identifier that
// contains either substring does not strip transaction safety from an otherwise atomic
// migration. Because the input has already had string literals and quoted bodies removed,
// a mention inside a literal or function body cannot reach this check.
//
// Takes upperStatement (string) which is the upper-cased, literal-free statement code.
//
// Returns bool which is true when the statement cannot run inside a transaction.
func statementRequiresNoTransaction(upperStatement string) bool {
	return containsWholeWord(upperStatement, "CONCURRENTLY") || hasLeadingWord(upperStatement, "VACUUM")
}

// containsWholeWord reports whether word appears in haystack delimited by non-identifier
// characters on both sides (or the string boundary).
//
// Takes haystack (string) which is the text to search.
// Takes word (string) which is the whole word to look for.
//
// Returns bool which is true when word appears as a whole word in haystack.
func containsWholeWord(haystack, word string) bool {
	start := 0
	for {
		index := strings.Index(haystack[start:], word)
		if index < 0 {
			return false
		}
		begin := start + index
		end := begin + len(word)
		beforeOK := begin == 0 || !isIdentifierPart(haystack[begin-1])
		afterOK := end == len(haystack) || !isIdentifierPart(haystack[end])
		if beforeOK && afterOK {
			return true
		}
		start = begin + 1
	}
}

// hasLeadingWord reports whether line begins with word followed by a non-identifier
// character or end of string (so "VACUUM" matches but "VACUUMING" does not).
//
// Takes line (string) which is the text whose prefix is tested.
// Takes word (string) which is the leading word to look for.
//
// Returns bool which is true when line begins with word as a whole word.
func hasLeadingWord(line, word string) bool {
	if !strings.HasPrefix(line, word) {
		return false
	}
	return len(line) == len(word) || !isIdentifierPart(line[len(word)])
}
