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

package i18n_domain

import (
	"strings"
	"unicode"

	"piko.sh/piko/internal/ast/ast_domain"
)

// PartKind represents the kind of part in an i18n template.
// It implements fmt.Stringer.
type PartKind uint8

const (
	// PartLiteral is plain text that is written to the output without changes.
	PartLiteral PartKind = iota

	// PartExpression represents a ${expression} interpolation.
	PartExpression

	// PartLinkedMessage represents a @key.path linked message reference.
	PartLinkedMessage
)

// String returns a string representation of the part kind.
//
// Returns string which is the human-readable name of the kind.
func (k PartKind) String() string {
	switch k {
	case PartLiteral:
		return "Literal"
	case PartExpression:
		return "Expression"
	case PartLinkedMessage:
		return "LinkedMessage"
	default:
		return "Unknown"
	}
}

// TemplatePart represents a segment of a parsed translation template.
// It supports ${expression} syntax and @linked.message references.
type TemplatePart struct {
	// Expression is the parsed AST for PartExpression, set at render time.
	// This field is not stored; it is filled in when rendering.
	Expression ast_domain.Expression

	// Literal holds the static text when Kind is PartLiteral.
	Literal string

	// ExprSource is the original expression text for PartExpression fields.
	// The expression is parsed when first needed and stored for later use.
	ExprSource string

	// LinkedKey is the message key path for linked messages (PartLinkedMessage).
	// For example, @common.greeting has LinkedKey "common.greeting".
	LinkedKey string

	// Kind indicates what type of content this template part holds.
	Kind PartKind
}

// templateParser holds state during template parsing.
type templateParser struct {
	// template is the raw input string being parsed.
	template string

	// buffer accumulates literal text between parsed expressions.
	buffer strings.Builder

	// parts holds the template segments built during parsing.
	parts []TemplatePart

	// errors collects parsing errors found during template processing.
	errors []string

	// position is the current character index within the template string.
	position int
}

// parse reads the template one character at a time and builds the result.
func (p *templateParser) parse() {
	for p.position < len(p.template) {
		switch {
		case p.isAt("${"):
			p.finaliseCurrentLiteral()
			p.parseExpression()
		case p.isAt("\\$") || p.isAt("\\@"):
			_ = p.buffer.WriteByte(p.template[p.position+1])
			p.position += 2
		case p.template[p.position] == '@':
			p.finaliseCurrentLiteral()
			p.parseLinkedMessage()
		default:
			_ = p.buffer.WriteByte(p.template[p.position])
			p.position++
		}
	}
	p.finaliseCurrentLiteral()
}

// isAt checks if the template has the given prefix at the current position.
//
// Takes prefix (string) which is the text to match at the current position.
//
// Returns bool which is true if the template matches the prefix at position.
func (p *templateParser) isAt(prefix string) bool {
	return strings.HasPrefix(p.template[p.position:], prefix)
}

// finaliseCurrentLiteral adds any buffered literal text as a template part.
func (p *templateParser) finaliseCurrentLiteral() {
	if p.buffer.Len() > 0 {
		p.parts = append(p.parts, TemplatePart{
			Expression: nil,
			Literal:    p.buffer.String(),
			ExprSource: "",
			LinkedKey:  "",
			Kind:       PartLiteral,
		})
		p.buffer.Reset()
	}
}

// parseExpression parses a ${...} expression and adds it to the template parts.
func (p *templateParser) parseExpression() {
	p.position += 2

	expressionStart := p.position
	endPosition, found := p.findMatchingBrace()
	if !found {
		p.errors = append(p.errors, "Unterminated expression: expected '}'")
		return
	}

	expressionSource := p.template[expressionStart:endPosition]
	p.parts = append(p.parts, TemplatePart{
		Expression: nil,
		Literal:    "",
		ExprSource: expressionSource,
		LinkedKey:  "",
		Kind:       PartExpression,
	})

	p.position = endPosition + 1
}

// findMatchingBrace finds the closing brace for an expression, handling
// nesting.
//
// Returns endPosition (int) which is the position of the matching closing brace,
// or -1 if not found.
// Returns found (bool) which indicates whether a matching brace was found.
func (p *templateParser) findMatchingBrace() (endPosition int, found bool) {
	braceLevel := 1
	for i := p.position; i < len(p.template); i++ {
		switch p.template[i] {
		case '{':
			braceLevel++
		case '}':
			braceLevel--
			if braceLevel == 0 {
				return i, true
			}
		}
	}
	return -1, false
}

// parseLinkedMessage parses a linked message reference in @key.path format.
func (p *templateParser) parseLinkedMessage() {
	p.position++

	keyStart := p.position
	p.scanIdentifier()

	if p.position == keyStart {
		p.errors = append(p.errors, "Expected identifier after '@'")
		return
	}

	for p.position < len(p.template) && p.template[p.position] == '.' {
		dotPosition := p.position
		p.position++

		identStart := p.position
		p.scanIdentifier()

		if p.position == identStart {
			p.errors = append(p.errors, "Expected identifier after '.' in linked message")
			p.position = dotPosition
			break
		}
	}

	linkedKey := p.template[keyStart:p.position]
	p.parts = append(p.parts, TemplatePart{
		Expression: nil,
		Literal:    "",
		ExprSource: "",
		LinkedKey:  linkedKey,
		Kind:       PartLinkedMessage,
	})
}

// scanIdentifier moves position forward past a valid identifier.
func (p *templateParser) scanIdentifier() {
	if p.position >= len(p.template) {
		return
	}

	character := rune(p.template[p.position])
	if !unicode.IsLetter(character) && character != '_' {
		return
	}
	p.position++

	for p.position < len(p.template) {
		character = rune(p.template[p.position])
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) && character != '_' {
			break
		}
		p.position++
	}
}

// ParseTemplate parses a translation template string into its parts.
// It handles ${expression} for value insertion, @key.path for linked
// messages, and \$ and \@ for escape sequences.
//
// Takes template (string) which is the translation template to parse.
//
// Returns []TemplatePart which contains the parsed template parts.
// Returns []string which contains any error messages, or nil on success.
func ParseTemplate(template string) ([]TemplatePart, []string) {
	if len(template) == 0 {
		return nil, nil
	}

	p := &templateParser{
		parts:    nil,
		errors:   nil,
		buffer:   strings.Builder{},
		template: template,
		position: 0,
	}
	p.parse()

	if len(p.errors) > 0 {
		return nil, p.errors
	}
	return p.parts, nil
}

// ExtractExpressions returns all expression sources found in a template string.
//
// Takes template (string) which contains the template text to parse.
//
// Returns []string which contains the source text of each expression found.
func ExtractExpressions(template string) []string {
	parts, _ := ParseTemplate(template)
	var exprs []string
	for _, part := range parts {
		if part.Kind == PartExpression {
			exprs = append(exprs, part.ExprSource)
		}
	}
	return exprs
}

// extractLinkedKeys returns all linked message keys from a template.
//
// Takes template (string) which contains the message template to parse.
//
// Returns []string which contains the keys of all linked messages found.
func extractLinkedKeys(template string) []string {
	parts, _ := ParseTemplate(template)
	var keys []string
	for _, part := range parts {
		if part.Kind == PartLinkedMessage {
			keys = append(keys, part.LinkedKey)
		}
	}
	return keys
}
