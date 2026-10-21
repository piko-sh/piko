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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func newSpawnTestService(t *testing.T, coordinator *coordinator_domain.MockCoordinatorService) *lifecycleService {
	t.Helper()
	deps := newLifecycleTestBuilder().GetDeps()
	deps.PathsConfig.BaseDir = "/project"
	deps.PathsConfig.PagesSourceDir = "pages"
	deps.InterpretedOrchestrator = &MockInterpretedOrchestrator{IsInitialisedFunc: func() bool { return true }}
	deps.CoordinatorService = coordinator
	deps.Resolver = &resolver_domain.MockResolver{
		GetModuleNameFunc: func() string { return "test-module" },
		GetBaseDirFunc:    func() string { return "/project" },
	}
	deps.FileSystem = NewMockFileSystem()
	service := mustBuildLifecycleService(t, deps)
	for index := range 4 {
		service.addEntryPointIfNotExists(context.Background(), fmt.Sprintf("test-module/pages/page%d.pk", index), componentType{isPage: true})
	}
	return service
}

func pageChange(index int) fileEventContext {
	relPath := fmt.Sprintf("pages/page%d.pk", index%4)
	return fileEventContext{
		ctx:        context.Background(),
		relPath:    relPath,
		artefactID: "test-module/" + relPath,
		event:      lifecycle_dto.FileEvent{Path: "/project/" + relPath, Type: lifecycle_dto.FileEventTypeWrite},
	}
}

func TestStop_ConcurrentChangesNeverRaceTheDrain(t *testing.T) {
	t.Parallel()

	for range 20 {
		coordinator := &coordinator_domain.MockCoordinatorService{
			GetOrBuildProjectFunc: func(ctx context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Millisecond):
					return &annotator_dto.ProjectAnnotationResult{}, nil
				}
			},
		}
		service := newSpawnTestService(t, coordinator)

		stop := make(chan struct{})
		var producers sync.WaitGroup
		for producer := range 4 {
			producers.Go(func() {
				for index := producer; ; index += 4 {
					select {
					case <-stop:
						return
					default:
					}
					service.handleCoreSourceChange(pageChange(index), false)
					service.registerInitialStyleDeps(context.Background(), nil)
				}
			})
		}

		time.Sleep(5 * time.Millisecond)
		require.NoError(t, service.Stop(context.Background()), "Stop must drain without racing new work")
		close(stop)
		producers.Wait()

		assert.False(t, service.trySpawn(context.Background(), "after-stop", func(context.Context) {
			t.Error("no work may start after Stop")
		}))
	}
}

func TestStop_CancelsARunningTargetedRebuild(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	coordinator := &coordinator_domain.MockCoordinatorService{
		GetOrBuildProjectFunc: func(ctx context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
			close(started)
			<-ctx.Done()
			return nil, context.Cause(ctx)
		},
	}
	service := newSpawnTestService(t, coordinator)

	service.handleCoreSourceChange(pageChange(0), false)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the targeted rebuild did not start")
	}

	require.NoError(t, service.Stop(context.Background()), "a running rebuild must be cancelled, not waited out")
	assert.Zero(t, coordinator.RequestRebuildCallCount.Load(), "a rebuild cancelled by Stop must not fall back to a full rebuild")
}

func TestRunTargetedRebuild_AbandonsAnInvalidatedBuild(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err          error
		name         string
		wantFallback bool
	}{
		{name: "invalidated build is abandoned", err: coordinator_domain.ErrBuildInvalidated, wantFallback: false},
		{name: "wrapped invalidation is abandoned", err: fmt.Errorf("waiting: %w", coordinator_domain.ErrBuildInvalidated), wantFallback: false},
		{name: "other failures fall back to a full rebuild", err: errors.New("annotation failed"), wantFallback: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			coordinator := &coordinator_domain.MockCoordinatorService{
				GetOrBuildProjectFunc: func(context.Context, []annotator_dto.EntryPoint, ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
					return nil, testCase.err
				},
			}
			service := newSpawnTestService(t, coordinator)

			service.runTargetedRebuild(context.Background(), "pages/page0.pk", []string{"pages/page0.pk"}, false)

			assert.Equal(t, testCase.wantFallback, coordinator.RequestRebuildCallCount.Load() == 1)
		})
	}
}
