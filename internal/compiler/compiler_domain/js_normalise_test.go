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

package compiler_domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	parsejs "github.com/tdewolff/parse/v2/js"
	esbuildast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/js_printer"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/esbuild/renamer"
)

func reparseEmittedJS(t *testing.T, emitted string) {
	t.Helper()

	log := logger.NewDeferLog(logger.DeferLogAll, nil)
	source := logger.Source{
		Index:          0,
		KeyPath:        logger.Path{Text: "emitted.js"},
		PrettyPaths:    logger.PrettyPaths{Rel: "emitted.js", Abs: "emitted.js"},
		Contents:       emitted,
		IdentifierName: "emitted",
	}
	_, ok := js_parser.Parse(log, source, js_parser.OptionsFromConfig(&config.Options{}))

	msgs := log.Done()
	var problems []string
	for _, msg := range msgs {
		if msg.Kind == logger.Error {
			problems = append(problems, msg.Data.Text)
		}
	}
	require.Emptyf(t, problems, "emitted JS does not reparse: %q -> %v", emitted, problems)
	require.Truef(t, ok, "emitted JS does not reparse: %q", emitted)
}

func assertNormalisedJS(t *testing.T, source, want string) {
	t.Helper()

	emitted := convertAndPrint(t, source)
	reparseEmittedJS(t, emitted)
	minified := minifyEmittedJS(t, emitted)
	assert.Truef(t,
		strings.Contains(minified, want) || strings.Contains(emitted, want),
		"want %q in emitted %q (minified %q)", want, emitted, minified)

	parser := NewTypeScriptParser()
	esbuildAST, err := parser.ParseTypeScript(source, "test.ts")
	require.NoError(t, err)
	tree, err := ConvertEsbuildToTdewolff(esbuildAST, NewRegistryContext())
	require.NoError(t, err)
	require.NoError(t, normaliseAST(tree))
	once := printStatements(tree)
	require.NoError(t, normaliseAST(tree))
	assert.Equal(t, once, printStatements(tree), "normalisation is not idempotent")
}

func printStatements(tree *parsejs.AST) string {
	var builder strings.Builder
	for i, statement := range tree.List {
		if i > 0 {
			builder.WriteString("\n")
		}
		statement.JS(&builder)
	}
	return builder.String()
}

func TestNormalise_ParenBacklog(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"comma as sole call argument", `f((a, b));`, "f((a,b))"},
		{"comma as array element", `const r = [(a, b)];`, "[(a,b)]"},
		{"comma as object value", `const r = { k: (a, b) };`, "{k:(a,b)}"},
		{"assignment as ternary condition", `let a; const r = (a = b) ? x : y;`, "(a=b)?x:y"},
		{"comma in ternary branch", `const r = a ? (b, c) : d;`, "a?(b,c):d"},
		{"logical as new callee", `const r = new (A || B)();`, "new(A||B)"},
		{"call as new callee", `const r = new (a.b())();`, "new(a.b())"},
		{"logical as call target", `const r = (a || b)();`, "(a||b)()"},
		{"await over binary", `async function f() { return await (a + b); }`, "await(a+b)"},
		{"comma as declarator init", `const x = (a, b);`, "x=(a,b)"},
		{"comma as default export", `export default (a, b);`, "(a,b)"},
		{"in inside for-init", `for ((a in b); ; ) { c(); }`, "for((a in b);"},
		{"spread over comma", `f(...(a, b));`, "f(...(a,b))"},
		{"comma as for-of subject", `for (const x of (a, b)) { c(); }`, "of(a,b)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertNormalisedJS(t, tc.source, tc.want)
		})
	}
}

func TestNormalise_PrecedenceAndAssociativity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"redundant parens dropped", `const r = a + b * c;`, "a+b*c"},
		{"needed parens kept", `const r = (a + b) * c;`, "(a+b)*c"},
		{"right of left-assoc groups", `const r = a - (b - c);`, "a-(b-c)"},
		{"pow is right assoc", `const r = a ** b ** c;`, "a**b**c"},
		{"pow left needs group", `const r = (a ** b) ** c;`, "(a**b)**c"},
		{"negated pow base", `const r = (-a) ** b;`, "(-a)**b"},
		{"negative literal pow base", `const r = (-2) ** 2;`, "(-2)**2"},
		{"nullish chain stays bare", `const r = a ?? b ?? c;`, "a??b??c"},
		{"and inside nullish groups", `const r = (a && b) ?? c;`, "(a&&b)??c"},
		{"nullish inside and groups", `const r = (a ?? b) && c;`, "(a??b)&&c"},
		{"nullish as member base", `const r = (a ?? b).c;`, "(a??b).c"},
		{"logical as member base", `const r = (a || b)[i];`, "(a||b)[i]"},
		{"unary as member base", `const r = (-a).b;`, "(-a).b"},
		{"optional chain stays bare", `const r = a?.b.c;`, "a?.b.c"},
		{"optional call stays bare", `const r = a?.b();`, "a?.b()"},
		{"arrow iife", `const r = (() => 1)();`, "(()=>"},
		{"ternary nests right bare", `const r = a ? b : c ? d : e;`, "a?b:c?d:e"},
		{"ternary as condition groups", `const r = (a ? b : c) ? d : e;`, "(a?b:c)?d:e"},
		{"class heritage expression", `class A extends (B || C) {}`, "extends(B||C)"},
		{"computed key comma", `const r = { [(a, b)]: 1 };`, "[(a,b)]:1"},
		{"nullish nested right stays bare", `const r = a ?? (b ?? c);`, "a??b??c"},
		{"new with args as member base", `const r = new A().b;`, "new A().b"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertNormalisedJS(t, tc.source, tc.want)
		})
	}
}

func TestNormalise_StatementPositions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"object literal statement", `({ a: 1 });`, "({a:1})"},
		{"function expression statement", `(function () {});`, "(function("},
		{"class expression statement", `(class {});`, "(class"},
		{"destructuring assignment statement", `let a; ({ a } = b);`, "({a}=b)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertNormalisedJS(t, tc.source, tc.want)
		})
	}
}

func TestNormalise_UnaryAndYield(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"double negation keeps space", `const r = - -a;`, "- -a"},
		{"typeof over binary", `const r = typeof (a + b);`, "typeof(a+b)"},
		{"void over comma", `const r = void (a, b);`, "void(a,b)"},
		{"delete stays bare", `const r = delete a.b;`, "delete a.b"},
		{"spread argument", `f(...a);`, "f(...a)"},
		{"spread element", `const r = [...a];`, "[...a]"},
		{"spread property", `const r = { ...a };`, "{...a}"},
		{"yield over comma", `function* g() { yield (a, b); }`, "yield(a,b)"},
		{"yield delegate", `function* g() { yield* h(); }`, "yield*h()"},
		{"bare yield", `function* g() { yield; }`, "yield"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertNormalisedJS(t, tc.source, tc.want)
		})
	}
}

func TestNormalise_ArgumentlessNewAsMemberBase(t *testing.T) {
	t.Parallel()

	newExpr := &parsejs.NewExpr{X: &parsejs.Var{Data: []byte("a")}}
	access := &parsejs.DotExpr{X: newExpr, Y: parsejs.LiteralExpr{
		TokenType: parsejs.IdentifierToken,
		Data:      []byte("b"),
	}}

	result, err := normaliseExpression(access, parsejs.OpExpr)
	require.NoError(t, err)

	var builder strings.Builder
	result.JS(&builder)
	assert.Equal(t, "new a().b", builder.String(), "a nil Args must still print its call parens")
	reparseEmittedJS(t, "const r = "+builder.String()+";")
}

func TestNormalise_PreservesMarkerGroup(t *testing.T) {
	t.Parallel()

	marker := &parsejs.GroupExpr{X: &parsejs.Var{Data: []byte("value")}}
	result, err := normaliseExpression(marker, parsejs.OpExpr)
	require.NoError(t, err)

	group, ok := result.(*parsejs.GroupExpr)
	require.True(t, ok, "marker GroupExpr was replaced: %T", result)
	assert.Same(t, marker, group)

	var builder strings.Builder
	result.JS(&builder)
	assert.Equal(t, "(value)", builder.String())
}

func printViaEsbuild(t *testing.T, source string) string {
	t.Helper()

	log := logger.NewDeferLog(logger.DeferLogAll, nil)
	src := logger.Source{
		Index:          0,
		KeyPath:        logger.Path{Text: "in.js"},
		PrettyPaths:    logger.PrettyPaths{Rel: "in.js", Abs: "in.js"},
		Contents:       source,
		IdentifierName: "in",
	}
	tree, ok := js_parser.Parse(log, src, js_parser.OptionsFromConfig(&config.Options{}))
	require.Truef(t, ok, "parse failed: %q", source)

	symbols := esbuildast.NewSymbolMap(1)
	symbols.SymbolsForSource[0] = tree.Symbols
	result := js_printer.Print(tree, symbols, renamer.NewNoOpRenamer(symbols), js_printer.Options{})
	return string(result.JS)
}

func TestNormalise_SemanticEquivalence(t *testing.T) {
	t.Parallel()

	sources := []string{
		`function f(a, b) { return f((a, b)); }`,
		`function f(a, b) { return [(a, b)]; }`,
		`function f(a, b) { return { "k": (a, b) }; }`,
		`function f(a, b, x, y) { return (a = b) ? x : y; }`,
		`function f(a, b, c, d) { return a ? (b, c) : d; }`,
		`function f(A, B) { return new (A || B)(); }`,
		`function f(a, b) { return (a || b)(); }`,
		`function f(a, b) { return a + b * a; }`,
		`function f(a, b, c) { return (a + b) * c; }`,
		`function f(a, b, c) { return a - (b - c); }`,
		`function f(a, b, c) { return a ** b ** c; }`,
		`function f(a, b, c) { return (a ** b) ** c; }`,
		`function f(a, b, c) { return (a && b) ?? c; }`,
		`function f(a, b, c) { return a ?? b ?? c; }`,
		`function f(a, b) { return !(a && b); }`,
		`function f(a, b) { return typeof (a + b); }`,
		`function f(a, b, i) { return (a || b)[i]; }`,
		`function f(a) { return (-a).toString(); }`,
		`function f(a, b, c, d, e) { return a ? b : c ? d : e; }`,
		`function f(a) { return - -a; }`,
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			emitted := convertAndPrint(t, source)
			reparseEmittedJS(t, emitted)
			assert.Equal(t, printViaEsbuild(t, source), printViaEsbuild(t, emitted),
				"emitted program differs from the source program: %q", emitted)
		})
	}
}

func TestNormalise_GoldenCorpusReparses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping golden corpus sweep in short mode")
	}
	t.Parallel()

	goldens, err := filepath.Glob(filepath.Join("..", "compiler_test", "testdata", "*", "golden.js"))
	require.NoError(t, err)
	require.NotEmpty(t, goldens, "golden corpus not found")

	for _, golden := range goldens {
		t.Run(filepath.Base(filepath.Dir(golden)), func(t *testing.T) {
			t.Parallel()
			content, err := os.ReadFile(golden)
			require.NoError(t, err)
			reparseEmittedJS(t, string(content))
		})
	}
}

func FuzzNormaliseReparse(f *testing.F) {
	seeds := []string{
		`f((a, b));`,
		`const r = [(a, b)];`,
		`const r = { k: (a, b) };`,
		`let a; const r = (a = b) ? x : y;`,
		`const r = a ? (b, c) : d;`,
		`const r = new (A || B)();`,
		`const r = (a || b)();`,
		`const x = (a, b);`,
		`async function g() { return await (a + b); }`,
		`function* g() { yield (a, b); }`,
		`const r = (a ** b) ** c;`,
		`const r = (a && b) ?? c;`,
		`({ a: 1 });`,
		`for ((a in b); ; ) { c(); }`,
		`class A extends (B || C) {}`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		parser := NewTypeScriptParser()
		if _, err := parser.ParseTypeScriptStrict(source, "fuzz.ts"); err != nil {
			return
		}
		esbuildAST, err := parser.ParseTypeScript(source, "fuzz.ts")
		if err != nil {
			return
		}
		tree, err := ConvertEsbuildToTdewolff(esbuildAST, NewRegistryContext())
		if err != nil || tree == nil {
			return
		}
		printed, err := printTdewolffAST(tree)
		if err != nil {
			return
		}
		reparseEmittedJS(t, printed)
	})
}

func TestNormalise_NilSlotsAreSafe(t *testing.T) {
	t.Parallel()

	normalised, err := normaliseExpression(nil, parsejs.OpExpr)
	require.NoError(t, err)
	assert.Nil(t, normalised)
	require.NoError(t, normaliseAST(nil))
	require.NoError(t, normaliseStatement(nil))

	for _, source := range []string{
		`const r = [1, , 3];`,
		`function f() { return; }`,
		`for (;;) { break; }`,
		`try { a(); } catch { b(); }`,
	} {
		emitted := convertAndPrint(t, source)
		reparseEmittedJS(t, emitted)
	}
}

func TestNormalise_DepthLimit(t *testing.T) {
	t.Parallel()

	nestArrays := func(depth int) parsejs.IExpr {
		var expression parsejs.IExpr = &parsejs.Var{Data: []byte("a")}
		for range depth {
			expression = &parsejs.ArrayExpr{List: []parsejs.Element{{Value: expression}}}
		}
		return expression
	}

	t.Run("accepts nesting within the ceiling", func(t *testing.T) {
		t.Parallel()

		_, err := normaliseExpression(nestArrays(defaultMaxNormaliseDepth-1), parsejs.OpExpr)
		require.NoError(t, err)
	})

	t.Run("reports nesting past the ceiling", func(t *testing.T) {
		t.Parallel()

		_, err := normaliseExpression(nestArrays(defaultMaxNormaliseDepth+1), parsejs.OpExpr)
		require.Error(t, err)
		assert.ErrorIs(t, err, errNormaliseTooDeep)
	})

	t.Run("reports nesting past the ceiling for a whole program", func(t *testing.T) {
		t.Parallel()

		tree := &parsejs.AST{}
		tree.List = append(tree.List, &parsejs.ExprStmt{Value: nestArrays(defaultMaxNormaliseDepth + 1)})

		err := normaliseAST(tree)
		require.Error(t, err)
		assert.ErrorIs(t, err, errNormaliseTooDeep)
	})

	t.Run("reports nesting past the ceiling for a single statement", func(t *testing.T) {
		t.Parallel()

		err := normaliseStatement(&parsejs.ExprStmt{Value: nestArrays(defaultMaxNormaliseDepth + 1)})
		require.Error(t, err)
		assert.ErrorIs(t, err, errNormaliseTooDeep)
	})

	t.Run("refuses to print a tree that exceeded the ceiling", func(t *testing.T) {
		t.Parallel()

		tree := &parsejs.AST{}
		tree.List = append(tree.List, &parsejs.ExprStmt{Value: nestArrays(defaultMaxNormaliseDepth + 1)})

		printed, err := printTdewolffAST(tree)
		require.Error(t, err)
		assert.ErrorIs(t, err, errNormaliseTooDeep)
		assert.Empty(t, printed, "an under-bracketed program must never be emitted")
	})

	t.Run("bounds statement recursion as well as expression recursion", func(t *testing.T) {
		t.Parallel()

		var statement parsejs.IStmt = &parsejs.ExprStmt{Value: &parsejs.Var{Data: []byte("a")}}
		for range defaultMaxNormaliseDepth + 1 {
			statement = &parsejs.BlockStmt{List: []parsejs.IStmt{statement}}
		}

		err := normaliseStatement(statement)
		require.Error(t, err)
		assert.ErrorIs(t, err, errNormaliseTooDeep)
	})
}

func TestContainsTopLevelIn(t *testing.T) {
	t.Parallel()

	variable := &parsejs.Var{Data: []byte("a")}
	inOperator := &parsejs.BinaryExpr{Op: parsejs.InToken, X: variable, Y: variable}
	plain := &parsejs.BinaryExpr{Op: parsejs.AddToken, X: variable, Y: variable}

	cases := []struct {
		expression parsejs.IExpr
		name       string
		want       bool
	}{
		{name: "nil has no in", expression: nil, want: false},
		{name: "bare variable has no in", expression: variable, want: false},
		{name: "the in operator itself", expression: inOperator, want: true},
		{
			name:       "in nested on the left of another operator",
			expression: &parsejs.BinaryExpr{Op: parsejs.AddToken, X: inOperator, Y: variable},
			want:       true,
		},
		{
			name:       "in nested on the right of another operator",
			expression: &parsejs.BinaryExpr{Op: parsejs.AddToken, X: variable, Y: inOperator},
			want:       true,
		},
		{name: "unrelated binary operator", expression: plain, want: false},
		{
			name:       "in inside a ternary condition",
			expression: &parsejs.CondExpr{Cond: inOperator, X: variable, Y: variable},
			want:       true,
		},
		{
			name:       "in inside a ternary consequent",
			expression: &parsejs.CondExpr{Cond: variable, X: inOperator, Y: variable},
			want:       true,
		},
		{
			name:       "in inside a ternary alternate",
			expression: &parsejs.CondExpr{Cond: variable, X: variable, Y: inOperator},
			want:       true,
		},
		{
			name:       "ternary without an in",
			expression: &parsejs.CondExpr{Cond: variable, X: variable, Y: variable},
			want:       false,
		},
		{
			name:       "in inside a comma sequence",
			expression: &parsejs.CommaExpr{List: []parsejs.IExpr{variable, inOperator}},
			want:       true,
		},
		{
			name:       "comma sequence without an in",
			expression: &parsejs.CommaExpr{List: []parsejs.IExpr{variable, variable}},
			want:       false,
		},
		{
			name:       "in under a unary operator",
			expression: &parsejs.UnaryExpr{Op: parsejs.NotToken, X: inOperator},
			want:       true,
		},
		{
			name:       "brackets shield an in from the for-init check",
			expression: &parsejs.GroupExpr{X: inOperator},
			want:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, containsTopLevelIn(tc.expression))
		})
	}
}

func TestStatementExpressionNeedsGroup(t *testing.T) {
	t.Parallel()

	variable := &parsejs.Var{Data: []byte("a")}
	object := &parsejs.ObjectExpr{}

	cases := []struct {
		expression parsejs.IExpr
		name       string
		want       bool
	}{
		{name: "bare variable is safe", expression: variable, want: false},
		{name: "object literal at the head", expression: object, want: true},
		{name: "function expression at the head", expression: &parsejs.FuncDecl{}, want: true},
		{name: "class expression at the head", expression: &parsejs.ClassDecl{}, want: true},
		{
			name:       "object literal on the left of an operator",
			expression: &parsejs.BinaryExpr{Op: parsejs.EqToken, X: object, Y: variable},
			want:       true,
		},
		{
			name:       "object literal on the right is not at the head",
			expression: &parsejs.BinaryExpr{Op: parsejs.EqToken, X: variable, Y: object},
			want:       false,
		},
		{
			name:       "object literal as a ternary condition",
			expression: &parsejs.CondExpr{Cond: object, X: variable, Y: variable},
			want:       true,
		},
		{
			name:       "postfix operator puts its operand first",
			expression: &parsejs.UnaryExpr{Op: parsejs.PostIncrToken, X: &parsejs.DotExpr{X: object}},
			want:       true,
		},
		{
			name:       "prefix operator does not put its operand first",
			expression: &parsejs.UnaryExpr{Op: parsejs.NotToken, X: object},
			want:       false,
		},
		{name: "member access over an object literal", expression: &parsejs.DotExpr{X: object}, want: true},
		{
			name:       "index access over an object literal",
			expression: &parsejs.IndexExpr{X: object, Y: variable},
			want:       true,
		},
		{name: "call over a function expression", expression: &parsejs.CallExpr{X: &parsejs.FuncDecl{}}, want: true},
		{name: "untagged template is safe", expression: &parsejs.TemplateExpr{}, want: false},
		{
			name:       "template tagged by a function expression",
			expression: &parsejs.TemplateExpr{Tag: &parsejs.FuncDecl{}},
			want:       true,
		},
		{
			name:       "comma sequence starting with an object literal",
			expression: &parsejs.CommaExpr{List: []parsejs.IExpr{object, variable}},
			want:       true,
		},
		{name: "empty comma sequence", expression: &parsejs.CommaExpr{}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, statementExpressionNeedsGroup(tc.expression))
		})
	}
}

func TestNormalise_DeclarationAndBindingSlots(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"exported declaration", `export const x = (a, b);`, "(a,b)"},
		{"exported function", `export function f() { return (a, b); }`, "return a,b"},
		{"exported class", `export class C { m() { return (a, b); } }`, "return a,b"},
		{"parameter default", `function f(x = (a, b)) { return x; }`, "(a,b)"},
		{"rest parameter binding", `function f(...rest) { return rest; }`, "rest"},
		{"array pattern default", `function f([x = (a, b)]) { return x; }`, "(a,b)"},
		{"object pattern default", `function f({ x = (a, b) }) { return x; }`, "(a,b)"},
		{"computed method name", `const o = { [(a, b)]() { return 1; } };`, "(a,b)"},
		{"class computed member", `class C { [(a, b)]() { return 1; } }`, "[(a,b)]"},
		{"class field initialiser", `class C { x = (a, b); }`, "(a,b)"},
		{"class static block", `class C { static { f((a, b)); } }`, "(a,b)"},
		{"class heritage call", `class C extends (A || B) {}`, "(A||B)"},
		{"for-of destructuring head", `for (const [x] of xs) { f(x); }`, "of xs"},
		{"for-in assignment head", `for (x.y in obj) { f(x); }`, "in obj"},
		{"labelled statement", `outer: for (;;) { break outer; }`, "outer"},
		{"do-while condition", `do { f(); } while ((a, b));`, "(a,b)"},
		{"switch discriminant", `switch ((a, b)) { case (c, d): f(); }`, "(a,b)"},
		{"throw over comma", `function f() { throw (a, b); }`, "throw a,b"},
		{"try catch finally", `try { f((a, b)); } catch (e) { g(e); } finally { h(); }`, "(a,b)"},
		{"catch destructuring binding", `try { f(); } catch ({ message = (a, b) }) { g(message); }`, "(a,b)"},
		{"tagged template keeps its tag", "const r = tag`${(a, b)}`;", "tag`${a,b}`"},
		{"function expression as new callee", `const r = new (function () {})();`, "new (function"},
		{"getter stays an accessor", `const o = { get x() { return (a, b); } };`, "get "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			emitted := convertAndPrint(t, tc.source)
			reparseEmittedJS(t, emitted)
			minified := minifyEmittedJS(t, emitted)
			assert.Truef(t,
				strings.Contains(minified, tc.want) || strings.Contains(emitted, tc.want),
				"want %q in emitted %q (minified %q)", tc.want, emitted, minified)
		})
	}
}

func TestConvert_TaggedTemplate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		source string
		want   string
	}{
		{name: "identifier tag with substitution", source: "const r = tag`hi ${x}`;", want: "tag`hi ${x}`"},
		{name: "identifier tag without substitution", source: "const r = tag`plain`;", want: "tag`plain`"},
		{name: "member access tag", source: "const r = a.b`x`;", want: "a.b`x`"},
		{
			name:   "untagged template survives a substitution",
			source: "const r = `plain ${x}`;",
			want:   "`plain ${x}`",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			emitted := convertAndPrint(t, tc.source)
			reparseEmittedJS(t, emitted)
			assert.Contains(t, emitted, tc.want)
		})
	}
}

func TestConvert_ClassAndObjectMembers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		source  string
		notWant string
		want    []string
	}{
		{
			name:   "computed class method keeps its brackets",
			source: `class C { [key]() { return 1; } }`,
			want:   []string{"[key]"},
		},
		{
			name:   "computed static class field keeps its brackets",
			source: `class C { static [key] = 1; }`,
			want:   []string{"[key]"},
		},
		{
			name:   "class static block keeps its body",
			source: `class C { static { globalThis.ready = true; } }`,
			want:   []string{"static", "globalThis.ready"},
		},
		{
			name:    "object getter stays an accessor",
			source:  `const o = { get x() { return 1; } };`,
			want:    []string{"get "},
			notWant: `"x": function`,
		},
		{
			name:    "object setter stays an accessor",
			source:  `const o = { set x(v) { this.v = v; } };`,
			want:    []string{"set "},
			notWant: `"x": function`,
		},
		{
			name:   "object generator method keeps its star",
			source: `const o = { *g() { yield 1; } };`,
			want:   []string{"*"},
		},
		{
			name:   "object async method keeps its modifier",
			source: `const o = { async m() { return 1; } };`,
			want:   []string{"async"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			emitted := convertAndPrint(t, tc.source)
			reparseEmittedJS(t, emitted)
			for _, want := range tc.want {
				assert.Contains(t, emitted, want)
			}
			if tc.notWant != "" {
				assert.NotContains(t, emitted, tc.notWant)
			}
		})
	}
}
