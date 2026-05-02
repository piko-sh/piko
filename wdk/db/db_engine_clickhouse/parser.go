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

package db_engine_clickhouse

import (
	"errors"
	"fmt"
	"strings"
)

// statementKind enumerates every top-level SQL statement the ClickHouse engine adapter
// recognises. Each kind routes to its own parsing branch in ApplyDDL / AnalyseQuery so
// additions stay localised.
type statementKind uint8

const (
	// statementKindCreateTable marks `CREATE TABLE ...`.
	statementKindCreateTable statementKind = iota

	// statementKindCreateView marks `CREATE VIEW ...`.
	statementKindCreateView

	// statementKindCreateMaterializedView marks `CREATE MATERIALIZED VIEW ...`.
	statementKindCreateMaterializedView

	// statementKindCreateDictionary marks `CREATE DICTIONARY ...`.
	statementKindCreateDictionary

	// statementKindCreateFunction marks `CREATE FUNCTION ...`.
	statementKindCreateFunction

	// statementKindCreateDatabase marks `CREATE DATABASE ...`.
	statementKindCreateDatabase

	// statementKindDropTable marks `DROP TABLE ...`.
	statementKindDropTable

	// statementKindDropView marks `DROP VIEW ...`.
	statementKindDropView

	// statementKindDropDictionary marks `DROP DICTIONARY ...`.
	statementKindDropDictionary

	// statementKindDropFunction marks `DROP FUNCTION ...`.
	statementKindDropFunction

	// statementKindDropDatabase marks `DROP DATABASE ...`.
	statementKindDropDatabase

	// statementKindAlterTable marks `ALTER TABLE ...`.
	statementKindAlterTable

	// statementKindRenameTable marks `RENAME TABLE ...`.
	statementKindRenameTable

	// statementKindExchangeTables marks `EXCHANGE TABLES ...`.
	statementKindExchangeTables

	// statementKindTruncate marks `TRUNCATE ...`.
	statementKindTruncate

	// statementKindOptimize marks a table-optimisation statement.
	statementKindOptimize

	// statementKindSystem marks `SYSTEM ...`.
	statementKindSystem

	// statementKindUse marks `USE ...`.
	statementKindUse

	// statementKindShow marks `SHOW ...` (read-only).
	statementKindShow

	// statementKindSet marks `SET ...`.
	statementKindSet

	// statementKindSelect marks `SELECT ...` (read-only).
	statementKindSelect

	// statementKindInsert marks `INSERT ...`.
	statementKindInsert

	// statementKindDelete marks a lightweight `DELETE FROM ... [WHERE ...]`.
	//
	// The ALTER TABLE ... DELETE mutation form is classified as statementKindAlterTable
	// instead.
	statementKindDelete

	// statementKindCreateUser marks `CREATE USER ...`.
	statementKindCreateUser

	// statementKindAlterUser marks `ALTER USER ...`.
	statementKindAlterUser

	// statementKindDropUser marks `DROP USER ...`.
	statementKindDropUser

	// statementKindCreateRole marks `CREATE ROLE ...`.
	statementKindCreateRole

	// statementKindAlterRole marks `ALTER ROLE ...`.
	statementKindAlterRole

	// statementKindDropRole marks `DROP ROLE ...`.
	statementKindDropRole

	// statementKindCreatePolicy marks `CREATE [ROW] POLICY ...`.
	statementKindCreatePolicy

	// statementKindAlterPolicy marks `ALTER [ROW] POLICY ...`.
	statementKindAlterPolicy

	// statementKindDropPolicy marks `DROP [ROW] POLICY ...`.
	statementKindDropPolicy

	// statementKindCreateQuota marks `CREATE QUOTA ...`.
	statementKindCreateQuota

	// statementKindAlterQuota marks `ALTER QUOTA ...`.
	statementKindAlterQuota

	// statementKindDropQuota marks `DROP QUOTA ...`.
	statementKindDropQuota

	// statementKindCreateSettingsProfile marks `CREATE SETTINGS PROFILE ...`.
	statementKindCreateSettingsProfile

	// statementKindAlterSettingsProfile marks `ALTER SETTINGS PROFILE ...`.
	statementKindAlterSettingsProfile

	// statementKindDropSettingsProfile marks `DROP SETTINGS PROFILE ...`.
	statementKindDropSettingsProfile

	// statementKindGrant marks `GRANT ...`.
	statementKindGrant

	// statementKindRevoke marks `REVOKE ...`.
	statementKindRevoke

	// statementKindExplain marks `EXPLAIN ...` (read-only).
	statementKindExplain

	// statementKindDescribeTable marks `DESCRIBE [TABLE] ...` / `DESC ...` (read-only).
	statementKindDescribeTable

	// statementKindCheckTable marks `CHECK TABLE ...` (read-only).
	statementKindCheckTable

	// statementKindBackup marks `BACKUP ...`.
	statementKindBackup

	// statementKindRestore marks `RESTORE ...`.
	statementKindRestore

	// statementKindKillQuery marks `KILL QUERY ...`.
	statementKindKillQuery

	// statementKindKillMutation marks `KILL MUTATION ...`.
	statementKindKillMutation

	// statementKindAttachTable marks top-level `ATTACH TABLE ...`.
	statementKindAttachTable

	// statementKindDetachTable marks top-level `DETACH TABLE ...`.
	statementKindDetachTable

	// statementKindUnknown marks a statement that matched no recognised shape.
	statementKindUnknown
)

const (
	// defaultMaxParseDepth caps every user-driven recursion in the parser.
	//
	// It bounds analyseSelect (subquery and CTE nesting) and the expression precedence chain
	// (parenthesis nesting). The cap is essential because Go raises a fatal, non-recoverable
	// stack overflow that the engine's recover guards cannot catch, so deeply nested input
	// would otherwise crash the host process. The value 256 sits far below the overflow
	// threshold yet out of the way for realistic queries; callers may override it with
	// WithMaxParseDepth. It matches the duckdb engine's defaultMaxParseDepth value (see
	// wdk/db/db_engine_duckdb/parser.go) so adversarial inputs bounded at one engine cannot
	// escape the bound when run through another.
	defaultMaxParseDepth = 256

	// ifNotExistsTokenWidth is the number of tokens consumed by the `IF NOT EXISTS` clause
	// when skipping it as a CREATE prefix.
	ifNotExistsTokenWidth = 3

	// multiWordObjectKeywordTokenCount is the minimum token count required to recognise a
	// two-word object kind ("ROW POLICY", "SETTINGS PROFILE") on a CREATE / ALTER / DROP
	// prefix.
	multiWordObjectKeywordTokenCount = 3

	// classifyPolicyKeyword is the POLICY keyword string the multi-word object classifiers
	// match on.
	classifyPolicyKeyword = "POLICY"

	// classifyProfileKeyword is the PROFILE keyword string the multi-word object classifiers
	// match on.
	classifyProfileKeyword = "PROFILE"

	// objectKindTable is the TABLE object keyword that may follow ATTACH / DETACH at the top
	// level, shared by the classifier and the parser.
	objectKindTable = "TABLE"

	// objectKindView is the VIEW object keyword that may follow ATTACH / DETACH at the top
	// level, shared by the classifier and the parser.
	objectKindView = "VIEW"

	// objectKindDictionary is the DICTIONARY object keyword that may follow ATTACH / DETACH
	// at the top level, shared by the classifier and the parser.
	objectKindDictionary = "DICTIONARY"

	// objectKindDatabase is the DATABASE object keyword that may follow ATTACH / DETACH at
	// the top level, shared by the classifier and the parser.
	objectKindDatabase = "DATABASE"

	// maxTokensPerStatement bounds the per-statement token stream length the consume helpers
	// walk.
	//
	// Real ClickHouse statements rarely exceed a few hundred tokens; the 100k headroom
	// covers generated SQL with large IN lists while still cutting off an adversarial input
	// that would otherwise drive a paren-balanced scan into an unbounded loop. The budget is
	// enforced by a single upfront whole-statement check in ApplyDDL and AnalyseQuery (`if
	// len(parsed.tokens) > maxTokensPerStatement`) run before any parsing begins, which
	// returns errTokenBudgetExceeded directly; there is no per-helper check or recover path.
	maxTokensPerStatement = 100_000
)

var (
	// errUnmatchedParenthesis is the canonical mismatched-paren sentinel returned when a
	// balanced-paren scan exhausts its input.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")

	// errAnalysisDepthExceeded is the recursion-depth sentinel returned when analysis
	// nesting exceeds maxParseDepth.
	errAnalysisDepthExceeded = errors.New("clickhouse: analysis recursion depth exceeded")

	// errUnexpectedEndOfWithInput is returned when a WITH clause ends before its body is
	// complete.
	errUnexpectedEndOfWithInput = errors.New("unexpected end of input in WITH clause")

	// errTokenBudgetExceeded is returned when a statement scan walks more tokens than
	// maxTokensPerStatement permits.
	errTokenBudgetExceeded = errors.New("clickhouse: per-statement token budget exceeded")

	// firstWordClassifiers dispatches on the first keyword to the appropriate sub-classifier
	// when the second token is also needed for disambiguation (e.g. CREATE TABLE vs CREATE
	// MATERIALIZED VIEW).
	firstWordClassifiers = map[string]func([]token) statementKind{
		"CREATE":   classifyCreateStatement,
		"DROP":     classifyDropStatement,
		"ALTER":    classifyAlterStatement,
		"RENAME":   classifyRenameStatement,
		"EXCHANGE": classifyExchangeStatement,
		"WITH":     classifyWithStatement,
		"DESCRIBE": classifyDescribeStatement,
		"DESC":     classifyDescribeStatement,
		"CHECK":    classifyCheckStatement,
		"KILL":     classifyKillStatement,
		"ATTACH":   classifyAttachStatement,
		"DETACH":   classifyDetachStatement,
	}

	// firstWordStaticKinds is the fast path for statements whose first keyword unambiguously
	// identifies the kind.
	firstWordStaticKinds = map[string]statementKind{
		"SELECT":   statementKindSelect,
		"INSERT":   statementKindInsert,
		"DELETE":   statementKindDelete,
		"TRUNCATE": statementKindTruncate,
		"OPTIMIZE": statementKindOptimize,
		"SYSTEM":   statementKindSystem,
		"USE":      statementKindUse,
		"SHOW":     statementKindShow,
		"SET":      statementKindSet,
		"EXPLAIN":  statementKindExplain,
		"BACKUP":   statementKindBackup,
		"RESTORE":  statementKindRestore,
		"GRANT":    statementKindGrant,
		"REVOKE":   statementKindRevoke,
	}

	// createObjectKinds dispatches CREATE-prefixed statements by the object keyword.
	// MATERIALIZED VIEW is handled specially because it is two words.
	createObjectKinds = map[string]statementKind{
		"TABLE":      statementKindCreateTable,
		"VIEW":       statementKindCreateView,
		"DICTIONARY": statementKindCreateDictionary,
		"FUNCTION":   statementKindCreateFunction,
		"DATABASE":   statementKindCreateDatabase,
		"USER":       statementKindCreateUser,
		"ROLE":       statementKindCreateRole,
		"QUOTA":      statementKindCreateQuota,
	}

	// dropObjectKinds dispatches DROP-prefixed statements by the object keyword.
	dropObjectKinds = map[string]statementKind{
		"TABLE":      statementKindDropTable,
		"VIEW":       statementKindDropView,
		"DICTIONARY": statementKindDropDictionary,
		"FUNCTION":   statementKindDropFunction,
		"DATABASE":   statementKindDropDatabase,
		"USER":       statementKindDropUser,
		"ROLE":       statementKindDropRole,
		"QUOTA":      statementKindDropQuota,
	}

	// alterObjectKinds dispatches ALTER-prefixed statements by the object keyword (USER /
	// ROLE / QUOTA). TABLE has its own action grammar and POLICY / PROFILE forms are handled
	// inline by classifyAlterStatement.
	alterObjectKinds = map[string]statementKind{
		"USER":  statementKindAlterUser,
		"ROLE":  statementKindAlterRole,
		"QUOTA": statementKindAlterQuota,
	}
)

// parsedStatement is the engine-private payload attached to each
// querier_dto.ParsedStatement. It carries the lexed tokens for this statement plus the
// classification result so downstream handlers do not re-scan.
type parsedStatement struct {
	// tokens holds the lexed tokens for this statement.
	tokens []token

	// kind holds the classification result for this statement.
	kind statementKind
}

// IsParsedStatement marks parsedStatement as implementing the marker interface required
// by querier_dto.ParsedStatement.
func (*parsedStatement) IsParsedStatement() {}

// parser is the per-statement parsing state for the ClickHouse engine. It maintains a
// forward-only cursor through the token stream plus the accumulator state used during DML
// analysis.
type parser struct {
	// namedParameterMap deduplicates `{name:Type}` placeholders by name so the generated Go
	// method exposes one parameter per distinct name.
	namedParameterMap map[string]int

	// firstParameterTypeError captures the first `{name:Type}` placeholder whose type tag
	// failed to parse.
	//
	// Surfacing the cause lets analyseSelect / analyseInsert wrap it into a
	// diagnostic-friendly error instead of silently degrading the parameter to an untyped
	// binding.
	firstParameterTypeError error

	// tokens holds the lexed token stream being parsed.
	tokens []token

	// position is the forward-only cursor into tokens.
	position int

	// parameterCount tracks the number of positional parameters seen so far.
	parameterCount int

	// analysisDepth bounds recursion through analyseSelect across compound branches, CTE
	// bodies, derived tables, scalar subqueries, and view bodies. Mirrors the duckdb /
	// postgres convention.
	analysisDepth int

	// expressionDepth bounds recursion through the parseExpression chain (lambda then or
	// then and through primary to paren).
	//
	// Without the bound, deeply nested expressions like `((((... ((1)) ...))))` can drive
	// the parser past the Go stack limit. It uses the same threshold as analysisDepth.
	expressionDepth int

	// maxParseDepth is the effective cap for analysisDepth and expressionDepth. newParser
	// seeds it with defaultMaxParseDepth; the engine overrides it from the dialect and child
	// parsers inherit it so the global cap holds across instances.
	maxParseDepth int
}

// newParser builds a parser positioned at the start of the supplied token stream with the
// default parse-depth cap and an empty named-parameter map.
//
// Takes tokens ([]token) which is the lexed statement to parse.
//
// Returns *parser which is the ready-to-use parser state.
func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		maxParseDepth:     defaultMaxParseDepth,
		namedParameterMap: make(map[string]int),
	}
}

// splitStatements partitions the token stream by top-level semicolons.
//
// ClickHouse does not use BEGIN..END procedural blocks the way postgres does (UDFs are
// lambda expressions, not statement blocks), so the splitter is straightforward.
//
// Takes tokens ([]token) which is the lexed input to partition.
//
// Returns [][]token which is one token slice per statement, EOF excluded.
func splitStatements(tokens []token) [][]token {
	var statements [][]token
	var current []token

	for _, tok := range tokens {
		if tok.kind == tokenEOF {
			break
		}
		if tok.kind == tokenSemicolon {
			if len(current) > 0 {
				statements = append(statements, current)
				current = nil
			}
			continue
		}
		current = append(current, tok)
	}

	if len(current) > 0 {
		statements = append(statements, current)
	}

	return statements
}

// classifyStatement identifies the kind of the statement represented by the token slice.
//
// The caller (ApplyDDL or AnalyseQuery) decides how to surface an unrecognised statement.
//
// Takes tokens ([]token) which is the statement to classify.
//
// Returns statementKind which is the recognised kind, or statementKindUnknown when the
// leading token matches no recognised shape.
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

// classifyCreateStatement recognises the CREATE family, including the two-word
// MATERIALIZED VIEW, ROW POLICY, and SETTINGS PROFILE object shapes.
//
// Takes tokens ([]token) which is the CREATE-prefixed statement.
//
// Returns statementKind which is the recognised CREATE kind, or statementKindUnknown when
// the object keyword is missing or unknown.
func classifyCreateStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	index := skipCreatePrefixes(tokens)
	if index >= len(tokens) {
		return statementKindUnknown
	}

	upper := strings.ToUpper(tokens[index].value)

	if upper == "MATERIALIZED" && index+1 < len(tokens) && strings.EqualFold(tokens[index+1].value, "VIEW") {
		return statementKindCreateMaterializedView
	}

	if upper == "ROW" && index+1 < len(tokens) && strings.EqualFold(tokens[index+1].value, classifyPolicyKeyword) {
		return statementKindCreatePolicy
	}
	if upper == classifyPolicyKeyword {
		return statementKindCreatePolicy
	}
	if upper == "SETTINGS" && index+1 < len(tokens) && strings.EqualFold(tokens[index+1].value, classifyProfileKeyword) {
		return statementKindCreateSettingsProfile
	}
	if upper == classifyProfileKeyword {
		return statementKindCreateSettingsProfile
	}

	if kind, found := createObjectKinds[upper]; found {
		return kind
	}

	return statementKindUnknown
}

// skipCreatePrefixes advances past optional `OR REPLACE` / `TEMPORARY` / `IF NOT EXISTS`
// modifiers to land on the object-kind keyword.
//
// Takes tokens ([]token) which is the CREATE-prefixed statement.
//
// Returns int which is the index of the next interesting token, or len(tokens) when the
// input is malformed.
func skipCreatePrefixes(tokens []token) int {
	index := 1
	for index < len(tokens) {
		upper := strings.ToUpper(tokens[index].value)
		switch upper {
		case "OR":
			if index+1 < len(tokens) && strings.EqualFold(tokens[index+1].value, "REPLACE") {
				index += 2
				continue
			}
			return index
		case "TEMPORARY", "TEMP":
			index++
			continue
		case "IF":
			if index+2 < len(tokens) &&
				strings.EqualFold(tokens[index+1].value, "NOT") &&
				strings.EqualFold(tokens[index+2].value, "EXISTS") {
				index += ifNotExistsTokenWidth
				continue
			}
			return index
		default:
			return index
		}
	}
	return index
}

// classifyDropStatement recognises the DROP family, including the two-word ROW POLICY and
// SETTINGS PROFILE object shapes.
//
// Takes tokens ([]token) which is the DROP-prefixed statement.
//
// Returns statementKind which is the recognised DROP kind, or statementKindUnknown when
// the object keyword is missing or unknown.
func classifyDropStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}
	second := strings.ToUpper(tokens[1].value)
	if second == "ROW" && len(tokens) >= multiWordObjectKeywordTokenCount && strings.EqualFold(tokens[2].value, classifyPolicyKeyword) {
		return statementKindDropPolicy
	}
	if second == classifyPolicyKeyword {
		return statementKindDropPolicy
	}
	if second == "SETTINGS" && len(tokens) >= multiWordObjectKeywordTokenCount && strings.EqualFold(tokens[2].value, classifyProfileKeyword) {
		return statementKindDropSettingsProfile
	}
	if second == classifyProfileKeyword {
		return statementKindDropSettingsProfile
	}
	if kind, found := dropObjectKinds[second]; found {
		return kind
	}
	return statementKindUnknown
}

// classifyAlterStatement recognises ALTER TABLE (which covers ClickHouse's ADD COLUMN /
// DROP COLUMN / MODIFY COLUMN / UPDATE / DELETE / RENAME COLUMN / MATERIALIZE COLUMN /
// MODIFY TTL / MODIFY ORDER BY sub-forms) plus the RBAC ALTER family (USER / ROLE / QUOTA
// / ROW POLICY / SETTINGS PROFILE).
//
// Takes tokens ([]token) which is the ALTER-prefixed statement.
//
// Returns statementKind which is the recognised ALTER kind, or statementKindUnknown when
// the object keyword is missing or unknown.
func classifyAlterStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}
	second := strings.ToUpper(tokens[1].value)
	if second == "TABLE" {
		return statementKindAlterTable
	}
	if second == "ROW" && len(tokens) >= multiWordObjectKeywordTokenCount && strings.EqualFold(tokens[2].value, classifyPolicyKeyword) {
		return statementKindAlterPolicy
	}
	if second == classifyPolicyKeyword {
		return statementKindAlterPolicy
	}
	if second == "SETTINGS" && len(tokens) >= multiWordObjectKeywordTokenCount && strings.EqualFold(tokens[2].value, classifyProfileKeyword) {
		return statementKindAlterSettingsProfile
	}
	if second == classifyProfileKeyword {
		return statementKindAlterSettingsProfile
	}
	if kind, found := alterObjectKinds[second]; found {
		return kind
	}
	return statementKindUnknown
}

// classifyDescribeStatement recognises `DESCRIBE [TABLE] name` / `DESC name`.
//
// All forms route to statementKindDescribeTable.
//
// Returns statementKind which is always statementKindDescribeTable.
func classifyDescribeStatement(_ []token) statementKind {
	return statementKindDescribeTable
}

// classifyCheckStatement recognises `CHECK TABLE name [PARTITION ...] [PART ...]`.
//
// Takes tokens ([]token) which is the CHECK-prefixed statement.
//
// Returns statementKind which is statementKindCheckTable on a match, or
// statementKindUnknown otherwise.
func classifyCheckStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}
	second := strings.ToUpper(tokens[1].value)
	if second == "TABLE" {
		return statementKindCheckTable
	}
	return statementKindUnknown
}

// classifyKillStatement recognises `KILL QUERY ...` and `KILL MUTATION ...`.
//
// Takes tokens ([]token) which is the KILL-prefixed statement.
//
// Returns statementKind which is statementKindKillQuery or statementKindKillMutation on a
// match, or statementKindUnknown otherwise.
func classifyKillStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}
	switch strings.ToUpper(tokens[1].value) {
	case "QUERY":
		return statementKindKillQuery
	case "MUTATION":
		return statementKindKillMutation
	default:
		return statementKindUnknown
	}
}

// classifyAttachStatement recognises top-level `ATTACH {TABLE | VIEW | DICTIONARY |
// DATABASE} name`.
//
// Bare `ATTACH name` (without one of the object keywords) falls back to
// statementKindUnknown so the caller can surface a diagnostic rather than misclassifying
// an unrelated statement as an ATTACH TABLE.
//
// Takes tokens ([]token) which is the ATTACH-prefixed statement.
//
// Returns statementKind which is statementKindAttachTable on a match, or
// statementKindUnknown otherwise.
func classifyAttachStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}
	switch strings.ToUpper(tokens[1].value) {
	case objectKindTable, objectKindView, objectKindDictionary, objectKindDatabase:
		return statementKindAttachTable
	default:
		return statementKindUnknown
	}
}

// classifyDetachStatement recognises top-level `DETACH {TABLE | VIEW | DICTIONARY |
// DATABASE} name`.
//
// Bare `DETACH name` (without one of the object keywords) falls back to
// statementKindUnknown so the caller can surface a diagnostic rather than misclassifying
// an unrelated statement as a DETACH TABLE.
//
// Takes tokens ([]token) which is the DETACH-prefixed statement.
//
// Returns statementKind which is statementKindDetachTable on a match, or
// statementKindUnknown otherwise.
func classifyDetachStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}
	switch strings.ToUpper(tokens[1].value) {
	case objectKindTable, objectKindView, objectKindDictionary, objectKindDatabase:
		return statementKindDetachTable
	default:
		return statementKindUnknown
	}
}

// classifyRenameStatement recognises `RENAME TABLE` / `RENAME DATABASE` / `RENAME
// DICTIONARY`.
//
// All three collapse to a single rename mutation kind; the parser side disambiguates and
// emits the appropriate catalogue change.
//
// Takes tokens ([]token) which is the RENAME-prefixed statement.
//
// Returns statementKind which is statementKindRenameTable on a match, or
// statementKindUnknown otherwise.
func classifyRenameStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}
	second := strings.ToUpper(tokens[1].value)
	if second == "TABLE" || second == "DICTIONARY" || second == "DATABASE" {
		return statementKindRenameTable
	}
	return statementKindUnknown
}

// classifyExchangeStatement recognises `EXCHANGE TABLES a AND b`, ClickHouse's atomic
// two-table swap.
//
// The classifier only checks for the leading `EXCHANGE TABLES` token sequence; the parser
// body validates the AND token and the optional ON CLUSTER clause.
//
// Takes tokens ([]token) which is the EXCHANGE-prefixed statement.
//
// Returns statementKind which is statementKindExchangeTables on a match, or
// statementKindUnknown otherwise.
func classifyExchangeStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}
	second := strings.ToUpper(tokens[1].value)
	if second == "TABLES" {
		return statementKindExchangeTables
	}
	return statementKindUnknown
}

// classifyWithStatement walks a WITH-prefixed statement to find the DML keyword (SELECT /
// INSERT) that follows the CTE list.
//
// It mirrors the duckdb implementation, tracking parenthesis depth and returning on the
// first top-level DML keyword.
//
// Takes tokens ([]token) which is the WITH-prefixed statement.
//
// Returns statementKind which is statementKindSelect or statementKindInsert, defaulting
// to statementKindSelect when no DML keyword is found.
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
		switch strings.ToUpper(tok.value) {
		case "SELECT":
			return statementKindSelect
		case "INSERT":
			return statementKindInsert
		}
	}
	return statementKindSelect
}

// current returns the token at the cursor without consuming it.
//
// Returns token which is the cursor token, or a synthesised EOF token when the cursor is
// past the end so callers need not bounds-check on every read.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token one past the cursor without consuming it.
//
// It is used for two-token disambiguation in DDL parsing.
//
// Returns token which is the lookahead token, or an EOF sentinel when out of range.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// peekAt returns the token at the given absolute index without consuming it.
//
// Takes index (int) which is the absolute token index to read.
//
// Returns token which is the token at index, or an EOF sentinel when the index is out of
// range.
func (p *parser) peekAt(index int) token {
	if index < 0 || index >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[index]
}

// advance consumes and returns the current token.
//
// Returns token which is the consumed token, or the same EOF sentinel as current() when
// at end so callers can chain.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// atEnd reports whether the cursor has consumed the entire token stream.
//
// Returns bool which is true when the cursor is at or past the end, or on EOF.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// expectKeyword consumes the next token if it matches any of the supplied keywords
// (case-insensitive).
//
// Takes keywords (...string) which are the accepted keyword spellings.
//
// Returns token which is the matched token on success.
// Returns error when the next token matches none of the keywords.
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

// matchKeyword conditionally consumes the next token if it matches the supplied keyword
// (case-insensitive).
//
// Takes keyword (string) which is the keyword to match.
//
// Returns bool which is true when the token was consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token is the supplied keyword (case-insensitive)
// without advancing the cursor.
//
// Takes keyword (string) which is the keyword to test.
//
// Returns bool which is true when the current token matches the keyword.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the supplied keywords
// (case-insensitive) without advancing the cursor.
//
// Takes keywords (...string) which are the keywords to test.
//
// Returns bool which is true when the current token matches any keyword.
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

// parseIdentifierOrKeyword reads the next token as an identifier.
//
// It accepts both bare identifiers and quoted forms (any token kind flagged
// tokenIdentifier by the tokeniser).
//
// Returns string which is the identifier text.
// Returns error when the next token is not an identifier.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// parseDatabaseQualifiedName reads either `name` or `database.name`.
//
// ClickHouse uses `database.table` qualification (postgres-style `schema.table`); the
// Piko catalogue treats the database as a schema for uniform resolution.
//
// Returns database (string) which is the qualifying database, empty when absent.
// Returns name (string) which is the object name.
// Returns err (error) when an identifier cannot be read.
func (p *parser) parseDatabaseQualifiedName() (database string, name string, err error) {
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

// skipParenthesised consumes a balanced parenthesised block starting at the current
// cursor position.
//
// It is used to skip past column-definition lists and parameter lists when the caller
// does not care about the contents.
//
// Returns error when the cursor is not on `(` or the parens are unbalanced.
func (p *parser) skipParenthesised() error {
	if p.current().kind != tokenLeftParen {
		return fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	depth := 1
	for depth > 0 && !p.atEnd() {
		switch p.current().kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		default:
		}
		p.advance()
	}
	if depth != 0 {
		return errUnmatchedParenthesis
	}
	return nil
}

// collectParenthesised consumes a balanced parenthesised block and returns the inner
// tokens, excluding the outer parens.
//
// It is used when the caller needs to re-parse the inner content (for example column
// definitions or function bodies).
//
// Returns []token which is the inner token slice.
// Returns error when the cursor is not on `(` or the parens are unbalanced.
func (p *parser) collectParenthesised() ([]token, error) {
	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	var inner []token
	depth := 1
	for depth > 0 && !p.atEnd() {
		tok := p.current()
		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
			if depth == 0 {
				p.advance()
				return inner, nil
			}
		default:
		}
		inner = append(inner, tok)
		p.advance()
	}
	return nil, errUnmatchedParenthesis
}

// mustKeyword is the panic-on-mismatch variant of expectKeyword.
//
// IMPORTANT RECOVER PRECONDITION: mustKeyword panics on mismatch. Callers MUST run inside
// a recover-protected frame. The public-API entry points ApplyDDL and AnalyseQuery
// (engine.go) install the required defer/recover at the top of each call, wrapping the
// recovered value plus a captured stack trace into a returned error so a malformed
// statement cannot crash the calling apply loop. Any new entry point that invokes
// mustKeyword (directly or transitively) must install its own recover or it must only be
// called when the caller has already classified the statement so a mismatch indicates a
// tokeniser invariant break, not user input.
//
// Takes keywords (...string) which are the accepted keyword spellings.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}
