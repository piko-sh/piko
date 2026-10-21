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

package generator_domain

import (
	"github.com/tdewolff/parse/v2"
	parsejs "github.com/tdewolff/parse/v2/js"
)

// pkScriptScanner reads the significant tokens of a PK script for top-level function
// detection, without parsing it.
//
// The tdewolff lexer cannot tell a regular expression literal from division by itself. It
// always lexes a slash as an operator and leaves its parser to call RegExp wherever the
// grammar expects an operand. The scanner has no parser, so it tracks whether the
// previous token leaves an operand expected and, when it does, re-lexes the slash as a
// regular expression. Lexed as code instead, the body of a pattern such as
// /^[a-z0-9_]{0,31}\/$/ yields backslash errors, stray quotes, unbalanced braces and
// comment openers that derail the scan.
//
// The scanner also skips whitespace and comments, resumes after lexer errors rather than
// treating them as the end of input, and tracks brace depth across every token it hands
// out, including those consumed by the declaration handlers.
type pkScriptScanner struct {
	// lexer produces the raw tokens.
	lexer *parsejs.Lexer

	// input holds the script source.
	input *parse.Input

	// parenIsControl holds one entry per open parenthesis, true when it encloses the
	// condition of an if, while, for or with statement.
	parenIsControl []bool

	// braceDepth is the number of open braces after the last token handed out.
	braceDepth int

	// previous is the last significant token, with a property name after a dot recorded as
	// an identifier.
	previous parsejs.TokenType

	// regExpAllowed reports whether a slash at the current position begins a regular
	// expression literal rather than a division.
	regExpAllowed bool

	// lineBreakBefore reports whether a line terminator separates the next token from the
	// previous one.
	lineBreakBefore bool

	// done reports whether the scan has reached the end of input.
	done bool
}

// newPKScriptScanner creates a scanner positioned at the start of source, where a regular
// expression literal may begin.
//
// Takes source (string) which is the PK script source to scan.
//
// Returns *pkScriptScanner which is ready to hand out tokens.
func newPKScriptScanner(source string) *pkScriptScanner {
	input := parse.NewInputString(source)
	return &pkScriptScanner{
		lexer:           parsejs.NewLexer(input),
		input:           input,
		parenIsControl:  nil,
		braceDepth:      0,
		previous:        parsejs.ErrorToken,
		regExpAllowed:   true,
		lineBreakBefore: true,
		done:            false,
	}
}

// next advances to the next significant token, skipping whitespace, comments and tokens
// the lexer could not read. A slash where an operand is expected comes back as a single
// RegExpToken holding the whole literal.
//
// Returns parsejs.TokenType which is the type of the token, or ErrorToken once the input
// is exhausted.
// Returns []byte which holds the raw bytes of the token.
func (s *pkScriptScanner) next() (parsejs.TokenType, []byte) {
	for !s.done {
		offset := s.input.Offset()
		tt, data := s.lexer.Next()
		if s.regExpAllowed && isDivisionToken(tt) {
			tt, data = s.lexer.RegExp()
		}
		if s.skip(tt, offset) {
			continue
		}
		s.record(tt)
		return tt, data
	}
	return parsejs.ErrorToken, nil
}

// skip reports whether a token carries nothing for the scan, noting line breaks and
// resuming after lexer errors on the way.
//
// Takes tt (parsejs.TokenType) which is the token just lexed.
// Takes offset (int) which is the input offset before the token was lexed.
//
// Returns bool which is true when the token should be passed over.
func (s *pkScriptScanner) skip(tt parsejs.TokenType, offset int) bool {
	switch tt {
	case parsejs.WhitespaceToken, parsejs.CommentToken:
		return true
	case parsejs.LineTerminatorToken, parsejs.CommentLineTerminatorToken:
		s.lineBreakBefore = true
		return true
	case parsejs.ErrorToken:
		s.resumeAfterError(offset)
		return true
	default:
		return false
	}
}

// resumeAfterError carries on past a token the lexer could not read, such as a
// decorator's @ or an unterminated string. The lexer has already moved beyond the bad
// input, so the scan only ends at the end of input or if the lexer made no progress.
//
// Takes offset (int) which is the input offset before the failed token was lexed.
func (s *pkScriptScanner) resumeAfterError(offset int) {
	if s.input.Err() != nil || s.input.Offset() == offset {
		s.done = true
		return
	}
	s.input.Shift()
}

// record updates the scanner's expression state and parenthesis and brace depths for a
// significant token.
//
// Takes tt (parsejs.TokenType) which is the token being handed out.
func (s *pkScriptScanner) record(tt parsejs.TokenType) {
	if isPropertyAccessToken(s.previous) && parsejs.IsIdentifierName(tt) {
		tt = parsejs.IdentifierToken
	}
	s.regExpAllowed = s.regExpMayFollow(tt)
	s.braceDepth = updateBraceDepth(tt, s.braceDepth)
	s.previous = tt
	s.lineBreakBefore = false
}

// regExpMayFollow reports whether a regular expression literal may begin after a token,
// which is whenever the token leaves an operand expected. It also maintains the stack of
// open parentheses, because only the parenthesis closing a control statement's condition
// is followed by a statement rather than an operator.
//
// Takes tt (parsejs.TokenType) which is the token being recorded.
//
// Returns bool which is true when a following slash opens a regular expression.
func (s *pkScriptScanner) regExpMayFollow(tt parsejs.TokenType) bool {
	switch tt {
	case parsejs.OpenParenToken:
		s.parenIsControl = append(s.parenIsControl, isControlKeyword(s.previous))
		return true
	case parsejs.CloseParenToken:
		return s.popParen()
	case parsejs.CloseBraceToken, parsejs.TemplateStartToken, parsejs.TemplateMiddleToken:
		return true
	case parsejs.CloseBracketToken, parsejs.IncrToken, parsejs.DecrToken:
		return false
	case parsejs.NotToken:
		return s.regExpAllowed || s.lineBreakBefore
	default:
		return isRegExpKeyword(tt) || parsejs.IsPunctuator(tt)
	}
}

// popParen closes the innermost open parenthesis.
//
// Returns bool which is true when the parenthesis enclosed a control statement's
// condition, so that a statement, and possibly a regular expression, follows.
func (s *pkScriptScanner) popParen() bool {
	last := len(s.parenIsControl) - 1
	if last < 0 {
		return false
	}
	isControl := s.parenIsControl[last]
	s.parenIsControl = s.parenIsControl[:last]
	return isControl
}

// isDivisionToken reports whether the lexer read a slash as division or division
// assignment, the two tokens RegExp can re-read as a regular expression literal.
//
// Takes tt (parsejs.TokenType) which is the token to check.
//
// Returns bool which is true for / and /=.
func isDivisionToken(tt parsejs.TokenType) bool {
	return tt == parsejs.DivToken || tt == parsejs.DivEqToken
}

// isPropertyAccessToken reports whether a token is followed by a property name, which is
// an identifier even when it is spelt like a keyword, as in options.default.
//
// Takes tt (parsejs.TokenType) which is the token to check.
//
// Returns bool which is true for . and ?.
func isPropertyAccessToken(tt parsejs.TokenType) bool {
	return tt == parsejs.DotToken || tt == parsejs.OptChainToken
}

// isControlKeyword reports whether a keyword introduces a parenthesised condition that is
// followed by a statement.
//
// Takes tt (parsejs.TokenType) which is the token before an open parenthesis.
//
// Returns bool which is true for if, while, for and with.
func isControlKeyword(tt parsejs.TokenType) bool {
	switch tt {
	case parsejs.IfToken, parsejs.WhileToken, parsejs.ForToken, parsejs.WithToken:
		return true
	default:
		return false
	}
}

// isRegExpKeyword reports whether a keyword leaves an operand expected, as return and
// typeof do. Every other keyword either cannot precede a slash or ends an operand, as
// this and null do.
//
// Takes tt (parsejs.TokenType) which is the token to check.
//
// Returns bool which is true when a regular expression literal may follow the keyword.
func isRegExpKeyword(tt parsejs.TokenType) bool {
	switch tt {
	case parsejs.ReturnToken, parsejs.TypeofToken, parsejs.InstanceofToken, parsejs.InToken,
		parsejs.NewToken, parsejs.DeleteToken, parsejs.VoidToken, parsejs.ThrowToken,
		parsejs.CaseToken, parsejs.DoToken, parsejs.ElseToken, parsejs.YieldToken,
		parsejs.AwaitToken, parsejs.DefaultToken, parsejs.ExtendsToken:
		return true
	default:
		return false
	}
}
