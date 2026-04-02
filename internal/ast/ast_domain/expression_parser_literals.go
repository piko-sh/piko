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

// Provides parsing methods for all literal types including numbers, strings,
// booleans, dates, times, and composite literals. Implements parser methods for
// identifiers, numeric literals, temporal types, template literals, arrays, and
// objects with validation and error reporting.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"piko.sh/piko/wdk/maths"
)

// parseIdentifier parses the current token as an identifier expression.
//
// Returns Expression which is the parsed identifier node.
// Returns []*Diagnostic which is always nil for this parser.
func (p *ExpressionParser) parseIdentifier(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	tokenValue := p.tokenValue()
	id := &Identifier{
		GoAnnotations:    nil,
		Name:             tokenValue,
		RelativeLocation: token.Location,
		SourceLength:     len(tokenValue),
	}
	p.advanceLexerToken()
	return id, nil
}

// parseNumericLiteral parses the current token as a number.
//
// Returns Expression which is an IntegerLiteral or FloatLiteral on success.
// Returns []*Diagnostic which contains an error when the token is not valid.
func (p *ExpressionParser) parseNumericLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	tokenValue := p.tokenValue()
	p.advanceLexerToken()
	if i, err := strconv.ParseInt(tokenValue, 10, 64); err == nil {
		return &IntegerLiteral{
			Value:            i,
			RelativeLocation: token.Location,
			GoAnnotations:    nil,
			SourceLength:     len(tokenValue),
		}, nil
	}
	if f, err := strconv.ParseFloat(tokenValue, 64); err == nil {
		return &FloatLiteral{
			Value:            f,
			RelativeLocation: token.Location,
			GoAnnotations:    nil,
			SourceLength:     len(tokenValue),
		}, nil
	}
	diagnostic := NewDiagnosticWithCode(Error, fmt.Sprintf("Invalid number literal: %s", tokenValue), tokenValue, CodeInvalidNumberLiteral, token.Location, p.sourcePath)
	return nil, []*Diagnostic{diagnostic}
}

// parseDecimalLiteral parses a decimal literal token into an expression.
//
// Returns Expression which is the parsed decimal literal.
// Returns []*Diagnostic which is always nil.
func (p *ExpressionParser) parseDecimalLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	value := p.tokenValue()
	p.advanceLexerToken()
	return &DecimalLiteral{
		Value:            value,
		RelativeLocation: token.Location,
		GoAnnotations:    nil,
		SourceLength:     len(value) + 1,
	}, nil
}

// parseBigIntLiteral parses a big integer literal from the current token.
//
// Returns Expression which is the parsed BigIntLiteral node.
// Returns []*Diagnostic which is nil on success.
func (p *ExpressionParser) parseBigIntLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	value := p.tokenValue()
	p.advanceLexerToken()
	return &BigIntLiteral{
		Value:            value,
		RelativeLocation: token.Location,
		GoAnnotations:    nil,
		SourceLength:     len(value) + 1,
	}, nil
}

// parseRuneLiteral parses a rune literal token and returns its expression.
//
// Returns Expression which is the parsed rune literal, or nil on error.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseRuneLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	value := p.tokenValue()
	p.advanceLexerToken()

	wrappedContent := `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	unquoted, err := strconv.Unquote(wrappedContent)
	if err != nil {
		message := fmt.Sprintf("Invalid escape sequence in rune literal r'%s'", value)
		diagnostic := NewDiagnosticWithCode(Error, message, value, CodeInvalidEscapeSequence, token.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}

	if utf8.RuneCountInString(unquoted) != 1 {
		message := fmt.Sprintf("Invalid rune literal r'%s': must contain exactly one character, but found %d", value, utf8.RuneCountInString(unquoted))
		diagnostic := NewDiagnosticWithCode(Error, message, value, CodeInvalidRuneLiteral, token.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}

	r, _ := utf8.DecodeRuneInString(unquoted)
	return &RuneLiteral{
		Value:            r,
		RelativeLocation: token.Location,
		GoAnnotations:    nil,
		SourceLength:     len(value) + RuneLiteralPrefixLength,
	}, nil
}

// parseDateTimeLiteral parses an RFC3339 datetime literal from the token
// stream.
//
// Returns Expression which is the parsed DateTimeLiteral on success.
// Returns []*Diagnostic which contains an error when the datetime format is
// invalid.
func (p *ExpressionParser) parseDateTimeLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	value := p.tokenValue()
	p.advanceLexerToken()

	if _, err := time.Parse(time.RFC3339, value); err != nil {
		message := fmt.Sprintf("Invalid datetime format (expected RFC3339 'YYYY-MM-DDTHH:mm:ssZ'): %v", err)
		diagnostic := NewDiagnosticWithCode(Error, message, value, CodeInvalidTemporalFormat, token.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}

	return &DateTimeLiteral{
		Value:            value,
		RelativeLocation: token.Location,
		GoAnnotations:    nil,
		SourceLength:     len(value) + DateTimeLiteralPrefixLength,
	}, nil
}

// parseTimeBasedLiteral parses and checks date or time literals.
//
// Takes format (string) which specifies the expected time format.
// Takes formatDesc (string) which describes the format for error messages.
// Takes prefixLength (int) which specifies the prefix length for the literal.
// Takes constructor (func(...)) which builds the Expression from valid input.
//
// Returns Expression which is the parsed literal, or nil if parsing fails.
// Returns []*Diagnostic which contains any errors found during parsing.
func (p *ExpressionParser) parseTimeBasedLiteral(
	format string,
	formatDesc string,
	prefixLength int,
	constructor func(value string, location Location, prefixLen int) Expression,
) (Expression, []*Diagnostic) {
	token := p.currentToken
	value := p.tokenValue()
	p.advanceLexerToken()

	if _, err := time.ParseInLocation(format, value, time.UTC); err != nil {
		message := fmt.Sprintf("Invalid %s format (expected '%s'): %v", formatDesc, format, err)
		diagnostic := NewDiagnosticWithCode(Error, message, value, CodeInvalidTemporalFormat, token.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}

	return constructor(value, token.Location, prefixLength), nil
}

// parseDateLiteral parses a date literal in YYYY-MM-DD format.
//
// Returns Expression which is the parsed date literal.
// Returns []*Diagnostic which contains any parse errors.
func (p *ExpressionParser) parseDateLiteral(_ context.Context) (Expression, []*Diagnostic) {
	return p.parseTimeBasedLiteral("2006-01-02", "date 'YYYY-MM-DD'", DateLiteralPrefixLength,
		func(value string, location Location, prefixLen int) Expression {
			return &DateLiteral{
				Value:            value,
				RelativeLocation: location,
				GoAnnotations:    nil,
				SourceLength:     len(value) + prefixLen,
			}
		})
}

// parseTimeLiteral parses a time value in HH:mm:ss format.
//
// Returns Expression which is the parsed time value.
// Returns []*Diagnostic which contains any parsing errors.
func (p *ExpressionParser) parseTimeLiteral(_ context.Context) (Expression, []*Diagnostic) {
	return p.parseTimeBasedLiteral("15:04:05", "time 'HH:mm:ss'", TimeLiteralPrefixLength,
		func(value string, location Location, prefixLen int) Expression {
			return &TimeLiteral{
				Value:            value,
				RelativeLocation: location,
				GoAnnotations:    nil,
				SourceLength:     len(value) + prefixLen,
			}
		})
}

// parseDurationLiteral parses a duration literal token into an expression.
//
// Returns Expression which is the parsed duration literal, or nil on failure.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseDurationLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	value := p.tokenValue()
	p.advanceLexerToken()

	if _, err := time.ParseDuration(value); err != nil {
		message := fmt.Sprintf("Invalid duration format: %v", err)
		diagnostic := NewDiagnosticWithCode(Error, message, value, CodeInvalidTemporalFormat, token.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}

	return &DurationLiteral{
		Value:            value,
		RelativeLocation: token.Location,
		GoAnnotations:    nil,
		SourceLength:     len(value) + DurationLiteralPrefixLength,
	}, nil
}

// parseStringLiteral parses a quoted string token into a StringLiteral node.
//
// Returns Expression which is the parsed string literal, or nil on error.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseStringLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	tokenValue := p.tokenValue()
	p.advanceLexerToken()
	unquoted, err := strconv.Unquote(tokenValue)
	if err == nil {
		return &StringLiteral{
			Value:            unquoted,
			RelativeLocation: token.Location,
			GoAnnotations:    nil,
			SourceLength:     len(tokenValue),
		}, nil
	}
	if len(tokenValue) >= 2 && tokenValue[0] == '\'' && tokenValue[len(tokenValue)-1] == '\'' {
		unquoted = tokenValue[1 : len(tokenValue)-1]
		return &StringLiteral{
			Value:            unquoted,
			RelativeLocation: token.Location,
			GoAnnotations:    nil,
			SourceLength:     len(tokenValue),
		}, nil
	}

	diagnostic := NewDiagnosticWithCode(Error, fmt.Sprintf("Invalid string literal: %s", err), tokenValue, CodeInvalidStringLiteral, token.Location, p.sourcePath)
	return nil, []*Diagnostic{diagnostic}
}

// parseBooleanLiteral parses a boolean literal token and returns the result.
//
// Returns Expression which is the parsed boolean literal.
// Returns []*Diagnostic which is always nil.
func (p *ExpressionParser) parseBooleanLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	tokenValue := p.tokenValue()
	value := p.currentToken.Type == tokenKeywordTrue
	p.advanceLexerToken()
	return &BooleanLiteral{
		Value:            value,
		RelativeLocation: token.Location,
		GoAnnotations:    nil,
		SourceLength:     len(tokenValue),
	}, nil
}

// parseNilLiteral parses a nil literal from the current token position.
//
// Returns Expression which is the parsed nil literal node.
// Returns []*Diagnostic which is always nil for this parser.
func (p *ExpressionParser) parseNilLiteral(_ context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	p.advanceLexerToken()
	return &NilLiteral{
		RelativeLocation: token.Location,
		GoAnnotations:    nil,
		SourceLength:     NilLiteralLength,
	}, nil
}

// parseLinkedMessage parses a linked message reference such as
// @common.greeting. The @ symbol is followed by a path expression, which is an
// identifier with optional member access using dot notation.
//
// Returns Expression which is the parsed linked message expression.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseLinkedMessage(ctx context.Context) (Expression, []*Diagnostic) {
	atTok := p.currentToken
	p.advanceLexerToken()

	if p.currentToken.Type != tokenIdent {
		diagnostic := NewDiagnosticWithCode(Error, "Expected identifier after '@'", p.tokenValue(), CodeMissingIdentifier, atTok.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}

	path, diagnostics := p.parseIdentifier(ctx)
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}

	for p.currentToken.Type == tokenDot {
		dotTok := p.currentToken
		p.advanceLexerToken()

		if p.currentToken.Type != tokenIdent {
			diagnostic := NewDiagnosticWithCode(Error, "Expected identifier after '.'", p.tokenValue(), CodeMissingIdentifier, dotTok.Location, p.sourcePath)
			return nil, []*Diagnostic{diagnostic}
		}

		propTok := p.currentToken
		propVal := p.tokenValue()
		p.advanceLexerToken()

		property := &Identifier{
			GoAnnotations:    nil,
			Name:             propVal,
			RelativeLocation: propTok.Location,
			SourceLength:     len(propVal),
		}

		path = &MemberExpression{
			Base:             path,
			Property:         property,
			GoAnnotations:    nil,
			Optional:         false,
			Computed:         false,
			RelativeLocation: dotTok.Location,
			SourceLength:     path.GetSourceLength() + 1 + len(propVal),
		}
	}

	sourceLength := 1 + path.GetSourceLength()

	return &LinkedMessageExpression{
		GoAnnotations:    nil,
		Path:             path,
		RelativeLocation: atTok.Location,
		SourceLength:     sourceLength,
	}, nil
}

// parseTemplateLiteralExpression parses a template literal token into an
// expression.
//
// Returns Expression which is the parsed template literal.
// Returns []*Diagnostic which contains any parsing errors.
func (p *ExpressionParser) parseTemplateLiteralExpression(ctx context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	tokenValue := p.tokenValue()
	p.advanceLexerToken()
	tl, diagnostics := parseTemplateLiteral(ctx, tokenValue, token.Location, p.sourcePath)
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}
	return tl, nil
}

// templateLiteralParser encapsulates the state for parsing within a `${...}`
// block.
type templateLiteralParser struct {
	// content holds the raw text of the template literal being parsed.
	content string

	// sourcePath is the file path used in error messages.
	sourcePath string

	// buffer collects literal characters between interpolations.
	buffer strings.Builder

	// parts holds the parsed sections of the template literal.
	parts []TemplateLiteralPart

	// diagnostics holds errors or warnings found during template parsing.
	diagnostics []*Diagnostic

	// parentLocation is the source location of the parent node, used to work out
	// offsets.
	parentLocation Location

	// position is the current position in the template string.
	position int
}

// parse reads the template content and finds both literal parts and
// interpolations.
//
// Returns *TemplateLiteral which contains the parsed template structure.
// Returns []*Diagnostic which holds any errors found during parsing.
func (p *templateLiteralParser) parse(ctx context.Context) (*TemplateLiteral, []*Diagnostic) {
	for p.position < len(p.content) {
		if p.isAt("${") {
			p.finaliseCurrentLiteral()
			if !p.parseInterpolation(ctx) {
				break
			}
		} else if p.isAt("\\") {
			p.handleEscapeSequence()
		} else {
			p.advance(1)
		}
	}
	p.finaliseCurrentLiteral()
	return &TemplateLiteral{GoAnnotations: nil, Parts: p.parts, RelativeLocation: p.parentLocation, SourceLength: len(p.content) + 2}, p.diagnostics
}

// finaliseCurrentLiteral adds any text in the buffer as a literal part and
// clears the buffer.
func (p *templateLiteralParser) finaliseCurrentLiteral() {
	if p.buffer.Len() > 0 {
		startOffset := p.position - p.buffer.Len()
		location := p.calculateLocation(startOffset)
		p.parts = append(p.parts, TemplateLiteralPart{
			Expression:       nil,
			Literal:          p.buffer.String(),
			IsLiteral:        true,
			RelativeLocation: location,
		})
		p.buffer.Reset()
	}
}

// parseInterpolation parses a ${...} interpolation expression in the template.
//
// Returns bool which is true when parsing succeeded, or false on error.
func (p *templateLiteralParser) parseInterpolation(ctx context.Context) bool {
	interpolationStartOffset := p.position
	p.position += 2
	expressionStartOffset := p.position

	endOffset, found := p.findMatchingBrace(expressionStartOffset)
	if !found {
		diagStartLocation := p.calculateLocation(interpolationStartOffset)
		p.diagnostics = append(p.diagnostics, NewDiagnosticWithCode(
			Error, "Unterminated expression in template literal: expected '}'",
			p.content[interpolationStartOffset:], CodeUnterminatedInterpolation,
			diagStartLocation, p.sourcePath,
		))
		return false
	}

	expressionString := p.content[expressionStartOffset:endOffset]
	expressionParser := NewExpressionParser(ctx, expressionString, p.sourcePath)
	parsedExpr, diagnostics := expressionParser.ParseExpression(ctx)
	expressionParser.Release()

	if len(diagnostics) > 0 {
		p.adjustAndAddDiags(diagnostics, expressionStartOffset)
	}

	if HasErrors(diagnostics) || parsedExpr == nil {
		return false
	}

	partLocation := p.calculateLocation(interpolationStartOffset)
	p.parts = append(p.parts, TemplateLiteralPart{
		Expression:       parsedExpr,
		Literal:          "",
		IsLiteral:        false,
		RelativeLocation: partLocation,
	})
	p.position = endOffset + 1
	return true
}

// findMatchingBrace finds the closing brace that matches an opening brace.
//
// Takes startOffset (int) which is the position to start searching from.
//
// Returns endOffset (int) which is the position of the matching closing brace.
// Returns found (bool) which shows whether a matching brace was found.
func (p *templateLiteralParser) findMatchingBrace(startOffset int) (endOffset int, found bool) {
	braceLevel := 1
	for i := startOffset; i < len(p.content); i++ {
		switch p.content[i] {
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

// adjustAndAddDiags updates diagnostic positions to match their location within
// the template literal and adds them to the parser's list.
//
// Takes diagnostics ([]*Diagnostic) which are the diagnostics to adjust and add.
// Takes expressionStartOffset (int) which is the byte offset where the expression
// starts within the template.
func (p *templateLiteralParser) adjustAndAddDiags(diagnostics []*Diagnostic, expressionStartOffset int) {
	for _, d := range diagnostics {
		expressionBaseLocation := p.calculateLocation(expressionStartOffset)
		if d.Location.Line == 1 {
			d.Location.Line = expressionBaseLocation.Line
			d.Location.Column = expressionBaseLocation.Column + d.Location.Column - 1
		} else {
			d.Location.Line = expressionBaseLocation.Line + d.Location.Line - 1
		}
		p.diagnostics = append(p.diagnostics, d)
	}
}

// handleEscapeSequence processes a backslash escape in the template literal.
func (p *templateLiteralParser) handleEscapeSequence() {
	if p.position+1 < len(p.content) {
		escapedChar := p.content[p.position+1]
		switch escapedChar {
		case '`', '$', '\\':
			_ = p.buffer.WriteByte(escapedChar)
			p.position += 2
			return
		}
	}
	_ = p.buffer.WriteByte('\\')
	p.position++
}

// isAt checks whether the remaining content starts with the given string.
//
// Takes s (string) which is the prefix to check for.
//
// Returns bool which is true if the content at the current position starts
// with s.
func (p *templateLiteralParser) isAt(s string) bool {
	return strings.HasPrefix(p.content[p.position:], s)
}

// advance moves the parser position forward by n characters, copying the
// skipped content to the output buffer.
//
// Takes n (int) which specifies the number of characters to advance.
func (p *templateLiteralParser) advance(n int) {
	p.buffer.WriteString(p.content[p.position : p.position+n])
	p.position += n
}

// calculateLocation works out the line and column position for a given byte
// offset within the template literal content.
//
// Takes offset (int) which is the byte position within the content.
//
// Returns Location which holds the calculated line and column numbers.
func (p *templateLiteralParser) calculateLocation(offset int) Location {
	line := p.parentLocation.Line
	column := p.parentLocation.Column + 1
	contentUpToOffset := p.content[:offset]

	if lastNewlineIndex := strings.LastIndex(contentUpToOffset, "\n"); lastNewlineIndex != -1 {
		line += strings.Count(contentUpToOffset, "\n")
		column = utf8.RuneCountInString(contentUpToOffset[lastNewlineIndex+1:]) + 1
	} else {
		column += utf8.RuneCountInString(contentUpToOffset)
	}
	return Location{Line: line, Column: column, Offset: 0}
}

// parseTemplateLiteral parses a raw template literal string that includes its
// backticks.
//
// Takes raw (string) which is the template literal with backticks.
// Takes parentLocation (Location) which is the parent location for
// error reporting.
// Takes sourcePath (string) which is the path to the source file.
//
// Returns *TemplateLiteral which is the parsed template structure.
// Returns []*Diagnostic which contains any parse errors or warnings.
func parseTemplateLiteral(ctx context.Context, raw string, parentLocation Location, sourcePath string) (*TemplateLiteral, []*Diagnostic) {
	if len(raw) < 2 || raw[0] != '`' || raw[len(raw)-1] != '`' {
		diagnostic := NewDiagnosticWithCode(Error, "Invalid template literal format: expected backticks", raw, CodeInvalidTemplateLiteral, parentLocation, sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}
	p := &templateLiteralParser{
		content:        raw[1 : len(raw)-1],
		sourcePath:     sourcePath,
		buffer:         strings.Builder{},
		parts:          nil,
		diagnostics:    nil,
		parentLocation: parentLocation,
		position:       0,
	}
	return p.parse(ctx)
}

// toString converts a value to its string form.
//
// Takes value (any) which is the value to convert.
//
// Returns string which is the string form of the value, or an empty string if
// value is nil.
func toString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case rune:
		return string(v)
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case maths.BigInt:
		s, err := v.String()
		if err != nil {
			return "ERR"
		}
		return s
	case maths.Decimal:
		s, err := v.String()
		if err != nil {
			return "ERR"
		}
		return s
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprint(value)
	}
}
