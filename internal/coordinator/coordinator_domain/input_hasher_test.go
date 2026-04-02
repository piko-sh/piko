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

package coordinator_domain

import (
	"context"
	"encoding/hex"
	"io/fs"
	"sync"
	"testing"
	"time"

	"path/filepath"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

type mockFileHashCache struct {
	persistErr error
	entries    map[string]fileHashCacheEntry
	getCalls   int
	setCalls   int
	loadCalls  int
	mu         sync.Mutex
}

type fileHashCacheEntry struct {
	modTime time.Time
	hash    string
}

func newMockFileHashCache() *mockFileHashCache {
	return &mockFileHashCache{
		entries: make(map[string]fileHashCacheEntry),
	}
}

func (m *mockFileHashCache) Get(_ context.Context, path string, modTime time.Time) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	entry, ok := m.entries[path]
	if !ok {
		return "", false
	}
	if !entry.modTime.Equal(modTime) {
		return "", false
	}
	return entry.hash, true
}

func (m *mockFileHashCache) Set(_ context.Context, path string, modTime time.Time, hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	m.entries[path] = fileHashCacheEntry{hash: hash, modTime: modTime}
}

func (m *mockFileHashCache) Load(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadCalls++
	return nil
}

func (m *mockFileHashCache) Persist(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.persistErr
}

func TestIsRelevantGoFile(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "regular go file",
			filename: "main.go",
			expected: true,
		},
		{
			name:     "test file",
			filename: "main_test.go",
			expected: false,
		},
		{
			name:     "not a go file",
			filename: "readme.md",
			expected: false,
		},
		{
			name:     "go file in path",
			filename: "handler.go",
			expected: true,
		},
		{
			name:     "test file with prefix",
			filename: "something_test.go",
			expected: false,
		},
		{
			name:     "file ending in go but not go extension",
			filename: "mango",
			expected: false,
		},
		{
			name:     "empty string",
			filename: "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isRelevantGoFile(tc.filename)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHandleDirEntry(t *testing.T) {
	s := &coordinatorService{}

	testCases := []struct {
		name        string
		dirname     string
		description string
		expectSkip  bool
	}{
		{
			name:        "regular directory",
			dirname:     "src",
			expectSkip:  false,
			description: "Regular directories should not be skipped",
		},
		{
			name:        "hidden directory (git)",
			dirname:     ".git",
			expectSkip:  true,
			description: "Hidden directories should be skipped",
		},
		{
			name:        "hidden directory (vscode)",
			dirname:     ".vscode",
			expectSkip:  true,
			description: "IDE directories should be skipped",
		},
		{
			name:        "vendor directory",
			dirname:     "vendor",
			expectSkip:  true,
			description: "Vendor directory should be skipped",
		},
		{
			name:        "dist directory",
			dirname:     "dist",
			expectSkip:  true,
			description: "Build output directory should be skipped",
		},
		{
			name:        "node_modules directory",
			dirname:     "node_modules",
			expectSkip:  true,
			description: "Node modules should be skipped",
		},
		{
			name:        "components directory",
			dirname:     "components",
			expectSkip:  false,
			description: "Source directories should not be skipped",
		},
		{
			name:        "hidden directory (idea)",
			dirname:     ".idea",
			expectSkip:  true,
			description: "IDE directories should be skipped",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.handleDirEntry(tc.dirname)
			if tc.expectSkip {
				assert.ErrorIs(t, err, fs.SkipDir, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}
		})
	}
}

func TestGetEntrypointPaths(t *testing.T) {
	t.Run("empty entry points", func(t *testing.T) {
		result := getEntrypointPaths(nil)
		assert.Empty(t, result)
	})

	t.Run("single entry point", func(t *testing.T) {
		entryPoints := []annotator_dto.EntryPoint{
			{Path: "/project/pages/index.pk", IsPage: true},
		}
		result := getEntrypointPaths(entryPoints)
		require.Len(t, result, 1)
		assert.Equal(t, "/project/pages/index.pk", result[0])
	})

	t.Run("multiple entry points", func(t *testing.T) {
		entryPoints := []annotator_dto.EntryPoint{
			{Path: "/project/pages/index.pk", IsPage: true},
			{Path: "/project/pages/about.pk", IsPage: true},
			{Path: "/project/components/header.pk", IsPage: false},
		}
		result := getEntrypointPaths(entryPoints)
		require.Len(t, result, 3)
		assert.Equal(t, "/project/pages/index.pk", result[0])
		assert.Equal(t, "/project/pages/about.pk", result[1])
		assert.Equal(t, "/project/components/header.pk", result[2])
	})
}

func TestGetEffectiveResolver(t *testing.T) {
	defaultResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/default" }}
	overrideResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/override" }}

	s := &coordinatorService{
		resolver: defaultResolver,
	}

	t.Run("nil build options returns default resolver", func(t *testing.T) {
		result := s.getEffectiveResolver(nil)
		assert.Same(t, defaultResolver, result)
	})

	t.Run("build options without resolver returns default", func(t *testing.T) {
		result := s.getEffectiveResolver(&buildOptions{})
		assert.Same(t, defaultResolver, result)
	})

	t.Run("build options with resolver override", func(t *testing.T) {
		result := s.getEffectiveResolver(&buildOptions{Resolver: overrideResolver})
		assert.Same(t, overrideResolver, result)
	})
}

func TestCollectHashResults(t *testing.T) {
	t.Run("empty results channel", func(t *testing.T) {
		results := make(chan fileHashResult)
		close(results)

		hashes, contents, err := collectHashResults(results, 0)
		require.NoError(t, err)
		assert.Empty(t, hashes)
		assert.Empty(t, contents)
	})

	t.Run("single result", func(t *testing.T) {
		results := make(chan fileHashResult, 1)
		results <- fileHashResult{
			path:    "/test/file.go",
			hash:    "abc123",
			content: []byte("package main"),
		}
		close(results)

		hashes, contents, err := collectHashResults(results, 1)
		require.NoError(t, err)
		require.Len(t, hashes, 1)
		require.Len(t, contents, 1)
		assert.Equal(t, "abc123", hashes["/test/file.go"])
		assert.Equal(t, []byte("package main"), contents["/test/file.go"])
	})

	t.Run("multiple results", func(t *testing.T) {
		results := make(chan fileHashResult, 3)
		results <- fileHashResult{path: "/a.go", hash: "hash1", content: []byte("a")}
		results <- fileHashResult{path: "/b.go", hash: "hash2", content: []byte("b")}
		results <- fileHashResult{path: "/c.go", hash: "hash3", content: []byte("c")}
		close(results)

		hashes, contents, err := collectHashResults(results, 3)
		require.NoError(t, err)
		require.Len(t, hashes, 3)
		require.Len(t, contents, 3)
		assert.Equal(t, "hash1", hashes["/a.go"])
		assert.Equal(t, "hash2", hashes["/b.go"])
		assert.Equal(t, "hash3", hashes["/c.go"])
	})
}

func TestExtractScriptBlockContent(t *testing.T) {
	t.Run("pk file with script block", func(t *testing.T) {
		content := []byte(`<script lang="go">
package main

func Handler() {}
</script>

<template>
<div>Hello</div>
</template>`)

		scriptContent, scriptHash, err := extractScriptBlockContent("/test.pk", content)
		require.NoError(t, err)
		assert.Contains(t, scriptContent, "package main")
		assert.Contains(t, scriptContent, "func Handler()")
		assert.NotEmpty(t, scriptHash)
	})

	t.Run("pk file without script block", func(t *testing.T) {
		content := []byte(`<template>
<div>Hello</div>
</template>`)

		scriptContent, scriptHash, err := extractScriptBlockContent("/test.pk", content)
		require.NoError(t, err)
		assert.Empty(t, scriptContent)
		assert.Empty(t, scriptHash)
	})

	t.Run("hash is deterministic", func(t *testing.T) {
		content := []byte(`<script lang="go">
package main
</script>`)

		_, hash1, err1 := extractScriptBlockContent("/test.pk", content)
		_, hash2, err2 := extractScriptBlockContent("/test.pk", content)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, hash1, hash2, "Same content should produce same hash")
	})

	t.Run("different paths produce different hashes", func(t *testing.T) {
		content := []byte(`<script lang="go">
package main
</script>`)

		_, hash1, err1 := extractScriptBlockContent("/path/a.pk", content)
		_, hash2, err2 := extractScriptBlockContent("/path/b.pk", content)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, hash1, hash2, "Different paths should produce different hashes")
	})
}

func TestHashIntrospectionContent(t *testing.T) {
	t.Run("pk files hash only script content", func(t *testing.T) {
		hasher := xxhash.New()
		paths := []string{"/test.pk"}
		contents := map[string][]byte{
			"/test.pk": []byte(`<script lang="go">
package main
</script>
<template>
<div>This should not affect hash</div>
</template>`),
		}

		scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
		require.NoError(t, err)
		assert.Len(t, scriptHashes, 1)
		assert.Contains(t, scriptHashes, "/test.pk")
	})

	t.Run("go files hash entire content", func(t *testing.T) {
		hasher := xxhash.New()
		paths := []string{"/main.go"}
		contents := map[string][]byte{
			"/main.go": []byte("package main\n\nfunc main() {}"),
		}

		scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
		require.NoError(t, err)
		assert.Empty(t, scriptHashes, "Go files should not produce script hashes")

		hash := hex.EncodeToString(hasher.Sum(nil))
		assert.NotEmpty(t, hash)
	})

	t.Run("mixed pk and go files", func(t *testing.T) {
		hasher := xxhash.New()
		paths := []string{"/main.go", "/page.pk"}
		contents := map[string][]byte{
			"/main.go": []byte("package main"),
			"/page.pk": []byte(`<script lang="go">
package page
</script>`),
		}

		scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
		require.NoError(t, err)
		assert.Len(t, scriptHashes, 1)
		assert.Contains(t, scriptHashes, "/page.pk")
	})

	t.Run("pk file without script block", func(t *testing.T) {
		hasher := xxhash.New()
		paths := []string{"/template-only.pk"}
		contents := map[string][]byte{
			"/template-only.pk": []byte(`<template>
<div>No script</div>
</template>`),
		}

		scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
		require.NoError(t, err)
		assert.Empty(t, scriptHashes)
	})

	t.Run("case insensitive file extension", func(t *testing.T) {
		hasher := xxhash.New()
		paths := []string{"/Page.PK", "/Main.GO"}
		contents := map[string][]byte{
			"/Page.PK": []byte(`<script lang="go">
package page
</script>`),
			"/Main.GO": []byte("package main"),
		}

		scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
		require.NoError(t, err)
		assert.Len(t, scriptHashes, 1)
	})
}

func TestHashDeterminism(t *testing.T) {
	t.Run("xxhash produces consistent results", func(t *testing.T) {
		content := []byte("test content for hashing")

		hasher1 := xxhash.New()
		_, _ = hasher1.Write(content)
		hash1 := hex.EncodeToString(hasher1.Sum(nil))

		hasher2 := xxhash.New()
		_, _ = hasher2.Write(content)
		hash2 := hex.EncodeToString(hasher2.Sum(nil))

		assert.Equal(t, hash1, hash2, "Same content should always produce same hash")
	})

	t.Run("order of writes affects hash", func(t *testing.T) {
		hasher1 := xxhash.New()
		_, _ = hasher1.WriteString("/a.go")
		_, _ = hasher1.WriteString("hash_a")
		_, _ = hasher1.WriteString("/b.go")
		_, _ = hasher1.WriteString("hash_b")
		hash1 := hex.EncodeToString(hasher1.Sum(nil))

		hasher2 := xxhash.New()
		_, _ = hasher2.WriteString("/b.go")
		_, _ = hasher2.WriteString("hash_b")
		_, _ = hasher2.WriteString("/a.go")
		_, _ = hasher2.WriteString("hash_a")
		hash2 := hex.EncodeToString(hasher2.Sum(nil))

		assert.NotEqual(t, hash1, hash2, "Different order should produce different hash")
	})
}

func TestFileHashCacheIntegration(t *testing.T) {
	t.Run("cache miss triggers file read", func(t *testing.T) {
		cache := newMockFileHashCache()
		ctx := context.Background()
		modTime := time.Now()

		hash, found := cache.Get(ctx, "/test.go", modTime)
		assert.False(t, found)
		assert.Empty(t, hash)
		assert.Equal(t, 1, cache.getCalls)
	})

	t.Run("cache hit returns stored hash", func(t *testing.T) {
		cache := newMockFileHashCache()
		ctx := context.Background()
		modTime := time.Now()

		cache.Set(ctx, "/test.go", modTime, "cached_hash")
		hash, found := cache.Get(ctx, "/test.go", modTime)

		assert.True(t, found)
		assert.Equal(t, "cached_hash", hash)
	})

	t.Run("cache miss when modtime differs", func(t *testing.T) {
		cache := newMockFileHashCache()
		ctx := context.Background()
		originalTime := time.Now()
		newTime := originalTime.Add(time.Second)

		cache.Set(ctx, "/test.go", originalTime, "cached_hash")
		hash, found := cache.Get(ctx, "/test.go", newTime)

		assert.False(t, found)
		assert.Empty(t, hash)
	})
}

func TestHasherDependencyDiscoverer(t *testing.T) {
	t.Run("discovers entry points", func(t *testing.T) {

		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
			},
		}
		resolver := &resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		}

		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: fsReader,
		}

		paths, err := discoverer.Discover(context.Background(), []string{"test-module/pages/index.pk"})
		require.NoError(t, err)
		assert.Len(t, paths, 1)
	})

	t.Run("discovers imports from pk files", func(t *testing.T) {

		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages

import "test-module/components/header.pk"
</script>`),
				"/project/components/header.pk": []byte(`<script lang="go">
package components
</script>`),
			},
		}
		resolver := &resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		}

		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: fsReader,
		}

		paths, err := discoverer.Discover(context.Background(), []string{"test-module/pages/index.pk"})
		require.NoError(t, err)
		assert.Len(t, paths, 2)
	})

	t.Run("ignores non-pk imports", func(t *testing.T) {
		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages

import "fmt"
import "github.com/some/package"
</script>`),
			},
		}
		resolver := &resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		}

		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: fsReader,
		}

		paths, err := discoverer.Discover(context.Background(), []string{"test-module/pages/index.pk"})
		require.NoError(t, err)
		assert.Len(t, paths, 1, "Should only include the entry point, not non-pk imports")
	})

	t.Run("handles pk file without script block", func(t *testing.T) {
		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<template>
<div>No script</div>
</template>`),
			},
		}
		resolver := &resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		}

		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: fsReader,
		}

		paths, err := discoverer.Discover(context.Background(), []string{"test-module/pages/index.pk"})
		require.NoError(t, err)
		assert.Len(t, paths, 1)
	})
}

func TestExtractPKImports(t *testing.T) {
	t.Run("extracts pk imports", func(t *testing.T) {
		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/test.pk": []byte(`<script lang="go">
package test

import (
	"fmt"
	"test-module/components/header.pk"
	"test-module/components/footer.pk"
)
</script>`),
			},
		}

		discoverer := &hasherDependencyDiscoverer{
			fsReader: fsReader,
		}

		imports, err := discoverer.extractPKImports(context.Background(), "/test.pk")
		require.NoError(t, err)
		assert.Len(t, imports, 2)
		assert.Contains(t, imports, "test-module/components/header.pk")
		assert.Contains(t, imports, "test-module/components/footer.pk")
	})

	t.Run("returns empty for no imports", func(t *testing.T) {
		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/test.pk": []byte(`<script lang="go">
package test
</script>`),
			},
		}

		discoverer := &hasherDependencyDiscoverer{
			fsReader: fsReader,
		}

		imports, err := discoverer.extractPKImports(context.Background(), "/test.pk")
		require.NoError(t, err)
		assert.Empty(t, imports)
	})

	t.Run("returns nil for no script block", func(t *testing.T) {
		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/test.pk": []byte(`<template>
<div>No script</div>
</template>`),
			},
		}

		discoverer := &hasherDependencyDiscoverer{
			fsReader: fsReader,
		}

		imports, err := discoverer.extractPKImports(context.Background(), "/test.pk")
		require.NoError(t, err)
		assert.Nil(t, imports)
	})

	t.Run("case insensitive pk extension matching", func(t *testing.T) {
		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/test.pk": []byte(`<script lang="go">
package test

import "test-module/Component.PK"
</script>`),
			},
		}

		discoverer := &hasherDependencyDiscoverer{
			fsReader: fsReader,
		}

		imports, err := discoverer.extractPKImports(context.Background(), "/test.pk")
		require.NoError(t, err)
		assert.Len(t, imports, 1)
		assert.Contains(t, imports, "test-module/Component.PK")
	})
}
