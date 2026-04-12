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

package interp_domain

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreprocessShortVar_RewritesTopLevel(t *testing.T) {
	t.Parallel()
	out, err := preprocessShortVar(`a := 1`)
	require.NoError(t, err)
	require.Equal(t, "var a = 1", out)
}

func TestPreprocessShortVar_IndentedLeftAlone(t *testing.T) {
	t.Parallel()
	out, err := preprocessShortVar(`  a := 1`)
	require.NoError(t, err)
	require.Equal(t, "  a := 1", out)
}

func TestPreprocessShortVar_ControlKeywordLeftAlone(t *testing.T) {
	t.Parallel()
	cases := []string{
		`if x := 1; x > 0 { _ = x }`,
		`for i := 0; i < 1; i++ { _ = i }`,
		`switch x := 1; x { case 1: _ = x }`,
		`go run()`,
		`defer cleanup()`,
		`return result`,
	}
	for _, code := range cases {
		out, err := preprocessShortVar(code)
		require.NoError(t, err)
		require.Equal(t, code, out, "control-keyword line must pass through unchanged: %q", code)
	}
}

func TestPreprocessShortVar_MultiValueErrors(t *testing.T) {
	t.Parallel()
	_, err := preprocessShortVar(`a, b := f()`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple names")
}

func TestPreprocessShortVar_BlankIdentErrors(t *testing.T) {
	t.Parallel()
	_, err := preprocessShortVar(`_ := f()`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "blank identifier")
}

func TestPreprocessShortVar_InsideBracesLeftAlone(t *testing.T) {
	t.Parallel()
	source := `func main() {
a := 1
}`
	out, err := preprocessShortVar(source)
	require.NoError(t, err)
	require.Equal(t, source, out, "lines inside a func body must stay untouched")
}

func TestPreprocessShortVar_PreservesMultiLineProgram(t *testing.T) {
	t.Parallel()
	source := `import "fmt"
a := 42
func run() string { return fmt.Sprintf("%d", a) }`
	out, err := preprocessShortVar(source)
	require.NoError(t, err)
	require.Contains(t, out, "var a = 42")
	require.Contains(t, out, `import "fmt"`)
	require.Contains(t, out, `func run()`)
}

func TestPreprocessShortVar_EmptyInputReturnsEmpty(t *testing.T) {
	t.Parallel()
	out, err := preprocessShortVar("")
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestPreprocessShortVar_RHSWithBraces(t *testing.T) {
	t.Parallel()
	out, err := preprocessShortVar(`x := []int{1, 2, 3}`)
	require.NoError(t, err)
	require.Equal(t, "var x = []int{1, 2, 3}", out)
}

func TestStartsWithControlKeyword_AllPrefixes(t *testing.T) {
	t.Parallel()
	positives := []string{
		"if x := 1", "if\tx := 1",
		"for i := 0", "for\ti := 0",
		"switch x := 1", "switch\tx := 1",
		"select ", "select\t",
		"go run()", "go\trun()",
		"defer cleanup()", "defer\tcleanup()",
		"return result", "return\tresult",
	}
	for _, code := range positives {
		require.True(t, startsWithControlKeyword(code), "expected positive for %q", code)
	}

	negatives := []string{
		"a := 1",
		"if(x)",
		"ifx := 1",
		"goto label",
		"",
	}
	for _, code := range negatives {
		require.False(t, startsWithControlKeyword(code), "expected negative for %q", code)
	}
}

func TestContainsTopLevelDefine(t *testing.T) {
	t.Parallel()
	require.True(t, containsTopLevelDefine(`a := 1`))
	require.True(t, containsTopLevelDefine(`x := struct{}{}`),
		"top-level := before a brace block must be detected")
	require.False(t, containsTopLevelDefine(`map[string]struct{}{a: {}}`),
		"no := token at all")
	require.False(t, containsTopLevelDefine(`func f() { a := 1 }`),
		"detect should require depth zero")
	require.False(t, containsTopLevelDefine(``), "empty input")
}

func TestParseSingleStatement_HappyPath(t *testing.T) {
	t.Parallel()
	stmt, ok := parseSingleStatement(`a := 1`)
	require.True(t, ok)
	assign, isAssign := stmt.(*ast.AssignStmt)
	require.True(t, isAssign, "expected *ast.AssignStmt, got %T", stmt)
	require.Len(t, assign.Lhs, 1)
	require.Len(t, assign.Rhs, 1)
}

func TestParseSingleStatement_ParseErrorReturnsFalse(t *testing.T) {
	t.Parallel()
	_, ok := parseSingleStatement(`!!!`)
	require.False(t, ok)
}

func TestParseSingleStatement_MultiStmtReturnsFalse(t *testing.T) {
	t.Parallel()
	_, ok := parseSingleStatement(`a := 1; b := 2`)
	require.False(t, ok)
}

func TestBraceTracker_NestingAndUnnesting(t *testing.T) {
	t.Parallel()
	tracker := newBraceTracker()
	require.True(t, tracker.atTopLevel())

	tracker.advance(`func main() {`)
	require.False(t, tracker.atTopLevel(), "after entering func body, not at top level")

	tracker.advance(`if true {`)
	require.False(t, tracker.atTopLevel(), "still inside two braces")

	tracker.advance(`}`)
	require.False(t, tracker.atTopLevel(), "one close still inside func")

	tracker.advance(`}`)
	require.True(t, tracker.atTopLevel(), "two closes returns to package scope")
}

func TestBraceTracker_ExtraCloseDoesNotUnderflow(t *testing.T) {
	t.Parallel()
	tracker := newBraceTracker()
	tracker.advance(`}`)
	require.True(t, tracker.atTopLevel(), "unmatched close should clamp depth to zero")
}

func TestBraceTracker_BracesInStringLiteralIgnored(t *testing.T) {
	t.Parallel()
	tracker := newBraceTracker()
	tracker.advance(`s := "{}"`)
	require.True(t, tracker.atTopLevel(),
		"braces inside string literal should not affect depth")
}

func TestRenderExprList(t *testing.T) {
	t.Parallel()
	expressions := []ast.Expr{
		&ast.Ident{Name: "a"},
		&ast.Ident{Name: "b"},
		&ast.BasicLit{Value: "3"},
	}
	require.Equal(t, "a, b, 3", renderExprList(expressions))
}

func TestExprAsString_FallbackBranch(t *testing.T) {
	t.Parallel()
	require.Equal(t, "a", exprAsString(&ast.Ident{Name: "a"}))
	require.Equal(t, "42", exprAsString(&ast.BasicLit{Value: "42"}))
	require.Equal(t, "<expr>", exprAsString(&ast.BinaryExpr{
		X: &ast.Ident{Name: "a"}, Y: &ast.Ident{Name: "b"},
	}),
		"binary expression should hit the fallback branch")
}

func TestPreprocessShortVar_TrailingWhitespaceTolerated(t *testing.T) {
	t.Parallel()
	out, err := preprocessShortVar("a := 1   \t\r")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "var a = 1"),
		"trailing whitespace should not break the rewrite")
}
