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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeColumnExpressionPatternAcceptsSafeForms(t *testing.T) {
	compiled := regexp.MustCompile(runtimeColumnExpressionPattern)

	for _, sample := range []string{
		"id",
		"created_at",
		"_underscored",
		"data->'foo'",
		"data->>'foo'",
		"data->'meta'->>'kind'",
		"data->0",
		"data->'items'->0->>'name'",
		"json_extract(data, '$.foo')",
		"json_extract(data,'$.foo.bar')",
		"json_extract(payload,   '$.nested[0].key')",

		"(data)::boolean",
		"(data->>'foo')::boolean",
		"(data->>'foo')::numeric",
		"(data->'meta'->>'kind')::text",
		"(data->>'count')::integer",

		"CAST(data AS REAL)",
		"CAST(data AS INTEGER)",
		"CAST(json_extract(data, '$.foo') AS REAL)",
		"CAST(json_extract(data, '$.count') AS INTEGER)",

		"cast(json_extract(data, '$.foo') as real)",
		"(data->>'flag')::BOOL",
	} {
		assert.Truef(t, compiled.MatchString(sample), "expected %q to be accepted", sample)
	}
}

func TestRuntimeColumnExpressionPatternAcceptsUnicodeIdentifiers(t *testing.T) {

	compiled := regexp.MustCompile(runtimeColumnExpressionPattern)

	for _, sample := range []string{
		"naïve",
		"café",
		"имя",
		"名前",
		"über_count",
		"naïve->>'foo'",
		"(café)::text",
		"CAST(café AS TEXT)",
		"json_extract(café, '$.foo')",
		"ключ2",
	} {
		assert.Truef(t, compiled.MatchString(sample), "expected unicode column %q to be accepted", sample)
	}
}

func TestRuntimeColumnExpressionPatternRejectsInjection(t *testing.T) {
	compiled := regexp.MustCompile(runtimeColumnExpressionPattern)

	for _, sample := range []string{
		"category OR 1=1",
		"category->>'x' OR 1=1",
		"category;DROP TABLE users",
		"category UNION SELECT password FROM users",
		"category->>'x''; DROP TABLE x; --",
		"json_extract(data, '$.x') OR 1=1",
		"json_extract(data, '$.x'); --",
		"json_extract(data, ''); DROP TABLE x; --')",
		"json_extract(data, '$.x'/*injected*/)",
		"data->'x'/*comment*/",
		"data --comment",
		"data /* hidden */",
		"",
		"   ",
		"\t",
		"data->>",
		"->'foo'",
		"json_extract(, '$.foo')",
		"json_extract(data)",
		"1=1",
		"select 1",
		"data->'unclosed",
		"data->>unquoted",

		"(data)::pg_class",
		"(data)::int; DROP TABLE users",
		"(data)::; --",
		"CAST(data AS bogus_type)",
		"CAST(data AS REAL); DELETE FROM users",
		"CAST(data AS REAL) OR 1=1",
		"((data)::int)::text",
		"(1 + 1)::int",
		"(SELECT 1)::int",
		"CAST((SELECT 1) AS REAL)",
	} {
		assert.Falsef(t, compiled.MatchString(strings.TrimSpace(sample)), "expected %q to be rejected", sample)
	}
}

func TestRuntimeBuilderHelperContainsExpectedDirections(t *testing.T) {
	file, err := EmitRuntimeBuilderHelperFile("mypkg", true)
	require.NoError(t, err)
	source := string(file.Content)

	for _, allowed := range []string{
		`"ASC":              true,`,
		`"DESC":             true,`,
		`"ASC NULLS FIRST":  true,`,
		`"ASC NULLS LAST":   true,`,
		`"DESC NULLS FIRST": true,`,
		`"DESC NULLS LAST":  true,`,
	} {
		assert.Containsf(t, source, allowed, "emitted helper missing direction entry %q", allowed)
	}
	assert.Contains(t, source, "func pikoNormaliseDirection(direction string) string")
}

func TestRuntimeBuilderHelperRejectsInjectionShapes(t *testing.T) {
	file, err := EmitRuntimeBuilderHelperFile("mypkg", true)
	require.NoError(t, err)
	source := string(file.Content)

	for _, forbidden := range []string{
		`"ASC; DROP TABLE`,
		`"ASC, name`,
		`"NULLS LAST":`,
		`"FIRST":`,
		`"LAST":`,
		`"DESC --`,
		`"DESC/*`,
	} {
		assert.NotContainsf(t, source, forbidden, "emitted helper contains injection-shaped entry %q", forbidden)
	}
}

func TestRuntimeBuilderHelperContainsDispatcher(t *testing.T) {
	file, err := EmitRuntimeBuilderHelperFile("mypkg", true)
	require.NoError(t, err)
	source := string(file.Content)

	for _, marker := range []string{
		`func pikoIsUnaryOperator(op string) bool`,
		`case "IS NULL", "IS NOT NULL":`,
		`func pikoIsMultiOperator(op string) bool`,
		`case "IN", "NOT IN":`,
		`func pikoReflectSlice(value any) []any`,
		`func pikoBuildWhereFragment(column, operator string, value any, paramCount int) (string, []any, int, error)`,
		`return column + " " + operator, nil, 0, nil`,
		`return "0=1", nil, 0, nil`,
		`return "1=1", nil, 0, nil`,
		`return column + " " + operator + " " + pikoBuildBindPlaceholder(paramCount), []any{value}, 1, nil`,
		`func pikoBuildBindPlaceholder(paramCount int) string`,
		`return "$" + strconv.Itoa(paramCount)`,
		`func pikoMultiValueLen(value any) int`,
		`elementsLen := pikoMultiValueLen(value)`,
		`if elementsLen > pikoMaxBindVariables {`,
		`errPikoTooManyBindVariables`,
	} {
		assert.Containsf(t, source, marker, "emitted helper missing dispatcher marker %q", marker)
	}
}

func TestBindLimitsHelperContainsSharedSentinel(t *testing.T) {
	file, err := EmitBindLimitsHelperFile("mypkg", 999)
	require.NoError(t, err)
	require.Equal(t, "bind_limits.go", file.Name)

	source := string(file.Content)
	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, file.Name, source, parser.AllErrors)
	require.NoError(t, parseError, "generated bind limits helper must be valid Go:\n%s", source)

	for _, marker := range []string{
		"package mypkg",
		`const pikoMaxBindVariables = 999`,
		`var errPikoTooManyBindVariables = errors.New("piko: too many bind variables")`,
	} {
		assert.Containsf(t, source, marker, "emitted bind limits helper missing marker %q", marker)
	}
}

func TestBindLimitsHelperFallsBackForNonPositiveCap(t *testing.T) {
	for _, cap := range []int{0, -1} {
		file, err := EmitBindLimitsHelperFile("mypkg", cap)
		require.NoError(t, err)
		assert.Containsf(
			t,
			string(file.Content),
			"const pikoMaxBindVariables = 65535",
			"non-positive cap %d should fall back to the default", cap,
		)
	}
}

func TestEmitRuntimeBuilderHelperFileProducesValidGo(t *testing.T) {
	file, err := EmitRuntimeBuilderHelperFile("mypkg", true)
	require.NoError(t, err)
	require.Equal(t, "runtime_helpers.go", file.Name)

	source := string(file.Content)
	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, file.Name, source, parser.AllErrors)
	require.NoError(t, parseError, "generated helper must be valid Go:\n%s", source)

	assert.Contains(t, source, "package mypkg")
	assert.Contains(t, source, "var pikoColumnExpressionRegex = regexp.MustCompile(`"+runtimeColumnExpressionPattern+"`)")
	assert.Contains(t, source, "func pikoValidColumnExpression(")
	assert.Contains(t, source, "func pikoExtractColumnRoot(")
}

func TestEmittedDispatcherExecutes(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles + runs the emitted helper")
	}

	file, err := EmitRuntimeBuilderHelperFile("main", true)
	require.NoError(t, err)

	tempDir := t.TempDir()
	if writeErr := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module dispatchermain\n\ngo 1.21\n"), 0o600); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(tempDir, file.Name), file.Content, 0o600); writeErr != nil {
		t.Fatalf("write helper: %v", writeErr)
	}
	writeBindLimitsHelper(t, tempDir, 999)

	driver := `package main

import (
	"fmt"
)

func main() {
	cases := []struct {
		name     string
		column   string
		operator string
		value    any
		paramIn  int
	}{
		{name: "unary_is_null", column: "name", operator: "IS NULL", value: nil, paramIn: 0},
		{name: "unary_is_not_null", column: "email", operator: "IS NOT NULL", value: "ignored", paramIn: 5},
		{name: "multi_in_any", column: "id", operator: "IN", value: []any{1, 2, 3}, paramIn: 0},
		{name: "multi_in_string", column: "tag", operator: "IN", value: []string{"a", "b"}, paramIn: 7},
		{name: "multi_in_empty", column: "id", operator: "IN", value: []any{}, paramIn: 4},
		{name: "multi_notin_empty", column: "id", operator: "NOT IN", value: []any{}, paramIn: 4},
		{name: "binary_eq", column: "id", operator: "=", value: 42, paramIn: 0},
	}
	for _, c := range cases {
		clause, args, added, err := pikoBuildWhereFragment(c.column, c.operator, c.value, c.paramIn)
		fmt.Printf("%s|clause=%q|argsLen=%d|added=%d|err=%v\n", c.name, clause, len(args), added, err)
	}
}
`
	if writeErr := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(driver), 0o600); writeErr != nil {
		t.Fatalf("write driver: %v", writeErr)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	output, runErr := cmd.CombinedOutput()
	require.NoErrorf(t, runErr, "driver failed: %s", output)

	wantLines := []string{
		`unary_is_null|clause="name IS NULL"|argsLen=0|added=0|err=<nil>`,
		`unary_is_not_null|clause="email IS NOT NULL"|argsLen=0|added=0|err=<nil>`,
		`multi_in_any|clause="id IN ($1, $2, $3)"|argsLen=3|added=3|err=<nil>`,
		`multi_in_string|clause="tag IN ($8, $9)"|argsLen=2|added=2|err=<nil>`,
		`multi_in_empty|clause="0=1"|argsLen=0|added=0|err=<nil>`,
		`multi_notin_empty|clause="1=1"|argsLen=0|added=0|err=<nil>`,
		`binary_eq|clause="id = $1"|argsLen=1|added=1|err=<nil>`,
	}
	for _, line := range wantLines {
		assert.Containsf(t, string(output), line, "driver output missing line %q\nfull output:\n%s", line, output)
	}
}

func writeBindLimitsHelper(t *testing.T, dir string, maxBindVariables int) {
	t.Helper()
	file, err := EmitBindLimitsHelperFile("main", maxBindVariables)
	require.NoError(t, err)
	if writeErr := os.WriteFile(filepath.Join(dir, file.Name), file.Content, 0o600); writeErr != nil {
		t.Fatalf("write bind limits helper: %v", writeErr)
	}
}

func TestEmittedDispatcherEnforcesCap(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles + runs the emitted helper")
	}

	file, err := EmitRuntimeBuilderHelperFile("main", true)
	require.NoError(t, err)

	tempDir := t.TempDir()
	if writeErr := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module dispatchercap\n\ngo 1.21\n"), 0o600); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(tempDir, file.Name), file.Content, 0o600); writeErr != nil {
		t.Fatalf("write helper: %v", writeErr)
	}
	writeBindLimitsHelper(t, tempDir, 4)

	driver := `package main

import (
	"errors"
	"fmt"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic=%v\n", r)
		}
	}()

	overClause, overArgs, overAdded, overErr := pikoBuildWhereFragment("id", "IN", []any{1, 2, 3, 4, 5}, 0)
	fmt.Printf("over|is=%v|clause=%q|argsLen=%d|added=%d|msg=%v\n",
		errors.Is(overErr, errPikoTooManyBindVariables), overClause, len(overArgs), overAdded, overErr)

	underClause, underArgs, underAdded, underErr := pikoBuildWhereFragment("id", "IN", []any{1, 2, 3, 4}, 0)
	fmt.Printf("under|clause=%q|argsLen=%d|added=%d|err=%v\n", underClause, len(underArgs), underAdded, underErr)
}
`
	if writeErr := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(driver), 0o600); writeErr != nil {
		t.Fatalf("write driver: %v", writeErr)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	output, runErr := cmd.CombinedOutput()
	require.NoErrorf(t, runErr, "driver failed: %s", output)

	combined := string(output)
	assert.NotContains(t, combined, "panic=", "dispatcher must not panic on an over-cap slice")
	assert.Contains(t, combined, `over|is=true|clause=""|argsLen=0|added=0|msg=`)
	assert.Contains(t, combined, `piko: column "id" IN/NOT IN list of 5 exceeds the limit of 4`)
	assert.Contains(t, combined, `under|clause="id IN ($1, $2, $3, $4)"|argsLen=4|added=4|err=<nil>`)
}

func TestEmittedDirectionNormalisation(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles + runs the emitted helper")
	}

	file, err := EmitRuntimeBuilderHelperFile("main", true)
	require.NoError(t, err)

	tempDir := t.TempDir()
	if writeErr := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module dispatchernorm\n\ngo 1.21\n"), 0o600); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(tempDir, file.Name), file.Content, 0o600); writeErr != nil {
		t.Fatalf("write helper: %v", writeErr)
	}
	writeBindLimitsHelper(t, tempDir, 999)

	driver := `package main

import "fmt"

func main() {
	for _, dir := range []string{"asc", "Asc", "DESC", " desc nulls last ", "DESC NULLS FIRST"} {
		normalised := pikoNormaliseDirection(dir)
		fmt.Printf("input=%q|normalised=%q|allowed=%v\n", dir, normalised, pikoAllowedDirections[normalised])
	}
}
`
	if writeErr := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(driver), 0o600); writeErr != nil {
		t.Fatalf("write driver: %v", writeErr)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	output, runErr := cmd.CombinedOutput()
	require.NoErrorf(t, runErr, "driver failed: %s", output)

	wantLines := []string{
		`input="asc"|normalised="ASC"|allowed=true`,
		`input="Asc"|normalised="ASC"|allowed=true`,
		`input="DESC"|normalised="DESC"|allowed=true`,
		`input=" desc nulls last "|normalised="DESC NULLS LAST"|allowed=true`,
		`input="DESC NULLS FIRST"|normalised="DESC NULLS FIRST"|allowed=true`,
	}
	for _, line := range wantLines {
		assert.Containsf(t, string(output), line, "normalisation missed for %q\nfull output:\n%s", line, output)
	}
}

func TestEmittedExtractColumnRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles + runs the emitted helper")
	}

	file, err := EmitRuntimeBuilderHelperFile("main", true)
	require.NoError(t, err)

	tempDir := t.TempDir()
	if writeErr := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module castextract\n\ngo 1.21\n"), 0o600); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(tempDir, file.Name), file.Content, 0o600); writeErr != nil {
		t.Fatalf("write helper: %v", writeErr)
	}
	writeBindLimitsHelper(t, tempDir, 999)

	driver := `package main

import "fmt"

func main() {
	for _, in := range []string{
		"version_data",
		"version_data->>'foo'",
		"json_extract(version_data, '$.foo')",
		"(version_data)::boolean",
		"(version_data->>'foo')::boolean",
		"(version_data->'meta'->>'kind')::text",
		"CAST(version_data AS REAL)",
		"CAST(json_extract(version_data, '$.foo') AS REAL)",
		"cast(json_extract(version_data, '$.count') as integer)",
	} {
		fmt.Printf("%s|%s\n", in, pikoExtractColumnRoot(in))
	}
}
`
	if writeErr := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(driver), 0o600); writeErr != nil {
		t.Fatalf("write driver: %v", writeErr)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	output, runErr := cmd.CombinedOutput()
	require.NoErrorf(t, runErr, "driver failed: %s", output)

	wantLines := []string{
		`version_data|version_data`,
		`version_data->>'foo'|version_data`,
		`json_extract(version_data, '$.foo')|version_data`,
		`(version_data)::boolean|version_data`,
		`(version_data->>'foo')::boolean|version_data`,
		`(version_data->'meta'->>'kind')::text|version_data`,
		`CAST(version_data AS REAL)|version_data`,
		`CAST(json_extract(version_data, '$.foo') AS REAL)|version_data`,
		`cast(json_extract(version_data, '$.count') as integer)|version_data`,
	}
	for _, line := range wantLines {
		assert.Containsf(t, string(output), line, "extractor missed root for %q\nfull output:\n%s", line, output)
	}
}
