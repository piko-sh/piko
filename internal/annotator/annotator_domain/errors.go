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

package annotator_domain

// Defines custom error types for compilation failures including parsing errors,
// semantic validation issues, and circular dependencies. Provides diagnostic
// formatting utilities to present compilation errors with source context for
// developers.

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// IsParseSoftError reports whether err originated from a tolerable
// parse failure during .pk file processing. Script-block syntax
// errors and template-diagnostic errors are considered soft so
// discovery (which only cares about imports) can continue past them;
// every other error is treated as fatal.
//
// Takes err (error) which is the error to classify.
//
// Returns true when err is (or wraps) a tolerable parse failure.
func IsParseSoftError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := errors.AsType[*scriptBlockParseError](err); ok {
		return true
	}
	var diagErr *ParseDiagnosticError
	return errors.As(err, &diagErr)
}

// ParseDiagnosticError represents an error that occurred during parsing of a
// template.
type ParseDiagnosticError struct {
	// SourcePath is the path to the file where the parsing error occurred.
	SourcePath string

	// TemplateSource is the template text that failed to parse.
	TemplateSource string

	// Diagnostics holds the list of parsing errors found in the source file.
	Diagnostics []*ast_domain.Diagnostic
}

// NewParseDiagnosticError creates a new ParseDiagnosticError with the given
// diagnostics and source details.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains the parsing errors.
// Takes sourcePath (string) which is the path to the source file.
// Takes templateSource (string) which is the original template content.
//
// Returns *ParseDiagnosticError which wraps the diagnostics with source
// context.
func NewParseDiagnosticError(diagnostics []*ast_domain.Diagnostic, sourcePath, templateSource string) *ParseDiagnosticError {
	return &ParseDiagnosticError{
		Diagnostics:    diagnostics,
		SourcePath:     sourcePath,
		TemplateSource: templateSource,
	}
}

// Error implements the error interface.
//
// Returns string which contains the number of parsing errors and the source
// file path.
func (e *ParseDiagnosticError) Error() string {
	if len(e.Diagnostics) == 1 {
		return fmt.Sprintf("found 1 parsing error in %s", e.SourcePath)
	}
	return fmt.Sprintf("found %d parsing errors in %s", len(e.Diagnostics), e.SourcePath)
}

// SemanticError represents an error found during semantic analysis.
// It implements the error interface.
type SemanticError struct {
	// Diagnostics holds the errors and warnings found during semantic analysis.
	Diagnostics []*ast_domain.Diagnostic
}

// NewSemanticError creates a new SemanticError with the given diagnostics.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains the diagnostic
// messages describing the semantic issues found.
//
// Returns *SemanticError which wraps the diagnostics for error reporting.
func NewSemanticError(diagnostics []*ast_domain.Diagnostic) *SemanticError {
	return &SemanticError{
		Diagnostics: diagnostics,
	}
}

// Error implements the error interface.
//
// Returns string which contains the count of semantic validation errors and
// warnings found during analysis.
func (e *SemanticError) Error() string {
	_, _, warningCount, errorCount := getDiagnosticCounts(e.Diagnostics)

	if len(e.Diagnostics) == 1 {
		return "found 1 semantic validation error"
	}
	return fmt.Sprintf("found %d semantic validation errors and %d semantic validation warnings", errorCount, warningCount)
}

// CircularDependencyError represents a loop in the component graph where
// packages depend on each other in a cycle. It implements the error interface.
type CircularDependencyError struct {
	// Path lists the package names that form the dependency cycle.
	Path []string
}

// NewCircularDependencyError creates a new CircularDependencyError with the
// given dependency path.
//
// Takes path ([]string) which specifies the chain of dependencies that form
// the cycle.
//
// Returns *CircularDependencyError which represents the circular dependency.
func NewCircularDependencyError(path []string) *CircularDependencyError {
	return &CircularDependencyError{Path: path}
}

// Error implements the error interface.
//
// Returns string which describes the circular dependency path.
func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.Path, " -> "))
}

// FormatAllDiagnostics formats all diagnostics grouped by source file into a
// string that is easy to read.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which provides the list of
// diagnostics to format.
// Takes sourceContents (map[string][]byte) which maps file paths to their
// source code for context display.
//
// Returns string which contains the formatted output, or an empty string if
// there are no diagnostics.
func FormatAllDiagnostics(diagnostics []*ast_domain.Diagnostic, sourceContents map[string][]byte) string {
	if len(diagnostics) == 0 {
		return ""
	}

	var builder strings.Builder

	diagsByFile := make(map[string][]*ast_domain.Diagnostic)
	for _, d := range diagnostics {
		if d.SourcePath == "" {
			builder.WriteString(d.Error() + "\n(Source path was missing for this diagnostic)\n\n")
			continue
		}
		diagsByFile[d.SourcePath] = append(diagsByFile[d.SourcePath], d)
	}

	sortedPaths := make([]string, 0, len(diagsByFile))
	for path := range diagsByFile {
		sortedPaths = append(sortedPaths, path)
	}
	slices.Sort(sortedPaths)

	for _, path := range sortedPaths {
		diagsForFile := diagsByFile[path]
		source, ok := sourceContents[path]

		if !ok {
			_, _ = fmt.Fprintf(&builder, "--> Found %d issue(s) in %s (source code unavailable for formatting):\n", len(diagsForFile), path)
			for _, d := range diagsForFile {
				builder.WriteString(" ● " + d.Error() + "\n")
			}
			builder.WriteString("\n")
			continue
		}

		builder.WriteString(ast_domain.FormatDiagnostics(path, string(source), diagsForFile))
	}

	return builder.String()
}

// getDiagnosticCounts counts diagnostics by severity level.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains the diagnostics to
// count.
//
// Returns debugCount (int) which is the number of debug-level diagnostics.
// Returns infoCount (int) which is the number of info-level diagnostics.
// Returns warningCount (int) which is the number of warning-level diagnostics.
// Returns errorCount (int) which is the number of error-level diagnostics.
func getDiagnosticCounts(diagnostics []*ast_domain.Diagnostic) (debugCount int, infoCount int, warningCount int, errorCount int) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == ast_domain.Debug {
			debugCount++
		}
		if diagnostic.Severity == ast_domain.Info {
			infoCount++
		}
		if diagnostic.Severity == ast_domain.Warning {
			warningCount++
		}
		if diagnostic.Severity == ast_domain.Error {
			errorCount++
		}
	}

	return debugCount, infoCount, warningCount, errorCount
}
