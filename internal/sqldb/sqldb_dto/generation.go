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

package sqldb_dto

import (
	"io/fs"
)

// Severity classifies a generation diagnostic.
type Severity uint8

const (
	// SeverityError indicates a problem that prevents code generation.
	SeverityError Severity = iota

	// SeverityWarning indicates a likely problem that does not block generation.
	SeverityWarning

	// SeverityHint indicates a suggestion for improvement.
	SeverityHint
)

// GeneratedDriver selects the database driver API the generated Go code targets.
type GeneratedDriver uint8

const (
	// DriverDatabaseSQL generates code against database/sql. It is the default.
	DriverDatabaseSQL GeneratedDriver = iota

	// DriverPgx generates code against github.com/jackc/pgx/v5.
	DriverPgx

	// DriverClickHouseNative generates code against the native ClickHouse driver.
	DriverClickHouseNative
)

// SourceError is one diagnostic produced while analysing migrations and queries.
type SourceError struct {
	// Filename is the path of the SQL file, relative to the source filesystem.
	Filename string

	// Message describes the problem.
	Message string

	// Code is the stable diagnostic code, such as Q012.
	Code string

	// Suggestion is an optional fix for the problem.
	Suggestion string

	// Line is the 1-based line where the problem starts.
	Line int

	// Column is the 1-based column where the problem starts.
	Column int

	// EndLine is the 1-based line where the problem ends.
	EndLine int

	// EndColumn is the 1-based column where the problem ends.
	EndColumn int

	// Severity classifies the diagnostic.
	Severity Severity
}

// GeneratedFile is one Go source file produced by code generation.
type GeneratedFile struct {
	// Name is the file name, without a directory.
	Name string

	// Content is the formatted Go source.
	Content []byte
}

// GenerationResult is the outcome of generating code for one database.
type GenerationResult struct {
	// Files holds the generated Go files.
	Files []GeneratedFile

	// Diagnostics holds every error, warning and hint found during analysis.
	Diagnostics []SourceError

	// QueryCount is the number of query methods the generated code exposes.
	QueryCount int
}

// TypeOverride maps a SQL type to a Go type in generated code.
type TypeOverride struct {
	// SQLTypeName is the SQL type to override, such as "numeric" or "public.mood".
	SQLTypeName string

	// GoPackage is the import path of the Go type, empty for built-in types.
	GoPackage string

	// GoName is the Go type name, such as "Decimal".
	GoName string
}

// CustomFunction declares a SQL function the analyser cannot discover from migrations,
// such as one provided by a database extension.
type CustomFunction struct {
	// Name is the SQL function name.
	Name string

	// ReturnType is the function's SQL result type.
	ReturnType string

	// Nullable selects null handling with "null_on_null" by default, "never_null", or
	// "called_on_null".
	Nullable string

	// DataAccess selects reader or writer routing with "read_only" by default or
	// "modifies_data".
	DataAccess string

	// Arguments lists the SQL types of the function's parameters.
	Arguments []string

	// MinArguments is the minimum number of arguments for functions with optional trailing
	// parameters.
	MinArguments int

	// IsAggregate reports whether the function aggregates rows.
	IsAggregate bool

	// IsVariadic reports whether the last argument repeats.
	IsVariadic bool
}

// GenerateRequest describes one code generation run.
type GenerateRequest struct {
	// Files is the filesystem holding the migration and query directories.
	Files fs.FS

	// PackageName is the Go package name of the generated code.
	PackageName string

	// MigrationDirectory is the directory of migration files within Files.
	MigrationDirectory string

	// QueryDirectory is the directory of query files within Files.
	QueryDirectory string

	// TypeOverrides replaces default SQL to Go type mappings.
	TypeOverrides []TypeOverride

	// CustomFunctions declares SQL functions the analyser cannot discover.
	CustomFunctions []CustomFunction

	// Driver selects the database driver API the generated code targets.
	Driver GeneratedDriver
}
