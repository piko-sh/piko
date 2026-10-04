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

package storage_domain

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

func TestResolveStorageTempSandbox(t *testing.T) {
	t.Parallel()

	injected := safedisk.NewMockSandbox(os.TempDir(), safedisk.ModeReadWrite)

	allowingFactory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          t.TempDir(),
		AllowedPaths: []string{os.TempDir()},
		Enabled:      true,
	})
	require.NoError(t, err)

	rejectingFactory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          t.TempDir(),
		AllowedPaths: []string{t.TempDir()},
		Enabled:      true,
	})
	require.NoError(t, err)

	testCases := []struct {
		factory      safedisk.Factory
		injected     safedisk.Sandbox
		name         string
		wantInjected bool
	}{
		{
			name:         "injected sandbox takes precedence over the factory",
			injected:     injected,
			factory:      allowingFactory,
			wantInjected: true,
		},
		{
			name:    "factory creates the sandbox when none is injected",
			factory: allowingFactory,
		},
		{
			name:    "factory failure falls back to a temp directory sandbox",
			factory: rejectingFactory,
		},
		{
			name: "no injected sandbox or factory uses a temp directory sandbox",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config := defaultServiceConfig()
			config.TempSandbox = testCase.injected
			config.TempSandboxFactory = testCase.factory

			sandbox := resolveStorageTempSandbox(context.Background(), &config)
			require.NotNil(t, sandbox)

			if testCase.wantInjected {
				assert.Same(t, injected, sandbox)
				return
			}

			t.Cleanup(func() { _ = sandbox.Close() })
			assert.Equal(t, safedisk.ModeReadWrite, sandbox.Mode())
		})
	}
}

func TestResolveStorageTempSandboxReturnsNilWhenTempDirectoryIsUnusable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.TempDir does not read TMPDIR on windows")
	}

	blockingFile := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blockingFile, nil, 0o600))
	t.Setenv("TMPDIR", filepath.Join(blockingFile, "temp"))

	config := defaultServiceConfig()

	assert.Nil(t, resolveStorageTempSandbox(context.Background(), &config))
}
