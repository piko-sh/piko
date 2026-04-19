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

import "go.lsp.dev/protocol"

// InlayHint represents an inline annotation displayed in the editor.
// It can show type information, parameter names, or other hints.
type InlayHint struct {
	// Label is the text to display for the hint.
	Label string `json:"label"`

	// Kind indicates the type of hint (type annotation or parameter name).
	Kind InlayHintKind `json:"kind,omitempty"`

	// Position is where the hint should be displayed.
	Position protocol.Position `json:"position"`

	// PaddingLeft adds space before the hint when true.
	PaddingLeft bool `json:"paddingLeft,omitempty"`

	// PaddingRight adds space after the hint when true.
	PaddingRight bool `json:"paddingRight,omitempty"`
}

// InlayHintKind represents the kind of inlay hint.
type InlayHintKind int

const (
	// InlayHintKindType represents a type annotation hint.
	InlayHintKindType InlayHintKind = 1

	// InlayHintKindParameter represents a parameter name hint.
	InlayHintKindParameter InlayHintKind = 2
)

// InlayHintParams is the parameter for textDocument/inlayHint requests.
type InlayHintParams struct {
	// TextDocument identifies the document to get hints for.
	TextDocument protocol.TextDocumentIdentifier `json:"textDocument"`

	// Range specifies the visible range to get hints for.
	Range protocol.Range `json:"range"`
}

// TypeHierarchyItem represents a type in the hierarchy.
type TypeHierarchyItem struct {
	// Data is preserved between prepare and supertypes/subtypes requests.
	Data any `json:"data,omitempty"`

	// Name is the name of the type.
	Name string `json:"name"`

	// Detail provides additional information about the type.
	Detail string `json:"detail,omitempty"`

	// URI is the document containing the type.
	URI protocol.DocumentURI `json:"uri"`

	// Tags contains additional symbol tags.
	Tags []protocol.SymbolTag `json:"tags,omitempty"`

	// Kind is the symbol kind (class, interface, etc.).
	Kind protocol.SymbolKind `json:"kind"`

	// Range is the full range of the type definition.
	Range protocol.Range `json:"range"`

	// SelectionRange is the range to select when navigating to the type.
	SelectionRange protocol.Range `json:"selectionRange"`
}

// TypeHierarchyPrepareParams is the parameter for the
// textDocument/prepareTypeHierarchy request.
type TypeHierarchyPrepareParams struct {
	protocol.TextDocumentPositionParams
}

// TypeHierarchySupertypesParams is the parameter for typeHierarchy/supertypes.
type TypeHierarchySupertypesParams struct {
	// Item is the type to get supertypes for.
	Item TypeHierarchyItem `json:"item"`
}

// TypeHierarchySubtypesParams is the parameter for typeHierarchy/subtypes.
type TypeHierarchySubtypesParams struct {
	// Item is the type to get subtypes for.
	Item TypeHierarchyItem `json:"item"`
}

// TypeHierarchyData holds the data needed to resolve supertypes/subtypes.
type TypeHierarchyData struct {
	// PackagePath is the canonical package path of the type.
	PackagePath string `json:"packagePath"`

	// TypeName is the name of the type.
	TypeName string `json:"typeName"`
}
