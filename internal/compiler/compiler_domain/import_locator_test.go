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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/logger"
)

func paddedImport(t *testing.T, distance int, path string) string {
	t.Helper()
	head := "import { one, pad as p"
	tail := " } from "
	padding := distance - len(head) - len(tail)
	require.Positive(t, padding, "distance %d is too short to pad", distance)
	return head + strings.Repeat("_", padding) + tail + "'" + path + "';"
}

func recoveredImportTexts(t *testing.T, source string) ([]string, error) {
	t.Helper()
	tree, err := NewTypeScriptParser().ParseTypeScriptStrict(source, "component.ts")
	require.NoError(t, err)
	locator, err := locateImportStatements(source)
	require.NoError(t, err)

	var texts []string
	for _, record := range tree.ImportRecords {
		if record.Kind != ast.ImportStmt {
			continue
		}
		text, _, err := extractImportTextFromSource(source, locator, record)
		if err != nil {
			return texts, err
		}
		texts = append(texts, text)
	}
	return texts, nil
}

func withoutSemicolon(statement string) string {
	return strings.TrimSuffix(statement, ";")
}

func TestLocateImportStatements(t *testing.T) {
	t.Parallel()

	source := strings.Join([]string{
		`import { a } from './a';`,
		`import b from './b';`,
		`import * as c from './c';`,
		`import './d';`,
		"import {\n  e,\n  f,\n} from './e';",
		`import { type T, g } from './g';`,
		`import { unused } from './unused';`,
		`export { h } from './h';`,
		`export * from './i';`,
		`console.log(a, b, c, e, f, g);`,
	}, "\n")

	locator, err := locateImportStatements(source)
	require.NoError(t, err)

	for _, path := range []string{"./a", "./b", "./c", "./d", "./e", "./g", "./unused"} {
		quoted := "'" + path + "'"
		pathStart := strings.Index(source, quoted)
		span, err := locator.statementSpan(int32(pathStart))
		require.NoError(t, err, "import of %s not located", path)
		assert.Equal(t, strings.LastIndex(source[:pathStart], "\nimport")+1, span.start, "import of %s located at the wrong statement", path)
		assert.Equal(t, pathStart+len(quoted), span.end, "import of %s ends at the wrong offset", path)
	}

	for _, path := range []string{"./h", "./i"} {
		_, err := locator.statementSpan(int32(strings.Index(source, "'"+path+"'")))
		assert.ErrorIs(t, err, errReexportUnsupported, "re-export of %s must not be located as an import", path)
	}
}

func TestLocateImportStatements_AttributesClause(t *testing.T) {
	t.Parallel()

	statement := "import data from './data.json' with {\n  type: 'json'\n}"
	source := statement + ";\nconsole.log(data);"

	locator, err := locateImportStatements(source)
	require.NoError(t, err)

	span, err := locator.statementSpan(int32(strings.Index(source, "'./data.json'")))
	require.NoError(t, err)
	assert.Equal(t, 0, span.start)
	assert.Equal(t, len(statement), span.end, "the span must reach the closing brace of the attributes clause")
}

func TestLocateImportStatements_EmptyAndInvalidScripts(t *testing.T) {
	t.Parallel()

	t.Run("empty script locates nothing", func(t *testing.T) {
		t.Parallel()
		locator, err := locateImportStatements("")
		require.NoError(t, err)
		_, err = locator.statementSpan(0)
		assert.ErrorIs(t, err, errNestedImportUnsupported)
	})

	t.Run("unparsable script is an error", func(t *testing.T) {
		t.Parallel()
		_, err := locateImportStatements("import { from")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "locating import statements")
	})
}

func TestImportStatementLocator_AddStatementRejectsMissingRecords(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		statement js_ast.Stmt
		name      string
	}{
		{name: "import", statement: js_ast.Stmt{Data: &js_ast.SImport{ImportRecordIndex: 2}}},
		{name: "export from", statement: js_ast.Stmt{Data: &js_ast.SExportFrom{ImportRecordIndex: 2}}},
		{name: "export star", statement: js_ast.Stmt{Data: &js_ast.SExportStar{ImportRecordIndex: 2}}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			locator := importStatementLocator{
				imports:   make(map[int32]importStatementSpan),
				reexports: make(map[int32]struct{}),
			}
			err := locator.addStatement([]ast.ImportRecord{{}}, tc.statement)
			require.ErrorIs(t, err, errImportNotRecovered)
			assert.Contains(t, err.Error(), "import record 2 of 1")
		})
	}
}

func TestImportStatementEnd(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		record ast.ImportRecord
		want   int
	}{
		{
			name:   "path only",
			record: ast.ImportRecord{Range: logger.Range{Loc: logger.Loc{Start: 18}, Len: 5}},
			want:   23,
		},
		{
			name: "attributes clause after the path",
			record: ast.ImportRecord{
				Range:        logger.Range{Loc: logger.Loc{Start: 18}, Len: 5},
				AssertOrWith: &ast.ImportAssertOrWith{InnerCloseBraceLoc: logger.Loc{Start: 40}},
			},
			want: 41,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, importStatementEnd(tc.record))
		})
	}
}

func TestExtractImportTextFromSource_ByteDistances(t *testing.T) {
	t.Parallel()

	first := `import { alpha, beta } from '@/lib/first';`
	for _, distance := range []int{195, 200, 201, 203, 205, 206, 210, 401, 403, 405} {
		t.Run(fmt.Sprintf("distance %d", distance), func(t *testing.T) {
			t.Parallel()
			second := paddedImport(t, distance, "@/lib/second")
			texts, err := recoveredImportTexts(t, first+"\n"+second+"\nconsole.log(alpha, beta, one, p);")
			require.NoError(t, err)
			assert.Equal(t, []string{withoutSemicolon(first), withoutSemicolon(second)}, texts)
		})
	}

	t.Run("lone import at 203 bytes", func(t *testing.T) {
		t.Parallel()
		only := paddedImport(t, 203, "@/lib/second")
		texts, err := recoveredImportTexts(t, only+"\nconsole.log(one);")
		require.NoError(t, err)
		assert.Equal(t, []string{withoutSemicolon(only)}, texts)
	})
}

func TestExtractImportTextFromSource_ImportInsideClause(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		statement string
	}{
		{name: "binding name starting with import", statement: `import { importedName } from '@/lib/third';`},
		{name: "default binding starting with import", statement: `import importX from '@/lib/third';`},
		{name: "aliased binding starting with import", statement: `import { importScene as scene } from '@/lib/third';`},
		{name: "keyword inside a comment in the clause", statement: `import { a, /* import */ b } from '@/lib/third';`},
		{name: "keyword inside a line comment in the clause", statement: "import {\n  a, // import\n  b,\n} from '@/lib/third';"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := `import { x } from './x';` + "\n" + tc.statement + "\nconsole.log(x);"
			texts, err := recoveredImportTexts(t, source)
			require.NoError(t, err)
			assert.Equal(t, []string{`import { x } from './x'`, withoutSemicolon(tc.statement)}, texts)
		})
	}
}

func TestExtractImportTextFromSource_StatementShapes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "attributes clause spanning several lines",
			source: "import data from './a.json' with {\n  type: 'json'\n};\nconsole.log(data);",
			want:   []string{"import data from './a.json' with {\n  type: 'json'\n}"},
		},
		{
			name:   "attributes clause on one line",
			source: "import data from './a.json' with { type: 'json' };\nconsole.log(data);",
			want:   []string{"import data from './a.json' with { type: 'json' }"},
		},
		{
			name:   "assert keyword",
			source: "import data from './a.json' assert { type: 'json' };\nconsole.log(data);",
			want:   []string{"import data from './a.json' assert { type: 'json' }"},
		},
		{
			name:   "no semicolon",
			source: "import { a } from './a'\nconsole.log(a)",
			want:   []string{"import { a } from './a'"},
		},
		{
			name:   "trailing comment holding a semicolon",
			source: "import { a } from './a' // trailing; comment\nconsole.log(a);",
			want:   []string{"import { a } from './a'"},
		},
		{
			name:   "two statements on one line",
			source: "import { a } from './a'; import { b } from './b';\nconsole.log(a, b);",
			want:   []string{"import { a } from './a'", "import { b } from './b'"},
		},
		{
			name:   "carriage return line endings",
			source: "import { a } from './a';\r\nimport { b } from './b';\r\nconsole.log(a, b);",
			want:   []string{"import { a } from './a'", "import { b } from './b'"},
		},
		{
			name:   "import after non-ASCII text",
			source: "const s = 'h\u00e9llo \u2713';\nimport { a } from './a';\nconsole.log(s, a);",
			want:   []string{"import { a } from './a'"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			texts, err := recoveredImportTexts(t, tc.source)
			require.NoError(t, err)
			assert.Equal(t, tc.want, texts)
		})
	}
}

func TestExtractImportTextFromSource_Errors(t *testing.T) {
	t.Parallel()

	t.Run("re-export is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := recoveredImportTexts(t, `import { a } from './a';`+"\n"+`export { b } from './b';`)
		require.ErrorIs(t, err, errReexportUnsupported)
		assert.Contains(t, err.Error(), `'./b'`)
	})

	t.Run("import inside a declare module block is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := recoveredImportTexts(t, "declare module 'x' { import { y } from 'y'; }\nconsole.log(1);")
		require.ErrorIs(t, err, errNestedImportUnsupported)
		assert.NotErrorIs(t, err, errReexportUnsupported)
		assert.Contains(t, err.Error(), `'y'`)
		assert.Contains(t, err.Error(), "declare module")
	})

	t.Run("path range outside the script is rejected", func(t *testing.T) {
		t.Parallel()
		record := ast.ImportRecord{Kind: ast.ImportStmt, Range: logger.Range{Loc: logger.Loc{Start: 100}, Len: 5}}
		_, _, err := extractImportTextFromSource("short", importStatementLocator{}, record)
		require.ErrorIs(t, err, errImportNotRecovered)
	})

	t.Run("statement range outside the script is rejected", func(t *testing.T) {
		t.Parallel()
		source := `import { a } from './a';`
		pathStart := int32(strings.Index(source, "'./a'"))
		locator := importStatementLocator{
			imports:   map[int32]importStatementSpan{pathStart: {start: 0, end: len(source) + 10}},
			reexports: map[int32]struct{}{},
		}
		record := ast.ImportRecord{Kind: ast.ImportStmt, Range: logger.Range{Loc: logger.Loc{Start: pathStart}, Len: 5}}
		_, _, err := extractImportTextFromSource(source, locator, record)
		require.ErrorIs(t, err, errImportNotRecovered)
		assert.Contains(t, err.Error(), "statement range outside the script")
	})
}

func TestBuildImportStatementsFromSource(t *testing.T) {
	t.Parallel()

	t.Run("every import survives at a chunk-boundary distance", func(t *testing.T) {
		t.Parallel()
		source := `import { alpha, beta } from '@/lib/first';` + "\n" +
			paddedImport(t, 203, "@/lib/second") + "\n" +
			`import { importedName } from '@/lib/third';` + "\n" +
			`console.log(alpha);`
		tree, err := NewTypeScriptParser().ParseTypeScriptStrict(source, "component.ts")
		require.NoError(t, err)

		statements, _, err := buildImportStatementsFromSource(context.Background(), tree, source, "example.com/app")
		require.NoError(t, err)
		require.Len(t, statements, 3, "imports used only by the template must still be hoisted")
	})

	t.Run("re-export fails the build", func(t *testing.T) {
		t.Parallel()
		source := `export * from './x';`
		tree, err := NewTypeScriptParser().ParseTypeScriptStrict(source, "component.ts")
		require.NoError(t, err)

		_, _, err = buildImportStatementsFromSource(context.Background(), tree, source, "example.com/app")
		require.ErrorIs(t, err, errReexportUnsupported)
	})

	t.Run("import inside a declare module block fails the build", func(t *testing.T) {
		t.Parallel()
		source := "declare module 'x' { import { y } from 'y'; }\nconsole.log(1);"
		tree, err := NewTypeScriptParser().ParseTypeScriptStrict(source, "component.ts")
		require.NoError(t, err)

		_, _, err = buildImportStatementsFromSource(context.Background(), tree, source, "example.com/app")
		require.ErrorIs(t, err, errNestedImportUnsupported)
	})

	hoistCases := []struct {
		name      string
		source    string
		wantPaths []string
	}{
		{
			name:      "inline type-only import is dropped",
			source:    "import { type Foo } from './types';\nconsole.log(1);",
			wantPaths: []string{},
		},
		{
			name:      "several inline type-only bindings are dropped",
			source:    "import { type A, type B } from './x';\nconsole.log(1);",
			wantPaths: []string{},
		},
		{
			name:      "type-only import beside a value import keeps the value import",
			source:    "import { type Foo } from './types';\nimport { a } from './a';\nconsole.log(a);",
			wantPaths: []string{"./a"},
		},
		{
			name:      "mixed type and value bindings keep the statement",
			source:    "import { type A, b } from './x';\nconsole.log(b);",
			wantPaths: []string{"./x"},
		},
		{
			name:      "attributes clause spanning several lines is hoisted",
			source:    "import data from './a.json' with {\n  type: 'json'\n};\nconsole.log(data);",
			wantPaths: []string{"./a.json"},
		},
	}
	for _, tc := range hoistCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tree, err := NewTypeScriptParser().ParseTypeScriptStrict(tc.source, "component.ts")
			require.NoError(t, err)

			statements, _, err := buildImportStatementsFromSource(context.Background(), tree, tc.source, "example.com/app")
			require.NoError(t, err)

			paths := make([]string, 0, len(statements))
			for _, statement := range statements {
				simport, ok := statement.Data.(*js_ast.SImport)
				require.True(t, ok, "hoisted statement is %T", statement.Data)
				paths = append(paths, tree.ImportRecords[simport.ImportRecordIndex].Path.Text)
			}
			assert.Equal(t, tc.wantPaths, paths)
		})
	}

	t.Run("attributes clause is carried on the merged record", func(t *testing.T) {
		t.Parallel()
		source := "import data from './a.json' with {\n  type: 'json'\n};\nconsole.log(data);"
		tree, err := NewTypeScriptParser().ParseTypeScriptStrict(source, "component.ts")
		require.NoError(t, err)

		statements, _, err := buildImportStatementsFromSource(context.Background(), tree, source, "example.com/app")
		require.NoError(t, err)
		require.Len(t, statements, 1)

		simport, ok := statements[0].Data.(*js_ast.SImport)
		require.True(t, ok)
		attributes := tree.ImportRecords[simport.ImportRecordIndex].AssertOrWith
		require.NotNil(t, attributes)
		assert.Equal(t, ast.WithKeyword, attributes.Keyword)
		require.Len(t, attributes.Entries, 1)
	})
}

func TestRecoversSingleImport(t *testing.T) {
	t.Parallel()

	recovers := func(text string) bool {
		_, statementAST, err := parseModuleLevelStatement(context.Background(), text)
		require.NoError(t, err)
		return recoversSingleImport(text, `'./b'`, statementAST)
	}

	assert.True(t, recovers(`import { b } from './b';`))
	assert.False(t, recovers(`import { a } from './a'; import { b } from './b';`), "a fused statement must be rejected")
	assert.False(t, recovers(`import { a } from './a';`), "a statement for another path must be rejected")
}
