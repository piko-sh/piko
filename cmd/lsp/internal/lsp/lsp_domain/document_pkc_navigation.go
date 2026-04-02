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
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/wdk/safeconv"
)

// statePrefix is the text pattern used to detect state property access.
const statePrefix = "state."

// checkPKCStatePropertyContext detects state.propName references in template
// attribute values and interpolation expressions.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character offset of the cursor in the line.
// Takes position (protocol.Position) which is the cursor position in the document.
//
// Returns *PKDefinitionContext for a state property, or nil if not matched.
func (*document) checkPKCStatePropertyContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	name, _, _, ok := scanStatePropertyAccess(line, cursor)
	if !ok {
		return nil
	}

	return &PKDefinitionContext{
		Name:     name,
		Kind:     PKDefPKCStateProperty,
		Position: position,
	}
}

// pkcSymbolLocation builds a single-location result from a symbol's position.
//
// Takes line (int) which specifies the line number of the symbol.
// Takes column (int) which specifies the column position of the symbol.
// Takes nameLen (int) which specifies the length of the symbol name.
//
// Returns []protocol.Location which contains the location spanning the symbol.
func (d *document) pkcSymbolLocation(line, column, nameLen int) []protocol.Location {
	return []protocol.Location{{
		URI: d.URI,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(line),
				Character: safeconv.IntToUint32(column),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(line),
				Character: safeconv.IntToUint32(column + nameLen),
			},
		},
	}}
}

// findPKCStatePropertyDefinition returns the location of a state property
// definition in the PKC script block using cached metadata.
//
// Takes name (string) which is the property name to find.
//
// Returns []protocol.Location which contains the property's definition.
// Returns error which is always nil.
func (d *document) findPKCStatePropertyDefinition(name string) ([]protocol.Location, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return nil, nil
	}

	prop, exists := meta.StateProperties[name]
	if !exists {
		return nil, nil
	}

	return d.pkcSymbolLocation(prop.Line, prop.Column, len(prop.Name)), nil
}

// findPKCHandlerDefinition returns the location of a function definition using
// cached PKC metadata, avoiding re-parsing the script.
//
// Takes name (string) which is the function name to find.
//
// Returns []protocol.Location which contains the function's definition.
// Returns error which is always nil.
func (d *document) findPKCHandlerDefinition(name string) ([]protocol.Location, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return nil, nil
	}

	function, exists := meta.Functions[name]
	if !exists {
		return nil, nil
	}

	return d.pkcSymbolLocation(function.Line, function.Column, len(function.Name)), nil
}

// findPKCRefDefinition finds a _ref or p-ref attribute in the PKC template
// by text scanning.
//
// Takes refName (string) which is the reference name to find.
//
// Returns []protocol.Location which contains the ref attribute's location.
// Returns error which is always nil.
func (d *document) findPKCRefDefinition(refName string) ([]protocol.Location, error) {
	sfc := d.getSFCResult()
	if sfc == nil || sfc.Template == "" {
		return nil, nil
	}

	templateBaseLine := sfc.TemplateContentLocation.Line - 1
	templateBaseCol := sfc.TemplateContentLocation.Column - 1

	for _, pattern := range []string{`_ref="` + refName + `"`, `p-ref="` + refName + `"`} {
		index := strings.Index(sfc.Template, pattern)
		if index == -1 {
			continue
		}

		relLine, relCol := byteOffsetToLineColumn(sfc.Template, index)
		absLine, absCol := adjustToDocumentPosition(relLine, relCol, templateBaseLine, templateBaseCol)

		return []protocol.Location{{
			URI: d.URI,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      safeconv.IntToUint32(absLine),
					Character: safeconv.IntToUint32(absCol),
				},
				End: protocol.Position{
					Line:      safeconv.IntToUint32(absLine),
					Character: safeconv.IntToUint32(absCol + len(pattern)),
				},
			},
		}}, nil
	}

	return nil, nil
}

// scanStatePropertyAccess detects a state.propName access pattern near the
// cursor by searching backwards for "state." and scanning forward to collect
// the property name, verifying it is not part of a longer identifier.
//
// Takes line (string) which is the source text to search.
// Takes cursor (int) which is the cursor position within the line.
//
// Returns name (string) which is the property name after "state.".
// Returns dotIndex (int) which is the position where "state." begins.
// Returns nameEnd (int) which is the position after the last character of the
// property name.
// Returns ok (bool) which is true when a valid state property was found at the
// cursor position.
func scanStatePropertyAccess(line string, cursor int) (name string, dotIndex, nameEnd int, ok bool) {
	end := min(cursor, len(line))
	textBefore := line[:end]

	dotIndex = strings.LastIndex(textBefore, statePrefix)
	if dotIndex == -1 {
		return "", 0, 0, false
	}

	nameStart := dotIndex + len(statePrefix)
	if nameStart > end {
		return "", 0, 0, false
	}

	if dotIndex > 0 && isJSIdentChar(line[dotIndex-1]) {
		return "", 0, 0, false
	}

	nameEnd = nameStart
	for nameEnd < len(line) && isJSIdentChar(line[nameEnd]) {
		nameEnd++
	}

	if nameEnd <= nameStart {
		return "", 0, 0, false
	}

	if cursor < nameStart || cursor > nameEnd {
		return "", 0, 0, false
	}

	return line[nameStart:nameEnd], dotIndex, nameEnd, true
}

// isJSIdentChar reports whether c is a valid JavaScript identifier character.
//
// Takes c (byte) which is the character to check.
//
// Returns bool which is true if c is a letter, digit, underscore, or dollar sign.
func isJSIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '$'
}
