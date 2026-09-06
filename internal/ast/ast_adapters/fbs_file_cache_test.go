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
	"encoding/binary"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestFbsFileCache_GetTreatsCorruptFilesAsMissesAndDeletesThem(t *testing.T) {
	testCases := []struct {
		mutate func([]byte) []byte
		name   string
	}{
		{name: "truncated to the schema hash", mutate: func(data []byte) []byte { return data[:32] }},
		{name: "truncated to half", mutate: func(data []byte) []byte { return data[:len(data)/2] }},
		{name: "root offset beyond payload", mutate: func(data []byte) []byte {
			binary.LittleEndian.PutUint32(data[32:], 0xfffffff0)
			return data
		}},
		{name: "every payload byte inverted", mutate: func(data []byte) []byte {
			for index := 32; index < len(data); index++ {
				data[index] = ^data[index]
			}
			return data
		}},
		{name: "random bytes", mutate: func(data []byte) []byte {
			return []byte("this is not a FlatBuffers AST at all, just some text")
		}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := t.Context()
			sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
			cache, err := newFbsFileCache(fbsFileCacheConfig{BaseDir: "/cache", Sandbox: sandbox})
			require.NoError(t, err)
			t.Cleanup(func() { cache.Shutdown(context.Background()) })
			cache.Start(ctx)

			tree, err := ast_domain.Parse(ctx, corruptionTestTemplate, "corrupt.pk", nil)
			require.NoError(t, err)
			require.NoError(t, cache.Set(ctx, "corrupt-key", tree))

			filePath := cache.getFilePath("corrupt-key")
			stored, err := sandbox.ReadFile(filePath)
			require.NoError(t, err)
			require.NoError(t, sandbox.WriteFile(filePath, testCase.mutate(append([]byte(nil), stored...)), cacheFilePermissions))

			ast, err := cache.Get(ctx, "corrupt-key")
			require.ErrorIs(t, err, ast_domain.ErrCacheMiss)
			assert.Nil(t, ast)

			assert.Eventually(t, func() bool {
				_, readErr := sandbox.ReadFile(filePath)
				return errors.Is(readErr, fs.ErrNotExist)
			}, 5*time.Second, 5*time.Millisecond)
		})
	}
}

func TestFbsFileCache_StartAndShutdownAreIdempotent(t *testing.T) {
	sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
	cache, err := newFbsFileCache(fbsFileCacheConfig{BaseDir: "/cache", Sandbox: sandbox, NumDeletionWorkers: 2})
	require.NoError(t, err)

	cache.Start(t.Context())
	cache.Start(t.Context())
	cache.Shutdown(context.Background())
	cache.Shutdown(context.Background())

	assert.NotPanics(t, func() { cache.enqueueDeletion(context.Background(), "after-shutdown") })
	assert.Zero(t, sandbox.CallCounts["Close"], "an injected sandbox is closed by its owner")
}

func TestFbsFileCache_EnqueueDeletionDropsWhenQueueIsFull(t *testing.T) {
	cache, err := newFbsFileCache(fbsFileCacheConfig{
		BaseDir:           "/cache",
		Sandbox:           safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite),
		DeletionQueueSize: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { cache.Shutdown(context.Background()) })

	cache.enqueueDeletion(context.Background(), "first")
	cache.enqueueDeletion(context.Background(), "second")

	require.Len(t, cache.deleteChan, 1)
	assert.Equal(t, "first", <-cache.deleteChan)
}

func TestFbsFileCache_ShutdownWithoutStart(t *testing.T) {
	cache, err := newFbsFileCache(fbsFileCacheConfig{
		BaseDir: "/cache",
		Sandbox: safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite),
	})
	require.NoError(t, err)

	assert.NotPanics(t, func() { cache.Shutdown(context.Background()) })
}

func TestFbsFileCache_WorkersStopWhenContextIsCancelled(t *testing.T) {
	cache, err := newFbsFileCache(fbsFileCacheConfig{
		BaseDir: "/cache",
		Sandbox: safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite),
	})
	require.NoError(t, err)
	t.Cleanup(func() { cache.Shutdown(context.Background()) })

	ctx, cancel := context.WithCancelCause(context.Background())
	cache.Start(ctx)
	cancel(errors.New("test finished with the workers"))

	stopped := make(chan struct{})
	go func() {
		cache.wg.Wait()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "deletion workers did not stop after their context was cancelled")
	}
}

func TestFbsFileCache_OwnedSandboxIsClosedOnShutdown(t *testing.T) {
	directory := t.TempDir()
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          directory,
		AllowedPaths: []string{directory},
		Enabled:      true,
	})
	require.NoError(t, err)

	cache, err := newFbsFileCache(fbsFileCacheConfig{BaseDir: directory, SandboxFactory: factory})
	require.NoError(t, err)
	require.True(t, cache.ownsSandbox)

	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}}}
	require.NoError(t, cache.Set(context.Background(), "owned-key", tree))

	cache.Shutdown(context.Background())

	_, err = cache.sandbox.ReadFile(cache.getFilePath("owned-key"))
	assert.Error(t, err, "reads through a closed sandbox fail")
}

func TestResolveCacheSandbox(t *testing.T) {
	directory := t.TempDir()
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          directory,
		AllowedPaths: []string{directory},
		Enabled:      true,
	})
	require.NoError(t, err)

	injected := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)

	testCases := []struct {
		name      string
		config    fbsFileCacheConfig
		wantOwned bool
		wantError bool
	}{
		{name: "injected sandbox is borrowed", config: fbsFileCacheConfig{BaseDir: directory, Sandbox: injected}},
		{name: "factory sandbox is owned", config: fbsFileCacheConfig{BaseDir: directory, SandboxFactory: factory}, wantOwned: true},
		{name: "default sandbox is owned", config: fbsFileCacheConfig{BaseDir: directory}, wantOwned: true},
		{name: "factory rejects a path outside its allowed paths", config: fbsFileCacheConfig{BaseDir: t.TempDir(), SandboxFactory: factory}, wantError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sandbox, owned, err := resolveCacheSandbox(testCase.config)
			if testCase.wantError {
				require.Error(t, err)
				assert.Nil(t, sandbox)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, sandbox)
			assert.Equal(t, testCase.wantOwned, owned)
			if owned {
				assert.NoError(t, sandbox.Close())
			}
		})
	}
}
