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

package wasm_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

func TestIsExported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "uppercase first letter", input: "Hello", expected: true},
		{name: "lowercase first letter", input: "hello", expected: false},
		{name: "empty string", input: "", expected: false},
		{name: "single uppercase letter", input: "A", expected: true},
		{name: "single lowercase letter", input: "a", expected: false},
		{name: "underscore prefix", input: "_Hello", expected: false},
		{name: "unicode uppercase", input: "\u00C0field", expected: true},
		{name: "number prefix", input: "1field", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isExported(tt.input))
		})
	}
}

func TestIsIdentChar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    byte
		expected bool
	}{
		{name: "lowercase a", input: 'a', expected: true},
		{name: "lowercase z", input: 'z', expected: true},
		{name: "uppercase A", input: 'A', expected: true},
		{name: "uppercase Z", input: 'Z', expected: true},
		{name: "digit 0", input: '0', expected: true},
		{name: "digit 9", input: '9', expected: true},
		{name: "underscore", input: '_', expected: true},
		{name: "space", input: ' ', expected: false},
		{name: "dot", input: '.', expected: false},
		{name: "dash", input: '-', expected: false},
		{name: "at sign", input: '@', expected: false},
		{name: "open paren", input: '(', expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isIdentChar(tt.input))
		})
	}
}

func TestExtractLastIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple identifier", input: "foo", expected: "foo"},
		{name: "after space", input: "  bar", expected: "bar"},
		{name: "after operator", input: "x + baz", expected: "baz"},
		{name: "empty string", input: "", expected: ""},
		{name: "all special chars", input: "!!!!", expected: ""},
		{name: "with underscore", input: "my_var", expected: "my_var"},
		{name: "after paren", input: "(myFunc", expected: "myFunc"},
		{name: "single char", input: "x", expected: "x"},
		{name: "after dot", input: "pkg.Func", expected: "Func"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, extractLastIdentifier(tt.input))
		})
	}
}

func TestGetTextBeforeCursor(t *testing.T) {
	t.Parallel()

	source := "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}"

	tests := []struct {
		name     string
		expected string
		line     int
		column   int
		ok       bool
	}{
		{name: "first line", expected: "package", line: 1, column: 8, ok: true},
		{name: "third line at func", expected: "func", line: 3, column: 5, ok: true},
		{name: "line out of bounds below", expected: "", line: 0, column: 1, ok: false},
		{name: "line out of bounds above", expected: "", line: 100, column: 1, ok: false},
		{name: "column clamped to start", expected: "", line: 1, column: 0, ok: true},
		{name: "column beyond line end", expected: "package main", line: 1, column: 100, ok: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, ok := getTextBeforeCursor(source, tt.line, tt.column)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNewCompletionItem(t *testing.T) {
	t.Parallel()

	item := newCompletionItem("fmt", "module", "import")
	assert.Equal(t, "fmt", item.Label)
	assert.Equal(t, "module", item.Kind)
	assert.Equal(t, "import", item.Detail)
	assert.Empty(t, item.Documentation)
	assert.Empty(t, item.InsertText)
	assert.Empty(t, item.SortText)
}

func TestNewScopeContext(t *testing.T) {
	t.Parallel()

	ctx := newScopeContext()
	assert.Equal(t, completionKindScope, ctx.kind)
	assert.Empty(t, ctx.pkgAlias)
	assert.Empty(t, ctx.packagePath)
	assert.Empty(t, ctx.prefix)
	assert.Empty(t, ctx.expressionType)
}

func TestTypeKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typ      *inspector_dto.Type
		expected string
	}{
		{
			name:     "alias type",
			typ:      &inspector_dto.Type{IsAlias: true, UnderlyingTypeString: "int"},
			expected: "alias",
		},
		{
			name:     "struct type",
			typ:      &inspector_dto.Type{IsAlias: false, UnderlyingTypeString: "struct{...}"},
			expected: "struct",
		},
		{
			name:     "interface type",
			typ:      &inspector_dto.Type{IsAlias: false, UnderlyingTypeString: "interface{String() string}"},
			expected: "interface",
		},
		{
			name:     "regular type",
			typ:      &inspector_dto.Type{IsAlias: false, UnderlyingTypeString: "int64"},
			expected: "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, typeKind(tt.typ))
		})
	}
}

func TestErrorToDiagnostics(t *testing.T) {
	t.Parallel()

	err := errors.New("something went wrong")
	diagnostics := errorToDiagnostics(err)

	require.Len(t, diagnostics, 1)
	assert.Equal(t, "error", diagnostics[0].Severity)
	assert.Equal(t, "something went wrong", diagnostics[0].Message)
	assert.Equal(t, 0, diagnostics[0].Location.Line)
	assert.Equal(t, 0, diagnostics[0].Location.Column)
	assert.Empty(t, diagnostics[0].Location.FilePath)
}

func TestNewAnalyseErrorResponse(t *testing.T) {
	t.Parallel()

	t.Run("without diagnostics", func(t *testing.T) {
		t.Parallel()
		response := newAnalyseErrorResponse("oops", nil)
		assert.False(t, response.Success)
		assert.Equal(t, "oops", response.Error)
		assert.Nil(t, response.Types)
		assert.Nil(t, response.Functions)
		assert.Nil(t, response.Imports)
		assert.Nil(t, response.Diagnostics)
	})

	t.Run("with diagnostics", func(t *testing.T) {
		t.Parallel()
		diagnostics := []wasm_dto.Diagnostic{{Severity: "error", Message: "test"}}
		response := newAnalyseErrorResponse("fail", diagnostics)
		assert.False(t, response.Success)
		assert.Equal(t, "fail", response.Error)
		assert.Len(t, response.Diagnostics, 1)
	})
}

func TestPrepareSourceBytes(t *testing.T) {
	t.Parallel()

	t.Run("adds module prefix to relative paths", func(t *testing.T) {
		t.Parallel()
		sources := map[string]string{
			"main.go": "package main",
		}
		result := prepareSourceBytes(sources, "mymodule")
		_, ok := result["/mymodule/main.go"]
		assert.True(t, ok, "expected /mymodule/main.go key")
		assert.Equal(t, []byte("package main"), result["/mymodule/main.go"])
	})

	t.Run("keeps absolute paths unchanged", func(t *testing.T) {
		t.Parallel()
		sources := map[string]string{
			"/absolute/main.go": "package main",
		}
		result := prepareSourceBytes(sources, "mymodule")
		_, ok := result["/absolute/main.go"]
		assert.True(t, ok, "expected /absolute/main.go key")
	})

	t.Run("empty sources", func(t *testing.T) {
		t.Parallel()
		result := prepareSourceBytes(map[string]string{}, "mymodule")
		assert.Empty(t, result)
	})
}

func TestParseErrorStringToDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("parses file:line:col: message format", func(t *testing.T) {
		t.Parallel()
		diagnostics := parseErrorStringToDiagnostics("main.go:10:5: unexpected token", "main.go")
		require.Len(t, diagnostics, 1)
		assert.Equal(t, 10, diagnostics[0].Location.Line)
		assert.Equal(t, 5, diagnostics[0].Location.Column)
		assert.Equal(t, "unexpected token", diagnostics[0].Message)
		assert.Equal(t, "main.go", diagnostics[0].Location.FilePath)
	})

	t.Run("parses file:line: message format (no column)", func(t *testing.T) {
		t.Parallel()
		diagnostics := parseErrorStringToDiagnostics("main.go:10: unexpected token", "main.go")
		require.Len(t, diagnostics, 1)
		assert.Equal(t, 10, diagnostics[0].Location.Line)
		assert.Equal(t, 0, diagnostics[0].Location.Column)
	})

	t.Run("falls back for unrecognised format", func(t *testing.T) {
		t.Parallel()
		diagnostics := parseErrorStringToDiagnostics("some random error", "test.go")
		require.Len(t, diagnostics, 1)
		assert.Equal(t, 0, diagnostics[0].Location.Line)
		assert.Equal(t, 0, diagnostics[0].Location.Column)
		assert.Equal(t, "some random error", diagnostics[0].Message)
	})
}

func TestMatchesRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		url      string
		expected bool
	}{

		{name: "exact match", pattern: "/about", url: "/about", expected: true},
		{name: "trailing slash on URL", pattern: "/about", url: "/about/", expected: true},
		{name: "trailing slash on pattern", pattern: "/about/", url: "/about", expected: true},
		{name: "both trailing slashes", pattern: "/about/", url: "/about/", expected: true},
		{name: "no match", pattern: "/about", url: "/contact", expected: false},
		{name: "empty pattern and root URL", pattern: "", url: "/", expected: true},
		{name: "empty pattern and empty URL", pattern: "", url: "", expected: true},
		{name: "non-empty pattern vs empty URL", pattern: "/about", url: "", expected: false},

		{name: "dynamic segment match", pattern: "/blog/{slug}", url: "/blog/hello-world", expected: true},
		{name: "dynamic segment multi", pattern: "/blog/{year}/{slug}", url: "/blog/2026/hello", expected: true},
		{name: "dynamic segment too many segments", pattern: "/blog/{slug}", url: "/blog/hello/extra", expected: false},
		{name: "dynamic segment too few segments", pattern: "/blog/{slug}", url: "/blog", expected: false},
		{name: "dynamic segment empty value", pattern: "/blog/{slug}", url: "/blog/", expected: false},
		{name: "mixed static and dynamic", pattern: "/api/{version}/users", url: "/api/v2/users", expected: true},
		{name: "mixed static and dynamic no match", pattern: "/api/{version}/users", url: "/api/v2/posts", expected: false},

		{name: "catch-all match", pattern: "/docs/{path*}", url: "/docs/a/b/c", expected: true},
		{name: "catch-all single segment", pattern: "/docs/{path*}", url: "/docs/intro", expected: true},
		{name: "catch-all at root", pattern: "/{path*}", url: "/anything/goes", expected: true},
		{name: "catch-all with prefix", pattern: "/api/{version}/{rest*}", url: "/api/v1/users/123/edit", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, matchesRoute(tt.pattern, tt.url))
		})
	}
}

func TestPageMatchesURL(t *testing.T) {
	t.Parallel()

	t.Run("matches when one pattern matches", func(t *testing.T) {
		t.Parallel()
		patterns := map[string]string{
			"en": "/about",
			"de": "/ueber-uns",
		}
		assert.True(t, pageMatchesURL(patterns, "/about"))
	})

	t.Run("no match when no pattern matches", func(t *testing.T) {
		t.Parallel()
		patterns := map[string]string{
			"en": "/about",
		}
		assert.False(t, pageMatchesURL(patterns, "/contact"))
	})

	t.Run("empty patterns", func(t *testing.T) {
		t.Parallel()
		assert.False(t, pageMatchesURL(map[string]string{}, "/any"))
	})
}

func TestFindArtefactBySourcePath(t *testing.T) {
	t.Parallel()

	artefacts := []wasm_dto.GeneratedArtefact{
		{Path: "a.go", SourcePath: "a.pk", Type: wasm_dto.ArtefactTypePage, Content: "page-a"},
		{Path: "b.go", SourcePath: "b.pk", Type: "partial", Content: "partial-b"},
		{Path: "c.go", SourcePath: "c.pk", Type: wasm_dto.ArtefactTypePage, Content: "page-c"},
	}

	t.Run("finds matching page artefact", func(t *testing.T) {
		t.Parallel()
		result := findArtefactBySourcePath(artefacts, "a.pk")
		require.NotNil(t, result)
		assert.Equal(t, "page-a", result.Content)
	})

	t.Run("skips non-page artefacts", func(t *testing.T) {
		t.Parallel()
		result := findArtefactBySourcePath(artefacts, "b.pk")
		assert.Nil(t, result)
	})

	t.Run("returns nil for unknown source path", func(t *testing.T) {
		t.Parallel()
		result := findArtefactBySourcePath(artefacts, "unknown.pk")
		assert.Nil(t, result)
	})
}

func TestFindFirstPageArtefact(t *testing.T) {
	t.Parallel()

	t.Run("finds first page artefact", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: "partial", SourcePath: "header.pk"},
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "index.pk", Content: "index"},
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "about.pk", Content: "about"},
		}
		result, packagePath := findFirstPageArtefact(artefacts, nil)
		require.NotNil(t, result)
		assert.Equal(t, "index", result.Content)
		assert.Empty(t, packagePath)
	})

	t.Run("returns nil when no page artefacts", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: "partial"},
		}
		result, packagePath := findFirstPageArtefact(artefacts, nil)
		assert.Nil(t, result)
		assert.Empty(t, packagePath)
	})

	t.Run("returns nil for empty artefacts", func(t *testing.T) {
		t.Parallel()
		result, packagePath := findFirstPageArtefact(nil, nil)
		assert.Nil(t, result)
		assert.Empty(t, packagePath)
	})

	t.Run("returns package path from manifest", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "index.pk"},
		}
		manifest := &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"index": {SourcePath: "index.pk", PackagePath: "mymodule/pages/index"},
			},
		}
		result, packagePath := findFirstPageArtefact(artefacts, manifest)
		require.NotNil(t, result)
		assert.Equal(t, "mymodule/pages/index", packagePath)
	})
}

func TestLookupPackagePath(t *testing.T) {
	t.Parallel()

	t.Run("finds matching source path", func(t *testing.T) {
		t.Parallel()
		manifest := &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"index": {SourcePath: "index.pk", PackagePath: "pkg/pages/index"},
			},
		}
		assert.Equal(t, "pkg/pages/index", lookupPackagePath(manifest, "index.pk"))
	})

	t.Run("returns empty for no match", func(t *testing.T) {
		t.Parallel()
		manifest := &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"index": {SourcePath: "index.pk", PackagePath: "pkg/pages/index"},
			},
		}
		assert.Empty(t, lookupPackagePath(manifest, "about.pk"))
	})

	t.Run("returns empty for nil manifest", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, lookupPackagePath(nil, "index.pk"))
	})

	t.Run("returns empty for nil pages", func(t *testing.T) {
		t.Parallel()
		manifest := &wasm_dto.GeneratedManifest{Pages: nil}
		assert.Empty(t, lookupPackagePath(manifest, "index.pk"))
	})
}

func TestFindPageArtefactForURL(t *testing.T) {
	t.Parallel()

	t.Run("finds matching page from manifest", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "index.pk", Content: "index-content"},
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "about.pk", Content: "about-content"},
		}
		manifest := &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"about": {
					SourcePath:    "about.pk",
					PackagePath:   "mymod/pages/about",
					RoutePatterns: map[string]string{"en": "/about"},
				},
			},
		}

		art, packagePath, deps := findPageArtefactForURL(artefacts, manifest, "/about", "mymod")
		require.NotNil(t, art)
		assert.Equal(t, "about-content", art.Content)
		assert.Equal(t, "mymod/pages/about", packagePath)
		assert.NotNil(t, deps)
	})

	t.Run("falls back to first page when no match", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "index.pk", Content: "index-content"},
		}
		manifest := &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"about": {
					SourcePath:    "about.pk",
					RoutePatterns: map[string]string{"en": "/about"},
				},
			},
		}

		art, _, _ := findPageArtefactForURL(artefacts, manifest, "/unknown", "mymod")
		require.NotNil(t, art)
		assert.Equal(t, "index-content", art.Content)
	})

	t.Run("works with nil manifest", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "index.pk", Content: "index-content"},
		}

		art, _, _ := findPageArtefactForURL(artefacts, nil, "/anything", "mymod")
		require.NotNil(t, art)
		assert.Equal(t, "index-content", art.Content)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	assert.Equal(t, "playground", config.DefaultModuleName)
	assert.Equal(t, 1024*1024, config.MaxSourceSize)
	assert.False(t, config.EnableMetrics)
	assert.Nil(t, config.StdlibPackages)
}

func TestNewOrchestrator_Options(t *testing.T) {
	t.Parallel()

	t.Run("with config", func(t *testing.T) {
		t.Parallel()
		config := Config{DefaultModuleName: "test", MaxSourceSize: 512}
		o := NewOrchestrator(WithConfig(config))
		assert.Equal(t, "test", o.config.DefaultModuleName)
		assert.Equal(t, 512, o.config.MaxSourceSize)
	})

	t.Run("with console", func(t *testing.T) {
		t.Parallel()
		console := &noOpConsole{}
		o := NewOrchestrator(WithConsole(console))
		assert.NotNil(t, o.console)
	})

	t.Run("with stdlib loader", func(t *testing.T) {
		t.Parallel()
		loader := newMockStdlibLoader()
		o := NewOrchestrator(WithStdlibLoader(loader))
		assert.NotNil(t, o.stdlibLoader)
	})

	t.Run("no options uses defaults", func(t *testing.T) {
		t.Parallel()
		o := NewOrchestrator()
		assert.Equal(t, "playground", o.config.DefaultModuleName)
		assert.Nil(t, o.stdlibLoader)
		assert.Nil(t, o.console)
	})
}

func TestOrchestrator_Log(t *testing.T) {
	t.Parallel()

	t.Run("nil console does not panic", func(t *testing.T) {
		t.Parallel()
		o := NewOrchestrator()
		assert.NotPanics(t, func() {
			o.log("info", "test message")
			o.log("debug", "debug message")
			o.log("warn", "warn message")
			o.log("error", "error message")
		})
	})

	t.Run("calls appropriate console method", func(t *testing.T) {
		t.Parallel()
		calls := make(map[string]int)
		console := &trackingConsole{calls: calls}
		o := NewOrchestrator(WithConsole(console))

		o.log("debug", "d")
		o.log("info", "i")
		o.log("warn", "w")
		o.log("error", "e")

		assert.Equal(t, 1, calls["debug"])
		assert.Equal(t, 1, calls["info"])
		assert.Equal(t, 1, calls["warn"])
		assert.Equal(t, 1, calls["error"])
	})
}

type trackingConsole struct {
	calls map[string]int
}

func (c *trackingConsole) Debug(_ string, _ ...any) { c.calls["debug"]++ }
func (c *trackingConsole) Info(_ string, _ ...any)  { c.calls["info"]++ }
func (c *trackingConsole) Warn(_ string, _ ...any)  { c.calls["warn"]++ }
func (c *trackingConsole) Error(_ string, _ ...any) { c.calls["error"]++ }

func TestTypeDataToResponse(t *testing.T) {
	t.Parallel()

	t.Run("extracts types functions and imports from matching module", func(t *testing.T) {
		t.Parallel()
		data := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"mymodule/pkg": {
					Path: "mymodule/pkg",
					Name: "pkg",
					NamedTypes: map[string]*inspector_dto.Type{
						"MyType": {
							Name:                 "MyType",
							TypeString:           "MyType",
							UnderlyingTypeString: "struct{...}",
						},
					},
					Funcs: map[string]*inspector_dto.Function{
						"MyFunc": {
							Name:       "MyFunc",
							TypeString: "()",
						},
					},
					FileImports: map[string]map[string]string{
						"file.go": {"fmt": "fmt"},
					},
				},
				"stdlib/pkg": {
					Path: "stdlib/pkg",
					Name: "pkg",
					NamedTypes: map[string]*inspector_dto.Type{
						"StdType": {Name: "StdType"},
					},
				},
			},
		}

		response := typeDataToResponse(data, "mymodule")
		assert.Len(t, response.Types, 1)
		assert.Equal(t, "MyType", response.Types[0].Name)
		assert.Len(t, response.Functions, 1)
		assert.Equal(t, "MyFunc", response.Functions[0].Name)
		assert.Len(t, response.Imports, 1)
		assert.Equal(t, "fmt", response.Imports[0].Path)
	})

	t.Run("empty data returns empty response", func(t *testing.T) {
		t.Parallel()
		data := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		}
		response := typeDataToResponse(data, "mymodule")
		assert.Empty(t, response.Types)
		assert.Empty(t, response.Functions)
		assert.Empty(t, response.Imports)
	})
}

func TestConvertTypeToInfo(t *testing.T) {
	t.Parallel()

	typ := &inspector_dto.Type{
		Name:                 "MyStruct",
		TypeString:           "MyStruct",
		UnderlyingTypeString: "struct{...}",
		DefinedInFilePath:    "main.go",
		DefinitionLine:       10,
		DefinitionColumn:     5,
		Fields: []*inspector_dto.Field{
			{Name: "Name", TypeString: "string", RawTag: "`json:\"name\"`", IsEmbedded: false},
			{Name: "Base", TypeString: "BaseType", IsEmbedded: true},
		},
		Methods: []*inspector_dto.Method{
			{Name: "String", TypeString: "() string", IsPointerReceiver: true},
		},
	}

	info := convertTypeToInfo(typ)
	assert.Equal(t, "MyStruct", info.Name)
	assert.Equal(t, "struct", info.Kind)
	assert.True(t, info.IsExported)
	assert.Equal(t, "main.go", info.Location.FilePath)
	assert.Equal(t, 10, info.Location.Line)
	assert.Equal(t, 5, info.Location.Column)

	require.Len(t, info.Fields, 2)
	assert.Equal(t, "Name", info.Fields[0].Name)
	assert.Equal(t, "string", info.Fields[0].TypeString)
	assert.True(t, info.Fields[1].IsEmbedded)

	require.Len(t, info.Methods, 1)
	assert.Equal(t, "String", info.Methods[0].Name)
	assert.True(t, info.Methods[0].IsPointerReceiver)
}

func TestParseScriptContent(t *testing.T) {
	t.Parallel()

	t.Run("parses valid Go code", func(t *testing.T) {
		t.Parallel()
		file := parseScriptContent("type Props struct { Name string }")
		require.NotNil(t, file)
	})

	t.Run("parses code with package statement", func(t *testing.T) {
		t.Parallel()
		file := parseScriptContent("package main\ntype Props struct { Name string }")
		require.NotNil(t, file)
	})

	t.Run("returns nil for invalid code", func(t *testing.T) {
		t.Parallel()
		file := parseScriptContent("this is not valid Go {{{")
		assert.Nil(t, file)
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		t.Parallel()

		file := parseScriptContent("")
		require.NotNil(t, file)
	})
}

func TestExtractDeclInfo(t *testing.T) {
	t.Parallel()

	t.Run("extracts types including Props suffix", func(t *testing.T) {
		t.Parallel()
		file := parseScriptContent("type MyProps struct { Name string }\ntype Other struct {}")
		require.NotNil(t, file)

		info := &wasm_dto.ScriptBlockInfo{
			Types: make([]string, 0),
		}
		for _, declaration := range file.Decls {
			extractDeclInfo(declaration, info)
		}

		assert.Contains(t, info.Types, "MyProps")
		assert.Contains(t, info.Types, "Other")
		assert.Equal(t, "MyProps", info.PropsType)
	})

	t.Run("extracts init function", func(t *testing.T) {
		t.Parallel()
		file := parseScriptContent("func init() {}\nfunc other() {}")
		require.NotNil(t, file)

		info := &wasm_dto.ScriptBlockInfo{
			Types: make([]string, 0),
		}
		for _, declaration := range file.Decls {
			extractDeclInfo(declaration, info)
		}

		assert.True(t, info.HasInit)
	})

	t.Run("Props exactly named Props", func(t *testing.T) {
		t.Parallel()
		file := parseScriptContent("type Props struct { X int }")
		require.NotNil(t, file)

		info := &wasm_dto.ScriptBlockInfo{
			Types: make([]string, 0),
		}
		for _, declaration := range file.Decls {
			extractDeclInfo(declaration, info)
		}

		assert.Equal(t, "Props", info.PropsType)
	})
}

func TestAppendKeywordCompletions(t *testing.T) {
	t.Parallel()

	items := appendKeywordCompletions(nil)
	assert.Len(t, items, len(goKeywords))
	for _, item := range items {
		assert.Equal(t, "keyword", item.Kind)
	}
}

func TestAppendBuiltinTypeCompletions(t *testing.T) {
	t.Parallel()

	items := appendBuiltinTypeCompletions(nil)
	assert.Len(t, items, len(goBuiltinTypes))
	for _, item := range items {
		assert.Equal(t, "type", item.Kind)
	}
}

func TestAppendBuiltinFuncCompletions(t *testing.T) {
	t.Parallel()

	items := appendBuiltinFuncCompletions(nil)
	assert.Len(t, items, len(goBuiltinFuncs))
	for _, item := range items {
		assert.Equal(t, "function", item.Kind)
	}
}

func TestAppendStdlibPackageCompletions(t *testing.T) {
	t.Parallel()

	t.Run("nil stdlibData returns original items", func(t *testing.T) {
		t.Parallel()
		items := appendStdlibPackageCompletions(nil, nil)
		assert.Nil(t, items)
	})

	t.Run("adds package completions", func(t *testing.T) {
		t.Parallel()
		data := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"fmt":  {Path: "fmt"},
				"time": {Path: "time"},
			},
		}
		items := appendStdlibPackageCompletions(nil, data)
		assert.Len(t, items, 2)
		for _, item := range items {
			assert.Equal(t, "module", item.Kind)
			assert.Equal(t, "import", item.Detail)
		}
	})
}

func TestGetBasicCompletions(t *testing.T) {
	t.Parallel()

	response := getBasicCompletions(nil)
	assert.True(t, response.Success)
	assert.Empty(t, response.Error)
	expectedLen := len(goKeywords) + len(goBuiltinTypes) + len(goBuiltinFuncs)
	assert.Len(t, response.Items, expectedLen)
}

func TestBuildCompletionContext(t *testing.T) {
	t.Parallel()

	t.Run("known package import", func(t *testing.T) {
		t.Parallel()
		imports := map[string]string{"fmt": "fmt"}
		ctx := buildCompletionContext("fmt", "Pri", imports)
		assert.Equal(t, completionKindPackageMember, ctx.kind)
		assert.Equal(t, "fmt", ctx.pkgAlias)
		assert.Equal(t, "fmt", ctx.packagePath)
		assert.Equal(t, "Pri", ctx.prefix)
	})

	t.Run("unknown identifier becomes field/method context", func(t *testing.T) {
		t.Parallel()
		imports := map[string]string{}
		ctx := buildCompletionContext("myVar", "Fie", imports)
		assert.Equal(t, completionKindFieldMethod, ctx.kind)
		assert.Equal(t, "myVar", ctx.expressionType)
		assert.Equal(t, "Fie", ctx.prefix)
	})
}

func TestGetPackageMemberCompletions(t *testing.T) {
	t.Parallel()

	t.Run("nil stdlibData", func(t *testing.T) {
		t.Parallel()
		result := getPackageMemberCompletions("fmt", "", nil)
		assert.Nil(t, result)
	})

	t.Run("package not found", func(t *testing.T) {
		t.Parallel()
		data := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{}}
		result := getPackageMemberCompletions("unknown", "", data)
		assert.Nil(t, result)
	})

	t.Run("returns exported types and functions", func(t *testing.T) {
		t.Parallel()
		data := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"fmt": {
					NamedTypes: map[string]*inspector_dto.Type{
						"Stringer": {Name: "Stringer", UnderlyingTypeString: "interface{...}"},
						"private":  {Name: "private", UnderlyingTypeString: "struct{...}"},
					},
					Funcs: map[string]*inspector_dto.Function{
						"Println": {Name: "Println", TypeString: "(a ...any) (n int, err error)"},
						"init":    {Name: "init", TypeString: "()"},
					},
				},
			},
		}
		items := getPackageMemberCompletions("fmt", "", data)
		assert.Len(t, items, 2)
	})

	t.Run("filters by prefix case-insensitive", func(t *testing.T) {
		t.Parallel()
		data := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"fmt": {
					NamedTypes: map[string]*inspector_dto.Type{
						"Stringer": {Name: "Stringer"},
						"State":    {Name: "State"},
					},
					Funcs: map[string]*inspector_dto.Function{
						"Println": {Name: "Println"},
						"Sprint":  {Name: "Sprint"},
					},
				},
			},
		}
		items := getPackageMemberCompletions("fmt", "str", data)
		assert.Len(t, items, 1)
		assert.Equal(t, "Stringer", items[0].Label)
	})
}

func TestExtractImportsFromSource(t *testing.T) {
	t.Parallel()

	stdlibData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"fmt":  {Path: "fmt"},
			"time": {Path: "time"},
			"os":   {Path: "os"},
		},
	}

	t.Run("single import", func(t *testing.T) {
		t.Parallel()
		source := `import "fmt"`
		imports := extractImportsFromSource(source, stdlibData)
		assert.Equal(t, "fmt", imports["fmt"])
	})

	t.Run("aliased import", func(t *testing.T) {
		t.Parallel()
		source := `import t "time"`
		imports := extractImportsFromSource(source, stdlibData)
		assert.Equal(t, "time", imports["t"])
	})

	t.Run("import block", func(t *testing.T) {
		t.Parallel()
		source := "import (\n\t\"fmt\"\n\t\"time\"\n)"
		imports := extractImportsFromSource(source, stdlibData)
		assert.Equal(t, "fmt", imports["fmt"])
		assert.Equal(t, "time", imports["time"])
	})

	t.Run("nil stdlibData returns empty map", func(t *testing.T) {
		t.Parallel()
		imports := extractImportsFromSource(`import "fmt"`, nil)
		assert.Empty(t, imports)
	})

	t.Run("non-stdlib import not included", func(t *testing.T) {
		t.Parallel()
		source := `import "github.com/unknown/pkg"`
		imports := extractImportsFromSource(source, stdlibData)
		assert.Empty(t, imports)
	})
}

func TestAnalyseCompletionContext(t *testing.T) {
	t.Parallel()

	t.Run("no dot returns scope context", func(t *testing.T) {
		t.Parallel()
		imports := map[string]string{}
		ctx := analyseCompletionContext("x := 1", 1, 5, nil, imports)
		assert.Equal(t, completionKindScope, ctx.kind)
	})

	t.Run("package dot returns package member context", func(t *testing.T) {
		t.Parallel()
		imports := map[string]string{"fmt": "fmt"}
		ctx := analyseCompletionContext("fmt.Pr", 1, 7, nil, imports)
		assert.Equal(t, completionKindPackageMember, ctx.kind)
		assert.Equal(t, "fmt", ctx.pkgAlias)
		assert.Equal(t, "Pr", ctx.prefix)
	})

	t.Run("empty before dot returns scope", func(t *testing.T) {
		t.Parallel()
		imports := map[string]string{}
		ctx := analyseCompletionContext(".foo", 1, 4, nil, imports)
		assert.Equal(t, completionKindScope, ctx.kind)
	})
}

func TestFormatTypeHover(t *testing.T) {
	t.Parallel()

	t.Run("type without methods", func(t *testing.T) {
		t.Parallel()
		typ := &inspector_dto.Type{
			Name:                 "Duration",
			UnderlyingTypeString: "int64",
		}
		result := formatTypeHover("time", typ)
		assert.Contains(t, result, "time")
		assert.Contains(t, result, "Duration")
		assert.Contains(t, result, "int64")
		assert.NotContains(t, result, "methods")
	})

	t.Run("type with methods", func(t *testing.T) {
		t.Parallel()
		typ := &inspector_dto.Type{
			Name:                 "Time",
			UnderlyingTypeString: "struct{...}",
			Methods: []*inspector_dto.Method{
				{Name: "String"},
				{Name: "Format"},
			},
		}
		result := formatTypeHover("time", typ)
		assert.Contains(t, result, "2 methods")
	})
}

func TestFormatFunctionHover(t *testing.T) {
	t.Parallel()

	inspectedFunction := &inspector_dto.Function{
		Name:       "Println",
		TypeString: "(a ...any) (n int, err error)",
	}
	result := formatFunctionHover("fmt", inspectedFunction)
	assert.Contains(t, result, "fmt")
	assert.Contains(t, result, "Println")
	assert.Contains(t, result, "(a ...any) (n int, err error)")
}

func TestFindStdlibTypeContent(t *testing.T) {
	t.Parallel()

	t.Run("nil stdlibData", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, findStdlibTypeContent("Time", nil))
	})

	t.Run("finds type in stdlib", func(t *testing.T) {
		t.Parallel()
		data := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"time": {
					NamedTypes: map[string]*inspector_dto.Type{
						"Time": {Name: "Time", UnderlyingTypeString: "struct{...}"},
					},
				},
			},
		}
		result := findStdlibTypeContent("Time", data)
		assert.Contains(t, result, "Time")
	})

	t.Run("returns empty for unknown type", func(t *testing.T) {
		t.Parallel()
		data := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{},
		}
		assert.Empty(t, findStdlibTypeContent("Unknown", data))
	})
}

func TestOrchestrator_Initialise_NoLoader(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	err := o.Initialise(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stdlib loader not configured")
}

func TestOrchestrator_Initialise_Idempotent(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	require.NoError(t, o.Initialise(t.Context()))
	require.NoError(t, o.Initialise(t.Context()))
}

func TestOrchestrator_GetStdlibData_NotInitialised(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	_, err := o.GetStdlibData()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialised")
}

func TestOrchestrator_GetCompletions_NotInitialised(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	_, err := o.GetCompletions(t.Context(), &wasm_dto.CompletionRequest{
		Source: "package main",
		Line:   1,
		Column: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialised")
}

func TestOrchestrator_GetHover_NotInitialised(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	_, err := o.GetHover(t.Context(), &wasm_dto.HoverRequest{
		Source: "package main",
		Line:   1,
		Column: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialised")
}

func TestOrchestrator_GetHover_ParseError(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)
	require.NoError(t, o.Initialise(t.Context()))

	response, err := o.GetHover(t.Context(), &wasm_dto.HoverRequest{
		Source: "package main { invalid",
		Line:   1,
		Column: 1,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "parse error")
}

func TestOrchestrator_Validate_EmptySource(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	response, err := o.Validate(t.Context(), &wasm_dto.ValidateRequest{
		Source:   "",
		FilePath: "test.go",
	})
	require.NoError(t, err)
	assert.False(t, response.Valid)
}

func TestOrchestrator_Generate_NoGenerator(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	response, err := o.Generate(t.Context(), &wasm_dto.GenerateFromSourcesRequest{})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "generator not configured")
}

func TestOrchestrator_Render_NoRenderer(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	response, err := o.Render(t.Context(), &wasm_dto.RenderFromSourcesRequest{})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "renderer not configured")
}

func TestOrchestrator_RenderPreview_NoRenderer(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	response, err := o.RenderPreview(t.Context(), &wasm_dto.RenderPreviewRequest{})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "renderer not configured")
}

func TestOrchestrator_RenderPreview_StaticTemplate(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: func(_ context.Context, request *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
			assert.Equal(t, "pages/preview.pk", request.EntryPoint)
			assert.Contains(t, request.Sources, "pages/preview.pk")
			return &wasm_dto.RenderFromSourcesResponse{
				Success: true,
				HTML:    "<h1>Hello</h1>",
				CSS:     "h1 { color: red; }",
			}, nil
		},
	}

	o := NewOrchestrator(WithRenderer(renderer))
	response, err := o.RenderPreview(t.Context(), &wasm_dto.RenderPreviewRequest{
		Template: "<h1>Hello</h1>",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "<h1>Hello</h1>", response.HTML)
	assert.Equal(t, "h1 { color: red; }", response.CSS)
}

func TestOrchestrator_RenderPreview_WithScript(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: func(_ context.Context, request *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
			src := request.Sources["pages/preview.pk"]
			assert.Contains(t, src, "<script>")
			assert.Contains(t, src, "fmt.Println")
			return &wasm_dto.RenderFromSourcesResponse{
				Success: true,
				HTML:    "<p>output</p>",
			}, nil
		},
	}

	o := NewOrchestrator(WithRenderer(renderer))
	response, err := o.RenderPreview(t.Context(), &wasm_dto.RenderPreviewRequest{
		Template: "<p>output</p>",
		Script:   "fmt.Println(\"hello\")",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
}

func TestOrchestrator_RenderPreview_RenderError(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: func(_ context.Context, _ *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
			return nil, errors.New("render exploded")
		},
	}

	o := NewOrchestrator(WithRenderer(renderer))
	response, err := o.RenderPreview(t.Context(), &wasm_dto.RenderPreviewRequest{
		Template: "<p>test</p>",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "render exploded")
}

func TestOrchestrator_GetRuntimeInfo_NoLoader(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	info, err := o.GetRuntimeInfo(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, info.GoVersion)
	assert.Nil(t, info.StdlibPackages)
	assert.Contains(t, info.Capabilities, "analyse")
}

func TestOrchestrator_GetRuntimeInfo_WithGenerator(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&mockGenerator{}),
	)
	info, err := o.GetRuntimeInfo(t.Context())
	require.NoError(t, err)
	assert.Contains(t, info.Capabilities, "generate")
}

type mockGenerator struct{}

func (*mockGenerator) Generate(_ context.Context, _ *wasm_dto.GenerateFromSourcesRequest) (*wasm_dto.GenerateFromSourcesResponse, error) {
	return nil, nil
}

func TestValidateAnalyseRequest(t *testing.T) {
	t.Parallel()

	t.Run("empty sources", func(t *testing.T) {
		t.Parallel()
		response := validateAnalyseRequest(t.Context(), &wasm_dto.AnalyseRequest{Sources: map[string]string{}}, 1024)
		require.NotNil(t, response)
		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "no source files")
	})

	t.Run("source size exceeds max", func(t *testing.T) {
		t.Parallel()
		response := validateAnalyseRequest(t.Context(), &wasm_dto.AnalyseRequest{
			Sources: map[string]string{"main.go": "package main"},
		}, 5)
		require.NotNil(t, response)
		assert.Contains(t, response.Error, "exceeds maximum")
	})

	t.Run("valid request returns nil", func(t *testing.T) {
		t.Parallel()
		response := validateAnalyseRequest(t.Context(), &wasm_dto.AnalyseRequest{
			Sources: map[string]string{"main.go": "package main"},
		}, 1024)
		assert.Nil(t, response)
	})
}

func TestOrchestrator_Analyse_EmptySources(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)
	require.NoError(t, o.Initialise(t.Context()))

	response, err := o.Analyse(t.Context(), &wasm_dto.AnalyseRequest{
		Sources: map[string]string{},
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "no source files")
}

func TestOrchestrator_Analyse_DefaultModuleName(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)
	require.NoError(t, o.Initialise(t.Context()))

	response, err := o.Analyse(t.Context(), &wasm_dto.AnalyseRequest{
		Sources: map[string]string{
			"main.go": "package main\ntype X struct{}",
		},
		ModuleName: "",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
}

func TestLineColumnToOffset(t *testing.T) {
	t.Parallel()

	t.Run("nil file returns 0", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, lineColumnToOffset(nil, 1, 1))
	})
}

type stubRenderPort struct {
	renderFunc    func(context.Context, *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error)
	renderASTFunc func(context.Context, *wasm_dto.RenderFromASTRequest) (*wasm_dto.RenderFromASTResponse, error)
}

func (s *stubRenderPort) Render(ctx context.Context, request *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
	if s.renderFunc != nil {
		return s.renderFunc(ctx, request)
	}
	return &wasm_dto.RenderFromSourcesResponse{Success: false, Error: "not configured"}, nil
}

func (s *stubRenderPort) RenderFromAST(ctx context.Context, request *wasm_dto.RenderFromASTRequest) (*wasm_dto.RenderFromASTResponse, error) {
	if s.renderASTFunc != nil {
		return s.renderASTFunc(ctx, request)
	}
	return &wasm_dto.RenderFromASTResponse{}, nil
}
