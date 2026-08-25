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

package compiler_adapters

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"cmp"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/cssinliner"
	"piko.sh/piko/internal/logger/logger_domain"
)

// cssPreProcessor adapts the shared cssinliner.Processor to the compiler's
// CSSPreProcessorPort interface, resolving @import statements in CSS before the CSS is
// embedded into compiled component output.
type cssPreProcessor struct {
	// processor provides CSS @import inlining.
	processor *cssinliner.Processor

	// fsReader reads imported CSS files from the filesystem.
	fsReader cssinliner.FSReaderPort

	// moduleName is the Go module name (for example a GitHub-hosted module path) used to
	// convert module-qualified source IDs to filesystem paths.
	moduleName string

	// baseDir is the absolute path to the project root (the directory containing go.mod),
	// used alongside moduleName for path conversion.
	baseDir string
}

var (
	_ compiler_domain.CSSPreProcessorPort = (*cssPreProcessor)(nil)
)

// NewCSSPreProcessor creates a new adapter that wraps a CSS processor to implement
// compiler_domain.CSSPreProcessorPort.
//
// Takes processor (*cssinliner.Processor) which provides CSS @import inlining.
// Takes fsReader (cssinliner.FSReaderPort) which reads imported CSS files from the
// filesystem.
// Takes moduleName (string) which is the Go module name for converting module-qualified
// paths to filesystem paths.
// Takes baseDir (string) which is the absolute path to the project root.
//
// Returns compiler_domain.CSSPreProcessorPort which resolves CSS @import statements
// before compilation.
func NewCSSPreProcessor(
	processor *cssinliner.Processor,
	fsReader cssinliner.FSReaderPort,
	moduleName string,
	baseDir string,
) compiler_domain.CSSPreProcessorPort {
	return &cssPreProcessor{
		processor:  processor,
		fsReader:   fsReader,
		moduleName: moduleName,
		baseDir:    baseDir,
	}
}

// InlineImports resolves @import statements in the given CSS content by reading the
// imported files and merging them into a single CSS string.
//
// The sourcePath may be a module-qualified path (for example a GitHub-hosted module path
// with a "/components/foo.pkc" suffix) which is converted to a filesystem path before CSS
// resolution.
//
// Takes cssContent (string) which is the raw CSS with potential @import rules.
// Takes sourcePath (string) which identifies the source file for resolving relative
// imports.
// Takes startLocation (ast_domain.Location) which is where the style content begins in
// the source file, so a failure is reported at the line the author wrote rather than at
// the top of the file.
//
// Returns string which is the CSS with all imports inlined.
// Returns error when import resolution or file reading fails.
func (p *cssPreProcessor) InlineImports(
	ctx context.Context,
	cssContent string,
	sourcePath string,
	startLocation ast_domain.Location,
) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "CSSPreProcessor.InlineImports",
		logger_domain.String("sourcePath", sourcePath),
		logger_domain.Int("cssLength", len(cssContent)),
	)
	defer span.End()

	fsPath := p.resolveToFilesystemPath(sourcePath)

	result, diagnostics, err := p.processor.Process(
		ctx,
		cssContent,
		fsPath,
		startLocation,
		p.fsReader,
	)
	if err != nil {
		return "", fmt.Errorf("CSS import inlining failed for %s: %w", sourcePath, err)
	}

	if ast_domain.HasErrors(diagnostics) {
		return "", newCSSImportError(sourcePath, p.processor.ImportDiagnosticCode(), diagnostics)
	}

	return result, nil
}

// resolveToFilesystemPath converts a module-qualified source path to an absolute
// filesystem path, returning non-module paths unchanged.
//
// Takes sourcePath (string) which is the path to resolve, potentially prefixed with the
// Go module name.
//
// Returns string which is the resolved filesystem path.
func (p *cssPreProcessor) resolveToFilesystemPath(sourcePath string) string {
	if p.moduleName == "" || p.baseDir == "" {
		return sourcePath
	}
	if relativePath, found := strings.CutPrefix(sourcePath, p.moduleName+"/"); found {
		return filepath.Join(p.baseDir, filepath.FromSlash(relativePath))
	}
	return sourcePath
}

// cssImportError reports every error-severity diagnostic raised while inlining a
// component's CSS imports.
type cssImportError struct {
	// SourcePath is the component whose styles failed.
	SourcePath string

	// ImportCode is the diagnostic code the inliner assigns to @import failures, which tells
	// them apart from parser diagnostics.
	ImportCode string

	// Diagnostics holds every error-severity diagnostic from the inlining pass.
	Diagnostics []*ast_domain.Diagnostic
}

// newCSSImportError collects the error-severity diagnostics from an inlining pass.
//
// Takes sourcePath (string) which identifies the component whose styles failed.
// Takes importCode (string) which marks a diagnostic as an @import failure.
// Takes diagnostics ([]*ast_domain.Diagnostic) which holds the diagnostics from inlining.
//
// Returns error which describes every error-severity diagnostic found.
func newCSSImportError(sourcePath, importCode string, diagnostics []*ast_domain.Diagnostic) error {
	failures := make([]*ast_domain.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic != nil && diagnostic.Severity == ast_domain.Error {
			failures = append(failures, diagnostic)
		}
	}

	return &cssImportError{SourcePath: sourcePath, ImportCode: importCode, Diagnostics: failures}
}

// Error renders one line per failing diagnostic, each naming the file and position it
// came from so the author can go straight to it.
//
// Returns string which is the summary of every failure.
func (e *cssImportError) Error() string {
	if len(e.Diagnostics) == 0 {
		return fmt.Sprintf("%s: CSS import inlining failed", e.SourcePath)
	}

	lines := make([]string, 0, len(e.Diagnostics))
	for _, diagnostic := range e.Diagnostics {
		lines = append(lines, describeCSSDiagnostic(e.SourcePath, e.ImportCode, diagnostic))
	}
	return strings.Join(lines, "; ")
}

// describeCSSDiagnostic formats one diagnostic as position, cause and message.
//
// Takes sourcePath (string) which identifies the component whose styles failed.
// Takes importCode (string) which marks a diagnostic as an @import failure.
// Takes diagnostic (*ast_domain.Diagnostic) which is the diagnostic to describe.
//
// Returns string which is the one-line description.
func describeCSSDiagnostic(sourcePath, importCode string, diagnostic *ast_domain.Diagnostic) string {
	file := cmp.Or(diagnostic.SourcePath, sourcePath)

	position := file
	if diagnostic.Location.Line > 0 {
		position = fmt.Sprintf("%s:%d:%d", file, diagnostic.Location.Line, diagnostic.Location.Column)
	}

	if diagnostic.Code == importCode && diagnostic.Expression != "" {
		return fmt.Sprintf("%s: cannot resolve @import %q: %s", position, diagnostic.Expression, diagnostic.Message)
	}
	return fmt.Sprintf("%s: %s", position, diagnostic.Message)
}
