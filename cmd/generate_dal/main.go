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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"piko.sh/piko/internal/querier/querier_adapters/emitter_go_sql"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// minimumArgCount is the minimum number of command-line arguments required: the program
	// name, base directory, and package name.
	minimumArgCount = 3

	// directoryPermissions defines the permissions for created output directories.
	directoryPermissions = 0o755

	// filePermissions defines the permissions for generated output files.
	filePermissions = 0o644
)

// fileReader is a filesystem-backed implementation of the file reader port used by the
// querier service.
type fileReader struct {
	// factory creates sandboxed filesystem handles for safe file access.
	factory safedisk.Factory
}

// ReadFile reads the entire contents of the file at the given path.
//
// Takes path (string) which is the absolute or relative file path to read.
//
// Returns []byte which is the raw file contents.
// Returns error when the file cannot be read.
func (r *fileReader) ReadFile(_ context.Context, path string) ([]byte, error) {
	parentDirectory := filepath.Dir(path)
	fileName := filepath.Base(path)
	sandbox, err := r.factory.Create("dal-read-file", parentDirectory, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox for %q: %w", parentDirectory, err)
	}
	defer func() { _ = sandbox.Close() }()

	return sandbox.ReadFile(fileName)
}

// ReadDir returns the directory entries for the given directory path.
//
// Takes directory (string) which is the directory to list.
//
// Returns []os.DirEntry which is the directory listing.
// Returns error when the directory cannot be read.
func (r *fileReader) ReadDir(_ context.Context, directory string) ([]os.DirEntry, error) {
	sandbox, err := r.factory.Create("dal-read-dir", directory, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox for %q: %w", directory, err)
	}
	defer func() { _ = sandbox.Close() }()

	return sandbox.ReadDir(".")
}

// main generates Go data-access-layer files from SQL migrations and query definitions. It
// expects a base directory containing migrations/ and queries/ subdirectories, a Go
// package name for the generated code, and an optional dialect (sqlite or postgres,
// default sqlite).
func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// selectEngine returns the SQL engine for the requested dialect.
//
// Takes dialect (string) which names the target dialect, either "sqlite" (the default for
// an empty value) or "postgres".
//
// Returns querier_domain.EnginePort which is the engine that parses and analyses the
// dialect.
// Returns error when the dialect is not recognised.
func selectEngine(dialect string) (querier_domain.EnginePort, error) {
	switch dialect {
	case "", "sqlite":
		return db_engine_sqlite.NewSQLiteEngine(), nil
	case "postgres":
		return db_engine_postgres.NewPostgresEngine(), nil
	default:
		return nil, fmt.Errorf("unsupported dialect %q (want sqlite or postgres)", dialect)
	}
}

// run parses command-line arguments and orchestrates DAL code generation.
//
// Returns error when arguments are missing or generation fails.
func run() error {
	if len(os.Args) < minimumArgCount {
		return errors.New("usage: generate_dal <base_dir> <package_name> [dialect]")
	}
	base := os.Args[1]
	pkgName := os.Args[2]
	dialect := "sqlite"
	if len(os.Args) > minimumArgCount {
		dialect = os.Args[minimumArgCount]
	}

	engine, err := selectEngine(dialect)
	if err != nil {
		return err
	}

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		return fmt.Errorf("factory: %w", err)
	}

	ctx := context.Background()
	service, err := querier_domain.NewQuerierService(querier_domain.QuerierPorts{
		Engine:     engine,
		Emitter:    emitter_go_sql.NewSQLEmitterForDialect(engine.Dialect()),
		FileReader: &fileReader{factory: factory},
	})
	if err != nil {
		return fmt.Errorf("service: %w", err)
	}

	migDir, _ := filepath.Abs(filepath.Join(base, "migrations"))
	queryDir, _ := filepath.Abs(filepath.Join(base, "queries"))

	result, err := service.GenerateDatabase(ctx, pkgName, &querier_dto.DatabaseConfig{
		MigrationDirectory: migDir,
		QueryDirectory:     queryDir,
	})
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	hasErrors := false
	for _, d := range result.Diagnostics {
		_, _ = fmt.Fprintf(os.Stderr, "[%d] %s:%d: %s\n", d.Severity, d.Filename, d.Line, d.Message)
		if d.Severity == querier_dto.SeverityError {
			hasErrors = true
		}
	}
	if hasErrors {
		return errors.New("generation produced errors")
	}

	return writeOutput(factory, base, result.Files)
}

// writeOutput writes the generated DAL files to the output directory inside a sandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem access.
// Takes base (string) which is the base directory that will contain the db/ output
// folder.
// Takes files ([]querier_dto.GeneratedFile) which holds the generated file names and
// contents.
//
// Returns error when the output directory cannot be created or a file cannot be written.
func writeOutput(factory safedisk.Factory, base string, files []querier_dto.GeneratedFile) error {
	outputDirectory := filepath.Join(base, "db")
	outputSandbox, err := factory.Create("dal-output-parent", filepath.Dir(outputDirectory), safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}
	defer func() { _ = outputSandbox.Close() }()

	if err := outputSandbox.MkdirAll(filepath.Base(outputDirectory), directoryPermissions); err != nil {
		return fmt.Errorf("mkdir %s: %w", outputDirectory, err)
	}

	innerSandbox, err := factory.Create("dal-output", outputDirectory, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}
	defer func() { _ = innerSandbox.Close() }()

	if err := emitter_shared.PruneStaleGeneratedFiles(innerSandbox, emitter_shared.GeneratedFileNameSet(files)); err != nil {
		return fmt.Errorf("prune stale output: %w", err)
	}

	for _, generatedFile := range files {
		if len(generatedFile.Content) == 0 {
			continue
		}
		if err := innerSandbox.WriteFile(generatedFile.Name, generatedFile.Content, filePermissions); err != nil {
			return fmt.Errorf("write %s: %w", generatedFile.Name, err)
		}
		fmt.Printf("wrote %s (%d bytes)\n", filepath.Join(outputDirectory, generatedFile.Name), len(generatedFile.Content))
	}

	return nil
}
