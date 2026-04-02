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

package db_catalogue_postgres

import (
	"context"
	"database/sql"
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// TypeNormaliser converts engine-specific type names to structured SQLType
// values. This is satisfied by any EnginePort implementation.
type TypeNormaliser interface {
	NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType
}

// PgIntrospectionProvider implements CatalogueProviderPort by querying
// a live PostgreSQL database using pg_catalog and information_schema.
type PgIntrospectionProvider struct {
	database *sql.DB

	typeNormaliser TypeNormaliser
}

// NewPgIntrospectionProvider creates a new PostgreSQL introspection-based
// catalogue provider.
func NewPgIntrospectionProvider(
	database *sql.DB,
	typeNormaliser TypeNormaliser,
) *PgIntrospectionProvider {
	return &PgIntrospectionProvider{
		database:       database,
		typeNormaliser: typeNormaliser,
	}
}

// BuildCatalogue introspects the PostgreSQL database and builds a schema
// catalogue covering tables, views, indexes, enums, composite types,
// functions, and extensions.
func (provider *PgIntrospectionProvider) BuildCatalogue(
	ctx context.Context,
) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
	catalogue := &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas:       make(map[string]*querier_dto.Schema),
		Extensions:    make(map[string]struct{}),
	}

	schemaNames, schemaError := provider.listSchemas(ctx)
	if schemaError != nil {
		return nil, nil, fmt.Errorf("listing schemas: %w", schemaError)
	}

	for _, schemaName := range schemaNames {
		if populateError := provider.populateSchema(ctx, catalogue, schemaName); populateError != nil {
			return nil, nil, populateError
		}
	}

	if extensionError := provider.introspectExtensions(ctx, catalogue); extensionError != nil {
		return nil, nil, fmt.Errorf("introspecting extensions: %w", extensionError)
	}

	return catalogue, nil, nil
}

func (provider *PgIntrospectionProvider) populateSchema(
	ctx context.Context,
	catalogue *querier_dto.Catalogue,
	schemaName string,
) error {
	schema := &querier_dto.Schema{
		Name:           schemaName,
		Tables:         make(map[string]*querier_dto.Table),
		Views:          make(map[string]*querier_dto.View),
		Enums:          make(map[string]*querier_dto.Enum),
		Functions:      make(map[string][]*querier_dto.FunctionSignature),
		CompositeTypes: make(map[string]*querier_dto.CompositeType),
		Sequences:      make(map[string]*querier_dto.Sequence),
	}
	catalogue.Schemas[schemaName] = schema

	if tableError := provider.introspectTables(ctx, schema); tableError != nil {
		return fmt.Errorf("introspecting tables in schema %s: %w", schemaName, tableError)
	}

	if viewError := provider.introspectViews(ctx, schema); viewError != nil {
		return fmt.Errorf("introspecting views in schema %s: %w", schemaName, viewError)
	}

	if enumError := provider.introspectEnums(ctx, schema); enumError != nil {
		return fmt.Errorf("introspecting enums in schema %s: %w", schemaName, enumError)
	}

	if compositeError := provider.introspectCompositeTypes(ctx, schema); compositeError != nil {
		return fmt.Errorf("introspecting composite types in schema %s: %w", schemaName, compositeError)
	}

	if functionError := provider.introspectFunctions(ctx, schema); functionError != nil {
		return fmt.Errorf("introspecting functions in schema %s: %w", schemaName, functionError)
	}

	return nil
}

func (provider *PgIntrospectionProvider) listSchemas(
	ctx context.Context,
) ([]string, error) {
	rows, queryError := provider.database.QueryContext(ctx,
		`SELECT schema_name FROM information_schema.schemata
		 WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		 ORDER BY schema_name`)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if scanError := rows.Scan(&name); scanError != nil {
			return nil, scanError
		}
		names = append(names, name)
	}

	return names, rows.Err()
}

func (provider *PgIntrospectionProvider) introspectTables(
	ctx context.Context,
	schema *querier_dto.Schema,
) error {
	tableNames, listError := provider.listTables(ctx, schema.Name)
	if listError != nil {
		return listError
	}

	for _, tableName := range tableNames {
		table, tableError := provider.introspectTable(ctx, schema.Name, tableName)
		if tableError != nil {
			return fmt.Errorf("introspecting table %s: %w", tableName, tableError)
		}
		schema.Tables[tableName] = table
	}

	return nil
}

func (provider *PgIntrospectionProvider) listTables(
	ctx context.Context,
	schemaName string,
) ([]string, error) {
	rows, queryError := provider.database.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = $1 AND table_type IN ('BASE TABLE', 'FOREIGN TABLE')
		 ORDER BY table_name`,
		schemaName)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if scanError := rows.Scan(&name); scanError != nil {
			return nil, scanError
		}
		names = append(names, name)
	}

	return names, rows.Err()
}

func (provider *PgIntrospectionProvider) introspectTable(
	ctx context.Context,
	schemaName string,
	tableName string,
) (*querier_dto.Table, error) {
	columns, columnError := provider.introspectColumns(ctx, schemaName, tableName)
	if columnError != nil {
		return nil, columnError
	}

	primaryKeyColumns, uniqueConstraints, constraintError := provider.introspectConstraints(ctx, schemaName, tableName)
	if constraintError != nil {
		return nil, constraintError
	}

	indexes, indexError := provider.introspectIndexes(ctx, schemaName, tableName)
	if indexError != nil {
		return nil, indexError
	}

	var constraints []querier_dto.Constraint
	for _, unique := range uniqueConstraints {
		constraints = append(constraints, querier_dto.Constraint{
			Name:    unique.Name,
			Kind:    querier_dto.ConstraintUnique,
			Columns: unique.Columns,
		})
	}

	return &querier_dto.Table{
		Name:        tableName,
		Schema:      schemaName,
		Columns:     columns,
		PrimaryKey:  primaryKeyColumns,
		Indexes:     indexes,
		Constraints: constraints,
	}, nil
}

func (provider *PgIntrospectionProvider) introspectColumns(
	ctx context.Context,
	schemaName string,
	tableName string,
) ([]querier_dto.Column, error) {
	rows, queryError := provider.queryColumns(ctx, schemaName, tableName)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	var columns []querier_dto.Column

	for rows.Next() {
		column, scanError := provider.scanColumn(rows)
		if scanError != nil {
			return nil, scanError
		}
		columns = append(columns, column)
	}

	return columns, rows.Err()
}

func (provider *PgIntrospectionProvider) queryColumns(
	ctx context.Context,
	schemaName string,
	tableName string,
) (*sql.Rows, error) {
	return provider.database.QueryContext(ctx,
		`SELECT
			c.column_name,
			c.udt_name,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			COALESCE(c.is_generated, 'NEVER') AS is_generated,
			COALESCE(c.generation_expression, '') AS generation_expression,
			COALESCE(c.is_identity, 'NO') AS is_identity,
			COALESCE(c.identity_generation, '') AS identity_generation
		 FROM information_schema.columns c
		 WHERE c.table_schema = $1 AND c.table_name = $2
		 ORDER BY c.ordinal_position`,
		schemaName, tableName)
}

type columnRow struct {
	columnName string

	udtName string

	isNullable string

	isGenerated string

	generationExpression string

	isIdentity string

	identityGeneration string

	columnDefault sql.NullString

	characterMaximumLength sql.NullInt64

	numericPrecision sql.NullInt64

	numericScale sql.NullInt64
}

func (provider *PgIntrospectionProvider) scanColumn(rows *sql.Rows) (querier_dto.Column, error) {
	var row columnRow

	scanError := rows.Scan(
		&row.columnName,
		&row.udtName,
		&row.isNullable,
		&row.columnDefault,
		&row.characterMaximumLength,
		&row.numericPrecision,
		&row.numericScale,
		&row.isGenerated,
		&row.generationExpression,
		&row.isIdentity,
		&row.identityGeneration,
	)
	if scanError != nil {
		return querier_dto.Column{}, scanError
	}

	modifiers := buildTypeModifiers(row)
	sqlType := provider.typeNormaliser.NormaliseTypeName(row.udtName, modifiers...)

	column := querier_dto.Column{
		Name:       row.columnName,
		SQLType:    sqlType,
		Nullable:   row.isNullable == "YES",
		HasDefault: row.columnDefault.Valid || row.isIdentity != "NO",
	}

	if row.isGenerated == "ALWAYS" {
		column.IsGenerated = true
		column.GeneratedKind = querier_dto.GeneratedKindStored
	}

	return column, nil
}

func buildTypeModifiers(row columnRow) []int {
	var modifiers []int
	if row.characterMaximumLength.Valid {
		modifiers = append(modifiers, int(row.characterMaximumLength.Int64))
	}
	if row.numericPrecision.Valid {
		modifiers = append(modifiers, int(row.numericPrecision.Int64))
	}
	if row.numericScale.Valid {
		modifiers = append(modifiers, int(row.numericScale.Int64))
	}
	return modifiers
}

type constraintEntry struct {
	Name string

	Columns []string
}

func (provider *PgIntrospectionProvider) introspectConstraints(
	ctx context.Context,
	schemaName string,
	tableName string,
) ([]string, []constraintEntry, error) {
	rows, queryError := provider.queryConstraints(ctx, schemaName, tableName)
	if queryError != nil {
		return nil, nil, queryError
	}
	defer rows.Close()

	primaryKeyColumnMap := make(map[int]string)
	uniqueConstraintMap := make(map[string][]string)
	var uniqueConstraintOrder []string

	for rows.Next() {
		var constraintName string
		var constraintType string
		var attributeName string
		var ordinal int

		scanError := rows.Scan(&constraintName, &constraintType, &attributeName, &ordinal)
		if scanError != nil {
			return nil, nil, scanError
		}

		switch constraintType {
		case "p":
			primaryKeyColumnMap[ordinal] = attributeName
		case "u":
			if _, exists := uniqueConstraintMap[constraintName]; !exists {
				uniqueConstraintOrder = append(uniqueConstraintOrder, constraintName)
			}
			uniqueConstraintMap[constraintName] = append(uniqueConstraintMap[constraintName], attributeName)
		}
	}

	if rowError := rows.Err(); rowError != nil {
		return nil, nil, rowError
	}

	primaryKeyColumns, uniqueConstraints := assembleConstraintResults(primaryKeyColumnMap, uniqueConstraintMap, uniqueConstraintOrder)
	return primaryKeyColumns, uniqueConstraints, nil
}

func (provider *PgIntrospectionProvider) queryConstraints(
	ctx context.Context,
	schemaName string,
	tableName string,
) (*sql.Rows, error) {
	return provider.database.QueryContext(ctx,
		`SELECT
			con.conname,
			con.contype,
			att.attname,
			un.ord
		 FROM pg_constraint con
		 JOIN pg_class t ON t.oid = con.conrelid
		 JOIN pg_namespace n ON n.oid = t.relnamespace
		 JOIN LATERAL unnest(con.conkey) WITH ORDINALITY AS un(attnum, ord) ON true
		 JOIN pg_attribute att ON att.attrelid = t.oid AND att.attnum = un.attnum
		 WHERE n.nspname = $1 AND t.relname = $2 AND con.contype IN ('p', 'u')
		 ORDER BY con.conname, un.ord`,
		schemaName, tableName)
}

func assembleConstraintResults(
	primaryKeyColumnMap map[int]string,
	uniqueConstraintMap map[string][]string,
	uniqueConstraintOrder []string,
) ([]string, []constraintEntry) {
	var primaryKeyColumns []string
	for i := 1; i <= len(primaryKeyColumnMap); i++ {
		primaryKeyColumns = append(primaryKeyColumns, primaryKeyColumnMap[i])
	}

	uniqueConstraints := make([]constraintEntry, 0, len(uniqueConstraintOrder))
	for _, name := range uniqueConstraintOrder {
		uniqueConstraints = append(uniqueConstraints, constraintEntry{
			Name:    name,
			Columns: uniqueConstraintMap[name],
		})
	}

	return primaryKeyColumns, uniqueConstraints
}

type indexEntry struct {
	columns []string

	isUnique bool

	isPrimary bool
}

func (provider *PgIntrospectionProvider) introspectIndexes(
	ctx context.Context,
	schemaName string,
	tableName string,
) ([]querier_dto.Index, error) {
	rows, queryError := provider.queryIndexes(ctx, schemaName, tableName)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	indexMap := make(map[string]*indexEntry)
	var indexOrder []string

	for rows.Next() {
		var indexName string
		var isUnique bool
		var isPrimary bool
		var attributeName string
		var columnPosition int

		scanError := rows.Scan(&indexName, &isUnique, &isPrimary, &attributeName, &columnPosition)
		if scanError != nil {
			return nil, scanError
		}

		entry, exists := indexMap[indexName]
		if !exists {
			entry = &indexEntry{isUnique: isUnique, isPrimary: isPrimary}
			indexMap[indexName] = entry
			indexOrder = append(indexOrder, indexName)
		}

		entry.columns = append(entry.columns, attributeName)
	}

	if rowError := rows.Err(); rowError != nil {
		return nil, rowError
	}

	return assembleIndexResults(indexMap, indexOrder), nil
}

func (provider *PgIntrospectionProvider) queryIndexes(
	ctx context.Context,
	schemaName string,
	tableName string,
) (*sql.Rows, error) {
	return provider.database.QueryContext(ctx,
		`SELECT
			i.relname AS index_name,
			ix.indisunique,
			ix.indisprimary,
			a.attname,
			array_position(ix.indkey, a.attnum) AS column_position
		 FROM pg_index ix
		 JOIN pg_class i ON i.oid = ix.indexrelid
		 JOIN pg_class t ON t.oid = ix.indrelid
		 JOIN pg_namespace n ON n.oid = t.relnamespace
		 JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		 WHERE n.nspname = $1 AND t.relname = $2
		 ORDER BY i.relname, array_position(ix.indkey, a.attnum)`,
		schemaName, tableName)
}

func assembleIndexResults(
	indexMap map[string]*indexEntry,
	indexOrder []string,
) []querier_dto.Index {
	indexes := make([]querier_dto.Index, 0, len(indexOrder))
	for _, indexName := range indexOrder {
		entry := indexMap[indexName]
		indexes = append(indexes, querier_dto.Index{
			Name:      indexName,
			Columns:   entry.columns,
			IsUnique:  entry.isUnique,
			IsPrimary: entry.isPrimary,
		})
	}
	return indexes
}
