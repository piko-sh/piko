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
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestEmitOTel_StaticQueries(t *testing.T) {
	t.Parallel()

	queries := []*querier_dto.AnalysedQuery{
		{Name: "list_tasks", SQL: "SELECT id, name FROM tasks"},
		{Name: "create_task", SQL: "INSERT INTO tasks (name) VALUES ($1)"},
		{Name: "get_task_by_id", SQL: "SELECT id, name FROM tasks WHERE id = $1"},
	}

	file, err := EmitOTel("generated", queries)
	require.NoError(t, err)
	assert.Equal(t, "otel.go", file.Name)

	source := string(file.Content)

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "otel.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated otel.go must be valid Go:\n%s", source)

	assert.Contains(t, source, "queryNameMap")
	assert.Contains(t, source, "QueryNameResolver")
	assert.Contains(t, source, `"ListTasks"`)
	assert.Contains(t, source, `"CreateTask"`)
	assert.Contains(t, source, `"GetTaskByID"`)
}

func TestEmitOTel_IncludesDynamicQueries(t *testing.T) {
	t.Parallel()

	queries := []*querier_dto.AnalysedQuery{
		{Name: "list_tasks", SQL: "SELECT id FROM tasks"},
		{Name: "search_tasks", IsDynamic: true, SQL: "SELECT id FROM tasks WHERE name LIKE $1"},
		{Name: "find_tasks", DynamicRuntime: true, SQL: "SELECT id FROM tasks WHERE status = $1"},
	}

	file, err := EmitOTel("generated", queries)
	require.NoError(t, err)

	source := string(file.Content)

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "otel.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated otel.go must be valid Go:\n%s", source)

	assert.Contains(t, source, `"ListTasks"`)
	assert.Contains(t, source, `"SearchTasks"`)
	assert.Contains(t, source, `"FindTasks"`)
}

func TestEmitOTel_ExcludesCopyFromQueries(t *testing.T) {
	t.Parallel()

	queries := []*querier_dto.AnalysedQuery{
		{Name: "list_tasks", SQL: "SELECT id FROM tasks"},
		{Name: "import_tasks", Command: querier_dto.QueryCommandCopyFrom, SQL: "INSERT INTO tasks (name) VALUES ($1)"},
	}

	file, err := EmitOTel("generated", queries)
	require.NoError(t, err)

	source := string(file.Content)

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "otel.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated otel.go must be valid Go:\n%s", source)

	assert.Contains(t, source, `"ListTasks"`)
	assert.NotContains(t, source, `"ImportTasks"`)
}

func TestEmitOTel_DeduplicatesIdenticalSQL(t *testing.T) {
	t.Parallel()

	queries := []*querier_dto.AnalysedQuery{
		{Name: "get_one_task", Command: querier_dto.QueryCommandOne, SQL: "SELECT id, name FROM tasks"},
		{Name: "list_tasks", Command: querier_dto.QueryCommandMany, SQL: "SELECT id, name FROM tasks"},
	}

	file, err := EmitOTel("generated", queries)
	require.NoError(t, err)

	source := string(file.Content)

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "otel.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated otel.go must be valid Go:\n%s", source)

	assert.Equal(t, 1, strings.Count(source, "queryNameMap = map[string]string{"),
		"the query name map must be emitted exactly once")
	assert.Contains(t, source, `"GetOneTask"`, "the first query owning the SQL value should win")
	assert.NotContains(t, source, `"ListTasks"`,
		"a second query with identical SQL must be de-duplicated, not emit a duplicate map key")
}

func TestEmitOTel_EmptyQueries(t *testing.T) {
	t.Parallel()

	file, err := EmitOTel("generated", nil)
	require.NoError(t, err)

	source := string(file.Content)

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "otel.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated otel.go must be valid Go:\n%s", source)

	assert.Contains(t, source, "queryNameMap")
	assert.Contains(t, source, "QueryNameResolver")
}

func TestEmitOTel_PackageName(t *testing.T) {
	t.Parallel()

	file, err := EmitOTel("mypackage", []*querier_dto.AnalysedQuery{
		{Name: "get_user", SQL: "SELECT id FROM users"},
	})
	require.NoError(t, err)

	source := string(file.Content)
	lines := strings.Split(source, "\n")

	foundPackage := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			assert.Equal(t, "package mypackage", trimmed)
			foundPackage = true
			break
		}
	}
	assert.True(t, foundPackage)
}
