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

package querier_domain

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// maxDirectiveValueDepth bounds nested list-literal recursion so a deeply nested
	// `[[[...]]]` value cannot overflow the stack with a fatal, non-recoverable error.
	maxDirectiveValueDepth = 256
)

// callArgs is the parsed contents of a directive call: a list of positional values
// followed by any number of keyword arguments.
//
// Multi-positional is supported; keyword arguments whose key matches a positional slot
// are resolved to that slot during validation.
type callArgs struct {
	// positionals holds the parsed positional arguments in source order.
	positionals []parsedPositional

	// keywordArguments holds the parsed keyword arguments in source order.
	keywordArguments []parsedKeywordArgument

	// openSpan covers the opening parenthesis of the call.
	openSpan querier_dto.TextSpan

	// closeSpan covers the closing parenthesis of the call.
	closeSpan querier_dto.TextSpan

	// closed reports whether the closing parenthesis was consumed.
	closed bool
}

// parsedPositional is one positional argument with its value and the span covering the
// value tokens for diagnostics.
type parsedPositional struct {
	// value is the parsed positional value.
	value parsedValue

	// span covers the value tokens for diagnostics.
	span querier_dto.TextSpan
}

// parsedKeywordArgument is one keyword argument with its key, value, and the spans
// covering the key, value, and the whole key-value pair respectively.
type parsedKeywordArgument struct {
	// key is the keyword argument name.
	key string

	// value is the parsed keyword argument value.
	value parsedValue

	// span covers the whole key-value pair.
	span querier_dto.TextSpan

	// keySpan covers the key token.
	keySpan querier_dto.TextSpan

	// valueSpan covers the value tokens.
	valueSpan querier_dto.TextSpan
}

// parsedValue carries the raw value text plus optional list-shape information.
//
// The raw field is always set; the list fields are set only when the value was parsed
// from a `[a, b, c]` literal.
type parsedValue struct {
	// raw is the value text, always set.
	raw string

	// asList holds the list elements when the value was a list literal.
	asList []string

	// listSpan covers the list literal when the value was a list.
	listSpan querier_dto.TextSpan

	// isList reports whether the value was parsed from a list literal.
	isList bool
}

// parseCallArgs reads the parenthesised argument list of a directive call.
//
// The opening paren must be the next token; on success the closing paren is consumed and
// args.closed is set to true.
//
// Takes lexer (*directiveLexer) which supplies the call tokens.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
// Takes directiveName (string) which names the directive for messages.
//
// Returns *callArgs which holds the parsed arguments, or nil when the opening paren is
// missing.
// Returns []querier_dto.SourceError which holds any diagnostics produced.
func parseCallArgs(lexer *directiveLexer, errorBuilder querier_dto.ErrorBuilder, directiveName string) (*callArgs, []querier_dto.SourceError) {
	var diagnostics []querier_dto.SourceError
	openToken := lexer.next()
	if openToken.kind != tokenLParen {
		diagnostics = append(diagnostics, errorBuilder.DirectiveSyntax(openToken.span, fmt.Sprintf("expected '(' after %s", directiveName)))
		return nil, diagnostics
	}
	args := &callArgs{openSpan: openToken.span}

	if next := lexer.peek(); next.kind == tokenRParen {
		args.closeSpan = lexer.next().span
		args.closed = true
		return args, diagnostics
	}

	for {
		tok := lexer.peek()
		if tok.kind == tokenEOF {
			diagnostics = append(diagnostics, errorBuilder.Unclosed(tok.span, directiveName))
			return args, diagnostics
		}
		if tok.kind == tokenRParen {
			args.closeSpan = lexer.next().span
			args.closed = true
			return args, diagnostics
		}

		argDiags := parseSingleArg(lexer, args, errorBuilder)
		diagnostics = append(diagnostics, argDiags...)

		sep := lexer.peek()
		if sep.kind == tokenComma {
			lexer.next()
			continue
		}
		if sep.kind == tokenRParen {
			args.closeSpan = lexer.next().span
			args.closed = true
			return args, diagnostics
		}
		if sep.kind == tokenEOF {
			diagnostics = append(diagnostics, errorBuilder.Unclosed(sep.span, directiveName))
			return args, diagnostics
		}
		diagnostics = append(diagnostics, errorBuilder.DirectiveSyntax(sep.span, fmt.Sprintf("expected ',' or ')', got %q", sep.lexeme)))
		lexer.next()
	}
}

// parseSingleArg parses one argument in a directive call.
//
// Positional arguments must precede any keyword argument; once a keyword argument is
// seen, any subsequent positional is reported as a syntax error.
//
// Takes lexer (*directiveLexer) which supplies the argument tokens.
// Takes args (*callArgs) which accumulates the parsed argument.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
//
// Returns []querier_dto.SourceError which holds any diagnostics produced.
func parseSingleArg(lexer *directiveLexer, args *callArgs, errorBuilder querier_dto.ErrorBuilder) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	keyOrValue := lexer.peek()

	if keyOrValue.kind == tokenIdent && lexer.peekN(1).kind == tokenColon {
		keyToken := lexer.next()
		lexer.next()
		value, valueSpan, valueDiags := parseValue(lexer, errorBuilder, 0)
		diagnostics = append(diagnostics, valueDiags...)
		args.keywordArguments = append(args.keywordArguments, parsedKeywordArgument{
			key:       keyToken.lexeme,
			value:     value,
			span:      mergeSpan(keyToken.span, valueSpan),
			keySpan:   keyToken.span,
			valueSpan: valueSpan,
		})
		return diagnostics
	}

	if len(args.keywordArguments) > 0 {
		diagnostics = append(diagnostics, errorBuilder.DirectiveSyntax(keyOrValue.span, "positional arguments must precede keyword arguments; later arguments must be 'key: value'"))
		_, _, valueDiags := parseValue(lexer, errorBuilder, 0)
		diagnostics = append(diagnostics, valueDiags...)
		return diagnostics
	}

	value, valueSpan, valueDiags := parseValue(lexer, errorBuilder, 0)
	diagnostics = append(diagnostics, valueDiags...)
	args.positionals = append(args.positionals, parsedPositional{value: value, span: valueSpan})
	return diagnostics
}

// parseValue reads one value token (string, number, identifier, or list literal).
//
// Dotted identifier chains are joined into a single raw value so `piko.foo.bar` survives
// intact. The depth argument tracks list-literal nesting so adversarial input cannot
// overflow the stack.
//
// Takes lexer (*directiveLexer) which supplies the value tokens.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
// Takes depth (int) which is the current list-literal nesting depth.
//
// Returns parsedValue which is the parsed value.
// Returns querier_dto.TextSpan which covers the value tokens.
// Returns []querier_dto.SourceError which holds any diagnostics produced.
func parseValue(lexer *directiveLexer, errorBuilder querier_dto.ErrorBuilder, depth int) (parsedValue, querier_dto.TextSpan, []querier_dto.SourceError) {
	var diagnostics []querier_dto.SourceError
	tok := lexer.peek()
	if depth >= maxDirectiveValueDepth {
		lexer.next()
		diagnostics = append(diagnostics, errorBuilder.DirectiveSyntax(tok.span, "value nesting too deep"))
		return parsedValue{raw: tok.lexeme}, tok.span, diagnostics
	}
	switch tok.kind {
	case tokenLBracket:
		return parseListValue(lexer, errorBuilder, depth)
	case tokenString:
		lexer.next()
		return parsedValue{raw: tok.lexeme}, tok.span, diagnostics
	case tokenNumber, tokenIdent:
		startTok := lexer.next()
		var raw strings.Builder
		raw.WriteString(startTok.lexeme)
		span := startTok.span
		for lexer.peek().kind == tokenDot {
			lexer.next()
			tail := lexer.next()
			if tail.kind != tokenIdent && tail.kind != tokenNumber {
				diagnostics = append(diagnostics, errorBuilder.DirectiveSyntax(tail.span, "expected identifier after '.'"))
				break
			}
			raw.WriteByte('.')
			raw.WriteString(tail.lexeme)
			span = mergeSpan(span, tail.span)
		}
		return parsedValue{raw: raw.String()}, span, diagnostics
	default:
		lexer.next()
		diagnostics = append(diagnostics, errorBuilder.DirectiveSyntax(tok.span, fmt.Sprintf("unexpected token %q in value position", tok.lexeme)))
		return parsedValue{raw: tok.lexeme}, tok.span, diagnostics
	}
}

// parseListValue reads a `[a, b, c]` list literal.
//
// Unterminated lists and mismatched closers (e.g. `[a, b)`) reset the result so
// downstream validation does not see a partial list; the caller still receives the value
// with isList=false so the value-shape diagnostic at the validator level keeps producing
// the expected Q-coded message.
//
// Takes lexer (*directiveLexer) which supplies the list tokens.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
// Takes depth (int) which is the current list-literal nesting depth.
//
// Returns parsedValue which is the parsed list value.
// Returns querier_dto.TextSpan which covers the list literal.
// Returns []querier_dto.SourceError which holds any diagnostics produced.
func parseListValue(lexer *directiveLexer, errorBuilder querier_dto.ErrorBuilder, depth int) (parsedValue, querier_dto.TextSpan, []querier_dto.SourceError) {
	var diagnostics []querier_dto.SourceError
	openTok := lexer.next()
	value := parsedValue{isList: true}

	if lexer.peek().kind == tokenRBracket {
		closeTok := lexer.next()
		value.listSpan = mergeSpan(openTok.span, closeTok.span)
		return value, value.listSpan, diagnostics
	}
	for {
		item, _, itemDiags := parseValue(lexer, errorBuilder, depth+1)
		diagnostics = append(diagnostics, itemDiags...)
		if item.isList {
			diagnostics = append(diagnostics, errorBuilder.InvalidListLiteral(item.listSpan, "nested list"))
			value.isList = false
			value.asList = nil
			value.listSpan = mergeSpan(openTok.span, drainEnclosingList(lexer))
			return value, value.listSpan, diagnostics
		}
		value.asList = append(value.asList, item.raw)
		sep := lexer.peek()
		switch sep.kind {
		case tokenComma:
			lexer.next()
			continue
		case tokenRBracket:
			closeTok := lexer.next()
			value.listSpan = mergeSpan(openTok.span, closeTok.span)
			return value, value.listSpan, diagnostics
		case tokenEOF:
			diagnostics = append(diagnostics, errorBuilder.Unclosed(sep.span, "list"))
			value.isList = false
			value.asList = nil
			value.listSpan = mergeSpan(openTok.span, sep.span)
			return value, value.listSpan, diagnostics
		case tokenIdent, tokenNumber, tokenString, tokenLParen, tokenRParen,
			tokenLBracket, tokenColon, tokenDot, tokenSigil, tokenInvalid:
		}

		diagnostics = append(diagnostics, errorBuilder.InvalidListLiteral(sep.span, sep.lexeme))
		value.isList = false
		value.asList = nil
		value.listSpan = mergeSpan(openTok.span, sep.span)
		consumeUntilListBoundary(lexer)
		if lexer.peek().kind == tokenRBracket {
			lexer.next()
		}
		return value, value.listSpan, diagnostics
	}
}

// drainEnclosingList consumes the remainder of the list literal open at depth one,
// balancing nested brackets.
//
// It returns the span of the closing bracket, or the last token seen at end-of-input. It
// recovers after a nested-list element is rejected so the error does not cascade into the
// tokens that follow the outer list.
//
// Takes lexer (*directiveLexer) which is positioned just after the rejected element.
//
// Returns querier_dto.TextSpan which is the span of the closing bracket or final token.
func drainEnclosingList(lexer *directiveLexer) querier_dto.TextSpan {
	depth := 1
	lastSpan := lexer.peek().span
	for depth > 0 {
		tok := lexer.next()
		lastSpan = tok.span
		switch tok.kind {
		case tokenLBracket:
			depth++
		case tokenRBracket:
			depth--
		case tokenEOF:
			return lastSpan
		default:
		}
	}
	return lastSpan
}

// consumeUntilListBoundary advances the lexer through unrecognised tokens until it finds
// a list separator, terminator, or end-of-input.
//
// It is used as a recovery hook after parseListValue rejects a bad literal so subsequent
// calls keep producing useful diagnostics rather than cascading errors.
//
// Takes lexer (*directiveLexer) which is advanced to the next list boundary.
func consumeUntilListBoundary(lexer *directiveLexer) {
	for {
		peek := lexer.peek()
		if peek.kind == tokenEOF || peek.kind == tokenRBracket || peek.kind == tokenComma || peek.kind == tokenRParen {
			return
		}
		lexer.next()
	}
}
