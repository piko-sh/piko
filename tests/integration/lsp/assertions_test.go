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

//go:build integration

package lsp_stress_test

import (
	"slices"
	"strings"
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/require"
)

func isGoplsDiagnostic(diagnostic protocol.Diagnostic) bool {
	switch diagnostic.Source {
	case "", "piko", "piko-lsp":
		return false
	default:
		return true
	}
}

func findPosition(t *testing.T, content, lineNeedle, symbol string) protocol.Position {
	t.Helper()
	line := findLine(content, lineNeedle)
	require.GreaterOrEqual(t, line, 0, "line containing %q not found", lineNeedle)
	lineText := strings.Split(content, "\n")[line]
	anchor := strings.Index(lineText, lineNeedle)
	offset := strings.Index(lineText[anchor:], symbol)
	require.GreaterOrEqual(t, offset, 0, "symbol %q not found at or after %q on line %q", symbol, lineNeedle, lineText)
	return protocol.Position{Line: uint32(line), Character: uint32(anchor + offset)}
}

func findLine(content, needle string) int {
	for index, line := range strings.Split(content, "\n") {
		if strings.Contains(line, needle) {
			return index
		}
	}
	return -1
}

func hasGoplsSource(diagnostics []protocol.Diagnostic) bool {
	return slices.ContainsFunc(diagnostics, isGoplsDiagnostic)
}

func noGoplsSource(diagnostics []protocol.Diagnostic) bool {
	return !hasGoplsSource(diagnostics)
}

func hasSyntheticGoplsDiagnostic(diagnostics []protocol.Diagnostic) bool {
	for index := range diagnostics {
		if isGoplsDiagnostic(diagnostics[index]) &&
			strings.Contains(diagnostics[index].Message, fakeSyntheticDiagnosticMessage) {
			return true
		}
	}
	return false
}

func goplsDiagnosticOnLine(line uint32) func([]protocol.Diagnostic) bool {
	return func(diagnostics []protocol.Diagnostic) bool {
		for index := range diagnostics {
			if isGoplsDiagnostic(diagnostics[index]) && diagnostics[index].Range.Start.Line == line {
				return true
			}
		}
		return false
	}
}

func anyDiagnosticMessageContains(diagnostics []protocol.Diagnostic, substr string) bool {
	for index := range diagnostics {
		if strings.Contains(diagnostics[index].Message, substr) {
			return true
		}
	}
	return false
}

func hoverContains(hover *protocol.Hover, substr string) bool {
	return hover != nil && strings.Contains(hover.Contents.Value, substr)
}

func completionHasItem(list *protocol.CompletionList, name string) bool {
	if list == nil {
		return false
	}
	for index := range list.Items {
		if list.Items[index].Label == name {
			return true
		}
	}
	return false
}

func locationInGoFile(locations []protocol.Location, pathContains string) bool {
	for index := range locations {
		target := string(locations[index].URI)
		if strings.HasSuffix(target, ".go") && strings.Contains(target, pathContains) {
			return true
		}
	}
	return false
}

func editsForURI(edit *protocol.WorkspaceEdit, uri protocol.DocumentURI) []protocol.TextEdit {
	if edit == nil {
		return nil
	}
	collected := append([]protocol.TextEdit(nil), edit.Changes[uri]...)
	for index := range edit.DocumentChanges {
		change := edit.DocumentChanges[index]
		if change.TextDocument.URI == uri {
			collected = append(collected, change.Edits...)
		}
	}
	return collected
}

func completionImportEdit(list *protocol.CompletionList, importPath string) (protocol.TextEdit, bool) {
	if list == nil {
		return protocol.TextEdit{}, false
	}
	for index := range list.Items {
		for _, edit := range list.Items[index].AdditionalTextEdits {
			if strings.Contains(edit.NewText, importPath) {
				return edit, true
			}
		}
	}
	return protocol.TextEdit{}, false
}

func rangeWithinBlock(rng protocol.Range, blockStartLine, blockEndLine int) bool {
	startLine := int(rng.Start.Line)
	return startLine > blockStartLine && startLine < blockEndLine
}
