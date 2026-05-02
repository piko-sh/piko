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

func dynamicSortableQuery(command querier_dto.QueryCommand) *querier_dto.AnalysedQuery {
	return &querier_dto.AnalysedQuery{
		Name:      "SearchPosts",
		Filename:  "posts.sql",
		SQL:       "SELECT id, title FROM posts WHERE author = ?1 ORDER BY created_at LIMIT ?2",
		Command:   command,
		IsDynamic: true,
		ReadOnly:  true,
		Parameters: []querier_dto.QueryParameter{
			{
				Name:    "author",
				Number:  1,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
				Kind:    querier_dto.ParameterDirectiveParam,
			},
			{
				Name:         "page_size",
				Number:       2,
				SQLType:      querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger},
				Context:      querier_dto.ParameterContextLimit,
				DefaultLimit: new(20),
				MaxLimit:     new(100),
			},
			{
				Name:            "sort",
				Number:          3,
				SQLType:         querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
				Kind:            querier_dto.ParameterDirectiveSortable,
				SortableColumns: []string{"created_at", "title"},
			},
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false), textColumn("title", false)},
	}
}

func TestEmitQueryFileDynamicQueryEmitsParamsAndSortableScaffolding(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := dynamicSortableQuery(querier_dto.QueryCommandMany)

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)

	assert.Contains(t, source, "type SearchPostsParams struct")
	assert.Contains(t, source, "type SearchPostsOrderBy string")
	assert.Contains(t, source, "SearchPostsOrderByCreatedAt SearchPostsOrderBy = \"created_at\"")
	assert.Contains(t, source, "type OrderDirection string")
	assert.Contains(t, source, "OrderDirection = \"ASC\"")

	assert.Contains(t, source, "if params.PageSize == 0")
	assert.Contains(t, source, "if params.PageSize > 100")

	assert.Contains(t, source, "query := searchposts")
	assert.Contains(t, source, "query = query + (\" ORDER BY \" + string(params.Sort))")
}

func TestEmitQueryFileDynamicCommandsProduceValidGo(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}

	tests := []struct {
		name           string
		command        querier_dto.QueryCommand
		wantResultPart string
	}{
		{name: "dynamic one", command: querier_dto.QueryCommandOne, wantResultPart: "(SearchPostsRow, error)"},
		{name: "dynamic many", command: querier_dto.QueryCommandMany, wantResultPart: "([]SearchPostsRow, error)"},
		{name: "dynamic exec", command: querier_dto.QueryCommandExec, wantResultPart: ") error {"},
		{name: "dynamic execresult", command: querier_dto.QueryCommandExecResult, wantResultPart: "(sql.Result, error)"},
		{name: "dynamic execrows", command: querier_dto.QueryCommandExecRows, wantResultPart: "(int64, error)"},
		{name: "dynamic stream", command: querier_dto.QueryCommandStream, wantResultPart: "func(yield func(SearchPostsRow, error) bool)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := dynamicSortableQuery(test.command)
			file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
			require.NoError(t, err)

			source := string(file.Content)
			requireValidGo(t, file.Content)
			assert.Containsf(t, source, test.wantResultPart, "emitted source missing %q\n%s", test.wantResultPart, source)
		})
	}
}

func TestEmitQueryFileDynamicSliceQueryEmitsExpansionAndSortable(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := &querier_dto.AnalysedQuery{
		Name:      "FilterPosts",
		Filename:  "posts.sql",
		SQL:       "SELECT id FROM posts WHERE tag IN (?1) ORDER BY id",
		Command:   querier_dto.QueryCommandMany,
		IsDynamic: true,
		ReadOnly:  true,
		Parameters: []querier_dto.QueryParameter{
			{
				Name:    "tags",
				Number:  1,
				IsSlice: true,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
				Kind:    querier_dto.ParameterDirectiveParam,
			},
			{
				Name:            "sort",
				Number:          2,
				SQLType:         querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
				Kind:            querier_dto.ParameterDirectiveSortable,
				SortableColumns: []string{"id"},
			},
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
	}

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)
	assert.Contains(t, source, "pikoExpandSlicePlaceholders(filterposts,")
	assert.Contains(t, source, "for _, v := range params.Tags")
	assert.Contains(t, source, "string(params.Sort)")
}

func TestFindSortableParameterReturnsFirstSortable(t *testing.T) {
	withSortable := dynamicSortableQuery(querier_dto.QueryCommandMany)
	found := FindSortableParameter(withSortable)
	require.NotNil(t, found)
	assert.Equal(t, "sort", found.Name)

	withoutSortable := &querier_dto.AnalysedQuery{
		Parameters: []querier_dto.QueryParameter{textParam("id", 1)},
	}
	assert.Nil(t, FindSortableParameter(withoutSortable))
}

func TestOrderByEnumTypeNameAppendsSuffix(t *testing.T) {
	assert.Equal(t, "SearchPostsOrderBy", OrderByEnumTypeName("SearchPosts"))
}
