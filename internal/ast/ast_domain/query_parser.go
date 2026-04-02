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

package ast_domain

// Parses CSS selector strings into structured selector sets for querying AST nodes with support for complex selectors.
// Implements pooled QueryParser instances that handle element selectors, classes, IDs, attributes, pseudo-classes, and combinators.

import (
	"slices"
	"strings"
	"sync"
)

// queryParserPool pools QueryParser instances to reduce allocations.
var queryParserPool = sync.Pool{
	New: func() any {
		return &QueryParser{}
	},
}

// SelectorSet represents a list of selector groups split by commas, such as
// "div, p > span".
type SelectorSet []SelectorGroup

// SelectorGroup represents a chain of selectors joined by combinators,
// such as "div > p".
type SelectorGroup []ComplexSelector

// ComplexSelector represents a simple selector with a combinator before it.
// The combinator defines the relationship between this selector and the one
// that comes before it in the chain.
type ComplexSelector struct {
	// Combinator specifies how two selectors relate: " " (descendant), ">" (child),
	// "+" (adjacent sibling), or "~" (general sibling).
	Combinator string

	// Simple is the selector that matches a single element.
	Simple SimpleSelector

	// Location specifies where this selector appears in the source file.
	Location Location
}

// SimpleSelector represents a CSS selector without combinators, such as
// "div.card[disabled]". It forms the basic building block for matching single
// elements before combinators like descendant or child are applied.
type SimpleSelector struct {
	// Tag is the element name to match (e.g. "div", "span", or "*" for any).
	Tag string

	// ID is the element identifier to match, used in #myid selectors.
	ID string

	// Classes lists CSS class names to match; elements must have all listed classes.
	Classes []string

	// Attributes holds the attribute selectors that elements must match.
	Attributes []AttributeSelector

	// PseudoClasses holds the pseudo-class selectors that must all match.
	PseudoClasses []PseudoClassSelector

	// Location specifies where the selector appears in the source code.
	Location Location
}

// AttributeSelector represents an attribute part of a selector, such as
// `[href^="https"]`.
type AttributeSelector struct {
	// Name is the attribute name to match against.
	Name string

	// Operator specifies how to match the attribute value: "=" for exact match,
	// "~=" for word match, "^=" for prefix, "$=" for suffix, or "*=" for contains.
	Operator string

	// Value is the attribute value to compare against.
	Value string

	// CaseInsensitive enables matching that ignores letter case when true.
	CaseInsensitive bool

	// Location specifies where this selector appears in the source file.
	Location Location
}

// PseudoClassSelector represents a CSS pseudo-class, such as :first-child or
// :not(.special).
type PseudoClassSelector struct {
	// SubSelector holds the selector that the pseudo-class operates on.
	SubSelector *SimpleSelector

	// Type is the pseudo-class name, such as "hover", "first-child", or "not".
	Type string

	// Value holds the argument for nth-position pseudo-classes like :nth-child(2n+1).
	Value string

	// Location is the position of this selector in the source text.
	Location Location
}

// QueryParser holds the state for parsing a selector string.
type QueryParser struct {
	// l is the lexer that splits the input query into tokens.
	l *QueryLexer

	// sourcePath is the file path used when creating diagnostic messages.
	sourcePath string

	// diagnostics holds parse errors found during parsing.
	diagnostics []*Diagnostic

	// current is the token the parser is currently examining.
	current QueryToken

	// peek holds the next token for lookahead parsing.
	peek QueryToken
}

// NewQueryParser creates a parser for the given lexer.
// The returned parser should be released with Release when done.
//
// Takes l (*QueryLexer) which provides the token stream to parse.
// Takes sourcePath (string) which identifies the source file for error
// messages.
//
// Returns *QueryParser which is ready to parse the token stream.
func NewQueryParser(l *QueryLexer, sourcePath string) *QueryParser {
	p, ok := queryParserPool.Get().(*QueryParser)
	if !ok {
		p = &QueryParser{}
	}
	p.l = l
	p.sourcePath = sourcePath
	p.diagnostics = p.diagnostics[:0]
	p.current = QueryToken{}
	p.peek = QueryToken{}
	p.nextToken()
	p.nextToken()
	return p
}

// Release returns the parser to the pool so it can be reused.
func (p *QueryParser) Release() {
	p.l = nil
	p.sourcePath = ""
	p.diagnostics = p.diagnostics[:0]
	queryParserPool.Put(p)
}

// Diagnostics returns any errors encountered during parsing.
//
// Returns []*Diagnostic which contains all parse errors found.
func (p *QueryParser) Diagnostics() []*Diagnostic {
	return p.diagnostics
}

// Parse reads a selector string and returns the parsed groups.
//
// Returns SelectorSet which holds the parsed selector groups.
func (p *QueryParser) Parse() SelectorSet {
	var set SelectorSet

	for p.current.Type == TokenComma {
		p.addDiagnostic("Selector cannot start with a comma.", p.current)
		p.nextToken()
	}

	group := p.parseSelectorGroup()
	if len(group) > 0 {
		set = append(set, group)
	}

	for p.current.Type == TokenComma {
		p.nextToken()
		p.skipWhitespace()
		if p.current.Type == TokenEOF {
			break
		}
		group := p.parseSelectorGroup()
		if len(group) > 0 {
			set = append(set, group)
		}
	}

	return set
}

// nextToken moves the parser forward to the next token in the stream.
func (p *QueryParser) nextToken() {
	p.current = p.peek
	p.peek = p.l.NextToken()
}

// parseSelectorGroup parses a single selector chain (e.g. "div > p").
//
// Returns SelectorGroup which holds the parsed selectors, or nil if parsing
// fails.
func (p *QueryParser) parseSelectorGroup() SelectorGroup {
	var group SelectorGroup
	p.skipWhitespace()

	if !p.parseFirstSelector(&group) {
		return nil
	}

	for {
		wasWhitespace := p.skipWhitespace()

		if p.current.Type == TokenEOF || p.current.Type == TokenComma {
			break
		}

		if !p.parseNextSelectorWithCombinator(&group, wasWhitespace) {
			break
		}
	}

	return group
}

// parseFirstSelector parses the first selector in a group.
//
// Takes group (*SelectorGroup) which receives the parsed selector.
//
// Returns bool which is true when a valid selector was parsed.
func (p *QueryParser) parseFirstSelector(group *SelectorGroup) bool {
	if !p.isStartOfSimpleSelector() {
		if p.current.Type != TokenEOF && p.current.Type != TokenComma {
			p.addDiagnostic("Expected a selector, but found '"+p.current.Literal+"'", p.current)
		}
		return false
	}

	*group = append(*group, ComplexSelector{
		Combinator: "",
		Simple:     p.parseSimpleSelector(),
		Location:   p.current.Location,
	})
	return true
}

// isExplicitCombinator checks if the current token is an explicit combinator.
//
// Returns bool which is true when the token is a combinator, plus, or tilde.
func (p *QueryParser) isExplicitCombinator() bool {
	return p.current.Type == TokenCombinator || p.current.Type == TokenPlus || p.current.Type == TokenTilde
}

// parseNextSelectorWithCombinator parses a selector with its preceding
// combinator.
//
// Takes group (*SelectorGroup) which receives the parsed selector.
// Takes wasWhitespace (bool) which shows if whitespace came before this token.
//
// Returns bool which is true if a selector was parsed, false otherwise.
func (p *QueryParser) parseNextSelectorWithCombinator(group *SelectorGroup, wasWhitespace bool) bool {
	combinator := p.parseCombinator(wasWhitespace)
	if combinator == "" {
		if p.current.Type != TokenEOF && p.current.Type != TokenComma {
			p.addDiagnostic("Unexpected token '"+p.current.Literal+"' after a selector.", p.current)
		}
		return false
	}

	if !p.isStartOfSimpleSelector() {
		p.addDiagnostic("Expected a selector after combinator '"+combinator+"'", p.current)
		return false
	}

	*group = append(*group, ComplexSelector{
		Combinator: combinator,
		Simple:     p.parseSimpleSelector(),
		Location:   p.current.Location,
	})
	return true
}

// parseCombinator extracts the combinator token from the input.
//
// Takes wasWhitespace (bool) which shows if whitespace came before this token.
//
// Returns string which is the combinator symbol, or empty if none found.
func (p *QueryParser) parseCombinator(wasWhitespace bool) string {
	if p.isExplicitCombinator() {
		combinator := p.current.Literal
		p.nextToken()
		p.skipWhitespace()
		return combinator
	}

	if wasWhitespace && p.isStartOfSimpleSelector() {
		return " "
	}

	return ""
}

// isStartOfSimpleSelector checks if the current token can start a simple
// selector.
//
// Returns bool which is true when the token type is a valid selector start.
func (p *QueryParser) isStartOfSimpleSelector() bool {
	switch p.current.Type {
	case TokenIdent, TokenStar, TokenHash, TokenDot, TokenLBracket, TokenColon:
		return true
	default:
		return false
	}
}

// skipWhitespace consumes all consecutive whitespace tokens.
//
// Returns bool which is true if any whitespace was skipped.
func (p *QueryParser) skipWhitespace() bool {
	skipped := false
	for p.current.Type == TokenWhitespace {
		p.nextToken()
		skipped = true
	}
	return skipped
}

// parseSimpleSelector parses a selector that has no combinators.
//
// Returns SimpleSelector which holds the parsed tag, ID, classes, attributes,
// and pseudo-classes.
func (p *QueryParser) parseSimpleSelector() SimpleSelector {
	ss := SimpleSelector{Tag: "", ID: "", Classes: nil, Attributes: nil, PseudoClasses: nil, Location: p.current.Location}

	if p.current.Type == TokenIdent || p.current.Type == TokenStar {
		ss.Tag = p.current.Literal
		p.nextToken()
	}

	for {
		switch p.current.Type {
		case TokenHash:
			p.nextToken()
			if p.expectCurrent(TokenIdent, "expected identifier after #") {
				ss.ID = p.current.Literal
				p.nextToken()
			}
		case TokenDot:
			p.nextToken()
			if p.expectCurrent(TokenIdent, "expected identifier after .") {
				ss.Classes = append(ss.Classes, p.current.Literal)
				p.nextToken()
			}
		case TokenLBracket:
			ss.Attributes = append(ss.Attributes, p.parseAttributeSelector())
		case TokenColon:
			ss.PseudoClasses = append(ss.PseudoClasses, p.parsePseudoClassSelector())
		default:
			return ss
		}
	}
}

// parsePseudoClassSelector parses pseudo-classes such as :first-child or
// :not(.foo).
//
// Returns PseudoClassSelector which holds the parsed pseudo-class, including
// its type, value, and any nested selector.
func (p *QueryParser) parsePseudoClassSelector() PseudoClassSelector {
	ps := PseudoClassSelector{SubSelector: nil, Type: "", Value: "", Location: p.current.Location}
	p.nextToken()

	if !p.expectCurrent(TokenIdent, "expected identifier for pseudo-class name") {
		return ps
	}
	ps.Type = p.current.Literal
	p.nextToken()

	if p.current.Type == TokenLParen {
		p.nextToken()
		p.skipWhitespace()

		switch ps.Type {
		case "not":
			if p.current.Type == TokenRParen {
				p.addDiagnostic("expected a selector inside :not(), but it was empty", p.current)
				break
			}
			ps.SubSelector = new(p.parseSimpleSelector())
		case "nth-child", "nth-of-type", "nth-last-child", "nth-last-of-type":
			ps.Value = p.parseNthValue()
		default:
			p.addDiagnostic(
				"unsupported functional pseudo-class '"+ps.Type+"'",
				QueryToken{Literal: ps.Type, Location: ps.Location, Type: TokenIdent},
			)
			for p.current.Type != TokenRParen && p.current.Type != TokenEOF {
				p.nextToken()
			}
		}

		p.expectCurrent(TokenRParen, "expected ')' to close pseudo-class function")
		p.nextToken()
	}

	return ps
}

// parseNthValue parses the value part of nth-child and similar
// pseudo-classes.
//
// Returns string which is the parsed value with whitespace removed.
func (p *QueryParser) parseNthValue() string {
	var valBuilder strings.Builder
	for p.current.Type != TokenRParen && p.current.Type != TokenEOF {
		if p.current.Type == TokenWhitespace {
			p.nextToken()
			continue
		}
		valBuilder.WriteString(p.current.Literal)
		p.nextToken()
	}
	return valBuilder.String()
}

// parseAttributeSelector parses an attribute selector such as [href^="https"].
//
// Returns AttributeSelector which holds the parsed attribute name, operator,
// value, and case sensitivity flag.
func (p *QueryParser) parseAttributeSelector() AttributeSelector {
	as := AttributeSelector{Name: "", Operator: "", Value: "", CaseInsensitive: false, Location: p.current.Location}
	p.nextToken()
	p.skipWhitespace()

	if !p.expectCurrent(TokenIdent, "expected attribute name") {
		for p.current.Type != TokenRBracket && p.current.Type != TokenEOF {
			p.nextToken()
		}
		if p.current.Type == TokenRBracket {
			p.nextToken()
		}
		return as
	}
	as.Name = p.current.Literal
	p.nextToken()
	p.skipWhitespace()

	if isAttributeOperator(p.current.Type) {
		as.Operator = p.current.Literal
		p.nextToken()
		p.skipWhitespace()
		if p.expectCurrentOneOf(
			[]QueryTokenType{TokenString, TokenIdent},
			"expected string or identifier for attribute value",
		) {
			as.Value = p.current.Literal
			p.nextToken()
			p.skipWhitespace()
		}
	}

	if p.current.Type == TokenIdent && strings.EqualFold(p.current.Literal, "i") {
		as.CaseInsensitive = true
		p.nextToken()
		p.skipWhitespace()
	}

	p.expectCurrent(TokenRBracket, "expected ']' to close attribute selector")
	p.nextToken()

	return as
}

// addDiagnostic records a parsing error with its location.
//
// Takes message (string) which describes what went wrong.
// Takes queryToken (QueryToken) which provides the position and text for
// the error.
func (p *QueryParser) addDiagnostic(message string, queryToken QueryToken) {
	d := NewDiagnosticWithCode(Error, message, queryToken.Literal, CodeCSSParseError, queryToken.Location, p.sourcePath)
	p.diagnostics = append(p.diagnostics, d)
}

// expectCurrent checks if the current token is of a specific type and adds a
// diagnostic if not.
//
// Takes t (QueryTokenType) which specifies the expected token type.
// Takes message (string) which provides the diagnostic message if the check fails.
//
// Returns bool which is true if the current token matches the expected type.
func (p *QueryParser) expectCurrent(t QueryTokenType, message string) bool {
	if p.current.Type != t {
		p.addDiagnostic(message, p.current)
		return false
	}
	return true
}

// expectCurrentOneOf checks if the current token matches one of the given types.
//
// Takes types ([]QueryTokenType) which lists the allowed token types.
// Takes message (string) which is the error message to add if no type matches.
//
// Returns bool which is true if the current token matches any of the types.
func (p *QueryParser) expectCurrentOneOf(types []QueryTokenType, message string) bool {
	if slices.Contains(types, p.current.Type) {
		return true
	}
	p.addDiagnostic(message, p.current)
	return false
}

// isAttributeOperator checks if a token type is a valid attribute operator.
//
// Takes t (QueryTokenType) which is the token type to check.
//
// Returns bool which is true if the token is an attribute operator.
func isAttributeOperator(t QueryTokenType) bool {
	switch t {
	case TokenEquals, TokenIncludes, TokenDashMatch, TokenPrefix, TokenSuffix, TokenContains:
		return true
	default:
		return false
	}
}
