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
	"strings"

	"go.lsp.dev/protocol"
)

// generateQuickFixes analyses a diagnostic and returns appropriate code
// actions. It examines the diagnostic's Code field to dispatch to specialised
// fix generators for type mismatches, undefined variables, missing properties,
// and other issues.
//
// Takes diagnostic (protocol.Diagnostic) which is the diagnostic to analyse.
// Takes document (*document) which provides the document context.
// Takes ws (*workspace) which provides workspace-level context.
//
// Returns []protocol.CodeAction which contains the applicable quick fixes.
func generateQuickFixes(ctx context.Context, diagnostic protocol.Diagnostic, document *document, ws *workspace) []protocol.CodeAction {
	actions := []protocol.CodeAction{}

	var diagnosticCode string
	if diagnostic.Code != nil {
		if codeString, ok := diagnostic.Code.(string); ok {
			diagnosticCode = codeString
		}
	}

	switch diagnosticCode {
	case DiagCodeTypeMismatch:
		if action := generateCoerceFix(ctx, diagnostic, document, ws); action != nil {
			actions = append(actions, *action)
		}
	case DiagCodeUndefinedVariable:
		actions = append(actions, generateUndefinedVariableFixes(ctx, diagnostic, document, ws)...)
	case DiagCodeUndefinedPartialAlias:
		actions = append(actions, generateUndefinedPartialAliasFixes(diagnostic, document, ws)...)
	case DiagCodeMissingRequiredProp:
		if action := generateAddMissingPropFix(ctx, diagnostic, document, ws); action != nil {
			actions = append(actions, *action)
		}
	case DiagCodeMissingImport:
		if action := generateAddImportFix(ctx, diagnostic, document, ws); action != nil {
			actions = append(actions, *action)
		}
	default:
		actions = append(actions, generateFallbackFixes(diagnostic, document, ws)...)
	}

	return actions
}

// generateFallbackFixes provides basic fixes when no structured diagnostic code
// is available. This provides backwards compatibility for diagnostics that
// haven't been updated to use codes.
//
// Takes diagnostic (protocol.Diagnostic) which is the diagnostic to
// generate fixes for.
//
// Returns []protocol.CodeAction which contains the fallback fix actions,
// or an empty slice if no fixes apply.
func generateFallbackFixes(diagnostic protocol.Diagnostic, _ *document, _ *workspace) []protocol.CodeAction {
	actions := []protocol.CodeAction{}
	message := strings.ToLower(diagnostic.Message)

	if strings.Contains(message, "undefined") || strings.Contains(message, "not found") {
		if varName := extractVariableName(diagnostic.Message); varName != "" {
			actions = append(actions, protocol.CodeAction{
				Title:       fmt.Sprintf("Add '%s' to component state", varName),
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
			})
		}
	}

	return actions
}

// extractVariableName tries to find a variable name in a diagnostic message.
// This is used as a fallback when structured diagnostic data is not available.
//
// Takes message (string) which is the diagnostic message to parse.
//
// Returns string which is the variable name found, or empty if none is found.
func extractVariableName(message string) string {
	if _, after, found := strings.Cut(message, "undefined:"); found {
		rest := strings.TrimSpace(after)
		if end := strings.IndexAny(rest, " \t\n.,;"); end != -1 {
			return rest[:end]
		}
		return rest
	}

	if before, after, found := strings.Cut(message, "'"); found {
		_ = before
		if varName, _, found := strings.Cut(after, "'"); found {
			return varName
		}
	}

	if _, after, found := strings.Cut(message, "variable "); found {
		rest := strings.TrimSpace(after)
		if end := strings.IndexAny(rest, " \t\n"); end != -1 {
			return rest[:end]
		}
	}

	return ""
}
