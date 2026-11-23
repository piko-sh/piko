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

package generate

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

var (
	//go:embed testdata/pre_migration.sha256
	preMigrationDigests string
)

func TestGoFixtureEmissionBaseline(t *testing.T) {
	cache := map[string][4]casefile.Case{}
	for line := range strings.SplitSeq(strings.TrimSpace(preMigrationDigests), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		require.True(t, len(fields) == 3 || len(fields) == 6, "bad golden entry: %s", line)
		target, path, hashes := fields[0], fields[1], fields[2:]
		cases, ok := cache[target]
		if !ok {
			for index := range cases {
				cases[index] = New(target, 7, uint64(index))
			}
			cache[target] = cases
		}
		for index, c := range cases {
			want := hashes[index%len(hashes)]
			got := fmt.Sprintf("%x", sha256.Sum256([]byte(c.Files[path])))
			assert.Equal(t, want, got, "%s seed 7 index %d %s", target, index, path)
		}
	}
}

func TestGoFixtureRoleShapesAndPaths(t *testing.T) {
	paths := []string{"/東京/\"first\"", "/line\nsecond", "/back\\slash", "subscriber-sample", "/東京/\"first\""}
	edition := "π\\\"\n"
	shapes := map[string]string{
		"string":      "i.role",
		"string-list": `[]string{"subscriber", i.role}`,
		"stringer":    "publicationRole(i.role)",
		"mixed-list":  `[]any{"subscriber", []any{publicationRole(i.role)}}`,
	}
	for shape, wantRole := range shapes {
		t.Run(shape, func(t *testing.T) {
			source, err := authProvider(edition, paths, shape)
			require.NoError(t, err)
			file := parseFixture(t, source)
			var role string
			var gotPaths []string
			values := map[string]bool{}
			ast.Inspect(file, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.CaseClause:
					if len(typed.List) != 1 {
						break
					}
					literal, ok := typed.List[0].(*ast.BasicLit)
					if ok && literal.Value == `"roles"` {
						statement, isReturn := typed.Body[0].(*ast.ReturnStmt)
						require.True(t, isReturn)
						var buffer bytes.Buffer
						require.NoError(t, format.Node(&buffer, token.NewFileSet(), statement.Results[0]))
						role = buffer.String()
					}
				case *ast.KeyValueExpr:
					if named(typed.Key, "PublicPaths") {
						composite, isComposite := typed.Value.(*ast.CompositeLit)
						require.True(t, isComposite)
						for _, element := range composite.Elts {
							literal, isLiteral := element.(*ast.BasicLit)
							require.True(t, isLiteral)
							value, err := strconv.Unquote(literal.Value)
							require.NoError(t, err)
							gotPaths = append(gotPaths, value)
						}
					}
				case *ast.BasicLit:
					if typed.Kind == token.STRING {
						value, err := strconv.Unquote(typed.Value)
						require.NoError(t, err)
						values[value] = true
					}
				}
				return true
			})
			assert.Equal(t, wantRole, role)
			assert.Equal(t, paths, gotPaths)
			for _, prefix := range []string{"reader-ticket-", "editor-ticket-", "subscriber-", "publisher-"} {
				assert.True(t, values[prefix+edition], "missing %q", prefix+edition)
			}
		})
	}
}

func TestGoFixtureRejectsUnknownRoleShape(t *testing.T) {
	_, err := authProvider("edition", nil, "number")
	require.ErrorContains(t, err, "unknown publication role shape")
}

func TestEditFixtureReportsStaleEdits(t *testing.T) {
	_, err := editFixture("actions/shop/place.go", []string{"never"}, func(ast.Node) string { return "" })
	require.ErrorContains(t, err, "never: expected one edit, found 0")
	_, err = editFixture("missing/fixture.go", nil, func(ast.Node) string { return "" })
	require.ErrorContains(t, err, "missing/fixture.go")
}

func TestGoFixtureIndependentEscapedValues(t *testing.T) {
	input := casefile.PartialComponentsInput{Labels: [2]string{"café \"edition\"", "last\nline"}, Edited: "東京\\path", Query: "q=π\tword", Score: -12}
	selected := "SELECTED\n\\"
	action := "FORM_SCORE/\"日本\""
	sku := "INVOICE_UNIT/\"π\""
	store, err := recordboardStore(input, selected)
	require.NoError(t, err)
	provider, err := formShimProvider(5, 72, action)
	require.NoError(t, err)
	place, err := checkoutPlace(casefile.CheckoutInput{SKU: sku, Unit: 143})
	require.NoError(t, err)
	testCases := []struct {
		source string
		values []string
	}{
		{source: store, values: []string{input.Labels[0], input.Labels[1], input.Edited, input.Query, selected}},
		{source: provider, values: []string{action}},
		{source: place, values: []string{sku}},
	}
	for _, testCase := range testCases {
		seen := map[string]int{}
		ast.Inspect(parseFixture(t, testCase.source), func(node ast.Node) bool {
			if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				value, err := strconv.Unquote(literal.Value)
				require.NoError(t, err)
				seen[value]++
			}
			return true
		})
		for _, value := range testCase.values {
			assert.Equal(t, 1, seen[value], "literal %q", value)
		}
	}
	require.Contains(t, store, "score := -12 + i")
	again, err := recordboardStore(input, selected)
	require.NoError(t, err)
	input.Score = 9
	input.Labels[0] = "other"
	changed, err := recordboardStore(input, selected)
	require.NoError(t, err)
	assert.NotEqual(t, again, changed)
	assert.Equal(t, again, store)
}

func parseFixture(t *testing.T, source string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, parser.SkipObjectResolution)
	require.NoError(t, err)
	return file
}
