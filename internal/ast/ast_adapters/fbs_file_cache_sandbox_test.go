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

package ast_adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewFbsFileCache_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	t.Run("creates cache with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})

		require.NoError(t, err)
		require.NotNil(t, cache)
		defer cache.Shutdown(context.Background())

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{TagName: "div"},
			},
		}

		err = cache.Set(context.Background(), "test-key", ast)
		require.NoError(t, err)

		retrieved, err := cache.Get(context.Background(), "test-key")
		require.NoError(t, err)
		assert.Equal(t, "div", retrieved.RootNodes[0].TagName)
	})

	t.Run("returns error when MkdirAll fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.MkdirAllErr = errors.New("disk full")

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})

		require.Error(t, err)
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "failed to create cache base directory")
	})

	t.Run("returns error when BaseDir is empty", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "",
			Sandbox: sandbox,
		})

		require.Error(t, err)
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "baseDir must be provided")
	})
}

func TestFbsFileCache_Get_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	t.Run("returns cache miss when file not found", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		_, err = cache.Get(context.Background(), "nonexistent-key")

		assert.ErrorIs(t, err, ast_domain.ErrCacheMiss)
	})

	t.Run("returns error when ReadFile fails with non-NotExist error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
		}
		require.NoError(t, cache.Set(context.Background(), "test-key", ast))

		sandbox.ReadFileErr = errors.New("disk read error")

		_, err = cache.Get(context.Background(), "test-key")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "disk read error")
	})
}

func TestFbsFileCache_Set_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	t.Run("returns error when WriteFile fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		sandbox.WriteFileErr = errors.New("disk write error")

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
		}

		err = cache.Set(context.Background(), "test-key", ast)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write temporary cache file")
	})

	t.Run("returns error when Rename fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		sandbox.RenameErr = errors.New("rename failed")

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
		}

		err = cache.Set(context.Background(), "test-key", ast)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to atomically move cache file")
	})
}

func TestFbsFileCache_SetWithTTL_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	t.Run("stores and retrieves entry with TTL", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "span"}},
		}

		err = cache.SetWithTTL(context.Background(), "ttl-key", ast, 1*time.Hour)
		require.NoError(t, err)

		retrieved, err := cache.Get(context.Background(), "ttl-key")
		require.NoError(t, err)
		assert.Equal(t, "span", retrieved.RootNodes[0].TagName)
		assert.NotNil(t, retrieved.ExpiresAtUnixNano)
	})

	t.Run("enqueues deletion for zero or negative TTL", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "p"}},
		}

		err = cache.SetWithTTL(context.Background(), "zero-ttl-key", ast, 0)
		require.NoError(t, err)
	})
}

func TestFbsFileCache_Delete_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing entry", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
		}

		require.NoError(t, cache.Set(context.Background(), "delete-key", ast))

		err = cache.Delete(context.Background(), "delete-key")
		require.NoError(t, err)

		_, err = cache.Get(context.Background(), "delete-key")
		assert.ErrorIs(t, err, ast_domain.ErrCacheMiss)
	})

	t.Run("returns no error for nonexistent key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		err = cache.Delete(context.Background(), "nonexistent-key")
		require.NoError(t, err)
	})

	t.Run("returns error when Remove fails with non-NotExist error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache, err := newFbsFileCache(fbsFileCacheConfig{
			BaseDir: "/cache",
			Sandbox: sandbox,
		})
		require.NoError(t, err)
		defer cache.Shutdown(context.Background())

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
		}
		require.NoError(t, cache.Set(context.Background(), "error-key", ast))

		sandbox.RemoveErr = errors.New("permission denied")

		err = cache.Delete(context.Background(), "error-key")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete cache file")
	})
}
