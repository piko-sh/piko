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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"

	"piko.sh/piko/internal/json"
)

func TestGoModuleCacheResolver_findModulePath(t *testing.T) {
	resolver := NewGoModuleCacheResolver()

	resolver.knownModules = []string{
		"github.com/org/project",
		"gitlab.com/company/team",
		"github.com/user/repo",
		"github.com/ui/lib",
		"go.uber.org/zap",
		"gopkg.in/yaml.v2",
	}

	testCases := []struct {
		name              string
		importPath        string
		expectedModule    string
		expectedPathInMod string
		expectErr         bool
	}{
		{
			name:              "Standard GitHub module with nested path",
			importPath:        "github.com/ui/lib/components/button.pk",
			expectedModule:    "github.com/ui/lib",
			expectedPathInMod: "components/button.pk",
			expectErr:         false,
		},
		{
			name:              "Standard GitHub module with deeply nested path",
			importPath:        "github.com/org/project/internal/components/card/card.pk",
			expectedModule:    "github.com/org/project",
			expectedPathInMod: "internal/components/card/card.pk",
			expectErr:         false,
		},
		{
			name:              "GitLab module",
			importPath:        "gitlab.com/company/team/partials/header.pk",
			expectedModule:    "gitlab.com/company/team",
			expectedPathInMod: "partials/header.pk",
			expectErr:         false,
		},
		{
			name:              "Module with just the file at root",
			importPath:        "github.com/user/repo/component.pk",
			expectedModule:    "github.com/user/repo",
			expectedPathInMod: "component.pk",
			expectErr:         false,
		},
		{
			name:              "Custom domain module (go.uber.org)",
			importPath:        "go.uber.org/zap/zapcore/field.pk",
			expectedModule:    "go.uber.org/zap",
			expectedPathInMod: "zapcore/field.pk",
			expectErr:         false,
		},
		{
			name:              "Vanity import (gopkg.in)",
			importPath:        "gopkg.in/yaml.v2/parser.pk",
			expectedModule:    "gopkg.in/yaml.v2",
			expectedPathInMod: "parser.pk",
			expectErr:         false,
		},
		{
			name:       "Module not in go.mod",
			importPath: "github.com/unknown/module/file.pk",
			expectErr:  true,
		},
		{
			name:       "Path too short for any known module",
			importPath: "button.pk",
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			modulePath, pathInModule, err := resolver.findModulePath(tc.importPath)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedModule, modulePath, "Module path should match")
				assert.Equal(t, tc.expectedPathInMod, pathInModule, "Path within module should match")
			}
		})
	}
}

func TestGoModuleCacheResolver_DetectLocalModule(t *testing.T) {
	resolver := NewGoModuleCacheResolver()
	ctx := context.Background()

	err := resolver.DetectLocalModule(ctx)
	if err != nil {
		t.Logf("DetectLocalModule failed because no go.mod found: %v", err)
	} else {
		assert.NotNil(t, resolver.knownModules, "Known modules should be populated")
		assert.Greater(t, len(resolver.knownModules), 0, "Should have loaded at least one module")
	}
}

func TestGoModuleCacheResolver_GetModuleName(t *testing.T) {
	resolver := NewGoModuleCacheResolver()

	moduleName := resolver.GetModuleName()
	assert.Equal(t, "", moduleName)
}

func TestGoModuleCacheResolver_GetBaseDir(t *testing.T) {
	resolver := NewGoModuleCacheResolver()

	baseDir := resolver.GetBaseDir()
	assert.Equal(t, "", baseDir)
}

func TestGoModuleCacheResolver_ConvertEntryPointPathToManifestKey(t *testing.T) {
	resolver := NewGoModuleCacheResolver()

	testCases := []struct {
		name        string
		entryPoint  string
		expectedKey string
	}{
		{
			name:        "External module path",
			entryPoint:  "github.com/ui/lib/components/button.pk",
			expectedKey: "github.com/ui/lib/components/button.pk",
		},
		{
			name:        "Simple path",
			entryPoint:  "pages/index.pk",
			expectedKey: "pages/index.pk",
		},
		{
			name:        "Empty path",
			entryPoint:  "",
			expectedKey: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolver.ConvertEntryPointPathToManifestKey(tc.entryPoint)
			assert.Equal(t, tc.expectedKey, result)
		})
	}
}

func TestGoModuleCacheResolver_ResolveCSSPath(t *testing.T) {
	resolver := NewGoModuleCacheResolver()
	ctx := context.Background()

	_, err := resolver.ResolveCSSPath(ctx, "github.com/ui/lib/styles/theme.css", "/some/directory")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestGoModuleCacheResolver_CacheThreadSafety(t *testing.T) {
	resolver := NewGoModuleCacheResolver()

	resolver.mu.Lock()
	resolver.dirCache["github.com/test/module"] = "/test/path"
	resolver.mu.Unlock()

	done := make(chan bool)
	for range 10 {
		go func() {
			resolver.mu.RLock()
			_ = resolver.dirCache["github.com/test/module"]
			resolver.mu.RUnlock()
			done <- true
		}()
	}

	for range 10 {
		<-done
	}

	resolver.mu.RLock()
	cachedPath := resolver.dirCache["github.com/test/module"]
	resolver.mu.RUnlock()
	assert.Equal(t, "/test/path", cachedPath)
}

func TestGoModuleCacheResolver_ResolvePKPath_ValidationErrors(t *testing.T) {
	resolver := NewGoModuleCacheResolver()
	ctx := context.Background()

	resolver.knownModules = []string{
		"github.com/known/module",
	}

	testCases := []struct {
		name          string
		importPath    string
		expectedError string
	}{
		{
			name:          "Module not in go.mod",
			importPath:    "github.com/unknown/module/component.pk",
			expectedError: "does not match any module in go.mod",
		},
		{
			name:          "Short path not matching any module",
			importPath:    "short/path.pk",
			expectedError: "does not match any module in go.mod",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolver.ResolvePKPath(ctx, tc.importPath, "")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func TestGoModuleCacheResolver_ResolvePKPath_ExtensionValidation(t *testing.T) {
	resolver := NewGoModuleCacheResolver()
	ctx := context.Background()

	resolver.knownModules = []string{
		"github.com/test/lib",
	}

	resolver.mu.Lock()
	resolver.dirCache["github.com/test/lib"] = "/fake/gomodcache/github.com/test/lib@v1.0.0"
	resolver.mu.Unlock()

	testCases := []struct {
		name       string
		importPath string
	}{
		{
			name:       "Go file instead of pk",
			importPath: "github.com/test/lib/component.go",
		},
		{
			name:       "Text file",
			importPath: "github.com/test/lib/readme.txt",
		},
		{
			name:       "No extension",
			importPath: "github.com/test/lib/component",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolver.ResolvePKPath(ctx, tc.importPath, "")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not a .pk file")
		})
	}
}

func BenchmarkGoModuleCacheResolver_findModulePath(b *testing.B) {
	resolver := NewGoModuleCacheResolver()
	importPath := "github.com/org/project/internal/components/deeply/nested/path/to/component.pk"

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = resolver.findModulePath(importPath)
	}
}

func BenchmarkGoModuleCacheResolver_CacheHit(b *testing.B) {
	resolver := NewGoModuleCacheResolver()

	resolver.mu.Lock()
	resolver.dirCache["github.com/test/module"] = "/test/path"
	resolver.mu.Unlock()

	b.ResetTimer()
	for b.Loop() {
		resolver.mu.RLock()
		_ = resolver.dirCache["github.com/test/module"]
		resolver.mu.RUnlock()
	}
}

func TestResolveWithinModule(t *testing.T) {
	t.Parallel()

	moduleDir := filepath.Join(string(filepath.Separator)+"cache", "mod@v1.0.0")
	tests := []struct {
		name             string
		filePathInModule string
		wantSuffix       string
		wantErr          bool
	}{
		{name: "nested asset", filePathInModule: "lib/icons/logo.svg", wantSuffix: filepath.Join("lib", "icons", "logo.svg")},
		{name: "module root file", filePathInModule: "go.mod", wantSuffix: "go.mod"},
		{name: "parent traversal", filePathInModule: "../../etc/passwd", wantErr: true},
		{name: "deep traversal", filePathInModule: "lib/../../../secret", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveWithinModule(moduleDir, tt.filePathInModule)
			if tt.wantErr {
				require.ErrorIs(t, err, errPathEscapesModule)
				return
			}
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(got, moduleDir))
			assert.True(t, strings.HasSuffix(got, tt.wantSuffix))
		})
	}
}

func TestCollectReplaceDirs(t *testing.T) {
	t.Parallel()

	goModDir := t.TempDir()
	goMod := []byte(`module example.com/app

go 1.26

require (
	example.com/ui v0.0.0
	example.com/proxied v1.2.3
)

replace example.com/ui => ./vendored/ui
replace example.com/proxied v1.2.3 => example.com/fork v1.4.0
`)
	modFile, err := modfile.Parse("go.mod", goMod, nil)
	require.NoError(t, err)

	got := collectReplaceDirs(modFile, goModDir)

	assert.Equal(t, filepath.Join(goModDir, "vendored", "ui"), got["example.com/ui"])

	assert.NotContains(t, got, "example.com/proxied")
}

func TestGoModuleCacheResolver_ResolveModuleDir_ReplaceFastPath(t *testing.T) {
	t.Parallel()

	moduleDir := t.TempDir()
	resolver := NewGoModuleCacheResolver()
	resolver.mu.Lock()
	resolver.replaceDirs["example.com/ui"] = moduleDir
	resolver.mu.Unlock()

	got, err := resolver.resolveModuleDir(context.Background(), "example.com/ui")
	require.NoError(t, err)
	assert.Equal(t, moduleDir, got)

	cached, ok := resolver.getCachedModuleDir(context.Background(), "example.com/ui")
	assert.True(t, ok)
	assert.Equal(t, moduleDir, cached)
}

func TestGoModuleCacheResolver_ResolveAssetPath_ViaReplaceDir(t *testing.T) {
	t.Parallel()

	moduleDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(moduleDir, "lib", "icons"), 0o750))
	assetPath := filepath.Join(moduleDir, "lib", "icons", "logo.svg")
	require.NoError(t, os.WriteFile(assetPath, []byte("<svg/>"), 0o600))

	resolver := NewGoModuleCacheResolver()
	resolver.mu.Lock()
	resolver.knownModules = []string{"example.com/ui"}
	resolver.replaceDirs["example.com/ui"] = moduleDir
	resolver.mu.Unlock()

	t.Run("resolves existing asset", func(t *testing.T) {
		t.Parallel()
		got, err := resolver.ResolveAssetPath(context.Background(), "example.com/ui/lib/icons/logo.svg", "")
		require.NoError(t, err)
		assert.Equal(t, assetPath, got)
	})

	t.Run("rejects traversal outside module", func(t *testing.T) {
		t.Parallel()
		_, err := resolver.ResolveAssetPath(context.Background(), "example.com/ui/../../../secret", "")
		require.ErrorIs(t, err, errPathEscapesModule)
	})

	t.Run("missing asset", func(t *testing.T) {
		t.Parallel()
		_, err := resolver.ResolveAssetPath(context.Background(), "example.com/ui/lib/icons/missing.svg", "")
		require.ErrorIs(t, err, errExternalAssetNotFound)
	})
}

func TestGoListModule_JSONParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "plain dir",
			payload: `{"Path":"example.com/ui","Dir":"/cache/ui@v1"}`,
			want:    "/cache/ui@v1",
		},
		{
			name:    "replace dir preferred",
			payload: `{"Path":"example.com/ui","Dir":"/cache/ui@v1","Replace":{"Dir":"/local/ui"}}`,
			want:    "/local/ui",
		},
		{
			name:    "not extracted",
			payload: `{"Path":"example.com/ui"}`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var module goListModule
			require.NoError(t, json.Unmarshal([]byte(tt.payload), &module))

			directory := module.Dir
			if module.Replace != nil && module.Replace.Dir != "" {
				directory = module.Replace.Dir
			}
			assert.Equal(t, tt.want, directory)
		})
	}
}
