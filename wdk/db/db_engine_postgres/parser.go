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

package db_engine_postgres

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// statementKind enumerates the top-level SQL statement forms the parser classifies.
type statementKind uint16

const (
	// statementKindUnknown marks a statement whose form cannot be classified from its
	// leading tokens. Placed first (value 0) so the zero value of statementKind is a safe
	// decline sentinel for extension Classify callbacks; no built-in kind can accidentally
	// share the decline value.
	statementKindUnknown statementKind = iota

	// statementKindCreateTable marks a CREATE TABLE statement.
	statementKindCreateTable

	// statementKindDropTable marks a DROP TABLE statement.
	statementKindDropTable

	// statementKindAlterTable marks an ALTER TABLE statement.
	statementKindAlterTable

	// statementKindCreateView marks a CREATE VIEW or MATERIALIZED VIEW statement.
	statementKindCreateView

	// statementKindDropView marks a DROP VIEW or DROP MATERIALIZED VIEW statement.
	statementKindDropView

	// statementKindCreateIndex marks a CREATE INDEX statement.
	statementKindCreateIndex

	// statementKindDropIndex marks a DROP INDEX statement.
	statementKindDropIndex

	// statementKindCreateTrigger marks a CREATE TRIGGER statement.
	statementKindCreateTrigger

	// statementKindDropTrigger marks a DROP TRIGGER statement.
	statementKindDropTrigger

	// statementKindCreateType marks a CREATE TYPE statement.
	statementKindCreateType

	// statementKindAlterType marks an ALTER TYPE statement.
	statementKindAlterType

	// statementKindDropType marks a DROP TYPE statement.
	statementKindDropType

	// statementKindCreateFunction marks a CREATE FUNCTION or PROCEDURE statement.
	statementKindCreateFunction

	// statementKindDropFunction marks a DROP FUNCTION or PROCEDURE statement.
	statementKindDropFunction

	// statementKindCreateSchema marks a CREATE SCHEMA statement.
	statementKindCreateSchema

	// statementKindDropSchema marks a DROP SCHEMA statement.
	statementKindDropSchema

	// statementKindCreateExtension marks a CREATE EXTENSION statement.
	statementKindCreateExtension

	// statementKindDropExtension marks a DROP EXTENSION statement.
	statementKindDropExtension

	// statementKindCreateSequence marks a CREATE SEQUENCE statement.
	statementKindCreateSequence

	// statementKindDropSequence marks a DROP SEQUENCE statement.
	statementKindDropSequence

	// statementKindComment marks a COMMENT statement.
	statementKindComment

	// statementKindSelect marks a SELECT statement.
	statementKindSelect

	// statementKindInsert marks an INSERT statement.
	statementKindInsert

	// statementKindUpdate marks an UPDATE statement.
	statementKindUpdate

	// statementKindDelete marks a DELETE statement.
	statementKindDelete

	// statementKindValues marks a standalone VALUES statement.
	statementKindValues

	// statementKindCount is the sentinel count of statement kinds.
	statementKindCount
)

const (
	// minTokensForCreateOrReplace is the minimum token count for a CREATE OR REPLACE form
	// before the object keyword can appear.
	minTokensForCreateOrReplace = 4

	// indexAfterOrReplace is the token index of the object keyword after CREATE OR REPLACE.
	indexAfterOrReplace = 3

	// defaultMaxParseDepth caps every user-driven recursion in the parser.
	//
	// It bounds analyseSelect (subquery, CTE, and compound-branch nesting) and the
	// expression precedence chain (parenthesis nesting). The cap is essential because Go
	// raises a fatal, non-recoverable stack overflow that the engine's recover guards cannot
	// catch, so deeply nested input would otherwise crash the host process. 256 is far below
	// the overflow threshold yet out of the way for realistic queries (hand-written queries
	// rarely exceed 10); callers may override it with WithMaxParseDepth.
	defaultMaxParseDepth = 256

	// maxDDLDepth caps recursion through DDL type-shape parsing.
	//
	// It bounds composite types, RETURNS TABLE columns, nested array dimensions, and nested
	// function argument types. Real postgres DDL never nests these constructs more than a
	// handful of levels deep; the cap exists so a maliciously crafted DDL cannot exhaust the
	// goroutine stack. It is a fixed const (independent of the configurable
	// defaultMaxParseDepth) because DDL nesting has no legitimate reason to grow and a tight
	// ceiling keeps the worst case small. The same const also bounds reloption parenthesis
	// nesting in extension_context.go.
	maxDDLDepth = 64
)

var (
	// errUnmatchedParenthesis is returned when a parenthesis scan hits the end of the token
	// stream without a matching close.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")

	// errAnalysisDepthExceeded is the sentinel returned when analyseSelect recursion exceeds
	// the configured maximum parse depth (defaultMaxParseDepth, or the WithMaxParseDepth
	// override).
	errAnalysisDepthExceeded = errors.New("postgres: analysis recursion depth exceeded")

	// errDDLDepthExceeded is the sentinel returned by composite type and RETURNS TABLE
	// column parsing when nested type definitions breach maxDDLDepth. Surfaces from
	// parseCreateTable, parseCreateType, and parseFunctionReturnsTableColumns as a wrapped
	// error so callers can pattern-match against the sentinel with errors.Is.
	errDDLDepthExceeded = errors.New("postgres: ddl recursion depth exceeded")

	// firstWordClassifiers maps a statement's leading keyword to the classifier that
	// resolves its specific kind by inspecting subsequent tokens.
	firstWordClassifiers = map[string]func([]token) statementKind{
		keywordCREATE: classifyCreateStatement,
		keywordDROP:   classifyDropStatement,
		"ALTER":       classifyAlterStatement,
		keywordWITH:   classifyWithStatement,
	}

	// firstWordStaticKinds maps a statement's leading keyword directly to its kind when no
	// further inspection is required.
	firstWordStaticKinds = map[string]statementKind{
		keywordSELECT: statementKindSelect,
		"INSERT":      statementKindInsert,
		"UPDATE":      statementKindUpdate,
		"DELETE":      statementKindDelete,
		keywordVALUES: statementKindValues,
		"COMMENT":     statementKindComment,
	}

	// createObjectKinds maps each CREATE object keyword to its specific statement kind so
	// the CREATE-statement classifier can avoid a long switch.
	createObjectKinds = map[string]statementKind{
		keywordTABLE:   statementKindCreateTable,
		"VIEW":         statementKindCreateView,
		"MATERIALIZED": statementKindCreateView,
		"INDEX":        statementKindCreateIndex,
		keywordUNIQUE:  statementKindCreateIndex,
		keywordTYPE:    statementKindCreateType,
		"FUNCTION":     statementKindCreateFunction,
		"PROCEDURE":    statementKindCreateFunction,
		keywordSCHEMA:  statementKindCreateSchema,
		"EXTENSION":    statementKindCreateExtension,
		"TRIGGER":      statementKindCreateTrigger,
		"SEQUENCE":     statementKindCreateSequence,
	}

	// dmlKeywords maps each DML keyword to its statement kind for use when classifying CTE
	// bodies.
	dmlKeywords = map[string]statementKind{
		keywordSELECT: statementKindSelect,
		"INSERT":      statementKindInsert,
		"UPDATE":      statementKindUpdate,
		"DELETE":      statementKindDelete,
		keywordVALUES: statementKindValues,
	}
)

// parsedStatement holds the tokens and classification of a single SQL statement extracted
// from a larger script.
type parsedStatement struct {
	// tokens holds the statement's tokens excluding the trailing semicolon.
	tokens []token

	// kind tags the statement's top-level form.
	kind statementKind

	// extensionOwner is the index into PostgresDialect.StatementExtensions of the extension
	// that claimed this statement during ParseStatements, or -1 when the kind is a built-in.
	//
	// Caching the index here lets dispatchDDL look up the owning extension in O(1) rather
	// than re-running Classify against every registered extension on every apply call. The
	// index becomes meaningless if the extension list is reordered between parse and
	// dispatch, but the PostgresEngine treats the list as immutable for the lifetime of the
	// engine.
	extensionOwner int
}

// IsParsedStatement marks the receiver as a parsed statement value.
func (*parsedStatement) IsParsedStatement() {}

// parser holds the mutable state used while walking a token stream and extracting
// parameter and table references.
type parser struct {
	// namedParameterMap deduplicates named parameters by mapping each name to its assigned
	// positional number.
	namedParameterMap map[string]int

	// insertProjectionTable is the INSERT target table whose columns the current INSERT ...
	// SELECT projection items are assigned to.
	//
	// Empty when the parser is not inside an INSERT ... SELECT projection list.
	insertProjectionTable string

	// tokens is the token stream being walked.
	tokens []token

	// parameterRefs collects every parameter reference found in the statement.
	parameterRefs []querier_dto.RawParameterReference

	// rawDerivedTables collects derived-table references found while walking FROM clauses.
	rawDerivedTables []querier_dto.RawDerivedTableReference

	// predicateSubqueries collects subqueries found in a token-scanned predicate position
	// (WHERE / HAVING / JOIN ON), so the domain layer can resolve each one's parameters
	// against the subquery's own scope rather than the parent.
	predicateSubqueries []*querier_dto.RawQueryAnalysis

	// rawTableValuedFunctions collects table-valued function references found while walking
	// FROM clauses.
	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	// insertProjectionColumns is the ordered INSERT target column list, used to map each
	// top-level INSERT ... SELECT projection item positionally onto its target column.
	//
	// Nil when the parser is not inside an INSERT ... SELECT projection list.
	insertProjectionColumns []string

	// position is the index of the next token to consume.
	position int

	// parameterCount tracks the highest positional parameter number assigned so far.
	parameterCount int

	// insertProjectionIndex is the zero-based ordinal of the INSERT ... SELECT projection
	// item currently being parsed.
	//
	// It advances once per top-level projection item, so a literal or column item still
	// consumes its target-column slot.
	insertProjectionIndex int

	// analysisDepth tracks recursion through analyseSelect across CTE bodies, compound
	// (UNION/INTERSECT/EXCEPT) branches, derived tables, scalar subqueries, and view bodies.
	//
	// When the counter exceeds maxParseDepth the parser returns errAnalysisDepthExceeded so
	// a pathologically deep query cannot blow the goroutine stack. Child parsers spawned for
	// nested constructs inherit the parent's depth so the global recursion bound holds
	// across parser instances.
	analysisDepth int

	// expressionDepth bounds recursion through the expression precedence chain so deeply
	// nested parentheses cannot overflow the stack. Mirrors analysisDepth: incremented
	// before the recursive descent and decremented after.
	expressionDepth int

	// maxParseDepth is the effective cap for analysisDepth and expressionDepth. Zero means
	// defaultMaxParseDepth; the engine sets it from the dialect and child parsers inherit it
	// via newChildParser so the global cap holds across instances.
	maxParseDepth int

	// ddlDepth tracks recursion through DDL type-shape parsing: composite type field types,
	// RETURNS TABLE column types, and nested column type modifiers.
	//
	// The counter is incremented before the recursive call and decremented after, mirroring
	// analysisDepth so the two budgets stay independent. When the counter would exceed
	// maxDDLDepth the parser returns errDDLDepthExceeded so a pathologically nested CREATE
	// TYPE cannot blow the goroutine stack.
	ddlDepth int

	// hasForUpdate records that the statement carried a FOR UPDATE row-locking clause.
	hasForUpdate bool

	// hasDataModifyingCTE records that a CTE contained INSERT, UPDATE, or DELETE.
	hasDataModifyingCTE bool

	// lastArgumentWasVariadic records that the last function argument parsed used the
	// VARIADIC keyword.
	lastArgumentWasVariadic bool
}

// newParser creates a parser over tokens with the default recursion cap.
//
// Takes tokens ([]token) which is the token stream to walk.
//
// Returns *parser which is ready to analyse the supplied tokens.
func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		maxParseDepth:     defaultMaxParseDepth,
		namedParameterMap: make(map[string]int),
	}
}

// newChildParser creates a parser over tokens that inherits the recursion state required
// to keep the global depth caps intact across parser instances.
//
// Nested constructs (derived tables, scalar subqueries, EXISTS subqueries, CTE bodies,
// view bodies, IN-list subqueries) spawn a fresh parser; routing every such spawn through
// this helper guarantees the child continues counting from the parent's analysisDepth and
// shares the same maxParseDepth, so no nesting path can reset the cap to zero and recurse
// without bound. parameterCount is also carried so positional bind numbers stay monotonic
// across the boundary.
//
// Takes tokens ([]token) which is the child statement's token slice.
//
// Returns *parser which is the child parser pre-seeded with the inherited depth and
// parameter state.
func (p *parser) newChildParser(tokens []token) *parser {
	child := newParser(tokens)
	child.parameterCount = p.parameterCount
	child.analysisDepth = p.analysisDepth
	child.expressionDepth = p.expressionDepth
	child.maxParseDepth = p.maxParseDepth
	return child
}

// splitStatements partitions the token stream by top-level semicolons.
//
// Most postgres procedural bodies wrap their content in dollar-quoted strings (handled at
// the lexer layer as a single token) so the splitter rarely sees BEGIN/END as bare
// identifiers, but DO blocks and plain trigger bodies without dollar-quoting can still
// appear. BEGIN...END nesting is tracked defensively so embedded semicolons stay attached
// to the enclosing statement.
//
// BEGIN only opens a block when there is preceding content for the current statement; a
// leading "BEGIN;" is a transaction marker, not a block opener. END only closes a block
// when blockDepth > 0.
//
// Takes tokens ([]token) which is the full token stream to partition.
//
// Returns [][]token which is one token slice per top-level statement.
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
			current, statements = handleStatementSemicolon(tok, current, statements, blockDepth)
			continue
		}
		if tok.kind == tokenIdentifier {
			proceduralContext := false
			if strings.EqualFold(tok.value, "BEGIN") {
				proceduralContext = isProceduralStatement(current)
			}
			blockDepth, caseDepth = adjustBlockDepth(tok.value, nextTokenValue(tokens, index), blockDepth, caseDepth, len(current), proceduralContext)
		}
		current = append(current, tok)
	}

	if len(current) > 0 {
		statements = append(statements, current)
	}

	return statements
}

// nextTokenValue returns the value of the token immediately following index, or the empty
// string when index is the last token.
//
// Used by the statement splitter to inspect the keyword after an END so it can tell `END
// IF` / `END LOOP` / `END CASE` (which close inner procedural constructs) from a bare END
// that closes the surrounding BEGIN block.
//
// Takes tokens ([]token) which is the token stream being scanned.
// Takes index (int) which is the index of the current token.
//
// Returns string which is the next token's value, or empty when index is the last token.
func nextTokenValue(tokens []token, index int) string {
	if index+1 >= len(tokens) {
		return ""
	}
	return tokens[index+1].value
}

// handleStatementSemicolon decides whether a semicolon ends the current statement or sits
// inside an open BEGIN...END block.
//
// Inside a block the semicolon is appended to the current statement so the procedural
// body stays intact. At top level a non-empty current statement is flushed and a fresh
// slice is returned; an empty current statement is left untouched so leading or stray
// semicolons do not emit empty entries.
//
// Takes semicolon (token) which is the semicolon being processed.
// Takes current ([]token) which is the in-flight statement being built.
// Takes statements ([][]token) which is the completed statement list.
// Takes blockDepth (int) which is the current BEGIN...END nesting depth.
//
// Returns []token which is the updated in-flight statement (nil after a top-level flush).
// Returns [][]token which is the updated completed statement list.
func handleStatementSemicolon(semicolon token, current []token, statements [][]token, blockDepth int) ([]token, [][]token) {
	if blockDepth > 0 {
		current = append(current, semicolon)
		return current, statements
	}
	if len(current) > 0 {
		statements = append(statements, current)
		current = nil
	}
	return current, statements
}

// adjustBlockDepth updates the BEGIN...END block-nesting counter and the inner
// expression-CASE counter when the current identifier opens or closes a procedural
// construct. BEGIN only opens a block when there is already preceding content; a leading
// `BEGIN;` is a transaction marker, not a block opener.
//
// A scalar `CASE ... END` expression inside a procedural body also uses a bare END.
// Without tracking it, that inner END would wrongly decrement the block depth and the
// next top-level semicolon would split the trigger/function body mid-statement. caseDepth
// records open CASE constructs (only while inside a block) so a bare END closes the CASE
// before it can close the BEGIN block; `END IF` / `END LOOP` / `END WHILE` / `END REPEAT`
// close inner constructs that did not touch caseDepth, and `END CASE` closes a
// statement-form CASE.
//
// Takes value (string) which is the identifier text being inspected.
// Takes nextValue (string) which is the token immediately after value (used to classify
// an END).
// Takes blockDepth (int) which is the current block-nesting depth.
// Takes caseDepth (int) which is the current open-CASE count.
// Takes currentLength (int) which is the number of tokens already in the in-flight
// statement.
//
// Takes proceduralContext (bool) which is true when the in-flight statement is a
// procedural definition (CREATE FUNCTION/PROCEDURE/TRIGGER or DO) whose body may contain
// a bare BEGIN...END block. BEGIN only opens a block in that context so a `begin` used as
// a column name or alias in an ordinary statement (e.g. `SELECT 1 AS begin; SELECT 2;`)
// does not swallow the rest of the file. PostgreSQL function/trigger bodies are normally
// dollar- or single-quoted (handled by the string scanners, never reaching here), so this
// gate does not lose any genuine block tracking.
//
// Returns the adjusted blockDepth and caseDepth.
func adjustBlockDepth(value, nextValue string, blockDepth, caseDepth, currentLength int, proceduralContext bool) (depth, cases int) {
	switch {
	case strings.EqualFold(value, "BEGIN") && currentLength > 0 && proceduralContext:
		return blockDepth + 1, caseDepth
	case strings.EqualFold(value, "CASE") && blockDepth > 0:
		return blockDepth, caseDepth + 1
	case strings.EqualFold(value, "END"):
		return adjustEndDepth(nextValue, blockDepth, caseDepth)
	}
	return blockDepth, caseDepth
}

// isProceduralStatement reports whether the in-flight statement is a procedural
// definition whose body may legitimately contain a bare BEGIN...END block.
//
// Such definitions are a top-level DO or a CREATE [OR REPLACE] [TEMP]
// FUNCTION/PROCEDURE/TRIGGER. Any other leading statement (for example
// SELECT/INSERT/UPDATE or CREATE TABLE/VIEW) returns false so a `begin` identifier is
// treated as a plain name, not a block opener.
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
			switch word {
			case "DO":
				return true
			case "CREATE":
				sawCreate = true
			default:
				return false
			}
			continue
		}
		switch word {
		case "OR", "REPLACE", "TEMP", "TEMPORARY", "CONSTRAINT":
			continue
		case "FUNCTION", "PROCEDURE", "TRIGGER":
			return true
		default:
			return false
		}
	}
	return false
}

// adjustEndDepth resolves an END keyword against the open CASE and block counters, using
// the following token to distinguish the END forms.
//
// Takes nextValue (string) which is the token immediately after END.
// Takes blockDepth (int) which is the current BEGIN...END nesting depth.
// Takes caseDepth (int) which is the current open-CASE count.
//
// Returns depth (int) which is the adjusted block-nesting depth.
// Returns cases (int) which is the adjusted open-CASE count.
//
// See adjustBlockDepth for the rationale.
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

// classifyStatement returns the kind of the statement represented by the supplied tokens.
//
// Takes tokens ([]token) which is the token stream of a single statement.
//
// Returns statementKind which is the resolved kind, or statementKindUnknown when the
// leading tokens are not recognised.
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

// classifyViaExtensionsIndexed consults each registered StatementExtension in
// registration order for the first valid extension claim.
//
// It returns the first non-zero StatementKind that sits inside the extension reservation
// range [StatementKindExtensionBase, ...) along with the registry index of the extension
// that claimed the statement. The cached index is stored on parsedStatement so
// dispatchDDL can skip the re-classification scan when the extension's Parse method runs.
//
// Extensions are expected to return either 0 (decline) or a kind in their reserved range;
// any kind below StatementKindExtensionBase is treated as a misuse, logged at debug
// level, and rejected so an extension cannot accidentally hijack a built-in handler by
// returning a raw integer like 1 that happens to alias statementKindCreateTable.
//
// Takes tokens ([]token) which is the candidate statement's token slice.
// Takes extensions ([]StatementExtension) which is the registry walked in registration
// order.
//
// Returns statementKind which is the first valid extension claim, or statementKindUnknown
// when every extension declines (or returns a kind outside the reservation range).
// Returns int which is the index of the claiming extension, or -1 when no extension
// claimed the statement.
func classifyViaExtensionsIndexed(tokens []token, extensions []StatementExtension) (statementKind, int) {
	for index, ext := range extensions {
		kind := ext.Classify(tokens)
		if kind == 0 {
			continue
		}
		if kind < StatementKindExtensionBase {
			log.Debug("postgres: extension Classify returned built-in kind; ignoring",
				logger_domain.Int("kind", int(kind)),
			)
			continue
		}
		return kind, index
	}
	return statementKindUnknown, -1
}

// classifyCreateStatement returns the kind of a CREATE statement.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns statementKind which is the specific CREATE kind, or statementKindUnknown when
// the object keyword is unrecognised.
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

// skipCreatePrefixes returns the index of the first token after any OR REPLACE and TEMP /
// TEMPORARY / UNLOGGED qualifiers following CREATE.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns int which is the index of the next object keyword, or len(tokens) when the
// prefixes run past the end of the stream.
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

	if upper == "TEMP" || upper == "TEMPORARY" || upper == "UNLOGGED" {
		if index+1 >= len(tokens) {
			return len(tokens)
		}
		index++
	}

	return index
}

// classifyDropStatement returns the kind of a DROP statement.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns statementKind which is the specific DROP kind, or statementKindUnknown when the
// object keyword is unrecognised.
func classifyDropStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	second := strings.ToUpper(tokens[1].value)
	switch second {
	case keywordTABLE:
		return statementKindDropTable
	case "VIEW", "MATERIALIZED":
		return statementKindDropView
	case "INDEX":
		return statementKindDropIndex
	case keywordTYPE:
		return statementKindDropType
	case "FUNCTION", "PROCEDURE":
		return statementKindDropFunction
	case keywordSCHEMA:
		return statementKindDropSchema
	case "EXTENSION":
		return statementKindDropExtension
	case "TRIGGER":
		return statementKindDropTrigger
	case "SEQUENCE":
		return statementKindDropSequence
	}

	return statementKindUnknown
}

// classifyAlterStatement returns the kind of an ALTER statement.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns statementKind which is the specific ALTER kind, or statementKindUnknown when
// the object keyword is unrecognised.
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
	}

	return statementKindUnknown
}

// classifyWithStatement returns the kind of a WITH (CTE) statement by scanning for the
// first DML keyword at top level.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns statementKind which is the DML kind embedded in the WITH, or
// statementKindSelect when no other DML keyword is found.
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

// classifyDMLKeyword reports whether a token value names a DML keyword and returns the
// associated kind when it does.
//
// Takes value (string) which is the identifier value to test.
//
// Returns statementKind which is the matched DML kind.
// Returns bool which is true when the value names a DML keyword.
func classifyDMLKeyword(value string) (statementKind, bool) {
	kind, matched := dmlKeywords[strings.ToUpper(value)]
	return kind, matched
}

// current returns the token at the current position without consuming it.
//
// Returns token which is the current token, or a tokenEOF token when the position is past
// the end of the stream.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token one past the current position without consuming it.
//
// Returns token which is the next token after current, or a tokenEOF token when there is
// none.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// advance consumes and returns the current token.
//
// Returns token which is the token that was at the current position.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// expectKeyword consumes the current token when it matches one of the supplied keywords.
//
// Takes keywords (...string) which is the list of acceptable keywords.
//
// Returns token which is the matched token.
// Returns error when the current token is not one of the expected keywords.
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

// matchKeyword consumes the current token when it matches keyword.
//
// Takes keyword (string) which is the keyword to match.
//
// Returns bool which is true when the keyword matched and was consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token matches keyword without consuming it.
//
// Takes keyword (string) which is the keyword to test for.
//
// Returns bool which is true when the current token matches.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the supplied keywords
// without consuming it.
//
// Takes keywords (...string) which is the list of keywords to test.
//
// Returns bool which is true when the current token matches one of them.
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

// atEnd reports whether the parser has reached the end of the token stream.
//
// Returns bool which is true at end of input.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// parseIdentifierOrKeyword consumes and returns the current identifier or quoted string
// value.
//
// Returns string which is the identifier value.
// Returns error when the current token is neither an identifier nor a string.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// parseSchemaQualifiedName consumes a name of the form `schema.name` or a bare `name`
// from the current position.
//
// Returns schema (string) which is the schema qualifier, or "" when absent.
// Returns name (string) which is the trailing identifier.
// Returns err (error) when an identifier cannot be parsed.
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
// position.
//
// Returns error when the current token is not '(' or the group is unmatched.
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

// collectParenthesised consumes a balanced parenthesised group and returns its inner
// tokens.
//
// Returns []token which is the slice of tokens between the outer parentheses.
// Returns error when the current token is not '(' or the group is unmatched.
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

// mustKeyword consumes one of keywords or panics.
//
// Takes keywords (...string) which is the list of acceptable keywords.
//
// Panics when expectKeyword returns an error.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

// mustSkipParenthesised consumes a balanced parenthesised group or panics.
//
// Panics when skipParenthesised returns an error.
func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

// mustIdentifierOrKeyword consumes an identifier or string token and returns its value,
// or panics.
//
// Returns string which is the consumed identifier value.
//
// Panics when parseIdentifierOrKeyword returns an error.
func (p *parser) mustIdentifierOrKeyword() string {
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		panic(fmt.Errorf("mustIdentifierOrKeyword: %w", err))
	}
	return name
}

// mustSchemaQualifiedName consumes a schema-qualified name or panics.
//
// Returns schema (string) which is the schema qualifier, or "" when absent.
// Returns name (string) which is the trailing identifier.
//
// Panics when parseSchemaQualifiedName returns an error.
func (p *parser) mustSchemaQualifiedName() (schema string, name string) {
	schema, name, err := p.parseSchemaQualifiedName()
	if err != nil {
		panic(fmt.Errorf("mustSchemaQualifiedName: %w", err))
	}
	return schema, name
}

// registerParameterFromToken records a parameter reference for a dollar, named, or
// sequential parameter token.
//
// Takes parameterToken (token) which is the parameter token to record.
// Takes context (querier_dto.ParameterContext) which classifies the parameter's syntactic
// role.
// Takes columnReference (*querier_dto.ColumnReference) which is the associated column
// when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when known.
//
// Returns int which is the parameter's positional number.
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

// registerSequentialParameter allocates the next positional number to a sequential
// placeholder and records its metadata.
//
// Takes context (querier_dto.ParameterContext) which classifies the parameter's syntactic
// role.
// Takes columnReference (*querier_dto.ColumnReference) which is the associated column
// when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when known.
//
// Returns int which is the allocated parameter number.
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

// registerDollarParameter records a `$n` placeholder reference and updates the parameter
// count to track the highest seen number.
//
// Takes parameterToken (token) which is the `$n` token.
// Takes context (querier_dto.ParameterContext) which classifies the parameter's syntactic
// role.
// Takes columnReference (*querier_dto.ColumnReference) which is the associated column
// when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when known.
//
// Returns int which is the parameter's positional number.
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

// registerNamedParameter records a `:name` placeholder reference, reusing an existing
// positional number when the name has been seen before.
//
// Takes parameterToken (token) which is the `:name` token.
// Takes context (querier_dto.ParameterContext) which classifies the parameter's syntactic
// role.
// Takes columnReference (*querier_dto.ColumnReference) which is the associated column
// when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when known.
//
// Returns int which is the parameter's positional number.
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

// isParameterToken reports whether a token kind represents a parameter placeholder.
//
// Takes kind (tokenKind) which is the token kind to test.
//
// Returns bool which is true for dollar or named parameter tokens.
func isParameterToken(kind tokenKind) bool {
	return kind == tokenDollarParam || kind == tokenNamedParam
}
