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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/wdk/clock"
)

type lifecycleTestBuilder struct {
	mockClock *clock.MockClock
	deps      LifecycleServiceDeps
}

func newLifecycleTestBuilder() *lifecycleTestBuilder {

	return &lifecycleTestBuilder{
		deps: LifecycleServiceDeps{
			PathsConfig: LifecyclePathsConfig{
				BaseDir: "/test",
			},
		},
		mockClock: nil,
	}
}

func (b *lifecycleTestBuilder) WithConfigProvider(cp *config.Provider) *lifecycleTestBuilder {
	b.deps.ConfigProvider = *cp
	return b
}

func (b *lifecycleTestBuilder) WithBaseDir(baseDir string) *lifecycleTestBuilder {
	b.deps.PathsConfig.BaseDir = baseDir
	return b
}

func (b *lifecycleTestBuilder) WithFileSystem(fs FileSystem) *lifecycleTestBuilder {
	b.deps.FileSystem = fs
	return b
}

func (b *lifecycleTestBuilder) WithMockFileSystem() *lifecycleTestBuilder {
	b.deps.FileSystem = NewMockFileSystem()
	return b
}

func (b *lifecycleTestBuilder) WithWatcher(watcher FileSystemWatcher) *lifecycleTestBuilder {
	b.deps.WatcherAdapter = watcher
	return b
}

func (b *lifecycleTestBuilder) WithMockWatcher() *lifecycleTestBuilder {
	b.deps.WatcherAdapter = &MockFileSystemWatcher{}
	return b
}

func (b *lifecycleTestBuilder) WithAssetPipeline(pipeline AssetPipelinePort) *lifecycleTestBuilder {
	b.deps.AssetPipeline = pipeline
	return b
}

func (b *lifecycleTestBuilder) WithBuildCacheInvalidator(invalidator BuildCacheInvalidator) *lifecycleTestBuilder {
	b.deps.BuildCacheInvalidator = invalidator
	return b
}

func (b *lifecycleTestBuilder) WithInterpretedOrchestrator(orchestrator InterpretedBuildOrchestrator) *lifecycleTestBuilder {
	b.deps.InterpretedOrchestrator = orchestrator
	return b
}

func (b *lifecycleTestBuilder) WithTemplaterService(service TemplaterRunnerSwapper) *lifecycleTestBuilder {
	b.deps.TemplaterService = service
	return b
}

func (b *lifecycleTestBuilder) WithRouterManager(manager RouterReloadNotifier) *lifecycleTestBuilder {
	b.deps.RouterManager = manager
	return b
}

func (b *lifecycleTestBuilder) WithClock(clk clock.Clock) *lifecycleTestBuilder {
	b.deps.Clock = clk
	return b
}

func (b *lifecycleTestBuilder) WithMockClock() *lifecycleTestBuilder {
	mockClock := clock.NewMockClock(time.Now())
	b.mockClock = mockClock
	b.deps.Clock = mockClock
	return b
}

func (b *lifecycleTestBuilder) WithMockClockAt(t time.Time) *lifecycleTestBuilder {
	mockClock := clock.NewMockClock(t)
	b.mockClock = mockClock
	b.deps.Clock = mockClock
	return b
}

func (b *lifecycleTestBuilder) GetMockClock() *clock.MockClock {
	return b.mockClock
}

func (b *lifecycleTestBuilder) Build() LifecycleService {
	return NewLifecycleService(&b.deps)
}

func (b *lifecycleTestBuilder) BuildInternal(t *testing.T) *lifecycleService {
	t.Helper()
	service := NewLifecycleService(&b.deps)
	ls, ok := service.(*lifecycleService)
	require.True(t, ok, "expected *lifecycleService")
	return ls
}

func (b *lifecycleTestBuilder) GetDeps() *LifecycleServiceDeps {
	return &b.deps
}
