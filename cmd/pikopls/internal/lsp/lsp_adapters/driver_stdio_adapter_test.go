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

package lsp_adapters

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/cmd/pikopls/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/safedisk"
)

type stdioStubCoordinatorService struct {
	coordinator_domain.CoordinatorService
}

type stdioStubResolverPort struct {
	resolver_domain.ResolverPort
}

type stdioStubFSReader struct {
	annotator_domain.FSReaderPort
}

func validStdioDeps(t *testing.T) StdioAdapterDeps {
	t.Helper()
	return StdioAdapterDeps{
		CoordinatorService:   &stdioStubCoordinatorService{},
		Resolver:             &stdioStubResolverPort{},
		TypeInspectorManager: inspector_domain.NewTypeBuilder(inspector_dto.Config{}),
		DocCache:             lsp_domain.NewDocumentCache(),
		LSPReader:            &stdioStubFSReader{},
		PathsConfig:          &config.PathsConfig{},
	}
}

func TestNewStdioAdapter(t *testing.T) {
	t.Parallel()

	t.Run("returns adapter when all dependencies are non-nil", func(t *testing.T) {
		t.Parallel()

		adapter, err := NewStdioAdapter(validStdioDeps(t))

		require.NoError(t, err)
		require.NotNil(t, adapter)
	})

	t.Run("returns error when coordinatorService is nil", func(t *testing.T) {
		t.Parallel()

		deps := validStdioDeps(t)
		deps.CoordinatorService = nil
		adapter, err := NewStdioAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "coordinatorService cannot be nil")
	})

	t.Run("returns error when resolver is nil", func(t *testing.T) {
		t.Parallel()

		deps := validStdioDeps(t)
		deps.Resolver = nil
		adapter, err := NewStdioAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "resolver cannot be nil")
	})

	t.Run("returns error when typeInspectorManager is nil", func(t *testing.T) {
		t.Parallel()

		deps := validStdioDeps(t)
		deps.TypeInspectorManager = nil
		adapter, err := NewStdioAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "typeInspectorManager cannot be nil")
	})

	t.Run("returns error when docCache is nil", func(t *testing.T) {
		t.Parallel()

		deps := validStdioDeps(t)
		deps.DocCache = nil
		adapter, err := NewStdioAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "docCache cannot be nil")
	})

	t.Run("returns error when lspReader is nil", func(t *testing.T) {
		t.Parallel()

		deps := validStdioDeps(t)
		deps.LSPReader = nil
		adapter, err := NewStdioAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "lspReader cannot be nil")
	})

	t.Run("returns error when pathsConfig is nil", func(t *testing.T) {
		t.Parallel()

		deps := validStdioDeps(t)
		deps.PathsConfig = nil
		adapter, err := NewStdioAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "pathsConfig cannot be nil")
	})
}

func TestSetupLogFile(t *testing.T) {
	t.Parallel()

	t.Run("creates log file with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		logSetup := setupLogFile(t.Context(), "/tmp/pikopls.log.protocol", sandbox, nil)

		require.NotNil(t, logSetup)
		assert.NotNil(t, logSetup.logFile)

		assert.Nil(t, logSetup.sandbox)
		logSetup.close()
	})

	t.Run("returns nil when OpenFile fails with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.OpenFileErr = errors.New("permission denied")

		logSetup := setupLogFile(t.Context(), "/tmp/pikopls.log.protocol", sandbox, nil)

		assert.Nil(t, logSetup)
	})

	t.Run("returns nil without touching the filesystem when no path is configured", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.OpenFileErr = errors.New("must not be called")

		assert.Nil(t, setupLogFile(t.Context(), "", sandbox, nil))
		assert.Nil(t, setupLogFile(t.Context(), "", nil, nil))
	})

	t.Run("creates the protocol log at the configured path", func(t *testing.T) {
		t.Parallel()

		logPath := filepath.Join(t.TempDir(), "pikopls-test.log.protocol")

		logSetup := setupLogFile(t.Context(), logPath, nil, nil)
		require.NotNil(t, logSetup)
		assert.NotNil(t, logSetup.sandbox, "a sandbox it created itself is owned and closed by the resources")
		logSetup.close()

		_, err := os.Stat(logPath)
		assert.NoError(t, err)
	})
}

func TestSetupLogFile_ReportsSandboxFailures(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when the factory cannot create the sandbox", func(t *testing.T) {
		t.Parallel()

		factory := &failingSandboxFactory{err: errors.New("sandbox denied")}
		assert.Nil(t, setupLogFile(t.Context(), filepath.Join(t.TempDir(), "pikopls.log.protocol"), nil, factory))
	})

	t.Run("returns nil when the log directory cannot be created", func(t *testing.T) {
		t.Parallel()

		blocker := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
		assert.Nil(t, setupLogFile(t.Context(), filepath.Join(blocker, "pikopls.log.protocol"), nil, nil))
	})
}

func TestOpenLogSandbox(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	factory, err := safedisk.NewCLIFactory(directory)
	require.NoError(t, err)

	sandbox, err := openLogSandbox(directory, factory)
	require.NoError(t, err)
	require.NoError(t, sandbox.Close())

	sandbox, err = openLogSandbox(directory, nil)
	require.NoError(t, err)
	require.NoError(t, sandbox.Close())

	_, err = openLogSandbox(directory, &failingSandboxFactory{err: errors.New("sandbox denied")})
	require.Error(t, err)
}

type failingSandboxFactory struct {
	safedisk.Factory
	err error
}

func (f *failingSandboxFactory) Create(string, string, safedisk.Mode) (safedisk.Sandbox, error) {
	return nil, f.err
}
