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
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// pikoNamespace is the namespace prefix shared by every directive.
	pikoNamespace = "piko"

	// literalTrue is the canonical boolean-true marker used in directive value validation.
	// The parser emits this exact string when normalising keyword argument values, so
	// concentrating it here keeps the emitted text consistent across producers and
	// consumers.
	literalTrue = "true"

	// literalFalse is the boolean-false counterpart to literalTrue.
	literalFalse = "false"

	// asKeywordLookahead is the number of bytes the parameter-binding detector reads past
	// the start of " AS " before deciding the line is a binding rather than free-form prose.
	asKeywordLookahead = 3

	// maxLogicalLineCount caps the logical directive lines a query block may produce.
	//
	// The collector is otherwise bounded only by block.sql size, so this defence-in-depth
	// ceiling stops a pathological or hostile input from forcing unbounded parsing work. The
	// cap is well above any realistic directive header.
	maxLogicalLineCount = 10_000

	// maxContinuationLineCount caps the continuation lines a logical line may absorb.
	//
	// The cap applies while the logical line's paren depth stays open. Without it, an
	// unclosed paren would pull every following comment line into one logical line, so the
	// cap mirrors maxLogicalLineCount as a per-line defence-in-depth ceiling. The limit is
	// well above any realistic multi-line directive header.
	maxContinuationLineCount = 10_000
)

// directiveParser parses directive comment blocks into structured representations.
//
// The function-call grammar covers a top header such as "-- piko.query(name: Foo,
// command: one)", a parameter binding such as "-- $1 as piko.optional(status)", and an
// embed such as "-- piko.embed(orders, from: o)". Multi-line continuation is honoured for
// any directive call: while paren depth remains open, subsequent comment-prefixed lines
// are joined onto the same logical directive.
type directiveParser struct {
	// prefixLookup maps parameter prefix bytes to their directive definitions
	// (engine-supplied: $, :, @, ?).
	prefixLookup map[byte]querier_dto.DirectiveParameterPrefix

	// commentStyle holds the comment syntax used by the SQL engine.
	commentStyle querier_dto.CommentStyle
}

// logicalLine is one directive after multi-line continuation has been resolved. It
// records the original physical line range so every token can map back to file
// coordinates.
type logicalLine struct {
	// content is the joined directive text with comment prefixes stripped from continuation
	// lines.
	content string

	// segments records the (physicalLine, contentStart, contentEnd, columnOffset) mapping
	// for each contributing physical line.
	segments []lineSegment

	// startLine is the one-based physical line number of the first line that contributed to
	// this logical directive.
	startLine int

	// endLine is the one-based physical line number of the last contributing line.
	endLine int

	// untermStringOffset, when non-negative, gives the byte offset in content where an
	// unterminated string literal begins. Reported by parseLogicalLine as Q035 before the
	// lexer/parser run so the real cause is surfaced instead of a downstream "unclosed
	// directive" cascade.
	untermStringOffset int

	// untermStringQuote is the opening quote character of the unterminated string when
	// untermStringOffset >= 0.
	untermStringQuote byte
}

// lineSegment maps a slice of the logical content back to a physical line and column.
//
// contentEnd is exclusive.
type lineSegment struct {
	// physicalLine is the one-based physical line number this segment came from.
	physicalLine int

	// contentStart is the inclusive start offset of this segment in the logical content.
	contentStart int

	// contentEnd is the exclusive end offset of this segment in the logical content.
	contentEnd int

	// columnOffset is the one-based column where the content begins on its physical line.
	columnOffset int
}

// newDirectiveParser constructs a parser bound to the engine's parameter prefixes and
// comment style.
//
// Takes prefixes ([]querier_dto.DirectiveParameterPrefix) which are the engine's
// parameter prefixes to recognise.
// Takes commentStyle (querier_dto.CommentStyle) which is the engine's comment syntax.
//
// Returns *directiveParser which is the configured parser.
func newDirectiveParser(
	prefixes []querier_dto.DirectiveParameterPrefix,
	commentStyle querier_dto.CommentStyle,
) *directiveParser {
	lookup := make(map[byte]querier_dto.DirectiveParameterPrefix, len(prefixes))
	for _, prefix := range prefixes {
		lookup[prefix.Prefix] = prefix
	}
	return &directiveParser{
		commentStyle: commentStyle,
		prefixLookup: lookup,
	}
}

// Parse turns a query block's directive comment lines into a structured DirectiveBlock
// plus any diagnostics produced along the way. Lines that do not begin with the engine
// comment prefix break the directive header; everything before that break is parsed.
//
// Takes block (queryBlock) which holds the SQL text and its starting line number.
// Takes filename (string) which is the source file name used for diagnostics.
//
// Returns *querier_dto.DirectiveBlock which is the parsed directive header.
// Returns []querier_dto.SourceError which are the diagnostics gathered during parsing.
func (p *directiveParser) Parse(
	block queryBlock,
	filename string,
) (*querier_dto.DirectiveBlock, []querier_dto.SourceError) {
	result := &querier_dto.DirectiveBlock{}
	errorBuilder := querier_dto.NewErrorBuilder(filename)
	var diagnostics []querier_dto.SourceError
	sawTopCall := false

	logicalLines, truncated := p.collectLogicalLines(block)
	if truncated {
		span := querier_dto.TextSpan{Line: block.startLine, Column: 1, EndLine: block.startLine, EndColumn: 1}
		diagnostics = append(diagnostics, errorBuilder.At(
			span,
			querier_dto.CodeDirectiveLineLimit,
			fmt.Sprintf("directive header exceeded %d logical lines and was truncated", maxLogicalLineCount),
		))
	}
	firstLine, lastLine := -1, -1

	for _, logical := range logicalLines {
		if firstLine == -1 {
			firstLine = logical.startLine
		}
		lastLine = logical.endLine

		if p.parseLogicalLine(logical, result, &diagnostics, errorBuilder) {
			sawTopCall = true
		}
	}

	setBlockSpan(result, firstLine, lastLine, block)
	if !sawTopCall {
		validateRequiredDirectives(result, &diagnostics, errorBuilder, block.startLine)
	}
	return result, diagnostics
}

// collectLogicalLines walks the block's physical lines and gathers the directive ones.
//
// A comment line that does not begin a directive (see looksLikeDirective) is prose owned
// by the SQL author and is skipped without scanning, so an apostrophe or an unbalanced
// bracket in that prose can neither be misreported as an unterminated string nor merge a
// following directive into itself through continuation. Each retained directive line is
// joined with any continuation lines whose paren depth is still open. Collection stops
// once maxLogicalLineCount logical lines have been gathered and reports the truncation
// through the second return value so Parse can emit a Q-coded diagnostic.
//
// Takes block (queryBlock) which holds the SQL text and its starting line number.
//
// Returns []logicalLine which are the collected directive lines after continuation.
// Returns bool which is true when collection stopped at maxLogicalLineCount.
func (p *directiveParser) collectLogicalLines(block queryBlock) ([]logicalLine, bool) {
	lines := strings.Split(block.sql, "\n")
	var result []logicalLine

	index := 0
	for index < len(lines) {
		stripped, columnOffset, ok := p.stripCommentPrefix(lines[index])
		if !ok {
			break
		}
		if strings.TrimSpace(stripped) == "" {
			index++
			continue
		}
		if !p.looksLikeDirective(stripped) {
			index++
			continue
		}
		if len(result) >= maxLogicalLineCount {
			return result, true
		}
		startPhysical := block.startLine + index
		built, newIndex := p.buildLogicalLine(lines, index, block, stripped, columnOffset, startPhysical)
		result = append(result, built)
		index = newIndex + 1
	}
	return result, false
}

// buildLogicalLine starts a logical line and joins its continuation lines.
//
// The line starts at lineIndex with the given stripped content, then keeps joining
// following comment-prefixed lines while the running paren depth remains open and no
// unterminated string literal has been seen. Continuation lines are accumulated into a
// strings.Builder (so joining stays linear rather than quadratic) and are capped at
// maxContinuationLineCount, so an unclosed paren cannot pull an unbounded slice of the
// file into a single logical line.
//
// Takes lines ([]string) which are the block's physical lines.
// Takes lineIndex (int) which is the index of the first line in lines.
// Takes block (queryBlock) which supplies the block's starting line number.
// Takes firstStripped (string) which is the first line with its comment prefix removed.
// Takes firstColumnOffset (int) which is the one-based column of the stripped content.
// Takes startPhysical (int) which is the physical line number of the first line.
//
// Returns logicalLine which is the assembled directive line.
// Returns int which is the physical-line index of the last contributing line.
func (p *directiveParser) buildLogicalLine(
	lines []string,
	lineIndex int,
	block queryBlock,
	firstStripped string,
	firstColumnOffset, startPhysical int,
) (logicalLine, int) {
	segments := []lineSegment{{
		physicalLine: startPhysical,
		contentStart: 0,
		contentEnd:   len(firstStripped),
		columnOffset: firstColumnOffset,
	}}
	var content strings.Builder
	content.WriteString(firstStripped)
	depth, untermCol, untermQuote := scanLineForDepth(firstStripped)
	untermOffset := -1
	if untermCol >= 0 {
		untermOffset = untermCol
	}

	index := lineIndex
	continuations := 0
	for depth > 0 && untermOffset < 0 && index+1 < len(lines) && continuations < maxContinuationLineCount {
		index++
		nextStripped, nextColumnOffset, nextOk := p.stripCommentPrefix(lines[index])
		if !nextOk {
			index--
			break
		}
		continuations++
		segmentStart := content.Len() + 1
		content.WriteByte(' ')
		content.WriteString(nextStripped)
		nextDepth, nextUntermCol, nextUntermQuote := scanLineForDepth(nextStripped)
		depth += nextDepth
		if nextUntermCol >= 0 {
			untermOffset = segmentStart + nextUntermCol
			untermQuote = nextUntermQuote
		}
		segments = append(segments, lineSegment{
			physicalLine: block.startLine + index,
			contentStart: segmentStart,
			contentEnd:   segmentStart + len(nextStripped),
			columnOffset: nextColumnOffset,
		})
	}

	return logicalLine{
		content:            content.String(),
		segments:           segments,
		startLine:          startPhysical,
		endLine:            block.startLine + index,
		untermStringOffset: untermOffset,
		untermStringQuote:  untermQuote,
	}, index
}

// stripCommentPrefix removes the engine comment prefix and any following whitespace from
// line.
//
// Takes line (string) which is the physical line to strip.
//
// Returns string which is the stripped suffix after the comment prefix.
// Returns int which is the one-based column at which the suffix begins.
// Returns bool which is true when the line was comment-prefixed.
func (p *directiveParser) stripCommentPrefix(line string) (string, int, bool) {
	trimLen := 0
	for trimLen < len(line) {
		ch := line[trimLen]
		if ch == ' ' || ch == '\t' {
			trimLen++
			continue
		}
		break
	}
	prefix := p.commentStyle.LinePrefix
	if !strings.HasPrefix(line[trimLen:], prefix) {
		return "", 0, false
	}
	after := trimLen + len(prefix)
	if after < len(line) && (line[after] == ' ' || line[after] == '\t') {
		after++
	}
	return line[after:], after + 1, true
}

// skipStringLiteral advances past a string literal beginning at openerPos in input.
//
// The opener satisfies input[openerPos] == quote. Both backslash-escaped pairs and
// SQL-standard doubled-quote escapes (two single quotes inside a '-string) are skipped as
// a unit so they do not prematurely close the literal. This mirrors the directive lexer's
// scanString; without the doubled single-quote handling the two disagreed on a value that
// contains a doubled single quote (the lexer keeps scanning; this scanner closed the
// string early and mis-measured paren depth on the rest of the line).
//
// Takes input (string) which is the line being scanned.
// Takes openerPos (int) which is the index of the opening quote.
// Takes quote (byte) which is the opening quote character.
//
// Returns int which is the index just after the closing quote.
// Returns bool which is true when the string was terminated.
func skipStringLiteral(input string, openerPos int, quote byte) (int, bool) {
	index := openerPos + 1
	for index < len(input) {
		if input[index] == '\\' && index+1 < len(input) {
			index += 2
			continue
		}
		if input[index] == quote {
			if index+1 < len(input) && input[index+1] == quote {
				index += 2
				continue
			}
			return index + 1, true
		}
		index++
	}
	return index, false
}

// scanLineForDepth walks input once, computing both the net paren/bracket depth change
// and the column of an unterminated opening quote when present.
//
// String literals are skipped during the depth count so quoted parens do not affect it.
// If a string opener is found whose closing quote does not appear before end-of-input,
// untermColumn is set to the byte offset of the opener and untermQuote is the opening
// character; the depth returned reflects only the brackets seen before the unterminated
// string, since the rest of the line is considered untrustworthy until the user closes
// the quote.
//
// Takes input (string) which is the line to scan.
//
// Returns depth (int) which is the net paren and bracket depth change.
// Returns untermColumn (int) which is the byte offset of an unterminated opener, or -1.
// Returns untermQuote (byte) which is the opening quote of the unterminated string.
func scanLineForDepth(input string) (depth, untermColumn int, untermQuote byte) {
	untermColumn = -1
	index := 0
	for index < len(input) {
		ch := input[index]
		switch ch {
		case '\'', '"':
			openerPos := index
			nextIndex, closed := skipStringLiteral(input, index, ch)
			index = nextIndex
			if !closed {
				untermColumn = openerPos
				untermQuote = ch
				return depth, untermColumn, untermQuote
			}
		case '(', '[':
			depth++
			index++
		case ')', ']':
			depth--
			index++
		default:
			index++
		}
	}
	return depth, untermColumn, untermQuote
}

// parseLogicalLine dispatches one logical directive line to the appropriate sub-parser
// based on its leading token shape.
//
// Takes logical (logicalLine) which is the directive line to dispatch.
// Takes result (*querier_dto.DirectiveBlock) which accumulates the parsed directives.
// Takes diagnostics (*[]querier_dto.SourceError) which collects emitted diagnostics.
// Takes errorBuilder (querier_dto.ErrorBuilder) which constructs the diagnostics.
//
// Returns bool which is true when the line was a top-level piko(...) call, so the outer
// loop can suppress the redundant required-keyword argument validation that fires when no
// top call was parsed at all.
func (p *directiveParser) parseLogicalLine(
	logical logicalLine,
	result *querier_dto.DirectiveBlock,
	diagnostics *[]querier_dto.SourceError,
	errorBuilder querier_dto.ErrorBuilder,
) bool {
	if logical.untermStringOffset >= 0 {
		lexer := newDirectiveLexer(logical)
		span := lexer.spanAt(logical.untermStringOffset, logical.untermStringOffset+1)
		*diagnostics = append(*diagnostics, errorBuilder.UnterminatedString(span, string(logical.untermStringQuote)))
		return false
	}

	lexer := newDirectiveLexer(logical)
	first := lexer.peek()
	if first.kind == tokenEOF {
		return false
	}

	if prefix, ok := p.prefixLookup[first.firstByte()]; ok && p.looksLikeParameterBinding(logical.content) {
		p.parseParameterBinding(lexer, prefix, result, diagnostics, errorBuilder)
		return false
	}

	if first.kind == tokenIdent && first.lexeme == pikoNamespace {
		return p.parsePikoCall(lexer, result, diagnostics, errorBuilder)
	}
	return false
}

// looksLikeParameterBinding peeks at the content to confirm the line resembles
// `<sigil><tail> as piko.X(...)` rather than free-form prose.
//
// Takes content (string) which is the logical line content to inspect.
//
// Returns bool which is true when the line resembles a parameter binding.
func (*directiveParser) looksLikeParameterBinding(content string) bool {
	upper := strings.ToUpper(content)
	for index := 0; index+asKeywordLookahead < len(upper); index++ {
		if upper[index] == ' ' && upper[index+1] == 'A' && upper[index+2] == 'S' &&
			(upper[index+asKeywordLookahead] == ' ' || upper[index+asKeywordLookahead] == '\t') {
			return true
		}
	}
	return false
}

// looksLikeDirective reports whether a stripped comment line begins a piko directive.
//
// collectLogicalLines uses it as the single gate deciding whether a leading comment line
// is parsed as a directive (scanned for paren depth, joined with continuation lines,
// checked for an unterminated string) or skipped as a free-form SQL comment owned by the
// SQL author.
//
// A directive takes one of two shapes. The first is a piko call: the leading token is the
// bare "piko" identifier that continues with "(" or ".", as in piko(...) or
// piko.embed(...). A comment that only starts with the word "piko" (such as "piko keeps
// this fast") does not continue that way and is prose. The second is a parameter binding:
// the leading token is an engine parameter sigil ($, :, @, ?) and the line carries an "
// as " keyword, as in "$1 as piko.optional(status)". The check reuses the lexer and the
// predicates the dispatcher applies, so a line accepted here is the same line
// parseLogicalLine would route to a sub-parser.
//
// Takes content (string) which is one comment line with its prefix already stripped.
//
// Returns bool reporting whether the line should be parsed as a directive.
func (p *directiveParser) looksLikeDirective(content string) bool {
	lexer := newDirectiveLexer(logicalLine{content: content})
	first := lexer.peek()
	if first.kind == tokenEOF {
		return false
	}
	if first.kind == tokenIdent && first.lexeme == pikoNamespace {
		second := lexer.peekN(1)
		return second.kind == tokenLParen || second.kind == tokenDot
	}
	if _, isPrefix := p.prefixLookup[first.firstByte()]; isPrefix && p.looksLikeParameterBinding(content) {
		return true
	}
	return false
}

// parsePikoCall parses a header-level directive call in the piko namespace.
//
// It handles forms such as `-- piko.query(...)` and `-- piko.embed(...)`. The canonical
// top-level header is `piko.query`; a bare `piko(...)` (no `.role` suffix) is not a
// registered directive and surfaces an UnknownDirective diagnostic.
//
// Takes lexer (*directiveLexer) which supplies the directive tokens.
// Takes result (*querier_dto.DirectiveBlock) which accumulates the parsed directives.
// Takes diagnostics (*[]querier_dto.SourceError) which collects emitted diagnostics.
// Takes errorBuilder (querier_dto.ErrorBuilder) which constructs the diagnostics.
//
// Returns bool which is true when the call resolved to the top-level header, letting the
// caller suppress redundant missing-required diagnostics from validateRequiredDirectives.
func (*directiveParser) parsePikoCall(
	lexer *directiveLexer,
	result *querier_dto.DirectiveBlock,
	diagnostics *[]querier_dto.SourceError,
	errorBuilder querier_dto.ErrorBuilder,
) bool {
	nameToken := lexer.next()
	directiveName := nameToken.lexeme
	directiveSpan := nameToken.span

	if dot := lexer.peek(); dot.kind == tokenDot {
		lexer.next()
		role := lexer.next()
		if role.kind != tokenIdent {
			*diagnostics = append(*diagnostics, errorBuilder.DirectiveSyntax(role.span, "expected directive name after 'piko.'"))
			return false
		}
		directiveName = pikoNamespace + "." + role.lexeme
		directiveSpan = mergeSpan(nameToken.span, role.span)
	}

	spec, found := querier_dto.LookupDirective(directiveName)
	if !found {
		*diagnostics = append(*diagnostics, errorBuilder.UnknownDirective(directiveSpan, directiveName, querier_dto.DirectiveNames()))
		return false
	}

	call, callErrors := parseCallArgs(lexer, errorBuilder, directiveName)
	*diagnostics = append(*diagnostics, callErrors...)
	if call == nil {
		return false
	}

	specErrors := validateCall(spec, call, errorBuilder, directiveName)
	*diagnostics = append(*diagnostics, specErrors...)

	switch spec.Role {
	case querier_dto.DirectiveRoleTop:
		applyTopCall(result, call)
		return true
	case querier_dto.DirectiveRoleHeader:
		applyHeaderCall(result, spec, call, errorBuilder, diagnostics)
	case querier_dto.DirectiveRoleMigration:

		wrongContext := errorBuilder.At(directiveSpan, querier_dto.CodeDirectiveWrongContext,
			spec.Name+" is a migration directive and is ignored in a query file; use piko.query for query headers")
		wrongContext.Severity = querier_dto.SeverityWarning
		*diagnostics = append(*diagnostics, wrongContext)
	case querier_dto.DirectiveRoleParam:
	}
	return false
}

// parseParameterBinding parses `-- $N as piko.X(name, kw: v, ...)`.
//
// Takes lexer (*directiveLexer) which supplies the directive tokens.
// Takes prefix (querier_dto.DirectiveParameterPrefix) which is the anchor's prefix.
// Takes result (*querier_dto.DirectiveBlock) which accumulates the parsed directives.
// Takes diagnostics (*[]querier_dto.SourceError) which collects emitted diagnostics.
// Takes errorBuilder (querier_dto.ErrorBuilder) which constructs the diagnostics.
func (*directiveParser) parseParameterBinding(
	lexer *directiveLexer,
	prefix querier_dto.DirectiveParameterPrefix,
	result *querier_dto.DirectiveBlock,
	diagnostics *[]querier_dto.SourceError,
	errorBuilder querier_dto.ErrorBuilder,
) {
	anchorToken := lexer.next()
	tail, number, anchorOk := parseParameterAnchor(anchorToken, prefix, diagnostics, errorBuilder)
	if !anchorOk {
		return
	}

	pikoToken, asOk := consumeAsPikoDot(lexer, diagnostics, errorBuilder)
	if !asOk {
		return
	}

	roleToken, roleOk := consumeRoleToken(lexer, diagnostics, errorBuilder)
	if !roleOk {
		return
	}
	directiveName := pikoNamespace + "." + roleToken.lexeme
	kindSpan := mergeSpan(pikoToken.span, roleToken.span)

	spec, found := querier_dto.LookupDirective(directiveName)
	if !found || spec.Role != querier_dto.DirectiveRoleParam {
		*diagnostics = append(*diagnostics, errorBuilder.UnknownDirective(roleToken.span, directiveName, paramDirectiveNames()))
		return
	}

	call, callErrors := resolveParameterCall(lexer, prefix, spec, anchorToken, tail, errorBuilder, directiveName)
	*diagnostics = append(*diagnostics, callErrors...)
	if call == nil {
		return
	}

	specErrors := validateCall(spec, call, errorBuilder, directiveName)
	*diagnostics = append(*diagnostics, specErrors...)

	if number == 0 && prefix.IsNamed {
		bound := 0
		for _, existing := range result.Parameters {
			if existing.Kind != querier_dto.ParameterDirectiveSortable {
				bound++
			}
		}
		number = bound + 1
	}
	directive := buildParameterDirective(spec, call, anchorToken, kindSpan, prefix.IsNamed, number)
	if directive != nil {
		result.Parameters = append(result.Parameters, directive)
	}
}

// parseParameterAnchor extracts the identifier tail and number from a parameter anchor.
//
// For named anchors (`:email`) the tail is the identifier; for positional anchors (`$1`)
// the tail must be a positive integer.
//
// Takes anchorToken (token) which is the parameter-anchor token.
// Takes prefix (querier_dto.DirectiveParameterPrefix) which is the anchor's prefix.
// Takes diagnostics (*[]querier_dto.SourceError) which collects emitted diagnostics.
// Takes errorBuilder (querier_dto.ErrorBuilder) which constructs the diagnostics.
//
// Returns string which is the identifier tail.
// Returns int which is the parsed number, zero for named anchors.
// Returns bool which is false when a diagnostic was emitted and the binding should be
// abandoned.
func parseParameterAnchor(
	anchorToken token,
	prefix querier_dto.DirectiveParameterPrefix,
	diagnostics *[]querier_dto.SourceError,
	errorBuilder querier_dto.ErrorBuilder,
) (string, int, bool) {
	anchorLex := anchorToken.lexeme
	if len(anchorLex) == 0 || anchorLex[0] != prefix.Prefix {
		*diagnostics = append(*diagnostics, errorBuilder.ParameterBindingSyntax(anchorToken.span, "parameter prefix", anchorLex, nil))
		return "", 0, false
	}
	tail := anchorLex[1:]
	if prefix.IsNamed {
		if prefix.Prefix == '{' {
			name := strings.TrimSuffix(tail, "}")
			if colon := strings.IndexByte(name, ':'); colon >= 0 {
				name = name[:colon]
			}
			name = strings.TrimSpace(name)
			if !isLegalParameterName(name) {
				*diagnostics = append(*diagnostics, errorBuilder.ParameterBindingSyntax(anchorToken.span, "identifier in '{name:Type}'", anchorLex, nil))
				return "", 0, false
			}
			return name, 0, true
		}
		if tail == "" {
			*diagnostics = append(*diagnostics, errorBuilder.ParameterBindingSyntax(anchorToken.span, "identifier after parameter prefix", anchorLex, nil))
			return "", 0, false
		}
		return tail, 0, true
	}
	parsed, parseErr := strconv.Atoi(tail)
	if parseErr != nil || parsed <= 0 {
		*diagnostics = append(*diagnostics, errorBuilder.ParameterBindingSyntax(anchorToken.span, fmt.Sprintf("positive integer after %q", string(prefix.Prefix)), tail, nil))
		return "", 0, false
	}
	return tail, parsed, true
}

// isLegalParameterName reports whether name is a valid bare parameter identifier (ASCII
// letter or underscore start, then letters/digits/underscores). Used to reject a
// malformed ClickHouse brace anchor before its corrupted text reaches name conversion.
//
// Takes name (string) which is the candidate parameter identifier.
//
// Returns bool which is true when name is a legal identifier.
func isLegalParameterName(name string) bool {
	if name == "" || !isIdentifierStart(name[0]) {
		return false
	}
	for index := 1; index < len(name); index++ {
		if !isIdentifierPart(name[index]) {
			return false
		}
	}
	return true
}

// consumeAsPikoDot reads the literal `as piko.` token sequence that follows a parameter
// anchor.
//
// Takes lexer (*directiveLexer) which supplies the directive tokens.
// Takes diagnostics (*[]querier_dto.SourceError) which collects emitted diagnostics.
// Takes errorBuilder (querier_dto.ErrorBuilder) which constructs the diagnostics.
//
// Returns token which is the consumed piko token, used to build a `piko.X` kind span.
// Returns bool which is false the moment the stream diverges from `as piko.`.
func consumeAsPikoDot(lexer *directiveLexer, diagnostics *[]querier_dto.SourceError, errorBuilder querier_dto.ErrorBuilder) (token, bool) {
	asToken := lexer.next()
	if asToken.kind != tokenIdent || !strings.EqualFold(asToken.lexeme, "as") {
		*diagnostics = append(*diagnostics, errorBuilder.ParameterBindingSyntax(asToken.span, "'as' after parameter anchor", asToken.lexeme, []string{"as"}))
		return token{}, false
	}
	pikoToken := lexer.next()
	if pikoToken.kind != tokenIdent || pikoToken.lexeme != pikoNamespace {
		*diagnostics = append(*diagnostics, errorBuilder.ParameterBindingSyntax(pikoToken.span, "'piko' after 'as'", pikoToken.lexeme, []string{pikoNamespace}))
		return token{}, false
	}
	dotToken := lexer.next()
	if dotToken.kind != tokenDot {
		*diagnostics = append(*diagnostics, errorBuilder.ParameterBindingSyntax(dotToken.span, "'.' after 'piko'", dotToken.lexeme, nil))
		return token{}, false
	}
	return pikoToken, true
}

// consumeRoleToken reads the role identifier that completes `piko.X`.
//
// Takes lexer (*directiveLexer) which supplies the directive tokens.
// Takes diagnostics (*[]querier_dto.SourceError) which collects emitted diagnostics.
// Takes errorBuilder (querier_dto.ErrorBuilder) which constructs the diagnostics.
//
// Returns token which is the role token when present.
// Returns bool which is false when the role slot did not contain an identifier.
func consumeRoleToken(lexer *directiveLexer, diagnostics *[]querier_dto.SourceError, errorBuilder querier_dto.ErrorBuilder) (token, bool) {
	roleToken := lexer.next()
	if roleToken.kind != tokenIdent {
		*diagnostics = append(*diagnostics, errorBuilder.ParameterBindingSyntax(roleToken.span, "role name after 'piko.'", roleToken.lexeme, paramDirectiveNames()))
		return token{}, false
	}
	return roleToken, true
}

// resolveParameterCall returns the parsed call arguments for a parameter binding.
//
// When the source omits the parenthesised argument list and the spec accepts an implicit
// name positional, a minimal call is synthesised from the anchor tail so named
// placeholders do not have to restate their identifier.
//
// Takes lexer (*directiveLexer) which supplies the directive tokens.
// Takes prefix (querier_dto.DirectiveParameterPrefix) which is the anchor's prefix.
// Takes spec (*querier_dto.DirectiveSpec) which describes the directive being parsed.
// Takes anchorToken (token) which is the parameter-anchor token.
// Takes tail (string) which is the anchor tail used for the synthesised name positional.
// Takes errorBuilder (querier_dto.ErrorBuilder) which constructs the diagnostics.
// Takes directiveName (string) which names the directive for diagnostics.
//
// Returns *callArgs which is the parsed or synthesised call arguments.
// Returns []querier_dto.SourceError which are the diagnostics from parsing the call.
func resolveParameterCall(
	lexer *directiveLexer,
	prefix querier_dto.DirectiveParameterPrefix,
	spec *querier_dto.DirectiveSpec,
	anchorToken token,
	tail string,
	errorBuilder querier_dto.ErrorBuilder,
	directiveName string,
) (*callArgs, []querier_dto.SourceError) {
	if lexer.peek().kind != tokenLParen && prefix.IsNamed && len(spec.Positionals) > 0 && spec.Positionals[0].Name == "name" {
		return &callArgs{
			positionals: []parsedPositional{{value: parsedValue{raw: tail}, span: anchorToken.span}},
			openSpan:    anchorToken.span,
			closeSpan:   anchorToken.span,
			closed:      true,
		}, nil
	}
	return parseCallArgs(lexer, errorBuilder, directiveName)
}

// setBlockSpan stretches result.Span to cover the inclusive physical line range spanned
// by the directive header.
//
// Takes result (*querier_dto.DirectiveBlock) whose Span is widened in place.
// Takes firstLine (int) which is the first directive line, or -1 when none exist.
// Takes lastLine (int) which is the last directive line.
// Takes block (queryBlock) which supplies the SQL text for end-column resolution.
func setBlockSpan(result *querier_dto.DirectiveBlock, firstLine, lastLine int, block queryBlock) {
	if firstLine == -1 {
		return
	}
	lines := strings.Split(block.sql, "\n")
	endLineIndex := lastLine - block.startLine
	endColumn := 1
	if endLineIndex >= 0 && endLineIndex < len(lines) {
		endColumn = len(lines[endLineIndex]) + 1
	}
	result.Span = querier_dto.TextSpan{
		Line:      firstLine,
		Column:    1,
		EndLine:   lastLine,
		EndColumn: endColumn,
	}
}

// validateRequiredDirectives emits the missing-name and missing-command diagnostics when
// no top-level `piko(...)` call appeared in the block.
//
// The diagnostics are anchored to the block's first line so the user sees the file
// location the directive header was expected to start at.
//
// Takes result (*querier_dto.DirectiveBlock) which is inspected for the required fields.
// Takes diagnostics (*[]querier_dto.SourceError) which collects emitted diagnostics.
// Takes errorBuilder (querier_dto.ErrorBuilder) which constructs the diagnostics.
// Takes startLine (int) which is the block's first line used to anchor the diagnostics.
func validateRequiredDirectives(result *querier_dto.DirectiveBlock, diagnostics *[]querier_dto.SourceError, errorBuilder querier_dto.ErrorBuilder, startLine int) {
	span := querier_dto.TextSpan{Line: startLine, Column: 1, EndLine: startLine, EndColumn: 1}
	if result.Name == nil {
		*diagnostics = append(*diagnostics, errorBuilder.At(span, querier_dto.CodeMissingDirective, "missing 'name' keyword argument on piko(...)"))
	}
	if result.Command == nil {
		*diagnostics = append(*diagnostics, errorBuilder.At(span, querier_dto.CodeMissingDirective, "missing 'command' keyword argument on piko(...)"))
	}
}

// parseCommandValue maps a directive's raw command identifier to its QueryCommand value.
//
// It handles values such as "one" and "many", and returns an error for values the spec
// does not recognise so the caller can decide whether to surface it as a diagnostic or
// fall back to defaults.
//
// Takes raw (string) which is the command identifier from the directive.
//
// Returns querier_dto.QueryCommand which is the mapped command value.
// Returns error when the command identifier is not recognised.
func parseCommandValue(raw string) (querier_dto.QueryCommand, error) {
	switch strings.ToLower(raw) {
	case "one":
		return querier_dto.QueryCommandOne, nil
	case "many":
		return querier_dto.QueryCommandMany, nil
	case "exec":
		return querier_dto.QueryCommandExec, nil
	case "execresult":
		return querier_dto.QueryCommandExecResult, nil
	case "execrows":
		return querier_dto.QueryCommandExecRows, nil
	case "batch":
		return querier_dto.QueryCommandBatch, nil
	case "stream":
		return querier_dto.QueryCommandStream, nil
	case "copyfrom":
		return querier_dto.QueryCommandCopyFrom, nil
	case "asyncexec":
		return querier_dto.QueryCommandAsyncExec, nil
	default:
		return 0, fmt.Errorf("unknown command %q", raw)
	}
}

// extractQueryDirectives folds the MetadataDirective entries on a DirectiveBlock into a
// QueryDirectives struct downstream consumers (catalogue builder, emitter) read directly.
// Each metadata key corresponds to the same QueryDirectives field as the legacy grammar
// so emitter code does not change.
//
// Takes block (*querier_dto.DirectiveBlock) whose Metadata entries are folded.
//
// Returns *querier_dto.QueryDirectives which is the assembled directive set.
func extractQueryDirectives(block *querier_dto.DirectiveBlock) *querier_dto.QueryDirectives {
	directives := &querier_dto.QueryDirectives{}
	for _, metadata := range block.Metadata {
		switch metadata.Directive {
		case "group_by":
			directives.GroupByKeys = append(directives.GroupByKeys, metadata.Value)
		case "nullable":
			setBoolOverride(&directives.NullableOverride, metadata.Value)
		case "readonly":
			setBoolOverride(&directives.ReadOnlyOverride, metadata.Value)
		case "optional":
			directives.Optional = strings.EqualFold(strings.TrimSpace(metadata.Value), literalTrue)
		case "runtime":

			switch strings.ToLower(strings.TrimSpace(metadata.Value)) {
			case literalTrue:
				directives.DynamicRuntime = true
			case literalFalse:
				directives.DynamicRuntime = false
			}
		case "dynamic":
			if strings.EqualFold(strings.TrimSpace(metadata.Value), "runtime") {
				directives.DynamicRuntime = true
			}
		}
	}
	return directives
}

// setBoolOverride parses a "true" / "false" string into a *bool pointer, leaving the
// pointer nil for any other value.
//
// The comparison is case-insensitive to match the validator (which lowercases the value
// before checking), so `nullable: TRUE` is honoured rather than silently dropped.
//
// Takes target (**bool) which receives a pointer to the parsed boolean.
// Takes value (string) which is the raw override text.
func setBoolOverride(target **bool, value string) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case literalTrue:
		*target = new(true)
	case literalFalse:
		*target = new(false)
	}
}

// paramDirectiveNames returns the names of every directive whose role is Param, used to
// populate the "did you mean" candidates emitted when an unknown directive is referenced
// from a parameter binding.
//
// Returns []string which are the param-role directive names.
func paramDirectiveNames() []string {
	var names []string
	for _, spec := range querier_dto.DirectiveSpecs {
		if spec.Role == querier_dto.DirectiveRoleParam {
			names = append(names, spec.Name)
		}
	}
	return names
}

// mergeSpan returns the smallest TextSpan that encloses both start and end. Used by the
// lexer / parser to widen the diagnostic anchor span to cover multi-token constructs.
//
// Takes start (querier_dto.TextSpan) which is the first span to enclose.
// Takes end (querier_dto.TextSpan) which is the second span to enclose.
//
// Returns querier_dto.TextSpan which is the smallest span covering both inputs.
func mergeSpan(start, end querier_dto.TextSpan) querier_dto.TextSpan {
	merged := start
	if end.EndLine > merged.EndLine || (end.EndLine == merged.EndLine && end.EndColumn > merged.EndColumn) {
		merged.EndLine = end.EndLine
		merged.EndColumn = end.EndColumn
	}
	if end.Line < merged.Line || (end.Line == merged.Line && end.Column < merged.Column) {
		merged.Line = end.Line
		merged.Column = end.Column
	}
	return merged
}
