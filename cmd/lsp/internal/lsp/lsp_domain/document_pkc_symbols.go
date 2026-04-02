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
	"go.lsp.dev/protocol"
	"piko.sh/piko/wdk/safeconv"
)

// pkcTopLevelSections is the expected number of top-level SFC sections
// (script, style, template) used to pre-allocate the symbols slice.
const pkcTopLevelSections = 3

// getPKCDocumentSymbols returns a hierarchical outline of a PKC file's
// structure, showing state properties, functions, and CSS classes grouped
// under their SFC block sections.
//
// Returns []any which contains the document symbols.
// Returns error which is always nil.
func (d *document) getPKCDocumentSymbols() ([]any, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return []any{}, nil
	}

	symbols := make([]protocol.DocumentSymbol, 0, pkcTopLevelSections)

	scriptSymbol := d.buildPKCScriptSymbol(meta)
	if scriptSymbol != nil {
		symbols = append(symbols, *scriptSymbol)
	}

	styleSymbol := buildPKCStyleSymbol(meta)
	if styleSymbol != nil {
		symbols = append(symbols, *styleSymbol)
	}

	result := make([]any, len(symbols))
	for i := range symbols {
		result[i] = symbols[i]
	}

	return result, nil
}

// zeroRange is a zero-valued range used for container symbols that do not have
// meaningful source positions.
var zeroRange = protocol.Range{
	Start: protocol.Position{Line: 0, Character: 0},
	End:   protocol.Position{Line: 0, Character: 0},
}

// buildPKCScriptSymbol creates the <script> section symbol with state
// properties and functions as children.
//
// Takes meta (*pkcMetadata) which provides the extracted script metadata.
//
// Returns *protocol.DocumentSymbol for the script section, or nil if empty.
func (*document) buildPKCScriptSymbol(meta *pkcMetadata) *protocol.DocumentSymbol {
	children := make([]protocol.DocumentSymbol, 0)

	if len(meta.StateProperties) > 0 {
		stateChildren := make([]protocol.DocumentSymbol, 0, len(meta.StateProperties))
		for _, prop := range meta.StateProperties {
			stateChildren = append(stateChildren, buildPKCPropertySymbol(prop))
		}

		children = append(children, protocol.DocumentSymbol{
			Name:           "state",
			Kind:           protocol.SymbolKindObject,
			Children:       stateChildren,
			Range:          zeroRange,
			SelectionRange: zeroRange,
		})
	}

	for _, function := range meta.Functions {
		children = append(children, buildPKCFunctionSymbol(function))
	}

	if len(children) == 0 {
		return nil
	}

	return &protocol.DocumentSymbol{
		Name:           "<script>",
		Kind:           protocol.SymbolKindNamespace,
		Children:       children,
		Range:          zeroRange,
		SelectionRange: zeroRange,
	}
}

// buildPKCStyleSymbol creates the <style> section symbol with CSS classes as
// children.
//
// Takes meta (*pkcMetadata) which provides the extracted style metadata.
//
// Returns *protocol.DocumentSymbol for the style section, or nil if empty.
func buildPKCStyleSymbol(meta *pkcMetadata) *protocol.DocumentSymbol {
	if len(meta.CSSClasses) == 0 {
		return nil
	}

	children := make([]protocol.DocumentSymbol, 0, len(meta.CSSClasses))
	for _, cls := range meta.CSSClasses {
		r := protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(cls.Line),
				Character: safeconv.IntToUint32(cls.Column),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(cls.Line),
				Character: safeconv.IntToUint32(cls.EndColumn),
			},
		}
		children = append(children, protocol.DocumentSymbol{
			Name:           "." + cls.Name,
			Kind:           protocol.SymbolKindClass,
			Range:          r,
			SelectionRange: r,
		})
	}

	return &protocol.DocumentSymbol{
		Name:           "<style>",
		Kind:           protocol.SymbolKindNamespace,
		Children:       children,
		Range:          zeroRange,
		SelectionRange: zeroRange,
	}
}

// buildPKCPropertySymbol creates a document symbol for a state property.
//
// Takes prop (*pkcStateProperty) which holds the property metadata.
//
// Returns protocol.DocumentSymbol for the property.
func buildPKCPropertySymbol(prop *pkcStateProperty) protocol.DocumentSymbol {
	r := protocol.Range{
		Start: protocol.Position{
			Line:      safeconv.IntToUint32(prop.Line),
			Character: safeconv.IntToUint32(prop.Column),
		},
		End: protocol.Position{
			Line:      safeconv.IntToUint32(prop.Line),
			Character: safeconv.IntToUint32(prop.Column + len(prop.Name)),
		},
	}

	return protocol.DocumentSymbol{
		Name:           prop.Name,
		Detail:         getPKCTypeString(prop),
		Kind:           protocol.SymbolKindProperty,
		Range:          r,
		SelectionRange: r,
	}
}

// buildPKCFunctionSymbol creates a document symbol for a function declaration.
//
// Takes function (*pkcFunction) which holds the function metadata.
//
// Returns protocol.DocumentSymbol for the function.
func buildPKCFunctionSymbol(function *pkcFunction) protocol.DocumentSymbol {
	r := protocol.Range{
		Start: protocol.Position{
			Line:      safeconv.IntToUint32(function.Line),
			Character: safeconv.IntToUint32(function.Column),
		},
		End: protocol.Position{
			Line:      safeconv.IntToUint32(function.Line),
			Character: safeconv.IntToUint32(function.Column + len(function.Name)),
		},
	}

	return protocol.DocumentSymbol{
		Name:           function.Name,
		Kind:           protocol.SymbolKindFunction,
		Range:          r,
		SelectionRange: r,
	}
}
