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
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// keywordIndex marks the INDEX sub-target of an ALTER TABLE action.
	keywordIndex = "INDEX"

	// keywordProjection marks the PROJECTION sub-target of an ALTER TABLE action.
	keywordProjection = "PROJECTION"

	// keywordStatistics marks the STATISTICS sub-target of an ALTER TABLE action.
	keywordStatistics = "STATISTICS"

	// keywordConstraint marks the CONSTRAINT sub-target of an ALTER TABLE action.
	keywordConstraint = "CONSTRAINT"

	// keywordType marks the TYPE keyword that precedes an index or statistic type.
	keywordType = "TYPE"

	// keywordRefresh marks the REFRESH sub-target of a MODIFY action.
	keywordRefresh = "REFRESH"

	// keywordEvery marks the EVERY interval keyword of a refresh policy.
	keywordEvery = "EVERY"

	// keywordAfter marks the AFTER interval keyword of a refresh policy.
	keywordAfter = "AFTER"

	// keywordSQL marks the SQL keyword used by the SQL SECURITY clause.
	keywordSQL = "SQL"

	// keywordDefiner marks the DEFINER sub-target of a MODIFY action.
	keywordDefiner = "DEFINER"

	// keywordPartition marks the PARTITION keyword of a partition operation.
	keywordPartition = "PARTITION"

	// keywordPart marks the PART keyword of a part-level partition operation.
	keywordPart = "PART"

	// keywordFrom marks the FROM keyword of a partition move or fetch operation.
	keywordFrom = "FROM"

	// keywordTo marks the TO keyword of a partition move operation.
	keywordTo = "TO"

	// keywordAttach marks the ATTACH keyword of a partition operation.
	keywordAttach = "ATTACH"

	// keywordDetach marks the DETACH keyword of a control operation (DETACH TABLE / VIEW /
	// DICTIONARY / DATABASE).
	keywordDetach = "DETACH"

	// keywordGranularity marks the GRANULARITY tail clause of an ADD INDEX action.
	//
	// It appears in `ALTER TABLE ... ADD INDEX ... TYPE ... GRANULARITY n`.
	keywordGranularity = "GRANULARITY"

	// keywordName marks the NAME keyword used by the `WITH NAME 'x'` backup-name modifier on
	// FREEZE / UNFREEZE partition operations.
	keywordName = "NAME"

	// keywordDetached marks the DETACHED prefix on `DROP DETACHED PARTITION` / `DROP
	// DETACHED PART`.
	keywordDetached = "DETACHED"

	// keywordWith marks the WITH keyword used by the `WITH NAME 'x'` backup-name tail and
	// `WITH FILL` LIMIT modifier.
	keywordWith = "WITH"

	// engineKeyProjectionName is the EngineSpecific key holding a projection name.
	engineKeyProjectionName = "PROJECTION_NAME"

	// engineKeyProjectionSelect is the EngineSpecific key holding a projection body.
	engineKeyProjectionSelect = "PROJECTION_SELECT"

	// engineKeyIndexName is the EngineSpecific key holding a skipping-index name.
	engineKeyIndexName = "INDEX_NAME"

	// engineKeyIndexExpr is the EngineSpecific key holding a skipping-index expression.
	engineKeyIndexExpr = "INDEX_EXPR"

	// engineKeyIndexType is the EngineSpecific key holding a skipping-index type.
	engineKeyIndexType = "INDEX_TYPE"

	// engineKeyIndexGranularity is the EngineSpecific key holding an index granularity.
	engineKeyIndexGranularity = "INDEX_GRANULARITY"

	// engineKeyConstraintName is the EngineSpecific key holding a constraint name.
	engineKeyConstraintName = "CONSTRAINT_NAME"

	// engineKeyConstraintCheck is the EngineSpecific key holding a constraint check body.
	engineKeyConstraintCheck = "CONSTRAINT_CHECK"

	// engineKeyStatsColumns is the EngineSpecific key holding a statistic column list.
	engineKeyStatsColumns = "STATS_COLUMNS"

	// engineKeyStatsTypes is the EngineSpecific key holding a statistic type list.
	engineKeyStatsTypes = "STATS_TYPES"

	// engineKeyColumnRemove is the EngineSpecific key holding a removed column property.
	engineKeyColumnRemove = "COLUMN_REMOVE"

	// engineKeyColumnModifyCmt is the EngineSpecific key holding a column comment.
	engineKeyColumnModifyCmt = "COLUMN_MODIFY_COMMENT"

	// engineKeyColumnResetSet is the EngineSpecific key holding a reset setting name.
	engineKeyColumnResetSet = "COLUMN_RESET_SETTING"

	// engineKeyMaterializePart is the EngineSpecific key holding a materialise partition.
	engineKeyMaterializePart = "MATERIALIZE_PARTITION"

	// engineKeyModifyRefreshKind is the EngineSpecific key holding a MODIFY refresh kind.
	engineKeyModifyRefreshKind = "MODIFY_REFRESH_KIND"

	// engineKeyMVRefreshKind is the EngineSpecific key holding a view refresh kind.
	engineKeyMVRefreshKind = "MV_REFRESH_KIND"

	// engineKeyMVRefreshInterval is the EngineSpecific key holding a refresh interval.
	engineKeyMVRefreshInterval = "MV_REFRESH_INTERVAL"

	// engineKeyMVRefreshOffset is the EngineSpecific key holding a view refresh offset.
	engineKeyMVRefreshOffset = "MV_REFRESH_OFFSET"

	// engineKeyMVRefreshAfter is the EngineSpecific key holding a view AFTER interval.
	engineKeyMVRefreshAfter = "MV_REFRESH_AFTER"

	// engineKeyMVSQLSecurity is the EngineSpecific key holding a view SQL security mode.
	engineKeyMVSQLSecurity = "MV_SQL_SECURITY"

	// engineKeyMVDefiner is the EngineSpecific key holding a view definer.
	engineKeyMVDefiner = "MV_DEFINER"

	// engineKeyNewQuery is the EngineSpecific key holding a replacement view query.
	engineKeyNewQuery = "NEW_QUERY"

	// engineKeyColumnName is the EngineSpecific key holding a column name.
	engineKeyColumnName = "COLUMN_NAME"

	// engineKeyMaterializeTarget is the EngineSpecific key holding a materialise target.
	engineKeyMaterializeTarget = "MATERIALIZE_TARGET"

	// engineKeyStatementBody is the EngineSpecific key holding a raw statement body.
	engineKeyStatementBody = "STATEMENT_BODY"

	// statsListSeparator joins captured statistic column / type identifier lists when they
	// are flattened into a single EngineSpecific value.
	statsListSeparator = ","
)

// parseAlterAddProjection captures the projection name and body of an `ALTER TABLE x ADD
// PROJECTION [IF NOT EXISTS] name (...)` clause.
//
// The body is captured verbatim into EngineSpecific[PROJECTION_SELECT] because ClickHouse
// projection bodies are SELECT-shaped and are not analysed structurally.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the added projection.
// Returns error when the projection name or body cannot be parsed.
func (p *parser) parseAlterAddProjection(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfNotExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	body := ""
	if p.current().kind == tokenLeftParen {
		captured, captureErr := p.captureParenthesisedBodyAsText()
		if captureErr != nil {
			return nil, captureErr
		}
		body = captured
	}
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAddProjection,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyProjectionName:   name,
			engineKeyProjectionSelect: body,
		},
	}, nil
}

// parseAlterAddIndex captures the name, expression, type, and granularity of an `ALTER
// TABLE x ADD INDEX [IF NOT EXISTS] name expr TYPE indextype [GRANULARITY n]` clause.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the added skipping index.
// Returns error when the index name cannot be parsed.
func (p *parser) parseAlterAddIndex(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfNotExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	indexExpression := p.captureIndexExpressionText()
	indexType, indexGranularity := p.captureIndexTypeAndGranularity()
	specific := map[string]string{
		engineKeyIndexName: name,
		engineKeyIndexExpr: indexExpression,
	}
	if indexType != "" {
		specific[engineKeyIndexType] = indexType
	}
	if indexGranularity != "" {
		specific[engineKeyIndexGranularity] = indexGranularity
	}
	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationAlterTableAddSkippingIndex,
		SchemaName:     database,
		TableName:      table,
		EngineSpecific: specific,
	}, nil
}

// captureIndexExpressionText reads the expression that follows an index name, stopping at
// TYPE, GRANULARITY, a top-level comma, or the end of input.
//
// Returns string which is the trimmed index expression text.
func (p *parser) captureIndexExpressionText() string {
	var builder strings.Builder
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tok.kind == tokenIdentifier &&
			(strings.EqualFold(tok.value, keywordType) || strings.EqualFold(tok.value, keywordGranularity)) {
			break
		}
		if depth == 0 && tok.kind == tokenComma {
			break
		}
		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			if depth == 0 {
				return strings.TrimSpace(builder.String())
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

// captureIndexTypeAndGranularity reads the optional `TYPE name(...)` and `GRANULARITY n`
// tail of an ALTER TABLE ADD INDEX clause.
//
// Returns indexType (string) which is the captured index type, empty when absent.
// Returns indexGranularity (string) which is the captured granularity, empty when absent.
func (p *parser) captureIndexTypeAndGranularity() (indexType string, indexGranularity string) {
	for {
		switch {
		case p.matchKeyword(keywordType):
			indexType = p.captureIndexTypeBody()
		case p.matchKeyword(keywordGranularity):
			if p.current().kind == tokenNumber {
				indexGranularity = p.current().value
				p.advance()
			}
		default:
			return indexType, indexGranularity
		}
	}
}

// captureIndexTypeBody reads the `name(args)` form following the `TYPE` keyword in an ADD
// INDEX clause.
//
// Returns string which is the trimmed index type with its argument body.
func (p *parser) captureIndexTypeBody() string {
	var builder strings.Builder
	if p.current().kind != tokenIdentifier {
		return ""
	}
	builder.WriteString(p.current().value)
	p.advance()
	if p.current().kind != tokenLeftParen {
		return strings.TrimSpace(builder.String())
	}
	body, err := p.captureParenthesisedBodyAsText()
	if err != nil {
		return strings.TrimSpace(builder.String())
	}
	builder.WriteByte('(')
	builder.WriteString(body)
	builder.WriteByte(')')
	return strings.TrimSpace(builder.String())
}

// parseAlterAddConstraint captures the name and check body of an `ALTER TABLE x ADD
// CONSTRAINT [IF NOT EXISTS] name CHECK expr` clause.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the added constraint.
// Returns error when the constraint name cannot be parsed.
func (p *parser) parseAlterAddConstraint(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfNotExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	specific := map[string]string{engineKeyConstraintName: name}
	if p.matchKeyword("CHECK") {
		specific[engineKeyConstraintCheck] = p.consumeUntilTopLevelCommaAsText()
	}
	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationAlterTableAddConstraint,
		SchemaName:     database,
		TableName:      table,
		ConstraintName: name,
		EngineSpecific: specific,
	}, nil
}

// parseAlterAddStatistics captures the column list and statistic type list of an `ALTER
// TABLE x ADD STATISTICS [IF NOT EXISTS] col1[, col2, ...] TYPE type1[, type2, ...]`
// clause.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the added statistics.
// Returns error when the clause cannot be parsed.
func (p *parser) parseAlterAddStatistics(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfNotExists()
	columns, types := p.parseStatisticsColumnAndTypeLists()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAddStatistics,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyStatsColumns: strings.Join(columns, statsListSeparator),
			engineKeyStatsTypes:   strings.Join(types, statsListSeparator),
		},
	}, nil
}

// parseStatisticsColumnAndTypeLists reads the `col[, col, ...] TYPE type[, type, ...]`
// body of an ADD or MODIFY STATISTICS clause.
//
// Returns columns ([]string) which are the parsed statistic column names.
// Returns types ([]string) which are the parsed statistic type names, empty when no TYPE
// clause is present.
func (p *parser) parseStatisticsColumnAndTypeLists() (columns, types []string) {
	columns = p.parseCommaSeparatedIdentifierList()
	if p.matchKeyword(keywordType) {
		types = p.parseCommaSeparatedIdentifierList()
	}
	return columns, types
}

// parseCommaSeparatedIdentifierList reads a `name[, name, ...]` list where each entry is
// a bare identifier.
//
// Returns []string which are the parsed identifier names in order.
func (p *parser) parseCommaSeparatedIdentifierList() []string {
	var names []string
	for !p.atEnd() {
		if p.current().kind != tokenIdentifier {
			break
		}
		if strings.EqualFold(p.current().value, keywordType) {
			break
		}
		names = append(names, p.current().value)
		p.advance()
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		break
	}
	return names
}

// parseAlterDropProjection captures the name of a `DROP PROJECTION [IF EXISTS] name`
// clause.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the dropped projection.
// Returns error when the projection name cannot be parsed.
func (p *parser) parseAlterDropProjection(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationAlterTableDropProjection,
		SchemaName:     database,
		TableName:      table,
		EngineSpecific: map[string]string{engineKeyProjectionName: name},
	}, nil
}

// parseAlterDropIndex captures the name of a `DROP INDEX [IF EXISTS] name` clause.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the dropped skipping index.
// Returns error when the index name cannot be parsed.
func (p *parser) parseAlterDropIndex(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationAlterTableDropSkippingIndex,
		SchemaName:     database,
		TableName:      table,
		EngineSpecific: map[string]string{engineKeyIndexName: name},
	}, nil
}

// parseAlterDropConstraint captures the name of a `DROP CONSTRAINT [IF EXISTS] name`
// clause.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the dropped constraint.
// Returns error when the constraint name cannot be parsed.
func (p *parser) parseAlterDropConstraint(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationAlterTableDropConstraint,
		SchemaName:     database,
		TableName:      table,
		ConstraintName: name,
		EngineSpecific: map[string]string{engineKeyConstraintName: name},
	}, nil
}

// parseAlterDropStatistics captures the column list of a `DROP STATISTICS [IF EXISTS]
// col1[, col2, ...]` clause.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the dropped statistics.
// Returns error when the clause cannot be parsed.
func (p *parser) parseAlterDropStatistics(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfExists()
	columns := p.parseCommaSeparatedIdentifierList()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableDropStatistics,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyStatsColumns: strings.Join(columns, statsListSeparator),
		},
	}, nil
}

// parseAlterDropDetached recognises `DROP DETACHED PARTITION expr` and `DROP DETACHED
// PART expr`.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the dropped detached partition or part.
// Returns error when neither PART nor PARTITION follows DROP DETACHED.
func (p *parser) parseAlterDropDetached(database, table string) (*querier_dto.CatalogueMutation, error) {
	isPart := p.matchKeyword(keywordPart)
	if !isPart && !p.matchKeyword(keywordPartition) {
		return nil, fmt.Errorf("expected PART or PARTITION after DROP DETACHED at position %d", p.current().position)
	}
	mutation, err := p.parseAlterPartitionOperation(database, table, keywordDrop, isPart)
	if err != nil {
		return nil, err
	}
	if mutation != nil {
		mutation.EngineSpecific["PARTITION_DETACHED"] = "true"
	}
	return mutation, nil
}

// parseAlterModifyColumnAction handles `MODIFY COLUMN [IF EXISTS] name {type | REMOVE
// prop | MODIFY COMMENT 'text' | RESET SETTING name}` clauses.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the column modification.
// Returns error when the column name or sub-action cannot be parsed.
func (p *parser) parseAlterModifyColumnAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	p.matchIfExists()
	savedPosition := p.position
	columnName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	if mutation, sub, subErr := p.tryParseModifyColumnSubAction(database, table, columnName); sub {
		return mutation, subErr
	}
	p.position = savedPosition
	column, err := p.parseClickHouseColumn()
	if err != nil {
		return nil, err
	}
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAlterColumn,
		SchemaName: database,
		TableName:  table,
		ColumnName: column.Name,
		Columns:    []querier_dto.Column{column},
	}, nil
}

// tryParseModifyColumnSubAction handles the REMOVE, MODIFY COMMENT, and RESET SETTING
// sub-target forms of MODIFY COLUMN.
//
// Once a sub-action keyword has been consumed the handled flag is true, so the caller
// must not fall through to column-type parsing; in that case a non-nil error means the
// sub-action was malformed. A malformed sub-action surfaces as an error rather than a
// silent no-op, otherwise the whole ALTER statement is dropped without explanation.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
// Takes columnName (string) which is the name of the column being modified.
//
// Returns *CatalogueMutation which describes the sub-action, nil when none was matched.
// Returns bool which is true once a sub-action keyword has been consumed.
// Returns error when a consumed sub-action keyword is malformed.
func (p *parser) tryParseModifyColumnSubAction(database, table, columnName string) (*querier_dto.CatalogueMutation, bool, error) {
	switch {
	case p.matchKeyword("REMOVE"):
		property, parseErr := p.parseIdentifierOrKeyword()
		if parseErr != nil {
			return nil, true, fmt.Errorf("expected column property after MODIFY COLUMN %s REMOVE at position %d: %w", columnName, p.current().position, parseErr)
		}
		return p.buildModifyColumnMutation(database, table, columnName,
			engineKeyColumnRemove, strings.ToUpper(property)), true, nil
	case p.matchKeyword("MODIFY"):
		if !p.matchKeyword("COMMENT") {
			return nil, true, fmt.Errorf("expected COMMENT after MODIFY COLUMN %s MODIFY at position %d", columnName, p.current().position)
		}
		commentText := ""
		if p.current().kind == tokenString {
			commentText = p.current().value
			p.advance()
		}
		return p.buildModifyColumnMutation(database, table, columnName,
			engineKeyColumnModifyCmt, commentText), true, nil
	case p.matchKeyword("RESET"):
		if !p.matchKeyword("SETTING") {
			return nil, true, fmt.Errorf("expected SETTING after MODIFY COLUMN %s RESET at position %d", columnName, p.current().position)
		}
		settingName, settingErr := p.parseIdentifierOrKeyword()
		if settingErr != nil {
			return nil, true, fmt.Errorf("expected setting name after MODIFY COLUMN %s RESET SETTING at position %d: %w", columnName, p.current().position, settingErr)
		}
		return p.buildModifyColumnMutation(database, table, columnName,
			engineKeyColumnResetSet, settingName), true, nil
	}
	return nil, false, nil
}

// buildModifyColumnMutation creates a MutationAlterTableModifyColumn with the supplied
// key and value pair populated under EngineSpecific.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
// Takes columnName (string) which is the name of the column being modified.
// Takes key (string) which is the EngineSpecific key to populate.
// Takes value (string) which is the EngineSpecific value to store under key.
//
// Returns *querier_dto.CatalogueMutation describing the column modification.
func (*parser) buildModifyColumnMutation(database, table, columnName, key, value string) *querier_dto.CatalogueMutation {
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableModifyColumn,
		SchemaName: database,
		TableName:  table,
		ColumnName: columnName,
		EngineSpecific: map[string]string{
			key: value,
		},
	}
}

// parseAlterModifyQuery handles `ALTER TABLE v MODIFY QUERY <select>` for refreshable
// materialised views.
//
// Takes database (string) which is the schema name of the target view.
// Takes table (string) which is the name of the target view.
//
// Returns *CatalogueMutation which describes the replacement query.
// Returns error which is always nil for this clause.
func (p *parser) parseAlterModifyQuery(database, table string) (*querier_dto.CatalogueMutation, error) {
	newQuery := p.consumeRemainderAsText()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableModifyQuery,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyNewQuery: newQuery,
		},
	}, nil
}

// parseAlterModifyRefresh handles the refresh-policy sub-targets of `ALTER TABLE v
// MODIFY` for refreshable materialised views.
//
// Takes database (string) which is the schema name of the target view.
// Takes table (string) which is the name of the target view.
// Takes kind (string) which is the MODIFY sub-target keyword being handled.
//
// Returns *CatalogueMutation which describes the refresh-policy change.
// Returns error which is always nil for this clause.
func (p *parser) parseAlterModifyRefresh(database, table, kind string) (*querier_dto.CatalogueMutation, error) {
	specific := map[string]string{engineKeyModifyRefreshKind: kind}
	switch kind {
	case keywordRefresh:
		p.captureRefreshClauseInto(specific)
	case "SQL_SECURITY":
		if security, parseErr := p.parseIdentifierOrKeyword(); parseErr == nil {
			specific[engineKeyMVSQLSecurity] = strings.ToUpper(security)
		}
	case keywordDefiner:
		if p.current().kind == tokenOperator && p.current().value == "=" {
			p.advance()
		}
		if name, parseErr := p.parseIdentifierOrKeyword(); parseErr == nil {
			specific[engineKeyMVDefiner] = name
		}
	}
	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationAlterTableModifyRefresh,
		SchemaName:     database,
		TableName:      table,
		EngineSpecific: specific,
	}, nil
}

// captureRefreshClauseInto reads the `EVERY <interval> [OFFSET ...]` or `AFTER
// <interval>` body following a MODIFY REFRESH keyword.
//
// Takes specific (map[string]string) which is the EngineSpecific map the captured refresh
// kind, interval, offset, or AFTER interval is written into.
func (p *parser) captureRefreshClauseInto(specific map[string]string) {
	switch {
	case p.matchKeyword(keywordEvery):
		specific[engineKeyMVRefreshKind] = keywordEvery
		specific[engineKeyMVRefreshInterval] = p.captureIntervalText()
		if p.matchKeyword("OFFSET") {
			specific[engineKeyMVRefreshOffset] = p.captureIntervalText()
		}
	case p.matchKeyword(keywordAfter):
		specific[engineKeyMVRefreshKind] = keywordAfter
		specific[engineKeyMVRefreshAfter] = p.captureIntervalText()
	}
}

// parseAlterModifyStatistics handles `ALTER TABLE x MODIFY STATISTICS col1[, col2, ...]
// TYPE type1[, type2, ...]`.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the modified statistics.
// Returns error which is always nil for this clause.
func (p *parser) parseAlterModifyStatistics(database, table string) (*querier_dto.CatalogueMutation, error) {
	columns, types := p.parseStatisticsColumnAndTypeLists()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableModifyStatistics,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyStatsColumns: strings.Join(columns, statsListSeparator),
			engineKeyStatsTypes:   strings.Join(types, statsListSeparator),
		},
	}, nil
}

// parseAlterMaterializeNamed reads the named target plus the optional `IN PARTITION expr`
// clause for MATERIALIZE COLUMN, INDEX, or PROJECTION.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
// Takes kind (querier_dto.MutationKind) which is the mutation kind to emit.
// Takes nameKey (string) which is the EngineSpecific key the target name is stored under.
//
// Returns *CatalogueMutation which describes the materialise action.
// Returns error when the target name cannot be parsed.
func (p *parser) parseAlterMaterializeNamed(database, table string, kind querier_dto.MutationKind, nameKey string) (*querier_dto.CatalogueMutation, error) {
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:           kind,
		SchemaName:     database,
		TableName:      table,
		EngineSpecific: map[string]string{nameKey: name},
	}
	p.captureMaterializeInPartition(mutation)
	return mutation, nil
}

// captureMaterializeInPartition consumes an optional `IN PARTITION expr` clause.
//
// Takes mutation (*querier_dto.CatalogueMutation) which is the mutation whose
// EngineSpecific map receives the captured partition expression.
func (p *parser) captureMaterializeInPartition(mutation *querier_dto.CatalogueMutation) {
	if !p.matchKeyword("IN") {
		return
	}
	if !p.matchKeyword(keywordPartition) {
		return
	}
	mutation.EngineSpecific[engineKeyMaterializePart] = p.consumeUntilTopLevelCommaAsText()
}

// parseAlterMaterializeStatistics emits a mutation for `MATERIALIZE STATISTICS col_list
// [IN PARTITION expr]`.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the materialise statistics action.
// Returns error which is always nil for this clause.
func (p *parser) parseAlterMaterializeStatistics(database, table string) (*querier_dto.CatalogueMutation, error) {
	columns := p.parseCommaSeparatedIdentifierList()
	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableMaterializeStatistics,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyStatsColumns: strings.Join(columns, statsListSeparator),
		},
	}
	p.captureMaterializeInPartition(mutation)
	return mutation, nil
}

// parseAlterMaterializeTTL emits a runtime-only mutation for the `MATERIALIZE TTL [IN
// PARTITION expr]` form.
//
// Takes database (string) which is the schema name of the target table.
// Takes table (string) which is the name of the target table.
//
// Returns *CatalogueMutation which describes the materialise TTL action.
// Returns error which is always nil for this clause.
func (p *parser) parseAlterMaterializeTTL(database, table string) (*querier_dto.CatalogueMutation, error) {
	mutation := &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationAlterTableMaterializeColumn,
		SchemaName:     database,
		TableName:      table,
		EngineSpecific: map[string]string{engineKeyMaterializeTarget: "TTL"},
	}
	p.captureMaterializeInPartition(mutation)
	return mutation, nil
}
