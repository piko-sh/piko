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

// viewEntry is one view's name and definition as listed by information_schema.views.
type viewEntry struct {
	// name is the view's identifier.
	name string

	// definition is the view's SELECT text, NULL when the role cannot see it.
	definition sql.NullString
}

// introspectViews fills the schema's Views map.
//
// Takes schema (*querier_dto.Schema) which receives the views.
//
// Returns error when any query or column introspection fails.
func (provider *PgIntrospectionProvider) introspectViews(
	ctx context.Context,
	schema *querier_dto.Schema,
) error {
	views, listError := provider.listViews(ctx, schema.Name)
	if listError != nil {
		return listError
	}

	for _, entry := range views {
		if cancelError := ctx.Err(); cancelError != nil {
			return cancelError
		}
		columns, columnError := provider.introspectColumns(ctx, schema.Name, entry.name)
		if columnError != nil {
			return fmt.Errorf("introspecting view columns for %s: %w", entry.name, columnError)
		}

		schema.Views[entry.name] = &querier_dto.View{
			Name:       entry.name,
			Schema:     schema.Name,
			Columns:    columns,
			Definition: entry.definition.String,
			Comment:    "",
			Origin:     querier_dto.MigrationOrigin{},
		}
	}

	return nil
}

// listViews returns the schema's views, closing the result set before the caller issues
// per-view queries.
//
// Takes schemaName (string) which selects the owning schema.
//
// Returns views ([]viewEntry) which lists the views in name order.
// Returns err (error) when the query, a scan, or closing the rows fails.
func (provider *PgIntrospectionProvider) listViews(
	ctx context.Context,
	schemaName string,
) (views []viewEntry, err error) {
	rows, queryError := provider.database.QueryContext(ctx,
		`SELECT table_name, view_definition
		 FROM information_schema.views
		 WHERE table_schema = $1
		 ORDER BY table_name`,
		schemaName)
	if queryError != nil {
		return nil, queryError
	}
	defer closeRows(rows, &err)

	for rows.Next() {
		var entry viewEntry
		if scanError := rows.Scan(&entry.name, &entry.definition); scanError != nil {
			return nil, scanError
		}
		views = append(views, entry)
	}

	if rowError := rows.Err(); rowError != nil {
		return nil, rowError
	}

	return views, nil
}

// introspectEnums fills the schema's Enums map.
//
// Takes schema (*querier_dto.Schema) which receives the enums.
//
// Returns err (error) when the query, a scan, or closing the rows fails.
func (provider *PgIntrospectionProvider) introspectEnums(
	ctx context.Context,
	schema *querier_dto.Schema,
) (err error) {
	rows, queryError := provider.database.QueryContext(ctx,
		`SELECT t.typname, e.enumlabel
		 FROM pg_type t
		 JOIN pg_enum e ON t.oid = e.enumtypid
		 JOIN pg_namespace n ON t.typnamespace = n.oid
		 WHERE n.nspname = $1
		 ORDER BY t.typname, e.enumsortorder`,
		schema.Name)
	if queryError != nil {
		return queryError
	}
	defer closeRows(rows, &err)

	enumMap := make(map[string][]string)
	var enumOrder []string

	for rows.Next() {
		var typeName string
		var enumLabel string

		if scanError := rows.Scan(&typeName, &enumLabel); scanError != nil {
			return scanError
		}

		if _, exists := enumMap[typeName]; !exists {
			enumOrder = append(enumOrder, typeName)
		}
		enumMap[typeName] = append(enumMap[typeName], enumLabel)
	}

	if rowError := rows.Err(); rowError != nil {
		return rowError
	}

	for _, typeName := range enumOrder {
		schema.Enums[typeName] = &querier_dto.Enum{
			Name:    typeName,
			Schema:  schema.Name,
			Values:  enumMap[typeName],
			Comment: "",
			Origin:  querier_dto.MigrationOrigin{},
		}
	}

	return nil
}

// compositeField is one attribute of a PostgreSQL composite type.
type compositeField struct {
	// attributeName is the field identifier inside the composite.
	attributeName string

	// typeName is the formatted PostgreSQL type for the field.
	typeName string
}

// introspectCompositeTypes fills the schema's CompositeTypes map.
//
// Takes schema (*querier_dto.Schema) which receives the composites.
//
// Returns err (error) when the query, a scan, or closing the rows fails.
func (provider *PgIntrospectionProvider) introspectCompositeTypes(
	ctx context.Context,
	schema *querier_dto.Schema,
) (err error) {
	rows, queryError := provider.queryCompositeTypes(ctx, schema.Name)
	if queryError != nil {
		return queryError
	}
	defer closeRows(rows, &err)

	compositeMap := make(map[string][]compositeField)
	var compositeOrder []string

	for rows.Next() {
		var typeName string
		var attributeName string
		var fieldTypeName string

		if scanError := rows.Scan(&typeName, &attributeName, &fieldTypeName); scanError != nil {
			return scanError
		}

		if _, exists := compositeMap[typeName]; !exists {
			compositeOrder = append(compositeOrder, typeName)
		}
		compositeMap[typeName] = append(compositeMap[typeName], compositeField{
			attributeName: attributeName,
			typeName:      fieldTypeName,
		})
	}

	if rowError := rows.Err(); rowError != nil {
		return rowError
	}

	provider.assembleCompositeTypes(schema, compositeMap, compositeOrder)

	return nil
}

// queryCompositeTypes runs the pg_type composite-attribute query.
//
// Takes schemaName (string) which selects the owning schema.
//
// Returns *sql.Rows which the caller must close.
// Returns error when the query fails to dispatch.
func (provider *PgIntrospectionProvider) queryCompositeTypes(
	ctx context.Context,
	schemaName string,
) (*sql.Rows, error) {
	return provider.database.QueryContext(ctx,
		`SELECT t.typname, a.attname, format_type(a.atttypid, a.atttypmod) AS type_name
		 FROM pg_type t
		 JOIN pg_namespace n ON t.typnamespace = n.oid
		 JOIN pg_attribute a ON a.attrelid = t.typrelid
		 WHERE t.typtype = 'c' AND n.nspname = $1 AND a.attnum > 0
		 AND NOT EXISTS (
			SELECT 1 FROM pg_class c
			WHERE c.oid = t.typrelid AND c.relkind IN ('r', 'v', 'm')
		 )
		 ORDER BY t.typname, a.attnum`,
		schemaName)
}

// assembleCompositeTypes materialises composite DTOs onto the schema.
//
// Takes schema (*querier_dto.Schema) which receives the composites.
// Takes compositeMap (map[string][]compositeField) which maps a composite name to its
// fields in declaration order.
// Takes compositeOrder ([]string) which preserves discovery order.
func (provider *PgIntrospectionProvider) assembleCompositeTypes(
	schema *querier_dto.Schema,
	compositeMap map[string][]compositeField,
	compositeOrder []string,
) {
	for _, typeName := range compositeOrder {
		fields := compositeMap[typeName]
		columns := make([]querier_dto.Column, 0, len(fields))

		for _, field := range fields {
			sqlType := provider.typeNormaliser.NormaliseTypeName(field.typeName)
			columns = append(columns, querier_dto.NewColumn(field.attributeName, sqlType, false))
		}

		schema.CompositeTypes[typeName] = &querier_dto.CompositeType{
			Name:   typeName,
			Schema: schema.Name,
			Fields: columns,
			Origin: querier_dto.MigrationOrigin{},
		}
	}
}

// introspectExtensions fills the catalogue's Extensions set.
//
// Ignores the built-in plpgsql extension because it is always present.
//
// Takes catalogue (*querier_dto.Catalogue) which receives entries.
//
// Returns err (error) when the query, a scan, or closing the rows fails.
func (provider *PgIntrospectionProvider) introspectExtensions(
	ctx context.Context,
	catalogue *querier_dto.Catalogue,
) (err error) {
	rows, queryError := provider.database.QueryContext(ctx,
		`SELECT extname FROM pg_extension WHERE extname != 'plpgsql'`)
	if queryError != nil {
		return queryError
	}
	defer closeRows(rows, &err)

	for rows.Next() {
		var extensionName string
		if scanError := rows.Scan(&extensionName); scanError != nil {
			return scanError
		}
		catalogue.Extensions[extensionName] = struct{}{}
	}

	return rows.Err()
}
