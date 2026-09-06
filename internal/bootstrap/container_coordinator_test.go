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

package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTypeInspectorCacheDir_ProjectBase(t *testing.T) {
	t.Parallel()

	base := t.TempDir()

	got, err := resolveTypeInspectorCacheDir(base)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(base, ".piko", "cache", "types"), got)
}

func TestResolveTypeInspectorCacheDir_FilesystemRootFallsBackToUserCache(t *testing.T) {
	t.Parallel()

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Skipf("no user cache directory available: %v", err)
	}

	got, err := resolveTypeInspectorCacheDir(string(filepath.Separator))
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(userCacheDir, "piko", "cache", "types"), got)
	assert.False(t, isFilesystemRoot(filepath.Dir(got)),
		"the cache directory must not be anchored at a filesystem root")
}

func TestIsFilesystemRoot(t *testing.T) {
	t.Parallel()

	assert.True(t, isFilesystemRoot(string(filepath.Separator)))
	assert.False(t, isFilesystemRoot(t.TempDir()))
}

func TestContainerGetResolverDetectsModule(t *testing.T) {
	testCases := []struct {
		name            string
		baseDirSuffix   string
		allowModuleRoot bool
	}{
		{
			name:          "go.mod in the base directory",
			baseDirSuffix: "",
		},
		{
			name:          "go.mod above the base directory and outside the allowed paths",
			baseDirSuffix: "app",
		},
		{
			name:            "go.mod above the base directory with its directory allowed",
			baseDirSuffix:   "app",
			allowModuleRoot: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			moduleRoot := t.TempDir()
			baseDir := filepath.Join(moduleRoot, tc.baseDirSuffix)
			require.NoError(t, os.MkdirAll(baseDir, 0o750))
			require.NoError(t, os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module example.com/app\n"), 0o600))

			c := NewContainer()
			c.serverConfig.Paths.BaseDir = new(baseDir)
			c.serverConfig.Security.Sandbox.Enabled = new(true)
			if tc.allowModuleRoot {
				c.serverConfig.Security.Sandbox.AllowedPaths = []string{moduleRoot}
			}

			resolver, err := c.GetResolver()

			require.NoError(t, err)
			assert.Equal(t, "example.com/app", resolver.GetModuleName())
		})
	}
}
