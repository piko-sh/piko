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
)

// getPKCCompletions returns completion suggestions for PKC files at the given
// position. It dispatches to specific completion handlers based on the
// detected trigger context.
//
// Takes position (protocol.Position) which is the cursor position.
//
// Returns *protocol.CompletionList which contains the matching completions.
// Returns error when completion fails.
func (d *document) getPKCCompletions(position protocol.Position) (*protocol.CompletionList, error) {
	completionCtx := analyseCompletionContext(d, position)

	switch completionCtx.TriggerKind {
	case triggerMemberAccess:
		return d.getPKCMemberCompletions(completionCtx.BaseExpression, completionCtx.Prefix)
	case triggerDirective:
		return d.getDirectiveCompletions(completionCtx.Prefix)
	case triggerEventHandler:
		return d.getPKCEventHandlerCompletions(completionCtx.Prefix)
	case triggerRefAccess:
		return d.getPKCRefCompletions(completionCtx.Prefix)
	case triggerStateAccessJS:
		return d.getPKCStateFieldCompletions(completionCtx.Prefix)
	case triggerCSSClassValue:
		return d.getCSSClassCompletions(completionCtx.Prefix)
	case triggerDirectiveValue:
		return d.getPKCDirectiveValueCompletions(completionCtx.Prefix)
	default:
		return emptyCompletionList(), nil
	}
}

// getPKCMemberCompletions routes member access completions for PKC files based
// on the base expression (state, this, refs).
//
// Takes base (string) which is the expression before the dot.
// Takes prefix (string) which filters completions by substring.
//
// Returns *protocol.CompletionList which contains matching member completions.
// Returns error which is always nil.
func (d *document) getPKCMemberCompletions(base, prefix string) (*protocol.CompletionList, error) {
	switch base {
	case "state":
		return d.getPKCStateFieldCompletions(prefix)
	case "this":
		return d.getPKCThisCompletions(prefix)
	case "refs":
		return d.getPKCRefCompletions(prefix)
	default:
		return emptyCompletionList(), nil
	}
}

// getPKCStateFieldCompletions returns completions for state properties.
//
// Takes prefix (string) which filters by substring (case-insensitive).
//
// Returns *protocol.CompletionList which contains matching state property
// completions.
// Returns error which is always nil.
func (d *document) getPKCStateFieldCompletions(prefix string) (*protocol.CompletionList, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return emptyCompletionList(), nil
	}

	items := make([]protocol.CompletionItem, 0, len(meta.StateProperties))
	for _, prop := range meta.StateProperties {
		if prefix != "" && !containsSubstring(prop.Name, prefix) {
			continue
		}

		items = append(items, protocol.CompletionItem{
			Label:  prop.Name,
			Kind:   protocol.CompletionItemKindProperty,
			Detail: getPKCTypeString(prop),
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// getPKCEventHandlerCompletions returns function completions for p-on:*=""
// event handler attributes, including event placeholder variables.
//
// Takes prefix (string) which filters by substring.
//
// Returns *protocol.CompletionList which contains matching function completions.
// Returns error which is always nil.
func (d *document) getPKCEventHandlerCompletions(prefix string) (*protocol.CompletionList, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return emptyCompletionList(), nil
	}

	items := make([]protocol.CompletionItem, 0, len(meta.Functions)+len(eventPlaceholders))
	items = append(items, d.getEventPlaceholderCompletions(prefix)...)

	for _, function := range meta.Functions {
		if prefix != "" && !containsSubstring(function.Name, prefix) {
			continue
		}

		detail := "Function"
		if function.Exported {
			detail = "Exported function"
		}

		items = append(items, protocol.CompletionItem{
			Label:  function.Name,
			Kind:   protocol.CompletionItemKindFunction,
			Detail: detail,
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// pkcThisCompletionItem holds metadata for a this.* completion entry in PKC
// scripts.
type pkcThisCompletionItem struct {
	// label string // label is the display text shown for this completion item.
	label string

	// detail is the human-readable description shown in the completion popup.
	detail string

	// documentation is the documentation text for this completion item.
	documentation string
}

// pkcThisCompletions defines the available this.* members in a PKC component.
var pkcThisCompletions = []pkcThisCompletionItem{
	{label: "onConnected", detail: "Lifecycle hook", documentation: "Called when the element is added to the DOM."},
	{label: "onDisconnected", detail: "Lifecycle hook", documentation: "Called when the element is removed from the DOM."},
	{label: "onUpdated", detail: "Lifecycle hook", documentation: "Called when observed attributes change."},
	{label: "onBeforeRender", detail: "Lifecycle hook", documentation: "Called before the component renders."},
	{label: "onAfterRender", detail: "Lifecycle hook", documentation: "Called after the component renders."},
	{label: "onCleanup", detail: "Lifecycle hook", documentation: "Registers a cleanup function that runs when the component disconnects."},
	{label: "refs", detail: "Element references", documentation: "Access elements with _ref or p-ref attributes."},
	{label: "getValue", detail: "Form method", documentation: "Return the form value for this component."},
	{label: "getName", detail: "Form method", documentation: "Return the form field name for this component."},
	{label: "attachSlotListener", detail: "Slot method", documentation: "Attaches a listener for slot content changes. The callback is invoked immediately with initial content."},
	{label: "getSlottedElements", detail: "Slot method", documentation: "Returns elements assigned to a named slot."},
	{label: "hasSlotContent", detail: "Slot method", documentation: "Checks whether a slot has any assigned content."},
}

// getPKCThisCompletions returns completions for this.* in a PKC script block.
//
// Takes prefix (string) which filters by substring.
//
// Returns *protocol.CompletionList which contains matching this.* completions.
// Returns error which is always nil.
func (*document) getPKCThisCompletions(prefix string) (*protocol.CompletionList, error) {
	items := make([]protocol.CompletionItem, 0, len(pkcThisCompletions))

	for _, item := range pkcThisCompletions {
		if prefix != "" && !containsSubstring(item.label, prefix) {
			continue
		}

		items = append(items, protocol.CompletionItem{
			Label:  item.label,
			Kind:   protocol.CompletionItemKindMethod,
			Detail: item.detail,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: item.documentation,
			},
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// getPKCRefCompletions returns completions for refs.* in PKC files.
//
// Takes prefix (string) which filters by substring.
//
// Returns *protocol.CompletionList which contains matching ref name completions.
// Returns error which is always nil.
func (d *document) getPKCRefCompletions(prefix string) (*protocol.CompletionList, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return emptyCompletionList(), nil
	}

	items := make([]protocol.CompletionItem, 0, len(meta.Refs))
	for _, ref := range meta.Refs {
		if prefix != "" && !containsSubstring(ref, prefix) {
			continue
		}

		items = append(items, protocol.CompletionItem{
			Label:  ref,
			Kind:   protocol.CompletionItemKindVariable,
			Detail: "Element reference",
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// getPKCDirectiveValueCompletions returns completions for directive value
// expressions in PKC files (e.g. inside p-if="", p-text="").
//
// Takes prefix (string) which filters completions by matching substrings.
//
// Returns *protocol.CompletionList which contains matching completions.
// Returns error which is always nil.
func (d *document) getPKCDirectiveValueCompletions(prefix string) (*protocol.CompletionList, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return emptyCompletionList(), nil
	}

	items := make([]protocol.CompletionItem, 0, 1+len(meta.Functions))

	if prefix == "" || containsSubstring("state", prefix) {
		items = append(items, protocol.CompletionItem{
			Label:  "state",
			Kind:   protocol.CompletionItemKindVariable,
			Detail: "Component state",
		})
	}

	for _, function := range meta.Functions {
		if prefix != "" && !containsSubstring(function.Name, prefix) {
			continue
		}

		items = append(items, protocol.CompletionItem{
			Label:  function.Name,
			Kind:   protocol.CompletionItemKindFunction,
			Detail: "Function",
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}
