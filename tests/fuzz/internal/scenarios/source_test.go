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

package scenarios

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestSuiteFixturesParseAndRetainFailureLocations(t *testing.T) {
	repository := "../../../.."
	patterns := []string{"testdata/*.go", "actions_fixture/*/testdata/*.go", "../publiccontracts/testdata/*.go", "../../testdata/targets/*.go"}
	legacyLines := map[string]map[string]int{
		"actions.go":          {`require.NoError(t, os.WriteFile(generated, wrapper,`: 47, `require.Equal(t, os.Getenv("PIKO_FUZZ_CASE_TOKEN")`: 65},
		"database.go":         {`require.NoError(t, os.WriteFile(p, file.Content,`: 72, `require.NoError(t, os.WriteFile(fixturePath,`: 89},
		"database_runtime.go": {`require.NoError(t, queries.Change(ctx,`: 83},
		"incremental.go":      {`require.NoError(t, result.server.Generate(t.Context(), piko.RunModeDevInterpreted))`: 207},
		"callbacks.go":        {`t.Run(fmt.Sprintf("pooled-%t"`: 21},
	}
	total := 0
	for _, pattern := range patterns {
		inputs, err := filepath.Glob(pattern)
		require.NoError(t, err)
		for _, input := range inputs {
			total++
			source := filepath.Join(scenarioDir, input)
			output, err := renderSuite(repository, source, nil)
			require.NoError(t, err, input)
			set := token.NewFileSet()
			_, err = parser.ParseFile(set, source, output, parser.ParseComments)
			require.NoError(t, err, input)
			for needle, want := range legacyLines[filepath.Base(input)] {
				offset := bytes.LastIndex(output, []byte(needle))
				require.GreaterOrEqual(t, offset, 0, "%s missing assertion %q", input, needle)
				var physical *token.File
				set.Iterate(func(file *token.File) bool { physical = file; return false })
				position := set.PositionFor(physical.Pos(offset), true)
				assert.Equal(t, want, position.Line, "%s %q", input, needle)
				if filepath.Base(input) != "database_runtime.go" {
					assert.Equal(t, "assertion-"+strconv.Itoa(want), assertionID(position.String()), "lost diagnostic identity for %s", position)
				}
			}
		}
	}
	assert.Equal(t, 27, total, "checked fixture programs")
	assert.Equal(t, "release-refs", assertionID("piko_fuzz_scenario_test.go:152: PIKO_CHECK release-refs: mismatch"))
}

func TestDatabaseCompanionVariants(t *testing.T) {
	repository := "../../../.."
	for tableIndex, table := range []string{"readings", "records", "entries"} {
		for orderIndex, order := range []string{"ASC", "DESC"} {
			for convertIndex, convert := range []string{"int32(q)", "fmt.Sprint(q)"} {
				choices := []byte{byte(tableIndex), byte(orderIndex), byte(convertIndex)}
				directory := t.TempDir()
				environment, err := stageCompanions(repository, directory, "database", &casefile.Case{Data: choices})
				require.NoError(t, err)
				path := filepath.Join(directory, "runtime_test.go")
				require.Equal(t, []string{"PIKO_FUZZ_DATABASE_RUNTIME=" + path}, environment)
				inner, err := os.ReadFile(path)
				require.NoError(t, err)
				for _, want := range []string{"Quantity: " + convert, "FROM " + table + " WHERE id=99", "FROM " + table + " AS p JOIN notes"} {
					assert.Contains(t, string(inner), want, "%s/%s/%s", table, order, convert)
				}
				outer, err := renderSuite(repository, scenarioDir+"/testdata/database.go", choices)
				require.NoError(t, err)
				for _, want := range []string{`const table = "` + table + `"`, `const direction = "` + order + `"`, `const kind = "` + []string{"INTEGER", "TEXT"}[convertIndex] + `"`} {
					assert.Contains(t, string(outer), want, "schema did not emit the drawn constant")
				}
			}
		}
	}
	directory := t.TempDir()
	environment, err := stageCompanions(repository, directory, "actions", &casefile.Case{})
	require.NoError(t, err)
	for _, entry := range environment {
		_, name, _ := strings.Cut(entry, "=")
		contents, err := os.ReadFile(name)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(name, directory+string(filepath.Separator)), "wrapper companion escaped staging")
		_, err = parser.ParseFile(token.NewFileSet(), name, contents, parser.ParseComments)
		require.NoError(t, err)
	}
	assert.Len(t, environment, 2, "expected both action companions")
}

func TestIncrementalGraphGoEditAndRestore(t *testing.T) {
	const rootBase = casefile.Module
	const oldDeclaration = "package graphmodel\n// request + and example.org/pikofuzzcase/pkg are documentation.\nfunc Label(request string)string{return request + \"-oak\"}\n"
	const sfcBegin = `<template><code>example.org/pikofuzzcase/pkg</code></template>
<script type="application/x-go">
package main
import p "example.org/pikofuzzcase/pkg/attrs"
import q "example.org/pikofuzzcase2/pkg"
const note="example.org/pikofuzzcase/pkg/attrs"
type Response struct{ Text string }
func Render(props p.Props) Response {return Response{Text:props.Label+"@"+props.Site}}
</script>
<style>code {color: green}</style>`
	for _, helper := range []string{"lib/graphmodel/label.go", "pkg/graphmodel/label.go", "pkg/graphtext/text.go"} {
		t.Run(helper, func(t *testing.T) {
			files := map[string]string{"pages/partial-graph.pk": sfcBegin, "partials/graph/leaf.pk": sfcBegin, helper: oldDeclaration}
			if helper == "pkg/graphtext/text.go" {
				files[helper] = strings.Replace(oldDeclaration, "func Label", "func Prefix", 1)
				files["pkg/graphmodel/label.go"] = "package graphmodel\nfunc Label(request string)string{return formatLabel(request)}"
				files["pkg/graphmodel/format.go"] = "package graphmodel\nimport phrase \"" + rootBase + "/pkg/graphtext\"\nfunc formatLabel(request string)string{return phrase.Prefix(request)}"
			}
			original := files[helper]
			directory := t.TempDir()
			require.NoError(t, stageGraph(directory, files))
			read := func(version, name string) string {
				t.Helper()
				contents, err := os.ReadFile(filepath.Join(directory, version, name))
				require.NoError(t, err)
				return string(contents)
			}
			base, changed := read("base", helper), read("edited", helper)
			assert.NotContains(t, base, `"-changed"`)
			assert.Contains(t, changed, `request + "-changed" + "-oak"`)
			assert.Contains(t, changed, "// request + and example.org/pikofuzzcase/pkg")
			baseLeaf, editedLeaf := read("base", "partials/graph/leaf.pk"), read("edited", "partials/graph/leaf.pk")
			assert.Contains(t, editedLeaf, `"edited:"`)
			assert.NotContains(t, baseLeaf, `"edited:"`)
			for _, want := range []string{`p "testcase_cache_partial_07/pkg/attrs"`, `q "example.org/pikofuzzcase2/pkg"`, `const note = "example.org/pikofuzzcase/pkg/attrs"`, `<code>example.org/pikofuzzcase/pkg</code>`, `<style>code {color: green}</style>`} {
				assert.Contains(t, editedLeaf, want)
			}
			if helper == "pkg/graphtext/text.go" {
				assert.Contains(t, read("base", "pkg/graphmodel/format.go"), `phrase "testcase_cache_partial_07/pkg/graphtext"`, "transitive import/alias was not relocated")
			}
			require.NoError(t, stageGraph(directory, files))
			assert.Equal(t, base, read("base", helper), "base edition drifted after replaying edit and restore")
			assert.Equal(t, baseLeaf, read("base", "partials/graph/leaf.pk"))
			assert.Equal(t, original, files[helper])
		})
	}
}

func TestStageGraphRejectsUnsupportedCases(t *testing.T) {
	require.ErrorIs(t, stageGraph(t.TempDir(), map[string]string{"pages/partial-graph.pk": ""}), errGraphSource)
}

func TestCallbackOperandIsTheDrawnGoExpression(t *testing.T) {
	seen := map[string]bool{}
	for index := range 48 {
		data := []byte{byte(index)}
		source, err := renderSuite("../../../..", scenarioDir+"/testdata/callbacks.go", data)
		require.NoError(t, err)
		file, err := parser.ParseFile(token.NewFileSet(), "callback.go", source, 0)
		require.NoError(t, err)
		found := false
		for node := range ast.Preorder(file) {
			spec, ok := node.(*ast.ValueSpec)
			if !ok || len(spec.Names) != 1 || spec.Names[0].Name != "invocation" {
				continue
			}
			literal, ok := spec.Values[0].(*ast.BasicLit)
			require.True(t, ok, "invocation is not a literal")
			expression, err := strconv.Unquote(literal.Value)
			require.NoError(t, err)
			_, err = parser.ParseExpr(expression)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("library.Ledger(%d).Advance(delta)", 1+sha256.Sum256(data)[0]%6), expression)
			seen[expression] = true
			found = true
		}
		require.True(t, found, "missing callback")
	}
	assert.Len(t, seen, 6, "callback operands")
}

func TestFixtureStringRejectsAmbiguousConstants(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", "package fixture\nconst a, b = \"x\", \"y\"\nconst c = 1\n", 0)
	require.NoError(t, err)
	_, err = fixtureString(file, "missing")
	require.ErrorIs(t, err, errFixtureShape)
	_, err = fixtureString(file, "c")
	require.ErrorIs(t, err, errFixtureShape)
}
