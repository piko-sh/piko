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

package jsident

import (
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/js_lexer"
)

const (
	// DefaultIdentifier is the identifier used when a name sanitises to nothing usable.
	DefaultIdentifier = "piko"

	// underscore is the repair character: it replaces illegal runes, prefixes leading digits
	// and suffixes reserved words.
	underscore = "_"

	// firstPrintableASCII is the first ASCII character that needs no escape.
	firstPrintableASCII = 0x20

	// lastPrintableASCII is the last ASCII character, above which the non-ASCII rules apply.
	lastPrintableASCII = 0x7E

	// firstSurrogate is the first UTF-16 surrogate code point. A surrogate cannot stand on
	// its own in source text and must be escaped.
	firstSurrogate = 0xD800

	// lastSurrogate is the last UTF-16 surrogate code point.
	lastSurrogate = 0xDFFF

	// byteOrderMark is U+FEFF, which is invisible and changes how the source is read.
	byteOrderMark = '\uFEFF'

	// lineSeparator is U+2028, which ends a line in older ECMAScript parsers even inside a
	// string literal.
	lineSeparator = '\u2028'

	// paragraphSeparator is U+2029, which ends a line in older ECMAScript parsers even
	// inside a string literal.
	paragraphSeparator = '\u2029'
)

var (
	// additionalReservedWords lists names the lexer tables leave out because they are only
	// reserved in some contexts. Piko emits nothing but strict-mode module code, where
	// binding any of them is a syntax error, so they count as reserved here.
	additionalReservedWords = []string{"await", "arguments", "eval"}

	// reservedWords is the set of words that may not be bound as an identifier in the
	// TypeScript piko emits.
	reservedWords = buildReservedWords()

	// separatorEscaper escapes the two separators esbuild's printer leaves raw.
	separatorEscaper = strings.NewReplacer(
		string(lineSeparator), `\u2028`,
		string(paragraphSeparator), `\u2029`,
	)
)

// IsReservedWord reports whether a word may not be bound as a TypeScript identifier.
//
// Takes name (string) which is the candidate identifier.
//
// Returns bool which is true when binding the name would be a syntax error.
func IsReservedWord(name string) bool {
	_, exists := reservedWords[name]
	return exists
}

// IsValidIdentifier reports whether a name can be emitted as a TypeScript binding name.
//
// Takes name (string) which is the candidate identifier.
//
// Returns bool which is true when the name is legal and not reserved.
func IsValidIdentifier(name string) bool {
	return js_ast.IsIdentifier(name) && !IsReservedWord(name)
}

// SanitiseIdentifier turns a user-controlled name into a legal TypeScript binding name.
//
// Takes name (string) which is the raw user-controlled name.
//
// Returns string which is a legal TypeScript identifier.
func SanitiseIdentifier(name string) string {
	if IsValidIdentifier(name) {
		return name
	}
	if name == "" {
		return DefaultIdentifier
	}

	candidate := name
	first, _ := utf8.DecodeRuneInString(name)
	if !js_ast.IsIdentifierStart(first) && js_ast.IsIdentifierContinue(first) {
		candidate = underscore + name
	}

	forced := js_ast.ForceValidIdentifier("", candidate)
	if IsReservedWord(forced) {
		return forced + underscore
	}

	return forced
}

// QuotePropertyName renders a property name for a TypeScript object literal or type.
//
// Takes name (string) which is the raw property name.
//
// Returns string which is the property name ready to place before its colon.
func QuotePropertyName(name string) string {
	if js_ast.IsIdentifier(name) {
		return name
	}

	return QuoteStringLiteral(name)
}

// QuoteStringLiteral renders a string as a double-quoted TypeScript literal.
//
// Takes value (string) which is the raw string.
//
// Returns string which is a complete quoted literal, delimiters included.
func QuoteStringLiteral(value string) string {
	quoted := separatorEscaper.Replace(string(helpers.QuoteForJSON(value, false)))
	if utf8.ValidString(quoted) {
		return quoted
	}

	repaired := strings.ToValidUTF8(value, string(utf8.RuneError))

	return separatorEscaper.Replace(string(helpers.QuoteForJSON(repaired, false)))
}

// IsPrintableUnescaped reports whether a rune can sit inside a double-quoted TypeScript
// literal exactly as it is.
//
// Takes character (rune) which is the candidate rune.
//
// Returns bool which is true when the rune needs no escape.
func IsPrintableUnescaped(character rune) bool {
	if character <= lastPrintableASCII {
		return character >= firstPrintableASCII && character != '\\' && character != '"'
	}
	if character == byteOrderMark || character == lineSeparator || character == paragraphSeparator {
		return false
	}

	return character < firstSurrogate || character > lastSurrogate
}

// buildReservedWords assembles the set of words that may not be bound as an identifier in
// the strict-mode module code piko emits.
//
// Returns map[string]struct{} which is the reserved word set.
func buildReservedWords() map[string]struct{} {
	capacity := len(js_lexer.Keywords) + len(js_lexer.StrictModeReservedWords) +
		len(additionalReservedWords)
	words := make(map[string]struct{}, capacity)

	for word := range js_lexer.Keywords {
		words[word] = struct{}{}
	}
	for word := range js_lexer.StrictModeReservedWords {
		words[word] = struct{}{}
	}
	for _, word := range additionalReservedWords {
		words[word] = struct{}{}
	}

	return words
}
