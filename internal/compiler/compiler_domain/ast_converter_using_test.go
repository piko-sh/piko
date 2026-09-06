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
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/esbuild/js_ast"
)

var (
	usingDeclarationPattern = regexp.MustCompile(`\busing\s+[A-Za-z_$][\w$]*\s*(=|of\b)`)
)

func TestConvertUsingDeclarations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		code            string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "block using declarations become registered consts",
			code: "function f() { using a = open('a'), b = open('b'); return a; }",
			wantContains: []string{
				"const$$using0=$$pikoUsingScope();",
				"consta=$$pikoUsingAdd($$using0,open(\"a\"),false),b=$$pikoUsingAdd($$using0,open(\"b\"),false);",
				"catch($$usingError0){$$pikoUsingFail($$using0,$$usingError0);}",
				"finally{$$pikoUsingDispose($$using0);}",
			},
			wantNotContains: []string{"await$$pikoUsingDispose"},
		},
		{
			name: "await using declarations await their disposal",
			code: "async function f() { await using a = open('a'); using b = open('b'); }",
			wantContains: []string{
				"consta=$$pikoUsingAdd($$using0,open(\"a\"),true);",
				"constb=$$pikoUsingAdd($$using0,open(\"b\"),false);",
				"finally{await$$pikoUsingDispose($$using0);}",
			},
		},
		{
			name: "nested blocks get their own scopes",
			code: "function f() { using a = open('a'); { using b = open('b'); } }",
			wantContains: []string{
				"$$pikoUsingAdd($$using0,open(\"a\"),false)",
				"$$pikoUsingAdd($$using1,open(\"b\"),false)",
			},
		},
		{
			name: "for-of using disposes each value",
			code: "function f(items) { for (using item of items) { use(item); } }",
			wantContains: []string{
				"for(const$$usingValue0ofitems){const$$using0=$$pikoUsingScope();",
				"constitem=$$pikoUsingAdd($$using0,$$usingValue0,false);",
			},
		},
		{
			name: "for await of with await using awaits each disposal",
			code: "async function f(items) { for await (await using item of items) { use(item); } }",
			wantContains: []string{
				"forawait(const$$usingValue0ofitems)",
				"constitem=$$pikoUsingAdd($$using0,$$usingValue0,true);",
				"finally{await$$pikoUsingDispose($$using0);}",
			},
		},
		{
			name: "for loop using disposes once the loop ends",
			code: "function f() { for (using held = open('h'); more(); ) { step(); } }",
			wantContains: []string{
				"{const$$using0=$$pikoUsingScope();try{constheld=$$pikoUsingAdd($$using0,open(\"h\"),false);for(;more();)",
			},
		},
		{
			name: "labelled for loop keeps its label on the loop",
			code: "function f() { outer: for (using held = open('h'); more(); ) { for (;;) { continue outer; } } }",
			wantContains: []string{
				"try{constheld=$$pikoUsingAdd($$using0,open(\"h\"),false);outer:for(;more();)",
				"continueouter;",
			},
		},
		{
			name: "class static block using declarations are lowered",
			code: "class Holder { static { using held = open('h'); } }",
			wantContains: []string{
				"static{const$$using0=$$pikoUsingScope();",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			emitted := convertAndPrint(t, tc.code)
			compact := strings.Join(strings.Fields(emitted), "")
			for _, want := range tc.wantContains {
				assert.Contains(t, compact, want)
			}
			for _, unwanted := range tc.wantNotContains {
				assert.NotContains(t, compact, unwanted)
			}
			assert.Equal(t, 1, strings.Count(emitted, "function $$pikoUsingDispose("), "the disposal helpers are emitted once")
			assert.NotRegexp(t, usingDeclarationPattern, emitted, "no using declaration may reach the output")
			minifyEmittedJS(t, emitted)
		})
	}
}

func TestConvertUsingDeclarations_HelpersOnlyWhenUsed(t *testing.T) {
	t.Parallel()

	emitted := convertAndPrint(t, "function f() { const a = open('a'); return a; }")
	assert.NotContains(t, emitted, "$$pikoUsing")
}

func TestConvertUsingDeclarations_Unsupported(t *testing.T) {
	t.Parallel()

	usingLocal := func(kind js_ast.LocalKind, declarations ...js_ast.Decl) *js_ast.SLocal {
		return &js_ast.SLocal{Kind: kind, Decls: declarations}
	}
	registry := NewRegistryContext()
	named := js_ast.Decl{Binding: registry.MakeBinding("held"), ValueOrNil: js_ast.Expr{Data: &js_ast.ENull{}}}

	t.Run("using outside a block", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, registry)

		_, err := converter.convertStatement(js_ast.Stmt{Data: usingLocal(js_ast.LocalUsing, named)})
		require.ErrorIs(t, err, errUsingUnsupported)
		assert.Contains(t, err.Error(), "directly in a block")
	})

	t.Run("using inside a switch case", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, registry)

		_, err := converter.convertCaseClause(js_ast.Case{Body: []js_ast.Stmt{{Data: usingLocal(js_ast.LocalUsing, named)}}})
		require.ErrorIs(t, err, errUsingUnsupported)
	})

	t.Run("using without a value", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, registry)

		_, err := converter.convertStatementList([]js_ast.Stmt{{Data: usingLocal(js_ast.LocalUsing, js_ast.Decl{Binding: registry.MakeBinding("held")})}})
		require.ErrorIs(t, err, errUsingUnsupported)
	})

	t.Run("for-of using with two declarations", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, registry)

		_, err := converter.convertSForOf(&js_ast.SForOf{
			Init:  js_ast.Stmt{Data: usingLocal(js_ast.LocalUsing, named, named)},
			Value: js_ast.Expr{Data: &js_ast.EArray{}},
			Body:  js_ast.Stmt{Data: &js_ast.SBlock{}},
		})
		require.ErrorIs(t, err, errUsingUnsupported)
	})
}

func TestScanUsingDeclarations(t *testing.T) {
	t.Parallel()

	local := func(kind js_ast.LocalKind) js_ast.Stmt {
		return js_ast.Stmt{Data: &js_ast.SLocal{Kind: kind}}
	}

	testCases := []struct {
		name         string
		statements   []js_ast.Stmt
		wantHasUsing bool
		wantIsAsync  bool
	}{
		{name: "no statements"},
		{name: "plain declarations", statements: []js_ast.Stmt{local(js_ast.LocalConst), local(js_ast.LocalLet), {Data: &js_ast.SEmpty{}}}},
		{name: "using declaration", statements: []js_ast.Stmt{local(js_ast.LocalConst), local(js_ast.LocalUsing)}, wantHasUsing: true},
		{name: "await using declaration", statements: []js_ast.Stmt{local(js_ast.LocalUsing), local(js_ast.LocalAwaitUsing)}, wantHasUsing: true, wantIsAsync: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			hasUsing, isAsync := scanUsingDeclarations(tc.statements)
			assert.Equal(t, tc.wantHasUsing, hasUsing)
			assert.Equal(t, tc.wantIsAsync, isAsync)
		})
	}
}
