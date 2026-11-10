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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/safedisk"
)

func TestGetSandboxFactory_AllowSymlinks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		allowSymlinks *bool
		name          string
		wantRefused   bool
	}{
		{name: "default allows symlinks", allowSymlinks: nil, wantRefused: false},
		{name: "explicit true allows symlinks", allowSymlinks: new(true), wantRefused: false},
		{name: "false refuses symlinks", allowSymlinks: new(false), wantRefused: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			baseDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(baseDir, "target.txt"), []byte("data"), 0o600))
			if err := os.Symlink("target.txt", filepath.Join(baseDir, "link.txt")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}

			c := NewContainer()
			c.serverConfig.Paths.BaseDir = new(baseDir)
			c.serverConfig.Security.Sandbox.Enabled = new(true)
			c.serverConfig.Security.Sandbox.AllowSymlinks = tc.allowSymlinks

			factory, err := c.GetSandboxFactory()
			require.NoError(t, err)
			sandbox, err := factory.Create("symlink-test", baseDir, safedisk.ModeReadOnly)
			require.NoError(t, err)
			t.Cleanup(func() { _ = sandbox.Close() })

			data, err := sandbox.ReadFile("link.txt")
			if tc.wantRefused {
				require.Error(t, err)
				assert.True(t, errors.Is(err, safedisk.ErrSymlinkNotAllowed))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "data", string(data))
		})
	}
}
