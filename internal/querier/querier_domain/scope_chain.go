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
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// scopeChain provides nested scope resolution for column references. Each
// scope can contain tables, CTEs, and LATERAL-visible tables, and can
// delegate to a parent scope for correlated subqueries and LATERAL joins.
//
// This enables correct column resolution across CTEs, subqueries, and
// LATERAL joins using nested scopes rather than a flat lookup.
type scopeChain struct {
	// parent holds the enclosing scope for correlated subqueries and LATERAL joins.
	parent *scopeChain

	// tables holds the tables registered in this scope, keyed by alias.
	tables map[string]*querier_dto.ScopedTable

	// ctes holds the resolved CTEs in this scope, keyed by lowercase name.
	ctes map[string]*resolvedCTE

	// lateralVisible holds parent-scope tables made visible for LATERAL references.
	lateralVisible []*querier_dto.ScopedTable

	// kind holds the scope kind (query, subquery, or lateral).
	kind querier_dto.ScopeKind
}

// resolvedCTE holds a CTE with its output columns fully resolved.
type resolvedCTE struct {
	// name holds the original CTE name as declared in the WITH clause.
	name string

	// columns holds the resolved output columns of the CTE.
	columns []querier_dto.ScopedColumn
}

// newScopeChain creates a new scope with the given kind
// and optional parent.
//
// Takes kind (querier_dto.ScopeKind) which specifies
// the scope kind (query, subquery, or lateral).
// Takes parent (*scopeChain) which specifies the
// enclosing scope, or nil for a root scope.
//
// Returns *scopeChain which holds the initialised scope
// with empty tables and CTEs.
func newScopeChain(kind querier_dto.ScopeKind, parent *scopeChain) *scopeChain {
	return &scopeChain{
		parent: parent,
		tables: make(map[string]*querier_dto.ScopedTable),
		ctes:   make(map[string]*resolvedCTE),
		kind:   kind,
	}
}

// AddTable registers a table in the current scope with
// JOIN-adjusted nullability. The catalogue table
// provides the base column types and nullability, which
// are then adjusted based on the join kind.
//
// Takes table (querier_dto.TableReference) which
// specifies the table name, schema, and alias.
// Takes joinKind (querier_dto.JoinKind) which specifies
// the join type for nullability adjustment.
// Takes catalogueTable (*querier_dto.Table) which
// specifies the catalogue entry with base column
// definitions.
//
// Returns error which indicates a registration failure,
// currently always nil.
func (s *scopeChain) AddTable(
	table querier_dto.TableReference,
	joinKind querier_dto.JoinKind,
	catalogueTable *querier_dto.Table,
) error {
	alias := table.Alias
	if alias == "" {
		alias = table.Name
	}

	columns := make([]querier_dto.ScopedColumn, len(catalogueTable.Columns))
	for i := range catalogueTable.Columns {
		nullable := catalogueTable.Columns[i].Nullable
		if joinKind == querier_dto.JoinLeft || joinKind == querier_dto.JoinFull || joinKind == querier_dto.JoinPositional {
			nullable = true
		}
		columns[i] = querier_dto.ScopedColumn{
			Name:     catalogueTable.Columns[i].Name,
			SQLType:  catalogueTable.Columns[i].SQLType,
			Nullable: nullable,
		}
	}

	if joinKind == querier_dto.JoinRight || joinKind == querier_dto.JoinFull || joinKind == querier_dto.JoinPositional {
		for _, existingTable := range s.tables {
			for i := range existingTable.Columns {
				existingTable.Columns[i].Nullable = true
			}
		}
	}

	s.tables[alias] = &querier_dto.ScopedTable{
		Schema:         table.Schema,
		Name:           table.Name,
		Alias:          alias,
		Columns:        columns,
		JoinKind:       joinKind,
		IsWithoutRowID: catalogueTable.IsWithoutRowID || catalogueTable.IsVirtual,
	}

	return nil
}

// AddCTE registers a resolved CTE in the current
// scope.
//
// Takes name (string) which specifies the CTE name as
// declared in the WITH clause.
// Takes columns ([]querier_dto.ScopedColumn) which
// specifies the resolved output columns of the CTE.
func (s *scopeChain) AddCTE(name string, columns []querier_dto.ScopedColumn) {
	s.ctes[strings.ToLower(name)] = &resolvedCTE{
		name:    name,
		columns: columns,
	}
}

// AddDerivedTable registers a virtual table (from
// UNNEST, FLATTEN, table-valued functions, or
// subqueries in FROM) in the current scope. Derived
// tables are resolved identically to catalogue tables.
//
// Takes reference (querier_dto.DerivedTableReference)
// which specifies the derived table alias, columns, and
// join kind.
func (s *scopeChain) AddDerivedTable(reference querier_dto.DerivedTableReference) {
	columns := reference.Columns
	if reference.JoinKind == querier_dto.JoinLeft || reference.JoinKind == querier_dto.JoinFull || reference.JoinKind == querier_dto.JoinPositional {
		columns = make([]querier_dto.ScopedColumn, len(reference.Columns))
		copy(columns, reference.Columns)
		for i := range columns {
			columns[i].Nullable = true
		}
	}
	s.tables[reference.Alias] = &querier_dto.ScopedTable{
		Alias:    reference.Alias,
		Columns:  columns,
		JoinKind: reference.JoinKind,
	}
}

// ResolveColumn walks the scope chain to find a column
// by optional table alias and column name.
//
// Resolution algorithm:
//  1. If tableAlias is set, find that table in current
//     scope; if not found and scope is LATERAL/subquery,
//     search parent.
//  2. If tableAlias is empty, search all tables for the
//     column name. Exactly one match is required;
//     multiple matches produce Q002 (ambiguity).
//  3. Check CTEs in current scope.
//  4. For LATERAL/subquery scopes, traverse to parent
//     and repeat.
//  5. Not found anywhere produces Q001 (unknown column).
//
// Takes tableAlias (string) which specifies the
// qualifying table alias, or empty for unqualified
// lookup.
// Takes columnName (string) which specifies the column
// name to resolve.
//
// Returns *querier_dto.ScopedColumn which holds the
// resolved column, or nil on error.
// Returns *querier_dto.ScopedTable which holds the
// containing table, or nil on error.
// Returns error which indicates a resolution failure
// (Q001 unknown, Q002 ambiguous).
func (s *scopeChain) ResolveColumn(
	tableAlias string,
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	if tableAlias != "" {
		return s.resolveQualifiedColumn(tableAlias, columnName)
	}
	return s.resolveUnqualifiedColumn(columnName)
}

// ExpandStar expands a SELECT * or table.* into all visible columns from the
// scope. If tableAlias is non-empty, only columns from that table are returned.
//
// Takes tableAlias (string) which specifies the table to expand, or empty for all tables.
//
// Returns []querier_dto.ScopedColumn which holds the expanded columns.
// Returns error which indicates Q003 if the specified table alias is unknown.
func (s *scopeChain) ExpandStar(tableAlias string) ([]querier_dto.ScopedColumn, error) {
	if tableAlias != "" {
		if table, exists := s.tables[tableAlias]; exists {
			result := make([]querier_dto.ScopedColumn, len(table.Columns))
			copy(result, table.Columns)
			return result, nil
		}
		if cte, exists := s.ctes[strings.ToLower(tableAlias)]; exists {
			result := make([]querier_dto.ScopedColumn, len(cte.columns))
			copy(result, cte.columns)
			return result, nil
		}
		return nil, fmt.Errorf("%s: unknown table %q in SELECT *", querier_dto.CodeUnknownTable, tableAlias)
	}

	var allColumns []querier_dto.ScopedColumn
	for _, table := range s.tables {
		allColumns = append(allColumns, table.Columns...)
	}
	return allColumns, nil
}

// CreateChildScope creates a new scope linked to this one as parent.
//
// Takes kind (querier_dto.ScopeKind) which specifies the child scope kind.
//
// Returns *scopeChain which holds the new child scope with this scope as parent.
func (s *scopeChain) CreateChildScope(kind querier_dto.ScopeKind) *scopeChain {
	return newScopeChain(kind, s)
}

// MarkLateralVisible makes specified tables from the
// parent scope visible to LATERAL subqueries in this
// scope.
//
// Takes tables ([]*querier_dto.ScopedTable) which
// specifies the parent-scope tables to make laterally
// visible.
func (s *scopeChain) MarkLateralVisible(tables []*querier_dto.ScopedTable) {
	s.lateralVisible = append(s.lateralVisible, tables...)
}

// resolveQualifiedColumn resolves a column reference
// that includes a table alias qualifier. Searches the
// current scope tables, CTEs, lateral-visible tables,
// and parent scopes in order.
//
// Takes tableAlias (string) which specifies the
// qualifying table alias.
// Takes columnName (string) which specifies the column
// name to find.
//
// Returns *querier_dto.ScopedColumn which holds the
// resolved column, or nil on error.
// Returns *querier_dto.ScopedTable which holds the
// containing table, or nil on error.
// Returns error which indicates Q001 if the table alias
// or column is unknown.
func (s *scopeChain) resolveQualifiedColumn(
	tableAlias string,
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	if table, exists := s.tables[tableAlias]; exists {
		return resolveColumnInTable(table, columnName, tableAlias)
	}

	if cte, exists := s.ctes[strings.ToLower(tableAlias)]; exists {
		return resolveColumnInCTE(cte, columnName, tableAlias)
	}

	column, table := resolveColumnInLateral(s.lateralVisible, tableAlias, columnName)
	if column != nil {
		return column, table, nil
	}

	if s.parent != nil && (s.kind == querier_dto.ScopeKindSubquery || s.kind == querier_dto.ScopeKindLateral) {
		return s.parent.resolveQualifiedColumn(tableAlias, columnName)
	}

	return nil, nil, fmt.Errorf("%s: unknown table or alias %q", querier_dto.CodeUnknownColumn, tableAlias)
}

// resolveColumnInTable searches for a column by name within a single scoped table.
// Falls back to synthesising an implicit rowid column for eligible tables.
//
// Takes table (*querier_dto.ScopedTable) which specifies the table to search.
// Takes columnName (string) which specifies the column name to find.
// Takes tableAlias (string) which specifies the alias used in error messages.
//
// Returns *querier_dto.ScopedColumn which holds the matched column, or nil on error.
// Returns *querier_dto.ScopedTable which holds the containing table, or nil on error.
// Returns error which indicates Q001 if the column is not found.
func resolveColumnInTable(
	table *querier_dto.ScopedTable,
	columnName string,
	tableAlias string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	for i := range table.Columns {
		if strings.EqualFold(table.Columns[i].Name, columnName) {
			return &table.Columns[i], table, nil
		}
	}
	if isImplicitRowID(columnName) && !table.IsWithoutRowID {
		rowidColumn := querier_dto.ScopedColumn{
			Name:     columnName,
			SQLType:  querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger},
			Nullable: false,
		}
		return &rowidColumn, table, nil
	}
	return nil, nil, fmt.Errorf("%s: unknown column %q in table %q", querier_dto.CodeUnknownColumn, columnName, tableAlias)
}

// resolveColumnInCTE searches for a column by name
// within a resolved CTE.
//
// Takes cte (*resolvedCTE) which specifies the CTE to
// search.
// Takes columnName (string) which specifies the column
// name to find.
// Takes tableAlias (string) which specifies the alias
// used in error messages.
//
// Returns *querier_dto.ScopedColumn which holds the
// matched column, or nil on error.
// Returns *querier_dto.ScopedTable which holds a
// synthetic scoped table for the CTE, or nil on error.
// Returns error which indicates Q001 if the column is
// not found.
func resolveColumnInCTE(
	cte *resolvedCTE,
	columnName string,
	tableAlias string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	for i := range cte.columns {
		if strings.EqualFold(cte.columns[i].Name, columnName) {
			cteTable := &querier_dto.ScopedTable{
				Name:    cte.name,
				Alias:   cte.name,
				Columns: cte.columns,
			}
			return &cte.columns[i], cteTable, nil
		}
	}
	return nil, nil, fmt.Errorf("%s: unknown column %q in CTE %q", querier_dto.CodeUnknownColumn, columnName, tableAlias)
}

// resolveColumnInLateral searches for a qualified
// column in the lateral-visible tables.
//
// Takes lateralVisible ([]*querier_dto.ScopedTable)
// which specifies the tables visible via LATERAL.
// Takes tableAlias (string) which specifies the
// qualifying table alias.
// Takes columnName (string) which specifies the column
// name to find.
//
// Returns *querier_dto.ScopedColumn which holds the
// matched column, or nil if not found.
// Returns *querier_dto.ScopedTable which holds the
// containing table, or nil if not found.
func resolveColumnInLateral(
	lateralVisible []*querier_dto.ScopedTable,
	tableAlias string,
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable) {
	for _, lateralTable := range lateralVisible {
		if lateralTable.Alias != tableAlias && lateralTable.Name != tableAlias {
			continue
		}
		for i := range lateralTable.Columns {
			if strings.EqualFold(lateralTable.Columns[i].Name, columnName) {
				return &lateralTable.Columns[i], lateralTable
			}
		}
	}
	return nil, nil
}

// resolveUnqualifiedColumn resolves a column reference without a table qualifier.
// Searches tables, implicit rowid, CTEs, lateral-visible tables, and parent scopes.
//
// Takes columnName (string) which specifies the column name to resolve.
//
// Returns *querier_dto.ScopedColumn which holds the resolved column, or nil on error.
// Returns *querier_dto.ScopedTable which holds the containing table, or nil on error.
// Returns error which indicates Q001 (unknown) or Q002 (ambiguous) column.
func (s *scopeChain) resolveUnqualifiedColumn(
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	column, table, matchCount := s.findColumnInTables(columnName)
	if matchCount == 1 {
		return column, table, nil
	}
	if matchCount > 1 {
		return nil, nil, fmt.Errorf("%s: ambiguous column reference %q", querier_dto.CodeAmbiguousColumn, columnName)
	}

	if isImplicitRowID(columnName) {
		column, table, err := s.resolveImplicitRowID(columnName)
		if column != nil || err != nil {
			return column, table, err
		}
	}

	column, table = s.resolveFromCTEsAndLateral(columnName)
	if column != nil {
		return column, table, nil
	}

	if s.parent != nil && (s.kind == querier_dto.ScopeKindSubquery || s.kind == querier_dto.ScopeKindLateral) {
		return s.parent.resolveUnqualifiedColumn(columnName)
	}

	return nil, nil, fmt.Errorf("%s: unknown column %q", querier_dto.CodeUnknownColumn, columnName)
}

// findColumnInTables searches all tables in the current
// scope for a column by name. Returns the match count
// to detect ambiguous references.
//
// Takes columnName (string) which specifies the column
// name to search for.
//
// Returns *querier_dto.ScopedColumn which holds the
// last matched column, or nil if none found.
// Returns *querier_dto.ScopedTable which holds the last
// matched table, or nil if none found.
// Returns int which holds the number of tables
// containing a matching column.
func (s *scopeChain) findColumnInTables(
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, int) {
	var foundColumn *querier_dto.ScopedColumn
	var foundTable *querier_dto.ScopedTable
	matchCount := 0

	for _, table := range s.tables {
		for i := range table.Columns {
			if strings.EqualFold(table.Columns[i].Name, columnName) {
				foundColumn = &table.Columns[i]
				foundTable = table
				matchCount++
			}
		}
	}

	return foundColumn, foundTable, matchCount
}

// resolveImplicitRowID resolves an implicit rowid
// column across all tables in the current scope.
// Exactly one eligible table must exist; multiple
// eligible tables produce Q002 (ambiguity).
//
// Takes columnName (string) which specifies the rowid
// alias (ROWID, _ROWID_, or OID).
//
// Returns *querier_dto.ScopedColumn which holds the
// synthesised rowid column, or nil if none found.
// Returns *querier_dto.ScopedTable which holds the
// containing table, or nil if none found.
// Returns error which indicates Q002 if multiple tables
// support implicit rowid.
func (s *scopeChain) resolveImplicitRowID(
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	rowidMatchCount := 0
	var rowidTable *querier_dto.ScopedTable
	for _, table := range s.tables {
		if !table.IsWithoutRowID {
			rowidMatchCount++
			rowidTable = table
		}
	}
	if rowidMatchCount == 1 {
		rowidColumn := querier_dto.ScopedColumn{
			Name:     columnName,
			SQLType:  querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger},
			Nullable: false,
		}
		return &rowidColumn, rowidTable, nil
	}
	if rowidMatchCount > 1 {
		return nil, nil, fmt.Errorf("%s: ambiguous column reference %q", querier_dto.CodeAmbiguousColumn, columnName)
	}
	return nil, nil, nil
}

// resolveFromCTEsAndLateral searches CTEs and
// lateral-visible tables for an unqualified column.
//
// Takes columnName (string) which specifies the column
// name to find.
//
// Returns *querier_dto.ScopedColumn which holds the
// matched column, or nil if not found.
// Returns *querier_dto.ScopedTable which holds the
// containing table or CTE, or nil if not found.
func (s *scopeChain) resolveFromCTEsAndLateral(
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable) {
	for _, cte := range s.ctes {
		for i := range cte.columns {
			if strings.EqualFold(cte.columns[i].Name, columnName) {
				cteTable := &querier_dto.ScopedTable{
					Name:    cte.name,
					Alias:   cte.name,
					Columns: cte.columns,
				}
				return &cte.columns[i], cteTable
			}
		}
	}

	for _, lateralTable := range s.lateralVisible {
		for i := range lateralTable.Columns {
			if strings.EqualFold(lateralTable.Columns[i].Name, columnName) {
				return &lateralTable.Columns[i], lateralTable
			}
		}
	}

	return nil, nil
}

// isImplicitRowID reports whether the given column name
// is a SQLite implicit rowid alias.
//
// Takes name (string) which specifies the column name
// to check.
//
// Returns bool which indicates true if the name is
// ROWID, _ROWID_, or OID (case-insensitive).
func isImplicitRowID(name string) bool {
	upper := strings.ToUpper(name)
	return upper == "ROWID" || upper == "_ROWID_" || upper == "OID"
}
