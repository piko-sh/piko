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

// Package db_engine_conformance holds a cross-engine behavioural conformance suite.
//
// It drives every SQL engine adapter through the public querier service and asserts that
// a shared set of invariants (statement splitting, parameter tracking, identifier
// quoting, and recursion-depth safety) holds identically across dialects. It exists to
// catch the drift class where a fix lands in one engine adapter but not its siblings,
// independently of the per-engine golden snapshots which can be regenerated and so cannot
// guard against drift.
package db_engine_conformance

import (
	"context"
	"io/fs"
	"path"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// migrationDir is the virtual directory the conformance harness serves migration files
	// from; it matches the path passed to the querier service as MigrationDirectory.
	migrationDir = "migrations"

	// queryDir is the virtual directory the conformance harness serves query files from.
	queryDir = "queries"
)

// virtualFileReader serves migration and query SQL from an in-memory map, so the
// conformance suite needs no temp files on disk. Paths are slash-joined to mirror the
// io/fs convention the querier readers use.
type virtualFileReader struct {
	// files maps a slash-joined virtual path to its raw SQL content.
	files map[string][]byte
}

// recordingEmitter captures the catalogue and analysed queries the querier service
// produces, so invariants can be asserted against them. It satisfies
// querier_domain.CodeEmitterPort with no-op generation.
type recordingEmitter struct {
	// catalogue is the catalogue captured from the most recent EmitModels call.
	catalogue *querier_dto.Catalogue

	// queries are the analysed queries captured from the most recent EmitQueries call.
	queries []*querier_dto.AnalysedQuery
}

// virtualDirEntry is a minimal fs.DirEntry for a virtual file (never a directory).
type virtualDirEntry struct {
	// name is the entry's base filename.
	name string
}

// newVirtualFileReader returns a virtualFileReader populated with the given migration and
// query files. The maps are keyed by bare filename (for example "001_schema.up.sql").
//
// Takes migrations (map[string]string) which holds migration filename to SQL content.
// Takes queries (map[string]string) which holds query filename to SQL content.
//
// Returns *virtualFileReader ready to serve those files under the migration and query
// dirs.
func newVirtualFileReader(migrations, queries map[string]string) *virtualFileReader {
	reader := &virtualFileReader{files: make(map[string][]byte, len(migrations)+len(queries))}
	for name, content := range migrations {
		reader.files[path.Join(migrationDir, name)] = []byte(content)
	}
	for name, content := range queries {
		reader.files[path.Join(queryDir, name)] = []byte(content)
	}
	return reader
}

// ReadFile returns the content stored for the given virtual path.
//
// Takes filePath (string) which is the slash-joined virtual path.
//
// Returns []byte which is the file content.
// Returns error which is fs.ErrNotExist when the path is unknown.
func (reader *virtualFileReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	content, exists := reader.files[filePath]
	if !exists {
		return nil, &fs.PathError{Op: "open", Path: filePath, Err: fs.ErrNotExist}
	}
	return content, nil
}

// ReadDir lists the entries directly under the given virtual directory, sorted by name.
//
// Takes directory (string) which is the virtual directory path.
//
// Returns []fs.DirEntry which holds the files directly under directory.
// Returns error which is always nil for the in-memory store.
func (reader *virtualFileReader) ReadDir(_ context.Context, directory string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	for filePath := range reader.files {
		if path.Dir(filePath) == directory {
			entries = append(entries, virtualDirEntry{name: path.Base(filePath)})
		}
	}
	slices.SortFunc(entries, func(first, second fs.DirEntry) int {
		return strings.Compare(first.Name(), second.Name())
	})
	return entries, nil
}

// Name returns the entry's base filename.
//
// Returns string which is the filename.
func (entry virtualDirEntry) Name() string { return entry.name }

// IsDir reports whether the entry is a directory; virtual entries are always files.
//
// Returns bool which is always false.
func (virtualDirEntry) IsDir() bool { return false }

// Type returns the entry's file-mode type bits; virtual entries are regular files.
//
// Returns fs.FileMode which is the zero value (regular file).
func (virtualDirEntry) Type() fs.FileMode { return 0 }

// Info is unsupported for virtual entries; the querier readers never call it.
//
// Returns fs.FileInfo which is always nil.
// Returns error which is always fs.ErrInvalid.
func (virtualDirEntry) Info() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

// EmitModels records the catalogue and emits nothing.
//
// Takes catalogue (*querier_dto.Catalogue) which is the catalogue to record.
//
// Returns []querier_dto.GeneratedFile which is always nil.
// Returns error which is always nil.
func (emitter *recordingEmitter) EmitModels(
	_ string,
	catalogue *querier_dto.Catalogue,
	_ *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	emitter.catalogue = catalogue
	return nil, nil
}

// EmitQueries records the analysed queries and emits nothing.
//
// Takes queries ([]*querier_dto.AnalysedQuery) which are the analysed queries to record.
//
// Returns []querier_dto.GeneratedFile which is always nil.
// Returns error which is always nil.
func (emitter *recordingEmitter) EmitQueries(
	_ string,
	queries []*querier_dto.AnalysedQuery,
	_ *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	emitter.queries = queries
	return nil, nil
}

// EmitQuerier emits a stub file.
//
// Returns querier_dto.GeneratedFile which is a placeholder.
// Returns error which is always nil.
func (*recordingEmitter) EmitQuerier(_ string, _ querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{Name: "querier.go", Content: []byte("// stub")}, nil
}

// EmitPrepared emits a stub file.
//
// Returns querier_dto.GeneratedFile which is a placeholder.
// Returns error which is always nil.
func (*recordingEmitter) EmitPrepared(_ string, _ []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{Name: "prepared.go", Content: []byte("// stub")}, nil
}

// EmitOTel emits a stub file.
//
// Returns querier_dto.GeneratedFile which is a placeholder.
// Returns error which is always nil.
func (*recordingEmitter) EmitOTel(_ string, _ []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{Name: "otel.go", Content: []byte("// stub")}, nil
}

// runGenerate drives the querier service for one engine over the supplied migration and
// query files, returning the recording emitter, the generation result, and any error.
//
// Takes engine (querier_domain.EnginePort) which is the dialect adapter under test.
// Takes migrations (map[string]string) which holds migration filename to SQL.
// Takes queries (map[string]string) which holds query filename to SQL.
//
// Returns *recordingEmitter which captured the catalogue and analysed queries.
// Returns *querier_dto.GenerationResult which holds the run diagnostics.
// Returns error when the service cannot be constructed or generation fails fatally.
func runGenerate(
	ctx context.Context,
	engine querier_domain.EnginePort,
	migrations, queries map[string]string,
) (*recordingEmitter, *querier_dto.GenerationResult, error) {
	reader := newVirtualFileReader(migrations, queries)
	emitter := &recordingEmitter{}
	service, serviceError := querier_domain.NewQuerierService(querier_domain.QuerierPorts{
		Engine:     engine,
		Emitter:    emitter,
		FileReader: reader,
	})
	if serviceError != nil {
		return nil, nil, serviceError
	}
	result, generateError := service.GenerateDatabase(ctx, "conformance", &querier_dto.DatabaseConfig{
		MigrationDirectory: migrationDir,
		QueryDirectory:     queryDir,
	})
	return emitter, result, generateError
}

// catalogueTableNames returns the lower-cased set of table names across every schema in
// the catalogue, so quoting/splitting assertions are dialect-agnostic.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state.
//
// Returns map[string]struct{} which is the set of lower-cased table names.
func catalogueTableNames(catalogue *querier_dto.Catalogue) map[string]struct{} {
	names := make(map[string]struct{})
	if catalogue == nil {
		return names
	}
	for _, schema := range catalogue.Schemas {
		if schema == nil {
			continue
		}
		for tableName := range schema.Tables {
			names[strings.ToLower(tableName)] = struct{}{}
		}
	}
	return names
}

// findQuery returns the analysed query with the given name, or nil when absent.
//
// Takes queries ([]*querier_dto.AnalysedQuery) which holds the analysed queries.
// Takes name (string) which is the query name to find.
//
// Returns *querier_dto.AnalysedQuery which is the matching query, or nil.
func findQuery(queries []*querier_dto.AnalysedQuery, name string) *querier_dto.AnalysedQuery {
	for _, query := range queries {
		if query != nil && query.Name == name {
			return query
		}
	}
	return nil
}
