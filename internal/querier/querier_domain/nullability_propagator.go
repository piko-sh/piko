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

// nullabilityPropagator applies final nullability adjustments to output
// columns and parameters after type resolution. This handles directive
// overrides, GROUP BY primary key functional dependency rules (SQL:2003),
// and parameter kind-specific nullability.
type nullabilityPropagator struct {
	// catalogue holds the schema state for primary key
	// lookups.
	catalogue *querier_dto.Catalogue
}

// newNullabilityPropagator creates a new nullability
// propagator with the given catalogue for primary key
// lookup.
//
// Takes catalogue (*querier_dto.Catalogue) which provides
// the schema state for primary key lookups.
//
// Returns *nullabilityPropagator which is ready to apply
// nullability adjustments.
func newNullabilityPropagator(catalogue *querier_dto.Catalogue) *nullabilityPropagator {
	return &nullabilityPropagator{
		catalogue: catalogue,
	}
}

// PropagateOutputNullability applies final nullability
// adjustments to output columns based on directives and
// GROUP BY functional dependency rules.
//
// Adjustments applied in order:
//  1. piko.nullable directive override
//  2. GROUP BY primary key rule
//
// Takes columns ([]querier_dto.OutputColumn) which holds
// the output columns to adjust.
// Takes queryDirectives (*querier_dto.QueryDirectives)
// which holds any nullable override directive.
// Takes scope (*scopeChain) which provides table alias
// resolution.
// Takes groupByColumns ([]querier_dto.ColumnReference)
// which holds the GROUP BY column references.
//
// Returns []querier_dto.OutputColumn which holds the
// adjusted output columns.
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

// PropagateParameterNullability applies final nullability
// adjustments to parameters based on their parameter kind
// declarations.
//
// Rules:
//   - piko.optional -> always nullable
//   - piko.limit -> always NOT NULL integer
//   - piko.offset -> always NOT NULL integer
//   - piko.slice -> always NOT NULL
//   - piko.param with nullable:true/false -> override
//
// Takes parameters ([]querier_dto.QueryParameter) which
// holds the parameters to adjust.
// Takes parameterDirectives
// ([]*querier_dto.ParameterDirective) which holds the
// directive declarations.
//
// Returns []querier_dto.QueryParameter which holds the
// adjusted parameters.
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
		switch directive.Kind {
		case querier_dto.ParameterDirectiveOptional:
			result[i].Nullable = true
			result[i].IsOptional = true
		case querier_dto.ParameterDirectiveSlice:
			result[i].Nullable = false
			result[i].IsSlice = true
		case querier_dto.ParameterDirectiveSortable:
			result[i].Nullable = false
			result[i].SortableColumns = directive.Columns
		case querier_dto.ParameterDirectiveLimit, querier_dto.ParameterDirectiveOffset:
			result[i].Nullable = false
		case querier_dto.ParameterDirectiveParam:
			if directive.Nullable != nil {
				result[i].Nullable = *directive.Nullable
			}
		}
	}

	return result
}

// findPrimaryKeyCoveredTables returns a set of table names
// whose full primary key is covered by the GROUP BY columns.
//
// Takes groupByColumns ([]querier_dto.ColumnReference)
// which holds the GROUP BY column references.
// Takes scope (*scopeChain) which provides table alias
// resolution.
//
// Returns map[string]bool which maps table names to true
// when their primary key is fully covered.
func (p *nullabilityPropagator) findPrimaryKeyCoveredTables(
	groupByColumns []querier_dto.ColumnReference,
	scope *scopeChain,
) map[string]bool {
	tableGroupedColumns := make(map[string][]string)
	for _, groupByColumn := range groupByColumns {
		tableGroupedColumns[groupByColumn.TableAlias] = append(
			tableGroupedColumns[groupByColumn.TableAlias], groupByColumn.ColumnName)
	}

	pkCoveredTables := make(map[string]bool)
	for tableAlias, groupedColumns := range tableGroupedColumns {
		scopedTable, exists := scope.tables[tableAlias]
		if !exists {
			continue
		}
		schemaName := scopedTable.Schema
		if schemaName == "" {
			schemaName = p.catalogue.DefaultSchema
		}
		if p.isPrimaryKeyFullyCovered(schemaName, scopedTable.Name, groupedColumns) {
			pkCoveredTables[scopedTable.Name] = true
		}
	}

	return pkCoveredTables
}

// applyGroupByNullability restores base column nullability
// for tables whose primary key is fully covered by GROUP BY.
//
// Takes result ([]querier_dto.OutputColumn) which holds the
// output columns to adjust in place.
// Takes pkCoveredTables (map[string]bool) which identifies
// tables with fully covered primary keys.
func (p *nullabilityPropagator) applyGroupByNullability(
	result []querier_dto.OutputColumn,
	pkCoveredTables map[string]bool,
) {
	for i := range result {
		if !pkCoveredTables[result[i].SourceTable] {
			continue
		}
		baseNullable := p.getBaseColumnNullability(result[i].SourceTable, result[i].SourceColumn)
		result[i].Nullable = baseNullable || result[i].Nullable
	}
}

// getBaseColumnNullability looks up the base nullability
// of a column from the catalogue by table and column name.
//
// Takes tableName (string) which identifies the table.
// Takes columnName (string) which identifies the column.
//
// Returns bool which is true if the column is nullable,
// defaulting to true when not found.
func (p *nullabilityPropagator) getBaseColumnNullability(tableName string, columnName string) bool {
	for _, schema := range p.catalogue.Schemas {
		table, exists := schema.Tables[tableName]
		if !exists {
			continue
		}
		for i := range table.Columns {
			if table.Columns[i].Name == columnName {
				return table.Columns[i].Nullable
			}
		}
	}
	return true
}

// isPrimaryKeyFullyCovered checks whether the given column
// names include all primary key columns of the specified
// table.
//
// Takes schemaName (string) which identifies the schema.
// Takes tableName (string) which identifies the table.
// Takes groupByColumns ([]string) which holds the column
// names to check against the primary key.
//
// Returns bool which is true if all primary key columns
// are present in the groupByColumns.
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
