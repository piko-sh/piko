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
	"cmp"
	"fmt"
	"runtime/debug"
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// engineClauseTTL is the EngineSpecific key for the table TTL clause.
	engineClauseTTL = "TTL"

	// engineClauseComment is the EngineSpecific key for the table COMMENT clause.
	engineClauseComment = "COMMENT"

	// engineClauseEngine is the EngineSpecific key for the ENGINE clause.
	engineClauseEngine = "ENGINE"

	// engineClauseOnCluster is the EngineSpecific key for the ON CLUSTER clause.
	engineClauseOnCluster = "ON_CLUSTER"

	// engineKeyMergeTreeVersionColumn captures the version-column parameter of
	// ReplacingMergeTree(version_column [, is_deleted_column]).
	engineKeyMergeTreeVersionColumn = "MERGETREE_VERSION_COLUMN"

	// engineKeyMergeTreeIsDeletedColumn captures the optional is_deleted column of
	// ReplacingMergeTree(version, is_deleted).
	engineKeyMergeTreeIsDeletedColumn = "MERGETREE_IS_DELETED_COLUMN"

	// engineKeyMergeTreeSignColumn captures the sign column of CollapsingMergeTree(sign) and
	// VersionedCollapsingMergeTree(sign, version).
	engineKeyMergeTreeSignColumn = "MERGETREE_SIGN_COLUMN"

	// engineKeyMergeTreeSummingColumns captures the optional summed-column list of
	// SummingMergeTree([(col, ...)]).
	engineKeyMergeTreeSummingColumns = "MERGETREE_SUMMING_COLUMNS"

	// engineKeyMergeTreeZooPath captures the Keeper/ZooKeeper path of
	// ReplicatedMergeTree(zoo_path, replica_name [, version_column]).
	engineKeyMergeTreeZooPath = "MERGETREE_ZOO_PATH"

	// engineKeyMergeTreeReplicaName captures the replica name of
	// ReplicatedMergeTree(zoo_path, replica_name [, version_column]).
	engineKeyMergeTreeReplicaName = "MERGETREE_REPLICA_NAME"
)

// parseCreateTable handles a ClickHouse CREATE TABLE statement and its engine clauses.
//
// The grammar covers `CREATE [OR REPLACE] [TEMPORARY] TABLE [IF NOT EXISTS]
// [database.]table (column_list) ENGINE = ... PARTITION BY ... ORDER BY ... SETTINGS ...
// TTL ...`. The engine declaration plus its modifier clauses are captured verbatim into
// EngineSpecific so downstream consumers (PREWHERE eligibility, FINAL semantics) can
// inspect them without re-parsing the SQL.
//
// Returns *querier_dto.CatalogueMutation with Kind=MutationCreateTable and the parsed
// column metadata.
// Returns error on malformed input.
func (p *parser) parseCreateTable() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("CREATE")
	p.skipCreatePrefixesInParser()
	if !p.matchKeyword("TABLE") {
		return nil, fmt.Errorf("expected TABLE keyword at position %d", p.current().position)
	}
	p.matchIfNotExists()

	database, name, nameError := p.parseDatabaseQualifiedName()
	if nameError != nil {
		return nil, fmt.Errorf("parsing table name: %w", nameError)
	}

	mutation := &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationCreateTable,
		SchemaName:     database,
		TableName:      name,
		EngineSpecific: map[string]string{},
	}

	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific[engineClauseOnCluster] = cluster
	}

	if p.current().kind != tokenLeftParen {
		if p.matchKeyword("AS") {
			if err := p.populateCTASColumns(mutation); err != nil {
				return nil, err
			}
			return mutation, nil
		}
		return nil, fmt.Errorf("expected '(' or AS at position %d", p.current().position)
	}

	columns, primaryKey, parseErr := p.parseCreateTableBody()
	if parseErr != nil {
		return nil, parseErr
	}
	mutation.Columns = columns
	mutation.PrimaryKey = primaryKey

	if err := p.parseTableEngineClauses(mutation); err != nil {
		return nil, err
	}

	return mutation, nil
}

// populateCTASColumns lifts a CREATE TABLE AS body into the mutation's column list.
//
// The projection of a `CREATE TABLE x AS SELECT ...` body becomes the column list. When
// the body instead names a source table (the clone form `CREATE TABLE x AS source`), the
// source table identifier is captured under the EngineSpecific `CTAS_SOURCE_TABLE` (and
// `CTAS_SOURCE_SCHEMA` when qualified) so downstream consumers can resolve the columns
// from the catalogue lookup. The remainder of the statement (engine clauses, etc.) is
// consumed unchanged so the CTAS tail still completes the parse. Failure to detect a
// SELECT body is not treated as an error because the source-table form is also valid.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the lifted columns.
//
// Returns error only when the SELECT body analyser surfaces one.
func (p *parser) populateCTASColumns(mutation *querier_dto.CatalogueMutation) error {
	if p.isKeyword("SELECT") || p.isKeyword("WITH") {
		return p.populateCTASFromSelect(mutation)
	}
	if p.current().kind == tokenIdentifier {
		schema, name, err := p.parseDatabaseQualifiedName()
		if err != nil {
			return err
		}
		mutation.EngineSpecific["CTAS_SOURCE_TABLE"] = name
		if schema != "" {
			mutation.EngineSpecific["CTAS_SOURCE_SCHEMA"] = schema
		}
	}
	if err := p.parseTableEngineClauses(mutation); err != nil {
		return err
	}
	p.consumeRemainder()
	return nil
}

// populateCTASFromSelect re-tokenises the remaining input through a fresh parser and runs
// analyseSelect on it. The analyser's output columns are lifted into the mutation's
// column list as nullable catalogue columns; subsequent type resolution refines them
// through the function / expression resolvers.
//
// On analyser panic the recovered value becomes the returned error so a malformed CTAS
// body cannot crash the apply loop; the stack trace is logged engine-side via log.Warn
// rather than embedded in the error, so user-facing surfaces never expose internal paths.
// The remainder of the source is consumed so the caller's engine-clause parse continues
// from a known position.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the analysed columns.
//
// Returns error when the SELECT analyser fails or the analyser body panics.
func (p *parser) populateCTASFromSelect(mutation *querier_dto.CatalogueMutation) (err error) {
	remainingTokens := p.tokens[p.position:]
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Warn("clickhouse: panic while analysing CTAS body",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			err = fmt.Errorf("clickhouse: panic while analysing CTAS body: %v", recovered)
		}
	}()
	nested := newParser(remainingTokens)
	nested.analysisDepth = p.analysisDepth
	nested.maxParseDepth = p.maxParseDepth
	analysis, analyseErr := nested.analyseSelect()
	p.consumeRemainder()
	if analyseErr != nil || analysis == nil {
		return analyseErr
	}
	mutation.Columns = columnsFromCTASAnalysis(analysis)
	return nil
}

// columnsFromCTASAnalysis converts the analyser's output columns into catalogue columns.
//
// Each column's nullability defaults to true because CTAS produces virtual columns whose
// nullability cannot be determined without full type resolution against the source
// tables. The downstream catalogue builder may tighten the flag once it can look up the
// source table.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which holds the analysed output columns.
//
// Returns []querier_dto.Column which are the catalogue columns, each nullable by default.
func columnsFromCTASAnalysis(analysis *querier_dto.RawQueryAnalysis) []querier_dto.Column {
	columns := make([]querier_dto.Column, 0, len(analysis.OutputColumns))
	for _, output := range analysis.OutputColumns {
		name := cmp.Or(output.Name, output.ColumnName)
		columns = append(columns, querier_dto.Column{
			Name:     name,
			SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			Nullable: true,
		})
	}
	return columns
}

// skipCreatePrefixesInParser advances past OR REPLACE / TEMPORARY / TEMP modifiers if
// present. Symmetric to skipCreatePrefixes used by classifyStatement but operates on the
// live parser cursor.
func (p *parser) skipCreatePrefixesInParser() {
	for {
		if p.matchKeyword("OR") {
			if !p.matchKeyword("REPLACE") {
				return
			}
			continue
		}
		if p.matchKeyword("TEMPORARY") || p.matchKeyword("TEMP") {
			continue
		}
		return
	}
}

// matchIfNotExists optionally consumes `IF NOT EXISTS`.
func (p *parser) matchIfNotExists() {
	if !p.matchKeyword("IF") {
		return
	}
	p.matchKeyword("NOT")
	p.matchKeyword("EXISTS")
}

// matchIfExists optionally consumes `IF EXISTS`.
func (p *parser) matchIfExists() {
	if !p.matchKeyword("IF") {
		return
	}
	p.matchKeyword("EXISTS")
}

// matchOnCluster optionally consumes the trailing `ON CLUSTER <name>` clause that
// ClickHouse accepts on every replicated DDL statement.
//
// Returns string which is the cluster name, or empty when no clause was present.
func (p *parser) matchOnCluster() string {
	saved := p.position
	if !p.matchKeyword("ON") {
		return ""
	}
	if !p.matchKeyword("CLUSTER") {
		p.position = saved
		return ""
	}
	if p.current().kind == tokenIdentifier {
		name := p.current().value
		p.advance()
		return name
	}
	if p.current().kind == tokenString {
		name := p.current().value
		p.advance()
		return name
	}

	p.position = saved
	return ""
}

// parseCreateTableBody reads the parenthesised column list.
//
// ClickHouse column constraints differ from postgres: there is no PRIMARY KEY / UNIQUE /
// REFERENCES / CHECK at the column level; instead columns accept DEFAULT, MATERIALIZED,
// ALIAS, EPHEMERAL, CODEC, TTL, and COMMENT. PRIMARY KEY is declared as a table-level
// clause or implicitly via the table-engine ORDER BY.
//
// Returns []querier_dto.Column which are the parsed column definitions.
// Returns []string which are the table-level primary key columns, or nil when none.
// Returns error which is non-nil on malformed input.
func (p *parser) parseCreateTableBody() ([]querier_dto.Column, []string, error) {
	if p.current().kind != tokenLeftParen {
		return nil, nil, fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()

	var columns []querier_dto.Column
	var primaryKey []string

	for {
		if p.current().kind == tokenRightParen {
			p.advance()
			return columns, primaryKey, nil
		}

		entry, err := p.parseCreateTableEntry()
		if err != nil {
			return nil, nil, err
		}
		if entry.IsColumn {
			columns = append(columns, entry.Column)
		}
		if entry.IsPrimaryKey {
			primaryKey = entry.PrimaryKey
		}

		switch p.current().kind {
		case tokenComma:
			p.advance()
			continue
		case tokenRightParen:
			p.advance()
			return columns, primaryKey, nil
		default:
			return nil, nil, fmt.Errorf("expected ',' or ')' at position %d", p.current().position)
		}
	}
}

// createTableEntry carries the result of one CREATE TABLE body entry. Exactly one of
// Column or PrimaryKey is meaningful at a time; the IsColumn / IsPrimaryKey flags say
// which.
type createTableEntry struct {
	// PrimaryKey holds the table-level primary key columns when IsPrimaryKey is true.
	PrimaryKey []string

	// Column holds the parsed column definition when IsColumn is true.
	Column querier_dto.Column

	// IsColumn reports whether Column carries a parsed column definition.
	IsColumn bool

	// IsPrimaryKey reports whether PrimaryKey carries a table-level primary key.
	IsPrimaryKey bool
}

// parseCreateTableEntry handles one entry inside a CREATE TABLE body.
//
// An entry is a secondary declaration (INDEX / CONSTRAINT / PROJECTION), a table-level
// PRIMARY KEY, or a column definition. The returned entry's IsColumn / IsPrimaryKey flags
// tell the caller which slot is populated; secondary declarations leave both flags false
// and the caller skips them silently.
//
// Returns createTableEntry which is the captured entry, with IsColumn true when Column is
// populated and IsPrimaryKey true when PrimaryKey is populated.
// Returns error which is non-nil for malformed input.
func (p *parser) parseCreateTableEntry() (createTableEntry, error) {
	if p.isAnyKeyword("INDEX", "CONSTRAINT", "PROJECTION") {
		p.skipUntilCommaOrCloseParen()
		return createTableEntry{}, nil
	}
	parsedKeys, isPrimaryKey, keyErr := p.tryParsePrimaryKeyClause()
	if keyErr != nil {
		return createTableEntry{}, keyErr
	}
	if isPrimaryKey {
		return createTableEntry{PrimaryKey: parsedKeys, IsPrimaryKey: true}, nil
	}
	parsedColumn, columnErr := p.parseClickHouseColumn()
	if columnErr != nil {
		return createTableEntry{}, columnErr
	}
	return createTableEntry{Column: parsedColumn, IsColumn: true}, nil
}

// tryParsePrimaryKeyClause checks for a table-level PRIMARY KEY declaration.
//
// Returns []string which are the primary key columns, or nil when the cursor is not on a
// PRIMARY KEY clause so the caller can fall through to column parsing.
// Returns bool which is true when a PRIMARY KEY clause was consumed.
// Returns error which is non-nil on malformed input within the clause.
func (p *parser) tryParsePrimaryKeyClause() ([]string, bool, error) {
	if !p.isKeyword("PRIMARY") {
		return nil, false, nil
	}
	saved := p.position
	p.advance()
	if !p.matchKeyword("KEY") {
		p.position = saved
		return nil, false, nil
	}
	if p.current().kind != tokenLeftParen {
		columnName, err := p.parseIdentifierOrKeyword()
		if err != nil {
			return nil, false, err
		}
		return []string{columnName}, true, nil
	}
	p.advance()
	var columns []string
	for {
		columnName, err := p.parseIdentifierOrKeyword()
		if err != nil {
			return nil, false, err
		}
		columns = append(columns, columnName)
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		if p.current().kind == tokenRightParen {
			p.advance()
			return columns, true, nil
		}
		return nil, false, fmt.Errorf("expected ',' or ')' in PRIMARY KEY at position %d", p.current().position)
	}
}

// parseClickHouseColumn reads one column definition.
//
// The column shape is "<name> <type> [NULL | NOT NULL] [DEFAULT <expr> | MATERIALIZED
// <expr> | ALIAS <expr> | EPHEMERAL [<expr>]] [CODEC(...)] [TTL <expr>] [COMMENT
// <string>]". The full grammar is accepted; the parser captures name plus type plus
// nullability plus default-presence flags. Modifier clauses past the type are parsed only
// for forward-progress and are otherwise discarded because they do not affect codegen.
//
// ClickHouse permits the postgres-style `col Int32 NOT NULL` and `col Int32 NULL`
// suffixes alongside the wrapper-form `Nullable(Int32)`. NULL on a non-Nullable inner
// type promotes the column to nullable; NOT NULL is a no-op when the inner type is
// already non-nullable (and rejects an outer Nullable wrapper).
//
// Returns querier_dto.Column which is the parsed column definition.
// Returns error when the column name or type cannot be parsed.
func (p *parser) parseClickHouseColumn() (querier_dto.Column, error) {
	columnName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return querier_dto.Column{}, fmt.Errorf("parsing column name: %w", err)
	}
	typeName, err := p.readClickHouseTypeText()
	if err != nil {
		return querier_dto.Column{}, fmt.Errorf("reading column %q type: %w", columnName, err)
	}
	typeResult, parseErr := parseClickHouseType(typeName)
	if parseErr != nil {
		return querier_dto.Column{}, fmt.Errorf("parsing column %q type %q: %w", columnName, typeName, parseErr)
	}

	column := querier_dto.Column{
		Name:     columnName,
		SQLType:  typeResult.SQLType,
		Nullable: typeResult.Nullable,
	}

	p.applyColumnNullSuffix(&column)

	for p.applyColumnModifier(&column) {
	}
	return column, nil
}

// applyColumnNullSuffix consumes an optional `NULL` / `NOT NULL` suffix after a type.
//
// NULL promotes the column to nullable; NOT NULL is a no-op for already non-nullable
// inner types because Nullable(T) is the canonical nullable form in ClickHouse. The
// cursor is left unchanged when no suffix is present.
//
// Takes column (*querier_dto.Column) whose nullability is updated in place.
func (p *parser) applyColumnNullSuffix(column *querier_dto.Column) {
	if p.matchKeyword("NOT") {
		if !p.matchKeyword("NULL") {
			p.position--
		}
		return
	}
	if p.matchKeyword("NULL") {
		column.Nullable = true
		if !column.SQLType.Nullable {
			column.SQLType.Nullable = true
		}
	}
}

// applyColumnModifier matches one column-modifier clause (DEFAULT, MATERIALIZED, ALIAS,
// EPHEMERAL, CODEC, TTL, COMMENT) and updates the column accordingly.
//
// Takes column (*querier_dto.Column) which is updated in place from the matched modifier.
//
// Returns bool which is false when the cursor is no longer on a recognised modifier,
// signalling the caller to stop iterating.
func (p *parser) applyColumnModifier(column *querier_dto.Column) bool {
	switch {
	case p.matchKeyword("DEFAULT"):
		column.HasDefault = true
		p.skipColumnModifierExpression()
	case p.matchKeyword("MATERIALIZED"):
		column.IsGenerated = true
		column.GeneratedKind = querier_dto.GeneratedKindStored
		p.skipColumnModifierExpression()
	case p.matchKeyword("ALIAS"):
		column.IsGenerated = true
		column.GeneratedKind = querier_dto.GeneratedKindVirtual
		p.skipColumnModifierExpression()
	case p.matchKeyword("EPHEMERAL"):
		column.HasDefault = true
		if p.current().kind != tokenComma && p.current().kind != tokenRightParen {
			p.skipColumnModifierExpression()
		}
	case p.matchKeyword("CODEC"):
		if p.current().kind == tokenLeftParen {
			_ = p.skipParenthesised()
		}
	case p.matchKeyword("TTL"):
		p.skipColumnModifierExpression()
	case p.matchKeyword("COMMENT"):
		if p.current().kind == tokenString {
			column.Comment = p.current().value
			p.advance()
		}
	default:
		return false
	}
	return true
}

// readClickHouseTypeText collects the raw text of a column type, including nested wrapper
// parens. The text is then handed to parseClickHouseType for structured parsing.
//
// The body is split into readNestedTypeBody which handles the inner paren-balanced scan;
// readClickHouseTypeText itself just loops over chained parenthesised groups so that
// types like `Decimal(P,S)(modifier)` are captured intact.
//
// Returns the raw type text and advances the cursor past the type.
func (p *parser) readClickHouseTypeText() (string, error) {
	var builder strings.Builder
	first := p.current()
	if first.kind != tokenIdentifier {
		return "", fmt.Errorf("expected type identifier at position %d", first.position)
	}
	builder.WriteString(first.value)
	p.advance()

	for p.current().kind == tokenLeftParen {
		builder.WriteByte('(')
		p.advance()
		p.readNestedTypeBody(&builder)
	}

	return builder.String(), nil
}

// readNestedTypeBody scans a balanced parenthesised body, writing each token's text.
//
// The cursor is left positioned past the closing `)`. Used by readClickHouseTypeText to
// collect everything between the outermost parens of a type modifier.
//
// Takes builder (*strings.Builder) which is the accumulator for the type's text.
func (p *parser) readNestedTypeBody(builder *strings.Builder) {
	depth := 1

	previous := tokenLeftParen
	for depth > 0 && !p.atEnd() {
		tok := p.current()
		switch tok.kind {
		case tokenLeftParen:
			depth++
			builder.WriteByte('(')
		case tokenRightParen:
			depth--
			builder.WriteByte(')')
			if depth == 0 {
				p.advance()
				return
			}
		case tokenComma:
			builder.WriteString(", ")
		case tokenString:
			if isValueLikeToken(previous) {
				builder.WriteByte(' ')
			}
			builder.WriteByte('\'')
			builder.WriteString(escapeClickHouseStringBody(tok.value))
			builder.WriteByte('\'')
		case tokenOperator, tokenNumber, tokenIdentifier:
			if tok.kind != tokenOperator && isValueLikeToken(previous) {
				builder.WriteByte(' ')
			}
			builder.WriteString(tok.value)
		case tokenDot:
			builder.WriteByte('.')
		default:
		}
		previous = tok.kind
		p.advance()
	}
}

// isValueLikeToken reports whether kind is a value-bearing token (identifier, number, or
// string literal) that must be separated from an adjacent value-bearing token by a space
// when re-emitting a type body. Operators, dots, commas, and parentheses are not
// value-like, so adjacency with them never forces a space.
//
// Takes kind (tokenKind) which is the token kind to classify.
//
// Returns bool which is true for identifier/number/string tokens.
func isValueLikeToken(kind tokenKind) bool {
	return kind == tokenIdentifier || kind == tokenNumber || kind == tokenString
}

// skipColumnModifierExpression advances past a column-modifier expression (DEFAULT, TTL).
//
// The expression is parsed only for forward-progress: the parser consumes until it hits a
// top-level comma or close paren. Nested parens are honoured.
func (p *parser) skipColumnModifierExpression() {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && (tok.kind == tokenComma || tok.kind == tokenRightParen) {
			return
		}
		if p.isAnyKeyword("DEFAULT", "MATERIALIZED", "ALIAS", "EPHEMERAL", "CODEC", "TTL", "COMMENT") && depth == 0 {
			return
		}
		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		default:
		}
		p.advance()
	}
}

// skipUntilCommaOrCloseParen advances past tokens until reaching a top-level comma or
// close paren. Used to skip INDEX / CONSTRAINT / PROJECTION declarations inside a CREATE
// TABLE body that we do not track in the catalogue.
func (p *parser) skipUntilCommaOrCloseParen() {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && (tok.kind == tokenComma || tok.kind == tokenRightParen) {
			return
		}
		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		default:
		}
		p.advance()
	}
}

// parseTableEngineClauses captures the engine-tail clauses into EngineSpecific.
//
// The clauses are `ENGINE = MergeTree(...) PARTITION BY ... ORDER BY ... SAMPLE BY ...
// PRIMARY KEY ... TTL ... SETTINGS ...`. Each clause is stored verbatim under its keyword
// so downstream code can inspect what was declared without re-parsing.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
//
// Returns error when a clause keyword is followed by malformed content (for example
// SETTINGS without a value list).
func (p *parser) parseTableEngineClauses(mutation *querier_dto.CatalogueMutation) error {
	for !p.atEnd() {
		handled, err := p.parseSingleEngineClause(mutation)
		if err != nil {
			return err
		}
		if !handled {
			return nil
		}
	}
	return nil
}

// parseSingleEngineClause matches one engine-tail clause and writes it into the
// mutation's EngineSpecific map. Reports handled=false when the cursor is no longer on a
// recognised clause, which signals the outer loop to stop.
//
// Takes mutation (*querier_dto.CatalogueMutation), the in-flight mutation to populate.
//
// Returns handled (bool), true when one clause was consumed.
// Returns error when the matched clause is malformed.
func (p *parser) parseSingleEngineClause(mutation *querier_dto.CatalogueMutation) (handled bool, err error) {
	switch {
	case p.matchKeyword(engineClauseEngine):
		return p.captureClauseInto(mutation, engineClauseEngine)
	case p.isKeyword("PARTITION"):
		return p.captureByClauseInto(mutation, "PARTITION", "PARTITION_BY")
	case p.isKeyword("ORDER"):
		if captureErr := p.captureOrderByClause(mutation); captureErr != nil {
			return false, captureErr
		}
		return true, nil
	case p.isKeyword("PRIMARY"):
		return p.capturePrimaryKeyClause(mutation)
	case p.isKeyword("SAMPLE"):
		return p.captureByClauseInto(mutation, "SAMPLE", "SAMPLE_BY")
	case p.matchKeyword("TTL"):
		mutation.EngineSpecific[engineClauseTTL] = p.captureExpressionUntilClauseKeyword()
		return true, nil
	case p.matchKeyword("SETTINGS"):
		mutation.EngineSpecific["SETTINGS"] = p.captureExpressionUntilClauseKeyword()
		return true, nil
	case p.matchKeyword("COMMENT"):
		if p.current().kind == tokenString {
			mutation.EngineSpecific[engineClauseComment] = p.current().value
			p.advance()
		}
		return true, nil
	case p.matchKeyword("AS"):

		p.consumeRemainder()
		return false, nil
	default:
		return false, nil
	}
}

// captureClauseInto reads the value of an `<keyword> = <expression>` clause and stores it
// under the given EngineSpecific key.
//
// Takes mutation (*querier_dto.CatalogueMutation), the mutation to populate.
// Takes engineKey (string), the EngineSpecific map key.
//
// Returns handled (bool), true on success.
// Returns error which is non-nil if the clause cannot be captured.
func (p *parser) captureClauseInto(mutation *querier_dto.CatalogueMutation, engineKey string) (bool, error) {
	value := p.captureClauseValue()
	mutation.EngineSpecific[engineKey] = value
	if engineKey == engineClauseEngine {
		extractMergeTreeFamilyParams(mutation, value)
	}
	return true, nil
}

// extractMergeTreeFamilyParams lifts MergeTree-family positional parameters into keys.
//
// When the engine is one of the MergeTree-family forms with positional parameters, each
// parameter is lifted into a dedicated EngineSpecific key. The recognised forms are
// ReplacingMergeTree(ver_col [, is_deleted_col]), CollapsingMergeTree(sign),
// VersionedCollapsingMergeTree(sign, version), SummingMergeTree([(col, col, ...)]), and
// ReplicatedMergeTree(zoo_path, replica_name [, version_column]). The function tolerates
// leading `=` whitespace and any case variant of the engine name. Engines outside this
// set are left alone; the caller's catch-all ENGINE key still carries the full text.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
// Takes engineText (string) which is the captured engine clause text.
func extractMergeTreeFamilyParams(mutation *querier_dto.CatalogueMutation, engineText string) {
	name, args := splitEngineNameAndArgs(engineText)
	if handler, found := mergeTreeFamilyHandlers[strings.ToLower(name)]; found {
		handler(mutation, name, args)
	}
}

var (
	// mergeTreeFamilyHandlers dispatches each known MergeTree-family engine name to its
	// argument-extraction function. Engines outside the table (Log, Memory, plain MergeTree,
	// JOIN/SET) are no-ops; the catch-all ENGINE key populated by the caller still records
	// the full text for those engines.
	mergeTreeFamilyHandlers = map[string]func(*querier_dto.CatalogueMutation, string, []string){
		"replacingmergetree":           applyReplacingMergeTreeArgs,
		"collapsingmergetree":          applyCollapsingMergeTreeArgs,
		"versionedcollapsingmergetree": applyVersionedCollapsingMergeTreeArgs,
		"summingmergetree":             applySummingMergeTreeArgs,
		"replicatedmergetree":          applyReplicatedMergeTreeArgs,
		"replicatedreplacingmergetree": applyReplicatedReplacingMergeTreeArgs,
	}
)

// applyReplacingMergeTreeArgs populates the ReplacingMergeTree-specific EngineSpecific
// keys from the parsed argument list. The first argument is the version column; the
// optional second argument is the is_deleted column.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
// Takes args ([]string) which are the parsed engine arguments in order.
func applyReplacingMergeTreeArgs(mutation *querier_dto.CatalogueMutation, _ string, args []string) {
	if len(args) >= 1 {
		mutation.EngineSpecific[engineKeyMergeTreeVersionColumn] = args[0]
	}
	if len(args) >= 2 {
		mutation.EngineSpecific[engineKeyMergeTreeIsDeletedColumn] = args[1]
	}
}

// applyCollapsingMergeTreeArgs populates the sign column for the CollapsingMergeTree
// engine variant.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
// Takes args ([]string) which are the parsed engine arguments in order.
func applyCollapsingMergeTreeArgs(mutation *querier_dto.CatalogueMutation, _ string, args []string) {
	if len(args) >= 1 {
		mutation.EngineSpecific[engineKeyMergeTreeSignColumn] = args[0]
	}
}

// applyVersionedCollapsingMergeTreeArgs captures both the sign and version columns of
// VersionedCollapsingMergeTree.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
// Takes args ([]string) which are the parsed engine arguments in order.
func applyVersionedCollapsingMergeTreeArgs(mutation *querier_dto.CatalogueMutation, _ string, args []string) {
	if len(args) >= 1 {
		mutation.EngineSpecific[engineKeyMergeTreeSignColumn] = args[0]
	}
	if len(args) >= 2 {
		mutation.EngineSpecific[engineKeyMergeTreeVersionColumn] = args[1]
	}
}

// applySummingMergeTreeArgs joins the SummingMergeTree column list into a single
// comma-separated string for the EngineSpecific key.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
// Takes args ([]string) which are the parsed engine arguments in order.
func applySummingMergeTreeArgs(mutation *querier_dto.CatalogueMutation, _ string, args []string) {
	if len(args) > 0 {
		mutation.EngineSpecific[engineKeyMergeTreeSummingColumns] = strings.Join(args, ", ")
	}
}

// applyReplicatedMergeTreeArgs captures the Keeper / ZooKeeper path plus replica name of
// ReplicatedMergeTree.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
// Takes args ([]string) which are the parsed engine arguments in order.
func applyReplicatedMergeTreeArgs(mutation *querier_dto.CatalogueMutation, _ string, args []string) {
	if len(args) >= 1 {
		mutation.EngineSpecific[engineKeyMergeTreeZooPath] = args[0]
	}
	if len(args) >= 2 {
		mutation.EngineSpecific[engineKeyMergeTreeReplicaName] = args[1]
	}
}

// applyReplicatedReplacingMergeTreeArgs captures the Keeper / ZooKeeper path plus replica
// name and the optional version column of ReplicatedReplacingMergeTree.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
// Takes args ([]string) which are the parsed engine arguments in order.
func applyReplicatedReplacingMergeTreeArgs(mutation *querier_dto.CatalogueMutation, _ string, args []string) {
	applyReplicatedMergeTreeArgs(mutation, "", args)
	const replicatedReplacingVersionIndex = 2
	if len(args) >= replicatedReplacingVersionIndex+1 {
		mutation.EngineSpecific[engineKeyMergeTreeVersionColumn] = args[replicatedReplacingVersionIndex]
	}
}

// splitEngineNameAndArgs decomposes an engine clause text into its name and arguments.
//
// A text such as `ReplacingMergeTree(version, is_deleted)` yields the name plus the
// trimmed positional argument list. The argument splitter is paren-aware so nested
// expressions (`(col, func(a, b))`) do not split on internal commas.
//
// Takes engineText (string) which is the captured engine clause text.
//
// Returns string which is the bare engine name.
// Returns []string which are the positional arguments, or nil when the engine has no
// parameter parens (for example `Log`).
func splitEngineNameAndArgs(engineText string) (string, []string) {
	trimmed := strings.TrimSpace(engineText)
	openIndex := indexOutsideQuotes(trimmed, '(')
	if openIndex < 0 {
		return trimmed, nil
	}
	name := strings.TrimSpace(trimmed[:openIndex])
	closeIndex := lastIndexOutsideQuotes(trimmed, ')')
	if closeIndex <= openIndex {
		return name, nil
	}
	body := trimmed[openIndex+1 : closeIndex]
	args := splitTopLevelArguments(body)
	return name, args
}

// indexOutsideQuotes returns the index of the first occurrence of target that lies
// outside a single-quoted string literal, or -1 when no such occurrence exists. Backslash
// escapes inside a literal are respected so an escaped quote does not prematurely close
// the string.
//
// Takes text (string) which is the text to scan.
// Takes target (byte) which is the character to locate.
//
// Returns int which is the byte index of the first unquoted target, or -1.
func indexOutsideQuotes(text string, target byte) int {
	inString := false
	for index := 0; index < len(text); index++ {
		character := text[index]
		if inString {
			switch character {
			case '\\':
				index++
			case '\'':
				inString = false
			}
			continue
		}
		if character == '\'' {
			inString = true
			continue
		}
		if character == target {
			return index
		}
	}
	return -1
}

// lastIndexOutsideQuotes returns the index of the last occurrence of target that lies
// outside a single-quoted string literal, or -1 when no such occurrence exists. Backslash
// escapes inside a literal are respected so an escaped quote does not prematurely close
// the string.
//
// Takes text (string) which is the text to scan.
// Takes target (byte) which is the character to locate.
//
// Returns int which is the byte index of the last unquoted target, or -1.
func lastIndexOutsideQuotes(text string, target byte) int {
	inString := false
	found := -1
	for index := 0; index < len(text); index++ {
		character := text[index]
		if inString {
			switch character {
			case '\\':
				index++
			case '\'':
				inString = false
			}
			continue
		}
		if character == '\'' {
			inString = true
			continue
		}
		if character == target {
			found = index
		}
	}
	return found
}

// splitTopLevelArguments splits a comma-separated argument list at the top level.
//
// Nested parens and single-quoted string literals are honoured. Each returned argument is
// whitespace-trimmed; empty entries are dropped. A comma or paren inside a single-quoted
// literal (with backslash escapes respected) is part of the literal rather than a
// delimiter.
//
// Takes body (string) which is the comma-separated argument text.
//
// Returns []string which are the trimmed top-level arguments.
func splitTopLevelArguments(body string) []string {
	var args []string
	var current strings.Builder
	depth := 0
	for index := 0; index < len(body); index++ {
		character := body[index]
		switch {
		case character == '\'':
			index = writeQuotedLiteral(&current, body, index)
		case character == '(':
			depth++
			current.WriteByte(character)
		case character == ')':
			depth--
			current.WriteByte(character)
		case character == ',' && depth == 0:
			if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
				args = append(args, trimmed)
			}
			current.Reset()
		default:
			current.WriteByte(character)
		}
	}
	if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
		args = append(args, trimmed)
	}
	return args
}

// writeQuotedLiteral copies the single-quoted literal that begins at the opening quote at
// start into builder, honouring backslash escapes, and returns the index of its closing
// quote so the caller's loop increment advances past the whole literal. An unterminated
// literal consumes to the end of input.
//
// Takes builder (*strings.Builder) which receives the copied literal bytes.
// Takes body (string) which is the text being scanned.
// Takes start (int) which is the index of the opening single quote.
//
// Returns int which is the index of the closing quote, or the final index on no close.
func writeQuotedLiteral(builder *strings.Builder, body string, start int) int {
	builder.WriteByte(body[start])
	for index := start + 1; index < len(body); index++ {
		character := body[index]
		builder.WriteByte(character)
		switch character {
		case '\\':
			if index+1 < len(body) {
				index++
				builder.WriteByte(body[index])
			}
		case '\'':
			return index
		}
	}
	return len(body) - 1
}

// captureByClauseInto handles the family of `<keyword> BY <expression>` clauses
// (PARTITION BY, SAMPLE BY). It advances past the leading keyword, requires BY, then
// captures the expression body into the supplied EngineSpecific key.
//
// Takes mutation (*querier_dto.CatalogueMutation), the mutation to populate.
// Takes keywordName (string), the leading clause keyword name (used for the error
// message).
// Takes engineKey (string), the EngineSpecific map key.
//
// Returns handled (bool), true when the clause was consumed.
// Returns error when BY did not follow the leading keyword.
func (p *parser) captureByClauseInto(mutation *querier_dto.CatalogueMutation, keywordName string, engineKey string) (bool, error) {
	p.advance()
	if !p.matchKeyword("BY") {
		return false, fmt.Errorf("expected BY after %s at position %d", keywordName, p.current().position)
	}
	mutation.EngineSpecific[engineKey] = p.captureExpressionUntilClauseKeyword()
	return true, nil
}

// captureOrderByClause is a specialisation of captureByClauseInto for ORDER BY, which
// additionally infers the primary key from the order expression when no explicit PRIMARY
// KEY clause has been seen.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific and PrimaryKey are
// populated.
//
// Returns error when BY did not follow ORDER.
func (p *parser) captureOrderByClause(mutation *querier_dto.CatalogueMutation) error {
	p.advance()
	if !p.matchKeyword("BY") {
		return fmt.Errorf("expected BY after ORDER at position %d", p.current().position)
	}
	value := p.captureExpressionUntilClauseKeyword()
	mutation.EngineSpecific["ORDER_BY"] = value
	if len(mutation.PrimaryKey) == 0 {
		mutation.PrimaryKey = inferPrimaryKeyFromOrderBy(value)
	}
	return nil
}

// capturePrimaryKeyClause matches the optional table-level PRIMARY KEY clause that
// follows ORDER BY in ClickHouse engine declarations. When PRIMARY is not followed by
// KEY, the parser rewinds and reports the loop should stop, because PRIMARY without KEY
// is not a valid clause keyword in this grammar position.
//
// Takes mutation (*querier_dto.CatalogueMutation) whose EngineSpecific map is populated.
//
// Returns bool which is true when a PRIMARY KEY clause was consumed.
// Returns error when the clause is malformed.
func (p *parser) capturePrimaryKeyClause(mutation *querier_dto.CatalogueMutation) (bool, error) {
	saved := p.position
	p.advance()
	if !p.matchKeyword("KEY") {
		p.position = saved
		return false, nil
	}
	mutation.EngineSpecific["PRIMARY_KEY"] = p.captureExpressionUntilClauseKeyword()
	return true, nil
}

// captureClauseValue captures the value of a clause like `ENGINE = MergeTree()`. Reads
// the `=`, then collects everything up to the next top-level clause keyword.
//
// Returns string which is the captured clause value text.
func (p *parser) captureClauseValue() string {
	if p.current().kind == tokenOperator && p.current().value == "=" {
		p.advance()
	}
	return p.captureExpressionUntilClauseKeyword()
}

// captureExpressionUntilClauseKeyword collects tokens into a flat textual representation
// until it reaches a top-level clause keyword (PARTITION, ORDER, PRIMARY, SAMPLE, TTL,
// SETTINGS, COMMENT, AS) or end-of-input. Nested parens are honoured.
//
// Returns string which is the captured expression text up to the clause boundary.
func (p *parser) captureExpressionUntilClauseKeyword() string {
	var builder strings.Builder
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tok.kind == tokenIdentifier && isClauseKeyword(tok.value) {
			return builder.String()
		}
		depth = writeClauseToken(&builder, tok, depth)
		p.advance()
	}
	return builder.String()
}

// isClauseKeyword reports whether the identifier text matches one of the engine-tail
// clause keywords that terminate captureExpressionUntilClauseKeyword. The comparison is
// case-insensitive.
//
// The refreshable materialised view tail keywords (POPULATE, EMPTY, DEFINER, SQL) also
// terminate the capture so MV declarations interleaved with engine clauses do not see
// their trailing modifiers eaten by the ORDER / TTL / SETTINGS body parser.
//
// Takes text (string) which is the identifier to test.
//
// Returns bool which is true when text is an engine-tail clause keyword.
func isClauseKeyword(text string) bool {
	switch strings.ToUpper(text) {
	case "PARTITION", "ORDER", "PRIMARY", "SAMPLE", engineClauseTTL, "SETTINGS", engineClauseComment, "AS", engineClauseEngine,
		"POPULATE", "EMPTY", "DEFINER", "SQL":
		return true
	default:
		return false
	}
}

// writeClauseToken serialises a single token into the textual representation used by the
// clause capture helpers. Returns the new paren-balance depth so the caller can track it
// across iterations.
//
// Takes builder (*strings.Builder), the target buffer.
// Takes tok (token), the token to render.
// Takes depth (int), the current paren depth before this token.
//
// Returns the new paren depth after accounting for this token.
func writeClauseToken(builder *strings.Builder, tok token, depth int) int {
	switch tok.kind {
	case tokenLeftParen:
		builder.WriteByte('(')
		return depth + 1
	case tokenRightParen:
		builder.WriteByte(')')
		return depth - 1
	case tokenComma:
		builder.WriteString(", ")
	case tokenIdentifier, tokenNumber:
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(tok.value)
	case tokenString:
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteByte('\'')
		builder.WriteString(escapeClickHouseStringBody(tok.value))
		builder.WriteByte('\'')
	case tokenOperator:
		builder.WriteString(tok.value)
	case tokenDot:
		builder.WriteByte('.')
	case tokenStar:
		builder.WriteByte('*')
	case tokenLeftBracket:
		builder.WriteByte('[')
	case tokenRightBracket:
		builder.WriteByte(']')
	case tokenCast:
		builder.WriteString("::")
	case tokenArrow:
		builder.WriteString(" -> ")
	default:
	}
	return depth
}

// consumeRemainder advances the cursor to end-of-input. Used for CTAS tails and other
// constructs the parser does not materially analyse.
func (p *parser) consumeRemainder() {
	for !p.atEnd() {
		p.advance()
	}
}

// consumeRemainderAsText is the text-capturing variant of consumeRemainder. It consumes
// every remaining token in the statement and returns the concatenated source text with
// identifier quoting preserved.
//
// LOSSY ROUND-TRIP: tokens are joined with single-space separators, so qualified-name
// forms such as `a.b` re-emit as `a . b` (the dot is its own token and is sandwiched
// between spaces). Callers that require lossless reconstruction must re-parse the
// original SQL rather than relying on the captured text. The lossy form is adequate for
// downstream consumers that store the statement verbatim for re-execution by a runtime
// that re-tokenises the stored text.
//
// Returns string which is the concatenated remaining source text, trimmed of surrounding
// whitespace.
func (p *parser) consumeRemainderAsText() string {
	var builder strings.Builder
	for !p.atEnd() {
		writeTokenAsSourceText(&builder, p.current())
		builder.WriteByte(' ')
		p.advance()
	}
	return strings.TrimSpace(builder.String())
}

// consumeUntilTopLevelComma advances the cursor to the next top-level comma or input end.
//
// The top-level comma (depth 0 with respect to parentheses) is left on the stream for the
// caller to handle. Used by sub-parsers that operate within a comma-separated list (for
// example multi-action ALTER TABLE statements) where consumeRemainder would inadvertently
// consume sibling clauses.
func (p *parser) consumeUntilTopLevelComma() {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tok.kind == tokenComma {
			return
		}
		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			if depth == 0 {
				return
			}
			depth--
		default:
		}
		p.advance()
	}
}

// consumeUntilTopLevelCommaAsText is the text variant of consumeUntilTopLevelComma.
//
// It consumes tokens up to the next top-level comma with identifier quoting preserved.
// The trailing comma is left on the stream so the outer multi-action loop continues. A
// labelled break exits the outer loop from inside the per-token switch. A bare `break`
// inside the switch would only fall through to the next statement after the switch (the
// writer and advance calls), which would consume the stopping token and drive the cursor
// past the boundary the caller is meant to see.
//
// Returns string which is the captured source text up to the comma, trimmed of any
// surrounding whitespace.
func (p *parser) consumeUntilTopLevelCommaAsText() string {
	var builder strings.Builder
	depth := 0
scan:
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tok.kind == tokenComma {
			break scan
		}
		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			if depth == 0 {
				break scan
			}
			depth--
		default:
		}
		writeTokenAsSourceText(&builder, tok)
		builder.WriteByte(' ')
		p.advance()
	}
	return strings.TrimSpace(builder.String())
}

// writeTokenAsSourceText writes the token's source representation to builder. String
// literals are quoted (with embedded single quotes doubled); identifiers that contain
// unsafe characters are wrapped in backticks; all other token kinds are emitted verbatim.
//
// Takes builder (*strings.Builder) which receives the rendered token text.
// Takes tok (token) which is the token to render.
func writeTokenAsSourceText(builder *strings.Builder, tok token) {
	switch tok.kind {
	case tokenString:
		builder.WriteByte('\'')
		builder.WriteString(escapeClickHouseStringBody(tok.value))
		builder.WriteByte('\'')
	case tokenIdentifier:
		if identifierNeedsQuoting(tok.value) {
			builder.WriteByte('`')
			builder.WriteString(strings.ReplaceAll(tok.value, "`", "``"))
			builder.WriteByte('`')
		} else {
			builder.WriteString(tok.value)
		}
	default:
		builder.WriteString(tok.value)
	}
}

// identifierNeedsQuoting reports whether an identifier cannot be re-emitted bare.
//
// An identifier needs quoting when it contains a character (whitespace, punctuation, or a
// leading digit) that would prevent it from being parsed as a bare identifier on its own,
// so re-emission must wrap it in backticks to preserve fidelity. The scan walks the value
// codepoint by codepoint so multi-byte runes (Unicode letters inside identifiers) are
// classified correctly rather than rejected byte-by-byte.
//
// Takes value (string) which is the identifier to test.
//
// Returns bool which is true when the identifier must be backtick-quoted.
func identifierNeedsQuoting(value string) bool {
	if value == "" {
		return false
	}
	leading, leadingWidth := utf8.DecodeRuneInString(value)
	if !isIdentStart(leading) {
		return true
	}
	for index := leadingWidth; index < len(value); {
		character, width := utf8.DecodeRuneInString(value[index:])
		if !isIdentPart(character) {
			return true
		}
		index += width
	}
	return false
}

// inferPrimaryKeyFromOrderBy extracts the column list from a ClickHouse `ORDER BY (col1,
// col2)` or `ORDER BY col1` clause. When no explicit PRIMARY KEY is declared, ClickHouse
// treats ORDER BY as the table's primary key for indexing purposes; we mirror that.
//
// The split is paren-aware via splitTopLevelArguments so multi-argument function calls in
// the key expression (for example `ORDER BY (cityHash64(a, b), c)`) are kept intact
// rather than torn at their internal commas.
//
// Takes orderBy (string) which is the captured ORDER BY expression text.
//
// Returns []string which are the inferred primary key columns.
func inferPrimaryKeyFromOrderBy(orderBy string) []string {
	body := strings.TrimSpace(orderBy)
	body = strings.TrimPrefix(body, "(")
	body = strings.TrimSuffix(body, ")")
	var keys []string
	for _, part := range splitTopLevelArguments(body) {
		fields := strings.Fields(part)
		if len(fields) > 0 {
			keys = append(keys, fields[0])
		}
	}
	return keys
}

// parseDropTable handles `DROP TABLE [IF EXISTS] [db.]table`.
//
// It produces a CatalogueMutation with MutationDropTable kind; the catalogue builder uses
// this to remove the table from the schema.
//
// Returns *querier_dto.CatalogueMutation with Kind=MutationDropTable.
// Returns error on malformed input.
func (p *parser) parseDropTable() (*querier_dto.CatalogueMutation, error) {
	return p.parseDropQualifiedObject("TABLE", querier_dto.MutationDropTable)
}

// parseDropView handles `DROP VIEW [IF EXISTS] [db.]view [ON CLUSTER c]`.
//
// Returns *querier_dto.CatalogueMutation with Kind=MutationDropView.
// Returns error on malformed input.
func (p *parser) parseDropView() (*querier_dto.CatalogueMutation, error) {
	return p.parseDropQualifiedObject("VIEW", querier_dto.MutationDropView)
}

// parseDropQualifiedObject is the shared body of parseDropTable and parseDropView.
// ClickHouse drops both TABLE and VIEW with the same grammar (`DROP {TABLE|VIEW} [IF
// EXISTS] [db.]name [ON CLUSTER c] [SYNC]`), so the only difference between the two
// callers is the object keyword and the resulting mutation kind.
//
// Takes objectKeyword (string), the keyword that follows DROP (TABLE / VIEW).
// Takes mutationKind (querier_dto.CatalogueMutationKind), the kind to stamp onto the
// produced mutation.
//
// Returns the parsed mutation, with EngineSpecific populated when an ON CLUSTER clause
// was present.
func (p *parser) parseDropQualifiedObject(
	objectKeyword string,
	mutationKind querier_dto.MutationKind,
) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("DROP")
	p.mustKeyword(objectKeyword)
	p.matchIfExists()
	database, name, err := p.parseDatabaseQualifiedName()
	if err != nil {
		return nil, err
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:       mutationKind,
		SchemaName: database,
		TableName:  name,
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific = map[string]string{engineClauseOnCluster: cluster}
	}
	p.matchKeyword("SYNC")
	return mutation, nil
}

// parseDropDatabase handles `DROP DATABASE [IF EXISTS] name [ON CLUSTER c]`. Modelled as
// a DropSchema mutation since ClickHouse databases map onto the Piko catalogue's schema
// concept.
//
// Returns *querier_dto.CatalogueMutation with Kind=MutationDropSchema.
// Returns error on malformed input.
func (p *parser) parseDropDatabase() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("DROP")
	p.mustKeyword("DATABASE")
	p.matchIfExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropSchema,
		SchemaName: name,
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific = map[string]string{engineClauseOnCluster: cluster}
	}
	p.matchKeyword("SYNC")
	return mutation, nil
}

// parseCreateDatabase handles `CREATE DATABASE [IF NOT EXISTS] name [ON CLUSTER c]
// [ENGINE = ...]`.
//
// Returns *querier_dto.CatalogueMutation with Kind=MutationCreateSchema.
// Returns error on malformed input.
func (p *parser) parseCreateDatabase() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("CREATE")
	p.skipCreatePrefixesInParser()
	p.mustKeyword("DATABASE")
	p.matchIfNotExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateSchema,
		SchemaName: name,
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific = map[string]string{engineClauseOnCluster: cluster}
	}

	if p.matchKeyword(engineClauseEngine) {
		if mutation.EngineSpecific == nil {
			mutation.EngineSpecific = map[string]string{}
		}
		mutation.EngineSpecific[engineClauseEngine] = p.captureClauseValue()
	}
	return mutation, nil
}
