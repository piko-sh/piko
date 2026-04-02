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

// MigrationCatalogueProvider builds a schema catalogue by reading migration
// SQL files and replaying their DDL statements. This is the default catalogue
// provider, suitable for engines that use file-based migration workflows.
type MigrationCatalogueProvider struct {
	// engine holds the SQL dialect parser and DDL interpreter.
	engine EnginePort

	// fileReader holds the filesystem access adapter.
	fileReader FileReaderPort

	// directory holds the path to the migration files.
	directory string
}

// NewMigrationCatalogueProvider creates a catalogue provider
// that reads migration files from the given directory.
//
// Takes engine (EnginePort) which provides SQL parsing and
// DDL interpretation.
// Takes fileReader (FileReaderPort) which provides
// filesystem access.
// Takes directory (string) which is the path to the
// migration files.
//
// Returns *MigrationCatalogueProvider which is ready to
// build catalogues.
func NewMigrationCatalogueProvider(
	engine EnginePort,
	fileReader FileReaderPort,
	directory string,
) *MigrationCatalogueProvider {
	return &MigrationCatalogueProvider{
		engine:     engine,
		fileReader: fileReader,
		directory:  directory,
	}
}

// BuildCatalogue reads migration files from the configured
// directory, parses their DDL statements via the engine
// adapter, and replays the mutations to construct the schema
// catalogue.
//
// Returns *querier_dto.Catalogue which holds the built
// schema state.
// Returns []querier_dto.SourceError which contains any
// diagnostics from parsing or applying migrations.
// Returns error when reading migration files fails.
func (provider *MigrationCatalogueProvider) BuildCatalogue(
	ctx context.Context,
) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
	builder := newCatalogueBuilder(provider.engine)

	migrationFiles, readError := readMigrationFiles(ctx, provider.fileReader, provider.directory)
	if readError != nil {
		return nil, nil, readError
	}

	allDiagnostics := warnNonConformingMigrationFiles(ctx, provider.fileReader, provider.directory)
	for _, migration := range migrationFiles {
		diagnostics := builder.ApplyMigration(ctx, migration.filename, migration.content, migration.index)
		allDiagnostics = append(allDiagnostics, diagnostics...)
	}

	catalogue := builder.Catalogue()
	propagateDataAccess(catalogue)

	return catalogue, allDiagnostics, nil
}
