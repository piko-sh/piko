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

package db_catalogue_sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// TypeNormaliser converts engine-specific type names to structured SQLType values.
// Satisfied by any EnginePort implementation.
type TypeNormaliser interface {
	// NormaliseTypeName converts a raw SQL type name to a structured SQLType.
	//
	// Takes name (string) which is the raw type name as reported by the engine.
	// Takes modifiers (...int) which carry optional precision or scale values.
	//
	// Returns querier_dto.SQLType which is the structured representation.
	NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType
}

const (
	// schemaMain is the SQLite default schema name.
	schemaMain = "main"

	// hiddenVirtualColumn marks a generated column stored as a virtual expression in PRAGMA
	// table_xinfo output.
	hiddenVirtualColumn = 2

	// hiddenStoredColumn marks a generated column whose value is stored on disk in PRAGMA
	// table_xinfo output.
	hiddenStoredColumn = 3
)

// PragmaIntrospectionProvider implements CatalogueProviderPort by querying a live SQLite
// database using PRAGMA commands.
type PragmaIntrospectionProvider struct {
	// database is the SQLite connection used to issue PRAGMA queries.
	database *sql.DB

	// typeNormaliser converts raw SQLite type strings to structured SQLType values.
	typeNormaliser TypeNormaliser
}

// NewPragmaIntrospectionProvider creates a new PRAGMA-based catalogue provider.
//
// Takes database (*sql.DB) which is the SQLite connection to introspect.
// Takes typeNormaliser (TypeNormaliser) which converts raw type names to structured
// SQLType values.
//
// Returns *PragmaIntrospectionProvider which is ready to build catalogues.
func NewPragmaIntrospectionProvider(
	database *sql.DB,
	typeNormaliser TypeNormaliser,
) *PragmaIntrospectionProvider {
	return &PragmaIntrospectionProvider{
		database:       database,
		typeNormaliser: typeNormaliser,
	}
}

// BuildCatalogue introspects the SQLite database and builds a schema catalogue.
//
// Returns *querier_dto.Catalogue which describes tables, views, and indexes.
// Returns []querier_dto.SourceError which lists per-object diagnostics, always nil for
// the SQLite provider.
// Returns error when a PRAGMA or introspection query fails.
func (provider *PragmaIntrospectionProvider) BuildCatalogue(
	ctx context.Context,
) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
	catalogue := &querier_dto.Catalogue{
		DefaultSchema: schemaMain,
		Schemas: map[string]*querier_dto.Schema{
			schemaMain: {
				Name:           schemaMain,
				Tables:         make(map[string]*querier_dto.Table),
				Views:          make(map[string]*querier_dto.View),
				Enums:          make(map[string]*querier_dto.Enum),
				Functions:      make(map[string][]*querier_dto.FunctionSignature),
				CompositeTypes: make(map[string]*querier_dto.CompositeType),
				Sequences:      make(map[string]*querier_dto.Sequence),
			},
		},
		Extensions: make(map[string]struct{}),
	}

	schema := catalogue.Schemas[schemaMain]

	tables, tableError := provider.listTables(ctx)
	if tableError != nil {
		return nil, nil, fmt.Errorf("listing tables: %w", tableError)
	}

	for _, tableName := range tables {
		if cancelError := ctx.Err(); cancelError != nil {
			return nil, nil, cancelError
		}
		table, introspectError := provider.introspectTable(ctx, tableName)
		if introspectError != nil {
			return nil, nil, fmt.Errorf("introspecting table %s: %w", tableName, introspectError)
		}
		schema.Tables[tableName] = table
	}

	views, viewError := provider.listViews(ctx)
	if viewError != nil {
		return nil, nil, fmt.Errorf("listing views: %w", viewError)
	}

	for _, viewName := range views {
		if cancelError := ctx.Err(); cancelError != nil {
			return nil, nil, cancelError
		}
		view, introspectError := provider.introspectView(ctx, viewName)
		if introspectError != nil {
			return nil, nil, fmt.Errorf("introspecting view %s: %w", viewName, introspectError)
		}
		schema.Views[viewName] = view
	}

	return catalogue, nil, nil
}

// listTables returns the names of user tables in the SQLite database.
//
// Returns []string which contains user table names in alphabetical order.
// Returns error when the catalogue query fails.
func (provider *PragmaIntrospectionProvider) listTables(
	ctx context.Context,
) ([]string, error) {
	return provider.queryStringColumn(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
}

// listViews returns the names of user views in the SQLite database.
//
// Returns []string which contains view names in alphabetical order.
// Returns error when the catalogue query fails.
func (provider *PragmaIntrospectionProvider) listViews(
	ctx context.Context,
) ([]string, error) {
	return provider.queryStringColumn(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'view' ORDER BY name")
}

// queryStringColumn runs a single-column text query and collects values into a slice.
//
// Takes query (string) which is the SQL selecting one text column per row.
// Takes args (...any) which are the positional query parameters.
//
// Returns []string which contains the scanned values in row order.
// Returns error when the query, a scan, or row iteration fails.
func (provider *PragmaIntrospectionProvider) queryStringColumn(
	ctx context.Context,
	query string,
	args ...any,
) ([]string, error) {
	rows, queryError := provider.database.QueryContext(ctx, query, args...)
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

// introspectTable builds a Table descriptor for the named table.
//
// Takes tableName (string) which identifies the table to introspect.
//
// Returns *querier_dto.Table which describes columns, primary key, and indexes.
// Returns error when a PRAGMA query fails.
func (provider *PragmaIntrospectionProvider) introspectTable(
	ctx context.Context,
	tableName string,
) (*querier_dto.Table, error) {
	columns, primaryKeyColumns, introspectError := provider.introspectColumns(ctx, tableName)
	if introspectError != nil {
		return nil, introspectError
	}

	indexes, indexError := provider.introspectIndexes(ctx, tableName)
	if indexError != nil {
		return nil, indexError
	}

	return &querier_dto.Table{
		Name:       tableName,
		Schema:     schemaMain,
		Columns:    columns,
		PrimaryKey: primaryKeyColumns,
		Indexes:    indexes,
	}, nil
}

// introspectView builds a View descriptor for the named view.
//
// Takes viewName (string) which identifies the view to introspect.
//
// Returns *querier_dto.View which describes the view columns.
// Returns error when a PRAGMA query fails.
func (provider *PragmaIntrospectionProvider) introspectView(
	ctx context.Context,
	viewName string,
) (*querier_dto.View, error) {
	columns, _, introspectError := provider.introspectColumns(ctx, viewName)
	if introspectError != nil {
		return nil, introspectError
	}

	return &querier_dto.View{
		Name:    viewName,
		Schema:  "main",
		Columns: columns,
	}, nil
}

// introspectColumns lists columns and primary key fields for a table or view.
//
// Takes tableName (string) which identifies the table or view to introspect.
//
// Returns []querier_dto.Column which describes each column.
// Returns []string which contains primary key column names in order.
// Returns error when the PRAGMA query fails.
func (provider *PragmaIntrospectionProvider) introspectColumns(
	ctx context.Context,
	tableName string,
) ([]querier_dto.Column, []string, error) {
	//nolint:gosec // trusted source (sqlite_master)
	rows, queryError := provider.database.QueryContext(ctx,
		fmt.Sprintf("PRAGMA table_xinfo(%s)", quoteIdentifier(tableName)))
	if queryError != nil {
		return nil, nil, queryError
	}
	defer rows.Close()

	var columns []querier_dto.Column

	primaryKeyByPosition := map[int]string{}

	for rows.Next() {
		var columnID int
		var name string
		var typeName string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		var hidden int

		scanError := rows.Scan(&columnID, &name, &typeName, &notNull, &defaultValue, &primaryKey, &hidden)
		if scanError != nil {
			return nil, nil, scanError
		}

		sqlType := provider.typeNormaliser.NormaliseTypeName(strings.TrimSpace(typeName))

		isGenerated, generatedKind := classifyGeneratedColumn(hidden)

		column := querier_dto.Column{
			Name:          name,
			SQLType:       sqlType,
			Nullable:      notNull == 0 && primaryKey == 0,
			HasDefault:    defaultValue.Valid || primaryKey > 0,
			IsGenerated:   isGenerated,
			GeneratedKind: generatedKind,
		}

		columns = append(columns, column)

		if primaryKey > 0 {
			primaryKeyByPosition[primaryKey] = name
		}
	}

	return columns, orderedPrimaryKeyColumns(primaryKeyByPosition), rows.Err()
}

// orderedPrimaryKeyColumns returns the primary-key column names ordered by their 1-based
// position within the key, or nil when the table has no primary key.
//
// Takes byPosition (map[int]string) which maps a 1-based key position to its column name.
//
// Returns []string which holds the column names in key order.
func orderedPrimaryKeyColumns(byPosition map[int]string) []string {
	if len(byPosition) == 0 {
		return nil
	}
	positions := make([]int, 0, len(byPosition))
	for position := range byPosition {
		positions = append(positions, position)
	}
	slices.Sort(positions)
	ordered := make([]string, len(positions))
	for index, position := range positions {
		ordered[index] = byPosition[position]
	}
	return ordered
}

// introspectIndexes lists indexes defined on the named table.
//
// Takes tableName (string) which identifies the table to introspect.
//
// Returns []querier_dto.Index which describes each index and its columns.
// Returns error when a PRAGMA query fails.
func (provider *PragmaIntrospectionProvider) introspectIndexes(
	ctx context.Context,
	tableName string,
) ([]querier_dto.Index, error) {
	//nolint:gosec // trusted source (sqlite_master)
	indexRows, queryError := provider.database.QueryContext(ctx,
		fmt.Sprintf("PRAGMA index_list(%s)", quoteIdentifier(tableName)))
	if queryError != nil {
		return nil, queryError
	}
	defer indexRows.Close()

	var indexes []querier_dto.Index

	for indexRows.Next() {
		var sequence int
		var indexName string
		var unique int
		var origin string
		var partial int

		scanError := indexRows.Scan(&sequence, &indexName, &unique, &origin, &partial)
		if scanError != nil {
			return nil, scanError
		}

		indexColumns, columnError := provider.introspectIndexColumns(ctx, indexName)
		if columnError != nil {
			return nil, columnError
		}

		indexes = append(indexes, querier_dto.Index{
			Name:     indexName,
			Columns:  indexColumns,
			IsUnique: unique != 0,
		})
	}

	return indexes, indexRows.Err()
}

// introspectIndexColumns lists the columns referenced by the named index.
//
// Takes indexName (string) which identifies the index to introspect.
//
// Returns []string which contains index column names in declaration order.
// Returns error when the PRAGMA query fails.
func (provider *PragmaIntrospectionProvider) introspectIndexColumns(
	ctx context.Context,
	indexName string,
) ([]string, error) {
	//nolint:gosec // trusted source (PRAGMA)
	rows, queryError := provider.database.QueryContext(ctx,
		fmt.Sprintf("PRAGMA index_info(%s)", quoteIdentifier(indexName)))
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	var columnNames []string
	for rows.Next() {
		var rank int
		var columnID int
		var name string

		if scanError := rows.Scan(&rank, &columnID, &name); scanError != nil {
			return nil, scanError
		}
		columnNames = append(columnNames, name)
	}

	return columnNames, rows.Err()
}

// classifyGeneratedColumn maps a PRAGMA table_xinfo hidden flag to its generated state.
//
// SQLite reports hiddenVirtualColumn for a VIRTUAL generated column and
// hiddenStoredColumn for a STORED one. Any other flag denotes an ordinary column.
//
// Takes hidden (int) which is the hidden flag from PRAGMA table_xinfo.
//
// Returns bool which is true when the column is a generated column.
// Returns querier_dto.GeneratedKind which distinguishes STORED from VIRTUAL, or
// GeneratedKindNone for an ordinary column.
func classifyGeneratedColumn(hidden int) (bool, querier_dto.GeneratedKind) {
	switch hidden {
	case hiddenStoredColumn:
		return true, querier_dto.GeneratedKindStored
	case hiddenVirtualColumn:
		return true, querier_dto.GeneratedKindVirtual
	default:
		return false, querier_dto.GeneratedKindNone
	}
}

// quoteIdentifier wraps a SQL identifier in double quotes and escapes inner quotes for
// safe PRAGMA interpolation.
//
// Takes identifier (string) which is the raw identifier to quote.
//
// Returns string which is the double-quoted identifier.
func quoteIdentifier(identifier string) string {
	return "\"" + strings.ReplaceAll(identifier, "\"", "\"\"") + "\""
}
