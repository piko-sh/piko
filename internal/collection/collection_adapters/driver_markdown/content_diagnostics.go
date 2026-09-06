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

package driver_markdown

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxReportedErrorsPerFile caps how many error diagnostics of one markdown file are
	// included in the error returned for it.
	maxReportedErrorsPerFile = 10

	// maxReportedWarningsPerFile caps how many non-error diagnostics of one markdown file
	// are included in its warning log entry.
	maxReportedWarningsPerFile = 10

	// maxReportedFilesWithErrors caps how many files' errors are joined into the error
	// returned for a collection.
	maxReportedFilesWithErrors = 20
)

var (
	// errMarkdownContent marks a markdown file whose processing produced error-severity
	// diagnostics, such as a malformed piko shortcode.
	errMarkdownContent = errors.New("markdown content has errors")
)

// contentDiagnosticsError turns the error-severity diagnostics of a processed markdown
// file into an error and logs any other diagnostics as a warning.
//
// Takes relativePath (string) which identifies the markdown file in messages.
// Takes diagnostics ([]*ast_domain.Diagnostic) which are the diagnostics produced while
// processing the file.
//
// Returns error which wraps errMarkdownContent and lists the error diagnostics, or nil
// when there are none.
func contentDiagnosticsError(ctx context.Context, relativePath string, diagnostics []*ast_domain.Diagnostic) error {
	errorDiagnostics, otherDiagnostics := partitionDiagnostics(diagnostics)
	logContentWarnings(ctx, relativePath, otherDiagnostics)
	if len(errorDiagnostics) == 0 {
		return nil
	}
	return fmt.Errorf("%w in %s: %w", errMarkdownContent, relativePath, joinBounded(errorDiagnostics, maxReportedErrorsPerFile))
}

// partitionDiagnostics splits diagnostics into error-severity ones, returned as errors,
// and the rest. Nil entries are skipped.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which are the diagnostics to split.
//
// Returns []error which holds the error-severity diagnostics.
// Returns []*ast_domain.Diagnostic which holds every other diagnostic.
func partitionDiagnostics(diagnostics []*ast_domain.Diagnostic) ([]error, []*ast_domain.Diagnostic) {
	var errorDiagnostics []error
	var otherDiagnostics []*ast_domain.Diagnostic
	for _, diagnostic := range diagnostics {
		if diagnostic == nil {
			continue
		}
		if diagnostic.Severity == ast_domain.Error {
			errorDiagnostics = append(errorDiagnostics, diagnostic)
			continue
		}
		otherDiagnostics = append(otherDiagnostics, diagnostic)
	}
	return errorDiagnostics, otherDiagnostics
}

// logContentWarnings logs the non-error diagnostics of a markdown file as one warning.
//
// Takes relativePath (string) which identifies the markdown file.
// Takes diagnostics ([]*ast_domain.Diagnostic) which are the diagnostics to report.
func logContentWarnings(ctx context.Context, relativePath string, diagnostics []*ast_domain.Diagnostic) {
	if len(diagnostics) == 0 {
		return
	}
	messages := make([]string, 0, min(len(diagnostics), maxReportedWarningsPerFile))
	for _, diagnostic := range diagnostics[:min(len(diagnostics), maxReportedWarningsPerFile)] {
		messages = append(messages, diagnostic.Error())
	}
	_, l := logger_domain.From(ctx, log)
	l.Warn("Markdown file has warnings",
		logger_domain.String(keyPath, relativePath),
		logger_domain.Int("warning_count", len(diagnostics)),
		logger_domain.Strings("warnings", messages))
}

// joinBounded joins at most limit errors, adding a note of how many were left out.
//
// Takes errs ([]error) which are the errors to join.
// Takes limit (int) which is the most errors to include.
//
// Returns error which joins the included errors, or nil when errs is empty.
func joinBounded(errs []error, limit int) error {
	if len(errs) <= limit {
		return errors.Join(errs...)
	}
	reported := make([]error, 0, limit+1)
	reported = append(reported, errs[:limit]...)
	reported = append(reported, fmt.Errorf("%d more omitted", len(errs)-limit))
	return errors.Join(reported...)
}
