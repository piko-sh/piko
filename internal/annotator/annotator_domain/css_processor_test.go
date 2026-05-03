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

package annotator_domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/cssinliner"
	"piko.sh/piko/internal/esbuild/compat"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/css_ast"
	es_logger "piko.sh/piko/internal/esbuild/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (r *testFSReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	content, exists := r.files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return []byte(content), nil
}

type testResolver struct {
	resolveCSSFunc func(ctx context.Context, importPath, fromDir string) (string, error)
}

func newTestResolver() *testResolver {
	return &testResolver{
		resolveCSSFunc: func(ctx context.Context, importPath, fromDir string) (string, error) {

			return importPath, nil
		},
	}
}

func (r *testResolver) ResolveCSSPath(ctx context.Context, importPath, fromDir string) (string, error) {
	return r.resolveCSSFunc(ctx, importPath, fromDir)
}
func (r *testResolver) ResolveModulePath(ctx context.Context, modulePath string) (string, error) {
	return "", errors.New("not implemented")
}
func (r *testResolver) ResolvePKPath(_ context.Context, _ string, _ string) (string, error) {
	return "", errors.New("not implemented")
}
func (r *testResolver) GetModuleName() string { return "test_module" }
func (r *testResolver) GetBaseDir() string    { return "/test" }
func (r *testResolver) DetectLocalModule(ctx context.Context) error {
	return nil
}
func (r *testResolver) ConvertEntryPointPathToManifestKey(entryPointPath string) string {

	return entryPointPath
}
func (r *testResolver) ResolveAssetPath(_ context.Context, importPath string, _ string) (string, error) {
	return importPath, nil
}
func (*testResolver) GetModuleDir(_ context.Context, _ string) (string, error) {
	return "", errors.New("GetModuleDir not implemented in test mock")
}
func (*testResolver) FindModuleBoundary(_ context.Context, _ string) (string, string, error) {
	return "", "", errors.New("FindModuleBoundary not implemented in test mock")
}

func simpleParse(t *testing.T, html string) *ast_domain.TemplateAST {
	ast, err := ast_domain.Parse(context.Background(), html, "test.pk", nil)
	require.NoError(t, err)
	return ast
}

func normaliseCSS(css string) string {
	css = strings.ReplaceAll(css, "\n", "")
	css = strings.ReplaceAll(css, "\t", "")
	for strings.Contains(css, "  ") {
		css = strings.ReplaceAll(css, "  ", " ")
	}
	css = strings.ReplaceAll(css, "{ ", "{")
	css = strings.ReplaceAll(css, " }", "}")
	css = strings.ReplaceAll(css, "; ", ";")
	css = strings.ReplaceAll(css, ": ", ":")
	css = strings.ReplaceAll(css, ", ", ",")
	css = strings.ReplaceAll(css, " > ", ">")
	css = strings.ReplaceAll(css, " (", "(")
	return css
}

func TestCSSProcessor_Process(t *testing.T) {
	testCases := []struct {
		setupFS     func(fs *testFSReader)
		diagCheck   func(t *testing.T, diagnostics []*ast_domain.Diagnostic)
		minifyOpts  *config.Options
		name        string
		inputCSS    string
		expectedCSS string
		expectDiags bool
	}{
		{
			name:        "should minify valid CSS",
			inputCSS:    `.my-class { color: red; font-size: 16px; } #my-id > p { margin: 10px; }`,
			expectedCSS: `.my-class{color:red;font-size:16px}#my-id>p{margin:10px}`,
		},
		{
			name:        "should return empty string for empty input",
			inputCSS:    "",
			expectedCSS: "",
		},
		{
			name:        "should return empty string for whitespace-only input",
			inputCSS:    "  \n\t  ",
			expectedCSS: "",
		},
		{
			name:     "should handle simple @import",
			inputCSS: `@import "/test/base.css"; .container { padding: 10px; }`,
			setupFS: func(fs *testFSReader) {
				fs.addFile("/test/base.css", `body { margin: 0; }`)
			},

			expectedCSS: `body{margin:0}.container{padding:10px}`,
		},
		{
			name:     "should handle multiple @import statements",
			inputCSS: `@import "a.css"; @import "b.css";`,
			setupFS: func(fs *testFSReader) {
				fs.addFile("a.css", `div{color:red}`)
				fs.addFile("b.css", `span{color:blue}`)
			},

			expectedCSS: `div{color:red}span{color:#00f}`,
		},
		{
			name:     "should handle nested @import statements",
			inputCSS: `@import "a.css";`,
			setupFS: func(fs *testFSReader) {
				fs.addFile("a.css", `@import "b.css"; .a{color:red}`)
				fs.addFile("b.css", `.b{color:blue}`)
			},

			expectedCSS: `.b{color:#00f}.a{color:red}`,
		},
		{
			name:     "should handle @import with media query",
			inputCSS: `@import "print.css" print;`,
			setupFS: func(fs *testFSReader) {
				fs.addFile("print.css", `body{font-size:12pt}`)
			},

			expectedCSS: `@media print{body{font-size:12pt}}`,
		},
		{
			name:     "should handle @import with layer",
			inputCSS: `@import "theme.css" layer(theme);`,
			setupFS: func(fs *testFSReader) {
				fs.addFile("theme.css", `:root{--bg:white}`)
			},

			expectedCSS: `@layer theme{:root{--bg:white}}`,
		},
		{
			name:        "should handle missing import file gracefully",
			inputCSS:    `@import "/nonexistent.css"; .container { padding: 10px; }`,
			expectedCSS: `.container{padding:10px}`,
			expectDiags: true,
			diagCheck: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				require.Len(t, diagnostics, 1)
				assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
				assert.Contains(t, diagnostics[0].Message, "file not found")
			},
		},
		{
			name:     "should detect circular @import",
			inputCSS: `@import "a.css";`,
			setupFS: func(fs *testFSReader) {
				fs.addFile("a.css", `@import "b.css";`)
				fs.addFile("b.css", `@import "a.css";`)
			},
			expectedCSS: "",
			expectDiags: true,
			diagCheck: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				require.Len(t, diagnostics, 1)
				assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
				assert.Contains(t, diagnostics[0].Message, "circular dependency detected: test.css -> a.css -> b.css -> a.css")
			},
		},
		{
			name:     "should preserve @charset at the top",
			inputCSS: `@charset "UTF-8"; body { color: black; }`,

			expectedCSS: `@charset "UTF-8";body{color:#000}`,
		},
		{
			name:     "should recover from CSS syntax errors and report them",
			inputCSS: `.my-class { color: red; #invalid-token }`,

			expectedCSS: `.my-class{color:red;#invalid-token}`,
			expectDiags: true,
			diagCheck: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				require.Len(t, diagnostics, 1)
				assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
				assert.Contains(t, diagnostics[0].Message, "Expected identifier but found \"#invalid-token\"")
			},
		},
		{
			name: "should not minify if options are disabled",
			inputCSS: `
				.my-class {
					color: red;
				}
			`,
			minifyOpts: &config.Options{
				MinifyWhitespace: false,
				MinifySyntax:     false,
			},

			expectedCSS: `.my-class {color:red;}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fsReader := newTestFSReader()
			resolver := newTestResolver()

			if tc.setupFS != nil {
				tc.setupFS(fsReader)
			}

			opts := &config.Options{
				MinifyWhitespace:       true,
				MinifySyntax:           true,
				UnsupportedCSSFeatures: compat.Nesting,
			}
			if tc.minifyOpts != nil {
				opts = tc.minifyOpts
			}
			processor := NewCSSProcessor(config.LoaderLocalCSS, opts, resolver)

			output, diagnostics, err := processor.Process(ctx, tc.inputCSS, "test.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0}, fsReader)
			require.NoError(t, err)

			if tc.expectDiags {
				assert.NotEmpty(t, diagnostics)
				if tc.diagCheck != nil {
					tc.diagCheck(t, diagnostics)
				}
			} else {
				assert.Empty(t, diagnostics)
			}

			assert.Equal(t, normaliseCSS(tc.expectedCSS), normaliseCSS(output))
		})
	}
}

func TestCSSProcessor_ProcessAndScope(t *testing.T) {
	scopeID := "p-abc123"

	testCases := []struct {
		name              string
		inputTemplate     string
		inputCSS          string
		expectedScopedCSS string
	}{
		{
			name:          "should scope basic selectors",
			inputTemplate: `<div><p>Hello</p></div>`,
			inputCSS:      `.container { padding: 10px; } p { color: blue; }`,

			expectedScopedCSS: `.container[partial~=p-abc123]{padding:10px}p[partial~=p-abc123]{color:#00f}`,
		},
		{
			name:          "should scope root and descendant selectors correctly",
			inputTemplate: `<div class="root"><p></p></div>`,
			inputCSS:      `.root { border: 1px solid black; } p { color: red; }`,

			expectedScopedCSS: `.root[partial~=p-abc123]{border:1px solid black}p[partial~=p-abc123]{color:red}`,
		},
		{
			name:          "should scope multiple selectors in a list",
			inputTemplate: `<h1></h1><p></p>`,
			inputCSS:      `h1, p { margin: 0; }`,

			expectedScopedCSS: `h1[partial~=p-abc123],p[partial~=p-abc123]{margin:0}`,
		},
		{
			name:          "should scope complex selectors with combinators",
			inputTemplate: `<div id="app"><ul><li>Item</li></ul></div>`,
			inputCSS:      `#app > ul li:first-child { font-weight: bold; }`,

			expectedScopedCSS: `#app[partial~=p-abc123]>ul[partial~=p-abc123] li[partial~=p-abc123]:first-child{font-weight:700}`,
		},
		{
			name:          "should scope inside media queries",
			inputTemplate: `<div><p></p></div>`,
			inputCSS:      `@media (min-width: 600px) { p { color: green; } }`,

			expectedScopedCSS: `@media(min-width:600px){p[partial~=p-abc123]{color:green}}`,
		},
		{
			name:          "should not scope inside @keyframes",
			inputTemplate: `<div></div>`,
			inputCSS: `
				@keyframes slide-in {
					from { transform: translateX(-100%); }
					to { transform: translateX(0); }
				}
				.animated { animation: slide-in 1s; }
			`,

			expectedScopedCSS: `@keyframes slide-in{0%{transform:translate(-100%)}to{transform:translate(0)}}.animated[partial~=p-abc123]{animation:slide-in 1s}`,
		},
		{
			name:          "should not scope :root selector",
			inputTemplate: `<div></div>`,
			inputCSS:      `:root { --main-color: blue; }`,

			expectedScopedCSS: `:root{--main-color:blue}`,
		},
		{
			name:          "should not scope html or body selectors",
			inputTemplate: `<div></div>`,
			inputCSS:      `html, body { font-family: sans-serif; }`,

			expectedScopedCSS: `html,body{font-family:sans-serif}`,
		},
		{
			name:          "should correctly scope pseudo-classes",
			inputTemplate: `<a href="#">Link</a>`,
			inputCSS:      `a:hover { text-decoration: underline; }`,

			expectedScopedCSS: `a[partial~=p-abc123]:hover{text-decoration:underline}`,
		},
		{
			name:          "should correctly scope pseudo-elements",
			inputTemplate: `<p>Text</p>`,
			inputCSS:      `p::before { content: "> "; }`,

			expectedScopedCSS: `p[partial~=p-abc123]:before{content:"> "}`,
		},
		{
			name:          "should handle :deep() pseudo-class for piercing scope",
			inputTemplate: `<div><span class="deep-child"></span></div>`,
			inputCSS:      `:deep(.deep-child) { color: purple; }`,

			expectedScopedCSS: `[partial~=p-abc123] .deep-child{color:purple}`,
		},
		{
			name:          "should handle :global() pseudo-class for unscoped selectors",
			inputTemplate: `<div></div>`,
			inputCSS:      `:global(.modal-backdrop) { position: fixed; }`,

			expectedScopedCSS: `.modal-backdrop{position:fixed}`,
		},
		{
			name:          "should handle nested :global()",
			inputTemplate: `<div class="parent"></div>`,
			inputCSS:      `.parent :global(div.child) { color: red; }`,

			expectedScopedCSS: `.parent div.child{color:red}`,
		},
		{
			name:              "should do nothing for empty CSS block",
			inputTemplate:     `<div></div>`,
			inputCSS:          "   ",
			expectedScopedCSS: "",
		},
		{
			name:          "should preserve break-before declarations",
			inputTemplate: `<div class="break-before"></div>`,
			inputCSS:      `.break-before { break-before: page; }`,

			expectedScopedCSS: `.break-before[partial~=p-abc123]{break-before:page}`,
		},
		{
			name:          "should preserve break-after declarations",
			inputTemplate: `<div class="break-after"></div>`,
			inputCSS:      `.break-after { break-after: page; }`,

			expectedScopedCSS: `.break-after[partial~=p-abc123]{break-after:page}`,
		},
		{
			name:          "should preserve break-inside declarations",
			inputTemplate: `<div class="no-break"></div>`,
			inputCSS:      `.no-break { break-inside: avoid; }`,

			expectedScopedCSS: `.no-break[partial~=p-abc123]{break-inside:avoid}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fsReader := newTestFSReader()
			resolver := newTestResolver()
			processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
				MinifyWhitespace:       true,
				MinifySyntax:           true,
				UnsupportedCSSFeatures: compat.Nesting,
			}, resolver)

			template := simpleParse(t, tc.inputTemplate)
			css := tc.inputCSS

			diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
				template:      template,
				cssBlock:      &css,
				scopeID:       scopeID,
				sourcePath:    "test.css",
				startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				fsReader:      fsReader,
			})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)

			assert.Equal(t, normaliseCSS(tc.expectedScopedCSS), normaliseCSS(css))
		})
	}
}

func TestCSSProcessor_ProcessAndScope_WithImports(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "p-xyz789"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/base.css", `.base { margin: 0; }`)
	fsReader.addFile("/test/theme.css", `@import "/test/base.css"; :root { --theme: blue; }`)

	template := simpleParse(t, `<div class="base local"></div>`)
	css := `@import "/test/theme.css"; .local { padding: 10px; }`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)

	assert.Contains(t, normalised, ".base[partial~=p-xyz789]")
	assert.Contains(t, normalised, ".local[partial~=p-xyz789]")
	assert.Contains(t, normalised, ":root{--theme:blue}")
	assert.NotContains(t, normalised, "@import")
}

func TestCSSProcessor_ProcessAndScope_Errors(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "p-abc123"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{}, resolver)

	t.Run("should return error for nil template", func(t *testing.T) {
		_, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
			template:      nil,
			cssBlock:      new(`p {}`),
			scopeID:       scopeID,
			sourcePath:    "test.css",
			startLocation: ast_domain.Location{},
			fsReader:      fsReader,
		})
		assert.Error(t, err)
	})

	t.Run("should return error for nil CSS pointer", func(t *testing.T) {
		template := simpleParse(t, `<div></div>`)
		_, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
			template:      template,
			cssBlock:      nil,
			scopeID:       scopeID,
			sourcePath:    "test.css",
			startLocation: ast_domain.Location{},
			fsReader:      fsReader,
		})
		assert.Error(t, err)
	})

	t.Run("should return error for empty scope ID", func(t *testing.T) {
		template := simpleParse(t, `<div></div>`)
		_, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
			template:      template,
			cssBlock:      new(`p {}`),
			scopeID:       "",
			sourcePath:    "test.css",
			startLocation: ast_domain.Location{},
			fsReader:      fsReader,
		})
		assert.Error(t, err)
	})
}

func TestCSSProcessor_DiagnosticLocations(t *testing.T) {
	testCases := []struct {
		setupFS       func(fs *testFSReader)
		diagCheck     func(t *testing.T, diagnostics []*ast_domain.Diagnostic)
		name          string
		inputCSS      string
		startLocation ast_domain.Location
	}{
		{
			name:          "should correctly offset location for missing import on first line",
			inputCSS:      `@import "/missing.css";`,
			startLocation: ast_domain.Location{Line: 10, Column: 5, Offset: 0},
			diagCheck: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				require.Len(t, diagnostics, 1)
				diagnostic := diagnostics[0]
				assert.Equal(t, ast_domain.Error, diagnostic.Severity)
				assert.Equal(t, "test.css", diagnostic.SourcePath)

				assert.Equal(t, 10, diagnostic.Location.Line)
				assert.Equal(t, 5, diagnostic.Location.Column)
			},
		},
		{
			name:          "should correctly offset location for syntax error on a subsequent line",
			inputCSS:      "\n  .my-class { color: red; #invalid-token }",
			startLocation: ast_domain.Location{Line: 5, Column: 3, Offset: 0},
			diagCheck: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				require.Len(t, diagnostics, 1)
				diagnostic := diagnostics[0]
				assert.Equal(t, ast_domain.Warning, diagnostic.Severity)
				assert.Equal(t, "test.css", diagnostic.SourcePath)

				assert.Equal(t, 6, diagnostic.Location.Line)

				assert.Greater(t, diagnostic.Location.Column, 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fsReader := newTestFSReader()
			resolver := newTestResolver()

			processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{}, resolver)

			if tc.setupFS != nil {
				tc.setupFS(fsReader)
			}

			_, diagnostics, err := processor.Process(ctx, tc.inputCSS, "test.css", tc.startLocation, fsReader)
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)

			if tc.diagCheck != nil {
				tc.diagCheck(t, diagnostics)
			}
		})
	}
}

func TestCSSProcessor_ProcessAndScope_MultipleClassSelectorsWithMediaQuery(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "partial_layout"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	template := simpleParse(t, `<div class="container-md"><div class="text-section"></div></div>`)
	css := `
		/* Containers */
		[class*="container-"] {
			margin: 0 auto;
		}
		.container-lg {
			max-width: 1180px;
		}
		.container-md {
			max-width: 780px;
		}

		/* Page layouts */
		.text-section {
			&:first-of-type {
				margin-top: 40px;
			}
		}

		@media screen and (max-width: 768px) {
			.text-section {
				&:first-of-type {
					margin-top: 30px;
				}
			}
		}
	`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)

	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".text-section", "Should contain .text-section class")
	assert.Contains(t, normalised, "@media", "Should contain media query")

	assert.NotContains(t, normalised, "@media(max-width:768px){.container-md[partial~=partial_layout]",
		"Media query should NOT incorrectly reference .container-md")

	assert.Contains(t, normalised, "@media screen and(max-width:768px){.text-section[partial~=partial_layout]:first-of-type",
		"Media query should correctly reference .text-section:first-of-type (not .container-md)")

	assert.Contains(t, normalised, ".container-md[partial~=partial_layout]", "Should scope .container-md")
}

func TestCSSProcessor_ProcessAndScope_PseudoClassSelectorInMediaQuery(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "partial_layout"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	template := simpleParse(t, `<div class="container-md"><div class="text-section"></div></div>`)
	css := `
		.container-md {
			max-width: 780px;
		}
		.text-section {
			margin-top: 40px;
		}

		@media screen and (max-width: 768px) {
			.text-section:first-of-type {
				margin-top: 30px;
			}
		}
	`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".text-section", "Should contain .text-section class")
	assert.Contains(t, normalised, "@media", "Should contain media query")

	assert.NotContains(t, normalised, "@media screen and(max-width:768px){[partial~=partial_layout] .container-md:first-of-type",
		"Media query should NOT incorrectly use .container-md")
	assert.Contains(t, normalised, ".text-section", "Media query should correctly reference .text-section")
}

func TestCSSProcessor_ProcessAndScope_ImportWithNestedMediaQuery(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "partial_layout"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/helpers.css", `
		/* Containers */
		[class*="container-"] {
			margin: 0 auto;
		}
		.container-lg {
			max-width: 1180px;
		}
		.container-md {
			max-width: 780px;
		}

		/* Page layouts */
		.text-section {
			&:first-of-type {
				margin-top: 40px;
			}
		}

		@media screen and (max-width: 768px) {
			.text-section {
				&:first-of-type {
					margin-top: 30px;
				}
			}
		}
	`)

	template := simpleParse(t, `<div class="container-md"><div class="text-section"><div class="layout-container"></div></div></div>`)
	css := `
		@import "/test/helpers.css";

		.layout-container {
			min-height: 100vh;
		}
	`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".text-section", "Should contain .text-section class")
	assert.Contains(t, normalised, ".container-md", "Should contain .container-md class")
	assert.Contains(t, normalised, "@media", "Should contain media query")

	mediaQueryStart := strings.Index(normalised, "@media")
	assert.True(t, mediaQueryStart >= 0, "Should contain @media query")

	mediaQuerySection := normalised[mediaQueryStart:]
	mediaQueryEnd := strings.Index(mediaQuerySection, "}}")
	if mediaQueryEnd > 0 && mediaQueryEnd < len(mediaQuerySection) {
		mediaQuerySection = mediaQuerySection[:mediaQueryEnd+2]
	}

	hasContainerMdInMediaQuery := strings.Contains(mediaQuerySection, ".container-md") &&
		strings.Contains(mediaQuerySection, ":first-of-type{margin-top:30px")

	hasTextSectionInMediaQuery := strings.Contains(mediaQuerySection, ".text-section[partial~=") &&
		strings.Contains(mediaQuerySection, ":first-of-type{margin-top:30px")

	if hasContainerMdInMediaQuery {
		t.Errorf("BUG REPRODUCED: Media query incorrectly contains .container-md instead of .text-section")
		t.Errorf("Media query section: %s", mediaQuerySection)
	}

	assert.True(t, hasTextSectionInMediaQuery, "Media query should correctly reference .text-section, not .container-md")
}

func TestCSSProcessor_EdgeCases_MultipleMediaQueries(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/styles.css", `
		.header { height: 60px; }
		.sidebar { width: 250px; }
		.content { flex: 1; }

		@media (max-width: 768px) {
			.header { height: 50px; }
		}

		@media (max-width: 480px) {
			.sidebar { width: 100%; }
			.content { padding: 10px; }
		}
	`)

	template := simpleParse(t, `<div class="header"><div class="sidebar"></div><div class="content"></div></div>`)
	css := `@import "/test/styles.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)

	assert.Contains(t, normalised, ".header", "Should contain .header")
	assert.Contains(t, normalised, ".sidebar", "Should contain .sidebar")
	assert.Contains(t, normalised, ".content", "Should contain .content")

	assert.NotContains(t, normalised, "@media(max-width:768px){[test-scope] .sidebar", "First media query should not have .sidebar")
	assert.NotContains(t, normalised, "@media(max-width:480px){[test-scope] .header", "Second media query should not have .header")
}

func TestCSSProcessor_EdgeCases_NestedAtRules(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/nested.css", `
		.grid { display: grid; }
		.flex { display: flex; }

		@media (min-width: 1024px) {
			@supports (display: grid) {
				.grid { grid-template-columns: repeat(3, 1fr); }
			}
		}

		@layer base {
			.flex { gap: 20px; }
		}
	`)

	template := simpleParse(t, `<div class="grid"><div class="flex"></div></div>`)
	css := `@import "/test/nested.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".grid", "Should contain .grid")
	assert.Contains(t, normalised, ".flex", "Should contain .flex")
}

func TestCSSProcessor_EdgeCases_ComplexSelectors(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/complex.css", `
		.parent { position: relative; }
		.child { position: absolute; }
		.sibling { margin-left: 10px; }

		@media (max-width: 768px) {
			.parent > .child { top: 0; }
			.parent + .sibling { margin-top: 20px; }
			.parent ~ .sibling { opacity: 0.5; }
		}
	`)

	template := simpleParse(t, `<div class="parent"><div class="child"></div></div><div class="sibling"></div>`)
	css := `@import "/test/complex.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".parent", "Should contain .parent")
	assert.Contains(t, normalised, ".child", "Should contain .child")
	assert.Contains(t, normalised, ".sibling", "Should contain .sibling")
}

func TestCSSProcessor_EdgeCases_IDSelectors(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/ids.css", `
		#header { height: 60px; }
		.content { padding: 20px; }

		@media (max-width: 768px) {
			#header { height: 50px; }
			.content { padding: 10px; }
		}
	`)

	template := simpleParse(t, `<div id="header"><div class="content"></div></div>`)
	css := `@import "/test/ids.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, "#header", "Should contain #header")
	assert.Contains(t, normalised, ".content", "Should contain .content")
}

func TestCSSProcessor_EdgeCases_AttributeSelectors(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/attrs.css", `
		[data-type="primary"] { color: blue; }
		.button { padding: 10px; }

		@media (max-width: 768px) {
			[data-type="primary"] { font-size: 14px; }
			.button { padding: 8px; }
		}
	`)

	template := simpleParse(t, `<button class="button" data-type="primary">Click</button>`)
	css := `@import "/test/attrs.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, "[data-type=primary]", "Should contain attribute selector")
	assert.Contains(t, normalised, ".button", "Should contain .button")
}

func TestCSSProcessor_EdgeCases_PseudoClasses(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/pseudo.css", `
		.link { color: blue; }
		.button { background: white; }

		@media (max-width: 768px) {
			.link:hover { color: darkblue; }
			.button:active { background: gray; }
			.link:visited { color: purple; }
		}
	`)

	template := simpleParse(t, `<a class="link">Link</a><button class="button">Button</button>`)
	css := `@import "/test/pseudo.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".link", "Should contain .link")
	assert.Contains(t, normalised, ".button", "Should contain .button")
}

func TestCSSProcessor_EdgeCases_PseudoElements(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/pseudo-elements.css", `
		.decorated { position: relative; }
		.badge { display: inline-block; }

		@media (max-width: 768px) {
			.decorated::before { content: "→"; }
			.decorated::after { content: "←"; }
			.badge::marker { color: red; }
		}
	`)

	template := simpleParse(t, `<div class="decorated"><span class="badge">New</span></div>`)
	css := `@import "/test/pseudo-elements.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".decorated", "Should contain .decorated")
	assert.Contains(t, normalised, ".badge", "Should contain .badge")
}

func TestCSSProcessor_EdgeCases_ChainedImports(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/c.css", `
		.base { margin: 0; }
		@media (max-width: 768px) {
			.base { margin: 10px; }
		}
	`)

	fsReader.addFile("/test/b.css", `
		@import "/test/c.css";
		.middle { padding: 0; }
		@media (max-width: 768px) {
			.middle { padding: 5px; }
		}
	`)

	fsReader.addFile("/test/a.css", `
		@import "/test/b.css";
		.top { border: none; }
		@media (max-width: 768px) {
			.top { border: 1px solid; }
		}
	`)

	template := simpleParse(t, `<div class="base"><div class="middle"><div class="top"></div></div></div>`)
	css := `@import "/test/a.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".base", "Should contain .base")
	assert.Contains(t, normalised, ".middle", "Should contain .middle")
	assert.Contains(t, normalised, ".top", "Should contain .top")
}

func TestCSSProcessor_EdgeCases_MultipleClassesOnSelector(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/multi.css", `
		.btn { padding: 10px; }
		.btn.primary { background: blue; }
		.btn.secondary { background: gray; }

		@media (max-width: 768px) {
			.btn.primary { padding: 8px; }
			.btn.secondary { font-size: 14px; }
		}
	`)

	template := simpleParse(t, `<button class="btn primary">Primary</button><button class="btn secondary">Secondary</button>`)
	css := `@import "/test/multi.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".btn", "Should contain .btn")
	assert.Contains(t, normalised, ".primary", "Should contain .primary")
	assert.Contains(t, normalised, ".secondary", "Should contain .secondary")
}

func TestCSSProcessor_EdgeCases_EmptyMediaQuery(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/empty.css", `
		.content { padding: 20px; }
		@media (max-width: 768px) {
		}
	`)

	template := simpleParse(t, `<div class="content"></div>`)
	css := `@import "/test/empty.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".content", "Should contain .content")
}

func TestCSSProcessor_EdgeCases_ManyClassSelectors(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	var cssBuilder strings.Builder
	var templateBuilder strings.Builder
	templateBuilder.WriteString(`<div class="class-0">`)
	for i := range 50 {
		fmt.Fprintf(&cssBuilder, ".class-%d { margin: %dpx; }\n", i, i)
		if i < 49 {
			fmt.Fprintf(&templateBuilder, `<div class="class-%d">`, i+1)
		}
	}
	cssBuilder.WriteString("@media (max-width: 768px) {\n")
	for i := range 50 {
		fmt.Fprintf(&cssBuilder, "  .class-%d { padding: %dpx; }\n", i, i)
	}
	cssBuilder.WriteString("}\n")
	for range 49 {
		templateBuilder.WriteString(`</div>`)
	}
	templateBuilder.WriteString(`</div>`)
	cssContent := cssBuilder.String()
	templateContent := templateBuilder.String()

	fsReader.addFile("/test/many.css", cssContent)

	template := simpleParse(t, templateContent)
	css := `@import "/test/many.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)

	assert.Contains(t, normalised, ".class-0", "Should contain .class-0")
	assert.Contains(t, normalised, ".class-25", "Should contain .class-25")
	assert.Contains(t, normalised, ".class-49", "Should contain .class-49")
}

func TestCSSProcessor_EdgeCases_NestedCSSNesting(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/nesting.css", `
		.card {
			border: 1px solid;
			&:hover {
				.title {
					color: blue;
				}
			}
		}

		@media (max-width: 768px) {
			.card {
				&:hover {
					.title {
						font-size: 14px;
					}
				}
			}
		}
	`)

	template := simpleParse(t, `<div class="card"><h3 class="title">Title</h3></div>`)
	css := `@import "/test/nesting.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".card", "Should contain .card")
	assert.Contains(t, normalised, ".title", "Should contain .title")
}

func TestCSSProcessor_EdgeCases_CSSVariables(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/vars.css", `
		:root {
			--primary-color: blue;
			--spacing: 20px;
		}

		.container {
			color: var(--primary-color);
			padding: var(--spacing);
		}

		@media (max-width: 768px) {
			:root {
				--spacing: 10px;
			}
			.container {
				padding: var(--spacing);
			}
		}
	`)

	template := simpleParse(t, `<div class="container"></div>`)
	css := `@import "/test/vars.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".container", "Should contain .container")
	assert.Contains(t, normalised, "--primary-color", "Should contain --primary-color")
	assert.Contains(t, normalised, "--spacing", "Should contain --spacing")
}

func TestCSSProcessor_EdgeCases_CalcExpressions(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/calc.css", `
		.box {
			width: calc(100% - 20px);
		}

		@media (max-width: 768px) {
			.box {
				width: calc(100% - 10px);
				height: calc(50vh - 30px);
			}
		}
	`)

	template := simpleParse(t, `<div class="box"></div>`)
	css := `@import "/test/calc.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".box", "Should contain .box")
}

func TestCSSProcessor_EdgeCases_ContainerQueries(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/container.css", `
		.wrapper {
			container-type: inline-size;
		}

		.card {
			padding: 20px;
		}

		@container (min-width: 400px) {
			.card {
				padding: 30px;
			}
		}
	`)

	template := simpleParse(t, `<div class="wrapper"><div class="card"></div></div>`)
	css := `@import "/test/container.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".wrapper", "Should contain .wrapper")
	assert.Contains(t, normalised, ".card", "Should contain .card")
}

func TestCSSProcessor_EdgeCases_MediaQueryRanges(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/ranges.css", `
		.responsive {
			font-size: 16px;
		}

		@media (400px <= width <= 800px) {
			.responsive {
				font-size: 18px;
			}
		}

		@media (width >= 1200px) {
			.responsive {
				font-size: 20px;
			}
		}
	`)

	template := simpleParse(t, `<div class="responsive"></div>`)
	css := `@import "/test/ranges.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".responsive", "Should contain .responsive")
}

func TestCSSProcessor_EdgeCases_MixedMediaTypes(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/media-types.css", `
		.content { color: black; }
		.sidebar { display: block; }

		@media screen and (max-width: 768px) {
			.sidebar { display: none; }
		}

		@media print {
			.content { color: black; }
			.sidebar { display: none; }
		}
	`)

	template := simpleParse(t, `<div class="content"><aside class="sidebar"></aside></div>`)
	css := `@import "/test/media-types.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".content", "Should contain .content")
	assert.Contains(t, normalised, ".sidebar", "Should contain .sidebar")
}

func TestCSSProcessor_EdgeCases_PreferenceMediaQueries(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/preferences.css", `
		.theme-light { background: white; color: black; }
		.theme-dark { background: black; color: white; }
		.animated { transition: all 0.3s; }

		@media (prefers-color-scheme: dark) {
			.theme-light { background: #222; color: #eee; }
		}

		@media (prefers-reduced-motion: reduce) {
			.animated { transition: none; }
		}
	`)

	template := simpleParse(t, `<div class="theme-light animated"></div>`)
	css := `@import "/test/preferences.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".theme-light", "Should contain .theme-light")
	assert.Contains(t, normalised, ".animated", "Should contain .animated")
}

func TestCSSProcessor_EdgeCases_SameClassDifferentContexts(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/contexts.css", `
		.item { color: blue; }
		.container .item { color: red; }
		div.item { color: green; }

		@media (max-width: 768px) {
			.item { font-size: 14px; }
			.container .item { font-size: 12px; }
			div.item { font-size: 16px; }
		}
	`)

	template := simpleParse(t, `<div class="container"><div class="item"></div></div>`)
	css := `@import "/test/contexts.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".item", "Should contain .item")
	assert.Contains(t, normalised, ".container", "Should contain .container")
}

func TestCSSProcessor_EdgeCases_OrientationMediaQuery(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/orientation.css", `
		.landscape-view { display: none; }
		.portrait-view { display: block; }

		@media (orientation: landscape) {
			.landscape-view { display: block; }
			.portrait-view { display: none; }
		}
	`)

	template := simpleParse(t, `<div class="landscape-view"></div><div class="portrait-view"></div>`)
	css := `@import "/test/orientation.css";`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	assert.Contains(t, normalised, ".landscape-view", "Should contain .landscape-view")
	assert.Contains(t, normalised, ".portrait-view", "Should contain .portrait-view")
}

func TestCSSProcessor_EdgeCases_KeyframeNamesAfterImport(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/animations.css", `
		@keyframes fadeIn {
			from { opacity: 0; }
			to { opacity: 1; }
		}
		@keyframes slideIn {
			from { transform: translateX(-100%); }
			to { transform: translateX(0); }
		}
		.animate-fade {
			animation: fadeIn 0.3s ease-in-out;
		}
		.animate-slide {
			animation: slideIn 0.5s ease-out;
		}
	`)

	template := simpleParse(t, `<div class="animate-fade"></div><div class="animate-slide"></div>`)

	css := `
@import "/test/animations.css";

.container {
	display: flex;
}
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, "@keyframes fadeIn", "Should contain @keyframes fadeIn")
	assert.Contains(t, normalised, "@keyframes slideIn", "Should contain @keyframes slideIn")
	assert.Contains(t, normalised, "animation:fadeIn", "Should reference fadeIn in animation property")
	assert.Contains(t, normalised, "animation:slideIn", "Should reference slideIn in animation property")

	assert.NotContains(t, normalised, "@keyframes container", "Keyframe name should not be corrupted to parent class")
}

func TestCSSProcessor_EdgeCases_HasAndNotPseudoClassSelectors(t *testing.T) {

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/helpers.css", `
		/* Section styles */
		.section-title {
			font-size: 1.5rem;
		}
		.section-heading {
			font-size: 1.25rem;
		}

		/* Button styles */
		.btn {
			padding: 10px 20px;
		}

		/* Text section with :has() and :not() */
		.text-section {
			a:not(:has(.btn)) {
				color: blue;
				text-decoration: underline;
			}
		}

		/* Input group with :not() */
		.input-group:not(.persistent) {
			display: flex;
			flex-direction: column;
		}

		@media screen and (max-width: 768px) {
			.input-group:not(.persistent) {
				flex-direction: column;
				gap: 15px;
			}
		}
	`)

	template := simpleParse(t, `<div class="text-section"><a class="btn">Link</a></div><div class="input-group persistent"></div><div class="layout-container"></div>`)

	css := `
@import "/test/helpers.css";

.layout-container {
	min-height: 100vh;
	display: flex;
	flex-direction: column;
}
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".btn", "Should contain .btn class")
	assert.Contains(t, normalised, ".section-heading", "Should contain .section-heading class")
	assert.Contains(t, normalised, ".persistent", "Should contain .persistent class")

	assert.Contains(t, normalised, ":not(:has(.btn))", "a:not(:has(.btn)) should preserve .btn inside :has()")

	assert.Contains(t, normalised, ":not(.persistent)", ".input-group:not(.persistent) should preserve .persistent inside :not()")

	assert.NotContains(t, normalised, ":not(:has(.section-heading))", "Should NOT have .section-heading inside :has() - it should be .btn")
	assert.NotContains(t, normalised, ":not(.input-group)", "Should NOT have .input-group inside :not() - it should be .persistent")
}

func TestCSSProcessor_SymbolReindex_SSHash_IDSelectors(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/ids.css", `
		#first-id {
			color: red;
		}
		#second-id {
			color: blue;
		}
		#third-id {
			color: green;
		}
	`)

	template := simpleParse(t, `<div id="first-id"></div><div id="second-id"></div><div id="third-id"></div>`)
	css := `
@import "/test/ids.css";

#parent-id {
	display: flex;
}
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, "#first-id", "Should contain #first-id")
	assert.Contains(t, normalised, "#second-id", "Should contain #second-id")
	assert.Contains(t, normalised, "#third-id", "Should contain #third-id")
	assert.Contains(t, normalised, "#parent-id", "Should contain #parent-id")

	assert.NotContains(t, normalised, "#parent-id{color:red}", "Parent ID should not have child's styles")
}

func TestCSSProcessor_SymbolReindex_SSClass_ClassSelectors(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/classes.css", `
		.alpha { color: red; }
		.beta { color: blue; }
		.gamma { color: green; }
		.delta { color: yellow; }
		.epsilon { color: orange; }
	`)

	template := simpleParse(t, `<div class="alpha beta gamma delta epsilon"></div>`)
	css := `
@import "/test/classes.css";

.omega { display: flex; }
.zeta { display: grid; }
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".alpha", "Should contain .alpha")
	assert.Contains(t, normalised, ".beta", "Should contain .beta")
	assert.Contains(t, normalised, ".gamma", "Should contain .gamma")
	assert.Contains(t, normalised, ".delta", "Should contain .delta")
	assert.Contains(t, normalised, ".epsilon", "Should contain .epsilon")
	assert.Contains(t, normalised, ".omega", "Should contain .omega")
	assert.Contains(t, normalised, ".zeta", "Should contain .zeta")
}

func TestCSSProcessor_SymbolReindex_SSPseudoClass_Is(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/is-selector.css", `
		.heading-one { font-size: 2rem; }
		.heading-two { font-size: 1.5rem; }

		:is(.heading-one, .heading-two) {
			font-weight: bold;
		}

		.container :is(.heading-one, .heading-two, .heading-three) {
			margin-bottom: 1rem;
		}
	`)

	template := simpleParse(t, `<div class="container"><h1 class="heading-one"></h1><h2 class="heading-two"></h2></div>`)
	css := `
@import "/test/is-selector.css";

.wrapper { display: block; }
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ":is(.heading-one", "Should contain :is(.heading-one")
	assert.Contains(t, normalised, ".heading-two", "Should contain .heading-two")
	assert.NotContains(t, normalised, ":is(.wrapper", "Should NOT corrupt :is() with parent class")
}

func TestCSSProcessor_SymbolReindex_SSPseudoClass_Where(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/where-selector.css", `
		.alert-success { background: green; }
		.alert-warning { background: yellow; }
		.alert-danger { background: red; }

		:where(.alert-success, .alert-warning, .alert-danger) {
			padding: 1rem;
			border-radius: 4px;
		}
	`)

	template := simpleParse(t, `<div class="alert-success"></div>`)
	css := `
@import "/test/where-selector.css";

.notification { margin: 1rem; }
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ":where(.alert-success", "Should contain :where(.alert-success")
	assert.Contains(t, normalised, ".alert-warning", "Should contain .alert-warning")
	assert.Contains(t, normalised, ".alert-danger", "Should contain .alert-danger")
}

func TestCSSProcessor_SymbolReindex_NestedPseudoClasses(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/nested-pseudo.css", `
		.card { border: 1px solid gray; }
		.card-header { font-weight: bold; }
		.card-body { padding: 1rem; }
		.icon { width: 24px; }

		/* Deeply nested pseudo-classes */
		.card:not(:has(.card-header)):has(.card-body) {
			padding: 2rem;
		}

		.list:has(.item:not(.disabled)) {
			opacity: 1;
		}
	`)

	template := simpleParse(t, `<div class="card"><div class="card-body"></div></div>`)
	css := `
@import "/test/nested-pseudo.css";

.wrapper { display: block; }
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".card-header", "Should contain .card-header")
	assert.Contains(t, normalised, ".card-body", "Should contain .card-body")
	assert.Contains(t, normalised, ":not(:has(.card-header))", "Should preserve :not(:has(.card-header))")
}

func TestCSSProcessor_SymbolReindex_URLTokens(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/urls.css", `
		.bg-image {
			background-image: url('/images/bg.png');
		}
		.icon-check {
			background: url('/icons/check.svg') no-repeat;
		}
	`)

	template := simpleParse(t, `<div class="bg-image"></div>`)
	css := `
@import "/test/urls.css";

.parent-bg {
	background: url('/parent.png');
}
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".bg-image", "Should contain .bg-image class")
	assert.Contains(t, normalised, ".icon-check", "Should contain .icon-check class")
	assert.Contains(t, normalised, ".parent-bg", "Should contain .parent-bg class")
}

func TestCSSProcessor_SymbolReindex_ChainedImports(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/base.css", `
		.base-one { color: #111; }
		.base-two { color: #222; }
	`)

	fsReader.addFile("/test/helpers.css", `
		@import "/test/base.css";
		.helper-one { margin: 1rem; }
		.helper-two { padding: 1rem; }
	`)

	fsReader.addFile("/test/layout.css", `
		@import "/test/helpers.css";
		.layout-one { display: flex; }
		.layout-two { display: grid; }

		/* :not() referencing classes from different levels */
		.container:not(.base-one):not(.helper-one) {
			width: 100%;
		}
	`)

	template := simpleParse(t, `<div class="base-one helper-one layout-one container"></div>`)
	css := `
@import "/test/layout.css";

.main-class { background: white; }
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".base-one", "Should contain .base-one from base.css")
	assert.Contains(t, normalised, ".base-two", "Should contain .base-two from base.css")
	assert.Contains(t, normalised, ".helper-one", "Should contain .helper-one from helpers.css")
	assert.Contains(t, normalised, ".helper-two", "Should contain .helper-two from helpers.css")
	assert.Contains(t, normalised, ".layout-one", "Should contain .layout-one from layout.css")
	assert.Contains(t, normalised, ".layout-two", "Should contain .layout-two from layout.css")
	assert.Contains(t, normalised, ".main-class", "Should contain .main-class from main")

	assert.Contains(t, normalised, ":not(.base-one)", "Should preserve :not(.base-one)")
	assert.Contains(t, normalised, ":not(.helper-one)", "Should preserve :not(.helper-one)")
}

func TestCSSProcessor_SymbolReindex_CombinedIDAndClass(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/combined.css", `
		#nav.active { background: blue; }
		#nav .nav-item.selected { font-weight: bold; }
		.container#main { max-width: 1200px; }
		#sidebar.collapsed .sidebar-item { display: none; }
	`)

	template := simpleParse(t, `<nav id="nav" class="active"><a class="nav-item selected"></a></nav>`)
	css := `
@import "/test/combined.css";

#root.loaded { opacity: 1; }
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, "#nav", "Should contain #nav")
	assert.Contains(t, normalised, ".active", "Should contain .active")
	assert.Contains(t, normalised, ".nav-item", "Should contain .nav-item")
	assert.Contains(t, normalised, ".selected", "Should contain .selected")
	assert.Contains(t, normalised, "#main", "Should contain #main")
	assert.Contains(t, normalised, "#sidebar", "Should contain #sidebar")
	assert.Contains(t, normalised, ".collapsed", "Should contain .collapsed")
}

func TestCSSProcessor_SymbolReindex_KeyframesWithAnimationProperty(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/animations.css", `
		@keyframes bounce {
			0%, 100% { transform: translateY(0); }
			50% { transform: translateY(-20px); }
		}
		@keyframes spin {
			from { transform: rotate(0deg); }
			to { transform: rotate(360deg); }
		}
		@keyframes pulse {
			0%, 100% { opacity: 1; }
			50% { opacity: 0.5; }
		}
		.bouncing { animation: bounce 1s infinite; }
		.spinning { animation: spin 2s linear infinite; }
		.pulsing { animation: pulse 1.5s ease-in-out infinite; }
	`)

	template := simpleParse(t, `<div class="bouncing spinning pulsing"></div>`)
	css := `
@import "/test/animations.css";

@keyframes slideIn {
	from { transform: translateX(-100%); }
	to { transform: translateX(0); }
}
.sliding { animation: slideIn 0.3s ease-out; }
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, "@keyframes bounce", "Should contain @keyframes bounce")
	assert.Contains(t, normalised, "@keyframes spin", "Should contain @keyframes spin")
	assert.Contains(t, normalised, "@keyframes pulse", "Should contain @keyframes pulse")
	assert.Contains(t, normalised, "@keyframes slideIn", "Should contain @keyframes slideIn")

	assert.Contains(t, normalised, "animation:bounce", "Should have animation:bounce")
	assert.Contains(t, normalised, "animation:spin", "Should have animation:spin")
	assert.Contains(t, normalised, "animation:pulse", "Should have animation:pulse")
	assert.Contains(t, normalised, "animation:slideIn", "Should have animation:slideIn")
}

func TestCSSProcessor_SymbolReindex_ComplexSelectorCombinations(t *testing.T) {
	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "test-scope"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	fsReader.addFile("/test/complex.css", `
		/* Descendant combinators */
		.parent .child { color: blue; }

		/* Child combinators */
		.container > .direct-child { margin: 0; }

		/* Adjacent sibling */
		.label + .input { margin-top: 0; }

		/* General sibling */
		.heading ~ .paragraph { margin-top: 1rem; }

		/* Multiple classes */
		.btn.primary.large { padding: 2rem; }

		/* Attribute with class */
		.input[type="text"].focused { border-color: blue; }
	`)

	template := simpleParse(t, `<div class="parent"><div class="child"></div></div>`)
	css := `
@import "/test/complex.css";

.wrapper .inner { display: flex; }
`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Log("Normalised CSS output:", normalised)

	assert.Contains(t, normalised, ".parent", "Should contain .parent")
	assert.Contains(t, normalised, ".child", "Should contain .child")
	assert.Contains(t, normalised, ".container", "Should contain .container")
	assert.Contains(t, normalised, ".direct-child", "Should contain .direct-child")
	assert.Contains(t, normalised, ".label", "Should contain .label")
	assert.Contains(t, normalised, ".input", "Should contain .input")
	assert.Contains(t, normalised, ".btn", "Should contain .btn")
	assert.Contains(t, normalised, ".primary", "Should contain .primary")
	assert.Contains(t, normalised, ".large", "Should contain .large")
	assert.Contains(t, normalised, ".focused", "Should contain .focused")
}

func TestNewCSSProcessor(t *testing.T) {
	t.Parallel()

	t.Run("creates processor with nil config uses empty defaults", func(t *testing.T) {
		t.Parallel()

		resolver := newTestResolver()
		processor := NewCSSProcessor(config.LoaderLocalCSS, nil, resolver)

		require.NotNil(t, processor)
		require.NotNil(t, processor.processor)
		require.NotNil(t, processor.processor.GetOptions())
		assert.Equal(t, resolver, processor.processor.GetResolver())
	})

	t.Run("creates processor with provided config", func(t *testing.T) {
		t.Parallel()

		resolver := newTestResolver()
		options := &config.Options{
			MinifyWhitespace: true,
			ASCIIOnly:        true,
		}
		processor := NewCSSProcessor(config.LoaderLocalCSS, options, resolver)

		require.NotNil(t, processor)
		assert.Same(t, options, processor.processor.GetOptions())
		assert.True(t, processor.processor.GetOptions().MinifyWhitespace)
		assert.True(t, processor.processor.GetOptions().ASCIIOnly)
	})

	t.Run("creates processor with nil resolver", func(t *testing.T) {
		t.Parallel()

		processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{}, nil)

		require.NotNil(t, processor)
		assert.Nil(t, processor.processor.GetResolver())
	})
}

func TestCSSProcessor_SetResolver(t *testing.T) {
	t.Parallel()

	t.Run("updates resolver on processor", func(t *testing.T) {
		t.Parallel()

		processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{}, nil)
		assert.Nil(t, processor.processor.GetResolver())

		newResolver := newTestResolver()
		processor.SetResolver(newResolver)

		assert.Equal(t, newResolver, processor.processor.GetResolver())
	})

	t.Run("replaces existing resolver", func(t *testing.T) {
		t.Parallel()

		originalResolver := newTestResolver()
		processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{}, originalResolver)
		assert.Equal(t, originalResolver, processor.processor.GetResolver())

		replacementResolver := newTestResolver()
		processor.SetResolver(replacementResolver)

		assert.Equal(t, replacementResolver, processor.processor.GetResolver())
		assert.NotEqual(t, originalResolver, replacementResolver)
	})

	t.Run("can set resolver to nil", func(t *testing.T) {
		t.Parallel()

		resolver := newTestResolver()
		processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{}, resolver)

		processor.SetResolver(nil)

		assert.Nil(t, processor.processor.GetResolver())
	})
}

func TestConvertESBuildMessagesToDiagnostics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		checkDiags    func(t *testing.T, diagnostics []*ast_domain.Diagnostic)
		name          string
		sourcePath    string
		messages      []es_logger.Msg
		startLocation ast_domain.Location
		expectedLen   int
	}{
		{
			name:        "returns nil for empty messages",
			messages:    []es_logger.Msg{},
			sourcePath:  "test.css",
			expectedLen: 0,
			checkDiags: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				assert.Nil(t, diagnostics)
			},
		},
		{
			name:        "returns nil for nil messages",
			messages:    nil,
			sourcePath:  "test.css",
			expectedLen: 0,
			checkDiags: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				assert.Nil(t, diagnostics)
			},
		},
		{
			name: "converts error message without location",
			messages: []es_logger.Msg{
				{
					Kind: es_logger.Error,
					Data: es_logger.MsgData{
						Text:     "unexpected token",
						Location: nil,
					},
				},
			},
			sourcePath:    "style.css",
			startLocation: ast_domain.Location{Line: 5, Column: 10},
			expectedLen:   1,
			checkDiags: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
				assert.Equal(t, "unexpected token", diagnostics[0].Message)
				assert.Equal(t, "style.css", diagnostics[0].SourcePath)
				assert.Equal(t, 0, diagnostics[0].Location.Line)
				assert.Equal(t, 0, diagnostics[0].Location.Column)
			},
		},
		{
			name: "converts warning message with location on first line",
			messages: []es_logger.Msg{
				{
					Kind: es_logger.Warning,
					Data: es_logger.MsgData{
						Text: "unknown property",
						Location: &es_logger.MsgLocation{
							Line:     0,
							Column:   5,
							LineText: ".class { unknwn: value; }",
						},
					},
				},
			},
			sourcePath:    "test.css",
			startLocation: ast_domain.Location{Line: 10, Column: 3},
			expectedLen:   1,
			checkDiags: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
				assert.Equal(t, "unknown property", diagnostics[0].Message)

				assert.Equal(t, 10, diagnostics[0].Location.Line)

				assert.Equal(t, 8, diagnostics[0].Location.Column)
				assert.Equal(t, ".class { unknwn: value; }", diagnostics[0].Expression)
			},
		},
		{
			name: "converts info message with location on subsequent line",
			messages: []es_logger.Msg{
				{
					Kind: es_logger.Info,
					Data: es_logger.MsgData{
						Text: "CSS info message",
						Location: &es_logger.MsgLocation{
							Line:     4,
							Column:   2,
							LineText: "  color: red;",
						},
					},
				},
			},
			sourcePath:    "test.css",
			startLocation: ast_domain.Location{Line: 20, Column: 1},
			expectedLen:   1,
			checkDiags: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				assert.Equal(t, ast_domain.Info, diagnostics[0].Severity)

				assert.Equal(t, 24, diagnostics[0].Location.Line)

				assert.Equal(t, 3, diagnostics[0].Location.Column)
			},
		},
		{
			name: "skips Note kind messages",
			messages: []es_logger.Msg{
				{
					Kind: es_logger.Note,
					Data: es_logger.MsgData{
						Text: "a note message",
					},
				},
			},
			sourcePath:  "test.css",
			expectedLen: 0,
			checkDiags: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)
			},
		},
		{
			name: "converts multiple messages of different kinds",
			messages: []es_logger.Msg{
				{
					Kind: es_logger.Error,
					Data: es_logger.MsgData{Text: "error one"},
				},
				{
					Kind: es_logger.Warning,
					Data: es_logger.MsgData{Text: "warning one"},
				},
				{
					Kind: es_logger.Note,
					Data: es_logger.MsgData{Text: "note skipped"},
				},
				{
					Kind: es_logger.Error,
					Data: es_logger.MsgData{Text: "error two"},
				},
			},
			sourcePath:  "multi.css",
			expectedLen: 3,
			checkDiags: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
				assert.Equal(t, "error one", diagnostics[0].Message)
				assert.Equal(t, ast_domain.Warning, diagnostics[1].Severity)
				assert.Equal(t, "warning one", diagnostics[1].Message)
				assert.Equal(t, ast_domain.Error, diagnostics[2].Severity)
				assert.Equal(t, "error two", diagnostics[2].Message)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			diagnostics := cssinliner.ConvertESBuildMessagesToDiagnostics(tc.messages, tc.sourcePath, tc.startLocation, "")

			if tc.expectedLen == 0 {
				tc.checkDiags(t, diagnostics)
			} else {
				require.Len(t, diagnostics, tc.expectedLen)
				tc.checkDiags(t, diagnostics)
			}
		})
	}
}

func TestScopeDirect(t *testing.T) {
	t.Parallel()

	scopeID := "p-test123"

	t.Run("returns empty selector unchanged", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{Selectors: nil}
		result := scopeDirect(selector, scopeID)

		assert.Empty(t, result.Selectors)
	})

	t.Run("returns empty slice of selectors unchanged", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{Selectors: []css_ast.CompoundSelector{}}
		result := scopeDirect(selector, scopeID)

		assert.Empty(t, result.Selectors)
	})

	t.Run("scopes a simple type selector", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "div"},
					},
				},
			},
		}
		result := scopeDirect(selector, scopeID)

		require.Len(t, result.Selectors, 1)

		assert.NotEmpty(t, result.Selectors[0].SubclassSelectors)
		attr, ok := result.Selectors[0].SubclassSelectors[0].Data.(*css_ast.SSAttribute)
		require.True(t, ok)
		assert.Equal(t, "partial", attr.NamespacedName.Name.Text)
		assert.Equal(t, "~=", attr.MatcherOp)
		assert.Equal(t, scopeID, attr.MatcherValue)
	})

	t.Run("skips html element", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "html"},
					},
				},
			},
		}
		result := scopeDirect(selector, scopeID)

		require.Len(t, result.Selectors, 1)

		assert.Empty(t, result.Selectors[0].SubclassSelectors)
	})

	t.Run("skips body element", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "body"},
					},
				},
			},
		}
		result := scopeDirect(selector, scopeID)

		require.Len(t, result.Selectors, 1)
		assert.Empty(t, result.Selectors[0].SubclassSelectors)
	})

	t.Run("skips selector with :root pseudo-class", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					SubclassSelectors: []css_ast.SubclassSelector{
						{Data: &css_ast.SSPseudoClass{Name: "root"}},
					},
				},
			},
		}
		result := scopeDirect(selector, scopeID)

		require.Len(t, result.Selectors, 1)

		require.Len(t, result.Selectors[0].SubclassSelectors, 1)
		pseudo, ok := result.Selectors[0].SubclassSelectors[0].Data.(*css_ast.SSPseudoClass)
		require.True(t, ok)
		assert.Equal(t, "root", pseudo.Name)
	})

	t.Run("scopes multiple compound selectors in chain", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "div"},
					},
				},
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "p"},
					},
				},
			},
		}
		result := scopeDirect(selector, scopeID)

		require.Len(t, result.Selectors, 2)

		assert.NotEmpty(t, result.Selectors[0].SubclassSelectors)
		assert.NotEmpty(t, result.Selectors[1].SubclassSelectors)
	})

	t.Run("scopes non-skipped selectors and skips body in chain", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "body"},
					},
				},
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "div"},
					},
				},
			},
		}
		result := scopeDirect(selector, scopeID)

		require.Len(t, result.Selectors, 2)

		assert.Empty(t, result.Selectors[0].SubclassSelectors)

		assert.NotEmpty(t, result.Selectors[1].SubclassSelectors)
	})
}

func TestShouldSkipCompoundScoping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		comp     *css_ast.CompoundSelector
		name     string
		expected bool
	}{
		{
			name: "skips html type selector",
			comp: &css_ast.CompoundSelector{
				TypeSelector: &css_ast.NamespacedName{
					Name: css_ast.NameToken{Text: "html"},
				},
			},
			expected: true,
		},
		{
			name: "skips HTML uppercase type selector",
			comp: &css_ast.CompoundSelector{
				TypeSelector: &css_ast.NamespacedName{
					Name: css_ast.NameToken{Text: "HTML"},
				},
			},
			expected: true,
		},
		{
			name: "skips body type selector",
			comp: &css_ast.CompoundSelector{
				TypeSelector: &css_ast.NamespacedName{
					Name: css_ast.NameToken{Text: "body"},
				},
			},
			expected: true,
		},
		{
			name: "skips BODY uppercase type selector",
			comp: &css_ast.CompoundSelector{
				TypeSelector: &css_ast.NamespacedName{
					Name: css_ast.NameToken{Text: "BODY"},
				},
			},
			expected: true,
		},
		{
			name: "skips :root pseudo-class",
			comp: &css_ast.CompoundSelector{
				SubclassSelectors: []css_ast.SubclassSelector{
					{Data: &css_ast.SSPseudoClass{Name: "root"}},
				},
			},
			expected: true,
		},
		{
			name: "skips :ROOT case-insensitive pseudo-class",
			comp: &css_ast.CompoundSelector{
				SubclassSelectors: []css_ast.SubclassSelector{
					{Data: &css_ast.SSPseudoClass{Name: "ROOT"}},
				},
			},
			expected: true,
		},
		{
			name: "does not skip regular type selector",
			comp: &css_ast.CompoundSelector{
				TypeSelector: &css_ast.NamespacedName{
					Name: css_ast.NameToken{Text: "div"},
				},
			},
			expected: false,
		},
		{
			name: "does not skip non-root pseudo-class",
			comp: &css_ast.CompoundSelector{
				SubclassSelectors: []css_ast.SubclassSelector{
					{Data: &css_ast.SSPseudoClass{Name: "hover"}},
				},
			},
			expected: false,
		},
		{
			name: "does not skip selector with attribute data",
			comp: &css_ast.CompoundSelector{
				SubclassSelectors: []css_ast.SubclassSelector{
					{Data: &css_ast.SSAttribute{
						NamespacedName: css_ast.NamespacedName{
							Name: css_ast.NameToken{Text: "data-type"},
						},
					}},
				},
			},
			expected: false,
		},
		{
			name:     "does not skip empty compound selector",
			comp:     &css_ast.CompoundSelector{},
			expected: false,
		},
		{
			name: "does not skip selector with nil type selector and non-root pseudo-classes",
			comp: &css_ast.CompoundSelector{
				TypeSelector: nil,
				SubclassSelectors: []css_ast.SubclassSelector{
					{Data: &css_ast.SSPseudoClass{Name: "first-child"}},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := shouldSkipCompoundScoping(tc.comp)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRemoveMarkerClass(t *testing.T) {
	t.Parallel()

	t.Run("returns original selector when location is nil", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "div"},
					},
				},
			},
		}

		result := removeMarkerClass(selector, nil)
		assert.Equal(t, 1, len(result.Selectors))
	})

	t.Run("returns cloned selector when compound index is out of bounds", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "div"},
					},
				},
			},
		}

		loc := &markerLocation{compoundIndex: 5, subclassIndex: 0}
		result := removeMarkerClass(selector, loc)
		assert.Equal(t, 1, len(result.Selectors))
	})

	t.Run("returns cloned selector when subclass index is out of bounds", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					SubclassSelectors: []css_ast.SubclassSelector{
						{Data: &css_ast.SSPseudoClass{Name: "hover"}},
					},
				},
			},
		}

		loc := &markerLocation{compoundIndex: 0, subclassIndex: 5}
		result := removeMarkerClass(selector, loc)
		require.Len(t, result.Selectors, 1)

		assert.Len(t, result.Selectors[0].SubclassSelectors, 1)
	})

	t.Run("removes marker at specific position", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					SubclassSelectors: []css_ast.SubclassSelector{
						{Data: &css_ast.SSPseudoClass{Name: "first"}},
						{Data: &css_ast.SSPseudoClass{Name: "marker"}},
						{Data: &css_ast.SSPseudoClass{Name: "last"}},
					},
				},
			},
		}

		loc := &markerLocation{compoundIndex: 0, subclassIndex: 1}
		result := removeMarkerClass(selector, loc)
		require.Len(t, result.Selectors, 1)
		require.Len(t, result.Selectors[0].SubclassSelectors, 2)
		first, ok := result.Selectors[0].SubclassSelectors[0].Data.(*css_ast.SSPseudoClass)
		require.True(t, ok)
		assert.Equal(t, "first", first.Name)
		last, ok := result.Selectors[0].SubclassSelectors[1].Data.(*css_ast.SSPseudoClass)
		require.True(t, ok)
		assert.Equal(t, "last", last.Name)
	})

	t.Run("removes only marker from single subclass list", func(t *testing.T) {
		t.Parallel()

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					SubclassSelectors: []css_ast.SubclassSelector{
						{Data: &css_ast.SSPseudoClass{Name: "onlyMarker"}},
					},
				},
			},
		}

		loc := &markerLocation{compoundIndex: 0, subclassIndex: 0}
		result := removeMarkerClass(selector, loc)
		require.Len(t, result.Selectors, 1)
		assert.Empty(t, result.Selectors[0].SubclassSelectors)
	})
}

func TestExtractRootDescriptors(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil template", func(t *testing.T) {
		t.Parallel()

		result := extractRootDescriptors(nil)
		assert.Nil(t, result)
	})

	t.Run("returns nil for template with no root nodes", func(t *testing.T) {
		t.Parallel()

		template := &ast_domain.TemplateAST{RootNodes: nil}
		result := extractRootDescriptors(template)
		assert.Nil(t, result)
	})

	t.Run("returns nil for template with empty root nodes", func(t *testing.T) {
		t.Parallel()

		template := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{}}
		result := extractRootDescriptors(template)
		assert.Nil(t, result)
	})

	t.Run("extracts descriptors from element nodes only", func(t *testing.T) {
		t.Parallel()

		template := simpleParse(t, `<div class="container" id="app"><p>Text</p></div>`)
		result := extractRootDescriptors(template)

		require.NotNil(t, result)
		require.Len(t, result, 1)
		assert.Equal(t, "div", result[0].tag)
		assert.Equal(t, "app", result[0].id)
		assert.Contains(t, result[0].classes, "container")
	})

	t.Run("extracts multiple root element descriptors", func(t *testing.T) {
		t.Parallel()

		template := simpleParse(t, `<header class="top"></header><main id="content"></main>`)
		result := extractRootDescriptors(template)

		require.NotNil(t, result)
		require.Len(t, result, 2)
		assert.Equal(t, "header", result[0].tag)
		assert.Contains(t, result[0].classes, "top")
		assert.Equal(t, "main", result[1].tag)
		assert.Equal(t, "content", result[1].id)
	})

	t.Run("extracts multiple CSS classes from class attribute", func(t *testing.T) {
		t.Parallel()

		template := simpleParse(t, `<div class="foo bar baz"></div>`)
		result := extractRootDescriptors(template)

		require.Len(t, result, 1)
		assert.Contains(t, result[0].classes, "foo")
		assert.Contains(t, result[0].classes, "bar")
		assert.Contains(t, result[0].classes, "baz")
	})
}

func TestHasDynamicClasses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name: "returns true when DirClass is set",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				DirClass: &ast_domain.Directive{},
			},
			expected: true,
		},
		{
			name: "returns true when dynamic attribute named class exists",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "class"},
				},
			},
			expected: true,
		},
		{
			name: "returns true when dynamic attribute named CLASS exists (case-insensitive)",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "CLASS"},
				},
			},
			expected: true,
		},
		{
			name: "returns false when no dynamic class bindings",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
			},
			expected: false,
		},
		{
			name: "returns false when dynamic attribute is not class",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "style"},
					{Name: "title"},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasDynamicClasses(tc.node)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHandleDeepMarker(t *testing.T) {
	t.Parallel()

	scopeID := "p-deep-test"

	t.Run("deep marker at index 0 uses descendant scoping", func(t *testing.T) {
		t.Parallel()

		template := simpleParse(t, `<div></div>`)
		transformer := newCSSScopeTransformer(scopeID, template, nil)

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{

					SubclassSelectors: []css_ast.SubclassSelector{
						{Data: &css_ast.SSPseudoClass{Name: "someClass"}},
					},
				},
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "span"},
					},
				},
			},
		}
		marker := &markerLocation{compoundIndex: 0, subclassIndex: 0}

		var scoped []css_ast.ComplexSelector
		result := transformer.handleDeepMarker(scoped, selector, marker, es_logger.Loc{})

		require.Len(t, result, 1)

		assert.GreaterOrEqual(t, len(result[0].Selectors), 2)
	})

	t.Run("deep marker at non-zero index scopes preceding compounds", func(t *testing.T) {
		t.Parallel()

		template := simpleParse(t, `<div></div>`)
		transformer := newCSSScopeTransformer(scopeID, template, nil)

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "div"},
					},
				},
				{

					SubclassSelectors: []css_ast.SubclassSelector{
						{Data: &css_ast.SSPseudoClass{Name: "deepMarker"}},
					},
				},
				{
					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "span"},
					},
				},
			},
		}
		marker := &markerLocation{compoundIndex: 1, subclassIndex: 0}

		var scoped []css_ast.ComplexSelector
		result := transformer.handleDeepMarker(scoped, selector, marker, es_logger.Loc{})

		require.Len(t, result, 1)

		require.NotEmpty(t, result[0].Selectors[0].SubclassSelectors)
		attr, ok := result[0].Selectors[0].SubclassSelectors[0].Data.(*css_ast.SSAttribute)
		require.True(t, ok)
		assert.Equal(t, "partial", attr.NamespacedName.Name.Text)
		assert.Equal(t, scopeID, attr.MatcherValue)
	})

	t.Run("deep marker with body in preceding compounds skips body scoping", func(t *testing.T) {
		t.Parallel()

		template := simpleParse(t, `<div></div>`)
		transformer := newCSSScopeTransformer(scopeID, template, nil)

		selector := css_ast.ComplexSelector{
			Selectors: []css_ast.CompoundSelector{
				{

					TypeSelector: &css_ast.NamespacedName{
						Name: css_ast.NameToken{Text: "body"},
					},
				},
				{

					SubclassSelectors: []css_ast.SubclassSelector{
						{Data: &css_ast.SSPseudoClass{Name: "marker"}},
					},
				},
			},
		}
		marker := &markerLocation{compoundIndex: 1, subclassIndex: 0}

		var scoped []css_ast.ComplexSelector
		result := transformer.handleDeepMarker(scoped, selector, marker, es_logger.Loc{})

		require.Len(t, result, 1)

		assert.Empty(t, result[0].Selectors[0].SubclassSelectors)
	})
}

func TestTransformKnownAtRule(t *testing.T) {
	t.Parallel()

	t.Run("transforms nested rules in known at-rule", func(t *testing.T) {
		t.Parallel()

		scopeID := "p-at-rule"
		template := simpleParse(t, `<div></div>`)
		transformer := newCSSScopeTransformer(scopeID, template, nil)

		data := &css_ast.RKnownAt{
			AtToken: "supports",
			Rules: []css_ast.Rule{
				{
					Data: &css_ast.RSelector{
						Selectors: []css_ast.ComplexSelector{
							{
								Selectors: []css_ast.CompoundSelector{
									{
										TypeSelector: &css_ast.NamespacedName{
											Name: css_ast.NameToken{Text: "div"},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		transformer.transformKnownAtRule(data)

		require.Len(t, data.Rules, 1)
		selectorRule, ok := data.Rules[0].Data.(*css_ast.RSelector)
		require.True(t, ok)
		require.Len(t, selectorRule.Selectors, 1)
		require.Len(t, selectorRule.Selectors[0].Selectors, 1)
		assert.NotEmpty(t, selectorRule.Selectors[0].Selectors[0].SubclassSelectors)
	})

	t.Run("tracks keyframes depth for keyframes at-token", func(t *testing.T) {
		t.Parallel()

		scopeID := "p-kf"
		template := simpleParse(t, `<div></div>`)
		transformer := newCSSScopeTransformer(scopeID, template, nil)

		assert.Equal(t, 0, transformer.keyframesDepth)

		data := &css_ast.RKnownAt{
			AtToken: "keyframes",
			Rules: []css_ast.Rule{
				{
					Data: &css_ast.RSelector{
						Selectors: []css_ast.ComplexSelector{
							{
								Selectors: []css_ast.CompoundSelector{
									{
										TypeSelector: &css_ast.NamespacedName{
											Name: css_ast.NameToken{Text: "div"},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		transformer.transformKnownAtRule(data)

		assert.Equal(t, 0, transformer.keyframesDepth)

		selectorRule, ok := data.Rules[0].Data.(*css_ast.RSelector)
		require.True(t, ok)
		require.Len(t, selectorRule.Selectors, 1)

		assert.Empty(t, selectorRule.Selectors[0].Selectors[0].SubclassSelectors)
	})

	t.Run("handles nil rules gracefully", func(t *testing.T) {
		t.Parallel()

		scopeID := "p-nil"
		template := simpleParse(t, `<div></div>`)
		transformer := newCSSScopeTransformer(scopeID, template, nil)

		data := &css_ast.RKnownAt{
			AtToken: "supports",
			Rules:   nil,
		}

		transformer.transformKnownAtRule(data)
		assert.Nil(t, data.Rules)
	})
}

func TestPreprocessScopingPseudoClasses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		input        string
		expectedCSS  string
		expectGlobal bool
		expectDeep   bool
	}{
		{
			name:         "no pseudo-classes returns input unchanged",
			input:        ".container { color: red; }",
			expectedCSS:  ".container { color: red; }",
			expectGlobal: false,
			expectDeep:   false,
		},
		{
			name:         "replaces :global() with marker class",
			input:        ":global(.modal) { display: block; }",
			expectedCSS:  ".modal.__piko_global__ { display: block; }",
			expectGlobal: true,
			expectDeep:   false,
		},
		{
			name:         "replaces :deep() with marker class",
			input:        ":deep(.child) { color: blue; }",
			expectedCSS:  ".__piko_deep__ .child { color: blue; }",
			expectGlobal: false,
			expectDeep:   true,
		},
		{
			name:         "replaces both :global() and :deep()",
			input:        ":global(.a) { } :deep(.b) { }",
			expectedCSS:  ".a.__piko_global__ { } .__piko_deep__ .b { }",
			expectGlobal: true,
			expectDeep:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, markers := preprocessScopingPseudoClasses(tc.input)

			assert.Equal(t, tc.expectedCSS, result)
			assert.Equal(t, tc.expectGlobal, markers.hasGlobal)
			assert.Equal(t, tc.expectDeep, markers.hasDeep)
		})
	}
}

func TestCSSProcessor_WithResolver(t *testing.T) {
	t.Parallel()

	t.Run("returns a new processor with given resolver", func(t *testing.T) {
		t.Parallel()

		options := &config.Options{}
		original := NewCSSProcessor(config.LoaderCSS, options, nil)

		newProcessor := original.WithResolver(nil)

		require.NotNil(t, newProcessor)
		assert.NotSame(t, original, newProcessor)
	})
}

func TestCSSProcessor_RejectsDeeplyNestedSelectors(t *testing.T) {
	t.Parallel()

	current := []css_ast.Rule{}
	for range maxCSSRuleDepth + 16 {
		nested := []css_ast.Rule{
			{Data: &css_ast.RAtLayer{Names: [][]string{{"piko"}}, Rules: current}},
		}
		current = nested
	}

	transformer := &cssScopeTransformer{scopeID: "scope"}

	done := make(chan struct{})
	go func() {
		defer close(done)
		transformer.transform(current)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("transform did not return promptly - depth cap missing or broken")
	}
}
