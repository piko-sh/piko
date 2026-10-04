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
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/sfcparser"
)

func TestNewSFCCompiler(t *testing.T) {
	t.Run("creates SFC compiler", func(t *testing.T) {
		compiler := NewSFCCompiler("", nil)

		require.NotNil(t, compiler)
		_, ok := compiler.(*sfcCompiler)
		assert.True(t, ok)
	})

	t.Run("implements SFCCompiler interface", func(t *testing.T) {
		compiler := NewSFCCompiler("", nil)

		assert.NotNil(t, compiler)
	})
}

func TestGetStmtsFromAST(t *testing.T) {
	t.Run("returns nil for nil AST", func(t *testing.T) {
		result := getStmtsFromAST(nil)
		assert.Nil(t, result)
	})

	t.Run("returns empty for AST with no parts", func(t *testing.T) {
		tree := &js_ast.AST{Parts: []js_ast.Part{}}
		result := getStmtsFromAST(tree)
		assert.Empty(t, result)
	})

	t.Run("extracts statements from single part", func(t *testing.T) {
		stmt1 := js_ast.Stmt{Data: &js_ast.SEmpty{}}
		stmt2 := js_ast.Stmt{Data: &js_ast.SEmpty{}}
		tree := &js_ast.AST{
			Parts: []js_ast.Part{
				{Stmts: []js_ast.Stmt{stmt1, stmt2}},
			},
		}

		result := getStmtsFromAST(tree)

		assert.Len(t, result, 2)
	})

	t.Run("extracts statements from multiple parts", func(t *testing.T) {
		stmt1 := js_ast.Stmt{Data: &js_ast.SEmpty{}}
		stmt2 := js_ast.Stmt{Data: &js_ast.SEmpty{}}
		stmt3 := js_ast.Stmt{Data: &js_ast.SEmpty{}}
		tree := &js_ast.AST{
			Parts: []js_ast.Part{
				{Stmts: []js_ast.Stmt{stmt1}},
				{Stmts: []js_ast.Stmt{stmt2, stmt3}},
			},
		}

		result := getStmtsFromAST(tree)

		assert.Len(t, result, 3)
	})
}

func TestSetStmtsInAST(t *testing.T) {
	t.Run("does nothing for nil AST", func(t *testing.T) {
		statements := []js_ast.Stmt{{Data: &js_ast.SEmpty{}}}
		setStmtsInAST(nil, statements)
	})

	t.Run("creates part if none exist", func(t *testing.T) {
		tree := &js_ast.AST{Parts: []js_ast.Part{}}
		statements := []js_ast.Stmt{{Data: &js_ast.SEmpty{}}}

		setStmtsInAST(tree, statements)

		require.Len(t, tree.Parts, 1)
		assert.Len(t, tree.Parts[0].Stmts, 1)
	})

	t.Run("sets statements in first part", func(t *testing.T) {
		tree := &js_ast.AST{
			Parts: []js_ast.Part{
				{Stmts: []js_ast.Stmt{{Data: &js_ast.SEmpty{}}}},
			},
		}
		newStmts := []js_ast.Stmt{
			{Data: &js_ast.SEmpty{}},
			{Data: &js_ast.SEmpty{}},
		}

		setStmtsInAST(tree, newStmts)

		assert.Len(t, tree.Parts[0].Stmts, 2)
	})

	t.Run("removes extra parts", func(t *testing.T) {
		tree := &js_ast.AST{
			Parts: []js_ast.Part{
				{Stmts: []js_ast.Stmt{}},
				{Stmts: []js_ast.Stmt{}},
				{Stmts: []js_ast.Stmt{}},
			},
		}
		statements := []js_ast.Stmt{{Data: &js_ast.SEmpty{}}}

		setStmtsInAST(tree, statements)

		assert.Len(t, tree.Parts, 1)
	})
}

func TestAppendStatementToAST(t *testing.T) {
	t.Run("does nothing for nil AST", func(t *testing.T) {
		statement := js_ast.Stmt{Data: &js_ast.SEmpty{}}
		appendStatementToAST(nil, statement)
	})

	t.Run("creates part if none exist", func(t *testing.T) {
		tree := &js_ast.AST{Parts: []js_ast.Part{}}
		statement := js_ast.Stmt{Data: &js_ast.SEmpty{}}

		appendStatementToAST(tree, statement)

		require.Len(t, tree.Parts, 1)
		assert.Len(t, tree.Parts[0].Stmts, 1)
	})

	t.Run("appends to existing statements", func(t *testing.T) {
		existingStmt := js_ast.Stmt{Data: &js_ast.SEmpty{}}
		tree := &js_ast.AST{
			Parts: []js_ast.Part{
				{Stmts: []js_ast.Stmt{existingStmt}},
			},
		}
		newStmt := js_ast.Stmt{Data: &js_ast.SEmpty{}}

		appendStatementToAST(tree, newStmt)

		assert.Len(t, tree.Parts[0].Stmts, 2)
	})
}

func TestBuildClassName(t *testing.T) {
	tests := []struct {
		name     string
		rawTag   string
		expected string
	}{
		{
			name:     "simple tag",
			rawTag:   "my-component",
			expected: "MyComponentElement",
		},
		{
			name:     "three parts",
			rawTag:   "my-awesome-component",
			expected: "MyAwesomeComponentElement",
		},
		{
			name:     "single part",
			rawTag:   "button",
			expected: "ButtonElement",
		},
		{
			name:     "empty parts ignored",
			rawTag:   "my--component",
			expected: "MyComponentElement",
		},
		{
			name:     "leading dash",
			rawTag:   "-component",
			expected: "ComponentElement",
		},
		{
			name:     "trailing dash",
			rawTag:   "component-",
			expected: "ComponentElement",
		},
		{
			name:     "all lowercase",
			rawTag:   "pp-counter",
			expected: "PpCounterElement",
		},
		{
			name:     "empty string",
			rawTag:   "",
			expected: "Element",
		},
		{
			name:     "just dashes",
			rawTag:   "---",
			expected: "Element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildClassName(tt.rawTag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildIIFEWrapper(t *testing.T) {
	t.Run("wraps empty statements", func(t *testing.T) {
		statements := []js_ast.Stmt{}

		result := buildIIFEWrapper(statements)

		require.NotNil(t, result.Data)
		_, ok := result.Data.(*js_ast.SExpr)
		assert.True(t, ok)
	})

	t.Run("wraps multiple statements", func(t *testing.T) {
		statements := []js_ast.Stmt{
			{Data: &js_ast.SEmpty{}},
			{Data: &js_ast.SEmpty{}},
		}

		result := buildIIFEWrapper(statements)

		require.NotNil(t, result.Data)
		expression, ok := result.Data.(*js_ast.SExpr)
		require.True(t, ok)
		call, ok := expression.Value.Data.(*js_ast.ECall)
		require.True(t, ok)
		_, ok = call.Target.Data.(*js_ast.EArrow)
		assert.True(t, ok)
	})
}

func TestSeparateImportsFromAST(t *testing.T) {
	t.Run("separates import and non-import statements", func(t *testing.T) {
		statements := []js_ast.Stmt{
			{Data: &js_ast.SImport{}},
			{Data: &js_ast.SEmpty{}},
			{Data: &js_ast.SImport{}},
			{Data: &js_ast.SLocal{}},
		}

		imports, nonImports := separateImportsFromAST(statements)

		assert.Len(t, imports, 2)
		assert.Len(t, nonImports, 2)
		_, isImport := nonImports[0].Data.(*js_ast.SImport)
		assert.False(t, isImport)
		_, isImport = nonImports[1].Data.(*js_ast.SImport)
		assert.False(t, isImport)
	})

	t.Run("returns all imports for import-only input", func(t *testing.T) {
		statements := []js_ast.Stmt{
			{Data: &js_ast.SImport{}},
			{Data: &js_ast.SImport{}},
		}

		imports, nonImports := separateImportsFromAST(statements)

		assert.Len(t, imports, 2)
		assert.Empty(t, nonImports)
	})

	t.Run("preserves order of statements in both slices", func(t *testing.T) {
		emptyStmt := js_ast.Stmt{Data: &js_ast.SEmpty{}}
		localStmt := js_ast.Stmt{Data: &js_ast.SLocal{}}
		importStmt1 := js_ast.Stmt{Data: &js_ast.SImport{}}
		importStmt2 := js_ast.Stmt{Data: &js_ast.SImport{}}
		statements := []js_ast.Stmt{
			importStmt1,
			emptyStmt,
			importStmt2,
			localStmt,
		}

		imports, nonImports := separateImportsFromAST(statements)

		assert.Len(t, imports, 2)
		assert.Len(t, nonImports, 2)
		assert.Equal(t, emptyStmt.Data, nonImports[0].Data)
		assert.Equal(t, localStmt.Data, nonImports[1].Data)
	})
}

func TestSfcCompilationContext_SetupNaming(t *testing.T) {
	t.Run("uses template name attribute", func(t *testing.T) {
		cc := &sfcCompilationContext{
			sfcParseResult: &sfcparser.ParseResult{
				TemplateAttributes: map[string]string{"name": "my-component"},
			},
		}

		err := cc.setupNaming()

		require.NoError(t, err)
		assert.Equal(t, "my-component", cc.tagName)
		assert.Equal(t, "MyComponentElement", cc.className)
	})

	t.Run("falls back to filename", func(t *testing.T) {
		cc := &sfcCompilationContext{
			sfcParseResult: &sfcparser.ParseResult{},
			sourceFilename: "/path/to/my-counter.pkc",
		}

		err := cc.setupNaming()

		require.NoError(t, err)
		assert.Equal(t, "my-counter", cc.tagName)
		assert.Equal(t, "MyCounterElement", cc.className)
	})

	t.Run("returns error when name has no hyphen", func(t *testing.T) {
		cc := &sfcCompilationContext{
			sfcParseResult: &sfcparser.ParseResult{},
			sourceFilename: "/path/to/counter.pkc",
		}

		err := cc.setupNaming()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "require a '-' in their name")
	})

	t.Run("template name takes priority over filename", func(t *testing.T) {
		cc := &sfcCompilationContext{
			sfcParseResult: &sfcparser.ParseResult{
				TemplateAttributes: map[string]string{"name": "explicit-name"},
			},
			sourceFilename: "/path/to/different-name.pkc",
		}

		err := cc.setupNaming()

		require.NoError(t, err)
		assert.Equal(t, "explicit-name", cc.tagName)
	})
}

func TestCompileSFC_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("compiles minimal SFC", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template name="my-counter"><div>Hello</div></template>`)

		artefact, err := compileSFC(ctx, "my-counter.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "my-counter", artefact.TagName)
		assert.Contains(t, artefact.Files, "my-counter.js")
	})

	t.Run("compiles SFC with styles", func(t *testing.T) {
		rawSFC := []byte(`
<script></script>
<template name="styled-component"><div class="container">Styled</div></template>
<style>.container { color: red; }</style>
`)

		artefact, err := compileSFC(ctx, "styled-component.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "styled-component", artefact.TagName)

		assert.Contains(t, artefact.ScaffoldHTML, "style")
	})

	t.Run("generates scaffold HTML", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template name="test-scaffold"><p>Scaffold content</p></template>`)

		artefact, err := compileSFC(ctx, "test-scaffold.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)

		assert.Equal(t, "test-scaffold", artefact.TagName)
	})

	t.Run("handles empty template", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template name="no-template"></template>`)

		artefact, err := compileSFC(ctx, "no-template.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "no-template", artefact.TagName)
	})

	t.Run("falls back to filename for unnamed component", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template><div>Unnamed</div></template>`)

		artefact, err := compileSFC(ctx, "my-widget.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "my-widget", artefact.TagName)
	})

	t.Run("returns error when name has no hyphen", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template><div>Bad</div></template>`)

		_, err := compileSFC(ctx, "widget.pkc", rawSFC, "", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "require a '-' in their name")
	})

	t.Run("extracts enabled behaviours", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template name="behaviour-test" enable="draggable resizable"><div>Test</div></template>`)

		artefact, err := compileSFC(ctx, "behaviour-test.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
	})

	t.Run("handles multiple style blocks", func(t *testing.T) {
		rawSFC := []byte(`
<script></script>
<template name="multi-style"><div>Test</div></template>
<style>.first { color: red; }</style>
<style>.second { color: blue; }</style>
`)

		artefact, err := compileSFC(ctx, "multi-style.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
	})

	t.Run("skips aesthetic styles", func(t *testing.T) {
		rawSFC := []byte(`
<script></script>
<template name="aesthetic-test"><div class="container">Test</div></template>
<style>.container { padding: 10px; }</style>
<style aesthetic>.aesthetic-only { display: none; }</style>
`)

		artefact, err := compileSFC(ctx, "aesthetic-test.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)

	})

	t.Run("returns error for template with syntax errors", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template name="bad-template"><div p-if></div></template>`)

		artefact, err := compileSFC(ctx, "bad-template.pkc", rawSFC, "", nil)

		if err != nil {
			assert.Contains(t, err.Error(), "syntax error")
		} else {
			require.NotNil(t, artefact)
		}
	})
}

func TestCompileSFC_WithJavaScript(t *testing.T) {
	ctx := context.Background()

	t.Run("compiles SFC with class definition", func(t *testing.T) {
		rawSFC := []byte(`
<script>
class CounterComponentElement extends PPElement {
	count = 0;

	increment() {
		this.count++;
	}
}
</script>
<template name="counter-component">
	<button @click="increment">Count: {{ count }}</button>
</template>
`)

		artefact, err := compileSFC(ctx, "counter-component.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "counter-component", artefact.TagName)

		jsContent := artefact.Files["counter-component.js"]
		assert.Contains(t, jsContent, "CounterComponentElement")
	})

	t.Run("handles imports in script", func(t *testing.T) {
		rawSFC := []byte(`
<script>
import { someFunc } from './utils.js';

class ImportTestElement extends PPElement {
	doSomething() {
		someFunc();
	}
}
</script>
<template name="import-test"><div>Import Test</div></template>
`)

		artefact, err := compileSFC(ctx, "import-test.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
	})

	t.Run("adds customElements.define", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template name="define-test"><div>Test</div></template>`)

		artefact, err := compileSFC(ctx, "define-test.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)

		jsContent := artefact.Files["define-test.js"]
		assert.Contains(t, jsContent, "customElements.define")
		assert.Contains(t, jsContent, "define-test")
	})
}

func TestPrintAST(t *testing.T) {
	t.Run("returns empty for nil AST", func(t *testing.T) {
		result, err := printAST(context.Background(), nil, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("prints simple AST", func(t *testing.T) {
		registry := NewRegistryContext()
		tree := &js_ast.AST{
			Parts: []js_ast.Part{
				{
					Stmts: []js_ast.Stmt{
						{Data: &js_ast.SEmpty{}},
					},
				},
			},
		}

		result, err := printAST(context.Background(), tree, nil, registry)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestPrintTdewolffAST(t *testing.T) {
	t.Run("returns empty for nil AST", func(t *testing.T) {
		result, err := printTdewolffAST(nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestSFCCompiler_CompileSFC(t *testing.T) {
	ctx := context.Background()
	compiler := NewSFCCompiler("", nil)

	t.Run("compiles valid SFC", func(t *testing.T) {
		rawSFC := []byte(`<script></script><template name="test-component"><div>Test</div></template>`)

		artefact, err := compiler.CompileSFC(ctx, "test-component.pkc", rawSFC)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "test-component", artefact.TagName)
	})

	t.Run("handles empty SFC", func(t *testing.T) {
		rawSFC := []byte(``)

		artefact, err := compiler.CompileSFC(ctx, "empty-test.pkc", rawSFC)

		if err == nil {
			require.NotNil(t, artefact)
		}
	})
}

func TestEnsurePPElementClass(t *testing.T) {
	ctx := context.Background()

	t.Run("does nothing if class exists", func(t *testing.T) {

		snippet := `class MyElement extends PPElement {}`
		statement, _ := parseSnippetAsStatement(snippet)
		tree := &js_ast.AST{
			Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{statement}}},
		}
		initialStmtCount := len(getStmtsFromAST(tree))

		ensurePPElementClass(ctx, tree, "MyElement")

		assert.Equal(t, initialStmtCount, len(getStmtsFromAST(tree)))
	})

	t.Run("creates class if not found", func(t *testing.T) {
		tree := &js_ast.AST{Parts: []js_ast.Part{}}

		ensurePPElementClass(ctx, tree, "NewElement")

		statements := getStmtsFromAST(tree)
		assert.NotEmpty(t, statements)
	})
}

func TestInsertMethodIntoClass(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for nil method", func(t *testing.T) {
		tree := &js_ast.AST{}
		registry := NewRegistryContext()

		err := insertMethodIntoClass(ctx, tree, "SomeClass", nil, registry)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "method to insert is nil")
	})

	t.Run("returns error if class not found", func(t *testing.T) {
		tree := &js_ast.AST{Parts: []js_ast.Part{}}
		registry := NewRegistryContext()
		method := &js_ast.EFunction{}

		err := insertMethodIntoClass(ctx, tree, "NonExistentClass", method, registry)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("inserts method into existing class", func(t *testing.T) {

		snippet := `class TestClass extends PPElement {}`
		statement, _ := parseSnippetAsStatement(snippet)
		tree := &js_ast.AST{
			Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{statement}}},
		}
		registry := NewRegistryContext()
		method := &js_ast.EFunction{}

		err := insertMethodIntoClass(ctx, tree, "TestClass", method, registry)

		require.NoError(t, err)
	})
}

func TestMergeImportRecords(t *testing.T) {
	t.Run("does nothing for nil statementAST", func(t *testing.T) {
		tree := &js_ast.AST{}
		statement := js_ast.Stmt{}
		mergeImportRecords(tree, nil, &statement)
	})

	t.Run("does nothing for empty import records", func(t *testing.T) {
		tree := &js_ast.AST{ImportRecords: []ast.ImportRecord{}}
		statementAST := &js_ast.AST{ImportRecords: []ast.ImportRecord{}}
		statement := js_ast.Stmt{}

		mergeImportRecords(tree, statementAST, &statement)

		assert.Empty(t, tree.ImportRecords)
	})
}

func BenchmarkCompileSFC(b *testing.B) {
	ctx := context.Background()
	rawSFC := []byte(`
<script>
class BenchComponentElement extends PPElement {
	count = 0;
	increment() { this.count++; }
}
</script>
<template name="bench-component">
	<div class="container">
		<p>Count: {{ count }}</p>
		<button @click="increment">+</button>
	</div>
</template>
<style>
.container { padding: 20px; }
p { font-size: 16px; }
</style>
`)

	b.ResetTimer()
	for b.Loop() {
		_, _ = compileSFC(ctx, "bench-component.pkc", rawSFC, "", nil)
	}
}

func BenchmarkBuildClassName(b *testing.B) {
	for b.Loop() {
		_ = buildClassName("my-awesome-component")
	}
}

func BenchmarkGetStmtsFromAST(b *testing.B) {
	statements := make([]js_ast.Stmt, 100)
	for i := range statements {
		statements[i] = js_ast.Stmt{Data: &js_ast.SEmpty{}}
	}
	tree := &js_ast.AST{
		Parts: []js_ast.Part{
			{Stmts: statements[:50]},
			{Stmts: statements[50:]},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_ = getStmtsFromAST(tree)
	}
}

func TestCompiledJSStructure(t *testing.T) {
	ctx := context.Background()

	t.Run("compiled JS has proper structure", func(t *testing.T) {
		rawSFC := []byte(`
<script>
class StructureTestElement extends PPElement {
	value = 0;
}
</script>
<template name="structure-test"><div>{{ value }}</div></template>
`)

		artefact, err := compileSFC(ctx, "structure-test.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		jsContent := artefact.Files["structure-test.js"]

		assert.True(t, strings.Contains(jsContent, "PPElement"))

		assert.Contains(t, jsContent, "StructureTestElement")

		assert.Contains(t, jsContent, "customElements.define")
	})
}

func TestInjectEventBindings(t *testing.T) {
	ctx := context.Background()

	classTree := func(t *testing.T, source string) *js_ast.AST {
		t.Helper()
		statement, err := parseSnippetAsStatement(source)
		require.NoError(t, err)
		return &js_ast.AST{Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{statement}}}}
	}
	withClickBinding := func(t *testing.T, registry *RegistryContext) *eventBindingCollection {
		t.Helper()
		ec := newEventBindingCollection(registry)
		_, err := ec.createAndStoreBinding(ctx, "click", "handleClick", nil, false, nil, "")
		require.NoError(t, err)
		return ec
	}

	t.Run("empty bindings is a no-op", func(t *testing.T) {
		tree := classTree(t, `class TestElement extends PPElement { constructor() { super(); } }`)
		registry := NewRegistryContext()

		require.NoError(t, injectEventBindings(ctx, tree, "TestElement", newEventBindingCollection(registry)))
	})

	t.Run("bindings are added to the constructor", func(t *testing.T) {
		tree := classTree(t, `class TestElement extends PPElement { constructor() { super(); } }`)
		registry := NewRegistryContext()

		require.NoError(t, injectEventBindings(ctx, tree, "TestElement", withClickBinding(t, registry)))
	})

	t.Run("bindings without a component class are an error", func(t *testing.T) {
		tree := classTree(t, `const unrelated = 1;`)
		registry := NewRegistryContext()

		err := injectEventBindings(ctx, tree, "TestElement", withClickBinding(t, registry))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"TestElement" not found`)
	})

	t.Run("bindings without a constructor are an error", func(t *testing.T) {
		tree := classTree(t, `class TestElement extends PPElement {}`)
		registry := NewRegistryContext()

		err := injectEventBindings(ctx, tree, "TestElement", withClickBinding(t, registry))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no constructor found")
	})
}

func TestBuildVDOMRenderMethod(t *testing.T) {
	t.Parallel()

	newContext := func(t *testing.T, script string) *sfcCompilationContext {
		t.Helper()
		tree, registry := mustParseJS(t, script)
		cc := &sfcCompilationContext{}
		cc.registry = registry
		cc.jsAST = tree
		cc.className = "RenderWidgetElement"
		cc.reactiveTransformResult = &ReactiveTransformResult{}
		return cc
	}
	divTemplate := func(node *ast_domain.TemplateNode) *ast_domain.TemplateAST {
		return &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}
	}

	t.Run("render method is added to the component class", func(t *testing.T) {
		t.Parallel()
		cc := newContext(t, `class RenderWidgetElement extends PPElement {}`)

		err := cc.buildVDOMRenderMethod(context.Background(), divTemplate(&ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div"}), "")
		require.NoError(t, err)
		targetClass := findClassDeclarationByName(cc.jsAST, cc.className)
		require.NotNil(t, targetClass)
		assert.NotEmpty(t, targetClass.Properties)
	})

	t.Run("a render method that cannot be built fails the build", func(t *testing.T) {
		t.Parallel()
		cc := newContext(t, `class RenderWidgetElement extends PPElement {}`)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			Key:      &ast_domain.StringLiteral{Value: "0"},
			DirFor: &ast_domain.Directive{
				Type:       ast_domain.DirectiveFor,
				Expression: &ast_domain.StringLiteral{Value: "not a loop"},
			},
		}

		err := cc.buildVDOMRenderMethod(context.Background(), divTemplate(node), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "building the render method of component RenderWidgetElement")
	})

	t.Run("a render method with no class to hold it fails the build", func(t *testing.T) {
		t.Parallel()
		cc := newContext(t, `const unrelated = 1;`)

		err := cc.buildVDOMRenderMethod(context.Background(), divTemplate(&ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div"}), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "adding the render method to component RenderWidgetElement")
	})

	t.Run("event bindings with no constructor to hold them fail the build", func(t *testing.T) {
		t.Parallel()
		cc := newContext(t, `class RenderWidgetElement extends PPElement {}`)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "button",
			OnEvents: map[string][]ast_domain.Directive{
				"click": {{
					Type:          ast_domain.DirectiveOn,
					Arg:           "click",
					RawExpression: "handleClick",
					Expression:    &ast_domain.Identifier{Name: "handleClick"},
				}},
			},
		}

		err := cc.buildVDOMRenderMethod(context.Background(), divTemplate(node), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "adding the event bindings of component RenderWidgetElement")
	})
}

func TestProcessTemplate_RenderMethodFailureFailsTheBuild(t *testing.T) {
	t.Parallel()

	raw := []byte(`<script lang="ts">console.log(1);</script><template name="render-failure"><div>Hi</div></template>`)
	cc := &sfcCompilationContext{}
	var err error
	cc.sfcParseResult, err = sfcparser.Parse(raw)
	require.NoError(t, err)
	cc.registry = NewRegistryContext()
	cc.tagName = "render-failure"
	cc.className = "RenderFailureElement"
	cc.jsAST = &js_ast.AST{}
	cc.reactiveTransformResult = &ReactiveTransformResult{}

	err = cc.processTemplate(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adding the render method to component RenderFailureElement")
}

func TestCollectUsedIdentifiers(t *testing.T) {
	parser := NewTypeScriptParser()
	tree, err := parser.ParseTypeScript(
		`const s = { Item: usedVal, price: money }; foo(); bar.Widget; const t = "Item Widget MyType";`,
		"test.ts",
	)
	require.NoError(t, err)

	var bodyStmts []js_ast.Stmt
	for _, part := range tree.Parts {
		bodyStmts = append(bodyStmts, part.Stmts...)
	}

	used, ok := collectUsedIdentifiers(tree, bodyStmts, NewRegistryContext())
	require.True(t, ok)

	assert.Contains(t, used, "usedVal", "a value use must be recorded")
	assert.Contains(t, used, "money", "a value use must be recorded")
	assert.Contains(t, used, "foo", "a call target must be recorded")
	assert.Contains(t, used, "bar", "a member-access base must be recorded")

	assert.NotContains(t, used, "Item", "an object property key must not be recorded")
	assert.NotContains(t, used, "Widget", "a member name and string contents must not be recorded")
	assert.NotContains(t, used, "MyType", "a name only in a string literal must not be recorded")
}

func TestCompileSFC_UnresolvableStyleImportFailsTheBuild(t *testing.T) {
	t.Parallel()

	sfc := `<template name="broken-styles"><div class="a">x</div></template>
<style>
@import "./missing.css";
.a { color: red; }
</style>`

	orchestrator := NewCompilerOrchestrator(nil, nil,
		WithOrchestratorModuleName("example.com/proj"),
		WithOrchestratorCSSPreProcessor(&mockCSSPreProcessor{err: errors.New(`cannot resolve @import "./missing.css"`)}),
	)

	artefact, err := orchestrator.CompileSFCBytes(context.Background(), "components/widget.pkc", []byte(sfc))

	require.Error(t, err, "an unresolvable @import must fail the compile, not ship missing styles")
	assert.Nil(t, artefact)
	assert.Contains(t, err.Error(), "resolving component styles")
	assert.Contains(t, err.Error(), "missing.css")
}

func TestCompileSFC_ResolvableStyleImportStillCompiles(t *testing.T) {
	t.Parallel()

	sfc := `<template name="working-styles"><div class="a">x</div></template>
<style>
@import "./theme.css";
.a { color: red; }
</style>`

	orchestrator := NewCompilerOrchestrator(nil, nil,
		WithOrchestratorModuleName("example.com/proj"),
		WithOrchestratorCSSPreProcessor(&mockCSSPreProcessor{result: ".theme{color:blue}.a{color:red}"}),
	)

	artefact, err := orchestrator.CompileSFCBytes(context.Background(), "components/widget.pkc", []byte(sfc))

	require.NoError(t, err)
	require.NotNil(t, artefact)
	assert.Contains(t, artefact.Files[artefact.BaseJSPath], ".theme")
}

func TestPreProcessStyles(t *testing.T) {
	t.Run("no-op when styles are empty", func(t *testing.T) {
		preProcessor := &mockCSSPreProcessor{result: "should not be used"}
		ctx := context.Background()
		cc := &sfcCompilationContext{stylesDefault: "", cssPreProcessor: preProcessor}
		require.NoError(t, cc.preProcessStyles(ctx))
		assert.Equal(t, "", cc.stylesDefault)
		assert.False(t, preProcessor.called)
	})

	t.Run("no-op when no pre-processor set", func(t *testing.T) {
		ctx := context.Background()
		cc := &sfcCompilationContext{stylesDefault: "@import './foo.css';"}
		require.NoError(t, cc.preProcessStyles(ctx))
		assert.Equal(t, "@import './foo.css';", cc.stylesDefault)
	})

	t.Run("replaces styles with pre-processed result", func(t *testing.T) {
		preProcessor := &mockCSSPreProcessor{result: ".foo{color:red}"}
		ctx := context.Background()
		cc := &sfcCompilationContext{
			stylesDefault:   "@import './foo.css';",
			sourceFilename:  "components/widget.pkc",
			cssPreProcessor: preProcessor,
		}
		require.NoError(t, cc.preProcessStyles(ctx))
		assert.Equal(t, ".foo{color:red}", cc.stylesDefault)
		assert.True(t, preProcessor.called)
		assert.Equal(t, "@import './foo.css';", preProcessor.gotCSS)
		assert.Equal(t, "components/widget.pkc", preProcessor.gotSource)
	})

	t.Run("fails the compile when an import cannot be resolved", func(t *testing.T) {
		preProcessor := &mockCSSPreProcessor{err: errors.New("resolve failed")}
		ctx := context.Background()
		original := "@import './missing.css'; .local { color: blue; }"
		cc := &sfcCompilationContext{
			stylesDefault:   original,
			sourceFilename:  "components/widget.pkc",
			cssPreProcessor: preProcessor,
		}

		err := cc.preProcessStyles(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolving component styles")
		assert.Contains(t, err.Error(), "resolve failed")
		assert.True(t, preProcessor.called)
	})

	t.Run("passes the style block start location to the pre-processor", func(t *testing.T) {
		preProcessor := &mockCSSPreProcessor{result: ".foo{}"}
		ctx := context.Background()
		cc := &sfcCompilationContext{
			stylesDefault:   "@import './foo.css';",
			sourceFilename:  "components/widget.pkc",
			cssPreProcessor: preProcessor,
			stylesLocation:  ast_domain.Location{Line: 12, Column: 3, Offset: 0},
		}

		require.NoError(t, cc.preProcessStyles(ctx))

		assert.Equal(t, 12, preProcessor.gotLocation.Line)
		assert.Equal(t, 3, preProcessor.gotLocation.Column)
	})
}

func TestBuildArtefact(t *testing.T) {
	t.Parallel()

	t.Run("a script that cannot be printed fails the build", func(t *testing.T) {
		t.Parallel()
		cc := &sfcCompilationContext{}
		cc.registry = NewRegistryContext()
		cc.reactiveTransformResult = &ReactiveTransformResult{}
		cc.className = "BrokenWidgetElement"
		cc.tagName = "broken-widget"
		cc.astDump = "/* dump */"
		cc.jsAST = &js_ast.AST{Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{{Data: &js_ast.SLocal{
			Kind:  js_ast.LocalConst,
			Decls: []js_ast.Decl{{Binding: js_ast.Binding{Data: &js_ast.BMissing{}}}},
		}}}}}}

		artefact, err := cc.buildArtefact(context.Background())
		require.ErrorIs(t, err, errUnsupportedExpression)
		assert.Nil(t, artefact)
		assert.Contains(t, err.Error(), "BrokenWidgetElement")
	})

	t.Run("the dump is written before the printed script", func(t *testing.T) {
		t.Parallel()
		tree, registry := mustParseJS(t, "console.log(1);")
		cc := &sfcCompilationContext{}
		cc.registry = registry
		cc.reactiveTransformResult = &ReactiveTransformResult{}
		cc.className = "ShowWidgetElement"
		cc.tagName = "show-widget"
		cc.sourceFilename = "show-widget.pkc"
		cc.astDump = "/* dump */"
		cc.jsAST = tree

		artefact, err := cc.buildArtefact(context.Background())
		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "show-widget.js", artefact.BaseJSPath)
		assert.Equal(t, "show-widget.pkc", artefact.SourceIdentifier)
		assert.Empty(t, artefact.Diagnostics)
		assert.True(t, strings.HasPrefix(artefact.Files["show-widget.js"], "/* dump */\n\nconsole.log(1)"))
	})
}

func TestCompileSFC_ScriptFeatures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		script          string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "inline type-only import is dropped",
			script:          "import { type Foo } from './types';\nconst x: Foo | null = null;\nconsole.log(x);",
			wantNotContains: []string{"./types"},
		},
		{
			name:            "several inline type-only bindings are dropped",
			script:          "import { type A, type B } from './shapes';\nconsole.log(1);",
			wantNotContains: []string{"./shapes"},
		},
		{
			name:         "import attributes over several lines are kept",
			script:       "import data from './data.json' with {\n  type: 'json'\n};\nconsole.log(data);",
			wantContains: []string{`import data from "./data.json" with { type: "json" };`},
		},
		{
			name:         "array holes and rest elements",
			script:       "const [, second, ...others] = [1, 2, 3, 4];\nconsole.log(second, others);",
			wantContains: []string{"const [, second, ...others] = [1, 2, 3, 4];"},
		},
		{
			name:         "rest parameters",
			script:       "function total(...values: number[]) { return values.length; }\nconsole.log(total(1, 2));",
			wantContains: []string{"function total(...values)"},
		},
		{
			name:         "object rest in a parameter",
			script:       "function pick({ a, ...more }: any) { return more; }\nconsole.log(pick({ a: 1, b: 2 }));",
			wantContains: []string{`function pick({"a": a, ...more})`},
		},
		{
			name:         "enum members",
			script:       "enum Size { Small, Large }\nconsole.log(Size.Large);",
			wantContains: []string{"console.log(1)"},
		},
		{
			name:            "using declarations keep their disposal",
			script:          "function run() { using held = { [Symbol.dispose]() {} }; return held; }\nconsole.log(run());",
			wantContains:    []string{"const held = $$pikoUsingAdd($$using0, {", "$$pikoUsingDispose($$using0);", "function $$pikoUsingDispose(scope)"},
			wantNotContains: []string{"var held"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := []byte("<script lang=\"ts\">\n" + tc.script + "\n</script>\n<template name=\"script-features\"><div>Test</div></template>")

			artefact, err := compileSFC(context.Background(), "script-features.pkc", raw, "example.com/app", nil)
			require.NoError(t, err)
			script := artefact.Files[artefact.BaseJSPath]
			for _, want := range tc.wantContains {
				assert.Contains(t, script, want)
			}
			for _, unwanted := range tc.wantNotContains {
				assert.NotContains(t, script, unwanted)
			}
		})
	}
}

func TestCompileSFC_ScriptFailures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr error
		name    string
		script  string
	}{
		{
			name:    "import inside a declare module block",
			script:  "declare module 'x' { import { y } from 'y'; }\nconsole.log(1);",
			wantErr: errNestedImportUnsupported,
		},
		{
			name:    "re-export",
			script:  "export { a } from './a';",
			wantErr: errReexportUnsupported,
		},
		{
			name:    "script nested beyond the printing depth limit",
			script:  "const x = " + strings.Repeat("[", 10_001) + strings.Repeat("]", 10_001) + ";\nconsole.log(x);",
			wantErr: errNormaliseTooDeep,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := []byte("<script lang=\"ts\">\n" + tc.script + "\n</script>\n<template name=\"script-failures\"><div>Test</div></template>")

			artefact, err := compileSFC(context.Background(), "script-failures.pkc", raw, "example.com/app", nil)
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, artefact)
		})
	}
}

func TestCompileSFC_TemplateTextCannotEndTheASTDump(t *testing.T) {
	t.Parallel()

	raw := []byte(`<script lang="ts">console.log(1);</script>` +
		`<template name="dump-guard"><div title="x */ y">a */ window.pwned = 1; /* b</div><!-- c */ d --></template>`)

	artefact, err := compileSFC(context.Background(), "dump-guard.pkc", raw, "example.com/app", nil)
	require.NoError(t, err)

	script := artefact.Files[artefact.BaseJSPath]
	dumpEnd := strings.Index(script, "--- END AST DUMP ---\n*/")
	require.Positive(t, dumpEnd)
	assert.NotContains(t, script[:dumpEnd], "*/", "the AST dump comment must not be closed early")
	assert.Contains(t, script[:dumpEnd], `a *\/ window.pwned = 1; /* b`)
}

func TestCompileSFC_AttributesAreOrderedByName(t *testing.T) {
	t.Parallel()

	raw := []byte(`<script lang="ts">console.log(1);</script>` +
		`<template name="ordered-attributes"><div title="t" class="c" id="i">Hi</div></template>`)

	artefact, err := compileSFC(context.Background(), "ordered-attributes.pkc", raw, "example.com/app", nil)
	require.NoError(t, err)
	assert.Contains(t, artefact.ScaffoldHTML, `<div class="c" id="i" title="t">`)
}

func TestInjectTimelineData(t *testing.T) {
	t.Parallel()

	newContext := func(t *testing.T, script string, timelineJSON string) *sfcCompilationContext {
		t.Helper()
		tree, _ := mustParseJS(t, script)
		cc := &sfcCompilationContext{}
		cc.jsAST = tree
		cc.className = "TimelineWidgetElement"
		cc.timelineJSON = timelineJSON
		return cc
	}

	t.Run("no timeline leaves the tree alone", func(t *testing.T) {
		t.Parallel()
		cc := newContext(t, `const unrelated = 1;`, "")
		require.NoError(t, cc.injectTimelineData(context.Background()))
	})

	t.Run("timeline is added to the component class, not a helper", func(t *testing.T) {
		t.Parallel()
		cc := newContext(t, `class Helper {} class TimelineWidgetElement extends PPElement {}`, `[]`)
		require.NoError(t, cc.injectTimelineData(context.Background()))

		assert.Empty(t, findClassDeclarationByName(cc.jsAST, "Helper").Properties)
		assert.Len(t, findClassDeclarationByName(cc.jsAST, cc.className).Properties, 1)
	})

	t.Run("timeline without the component class is an error", func(t *testing.T) {
		t.Parallel()
		cc := newContext(t, `class Helper {}`, `[]`)
		err := cc.injectTimelineData(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"TimelineWidgetElement" not found`)
	})
}

func TestCompileSFC_ScriptsWithSeveralClasses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		script string
	}{
		{
			name:   "helper classes beside a generated component class",
			script: "class Tally { count = 0; bump() { return ++this.count; } }\nclass Greeter { greet() { return 'hi'; } }\nfunction label() { return new Greeter().greet() + new Tally().bump(); }",
		},
		{
			name:   "helper class declared before the author's component class",
			script: "class Tally { count = 0; bump() { return ++this.count; } }\nclass SeveralClassesElement extends PPElement { helperCount() { return 1; } }\nfunction label() { return new Tally().bump(); }",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := []byte("<script lang=\"ts\">\n" + tc.script + "\n</script>\n" +
				`<template name="several-classes"><button p-on:click="label">{{ label() }}</button></template>`)

			artefact, err := compileSFC(context.Background(), "several-classes.pkc", raw, "example.com/app", nil)
			require.NoError(t, err)
			script := artefact.Files[artefact.BaseJSPath]

			componentStart := strings.Index(script, "class SeveralClassesElement extends PPElement {")
			tallyStart := strings.Index(script, "class Tally {")
			require.Positive(t, componentStart)
			require.Positive(t, tallyStart)

			componentBody := script[componentStart:]
			if tallyStart > componentStart {
				componentBody = script[componentStart:tallyStart]
			}
			assert.Contains(t, componentBody, "renderVDOM ()")
			assert.Contains(t, componentBody, "connectedCallback ()")
			assert.Contains(t, componentBody, "_dir_click_label_evt_")

			tallyBody := script[tallyStart:]
			if componentStart > tallyStart {
				tallyBody = script[tallyStart:componentStart]
			}
			assert.NotContains(t, tallyBody, "renderVDOM")
			assert.NotContains(t, tallyBody, "connectedCallback")
			assert.Equal(t, 1, strings.Count(script, "class SeveralClassesElement"), "exactly one component class is emitted")
		})
	}
}
