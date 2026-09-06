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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

type fakeModuleSandboxFactory struct {
	sandbox     safedisk.Sandbox
	createErr   error
	createCalls int
	allowed     bool
}

func (f *fakeModuleSandboxFactory) Create(_ string, _ string, _ safedisk.Mode) (safedisk.Sandbox, error) {
	f.createCalls++
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.sandbox, nil
}

func (f *fakeModuleSandboxFactory) MustCreate(purpose string, path string, mode safedisk.Mode) safedisk.Sandbox {
	sandbox, err := f.Create(purpose, path, mode)
	if err != nil {
		panic(err)
	}
	return sandbox
}

func (f *fakeModuleSandboxFactory) IsPathAllowed(_ string) bool {
	return f.allowed
}

func (*fakeModuleSandboxFactory) AllowedPaths() []string {
	return nil
}

func TestReadModuleName(t *testing.T) {
	createTempGoMod := func(t *testing.T, content string) string {
		t.Helper()
		directory := t.TempDir()
		goModPath := filepath.Join(directory, "go.mod")
		require.NoError(t, os.WriteFile(goModPath, []byte(content), 0644))
		return goModPath
	}

	testCases := []struct {
		name           string
		goModContent   string
		expectedModule string
		errContains    string
		expectErr      bool
	}{
		{
			name:           "Standard module line",
			goModContent:   "module my/project/name\n\ngo 1.25\n",
			expectedModule: "my/project/name",
			expectErr:      false,
		},
		{
			name:           "Module line with extra whitespace",
			goModContent:   "\t module    my/project/name   \n",
			expectedModule: "my/project/name",
			expectErr:      false,
		},
		{
			name:           "Module line with comments before",
			goModContent:   "# This is a comment\n// Another comment\nmodule myproject\n",
			expectedModule: "myproject",
			expectErr:      false,
		},
		{
			name:         "No module line",
			goModContent: "go 1.25\n\nrequire github.com/stretchr/testify v1.8.0\n",
			expectErr:    true,
			errContains:  "no 'module' line found",
		},
		{
			name:         "Empty file",
			goModContent: "",
			expectErr:    true,
			errContains:  "no 'module' line found",
		},
		{
			name:           "Module line with trailing comment",
			goModContent:   "module my/project // the main module\n",
			expectedModule: "my/project",
			expectErr:      false,
		},
		{
			name:           "Quoted module path",
			goModContent:   "module \"my/quoted/project\"\n",
			expectedModule: "my/quoted/project",
			expectErr:      false,
		},
		{
			name:         "Module keyword without a path",
			goModContent: "module\ngo 1.25\n",
			expectErr:    true,
			errContains:  "no 'module' line found",
		},
		{
			name:         "Typo in module directive",
			goModContent: "modul myproject\ngo 1.25\n",
			expectErr:    true,
			errContains:  "no 'module' line found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			goModPath := createTempGoMod(t, tc.goModContent)

			moduleName, err := ReadModuleName(context.Background(), goModPath, nil)

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedModule, moduleName)
			}
		})
	}

	t.Run("Non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(t.TempDir(), "go.mod")
		_, err := ReadModuleName(context.Background(), nonExistentPath, nil)
		require.Error(t, err)
	})
}

func TestReadModuleNameWithSandboxFactory(t *testing.T) {
	testCases := []struct {
		createErr           error
		name                string
		expectedModule      string
		errContains         string
		expectedCreateCalls int
		allowed             bool
		expectErr           bool
	}{
		{
			name:                "factory allowing the module directory supplies the sandbox",
			allowed:             true,
			expectedModule:      "injected/module",
			expectedCreateCalls: 1,
		},
		{
			name:                "module directory outside the allowed paths is read through a dedicated sandbox",
			allowed:             false,
			expectedModule:      "disk/project",
			expectedCreateCalls: 0,
		},
		{
			name:                "factory failure is reported",
			allowed:             true,
			createErr:           errors.New("sandbox refused"),
			expectedCreateCalls: 1,
			expectErr:           true,
			errContains:         "creating sandbox for go.mod",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectRoot := t.TempDir()
			goModPath := filepath.Join(projectRoot, "go.mod")
			require.NoError(t, os.WriteFile(goModPath, []byte("module disk/project\n"), 0644))

			injected := safedisk.NewMockSandbox(projectRoot, safedisk.ModeReadOnly)
			injected.AddFile("go.mod", []byte("module injected/module\n"))
			factory := &fakeModuleSandboxFactory{
				sandbox:     injected,
				createErr:   tc.createErr,
				createCalls: 0,
				allowed:     tc.allowed,
			}

			moduleName, err := ReadModuleName(context.Background(), goModPath, factory)

			assert.Equal(t, tc.expectedCreateCalls, factory.createCalls)
			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedModule, moduleName)
		})
	}
}

func TestReadModuleNameRejectsOversizedGoMod(t *testing.T) {
	goModPath := filepath.Join(t.TempDir(), "go.mod")
	require.NoError(t, os.WriteFile(goModPath, []byte("module big/project\n"), 0644))
	require.NoError(t, os.Truncate(goModPath, maxGoModFileBytes+1))

	_, err := ReadModuleName(context.Background(), goModPath, nil)

	require.ErrorIs(t, err, safedisk.ErrFileExceedsLimit)
}

func TestCloseModuleSandbox(t *testing.T) {
	testCases := []struct {
		closeErr error
		name     string
	}{
		{name: "close succeeds", closeErr: nil},
		{name: "close failure is logged rather than propagated", closeErr: errors.New("close failed")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sandbox := safedisk.NewMockSandbox(t.TempDir(), safedisk.ModeReadOnly)
			sandbox.CloseErr = tc.closeErr

			closeModuleSandbox(context.Background(), sandbox, "/project/go.mod")

			assert.Equal(t, 1, sandbox.CallCounts["Close"])
		})
	}
}
