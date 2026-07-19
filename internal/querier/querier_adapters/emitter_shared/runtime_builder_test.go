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
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

type fakeRuntimeBuilderStrategy struct{}

func (*fakeRuntimeBuilderStrategy) ConnectionField(_ *querier_dto.AnalysedQuery) string {
	return IdentReader
}

func (*fakeRuntimeBuilderStrategy) DBCall(field string, method string, args []ast.Expr) *ast.CallExpr {
	return goastutil.CallExpr(
		goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), field),
			method,
		),
		args...,
	)
}

func (*fakeRuntimeBuilderStrategy) QueryMethod() string     { return "QueryContext" }
func (*fakeRuntimeBuilderStrategy) QueryRowMethod() string  { return "QueryRowContext" }
func (*fakeRuntimeBuilderStrategy) ExecMethod() string      { return "ExecContext" }
func (*fakeRuntimeBuilderStrategy) ExecReturnsResult() bool { return true }

func (*fakeRuntimeBuilderStrategy) QueriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(IdentQueries))),
	)
}

func (*fakeRuntimeBuilderStrategy) ExecResultReturnType() ast.Expr {
	return goastutil.SelectorExpr("sql", "Result")
}

func (*fakeRuntimeBuilderStrategy) ExecResultImport(_ *ImportTracker) {}

func (*fakeRuntimeBuilderStrategy) NoRowsSentinel() ast.Expr {
	return goastutil.SelectorExpr("sql", "ErrNoRows")
}

func (*fakeRuntimeBuilderStrategy) NoRowsImport(tracker *ImportTracker) {
	tracker.AddImport("database/sql")
}

func (*fakeRuntimeBuilderStrategy) BuildExecRowsBody(_ []ast.Expr, _ string) []ast.Stmt {
	return nil
}

func (*fakeRuntimeBuilderStrategy) BuilderQueryCall(argsExpr ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), IdentQueriesReceiver),
				IdentReader,
			),
			"QueryContext",
		),
		Args:     []ast.Expr{goastutil.CachedIdent(IdentCtx), goastutil.CachedIdent(IdentQuery), argsExpr},
		Ellipsis: 1,
	}
}

func (*fakeRuntimeBuilderStrategy) BuilderQueryRowCall(argsExpr ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), IdentQueriesReceiver),
				IdentReader,
			),
			"QueryRowContext",
		),
		Args:     []ast.Expr{goastutil.CachedIdent(IdentCtx), goastutil.CachedIdent(IdentQuery), argsExpr},
		Ellipsis: 1,
	}
}

func (*fakeRuntimeBuilderStrategy) RuntimeBuilderImports(_ *ImportTracker) {}
func (*fakeRuntimeBuilderStrategy) NeedsSliceExpansion() bool              { return true }
func (*fakeRuntimeBuilderStrategy) PlaceholderMarker() rune                { return '?' }
func (*fakeRuntimeBuilderStrategy) ArrayJSONWrapFunc() string              { return "" }
func (*fakeRuntimeBuilderStrategy) QuoteIdentifier(name string) string     { return `"` + name + `"` }
func (*fakeRuntimeBuilderStrategy) MaxBindVariables() int                  { return 999 }
func (*fakeRuntimeBuilderStrategy) UsesNumberedParams() bool               { return false }
func (*fakeRuntimeBuilderStrategy) PreservesPlaceholderIndices() bool      { return true }
func (*fakeRuntimeBuilderStrategy) RuntimeBuilderUsesNumberedPlaceholders() bool {
	return true
}
func (*fakeRuntimeBuilderStrategy) WrapParameterAccess(access ast.Expr, _ string) ast.Expr {
	return access
}
func (*fakeRuntimeBuilderStrategy) UsesBracedNamedPlaceholders() bool { return false }
func (*fakeRuntimeBuilderStrategy) ParameterAccessImports() []string  { return nil }

func (*fakeRuntimeBuilderStrategy) ParameterAccessHelperFile(_ string) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{}, nil
}

func renderDecl(t *testing.T, decl ast.Decl) string {
	t.Helper()
	var buf bytes.Buffer
	fileSet := token.NewFileSet()
	require.NoError(t, printer.Fprint(&buf, fileSet, decl))
	return buf.String()
}

func TestBuilderCountMethodUsesWhereArgsOnly(t *testing.T) {
	strategy := &fakeRuntimeBuilderStrategy{}
	decl := buildBuilderCountMethod("SearchPostsBuilder", strategy)
	source := renderDecl(t, decl)

	assert.Contains(t, source, "builder.whereArgs", "Count must read whereArgs, not args")
	assert.NotContains(t, source, "builder.parameterCount = ", "Count must not mutate parameterCount")
	assert.NotContains(t, source, "builder.whereArgs = ", "Count must not mutate whereArgs")
	assert.Contains(t, source, "buildCountQuery",
		"Count must delegate to the pre-derived buildCountQuery helper instead of buildQuery")
}

func TestBuilderBuildQueryDoesNotMutateBuilder(t *testing.T) {
	strategy := &fakeRuntimeBuilderStrategy{}
	decl := buildBuilderBuildQueryMethod("SearchPostsBuilder", false, strategy)
	source := renderDecl(t, decl)

	assert.Contains(t, source, "args := append([]any{}, builder.whereArgs...)", "buildQuery must snapshot whereArgs into local args")
	assert.Contains(t, source, "parameterCount := builder.parameterCount", "buildQuery must seed local parameterCount from builder")
	assert.NotContains(t, source, "builder.whereArgs = append(", "buildQuery must not mutate builder.whereArgs")
	assert.NotContains(t, source, "builder.parameterCount = ", "buildQuery must not mutate builder.parameterCount")
}

func TestBuilderWhereRecordsPendingErrorWithoutPanic(t *testing.T) {
	decl := buildBuilderWhereMethod(&querier_dto.AnalysedQuery{Name: "SearchPosts"}, "SearchPostsBuilder")
	source := renderDecl(t, decl)

	assert.Contains(t, source, "clause, args, addedParams, err := pikoBuildWhereFragment(",
		"Where must capture the dispatcher error")
	assert.Contains(t, source, "if err != nil {",
		"Where must guard on the dispatcher error")
	assert.Contains(t, source, "if builder.pendingError == nil {",
		"Where must keep the first error (first error wins)")
	assert.Contains(t, source, "builder.pendingError = err",
		"Where must record the error on the builder")

	assert.NotContains(t, source, "panic(\"piko",
		"Where must not panic for the oversized-list path")
}

func TestBuilderBuildQueryReturnsPendingError(t *testing.T) {
	strategy := &fakeRuntimeBuilderStrategy{}
	source := renderDecl(t, buildBuilderBuildQueryMethod("SearchPostsBuilder", false, strategy))

	assert.Contains(t, source, "if builder.pendingError != nil {",
		"buildQuery must guard on the pending error")
	assert.Contains(t, source, "return \"\", []any{}, builder.pendingError",
		"buildQuery must return the pending error ahead of assembly")
}

func TestBuilderBuildCountQueryReturnsPendingError(t *testing.T) {
	source := renderDecl(t, buildBuilderBuildCountQueryMethod("SearchPostsBuilder", "searchpostsCountSQL", false))

	assert.Contains(t, source, "if builder.pendingError != nil {",
		"buildCountQuery must guard on the pending error")
	assert.Contains(t, source, "return \"\", builder.pendingError",
		"buildCountQuery must return the pending error ahead of assembly")
}

func TestEmittedSliceExpanderEnforcesCap(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles + runs the emitted helper")
	}

	const bindCap = 3
	sliceFile, sliceErr := EmitSliceHelperFile("main", '?', true)
	require.NoError(t, sliceErr)
	bindLimitsFile, bindErr := EmitBindLimitsHelperFile("main", bindCap)
	require.NoError(t, bindErr)

	tempDir := t.TempDir()
	writes := map[string][]byte{
		"go.mod":            []byte("module sliceexpandercap\n\ngo 1.22\n"),
		sliceFile.Name:      sliceFile.Content,
		bindLimitsFile.Name: bindLimitsFile.Content,
	}
	for name, content := range writes {
		if writeErr := os.WriteFile(filepath.Join(tempDir, name), content, 0o600); writeErr != nil {
			t.Fatalf("write %s: %v", name, writeErr)
		}
	}

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

	overQuery, overErr := pikoExpandSlicePlaceholders("SELECT id FROM posts WHERE id IN (?1)", []pikoSliceExpansionSpec{{1, 5}})
	fmt.Printf("over|is=%v|query=%q|msg=%v\n", errors.Is(overErr, errPikoTooManyBindVariables), overQuery, overErr)

	underQuery, underErr := pikoExpandSlicePlaceholders("SELECT id FROM posts WHERE id IN (?1)", []pikoSliceExpansionSpec{{1, 3}})
	fmt.Printf("under|query=%q|err=%v\n", underQuery, underErr)
}
`
	if writeErr := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(driver), 0o600); writeErr != nil {
		t.Fatalf("write driver: %v", writeErr)
	}

	command := exec.Command("go", "run", ".")
	command.Dir = tempDir
	output, runErr := command.CombinedOutput()
	require.NoErrorf(t, runErr, "driver failed: %s", output)

	combined := string(output)
	assert.NotContains(t, combined, "panic=", "slice expander must not panic on an over-cap spec")
	assert.Contains(t, combined, `over|is=true|query=""|msg=`)
	assert.Contains(t, combined, "piko: expanded query of 5 bind variables exceeds the limit of 3")
	assert.Contains(t, combined, `under|query="SELECT id FROM posts WHERE id IN (?1,?2,?3)"|err=<nil>`)
}

func TestEmittedSliceExpanderPostgresMarkerRenumbersDollarPlaceholders(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles + runs the emitted postgres helper")
	}

	const bindCap = 65535
	sliceFile, sliceErr := EmitSliceHelperFile("main", '$', true)
	require.NoError(t, sliceErr)
	bindLimitsFile, bindErr := EmitBindLimitsHelperFile("main", bindCap)
	require.NoError(t, bindErr)

	tempDir := t.TempDir()
	writes := map[string][]byte{
		"go.mod":            []byte("module slicedollar\n\ngo 1.22\n"),
		sliceFile.Name:      sliceFile.Content,
		bindLimitsFile.Name: bindLimitsFile.Content,
	}
	for name, content := range writes {
		if writeErr := os.WriteFile(filepath.Join(tempDir, name), content, 0o600); writeErr != nil {
			t.Fatalf("write %s: %v", name, writeErr)
		}
	}

	driver := `package main

import "fmt"

func main() {
	single, _ := pikoExpandSlicePlaceholders("SELECT id FROM t WHERE id IN ($1)", []pikoSliceExpansionSpec{{1, 3}})
	fmt.Printf("single|%q\n", single)

	mixed, _ := pikoExpandSlicePlaceholders("SELECT id FROM t WHERE a IN ($1) AND b = $2", []pikoSliceExpansionSpec{{1, 3}, {2, 1}})
	fmt.Printf("mixed|%q\n", mixed)

	empty, _ := pikoExpandSlicePlaceholders("SELECT id FROM t WHERE id IN ($1)", []pikoSliceExpansionSpec{{1, 0}})
	fmt.Printf("empty|%q\n", empty)
}
`
	if writeErr := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(driver), 0o600); writeErr != nil {
		t.Fatalf("write driver: %v", writeErr)
	}

	command := exec.Command("go", "run", ".")
	command.Dir = tempDir
	output, runErr := command.CombinedOutput()
	require.NoErrorf(t, runErr, "driver failed: %s", output)

	combined := string(output)
	assert.Contains(t, combined, `single|"SELECT id FROM t WHERE id IN ($1,$2,$3)"`,
		"a postgres slice must expand to numbered $N placeholders")
	assert.Contains(t, combined, `mixed|"SELECT id FROM t WHERE a IN ($1,$2,$3) AND b = $4"`,
		"a trailing scalar placeholder must renumber past the expanded slice")
	assert.Contains(t, combined, `empty|"SELECT id FROM t WHERE id IN (NULL)"`,
		"an empty postgres slice must collapse to the (NULL) sentinel")
}

func TestEmittedSliceExpanderUsesCmpCompareComparator(t *testing.T) {
	file, err := EmitSliceHelperFile("mydb", '?', true)
	require.NoError(t, err)
	source := string(file.Content)

	assert.Contains(t, source, `"cmp"`, "slice helper must import cmp for the comparator")
	assert.Contains(t, source, "cmp.Compare(a.Placeholder, b.Placeholder)",
		"slice helper must compare placeholders via cmp.Compare")
	assert.NotContains(t, source, "a.Placeholder - b.Placeholder",
		"slice helper must not use the overflow-prone subtraction comparator")
}

func TestEmittedColumnValidatorAcceptsUnicodeColumn(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles + runs the emitted helper")
	}

	helperFile, err := EmitRuntimeBuilderHelperFile("main", true)
	require.NoError(t, err)
	bindLimitsFile, bindErr := EmitBindLimitsHelperFile("main", 999)
	require.NoError(t, bindErr)

	tempDir := t.TempDir()
	writes := map[string][]byte{
		"go.mod":            []byte("module unicodecolumnvalidator\n\ngo 1.22\n"),
		helperFile.Name:     helperFile.Content,
		bindLimitsFile.Name: bindLimitsFile.Content,
	}
	for name, content := range writes {
		if writeErr := os.WriteFile(filepath.Join(tempDir, name), content, 0o600); writeErr != nil {
			t.Fatalf("write %s: %v", name, writeErr)
		}
	}

	driver := `package main

import "fmt"

func main() {
	columns := []string{"café", "имя", "名前", "über_count"}
	for _, column := range columns {
		fmt.Printf("%s|valid=%v|root=%q\n", column, pikoValidColumnExpression(column), pikoExtractColumnRoot(column))
	}
}
`
	if writeErr := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(driver), 0o600); writeErr != nil {
		t.Fatalf("write driver: %v", writeErr)
	}

	command := exec.Command("go", "run", ".")
	command.Dir = tempDir
	output, runErr := command.CombinedOutput()
	require.NoErrorf(t, runErr, "driver failed: %s", output)

	combined := string(output)
	for _, want := range []string{
		`café|valid=true|root="café"`,
		`имя|valid=true|root="имя"`,
		`名前|valid=true|root="名前"`,
		`über_count|valid=true|root="über_count"`,
	} {
		assert.Containsf(t, combined, want, "validator output missing line %q\nfull output:\n%s", want, combined)
	}
}
