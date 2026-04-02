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
	"fmt"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// pkcDiagnosticSource is the source label for PKC diagnostics.
	pkcDiagnosticSource = "piko"

	// minHandlerRefSuffixLen is the minimum number of characters after an event
	// binding equals sign (e.g. `="x"`) needed for a valid handler reference.
	minHandlerRefSuffixLen = 3

	// diagnosticSliceCapacity is the default pre-allocation capacity for
	// diagnostic slices.
	diagnosticSliceCapacity = 4
)

// getPKCDiagnostics returns lightweight diagnostics for a PKC file by
// checking template references against extracted metadata.
//
// Returns []protocol.Diagnostic with any issues found.
func (d *document) getPKCDiagnostics() []protocol.Diagnostic {
	meta := d.getPKCMetadata()
	sfc := d.getSFCResult()
	if meta == nil || sfc == nil || sfc.Template == "" {
		return nil
	}

	baseLine := sfc.TemplateContentLocation.Line - 1
	baseCol := sfc.TemplateContentLocation.Column - 1

	diagnostics := make([]protocol.Diagnostic, 0, diagnosticSliceCapacity)
	diagnostics = append(diagnostics, checkPKCUnknownStateRefs(sfc.Template, baseLine, baseCol, meta.StateProperties)...)
	diagnostics = append(diagnostics, checkPKCUnknownHandlerRefs(sfc.Template, baseLine, baseCol, meta.Functions)...)

	return diagnostics
}

// makePKCDiagnostic builds a warning diagnostic at a template byte offset.
//
// Takes template (string) which contains the template text for position
// calculation.
// Takes byteOffset (int) which specifies the byte position within the template.
// Takes nameLen (int) which defines the length of the symbol name for range
// calculation.
// Takes baseLine (int) which provides the starting line in the document.
// Takes baseCol (int) which provides the starting column in the document.
// Takes message (string) which contains the diagnostic message to display.
//
// Returns protocol.Diagnostic which is a warning diagnostic with the
// calculated position range.
func makePKCDiagnostic(
	template string,
	byteOffset int,
	nameLen int,
	baseLine int,
	baseCol int,
	message string,
) protocol.Diagnostic {
	relLine, relCol := byteOffsetToLineColumn(template, byteOffset)
	absLine, absCol := adjustToDocumentPosition(relLine, relCol, baseLine, baseCol)

	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(absLine),
				Character: safeconv.IntToUint32(absCol),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(absLine),
				Character: safeconv.IntToUint32(absCol + nameLen),
			},
		},
		Severity: protocol.DiagnosticSeverityWarning,
		Source:   pkcDiagnosticSource,
		Message:  message,
	}
}

// checkPKCUnknownStateRefs scans the template for state.propName references
// and reports unknowns as diagnostics.
//
// Takes template (string) which is the raw template text.
// Takes baseLine (int) which is the 0-based line offset of the template
// content in the document.
// Takes baseCol (int) which is the 0-based column offset of the template
// content in the document.
// Takes stateProps (map[string]*pkcStateProperty) which holds the known state
// properties.
//
// Returns []protocol.Diagnostic for unknown state references.
func checkPKCUnknownStateRefs(
	template string,
	baseLine int,
	baseCol int,
	stateProps map[string]*pkcStateProperty,
) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	index := 0
	for index < len(template) {
		position := strings.Index(template[index:], statePrefix)
		if position == -1 {
			break
		}

		absPos := index + position

		if absPos > 0 && isJSIdentChar(template[absPos-1]) {
			index = absPos + 1
			continue
		}

		nameStart := absPos + len(statePrefix)
		nameEnd := nameStart
		for nameEnd < len(template) && isJSIdentChar(template[nameEnd]) {
			nameEnd++
		}

		if nameEnd <= nameStart {
			index = absPos + 1
			continue
		}

		propName := template[nameStart:nameEnd]
		if _, exists := stateProps[propName]; !exists {
			diagnostics = append(diagnostics, makePKCDiagnostic(
				template, nameStart, len(propName), baseLine, baseCol,
				fmt.Sprintf("Unknown state property '%s'", propName),
			))
		}

		index = nameEnd
	}

	return diagnostics
}

// checkPKCUnknownHandlerRefs scans the template for p-on:*="handler"
// references and reports unknown handlers as diagnostics.
//
// Takes template (string) which is the raw template text.
// Takes baseLine (int) which is the 0-based line offset.
// Takes baseCol (int) which is the 0-based column offset.
// Takes functions (map[string]*pkcFunction) which holds the known functions.
//
// Returns []protocol.Diagnostic for unknown handler references.
func checkPKCUnknownHandlerRefs(
	template string,
	baseLine int,
	baseCol int,
	functions map[string]*pkcFunction,
) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	index := 0
	for index < len(template) {
		position := strings.Index(template[index:], `p-on:`)
		if position == -1 {
			break
		}

		absPos := index + position

		eqIndex := strings.Index(template[absPos:], `="`)
		if eqIndex == -1 || absPos+eqIndex > len(template)-minHandlerRefSuffixLen {
			index = absPos + 1
			continue
		}

		valueStart := absPos + eqIndex + 2

		quoteEnd := strings.IndexByte(template[valueStart:], '"')
		if quoteEnd == -1 {
			break
		}

		handlerValue := template[valueStart : valueStart+quoteEnd]

		handlerName := extractHandlerName(handlerValue)

		if isSimpleFunctionRef(handlerName) {
			if _, exists := functions[handlerName]; !exists {
				diagnostics = append(diagnostics, makePKCDiagnostic(
					template, valueStart, len(handlerName), baseLine, baseCol,
					fmt.Sprintf("Unknown handler function '%s'", handlerName),
				))
			}
		}

		index = valueStart + quoteEnd + 1
	}

	return diagnostics
}

// extractHandlerName strips a trailing parenthesised argument
// list and trims whitespace to yield the bare function name
// from a handler attribute value.
//
// Takes handlerValue (string) which is the raw handler
// attribute value to parse.
//
// Returns string which is the bare function name.
func extractHandlerName(handlerValue string) string {
	name := handlerValue
	if parenIndex := strings.IndexByte(name, '('); parenIndex != -1 {
		name = name[:parenIndex]
	}
	return strings.TrimSpace(name)
}

// isSimpleFunctionRef reports whether name looks like a plain
// function identifier: non-empty, no dots, no spaces.
//
// Takes name (string) which is the handler name to check.
//
// Returns bool which is true if the name is a simple
// identifier.
func isSimpleFunctionRef(name string) bool {
	return name != "" && !strings.Contains(name, ".") && !strings.Contains(name, " ")
}
