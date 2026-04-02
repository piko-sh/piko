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
	"errors"
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// catalogueBuilder constructs a Catalogue by replaying DDL mutations from
// migration files. Each migration file is parsed via the engine adapter and
// the resulting mutations are applied sequentially to build up the schema
// state.
type catalogueBuilder struct {
	// engine holds the database engine adapter used for parsing and DDL
	// interpretation.
	engine EnginePort

	// catalogue holds the schema state being built up by replaying mutations.
	catalogue *querier_dto.Catalogue
}

// newCatalogueBuilder creates a new catalogue builder with an empty catalogue.
// The default schema is set based on the engine dialect.
//
// Takes engine (EnginePort) which provides the database dialect and DDL
// interpretation.
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

// ApplyMigration parses and applies all DDL statements from a single migration
// file to the catalogue.
//
// Takes filename (string) which identifies the migration file for diagnostic
// messages.
// Takes content ([]byte) which holds the raw SQL content of the migration.
// Takes migrationIndex (int) which specifies the ordinal position of this
// migration in the sequence.
//
// Returns []querier_dto.SourceError which holds any source-mapped diagnostics
// from parsing or applying the migration.
func (b *catalogueBuilder) ApplyMigration(
	ctx context.Context,
	filename string,
	content []byte,
	migrationIndex int,
) []querier_dto.SourceError {
	_, span, _ := log.Span(ctx, "CatalogueBuilder.ApplyMigration")
	defer span.End()

	stripped := stripDownMigration(content)
	strippedContent := string(stripped)

	readOnlyOverrides := scanMigrationReadOnlyOverrides(
		strippedContent, b.engine.CommentStyle().LinePrefix,
	)

	statements, parseError := b.engine.ParseStatements(strippedContent)
	if parseError != nil {
		return []querier_dto.SourceError{
			{
				Filename: filename,
				Line:     1,
				Column:   1,
				Message:  fmt.Sprintf("failed to parse migration: %s", parseError.Error()),
				Severity: querier_dto.SeverityError,
				Code:     querier_dto.CodeParseError,
			},
		}
	}

	origin := querier_dto.MigrationOrigin{
		Filename: filename,
		Index:    migrationIndex,
	}

	return b.applyStatements(ctx, statements, origin, readOnlyOverrides, filename)
}

// Catalogue returns the built catalogue.
//
// Returns *querier_dto.Catalogue which holds the accumulated schema state.
func (b *catalogueBuilder) Catalogue() *querier_dto.Catalogue {
	return b.catalogue
}

// applyStatements iterates over parsed DDL statements, interprets each one
// via the engine, and applies the resulting mutations to the catalogue.
//
// Takes statements ([]querier_dto.ParsedStatement) which holds the parsed DDL
// statements.
// Takes origin (querier_dto.MigrationOrigin) which identifies the source
// migration file.
// Takes readOnlyOverrides (map[string]*bool) which holds per-table read-only
// overrides from migration comments.
// Takes filename (string) which identifies the file for diagnostic messages.
//
// Returns []querier_dto.SourceError which holds any diagnostics encountered
// during interpretation or application.
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

		mutation, ddlError := b.engine.ApplyDDL(statement)
		if ddlError != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: filename,
				Line:     1,
				Column:   1,
				Message:  fmt.Sprintf("failed to interpret DDL: %s", ddlError.Error()),
				Severity: querier_dto.SeverityError,
				Code:     querier_dto.CodeParseError,
			})
			continue
		}

		if mutation == nil {
			continue
		}

		mutation.Origin = origin
		applyMigrationReadOnlyOverride(mutation, readOnlyOverrides)

		if mutationError := b.applyMutation(ctx, mutation); mutationError != nil {
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

	return diagnostics
}

var mutationHandlers [querier_dto.MutationKindCount]func(*catalogueBuilder, *querier_dto.CatalogueMutation) error

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
		querier_dto.MutationCreateTrigger:            func(*catalogueBuilder, *querier_dto.CatalogueMutation) error { return nil },
		querier_dto.MutationDropTrigger:              func(*catalogueBuilder, *querier_dto.CatalogueMutation) error { return nil },
		querier_dto.MutationAlterTableAddConstraint:  (*catalogueBuilder).applyAlterTableAddConstraint,
		querier_dto.MutationAlterTableDropConstraint: (*catalogueBuilder).applyAlterTableDropConstraint,
		querier_dto.MutationCreateSequence:           (*catalogueBuilder).applyCreateSequence,
		querier_dto.MutationDropSequence:             (*catalogueBuilder).applyDropSequence,
	}
}

// applyMutation dispatches a single catalogue mutation to the appropriate
// handler based on its kind.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the mutation to
// apply.
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

// applyCreateTable adds a new table to the catalogue, inheriting columns from
// parent tables and resolving custom column types.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the table
// definition.
//
// Returns error when the table already exists.
func (b *catalogueBuilder) applyCreateTable(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	if _, exists := schema.Tables[mutation.TableName]; exists {
		return fmt.Errorf("table %s.%s already exists", schema.Name, mutation.TableName)
	}

	var inherited []querier_dto.Column
	for _, parent := range mutation.InheritsTables {
		parentTable, findError := b.findTable(parent.Schema, parent.Name)
		if findError != nil {
			continue
		}
		inherited = append(inherited, parentTable.Columns...)
	}

	childColumnNames := make(map[string]struct{}, len(mutation.Columns))
	for i := range mutation.Columns {
		childColumnNames[mutation.Columns[i].Name] = struct{}{}
	}

	var columns []querier_dto.Column
	for i := range inherited {
		if _, overridden := childColumnNames[inherited[i].Name]; !overridden {
			columns = append(columns, inherited[i])
		}
	}
	columns = append(columns, mutation.Columns...)

	for i := range columns {
		columns[i].Origin = mutation.Origin
		b.resolveCustomColumnType(&columns[i])
	}
	for i := range mutation.Constraints {
		mutation.Constraints[i].Origin = mutation.Origin
	}
	schema.Tables[mutation.TableName] = &querier_dto.Table{
		Name:              mutation.TableName,
		Schema:            schema.Name,
		Columns:           columns,
		PrimaryKey:        mutation.PrimaryKey,
		Constraints:       mutation.Constraints,
		IsVirtual:         mutation.IsVirtual,
		VirtualModuleName: mutation.VirtualModuleName,
		IsWithoutRowID:    mutation.IsWithoutRowID,
		Origin:            mutation.Origin,
	}
	return nil
}

// applyDropTable removes a table from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the table
// to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropTable(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.Tables, mutation.TableName)
	return nil
}

// applyAlterTableAddColumn appends new columns to an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the columns to
// add.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableAddColumn(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	if len(mutation.Columns) > 0 {
		for i := range mutation.Columns {
			mutation.Columns[i].Origin = mutation.Origin
			b.resolveCustomColumnType(&mutation.Columns[i])
		}
		table.Columns = append(table.Columns, mutation.Columns...)
	}
	return nil
}

// applyAlterTableDropColumn removes a column from an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the column
// to drop.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableDropColumn(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	filtered := make([]querier_dto.Column, 0, len(table.Columns))
	for i := range table.Columns {
		if table.Columns[i].Name != mutation.ColumnName {
			filtered = append(filtered, table.Columns[i])
		}
	}
	table.Columns = filtered
	return nil
}

// applyAlterTableAlterColumn replaces a column definition in an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the new column
// definition.
//
// Returns error when the target table or column is not found.
func (b *catalogueBuilder) applyAlterTableAlterColumn(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	if len(mutation.Columns) == 0 {
		return nil
	}
	altered := mutation.Columns[0]
	for i := range table.Columns {
		if table.Columns[i].Name == mutation.ColumnName {
			altered.Origin = table.Columns[i].Origin
			table.Columns[i] = altered
			return nil
		}
	}
	return fmt.Errorf("column %s not found in table %s", mutation.ColumnName, mutation.TableName)
}

// applyAlterTableRenameColumn renames a column in an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the old and new
// column names.
//
// Returns error when the target table or column is not found.
func (b *catalogueBuilder) applyAlterTableRenameColumn(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	for i := range table.Columns {
		if table.Columns[i].Name == mutation.ColumnName {
			table.Columns[i].Name = mutation.NewName
			return nil
		}
	}
	return fmt.Errorf("column %s not found in table %s", mutation.ColumnName, mutation.TableName)
}

// applyAlterTableRenameTable renames a table within its schema.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the old and new
// table names.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableRenameTable(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	table, exists := schema.Tables[mutation.TableName]
	if !exists {
		return fmt.Errorf("table %s not found in schema %s", mutation.TableName, schema.Name)
	}
	delete(schema.Tables, mutation.TableName)
	table.Name = mutation.NewName
	schema.Tables[mutation.NewName] = table
	return nil
}

// applyAlterTableSetSchema moves a table from one schema to another.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the source schema
// and target schema names.
//
// Returns error when the target table is not found in the source schema.
func (b *catalogueBuilder) applyAlterTableSetSchema(mutation *querier_dto.CatalogueMutation) error {
	sourceSchema := b.resolveSchema(mutation.SchemaName)
	table, exists := sourceSchema.Tables[mutation.TableName]
	if !exists {
		return fmt.Errorf("table %s not found in schema %s", mutation.TableName, sourceSchema.Name)
	}
	delete(sourceSchema.Tables, mutation.TableName)

	targetSchema := b.resolveSchema(mutation.NewName)
	table.Schema = targetSchema.Name
	targetSchema.Tables[mutation.TableName] = table
	return nil
}

// applyCreateSequence adds a new sequence to the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the sequence
// definition.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateSequence(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	schema.Sequences[mutation.SequenceName] = &querier_dto.Sequence{
		Name:          mutation.SequenceName,
		Schema:        schema.Name,
		OwnedByTable:  mutation.OwnedByTable,
		OwnedByColumn: mutation.OwnedByColumn,
		Origin:        mutation.Origin,
	}
	return nil
}

// applyDropSequence removes a sequence from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the
// sequence to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropSequence(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.Sequences, mutation.SequenceName)
	return nil
}

// applyAlterTableAddConstraint appends new constraints to an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the constraints
// to add.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableAddConstraint(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	for i := range mutation.Constraints {
		mutation.Constraints[i].Origin = mutation.Origin
	}
	table.Constraints = append(table.Constraints, mutation.Constraints...)
	return nil
}

// applyAlterTableDropConstraint removes a named constraint from an existing
// table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the
// constraint to drop.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableDropConstraint(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	filtered := table.Constraints[:0]
	for _, constraint := range table.Constraints {
		if constraint.Name != mutation.ConstraintName {
			filtered = append(filtered, constraint)
		}
	}
	table.Constraints = filtered
	return nil
}

// applyCreateEnum adds a new enum type to the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the enum name
// and values.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateEnum(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	schema.Enums[mutation.EnumName] = &querier_dto.Enum{
		Name:   mutation.EnumName,
		Schema: schema.Name,
		Values: mutation.EnumValues,
		Origin: mutation.Origin,
	}
	return nil
}

// applyAlterEnumAddValue appends new values to an existing enum type.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the values to
// add.
//
// Returns error when the enum is not found.
func (b *catalogueBuilder) applyAlterEnumAddValue(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	enum, exists := schema.Enums[mutation.EnumName]
	if !exists {
		return fmt.Errorf("enum %s not found in schema %s", mutation.EnumName, schema.Name)
	}
	enum.Values = append(enum.Values, mutation.EnumValues...)
	return nil
}

// applyAlterEnumRenameValue renames a value within an existing enum type.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the old and new
// enum value names.
//
// Returns error when the enum is not found.
func (b *catalogueBuilder) applyAlterEnumRenameValue(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	enum, exists := schema.Enums[mutation.EnumName]
	if !exists {
		return fmt.Errorf("enum %s not found in schema %s", mutation.EnumName, schema.Name)
	}
	if len(mutation.EnumValues) >= 2 {
		oldValue := mutation.EnumValues[0]
		newValue := mutation.EnumValues[1]
		for i, value := range enum.Values {
			if value == oldValue {
				enum.Values[i] = newValue
				return nil
			}
		}
	}
	return nil
}

// applyDropEnum removes an enum type from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the enum
// to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropEnum(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.Enums, mutation.EnumName)
	return nil
}

// applyCreateCompositeType adds a new composite type to the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the composite
// type definition.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateCompositeType(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	fields := mutation.Columns
	for i := range fields {
		fields[i].Origin = mutation.Origin
	}
	schema.CompositeTypes[mutation.EnumName] = &querier_dto.CompositeType{
		Name:   mutation.EnumName,
		Schema: schema.Name,
		Fields: fields,
		Origin: mutation.Origin,
	}
	return nil
}

// applyDropType removes a composite type or enum from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the type
// to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropType(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.CompositeTypes, mutation.EnumName)
	delete(schema.Enums, mutation.EnumName)
	return nil
}

// applyCreateFunction adds or replaces a function signature in the catalogue,
// resolving the function body for SQL-language functions.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the function
// signature and body.
//
// Returns error when the mutation is missing a function signature.
func (b *catalogueBuilder) applyCreateFunction(ctx context.Context, mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	if mutation.FunctionSignature == nil {
		return errors.New("CREATE FUNCTION mutation missing function signature")
	}
	mutation.FunctionSignature.Origin = mutation.Origin

	if mutation.FunctionSignature.IsStrict {
		mutation.FunctionSignature.NullableBehaviour = querier_dto.FunctionNullableReturnsNullOnNull
	}

	b.resolveFunctionBody(ctx, mutation.FunctionSignature)

	functionName := mutation.FunctionSignature.Name
	existing := schema.Functions[functionName]

	for i, overload := range existing {
		if argumentTypesMatch(overload.Arguments, mutation.FunctionSignature.Arguments) {
			schema.Functions[functionName][i] = mutation.FunctionSignature
			return nil
		}
	}

	schema.Functions[functionName] = append(schema.Functions[functionName], mutation.FunctionSignature)
	return nil
}

// resolveFunctionBody analyses the function body to determine data access
// level and return type. SQL-language functions are fully parsed and analysed;
// other languages are scanned for DML keywords.
//
// Takes signature (*querier_dto.FunctionSignature) which holds the function
// body and metadata to populate.
func (b *catalogueBuilder) resolveFunctionBody(ctx context.Context, signature *querier_dto.FunctionSignature) {
	if signature.BodySQL == "" {
		return
	}

	if strings.EqualFold(signature.Language, "sql") {
		b.analyseSQLFunctionBody(ctx, signature)
		return
	}

	if signature.DataAccess == querier_dto.DataAccessUnknown {
		signature.DataAccess = scanBodyForDML(signature.BodySQL)
	}
}

// analyseSQLFunctionBody parses and analyses a SQL-language function body to
// determine called functions, data access level, and return type.
//
// Takes signature (*querier_dto.FunctionSignature) which holds the SQL body
// and receives the analysis results.
func (b *catalogueBuilder) analyseSQLFunctionBody(ctx context.Context, signature *querier_dto.FunctionSignature) {
	statements, parseError := b.engine.ParseStatements(signature.BodySQL)
	if parseError != nil || len(statements) == 0 {
		return
	}

	primaryStatement := statements[len(statements)-1]
	rawAnalysis, analysisError := b.engine.AnalyseQuery(b.catalogue, primaryStatement)
	if analysisError != nil || rawAnalysis == nil {
		return
	}

	signature.CalledFunctions = collectFunctionCalls(rawAnalysis)

	if signature.DataAccess == querier_dto.DataAccessUnknown {
		if rawAnalysis.ReadOnly {
			signature.DataAccess = querier_dto.DataAccessReadOnly
		} else {
			signature.DataAccess = querier_dto.DataAccessModifiesData
		}
	}

	if signature.ReturnType.Category != querier_dto.TypeCategoryUnknown || signature.ReturnsSet {
		return
	}

	analyser := newQueryAnalyser(b.engine, b.catalogue)
	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

	_ = analyser.resolveCTEs(ctx, rawAnalysis.CTEDefinitions, scope)
	_ = analyser.buildScopeChain(rawAnalysis, scope)
	_ = analyser.resolveTableValuedFunctions(rawAnalysis.RawTableValuedFunctions, scope)
	_ = analyser.resolveRawDerivedTables(ctx, rawAnalysis.RawDerivedTables, scope)

	outputColumns, _, _ := analyser.typeResolver.ResolveOutputColumns(
		ctx, rawAnalysis.OutputColumns, scope,
	)

	if len(outputColumns) == 1 {
		signature.ReturnType = outputColumns[0].SQLType
	}
}

// scanBodyForDML scans a function body string for DML keywords to determine
// the data access level.
//
// Takes body (string) which holds the function body text to scan.
//
// Returns querier_dto.FunctionDataAccess which is DataAccessModifiesData if any
// DML keyword is found, or DataAccessReadOnly otherwise.
func scanBodyForDML(body string) querier_dto.FunctionDataAccess {
	upper := strings.ToUpper(body)
	dmlKeywords := [...]string{"INSERT ", "UPDATE ", "DELETE ", "TRUNCATE "}
	for _, keyword := range dmlKeywords {
		if strings.Contains(upper, keyword) {
			return querier_dto.DataAccessModifiesData
		}
	}
	return querier_dto.DataAccessReadOnly
}

// applyDropFunction removes a function or a specific overload from the
// catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the
// function to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropFunction(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	if mutation.FunctionSignature == nil {
		delete(schema.Functions, mutation.TableName)
		return nil
	}
	functionName := mutation.FunctionSignature.Name
	if len(mutation.FunctionSignature.Arguments) == 0 {
		delete(schema.Functions, functionName)
		return nil
	}
	existing := schema.Functions[functionName]
	filtered := make([]*querier_dto.FunctionSignature, 0, len(existing))
	for _, overload := range existing {
		if !argumentTypesMatch(overload.Arguments, mutation.FunctionSignature.Arguments) {
			filtered = append(filtered, overload)
		}
	}
	if len(filtered) == 0 {
		delete(schema.Functions, functionName)
	} else {
		schema.Functions[functionName] = filtered
	}
	return nil
}

// applyCreateSchema adds a new empty schema to the catalogue if it does not
// already exist.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the schema name.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateSchema(mutation *querier_dto.CatalogueMutation) error {
	if _, exists := b.catalogue.Schemas[mutation.SchemaName]; !exists {
		b.catalogue.Schemas[mutation.SchemaName] = newEmptySchema(mutation.SchemaName)
	}
	return nil
}

// applyDropSchema removes a schema and all its contents from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the schema
// to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropSchema(mutation *querier_dto.CatalogueMutation) error {
	delete(b.catalogue.Schemas, mutation.SchemaName)
	return nil
}

// applyCreateView adds a new view to the catalogue, resolving its columns
// from the view definition when available.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the view
// definition.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateView(ctx context.Context, mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)

	var columns []querier_dto.Column
	if mutation.ViewDefinition != nil {
		columns = b.resolveViewColumns(ctx, mutation.ViewDefinition)
	} else {
		columns = mutation.Columns
	}

	for i := range columns {
		columns[i].Origin = mutation.Origin
	}
	schema.Views[mutation.TableName] = &querier_dto.View{
		Name:    mutation.TableName,
		Schema:  schema.Name,
		Columns: columns,
		Origin:  mutation.Origin,
	}
	return nil
}

// resolveViewColumns analyses a view definition query to determine the output
// column names, types, and nullability.
//
// Takes definition (*querier_dto.RawQueryAnalysis) which holds the parsed view
// query.
//
// Returns []querier_dto.Column which holds the resolved view columns.
func (b *catalogueBuilder) resolveViewColumns(ctx context.Context, definition *querier_dto.RawQueryAnalysis) []querier_dto.Column {
	analyser := newQueryAnalyser(b.engine, b.catalogue)
	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

	_ = analyser.resolveCTEs(ctx, definition.CTEDefinitions, scope)
	_ = analyser.buildScopeChain(definition, scope)
	_ = analyser.resolveTableValuedFunctions(definition.RawTableValuedFunctions, scope)
	_ = analyser.resolveRawDerivedTables(ctx, definition.RawDerivedTables, scope)

	outputColumns, _, _ := analyser.typeResolver.ResolveOutputColumns(
		ctx, definition.OutputColumns, scope,
	)

	if len(definition.CompoundBranches) > 0 {
		_ = analyser.resolveCompoundBranches(ctx, definition.CompoundBranches, outputColumns)
	}

	columns := make([]querier_dto.Column, len(outputColumns))
	for i := range outputColumns {
		columns[i] = querier_dto.Column{
			Name:     outputColumns[i].Name,
			SQLType:  outputColumns[i].SQLType,
			Nullable: outputColumns[i].Nullable,
		}
	}
	return columns
}

// applyDropView removes a view from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the view
// to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropView(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.Views, mutation.TableName)
	return nil
}

// applyCreateIndex adds an index record to the target table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the index
// definition.
//
// Returns error (always nil, silently ignores missing tables).
func (b *catalogueBuilder) applyCreateIndex(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return nil
	}
	table.Indexes = append(table.Indexes, querier_dto.Index{
		Name:   mutation.NewName,
		Origin: mutation.Origin,
	})
	return nil
}

// applyDropIndex handles DROP INDEX mutations. Index tracking is not currently
// needed for query analysis, so this is a no-op.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the index
// to drop.
//
// Returns error (always nil).
func (*catalogueBuilder) applyDropIndex(_ *querier_dto.CatalogueMutation) error {
	return nil
}

// applyCreateExtension registers an extension and loads any functions it
// provides via the engine's extension loader.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the extension
// name.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateExtension(mutation *querier_dto.CatalogueMutation) error {
	b.catalogue.Extensions[mutation.TableName] = struct{}{}

	if loader, ok := b.engine.(ExtensionLoaderPort); ok {
		if functions := loader.LoadExtensionFunctions(mutation.TableName); len(functions) > 0 {
			schema := b.resolveSchema(mutation.SchemaName)
			for _, function := range functions {
				schema.Functions[function.Name] = append(schema.Functions[function.Name], function)
			}
		}
	}

	return nil
}

// applyComment handles COMMENT ON mutations. Comment tracking is not currently
// needed for query analysis, so this is a no-op.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the comment
// details.
//
// Returns error (always nil).
func (*catalogueBuilder) applyComment(_ *querier_dto.CatalogueMutation) error {
	return nil
}

// resolveCustomColumnType resolves a column's type against known enums and
// composite types when the type category is unknown.
//
// Takes column (*querier_dto.Column) which holds the column whose type to
// resolve.
func (b *catalogueBuilder) resolveCustomColumnType(column *querier_dto.Column) {
	if column.SQLType.Category != querier_dto.TypeCategoryUnknown {
		return
	}
	typeName := column.SQLType.EngineName
	if typeName == "" {
		return
	}
	for _, schema := range b.catalogue.Schemas {
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

// resolveSchema returns the schema for the given name, creating it if it does
// not exist. An empty name is treated as the default schema.
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

// findTable looks up a table by schema and name, returning an error if not
// found.
//
// Takes schemaName (string) which specifies the schema to search in.
// Takes tableName (string) which specifies the table to find.
//
// Returns *querier_dto.Table which is the found table.
// Returns error when the table does not exist.
func (b *catalogueBuilder) findTable(schemaName string, tableName string) (*querier_dto.Table, error) {
	schema := b.resolveSchema(schemaName)
	table, exists := schema.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table %s not found in schema %s", tableName, schema.Name)
	}
	return table, nil
}

// newEmptySchema creates a new schema with all map fields initialised to empty
// maps.
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

// argumentTypesMatch checks whether two function argument lists have the same
// types by comparing category and engine name.
//
// Takes left ([]querier_dto.FunctionArgument) which holds the first argument
// list.
// Takes right ([]querier_dto.FunctionArgument) which holds the second argument
// list.
//
// Returns bool which is true if the argument lists have the same length and
// matching types.
func argumentTypesMatch(left []querier_dto.FunctionArgument, right []querier_dto.FunctionArgument) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Type.Category != right[i].Type.Category {
			return false
		}
		if left[i].Type.EngineName != right[i].Type.EngineName {
			return false
		}
	}
	return true
}
