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
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// maxScopeChainDepth caps how deep ResolveColumn and ExpandStar walk the parent chain.
	//
	// Hand-written SQL nests subqueries no more than a handful of levels in practice; the 64
	// limit mirrors maxInferExpressionNameDepth in type_resolver.go so a malformed catalogue
	// or maliciously crafted parser output cannot blow the Go stack here.
	maxScopeChainDepth = 64
)

var (
	// errScopeChainDepthExceeded is returned by the qualified and unqualified resolvers when
	// the parent-chain traversal reaches maxScopeChainDepth. Callers see this as an
	// unknown-column failure so the user message points at the lookup site rather than the
	// depth limit.
	errScopeChainDepthExceeded = errors.New("scope chain parent traversal exceeded depth limit")
)

// scopeChain provides nested scope resolution for column references. Each scope can
// contain tables, CTEs, and LATERAL-visible tables, and can delegate to a parent scope
// for correlated subqueries and LATERAL joins.
//
// This enables correct column resolution across CTEs, subqueries, and LATERAL joins using
// nested scopes rather than a flat lookup.
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

// newScopeChain creates a new scope with the given kind and optional parent.
//
// Takes kind (querier_dto.ScopeKind) which specifies the scope kind (query, subquery, or
// lateral).
// Takes parent (*scopeChain) which specifies the enclosing scope, or nil for a root
// scope.
//
// Returns *scopeChain which holds the initialised scope with empty tables and CTEs.
func newScopeChain(kind querier_dto.ScopeKind, parent *scopeChain) *scopeChain {
	return &scopeChain{
		parent: parent,
		tables: make(map[string]*querier_dto.ScopedTable),
		ctes:   make(map[string]*resolvedCTE),
		kind:   kind,
	}
}

// AddTable registers a table in the current scope with JOIN-adjusted nullability. The
// catalogue table provides the base column types and nullability, which are then adjusted
// based on the join kind.
//
// Takes table (querier_dto.TableReference) which specifies the table name, schema, and
// alias.
// Takes joinKind (querier_dto.JoinKind) which specifies the join type for nullability
// adjustment.
// Takes catalogueTable (*querier_dto.Table) which specifies the catalogue entry with base
// column definitions.
//
// Returns error which indicates a registration failure, currently always nil.
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
			SQLType:  arrayWrappedSQLType(&catalogueTable.Columns[i]),
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

// AddCTE registers a resolved CTE in the current scope.
//
// Takes name (string) which specifies the CTE name as declared in the WITH clause.
// Takes columns ([]querier_dto.ScopedColumn) which specifies the resolved output columns
// of the CTE.
func (s *scopeChain) AddCTE(name string, columns []querier_dto.ScopedColumn) {
	s.ctes[strings.ToLower(name)] = &resolvedCTE{
		name:    name,
		columns: columns,
	}
}

// AddCTEAsTable registers a CTE referenced from a FROM or JOIN clause as a scoped table
// under the given alias.
//
// Unqualified column lookups walk s.tables first, so promoting the CTE here makes
// unqualified resolution deterministic when multiple CTEs are in scope but only some are
// actually referenced. Without this, resolveFromCTEsAndLateral iterates every CTE in the
// scope and the first one with a matching column name wins, which is both
// non-deterministic (map iteration) and semantically wrong.
//
// Takes alias (string) which specifies the alias the CTE was referenced under, or the
// CTE's own name when no AS clause was used.
// Takes columns ([]querier_dto.ScopedColumn) which specifies the resolved output columns
// of the CTE.
// Takes joinKind (querier_dto.JoinKind) which specifies the join kind so nullability
// propagates correctly when the CTE participates in an outer JOIN.
//
// The supplied columns slice is always deep-copied before being stored. A CTE referenced
// as a table shares its column slice with the CTE's own definition
// (s.ctes[name].columns); if it were stored by reference, a later RIGHT/FULL/POSITIONAL
// join that flips Nullable on the existing table's columns in place would corrupt the
// CTE's stored nullability for every subsequent reference. Outer joins additionally set
// Nullable on the copy.
func (s *scopeChain) AddCTEAsTable(alias string, columns []querier_dto.ScopedColumn, joinKind querier_dto.JoinKind) {
	scoped := make([]querier_dto.ScopedColumn, len(columns))
	copy(scoped, columns)
	if joinKind == querier_dto.JoinLeft || joinKind == querier_dto.JoinFull || joinKind == querier_dto.JoinPositional {
		for i := range scoped {
			scoped[i].Nullable = true
		}
	}
	s.tables[alias] = &querier_dto.ScopedTable{
		Name:     alias,
		Alias:    alias,
		Columns:  scoped,
		JoinKind: joinKind,
	}
}

// AddDerivedTable registers a virtual table (from UNNEST, FLATTEN, table-valued
// functions, or subqueries in FROM) in the current scope. Derived tables are resolved
// identically to catalogue tables.
//
// Takes reference (querier_dto.DerivedTableReference) which specifies the derived table
// alias, columns, and join kind.
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

// ResolveColumn walks the scope chain to find a column by optional table alias and column
// name.
//
// When tableAlias is set, the named table is sought in the current scope and, for LATERAL
// or subquery scopes, in the parent when absent. When tableAlias is empty, every table is
// searched for the column name, requiring exactly one match because multiple matches
// produce Q002 (ambiguity). CTEs in the current scope are checked next, then LATERAL or
// subquery scopes traverse to the parent and repeat. A column not found anywhere produces
// Q001 (unknown column).
//
// Takes tableAlias (string) which specifies the qualifying table alias, or empty for
// unqualified lookup.
// Takes columnName (string) which specifies the column name to resolve.
//
// Returns *querier_dto.ScopedColumn which holds the resolved column, or nil on error.
// Returns *querier_dto.ScopedTable which holds the containing table, or nil on error.
// Returns error which indicates a resolution failure (Q001 unknown, Q002 ambiguous).
func (s *scopeChain) ResolveColumn(
	tableAlias string,
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	if tableAlias != "" {
		return s.resolveQualifiedColumn(tableAlias, columnName, 0)
	}
	return s.resolveUnqualifiedColumn(columnName, 0)
}

// ExpandStar expands a SELECT * or table.* into all visible columns from the scope. If
// tableAlias is non-empty, only columns from that table are returned.
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

	tableAliases := slices.Sorted(maps.Keys(s.tables))

	var allColumns []querier_dto.ScopedColumn
	for _, alias := range tableAliases {
		allColumns = append(allColumns, s.tables[alias].Columns...)
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

// MarkLateralVisible makes specified tables from the parent scope visible to LATERAL
// subqueries in this scope.
//
// Takes tables ([]*querier_dto.ScopedTable) which specifies the parent-scope tables to
// make laterally visible.
func (s *scopeChain) MarkLateralVisible(tables []*querier_dto.ScopedTable) {
	s.lateralVisible = append(s.lateralVisible, tables...)
}

// resolveQualifiedColumn resolves a column reference that includes a table alias
// qualifier.
//
// It searches the current scope tables, CTEs, lateral-visible tables, and parent scopes
// in order. The depth counter caps the parent-chain traversal at maxScopeChainDepth to
// defend against runaway recursion on a malformed scope chain.
//
// Takes tableAlias (string) which specifies the qualifying table alias.
// Takes columnName (string) which specifies the column name to find.
// Takes depth (int) which holds the current parent-traversal depth. The root call passes
// zero and each recursive parent call increments by one.
//
// Returns *querier_dto.ScopedColumn which holds the resolved column, or nil on error.
// Returns *querier_dto.ScopedTable which holds the containing table, or nil on error.
// Returns error which indicates Q001 if the table alias or column is unknown, or
// errScopeChainDepthExceeded when the parent chain ran past maxScopeChainDepth.
func (s *scopeChain) resolveQualifiedColumn(
	tableAlias string,
	columnName string,
	depth int,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	if depth >= maxScopeChainDepth {
		return nil, nil, errScopeChainDepthExceeded
	}
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
		return s.parent.resolveQualifiedColumn(tableAlias, columnName, depth+1)
	}

	return nil, nil, fmt.Errorf("%s: unknown table or alias %q", querier_dto.CodeUnknownColumn, tableAlias)
}

// resolveColumnInTable searches for a column by name within a single scoped table. Falls
// back to synthesising an implicit rowid column for eligible tables.
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

// resolveColumnInCTE searches for a column by name within a resolved CTE.
//
// Takes cte (*resolvedCTE) which specifies the CTE to search.
// Takes columnName (string) which specifies the column name to find.
// Takes tableAlias (string) which specifies the alias used in error messages.
//
// Returns *querier_dto.ScopedColumn which holds the matched column, or nil on error.
// Returns *querier_dto.ScopedTable which holds a synthetic scoped table for the CTE, or
// nil on error.
// Returns error which indicates Q001 if the column is not found.
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

// resolveColumnInLateral searches for a qualified column in the lateral-visible tables.
//
// Takes lateralVisible ([]*querier_dto.ScopedTable) which specifies the tables visible
// via LATERAL.
// Takes tableAlias (string) which specifies the qualifying table alias.
// Takes columnName (string) which specifies the column name to find.
//
// Returns *querier_dto.ScopedColumn which holds the matched column, or nil if not found.
// Returns *querier_dto.ScopedTable which holds the containing table, or nil if not found.
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
//
// It searches tables, implicit rowid, CTEs, lateral-visible tables, and parent scopes.
// The depth counter caps the parent-chain traversal at maxScopeChainDepth to defend
// against runaway recursion on a malformed scope chain.
//
// Takes columnName (string) which specifies the column name to resolve.
// Takes depth (int) which holds the current parent-traversal depth. The root call passes
// zero and each recursive parent call increments by one.
//
// Returns *querier_dto.ScopedColumn which holds the resolved column, or nil on error.
// Returns *querier_dto.ScopedTable which holds the containing table, or nil on error.
// Returns error which indicates Q001 (unknown), Q002 (ambiguous), or
// errScopeChainDepthExceeded when the parent chain ran past maxScopeChainDepth.
func (s *scopeChain) resolveUnqualifiedColumn(
	columnName string,
	depth int,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable, error) {
	if depth >= maxScopeChainDepth {
		return nil, nil, errScopeChainDepthExceeded
	}
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
		return s.parent.resolveUnqualifiedColumn(columnName, depth+1)
	}

	return nil, nil, fmt.Errorf("%s: unknown column %q", querier_dto.CodeUnknownColumn, columnName)
}

// findColumnInTables searches all tables in the current scope for a column by name.
// Returns the match count to detect ambiguous references.
//
// Takes columnName (string) which specifies the column name to search for.
//
// Returns *querier_dto.ScopedColumn which holds the last matched column, or nil if none
// found.
// Returns *querier_dto.ScopedTable which holds the last matched table, or nil if none
// found.
// Returns int which holds the number of tables containing a matching column.
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

// resolveImplicitRowID resolves an implicit rowid column across all tables in the current
// scope. Exactly one eligible table must exist; multiple eligible tables produce Q002
// (ambiguity).
//
// Takes columnName (string) which specifies the rowid alias (ROWID, _ROWID_, or OID).
//
// Returns *querier_dto.ScopedColumn which holds the synthesised rowid column, or nil if
// none found.
// Returns *querier_dto.ScopedTable which holds the containing table, or nil if none
// found.
// Returns error which indicates Q002 if multiple tables support implicit rowid.
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

// resolveFromCTEsAndLateral searches CTEs and lateral-visible tables for an unqualified
// column.
//
// Takes columnName (string) which specifies the column name to find.
//
// Returns *querier_dto.ScopedColumn which holds the matched column, or nil if not found.
// Returns *querier_dto.ScopedTable which holds the containing table or CTE, or nil if not
// found.
func (s *scopeChain) resolveFromCTEsAndLateral(
	columnName string,
) (*querier_dto.ScopedColumn, *querier_dto.ScopedTable) {
	cteNames := slices.Sorted(maps.Keys(s.ctes))
	for _, name := range cteNames {
		cte := s.ctes[name]
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

// isImplicitRowID reports whether the given column name is a SQLite implicit rowid alias.
//
// Takes name (string) which specifies the column name to check.
//
// Returns bool which indicates true if the name is ROWID, _ROWID_, or OID
// (case-insensitive).
func isImplicitRowID(name string) bool {
	upper := strings.ToUpper(name)
	return upper == "ROWID" || upper == "_ROWID_" || upper == "OID"
}

// arrayWrappedSQLType returns the column's SQLType with array wrapping applied when the
// catalogue captured the column as an array (e.g. `tokens TEXT[]`).
//
// The catalogue stores arrays as a separate Column.IsArray flag and ArrayDimensions
// counter rather than folding them into SQLType, so callers that propagate scope columns
// downstream (parameters bound to array columns, derived-table column types, and so on)
// must reconstitute the Array category here or the downstream type mapper falls back to
// the scalar element type.
//
// Takes column (*querier_dto.Column) which is the catalogue column being scoped. Passed
// by pointer because Column is a heavy value type and copying it on every call shows up
// in the lint budget.
//
// Returns querier_dto.SQLType which is either the original SQLType or an Array-wrapped
// variant carrying the original as ElementType.
func arrayWrappedSQLType(column *querier_dto.Column) querier_dto.SQLType {
	if !column.IsArray || column.ArrayDimensions <= 0 {
		return column.SQLType
	}

	dimensions := min(column.ArrayDimensions, maxArrayDimensions)
	wrapped := column.SQLType
	for range dimensions {
		inner := wrapped
		wrapped = querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  inner.EngineName + "[]",
			ElementType: &inner,
		}
	}
	return wrapped
}
