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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func catalogueColumn(name string, category querier_dto.SQLTypeCategory, nullable bool) querier_dto.Column {
	return querier_dto.Column{
		Name:     name,
		SQLType:  querier_dto.SQLType{EngineName: strings.ToLower(name), Category: category},
		Nullable: nullable,
	}
}

func catalogueTable(name string, columns ...querier_dto.Column) *querier_dto.Table {
	return &querier_dto.Table{
		Name:    name,
		Columns: columns,
	}
}

func scopedCol(name string, category querier_dto.SQLTypeCategory, nullable bool) querier_dto.ScopedColumn {
	return querier_dto.ScopedColumn{
		Name:     name,
		SQLType:  querier_dto.SQLType{EngineName: strings.ToLower(name), Category: category},
		Nullable: nullable,
	}
}

func TestScopeChain_AddTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		joinKind         querier_dto.JoinKind
		catalogueColumns []querier_dto.Column
		tableRef         querier_dto.TableReference

		wantNullable []bool

		existingBecomeNullable bool
	}{
		{
			name:     "inner join preserves nullability",
			joinKind: querier_dto.JoinInner,
			catalogueColumns: []querier_dto.Column{
				catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
				catalogueColumn("email", querier_dto.TypeCategoryText, true),
			},
			tableRef:               querier_dto.TableReference{Name: "users"},
			wantNullable:           []bool{false, true},
			existingBecomeNullable: false,
		},
		{
			name:     "left join forces all added columns nullable",
			joinKind: querier_dto.JoinLeft,
			catalogueColumns: []querier_dto.Column{
				catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
				catalogueColumn("name", querier_dto.TypeCategoryText, false),
			},
			tableRef:               querier_dto.TableReference{Name: "orders"},
			wantNullable:           []bool{true, true},
			existingBecomeNullable: false,
		},
		{
			name:     "right join makes existing tables nullable",
			joinKind: querier_dto.JoinRight,
			catalogueColumns: []querier_dto.Column{
				catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
			},
			tableRef:               querier_dto.TableReference{Name: "orders"},
			wantNullable:           []bool{false},
			existingBecomeNullable: true,
		},
		{
			name:     "full join makes all columns nullable",
			joinKind: querier_dto.JoinFull,
			catalogueColumns: []querier_dto.Column{
				catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
				catalogueColumn("amount", querier_dto.TypeCategoryDecimal, false),
			},
			tableRef:               querier_dto.TableReference{Name: "orders"},
			wantNullable:           []bool{true, true},
			existingBecomeNullable: true,
		},
		{
			name:     "positional join forces both directions nullable",
			joinKind: querier_dto.JoinPositional,
			catalogueColumns: []querier_dto.Column{
				catalogueColumn("val", querier_dto.TypeCategoryInteger, false),
			},
			tableRef:               querier_dto.TableReference{Name: "series"},
			wantNullable:           []bool{true},
			existingBecomeNullable: true,
		},
		{
			name:     "alias used when present",
			joinKind: querier_dto.JoinInner,
			catalogueColumns: []querier_dto.Column{
				catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
			},
			tableRef: querier_dto.TableReference{
				Name:  "users",
				Alias: "u",
			},
			wantNullable:           []bool{false},
			existingBecomeNullable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := newScopeChain(querier_dto.ScopeKindQuery, nil)

			existingCat := catalogueTable("existing",
				catalogueColumn("pk", querier_dto.TypeCategoryInteger, false),
			)
			err := sc.AddTable(querier_dto.TableReference{Name: "existing"}, querier_dto.JoinInner, existingCat)
			require.NoError(t, err)

			cat := catalogueTable(tt.tableRef.Name, tt.catalogueColumns...)
			err = sc.AddTable(tt.tableRef, tt.joinKind, cat)
			require.NoError(t, err)

			lookupKey := tt.tableRef.Alias
			if lookupKey == "" {
				lookupKey = tt.tableRef.Name
			}

			scopedTable, exists := sc.tables[lookupKey]
			require.True(t, exists, "table should be registered under %q", lookupKey)

			require.Len(t, scopedTable.Columns, len(tt.wantNullable))
			for i, wantNull := range tt.wantNullable {
				assert.Equal(t, wantNull, scopedTable.Columns[i].Nullable,
					"column %q nullability mismatch", scopedTable.Columns[i].Name)
			}

			existingScoped := sc.tables["existing"]
			require.NotNil(t, existingScoped)
			assert.Equal(t, tt.existingBecomeNullable, existingScoped.Columns[0].Nullable,
				"existing table column nullability mismatch")
		})
	}
}

func TestScopeChain_AddCTE(t *testing.T) {
	t.Parallel()

	t.Run("registers CTE by lowercase name with correct columns", func(t *testing.T) {
		t.Parallel()

		sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
		cols := []querier_dto.ScopedColumn{
			scopedCol("id", querier_dto.TypeCategoryInteger, false),
			scopedCol("label", querier_dto.TypeCategoryText, true),
		}

		sc.AddCTE("ActiveUsers", cols)

		cte, exists := sc.ctes["activeusers"]
		require.True(t, exists)
		assert.Equal(t, "ActiveUsers", cte.name, "original casing should be preserved in name field")
		assert.Equal(t, cols, cte.columns)
	})

	t.Run("case-insensitive lookup works", func(t *testing.T) {
		t.Parallel()

		sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
		cols := []querier_dto.ScopedColumn{
			scopedCol("x", querier_dto.TypeCategoryInteger, false),
		}

		sc.AddCTE("MyCTE", cols)

		for _, alias := range []string{"MyCTE", "mycte", "MYCTE", "myCTE"} {
			col, tbl, err := sc.ResolveColumn(alias, "x")
			require.NoError(t, err, "alias %q should resolve", alias)
			assert.Equal(t, "x", col.Name)
			assert.Equal(t, "MyCTE", tbl.Name, "CTE table name should preserve original casing")
		}
	})
}

func TestScopeChain_AddDerivedTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		joinKind     querier_dto.JoinKind
		inputCols    []querier_dto.ScopedColumn
		wantNullable []bool
	}{
		{
			name:     "left join derived table forces nullable",
			joinKind: querier_dto.JoinLeft,
			inputCols: []querier_dto.ScopedColumn{
				scopedCol("val", querier_dto.TypeCategoryInteger, false),
				scopedCol("tag", querier_dto.TypeCategoryText, false),
			},
			wantNullable: []bool{true, true},
		},
		{
			name:     "inner join derived table preserves nullability",
			joinKind: querier_dto.JoinInner,
			inputCols: []querier_dto.ScopedColumn{
				scopedCol("val", querier_dto.TypeCategoryInteger, false),
				scopedCol("tag", querier_dto.TypeCategoryText, true),
			},
			wantNullable: []bool{false, true},
		},
		{
			name:     "full join derived table forces nullable",
			joinKind: querier_dto.JoinFull,
			inputCols: []querier_dto.ScopedColumn{
				scopedCol("val", querier_dto.TypeCategoryInteger, false),
			},
			wantNullable: []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
			ref := querier_dto.DerivedTableReference{
				Alias:    "derived",
				Columns:  tt.inputCols,
				JoinKind: tt.joinKind,
			}

			sc.AddDerivedTable(ref)

			scopedTable, exists := sc.tables["derived"]
			require.True(t, exists)
			require.Len(t, scopedTable.Columns, len(tt.wantNullable))
			for i, wantNull := range tt.wantNullable {
				assert.Equal(t, wantNull, scopedTable.Columns[i].Nullable,
					"column %q nullability mismatch", scopedTable.Columns[i].Name)
			}
		})
	}
}

func TestScopeChain_ResolveColumn_Qualified(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func() *scopeChain
		tableAlias string
		columnName string
		wantCol    string
		wantTable  string
		wantErrStr string
	}{
		{
			name: "table exists and column found",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("users",
					catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
					catalogueColumn("email", querier_dto.TypeCategoryText, true),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "users"}, querier_dto.JoinInner, cat)
				return sc
			},
			tableAlias: "users",
			columnName: "email",
			wantCol:    "email",
			wantTable:  "users",
		},
		{
			name: "table exists but column not found",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("users",
					catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "users"}, querier_dto.JoinInner, cat)
				return sc
			},
			tableAlias: "users",
			columnName: "nonexistent",
			wantErrStr: "Q001",
		},
		{
			name: "CTE match",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				sc.AddCTE("recent_orders", []querier_dto.ScopedColumn{
					scopedCol("order_id", querier_dto.TypeCategoryInteger, false),
				})
				return sc
			},
			tableAlias: "recent_orders",
			columnName: "order_id",
			wantCol:    "order_id",
			wantTable:  "recent_orders",
		},
		{
			name: "LATERAL visible match",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindLateral, nil)
				lateralTable := &querier_dto.ScopedTable{
					Name:  "products",
					Alias: "p",
					Columns: []querier_dto.ScopedColumn{
						scopedCol("price", querier_dto.TypeCategoryDecimal, false),
					},
				}
				sc.MarkLateralVisible([]*querier_dto.ScopedTable{lateralTable})
				return sc
			},
			tableAlias: "p",
			columnName: "price",
			wantCol:    "price",
			wantTable:  "products",
		},
		{
			name: "parent scope traversal with subquery kind",
			setup: func() *scopeChain {
				parent := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("accounts",
					catalogueColumn("balance", querier_dto.TypeCategoryDecimal, false),
				)
				_ = parent.AddTable(querier_dto.TableReference{Name: "accounts"}, querier_dto.JoinInner, cat)
				child := parent.CreateChildScope(querier_dto.ScopeKindSubquery)
				return child
			},
			tableAlias: "accounts",
			columnName: "balance",
			wantCol:    "balance",
			wantTable:  "accounts",
		},
		{
			name: "parent scope traversal with lateral kind",
			setup: func() *scopeChain {
				parent := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("events",
					catalogueColumn("ts", querier_dto.TypeCategoryTemporal, false),
				)
				_ = parent.AddTable(querier_dto.TableReference{Name: "events"}, querier_dto.JoinInner, cat)
				child := parent.CreateChildScope(querier_dto.ScopeKindLateral)
				return child
			},
			tableAlias: "events",
			columnName: "ts",
			wantCol:    "ts",
			wantTable:  "events",
		},
		{
			name: "unknown table returns Q001",
			setup: func() *scopeChain {
				return newScopeChain(querier_dto.ScopeKindQuery, nil)
			},
			tableAlias: "ghost",
			columnName: "id",
			wantErrStr: "Q001",
		},
		{
			name: "implicit rowid on qualified table",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("items",
					catalogueColumn("name", querier_dto.TypeCategoryText, false),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "items"}, querier_dto.JoinInner, cat)
				return sc
			},
			tableAlias: "items",
			columnName: "rowid",
			wantCol:    "rowid",
			wantTable:  "items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := tt.setup()
			col, tbl, err := sc.ResolveColumn(tt.tableAlias, tt.columnName)

			if tt.wantErrStr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrStr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, col)
			require.NotNil(t, tbl)
			assert.Equal(t, tt.wantCol, col.Name)
			assert.Equal(t, tt.wantTable, tbl.Name)
		})
	}
}

func TestScopeChain_ResolveColumn_Unqualified(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func() *scopeChain
		columnName string
		wantCol    string
		wantTable  string
		wantErrStr string
	}{
		{
			name: "single match returns the column",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("users",
					catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
					catalogueColumn("email", querier_dto.TypeCategoryText, true),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "users"}, querier_dto.JoinInner, cat)
				return sc
			},
			columnName: "email",
			wantCol:    "email",
			wantTable:  "users",
		},
		{
			name: "ambiguous column produces Q002",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat1 := catalogueTable("users",
					catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
				)
				cat2 := catalogueTable("orders",
					catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "users"}, querier_dto.JoinInner, cat1)
				_ = sc.AddTable(querier_dto.TableReference{Name: "orders"}, querier_dto.JoinInner, cat2)
				return sc
			},
			columnName: "id",
			wantErrStr: "Q002",
		},
		{
			name: "CTE match when not in tables",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				sc.AddCTE("totals", []querier_dto.ScopedColumn{
					scopedCol("sum_amount", querier_dto.TypeCategoryDecimal, false),
				})
				return sc
			},
			columnName: "sum_amount",
			wantCol:    "sum_amount",
			wantTable:  "totals",
		},
		{
			name: "LATERAL match",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindLateral, nil)
				lateralTable := &querier_dto.ScopedTable{
					Name:  "outer_t",
					Alias: "outer_t",
					Columns: []querier_dto.ScopedColumn{
						scopedCol("x", querier_dto.TypeCategoryInteger, false),
					},
				}
				sc.MarkLateralVisible([]*querier_dto.ScopedTable{lateralTable})
				return sc
			},
			columnName: "x",
			wantCol:    "x",
			wantTable:  "outer_t",
		},
		{
			name: "parent scope traversal",
			setup: func() *scopeChain {
				parent := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("config",
					catalogueColumn("key", querier_dto.TypeCategoryText, false),
				)
				_ = parent.AddTable(querier_dto.TableReference{Name: "config"}, querier_dto.JoinInner, cat)
				child := parent.CreateChildScope(querier_dto.ScopeKindSubquery)
				return child
			},
			columnName: "key",
			wantCol:    "key",
			wantTable:  "config",
		},
		{
			name: "implicit rowid with single table",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("items",
					catalogueColumn("name", querier_dto.TypeCategoryText, false),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "items"}, querier_dto.JoinInner, cat)
				return sc
			},
			columnName: "rowid",
			wantCol:    "rowid",
			wantTable:  "items",
		},
		{
			name: "implicit rowid ambiguous with multiple tables",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat1 := catalogueTable("t1",
					catalogueColumn("a", querier_dto.TypeCategoryText, false),
				)
				cat2 := catalogueTable("t2",
					catalogueColumn("b", querier_dto.TypeCategoryText, false),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "t1"}, querier_dto.JoinInner, cat1)
				_ = sc.AddTable(querier_dto.TableReference{Name: "t2"}, querier_dto.JoinInner, cat2)
				return sc
			},
			columnName: "rowid",
			wantErrStr: "Q002",
		},
		{
			name: "implicit rowid not resolved for WithoutRowID table",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := &querier_dto.Table{
					Name:           "strict_t",
					Columns:        []querier_dto.Column{catalogueColumn("pk", querier_dto.TypeCategoryInteger, false)},
					IsWithoutRowID: true,
				}
				_ = sc.AddTable(querier_dto.TableReference{Name: "strict_t"}, querier_dto.JoinInner, cat)
				return sc
			},
			columnName: "rowid",
			wantErrStr: "Q001",
		},
		{
			name: "column not found anywhere returns Q001",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat := catalogueTable("users",
					catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "users"}, querier_dto.JoinInner, cat)
				return sc
			},
			columnName: "phantom",
			wantErrStr: "Q001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := tt.setup()
			col, tbl, err := sc.ResolveColumn("", tt.columnName)

			if tt.wantErrStr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrStr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, col)
			require.NotNil(t, tbl)
			assert.Equal(t, tt.wantCol, col.Name)
			assert.Equal(t, tt.wantTable, tbl.Name)
		})
	}
}

func TestScopeChain_ExpandStar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func() *scopeChain
		tableAlias string
		wantCols   []string
		wantErrStr string
	}{
		{
			name: "qualified table returns only that table's columns",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat1 := catalogueTable("users",
					catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
					catalogueColumn("name", querier_dto.TypeCategoryText, false),
				)
				cat2 := catalogueTable("orders",
					catalogueColumn("order_id", querier_dto.TypeCategoryInteger, false),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "users"}, querier_dto.JoinInner, cat1)
				_ = sc.AddTable(querier_dto.TableReference{Name: "orders"}, querier_dto.JoinInner, cat2)
				return sc
			},
			tableAlias: "users",
			wantCols:   []string{"id", "name"},
		},
		{
			name: "qualified CTE returns CTE columns",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				sc.AddCTE("active", []querier_dto.ScopedColumn{
					scopedCol("user_id", querier_dto.TypeCategoryInteger, false),
					scopedCol("status", querier_dto.TypeCategoryText, false),
				})
				return sc
			},
			tableAlias: "active",
			wantCols:   []string{"user_id", "status"},
		},
		{
			name: "unknown qualified table returns Q003",
			setup: func() *scopeChain {
				return newScopeChain(querier_dto.ScopeKindQuery, nil)
			},
			tableAlias: "ghost",
			wantErrStr: "Q003",
		},
		{
			name: "unqualified returns all columns from all tables",
			setup: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				cat1 := catalogueTable("users",
					catalogueColumn("id", querier_dto.TypeCategoryInteger, false),
				)
				cat2 := catalogueTable("orders",
					catalogueColumn("order_id", querier_dto.TypeCategoryInteger, false),
				)
				_ = sc.AddTable(querier_dto.TableReference{Name: "users"}, querier_dto.JoinInner, cat1)
				_ = sc.AddTable(querier_dto.TableReference{Name: "orders"}, querier_dto.JoinInner, cat2)
				return sc
			},
			tableAlias: "",
			wantCols:   []string{"id", "order_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := tt.setup()
			cols, err := sc.ExpandStar(tt.tableAlias)

			if tt.wantErrStr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrStr)
				return
			}

			require.NoError(t, err)

			gotNames := make([]string, len(cols))
			for i, c := range cols {
				gotNames[i] = c.Name
			}

			if tt.tableAlias != "" {

				assert.Equal(t, tt.wantCols, gotNames)
			} else {

				assert.ElementsMatch(t, tt.wantCols, gotNames)
			}
		})
	}
}

func TestScopeChain_CreateChildScope(t *testing.T) {
	t.Parallel()

	t.Run("parent is linked correctly and kind is set", func(t *testing.T) {
		t.Parallel()

		parent := newScopeChain(querier_dto.ScopeKindQuery, nil)
		child := parent.CreateChildScope(querier_dto.ScopeKindSubquery)

		assert.Equal(t, parent, child.parent, "child's parent should reference the parent scope")
		assert.Equal(t, querier_dto.ScopeKindSubquery, child.kind, "child kind should be ScopeKindSubquery")
		assert.NotNil(t, child.tables, "tables map should be initialised")
		assert.NotNil(t, child.ctes, "ctes map should be initialised")
	})

	t.Run("grandchild scope traverses two levels", func(t *testing.T) {
		t.Parallel()

		root := newScopeChain(querier_dto.ScopeKindQuery, nil)
		cat := catalogueTable("root_table",
			catalogueColumn("val", querier_dto.TypeCategoryInteger, false),
		)
		_ = root.AddTable(querier_dto.TableReference{Name: "root_table"}, querier_dto.JoinInner, cat)

		child := root.CreateChildScope(querier_dto.ScopeKindSubquery)
		grandchild := child.CreateChildScope(querier_dto.ScopeKindSubquery)

		col, tbl, err := grandchild.ResolveColumn("root_table", "val")
		require.NoError(t, err)
		assert.Equal(t, "val", col.Name)
		assert.Equal(t, "root_table", tbl.Name)
	})
}

func TestScopeChain_MarkLateralVisible(t *testing.T) {
	t.Parallel()

	t.Run("tables are appended to lateral visible list", func(t *testing.T) {
		t.Parallel()

		sc := newScopeChain(querier_dto.ScopeKindLateral, nil)
		assert.Empty(t, sc.lateralVisible)

		t1 := &querier_dto.ScopedTable{
			Name:  "a",
			Alias: "a",
			Columns: []querier_dto.ScopedColumn{
				scopedCol("x", querier_dto.TypeCategoryInteger, false),
			},
		}
		t2 := &querier_dto.ScopedTable{
			Name:  "b",
			Alias: "b",
			Columns: []querier_dto.ScopedColumn{
				scopedCol("y", querier_dto.TypeCategoryText, true),
			},
		}

		sc.MarkLateralVisible([]*querier_dto.ScopedTable{t1})
		require.Len(t, sc.lateralVisible, 1)

		sc.MarkLateralVisible([]*querier_dto.ScopedTable{t2})
		require.Len(t, sc.lateralVisible, 2)

		assert.Equal(t, "a", sc.lateralVisible[0].Name)
		assert.Equal(t, "b", sc.lateralVisible[1].Name)
	})
}

func TestIsImplicitRowID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "rowid lowercase", input: "rowid", want: true},
		{name: "ROWID uppercase", input: "ROWID", want: true},
		{name: "_rowid_ lowercase", input: "_rowid_", want: true},
		{name: "_ROWID_ uppercase", input: "_ROWID_", want: true},
		{name: "OID uppercase", input: "OID", want: true},
		{name: "oid lowercase", input: "oid", want: true},
		{name: "id is not implicit rowid", input: "id", want: false},
		{name: "row_id with underscore is not implicit rowid", input: "row_id", want: false},
		{name: "empty string is not implicit rowid", input: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, isImplicitRowID(tt.input))
		})
	}
}
