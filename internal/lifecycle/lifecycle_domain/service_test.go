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

package lifecycle_domain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/clock"
)

func mustBuildLifecycleService(t *testing.T, deps *LifecycleServiceDeps) *lifecycleService {
	t.Helper()
	service := NewLifecycleService(deps)
	ls, ok := service.(*lifecycleService)
	require.True(t, ok, "expected *lifecycleService")
	return ls
}

func TestNewLifecycleService(t *testing.T) {
	t.Parallel()

	t.Run("creates service with all dependencies", func(t *testing.T) {
		t.Parallel()

		service := newLifecycleTestBuilder().
			WithMockFileSystem().
			WithMockWatcher().
			WithMockClock().
			Build()

		require.NotNil(t, service)
	})

	t.Run("defaults to OS filesystem when nil", func(t *testing.T) {
		t.Parallel()

		deps := &LifecycleServiceDeps{
			PathsConfig: LifecyclePathsConfig{
				BaseDir: "/test",
			},
			FileSystem: nil,
		}

		service := NewLifecycleService(deps)
		require.NotNil(t, service)
	})

	t.Run("defaults to real clock when nil", func(t *testing.T) {
		t.Parallel()

		deps := &LifecycleServiceDeps{
			PathsConfig: LifecyclePathsConfig{
				BaseDir: "/test",
			},
			Clock: nil,
		}

		service := NewLifecycleService(deps)
		require.NotNil(t, service)
	})
}

func TestLifecycleTestBuilder(t *testing.T) {
	t.Parallel()

	t.Run("builder with all options", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockWatcher := &MockFileSystemWatcher{}

		builder := newLifecycleTestBuilder().
			WithFileSystem(mockFS).
			WithWatcher(mockWatcher).
			WithMockClock().
			WithBaseDir("/custom/base")

		service := builder.Build()
		require.NotNil(t, service)

		mockClock := builder.GetMockClock()
		require.NotNil(t, mockClock)
	})

	t.Run("builder with mock clock at specific time", func(t *testing.T) {
		t.Parallel()

		fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		builder := newLifecycleTestBuilder().WithMockClockAt(fixedTime)

		mockClock := builder.GetMockClock()
		require.NotNil(t, mockClock)
		assert.Equal(t, fixedTime, mockClock.Now())
	})

	t.Run("GetDeps returns modifiable deps", func(t *testing.T) {
		t.Parallel()

		builder := newLifecycleTestBuilder()
		deps := builder.GetDeps()

		deps.PathsConfig.BaseDir = "/modified"

		service := builder.Build()
		require.NotNil(t, service)
	})

	t.Run("builder with mock file system helper", func(t *testing.T) {
		t.Parallel()

		service := newLifecycleTestBuilder().
			WithMockFileSystem().
			Build()

		require.NotNil(t, service)
	})

	t.Run("builder with mock watcher helper", func(t *testing.T) {
		t.Parallel()

		service := newLifecycleTestBuilder().
			WithMockWatcher().
			Build()

		require.NotNil(t, service)
	})

	t.Run("builder with website config", func(t *testing.T) {
		t.Parallel()

		service := newLifecycleTestBuilder().
			WithWebsiteConfig(config.WebsiteConfig{}).
			Build()

		require.NotNil(t, service)
	})

	t.Run("builder with asset pipeline", func(t *testing.T) {
		t.Parallel()

		mockPipeline := &daemon_domain.MockAssetPipeline{}
		builder := newLifecycleTestBuilder().WithAssetPipeline(mockPipeline)

		deps := builder.GetDeps()
		assert.Equal(t, mockPipeline, deps.AssetPipeline)
	})

	t.Run("builder with build cache invalidator", func(t *testing.T) {
		t.Parallel()

		mockInvalidator := &daemon_domain.MockBuildCacheInvalidator{}
		builder := newLifecycleTestBuilder().WithBuildCacheInvalidator(mockInvalidator)

		deps := builder.GetDeps()
		assert.Equal(t, mockInvalidator, deps.BuildCacheInvalidator)
	})

	t.Run("builder with interpreted orchestrator", func(t *testing.T) {
		t.Parallel()

		mockOrchestrator := &mockInterpretedBuildOrchestrator{}
		builder := newLifecycleTestBuilder().WithInterpretedOrchestrator(mockOrchestrator)

		deps := builder.GetDeps()
		assert.Equal(t, mockOrchestrator, deps.InterpretedOrchestrator)
	})

	t.Run("builder with templater service", func(t *testing.T) {
		t.Parallel()

		mockTemplater := &mockTemplaterRunnerSwapper{}
		builder := newLifecycleTestBuilder().WithTemplaterService(mockTemplater)

		deps := builder.GetDeps()
		assert.Equal(t, mockTemplater, deps.TemplaterService)
	})

	t.Run("builder with router manager", func(t *testing.T) {
		t.Parallel()

		mockRouter := &mockRouterReloadNotifier{}
		builder := newLifecycleTestBuilder().WithRouterManager(mockRouter)

		deps := builder.GetDeps()
		assert.Equal(t, mockRouter, deps.RouterManager)
	})

	t.Run("builder with clock", func(t *testing.T) {
		t.Parallel()

		mockClk := clock.NewMockClock(time.Now())
		builder := newLifecycleTestBuilder().WithClock(mockClk)

		deps := builder.GetDeps()
		assert.Equal(t, mockClk, deps.Clock)
	})

	t.Run("GetMockClock returns nil when not set", func(t *testing.T) {
		t.Parallel()

		builder := newLifecycleTestBuilder()
		assert.Nil(t, builder.GetMockClock())
	})
}

func TestLifecycleService_GetEntryPoints(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no entry points", func(t *testing.T) {
		t.Parallel()

		service := newLifecycleTestBuilder().BuildInternal(t)

		result := service.GetEntryPoints()
		assert.Empty(t, result)
	})

	t.Run("returns copy of entry points", func(t *testing.T) {
		t.Parallel()

		service := newLifecycleTestBuilder().BuildInternal(t)

		service.addEntryPointIfNotExists(context.Background(), "pages/home.pk", componentType{isPage: true})
		service.addEntryPointIfNotExists(context.Background(), "pages/about.pk", componentType{isPage: true})

		result := service.GetEntryPoints()
		assert.Len(t, result, 2)

		result[0].Path = "modified"
		original := service.GetEntryPoints()
		assert.NotEqual(t, "modified", original[0].Path)
	})
}

func TestLifecycleService_RequestRebuild(t *testing.T) {
	t.Parallel()

	t.Run("no-op when coordinator is nil", func(t *testing.T) {
		t.Parallel()

		service := newLifecycleTestBuilder().BuildInternal(t)
		service.coordinatorService = nil

		service.RequestRebuild(context.Background(), "test-causation")
	})

	t.Run("calls coordinator with entry points", func(t *testing.T) {
		t.Parallel()

		mockCoordinator := &mockTrackingCoordinatorService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.CoordinatorService = mockCoordinator
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		service.addEntryPointIfNotExists(context.Background(), "pages/home.pk", componentType{isPage: true})

		service.RequestRebuild(context.Background(), "test-causation")

		assert.True(t, mockCoordinator.rebuildCalled)
	})
}

func TestLifecycleService_Stop(t *testing.T) {
	t.Parallel()

	t.Run("stops cleanly with mock watcher", func(t *testing.T) {
		t.Parallel()

		mockWatcher := &MockFileSystemWatcher{}
		service := newLifecycleTestBuilder().
			WithWatcher(mockWatcher).
			BuildInternal(t)

		service.stopChan = make(chan struct{})

		err := service.Stop(context.Background())
		assert.NoError(t, err)
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		t.Parallel()

		mockWatcher := &MockFileSystemWatcher{}
		service := newLifecycleTestBuilder().
			WithWatcher(mockWatcher).
			BuildInternal(t)

		service.stopChan = make(chan struct{})

		err := service.Stop(context.Background())
		assert.NoError(t, err)

		err = service.Stop(context.Background())
		assert.NoError(t, err)
	})

	t.Run("stops cleanly without watcher", func(t *testing.T) {
		t.Parallel()

		service := newLifecycleTestBuilder().BuildInternal(t)

		service.stopChan = make(chan struct{})
		service.watcherAdapter = nil

		err := service.Stop(context.Background())
		assert.NoError(t, err)
	})
}

func TestLifecycleService_getAssetSourceDirs(t *testing.T) {
	t.Parallel()

	t.Run("returns all configured asset dirs", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.PathsConfig.AssetsSourceDir = "assets"
		deps.PathsConfig.PagesSourceDir = "pages"
		deps.PathsConfig.ComponentsSourceDir = "components"
		deps.PathsConfig.PartialsSourceDir = "partials"
		deps.PathsConfig.I18nSourceDir = "i18n"

		service := mustBuildLifecycleService(t, deps)

		dirs := service.getAssetSourceDirs()

		assert.Len(t, dirs, 5)
	})

	t.Run("skips empty directory configs", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.PathsConfig.AssetsSourceDir = "assets"
		deps.PathsConfig.PagesSourceDir = ""
		deps.PathsConfig.ComponentsSourceDir = ""

		service := mustBuildLifecycleService(t, deps)

		dirs := service.getAssetSourceDirs()

		assert.Len(t, dirs, 1)
		assert.Contains(t, dirs[0], "assets")
	})
}

func TestLifecycleService_getStaticWatchDirs(t *testing.T) {
	t.Parallel()

	t.Run("returns nil without resolver", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = nil

		service := mustBuildLifecycleService(t, deps)

		dirs := service.getStaticWatchDirs()
		assert.Nil(t, dirs)
	})

	t.Run("returns dirs with resolver", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.PagesSourceDir = "pages"
		deps.PathsConfig.PartialsSourceDir = "partials"
		deps.PathsConfig.ComponentsSourceDir = ""
		deps.FileSystem = NewMockFileSystem()

		service := mustBuildLifecycleService(t, deps)

		dirs := service.getStaticWatchDirs()
		assert.NotNil(t, dirs)

		assert.GreaterOrEqual(t, len(dirs), 1)
	})
}

func TestLifecycleService_determineComponentType(t *testing.T) {
	t.Parallel()

	t.Run("identifies page component", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		ct := service.determineComponentType("pages/home.pk")

		assert.True(t, ct.isPage)
		assert.False(t, ct.isPartial)
		assert.False(t, ct.isEmail)
	})

	t.Run("identifies partial component", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.PathsConfig.PartialsSourceDir = "partials"

		service := mustBuildLifecycleService(t, deps)

		ct := service.determineComponentType("partials/header.pk")

		assert.False(t, ct.isPage)
		assert.True(t, ct.isPartial)
		assert.False(t, ct.isEmail)
	})

	t.Run("identifies email component", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.PathsConfig.EmailsSourceDir = "emails"

		service := mustBuildLifecycleService(t, deps)

		ct := service.determineComponentType("emails/welcome.pk")

		assert.False(t, ct.isPage)
		assert.False(t, ct.isPartial)
		assert.True(t, ct.isEmail)
	})

	t.Run("returns false for unknown component", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		ct := service.determineComponentType("other/file.pk")

		assert.False(t, ct.isPage)
		assert.False(t, ct.isPartial)
		assert.False(t, ct.isEmail)
	})
}

func TestLifecycleService_addEntryPointIfNotExists(t *testing.T) {
	t.Parallel()

	t.Run("adds new entry point", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		service := mustBuildLifecycleService(t, deps)

		service.addEntryPointIfNotExists(context.Background(), "module/pages/home.pk", componentType{isPage: true})

		assert.Len(t, service.entryPoints, 1)
		assert.Equal(t, "module/pages/home.pk", service.entryPoints[0].Path)
		assert.True(t, service.entryPoints[0].IsPage)
	})

	t.Run("does not add duplicate entry point", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		service := mustBuildLifecycleService(t, deps)

		service.addEntryPointIfNotExists(context.Background(), "module/pages/home.pk", componentType{isPage: true})
		service.addEntryPointIfNotExists(context.Background(), "module/pages/home.pk", componentType{isPage: true})

		assert.Len(t, service.entryPoints, 1)
	})
}

func TestLifecycleService_removeEntryPoint(t *testing.T) {
	t.Parallel()

	t.Run("removes existing entry point", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		service := mustBuildLifecycleService(t, deps)

		service.addEntryPointIfNotExists(context.Background(), "module/pages/home.pk", componentType{isPage: true})
		service.addEntryPointIfNotExists(context.Background(), "module/pages/about.pk", componentType{isPage: true})

		service.removeEntryPoint("module/pages/home.pk")

		assert.Len(t, service.entryPoints, 1)
		assert.Equal(t, "module/pages/about.pk", service.entryPoints[0].Path)
	})

	t.Run("does nothing for non-existent entry point", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		service := mustBuildLifecycleService(t, deps)

		service.addEntryPointIfNotExists(context.Background(), "module/pages/home.pk", componentType{isPage: true})
		service.removeEntryPoint("module/pages/nonexistent.pk")

		assert.Len(t, service.entryPoints, 1)
	})
}

func TestLifecycleService_clearSvgCacheIfNeeded(t *testing.T) {
	t.Parallel()

	t.Run("clears cache for SVG files", func(t *testing.T) {
		t.Parallel()

		mockRenderRegistry := &mockRenderRegistryPort{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RenderRegistryPort = mockRenderRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			event:      lifecycle_dto.FileEvent{Path: "/test/icons/logo.svg"},
			artefactID: "icons/logo.svg",
		}

		service.clearSvgCacheIfNeeded(fec)

		assert.Contains(t, mockRenderRegistry.clearedSvgIDs, "icons/logo.svg")
	})

	t.Run("does not clear cache for non-SVG files", func(t *testing.T) {
		t.Parallel()

		mockRenderRegistry := &mockRenderRegistryPort{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RenderRegistryPort = mockRenderRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			event:      lifecycle_dto.FileEvent{Path: "/test/images/logo.png"},
			artefactID: "images/logo.png",
		}

		service.clearSvgCacheIfNeeded(fec)

		assert.Empty(t, mockRenderRegistry.clearedSvgIDs)
	})

	t.Run("no-op when render registry nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RenderRegistryPort = nil

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			event:      lifecycle_dto.FileEvent{Path: "/test/icons/logo.svg"},
			artefactID: "icons/logo.svg",
		}

		service.clearSvgCacheIfNeeded(fec)
	})
}

func TestInterpretedManifestStoreViewAdapter(t *testing.T) {
	t.Parallel()

	t.Run("delegates GetKeys to runner", func(t *testing.T) {
		t.Parallel()

		runner := &mockInterpretedRunner{
			keys: []string{"page1", "page2"},
		}

		adapter := &interpretedManifestStoreViewAdapter{runner: runner}

		keys := adapter.GetKeys()
		assert.Equal(t, []string{"page1", "page2"}, keys)
	})

	t.Run("delegates GetPageEntry to runner", func(t *testing.T) {
		t.Parallel()

		mockEntry := &templater_domain.MockPageEntryView{}
		runner := &mockInterpretedRunner{
			entries: map[string]templater_domain.PageEntryView{
				"page1": mockEntry,
			},
		}

		adapter := &interpretedManifestStoreViewAdapter{runner: runner}

		entry, found := adapter.GetPageEntry("page1")
		assert.True(t, found)
		assert.Equal(t, mockEntry, entry)
	})

	t.Run("returns false for missing entry", func(t *testing.T) {
		t.Parallel()

		runner := &mockInterpretedRunner{
			entries: map[string]templater_domain.PageEntryView{},
		}

		adapter := &interpretedManifestStoreViewAdapter{runner: runner}

		_, found := adapter.GetPageEntry("missing")
		assert.False(t, found)
	})
}

func TestNewInterpretedManifestStoreView(t *testing.T) {
	t.Parallel()

	t.Run("creates adapter for interpreted runner", func(t *testing.T) {
		t.Parallel()

		runner := &mockInterpretedRunner{
			keys: []string{"page1"},
		}

		view := newInterpretedManifestStoreView(runner)
		require.NotNil(t, view)

		keys := view.GetKeys()
		assert.Equal(t, []string{"page1"}, keys)
	})

	t.Run("panics for non-interpreted runner", func(t *testing.T) {
		t.Parallel()

		nonInterpretedRunner := &mockNonInterpretedRunner{}

		assert.Panics(t, func() {
			newInterpretedManifestStoreView(nonInterpretedRunner)
		})
	})
}

type mockRenderRegistryPort struct {
	clearedSvgIDs []string
}

func (m *mockRenderRegistryPort) ClearSvgCache(_ context.Context, svgID string) {
	m.clearedSvgIDs = append(m.clearedSvgIDs, svgID)
}

func (m *mockRenderRegistryPort) ClearComponentCache(_ context.Context, _ string) {}
func (m *mockRenderRegistryPort) GetComponentMetadata(_ context.Context, _ string) (*render_dto.ComponentMetadata, error) {
	return nil, nil
}
func (m *mockRenderRegistryPort) GetAssetRawSVG(_ context.Context, _ string) (*render_domain.ParsedSvgData, error) {
	return nil, nil
}
func (m *mockRenderRegistryPort) BulkGetAssetRawSVG(_ context.Context, _ []string) (map[string]*render_domain.ParsedSvgData, error) {
	return nil, nil
}
func (m *mockRenderRegistryPort) BulkGetComponentMetadata(_ context.Context, _ []string) (map[string]*render_dto.ComponentMetadata, error) {
	return nil, nil
}
func (m *mockRenderRegistryPort) GetStats() render_domain.RegistryAdapterStats {
	return render_domain.RegistryAdapterStats{}
}
func (m *mockRenderRegistryPort) GetArtefactServePath(_ context.Context, _ string) string {
	return ""
}
func (m *mockRenderRegistryPort) UpsertArtefact(_ context.Context, _ string, _ string, _ io.Reader, _ string, _ []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

type mockInterpretedRunner struct {
	entries map[string]templater_domain.PageEntryView
	keys    []string
}

func (m *mockInterpretedRunner) GetKeys() []string {
	return m.keys
}

func (m *mockInterpretedRunner) GetPageEntryByPath(path string) (templater_domain.PageEntryView, bool) {
	entry, ok := m.entries[path]
	return entry, ok
}

func (m *mockInterpretedRunner) RunPage(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockInterpretedRunner) RunPartial(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockInterpretedRunner) RunPartialWithProps(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockInterpretedRunner) GetPageEntry(_ context.Context, _ string) (templater_domain.PageEntryView, error) {
	return nil, nil
}

type mockInterpretedBuildOrchestrator struct{}

func (m *mockInterpretedBuildOrchestrator) BuildRunner(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) (templater_domain.ManifestRunnerPort, error) {
	return nil, nil
}

func (m *mockInterpretedBuildOrchestrator) MarkDirty(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) error {
	return nil
}

func (m *mockInterpretedBuildOrchestrator) MarkComponentsDirty(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) error {
	return nil
}

func (m *mockInterpretedBuildOrchestrator) IsInitialised() bool { return false }

func (m *mockInterpretedBuildOrchestrator) GetAffectedComponents(_ string) []string { return nil }

func (m *mockInterpretedBuildOrchestrator) ProactiveRecompile(_ context.Context) error { return nil }

type mockTemplaterRunnerSwapper struct{}

func (m *mockTemplaterRunnerSwapper) SetRunner(_ templater_domain.ManifestRunnerPort) {}

type mockRouterReloadNotifier struct{}

func (*mockRouterReloadNotifier) ReloadRoutes(_ context.Context, _ templater_domain.ManifestStoreView) error {
	return nil
}

type mockNonInterpretedRunner struct{}

func (m *mockNonInterpretedRunner) RunPage(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockNonInterpretedRunner) RunPartial(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockNonInterpretedRunner) RunPartialWithProps(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return nil, templater_dto.InternalMetadata{}, "", nil
}

func (m *mockNonInterpretedRunner) GetPageEntry(_ context.Context, _ string) (templater_domain.PageEntryView, error) {
	return nil, nil
}

func TestLifecycleService_discoverAssetFiles(t *testing.T) {
	t.Parallel()

	t.Run("discovers files in asset directories", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddFile("/project/assets/logo.png", []byte("png data"))
		mockFS.AddFile("/project/assets/style.css", []byte("css data"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		files := service.discoverAssetFiles(context.Background(), []string{"/project/assets"})

		assert.GreaterOrEqual(t, len(files), 2)
	})

	t.Run("returns empty for non-existent directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS

		service := mustBuildLifecycleService(t, deps)

		files := service.discoverAssetFiles(context.Background(), []string{"/nonexistent"})

		assert.Empty(t, files)
	})

	t.Run("returns empty for empty dirs list", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		service := mustBuildLifecycleService(t, deps)

		files := service.discoverAssetFiles(context.Background(), []string{})

		assert.Empty(t, files)
	})

	t.Run("ctx cancellation does not deadlock walkers", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		for i := range fileEventChannelBuffer * 4 {
			mockFS.AddFile(fmt.Sprintf("/project/assets/file-%05d.css", i), []byte("body{}"))
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(errors.New("test triggered cancellation before discovery completes"))

		done := make(chan struct{})
		go func() {
			defer close(done)
			service.discoverAssetFiles(ctx, []string{"/project/assets"})
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("discoverAssetFiles deadlocked when context was cancelled before consumer started")
		}
	})
}

func TestLifecycleService_buildFileEventContext(t *testing.T) {
	t.Parallel()

	t.Run("builds context for valid file", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/assets/logo.png",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		fec, ok := service.buildFileEventContext(context.Background(), event)

		assert.True(t, ok)
		assert.Equal(t, "assets/logo.png", fec.relPath)
		assert.Contains(t, fec.artefactID, "assets/logo.png")
	})

	t.Run("returns false for irrelevant file", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/node_modules/package.json",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		_, ok := service.buildFileEventContext(context.Background(), event)

		assert.False(t, ok)
	})
}

func TestLifecycleService_discoverInitialEntryPoints(t *testing.T) {
	t.Parallel()

	t.Run("returns error without resolver", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = nil

		service := mustBuildLifecycleService(t, deps)

		_, err := service.discoverInitialEntryPoints(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no resolver provided")
	})

	t.Run("discovers pk files in pages directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/pages")
		mockFS.AddFile("/project/pages/home.pk", []byte("home page"))
		mockFS.AddFile("/project/pages/about.pk", []byte("about page"))
		mockFS.AddFile("/project/pages/nested/deep.pk", []byte("nested page"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		entryPoints, err := service.discoverInitialEntryPoints(context.Background())

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entryPoints), 3)

		for _, ep := range entryPoints {
			assert.True(t, ep.IsPage)
		}
	})

	t.Run("skips files starting with underscore", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/pages")
		mockFS.AddFile("/project/pages/home.pk", []byte("home page"))
		mockFS.AddFile("/project/pages/_partial.pk", []byte("partial"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		entryPoints, err := service.discoverInitialEntryPoints(context.Background())

		require.NoError(t, err)

		assert.Len(t, entryPoints, 1)
	})

	t.Run("discovers partials when directory exists", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/pages")
		mockFS.AddFile("/project/pages/home.pk", []byte("home page"))
		mockFS.AddDir("/project/partials")
		mockFS.AddFile("/project/partials/header.pk", []byte("header"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.PagesSourceDir = "pages"
		deps.PathsConfig.PartialsSourceDir = "partials"

		service := mustBuildLifecycleService(t, deps)

		entryPoints, err := service.discoverInitialEntryPoints(context.Background())

		require.NoError(t, err)
		assert.Len(t, entryPoints, 2)

		hasPage := false
		hasPartial := false
		for _, ep := range entryPoints {
			if ep.IsPage {
				hasPage = true
			} else {
				hasPartial = true
			}
		}
		assert.True(t, hasPage)
		assert.True(t, hasPartial)
	})
}

func TestLifecycleService_discoverEntryPointsInDir(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for empty directory path", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		service := mustBuildLifecycleService(t, deps)

		entryPoints, err := service.discoverEntryPointsInDir(context.Background(), "", entryPointDiscoveryConfig{})

		assert.NoError(t, err)
		assert.Nil(t, entryPoints)
	})

	t.Run("discovers files in directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/pages")
		mockFS.AddFile("/project/pages/home.pk", []byte("home"))
		mockFS.AddFile("/project/pages/about.pk", []byte("about"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		discoveryConfig := entryPointDiscoveryConfig{
			baseDir:    "/project",
			moduleName: "test-module",
			isPage:     true,
			isPublic:   true,
		}

		entryPoints, err := service.discoverEntryPointsInDir(context.Background(), "pages", discoveryConfig)

		require.NoError(t, err)
		assert.Len(t, entryPoints, 2)
	})
}

func TestLifecycleService_tryCreateEntryPoint(t *testing.T) {
	t.Parallel()

	t.Run("creates entry point for pk file", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		discoveryConfig := entryPointDiscoveryConfig{
			baseDir:    "/project",
			moduleName: "test-module",
			isPage:     true,
			isPublic:   true,
		}

		entry := &mockDirEntry{name: "home.pk", isDir: false}
		ep := service.tryCreateEntryPoint("/project/pages/home.pk", entry, discoveryConfig)

		require.NotNil(t, ep)
		assert.True(t, ep.IsPage)
		assert.True(t, ep.IsPublic)
		assert.Contains(t, ep.Path, "home.pk")
	})

	t.Run("returns nil for directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS

		service := mustBuildLifecycleService(t, deps)

		discoveryConfig := entryPointDiscoveryConfig{
			baseDir:    "/project",
			moduleName: "test-module",
			isPage:     true,
		}

		entry := &mockDirEntry{name: "subdir", isDir: true}
		ep := service.tryCreateEntryPoint("/project/pages/subdir", entry, discoveryConfig)

		assert.Nil(t, ep)
	})

	t.Run("returns nil for non-pk file", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS

		service := mustBuildLifecycleService(t, deps)

		discoveryConfig := entryPointDiscoveryConfig{
			baseDir:    "/project",
			moduleName: "test-module",
			isPage:     true,
		}

		entry := &mockDirEntry{name: "readme.md", isDir: false}
		ep := service.tryCreateEntryPoint("/project/pages/readme.md", entry, discoveryConfig)

		assert.Nil(t, ep)
	})

	t.Run("returns nil for underscore prefixed file", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS

		service := mustBuildLifecycleService(t, deps)

		discoveryConfig := entryPointDiscoveryConfig{
			baseDir:    "/project",
			moduleName: "test-module",
			isPage:     true,
		}

		entry := &mockDirEntry{name: "_partial.pk", isDir: false}
		ep := service.tryCreateEntryPoint("/project/pages/_partial.pk", entry, discoveryConfig)

		assert.Nil(t, ep)
	})
}

func TestLifecycleService_updateBuildContext(t *testing.T) {
	t.Parallel()

	t.Run("adds entry point on create event", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/pages/new.pk",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		service.updateBuildContext(context.Background(), event, "pages/new.pk")

		assert.Len(t, service.entryPoints, 1)
	})

	t.Run("removes entry point on remove event", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/pages/old.pk",
			Type: lifecycle_dto.FileEventTypeCreate,
		}
		service.updateBuildContext(context.Background(), event, "pages/old.pk")
		assert.Len(t, service.entryPoints, 1)

		event.Type = lifecycle_dto.FileEventTypeRemove
		service.updateBuildContext(context.Background(), event, "pages/old.pk")

		assert.Empty(t, service.entryPoints)
	})

	t.Run("no-op without resolver", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = nil

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/pages/new.pk",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		service.updateBuildContext(context.Background(), event, "pages/new.pk")

		assert.Empty(t, service.entryPoints)
	})

	t.Run("no-op for non-pk file", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/pages/style.css",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		service.updateBuildContext(context.Background(), event, "pages/style.css")

		assert.Empty(t, service.entryPoints)
	})
}

func TestLifecycleService_handleCoreSourceChange(t *testing.T) {
	t.Parallel()

	t.Run("invalidates cache when invalidator present", func(t *testing.T) {
		t.Parallel()

		mockInvalidator := &mockTrackingBuildCacheInvalidator{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.BuildCacheInvalidator = mockInvalidator
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "pages/home.pk",
			artefactID: "test-module/pages/home.pk",
			event: lifecycle_dto.FileEvent{
				Path: "/project/pages/home.pk",
				Type: lifecycle_dto.FileEventTypeWrite,
			},
		}

		service.handleCoreSourceChange(fec, false)

		assert.True(t, mockInvalidator.invalidated)
	})

	t.Run("skips invalidation on initial seed", func(t *testing.T) {
		t.Parallel()

		mockInvalidator := &mockTrackingBuildCacheInvalidator{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.BuildCacheInvalidator = mockInvalidator
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:     context.Background(),
			relPath: "pages/home.pk",
			event: lifecycle_dto.FileEvent{
				Path: "/project/pages/home.pk",
				Type: lifecycle_dto.FileEventTypeCreate,
			},
		}

		service.handleCoreSourceChange(fec, true)

		assert.False(t, mockInvalidator.invalidated)
	})
}

type mockTrackingBuildCacheInvalidator struct {
	invalidated bool
}

func (m *mockTrackingBuildCacheInvalidator) InvalidateBuildCache() {
	m.invalidated = true
}

func TestLifecycleService_handleFileEvent(t *testing.T) {
	t.Parallel()

	t.Run("routes core source to handleCoreSourceChange", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockInvalidator := &mockTrackingBuildCacheInvalidator{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.BuildCacheInvalidator = mockInvalidator
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/pages/home.pk",
			Type: lifecycle_dto.FileEventTypeWrite,
		}

		service.handleFileEvent(context.Background(), event, false)

		assert.True(t, mockInvalidator.invalidated)
	})

	t.Run("routes asset files to handleAssetChange", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/project/assets/logo.png", []byte("png data"))

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/assets/logo.png",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		service.handleFileEvent(context.Background(), event, false)

		assert.Len(t, mockRegistry.upsertedArtefacts, 1)
	})

	t.Run("ignores irrelevant files", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/node_modules/package.json",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		service.handleFileEvent(context.Background(), event, false)

		assert.Empty(t, mockRegistry.upsertedArtefacts)
	})
}

func TestLifecycleService_handleAssetChange(t *testing.T) {
	t.Parallel()

	t.Run("upserts on create event", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/project/assets/logo.png", []byte("png data"))

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "assets/logo.png",
			artefactID: "test-module/assets/logo.png",
			event: lifecycle_dto.FileEvent{
				Path: "/project/assets/logo.png",
				Type: lifecycle_dto.FileEventTypeCreate,
			},
		}

		service.handleAssetChange(fec)

		assert.Len(t, mockRegistry.upsertedArtefacts, 1)
		assert.Equal(t, "test-module/assets/logo.png", mockRegistry.upsertedArtefacts[0])
	})

	t.Run("upserts on write event", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/project/assets/style.css", []byte("css data"))

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "assets/style.css",
			artefactID: "test-module/assets/style.css",
			event: lifecycle_dto.FileEvent{
				Path: "/project/assets/style.css",
				Type: lifecycle_dto.FileEventTypeWrite,
			},
		}

		service.handleAssetChange(fec)

		assert.Len(t, mockRegistry.upsertedArtefacts, 1)
	})

	t.Run("deletes on remove event", func(t *testing.T) {
		t.Parallel()

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "assets/old.png",
			artefactID: "test-module/assets/old.png",
			event: lifecycle_dto.FileEvent{
				Path: "/project/assets/old.png",
				Type: lifecycle_dto.FileEventTypeRemove,
			},
		}

		service.handleAssetChange(fec)

		assert.Len(t, mockRegistry.deletedArtefacts, 1)
		assert.Equal(t, "test-module/assets/old.png", mockRegistry.deletedArtefacts[0])
	})

	t.Run("deletes on rename event", func(t *testing.T) {
		t.Parallel()

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "assets/renamed.png",
			artefactID: "test-module/assets/renamed.png",
			event: lifecycle_dto.FileEvent{
				Path: "/project/assets/renamed.png",
				Type: lifecycle_dto.FileEventTypeRename,
			},
		}

		service.handleAssetChange(fec)

		assert.Len(t, mockRegistry.deletedArtefacts, 1)
	})
}

func TestLifecycleService_upsertAssetArtefact(t *testing.T) {
	t.Parallel()

	t.Run("successfully upserts artefact", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/project/assets/logo.png", []byte("png content"))

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "assets/logo.png",
			artefactID: "test-module/assets/logo.png",
			event: lifecycle_dto.FileEvent{
				Path: "/project/assets/logo.png",
				Type: lifecycle_dto.FileEventTypeCreate,
			},
		}

		service.upsertAssetArtefact(fec)

		assert.Len(t, mockRegistry.upsertedArtefacts, 1)
		assert.Equal(t, "test-module/assets/logo.png", mockRegistry.upsertedArtefacts[0])
	})

	t.Run("logs error when file cannot be opened", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "assets/missing.png",
			artefactID: "test-module/assets/missing.png",
			event: lifecycle_dto.FileEvent{
				Path: "/project/assets/missing.png",
				Type: lifecycle_dto.FileEventTypeCreate,
			},
		}

		service.upsertAssetArtefact(fec)

		assert.Empty(t, mockRegistry.upsertedArtefacts)
	})

	t.Run("logs error when registry upsert fails", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/project/assets/logo.png", []byte("png content"))

		mockRegistry := &mockTrackingRegistryService{
			upsertError: errors.New("registry error"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "assets/logo.png",
			artefactID: "test-module/assets/logo.png",
			event: lifecycle_dto.FileEvent{
				Path: "/project/assets/logo.png",
				Type: lifecycle_dto.FileEventTypeCreate,
			},
		}

		service.upsertAssetArtefact(fec)
	})
}

func TestLifecycleService_deleteAssetArtefact(t *testing.T) {
	t.Parallel()

	t.Run("successfully deletes artefact", func(t *testing.T) {
		t.Parallel()

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			artefactID: "test-module/assets/old.png",
		}

		service.deleteAssetArtefact(fec)

		assert.Len(t, mockRegistry.deletedArtefacts, 1)
		assert.Equal(t, "test-module/assets/old.png", mockRegistry.deletedArtefacts[0])
	})

	t.Run("ignores ErrArtefactNotFound error", func(t *testing.T) {
		t.Parallel()

		mockRegistry := &mockTrackingRegistryService{
			deleteError: registry_domain.ErrArtefactNotFound,
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			artefactID: "test-module/assets/nonexistent.png",
		}

		service.deleteAssetArtefact(fec)
	})

	t.Run("logs other delete errors", func(t *testing.T) {
		t.Parallel()

		mockRegistry := &mockTrackingRegistryService{
			deleteError: errors.New("storage error"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			artefactID: "test-module/assets/problematic.png",
		}

		service.deleteAssetArtefact(fec)
	})
}

func TestLifecycleService_processBuildNotification(t *testing.T) {
	t.Parallel()

	t.Run("handles nil result gracefully", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		service := mustBuildLifecycleService(t, deps)

		notification := coordinator_domain.BuildNotification{
			CausationID: "test-123",
			Result:      nil,
		}

		service.processBuildNotification(context.Background(), notification)
	})

	t.Run("processes asset manifest when pipeline present", func(t *testing.T) {
		t.Parallel()

		mockPipeline := &mockTrackingAssetPipeline{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.AssetPipeline = mockPipeline

		service := mustBuildLifecycleService(t, deps)

		notification := coordinator_domain.BuildNotification{
			CausationID: "test-123",
			Result: &annotator_dto.ProjectAnnotationResult{
				FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
					{SourcePath: "test-artefact"},
				},
			},
		}

		service.processBuildNotification(context.Background(), notification)

		assert.True(t, mockPipeline.processed)
	})
}

func TestLifecycleService_processAssetManifest(t *testing.T) {
	t.Parallel()

	t.Run("no-op when pipeline is nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.AssetPipeline = nil

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{SourcePath: "test-artefact"},
			},
		}

		service.processAssetManifest(context.Background(), result)
	})

	t.Run("no-op when manifest is empty", func(t *testing.T) {
		t.Parallel()

		mockPipeline := &mockTrackingAssetPipeline{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.AssetPipeline = mockPipeline

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{},
		}

		service.processAssetManifest(context.Background(), result)

		assert.False(t, mockPipeline.processed)
	})

	t.Run("processes manifest when pipeline present and manifest non-empty", func(t *testing.T) {
		t.Parallel()

		mockPipeline := &mockTrackingAssetPipeline{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.AssetPipeline = mockPipeline

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{SourcePath: "test-artefact"},
			},
		}

		service.processAssetManifest(context.Background(), result)

		assert.True(t, mockPipeline.processed)
	})

	t.Run("logs error when pipeline processing fails", func(t *testing.T) {
		t.Parallel()

		mockPipeline := &mockTrackingAssetPipeline{
			processError: errors.New("pipeline error"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.AssetPipeline = mockPipeline

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{SourcePath: "test-artefact"},
			},
		}

		service.processAssetManifest(context.Background(), result)

		assert.True(t, mockPipeline.processed)
	})
}

func TestLifecycleService_handleInterpretedBuild(t *testing.T) {
	t.Parallel()

	t.Run("no-op when orchestrator is nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = nil

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleInterpretedBuild(context.Background(), result)
	})

	t.Run("no-op when templater service is nil", func(t *testing.T) {
		t.Parallel()

		mockOrchestrator := &mockTrackingInterpretedOrchestrator{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = mockOrchestrator
		deps.TemplaterService = nil

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleInterpretedBuild(context.Background(), result)

		assert.False(t, mockOrchestrator.buildCalled)
	})

	t.Run("calls handleInitialBuild when not initialised", func(t *testing.T) {
		t.Parallel()

		mockOrchestrator := &mockTrackingInterpretedOrchestrator{
			initialised: false,
		}
		mockTemplater := &mockTrackingTemplaterSwapper{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = mockOrchestrator
		deps.TemplaterService = mockTemplater

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleInterpretedBuild(context.Background(), result)

		assert.True(t, mockOrchestrator.buildCalled)
	})

	t.Run("calls handleIncrementalBuild when already initialised", func(t *testing.T) {
		t.Parallel()

		mockOrchestrator := &mockTrackingInterpretedOrchestrator{
			initialised: true,
		}
		mockTemplater := &mockTrackingTemplaterSwapper{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = mockOrchestrator
		deps.TemplaterService = mockTemplater

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleInterpretedBuild(context.Background(), result)

		assert.True(t, mockOrchestrator.markDirtyCalled)
	})
}

func TestLifecycleService_handleInitialBuild(t *testing.T) {
	t.Parallel()

	t.Run("builds and sets runner successfully", func(t *testing.T) {
		t.Parallel()

		mockRunner := &mockInterpretedRunner{
			keys: []string{"page1"},
			entries: map[string]templater_domain.PageEntryView{
				"page1": &templater_domain.MockPageEntryView{},
			},
		}
		mockOrchestrator := &mockTrackingInterpretedOrchestrator{
			runner: mockRunner,
		}
		mockTemplater := &mockTrackingTemplaterSwapper{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = mockOrchestrator
		deps.TemplaterService = mockTemplater

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleInitialBuild(context.Background(), result)

		assert.True(t, mockOrchestrator.buildCalled)
		assert.True(t, mockTemplater.setRunnerCalled)
	})

	t.Run("logs error when build fails", func(t *testing.T) {
		t.Parallel()

		mockOrchestrator := &mockTrackingInterpretedOrchestrator{
			buildError: errors.New("build failed"),
		}
		mockTemplater := &mockTrackingTemplaterSwapper{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = mockOrchestrator
		deps.TemplaterService = mockTemplater

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleInitialBuild(context.Background(), result)

		assert.True(t, mockOrchestrator.buildCalled)
		assert.False(t, mockTemplater.setRunnerCalled)
	})

	t.Run("reloads routes when router manager present", func(t *testing.T) {
		t.Parallel()

		mockRunner := &mockInterpretedRunner{
			keys: []string{"page1"},
			entries: map[string]templater_domain.PageEntryView{
				"page1": &templater_domain.MockPageEntryView{},
			},
		}
		mockOrchestrator := &mockTrackingInterpretedOrchestrator{
			runner: mockRunner,
		}
		mockTemplater := &mockTrackingTemplaterSwapper{}
		mockRouter := &mockTrackingRouterManager{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = mockOrchestrator
		deps.TemplaterService = mockTemplater
		deps.RouterManager = mockRouter

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleInitialBuild(context.Background(), result)

		assert.True(t, mockRouter.reloadCalled)
	})
}

func TestLifecycleService_handleIncrementalBuild(t *testing.T) {
	t.Parallel()

	t.Run("marks components dirty", func(t *testing.T) {
		t.Parallel()

		mockOrchestrator := &mockTrackingInterpretedOrchestrator{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = mockOrchestrator

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleIncrementalBuild(context.Background(), result)

		assert.True(t, mockOrchestrator.markDirtyCalled)
	})

	t.Run("logs error when mark dirty fails", func(t *testing.T) {
		t.Parallel()

		mockOrchestrator := &mockTrackingInterpretedOrchestrator{
			markDirtyError: errors.New("mark dirty failed"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.InterpretedOrchestrator = mockOrchestrator

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{}

		service.handleIncrementalBuild(context.Background(), result)

		assert.True(t, mockOrchestrator.markDirtyCalled)
	})
}

func TestLifecycleService_reloadRoutesIfNeeded(t *testing.T) {
	t.Parallel()

	t.Run("no-op when router manager is nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RouterManager = nil

		service := mustBuildLifecycleService(t, deps)

		mockRunner := &mockInterpretedRunner{
			keys: []string{"page1"},
		}

		service.reloadRoutesIfNeeded(context.Background(), mockRunner)
	})

	t.Run("reloads routes successfully", func(t *testing.T) {
		t.Parallel()

		mockRouter := &mockTrackingRouterManager{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RouterManager = mockRouter

		service := mustBuildLifecycleService(t, deps)

		mockRunner := &mockInterpretedRunner{
			keys: []string{"page1"},
			entries: map[string]templater_domain.PageEntryView{
				"page1": &templater_domain.MockPageEntryView{},
			},
		}

		service.reloadRoutesIfNeeded(context.Background(), mockRunner)

		assert.True(t, mockRouter.reloadCalled)
	})

	t.Run("logs error when reload fails", func(t *testing.T) {
		t.Parallel()

		mockRouter := &mockTrackingRouterManager{
			reloadError: errors.New("reload failed"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RouterManager = mockRouter

		service := mustBuildLifecycleService(t, deps)

		mockRunner := &mockInterpretedRunner{
			keys: []string{"page1"},
			entries: map[string]templater_domain.PageEntryView{
				"page1": &templater_domain.MockPageEntryView{},
			},
		}

		service.reloadRoutesIfNeeded(context.Background(), mockRunner)

		assert.True(t, mockRouter.reloadCalled)
	})
}

type mockTrackingRegistryService struct {
	upsertError       error
	deleteError       error
	upsertedArtefacts []string
	deletedArtefacts  []string
	mu                sync.Mutex
}

func (m *mockTrackingRegistryService) UpsertArtefact(_ context.Context, artefactID string, _ string, _ io.Reader, _ string, _ []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
	m.mu.Lock()
	m.upsertedArtefacts = append(m.upsertedArtefacts, artefactID)
	upsertError := m.upsertError
	m.mu.Unlock()

	if upsertError != nil {
		return nil, upsertError
	}
	return &registry_dto.ArtefactMeta{}, nil
}

func (m *mockTrackingRegistryService) DeleteArtefact(_ context.Context, artefactID string) error {
	m.mu.Lock()
	m.deletedArtefacts = append(m.deletedArtefacts, artefactID)
	deleteError := m.deleteError
	m.mu.Unlock()

	return deleteError
}

func (m *mockTrackingRegistryService) AddVariant(_ context.Context, _ string, _ *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) GetArtefact(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) GetMultipleArtefacts(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) ListAllArtefactIDs(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) SearchArtefacts(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) SearchArtefactsByTagValues(_ context.Context, _ string, _ []string) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) FindArtefactByVariantStorageKey(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) GetVariantData(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) GetVariantChunk(_ context.Context, _ *registry_dto.Variant, _ string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) GetVariantDataRange(_ context.Context, _ *registry_dto.Variant, _, _ int64) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) GetBlobStore(_ string) (registry_domain.BlobStore, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) PopGCHints(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
	return nil, nil
}
func (m *mockTrackingRegistryService) ListBlobStoreIDs() []string {
	return nil
}
func (m *mockTrackingRegistryService) ArtefactEventsPublished() int64 {
	return 0
}

type mockTrackingAssetPipeline struct {
	processError error
	processed    bool
}

func (m *mockTrackingAssetPipeline) ProcessBuildResult(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) error {
	m.processed = true
	return m.processError
}

type mockTrackingInterpretedOrchestrator struct {
	buildError      error
	markDirtyError  error
	runner          templater_domain.ManifestRunnerPort
	initialised     bool
	buildCalled     bool
	markDirtyCalled bool
}

func (m *mockTrackingInterpretedOrchestrator) BuildRunner(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) (templater_domain.ManifestRunnerPort, error) {
	m.buildCalled = true
	if m.buildError != nil {
		return nil, m.buildError
	}
	return m.runner, nil
}

func (m *mockTrackingInterpretedOrchestrator) MarkDirty(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) error {
	m.markDirtyCalled = true
	return m.markDirtyError
}

func (m *mockTrackingInterpretedOrchestrator) MarkComponentsDirty(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) error {
	return nil
}

func (m *mockTrackingInterpretedOrchestrator) IsInitialised() bool {
	return m.initialised
}

func (m *mockTrackingInterpretedOrchestrator) GetAffectedComponents(_ string) []string { return nil }

func (m *mockTrackingInterpretedOrchestrator) ProactiveRecompile(_ context.Context) error {
	return nil
}

type mockTrackingTemplaterSwapper struct {
	setRunnerCalled bool
}

func (m *mockTrackingTemplaterSwapper) SetRunner(_ templater_domain.ManifestRunnerPort) {
	m.setRunnerCalled = true
}

type mockTrackingRouterManager struct {
	reloadError  error
	reloadCalled bool
}

func (m *mockTrackingRouterManager) ReloadRoutes(_ context.Context, _ templater_domain.ManifestStoreView) error {
	m.reloadCalled = true
	return m.reloadError
}

type mockTrackingCoordinatorService struct {
	rebuildCalled bool
}

func (m *mockTrackingCoordinatorService) Subscribe(_ string) (<-chan coordinator_domain.BuildNotification, coordinator_domain.UnsubscribeFunc) {
	return make(chan coordinator_domain.BuildNotification), func() {}
}

func (m *mockTrackingCoordinatorService) GetResult(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
	return nil, nil
}

func (m *mockTrackingCoordinatorService) GetOrBuildProject(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
	return nil, nil
}

func (m *mockTrackingCoordinatorService) RequestRebuild(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) {
	m.rebuildCalled = true
}

func (m *mockTrackingCoordinatorService) GetLastSuccessfulBuild() (*annotator_dto.ProjectAnnotationResult, bool) {
	return nil, false
}

func (m *mockTrackingCoordinatorService) Invalidate(_ context.Context) error {
	return nil
}

func (m *mockTrackingCoordinatorService) Shutdown(_ context.Context) {}

func TestLifecycleService_watchLoop(t *testing.T) {
	t.Parallel()

	t.Run("exits on context cancellation", func(t *testing.T) {
		t.Parallel()

		events := make(chan lifecycle_dto.FileEvent, 10)
		service := newLifecycleTestBuilder().BuildInternal(t)
		service.stopChan = make(chan struct{})

		ctx, cancel := context.WithCancelCause(context.Background())

		done := make(chan struct{})
		go func() {
			service.watchLoop(ctx, events)
			close(done)
		}()

		cancel(fmt.Errorf("test: simulating cancelled context"))

		select {
		case <-done:

		case <-time.After(time.Second):
			t.Error("watchLoop did not exit on context cancellation")
		}
	})

	t.Run("exits on stop channel close", func(t *testing.T) {
		t.Parallel()

		events := make(chan lifecycle_dto.FileEvent, 10)
		service := newLifecycleTestBuilder().BuildInternal(t)
		service.stopChan = make(chan struct{})

		ctx := context.Background()

		done := make(chan struct{})
		go func() {
			service.watchLoop(ctx, events)
			close(done)
		}()

		close(service.stopChan)

		select {
		case <-done:

		case <-time.After(time.Second):
			t.Error("watchLoop did not exit on stop channel close")
		}
	})

	t.Run("exits on events channel close", func(t *testing.T) {
		t.Parallel()

		events := make(chan lifecycle_dto.FileEvent, 10)
		service := newLifecycleTestBuilder().BuildInternal(t)
		service.stopChan = make(chan struct{})

		ctx := context.Background()

		done := make(chan struct{})
		go func() {
			service.watchLoop(ctx, events)
			close(done)
		}()

		close(events)

		select {
		case <-done:

		case <-time.After(time.Second):
			t.Error("watchLoop did not exit on events channel close")
		}
	})

	t.Run("processes file events", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddFile("/project/assets/test.css", []byte("css content"))

		events := make(chan lifecycle_dto.FileEvent, 10)
		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)
		service.stopChan = make(chan struct{})

		ctx, cancel := context.WithCancelCause(context.Background())

		done := make(chan struct{})
		go func() {
			service.watchLoop(ctx, events)
			close(done)
		}()

		events <- lifecycle_dto.FileEvent{
			Path: "/project/assets/test.css",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		time.Sleep(50 * time.Millisecond)

		cancel(fmt.Errorf("test: cleanup"))
		<-done

		assert.GreaterOrEqual(t, len(mockRegistry.upsertedArtefacts), 1)
	})
}

func TestLifecycleService_walkAssetDir(t *testing.T) {
	t.Parallel()

	t.Run("walks and collects files from directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddFile("/project/assets/image.png", []byte("png data"))
		mockFS.AddFile("/project/assets/style.css", []byte("css data"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		fileChan := make(chan lifecycle_dto.FileEvent, 10)
		var wg sync.WaitGroup
		wg.Add(1)

		go service.walkAssetDir(context.Background(), "/project/assets", fileChan, &wg)

		wg.Wait()
		close(fileChan)

		var events []lifecycle_dto.FileEvent
		for event := range fileChan {
			events = append(events, event)
		}

		assert.GreaterOrEqual(t, len(events), 2)
	})

	t.Run("handles non-existent directory gracefully", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		fileChan := make(chan lifecycle_dto.FileEvent, 10)
		var wg sync.WaitGroup
		wg.Add(1)

		go service.walkAssetDir(context.Background(), "/nonexistent", fileChan, &wg)

		wg.Wait()
		close(fileChan)

		var events []lifecycle_dto.FileEvent
		for event := range fileChan {
			events = append(events, event)
		}

		assert.Empty(t, events)
	})

	t.Run("skips directories and only collects files", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddDir("/project/assets/subdir")
		mockFS.AddFile("/project/assets/file.css", []byte("css"))
		mockFS.AddFile("/project/assets/subdir/nested.css", []byte("nested css"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		fileChan := make(chan lifecycle_dto.FileEvent, 10)
		var wg sync.WaitGroup
		wg.Add(1)

		go service.walkAssetDir(context.Background(), "/project/assets", fileChan, &wg)

		wg.Wait()
		close(fileChan)

		var events []lifecycle_dto.FileEvent
		for event := range fileChan {
			events = append(events, event)
		}

		for _, event := range events {
			assert.False(t, strings.HasSuffix(event.Path, "/"))
		}
		assert.GreaterOrEqual(t, len(events), 2)
	})

	t.Run("handles nested directory structure", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddDir("/project/assets/images")
		mockFS.AddDir("/project/assets/images/icons")
		mockFS.AddFile("/project/assets/style.css", []byte("css"))
		mockFS.AddFile("/project/assets/images/logo.png", []byte("logo"))
		mockFS.AddFile("/project/assets/images/icons/menu.svg", []byte("svg"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		fileChan := make(chan lifecycle_dto.FileEvent, 10)
		var wg sync.WaitGroup
		wg.Add(1)

		go service.walkAssetDir(context.Background(), "/project/assets", fileChan, &wg)

		wg.Wait()
		close(fileChan)

		var events []lifecycle_dto.FileEvent
		for event := range fileChan {
			events = append(events, event)
		}

		assert.GreaterOrEqual(t, len(events), 3)
	})
}

func TestLifecycleService_processFilesWithLimiter(t *testing.T) {
	t.Parallel()

	t.Run("handles empty file list", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		service := mustBuildLifecycleService(t, deps)

		limiter := make(chan struct{}, 4)

		service.processFilesWithLimiter(context.Background(), []lifecycle_dto.FileEvent{}, limiter)
	})

	t.Run("processes all files concurrently", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddFile("/project/assets/file1.css", []byte("css1"))
		mockFS.AddFile("/project/assets/file2.css", []byte("css2"))
		mockFS.AddFile("/project/assets/file3.css", []byte("css3"))

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		files := []lifecycle_dto.FileEvent{
			{Path: "/project/assets/file1.css", Type: lifecycle_dto.FileEventTypeCreate},
			{Path: "/project/assets/file2.css", Type: lifecycle_dto.FileEventTypeCreate},
			{Path: "/project/assets/file3.css", Type: lifecycle_dto.FileEventTypeCreate},
		}

		limiter := make(chan struct{}, 4)

		service.processFilesWithLimiter(context.Background(), files, limiter)

		assert.Len(t, mockRegistry.upsertedArtefacts, 3)
	})

	t.Run("respects limiter capacity", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		for i := range 10 {
			mockFS.AddFile(fmt.Sprintf("/project/assets/file%d.css", i), []byte("css"))
		}

		mockRegistry := &mockCountingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		files := make([]lifecycle_dto.FileEvent, 0, 10)
		for i := range 10 {
			files = append(files, lifecycle_dto.FileEvent{
				Path: fmt.Sprintf("/project/assets/file%d.css", i),
				Type: lifecycle_dto.FileEventTypeCreate,
			})
		}

		limiter := make(chan struct{}, 2)

		service.processFilesWithLimiter(context.Background(), files, limiter)

		assert.Equal(t, int64(10), mockRegistry.upsertCount.Load())
	})
}

func TestLifecycleService_seedThemeArtefact(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when renderer is nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Renderer = nil

		service := mustBuildLifecycleService(t, deps)

		err := service.seedThemeArtefact(context.Background())
		assert.NoError(t, err)
	})

	t.Run("returns error when BuildThemeCSS fails", func(t *testing.T) {
		t.Parallel()

		mockRenderer := &mockTrackingRenderer{
			buildThemeCSSError: errors.New("failed to build theme CSS"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Renderer = mockRenderer

		service := mustBuildLifecycleService(t, deps)

		err := service.seedThemeArtefact(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to build theme CSS")
	})

	t.Run("returns error when UpsertArtefact fails", func(t *testing.T) {
		t.Parallel()

		mockRenderer := &mockTrackingRenderer{
			buildThemeCSSResult: []byte("body { color: black; }"),
		}
		mockRegistry := &mockTrackingRegistryService{
			upsertError: errors.New("failed to upsert"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Renderer = mockRenderer
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		err := service.seedThemeArtefact(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create theme artefact")
	})

	t.Run("succeeds when all operations complete", func(t *testing.T) {
		t.Parallel()

		mockRenderer := &mockTrackingRenderer{
			buildThemeCSSResult: []byte("body { color: black; }"),
		}
		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Renderer = mockRenderer
		deps.RegistryService = mockRegistry

		service := mustBuildLifecycleService(t, deps)

		err := service.seedThemeArtefact(context.Background())

		require.NoError(t, err)
		assert.Contains(t, mockRegistry.upsertedArtefacts, "theme.css")
	})
}

func TestLifecycleService_buildFileEventContext_pathConversion(t *testing.T) {
	t.Parallel()

	t.Run("converts backslashes to forward slashes", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddDir("/project/assets/nested")
		mockFS.AddDir("/project/assets/nested/deep")
		mockFS.AddFile("/project/assets/nested/deep/file.css", []byte("css"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/assets/nested/deep/file.css",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		fec, ok := service.buildFileEventContext(context.Background(), event)

		assert.True(t, ok)

		assert.Equal(t, "assets/nested/deep/file.css", fec.relPath)
		assert.NotContains(t, fec.relPath, "\\")
	})

	t.Run("builds correct artefact ID with module name", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddFile("/project/assets/logo.png", []byte("png"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/project/assets/logo.png",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		fec, ok := service.buildFileEventContext(context.Background(), event)

		assert.True(t, ok)
		assert.Equal(t, "test-module/assets/logo.png", fec.artefactID)
	})

	t.Run("handles relative path computation error", func(t *testing.T) {
		t.Parallel()

		mockFS := &mockFailingRelFileSystem{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		event := lifecycle_dto.FileEvent{
			Path: "/completely/different/path/file.css",
			Type: lifecycle_dto.FileEventTypeCreate,
		}

		_, ok := service.buildFileEventContext(context.Background(), event)

		assert.False(t, ok)
	})
}

func TestLifecycleService_processWalkedFile(t *testing.T) {
	t.Parallel()

	t.Run("sends event for relevant file", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddFile("/project/assets/style.css", []byte("css"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		fileChan := make(chan lifecycle_dto.FileEvent, 10)
		paths := &deps.PathsConfig

		err := service.processWalkedFile(context.Background(), "/project/assets/style.css", paths, fileChan)

		require.NoError(t, err)
		assert.Len(t, fileChan, 1)

		event := <-fileChan
		assert.Equal(t, "/project/assets/style.css", event.Path)
		assert.Equal(t, lifecycle_dto.FileEventTypeCreate, event.Type)
	})

	t.Run("skips irrelevant files", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		fileChan := make(chan lifecycle_dto.FileEvent, 10)
		paths := &deps.PathsConfig

		err := service.processWalkedFile(context.Background(), "/project/node_modules/package.json", paths, fileChan)

		require.NoError(t, err)
		assert.Empty(t, fileChan)
	})

	t.Run("converts path separators to forward slashes", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/assets")
		mockFS.AddDir("/project/assets/images")
		mockFS.AddFile("/project/assets/images/logo.png", []byte("png"))

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.AssetsSourceDir = "assets"

		service := mustBuildLifecycleService(t, deps)

		fileChan := make(chan lifecycle_dto.FileEvent, 10)
		paths := &deps.PathsConfig

		err := service.processWalkedFile(context.Background(), "/project/assets/images/logo.png", paths, fileChan)

		require.NoError(t, err)
		assert.Len(t, fileChan, 1)
	})
}

type mockTrackingRenderer struct {
	buildThemeCSSError  error
	buildThemeCSSResult []byte
}

func (m *mockTrackingRenderer) BuildThemeCSS(_ context.Context, _ *config.WebsiteConfig) ([]byte, error) {
	if m.buildThemeCSSError != nil {
		return nil, m.buildThemeCSSError
	}
	return m.buildThemeCSSResult, nil
}

func (m *mockTrackingRenderer) CollectMetadata(_ context.Context, _ *http.Request, _ *templater_dto.InternalMetadata, _ *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
	return nil, nil, nil
}

func (m *mockTrackingRenderer) RenderAST(_ context.Context, _ io.Writer, _ http.ResponseWriter, _ *http.Request, _ render_domain.RenderASTOptions) error {
	return nil
}

func (m *mockTrackingRenderer) RenderEmail(_ context.Context, _ io.Writer, _ *http.Request, _ render_domain.RenderEmailOptions) error {
	return nil
}

func (m *mockTrackingRenderer) RenderASTToPlainText(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
	return "", nil
}

func (m *mockTrackingRenderer) GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest {
	return nil
}

type mockFailingRelFileSystem struct{}

func (m *mockFailingRelFileSystem) WalkDir(_ string, _ fs.WalkDirFunc) error {
	return nil
}

func (m *mockFailingRelFileSystem) Open(_ string) (io.ReadCloser, error) {
	return nil, fs.ErrNotExist
}

func (m *mockFailingRelFileSystem) Stat(_ string) (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func (m *mockFailingRelFileSystem) Rel(_, _ string) (string, error) {
	return "", errors.New("cannot compute relative path")
}

func (m *mockFailingRelFileSystem) Join(element ...string) string {
	return filepath.Join(element...)
}

func (m *mockFailingRelFileSystem) IsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

type mockCountingRegistryService struct {
	upsertCount atomic.Int64
}

func (m *mockCountingRegistryService) UpsertArtefact(_ context.Context, _ string, _ string, _ io.Reader, _ string, _ []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
	m.upsertCount.Add(1)
	return &registry_dto.ArtefactMeta{}, nil
}

func (m *mockCountingRegistryService) DeleteArtefact(_ context.Context, _ string) error {
	return nil
}

func (m *mockCountingRegistryService) AddVariant(_ context.Context, _ string, _ *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) GetArtefact(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) GetMultipleArtefacts(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) ListAllArtefactIDs(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) SearchArtefacts(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) SearchArtefactsByTagValues(_ context.Context, _ string, _ []string) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) FindArtefactByVariantStorageKey(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) GetVariantData(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) GetVariantChunk(_ context.Context, _ *registry_dto.Variant, _ string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) GetVariantDataRange(_ context.Context, _ *registry_dto.Variant, _, _ int64) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) GetBlobStore(_ string) (registry_domain.BlobStore, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) PopGCHints(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
	return nil, nil
}
func (m *mockCountingRegistryService) ListBlobStoreIDs() []string {
	return nil
}
func (m *mockCountingRegistryService) ArtefactEventsPublished() int64 {
	return 0
}

type mockTrackingClearComponentRenderRegistry struct {
	mockRenderRegistryPort
	clearedComponentIDs []string
}

func (m *mockTrackingClearComponentRenderRegistry) ClearComponentCache(_ context.Context, componentType string) {
	m.clearedComponentIDs = append(m.clearedComponentIDs, componentType)
}

type mockTrackingFileSystemWatcher struct {
	eventsChan   chan lifecycle_dto.FileEvent
	updateError  error
	updatedFiles []string
	updateCalled bool
}

func (w *mockTrackingFileSystemWatcher) Watch(_ context.Context, _, _ []string) (<-chan lifecycle_dto.FileEvent, error) {
	return w.eventsChan, nil
}

func (w *mockTrackingFileSystemWatcher) UpdateWatchedFiles(_ context.Context, files []string) error {
	w.updateCalled = true
	w.updatedFiles = files
	return w.updateError
}

func (w *mockTrackingFileSystemWatcher) Close() error {
	return nil
}

type mockTrackingComponentRegistry struct {
	registerError        error
	registeredComponents []component_dto.ComponentDefinition
}

func (m *mockTrackingComponentRegistry) Register(definition component_dto.ComponentDefinition) error {
	if m.registerError != nil {
		return m.registerError
	}
	m.registeredComponents = append(m.registeredComponents, definition)
	return nil
}

func (m *mockTrackingComponentRegistry) RegisterBatch(_ []component_dto.ComponentDefinition) error {
	return nil
}

func (m *mockTrackingComponentRegistry) IsRegistered(_ string) bool {
	return false
}

func (m *mockTrackingComponentRegistry) Get(_ string) (*component_dto.ComponentDefinition, bool) {
	return nil, false
}

func (m *mockTrackingComponentRegistry) All() []component_dto.ComponentDefinition {
	return m.registeredComponents
}

func (m *mockTrackingComponentRegistry) Count() int {
	return len(m.registeredComponents)
}

func (m *mockTrackingComponentRegistry) TagNames() []string {
	names := make([]string, len(m.registeredComponents))
	for i, c := range m.registeredComponents {
		names[i] = c.TagName
	}
	return names
}

type moduleBoundaryResult struct {
	err        error
	moduleBase string
	subpath    string
}

type moduleDirResult struct {
	err       error
	directory string
}

type mockTrackingResolver struct {
	moduleBoundaries map[string]moduleBoundaryResult
	moduleDirs       map[string]moduleDirResult
	moduleName       string
}

func (m *mockTrackingResolver) DetectLocalModule(_ context.Context) error { return nil }
func (m *mockTrackingResolver) GetModuleName() string                     { return m.moduleName }
func (m *mockTrackingResolver) GetBaseDir() string                        { return "/test" }
func (m *mockTrackingResolver) ResolvePKPath(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockTrackingResolver) ResolveCSSPath(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockTrackingResolver) ResolveAssetPath(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockTrackingResolver) ConvertEntryPointPathToManifestKey(_ string) string { return "" }

func (m *mockTrackingResolver) GetModuleDir(_ context.Context, moduleBase string) (string, error) {
	if result, ok := m.moduleDirs[moduleBase]; ok {
		return result.directory, result.err
	}
	return "", errors.New("module not found: " + moduleBase)
}

func (m *mockTrackingResolver) FindModuleBoundary(_ context.Context, modulePath string) (string, string, error) {
	if result, ok := m.moduleBoundaries[modulePath]; ok {
		return result.moduleBase, result.subpath, result.err
	}
	return "", "", errors.New("boundary not found: " + modulePath)
}

func TestLifecycleService_clearComponentCacheIfNeeded(t *testing.T) {
	t.Parallel()

	t.Run("clears component cache for pkc files", func(t *testing.T) {
		t.Parallel()

		mockRenderRegistry := &mockTrackingClearComponentRenderRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RenderRegistryPort = mockRenderRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "components/my-button.pkc",
			artefactID: "test-module/components/my-button.pkc",
			event: lifecycle_dto.FileEvent{
				Path: "/project/components/my-button.pkc",
				Type: lifecycle_dto.FileEventTypeWrite,
			},
		}

		service.clearComponentCacheIfNeeded(fec)

		require.Len(t, mockRenderRegistry.clearedComponentIDs, 1)
		assert.Equal(t, "my-button", mockRenderRegistry.clearedComponentIDs[0])
	})

	t.Run("does not clear cache for non-pkc files", func(t *testing.T) {
		t.Parallel()

		mockRenderRegistry := &mockTrackingClearComponentRenderRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RenderRegistryPort = mockRenderRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "assets/style.css",
			artefactID: "test-module/assets/style.css",
			event: lifecycle_dto.FileEvent{
				Path: "/project/assets/style.css",
				Type: lifecycle_dto.FileEventTypeWrite,
			},
		}

		service.clearComponentCacheIfNeeded(fec)

		assert.Empty(t, mockRenderRegistry.clearedComponentIDs)
	})

	t.Run("no-op when render registry is nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RenderRegistryPort = nil

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "components/my-button.pkc",
			artefactID: "test-module/components/my-button.pkc",
			event: lifecycle_dto.FileEvent{
				Path: "/project/components/my-button.pkc",
				Type: lifecycle_dto.FileEventTypeWrite,
			},
		}

		service.clearComponentCacheIfNeeded(fec)
	})

	t.Run("derives tag name from filename without extension", func(t *testing.T) {
		t.Parallel()

		mockRenderRegistry := &mockTrackingClearComponentRenderRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.RenderRegistryPort = mockRenderRegistry

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "components/nested/fancy-card.pkc",
			artefactID: "test-module/components/nested/fancy-card.pkc",
			event: lifecycle_dto.FileEvent{
				Path: "/project/components/nested/fancy-card.pkc",
				Type: lifecycle_dto.FileEventTypeWrite,
			},
		}

		service.clearComponentCacheIfNeeded(fec)

		require.Len(t, mockRenderRegistry.clearedComponentIDs, 1)
		assert.Equal(t, "fancy-card", mockRenderRegistry.clearedComponentIDs[0])
	})
}

func TestLifecycleService_extractAssetPathsFromManifest(t *testing.T) {
	t.Parallel()

	t.Run("extracts paths with module prefix stripped", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath:           "test-module/assets/logo.png",
					AssetType:            "img",
					TransformationParams: nil,
				},
				{
					SourcePath:           "test-module/assets/style.css",
					AssetType:            "css",
					TransformationParams: nil,
				},
			},
		}

		paths := service.extractAssetPathsFromManifest(result)

		require.Len(t, paths, 2)
		assert.Contains(t, paths[0], "assets/logo.png")
		assert.Contains(t, paths[1], "assets/style.css")
	})

	t.Run("handles paths without module prefix", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.Resolver = nil
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath:           "assets/image.jpg",
					AssetType:            "img",
					TransformationParams: nil,
				},
			},
		}

		paths := service.extractAssetPathsFromManifest(result)

		require.Len(t, paths, 1)
		assert.Contains(t, paths[0], "assets/image.jpg")
	})

	t.Run("returns empty for empty manifest", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{},
		}

		paths := service.extractAssetPathsFromManifest(result)

		assert.Empty(t, paths)
	})
}

func TestLifecycleService_updateWatchedFilesFromBuild(t *testing.T) {
	t.Parallel()

	t.Run("no-op when watcher is nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.WatcherAdapter = nil

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{SourcePath: "test-artefact", AssetType: "img", TransformationParams: nil},
			},
		}

		service.updateWatchedFilesFromBuild(context.Background(), result)
	})

	t.Run("no-op when manifest is nil", func(t *testing.T) {
		t.Parallel()

		mockWatcher := &MockFileSystemWatcher{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.WatcherAdapter = mockWatcher

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: nil,
		}

		service.updateWatchedFilesFromBuild(context.Background(), result)
	})

	t.Run("updates watcher with asset paths", func(t *testing.T) {
		t.Parallel()

		mockWatcher := &mockTrackingFileSystemWatcher{
			eventsChan: make(chan lifecycle_dto.FileEvent, 100),
		}
		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.WatcherAdapter = mockWatcher
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{SourcePath: "test-module/assets/logo.png", AssetType: "img", TransformationParams: nil},
			},
		}

		service.updateWatchedFilesFromBuild(context.Background(), result)

		assert.True(t, mockWatcher.updateCalled)
		require.Len(t, mockWatcher.updatedFiles, 1)
	})

	t.Run("logs error when update fails", func(t *testing.T) {
		t.Parallel()

		mockWatcher := &mockTrackingFileSystemWatcher{
			eventsChan:  make(chan lifecycle_dto.FileEvent, 100),
			updateError: errors.New("update failed"),
		}
		mockFS := NewMockFileSystem()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.WatcherAdapter = mockWatcher
		deps.FileSystem = mockFS
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{SourcePath: "test-module/assets/logo.png", AssetType: "img", TransformationParams: nil},
			},
		}

		service.updateWatchedFilesFromBuild(context.Background(), result)
		assert.True(t, mockWatcher.updateCalled)
	})
}

func TestLifecycleService_handleCoreSourceChange_pkcFile(t *testing.T) {
	t.Parallel()

	t.Run("upserts artefact for pkc file change", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/project/components/my-comp.pkc", []byte("component source"))

		mockRegistry := &mockTrackingRegistryService{}
		mockRenderRegistry := &mockTrackingClearComponentRenderRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry
		deps.RenderRegistryPort = mockRenderRegistry
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.ComponentsSourceDir = "components"

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "components/my-comp.pkc",
			artefactID: "test-module/components/my-comp.pkc",
			event: lifecycle_dto.FileEvent{
				Path: "/project/components/my-comp.pkc",
				Type: lifecycle_dto.FileEventTypeWrite,
			},
		}

		service.handleCoreSourceChange(fec, false)

		require.Len(t, mockRenderRegistry.clearedComponentIDs, 1)
		assert.Equal(t, "my-comp", mockRenderRegistry.clearedComponentIDs[0])
		assert.Len(t, mockRegistry.upsertedArtefacts, 1)
	})

	t.Run("does not upsert pkc on initial seed", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/project/components/my-comp.pkc", []byte("component source"))

		mockRegistry := &mockTrackingRegistryService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.RegistryService = mockRegistry
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.ComponentsSourceDir = "components"

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "components/my-comp.pkc",
			artefactID: "test-module/components/my-comp.pkc",
			event: lifecycle_dto.FileEvent{
				Path: "/project/components/my-comp.pkc",
				Type: lifecycle_dto.FileEventTypeCreate,
			},
		}

		service.handleCoreSourceChange(fec, true)

		assert.Empty(t, mockRegistry.upsertedArtefacts)
	})

	t.Run("requests rebuild when no invalidator but coordinator present", func(t *testing.T) {
		t.Parallel()

		mockCoordinator := &mockTrackingCoordinatorService{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.CoordinatorService = mockCoordinator
		deps.BuildCacheInvalidator = nil
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.PathsConfig.PagesSourceDir = "pages"

		service := mustBuildLifecycleService(t, deps)

		fec := fileEventContext{
			ctx:        context.Background(),
			relPath:    "pages/home.pk",
			artefactID: "test-module/pages/home.pk",
			event: lifecycle_dto.FileEvent{
				Path: "/project/pages/home.pk",
				Type: lifecycle_dto.FileEventTypeWrite,
			},
		}

		service.handleCoreSourceChange(fec, false)

		assert.True(t, mockCoordinator.rebuildCalled)
	})
}

func TestLifecycleService_discoverAndRegisterComponents(t *testing.T) {
	t.Parallel()

	t.Run("registers local pkc files", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/components")
		mockFS.AddFile("/project/components/my-button.pkc", []byte("button"))
		mockFS.AddFile("/project/components/my-card.pkc", []byte("card"))

		mockComponentRegistry := &mockTrackingComponentRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.ComponentRegistry = mockComponentRegistry
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.ComponentsSourceDir = "components"

		service := mustBuildLifecycleService(t, deps)

		err := service.discoverAndRegisterComponents(context.Background())

		assert.NoError(t, err)
		assert.Len(t, mockComponentRegistry.registeredComponents, 2)
	})

	t.Run("handles non-existent components directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		mockComponentRegistry := &mockTrackingComponentRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.ComponentRegistry = mockComponentRegistry
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.ComponentsSourceDir = "components"

		service := mustBuildLifecycleService(t, deps)

		err := service.discoverAndRegisterComponents(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, mockComponentRegistry.registeredComponents)
	})

	t.Run("skips non-pkc files", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/components")
		mockFS.AddFile("/project/components/my-button.pkc", []byte("button"))
		mockFS.AddFile("/project/components/readme.md", []byte("docs"))
		mockFS.AddFile("/project/components/style.css", []byte("css"))

		mockComponentRegistry := &mockTrackingComponentRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.ComponentRegistry = mockComponentRegistry
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.ComponentsSourceDir = "components"

		service := mustBuildLifecycleService(t, deps)

		err := service.discoverAndRegisterComponents(context.Background())

		assert.NoError(t, err)
		assert.Len(t, mockComponentRegistry.registeredComponents, 1)
		assert.Equal(t, "my-button", mockComponentRegistry.registeredComponents[0].TagName)
	})

	t.Run("handles empty components directory config", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockComponentRegistry := &mockTrackingComponentRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.ComponentRegistry = mockComponentRegistry
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.ComponentsSourceDir = ""

		service := mustBuildLifecycleService(t, deps)

		err := service.discoverAndRegisterComponents(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, mockComponentRegistry.registeredComponents)
	})

	t.Run("records registration errors without failing", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/components")
		mockFS.AddFile("/project/components/my-button.pkc", []byte("button"))

		mockComponentRegistry := &mockTrackingComponentRegistry{
			registerError: errors.New("duplicate tag name"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.ComponentRegistry = mockComponentRegistry
		deps.PathsConfig.BaseDir = "/project"
		deps.PathsConfig.ComponentsSourceDir = "components"

		service := mustBuildLifecycleService(t, deps)

		err := service.discoverAndRegisterComponents(context.Background())

		assert.NoError(t, err)
	})
}

func TestLifecycleService_walkAndRegisterLocalComponents(t *testing.T) {
	t.Parallel()

	t.Run("registers components and returns count", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/components")
		mockFS.AddFile("/project/components/my-button.pkc", []byte("button"))
		mockFS.AddFile("/project/components/my-card.pkc", []byte("card"))

		mockComponentRegistry := &mockTrackingComponentRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.ComponentRegistry = mockComponentRegistry
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		registered, regErrors, err := service.walkAndRegisterLocalComponents(context.Background(), "/project/components", "/project")

		assert.NoError(t, err)
		assert.Equal(t, 2, registered)
		assert.Empty(t, regErrors)
	})

	t.Run("returns not-exist error for missing directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockComponentRegistry := &mockTrackingComponentRegistry{}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.ComponentRegistry = mockComponentRegistry
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		_, _, err := service.walkAndRegisterLocalComponents(context.Background(), "/project/nonexistent", "/project")

		assert.Error(t, err)
	})

	t.Run("collects registration errors", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project/components")
		mockFS.AddFile("/project/components/bad-comp.pkc", []byte("bad"))

		mockComponentRegistry := &mockTrackingComponentRegistry{
			registerError: errors.New("registration failed"),
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = mockFS
		deps.ComponentRegistry = mockComponentRegistry
		deps.PathsConfig.BaseDir = "/project"

		service := mustBuildLifecycleService(t, deps)

		registered, regErrors, err := service.walkAndRegisterLocalComponents(context.Background(), "/project/components", "/project")

		assert.NoError(t, err)
		assert.Equal(t, 0, registered)
		assert.Len(t, regErrors, 1)
	})
}

func TestLifecycleService_resolveExternalComponentDirs(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when resolver is nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = nil

		service := mustBuildLifecycleService(t, deps)

		result := service.resolveExternalComponentDirs(context.Background())

		assert.Nil(t, result)
	})

	t.Run("returns nil when no external components", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.ExternalComponents = nil

		service := mustBuildLifecycleService(t, deps)

		result := service.resolveExternalComponentDirs(context.Background())

		assert.Nil(t, result)
	})

	t.Run("skips components with empty module path", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.ExternalComponents = []component_dto.ComponentDefinition{
			{
				TagName:    "empty-module",
				SourcePath: "test.pkc",
				ModulePath: "",
				AssetPaths: nil,
				IsExternal: true,
			},
		}

		service := mustBuildLifecycleService(t, deps)

		result := service.resolveExternalComponentDirs(context.Background())

		assert.Empty(t, result)
	})

	t.Run("deduplicates module paths", func(t *testing.T) {
		t.Parallel()

		mockResolver := &mockTrackingResolver{
			moduleName: "test-module",
			moduleBoundaries: map[string]moduleBoundaryResult{
				"github.com/ext/ui": {
					moduleBase: "github.com/ext/ui",
					subpath:    "",
					err:        nil,
				},
			},
			moduleDirs: map[string]moduleDirResult{
				"github.com/ext/ui": {
					directory: "/go/pkg/mod/github.com/ext/ui",
					err:       nil,
				},
			},
		}

		deps := newLifecycleTestBuilder().GetDeps()
		deps.FileSystem = NewMockFileSystem()
		deps.Resolver = mockResolver
		deps.ExternalComponents = []component_dto.ComponentDefinition{
			{
				TagName:    "ext-button",
				SourcePath: "button.pkc",
				ModulePath: "github.com/ext/ui",
				AssetPaths: nil,
				IsExternal: true,
			},
			{
				TagName:    "ext-card",
				SourcePath: "card.pkc",
				ModulePath: "github.com/ext/ui",
				AssetPaths: nil,
				IsExternal: true,
			},
		}

		service := mustBuildLifecycleService(t, deps)

		result := service.resolveExternalComponentDirs(context.Background())

		assert.Len(t, result, 1)
	})
}

func TestLifecycleService_resolveExternalAssetDirs(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when resolver is nil", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = nil

		service := mustBuildLifecycleService(t, deps)

		result := service.resolveExternalAssetDirs(context.Background())

		assert.Nil(t, result)
	})

	t.Run("returns nil when no external components", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.ExternalComponents = nil

		service := mustBuildLifecycleService(t, deps)

		result := service.resolveExternalAssetDirs(context.Background())

		assert.Nil(t, result)
	})

	t.Run("skips components without asset paths", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.ExternalComponents = []component_dto.ComponentDefinition{
			{
				TagName:    "ext-comp",
				SourcePath: "test.pkc",
				ModulePath: "github.com/ext/ui",
				AssetPaths: nil,
				IsExternal: true,
			},
		}

		service := mustBuildLifecycleService(t, deps)

		result := service.resolveExternalAssetDirs(context.Background())

		assert.Empty(t, result)
	})

	t.Run("skips components with empty module path", func(t *testing.T) {
		t.Parallel()

		deps := newLifecycleTestBuilder().GetDeps()
		deps.Resolver = &resolver_domain.MockResolver{
			GetModuleNameFunc: func() string { return "test-module" },
			GetBaseDirFunc:    func() string { return "/test" },
		}
		deps.ExternalComponents = []component_dto.ComponentDefinition{
			{
				TagName:    "ext-comp",
				SourcePath: "test.pkc",
				ModulePath: "",
				AssetPaths: []string{"lib/icons"},
				IsExternal: true,
			},
		}

		service := mustBuildLifecycleService(t, deps)

		result := service.resolveExternalAssetDirs(context.Background())

		assert.Empty(t, result)
	})
}

func TestSandboxedFileSystem_Rel(t *testing.T) {
	t.Parallel()

	fsys := &sandboxedFileSystem{
		sandbox: nil,
	}

	rel, err := fsys.Rel("/home/user/project", "/home/user/project/src/main.go")
	require.NoError(t, err)
	assert.Equal(t, "src/main.go", rel)
}

func TestSandboxedFileSystem_Join(t *testing.T) {
	t.Parallel()

	fsys := &sandboxedFileSystem{
		sandbox: nil,
	}

	result := fsys.Join("a", "b", "c")
	assert.Equal(t, "a/b/c", result)
}

func TestSandboxedFileSystem_IsNotExist(t *testing.T) {
	t.Parallel()

	t.Run("returns true for os.ErrNotExist", func(t *testing.T) {
		t.Parallel()

		fsys := &sandboxedFileSystem{
			sandbox: nil,
		}

		assert.True(t, fsys.IsNotExist(os.ErrNotExist))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		t.Parallel()

		fsys := &sandboxedFileSystem{
			sandbox: nil,
		}

		assert.False(t, fsys.IsNotExist(os.ErrPermission))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		t.Parallel()

		fsys := &sandboxedFileSystem{
			sandbox: nil,
		}

		assert.False(t, fsys.IsNotExist(nil))
	})
}
