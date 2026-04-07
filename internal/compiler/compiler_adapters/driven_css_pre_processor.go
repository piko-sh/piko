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

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/cssinliner"
	"piko.sh/piko/internal/logger/logger_domain"
)

// cssPreProcessor adapts the shared cssinliner.Processor to the
// compiler's CSSPreProcessorPort interface, resolving @import statements
// in CSS before the CSS is embedded into compiled component output.
type cssPreProcessor struct {
	// processor provides CSS @import inlining.
	processor *cssinliner.Processor

	// fsReader reads imported CSS files from the filesystem.
	fsReader cssinliner.FSReaderPort

	// moduleName is the Go module name (e.g. "github.com/org/repo") used to
	// convert module-qualified source IDs to filesystem paths.
	moduleName string

	// baseDir is the absolute path to the project root (the directory
	// containing go.mod), used alongside moduleName for path conversion.
	baseDir string
}

var _ compiler_domain.CSSPreProcessorPort = (*cssPreProcessor)(nil)

// NewCSSPreProcessor creates a new adapter that wraps a CSS processor to
// implement compiler_domain.CSSPreProcessorPort.
//
// Takes processor (*cssinliner.Processor) which provides CSS @import
// inlining.
// Takes fsReader (cssinliner.FSReaderPort) which reads imported CSS
// files from the filesystem.
// Takes moduleName (string) which is the Go module name for converting
// module-qualified paths to filesystem paths.
// Takes baseDir (string) which is the absolute path to the project root.
//
// Returns compiler_domain.CSSPreProcessorPort which resolves CSS @import
// statements before compilation.
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

// InlineImports resolves @import statements in the given CSS content by
// reading the imported files and merging them into a single CSS string.
//
// The sourcePath may be a module-qualified path (e.g.
// "github.com/org/repo/components/foo.pkc") which is converted to a
// filesystem path before CSS resolution.
//
// Takes cssContent (string) which is the raw CSS with potential @import rules.
// Takes sourcePath (string) which identifies the source file for resolving
// relative imports.
//
// Returns string which is the CSS with all imports inlined.
// Returns error when import resolution or file reading fails.
func (p *cssPreProcessor) InlineImports(ctx context.Context, cssContent string, sourcePath string) (string, error) {
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
		ast_domain.Location{Line: 1, Column: 1},
		p.fsReader,
	)
	if err != nil {
		return "", fmt.Errorf("CSS import inlining failed for %s: %w", sourcePath, err)
	}

	if ast_domain.HasErrors(diagnostics) {
		return "", fmt.Errorf("CSS import inlining produced errors for %s: %s", sourcePath, diagnostics[0].Message)
	}

	return result, nil
}

// resolveToFilesystemPath converts a module-qualified source path to an
// absolute filesystem path. If the path is not module-qualified (e.g.
// already a filesystem path), it is returned unchanged.
//
// Takes sourcePath (string) which is the path to resolve, potentially
// prefixed with the Go module name.
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
