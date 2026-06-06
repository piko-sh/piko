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

package db_engine_duckdb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// statementKind tags a parsed SQL statement with its top-level category.
type statementKind uint8

const (
	// statementKindCreateTable identifies CREATE TABLE statements.
	statementKindCreateTable statementKind = iota

	// statementKindDropTable identifies DROP TABLE statements.
	statementKindDropTable

	// statementKindAlterTable identifies ALTER TABLE statements.
	statementKindAlterTable

	// statementKindCreateView identifies CREATE VIEW statements.
	statementKindCreateView

	// statementKindDropView identifies DROP VIEW statements.
	statementKindDropView

	// statementKindCreateIndex identifies CREATE INDEX statements.
	statementKindCreateIndex

	// statementKindDropIndex identifies DROP INDEX statements.
	statementKindDropIndex

	// statementKindCreateType identifies CREATE TYPE statements.
	statementKindCreateType

	// statementKindAlterType identifies ALTER TYPE statements.
	statementKindAlterType

	// statementKindDropType identifies DROP TYPE statements.
	statementKindDropType

	// statementKindCreateFunction identifies CREATE FUNCTION statements.
	statementKindCreateFunction

	// statementKindDropFunction identifies DROP FUNCTION statements.
	statementKindDropFunction

	// statementKindCreateMacro identifies CREATE MACRO statements.
	statementKindCreateMacro

	// statementKindDropMacro identifies DROP MACRO statements.
	statementKindDropMacro

	// statementKindCreateSchema identifies CREATE SCHEMA statements.
	statementKindCreateSchema

	// statementKindDropSchema identifies DROP SCHEMA statements.
	statementKindDropSchema

	// statementKindCreateSequence identifies CREATE SEQUENCE statements.
	statementKindCreateSequence

	// statementKindDropSequence identifies DROP SEQUENCE statements.
	statementKindDropSequence

	// statementKindAlterSequence identifies ALTER SEQUENCE statements.
	statementKindAlterSequence

	// statementKindComment identifies COMMENT ON statements.
	statementKindComment

	// statementKindInstall identifies INSTALL extension statements.
	statementKindInstall

	// statementKindLoad identifies LOAD extension statements.
	statementKindLoad

	// statementKindSelect identifies SELECT statements.
	statementKindSelect

	// statementKindInsert identifies INSERT statements.
	statementKindInsert

	// statementKindUpdate identifies UPDATE statements.
	statementKindUpdate

	// statementKindDelete identifies DELETE statements.
	statementKindDelete

	// statementKindValues identifies bare VALUES statements.
	statementKindValues

	// statementKindUnknown marks statements the classifier could not name.
	statementKindUnknown

	// statementKindCount is the sentinel one past the last statement kind.
	statementKindCount
)

const (
	// minTokensForCreateOrReplace is the minimum token count needed to recognise a CREATE OR
	// REPLACE prefix.
	minTokensForCreateOrReplace = 4

	// indexAfterOrReplace is the token index immediately following the CREATE OR REPLACE
	// prefix.
	indexAfterOrReplace = 3

	// schemaQualifiedCallLookahead is the offset from a schema identifier to the opening
	// parenthesis of a schema-qualified call (schema . name (), so the '(' sits three tokens
	// past the schema token.
	schemaQualifiedCallLookahead = 3

	// defaultMaxParseDepth caps every user-driven recursion in the parser.
	//
	// The capped recursions are analyseSelect (subquery and CTE nesting), the expression
	// precedence chain (parenthesis nesting), and compound column-type nesting. The cap is
	// essential because Go raises a fatal, non-recoverable stack overflow that the engine's
	// recover guards cannot catch, so deeply nested input would otherwise crash the host
	// process. 256 is far below the overflow threshold yet out of the way for realistic
	// queries; callers may override it with WithMaxParseDepth.
	defaultMaxParseDepth = 256
)

var (
	// errUnmatchedParenthesis is returned when a parenthesis scan reaches the end of input
	// without closing every opened group.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")

	// errAnalysisDepthExceeded is the sentinel returned when parser recursion exceeds the
	// configured maximum parse depth.
	errAnalysisDepthExceeded = errors.New("duckdb: analysis recursion depth exceeded")

	// errMalformedFunctionArguments is the sentinel returned when a function or macro
	// argument list contains a token the argument parser cannot consume, which would
	// otherwise leave the argument-list loop unable to make progress.
	errMalformedFunctionArguments = errors.New("duckdb: malformed function argument list")
)

// parsedStatement holds the tokens and classified kind for one SQL statement extracted
// from a multi-statement script.
type parsedStatement struct {
	// tokens are the lexical tokens that make up this statement, in order.
	tokens []token

	// kind is the classified statement category.
	kind statementKind
}

// IsParsedStatement marks the type as a parsed statement for interface dispatch.
func (*parsedStatement) IsParsedStatement() {}

// parser holds the state required to walk a flat token stream into the structured querier
// DTOs used by downstream code.
//
// The must* helpers (mustKeyword, mustSkipParenthesised, mustSchemaQualifiedName) panic
// on a malformed token stream to unwind deep parsing without threading errors through
// every frame, so every external entry point that drives the parser MUST run beneath a
// recover. ApplyDDL and AnalyseQuery already install that recover; any new caller has to
// do the same or a parse panic will escape to the host.
type parser struct {
	// namedParameterMap maps a named parameter's identifier to the sequential number it was
	// first assigned.
	namedParameterMap map[string]int

	// insertProjectionTable is the INSERT target table whose columns the current INSERT ...
	// SELECT projection items are assigned to.
	//
	// It is empty when not parsing such a projection list.
	insertProjectionTable string

	// tokens is the flat token stream produced by the tokeniser.
	tokens []token

	// parameterRefs collects every bind parameter discovered while walking the statement.
	parameterRefs []querier_dto.RawParameterReference

	// rawDerivedTables holds derived table references observed in FROM clauses for later
	// analysis.
	rawDerivedTables []querier_dto.RawDerivedTableReference

	// predicateSubqueries collects subqueries found in a token-scanned predicate position
	// (WHERE / HAVING / JOIN ON), so the domain layer can resolve each one's parameters
	// against the subquery's own scope rather than the parent.
	predicateSubqueries []*querier_dto.RawQueryAnalysis

	// rawTableValuedFunctions holds table-valued function references observed in FROM
	// clauses for later analysis.
	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	// insertProjectionColumns is the ordered INSERT target column list, active only while
	// the INSERT ... SELECT body's projection list is parsed.
	//
	// It is nil otherwise.
	insertProjectionColumns []string

	// position is the index of the next token to consume.
	position int

	// parameterCount tracks the highest parameter number assigned so far.
	parameterCount int

	// analysisDepth bounds recursion through analyseSelect across compound branches, CTE
	// bodies, derived tables, scalar subqueries, and view bodies. Child parsers inherit the
	// parent depth so the global cap holds across instances.
	analysisDepth int

	// expressionDepth bounds recursion through the expression precedence chain so deeply
	// nested parentheses cannot overflow the stack.
	expressionDepth int

	// typeDepth bounds recursion through compound column-type parsing (STRUCT/MAP/LIST/UNION
	// nesting).
	typeDepth int

	// maxParseDepth is the effective cap for analysisDepth, expressionDepth and typeDepth.
	// Zero means defaultMaxParseDepth; the engine sets it from the dialect and child parsers
	// inherit it.
	maxParseDepth int

	// insertProjectionIndex is the zero-based ordinal of the projection item currently being
	// parsed in an INSERT ... SELECT body.
	insertProjectionIndex int

	// hasForUpdate is set when the statement carries a FOR UPDATE locking clause.
	hasForUpdate bool

	// hasDataModifyingCTE is set when a CTE uses INSERT, UPDATE, or DELETE.
	hasDataModifyingCTE bool

	// lastArgumentWasVariadic records whether the most recently parsed call argument used
	// variadic syntax.
	lastArgumentWasVariadic bool
}

// newParser builds a parser over the given token stream with the default parse depth cap.
//
// Takes tokens ([]token) which is the flat token stream to walk.
//
// Returns *parser which is the initialised parser.
func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		maxParseDepth:     defaultMaxParseDepth,
		namedParameterMap: make(map[string]int),
	}
}

// splitStatements partitions the token stream by top-level semicolons.
//
// BEGIN...END blocks (procedural macro bodies) contain inner semicolons that belong to
// the enclosing statement; the splitter tracks block depth so those inner semicolons stay
// attached to the right statement.
//
// BEGIN only opens a block when it appears with preceding content in a procedural
// definition (CREATE [OR REPLACE] MACRO or FUNCTION). A bare `begin` used as a column
// alias or ordinary identifier (for example `SELECT 1 AS begin; SELECT 2;`) must not open
// a block and swallow the trailing statement, so the isProceduralStatement gate guards
// the open.
//
// END decrements blockDepth only when it closes the outer BEGIN..END block. END followed
// by an inner-end keyword (IF, CASE, LOOP, WHILE, REPEAT) closes an inner control
// structure rather than the outer block, so depth must stay. A scalar CASE...END
// expression inside the body also uses a bare END; caseDepth tracks open CASE constructs
// so such an inner END cannot prematurely close the BEGIN block.
//
// Takes tokens ([]token) which is the multi-statement token stream to partition.
//
// Returns [][]token which is one token slice per top-level statement.
func splitStatements(tokens []token) [][]token {
	var statements [][]token
	var current []token
	blockDepth := 0
	caseDepth := 0

	for index, tok := range tokens {
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
			blockDepth, caseDepth = updateSplitStatementsBlockDepth(
				blockDepth, caseDepth, tok, len(current), tokens, index, proceduralContext)
		}
		current = append(current, tok)
	}

	if len(current) > 0 {
		statements = append(statements, current)
	}

	return statements
}

// isProceduralStatement reports whether the in-progress statement is a procedural
// definition whose body may contain a bare BEGIN...END block.
//
// A procedural definition is a CREATE [OR REPLACE] [TEMP] MACRO or FUNCTION. Any other
// leading statement (for example SELECT, INSERT, UPDATE, or CREATE TABLE and VIEW)
// returns false so a `begin` identifier is treated as a plain name, not a block opener.
//
// Takes current ([]token) which is the tokens accumulated for the statement so far.
//
// Returns bool which is true for a procedural-body-bearing statement.
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
		case "OR", "REPLACE", "TEMP", "TEMPORARY":
			continue
		case "MACRO", "FUNCTION":
			return true
		default:
			return false
		}
	}
	return false
}

// handleSplitStatementsSemicolon either appends the semicolon to the in-progress
// statement when blockDepth > 0 (inside a BEGIN..END body) or flushes the current
// statement to the output list. Returns the possibly-extended statement list and the next
// in-progress slice.
//
// Takes statements ([][]token) which is the running statement list.
// Takes current ([]token) which is the in-progress statement being built.
// Takes tok (token) which is the semicolon under consideration.
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

// updateSplitStatementsBlockDepth adjusts blockDepth and caseDepth for an identifier
// token.
//
// BEGIN opens a block only when there is preceding content and the in-progress statement
// is a procedural definition. A scalar CASE increments caseDepth while inside a block so
// its bare END closes the CASE before it can close the BEGIN block. END decrements
// caseDepth first (when a CASE is open) and otherwise decrements blockDepth, unless it
// introduces an inner control structure (END IF, END LOOP, END WHILE, or END REPEAT)
// which leaves both counters untouched.
//
// Takes blockDepth (int) which is the current block-nesting depth.
// Takes caseDepth (int) which is the current open-CASE count.
// Takes tok (token) which is the identifier under consideration.
// Takes currentLength (int) which is the number of tokens already in the in-progress
// statement.
// Takes tokens ([]token) which is the full token stream, used for lookahead after END.
// Takes index (int) which is the position of tok within tokens.
// Takes proceduralContext (bool) which is true when the in-progress statement is a
// procedural definition whose body may contain a bare BEGIN...END block.
//
// Returns the updated blockDepth and caseDepth.
func updateSplitStatementsBlockDepth(
	blockDepth, caseDepth int, tok token, currentLength int, tokens []token, index int, proceduralContext bool,
) (depth, cases int) {
	switch {
	case strings.EqualFold(tok.value, "BEGIN") && currentLength > 0 && proceduralContext:
		return blockDepth + 1, caseDepth
	case strings.EqualFold(tok.value, "CASE") && blockDepth > 0:
		return blockDepth, caseDepth + 1
	case strings.EqualFold(tok.value, "END"):
		return adjustEndDepth(nextTokenValue(tokens, index+1), blockDepth, caseDepth)
	}
	return blockDepth, caseDepth
}

// nextTokenValue returns the value of the token at lookahead, or the empty string when
// lookahead is past the end of the stream. Token streams do not include whitespace so a
// single position offset suffices to inspect the keyword after an END.
//
// Takes tokens ([]token) which is the full token stream.
// Takes lookahead (int) which is the index of the token to inspect.
//
// Returns string which is the token value at lookahead, or empty.
func nextTokenValue(tokens []token, lookahead int) string {
	if lookahead >= len(tokens) {
		return ""
	}
	return tokens[lookahead].value
}

// adjustEndDepth resolves an END keyword against the open CASE and block counters, using
// the following token to distinguish the END forms.
//
// END IF, END LOOP, END WHILE, and END REPEAT close inner control structures and leave
// both counters untouched. END CASE closes a statement-form CASE (decrementing caseDepth
// when one is open). A bare END closes an open scalar CASE first, then the surrounding
// BEGIN block.
//
// Takes nextValue (string) which is the token immediately after END.
// Takes blockDepth (int) which is the current block-nesting depth.
// Takes caseDepth (int) which is the current open-CASE count.
//
// Returns the updated blockDepth and caseDepth.
func adjustEndDepth(nextValue string, blockDepth, caseDepth int) (depth, cases int) {
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

var (
	// firstWordClassifiers maps a leading keyword to a function that classifies the
	// statement based on further tokens.
	firstWordClassifiers = map[string]func([]token) statementKind{
		keywordCREATE: classifyCreateStatement,
		keywordDROP:   classifyDropStatement,
		"ALTER":       classifyAlterStatement,
		keywordWITH:   classifyWithStatement,
	}

	// firstWordStaticKinds maps leading keywords whose statement kind is fully determined
	// without inspecting further tokens.
	firstWordStaticKinds = map[string]statementKind{
		keywordSELECT:  statementKindSelect,
		"INSERT":       statementKindInsert,
		"UPDATE":       statementKindUpdate,
		"DELETE":       statementKindDelete,
		keywordVALUES:  statementKindValues,
		keywordINSTALL: statementKindInstall,
		keywordLOAD:    statementKindLoad,
		"COMMENT":      statementKindComment,
	}
)

// classifyStatement labels a statement from its first token.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the classified category or statementKindUnknown when the
// lead-in is not recognised.
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
	// createObjectKinds maps the keyword following CREATE (and any prefixes) to the matching
	// statement kind.
	createObjectKinds = map[string]statementKind{
		keywordTABLE:  statementKindCreateTable,
		"VIEW":        statementKindCreateView,
		"INDEX":       statementKindCreateIndex,
		keywordUNIQUE: statementKindCreateIndex,
		keywordTYPE:   statementKindCreateType,
		"FUNCTION":    statementKindCreateFunction,
		keywordMACRO:  statementKindCreateMacro,
		keywordSCHEMA: statementKindCreateSchema,
		"SEQUENCE":    statementKindCreateSequence,
	}
)

// classifyCreateStatement decides which CREATE variant the tokens describe.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the CREATE variant or statementKindUnknown when no
// object keyword is recognised.
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

// skipCreatePrefixes returns the index of the first object keyword past any OR REPLACE
// and TEMP/TEMPORARY prefixes.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns int which is the index of the object keyword, or len(tokens) when the prefix is
// malformed.
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

// classifyDropStatement labels a DROP statement by its object keyword.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the DROP variant or statementKindUnknown when the object
// keyword is not recognised.
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
	case keywordTYPE:
		return statementKindDropType
	case "FUNCTION":
		return statementKindDropFunction
	case keywordMACRO:
		return statementKindDropMacro
	case keywordSCHEMA:
		return statementKindDropSchema
	case "SEQUENCE":
		return statementKindDropSequence
	}

	return statementKindUnknown
}

// classifyAlterStatement labels an ALTER statement by its object keyword.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the ALTER variant or statementKindUnknown when the
// object keyword is not recognised.
func classifyAlterStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	second := strings.ToUpper(tokens[1].value)
	switch second {
	case keywordTABLE:
		return statementKindAlterTable
	case keywordTYPE:
		return statementKindAlterType
	case "SEQUENCE":
		return statementKindAlterSequence
	}

	return statementKindUnknown
}

// classifyWithStatement scans a WITH statement's body for the first top-level DML keyword
// to decide the statement kind.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the DML kind found, defaulting to statementKindSelect
// when nothing more specific is seen.
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
	// dmlKeywords maps DML-introducing keywords to their statement kind.
	dmlKeywords = map[string]statementKind{
		keywordSELECT: statementKindSelect,
		"INSERT":      statementKindInsert,
		"UPDATE":      statementKindUpdate,
		"DELETE":      statementKindDelete,
		keywordVALUES: statementKindValues,
	}
)

// classifyDMLKeyword looks up a candidate DML keyword.
//
// Takes value (string) which is the raw identifier value.
//
// Returns statementKind which is the matching kind when found.
// Returns bool which is true when value matches a DML keyword.
func classifyDMLKeyword(value string) (statementKind, bool) {
	kind, matched := dmlKeywords[strings.ToUpper(value)]
	return kind, matched
}

// current returns the token at the parser's position.
//
// Returns token which is the current token, or an EOF token when position is past the
// end.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token one past the parser's position without advancing.
//
// Returns token which is the look-ahead token, or an EOF token when no further tokens
// remain.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// advance consumes the current token and moves position forward.
//
// Returns token which is the token that was current before advancing.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// expectKeyword consumes the current token when it matches any of the given keywords.
//
// Takes keywords (...string) which is the set of acceptable keyword spellings, compared
// case-insensitively.
//
// Returns token which is the consumed token on success.
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

// matchKeyword consumes the current token when it equals keyword.
//
// Takes keyword (string) which is the keyword to match, case-insensitively.
//
// Returns bool which is true when the current token matched and was consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token equals keyword without consuming it.
//
// Takes keyword (string) which is the keyword to compare, case-insensitively.
//
// Returns bool which is true when the current token matches keyword.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the given keywords
// without consuming it.
//
// Takes keywords (...string) which is the set of keywords to test against,
// case-insensitively.
//
// Returns bool which is true when the current token matches any of the keywords.
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

// atEnd reports whether the parser has consumed all tokens.
//
// Returns bool which is true when position is past the last token or the current token is
// EOF.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// parseIdentifierOrKeyword consumes an identifier or quoted string, accepting both
// because DuckDB allows quoted identifiers.
//
// Returns string which is the consumed token's value.
// Returns error when the current token is neither an identifier nor a string literal.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// parseSchemaQualifiedName consumes an optionally schema-qualified identifier of the form
// "name" or "schema.name".
//
// Returns schema (string) which is the schema name when present, else empty.
// Returns name (string) which is the unqualified identifier.
// Returns err (error) when the identifier cannot be parsed.
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

// skipParenthesised consumes a balanced parenthesised group starting at the current
// token.
//
// Returns error when the current token is not '(' or the group is unbalanced.
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
// that lay strictly inside it.
//
// Returns []token which holds the inner tokens, excluding the outer parentheses.
// Returns error when the current token is not '(' or the group is unbalanced.
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

// mustKeyword consumes one of keywords and panics on mismatch.
//
// Takes keywords (...string) which is the set of acceptable keywords.
//
// Panics when the current token does not match any keyword.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

// mustSkipParenthesised consumes a balanced parenthesised group and panics on mismatch.
//
// Panics when skipParenthesised fails.
func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

// mustSchemaQualifiedName consumes a schema-qualified identifier and panics on failure.
//
// Returns schema (string) which is the schema portion when present.
// Returns name (string) which is the unqualified identifier.
//
// Panics when parseSchemaQualifiedName returns an error.
func (p *parser) mustSchemaQualifiedName() (schema string, name string) {
	schema, name, err := p.parseSchemaQualifiedName()
	if err != nil {
		panic(fmt.Errorf("mustSchemaQualifiedName: %w", err))
	}
	return schema, name
}

// registerParameterFromToken records a bind parameter reference, dispatching by the
// parameter token's shape.
//
// Takes parameterToken (token) which is the original parameter token.
// Takes context (querier_dto.ParameterContext) which records where in the statement the
// parameter appears.
// Takes columnReference (*querier_dto.ColumnReference) which is the inferred target
// column, or nil when none is known.
// Takes castType (*querier_dto.SQLType) which is the inline cast type applied to the
// parameter, or nil when no cast applies.
//
// Returns int which is the parameter's sequential number.
func (p *parser) registerParameterFromToken(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	switch parameterToken.kind {
	case tokenDollarParam:
		return p.registerDollarParameter(parameterToken, context, columnReference, castType)
	case tokenNamedParam:
		return p.registerNamedParameter(parameterToken, context, columnReference, castType)
	default:
		return p.registerSequentialParameter(context, columnReference, castType)
	}
}

// registerSequentialParameter assigns the next sequential parameter number and records
// the reference.
//
// Takes context (querier_dto.ParameterContext) which records where the parameter appears.
// Takes columnReference (*querier_dto.ColumnReference) which is the inferred target
// column, or nil when none is known.
// Takes castType (*querier_dto.SQLType) which is the inline cast type, or nil when no
// cast applies.
//
// Returns int which is the newly assigned parameter number.
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

// registerDollarParameter records a $N positional parameter, preserving the explicit
// number and bumping the watermark when needed.
//
// Takes parameterToken (token) which carries the $N spelling.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears.
// Takes columnReference (*querier_dto.ColumnReference) which is the inferred target
// column, or nil when none is known.
// Takes castType (*querier_dto.SQLType) which is the inline cast type, or nil when no
// cast applies.
//
// Returns int which is the parameter's number as written.
func (p *parser) registerDollarParameter(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	number, conversionError := strconv.Atoi(parameterToken.value[1:])
	if conversionError != nil {
		number = 0
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

// registerNamedParameter records a :name parameter, reusing the number previously
// assigned to the same name when one exists.
//
// Takes parameterToken (token) which carries the :name spelling.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears.
// Takes columnReference (*querier_dto.ColumnReference) which is the inferred target
// column, or nil when none is known.
// Takes castType (*querier_dto.SQLType) which is the inline cast type, or nil when no
// cast applies.
//
// Returns int which is the number associated with the name.
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

// isParameterToken reports whether a token kind represents a bind parameter placeholder.
//
// Takes kind (tokenKind) which is the token kind to test.
//
// Returns bool which is true for $N and :name parameter kinds.
func isParameterToken(kind tokenKind) bool {
	return kind == tokenDollarParam || kind == tokenNamedParam
}
