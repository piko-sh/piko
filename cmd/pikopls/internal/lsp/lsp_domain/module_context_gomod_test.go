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

package lsp_domain

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

func TestTryReadGoMod(t *testing.T) {
	testCases := []struct {
		setup       func(t *testing.T, directory string)
		name        string
		errContains string
		restrict    bool
		expectFound bool
		expectErr   bool
	}{
		{
			name: "valid go.mod is found",
			setup: func(t *testing.T, directory string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module github.com/example/myapp\n\ngo 1.24\n"), 0o600))
			},
			expectFound: true,
		},
		{
			name: "valid go.mod outside the factory's allowed paths is still read",
			setup: func(t *testing.T, directory string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module github.com/example/myapp\n"), 0o600))
			},
			restrict:    true,
			expectFound: true,
		},
		{
			name:        "directory without go.mod is skipped",
			setup:       func(*testing.T, string) {},
			expectFound: false,
		},
		{
			name: "directory named go.mod is skipped",
			setup: func(t *testing.T, directory string) {
				t.Helper()
				require.NoError(t, os.Mkdir(filepath.Join(directory, "go.mod"), 0o750))
			},
			expectFound: false,
		},
		{
			name: "go.mod without a module line is an error",
			setup: func(t *testing.T, directory string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("go 1.24\n"), 0o600))
			},
			expectErr:   true,
			errContains: "no 'module' line",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			tc.setup(t, directory)

			var factory safedisk.Factory
			if tc.restrict {
				otherRoot := t.TempDir()
				restricted, err := safedisk.NewFactory(safedisk.FactoryConfig{
					CWD:          otherRoot,
					AllowedPaths: []string{otherRoot},
					Enabled:      true,
				})
				require.NoError(t, err)
				factory = restricted
			}

			found, err := tryReadGoMod(context.Background(), directory, factory)

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectFound, found)
		})
	}
}

func TestSourceDirectoryExists(t *testing.T) {
	testCases := []struct {
		name         string
		relDir       string
		restrict     bool
		createSource bool
		expected     bool
	}{
		{
			name:         "existing source directory is reported",
			relDir:       "pages",
			createSource: true,
			expected:     true,
		},
		{
			name:     "missing source directory is skipped",
			relDir:   "pages",
			expected: false,
		},
		{
			name:         "a factory refusing the module root skips the walk",
			relDir:       "pages",
			createSource: true,
			restrict:     true,
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			moduleRoot := t.TempDir()
			if tc.createSource {
				require.NoError(t, os.Mkdir(filepath.Join(moduleRoot, tc.relDir), 0o750))
			}

			mc := &ModuleContext{}
			mc.ModuleRoot = moduleRoot
			if tc.restrict {
				otherRoot := t.TempDir()
				factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
					CWD:          otherRoot,
					AllowedPaths: []string{otherRoot},
					Enabled:      true,
				})
				require.NoError(t, err)
				mc.sandboxFactory = factory
			}

			exists := mc.sourceDirectoryExists(context.Background(), tc.relDir, filepath.Join(moduleRoot, tc.relDir))

			assert.Equal(t, tc.expected, exists)
		})
	}
}
