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
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func runtimeBuilderQuery() *querier_dto.AnalysedQuery {
	return &querier_dto.AnalysedQuery{
		Name:                    "SearchPosts",
		Filename:                "posts.sql",
		SQL:                     "SELECT id, title FROM posts WHERE environment_id = ?1",
		CountSQL:                "SELECT COUNT(*) FROM posts WHERE environment_id = ?1",
		Command:                 querier_dto.QueryCommandMany,
		DynamicRuntime:          true,
		ReadOnly:                true,
		BaseQueryHasWhereClause: true,
		Parameters: []querier_dto.QueryParameter{
			textParam("environment_id", 1),
		},
		AllowedColumns: []querier_dto.AllowedColumn{
			{Name: "title", SourceExpression: "posts.title", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			{Name: "created_at", SourceExpression: "posts.created_at", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
		},
		OutputColumns: []querier_dto.OutputColumn{textColumn("id", false), textColumn("title", false)},
	}
}

func TestBuildRuntimeBuilderDeclarationsEmitsFullBuilderSurface(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := runtimeBuilderQuery()
	tracker := NewImportTracker()

	declarations := BuildRuntimeBuilderDeclarations(query, textMappings(), tracker, strategy)
	require.NotEmpty(t, declarations)

	var builder strings.Builder
	for _, decl := range declarations {
		builder.WriteString(renderDecl(t, decl))
		builder.WriteByte('\n')
	}
	source := builder.String()

	assert.Contains(t, source, "searchpostsAllowedColumns = map[string]string")
	assert.Contains(t, source, `"title": "\"posts\".\"title\""`)
	assert.Contains(t, source, "type SearchPostsBuilder struct")
	assert.Contains(t, source, "func (queries *Queries) SearchPosts(ctx context.Context, environmentID string) *SearchPostsBuilder")
	assert.Contains(t, source, "func (builder *SearchPostsBuilder) Where(")
	assert.Contains(t, source, "func (builder *SearchPostsBuilder) OrderBy(column string, direction string) *SearchPostsBuilder")
	assert.Contains(t, source, "func (builder *SearchPostsBuilder) Limit(n int) *SearchPostsBuilder")
	assert.Contains(t, source, "func (builder *SearchPostsBuilder) Offset(n int) *SearchPostsBuilder")
	assert.Contains(t, source, "func (builder *SearchPostsBuilder) All(ctx context.Context) ([]SearchPostsRow, error)")
	assert.Contains(t, source, "func (builder *SearchPostsBuilder) One(ctx context.Context) (SearchPostsRow, error)")
	assert.Contains(t, source, "func (builder *SearchPostsBuilder) Count(ctx context.Context) (int64, error)")
	assert.Contains(t, source, "func (builder *SearchPostsBuilder) buildCountQuery() (string, error)")

	assert.Contains(t, source, "builder.limitValue = n")
	assert.Contains(t, source, "builder.offsetValue = n")

	assert.Contains(t, source, "pikoNormaliseDirection(direction)")
	assert.Contains(t, source, "resolvedColumn, columnAllowed := searchpostsAllowedColumns[columnRoot]")
	assert.Contains(t, source, "column = strings.Replace(column, columnRoot, resolvedColumn, 1)")
}

func TestBuildRuntimeBuilderDeclarationsSkipsCountWhenNoCountSQL(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := runtimeBuilderQuery()
	query.CountSQL = ""
	tracker := NewImportTracker()

	declarations := BuildRuntimeBuilderDeclarations(query, textMappings(), tracker, strategy)

	var builder strings.Builder
	for _, decl := range declarations {
		builder.WriteString(renderDecl(t, decl))
		builder.WriteByte('\n')
	}
	source := builder.String()

	assert.NotContains(t, source, "func (builder *SearchPostsBuilder) Count(")
	assert.NotContains(t, source, "buildCountQuery")
}

func TestEmitQueryFileRuntimeBuilderProducesValidGo(t *testing.T) {
	strategy := &indexedStrategy{preservesIndices: true}
	query := runtimeBuilderQuery()

	file, err := EmitQueryFile("mypkg", query.Filename, []*querier_dto.AnalysedQuery{query}, textMappings(), strategy, nil)
	require.NoError(t, err)

	source := string(file.Content)
	requireValidGo(t, file.Content)
	assert.Contains(t, source, "type SearchPostsBuilder struct")
	assert.Contains(t, source, "const searchpostsCountSQL = ")
}
