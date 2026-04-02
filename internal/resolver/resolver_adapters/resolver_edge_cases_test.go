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

package resolver_adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/tools/go/packages"

	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestNewGoModuleCacheResolverWithWorkingDir_ValidGoMod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	goModContent := "module testmod\n\ngo 1.25\n\nrequire (\n\tgithub.com/stretchr/testify v1.9.0\n)\n"
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	resolver, err := NewGoModuleCacheResolverWithWorkingDir(tempDir)
	require.NoError(t, err)
	require.NotNil(t, resolver)
	assert.Equal(t, tempDir, resolver.workingDir)
	assert.NotNil(t, resolver.knownModules)
	assert.Contains(t, resolver.knownModules, "github.com/stretchr/testify")
}

func TestNewGoModuleCacheResolverWithWorkingDir_NoGoMod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	_, err := NewGoModuleCacheResolverWithWorkingDir(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load modules from go.mod")
}

func TestNewGoModuleCacheResolverWithWorkingDir_InvalidGoMod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("this is not valid go.mod content!!!"), 0644)
	require.NoError(t, err)

	_, err = NewGoModuleCacheResolverWithWorkingDir(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load modules from go.mod")
}

func TestNewGoModuleCacheResolverWithWorkingDir_EmptyGoMod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module emptymod\n\ngo 1.25\n"), 0644)
	require.NoError(t, err)

	resolver, err := NewGoModuleCacheResolverWithWorkingDir(tempDir)
	require.NoError(t, err)
	require.NotNil(t, resolver)
	assert.Empty(t, resolver.knownModules, "no require statements means empty module list")
}

func TestGoModuleCacheResolver_loadKnownModulesFromGoMod_WithWorkingDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	goModContent := "module testmod\n\ngo 1.25\n\nrequire (\n\tgithub.com/alpha/pkg v1.0.0\n\tgithub.com/beta/longerpkg v1.3.0\n)\n"
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	r := &GoModuleCacheResolver{
		dirCache:   make(map[string]string),
		workingDir: tempDir,
	}

	err = r.loadKnownModulesFromGoMod(context.Background())
	require.NoError(t, err)
	require.Len(t, r.knownModules, 2)

	assert.Equal(t, "github.com/beta/longerpkg", r.knownModules[0])
	assert.Equal(t, "github.com/alpha/pkg", r.knownModules[1])
}

func TestGoModuleCacheResolver_loadKnownModulesFromGoMod_NonExistentDir(t *testing.T) {
	t.Parallel()

	r := &GoModuleCacheResolver{
		dirCache:   make(map[string]string),
		workingDir: "/nonexistent/directory/that/does/not/exist",
	}

	err := r.loadKnownModulesFromGoMod(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read go.mod")
}

func TestGoModuleCacheResolver_cacheModuleDir(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.cacheModuleDir(context.Background(), "github.com/test/mod", "/some/cache/path")

	r.mu.RLock()
	directory, ok := r.dirCache["github.com/test/mod"]
	r.mu.RUnlock()

	assert.True(t, ok)
	assert.Equal(t, "/some/cache/path", directory)
}

func TestGoModuleCacheResolver_cacheModuleDir_Overwrite(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.cacheModuleDir(context.Background(), "github.com/test/mod", "/old/path")
	r.cacheModuleDir(context.Background(), "github.com/test/mod", "/new/path")

	r.mu.RLock()
	directory := r.dirCache["github.com/test/mod"]
	r.mu.RUnlock()

	assert.Equal(t, "/new/path", directory)
}

func TestGoModuleCacheResolver_cacheModuleDir_MultipleConcurrent(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()

	done := make(chan bool, 20)
	for i := range 20 {
		go func(index int) {
			r.cacheModuleDir(context.Background(), "github.com/test/mod", "/path")
			done <- true
		}(i)
	}

	for range 20 {
		<-done
	}

	r.mu.RLock()
	directory := r.dirCache["github.com/test/mod"]
	r.mu.RUnlock()

	assert.Equal(t, "/path", directory)
}

func TestGoModuleCacheResolver_getCachedModuleDir_Hit(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.mu.Lock()
	r.dirCache["github.com/cached/mod"] = "/cached/path"
	r.mu.Unlock()

	directory, ok := r.getCachedModuleDir(context.Background(), "github.com/cached/mod")
	assert.True(t, ok)
	assert.Equal(t, "/cached/path", directory)
}

func TestGoModuleCacheResolver_getCachedModuleDir_Miss(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	directory, ok := r.getCachedModuleDir(context.Background(), "github.com/uncached/mod")
	assert.False(t, ok)
	assert.Empty(t, directory)
}

func TestGoModuleCacheResolver_resolveModuleDir_CacheHit(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.mu.Lock()
	r.dirCache["github.com/known/mod"] = "/cache/github.com/known/mod@v1.0.0"
	r.mu.Unlock()

	directory, err := r.resolveModuleDir(context.Background(), "github.com/known/mod")
	require.NoError(t, err)
	assert.Equal(t, "/cache/github.com/known/mod@v1.0.0", directory)
}

func TestGoModuleCacheResolver_FindModuleBoundary_Success(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.knownModules = []string{"github.com/org/lib", "github.com/org"}

	mod, sub, err := r.FindModuleBoundary(context.Background(), "github.com/org/lib/components/button.pk")
	require.NoError(t, err)
	assert.Equal(t, "github.com/org/lib", mod)
	assert.Equal(t, "components/button.pk", sub)
}

func TestGoModuleCacheResolver_FindModuleBoundary_NotFound(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.knownModules = []string{"github.com/org/lib"}

	_, _, err := r.FindModuleBoundary(context.Background(), "github.com/unknown/module/file.pk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match any module")
}

func TestGoModuleCacheResolver_FindModuleBoundary_NotInitialised(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	_, _, err := r.FindModuleBoundary(context.Background(), "any/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module list not initialised")
}

func TestGoModuleCacheResolver_GetModuleDir_CacheHit(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.mu.Lock()
	r.dirCache["github.com/test/lib"] = "/gomodcache/github.com/test/lib@v1.0.0"
	r.mu.Unlock()

	directory, err := r.GetModuleDir(context.Background(), "github.com/test/lib")
	require.NoError(t, err)
	assert.Equal(t, "/gomodcache/github.com/test/lib@v1.0.0", directory)
}

func TestGoModuleCacheResolver_constructAndValidatePKPath_Valid(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	ctx := context.Background()

	path, err := r.constructAndValidatePKPath(ctx, "/cache/mod@v1.0.0", "components/button.pk", "mod/components/button.pk")
	require.NoError(t, err)
	assert.Contains(t, path, "button.pk")
	assert.Contains(t, path, "components")
}

func TestGoModuleCacheResolver_constructAndValidatePKPath_NotPK(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	ctx := context.Background()

	_, err := r.constructAndValidatePKPath(ctx, "/cache/mod@v1.0.0", "components/button.go", "mod/components/button.go")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a .pk file")
}

func TestGoModuleCacheResolver_constructAndValidatePKPath_CaseInsensitive(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	ctx := context.Background()

	path, err := r.constructAndValidatePKPath(ctx, "/cache/mod@v1.0.0", "components/BUTTON.PK", "mod/components/BUTTON.PK")
	require.NoError(t, err)
	assert.Contains(t, path, "BUTTON.PK")
}

func TestGoModuleCacheResolver_extractModuleDirFromPackages_EmptySlice(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	_, err := r.extractModuleDirFromPackages([]*packages.Package{}, "github.com/test/mod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in module cache")
}

func TestGoModuleCacheResolver_extractModuleDirFromPackages_NilModule(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	pkgs := []*packages.Package{{}}
	_, err := r.extractModuleDirFromPackages(pkgs, "github.com/test/mod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in module cache")
}

func TestGoModuleCacheResolver_extractModuleDirFromPackages_WithModule(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	pkgs := []*packages.Package{
		{
			Module: &packages.Module{
				Dir: "/gomodcache/github.com/test/mod@v1.0.0",
			},
		},
	}
	directory, err := r.extractModuleDirFromPackages(pkgs, "github.com/test/mod")
	require.NoError(t, err)
	assert.Equal(t, "/gomodcache/github.com/test/mod@v1.0.0", directory)
}

func TestGoModuleCacheResolver_findModulePath_GreedyLongestMatch(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()

	r.knownModules = []string{
		"github.com/org/repo/submodule",
		"github.com/org/repo",
	}

	mod, sub, err := r.findModulePath("github.com/org/repo/submodule/file.pk")
	require.NoError(t, err)
	assert.Equal(t, "github.com/org/repo/submodule", mod)
	assert.Equal(t, "file.pk", sub)
}

func TestGoModuleCacheResolver_findModulePath_EmptyModuleList(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.knownModules = []string{}

	_, _, err := r.findModulePath("github.com/any/module/file.pk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match any module in go.mod")
}

func TestGoModuleCacheResolver_ResolvePKPath_AliasWithEmptyContainingFile(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.knownModules = []string{"github.com/test/mod"}

	_, err := r.ResolvePKPath(context.Background(), "@/components/button.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding module alias")
}

func TestGoModuleCacheResolver_recordResolutionDuration(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()

	r.recordResolutionDuration(context.Background(), time.Now())
}

func TestLocalModuleResolver_ResolvePKPath_ValidModuleAbsolute(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "components"), 0755))

	goModContent := "module mymod\n\ngo 1.25\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte(goModContent), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "mymod", r.GetModuleName())

	path, err := r.ResolvePKPath(context.Background(), "mymod/components/button.pk", "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(projectRoot, "components", "button.pk"), path)
}

func TestLocalModuleResolver_ResolvePKPath_WrongModule(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolvePKPath(context.Background(), "othermod/components/button.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid component import path")
}

func TestLocalModuleResolver_ResolvePKPath_NoModuleDetected(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	r := NewLocalModuleResolver(tempDir)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolvePKPath(context.Background(), "somemod/components/button.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no local module detected")
}

func TestLocalModuleResolver_ResolvePKPath_NonPKFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolvePKPath(context.Background(), "mymod/components/button.go", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolved path is not a .pk file")
}

func TestLocalModuleResolver_ResolvePKPath_RemotePath(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolvePKPath(context.Background(), "https://example.com/component.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestLocalModuleResolver_ResolvePKPath_WithAliasExpansion(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	componentsDir := filepath.Join(projectRoot, "components")
	require.NoError(t, os.MkdirAll(componentsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	containingFile := filepath.Join(componentsDir, "parent.pk")
	path, err := r.ResolvePKPath(context.Background(), "@/components/button.pk", containingFile)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(projectRoot, "components", "button.pk"), path)
}

func TestLocalModuleResolver_ResolvePKPath_AliasWithEmptyContainingFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolvePKPath(context.Background(), "@/components/button.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding module alias")
}

func TestLocalModuleResolver_ResolveAssetPath_Valid(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "lib", "icons"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	path, err := r.ResolveAssetPath(context.Background(), "mymod/lib/icons/arrow.svg", "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(projectRoot, "lib", "icons", "arrow.svg"), path)
}

func TestLocalModuleResolver_ResolveAssetPath_WrongModule(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolveAssetPath(context.Background(), "othermod/lib/icon.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid asset path")
}

func TestLocalModuleResolver_ResolveAssetPath_WithAlias(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "lib"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "components"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	containingFile := filepath.Join(projectRoot, "components", "page.pk")
	path, err := r.ResolveAssetPath(context.Background(), "@/lib/icon.svg", containingFile)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(projectRoot, "lib", "icon.svg"), path)
}

func TestLocalModuleResolver_ResolveAssetPath_AliasEmptyContaining(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolveAssetPath(context.Background(), "@/lib/icon.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding module alias")
}

func TestLocalModuleResolver_ResolveAssetPath_NoModuleInfo(t *testing.T) {
	t.Parallel()

	r := NewLocalModuleResolver("/project")
	_, err := r.ResolveAssetPath(context.Background(), "somemod/icon.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module information is missing")
}

func TestLocalModuleResolver_DetectLocalModule_FromFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module testmod\n\ngo 1.25\n"), 0644))

	filePath := filepath.Join(tempDir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main"), 0644))

	r := NewLocalModuleResolver(filePath)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "testmod", r.GetModuleName())
	assert.Equal(t, tempDir, r.GetBaseDir())
}

func TestLocalModuleResolver_DetectLocalModule_NestedDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module deepmod\n\ngo 1.25\n"), 0644))
	nestedDir := filepath.Join(tempDir, "cmd", "server")
	require.NoError(t, os.MkdirAll(nestedDir, 0755))

	r := NewLocalModuleResolver(nestedDir)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "deepmod", r.GetModuleName())
	assert.Equal(t, tempDir, r.GetBaseDir())
}

func TestLocalModuleResolver_DetectLocalModule_NoGoMod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	r := NewLocalModuleResolver(tempDir)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)
	assert.Empty(t, r.GetModuleName())

	absDir, _ := filepath.Abs(tempDir)
	assert.Equal(t, absDir, r.GetBaseDir())
}

func TestLocalModuleResolver_ResolveCSSPath_WithAlias(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "styles"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	path, err := r.ResolveCSSPath(context.Background(), "@/styles/theme.css", projectRoot)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(projectRoot, "styles", "theme.css"), path)
}

func TestLocalModuleResolver_ResolveCSSPath_AliasNonCSS(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "styles"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolveCSSPath(context.Background(), "@/styles/config.json", projectRoot)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a .css file")
}

func TestLocalModuleResolver_resolveModulePathInternal_NotInModule(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.resolveModulePathInternal(context.Background(), "othermod/file.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in local module")
}

func TestExpandModuleAlias_WithGoMod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module github.com/test/project\n\ngo 1.25\n"), 0644))

	pagesDir := filepath.Join(tempDir, "pages")
	require.NoError(t, os.MkdirAll(pagesDir, 0755))

	containingFile := filepath.Join(pagesDir, "index.pk")
	result, err := ExpandModuleAlias("@/components/button.pk", containingFile)
	require.NoError(t, err)
	assert.Equal(t, "github.com/test/project/components/button.pk", result)
}

func TestExpandModuleAlias_NoGoMod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	pagesDir := filepath.Join(tempDir, "pages")
	require.NoError(t, os.MkdirAll(pagesDir, 0755))

	containingFile := filepath.Join(pagesDir, "index.pk")
	_, err := ExpandModuleAlias("@/components/button.pk", containingFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot expand '@' alias")
}

func TestExpandModuleAlias_NonAlias(t *testing.T) {
	t.Parallel()

	result, err := ExpandModuleAlias("github.com/test/lib/button.pk", "/some/file.pk")
	require.NoError(t, err)
	assert.Equal(t, "github.com/test/lib/button.pk", result)
}

func TestFindModuleNameForPath_Valid(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module github.com/test/findmod\n\ngo 1.25\n"), 0644))

	srcDir := filepath.Join(tempDir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0755))

	filePath := filepath.Join(srcDir, "file.go")
	name, err := findModuleNameForPath(filePath)
	require.NoError(t, err)
	assert.Equal(t, "github.com/test/findmod", name)
}

func TestFindModuleNameForPath_NoGoMod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "file.go")
	_, err := findModuleNameForPath(filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no go.mod found")
}

func TestFindModuleNameForPath_Caching(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module cached/mod\n\ngo 1.25\n"), 0644))

	srcDir := filepath.Join(tempDir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0755))

	filePath := filepath.Join(srcDir, "file.go")

	name1, err := findModuleNameForPath(filePath)
	require.NoError(t, err)
	assert.Equal(t, "cached/mod", name1)

	name2, err := findModuleNameForPath(filePath)
	require.NoError(t, err)
	assert.Equal(t, "cached/mod", name2)
}

func TestChainedResolver_FindModuleBoundary_FirstSucceeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	goModResolver := NewGoModuleCacheResolver()
	goModResolver.knownModules = []string{"github.com/org/lib"}

	cr := NewChainedResolver(goModResolver)
	mod, sub, err := cr.FindModuleBoundary(ctx, "github.com/org/lib/components/button.pk")
	require.NoError(t, err)
	assert.Equal(t, "github.com/org/lib", mod)
	assert.Equal(t, "components/button.pk", sub)
}

func TestChainedResolver_FindModuleBoundary_FirstFailsSecondSucceeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	inMemory := NewInMemoryModuleResolver("local", "/project")

	goModResolver := NewGoModuleCacheResolver()
	goModResolver.knownModules = []string{"github.com/org/lib"}

	cr := NewChainedResolver(inMemory, goModResolver)
	mod, sub, err := cr.FindModuleBoundary(ctx, "github.com/org/lib/button.pk")
	require.NoError(t, err)
	assert.Equal(t, "github.com/org/lib", mod)
	assert.Equal(t, "button.pk", sub)
}

func TestChainedResolver_ResolveCSSPath_AllFail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	first := &resolver_domain.MockResolver{
		ResolveCSSPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
			return "", assert.AnError
		},
	}
	second := &resolver_domain.MockResolver{
		ResolveCSSPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
			return "", assert.AnError
		},
	}

	cr := NewChainedResolver(first, second)
	_, err := cr.ResolveCSSPath(ctx, "some/path.css", "/directory")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve CSS")
}

func TestChainedResolver_ResolveCSSPath_FirstSucceeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	first := &resolver_domain.MockResolver{
		ResolveCSSPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
			return "/resolved/style.css", nil
		},
	}

	cr := NewChainedResolver(first)
	path, err := cr.ResolveCSSPath(ctx, "mod/style.css", "/directory")
	require.NoError(t, err)
	assert.Equal(t, "/resolved/style.css", path)
}

func TestChainedResolver_ResolveAssetPath_AllFail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	first := &resolver_domain.MockResolver{
		ResolveAssetPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
			return "", assert.AnError
		},
	}
	second := &resolver_domain.MockResolver{
		ResolveAssetPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
			return "", assert.AnError
		},
	}

	cr := NewChainedResolver(first, second)
	_, err := cr.ResolveAssetPath(ctx, "mod/icon.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve asset")
	assert.Contains(t, err.Error(), "2 resolvers")
}

func TestChainedResolver_GetModuleDir_FirstSucceeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	r := NewGoModuleCacheResolver()
	r.mu.Lock()
	r.dirCache["github.com/test/lib"] = "/gomodcache/github.com/test/lib@v1.0.0"
	r.mu.Unlock()

	cr := NewChainedResolver(r)
	directory, err := cr.GetModuleDir(ctx, "github.com/test/lib")
	require.NoError(t, err)
	assert.Equal(t, "/gomodcache/github.com/test/lib@v1.0.0", directory)
}

func TestInMemoryModuleResolver_ResolvePKPath_NonPKFile(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")
	_, err := r.ResolvePKPath(context.Background(), "mymod/components/button.go", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolved path is not a .pk file")
}

func TestInMemoryModuleResolver_ResolvePKPath_WrongModuleWithModuleName(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")
	_, err := r.ResolvePKPath(context.Background(), "othermod/button.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid component import path")
}

func TestInMemoryModuleResolver_ResolveCSSPath_ModuleAbsoluteNonCSS(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")
	_, err := r.ResolveCSSPath(context.Background(), "mymod/styles/config.json", "/directory")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a .css file")
}

func TestInMemoryModuleResolver_ResolveAssetPath_WrongModuleWithName(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")
	_, err := r.ResolveAssetPath(context.Background(), "othermod/icon.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid asset path")
}

func TestInMemoryModuleResolver_ResolveAssetPath_ModulePathInternalError(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("", "")
	_, err := r.ResolveAssetPath(context.Background(), "something/icon.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module information is not configured")
}

func TestFindGoMod_FromFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	goModPath := filepath.Join(tempDir, "go.mod")
	require.NoError(t, os.WriteFile(goModPath, []byte("module testmod\n"), 0644))

	filePath := filepath.Join(tempDir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main"), 0644))

	found, err := findGoMod(filePath)
	require.NoError(t, err)
	assert.Equal(t, goModPath, found)
}

func TestReadModuleName_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := readModuleName("/nonexistent/go.mod", nil)
	require.Error(t, err)
}

func TestInMemoryModuleResolver_ResolveCSSPath_WithModuleAlias(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewInMemoryModuleResolver("mymod", tempDir)

	path, err := r.ResolveCSSPath(context.Background(), "@/styles/theme.css", tempDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tempDir, "styles", "theme.css"), path)
}

func TestInMemoryModuleResolver_ResolveCSSPath_AliasNonCSS(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewInMemoryModuleResolver("mymod", tempDir)

	_, err := r.ResolveCSSPath(context.Background(), "@/styles/config.json", tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a .css file")
}

func TestInMemoryModuleResolver_ResolveCSSPath_AliasExpandError(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")

	_, err := r.ResolveCSSPath(context.Background(), "@/styles/theme.css", "/nonexistent/directory/for/test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding module alias")
}

func TestInMemoryModuleResolver_ResolveCSSPath_AliasModulePathError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module differentmod\n\ngo 1.25\n"), 0644))

	r := NewInMemoryModuleResolver("mymod", "/project")

	_, err := r.ResolveCSSPath(context.Background(), "@/styles/theme.css", tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving CSS module path")
}

func TestLocalModuleResolver_DetectLocalModule_BadModuleName(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("go 1.25\n"), 0644))

	r := NewLocalModuleResolver(tempDir)
	err := r.DetectLocalModule(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse module name")
}

func TestLocalModuleResolver_DetectLocalModule_NonExistentDir(t *testing.T) {
	t.Parallel()

	r := NewLocalModuleResolver("/this/path/does/not/exist/at/all")
	err := r.DetectLocalModule(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error finding go.mod")
}

func TestLocalModuleResolver_ResolveCSSPath_AliasExpandError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolveCSSPath(context.Background(), "@/styles/theme.css", "/nonexistent/directory/somewhere")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding module alias")
}

func TestLocalModuleResolver_ResolveCSSPath_ModuleAbsoluteNotInModule(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectRoot := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewLocalModuleResolver(projectRoot)
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)

	_, err = r.ResolveCSSPath(context.Background(), "mymod", projectRoot)
	require.Error(t, err)
}

func TestGoModuleCacheResolver_ResolvePKPath_ValidWithCache(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.knownModules = []string{"github.com/test/lib"}
	r.mu.Lock()
	r.dirCache["github.com/test/lib"] = "/gomodcache/github.com/test/lib@v1.0.0"
	r.mu.Unlock()

	path, err := r.ResolvePKPath(context.Background(), "github.com/test/lib/components/button.pk", "")
	require.NoError(t, err)
	assert.Contains(t, path, "button.pk")
	assert.Contains(t, path, "components")
}

func TestGoModuleCacheResolver_ResolvePKPath_NonPKWithCache(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.knownModules = []string{"github.com/test/lib"}
	r.mu.Lock()
	r.dirCache["github.com/test/lib"] = "/gomodcache/github.com/test/lib@v1.0.0"
	r.mu.Unlock()

	_, err := r.ResolvePKPath(context.Background(), "github.com/test/lib/components/button.go", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a .pk file")
}

func TestGoModuleCacheResolver_loadKnownModulesFromGoMod_UsesCwd(t *testing.T) {

	projectRoot, err := findGoMod(".")
	if err != nil || projectRoot == "" {
		t.Skip("Could not find go.mod from current working directory")
	}

	r := &GoModuleCacheResolver{
		dirCache:   make(map[string]string),
		workingDir: filepath.Dir(projectRoot),
	}

	err = r.loadKnownModulesFromGoMod(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, r.knownModules)
	assert.Greater(t, len(r.knownModules), 0)
}

func TestGoModuleCacheResolver_extractModuleDirFromPackages_WithErrors(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()

	pkgs := []*packages.Package{
		{
			Errors: []packages.Error{
				{Msg: "some error"},
			},
		},
	}
	_, err := r.extractModuleDirFromPackages(pkgs, "github.com/test/mod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "errors while loading package")
}

func TestInMemoryModuleResolver_ResolvePKPath_AliasExpandError(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")
	_, err := r.ResolvePKPath(context.Background(), "@/components/button.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding module alias")
}

func TestInMemoryModuleResolver_ResolveAssetPath_AliasExpandError(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")
	_, err := r.ResolveAssetPath(context.Background(), "@/lib/icon.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding module alias")
}

func TestInMemoryModuleResolver_ResolveAssetPath_WithValidAlias(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "components"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module mymod\n\ngo 1.25\n"), 0644))

	r := NewInMemoryModuleResolver("mymod", tempDir)
	containingFile := filepath.Join(tempDir, "components", "page.pk")
	path, err := r.ResolveAssetPath(context.Background(), "@/lib/icon.svg", containingFile)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tempDir, "lib", "icon.svg"), path)
}

func TestGoModuleCacheResolver_resolveModuleDir_CacheMissFails(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()

	_, err := r.resolveModuleDir(context.Background(), "github.com/nonexistent/module")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading module")
}

func TestInMemoryModuleResolver_ResolvePKPath_ValidDeep(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("github.com/org/project", "/project/root")
	path, err := r.ResolvePKPath(context.Background(), "github.com/org/project/components/card/card.pk", "")
	require.NoError(t, err)
	expected := filepath.Join("/project/root", "components", "card", "card.pk")
	assert.Equal(t, expected, path)
}

func TestLocalModuleResolver_ResolvePKPath_ModulePathInternalFailure(t *testing.T) {
	t.Parallel()

	r := &LocalModuleResolver{
		moduleName: "mymod",
		baseDir:    "",
	}

	_, err := r.ResolvePKPath(context.Background(), "mymod/components/button.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module information is missing")
}

func TestLocalModuleResolver_ResolveAssetPath_ModulePathInternalError(t *testing.T) {
	t.Parallel()

	r := &LocalModuleResolver{
		moduleName: "mymod",
		baseDir:    "",
	}

	_, err := r.ResolveAssetPath(context.Background(), "mymod/lib/icon.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module information is missing")
}
