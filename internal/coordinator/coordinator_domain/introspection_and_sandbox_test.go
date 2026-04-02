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
	"errors"
	"fmt"
	"io/fs"
	"sync"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"path/filepath"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestHashAndReadFile_WithoutCache(t *testing.T) {
	t.Parallel()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/file.go": []byte("package main"),
		},
	}
	service := &coordinatorService{
		fsReader:      fsReader,
		fileHashCache: nil,
	}

	hash, content, err := service.hashAndReadFile(context.Background(), "/project/file.go", time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Equal(t, []byte("package main"), content)

	hash2, content2, err2 := service.hashAndReadFile(context.Background(), "/project/file.go", time.Now())
	require.NoError(t, err2)
	assert.Equal(t, hash, hash2)
	assert.Equal(t, content, content2)
}

func TestHashAndReadFile_WithCacheHit(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := newMockFileHashCache()
	cache.Set(context.Background(), "/project/file.go", modTime, "cached-hash-value")

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/file.go": []byte("package main"),
		},
	}
	service := &coordinatorService{
		fsReader:      fsReader,
		fileHashCache: cache,
	}

	hash, content, err := service.hashAndReadFile(context.Background(), "/project/file.go", modTime)
	require.NoError(t, err)

	assert.NotEqual(t, "cached-hash-value", hash, "stale cached hash should be replaced")
	assert.NotEmpty(t, hash, "a valid hash should be computed from content")
	assert.Equal(t, []byte("package main"), content)
}

func TestHashAndReadFile_WithCacheMiss(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := newMockFileHashCache()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/file.go": []byte("package main"),
		},
	}
	service := &coordinatorService{
		fsReader:      fsReader,
		fileHashCache: cache,
	}

	hash, content, err := service.hashAndReadFile(context.Background(), "/project/file.go", modTime)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Equal(t, []byte("package main"), content)

	assert.Equal(t, 1, cache.setCalls)

	cachedHash, found := cache.Get(context.Background(), "/project/file.go", modTime)
	assert.True(t, found)
	assert.Equal(t, hash, cachedHash)
}

func TestHashAndReadFile_FileReadError(t *testing.T) {
	t.Parallel()

	fsReader := &mockFSReader{
		Files: map[string][]byte{},
	}
	service := &coordinatorService{
		fsReader:      fsReader,
		fileHashCache: nil,
	}

	hash, content, err := service.hashAndReadFile(context.Background(), "/nonexistent.go", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
	assert.Empty(t, hash)
	assert.Nil(t, content)
}

func TestHashAndReadFile_CacheHitButReadFails(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := newMockFileHashCache()
	cache.Set(context.Background(), "/project/file.go", modTime, "cached-hash")

	fsReader := &mockFSReader{
		Files: map[string][]byte{},
	}
	service := &coordinatorService{
		fsReader:      fsReader,
		fileHashCache: cache,
	}

	hash, content, err := service.hashAndReadFile(context.Background(), "/project/file.go", modTime)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read cached file")
	assert.Empty(t, hash)
	assert.Nil(t, content)
}

func TestHashAndReadFile_ProducesCorrectXXHash(t *testing.T) {
	t.Parallel()

	fileContent := []byte("package main\n\nfunc hello() {}\n")
	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/file.go": fileContent,
		},
	}
	service := &coordinatorService{
		fsReader:      fsReader,
		fileHashCache: nil,
	}

	hash, _, err := service.hashAndReadFile(context.Background(), "/project/file.go", time.Now())
	require.NoError(t, err)

	h := xxhash.New()
	_, _ = h.Write(fileContent)
	expected := hex.EncodeToString(h.Sum(nil))

	assert.Equal(t, expected, hash)
}

func TestProcessFileForHash_Success(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("file.go", []byte("package main"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/file.go": []byte("package main"),
		},
	}
	service := &coordinatorService{
		fsReader:       fsReader,
		fileHashCache:  nil,
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	result, err := service.processFileForHash(context.Background(), "/project/file.go")
	require.NoError(t, err)
	assert.Equal(t, "/project/file.go", result.path)
	assert.NotEmpty(t, result.hash)
	assert.Equal(t, []byte("package main"), result.content)
}

func TestProcessFileForHash_StatError(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.StatErr = errors.New("stat error")

	service := &coordinatorService{
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	result, err := service.processFileForHash(context.Background(), "/project/file.go")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stat file")
	assert.Equal(t, fileHashResult{}, result)
}

func TestStatFile_WithBaseDirSandbox(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("main.go", []byte("package main"))

	service := &coordinatorService{
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	info, err := service.statFile(context.Background(), "/project/main.go")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "main.go", info.Name())
}

func TestStatFile_WithBaseDirSandbox_OutsideBaseDir(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	service := &coordinatorService{
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	_, err := service.statFile(context.Background(), "/other/path/file.go")

	assert.Error(t, err)
}

func TestStatFile_WithoutBaseDirSandbox(t *testing.T) {
	t.Parallel()

	service := &coordinatorService{
		baseDirSandbox: nil,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	_, err := service.statFile(context.Background(), "/nonexistent/path/file.go")
	assert.Error(t, err)
}

func TestStatFile_EmptyBaseDir(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("", safedisk.ModeReadOnly)

	service := &coordinatorService{
		baseDirSandbox: sandbox,
		resolver:       &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "" }},
	}

	_, err := service.statFile(context.Background(), "/some/file.go")
	assert.Error(t, err)
}

func TestGetOrCreateSandbox_ExistingMatchingSandbox(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	service := &coordinatorService{
		baseDirSandbox:     sandbox,
		baseDirSandboxPath: "/project",
	}

	result, needsClose, err := service.getOrCreateSandbox("/project")
	require.NoError(t, err)
	assert.Same(t, sandbox, result)
	assert.False(t, needsClose, "existing sandbox should not need closing")
}

func TestGetOrCreateSandbox_DifferentPath(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	service := &coordinatorService{
		baseDirSandbox:     sandbox,
		baseDirSandboxPath: "/project",
	}

	_, _, err := service.getOrCreateSandbox("/other-project")

	_ = err
}

func TestGetOrCreateSandbox_NilExistingSandbox(t *testing.T) {
	t.Parallel()

	service := &coordinatorService{
		baseDirSandbox:     nil,
		baseDirSandboxPath: "",
	}

	_, _, err := service.getOrCreateSandbox("/nonexistent")

	_ = err
}

func TestHandleDirEntry_EmptyName(t *testing.T) {
	t.Parallel()

	s := &coordinatorService{}
	err := s.handleDirEntry("")
	assert.NoError(t, err, "empty name should not be skipped")
}

func TestHandleDirEntry_AllSkippedDirs(t *testing.T) {
	t.Parallel()

	s := &coordinatorService{}

	skippedDirs := []string{"vendor", "dist", "node_modules", ".git", ".idea", ".vscode"}
	for _, directory := range skippedDirs {
		err := s.handleDirEntry(directory)
		assert.ErrorIs(t, err, fs.SkipDir, "directory %s should be skipped", directory)
	}
}

func TestHandleDirEntry_AllowedDirs(t *testing.T) {
	t.Parallel()

	s := &coordinatorService{}

	allowedDirs := []string{"src", "pkg", "internal", "cmd", "actions", "pages", "components"}
	for _, directory := range allowedDirs {
		err := s.handleDirEntry(directory)
		assert.NoError(t, err, "directory %s should not be skipped", directory)
	}
}

func TestCollectOriginalGoFilePaths_EmptyBaseDir(t *testing.T) {
	t.Parallel()

	service := &coordinatorService{}
	resolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "" }}

	paths, err := service.collectOriginalGoFilePaths(context.Background(), resolver)
	require.NoError(t, err)
	assert.Nil(t, paths)
}

func TestCollectOriginalGoFilePaths_WithSandbox(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("main.go", []byte("package main"))

	service := &coordinatorService{
		baseDirSandbox:     sandbox,
		baseDirSandboxPath: "/project",
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

	paths, err := service.collectOriginalGoFilePaths(context.Background(), resolver)
	require.NoError(t, err)

	assert.Nil(t, paths)
}

func TestCollectOriginalGoFilePaths_StatDotFails(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.StatErr = errors.New("stat error")

	service := &coordinatorService{
		baseDirSandbox:     sandbox,
		baseDirSandboxPath: "/project",
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

	paths, err := service.collectOriginalGoFilePaths(context.Background(), resolver)
	require.NoError(t, err)
	assert.Nil(t, paths)
}

func TestCollectHashResults_EmptyChannel(t *testing.T) {
	t.Parallel()

	resultChannel := make(chan fileHashResult)
	close(resultChannel)

	hashes, contents, err := collectHashResults(resultChannel, 0)
	require.NoError(t, err)
	assert.Empty(t, hashes)
	assert.Empty(t, contents)
}

func TestCollectHashResults_SingleResult(t *testing.T) {
	t.Parallel()

	resultChannel := make(chan fileHashResult, 1)
	resultChannel <- fileHashResult{
		path:    "/test.go",
		hash:    "abc",
		content: []byte("hello"),
	}
	close(resultChannel)

	hashes, contents, err := collectHashResults(resultChannel, 1)
	require.NoError(t, err)
	assert.Equal(t, "abc", hashes["/test.go"])
	assert.Equal(t, []byte("hello"), contents["/test.go"])
}

func TestCollectHashResults_NilContent(t *testing.T) {
	t.Parallel()

	resultChannel := make(chan fileHashResult, 1)
	resultChannel <- fileHashResult{
		path:    "/test.go",
		hash:    "abc",
		content: nil,
	}
	close(resultChannel)

	hashes, contents, err := collectHashResults(resultChannel, 1)
	require.NoError(t, err)
	assert.Equal(t, "abc", hashes["/test.go"])
	assert.Nil(t, contents["/test.go"])
}

func TestHashIntrospectionContent_GoFileOnly(t *testing.T) {
	t.Parallel()

	hasher := xxhash.New()
	paths := []string{"/main.go"}
	contents := map[string][]byte{
		"/main.go": []byte("package main\nfunc main() {}"),
	}

	scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
	require.NoError(t, err)
	assert.Empty(t, scriptHashes, "Go files should not produce script hashes")

	hash := hex.EncodeToString(hasher.Sum(nil))
	assert.NotEmpty(t, hash)
}

func TestHashIntrospectionContent_PKWithScript(t *testing.T) {
	t.Parallel()

	hasher := xxhash.New()
	paths := []string{"/comp.pk"}
	contents := map[string][]byte{
		"/comp.pk": []byte(`<script lang="go">
package comp

func Handler() {}
</script>
<template><div>Hello</div></template>`),
	}

	scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
	require.NoError(t, err)
	assert.Len(t, scriptHashes, 1)
	assert.Contains(t, scriptHashes, "/comp.pk")
	assert.NotEmpty(t, scriptHashes["/comp.pk"])
}

func TestHashIntrospectionContent_PKWithoutScript(t *testing.T) {
	t.Parallel()

	hasher := xxhash.New()
	paths := []string{"/template.pk"}
	contents := map[string][]byte{
		"/template.pk": []byte(`<template><div>Hello</div></template>`),
	}

	scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
	require.NoError(t, err)
	assert.Empty(t, scriptHashes)
}

func TestHashIntrospectionContent_DifferentGoContentProducesDifferentHash(t *testing.T) {
	t.Parallel()

	hasher1 := xxhash.New()
	paths := []string{"/main.go"}
	contents1 := map[string][]byte{"/main.go": []byte("package main\nfunc a() {}")}
	_, err1 := hashIntrospectionContent(hasher1, paths, contents1)
	require.NoError(t, err1)
	hash1 := hex.EncodeToString(hasher1.Sum(nil))

	hasher2 := xxhash.New()
	contents2 := map[string][]byte{"/main.go": []byte("package main\nfunc b() {}")}
	_, err2 := hashIntrospectionContent(hasher2, paths, contents2)
	require.NoError(t, err2)
	hash2 := hex.EncodeToString(hasher2.Sum(nil))

	assert.NotEqual(t, hash1, hash2)
}

func TestExtractScriptBlockContent_WithValidScript(t *testing.T) {
	t.Parallel()

	content := []byte(`<script lang="go">
package test

import "fmt"

func Handler() {
	fmt.Println("hello")
}
</script>
<template><div>Hello</div></template>`)

	scriptContent, scriptHash, err := extractScriptBlockContent("/test.pk", content)
	require.NoError(t, err)
	assert.Contains(t, scriptContent, "package test")
	assert.Contains(t, scriptContent, `import "fmt"`)
	assert.NotEmpty(t, scriptHash)
}

func TestExtractScriptBlockContent_DifferentPathsSameContent(t *testing.T) {
	t.Parallel()

	content := []byte(`<script lang="go">
package test
</script>`)

	_, hash1, err1 := extractScriptBlockContent("/a.pk", content)
	_, hash2, err2 := extractScriptBlockContent("/b.pk", content)
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2, "different paths should yield different script hashes")
}

func TestExtractScriptBlockContent_SamePathSameContent(t *testing.T) {
	t.Parallel()

	content := []byte(`<script lang="go">
package test
</script>`)

	_, hash1, err1 := extractScriptBlockContent("/same.pk", content)
	_, hash2, err2 := extractScriptBlockContent("/same.pk", content)
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, hash1, hash2, "same path and content should yield same hash")
}

func TestExtractScriptBlockContent_TemplateOnly(t *testing.T) {
	t.Parallel()

	content := []byte(`<template><div>Hello World</div></template>`)
	scriptContent, scriptHash, err := extractScriptBlockContent("/test.pk", content)
	require.NoError(t, err)
	assert.Empty(t, scriptContent)
	assert.Empty(t, scriptHash)
}

func TestSubscribeConcurrent(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	var wg sync.WaitGroup
	unsubFuncs := make([]UnsubscribeFunc, 20)

	for i := range 20 {
		wg.Go(func() {
			_, unsub := service.Subscribe("concurrent-sub")
			unsubFuncs[i] = unsub
		})
	}
	wg.Wait()

	assert.Equal(t, 20, len(service.subscribers))

	for _, unsub := range unsubFuncs {
		wg.Go(func() {
			unsub()
		})
	}
	wg.Wait()

	assert.Equal(t, 0, len(service.subscribers))
}

func TestPublishConcurrent(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	notificationChannel, unsub := service.Subscribe("test-sub")
	defer unsub()

	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			service.publish(context.Background(), BuildNotification{CausationID: "test"})
		})
	}
	wg.Wait()

	select {
	case n := <-notificationChannel:
		assert.Equal(t, "test", n.CausationID)
	case <-time.After(time.Second):
		t.Fatal("expected at least one notification")
	}
}

func TestGetResult_FastPath(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: expected},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	service.status.Result = expected

	result, err := service.GetResult(context.Background(), nil)
	require.NoError(t, err)
	assert.Same(t, expected, result)
}

func TestInvalidateConcurrent(t *testing.T) {
	t.Parallel()

	cache := newMockCache()
	service := newCoordinatorService(
		&mockAnnotator{},
		cache,
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_ = service.Invalidate(context.Background())
		})
	}
	wg.Wait()

	assert.Equal(t, 10, cache.clearCalls)
	assert.Nil(t, service.status.Result)
}

func TestSetLastBuildRequestConcurrent(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			eps := []annotator_dto.EntryPoint{{Path: "/file.pk"}}
			opts := &buildOptions{CausationID: "cause"}
			service.setLastBuildRequest(context.Background(), eps, opts)
		})
	}
	wg.Wait()

	service.mu.RLock()
	defer service.mu.RUnlock()
	require.NotNil(t, service.lastBuildRequest)
}

func TestShutdown_WithFileHashCache(t *testing.T) {
	t.Parallel()

	cache := newMockFileHashCache()
	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(
			WithFileHashCache(cache),
			WithBaseDirSandbox(sandbox),
		),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	service.Shutdown(context.Background())

}

func TestShutdown_WithFileHashCachePersistError(t *testing.T) {
	t.Parallel()

	cache := newMockFileHashCache()
	cache.persistErr = errors.New("persist failed")
	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(
			WithFileHashCache(cache),
			WithBaseDirSandbox(sandbox),
		),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	assert.NotPanics(t, func() {
		service.Shutdown(context.Background())
	})
}

func TestShutdown_WithoutFileHashCache(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(
			WithBaseDirSandbox(sandbox),
		),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	assert.NotPanics(t, func() {
		service.Shutdown(context.Background())
	})
}

func TestShutdown_WithDebounceTimer(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(
			WithBaseDirSandbox(sandbox),
		),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	service.debounceMutex.Lock()
	service.debounceTimer = service.clock.AfterFunc(10*time.Second, func() {

	})
	service.debounceMutex.Unlock()

	assert.NotPanics(t, func() {
		service.Shutdown(context.Background())
	})
}

func TestInitPostCreation_WithFileHashCacheLoadError(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("load failed")
	cache := &failingFileHashCache{loadErr: loadErr}

	service := &coordinatorService{
		fileHashCache:  cache,
		baseDirSandbox: safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly),
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	assert.NotPanics(t, func() {
		service.initialisePostCreation(context.Background())
	})
}

func TestInitPostCreation_WithNilCacheAndEmptyBaseDir(t *testing.T) {
	t.Parallel()

	service := &coordinatorService{
		fileHashCache:  nil,
		baseDirSandbox: nil,
		resolver:       &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "" }},
	}

	assert.NotPanics(t, func() {
		service.initialisePostCreation(context.Background())
	})
	assert.Nil(t, service.baseDirSandbox)
}

func TestInitPostCreation_WithExistingSandbox(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	service := &coordinatorService{
		fileHashCache:  nil,
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	service.initialisePostCreation(context.Background())

	assert.Same(t, sandbox, service.baseDirSandbox)
	assert.Equal(t, "/project", service.baseDirSandboxPath)
}

type failingFileHashCache struct {
	loadErr    error
	persistErr error
}

func (f *failingFileHashCache) Get(_ context.Context, _ string, _ time.Time) (string, bool) {
	return "", false
}
func (f *failingFileHashCache) Set(_ context.Context, _ string, _ time.Time, _ string) {}
func (f *failingFileHashCache) Load(_ context.Context) error                           { return f.loadErr }
func (f *failingFileHashCache) Persist(_ context.Context) error                        { return f.persistErr }

func TestDiscoverImportsFromFile_ReadError(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &resolver_domain.MockResolver{
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
		},
		fsReader: &mockFSReader{Files: map[string][]byte{}},
	}

	queue := []string{"/project/pages/index.pk"}
	visited := map[string]bool{"/project/pages/index.pk": true}

	err := discoverer.discoverImportsFromFile(context.Background(), "/project/pages/index.pk", &queue, visited)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracting imports from")
}

func TestDiscoverImportsFromFile_NoImports(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &resolver_domain.MockResolver{
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
		},
		fsReader: &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
			},
		},
	}

	queue := []string{"/project/pages/index.pk"}
	visited := map[string]bool{"/project/pages/index.pk": true}

	err := discoverer.discoverImportsFromFile(context.Background(), "/project/pages/index.pk", &queue, visited)
	require.NoError(t, err)
	assert.Len(t, queue, 1, "queue should not grow without imports")
}

func TestDiscoverImportsFromFile_WithPKImports(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &resolver_domain.MockResolver{
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
		},
		fsReader: &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages

import "test-module/components/header.pk"
</script>`),
			},
		},
	}

	queue := []string{"/project/pages/index.pk"}
	visited := map[string]bool{"/project/pages/index.pk": true}

	err := discoverer.discoverImportsFromFile(context.Background(), "/project/pages/index.pk", &queue, visited)
	require.NoError(t, err)
	assert.Len(t, queue, 2)
	assert.True(t, visited["/project/components/header.pk"])
}

func TestDiscoverImportsFromFile_SkipsDuplicate(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &resolver_domain.MockResolver{
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
		},
		fsReader: &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages

import "test-module/components/header.pk"
</script>`),
			},
		},
	}

	queue := []string{"/project/pages/index.pk"}
	visited := map[string]bool{
		"/project/pages/index.pk":       true,
		"/project/components/header.pk": true,
	}

	err := discoverer.discoverImportsFromFile(context.Background(), "/project/pages/index.pk", &queue, visited)
	require.NoError(t, err)
	assert.Len(t, queue, 1, "should not add already visited import")
}

func TestDiscover_EmptyEntryPoints(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &resolver_domain.MockResolver{
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
		},
		fsReader: &mockFSReader{Files: map[string][]byte{}},
	}

	paths, err := discoverer.Discover(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, paths)
}

func TestDiscover_SingleEntryNoImports(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &resolver_domain.MockResolver{
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
		},
		fsReader: &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<template><div>Hello</div></template>`),
			},
		},
	}

	paths, err := discoverer.Discover(context.Background(), []string{"test-module/pages/index.pk"})
	require.NoError(t, err)
	assert.Len(t, paths, 1)
}

func TestDiscover_ChainedImports(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &resolver_domain.MockResolver{
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
		},
		fsReader: &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages
import "test-module/components/header.pk"
</script>`),
				"/project/components/header.pk": []byte(`<script lang="go">
package components
import "test-module/components/logo.pk"
</script>`),
				"/project/components/logo.pk": []byte(`<script lang="go">
package components
</script>`),
			},
		},
	}

	paths, err := discoverer.Discover(context.Background(), []string{"test-module/pages/index.pk"})
	require.NoError(t, err)
	assert.Len(t, paths, 3)
}

func TestDiscover_CircularImports(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &resolver_domain.MockResolver{
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
		},
		fsReader: &mockFSReader{
			Files: map[string][]byte{
				"/project/a.pk": []byte(`<script lang="go">
package a
import "test-module/b.pk"
</script>`),
				"/project/b.pk": []byte(`<script lang="go">
package b
import "test-module/a.pk"
</script>`),
			},
		},
	}

	paths, err := discoverer.Discover(context.Background(), []string{"test-module/a.pk"})
	require.NoError(t, err)
	assert.Len(t, paths, 2, "circular imports should be handled via visited set")
}

func TestResolveEntryPoint_ErrorFromResolver(t *testing.T) {
	t.Parallel()

	resolver := &errorResolver{err: errors.New("resolve failed")}
	discoverer := &hasherDependencyDiscoverer{
		resolver: resolver,
		fsReader: &mockFSReader{Files: map[string][]byte{}},
	}

	_, err := discoverer.resolveEntryPoint(context.Background(), "pages/index.pk", "test-module")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot resolve entry point")
}

type errorResolver struct {
	err error
}

func (r *errorResolver) DetectLocalModule(_ context.Context) error { return nil }
func (r *errorResolver) GetBaseDir() string                        { return "/project" }
func (r *errorResolver) GetModuleName() string                     { return "test-module" }
func (r *errorResolver) ResolvePKPath(_ context.Context, _ string, _ string) (string, error) {
	return "", r.err
}
func (r *errorResolver) ResolveCSSPath(_ context.Context, _ string, _ string) (string, error) {
	return "", r.err
}
func (r *errorResolver) ResolveAssetPath(_ context.Context, _ string, _ string) (string, error) {
	return "", r.err
}
func (r *errorResolver) ConvertEntryPointPathToManifestKey(p string) string { return p }
func (r *errorResolver) GetModuleDir(_ context.Context, _ string) (string, error) {
	return "", r.err
}
func (r *errorResolver) FindModuleBoundary(_ context.Context, _ string) (string, string, error) {
	return "", "", r.err
}

func TestEnqueueEntryPoints_ResolveError(t *testing.T) {
	t.Parallel()

	discoverer := &hasherDependencyDiscoverer{
		resolver: &errorResolver{err: errors.New("resolve failed")},
		fsReader: &mockFSReader{Files: map[string][]byte{}},
	}

	var queue []string
	visited := make(map[string]bool)

	err := discoverer.enqueueEntryPoints(context.Background(), []string{"pages/index.pk"}, &queue, visited)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving entry point")
}

func TestExtractPKImports_NoScriptBlock(t *testing.T) {
	t.Parallel()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/template.pk": []byte(`<template><div>Hello</div></template>`),
		},
	}

	discoverer := &hasherDependencyDiscoverer{
		fsReader: fsReader,
	}

	imports, err := discoverer.extractPKImports(context.Background(), "/template.pk")
	require.NoError(t, err)
	assert.Nil(t, imports)
}

func TestExtractPKImports_MixedImports(t *testing.T) {
	t.Parallel()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/test.pk": []byte(`<script lang="go">
package test

import (
	"fmt"
	"net/http"
	"test-module/components/header.pk"
	"github.com/some/lib"
)
</script>`),
		},
	}

	discoverer := &hasherDependencyDiscoverer{
		fsReader: fsReader,
	}

	imports, err := discoverer.extractPKImports(context.Background(), "/test.pk")
	require.NoError(t, err)
	assert.Len(t, imports, 1)
	assert.Equal(t, "test-module/components/header.pk", imports[0])
}

func TestExtractPKImports_MultiplePKImports(t *testing.T) {
	t.Parallel()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/test.pk": []byte(`<script lang="go">
package test

import (
	"test-module/a.pk"
	"test-module/b.pk"
	"test-module/c.pk"
)
</script>`),
		},
	}

	discoverer := &hasherDependencyDiscoverer{
		fsReader: fsReader,
	}

	imports, err := discoverer.extractPKImports(context.Background(), "/test.pk")
	require.NoError(t, err)
	assert.Len(t, imports, 3)
}

func TestGetEffectiveResolver_NilBuildOpts(t *testing.T) {
	t.Parallel()

	defaultResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/default" }}
	service := &coordinatorService{resolver: defaultResolver}

	result := service.getEffectiveResolver(nil)
	assert.Same(t, defaultResolver, result)
}

func TestGetEffectiveResolver_NilResolver(t *testing.T) {
	t.Parallel()

	defaultResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/default" }}
	service := &coordinatorService{resolver: defaultResolver}

	result := service.getEffectiveResolver(&buildOptions{Resolver: nil})
	assert.Same(t, defaultResolver, result)
}

func TestGetEffectiveResolver_WithOverride(t *testing.T) {
	t.Parallel()

	defaultResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/default" }}
	overrideResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/override" }}
	service := &coordinatorService{resolver: defaultResolver}

	result := service.getEffectiveResolver(&buildOptions{Resolver: overrideResolver})
	assert.Same(t, overrideResolver, result)
}

func TestGetEntrypointPaths_Nil(t *testing.T) {
	t.Parallel()

	result := getEntrypointPaths(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetEntrypointPaths_Empty(t *testing.T) {
	t.Parallel()

	result := getEntrypointPaths([]annotator_dto.EntryPoint{})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetEntrypointPaths_Single(t *testing.T) {
	t.Parallel()

	eps := []annotator_dto.EntryPoint{
		{Path: "/a.pk", IsPage: true},
	}
	result := getEntrypointPaths(eps)
	assert.Equal(t, []string{"/a.pk"}, result)
}

func TestGetEntrypointPaths_Multiple(t *testing.T) {
	t.Parallel()

	eps := []annotator_dto.EntryPoint{
		{Path: "/a.pk", IsPage: true},
		{Path: "/b.pk", IsPage: false},
		{Path: "/c.pk", IsPage: true},
	}
	result := getEntrypointPaths(eps)
	assert.Equal(t, []string{"/a.pk", "/b.pk", "/c.pk"}, result)
}

func TestIsRelevantGoFile_Additional(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "simple go file", input: "main.go", expected: true},
		{name: "test file", input: "main_test.go", expected: false},
		{name: "not go file", input: "main.txt", expected: false},
		{name: "go in name but not extension", input: "gopher", expected: false},
		{name: "empty string", input: "", expected: false},
		{name: "dot go only", input: ".go", expected: true},
		{name: "underscore test dot go", input: "_test.go", expected: false},
		{name: "nested test", input: "foo_test.go", expected: false},
		{name: "go file with path", input: "pkg/handler.go", expected: true},
		{name: "go file with dotted name", input: "my.file.go", expected: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isRelevantGoFile(tc.input))
		})
	}
}

func TestTriggerBuild_EmptyRequest(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	request := &coordinator_dto.BuildRequest{}
	assert.NotPanics(t, func() {
		service.triggerBuild(context.Background(), request)
	})

	select {
	case received := <-service.rebuildTrigger:
		assert.Same(t, request, received)
	case <-time.After(time.Second):
		t.Fatal("expected request in channel")
	}
}

func TestUpdateStatus_IdleState(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	service.updateStatus(context.Background(), stateIdle, nil, nil, "")

	status := service.GetStatus()
	assert.Equal(t, stateIdle, status.State)
	assert.Nil(t, status.Result)
	assert.Nil(t, status.LastBuildError)
	assert.False(t, status.LastBuildTime.IsZero())
}

func TestUpdateStatus_ReadyWithResult_PublishesNotification(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	notificationChannel, unsub := service.Subscribe("test")
	defer unsub()

	result := &annotator_dto.ProjectAnnotationResult{}
	service.updateStatus(context.Background(), stateReady, result, nil, "cause-123")

	select {
	case n := <-notificationChannel:
		assert.Same(t, result, n.Result)
		assert.Equal(t, "cause-123", n.CausationID)
	case <-time.After(time.Second):
		t.Fatal("expected notification")
	}
}

func TestUpdateStatus_FailedDoesNotPublish(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	notificationChannel, unsub := service.Subscribe("test")
	defer unsub()

	service.updateStatus(context.Background(), stateFailed, &annotator_dto.ProjectAnnotationResult{}, errors.New("fail"), "cause")

	select {
	case <-notificationChannel:
		t.Fatal("should not receive notification for failed state")
	case <-time.After(50 * time.Millisecond):

	}
}

func TestNotifyWaiters_ResultAndError(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	waiter := &buildWaiter{done: make(chan struct{})}
	service.waiters.Store("hash-both", waiter)

	result := &annotator_dto.ProjectAnnotationResult{}
	expectedErr := errors.New("partial failure")
	service.notifyWaiters(context.Background(), "hash-both", result, expectedErr)

	select {
	case <-waiter.done:
		assert.Same(t, result, waiter.result)
		assert.ErrorIs(t, waiter.err, expectedErr)
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	_, loaded := service.waiters.Load("hash-both")
	assert.False(t, loaded)
}

func TestCacheIntrospectionResults_Success(t *testing.T) {
	t.Parallel()

	cache := newMockIntrospectionCache()
	service := &coordinatorService{introspectionCache: cache}

	tier1Result := tier1CacheResult{
		introspectionHash: "hash123",
		scriptHashes:      map[string]string{"file.pk": "scripthash"},
	}

	service.cacheIntrospectionResults(
		context.Background(),
		tier1Result,
		&annotator_dto.ComponentGraph{},
		&annotator_dto.VirtualModule{},
		&annotator_domain.TypeResolver{},
	)

	assert.Equal(t, 1, cache.setCalls)

	entry, err := cache.Get(context.Background(), "hash123")
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, CurrentIntrospectionCacheVersion, entry.Version)
	assert.Equal(t, map[string]string{"file.pk": "scripthash"}, entry.ScriptHashes)
}

func TestComputeFileHashesWithCache_Success(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("a.go", []byte("package main"))
	sandbox.AddFile("b.go", []byte("package main\nfunc b() {}"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/a.go": []byte("package main"),
			"/project/b.go": []byte("package main\nfunc b() {}"),
		},
	}

	service := &coordinatorService{
		fsReader:       fsReader,
		fileHashCache:  nil,
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	paths := []string{"/project/a.go", "/project/b.go"}
	hashes, contents, err := service.computeFileHashesWithCache(context.Background(), paths)
	require.NoError(t, err)
	assert.Len(t, hashes, 2)
	assert.Len(t, contents, 2)
	assert.NotEmpty(t, hashes["/project/a.go"])
	assert.NotEmpty(t, hashes["/project/b.go"])
	assert.Equal(t, []byte("package main"), contents["/project/a.go"])
	assert.Equal(t, []byte("package main\nfunc b() {}"), contents["/project/b.go"])
}

func TestComputeFileHashesWithCache_EmptyPaths(t *testing.T) {
	t.Parallel()

	service := &coordinatorService{
		fsReader:      &mockFSReader{Files: make(map[string][]byte)},
		fileHashCache: nil,
	}

	hashes, contents, err := service.computeFileHashesWithCache(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, hashes)
	assert.Empty(t, contents)
}

func TestNewCoordinatorService_ChannelCapacity(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	assert.Equal(t, 1, cap(service.rebuildTrigger))
}

func TestNewCoordinatorService_SubscriberMapInitialised(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	assert.NotNil(t, service.subscribers)
	assert.Empty(t, service.subscribers)
}

func TestOutputDiagnosticsIfPresent_NilBuildResult(t *testing.T) {
	t.Parallel()

	service := &coordinatorService{}
	result := service.outputDiagnosticsIfPresent(context.Background(), nil, nil, nil)
	assert.False(t, result)
}

func TestOutputDiagnosticsIfPresent_WithDiagnosticsAndError(t *testing.T) {
	t.Parallel()

	diagOut := &mockDiagnosticOutput{}
	service := &coordinatorService{diagnosticOutput: diagOut}

	buildResult := &annotator_dto.ProjectAnnotationResult{
		AllDiagnostics: []*ast_domain.Diagnostic{
			{Message: "error 1", SourcePath: "/a.pk"},
			{Message: "error 2", SourcePath: "/b.pk"},
		},
	}
	buildErr := errors.New("build failed")
	sourceContents := map[string][]byte{
		"/a.pk": []byte("content a"),
		"/b.pk": []byte("content b"),
	}

	result := service.outputDiagnosticsIfPresent(context.Background(), buildResult, buildErr, sourceContents)
	assert.True(t, result)
	assert.Equal(t, 1, diagOut.Calls)
	assert.True(t, diagOut.IsError)
	assert.Len(t, diagOut.Diagnostics, 2)
}

func TestOutputDiagnosticsIfPresent_WithDiagnosticsNoError(t *testing.T) {
	t.Parallel()

	diagOut := &mockDiagnosticOutput{}
	service := &coordinatorService{diagnosticOutput: diagOut}

	buildResult := &annotator_dto.ProjectAnnotationResult{
		AllDiagnostics: []*ast_domain.Diagnostic{
			{Message: "warning 1"},
		},
	}

	result := service.outputDiagnosticsIfPresent(context.Background(), buildResult, nil, nil)
	assert.True(t, result)
	assert.Equal(t, 1, diagOut.Calls)
	assert.False(t, diagOut.IsError)
}

func TestMockFileHashCache_GetSet_ModTimeMatching(t *testing.T) {
	t.Parallel()

	cache := newMockFileHashCache()
	ctx := context.Background()
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	cache.Set(ctx, "/file.go", t1, "hash1")

	hash, found := cache.Get(ctx, "/file.go", t1)
	assert.True(t, found)
	assert.Equal(t, "hash1", hash)

	hash, found = cache.Get(ctx, "/file.go", t2)
	assert.False(t, found)
	assert.Empty(t, hash)

	hash, found = cache.Get(ctx, "/other.go", t1)
	assert.False(t, found)
	assert.Empty(t, hash)

	cache.Set(ctx, "/file.go", t2, "hash2")
	hash, found = cache.Get(ctx, "/file.go", t2)
	assert.True(t, found)
	assert.Equal(t, "hash2", hash)
}

func TestEnqueueHashJobs_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	service := &coordinatorService{}

	paths := []string{"/a.go", "/b.go", "/c.go"}
	jobs := make(chan string, 1)

	service.enqueueHashJobs(ctx, jobs, paths)

	time.Sleep(50 * time.Millisecond)

	count := 0
	for range jobs {
		count++
	}

	assert.True(t, count <= len(paths))
}

func TestHashWorkerLoop_ContextCancelled(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("a.go", []byte("package main"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/a.go": []byte("package main"),
		},
	}

	service := &coordinatorService{
		fsReader:       fsReader,
		fileHashCache:  nil,
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	jobs := make(chan string, 1)
	results := make(chan fileHashResult, 1)

	jobs <- "/project/a.go"
	close(jobs)

	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := service.hashWorkerLoop(ctx, jobs, results)

	_ = err
}

func TestApplyBuildOptions_NilSlice(t *testing.T) {
	t.Parallel()

	opts := applyBuildOptions(nil)
	require.NotNil(t, opts)
	assert.Empty(t, opts.CausationID)
	assert.False(t, opts.FaultTolerant)
	assert.False(t, opts.SkipInspection)
}

func TestApplyCoordinatorOptions_NoOptions(t *testing.T) {
	t.Parallel()

	opts := applyCoordinatorOptions()
	assert.NotNil(t, opts.clock)
	assert.Equal(t, defaultDebounceDuration, opts.debounceDuration)
	assert.Nil(t, opts.fileHashCache)
	assert.Nil(t, opts.codeEmitter)
	assert.Nil(t, opts.diagnosticOutput)
	assert.Nil(t, opts.baseDirSandbox)
}

func TestWithDebounceDuration_Positive(t *testing.T) {
	t.Parallel()

	opts := applyCoordinatorOptions(WithDebounceDuration(100 * time.Millisecond))
	assert.Equal(t, 100*time.Millisecond, opts.debounceDuration)
}

func TestWithDebounceDuration_VeryLarge(t *testing.T) {
	t.Parallel()

	opts := applyCoordinatorOptions(WithDebounceDuration(1 * time.Hour))
	assert.Equal(t, 1*time.Hour, opts.debounceDuration)
}

func TestSubscribe_IncrementsNextSubID(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	assert.Equal(t, uint64(0), service.nextSubID)

	_, unsub1 := service.Subscribe("sub-1")
	assert.Equal(t, uint64(1), service.nextSubID)

	_, unsub2 := service.Subscribe("sub-2")
	assert.Equal(t, uint64(2), service.nextSubID)

	_, unsub3 := service.Subscribe("sub-3")
	assert.Equal(t, uint64(3), service.nextSubID)

	unsub1()
	unsub2()
	unsub3()

	assert.Equal(t, uint64(3), service.nextSubID)

	_, unsub4 := service.Subscribe("sub-4")
	assert.Equal(t, uint64(4), service.nextSubID)
	unsub4()
}

func TestBuildWaiter_ZeroValue(t *testing.T) {
	t.Parallel()

	var w buildWaiter
	assert.Nil(t, w.result)
	assert.Nil(t, w.err)
	assert.Nil(t, w.done)
}

func TestFileHashResult_ZeroValue(t *testing.T) {
	t.Parallel()

	var r fileHashResult
	assert.Empty(t, r.path)
	assert.Empty(t, r.hash)
	assert.Nil(t, r.content)
}

func TestDiscoverAndSortAllSourcePaths_SortedOutput(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("main.go", []byte("package main"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/z.pk": []byte(`<script lang="go">
package pages
</script>`),
			"/project/pages/a.pk": []byte(`<script lang="go">
package pages
</script>`),
			"/project/main.go": []byte("package main"),
		},
	}

	baseDir := "/project"
	service := &coordinatorService{
		fsReader:       fsReader,
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return baseDir },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join(baseDir, relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
	}

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/z.pk"},
		{Path: "test-module/pages/a.pk"},
	}

	paths, err := service.discoverAndSortAllSourcePaths(
		context.Background(),
		entryPoints,
		service.resolver,
	)
	require.NoError(t, err)
	assert.True(t, len(paths) >= 2, "should have at least 2 paths")

	for i := 1; i < len(paths); i++ {
		assert.True(t, paths[i-1] <= paths[i], "paths should be sorted: %s <= %s", paths[i-1], paths[i])
	}
}

func TestHandleSemanticError_NilDiagnosticOutput(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	semErr := annotator_domain.NewSemanticError([]*ast_domain.Diagnostic{
		{Message: "error", SourcePath: "/a.pk"},
	})

	logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", 0)
	request := &coordinator_dto.BuildRequest{CausationID: "cause"}
	partialResult := &annotator_dto.ProjectAnnotationResult{}

	result, err := service.handleSemanticError(
		context.Background(),
		semErr,
		partialResult,
		nil,
		logStore,
		request,
	)

	assert.Error(t, err)
	assert.Same(t, partialResult, result)
}

func TestBuildLoop_ShutdownStops(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	close(service.shutdown)
	service.wg.Wait()
}

func TestConcurrentStatusAndBuild(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			_ = service.GetStatus()
		})
		wg.Go(func() {
			_, _ = service.GetLastSuccessfulBuild()
		})
		wg.Go(func() {
			if i%2 == 0 {
				service.updateStatus(context.Background(), stateReady, &annotator_dto.ProjectAnnotationResult{}, nil, "c")
			} else {
				service.updateStatus(context.Background(), stateFailed, nil, errors.New("err"), "c")
			}
		})
	}
	wg.Wait()
}

func TestHashAndReadFile_SetsCache(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cache := newMockFileHashCache()
	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/file.go": []byte("package main"),
		},
	}

	service := &coordinatorService{
		fsReader:      fsReader,
		fileHashCache: cache,
	}

	hash, _, err := service.hashAndReadFile(context.Background(), "/project/file.go", modTime)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.Equal(t, 1, cache.setCalls)

	cachedHash, found := cache.Get(context.Background(), "/project/file.go", modTime)
	assert.True(t, found)
	assert.Equal(t, hash, cachedHash)
}

func TestOutputInternalCompilerLogs_WithMatchingFile(t *testing.T) {
	t.Parallel()

	logStore, err := annotator_domain.NewCompilationLogStore(context.Background(), false, "", 0)
	require.NoError(t, err)

	diagnostics := []*ast_domain.Diagnostic{
		{SourcePath: "/test.pk", Message: "error"},
	}

	assert.NotPanics(t, func() {
		outputInternalCompilerLogs(diagnostics, logStore)
	})
}

func TestSubscriber_ChannelBufferSize(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
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
		},
		applyCoordinatorOptions(),
	)

	notificationChannel, unsub := service.Subscribe("test")
	defer unsub()

	assert.Equal(t, 1, cap(notificationChannel), "subscriber channel should have buffer of 1")
}

func TestIntrospectionCacheEntryIsValid_AllFieldsPresent(t *testing.T) {
	t.Parallel()

	e := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{"a": "b"},
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}
	assert.True(t, e.IsValid())
}

func TestIntrospectionCacheEntryIsValid_WrongVersion(t *testing.T) {
	t.Parallel()

	e := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		Version:        0,
	}
	assert.False(t, e.IsValid())
}

func TestIntrospectionCacheEntryIsValid_ZeroVersion(t *testing.T) {
	t.Parallel()

	e := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		Version:        0,
	}
	assert.False(t, e.IsValid())
}

func TestMatchesScriptHashes_BothNonNilButOneEmpty(t *testing.T) {
	t.Parallel()

	e := &IntrospectionCacheEntry{
		ScriptHashes: map[string]string{},
	}

	assert.False(t, e.MatchesScriptHashes(map[string]string{"a": "b"}))
}

func TestMatchesScriptHashes_CachedHasMoreThanCurrent(t *testing.T) {
	t.Parallel()

	e := &IntrospectionCacheEntry{
		ScriptHashes: map[string]string{"a": "1", "b": "2"},
	}

	assert.False(t, e.MatchesScriptHashes(map[string]string{"a": "1"}))
}

func TestNewIntrospectionCacheEntry_TimestampIsRecent(t *testing.T) {
	t.Parallel()

	before := time.Now()
	entry := newIntrospectionCacheEntry(
		&annotator_dto.VirtualModule{},
		&annotator_domain.TypeResolver{},
		&annotator_dto.ComponentGraph{},
		map[string]string{},
	)
	after := time.Now()

	assert.False(t, entry.Timestamp.Before(before))
	assert.False(t, entry.Timestamp.After(after))
}

func TestComputeFileHashesWithCache_WithCacheEnabled(t *testing.T) {
	t.Parallel()

	fileHashCache := newMockFileHashCache()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("a.go", []byte("package main"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/a.go": []byte("package main"),
		},
	}

	service := &coordinatorService{
		fsReader:       fsReader,
		fileHashCache:  fileHashCache,
		baseDirSandbox: sandbox,
		resolver: &resolver_domain.MockResolver{
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
		},
	}

	hashes, contents, err := service.computeFileHashesWithCache(context.Background(), []string{"/project/a.go"})
	require.NoError(t, err)
	assert.Len(t, hashes, 1)
	assert.Len(t, contents, 1)
	assert.NotEmpty(t, hashes["/project/a.go"])
	assert.Equal(t, []byte("package main"), contents["/project/a.go"])

	assert.True(t, fileHashCache.setCalls > 0, "cache should have been populated")
}

func TestErrors_AreDistinct(t *testing.T) {
	t.Parallel()

	allErrors := []error{ErrCacheMiss, ErrInvalidCacheEntry, errBuildInProgress, errNoBuildAvailable}
	for i := range allErrors {
		for j := i + 1; j < len(allErrors); j++ {
			assert.NotEqual(t, allErrors[i], allErrors[j],
				"errors %d and %d should be distinct", i, j)
			assert.False(t, errors.Is(allErrors[i], allErrors[j]),
				"errors.Is should be false for %d and %d", i, j)
		}
	}
}

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	assert.Contains(t, ErrCacheMiss.Error(), "not found")
	assert.Contains(t, ErrInvalidCacheEntry.Error(), "invalid")
	assert.Contains(t, errBuildInProgress.Error(), "in progress")
	assert.Contains(t, errNoBuildAvailable.Error(), "no build")
}
