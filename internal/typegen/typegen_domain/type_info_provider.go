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

package typegen_domain

// TypeInfoProvider provides TypeScript type information for LSP intellisense.
// This is the interface that the LSP depends on to get completion data for
// piko.* and action.* namespaces.
type TypeInfoProvider interface {
	// GetPikoCompletions returns completions for the piko namespace,
	// returning top-level completions (refs, partial, etc.) when
	// namespace is empty, or sub-namespace completions when namespace
	// is "nav", "form", etc.
	//
	// Takes namespace (string) which specifies the
	// sub-namespace, or empty for top-level.
	//
	// Returns []CompletionItem which contains the available completions.
	GetPikoCompletions(namespace string) []CompletionItem

	// GetPikoSubNamespaces returns the available piko sub-namespaces.
	// Used when the user types "piko." to show nav, form, ui, etc.
	//
	// Returns []string which contains the sub-namespace names.
	GetPikoSubNamespaces() []string

	// GetActionCompletions returns completions for registered actions, filtered by
	// the given prefix.
	//
	// Takes prefix (string) which filters action names (e.g., "customer" matches
	// customer.* actions).
	//
	// Returns []CompletionItem which contains matching actions discovered from the
	// ActionManifest during annotation.
	GetActionCompletions(prefix string) []CompletionItem
}

// CompletionItem represents a single completion suggestion.
type CompletionItem struct {
	// Label is the display name shown in the completion list.
	Label string

	// Detail is additional information shown next to the label (e.g., signature).
	Detail string

	// Documentation is the full documentation for the item.
	Documentation string

	// InsertText is the text to insert when the item is selected.
	// If empty, Label is used.
	InsertText string

	// Kind indicates the type of completion (function, property, etc.).
	Kind CompletionItemKind
}

// CompletionItemKind indicates the type of a completion item.
type CompletionItemKind int

const (
	// CompletionKindFunction is a function completion item.
	CompletionKindFunction CompletionItemKind = iota + 1

	// CompletionKindProperty indicates a property completion item.
	CompletionKindProperty

	// CompletionKindModule represents a module completion item.
	CompletionKindModule

	// CompletionKindVariable represents a variable in completion suggestions.
	CompletionKindVariable
)
