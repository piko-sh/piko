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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// keywordColumn is the COLUMN keyword used in ALTER TABLE ... COLUMN.
	keywordColumn = "COLUMN"

	// keywordCreate is the CREATE keyword used to introduce a wide range of object kinds
	// (DICTIONARY, FUNCTION, ...).
	keywordCreate = "CREATE"

	// keywordDictionary is the DICTIONARY keyword used in CREATE DICTIONARY / DROP
	// DICTIONARY.
	keywordDictionary = "DICTIONARY"

	// engineKeyExchangeTarget is the EngineSpecific key used by EXCHANGE TABLES to record
	// the right-hand-side table identifier (the swap target). The primary mutation's
	// SchemaName/TableName pair carries the left-hand-side; consumers read both halves to
	// re-emit or analyse the swap.
	engineKeyExchangeTarget = "EXCHANGE_TARGET"

	// engineKeyAsyncBody is the EngineSpecific key marking an asynchronous mutation (ALTER
	// TABLE ... UPDATE / DELETE).
	//
	// It is written both on the DDL CatalogueMutation path (so the migration runner can
	// replay the body verbatim) and on the query-analysis RawQueryAnalysis path (so the
	// querier's async-exec diagnostic pass can recommend the asyncexec command). The value
	// is the captured SET/WHERE body text. The string literal must match asyncExecBodyKey in
	// internal/querier/querier_domain/async_exec_pass.go.
	engineKeyAsyncBody = "ASYNC_BODY"

	// dependsOnSeparator is the joiner used to flatten the DEPENDS ON table list captured
	// under EngineSpecific[MV_REFRESH_DEPENDS_ON].
	dependsOnSeparator = ", "

	// mvBooleanTrue is the canonical truthy string written into the MV_REFRESH_* /
	// MATERIALIZED / POPULATE EngineSpecific keys.
	mvBooleanTrue = "true"

	// keywordPopulate marks the POPULATE modifier in a CREATE MATERIALIZED VIEW body.
	keywordPopulate = "POPULATE"

	// keywordAs is the AS keyword that introduces a view body.
	keywordAs = "AS"

	// keywordEmpty marks the EMPTY modifier in a refreshable materialised view declaration.
	keywordEmpty = "EMPTY"

	// keywordDrop is the DROP keyword used by partition operations originating from `DROP
	// PARTITION` / `DROP DETACHED PARTITION`.
	keywordDrop = "DROP"

	// engineKeyMaterializedTarget is the EngineSpecific key under which `CREATE MATERIALIZED
	// VIEW ... TO target` captures the qualified destination table name.
	//
	// Downstream consumers (catalogue builder, codegen) read this key to wire the view's
	// projection through to the target table.
	engineKeyMaterializedTarget = "MATERIALIZED_TARGET"
)

// parseAlterTable handles every ClickHouse `ALTER TABLE [db.]table ...` variant.
//
// ClickHouse uses ALTER UPDATE / ALTER DELETE as mutation forms (no traditional DML
// UPDATE / DELETE); the parser dispatches on the action keyword after the table name and
// emits one mutation per action. Multi-action ALTER (separated by commas) emits the
// primary mutation plus AdditionalMutations entries.
//
// Returns *querier_dto.CatalogueMutation which is the primary ALTER mutation.
// Returns error when the table name or any action cannot be parsed.
func (p *parser) parseAlterTable() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("ALTER")
	p.mustKeyword("TABLE")
	p.matchIfExists()

	database, name, nameError := p.parseDatabaseQualifiedName()
	if nameError != nil {
		return nil, nameError
	}

	cluster := p.matchOnCluster()

	primary, err := p.parseSingleAlterAction(database, name)
	if err != nil {
		return nil, err
	}

	for p.current().kind == tokenComma {
		p.advance()
		additional, additionalErr := p.parseSingleAlterAction(database, name)
		if additionalErr != nil {
			return nil, additionalErr
		}
		if additional == nil {
			continue
		}
		if primary == nil {
			primary = additional
			continue
		}
		primary.AdditionalMutations = append(primary.AdditionalMutations, additional)
	}

	if cluster != "" && primary != nil {
		if primary.EngineSpecific == nil {
			primary.EngineSpecific = map[string]string{}
		}
		primary.EngineSpecific[engineClauseOnCluster] = cluster
	}

	return primary, nil
}

// parseSingleAlterAction reads one ALTER TABLE action and returns the corresponding
// CatalogueMutation.
//
// Recognised actions cover ADD COLUMN with optional DEFAULT and AFTER, DROP COLUMN,
// MODIFY COLUMN, RENAME COLUMN ... TO, MATERIALIZE COLUMN, MODIFY TTL, MODIFY ORDER BY,
// MODIFY SETTING, UPDATE ... WHERE, and DELETE WHERE. Unknown actions return a non-nil
// error so the caller can surface a diagnostic; the cursor is left at the offending
// token.
//
// Takes database (string) which is the schema name parsed from the table name.
// Takes table (string) which is the table name the action applies to.
//
// Returns *querier_dto.CatalogueMutation which is the parsed action mutation.
// Returns error when the action keyword is unrecognised or its body is malformed.
func (p *parser) parseSingleAlterAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	switch {
	case p.matchKeyword("ADD"):
		return p.parseAlterAddAction(database, table)
	case p.matchKeyword(keywordDrop):
		return p.parseAlterDropAction(database, table)
	case p.matchKeyword("MODIFY"):
		return p.parseAlterModifyAction(database, table)
	case p.matchKeyword("RENAME"):
		return p.parseAlterRenameAction(database, table)
	case p.matchKeyword("MATERIALIZE"):
		return p.parseAlterMaterializeAction(database, table)
	case p.matchKeyword("UPDATE"):
		return p.parseAlterUpdateAction(database, table)
	case p.matchKeyword("DELETE"):
		return p.parseAlterDeleteAction(database, table)
	case p.matchKeyword("CLEAR"):

		p.consumeUntilTopLevelComma()
		return nil, nil
	case p.matchKeyword("COMMENT"):

		p.consumeUntilTopLevelComma()
		return nil, nil
	case p.matchKeyword("FREEZE"):
		return p.parseAlterPartitionOperation(database, table, "FREEZE", false)
	case p.matchKeyword("UNFREEZE"):
		return p.parseAlterPartitionOperation(database, table, "UNFREEZE", false)
	case p.matchKeyword("ATTACH"):
		return p.parseAlterPartitionOperation(database, table, keywordAttach, false)
	case p.matchKeyword("DETACH"):
		return p.parseAlterPartitionOperation(database, table, "DETACH", false)
	case p.matchKeyword("MOVE"):
		return p.parseAlterPartitionOperation(database, table, "MOVE", false)
	case p.matchKeyword("REPLACE"):
		return p.parseAlterPartitionOperation(database, table, "REPLACE", false)
	case p.matchKeyword("FETCH"):
		return p.parseAlterPartitionOperation(database, table, "FETCH", false)
	default:
		return nil, fmt.Errorf("unexpected ALTER TABLE action at position %d", p.current().position)
	}
}

// parseAlterAddAction handles `ALTER TABLE x ADD COLUMN ...` and the less-common sibling
// forms (ADD INDEX / ADD CONSTRAINT / ADD PROJECTION / ADD STATISTICS). Each sibling form
// is captured into its own mutation kind so the migration runner and downstream codegen
// can re-emit the action without re-parsing the source.
//
// Takes database (string) which is the schema name the column is added to.
// Takes table (string) which is the table name the column is added to.
//
// Returns *querier_dto.CatalogueMutation which is the ADD action mutation.
// Returns error when the column definition or sibling form is malformed.
func (p *parser) parseAlterAddAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	switch {
	case p.matchKeyword(keywordProjection):
		return p.parseAlterAddProjection(database, table)
	case p.matchKeyword(keywordIndex):
		return p.parseAlterAddIndex(database, table)
	case p.matchKeyword(keywordConstraint):
		return p.parseAlterAddConstraint(database, table)
	case p.matchKeyword(keywordStatistics):
		return p.parseAlterAddStatistics(database, table)
	}
	p.matchKeyword(keywordColumn)
	p.matchIfNotExists()

	column, err := p.parseClickHouseColumn()
	if err != nil {
		return nil, err
	}

	if p.matchKeyword(keywordAfter) {
		if _, identErr := p.parseIdentifierOrKeyword(); identErr != nil {
			return nil, identErr
		}
	}

	_ = p.matchKeyword("FIRST")

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAddColumn,
		SchemaName: database,
		TableName:  table,
		Columns:    []querier_dto.Column{column},
	}, nil
}

// parseAlterDropAction handles `ALTER TABLE x DROP COLUMN <name>` and its sibling forms
// (DROP INDEX / DROP CONSTRAINT / DROP PROJECTION / DROP STATISTICS / DROP PARTITION /
// DROP DETACHED PARTITION).
//
// Takes database (string) which is the schema name the drop applies to.
// Takes table (string) which is the table name the drop applies to.
//
// Returns *querier_dto.CatalogueMutation which is the DROP action mutation.
// Returns error when the column name or sibling form is malformed.
func (p *parser) parseAlterDropAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	switch {
	case p.matchKeyword(keywordProjection):
		return p.parseAlterDropProjection(database, table)
	case p.matchKeyword(keywordIndex):
		return p.parseAlterDropIndex(database, table)
	case p.matchKeyword(keywordConstraint):
		return p.parseAlterDropConstraint(database, table)
	case p.matchKeyword(keywordStatistics):
		return p.parseAlterDropStatistics(database, table)
	case p.matchKeyword(keywordPartition):
		return p.parseAlterPartitionOperation(database, table, keywordDrop, false)
	case p.matchKeyword(keywordDetached):
		return p.parseAlterDropDetached(database, table)
	}
	p.matchKeyword(keywordColumn)
	p.matchIfExists()

	columnName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableDropColumn,
		SchemaName: database,
		TableName:  table,
		ColumnName: columnName,
	}, nil
}

// parseAlterModifyAction handles `MODIFY COLUMN ...` (type change) plus `MODIFY TTL`,
// `MODIFY ORDER BY`, `MODIFY SETTING`, `MODIFY QUERY` (for materialised views), `MODIFY
// REFRESH / DEFINER / SQL SECURITY` (refreshable materialised views), `MODIFY
// STATISTICS`, and other sub-targets.
//
// Takes database (string) which is the schema name the modify applies to.
// Takes table (string) which is the table name the modify applies to.
//
// Returns *querier_dto.CatalogueMutation which is the MODIFY action mutation.
// Returns error when the MODIFY target is unrecognised or its body is malformed.
func (p *parser) parseAlterModifyAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	switch {
	case p.matchKeyword(keywordColumn):
		return p.parseAlterModifyColumnAction(database, table)
	case p.matchKeyword("QUERY"):
		return p.parseAlterModifyQuery(database, table)
	case p.matchKeyword(keywordRefresh):
		return p.parseAlterModifyRefresh(database, table, keywordRefresh)
	case p.matchKeyword(keywordSQL):
		if !p.matchKeyword("SECURITY") {
			return nil, fmt.Errorf("expected SECURITY after MODIFY SQL at position %d", p.current().position)
		}
		return p.parseAlterModifyRefresh(database, table, "SQL_SECURITY")
	case p.matchKeyword(keywordDefiner):
		return p.parseAlterModifyRefresh(database, table, keywordDefiner)
	case p.matchKeyword(keywordStatistics):
		return p.parseAlterModifyStatistics(database, table)
	case p.isAnyKeyword("TTL", "ORDER", "SETTING", "SAMPLE", "REMOVE", "RESET"):
		p.consumeUntilTopLevelComma()
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected MODIFY target at position %d", p.current().position)
	}
}

// parseAlterRenameAction handles `RENAME COLUMN <old> TO <new>`.
//
// Takes database (string) which is the schema name the rename applies to.
// Takes table (string) which is the table name the rename applies to.
//
// Returns *querier_dto.CatalogueMutation which is the RENAME COLUMN mutation.
// Returns error when COLUMN, the names, or TO are missing or malformed.
func (p *parser) parseAlterRenameAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	if !p.matchKeyword(keywordColumn) {
		return nil, fmt.Errorf("expected COLUMN after RENAME at position %d", p.current().position)
	}
	p.matchIfExists()
	oldName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	if !p.matchKeyword("TO") {
		return nil, fmt.Errorf("expected TO after RENAME COLUMN at position %d", p.current().position)
	}
	newName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableRenameColumn,
		SchemaName: database,
		TableName:  table,
		ColumnName: oldName,
		NewName:    newName,
	}, nil
}

// parseAlterMaterializeAction handles the MATERIALIZE family of ALTER actions and emits a
// dedicated mutation per sub-target.
//
// The recognised sub-targets are MATERIALIZE COLUMN, MATERIALIZE INDEX, MATERIALIZE
// PROJECTION, MATERIALIZE TTL, and MATERIALIZE STATISTICS, each accepting an optional IN
// PARTITION expression.
//
// Takes database (string) which is the schema name the action applies to.
// Takes table (string) which is the table name the action applies to.
//
// Returns *querier_dto.CatalogueMutation which is the MATERIALIZE mutation.
// Returns error when the sub-target is unrecognised or its body is malformed.
func (p *parser) parseAlterMaterializeAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	switch {
	case p.matchKeyword(keywordProjection):
		return p.parseAlterMaterializeNamed(database, table,
			querier_dto.MutationAlterTableMaterializeProjection, engineKeyProjectionName)
	case p.matchKeyword(keywordIndex):
		return p.parseAlterMaterializeNamed(database, table,
			querier_dto.MutationAlterTableMaterializeIndex, engineKeyIndexName)
	case p.matchKeyword(keywordStatistics):
		return p.parseAlterMaterializeStatistics(database, table)
	case p.matchKeyword(keywordColumn):
		return p.parseAlterMaterializeNamed(database, table,
			querier_dto.MutationAlterTableMaterializeColumn, engineKeyColumnName)
	case p.matchKeyword("TTL"):
		return p.parseAlterMaterializeTTL(database, table)
	default:
		return nil, fmt.Errorf("expected COLUMN / INDEX / PROJECTION / TTL / STATISTICS after MATERIALIZE at position %d", p.current().position)
	}
}

// parseAlterUpdateAction handles `ALTER TABLE x UPDATE col = expr [, ...] WHERE
// predicate`, ClickHouse's asynchronous mutation form.
//
// The catalogue records a MutationAsyncDataUpdate with the SET / WHERE clause body
// captured into EngineSpecific so the migration runner can execute the statement verbatim
// and codegen can choose to expose it or skip it. The mutation does not change the
// table's schema view, so type resolution and column lookups against the table remain
// stable across the migration.
//
// Takes database (string) which is the schema name the update applies to.
// Takes table (string) which is the table name the update applies to.
//
// Returns *querier_dto.CatalogueMutation which is the async update mutation.
// Returns error when the update body cannot be captured.
func (p *parser) parseAlterUpdateAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	body := p.consumeUntilTopLevelCommaAsText()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAsyncDataUpdate,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyAsyncBody: body,
		},
	}, nil
}

// parseAlterDeleteAction handles `ALTER TABLE x DELETE WHERE predicate`. Like ALTER
// UPDATE, this is asynchronous; the WHERE body is captured into EngineSpecific as
// ASYNC_BODY so the statement can be replayed verbatim by the migration runner.
//
// Takes database (string) which is the schema name the delete applies to.
// Takes table (string) which is the table name the delete applies to.
//
// Returns *querier_dto.CatalogueMutation which is the async delete mutation.
// Returns error when the delete body cannot be captured.
func (p *parser) parseAlterDeleteAction(database, table string) (*querier_dto.CatalogueMutation, error) {
	body := p.consumeUntilTopLevelCommaAsText()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAsyncDataDelete,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyAsyncBody: body,
		},
	}, nil
}

// parseCreateView handles `CREATE [OR REPLACE] VIEW [IF NOT EXISTS] [db.]view [AS] SELECT
// ...`. The view's SELECT body is captured but not analysed at parse time; column types
// resolve when a query against the view is analysed.
//
// Returns *querier_dto.CatalogueMutation which is the CREATE VIEW mutation.
// Returns error when the view name or body cannot be parsed.
func (p *parser) parseCreateView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCreate)
	p.skipCreatePrefixesInParser()
	p.mustKeyword("VIEW")
	p.matchIfNotExists()

	database, name, err := p.parseDatabaseQualifiedName()
	if err != nil {
		return nil, err
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateView,
		SchemaName: database,
		TableName:  name,
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific = map[string]string{engineClauseOnCluster: cluster}
	}

	declaredColumns := p.tryParseViewColumnList()
	p.matchKeyword("AS")
	viewBody, viewBodyErr := p.analyseViewBody(declaredColumns)
	mutation.ViewDefinition = viewBody
	if mutation.ViewDefinition != nil {
		mutation.Columns = viewColumnsFromAnalysis(mutation.ViewDefinition, declaredColumns)
	}
	if viewBodyErr != nil {
		return mutation, viewBodyErr
	}
	return mutation, nil
}

// tryParseViewColumnList consumes an optional parenthesised column list following a view
// name (`CREATE VIEW v (a, b, c) AS SELECT ...`).
//
// Returns []string which are the declared column names, or nil when the next token is not
// a `(`.
func (p *parser) tryParseViewColumnList() []string {
	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()
	var names []string
	for !p.atEnd() {
		if p.current().kind != tokenIdentifier {
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
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return names
}

// analyseViewBody re-tokenises the remaining input through a fresh parser and runs
// analyseSelect on it.
//
// The result is (nil, nil) when the body is empty or does not start with a SELECT/WITH
// keyword; the caller then falls back to consumeRemainder behaviour. Panics in the nested
// analyser are recovered into an error so a malformed view body cannot crash the DDL
// apply path; the stack trace is logged engine-side via log.Warn rather than embedded in
// the error, so user-facing surfaces never expose internal paths.
//
// Takes declaredColumns ([]string) which are the explicit view column names that override
// the analyser's output names when present.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysed SELECT body, or nil when
// the body is empty or not a SELECT.
// Returns error when the nested analyser panics.
func (p *parser) analyseViewBody(declaredColumns []string) (result *querier_dto.RawQueryAnalysis, err error) {
	remainingTokens := p.tokens[p.position:]
	if len(remainingTokens) == 0 {
		p.consumeRemainder()
		return nil, nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Warn("clickhouse: panic while analysing view body",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			err = fmt.Errorf("clickhouse: panic while analysing view body: %v", recovered)
			result = nil
		}
	}()

	viewParser := newParser(remainingTokens)
	viewParser.analysisDepth = p.analysisDepth
	viewParser.maxParseDepth = p.maxParseDepth
	if !viewParser.isKeyword("SELECT") && !viewParser.isKeyword("WITH") {
		p.consumeRemainder()
		return nil, nil
	}
	viewAnalysis, analyseError := viewParser.analyseSelect()
	if analyseError != nil || viewAnalysis == nil {
		p.consumeRemainder()
		return nil, nil
	}
	p.consumeRemainder()

	if len(declaredColumns) > 0 {
		overlayViewColumnNamesOnAnalysis(viewAnalysis, declaredColumns)
	}
	return viewAnalysis, nil
}

// overlayViewColumnNamesOnAnalysis renames the analysis's output columns to match a
// declared `CREATE VIEW v (col1, col2)` list. Tolerates declared lists longer than the
// projection by appending name-only entries.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which holds the output columns to rename
// in place.
// Takes columnNames ([]string) which are the declared names applied in order.
func overlayViewColumnNamesOnAnalysis(analysis *querier_dto.RawQueryAnalysis, columnNames []string) {
	for columnIndex, name := range columnNames {
		if columnIndex < len(analysis.OutputColumns) {
			analysis.OutputColumns[columnIndex].Name = name
		} else {
			analysis.OutputColumns = append(analysis.OutputColumns, querier_dto.RawOutputColumn{
				Name: name,
			})
		}
	}
}

// viewColumnsFromAnalysis lifts the analyser's output columns into the catalogue's Column
// shape, preferring declared names when the caller supplied them.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which holds the output columns to
// convert.
// Takes declaredColumns ([]string) which are the declared names that override the
// analyser names when present.
//
// Returns []querier_dto.Column which are the catalogue columns for the view.
func viewColumnsFromAnalysis(analysis *querier_dto.RawQueryAnalysis, declaredColumns []string) []querier_dto.Column {
	columns := make([]querier_dto.Column, 0, len(analysis.OutputColumns))
	for index, output := range analysis.OutputColumns {
		name := output.Name
		if index < len(declaredColumns) && declaredColumns[index] != "" {
			name = declaredColumns[index]
		}
		name = cmp.Or(name, output.ColumnName)
		columns = append(columns, querier_dto.Column{
			Name:     name,
			SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			Nullable: true,
		})
	}
	return columns
}

// parseCreateMaterializedView handles the full `CREATE MATERIALIZED VIEW` grammar.
//
// It accepts the optional IF NOT EXISTS guard, a column list, the refresh clauses
// (REFRESH, RANDOMIZE FOR, DEPENDS ON, APPEND or EMPTY), a storage target (TO
// target_table or an inline ENGINE / PARTITION BY / ORDER BY declaration), the trailing
// modifiers (POPULATE, DEFINER, SQL SECURITY, COMMENT), and the AS SELECT body.
// Refreshable materialised view clauses are captured into EngineSpecific under MV_* keys.
//
// Returns *querier_dto.CatalogueMutation which is the CREATE VIEW mutation.
// Returns error when any clause or the SELECT body cannot be parsed.
func (p *parser) parseCreateMaterializedView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCreate)
	p.skipCreatePrefixesInParser()
	p.mustKeyword("MATERIALIZED")
	p.mustKeyword("VIEW")
	p.matchIfNotExists()

	database, name, err := p.parseDatabaseQualifiedName()
	if err != nil {
		return nil, err
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationCreateView,
		SchemaName:     database,
		TableName:      name,
		EngineSpecific: map[string]string{"MATERIALIZED": mvBooleanTrue},
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific[engineClauseOnCluster] = cluster
	}
	declaredColumns := p.tryParseViewColumnList()
	p.parseMaterializedViewRefreshClauses(mutation)
	if storageErr := p.parseMaterializedViewStorageClauses(mutation); storageErr != nil {
		return nil, storageErr
	}
	if modifierErr := p.parseMaterializedViewTrailingModifiers(mutation); modifierErr != nil {
		return nil, modifierErr
	}
	return p.parseMaterializedViewBody(mutation, declaredColumns)
}

// parseMaterializedViewStorageClauses recognises the optional `TO target` clause or the
// inline storage declaration (ENGINE / PARTITION / ORDER / etc.) following the refresh
// clauses.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured storage
// metadata in its EngineSpecific map.
//
// Returns error when the TO target or an inline engine clause is malformed.
func (p *parser) parseMaterializedViewStorageClauses(mutation *querier_dto.CatalogueMutation) error {
	if p.matchKeyword(keywordTo) {
		_, target, targetErr := p.parseDatabaseQualifiedName()
		if targetErr != nil {
			return targetErr
		}
		mutation.EngineSpecific[engineKeyMaterializedTarget] = target
		return nil
	}
	if p.isAnyKeyword(keywordPopulate, keywordEmpty, keywordAs, keywordDefiner, keywordSQL, engineClauseComment) {
		return nil
	}
	return p.parseTableEngineClauses(mutation)
}

// parseMaterializedViewTrailingModifiers recognises the POPULATE / EMPTY / DEFINER / SQL
// SECURITY / COMMENT modifiers placed between the storage clause and the AS SELECT body.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured modifier
// metadata in its EngineSpecific map.
//
// Returns error when a recognised security clause has a malformed body.
func (p *parser) parseMaterializedViewTrailingModifiers(mutation *querier_dto.CatalogueMutation) error {
	if p.matchKeyword(keywordPopulate) {
		mutation.EngineSpecific[keywordPopulate] = mvBooleanTrue
	}
	if p.matchKeyword(keywordEmpty) {
		mutation.EngineSpecific["MV_REFRESH_EMPTY"] = mvBooleanTrue
	}
	if securityErr := p.parseMaterializedViewSecurityClauses(mutation); securityErr != nil {
		return securityErr
	}
	if p.matchKeyword("COMMENT") && p.current().kind == tokenString {
		mutation.EngineSpecific[engineClauseComment] = p.current().value
		p.advance()
	}
	return nil
}

// parseMaterializedViewBody analyses the optional `AS SELECT ...` body. The declared
// column list (when present) overrides the analyser's output column names so the
// materialised view's projection matches the explicit declaration.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the analysed view
// definition and columns.
// Takes declaredColumns ([]string) which override the analyser's output names.
//
// Returns *querier_dto.CatalogueMutation which is the populated mutation.
// Returns error when the view body cannot be analysed.
func (p *parser) parseMaterializedViewBody(mutation *querier_dto.CatalogueMutation, declaredColumns []string) (*querier_dto.CatalogueMutation, error) {
	if !p.matchKeyword(keywordAs) {
		p.consumeRemainder()
		return mutation, nil
	}
	viewBody, bodyErr := p.analyseViewBody(declaredColumns)
	mutation.ViewDefinition = viewBody
	if mutation.ViewDefinition != nil {
		mutation.Columns = viewColumnsFromAnalysis(mutation.ViewDefinition, declaredColumns)
	}
	return mutation, bodyErr
}

// parseMaterializedViewRefreshClauses recognises the refreshable MV REFRESH / RANDOMIZE
// FOR / DEPENDS ON / APPEND clauses.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured refresh
// metadata in its EngineSpecific map.
func (p *parser) parseMaterializedViewRefreshClauses(mutation *querier_dto.CatalogueMutation) {
	for {
		switch {
		case p.matchKeyword(keywordRefresh):
			p.captureRefreshClause(mutation)
		case p.matchKeyword("RANDOMIZE"):
			if p.matchKeyword("FOR") {
				mutation.EngineSpecific["MV_REFRESH_RANDOMIZE"] = p.captureIntervalText()
			}
		case p.matchKeyword("DEPENDS"):
			if p.matchKeyword("ON") {
				mutation.EngineSpecific["MV_REFRESH_DEPENDS_ON"] = p.captureDependsOnList()
			}
		case p.matchKeyword("APPEND"):
			mutation.EngineSpecific["MV_REFRESH_APPEND"] = mvBooleanTrue
		default:
			return
		}
	}
}

// captureRefreshClause reads the body of a REFRESH clause.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured refresh
// kind and interval in its EngineSpecific map.
func (p *parser) captureRefreshClause(mutation *querier_dto.CatalogueMutation) {
	switch {
	case p.matchKeyword(keywordEvery):
		mutation.EngineSpecific[engineKeyMVRefreshKind] = keywordEvery
		mutation.EngineSpecific[engineKeyMVRefreshInterval] = p.captureIntervalText()
		if p.matchKeyword("OFFSET") {
			mutation.EngineSpecific[engineKeyMVRefreshOffset] = p.captureIntervalText()
		}
	case p.matchKeyword(keywordAfter):
		mutation.EngineSpecific[engineKeyMVRefreshKind] = keywordAfter
		mutation.EngineSpecific[engineKeyMVRefreshAfter] = p.captureIntervalText()
	}
}

// captureIntervalText reads an interval-shaped expression up to a recognised tail keyword
// or top-level boundary. The terminating set is the recognised MV boundary keyword family
// plus a top-level comma; splitStatements strips semicolons before parsing so the helper
// does not need to guard against them inside a single statement's token stream.
//
// Returns string which is the trimmed interval expression text.
func (p *parser) captureIntervalText() string {
	var builder strings.Builder
	for !p.atEnd() {
		tok := p.current()
		if tok.kind == tokenIdentifier && isMaterializedViewBoundaryKeyword(tok.value) {
			break
		}
		if tok.kind == tokenComma {
			break
		}
		writeTokenAsSourceText(&builder, tok)
		builder.WriteByte(' ')
		p.advance()
	}
	return strings.TrimSpace(builder.String())
}

// isMaterializedViewBoundaryKeyword reports whether the identifier terminates an interval
// or DEPENDS-ON expression.
//
// Takes text (string) which is the identifier to test.
//
// Returns bool which is true when the identifier ends the expression.
func isMaterializedViewBoundaryKeyword(text string) bool {
	switch strings.ToUpper(text) {
	case "OFFSET", "RANDOMIZE", "DEPENDS", "APPEND", keywordTo, engineClauseEngine,
		"POPULATE", "EMPTY", keywordDefiner, keywordSQL, engineClauseComment, "AS",
		"PARTITION", "ORDER", "PRIMARY", "SAMPLE", engineClauseTTL, "SETTINGS":
		return true
	default:
		return false
	}
}

// captureDependsOnList reads the comma-separated table identifier list following `DEPENDS
// ON`.
//
// Returns string which is the table list flattened with the dependsOnSeparator.
func (p *parser) captureDependsOnList() string {
	var entries []string
	for !p.atEnd() {
		_, name, parseErr := p.parseDatabaseQualifiedName()
		if parseErr != nil {
			return strings.Join(entries, dependsOnSeparator)
		}
		entries = append(entries, name)
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		break
	}
	return strings.Join(entries, dependsOnSeparator)
}

// parseCreateDictionary handles `CREATE DICTIONARY [IF NOT EXISTS] [db.]dict
// (column_list) PRIMARY KEY (cols) SOURCE(name(...)) LAYOUT(name(...)) LIFETIME(MIN n MAX
// m | seconds) RANGE(MIN col MAX col) SETTINGS(k = v, ...) COMMENT 'text'`.
//
// Dictionaries are ClickHouse's key-value lookup tables; the catalogue captures their
// column shape plus structured engine metadata so codegen can expose dictGet() helpers
// with typed return values. The SOURCE / LAYOUT / LIFETIME / RANGE clause bodies are
// captured under dedicated EngineSpecific keys; the parser reads each one declaratively
// so downstream consumers do not have to re-parse the SQL text.
//
// Returns a CatalogueMutation of kind MutationCreateDictionary with columns, primary key
// list, and engine-specific metadata. Returns a non-nil error when a malformed clause is
// encountered.
func (p *parser) parseCreateDictionary() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCreate)
	p.skipCreatePrefixesInParser()
	p.mustKeyword(keywordDictionary)
	p.matchIfNotExists()

	database, name, err := p.parseDatabaseQualifiedName()
	if err != nil {
		return nil, err
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationCreateDictionary,
		SchemaName:     database,
		TableName:      name,
		EngineSpecific: map[string]string{keywordDictionary: "true"},
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific[engineClauseOnCluster] = cluster
	}

	if p.current().kind == tokenLeftParen {
		columns, columnErr := p.parseDictionaryColumnList()
		if columnErr != nil {
			return nil, columnErr
		}
		mutation.Columns = columns
	}

	for {
		handled, clauseErr := p.parseSingleDictionaryClause(mutation)
		if clauseErr != nil {
			return nil, clauseErr
		}
		if !handled {
			break
		}
	}
	p.consumeRemainder()
	return mutation, nil
}

// parseDictionaryColumnList reads the parenthesised column list of a CREATE DICTIONARY
// body. Each entry is a column definition that shares the CREATE TABLE column grammar
// (DEFAULT, EXPRESSION, HIERARCHICAL, INJECTIVE, IS_OBJECT_ID); modifiers beyond the type
// are consumed for forward progress but only the name and the type land on the captured
// Column.
//
// Returns []querier_dto.Column which are the captured dictionary columns.
// Returns error when the list is not parenthesised or a column is malformed.
func (p *parser) parseDictionaryColumnList() ([]querier_dto.Column, error) {
	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	var columns []querier_dto.Column
	for {
		if p.current().kind == tokenRightParen {
			p.advance()
			return columns, nil
		}
		column, err := p.parseClickHouseColumn()
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)
		switch p.current().kind {
		case tokenComma:
			p.advance()
			continue
		case tokenRightParen:
			p.advance()
			return columns, nil
		default:
			return nil, fmt.Errorf("expected ',' or ')' in dictionary columns at position %d", p.current().position)
		}
	}
}

// parseSingleDictionaryClause matches one trailing clause of a CREATE DICTIONARY body and
// writes its captured value into the mutation's EngineSpecific map.
//
// The recognised clauses are PRIMARY KEY for the key columns, SOURCE for the data source
// descriptor, LAYOUT for the in-memory layout descriptor, LIFETIME for the refresh
// schedule, RANGE for ranged dictionary bounds, SETTINGS for engine settings, and COMMENT
// for the descriptive comment.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured clause
// value in its EngineSpecific map.
//
// Returns bool which is true when a clause was consumed and false when the outer loop
// should stop.
// Returns error when a clause body is malformed.
func (p *parser) parseSingleDictionaryClause(mutation *querier_dto.CatalogueMutation) (bool, error) {
	if p.isKeyword("PRIMARY") {
		return p.captureDictionaryPrimaryKey(mutation)
	}
	if p.matchKeyword("SOURCE") {
		return p.captureNamedDictionaryClause(mutation, "DICTIONARY_SOURCE")
	}
	if p.matchKeyword("LAYOUT") {
		return p.captureNamedDictionaryClause(mutation, "DICTIONARY_LAYOUT")
	}
	if p.matchKeyword("LIFETIME") {
		return p.captureDictionaryLifetimeClause(mutation)
	}
	if p.matchKeyword("RANGE") {
		return p.captureParenthesisedDictionaryClause(mutation, "DICTIONARY_RANGE")
	}
	if p.matchKeyword("SETTINGS") {
		return p.captureDictionarySettingsClause(mutation)
	}
	if p.matchKeyword("COMMENT") {
		p.captureDictionaryCommentClause(mutation)
		return true, nil
	}
	return false, nil
}

// captureNamedDictionaryClause reads a `name(body)` clause (SOURCE, LAYOUT) and writes
// the reconstructed text under the supplied EngineSpecific key. The leading clause
// keyword has already been consumed.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured clause
// body.
// Takes engineKey (string) which is the EngineSpecific key to store the body under.
//
// Returns bool which is true when the clause was consumed.
// Returns error when the named paren body is malformed.
func (p *parser) captureNamedDictionaryClause(mutation *querier_dto.CatalogueMutation, engineKey string) (bool, error) {
	body, err := p.captureNamedParenBody()
	if err != nil {
		return false, err
	}
	mutation.EngineSpecific[engineKey] = body
	return true, nil
}

// captureDictionaryLifetimeClause delegates to captureDictionaryLifetime and stores the
// result on the mutation under the canonical DICTIONARY_LIFETIME_SECONDS key.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured lifetime
// value.
//
// Returns bool which is true when the clause was consumed.
// Returns error when the LIFETIME body is malformed.
func (p *parser) captureDictionaryLifetimeClause(mutation *querier_dto.CatalogueMutation) (bool, error) {
	seconds, err := p.captureDictionaryLifetime()
	if err != nil {
		return false, err
	}
	mutation.EngineSpecific["DICTIONARY_LIFETIME_SECONDS"] = seconds
	return true, nil
}

// captureParenthesisedDictionaryClause reads a parenthesised body (RANGE, SETTINGS) and
// writes the captured text under the supplied EngineSpecific key.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured clause
// body.
// Takes engineKey (string) which is the EngineSpecific key to store the body under.
//
// Returns bool which is true when the clause was consumed.
// Returns error when the parenthesised body is malformed.
func (p *parser) captureParenthesisedDictionaryClause(mutation *querier_dto.CatalogueMutation, engineKey string) (bool, error) {
	body, err := p.captureParenthesisedBodyAsText()
	if err != nil {
		return false, err
	}
	mutation.EngineSpecific[engineKey] = body
	return true, nil
}

// captureDictionarySettingsClause handles the SETTINGS clause body. Unlike
// captureParenthesisedDictionaryClause it tolerates the settings-keyword form without a
// body (where the cursor is left past the keyword without a leading paren).
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured settings
// body.
//
// Returns bool which is true when the clause was consumed.
// Returns error when the settings body is malformed.
func (p *parser) captureDictionarySettingsClause(mutation *querier_dto.CatalogueMutation) (bool, error) {
	if p.current().kind != tokenLeftParen {
		return true, nil
	}
	body, err := p.captureParenthesisedBodyAsText()
	if err != nil {
		return false, err
	}
	mutation.EngineSpecific["SETTINGS"] = body
	return true, nil
}

// captureDictionaryCommentClause consumes a trailing `COMMENT 'text'` clause and writes
// the text under the EngineSpecific COMMENT key. When the cursor is not on a string
// literal the clause is dropped silently (treating the comment as empty).
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured comment
// text.
func (p *parser) captureDictionaryCommentClause(mutation *querier_dto.CatalogueMutation) {
	if p.current().kind == tokenString {
		mutation.EngineSpecific[engineClauseComment] = p.current().value
		p.advance()
	}
}

// captureDictionaryPrimaryKey consumes the `PRIMARY KEY (col, ...)` or `PRIMARY KEY col`
// form and stores the key column list onto the mutation.
//
// The list is recorded under both the catalogue's PrimaryKey field and the EngineSpecific
// DICTIONARY_PRIMARY_KEY key for ClickHouse-aware consumers. The cursor is rewound and
// handled=false is returned when PRIMARY is not followed by KEY so an unrelated
// identifier starting with PRIMARY does not get misclassified.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured primary key
// columns.
//
// Returns bool which is true when the clause was consumed.
// Returns error when the key list is malformed.
func (p *parser) captureDictionaryPrimaryKey(mutation *querier_dto.CatalogueMutation) (bool, error) {
	saved := p.position
	p.advance()
	if !p.matchKeyword("KEY") {
		p.position = saved
		return false, nil
	}
	keys, err := p.parseDictionaryPrimaryKeyList()
	if err != nil {
		return false, err
	}
	mutation.PrimaryKey = keys
	mutation.EngineSpecific["DICTIONARY_PRIMARY_KEY"] = strings.Join(keys, ", ")
	return true, nil
}

// parseDictionaryPrimaryKeyList reads the column list after a `PRIMARY KEY` keyword
// inside a CREATE DICTIONARY body. Both the parenthesised `PRIMARY KEY (a, b)` and the
// bare `PRIMARY KEY a` forms are accepted.
//
// Returns []string which are the primary key column names.
// Returns error when an identifier or closing punctuation is missing.
func (p *parser) parseDictionaryPrimaryKeyList() ([]string, error) {
	if p.current().kind != tokenLeftParen {
		name, err := p.parseIdentifierOrKeyword()
		if err != nil {
			return nil, err
		}
		return []string{name}, nil
	}
	p.advance()
	var keys []string
	for {
		name, err := p.parseIdentifierOrKeyword()
		if err != nil {
			return nil, err
		}
		keys = append(keys, name)
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		if p.current().kind == tokenRightParen {
			p.advance()
			return keys, nil
		}
		return nil, fmt.Errorf("expected ',' or ')' in PRIMARY KEY at position %d", p.current().position)
	}
}

// captureNamedParenBody reads a `name(body)` construct (used by SOURCE / LAYOUT clauses)
// and returns the concatenated text "name(body_text)". The body text mirrors the source
// representation so downstream consumers can re-emit verbatim.
//
// Returns string which is the reconstructed "name(body)" text.
// Returns error when the opening paren, name, or closing paren is missing.
func (p *parser) captureNamedParenBody() (string, error) {
	if p.current().kind != tokenLeftParen {
		return "", fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	if p.current().kind != tokenIdentifier {
		return "", fmt.Errorf("expected name identifier at position %d", p.current().position)
	}
	name := p.current().value
	p.advance()
	body := ""
	if p.current().kind == tokenLeftParen {
		captured, err := p.captureParenthesisedBodyAsText()
		if err != nil {
			return "", err
		}
		body = captured
	}
	if p.current().kind != tokenRightParen {
		return "", fmt.Errorf("expected ')' at position %d", p.current().position)
	}
	p.advance()
	return name + "(" + body + ")", nil
}

// captureParenthesisedBodyAsText reads a balanced parenthesised body (cursor on the
// leading `(`) and returns the inner source text without the outer parens.
//
// Nested parens are preserved verbatim. The dictionary clause helpers use it to capture
// SOURCE/LAYOUT/RANGE/SETTINGS bodies for downstream re-emission.
//
// Returns string which is the inner body text without the outer parens.
// Returns error when the parentheses are unbalanced.
func (p *parser) captureParenthesisedBodyAsText() (string, error) {
	if p.current().kind != tokenLeftParen {
		return "", fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	var builder strings.Builder
	depth := 1
	for !p.atEnd() && depth > 0 {
		tok := p.current()
		newDepth, finished := emitCaptureToken(&builder, tok, depth)
		depth = newDepth
		if finished {
			p.advance()
			return builder.String(), nil
		}
		p.advance()
	}
	return "", errUnmatchedParenthesis
}

// emitCaptureToken writes one token's serialised form into builder (used by
// captureParenthesisedBodyAsText). Returns the updated paren depth plus finished=true
// when the matching close paren has been reached, so the caller can return the
// accumulated text.
//
// Takes builder (*strings.Builder), the destination buffer.
// Takes tok (token), the token to emit.
// Takes depth (int), the current paren depth before this token.
//
// Returns newDepth (int), the depth after accounting for this token.
// Returns finished (bool), true when this token closed the outer parens.
func emitCaptureToken(builder *strings.Builder, tok token, depth int) (newDepth int, finished bool) {
	switch tok.kind {
	case tokenLeftParen:
		builder.WriteByte('(')
		return depth + 1, false
	case tokenRightParen:
		depth--
		if depth == 0 {
			return depth, true
		}
		builder.WriteByte(')')
		return depth, false
	case tokenComma:
		builder.WriteString(", ")
	case tokenString:
		emitSpacingIfNeeded(builder)
		builder.WriteByte('\'')
		builder.WriteString(escapeClickHouseStringBody(tok.value))
		builder.WriteByte('\'')
	case tokenIdentifier, tokenNumber, tokenOperator:
		emitSpacingIfNeeded(builder)
		builder.WriteString(tok.value)
	case tokenDot:
		builder.WriteByte('.')
	default:
	}
	return depth, false
}

// escapeClickHouseStringBody re-escapes a decoded string-literal body so it can be safely
// re-wrapped in single quotes.
//
// The tokeniser stores the unescaped body (it collapses a doubled single quote and a
// backslash-escaped single quote to one single quote, and a doubled backslash to one
// backslash), so a faithful re-emit must escape the backslash first (otherwise the
// backslashes added for quotes would themselves be doubled) and then double the single
// quotes. This is the same convention the generated ClickHouse value formatter uses for
// composite literals.
//
// Takes body (string) which is the decoded string-literal contents.
//
// Returns string which is the re-escaped body without the enclosing quotes.
func escapeClickHouseStringBody(body string) string {
	escaped := strings.ReplaceAll(body, "\\", "\\\\")
	return strings.ReplaceAll(escaped, "'", "''")
}

// emitSpacingIfNeeded writes a separator space when the last emitted byte requires one
// before the next identifier-like token. Centralised so emitCaptureToken's call-sites do
// not duplicate the lastByteNeedsSpace guard.
//
// Takes builder (*strings.Builder) which is the destination buffer to space.
func emitSpacingIfNeeded(builder *strings.Builder) {
	if builder.Len() > 0 && lastByteNeedsSpace(builder.String()) {
		builder.WriteByte(' ')
	}
}

// lastByteNeedsSpace reports whether the previously emitted byte requires a separating
// space before the next identifier-like token. Used by captureParenthesisedBodyAsText to
// avoid concatenating two identifiers together when their tokens were adjacent in the
// source.
//
// Takes emitted (string) which is the text accumulated so far.
//
// Returns bool which is true when a separating space is required.
func lastByteNeedsSpace(emitted string) bool {
	if emitted == "" {
		return false
	}
	last := emitted[len(emitted)-1]
	switch last {
	case '(', ' ', '.':
		return false
	default:
		return true
	}
}

// captureDictionaryLifetime reads the LIFETIME clause body.
//
// ClickHouse accepts the single-integer form LIFETIME(seconds) and the refresh window
// form LIFETIME(MIN n MAX m) in seconds. The returned string is either the literal
// seconds value or the formatted "MIN n MAX m" pair, depending on which form the source
// used.
//
// Returns string which is the captured lifetime value.
// Returns error when the LIFETIME parentheses or contents are malformed.
func (p *parser) captureDictionaryLifetime() (string, error) {
	if p.current().kind != tokenLeftParen {
		return "", fmt.Errorf("expected '(' after LIFETIME at position %d", p.current().position)
	}
	p.advance()
	var minValue, maxValue, single string
	for !p.atEnd() && p.current().kind != tokenRightParen {
		switch {
		case p.matchKeyword("MIN"):
			value, err := p.captureLifetimeNumber()
			if err != nil {
				return "", err
			}
			minValue = value
		case p.matchKeyword("MAX"):
			value, err := p.captureLifetimeNumber()
			if err != nil {
				return "", err
			}
			maxValue = value
		case p.current().kind == tokenNumber:
			single = p.current().value
			p.advance()
		default:
			p.advance()
		}
	}
	if p.current().kind != tokenRightParen {
		return "", fmt.Errorf("expected ')' to close LIFETIME at position %d", p.current().position)
	}
	p.advance()
	return formatLifetimeValue(minValue, maxValue, single), nil
}

// formatLifetimeValue renders a captured LIFETIME clause body. When MIN and/or MAX bounds
// were present it emits only the bounds that actually appeared (so LIFETIME(MIN n) with
// no MAX does not produce a malformed "MIN n MAX " value with a trailing empty bound);
// otherwise it returns the single-integer form.
//
// Takes minValue / maxValue (string) which are the captured MIN / MAX bounds (empty when
// absent), and single (string) which is the single-integer lifetime.
//
// Returns the formatted LIFETIME body.
func formatLifetimeValue(minValue, maxValue, single string) string {
	if minValue == "" && maxValue == "" {
		return single
	}
	parts := make([]string, 0, 2)
	if minValue != "" {
		parts = append(parts, "MIN "+minValue)
	}
	if maxValue != "" {
		parts = append(parts, "MAX "+maxValue)
	}
	return strings.Join(parts, " ")
}

// captureLifetimeNumber reads a single numeric literal used as the MIN or MAX bound of a
// LIFETIME clause.
//
// Returns string which is the literal number text.
// Returns error when the cursor is not on a number.
func (p *parser) captureLifetimeNumber() (string, error) {
	if p.current().kind != tokenNumber {
		return "", fmt.Errorf("expected integer at position %d", p.current().position)
	}
	value := p.current().value
	p.advance()
	return value, nil
}

// parseDropDictionary handles `DROP DICTIONARY [IF EXISTS] [db.]name`.
//
// Returns *querier_dto.CatalogueMutation which is the DROP DICTIONARY mutation.
// Returns error when the dictionary name cannot be parsed.
func (p *parser) parseDropDictionary() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDrop)
	p.mustKeyword(keywordDictionary)
	p.matchIfExists()
	database, name, err := p.parseDatabaseQualifiedName()
	if err != nil {
		return nil, err
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropDictionary,
		SchemaName: database,
		TableName:  name,
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific = map[string]string{engineClauseOnCluster: cluster}
	}
	return mutation, nil
}

// parseRenameTable handles `RENAME TABLE old TO new [, ...]` (and the DICTIONARY /
// DATABASE variants). Multiple renames in a single statement produce a primary mutation
// plus AdditionalMutations.
//
// Returns *querier_dto.CatalogueMutation which is the primary rename mutation.
// Returns error when the RENAME keyword or any clause cannot be parsed.
func (p *parser) parseRenameTable() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("RENAME")

	if _, err := p.expectKeyword("TABLE", "DICTIONARY", "DATABASE"); err != nil {
		return nil, err
	}

	primary, err := p.parseSingleRename()
	if err != nil {
		return nil, err
	}
	for p.current().kind == tokenComma {
		p.advance()
		additional, additionalErr := p.parseSingleRename()
		if additionalErr != nil {
			return nil, additionalErr
		}
		primary.AdditionalMutations = append(primary.AdditionalMutations, additional)
	}
	return primary, nil
}

// parseSingleRename reads one `old [TO|AS] new` rename clause. Optional trailing `ON
// CLUSTER c` is consumed and captured into EngineSpecific.
//
// Returns *querier_dto.CatalogueMutation which is the rename mutation.
// Returns error when the names or the TO keyword are missing.
func (p *parser) parseSingleRename() (*querier_dto.CatalogueMutation, error) {
	oldDB, oldName, err := p.parseDatabaseQualifiedName()
	if err != nil {
		return nil, err
	}
	if !p.matchKeyword("TO") {
		return nil, fmt.Errorf("expected TO at position %d", p.current().position)
	}
	_, newName, newErr := p.parseDatabaseQualifiedName()
	if newErr != nil {
		return nil, newErr
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableRenameTable,
		SchemaName: oldDB,
		TableName:  oldName,
		NewName:    newName,
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific = map[string]string{engineClauseOnCluster: cluster}
	}
	return mutation, nil
}

// parseExchangeTables handles `EXCHANGE TABLES a AND b [ON CLUSTER c]`, ClickHouse's
// atomic two-table swap. The primary mutation captures the left table on its SchemaName /
// TableName pair; the right table is encoded into EngineSpecific under
// engineKeyExchangeTarget as `[db.]name` so downstream consumers can read both sides.
//
// Both halves of the swap may carry a database qualifier; the optional ON CLUSTER clause
// is captured under engineClauseOnCluster matching the convention used by
// parseSingleRename.
//
// Returns *querier_dto.CatalogueMutation which is the EXCHANGE TABLES mutation.
// Returns error when TABLES, AND, or either table name is missing.
func (p *parser) parseExchangeTables() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("EXCHANGE")
	if !p.matchKeyword("TABLES") {
		return nil, fmt.Errorf("expected TABLES after EXCHANGE at position %d", p.current().position)
	}
	leftDB, leftName, leftErr := p.parseDatabaseQualifiedName()
	if leftErr != nil {
		return nil, leftErr
	}
	if !p.matchKeyword("AND") {
		return nil, fmt.Errorf("expected AND between exchange targets at position %d", p.current().position)
	}
	rightDB, rightName, rightErr := p.parseDatabaseQualifiedName()
	if rightErr != nil {
		return nil, rightErr
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationExchangeTables,
		SchemaName:     leftDB,
		TableName:      leftName,
		EngineSpecific: map[string]string{engineKeyExchangeTarget: qualifiedName(rightDB, rightName)},
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		mutation.EngineSpecific[engineClauseOnCluster] = cluster
	}
	return mutation, nil
}

// qualifiedName joins a database qualifier and a name into the `database.name` form,
// dropping the qualifier when empty so the output remains a single bare identifier.
//
// Takes database (string), the optional database qualifier.
// Takes name (string), the object name.
//
// Returns the qualified form, or the bare name when database is "".
func qualifiedName(database string, name string) string {
	if database == "" {
		return quoteClickHouseIdentifierIfNeeded(name)
	}
	return quoteClickHouseIdentifierIfNeeded(database) + "." + quoteClickHouseIdentifierIfNeeded(name)
}

// quoteClickHouseIdentifierIfNeeded backtick-quotes an identifier when it falls outside
// the unquoted-identifier grammar.
//
// An identifier with a leading digit, or any byte that is not a letter, digit or
// underscore, is quoted so a re-emitted qualified name round-trips losslessly instead of
// producing an invalid bare identifier. Plain identifiers are returned unchanged so the
// common case is byte-for-byte identical to before.
//
// Takes identifier (string) which is the bare identifier to consider quoting.
//
// Returns string which is the identifier, backtick-quoted only when required.
func quoteClickHouseIdentifierIfNeeded(identifier string) string {
	if !clickHouseIdentifierNeedsQuoting(identifier) {
		return identifier
	}
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

// clickHouseIdentifierNeedsQuoting reports whether identifier needs quoting.
//
// It returns true when identifier falls outside the unquoted identifier grammar and
// therefore must be quoted to round-trip. The scan is rune-aware (delegating to
// identifierNeedsQuoting) so multi-byte Unicode letters inside an identifier are
// classified correctly rather than over-quoted byte-by-byte. An empty string is treated
// as needing quoting so it cannot collapse into an adjacent token; this differs from
// identifierNeedsQuoting, which reports an empty identifier as not needing quoting.
//
// Takes identifier (string) which is the identifier to test.
//
// Returns bool which is true when the identifier must be quoted.
func clickHouseIdentifierNeedsQuoting(identifier string) bool {
	if identifier == "" {
		return true
	}
	return identifierNeedsQuoting(identifier)
}

// parseTruncateTable handles `TRUNCATE TABLE [IF EXISTS] [db.]name`. Pure runtime
// operation; no catalogue effect.
//
// Returns *querier_dto.CatalogueMutation which is always nil for this operation.
// Returns error which is always nil because the statement is consumed wholesale.
func (p *parser) parseTruncateTable() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("TRUNCATE")
	p.matchKeyword("TABLE")
	p.matchIfExists()
	p.consumeRemainder()
	return nil, nil
}

// parseOptimize handles the ClickHouse table-optimisation DDL (`... TABLE [db.]name
// [FINAL] [DEDUPLICATE] [BY col_list]`).
//
// This is a pure runtime operation with no catalogue effect.
//
// Returns *querier_dto.CatalogueMutation which is always nil for this operation.
// Returns error which is always nil because the statement is consumed wholesale.
func (p *parser) parseOptimize() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("OPTIMIZE")
	p.consumeRemainder()
	return nil, nil
}

// parseSystem handles `SYSTEM FLUSH | RELOAD | START | STOP ...`. Pure runtime operation.
//
// Returns *querier_dto.CatalogueMutation which is always nil for this operation.
// Returns error which is always nil because the statement is consumed wholesale.
func (p *parser) parseSystem() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("SYSTEM")
	p.consumeRemainder()
	return nil, nil
}

// parseUseDatabase handles `USE name`. Pure session operation; no catalogue effect.
//
// Returns *querier_dto.CatalogueMutation which is always nil for this operation.
// Returns error which is always nil because the statement is consumed wholesale.
func (p *parser) parseUseDatabase() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("USE")
	p.consumeRemainder()
	return nil, nil
}

// parseShow handles SHOW TABLES / SHOW DATABASES / SHOW CREATE TABLE / etc. Pure
// introspection; no catalogue effect.
//
// Returns *querier_dto.CatalogueMutation which is always nil for this operation.
// Returns error which is always nil because the statement is consumed wholesale.
func (p *parser) parseShow() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("SHOW")
	p.consumeRemainder()
	return nil, nil
}

// parseSet handles `SET name = value`. Pure session operation; no catalogue effect.
//
// Returns *querier_dto.CatalogueMutation which is always nil for this operation.
// Returns error which is always nil because the statement is consumed wholesale.
func (p *parser) parseSet() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("SET")
	p.consumeRemainder()
	return nil, nil
}
