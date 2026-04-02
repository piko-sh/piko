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

package lifecycle_adapters

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestBuildService_walkAndStreamFiles_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("walks and streams all files in directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		libDir := filepath.Join(tmpDir, "lib")
		iconsDir := filepath.Join(libDir, "icons")
		require.NoError(t, os.MkdirAll(iconsDir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(libDir, "style.css"), []byte("body{}"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(iconsDir, "arrow.svg"), []byte("<svg>"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(iconsDir, "home.svg"), []byte("<svg>"), 0644))

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:         tmpDir,
				AssetsSourceDir: "lib",
			},
		}

		ctx := context.Background()
		fileEvents := make(chan lifecycle_dto.FileEvent, 10)
		var fileCount atomic.Int64
		err := bs.walkAndStreamFiles(ctx, fileEvents, &fileCount)
		close(fileEvents)

		require.NoError(t, err)
		assert.Equal(t, int64(3), fileCount.Load())

		var events []lifecycle_dto.FileEvent
		for event := range fileEvents {
			events = append(events, event)
		}

		assert.Len(t, events, 3)
		for _, event := range events {
			assert.Equal(t, lifecycle_dto.FileEventTypeCreate, event.Type)
		}
	})

	t.Run("skips directories - only streams files", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		libDir := filepath.Join(tmpDir, "lib")
		nestedDir := filepath.Join(libDir, "nested", "deep")
		require.NoError(t, os.MkdirAll(nestedDir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "file.txt"), []byte("data"), 0644))

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:         tmpDir,
				AssetsSourceDir: "lib",
			},
		}

		ctx := context.Background()
		fileEvents := make(chan lifecycle_dto.FileEvent, 10)
		var fileCount atomic.Int64
		err := bs.walkAndStreamFiles(ctx, fileEvents, &fileCount)
		close(fileEvents)

		require.NoError(t, err)
		assert.Equal(t, int64(1), fileCount.Load(), "Should only count files, not directories")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		libDir := filepath.Join(tmpDir, "lib")
		require.NoError(t, os.MkdirAll(libDir, 0755))

		for i := range 100 {
			require.NoError(t, os.WriteFile(
				filepath.Join(libDir, filepath.Base(t.Name())+string(rune('a'+i%26))+".txt"),
				[]byte("data"),
				0644,
			))
		}

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:         tmpDir,
				AssetsSourceDir: "lib",
			},
		}

		ctx, cancel := context.WithCancelCause(context.Background())
		fileEvents := make(chan lifecycle_dto.FileEvent, 1)
		var fileCount atomic.Int64

		cancel(fmt.Errorf("test: simulating cancelled context"))

		err := bs.walkAndStreamFiles(ctx, fileEvents, &fileCount)
		close(fileEvents)

		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("handles empty directory gracefully", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		libDir := filepath.Join(tmpDir, "lib")
		require.NoError(t, os.MkdirAll(libDir, 0755))

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:         tmpDir,
				AssetsSourceDir: "lib",
			},
		}

		ctx := context.Background()
		fileEvents := make(chan lifecycle_dto.FileEvent, 10)
		var fileCount atomic.Int64
		err := bs.walkAndStreamFiles(ctx, fileEvents, &fileCount)
		close(fileEvents)

		require.NoError(t, err)
		assert.Equal(t, int64(0), fileCount.Load())
	})

	t.Run("walks multiple directories", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		libDir := filepath.Join(tmpDir, "lib")
		componentsDir := filepath.Join(tmpDir, "components")
		require.NoError(t, os.MkdirAll(libDir, 0755))
		require.NoError(t, os.MkdirAll(componentsDir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(libDir, "style.css"), []byte("css"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(componentsDir, "card.go"), []byte("go"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(componentsDir, "button.go"), []byte("go"), 0644))

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:             tmpDir,
				AssetsSourceDir:     "lib",
				ComponentsSourceDir: "components",
			},
		}

		ctx := context.Background()
		fileEvents := make(chan lifecycle_dto.FileEvent, 10)
		var fileCount atomic.Int64
		err := bs.walkAndStreamFiles(ctx, fileEvents, &fileCount)
		close(fileEvents)

		require.NoError(t, err)
		assert.Equal(t, int64(3), fileCount.Load())
	})
}

func TestBuildService_computeArtefactID_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("computes correct artefact ID from real path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		libDir := filepath.Join(tmpDir, "lib", "icons")
		require.NoError(t, os.MkdirAll(libDir, 0755))

		filePath := filepath.Join(libDir, "arrow.svg")
		require.NoError(t, os.WriteFile(filePath, []byte("<svg>"), 0644))

		resolver := &resolver_domain.MockResolver{
			GetModuleNameFunc:                      func() string { return "example.com/myproject" },
			ConvertEntryPointPathToManifestKeyFunc: func(p string) string { return p },
		}
		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir: tmpDir,
			},
			resolver: resolver,
		}

		ctx := context.Background()
		_, span, _ := log.Span(ctx, "test")
		defer span.End()

		artefactID, relPath, err := bs.computeArtefactID(ctx, filePath, span)

		require.NoError(t, err)
		assert.Equal(t, "example.com/myproject/lib/icons/arrow.svg", artefactID)
		assert.Equal(t, "lib/icons/arrow.svg", relPath)
	})
}

func TestBuildService_openFileForProcessing_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("opens existing file successfully", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		expectedContent := []byte("test content")
		require.NoError(t, os.WriteFile(filePath, expectedContent, 0644))

		bs := &buildService{}

		ctx := context.Background()
		_, span, _ := log.Span(ctx, "test")
		defer span.End()

		file, err := bs.openFileForProcessing(ctx, filePath, span)

		require.NoError(t, err)
		require.NotNil(t, file)
		defer func() { _ = file.Close() }()

		content := make([]byte, len(expectedContent))
		n, err := file.Read(content)
		require.NoError(t, err)
		assert.Equal(t, len(expectedContent), n)
		assert.Equal(t, expectedContent, content)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		nonExistentPath := filepath.Join(tmpDir, "does-not-exist.txt")

		bs := &buildService{}

		ctx := context.Background()
		_, span, _ := log.Span(ctx, "test")
		defer span.End()

		file, err := bs.openFileForProcessing(ctx, nonExistentPath, span)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})
}
