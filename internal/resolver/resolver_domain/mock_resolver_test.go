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

package resolver_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockResolver_DetectLocalModule(t *testing.T) {
	t.Parallel()

	t.Run("nil DetectLocalModuleFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		err := m.DetectLocalModule(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DetectLocalModuleCallCount))
	})

	t.Run("delegates to DetectLocalModuleFunc", func(t *testing.T) {
		t.Parallel()

		var calledWith context.Context
		m := &MockResolver{
			DetectLocalModuleFunc: func(ctx context.Context) error {
				calledWith = ctx
				return nil
			},
		}

		ctx := context.Background()
		err := m.DetectLocalModule(ctx)

		require.NoError(t, err)
		assert.Equal(t, ctx, calledWith)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DetectLocalModuleCallCount))
	})

	t.Run("propagates error from DetectLocalModuleFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("detect failed")
		m := &MockResolver{
			DetectLocalModuleFunc: func(_ context.Context) error {
				return expectedErr
			},
		}

		err := m.DetectLocalModule(context.Background())

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DetectLocalModuleCallCount))
	})
}

func TestMockResolver_GetModuleName(t *testing.T) {
	t.Parallel()

	t.Run("nil GetModuleNameFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		result := m.GetModuleName()

		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetModuleNameCallCount))
	})

	t.Run("delegates to GetModuleNameFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		m := &MockResolver{
			GetModuleNameFunc: func() string {
				called = true
				return "piko.sh/piko"
			},
		}

		result := m.GetModuleName()

		assert.True(t, called)
		assert.Equal(t, "piko.sh/piko", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetModuleNameCallCount))
	})
}

func TestMockResolver_GetBaseDir(t *testing.T) {
	t.Parallel()

	t.Run("nil GetBaseDirFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		result := m.GetBaseDir()

		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetBaseDirCallCount))
	})

	t.Run("delegates to GetBaseDirFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		m := &MockResolver{
			GetBaseDirFunc: func() string {
				called = true
				return "/home/user/project"
			},
		}

		result := m.GetBaseDir()

		assert.True(t, called)
		assert.Equal(t, "/home/user/project", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetBaseDirCallCount))
	})
}

func TestMockResolver_ResolvePKPath(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolvePKPathFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		result, err := m.ResolvePKPath(context.Background(), "@/partials/card.pk", "/src/pages/index.pk")

		assert.NoError(t, err)
		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolvePKPathCallCount))
	})

	t.Run("delegates to ResolvePKPathFunc", func(t *testing.T) {
		t.Parallel()

		var gotCtx context.Context
		var gotImport, gotContaining string

		m := &MockResolver{
			ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
				gotCtx = ctx
				gotImport = importPath
				gotContaining = containingFilePath
				return "/resolved/partials/card.pk", nil
			},
		}

		ctx := context.Background()
		result, err := m.ResolvePKPath(ctx, "@/partials/card.pk", "/src/pages/index.pk")

		require.NoError(t, err)
		assert.Equal(t, ctx, gotCtx)
		assert.Equal(t, "@/partials/card.pk", gotImport)
		assert.Equal(t, "/src/pages/index.pk", gotContaining)
		assert.Equal(t, "/resolved/partials/card.pk", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolvePKPathCallCount))
	})

	t.Run("propagates error from ResolvePKPathFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("resolution failed")
		m := &MockResolver{
			ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
				return "", expectedErr
			},
		}

		result, err := m.ResolvePKPath(context.Background(), "bad/path", "/src/index.pk")

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolvePKPathCallCount))
	})
}

func TestMockResolver_ResolveCSSPath(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolveCSSPathFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		result, err := m.ResolveCSSPath(context.Background(), "@/styles/theme.css", "/src/pages")

		assert.NoError(t, err)
		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveCSSPathCallCount))
	})

	t.Run("delegates to ResolveCSSPathFunc", func(t *testing.T) {
		t.Parallel()

		var gotCtx context.Context
		var gotImport, gotDir string

		m := &MockResolver{
			ResolveCSSPathFunc: func(ctx context.Context, importPath string, containingDir string) (string, error) {
				gotCtx = ctx
				gotImport = importPath
				gotDir = containingDir
				return "/resolved/styles/theme.css", nil
			},
		}

		ctx := context.Background()
		result, err := m.ResolveCSSPath(ctx, "@/styles/theme.css", "/src/pages")

		require.NoError(t, err)
		assert.Equal(t, ctx, gotCtx)
		assert.Equal(t, "@/styles/theme.css", gotImport)
		assert.Equal(t, "/src/pages", gotDir)
		assert.Equal(t, "/resolved/styles/theme.css", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveCSSPathCallCount))
	})

	t.Run("propagates error from ResolveCSSPathFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("CSS resolution failed")
		m := &MockResolver{
			ResolveCSSPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
				return "", expectedErr
			},
		}

		result, err := m.ResolveCSSPath(context.Background(), "bad.css", "/src")

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveCSSPathCallCount))
	})
}

func TestMockResolver_ResolveAssetPath(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolveAssetPathFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		result, err := m.ResolveAssetPath(context.Background(), "@/icons/arrow.svg", "/src/components/nav.pk")

		assert.NoError(t, err)
		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveAssetPathCallCount))
	})

	t.Run("delegates to ResolveAssetPathFunc", func(t *testing.T) {
		t.Parallel()

		var gotCtx context.Context
		var gotImport, gotContaining string

		m := &MockResolver{
			ResolveAssetPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
				gotCtx = ctx
				gotImport = importPath
				gotContaining = containingFilePath
				return "/resolved/icons/arrow.svg", nil
			},
		}

		ctx := context.Background()
		result, err := m.ResolveAssetPath(ctx, "@/icons/arrow.svg", "/src/components/nav.pk")

		require.NoError(t, err)
		assert.Equal(t, ctx, gotCtx)
		assert.Equal(t, "@/icons/arrow.svg", gotImport)
		assert.Equal(t, "/src/components/nav.pk", gotContaining)
		assert.Equal(t, "/resolved/icons/arrow.svg", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveAssetPathCallCount))
	})

	t.Run("propagates error from ResolveAssetPathFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("asset not found")
		m := &MockResolver{
			ResolveAssetPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
				return "", expectedErr
			},
		}

		result, err := m.ResolveAssetPath(context.Background(), "missing.svg", "/src/nav.pk")

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveAssetPathCallCount))
	})
}

func TestMockResolver_ConvertEntryPointPathToManifestKey(t *testing.T) {
	t.Parallel()

	t.Run("nil ConvertEntryPointPathToManifestKeyFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		result := m.ConvertEntryPointPathToManifestKey("piko.sh/piko/pages/index.pk")

		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ConvertEntryPointPathToManifestKeyCallCount))
	})

	t.Run("delegates to ConvertEntryPointPathToManifestKeyFunc", func(t *testing.T) {
		t.Parallel()

		var gotPath string
		m := &MockResolver{
			ConvertEntryPointPathToManifestKeyFunc: func(entryPointPath string) string {
				gotPath = entryPointPath
				return "pages/index.pk"
			},
		}

		result := m.ConvertEntryPointPathToManifestKey("piko.sh/piko/pages/index.pk")

		assert.Equal(t, "piko.sh/piko/pages/index.pk", gotPath)
		assert.Equal(t, "pages/index.pk", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ConvertEntryPointPathToManifestKeyCallCount))
	})
}

func TestMockResolver_GetModuleDir(t *testing.T) {
	t.Parallel()

	t.Run("nil GetModuleDirFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		result, err := m.GetModuleDir(context.Background(), "piko.sh/piko")

		assert.NoError(t, err)
		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetModuleDirCallCount))
	})

	t.Run("delegates to GetModuleDirFunc", func(t *testing.T) {
		t.Parallel()

		var gotCtx context.Context
		var gotModule string

		m := &MockResolver{
			GetModuleDirFunc: func(ctx context.Context, modulePath string) (string, error) {
				gotCtx = ctx
				gotModule = modulePath
				return "/go/pkg/mod/piko.sh/piko@v1.0.0", nil
			},
		}

		ctx := context.Background()
		result, err := m.GetModuleDir(ctx, "piko.sh/piko")

		require.NoError(t, err)
		assert.Equal(t, ctx, gotCtx)
		assert.Equal(t, "piko.sh/piko", gotModule)
		assert.Equal(t, "/go/pkg/mod/piko.sh/piko@v1.0.0", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetModuleDirCallCount))
	})

	t.Run("propagates error from GetModuleDirFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("module not downloaded")
		m := &MockResolver{
			GetModuleDirFunc: func(_ context.Context, _ string) (string, error) {
				return "", expectedErr
			},
		}

		result, err := m.GetModuleDir(context.Background(), "unknown/mod")

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetModuleDirCallCount))
	})
}

func TestMockResolver_FindModuleBoundary(t *testing.T) {
	t.Parallel()

	t.Run("nil FindModuleBoundaryFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockResolver{}
		modPath, subpath, err := m.FindModuleBoundary(context.Background(), "piko.sh/piko/docs")

		assert.NoError(t, err)
		assert.Equal(t, "", modPath)
		assert.Equal(t, "", subpath)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FindModuleBoundaryCallCount))
	})

	t.Run("delegates to FindModuleBoundaryFunc", func(t *testing.T) {
		t.Parallel()

		var gotCtx context.Context
		var gotImport string

		m := &MockResolver{
			FindModuleBoundaryFunc: func(ctx context.Context, importPath string) (string, string, error) {
				gotCtx = ctx
				gotImport = importPath
				return "piko.sh/piko", "docs", nil
			},
		}

		ctx := context.Background()
		modPath, subpath, err := m.FindModuleBoundary(ctx, "piko.sh/piko/docs")

		require.NoError(t, err)
		assert.Equal(t, ctx, gotCtx)
		assert.Equal(t, "piko.sh/piko/docs", gotImport)
		assert.Equal(t, "piko.sh/piko", modPath)
		assert.Equal(t, "docs", subpath)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FindModuleBoundaryCallCount))
	})

	t.Run("propagates error from FindModuleBoundaryFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("no matching module")
		m := &MockResolver{
			FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
				return "", "", expectedErr
			},
		}

		modPath, subpath, err := m.FindModuleBoundary(context.Background(), "unknown/path")

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, "", modPath)
		assert.Equal(t, "", subpath)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FindModuleBoundaryCallCount))
	})
}

func TestMockResolver_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockResolver
	ctx := context.Background()

	err := m.DetectLocalModule(ctx)
	assert.NoError(t, err)

	name := m.GetModuleName()
	assert.Equal(t, "", name)

	baseDir := m.GetBaseDir()
	assert.Equal(t, "", baseDir)

	pkResult, pkErr := m.ResolvePKPath(ctx, "import", "containing")
	assert.NoError(t, pkErr)
	assert.Equal(t, "", pkResult)

	cssResult, cssErr := m.ResolveCSSPath(ctx, "import.css", "/directory")
	assert.NoError(t, cssErr)
	assert.Equal(t, "", cssResult)

	assetResult, assetErr := m.ResolveAssetPath(ctx, "asset.svg", "/file.pk")
	assert.NoError(t, assetErr)
	assert.Equal(t, "", assetResult)

	manifestKey := m.ConvertEntryPointPathToManifestKey("mod/pages/index.pk")
	assert.Equal(t, "", manifestKey)

	modDir, modDirErr := m.GetModuleDir(ctx, "piko.sh/piko")
	assert.NoError(t, modDirErr)
	assert.Equal(t, "", modDir)

	modPath, subpath, boundaryErr := m.FindModuleBoundary(ctx, "piko.sh/piko/docs")
	assert.NoError(t, boundaryErr)
	assert.Equal(t, "", modPath)
	assert.Equal(t, "", subpath)

	assert.Equal(t, int64(1), atomic.LoadInt64(&m.DetectLocalModuleCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetModuleNameCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetBaseDirCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolvePKPathCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveCSSPathCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveAssetPathCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.ConvertEntryPointPathToManifestKeyCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetModuleDirCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.FindModuleBoundaryCallCount))
}

func TestMockResolver_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockResolver{}
	ctx := context.Background()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_ = m.DetectLocalModule(ctx)
			_ = m.GetModuleName()
			_ = m.GetBaseDir()
			_, _ = m.ResolvePKPath(ctx, "import", "containing")
			_, _ = m.ResolveCSSPath(ctx, "import.css", "/directory")
			_, _ = m.ResolveAssetPath(ctx, "asset.svg", "/file.pk")
			_ = m.ConvertEntryPointPathToManifestKey("entry")
			_, _ = m.GetModuleDir(ctx, "mod")
			_, _, _ = m.FindModuleBoundary(ctx, "path")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DetectLocalModuleCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetModuleNameCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetBaseDirCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ResolvePKPathCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ResolveCSSPathCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ResolveAssetPathCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ConvertEntryPointPathToManifestKeyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetModuleDirCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.FindModuleBoundaryCallCount))
}
