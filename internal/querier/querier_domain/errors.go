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
	"errors"
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	// ErrMissingEnginePort is returned when a querier service is created
	// without an engine adapter.
	ErrMissingEnginePort = errors.New("querier service requires an engine port")

	// ErrMissingEmitterPort is returned when a querier service is created
	// without a code emitter adapter.
	ErrMissingEmitterPort = errors.New("querier service requires a code emitter port")

	// ErrMissingFileReaderPort is returned when a querier service is created
	// without a file reader adapter.
	ErrMissingFileReaderPort = errors.New("querier service requires a file reader port")
)

// CatalogueError represents a failure during migration replay.
type CatalogueError struct {
	// Cause holds the underlying error that triggered this catalogue error.
	Cause error

	// Filename holds the path of the migration file where the error occurred.
	Filename string

	// Message holds a human-readable description of what went wrong.
	Message string

	// Line holds the one-based line number in the migration file.
	Line int

	// MigrationIndex holds the zero-based index of the migration in the replay sequence.
	MigrationIndex int
}

// NewCatalogueError creates a new catalogue error with source location.
//
// Takes filename (string) which specifies the migration file path.
// Takes line (int) which specifies the one-based line number of the error.
// Takes migrationIndex (int) which specifies the zero-based position in the replay sequence.
// Takes message (string) which specifies the human-readable error description.
// Takes cause (error) which specifies the underlying error that triggered this failure.
//
// Returns *CatalogueError which holds the fully populated error with source location.
func NewCatalogueError(filename string, line int, migrationIndex int, message string, cause error) *CatalogueError {
	return &CatalogueError{
		Filename:       filename,
		Line:           line,
		MigrationIndex: migrationIndex,
		Message:        message,
		Cause:          cause,
	}
}

// Error returns a human-readable error message with source location.
//
// Returns string which holds the formatted error including filename and line number.
func (e *CatalogueError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("migration %s:%d: %s", e.Filename, e.Line, e.Message)
	}
	return fmt.Sprintf("migration %s: %s", e.Filename, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/errors.As.
//
// Returns error which holds the wrapped cause, or nil if no cause was set.
func (e *CatalogueError) Unwrap() error {
	return e.Cause
}

// QueryAnalysisError wraps one or more SourceError diagnostics from query
// analysis.
type QueryAnalysisError struct {
	// Filename holds the path of the query file that produced the diagnostics.
	Filename string

	// Diagnostics holds the list of source errors found during analysis.
	Diagnostics []querier_dto.SourceError
}

// NewQueryAnalysisError creates a new query analysis error wrapping
// diagnostics.
//
// Takes filename (string) which specifies the query file path.
// Takes diagnostics ([]querier_dto.SourceError) which specifies the analysis errors found.
//
// Returns *QueryAnalysisError which holds the error wrapping all diagnostics.
func NewQueryAnalysisError(filename string, diagnostics []querier_dto.SourceError) *QueryAnalysisError {
	return &QueryAnalysisError{
		Filename:    filename,
		Diagnostics: diagnostics,
	}
}

// Error returns a summary of analysis errors found in the query file.
//
// Returns string which holds the formatted count and filename.
func (e *QueryAnalysisError) Error() string {
	return fmt.Sprintf("found %d analysis errors in %s", len(e.Diagnostics), e.Filename)
}

// DirectiveSyntaxError represents a malformed piko. directive in a SQL
// query file.
type DirectiveSyntaxError struct {
	// Filename holds the path of the query file containing the malformed directive.
	Filename string

	// Message holds a human-readable description of the syntax error.
	Message string

	// Code holds the diagnostic error code, e.g. "Q007".
	Code string

	// Line holds the one-based line number of the directive.
	Line int

	// Column holds the one-based column number within the line.
	Column int
}

// NewDirectiveSyntaxError creates a new directive syntax error with source
// location.
//
// Takes filename (string) which specifies the query file path.
// Takes line (int) which specifies the one-based line number.
// Takes column (int) which specifies the one-based column number.
// Takes message (string) which specifies the human-readable error description.
//
// Returns *DirectiveSyntaxError which holds the fully populated syntax error.
func NewDirectiveSyntaxError(filename string, line int, column int, message string) *DirectiveSyntaxError {
	return &DirectiveSyntaxError{
		Filename: filename,
		Line:     line,
		Column:   column,
		Message:  message,
		Code:     querier_dto.CodeDirectiveSyntax,
	}
}

// Error returns a formatted error message with source location and error code.
//
// Returns string which holds the error formatted as "file:line:col: code message".
func (e *DirectiveSyntaxError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s %s", e.Filename, e.Line, e.Column, e.Code, e.Message)
}
