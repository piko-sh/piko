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
	"piko.sh/piko/internal/querier/querier_dto"
)

// nullabilityPropagator applies final nullability adjustments to output columns and
// parameters after type resolution. This handles directive overrides, GROUP BY primary
// key functional dependency rules (SQL:2003), and parameter kind-specific nullability.
type nullabilityPropagator struct {
	// catalogue holds the schema state for primary key lookups.
	catalogue *querier_dto.Catalogue
}

// newNullabilityPropagator creates a new nullability propagator with the given catalogue
// for primary key lookup.
//
// Takes catalogue (*querier_dto.Catalogue) which provides the schema state for primary
// key lookups.
//
// Returns *nullabilityPropagator which is ready to apply nullability adjustments.
func newNullabilityPropagator(catalogue *querier_dto.Catalogue) *nullabilityPropagator {
	return &nullabilityPropagator{
		catalogue: catalogue,
	}
}

// PropagateOutputNullability applies final nullability adjustments to output columns
// based on directives and GROUP BY functional dependency rules.
//
// Adjustments are applied in order: first the piko.nullable directive override, then the
// GROUP BY primary key rule.
//
// Takes columns ([]querier_dto.OutputColumn) which holds the output columns to adjust.
// Takes queryDirectives (*querier_dto.QueryDirectives) which holds any nullable override
// directive.
// Takes scope (*scopeChain) which provides table alias resolution.
// Takes groupByColumns ([]querier_dto.ColumnReference) which holds the GROUP BY column
// references.
//
// Returns []querier_dto.OutputColumn which holds the adjusted output columns.
func (p *nullabilityPropagator) PropagateOutputNullability(
	columns []querier_dto.OutputColumn,
	queryDirectives *querier_dto.QueryDirectives,
	scope *scopeChain,
	groupByColumns []querier_dto.ColumnReference,
) []querier_dto.OutputColumn {
	result := make([]querier_dto.OutputColumn, len(columns))
	copy(result, columns)

	if queryDirectives != nil && queryDirectives.NullableOverride != nil {
		for i := range result {
			result[i].Nullable = *queryDirectives.NullableOverride
		}
		return result
	}

	if len(groupByColumns) > 0 {
		pkCoveredTables := p.findPrimaryKeyCoveredTables(groupByColumns, scope)
		p.applyGroupByNullability(result, pkCoveredTables)
	}

	return result
}

// PropagateParameterNullability applies final nullability adjustments to parameters based
// on their parameter kind declarations.
//
// A piko.optional parameter is always nullable. A piko.limit or piko.offset parameter is
// always a NOT NULL integer. A piko.slice parameter is always NOT NULL. A piko.param
// parameter takes the nullable:true or nullable:false override when one is supplied.
//
// Takes parameters ([]querier_dto.QueryParameter) which holds the parameters to adjust.
// Takes parameterDirectives ([]*querier_dto.ParameterDirective) which holds the directive
// declarations.
//
// Returns []querier_dto.QueryParameter which holds the adjusted parameters.
func (*nullabilityPropagator) PropagateParameterNullability(
	parameters []querier_dto.QueryParameter,
	parameterDirectives []*querier_dto.ParameterDirective,
) []querier_dto.QueryParameter {
	result := make([]querier_dto.QueryParameter, len(parameters))
	copy(result, parameters)

	directiveMap := make(map[int]*querier_dto.ParameterDirective, len(parameterDirectives))
	for _, directive := range parameterDirectives {
		directiveMap[directive.Number] = directive
	}

	for i := range result {
		directive, exists := directiveMap[result[i].Number]
		if !exists {
			continue
		}

		result[i].Kind = directive.Kind
		result[i].IsOptional = directive.IsOptional
		result[i].IsSlice = directive.IsSlice
		switch {
		case directive.Kind == querier_dto.ParameterDirectiveSortable:
			result[i].Nullable = false
			result[i].SortableColumns = directive.Columns
		case directive.IsOptional:

			result[i].Nullable = true
		case directive.IsSlice:
			result[i].Nullable = false
		case result[i].IsPaginationBound():

			result[i].Nullable = false
		default:
			if directive.Nullable != nil {
				result[i].Nullable = *directive.Nullable
			}
		}
	}

	return result
}

// findPrimaryKeyCoveredTables returns the set of schema-qualified tables whose full
// primary key is covered by the GROUP BY columns.
//
// Each GROUP BY column is attributed to its source table: a qualified reference resolves
// its alias directly in scope, while an unqualified reference (e.g. GROUP BY id) is
// resolved against the scope chain so its containing table is still found. Without the
// unqualified path GROUP BY columns written without a table alias would never bucket to a
// table and the functional-dependency rule could never fire.
//
// Takes groupByColumns ([]querier_dto.ColumnReference) which holds the GROUP BY column
// references.
// Takes scope (*scopeChain) which provides table alias resolution.
//
// Returns map[schemaTableKey]bool which maps a schema-qualified table to true when its
// primary key is fully covered.
func (p *nullabilityPropagator) findPrimaryKeyCoveredTables(
	groupByColumns []querier_dto.ColumnReference,
	scope *scopeChain,
) map[schemaTableKey]bool {
	tableGroupedColumns := make(map[schemaTableKey][]string)
	for _, groupByColumn := range groupByColumns {
		scopedTable := p.resolveGroupByTable(groupByColumn, scope)
		if scopedTable == nil {
			continue
		}
		key := p.schemaTableKeyFor(scopedTable.Schema, scopedTable.Name)
		tableGroupedColumns[key] = append(tableGroupedColumns[key], groupByColumn.ColumnName)
	}

	pkCoveredTables := make(map[schemaTableKey]bool)
	for key, columnNames := range tableGroupedColumns {
		if p.isPrimaryKeyFullyCovered(key.schema, key.table, columnNames) {
			pkCoveredTables[key] = true
		}
	}

	return pkCoveredTables
}

// schemaTableKey identifies a table by its schema and name so primary-key coverage and
// base-nullability lookups stay unambiguous when two schemas hold a table of the same
// name.
type schemaTableKey struct {
	// schema is the resolved (non-empty) schema name.
	schema string

	// table is the table name.
	table string
}

// schemaTableKeyFor builds a schemaTableKey, resolving an empty schema to the catalogue's
// default schema so keys derived from output columns and from the scope chain agree.
//
// Takes schema (string) which is the attributed schema, possibly empty.
// Takes table (string) which is the table name.
//
// Returns schemaTableKey which is the normalised key.
func (p *nullabilityPropagator) schemaTableKeyFor(schema string, table string) schemaTableKey {
	if schema == "" {
		schema = p.catalogue.DefaultSchema
	}
	return schemaTableKey{schema: schema, table: table}
}

// resolveGroupByTable attributes a GROUP BY column reference to its source table. A
// qualified reference looks its alias up directly; an unqualified reference is resolved
// against the scope chain so the containing table is still discovered.
//
// Takes groupByColumn (querier_dto.ColumnReference) which holds the GROUP BY reference.
// Takes scope (*scopeChain) which provides alias and column resolution.
//
// Returns *querier_dto.ScopedTable which holds the attributed table, or nil when no table
// could be determined.
func (*nullabilityPropagator) resolveGroupByTable(
	groupByColumn querier_dto.ColumnReference,
	scope *scopeChain,
) *querier_dto.ScopedTable {
	if groupByColumn.TableAlias != "" {
		if scopedTable, exists := scope.tables[groupByColumn.TableAlias]; exists {
			return scopedTable
		}
		return nil
	}
	_, scopedTable, resolveError := scope.ResolveColumn("", groupByColumn.ColumnName)
	if resolveError != nil {
		return nil
	}
	return scopedTable
}

// applyGroupByNullability restores base column nullability for tables whose primary key
// is fully covered by GROUP BY.
//
// When a table's full primary key appears in GROUP BY, every column of that table is
// functionally dependent on the grouping key (SQL:2003), so each such output column takes
// its catalogue base nullability. This deliberately overwrites a nullability that
// resolution had raised (for example a LEFT JOIN that marked a NOT NULL column nullable),
// because the functional dependency guarantees one row per group from that table. An OR
// against the resolved nullability would leave the raised value in place and make the
// rule inert.
//
// Takes result ([]querier_dto.OutputColumn) which holds the output columns to adjust in
// place.
// Takes pkCoveredTables (map[schemaTableKey]bool) which identifies tables with fully
// covered primary keys.
func (p *nullabilityPropagator) applyGroupByNullability(
	result []querier_dto.OutputColumn,
	pkCoveredTables map[schemaTableKey]bool,
) {
	for i := range result {
		key := p.schemaTableKeyFor(result[i].SourceSchema, result[i].SourceTable)
		if !pkCoveredTables[key] {
			continue
		}
		result[i].Nullable = p.getBaseColumnNullability(key.schema, key.table, result[i].SourceColumn)
	}
}

// getBaseColumnNullability looks up the base nullability of a column from the catalogue
// in a specific schema, so a table name shared across schemas resolves deterministically.
//
// Takes schemaName (string) which identifies the resolved schema.
// Takes tableName (string) which identifies the table.
// Takes columnName (string) which identifies the column.
//
// Returns bool which is true if the column is nullable, defaulting to true when not
// found.
func (p *nullabilityPropagator) getBaseColumnNullability(schemaName string, tableName string, columnName string) bool {
	schema, exists := p.catalogue.Schemas[schemaName]
	if !exists {
		return true
	}
	table, exists := schema.Tables[tableName]
	if !exists {
		return true
	}
	for i := range table.Columns {
		if table.Columns[i].Name == columnName {
			return table.Columns[i].Nullable
		}
	}
	return true
}

// isPrimaryKeyFullyCovered checks whether the given column names include all primary key
// columns of the specified table.
//
// Takes schemaName (string) which identifies the schema.
// Takes tableName (string) which identifies the table.
// Takes groupByColumns ([]string) which holds the column names to check against the
// primary key.
//
// Returns bool which is true if all primary key columns are present in the
// groupByColumns.
func (p *nullabilityPropagator) isPrimaryKeyFullyCovered(
	schemaName string,
	tableName string,
	groupByColumns []string,
) bool {
	resolvedSchema := schemaName
	if resolvedSchema == "" {
		resolvedSchema = p.catalogue.DefaultSchema
	}

	schema, exists := p.catalogue.Schemas[resolvedSchema]
	if !exists {
		return false
	}

	table, exists := schema.Tables[tableName]
	if !exists {
		return false
	}

	if len(table.PrimaryKey) == 0 {
		return false
	}

	groupBySet := make(map[string]struct{}, len(groupByColumns))
	for _, column := range groupByColumns {
		groupBySet[column] = struct{}{}
	}

	for _, primaryKeyColumn := range table.PrimaryKey {
		if _, covered := groupBySet[primaryKeyColumn]; !covered {
			return false
		}
	}

	return true
}
