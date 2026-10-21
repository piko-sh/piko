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

	"piko.sh/piko/internal/sqldb/sqldb_adapters/generated_writer"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_postgres"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
	"piko.sh/safedisk"
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

// selectEngine returns the database engine for the requested dialect.
//
// Takes dialect (string) which names the target dialect, either "sqlite" (the default for
// an empty value) or "postgres".
//
// Returns db.EngineConfig which supplies the toolchain for the dialect.
// Returns error when the dialect is not recognised.
func selectEngine(dialect string) (db.EngineConfig, error) {
	switch dialect {
	case "", "sqlite":
		return db_engine_sqlite.SQLite(), nil
	case "postgres":
		return db_engine_postgres.Postgres(), nil
	default:
		return db.EngineConfig{}, fmt.Errorf("unsupported dialect %q (want sqlite or postgres)", dialect)
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
	packageName := os.Args[2]
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

	result, err := db.Generate(context.Background(), engine, db.GenerateRequest{
		Files:              os.DirFS(base),
		PackageName:        packageName,
		MigrationDirectory: "migrations",
		QueryDirectory:     "queries",
		TypeOverrides:      nil,
		CustomFunctions:    nil,
		Driver:             db.DriverDatabaseSQL,
	})
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	hasErrors := false
	for _, diagnostic := range result.Diagnostics {
		_, _ = fmt.Fprintf(os.Stderr, "[%d] %s:%d: %s\n", diagnostic.Severity, diagnostic.Filename, diagnostic.Line, diagnostic.Message)
		if diagnostic.Severity == db.SeverityError {
			hasErrors = true
		}
	}
	if hasErrors {
		return errors.New("generation produced errors")
	}

	return writeOutput(factory, base, result.Files)
}

// writeOutput writes the generated DAL files to the output directory inside a sandbox,
// pruning files an earlier run generated but this run no longer produces.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem access.
// Takes base (string) which is the base directory that will contain the db/ output
// folder.
// Takes files ([]db.GeneratedFile) which holds the generated file names and contents.
//
// Returns error when the output directory cannot be created or a file cannot be written.
func writeOutput(factory safedisk.Factory, base string, files []db.GeneratedFile) error {
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

	if err := generated_writer.WriteFiles(innerSandbox, files, filePermissions); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	for _, generatedFile := range files {
		if len(generatedFile.Content) == 0 {
			continue
		}
		fmt.Printf("wrote %s (%d bytes)\n", filepath.Join(outputDirectory, generatedFile.Name), len(generatedFile.Content))
	}

	return nil
}
