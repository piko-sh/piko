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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func embeddedColumn(name, embedTable string, isOuter bool) querier_dto.OutputColumn {
	return querier_dto.OutputColumn{
		Name:         name,
		SQLType:      querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
		IsEmbedded:   true,
		EmbedTable:   embedTable,
		EmbedIsOuter: isOuter,

		Nullable: isOuter,
	}
}

func TestEmitQueryFileEmbedQueryEmitsNestedStructs(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:     "GetOrder",
		Filename: "orders.sql",
		SQL:      "SELECT o.id, o.total, u.id, u.name FROM orders o JOIN users u ON u.id = o.user_id LEFT JOIN coupons c ON c.id = o.coupon_id",
		Command:  querier_dto.QueryCommandMany,
		ReadOnly: true,
		OutputColumns: []querier_dto.OutputColumn{
			textColumn("id", false),
			embeddedColumn("name", "users", false),
			embeddedColumn("code", "coupons", true),
		},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Contains(t, source, "type GetOrderUsers struct")
	assert.Contains(t, source, "type GetOrderCoupons struct")
	assert.Contains(t, source, "type GetOrderRow struct")

	assert.Regexp(t, `Users\s+GetOrderUsers`, source)
	assert.Regexp(t, `Coupons\s+\*GetOrderCoupons`, source)

	assert.Contains(t, source, "row.Coupons = &GetOrderCoupons{}")
	assert.Contains(t, source, "if row.Coupons.Code == nil")
}

func TestEmitQueryFileOuterEmbedNonNilableSentinelSkipsNilCheck(t *testing.T) {

	strategy := &indexedStrategy{preservesIndices: true}
	valueColumn := querier_dto.OutputColumn{
		Name:         "code",
		SQLType:      querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
		IsEmbedded:   true,
		EmbedTable:   "coupons",
		EmbedIsOuter: true,
		Nullable:     false,
	}
	query := &querier_dto.AnalysedQuery{
		Name:     "GetOrder",
		Filename: "orders.sql",
		SQL:      "SELECT o.id, c.code FROM orders o LEFT JOIN coupons c ON c.id = o.coupon_id",
		Command:  querier_dto.QueryCommandMany,
		ReadOnly: true,
		OutputColumns: []querier_dto.OutputColumn{
			textColumn("id", false),
			valueColumn,
		},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Contains(t, source, "row.Coupons = &GetOrderCoupons{}")
	assert.NotContains(t, source, "row.Coupons.Code == nil",
		"a value-typed sentinel must not produce a nil comparison")
}

func TestEmitQueryFileOuterEmbedPicksFirstNilableSentinel(t *testing.T) {

	strategy := &indexedStrategy{preservesIndices: true}
	valueColumn := querier_dto.OutputColumn{
		Name:         "code",
		SQLType:      querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
		IsEmbedded:   true,
		EmbedTable:   "coupons",
		EmbedIsOuter: true,
		Nullable:     false,
	}
	nilableColumn := querier_dto.OutputColumn{
		Name:         "label",
		SQLType:      querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
		IsEmbedded:   true,
		EmbedTable:   "coupons",
		EmbedIsOuter: true,
		Nullable:     true,
	}
	query := &querier_dto.AnalysedQuery{
		Name:     "GetOrder",
		Filename: "orders.sql",
		SQL:      "SELECT o.id, c.code, c.label FROM orders o LEFT JOIN coupons c ON c.id = o.coupon_id",
		Command:  querier_dto.QueryCommandMany,
		ReadOnly: true,
		OutputColumns: []querier_dto.OutputColumn{
			textColumn("id", false),
			valueColumn,
			nilableColumn,
		},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.NotContains(t, source, "row.Coupons.Code == nil")
	assert.Contains(t, source, "if row.Coupons.Label == nil")
}

func TestEmitQueryFileCollidingColumnNamesProduceCompilableStruct(t *testing.T) {

	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:     "GetThings",
		Filename: "things.sql",
		SQL:      "SELECT a AS foo_bar, b AS foo__bar FROM things",
		Command:  querier_dto.QueryCommandMany,
		ReadOnly: true,
		OutputColumns: []querier_dto.OutputColumn{
			textColumn("foo_bar", false),
			textColumn("foo__bar", false),
		},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Contains(t, source, "FooBar  string")
	assert.Contains(t, source, "FooBar2 string")
	assert.Contains(t, source, "&row.FooBar")
	assert.Contains(t, source, "&row.FooBar2")
}

func TestEmitQueryFileLeadingDigitColumnNameProducesCompilableStruct(t *testing.T) {

	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:     "GetUser",
		Filename: "users.sql",
		SQL:      `SELECT enabled AS "2fa_enabled" FROM users`,
		Command:  querier_dto.QueryCommandMany,
		ReadOnly: true,
		OutputColumns: []querier_dto.OutputColumn{
			textColumn("2fa_enabled", false),
		},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Contains(t, source, "_2faEnabled string")
	assert.Contains(t, source, "&row._2faEnabled")
}

func TestEmitQueryFileGroupedQueryEmitsGroupingScaffolding(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:       "ListOrdersWithItems",
		Filename:   "orders.sql",
		SQL:        "SELECT o.id, i.sku FROM orders o JOIN items i ON i.order_id = o.id",
		Command:    querier_dto.QueryCommandMany,
		ReadOnly:   true,
		GroupByKey: []string{"orders.id"},
		OutputColumns: []querier_dto.OutputColumn{
			embeddedColumn("id", "orders", false),
			embeddedColumn("sku", "items", false),
		},
	}

	require.True(t, HasGroupByKey(query), "fixture must be a group_by query")

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Regexp(t, `Items\s+\[\]ListOrdersWithItemsItems`, source)

	assert.Contains(t, source, "groupIndex")
	assert.Contains(t, source, "groupOrder")
	assert.Contains(t, source, "exists")
}

func TestGroupByKeyColumnAndTableSplitOnDot(t *testing.T) {
	dotted := &querier_dto.AnalysedQuery{GroupByKey: []string{"orders.id"}}
	assert.Equal(t, "id", GroupByKeyColumn(dotted))
	assert.Equal(t, "orders", GroupByKeyTable(dotted))

	bare := &querier_dto.AnalysedQuery{GroupByKey: []string{"id"}}
	assert.Equal(t, "id", GroupByKeyColumn(bare))
	assert.Equal(t, "", GroupByKeyTable(bare))

	none := &querier_dto.AnalysedQuery{}
	assert.Equal(t, "", GroupByKeyColumn(none))
	assert.Equal(t, "", GroupByKeyTable(none))
}

func TestGroupColumnsByEmbedSeparatesFlatAndGroupedColumns(t *testing.T) {
	columns := []querier_dto.OutputColumn{
		textColumn("id", false),
		embeddedColumn("name", "users", false),
		embeddedColumn("email", "users", false),
		embeddedColumn("code", "coupons", true),
	}

	flat, groups := GroupColumnsByEmbed(columns)
	require.Len(t, flat, 1)
	assert.Equal(t, "id", flat[0].Name)

	require.Len(t, groups, 2)
	assert.Equal(t, "users", groups[0].TableName)
	assert.Len(t, groups[0].Columns, 2)
	assert.False(t, groups[0].IsOuter)
	assert.Equal(t, "coupons", groups[1].TableName)
	assert.True(t, groups[1].IsOuter)
}

func TestEmbedStructNameAndDetailEmbed(t *testing.T) {
	assert.Equal(t, "GetOrderUsers", EmbedStructName("GetOrder", "users"))

	keyGroup := EmbedGroup{TableName: "orders"}
	detailGroup := EmbedGroup{TableName: "items"}
	assert.False(t, IsGroupByDetailEmbed(keyGroup, "orders"), "the key table is not a detail embed")
	assert.True(t, IsGroupByDetailEmbed(detailGroup, "orders"), "a non-key table is a detail embed")
	assert.False(t, IsGroupByDetailEmbed(detailGroup, ""), "no key table means no detail embeds")
}

func TestEmitQueryFileDynamicGroupedQueryEmitsParamPreamble(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:       "ListOrdersForUser",
		Filename:   "orders.sql",
		SQL:        "SELECT o.id, i.sku FROM orders o JOIN items i ON i.order_id = o.id WHERE o.user_id = ?1",
		Command:    querier_dto.QueryCommandMany,
		IsDynamic:  true,
		ReadOnly:   true,
		GroupByKey: []string{"orders.id"},
		Parameters: []querier_dto.QueryParameter{
			textParam("user_id", 1),
		},
		OutputColumns: []querier_dto.OutputColumn{
			embeddedColumn("id", "orders", false),
			embeddedColumn("sku", "items", false),
		},
	}

	require.True(t, HasGroupByKey(query), "fixture must be a group_by query")

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)
	assert.Contains(t, source, "params ListOrdersForUserParams")
	assert.Contains(t, source, "params.UserID")
	assert.Contains(t, source, "groupIndex")
}

func TestEmitQueryFileMultiParamAnonymousEngineOrdersArgs(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: false}
	query := &querier_dto.AnalysedQuery{
		Name:     "MatchPair",
		Filename: "pairs.sql",
		SQL:      "SELECT id FROM pairs WHERE a = ?2 AND b = ?1 AND c = ?2",
		Command:  querier_dto.QueryCommandOne,
		ReadOnly: true,
		Parameters: []querier_dto.QueryParameter{
			textParam("first", 1),
			textParam("second", 2),
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Contains(t, source, "WHERE a = ? AND b = ? AND c = ?")
	assert.Contains(t, source, "params.Second, params.First, params.Second")
}

func TestEmitQueryFileStreamSliceEmitsYieldGuard(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:     "StreamByIDs",
		Filename: "stream.sql",
		SQL:      "SELECT id FROM users WHERE id IN (?1)",
		Command:  querier_dto.QueryCommandStream,
		ReadOnly: true,
		Parameters: []querier_dto.QueryParameter{
			{
				Name:    "ids",
				Number:  1,
				IsSlice: true,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
				Kind:    querier_dto.ParameterDirectiveParam,
			},
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)
	assert.Contains(t, source, "pikoExpandSlicePlaceholders(streambyids,")
	assert.Contains(t, source, "yield(StreamByIDsRow{}, expansionError)")
}

func TestEmitQueryFileRequiresRowTypeEmitsEmptyStub(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:          "MysteryRows",
		Filename:      "mystery.sql",
		SQL:           "SELECT * FROM mystery_view",
		Command:       querier_dto.QueryCommandMany,
		ReadOnly:      true,
		OutputColumns: nil,
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)
	assert.Regexp(t, `type MysteryRowsRow struct \{\s*\}`, source)
}

func TestEmitQueryFileDynamicAnonymousPlaceholderOrdersArgs(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: false}
	query := &querier_dto.AnalysedQuery{
		Name:      "FindThings",
		Filename:  "things.sql",
		SQL:       "SELECT id FROM things WHERE a = ?2 AND b = ?1 AND c = ?2",
		Command:   querier_dto.QueryCommandMany,
		IsDynamic: true,
		ReadOnly:  true,
		Parameters: []querier_dto.QueryParameter{
			textParam("first", 1),
			textParam("second", 2),
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Contains(t, source, "params.Second, params.First, params.Second")
}
