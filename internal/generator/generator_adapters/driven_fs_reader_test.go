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

package generator_adapters

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewFSReader(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadOnly)
	defer sandbox.Close()
	reader := NewFSReader(sandbox)

	require.NotNil(t, reader)

	var _ generator_domain.FSReaderPort = reader
}

func TestFSReader_ReadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*safedisk.MockSandbox)
		filePath  string
		wantData  []byte
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.AddFile("test.txt", []byte("hello world"))
			},
			filePath: "test.txt",
			wantData: []byte("hello world"),
		},
		{
			name:      "file not found",
			setupMock: func(_ *safedisk.MockSandbox) {},
			filePath:  "nonexistent.txt",
			wantErr:   true,
		},
		{
			name: "read file error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.ReadFileErr = errors.New("disk error")
			},
			filePath: "test.txt",
			wantErr:  true,
		},
		{
			name: "empty file",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.AddFile("empty.txt", []byte{})
			},
			filePath: "empty.txt",
			wantData: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadOnly)
			defer sandbox.Close()
			tt.setupMock(sandbox)

			reader := NewFSReader(sandbox)
			data, err := reader.ReadFile(context.Background(), tt.filePath)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantData, data)
		})
	}
}

func TestFSReader_ReadFile_ExternalModule(t *testing.T) {
	t.Parallel()

	moduleDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module example.com/ui\n\ngo 1.26\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(moduleDir, "lib", "icons"), 0o750))
	assetPath := filepath.Join(moduleDir, "lib", "icons", "logo.svg")
	require.NoError(t, os.WriteFile(assetPath, []byte("<svg/>"), 0o600))

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	defer sandbox.Close()
	reader := NewFSReader(sandbox)
	t.Cleanup(func() { _ = reader.Close() })

	data, err := reader.ReadFile(context.Background(), assetPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("<svg/>"), data)

	otherPath := filepath.Join(moduleDir, "go.mod")
	_, err = reader.ReadFile(context.Background(), otherPath)
	require.NoError(t, err)
	assert.Len(t, reader.externalSandboxes, 1)

	require.NoError(t, reader.Close())
	assert.Empty(t, reader.externalSandboxes)
}

func TestFSReader_ReadFile_ExternalNoModuleRoot(t *testing.T) {
	t.Parallel()

	orphanDir := t.TempDir()
	orphanPath := filepath.Join(orphanDir, "stray.svg")
	require.NoError(t, os.WriteFile(orphanPath, []byte("x"), 0o600))

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	defer sandbox.Close()
	reader := NewFSReader(sandbox)
	t.Cleanup(func() { _ = reader.Close() })

	_, err := reader.ReadFile(context.Background(), orphanPath)
	require.Error(t, err)
	assert.ErrorIs(t, err, errNoModuleRoot)
}

func TestFindModuleRoot(t *testing.T) {
	t.Parallel()

	moduleDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module example.com/ui\n"), 0o600))
	nestedDir := filepath.Join(moduleDir, "partials", "deep")
	require.NoError(t, os.MkdirAll(nestedDir, 0o750))

	t.Run("finds nearest go.mod above file", func(t *testing.T) {
		t.Parallel()
		root, err := findModuleRoot(filepath.Join(nestedDir, "card.pk"))
		require.NoError(t, err)
		assert.Equal(t, moduleDir, root)
	})

	t.Run("errors when no go.mod found", func(t *testing.T) {
		t.Parallel()
		orphan := t.TempDir()
		_, err := findModuleRoot(filepath.Join(orphan, "stray.svg"))
		require.ErrorIs(t, err, errNoModuleRoot)
	})
}
