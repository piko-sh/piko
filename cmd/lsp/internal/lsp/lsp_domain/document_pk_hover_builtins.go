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

// sectionSeparator is the markdown section separator used in hover documentation.
const sectionSeparator = "\n\n"

// pikoBuiltinDocumentation holds documentation for a Piko builtin function.
type pikoBuiltinDocumentation struct {
	// Name is the function name, such as "len" or "T".
	Name string

	// Signature holds the function or method signature text.
	Signature string

	// Description explains what the built-in does.
	Description string

	// Accepts describes the argument types accepted by the built-in.
	Accepts string

	// Returns describes what the function gives back.
	Returns string

	// Example contains sample code showing how to use this built-in.
	Example string

	// Note contains extra information to display to the user.
	Note string

	// DocumentsURL is the relative path to the documentation page.
	DocumentsURL string
}

var (
	// pikoBuiltinDocumentations is the registry of all documented
	// Piko builtin functions.
	pikoBuiltinDocumentations = map[string]pikoBuiltinDocumentation{
		"len": {
			Name:        "len",
			Signature:   "len(x) int",
			Description: "Returns the length of an array, slice, map, or string.",
			Accepts:     "Array, slice, map, or string",
			Returns:     "`int` - the number of elements or characters",
			Example: `<span p-if="len(state.items) > 0">
	  {{ len(state.items) }} items found
	</span>`,
			DocumentsURL: "/docs/api/builtins/len",
		},
		"cap": {
			Name:         "cap",
			Signature:    "cap(x) int",
			Description:  "Returns the capacity of an array or slice.",
			Accepts:      "Array or slice",
			Returns:      "`int` - the capacity (maximum length without reallocation)",
			Example:      `<span p-text="cap(state.buffer)"></span>`,
			Note:         "For arrays, cap returns the same value as len.",
			DocumentsURL: "/docs/api/builtins/cap",
		},
		"append": {
			Name:         "append",
			Signature:    "append(slice, elems...) []T",
			Description:  "Appends elements to a slice and returns the resulting slice.",
			Accepts:      "First argument must be a slice, subsequent arguments must be assignable to the slice element type",
			Returns:      "Same type as the input slice",
			Example:      `{{ append(state.tags, "new-tag") }}`,
			Note:         "The original slice is not modified; a new slice is returned.",
			DocumentsURL: "/docs/api/builtins/append",
		},
		"min": {
			Name:        "min",
			Signature:   "min(x, y...) T",
			Description: "Returns the smallest value from the provided arguments.",
			Accepts:     "Ordered types (numbers or strings) - all arguments must be the same type",
			Returns:     "Same type as the arguments",
			Example: `<span p-text="min(state.price, state.salePrice)"></span>
	<span p-text="min(100, state.quantity, state.maxQuantity)"></span>`,
			DocumentsURL: "/docs/api/builtins/min",
		},
		"max": {
			Name:        "max",
			Signature:   "max(x, y...) T",
			Description: "Returns the largest value from the provided arguments.",
			Accepts:     "Ordered types (numbers or strings) - all arguments must be the same type",
			Returns:     "Same type as the arguments",
			Example: `<span p-text="max(state.score, state.highScore)"></span>
	<progress :value="state.progress" :max="max(100, state.target)"></progress>`,
			DocumentsURL: "/docs/api/builtins/max",
		},
		"T": {
			Name:        "T",
			Signature:   "T(key string, fallback ...string) string",
			Description: "Global translation lookup - retrieves a translated string by key from global translations.",
			Accepts:     "Translation key (string), optional fallback values (strings)",
			Returns:     "`string` - the translated text, or fallback if key not found",
			Example: `<h1 p-text="T('welcome_message')">Welcome</h1>
	<p p-text="T('greeting', 'Hello')">Fallback if key missing</p>`,
			Note:         "Issues a warning if key not found and no fallback provided.",
			DocumentsURL: "/docs/api/builtins/T",
		},
		"LT": {
			Name:        "LT",
			Signature:   "LT(key string, fallback ...string) string",
			Description: "Local translation lookup - retrieves a translated string by key from component-scoped translations.",
			Accepts:     "Translation key (string), optional fallback values (strings)",
			Returns:     "`string` - the translated text, or fallback if key not found",
			Example: `<label p-text="LT('form.email_label')">Email</label>
	<button p-text="LT('buttons.submit', 'Submit')">Submit</button>`,
			Note:         "Uses translations defined in the component's <i18n> block.",
			DocumentsURL: "/docs/api/builtins/LT",
		},
		"F": {
			Name:        "F",
			Signature:   "F(value any) *FormatBuilder",
			Description: "Locale-free formatting - formats a value using its default string representation without thousand separators or currency symbols.",
			Accepts:     "int, float64, string, bool, maths.Decimal, maths.BigInt, maths.Money, time.Time, time.Duration (and pointer variants)",
			Returns:     "`*FormatBuilder` - implements `fmt.Stringer`, auto-stringified in templates",
			Example: `<span p-text="F(state.Price).Precision(2)"></span>
	<p>{{ F(state.Total) }}</p>`,
			Note: "Chain `.Precision(n)`, `.Locale(code)`, `.Short()`, `.Medium()`, `.Long()`, `.Full()`, " +
				"`.DateOnly()`, `.TimeOnly()`, `.UTC()` for further control. Panic-safe: error-state values return empty string.",
			DocumentsURL: "/docs/api/utils/format-builder",
		},
		"LF": {
			Name:        "LF",
			Signature:   "LF(value any) *FormatBuilder",
			Description: "Locale-aware formatting - formats a value using the current request locale with thousand separators, decimal separators, and currency symbols.",
			Accepts:     "int, float64, string, bool, maths.Decimal, maths.BigInt, maths.Money, time.Time, time.Duration (and pointer variants)",
			Returns:     "`*FormatBuilder` - implements `fmt.Stringer`, auto-stringified in templates",
			Example: `<span p-text="r.LF(state.Price)"></span>
	<span p-text="r.LF(state.CreatedAt).Short().DateOnly()"></span>`,
			Note: "Can also be called as `r.LF()` in templates. Chain `.Precision(n)`, `.Locale(code)`, " +
				"`.Short()`, `.Medium()`, `.Long()`, `.Full()`, `.DateOnly()`, `.TimeOnly()`, `.UTC()` for further control.",
			DocumentsURL: "/docs/api/utils/format-builder",
		},
		"string": {
			Name:        "string",
			Signature:   "string(x) string",
			Description: "Converts a value to its string representation.",
			Accepts:     "int, uint, float, bool, byte, rune, Decimal, BigInt, time.Time",
			Returns:     "`string`",
			Example: `<span p-text="string(state.count)"></span>
	<input :value="string(state.price)" />`,
			DocumentsURL: "/docs/api/builtins/string",
		},
		"int": {
			Name:        "int",
			Signature:   "int(x) int",
			Description: "Converts a value to an integer.",
			Accepts:     "int, uint, float, bool, byte, rune, string, Decimal, BigInt",
			Returns:     "`int`",
			Example: `<span p-text="int(state.floatValue)"></span>
	<span p-text="int('42')"></span>`,
			Note:         "Truncates floating-point values toward zero.",
			DocumentsURL: "/docs/api/builtins/int",
		},
		"int64": {
			Name:         "int64",
			Signature:    "int64(x) int64",
			Description:  "Converts a value to a 64-bit integer.",
			Accepts:      "int, uint, float, bool, byte, rune, string, Decimal, BigInt",
			Returns:      "`int64`",
			Example:      `<span p-text="int64(state.largeNumber)"></span>`,
			DocumentsURL: "/docs/api/builtins/int64",
		},
		"int32": {
			Name:         "int32",
			Signature:    "int32(x) int32",
			Description:  "Converts a value to a 32-bit integer.",
			Accepts:      "int, uint, float, bool, byte, rune, string, Decimal, BigInt",
			Returns:      "`int32`",
			Example:      `<span p-text="int32(state.value)"></span>`,
			DocumentsURL: "/docs/api/builtins/int32",
		},
		"int16": {
			Name:         "int16",
			Signature:    "int16(x) int16",
			Description:  "Converts a value to a 16-bit integer.",
			Accepts:      "int, uint, float, bool, byte, rune, string, Decimal, BigInt",
			Returns:      "`int16`",
			Example:      `<span p-text="int16(state.smallValue)"></span>`,
			DocumentsURL: "/docs/api/builtins/int16",
		},
		"float": {
			Name:        "float",
			Signature:   "float(x) float64",
			Description: "Converts a value to a 64-bit floating-point number.",
			Accepts:     "int, uint, float, bool, byte, rune, string, Decimal, BigInt",
			Returns:     "`float64`",
			Example: `<span p-text="float(state.intValue)"></span>
	<span p-text="float('3.14')"></span>`,
			DocumentsURL: "/docs/api/builtins/float",
		},
		"float64": {
			Name:         "float64",
			Signature:    "float64(x) float64",
			Description:  "Converts a value to a 64-bit floating-point number.",
			Accepts:      "int, uint, float, bool, byte, rune, string, Decimal, BigInt",
			Returns:      "`float64`",
			Example:      `<span p-text="float64(state.value)"></span>`,
			DocumentsURL: "/docs/api/builtins/float64",
		},
		"float32": {
			Name:         "float32",
			Signature:    "float32(x) float32",
			Description:  "Converts a value to a 32-bit floating-point number.",
			Accepts:      "int, uint, float, bool, byte, rune, string, Decimal, BigInt",
			Returns:      "`float32`",
			Example:      `<span p-text="float32(state.value)"></span>`,
			Note:         "May lose precision compared to float64.",
			DocumentsURL: "/docs/api/builtins/float32",
		},
		"bool": {
			Name:        "bool",
			Signature:   "bool(x) bool",
			Description: "Converts a value to a boolean.",
			Accepts:     "string, int, uint, float, bool, byte, rune, Decimal, BigInt, time.Time",
			Returns:     "`bool`",
			Example: `<div p-if="bool(state.enabled)">Enabled</div>
	<div p-if="bool(state.count)">Has items</div>`,
			Note:         "Zero values convert to false, non-zero to true. Empty strings are false.",
			DocumentsURL: "/docs/api/builtins/bool",
		},
		"decimal": {
			Name:        "decimal",
			Signature:   "decimal(x) maths.Decimal",
			Description: "Converts a value to an arbitrary-precision decimal number.",
			Accepts:     "int, uint, byte, rune, string, Decimal, BigInt (NOT float, bool, time.Time)",
			Returns:     "`maths.Decimal` - arbitrary precision decimal",
			Example: `<span p-text="decimal('123.456')"></span>
	<span p-text="decimal(state.price)"></span>`,
			Note:         "Use for financial calculations where precision matters. Does not accept float to avoid precision loss.",
			DocumentsURL: "/docs/api/builtins/decimal",
		},
		"bigint": {
			Name:        "bigint",
			Signature:   "bigint(x) maths.BigInt",
			Description: "Converts a value to an arbitrary-precision integer.",
			Accepts:     "int, uint, byte, rune, string, Decimal, BigInt (NOT float, bool, time.Time)",
			Returns:     "`maths.BigInt` - arbitrary precision integer",
			Example: `<span p-text="bigint('99999999999999999999')"></span>
	<span p-text="bigint(state.largeId)"></span>`,
			Note:         "Use for very large integers that exceed int64 range.",
			DocumentsURL: "/docs/api/builtins/bigint",
		},
	}

	// builtinNames is the list of all builtin function names for quick lookup.
	builtinNames = map[string]struct{}{
		"len": {}, "cap": {}, "append": {}, "min": {}, "max": {},
		"T": {}, "LT": {}, "F": {}, "LF": {},
		"string": {}, "int": {}, "int64": {}, "int32": {}, "int16": {},
		"float": {}, "float64": {}, "float32": {},
		"bool": {}, "decimal": {}, "bigint": {},
	}
)

// checkBuiltinHoverContext checks if the cursor is on a built-in function name.
//
// Takes line (string) which is the current line text.
// Takes cursor (int) which is the cursor position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context when the cursor is on
// a built-in function, or nil when no match is found.
func (*document) checkBuiltinHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	builtinName, startPosition, endPosition := findBuiltinAtCursor(line, cursor)
	if builtinName == "" {
		return nil
	}

	if _, exists := pikoBuiltinDocumentations[builtinName]; !exists {
		return nil
	}

	return &PKHoverContext{
		Kind:     PKDefBuiltinFunction,
		Name:     builtinName,
		Position: position,
		Range: protocol.Range{
			Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(startPosition)},
			End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(endPosition)},
		},
	}
}

// getBuiltinHover returns hover information for a builtin function.
//
// Takes ctx (*PKHoverContext) which provides the hover request context.
//
// Returns *protocol.Hover which contains the hover information to display.
// Returns error which is always nil here.
func (*document) getBuiltinHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	builtinDocumentation, exists := pikoBuiltinDocumentations[ctx.Name]
	if !exists {
		return nil, nil
	}

	content := formatBuiltinDocumentation(builtinDocumentation)

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: content,
		},
		Range: &ctx.Range,
	}, nil
}

// findBuiltinAtCursor finds a builtin function name at the cursor position.
//
// Takes line (string) which contains the text to search.
// Takes cursor (int) which is the position within the line.
//
// Returns builtinName (string) which is the builtin name if found, or empty
// if no builtin is at the cursor.
// Returns startPosition (int) which is the start position of the builtin name.
// Returns endPosition (int) which is the end position of the builtin name.
func findBuiltinAtCursor(line string, cursor int) (builtinName string, startPosition, endPosition int) {
	if cursor > len(line) {
		return "", 0, 0
	}

	start := cursor
	for start > 0 && isIdentifierChar(line[start-1]) {
		start--
	}

	end := cursor
	for end < len(line) && isIdentifierChar(line[end]) {
		end++
	}

	if start == end {
		return "", 0, 0
	}

	word := line[start:end]

	if _, exists := builtinNames[word]; !exists {
		return "", 0, 0
	}

	afterWord := end
	for afterWord < len(line) && (line[afterWord] == ' ' || line[afterWord] == '\t') {
		afterWord++
	}
	if afterWord >= len(line) || line[afterWord] != '(' {
		return "", 0, 0
	}

	if start > 0 && line[start-1] == '.' {
		return "", 0, 0
	}

	return word, start, end
}

// isIdentifierChar reports whether the given byte is valid in an identifier.
//
// Takes c (byte) which is the character to check.
//
// Returns bool which is true if the character is a letter, digit, or underscore.
func isIdentifierChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// formatBuiltinDocumentation formats built-in function documentation as
// markdown.
//
// Takes builtinDocumentation (pikoBuiltinDocumentation) which
// holds the function documentation to format.
//
// Returns string which is the formatted markdown text.
func formatBuiltinDocumentation(builtinDocumentation pikoBuiltinDocumentation) string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "## `%s`%s", builtinDocumentation.Name, sectionSeparator)

	b.WriteString(builtinDocumentation.Description)
	b.WriteString(sectionSeparator)

	if builtinDocumentation.Signature != "" {
		b.WriteString("**Signature:** `")
		b.WriteString(builtinDocumentation.Signature)
		b.WriteString("`")
		b.WriteString(sectionSeparator)
	}

	if builtinDocumentation.Accepts != "" {
		b.WriteString("**Accepts:** ")
		b.WriteString(builtinDocumentation.Accepts)
		b.WriteString(sectionSeparator)
	}

	if builtinDocumentation.Returns != "" {
		b.WriteString("**Returns:** ")
		b.WriteString(builtinDocumentation.Returns)
		b.WriteString(sectionSeparator)
	}

	if builtinDocumentation.Example != "" {
		b.WriteString("**Example:**\n```piko\n")
		b.WriteString(builtinDocumentation.Example)
		b.WriteString("\n```")
		b.WriteString(sectionSeparator)
	}

	if builtinDocumentation.Note != "" {
		b.WriteString("**Note:** ")
		b.WriteString(builtinDocumentation.Note)
		b.WriteString(sectionSeparator)
	}

	if builtinDocumentation.DocumentsURL != "" {
		b.WriteString("---")
		b.WriteString(sectionSeparator)
		_, _ = fmt.Fprintf(&b, "[Documentation](%s)", builtinDocumentation.DocumentsURL)
	}

	return b.String()
}
