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

package emitter_go_sql

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func batchInsertQuery(command querier_dto.QueryCommand) *querier_dto.AnalysedQuery {
	return &querier_dto.AnalysedQuery{
		Name:     "InsertEvents",
		Command:  command,
		SQL:      "INSERT INTO events (name, value) VALUES (?, ?)",
		Filename: "events.sql",
		Parameters: []querier_dto.QueryParameter{
			{Number: 1, Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			{Number: 2, Name: "value", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
		},
	}
}

func findGeneratedFile(t *testing.T, files []querier_dto.GeneratedFile, name string) string {
	t.Helper()
	for _, file := range files {
		if file.Name == name {
			return string(file.Content)
		}
	}
	require.Failf(t, "missing generated file", "expected %q in %v", name, fileNames(files))
	return ""
}

func fileNames(files []querier_dto.GeneratedFile) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name)
	}
	return names
}

func TestEmitQueriesBatchCommandEmitsChunkedMultiRowInsert(t *testing.T) {
	emitter := NewSQLEmitter()

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{batchInsertQuery(querier_dto.QueryCommandBatch)}, defaultMappings())
	require.NoError(t, err)

	source := findGeneratedFile(t, files, "events.sql.go")

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "events.sql.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated code must be valid Go:\n%s", source)

	assert.Contains(t, source, "type InsertEventsParams struct")
	assert.Contains(t, source, "func (queries *Queries) InsertEvents(ctx context.Context, params []InsertEventsParams) error")

	assert.Contains(t, source, "if len(params) == 0 {")

	assert.Contains(t, source, "for offset := 0; offset < len(params); offset += 499 {")
	assert.Contains(t, source, "end := min(offset+499, len(params))")
	assert.Contains(t, source, "chunk := params[offset:end]")

	assert.Contains(t, source, `values.WriteString("(?,?)")`)
	assert.Contains(t, source, "args = append(args, item.Name)")
	assert.Contains(t, source, "args = append(args, item.Value)")
	assert.Contains(t, source, "values.WriteString(\", \")")

	assert.Contains(t, source, "queries.writer.ExecContext(ctx, pikoBatchExpandValues(insertevents, values.String()), args...)")
}

func TestEmitQueriesBatchCommandEmitsExpanderHelperFile(t *testing.T) {
	emitter := NewSQLEmitter()

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{batchInsertQuery(querier_dto.QueryCommandBatch)}, defaultMappings())
	require.NoError(t, err)

	helperSource := findGeneratedFile(t, files, "batch_helpers.go")

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "batch_helpers.go", helperSource, parser.AllErrors)
	require.NoError(t, parseError, "generated helper must be valid Go:\n%s", helperSource)

	assert.Contains(t, helperSource, "func pikoBatchExpandValues(query string, multiValues string) string")

	assert.Contains(t, helperSource, `pikoBatchValuesKeyword = regexp.MustCompile("(?i)\\bVALUES\\b")`)
	assert.Contains(t, helperSource, "loc := pikoBatchValuesKeyword.FindStringIndex(query)")

	assert.Contains(t, helperSource, "func pikoBatchTupleEnd(query string, from int) int")
	assert.Contains(t, helperSource, "tupleEnd := pikoBatchTupleEnd(query, keywordEnd)")

	assert.NotContains(t, helperSource, "func pikoBatchNumberedTuple(")
}

func TestBatchExpandValuesPreservesTrailingClause(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles + runs the emitted helper")
	}

	for _, numbered := range []bool{false, true} {
		name := "positional"
		if numbered {
			name = "numbered"
		}
		t.Run(name, func(t *testing.T) {
			source := batchHelperSource("main", numbered)
			tempDir := t.TempDir()
			writeBatchHelperModule(t, tempDir, source)

			output := runBatchHelperDriver(t, tempDir)

			wantLines := []string{

				`plain|out="INSERT INTO t (a, b) VALUES (?, ?), (?, ?)"`,

				`on_conflict|out="INSERT INTO t (a, b) VALUES (?, ?), (?, ?) ON CONFLICT (a) DO NOTHING"`,

				`returning|out="INSERT INTO t (a) VALUES (?), (?) RETURNING id"`,

				`nested_call|out="INSERT INTO t (a, b) VALUES (?, ?), (?, ?) RETURNING id"`,

				`literal_paren|out="INSERT INTO t (a) VALUES (?), (?) ON CONFLICT DO NOTHING"`,
			}
			for _, line := range wantLines {
				assert.Containsf(t, output, line, "driver output missing line %q\nfull output:\n%s", line, output)
			}
		})
	}
}

func writeBatchHelperModule(t *testing.T, dir, helperSource string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module batchhelpermain\n\ngo 1.22\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "batch_helpers.go"), []byte(helperSource), 0o600); err != nil {
		t.Fatalf("write helper: %v", err)
	}
}

func runBatchHelperDriver(t *testing.T, dir string) string {
	t.Helper()
	driver := `package main

import "fmt"

func main() {
	cases := []struct {
		name        string
		query       string
		multiValues string
	}{
		{name: "plain", query: "INSERT INTO t (a, b) VALUES (?, ?)", multiValues: "(?, ?), (?, ?)"},
		{name: "on_conflict", query: "INSERT INTO t (a, b) VALUES (?, ?) ON CONFLICT (a) DO NOTHING", multiValues: "(?, ?), (?, ?)"},
		{name: "returning", query: "INSERT INTO t (a) VALUES (?) RETURNING id", multiValues: "(?), (?)"},
		{name: "nested_call", query: "INSERT INTO t (a, b) VALUES (?, COALESCE(?, 0)) RETURNING id", multiValues: "(?, ?), (?, ?)"},
		{name: "literal_paren", query: "INSERT INTO t (a) VALUES ('a)b') ON CONFLICT DO NOTHING", multiValues: "(?), (?)"},
	}
	for _, c := range cases {
		fmt.Printf("%s|out=%q\n", c.name, pikoBatchExpandValues(c.query, c.multiValues))
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(driver), 0o600); err != nil {
		t.Fatalf("write driver: %v", err)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	output, runErr := cmd.CombinedOutput()
	require.NoErrorf(t, runErr, "driver failed: %s", output)
	return string(output)
}

func TestEmitQueriesCopyFromCommandReusesMultiRowInsert(t *testing.T) {
	emitter := NewSQLEmitter()
	query := batchInsertQuery(querier_dto.QueryCommandCopyFrom)

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{query}, defaultMappings())
	require.NoError(t, err)

	source := findGeneratedFile(t, files, "events.sql.go")

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "events.sql.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated code must be valid Go:\n%s", source)

	assert.Contains(t, source, "type InsertEventsParams struct")
	assert.Contains(t, source, "func (queries *Queries) InsertEvents(ctx context.Context, params []InsertEventsParams) error")
	assert.Contains(t, source, "pikoBatchExpandValues(insertevents, values.String())")

	helperSource := findGeneratedFile(t, files, "batch_helpers.go")
	assert.Contains(t, helperSource, "func pikoBatchExpandValues(")
}

func TestEmitQueriesBatchCommandForMySQLKeepsPlainTuple(t *testing.T) {
	emitter := NewSQLEmitterForMySQL()

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{batchInsertQuery(querier_dto.QueryCommandBatch)}, defaultMappings())
	require.NoError(t, err)

	source := findGeneratedFile(t, files, "events.sql.go")

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "events.sql.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated code must be valid Go:\n%s", source)

	assert.Contains(t, source, "offset += 32767")

	assert.Contains(t, source, `values.WriteString("(?,?)")`)
	helperSource := findGeneratedFile(t, files, "batch_helpers.go")
	assert.NotContains(t, helperSource, "func pikoBatchNumberedTuple(")
}

func TestEmitQueriesBatchCommandSingleColumnUsesOneTupleSlot(t *testing.T) {
	emitter := NewSQLEmitter()
	query := &querier_dto.AnalysedQuery{
		Name:     "InsertNames",
		Command:  querier_dto.QueryCommandBatch,
		SQL:      "INSERT INTO names (label) VALUES (?)",
		Filename: "names.sql",
		Parameters: []querier_dto.QueryParameter{
			{Number: 1, Name: "label", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
		},
	}

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{query}, defaultMappings())
	require.NoError(t, err)

	source := findGeneratedFile(t, files, "names.sql.go")

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "names.sql.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated code must be valid Go:\n%s", source)

	assert.Contains(t, source, "offset += 999")
	assert.Contains(t, source, `values.WriteString("(?)")`)
	assert.Contains(t, source, "args = append(args, item.Label)")
}

func TestEmitQueriesBatchCommandGuardsRowExceedingBindCap(t *testing.T) {
	emitter := NewSQLEmitter()

	const columnsPerRow = maxSQLiteBindVariables + 1
	parameters := make([]querier_dto.QueryParameter, 0, columnsPerRow)
	for index := range columnsPerRow {
		parameters = append(parameters, querier_dto.QueryParameter{
			Number:  index + 1,
			Name:    "field" + strconv.Itoa(index),
			SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger},
		})
	}
	query := &querier_dto.AnalysedQuery{
		Name:       "InsertWide",
		Command:    querier_dto.QueryCommandBatch,
		SQL:        "INSERT INTO wide (cols) VALUES (?)",
		Filename:   "wide.sql",
		Parameters: parameters,
	}

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{query}, defaultMappings())
	require.NoError(t, err)

	source := findGeneratedFile(t, files, "wide.sql.go")

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "wide.sql.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated code must be valid Go:\n%s", source)

	assert.Contains(t, source, `"fmt"`)
	assert.Contains(t, source,
		`return fmt.Errorf("piko: each batch row binds %d variables, which exceeds the per-statement limit of %d", 1000, 999)`)

	emptyGuardOffset := strings.Index(source, "if len(params) == 0 {")
	oversizedGuardOffset := strings.Index(source, "exceeds the per-statement limit")
	require.GreaterOrEqual(t, emptyGuardOffset, 0)
	require.GreaterOrEqual(t, oversizedGuardOffset, 0)
	assert.Less(t, emptyGuardOffset, oversizedGuardOffset,
		"the empty-params no-op must precede the oversized-row guard")
}

func TestEmitQueriesBatchCommandWritesSeparatorOnlyBetweenRows(t *testing.T) {
	emitter := NewSQLEmitter()

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{batchInsertQuery(querier_dto.QueryCommandBatch)}, defaultMappings())
	require.NoError(t, err)

	source := findGeneratedFile(t, files, "events.sql.go")

	separatorOffset := strings.Index(source, `if i > 0 {`)
	require.GreaterOrEqual(t, separatorOffset, 0, "expected the separator guard in:\n%s", source)
	assert.Less(t, separatorOffset, strings.Index(source, `values.WriteString(", ")`),
		"the separator guard must precede the separator write")
}

func TestCharLitProducesValidGoRuneLiterals(t *testing.T) {

	tests := []struct {
		input    rune
		expected string
	}{
		{'(', "'('"},
		{')', "')'"},
		{',', "','"},
		{'$', "'$'"},
		{'?', "'?'"},
		{'\'', `'\''`},
		{'\\', `'\\'`},
		{'\n', `'\n'`},
	}

	for _, test := range tests {
		t.Run(strconv.QuoteRune(test.input), func(t *testing.T) {
			literal := charLit(test.input)
			assert.Equal(t, test.expected, literal.Value)
			assert.Equal(t, token.CHAR, literal.Kind)
		})
	}
}
