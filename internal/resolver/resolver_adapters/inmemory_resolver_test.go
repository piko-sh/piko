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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestNewInMemoryModuleResolver(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymodule", "/project")
	require.NotNil(t, r)
	assert.Equal(t, "mymodule", r.GetModuleName())
	assert.Equal(t, "/project", r.GetBaseDir())
}

func TestInMemoryModuleResolver_DetectLocalModule(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mod", "/directory")
	err := r.DetectLocalModule(context.Background())
	require.NoError(t, err)
}

func TestInMemoryModuleResolver_ConvertEntryPointPathToManifestKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		moduleName string
		input      string
		expected   string
	}{
		{name: "strips module prefix", moduleName: "mymod", input: "mymod/pages/index.pk", expected: "pages/index.pk"},
		{name: "no match keeps original", moduleName: "mymod", input: "othermod/pages/index.pk", expected: "othermod/pages/index.pk"},
		{name: "empty input", moduleName: "mymod", input: "", expected: ""},
		{name: "only module name no slash", moduleName: "mymod", input: "mymod", expected: "mymod"},
		{name: "github style module", moduleName: "github.com/org/repo", input: "github.com/org/repo/pages/index.pk", expected: "pages/index.pk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewInMemoryModuleResolver(tt.moduleName, "/base")
			assert.Equal(t, tt.expected, r.ConvertEntryPointPathToManifestKey(tt.input))
		})
	}
}

func TestInMemoryModuleResolver_ResolvePKPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("resolves module-absolute pk path", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		path, err := r.ResolvePKPath(ctx, "mymod/components/button.pk", "")
		require.NoError(t, err)
		assert.Contains(t, path, "components")
		assert.Contains(t, path, "button.pk")
	})

	t.Run("rejects non-pk file", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		_, err := r.ResolvePKPath(ctx, "mymod/components/button.go", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a .pk file")
	})

	t.Run("rejects path not matching module", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		_, err := r.ResolvePKPath(ctx, "othermod/components/button.pk", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "import path")
	})

	t.Run("rejects when module name is empty", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("", "/project")
		_, err := r.ResolvePKPath(ctx, "something/button.pk", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no module name configured")
	})
}

func TestInMemoryModuleResolver_ResolveCSSPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("resolves module-absolute CSS path", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		path, err := r.ResolveCSSPath(ctx, "mymod/styles/theme.css", "/project")
		require.NoError(t, err)
		assert.Contains(t, path, "styles")
		assert.Contains(t, path, "theme.css")
	})

	t.Run("resolves relative path", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		path, err := r.ResolveCSSPath(ctx, "./styles.css", "/project/components")
		require.NoError(t, err)
		assert.Contains(t, path, "styles.css")
	})

	t.Run("resolves parent relative path", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		path, err := r.ResolveCSSPath(ctx, "../styles/global.css", "/project/components/button")
		require.NoError(t, err)
		assert.Contains(t, path, "global.css")
	})

	t.Run("rejects non-css file", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		_, err := r.ResolveCSSPath(ctx, "mymod/styles/config.json", "/project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a .css file")
	})

	t.Run("rejects invalid import path", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		_, err := r.ResolveCSSPath(ctx, "styles/theme.css", "/project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid CSS import path")
	})

	t.Run("rejects relative non-css file", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		_, err := r.ResolveCSSPath(ctx, "./config.json", "/project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a .css file")
	})
}

func TestInMemoryModuleResolver_ResolveAssetPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("resolves module-absolute asset path", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		path, err := r.ResolveAssetPath(ctx, "mymod/lib/icons/arrow.svg", "")
		require.NoError(t, err)
		assert.Contains(t, path, "arrow.svg")
	})

	t.Run("rejects path not matching module with helpful error", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		_, err := r.ResolveAssetPath(ctx, "othermod/lib/icon.svg", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid asset path")
	})

	t.Run("rejects when module info not configured", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("", "")
		_, err := r.ResolveAssetPath(ctx, "something/icon.svg", "")
		require.Error(t, err)
	})
}

func TestInMemoryModuleResolver_GetModuleDir(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")
	_, err := r.GetModuleDir(context.Background(), "some/module")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support external module resolution")
}

func TestInMemoryModuleResolver_FindModuleBoundary(t *testing.T) {
	t.Parallel()

	r := NewInMemoryModuleResolver("mymod", "/project")
	_, _, err := r.FindModuleBoundary(context.Background(), "some/import/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support module boundary detection")
}

func TestInMemoryModuleResolver_ResolveModulePathInternal(t *testing.T) {
	t.Parallel()

	t.Run("resolves path within module", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		path, err := r.resolveModulePathInternal("mymod/lib/file.txt")
		require.NoError(t, err)
		assert.Contains(t, path, "/project")
		assert.Contains(t, path, "file.txt")
	})

	t.Run("rejects path outside module", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "/project")
		_, err := r.resolveModulePathInternal("othermod/lib/file.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in configured module")
	})

	t.Run("rejects when module name is empty", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("", "")
		_, err := r.resolveModulePathInternal("any/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "module information is not configured")
	})

	t.Run("rejects when base directory is empty", func(t *testing.T) {
		t.Parallel()
		r := NewInMemoryModuleResolver("mymod", "")
		_, err := r.resolveModulePathInternal("mymod/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "module information is not configured")
	})
}

func TestHasModuleAliasPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "has @ alias", path: "@/components/button.pk", expected: true},
		{name: "no @ alias", path: "mymod/components/button.pk", expected: false},
		{name: "empty string", path: "", expected: false},
		{name: "just @", path: "@", expected: false},
		{name: "@/ alone", path: "@/", expected: true},
		{name: "@ without slash", path: "@components/button.pk", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, hasModuleAliasPrefix(tt.path))
		})
	}
}

func TestExpandModuleAlias_NoAlias(t *testing.T) {
	t.Parallel()

	result, err := ExpandModuleAlias("mymod/components/button.pk", "/project/file.pk")
	require.NoError(t, err)
	assert.Equal(t, "mymod/components/button.pk", result)
}

func TestExpandModuleAlias_EmptyContainingFile(t *testing.T) {
	t.Parallel()

	_, err := ExpandModuleAlias("@/components/button.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no containing file context")
}

func TestChainedResolver_ResolveAssetPath_EmptyChain(t *testing.T) {
	t.Parallel()

	cr := NewChainedResolver()
	_, err := cr.ResolveAssetPath(context.Background(), "some/asset.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no resolvers configured")
}

func TestChainedResolver_ResolveCSSPath_EmptyChain(t *testing.T) {
	t.Parallel()

	cr := NewChainedResolver()
	_, err := cr.ResolveCSSPath(context.Background(), "some/style.css", "/directory")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no resolvers configured")
}

func TestChainedResolver_GetModuleDir_EmptyChain(t *testing.T) {
	t.Parallel()

	cr := NewChainedResolver()
	_, err := cr.GetModuleDir(context.Background(), "some/module")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no resolvers configured")
}

func TestChainedResolver_FindModuleBoundary_EmptyChain(t *testing.T) {
	t.Parallel()

	cr := NewChainedResolver()
	_, _, err := cr.FindModuleBoundary(context.Background(), "some/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no resolvers configured")
}

func TestChainedResolver_DetectLocalModule_EmptyChain(t *testing.T) {
	t.Parallel()

	cr := NewChainedResolver()
	err := cr.DetectLocalModule(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no resolvers configured")
}

func TestChainedResolver_GetModuleName_EmptyChain(t *testing.T) {
	t.Parallel()

	cr := NewChainedResolver()
	assert.Empty(t, cr.GetModuleName())
}

func TestChainedResolver_GetBaseDir_EmptyChain(t *testing.T) {
	t.Parallel()

	cr := NewChainedResolver()
	assert.Empty(t, cr.GetBaseDir())
}

func TestChainedResolver_ConvertEntryPointPath_EmptyChain(t *testing.T) {
	t.Parallel()

	cr := NewChainedResolver()
	assert.Equal(t, "original/path.pk", cr.ConvertEntryPointPathToManifestKey("original/path.pk"))
}

func TestChainedResolver_ResolveAssetPath_FirstFailsSecondSucceeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	first := &resolver_domain.MockResolver{
		ResolveAssetPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
			return "", assert.AnError
		},
	}
	second := &resolver_domain.MockResolver{
		ResolveAssetPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
			return "/found/asset.svg", nil
		},
	}

	cr := NewChainedResolver(first, second)
	path, err := cr.ResolveAssetPath(ctx, "mymod/asset.svg", "")
	require.NoError(t, err)
	assert.Equal(t, "/found/asset.svg", path)
}

func TestChainedResolver_GetModuleDir_FirstFailsSecondSucceeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	first := &resolver_domain.MockResolver{
		GetModuleDirFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not implemented")
		},
	}
	second := &resolver_domain.MockResolver{
		GetModuleDirFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not implemented")
		},
	}

	cr := NewChainedResolver(first, second)
	_, err := cr.GetModuleDir(ctx, "some/mod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve")
}

func TestChainedResolver_FindModuleBoundary_AllFail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	first := &resolver_domain.MockResolver{
		FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
			return "", "", errors.New("not implemented")
		},
	}
	second := &resolver_domain.MockResolver{
		FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
			return "", "", errors.New("not implemented")
		},
	}

	cr := NewChainedResolver(first, second)
	_, _, err := cr.FindModuleBoundary(ctx, "some/import/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find module boundary")
}

func TestGoModuleCacheResolver_ResolveAssetPath(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	_, err := r.ResolveAssetPath(context.Background(), "mod/asset.svg", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestGoModuleCacheResolver_findModulePath_NotInitialised(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()

	_, _, err := r.findModulePath("some/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module list not initialised")
}

func TestGoModuleCacheResolver_findModulePath_ExactMatch(t *testing.T) {
	t.Parallel()

	r := NewGoModuleCacheResolver()
	r.knownModules = []string{"github.com/org/project"}

	mod, sub, err := r.findModulePath("github.com/org/project")
	require.NoError(t, err)
	assert.Equal(t, "github.com/org/project", mod)
	assert.Empty(t, sub)
}

func TestLocalModuleResolver_GetModuleDir(t *testing.T) {
	t.Parallel()

	r := NewLocalModuleResolver("/project")
	_, err := r.GetModuleDir(context.Background(), "some/mod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "local resolver does not support external module resolution")
}

func TestLocalModuleResolver_FindModuleBoundary(t *testing.T) {
	t.Parallel()

	r := NewLocalModuleResolver("/project")
	_, _, err := r.FindModuleBoundary(context.Background(), "some/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "local resolver does not support module boundary detection")
}

func TestLocalModuleResolver_IsRemote(t *testing.T) {
	t.Parallel()

	r := NewLocalModuleResolver("/project")

	assert.True(t, r.isRemote("http://example.com/component.pk"))
	assert.True(t, r.isRemote("https://example.com/component.pk"))
	assert.False(t, r.isRemote("mymod/components/button.pk"))
	assert.False(t, r.isRemote("./relative.pk"))
	assert.False(t, r.isRemote(""))
}

func TestLocalModuleResolver_FetchRemotePK(t *testing.T) {
	t.Parallel()

	r := NewLocalModuleResolver("/project")
	_, err := r.fetchRemotePK(context.Background(), "https://example.com/component.pk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestLocalModuleResolver_ResolveModulePathInternal_NoModuleInfo(t *testing.T) {
	t.Parallel()

	r := NewLocalModuleResolver("/project")

	_, err := r.resolveModulePathInternal(context.Background(), "any/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module information is missing")
}
