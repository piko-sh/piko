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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
	es_logger "piko.sh/piko/internal/esbuild/logger"
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
		result := printAST(context.Background(), nil, nil, nil)
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

		result := printAST(context.Background(), tree, nil, registry)

		assert.NotNil(t, result)
	})
}

func TestPrintTdewolffAST(t *testing.T) {
	t.Run("returns empty for nil AST", func(t *testing.T) {
		result := printTdewolffAST(nil)
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

func TestFindImportKeyword(t *testing.T) {
	testCases := []struct {
		name       string
		source     string
		pathStart  int
		wantResult int
	}{
		{
			name:       "finds import before path",
			source:     `import { foo } from './foo';`,
			pathStart:  21,
			wantResult: 0,
		},
		{
			name:       "returns -1 when no import found",
			source:     `const x = require('./foo');`,
			pathStart:  20,
			wantResult: -1,
		},
		{
			name:       "ignores import inside identifier",
			source:     `const reimport = 1; import { x } from './x';`,
			pathStart:  40,
			wantResult: 20,
		},
		{
			name:       "finds import at start of string",
			source:     `import './side-effect';`,
			pathStart:  8,
			wantResult: 0,
		},
		{
			name:       "returns -1 for empty source",
			source:     ``,
			pathStart:  0,
			wantResult: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findImportKeyword(tc.source, tc.pathStart)
			assert.Equal(t, tc.wantResult, result)
		})
	}
}

func TestFindStatementEnd(t *testing.T) {
	testCases := []struct {
		name       string
		source     string
		pathEnd    int
		wantResult int
	}{
		{
			name:       "finds semicolon",
			source:     `import './foo';`,
			pathEnd:    13,
			wantResult: 15,
		},
		{
			name:       "finds newline after quote",
			source:     "import './foo'\nconst x = 1;",
			pathEnd:    13,
			wantResult: 14,
		},
		{
			name:       "reaches end of string",
			source:     `import './foo'`,
			pathEnd:    13,
			wantResult: 14,
		},
		{
			name:       "pathEnd at end of source",
			source:     `import './foo';`,
			pathEnd:    15,
			wantResult: 15,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findStatementEnd(tc.source, tc.pathEnd)
			assert.Equal(t, tc.wantResult, result)
		})
	}
}

func TestIsIdentifierChar(t *testing.T) {
	testCases := []struct {
		name   string
		c      byte
		expect bool
	}{
		{name: "lowercase letter", c: 'a', expect: true},
		{name: "uppercase letter", c: 'Z', expect: true},
		{name: "digit", c: '5', expect: true},
		{name: "underscore", c: '_', expect: true},
		{name: "dollar sign", c: '$', expect: true},
		{name: "space", c: ' ', expect: false},
		{name: "semicolon", c: ';', expect: false},
		{name: "dot", c: '.', expect: false},
		{name: "quote", c: '\'', expect: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, isIdentifierChar(tc.c))
		})
	}
}

func TestExtractImportTextFromSource(t *testing.T) {
	t.Run("returns empty for empty source", func(t *testing.T) {
		result := extractImportTextFromSource("", ast.ImportRecord{})
		assert.Empty(t, result)
	})

	t.Run("returns empty for out-of-bounds range", func(t *testing.T) {
		result := extractImportTextFromSource("short", ast.ImportRecord{
			Range: es_logger.Range{Loc: es_logger.Loc{Start: 100}, Len: 5},
		})
		assert.Empty(t, result)
	})

	t.Run("returns empty when no import keyword found", func(t *testing.T) {
		src := `const x = require('./foo');`
		result := extractImportTextFromSource(src, ast.ImportRecord{
			Range: es_logger.Range{Loc: es_logger.Loc{Start: 19}, Len: 7},
		})
		assert.Empty(t, result)
	})
}

func TestInjectEventBindings(t *testing.T) {
	ctx := context.Background()

	t.Run("empty bindings is a no-op", func(t *testing.T) {
		statement, _ := parseSnippetAsStatement(`class TestElement extends PPElement { constructor() { super(); } }`)
		tree := &js_ast.AST{
			Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{statement}}},
		}
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)

		injectEventBindings(ctx, tree, "TestElement", ec)

	})
}
