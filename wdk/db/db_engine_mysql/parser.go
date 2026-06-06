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

package db_engine_mysql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// statementKind enumerates the high-level categories of SQL statement that the splitter
// and classifier recognise.
type statementKind uint8

const (
	// statementKindCreateTable identifies a CREATE TABLE statement.
	statementKindCreateTable statementKind = iota

	// statementKindDropTable identifies a DROP TABLE statement.
	statementKindDropTable

	// statementKindAlterTable identifies an ALTER TABLE statement.
	statementKindAlterTable

	// statementKindCreateView identifies a CREATE VIEW statement.
	statementKindCreateView

	// statementKindDropView identifies a DROP VIEW statement.
	statementKindDropView

	// statementKindCreateIndex identifies a CREATE INDEX statement.
	statementKindCreateIndex

	// statementKindDropIndex identifies a DROP INDEX statement.
	statementKindDropIndex

	// statementKindCreateTrigger identifies a CREATE TRIGGER statement.
	statementKindCreateTrigger

	// statementKindDropTrigger identifies a DROP TRIGGER statement.
	statementKindDropTrigger

	// statementKindCreateDatabase identifies a CREATE DATABASE or SCHEMA statement.
	statementKindCreateDatabase

	// statementKindDropDatabase identifies a DROP DATABASE or SCHEMA statement.
	statementKindDropDatabase

	// statementKindSelect identifies a SELECT statement.
	statementKindSelect

	// statementKindInsert identifies an INSERT statement.
	statementKindInsert

	// statementKindUpdate identifies an UPDATE statement.
	statementKindUpdate

	// statementKindDelete identifies a DELETE statement.
	statementKindDelete

	// statementKindValues identifies a standalone VALUES statement.
	statementKindValues

	// statementKindReplace identifies a REPLACE statement.
	statementKindReplace

	// statementKindCreateFunction identifies a CREATE FUNCTION or PROCEDURE statement.
	statementKindCreateFunction

	// statementKindDropFunction identifies a DROP FUNCTION or PROCEDURE statement.
	statementKindDropFunction

	// statementKindUnknown identifies a statement whose kind could not be determined.
	statementKindUnknown

	// statementKindCount is the sentinel count of recognised statement kinds.
	statementKindCount
)

const (
	// minTokensForCreateOrReplace is the minimum number of tokens required for a CREATE OR
	// REPLACE prefix to be considered well-formed.
	minTokensForCreateOrReplace = 4

	// indexAfterOrReplace is the token index following CREATE OR REPLACE.
	indexAfterOrReplace = 3

	// defaultMaxParseDepth caps every user-driven recursion in the parser.
	//
	// The cap covers analyseSelect (subquery/CTE nesting) and the expression precedence
	// chain (parenthesis nesting). It is essential because Go raises a fatal,
	// non-recoverable stack overflow that the engine's recover guards cannot catch, so
	// deeply nested input would otherwise crash the host process. 256 is far below the
	// overflow threshold yet out of the way for realistic queries; callers may override it
	// with WithMaxParseDepth.
	defaultMaxParseDepth = 256
)

var (
	// errUnmatchedParenthesis is returned when the parser encounters a parenthesis group
	// that is not closed before the end of the token stream.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")

	// errAnalysisDepthExceeded is the sentinel returned when parser recursion exceeds the
	// configured maximum parse depth.
	errAnalysisDepthExceeded = errors.New("mysql: analysis recursion depth exceeded")
)

// parsedStatement bundles a token slice with its classified statement kind.
type parsedStatement struct {
	// tokens holds the raw token slice for the statement.
	tokens []token

	// kind records the classified statement category.
	kind statementKind
}

// IsParsedStatement marks the type as conforming to the parsed statement marker
// interface.
func (*parsedStatement) IsParsedStatement() {}

// parser holds the mutable state required to walk a single SQL statement and accumulate
// parameter and table references.
type parser struct {
	// namedParameterMap maps named parameter labels to their assigned sequential numbers.
	namedParameterMap map[string]int

	// insertProjectionTable is the INSERT target table whose columns the current INSERT ...
	// SELECT projection items are assigned to.
	//
	// Empty when not inside an INSERT ... SELECT projection.
	insertProjectionTable string

	// tokens is the token stream being parsed.
	tokens []token

	// parameterRefs accumulates references to bind parameters encountered during parsing.
	parameterRefs []querier_dto.RawParameterReference

	// rawDerivedTables collects parsed derived (subquery) table references.
	rawDerivedTables []querier_dto.RawDerivedTableReference

	// predicateSubqueries collects subqueries found in a token-scanned predicate position
	// (WHERE / HAVING / JOIN ON), so the domain layer can resolve each one's parameters
	// against the subquery's own scope rather than the parent.
	predicateSubqueries []*querier_dto.RawQueryAnalysis

	// rawTableValuedFunctions collects parsed table-valued function references appearing in
	// FROM clauses.
	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	// insertProjectionColumns is the ordered INSERT target column list whose entries the
	// INSERT ... SELECT projection items map onto positionally.
	//
	// Nil when not inside an INSERT ... SELECT projection.
	insertProjectionColumns []string

	// position is the index of the next token to consume.
	position int

	// parameterCount is the running count of parameters assigned so far.
	parameterCount int

	// analysisDepth bounds recursion through analyseSelect across compound branches, CTE
	// bodies, derived tables, scalar subqueries, and EXISTS / IN-subquery expressions. Child
	// parsers inherit the parent depth so the global cap holds across instances.
	analysisDepth int

	// expressionDepth bounds recursion through the expression precedence chain so deeply
	// nested parentheses cannot overflow the stack.
	expressionDepth int

	// maxParseDepth is the effective cap for analysisDepth and expressionDepth. Zero means
	// defaultMaxParseDepth; the engine sets it from the dialect and child parsers inherit
	// it.
	maxParseDepth int

	// insertProjectionIndex is the zero-based ordinal of the projection item currently being
	// parsed within the INSERT ... SELECT select list.
	insertProjectionIndex int

	// hasForUpdate records whether the statement carries a FOR UPDATE / FOR SHARE locking
	// clause.
	hasForUpdate bool

	// hasDataModifyingCTE records whether any CTE in the statement performs a data-modifying
	// operation.
	hasDataModifyingCTE bool
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

// statementSplitter walks a token stream and divides it into separate statements,
// honouring DELIMITER commands.
type statementSplitter struct {
	// statements holds the completed statement slices produced so far.
	statements [][]token

	// current accumulates tokens for the in-progress statement.
	current []token

	// customDelimiter overrides the default semicolon terminator when set.
	customDelimiter string

	// tokens is the input token stream being split.
	tokens []token

	// position is the index of the next token to consume.
	position int

	// blockDepth tracks open BEGIN...END blocks. MySQL normally uses the DELIMITER directive
	// to escape trigger/procedure bodies, but migrations that omit DELIMITER must still
	// parse correctly; counting blocks defensively keeps embedded semicolons attached to the
	// enclosing statement either way.
	blockDepth int

	// caseDepth tracks open scalar CASE...END expressions inside a BEGIN...END body.
	//
	// A scalar CASE ends with a bare END, so without this counter that inner END would
	// wrongly decrement blockDepth and the next top-level semicolon would split the
	// procedure body mid-statement. caseDepth is only incremented while blockDepth is
	// greater than zero.
	caseDepth int
}

// splitStatements divides a token stream into one slice per statement.
//
// Takes tokens ([]token) which is the lexed token stream.
//
// Returns [][]token which contains one entry per discovered statement.
func splitStatements(tokens []token) [][]token {
	splitter := &statementSplitter{tokens: tokens}
	splitter.run()
	return splitter.statements
}

// run drives the splitter loop until the token stream is exhausted.
func (s *statementSplitter) run() {
	for s.position < len(s.tokens) {
		tok := s.tokens[s.position]

		if tok.kind == tokenEOF {
			break
		}
		if s.handleDelimiterCommand(tok) {
			continue
		}
		if s.handleStatementBoundary(tok) {
			continue
		}
		s.trackBlockKeyword(tok)
		s.current = append(s.current, tok)
		s.position++
	}
	s.flushCurrent()
}

// trackBlockKeyword adjusts blockDepth and caseDepth on bare BEGIN / CASE / END
// identifiers so handleStatementBoundary knows when a semicolon belongs to the enclosing
// statement.
//
// BEGIN only opens a block when the in-flight statement is a procedural definition
// (CREATE FUNCTION / PROCEDURE / TRIGGER) whose body may legitimately contain a bare
// BEGIN...END block. A `begin` used as a column name or alias in an ordinary statement
// (e.g. `SELECT 1 AS begin; SELECT 2;`) must not open a block and swallow the next
// statement. A leading "BEGIN;" is the MySQL transaction marker, so the in-flight
// statement must also be non-empty.
//
// A scalar CASE...END expression inside a procedural body also ends with a bare END.
// caseDepth records open CASE constructs (only while inside a block) so a bare END closes
// the CASE before it can close the BEGIN block. END decrements blockDepth ONLY when it
// closes the outer BEGIN..END block. The token stream sees END followed by
// IF/LOOP/WHILE/REPEAT as two separate identifiers; in that case the inner control
// structure is closing, not the outer block, so depth must stay.
//
// Takes tok (token) which is the identifier token under consideration.
func (s *statementSplitter) trackBlockKeyword(tok token) {
	if tok.kind != tokenIdentifier {
		return
	}
	if strings.EqualFold(tok.value, "BEGIN") && len(s.current) > 0 && isProceduralStatement(s.current) {
		s.blockDepth++
		return
	}
	if strings.EqualFold(tok.value, "CASE") && s.blockDepth > 0 {
		s.caseDepth++
		return
	}
	if strings.EqualFold(tok.value, "END") {
		s.adjustEndDepth()
	}
}

// adjustEndDepth resolves an END identifier against the open CASE and block counters.
//
// `END IF` / `END LOOP` / `END WHILE` / `END REPEAT` close inner procedural constructs
// that did not touch caseDepth, so they are ignored. `END CASE` and a bare END close an
// open scalar CASE first (so its END cannot leak through to the block), and a bare END
// otherwise closes the surrounding BEGIN block.
func (s *statementSplitter) adjustEndDepth() {
	switch strings.ToUpper(s.peekNextValue()) {
	case "IF", "LOOP", "WHILE", "REPEAT":
		return
	case "CASE":
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

// peekNextValue returns the value of the token immediately after the current position, or
// the empty string when the current END is the last token. The token stream does not
// include whitespace so a single position offset suffices.
//
// Returns string which is the next token's value, or empty at end of input.
func (s *statementSplitter) peekNextValue() string {
	next := s.position + 1
	if next >= len(s.tokens) {
		return ""
	}
	return s.tokens[next].value
}

// isProceduralStatement reports whether the in-flight statement is a procedural
// definition whose body may contain a bare BEGIN...END block.
//
// The qualifying forms are CREATE FUNCTION / PROCEDURE / TRIGGER, optionally with OR
// REPLACE, DEFINER, or SQL SECURITY clauses in between. Any other leading statement (for
// example SELECT/INSERT/UPDATE or CREATE TABLE/VIEW) returns false so a `begin`
// identifier is treated as a plain name, not a block opener. The check is deliberately
// tolerant of the intervening clauses MySQL allows between CREATE and the object keyword
// (`CREATE DEFINER = 'user'@'host' PROCEDURE ...`): it only requires the statement to
// begin with CREATE and to contain FUNCTION/PROCEDURE/TRIGGER as an identifier somewhere
// before the BEGIN that triggered the check.
//
// Takes current ([]token) which is the tokens accumulated for the statement so far.
//
// Returns bool which is true for a procedural-body-bearing statement.
func isProceduralStatement(current []token) bool {
	if len(current) == 0 || current[0].kind != tokenIdentifier || !strings.EqualFold(current[0].value, "CREATE") {
		return false
	}
	for index := 1; index < len(current); index++ {
		if current[index].kind != tokenIdentifier {
			continue
		}
		switch strings.ToUpper(current[index].value) {
		case "FUNCTION", "PROCEDURE", "TRIGGER":
			return true
		}
	}
	return false
}

// handleDelimiterCommand consumes a DELIMITER directive, flushing the current statement
// and recording the new custom delimiter (or clearing it when reset to a semicolon).
//
// Takes tok (token) which is the current token under inspection.
//
// Returns bool reporting whether a DELIMITER directive was consumed.
func (s *statementSplitter) handleDelimiterCommand(tok token) bool {
	if tok.kind != tokenIdentifier || !strings.EqualFold(tok.value, "DELIMITER") {
		return false
	}
	s.flushCurrent()
	s.position++
	if s.position < len(s.tokens) {
		delimiter := s.tokens[s.position].value
		if delimiter == ";" {
			s.customDelimiter = ""
		} else {
			s.customDelimiter = delimiter
		}
	}
	s.position++
	return true
}

// handleStatementBoundary detects the end of the current statement.
//
// Takes tok (token) which is the current token under inspection.
//
// Returns bool reporting whether the statement boundary was consumed.
func (s *statementSplitter) handleStatementBoundary(tok token) bool {
	if s.customDelimiter != "" {
		if (tok.kind == tokenIdentifier || tok.kind == tokenOperator) && tok.value == s.customDelimiter {
			s.flushCurrent()
			s.position++
			return true
		}
		return false
	}
	if tok.kind == tokenSemicolon {
		if s.blockDepth > 0 {
			return false
		}
		s.flushCurrent()
		s.position++
		return true
	}
	return false
}

// flushCurrent appends the in-progress statement to the result slice when any tokens have
// been accumulated.
func (s *statementSplitter) flushCurrent() {
	if len(s.current) > 0 {
		s.statements = append(s.statements, s.current)
		s.current = nil
	}
}

var (
	// firstWordClassifiers dispatches on the leading keyword for statements whose kind
	// requires further inspection beyond the first word.
	firstWordClassifiers = map[string]func([]token) statementKind{
		keywordCREATE: classifyCreateStatement,
		keywordDROP:   classifyDropStatement,
		"ALTER":       classifyAlterStatement,
		keywordWITH:   classifyWithStatement,
	}

	// firstWordStaticKinds maps leading keywords whose statement kind is fully determined by
	// the first word.
	firstWordStaticKinds = map[string]statementKind{
		keywordSELECT:  statementKindSelect,
		"INSERT":       statementKindInsert,
		"UPDATE":       statementKindUpdate,
		"DELETE":       statementKindDelete,
		keywordVALUES:  statementKindValues,
		keywordREPLACE: statementKindReplace,
	}
)

// classifyStatement returns the statementKind for a tokenised statement.
//
// Takes tokens ([]token) which is the tokenised statement to classify.
//
// Returns statementKind which is the inferred kind, or statementKindUnknown when no rule
// matches.
func classifyStatement(tokens []token) statementKind {
	if len(tokens) == 0 {
		return statementKindUnknown
	}

	first := strings.ToUpper(tokens[0].value)

	if kind, found := firstWordStaticKinds[first]; found {
		return kind
	}

	if classifier, found := firstWordClassifiers[first]; found {
		return classifier(tokens)
	}

	return statementKindUnknown
}

var (
	// createObjectKinds maps the noun following CREATE (optionally after modifiers) to the
	// matching statementKind.
	createObjectKinds = map[string]statementKind{
		keywordTABLE:     statementKindCreateTable,
		"VIEW":           statementKindCreateView,
		"INDEX":          statementKindCreateIndex,
		keywordUNIQUE:    statementKindCreateIndex,
		"TRIGGER":        statementKindCreateTrigger,
		keywordDATABASE:  statementKindCreateDatabase,
		keywordSCHEMA:    statementKindCreateDatabase,
		keywordFUNCTION:  statementKindCreateFunction,
		keywordPROCEDURE: statementKindCreateFunction,
	}
)

// classifyCreateStatement returns the statementKind for a CREATE statement.
//
// Takes tokens ([]token) which is the tokenised CREATE statement.
//
// Returns statementKind which is the kind of object being created, or
// statementKindUnknown when the object noun is unrecognised.
func classifyCreateStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	index := skipCreatePrefixes(tokens)
	if index >= len(tokens) {
		return statementKindUnknown
	}

	upper := strings.ToUpper(tokens[index].value)
	if kind, found := createObjectKinds[upper]; found {
		return kind
	}

	return statementKindUnknown
}

// skipCreatePrefixes advances past CREATE modifiers such as OR REPLACE and TEMPORARY to
// locate the object-kind keyword.
//
// Takes tokens ([]token) which is the tokenised CREATE statement.
//
// Returns int which is the index of the object-kind keyword, or len(tokens) when the
// prefix is malformed.
func skipCreatePrefixes(tokens []token) int {
	index := 1
	upper := strings.ToUpper(tokens[index].value)

	if upper == "OR" {
		if len(tokens) < minTokensForCreateOrReplace {
			return len(tokens)
		}
		index = indexAfterOrReplace
		upper = strings.ToUpper(tokens[index].value)
	}

	if upper == "TEMP" || upper == "TEMPORARY" {
		if index+1 >= len(tokens) {
			return len(tokens)
		}
		index++
	}

	return index
}

// classifyDropStatement returns the statementKind for a DROP statement.
//
// Takes tokens ([]token) which is the tokenised DROP statement.
//
// Returns statementKind which is the kind of object being dropped, or
// statementKindUnknown when the object noun is unrecognised.
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
	case "TRIGGER":
		return statementKindDropTrigger
	case keywordDATABASE, keywordSCHEMA:
		return statementKindDropDatabase
	case keywordFUNCTION, keywordPROCEDURE:
		return statementKindDropFunction
	}

	return statementKindUnknown
}

// classifyAlterStatement returns the statementKind for an ALTER statement.
//
// Takes tokens ([]token) which is the tokenised ALTER statement.
//
// Returns statementKind which is statementKindAlterTable for ALTER TABLE and
// statementKindUnknown for every other ALTER target.
func classifyAlterStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	if strings.EqualFold(tokens[1].value, keywordTABLE) {
		return statementKindAlterTable
	}

	return statementKindUnknown
}

// classifyWithStatement returns the statementKind for a WITH statement by scanning for
// the first DML keyword at outer scope.
//
// Takes tokens ([]token) which is the tokenised WITH statement.
//
// Returns statementKind which is the kind of the trailing DML, defaulting to
// statementKindSelect.
func classifyWithStatement(tokens []token) statementKind {
	depth := 0
	for _, tok := range tokens {
		if tok.kind == tokenLeftParen {
			depth++
			continue
		}
		if tok.kind == tokenRightParen {
			depth--
			continue
		}
		if depth != 0 || tok.kind != tokenIdentifier {
			continue
		}
		if kind, matched := classifyDMLKeyword(tok.value); matched {
			return kind
		}
	}
	return statementKindSelect
}

var (
	// dmlKeywords maps the data-manipulation keywords recognised inside a WITH statement to
	// their matching statementKind.
	dmlKeywords = map[string]statementKind{
		keywordSELECT:  statementKindSelect,
		"INSERT":       statementKindInsert,
		"UPDATE":       statementKindUpdate,
		"DELETE":       statementKindDelete,
		keywordVALUES:  statementKindValues,
		keywordREPLACE: statementKindReplace,
	}
)

// classifyDMLKeyword resolves a token value to its DML statementKind.
//
// Takes value (string) which is the candidate keyword.
//
// Returns statementKind which is the matched kind when found.
// Returns bool which reports whether the lookup succeeded.
func classifyDMLKeyword(value string) (statementKind, bool) {
	kind, matched := dmlKeywords[strings.ToUpper(value)]
	return kind, matched
}

// current returns the token at the parser's current position, or an EOF sentinel when the
// stream is exhausted.
//
// Returns token which is the current token or an EOF sentinel.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token immediately after the current position without advancing the
// parser.
//
// Returns token which is the look-ahead token or an EOF sentinel.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// advance consumes the current token and moves the parser forward.
//
// Returns token which is the token that was at the current position.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// expectKeyword consumes the current token when it matches one of the supplied keywords,
// otherwise returns an error.
//
// Takes keywords (...string) which lists the acceptable keywords.
//
// Returns token which is the matched token on success.
// Returns error when the current token is not one of the keywords.
func (p *parser) expectKeyword(keywords ...string) (token, error) {
	tok := p.current()
	if tok.kind != tokenIdentifier {
		return token{}, fmt.Errorf("expected keyword %v, got %q at position %d",
			keywords, tok.value, tok.position)
	}
	for _, keyword := range keywords {
		if strings.EqualFold(tok.value, keyword) {
			p.position++
			return tok, nil
		}
	}
	return token{}, fmt.Errorf("expected keyword %v, got %q at position %d",
		keywords, tok.value, tok.position)
}

// matchKeyword consumes the current token when it equals the keyword.
//
// Takes keyword (string) which is the keyword to match.
//
// Returns bool reporting whether the keyword was matched and consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token matches the given keyword.
//
// Takes keyword (string) which is the keyword to test.
//
// Returns bool reporting whether the keyword matches the current token.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the keywords.
//
// Takes keywords (...string) which lists the candidate keywords.
//
// Returns bool reporting whether a match was found.
func (p *parser) isAnyKeyword(keywords ...string) bool {
	tok := p.current()
	if tok.kind != tokenIdentifier {
		return false
	}
	for _, keyword := range keywords {
		if strings.EqualFold(tok.value, keyword) {
			return true
		}
	}
	return false
}

// atEnd reports whether the parser has consumed all input tokens.
//
// Returns bool which is true when no further tokens remain.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// parseIdentifierOrKeyword consumes an identifier or string literal and returns its
// value.
//
// Returns string which is the consumed identifier or string value.
// Returns error when the current token is neither an identifier nor a string literal.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// parseSchemaQualifiedName parses a possibly database-qualified name.
//
// For MySQL, the schema component corresponds to the database name.
//
// Returns schema (string) which is the database qualifier or empty when the name was
// unqualified.
// Returns name (string) which is the parsed identifier.
// Returns err (error) when an identifier cannot be parsed at the current position.
func (p *parser) parseSchemaQualifiedName() (schema string, name string, err error) {
	first, parseError := p.parseIdentifierOrKeyword()
	if parseError != nil {
		return "", "", parseError
	}

	if p.current().kind == tokenDot {
		p.advance()
		second, secondError := p.parseIdentifierOrKeyword()
		if secondError != nil {
			return "", "", secondError
		}
		return first, second, nil
	}

	return "", first, nil
}

// skipParenthesised consumes a balanced parenthesised group at the current position
// without retaining its contents.
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

// collectParenthesised consumes a balanced parenthesised group and returns the tokens it
// contained, excluding the outer parentheses.
//
// Returns []token which is the slice of inner tokens.
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

// mustKeyword consumes one of the supplied keywords and panics when none match.
//
// Takes keywords (...string) which lists the acceptable keywords.
//
// Panics when the current token does not match any of the keywords.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

// mustSkipParenthesised skips a parenthesised group and panics on failure.
//
// Panics when the group is malformed or the opening parenthesis is missing.
func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

// mustSchemaQualifiedName parses a schema-qualified name and panics on failure.
//
// Returns schema (string) which is the database qualifier or empty.
// Returns name (string) which is the parsed identifier.
//
// Panics when no identifier can be parsed at the current position.
func (p *parser) mustSchemaQualifiedName() (schema string, name string) {
	schema, name, err := p.parseSchemaQualifiedName()
	if err != nil {
		panic(fmt.Errorf("mustSchemaQualifiedName: %w", err))
	}
	return schema, name
}

// functionArgumentMetadata carries the enclosing function name and the zero-based
// argument slot of a placeholder that sits inside a function or table-valued function
// argument list.
//
// The analyser pairs these two facts to look up the matched function signature's declared
// argument type and back-propagate it onto an otherwise untyped placeholder. Both fields
// are zero-valued (empty name, zero ordinal) for the common non-function-argument case,
// leaving the placeholder unaffected.
type functionArgumentMetadata struct {
	// enclosingFunctionName is the lower-cased, optionally schema-qualified name of the
	// function whose argument list the placeholder sits in, recorded exactly as the engine
	// records the function name on RawTableValuedFunctionReference / FunctionCallExpression.
	// Empty when the placeholder is not a function argument.
	enclosingFunctionName string

	// argumentOrdinal is the zero-based position of the placeholder among the enclosing
	// call's top-level arguments (the argument slot, not the parameter number). Meaningful
	// only when enclosingFunctionName is non-empty.
	argumentOrdinal int
}

// registerParameterFromToken records a bind parameter encountered in the token stream,
// dispatching to named or sequential registration.
//
// Takes parameterToken (token) which is the parameter token being recorded.
// Takes context (querier_dto.ParameterContext) which describes the syntactic context
// where the parameter occurs.
// Takes columnReference (*querier_dto.ColumnReference) which is the column the parameter
// is bound against when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when the
// parameter carries one.
//
// Returns int which is the assigned parameter number.
func (p *parser) registerParameterFromToken(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	return p.registerParameterFromTokenWithFunctionArgument(
		parameterToken, context, columnReference, castType, functionArgumentMetadata{})
}

// registerParameterFromTokenWithFunctionArgument records a bind parameter and
// additionally stamps the enclosing function/TVF argument metadata onto the resulting
// reference.
//
// Used by the flat-scan WHERE/HAVING resolver, which can compute the enclosing function
// name and argument ordinal directly from the token stream at registration time. The
// common registerParameterFromToken path forwards an empty functionArgumentMetadata so
// non-function placeholders are unaffected.
//
// Takes parameterToken (token) which is the parameter token being recorded.
// Takes context (querier_dto.ParameterContext) which describes the syntactic context.
// Takes columnReference (*querier_dto.ColumnReference) which is the bound column when
// known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when present.
// Takes functionArgument (functionArgumentMetadata) which carries the enclosing function
// name and argument ordinal, empty when the placeholder is not a function argument.
//
// Returns int which is the assigned parameter number.
func (p *parser) registerParameterFromTokenWithFunctionArgument(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
	functionArgument functionArgumentMetadata,
) int {
	if parameterToken.kind == tokenNamedParam {
		return p.registerNamedParameter(parameterToken, context, columnReference, castType, functionArgument)
	}
	if parameterToken.kind == tokenQuestionMark && len(parameterToken.value) > 1 {
		return p.registerNumberedQuestionMark(parameterToken, context, columnReference, castType, functionArgument)
	}
	return p.registerSequentialParameter(context, columnReference, castType, functionArgument)
}

// registerNumberedQuestionMark records a '?N' positional bind parameter whose explicit
// index permits the same number to appear multiple times in the SQL while binding to a
// single argument slot. Reuse of an index appends another reference with the same Number
// so downstream merging treats them as one parameter.
//
// Takes parameterToken (token) which holds the '?N' token with its numeric suffix.
// Takes context (querier_dto.ParameterContext) which describes the syntactic context
// where the parameter occurs.
// Takes columnReference (*querier_dto.ColumnReference) which is the column the parameter
// is bound against when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when the
// parameter carries one.
// Takes functionArgument (functionArgumentMetadata) which carries the enclosing function
// name and argument ordinal, empty when the placeholder is not a function argument.
//
// Returns int which is the parameter number from the token's numeric suffix.
func (p *parser) registerNumberedQuestionMark(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
	functionArgument functionArgumentMetadata,
) int {
	number, parseError := strconv.Atoi(parameterToken.value[1:])
	if parseError != nil || number <= 0 {
		return p.registerSequentialParameter(context, columnReference, castType, functionArgument)
	}
	if number > p.parameterCount {
		p.parameterCount = number
	}
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:                number,
		Context:               context,
		ColumnReference:       columnReference,
		CastType:              castType,
		EnclosingFunctionName: functionArgument.enclosingFunctionName,
		ArgumentOrdinal:       functionArgument.argumentOrdinal,
	})
	return number
}

// registerSequentialParameter records a positional bind parameter and returns the freshly
// assigned number.
//
// Takes context (querier_dto.ParameterContext) which describes the syntactic context
// where the parameter occurs.
// Takes columnReference (*querier_dto.ColumnReference) which is the column the parameter
// is bound against when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when the
// parameter carries one.
// Takes functionArgument (functionArgumentMetadata) which carries the enclosing function
// name and argument ordinal, empty when the placeholder is not a function argument.
//
// Returns int which is the newly assigned parameter number.
func (p *parser) registerSequentialParameter(
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
	functionArgument functionArgumentMetadata,
) int {
	p.parameterCount++
	number := p.parameterCount
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:                number,
		Context:               context,
		ColumnReference:       columnReference,
		CastType:              castType,
		EnclosingFunctionName: functionArgument.enclosingFunctionName,
		ArgumentOrdinal:       functionArgument.argumentOrdinal,
	})
	return number
}

// registerNamedParameter records a named bind parameter, reusing the previously assigned
// number when the same name has been seen before.
//
// Takes parameterToken (token) which is the named parameter token.
// Takes context (querier_dto.ParameterContext) which describes the syntactic context
// where the parameter occurs.
// Takes columnReference (*querier_dto.ColumnReference) which is the column the parameter
// is bound against when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when the
// parameter carries one.
// Takes functionArgument (functionArgumentMetadata) which carries the enclosing function
// name and argument ordinal, empty when the placeholder is not a function argument.
//
// Returns int which is the assigned (or reused) parameter number.
func (p *parser) registerNamedParameter(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
	functionArgument functionArgumentMetadata,
) int {
	name := parameterToken.value[1:]
	if existingNumber, exists := p.namedParameterMap[name]; exists {
		p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
			Number:                existingNumber,
			Name:                  name,
			Context:               context,
			ColumnReference:       columnReference,
			CastType:              castType,
			EnclosingFunctionName: functionArgument.enclosingFunctionName,
			ArgumentOrdinal:       functionArgument.argumentOrdinal,
		})
		return existingNumber
	}
	p.parameterCount++
	number := p.parameterCount
	p.namedParameterMap[name] = number
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:                number,
		Name:                  name,
		Context:               context,
		ColumnReference:       columnReference,
		CastType:              castType,
		EnclosingFunctionName: functionArgument.enclosingFunctionName,
		ArgumentOrdinal:       functionArgument.argumentOrdinal,
	})
	return number
}

// isParameterToken reports whether the token kind represents a bind parameter
// placeholder.
//
// Takes kind (tokenKind) which is the token kind to test.
//
// Returns bool reporting whether the token denotes a bind parameter.
func isParameterToken(kind tokenKind) bool {
	return kind == tokenQuestionMark || kind == tokenNamedParam
}
