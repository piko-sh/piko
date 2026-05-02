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

package db_engine_timescaledb

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

const (
	// statementKindCreateHypertable classifies a CREATE HYPERTABLE form. Each kind sits in
	// the reserved [StatementKindExtensionBase, StatementKindExtensionBase + 1000) range and
	// dispatches through tsExtension.Parse, with the host engine ignoring the exact numeric
	// value as long as it falls within the reserved window.
	statementKindCreateHypertable db_engine_postgres.StatementKind = db_engine_postgres.StatementKindExtensionBase + iota

	// statementKindCreateHypertableCall classifies a SELECT create_hypertable() migration
	// call form.
	statementKindCreateHypertableCall

	// statementKindCreateContinuousAggregate classifies a CREATE MATERIALIZED VIEW
	// continuous-aggregate form.
	statementKindCreateContinuousAggregate

	// statementKindAlterTableCompression classifies an ALTER ... SET (timescaledb.compress =
	// ...) form.
	statementKindAlterTableCompression

	// statementKindAddCompressionPolicy classifies a SELECT add_compression_policy(...)
	// form, which the host postgres classifier would otherwise label as a plain SELECT,
	// losing the structured metadata downstream consumers need.
	statementKindAddCompressionPolicy

	// statementKindRemoveCompressionPolicy classifies a SELECT
	// remove_compression_policy(...) form.
	statementKindRemoveCompressionPolicy

	// statementKindAddColumnstorePolicy classifies a SELECT add_columnstore_policy(...)
	// form.
	statementKindAddColumnstorePolicy

	// statementKindRemoveColumnstorePolicy classifies a SELECT
	// remove_columnstore_policy(...) form.
	statementKindRemoveColumnstorePolicy

	// statementKindAddRetentionPolicy classifies a SELECT add_retention_policy(...) form.
	statementKindAddRetentionPolicy

	// statementKindRemoveRetentionPolicy classifies a SELECT remove_retention_policy(...)
	// form.
	statementKindRemoveRetentionPolicy

	// statementKindAddContinuousAggregatePolicy classifies a SELECT
	// add_continuous_aggregate_policy(...) form.
	statementKindAddContinuousAggregatePolicy

	// statementKindRemoveContinuousAggregatePolicy classifies a SELECT
	// remove_continuous_aggregate_policy(...) form.
	statementKindRemoveContinuousAggregatePolicy

	// statementKindAddReorderPolicy classifies a SELECT add_reorder_policy(...) form.
	statementKindAddReorderPolicy

	// statementKindRemoveReorderPolicy classifies a SELECT remove_reorder_policy(...) form.
	statementKindRemoveReorderPolicy

	// statementKindAddJob classifies a SELECT add_job(...) form. The job machinery is the
	// lower-level surface that the policy helpers drive, and classifying it explicitly lets
	// downstream consumers observe both the high-level policy intent and the raw job
	// operation.
	statementKindAddJob

	// statementKindAlterJob classifies a SELECT alter_job(...) form.
	statementKindAlterJob

	// statementKindDeleteJob classifies a SELECT delete_job(...) form.
	statementKindDeleteJob

	// statementKindRunJob classifies a SELECT run_job(...) form.
	statementKindRunJob

	// statementKindAddDimension classifies a SELECT add_dimension(...) form. These calls
	// mutate hypertable metadata in place, so the catalogue treats them as
	// MutationAlterTable: the existing relation gains an EngineSpecific marker without a
	// duplicate CREATE TABLE arrival.
	statementKindAddDimension

	// statementKindSetChunkTimeInterval classifies a SELECT set_chunk_time_interval(...)
	// form.
	statementKindSetChunkTimeInterval

	// statementKindSetIntegerNowFunc classifies a SELECT set_integer_now_func(...) form.
	statementKindSetIntegerNowFunc

	// statementKindEnableChunkSkipping classifies a SELECT enable_chunk_skipping(...) form.
	statementKindEnableChunkSkipping

	// statementKindDisableChunkSkipping classifies a SELECT disable_chunk_skipping(...)
	// form.
	statementKindDisableChunkSkipping

	// statementKindRefreshContinuousAggregate captures a CALL
	// refresh_continuous_aggregate(...) form. Postgres uses CALL for procedure invocation,
	// which the built-in classifier labels as Unknown, so lifting it lets
	// continuous-aggregate refreshes surface as structured mutations.
	statementKindRefreshContinuousAggregate

	// statementKindCountSentinel marks the slot one past the last declared TimescaleDB
	// statement kind. The compile-time assertion below pins this value to
	// tsExtensionMaxKinds so adding or removing a kind without updating the audit constant
	// fails the build rather than silently drifting.
	statementKindCountSentinel
)

const (
	// declaredStatementKindCount is the number of statement kinds the extension declares
	// above. The named constant keeps the compile-time invariant below short and
	// self-describing without sprinkling raw casts across the assertion line.
	declaredStatementKindCount = int(statementKindCountSentinel) - int(statementKindCreateHypertable)

	// tsExtensionMaxKinds is the exclusive upper bound of statement kinds the TimescaleDB
	// extension owns.
	//
	// It is set explicitly so a contributor can audit the entire range from a single
	// declaration without counting case arms by hand, and adding a new statement kind
	// requires incrementing it so audits stay in lockstep. The runtime path does not read
	// this value; it exists to anchor the audit contract.
	tsExtensionMaxKinds = 24

	// minTokensForContinuousAggregate is the smallest token count that could form a CREATE
	// MATERIALIZED VIEW continuous-aggregate statement before any reloption body. Anything
	// shorter cannot match the pattern.
	minTokensForContinuousAggregate = 4

	// minTokensForHypertableCall is the smallest token count for a `SELECT
	// create_hypertable(...)` migration call (SELECT + name + open paren + at least one
	// argument).
	minTokensForHypertableCall = 4

	// minTokensForAlterCompression is the smallest token count for an ALTER TABLE or ALTER
	// MATERIALIZED VIEW ... SET (timescaledb.compress = ...) statement.
	//
	// The compression reloption keyword and its enclosing parens push the minimum to five
	// tokens.
	minTokensForAlterCompression = 5

	// minTokensForSelectFunctionCall is the smallest token count for `SELECT name(arg)`
	// (SELECT + identifier + open paren + close paren).
	minTokensForSelectFunctionCall = 4

	// minTokensForCallStatement is the smallest token count for `CALL name(arg)` (CALL +
	// identifier + open paren + close paren).
	minTokensForCallStatement = 4

	// continuousAggregateSchemaTokens is the offset from a candidate `timescaledb`
	// identifier to the trailing `continuous` identifier inside a reloption body
	// (timescaledb . continuous = ...).
	continuousAggregateSchemaTokens = 2

	// alterCompressionMinLookahead is the offset from a `SET` token at which the
	// alter-compression matcher still has enough tokens to inspect a `( timescaledb` payload
	// (SET ident ( ident -> +3).
	alterCompressionMinLookahead = 3

	// alterCompressionMaxOffset bounds the search window for the reloption `(` that follows
	// a SET keyword inside an ALTER TABLE ... SET (...) compression statement.
	//
	// Four tokens is enough to skip optional schema or column qualifiers between SET and the
	// payload paren.
	alterCompressionMaxOffset = 4
)

var (
	_ = [1]struct{}{}[declaredStatementKindCount-tsExtensionMaxKinds]

	// policyClassifierTable maps a lower-cased function-name token to the extension
	// statement kind that owns it.
	//
	// SELECT-form policy calls and hypertable-management calls share the same dispatch path
	// through parsePolicyCall, so registering a new function is a one-line edit here plus a
	// parser-side handler hook. Centralising the mapping keeps the classifier branchless and
	// makes the audited surface trivially greppable.
	policyClassifierTable = map[string]db_engine_postgres.StatementKind{
		"add_compression_policy":             statementKindAddCompressionPolicy,
		"remove_compression_policy":          statementKindRemoveCompressionPolicy,
		"add_columnstore_policy":             statementKindAddColumnstorePolicy,
		"remove_columnstore_policy":          statementKindRemoveColumnstorePolicy,
		"add_retention_policy":               statementKindAddRetentionPolicy,
		"remove_retention_policy":            statementKindRemoveRetentionPolicy,
		"add_continuous_aggregate_policy":    statementKindAddContinuousAggregatePolicy,
		"remove_continuous_aggregate_policy": statementKindRemoveContinuousAggregatePolicy,
		"add_reorder_policy":                 statementKindAddReorderPolicy,
		"remove_reorder_policy":              statementKindRemoveReorderPolicy,
		"add_job":                            statementKindAddJob,
		"alter_job":                          statementKindAlterJob,
		"delete_job":                         statementKindDeleteJob,
		"run_job":                            statementKindRunJob,
		"add_dimension":                      statementKindAddDimension,
		"set_chunk_time_interval":            statementKindSetChunkTimeInterval,
		"set_integer_now_func":               statementKindSetIntegerNowFunc,
		"enable_chunk_skipping":              statementKindEnableChunkSkipping,
		"disable_chunk_skipping":             statementKindDisableChunkSkipping,
	}
)

// tsExtension implements db_engine_postgres.StatementExtension for TimescaleDB-specific
// DDL. The classifier inspects the leading tokens of each statement; the parser
// dispatches to the right builder based on the recognised kind.
type tsExtension struct{}

// newTimescaleExtension constructs a stateless extension instance for registration with
// the postgres engine.
//
// Returns tsExtension which is the ready-to-register extension value.
func newTimescaleExtension() tsExtension {
	return tsExtension{}
}

// Classify returns a non-zero StatementKind for statements this extension claims.
//
// The recognised shapes are CREATE HYPERTABLE name (cols...), CREATE MATERIALIZED VIEW
// name WITH (timescaledb.continuous = ...), SELECT create_hypertable('table', 'column',
// ...), ALTER ... SET (timescaledb.compress = ...), the SELECT add_*_policy /
// remove_*_policy and *_job calls, the SELECT add_dimension and set_chunk_time_interval
// style management calls, and CALL refresh_continuous_aggregate(...).
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
//
// Returns db_engine_postgres.StatementKind which is the matched kind, or 0 when this
// extension declines the statement.
func (tsExtension) Classify(tokens []db_engine_postgres.Token) db_engine_postgres.StatementKind {
	switch {
	case matchesCreateHypertable(tokens):
		return statementKindCreateHypertable
	case matchesCreateContinuousAggregate(tokens):
		return statementKindCreateContinuousAggregate
	case matchesCreateHypertableCall(tokens):
		return statementKindCreateHypertableCall
	case matchesAlterTableCompression(tokens):
		return statementKindAlterTableCompression
	case matchesCallRefreshContinuousAggregate(tokens):
		return statementKindRefreshContinuousAggregate
	}
	if kind := classifySelectFunctionCall(tokens); kind != 0 {
		return kind
	}
	return 0
}

// Parse dispatches based on the previously-classified kind.
//
// Takes p (db_engine_postgres.ParserContext) which is the parser context for the
// statement being parsed.
// Takes kind (db_engine_postgres.StatementKind) which is the kind Classify returned
// earlier for this statement.
//
// Returns *querier_dto.CatalogueMutation which is the parsed mutation.
// Returns error when the kind is not one this extension owns.
func (tsExtension) Parse(p db_engine_postgres.ParserContext, kind db_engine_postgres.StatementKind) (*querier_dto.CatalogueMutation, error) {
	switch kind {
	case statementKindCreateHypertable:
		return parseCreateHypertable(p)
	case statementKindCreateContinuousAggregate:
		return parseCreateContinuousAggregate(p)
	case statementKindCreateHypertableCall:
		return parseCreateHypertableCall(p)
	case statementKindAlterTableCompression:
		return parseAlterTableCompression(p)
	case statementKindRefreshContinuousAggregate:
		return parseRefreshContinuousAggregateCall(p)
	}
	if isPolicyOrManagementKind(kind) {
		return parsePolicyCall(p, kind)
	}
	return nil, fmt.Errorf("timescaledb: unknown extension kind %d", kind)
}

// isPolicyOrManagementKind reports whether kind belongs to the SELECT-form policy or
// hypertable-management family handled by parsePolicyCall.
//
// The check uses a range comparison rather than a case list so new kinds added to
// policyClassifierTable do not need a matching dispatcher arm; the parser handles
// classification through the function-name lookup it performed earlier.
//
// Takes kind (db_engine_postgres.StatementKind) which is the kind to test.
//
// Returns bool which is true when the kind falls in the policy or management range.
func isPolicyOrManagementKind(kind db_engine_postgres.StatementKind) bool {
	return kind >= statementKindAddCompressionPolicy && kind <= statementKindDisableChunkSkipping
}

// classifySelectFunctionCall walks the SELECT-form policy and hypertable-management
// dispatch table and returns the matching kind, or 0 when the leading tokens do not name
// a recognised function. The function-name token must be immediately followed by `(` so a
// bare `SELECT add_job FROM job_registry` reference does not collide with the policy
// classifier.
//
// Takes tokens ([]db_engine_postgres.Token) which is the candidate statement's token
// slice.
//
// Returns db_engine_postgres.StatementKind which is the matched kind or 0 to decline.
func classifySelectFunctionCall(tokens []db_engine_postgres.Token) db_engine_postgres.StatementKind {
	if len(tokens) < minTokensForSelectFunctionCall {
		return 0
	}
	if !strings.EqualFold(tokens[0].Value(), "SELECT") {
		return 0
	}
	if tokens[2].Kind() != db_engine_postgres.TokenLeftParen {
		return 0
	}
	name := strings.ToLower(tokens[1].Value())
	if kind, ok := policyClassifierTable[name]; ok {
		return kind
	}
	return 0
}

// matchesCallRefreshContinuousAggregate reports whether the statement is `CALL
// refresh_continuous_aggregate(...)`.
//
// Postgres CALL is the procedure-invocation form, and TimescaleDB uses it for the
// incremental continuous-aggregate refresh helper because the procedure cannot run inside
// a regular SELECT transaction.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
//
// Returns bool which is true when the statement matches the CALL refresh shape.
func matchesCallRefreshContinuousAggregate(tokens []db_engine_postgres.Token) bool {
	if len(tokens) < minTokensForCallStatement {
		return false
	}
	if !strings.EqualFold(tokens[0].Value(), "CALL") {
		return false
	}
	if !strings.EqualFold(tokens[1].Value(), funcNameRefreshContinuousAggregate) {
		return false
	}
	return tokens[2].Kind() == db_engine_postgres.TokenLeftParen
}

// matchesCreateHypertable reports whether the tokens start with `CREATE [IF NOT EXISTS]
// HYPERTABLE`.
//
// This is TimescaleDB's keyword form of hypertable creation. TimescaleDB rejects `CREATE
// OR REPLACE HYPERTABLE`, so the classifier does not skip OR or REPLACE, and statements
// using that shape pass through to the built-in handler which produces the authentic
// postgres rejection message.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
//
// Returns bool which is true when the keyword form matches.
func matchesCreateHypertable(tokens []db_engine_postgres.Token) bool {
	if len(tokens) < 2 {
		return false
	}
	if !strings.EqualFold(tokens[0].Value(), "CREATE") {
		return false
	}
	for index := 1; index < len(tokens); index++ {
		upper := strings.ToUpper(tokens[index].Value())
		switch upper {
		case "IF", "NOT", "EXISTS":
			continue
		case "HYPERTABLE":
			return true
		default:
			return false
		}
	}
	return false
}

// matchesCreateContinuousAggregate reports whether the tokens form a CREATE
// continuous-aggregate statement.
//
// The recognised shape is `CREATE [OR REPLACE] MATERIALIZED VIEW ... WITH
// (timescaledb.continuous = ...)`. The check requires MATERIALIZED VIEW to appear in
// adjacent positions, skipping only the optional `OR REPLACE` and any `IF NOT EXISTS`
// prefix, then scans for `timescaledb.continuous` inside a parenthesised reloption body.
// Scanning for the reloption key only within a paren-depth-1 window prevents false
// positives where the SELECT body references `timescaledb.continuous` as an identifier.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
//
// Returns bool which is true when the continuous-aggregate shape matches.
func matchesCreateContinuousAggregate(tokens []db_engine_postgres.Token) bool {
	if len(tokens) < minTokensForContinuousAggregate {
		return false
	}
	if !strings.EqualFold(tokens[0].Value(), "CREATE") {
		return false
	}
	cursor, ok := skipCreateMaterializedViewPrefix(tokens)
	if !ok {
		return false
	}
	withIndex := findContinuousAggregateWithKeyword(tokens, cursor)
	if withIndex < 0 {
		return false
	}
	return reloptionBodyMentionsContinuous(tokens, withIndex+1)
}

// skipCreateMaterializedViewPrefix walks past the `CREATE [OR REPLACE] MATERIALIZED VIEW`
// keywords of a candidate continuous-aggregate statement.
//
// The caller has already verified that tokens[0] is CREATE.
//
// Takes tokens ([]db_engine_postgres.Token) which is the candidate statement's token
// slice.
//
// Returns int which is the cursor position past VIEW.
// Returns bool which is true when the leading prefix matched.
func skipCreateMaterializedViewPrefix(tokens []db_engine_postgres.Token) (int, bool) {
	cursor := 1
	if cursor < len(tokens) && strings.EqualFold(tokens[cursor].Value(), "OR") {
		cursor++
		if cursor >= len(tokens) || !strings.EqualFold(tokens[cursor].Value(), "REPLACE") {
			return cursor, false
		}
		cursor++
	}
	if cursor+1 >= len(tokens) {
		return cursor, false
	}
	if !strings.EqualFold(tokens[cursor].Value(), "MATERIALIZED") {
		return cursor, false
	}
	if !strings.EqualFold(tokens[cursor+1].Value(), "VIEW") {
		return cursor, false
	}
	return cursor + 2, true
}

// findContinuousAggregateWithKeyword scans forward from the view name for the first
// `WITH` token that is immediately followed by `(`.
//
// The reloption body is mandatory for a continuous aggregate, so a bare `WITH` such as
// one inside a SELECT body without an opening paren stops the search.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes start (int) which is the index of the first token to inspect.
//
// Returns int which is the absolute index of the `WITH` token, or -1 when no qualifying
// pair exists.
func findContinuousAggregateWithKeyword(tokens []db_engine_postgres.Token, start int) int {
	for index := start; index < len(tokens); index++ {
		if !strings.EqualFold(tokens[index].Value(), "WITH") {
			continue
		}
		if index+1 >= len(tokens) || tokens[index+1].Kind() != db_engine_postgres.TokenLeftParen {
			return -1
		}
		return index
	}
	return -1
}

// reloptionBodyMentionsContinuous reports whether the reloption body contains the
// `timescaledb . continuous` identifier triple at the first nesting level.
//
// The scan starts at openParenIndex and skips over nested parentheses, for example inside
// a default value expression, returning false when the body closes without finding the
// key. It also bails with false when nested parens exceed maxParenDepth so adversarial
// inputs do not provoke unbounded work, in which case the statement falls through to the
// built-in postgres handler.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes openParenIndex (int) which is the index of the `(` token that opens the reloption
// body.
//
// Returns bool which is true when the body contains the `timescaledb.continuous`
// reloption key.
func reloptionBodyMentionsContinuous(tokens []db_engine_postgres.Token, openParenIndex int) bool {
	depth := 0
	for index := openParenIndex; index < len(tokens); index++ {
		switch tokens[index].Value() {
		case "(":
			if depth >= maxParenDepth {
				return false
			}
			depth++
		case ")":
			depth--
			if depth == 0 {
				return false
			}
		}
		if depth != 1 {
			continue
		}
		if isTimescaledbContinuousTriple(tokens, index) {
			return true
		}
	}
	return false
}

// isTimescaledbContinuousTriple reports whether the three tokens starting at index spell
// the reloption key `timescaledb . continuous` (case-insensitive on the identifiers).
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes index (int) which is the candidate start position.
//
// Returns bool which is true when the triple is present.
func isTimescaledbContinuousTriple(tokens []db_engine_postgres.Token, index int) bool {
	if index+continuousAggregateSchemaTokens >= len(tokens) {
		return false
	}
	return strings.EqualFold(tokens[index].Value(), "timescaledb") &&
		tokens[index+1].Kind() == db_engine_postgres.TokenDot &&
		strings.EqualFold(tokens[index+continuousAggregateSchemaTokens].Value(), "continuous")
}

// matchesCreateHypertableCall reports whether the statement is a SELECT calling the
// create_hypertable() function, the most common migration form.
//
// It requires the identifier to be immediately followed by a `(` so that `SELECT id FROM
// create_hypertable_metadata` and other bare-identifier mentions do not collide.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
//
// Returns bool which is true when the SELECT create_hypertable call shape matches.
func matchesCreateHypertableCall(tokens []db_engine_postgres.Token) bool {
	if len(tokens) < minTokensForHypertableCall {
		return false
	}
	if !strings.EqualFold(tokens[0].Value(), "SELECT") {
		return false
	}
	if !strings.EqualFold(tokens[1].Value(), "create_hypertable") {
		return false
	}
	return tokens[2].Kind() == db_engine_postgres.TokenLeftParen
}

// matchesAlterTableCompression reports whether the statement is an ALTER compression
// statement.
//
// The recognised shapes are `ALTER TABLE x SET (timescaledb.compress = ...)` and the
// equivalent ALTER MATERIALIZED VIEW form. The matcher requires the second keyword to be
// TABLE or MATERIALIZED VIEW, then scans forward for a SET keyword followed by `(
// timescaledb`. The `SET SCHEMA` form is declined so the built-in postgres handler claims
// it.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
//
// Returns bool which is true when the ALTER compression shape matches.
func matchesAlterTableCompression(tokens []db_engine_postgres.Token) bool {
	if len(tokens) < minTokensForAlterCompression {
		return false
	}
	if !strings.EqualFold(tokens[0].Value(), "ALTER") {
		return false
	}
	if !alterTargetIsTableOrView(tokens) {
		return false
	}
	return scanForCompressionSetClause(tokens)
}

// alterTargetIsTableOrView reports whether the second keyword of an ALTER statement is
// `TABLE` or `MATERIALIZED VIEW`. The third token is required to be VIEW when
// MATERIALIZED is present.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice. The
// caller has already verified that len(tokens) is at least minTokensForAlterCompression.
//
// Returns bool which is true when the target keyword shape matches.
func alterTargetIsTableOrView(tokens []db_engine_postgres.Token) bool {
	upperSecond := strings.ToUpper(tokens[1].Value())
	if upperSecond == "MATERIALIZED" {
		return strings.EqualFold(tokens[2].Value(), "VIEW")
	}
	return upperSecond == "TABLE"
}

// scanForCompressionSetClause walks forward from the third token of an ALTER statement,
// looking for a `SET (timescaledb...` payload. The `SET SCHEMA` form is declined so that
// built-in postgres handlers claim it; the scan continues across additional SET keywords
// until one matches the compression shape or the tokens are exhausted.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
//
// Returns bool which is true when a compression-style SET payload is found.
func scanForCompressionSetClause(tokens []db_engine_postgres.Token) bool {
	for index := 2; index+alterCompressionMinLookahead < len(tokens); index++ {
		if !strings.EqualFold(tokens[index].Value(), "SET") {
			continue
		}

		if index+1 < len(tokens) && strings.EqualFold(tokens[index+1].Value(), "SCHEMA") {
			return false
		}
		if compressionPayloadStartsAt(tokens, index) {
			return true
		}
	}
	return false
}

// compressionPayloadStartsAt reports whether one of the tokens in the
// alterCompressionMaxOffset-wide window after a SET keyword opens `(timescaledb...`. The
// window is small enough to skip optional column qualifiers before the reloption body but
// tight enough to avoid grabbing unrelated parentheses further along.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes setIndex (int) which is the index of the SET keyword.
//
// Returns bool which is true when the expected payload pattern is found within the
// window.
func compressionPayloadStartsAt(tokens []db_engine_postgres.Token, setIndex int) bool {
	for offset := 1; offset <= alterCompressionMaxOffset && setIndex+offset+2 < len(tokens); offset++ {
		if tokens[setIndex+offset].Kind() == db_engine_postgres.TokenLeftParen &&
			strings.EqualFold(tokens[setIndex+offset+1].Value(), "timescaledb") {
			return true
		}
	}
	return false
}
