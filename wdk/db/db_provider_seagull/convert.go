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

package db_provider_seagull

import (
	"context"

	"piko.sh/piko/internal/sqldb/sqldb_domain"
	"piko.sh/piko/internal/sqldb/sqldb_dto"

	"seagull.sh/seagull"
	"seagull.sh/seagull/sdk/emit"
	"seagull.sh/seagull/sdk/migrate"
)

// migrationService presents a Seagull migrator through Piko's port.
type migrationService struct {
	// inner is the Seagull migrator.
	inner *migrate.Migrator
}

var (
	_ sqldb_domain.MigrationService = (*migrationService)(nil)

	_ sqldb_domain.SeedService = (*seedService)(nil)
)

// Up applies every pending migration.
//
// Returns int which is the number applied.
// Returns error which matches both Piko's and Seagull's error vocabulary.
func (s *migrationService) Up(ctx context.Context) (int, error) {
	applied, err := s.inner.Up(ctx)
	return applied, translateError(err)
}

// Down reverts the most recent steps migrations.
//
// Takes steps (int) which is how many to revert.
//
// Returns int which is the number reverted.
// Returns error which matches both error vocabularies.
func (s *migrationService) Down(ctx context.Context, steps int) (int, error) {
	reverted, err := s.inner.Down(ctx, steps)
	return reverted, translateError(err)
}

// Status reports every migration and whether it has been applied.
//
// Returns []sqldb_dto.MigrationStatus which is ordered by version.
// Returns error which matches both error vocabularies.
func (s *migrationService) Status(ctx context.Context) ([]sqldb_dto.MigrationStatus, error) {
	statuses, err := s.inner.Status(ctx)
	if err != nil {
		return nil, translateError(err)
	}
	result := make([]sqldb_dto.MigrationStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, sqldb_dto.MigrationStatus{
			AppliedAt:        status.AppliedAt,
			LastStatement:    status.LastStatement,
			Name:             status.Name,
			Filename:         status.Filename,
			Version:          status.Version,
			Applied:          status.Applied,
			ChecksumMatch:    status.ChecksumMatch,
			HasDownMigration: status.HasDownMigration,
			Dirty:            status.Dirty,
		})
	}
	return result, nil
}

// UpTo applies pending migrations up to and including targetVersion.
//
// Takes targetVersion (int64) which is the last version to apply.
//
// Returns int which is the number applied.
// Returns error which matches both error vocabularies.
func (s *migrationService) UpTo(ctx context.Context, targetVersion int64) (int, error) {
	applied, err := s.inner.UpTo(ctx, targetVersion)
	return applied, translateError(err)
}

// DownTo reverts migrations newer than targetVersion.
//
// Takes targetVersion (int64) which is the version to keep.
//
// Returns int which is the number reverted.
// Returns error which matches both error vocabularies.
func (s *migrationService) DownTo(ctx context.Context, targetVersion int64) (int, error) {
	reverted, err := s.inner.DownTo(ctx, targetVersion)
	return reverted, translateError(err)
}

// Validate checks applied migrations against the files.
//
// Returns error which matches both error vocabularies.
func (s *migrationService) Validate(ctx context.Context) error {
	return translateError(s.inner.Validate(ctx))
}

// seedService presents a Seagull seeder through Piko's port.
type seedService struct {
	// inner is the Seagull seeder.
	inner *migrate.Seeder
}

// Apply runs every seed that has not been applied.
//
// Returns int which is the number applied.
// Returns error which matches both error vocabularies.
func (s *seedService) Apply(ctx context.Context) (int, error) {
	applied, err := s.inner.Apply(ctx)
	return applied, translateError(err)
}

// Status reports every seed and whether it has been applied.
//
// Returns []sqldb_dto.SeedStatus which is ordered by version.
// Returns error which matches both error vocabularies.
func (s *seedService) Status(ctx context.Context) ([]sqldb_dto.SeedStatus, error) {
	statuses, err := s.inner.Status(ctx)
	if err != nil {
		return nil, translateError(err)
	}
	result := make([]sqldb_dto.SeedStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, sqldb_dto.SeedStatus{
			AppliedAt:     status.AppliedAt,
			Name:          status.Name,
			Filename:      status.Filename,
			Version:       status.Version,
			Applied:       status.Applied,
			ChecksumMatch: status.ChecksumMatch,
		})
	}
	return result, nil
}

// Reseed clears the seed history and applies every seed again.
//
// Returns int which is the number applied.
// Returns error which matches both error vocabularies.
func (s *seedService) Reseed(ctx context.Context) (int, error) {
	applied, err := s.inner.Reseed(ctx)
	return applied, translateError(err)
}

// fromSeagullDirection converts a Seagull migration direction.
//
// Takes direction (migrate.MigrationDirection) which is Seagull's value.
//
// Returns sqldb_dto.MigrationDirection which is Piko's value.
func fromSeagullDirection(direction migrate.MigrationDirection) sqldb_dto.MigrationDirection {
	if direction == migrate.MigrationDirectionDown {
		return sqldb_dto.DirectionDown
	}
	return sqldb_dto.DirectionUp
}

// fromSeagullHookContext converts a per-migration hook context.
//
// Takes hookContext (migrate.MigrationHookContext) which is Seagull's value.
//
// Returns sqldb_dto.MigrationHookContext which is Piko's value.
func fromSeagullHookContext(hookContext migrate.MigrationHookContext) sqldb_dto.MigrationHookContext {
	return sqldb_dto.MigrationHookContext{
		Name:      hookContext.Name,
		Version:   hookContext.Version,
		Direction: fromSeagullDirection(hookContext.Direction),
	}
}

// fromSeagullRunHookContext converts a run hook context.
//
// Takes hookContext (migrate.MigrationRunHookContext) which is Seagull's value.
//
// Returns sqldb_dto.MigrationRunHookContext which is Piko's value.
func fromSeagullRunHookContext(hookContext migrate.MigrationRunHookContext) sqldb_dto.MigrationRunHookContext {
	return sqldb_dto.MigrationRunHookContext{
		PendingVersions: hookContext.PendingVersions,
		PendingCount:    hookContext.PendingCount,
		Direction:       fromSeagullDirection(hookContext.Direction),
	}
}

// fromSeagullFiles converts generated files.
//
// Takes files ([]emit.GeneratedFile) which are Seagull's files.
//
// Returns []sqldb_dto.GeneratedFile which are Piko's files.
func fromSeagullFiles(files []emit.GeneratedFile) []sqldb_dto.GeneratedFile {
	result := make([]sqldb_dto.GeneratedFile, 0, len(files))
	for _, file := range files {
		result = append(result, sqldb_dto.GeneratedFile{Name: file.Name, Content: file.Content})
	}
	return result
}

// fromSeagullDiagnostics converts diagnostics.
//
// Takes diagnostics ([]seagull.SourceError) which are Seagull's diagnostics.
//
// Returns []sqldb_dto.SourceError which are Piko's diagnostics.
func fromSeagullDiagnostics(diagnostics []seagull.SourceError) []sqldb_dto.SourceError {
	if len(diagnostics) == 0 {
		return nil
	}
	result := make([]sqldb_dto.SourceError, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		result = append(result, sqldb_dto.SourceError{
			Filename:   diagnostic.Filename,
			Message:    diagnostic.Message,
			Code:       diagnostic.Code,
			Suggestion: diagnostic.Suggestion,
			Line:       diagnostic.Line,
			Column:     diagnostic.Column,
			EndLine:    diagnostic.EndLine,
			EndColumn:  diagnostic.EndColumn,
			Severity:   fromSeagullSeverity(diagnostic.Severity),
		})
	}
	return result
}

// fromSeagullSeverity converts a diagnostic severity.
//
// Takes severity (seagull.ErrorSeverity) which is Seagull's value.
//
// Returns sqldb_dto.Severity which is Piko's value.
func fromSeagullSeverity(severity seagull.ErrorSeverity) sqldb_dto.Severity {
	switch severity {
	case seagull.SeverityWarning:
		return sqldb_dto.SeverityWarning
	case seagull.SeverityHint:
		return sqldb_dto.SeverityHint
	default:
		return sqldb_dto.SeverityError
	}
}

// toSeagullTypeOverrides converts type overrides.
//
// Takes overrides ([]sqldb_dto.TypeOverride) which are Piko's overrides.
//
// Returns []seagull.TypeOverride which are Seagull's overrides.
func toSeagullTypeOverrides(overrides []sqldb_dto.TypeOverride) []seagull.TypeOverride {
	if len(overrides) == 0 {
		return nil
	}
	result := make([]seagull.TypeOverride, 0, len(overrides))
	for _, override := range overrides {
		result = append(result, seagull.TypeOverride{
			SQLTypeName: override.SQLTypeName,
			GoPackage:   override.GoPackage,
			GoName:      override.GoName,
		})
	}
	return result
}

// toSeagullCustomFunctions converts custom function declarations.
//
// Takes functions ([]sqldb_dto.CustomFunction) which are Piko's declarations.
//
// Returns []seagull.CustomFunction which are Seagull's declarations.
func toSeagullCustomFunctions(functions []sqldb_dto.CustomFunction) []seagull.CustomFunction {
	if len(functions) == 0 {
		return nil
	}
	result := make([]seagull.CustomFunction, 0, len(functions))
	for _, function := range functions {
		result = append(result, seagull.CustomFunction{
			Name:         function.Name,
			ReturnType:   function.ReturnType,
			Nullable:     function.Nullable,
			DataAccess:   function.DataAccess,
			Arguments:    function.Arguments,
			MinArguments: function.MinArguments,
			IsAggregate:  function.IsAggregate,
			IsVariadic:   function.IsVariadic,
		})
	}
	return result
}
