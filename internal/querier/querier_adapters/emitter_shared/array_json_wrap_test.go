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

package emitter_shared

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/querier/querier_dto"
)

func arrayColumn(name string) querier_dto.OutputColumn {
	return querier_dto.OutputColumn{Name: name, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryArray}}
}

func scalarColumn(name string) querier_dto.OutputColumn {
	return querier_dto.OutputColumn{Name: name, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}}
}

func srcColumn(name string, qualifier string, category querier_dto.SQLTypeCategory) querier_dto.OutputColumn {
	return querier_dto.OutputColumn{
		Name:            name,
		SourceColumn:    name,
		SourceQualifier: qualifier,
		SQLType:         querier_dto.SQLType{Category: category},
	}
}

func dqQuote(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func wrapTestMappings() *querier_dto.TypeMappingTable {
	return &querier_dto.TypeMappingTable{Mappings: []querier_dto.TypeMapping{
		{SQLCategory: querier_dto.TypeCategoryArray, NotNull: querier_dto.GoType{Name: "[]any"}, Nullable: querier_dto.GoType{Name: "[]any"}},
		{SQLCategory: querier_dto.TypeCategoryText, NotNull: querier_dto.GoType{Name: "string"}, Nullable: querier_dto.GoType{Name: "*string"}},
	}}
}

func TestWrapArrayColumnsAsJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		sql         string
		columns     []querier_dto.OutputColumn
		wantSQL     string
		wantWrapped []int
	}{
		{
			name:        "bare array column",
			sql:         "SELECT id, tags FROM t WHERE id = $1",
			columns:     []querier_dto.OutputColumn{scalarColumn("id"), arrayColumn("tags")},
			wantSQL:     `SELECT id, to_json(tags) AS "tags" FROM t WHERE id = $1`,
			wantWrapped: []int{1},
		},
		{
			name:        "qualified array column aliases to output name",
			sql:         "SELECT t.tags FROM t",
			columns:     []querier_dto.OutputColumn{arrayColumn("tags")},
			wantSQL:     `SELECT to_json(t.tags) AS "tags" FROM t`,
			wantWrapped: []int{0},
		},
		{
			name:        "explicit AS alias is preserved",
			sql:         "SELECT tags AS labels FROM t",
			columns:     []querier_dto.OutputColumn{arrayColumn("labels")},
			wantSQL:     "SELECT to_json(tags) AS labels FROM t",
			wantWrapped: []int{0},
		},
		{
			name:        "double-quoted identifier with a space is wrapped (lexer, not regex)",
			sql:         `SELECT "my col" FROM t`,
			columns:     []querier_dto.OutputColumn{arrayColumn("my col")},
			wantSQL:     `SELECT to_json("my col") AS "my col" FROM t`,
			wantWrapped: []int{0},
		},
		{
			name:        "double-quoted identifier with an escaped quote is wrapped",
			sql:         `SELECT "we""ird" FROM t`,
			columns:     []querier_dto.OutputColumn{arrayColumn(`we"ird`)},
			wantSQL:     `SELECT to_json("we""ird") AS "we""ird" FROM t`,
			wantWrapped: []int{0},
		},
		{
			name:        "array_agg expression with AS",
			sql:         "SELECT array_agg(x) AS xs FROM t",
			columns:     []querier_dto.OutputColumn{arrayColumn("xs")},
			wantSQL:     "SELECT to_json(array_agg(x)) AS xs FROM t",
			wantWrapped: []int{0},
		},
		{
			name:        "comma inside a function call is not a split",
			sql:         "SELECT id, coalesce(a, b) AS c, tags FROM t",
			columns:     []querier_dto.OutputColumn{scalarColumn("id"), scalarColumn("c"), arrayColumn("tags")},
			wantSQL:     `SELECT id, coalesce(a, b) AS c, to_json(tags) AS "tags" FROM t`,
			wantWrapped: []int{2},
		},
		{
			name:        "plain DISTINCT is preserved",
			sql:         "SELECT DISTINCT tags FROM t",
			columns:     []querier_dto.OutputColumn{arrayColumn("tags")},
			wantSQL:     `SELECT DISTINCT to_json(tags) AS "tags" FROM t`,
			wantWrapped: []int{0},
		},
		{
			name:        "DISTINCT ON is preserved",
			sql:         "SELECT DISTINCT ON (id) tags FROM t",
			columns:     []querier_dto.OutputColumn{arrayColumn("tags")},
			wantSQL:     `SELECT DISTINCT ON (id) to_json(tags) AS "tags" FROM t`,
			wantWrapped: []int{0},
		},
		{
			name:        "DISTINCT ON with a nested-parenthesis key list is preserved",
			sql:         "SELECT DISTINCT ON (coalesce(a, b)) tags FROM t",
			columns:     []querier_dto.OutputColumn{arrayColumn("tags")},
			wantSQL:     `SELECT DISTINCT ON (coalesce(a, b)) to_json(tags) AS "tags" FROM t`,
			wantWrapped: []int{0},
		},
		{
			name:        "comment between DISTINCT ON and the projection is preserved",
			sql:         "SELECT DISTINCT ON (id) -- pick latest\ntags FROM t",
			columns:     []querier_dto.OutputColumn{arrayColumn("tags")},
			wantSQL:     "SELECT DISTINCT ON (id) -- pick latest\nto_json(tags) AS \"tags\" FROM t",
			wantWrapped: []int{0},
		},
		{
			name:        "SELECT * expands into an explicit projection",
			sql:         "SELECT * FROM users",
			columns:     []querier_dto.OutputColumn{srcColumn("id", "users", querier_dto.TypeCategoryText), srcColumn("tags", "users", querier_dto.TypeCategoryArray)},
			wantSQL:     `SELECT "users"."id", to_json("users"."tags") AS "tags" FROM users`,
			wantWrapped: []int{1},
		},
		{
			name:        "qualified table.* expands",
			sql:         "SELECT u.* FROM users u",
			columns:     []querier_dto.OutputColumn{srcColumn("id", "u", querier_dto.TypeCategoryText), srcColumn("tags", "u", querier_dto.TypeCategoryArray)},
			wantSQL:     `SELECT "u"."id", to_json("u"."tags") AS "tags" FROM users u`,
			wantWrapped: []int{1},
		},
		{
			name:        "reserved-word qualifier and column are quoted",
			sql:         "SELECT * FROM orders AS \"order\"",
			columns:     []querier_dto.OutputColumn{srcColumn("id", "order", querier_dto.TypeCategoryText), srcColumn("order", "order", querier_dto.TypeCategoryArray)},
			wantSQL:     `SELECT "order"."id", to_json("order"."order") AS "order" FROM orders AS "order"`,
			wantWrapped: []int{1},
		},
		{
			name:        "DISTINCT ON with SELECT * expands and preserves the key list",
			sql:         "SELECT DISTINCT ON (id) * FROM users",
			columns:     []querier_dto.OutputColumn{srcColumn("id", "users", querier_dto.TypeCategoryText), srcColumn("tags", "users", querier_dto.TypeCategoryArray)},
			wantSQL:     `SELECT DISTINCT ON (id) "users"."id", to_json("users"."tags") AS "tags" FROM users`,
			wantWrapped: []int{1},
		},
		{
			name:        "INSERT RETURNING wraps an array column",
			sql:         "INSERT INTO t (name) VALUES ($1) RETURNING id, tags",
			columns:     []querier_dto.OutputColumn{scalarColumn("id"), arrayColumn("tags")},
			wantSQL:     `INSERT INTO t (name) VALUES ($1) RETURNING id, to_json(tags) AS "tags"`,
			wantWrapped: []int{1},
		},
		{
			name:        "INSERT RETURNING preserves a trailing semicolon",
			sql:         "INSERT INTO t (name) VALUES ($1) RETURNING id, tags;",
			columns:     []querier_dto.OutputColumn{scalarColumn("id"), arrayColumn("tags")},
			wantSQL:     `INSERT INTO t (name) VALUES ($1) RETURNING id, to_json(tags) AS "tags";`,
			wantWrapped: []int{1},
		},
		{
			name:        "INSERT with a semicolon in a string literal terminates on the real semicolon",
			sql:         "INSERT INTO t (a) VALUES (';') RETURNING id, tags;",
			columns:     []querier_dto.OutputColumn{scalarColumn("id"), arrayColumn("tags")},
			wantSQL:     `INSERT INTO t (a) VALUES (';') RETURNING id, to_json(tags) AS "tags";`,
			wantWrapped: []int{1},
		},
		{
			name:        "UPDATE RETURNING wraps an array column",
			sql:         "UPDATE t SET name = $1 WHERE id = $2 RETURNING tags",
			columns:     []querier_dto.OutputColumn{arrayColumn("tags")},
			wantSQL:     `UPDATE t SET name = $1 WHERE id = $2 RETURNING to_json(tags) AS "tags"`,
			wantWrapped: []int{0},
		},
		{
			name:        "DELETE RETURNING wraps an array column",
			sql:         "DELETE FROM t WHERE id = $1 RETURNING tags",
			columns:     []querier_dto.OutputColumn{arrayColumn("tags")},
			wantSQL:     `DELETE FROM t WHERE id = $1 RETURNING to_json(tags) AS "tags"`,
			wantWrapped: []int{0},
		},
		{
			name:        "INSERT ON CONFLICT RETURNING star expands and wraps",
			sql:         "INSERT INTO t (id, tags) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET tags = EXCLUDED.tags RETURNING *",
			columns:     []querier_dto.OutputColumn{srcColumn("id", "t", querier_dto.TypeCategoryText), srcColumn("tags", "t", querier_dto.TypeCategoryArray)},
			wantSQL:     `INSERT INTO t (id, tags) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET tags = EXCLUDED.tags RETURNING "t"."id", to_json("t"."tags") AS "tags"`,
			wantWrapped: []int{1},
		},
		{
			name:        "RETURNING explicit AS alias is preserved",
			sql:         "INSERT INTO t (a) VALUES ($1) RETURNING tags AS labels",
			columns:     []querier_dto.OutputColumn{arrayColumn("labels")},
			wantSQL:     "INSERT INTO t (a) VALUES ($1) RETURNING to_json(tags) AS labels",
			wantWrapped: []int{0},
		},
		{
			name:        "INSERT from SELECT RETURNING wraps the RETURNING list, not the source SELECT",
			sql:         "INSERT INTO t (a) SELECT a FROM src RETURNING id, tags",
			columns:     []querier_dto.OutputColumn{scalarColumn("id"), arrayColumn("tags")},
			wantSQL:     `INSERT INTO t (a) SELECT a FROM src RETURNING id, to_json(tags) AS "tags"`,
			wantWrapped: []int{1},
		},
		{
			name:        "FROM-less SELECT function call array is wrapped",
			sql:         "SELECT compute(a, b)::int[] AS xs",
			columns:     []querier_dto.OutputColumn{arrayColumn("xs")},
			wantSQL:     "SELECT to_json(compute(a, b)::int[]) AS xs",
			wantWrapped: []int{0},
		},
		{
			name:        "FROM-less SELECT array with a trailing semicolon",
			sql:         "SELECT compute()::int[] AS xs;",
			columns:     []querier_dto.OutputColumn{arrayColumn("xs")},
			wantSQL:     "SELECT to_json(compute()::int[]) AS xs;",
			wantWrapped: []int{0},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			gotSQL, gotWrapped := WrapArrayColumnsAsJSON(testCase.sql, testCase.columns, "to_json", dqQuote, wrapTestMappings())
			assert.Equal(t, testCase.wantSQL, gotSQL)
			wrappedIndices := make([]int, 0, len(gotWrapped))
			for index := range gotWrapped {
				wrappedIndices = append(wrappedIndices, index)
			}
			assert.ElementsMatch(t, testCase.wantWrapped, wrappedIndices)
		})
	}
}

func TestWrapArrayColumnsAsJSONFallsBackSafely(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		sql     string
		columns []querier_dto.OutputColumn
		jsonFn  string
	}{
		{
			name:    "select star without source columns bails",
			sql:     "SELECT * FROM t",
			columns: []querier_dto.OutputColumn{scalarColumn("id"), arrayColumn("tags")},
			jsonFn:  "to_json",
		},
		{
			name:    "compound union",
			sql:     "SELECT tags FROM a UNION SELECT tags FROM b",
			columns: []querier_dto.OutputColumn{arrayColumn("tags")},
			jsonFn:  "to_json",
		},
		{
			name:    "star mixed with another item bails",
			sql:     "SELECT *, extra FROM t",
			columns: []querier_dto.OutputColumn{srcColumn("id", "t", querier_dto.TypeCategoryText), srcColumn("tags", "t", querier_dto.TypeCategoryArray), scalarColumn("extra")},
			jsonFn:  "to_json",
		},
		{
			name:    "no array columns",
			sql:     "SELECT id, name FROM t",
			columns: []querier_dto.OutputColumn{scalarColumn("id"), scalarColumn("name")},
			jsonFn:  "to_json",
		},
		{
			name:    "empty json func disables wrap",
			sql:     "SELECT tags FROM t",
			columns: []querier_dto.OutputColumn{arrayColumn("tags")},
			jsonFn:  "",
		},
		{
			name:    "embedded columns bail",
			sql:     "SELECT a.tags FROM a",
			columns: []querier_dto.OutputColumn{{Name: "tags", IsEmbedded: true, SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryArray}}},
			jsonFn:  "to_json",
		},
		{
			name:    "RETURNING star without source columns bails",
			sql:     "INSERT INTO t (a) VALUES ($1) RETURNING *",
			columns: []querier_dto.OutputColumn{scalarColumn("id"), arrayColumn("tags")},
			jsonFn:  "to_json",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			gotSQL, gotWrapped := WrapArrayColumnsAsJSON(testCase.sql, testCase.columns, testCase.jsonFn, dqQuote, wrapTestMappings())
			assert.Equal(t, testCase.sql, gotSQL, "fallback must return the SQL unchanged")
			assert.Nil(t, gotWrapped, "fallback must wrap nothing")
		})
	}
}
