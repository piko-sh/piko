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
	"context"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// catalogueBuilder constructs a Catalogue by replaying DDL mutations from migration
// files. Each migration file is parsed via the engine adapter and the resulting mutations
// are applied sequentially to build up the schema state.
type catalogueBuilder struct {
	// engine holds the database engine adapter used for parsing and DDL interpretation.
	engine EnginePort

	// catalogue holds the schema state being built up by replaying mutations.
	catalogue *querier_dto.Catalogue
}

// newCatalogueBuilder creates a new catalogue builder with an empty catalogue. The
// default schema is set based on the engine dialect.
//
// Takes engine (EnginePort) which provides the database dialect and DDL interpretation.
//
// Returns *catalogueBuilder which is ready to apply migration mutations.
func newCatalogueBuilder(engine EnginePort) *catalogueBuilder {
	defaultSchema := engine.DefaultSchema()

	catalogue := &querier_dto.Catalogue{
		Schemas:       make(map[string]*querier_dto.Schema),
		DefaultSchema: defaultSchema,
		Extensions:    make(map[string]struct{}),
	}

	if defaultSchema != "" {
		catalogue.Schemas[defaultSchema] = newEmptySchema(defaultSchema)
	}

	return &catalogueBuilder{
		engine:    engine,
		catalogue: catalogue,
	}
}

// ApplyMigration parses and applies all DDL statements from a single migration file to
// the catalogue.
//
// Takes filename (string) which identifies the migration file for diagnostic messages.
// Takes content ([]byte) which holds the raw SQL content of the migration.
// Takes migrationIndex (int) which specifies the ordinal position of this migration in
// the sequence.
//
// Returns []querier_dto.SourceError which holds any source-mapped diagnostics from
// parsing or applying the migration.
func (b *catalogueBuilder) ApplyMigration(
	ctx context.Context,
	filename string,
	content []byte,
	migrationIndex int,
) []querier_dto.SourceError {
	ctx, span, _ := log.Span(ctx, "CatalogueBuilder.ApplyMigration")
	defer span.End()

	stripped := stripDownMigration(content)
	strippedContent := string(stripped)

	commentPrefix := b.engine.CommentStyle().LinePrefix
	readOnlyOverrides := scanMigrationReadOnlyOverrides(strippedContent, commentPrefix)
	columnOverrides := scanMigrationColumnOverrides(strippedContent, commentPrefix)

	statements, parseError := b.engine.ParseStatements(strippedContent)
	if parseError != nil {
		return []querier_dto.SourceError{
			{
				Filename: filename,
				Line:     1,
				Column:   1,
				Message:  fmt.Errorf("failed to parse migration: %w", parseError).Error(),
				Severity: querier_dto.SeverityError,
				Code:     querier_dto.CodeParseError,
			},
		}
	}

	origin := querier_dto.MigrationOrigin{
		Filename: filename,
		Index:    migrationIndex,
	}

	diagnostics := b.applyStatements(ctx, statements, origin, readOnlyOverrides, filename)
	diagnostics = append(diagnostics, b.applyMigrationColumnOverrides(columnOverrides, filename)...)
	diagnostics = append(diagnostics, migrationConsistencyWarnings(stripped, strippedContent, commentPrefix, filename, len(statements))...)

	return diagnostics
}

// migrationConsistencyWarnings returns advisory diagnostics for a migration file: a Q044
// for each misplaced piko.query header (ignored in migrations), and a Q043 when a non-
// transactional statement is mixed with other statements (the whole migration then runs
// without a transaction and is not atomic).
//
// Takes stripped ([]byte) and strippedContent (string) which are the down-stripped
// migration content, commentPrefix (string) for the engine's line comment, filename
// (string) for source mapping, and statementCount (int) the number of parsed statements.
//
// Returns []querier_dto.SourceError which holds the warnings, or nil when there are none.
func migrationConsistencyWarnings(
	stripped []byte,
	strippedContent, commentPrefix, filename string,
	statementCount int,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, lineNumber := range scanMisplacedQueryDirectives(strippedContent, commentPrefix) {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: filename,
			Line:     lineNumber,
			Column:   1,
			Message: "piko.query is a query directive and is ignored in a migration file; use " +
				"piko.migration for migration directives",
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeDirectiveWrongContext,
		})
	}

	if hasNoTransactionDirective(stripped) && statementCount > 1 {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: filename,
			Line:     1,
			Column:   1,
			Message: "migration mixes a non-transactional statement (piko.migration(no_transaction: true) " +
				"or an auto-detected statement such as CREATE INDEX CONCURRENTLY) with other statements; " +
				"the whole migration runs without a transaction and is not atomic, so move the " +
				"non-transactional statement into its own migration file",
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeMigrationMixedTransaction,
		})
	}

	return diagnostics
}

// collectKnownTableColumns walks every schema's tables in the catalogue and produces the
// full list of qualified table.column references in lowercase form. The result serves as
// the Levenshtein candidate set for Q037.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schemas to walk.
//
// Returns []string which is the sorted list of lowercase table.column references.
func collectKnownTableColumns(catalogue *querier_dto.Catalogue) []string {
	if catalogue == nil {
		return nil
	}
	var known []string
	for _, schema := range catalogue.Schemas {
		if schema == nil {
			continue
		}
		for _, table := range schema.Tables {
			if table == nil {
				continue
			}
			for columnIndex := range table.Columns {
				known = append(known, strings.ToLower(table.Name+"."+table.Columns[columnIndex].Name))
			}
		}
	}

	slices.Sort(known)
	return known
}

// findCatalogueColumn walks the catalogue's schemas to find the column matching the given
// (table, column) reference case-insensitively.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schemas to search.
// Takes tableName (string) which is the table to match case-insensitively.
// Takes columnName (string) which is the column to match case-insensitively.
//
// Returns *querier_dto.Column which is the matching column, or nil when no match exists.
func findCatalogueColumn(catalogue *querier_dto.Catalogue, tableName, columnName string) *querier_dto.Column {
	if catalogue == nil {
		return nil
	}

	for _, schemaName := range sortedKeys(catalogue.Schemas) {
		if column := findColumnInSchema(catalogue.Schemas[schemaName], tableName, columnName); column != nil {
			return column
		}
	}
	return nil
}

// findColumnInSchema is the per-schema slice of findCatalogueColumn.
//
// The helper is extracted so the outer loop stays flat and the cognitive-complexity
// budget is respected.
//
// Takes schema (*querier_dto.Schema) which is the schema to search.
// Takes tableName (string) which is the table to match case-insensitively.
// Takes columnName (string) which is the column to match case-insensitively.
//
// Returns *querier_dto.Column which is the matching column, or nil when the schema is
// unset or no matching table holds the column.
func findColumnInSchema(schema *querier_dto.Schema, tableName, columnName string) *querier_dto.Column {
	if schema == nil {
		return nil
	}
	for _, table := range schema.Tables {
		if column := findCatalogueColumnInTable(table, tableName, columnName); column != nil {
			return column
		}
	}
	return nil
}

// findCatalogueColumnInTable returns the column matching columnName when table itself
// matches tableName case-insensitively. The pointer it returns aliases the underlying
// slice element so callers can mutate the catalogue in place.
//
// Takes table (*querier_dto.Table) which is the candidate table to inspect.
// Takes tableName (string) which is the table name to match case-insensitively.
// Takes columnName (string) which is the column to match case-insensitively.
//
// Returns *querier_dto.Column which aliases the matching column, or nil when no match.
func findCatalogueColumnInTable(table *querier_dto.Table, tableName, columnName string) *querier_dto.Column {
	if table == nil || !strings.EqualFold(table.Name, tableName) {
		return nil
	}
	for columnIndex := range table.Columns {
		if strings.EqualFold(table.Columns[columnIndex].Name, columnName) {
			return &table.Columns[columnIndex]
		}
	}
	return nil
}

// Catalogue returns the built catalogue.
//
// Returns *querier_dto.Catalogue which holds the accumulated schema state.
func (b *catalogueBuilder) Catalogue() *querier_dto.Catalogue {
	return b.catalogue
}

// applyMigrationColumnOverrides walks the parsed column-override directives and applies
// each to its target Column in the catalogue. Unknown table.column references produce
// Q037 diagnostics with a Levenshtein suggestion drawn from the catalogue's known
// table.column references.
//
// Takes overrides ([]migrationColumnOverride) which are the parsed column overrides.
// Takes filename (string) which identifies the file for diagnostic messages.
//
// Returns []querier_dto.SourceError which holds any override diagnostics, nil when none.
func (b *catalogueBuilder) applyMigrationColumnOverrides(overrides []migrationColumnOverride, filename string) []querier_dto.SourceError {
	if len(overrides) == 0 {
		return nil
	}

	knownColumns := collectKnownTableColumns(b.catalogue)
	errorBuilder := querier_dto.NewErrorBuilder(filename)

	var diagnostics []querier_dto.SourceError
	for _, override := range overrides {
		column := findCatalogueColumn(b.catalogue, override.Table, override.Column)
		if column == nil {
			diagnostics = append(diagnostics, errorBuilder.UnknownOverrideMigrationColumn(
				querier_dto.TextSpan{Line: 1, Column: 1, EndLine: 1, EndColumn: 1},
				override.Table+"."+override.Column,
				knownColumns,
			))
			continue
		}

		if override.SQLType != "" {
			column.SQLTypeOverride = override.SQLType
		}
		if override.GoType != "" {
			lastDot := strings.LastIndex(override.GoType, ".")
			switch {
			case lastDot < 0:
				column.GoTypeOverride = &querier_dto.GoType{Name: override.GoType}
			case lastDot > 0 && lastDot < len(override.GoType)-1:
				column.GoTypeOverride = &querier_dto.GoType{
					Package: override.GoType[:lastDot],
					Name:    override.GoType[lastDot+1:],
				}
			default:
				diagnostics = append(diagnostics, querier_dto.SourceError{
					Filename: filename,
					Line:     1,
					Column:   1,
					Message: fmt.Sprintf(
						"migration go_type override %q for %s.%s is malformed (a leading or trailing dot is not a valid qualified type)",
						override.GoType, override.Table, override.Column,
					),
					Severity: querier_dto.SeverityWarning,
					Code:     querier_dto.CodeUnknownOverrideMigrationColumn,
				})
			}
		}
		if override.Nullable != nil {
			column.NullableOverride = override.Nullable
		}
	}
	return diagnostics
}

// applyStatements iterates over parsed DDL statements, interprets each one via the
// engine, and applies the resulting mutations to the catalogue.
//
// Takes statements ([]querier_dto.ParsedStatement) which holds the parsed DDL statements.
// Takes origin (querier_dto.MigrationOrigin) which identifies the source migration file.
// Takes readOnlyOverrides (map[string]*bool) which holds per-table read-only overrides
// from migration comments.
// Takes filename (string) which identifies the file for diagnostic messages.
//
// Returns []querier_dto.SourceError which holds any diagnostics encountered during
// interpretation or application.
func (b *catalogueBuilder) applyStatements(
	ctx context.Context,
	statements []querier_dto.ParsedStatement,
	origin querier_dto.MigrationOrigin,
	readOnlyOverrides map[string]*bool,
	filename string,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, statement := range statements {
		if ctx.Err() != nil {
			return diagnostics
		}

		mutation, ddlError := b.engine.ApplyDDL(ctx, statement)
		if ddlError != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: filename,
				Line:     1,
				Column:   1,
				Message:  fmt.Errorf("failed to interpret DDL: %w", ddlError).Error(),
				Severity: querier_dto.SeverityError,
				Code:     querier_dto.CodeParseError,
			})
			continue
		}

		if mutation == nil {
			continue
		}

		mutations := append([]*querier_dto.CatalogueMutation{mutation}, mutation.AdditionalMutations...)
		mutation.AdditionalMutations = nil
		for _, next := range mutations {
			if ctx.Err() != nil {
				return diagnostics
			}
			next.Origin = origin
			applyMigrationReadOnlyOverride(next, readOnlyOverrides)

			if mutationError := b.applyMutation(ctx, next); mutationError != nil {
				diagnostics = append(diagnostics, querier_dto.SourceError{
					Filename: filename,
					Line:     1,
					Column:   1,
					Message:  mutationError.Error(),
					Severity: querier_dto.SeverityError,
					Code:     querier_dto.CodeParseError,
				})
			}
		}
	}

	return diagnostics
}

var (
	// mutationHandlers maps DDL mutation kinds to their catalogue builder handler functions.
	// Mutation kinds intentionally absent from this table fall through to the explicit
	// switch in applyMutation (MutationCreateFunction, MutationCreateView) or are silently
	// ignored as no-ops via catalogueMutationNoOp; see noOpMutations for the deliberate
	// list.
	mutationHandlers [querier_dto.MutationKindCount]func(*catalogueBuilder, *querier_dto.CatalogueMutation) error

	// noOpMutations enumerates the mutation kinds the catalogue builder deliberately
	// ignores.
	//
	// These kinds do not affect the catalogue-level schema state used for query analysis.
	// CreateTrigger and DropTrigger fire at runtime and the analyser never inspects them.
	// AsyncDataUpdate and AsyncDataDelete are data-plane operations that do not touch the
	// catalogue's schema. CreateDictionary and DropDictionary cover ClickHouse dictionaries,
	// which are opaque lookup tables accessed via dictGet() whose column structure the
	// catalogue does not need. ExchangeTables is an atomic ClickHouse table swap, a runtime
	// operation that does not change either table's schema. The ClickHouse projection,
	// data-skipping index, statistics, partition, refresh, RBAC, attach, backup, and kill
	// kinds are runtime-only or RBAC operations that do not affect the catalogue's column
	// structure used for query analysis.
	noOpMutations = [...]querier_dto.MutationKind{
		querier_dto.MutationCreateTrigger,
		querier_dto.MutationDropTrigger,
		querier_dto.MutationAsyncDataUpdate,
		querier_dto.MutationAsyncDataDelete,
		querier_dto.MutationCreateDictionary,
		querier_dto.MutationDropDictionary,
		querier_dto.MutationExchangeTables,
		querier_dto.MutationAlterTableAddProjection,
		querier_dto.MutationAlterTableDropProjection,
		querier_dto.MutationAlterTableMaterializeProjection,
		querier_dto.MutationAlterTableAddSkippingIndex,
		querier_dto.MutationAlterTableDropSkippingIndex,
		querier_dto.MutationAlterTableMaterializeIndex,
		querier_dto.MutationAlterTableAddStatistics,
		querier_dto.MutationAlterTableDropStatistics,
		querier_dto.MutationAlterTableMaterializeStatistics,
		querier_dto.MutationAlterTableModifyStatistics,
		querier_dto.MutationAlterTableMaterializeColumn,
		querier_dto.MutationAlterTableModifyColumn,
		querier_dto.MutationAlterTableModifyQuery,
		querier_dto.MutationAlterTableModifyRefresh,
		querier_dto.MutationAlterTablePartition,
		querier_dto.MutationCreateUser,
		querier_dto.MutationAlterUser,
		querier_dto.MutationDropUser,
		querier_dto.MutationCreateRole,
		querier_dto.MutationAlterRole,
		querier_dto.MutationDropRole,
		querier_dto.MutationCreatePolicy,
		querier_dto.MutationAlterPolicy,
		querier_dto.MutationDropPolicy,
		querier_dto.MutationCreateQuota,
		querier_dto.MutationAlterQuota,
		querier_dto.MutationDropQuota,
		querier_dto.MutationCreateSettingsProfile,
		querier_dto.MutationAlterSettingsProfile,
		querier_dto.MutationDropSettingsProfile,
		querier_dto.MutationGrantManagement,
		querier_dto.MutationAttachTable,
		querier_dto.MutationDetachTable,
		querier_dto.MutationBackup,
		querier_dto.MutationRestore,
		querier_dto.MutationKillQuery,
		querier_dto.MutationKillMutation,
	}
)

// catalogueMutationNoOp is the shared handler for mutation kinds that the catalogue
// builder deliberately ignores. Centralising the function pointer keeps mutationHandlers
// compact and makes the intent of the no-op entries obvious at the call site.
//
// Takes builder (*catalogueBuilder) which is ignored by the no-op handler.
// Takes mutation (*querier_dto.CatalogueMutation) which is ignored by the no-op handler.
//
// Returns error which is always nil.
func catalogueMutationNoOp(*catalogueBuilder, *querier_dto.CatalogueMutation) error {
	return nil
}

func init() {
	mutationHandlers = [querier_dto.MutationKindCount]func(*catalogueBuilder, *querier_dto.CatalogueMutation) error{
		querier_dto.MutationCreateTable:              (*catalogueBuilder).applyCreateTable,
		querier_dto.MutationDropTable:                (*catalogueBuilder).applyDropTable,
		querier_dto.MutationAlterTableAddColumn:      (*catalogueBuilder).applyAlterTableAddColumn,
		querier_dto.MutationAlterTableDropColumn:     (*catalogueBuilder).applyAlterTableDropColumn,
		querier_dto.MutationAlterTableAlterColumn:    (*catalogueBuilder).applyAlterTableAlterColumn,
		querier_dto.MutationAlterTableRenameColumn:   (*catalogueBuilder).applyAlterTableRenameColumn,
		querier_dto.MutationAlterTableRenameTable:    (*catalogueBuilder).applyAlterTableRenameTable,
		querier_dto.MutationAlterTableSetSchema:      (*catalogueBuilder).applyAlterTableSetSchema,
		querier_dto.MutationCreateEnum:               (*catalogueBuilder).applyCreateEnum,
		querier_dto.MutationAlterEnumAddValue:        (*catalogueBuilder).applyAlterEnumAddValue,
		querier_dto.MutationAlterEnumRenameValue:     (*catalogueBuilder).applyAlterEnumRenameValue,
		querier_dto.MutationDropEnum:                 (*catalogueBuilder).applyDropEnum,
		querier_dto.MutationCreateCompositeType:      (*catalogueBuilder).applyCreateCompositeType,
		querier_dto.MutationDropType:                 (*catalogueBuilder).applyDropType,
		querier_dto.MutationDropFunction:             (*catalogueBuilder).applyDropFunction,
		querier_dto.MutationCreateSchema:             (*catalogueBuilder).applyCreateSchema,
		querier_dto.MutationDropSchema:               (*catalogueBuilder).applyDropSchema,
		querier_dto.MutationDropView:                 (*catalogueBuilder).applyDropView,
		querier_dto.MutationCreateIndex:              (*catalogueBuilder).applyCreateIndex,
		querier_dto.MutationDropIndex:                (*catalogueBuilder).applyDropIndex,
		querier_dto.MutationCreateExtension:          (*catalogueBuilder).applyCreateExtension,
		querier_dto.MutationComment:                  (*catalogueBuilder).applyComment,
		querier_dto.MutationAlterTableAddConstraint:  (*catalogueBuilder).applyAlterTableAddConstraint,
		querier_dto.MutationAlterTableDropConstraint: (*catalogueBuilder).applyAlterTableDropConstraint,
		querier_dto.MutationCreateSequence:           (*catalogueBuilder).applyCreateSequence,
		querier_dto.MutationDropSequence:             (*catalogueBuilder).applyDropSequence,
	}
	for _, kind := range noOpMutations {
		mutationHandlers[kind] = catalogueMutationNoOp
	}
}

// applyMutation dispatches a single catalogue mutation to the appropriate handler based
// on its kind.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the mutation to apply.
//
// Returns error when the mutation handler fails or the kind is unknown.
func (b *catalogueBuilder) applyMutation(ctx context.Context, mutation *querier_dto.CatalogueMutation) error {
	switch mutation.Kind {
	case querier_dto.MutationCreateFunction:
		return b.applyCreateFunction(ctx, mutation)
	case querier_dto.MutationCreateView:
		return b.applyCreateView(ctx, mutation)
	default:
		if int(mutation.Kind) < len(mutationHandlers) && mutationHandlers[mutation.Kind] != nil {
			return mutationHandlers[mutation.Kind](b, mutation)
		}
		return fmt.Errorf("unknown mutation kind: %d", mutation.Kind)
	}
}

// resolveCustomColumnType resolves a column's type against known enums and composite
// types when the type category is unknown.
//
// Takes column (*querier_dto.Column) which holds the column whose type to resolve.
func (b *catalogueBuilder) resolveCustomColumnType(column *querier_dto.Column) {
	if column.SQLType.Category != querier_dto.TypeCategoryUnknown {
		return
	}
	typeName := column.SQLType.EngineName
	if typeName == "" {
		return
	}

	for _, schemaName := range sortedKeys(b.catalogue.Schemas) {
		schema := b.catalogue.Schemas[schemaName]
		if enum, exists := schema.Enums[typeName]; exists {
			column.SQLType.Category = querier_dto.TypeCategoryEnum
			column.SQLType.EnumValues = enum.Values
			column.SQLType.Schema = enum.Schema
			return
		}
		if _, exists := schema.CompositeTypes[typeName]; exists {
			column.SQLType.Category = querier_dto.TypeCategoryComposite
			column.SQLType.Schema = schema.Name
			return
		}
	}
}

// resolveSchema returns the schema for the given name, creating it if it does not exist.
// An empty name is treated as the default schema.
//
// Takes schemaName (string) which specifies the schema name to resolve.
//
// Returns *querier_dto.Schema which is the resolved or newly created schema.
func (b *catalogueBuilder) resolveSchema(schemaName string) *querier_dto.Schema {
	if schemaName == "" {
		schemaName = b.catalogue.DefaultSchema
	}
	schema, exists := b.catalogue.Schemas[schemaName]
	if !exists {
		schema = newEmptySchema(schemaName)
		b.catalogue.Schemas[schemaName] = schema
	}
	return schema
}

// lookupSchema returns the schema for the given name without creating it.
//
// An empty name is treated as the default schema. The lookup is read-only, so a failed
// search does not leave a phantom empty schema behind in the catalogue.
//
// Takes schemaName (string) which specifies the schema name to look up.
//
// Returns *querier_dto.Schema which is the matching schema, or nil when absent.
// Returns string which is the resolved schema name (default schema substituted for
// empty).
func (b *catalogueBuilder) lookupSchema(schemaName string) (*querier_dto.Schema, string) {
	if schemaName == "" {
		schemaName = b.catalogue.DefaultSchema
	}
	return b.catalogue.Schemas[schemaName], schemaName
}

// findTable looks up a table by schema and name, returning an error if not found. The
// lookup is read-only and never creates a schema.
//
// Takes schemaName (string) which specifies the schema to search in.
// Takes tableName (string) which specifies the table to find.
//
// Returns *querier_dto.Table which is the found table.
// Returns error when the table does not exist.
func (b *catalogueBuilder) findTable(schemaName string, tableName string) (*querier_dto.Table, error) {
	schema, resolvedName := b.lookupSchema(schemaName)
	if schema == nil {
		return nil, fmt.Errorf("table %s not found in schema %s", tableName, resolvedName)
	}
	table, exists := schema.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table %s not found in schema %s", tableName, resolvedName)
	}
	return table, nil
}

// newEmptySchema creates a new schema with all map fields initialised to empty maps.
//
// Takes name (string) which specifies the schema name.
//
// Returns *querier_dto.Schema which is the initialised empty schema.
func newEmptySchema(name string) *querier_dto.Schema {
	return &querier_dto.Schema{
		Name:           name,
		Tables:         make(map[string]*querier_dto.Table),
		Views:          make(map[string]*querier_dto.View),
		Enums:          make(map[string]*querier_dto.Enum),
		Functions:      make(map[string][]*querier_dto.FunctionSignature),
		CompositeTypes: make(map[string]*querier_dto.CompositeType),
		Sequences:      make(map[string]*querier_dto.Sequence),
	}
}
