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

func generatedFileByName(t *testing.T, files []querier_dto.GeneratedFile, name string) querier_dto.GeneratedFile {
	t.Helper()
	for index := range files {
		if files[index].Name == name {
			return files[index]
		}
	}
	t.Fatalf("expected a generated file named %q, got %d files", name, len(files))
	return querier_dto.GeneratedFile{}
}

func TestEmitQueriesGroupsByFilenameAndEmitsSharedHelpers(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}

	queries := []*querier_dto.AnalysedQuery{
		{
			Name:          "GetUser",
			Filename:      "users.sql",
			SQL:           "SELECT id FROM users WHERE id = ?1",
			Command:       querier_dto.QueryCommandOne,
			ReadOnly:      true,
			Parameters:    []querier_dto.QueryParameter{textParam("id", 1)},
			OutputColumns: []querier_dto.OutputColumn{textColumn("id", false)},
		},
		{
			Name:     "PostsByIDs",
			Filename: "posts.sql",
			SQL:      "SELECT id FROM posts WHERE id IN (?1)",
			Command:  querier_dto.QueryCommandMany,
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
		},
	}

	files, err := EmitQueries("mypkg", queries, textMappings(), strategy, nil)
	require.NoError(t, err)

	usersFile := generatedFileByName(t, files, "users.sql.go")
	postsFile := generatedFileByName(t, files, "posts.sql.go")
	requireValidGo(t, usersFile.Content)
	requireValidGo(t, postsFile.Content)

	bindLimits := generatedFileByName(t, files, "bind_limits.go")
	sliceHelpers := generatedFileByName(t, files, "slice_helpers.go")
	requireValidGo(t, bindLimits.Content)
	requireValidGo(t, sliceHelpers.Content)

	assert.Contains(t, string(bindLimits.Content), "const pikoMaxBindVariables = 999")
	assert.Contains(t, string(sliceHelpers.Content), "func pikoExpandSlicePlaceholders(")
}

func TestEmitQueriesEmitsRuntimeBuilderHelperWhenDynamicRuntimeUsed(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := runtimeBuilderQuery()

	files, err := EmitQueries("mypkg", []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	runtimeHelpers := generatedFileByName(t, files, "runtime_helpers.go")
	bindLimits := generatedFileByName(t, files, "bind_limits.go")
	requireValidGo(t, runtimeHelpers.Content)
	requireValidGo(t, bindLimits.Content)
	assert.Contains(t, string(runtimeHelpers.Content), "func pikoBuildWhereFragment(")
}

func TestEmitQueriesReturnsNilForNoQueries(t *testing.T) {
	files, err := EmitQueries("mypkg", nil, textMappings(), &indexedStrategy{}, nil)
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestEmitModelsBuildsStructPerTable(t *testing.T) {
	catalogue := &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name: "public",
				Tables: map[string]*querier_dto.Table{
					"users": {
						Name:   "users",
						Schema: "public",
						Columns: []querier_dto.Column{
							{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
							{Name: "tags", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}, ArrayDimensions: 1},
						},
					},
				},
			},
			"tenancy": {
				Name: "tenancy",
				Tables: map[string]*querier_dto.Table{
					"accounts": {
						Name:   "accounts",
						Schema: "tenancy",
						Columns: []querier_dto.Column{
							{Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
						},
					},
				},
			},
		},
	}

	files, err := EmitModels("mypkg", catalogue, textMappings())
	require.NoError(t, err)
	require.Len(t, files, 1)

	modelsFile := files[0]
	require.Equal(t, "models.go", modelsFile.Name)
	source := string(modelsFile.Content)
	requireValidGo(t, modelsFile.Content)

	assert.Contains(t, source, "type Users struct")
	assert.Contains(t, source, "Tags []string")

	assert.Contains(t, source, "type TenancyAccounts struct")
}

func TestEmitModelsReturnsNilForEmptyCatalogue(t *testing.T) {
	files, err := EmitModels("mypkg", &querier_dto.Catalogue{Schemas: map[string]*querier_dto.Schema{}}, textMappings())
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestEmitOTelBuildsQueryNameMap(t *testing.T) {
	queries := []*querier_dto.AnalysedQuery{
		{Name: "list_tasks", Command: querier_dto.QueryCommandMany},
		{Name: "create_task", Command: querier_dto.QueryCommandExec},

		{Name: "bulk_insert", Command: querier_dto.QueryCommandCopyFrom},
	}

	file, err := EmitOTel("mypkg", queries)
	require.NoError(t, err)
	require.Equal(t, "otel.go", file.Name)

	source := string(file.Content)
	requireValidGo(t, file.Content)
	assert.Contains(t, source, "var queryNameMap = map[string]string")
	assert.Contains(t, source, "listTasks:")
	assert.Contains(t, source, "\"ListTasks\"")
	assert.Contains(t, source, "func QueryNameResolver(")
	assert.NotContains(t, source, "bulkInsert:")
}
