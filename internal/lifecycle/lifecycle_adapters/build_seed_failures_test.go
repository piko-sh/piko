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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/colour"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

var (
	errDiskFull = errors.New("disk full")
)

type stubThemeRenderer struct {
	render_domain.RenderService
}

func (stubThemeRenderer) BuildThemeCSS(_ context.Context, _ *config.WebsiteConfig) ([]byte, error) {
	return []byte(":root{}"), nil
}

type stubCountingBridge struct{}

func (stubCountingBridge) ArtefactEventsProcessed() int64 { return 0 }

func failingUpsertRegistry(failingSuffix string) *registry_domain.MockRegistryService {
	return &registry_domain.MockRegistryService{
		UpsertArtefactFunc: func(_ context.Context, artefactID, _ string, sourceData io.Reader, _ string, _ []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
			if sourceData != nil {
				_, _ = io.Copy(io.Discard, sourceData)
			}
			if strings.HasSuffix(artefactID, failingSuffix) {
				return nil, errDiskFull
			}
			return &registry_dto.ArtefactMeta{ID: artefactID}, nil
		},
	}
}

func siteResolver() *resolver_domain.MockResolver {
	return &resolver_domain.MockResolver{GetModuleNameFunc: func() string { return "example.com/site" }}
}

func TestSeedFailureLog(t *testing.T) {
	t.Parallel()

	t.Run("take returns the failures in order and empties the log", func(t *testing.T) {
		t.Parallel()

		var failures seedFailureLog
		failures.add(lifecycle_domain.SeedFailure{Path: "a", Stage: seedStageWalk, Error: "one"})
		failures.add(lifecycle_domain.SeedFailure{Path: "b", Stage: seedStageUpsert, Error: "two"})

		taken := failures.take()
		require.Len(t, taken, 2)
		assert.Equal(t, "a", taken[0].Path)
		assert.Equal(t, "b", taken[1].Path)
		assert.Nil(t, failures.take(), "a second take finds nothing")
	})

	t.Run("concurrent adds are all kept", func(t *testing.T) {
		t.Parallel()

		const writers = 8
		const perWriter = 50

		var failures seedFailureLog
		var wg sync.WaitGroup
		for w := range writers {
			wg.Go(func() {
				for i := range perWriter {
					failures.add(lifecycle_domain.SeedFailure{Path: fmt.Sprintf("%d/%d", w, i), Stage: seedStageRead, Error: "x"})
				}
			})
		}
		wg.Wait()

		assert.Len(t, failures.take(), writers*perWriter)
	})
}

func TestBuildService_recordSeedFailure(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	outsidePath := filepath.Join(filepath.Dir(baseDir), "elsewhere", "icon.svg")

	testCases := []struct {
		name      string
		stage     string
		path      string
		err       error
		wantPath  string
		wantError string
		recorded  bool
	}{
		{
			name:     "missing path while walking is not a failure",
			stage:    seedStageWalk,
			path:     filepath.Join(baseDir, "lib"),
			err:      &fs.PathError{Op: "lstat", Path: "lib", Err: fs.ErrNotExist},
			recorded: false,
		},
		{
			name:     "missing file while reading is not a failure",
			stage:    seedStageRead,
			path:     filepath.Join(baseDir, "lib", "4913"),
			err:      fmt.Errorf("opening file: %w", fs.ErrNotExist),
			recorded: false,
		},
		{
			name:      "missing blob while upserting is a failure",
			stage:     seedStageUpsert,
			path:      filepath.Join(baseDir, "lib", "style.css"),
			err:       fmt.Errorf("renaming blob: %w", fs.ErrNotExist),
			recorded:  true,
			wantPath:  "lib/style.css",
			wantError: "renaming blob: file does not exist",
		},
		{
			name:      "unreadable directory while walking is a failure",
			stage:     seedStageWalk,
			path:      filepath.Join(baseDir, "lib", "locked"),
			err:       &fs.PathError{Op: "open", Path: "lib/locked", Err: fs.ErrPermission},
			recorded:  true,
			wantPath:  "lib/locked",
			wantError: "open lib/locked: permission denied",
		},
		{
			name:      "path outside the project keeps its absolute form",
			stage:     seedStageUpsert,
			path:      outsidePath,
			err:       errDiskFull,
			recorded:  true,
			wantPath:  outsidePath,
			wantError: "disk full",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bs := &buildService{pathsConfig: lifecycle_domain.LifecyclePathsConfig{BaseDir: baseDir}}
			bs.recordSeedFailure(context.Background(), tc.stage, tc.path, tc.err)

			failures := bs.seedFailures.take()
			if !tc.recorded {
				assert.Empty(t, failures)
				return
			}
			require.Len(t, failures, 1)
			assert.Equal(t, lifecycle_domain.SeedFailure{Path: tc.wantPath, Stage: tc.stage, Error: tc.wantError}, failures[0])
		})
	}
}

func TestBuildService_walkAndStreamFiles_SeedFailures(t *testing.T) {
	t.Parallel()

	t.Run("a configured directory the project does not have is not a failure", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{pathsConfig: lifecycle_domain.LifecyclePathsConfig{
			BaseDir:             t.TempDir(),
			AssetsSourceDir:     "lib",
			ComponentsSourceDir: "components",
		}}

		fileEvents := make(chan lifecycle_dto.FileEvent, 1)
		var fileCount atomic.Int64
		require.NoError(t, bs.walkAndStreamFiles(context.Background(), fileEvents, &fileCount))
		close(fileEvents)

		assert.Zero(t, fileCount.Load())
		assert.Empty(t, bs.seedFailures.take())
	})

	t.Run("an unreadable directory is recorded and the rest is still walked", func(t *testing.T) {
		t.Parallel()
		if os.Geteuid() == 0 {
			t.Skip("root can read a directory without permissions")
		}

		tmpDir := t.TempDir()
		lockedDir := filepath.Join(tmpDir, "lib", "locked")
		require.NoError(t, os.MkdirAll(lockedDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(lockedDir, "hidden.css"), []byte("a{}"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "lib", "style.css"), []byte("b{}"), 0o600))
		require.NoError(t, os.Chmod(lockedDir, 0))
		t.Cleanup(func() { _ = os.Chmod(lockedDir, 0o755) })

		bs := &buildService{pathsConfig: lifecycle_domain.LifecyclePathsConfig{BaseDir: tmpDir, AssetsSourceDir: "lib"}}

		fileEvents := make(chan lifecycle_dto.FileEvent, 4)
		var fileCount atomic.Int64
		require.NoError(t, bs.walkAndStreamFiles(context.Background(), fileEvents, &fileCount))
		close(fileEvents)

		assert.Equal(t, int64(1), fileCount.Load(), "the readable file is still streamed")
		failures := bs.seedFailures.take()
		require.Len(t, failures, 1)
		assert.Equal(t, "lib/locked", failures[0].Path)
		assert.Equal(t, seedStageWalk, failures[0].Stage)
		assert.Contains(t, failures[0].Error, "permission denied")
	})
}

func TestBuildService_handleFileEvent_SeedFailures(t *testing.T) {
	t.Parallel()

	newProject := func(t *testing.T) (string, string) {
		t.Helper()
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "lib"), 0o755))
		stylePath := filepath.Join(tmpDir, "lib", "style.css")
		require.NoError(t, os.WriteFile(stylePath, []byte("body{}"), 0o600))
		return tmpDir, stylePath
	}

	t.Run("a failed registry upsert is recorded", func(t *testing.T) {
		t.Parallel()

		tmpDir, stylePath := newProject(t)
		bs := &buildService{
			registryService: failingUpsertRegistry("style.css"),
			resolver:        siteResolver(),
			pathsConfig:     lifecycle_domain.LifecyclePathsConfig{BaseDir: tmpDir},
		}

		bs.handleFileEvent(context.Background(), lifecycle_dto.FileEvent{Path: stylePath, Type: lifecycle_dto.FileEventTypeCreate})

		failures := bs.seedFailures.take()
		require.Len(t, failures, 1)
		assert.Equal(t, "lib/style.css", failures[0].Path)
		assert.Equal(t, seedStageUpsert, failures[0].Stage)
		assert.Contains(t, failures[0].Error, "disk full")
	})

	t.Run("a file removed before it is read is not a failure", func(t *testing.T) {
		t.Parallel()

		tmpDir, _ := newProject(t)
		registry := failingUpsertRegistry("never")
		bs := &buildService{
			registryService: registry,
			resolver:        siteResolver(),
			pathsConfig:     lifecycle_domain.LifecyclePathsConfig{BaseDir: tmpDir},
		}

		bs.handleFileEvent(context.Background(), lifecycle_dto.FileEvent{
			Path: filepath.Join(tmpDir, "lib", "4913"),
			Type: lifecycle_dto.FileEventTypeCreate,
		})

		assert.Empty(t, bs.seedFailures.take())
		assert.Zero(t, registry.UpsertArtefactCallCount.Load())
	})

	t.Run("a seeded file records nothing", func(t *testing.T) {
		t.Parallel()

		tmpDir, stylePath := newProject(t)
		bs := &buildService{
			registryService: failingUpsertRegistry("never"),
			resolver:        siteResolver(),
			pathsConfig:     lifecycle_domain.LifecyclePathsConfig{BaseDir: tmpDir},
		}

		bs.handleFileEvent(context.Background(), lifecycle_dto.FileEvent{Path: stylePath, Type: lifecycle_dto.FileEventTypeCreate})

		assert.Empty(t, bs.seedFailures.take())
	})
}

func TestBuildService_seedExternalEntries_SeedFailures(t *testing.T) {
	t.Parallel()

	t.Run("a component directory the walk cannot read is recorded", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{registryService: &registry_domain.MockRegistryService{}}
		walkErr := &fs.PathError{Op: "open", Path: "/mod/components", Err: fs.ErrPermission}

		require.NoError(t, bs.seedExternalComponentEntry(context.Background(), "/mod/components", "example.com/mod/components",
			"/mod/components/private", nil, walkErr))

		failures := bs.seedFailures.take()
		require.Len(t, failures, 1)
		assert.Equal(t, "/mod/components/private", failures[0].Path)
		assert.Equal(t, seedStageWalk, failures[0].Stage)
	})

	t.Run("an asset directory the module does not have is not a failure", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{registryService: &registry_domain.MockRegistryService{}}
		walkErr := &fs.PathError{Op: "lstat", Path: "/mod/lib/icons", Err: fs.ErrNotExist}

		require.NoError(t, bs.seedExternalAssetEntry(context.Background(), "/mod/lib/icons", "example.com/mod/lib/icons",
			"/mod/lib/icons", nil, walkErr))

		assert.Empty(t, bs.seedFailures.take())
	})

	t.Run("a failed external asset upsert is recorded", func(t *testing.T) {
		t.Parallel()

		assetDir := t.TempDir()
		iconPath := filepath.Join(assetDir, "arrow.svg")
		require.NoError(t, os.WriteFile(iconPath, []byte("<svg/>"), 0o600))
		entries, err := os.ReadDir(assetDir)
		require.NoError(t, err)

		bs := &buildService{registryService: failingUpsertRegistry("arrow.svg")}

		require.NoError(t, bs.seedExternalAssetEntry(context.Background(), assetDir, "example.com/mod/lib/icons", iconPath, entries[0], nil))

		failures := bs.seedFailures.take()
		require.Len(t, failures, 1)
		assert.Equal(t, iconPath, failures[0].Path)
		assert.Equal(t, seedStageUpsert, failures[0].Stage)
		assert.Contains(t, failures[0].Error, "example.com/mod/lib/icons/arrow.svg")
		assert.Contains(t, failures[0].Error, "disk full")
	})

	t.Run("a failed external component upsert is recorded", func(t *testing.T) {
		t.Parallel()

		componentDir := t.TempDir()
		cardPath := filepath.Join(componentDir, "card.pkc")
		require.NoError(t, os.WriteFile(cardPath, []byte("<template></template>"), 0o600))
		entries, err := os.ReadDir(componentDir)
		require.NoError(t, err)

		bs := &buildService{registryService: failingUpsertRegistry("card.pkc")}

		require.NoError(t, bs.seedExternalComponentEntry(context.Background(), componentDir, "example.com/mod/components", cardPath, entries[0], nil))

		failures := bs.seedFailures.take()
		require.Len(t, failures, 1)
		assert.Equal(t, seedStageUpsert, failures[0].Stage)
		assert.Contains(t, failures[0].Error, "disk full")
	})
}

func TestBuildService_buildResult_TakesSeedFailures(t *testing.T) {
	t.Parallel()

	bs := &buildService{
		orchestratorService: &stubOrchestratorService{dispatcher: &stubTaskDispatcher{}},
	}
	bs.seedFailures.add(lifecycle_domain.SeedFailure{Path: "lib/style.css", Stage: seedStageUpsert, Error: "disk full"})

	result := bs.buildResult(context.Background(), time.Now(), orchestrator_domain.DispatcherStats{})

	require.Len(t, result.SeedFailures, 1)
	assert.Zero(t, result.TotalFailed)
	assert.True(t, result.HasFailures(), "a seed failure fails the build even when every task succeeded")
	assert.Empty(t, bs.seedFailures.take(), "the failures are handed to the result once")
}

func TestBuildService_RunBuild_SeedFailureFailsTheBuild(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "lib")
	require.NoError(t, os.MkdirAll(libDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "ok.css"), []byte("a{}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "broken.css"), []byte("b{}"), 0o600))

	registry := failingUpsertRegistry("broken.css")
	bs := &buildService{
		registryService:     registry,
		orchestratorService: &stubOrchestratorService{dispatcher: &stubTaskDispatcher{}},
		bridge:              stubCountingBridge{},
		renderer:            stubThemeRenderer{},
		resolver:            siteResolver(),
		websiteConfig:       &config.WebsiteConfig{},
		pathsConfig:         lifecycle_domain.LifecyclePathsConfig{BaseDir: tmpDir, AssetsSourceDir: "lib"},
	}

	result, err := bs.RunBuild(context.Background())

	require.NoError(t, err, "a seed failure is reported in the result, like a failed task")
	require.NotNil(t, result)
	assert.Zero(t, result.TotalFailed)
	assert.True(t, result.HasFailures())
	require.Len(t, result.SeedFailures, 1)
	assert.Equal(t, "lib/broken.css", result.SeedFailures[0].Path)
	assert.Equal(t, seedStageUpsert, result.SeedFailures[0].Stage)
	assert.Equal(t, int64(3), registry.UpsertArtefactCallCount.Load(), "theme, ok.css and broken.css are all upserted")
}

func TestFormatBuildSummary_SeedFailures(t *testing.T) {
	previous := colour.Enabled()
	colour.SetEnabled(false)
	t.Cleanup(func() { colour.SetEnabled(previous) })

	result := &lifecycle_domain.BuildResult{
		TotalDispatched: 2,
		TotalCompleted:  2,
		SeedFailures: []lifecycle_domain.SeedFailure{
			{Path: "lib/broken.css", Stage: seedStageUpsert, Error: "upserting artefact: disk full"},
			{Path: "lib/locked", Stage: seedStageWalk, Error: "open lib/locked: permission denied"},
		},
	}

	t.Run("build summary", func(t *testing.T) {
		output := FormatBuildSummary(result)

		assert.Contains(t, output, "✗ 2 source files could not be seeded")
		assert.Contains(t, output, "seeding failed during upsert")
		assert.Contains(t, output, "lib/broken.css")
		assert.Contains(t, output, "upserting artefact: disk full")
		assert.Contains(t, output, "seeding failed during walk")
		assert.Contains(t, output, "lib/locked")
	})

	t.Run("combined summary", func(t *testing.T) {
		output := FormatCombinedSummary(&GeneratorResult{}, result, time.Second, 2*time.Second)

		assert.Contains(t, output, "✗ 2 source files could not be seeded")
		assert.Contains(t, output, "lib/broken.css")
		assert.Contains(t, output, "permission denied")
	})

	t.Run("no seed failures adds no line", func(t *testing.T) {
		output := FormatBuildSummary(&lifecycle_domain.BuildResult{TotalDispatched: 1, TotalCompleted: 1})

		assert.NotContains(t, output, "could not be seeded")
	})
}
