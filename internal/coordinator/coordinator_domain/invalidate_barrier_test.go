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

package coordinator_domain

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/safedisk"
)

const (
	barrierTestPages = 3
)

func newBarrierTestService(t *testing.T, annotator *mockAnnotator) *coordinatorService {
	t.Helper()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	files := make(map[string][]byte, barrierTestPages)
	for index := range barrierTestPages {
		relPath := fmt.Sprintf("pages/page%d.pk", index)
		sandbox.AddFile(relPath, []byte("content"))
		files["/project/"+relPath] = []byte("content")
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: files},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				if after, ok := strings.CutPrefix(importPath, "test-module/"); ok {
					return filepath.Join("/project", after), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox), WithMaxBuildWaitDuration(10*time.Second)),
	)
	service.wg.Add(1)
	go service.buildLoop(context.Background())
	return service
}

func barrierTestRequest(index int) *coordinator_dto.BuildRequest {
	return &coordinator_dto.BuildRequest{
		CausationID: fmt.Sprintf("request-%d", index),
		EntryPoints: []annotator_dto.EntryPoint{
			{Path: fmt.Sprintf("test-module/pages/page%d.pk", index%barrierTestPages), IsPage: true},
		},
	}
}

func TestInvalidate_ConcurrentWithBuildsNeverPanics(t *testing.T) {
	t.Parallel()

	service := newBarrierTestService(t, &mockAnnotator{
		ResultToReturn: &annotator_dto.ProjectAnnotationResult{},
		BuildDelay:     time.Millisecond,
	})
	defer service.Shutdown(context.Background())

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
				service.triggerBuild(context.Background(), barrierTestRequest(index))
			}
		})
	}

	for range 200 {
		require.NoError(t, service.Invalidate(context.Background()),
			"Invalidate must wait for in-flight builds while new ones keep starting")
	}
	close(stop)
	producers.Wait()
}

func TestInvalidate_RunningBuildCannotRestoreItsStatus(t *testing.T) {
	t.Parallel()

	stale := &annotator_dto.ProjectAnnotationResult{AnnotatedComponentCount: 1}
	annotator := &mockAnnotator{ResultToReturn: stale, BuildDelay: 200 * time.Millisecond}
	service := newBarrierTestService(t, annotator)
	defer service.Shutdown(context.Background())

	notifications, unsubscribe := service.Subscribe("barrier")
	t.Cleanup(unsubscribe)

	service.triggerBuild(context.Background(), barrierTestRequest(0))
	require.Eventually(t, func() bool { return annotator.GetCallCount() == 1 }, 5*time.Second, time.Millisecond,
		"the build must be running before Invalidate")

	require.NoError(t, service.Invalidate(context.Background()))

	assert.Nil(t, service.GetStatus().Result, "the overtaken build must not restore its result after Invalidate")
	_, ok := service.GetLastSuccessfulBuild()
	assert.False(t, ok)
	select {
	case notification := <-notifications:
		t.Fatalf("the overtaken build must not publish its result, got %q", notification.CausationID)
	default:
	}
}

func TestShutdown_ReleasesWaiters(t *testing.T) {
	t.Parallel()

	annotator := &mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}, BuildDelay: 300 * time.Millisecond}
	service := newBarrierTestService(t, annotator)

	service.triggerBuild(context.Background(), barrierTestRequest(0))
	require.Eventually(t, func() bool { return annotator.GetCallCount() == 1 }, 5*time.Second, time.Millisecond)

	waited := make(chan error, 1)
	go func() {
		_, err := service.waitForBuildResult(context.Background(), "queued-hash", barrierTestRequest(1).EntryPoints, nil)
		waited <- err
	}()
	require.Eventually(t, func() bool {
		_, registered := service.waiters.Load("queued-hash")
		return registered
	}, 5*time.Second, time.Millisecond)

	service.Shutdown(context.Background())

	select {
	case err := <-waited:
		assert.ErrorIs(t, err, ErrBuildInvalidated, "a waiter whose build never ran must be released on shutdown")
	case <-time.After(5 * time.Second):
		t.Fatal("waiter was not released by Shutdown")
	}
}
