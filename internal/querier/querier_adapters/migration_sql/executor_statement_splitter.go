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

package migration_sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const (
	// keywordCASE is the SQL CASE keyword tracked for BEGIN..END block-depth balancing so a
	// scalar CASE inside a procedural body does not prematurely close the block.
	keywordCASE = "CASE"
)

var (
	// ErrMalformedSQLStatement is returned when migration SQL contains unterminated string
	// literals, unterminated dollar-quoted blocks, or unterminated block comments. Callers
	// can detect this with errors.Is.
	ErrMalformedSQLStatement = errors.New("malformed SQL statement")
)

// statementSplitter holds the state for splitStatements as a small state machine. It
// keeps the per-mode scanners small so each one stays well below the cognitive-complexity
// threshold.
type statementSplitter struct {
	// current accumulates the runes of the statement currently being scanned.
	current strings.Builder

	// statements collects flushed statements in input order.
	statements []string

	// runes is the full input decoded as a rune slice for index-based scanning.
	runes []rune

	// index is the cursor into runes for the next rune to scan.
	index int

	// blockDepth tracks open SQLite BEGIN...END blocks so embedded semicolons inside a
	// CREATE TRIGGER body do not terminate the outer CREATE TRIGGER statement. PostgreSQL
	// function bodies use dollar-quoting (handled separately) and never count toward
	// blockDepth, so the same splitter stays correct for both engines.
	blockDepth int

	// caseDepth tracks open CASE expressions inside a BEGIN...END block.
	//
	// A scalar `... = CASE WHEN ... END` inside a trigger/procedure body closes with a bare
	// END that must NOT decrement blockDepth, otherwise the following semicolon splits the
	// body mid-statement. caseDepth is only counted while blockDepth > 0.
	caseDepth int

	// backslashEscapes is true for dialects (ClickHouse, MySQL) that treat a backslash
	// inside a single-quoted string literal as an escape character, so a `\'` does not
	// terminate the literal. Standard-SQL dialects leave it false.
	backslashEscapes bool

	// statementHasContent is true once a non-whitespace rune has been written into the
	// current statement buffer since the last flush.
	//
	// It lets the BEGIN word-boundary check decide in O(1) whether BEGIN opens a
	// trigger/procedure body (content already present) or is a standalone statement (buffer
	// still blank), instead of re-scanning the whole buffer with TrimSpace on every BEGIN,
	// which is quadratic on a buffer full of leading whitespace.
	statementHasContent bool
}

// writeRune appends r to the current statement buffer. The wrapper exists so the
// unhandled-error linter only sees one ignored WriteRune call site rather than many;
// (*strings.Builder).WriteRune is documented to always return nil.
//
// Takes r (rune) which is the rune to append.
func (s *statementSplitter) writeRune(r rune) {
	_, _ = s.current.WriteRune(r)
	if !s.statementHasContent && !unicode.IsSpace(r) {
		s.statementHasContent = true
	}
}

// writeCommentRune appends a comment rune to the current statement buffer WITHOUT marking
// the statement as having content.
//
// Comment text is purely descriptive, so a leading "-- describe this" before a standalone
// BEGIN must not make the BEGIN look like a trigger/procedure-body opener: that would
// swallow every embedded `;` and collapse a BEGIN/COMMIT transaction-control pair into
// one un-splittable statement.
//
// Takes r (rune) which is the comment rune to append.
func (s *statementSplitter) writeCommentRune(r rune) {
	_, _ = s.current.WriteRune(r)
}

// writeRange appends runes[start:end] to the current statement buffer. Every caller
// copies a keyword or quote-delimited token, so the range always carries non-whitespace
// content.
//
// Takes start (int) which is the inclusive start index.
// Takes end (int) which is the exclusive end index.
func (s *statementSplitter) writeRange(start, end int) {
	_, _ = s.current.WriteString(string(s.runes[start:end]))
	s.statementHasContent = true
}

// flush trims and emits the buffered statement when non-empty.
func (s *statementSplitter) flush() {
	stmt := strings.TrimSpace(s.current.String())
	s.current.Reset()
	s.statementHasContent = false
	if stmt != "" {
		s.statements = append(s.statements, stmt)
	}
}

// scanLineComment consumes a "-- ..." comment up to and excluding the newline. Comment
// text is written through writeCommentRune so it never sets statementHasContent (see M5).
func (s *statementSplitter) scanLineComment() {
	s.writeCommentRune(s.runes[s.index])
	s.writeCommentRune(s.runes[s.index+1])
	s.index += 2
	for s.index < len(s.runes) && s.runes[s.index] != '\n' {
		s.writeCommentRune(s.runes[s.index])
		s.index++
	}
}

// scanBlockComment consumes a "/* ... */" block comment.
//
// Comment text is written through writeCommentRune so it never sets statementHasContent
// (see M5).
//
// Returns error wrapping ErrMalformedSQLStatement when the comment never terminates.
func (s *statementSplitter) scanBlockComment() error {
	s.writeCommentRune(s.runes[s.index])
	s.writeCommentRune(s.runes[s.index+1])
	s.index += 2
	for s.index < len(s.runes) {
		if s.runes[s.index] == '*' && s.index+1 < len(s.runes) && s.runes[s.index+1] == '/' {
			s.writeCommentRune(s.runes[s.index])
			s.writeCommentRune(s.runes[s.index+1])
			s.index += 2
			return nil
		}
		s.writeCommentRune(s.runes[s.index])
		s.index++
	}
	return fmt.Errorf("unterminated block comment: %w", ErrMalformedSQLStatement)
}

// scanSingleQuotedString consumes a single-quoted literal, treating a doubled single
// quote as an embedded quote.
//
// Returns error which wraps ErrMalformedSQLStatement when the literal never terminates.
func (s *statementSplitter) scanSingleQuotedString() error {
	s.writeRune(s.runes[s.index])
	s.index++
	for s.index < len(s.runes) {
		if s.backslashEscapes && s.runes[s.index] == '\\' && s.index+1 < len(s.runes) {
			s.writeRune(s.runes[s.index])
			s.writeRune(s.runes[s.index+1])
			s.index += 2
			continue
		}
		if s.runes[s.index] != '\'' {
			s.writeRune(s.runes[s.index])
			s.index++
			continue
		}
		if s.index+1 < len(s.runes) && s.runes[s.index+1] == '\'' {
			s.writeRune(s.runes[s.index])
			s.writeRune(s.runes[s.index+1])
			s.index += 2
			continue
		}
		s.writeRune(s.runes[s.index])
		s.index++
		return nil
	}
	return fmt.Errorf("unterminated string literal: %w", ErrMalformedSQLStatement)
}

// scanDollarQuotedBlock consumes a $tag$ ... $tag$ block when the current position opens
// such a block.
//
// Returns bool which is true when a block was consumed, false otherwise.
// Returns error which wraps ErrMalformedSQLStatement when the block never terminates.
func (s *statementSplitter) scanDollarQuotedBlock() (bool, error) {
	tag, advance, ok := readDollarQuoteTag(s.runes, s.index)
	if !ok {
		return false, nil
	}
	s.writeRange(s.index, s.index+advance)
	s.index += advance
	for s.index < len(s.runes) {
		if s.runes[s.index] == '$' {
			closeTag, closeAdvance, closeOk := readDollarQuoteTag(s.runes, s.index)
			if closeOk && closeTag == tag {
				s.writeRange(s.index, s.index+closeAdvance)
				s.index += closeAdvance
				return true, nil
			}
		}
		s.writeRune(s.runes[s.index])
		s.index++
	}
	return true, fmt.Errorf("unterminated dollar-quoted block (tag=%q): %w", tag, ErrMalformedSQLStatement)
}

// step processes a single token from the input, advancing s.index.
//
// Returns error wrapping ErrMalformedSQLStatement on a malformed lex token.
func (s *statementSplitter) step() error {
	c := s.runes[s.index]
	switch {
	case c == '-' && s.index+1 < len(s.runes) && s.runes[s.index+1] == '-':
		s.scanLineComment()
		return nil
	case c == '/' && s.index+1 < len(s.runes) && s.runes[s.index+1] == '*':
		return s.scanBlockComment()
	case c == '\'':
		return s.scanSingleQuotedString()
	case c == '$':
		consumed, err := s.scanDollarQuotedBlock()
		if err != nil {
			return err
		}
		if !consumed {
			s.writeRune(c)
			s.index++
		}
		return nil
	case c == ';':
		if s.blockDepth > 0 {
			s.writeRune(c)
			s.index++
			return nil
		}
		s.flush()
		s.index++
		return nil
	default:
		if consumed := s.tryScanBlockKeyword(); consumed {
			return nil
		}
		s.writeRune(c)
		s.index++
		return nil
	}
}

// tryScanBlockKeyword detects the SQL keywords BEGIN and END at a word boundary and
// updates blockDepth accordingly.
//
// The keyword grammar stays narrow: BEGIN only opens a block when followed by whitespace
// and the cursor is not at the very start of the statement (so a standalone "BEGIN
// TRANSACTION" statement at the start of a migration file does not switch the splitter
// into trigger-body mode). END only closes a block when blockDepth > 0, so an identifier
// named "end" cannot accidentally decrement the counter below zero.
//
// END decrements blockDepth only when it closes the surrounding BEGIN..END block. An END
// that closes an inner control structure (END IF, END CASE, END LOOP, END WHILE, END
// REPEAT) or terminates a CASE expression appearing before another clause (END WHEN, END
// FROM, END ELSE, END THEN) must not decrement depth, otherwise the next `;` is treated
// as the outer statement terminator and the trigger/procedure body is split
// mid-control-flow.
//
// Returns bool which is true when a keyword was consumed (the runes were copied into the
// buffer and s.index advanced) and false when the cursor was not at a recognised keyword
// and the default-case copy should run.
func (s *statementSplitter) tryScanBlockKeyword() bool {
	if !isWordBoundaryBefore(s.runes, s.index) {
		return false
	}
	const (
		beginLen = 5
		caseLen  = 4
		endLen   = 3
	)
	if s.matchKeywordAt("BEGIN", beginLen) {
		if !s.statementHasContent {
			return false
		}
		s.writeRange(s.index, s.index+beginLen)
		s.index += beginLen
		s.blockDepth++
		return true
	}

	if s.blockDepth > 0 && s.matchKeywordAt(keywordCASE, caseLen) {
		s.writeRange(s.index, s.index+caseLen)
		s.index += caseLen
		s.caseDepth++
		return true
	}
	if s.blockDepth > 0 && s.matchKeywordAt("END", endLen) {
		nextWord := peekNextWordAfterWhitespace(s.runes, s.index+endLen)
		s.writeRange(s.index, s.index+endLen)
		s.index += endLen
		s.applyEndKeyword(nextWord)

		if nextWord == keywordCASE {
			s.consumeNextKeyword(keywordCASE, caseLen)
		}
		return true
	}
	return false
}

// consumeNextKeyword skips whitespace and SQL comments then appends the keyword of the
// given rune length to the buffer and advances past it.
//
// It is used after an `END CASE` closer to consume the trailing keyword that the END
// handler already accounted for, so the keyword is not re-interpreted by the main
// scanner.
//
// Takes keyword (string) which is the upper-case keyword expected next.
// Takes length (int) which is the rune length of keyword.
func (s *statementSplitter) consumeNextKeyword(keyword string, length int) {
	index := s.index
	for index < len(s.runes) {
		switch {
		case isASCIIWhitespace(s.runes[index]):
			index++
		case index+1 < len(s.runes) && s.runes[index] == '-' && s.runes[index+1] == '-':
			index = skipPeekLineComment(s.runes, index)
		case index+1 < len(s.runes) && s.runes[index] == '/' && s.runes[index+1] == '*':
			index = skipPeekBlockComment(s.runes, index)
		default:
			if index+length <= len(s.runes) && equalsKeywordIgnoreCase(s.runes[index:index+length], keyword) {
				s.writeRange(s.index, index+length)
				s.index = index + length
			}
			return
		}
	}
}

// matchKeywordAt reports whether the keyword of the given rune length sits at the cursor
// at a trailing word boundary (case-insensitive).
//
// Takes keyword (string) which is the upper-case keyword to match.
// Takes length (int) which is the rune length of keyword.
//
// Returns bool which is true when the keyword matches at the cursor with a trailing
// boundary.
func (s *statementSplitter) matchKeywordAt(keyword string, length int) bool {
	return s.index+length <= len(s.runes) &&
		equalsKeywordIgnoreCase(s.runes[s.index:s.index+length], keyword) &&
		isWordBoundaryAfter(s.runes, s.index+length)
}

// applyEndKeyword resolves an END keyword against the open CASE and block counters using
// the following word, mirroring the postgres parser's adjustEndDepth.
//
// END IF, LOOP, WHILE, and REPEAT close inner control structures with no depth change.
// END CASE closes a statement-form CASE by decrementing caseDepth. A bare END closes an
// open expression-CASE first by decrementing caseDepth and only decrements blockDepth
// once no CASE is open, so `... = CASE ... END;` inside a trigger body does not split the
// body at the following semicolon.
//
// Takes nextWord (string) which is the upper-cased word following END (empty at EOF or
// before punctuation such as a semicolon).
func (s *statementSplitter) applyEndKeyword(nextWord string) {
	switch nextWord {
	case "IF", "LOOP", "WHILE", "REPEAT":
		return
	case keywordCASE:
		if s.caseDepth > 0 {
			s.caseDepth--
		}
		return
	}
	if s.caseDepth > 0 {
		s.caseDepth--
		return
	}
	if s.blockDepth > 0 {
		s.blockDepth--
	}
}

// peekNextWordAfterWhitespace returns the next contiguous identifier word in runes
// starting at or after start, skipping ASCII whitespace and SQL line/block comments.
//
// The returned word is upper-case so the caller compares against upper-case keyword
// literals.
//
// Takes runes ([]rune) which is the SQL content being scanned.
// Takes start (int) which is the index to begin scanning from.
//
// Returns string which is the next identifier word in upper case, or empty when none is
// found before end of input.
func peekNextWordAfterWhitespace(runes []rune, start int) string {
	index := start
	for index < len(runes) {
		switch {
		case isASCIIWhitespace(runes[index]):
			index++
		case index+1 < len(runes) && runes[index] == '-' && runes[index+1] == '-':
			index = skipPeekLineComment(runes, index)
		case index+1 < len(runes) && runes[index] == '/' && runes[index+1] == '*':
			index = skipPeekBlockComment(runes, index)
		default:
			if !isIdentifierRune(runes[index]) {
				return ""
			}
			wordStart := index
			for index < len(runes) && isIdentifierRune(runes[index]) {
				index++
			}
			return upperASCII(runes[wordStart:index])
		}
	}
	return ""
}

// skipPeekLineComment advances past a `--` line comment starting at index, stopping at
// the newline or end of input.
//
// Takes runes ([]rune) which is the SQL content being scanned.
// Takes index (int) which is the index of the leading `-` rune.
//
// Returns int which is the index of the newline or end of input after the comment.
func skipPeekLineComment(runes []rune, index int) int {
	index += 2
	for index < len(runes) && runes[index] != '\n' {
		index++
	}
	return index
}

// skipPeekBlockComment advances past a `/* ... */` block comment starting at index,
// including the closing `*/`.
//
// When the closer is missing the scanner stops at end of input so the caller terminates.
//
// Takes runes ([]rune) which is the SQL content being scanned.
// Takes index (int) which is the index of the leading `/` rune.
//
// Returns int which is the index after the closing `*/`, or end of input when
// unterminated.
func skipPeekBlockComment(runes []rune, index int) int {
	index += 2
	for index+1 < len(runes) {
		if runes[index] == '*' && runes[index+1] == '/' {
			return index + 2
		}
		index++
	}
	return index
}

// isASCIIWhitespace recognises the whitespace characters defined by the SQL standard for
// token separation: space, tab, newline, carriage return, vertical tab, form feed.
//
// Takes r (rune) which is the rune to test.
//
// Returns bool which is true when the rune is SQL token-separating whitespace.
func isASCIIWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f'
}

// upperASCII converts ASCII lower-case runes to upper-case, leaving other runes
// unchanged.
//
// Used to normalise look-ahead words for case-insensitive keyword comparison.
//
// Takes runes ([]rune) which is the word to upper-case.
//
// Returns string which is the upper-cased word.
func upperASCII(runes []rune) string {
	builder := make([]rune, len(runes))
	for index, r := range runes {
		if r >= 'a' && r <= 'z' {
			builder[index] = r - ('a' - 'A')
			continue
		}
		builder[index] = r
	}
	return string(builder)
}

// isWordBoundaryBefore reports whether position pos in runes is preceded by a
// non-identifier character or the start of input.
//
// Identifier characters here are ASCII letters, digits, and underscore.
//
// Takes runes ([]rune) which is the content being scanned.
// Takes pos (int) which is the position to test.
//
// Returns bool which is true when a word boundary precedes the position.
func isWordBoundaryBefore(runes []rune, pos int) bool {
	if pos == 0 {
		return true
	}
	return !isIdentifierRune(runes[pos-1])
}

// isWordBoundaryAfter reports whether position pos in runes is followed by a
// non-identifier character or the end of input.
//
// Takes runes ([]rune) which is the content being scanned.
// Takes pos (int) which is the position to test.
//
// Returns bool which is true when a word boundary follows the position.
func isWordBoundaryAfter(runes []rune, pos int) bool {
	if pos >= len(runes) {
		return true
	}
	return !isIdentifierRune(runes[pos])
}

// isIdentifierRune reports whether r is an ASCII letter, digit, or underscore.
//
// Takes r (rune) which is the rune to test.
//
// Returns bool which is true when the rune may appear within an identifier.
func isIdentifierRune(r rune) bool {
	return r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// equalsKeywordIgnoreCase reports whether the rune slice equals the expected keyword,
// case-insensitively against ASCII letters.
//
// The caller supplies the expected keyword in upper-case so the per-rune comparison stays
// branch-free.
//
// Takes actual ([]rune) which is the candidate slice to compare.
// Takes expected (string) which is the upper-case keyword to compare against.
//
// Returns bool which is true when the slice matches the keyword ignoring ASCII case.
func equalsKeywordIgnoreCase(actual []rune, expected string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, r := range actual {
		want := rune(expected[i])
		if r == want {
			continue
		}
		if r >= 'a' && r <= 'z' && r-('a'-'A') == want {
			continue
		}
		return false
	}
	return true
}

// splitStatements splits migration SQL content into individual statements, honouring SQL
// lexical structure so that semicolons inside string literals, dollar-quoted blocks, and
// comments are not treated as statement terminators. Empty statements are skipped.
//
// The splitter recognises single-quoted string literals with a doubled single quote as an
// embedded quote, PostgreSQL dollar-quoted blocks with an optional tag (such as $$ ... $$
// or $tag$ ... $tag$), "--" line comments through to end-of-line, and non-nested "/* ...
// */" block comments.
//
// Takes content (string) which holds the raw migration SQL.
//
// Returns []string which holds the individual non-empty SQL statements.
// Returns error which wraps ErrMalformedSQLStatement when an unterminated string,
// dollar-quote, or block comment is detected.
func splitStatements(content string) ([]string, error) {
	return splitStatementsWithOptions(content, false)
}

// splitStatementsWithOptions is splitStatements with dialect-specific lexing options.
//
// Takes content (string) which holds the raw migration SQL.
// Takes backslashEscapes (bool) which, when true, makes a backslash inside a
// single-quoted string literal an escape character (ClickHouse, MySQL) so a `\'` does not
// terminate it.
//
// Returns []string which holds the individual non-empty SQL statements.
// Returns error which wraps ErrMalformedSQLStatement on an unterminated lexical
// construct.
func splitStatementsWithOptions(content string, backslashEscapes bool) ([]string, error) {
	splitter := &statementSplitter{
		statements:       make([]string, 0, defaultStatementCapacity),
		runes:            []rune(content),
		backslashEscapes: backslashEscapes,
	}
	for splitter.index < len(splitter.runes) {
		if err := splitter.step(); err != nil {
			return nil, err
		}
	}
	if splitter.blockDepth > 0 {
		return nil, fmt.Errorf(
			"unterminated BEGIN block (depth=%d): %w", splitter.blockDepth, ErrMalformedSQLStatement,
		)
	}
	if splitter.caseDepth > 0 {
		return nil, fmt.Errorf(
			"unterminated CASE expression (depth=%d): %w", splitter.caseDepth, ErrMalformedSQLStatement,
		)
	}
	splitter.flush()
	return splitter.statements, nil
}

// readDollarQuoteTag detects whether position start in runes is the start of a
// dollar-quote token (e.g. $$ or $tag$).
//
// Takes runes ([]rune) which holds the SQL content as runes.
// Takes start (int) which is the index of the leading '$' rune.
//
// Returns string which is the inner tag (empty for $$).
// Returns int which is the number of runes consumed for the full token.
// Returns bool which is true when start indeed marks a dollar-quote token.
func readDollarQuoteTag(runes []rune, start int) (string, int, bool) {
	if start >= len(runes) || runes[start] != '$' {
		return "", 0, false
	}

	if start+1 < len(runes) && runes[start+1] != '$' && !isDollarTagStartRune(runes[start+1]) {
		return "", 0, false
	}
	end := start + 1
	for end < len(runes) && runes[end] != '$' {
		if !isDollarTagPartRune(runes[end]) {
			return "", 0, false
		}
		end++
	}
	if end >= len(runes) {
		return "", 0, false
	}
	return string(runes[start+1 : end]), end - start + 1, true
}

// isDollarTagStartRune reports whether r may begin a dollar-quote tag: a letter or
// underscore, matching the PostgreSQL grammar.
//
// Takes r (rune) which is the candidate rune.
//
// Returns bool which is true for a letter or underscore.
func isDollarTagStartRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isDollarTagPartRune reports whether r may appear within a dollar-quote tag body: a
// tag-start rune or an ASCII digit.
//
// Takes r (rune) which is the candidate rune.
//
// Returns bool which is true for a tag-start rune or a digit.
func isDollarTagPartRune(r rune) bool {
	return isDollarTagStartRune(r) || (r >= '0' && r <= '9')
}

// execStatements splits migration SQL on semicolons and executes each non-empty statement
// individually. Statements up to and including skipUpTo are skipped, allowing retry from
// where a partial application left off.
//
// Takes runner which satisfies ExecContext for executing SQL.
// Takes content (string) which holds the raw migration SQL.
// Takes version (int64) which identifies the migration for error messages.
// Takes skipUpTo (int) which is the 0-based index of statements to skip (-1 means execute
// all from the start).
//
// Returns statementsExecuted (int) which is the count of statements successfully
// executed.
// Returns err (error) when any individual statement fails, including which statement
// index failed.
func (executor *Executor) execStatements(
	ctx context.Context,
	runner interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	},
	content string,
	version int64,
	skipUpTo int,
) (statementsExecuted int, err error) {
	statements, splitError := splitStatementsWithOptions(content, executor.dialectConfig.BackslashEscapes)
	if splitError != nil {
		return 0, fmt.Errorf("splitting migration %d statements: %w", version, splitError)
	}

	for i, stmt := range statements {
		if i <= skipUpTo {
			continue
		}
		if cancelErr := ctx.Err(); cancelErr != nil {
			return statementsExecuted, fmt.Errorf(
				"migration %d cancelled before statement %d: %w", version, i+1, cancelErr,
			)
		}
		if _, execError := runner.ExecContext(ctx, stmt); execError != nil {
			return statementsExecuted, fmt.Errorf(
				"statement %d/%d of migration %d: %w",
				i+1, len(statements), version, execError,
			)
		}
		statementsExecuted++
	}

	return statementsExecuted, nil
}
