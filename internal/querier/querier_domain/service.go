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
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// QuerierPorts holds the port interfaces required by the querier service.
type QuerierPorts struct {
	// Engine is the SQL dialect parser and analyser.
	Engine EnginePort

	// Emitter generates source code from analysed queries.
	Emitter CodeEmitterPort

	// FileReader reads migration and query SQL files.
	FileReader FileReaderPort

	// CatalogueProvider optionally overrides how the schema catalogue is built.
	// When nil, the service falls back to migration-file-based catalogue
	// building using the Engine and FileReader ports.
	CatalogueProvider CatalogueProviderPort
}

// querierService orchestrates the three-phase pipeline: catalogue building
// from migrations, query analysis with type resolution, and code generation
// via the emitter adapter. Each call to GenerateDatabase creates fresh
// instances of all internal components - the service itself is stateless and
// safe for concurrent use.
type querierService struct {
	// engine holds the SQL dialect parser and analyser.
	engine EnginePort

	// emitter holds the code generation adapter.
	emitter CodeEmitterPort

	// fileReader holds the filesystem access adapter.
	fileReader FileReaderPort

	// catalogueProvider holds an optional override for
	// catalogue building.
	catalogueProvider CatalogueProviderPort
}

// NewQuerierService creates a new querier service from the
// given ports.
//
// Takes ports (QuerierPorts) which holds the required
// adapter interfaces.
//
// Returns QuerierServicePort which is the initialised
// service.
// Returns error when any required port is nil.
func NewQuerierService(ports QuerierPorts) (QuerierServicePort, error) {
	if ports.Engine == nil {
		return nil, ErrMissingEnginePort
	}
	if ports.Emitter == nil {
		return nil, ErrMissingEmitterPort
	}
	if ports.FileReader == nil {
		return nil, ErrMissingFileReaderPort
	}

	return &querierService{
		engine:            ports.Engine,
		emitter:           ports.Emitter,
		fileReader:        ports.FileReader,
		catalogueProvider: ports.CatalogueProvider,
	}, nil
}

// GenerateDatabase runs the full three-phase pipeline for a
// named database connection.
//
// Takes name (string) which identifies the database
// connection.
// Takes config (*querier_dto.DatabaseConfig) which specifies
// the engine, migration paths, and query paths.
//
// Returns *querier_dto.GenerationResult which contains the
// generated files and any diagnostics.
// Returns error when generation fails fatally.
func (s *querierService) GenerateDatabase(
	ctx context.Context,
	name string,
	config *querier_dto.DatabaseConfig,
	_ string,
) (*querier_dto.GenerationResult, error) {
	ctx, logger := logger_domain.From(ctx, log)
	ctx, span, _ := log.Span(ctx, "QuerierService.GenerateDatabase")
	defer span.End()

	start := time.Now()
	generationCount.Add(ctx, 1)
	logger.Trace("starting database generation", logger_domain.String("database", name))

	catalogue, catalogueDiagnostics, catalogueError := s.buildCatalogue(ctx, config)
	if catalogueError != nil {
		generationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("building catalogue for %s: %w", name, catalogueError)
	}

	queries, queryDiagnostics, queryError := s.analyseQueries(ctx, catalogue, config)
	if queryError != nil {
		generationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("analysing queries for %s: %w", name, queryError)
	}

	allDiagnostics := make([]querier_dto.SourceError, 0, len(catalogueDiagnostics)+len(queryDiagnostics))
	allDiagnostics = append(allDiagnostics, catalogueDiagnostics...)
	allDiagnostics = append(allDiagnostics, queryDiagnostics...)

	if diagnosticsContainErrors(allDiagnostics) {
		generationDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
		generationErrorCount.Add(ctx, 1)
		logger.Warn("generation aborted due to errors",
			logger_domain.String("database", name),
			logger_domain.Int("errorCount", len(allDiagnostics)),
		)
		return &querier_dto.GenerationResult{Diagnostics: allDiagnostics}, nil
	}

	typeMapper := NewTypeMapper(s.engine.BuiltinTypes())
	mappings := typeMapper.BuildMappingTable(config.TypeOverrides)

	generatedFiles, emitError := s.emitAllFiles(ctx, name, catalogue, queries, mappings)
	if emitError != nil {
		return nil, emitError
	}

	duration := time.Since(start).Milliseconds()
	generationDuration.Record(ctx, float64(duration))

	logger.Trace("database generation complete",
		logger_domain.String("database", name),
		logger_domain.Int("files", len(generatedFiles)),
		logger_domain.Int("queries", len(queries)),
		logger_domain.Int64("durationMs", duration),
	)

	return &querier_dto.GenerationResult{
		Files:       generatedFiles,
		Diagnostics: allDiagnostics,
	}, nil
}

// emitAllFiles generates all output files (models, queries,
// querier struct, and prepared statements) via the emitter.
//
// Takes name (string) which is used as the package name.
// Takes catalogue (*querier_dto.Catalogue) which holds the
// schema state.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the
// type-checked queries.
// Takes mappings (*querier_dto.TypeMappingTable) which
// defines SQL-to-Go type mappings.
//
// Returns []querier_dto.GeneratedFile which contains all
// generated source files.
// Returns error when any emission step fails.
func (s *querierService) emitAllFiles(
	ctx context.Context,
	name string,
	catalogue *querier_dto.Catalogue,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	var generatedFiles []querier_dto.GeneratedFile

	modelFiles, modelError := s.emitter.EmitModels(name, catalogue, mappings)
	if modelError != nil {
		generationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("emitting models for %s: %w", name, modelError)
	}
	generatedFiles = append(generatedFiles, modelFiles...)

	queryFiles, queryFileError := s.emitter.EmitQueries(name, queries, mappings)
	if queryFileError != nil {
		generationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("emitting queries for %s: %w", name, queryFileError)
	}
	generatedFiles = append(generatedFiles, queryFiles...)

	var capabilities querier_dto.QueryCapabilities
	for _, query := range queries {
		switch query.Command {
		case querier_dto.QueryCommandBatch:
			capabilities |= querier_dto.CapabilityBatch
		case querier_dto.QueryCommandCopyFrom:
			capabilities |= querier_dto.CapabilityCopyFrom
		}
	}

	querierFile, querierError := s.emitter.EmitQuerier(name, capabilities)
	if querierError != nil {
		generationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("emitting querier for %s: %w", name, querierError)
	}
	generatedFiles = append(generatedFiles, querierFile)

	preparedFile, preparedError := s.emitter.EmitPrepared(name, queries)
	if preparedError != nil {
		generationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("emitting prepared statements for %s: %w", name, preparedError)
	}
	generatedFiles = append(generatedFiles, preparedFile)

	otelFile, otelError := s.emitter.EmitOTel(name, queries)
	if otelError != nil {
		generationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("emitting otel for %s: %w", name, otelError)
	}
	generatedFiles = append(generatedFiles, otelFile)

	return generatedFiles, nil
}

// diagnosticsContainErrors reports whether any diagnostic
// has error severity.
//
// Takes diagnostics ([]querier_dto.SourceError) which holds
// the diagnostics to check.
//
// Returns bool which is true if any diagnostic has error
// severity.
func diagnosticsContainErrors(diagnostics []querier_dto.SourceError) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == querier_dto.SeverityError {
			return true
		}
	}
	return false
}

// buildCatalogue executes Phase 1: reads migration files
// and replays DDL statements to build the schema catalogue.
//
// Takes config (*querier_dto.DatabaseConfig) which specifies
// the migration directory.
//
// Returns *querier_dto.Catalogue which holds the built
// schema state.
// Returns []querier_dto.SourceError which contains any
// diagnostics.
// Returns error when catalogue building fails.
func (s *querierService) buildCatalogue(
	ctx context.Context,
	config *querier_dto.DatabaseConfig,
) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
	ctx, span, _ := log.Span(ctx, "QuerierService.buildCatalogue")
	defer span.End()

	start := time.Now()
	catalogueBuildCount.Add(ctx, 1)

	provider := s.catalogueProvider
	if provider == nil {
		provider = NewMigrationCatalogueProvider(s.engine, s.fileReader, config.MigrationDirectory)
	}

	catalogue, diagnostics, buildError := provider.BuildCatalogue(ctx)

	duration := time.Since(start).Milliseconds()
	catalogueBuildDuration.Record(ctx, float64(duration))

	return catalogue, diagnostics, buildError
}

// analyseQueries executes Phase 2: reads query files and
// runs the full analysis pipeline on each query.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the
// schema state.
// Takes config (*querier_dto.DatabaseConfig) which specifies
// the query directory and custom functions.
//
// Returns []*querier_dto.AnalysedQuery which holds the
// type-checked queries.
// Returns []querier_dto.SourceError which contains any
// diagnostics.
// Returns error when query reading fails.
func (s *querierService) analyseQueries(
	ctx context.Context,
	catalogue *querier_dto.Catalogue,
	config *querier_dto.DatabaseConfig,
) ([]*querier_dto.AnalysedQuery, []querier_dto.SourceError, error) {
	ctx, span, _ := log.Span(ctx, "QuerierService.analyseQueries")
	defer span.End()

	start := time.Now()
	queryAnalysisCount.Add(ctx, 1)

	queryFiles, readError := readMigrationFiles(ctx, s.fileReader, config.QueryDirectory)
	if readError != nil {
		return nil, nil, readError
	}

	if len(config.CustomFunctions) > 0 {
		mergeCustomFunctions(catalogue, s.engine, config.CustomFunctions)
	}

	analyser := newQueryAnalyser(s.engine, catalogue)

	commentStyle := s.engine.CommentStyle()

	var allQueries []*querier_dto.AnalysedQuery
	var allDiagnostics []querier_dto.SourceError

	for _, queryFile := range queryFiles {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}

		blocks := splitQueryFile(queryFile.content, commentStyle)

		for _, block := range blocks {
			query, diagnostics := analyser.AnalyseQuery(ctx, block, queryFile.filename)
			allDiagnostics = append(allDiagnostics, diagnostics...)

			if query != nil {
				allQueries = append(allQueries, query)
			}
		}
	}

	duplicateDiagnostics := analyser.validator.ValidateDuplicateNames(allQueries)
	allDiagnostics = append(allDiagnostics, duplicateDiagnostics...)

	duration := time.Since(start).Milliseconds()
	queryAnalysisDuration.Record(ctx, float64(duration))

	return allQueries, allDiagnostics, nil
}
