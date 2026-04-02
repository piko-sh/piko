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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// TypeNormaliser converts engine-specific type names to structured SQLType
// values. This is satisfied by any EnginePort implementation.
type TypeNormaliser interface {
	NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType
}

const (
	schemaMain = "main"

	hiddenVirtualColumn = 2

	hiddenStoredColumn = 3
)

// PragmaIntrospectionProvider implements CatalogueProviderPort by querying
// a live SQLite database using PRAGMA commands.
type PragmaIntrospectionProvider struct {
	database *sql.DB

	typeNormaliser TypeNormaliser
}

// NewPragmaIntrospectionProvider creates a new PRAGMA-based catalogue provider.
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
		view, introspectError := provider.introspectView(ctx, viewName)
		if introspectError != nil {
			return nil, nil, fmt.Errorf("introspecting view %s: %w", viewName, introspectError)
		}
		schema.Views[viewName] = view
	}

	return catalogue, nil, nil
}

func (provider *PragmaIntrospectionProvider) listTables(
	ctx context.Context,
) ([]string, error) {
	rows, queryError := provider.database.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
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

func (provider *PragmaIntrospectionProvider) listViews(
	ctx context.Context,
) ([]string, error) {
	rows, queryError := provider.database.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'view' ORDER BY name")
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
	var primaryKeyColumns []string

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

		column := querier_dto.Column{
			Name:       name,
			SQLType:    sqlType,
			Nullable:   notNull == 0,
			HasDefault: defaultValue.Valid || primaryKey > 0,
		}

		if hidden == hiddenVirtualColumn || hidden == hiddenStoredColumn {
			column.IsGenerated = true
			if hidden == hiddenStoredColumn {
				column.GeneratedKind = querier_dto.GeneratedKindStored
			} else {
				column.GeneratedKind = querier_dto.GeneratedKindVirtual
			}
		}

		columns = append(columns, column)

		if primaryKey > 0 {
			primaryKeyColumns = append(primaryKeyColumns, name)
		}
	}

	return columns, primaryKeyColumns, rows.Err()
}

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

func quoteIdentifier(identifier string) string {
	return "\"" + strings.ReplaceAll(identifier, "\"", "\"\"") + "\""
}
