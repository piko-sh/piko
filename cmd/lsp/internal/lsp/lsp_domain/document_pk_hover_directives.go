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

// pikoDirectiveDocumentation holds documentation for a Piko directive.
type pikoDirectiveDocumentation struct {
	// Name is the directive name, such as "p-if" or "p-on".
	Name string

	// Description explains what this directive does.
	Description string

	// Syntax holds the usage pattern for the directive.
	Syntax string

	// Accepts describes the values the directive can take.
	Accepts string

	// Example provides a sample code snippet showing how to use this directive.
	Example string

	// Note contains extra information or guidance about the directive.
	Note string

	// DocumentsURL is the path to the documentation page for this directive.
	DocumentsURL string

	// Modifiers lists the available modifiers for p-on and p-event directives.
	Modifiers []string
}

// pikoDirectiveDocumentations is the registry of all documented Piko directives.
var pikoDirectiveDocumentations = map[string]pikoDirectiveDocumentation{
	"p-if": {
		Name:         "p-if",
		Description:  "Conditional rendering - element is excluded from output when expression is false",
		Syntax:       `p-if="expression"`,
		Accepts:      "Boolean expression",
		Example:      `<div p-if="state.isLoggedIn">Welcome back!</div>`,
		DocumentsURL: "/docs/api/directives/p-if",
	},
	"p-else-if": {
		Name:        "p-else-if",
		Description: "Alternative condition in if-else chain - evaluated when preceding p-if is false",
		Syntax:      `p-else-if="expression"`,
		Accepts:     "Boolean expression",
		Example: `<div p-if="state.role == 'admin'">Admin</div>
<div p-else-if="state.role == 'user'">User</div>`,
		Note:         "Must immediately follow a p-if or another p-else-if element.",
		DocumentsURL: "/docs/api/directives/p-else-if",
	},
	"p-else": {
		Name:        "p-else",
		Description: "Fallback branch - rendered when all preceding p-if/p-else-if conditions are false",
		Syntax:      `p-else`,
		Accepts:     "No value (presence only)",
		Example: `<div p-if="state.items.length > 0">Items found</div>
<div p-else>No items</div>`,
		Note:         "Must immediately follow a p-if or p-else-if element.",
		DocumentsURL: "/docs/api/directives/p-else",
	},
	"p-for": {
		Name:        "p-for",
		Description: "Loop iteration - repeats element for each item in a collection",
		Syntax:      `p-for="item in items"` + " or " + `p-for="(index, item) in items"`,
		Accepts:     "ForIn expression",
		Example: `<li p-for="user in state.users" p-key="user.id">
  <span p-text="user.name"></span>
</li>`,
		Note:         "Always use p-key with p-for for proper DOM reconciliation.",
		DocumentsURL: "/docs/api/directives/p-for",
	},
	"p-show": {
		Name:         "p-show",
		Description:  "Toggle visibility via CSS display property - element always remains in DOM",
		Syntax:       `p-show="expression"`,
		Accepts:      "Boolean expression",
		Example:      `<div p-show="state.isVisible">This toggles visibility</div>`,
		Note:         "Unlike p-if, the element is always rendered but hidden with display:none when false.",
		DocumentsURL: "/docs/api/directives/p-show",
	},

	"p-bind": {
		Name:        "p-bind",
		Description: "Dynamic attribute binding - binds an expression to any HTML attribute",
		Syntax:      `p-bind:attributeName="expression"`,
		Accepts:     "Expression that evaluates to an attribute value",
		Example: `<a p-bind:href="state.linkUrl">Dynamic link</a>
<img p-bind:src="state.imageUrl" p-bind:alt="state.imageAlt" />`,
		Note:         "Shorthand: :attributeName (e.g., :href instead of p-bind:href)",
		DocumentsURL: "/docs/api/directives/p-bind",
	},
	"p-text": {
		Name:         "p-text",
		Description:  "Set element text content from expression - output is HTML-escaped",
		Syntax:       `p-text="expression"`,
		Accepts:      "Expression that evaluates to a string",
		Example:      `<span p-text="state.userName">Placeholder</span>`,
		Note:         "Content is automatically HTML-escaped for security.",
		DocumentsURL: "/docs/api/directives/p-text",
	},
	"p-html": {
		Name:         "p-html",
		Description:  "Inject raw HTML content - bypasses HTML escaping",
		Syntax:       `p-html="expression"`,
		Accepts:      "Expression that evaluates to an HTML string",
		Example:      `<div p-html="state.richContent"></div>`,
		Note:         "Only use with trusted content to avoid XSS vulnerabilities.",
		DocumentsURL: "/docs/api/directives/p-html",
	},
	"p-model": {
		Name:        "p-model",
		Description: "Two-way data binding for form elements",
		Syntax:      `p-model="dataPath"`,
		Accepts:     "Data property path",
		Example: `<input p-model="state.form.email" type="email" />
<textarea p-model="state.form.message"></textarea>`,
		Note:         "Works with input, textarea, and select elements.",
		DocumentsURL: "/docs/api/directives/p-model",
	},

	"p-on": {
		Name:        "p-on",
		Description: "DOM event handler binding",
		Syntax:      `p-on:event[.modifier...]="handler"`,
		Accepts:     "Handler function name or expression",
		Example: `<button p-on:click="handleClick">Click me</button>
<form p-on:submit.prevent="action.contact.Send($form)">...</form>`,
		Modifiers:    []string{".prevent", ".stop", ".once", ".self"},
		Note:         "Modifiers are composable: p-on:submit.prevent.once",
		DocumentsURL: "/docs/api/directives/p-on",
	},
	"p-event": {
		Name:         "p-event",
		Description:  "Custom component event handler binding",
		Syntax:       `p-event:eventName[.modifier...]="handler"`,
		Accepts:      "Handler function name or expression",
		Example:      `<my-component p-event:update="handleUpdate"></my-component>`,
		Modifiers:    []string{".prevent", ".stop", ".once", ".self"},
		Note:         "Use for component-specific events, not standard DOM events. Modifiers are composable.",
		DocumentsURL: "/docs/api/directives/p-event",
	},

	"p-class": {
		Name:         "p-class",
		Description:  "Dynamic CSS class binding",
		Syntax:       `p-class="expression"`,
		Accepts:      "Object, string, or array of class names",
		Example:      `<div p-class="{ active: state.isActive, disabled: state.isDisabled }">...</div>`,
		Note:         "Merged with static class attribute if present.",
		DocumentsURL: "/docs/api/directives/p-class",
	},
	"p-style": {
		Name:         "p-style",
		Description:  "Dynamic inline style binding",
		Syntax:       `p-style="expression"`,
		Accepts:      "Style object with CSS properties",
		Example:      `<div p-style="{ color: state.textColor, fontSize: state.fontSize + 'px' }">...</div>`,
		DocumentsURL: "/docs/api/directives/p-style",
	},

	"p-ref": {
		Name:        "p-ref",
		Description: "Create a named reference to access the DOM element from scripts",
		Syntax:      `p-ref="refName"`,
		Accepts:     "Valid JavaScript identifier",
		Example: `<canvas p-ref="myCanvas"></canvas>
<!-- Access via refs.myCanvas in client script -->`,
		Note:         "Value must be a simple identifier (no expressions).",
		DocumentsURL: "/docs/api/directives/p-ref",
	},
	"p-slot": {
		Name:        "p-slot",
		Description: "Assign element content to a named slot in the parent partial",
		Syntax:      `p-slot="slotName"`,
		Accepts:     "Slot name string",
		Example: `<div p-slot="header">
  <h1>Page Title</h1>
</div>`,
		Note:         "Used when invoking partials to target specific slot locations.",
		DocumentsURL: "/docs/api/directives/p-slot",
	},
	"p-key": {
		Name:         "p-key",
		Description:  "Set unique key for DOM reconciliation in loops",
		Syntax:       `p-key="expression"`,
		Accepts:      "Expression that evaluates to a unique identifier",
		Example:      `<li p-for="item in state.items" p-key="item.id">...</li>`,
		Note:         "Essential for efficient updates when list items change.",
		DocumentsURL: "/docs/api/directives/p-key",
	},
	"p-context": {
		Name:         "p-context",
		Description:  "Set scoping prefix for automatic key generation within subtree",
		Syntax:       `p-context="expression"`,
		Accepts:      "Expression for key prefix",
		Example:      `<div p-context="'section-' + state.sectionId">...</div>`,
		Note:         "Framework-internal directive for key scoping.",
		DocumentsURL: "/docs/api/directives/p-context",
	},

	"p-scaffold": {
		Name:         "p-scaffold",
		Description:  "Component structure generation marker (framework-internal)",
		Syntax:       `p-scaffold` + " or " + `p-scaffold="true"`,
		Accepts:      "Boolean flag (presence means true)",
		Example:      `<div p-scaffold>...</div>`,
		Note:         "Used internally during component compilation.",
		DocumentsURL: "/docs/api/directives/p-scaffold",
	},

	"p-timeline": {
		Name:         "p-timeline",
		Description:  "Animation timeline directive - controls element visibility during timeline playback (PKC only)",
		Syntax:       `p-timeline:hidden`,
		Accepts:      "No value (presence only)",
		Example:      `<h1 p-ref="title" p-timeline:hidden>Hello, Piko</h1>`,
		DocumentsURL: "/docs/api/directives/p-timeline",
		Note: "Elements with p-timeline:hidden are hidden via CSS until the animation timeline's show action reveals them. " +
			"The compiler transforms the attribute to p-timeline-hidden in the output. " +
			"Only supported in PKC files with enable=\"animation\".",
	},
}

// checkDirectiveHoverContext checks if the cursor is on a p-* directive.
//
// Takes line (string) which is the current line text.
// Takes cursor (int) which is the cursor position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context when the cursor is on
// a directive, or nil when no match is found.
func (*document) checkDirectiveHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	directiveName, startPosition, endPosition := findDirectiveAtCursor(line, cursor)
	if directiveName == "" {
		return nil
	}

	lookupName := normaliseDirectiveName(directiveName)

	if _, exists := pikoDirectiveDocumentations[lookupName]; !exists {
		return nil
	}

	return &PKHoverContext{
		Kind:     PKDefDirective,
		Name:     lookupName,
		Position: position,
		Range: protocol.Range{
			Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(startPosition)},
			End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(endPosition)},
		},
	}
}

// getDirectiveHover returns hover information for a p-* directive.
//
// Takes ctx (*PKHoverContext) which provides the hover request context.
//
// Returns *protocol.Hover which contains the hover information to display,
// or nil if the directive is not found.
// Returns error which is always nil for this method.
func (*document) getDirectiveHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	directiveDocumentation, exists := pikoDirectiveDocumentations[ctx.Name]
	if !exists {
		return nil, nil
	}

	content := formatDirectiveDocumentation(directiveDocumentation)

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: content,
		},
		Range: &ctx.Range,
	}, nil
}

// findDirectiveAtCursor finds a p-* directive at the cursor position.
//
// Takes line (string) which contains the text to search.
// Takes cursor (int) which is the position within the line.
//
// Returns directiveName (string) which is the full directive name (e.g.
// "p-on:click.prevent"), or empty if no directive is found.
// Returns startPosition (int) which is the start position of the directive.
// Returns endPosition (int) which is the end position of the directive.
func findDirectiveAtCursor(line string, cursor int) (directiveName string, startPosition, endPosition int) {
	searchStart := max(0, cursor-50)
	searchEnd := min(cursor+30, len(line))

	lastIndex := findDirectiveStartPosition(line, searchStart, searchEnd, cursor)
	if lastIndex == -1 {
		return "", 0, 0
	}

	endIndex := findDirectiveEndPosition(line, lastIndex)

	if cursor < lastIndex || cursor > endIndex {
		return "", 0, 0
	}

	return line[lastIndex:endIndex], lastIndex, endIndex
}

// findDirectiveStartPosition finds the starting position of a p- directive near
// the cursor.
//
// Takes line (string) which contains the text to search.
// Takes searchStart (int) which specifies the start index of the search range.
// Takes searchEnd (int) which specifies the end index of the search range.
// Takes cursor (int) which is the cursor position to search near.
//
// Returns int which is the index of the last valid directive prefix at or
// before the cursor, or -1 if none is found.
func findDirectiveStartPosition(line string, searchStart, searchEnd, cursor int) int {
	lastIndex := -1
	for i := searchStart; i < searchEnd; i++ {
		if !isDirectivePrefix(line, i) {
			continue
		}
		if isPrecededByValidChar(line, i) && i <= cursor {
			lastIndex = i
		}
	}
	return lastIndex
}

// isDirectivePrefix checks if position i in line starts with "p-".
//
// Takes line (string) which is the text to check.
// Takes i (int) which is the position to check from.
//
// Returns bool which is true if the prefix "p-" is found at position i.
func isDirectivePrefix(line string, i int) bool {
	return i+2 <= len(line) && line[i] == 'p' && line[i+1] == '-'
}

// isPrecededByValidChar checks if the character before position i is valid
// (whitespace, quote, or start of line).
//
// Takes line (string) which is the text to check.
// Takes i (int) which is the position to check before.
//
// Returns bool which is true if the preceding character is whitespace, a quote,
// or if i is at the start of the line.
func isPrecededByValidChar(line string, i int) bool {
	if i == 0 {
		return true
	}
	previous := line[i-1]
	return previous == ' ' || previous == '\t' || previous == '"' || previous == '\''
}

// findDirectiveEndPosition finds the end position of a
// directive starting at lastIndex.
//
// Takes line (string) which contains the text to scan.
// Takes lastIndex (int) which is the starting position within the line.
//
// Returns int which is the position after the last directive character.
func findDirectiveEndPosition(line string, lastIndex int) int {
	endIndex := lastIndex
	for i := lastIndex; i < len(line); i++ {
		if !isDirectiveChar(line[i]) {
			break
		}
		endIndex = i + 1
	}
	return endIndex
}

// isDirectiveChar reports whether the character is valid in a directive name.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if the character is a letter, digit, hyphen,
// colon, or full stop.
func isDirectiveChar(character byte) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
		(character >= '0' && character <= '9') || character == '-' || character == ':' || character == '.'
}

// normaliseDirectiveName extracts the base directive name for documentation
// lookup.
//
// Takes fullName (string) which may contain arguments or modifiers.
//
// Returns string which is the base directive name.
func normaliseDirectiveName(fullName string) string {
	if strings.HasPrefix(fullName, "p-bind:") {
		return "p-bind"
	}
	if strings.HasPrefix(fullName, "p-on:") {
		return "p-on"
	}
	if strings.HasPrefix(fullName, "p-event:") {
		return "p-event"
	}

	if dotIndex := strings.Index(fullName, "."); dotIndex != -1 {
		fullName = fullName[:dotIndex]
	}

	return fullName
}

// formatDirectiveDocumentation formats directive documentation as markdown.
//
// Takes directiveDocumentation (pikoDirectiveDocumentation)
// which holds the directive details to format.
//
// Returns string which is the formatted markdown text.
func formatDirectiveDocumentation(directiveDocumentation pikoDirectiveDocumentation) string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "## `%s`\n\n", directiveDocumentation.Name)

	b.WriteString(directiveDocumentation.Description)
	b.WriteString("\n\n")

	if directiveDocumentation.Syntax != "" {
		b.WriteString("**Syntax:** `")
		b.WriteString(directiveDocumentation.Syntax)
		b.WriteString("`\n\n")
	}

	if directiveDocumentation.Accepts != "" {
		b.WriteString("**Accepts:** ")
		b.WriteString(directiveDocumentation.Accepts)
		b.WriteString("\n\n")
	}

	if len(directiveDocumentation.Modifiers) > 0 {
		b.WriteString("**Modifiers:**\n")
		for _, mod := range directiveDocumentation.Modifiers {
			_, _ = fmt.Fprintf(&b, "- `%s`", mod)
			b.WriteString(modifierDescription(mod))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if directiveDocumentation.Example != "" {
		b.WriteString("**Example:**\n```html\n")
		b.WriteString(directiveDocumentation.Example)
		b.WriteString("\n```\n\n")
	}

	if directiveDocumentation.Note != "" {
		b.WriteString("**Note:** ")
		b.WriteString(directiveDocumentation.Note)
		b.WriteString("\n\n")
	}

	if directiveDocumentation.DocumentsURL != "" {
		b.WriteString("---\n\n")
		_, _ = fmt.Fprintf(&b, "[Documentation](%s)", directiveDocumentation.DocumentsURL)
	}

	return b.String()
}

// modifierDescription returns a description for event modifiers.
//
// Takes mod (string) which is the modifier name to look up.
//
// Returns string which is the human-readable description, or empty if unknown.
func modifierDescription(mod string) string {
	switch mod {
	case ".prevent":
		return " - calls preventDefault()"
	case ".stop":
		return " - calls stopPropagation()"
	case ".once":
		return " - handler fires only once"
	case ".self":
		return " - only fires when event.target is the element itself"
	default:
		return ""
	}
}
