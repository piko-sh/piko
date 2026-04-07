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

package cssinliner

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
	es_logger "piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

type testFSReader struct {
	files map[string]string
}

func newTestFSReader() *testFSReader {
	return &testFSReader{
		files: make(map[string]string),
	}
}

func (r *testFSReader) addFile(path, content string) {
	r.files[path] = content
}

func (r *testFSReader) ReadFile(_ context.Context, path string) ([]byte, error) {
	content, exists := r.files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return []byte(content), nil
}

func newPassthroughResolver() *resolver_domain.MockResolver {
	return &resolver_domain.MockResolver{
		ResolveCSSPathFunc: func(_ context.Context, importPath, _ string) (string, error) {
			return importPath, nil
		},
	}
}

func newTestProcessor(resolver resolver_domain.ResolverPort) *Processor {
	return NewProcessor(ProcessorConfig{
		Resolver: resolver,
		Loader:   config.LoaderLocalCSS,
		Options: &config.Options{
			MinifyWhitespace: true,
			MinifySyntax:     true,
		},
	})
}

func newTestInliner(processor *Processor, fsReader FSReaderPort) *Inliner {
	return GetInliner(processor.GetResolver(), processor.GetParserOptions(), fsReader, "TEST")
}

func TestCSSInliner_BasicImport(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should inline a single @import statement", func(t *testing.T) {
		fsReader.addFile("/test/base.css", `body { margin: 0; padding: 0; }`)

		css := `@import "/test/base.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.NotEmpty(t, tree.Rules)
	})

	t.Run("should inline multiple @import statements", func(t *testing.T) {
		fsReader.addFile("/test/reset.css", `* { margin: 0; }`)
		fsReader.addFile("/test/base.css", `body { padding: 0; }`)

		css := `
			@import "/test/reset.css";
			@import "/test/base.css";
		`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.NotEmpty(t, tree.Rules)
	})

	t.Run("should preserve local CSS rules alongside imports", func(t *testing.T) {
		fsReader.addFile("/test/base.css", `body { margin: 0; }`)

		css := `
			@import "/test/base.css";
			.container { padding: 10px; }
		`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.True(t, len(tree.Rules) >= 2, "Should have at least 2 rules")
	})
}

func TestCSSInliner_ImportChain(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should resolve linear import chain A → B → C", func(t *testing.T) {
		fsReader.addFile("/test/c.css", `h3 { font-size: 1em; }`)
		fsReader.addFile("/test/b.css", `@import "/test/c.css"; h2 { font-size: 1.5em; }`)
		fsReader.addFile("/test/a.css", `@import "/test/b.css"; h1 { font-size: 2em; }`)

		css := `@import "/test/a.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.True(t, len(tree.Rules) >= 3, "Should have rules from entire import chain")
	})

	t.Run("should handle diamond dependency (A imports B and C, both import D)", func(t *testing.T) {
		fsReader.addFile("/test/d.css", `p { color: black; }`)
		fsReader.addFile("/test/b.css", `@import "/test/d.css"; .b { font-weight: bold; }`)
		fsReader.addFile("/test/c.css", `@import "/test/d.css"; .c { font-style: italic; }`)
		fsReader.addFile("/test/a.css", `@import "/test/b.css"; @import "/test/c.css";`)

		css := `@import "/test/a.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.NotEmpty(t, tree.Rules)
	})
}

func TestCSSInliner_CircularDependencies(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should detect simple circular import A → B → A", func(t *testing.T) {
		fsReader.addFile("/test/a.css", `@import "/test/b.css"; .a { color: red; }`)
		fsReader.addFile("/test/b.css", `@import "/test/a.css"; .b { color: blue; }`)

		css := `@import "/test/a.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		assert.Nil(t, tree)
		require.NotEmpty(t, diagnostics)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "circular")
	})

	t.Run("should detect longer circular import A → B → C → A", func(t *testing.T) {
		fsReader.addFile("/test/a.css", `@import "/test/b.css"; .a { color: red; }`)
		fsReader.addFile("/test/b.css", `@import "/test/c.css"; .b { color: blue; }`)
		fsReader.addFile("/test/c.css", `@import "/test/a.css"; .c { color: green; }`)

		css := `@import "/test/a.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		assert.Nil(t, tree)
		require.NotEmpty(t, diagnostics)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "circular")
	})

	t.Run("should detect self-import (A imports A)", func(t *testing.T) {
		fsReader.addFile("/test/a.css", `@import "/test/a.css"; .a { color: red; }`)

		css := `@import "/test/a.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		assert.Nil(t, tree)
		require.NotEmpty(t, diagnostics)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "circular")
	})
}

func TestCSSInliner_Caching(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should cache imported files to avoid re-parsing", func(t *testing.T) {
		fsReader.addFile("/test/d.css", `p { color: black; }`)
		fsReader.addFile("/test/b.css", `@import "/test/d.css"; .b { font-weight: bold; }`)
		fsReader.addFile("/test/c.css", `@import "/test/d.css"; .c { font-style: italic; }`)

		css := `@import "/test/b.css"; @import "/test/c.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.NotEmpty(t, inliner.cache)
		_, cached := inliner.cache["/test/d.css"]
		assert.True(t, cached, "d.css should be cached after being imported twice")
	})

	t.Run("cache should provide deep copies to prevent mutation", func(t *testing.T) {
		fsReader.addFile("/test/shared.css", `.shared { margin: 0; }`)

		css1 := `@import "/test/shared.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree1, _ := inliner.InlineAndParse(ctx, css1, "main1.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		css2 := `@import "/test/shared.css";`
		tree2, _ := inliner.InlineAndParse(ctx, css2, "main2.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree1)
		require.NotNil(t, tree2)

		assert.NotSame(t, tree1, tree2, "Trees should be different instances")
	})
}

func TestCSSInliner_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should report diagnostic for missing import file", func(t *testing.T) {
		css := `@import "/nonexistent.css"; .local { color: red; }`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		require.NotEmpty(t, diagnostics)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "not found")

		assert.NotEmpty(t, tree.Rules)
	})

	t.Run("should handle resolver errors gracefully", func(t *testing.T) {
		failingResolver := &resolver_domain.MockResolver{
			ResolveCSSPathFunc: func(_ context.Context, importPath, _ string) (string, error) {
				return "", fmt.Errorf("resolver error: cannot resolve %s", importPath)
			},
		}
		failingProcessor := newTestProcessor(failingResolver)

		css := `@import "./some.css"; .local { color: red; }`
		inliner := newTestInliner(failingProcessor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		require.NotEmpty(t, diagnostics)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "resolver error")
	})

	t.Run("should handle partial success with some missing imports", func(t *testing.T) {
		fsReader.addFile("/test/valid.css", `.valid { color: green; }`)

		css := `
			@import "/test/valid.css";
			@import "/missing1.css";
			@import "/missing2.css";
			.local { color: blue; }
		`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)

		assert.Len(t, diagnostics, 2)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Equal(t, ast_domain.Error, diagnostics[1].Severity)

		assert.NotEmpty(t, tree.Rules)
	})
}

func TestCSSInliner_SymbolReIndexing(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should correctly re-index symbols when merging ASTs", func(t *testing.T) {
		fsReader.addFile("/test/a.css", `.class-a { color: red; }`)
		fsReader.addFile("/test/b.css", `.class-b { color: blue; }`)

		css := `@import "/test/a.css"; @import "/test/b.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.NotEmpty(t, tree.Symbols, "Merged tree should have symbols")
	})

	t.Run("should handle nested imports with symbol re-indexing", func(t *testing.T) {
		fsReader.addFile("/test/base.css", `#id-base { margin: 0; }`)
		fsReader.addFile("/test/middle.css", `@import "/test/base.css"; .class-middle { padding: 0; }`)
		fsReader.addFile("/test/top.css", `@import "/test/middle.css"; .class-top { border: 0; }`)

		css := `@import "/test/top.css";`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.NotEmpty(t, tree.Symbols, "Should have symbols from all files in the chain")
	})
}

func TestCSSInliner_ASTMerging(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should preserve rule order when merging", func(t *testing.T) {
		fsReader.addFile("/test/first.css", `/* First */ .first { color: red; }`)
		fsReader.addFile("/test/second.css", `/* Second */ .second { color: blue; }`)

		css := `
			/* Before imports */
			@import "/test/first.css";
			@import "/test/second.css";
			/* After imports */
			.third { color: green; }
		`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.True(t, len(tree.Rules) >= 3, "Should have at least 3 rules")
	})

	t.Run("should merge metadata (ImportRecords, Symbols, etc.)", func(t *testing.T) {
		fsReader.addFile("/test/imported.css", `.imported { color: purple; }`)

		css := `@import "/test/imported.css"; .local { color: orange; }`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.NotEmpty(t, tree.Symbols, "Merged tree should preserve symbols")

		assert.NotNil(t, tree.ImportRecords)
	})
}

func TestCSSInliner_EdgeCases(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should handle empty imported file", func(t *testing.T) {
		fsReader.addFile("/test/empty.css", ``)

		css := `@import "/test/empty.css"; .local { color: red; }`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)

		assert.NotEmpty(t, tree.Rules)
	})

	t.Run("should handle imported file with only comments", func(t *testing.T) {
		fsReader.addFile("/test/comments.css", `/* This is just a comment */`)

		css := `@import "/test/comments.css"; .local { color: red; }`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)
	})

	t.Run("should handle CSS with no imports", func(t *testing.T) {
		css := `.local { color: red; }`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)
		assert.NotEmpty(t, tree.Rules)
	})

	t.Run("should handle whitespace-only CSS", func(t *testing.T) {
		css := `   \n\t   `
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)

		_ = diagnostics
	})
}

func TestCloneR(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil input", func(t *testing.T) {
		t.Parallel()
		result := cloneR(nil)
		assert.Nil(t, result)
	})

	t.Run("clones RAtCharset", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RAtCharset{Encoding: "UTF-8"}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RAtCharset)
		require.True(t, ok)
		assert.Equal(t, "UTF-8", cloned.Encoding)
		assert.NotSame(t, original, cloned)
	})

	t.Run("clones RAtImport without conditions", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RAtImport{
			ImportRecordIndex: 42,
			ImportConditions:  nil,
		}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RAtImport)
		require.True(t, ok)
		assert.NotSame(t, original, cloned)
		assert.Equal(t, uint32(42), cloned.ImportRecordIndex)
		assert.Nil(t, cloned.ImportConditions)
	})

	t.Run("clones RAtImport with conditions", func(t *testing.T) {
		t.Parallel()
		conditions := &css_ast.ImportConditions{
			Layers: []css_ast.Token{
				{Kind: css_lexer.TFunction, Text: "layer"},
			},
		}
		original := &css_ast.RAtImport{
			ImportRecordIndex: 7,
			ImportConditions:  conditions,
		}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RAtImport)
		require.True(t, ok)
		assert.NotSame(t, original, cloned)
		assert.Equal(t, uint32(7), cloned.ImportRecordIndex)
		require.NotNil(t, cloned.ImportConditions)
		assert.NotSame(t, original.ImportConditions, cloned.ImportConditions)
	})

	t.Run("clones RComment", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RComment{Text: "test comment"}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RComment)
		require.True(t, ok)
		assert.Equal(t, "test comment", cloned.Text)
		assert.NotSame(t, original, cloned)
	})

	t.Run("clones RComment with empty text", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RComment{Text: ""}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RComment)
		require.True(t, ok)
		assert.NotSame(t, original, cloned)
		assert.Equal(t, "", cloned.Text)
	})

	t.Run("clones RUnknownAt with prelude and block", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RUnknownAt{
			AtToken: "@custom",
			Prelude: []css_ast.Token{{Kind: css_lexer.TIdent, Text: "foo"}},
			Block:   []css_ast.Token{{Kind: css_lexer.TIdent, Text: "bar"}},
		}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RUnknownAt)
		require.True(t, ok)
		assert.NotSame(t, original, cloned)
		assert.Equal(t, "@custom", cloned.AtToken)
		require.Len(t, cloned.Prelude, 1)
		assert.Equal(t, "foo", cloned.Prelude[0].Text)
		require.Len(t, cloned.Block, 1)
		assert.Equal(t, "bar", cloned.Block[0].Text)
	})

	t.Run("clones RUnknownAt with empty prelude and block", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RUnknownAt{
			AtToken: "@empty",
		}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RUnknownAt)
		require.True(t, ok)
		assert.NotSame(t, original, cloned)
		assert.Equal(t, "@empty", cloned.AtToken)
		assert.Nil(t, cloned.Prelude)
		assert.Nil(t, cloned.Block)
	})

	t.Run("clones RDeclaration", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RDeclaration{
			KeyText: "color",
			Value:   []css_ast.Token{{Kind: css_lexer.TIdent, Text: "red"}},
		}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RDeclaration)
		require.True(t, ok)
		assert.Equal(t, "color", cloned.KeyText)
		assert.NotSame(t, original, cloned)
	})

	t.Run("clones RBadDeclaration", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RBadDeclaration{
			Tokens: []css_ast.Token{{Kind: css_lexer.TIdent, Text: "bad"}},
		}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RBadDeclaration)
		require.True(t, ok)
		require.Len(t, cloned.Tokens, 1)
		assert.NotSame(t, original, cloned)
	})

	t.Run("clones RAtLayer", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RAtLayer{
			Names: [][]string{{"base", "theme"}},
			Rules: []css_ast.Rule{},
		}
		result := cloneR(original)
		require.NotNil(t, result)
		cloned, ok := result.(*css_ast.RAtLayer)
		require.True(t, ok)
		require.Len(t, cloned.Names, 1)
		assert.Equal(t, []string{"base", "theme"}, cloned.Names[0])
		assert.NotSame(t, original, cloned)
	})

	t.Run("clones RAtMedia", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.RAtMedia{
			Rules: []css_ast.Rule{},
		}
		result := cloneR(original)
		require.NotNil(t, result)
		_, ok := result.(*css_ast.RAtMedia)
		assert.True(t, ok)
	})
}

func TestReIndexRule(t *testing.T) {
	t.Parallel()

	t.Run("re-indexes RSelector rule", func(t *testing.T) {
		t.Parallel()
		rule := css_ast.Rule{
			Data: &css_ast.RSelector{
				Rules: []css_ast.Rule{
					{Data: &css_ast.RDeclaration{
						KeyText: "color",
						Value:   []css_ast.Token{{Kind: css_lexer.TIdent, Text: "red"}},
					}},
				},
			},
		}
		reIndexRule(&rule, 10, 20)
		require.NotNil(t, rule.Data)
	})

	t.Run("re-indexes RKnownAt rule", func(t *testing.T) {
		t.Parallel()
		rule := css_ast.Rule{
			Data: &css_ast.RKnownAt{
				AtToken: "@media",
				Prelude: []css_ast.Token{{Kind: css_lexer.TIdent, Text: "screen"}},
				Rules:   []css_ast.Rule{},
			},
		}
		reIndexRule(&rule, 5, 3)
		require.NotNil(t, rule.Data)
	})

	t.Run("re-indexes RDeclaration rule", func(t *testing.T) {
		t.Parallel()
		rule := css_ast.Rule{
			Data: &css_ast.RDeclaration{
				KeyText: "background",
				Value:   []css_ast.Token{{Kind: css_lexer.TIdent, Text: "blue"}},
			},
		}
		reIndexRule(&rule, 2, 1)
		require.NotNil(t, rule.Data)
	})

	t.Run("re-indexes RAtLayer rule", func(t *testing.T) {
		t.Parallel()
		rule := css_ast.Rule{
			Data: &css_ast.RAtLayer{
				Names: [][]string{{"base"}},
				Rules: []css_ast.Rule{},
			},
		}
		reIndexRule(&rule, 0, 0)
		require.NotNil(t, rule.Data)
	})

	t.Run("re-indexes RAtKeyframes rule", func(t *testing.T) {
		t.Parallel()
		rule := css_ast.Rule{
			Data: &css_ast.RAtKeyframes{
				AtToken: "@keyframes",
				Blocks:  []css_ast.KeyframeBlock{},
			},
		}
		reIndexRule(&rule, 1, 2)
		require.NotNil(t, rule.Data)
	})

	t.Run("re-indexes RBadDeclaration rule", func(t *testing.T) {
		t.Parallel()
		rule := css_ast.Rule{
			Data: &css_ast.RBadDeclaration{
				Tokens: []css_ast.Token{{Kind: css_lexer.TIdent, Text: "bad"}},
			},
		}
		reIndexRule(&rule, 3, 4)
		require.NotNil(t, rule.Data)
	})

	t.Run("re-indexes RUnknownAt rule", func(t *testing.T) {
		t.Parallel()
		rule := css_ast.Rule{
			Data: &css_ast.RUnknownAt{
				AtToken: "@custom",
				Prelude: []css_ast.Token{{Kind: css_lexer.TIdent, Text: "x"}},
				Block:   []css_ast.Token{{Kind: css_lexer.TIdent, Text: "y"}},
			},
		}
		reIndexRule(&rule, 7, 8)
		require.NotNil(t, rule.Data)
	})
}

func TestExtractLayerNamesFromTokens(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty tokens", func(t *testing.T) {
		t.Parallel()
		result := extractLayerNamesFromTokens([]css_ast.Token{})
		require.Len(t, result, 1)
		assert.Empty(t, result[0])
	})

	t.Run("returns empty slice for nil tokens", func(t *testing.T) {
		t.Parallel()
		result := extractLayerNamesFromTokens(nil)
		require.Len(t, result, 1)
		assert.Empty(t, result[0])
	})

	t.Run("extracts layer name from layer function token", func(t *testing.T) {
		t.Parallel()
		children := []css_ast.Token{
			{Kind: css_lexer.TIdent, Text: "base"},
		}
		tokens := []css_ast.Token{
			{Kind: css_lexer.TFunction, Text: "layer", Children: &children},
		}
		result := extractLayerNamesFromTokens(tokens)
		require.Len(t, result, 1)
		require.Len(t, result[0], 1)
		assert.Equal(t, "base", result[0][0])
	})

	t.Run("extracts dotted layer name", func(t *testing.T) {
		t.Parallel()
		children := []css_ast.Token{
			{Kind: css_lexer.TIdent, Text: "foo"},
			{Kind: css_lexer.TDelimDot},
			{Kind: css_lexer.TIdent, Text: "bar"},
		}
		tokens := []css_ast.Token{
			{Kind: css_lexer.TFunction, Text: "layer", Children: &children},
		}
		result := extractLayerNamesFromTokens(tokens)
		require.Len(t, result, 1)
		require.Len(t, result[0], 2)
		assert.Equal(t, "foo", result[0][0])
		assert.Equal(t, "bar", result[0][1])
	})

	t.Run("returns default for non-layer function token", func(t *testing.T) {
		t.Parallel()
		tokens := []css_ast.Token{
			{Kind: css_lexer.TIdent, Text: "screen"},
		}
		result := extractLayerNamesFromTokens(tokens)
		require.Len(t, result, 1)
		assert.Empty(t, result[0])
	})

	t.Run("returns default when layer function has nil children", func(t *testing.T) {
		t.Parallel()
		tokens := []css_ast.Token{
			{Kind: css_lexer.TFunction, Text: "layer", Children: nil},
		}
		result := extractLayerNamesFromTokens(tokens)
		require.Len(t, result, 1)
		assert.Empty(t, result[0])
	})
}

func TestParseLayerNameFromChildren(t *testing.T) {
	t.Parallel()

	t.Run("returns empty for empty children", func(t *testing.T) {
		t.Parallel()
		result := parseLayerNameFromChildren([]css_ast.Token{})
		require.Len(t, result, 1)
		assert.Empty(t, result[0])
	})

	t.Run("returns single part for single ident", func(t *testing.T) {
		t.Parallel()
		children := []css_ast.Token{
			{Kind: css_lexer.TIdent, Text: "base"},
		}
		result := parseLayerNameFromChildren(children)
		require.Len(t, result, 1)
		require.Len(t, result[0], 1)
		assert.Equal(t, "base", result[0][0])
	})

	t.Run("returns multiple parts for dotted name", func(t *testing.T) {
		t.Parallel()
		children := []css_ast.Token{
			{Kind: css_lexer.TIdent, Text: "framework"},
			{Kind: css_lexer.TDelimDot},
			{Kind: css_lexer.TIdent, Text: "layout"},
		}
		result := parseLayerNameFromChildren(children)
		require.Len(t, result, 1)
		require.Len(t, result[0], 2)
		assert.Equal(t, "framework", result[0][0])
		assert.Equal(t, "layout", result[0][1])
	})

	t.Run("returns default when no ident tokens present", func(t *testing.T) {
		t.Parallel()
		children := []css_ast.Token{
			{Kind: css_lexer.TDelimDot},
		}
		result := parseLayerNameFromChildren(children)
		require.Len(t, result, 1)
		assert.Empty(t, result[0])
	})
}

func TestWrapImportedASTWithConditions(t *testing.T) {
	t.Parallel()

	t.Run("returns original AST when conditions is nil", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.AST{
			Rules: []css_ast.Rule{
				{Data: &css_ast.RComment{Text: "test"}},
			},
		}
		result := WrapImportedASTWithConditions(original, nil, es_logger.Loc{})
		assert.Same(t, original, result)
	})

	t.Run("wraps with layer condition", func(t *testing.T) {
		t.Parallel()
		children := []css_ast.Token{
			{Kind: css_lexer.TIdent, Text: "base"},
		}
		original := &css_ast.AST{
			Rules: []css_ast.Rule{
				{Data: &css_ast.RComment{Text: "inner"}},
			},
		}
		conditions := &css_ast.ImportConditions{
			Layers: []css_ast.Token{
				{Kind: css_lexer.TFunction, Text: "layer", Children: &children},
			},
		}
		result := WrapImportedASTWithConditions(original, conditions, es_logger.Loc{})
		require.NotNil(t, result)
		require.Len(t, result.Rules, 1)
		layer, ok := result.Rules[0].Data.(*css_ast.RAtLayer)
		require.True(t, ok)
		require.Len(t, layer.Names, 1)
		assert.Equal(t, []string{"base"}, layer.Names[0])
	})

	t.Run("wraps with supports condition", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.AST{
			Rules: []css_ast.Rule{
				{Data: &css_ast.RComment{Text: "inner"}},
			},
		}
		conditions := &css_ast.ImportConditions{
			Supports: []css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "display: grid"},
			},
		}
		result := WrapImportedASTWithConditions(original, conditions, es_logger.Loc{})
		require.NotNil(t, result)
		require.Len(t, result.Rules, 1)
		knownAt, ok := result.Rules[0].Data.(*css_ast.RKnownAt)
		require.True(t, ok)
		assert.Equal(t, "@supports", knownAt.AtToken)
	})

	t.Run("wraps with media query condition", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.AST{
			Rules: []css_ast.Rule{
				{Data: &css_ast.RComment{Text: "inner"}},
			},
		}
		conditions := &css_ast.ImportConditions{
			Queries: []css_ast.MediaQuery{
				{Loc: es_logger.Loc{}},
			},
		}
		result := WrapImportedASTWithConditions(original, conditions, es_logger.Loc{})
		require.NotNil(t, result)
		require.Len(t, result.Rules, 1)
		media, ok := result.Rules[0].Data.(*css_ast.RAtMedia)
		require.True(t, ok)
		require.Len(t, media.Queries, 1)
	})
}

func TestCloneAST(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil input", func(t *testing.T) {
		t.Parallel()
		result := CloneAST(nil)
		assert.Nil(t, result)
	})

	t.Run("clones AST with rules and symbols", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.AST{
			Symbols: []es_ast.Symbol{
				{OriginalName: "test"},
			},
			ImportRecords: []es_ast.ImportRecord{
				{},
			},
			Rules: []css_ast.Rule{
				{Data: &css_ast.RComment{Text: "hello"}},
			},
			ApproximateLineCount: 42,
		}
		clone := CloneAST(original)

		require.NotNil(t, clone)
		assert.NotSame(t, original, clone)
		assert.Equal(t, int32(42), clone.ApproximateLineCount)
		require.Len(t, clone.Symbols, 1)
		require.Len(t, clone.ImportRecords, 1)
		require.Len(t, clone.Rules, 1)
	})

	t.Run("clones AST with empty slices", func(t *testing.T) {
		t.Parallel()
		original := &css_ast.AST{
			ApproximateLineCount: 0,
		}
		clone := CloneAST(original)

		require.NotNil(t, clone)
		assert.NotSame(t, original, clone)
		assert.Nil(t, clone.Symbols)
		assert.Nil(t, clone.ImportRecords)
		assert.Nil(t, clone.Rules)
	})
}

func TestGetInliner(t *testing.T) {
	t.Run("returns configured inliner from pool", func(t *testing.T) {
		fsReader := newTestFSReader()
		resolver := newPassthroughResolver()
		processor := newTestProcessor(resolver)

		ci := GetInliner(processor.GetResolver(), processor.GetParserOptions(), fsReader, "TEST")

		require.NotNil(t, ci)
		assert.Same(t, resolver, ci.resolver)
		assert.NotNil(t, ci.cache)
		assert.Nil(t, ci.diagnostics)

		PutInliner(ci)
	})
}

func TestMergeASTs(t *testing.T) {
	t.Run("merges child rules into parent", func(t *testing.T) {
		parent := &css_ast.AST{
			Rules: []css_ast.Rule{
				{Data: &css_ast.RComment{Text: "parent"}},
			},
			Symbols:       []es_ast.Symbol{},
			ImportRecords: []es_ast.ImportRecord{},
		}
		child := &css_ast.AST{
			Rules: []css_ast.Rule{
				{Data: &css_ast.RComment{Text: "child"}},
			},
			Symbols:       []es_ast.Symbol{},
			ImportRecords: []es_ast.ImportRecord{},
		}

		MergeASTs(parent, child)

		require.Len(t, parent.Rules, 2)

		firstComment, ok := parent.Rules[0].Data.(*css_ast.RComment)
		require.True(t, ok)
		assert.Equal(t, "child", firstComment.Text)
		secondComment, ok := parent.Rules[1].Data.(*css_ast.RComment)
		require.True(t, ok)
		assert.Equal(t, "parent", secondComment.Text)
	})
}

func TestCSSInliner_ConditionalImports(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newPassthroughResolver()
	processor := newTestProcessor(resolver)

	t.Run("should handle import with layer condition", func(t *testing.T) {
		fsReader.addFile("/test/layered.css", `.layered { display: block; }`)

		css := `@import "/test/layered.css" layer(base);`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)
		assert.NotEmpty(t, tree.Rules)
	})

	t.Run("should handle import with media query", func(t *testing.T) {
		fsReader.addFile("/test/responsive.css", `.responsive { width: 100%; }`)

		css := `@import "/test/responsive.css" screen and (min-width: 768px);`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)
		assert.NotEmpty(t, tree.Rules)
	})

	t.Run("should handle import with supports condition", func(t *testing.T) {
		fsReader.addFile("/test/modern.css", `.modern { display: grid; }`)

		css := `@import "/test/modern.css" supports(display: grid);`
		inliner := newTestInliner(processor, fsReader)
		defer PutInliner(inliner)
		tree, diagnostics := inliner.InlineAndParse(ctx, css, "main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.NotNil(t, tree)
		assert.Empty(t, diagnostics)
		assert.NotEmpty(t, tree.Rules)
	})
}
