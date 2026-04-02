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

package typegen_domain

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/typegen/typegen_frontend"
	"piko.sh/piko/wdk/safedisk"
)

const testActionStubContent = "declare const actions: Record<string, unknown>;\n"

func newTestTypeDefService() *TypeDefinitionService {
	return NewTypeDefinitionService(typegen_frontend.EmbeddedTypeDefinitions, testActionStubContent)
}

func TestNewTypeDefinitionService(t *testing.T) {
	t.Parallel()
	service := newTestTypeDefService()
	require.NotNil(t, service)
}

func TestEnsureTypeDefinitions(t *testing.T) {
	t.Parallel()

	t.Run("writes files to empty directory", func(t *testing.T) {
		t.Parallel()
		destDir := t.TempDir()
		service := newTestTypeDefService()

		err := service.EnsureTypeDefinitions(context.Background(), destDir)
		require.NoError(t, err)

		pikoIDEPath := filepath.Join(destDir, "piko-ide.d.ts")
		content, err := os.ReadFile(pikoIDEPath)
		require.NoError(t, err)
		assert.NotEmpty(t, content)

		actionStubPath := filepath.Join(destDir, "piko-actions.d.ts")
		content, err = os.ReadFile(actionStubPath)
		require.NoError(t, err)
		assert.Equal(t, testActionStubContent, string(content))
	})
}

func TestEnsureTypeDefinitionsWithOptions_OnlyIfNotExists(t *testing.T) {
	t.Parallel()

	t.Run("preserves existing files", func(t *testing.T) {
		t.Parallel()
		destDir := t.TempDir()
		service := newTestTypeDefService()

		sentinelContent := "// sentinel\n"
		err := os.WriteFile(filepath.Join(destDir, "piko-actions.d.ts"), []byte(sentinelContent), 0o640)
		require.NoError(t, err)

		err = service.EnsureTypeDefinitionsWithOptions(context.Background(), destDir, EnsureOptions{
			OnlyIfNotExists: true,
		})
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(destDir, "piko-actions.d.ts"))
		require.NoError(t, err)
		assert.Equal(t, sentinelContent, string(content))
	})

	t.Run("writes missing files", func(t *testing.T) {
		t.Parallel()
		destDir := t.TempDir()
		service := newTestTypeDefService()

		err := service.EnsureTypeDefinitionsWithOptions(context.Background(), destDir, EnsureOptions{
			OnlyIfNotExists: true,
		})
		require.NoError(t, err)

		actionStubPath := filepath.Join(destDir, "piko-actions.d.ts")
		content, err := os.ReadFile(actionStubPath)
		require.NoError(t, err)
		assert.Equal(t, testActionStubContent, string(content))
	})
}

func TestEnsureTypeDefinitionsWithOptions_OverwritesByDefault(t *testing.T) {
	t.Parallel()
	destDir := t.TempDir()
	service := newTestTypeDefService()

	sentinelContent := "// sentinel\n"
	err := os.WriteFile(filepath.Join(destDir, "piko-actions.d.ts"), []byte(sentinelContent), 0o640)
	require.NoError(t, err)

	err = service.EnsureTypeDefinitionsWithOptions(context.Background(), destDir, EnsureOptions{})
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(destDir, "piko-actions.d.ts"))
	require.NoError(t, err)
	assert.Equal(t, testActionStubContent, string(content))
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing file", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		err := os.WriteFile(filepath.Join(directory, "exists.txt"), []byte("hello"), 0o640)
		require.NoError(t, err)

		factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
			CWD:          directory,
			AllowedPaths: []string{directory},
			Enabled:      true,
		})
		require.NoError(t, err)

		sandbox, err := factory.Create("test", directory, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		exists, err := fileExists(sandbox, "exists.txt")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for missing file", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()

		factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
			CWD:          directory,
			AllowedPaths: []string{directory},
			Enabled:      true,
		})
		require.NoError(t, err)

		sandbox, err := factory.Create("test", directory, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		exists, err := fileExists(sandbox, "missing.txt")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
