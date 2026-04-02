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

package lsp_domain

import (
	"context"
	"fmt"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// convertDiagnosticsToLSP converts AST diagnostics to LSP protocol format.
// It filters to include only diagnostics for the given file path and removes
// duplicates based on message, position, and severity.
//
// Takes allDiagnostics ([]*ast_domain.Diagnostic) which contains the source
// diagnostics to convert.
// Takes absPath (string) which is the absolute file path to filter by.
//
// Returns []protocol.Diagnostic which contains the filtered LSP diagnostics
// for the given file.
func convertDiagnosticsToLSP(ctx context.Context, allDiagnostics []*ast_domain.Diagnostic, absPath string) []protocol.Diagnostic {
	lspDiagnostics := make([]protocol.Diagnostic, 0, len(allDiagnostics))
	seen := make(map[string]struct{}, len(allDiagnostics))

	for _, d := range allDiagnostics {
		if isDuplicateDiagnostic(d, seen) {
			continue
		}

		if !shouldIncludeDiagnostic(ctx, d, absPath) {
			continue
		}

		lspDiagnostics = append(lspDiagnostics, buildLSPDiagnostic(d))
	}

	return lspDiagnostics
}

// isDuplicateDiagnostic checks if a diagnostic has already been seen and marks
// it as seen.
//
// Takes d (*ast_domain.Diagnostic) which is the diagnostic to check.
// Takes seen (map[string]struct{}) which tracks diagnostics already seen.
//
// Returns bool which is true if the diagnostic was already seen.
func isDuplicateDiagnostic(d *ast_domain.Diagnostic, seen map[string]struct{}) bool {
	key := fmt.Sprintf("%s:%d:%d:%d", d.Message, d.Location.Line, d.Location.Column, d.Severity)
	if _, exists := seen[key]; exists {
		return true
	}
	seen[key] = struct{}{}
	return false
}

// shouldIncludeDiagnostic checks whether a diagnostic should be included based
// on its file path.
//
// Takes d (*ast_domain.Diagnostic) which is the diagnostic to check.
// Takes absPath (string) which is the absolute path to match against.
//
// Returns bool which is true when the diagnostic's source path matches absPath.
func shouldIncludeDiagnostic(ctx context.Context, d *ast_domain.Diagnostic, absPath string) bool {
	_, l := logger_domain.From(ctx, log)

	if d.SourcePath != absPath {
		l.Info("Diagnostic filtered out (path mismatch)",
			logger_domain.String("diagnosticPath", d.SourcePath),
			logger_domain.String("targetPath", absPath),
			logger_domain.String("message", d.Message))
		return false
	}

	l.Info("Diagnostic INCLUDED",
		logger_domain.String("diagnosticPath", d.SourcePath),
		logger_domain.Bool("emptyPath", d.SourcePath == ""),
		logger_domain.String("message", d.Message))
	return true
}

// buildLSPDiagnostic converts an AST diagnostic to LSP protocol format.
//
// Takes d (*ast_domain.Diagnostic) which is the diagnostic to convert.
//
// Returns protocol.Diagnostic which is the converted LSP diagnostic.
func buildLSPDiagnostic(d *ast_domain.Diagnostic) protocol.Diagnostic {
	endChar := calculateEndChar(d)

	var code any
	if d.Code != "" {
		code = d.Code
	}

	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(d.Location.Line - 1),
				Character: safeconv.IntToUint32(d.Location.Column - 1),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(d.Location.Line - 1),
				Character: safeconv.IntToUint32(endChar),
			},
		},
		Severity:           convertSeverityToLSP(d.Severity),
		Code:               code,
		Source:             "piko",
		Message:            d.Message,
		RelatedInformation: convertRelatedInfoToLSP(d),
		Data:               d.Data,
	}
}

// calculateEndChar computes the end character position for a diagnostic range.
//
// Takes d (*ast_domain.Diagnostic) which provides the location and expression
// data for the diagnostic.
//
// Returns int which is the end character position. When the expression is
// empty, adds one to ensure at least one character is highlighted.
func calculateEndChar(d *ast_domain.Diagnostic) int {
	endChar := d.Location.Column - 1 + len(d.Expression)
	if len(d.Expression) == 0 {
		endChar++
	}
	return endChar
}

// convertSeverityToLSP converts an AST diagnostic severity to its LSP protocol
// equivalent. Returns Warning as the default for unknown severity levels.
//
// Takes severity (ast_domain.Severity) which specifies the severity to convert.
//
// Returns protocol.DiagnosticSeverity which is the matching LSP severity.
func convertSeverityToLSP(severity ast_domain.Severity) protocol.DiagnosticSeverity {
	switch severity {
	case ast_domain.Error:
		return protocol.DiagnosticSeverityError
	case ast_domain.Info:
		return protocol.DiagnosticSeverityInformation
	default:
		return protocol.DiagnosticSeverityWarning
	}
}

// convertRelatedInfoToLSP turns AST related information into LSP protocol
// format.
//
// Takes d (*ast_domain.Diagnostic) which holds the diagnostic with related
// information to convert.
//
// Returns []protocol.DiagnosticRelatedInformation which holds the converted
// related information, or nil if there is none.
func convertRelatedInfoToLSP(d *ast_domain.Diagnostic) []protocol.DiagnosticRelatedInformation {
	if len(d.RelatedInfo) == 0 {
		return nil
	}

	relatedInfo := make([]protocol.DiagnosticRelatedInformation, 0, len(d.RelatedInfo))
	for _, ri := range d.RelatedInfo {
		relatedInfo = append(relatedInfo, protocol.DiagnosticRelatedInformation{
			Location: protocol.Location{
				URI: protocol.DocumentURI("file://" + d.SourcePath),
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      safeconv.IntToUint32(ri.Location.Line - 1),
						Character: safeconv.IntToUint32(ri.Location.Column - 1),
					},
					End: protocol.Position{
						Line:      safeconv.IntToUint32(ri.Location.Line - 1),
						Character: safeconv.IntToUint32(ri.Location.Column),
					},
				},
			},
			Message: ri.Message,
		})
	}
	return relatedInfo
}
