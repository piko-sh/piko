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

// Maps Go compiler errors from generated code back to their original template
// source locations. Parses error messages, resolves virtual file paths, and
// creates diagnostics with accurate line and column information for developers.

import (
	"context"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// goCompileError holds a single error from the Go compiler.
type goCompileError struct {
	// FilePath is the absolute path to the file where the error occurred.
	FilePath string

	// Message holds the error text from the Go compiler.
	Message string

	// Line is the 1-based line number where the error occurred.
	Line int

	// Column is the column number where the error occurred, starting at 1.
	Column int
}

// sourceLocation represents a position and span in the PK source file.
type sourceLocation struct {
	// Expression is the matched text used for highlighting.
	Expression string

	// Line is the 1-based line number in the source file.
	Line int

	// Column is the column number, starting from 1.
	Column int

	// Length is the number of characters to highlight.
	Length int
}

const (
	// goErrorMatchCount is the expected number of matches from goErrorPattern.
	goErrorMatchCount = 5

	// goErrorMatchFilePath is the capture group index for the file path.
	goErrorMatchFilePath = 1

	// goErrorMatchLine is the index for the line number in regex match results.
	goErrorMatchLine = 2

	// goErrorMatchColumn is the index for the column number in a regex match.
	goErrorMatchColumn = 3

	// goErrorMatchMessage is the index for the error message capture group.
	goErrorMatchMessage = 4
)

var (
	// goErrorPattern matches Go compiler error messages in the format
	// /path/to/file.go:line:column: message.
	goErrorPattern = regexp.MustCompile(`^(.+\.go):(\d+):(\d+):\s*(.+)$`)

	// methodCallPattern matches "expr.method" patterns in error messages. Used to
	// find method calls like "editorService.GetFolderAncestors" for highlighting.
	methodCallPattern = regexp.MustCompile(`(\w+\.\w+)`)

	// identifierPattern extracts any identifier from error messages for source
	// searching.
	identifierPattern = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\b`)

	// noiseWords contains words to filter out when extracting search terms.
	// Includes Go keywords, types, and common words from error messages.
	noiseWords = map[string]bool{
		"type": true, "func": true, "var": true, "const": true,
		"if": true, "else": true, "for": true, "range": true,
		"return": true, "break": true, "continue": true,
		"switch": true, "case": true, "default": true, "select": true,
		"chan": true, "map": true, "struct": true, "interface": true,
		"package": true, "import": true, "defer": true, "go": true,
		"nil": true, "true": true, "false": true,
		"int": true, "string": true, "bool": true, "float64": true,
		"error": true, "any": true, "byte": true, "rune": true,
		"undefined": true, "undeclared": true, "cannot": true, "use": true,
		"as": true, "in": true, "to": true, "of": true, "has": true,
		"no": true, "field": true, "or": true, "method": true,
		"variable": true, "value": true, "argument": true, "call": true,
		"not": true, "enough": true, "too": true, "many": true,
		"arguments": true, "name": true, "have": true, "want": true,
		"but": true, "does": true, "the": true, "is": true,
		"number": true, "expected": true, "found": true, "missing": true,
		"declared": true, "unused": true, "redeclared": true, "invalid": true,
		"untyped": true, "constant": true, "expression": true, "statement": true,
	}
)

// parseGoCompileError parses a Go compiler error message into structured data.
// It uses the first line to extract file path, line, and column details, but
// keeps the full message for display.
//
// Takes errMessage (string) which is the raw error message from the Go compiler.
//
// Returns *goCompileError which holds the parsed error details.
// Returns bool which is true when the error was parsed successfully.
func parseGoCompileError(errMessage string) (*goCompileError, bool) {
	firstLine, _, _ := strings.Cut(errMessage, "\n")

	matches := goErrorPattern.FindStringSubmatch(firstLine)
	if len(matches) != goErrorMatchCount {
		return nil, false
	}

	line, err := strconv.Atoi(matches[goErrorMatchLine])
	if err != nil {
		return nil, false
	}

	column, err := strconv.Atoi(matches[goErrorMatchColumn])
	if err != nil {
		return nil, false
	}

	fullMessage := matches[goErrorMatchMessage]
	prefix := matches[goErrorMatchFilePath] + ":" + matches[goErrorMatchLine] + ":" + matches[goErrorMatchColumn] + ": "
	if _, afterPrefix, found := strings.Cut(errMessage, prefix); found {
		if len(afterPrefix) > len(fullMessage) {
			fullMessage = afterPrefix
		}
	}

	return &goCompileError{
		FilePath: matches[goErrorMatchFilePath],
		Line:     line,
		Column:   column,
		Message:  fullMessage,
	}, true
}

// mapGoErrorToDiagnostic converts a Go compiler error into a diagnostic that
// points to the original PK source file. When no mapping exists, it points to
// the Go file instead.
//
// Takes goErr (*goCompileError) which is the parsed Go compiler error.
// Takes virtualModule (*annotator_dto.VirtualModule) which maps generated
// files to source files.
// Takes fsReader (FSReaderPort) which reads files from the file system.
//
// Returns *ast_domain.Diagnostic which is the mapped diagnostic.
// Returns bool which is true when the error was mapped to the source file.
func mapGoErrorToDiagnostic(ctx context.Context, goErr *goCompileError, virtualModule *annotator_dto.VirtualModule, fsReader FSReaderPort) (*ast_domain.Diagnostic, bool) {
	vc := findVirtualComponentForGeneratedFile(goErr.FilePath, virtualModule)
	if vc == nil {
		return createDiagnosticForGoFile(ctx, goErr, virtualModule, fsReader)
	}

	pkLocation := mapGeneratedLineToSource(ctx, goErr, vc, virtualModule, fsReader)

	return &ast_domain.Diagnostic{
		Data:        nil,
		Message:     goErr.Message,
		Expression:  pkLocation.Expression,
		SourcePath:  vc.Source.SourcePath,
		Code:        annotator_dto.CodeGoCompilationError,
		RelatedInfo: nil,
		Location: ast_domain.Location{
			Line:   pkLocation.Line,
			Column: pkLocation.Column,
			Offset: 0,
		},
		SourceLength: pkLocation.Length,
		Severity:     ast_domain.Error,
	}, true
}

// createDiagnosticForGoFile creates a diagnostic for errors in standard Go
// files that are not generated from PK source.
//
// Takes goErr (*goCompileError) which is the parsed Go compiler error.
// Takes vm (*annotator_dto.VirtualModule) which provides source overlay paths.
// Takes fsReader (FSReaderPort) which reads files from the file system.
//
// Returns *ast_domain.Diagnostic which is the created diagnostic.
// Returns bool which is true when the diagnostic was created successfully.
func createDiagnosticForGoFile(ctx context.Context, goErr *goCompileError, vm *annotator_dto.VirtualModule, fsReader FSReaderPort) (*ast_domain.Diagnostic, bool) {
	ctx, l := logger_domain.From(ctx, log)
	sourcePath := goErr.FilePath

	if vm != nil && vm.SourceOverlay != nil {
		for overlayPath := range vm.SourceOverlay {
			if overlayPath == goErr.FilePath || strings.HasSuffix(goErr.FilePath, filepath.Base(overlayPath)) {
				sourcePath = overlayPath
				break
			}
		}
	}

	searchTerms := extractSearchTerms(goErr.Message)
	var expression string
	var length int

	sourceBytes, err := fsReader.ReadFile(ctx, sourcePath)
	if err != nil {
		l.Trace("Failed to read Go source file for error mapping",
			logger_domain.String("source_path", sourcePath),
			logger_domain.Error(err),
		)
	} else {
		source := string(sourceBytes)
		for _, term := range searchTerms {
			location := findTermInSource(source, term)
			if location.Line > 0 && location.Line == goErr.Line {
				expression = location.Expression
				length = location.Length
				break
			}
		}
	}

	return &ast_domain.Diagnostic{
		Data:        nil,
		Message:     goErr.Message,
		Expression:  expression,
		SourcePath:  sourcePath,
		Code:        annotator_dto.CodeGoCompilationError,
		RelatedInfo: nil,
		Location: ast_domain.Location{
			Line:   goErr.Line,
			Column: goErr.Column,
			Offset: 0,
		},
		SourceLength: length,
		Severity:     ast_domain.Error,
	}, true
}

// findVirtualComponentForGeneratedFile finds the VirtualComponent that owns
// the given generated file path.
//
// Takes generatedPath (string) which is the path to the generated Go file.
// Takes vm (*annotator_dto.VirtualModule) which holds the component mappings.
//
// Returns *annotator_dto.VirtualComponent which owns the file, or nil if not
// found.
func findVirtualComponentForGeneratedFile(generatedPath string, vm *annotator_dto.VirtualModule) *annotator_dto.VirtualComponent {
	generatedPath = filepath.Clean(generatedPath)

	for _, vc := range vm.ComponentsByGoPath {
		if filepath.Clean(vc.VirtualGoFilePath) == generatedPath {
			return vc
		}
	}

	generatedDir := filepath.Dir(generatedPath)
	for _, vc := range vm.ComponentsByGoPath {
		vcDir := filepath.Dir(vc.VirtualGoFilePath)
		if filepath.Clean(vcDir) == generatedDir {
			return vc
		}
	}

	return nil
}

// mapGeneratedLineToSource attempts to map a line number from the generated
// Go file back to the PK source file.
//
// Takes ctx (context.Context) which is the context for file operations.
// Takes goErr (*goCompileError) which contains the error location.
// Takes vc (*annotator_dto.VirtualComponent) which provides the component
// context.
// Takes fsReader (FSReaderPort) which provides safe file reading operations.
//
// Returns sourceLocation which contains the best-guess line and column in the
// PK source.
func mapGeneratedLineToSource(ctx context.Context, goErr *goCompileError, vc *annotator_dto.VirtualComponent, _ *annotator_dto.VirtualModule, fsReader FSReaderPort) sourceLocation {
	ctx, l := logger_domain.From(ctx, log)
	searchTerms := extractSearchTerms(goErr.Message)

	if vc.Source.SourcePath != "" {
		sourceBytes, err := fsReader.ReadFile(ctx, vc.Source.SourcePath)
		if err != nil {
			l.Trace("Failed to read PK source file for error mapping",
				logger_domain.String("source_path", vc.Source.SourcePath),
				logger_domain.Error(err),
			)
		} else {
			source := string(sourceBytes)

			for _, term := range searchTerms {
				location := findTermInSource(source, term)
				if location.Line > 0 {
					return location
				}
			}
		}
	}

	return sourceLocation{
		Expression: "",
		Line:       estimateLineFromGeneratedLine(goErr.Line, vc),
		Column:     1,
		Length:     0,
	}
}

// extractSearchTerms pulls searchable names from a Go compiler error message.
// It first looks for method calls (such as expr.method), then pulls out single
// names.
//
// Takes errMessage (string) which contains the Go compiler error message to parse.
//
// Returns []string which contains the pulled terms in priority order.
func extractSearchTerms(errMessage string) []string {
	var terms []string
	seen := make(map[string]bool)

	for _, match := range methodCallPattern.FindAllStringSubmatch(errMessage, -1) {
		if len(match) >= 2 && !seen[match[1]] {
			terms = append(terms, match[1])
			seen[match[1]] = true
		}
	}

	for _, match := range identifierPattern.FindAllStringSubmatch(errMessage, -1) {
		if len(match) >= 2 && !seen[match[1]] && !isNoiseWord(match[1]) {
			terms = append(terms, match[1])
			seen[match[1]] = true
		}
	}

	return terms
}

// isNoiseWord reports whether the word should be filtered out during search
// term extraction.
//
// Takes word (string) which is the word to check.
//
// Returns bool which is true if the word is too short or is a common word
// that does not help with searching.
func isNoiseWord(word string) bool {
	return len(word) < 2 || noiseWords[word]
}

// findTermInSource searches for a term in source code and returns its location.
// It prefers matches on code lines over matches in comments.
//
// Takes source (string) which is the source code to search.
// Takes term (string) which is the text to find.
//
// Returns sourceLocation with Line, Column, Length, and Expression set.
// Returns sourceLocation with Line set to 0 if the term is not found.
func findTermInSource(source, term string) sourceLocation {
	lines := strings.Split(source, "\n")

	for i, line := range lines {
		if isCommentLine(line) {
			continue
		}

		index := strings.Index(line, term)
		if index >= 0 {
			return sourceLocation{
				Line:       i + 1,
				Column:     index + 1,
				Length:     len(term),
				Expression: term,
			}
		}
	}

	for i, line := range lines {
		index := strings.Index(line, term)
		if index >= 0 {
			return sourceLocation{
				Line:       i + 1,
				Column:     index + 1,
				Length:     len(term),
				Expression: term,
			}
		}
	}

	return sourceLocation{}
}

// isCommentLine checks whether a line is a comment.
//
// Takes line (string) which is the source line to check.
//
// Returns bool which is true if the line starts with // or /* after
// removing leading whitespace.
func isCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*")
}

// estimateLineFromGeneratedLine gives a rough estimate of the source line
// number based on a line number from the generated file.
//
// Takes generatedLine (int) which is the line number in the generated file.
// Takes vc (*annotator_dto.VirtualComponent) which provides component context.
//
// Returns int which is the estimated line number in the source file.
func estimateLineFromGeneratedLine(generatedLine int, vc *annotator_dto.VirtualComponent) int {
	if vc.Source.Script != nil && vc.Source.Script.ScriptStartLocation.Line > 0 {
		offset := max(generatedLine-10, 1)
		return vc.Source.Script.ScriptStartLocation.Line + offset
	}

	return generatedLine
}

// convertTypeInspectorErrorToDiagnostics parses build errors from TypeInspector
// and converts them into diagnostics that point to PK source files.
//
// Takes err (error) which is the error from TypeInspector.Build.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides mappings
// between generated code and source files.
// Takes fsReader (FSReaderPort) which provides file reading.
//
// Returns []*ast_domain.Diagnostic which contains the converted diagnostics,
// or nil when err or virtualModule is nil.
func convertTypeInspectorErrorToDiagnostics(ctx context.Context, err error, virtualModule *annotator_dto.VirtualModule, fsReader FSReaderPort) []*ast_domain.Diagnostic {
	if err == nil || virtualModule == nil {
		return nil
	}

	errMessage := err.Error()
	var diagnostics []*ast_domain.Diagnostic

	if index := strings.Index(errMessage, "errors found during package loading: "); index >= 0 {
		errMessage = errMessage[index+len("errors found during package loading: "):]
	}

	if index := strings.Index(errMessage, "failed to load packages from source: "); index >= 0 {
		errMessage = errMessage[index+len("failed to load packages from source: "):]
	}

	for part := range strings.SplitSeq(errMessage, "; ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		goErr, ok := parseGoCompileError(part)
		if !ok {
			continue
		}

		diagnostic, ok := mapGoErrorToDiagnostic(ctx, goErr, virtualModule, fsReader)
		if ok {
			diagnostics = append(diagnostics, diagnostic)
		}
	}

	return diagnostics
}
