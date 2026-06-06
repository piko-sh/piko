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

package db_engine_sqlite

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// statementKind classifies a top-level SQL statement by its leading keyword sequence.
type statementKind uint8

const (
	// statementKindCreateTable denotes a CREATE TABLE statement.
	statementKindCreateTable statementKind = iota

	// statementKindDropTable denotes a DROP TABLE statement.
	statementKindDropTable

	// statementKindAlterTable denotes an ALTER TABLE statement.
	statementKindAlterTable

	// statementKindCreateView denotes a CREATE VIEW statement.
	statementKindCreateView

	// statementKindDropView denotes a DROP VIEW statement.
	statementKindDropView

	// statementKindCreateIndex denotes a CREATE INDEX statement.
	statementKindCreateIndex

	// statementKindDropIndex denotes a DROP INDEX statement.
	statementKindDropIndex

	// statementKindCreateTrigger denotes a CREATE TRIGGER statement.
	statementKindCreateTrigger

	// statementKindDropTrigger denotes a DROP TRIGGER statement.
	statementKindDropTrigger

	// statementKindCreateVirtualTable denotes a CREATE VIRTUAL TABLE statement.
	statementKindCreateVirtualTable

	// statementKindSelect denotes a SELECT statement.
	statementKindSelect

	// statementKindInsert denotes an INSERT or REPLACE statement.
	statementKindInsert

	// statementKindUpdate denotes an UPDATE statement.
	statementKindUpdate

	// statementKindDelete denotes a DELETE statement.
	statementKindDelete

	// statementKindValues denotes a bare VALUES statement.
	statementKindValues

	// statementKindUnknown denotes any unrecognised statement shape.
	statementKindUnknown
)

const (
	// defaultMaxParseDepth caps every user-driven recursion in the parser.
	//
	// The cap covers analyseSelect (subquery/CTE/derived-table nesting) and the expression
	// precedence chain (parenthesis nesting). It is essential because Go raises a fatal,
	// non-recoverable stack overflow that the engine's recover guards cannot catch, so
	// deeply nested input would otherwise crash the host process. 256 is far below the
	// overflow threshold yet out of the way for realistic queries; callers may override it
	// with WithMaxParseDepth.
	defaultMaxParseDepth = 256
)

var (
	// errUnmatchedParenthesis is returned when a parenthesised group lacks a closing token.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")

	// errAnalysisDepthExceeded is the sentinel returned when parser recursion exceeds the
	// configured maximum parse depth.
	errAnalysisDepthExceeded = errors.New("sqlite: analysis recursion depth exceeded")
)

// parsedStatement holds the token slice and classified kind for one top-level SQL
// statement.
type parsedStatement struct {
	// tokens are the lexed tokens that form the statement body.
	tokens []token

	// kind records the classified statement shape.
	kind statementKind
}

// IsParsedStatement is a marker method that identifies the type as a parsed SQL
// statement.
func (*parsedStatement) IsParsedStatement() {}

// parser walks a token slice and produces analysis records describing the statement's
// parameters, derived tables, and other interesting clauses.
type parser struct {
	// namedParameterMap remembers the parameter index assigned to each named placeholder for
	// re-use across references.
	namedParameterMap map[string]int

	// insertTargetTable is the unqualified INSERT target table name used as the column
	// reference table alias for INSERT ... SELECT projection placeholders.
	//
	// The qualifier is load-bearing because a bare column name (for example "reason") may
	// exist on several tables. It is consumed and cleared alongside insertTargetColumns.
	insertTargetTable string

	// insertTargetColumns holds the target column names of an enclosing INSERT ... SELECT so
	// the SELECT body's top-level projection placeholders bind to the INSERT target column
	// positionally, mirroring the VALUES path.
	//
	// It is consumed and cleared by the first parseSelectList so nested subqueries and
	// compound branches do not inherit the binding.
	insertTargetColumns []string

	// rawDerivedTables collects every subquery used as a derived table.
	rawDerivedTables []querier_dto.RawDerivedTableReference

	// predicateSubqueries collects every subquery found in a token-scanned predicate
	// position (WHERE / HAVING / JOIN ON), so the domain layer can resolve its parameters
	// against the subquery's own scope rather than the parent.
	predicateSubqueries []*querier_dto.RawQueryAnalysis

	// rawTableValuedFunctions collects every table-valued function call.
	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	// parameterRefs collects every observed parameter reference.
	parameterRefs []querier_dto.RawParameterReference

	// tokens are the lexed tokens being walked.
	tokens []token

	// position is the current token index.
	position int

	// parameterCount is the highest parameter index assigned so far.
	parameterCount int

	// analysisDepth bounds recursion through analyseSelect across compound branches, CTE
	// bodies, derived tables, scalar subqueries, and view bodies. Child parsers inherit the
	// parent depth so the global cap holds across instances.
	analysisDepth int

	// expressionDepth bounds recursion through the expression precedence chain so deeply
	// nested parentheses cannot overflow the stack.
	expressionDepth int

	// maxParseDepth is the effective cap for analysisDepth and expressionDepth. The engine
	// sets it from the dialect and child parsers inherit it; newParser seeds it with
	// defaultMaxParseDepth.
	maxParseDepth int
}

// newParser creates a parser over the token stream seeded with the default depth cap.
//
// Takes tokens ([]token) which is the lexed token stream to walk.
//
// Returns *parser which is ready to analyse the statement.
func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		maxParseDepth:     defaultMaxParseDepth,
		namedParameterMap: make(map[string]int),
	}
}

// splitStatements partitions the token stream by top-level semicolons. SQLite triggers
// wrap their bodies in BEGIN...END blocks containing inner semicolons; those inner
// semicolons belong to the trigger body, not to the migration as a whole, so the splitter
// tracks block nesting and only flushes at depth zero.
//
// BEGIN only opens a block when the in-flight statement is a procedural definition (a
// CREATE TRIGGER body). A bare BEGIN at the start of input is the transaction marker, and
// a `begin` used as a column name or alias in an ordinary statement (for example `SELECT
// 1 AS begin; SELECT 2;`) must not swallow the rest of the input. END only closes a block
// when blockDepth > 0, so identifiers that happen to spell END cannot drive the depth
// negative.
//
// Takes tokens ([]token) which is the lexed token stream to partition.
//
// Returns [][]token which contains one entry per discovered statement.
func splitStatements(tokens []token) [][]token {
	var statements [][]token
	var current []token
	blockDepth := 0
	caseDepth := 0

	for index := range tokens {
		tok := tokens[index]
		if tok.kind == tokenEOF {
			break
		}
		if tok.kind == tokenSemicolon {
			statements, current = handleSplitStatementsSemicolon(statements, current, tok, blockDepth)
			continue
		}
		if tok.kind == tokenIdentifier {
			proceduralContext := false
			if strings.EqualFold(tok.value, "BEGIN") {
				proceduralContext = isProceduralStatement(current)
			}
			blockDepth, caseDepth = updateSplitStatementsBlockDepth(blockDepth, caseDepth, tok, splitStatementsNextValue(tokens, index), len(current), proceduralContext)
		}
		current = append(current, tok)
	}

	if len(current) > 0 {
		statements = append(statements, current)
	}

	return statements
}

// isProceduralStatement reports whether the in-flight statement is a procedural
// definition whose body may legitimately contain a bare BEGIN...END block: a CREATE
// [TEMP] TRIGGER. Any other leading statement (for example SELECT/INSERT/UPDATE or CREATE
// TABLE/VIEW) returns false so a `begin` identifier is treated as a plain name, not a
// block opener.
//
// Takes current ([]token) which is the tokens accumulated for the statement so far.
//
// Returns bool which is true for a trigger definition whose body bears a BEGIN...END
// block.
func isProceduralStatement(current []token) bool {
	sawCreate := false
	for index := range current {
		if current[index].kind != tokenIdentifier {
			continue
		}
		word := strings.ToUpper(current[index].value)
		if !sawCreate {
			if word == "CREATE" {
				sawCreate = true
				continue
			}
			return false
		}
		switch word {
		case "TEMP", "TEMPORARY":
			continue
		case keywordTRIGGER:
			return true
		default:
			return false
		}
	}
	return false
}

// splitStatementsNextValue returns the value of the token after index, or the empty
// string at end of input. The lookahead classifies an END as closing an inner construct
// (END IF / END LOOP / END CASE) versus the surrounding BEGIN block.
//
// Takes tokens ([]token) which is the token stream to read from.
// Takes index (int) which is the position of the current token.
//
// Returns string which is the next token's value, or empty at end of input.
func splitStatementsNextValue(tokens []token, index int) string {
	if index+1 >= len(tokens) {
		return ""
	}
	return tokens[index+1].value
}

// handleSplitStatementsSemicolon either appends the semicolon to the in-progress
// statement when blockDepth > 0 (inside a BEGIN..END body) or flushes the current
// statement to the output list. Returns the possibly-extended statement list and the next
// in-progress slice.
//
// Takes statements ([][]token) which is the running statement list.
// Takes current ([]token) which is the in-progress statement being built.
// Takes tok (token) which is the semicolon token under consideration.
// Takes blockDepth (int) which is the nested BEGIN..END depth.
//
// Returns [][]token which is the updated statement list.
// Returns []token which is the updated in-progress statement.
func handleSplitStatementsSemicolon(statements [][]token, current []token, tok token, blockDepth int) ([][]token, []token) {
	if blockDepth > 0 {
		return statements, append(current, tok)
	}
	if len(current) > 0 {
		return append(statements, current), nil
	}
	return statements, current
}

// updateSplitStatementsBlockDepth adjusts the BEGIN..END block-nesting counter and the
// inner expression-CASE counter for an identifier token.
//
// BEGIN opens a block only when something has already been emitted for the current
// statement and that statement is a procedural definition (a CREATE TRIGGER body);
// otherwise a `begin` used as an identifier or alias is left as plain content. A scalar
// `CASE ... END` inside a trigger body also uses a bare END; caseDepth tracks open CASE
// constructs (only while inside a block) so a bare END closes the CASE before it can
// close the BEGIN block, preventing a trigger body that contains a CASE expression from
// being mis-split at the next top-level semicolon.
//
// Takes blockDepth (int) and caseDepth (int) which are the current counters.
// Takes tok (token) which is the identifier under consideration.
// Takes nextValue (string) which is the token after tok (classifies END).
// Takes currentLength (int) which is the number of tokens already in the in-progress
// statement.
// Takes proceduralContext (bool) which is true when the in-flight statement is a
// procedural definition whose body may open a BEGIN block.
//
// Returns the updated blockDepth and caseDepth.
func updateSplitStatementsBlockDepth(blockDepth, caseDepth int, tok token, nextValue string, currentLength int, proceduralContext bool) (depth, cases int) {
	switch {
	case strings.EqualFold(tok.value, "BEGIN") && currentLength > 0 && proceduralContext:
		return blockDepth + 1, caseDepth
	case strings.EqualFold(tok.value, "CASE") && blockDepth > 0:
		return blockDepth, caseDepth + 1
	case strings.EqualFold(tok.value, "END"):
		return adjustSplitStatementsEnd(nextValue, blockDepth, caseDepth)
	default:
		return blockDepth, caseDepth
	}
}

// adjustSplitStatementsEnd resolves an END against the open CASE and block counters using
// the following token.
//
// END IF / END LOOP / END WHILE / END REPEAT close inner constructs that did not touch
// caseDepth; END CASE closes a statement-form CASE; a bare END closes an open
// expression-CASE first, then the block.
//
// Takes nextValue (string) which is the token immediately after the END.
// Takes blockDepth (int) which is the current BEGIN..END nesting depth.
// Takes caseDepth (int) which is the current open expression-CASE count.
//
// Returns depth (int) which is the updated BEGIN..END nesting depth.
// Returns cases (int) which is the updated open expression-CASE count.
func adjustSplitStatementsEnd(nextValue string, blockDepth, caseDepth int) (depth, cases int) {
	switch strings.ToUpper(nextValue) {
	case "IF", "LOOP", "WHILE", "REPEAT":
		return blockDepth, caseDepth
	case "CASE":
		if caseDepth > 0 {
			return blockDepth, caseDepth - 1
		}
		return blockDepth, caseDepth
	}
	if caseDepth > 0 {
		return blockDepth, caseDepth - 1
	}
	if blockDepth > 0 {
		return blockDepth - 1, caseDepth
	}
	return blockDepth, caseDepth
}

// classifyStatement identifies the statement kind from its leading keyword.
//
// Takes tokens ([]token) which holds the statement tokens.
//
// Returns statementKind which describes the classified shape.
func classifyStatement(tokens []token) statementKind {
	if len(tokens) == 0 {
		return statementKindUnknown
	}

	first := strings.ToUpper(tokens[0].value)

	switch first {
	case keywordCREATE:
		return classifyCreateStatement(tokens)
	case keywordDROP:
		return classifyDropStatement(tokens)
	case "ALTER":
		return statementKindAlterTable
	case keywordSELECT:
		return statementKindSelect
	case "INSERT", "REPLACE":
		return statementKindInsert
	case "UPDATE":
		return statementKindUpdate
	case "DELETE":
		return statementKindDelete
	case keywordVALUES:
		return statementKindValues
	case keywordWITH:
		return classifyWithStatement(tokens)
	}

	return statementKindUnknown
}

// classifyCreateStatement resolves the variant of a CREATE statement.
//
// Takes tokens ([]token) which begins with the CREATE keyword.
//
// Returns statementKind which is the specific create variant.
func classifyCreateStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	second := strings.ToUpper(tokens[1].value)
	switch second {
	case keywordTABLE:
		return statementKindCreateTable
	case "TEMP", "TEMPORARY":
		return classifyCreateTempStatement(tokens)
	case "VIEW":
		return statementKindCreateView
	case "INDEX", keywordUNIQUE:
		return statementKindCreateIndex
	case "VIRTUAL":
		return statementKindCreateVirtualTable
	case keywordTRIGGER:
		return statementKindCreateTrigger
	}

	return statementKindUnknown
}

// classifyCreateTempStatement resolves CREATE TEMP and CREATE TEMPORARY variants.
//
// Takes tokens ([]token) which begins with CREATE TEMP or CREATE TEMPORARY.
//
// Returns statementKind which is the specific temporary create variant.
func classifyCreateTempStatement(tokens []token) statementKind {
	if len(tokens) < minTokensForClassification {
		return statementKindUnknown
	}

	third := strings.ToUpper(tokens[2].value)
	switch third {
	case keywordTABLE:
		return statementKindCreateTable
	case "VIEW":
		return statementKindCreateView
	case keywordTRIGGER:
		return statementKindCreateTrigger
	}

	return statementKindUnknown
}

// classifyDropStatement resolves the variant of a DROP statement.
//
// Takes tokens ([]token) which begins with the DROP keyword.
//
// Returns statementKind which is the specific drop variant.
func classifyDropStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	second := strings.ToUpper(tokens[1].value)
	switch second {
	case keywordTABLE:
		return statementKindDropTable
	case "VIEW":
		return statementKindDropView
	case "INDEX":
		return statementKindDropIndex
	case keywordTRIGGER:
		return statementKindDropTrigger
	}

	return statementKindUnknown
}

var (
	// withBodyStatementKinds maps the keyword that follows a WITH clause to the matching
	// outer-statement kind.
	withBodyStatementKinds = map[string]statementKind{
		keywordSELECT: statementKindSelect,
		"INSERT":      statementKindInsert,
		"REPLACE":     statementKindInsert,
		"UPDATE":      statementKindUpdate,
		"DELETE":      statementKindDelete,
		keywordVALUES: statementKindValues,
	}
)

// classifyWithStatement resolves the outer statement that follows a WITH clause.
//
// Takes tokens ([]token) which begins with the WITH keyword.
//
// Returns statementKind which is the outer DML kind, defaulting to SELECT.
func classifyWithStatement(tokens []token) statementKind {
	depth := 0
	for _, tok := range tokens {
		switch tok.kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		case tokenIdentifier:
			if depth != 0 {
				continue
			}
			if kind, found := withBodyStatementKinds[strings.ToUpper(tok.value)]; found {
				return kind
			}
		}
	}
	return statementKindSelect
}

// current returns the token at the parser's current position.
//
// Returns token which is the active token, or a synthetic EOF token when past the end.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token immediately after the current position.
//
// Returns token which is the next token, or a synthetic EOF token when past the end.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// tokenAt returns the token at an absolute position.
//
// Takes index (int) which is the absolute token index to read.
//
// Returns token which is the token at the index, or a synthetic EOF token when out of
// range.
func (p *parser) tokenAt(index int) token {
	if index < 0 || index >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[index]
}

// advance consumes and returns the current token.
//
// Returns token which is the token at the position before advancing.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// expectKeyword consumes the current token when it matches any of the supplied keywords.
//
// Takes keywords (...string) which are the accepted keyword spellings.
//
// Returns token which is the consumed keyword token on success.
// Returns error when the current token is not an identifier matching one of the keywords.
func (p *parser) expectKeyword(keywords ...string) (token, error) {
	tok := p.current()
	if tok.kind != tokenIdentifier {
		return token{}, fmt.Errorf("expected keyword %v, got %q at position %d",
			keywords, tok.value, tok.position)
	}
	upper := strings.ToUpper(tok.value)
	for _, keyword := range keywords {
		if strings.EqualFold(upper, keyword) {
			p.position++
			return tok, nil
		}
	}
	return token{}, fmt.Errorf("expected keyword %v, got %q at position %d",
		keywords, tok.value, tok.position)
}

// matchKeyword consumes the current token if it matches the keyword.
//
// Takes keyword (string) which is the keyword to test against.
//
// Returns bool which is true when the keyword was matched and consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token matches the keyword without consuming it.
//
// Takes keyword (string) which is the keyword to test against.
//
// Returns bool which is true when the current token is the keyword.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the supplied keywords
// without consuming it.
//
// Takes keywords (...string) which are the candidate keyword spellings.
//
// Returns bool which is true when the current token matches one keyword.
func (p *parser) isAnyKeyword(keywords ...string) bool {
	tok := p.current()
	if tok.kind != tokenIdentifier {
		return false
	}
	upper := strings.ToUpper(tok.value)
	for _, keyword := range keywords {
		if strings.EqualFold(upper, keyword) {
			return true
		}
	}
	return false
}

// atEnd reports whether the parser has consumed every input token.
//
// Returns bool which is true when no more meaningful tokens remain.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// parseIdentifierOrKeyword consumes the next identifier or string literal and returns its
// value.
//
// Returns string which is the value of the consumed token.
// Returns error when the current token is neither an identifier nor a string literal.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// skipParenthesised consumes a balanced parenthesised group and discards its contents.
//
// Returns error when the opening parenthesis is missing or the group is unmatched.
func (p *parser) skipParenthesised() error {
	if p.current().kind != tokenLeftParen {
		return fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	depth := 1
	for depth > 0 && !p.atEnd() {
		switch p.current().kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		}
		p.advance()
	}
	if depth != 0 {
		return errUnmatchedParenthesis
	}
	return nil
}

// collectParenthesised consumes a balanced parenthesised group and returns the tokens
// between the parentheses.
//
// Returns []token which holds the inner tokens of the group.
// Returns error when the opening parenthesis is missing or the group is unmatched.
func (p *parser) collectParenthesised() ([]token, error) {
	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	var inner []token
	depth := 1
	for depth > 0 && !p.atEnd() {
		tok := p.current()
		switch tok.kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
			if depth == 0 {
				p.advance()
				return inner, nil
			}
		}
		inner = append(inner, tok)
		p.advance()
	}
	return nil, errUnmatchedParenthesis
}

// mustKeyword expects one of the supplied keywords and panics on mismatch.
//
// Takes keywords (...string) which are the accepted keyword spellings.
//
// Panics when no keyword matches the current token.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

// mustSkipParenthesised expects a balanced parenthesised group and panics on mismatch.
//
// Panics when no opening parenthesis is present or the group is unmatched.
func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

// registerParameterFromToken dispatches parameter registration based on the placeholder
// token kind.
//
// Takes parameterToken (token) which is the placeholder token.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears
// in the statement.
// Takes columnReference (*querier_dto.ColumnReference) which links the parameter to its
// target column when known.
// Takes castType (*querier_dto.SQLType) which holds an explicit CAST target when present.
//
// Returns int which is the resolved parameter index.
func (p *parser) registerParameterFromToken(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	switch parameterToken.kind {
	case tokenNumberedParam:
		return p.registerNumberedParameter(parameterToken, context, columnReference, castType)
	case tokenNamedParam:
		return p.registerNamedParameter(parameterToken, context, columnReference, castType)
	default:
		return p.registerSequentialParameter(context, columnReference, castType)
	}
}

// registerSequentialParameter assigns the next available index to an anonymous
// placeholder.
//
// Takes context (querier_dto.ParameterContext) which records where the parameter appears
// in the statement.
// Takes columnReference (*querier_dto.ColumnReference) which links the parameter to its
// target column when known.
// Takes castType (*querier_dto.SQLType) which holds an explicit CAST target when present.
//
// Returns int which is the newly assigned parameter index.
func (p *parser) registerSequentialParameter(
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	p.parameterCount++
	number := p.parameterCount
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:          number,
		Context:         context,
		ColumnReference: columnReference,
		CastType:        castType,
	})
	return number
}

// registerNumberedParameter records a `?N` placeholder reference.
//
// Takes parameterToken (token) which carries the `?N` literal.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears
// in the statement.
// Takes columnReference (*querier_dto.ColumnReference) which links the parameter to its
// target column when known.
// Takes castType (*querier_dto.SQLType) which holds an explicit CAST target when present.
//
// Returns int which is the parameter index parsed from the token.
func (p *parser) registerNumberedParameter(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	number, parseError := strconv.Atoi(parameterToken.value[1:])
	if parseError != nil || number <= 0 {
		return p.registerSequentialParameter(context, columnReference, castType)
	}
	if number > p.parameterCount {
		p.parameterCount = number
	}
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:          number,
		Context:         context,
		ColumnReference: columnReference,
		CastType:        castType,
	})
	return number
}

// registerNamedParameter records a named placeholder, reusing the index when the name has
// been seen before.
//
// Takes parameterToken (token) which carries the named placeholder.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears
// in the statement.
// Takes columnReference (*querier_dto.ColumnReference) which links the parameter to its
// target column when known.
// Takes castType (*querier_dto.SQLType) which holds an explicit CAST target when present.
//
// Returns int which is the index associated with the parameter name.
func (p *parser) registerNamedParameter(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	name := parameterToken.value[1:]
	if existingNumber, exists := p.namedParameterMap[name]; exists {
		p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
			Number:          existingNumber,
			Name:            name,
			Context:         context,
			ColumnReference: columnReference,
			CastType:        castType,
		})
		return existingNumber
	}
	p.parameterCount++
	number := p.parameterCount
	p.namedParameterMap[name] = number
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:          number,
		Name:            name,
		Context:         context,
		ColumnReference: columnReference,
		CastType:        castType,
	})
	return number
}
