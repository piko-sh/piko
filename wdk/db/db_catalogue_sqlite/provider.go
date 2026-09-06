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
	"errors"
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

	// indexOriginPrimaryKey is the PRAGMA index_list origin reported for the automatic index
	// that backs a PRIMARY KEY constraint.
	indexOriginPrimaryKey = "pk"

	// listTablesQuery selects user tables, excluding SQLite's internal sqlite_ tables. The
	// underscore is escaped because LIKE otherwise treats it as a single-character wildcard
	// and would also hide user tables such as "sqliteapp".
	listTablesQuery = `SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite\_%' ESCAPE '\'
		ORDER BY name`

	// listViewsQuery selects user views.
	listViewsQuery = "SELECT name FROM sqlite_master WHERE type = 'view' ORDER BY name"

	// tableColumnsQuery selects every column of a table or view, including hidden generated
	// columns, through the table-valued form of PRAGMA table_xinfo so the relation name is
	// bound as a parameter rather than interpolated.
	tableColumnsQuery = `SELECT cid, name, type, "notnull", dflt_value, pk, hidden
		FROM pragma_table_xinfo(?)
		ORDER BY cid`

	// tableIndexesQuery selects every index of a table together with its key columns.
	//
	// Joining the table-valued PRAGMA functions in one statement avoids issuing a second
	// query while the first result set is still open, which would deadlock a pool limited to
	// one connection. Expression key columns have a NULL name.
	tableIndexesQuery = `SELECT index_list.name, index_list."unique", index_list.origin, index_info.name
		FROM pragma_index_list(?) AS index_list
		LEFT JOIN pragma_index_info(index_list.name) AS index_info
		ORDER BY index_list.seq, index_info.seqno`
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
	return provider.queryStringColumn(ctx, listTablesQuery)
}

// listViews returns the names of user views in the SQLite database.
//
// Returns []string which contains view names in alphabetical order.
// Returns error when the catalogue query fails.
func (provider *PragmaIntrospectionProvider) listViews(
	ctx context.Context,
) ([]string, error) {
	return provider.queryStringColumn(ctx, listViewsQuery)
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
) (names []string, err error) {
	rows, queryError := provider.database.QueryContext(ctx, query, args...)
	if queryError != nil {
		return nil, queryError
	}
	defer closeRows(rows, &err)

	for rows.Next() {
		var name string
		if scanError := rows.Scan(&name); scanError != nil {
			return nil, scanError
		}
		names = append(names, name)
	}
	if rowsError := rows.Err(); rowsError != nil {
		return nil, rowsError
	}
	return names, nil
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
		Name:              tableName,
		Schema:            schemaMain,
		Columns:           columns,
		PrimaryKey:        primaryKeyColumns,
		Indexes:           indexes,
		Comment:           "",
		VirtualModuleName: "",
		Constraints:       nil,
		Origin:            querier_dto.MigrationOrigin{},
		IsVirtual:         false,
		IsWithoutRowID:    false,
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
		Name:       viewName,
		Schema:     "main",
		Columns:    columns,
		Definition: "",
		Comment:    "",
		Origin:     querier_dto.MigrationOrigin{},
	}, nil
}

// introspectColumns lists columns and primary key fields for a table or view.
//
// Takes tableName (string) which identifies the table or view to introspect.
//
// Returns columns ([]querier_dto.Column) which describes each column.
// Returns primaryKey ([]string) which contains primary key column names in order.
// Returns err (error) when the PRAGMA query, a scan, or closing the rows fails.
func (provider *PragmaIntrospectionProvider) introspectColumns(
	ctx context.Context,
	tableName string,
) (columns []querier_dto.Column, primaryKey []string, err error) {
	rows, queryError := provider.database.QueryContext(ctx, tableColumnsQuery, tableName)
	if queryError != nil {
		return nil, nil, queryError
	}
	defer closeRows(rows, &err)

	primaryKeyByPosition := map[int]string{}

	for rows.Next() {
		column, primaryKeyPosition, scanError := provider.scanColumn(rows)
		if scanError != nil {
			return nil, nil, scanError
		}

		columns = append(columns, column)

		if primaryKeyPosition > 0 {
			primaryKeyByPosition[primaryKeyPosition] = column.Name
		}
	}

	if rowsError := rows.Err(); rowsError != nil {
		return nil, nil, rowsError
	}

	return columns, orderedPrimaryKeyColumns(primaryKeyByPosition), nil
}

// scanColumn decodes one PRAGMA table_xinfo row into a column descriptor.
//
// Takes rows (*sql.Rows) which is positioned on the row to decode.
//
// Returns querier_dto.Column which describes the column.
// Returns int which is the column's 1-based position within the primary key, or 0 when
// the column is not part of it.
// Returns error when the row cannot be scanned.
func (provider *PragmaIntrospectionProvider) scanColumn(rows *sql.Rows) (querier_dto.Column, int, error) {
	var columnID int
	var name string
	var typeName string
	var notNull int
	var defaultValue sql.NullString
	var primaryKeyPosition int
	var hidden int

	if scanError := rows.Scan(&columnID, &name, &typeName, &notNull, &defaultValue, &primaryKeyPosition, &hidden); scanError != nil {
		return querier_dto.Column{}, 0, scanError
	}

	sqlType := provider.typeNormaliser.NormaliseTypeName(strings.TrimSpace(typeName))
	column := querier_dto.NewColumn(name, sqlType, notNull == 0 && primaryKeyPosition == 0)
	column.HasDefault = defaultValue.Valid || primaryKeyPosition > 0
	column.IsGenerated, column.GeneratedKind = classifyGeneratedColumn(hidden)

	return column, primaryKeyPosition, nil
}

// introspectIndexes lists indexes defined on the named table.
//
// Every index and its key columns are read through one query whose rows are fully
// consumed before it returns, so no second statement is issued while a result set is
// open. That matters because the SQLite drivers cap the pool at one connection, where a
// nested query would wait forever for the connection the outer rows still hold.
//
// Takes tableName (string) which identifies the table to introspect.
//
// Returns indexes ([]querier_dto.Index) which describes each index and its named key
// columns, omitting indexes whose keys are all expressions.
// Returns err (error) when the query, a scan, or closing the rows fails.
func (provider *PragmaIntrospectionProvider) introspectIndexes(
	ctx context.Context,
	tableName string,
) (indexes []querier_dto.Index, err error) {
	rows, queryError := provider.database.QueryContext(ctx, tableIndexesQuery, tableName)
	if queryError != nil {
		return nil, queryError
	}
	defer closeRows(rows, &err)

	positionByName := map[string]int{}

	for rows.Next() {
		var indexName string
		var unique int
		var origin string
		var columnName sql.NullString

		if scanError := rows.Scan(&indexName, &unique, &origin, &columnName); scanError != nil {
			return nil, scanError
		}

		position, seen := positionByName[indexName]
		if !seen {
			position = len(indexes)
			positionByName[indexName] = position
			indexes = append(indexes, querier_dto.Index{
				Name:      indexName,
				Columns:   nil,
				IsUnique:  unique != 0,
				IsPrimary: origin == indexOriginPrimaryKey,
				Origin:    querier_dto.MigrationOrigin{},
			})
		}

		if columnName.Valid {
			indexes[position].Columns = append(indexes[position].Columns, columnName.String)
		}
	}

	if rowsError := rows.Err(); rowsError != nil {
		return nil, rowsError
	}

	return slices.DeleteFunc(indexes, func(index querier_dto.Index) bool {
		return len(index.Columns) == 0
	}), nil
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

// closeRows closes rows and joins any close failure into the caller's named error result.
//
// Takes rows (*sql.Rows) which is the result set to close.
// Takes err (*error) which is the caller's named error result that receives a close
// failure.
func closeRows(rows *sql.Rows, err *error) {
	if closeError := rows.Close(); closeError != nil {
		*err = errors.Join(*err, fmt.Errorf("closing rows: %w", closeError))
	}
}
