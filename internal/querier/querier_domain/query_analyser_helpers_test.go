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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestComputeDynamicFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		parameters         []*querier_dto.ParameterDirective
		wantDynamic        bool
		wantDynamicColumns []string
	}{
		{
			name:               "no parameters yields not dynamic",
			parameters:         nil,
			wantDynamic:        false,
			wantDynamicColumns: nil,
		},
		{
			name: "regular param does not set dynamic",
			parameters: []*querier_dto.ParameterDirective{
				{Name: "user_id", Kind: querier_dto.ParameterDirectiveParam},
			},
			wantDynamic:        false,
			wantDynamicColumns: nil,
		},
		{
			name: "slice param does not set dynamic",
			parameters: []*querier_dto.ParameterDirective{
				{Name: "ids", Kind: querier_dto.ParameterDirectiveSlice},
			},
			wantDynamic:        false,
			wantDynamicColumns: nil,
		},
		{
			name: "optional param sets dynamic",
			parameters: []*querier_dto.ParameterDirective{
				{Name: "email", Kind: querier_dto.ParameterDirectiveOptional},
			},
			wantDynamic:        true,
			wantDynamicColumns: nil,
		},
		{
			name: "sortable param sets dynamic and collects columns",
			parameters: []*querier_dto.ParameterDirective{
				{
					Name:    "sort",
					Kind:    querier_dto.ParameterDirectiveSortable,
					Columns: []string{"name", "created_at"},
				},
			},
			wantDynamic:        true,
			wantDynamicColumns: []string{"name", "created_at"},
		},
		{
			name: "limit param sets dynamic",
			parameters: []*querier_dto.ParameterDirective{
				{Name: "page_size", Kind: querier_dto.ParameterDirectiveLimit},
			},
			wantDynamic:        true,
			wantDynamicColumns: nil,
		},
		{
			name: "offset param sets dynamic",
			parameters: []*querier_dto.ParameterDirective{
				{Name: "page_offset", Kind: querier_dto.ParameterDirectiveOffset},
			},
			wantDynamic:        true,
			wantDynamicColumns: nil,
		},
		{
			name: "combined dynamic params accumulate columns",
			parameters: []*querier_dto.ParameterDirective{
				{Name: "user_id", Kind: querier_dto.ParameterDirectiveParam},
				{Name: "email", Kind: querier_dto.ParameterDirectiveOptional},
				{
					Name:    "sort",
					Kind:    querier_dto.ParameterDirectiveSortable,
					Columns: []string{"name"},
				},
				{Name: "page_size", Kind: querier_dto.ParameterDirectiveLimit},
				{Name: "page_offset", Kind: querier_dto.ParameterDirectiveOffset},
				{
					Name:    "order",
					Kind:    querier_dto.ParameterDirectiveSortable,
					Columns: []string{"email", "id"},
				},
			},
			wantDynamic:        true,
			wantDynamicColumns: []string{"name", "email", "id"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotDynamic, gotColumns := computeDynamicFlags(testCase.parameters)

			assert.Equal(t, testCase.wantDynamic, gotDynamic)
			assert.Equal(t, testCase.wantDynamicColumns, gotColumns)
		})
	}
}

func TestAppendUniqueColumns(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}
	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}

	tests := []struct {
		name     string
		base     []querier_dto.AllowedColumn
		table    *querier_dto.Table
		seen     map[string]struct{}
		expected []querier_dto.AllowedColumn
	}{
		{
			name:     "nil table returns base unchanged",
			base:     []querier_dto.AllowedColumn{{Name: "id", SQLType: intType}},
			table:    nil,
			seen:     map[string]struct{}{"id": {}},
			expected: []querier_dto.AllowedColumn{{Name: "id", SQLType: intType}},
		},
		{
			name: "no overlap appends all columns",
			base: nil,
			table: &querier_dto.Table{
				Name: "users",
				Columns: []querier_dto.Column{
					{Name: "id", SQLType: intType},
					{Name: "email", SQLType: textType},
				},
			},
			seen: make(map[string]struct{}),
			expected: []querier_dto.AllowedColumn{
				{Name: "id", SQLType: intType},
				{Name: "email", SQLType: textType},
			},
		},
		{
			name: "duplicates are skipped",
			base: []querier_dto.AllowedColumn{{Name: "id", SQLType: intType}},
			table: &querier_dto.Table{
				Name: "orders",
				Columns: []querier_dto.Column{
					{Name: "id", SQLType: intType},
					{Name: "total", SQLType: intType},
				},
			},
			seen: map[string]struct{}{"id": {}},
			expected: []querier_dto.AllowedColumn{
				{Name: "id", SQLType: intType},
				{Name: "total", SQLType: intType},
			},
		},
		{
			name:     "empty table columns returns base unchanged",
			base:     []querier_dto.AllowedColumn{{Name: "id", SQLType: intType}},
			table:    &querier_dto.Table{Name: "empty"},
			seen:     map[string]struct{}{"id": {}},
			expected: []querier_dto.AllowedColumn{{Name: "id", SQLType: intType}},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := appendUniqueColumns(testCase.base, testCase.seen, testCase.table)

			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestFindTable(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("public")

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}
	usersTable := newTestTable("users", querier_dto.Column{Name: "id", SQLType: intType})
	catalogue.Schemas["public"].Tables["users"] = usersTable

	catalogue.Schemas["analytics"] = &querier_dto.Schema{
		Name:           "analytics",
		Tables:         map[string]*querier_dto.Table{},
		Views:          map[string]*querier_dto.View{},
		Enums:          map[string]*querier_dto.Enum{},
		Functions:      map[string][]*querier_dto.FunctionSignature{},
		CompositeTypes: map[string]*querier_dto.CompositeType{},
		Sequences:      map[string]*querier_dto.Sequence{},
	}
	eventsTable := newTestTable("events", querier_dto.Column{Name: "event_id", SQLType: intType})
	catalogue.Schemas["analytics"].Tables["events"] = eventsTable

	analyser := &queryAnalyser{
		engine:    &mockEngine{},
		catalogue: catalogue,
	}

	tests := []struct {
		name      string
		schema    string
		tableName string
		wantTable *querier_dto.Table
		wantNil   bool
	}{
		{
			name:      "table exists in default schema with empty schema arg",
			schema:    "",
			tableName: "users",
			wantTable: usersTable,
		},
		{
			name:      "table exists in explicit schema",
			schema:    "analytics",
			tableName: "events",
			wantTable: eventsTable,
		},
		{
			name:      "table not found in schema",
			schema:    "public",
			tableName: "nonexistent",
			wantNil:   true,
		},
		{
			name:      "schema not found",
			schema:    "unknown_schema",
			tableName: "users",
			wantNil:   true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := analyser.findTable(testCase.schema, testCase.tableName)

			if testCase.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, testCase.wantTable, got)
			}
		})
	}
}

func TestBlockError(t *testing.T) {
	t.Parallel()

	got := blockError("queries/users.sql", 42, "Q010", querier_dto.SeverityError, "failed to parse query")

	assert.Equal(t, querier_dto.SourceError{
		Filename: "queries/users.sql",
		Line:     42,
		Column:   1,
		Message:  "failed to parse query",
		Severity: querier_dto.SeverityError,
		Code:     "Q010",
	}, got)
}

func TestBlockError_warning_severity(t *testing.T) {
	t.Parallel()

	got := blockError("file.sql", 10, "Q003", querier_dto.SeverityWarning, "unknown table")

	assert.Equal(t, "file.sql", got.Filename)
	assert.Equal(t, 10, got.Line)
	assert.Equal(t, 1, got.Column)
	assert.Equal(t, querier_dto.SeverityWarning, got.Severity)
	assert.Equal(t, "Q003", got.Code)
	assert.Equal(t, "unknown table", got.Message)
}

func TestAddFileLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		diagnostics []querier_dto.SourceError
		filename    string
		startLine   int
		expected    []querier_dto.SourceError
	}{
		{
			name:        "empty diagnostics returns empty slice",
			diagnostics: nil,
			filename:    "file.sql",
			startLine:   10,
			expected:    []querier_dto.SourceError{},
		},
		{
			name: "sets filename when empty",
			diagnostics: []querier_dto.SourceError{
				{Message: "err1"},
			},
			filename:  "queries/users.sql",
			startLine: 5,
			expected: []querier_dto.SourceError{
				{
					Filename: "queries/users.sql",
					Line:     5,
					Column:   1,
					Message:  "err1",
				},
			},
		},
		{
			name: "sets line when zero",
			diagnostics: []querier_dto.SourceError{
				{Filename: "existing.sql", Message: "err2"},
			},
			filename:  "other.sql",
			startLine: 20,
			expected: []querier_dto.SourceError{
				{
					Filename: "existing.sql",
					Line:     20,
					Column:   1,
					Message:  "err2",
				},
			},
		},
		{
			name: "preserves existing filename and line",
			diagnostics: []querier_dto.SourceError{
				{Filename: "original.sql", Line: 99, Column: 5, Message: "err3"},
			},
			filename:  "replacement.sql",
			startLine: 1,
			expected: []querier_dto.SourceError{
				{
					Filename: "original.sql",
					Line:     99,
					Column:   5,
					Message:  "err3",
				},
			},
		},
		{
			name: "sets column to 1 when zero",
			diagnostics: []querier_dto.SourceError{
				{Filename: "f.sql", Line: 3, Column: 0, Message: "err4"},
			},
			filename:  "f.sql",
			startLine: 1,
			expected: []querier_dto.SourceError{
				{
					Filename: "f.sql",
					Line:     3,
					Column:   1,
					Message:  "err4",
				},
			},
		},
		{
			name: "multiple diagnostics handled independently",
			diagnostics: []querier_dto.SourceError{
				{Message: "first"},
				{Filename: "kept.sql", Line: 7, Column: 3, Message: "second"},
			},
			filename:  "default.sql",
			startLine: 10,
			expected: []querier_dto.SourceError{
				{
					Filename: "default.sql",
					Line:     10,
					Column:   1,
					Message:  "first",
				},
				{
					Filename: "kept.sql",
					Line:     7,
					Column:   3,
					Message:  "second",
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := addFileLocation(testCase.diagnostics, testCase.filename, testCase.startLine)

			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestExtractEmbedTableNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{
			name:     "no embed directives",
			sql:      "SELECT u.id, u.name FROM users u",
			expected: nil,
		},
		{
			name:     "single embed",
			sql:      "SELECT u.id, /* piko.embed(addresses) */ a.* FROM users u JOIN addresses a ON a.user_id = u.id",
			expected: []string{"addresses"},
		},
		{
			name:     "multiple embeds",
			sql:      "SELECT u.id, /* piko.embed(addresses) */ a.*, /* piko.embed(orders) */ o.* FROM users u",
			expected: []string{"addresses", "orders"},
		},
		{
			name:     "embed with whitespace around table name",
			sql:      "SELECT /* piko.embed(  roles  ) */ r.* FROM roles r",
			expected: []string{"roles"},
		},
		{
			name:     "empty embed is ignored",
			sql:      "SELECT /* piko.embed() */ * FROM users",
			expected: nil,
		},
		{
			name:     "unclosed embed marker stops scanning",
			sql:      "SELECT /* piko.embed(broken FROM users",
			expected: nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := extractEmbedTableNames(testCase.sql)

			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestIsOuterJoinTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tableName string
		tables    map[string]*querier_dto.ScopedTable
		expected  bool
	}{
		{
			name:      "left join returns true",
			tableName: "addresses",
			tables: map[string]*querier_dto.ScopedTable{
				"addresses": {Name: "addresses", Alias: "addresses", JoinKind: querier_dto.JoinLeft},
			},
			expected: true,
		},
		{
			name:      "right join returns true",
			tableName: "orders",
			tables: map[string]*querier_dto.ScopedTable{
				"orders": {Name: "orders", Alias: "orders", JoinKind: querier_dto.JoinRight},
			},
			expected: true,
		},
		{
			name:      "full join returns true",
			tableName: "payments",
			tables: map[string]*querier_dto.ScopedTable{
				"payments": {Name: "payments", Alias: "payments", JoinKind: querier_dto.JoinFull},
			},
			expected: true,
		},
		{
			name:      "positional join returns true",
			tableName: "series",
			tables: map[string]*querier_dto.ScopedTable{
				"series": {Name: "series", Alias: "series", JoinKind: querier_dto.JoinPositional},
			},
			expected: true,
		},
		{
			name:      "inner join returns false",
			tableName: "users",
			tables: map[string]*querier_dto.ScopedTable{
				"users": {Name: "users", Alias: "users", JoinKind: querier_dto.JoinInner},
			},
			expected: false,
		},
		{
			name:      "cross join returns false",
			tableName: "settings",
			tables: map[string]*querier_dto.ScopedTable{
				"settings": {Name: "settings", Alias: "settings", JoinKind: querier_dto.JoinCross},
			},
			expected: false,
		},
		{
			name:      "table not found returns false",
			tableName: "nonexistent",
			tables: map[string]*querier_dto.ScopedTable{
				"users": {Name: "users", Alias: "users", JoinKind: querier_dto.JoinLeft},
			},
			expected: false,
		},
		{
			name:      "matches by alias case-insensitively",
			tableName: "ADDR",
			tables: map[string]*querier_dto.ScopedTable{
				"addr": {Name: "addresses", Alias: "addr", JoinKind: querier_dto.JoinLeft},
			},
			expected: true,
		},
		{
			name:      "matches by name case-insensitively",
			tableName: "ADDRESSES",
			tables: map[string]*querier_dto.ScopedTable{
				"a": {Name: "addresses", Alias: "a", JoinKind: querier_dto.JoinLeft},
			},
			expected: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scope := &scopeChain{
				tables: testCase.tables,
				ctes:   make(map[string]*resolvedCTE),
			}

			got := isOuterJoinTable(testCase.tableName, scope)

			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestOutputColumnsToScoped(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}
	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}

	tests := []struct {
		name     string
		columns  []querier_dto.OutputColumn
		expected []querier_dto.ScopedColumn
	}{
		{
			name:     "empty input returns empty slice",
			columns:  nil,
			expected: []querier_dto.ScopedColumn{},
		},
		{
			name: "converts output columns to scoped columns",
			columns: []querier_dto.OutputColumn{
				{Name: "id", SQLType: intType, Nullable: false},
				{Name: "email", SQLType: textType, Nullable: true},
			},
			expected: []querier_dto.ScopedColumn{
				{Name: "id", SQLType: intType, Nullable: false},
				{Name: "email", SQLType: textType, Nullable: true},
			},
		},
		{
			name: "extra output column fields are not carried over",
			columns: []querier_dto.OutputColumn{
				{
					Name:         "name",
					SQLType:      textType,
					Nullable:     false,
					SourceTable:  "users",
					SourceColumn: "name",
					IsEmbedded:   true,
					EmbedTable:   "users",
				},
			},
			expected: []querier_dto.ScopedColumn{
				{Name: "name", SQLType: textType, Nullable: false},
			},
		},
	}

	analyser := &queryAnalyser{}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := analyser.outputColumnsToScoped(testCase.columns)

			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestResolveEmbedDirectives(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}
	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}

	tests := []struct {
		name     string
		sql      string
		columns  []querier_dto.OutputColumn
		scope    *scopeChain
		expected []querier_dto.OutputColumn
	}{
		{
			name: "no embed directives returns columns unchanged",
			sql:  "SELECT id, name FROM users",
			columns: []querier_dto.OutputColumn{
				{Name: "id", SQLType: intType, SourceTable: "users"},
				{Name: "name", SQLType: textType, SourceTable: "users"},
			},
			scope: &scopeChain{
				tables: make(map[string]*querier_dto.ScopedTable),
				ctes:   make(map[string]*resolvedCTE),
			},
			expected: []querier_dto.OutputColumn{
				{Name: "id", SQLType: intType, SourceTable: "users"},
				{Name: "name", SQLType: textType, SourceTable: "users"},
			},
		},
		{
			name: "embed marks matching source table columns",
			sql:  "SELECT u.id, /* piko.embed(addresses) */ a.street, a.city FROM users u JOIN addresses a ON a.user_id = u.id",
			columns: []querier_dto.OutputColumn{
				{Name: "id", SQLType: intType, SourceTable: "users"},
				{Name: "street", SQLType: textType, SourceTable: "addresses"},
				{Name: "city", SQLType: textType, SourceTable: "addresses"},
			},
			scope: &scopeChain{
				tables: map[string]*querier_dto.ScopedTable{
					"a": {Name: "addresses", Alias: "a", JoinKind: querier_dto.JoinInner},
				},
				ctes: make(map[string]*resolvedCTE),
			},
			expected: []querier_dto.OutputColumn{
				{Name: "id", SQLType: intType, SourceTable: "users"},
				{Name: "street", SQLType: textType, SourceTable: "addresses", IsEmbedded: true, EmbedTable: "addresses", EmbedIsOuter: false},
				{Name: "city", SQLType: textType, SourceTable: "addresses", IsEmbedded: true, EmbedTable: "addresses", EmbedIsOuter: false},
			},
		},
		{
			name: "embed with outer join sets EmbedIsOuter",
			sql:  "SELECT u.id, /* piko.embed(addresses) */ a.street FROM users u LEFT JOIN addresses a ON a.user_id = u.id",
			columns: []querier_dto.OutputColumn{
				{Name: "id", SQLType: intType, SourceTable: "users"},
				{Name: "street", SQLType: textType, SourceTable: "addresses"},
			},
			scope: &scopeChain{
				tables: map[string]*querier_dto.ScopedTable{
					"addresses": {Name: "addresses", Alias: "addresses", JoinKind: querier_dto.JoinLeft},
				},
				ctes: make(map[string]*resolvedCTE),
			},
			expected: []querier_dto.OutputColumn{
				{Name: "id", SQLType: intType, SourceTable: "users"},
				{Name: "street", SQLType: textType, SourceTable: "addresses", IsEmbedded: true, EmbedTable: "addresses", EmbedIsOuter: true},
			},
		},
		{
			name: "case-insensitive matching on source table",
			sql:  "SELECT /* piko.embed(Users) */ u.name FROM users u",
			columns: []querier_dto.OutputColumn{
				{Name: "name", SQLType: textType, SourceTable: "users"},
			},
			scope: &scopeChain{
				tables: map[string]*querier_dto.ScopedTable{
					"u": {Name: "users", Alias: "u", JoinKind: querier_dto.JoinInner},
				},
				ctes: make(map[string]*resolvedCTE),
			},
			expected: []querier_dto.OutputColumn{
				{Name: "name", SQLType: textType, SourceTable: "users", IsEmbedded: true, EmbedTable: "Users", EmbedIsOuter: false},
			},
		},
		{
			name: "columns with empty source table are skipped",
			sql:  "SELECT /* piko.embed(users) */ count(*) as cnt FROM users",
			columns: []querier_dto.OutputColumn{
				{Name: "cnt", SQLType: intType, SourceTable: ""},
			},
			scope: &scopeChain{
				tables: make(map[string]*querier_dto.ScopedTable),
				ctes:   make(map[string]*resolvedCTE),
			},
			expected: []querier_dto.OutputColumn{
				{Name: "cnt", SQLType: intType, SourceTable: ""},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := resolveEmbedDirectives(testCase.sql, testCase.columns, testCase.scope)

			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestResolveTableReference(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}
	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["users"] = &querier_dto.Table{
		Name:   "users",
		Schema: "public",
		Columns: []querier_dto.Column{
			{Name: "id", SQLType: intType},
			{Name: "email", SQLType: textType},
		},
	}
	catalogue.Schemas["public"].Views["active_users"] = &querier_dto.View{
		Name:   "active_users",
		Schema: "public",
		Columns: []querier_dto.Column{
			{Name: "id", SQLType: intType},
		},
	}

	analyser := &queryAnalyser{
		engine:    &mockEngine{},
		catalogue: catalogue,
	}

	tests := []struct {
		name      string
		reference querier_dto.TableReference
		wantName  string
		wantErr   string
	}{
		{
			name:      "known table resolves successfully",
			reference: querier_dto.TableReference{Name: "users"},
			wantName:  "users",
		},
		{
			name:      "known table in explicit schema resolves",
			reference: querier_dto.TableReference{Schema: "public", Name: "users"},
			wantName:  "users",
		},
		{
			name:      "view resolves as table",
			reference: querier_dto.TableReference{Name: "active_users"},
			wantName:  "active_users",
		},
		{
			name:      "unknown table returns error",
			reference: querier_dto.TableReference{Name: "nonexistent"},
			wantErr:   `unknown table or view "nonexistent"`,
		},
		{
			name:      "unknown schema returns error",
			reference: querier_dto.TableReference{Schema: "private", Name: "users"},
			wantErr:   `unknown schema "private"`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := analyser.resolveTableReference(testCase.reference)

			if testCase.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, testCase.wantName, got.Name)
			}
		})
	}
}

func TestBuildScopeChain(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}
	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}

	t.Run("single FROM table populates scope", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = &querier_dto.Table{
			Name: "users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: intType},
				{Name: "name", SQLType: textType},
			},
		}

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		raw := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "users"},
			},
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		diagnostics := analyser.buildScopeChain(raw, scope)

		assert.Empty(t, diagnostics)
		assert.Contains(t, scope.tables, "users")
		assert.Len(t, scope.tables["users"].Columns, 2)
	})

	t.Run("join adds both tables to scope", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = &querier_dto.Table{
			Name: "users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: intType},
			},
		}
		catalogue.Schemas["public"].Tables["orders"] = &querier_dto.Table{
			Name: "orders",
			Columns: []querier_dto.Column{
				{Name: "order_id", SQLType: intType},
			},
		}

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		raw := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "users"},
			},
			JoinClauses: []querier_dto.JoinClause{
				{
					Table: querier_dto.TableReference{Name: "orders"},
					Kind:  querier_dto.JoinLeft,
				},
			},
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		diagnostics := analyser.buildScopeChain(raw, scope)

		assert.Empty(t, diagnostics)
		assert.Contains(t, scope.tables, "users")
		assert.Contains(t, scope.tables, "orders")
	})

	t.Run("unknown FROM table produces warning diagnostic", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		raw := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "nonexistent"},
			},
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		diagnostics := analyser.buildScopeChain(raw, scope)

		require.Len(t, diagnostics, 1)
		assert.Equal(t, querier_dto.SeverityWarning, diagnostics[0].Severity)
		assert.Equal(t, "Q003", diagnostics[0].Code)
		assert.Contains(t, diagnostics[0].Message, "nonexistent")
	})

	t.Run("FROM table matching CTE reuses CTE instead of catalogue lookup", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		scope.AddCTE("recent_users", []querier_dto.ScopedColumn{
			{Name: "id", SQLType: intType, Nullable: false},
		})

		raw := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "recent_users"},
			},
		}

		diagnostics := analyser.buildScopeChain(raw, scope)

		assert.Empty(t, diagnostics)

		assert.Empty(t, scope.tables)
		assert.Contains(t, scope.ctes, "recent_users")
	})

	t.Run("FROM table matching CTE with different alias registers alias", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		scope.AddCTE("recent_users", []querier_dto.ScopedColumn{
			{Name: "id", SQLType: intType, Nullable: false},
		})

		raw := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "recent_users", Alias: "ru"},
			},
		}

		diagnostics := analyser.buildScopeChain(raw, scope)

		assert.Empty(t, diagnostics)

		assert.Contains(t, scope.ctes, "ru")
	})

	t.Run("derived tables are added to scope", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		raw := &querier_dto.RawQueryAnalysis{
			DerivedTables: []querier_dto.DerivedTableReference{
				{
					Alias: "sub",
					Columns: []querier_dto.ScopedColumn{
						{Name: "x", SQLType: intType},
					},
					JoinKind: querier_dto.JoinInner,
				},
			},
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		diagnostics := analyser.buildScopeChain(raw, scope)

		assert.Empty(t, diagnostics)
		assert.Contains(t, scope.tables, "sub")
	})
}

func TestResolveJoinClauses(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}

	t.Run("known join table is added to scope", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["orders"] = &querier_dto.Table{
			Name:    "orders",
			Columns: []querier_dto.Column{{Name: "order_id", SQLType: intType}},
		}

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

		diagnostics := analyser.resolveJoinClauses(
			[]querier_dto.JoinClause{
				{
					Table: querier_dto.TableReference{Name: "orders"},
					Kind:  querier_dto.JoinInner,
				},
			},
			scope,
		)

		assert.Empty(t, diagnostics)
		assert.Contains(t, scope.tables, "orders")
	})

	t.Run("unknown join table produces warning", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

		diagnostics := analyser.resolveJoinClauses(
			[]querier_dto.JoinClause{
				{
					Table: querier_dto.TableReference{Name: "missing"},
					Kind:  querier_dto.JoinLeft,
				},
			},
			scope,
		)

		require.Len(t, diagnostics, 1)
		assert.Equal(t, "Q003", diagnostics[0].Code)
		assert.Contains(t, diagnostics[0].Message, "missing")
	})

	t.Run("join referencing CTE uses CTE columns", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		scope.AddCTE("my_cte", []querier_dto.ScopedColumn{
			{Name: "val", SQLType: intType},
		})

		diagnostics := analyser.resolveJoinClauses(
			[]querier_dto.JoinClause{
				{
					Table: querier_dto.TableReference{Name: "my_cte", Alias: "mc"},
					Kind:  querier_dto.JoinInner,
				},
			},
			scope,
		)

		assert.Empty(t, diagnostics)

		assert.Contains(t, scope.ctes, "mc")
	})
}

func TestAssembleQuery(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}
	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}

	t.Run("basic assembly populates all fields", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		directives := &querier_dto.QueryDirectives{
			GroupByKeys: []string{"user_id"},
		}

		rawAnalysis := &querier_dto.RawQueryAnalysis{
			InsertTable:   "users",
			InsertColumns: []string{"id", "name"},
			ReadOnly:      true,
		}

		outputColumns := []querier_dto.OutputColumn{
			{Name: "id", SQLType: intType},
			{Name: "name", SQLType: textType},
		}

		parameters := []querier_dto.QueryParameter{
			{Name: "user_id", Number: 1, SQLType: intType},
		}

		got := analyser.assembleQuery(assembleQueryInput{
			queryName:     "GetUser",
			queryCommand:  querier_dto.QueryCommandOne,
			block:         queryBlock{sql: "SELECT id, name FROM users WHERE id = $1", startLine: 5},
			filename:      "queries/users.sql",
			outputColumns: outputColumns,
			parameters:    parameters,
			isDynamic:     false,
			directives:    directives,
			rawAnalysis:   rawAnalysis,
		})

		require.NotNil(t, got)
		assert.Equal(t, "GetUser", got.Name)
		assert.Equal(t, querier_dto.QueryCommandOne, got.Command)
		assert.Equal(t, "SELECT id, name FROM users WHERE id = $1", got.SQL)
		assert.Equal(t, "queries/users.sql", got.Filename)
		assert.Equal(t, 5, got.Line)
		assert.Equal(t, outputColumns, got.OutputColumns)
		assert.Equal(t, parameters, got.Parameters)
		assert.False(t, got.IsDynamic)
		assert.Equal(t, []string{"user_id"}, got.GroupByKey)
		assert.Equal(t, "users", got.InsertTable)
		assert.Equal(t, []string{"id", "name"}, got.InsertColumns)
		assert.True(t, got.ReadOnly)
	})

	t.Run("read-only override takes precedence over raw analysis", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		directives := &querier_dto.QueryDirectives{
			ReadOnlyOverride: new(false),
		}

		rawAnalysis := &querier_dto.RawQueryAnalysis{
			ReadOnly: true,
		}

		got := analyser.assembleQuery(assembleQueryInput{
			queryName:   "ForceWritable",
			directives:  directives,
			rawAnalysis: rawAnalysis,
			block:       queryBlock{sql: "SELECT 1"},
		})

		assert.False(t, got.ReadOnly)
	})

	t.Run("dynamic runtime populates allowed columns", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = &querier_dto.Table{
			Name: "users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: intType},
				{Name: "email", SQLType: textType},
			},
		}

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		directives := &querier_dto.QueryDirectives{
			DynamicRuntime: true,
		}

		rawAnalysis := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "users"},
			},
		}

		got := analyser.assembleQuery(assembleQueryInput{
			queryName:   "DynamicUsers",
			directives:  directives,
			rawAnalysis: rawAnalysis,
			block:       queryBlock{sql: "SELECT * FROM users"},
		})

		require.Len(t, got.AllowedColumns, 2)
		assert.Equal(t, "id", got.AllowedColumns[0].Name)
		assert.Equal(t, "email", got.AllowedColumns[1].Name)
		assert.True(t, got.DynamicRuntime)
	})
}

func TestPopulateCTEScope(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}

	t.Run("FROM table referencing parent CTE adds to CTE scope", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		parentScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		parentScope.AddCTE("parent_cte", []querier_dto.ScopedColumn{
			{Name: "val", SQLType: intType},
		})

		cteScope := parentScope.CreateChildScope(querier_dto.ScopeKindCTE)

		diagnostics := analyser.populateCTEScope(
			[]querier_dto.TableReference{
				{Name: "parent_cte"},
			},
			parentScope,
			cteScope,
		)

		assert.Empty(t, diagnostics)
		assert.Contains(t, cteScope.ctes, "parent_cte")
	})

	t.Run("FROM table referencing catalogue table adds to CTE scope", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = &querier_dto.Table{
			Name:    "users",
			Columns: []querier_dto.Column{{Name: "id", SQLType: intType}},
		}

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		parentScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		cteScope := parentScope.CreateChildScope(querier_dto.ScopeKindCTE)

		diagnostics := analyser.populateCTEScope(
			[]querier_dto.TableReference{
				{Name: "users"},
			},
			parentScope,
			cteScope,
		)

		assert.Empty(t, diagnostics)
		assert.Contains(t, cteScope.tables, "users")
	})

	t.Run("unknown FROM table in CTE produces warning", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		parentScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		cteScope := parentScope.CreateChildScope(querier_dto.ScopeKindCTE)

		diagnostics := analyser.populateCTEScope(
			[]querier_dto.TableReference{
				{Name: "no_such_table"},
			},
			parentScope,
			cteScope,
		)

		require.Len(t, diagnostics, 1)
		assert.Equal(t, "Q003", diagnostics[0].Code)
		assert.Contains(t, diagnostics[0].Message, "no_such_table")
	})

	t.Run("CTE alias is used when different from name", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		parentScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		parentScope.AddCTE("my_cte", []querier_dto.ScopedColumn{
			{Name: "col", SQLType: intType},
		})

		cteScope := parentScope.CreateChildScope(querier_dto.ScopeKindCTE)

		diagnostics := analyser.populateCTEScope(
			[]querier_dto.TableReference{
				{Name: "my_cte", Alias: "mc"},
			},
			parentScope,
			cteScope,
		)

		assert.Empty(t, diagnostics)
		assert.Contains(t, cteScope.ctes, "mc")
	})
}

func TestExtractAllowedColumns(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger}
	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}

	t.Run("no FROM tables returns empty", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		raw := &querier_dto.RawQueryAnalysis{}

		got := analyser.extractAllowedColumns(raw)

		assert.Empty(t, got)
	})

	t.Run("collects columns from FROM tables", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = &querier_dto.Table{
			Name: "users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: intType},
				{Name: "name", SQLType: textType},
			},
		}

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		raw := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "users"},
			},
		}

		got := analyser.extractAllowedColumns(raw)

		require.Len(t, got, 2)
		assert.Equal(t, "id", got[0].Name)
		assert.Equal(t, "name", got[1].Name)
	})

	t.Run("collects columns from JOIN tables without duplicates", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = &querier_dto.Table{
			Name: "users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: intType},
				{Name: "name", SQLType: textType},
			},
		}
		catalogue.Schemas["public"].Tables["orders"] = &querier_dto.Table{
			Name: "orders",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: intType},
				{Name: "total", SQLType: intType},
			},
		}

		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		raw := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "users"},
			},
			JoinClauses: []querier_dto.JoinClause{
				{
					Table: querier_dto.TableReference{Name: "orders"},
					Kind:  querier_dto.JoinInner,
				},
			},
		}

		got := analyser.extractAllowedColumns(raw)

		require.Len(t, got, 3)
		names := make([]string, len(got))
		for i, col := range got {
			names[i] = col.Name
		}
		assert.Contains(t, names, "id")
		assert.Contains(t, names, "name")
		assert.Contains(t, names, "total")
	})

	t.Run("unknown table is safely skipped", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		analyser := &queryAnalyser{
			engine:    &mockEngine{},
			catalogue: catalogue,
		}

		raw := &querier_dto.RawQueryAnalysis{
			FromTables: []querier_dto.TableReference{
				{Name: "nonexistent"},
			},
		}

		got := analyser.extractAllowedColumns(raw)

		assert.Empty(t, got)
	})
}
