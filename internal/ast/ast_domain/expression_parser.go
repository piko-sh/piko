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

// Implements the expression parser that converts token sequences from the lexer into expression AST nodes.
// Uses recursive descent parsing with operator precedence, pooled parser instances, and configurable feature flags for different parsing contexts.

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"unicode"
	"unicode/utf8"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// RuneLiteralPrefixLength is the length of a rune literal prefix and quotes.
	RuneLiteralPrefixLength = 3

	// DateTimeLiteralPrefixLength is the length of the dt'' prefix and quotes for
	// datetime literals.
	DateTimeLiteralPrefixLength = 4

	// DateLiteralPrefixLength is the length of the d'' prefix and quotes for
	// date literals.
	DateLiteralPrefixLength = 3

	// TimeLiteralPrefixLength is the length of the t'' prefix for time literals.
	TimeLiteralPrefixLength = 3

	// DurationLiteralPrefixLength is the length of the "du''" prefix and quotes
	// in duration literals.
	DurationLiteralPrefixLength = 4

	// NilLiteralLength is the length of the nil keyword in characters.
	NilLiteralLength = 3

	// MaxExpressionDepth is the maximum nesting depth allowed for expressions.
	// This limit stops stack overflow errors from deeply nested expressions
	// such as ((((x)))).
	MaxExpressionDepth = 256

	// asciiTableSize is the number of entries in the ASCII lookup tables.
	asciiTableSize = 128

	// prefixParseFunctionTableSize is the size of the prefix parse
	// function lookup table. It must be larger than the highest token
	// type value (currently 21).
	prefixParseFunctionTableSize = 32
)

// prefixParseFunction defines a function type for parsing prefix expressions.
type prefixParseFunction func(*ExpressionParser, context.Context) (Expression, []*Diagnostic)

var (
	// prefixParseFunctions is an array dispatch table that maps token types to
	// prefix parsing functions.
	//
	// Array lookup is faster than map lookup (~10x). Zero values (nil) mean
	// no prefix parse function exists for that token type.
	prefixParseFunctions [prefixParseFunctionTableSize]prefixParseFunction

	// parserPool provides pooled parser instances to reduce allocations.
	parserPool = sync.Pool{
		New: func() any {
			return &ExpressionParser{
				tokens:       make([]lexerToken, 0, 64),
				sourcePath:   "",
				input:        "",
				currentToken: lexerToken{},
				tokenIndex:   0,
				depth:        0,
			}
		},
	}

	// expressionCache caches parsed expressions to avoid re-parsing identical
	// strings. This improves performance for repeated parsing of the same
	// expressions.
	expressionCache sync.Map

	// identStartTable maps ASCII bytes to whether they can start an identifier,
	// using a lookup table for O(1) character class checking. Includes '$' as valid
	// per JavaScript identifier rules (used for $event).
	identStartTable = [asciiTableSize]bool{
		'_': true, '$': true,
		'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true,
		'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true,
		'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true,
		'v': true, 'w': true, 'x': true, 'y': true, 'z': true,
		'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true,
		'H': true, 'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true,
		'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true, 'U': true,
		'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true,
	}

	// identCharTable maps ASCII bytes to true if they can appear in an
	// identifier (not in the first position).
	// Includes '$' as valid per JavaScript identifier rules.
	identCharTable = [asciiTableSize]bool{
		'_': true, '$': true,
		'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true,
		'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true,
		'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true,
		'v': true, 'w': true, 'x': true, 'y': true, 'z': true,
		'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true,
		'H': true, 'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true,
		'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true, 'U': true,
		'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true,
		'0': true, '1': true, '2': true, '3': true, '4': true,
		'5': true, '6': true, '7': true, '8': true, '9': true,
	}
)

// ExpressionParser converts a sequence of tokens from the lexer into an AST.
// Field order is set to reduce GC pointer bitmap size.
type ExpressionParser struct {
	// tokens holds the lexer tokens. The parser owns this slice and
	// reuses it across Reset calls.
	tokens []lexerToken

	// sourcePath is the file path shown in error messages.
	sourcePath string

	// input is the expression string being parsed; stored for zero-copy token values.
	input string

	// currentToken holds the token being processed at the current position.
	currentToken lexerToken

	// tokenIndex is the current position in the token slice; -1 means before
	// the first token.
	tokenIndex int

	// depth tracks the current nesting level of expressions to prevent stack overflow.
	depth int
}

// cachedExpression holds a parsed expression along with its diagnostic messages.
type cachedExpression struct {
	// expression holds the parsed expression; nil if parsing failed.
	expression Expression

	// diagnostics holds the diagnostic messages from parsing.
	diagnostics []*Diagnostic
}

// NewExpressionParser creates and sets up a new parser for the given
// expression string.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes expression (string) which is the expression to parse.
// Takes sourcePath (string) which identifies the source file for error
// messages.
//
// Returns *ExpressionParser which is the set up parser ready for use.
func NewExpressionParser(ctx context.Context, expression string, sourcePath string) *ExpressionParser {
	p, ok := parserPool.Get().(*ExpressionParser)
	if !ok {
		var l logger_domain.Logger
		ctx, l = logger_domain.From(ctx, log)
		l.Error("parserPool returned unexpected type, allocating new instance")
		p = &ExpressionParser{}
	}
	p.sourcePath = sourcePath
	p.input = expression
	p.tokens = lexInto(ctx, expression, p.tokens)
	p.currentToken = lexerToken{}
	p.tokenIndex = -1
	p.depth = 0
	p.advanceLexerToken()
	return p
}

// Release returns the parser to the pool for reuse. Call this when you have
// finished with the parser to reduce memory allocations.
func (p *ExpressionParser) Release() {
	p.input = ""
	p.sourcePath = ""
	parserPool.Put(p)
}

// Reset reinitialises the parser with a new expression string and source path.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes expression (string) which is the new expression to parse.
// Takes sourcePath (string) which identifies the source for error messages.
func (p *ExpressionParser) Reset(ctx context.Context, expression string, sourcePath string) {
	p.input = expression
	p.tokens = lexInto(ctx, expression, p.tokens)
	p.sourcePath = sourcePath
	p.tokenIndex = -1
	p.depth = 0
	p.advanceLexerToken()
}

// forLoopVarInfo holds token info for for-loop variables without allocating
// Identifiers, allowing speculative parsing without wasted allocations.
type forLoopVarInfo struct {
	// name is the loop variable identifier.
	name string

	// location is the source position of the loop variable name.
	location Location
}

// ParseExpression is the main entry point for parsing. It handles the special
// case of for-loop syntax before falling back to standard precedence-based
// parsing.
//
// Takes ctx (context.Context) which carries the request-scoped logger and
// cancellation signal.
//
// Returns Expression which is the parsed expression node.
// Returns []*Diagnostic which contains any parse errors or warnings.
func (p *ExpressionParser) ParseExpression(ctx context.Context) (Expression, []*Diagnostic) {
	start := p.tokenIndex
	startLocation := p.currentToken.Location

	if expression, diagnostics, handled := p.tryParseForLoop(ctx, start, startLocation); handled {
		return expression, diagnostics
	}

	return p.parseAndValidate(ctx)
}

// tryParseForLoop attempts to parse a for-loop expression in the form
// `(var, var) in collection` or `var in collection`.
//
// Takes start (int) which marks the token position to return to on failure.
// Takes startLocation (Location) which specifies the source location for the
// resulting expression.
//
// Returns Expression which is the parsed for-loop expression on success.
// Returns []*Diagnostic which contains any parsing errors found.
// Returns bool which is true if a for-loop was parsed or failed, or false if
// the tokens do not form a for-loop expression.
func (p *ExpressionParser) tryParseForLoop(ctx context.Context, start int, startLocation Location) (Expression, []*Diagnostic, bool) {
	idxInfo, itemInfo, isLoop := p.parseForLoopVarSyntax()
	if !isLoop {
		return nil, nil, false
	}

	if p.currentToken.Type != tokenKeywordIn || p.tokenValue() != "in" {
		p.backtrack(start)
		return nil, nil, false
	}

	var idxVar *Identifier
	if idxInfo.name != "" {
		idxVar = &Identifier{GoAnnotations: nil, Name: idxInfo.name, RelativeLocation: idxInfo.location, SourceLength: len(idxInfo.name)}
	}
	itemVar := &Identifier{GoAnnotations: nil, Name: itemInfo.name, RelativeLocation: itemInfo.location, SourceLength: len(itemInfo.name)}

	expression, diagnostics := p.parseForInExpressionBody(ctx, idxVar, itemVar, startLocation)
	if len(diagnostics) > 0 {
		return nil, diagnostics, true
	}

	if endDiags := p.validateEndState(expression); len(endDiags) > 0 {
		return nil, endDiags, true
	}
	return expression, nil, true
}

// parseAndValidate parses an expression and checks the end state.
//
// Returns Expression which is the parsed expression, or nil on failure.
// Returns []*Diagnostic which holds any parsing or validation errors.
func (p *ExpressionParser) parseAndValidate(ctx context.Context) (Expression, []*Diagnostic) {
	expression, diagnostics := p.parseExpressionWithPrecedence(ctx, 0)
	if len(diagnostics) > 0 {
		if expression != nil {
			if endDiags := p.validateEndState(expression); len(endDiags) > 0 {
				diagnostics = append(diagnostics, endDiags...)
			}
			return expression, diagnostics
		}
		return nil, diagnostics
	}

	if endDiags := p.validateEndState(expression); len(endDiags) > 0 {
		return nil, endDiags
	}
	return expression, nil
}

// tokenValue returns the string value of the current token.
// This is a zero-copy operation that slices the original input.
//
// Returns string which is the token value at the current position.
func (p *ExpressionParser) tokenValue() string {
	return p.currentToken.getValue(p.input)
}

// parseExpressionWithPrecedence is the core of the Pratt parser. It handles
// prefix, infix, and postfix operators based on their precedence levels.
//
// Takes minPrec (int) which sets the lowest precedence level for operators
// that will be parsed.
//
// Returns Expression which is the parsed expression tree.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseExpressionWithPrecedence(ctx context.Context, minPrec int) (Expression, []*Diagnostic) {
	p.depth++
	defer func() { p.depth-- }()

	if p.depth > MaxExpressionDepth {
		diagnostic := NewDiagnosticWithCode(Error, "expression exceeds maximum nesting depth", "", CodeExpressionDepthExceeded, p.currentToken.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}

	left, diagnostics := p.parsePrefix(ctx)
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}

	return p.parseOperators(ctx, left, minPrec)
}

// parseOperators handles the operator parsing loop for postfix and infix
// operators.
//
// Takes left (Expression) which is the left-hand side of the expression.
// Takes minPrec (int) which is the minimum precedence level to parse.
//
// Returns Expression which is the fully parsed expression tree.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseOperators(ctx context.Context, left Expression, minPrec int) (Expression, []*Diagnostic) {
	for {
		result, diagnostics, done := p.tryParseOperator(ctx, left, minPrec)
		if done {
			return result, diagnostics
		}
		left = result
	}
}

// tryParseOperator attempts to parse the next operator based on the current
// token.
//
// Takes left (Expression) which is the left-hand side of the expression.
// Takes minPrec (int) which is the minimum precedence level to consider.
//
// Returns Expression which is the parsed expression, possibly unchanged.
// Returns []*Diagnostic which contains any parsing diagnostics.
// Returns bool which indicates whether parsing should stop (true means done).
func (p *ExpressionParser) tryParseOperator(ctx context.Context, left Expression, minPrec int) (Expression, []*Diagnostic, bool) {
	switch p.currentToken.Type {
	case tokenLParen, tokenDot, tokenOptionalDot, tokenLBracket, tokenOptionalBracket:
		return p.tryParsePostfixOp(ctx, left, minPrec)
	case tokenSymbol:
		return p.tryParseInfixOp(ctx, left, minPrec)
	default:
		return left, nil, true
	}
}

// tryParsePostfixOp parses postfix operators such as calls, member access,
// and indexing.
//
// Takes left (Expression) which is the expression to extend with operators.
// Takes minPrec (int) which is the minimum precedence level to parse.
//
// Returns Expression which is the result with any postfix operators applied.
// Returns []*Diagnostic which contains any parsing errors found.
// Returns bool which is true when parsing errors were found.
func (p *ExpressionParser) tryParsePostfixOp(ctx context.Context, left Expression, minPrec int) (Expression, []*Diagnostic, bool) {
	if minPrec >= precPostfix {
		return left, nil, true
	}
	result, diagnostics := p.parsePostfix(ctx, left)
	result, diagnostics = handlePartialExpressionResult(result, diagnostics)
	return result, diagnostics, len(diagnostics) > 0
}

// tryParseInfixOp parses an infix operator if one exists at the current
// position with enough precedence.
//
// Takes left (Expression) which is the left-hand operand already parsed.
// Takes minPrec (int) which is the minimum precedence level to consider.
//
// Returns Expression which is the combined expression if an operator was found.
// Returns []*Diagnostic which contains any parse errors found.
// Returns bool which is true when errors occurred.
func (p *ExpressionParser) tryParseInfixOp(ctx context.Context, left Expression, minPrec int) (Expression, []*Diagnostic, bool) {
	opString := p.tokenValue()
	prec := getPrecedence(opString)
	if prec <= 0 || prec < minPrec {
		return left, nil, true
	}
	result, diagnostics := p.parseInfix(ctx, left)
	result, diagnostics = handlePartialExpressionResult(result, diagnostics)
	return result, diagnostics, len(diagnostics) > 0
}

// parsePrefix parses an expression that starts a statement, such as a
// literal, identifier, or unary operator.
//
// Returns Expression which is the parsed result, or nil on error.
// Returns []*Diagnostic which contains any errors found during parsing.
func (p *ExpressionParser) parsePrefix(ctx context.Context) (Expression, []*Diagnostic) {
	token := p.currentToken
	tokenValue := p.tokenValue()

	switch token.Type {
	case tokenSymbol:
		if tokenValue == "!" || tokenValue == "-" || tokenValue == "~" {
			return p.parseUnary(ctx)
		}
	case tokenEOF:
		diagnostic := NewDiagnosticWithCode(Error, "Unexpected end of expression while parsing primary value", "", CodeUnexpectedEOF, token.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	case tokenError:
		diagnostic := NewDiagnosticWithCode(Error, "Lexer error: "+token.errorMessage, tokenValue, CodeLexerError, token.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	default:
	}

	if token.Type < prefixParseFunctionTableSize {
		if parseFunction := prefixParseFunctions[token.Type]; parseFunction != nil {
			return parseFunction(p, ctx)
		}
	}

	msg := fmt.Sprintf("Unexpected token in expression: found '%s' where a value was expected", tokenValue)
	diagnostic := NewDiagnosticWithCode(
		Error, msg, tokenValue, CodeUnexpectedToken, token.Location, p.sourcePath,
	)
	return nil, []*Diagnostic{diagnostic}
}

// parseInfix parses binary and ternary operators.
//
// Takes left (Expression) which is the left-hand side of the operator.
//
// Returns Expression which is the parsed binary or ternary expression.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseInfix(ctx context.Context, left Expression) (Expression, []*Diagnostic) {
	if p.tokenValue() == "?" {
		return p.parseTernaryExpression(ctx, left)
	}

	opToken := p.currentToken
	opVal := p.tokenValue()
	operator := BinaryOp(opVal)
	prec := getPrecedence(string(operator))
	p.advanceLexerToken()

	if p.currentToken.Type == tokenEOF {
		diagnostic := NewDiagnosticWithCode(Error, "Expected expression on the right side of the operator", string(operator), CodeMissingOperand, opToken.Location, p.sourcePath)
		return left, []*Diagnostic{diagnostic}
	}

	right, diagnostics := p.parseExpressionWithPrecedence(ctx, prec)
	if len(diagnostics) > 0 {
		return left, diagnostics
	}
	if right == nil {
		diagnostic := NewDiagnosticWithCode(Error, "Expected expression on the right side of the operator", string(operator), CodeMissingOperand, opToken.Location, p.sourcePath)
		return left, []*Diagnostic{diagnostic}
	}

	return &BinaryExpression{
		Left:             left,
		Operator:         operator,
		Right:            right,
		RelativeLocation: left.GetRelativeLocation(),
		GoAnnotations:    nil,
		SourceLength:     right.GetRelativeLocation().Offset + right.GetSourceLength() - left.GetRelativeLocation().Offset,
	}, nil
}

// parsePostfix handles operators that follow an expression, like function
// calls, member access, and index access.
//
// Takes left (Expression) which is the preceding expression to extend.
//
// Returns Expression which is the combined postfix expression.
// Returns []*Diagnostic which contains any parse errors encountered.
func (p *ExpressionParser) parsePostfix(ctx context.Context, left Expression) (Expression, []*Diagnostic) {
	switch p.currentToken.Type {
	case tokenLParen:
		return p.parseFunctionCall(ctx, left)
	case tokenDot, tokenOptionalDot:
		return p.parseMemberAccess(left)
	case tokenLBracket, tokenOptionalBracket:
		return p.parseIndexExpression(ctx, left)
	default:
		diagnostic := NewDiagnosticWithCode(Error, "Internal parser error: unexpected token in parsePostfix", p.tokenValue(), CodeInternalParserError, p.currentToken.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}
}

// parseUnary parses a unary expression with a prefix operator.
//
// Returns Expression which is the parsed unary expression, or nil on error.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseUnary(ctx context.Context) (Expression, []*Diagnostic) {
	opToken := p.currentToken
	opVal := p.tokenValue()
	operator := UnaryOp(opVal)
	p.advanceLexerToken()

	right, diagnostics := p.parseExpressionWithPrecedence(ctx, precPrefix)
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}
	if right == nil {
		diagnostic := NewDiagnosticWithCode(Error, "Expected expression after unary operator", string(operator), CodeMissingOperand, opToken.Location, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}
	return &UnaryExpression{
		Operator:         operator,
		Right:            right,
		RelativeLocation: opToken.Location,
		GoAnnotations:    nil,
		SourceLength:     right.GetRelativeLocation().Offset + right.GetSourceLength() - opToken.Location.Offset,
	}, nil
}

// parseMemberAccess parses a dot or optional chaining member access.
//
// Takes base (Expression) which is the expression to access a member from.
//
// Returns Expression which is the member access expression, or the base
// expression if parsing fails.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseMemberAccess(base Expression) (Expression, []*Diagnostic) {
	isOptional := p.currentToken.Type == tokenOptionalDot
	dotLocation := p.currentToken.Location
	p.advanceLexerToken()

	if p.currentToken.Type == tokenEOF {
		diagnostic := NewDiagnosticWithCode(Error, "Expected identifier after '.' or '?.'", "", CodeMissingIdentifier, dotLocation, p.sourcePath)
		return base, []*Diagnostic{diagnostic}
	}

	if p.currentToken.Type != tokenIdent {
		diagnostic := NewDiagnosticWithCode(Error, "Expected identifier after '.' or '?.'", p.tokenValue(), CodeMissingIdentifier, dotLocation, p.sourcePath)
		return base, []*Diagnostic{diagnostic}
	}
	propTok := p.currentToken
	propVal := p.tokenValue()
	prop := &Identifier{
		GoAnnotations:    nil,
		Name:             propVal,
		RelativeLocation: propTok.Location,
		SourceLength:     len(propVal),
	}
	p.advanceLexerToken()
	return &MemberExpression{
		Base:             base,
		Property:         prop,
		GoAnnotations:    nil,
		Optional:         isOptional,
		Computed:         false,
		RelativeLocation: base.GetRelativeLocation(),
		SourceLength:     propTok.Location.Offset + len(propVal) - base.GetRelativeLocation().Offset,
	}, nil
}

// parseIndexExpression parses an index access expression like a[i] or a?[i].
//
// Takes base (Expression) which is the expression being indexed.
//
// Returns Expression which is the parsed index expression, or the base if an
// error occurs.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseIndexExpression(ctx context.Context, base Expression) (Expression, []*Diagnostic) {
	isOptional := p.currentToken.Type == tokenOptionalBracket
	bracketLocation := p.currentToken.Location
	p.advanceLexerToken()

	if p.currentToken.Type == tokenEOF {
		diagnostic := NewDiagnosticWithCode(Error, "Incomplete index expression: expected index expression", "", CodeIncompleteConstruct, bracketLocation, p.sourcePath)
		return base, []*Diagnostic{diagnostic}
	}
	if p.currentToken.Type == tokenRBracket {
		diagnostic := NewDiagnosticWithCode(Error, "Empty index expression: expected index expression", "", CodeIncompleteConstruct, bracketLocation, p.sourcePath)
		p.advanceLexerToken()
		return base, []*Diagnostic{diagnostic}
	}

	index, diagnostics := p.parseExpressionWithPrecedence(ctx, 0)
	if len(diagnostics) > 0 {
		return base, diagnostics
	}

	if p.currentToken.Type != tokenRBracket {
		tokenValue := p.tokenValue()
		msg := fmt.Sprintf("Expected ']' in index expression, got '%s' instead", tokenValue)
		diagnostic := NewDiagnosticWithCode(
			Error, msg, tokenValue, CodeMissingClosingDelimiter, bracketLocation, p.sourcePath,
		)
		return base, []*Diagnostic{diagnostic}
	}
	closingBracket := p.currentToken
	closingVal := p.tokenValue()
	p.advanceLexerToken()
	return &IndexExpression{
		Base:             base,
		Index:            index,
		GoAnnotations:    nil,
		Optional:         isOptional,
		RelativeLocation: base.GetRelativeLocation(),
		SourceLength:     closingBracket.Location.Offset + len(closingVal) - base.GetRelativeLocation().Offset,
	}, nil
}

// parseForLoopVarSyntax parses for loop variable declarations.
//
// Returns idxInfo (forLoopVarInfo) which holds the index variable details, or
// empty if not present.
// Returns itemInfo (forLoopVarInfo) which holds the item variable details.
// Returns isLoop (bool) which is true if valid loop syntax was found.
func (p *ExpressionParser) parseForLoopVarSyntax() (idxInfo, itemInfo forLoopVarInfo, isLoop bool) {
	save := p.tokenIndex
	if p.currentToken.Type == tokenIdent {
		itemInfo = forLoopVarInfo{name: p.tokenValue(), location: p.currentToken.Location}
		p.advanceLexerToken()
		return forLoopVarInfo{}, itemInfo, true
	}
	if p.currentToken.Type == tokenLParen {
		p.advanceLexerToken()
		if p.currentToken.Type != tokenIdent {
			p.backtrack(save)
			return forLoopVarInfo{}, forLoopVarInfo{}, false
		}
		idxInfo = forLoopVarInfo{name: p.tokenValue(), location: p.currentToken.Location}
		p.advanceLexerToken()

		if p.currentToken.Type != tokenComma {
			p.backtrack(save)
			return forLoopVarInfo{}, forLoopVarInfo{}, false
		}
		p.advanceLexerToken()

		if p.currentToken.Type != tokenIdent {
			p.backtrack(save)
			return forLoopVarInfo{}, forLoopVarInfo{}, false
		}
		itemInfo = forLoopVarInfo{name: p.tokenValue(), location: p.currentToken.Location}
		p.advanceLexerToken()

		if p.currentToken.Type != tokenRParen {
			p.backtrack(save)
			return forLoopVarInfo{}, forLoopVarInfo{}, false
		}
		p.advanceLexerToken()
		return idxInfo, itemInfo, true
	}
	p.backtrack(save)
	return forLoopVarInfo{}, forLoopVarInfo{}, false
}

// parseForInExpressionBody parses the body of a for-in expression after the
// variable declarations.
//
// Takes idxVar (*Identifier) which is the optional index variable.
// Takes itemVar (*Identifier) which is the item variable for iteration.
// Takes expressionStartLocation (Location) which marks where the expression began.
//
// Returns Expression which is the parsed for-in expression.
// Returns []*Diagnostic which contains any parsing errors encountered.
func (p *ExpressionParser) parseForInExpressionBody(ctx context.Context, idxVar, itemVar *Identifier, expressionStartLocation Location) (Expression, []*Diagnostic) {
	inKeywordLocation := p.currentToken.Location
	p.advanceLexerToken()

	if p.currentToken.Type == tokenEOF {
		diagnostic := NewDiagnosticWithCode(Error, "Expected an expression for the collection after 'in'", "in", CodeMissingOperand, inKeywordLocation, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}

	coll, diagnostics := p.parseExpressionWithPrecedence(ctx, 0)
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}
	if coll == nil {
		diagnostic := NewDiagnosticWithCode(Error, "Expected an expression for the collection after 'in'", "in", CodeMissingOperand, inKeywordLocation, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}
	return &ForInExpression{
		IndexVariable:    idxVar,
		ItemVariable:     itemVar,
		Collection:       coll,
		GoAnnotations:    nil,
		RelativeLocation: expressionStartLocation,
		SourceLength:     coll.GetRelativeLocation().Offset + coll.GetSourceLength() - expressionStartLocation.Offset,
	}, nil
}

// parseParenthesisedExpression parses an expression inside parentheses.
//
// Returns Expression which is the parsed inner expression with its source
// location adjusted to include the parentheses.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseParenthesisedExpression(ctx context.Context) (Expression, []*Diagnostic) {
	openParenLocation := p.currentToken.Location
	p.advanceLexerToken()

	expression, diagnostics := p.parseExpressionWithPrecedence(ctx, 0)
	if len(diagnostics) > 0 {
		if expression != nil && p.currentToken.Type == tokenRParen {
			closeParenLocation := p.currentToken.Location
			p.advanceLexerToken()
			p.adjustExprForParentheses(expression, openParenLocation, closeParenLocation)
			return expression, diagnostics
		}
		return nil, diagnostics
	}
	if p.currentToken.Type != tokenRParen {
		tokenValue := p.tokenValue()
		msg := fmt.Sprintf("Expected ')' in parenthesised expression, got '%s' instead", tokenValue)
		diagnostic := NewDiagnosticWithCode(
			Error, msg, tokenValue, CodeMissingClosingDelimiter, openParenLocation, p.sourcePath,
		)
		return nil, []*Diagnostic{diagnostic}
	}

	closeParenLocation := p.currentToken.Location
	p.advanceLexerToken()

	p.adjustExprForParentheses(expression, openParenLocation, closeParenLocation)

	return expression, nil
}

// adjustExprForParentheses updates an expression's location and length to
// include its surrounding parentheses. This means expressions like
// !(x > 10) have the correct source length of 9, including the opening
// parenthesis.
//
// Takes expression (Expression) which is the expression to adjust.
// Takes openParenLocation (Location) which is the position of the opening
// parenthesis.
// Takes closeParenLocation (Location) which is the position of the closing
// parenthesis.
func (*ExpressionParser) adjustExprForParentheses(expression Expression, openParenLocation, closeParenLocation Location) {
	if expression == nil {
		return
	}

	newSourceLength := closeParenLocation.Offset + 1 - openParenLocation.Offset

	expression.SetLocation(openParenLocation, newSourceLength)
}

// parseObjectLiteral parses an object literal from the token stream.
//
// Returns Expression which is the parsed object literal, or nil on error.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseObjectLiteral(ctx context.Context) (Expression, []*Diagnostic) {
	braceLocation := p.currentToken.Location
	objectLiteral := &ObjectLiteral{
		Pairs:            make(map[string]Expression),
		RelativeLocation: braceLocation,
		GoAnnotations:    nil,
		SourceLength:     0,
	}
	p.advanceLexerToken()

	if p.currentToken.Type == tokenRBrace {
		closeBrace := p.currentToken
		closeVal := p.tokenValue()
		p.advanceLexerToken()
		objectLiteral.SourceLength = closeBrace.Location.Offset + len(closeVal) - braceLocation.Offset
		return objectLiteral, nil
	}

	for p.currentToken.Type != tokenRBrace && p.currentToken.Type != tokenEOF {
		key, value, diagnostics := p.parseObjectPair(ctx)
		if len(diagnostics) > 0 {
			return nil, diagnostics
		}
		objectLiteral.Pairs[key] = value

		if p.currentToken.Type == tokenRBrace {
			break
		}

		if p.currentToken.Type != tokenComma {
			tokenValue := p.tokenValue()
			msg := fmt.Sprintf("Expected ',' or '}' after value, but got '%s'", tokenValue)
			diagnostic := NewDiagnosticWithCode(
				Error, msg, tokenValue, CodeMissingSeparator, p.currentToken.Location, p.sourcePath,
			)
			return nil, []*Diagnostic{diagnostic}
		}
		p.advanceLexerToken()
	}

	if p.currentToken.Type != tokenRBrace {
		diagnostic := NewDiagnosticWithCode(Error, "Expected '}' to close object literal", "", CodeMissingClosingDelimiter, braceLocation, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}
	closeBrace := p.currentToken
	closeVal := p.tokenValue()
	p.advanceLexerToken()
	objectLiteral.SourceLength = closeBrace.Location.Offset + len(closeVal) - braceLocation.Offset
	return objectLiteral, nil
}

// parseObjectPair parses a single key-value pair within an object literal.
//
// Returns string which is the parsed key name.
// Returns Expression which is the parsed value expression.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseObjectPair(ctx context.Context) (string, Expression, []*Diagnostic) {
	key, keyDiags := p.parseObjectKey()
	if len(keyDiags) > 0 {
		return "", nil, keyDiags
	}

	if p.currentToken.Type != tokenColon {
		diagnostic := NewDiagnosticWithCode(Error, "Expected ':' after object key", p.tokenValue(), CodeMissingSeparator, p.currentToken.Location, p.sourcePath)
		return "", nil, []*Diagnostic{diagnostic}
	}
	p.advanceLexerToken()

	value, valDiags := p.parseExpressionWithPrecedence(ctx, 0)
	if len(valDiags) > 0 {
		return "", nil, valDiags
	}
	if value == nil {
		diagnostic := NewDiagnosticWithCode(Error, "Expected an expression for object value", p.tokenValue(), CodeMissingOperand, p.currentToken.Location, p.sourcePath)
		return "", nil, []*Diagnostic{diagnostic}
	}

	return key, value, nil
}

// parseObjectKey extracts the key from an object literal entry.
//
// Returns string which is the parsed key value.
// Returns []*Diagnostic which contains errors when the key is not a valid
// identifier or string.
func (p *ExpressionParser) parseObjectKey() (string, []*Diagnostic) {
	keyLocation := p.currentToken.Location
	tokenValue := p.tokenValue()
	switch p.currentToken.Type {
	case tokenIdent:
		p.advanceLexerToken()
		return tokenValue, nil
	case tokenString:
		unquoted, err := strconv.Unquote(tokenValue)
		if err == nil {
			p.advanceLexerToken()
			return unquoted, nil
		}
		if len(tokenValue) >= 2 && tokenValue[0] == '\'' && tokenValue[len(tokenValue)-1] == '\'' {
			unquoted = tokenValue[1 : len(tokenValue)-1]
			p.advanceLexerToken()
			return unquoted, nil
		}
		diagnostic := NewDiagnosticWithCode(Error, fmt.Sprintf("Invalid string literal for object key: %s", err), tokenValue, CodeInvalidStringLiteral, keyLocation, p.sourcePath)
		return "", []*Diagnostic{diagnostic}
	default:
		diagnostic := NewDiagnosticWithCode(Error, "Expected an identifier or string for object key", tokenValue, CodeUnexpectedToken, p.currentToken.Location, p.sourcePath)
		return "", []*Diagnostic{diagnostic}
	}
}

// parseArrayLiteral parses an array literal from the token stream.
//
// Returns Expression which is the parsed array literal, or nil on error.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseArrayLiteral(ctx context.Context) (Expression, []*Diagnostic) {
	bracketLocation := p.currentToken.Location
	arr := &ArrayLiteral{
		Elements:         []Expression{},
		RelativeLocation: bracketLocation,
		GoAnnotations:    nil,
		SourceLength:     0,
	}
	p.advanceLexerToken()

	if p.currentToken.Type == tokenRBracket {
		closeBracket := p.currentToken
		closeVal := p.tokenValue()
		p.advanceLexerToken()
		arr.SourceLength = closeBracket.Location.Offset + len(closeVal) - bracketLocation.Offset
		return arr, nil
	}

	firstElement, diagnostics := p.parseExpressionWithPrecedence(ctx, precLowest)
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}
	arr.Elements = append(arr.Elements, firstElement)

	for p.currentToken.Type == tokenComma {
		p.advanceLexerToken()
		if p.currentToken.Type == tokenRBracket {
			break
		}
		nextElement, diagnostics := p.parseExpressionWithPrecedence(ctx, precLowest)
		if len(diagnostics) > 0 {
			return nil, diagnostics
		}
		arr.Elements = append(arr.Elements, nextElement)
	}

	if p.currentToken.Type != tokenRBracket {
		tokenValue := p.tokenValue()
		diagnostic := NewDiagnosticWithCode(Error, fmt.Sprintf("Expected ']' to close array literal, but got '%s'", tokenValue), tokenValue, CodeMissingClosingDelimiter, bracketLocation, p.sourcePath)
		return nil, []*Diagnostic{diagnostic}
	}
	closeBracket := p.currentToken
	closeVal := p.tokenValue()
	p.advanceLexerToken()
	arr.SourceLength = closeBracket.Location.Offset + len(closeVal) - bracketLocation.Offset
	return arr, nil
}

// parseFunctionCall parses a function call expression from the token stream.
//
// Takes callee (Expression) which is the expression being called.
//
// Returns Expression which is the parsed call expression. Returns a partial
// result if parsing fails.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseFunctionCall(ctx context.Context, callee Expression) (Expression, []*Diagnostic) {
	leftParenLocation := p.currentToken.Location
	p.advanceLexerToken()
	var arguments []Expression

	if p.currentToken.Type == tokenEOF {
		diagnostic := NewDiagnosticWithCode(Error, "Incomplete function call: expected arguments or ')'", "", CodeIncompleteConstruct, leftParenLocation, p.sourcePath)
		return newPartialCallExpr(callee, arguments, leftParenLocation, leftParenLocation), []*Diagnostic{diagnostic}
	}

	if p.currentToken.Type == tokenRParen {
		return p.completeCallExpr(callee, arguments, leftParenLocation)
	}

	arguments, diagnostics := p.parseFirstArgument(ctx, arguments)
	if len(diagnostics) > 0 {
		return newPartialCallExpr(callee, arguments, leftParenLocation, leftParenLocation), diagnostics
	}

	arguments, diagnostics = p.parseRemainingArguments(ctx, arguments)
	if len(diagnostics) > 0 {
		return newPartialCallExpr(callee, arguments, leftParenLocation, leftParenLocation), diagnostics
	}

	if p.currentToken.Type != tokenRParen {
		tokenValue := p.tokenValue()
		msg := fmt.Sprintf("Expected ',' or ')' in function call arguments, but got '%s'", tokenValue)
		diagnostic := NewDiagnosticWithCode(
			Error, msg, tokenValue, CodeMissingSeparator, leftParenLocation, p.sourcePath,
		)
		return newPartialCallExpr(callee, arguments, leftParenLocation, leftParenLocation), []*Diagnostic{diagnostic}
	}

	return p.completeCallExpr(callee, arguments, leftParenLocation)
}

// parseFirstArgument parses the first argument in a function call.
//
// Takes arguments ([]Expression) which holds any arguments parsed so far.
//
// Returns []Expression which contains the arguments with the first one added.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseFirstArgument(ctx context.Context, arguments []Expression) ([]Expression, []*Diagnostic) {
	firstArgument, diagnostics := p.parseExpressionWithPrecedence(ctx, 0)
	if len(diagnostics) > 0 {
		if firstArgument != nil {
			arguments = append(arguments, firstArgument)
		}
		return arguments, diagnostics
	}
	return append(arguments, firstArgument), nil
}

// parseRemainingArguments parses more arguments after the first one in a
// function call.
//
// Takes arguments ([]Expression) which contains any arguments already parsed.
//
// Returns []Expression which contains all parsed arguments including the input.
// Returns []*Diagnostic which contains any parsing errors found.
func (p *ExpressionParser) parseRemainingArguments(ctx context.Context, arguments []Expression) ([]Expression, []*Diagnostic) {
	for p.currentToken.Type == tokenComma {
		p.advanceLexerToken()

		if p.currentToken.Type == tokenEOF {
			diagnostic := NewDiagnosticWithCode(Error, "Incomplete function call: expected argument after ','", "", CodeIncompleteConstruct, p.currentToken.Location, p.sourcePath)
			return arguments, []*Diagnostic{diagnostic}
		}
		if p.currentToken.Type == tokenRParen {
			diagnostic := NewDiagnosticWithCode(Error, "Unexpected ')' after ',' in function call", "", CodeIncompleteConstruct, p.currentToken.Location, p.sourcePath)
			return arguments, []*Diagnostic{diagnostic}
		}

		nextArg, diagnostics := p.parseExpressionWithPrecedence(ctx, 0)
		if len(diagnostics) > 0 {
			if nextArg != nil {
				arguments = append(arguments, nextArg)
			}
			return arguments, diagnostics
		}
		arguments = append(arguments, nextArg)
	}
	return arguments, nil
}

// completeCallExpr builds a CallExpression after parsing the
// closing bracket.
//
// Takes callee (Expression) which is the function or method being called.
// Takes arguments ([]Expression) which are the parsed argument expressions.
// Takes leftParenLocation (Location) which marks the opening bracket position.
//
// Returns *CallExpression which represents the complete function call node.
// Returns []*Diagnostic which is always nil here.
func (p *ExpressionParser) completeCallExpr(callee Expression, arguments []Expression, leftParenLocation Location) (*CallExpression, []*Diagnostic) {
	rparenTok := p.currentToken
	rparenVal := p.tokenValue()
	p.advanceLexerToken()
	return &CallExpression{
		Callee:           callee,
		GoAnnotations:    nil,
		Args:             arguments,
		RelativeLocation: callee.GetRelativeLocation(),
		LparenLocation:   leftParenLocation,
		RparenLocation:   rparenTok.Location,
		SourceLength:     rparenTok.Location.Offset + len(rparenVal) - callee.GetRelativeLocation().Offset,
	}, nil
}

// parseTernaryExpression parses a ternary conditional expression.
//
// Takes condition (Expression) which is the condition before the '?' token.
//
// Returns Expression which is the complete ternary expression, or the original
// condition if parsing fails.
// Returns []*Diagnostic which contains any parse errors found.
func (p *ExpressionParser) parseTernaryExpression(ctx context.Context, condition Expression) (Expression, []*Diagnostic) {
	p.advanceLexerToken()

	if p.currentToken.Type == tokenEOF {
		diagnostic := NewDiagnosticWithCode(Error, "Incomplete ternary expression: expected consequent expression after '?'", "", CodeIncompleteConstruct, p.currentToken.Location, p.sourcePath)
		return condition, []*Diagnostic{diagnostic}
	}

	consequent, diagnostics := p.parseExpressionWithPrecedence(ctx, precLowest)
	if len(diagnostics) > 0 {
		return condition, diagnostics
	}

	if p.currentToken.Type == tokenEOF {
		diagnostic := NewDiagnosticWithCode(Error, "Incomplete ternary expression: expected ':' after consequent", "", CodeMissingSeparator, p.currentToken.Location, p.sourcePath)
		return condition, []*Diagnostic{diagnostic}
	}
	if p.currentToken.Type != tokenColon {
		diagnostic := NewDiagnosticWithCode(Error, "Expected ':' for ternary operator", p.tokenValue(), CodeMissingSeparator, p.currentToken.Location, p.sourcePath)
		return condition, []*Diagnostic{diagnostic}
	}
	p.advanceLexerToken()

	if p.currentToken.Type == tokenEOF {
		diagnostic := NewDiagnosticWithCode(Error, "Incomplete ternary expression: expected alternate expression after ':'", "", CodeIncompleteConstruct, p.currentToken.Location, p.sourcePath)
		return condition, []*Diagnostic{diagnostic}
	}

	alternate, diagnostics := p.parseExpressionWithPrecedence(ctx, precTernary-1)
	if len(diagnostics) > 0 {
		return condition, diagnostics
	}
	if alternate == nil {
		diagnostic := NewDiagnosticWithCode(Error, "Failed to parse alternate expression in ternary operator", "", CodeIncompleteConstruct, p.currentToken.Location, p.sourcePath)
		return condition, []*Diagnostic{diagnostic}
	}
	return &TernaryExpression{
		Condition:        condition,
		Consequent:       consequent,
		Alternate:        alternate,
		RelativeLocation: condition.GetRelativeLocation(),
		GoAnnotations:    nil,
		SourceLength:     alternate.GetRelativeLocation().Offset + alternate.GetSourceLength() - condition.GetRelativeLocation().Offset,
	}, nil
}

// backtrack restores the parser to a previous token position.
//
// Takes position (int) which is the token index to return to.
func (p *ExpressionParser) backtrack(position int) {
	p.tokenIndex = position
	if p.tokenIndex < 0 {
		p.tokenIndex = -1
	}
	if p.tokenIndex < len(p.tokens) {
		p.currentToken = p.tokens[p.tokenIndex]
	} else {
		p.currentToken = lexerToken{offset: 0, length: 0, errorMessage: "", Location: Location{}, Type: tokenEOF}
	}
}

// advanceLexerToken moves to the next token in the token stream.
func (p *ExpressionParser) advanceLexerToken() {
	p.tokenIndex++
	if p.tokenIndex < len(p.tokens) {
		p.currentToken = p.tokens[p.tokenIndex]
	} else {
		p.currentToken = lexerToken{offset: 0, length: 0, errorMessage: "", Location: Location{}, Type: tokenEOF}
	}
}

// validateEndState checks the parser state after parsing an expression.
//
// Takes expression (Expression) which is the parsed expression to check
// against.
//
// Returns []*Diagnostic which contains any syntax errors found, or nil if
// the state is valid.
func (p *ExpressionParser) validateEndState(expression Expression) []*Diagnostic {
	if expression != nil && p.currentToken.Type == tokenComma {
		message := "Invalid syntax. A comma is not a valid operator here. If you are trying to write a for-loop, variables must be enclosed in parentheses, e.g., `(index, item) in collection`."
		diagnostic := NewDiagnosticWithCode(Error, message, ",", CodeUnexpectedComma, p.currentToken.Location, p.sourcePath)
		return []*Diagnostic{diagnostic}
	}
	if p.currentToken.Type != tokenEOF && p.currentToken.Type != tokenError {
		tokenValue := p.tokenValue()
		diagnostic := NewDiagnosticWithCode(Error, fmt.Sprintf("Unexpected tokens after expression: '%s'", tokenValue), tokenValue, CodeTrailingTokens, p.currentToken.Location, p.sourcePath)
		return []*Diagnostic{diagnostic}
	}
	if p.currentToken.Type == tokenError {
		diagnostic := NewDiagnosticWithCode(Error, "Lexer error: "+p.currentToken.errorMessage, p.tokenValue(), CodeLexerError, p.currentToken.Location, p.sourcePath)
		return []*Diagnostic{diagnostic}
	}
	return nil
}

// ParseExpressionCached parses an expression with caching.
//
// If the same expression string was parsed before, returns a clone of the
// cached result. This is the preferred API for production use where the same
// expressions may be parsed repeatedly.
//
// Takes expression (string) which is the expression text to parse.
// Takes sourcePath (string) which identifies the source file for
// diagnostics.
//
// Returns Expression which is the parsed expression, or nil if empty.
// Returns []*Diagnostic which contains any parse errors or warnings.
func ParseExpressionCached(ctx context.Context, expression, sourcePath string) (Expression, []*Diagnostic) {
	if expression == "" {
		return nil, nil
	}

	cached, ok := expressionCache.Load(expression)
	if !ok {
		return parseAndCacheExpression(ctx, expression, sourcePath)
	}

	ce, ok := cached.(cachedExpression)
	if !ok {
		expressionCache.Delete(expression)
		return parseAndCacheExpression(ctx, expression, sourcePath)
	}

	if ce.expression != nil {
		return ce.expression.Clone(), ce.diagnostics
	}
	return nil, ce.diagnostics
}

// ClearExpressionCache resets the expression cache to an empty state.
// Intended for testing.
func ClearExpressionCache() {
	expressionCache = sync.Map{}
}

// parseAndCacheExpression parses an expression and stores the result in a
// cache for later use.
//
// Takes expression (string) which is the expression text to parse.
// Takes sourcePath (string) which identifies the source file for
// diagnostics.
//
// Returns Expression which is the parsed expression, even if invalid.
// Returns []*Diagnostic which contains any parse errors or warnings.
func parseAndCacheExpression(ctx context.Context, expression, sourcePath string) (Expression, []*Diagnostic) {
	parser := NewExpressionParser(ctx, expression, sourcePath)
	parsed, diagnostics := parser.ParseExpression(ctx)
	parser.Release()

	expressionCache.Store(expression, cachedExpression{expression: parsed, diagnostics: diagnostics})

	return parsed, diagnostics
}

// handlePartialExpressionResult provides fault-tolerant error handling for
// expressions.
//
// If the expression is not nil, it returns both the expression and
// diagnostics. This is critical for LSP features like completion on
// incomplete expressions.
//
// Takes expression (Expression) which is the parsed expression,
// possibly partial.
// Takes diagnostics ([]*Diagnostic) which contains any parsing errors
// or warnings.
//
// Returns Expression which is the input expression, or nil if parsing
// failed.
// Returns []*Diagnostic which contains the diagnostics, or nil if
// empty.
func handlePartialExpressionResult(expression Expression, diagnostics []*Diagnostic) (Expression, []*Diagnostic) {
	if len(diagnostics) == 0 {
		return expression, nil
	}
	if expression != nil {
		return expression, diagnostics
	}
	return nil, diagnostics
}

// newPartialCallExpr creates a CallExpression from the given parts.
//
// Takes callee (Expression) which is the function or method being called.
// Takes arguments ([]Expression) which holds the arguments to pass.
// Takes leftParenLocation (Location) which marks the opening parenthesis position.
// Takes rightParenLocation (Location) which marks the closing
// parenthesis position.
//
// Returns *CallExpression which is the built call expression.
func newPartialCallExpr(callee Expression, arguments []Expression, leftParenLocation, rightParenLocation Location) *CallExpression {
	sourceLen := calculateCallExprSourceLength(callee, arguments, leftParenLocation)
	return &CallExpression{
		Callee:           callee,
		GoAnnotations:    nil,
		Args:             arguments,
		RelativeLocation: callee.GetRelativeLocation(),
		LparenLocation:   leftParenLocation,
		RparenLocation:   rightParenLocation,
		SourceLength:     sourceLen,
	}
}

// calculateCallExprSourceLength returns the total length in bytes of a call
// expression, from the start of the callee to the closing parenthesis.
//
// Takes callee (Expression) which is the function or method being called.
// Takes arguments ([]Expression) which is the list of arguments
// passed to the call.
// Takes leftParenLocation (Location) which is the position of the
// opening parenthesis.
//
// Returns int which is the byte length of the complete call expression.
func calculateCallExprSourceLength(callee Expression, arguments []Expression, leftParenLocation Location) int {
	if len(arguments) > 0 {
		lastArg := arguments[len(arguments)-1]
		return lastArg.GetRelativeLocation().Offset + lastArg.GetSourceLength() - callee.GetRelativeLocation().Offset
	}
	return leftParenLocation.Offset + 1 - callee.GetRelativeLocation().Offset
}

// isIdentStart reports whether r can begin a Go identifier.
//
// Takes r (rune) which is the character to check.
//
// Returns bool which is true if r is a letter or underscore.
func isIdentStart(r rune) bool {
	if r < utf8.RuneSelf {
		return identStartTable[r]
	}
	return unicode.IsLetter(r)
}

// isIdentChar reports whether r is a valid character in a Go identifier.
//
// Takes r (rune) which is the character to check.
//
// Returns bool which is true if r is a letter, digit, or combining mark.
func isIdentChar(r rune) bool {
	if r < utf8.RuneSelf {
		return identCharTable[r]
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
}

func init() {
	prefixParseFunctions[tokenIdent] = (*ExpressionParser).parseIdentifier
	prefixParseFunctions[tokenNumber] = (*ExpressionParser).parseNumericLiteral
	prefixParseFunctions[tokenDecimal] = (*ExpressionParser).parseDecimalLiteral
	prefixParseFunctions[tokenBigInt] = (*ExpressionParser).parseBigIntLiteral
	prefixParseFunctions[tokenDateTime] = (*ExpressionParser).parseDateTimeLiteral
	prefixParseFunctions[tokenDate] = (*ExpressionParser).parseDateLiteral
	prefixParseFunctions[tokenTime] = (*ExpressionParser).parseTimeLiteral
	prefixParseFunctions[tokenDuration] = (*ExpressionParser).parseDurationLiteral
	prefixParseFunctions[tokenRune] = (*ExpressionParser).parseRuneLiteral
	prefixParseFunctions[tokenString] = (*ExpressionParser).parseStringLiteral
	prefixParseFunctions[tokenTemplateLiteral] = (*ExpressionParser).parseTemplateLiteralExpression
	prefixParseFunctions[tokenLBrace] = (*ExpressionParser).parseObjectLiteral
	prefixParseFunctions[tokenLBracket] = (*ExpressionParser).parseArrayLiteral
	prefixParseFunctions[tokenLParen] = (*ExpressionParser).parseParenthesisedExpression
	prefixParseFunctions[tokenKeywordTrue] = (*ExpressionParser).parseBooleanLiteral
	prefixParseFunctions[tokenKeywordFalse] = (*ExpressionParser).parseBooleanLiteral
	prefixParseFunctions[tokenKeywordNil] = (*ExpressionParser).parseNilLiteral
	prefixParseFunctions[tokenAt] = (*ExpressionParser).parseLinkedMessage
}
