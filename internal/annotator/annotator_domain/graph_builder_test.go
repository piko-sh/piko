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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

type mockFSReader struct {
	files      map[string][]byte
	failOnRead string
}

func (m *mockFSReader) ReadFile(_ context.Context, path string) ([]byte, error) {
	if m.failOnRead != "" && m.failOnRead == path {
		return nil, os.ErrNotExist
	}
	content, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return content, nil
}

func newGraphBuilderMockResolver(moduleName, baseDir string) *resolver_domain.MockResolver {
	return &resolver_domain.MockResolver{
		GetBaseDirFunc:    func() string { return baseDir },
		GetModuleNameFunc: func() string { return moduleName },
		ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
			if filepath.IsAbs(importPath) {
				return importPath, nil
			}
			return filepath.Join(baseDir, importPath), nil
		},
		ResolveCSSPathFunc: func(_ context.Context, importPath string, containingDir string) (string, error) {
			if filepath.IsAbs(importPath) {
				return importPath, nil
			}
			return filepath.Join(containingDir, importPath), nil
		},
		ResolveAssetPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
			if filepath.IsAbs(importPath) {
				return importPath, nil
			}
			return filepath.Join(baseDir, importPath), nil
		},
		ConvertEntryPointPathToManifestKeyFunc: func(entryPointPath string) string {
			prefix := moduleName + "/"
			if result, found := strings.CutPrefix(entryPointPath, prefix); found {
				return result
			}
			return entryPointPath
		},
	}
}

type mockComponentCache struct{}

func (m *mockComponentCache) GetOrSet(ctx context.Context, key string, loader func(context.Context) (*annotator_dto.ParsedComponent, error)) (*annotator_dto.ParsedComponent, error) {

	return loader(ctx)
}

func (m *mockComponentCache) Clear(_ context.Context) {

}

var _ ComponentCachePort = (*mockComponentCache)(nil)

type testHarness struct {
	t        *testing.T
	fsReader *mockFSReader
	resolver *resolver_domain.MockResolver
	cache    *mockComponentCache
	builder  *GraphBuilder
}

func newTestHarness(t *testing.T, moduleName, baseDir string) *testHarness {
	t.Helper()
	fsReader := &mockFSReader{files: make(map[string][]byte)}
	resolver := newGraphBuilderMockResolver(moduleName, baseDir)
	cache := &mockComponentCache{}
	builder := NewGraphBuilder(resolver, fsReader, cache, AnnotatorPathsConfig{}, false)
	return &testHarness{
		t:        t,
		fsReader: fsReader,
		resolver: resolver,
		cache:    cache,
		builder:  builder,
	}
}

func (h *testHarness) addFile(path, content string) {
	h.fsReader.files[filepath.Join(h.resolver.GetBaseDir(), path)] = []byte(content)
}

func TestGraphBuilder_Build(t *testing.T) {
	baseDir := "/app/src"

	t.Run("Basic Graph Structures", func(t *testing.T) {
		t.Run("should parse a single file with no imports", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("main.pk", `<script type="application/x-go">package main</script>`)
			entryPath := filepath.Join(baseDir, "main.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
			assert.Len(t, graph.Components, 1)
		})
		t.Run("should parse a linear dependency chain", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("pages/main.pk", `<script type="application/x-go">package main; import _ "partials/card.pk"</script>`)
			h.addFile("partials/card.pk", `<script type="application/x-go">package card</script>`)
			entryPath := filepath.Join(baseDir, "pages/main.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
			assert.Len(t, graph.Components, 2)
		})
		t.Run("should handle a diamond dependency graph without duplication", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("main.pk", `<script type="application/x-go">package main; import _ "components/header.pk"; import _ "components/footer.pk"</script>`)
			h.addFile("components/header.pk", `<script type="application/x-go">package header; import _ "components/icon.pk"</script>`)
			h.addFile("components/footer.pk", `<script type="application/x-go">package footer; import _ "components/icon.pk"</script>`)
			h.addFile("components/icon.pk", `<script type="application/x-go">package icon</script>`)
			entryPath := filepath.Join(baseDir, "main.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
			assert.Len(t, graph.Components, 4)
		})
		t.Run("should handle circular dependencies gracefully", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("a.pk", `<script type="application/x-go">package a; import _ "b.pk"</script>`)
			h.addFile("b.pk", `<script type="application/x-go">package b; import _ "a.pk"</script>`)
			entryPath := filepath.Join(baseDir, "a.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})
			require.NoError(t, err)

			assert.NotEmpty(t, diagnostics)
			assert.Len(t, graph.Components, 2)
		})
	})

	t.Run("File System and Content Edge Cases", func(t *testing.T) {
		t.Run("should handle a missing imported file during discovery", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("main.pk", `<script type="application/x-go">package main; import _ "nonexistent.pk"</script>`)
			entryPath := filepath.Join(baseDir, "main.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to read file during discovery")

			_ = diagnostics
			_ = graph
		})
		t.Run("should return a fatal diagnostic error for invalid template syntax", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("main.pk", `<template><div>{{ broken</template>`)
			entryPath := filepath.Join(baseDir, "main.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})
			require.NoError(t, err)

			assert.True(t, ast_domain.HasErrors(diagnostics))

			assert.NotNil(t, graph)

			if len(diagnostics) > 0 {
				parseErr := &ParseDiagnosticError{Diagnostics: diagnostics}
				diagErr, ok := errors.AsType[*ParseDiagnosticError](parseErr)
				assert.True(t, ok)
				_ = diagErr
			}
		})
		t.Run("should handle component with only a template block", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("main.pk", `<template><div>Icon</div></template>`)
			entryPath := filepath.Join(baseDir, "main.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
			require.Len(t, graph.Components, 1)

			hash := graph.PathToHashedName[entryPath]
			parsedComponent := graph.Components[hash]

			assert.NotNil(t, parsedComponent.Template)

			assert.NotNil(t, parsedComponent.Script)
			assert.Equal(t, "piko_default", parsedComponent.Script.GoPackageName)
		})
	})

	t.Run("Go Package Name Parsing", func(t *testing.T) {
		t.Run("should correctly parse package name from script block", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("main.pk", `<script type="application/x-go">package main</script>`)
			entryPath := filepath.Join(baseDir, "main.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)

			hash := graph.PathToHashedName[entryPath]
			parsedComponent := graph.Components[hash]

			require.NotNil(t, parsedComponent.Script)
			assert.Equal(t, "main", parsedComponent.Script.GoPackageName)

		})
		t.Run("should correctly parse package name from subdirectory component", func(t *testing.T) {
			h := newTestHarness(t, "my-project", baseDir)
			h.addFile("partials/card.pk", `<script type="application/x-go">package card</script>`)
			entryPath := filepath.Join(baseDir, "partials", "card.pk")
			graph, diagnostics, err := h.builder.Build(context.Background(), []string{entryPath})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)

			hash := graph.PathToHashedName[entryPath]
			parsedComponent := graph.Components[hash]

			require.NotNil(t, parsedComponent.Script)
			assert.Equal(t, "card", parsedComponent.Script.GoPackageName)
		})
	})
}

func Test_buildAliasFromPath(t *testing.T) {
	aliasRegex := regexp.MustCompile(`^([a-z0-9_]+)_([a-f0-9]{8})$`)

	testCases := []struct {
		name                  string
		input                 string
		expectedSanitisedPart string
	}{
		{name: "Standard Path", input: "/app/src/components/user-card.pk", expectedSanitisedPart: "app_src_components_user_card"},
		{name: "Path with Hyphens", input: "/app-src/components/user-card.pk", expectedSanitisedPart: "app_src_components_user_card"},
		{name: "Path with Special Chars", input: "/app/src/{id}/card.pk", expectedSanitisedPart: "app_src_id_card"},
		{name: "Windows Path", input: `C:\Users\Test\components\user-card.pk`, expectedSanitisedPart: "c_users_test_components_user_card"},
		{name: "No Extension", input: "/app/src/component", expectedSanitisedPart: "app_src_component"},
		{name: "Invocation Key", input: "card:is-primary=true", expectedSanitisedPart: "card_is_primary_true"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildAliasFromPath(tc.input)
			matches := aliasRegex.FindStringSubmatch(result)
			require.Len(t, matches, 3, "Alias '%s' does not match the expected format 'sanitised_hash'", result)
			assert.Equal(t, tc.expectedSanitisedPart, matches[1], "The sanitised part of the alias is incorrect")
			assert.Len(t, matches[2], shortHashLength, "Hash part should be 8 characters long")
			assert.Equal(t, result, buildAliasFromPath(tc.input), "Function should be deterministic")
		})
	}
}

func TestSanitiseForPackageName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "Simple lowercase", input: "hello", expected: "hello"},
		{name: "Simple CamelCase", input: "HelloWorld", expected: "helloworld"},
		{name: "With hyphens", input: "hello-world", expected: "hello_world"},
		{name: "Starts with number", input: "1world", expected: "p1world"},
		{name: "Starts with separator", input: "-world", expected: "world"},
		{name: "Multiple separators", input: "hello--world", expected: "hello_world"},
		{name: "Empty string", input: "", expected: defaultPackageName},
		{name: "Ends with separator", input: "world-", expected: "world"},
		{name: "Complex case", input: "1-API-v2_Component-Test.", expected: "p1_api_v2_component_test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, SanitiseForPackageName(tc.input))
		})
	}
}

func TestExtractModuleImportPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard GOMODCACHE path with version",
			input:    "/home/user/go/pkg/mod/github.com/ui/lib@v1.2.3/components/button.pk",
			expected: "github.com/ui/lib/components/button.pk",
		},
		{
			name:     "GOMODCACHE path without version",
			input:    "/home/user/go/pkg/mod/github.com/ui/lib/components/button.pk",
			expected: "github.com/ui/lib/components/button.pk",
		},
		{
			name:     "path without pkg/mod prefix",
			input:    "/home/user/project/components/button.pk",
			expected: "/home/user/project/components/button.pk",
		},
		{
			name:     "version at end of path",
			input:    "/home/user/go/pkg/mod/github.com/org/repo@v2.0.0",
			expected: "github.com/org/repo",
		},
		{
			name:     "complex module path",
			input:    "/go/pkg/mod/piko.sh/piko-ui@v0.1.0/partials/card.pk",
			expected: "piko.sh/piko-ui/partials/card.pk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractModuleImportPath(tc.input)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateCacheKey(t *testing.T) {
	t.Parallel()

	t.Run("same content and path produces same key", func(t *testing.T) {
		t.Parallel()

		path := "/project/pages/home.pk"
		content := []byte("<template><div>Hello</div></template>")

		key1 := generateCacheKey(path, content)
		key2 := generateCacheKey(path, content)

		assert.Equal(t, key1, key2)
	})

	t.Run("different paths produce different keys", func(t *testing.T) {
		t.Parallel()

		content := []byte("<template><div>Hello</div></template>")

		key1 := generateCacheKey("/project/pages/home.pk", content)
		key2 := generateCacheKey("/project/pages/about.pk", content)

		assert.NotEqual(t, key1, key2)
	})

	t.Run("different content produces different keys", func(t *testing.T) {
		t.Parallel()

		path := "/project/pages/home.pk"

		key1 := generateCacheKey(path, []byte("<template><div>Hello</div></template>"))
		key2 := generateCacheKey(path, []byte("<template><div>Goodbye</div></template>"))

		assert.NotEqual(t, key1, key2)
	})

	t.Run("key is hexadecimal string", func(t *testing.T) {
		t.Parallel()

		key := generateCacheKey("/test.pk", []byte("content"))

		assert.Regexp(t, "^[0-9a-f]+$", key)
	})
}

func TestResolveAndQueueEntryPoints(t *testing.T) {
	t.Parallel()

	baseDir := "/project/src"

	t.Run("should resolve absolute paths without calling resolver", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		absPath := filepath.Join(baseDir, "pages/home.pk")

		queue, visited, diagnostics := h.builder.resolveAndQueueEntryPoints(context.Background(), []string{absPath})

		assert.Empty(t, diagnostics)
		assert.Equal(t, []string{absPath}, queue)
		assert.True(t, visited[absPath])
	})

	t.Run("should resolve relative paths using the resolver", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		relPath := "pages/home.pk"
		expectedResolved := filepath.Join(baseDir, relPath)

		queue, visited, diagnostics := h.builder.resolveAndQueueEntryPoints(context.Background(), []string{relPath})

		assert.Empty(t, diagnostics)
		require.Len(t, queue, 1)
		assert.Equal(t, expectedResolved, queue[0])
		assert.True(t, visited[expectedResolved])
	})

	t.Run("should deduplicate identical paths", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		absPath := filepath.Join(baseDir, "main.pk")

		queue, visited, diagnostics := h.builder.resolveAndQueueEntryPoints(context.Background(), []string{absPath, absPath, absPath})

		assert.Empty(t, diagnostics)
		assert.Len(t, queue, 1)
		assert.True(t, visited[absPath])
	})

	t.Run("should produce a diagnostic when relative path resolution fails", func(t *testing.T) {
		t.Parallel()

		failingResolver := &graphBuilderMockResolverWithFailures{
			baseDir:    baseDir,
			moduleName: "my-project",
			failPaths:  map[string]bool{"nonexistent.pk": true},
		}
		builder := NewGraphBuilder(failingResolver, &mockFSReader{files: map[string][]byte{}}, &mockComponentCache{}, AnnotatorPathsConfig{}, false)

		queue, _, diagnostics := builder.resolveAndQueueEntryPoints(context.Background(), []string{"nonexistent.pk"})

		require.NotEmpty(t, diagnostics)
		assert.Contains(t, diagnostics[0].Message, "Cannot resolve entry point")
		assert.Empty(t, queue)
	})

	t.Run("should handle empty entry points", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)

		queue, visited, diagnostics := h.builder.resolveAndQueueEntryPoints(context.Background(), []string{})

		assert.Empty(t, diagnostics)
		assert.Empty(t, queue)
		assert.Empty(t, visited)
	})
}

func TestTraverseDependencyGraph(t *testing.T) {
	t.Parallel()

	baseDir := "/project/src"

	t.Run("should discover transitive dependencies via BFS", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		mainPath := filepath.Join(baseDir, "main.pk")
		cardPath := filepath.Join(baseDir, "card.pk")
		h.fsReader.files[mainPath] = []byte(`<script type="application/x-go">package main; import _ "card.pk"</script>`)
		h.fsReader.files[cardPath] = []byte(`<script type="application/x-go">package card</script>`)

		visited := map[string]bool{mainPath: true}
		var diagnostics []*ast_domain.Diagnostic

		resultQueue, err := h.builder.traverseDependencyGraph(context.Background(), []string{mainPath}, visited, &diagnostics)

		require.NoError(t, err)
		assert.Empty(t, diagnostics)
		assert.Contains(t, resultQueue, cardPath)
		assert.True(t, visited[cardPath])
	})

	t.Run("should return error when file cannot be read", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		missingPath := filepath.Join(baseDir, "missing.pk")
		visited := map[string]bool{missingPath: true}
		var diagnostics []*ast_domain.Diagnostic

		_, err := h.builder.traverseDependencyGraph(context.Background(), []string{missingPath}, visited, &diagnostics)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file during discovery")
	})

	t.Run("should produce diagnostic when import resolution fails", func(t *testing.T) {
		t.Parallel()

		failingResolver := &graphBuilderMockResolverWithFailures{
			baseDir:    baseDir,
			moduleName: "my-project",
			failPaths:  map[string]bool{"broken.pk": true},
		}
		fsReader := &mockFSReader{files: map[string][]byte{
			filepath.Join(baseDir, "main.pk"): []byte(`<script type="application/x-go">package main; import _ "broken.pk"</script>`),
		}}
		builder := NewGraphBuilder(failingResolver, fsReader, &mockComponentCache{}, AnnotatorPathsConfig{}, false)

		mainPath := filepath.Join(baseDir, "main.pk")
		visited := map[string]bool{mainPath: true}
		var diagnostics []*ast_domain.Diagnostic

		resultQueue, err := builder.traverseDependencyGraph(context.Background(), []string{mainPath}, visited, &diagnostics)

		require.NoError(t, err)
		require.NotEmpty(t, diagnostics)
		assert.Contains(t, diagnostics[0].Message, "failed to resolve import")
		assert.Len(t, resultQueue, 1, "queue should only contain the original entry point")
	})

	t.Run("should not revisit already visited paths", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		mainPath := filepath.Join(baseDir, "main.pk")
		cardPath := filepath.Join(baseDir, "card.pk")
		h.fsReader.files[mainPath] = []byte(`<script type="application/x-go">package main; import _ "card.pk"</script>`)
		h.fsReader.files[cardPath] = []byte(`<script type="application/x-go">package card</script>`)

		visited := map[string]bool{mainPath: true, cardPath: true}
		var diagnostics []*ast_domain.Diagnostic

		resultQueue, err := h.builder.traverseDependencyGraph(context.Background(), []string{mainPath}, visited, &diagnostics)

		require.NoError(t, err)
		assert.Len(t, resultQueue, 1, "should not add already-visited card.pk to the queue")
	})
}

func TestProcessParseResultError(t *testing.T) {
	t.Parallel()

	builder := &GraphBuilder{}

	t.Run("should return nil for a successful result", func(t *testing.T) {
		t.Parallel()

		result := &parseResult{err: nil}
		diagnostics, fatalErr := builder.processParseResultError(result)

		assert.Nil(t, diagnostics)
		assert.Nil(t, fatalErr)
	})

	t.Run("should extract diagnostics from ParseDiagnosticError", func(t *testing.T) {
		t.Parallel()

		expectedDiag := ast_domain.NewDiagnostic(ast_domain.Error, "template error", "test.pk", ast_domain.Location{}, "/test.pk")
		diagErr := &ParseDiagnosticError{Diagnostics: []*ast_domain.Diagnostic{expectedDiag}}
		result := &parseResult{err: diagErr}

		diagnostics, fatalErr := builder.processParseResultError(result)

		assert.Nil(t, fatalErr)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, "template error", diagnostics[0].Message)
	})

	t.Run("should extract diagnostics from scriptBlockParseError", func(t *testing.T) {
		t.Parallel()

		scriptErr := &scriptBlockParseError{reason: "unexpected EOF"}
		result := &parseResult{err: scriptErr, path: "/test.pk"}

		diagnostics, fatalErr := builder.processParseResultError(result)

		assert.Nil(t, fatalErr)
		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "cannot parse <script> snippet")
	})

	t.Run("should return unknown error types as fatal", func(t *testing.T) {
		t.Parallel()

		unknownErr := errors.New("disk failure")
		result := &parseResult{err: unknownErr}

		diagnostics, fatalErr := builder.processParseResultError(result)

		assert.Nil(t, diagnostics)
		require.Error(t, fatalErr)
		assert.Equal(t, "disk failure", fatalErr.Error())
	})
}

func TestRegisterBrokenComponent(t *testing.T) {
	t.Parallel()

	baseDir := "/project/src"

	t.Run("should register a stub entry for a broken component", func(t *testing.T) {
		t.Parallel()

		resolver := newGraphBuilderMockResolver("my-project", baseDir)
		pathsConfig := AnnotatorPathsConfig{
			PagesSourceDir:    "pages",
			PartialsSourceDir: "partials",
			EmailsSourceDir:   "emails",
			E2ESourceDir:      "e2e",
		}
		builder := NewGraphBuilder(resolver, nil, nil, pathsConfig, true)

		graph := &annotator_dto.ComponentGraph{
			Components:       make(map[string]*annotator_dto.ParsedComponent),
			PathToHashedName: make(map[string]string),
			HashedNameToPath: make(map[string]string),
		}

		brokenPath := filepath.Join(baseDir, "pages", "broken.pk")
		builder.registerBrokenComponent(context.Background(), graph, brokenPath)

		hash, ok := graph.PathToHashedName[brokenPath]
		require.True(t, ok, "path should be registered in PathToHashedName")
		assert.NotEmpty(t, hash)

		comp, ok := graph.Components[hash]
		require.True(t, ok, "component should be registered in Components")
		assert.Equal(t, brokenPath, comp.SourcePath)
		assert.Equal(t, "page", comp.ComponentType)
	})

	t.Run("should mark external component as external", func(t *testing.T) {
		t.Parallel()

		resolver := newGraphBuilderMockResolver("my-project", baseDir)
		pathsConfig := AnnotatorPathsConfig{
			PagesSourceDir:    "pages",
			PartialsSourceDir: "partials",
			EmailsSourceDir:   "",
			E2ESourceDir:      "e2e",
		}
		builder := NewGraphBuilder(resolver, nil, nil, pathsConfig, true)

		graph := &annotator_dto.ComponentGraph{
			Components:       make(map[string]*annotator_dto.ParsedComponent),
			PathToHashedName: make(map[string]string),
			HashedNameToPath: make(map[string]string),
		}

		externalPath := "/home/user/go/pkg/mod/github.com/ext/lib@v1.0.0/card.pk"
		builder.registerBrokenComponent(context.Background(), graph, externalPath)

		hash := graph.PathToHashedName[externalPath]
		comp := graph.Components[hash]
		assert.True(t, comp.IsExternal)
	})
}

func TestDetermineComponentType(t *testing.T) {
	t.Parallel()

	baseDir := "/project/src"

	testCases := []struct {
		name         string
		absolutePath string
		pagesDir     string
		partialsDir  string
		emailsDir    string
		e2eDir       string
		expectedType string
	}{
		{
			name:         "should identify a page component",
			absolutePath: filepath.Join(baseDir, "pages", "home.pk"),
			pagesDir:     "pages",
			partialsDir:  "partials",
			emailsDir:    "emails",
			e2eDir:       "e2e",
			expectedType: "page",
		},
		{
			name:         "should identify a partial component",
			absolutePath: filepath.Join(baseDir, "partials", "card.pk"),
			pagesDir:     "pages",
			partialsDir:  "partials",
			emailsDir:    "emails",
			e2eDir:       "e2e",
			expectedType: "partial",
		},
		{
			name:         "should identify an email component",
			absolutePath: filepath.Join(baseDir, "emails", "welcome.pk"),
			pagesDir:     "pages",
			partialsDir:  "partials",
			emailsDir:    "emails",
			e2eDir:       "e2e",
			expectedType: "email",
		},
		{
			name:         "should identify an e2e page",
			absolutePath: filepath.Join(baseDir, "e2e", "pages", "test.pk"),
			pagesDir:     "pages",
			partialsDir:  "partials",
			emailsDir:    "emails",
			e2eDir:       "e2e",
			expectedType: "page",
		},
		{
			name:         "should identify an e2e partial",
			absolutePath: filepath.Join(baseDir, "e2e", "partials", "widget.pk"),
			pagesDir:     "pages",
			partialsDir:  "partials",
			emailsDir:    "emails",
			e2eDir:       "e2e",
			expectedType: "partial",
		},
		{
			name:         "should default to component when path does not match any directory",
			absolutePath: filepath.Join(baseDir, "lib", "helpers.pk"),
			pagesDir:     "pages",
			partialsDir:  "partials",
			emailsDir:    "emails",
			e2eDir:       "e2e",
			expectedType: "component",
		},
		{
			name:         "should default to component when emails directory is empty and path is in emails",
			absolutePath: filepath.Join(baseDir, "emails", "invite.pk"),
			pagesDir:     "pages",
			partialsDir:  "partials",
			emailsDir:    "",
			e2eDir:       "e2e",
			expectedType: "component",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolver := newGraphBuilderMockResolver("my-project", baseDir)
			pathsConfig := AnnotatorPathsConfig{
				PagesSourceDir:    tc.pagesDir,
				PartialsSourceDir: tc.partialsDir,
				EmailsSourceDir:   tc.emailsDir,
				E2ESourceDir:      tc.e2eDir,
			}
			builder := NewGraphBuilder(resolver, nil, nil, pathsConfig, false)

			result := builder.determineComponentType(context.Background(), tc.absolutePath)

			assert.Equal(t, tc.expectedType, result)
		})
	}
}

func TestReadAndParse(t *testing.T) {
	t.Parallel()

	baseDir := "/project/src"

	t.Run("should parse a valid file and return its component", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		filePath := filepath.Join(baseDir, "pages", "home.pk")
		h.fsReader.files[filePath] = []byte(`<template><div>Hello</div></template>`)

		comp, data, err := h.builder.readAndParse(context.Background(), filePath)

		require.NoError(t, err)
		assert.NotNil(t, comp)
		assert.NotNil(t, data)
		assert.Contains(t, string(data), "Hello")
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)

		comp, data, err := h.builder.readAndParse(context.Background(), "/nonexistent/file.pk")

		require.Error(t, err)
		assert.Nil(t, comp)
		assert.Nil(t, data)
	})

	t.Run("should return file data even when parsing fails", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		filePath := filepath.Join(baseDir, "bad.pk")
		h.fsReader.files[filePath] = []byte(`<template><div>{{ broken</template>`)

		comp, data, err := h.builder.readAndParse(context.Background(), filePath)

		require.Error(t, err)
		assert.Nil(t, comp, "component should be nil in strict mode")
		assert.NotNil(t, data, "file data should still be returned")
	})

	t.Run("should return partial component in fault-tolerant mode", func(t *testing.T) {
		t.Parallel()

		resolver := newGraphBuilderMockResolver("my-project", baseDir)
		fsReader := &mockFSReader{files: map[string][]byte{
			filepath.Join(baseDir, "bad.pk"): []byte(`<template><div>{{ broken</template><script type="application/x-go">package main</script>`),
		}}
		pathsConfig := AnnotatorPathsConfig{
			PagesSourceDir:    "pages",
			PartialsSourceDir: "partials",
			EmailsSourceDir:   "",
			E2ESourceDir:      "e2e",
		}
		builder := NewGraphBuilder(resolver, fsReader, &mockComponentCache{}, pathsConfig, true)

		comp, data, err := builder.readAndParse(context.Background(), filepath.Join(baseDir, "bad.pk"))

		require.Error(t, err, "should still return the parse error")
		assert.NotNil(t, comp, "partial component should be returned in fault-tolerant mode")
		assert.NotNil(t, data)
	})
}

func TestParseAllPaths(t *testing.T) {
	t.Parallel()

	baseDir := "/project/src"

	t.Run("should parse multiple files into a component graph", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		h.fsReader.files[filepath.Join(baseDir, "a.pk")] = []byte(`<template><div>A</div></template>`)
		h.fsReader.files[filepath.Join(baseDir, "b.pk")] = []byte(`<template><div>B</div></template>`)

		paths := []string{
			filepath.Join(baseDir, "a.pk"),
			filepath.Join(baseDir, "b.pk"),
		}

		graph, diagnostics, err := h.builder.parseAllPaths(context.Background(), paths)

		require.NoError(t, err)
		assert.Empty(t, diagnostics)
		assert.Len(t, graph.Components, 2)
		assert.Len(t, graph.AllSourceContents, 2)
	})

	t.Run("should handle single file", func(t *testing.T) {
		t.Parallel()

		h := newTestHarness(t, "my-project", baseDir)
		h.fsReader.files[filepath.Join(baseDir, "only.pk")] = []byte(`<script type="application/x-go">package only</script>`)

		paths := []string{filepath.Join(baseDir, "only.pk")}

		graph, diagnostics, err := h.builder.parseAllPaths(context.Background(), paths)

		require.NoError(t, err)
		assert.Empty(t, diagnostics)
		assert.Len(t, graph.Components, 1)
	})
}

type graphBuilderMockResolverWithFailures struct {
	failPaths  map[string]bool
	baseDir    string
	moduleName string
}

func (m *graphBuilderMockResolverWithFailures) DetectLocalModule(_ context.Context) error { return nil }
func (m *graphBuilderMockResolverWithFailures) GetBaseDir() string                        { return m.baseDir }
func (m *graphBuilderMockResolverWithFailures) GetModuleName() string {
	return m.moduleName
}
func (m *graphBuilderMockResolverWithFailures) ResolvePKPath(_ context.Context, importPath string, _ string) (string, error) {
	if m.failPaths[importPath] {
		return "", errors.New("failed to resolve: " + importPath)
	}
	if filepath.IsAbs(importPath) {
		return importPath, nil
	}
	return filepath.Join(m.baseDir, importPath), nil
}
func (m *graphBuilderMockResolverWithFailures) ResolveCSSPath(_ context.Context, importPath string, containingDir string) (string, error) {
	if filepath.IsAbs(importPath) {
		return importPath, nil
	}
	return filepath.Join(containingDir, importPath), nil
}
func (m *graphBuilderMockResolverWithFailures) ConvertEntryPointPathToManifestKey(entryPointPath string) string {
	moduleName := m.moduleName + "/"
	if result, found := strings.CutPrefix(entryPointPath, moduleName); found {
		return result
	}
	return entryPointPath
}
func (m *graphBuilderMockResolverWithFailures) ResolveAssetPath(_ context.Context, importPath string, _ string) (string, error) {
	return filepath.Join(m.baseDir, importPath), nil
}
func (*graphBuilderMockResolverWithFailures) GetModuleDir(_ context.Context, _ string) (string, error) {
	return "", errors.New("GetModuleDir not implemented")
}
func (*graphBuilderMockResolverWithFailures) FindModuleBoundary(_ context.Context, _ string) (string, string, error) {
	return "", "", errors.New("FindModuleBoundary not implemented")
}

var _ resolver_domain.ResolverPort = (*graphBuilderMockResolverWithFailures)(nil)

func TestShortHash(t *testing.T) {
	t.Parallel()

	t.Run("returns 8 character hex string", func(t *testing.T) {
		t.Parallel()

		result := shortHash("some/path/file.pk")

		assert.Len(t, result, 8)
		assert.Regexp(t, "^[0-9a-f]{8}$", result)
	})

	t.Run("same input produces same hash", func(t *testing.T) {
		t.Parallel()

		hash1 := shortHash("test/path")
		hash2 := shortHash("test/path")

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		t.Parallel()

		hash1 := shortHash("path/one")
		hash2 := shortHash("path/two")

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("empty string produces valid hash", func(t *testing.T) {
		t.Parallel()

		result := shortHash("")

		assert.Len(t, result, 8)
		assert.Regexp(t, "^[0-9a-f]{8}$", result)
	})
}
